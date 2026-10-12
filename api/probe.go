package api

import (
	"net/http"

	"dv-page/core"
	"dv-page/httpx"
)

type statusResponse struct {
	Status string `json:"status"`
}

// handleLivez 只看进程是否还能应答，不查依赖、不记日志。
func (s *server) handleLivez(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		httpx.WriteError(w, s.logger(), http.StatusMethodNotAllowed, negotiateLanguage(r), "error.method-not-allowed")
		return
	}

	httpx.WriteJSON(w, s.logger(), http.StatusOK, statusResponse{Status: "ok"})
}

// handleReadyz 查依赖就绪，并按约定留一条审计日志。
func (s *server) handleReadyz(w http.ResponseWriter, r *http.Request) {
	lang := negotiateLanguage(r)

	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		httpx.WriteError(w, s.logger(), http.StatusMethodNotAllowed, lang, "error.method-not-allowed")
		return
	}

	if !s.readyTokenMatches(r) {
		s.logReadyz(r, http.StatusForbidden, "token mismatch")
		httpx.WriteError(w, s.logger(), http.StatusForbidden, lang, "error.forbidden")
		return
	}

	if err := s.opts.App.Ready(); err != nil {
		s.logReadyz(r, http.StatusServiceUnavailable, err.Error())
		httpx.WriteError(w, s.logger(), http.StatusServiceUnavailable, lang, "readyz.not-ready", err.Error())
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
