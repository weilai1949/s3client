package service

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
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
	origMarshal := jobMarshal
	t.Cleanup(func() { jobMarshal = origMarshal })

	boom := errors.New("boom")
	rec := []JobRecord{{ID: "j", Status: JobStatusRunning}}

	t.Run("write fails", func(t *testing.T) {
		// 父目录不存在 → 原子写在创建临时文件阶段失败，必须冒泡而非静默丢更新。
		missing := filepath.Join(t.TempDir(), "no-such-dir", "j.json")
		if err := NewFileJobPersister(missing).Save(rec); err == nil {
			t.Fatal("Save into missing dir must fail")
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
		// 目标路径是已存在的目录 → rename 阶段失败，必须报错并清掉临时文件
		//（否则下次 Load 可能读到 .tmp 半成品）。
		dir := t.TempDir()
		path := filepath.Join(dir, "j.json")
		if err := os.Mkdir(path, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := NewFileJobPersister(path).Save(rec); err == nil {
			t.Fatal("Save onto existing dir must fail")
		}
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

// TestJobResultUnmarshalAcceptsLegacyFailKeys 落盘兼容：新格式写 `failedKeys`（公共契约），
// 旧 jobs.json 里的 `failKeys` 仍须能读出，避免升级后历史任务的失败清单丢失。
func TestJobResultUnmarshalAcceptsLegacyFailKeys(t *testing.T) {
	var legacy JobResult
	if err := json.Unmarshal([]byte(`{"migrated":1,"failed":2,"lastError":"old","failKeys":["old.txt"]}`), &legacy); err != nil {
		t.Fatalf("unmarshal legacy failKeys: %v", err)
	}
	if len(legacy.FailKeys) != 1 || legacy.FailKeys[0] != "old.txt" {
		t.Fatalf("legacy failKeys 未兼容读出: %+v", legacy)
	}
	// JSON 名保持历史契约 `lastError`（Go 字段已更正为 FirstError，review Nit）。
	if legacy.FirstError != "old" {
		t.Fatalf("FirstError = %q, want old（lastError 落盘名必须仍能读出）", legacy.FirstError)
	}

	var current JobResult
	if err := json.Unmarshal([]byte(`{"migrated":1,"failed":2,"failedKeys":["new.txt"]}`), &current); err != nil {
		t.Fatalf("unmarshal failedKeys: %v", err)
	}
	if len(current.FailKeys) != 1 || current.FailKeys[0] != "new.txt" {
		t.Fatalf("failedKeys 未读出: %+v", current)
	}

	var bad JobResult
	if err := json.Unmarshal([]byte(`{"migrated":"not-an-int"}`), &bad); err == nil {
		t.Fatal("字段类型错误的 JSON 必须返回错误，不得静默吞掉")
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

// ---- 落盘乱序（评审 R5）----

// blockingJobPersister 被 arm 后的首次 Save 阻塞至放行，用于制造「先快照者后落盘」
// 的乱序窗口；disk 记录最后一次**完成**的 Save（= 重启后读到的状态）。
type blockingJobPersister struct {
	mu      sync.Mutex
	armed   bool
	started bool
	entered chan struct{}
	release chan struct{}
	disk    []JobRecord
}

func (p *blockingJobPersister) Load() ([]JobRecord, error) { return nil, nil }

// arm 让下一次 Save 进入阻塞分支。
func (p *blockingJobPersister) arm() {
	p.mu.Lock()
	p.armed = true
	p.mu.Unlock()
}

func (p *blockingJobPersister) Save(recs []JobRecord) error {
	p.mu.Lock()
	if p.armed && !p.started {
		p.started = true
		p.mu.Unlock()
		close(p.entered)
		<-p.release
	} else {
		p.mu.Unlock()
	}
	p.mu.Lock()
	p.disk = append([]JobRecord(nil), recs...)
	p.mu.Unlock()
	return nil
}

// diskSnapshot 返回最后一次完成的 Save 的快照副本。
func (p *blockingJobPersister) diskSnapshot() []JobRecord {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]JobRecord(nil), p.disk...)
}

// TestJobRegistryPersistKeepsNewestSnapshot（评审 R5）：终态落盘与后续落盘不串行时，
// 「先快照者后落盘」会把磁盘覆盖回旧状态——重启后 done 任务被 restore 标回
// interrupted，触发虚假对账告警。行为级断言：并发落盘全部完成后，磁盘必须是最终态
// （job1 终态 + job2 在册）。
func TestJobRegistryPersistKeepsNewestSnapshot(t *testing.T) {
	p := &blockingJobPersister{entered: make(chan struct{}), release: make(chan struct{})}
	reg := NewJobRegistryWithPersister(p)
	t.Cleanup(reg.Stop)

	// 首次落盘不 arm（建 job1 时正常完成），随后 arm 把 job1.Finish 的落盘拦住。
	job1, err := reg.TryCreate(10, func() {})
	if err != nil {
		t.Fatalf("TryCreate job1: %v", err)
	}
	p.arm()
	g1done := make(chan struct{})
	go func() {
		defer close(g1done)
		job1.Finish(JobResult{Migrated: 1}, JobStatusDone)
	}()
	select {
	case <-p.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("job1.Finish 未进入落盘阻塞点")
	}

	// G2 = 并发创建 job2：旧实现下其 Save 先完成（含 job2 的最终态），随后 job1 的
	// 放行落盘会把磁盘覆盖回「只有 job1、且以先快照为准」的旧态；新实现下它在
	// 落盘互斥上等待，放行后才快照 + 落盘。
	g2done := make(chan struct{})
	go func() {
		defer close(g2done)
		if _, cerr := reg.TryCreate(5, func() {}); cerr != nil {
			t.Errorf("TryCreate job2: %v", cerr)
		}
	}()
	select {
	case <-g2done:
	case <-time.After(100 * time.Millisecond):
	}

	close(p.release)
	for _, d := range []chan struct{}{g1done, g2done} {
		select {
		case <-d:
		case <-time.After(5 * time.Second):
			t.Fatal("落盘协程未在放行后返回（落盘互斥死锁？）")
		}
	}

	disk := p.diskSnapshot()
	if len(disk) != 2 {
		t.Fatalf("磁盘终态条目数 = %d, want 2（旧快照后落盘覆盖了新状态）: %+v", len(disk), disk)
	}
	var job1Rec *JobRecord
	for i := range disk {
		if disk[i].ID == job1.ID {
			job1Rec = &disk[i]
		}
	}
	if job1Rec == nil {
		t.Fatalf("磁盘终态缺 job1: %+v", disk)
	}
	if job1Rec.Result.Migrated != 1 || !IsTerminalJobStatus(job1Rec.Status) {
		t.Errorf("磁盘上 job1 非终态：status=%q result=%+v", job1Rec.Status, job1Rec.Result)
	}
}
