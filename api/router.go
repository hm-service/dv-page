// Package api 是 HTTP 层：路由、中间件、语言协商、探针与页面接口。
package api

import (
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
	Logger     *slog.Logger
}

type server struct {
	opts Options
}

// NewRouter 挂上全部路由；方法校验留在处理器里，好让 405 也能走 i18n。
func NewRouter(opts Options) http.Handler {
	s := &server{opts: opts}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/profile", s.handleProfile)
	mux.HandleFunc("/api/site", s.handleSite)
	mux.HandleFunc("/livez", s.handleLivez)
	mux.HandleFunc("/readyz", s.handleReadyz)
	mux.HandleFunc("/", s.handleStatic)

	return s.withAccessLog(s.withRecover(s.withBodyLimit(mux)))
}

func (s *server) clientIP(r *http.Request) string {
	return httpx.ClientIP(r, s.opts.TrustedProxies)
}

func (s *server) logger() *slog.Logger {
	return s.opts.Logger
}
