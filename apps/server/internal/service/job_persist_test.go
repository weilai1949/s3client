package service

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// fakeJobPersister 内存落盘替身：记录每次 Save 的清单，可预置 Load 结果与错误。
type fakeJobPersister struct {
	loaded    []JobRecord
	loadErr   error
	saves     [][]JobRecord
	saveErr   error
	loadCalls int
}

func (f *fakeJobPersister) Load() ([]JobRecord, error) {
	f.loadCalls++
	if f.loadErr != nil {
		return nil, f.loadErr
	}
	return f.loaded, nil
}

func (f *fakeJobPersister) Save(recs []JobRecord) error {
	cp := make([]JobRecord, len(recs))
	copy(cp, recs)
	f.saves = append(f.saves, cp)
	return f.saveErr
}

// ---- FileJobPersister ----

func TestFileJobPersisterRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "jobs.json")
	p := NewFileJobPersister(path)

	// 无既有文件：视为空清单，而非错误（首次启动路径）。
	recs, err := p.Load()
	if err != nil || recs != nil {
		t.Fatalf("load missing file = %v, %v; want nil, nil", recs, err)
	}

	created := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	want := []JobRecord{{
		ID: "job-1", Created: created, Total: 3,
		Status:   JobStatusRunning,
		Progress: JobProgress{Done: 1, Total: 3, Migrated: 1, Status: JobStatusRunning},
	}}
	if err := p.Save(want); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := p.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(got) != 1 || got[0].ID != "job-1" || got[0].Total != 3 {
		t.Fatalf("load = %+v, want 1 record job-1/3", got)
	}
	if !got[0].Created.Equal(created) {
		t.Errorf("created = %v, want %v", got[0].Created, created)
	}
	if got[0].Progress.Migrated != 1 {
		t.Errorf("progress.migrated = %d, want 1", got[0].Progress.Migrated)
	}

	// 落盘文件必须收紧权限，避免任务清单泄露 key/错误信息。
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("perm = %o, want 600", perm)
	}
}

func TestFileJobPersisterLoadEmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "jobs.json")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	recs, err := NewFileJobPersister(path).Load()
	if err != nil || recs != nil {
		t.Fatalf("load empty = %v, %v; want nil, nil", recs, err)
	}
}

func TestFileJobPersisterLoadErrors(t *testing.T) {
	t.Run("corrupt json", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "jobs.json")
		if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := NewFileJobPersister(path).Load(); err == nil {
			t.Fatal("want parse error")
		}
	})

	t.Run("unreadable path", func(t *testing.T) {
		dir := t.TempDir()
		// 目录当文件读：非 NotExist 的 IO 错误必须冒泡，而不是被当作空清单。
		if _, err := NewFileJobPersister(dir).Load(); err == nil {
			t.Fatal("want read error")
		}
	})
}

func TestFileJobPersisterSaveErrors(t *testing.T) {
	origWrite, origRename, origMarshal := jobWriteFile, jobRename, jobMarshal
	t.Cleanup(func() { jobWriteFile, jobRename, jobMarshal = origWrite, origRename, origMarshal })

	boom := errors.New("boom")
	rec := []JobRecord{{ID: "j", Status: JobStatusRunning}}

	t.Run("write fails", func(t *testing.T) {
		jobWriteFile = func(string, []byte, os.FileMode) error { return boom }
		defer func() { jobWriteFile = origWrite }()
		if err := NewFileJobPersister(filepath.Join(t.TempDir(), "j.json")).Save(rec); !errors.Is(err, boom) {
			t.Fatalf("err = %v, want boom", err)
		}
	})

	t.Run("marshal fails", func(t *testing.T) {
		jobMarshal = func(any) ([]byte, error) { return nil, boom }
		defer func() { jobMarshal = origMarshal }()
		if err := NewFileJobPersister(filepath.Join(t.TempDir(), "j.json")).Save(rec); !errors.Is(err, boom) {
			t.Fatalf("err = %v, want boom", err)
		}
	})

	t.Run("rename fails and removes temp file", func(t *testing.T) {
		jobRename = func(string, string) error { return boom }
		defer func() { jobRename = origRename }()
		path := filepath.Join(t.TempDir(), "j.json")
		if err := NewFileJobPersister(path).Save(rec); !errors.Is(err, boom) {
			t.Fatalf("err = %v, want boom", err)
		}
		// rename 失败后不得留下半成品临时文件（否则下次 Load 读到残缺 JSON）。
		if _, err := os.Stat(path + ".tmp"); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("temp file left behind: %v", err)
		}
	})
}

