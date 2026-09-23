package service

// job_cover_test.go —— job.go 剩余分支的行为测试：
//   - restore 跳过无 ID 的损坏记录；
//   - List 在 Created 相同时按 ID 升序稳定排序；
//   - Create 在容量耗尽时的兼容语义（返回已终结任务，不注册）；
//   - 无状态进度帧不得让清单快照出现空状态；
//   - Emit 的中间进度落盘节流；
//   - WithMaxJobs 的上限生效语义（每实例，不影响其它注册表）。
//
// 这些断言的是「外部可见行为」（清单内容、落盘次数、任务终态），不是内部变量。

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestJobRegistryRestoreSkipsRecordsWithoutID 损坏记录（无 ID）必须被跳过：
// 没有 ID 的任务无法被查询/取消/对账，注册进清单只会污染前端视图。
func TestJobRegistryRestoreSkipsRecordsWithoutID(t *testing.T) {
	p := &fakeJobPersister{loaded: []JobRecord{
		{ID: "", Total: 1, Status: JobStatusRunning},
		{ID: "keep", Total: 1, Status: JobStatusDone},
	}}
	r := NewJobRegistryWithPersister(p)
	defer r.Stop()

	got := r.List()
	if len(got) != 1 || got[0].ID != "keep" {
		t.Fatalf("list = %+v, want only the record with an id", got)
	}
}

// TestJobRegistryListTieBreaksByIDAscending Created 相同时按 ID 升序：
// map 迭代顺序随机，若无 tie-break，同一份清单每次序列化都会抖动，
// 前端列表与落盘文件会出现无意义的顺序变化。
func TestJobRegistryListTieBreaksByIDAscending(t *testing.T) {
	r := NewJobRegistry()
	defer r.Stop()

	_, cancel := context.WithCancel(context.Background())
	a := r.Create(1, cancel)
	b := r.Create(1, cancel)

	// 强制同刻创建，让排序走到 ID 比较分支。
	created := time.Now()
	for _, j := range []*Job{a, b} {
		j.mu.Lock()
		j.Created = created
		j.mu.Unlock()
	}

	got := r.List()
	if len(got) != 2 {
		t.Fatalf("list = %+v, want 2", got)
	}
	if got[0].ID > got[1].ID {
		t.Errorf("order = %s,%s; want ascending by id when Created is equal", got[0].ID, got[1].ID)
	}
}

// TestJobRegistryCreateReturnsTerminalJobAtCapacity Create 保持历史签名（不返回 error）：
// 容量耗尽时必须给出一个「已终结」的合法 *Job——调用方拿到它不会 panic，也不会
// 误以为任务在运行而无限等待；同时该任务不得占用在册名额。
func TestJobRegistryCreateReturnsTerminalJobAtCapacity(t *testing.T) {
	r := NewJobRegistry(WithMaxJobs(1))
	defer r.Stop()

	_, cancel := context.WithCancel(context.Background())
	if _, err := r.TryCreate(1, cancel); err != nil {
		t.Fatalf("occupy slot: %v", err)
	}

	j := r.Create(2, cancel)
	if j == nil || j.ID == "" {
		t.Fatalf("Create = %+v, want a usable job", j)
	}
	prog, _, done := j.Snapshot()
	if !done {
		t.Error("fallback job must be terminal so callers do not wait forever")
	}
	if prog.Status != JobStatusCancelled || prog.Total != 2 {
		t.Errorf("progress = %+v, want cancelled/2", prog)
	}
	if _, ok := r.Get(j.ID); ok {
		t.Error("rejected job must not be registered")
	}
	if got := len(r.List()); got != 1 {
		t.Errorf("jobs = %d, want 1 (only the occupying job)", got)
	}
}

