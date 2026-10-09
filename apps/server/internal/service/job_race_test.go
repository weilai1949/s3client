package service

// job_race_test.go —— docs/archive/review-2026-09-19.md §B7：Emit 必须在锁内投递。
//
// 旧实现先在锁内快照订阅者、解锁后再发送；Finish 会在同一把锁下清空订阅表、
// 解锁后 close 这些 channel。两者之间没有任何互斥 → 一旦 Emit 与 Finish 并发，
// Emit 就可能对已 close 的 channel 发送 → panic（进程不 recover，直接退出）。
//
// 复现方式：让 Emit 停在落盘钩子（此时它已锁内投递完毕、正要回写进度），并发调用
// Finish 关闭订阅通道，再放行 Emit —— 若投递不在锁内（§B7 之前的旧实现），Finish
// 的 close 与 Emit 的发送无互斥，必然 send on closed channel。本测试因此同时守护
// 「投递锁内」的结构性保证与「Emit 落盘期间并发 Finish 不 panic、订阅者收到终态帧」。

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// gatePersister 只在被 arm 的那一次 Save 上阻塞，用于把 Emit 卡在解锁之后、投递之前。
type gatePersister struct {
	armed   atomic.Bool
	entered chan struct{}
	release chan struct{}
}

func (p *gatePersister) Save([]JobRecord) error {
	if p.armed.CompareAndSwap(true, false) {
		close(p.entered)
		<-p.release
	}
	return nil
}

// Load 是 JobPersister 的恢复入口；本测试不涉及恢复。
func (p *gatePersister) Load() ([]JobRecord, error) { return nil, nil }

// TestJobEmitFinishConcurrentDoesNotPanic Emit 与 Finish 并发时不得 panic，
// 且订阅者必须在收到终态帧后看到通道关闭。
func TestJobEmitFinishConcurrentDoesNotPanic(t *testing.T) {
	old := jobProgressPersistEvery
	jobProgressPersistEvery = 0 // 让每次 Emit 都走落盘钩子
	t.Cleanup(func() { jobProgressPersistEvery = old })

	p := &gatePersister{entered: make(chan struct{}), release: make(chan struct{})}
	reg := NewJobRegistryWithPersister(p)
	t.Cleanup(reg.Stop)
	job, err := reg.TryCreate(10, func() {})
	if err != nil {
		t.Fatalf("TryCreate: %v", err)
	}
	ch, ok := job.Subscribe()
	if !ok {
		t.Fatal("Subscribe rejected")
	}

	p.armed.Store(true)
	emitDone := make(chan struct{})
	go func() {
		defer close(emitDone)
		job.Emit(JobProgress{Done: 1, Total: 10, Status: JobStatusRunning})
	}()

	select {
	case <-p.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("Emit 未进入落盘钩子")
	}
	// Emit 此刻停在落盘钩子里（投递已在锁内完成），Finish 与其并发执行：清空
	// 订阅表、投递终态。Finish 放到独立 goroutine——落盘串行化（评审 R5 的
	// persistMu）后，Finish 的终态落盘会等 Emit 的 Save 放行，主线程直调会与
	// 下方 close(release) 互相等待而死锁。等 100ms：实现不串行时 Finish 早已
	// 全程跑完（窗口 = 与 park 中的 Emit 并发）；串行时它在 persistMu 上等待，
	// 由下方放行解除——两种情况都不死锁。
	finishDone := make(chan struct{})
	go func() {
		defer close(finishDone)
		job.Finish(JobResult{Migrated: 1}, JobStatusDone)
	}()
	select {
	case <-finishDone:
	case <-time.After(100 * time.Millisecond):
	}
	close(p.release)
	<-emitDone
	<-finishDone

	sawFinal := false
	for prog := range ch {
		if prog.Status == JobStatusDone {
			sawFinal = true
		}
	}
	if !sawFinal {
		t.Fatal("订阅者未收到终态帧")
	}
}

// TestJobEmitFinishConcurrentStress 高并发反复交错，覆盖 Emit/Finish/Subscribe 的竞态。
func TestJobEmitFinishConcurrentStress(t *testing.T) {
	reg := NewJobRegistry()
	t.Cleanup(reg.Stop)
	for round := 0; round < 20; round++ {
		job, err := reg.TryCreate(10, func() {})
		if err != nil {
			t.Fatalf("TryCreate: %v", err)
		}
		ch, ok := job.Subscribe()
		if !ok {
			t.Fatal("Subscribe rejected")
		}
		// 消费者必须与 Emit/Finish 并发跑：Finish 的终态投递会等订阅者腾出缓冲，
		// 先 wg.Wait() 再读通道等于让 Finish 走满 5s 超时。
		consumed := make(chan struct{})
		go func() {
			defer close(consumed)
			for range ch {
			}
		}()

		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				job.Emit(JobProgress{Done: i, Total: 10, Status: JobStatusRunning})
			}
		}()
		go func() {
			defer wg.Done()
			job.Finish(JobResult{Migrated: 1}, JobStatusDone)
		}()
		wg.Wait()
		select {
		case <-consumed: // Finish 关闭了订阅通道，消费者随之退出
		case <-time.After(2 * time.Second):
			t.Fatalf("round %d: 订阅通道未被关闭", round)
		}
	}
}
