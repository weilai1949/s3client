package service

// scheduler.go —— 计划任务调度循环（ROADMAP §三 #6）。
//
// 职责：持有计划清单（内存 + schedules.json 落盘）、按 NextRunAt 到点触发、
// 维护运行态（LastRunAt / LastJobID / LastError）。触发本身委托注入的
// ScheduleTrigger（handler 层负责解析账号 → 建异步任务 → 跑 SyncKeys），
// 保持 service 不依赖 store / handler 的分层。
//
// 三条核心语义（各有测试钉住）：
//  1. **补跑一次**：进程停机期间错过的多个槽，恢复后只补跑一次并直接排到未来
//     （`NextRunAt <= now` 判定 + 触发前先前移），不逐次补——避免恢复瞬间洪水；
//  2. **不叠加**：trigger 返回 ErrScheduleRunning（上一轮未结束）时跳过本轮，
//     不覆盖运行态、不产生新任务，但排期照常前移（否则每 tick 撞车）；
//  3. **失败即前移**：触发失败（账号失效等）记录 LastError 后同样前移排期，
//     坏配置不会每 tick 重试轰炸；LastError 在下次成功或用户更新计划时清除。

import (
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ErrScheduleNotFound 表示按 id 未找到计划。
var ErrScheduleNotFound = errors.New("schedule not found")

// ErrScheduleRunning 表示该计划上一轮执行尚未结束（不叠加）。
// 由 trigger 返回；调度循环据此跳过本轮，HTTP 层映射为 409。
var ErrScheduleRunning = errors.New("schedule run already in progress")

// ScheduleTrigger 触发一次计划执行，返回异步任务 id。
//
// 契约：
//   - 出错时必须返回空 jobID（失败的尝试不得记录任务 id）；
//   - 返回 ErrScheduleRunning 表示上一轮未结束，调用方（调度循环）跳过本轮；
//   - 非阻塞：任务创建后立即返回，执行进度经 JobRegistry / SSE 观测。
type ScheduleTrigger func(s Schedule) (jobID string, err error)

// scheduleNow 是调度器的「当前时刻」来源，测试可替换以钉住时间。
// 替换可能与运行中的 loop 协程并发，故经 RWMutex 保护（测试 helper 见
// scheduler_test.go withFixedNow / setScheduleNow）。
var (
	scheduleNowMu sync.RWMutex
	scheduleNowFn = time.Now
)

// scheduleNow 返回当前时刻（可被测试替换）。
func scheduleNow() time.Time {
	scheduleNowMu.RLock()
	f := scheduleNowFn
	scheduleNowMu.RUnlock()
	return f()
}

// setScheduleNow 替换时刻来源并返回还原函数（测试用；生产不调用）。
func setScheduleNow(f func() time.Time) func() {
	scheduleNowMu.Lock()
	prev := scheduleNowFn
	scheduleNowFn = f
	scheduleNowMu.Unlock()
	return func() { setScheduleNow(prev) }
}

// scheduleTickEvery 是调度循环的评估间隔：30 秒。
// cron 粒度为 1 分钟，30s 评估保证到点后 ≤60s 内触发（评估在 0s 或 30s 两个相位）。
// 暴露为包级变量供测试加速（同 reapInterval）。
var scheduleTickEvery = 30 * time.Second

// Scheduler 计划注册表（清单 + 落盘 + 调度循环）。
type Scheduler struct {
	mu        sync.Mutex
	schedules map[string]*Schedule
	persister SchedulePersister
	trigger   ScheduleTrigger
	stopCh    chan struct{}
	once      sync.Once
	// persistMu 串行化「快照 + Save」整体（评审 R5，防旧快照后落盘覆盖新状态）；
	// 锁序 persistMu → mu，见 persist 注释。
	persistMu sync.Mutex
}

// NewScheduler 构造调度器：恢复既有清单（Load 失败降级为空，不阻塞启动）、
// 启动调度循环。persister 为 nil 时纯内存（同 JobRegistry 的 setter 模式）。
func NewScheduler(p SchedulePersister, trigger ScheduleTrigger) *Scheduler {
	sc := &Scheduler{
		schedules: make(map[string]*Schedule),
		persister: p,
		trigger:   trigger,
		stopCh:    make(chan struct{}),
	}
	sc.restore()
	// 间隔在创建协程**前**于调用方 goroutine 捕获：测试改写 scheduleTickEvery 与
	// 本读取同在主 goroutine 串行，loop 协程内不再触碰共享变量（-race 口径）。
	go sc.loop(scheduleTickEvery)
	return sc
}

// restore 载入历史清单；Load 失败静默降级（计划清单是辅助信息，同 jobs.json 口径）。
// 空 id 的损坏条目跳过（无法寻址，保留会污染 map key）。
func (sc *Scheduler) restore() {
	if sc.persister == nil {
		return
	}
	recs, err := sc.persister.Load()
	if err != nil {
		return
	}
	for _, rec := range recs {
		if rec.ID == "" {
			continue
		}
		cp := rec
		sc.schedules[cp.ID] = &cp
	}
}

// persist 串行化「快照 + Save」后回写整份清单；失败静默降级（下一次状态变更会再试）。
//
// 落盘互斥 persistMu（评审 R5）：快照与 Save 必须作为**一个整体**串行——否则两次
// 并发 persist 可能「后快照者先落盘、先快照者后落盘」，旧快照把磁盘覆盖回旧状态
// （重启回退：丢编辑 / 丢新增，见 TestSchedulerPersistKeepsNewestSnapshot）。
// sc.mu 仍只包住快照生成——落盘 IO 不拖长与 Get/Update 的内存互斥窗口。
// 锁序：persistMu → sc.mu（persist 内部），任何路径不得在持有 sc.mu 时取 persistMu；
// 调用方**不得**已持有 sc.mu（否则死锁，历史不变式不变）。
func (sc *Scheduler) persist() {
	if sc.persister == nil {
		return
	}
	sc.persistMu.Lock()
	defer sc.persistMu.Unlock()
	sc.mu.Lock()
	recs := sc.listLocked()
	sc.mu.Unlock()
	_ = sc.persister.Save(recs)
}

// listLocked 生成快照（调用方须持有 sc.mu）。
func (sc *Scheduler) listLocked() []Schedule {
	out := make([]Schedule, 0, len(sc.schedules))
	for _, s := range sc.schedules {
		out = append(out, *s)
	}
	return out
}

// List 返回计划清单快照（创建时间倒序，最新在前）。
func (sc *Scheduler) List() []Schedule {
	sc.mu.Lock()
	out := sc.listLocked()
	sc.mu.Unlock()

	// 与 JobRegistry.List 同序：最新在前，同刻按 id 稳定。
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			a, b := out[i], out[j]
			if a.CreatedAt.Before(b.CreatedAt) ||
				(a.CreatedAt.Equal(b.CreatedAt) && a.ID > b.ID) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

// Get 按 id 取计划快照；不存在返回 false。
// 生产引用：handler/scope.go 对 schedules/{id} 路径注入计划桶引用（前缀作用域判定）。
func (sc *Scheduler) Get(id string) (Schedule, bool) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	s, ok := sc.schedules[id]
	if !ok {
		return Schedule{}, false
	}
	return *s, true
}

// Create 校验、赋 id / 时间 / 排期并落盘。
func (sc *Scheduler) Create(s Schedule) (Schedule, error) {
	now := scheduleNow()
	if err := s.Validate(now); err != nil {
		return Schedule{}, err
	}
	s.ID = uuid.NewString()
	s.CreatedAt = now
	s.NextRunAt = s.ComputeNext(now)

	sc.mu.Lock()
	sc.schedules[s.ID] = &s
	out := s
	sc.mu.Unlock()
	sc.persist()
	return out, nil
}

// Update 整体替换计划内容（保留 id / CreatedAt / 运行态）。
//
// 排期规则：cron 变更或已过期 → 重算；否则保留原排期（避免无谓编辑把即将
// 到点的备份推迟）。更新清除 LastError——修复配置后从干净状态重新出发。
func (sc *Scheduler) Update(id string, in Schedule) (Schedule, error) {
	now := scheduleNow()
	if err := in.Validate(now); err != nil {
		return Schedule{}, err
	}
	sc.mu.Lock()
	old, ok := sc.schedules[id]
	if !ok {
		sc.mu.Unlock()
		return Schedule{}, ErrScheduleNotFound
	}
	merged := in
	merged.ID = old.ID
	merged.CreatedAt = old.CreatedAt
	merged.LastRunAt = old.LastRunAt
	merged.LastJobID = old.LastJobID
	merged.LastError = ""
	merged.NextRunAt = old.NextRunAt
	if in.Cron != old.Cron || merged.NextRunAt.IsZero() || !merged.NextRunAt.After(now) {
		merged.NextRunAt = merged.ComputeNext(now)
	}
	sc.schedules[id] = &merged
	out := merged
	sc.mu.Unlock()
	sc.persist()
	return out, nil
}

// Delete 移除计划并落盘；不存在报 ErrScheduleNotFound。
func (sc *Scheduler) Delete(id string) error {
	sc.mu.Lock()
	if _, ok := sc.schedules[id]; !ok {
		sc.mu.Unlock()
		return ErrScheduleNotFound
	}
	delete(sc.schedules, id)
	sc.mu.Unlock()
	sc.persist()
	return nil
}

// RunNow 立即触发一次（手动执行）：不改动自动排期，只记录运行态。
// ErrScheduleRunning / 其它 trigger 错误原样冒泡给 HTTP 层映射状态码。
func (sc *Scheduler) RunNow(id string) (string, error) {
	sc.mu.Lock()
	s, ok := sc.schedules[id]
	if !ok {
		sc.mu.Unlock()
		return "", ErrScheduleNotFound
	}
	snapshot := *s
	sc.mu.Unlock()

	jobID, err := sc.trigger(snapshot)
	switch {
	case err == nil:
		sc.recordRun(id, scheduleNow(), jobID, "")
		return jobID, nil
	case errors.Is(err, ErrScheduleRunning):
		// 上一轮未结束：不记录（RunNow 被拒不是一次执行），原样冒泡给 HTTP 层。
		return "", err
	default:
		// 非 running 的失败：记录原因供 UI 展示；jobID 契约上必为空。
		sc.recordRun(id, scheduleNow(), "", err.Error())
		return "", err
	}
}

// recordRun 写入一次尝试的运行态并落盘；记录时计划可能已被并发删除（跳过）。
func (sc *Scheduler) recordRun(id string, at time.Time, jobID, lastErr string) {
	sc.mu.Lock()
	s, ok := sc.schedules[id]
	if ok {
		s.LastRunAt = at
		s.LastJobID = jobID
		s.LastError = lastErr
	}
	sc.mu.Unlock()
	if ok {
		sc.persist()
	}
}

// Tick 评估并触发所有到点计划（导出供循环与测试共用同一入口）。
// now 由调用方给出（循环用 time.Now，测试钉时刻），保证判定可复现。
func (sc *Scheduler) Tick(now time.Time) {
	type due struct {
		id string
		s  Schedule
	}
	var dues []due

	sc.mu.Lock()
	for id, s := range sc.schedules {
		if !s.Enabled || s.NextRunAt.IsZero() || s.NextRunAt.After(now) {
			continue
		}
		// 触发前先算好下一次：结构性非法（手改文件塞坏 cron）→ 排期清零停摆。
		next := s.ComputeNext(now)
		if next.IsZero() {
			s.NextRunAt = time.Time{}
			continue
		}
		s.NextRunAt = next
		dues = append(dues, due{id: id, s: *s})
	}
	advanced := len(dues) > 0
	sc.mu.Unlock()
	// 落盘在锁外做（persist 内部自会加锁生成快照）。
	if advanced {
		sc.persist()
	}

	for _, d := range dues {
		jobID, err := sc.trigger(d.s)
		switch {
		case err == nil:
			sc.recordRun(d.id, now, jobID, "")
		case errors.Is(err, ErrScheduleRunning):
			// 上一轮未结束：不叠加、不覆盖运行态（排期已在上方前移）。
		default:
			sc.recordRun(d.id, now, "", err.Error())
		}
	}
}

// loop 评估循环；间隔由 NewScheduler 捕获传入（见彼处注释）。
func (sc *Scheduler) loop(every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			sc.Tick(scheduleNow())
		case <-sc.stopCh:
			return
		}
	}
}

// Stop 停止调度循环（幂等；不取消已在跑的任务——那是 JobRegistry 的职责）。
func (sc *Scheduler) Stop() {
	sc.once.Do(func() { close(sc.stopCh) })
}
