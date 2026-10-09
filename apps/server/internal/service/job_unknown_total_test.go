package service

import (
	"context"
	"testing"
)

// TestJobLearnsUnknownTotalFromProgress 计划同步任务创建时总数未知（total=0）：
// SyncKeys 列举出源侧对象数后经首帧携带 Total，Job 必须从进度帧学习总数——
// 否则 record() 与 Finish 终帧会把 0 当总数回显（「5/0」进度、中断恢复后 0 总数）。
//
// 只在「创建时未知（0）」时学习：已知总数的任务其进度帧 Total 与创建值恒等，
// 不写入 ⇒ 与 migrateAsync 等调用方对 job.Total 的无锁读（响应体回显）无并发写，
// 不引入数据竞态。
func TestJobLearnsUnknownTotalFromProgress(t *testing.T) {
	r := NewJobRegistry()
	defer r.Stop()
	_, cancel := context.WithCancel(context.Background())

	job, err := r.TryCreate(0, cancel)
	if err != nil {
		t.Fatalf("TryCreate: %v", err)
	}
	if job.Total != 0 {
		t.Fatalf("Total = %d, want 0（创建时未知）", job.Total)
	}

	// 首个携带正总数的帧：学习为任务总数（record 可见）。
	job.Emit(JobProgress{Done: 0, Total: 100, Status: JobStatusRunning})
	if got := r.List()[0].Total; got != 100 {
		t.Fatalf("学习后 record.Total = %d, want 100", got)
	}

	// Total=0 的帧（空列举等）不得把已学习的总数抹掉。
	job.Emit(JobProgress{Done: 0, Total: 0, Status: JobStatusRunning})
	if got := r.List()[0].Total; got != 100 {
		t.Errorf("零值帧抹掉了总数: %d, want 100", got)
	}

	// 已学习后不再改写（首学为准）：后续帧携带不同总数不覆盖。
	job.Emit(JobProgress{Done: 40, Total: 40, Migrated: 40, Status: JobStatusRunning})
	if got := r.List()[0].Total; got != 100 {
		t.Errorf("后续帧改写了总数: %d, want 100（首学为准）", got)
	}

	// 终帧必须用学习到的总数（Finish 从 j.Total 取 Done/Total）。
	job.Finish(JobResult{Migrated: 40}, JobStatusDone)
	ch, _ := job.Subscribe()
	final := <-ch
	if final.Total != 100 || final.Done != 100 {
		t.Errorf("终帧 = %+v, want Done/Total = 100/100（学习到的总数）", final)
	}
	if final.Migrated != 40 {
		t.Errorf("终帧 migrated = %d, want 40", final.Migrated)
	}

	// 落盘记录同样携带学习到的总数（中断恢复后「done/总数」才正确）。
	if got := r.List()[0].Total; got != 100 {
		t.Errorf("落盘 record.Total = %d, want 100", got)
	}
}

// TestJobKnownTotalNotOverwrittenByFrame 已知总数的任务不得被进度帧改写总数——
// 保证 migrateAsync / copy 等既有路径对 job.Total 的无锁读与 Emit 写入永不并发。
func TestJobKnownTotalNotOverwrittenByFrame(t *testing.T) {
	r := NewJobRegistry()
	defer r.Stop()
	_, cancel := context.WithCancel(context.Background())

	job, err := r.TryCreate(2, cancel)
	if err != nil {
		t.Fatalf("TryCreate: %v", err)
	}
	job.Emit(JobProgress{Done: 1, Total: 999, Status: JobStatusRunning})
	if got := r.List()[0].Total; got != 2 {
		t.Errorf("record.Total = %d, want 2（已知总数不被帧改写）", got)
	}
}
