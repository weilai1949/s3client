package service

import (
	"errors"
	"sync"
	"testing"
	"time"
)

// ---- 测试替身 ----

// fakeTrigger 记录触发的计划 id，可按计划注入错误（模拟账号失效 / 任务在跑）。
type fakeTrigger struct {
	mu     sync.Mutex
	calls  []Schedule
	errs   map[string]error
	jobSeq int
}

func newFakeTrigger() *fakeTrigger {
	return &fakeTrigger{errs: map[string]error{}}
}

// run 满足 ScheduleTrigger：出错时按契约返回空 jobID。
func (f *fakeTrigger) run(s Schedule) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, s)
	if err := f.errs[s.ID]; err != nil {
		return "", err
	}
	f.jobSeq++
	return "job-" + s.ID, nil
}

func (f *fakeTrigger) calledIDs() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, 0, len(f.calls))
	for _, s := range f.calls {
		out = append(out, s.ID)
	}
	return out
}

// fakeSchedulePersister 内存落盘替身（镜像 fakeJobPersister；Save 可能被并发调用，需加锁）。
type fakeSchedulePersister struct {
	mu      sync.Mutex
	loaded  []Schedule
	loadErr error
	saves   [][]Schedule
	saveErr error
}

func (f *fakeSchedulePersister) Load() ([]Schedule, error) {
	if f.loadErr != nil {
		return nil, f.loadErr
	}
	return f.loaded, nil
}

func (f *fakeSchedulePersister) Save(recs []Schedule) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	cp := make([]Schedule, len(recs))
	copy(cp, recs)
	f.saves = append(f.saves, cp)
	return f.saveErr
}

// withFixedNow 把 scheduleNow 钉到固定时刻（测试专用，t.Cleanup 还原）。
func withFixedNow(t *testing.T, now time.Time) {
	t.Helper()
	restore := setScheduleNow(func() time.Time { return now })
	t.Cleanup(restore)
}

// ---- Create / List / Get ----

func TestSchedulerCreatePersistsAndComputesNextRun(t *testing.T) {
	sc, _, p := newTestScheduler(t)
	fixed := time.Date(2026, 10, 8, 10, 30, 0, 0, time.UTC)
	withFixedNow(t, fixed)

	s := newValidSchedule("")
	created, err := sc.Create(s)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == "" {
		t.Error("ID 未生成")
	}
	if !created.CreatedAt.Equal(fixed) {
		t.Errorf("CreatedAt = %v, want %v", created.CreatedAt, fixed)
	}
	// cron = `0 2 * * *`，固定时刻 10:30 → 下一次是次日 02:00。
	wantNext := time.Date(2026, 10, 9, 2, 0, 0, 0, time.UTC)
	if !created.NextRunAt.Equal(wantNext) {
		t.Errorf("NextRunAt = %v, want %v", created.NextRunAt, wantNext)
	}
	// 创建即落盘：进程若在此后崩溃，重启必须能看到计划。
	if len(p.saves) != 1 || len(p.saves[0]) != 1 {
		t.Fatalf("saves = %d/%v, want 1 次含 1 条", len(p.saves), p.saves)
	}
}

func TestSchedulerCreateValidates(t *testing.T) {
	sc, _, p := newTestScheduler(t)
	s := newValidSchedule("")
	s.SourceBucket = ""
	if _, err := sc.Create(s); err == nil {
		t.Fatal("缺 sourceBucket 必须拒绝")
	}
	s = newValidSchedule("")
	s.Cron = "61 * * * *"
	if _, err := sc.Create(s); err == nil {
		t.Fatal("非法 cron 必须拒绝")
	}
	if len(p.saves) != 0 {
		t.Errorf("校验失败不得落盘: %d", len(p.saves))
	}
}

