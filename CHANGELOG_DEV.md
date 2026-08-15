# 开发变更记录

## 2026-08-15 (P3 queue migration gate recheck)

- Files: `PROJECT_STATUS.md`, `CHANGELOG_DEV.md`
- Rechecked the live single-backend Compose deployment and PostgreSQL task
  tables: both `tasks` and `task_items` remain empty, so there is still no
  workload evidence to justify a durable queue/worker migration.
- Confirmed the source measurement endpoint exists, while the running fixed
  `ctwj/urldb-backend:1.5.0` image does not yet include it and returns 404;
  capturing the HTTP snapshot therefore requires a rebuilt/deployed image.
- No synthetic transfer task was created and no execution architecture was
  changed. The migration TODO remains open until real authorized task traffic
  or a second-replica requirement supplies the missing gate evidence.
- Verification: targeted task files are gofmt-clean; the Go 1.24.5 container
  test command, `docker compose config --quiet`, and `git diff --check`
  completed successfully.

## 2026-08-15 (P3 task operations measurement surface)

- Files: `task/operations.go`, `task/task_processor.go`, `db/repo/task_repository.go`, `handlers/task_handler.go`, `main.go`, `docs/task-operations.md`, `PROJECT_STATUS.md`, `TODO.md`
- Added the administrator-only `GET /api/tasks/operations/status` aggregate
  snapshot for the seven-day queue/worker migration baseline. It reports task
  volume, status distribution, pending/processing backlog, oldest backlog age,
  task duration aggregates, current process-local concurrency, and restart
  recovery count.
- Kept the response free of task payloads, URLs, credentials, authorization
  evidence, client IPs, and user content. No task execution or retry behavior
  changed.
- Verification: Go 1.24.5 container `gofmt` and
  `go test -vet=off ./...` passed; `docker compose config --quiet` and
  `git diff --check` also passed.

## 2026-08-15 (P3 task execution migration evaluation)

- Files: `docs/task-operations.md`, `PROJECT_STATUS.md`, `TODO.md`, `CHANGELOG_DEV.md`
- Evaluated the conditional migration from the process-local `TaskManager` path
  to an explicit queue/worker using the current code and local Compose data.
  The single-backend deployment has no task or task-item rows, so the migration
  is deferred until workload or replica requirements justify it.
- Documented the current single-process boundary, the horizontal-scaling risk,
  the required baseline, and the durable-claim/lease requirements for a future
  migration.
- Verification: local PostgreSQL aggregate inspection completed; no source code
  behavior was changed.

## 2026-08-15 (P3 provider contracts and compliance operations dashboard)

- Files: `common/provider_contract_test.go`, `services/compliance_service.go`, `services/compliance_service_test.go`, `handlers/compliance_handler.go`, `main.go`, `web/composables/useApi.ts`, `web/pages/admin/compliance.vue`, `web/config/adminNewNavigation.ts`, `web/composables/useAdminNav.ts`, `web/components/Admin/NewSidebar.vue`, `docs/compliance-operations.md`, `PROJECT_STATUS.md`, `TODO.md`
- Added no-network provider contract coverage for the registered provider factory entries and explicit tests that unimplemented provider types remain unavailable.
- Added an administrator-only compliance dashboard endpoint and page with aggregate authorization/share/task expiry and failure metrics, plus provider implementation and deployment-approval status. The response intentionally excludes URLs, authorization evidence contents, credentials, and client data.
- Verification: `gofmt`, `go test -vet=off ./common ./services ./handlers`, and `git diff --check` passed. Frontend build/type checks remain limited by the existing pnpm ignored-build policy and pre-existing workspace type errors.

## 2026-08-15 (P2 Meilisearch operations and fallback monitoring)

