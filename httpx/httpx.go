package httpx

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"

	"dv-page/i18n"
)

// WriteJSON 只做编码与响应头；此时状态码已发出，编码失败只能记日志。
func WriteJSON(w http.ResponseWriter, logger *slog.Logger, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		logger.Error("encode json response", "error", err)
	}
}

// 错误响应体统一是 {"detail": "…"}，前端只读 detail。
type detailBody struct {
	Detail string `json:"detail"`
}

// SetLanguageHeaders 标出响应正文用的是哪种语言，并声明按 Accept-Language 变化。
func SetLanguageHeaders(w http.ResponseWriter, lang i18n.Language) {
	w.Header().Set("Content-Language", string(lang))
	w.Header().Set("Vary", "Accept-Language")
}

// WriteError 按 lang 渲染面向用户的文案；文案键只进日志，不随响应发出去。
func WriteError(
	w http.ResponseWriter,
	logger *slog.Logger,
	status int,
	lang i18n.Language,
	key string,
	args ...any,
) {
	if status >= http.StatusInternalServerError {
		logger.Error("request failed", "status", status, "key", key)
	}

	SetLanguageHeaders(w, lang)
	WriteJSON(w, logger, status, detailBody{Detail: i18n.T(lang, key, args...)})
}

// TrustedProxies 解析逗号分隔的 CIDR；裸 IP 按单机掩码补全。
func TrustedProxies(spec string) ([]*net.IPNet, error) {
	var networks []*net.IPNet

	for _, item := range strings.Split(spec, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}

		if !strings.Contains(item, "/") {
			ip := net.ParseIP(item)
			if ip == nil {
				return nil, fmt.Errorf("trusted proxy %q is not an ip or cidr", item)
			}

			if ip.To4() != nil {
				item += "/32"
			} else {
				item += "/128"
			}
		}

		_, network, err := net.ParseCIDR(item)
		if err != nil {
			return nil, fmt.Errorf("trusted proxy %q: %w", item, err)
		}

		networks = append(networks, network)
	}

	return networks, nil
}

// ClientIP 返回真实客户端 IP：只有对端可信时才采信 X-Forwarded-For。
func ClientIP(r *http.Request, trusted []*net.IPNet) string {
	peer := remoteIP(r.RemoteAddr)
	peerIP := net.ParseIP(peer)

	if peerIP == nil || !isTrusted(peerIP, trusted) {
		return peer
	}

	forwarded := forwardedFor(r.Header.Values("X-Forwarded-For"))
	for index := len(forwarded) - 1; index >= 0; index-- {
		candidate := net.ParseIP(forwarded[index])
		if candidate == nil || isTrusted(candidate, trusted) {
			continue
		}

		return candidate.String()
	}

	// 整条链都可信时，最左端才是最初的客户端。
	if len(forwarded) > 0 {
		return forwarded[0]
	}

	// 没有 XFF 时退回 X-Real-IP，同样只在直连对端可信时才采信。
	if real := net.ParseIP(strings.TrimSpace(r.Header.Get("X-Real-IP"))); real != nil {
		return real.String()
	}

	return peer
}

func remoteIP(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}

	return host
}

func forwardedFor(values []string) []string {
	var addresses []string

	for _, value := range values {
		for _, item := range strings.Split(value, ",") {
			if item = strings.TrimSpace(item); item != "" {
				addresses = append(addresses, item)
			}
		}
	}

	return addresses
}

func isTrusted(ip net.IP, trusted []*net.IPNet) bool {
	for _, network := range trusted {
		if network.Contains(ip) {
			return true
		}
	}

	return false
}