func TestSchedulerListOrderAndCreateIDUniqueness(t *testing.T) {
	sc, _, _ := newTestScheduler(t)
	withFixedNow(t, time.Date(2026, 10, 8, 10, 0, 0, 0, time.UTC))
	a, err := sc.Create(newValidSchedule(""))
	if err != nil {
		t.Fatal(err)
	}
	withFixedNow(t, time.Date(2026, 10, 8, 11, 0, 0, 0, time.UTC))
	b, err := sc.Create(newValidSchedule(""))
	if err != nil {
		t.Fatal(err)
	}
	if a.ID == b.ID {
		t.Fatalf("ID 必须唯一: %s", a.ID)
	}
	got := sc.List()
	if len(got) != 2 || got[0].ID != b.ID || got[1].ID != a.ID {
		t.Fatalf("List 顺序 = %v, want 最新在前 [%s %s]", got, b.ID, a.ID)
	}
	// 返回值是副本：篡改不得影响注册表内部状态。
	got[0].Cron = "tampered"
	if again := sc.List(); again[0].Cron == "tampered" {
		t.Error("List 泄漏内部状态")
	}
}

// ---- Update ----

func TestSchedulerUpdate(t *testing.T) {
	fixed := time.Date(2026, 10, 8, 10, 30, 0, 0, time.UTC)
	t.Run("cron 不变且 nextRun 未过期则保留排期", func(t *testing.T) {
		sc, _, _ := newTestScheduler(t)
		withFixedNow(t, fixed)
		created, err := sc.Create(newValidSchedule(""))
		if err != nil {
			t.Fatal(err)
		}
		in := newValidSchedule("")
		in.TargetPrefix = "new-backup/"
		updated, err := sc.Update(created.ID, in)
		if err != nil {
			t.Fatalf("Update: %v", err)
		}
		if updated.TargetPrefix != "new-backup/" {
			t.Errorf("TargetPrefix = %q", updated.TargetPrefix)
		}
		if !updated.NextRunAt.Equal(created.NextRunAt) {
			t.Errorf("NextRunAt 被无谓重算: %v → %v", created.NextRunAt, updated.NextRunAt)
		}
		if !updated.CreatedAt.Equal(created.CreatedAt) {
			t.Errorf("CreatedAt 被改写: %v", updated.CreatedAt)
		}
		if updated.ID != created.ID {
			t.Errorf("ID 被改写: %q", updated.ID)
		}
	})

	t.Run("cron 变更则重算排期", func(t *testing.T) {
		sc, _, _ := newTestScheduler(t)
		withFixedNow(t, fixed)
		created, err := sc.Create(newValidSchedule(""))
		if err != nil {
			t.Fatal(err)
		}
		in := newValidSchedule("")
		in.Cron = "30 3 * * *" // 次日 03:30
		updated, err := sc.Update(created.ID, in)
		if err != nil {
			t.Fatal(err)
		}
		want := time.Date(2026, 10, 9, 3, 30, 0, 0, time.UTC)
		if !updated.NextRunAt.Equal(want) {
			t.Errorf("NextRunAt = %v, want %v", updated.NextRunAt, want)
		}
	})

	t.Run("nextRun 已过期则重算（经用户操作不产生补跑）", func(t *testing.T) {
		sc, _, p := newTestScheduler(t)
		withFixedNow(t, fixed)
		created, err := sc.Create(newValidSchedule(""))
		if err != nil {
			t.Fatal(err)
		}
		// 模拟「停用期间排期过期」：直接推进 scheduleNow 再更新。
		later := fixed.Add(48 * time.Hour)
		withFixedNow(t, later)
		in := newValidSchedule("")
		in.TargetPrefix = "x/"
		updated, err := sc.Update(created.ID, in)
		if err != nil {
			t.Fatal(err)
		}
		if !updated.NextRunAt.After(later) {
			t.Errorf("NextRunAt = %v 不晚于 now=%v（过期排期必须重算）", updated.NextRunAt, later)
		}
		if len(p.saves) < 2 {
			t.Errorf("Update 必须落盘: saves = %d", len(p.saves))
		}
	})

	t.Run("更新清除历史 LastError（修复后重新出发）", func(t *testing.T) {
		sc, trig, _ := newTestScheduler(t)
		withFixedNow(t, fixed)
		created, err := sc.Create(newValidSchedule(""))
		if err != nil {
			t.Fatal(err)
		}
		trig.errs[created.ID] = errors.New("source account not found")
		withFixedNow(t, fixed.Add(time.Hour))
		if _, err := sc.RunNow(created.ID); err == nil {
			t.Fatal("注入错误应冒泡")
		}
		in := newValidSchedule("")
		updated, err := sc.Update(created.ID, in)
		if err != nil {
			t.Fatal(err)
		}
		if updated.LastError != "" {
			t.Errorf("LastError = %q, want 空（更新后清除）", updated.LastError)
		}
	})

	t.Run("未知 id 报 ErrScheduleNotFound", func(t *testing.T) {
		sc, _, _ := newTestScheduler(t)
		if _, err := sc.Update("nope", newValidSchedule("")); !errors.Is(err, ErrScheduleNotFound) {
			t.Errorf("err = %v, want ErrScheduleNotFound", err)
		}
	})
}