func TestIsTerminalJobStatus(t *testing.T) {
	for _, s := range []string{JobStatusDone, JobStatusCancelled, JobStatusInterrupted} {
		if !IsTerminalJobStatus(s) {
			t.Errorf("%q should be terminal", s)
		}
	}
	for _, s := range []string{JobStatusRunning, "", "weird"} {
		if IsTerminalJobStatus(s) {
			t.Errorf("%q should not be terminal", s)
		}
	}
}

// ---- JobRegistry 持久化接入 ----

func TestJobRegistryPersistsCreateAndFinish(t *testing.T) {
	p := &fakeJobPersister{}
	r := NewJobRegistryWithPersister(p)
	defer r.Stop()

	_, cancel := context.WithCancel(context.Background())
	job := r.Create(2, cancel)

	// Create 即落盘：进程若在此后崩溃，重启才能看到「未完成任务」。
	if len(p.saves) != 1 {
		t.Fatalf("saves after create = %d, want 1", len(p.saves))
	}
	if got := p.saves[0]; len(got) != 1 || got[0].ID != job.ID || got[0].Status != JobStatusRunning {
		t.Fatalf("create record = %+v, want running %s", got, job.ID)
	}

	job.Finish(JobResult{Migrated: 1, Failed: 1, FailKeys: []string{"b"}}, JobStatusDone)

	last := p.saves[len(p.saves)-1]
	if len(last) != 1 || last[0].Status != JobStatusDone {
		t.Fatalf("finish record = %+v, want done", last)
	}
	if last[0].Result.Migrated != 1 || last[0].Result.Failed != 1 || len(last[0].Result.FailKeys) != 1 {
		t.Errorf("result not persisted: %+v", last[0].Result)
	}
}

func TestJobRegistryRecoversInterruptedJobs(t *testing.T) {
	created := time.Now().Add(-time.Hour)
	p := &fakeJobPersister{loaded: []JobRecord{
		{
			ID: "was-running", Created: created, Total: 5, Status: JobStatusRunning,
			Progress: JobProgress{Done: 2, Total: 5, Migrated: 2, Status: JobStatusRunning},
		},
		{
			ID: "was-done", Created: created.Add(time.Minute), Total: 1, Status: JobStatusDone,
			Progress: JobProgress{Done: 1, Total: 1, Migrated: 1, Status: JobStatusDone},
		},
	}}
	r := NewJobRegistryWithPersister(p)
	defer r.Stop()

	// 未完成任务必须被标记为 interrupted：重启后不可能再继续跑，谎报 running 会误导前端。
	recs := r.List()
	byID := map[string]JobRecord{}
	for _, rec := range recs {
		byID[rec.ID] = rec
	}
	if len(recs) != 2 {
		t.Fatalf("list = %+v, want 2 records", recs)
	}
	if got := byID["was-running"].Status; got != JobStatusInterrupted {
		t.Errorf("was-running status = %q, want interrupted", got)
	}
	// 已完成任务保持原状态，不被误改。
	if got := byID["was-done"].Status; got != JobStatusDone {
		t.Errorf("was-done status = %q, want done", got)
	}

	// 恢复的任务必须能按 id 查到，且 SSE/轮询立即拿到终态。
	j, ok := r.Get("was-running")
	if !ok {
		t.Fatal("recovered job not retrievable by id")
	}
	prog, _, done := j.Snapshot()
	if !done || prog.Status != JobStatusInterrupted {
		t.Fatalf("snapshot = %+v done=%v, want interrupted/done", prog, done)
	}
	ch, _ := j.Subscribe()
	select {
	case p := <-ch:
		if p.Status != JobStatusInterrupted {
			t.Errorf("subscribe status = %q, want interrupted", p.Status)
		}
	case <-time.After(time.Second):
		t.Fatal("no terminal frame for recovered job")
	}

	// 恢复后未完成任务不再可取消（没有可取消的 goroutine）。
	if cancelled, alreadyDone := j.Cancel(); cancelled || !alreadyDone {
		t.Errorf("cancel = %v/%v, want false/true", cancelled, alreadyDone)
	}
}

func TestJobRegistryRecoveryPersistsInterruptedStatus(t *testing.T) {
	p := &fakeJobPersister{loaded: []JobRecord{
		{ID: "a", Status: JobStatusRunning, Total: 1},
	}}
	r := NewJobRegistryWithPersister(p)
	defer r.Stop()

	// 恢复即回写：否则每次重启都会对同一条记录重复告警。
	if len(p.saves) == 0 {
		t.Fatal("recovery did not persist interrupted status")
	}
	last := p.saves[len(p.saves)-1]
	if len(last) != 1 || last[0].Status != JobStatusInterrupted {
		t.Fatalf("persisted = %+v, want interrupted", last)
	}
}

