# AstrBot API Key 兼容与诊断修复

本次修复把服务 API Key 探测从需要 `system` 权限的统计接口改为 `GET /api/v1/bots`，使用开发者 API Key 支持的 `bot` 权限。指定机器人实例仍使用 `/api/v1/bots/stats`。已有任务自动兼容，无需重新创建，不会修改数据库。

同时细分了 DNS、证书不受信任、证书域名不匹配、证书过期、连接拒绝和重定向错误。密钥错误、权限不足、API 地址错误分别显示 HTTP 401、403、404 的诊断。错误内容不包含密钥或上游原始响应。

## 宝塔更新后端

1. 在宝塔终端停止服务并备份现有程序：

```sh
systemctl stop komari-availability
cp /www/wwwroot/komari/backend/komari /www/wwwroot/komari/backend/komari.before-astrbot-fix
```

2. 上传 `komari-astrbot-backend-update.zip` 到 `/www/wwwroot/komari/`，解压覆盖 `backend/komari`。包内仅包含后端程序和本说明，-fix
```

4. 等待下一次检查。选择 API Key、当前 v1 API，并给密钥授予 `bot` 权限。新版前端包也修正了权限提示；仅更新后端即可应用探测修复。

## 网络与 TLS

网络连接错误和权限错误是不同阶段的问题。必须先能连接目标 API，才可能进行密钥验证。

在运行 Komari 的服务器执行（不携带密钥）：

```sh
curl -i --max-time 15 https://你的AstrBot域名/api/v1/bots
```

如果得到 HTTP 401/403，表示网络和 TLS 连接已经成功，之后再核对认证。`curl: (60)` 表示证书信任链问题，应在宝塔 SSL 配置中使用完整证书链；不能通过更换 API Key 解决。

HTTPS 域名通常通过 443 端口反向代理到 Docker 的 HTTP 6185 端口。不要默认将 `:6185` 加到 HTTPS 域名后，除非该端口本身配置了 HTTPS。

若 AstrBot 与原生部署的 Komari 确实在同一台机器，且 Docker 已把 6185 映射到宿主机，可填写 `http://127.0.0.1:6185`。如果 Komari 本身在 Docker 中，127.0.0.1 则指向 Komari 容器自身，不能这样使用。