// ---- Delete ----

func TestSchedulerDelete(t *testing.T) {
	sc, _, p := newTestScheduler(t)
	withFixedNow(t, time.Date(2026, 10, 8, 10, 0, 0, 0, time.UTC))
	created, err := sc.Create(newValidSchedule(""))
	if err != nil {
		t.Fatal(err)
	}
	if err := sc.Delete(created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if got := sc.List(); len(got) != 0 {
		t.Errorf("删除后 List = %v, want 空", got)
	}
	saves := len(p.saves)
	if err := sc.Delete("nope"); !errors.Is(err, ErrScheduleNotFound) {
		t.Errorf("重复删除 err = %v, want ErrScheduleNotFound", err)
	}
	if len(p.saves) != saves {
		t.Error("删除失败不得落盘")
	}
}

// ---- Load / 降级 ----

func TestSchedulerLoadDegradesAndRestores(t *testing.T) {
	t.Run("load 失败降级为空且可继续使用", func(t *testing.T) {
		p := &fakeSchedulePersister{loadErr: errors.New("disk")}
		sc := NewScheduler(p, newFakeTrigger().run)
		t.Cleanup(sc.Stop)
		if got := sc.List(); len(got) != 0 {
			t.Errorf("List = %v, want 空", got)
		}
		// 降级后仍可创建（纯内存继续服务）。
		withFixedNow(t, time.Date(2026, 10, 8, 10, 0, 0, 0, time.UTC))
		if _, err := sc.Create(newValidSchedule("")); err != nil {
			t.Errorf("降级后 Create: %v", err)
		}
	})

	t.Run("恢复保留存储字段且跳过空 id", func(t *testing.T) {
		stored := newValidSchedule("kept")
		stored.LastRunAt = time.Date(2026, 10, 7, 2, 0, 0, 0, time.UTC)
		stored.LastError = "previous failure"
		p := &fakeSchedulePersister{loaded: []Schedule{{}, stored}}
		sc := NewScheduler(p, newFakeTrigger().run)
		t.Cleanup(sc.Stop)

		got := sc.List()
		if len(got) != 1 || got[0].ID != "kept" {
			t.Fatalf("List = %+v, want 仅 kept", got)
		}
		if !got[0].LastRunAt.Equal(stored.LastRunAt) || got[0].LastError != "previous failure" {
			t.Errorf("恢复丢失运行态: %+v", got[0])
		}
	})

	t.Run("save 失败不影响内存态", func(t *testing.T) {
		p := &fakeSchedulePersister{saveErr: errors.New("disk full")}
		sc := NewScheduler(p, newFakeTrigger().run)
		t.Cleanup(sc.Stop)
		withFixedNow(t, time.Date(2026, 10, 8, 10, 0, 0, 0, time.UTC))
		if _, err := sc.Create(newValidSchedule("")); err != nil {
			t.Errorf("Create: %v", err)
		}
		if got := sc.List(); len(got) != 1 {
			t.Errorf("List = %v, want 1（落盘失败不得丢内存态）", got)
		}
	})
}

// ---- Tick：到点触发 / 补跑一次 / 不叠加 ----

func newTestScheduler(t *testing.T) (*Scheduler, *fakeTrigger, *fakeSchedulePersister) {
	t.Helper()
	fp := &fakeSchedulePersister{}
	trig := newFakeTrigger()
	sc := NewScheduler(fp, trig.run)
	t.Cleanup(sc.Stop)
	return sc, trig, fp
}

func TestSchedulerTickFiresDueSchedule(t *testing.T) {
	sc, trig, p := newTestScheduler(t)
	fixed := time.Date(2026, 10, 8, 10, 0, 0, 0, time.UTC)
	withFixedNow(t, fixed)
	created, err := sc.Create(newValidSchedule(""))
	if err != nil {
		t.Fatal(err)
	}
	// 到点时刻：次日 02:00。
	fireAt := time.Date(2026, 10, 9, 2, 0, 1, 0, time.UTC)
	saves := len(p.saves)
	sc.Tick(fireAt)

	if ids := trig.calledIDs(); len(ids) != 1 || ids[0] != created.ID {
		t.Fatalf("触发 = %v, want [%s]", ids, created.ID)
	}
	got := sc.List()[0]
	if !got.LastRunAt.Equal(fireAt) {
		t.Errorf("LastRunAt = %v, want %v", got.LastRunAt, fireAt)
	}
	if got.LastJobID != "job-"+created.ID {
		t.Errorf("LastJobID = %q", got.LastJobID)
	}
	if got.LastError != "" {
		t.Errorf("LastError = %q, want 空", got.LastError)
	}
	// 触发后必须前移到未来时刻，否则下一 tick 会重复触发。
	if !got.NextRunAt.After(fireAt) {
		t.Errorf("NextRunAt = %v 不晚于触发时刻（会重复触发）", got.NextRunAt)
	}
	if len(p.saves) <= saves {
		t.Error("触发后必须落盘")
	}
}

func TestSchedulerTickSkipsNotDueAndDisabled(t *testing.T) {
	sc, trig, _ := newTestScheduler(t)
	fixed := time.Date(2026, 10, 8, 10, 0, 0, 0, time.UTC)
	withFixedNow(t, fixed)
	due, err := sc.Create(newValidSchedule(""))
	if err != nil {
		t.Fatal(err)
	}
	// 未到期：nextRun 在未来。
	sc.Tick(fixed)
	if ids := trig.calledIDs(); len(ids) != 0 {
		t.Fatalf("未到期被触发: %v", ids)
	}

	disabled := newValidSchedule("")
	disabled.Enabled = false
	disabledS, err := sc.Create(disabled)
	if err != nil {
		t.Fatal(err)
	}
	// 到点但停用：不触发。
	sc.Tick(time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC))
	for _, id := range trig.calledIDs() {
		if id == disabledS.ID {
			t.Error("停用计划不得自动触发")
		}
	}
	// 未到期计划仍在清单中（Tick 不移除，只跳过触发）。
	if got := sc.List(); len(got) != 2 || got[0].ID != due.ID && got[1].ID != due.ID {
		t.Errorf("List 未含未到期计划 %s: %+v", due.ID, got)
	}
}

