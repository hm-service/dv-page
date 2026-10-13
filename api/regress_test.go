package api_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"dv-page/internal/testenv"
)

// 回归：以前 `/api/` 下不存在的路径会落到 SPA 回退上，调用方拿到的是 200 + 入口 HTML，
// 于是「接口没了」被当成成功。现在必须是 404 JSON。
func TestUnknownAPIPathIsNotFoundJSON(t *testing.T) {
	handler, _ := testenv.Server(t, testenv.Options{})

	for _, target := range []string{"/api/legacy", "/api/site", "/api/profile/extra"} {
		t.Run(target, func(t *testing.T) {
			recorder := do(handler, httptest.NewRequest(http.MethodGet, target, nil))

			if recorder.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
			}
			if contentType := recorder.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "application/json") {
				t.Fatalf("content-type = %q, want json: an HTML fallback here is exactly the old bug", contentType)
			}
		})
	}
}
