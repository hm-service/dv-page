package core

import "testing"

func TestSecureEqual(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		want bool
	}{
		{name: "equal", a: "readyz-token", b: "readyz-token", want: true},
		{name: "different content", a: "readyz-token", b: "readyz-tokeN", want: false},
		{name: "different length", a: "readyz-token", b: "readyz", want: false},
		{name: "both empty", a: "", b: "", want: true},
		{name: "one empty", a: "", b: "x", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SecureEqual(tt.a, tt.b); got != tt.want {
				t.Fatalf("SecureEqual(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}