func TestSchedulerTickCatchUpOnceAfterDowntime(t *testing.T) {
	sc, trig, _ := newTestScheduler(t)
	fixed := time.Date(2026, 10, 8, 10, 0, 0, 0, time.UTC)
	withFixedNow(t, fixed)
	created, err := sc.Create(newValidSchedule("")) // next = 10-09 02:00
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == "" {
		t.Fatal("Create 未分配 id")
	}
	// 停机 3 天后恢复：补跑一次，且直接排到未来，不逐次补 3 个槽。
	resume := time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC)
	sc.Tick(resume)

	if ids := trig.calledIDs(); len(ids) != 1 {
		t.Fatalf("触发次数 = %d, want 1（补跑一次，不逐次补）: %v", len(ids), ids)
	}
	got := sc.List()[0]
	if !got.NextRunAt.After(resume) {
		t.Errorf("NextRunAt = %v 不晚于恢复时刻", got.NextRunAt)
	}
	// 第二次 tick 不再触发（已排到未来）。
	sc.Tick(resume.Add(time.Second))
	if ids := trig.calledIDs(); len(ids) != 1 {
		t.Errorf("补跑后重复触发: %v", ids)
	}
}

func TestSchedulerTickTriggerErrorRecordsLastError(t *testing.T) {
	sc, trig, _ := newTestScheduler(t)
	fixed := time.Date(2026, 10, 8, 10, 0, 0, 0, time.UTC)
	withFixedNow(t, fixed)
	created, err := sc.Create(newValidSchedule(""))
	if err != nil {
		t.Fatal(err)
	}
	trig.errs[created.ID] = errors.New("source account not found")

	fireAt := time.Date(2026, 10, 9, 2, 0, 1, 0, time.UTC)
	sc.Tick(fireAt)

	got := sc.List()[0]
	if got.LastError != "source account not found" {
		t.Errorf("LastError = %q", got.LastError)
	}
	if got.LastJobID != "" {
		t.Errorf("失败的尝试不得记录 jobID: %q", got.LastJobID)
	}
	if !got.LastRunAt.Equal(fireAt) {
		t.Errorf("LastRunAt = %v, want %v（尝试即记录）", got.LastRunAt, fireAt)
	}
	// 失败同样前移排期：坏配置不得每 tick 重试轰炸。
	if !got.NextRunAt.After(fireAt) {
		t.Errorf("NextRunAt = %v 不前移（会每 tick 重试）", got.NextRunAt)
	}
}

