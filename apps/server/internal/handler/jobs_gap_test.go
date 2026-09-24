package handler

// jobs_gap_test.go —— 4 个异步端点在「在册任务达上限」时的行为：
// 必须返回 503（而不是无限接受并堆积 goroutine/SSE 订阅/落盘条目），
// 且不得因注册失败而泄漏 WithTimeout 派生的定时器。
//
// newJob 自身的 503 与 cancel 语义已由 jobs_test.go 覆盖；此处走真实 HTTP
// 路由，验证四个调用点都把 ok=false 正确转化为响应。

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/weilai1949/s3clinet/apps/server/internal/service"
)

// TestAsyncEndpointsReturn503AtCapacity 占满在册名额后，四个异步端点
// 均须返回 503，且不得注册新任务。
func TestAsyncEndpointsReturn503AtCapacity(t *testing.T) {
	srv := olFake(t, func(r *http.Request) olResp {
		if r.URL.Query().Has("list-type") {
			return olXML(http.StatusOK, listBucketXML([]string{"p/1"}, false, ""))
		}
		return olPlain(http.StatusOK)
	})
	env := accNewEnv(t, srv.URL, "b")
	t.Cleanup(env.hnd.Shutdown)
	FillJobSlotsForTest(t, env.hnd)

	dst := olCreateAccount(t, env, "dst", srv.URL, "ak", "db")

	// 占满剩余名额（不 Finish，保持「进行中」）。
	held, ok := env.hnd.newJob(httptest.NewRecorder(), 1, func() {})
	if !ok {
		t.Fatal("first job should be accepted")
	}
	t.Cleanup(func() { held.Finish(service.JobResult{}, service.JobStatusDone) })
	baseline := len(env.hnd.migrateJobs.List())

	cases := []struct {
		name, method, path, body string
	}{
		{"copy-objects/async", http.MethodPost, "/api/accounts/" + env.acc.ID + "/copy-objects/async",
			`{"bucket":"b","keys":["a.txt"]}`},
		{"copy-prefix/async", http.MethodPost, "/api/accounts/" + env.acc.ID + "/copy-prefix/async",
			`{"bucket":"b","prefix":"p/","targetPrefix":"q/"}`},
		{"delete-prefix/async", http.MethodPost, "/api/accounts/" + env.acc.ID + "/delete-prefix/async",
			`{"bucket":"b","prefix":"p/"}`},
		{"migrate/async", http.MethodPost, "/api/migrate/async",
			olMigrateBody(env.acc.ID, "b", dst.ID, "db", []string{"a.txt"})},
	}
	for _, c := range cases {
		rr := env.accDoRec(c.method, c.path, c.body)
		if rr.Code != http.StatusServiceUnavailable {
			t.Errorf("%s status = %d, want 503; body=%s", c.name, rr.Code, rr.Body.String())
		}
		if got := len(env.hnd.migrateJobs.List()); got != baseline {
			t.Errorf("%s: jobs = %d, want %d (rejected request must not register)", c.name, got, baseline)
		}
	}
}