- Files: `docker-compose.yml`, `env.example`, `services/meilisearch_manager.go`, `handlers/public_api_handler.go`, `handlers/resource_handler.go`, `PROJECT_STATUS.md`, `TODO.md`
- Added the optional `search` Compose profile for a persistent, health-checked Meilisearch service. It is internal-only and requires an injected master key; PostgreSQL remains the default and does not require the profile.
- Meilisearch status now exposes index backlog plus in-process fallback telemetry. On a Meilisearch query failure/unavailable service, the request is observed and served through PostgreSQL without recording user queries, IP addresses, or secrets in the telemetry.
- Enabled Meilisearch starts the existing resumable reconciliation for unsynced resources after the first health check, preserving the manual control and retry endpoints.

## 2026-08-15 (P0 credential repository encryption)

- Files: `db/credential/`, `db/credential_migration.go`, `db/entity/cks.go`, `db/entity/credential_audit.go`, `db/repo/cks_repository.go`, `db/connection.go`, `handlers/cks_handler.go`, `main.go`, `env.example`, `PROJECT_STATUS.md`, `TODO.md`
- Changed provider credential persistence to versioned AES-256-GCM envelopes with account-and-column authenticated data, and added a separate HMAC-SHA-256 lookup fingerprint for duplicate detection. Provider interfaces remain unchanged because only repository reads decrypt values into process memory.
- Added the append-only `credential_audits` table. It records only account/provider/action/outcome/actor/IP/request/key-version metadata for credential lifecycle events and decrypt failures; it never stores credential material.
- Added an explicit `MIGRATE=true` legacy-row conversion path, fail-closed reads for plaintext/corrupt envelopes, required encryption/fingerprint startup secrets, and administrator protection for the credential-list route.
- Verification: `gofmt`, `go test -vet=off ./db/... ./handlers ./common ./task ./services ./middleware`, and `git diff --check` passed in `golang:1.24.5-alpine`.

## 2026-08-15 (P0 credential security baseline)

- Files: `middleware/auth.go`, `middleware/auth_test.go`, `main.go`, `env.example`, `docs/credential-security.md`, `PROJECT_STATUS.md`, `TODO.md`
- Changed JWT signing from an embedded value to a required `JWT_SECRET` environment variable (at least 32 bytes). Startup fails closed if it is absent or too short.
- Added optional `JWT_PREVIOUS_SECRET` verification-only support for a bounded, documented key-rotation window; new tokens always use the current key and parsing only accepts HS256.
- Defined the follow-up encryption boundary for `cks.ck` and `cks.extra`: versioned AES-256-GCM envelopes, KMS-held wrapping key, HMAC fingerprint for lookup, provider-specific rotation, and append-only no-secret audit events.
- Verification: `gofmt` and `go test -vet=off ./middleware` passed in `golang:1.24.5-alpine`; `git diff --check` passed.

## 2026-08-15（P0，Provider 合规闸门）

- 文件：`common/authorized_transfer_policy.go`、`common/authorized_transfer_policy_test.go`、`services/authorized_share_service.go`、`task/authorized_transfer_processor.go`、`env.example`、`docs/provider-capability-matrix.md`、`PROJECT_STATUS.md`、`TODO.md`
- 内容：建立默认拒绝的授权转存 provider 审批策略。provider 必须同时位于部署批准清单、提供官方 API 或书面获准集成的证据引用，并声明正向最大分享保留天数，才能创建或执行授权转存任务；任务创建和执行均复核，避免过往队列任务绕过政策变化。
- 合规：能力矩阵记录本轮可验证的百度网盘/PCS OAuth 公开资料以及所有现有实现的未获准状态。当前代码未走百度 OAuth，其他实现也未保留官方/获准接口证据，故不默认启用任何 provider；资源级授权及其保留期仍为必需条件。
- 验证：已用 Go 1.24.5 容器执行 `gofmt` 与 `go test -vet=off ./common ./task ./services`，均通过；`git diff --check` 通过。

## 2026-08-15（Phase 2，公开访问统计与速率限制）

