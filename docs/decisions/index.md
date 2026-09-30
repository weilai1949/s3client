# 架构决策记录（ADR）

> 本目录记录 s3clinet 的关键架构决策（Architecture Decision Records），说明「为什么这么做」以及
> 考虑过的替代方案。
>
> **格式**：轻量 ADR，H2 章节固定为 `Status` / `Date` / `Context` / `Decision` /
> `Alternatives Considered` / `Consequences`（**英文 H2**，正文中文）；复制
> [`0000-template.md`](0000-template.md) 起新篇，并按下表登记。
>
> **Status 取值**：`Proposed`（已提出、未定）/ `Accepted`（已采纳并生效）/ `Deprecated`（不再推荐、
> 但无替代）/ `Superseded by ADR-NNN`（被新决策取代）。ADR **不归档、不删除**；决策不变而只是
> 补充事实 / 收紧配置时，在篇内**追加** `## Update（YYYY-MM-DD）` 一节（下表「最新更新」列登记），
> 不改写历史结论。

| # | 决策 | 状态 | 日期（决策） | 最新更新 |
|---|---|---|---|---|
| [ADR-001](0001-desktop-no-ipc.md) | 桌面端采用 B/S 架构、不使用 Tauri IPC | Accepted（已采纳） | 2026-09-16（回溯） | — |
| [ADR-002](0002-store-fail-closed.md) | 存储不可用时硬失败而非降级只读 | Accepted（已采纳） | 2026-09-16（回溯） | — |
| [ADR-003](0003-ssrf-private-allow.md) | SSRF 防护放行私网 / 回环地址（新增 `S3C_SSRF_DENY_PRIVATE` 可选加固） | Accepted（已采纳；默认策略未变） | 2026-09-16（回溯） | [2026-09-19](0003-ssrf-private-allow.md#update2026-09-19) |
| [ADR-004](0004-minimal-frontend-deps.md) | 前端生产依赖仅保留 vue | Accepted（已采纳；决策未变，仅更正事实陈述） | 2026-09-16（回溯） | [2026-09-19](0004-minimal-frontend-deps.md#update2026-09-19) |
| [ADR-005](0005-sse-async-jobs.md) | 异步任务与进度推送统一走服务端 JobRegistry + SSE，前端单一订阅实现 | Accepted（已采纳） | 2026-09-18（回溯） | — |
| [ADR-006](0006-store-drivers-atomic-write.md) | 账号存储三驱动（json / sqlite / encrypted）+ 原子写 + 单写者锁 | Accepted（已采纳） | 2026-09-18（回溯） | — |
| [ADR-007](0007-presign-direct-upload.md) | 浏览器预签名直传，对象字节不经过本服务 | Accepted（已采纳） | 2026-09-18（回溯） | — |
| [ADR-008](0008-frontend-zero-dep-stack.md) | 前端零运行时依赖下的自研路由 / 状态 / i18n / HTTP（ADR-004 姊妹篇） | Accepted（已采纳） | 2026-09-18（回溯） | — |
| [ADR-009](0009-bounded-concurrency.md) | 批量与流式请求全部有界并发 | Accepted（已采纳） | 2026-09-18（回溯） | — |
| [ADR-010](0010-zip-streaming.md) | ZIP 服务端流式打包（不落盘 + 客户端流式落盘 / blob 兜底） | Accepted（已采纳） | 2026-09-18（回溯） | — |
| [ADR-011](0011-single-instance-no-ha.md) | 单实例部署、无 HA 路径 | Accepted（已采纳） | 2026-09-19（回溯） | — |
| [ADR-012](0012-rest-no-version-prefix.md) | REST 契约无版本前缀，破坏性变更靠发布节奏缓冲 | Accepted（已采纳） | 2026-09-18（回溯） | — |

> 「日期（决策）」是决策成立的时间；`（回溯）` 表示该日期是事后补记，决策自项目初版沿用。
> 「最新更新」列指向篇内 `## Update（…）` 小节——本列的每一条都必须在对应 ADR 里真实存在
> （链接由 `apps/server/doc_link_gate_test.go` 校验，锚点必须真实）。
