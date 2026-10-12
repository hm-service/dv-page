package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"dv-page/internal/testenv"
)

type sitePayload struct {
	ICP           string `json:"icp"`
	ICPURL        string `json:"icpUrl"`
	CopyrightYear int    `json:"copyrightYear"`
	Lang          string `json:"lang"`
}

type profilePayload struct {
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

type errorPayload struct {
	Error struct {
		Key     string `json:"key"`
		Message string `json:"message"`
	} `json:"error"`
}

func do(handler http.Handler, request *http.Request) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	return recorder
}

func decode[T any](t *testing.T, recorder *httptest.ResponseRecorder) T {
	t.Helper()

	var payload T
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode body %q: %v", recorder.Body.String(), err)
	}

	return payload
}

func TestSiteEndpoint(t *testing.T) {
	handler, _ := testenv.Server(t, testenv.Options{})

	tests := []struct {
		name     string
		target   string
		accept   string
		wantLang string
	}{
		{name: "default language", target: "/api/site", wantLang: "zh-CN"},
		{name: "accept language english", target: "/api/site", accept: "en-US,en;q=0.9", wantLang: "en"},
		{name: "accept language chinese", target: "/api/site", accept: "zh-CN,zh;q=0.9", wantLang: "zh-CN"},
		{name: "query wins over header", target: "/api/site?lang=en", accept: "zh-CN", wantLang: "en"},
		{name: "unsupported query falls back", target: "/api/site?lang=fr", accept: "en", wantLang: "en"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, tt.target, nil)
			if tt.accept != "" {
				request.Header.Set("Accept-Language", tt.accept)
			}

			recorder := do(handler, request)
			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
			}

			payload := decode[sitePayload](t, recorder)
			if payload.ICP == "" || payload.ICPURL == "" {
				t.Fatalf("payload = %+v, want the site facts", payload)
			}
			if payload.CopyrightYear != 2026 {
				t.Fatalf("copyrightYear = %d, want 2026", payload.CopyrightYear)
			}
			if payload.Lang != tt.wantLang {
				t.Fatalf("lang = %q, want %q", payload.Lang, tt.wantLang)
			}
		})
	}
}

func TestSiteEndpointRejectsOtherMethods(t *testing.T) {
	handler, _ := testenv.Server(t, testenv.Options{})

	request := httptest.NewRequest(http.MethodPost, "/api/site", strings.NewReader("{}"))
	request.Header.Set("Accept-Language", "en")

	recorder := do(handler, request)
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusMethodNotAllowed)
	}

	payload := decode[errorPayload](t, recorder)
	if payload.Error.Key != "error.method-not-allowed" || payload.Error.Message != "request method not allowed" {
		t.Fatalf("error = %+v, want the localized method error", payload.Error)
	}
}

func TestProfileEndpoint(t *testing.T) {
	handler, _ := testenv.Server(t, testenv.Options{})

	recorder := do(handler, httptest.NewRequest(http.MethodGet, "/api/profile", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	payload := decode[profilePayload](t, recorder)
	if payload.Name != testenv.Name() {
		t.Fatalf("name = %q, want %q", payload.Name, testenv.Name())
	}
	if payload.Avatar != testenv.AvatarBase64() {
		t.Fatal("avatar is not the base64 of the served avatar bytes")
	}
	if etag := recorder.Header().Get("ETag"); etag == "" {
		t.Fatal("ETag header is missing")
	}
	if cache := recorder.Header().Get("Cache-Control"); !strings.Contains(cache, "max-age=86400") {
		t.Fatalf("cache-control = %q, want a long max-age", cache)
	}
}

func TestProfileEndpointHonoursConditionalRequests(t *testing.T) {
	handler, _ := testenv.Server(t, testenv.Options{})

	first := do(handler, httptest.NewRequest(http.MethodGet, "/api/profile", nil))
	etag := first.Header().Get("ETag")
	if etag == "" {
		t.Fatal("first response has no ETag")
	}

	request := httptest.NewRequest(http.MethodGet, "/api/profile", nil)
	request.Header.Set("If-None-Match", etag)

	recorder := do(handler, request)
	if recorder.Code != http.StatusNotModified {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotModified)
	}
	if recorder.Body.Len() != 0 {
		t.Fatalf("body = %q, want empty on 304", recorder.Body.String())
	}

	stale := httptest.NewRequest(http.MethodGet, "/api/profile", nil)
	stale.Header.Set("If-None-Match", `"stale"`)
	if recorder := do(handler, stale); recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d for a stale etag", recorder.Code, http.StatusOK)
	}
}

