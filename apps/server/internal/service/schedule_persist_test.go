package service

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
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
