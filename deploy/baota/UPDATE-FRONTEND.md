# 主页同时展示服务器与服务状态：前端更新

适用于此前已部署本项目定制后端的站点。本次只更新前端，不需要替换后端程序、重启后端、迁移数据库或重新初始化。

## 宝塔更新

1. 在宝塔“文件”中先备份现有 `/www/wwwroot/komari/frontend/` 目录。
2. 上传 `komari-homepage-frontend-update.zip` 到 `/www/wwwroot/komari/` 并解压，允许覆盖同名的前端文件。
3. 确认新的 `index.html` 仍在 `/www/wwwroot/komari/frontend/index.html`，不要多套一层目录。
4. 浏览器访问主页 `/`，按 Ctrl+F5 强制刷新。如果站点经过 CDN 缓存，刷新首页缓存。
5. 在后台“服务可用性”中，将希望在主页显示的监测勾选“在公开状态页展示”，并启用该监测。

更新包只包含 `frontend/` 和本说明，不会写入 `backend/` 或数据库。首页在服务器列表下方新增服务状态区，显示网站和 AstrBot 的状态、24 小时可用率、响应耗时与历史记录，每 15 秒刷新。原管理页 `/admin/service-monitor` 和公开详情页 `/service-monitor` 保留。

主页使用公开 API，因此私有或暂停的监测不会出现在这里。服务器接口与服务接口分别加载，一个接口报错不会阻止另一个区域显示。

## Docker 更新

如果采用原来的 Docker Compose 部署，更新源码目录 `komari-web/` 后执行：

```sh
docker compose build frontend
docker compose up -d --no-deps frontend
```

Docker 前端文件在镜像内，不能只替换宿主机某个目录并期望容器自动更新。宝塔原生静态站点可直接使用本压缩包覆盖前端。
