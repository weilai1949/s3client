package service

// job_finish_reap_test.go —— docs/archive/code-review-2026-09-24.md:
//   - R8：Reap 的 TTL 必须以「完成时间」为准——跑超 30 分钟的任务 Finish 后
//     不得在 ≤5 分钟内被清掉（列表丢终态、轮询 404、jobs.json 记录消失）；
//   - R9：Finish 之后迟到的 running 帧不得把已完成任务改写为非终态，
//     否则重启后 restore() 误标 interrupted，触发虚假对账告警。
//
// 断言的都是外部可见行为：Get/List 的查询结果与持久化文件内容。

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestLongRunningJobSurvivesReapWithinTTL R8：完成时间才是 TTL 起点。
// 创建于 2 小时前的任务 Finish 后必须仍可查询——旧实现以 Created 计 TTL，
// 这样的任务 Finish 后第一次 reap（≤5 分钟）就被清掉。
func TestLongRunningJobSurvivesReapWithinTTL(t *testing.T) {
	r := NewJobRegistry()
	defer r.Stop()

	j, err := r.TryCreate(1, func() {})
	if err != nil {
		t.Fatalf("TryCreate: %v", err)
	}
	j.mu.Lock()
	j.Created = time.Now().Add(-2 * time.Hour) // 长耗时任务：创建于 2 小时前
	j.mu.Unlock()
	j.Finish(JobResult{Migrated: 1}, JobStatusDone)

	r.Reap()

	if _, ok := r.Get(j.ID); !ok {
		t.Fatal("刚完成的长耗时任务在 TTL 内必须仍可查询（TTL 应从完成时间起算）")
	}
}

// TestEmitAfterFinishKeepsTerminalState R9：RunBatch 每帧进度都带 Status:"running"，
// Finish 之后迟到的帧必须被丢弃——内存快照与持久化文件都得保持终态。
// 持久化用真实文件：R9 的危害发生在重启后 restore() 读到非终态记录时。
func TestEmitAfterFinishKeepsTerminalState(t *testing.T) {
	// 每次 Emit 都落盘，否则迟到帧只改内存，盖不住「持久化也保持终态」这条断言。
	old := jobProgressPersistEvery
	jobProgressPersistEvery = 0
	t.Cleanup(func() { jobProgressPersistEvery = old })

	path := filepath.Join(t.TempDir(), "jobs.json")
	p := NewFileJobPersister(path)
	r := NewJobRegistryWithPersister(p)
	defer r.Stop()

	j, err := r.TryCreate(2, func() {})
	if err != nil {
		t.Fatalf("TryCreate: %v", err)
	}
	j.Finish(JobResult{Migrated: 2}, JobStatusDone)

	// Finish 之后迟到的 running 帧（异步批量 goroutine 收尾时常见）。
	j.Emit(JobProgress{Done: 1, Total: 2, Status: JobStatusRunning})

	prog, _, done := j.Snapshot()
	if !done || prog.Status != JobStatusDone {
		t.Fatalf("snapshot = %+v done=%v, want terminal %q（迟到 running 帧改写了终态）",
			prog, done, JobStatusDone)
	}

	recs := r.List()
	if len(recs) != 1 || recs[0].Status != JobStatusDone || recs[0].Progress.Status != JobStatusDone {
		t.Fatalf("list = %+v, want done status kept", recs)
	}

	got, err := p.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(got) != 1 || got[0].Status != JobStatusDone || got[0].Progress.Status != JobStatusDone {
		t.Fatalf("persisted = %+v, want terminal done（持久化文件被迟到帧写回非终态）", got)
	}
}

// TestFinishedJobTTLPersistsFinishTime R8：完成时刻必须随记录落盘，重启恢复后
// Reap 仍按完成时刻算 TTL——否则「创建于 2 小时前、刚完成」的任务恢复后第一次
// reap（≤5 分钟）就被清掉，列表丢终态、轮询 404。
func TestFinishedJobTTLPersistsFinishTime(t *testing.T) {
	path := filepath.Join(t.TempDir(), "jobs.json")
	p := NewFileJobPersister(path)

	r1 := NewJobRegistryWithPersister(p)
	j, err := r1.TryCreate(1, func() {})
	if err != nil {
		t.Fatalf("TryCreate: %v", err)
	}
	j.mu.Lock()
	j.Created = time.Now().Add(-2 * time.Hour) // 长耗时任务：创建于 2 小时前
	j.mu.Unlock()
	j.Finish(JobResult{Migrated: 1}, JobStatusDone)
	r1.Stop() // 终态已落盘，先停掉写方再重启恢复

	// 落盘文件必须带 finishedAt（wire 名固定，omitzero 只对零值省略）。
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read jobs.json: %v", err)
	}
	if !strings.Contains(string(raw), `"finishedAt"`) {
		t.Fatalf("jobs.json 缺少 finishedAt 字段: %s", raw)
	}

	r2 := NewJobRegistryWithPersister(p)
	defer r2.Stop()
	r2.Reap()

	if _, ok := r2.Get(j.ID); !ok {
		t.Fatal("恢复后的刚完成任务在 TTL 内必须仍可查询（TTL 应从落盘的完成时间起算）")
	}
}

// TestLegacyRecordWithoutFinishedAtFallsBackToCreated 旧版 jobs.json 没有
// finishedAt：Reap 回退按 Created 计 TTL——历史到期记录仍会被清理（不残留），
// 这也是「完成时刻缺失」分支的行为锚点。
func TestLegacyRecordWithoutFinishedAtFallsBackToCreated(t *testing.T) {
	p := &fakeJobPersister{loaded: []JobRecord{
		{ID: "old-done", Created: time.Now().Add(-2 * time.Hour), Total: 1, Status: JobStatusDone},
	}}
	r := NewJobRegistryWithPersister(p)
	defer r.Stop()

	r.Reap()

	if got := r.List(); len(got) != 0 {
		t.Fatalf("list after reap = %+v, want empty（无 finishedAt 的旧记录应回退 Created 计 TTL）", got)
	}
}
