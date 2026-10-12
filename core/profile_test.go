package core

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 一张够 DetectContentType 认出来的最小 PNG：只有签名，不追求能解码。
var pngSignature = []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}

func mustAvatar(t *testing.T, raw []byte) Avatar {
	t.Helper()

	avatar, err := NewAvatar(raw)
	if err != nil {
		t.Fatalf("NewAvatar() = %v", err)
	}

	return avatar
}

func TestNewAvatar(t *testing.T) {
	t.Run("image bytes", func(t *testing.T) {
		avatar := mustAvatar(t, pngSignature)

		if string(avatar.Bytes) != string(pngSignature) {
			t.Fatalf("Bytes = %v, want the raw bytes", avatar.Bytes)
		}
		if want := base64.StdEncoding.EncodeToString(pngSignature); avatar.Base64 != want {
			t.Fatalf("Base64 = %q, want %q", avatar.Base64, want)
		}
	})

	t.Run("empty", func(t *testing.T) {
		if _, err := NewAvatar(nil); err == nil || !strings.Contains(err.Error(), "avatar is empty") {
			t.Fatalf("NewAvatar() = %v, want an empty complaint", err)
		}
	})

	t.Run("not an image", func(t *testing.T) {
		if _, err := NewAvatar([]byte("not an image at all")); err == nil || !strings.Contains(err.Error(), "not an image") {
			t.Fatalf("NewAvatar() = %v, want an image complaint", err)
		}
	})
}

func TestLoadAvatarFile(t *testing.T) {
	t.Run("png file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "avatar.png")
		if err := os.WriteFile(path, pngSignature, 0o600); err != nil {
			t.Fatalf("write avatar: %v", err)
		}

		avatar, err := LoadAvatarFile(path)
		if err != nil {
			t.Fatalf("LoadAvatarFile() = %v", err)
		}
		if want := base64.StdEncoding.EncodeToString(pngSignature); avatar.Base64 != want {
			t.Fatalf("Base64 = %q, want %q", avatar.Base64, want)
		}
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := LoadAvatarFile(filepath.Join(t.TempDir(), "absent.jpg"))
		if err == nil || !strings.Contains(err.Error(), "read avatar file") {
			t.Fatalf("LoadAvatarFile() = %v, want a read complaint", err)
		}
	})

	t.Run("not an image", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "avatar.jpg")
		if err := os.WriteFile(path, []byte("plain text"), 0o600); err != nil {
			t.Fatalf("write avatar: %v", err)
		}

		if _, err := LoadAvatarFile(path); err == nil || !strings.Contains(err.Error(), "not an image") {
			t.Fatalf("LoadAvatarFile() = %v, want an image complaint", err)
		}
	})
}

func TestNewProfile(t *testing.T) {
	avatar := mustAvatar(t, pngSignature)

	t.Run("valid", func(t *testing.T) {
		profile, err := NewProfile("Daveny", avatar)
		if err != nil {
			t.Fatalf("NewProfile() = %v", err)
		}
		if profile.Name != "Daveny" {
			t.Fatalf("Name = %q, want %q", profile.Name, "Daveny")
		}
		if profile.ETag() == "" || !strings.HasPrefix(profile.ETag(), `"`) {
			t.Fatalf("ETag = %q, want a quoted etag", profile.ETag())
		}
	})

	tests := []struct {
		name    string
		profile string
		avatar  Avatar
		wantErr string
	}{
		{name: "blank name", profile: " ", avatar: avatar, wantErr: "profile name must not be empty"},
		{name: "empty avatar", profile: "Daveny", avatar: Avatar{}, wantErr: "profile avatar must not be empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewProfile(tt.profile, tt.avatar); err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("NewProfile() = %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestProfileETagTracksNameAndAvatar(t *testing.T) {
	first := mustAvatar(t, pngSignature)
	second := mustAvatar(t, append(append([]byte{}, pngSignature...), 0x00))

	base, err := NewProfile("Daveny", first)
	if err != nil {
		t.Fatalf("NewProfile() = %v", err)
	}
	same, err := NewProfile("Daveny", first)
	if err != nil {
		t.Fatalf("NewProfile() = %v", err)
	}
	renamed, err := NewProfile("Daveny Renamed", first)
	if err != nil {
		t.Fatalf("NewProfile() = %v", err)
	}
	replaced, err := NewProfile("Daveny", second)
	if err != nil {
		t.Fatalf("NewProfile() = %v", err)
	}

	if base.ETag() != same.ETag() {
		t.Fatalf("same input produced different etags: %q vs %q", base.ETag(), same.ETag())
	}
	if base.ETag() == renamed.ETag() {
		t.Fatal("renaming must change the etag")
	}
	if base.ETag() == replaced.ETag() {
		t.Fatal("replacing the avatar must change the etag")
	}
}

func TestProfileNotModified(t *testing.T) {
	profile, err := NewProfile("Daveny", mustAvatar(t, pngSignature))
	if err != nil {
		t.Fatalf("NewProfile() = %v", err)
	}

	tests := []struct {
		name        string
		ifNoneMatch string
		want        bool
	}{
		{name: "matching etag", ifNoneMatch: profile.ETag(), want: true},
		{name: "padded etag", ifNoneMatch: " " + profile.ETag() + " ", want: true},
		{name: "other etag", ifNoneMatch: `"something-else"`, want: false},
		{name: "empty header", ifNoneMatch: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := profile.NotModified(tt.ifNoneMatch); got != tt.want {
				t.Fatalf("NotModified(%q) = %v, want %v", tt.ifNoneMatch, got, tt.want)
			}
		})
	}
}
