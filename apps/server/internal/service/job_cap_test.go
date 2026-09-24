package service

import (
	"context"
	"errors"
	"testing"
)

// fillUntilFull 持续创建在册任务直到触发上限，返回成功创建数。
//
// 不把 defaultMaxJobs 写死：断言的是「存在上限」而非具体数值，上限值可演进；
// 安全上界（4096）让「上限意外消失」表现为显式失败而不是挂死。
// 顺带断言 TryCreate 契约：拒绝时返回 nil job。
func fillUntilFull(t *testing.T, r *JobRegistry) int {
	t.Helper()
	_, cancel := context.WithCancel(context.Background())
	for n := 0; n <= 4096; n++ {
		j, err := r.TryCreate(1, cancel)
		if err != nil {
			if !errors.Is(err, ErrTooManyJobs) {
				t.Fatalf("fill: unexpected err %v", err)
			}
			if j != nil {
				t.Errorf("job = %+v, want nil on rejection", j)
			}
			return n
		}
	}
	t.Fatal("在册任务必须有上限（创建 4097 个仍未拒绝，请复核 defaultMaxJobs 与安全上界）")
	return -1
}

// TestJobRegistryTryCreateRejectsBeyondCap JobRegistry 必须有总任务上限：
// 每个任务都持有 goroutine、订阅者与落盘条目，无上限时短时间大量请求可耗尽资源
// （todolist #17 / ASSESSMENT M4）。
func TestJobRegistryTryCreateRejectsBeyondCap(t *testing.T) {
	r := NewJobRegistry()
	defer r.Stop()

	accepted := fillUntilFull(t, r)
	// 拒绝不得留下副作用（jobs 数量不变）。
	if got := len(r.List()); got != accepted {
		t.Errorf("jobs = %d, want %d (rejected job must not be registered)", got, accepted)
	}
}

// TestJobRegistryTryCreateAllowsAfterTerminal 任务完成后应能再创建：
// 上限约束的是「在册任务数」，不能因为历史任务堆积而永久拒绝新任务。
func TestJobRegistryTryCreateAllowsAfterTerminal(t *testing.T) {
	r := NewJobRegistry()
	defer r.Stop()
	_, cancel := context.WithCancel(context.Background())

	first, err := r.TryCreate(1, cancel)
	if err != nil {
		t.Fatal(err)
	}
	fillUntilFull(t, r) // 填满剩余名额：此后任何创建都必须被拒

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
// 值无关口径：带「恢复的中断记录」与不带的注册表，能接受的进行中任务数必须相同
// （若中断记录占用名额，前者会少接受一个）。
func TestJobRegistryCapCountsRunningOnly(t *testing.T) {
	acceptUntilFull := func(loaded []JobRecord) int {
		p := &fakeJobPersister{loaded: loaded}
		r := NewJobRegistryWithPersister(p)
		defer r.Stop()
		return fillUntilFull(t, r)
	}
	// restore 会把仍为 running 的历史记录标记为 interrupted（终态）。
	interrupted := []JobRecord{{ID: "old-interrupted", Total: 1, Status: JobStatusRunning}}
	if with, without := acceptUntilFull(interrupted), acceptUntilFull(nil); with != without {
		t.Errorf("接受数：带恢复中断记录 %d，不带 %d——恢复的中断任务不得占用在册名额", with, without)
	}
}

// TestJobSubscribeCap 单任务的并发订阅数必须有上限：Finish 对每个订阅者都要投递
// （带 finishSendTimeout），订阅数无界时终态关闭的最坏耗时随订阅数线性增长，
// 而 SSE 路由的 withStreamLimit 只管「全局并发流」，管不住同一任务被反复订阅。
func TestJobSubscribeCap(t *testing.T) {
	r := NewJobRegistry()
	defer r.Stop()
	j := r.Create(1, nil)

	n := 0
	for {
		ch, ok := j.Subscribe()
		if !ok {
			break
		}
		if ch == nil {
			t.Fatal("ok=true 时必须返回可用 channel")
		}
		n++
		if n > 1024 {
			t.Fatal("Subscribe 无上限：已接受 1024 个订阅者")
		}
	}
	if n != maxSubscribersPerJob {
		t.Errorf("订阅上限 = %d, want %d", n, maxSubscribersPerJob)
	}

	// 终态任务不受上限约束：重连/刷新必须仍能拿到终态快照（否则订阅位被占满后永远读不到结果）。
	j.Finish(JobResult{}, JobStatusDone)
	ch, ok := j.Subscribe()
	if !ok || ch == nil {
		t.Fatal("终态任务必须仍可订阅（不受上限约束）")
	}
	p, open := <-ch
	if !open || p.Status != JobStatusDone {
		t.Fatalf("终态快照 = %+v open=%v, want done", p, open)
	}
}