func TestSchedulerTickRunningSkipDoesNotRecord(t *testing.T) {
	sc, trig, _ := newTestScheduler(t)
	fixed := time.Date(2026, 10, 8, 10, 0, 0, 0, time.UTC)
	withFixedNow(t, fixed)
	created, err := sc.Create(newValidSchedule(""))
	if err != nil {
		t.Fatal(err)
	}
	trig.errs[created.ID] = ErrScheduleRunning

	fireAt := time.Date(2026, 10, 9, 2, 0, 1, 0, time.UTC)
	sc.Tick(fireAt)

	got := sc.List()[0]
	// 上一轮还在跑：不覆盖运行态（LastRunAt 保持零），排期仍前移避免堆积。
	if !got.LastRunAt.IsZero() || got.LastError != "" {
		t.Errorf("在跑跳过不得记录: lastRunAt=%v lastError=%q", got.LastRunAt, got.LastError)
	}
	if !got.NextRunAt.After(fireAt) {
		t.Errorf("NextRunAt = %v 不前移（会每 tick 撞车）", got.NextRunAt)
	}
}

func TestSchedulerTickInvalidStoredCronStops(t *testing.T) {
	// 手改 schedules.json 塞入非法 cron：不得 panic，且排期清零停摆（不再触发）。
	p := &fakeSchedulePersister{loaded: []Schedule{{
		ID: "hand-edit", Enabled: true,
		Cron:      "61 * * * *",
		NextRunAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
	}}}
	trig := newFakeTrigger()
	sc := NewScheduler(p, trig.run)
	t.Cleanup(sc.Stop)

	sc.Tick(time.Date(2026, 10, 9, 2, 0, 0, 0, time.UTC))
	if ids := trig.calledIDs(); len(ids) != 0 {
		t.Errorf("非法 cron 不得触发: %v", ids)
	}
	got := sc.List()[0]
	if !got.NextRunAt.IsZero() {
		t.Errorf("NextRunAt = %v, want 零值停摆", got.NextRunAt)
	}
	// 停摆原因必须落 LastError（KNOWN_ISSUES #83），而不是静默不动；且要落盘。
	if got.LastError == "" {
		t.Error("非法 cron 停摆未记录 LastError")
	}
	if len(p.saves) == 0 || p.saves[0][0].LastError != got.LastError {
		t.Errorf("停摆状态未落盘: saves=%v lastError=%q", p.saves, got.LastError)
	}
}

