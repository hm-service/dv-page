package api

import (
	"net/http"

	"dv-page/httpx"
)

// 名字与头像在 /api/profile，这里只放站点自身的事实。
type siteResponse struct {
	ICP           string `json:"icp"`
	ICPURL        string `json:"icpUrl"`
	CopyrightYear int    `json:"copyrightYear"`
	// Lang 是本次协商结果，前端据此决定首屏语言。
	Lang string `json:"lang"`
}

func (s *server) handleSite(w http.ResponseWriter, r *http.Request) {
	lang := negotiateLanguage(r)

	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		httpx.WriteError(w, s.logger(), http.StatusMethodNotAllowed, lang, "error.method-not-allowed")
		return
	}

	site := s.opts.App.SiteInfo()
	httpx.WriteJSON(w, s.logger(), http.StatusOK, siteResponse{
		ICP:           site.ICP,
		ICPURL:        site.ICPURL,
		CopyrightYear: site.CopyrightYear,
		Lang:          string(lang),
	})
}
