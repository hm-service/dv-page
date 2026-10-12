package core

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
)

// Avatar 是启动时读进内存的头像：原始字节与内容类型都只存一份，全进程共享。
type Avatar struct {
	Bytes       []byte
	ContentType string

	// etag 是内容的摘要，启动时算一次；换图即换 ETag。
	etag string
}

// NewAvatar 在启动时把关：非空、而且确实是一张图片。
func NewAvatar(raw []byte) (Avatar, error) {
	if len(raw) == 0 {
		return Avatar{}, errors.New("avatar is empty")
	}

	contentType := http.DetectContentType(raw)
	if !strings.HasPrefix(contentType, "image/") {
		return Avatar{}, fmt.Errorf("avatar is not an image (detected %s)", contentType)
	}

	sum := sha256.Sum256(raw)

	return Avatar{
		Bytes:       raw,
		ContentType: contentType,
		etag:        `"` + hex.EncodeToString(sum[:16]) + `"`,
	}, nil
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

// ETag 供响应头与条件请求使用。
func (a Avatar) ETag() string {
	return a.etag
}

// Profile 是主页要的全部事实：人（名字、头像）与站点（备案、版权）。
type Profile struct {
	Name   string
	Avatar Avatar
	Site   Site

	// etag 覆盖 JSON 档案里的内容（名字与站点事实）；头像走自己的接口，用头像自己的 ETag。
	etag string
}

// NewProfile 拼出档案并算好 ETag。
func NewProfile(name string, avatar Avatar, site Site) (Profile, error) {
	if strings.TrimSpace(name) == "" {
		return Profile{}, errors.New("profile name must not be empty")
	}
	if len(avatar.Bytes) == 0 {
		return Profile{}, errors.New("profile avatar must not be empty")
	}
	if err := site.Validate(); err != nil {
		return Profile{}, err
	}

	sum := func() string {
		digest := sha256.New()
		for _, part := range []string{name, site.ICP, site.ICPURL, strconv.Itoa(site.CopyrightYear)} {
			digest.Write([]byte(part))
			digest.Write([]byte{0}) // 分隔符，免得字段拼起来产生歧义
		}

		return `"` + hex.EncodeToString(digest.Sum(nil)[:16]) + `"`
	}()

	return Profile{
		Name:   name,
		Avatar: avatar,
		Site:   site,
		etag:   sum,
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
