package app

import (
	"errors"
	"fmt"
	"io/fs"

	"dv-page/core"
)

// Config 是应用启动时需要的全部外部输入。
type Config struct {
	// Name 来自 MY_NAME，是主页上展示的名字。
	Name string
	// AvatarFile 指定头像文件；为空则用 DefaultAvatar（随二进制发布的内置图）。
	AvatarFile    string
	DefaultAvatar []byte
	Site          core.Site
	Assets        fs.FS
}

// App 是用例层：启动时一次性把关、把头像读进内存，之后只读。
type App struct {
	profile core.Profile
	assets  fs.FS
}

// New 校验配置并加载头像；任何一项不成立都应当让进程起不来。
//
// 前端产物只要求「嵌进去了东西」：干净检出时 justfile 会放一个占位 dist，让接口能单独跑起来，
// 页面是否真的可用交给 /readyz 断言。
func New(cfg Config) (*App, error) {
	avatar, err := loadAvatar(cfg)
	if err != nil {
		return nil, err
	}

	profile, err := core.NewProfile(cfg.Name, avatar, cfg.Site)
	if err != nil {
		return nil, err
	}
	if cfg.Assets == nil {
		return nil, errors.New("assets must not be nil")
	}

	entries, err := fs.ReadDir(cfg.Assets, ".")
	if err != nil {
		return nil, fmt.Errorf("frontend assets: %w", err)
	}
	if len(entries) == 0 {
		return nil, errors.New("frontend assets are empty")
	}

	return &App{profile: profile, assets: cfg.Assets}, nil
}

func loadAvatar(cfg Config) (core.Avatar, error) {
	if cfg.AvatarFile != "" {
		return core.LoadAvatarFile(cfg.AvatarFile)
	}

	avatar, err := core.NewAvatar(cfg.DefaultAvatar)
	if err != nil {
		return core.Avatar{}, fmt.Errorf("builtin avatar: %w", err)
	}

	return avatar, nil
}

// Profile 是主页要的全部事实；名字、头像、备案在启动时就已定好并读进内存。
func (a *App) Profile() core.Profile {
	return a.profile
}

// Ready 是 /readyz 的判据。应用没有外部依赖，唯一要确认的是内嵌的前端产物还在：
// 本机用占位 dist 只跑接口时，它就应当报未就绪。
func (a *App) Ready() error {
	if _, err := fs.Stat(a.assets, "index.html"); err != nil {
		return fmt.Errorf("frontend assets: %w", err)
	}

	return nil
}
