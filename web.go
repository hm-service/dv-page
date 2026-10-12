package main

import (
	"embed"
	"io/fs"
)

// all: 前缀让 dist 里的下划线、点号开头的文件也能进二进制。
//
//go:embed all:frontend/dist
var embeddedDist embed.FS

// distFS 把前端构建产物暴露成以 dist 为根的 FS。
func distFS() (fs.FS, error) {
	return fs.Sub(embeddedDist, "frontend/dist")
}
