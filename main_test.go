package main

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"strings"
	"testing"
	"time"
)

func TestEnv(t *testing.T) {
	t.Run("uses the fallback when unset", func(t *testing.T) {
		t.Setenv("DV_PAGE_TEST_VALUE", "")

		if got := env("DV_PAGE_TEST_VALUE", "fallback"); got != "fallback" {
			t.Fatalf("env() = %q, want %q", got, "fallback")
		}
	})

	t.Run("uses the configured value", func(t *testing.T) {
		t.Setenv("DV_PAGE_TEST_VALUE", "configured")

		if got := env("DV_PAGE_TEST_VALUE", "fallback"); got != "configured" {
			t.Fatalf("env() = %q, want %q", got, "configured")
		}
	})
}

func TestIntInRange(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		fallback int
		min      int
		max      int
		want     int
		wantErr  string
	}{
		{name: "unset uses fallback", fallback: 2026, min: 1970, max: 9999, want: 2026},
		{name: "parsed", value: "2030", fallback: 2026, min: 1970, max: 9999, want: 2030},
		{name: "invalid", value: "twenty", fallback: 2026, min: 1970, max: 9999, wantErr: "must be an integer"},
		{name: "below range", value: "0", fallback: 128, min: 1, max: 4096, wantErr: "must be within [1, 4096]"},
		{name: "above range", value: "99999", fallback: 8080, min: 1, max: 65535, wantErr: "must be within [1, 65535]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("DV_PAGE_TEST_INT", tt.value)

			got, err := intInRange("DV_PAGE_TEST_INT", tt.fallback, tt.min, tt.max)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("intInRange() = (%d, %v), want error containing %q", got, err, tt.wantErr)
				}

				return
			}

			if err != nil || got != tt.want {
				t.Fatalf("intInRange() = (%d, %v), want (%d, nil)", got, err, tt.want)
			}
		})
	}
}

func TestReadyzToken(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "unset means open"},
		{name: "valid", value: "your_readyz_token"},
		{name: "too short", value: "short", wantErr: true},
		{name: "uppercase refused", value: "Your_Readyz_Token", wantErr: true},
		{name: "punctuation refused", value: "your-readyz-token", wantErr: true},
		{name: "too long", value: strings.Repeat("a", 129), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("READYZ_TOKEN", tt.value)

			got, err := readyzToken()
			if tt.wantErr {
				if err == nil {
					t.Fatalf("readyzToken() = (%q, nil), want an error", got)
				}

				return
			}
			if err != nil || got != tt.value {
				t.Fatalf("readyzToken() = (%q, %v), want (%q, nil)", got, err, tt.value)
			}
		})
	}
}

func TestListenAddress(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		t.Setenv("HOST", "")
		t.Setenv("PORT", "")

		address, err := listenAddress()
		if err != nil || address != "127.0.0.1:8080" {
			t.Fatalf("listenAddress() = (%q, %v), want 127.0.0.1:8080", address, err)
		}
	})

	t.Run("rejects a hostname", func(t *testing.T) {
		t.Setenv("HOST", "localhost")
		t.Setenv("PORT", "8080")

		if _, err := listenAddress(); err == nil || !strings.Contains(err.Error(), "must be an ip address") {
			t.Fatalf("listenAddress() = %v, want an ip complaint", err)
		}
	})

	t.Run("rejects a bad port", func(t *testing.T) {
		t.Setenv("HOST", "0.0.0.0")
		t.Setenv("PORT", "70000")

		if _, err := listenAddress(); err == nil || !strings.Contains(err.Error(), "must be within") {
			t.Fatalf("listenAddress() = %v, want a port complaint", err)
		}
	})
}

func TestLoadTimezone(t *testing.T) {
	if timezone, err := loadTimezone("Asia/Shanghai"); err != nil || timezone.String() != "Asia/Shanghai" {
		t.Fatalf("loadTimezone(Asia/Shanghai) = (%v, %v)", timezone, err)
	}

	if _, err := loadTimezone("Nowhere/Unknown"); err == nil {
		t.Fatal("loadTimezone(Nowhere/Unknown) = nil error, want a complaint")
	}
}

func TestNewLoggerWritesTimestampJSON(t *testing.T) {
	var output bytes.Buffer
	timezone, err := loadTimezone("Asia/Shanghai")
	if err != nil {
		t.Fatalf("loadTimezone() = %v", err)
	}

	newLogger(&output, timezone).Info("request", "client_ip", "203.0.113.9")

	var line struct {
		Timestamp string `json:"timestamp"`
		Level     string `json:"level"`
		Msg       string `json:"msg"`
		IP        string `json:"client_ip"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(output.Bytes()), &line); err != nil {
		t.Fatalf("decode log line %q: %v", output.String(), err)
	}

	if line.Msg != "request" || line.IP != "203.0.113.9" || line.Level != "INFO" {
		t.Fatalf("log line = %+v, want the request attributes", line)
	}

	stamp, err := time.Parse(time.RFC3339, line.Timestamp)
	if err != nil {
		t.Fatalf("timestamp %q is not RFC 3339: %v", line.Timestamp, err)
	}
	if _, offset := stamp.Zone(); offset != 8*60*60 {
		t.Fatalf("timestamp offset = %d, want Asia/Shanghai (+08:00)", offset)
	}
}

func TestDistFSHasFrontendArtifacts(t *testing.T) {
	assets, err := distFS()
	if err != nil {
		t.Fatalf("distFS() = %v", err)
	}

	entries, err := fs.ReadDir(assets, ".")
	if err != nil {
		t.Fatalf("read embedded dist: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("embedded frontend dist is empty")
	}
}
