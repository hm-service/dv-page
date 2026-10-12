package api

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}

// blocker 只把指定路径挡住，用来把某个分组的槽位占满。
func blocker(blockOn func(string) bool) (http.Handler, chan struct{}, chan struct{}) {
	arrived := make(chan struct{}, 16)
	release := make(chan struct{})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if blockOn(r.URL.Path) {
			arrived <- struct{}{}
			<-release
		}
		w.WriteHeader(http.StatusOK)
	})

	return handler, arrived, release
}

func start(handler http.Handler, path string) <-chan int {
	done := make(chan int, 1)

	go func() {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		done <- recorder.Code
	}()

	return done
}

func waitArrived(t *testing.T, arrived <-chan struct{}) {
	t.Helper()

	select {
	case <-arrived:
	case <-time.After(2 * time.Second):
		t.Fatal("the blocking request never reached the handler")
	}
}

func status(t *testing.T, handler http.Handler, path string) int {
	t.Helper()

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))

	return recorder.Code
}

func TestLimiterRejectsWhenSaturated(t *testing.T) {
	handler, arrived, release := blocker(func(string) bool { return true })
	limited := newLimiter(1, testLogger()).wrap(handler)

	done := start(limited, "/api/profile")
	waitArrived(t, arrived)

	request := httptest.NewRequest(http.MethodGet, "/api/profile", nil)
	request.Header.Set("Accept-Language", "en")
	recorder := httptest.NewRecorder()
	limited.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
	if retryAfter := recorder.Header().Get("Retry-After"); retryAfter == "" {
		t.Fatal("Retry-After header is missing")
	}
	body := recorder.Body.String()
	if !strings.Contains(body, "error.too-many-requests") || !strings.Contains(body, "too many requests") {
		t.Fatalf("body = %q, want the localized error", body)
	}

	close(release)
	if code := <-done; code != http.StatusOK {
		t.Fatalf("the occupying request finished with %d, want %d", code, http.StatusOK)
	}
}

func TestLimiterReturnsItsSlot(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	limited := newLimiter(1, testLogger()).wrap(handler)

	for attempt := 0; attempt < 200; attempt++ {
		if code := status(t, limited, "/api/profile"); code != http.StatusOK {
			t.Fatalf("request %d status = %d, want %d (slot leaked?)", attempt, code, http.StatusOK)
		}
	}
}

// 两组槽位互相独立：一组打满，另一组照常服务。
func TestConcurrencyGroupsAreIndependent(t *testing.T) {
	t.Run("api saturated does not block static", func(t *testing.T) {
		handler, arrived, release := blocker(isAPIPath)
		limited := withConcurrencyLimits(newLimiter(1, testLogger()), newLimiter(2, testLogger()), handler)

		done := start(limited, "/api/profile")
		waitArrived(t, arrived)

		if code := status(t, limited, "/api/profile"); code != http.StatusServiceUnavailable {
			t.Fatalf("second api request status = %d, want %d", code, http.StatusServiceUnavailable)
		}
		if code := status(t, limited, "/assets/index-abc123.js"); code != http.StatusOK {
			t.Fatalf("static request status = %d, want %d (static slots must be independent)", code, http.StatusOK)
		}

		close(release)
		<-done
	})

	t.Run("static saturated does not block api", func(t *testing.T) {
		handler, arrived, release := blocker(func(path string) bool { return !isAPIPath(path) })
		limited := withConcurrencyLimits(newLimiter(2, testLogger()), newLimiter(1, testLogger()), handler)

		done := start(limited, "/assets/block.js")
		waitArrived(t, arrived)

		if code := status(t, limited, "/assets/other.js"); code != http.StatusServiceUnavailable {
			t.Fatalf("second static request status = %d, want %d", code, http.StatusServiceUnavailable)
		}
		if code := status(t, limited, "/api/profile"); code != http.StatusOK {
			t.Fatalf("api request status = %d, want %d (api slots must be independent)", code, http.StatusOK)
		}

		close(release)
		<-done
	})
}

// 探针按约定算在 API 组里：API 打满时它们同样会被限流。
func TestProbesShareTheAPIGroup(t *testing.T) {
	handler, arrived, release := blocker(isAPIPath)
	limited := withConcurrencyLimits(newLimiter(1, testLogger()), newLimiter(8, testLogger()), handler)

	done := start(limited, "/api/profile")
	waitArrived(t, arrived)

	for _, path := range []string{"/livez", "/readyz"} {
		if code := status(t, limited, path); code != http.StatusServiceUnavailable {
			t.Fatalf("%s status = %d, want %d", path, code, http.StatusServiceUnavailable)
		}
	}
	if code := status(t, limited, "/"); code != http.StatusOK {
		t.Fatalf("static entry status = %d, want %d", code, http.StatusOK)
	}

	close(release)
	<-done
}

func TestNewRouterRejectsNonPositiveLimits(t *testing.T) {
	if _, err := NewRouter(Options{MaxAPIConcurrency: 1, MaxStaticConcurrency: 0}); err == nil {
		t.Fatal("NewRouter() = nil error, want a complaint about static concurrency")
	}
	if _, err := NewRouter(Options{MaxStaticConcurrency: 1}); err == nil {
		t.Fatal("NewRouter() = nil error, want a complaint about api concurrency")
	}
}
