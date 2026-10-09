package handler

// storage_report_cancel_test.go —— 补 storageReport 列举前的 ctx 取消分支
// （storage_report_test.go 的 HTTP 用例覆盖不到：请求上下文已在进入 handler 前取消）。
// 直调 handler 并注入已取消上下文 + PathValue，断言不会访问 S3 且返回非 2xx。

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStorageReportCancelledContext(t *testing.T) {
	env := srFake(t, func(string) string {
		t.Fatal("ctx 已取消时不得访问 S3")
		return ""
	})
	req := httptest.NewRequest(http.MethodGet, "/api/accounts/"+env.acc.ID+"/storage-report", nil)
	req.SetPathValue("id", env.acc.ID)
	ctx, cancel := context.WithCancel(req.Context())
	cancel()
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	env.hnd.storageReport(rr, req)
	if rr.Code < 400 {
		t.Fatalf("cancelled storageReport status = %d, want >= 400（不访问 S3）", rr.Code)
	}
}
