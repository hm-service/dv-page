# dv-page

Daveny 的个人主页。交付物是一个静态二进制：前端产物 `go:embed` 进去，名字与头像由接口给（头像启动时读进内存），运行时不写任何文件。

![封面：简体中文](docs/screen-hero-zh.png)

![封面：英文](docs/screen-hero-en.png)

![第二屏：备案信息](docs/screen-details.png)

| 技术栈 | 版本                       |
| ------ | -------------------------- |
| Go     | 1.27（纯标准库，无 CGO）   |
| Node   | 26                         |
| pnpm   | 12                         |
| Vue    | 3 + Vite + TypeScript      |

## 快速开始

```bash
just build     # 构建前端 → 交叉编译 amd64/arm64 到 .dist/
just serve     # 本机跑完整产物（监听 127.0.0.1:8080）
```

前端开发：一个终端 `just run`（只跑后端，静态资源用占位），另一个终端 `cd frontend && pnpm dev`；Vite 把 `/api` 代理到 `127.0.0.1:8080`。干净检出时占位 dist 只够把接口跑起来：页面是 404、`/readyz` 是 503，要整页就跑 `just serve`。

## 命令

`just --list` 看全部；常用：

| 命令              | 说明                                                       |
| ----------------- | ---------------------------------------------------------- |
| `just build`      | 构建前端并交叉编译全部架构到 `.dist/dv-page-<os>-<arch>`   |
| `just serve`      | 先构建前端再 `go run .`，本机跑真实产物                    |
| `just run`        | 只跑后端（前端走 `pnpm dev`）                              |
| `just check`      | 提交前：`go vet` + `go test` + 前端 `vue-tsc` 类型检查     |
| `just test`       | `go test ./... -count=1`                                   |
| `just cover`      | 覆盖率报告（`go tool cover -func`）                        |
| `just fmt`        | `go fmt ./...`                                             |
| `just clean`      | 清掉 `.dist/` 与仓库内的 Go 缓存                           |

## 接口

| 方法 | 路径           | 说明                                                                     |
| ---- | -------------- | ------------------------------------------------------------------------ |
| GET  | `/api/profile` | `{name, avatar}`，`avatar` 是头像图片的 base64；带 ETag 的长缓存          |
| GET  | `/api/site`    | `{icp, icpUrl, copyrightYear, lang}`，`lang` 是协商结果                   |
| GET  | `/livez`       | 进程存活，不查依赖、不记日志                                             |
| GET  | `/readyz`      | 就绪（内嵌前端产物在不在），支持 `?token=`，每次调用记一条审计日志        |
| GET  | `/*`           | 前端静态文件；`assets/` 长缓存，其余 `no-cache`，无扩展名路径回退入口     |

```bash
curl -s localhost:8080/api/profile | head -c 120
# {"name":"Daveny","avatar":"/9j/4AAQSkZJRgABAQAAAQABAAD..."}
curl -s localhost:8080/api/site
# {"icp":"蜀ICP备2026015939号-1","icpUrl":"https://beian.miit.gov.cn/","copyrightYear":2026,"lang":"zh-CN"}
```

头像在启动时读进内存：`MY_NAME` 定名字，`AVATAR_FILE` 指头像文件（留空就用 `builtin/avatar.jpg` 那张内置图）。应用无状态、不写文件，换头像只要挂一个只读文件重启，不用重建镜像：

```bash
podman run -e MY_NAME=Daveny -e AVATAR_FILE=/etc/dv-page/avatar.png -v ./avatar.png:/etc/dv-page/avatar.png:ro dv-page
```

## 环境变量

