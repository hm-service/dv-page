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
	if !strings.Contains(body, `"detail":"too many requests`) {
		t.Fatalf("body = %q, want the localized detail", body)
	}
	if strings.Contains(body, "error.too-many-requests") {
		t.Fatalf("body = %q, must not leak the copy key", body)
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

// 三组槽位互相独立：一组打满，另外两组照常服务。
func TestConcurrencyGroupsAreIndependent(t *testing.T) {
	t.Run("api saturated does not block static", func(t *testing.T) {
		handler, arrived, release := blocker(isAPIPath)
		limited := withConcurrencyLimits(
			newLimiter(1, testLogger()), newLimiter(2, testLogger()), newLimiter(2, testLogger()), handler)

		done := start(limited, "/api/profile")
		waitArrived(t, arrived)

		if code := status(t, limited, "/api/profile"); code != http.StatusServiceUnavailable {
			t.Fatalf("second api request status = %d, want %d", code, http.StatusServiceUnavailable)
		}
		if code := status(t, limited, "/assets/index-abc123.js"); code != http.StatusOK {
			t.Fatalf("static request status = %d, want %d (static slots must be independent)", code, http.StatusOK)
		}
		if code := status(t, limited, "/readyz"); code != http.StatusOK {
			t.Fatalf("readyz status = %d, want %d (readyz slots must be independent)", code, http.StatusOK)
		}

		close(release)
		<-done
	})

	t.Run("static saturated does not block api", func(t *testing.T) {
		handler, arrived, release := blocker(func(path string) bool { return !isAPIPath(path) && path != "/readyz" && path != "/livez" })
		limited := withConcurrencyLimits(
			newLimiter(2, testLogger()), newLimiter(1, testLogger()), newLimiter(2, testLogger()), handler)

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

	t.Run("readyz saturated does not block api", func(t *testing.T) {
		handler, arrived, release := blocker(func(path string) bool { return path == "/readyz" })
		limited := withConcurrencyLimits(
			newLimiter(2, testLogger()), newLimiter(2, testLogger()), newLimiter(1, testLogger()), handler)

		done := start(limited, "/readyz")
		waitArrived(t, arrived)

		if code := status(t, limited, "/readyz"); code != http.StatusServiceUnavailable {
			t.Fatalf("second readyz request status = %d, want %d", code, http.StatusServiceUnavailable)
		}
		if code := status(t, limited, "/api/profile"); code != http.StatusOK {
			t.Fatalf("api request status = %d, want %d (api slots must be independent)", code, http.StatusOK)
		}

		close(release)
		<-done
	})
}

// /livez 不设闸：别的组全打满，它也必须照常 200。
func TestLivezIsNeverGated(t *testing.T) {
	handler, arrived, release := blocker(func(path string) bool { return path != livezPath })
	limited := withConcurrencyLimits(
		newLimiter(1, testLogger()), newLimiter(1, testLogger()), newLimiter(1, testLogger()), handler)

	occupiers := []<-chan int{
		start(limited, "/api/profile"),
		start(limited, "/readyz"),
		start(limited, "/assets/bg.jpg"),
	}
	for range occupiers {
		waitArrived(t, arrived)
	}

	if code := status(t, limited, livezPath); code != http.StatusOK {
		t.Fatalf("livez status = %d, want %d", code, http.StatusOK)
	}

	close(release)
	for _, done := range occupiers {
		<-done
	}
}

// /readyz 有独立闸门，容量固定为 readyzConcurrency。
func TestReadyzHasItsOwnGate(t *testing.T) {
	handler, arrived, release := blocker(func(path string) bool { return path == readyzPath })
	limited := withConcurrencyLimits(
		newLimiter(8, testLogger()), newLimiter(8, testLogger()), newLimiter(readyzConcurrency, testLogger()), handler)

	occupiers := make([]<-chan int, 0, readyzConcurrency)
	for i := 0; i < readyzConcurrency; i++ {
		occupiers = append(occupiers, start(limited, readyzPath))
		waitArrived(t, arrived)
	}

	request := httptest.NewRequest(http.MethodGet, readyzPath, nil)
	request.Header.Set("Accept-Language", "en")
	recorder := httptest.NewRecorder()
	limited.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d after %d in-flight requests", recorder.Code, http.StatusServiceUnavailable, readyzConcurrency)
	}
	if recorder.Header().Get("Retry-After") == "" {
		t.Fatal("Retry-After header is missing")
	}

	close(release)
	for _, done := range occupiers {
		<-done
	}
}

func TestNewRouterRejectsNonPositiveLimits(t *testing.T) {
	if _, err := NewRouter(Options{MaxAPIConcurrency: 1, MaxStaticConcurrency: 0}); err == nil {
		t.Fatal("NewRouter() = nil error, want a complaint about static concurrency")
	}
	if _, err := NewRouter(Options{MaxStaticConcurrency: 1}); err == nil {
		t.Fatal("NewRouter() = nil error, want a complaint about api concurrency")
	}
}
