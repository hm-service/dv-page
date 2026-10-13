// Package api 是 HTTP 层：路由、中间件、语言协商、探针与页面接口。
package api

import (
	"errors"
	"io/fs"
	"log/slog"
	"net"
	"net/http"

	"dv-page/app"
	"dv-page/httpx"
)

// Options 是 HTTP 层需要的外部依赖。
type Options struct {
	App            *app.App
	Assets         fs.FS
	TrustedProxies []*net.IPNet
	// ReadyToken 为空表示 /readyz 不设门槛；设置后必须以 ?token= 命中。
	ReadyToken string
	// MaxAPIConcurrency / MaxStaticConcurrency 是两组在途请求的上限，都必须 >= 1。
	MaxAPIConcurrency    int
	MaxStaticConcurrency int
	Logger               *slog.Logger
}

type server struct {
	opts Options
}

// NewRouter 挂上全部路由。每个路径都额外注册一条「方法不对也算匹配」的回退，
// 否则 ServeMux 只有在没有任何模式匹配时才回 405，而站点级 `/` 回退会把它变成 404。
func NewRouter(opts Options) (http.Handler, error) {
	if opts.MaxAPIConcurrency < 1 {
		return nil, errors.New("max api concurrency must be at least 1")
	}
	if opts.MaxStaticConcurrency < 1 {
		return nil, errors.New("max static concurrency must be at least 1")
	}

	s := &server{opts: opts}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/profile", s.handleProfile)
	mux.HandleFunc("/api/profile", s.handleMethodNotAllowed)
	mux.HandleFunc("GET /api/avatar", s.handleAvatar)
	mux.HandleFunc("/api/avatar", s.handleMethodNotAllowed)
	mux.HandleFunc("GET /livez", s.handleLivez)
	mux.HandleFunc("/livez", s.handleMethodNotAllowed)
	mux.HandleFunc("GET /readyz", s.handleReadyz)
	mux.HandleFunc("/readyz", s.handleMethodNotAllowed)
	mux.HandleFunc("/", s.handleStatic)

	limited := withConcurrencyLimits(
		newLimiter(opts.MaxAPIConcurrency, opts.Logger),
		newLimiter(opts.MaxStaticConcurrency, opts.Logger),
		s.withBodyLimit(mux),
	)

	return s.withAccessLog(s.withRecover(s.withSecurityHeaders(limited))), nil
}

func (s *server) clientIP(r *http.Request) string {
	return httpx.ClientIP(r, s.opts.TrustedProxies)
}

func (s *server) logger() *slog.Logger {
	return s.opts.Logger
}

// handleMethodNotAllowed 是那些「路径对、方法不对」的回退：Go 的 GET 模式同时接受 HEAD。
func (s *server) handleMethodNotAllowed(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Allow", "GET, HEAD")
	httpx.WriteError(w, s.logger(), http.StatusMethodNotAllowed, negotiateLanguage(r), "error.method-not-allowed")
}
