# Komari 网站与 AstrBot 可用性监测（Linux，前后端分离）

仓库名称：`komari-service-monitor-addon`。本仓库保存当前开发源码，尚未创建稳定版 Release。

Lumina 主页与完整控制台的构建、组装和宝塔配置见 [Lumina 适配说明](komari-theme-Lumina/README-SERVICES.md)。此目录还包含 `komari-theme-Lumina/` 主题源码，以及 `toolchain/assemble-lumina.cjs` 静态前端组装脚本。

这是基于 Komari 及相关前端的功能扩展，非官方发行版。上游链接、修改范围和公开再分发前需要确认的许可证事项见 [上游来源与许可核对](UPSTREAM.md)。当前仍有用户报告的控制台 Slot 组件错误待修复，尚不应作为稳定版本发布。

宝塔用户可使用预编译部署包，按 [宝塔部署指南](deploy/baota/README.md) 上传前后端文件并配置 Nginx，无需在服务器上编译。

主页现在会在服务器列表下方同时显示公开的网站和 AstrBot 状态。已部署用户可按 [前端更新说明](deploy/baota/UPDATE-FRONTEND.md) 只更新前端，无需重新初始化或更换后端。

项目包含两个独立源码目录：

- `komari/`：Go API、数据库、定时探测和状态历史。
- `komari-web/`：React + TypeScript 前端，管理入口 `/admin/service-monitor`，公开页 `/service-monitor`。
- `deploy/`：Linux Docker 构建和 Nginx 配置。

## Linux 部署

安装 Docker Engine 与 Docker Compose 插件后，在本目录执行：

```sh
docker compose up -d --build
```

后端构建默认通过 `https://goproxy.cn` 下载 Go 依赖，并输出下载请求。如果部署网络更适合官方代理，可先执行 `docker compose build --build-arg GOPROXY=https://proxy.golang.org,direct backend`，再执行 `docker compose up -d`。主机环境的 `go env` 不会自动改变 Docker 构建容器中的配置。

首次访问 `http://服务器IP:8080/install` 完成原有 Komari 初始化，然后登录管理后台。在“运维”菜单中选择“服务可用性”，添加网站或 AstrBot。

修改公开端口：

```sh
KOMARI_WEB_PORT=80 docker compose up -d --build
```

前端 Nginx 单独提供静态网页，并将 `/api/` 和 WebSocket 代理至内网后端。后端使用 `api_only` 构建标记，不打包或提供 SPA 网页。两个服务分别构建、运行和更新；浏览器通过同源代理使用登录 Cookie，无需将 AstrBot 密钥放在前端。

数据库保存在 `komari-data` 命名卷；更新代码后重新执行部署命令即可。请保留数据卷；`docker compose down -v` 会删除数据。

## 监测配置

网站填写完整 URL，可设置状态码、关键词、超时和检查间隔。AstrBot 填 Dashboard 根地址，例如 `http://host.docker.internal:6185`。容器内 `127.0.0.1` 指向 Komari 容器自身；AstrBot 在宿主机上运行时应使用宿主机地址，并确保监听地址允许容器访问。

AstrBot v1 的 API Key 需要 `bot` 权限：服务检查使用只读机器人列表，指定实例时读取运行状态。开发者 API Key 不支持 `system` 权限。也支持旧版 `/api/stat/get` 的 Dashboard JWT。默认不公开监测；勾选后出现在公开状态页。

功能、状态判定和 API 详见 [监测说明](komari/docs/service-monitor.md)。

## 分别构建和部署

```sh
docker build -f deploy/backend.Dockerfile -t komari-api .
docker build -f deploy/frontend.Dockerfile -t komari-web .
```

如果部署到不同主机，修改 `deploy/nginx.conf` 的 `proxy_pass`，指向后端内网地址，保留 API 和 WebSocket 代理配置，然后重新构建前端镜像。

此部署使用独立构建的默认前端。前端外观和自定义页面通过 `komari-web/` 更新，不通过后端替换嵌入式主题来更新。

## 本地开发（Linux）

```sh
# 终端 1：Go 1.25+、gcc；复制的 JSON 只是默认主题配置定义
mkdir -p komari/web/public/defaultTheme
cp komari-web/komari-theme.json komari/web/public/defaultTheme/
cd komari
go run -tags api_only . server

# 终端 2：Node.js 22.16+、pnpm 11.25
cd komari-web
pnpm install --frozen-lockfile --ignore-scripts
pnpm dev
```

Vite 开发代理的配置方式沿用 `komari-web/vite.config.ts`。生产部署推荐使用上面的 Docker Compose。

已提供预编译 Linux amd64 API 服务和前端静态包时，也可以分别运行后端与 Nginx；配置可参照 `deploy/nginx.conf`，将其中的 `backend:25774` 替换成后端地址。Linux 后端首次运行前需 `chmod +x komari-linux-amd64`，通过 `./komari-linux-amd64 server` 启动。

当前工作区中的 `astrbot-reference/` 与 `toolchain/` 是接口核对及验证工具，不属于部署内容，也不会进入 Docker 镜像。
