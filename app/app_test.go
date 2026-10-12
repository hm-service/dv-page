package app

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"dv-page/builtin"
	"dv-page/core"
)

func assetsWithIndex() fstest.MapFS {
	return fstest.MapFS{
		"index.html": {Data: []byte("<!doctype html>")},
	}
}

func validConfig(t *testing.T) Config {
	t.Helper()

	return Config{
		Name:          "Daveny",
		DefaultAvatar: builtin.Avatar(),
		Site: core.Site{
			ICP:           "蜀ICP备2026015939号-1",
			ICPURL:        "https://beian.miit.gov.cn/",
			CopyrightYear: 2026,
		},
		Assets: assetsWithIndex(),
	}
}

func writeAvatarFile(t *testing.T, name string, raw []byte) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatalf("write avatar file: %v", err)
	}

	return path
}

func TestNew(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		cfg := validConfig(t)

		application, err := New(cfg)
		if err != nil {
			t.Fatalf("New() = %v, want nil", err)
		}

		profile := application.Profile()
		if profile.Name != "Daveny" {
			t.Fatalf("Profile().Name = %q, want %q", profile.Name, "Daveny")
		}
		if !bytes.Equal(profile.Avatar.Bytes, builtin.Avatar()) {
			t.Fatal("Profile().Avatar.Bytes do not match the builtin avatar")
		}
		if profile.Avatar.ContentType == "" {
			t.Fatal("Profile().Avatar.ContentType is empty")
		}
		if profile.ETag() == "" {
			t.Fatal("Profile().ETag() is empty")
		}
		if got := profile.Site.ICP; got != "蜀ICP备2026015939号-1" {
			t.Fatalf("Profile().Site.ICP = %q, want the configured icp", got)
		}
	})

	t.Run("blank name", func(t *testing.T) {
		cfg := validConfig(t)
		cfg.Name = "   "

		if _, err := New(cfg); err == nil || !strings.Contains(err.Error(), "profile name") {
			t.Fatalf("New() = %v, want a profile name complaint", err)
		}
	})

	t.Run("avatar file missing", func(t *testing.T) {
		cfg := validConfig(t)
		cfg.AvatarFile = filepath.Join(t.TempDir(), "absent.jpg")

		if _, err := New(cfg); err == nil || !strings.Contains(err.Error(), "read avatar file") {
			t.Fatalf("New() = %v, want a read complaint", err)
		}
	})

	t.Run("avatar file is not an image", func(t *testing.T) {
		cfg := validConfig(t)
		cfg.AvatarFile = writeAvatarFile(t, "avatar.jpg", []byte("plain text"))

		if _, err := New(cfg); err == nil || !strings.Contains(err.Error(), "not an image") {
			t.Fatalf("New() = %v, want an image complaint", err)
		}
	})

	t.Run("avatar file overrides the builtin avatar", func(t *testing.T) {
		replacement := []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}
		cfg := validConfig(t)
		cfg.AvatarFile = writeAvatarFile(t, "avatar.png", replacement)

		application, err := New(cfg)
		if err != nil {
			t.Fatalf("New() = %v", err)
		}

		if got := application.Profile().Avatar.Bytes; !bytes.Equal(got, replacement) {
			t.Fatalf("Profile().Avatar.Bytes = %v, want the replacement avatar", got)
		}
	})

	t.Run("builtin avatar missing", func(t *testing.T) {
		cfg := validConfig(t)
		cfg.DefaultAvatar = nil

		if _, err := New(cfg); err == nil || !strings.Contains(err.Error(), "builtin avatar") {
			t.Fatalf("New() = %v, want a builtin avatar complaint", err)
		}
	})

	t.Run("invalid site", func(t *testing.T) {
		cfg := validConfig(t)
		cfg.Site.ICP = ""

		if _, err := New(cfg); err == nil || !strings.Contains(err.Error(), "site icp") {
			t.Fatalf("New() = %v, want a site icp complaint", err)
		}
	})

	t.Run("missing assets", func(t *testing.T) {
		cfg := validConfig(t)
		cfg.Assets = nil

		if _, err := New(cfg); err == nil || !strings.Contains(err.Error(), "assets") {
			t.Fatalf("New() = %v, want an assets complaint", err)
		}
	})

	t.Run("empty assets", func(t *testing.T) {
		cfg := validConfig(t)
		cfg.Assets = fstest.MapFS{}

		if _, err := New(cfg); err == nil || !strings.Contains(err.Error(), "frontend assets are empty") {
			t.Fatalf("New() = %v, want an empty assets complaint", err)
		}
	})
}

func TestReady(t *testing.T) {
	t.Run("ready", func(t *testing.T) {
		application, err := New(validConfig(t))
		if err != nil {
			t.Fatalf("New() = %v", err)
		}

		if err := application.Ready(); err != nil {
			t.Fatalf("Ready() = %v, want nil", err)
		}
	})

	t.Run("entry file missing", func(t *testing.T) {
		cfg := validConfig(t)
		// 占位 dist：接口能起来，但页面还不存在。
		cfg.Assets = fstest.MapFS{"STUB": {Data: []byte{}}}

		application, err := New(cfg)
		if err != nil {
			t.Fatalf("New() = %v, want a placeholder dist to boot", err)
		}

		if err := application.Ready(); err == nil || !strings.Contains(err.Error(), "frontend assets") {
			t.Fatalf("Ready() = %v, want a frontend assets complaint", err)
		}
	})
}