// TestRecoveredInterruptedJobSurvivesReap 恢复的 interrupted 任务不得被 reap 立刻清掉。
// 它的 Created 早于 TTL 截止点，若沿用「done && Created < cutoff」判定，首次 reap
// （≤5 分钟）就会删除记录，使「重启后仍能看到未完成任务」失效。
func TestRecoveredInterruptedJobSurvivesReap(t *testing.T) {
	p := &fakeJobPersister{loaded: []JobRecord{
		{ID: "old-interrupted", Created: time.Now().Add(-24 * time.Hour), Total: 1, Status: JobStatusRunning},
	}}
	r := NewJobRegistryWithPersister(p)
	defer r.Stop()

	r.Reap()

	if got := r.List(); len(got) != 1 || got[0].ID != "old-interrupted" {
		t.Fatalf("list after reap = %+v, want interrupted job retained", got)
	}
}

func TestJobRegistryWithoutPersisterStaysInMemory(t *testing.T) {
	r := NewJobRegistry() // 不注入：历史行为，纯内存
	defer r.Stop()
	_, cancel := context.WithCancel(context.Background())
	job := r.Create(1, cancel)
	job.Finish(JobResult{Migrated: 1}, JobStatusDone)

	if got := r.List(); len(got) != 1 || got[0].Status != JobStatusDone {
		t.Fatalf("list = %+v, want 1 done record", got)
	}
}

func TestJobRegistryListOrdersNewestFirst(t *testing.T) {
	r := NewJobRegistry()
	defer r.Stop()

	_, cancel := context.WithCancel(context.Background())
	older := r.Create(1, cancel)
	time.Sleep(2 * time.Millisecond)
	newer := r.Create(1, cancel)

	got := r.List()
	if len(got) != 2 {
		t.Fatalf("list = %+v, want 2", got)
	}
	if got[0].ID != newer.ID || got[1].ID != older.ID {
		t.Errorf("order = %s,%s; want newest first (%s,%s)", got[0].ID, got[1].ID, newer.ID, older.ID)
	}
}

func TestJobRegistryListIncludesProgress(t *testing.T) {
	r := NewJobRegistry()
	defer r.Stop()
	_, cancel := context.WithCancel(context.Background())
	job := r.Create(3, cancel)
	job.Emit(JobProgress{Done: 2, Total: 3, Migrated: 2, Status: JobStatusRunning})

	got := r.List()
	if len(got) != 1 {
		t.Fatalf("list = %+v, want 1", got)
	}
	if got[0].Progress.Done != 2 || got[0].Progress.Migrated != 2 {
		t.Errorf("progress = %+v, want done=2 migrated=2", got[0].Progress)
	}
}

func TestJobRegistryToleratesPersisterErrors(t *testing.T) {
	boom := errors.New("disk full")
	p := &fakeJobPersister{loadErr: boom, saveErr: boom}

	// Load 失败不得 panic，也不得阻止服务启动：任务清单是辅助信息，
	// 存储故障时降级为纯内存（与「存储不可用硬失败」的账号存储不同）。
	r := NewJobRegistryWithPersister(p)
	defer r.Stop()

	_, cancel := context.WithCancel(context.Background())
	job := r.Create(1, cancel)
	job.Finish(JobResult{Migrated: 1}, JobStatusDone)

	got := r.List()
	if len(got) != 1 || got[0].Status != JobStatusDone {
		t.Fatalf("list = %+v, want 1 done record despite persister failure", got)
	}
}

func TestJobRegistryListIsCopy(t *testing.T) {
	p := &fakeJobPersister{}
	r := NewJobRegistryWithPersister(p)
	defer r.Stop()
	_, cancel := context.WithCancel(context.Background())
	r.Create(2, cancel)

	got := r.List()
	got[0].Status = "tampered"
	got[0].Result.FailKeys = append(got[0].Result.FailKeys, "x")

	// 调用方（handler 序列化）不得改到注册表内部状态。
	again := r.List()
	if again[0].Status == "tampered" {
		t.Error("List leaked internal record")
	}
}

func TestJobRecordJSONShape(t *testing.T) {
	// 落盘 JSON 字段名是跨版本兼容面：改名会让旧文件读不出来。
	b, err := json.Marshal(JobRecord{ID: "x", Status: JobStatusRunning, Total: 1})
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"id", "created", "total", "status", "progress", "result"} {
		if _, ok := m[k]; !ok {
			t.Errorf("missing json field %q in %s", k, b)
		}
	}
}
