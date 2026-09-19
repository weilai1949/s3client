package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/weilai1949/s3clinet/apps/server/internal/service"
)

// TestJobsListEndpoint 验证 GET /api/migrate/jobs 返回任务清单（含未完成/中断任务）。
func TestJobsListEndpoint(t *testing.T) {
	h, _ := gapStoreHandler(t)
	h.SetJobPersister(service.NewFileJobPersister(filepath.Join(t.TempDir(), "jobs.json")))
	t.Cleanup(h.Shutdown)

	// 造一个已完成任务，确保清单能真实反映注册表内容。
	_, cancel := context.WithCancel(context.Background())
	job := h.migrateJobs.Create(2, cancel)
	job.Finish(service.JobResult{Migrated: 2}, service.JobStatusDone)

	rr := httptest.NewRecorder()
	h.jobsList(rr, httptest.NewRequest(http.MethodGet, "/api/migrate/jobs", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rr.Code, rr.Body)
	}

	var resp struct {
		Jobs []service.JobRecord `json:"jobs"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v; body=%s", err, rr.Body)
	}
	if len(resp.Jobs) != 1 {
		t.Fatalf("jobs = %+v, want 1", resp.Jobs)
	}
	if resp.Jobs[0].ID != job.ID || resp.Jobs[0].Status != service.JobStatusDone {
		t.Errorf("job = %+v, want %s/done", resp.Jobs[0], job.ID)
	}
	if resp.Jobs[0].Result.Migrated != 2 {
		t.Errorf("result.migrated = %d, want 2", resp.Jobs[0].Result.Migrated)
	}
}

// TestJobsListEmptyIsArray 空清单必须序列化为 []，不能是 null（前端直接 .length / map）。
func TestJobsListEmptyIsArray(t *testing.T) {
	h, _ := gapStoreHandler(t)
	t.Cleanup(h.Shutdown)

	rr := httptest.NewRecorder()
	h.jobsList(rr, httptest.NewRequest(http.MethodGet, "/api/migrate/jobs", nil))

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(rr.Body.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	if string(raw["jobs"]) != "[]" {
		t.Errorf("jobs = %s, want []", raw["jobs"])
	}
}

// TestSetJobPersisterRecoversInterruptedJobs 验证启动装配路径：
// 注入 persister 后，历史 running 任务对前端呈现为 interrupted。
func TestSetJobPersisterRecoversInterruptedJobs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "jobs.json")

	// 模拟「上一次进程崩溃时留下 running 任务」。
	seed := service.NewFileJobPersister(path)
	if err := seed.Save([]service.JobRecord{{
		ID: "crashed", Total: 3, Status: service.JobStatusRunning,
		Progress: service.JobProgress{Done: 1, Total: 3, Migrated: 1, Status: service.JobStatusRunning},
	}}); err != nil {
		t.Fatal(err)
	}

	h, _ := gapStoreHandler(t)
	h.SetJobPersister(service.NewFileJobPersister(path))
	t.Cleanup(h.Shutdown)

	// 轮询接口必须能查到该任务，且为终态（前端据此提示「已中断，需对账」）。
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/migrate/jobs/crashed", nil)
	req.SetPathValue("id", "crashed")
	h.migrateJobStatus(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rr.Code, rr.Body)
	}
	var resp struct {
		Done     bool                `json:"done"`
		Progress service.JobProgress `json:"progress"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if !resp.Done {
		t.Error("done = false, want true for interrupted job")
	}
	if resp.Progress.Status != service.JobStatusInterrupted {
		t.Errorf("status = %q, want interrupted", resp.Progress.Status)
	}
}

// TestSetJobPersisterReplacesRegistry 替换 persister 时旧注册表必须停止（避免 goroutine 泄漏）。
func TestSetJobPersisterReplacesRegistry(t *testing.T) {
	h, _ := gapStoreHandler(t)
	old := h.migrateJobs

	_, cancel := context.WithCancel(context.Background())
	job := old.Create(1, cancel)
	job.Finish(service.JobResult{Migrated: 1}, service.JobStatusDone)

	h.SetJobPersister(service.NewFileJobPersister(filepath.Join(t.TempDir(), "j.json")))
	t.Cleanup(h.Shutdown)

	if h.migrateJobs == old {
		t.Fatal("registry not replaced")
	}
	// 旧注册表已停止：其 reap 循环退出，Stop 幂等不 panic。
	old.Stop()
}

// TestNewJobReturns503AtCapacity 在册任务达上限时异步端点返回 503 而非无限接受
// （todolist #17 / ASSESSMENT M4）。用 ctx 取消状态间接确认 cancel() 被调用，
// 避免注册失败时泄漏 WithTimeout 定时器。
func TestNewJobReturns503AtCapacity(t *testing.T) {
	old := service.SetMaxJobsForTest(1)
	t.Cleanup(func() { service.SetMaxJobsForTest(old) })

	h, _ := gapStoreHandler(t)
	t.Cleanup(h.Shutdown)

	// 占满唯一名额（首个 job 的返回值不参与断言，只需注册成功）。
	if _, ok := h.newJob(httptest.NewRecorder(), 1, func() {}); !ok {
		t.Fatal("first job should be accepted")
	}

	// 第二个必须 503，且传入的 cancel 必须已被调用（否则定时器泄漏）。
	cancelled := false
	rr := httptest.NewRecorder()
	job, ok := h.newJob(rr, 1, func() { cancelled = true })
	if ok || job != nil {
		t.Fatalf("job=%v ok=%v, want nil/false at capacity", job, ok)
	}
	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503; body=%s", rr.Code, rr.Body)
	}
	if !cancelled {
		t.Error("cancel() must be called when registration fails")
	}
}
