package api

import (
	"net/http"

	"dv-page/core"
	"dv-page/httpx"
)

const (
	livezPath  = "/livez"
	readyzPath = "/readyz"
)

type statusResponse struct {
	Status string `json:"status"`
}

// handleLivez 只看进程是否还能应答：只要服务在跑就恒 200。
func (s *server) handleLivez(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, s.logger(), http.StatusOK, statusResponse{Status: "ok"})
}

// handleReadyz 查依赖，并按约定留一条审计日志。
// 失败的具体原因只进日志，响应里只给一句通用文案。
func (s *server) handleReadyz(w http.ResponseWriter, r *http.Request) {
	lang := negotiateLanguage(r)

	if !s.readyTokenMatches(r) {
		s.logReadyz(r, http.StatusUnauthorized, "token mismatch")
		w.Header().Set("WWW-Authenticate", "Bearer")
		httpx.WriteError(w, s.logger(), http.StatusUnauthorized, lang, "error.unauthorized")

		return
	}

	if err := s.opts.App.Ready(); err != nil {
		s.logReadyz(r, http.StatusServiceUnavailable, err.Error())
		httpx.WriteError(w, s.logger(), http.StatusServiceUnavailable, lang, "error.unavailable")

		return
	}

	s.logReadyz(r, http.StatusOK, "")
	httpx.WriteJSON(w, s.logger(), http.StatusOK, statusResponse{Status: "ready"})
}

func (s *server) readyTokenMatches(r *http.Request) bool {
	if s.opts.ReadyToken == "" {
		return true
	}

	return core.SecureEqual(r.URL.Query().Get("token"), s.opts.ReadyToken)
}

func (s *server) logReadyz(r *http.Request, status int, reason string) {
	attrs := []any{"client_ip", s.clientIP(r), "status", status}
	if reason != "" {
		attrs = append(attrs, "reason", reason)
	}

	s.logger().Info("readyz", attrs...)
}
