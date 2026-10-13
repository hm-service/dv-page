package i18n

import (
	"fmt"
	"strconv"
	"strings"
)

// Language 是后端认可的两种语言；其它中文变体一律归到 zh-CN。
type Language string

const (
	Chinese Language = "zh-CN"
	English Language = "en"

	Default = Chinese
)

var messages = map[Language]map[string]string{
	Chinese: {
		"error.internal":           "服务器内部错误",
		"error.method-not-allowed": "请求方法不被允许",
		"error.not-found":          "资源不存在",
		"error.too-many-requests":  "请求过多，请稍后重试",
		"error.unauthorized":       "缺少或错误的凭据",
		"error.unavailable":        "服务暂不可用",
	},
	English: {
		"error.internal":           "internal server error",
		"error.method-not-allowed": "request method not allowed",
		"error.not-found":          "resource not found",
		"error.too-many-requests":  "too many requests, please retry later",
		"error.unauthorized":       "missing or invalid credentials",
		"error.unavailable":        "service unavailable",
	},
}

// T 渲染文案；未配置的键原样返回键名，漏配要能在响应里一眼看见，不做回退。
func T(lang Language, key string, args ...any) string {
	text, ok := messages[lang][key]
	if !ok {
		return key
	}
	if len(args) == 0 {
		return text
	}

	return fmt.Sprintf(text, args...)
}

// Parse 把 Accept-Language 或 lang 参数里的单个标签归一化。
func Parse(value string) (Language, bool) {
	base, _, _ := strings.Cut(strings.ToLower(strings.TrimSpace(value)), "-")

	switch base {
	case "zh":
		return Chinese, true
	case "en":
		return English, true
	default:
		return "", false
	}
}

// Negotiate 按 q 值挑一种语言；都不认识或没给 q 时用默认语言。
func Negotiate(header string) Language {
	best := Default
	bestQuality := -1.0

	for _, part := range strings.Split(header, ",") {
		tag, quality := splitQuality(part)
		if tag == "" {
			continue
		}

		lang, ok := Parse(tag)
		if !ok {
			if tag != "*" {
				continue
			}
			lang = Default
		}
		if quality > bestQuality {
			best, bestQuality = lang, quality
		}
	}

	return best
}

// splitQuality 拆出「标签 + q 值」；q=0 表示客户端不接受，直接丢弃。
func splitQuality(part string) (string, float64) {
	fields := strings.Split(part, ";")
	tag := strings.TrimSpace(fields[0])
	quality := 1.0

	for _, param := range fields[1:] {
		key, value, found := strings.Cut(strings.TrimSpace(param), "=")
		if !found || !strings.EqualFold(strings.TrimSpace(key), "q") {
			continue
		}

		parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		if err != nil {
			return "", 0
		}
		quality = parsed
	}

	if quality <= 0 || tag == "" {
		return "", 0
	}

	return tag, quality
}
