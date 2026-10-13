package i18n

import "testing"

// 每种语言都必须有同一套键：漏一列就是缺文案，靠这个用例钉住。
func TestMessagesCoverEveryLanguage(t *testing.T) {
	languages := []Language{Chinese, English}

	union := map[string]struct{}{}
	for _, lang := range languages {
		table, ok := messages[lang]
		if !ok {
			t.Fatalf("%q has no message table", lang)
		}
		for key := range table {
			union[key] = struct{}{}
		}
	}

	for _, lang := range languages {
		table := messages[lang]
		if len(table) != len(union) {
			t.Errorf("%q has %d keys, want %d", lang, len(table), len(union))
		}
		for key := range union {
			if _, ok := table[key]; !ok {
				t.Errorf("%q is missing key %q", lang, key)
			}
		}
	}
}

func TestT(t *testing.T) {
	// 现有文案表没有占位符，用两个临时键把「键 + 参数」的约定钉住。
	messages[Chinese]["test.placeholder"] = "未就绪：%s"
	messages[English]["test.placeholder"] = "not ready: %s"
	t.Cleanup(func() {
		delete(messages[Chinese], "test.placeholder")
		delete(messages[English], "test.placeholder")
	})

	tests := []struct {
		name string
		lang Language
		key  string
		args []any
		want string
	}{
		{name: "chinese", lang: Chinese, key: "error.not-found", want: "资源不存在"},
		{name: "english", lang: English, key: "error.not-found", want: "resource not found"},
		{name: "placeholder", lang: English, key: "test.placeholder", args: []any{"boom"}, want: "not ready: boom"},
		{name: "chinese placeholder", lang: Chinese, key: "test.placeholder", args: []any{"目录不可写"}, want: "未就绪：目录不可写"},
		{name: "unknown key returns key", lang: Chinese, key: "error.missing", want: "error.missing"},
		{name: "unknown language returns key", lang: Language("fr"), key: "error.not-found", want: "error.not-found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := T(tt.lang, tt.key, tt.args...); got != tt.want {
				t.Fatalf("T(%q, %q, %v) = %q, want %q", tt.lang, tt.key, tt.args, got, tt.want)
			}
		})
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		value string
		want  Language
		ok    bool
	}{
		{value: "zh", want: Chinese, ok: true},
		{value: "zh-CN", want: Chinese, ok: true},
		{value: "ZH-hans", want: Chinese, ok: true},
		{value: "en", want: English, ok: true},
		{value: "en-US", want: English, ok: true},
		{value: " en-gb ", want: English, ok: true},
		{value: "fr", ok: false},
		{value: "", ok: false},
		{value: "*", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			got, ok := Parse(tt.value)
			if ok != tt.ok || got != tt.want {
				t.Fatalf("Parse(%q) = (%q, %v), want (%q, %v)", tt.value, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestNegotiate(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   Language
	}{
		{name: "empty uses default", header: "", want: Default},
		{name: "chinese wins", header: "zh-CN,zh;q=0.9,en;q=0.8", want: Chinese},
		{name: "english wins", header: "en-US,en;q=0.9,zh;q=0.8", want: English},
		{name: "quality orders", header: "zh;q=0.3,en;q=0.7", want: English},
		{name: "wildcard falls back to default", header: "*", want: Default},
		{name: "unknown only", header: "fr-FR,de;q=0.9", want: Default},
		{name: "zero quality is refused", header: "zh;q=0,en;q=0", want: Default},
		{name: "malformed quality is skipped", header: "en;q=abc,zh;q=0.4", want: Chinese},
		{name: "first tag wins on tie", header: "en;q=0.5,zh;q=0.5", want: English},
		{name: "spaces tolerated", header: " en-GB ; q=0.9 , zh ; q=0.1 ", want: English},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Negotiate(tt.header); got != tt.want {
				t.Fatalf("Negotiate(%q) = %q, want %q", tt.header, got, tt.want)
			}
		})
	}
}