// TestJobEmitPersistsProgressThrottled Emit 的中间进度落盘按窗口节流：
// 窗口内写盘、窗口外丢弃——批量任务每帧写盘会把 IO 放大到不可接受，
// 但完全不写又会让重启后的进度退回 0。
func TestJobEmitPersistsProgressThrottled(t *testing.T) {
	old := jobProgressPersistEvery
	t.Cleanup(func() { jobProgressPersistEvery = old })

	p := &fakeJobPersister{}
	r := NewJobRegistryWithPersister(p)
	defer r.Stop()

	_, cancel := context.WithCancel(context.Background())
	job := r.Create(3, cancel)
	base := len(p.saves) // Create 已落盘一次

	// 窗口为 0：本次 Emit 必须写盘，且落盘内容反映最新进度。
	jobProgressPersistEvery = 0
	job.Emit(JobProgress{Done: 1, Total: 3, Migrated: 1, Status: JobStatusRunning})
	if len(p.saves) != base+1 {
		t.Fatalf("saves = %d, want %d (Emit must persist within window)", len(p.saves), base+1)
	}
	if got := p.saves[len(p.saves)-1][0].Progress.Done; got != 1 {
		t.Errorf("persisted progress.done = %d, want 1", got)
	}

	// 窗口远大于间隔：下一次 Emit 被节流，不得写盘。
	jobProgressPersistEvery = time.Hour
	before := len(p.saves)
	job.Emit(JobProgress{Done: 2, Total: 3, Migrated: 2, Status: JobStatusRunning})
	if len(p.saves) != before {
		t.Fatalf("saves = %d, want %d (Emit must be throttled)", len(p.saves), before)
	}
}

// TestJobRecordStatusFallsBackForStatuslessFrames 进度帧允许不带 status
// （JobProgress.Status 是 omitempty，service.Progress.Status 同样可选），但清单快照
// 必须给出确定状态：空状态落盘后会被 restore 当成「非终态」，把已完成任务误标为
// interrupted，进而触发「需人工对账」的假告警。
func TestJobRecordStatusFallsBackForStatuslessFrames(t *testing.T) {
	r := NewJobRegistry()
	defer r.Stop()

	_, cancel := context.WithCancel(context.Background())
	running, err := r.TryCreate(2, cancel)
	if err != nil {
		t.Fatal(err)
	}
	running.Emit(JobProgress{Done: 1, Total: 2}) // 不带 Status 的中间帧

	finished, err := r.TryCreate(1, cancel)
	if err != nil {
		t.Fatal(err)
	}
	finished.Finish(JobResult{Migrated: 1}, JobStatusDone)
	finished.Emit(JobProgress{Done: 1, Total: 1}) // 终态后迟到的无状态帧

	byID := map[string]JobRecord{}
	for _, rec := range r.List() {
		byID[rec.ID] = rec
	}
	if got := byID[running.ID].Status; got != JobStatusRunning {
		t.Errorf("running job status = %q, want %q", got, JobStatusRunning)
	}
	rec := byID[finished.ID]
	if rec.Status != JobStatusDone {
		t.Errorf("finished job status = %q, want %q (迟到帧不得把终态降级)", rec.Status, JobStatusDone)
	}
	if rec.Progress.Status != JobStatusDone {
		t.Errorf("finished job progress.status = %q, want %q", rec.Progress.Status, JobStatusDone)
	}
}

// TestWithMaxJobsLimitsRegistry 在册上限是**每个注册表实例**的属性：
// WithMaxJobs(n) 生效后第 n+1 个未终结任务必须被拒（ErrTooManyJobs），
// 且不影响其它实例（原全局钩子会让并发用例相互污染，见
// docs/archive/review-2026-09-19.md §A2）。
func TestWithMaxJobsLimitsRegistry(t *testing.T) {
	limited := NewJobRegistry(WithMaxJobs(1))
	defer limited.Stop()
	if _, err := limited.TryCreate(1, func() {}); err != nil {
		t.Fatalf("first job: %v", err)
	}
	if _, err := limited.TryCreate(1, func() {}); !errors.Is(err, ErrTooManyJobs) {
		t.Fatalf("second job err = %v, want ErrTooManyJobs", err)
	}

	// 另一个实例不受影响：默认上限远大于 1。
	other := NewJobRegistry()
	defer other.Stop()
	if _, err := other.TryCreate(1, func() {}); err != nil {
		t.Fatalf("independent registry should accept: %v", err)
	}
}

// TestWithMaxJobsIgnoresNonPositive 非正上限视为未设置，回落默认值（构造期防御）。
func TestWithMaxJobsIgnoresNonPositive(t *testing.T) {
	r := NewJobRegistry(WithMaxJobs(0))
	defer r.Stop()
	if r.maxJobs != defaultMaxJobs {
		t.Fatalf("maxJobs = %d, want default %d", r.maxJobs, defaultMaxJobs)
	}
}
