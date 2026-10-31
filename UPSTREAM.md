# 上游来源与许可证核对

核对日期：2026-10-31。此文件用于记录来源，不是对上游代码重新授权，也不表示获得了作者的额外许可。

本工作区是基于上游源码的功能扩展，不是 Komari 或 Lumina 官方发行版。

| 组件 | 上游仓库 | 本地基线 | 授权核对 |
| --- | --- | --- | --- |
| Komari 后端 | https://github.com/komari-monitor/komari | 96cf0f79d8cfdf6cc8be54fd1976bbc026d58f7f | MIT；保留 komari/LICENSE、komari/NOTICE 及原有版权声明 |
| Komari 默认前端 | https://github.com/komari-monitor/komari-web | 797ca79daee6eac59164013e18bf5d098df9238c | 本地及当前上游根目录未发现许可证文件；GitHub API 的 license 字段为空，完整再分发授权待确认 |
| Lumina 主题 | https://github.com/stqfdyr/komari-theme-Lumina | main 分支源码快照，主题版本 1.1.6；未记录快照提交号 | 本地及当前上游根目录未发现许可证文件；GitHub API 的 license 字段为空，完整再分发授权待确认 |

GitHub API 未识别到许可证本身不能证明不存在其他授权，仍需核对作者的明确声明或书面许可。README 的二次开发说明也不应擅自替换成 MIT 或其他完整许可证。

## 本次修改

- 网站 HTTP(S) 可用性监测、AstrBot API 可用性监测、调度、历史记录和权限隔离。
- Linux API-only 后端、独立静态前端、Docker Compose 和宝塔部署配置。
- 默认前端监测管理与公开展示、Lumina 监测卡片、主页和原控制台的统一入口。

AstrBot 仓库 https://github.com/AstrBotDevs/AstrBot 用于核对接口；astrbot-reference 不是发布源码的一部分。本项目通过 HTTP API 与 AstrBot 通信，不分发该参考目录。

## 公开发布准备

- 保留各上游署名、仓库链接、许可证和 NOTICE；不要给整个混合项目直接套用 MIT。
- 两份前端的公开修改版源码及其预编译分发包，均需先确认相应再分发授权。注明来源不能替代许可。
- 如果暂时无法确认授权，可另外准备只含独立新增代码及接入说明的仓库；不把未确认授权的前端源码或编译包装入该仓库。
- 不上传数据库、真实密钥、登录令牌、运行日志、node_modules、工具链缓存及私人配置。

GitHub 关于许可证的说明：https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/licensing-a-repository

## 当前验证状态

已验证原控制台基本路由与真实后端登录、监测保存和公开展示。用户随后报告了 Primitive.div / Slot 组件错误，仍未完成定位和修复验证；不能把当前版本标记为已修复该问题的稳定发行版。
