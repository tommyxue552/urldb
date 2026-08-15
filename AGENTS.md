# urlDB 二次开发协作说明

## 项目目标

在仅处理自有、获授权或允许公开分享资源的前提下，基于 urlDB 构建网盘资源搜索与分发站：命中已有自有分享链接时直接返回；未命中时创建合规转存任务，成功后创建并保存自有分享链接。

## 技术栈与目录

- 后端：Go、Gin、GORM、PostgreSQL（`main.go`、`handlers/`、`services/`、`db/`）。
- 前端：Nuxt 3、Vue 3、TypeScript、Pinia、Naive UI（`web/`）。
- 搜索：PostgreSQL 回退 + 可选 Meilisearch（`services/meilisearch_*`）。
- 网盘实现：`common/`；任务：`task/`；调度：`scheduler/`；部署：`docker-compose.yml`、`Dockerfile`、`nginx/`。

## 开发规则

1. 修改前先读 `PROJECT_STATUS.md`；每完成阶段更新它。
2. 每项重要修改都追加 `CHANGELOG_DEV.md`。
3. 未经明确要求，不删除既有功能、不重写核心架构、不替换成熟 provider 架构。
4. 每次修改保持项目可启动，并进行与风险相称的验证。
5. 优先复用 `PanFactory`、`BasePanService`、现有任务和资源保存链路；只使用官方 API 或稳定、合规接口。
6. 不实现绕过风控、验证码、权限、会员限制、版权保护的能力；不得录入或分发未获授权资源。
7. 不提交真实 Cookie、Token、密码或网盘账号；配置位置见 `env.example` 和后台系统配置。
