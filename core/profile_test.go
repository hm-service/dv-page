package core

import (
	"bytes"
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

		if !bytes.Equal(avatar.Bytes, pngSignature) {
			t.Fatalf("Bytes = %v, want the raw bytes", avatar.Bytes)
		}
		if avatar.ContentType != "image/png" {
			t.Fatalf("ContentType = %q, want image/png", avatar.ContentType)
		}
		if avatar.ETag() == "" || !strings.HasPrefix(avatar.ETag(), `"`) {
			t.Fatalf("ETag = %q, want a quoted etag", avatar.ETag())
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
		if !bytes.Equal(avatar.Bytes, pngSignature) {
			t.Fatal("Bytes do not match the file contents")
		}
		if avatar.ContentType != "image/png" {
			t.Fatalf("ContentType = %q, want image/png", avatar.ContentType)
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
	site := validSite()

	t.Run("valid", func(t *testing.T) {
		profile, err := NewProfile("Daveny", avatar, site)
		if err != nil {
			t.Fatalf("NewProfile() = %v", err)
		}
		if profile.Name != "Daveny" {
			t.Fatalf("Name = %q, want %q", profile.Name, "Daveny")
		}
		if profile.Site.ICP != site.ICP {
			t.Fatalf("Site.ICP = %q, want %q", profile.Site.ICP, site.ICP)
		}
		if profile.ETag() == "" || !strings.HasPrefix(profile.ETag(), `"`) {
			t.Fatalf("ETag = %q, want a quoted etag", profile.ETag())
		}
	})

	tests := []struct {
		name    string
		profile string
		avatar  Avatar
		mutate  func(*Site)
		wantErr string
	}{
		{name: "blank name", profile: " ", avatar: avatar, wantErr: "profile name must not be empty"},
		{name: "empty avatar", profile: "Daveny", avatar: Avatar{}, wantErr: "profile avatar must not be empty"},
		{name: "invalid site", profile: "Daveny", avatar: avatar, mutate: func(s *Site) { s.ICP = "" }, wantErr: "site icp must not be empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			site := validSite()
			if tt.mutate != nil {
				tt.mutate(&site)
			}

			if _, err := NewProfile(tt.profile, tt.avatar, site); err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("NewProfile() = %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestProfileETagTracksItsContent(t *testing.T) {
	first := mustAvatar(t, pngSignature)
	second := mustAvatar(t, append(append([]byte{}, pngSignature...), 0x00))
	site := validSite()

	build := func(t *testing.T, name string, avatar Avatar, site Site) Profile {
		t.Helper()

		profile, err := NewProfile(name, avatar, site)
		if err != nil {
			t.Fatalf("NewProfile() = %v", err)
		}

		return profile
	}

	base := build(t, "Daveny", first, site)
	same := build(t, "Daveny", first, site)

	renamedSite := validSite()
	renamedSite.ICP = "蜀ICP备2026015939号-2"
	renamed := build(t, "Daveny Renamed", first, site)
	replaced := build(t, "Daveny", second, site)
	refiled := build(t, "Daveny", first, renamedSite)

	if base.ETag() != same.ETag() {
		t.Fatalf("same input produced different etags: %q vs %q", base.ETag(), same.ETag())
	}
	for name, other := range map[string]Profile{
		"renaming":         renamed,
		"changing the icp": refiled,
	} {
		if base.ETag() == other.ETag() {
			t.Fatalf("%s must change the etag", name)
		}
	}

	// 头像已经不在档案 JSON 里：换图不该让档案的 ETag 变化，那是 /api/avatar 的事。
	if base.ETag() != replaced.ETag() {
		t.Fatal("replacing the avatar must not change the profile etag")
	}
}

func TestAvatarETagTracksContent(t *testing.T) {
	first := mustAvatar(t, pngSignature)
	same := mustAvatar(t, pngSignature)
	other := mustAvatar(t, append(append([]byte{}, pngSignature...), 0x00))

	if first.ETag() != same.ETag() {
		t.Fatalf("same bytes produced different etags: %q vs %q", first.ETag(), same.ETag())
	}
	if first.ETag() == other.ETag() {
		t.Fatal("different bytes must produce different etags")
	}
}

func TestProfileNotModified(t *testing.T) {
	profile, err := NewProfile("Daveny", mustAvatar(t, pngSignature), validSite())
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
