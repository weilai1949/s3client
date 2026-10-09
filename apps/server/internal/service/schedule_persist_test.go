package service

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// newValidSchedule 构造一个通过校验的计划（测试公共夹具）。
func newValidSchedule(id string) Schedule {
	now := time.Date(2026, 10, 8, 10, 0, 0, 0, time.UTC)
	return Schedule{
		ID:              id,
		SourceAccountID: "src-acc",
		SourceBucket:    "src-bucket",
		SourcePrefix:    "data/",
		TargetAccountID: "dst-acc",
		TargetBucket:    "dst-bucket",
		TargetPrefix:    "backup/",
		Mode:            CompareETag,
		Cron:            "0 2 * * *",
		Enabled:         true,
		CreatedAt:       now,
		NextRunAt:       time.Date(2026, 10, 9, 2, 0, 0, 0, time.UTC),
	}
}

// ---- Schedule.Validate ----

func TestScheduleValidate(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		mutate  func(*Schedule)
		wantSub string
	}{
		{"缺源账号", func(s *Schedule) { s.SourceAccountID = "" }, "sourceAccountId"},
		{"缺源桶", func(s *Schedule) { s.SourceBucket = "" }, "sourceBucket"},
		{"缺目标账号", func(s *Schedule) { s.TargetAccountID = "" }, "targetAccountId"},
		{"缺目标桶", func(s *Schedule) { s.TargetBucket = "" }, "targetBucket"},
		{"缺 cron", func(s *Schedule) { s.Cron = "" }, "cron"},
		{"cron 非法", func(s *Schedule) { s.Cron = "61 * * * *" }, "cron"},
		{"mode 非法", func(s *Schedule) { s.Mode = "weird" }, "mode"},
		{"永不触发的 cron", func(s *Schedule) { s.Cron = "0 0 30 2 *" }, "never"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := newValidSchedule("s1")
			tc.mutate(&s)
			err := s.Validate(time.Now())
			if err == nil {
				t.Fatal("Validate = nil, want error")
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Errorf("err = %q, want substring %q", err, tc.wantSub)
			}
		})
	}

	t.Run("合法通过", func(t *testing.T) {
		t.Parallel()
		s := newValidSchedule("s1")
		if err := s.Validate(time.Now()); err != nil {
			t.Errorf("Validate = %v, want nil", err)
		}
	})

	t.Run("空 mode 视为 etag 默认值", func(t *testing.T) {
		t.Parallel()
		s := newValidSchedule("s1")
		s.Mode = ""
		if err := s.Validate(time.Now()); err != nil {
			t.Errorf("空 mode 应默认通过（与 syncHandler 一致），err = %v", err)
		}
	})
}

// ---- FileSchedulePersister ----

func TestFileSchedulePersisterRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "schedules.json")
	p := NewFileSchedulePersister(path)

	// 首次启动：无文件 = 空清单，而非错误。
	got, err := p.Load()
	if err != nil || got != nil {
		t.Fatalf("load missing = %v, %v; want nil, nil", got, err)
	}

	want := []Schedule{newValidSchedule("s1")}
	if err := p.Save(want); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err = p.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(got) != 1 || got[0].ID != "s1" || got[0].Cron != "0 2 * * *" {
		t.Fatalf("load = %+v, want 1 schedule s1", got)
	}
	if !got[0].NextRunAt.Equal(want[0].NextRunAt) {
		t.Errorf("nextRunAt = %v, want %v", got[0].NextRunAt, want[0].NextRunAt)
	}

	// 0600：计划清单含账号/桶布局（非密钥，但同 jobs.json 口径收紧）。
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("perm = %o, want 600", perm)
	}
}

func TestFileSchedulePersisterLoadEmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "schedules.json")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := NewFileSchedulePersister(path).Load()
	if err != nil || got != nil {
		t.Fatalf("load empty = %v, %v; want nil, nil", got, err)
	}
}

func TestFileSchedulePersisterLoadErrors(t *testing.T) {
	t.Run("corrupt json", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "schedules.json")
		if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := NewFileSchedulePersister(path).Load(); err == nil {
			t.Fatal("want parse error")
		}
	})
	t.Run("unreadable path", func(t *testing.T) {
		if _, err := NewFileSchedulePersister(t.TempDir()).Load(); err == nil {
			t.Fatal("want read error（目录当文件）")
		}
	})
}

