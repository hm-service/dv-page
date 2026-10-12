package api

import (
	"io"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"dv-page/httpx"
)

const indexFile = "index.html"

// 构建产物带内容哈希，可以长期缓存；入口文件必须每次回源。
const immutableCacheControl = "public, max-age=31536000, immutable"

func (s *server) handleStatic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		httpx.WriteError(w, s.logger(), http.StatusMethodNotAllowed, negotiateLanguage(r), "error.method-not-allowed")
		return
	}

	name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
	if name == "" || name == "." {
		name = indexFile
	}

	if file, err := s.opts.Assets.Open(name); err == nil {
		defer file.Close()
		if s.serveFile(w, r, name, file) {
			return
		}
	}

	// 单页应用回退：只有「像导航」的路径才回到入口，缺的静态资源仍按 404 处理。
	if !strings.Contains(path.Base(name), ".") {
		s.serveIndex(w, r)
		return
	}

	httpx.WriteError(w, s.logger(), http.StatusNotFound, negotiateLanguage(r), "error.not-found")
}

func (s *server) serveIndex(w http.ResponseWriter, r *http.Request) {
	file, err := s.opts.Assets.Open(indexFile)
	if err != nil {
		// 只可能是构建没产出入口文件（例如本机用占位 dist 跑接口），按缺资源处理。
		httpx.WriteError(w, s.logger(), http.StatusNotFound, negotiateLanguage(r), "error.not-found")
		return
	}
	defer file.Close()

	if !s.serveFile(w, r, indexFile, file) {
		httpx.WriteError(w, s.logger(), http.StatusInternalServerError, negotiateLanguage(r), "error.internal")
	}
}

// serveFile 报告自己是否真的把内容发出去了；目录与不可寻址的文件交给调用方兜底。
func (s *server) serveFile(w http.ResponseWriter, r *http.Request, name string, file fs.File) bool {
	info, err := file.Stat()
	if err != nil || info.IsDir() {
		return false
	}

	seeker, ok := file.(io.ReadSeeker)
	if !ok {
		return false
	}

	if strings.HasPrefix(name, "assets/") {
		w.Header().Set("Cache-Control", immutableCacheControl)
	} else {
		w.Header().Set("Cache-Control", "no-cache")
	}

	http.ServeContent(w, r, name, info.ModTime(), seeker)

	return true
}
