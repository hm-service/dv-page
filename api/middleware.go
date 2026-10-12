package api

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"dv-page/httpx"
)

// 输入上限：本应用没有请求体，留一个保守的天花板挡住意外的提交。
const maxRequestBodyBytes = 1 << 20

func (s *server) withBodyLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
		}

		next.ServeHTTP(w, r)
	})
}

func (s *server) withRecover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			recovered := recover()
			if recovered == nil {
				return
			}

			s.logger().Error("panic recovered",
				"panic", fmt.Sprint(recovered),
				"path", r.URL.Path,
				"stack", string(debug.Stack()),
			)
			httpx.WriteError(w, s.logger(), http.StatusInternalServerError, negotiateLanguage(r), "error.internal")
		}()

		next.ServeHTTP(w, r)
	})
}

// statusRecorder 让访问日志拿到状态码与响应体字节数。
type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(payload []byte) (int, error) {
	written, err := r.ResponseWriter.Write(payload)
	r.bytes += written

	return written, err
}

// withAccessLog：一条请求一行 JSON，时间戳由 handler 按 TZ 输出 RFC 3339。
func (s *server) withAccessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(recorder, r)

		s.logger().Info("request",
			"client_ip", s.clientIP(r),
			"method", r.Method,
			// 故意只记 path：?token= 之类的凭据不该落到日志里。
			"path", r.URL.Path,
			"status", recorder.status,
			"bytes", recorder.bytes,
			"duration_ms", time.Since(started).Milliseconds(),
			"lang", string(negotiateLanguage(r)),
		)
	})
}
