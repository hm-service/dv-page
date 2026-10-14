package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"syscall"
	"time"

	"dv-page/api"
	"dv-page/app"
	"dv-page/builtin"
	"dv-page/core"
	"dv-page/httpx"
)

// 退出码：0 正常停机；1 启动或运行失败；254 停机超时被强关；255 关资源失败。
// 本应用不持有数据库与文件句柄，255 这条路径不存在。
const (
	exitFailure        = 1
	exitForcedShutdown = 254
	shutdownTimeout    = 5 * time.Second
)

func main() {
	os.Exit(run())
}

func run() int {
	timezone, err := loadTimezone(env("TZ", "Asia/Shanghai"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "startup failed:", err)

		return exitFailure
	}

	logger := newLogger(os.Stdout, timezone)

	if err := serve(logger, timezone); err != nil {
		logger.Error("server stopped with error", "error", err)

		return exitFailure
	}

	return 0
}

// newLogger 把结构化 JSON 写到 stdout，交给容器运行时收集；时间戳按 TZ 输出 RFC 3339。
func newLogger(output io.Writer, timezone *time.Location) *slog.Logger {
	return slog.New(slog.NewJSONHandler(output, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, attr slog.Attr) slog.Attr {
			if attr.Key != slog.TimeKey {
				return attr
			}

			stamp, ok := attr.Value.Any().(time.Time)
			if !ok {
				return attr
			}

			return slog.String("timestamp", stamp.In(timezone).Format(time.RFC3339))
		},
	}))
}

func serve(logger *slog.Logger, timezone *time.Location) error {
	assets, err := distFS()
	if err != nil {
		return err
	}

	application, err := newApp(assets)
	if err != nil {
		return err
	}

	trustedProxies, err := httpx.TrustedProxies(env("TRUSTED_PROXIES", "127.0.0.1/32,::1/128"))
	if err != nil {
		return err
	}

	maxAPI, err := intInRange("MAX_API_CONCURRENCY", 128, 1, 4096)
	if err != nil {
		return err
	}

	maxStatic, err := intInRange("MAX_STATIC_CONCURRENCY", 64, 1, 4096)
	if err != nil {
		return err
	}

	readyToken, err := readyzToken()
	if err != nil {
		return err
	}

	address, err := listenAddress()
	if err != nil {
		return err
	}

	handler, err := api.NewRouter(api.Options{
		App:                  application,
		Assets:               assets,
		TrustedProxies:       trustedProxies,
		ReadyToken:           readyToken,
		MaxAPIConcurrency:    maxAPI,
		MaxStaticConcurrency: maxStatic,
		Logger:               logger,
	})
	if err != nil {
		return err
	}

	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	server := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	logger.Info("server started",
		"addr", listener.Addr().String(),
		"start_at", time.Now().In(timezone).Format(time.RFC3339),
		"name", application.Profile().Name,
		"avatar_bytes", len(application.Profile().Avatar.Bytes),
		"max_api_concurrency", maxAPI,
		"max_static_concurrency", maxStatic,
	)

	// 只取第一个停止信号：之后再来的 SIGINT/SIGTERM 留在 channel 里丢掉，
	// 免得第二次信号打断正在进行的优雅停机。
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)

	failed := make(chan error, 1)
	go func() {
		failed <- server.Serve(listener)
	}()

	select {
	case err := <-failed:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}

		return err
	case received := <-signals:
		logger.Info("shutdown signal received", "signal", received.String())

		shutdown, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := server.Shutdown(shutdown); err != nil {
			// 期限内没停干净：强关并立刻退出，由 254 表明是异常停机。
			_ = server.Close()
			logger.Error("shutdown timed out", "signal", received.String(), "error", err)

			os.Exit(exitForcedShutdown)
		}

		logger.Info("server shutdown",
			"signal", received.String(),
			"stop_at", time.Now().In(timezone).Format(time.RFC3339),
		)

		return nil
	}
}

// newApp 组装用例层；应用自己的配置同样在启动时一次性校验。
func newApp(assets fs.FS) (*app.App, error) {
	year, err := intInRange("SITE_COPYRIGHT_YEAR", time.Now().Year(), 1970, 9999)
	if err != nil {
		return nil, err
	}

	return app.New(app.Config{
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
}

func loadTimezone(name string) (*time.Location, error) {
	timezone, err := time.LoadLocation(name)
	if err != nil {
		return nil, fmt.Errorf("env TZ %q: %w", name, err)
	}

	return timezone, nil
}

func listenAddress() (string, error) {
	host := env("HOST", "127.0.0.1")
	if net.ParseIP(host) == nil {
		return "", fmt.Errorf("env HOST must be an ip address, got %q", host)
	}

	port, err := intInRange("PORT", 8080, 1, 65535)
	if err != nil {
		return "", err
	}

	return net.JoinHostPort(host, strconv.Itoa(port)), nil
}

var readyzTokenPattern = regexp.MustCompile(`^[A-Za-z0-9_]{16,128}$`)

// readyzToken 为空表示 /readyz 不设门槛；设了就必须是 16–128 位的字母、数字或下划线。
// 读进内存后立刻从进程环境里抹掉，免得子进程把这份凭据继承走。
func readyzToken() (string, error) {
	token := os.Getenv("READYZ_TOKEN")
	os.Unsetenv("READYZ_TOKEN")

	if token == "" {
		return "", nil
	}
	if !readyzTokenPattern.MatchString(token) {
		return "", errors.New("env READYZ_TOKEN must be 16-128 characters of letters, digits or underscore")
	}

	return token, nil
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

// intInRange 用于启动时必须合法的数字配置：解析不了或越界都当场拒绝启动。
func intInRange(key string, fallback, min, max int) (int, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback, nil
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, errors.New("env " + key + " must be an integer, got " + strconv.Quote(raw))
	}
	if value < min || value > max {
		return 0, fmt.Errorf("env %s must be within [%d, %d], got %d", key, min, max, value)
	}

	return value, nil
}
