# ADR-002：存储不可用时硬失败而非降级只读

## Status

Accepted

## Date

2026-09-16（回溯记录；决策自项目初版沿用）

## Context

s3clinet 的账号存储（json / sqlite / encrypted）是配置的单一事实来源。当存储不可用（磁盘故障、权限错误、加密密钥丢失）时，服务如何表现？

方案对比：

1. **降级只读**：继续提供对象浏览/下载等读操作，拒绝写操作。
2. **硬失败**：启动时存储打开失败即退出；运行中 `/api/health` 的 store 探测失败返回 503。

## Decision

**硬失败**：`store.Open` 失败 → 进程退出（`main.go:61-66`）；`/api/health` 的 store 探测失败 → 503（`health.go:9-18`）。不做降级。

## Alternatives Considered

### 降级只读
- Pros：部分功能可用。
- Cons：**写丢失风险**——内存中的账号状态与磁盘不一致时，任何写操作（含后台持久化）都可能静默丢数据；「看起来可用」比「明确不可用」更危险（用户可能在配置丢失状态下继续操作）。
- Rejected：账号配置存储是本系统最不该丢的数据。

## Consequences

- 存储故障可被健康检查与容器编排（HEALTHCHECK / 重启策略）立即发现。
- 运维需保证 `/data` 卷（或存储文件路径）持久化与权限正确。
- 部署文档明确此语义（[docs/deployment.md](../deployment.md) §6.1）。