// ---- RunNow：手动触发不改排期 ----

func TestSchedulerRunNow(t *testing.T) {
	sc, trig, _ := newTestScheduler(t)
	fixed := time.Date(2026, 10, 8, 10, 0, 0, 0, time.UTC)
	withFixedNow(t, fixed)
	created, err := sc.Create(newValidSchedule(""))
	if err != nil {
		t.Fatal(err)
	}

	jobID, err := sc.RunNow(created.ID)
	if err != nil {
		t.Fatalf("RunNow: %v", err)
	}
	if jobID != "job-"+created.ID {
		t.Errorf("jobID = %q", jobID)
	}
	got := sc.List()[0]
	if !got.LastRunAt.Equal(fixed) {
		t.Errorf("LastRunAt = %v, want %v", got.LastRunAt, fixed)
	}
	// 手动触发不得挪动自动排期。
	wantNext := time.Date(2026, 10, 9, 2, 0, 0, 0, time.UTC)
	if !got.NextRunAt.Equal(wantNext) {
		t.Errorf("手动触发改动了 NextRunAt: %v, want %v", got.NextRunAt, wantNext)
	}
	if ids := trig.calledIDs(); len(ids) != 1 {
		t.Errorf("触发 = %v", ids)
	}

	if _, err := sc.RunNow("nope"); !errors.Is(err, ErrScheduleNotFound) {
		t.Errorf("未知 id err = %v, want ErrScheduleNotFound", err)
	}
}

func TestSchedulerRunNowPropagatesRunningAndRecordsError(t *testing.T) {
	sc, trig, _ := newTestScheduler(t)
	fixed := time.Date(2026, 10, 8, 10, 0, 0, 0, time.UTC)
	withFixedNow(t, fixed)
	created, err := sc.Create(newValidSchedule(""))
	if err != nil {
		t.Fatal(err)
	}

	// 已有任务在跑 → ErrScheduleRunning 冒泡且不记录。
	trig.errs[created.ID] = ErrScheduleRunning
	if _, err := sc.RunNow(created.ID); !errors.Is(err, ErrScheduleRunning) {
		t.Fatalf("err = %v, want ErrScheduleRunning", err)
	}
	if got := sc.List()[0]; !got.LastRunAt.IsZero() {
		t.Errorf("在跑时不得记录 LastRunAt: %v", got.LastRunAt)
	}

	// 其它错误 → 记录 LastError。
	trig.errs[created.ID] = errors.New("invalid source account configuration")
	if _, err := sc.RunNow(created.ID); err == nil {
		t.Fatal("错误应冒泡")
	}
	if got := sc.List()[0]; got.LastError != "invalid source account configuration" {
		t.Errorf("LastError = %q", got.LastError)
	}
}

// ---- 循环 / Stop ----

func TestSchedulerLoopFiresAndStopStops(t *testing.T) {
	orig := scheduleTickEvery
	scheduleTickEvery = 5 * time.Millisecond
	t.Cleanup(func() { scheduleTickEvery = orig })

	p := &fakeSchedulePersister{}
	trig := newFakeTrigger()
	sc := NewScheduler(p, trig.run)
	withFixedNow(t, time.Date(2026, 10, 8, 10, 0, 0, 0, time.UTC))
	created, err := sc.Create(newValidSchedule("")) // next = 次日 02:00
	if err != nil {
		t.Fatal(err)
	}
	// 推进「现在」到次日 02:00 之后，让循环 tick 判定到期。
	later := time.Date(2026, 10, 9, 3, 0, 0, 0, time.UTC)
	withFixedNow(t, later)

	deadline := time.After(2 * time.Second)
	for {
		if ids := trig.calledIDs(); len(ids) == 1 && ids[0] == created.ID {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("循环未在时限内触发: %v", trig.calledIDs())
		default:
			time.Sleep(2 * time.Millisecond)
		}
	}

	// Stop 之后：循环不再产生新触发（残留 in-flight 的不计）。
	sc.Stop()
	sc.Stop() // 幂等
	before := len(trig.calledIDs())
	time.Sleep(20 * time.Millisecond)
	if after := len(trig.calledIDs()); after != before {
		t.Errorf("Stop 后仍触发: %d → %d", before, after)
	}
}

