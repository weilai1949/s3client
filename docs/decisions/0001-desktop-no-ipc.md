# ADR-001：桌面端采用 B/S 架构、不使用 Tauri IPC

## Status

Accepted

## Date

2026-09-16（回溯记录；决策自项目初版沿用）

## Context

s3clinet 提供 Web 端与 Tauri 2 桌面端。桌面端有两条技术路线：

1. **传统 Tauri**：Rust 侧定义 command + IPC（invoke），前端通过 `@tauri-apps/api` 调用本地能力。
2. **B/S 无 IPC**：桌面壳仅加载前端，前端与本地 Go 后端（或任意远程后端）全走 HTTP。

选择考量：功能复用（Web 与桌面共用同一前端与后端）、攻击面、维护成本。

## Decision

桌面端采用 **B/S 架构，不使用 Tauri IPC**：无 command、无插件调用，`withGlobalTauri` 关闭，
capabilities 为空权限集，前端通过 HTTP 访问后端（本地 `http://127.0.0.1:8080` 或用户在「服务器」设置中配置的远程后端）。

## Alternatives Considered

### 传统 Tauri IPC（Rust command + invoke）
- Pros：可访问系统能力（文件系统、托盘等）。
- Cons：Web 与桌面功能分叉（前端需维护 IPC 分支）；Rust 侧新增攻击面与审计负担。
- Rejected：s3clinet 的功能 100% 是 S3 操作，全部由 HTTP API 承载，无需系统级能力。

## Consequences

- Web 端与桌面端复用同一套前后端，功能一致性有保障。
- 桌面端攻击面最小：无 IPC 通道、无命令注入面（安全审计边界 E 评为「已缓解」）。
- 代价：桌面端无法使用需系统权限的特性（如系统托盘原生通知），当前无此需求。
