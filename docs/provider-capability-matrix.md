# Provider transfer capability matrix

Last reviewed: 2026-08-15

This matrix controls automated **authorized transfer + creation of a service-owned share**. It does not govern passive indexing, link checking, or a user manually opening a link.

| Provider | Current code path | Official/approved interface verified for this deployment | Automated authorized transfer status | Evidence / retention needed before enabling |
| --- | --- | --- | --- | --- |
| Baidu Netdisk | `common/baidu_pan.go`, Cookie/BDSToken web endpoint client | Baidu’s developer site describes the Netdisk/PCS open platform and OAuth-granted personal storage access, including file storage and sharing. The current client does **not** use that OAuth integration. | Disabled by default | An approved OAuth application or written provider integration approval; record its URL/contract ID and maximum allowed public-share retention. |
| Quark Netdisk | `common/quark_pan.go`, Cookie web endpoint client | Not verified in this review. | Disabled by default | Official API documentation or written provider approval; retention limit. |
| Aliyun Drive | `common/alipan.go`, refresh-token/web client | Not verified in this review. | Disabled by default | Official API documentation or written provider approval; retention limit. |
| UC Netdisk | `common/uc_pan.go`, Cookie web endpoint client | Not verified in this review. | Disabled by default | Official API documentation or written provider approval; retention limit. |
| Xunlei Netdisk | `common/xunlei_pan.go`, token/web endpoint client | Not verified in this review. | Disabled by default | Official API documentation or written provider approval; retention limit. |
| Tianyi, 123pan, 115 | URL recognition only; no enabled factory implementation | Not verified, and no transfer implementation is enabled. | Disabled | A provider approval plus a compliant implementation and retention limit. |

## Sources checked

- Baidu’s [developer services page](https://openapi.baidu.com/light) lists the Baidu Netdisk open platform for backup, sharing, and synchronization tools.
- Baidu’s [PCS service page](https://openapi.baidu.com/ms/pcs) describes personal cloud storage and file sharing.
- Baidu’s [OAuth authorization page](https://openapi.baidu.com/oauth/2.0/authorize?client_id=q8WE4EpCsau1oS0MplgMKNBn&redirect_uri=oob&response_type=code&scope=basic+netdisk) shows Netdisk access is granted through an authorization scope.

Absence from this table is not a finding that a provider has no program. It means this deployment has not retained enough official documentation or provider approval to automate transfers safely.

## Activation procedure

1. Obtain and retain the provider’s official API terms or written integration approval.
2. Confirm the code path uses that approved interface; do not enable a Cookie or reverse-engineered path merely because a public API exists.
3. Set a positive maximum share retention period and add the provider’s identifier to `AUTHORIZED_TRANSFER_APPROVED_PROVIDERS`.
4. Configure `AUTHORIZED_TRANSFER_PROVIDER_<NAME>_APPROVAL_REF` and `AUTHORIZED_TRANSFER_PROVIDER_<NAME>_MAX_SHARE_RETENTION_DAYS` in deployment secrets/configuration.
5. For each resource, retain resource-specific authorization evidence and a `retention_until` value. The shorter applicable retention period must win operationally.

The application fails closed when any of these deployment declarations are absent. The guard is checked both when a task is created and again immediately before execution, so a previously queued task cannot bypass a withdrawn approval.