func TestProfileEndpointUsesTheConfiguredName(t *testing.T) {
	handler, _ := testenv.Server(t, testenv.Options{Name: "Hiro"})

	payload := decode[profilePayload](t, do(handler, httptest.NewRequest(http.MethodGet, "/api/profile", nil)))
	if payload.Name != "Hiro" {
		t.Fatalf("name = %q, want %q", payload.Name, "Hiro")
	}
}

func TestProfileEndpointRejectsOtherMethods(t *testing.T) {
	handler, _ := testenv.Server(t, testenv.Options{})

	recorder := do(handler, httptest.NewRequest(http.MethodPost, "/api/profile", strings.NewReader("{}")))
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusMethodNotAllowed)
	}
}

func TestLivez(t *testing.T) {
	handler, _ := testenv.Server(t, testenv.Options{})

	recorder := do(handler, httptest.NewRequest(http.MethodGet, "/livez", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	payload := decode[struct {
		Status string `json:"status"`
	}](t, recorder)
	if payload.Status != "ok" {
		t.Fatalf("status = %q, want %q", payload.Status, "ok")
	}
}

func TestReadyz(t *testing.T) {
	t.Run("without token", func(t *testing.T) {
		handler, _ := testenv.Server(t, testenv.Options{})

		recorder := do(handler, httptest.NewRequest(http.MethodGet, "/readyz", nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
		}
	})

	t.Run("token required", func(t *testing.T) {
		handler, _ := testenv.Server(t, testenv.Options{ReadyToken: "s3cr3t"})

		tests := []struct {
			name   string
			target string
			want   int
		}{
			{name: "missing", target: "/readyz", want: http.StatusForbidden},
			{name: "wrong", target: "/readyz?token=nope", want: http.StatusForbidden},
			{name: "right", target: "/readyz?token=s3cr3t", want: http.StatusOK},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				recorder := do(handler, httptest.NewRequest(http.MethodGet, tt.target, nil))
				if recorder.Code != tt.want {
					t.Fatalf("status = %d, want %d", recorder.Code, tt.want)
				}
			})
		}
	})

	t.Run("frontend not built", func(t *testing.T) {
		// 占位 dist：进程活着，但页面还端不出来。
		handler, _ := testenv.Server(t, testenv.Options{
			Assets: fstest.MapFS{"STUB": {Data: []byte{}}},
		})

		request := httptest.NewRequest(http.MethodGet, "/readyz", nil)
		request.Header.Set("Accept-Language", "en")

		recorder := do(handler, request)
		if recorder.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
		}

		payload := decode[errorPayload](t, recorder)
		if payload.Error.Key != "readyz.not-ready" || !strings.Contains(payload.Error.Message, "index.html") {
			t.Fatalf("error = %+v, want the readiness complaint", payload.Error)
		}
	})
}

