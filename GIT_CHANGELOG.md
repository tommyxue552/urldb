# Git 更新日志

## 2026-08-15 — 初始导入与授权分享链路（待提交）

### 基线

- 导入 urlDB v1.5.0 源码快照及现有前后端、任务、Provider、搜索和部署结构。
- 建立项目架构、开发状态、待办事项与协作说明文档。
- 当前仓库尚无历史提交，本条记录对应首个 Git 工作区快照。

### 新增

- 新增资源授权记录 `resource_authorizations`，保存授权状态、证据类型、证据引用、保留期限与核验信息。
- 新增自有分享记录 `owned_shares`，按“资源 + 目标网盘 + 账号”建立唯一约束。
- 新增授权登记、自有有效分享查询和幂等转存任务 API。
- 新增 `authorized_transfer` 任务类型与执行处理器，复用现有 `TaskManager`、`PanFactory` 和 Provider 转存接口。
- 新增失败任务管理员重试接口：`POST /api/resources/:id/owned-shares/tasks/:taskID/retry`。

### 行为变更

- 创建授权转存任务前会校验资源授权、授权保留期限、目标账号有效性及账号与网盘归属。
- 已存在有效自有分享时直接复用，不重复创建任务。
- 新建任务使用稳定幂等键，避免同一资源、网盘和账号产生重复任务。
- 任务创建后异步执行；执行前再次校验授权与账号状态。
- 成功结果幂等写入 `owned_shares`，并在 `task_items.output_data` 中记录分享记录 ID、分享 URL、FID 和复用标记。
- 仅支持资源来源与目标账号属于同一网盘平台的转存；跨平台请求会失败并保留可观察错误信息。

### 兼容性与合规

- 保留原有 `resources.save_url`、批量转存和自动转存链路，不替换既有 Provider 架构。
- 不实现验证码、权限、风控、会员限制或版权保护绕过。
- 不包含真实 Cookie、Token、密码或网盘账号配置。

### 验证状态

- `docker compose config --quiet` 已通过。
- urlDB v1.5.0 官方镜像曾验证 PostgreSQL、后端版本 API、前端首页、登录页和管理员登录可运行。
- 当前源码尚未完成 Go 编译和测试：本机缺少 Go 工具链，Docker Hub 拉取 `golang:1.24.5-alpine` 时网络超时。
- 当前运行的 v1.5.0 后端镜像不包含本次二次开发源码变更。

### 已知问题

- migration 使用 `system_health` 单数表名创建索引，而 GORM 实际表名为 `system_healths`。
- JWT 密钥仍硬编码在 `middleware/auth.go`。
- 账号 Cookie/Token 尚未实现加密、轮换与访问审计。
- Compose 尚未包含可选的 Meilisearch/Redis 服务。
- 默认管理员账号必须在首次部署后立即修改密码。

### 建议首个提交信息

```text
feat: initialize PanSearch authorized sharing workflow

- import urlDB v1.5.0 project baseline
- add resource authorization and owned-share models
- add idempotent authorized transfer tasks
- add same-provider execution and admin retry flow
- add architecture, status, todo, and development documentation
```
