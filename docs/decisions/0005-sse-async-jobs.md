# ADR-005：异步任务与进度推送统一走服务端 JobRegistry + SSE，前端单一订阅实现

## Status

Accepted

## Date

2026-09-18（回溯记录；决策自 2026-09-18 首次引入 `service/job.go` / `handler/stream.go` 起沿用）

## Context

批量复制 / 移动 / 删除等长任务需要进度反馈、可取消、跨重启可对账。历史教训（
[`DEVELOPMENT.md`](../DEVELOPMENT.md) §7 S7 / P0-4）：SSE 终态检测曾在
`MigratePanel` 与 `useObjectActions` / `DestDialog` **各写一份**，流以 EOF 结束时 Promise 悬挂、
`opsBusy` 永不复位、按钮永久禁用。约束：单进程内存注册表、无外部队列；SSE 复用既有 HTTP
鉴权与中间件，不引入第二套传输。

现状（2026-09-30 回读源码核实，路径与常量逐条比对）：

- [`apps/server/internal/service/job.go`](../../apps/server/internal/service/job.go)：`JobRegistry`
  内存注册表 + reap 循环；`Job` 持有进度 / 结果 / 订阅表；`TryCreate` 在册上限 256
  （`defaultMaxJobs`，注释：无上限会耗尽内存与 goroutine，KNOWN_ISSUES #17 / ASSESSMENT M4）；
  `JobTTL` 30 分钟（从**完成时刻** `finishedAt` 起算，review R8）、`JobInterruptedTTL` 7 天；
  `JobTimeout` 2 小时；`SSEHeartbeatEvery` 15 秒；每任务订阅上限 16（`maxSubscribersPerJob`）。
- [`apps/server/internal/service/job_persist.go`](../../apps/server/internal/service/job_persist.go)：
  `FileJobPersister` 可选落盘（整份 JSON 原子写）；Load/Save 失败**降级为内存态**，不阻止启动
  （与账号存储「不可用则硬失败」的取舍不同，见 [ADR-002](0002-store-fail-closed.md)）；恢复时
  非终态任务一律标记 `interrupted`（复制成功但源未删除的对账证据）。
- [`apps/server/internal/handler/migrate_async.go`](../../apps/server/internal/handler/migrate_async.go)：
  SSE 端点（`text/event-stream`、15 秒 `event: ping` 心跳、客户端断开感知、终态帧后关流）；
  任务上下文超时 `migrateJobTimeout = service.JobTimeout`。
- [`apps/web/src/api/jobs.ts`](../../apps/web/src/api/jobs.ts)：`subscribeMigrateEvents`
  **单一订阅实现**——SSE + EOF 后轮询回读直到终态（30s 上限、500ms 间隔）+ 45s 空闲超时
  （任何数据含心跳到达即重置）+ 连续 3 次回读失败快速 `onError`；三个调用方共用。

## Decision

**长任务一律注册进服务端 `JobRegistry` 并用 SSE 推送进度；前端只允许调用
`subscribeMigrateEvents` 这一个订阅实现。** 任务清单可选落盘；重启把非终态任务标记
`interrupted` 供用户对账；终态由服务端统一定义（`running` 之外均为终态，含 `interrupted`）。

## Alternatives Considered

### 前端纯轮询 `GET /api/migrate/jobs/{id}`
- Pros：实现简单、无长连接。
- Cons：进度延迟大、空闲请求浪费；按钮状态依赖前端计时器。
- **未被完全拒绝**：保留为 SSE 流 EOF 后的兜底轮询（`jobs.ts` 现状），与 SSE 主路径并存。

### WebSocket 双向通道
- Pros：双向通信。
- Cons：本场景只需要服务端 → 客户端单向推送；SSE 复用既有 HTTP 鉴权 / 中间件，无握手改造。
- （当时是否明确讨论过 WebSocket 未在仓库文档中找到记录，此对比为 2026-09-30 回溯补记的
  工程判断，**未验证**。）

### 任务清单不落盘（纯内存）
- Pros：实现最简。
- Cons：进程重启后「复制成功但源未删除」的对账证据丢失（ASSESSMENT S1 / KNOWN_ISSUES #19）。
- 已被推翻：落盘由 `FileJobPersister` 承载，失败只降级不硬失败。

### 每个调用方各自实现 SSE 订阅
- Pros：无。
- Cons：实测造成终态检测分叉与悬挂（[`DEVELOPMENT.md`](../DEVELOPMENT.md) §7 S7）。
- 已被收敛推翻：单一实现 + `subscribeMigrateEvents` 是唯一入口。

## Consequences

- 前端删除「终态判定」职责，三处调用方（`ctxDeleteFolder` / `DestDialog` / `MigratePanel`）共用单一实现。
- 新增失败模式被三层兜底覆盖：静默断流（心跳 + 45s 空闲超时）、EOF 无终态（回读轮询）、
  回读失败（连续 3 次快速 `onError`）。
- 服务端不变量（`job.go` 注释）：`Emit` 必须在锁内投递（防 send on closed channel panic）、
  `Finish` 之后丢弃迟到帧（防重启后被误标 `interrupted` 的虚假对账告警）、Reap TTL 从完成时刻起算。
- 任务清单落盘失败只降级，不阻止启动——与 ADR-002 的账号存储语义刻意不同。