func TestStaticFiles(t *testing.T) {
	handler, _ := testenv.Server(t, testenv.Options{})

	tests := []struct {
		name          string
		method        string
		target        string
		wantStatus    int
		wantCache     string
		wantBody      string
		wantBodyEmpty bool
	}{
		{
			name:       "entry file",
			method:     http.MethodGet,
			target:     "/",
			wantStatus: http.StatusOK,
			wantCache:  "no-cache",
			wantBody:   `id="app"`,
		},
		{
			name:       "hashed asset is immutable",
			method:     http.MethodGet,
			target:     "/assets/index-abc123.js",
			wantStatus: http.StatusOK,
			wantCache:  "immutable",
			wantBody:   "dv-page",
		},
		{
			name:       "unknown navigation falls back to entry",
			method:     http.MethodGet,
			target:     "/some/deep/link",
			wantStatus: http.StatusOK,
			wantCache:  "no-cache",
			wantBody:   `id="app"`,
		},
		{
			name:       "missing asset is not the entry file",
			method:     http.MethodGet,
			target:     "/assets/missing.js",
			wantStatus: http.StatusNotFound,
			wantBody:   "error.not-found",
		},
		{
			name:          "head has no body",
			method:        http.MethodHead,
			target:        "/",
			wantStatus:    http.StatusOK,
			wantBodyEmpty: true,
		},
		{
			name:       "post is rejected",
			method:     http.MethodPost,
			target:     "/",
			wantStatus: http.StatusMethodNotAllowed,
			wantBody:   "error.method-not-allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := do(handler, httptest.NewRequest(tt.method, tt.target, nil))

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.wantStatus)
			}
			if tt.wantCache != "" && !strings.Contains(recorder.Header().Get("Cache-Control"), tt.wantCache) {
				t.Fatalf("cache-control = %q, want it to contain %q", recorder.Header().Get("Cache-Control"), tt.wantCache)
			}
			if tt.wantBodyEmpty {
				if recorder.Body.Len() != 0 {
					t.Fatalf("body = %q, want empty", recorder.Body.String())
				}

				return
			}
			if !strings.Contains(recorder.Body.String(), tt.wantBody) {
				t.Fatalf("body = %q, want it to contain %q", recorder.Body.String(), tt.wantBody)
			}
		})
	}
}

func TestPlaceholderDistKeepsTheApiAlive(t *testing.T) {
	// 干净检出时 justfile 只造占位 dist：接口要能用，页面 404、就绪探针 503。
	handler, _ := testenv.Server(t, testenv.Options{
		Assets: fstest.MapFS{"STUB": {Data: []byte{}}},
	})

	tests := []struct {
		name       string
		target     string
		wantStatus int
	}{
		{name: "api works", target: "/api/site", wantStatus: http.StatusOK},
		{name: "profile works", target: "/api/profile", wantStatus: http.StatusOK},
		{name: "entry page missing", target: "/", wantStatus: http.StatusNotFound},
		{name: "deep link missing", target: "/some/link", wantStatus: http.StatusNotFound},
		{name: "not ready", target: "/readyz", wantStatus: http.StatusServiceUnavailable},
		{name: "livez still ok", target: "/livez", wantStatus: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := do(handler, httptest.NewRequest(http.MethodGet, tt.target, nil))
			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.wantStatus)
			}
		})
	}
}

func TestAccessLog(t *testing.T) {
	t.Run("records client ip and status", func(t *testing.T) {
		var logged bytes.Buffer
		handler, _ := testenv.Server(t, testenv.Options{Logger: testenv.Logger(&logged)})

		recorder := do(handler, httptest.NewRequest(http.MethodGet, "/livez", nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
		}

		line := logged.String()
		for _, want := range []string{`"client_ip":"192.0.2.1"`, `"status":200`, `"path":"/livez"`, `"method":"GET"`, `"time":`} {
			if !strings.Contains(line, want) {
				t.Fatalf("log = %q, want it to contain %q", line, want)
			}
		}
	})

	t.Run("uses forwarded ip behind a trusted proxy", func(t *testing.T) {
		var logged bytes.Buffer
		handler, _ := testenv.Server(t, testenv.Options{
			Logger:  testenv.Logger(&logged),
			Trusted: "10.0.0.0/8",
		})

		request := httptest.NewRequest(http.MethodGet, "/livez", nil)
		request.RemoteAddr = "10.0.0.5:44444"
		request.Header.Set("X-Forwarded-For", "203.0.113.9")

		do(handler, request)

		if line := logged.String(); !strings.Contains(line, `"client_ip":"203.0.113.9"`) {
			t.Fatalf("log = %q, want the forwarded client ip", line)
		}
	})

	t.Run("keeps the readyz token out of the log", func(t *testing.T) {
		var logged bytes.Buffer
		handler, _ := testenv.Server(t, testenv.Options{
			Logger:     testenv.Logger(&logged),
			ReadyToken: "s3cr3t",
		})

		recorder := do(handler, httptest.NewRequest(http.MethodGet, "/readyz?token=s3cr3t", nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
		}

		line := logged.String()
		if strings.Contains(line, "s3cr3t") {
			t.Fatalf("log = %q, must not contain the token", line)
		}
		if !strings.Contains(line, `"path":"/readyz"`) {
			t.Fatalf("log = %q, want the path without the query", line)
		}
	})
}
