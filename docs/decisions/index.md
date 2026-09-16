# 架构决策记录（ADR）

> 本目录记录 s3clinet 的关键架构决策（Architecture Decision Records），说明「为什么这么做」以及
> 考虑过的替代方案。格式遵循轻量 ADR 模板（状态 / 日期 / 背景 / 决策 / 替代方案 / 后果）。
>
> 决策索引：

| # | 决策 | 状态 |
|---|---|---|
| [ADR-001](0001-desktop-no-ipc.md) | 桌面端采用 B/S 架构、不使用 Tauri IPC | Accepted |
| [ADR-002](0002-store-fail-closed.md) | 存储不可用时硬失败而非降级只读 | Accepted |
| [ADR-003](0003-ssrf-private-allow.md) | SSRF 防护放行私网/回环地址 | Accepted |
| [ADR-004](0004-minimal-frontend-deps.md) | 前端生产依赖仅保留 vue | Accepted |