| 变量名                | 默认值                            | 说明                                      |
| --------------------- | --------------------------------- | ----------------------------------------- |
| `HOST` / `PORT`       | `127.0.0.1` / `8080`              | 监听地址与端口                            |
| `READYZ_TOKEN`        | 空                                | 非空时 `/readyz` 必须带 `?token=`          |
| `TRUSTED_PROXIES`     | `127.0.0.1/32,::1/128`            | 可信反代段，决定是否采信 `X-Forwarded-For` |
| `TZ`                  | 容器不给就 UTC                    | 日志时间戳时区                            |
| `GOMEMLIMIT`          | 空                                | Go 运行时直接读取                          |
| `MY_NAME`             | `Daveny`                          | 主页展示的名字（也用作版权署名）          |
| `AVATAR_FILE`         | 空（用内置 `builtin/avatar.jpg`） | 头像文件路径，启动时读进内存              |
| `SITE_ICP`            | `蜀ICP备2026015939号-1`           | 备案号                                    |
| `SITE_ICP_URL`        | `https://beian.miit.gov.cn/`      | 备案号链接，必须是绝对 http(s) URL        |
| `SITE_COPYRIGHT_YEAR` | 当年                              | 版权年份，1970–9999                       |

配置在启动时一次性校验，不合法就直接起不来（`AVATAR_FILE` 读不到也是）。应用本身不写任何文件，没有 `DATA_DIR` 这类目录。页面与接口的设计写在 [DESIGN.md](DESIGN.md)。

## 测试与覆盖率

`just check` 全绿；实测覆盖率（`go test ./... -count=1 -cover`）：

| 包                  | 覆盖率 |
| ------------------- | ------ |
| `dv-page`（接线）   | 34.0%  |
| `dv-page/api`       | 91.7%  |
| `dv-page/app`       | 96.8%  |
| `dv-page/core`      | 88.7%  |
| `dv-page/httpx`     | 93.9%  |
| `dv-page/i18n`      | 97.4%  |
| 全模块合计          | 76.6%  |

前端改动后用 `tools/shot.mjs` 实跑浏览器截图并转发控制台错误（无头 Chrome 默认 `en-US`，验中文要显式设 `localStorage.lang`）：

```bash
just serve &
node tools/shot.mjs --url http://127.0.0.1:8080/ --out .shots/hero.png \
  --wait 4500 --pre-eval "localStorage.setItem('lang','zh-CN')" --console
```

页面里有调试入口 `window.__app`，可用 CDP 直接读 `state()`。

## 部署

镜像由 CI 双架构构建并推送（`ghcr.io/<owner>/dv-page`，日期 + `latest` 两个 tag）。Pod 描述在 [k8s.yaml](k8s.yaml)：

```bash
podman kube play k8s.yaml
```

容器以非 root 运行、`readOnlyRootFilesystem`、`drop: ALL`，不挂任何卷；探针 `/livez` 与 `/readyz?token=...` 各一。

## 目录

| 路径                  | 说明                                                       |
| --------------------- | ---------------------------------------------------------- |
| `main.go`             | 入口：读环境变量 → 建应用容器 → 挂路由 → 监听              |
| `web.go`              | `go:embed frontend/dist`，把前端产物编进二进制             |
| `api/`                | HTTP 层：路由、中间件、语言协商、探针、画像 / 站点接口、静态托管 |
| `app/`                | 用例层：启动校验、加载头像、就绪判据                       |
| `core/`               | 纯逻辑：画像与站点校验、头像读取、文件权限、恒定时间比较    |
| `httpx/`              | HTTP 小工具：JSON / 错误响应、真实客户端 IP                |
| `i18n/`               | 后端文案表与语言协商                                       |
| `builtin/`            | 随二进制发布的内置资源（默认头像）                         |
| `frontend/`           | 前端工程（pnpm + Vite + Vue + TS），产物 `frontend/dist`   |
| `internal/testenv/`   | 测试脚手架：假前端产物、假头像、现成路由                   |
| `tools/shot.mjs`      | 零依赖 CDP 截图工具                                        |
| `docs/`               | README 引用的截图                                          |
| `.ref/` `.prototype/` | 本地参考：模板 clone、重写前的静态页（都不入库）           |
