package api

import (
	"net/http"

	"dv-page/i18n"
)

// negotiateLanguage：显式的 ?lang= 优先于 Accept-Language。
func negotiateLanguage(r *http.Request) i18n.Language {
	if explicit, ok := i18n.Parse(r.URL.Query().Get("lang")); ok {
		return explicit
	}

	return i18n.Negotiate(r.Header.Get("Accept-Language"))
}
