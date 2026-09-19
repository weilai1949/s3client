package handler

import "github.com/weilai1949/s3clinet/apps/server/internal/service"

// export_test.go —— 仅在 `go test` 编译本包时参与构建，不进入生产二进制。
//
// 这里放"需要触达 handler 私有字段、但不该出现在生产 API 里"的测试接缝。
// 禁止把 `*ForTest` 之类的钩子写进生产文件（见 docs/review-2026-09-19.md §A2）。

// SetJobCapForTest 把 handler 内部的异步任务注册表换成一个在册上限为 n 的新实例，
// 用于验证「任务达上限 → 异步端点返回 503」；旧注册表会被 Stop（其 reap 协程退出）。
func SetJobCapForTest(h *Handler, n int) {
	h.migrateJobs.Stop()
	h.migrateJobs = service.NewJobRegistry(service.WithMaxJobs(n))
}
