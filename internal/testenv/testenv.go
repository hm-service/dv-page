// Package testenv 是测试脚手架：临时数据目录、最小前端产物、现成的路由。
package testenv

import (
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"testing"
	"testing/fstest"

	"dv-page/api"
	"dv-page/app"
	"dv-page/builtin"
	"dv-page/core"
	"dv-page/httpx"
)

// Assets 是一份够用的假前端产物：入口文件齐全，静态资源带内容哈希的样子。
func Assets() fs.FS {
	return fstest.MapFS{
		"index.html": {
			Data: []byte("<!doctype html><html><body><div id=\"app\"></div></body></html>"),
		},
		"assets/index-abc123.js": {Data: []byte("console.log('dv-page')")},
		"favicon.svg":            {Data: []byte("<svg xmlns=\"http://www.w3.org/2000/svg\"/>")},
	}
}

// Site 是一份合法的站点事实，字段可按用例覆盖。
func Site() core.Site {
	return core.Site{
		ICP:           "蜀ICP备2026015939号-1",
		ICPURL:        "https://beian.miit.gov.cn/",
		CopyrightYear: 2026,
	}
}

// Name 是默认的展示名。
func Name() string {
	return "Daveny"
}

// AvatarBytes 是内置头像的原始字节：/api/avatar 返回的就该是这一份。
func AvatarBytes() []byte {
	return builtin.Avatar()
}

// Logger 把日志收进调用方给的 writer，便于断言访问日志与审计日志。
func Logger(w io.Writer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(w, nil))
}

type Options struct {
	Name       string
	AvatarFile string
	Site       *core.Site
	Assets     fs.FS
	ReadyToken string
	Trusted    string
	Logger     *slog.Logger
	// 两个并发上限为 0 时用下面的默认值。
	MaxAPIConcurrency    int
	MaxStaticConcurrency int
}

// 与生产的默认值一致；限流用例可以压低。
const (
	DefaultMaxAPIConcurrency    = 128
	DefaultMaxStaticConcurrency = 64
)

// Server 起一套现成的服务：日志默认丢弃。
func Server(t *testing.T, opts Options) (http.Handler, *app.App) {
	t.Helper()

	name := opts.Name
	if name == "" {
		name = Name()
	}

	site := Site()
	if opts.Site != nil {
		site = *opts.Site
	}

	assets := opts.Assets
	if assets == nil {
		assets = Assets()
	}

	logger := opts.Logger
	if logger == nil {
		logger = Logger(io.Discard)
	}

	trustedSpec := opts.Trusted
	if trustedSpec == "" {
		trustedSpec = "127.0.0.1/32,::1/128"
	}

	trusted, err := httpx.TrustedProxies(trustedSpec)
	if err != nil {
		t.Fatalf("parse trusted proxies: %v", err)
	}

	application, err := app.New(app.Config{
		Name:          name,
		AvatarFile:    opts.AvatarFile,
		DefaultAvatar: builtin.Avatar(),
		Site:          site,
		Assets:        assets,
	})
	if err != nil {
		t.Fatalf("build app: %v", err)
	}

	maxAPI := opts.MaxAPIConcurrency
	if maxAPI == 0 {
		maxAPI = DefaultMaxAPIConcurrency
	}
	maxStatic := opts.MaxStaticConcurrency
	if maxStatic == 0 {
		maxStatic = DefaultMaxStaticConcurrency
	}

	handler, err := api.NewRouter(api.Options{
		App:                  application,
		Assets:               assets,
		TrustedProxies:       trusted,
		ReadyToken:           opts.ReadyToken,
		MaxAPIConcurrency:    maxAPI,
		MaxStaticConcurrency: maxStatic,
		Logger:               logger,
	})
	if err != nil {
		t.Fatalf("build router: %v", err)
	}

	return handler, application
}
