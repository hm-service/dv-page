package core

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// Site 是主页上「站点」的部分：备案与版权，与语言无关。
type Site struct {
	ICP           string
	ICPURL        string
	CopyrightYear int
}

// Validate 是启动时的把关：配置错误应当尽早失败，而不是渲染出半张页面。
func (s Site) Validate() error {
	if strings.TrimSpace(s.ICP) == "" {
		return errors.New("site icp must not be empty")
	}

	parsed, err := url.Parse(s.ICPURL)
	if err != nil {
		return fmt.Errorf("site icp url %q: %w", s.ICPURL, err)
	}
	if (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return fmt.Errorf("site icp url %q must be an absolute http(s) url", s.ICPURL)
	}
	if s.CopyrightYear < 1970 || s.CopyrightYear > 9999 {
		return fmt.Errorf("site copyright year %d is out of range", s.CopyrightYear)
	}

	return nil
}
