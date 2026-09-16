package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/weilai1949/s3clinet/server/internal/service"
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
