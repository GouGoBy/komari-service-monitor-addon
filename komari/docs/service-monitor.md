# 网站与 AstrBot 可用性监测

部署采用 Linux 前后端分离：`komari/` 提供 API，`komari-web/` 提供 React 网页，由 Nginx 代理 API。根目录 `compose.yaml` 可分别构建并启动两个服务。启动修改后的 Komari 后，数据库自动创建 `service_monitors` 和 `service_checks` 表，无需更新 Agent。

## 使用

1. 通过原有登录页登录 Komari 管理员账号。
2. 打开 `/admin/service-monitor`，点击“添加监测”。
3. 网站选择 HTTP(S)，填写完整 URL、检查间隔和超时。默认接受最终响应的 2xx 状态码，也可指定状态码和页面关键词。
4. AstrBot 填写 Dashboard 根地址（默认端口 6185），选择 API 版本和认证方式，填写密钥。编辑时密钥留空保留原值。
5. 按需勾选“在公开状态页展示”。启用且公开的监测会同时显示在主页 `/` 的服务器列表下方和 `/service-monitor` 详情页，默认新建监测不公开。

探测从 Komari 服务端发起，支持内网地址。默认验证 TLS 证书。间隔为 10–86400 秒，超时为 1–60 秒且须小于间隔。同一任务不重叠执行，同时最多执行 16 个探测；繁忙时跳过本轮。保存后立即检查，重启后恢复调度。

网站跟随至多 10 次重定向，检查最终状态码。关键词检查和 AstrBot JSON 响应限制为 2 MiB。AstrBot 拒绝跨源重定向，避免认证信息被转发到其他站点。

## AstrBot 接口与权限

基于 AstrBot 官方源码的接口约定：

- 当前 API：`GET /api/v1/bots`，需要具有 `bot` 权限的 API Key，使用 `X-API-Key` 请求头。响应须为 `status: "ok"`，且 `data.bots` 为数组。只要接口响应有效，即使未配置机器人也表示服务 API 可用。
- 原先保存的 `/api/v1/stats` 配置会自动采用上述列表接口探测，不需要重新创建任务。AstrBot 开发者 API Key 不开放 `system` 权限，不能使用该权限保护的统计接口进行通用 API Key 探测。
- 指定机器人实例：`GET /api/v1/bots/stats`，需要 `bot` 权限。填写 AstrBot 配置中的实例 ID，匹配 `data.platforms[].id`，仅 `status: "running"` 判为可用；实例缺失、停止、错误均判为不可用。
- 旧版 Dashboard：选择 `/api/stat/get` 和 Bearer 认证，填写 Dashboard JWT。JWT 过期会显示不可用，更新后恢复。旧版不支持本功能的实例检查。
- 反向代理前缀可放在根地址中，例如 `https://example.com/astrbot`。不要在根地址中填写 API 路径或查询参数。

此功能检查 API 可用性或平台适配器报告的运行状态，不发送聊天消息，也不证明消息平台、插件和模型供应商的完整调用链可用。

参考源码：[API 路由前缀](https://github.com/AstrBotDevs/AstrBot/blob/master/astrbot/dashboard/api/router.py)、[认证](https://github.com/AstrBotDevs/AstrBot/blob/master/astrbot/dashboard/api/auth.py)、[统计接口](https://github.com/AstrBotDevs/AstrBot/blob/master/astrbot/dashboard/api/stats.py)、[机器人接口](https://github.com/AstrBotDevs/AstrBot/blob/master/astrbot/dashboard/api/bots.py)、[平台状态](https://github.com/AstrBotDevs/AstrBot/blob/master/astrbot/core/platform/manager.py)。

## 状态和历史

页面显示正常、不可用、已暂停或等待检查/数据过期。最新结果超过 `2 × interval + timeout` 秒，或早于配置更新时间，则显示未知，避免旧成功记录长期显示正常。

24 小时可用率 = 最近 24 小时成功检查数 / 实际检查数；没有检查时显示空值。它是采样成功率，服务停机期间没有补写检查记录。历史保存在 Komari 配置数据库，每小时清理 30 天以前的数据；页面/API 返回最近 24 小时至多 1000 条记录。

## API

沿用 Komari 的管理员认证及 JSON-RPC 权限体系。REST 返回 `{ "status": "success", "data": ... }`。

| 方法 | 地址 | 用途 |
|---|---|---|
| GET | `/api/admin/services` | 配置、当前状态、24 小时统计 |
| POST | `/api/admin/services/save` | 创建或完整更新配置 |
| POST | `/api/admin/services/delete` | 删除配置及历史，参数 `{"id":1}` |
| GET | `/api/admin/services/history?id=1` | 管理员历史 |
| GET | `/api/services` | 公开且启用的服务状态 |
| GET | `/api/services/history?id=1` | 公开服务历史 |

网站配置示例：

```json
{"name":"官网","type":"http","url":"https://example.com","interval":60,"timeout":10,"expected_status":200,"keyword":"Welcome","enabled":true,"public":true}
```

AstrBot 配置示例：

```json
{"name":"AstrBot","type":"astrbot","url":"http://astrbot:6185","interval":60,"timeout":10,"auth_type":"apikey","api_path":"/api/v1/bots","token":"YOUR_API_KEY","bot_id":"","enabled":true,"public":false}
```

更新时携带 `id` 和完整配置，省略 `token` 可保留密钥。密钥存储在服务端配置数据库，查询不返回密钥；公开 API 不返回 URL、配置和诊断错误，且不可读取私有或暂停任务的历史。数据库和备份应按凭据文件保管。

对应 RPC 方法为 `admin:getServiceMonitors`、`admin:saveServiceMonitor`、`admin:deleteServiceMonitor`、`admin:getServiceHistory`、`public:getServiceMonitors`、`public:getServiceHistory`。管理和公开页面属于独立前端，后端使用 `-tags api_only` 构建时不嵌入或提供 SPA。

## 验证

使用项目要求的 Go 和 C 编译器，将 `komari-web/komari-theme.json` 复制至 `komari/web/public/defaultTheme/` 后，在后端目录执行（此 JSON 仅定义配置项）：

```sh
go test -tags api_only ./internal/availability ./web/rpc/jsonrpc ./web/router ./internal/server ./web/public
go build -tags api_only .
```
