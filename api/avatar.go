package api

import (
	"bytes"
	"net/http"
	"time"

	"dv-page/httpx"
)

func (s *server) handleAvatar(w http.ResponseWriter, r *http.Request) {
	lang := negotiateLanguage(r)

	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		httpx.WriteError(w, s.logger(), http.StatusMethodNotAllowed, lang, "error.method-not-allowed")
		return
	}

	avatar := s.opts.App.Profile().Avatar

	w.Header().Set("Content-Type", avatar.ContentType)
	w.Header().Set("Cache-Control", cacheControlDay)
	w.Header().Set("ETag", avatar.ETag())

	// ServeContent 负责 Range 与条件请求，并按上面的 ETag 处理 If-None-Match；
	// 名字留空，免得它再按扩展名嗅探一次类型。
	http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(avatar.Bytes))
}
