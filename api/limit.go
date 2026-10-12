package api

import (
	"log/slog"
	"net/http"
	"strings"

	"dv-page/httpx"
)

// limiter 是一个固定容量的信号量：满了立刻 503，不排队——排队只是把内存压力换成
// 延迟，还会把连接堆在网关里。API 与静态文件各持一个实例，槽位互不挤占。
type limiter struct {
	slots  chan struct{}
	logger *slog.Logger
}

func newLimiter(max int, logger *slog.Logger) *limiter {
	return &limiter{slots: make(chan struct{}, max), logger: logger}
}

func (l *limiter) wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case l.slots <- struct{}{}:
			defer func() { <-l.slots }()
			next.ServeHTTP(w, r)
		default:
			w.Header().Set("Retry-After", "1")
			httpx.WriteError(w, l.logger, http.StatusServiceUnavailable, negotiateLanguage(r), "error.too-many-requests")
		}
	})
}

// isAPIPath 把请求分成两类：API（含两个探针）与静态文件。
func isAPIPath(path string) bool {
	return path == "/livez" || path == "/readyz" || strings.HasPrefix(path, "/api/")
}

// withConcurrencyLimits 按路径分发到各自的信号量。
func withConcurrencyLimits(api, static *limiter, next http.Handler) http.Handler {
	apiHandler := api.wrap(next)
	staticHandler := static.wrap(next)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isAPIPath(r.URL.Path) {
			apiHandler.ServeHTTP(w, r)
			return
		}

		staticHandler.ServeHTTP(w, r)
	})
}
