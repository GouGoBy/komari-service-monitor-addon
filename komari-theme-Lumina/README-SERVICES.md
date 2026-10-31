# Lumina 主页 + 完整 Komari 控制台（services.3）

基于 [stqfdyr/komari-theme-Lumina](https://github.com/stqfdyr/komari-theme-Lumina) 1.1.6 和本次定制的 Komari 默认前端。保留原作者署名。

## 页面与功能

- `/`：Lumina 主页，展示服务器以及网站、AstrBot 的公开状态，保留深浅色、可用率、耗时、检查记录和 15 秒刷新。
- `/admin/login`：Komari 原登录页面，沿用现有账户、密码及会话。
- `/admin/dashboard`：Komari 原仪表盘，保留完整侧栏。
- `/admin/servers`：原服务器管理。
- `/admin/service-monitor`：网站与 AstrBot 监测管理，新增、编辑、暂停、删除、私有任务和密钥保留等功能不变。
- 原账户、通知、站点设置、主题、插件、工作台等页面均保留。
- Lumina 主页有明确的“控制台”按钮；“管理监测”进入原控制台。原先 `/?view=service-manage` 也转入控制台。
- 控制台“外观与插件”分组内提供“Lumina 主题设置”，打开主题自己的设置界面。主题设置不替代控制台。

需要本次定制后端提供服务监测 API。AstrBot 开发者 API Key 需授予 `bot` 权限。

## 宝塔升级

使用 **komari-lumina-full-console.zip**。这是一份完整静态前端，包含 Lumina 和 Komari 控制台两套应用的全部编译资源，无后端程序和数据库。

1. 备份 `/www/wwwroot/komari/frontend` 和网站 Nginx 配置。
2. 将旧 `frontend` 目录改名备份，把新包解压到 `/www/wwwroot/komari/`。不要仅替换 index.html 或上传标准主题包。确认文件为 `/www/wwwroot/komari/frontend/index.html`，避免多套一层目录。
3. 宝塔网站目录仍为 `/www/wwwroot/komari/frontend`。网站 `server` 内需要 `/api/` 代理到 `127.0.0.1:25774`，以及 `location / { try_files $uri $uri/ /index.html; }`。
4. 包内 `deploy/nginx-lumina-locations.conf` 提供完整 location 规则。替换已有同名规则，保留域名、SSL、日志等配置，通过检查后重新加载 Nginx。原版的 `/admin/` 回退到 `/control/index.html` 的规则也受支持，因为两个 HTML 入口现已完全相同。
5. 浏览器 Ctrl+F5 强制刷新。先访问 `/`，再点击“控制台”登录；也可以直接访问 `/admin/login`。如浏览器保留旧站点缓存，请用无痕窗口验证。

后端服务和数据库不需要改动，不需要重新初始化，也不需要在服务器编译。现有账户密码继续使用。

此版在启动任何 React 路由之前选择应用：`/admin`、`/terminal`、安装及恢复页面加载原 Komari 前端，其余页面加载 Lumina。控制台直接在当前地址启动，不经过上一版的 URL 中转。两套 CSS 按需加载，避免主页样式影响控制台。

`/control/index.html` 保留为旧配置的兼容入口，默认打开 `/admin/dashboard`。如果 Nginx 对 `/admin` 直接返回服务器错误或整站反代到 API-only 后端，仍需修正 Nginx 配置；静态前端无法覆盖服务器返回的错误。

宝塔静态部署下，后台安装/切换主题不会自动替换网站目录；更换静态主页仍需更新该目录。API-only 后端与原生 Komari 的主题分发方式不同。

回滚：恢复旧 `frontend` 目录和 Nginx 配置后重新加载。

## 标准主题部署

`Lumina-services-theme.zip` 用于非 API-only 的 Komari 原生主题分发部署；原生 Komari 负责提供默认控制台。单独上传主题无法给原版后端添加监测接口。你当前的宝塔静态部署应使用上述完整包。

## 源码

`komari-lumina-services-source.zip` 包含 Lumina、完整 komari-web、统一入口和组装脚本。

两个前端分别安装依赖并执行 `npm run build`（默认前端也可使用 pnpm）；**两者 base 均为 `/`**，无需 `VITE_CONTROL_ENTRY` 或 `/control/` 构建。然后在源码包根目录执行 `node toolchain/assemble-lumina.cjs <输出目录>`，生成包含两套应用的统一静态站点。组装脚本仅使用 Node 内置模块，要求 Node.js 22+。

统一入口源码是 `deploy/frontend-entry.js` 和 `deploy/frontend-index.html`；Lumina 新增监测样式位于 `src/components/services/`。服务监测后端沿用此前的修改，无需更新。