func TestFileSchedulePersisterSaveErrors(t *testing.T) {
	orig := scheduleMarshal
	t.Cleanup(func() { scheduleMarshal = orig })

	t.Run("write fails", func(t *testing.T) {
		missing := filepath.Join(t.TempDir(), "no-such-dir", "s.json")
		if err := NewFileSchedulePersister(missing).Save([]Schedule{newValidSchedule("s")}); err == nil {
			t.Fatal("Save into missing dir must fail")
		}
	})
	t.Run("marshal fails", func(t *testing.T) {
		boom := errors.New("boom")
		scheduleMarshal = func(any) ([]byte, error) { return nil, boom }
		if err := NewFileSchedulePersister(filepath.Join(t.TempDir(), "s.json")).Save(nil); !errors.Is(err, boom) {
			t.Fatalf("err = %v, want boom", err)
		}
	})
	t.Run("rename fails and removes temp file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "s.json")
		if err := os.Mkdir(path, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := NewFileSchedulePersister(path).Save([]Schedule{newValidSchedule("s")}); err == nil {
			t.Fatal("Save onto existing dir must fail")
		}
		if _, err := os.Stat(path + ".tmp"); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("temp file left behind: %v", err)
		}
	})
}

// TestScheduleJSONShape 落盘 / API JSON 字段名是跨版本契约面，改名即破坏既有文件。
func TestScheduleJSONShape(t *testing.T) {
	b, err := json.Marshal(newValidSchedule("x"))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{
		"id", "sourceAccountId", "sourceBucket", "sourcePrefix",
		"targetAccountId", "targetBucket", "targetPrefix",
		"mode", "cron", "enabled", "createdAt", "nextRunAt",
	} {
		if _, ok := m[k]; !ok {
			t.Errorf("missing json field %q in %s", k, b)
		}
	}
	// lastRunAt / lastJobId 从未运行时省略（omitzero/omitempty），运行后出现。
	if _, ok := m["lastRunAt"]; ok {
		t.Errorf("lastRunAt 不应出现在未运行的计划里: %s", b)
	}
}

// ---- 落盘乱序（评审 R5）----

// blockingSchedulePersister 被 arm 后的首次 Save 阻塞至放行，用于制造
// 「先快照者后落盘」的乱序窗口；disk 记录最后一次**完成**的 Save（= 重启后读到的状态）。
type blockingSchedulePersister struct {
	mu      sync.Mutex
	armed   bool
	started bool
	entered chan struct{}
	release chan struct{}
	disk    []Schedule
}

func (p *blockingSchedulePersister) Load() ([]Schedule, error) { return nil, nil }

// arm 让下一次 Save 进入阻塞分支。
func (p *blockingSchedulePersister) arm() {
	p.mu.Lock()
	p.armed = true
	p.mu.Unlock()
}

func (p *blockingSchedulePersister) Save(recs []Schedule) error {
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
	p.disk = append([]Schedule(nil), recs...)
	p.mu.Unlock()
	return nil
}

// diskSnapshot 返回最后一次完成的 Save 的快照副本。
func (p *blockingSchedulePersister) diskSnapshot() []Schedule {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]Schedule(nil), p.disk...)
}

// TestSchedulerPersistKeepsNewestSnapshot（评审 R5）：两次并发 persist 若「快照」
// 与「Save」不作为整体串行，后快照者先落盘、先快照者后落盘——磁盘最终停在旧状态
// （重启回退：丢编辑 / 丢新增）。行为级断言：并发落盘全部完成后，磁盘必须是最终态
// （A 的编辑已保存且 B 在册）。
func TestSchedulerPersistKeepsNewestSnapshot(t *testing.T) {
	p := &blockingSchedulePersister{entered: make(chan struct{}), release: make(chan struct{})}
	sc := NewScheduler(p, func(s Schedule) (string, error) { return "", nil })
	t.Cleanup(sc.Stop)

	created, err := sc.Create(newValidSchedule(""))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// G1 = 编辑 A：其 persist 被 arm 拦住（快照 A 旧值，落盘未完成）。
	p.arm()
	g1done := make(chan struct{})
	go func() {
		defer close(g1done)
		in := newValidSchedule("")
		in.SourceBucket = "edited-bucket"
		if _, uerr := sc.Update(created.ID, in); uerr != nil {
			t.Errorf("Update: %v", uerr)
		}
	}()
	select {
	case <-p.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("G1 未进入落盘阻塞点")
	}

	// G2 = 新增 B：旧实现下它的 Save 会先完成（记录最终态），随后 G1 放行把磁盘
	// 覆盖回旧态；新实现下它在落盘互斥上等待，放行后才快照 + 落盘。
	g2done := make(chan struct{})
	go func() {
		defer close(g2done)
		if _, cerr := sc.Create(newValidSchedule("")); cerr != nil {
			t.Errorf("Create B: %v", cerr)
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
	var gotA *Schedule
	seenB := false
	for i := range disk {
		switch disk[i].ID {
		case created.ID:
			gotA = &disk[i]
		default:
			seenB = true
		}
	}
	if !seenB || gotA == nil {
		t.Fatalf("磁盘终态缺条目：A=%v B=%v（disk=%+v）", gotA != nil, seenB, disk)
	}
	if gotA.SourceBucket != "edited-bucket" {
		t.Errorf("磁盘上 A 的编辑丢失（旧快照后落盘覆盖新状态）：sourceBucket=%q, want edited-bucket",
			gotA.SourceBucket)
	}
}
