package api

import (
	"net/http"

	"dv-page/httpx"
)

// 头像 base64 有几百 KB，缓存一天；ETag 保证换图或改名后能立刻失效。
const profileCacheControl = "public, max-age=86400"

type profileResponse struct {
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

func (s *server) handleProfile(w http.ResponseWriter, r *http.Request) {
	lang := negotiateLanguage(r)

	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		httpx.WriteError(w, s.logger(), http.StatusMethodNotAllowed, lang, "error.method-not-allowed")
		return
	}

	profile := s.opts.App.Profile()

	w.Header().Set("Cache-Control", profileCacheControl)
	w.Header().Set("ETag", profile.ETag())

	if profile.NotModified(r.Header.Get("If-None-Match")) {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	httpx.WriteJSON(w, s.logger(), http.StatusOK, profileResponse{
		Name:   profile.Name,
		Avatar: profile.Avatar.Base64,
	})
}
