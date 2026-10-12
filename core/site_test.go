package core

import (
	"strings"
	"testing"
)

func validSite() Site {
	return Site{
		ICP:           "蜀ICP备2026015939号-1",
		ICPURL:        "https://beian.miit.gov.cn/",
		CopyrightYear: 2026,
	}
}

func TestSiteValidate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Site)
		wantErr string
	}{
		{name: "valid", mutate: func(*Site) {}},
		{name: "blank icp", mutate: func(s *Site) { s.ICP = "\t" }, wantErr: "site icp must not be empty"},
		{name: "relative icp url", mutate: func(s *Site) { s.ICPURL = "/beian" }, wantErr: "must be an absolute http(s) url"},
		{name: "unsupported scheme", mutate: func(s *Site) { s.ICPURL = "ftp://beian.miit.gov.cn" }, wantErr: "must be an absolute http(s) url"},
		{name: "year too small", mutate: func(s *Site) { s.CopyrightYear = 1969 }, wantErr: "out of range"},
		{name: "year too large", mutate: func(s *Site) { s.CopyrightYear = 10000 }, wantErr: "out of range"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			site := validSite()
			tt.mutate(&site)

			err := site.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() = %v, want nil", err)
				}

				return
			}

			if err == nil {
				t.Fatalf("Validate() = nil, want error containing %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Validate() = %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}
