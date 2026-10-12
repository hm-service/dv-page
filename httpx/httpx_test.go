package httpx

import (
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}

func mustProxies(t *testing.T, spec string) []*net.IPNet {
	t.Helper()

	networks, err := TrustedProxies(spec)
	if err != nil {
		t.Fatalf("TrustedProxies(%q) = %v", spec, err)
	}

	return networks
}

func TestTrustedProxies(t *testing.T) {
	t.Run("cidr and bare ip", func(t *testing.T) {
		networks, err := TrustedProxies("127.0.0.1/32, ::1, 10.89.2.0/24")
		if err != nil {
			t.Fatalf("TrustedProxies() = %v, want nil", err)
		}
		if len(networks) != 3 {
			t.Fatalf("len = %d, want 3", len(networks))
		}
	})

	t.Run("empty spec", func(t *testing.T) {
		networks, err := TrustedProxies(" , ")
		if err != nil {
			t.Fatalf("TrustedProxies() = %v, want nil", err)
		}
		if len(networks) != 0 {
			t.Fatalf("len = %d, want 0", len(networks))
		}
	})

	t.Run("invalid entry", func(t *testing.T) {
		if _, err := TrustedProxies("nope/24"); err == nil {
			t.Fatal("TrustedProxies() = nil, want error")
		}
	})
}

func TestClientIP(t *testing.T) {
	tests := []struct {
		name    string
		trusted string
		remote  string
		headers []string
		want    string
	}{
		{
			name:    "untrusted peer ignores forwarded header",
			trusted: "127.0.0.1/32",
			remote:  "192.0.2.10:5555",
			headers: []string{"203.0.113.9"},
			want:    "192.0.2.10",
		},
		{
			name:    "trusted peer uses rightmost untrusted hop",
			trusted: "10.0.0.0/8",
			remote:  "10.0.0.5:5555",
			headers: []string{"203.0.113.9, 10.0.0.7"},
			want:    "203.0.113.9",
		},
		{
			name:    "spoofed first hop is skipped when chain is trusted",
			trusted: "10.0.0.0/8,203.0.113.0/24",
			remote:  "10.0.0.5:5555",
			headers: []string{"203.0.113.9", "198.51.100.4"},
			want:    "198.51.100.4",
		},
		{
			name:    "all hops trusted returns the leftmost",
			trusted: "10.0.0.0/8",
			remote:  "10.0.0.5:5555",
			headers: []string{"10.0.0.9, 10.0.0.7"},
			want:    "10.0.0.9",
		},
		{
			name:    "trusted peer without header",
			trusted: "127.0.0.1/32",
			remote:  "127.0.0.1:5555",
			want:    "127.0.0.1",
		},
		{
			name:    "no trusted proxies configured",
			trusted: "",
			remote:  "127.0.0.1:5555",
			headers: []string{"203.0.113.9"},
			want:    "127.0.0.1",
		},
		{
			name:    "garbage forwarded hop is ignored",
			trusted: "10.0.0.0/8",
			remote:  "10.0.0.5:5555",
			headers: []string{"not-an-ip, 198.51.100.4"},
			want:    "198.51.100.4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			request.RemoteAddr = tt.remote
			for _, hop := range tt.headers {
				request.Header.Add("X-Forwarded-For", hop)
			}

			if got := ClientIP(request, mustProxies(t, tt.trusted)); got != tt.want {
				t.Fatalf("ClientIP() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestClientIPWithoutPort(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.RemoteAddr = "127.0.0.1"

	if got := ClientIP(request, nil); got != "127.0.0.1" {
		t.Fatalf("ClientIP() = %q, want %q", got, "127.0.0.1")
	}
}

func TestWriteError(t *testing.T) {
	recorder := httptest.NewRecorder()

	WriteError(recorder, discardLogger(), http.StatusNotFound, "en", "error.not-found")

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
	if contentType := recorder.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "application/json") {
		t.Fatalf("content type = %q, want json", contentType)
	}
	if body := recorder.Body.String(); !strings.Contains(body, "resource not found") || !strings.Contains(body, "error.not-found") {
		t.Fatalf("body = %q, want key and rendered message", body)
	}
}
