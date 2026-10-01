package store

import "sync/atomic"

// writeFailures 统计账号库**真实写入失败**的次数（落盘 / SQL 写入出错，写操作已回滚）。
//
// 用于 /api/metrics 的 `s3c_store_write_failures_total`（ROADMAP #18）。为什么需要它：
// `json` / `encrypted` 驱动的 `Ping()` 恒为 nil，`s3c_store_up` 对这两类驱动永远是 1
// （见 docs/data-model.md §0 与 OPERATIONS.md §3.1），存储故障只能以业务 5xx 的形式
// 被动暴露；写入失败计数是这两类驱动**主动**暴露「磁盘写满 / 只读挂载」的信号。
// 业务性拒绝（重复 ID、NotFound）不计入——否则客户端 4xx 会触发存储告警。
var writeFailures atomic.Int64

// WriteFailureCount 返回账号库写入失败的累计次数（进程级，重启清零）。
func WriteFailureCount() int64 { return writeFailures.Load() }

// noteWriteFailure 记一次写入失败；只在「真实写入尝试出错」的分支调用。
func noteWriteFailure() { writeFailures.Add(1) }
