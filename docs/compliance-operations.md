# 合规审计与运营看板

## 管理员 API

`GET /api/compliance/dashboard` 需要管理员 JWT。响应只包含聚合计数和 provider 部署审批元数据，不返回原始资源 URL、自有分享 URL、授权证据内容或账号凭据。

报告包括：

- 授权记录的总数、当前有效数、已过期数和未来 7 天到期数；
- 自有分享记录的总数、当前有效数、失效数、过期数和未来 7 天到期数；
- `authorized_transfer` 任务的总数，以及 pending/running/completed/failed 状态计数；
- 每个已登记 provider 是否存在实现、是否满足 `PanService` 转存契约、是否配置了部署审批引用和正向最大分享保留期；
- 被合规闸门阻断的 provider 数量。

Provider 状态检查只创建本地服务对象，不发起外部请求，也不读取或验证账号凭据。授权转存仍由创建任务和任务执行阶段的现有合规闸门再次复核。

## Provider 契约测试

`common/provider_contract_test.go` 对当前工厂已注册的 Quark、Aliyun、Baidu、UC 和 Xunlei provider 验证：URL 识别、工厂构造、`GetServiceType` 一致性以及 `PanService` 接口可用；同时固化 Tianyi、123pan 和 115 尚未注册时必须失败。测试使用占位配置，不连接任何网盘或提交真实凭据。
