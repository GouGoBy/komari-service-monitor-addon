# 宝塔部署：Linux amd64 预编译版

此部署包直接运行 Go 后端和前端静态文件，不需要 Docker、Go SDK、Node.js 或现场编译。适用于 `uname -m` 输出 `x86_64` 的 Linux 服务器，包含静态链接的 Linux amd64 后端。当前版本经过交叉编译、后端测试及前端浏览器验证，未在宝塔实机启动测试。

## 1. 上传和解压

宝塔软件商店安装 Nginx。在“文件”中创建 `/www/wwwroot/komari`，将整个宝塔部署包上传到这里并解压，使目录为：

```text
/www/wwwroot/komari/
  backend/komari
  frontend/index.html
  frontend/assets/
  frontend/themes/default/komari-theme.json
  deploy/komari-availability.service
  deploy/nginx-locations.conf
  README.md
```

请确认没有多套一层解压目录。`backend/` 不应设为网站根目录。

## 2. 启动后端

在宝塔终端中用 root 执行：

```sh
chmod +x /www/wwwroot/komari/backend/komari
chown -R www:www /www/wwwroot/komari/backend
cp /www/wwwroot/komari/deploy/komari-availability.service /etc/systemd/system/komari-availability.service
systemctl daemon-reload
systemctl enable --now komari-availability
systemctl status komari-availability --no-pager
```

出现 `active (running)` 表示后端已启动，服务由 systemd 管理，SSH 断开不影响运行，并配置开机启动和失败重启。宝塔默认建立 `www` 用户；若服务器修改过该用户，须同步修改服务文件中的 User/Group 和目录权限。

查看后端日志：

```sh
journalctl -u komari-availability -n 50 --no-pager
```

后端只监听 `127.0.0.1:25774`，供同机 Nginx 代理，不需要开放 25774 端口。

## 3. 在宝塔创建静态网站

进入“网站 → 添加站点”，填写域名，PHP 选择“纯静态”（或不启用 PHP），不创建数据库。站点根目录填写：

```text
/www/wwwroot/komari/frontend
```

进入该网站的“配置文件”，将 `deploy/nginx-locations.conf` 内容添加到现有 `server { ... }` 内。如果已经有 `location / { ... }`，替换该段，不能重复添加。也不要在宝塔反向代理菜单中再给整个站点添加根路径反代：网页由静态目录提供，仅 `/api/` 转发给后端。

保留宝塔生成的证书、日志、域名等配置。保存配置并确认 Nginx 校验通过。

## 4. 初始化

域名解析指向服务器，云安全组及宝塔防火墙开放网站所用的 80/443 端口。访问：

```text
http://你的域名/install
```

完成初始化后登录，在“监控与运维 → 服务可用性”添加监测。勾选“在公开状态页展示”并启用监测，即会在主页 `/` 的服务器列表下方显示；公开详情页 `/service-monitor` 仍保留。可以在宝塔该站点的 SSL 页面申请证书并启用 HTTPS。

此部署中 AstrBot 若运行在同一台主机，可以填 `http://127.0.0.1:6185`；网站和 AstrBot 的探测均由后端执行。

## 5. 数据与维护

数据位于 `/www/wwwroot/komari/backend/data/`。这是原生运行的独立数据目录，与之前 Docker 的 `komari-data` 卷不同。已有 Docker 数据需要另外迁移，不能期望自动沿用。

更新前备份数据，更新程序时停止服务、替换程序和前端、保持数据目录及权限，然后启动：

```sh
systemctl stop komari-availability
# 在宝塔文件管理中替换 backend/komari 和 frontend/，保留 backend/data/
chmod +x /www/wwwroot/komari/backend/komari
systemctl start komari-availability
```

如果后端无法启动，先看 `journalctl` 的错误；如果网页刷新后 404，检查 SPA 的 `try_files` 配置；如果 API 502，检查后端是否在运行。