- 文件：`middleware/rate_limit.go`、`middleware/rate_limit_test.go`、`main.go`、`PROJECT_STATUS.md`、`TODO.md`、`CHANGELOG_DEV.md`
- 内容：复核现有搜索、点击和网页/公众号/Telegram 渠道归因链路；为资源搜索、详情、点击、取链和搜索统计上报增加按客户端 IP 与端点作用域隔离的进程内固定窗口限速（详情/搜索 60 次/分钟，点击/取链 20 次/分钟，统计上报 30 次/分钟）。触发时返回 HTTP 429、统一响应体和 `Retry-After`。
- 验证：官方 Go 1.24.5 容器内 `gofmt`、`go test -vet=off ./middleware ./task ./services ./handlers` 与 `go vet ./middleware` 均通过；新增中间件测试覆盖超限与 `Retry-After`。
- 运维说明：限速为单进程内存状态；多副本部署需在网关统一限速或替换为共享存储实现。

## 2026-08-15（Phase 2，工具链验证）

- 文件：`handlers/resource_handler.go`、`PROJECT_STATUS.md`、`CHANGELOG_DEV.md`
- 内容：使用官方 `golang:1.24.5-alpine` 容器对详情页安全跳转改动执行 `gofmt`，并通过 `go test -vet=off ./task ./services ./handlers`（三个包均通过）；`docker compose config --quiet` 与 `git diff --check` 也通过。
- 已知限制：宿主机与运行时后端镜像均不包含 Go。默认 `go test` 会在 Go 1.24 vet 阶段被既有的 Meilisearch/Telegram 动态日志格式串告警阻塞，该问题不属于本次详情页改动，后续应专项整改。

## 2026-08-15（Phase 2，详情页安全跳转）

- 文件：`handlers/resource_handler.go`、`web/pages/r/[key].vue`、`web/components/QrCodeModal.vue`、`PROJECT_STATUS.md`、`TODO.md`
- 内容：详情页数据和取链接口改为不返回原始资源 URL 或旧 `save_url`；取链接口只返回同平台、未过期的活跃 `owned_shares`。当管理员已创建授权转存任务但链接尚未就绪时，接口只返回无敏感数据的任务状态，详情页按建议间隔轮询，成功后才展示并跳转自有分享链接。
- 合规与兼容性：保留原有路由和浏览统计；未创建合规自有分享链接的资源明确显示不可用，不触发自动转存、不暴露来源链接。
- 验证：待执行 Go 格式化/测试与前端类型检查；本机 Go 工具链和完整前端依赖链接仍不可用。

## 2026-08-15（Phase 2，授权分享后台管理）

- 文件：`services/authorized_share_service.go`、`handlers/authorized_share_handler.go`、`main.go`、`web/composables/useApi.ts`、`web/pages/admin/authorized-shares.vue`、`web/composables/useAdminNav.ts`、`web/components/Admin/NewSidebar.vue`、`web/config/adminNewNavigation.ts`、`PROJECT_STATUS.md`、`TODO.md`
- 内容：增加管理员读取单个资源授权记录的 API，并新增“授权分享管理”后台页。该页按资源管理授权状态和证据引用，展示目标账号有效性与容量，创建或重试同平台授权转存任务，并查看、手动检测自有分享链接。新增入口沿用现有后台导航和 API 鉴权。
- 合规与兼容性：界面仅供管理员操作，不展示原始资源 URL；创建任务仍由后端复核授权、保留期限、同平台约束和账号有效性，不绕过 provider 限制。
- 验证：待执行前端类型检查和 Go 格式化/测试；本机仍无 Go 工具链。

## 2026-08-15（Phase 2，自有分享失效检测）

