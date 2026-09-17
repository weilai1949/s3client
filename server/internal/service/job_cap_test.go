package service

import (
	"context"
	"errors"
	"testing"
)

// TestJobRegistryTryCreateRejectsBeyondCap JobRegistry 必须有总任务上限：
// 每个任务都持有 goroutine、订阅者与落盘条目，无上限时短时间大量请求可耗尽资源
// （todolist #17 / ASSESSMENT M4）。
func TestJobRegistryTryCreateRejectsBeyondCap(t *testing.T) {
	old := maxJobs
	maxJobs = 3
	t.Cleanup(func() { maxJobs = old })

	r := NewJobRegistry()
	defer r.Stop()

	_, cancel := context.WithCancel(context.Background())
	for i := 0; i < 3; i++ {
		j, err := r.TryCreate(1, cancel)
		if err != nil {
			t.Fatalf("create %d: unexpected err %v", i, err)
		}
		if j == nil {
			t.Fatalf("create %d: nil job", i)
		}
	}

	// 第 4 个必须被拒绝，且不得留下任何副作用（jobs 数量不变）。
	j, err := r.TryCreate(1, cancel)
	if !errors.Is(err, ErrTooManyJobs) {
		t.Fatalf("err = %v, want ErrTooManyJobs", err)
	}
	if j != nil {
		t.Errorf("job = %+v, want nil on rejection", j)
	}
	if got := len(r.List()); got != 3 {
		t.Errorf("jobs = %d, want 3 (rejected job must not be registered)", got)
	}
}

// TestJobRegistryTryCreateAllowsAfterTerminal 任务完成后应能再创建：
// 上限约束的是「在册任务数」，不能因为历史任务堆积而永久拒绝新任务。
func TestJobRegistryTryCreateAllowsAfterTerminal(t *testing.T) {
	old := maxJobs
	maxJobs = 1
	t.Cleanup(func() { maxJobs = old })

	r := NewJobRegistry()
	defer r.Stop()
	_, cancel := context.WithCancel(context.Background())

	first, err := r.TryCreate(1, cancel)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.TryCreate(1, cancel); !errors.Is(err, ErrTooManyJobs) {
		t.Fatalf("err = %v, want ErrTooManyJobs while running", err)
	}

	first.Finish(JobResult{Migrated: 1}, JobStatusDone)

	// 终态任务不再占用「进行中」名额。
	if _, err := r.TryCreate(1, cancel); err != nil {
		t.Fatalf("create after finish: %v", err)
	}
}

// TestJobRegistryCreateUnchanged Create 保持历史语义（不返回 error、总是给出任务），
// 以免波及 70+ 处既有调用点。
func TestJobRegistryCreateUnchanged(t *testing.T) {
	r := NewJobRegistry()
	defer r.Stop()
	_, cancel := context.WithCancel(context.Background())
	j := r.Create(1, cancel)
	if j == nil || j.ID == "" {
		t.Fatalf("Create returned %+v", j)
	}
}

// TestJobRegistryCapCountsRunningOnly 已中断任务属终态，不占用进行中名额。
func TestJobRegistryCapCountsRunningOnly(t *testing.T) {
	old := maxJobs
	maxJobs = 1
	t.Cleanup(func() { maxJobs = old })

	p := &fakeJobPersister{loaded: []JobRecord{
		{ID: "old-interrupted", Total: 1, Status: JobStatusRunning},
	}}
	r := NewJobRegistryWithPersister(p)
	defer r.Stop()

	_, cancel := context.WithCancel(context.Background())
	if _, err := r.TryCreate(1, cancel); err != nil {
		t.Fatalf("interrupted job should not consume a running slot: %v", err)
	}
}
