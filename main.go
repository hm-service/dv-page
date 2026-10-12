package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"dv-page/api"
	"dv-page/app"
	"dv-page/builtin"
	"dv-page/core"
	"dv-page/httpx"
)

func main() {
	logger := newLogger(os.Stdout)

	if err := run(logger); err != nil {
		logger.Error("server stopped with error", "error", err)
		os.Exit(1)
	}
}

// newLogger 把结构化 JSON 写到 stdout，交给容器运行时收集；时间戳按 TZ 输出 RFC 3339。
func newLogger(output io.Writer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(output, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, attr slog.Attr) slog.Attr {
			if attr.Key != slog.TimeKey {
				return attr
			}

			stamp, ok := attr.Value.Any().(time.Time)
			if !ok {
				return attr
			}

			return slog.String(slog.TimeKey, stamp.Format(time.RFC3339))
		},
	}))
}

func run(logger *slog.Logger) error {
	assets, err := distFS()
	if err != nil {
		return err
	}

	year, err := envInt("SITE_COPYRIGHT_YEAR", time.Now().Year())
	if err != nil {
		return err
	}

	application, err := app.New(app.Config{
		Name: env("MY_NAME", "Daveny"),
		// AVATAR_FILE 留空就用内置头像，省得部署时先放一张图才能起来。
		AvatarFile:    os.Getenv("AVATAR_FILE"),
		DefaultAvatar: builtin.Avatar(),
		Site: core.Site{
			ICP:           env("SITE_ICP", "蜀ICP备2026015939号-1"),
			ICPURL:        env("SITE_ICP_URL", "https://beian.miit.gov.cn/"),
			CopyrightYear: year,
		},
		Assets: assets,
	})
	if err != nil {
		return err
	}

	trustedProxies, err := httpx.TrustedProxies(env("TRUSTED_PROXIES", "127.0.0.1/32,::1/128"))
	if err != nil {
		return err
	}

	address := net.JoinHostPort(env("HOST", "127.0.0.1"), env("PORT", "8080"))
	server := &http.Server{
		Addr: address,
		Handler: api.NewRouter(api.Options{
			App:            application,
			Assets:         assets,
			TrustedProxies: trustedProxies,
			ReadyToken:     os.Getenv("READYZ_TOKEN"),
			Logger:         logger,
		}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	signals, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	failed := make(chan error, 1)
	go func() {
		failed <- server.ListenAndServe()
	}()

	logger.Info("server started",
		"address", address,
		"name", application.Profile().Name,
		"avatar_bytes", len(application.Profile().Avatar.Bytes),
	)

	select {
	case err := <-failed:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}

		return err
	case <-signals.Done():
		logger.Info("shutdown requested")

		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		return server.Shutdown(shutdown)
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

func envInt(key string, fallback int) (int, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback, nil
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, errors.New("env " + key + " must be an integer, got " + strconv.Quote(raw))
	}

	return value, nil
}
