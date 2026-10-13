package api

import (
	"net/http"

	"dv-page/httpx"
)

// 档案与头像都能缓存一天；ETag 保证换内容后旧缓存立刻失效。
const cacheControlDay = "public, max-age=86400"

// siteResponse 是站点事实，嵌在 profile 里一起下发。
type siteResponse struct {
	ICP           string `json:"icp"`
	ICPURL        string `json:"icpUrl"`
	CopyrightYear int    `json:"copyrightYear"`
}

// 头像是二进制资源，单独走 /api/avatar，不塞进 JSON。
type profileResponse struct {
	Name string       `json:"name"`
	Site siteResponse `json:"site"`
	// Lang 是本次协商结果，前端据此决定首屏语言。
	Lang string `json:"lang"`
}

func (s *server) handleProfile(w http.ResponseWriter, r *http.Request) {
	lang := negotiateLanguage(r)
	profile := s.opts.App.Profile()

	httpx.SetLanguageHeaders(w, lang)
	w.Header().Set("Cache-Control", cacheControlDay)
	w.Header().Set("ETag", profile.ETag())

	if profile.NotModified(r.Header.Get("If-None-Match")) {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	httpx.WriteJSON(w, s.logger(), http.StatusOK, profileResponse{
		Name: profile.Name,
		Site: siteResponse{
			ICP:           profile.Site.ICP,
			ICPURL:        profile.Site.ICPURL,
			CopyrightYear: profile.Site.CopyrightYear,
		},
		Lang: string(lang),
	})
}
