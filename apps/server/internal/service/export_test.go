package service

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// export_test.go —— 仅在 `go test` 编译本包时参与构建，不进入生产二进制
//（与 handler/export_test.go 同一纪律：测试接缝不进生产文件，
// 见 docs/archive/code-review-2026-09-24.md R19b）。

// Create 注册新任务的测试便利封装：容量超限时不返回 error，而是给出一个
// 已终结的任务，使测试不必逐处处理 ErrTooManyJobs。生产路径一律走
// TryCreate（handler newJob），本方法只服务本包测试。
func (r *JobRegistry) Create(total int, cancel context.CancelFunc) *Job {
	j, err := r.TryCreate(total, cancel)
	if err != nil {
		// 仅在容量超限时发生：给出一个已终结的任务，
		// 使调用方拿到合法 *Job 而不 panic，且不会真正占用资源。
		j = &Job{
			ID:       uuid.NewString(),
			Created:  time.Now(),
			Total:    total,
			progress: JobProgress{Total: total, Status: JobStatusCancelled},
			done:     true,
			subs:     make(map[chan JobProgress]struct{}),
		}
	}
	return j
}
