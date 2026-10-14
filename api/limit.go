package api

import (
	"log/slog"
	"net/http"
	"strings"

	"dv-page/httpx"
)

// /readyz 的闸门容量固定：它要查依赖（磁盘 / 数据库），但只给内部探针用，
// 并发预期极低，所以不做成环境变量。
const readyzConcurrency = 4

// limiter 是一个固定容量的信号量：满了立刻 503，不排队——排队只是把内存压力换成
// 延迟，还会把连接堆在网关里。各组各持一个实例，槽位互不挤占。
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

// isAPIPath 只认 /api/ 前缀：两个探针的闸门规则与业务接口不同，单独判断。
func isAPIPath(path string) bool {
	return strings.HasPrefix(path, "/api/")
}

// withConcurrencyLimits 按路径分发到各自的信号量：静态文件一组，业务接口一组，
// /readyz 一组；/livez 不设闸——它只是个「进程还在」的廉价应答，必须恒 200。
func withConcurrencyLimits(api, static, readyz *limiter, next http.Handler) http.Handler {
	apiHandler := api.wrap(next)
	staticHandler := static.wrap(next)
	readyzHandler := readyz.wrap(next)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == livezPath:
			next.ServeHTTP(w, r)
		case r.URL.Path == readyzPath:
			readyzHandler.ServeHTTP(w, r)
		case isAPIPath(r.URL.Path):
			apiHandler.ServeHTTP(w, r)
		default:
			staticHandler.ServeHTTP(w, r)
		}
	})
}
