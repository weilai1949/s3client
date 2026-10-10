package service

// persist_metrics_test.go —— 计划 / 任务清单落盘失败可观测（KNOWN_ISSUES #83）。

import (
	"errors"
	"testing"
	"time"
)

// TestPersistFailureCountObservesSaveErrors 两条持久化路径的 Save 失败都必须让
// PersistFailureCount 增长，同时不改「内存态保真」的降级行为。
func TestPersistFailureCountObservesSaveErrors(t *testing.T) {
	before := PersistFailureCount()

	// 计划清单落盘失败。
	sp := &fakeSchedulePersister{saveErr: errors.New("disk full")}
	sc := NewScheduler(sp, newFakeTrigger().run)
	t.Cleanup(sc.Stop)
	withFixedNow(t, time.Date(2026, 10, 8, 10, 0, 0, 0, time.UTC))
	if _, err := sc.Create(newValidSchedule("")); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got := PersistFailureCount(); got < before+1 {
		t.Fatalf("计划落盘失败未计数: %d, want >= %d", got, before+1)
	}
	if got := sc.List(); len(got) != 1 {
		t.Fatalf("落盘失败不得丢内存态: %v", got)
	}

	// 任务清单落盘失败。
	jp := &fakeJobPersister{saveErr: errors.New("disk full")}
	r := NewJobRegistryWithPersister(jp)
	t.Cleanup(r.Stop)
	if _, err := r.TryCreate(1, nil); err != nil {
		t.Fatalf("TryCreate: %v", err)
	}
	if got := PersistFailureCount(); got < before+2 {
		t.Fatalf("任务落盘失败未计数: %d, want >= %d", got, before+2)
	}
}
