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

func TestEnvInt(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		fallback int
		want     int
		wantErr  string
	}{
		{name: "unset uses fallback", fallback: 2026, want: 2026},
		{name: "parsed", value: "2030", fallback: 2026, want: 2030},
		{name: "invalid", value: "twenty", fallback: 2026, wantErr: "must be an integer"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("DV_PAGE_TEST_YEAR", tt.value)

			got, err := envInt("DV_PAGE_TEST_YEAR", tt.fallback)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("envInt() = (%d, %v), want error containing %q", got, err, tt.wantErr)
				}

				return
			}

			if err != nil || got != tt.want {
				t.Fatalf("envInt() = (%d, %v), want (%d, nil)", got, err, tt.want)
			}
		})
	}
}

func TestNewLoggerWritesRFC3339JSON(t *testing.T) {
	var output bytes.Buffer

	newLogger(&output).Info("request", "client_ip", "203.0.113.9")

	var line struct {
		Time  string `json:"time"`
		Level string `json:"level"`
		Msg   string `json:"msg"`
		IP    string `json:"client_ip"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(output.Bytes()), &line); err != nil {
		t.Fatalf("decode log line %q: %v", output.String(), err)
	}

	if line.Msg != "request" || line.IP != "203.0.113.9" || line.Level != "INFO" {
		t.Fatalf("log line = %+v, want the request attributes", line)
	}
	if _, err := time.Parse(time.RFC3339, line.Time); err != nil {
		t.Fatalf("time %q is not RFC 3339: %v", line.Time, err)
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
