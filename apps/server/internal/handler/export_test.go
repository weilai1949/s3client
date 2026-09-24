package handler

import (
	"errors"
	"testing"

	"github.com/weilai1949/s3clinet/apps/server/internal/service"
)

// export_test.go —— 仅在 `go test` 编译本包时参与构建，不进入生产二进制。
//
// 这里放"需要触达 handler 私有字段、但不该出现在生产 API 里"的测试接缝。
// 禁止把 `*ForTest` 之类的钩子写进生产文件（见 docs/archive/review-2026-09-19.md §A2）。

// FillJobSlotsForTest 把 handler 内部的异步任务注册表填到只剩一个在册名额：
// 先创建至触发 ErrTooManyJobs，再终结最后一个任务释放一格——调用方随后
// 「首成功、次 503」的边界断言与原 SetJobCapForTest(h, 1) 语义一致。
//
// 不再注入小上限：service 的在册上限不是可配置项——原 RegistryOption / WithMaxJobs
// 仅测试引用，已按死代码纪律删除（默认上限见 service 的 defaultMaxJobs）。
func FillJobSlotsForTest(t *testing.T, h *Handler) {
	t.Helper()
	h.migrateJobs.Stop()
	h.migrateJobs = service.NewJobRegistry()

	var last *service.Job
	reached := false
	for created := 0; created <= 4096; created++ { // 安全上界：上限意外消失时显式失败而非挂死
		j, err := h.migrateJobs.TryCreate(1, func() {})
		if err != nil {
			if !errors.Is(err, service.ErrTooManyJobs) {
				t.Fatalf("填满在册名额: %v", err)
			}
			reached = true
			break
		}
		last = j
	}
	if !reached || last == nil {
		t.Fatal("在册任务必须有上限且首个创建必须成功（复核 defaultMaxJobs 与安全上界）")
	}
	// 终结最后一个任务释放一格：终态任务仍留在 List 中，但不占「进行中」名额。
	last.Finish(service.JobResult{}, service.JobStatusDone)
}
