# 项目状态

- 已完成：获取 urlDB v1.5.0 源码快照；完成代码/部署梳理；Docker Desktop 已启动；官方 Compose 已成功运行，首页、登录页、版本 API 和管理员登录已验证；已建立二开文档。
- 正在开发：P0 安全与合规基础设施。
- 下一步：落实 JWT 密钥环境变量化，并设计 Cookie/Token 的加密、轮换和审计方案。
- 当前问题：宿主机未安装 Go，不能原生启动后端；已使用官方 `golang:1.24.5-alpine` 容器完成 `gofmt` 及 `go test -vet=off ./task ./services ./handlers`。默认 `go test` 的 Go 1.24 vet 会因既有 Meilisearch/Telegram 动态日志格式串告警失败，需在后续专项修复；镜像内 migration 对 `system_health`（单数）建索引，但 GORM 实际表名是 `system_healths`，PostgreSQL 日志有该错误，服务仍可运行。
- 已知 Bug/风险：JWT 密钥硬编码在 `middleware/auth.go`；Compose 未包含 Meilisearch/Redis；账号凭据以 Cookie/Extra 保存，需要加密与权限审计；默认后台账户为 `admin` / `password`，首次部署必须立刻改密。
- 本地启动：`docker compose up -d`，入口预期为 `http://localhost:3030`；PostgreSQL 暴露 `localhost:5431`。原生前端：`cd web; pnpm dev`（后端需 Go + PostgreSQL）。
- 重要配置：数据库和迁移在 `env.example`；网盘账号在后台“账号管理”（表 `cks`）；Meilisearch 在后台系统配置。
- 最近一次开发：2026-08-15，完成资源详情页的自有分享安全跳转与授权转存任务状态轮询，并完成容器化 Go 工具链验证。

## Phase 1（2026-08-15）

- 状态：已完成“授权资源 + 自有分享链接”的基础数据模型和幂等转存请求 API。
- 下一步：继续 Phase 2，扩展后台管理界面。
- 执行：同平台、授权仍有效且 provider 已实现的请求会由任务处理器异步执行；跨平台和未实现 provider 不会降级或绕过限制，而是保留明确失败原因，供管理员修正后重试。

## Phase 2（2026-08-15，已完成）

- 已完成：注册 `authorized_transfer` 任务处理器；创建请求后自动启动；将分享 URL/FID 写入 `owned_shares` 并将可观测结果写入 `task_items.output_data`；失败任务可通过管理员重试接口重置并重新执行，保持原幂等键。
- 已完成：自有分享链接的 PanCheck 检测、检测元数据记录和明确失效结果的状态回写；管理员可按资源触发检测。
- 已完成：新增“授权分享管理”后台页，可按资源登记/查看授权证据、选择同平台有效账号并显示健康/容量、创建或重试授权转存任务、查看与检测自有分享链接。
- 已完成：资源详情页仅返回活跃自有分享链接；没有可用链接时会展示管理员创建的授权转存任务状态并轮询，任务成功后才允许跳转。原始资源 URL 和旧 `save_url` 不再由取链接口返回。
- 已完成：复核既有搜索、点击和渠道归因统计实现（网页/公众号/Telegram）；为公开搜索、详情、点击、取链和统计上报增加按客户端 IP、端点作用域隔离的固定窗口速率限制。详情页安全跳转改动已在 Go 1.24.5 容器中完成格式化和目标包测试。
- 状态：已完成。

## P0（2026-08-15，Provider 合规闸门）

- 状态：已完成本轮能力审查与默认拒绝控制。
- 已完成：将现有 provider 实现与官方/获准接口证据逐项比对；当前夸克、阿里、百度、UC、迅雷实现均不应因可运行而自动视作获准。百度公开资料表明存在网盘/PCS OAuth 开放能力，但当前实现未接入该授权路径。
- 已完成：新增部署级 provider 合规闸门；未显式列入批准清单、未记录批准引用或未设置正向最大分享保留期的 provider，均不能创建或执行授权转存任务。创建和执行均复核，避免已排队任务绕过后来撤销的批准。
- 证据与操作说明：`docs/provider-capability-matrix.md`；部署变量见 `env.example`。当前默认禁用全部自动授权转存，待取得官方 API/书面获准集成资格并接入相应接口后再逐项启用。
- 验证：Go 1.24.5 容器内 `gofmt` 与 `go test -vet=off ./common ./task ./services` 通过；`git diff --check` 通过。
