# 项目状态

- 已完成：获取 urlDB v1.5.0 源码快照；完成代码/部署梳理；Docker Desktop 已启动；官方 Compose 已成功运行，首页、登录页、版本 API 和管理员登录已验证；已建立二开文档。
- 正在开发：Phase 2（授权转存执行处理）。
- 下一步：实现分享失效检测与状态回写，再扩展后台管理与统计。
- 当前问题：本机未安装 Go，不能原生启动后端；镜像内 migration 对 `system_health`（单数）建索引，但 GORM 实际表名是 `system_healths`，PostgreSQL 日志有该错误，服务仍可运行。
- 已知 Bug/风险：JWT 密钥硬编码在 `middleware/auth.go`；Compose 未包含 Meilisearch/Redis；账号凭据以 Cookie/Extra 保存，需要加密与权限审计；默认后台账户为 `admin` / `password`，首次部署必须立刻改密。
- 本地启动：`docker compose up -d`，入口预期为 `http://localhost:3030`；PostgreSQL 暴露 `localhost:5431`。原生前端：`cd web; pnpm dev`（后端需 Go + PostgreSQL）。
- 重要配置：数据库和迁移在 `env.example`；网盘账号在后台“账号管理”（表 `cks`）；Meilisearch 在后台系统配置。
- 最近一次开发：2026-08-15，完成授权转存执行器、任务启动和失败重试接口；尚待 Go 工具链验证。

## Phase 1（2026-08-15）

- 状态：已完成“授权资源 + 自有分享链接”的基础数据模型和幂等转存请求 API。
- 下一步：继续 Phase 2，完成分享失效检测与状态回写。
- 执行：同平台、授权仍有效且 provider 已实现的请求会由任务处理器异步执行；跨平台和未实现 provider 不会降级或绕过限制，而是保留明确失败原因，供管理员修正后重试。

## Phase 2（2026-08-15，进行中）

- 已完成：注册 `authorized_transfer` 任务处理器；创建请求后自动启动；将分享 URL/FID 写入 `owned_shares` 并将可观测结果写入 `task_items.output_data`；失败任务可通过管理员重试接口重置并重新执行，保持原幂等键。
- 待完成：分享失效检测及状态回写；后台管理界面。
