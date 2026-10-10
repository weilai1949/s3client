# ADR-009：批量与流式请求全部有界并发

## Status

Accepted

## Date

2026-09-18（回溯记录；`service/batch.go` / `handler/stream.go` 于 2026-09-18 首次引入）

## Context

批量复制 / 删除 / 迁移与流式下载 / 代理是资源放大器：无界并发会让内存、goroutine 与上游
S3 连接被短时间内的大量请求拖垮。目标：**内存有界 + 过载显式失败（503）**，而不是拖垮进程。

现状（2026-09-30 回读源码核实，常量逐条比对）：

- [`apps/server/internal/service/batch.go`](../../apps/server/internal/service/batch.go)
  `RunBatch`：`jobs` / `results` 均**无缓冲**通道——worker 提交结果即与消费者同步，内存
  O(workers)；进度逐条即时回调（旧实现 `wg.Wait()` 后排空 results，进度全程停在 0/total）。
- [`apps/server/internal/handler/stream.go`](../../apps/server/internal/handler/stream.go)：
  全局 **32** 并发（`maxConcurrentStreams`，channel 信号量），饱和返回 503
  `too many concurrent streaming requests`；**滚动空闲写超时 5 分钟**（`streamIdleTimeout`，
  每次成功写出后刷新——慢网大文件不会被绝对截止误杀）。
- [`apps/server/internal/service/zip.go`](../../apps/server/internal/service/zip.go)：
  拉取有界并发 **4**（`zipFetchWorkers`），写入串行（`zip.Writer` 非并发安全）。
- 前端：分段上传 **4** 路并发（`PART_CONCURRENCY`）、上传队列同时 **2** 个文件
  （`UPLOAD_CONCURRENCY`）、批量改元数据 **4** 路（`BATCH_META_CONCURRENCY`）。
- 任务在册上限 256（见 [ADR-005](0005-sse-async-jobs.md) 的 `defaultMaxJobs`）。

## Decision

**所有批量 / 流式路径设固定上限：无缓冲通道（内存 O(workers)）、流式全局 32 并发、ZIP
拉取 4、上传队列 2、分段 4；上限是常量，不做配置项。** 饱和时显式 503 / 拒绝。

## Alternatives Considered

### 无界并发（来多少开多少）
- Pros：实现最简。
- Cons：内存 / goroutine 耗尽（与 `defaultMaxJobs` 的背景一致：KNOWN_ISSUES #17 /
  [`../archive/assessment.md`](../archive/assessment.md) M4）。
- 已被推翻。

### 缓冲通道收集全部结果再处理
- Pros：实现简单。
- Cons：内存 O(n) + 进度一次性灌出（`batch.go` 注释记录旧实现即如此，进度全程停在 0/total）。
- 已被推翻。

### 上限做成配置项
- Pros：可按部署调参。
- Cons：当前无按部署调参需求，固定常量减少配置面；原 `RegistryOption` / `WithMaxJobs`
  只被测试引用，按死代码纪律删除（`job.go` 注释）。
- 被拒。

## Consequences

- 过载行为显式且可观测：503 可被客户端 / 网关感知（对应 Runbook R-5 / R-8，
  [`OPERATIONS.md`](../OPERATIONS.md)）。
- 常量值由单实例内存预算推导（compose 512MB 限额）；调整内存预算前必须先重算这些常量。
- 无缓冲通道的纪律：消费端必须**始终消费** results，否则 worker 永久阻塞
  （`zip.go` 注释 [`../archive/review-2026-09-19.md`](../archive/review-2026-09-19.md) §B4 记录过该失败模式）。