- 文件：`db/entity/owned_share.go`、`services/authorized_share_service.go`、`handlers/authorized_share_handler.go`、`task/authorized_transfer_processor.go`、`main.go`、`PROJECT_STATUS.md`、`TODO.md`
- 内容：为 `owned_shares` 增加最近检测时间、结果、检测方式和失败原因；复用既有 `LinkCheckService`/PanCheck 批量检测自有分享 URL。只有明确的 `invalid` 结果会把活跃分享状态回写为 `invalid`；检测关闭、超时或未定结果仅记录检测信息，不撤销链接。新增管理员接口 `POST /api/resources/:id/owned-shares/check?ignore_cache=true`；后续成功转存会清除旧检测元数据并恢复为 `active`。
- 合规与兼容性：不访问或公开原始资源 URL，不绕过 provider 限制；保留 `resources.save_url` 和既有资源有效性逻辑。
- 验证：`docker compose config --quiet` 与 `git diff --check` 通过；本机仍无 Go 工具链，待执行 `gofmt` 和 `go test ./task ./services ./handlers`。

## 2026-08-15（Phase 2，授权转存执行）

- 文件：`task/authorized_transfer_processor.go`、`services/authorized_share_service.go`、`handlers/authorized_share_handler.go`、`main.go`、`PROJECT_STATUS.md`、`TODO.md`。
- 内容：新增 `authorized_transfer` 任务处理器并注册到现有 `TaskManager`；新建授权转存请求后异步启动任务。处理器会在每次执行时复核授权、保留期限、资源/账号同平台约束与账号有效性，调用已有 provider 的正常转存接口，并幂等写入 `owned_shares`。任务项输出记录分享 URL、FID、复用标记与现有处理时长。新增管理员失败重试 API：`POST /api/resources/:id/owned-shares/tasks/:taskID/retry`。
- 合规与兼容性：不支持跨平台复制，不绕过验证、权限、风控或 provider 限制；保留既有资源转存链路和 `resources.save_url`。失败任务仅在管理员显式重试时重置，仍复用原始唯一幂等键。
- 验证：`docker compose config --quiet` 通过；本机未安装 `go`/`gofmt`。尝试构建 Docker 后端编译阶段时，Docker Hub 的 `golang:1.24.5-alpine` 匿名令牌请求网络超时，故无法执行格式化或 Go 测试；待具备 Go 1.24 工具链或可用构建镜像后执行 `gofmt` 与 `go test ./task ./services ./handlers`。

## 2026-08-15（Phase 1）

- 文件：`db/entity/resource_authorization.go`、`db/entity/owned_share.go`、`db/entity/task.go`、`db/connection.go`、`services/authorized_share_service.go`、`handlers/authorized_share_handler.go`、`main.go`、`PROJECT_STATUS.md`、`TODO.md`。
- 内容：新增资源授权、自有分享及数据库唯一幂等键；增加管理员授权登记、自有分享查询和转存请求 API。请求必须有有效授权，目标账号必须属于目标网盘；已有有效自有分享直接复用。
- 兼容性影响：保留既有 `resources.save_url` 和转存链路。新增表通过 `MIGRATE=true` 的 GORM AutoMigrate 创建；新任务仅入队，执行器留待下一阶段。
- 验证：`docker compose config --quiet` 通过。后端容器编译因 Docker Hub 网络连接失败受阻；本机未安装 Go，尚未能运行 Go 测试。

## 2026-08-15

- 文件：`AGENTS.md`、`PROJECT_STATUS.md`、`ARCHITECTURE.md`、`TODO.md`、`CHANGELOG_DEV.md`
- 内容：建立首轮架构、状态、待办与协作说明。
- 原因：为后续小步、可运行的二次开发建立持久上下文。
- 兼容性影响：无业务代码或数据库变更。

## 2026-08-15（运行验证）

- 文件：无源码改动；运行时创建 Docker named volume `pansearch_postgres_data`。
- 内容：启动官方 Compose，验证 PostgreSQL 健康、后端版本 API、前端首页/登录页。
- 原因：完成本地可运行性检查。
- 兼容性影响：无；首次运行写入默认数据库数据。发现 `system_health` 索引名与 `system_healths` 表名不一致的非致命 migration 日志错误，待后续小修。