// ---- 并发（-race）----

func TestSchedulerConcurrentAccess(t *testing.T) {
	sc, _, _ := newTestScheduler(t)
	withFixedNow(t, time.Date(2026, 10, 8, 10, 0, 0, 0, time.UTC))
	created, err := sc.Create(newValidSchedule(""))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 9, 3, 0, 0, 0, time.UTC)

	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sc.Tick(now)
			_, _ = sc.RunNow(created.ID)
			_ = sc.List()
		}()
	}
	wg.Wait()
}

// ---- 无 persister 纯内存模式（handler.New 的默认构造）----

func TestSchedulerWithoutPersisterStaysInMemory(t *testing.T) {
	sc := NewScheduler(nil, newFakeTrigger().run)
	t.Cleanup(sc.Stop)
	withFixedNow(t, time.Date(2026, 10, 8, 10, 0, 0, 0, time.UTC))

	// restore/persist 的 nil 分支必须可走：不 panic、状态保留在内存。
	created, err := sc.Create(newValidSchedule(""))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got := sc.List(); len(got) != 1 || got[0].ID != created.ID {
		t.Fatalf("List = %+v, want 1 条内存计划", got)
	}
	// 触发同样只记内存，不触碰落盘。
	jobID, err := sc.RunNow(created.ID)
	if err != nil || jobID == "" {
		t.Fatalf("RunNow = %q, %v", jobID, err)
	}
}

// TestSchedulerUpdateRejectsInvalid 输入非法时 Update 必须拒绝且不改动既有计划。
func TestSchedulerUpdateRejectsInvalid(t *testing.T) {
	sc, _, _ := newTestScheduler(t)
	withFixedNow(t, time.Date(2026, 10, 8, 10, 0, 0, 0, time.UTC))
	created, err := sc.Create(newValidSchedule(""))
	if err != nil {
		t.Fatal(err)
	}
	bad := newValidSchedule("")
	bad.Cron = "61 * * * *"
	if _, err := sc.Update(created.ID, bad); err == nil {
		t.Fatal("非法 cron 的 Update 必须报错")
	}
	// 既有计划不得被半程修改。
	got := sc.List()
	if len(got) != 1 || got[0].Cron != created.Cron {
		t.Errorf("失败的 Update 泄漏了改动: %+v", got)
	}
}

// TestSchedulerGet 覆盖 Get 的命中/未命中（生产引用为 scope.go 的计划桶注入）。
func TestSchedulerGet(t *testing.T) {
	sc, _, _ := newTestScheduler(t)
	withFixedNow(t, time.Date(2026, 10, 8, 10, 0, 0, 0, time.UTC))
	created, err := sc.Create(newValidSchedule(""))
	if err != nil {
		t.Fatal(err)
	}
	got, ok := sc.Get(created.ID)
	if !ok || got.ID != created.ID || got.Cron != created.Cron {
		t.Fatalf("Get = %+v, ok=%v", got, ok)
	}
	// 返回副本：篡改不影响注册表内部状态。
	got.Cron = "tampered"
	again, _ := sc.Get(created.ID)
	if again.Cron != created.Cron {
		t.Errorf("Get 泄漏内部状态: %q", again.Cron)
	}
	if _, ok := sc.Get("nope"); ok {
		t.Error("不存在的 id 应返回 false")
	}
}
