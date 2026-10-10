package service

// persist_metrics.go —— 计划 / 任务清单落盘失败的可见性。
//
// Scheduler 与 JobRegistry 的落盘遵循「失败静默降级、下一次状态变更重试」的既有
// 设计（内存态不受影响，见 scheduler_test「save 失败不影响内存态」），但原先
// `_ = persister.Save(...)` 连失败事实都不留下，运维无从发现磁盘满 / 只读
// （KNOWN_ISSUES #83）。这里只**计数**、不改变降级行为，由 /api/metrics 的
// `s3c_persist_failures_total` 暴露以便告警。

import "sync/atomic"

// persistFailures 累计计划 / 任务清单落盘失败次数（进程级，不清零）。
var persistFailures atomic.Int64

// notePersistFailure 记一次落盘失败（计划 / 任务持久化路径调用）。
func notePersistFailure() { persistFailures.Add(1) }

// PersistFailureCount 返回落盘失败累计次数，供 /api/metrics 暴露。
func PersistFailureCount() int64 { return persistFailures.Load() }
