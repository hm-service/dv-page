package core

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
)

// Avatar 是启动时读进内存的头像：字节留着算 ETag，base64 是每个请求直接复用的那份。
type Avatar struct {
	Bytes  []byte
	Base64 string
}

// NewAvatar 在启动时把关：非空、而且确实是一张图片。
func NewAvatar(raw []byte) (Avatar, error) {
	if len(raw) == 0 {
		return Avatar{}, errors.New("avatar is empty")
	}

	if contentType := http.DetectContentType(raw); !strings.HasPrefix(contentType, "image/") {
		return Avatar{}, fmt.Errorf("avatar is not an image (detected %s)", contentType)
	}

	return Avatar{Bytes: raw, Base64: base64.StdEncoding.EncodeToString(raw)}, nil
}

// LoadAvatarFile 从头像文件读一次；之后进程内不再碰磁盘。
func LoadAvatarFile(path string) (Avatar, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Avatar{}, fmt.Errorf("read avatar file %q: %w", path, err)
	}

	avatar, err := NewAvatar(raw)
	if err != nil {
		return Avatar{}, fmt.Errorf("avatar file %q: %w", path, err)
	}

	return avatar, nil
}

// Profile 是主页上「人」的部分。
type Profile struct {
	Name   string
	Avatar Avatar

	// etag 由名字与头像内容推出，启动时算一次；换名字或换图都会让旧缓存失效。
	etag string
}

// NewProfile 拼出档案并算好 ETag。
func NewProfile(name string, avatar Avatar) (Profile, error) {
	if strings.TrimSpace(name) == "" {
		return Profile{}, errors.New("profile name must not be empty")
	}
	if len(avatar.Bytes) == 0 || avatar.Base64 == "" {
		return Profile{}, errors.New("profile avatar must not be empty")
	}

	sum := sha256.Sum256([]byte(name + "\x00" + string(avatar.Bytes)))

	return Profile{
		Name:   name,
		Avatar: avatar,
		etag:   `"` + hex.EncodeToString(sum[:16]) + `"`,
	}, nil
}

// ETag 供响应头与条件请求使用。
func (p Profile) ETag() string {
	return p.etag
}

// NotModified 判断客户端缓存是否还有效。
func (p Profile) NotModified(ifNoneMatch string) bool {
	return p.etag != "" && strings.TrimSpace(ifNoneMatch) == p.etag
}
