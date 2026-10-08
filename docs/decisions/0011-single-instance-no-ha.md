# ADR-011：单实例部署、无 HA 路径

## Status

Accepted

## Date

2026-09-19（回溯记录；`store/lock.go` 于 2026-09-19 落地 flock 单写者锁；「单实例」约束
本身自项目初版沿用）

## Context

账号存储是本地文件（json / sqlite / encrypted），`JobRegistry` 在内存中，单 token 模型
没有租约或选主机制。两个副本共享同一数据卷会**静默互相覆盖写入、重复执行迁移任务**
（[`store/lock.go`](../../apps/server/internal/store/lock.go) 注释，依据
[`ROADMAP.md`](../ROADMAP.md) §5.1 R4）。

现状（2026-09-30 回读核实）：

- [`apps/server/internal/store/lock.go`](../../apps/server/internal/store/lock.go)：
  `AcquireDataDirLock` 对 `S3C_DATA_DIR` 加 `flock`（`.s3client.lock`），第二实例**启动即
  失败**；锁由内核在进程退出（含 panic / SIGKILL）时释放，无陈旧锁文件；非 unix 平台
  **no-op**（`lock_other.go`），单副本约束仍靠部署方式保证。
- [`OPERATIONS.md`](../OPERATIONS.md) §7.2「为什么没有 HA 路径」明确六条约束与结论：
  单点故障，升级与恢复都必须停机；RTO 由发现时间 + 人工介入速度决定。
- 多副本 / HA（store 外置）是 [`ROADMAP.md`](../ROADMAP.md) §三 3.2 #15 的**未排期候选**。
- [`CONFIGURATION.md`](../CONFIGURATION.md) fail-closed 清单含「`S3C_DATA_DIR` 已被另一
  实例加锁 → 拒绝启动」。

## Decision

**明确维持单实例：文件型存储 + flock 单写者锁 + 启动即失败；不建设 HA 路径。** 多副本
场景明确列为未排期候选，而不是假装支持。

## Alternatives Considered

### 多实例 + 外置数据库 + 选主 / 租约
- Pros：可用性、横向扩展。
- Cons：引入外置状态与选主复杂度，与「本地文件自托管」定位冲突；当前无多副本部署需求
  （[`OPERATIONS.md`](../OPERATIONS.md) §7.2 结论）。
- 未采纳：列为未排期候选（ROADMAP §三 3.2 #15）。

### 不加锁，只靠部署纪律
- Pros：实现最简。
- Cons：误起第二个实例会静默破坏数据（互相覆盖写）。
- 已被推翻：flock + 启动即失败（`lock.go` 注释）。

### 应用层 marker 文件锁
- Pros：跨平台一致。
- Cons：进程崩溃残留陈旧锁文件需人工清理；内核 flock 随进程退出自动释放，无此问题
  （`lock.go` 注释）。
- 被拒。

## Consequences

- RTO 主要由发现时间 + 人工介入决定（[`OPERATIONS.md`](../OPERATIONS.md) §7.1）。
- 非 unix 平台锁 no-op 是**已知缺口**（[`OPERATIONS.md`](../OPERATIONS.md) §7.2、
  [`ROADMAP.md`](../ROADMAP.md) §5.1 R4）。
- 将来若做 HA：本 ADR 需被 Supersede，并重审存储（外置 / 复制）与任务表（持久化队列）设计。
