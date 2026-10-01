package service

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	JobTTL            = 30 * time.Minute
	JobTimeout        = 2 * time.Hour
	SSEHeartbeatEvery = 15 * time.Second

	// JobInterruptedTTL 是「重启中断」任务的保留期。这类任务是待对账证据
	// （例如移动任务中复制成功但源未删除），保留 7 天便于用户发现并处理。
	JobInterruptedTTL = 7 * 24 * time.Hour
)

// 中间进度落盘节流间隔：每帧都写盘会让批量任务的 IO 放大到不可接受，
// 但完全依赖终态落盘又会让「重启时进度」退回 0。折中为固定间隔采样。
// 暴露为包级变量以便测试快速触发；生产保持默认值。
var jobProgressPersistEvery = 2 * time.Second

// JobProgress 异步任务进度（SSE/轮询 JSON 形状：migrated）。
type JobProgress struct {
	Done     int    `json:"done"`
	Total    int    `json:"total"`
	Migrated int    `json:"migrated"`
	Failed   int    `json:"failed"`
	Key      string `json:"key,omitempty"`
	Error    string `json:"error,omitempty"`
	Status   string `json:"status,omitempty"`
}

// JobResult 异步任务终态汇总。
// JSON 标签是落盘格式与公共契约的一部分：对外统一 `failedKeys`（OpenAPI / 前端 /
// docs/api.md）；`job_persist.go` 的 UnmarshalJSON 兼容旧落盘格式里的 `failKeys`。
type JobResult struct {
	Migrated int `json:"migrated"`
	Failed   int `json:"failed"`
	// FirstError 是首个失败的错误（仅在空时写入，永远取第一条；旧名 LastError
	// 名不副实，review Nit）。JSON 名保持历史契约 `lastError` 不变。
	FirstError string   `json:"lastError,omitempty"`
	FailKeys   []string `json:"failedKeys,omitempty"`
}

// Job 单次异步批量任务。
type Job struct {
	ID      string
	Created time.Time
	Total   int

	mu       sync.Mutex
	progress JobProgress
	result   JobResult
	done     bool
	// finishedAt 是进入终态（Finish/恢复标记 interrupted）的时刻，Reap 的 TTL
	// 从它起算而非 Created：跑超 30 分钟的任务若按 Created 计时，Finish 后
	// ≤5 分钟即被清掉——列表丢终态、轮询 404、jobs.json 记录消失（review R8）。
	// 零值表示未记录（旧落盘记录），Reap 回退用 Created。
	finishedAt time.Time
	cancel     context.CancelFunc
	subs       map[chan JobProgress]struct{}

	// persist 由注册表注入（nil = 纯内存）；lastSave 用于节流中间进度落盘。
	persist  func()
	lastSave time.Time
}

// JobRegistry 任务注册表（reap + 关停取消 + 可选落盘 + 在册上限）。
type JobRegistry struct {
	mu        sync.Mutex
	jobs      map[string]*Job
	stopCh    chan struct{}
	once      sync.Once
	persister JobPersister
	maxJobs   int
}

// defaultMaxJobs 是在册任务上限：每个任务持有 goroutine、SSE 订阅与落盘条目，
// 无上限时短时间内的大量异步请求可耗尽内存与 goroutine（KNOWN_ISSUES #17 / ASSESSMENT M4）。
const defaultMaxJobs = 256

// NewJobRegistry 创建纯内存注册表（不落盘，与历史行为一致）并启动 reap 循环。
func NewJobRegistry() *JobRegistry {
	return NewJobRegistryWithPersister(nil)
}

// NewJobRegistryWithPersister 创建注册表并恢复既有任务清单（persister 为 nil 时纯内存）。
//
// 恢复语义：上次进程退出时仍在 running 的任务不可能继续执行，一律标记为 interrupted，
// 使「复制成功但源未删除」这类半途中断的移动任务在重启后仍可被前端看到并对账
// （ASSESSMENT S1 / KNOWN_ISSUES #19）。已完成任务的终态原样保留。
//
// 任务清单属于辅助信息：Load/Save 失败只降级为内存态，不影响服务启动
// （与账号存储「不可用则硬失败」的取舍不同，见 ADR-002）。
//
// 在册上限恒为 defaultMaxJobs、不可配置：原 RegistryOption / WithMaxJobs 只被测试
// 引用（生产零引用），已按死代码纪律删除；测试需要「在册满」语义时填到上限即可
// （见 job_cap_test.go 的 fillUntilFull）。
func NewJobRegistryWithPersister(p JobPersister) *JobRegistry {
	r := &JobRegistry{
		jobs:      make(map[string]*Job),
		stopCh:    make(chan struct{}),
		persister: p,
		maxJobs:   defaultMaxJobs,
	}
	r.restore()
	go r.reapLoop()
	return r
}

// restore 载入历史任务清单；未完成任务标记为 interrupted 并回写。
func (r *JobRegistry) restore() {
	if r.persister == nil {
		return
	}
	recs, err := r.persister.Load()
	if err != nil {
		return // 降级：清单损坏/不可读不应阻止启动
	}
	changed := false
	for _, rec := range recs {
		if rec.ID == "" {
			continue
		}
		status := rec.Status
		if !IsTerminalJobStatus(status) {
			status = JobStatusInterrupted
			changed = true
		}
		progress := rec.Progress
		progress.Total = rec.Total
		progress.Status = status
		r.jobs[rec.ID] = &Job{
			ID:       rec.ID,
			Created:  rec.Created,
			Total:    rec.Total,
			progress: progress,
			result:   rec.Result,
			// 旧落盘记录无 finishedAt（零值），Reap 会回退用 Created 计 TTL。
			finishedAt: rec.FinishedAt,
			done:       true, // 重启后不可能再推进，直接视为终态
			subs:       make(map[chan JobProgress]struct{}),
		}
	}
	if changed {
		r.persistJobs()
	}
}

// persistJobs 回写整份清单；失败静默降级（下一次状态变更会再试）。
func (r *JobRegistry) persistJobs() {
	if r.persister == nil {
		return
	}
	_ = r.persister.Save(r.List())
}

// List 返回任务清单快照（按创建时间倒序，最新在前），供 API/前端展示未完成任务。
func (r *JobRegistry) List() []JobRecord {
	r.mu.Lock()
	recs := make([]JobRecord, 0, len(r.jobs))
	for _, j := range r.jobs {
		recs = append(recs, j.record())
	}
	r.mu.Unlock()

	sort.Slice(recs, func(i, k int) bool {
		if !recs[i].Created.Equal(recs[k].Created) {
			return recs[i].Created.After(recs[k].Created)
		}
		return recs[i].ID < recs[k].ID // 同刻创建时保证顺序稳定
	})
	return recs
}

// record 生成任务的可序列化快照（含 result 内切片的深拷贝，避免泄露内部状态）。
//
// 快照状态不得为空：JobProgress.Status 是 omitempty 的可选字段，进度帧允许不带状态
// （service.Progress.Status 同样可选），但 JobRecord.Status 是落盘与恢复的判据——
// 空状态会被 restore 当成「非终态」，把任务误标为 interrupted（触发人工对账告警）。
// 状态缺失只可能是「运行中任务发过无状态帧」：Finish 恒写入终态（空 status 补
// "done"），且 Emit 对已终结任务直接丢帧（review R9），故 done ⇒ Status 必非空，
// 兜底只需按 running 补齐，不存在「done 且空状态」的分支。
func (j *Job) record() JobRecord {
	j.mu.Lock()
	defer j.mu.Unlock()
	status := j.progress.Status
	if status == "" {
		status = JobStatusRunning
	}
	result := j.result
	if j.result.FailKeys != nil {
		result.FailKeys = append([]string(nil), j.result.FailKeys...)
	}
	progress := j.progress
	progress.Status = status
	return JobRecord{
		ID: j.ID, Created: j.Created, Total: j.Total,
		FinishedAt: j.finishedAt,
		Status:     status, Progress: progress, Result: result,
	}
}

// Stop 取消未完成任务并停止 reap。
func (r *JobRegistry) Stop() {
	r.once.Do(func() { close(r.stopCh) })
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, j := range r.jobs {
		j.mu.Lock()
		if j.cancel != nil && !j.done {
			j.cancel()
		}
		j.mu.Unlock()
	}
}

// 这些变量暴露为包级变量以允许测试快速触发；生产保持默认值。
var (
	reapInterval      = 5 * time.Minute
	finishSendTimeout = 5 * time.Second
)

func (r *JobRegistry) reapLoop() {
	t := time.NewTicker(reapInterval)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			r.Reap()
		case <-r.stopCh:
			return
		}
	}
}

// Reap 清理过期已完成任务（测试可调用）。
//
// TTL 从**完成时刻**（finishedAt）起算而不是 Created：跑超 30 分钟的任务若按
// Created 计时，Finish 后第一次 reap（≤5 分钟）就会被清掉——列表丢终态、轮询
// 404、jobs.json 记录消失（review R8）。旧落盘记录没有 finishedAt，回退 Created
// 保持历史行为。
//
// interrupted 任务用更长的保留期：它是「需人工对账」的证据（如复制成功但源未删除），
// 若沿用 30 分钟 TTL，重启后首次 reap 就会把刚恢复的记录删掉，使恢复功能形同虚设。
func (r *JobRegistry) Reap() {
	now := time.Now()
	cutoff := now.Add(-JobTTL)
	interruptedCutoff := now.Add(-JobInterruptedTTL)
	removed := false
	r.mu.Lock()
	for id, j := range r.jobs {
		j.mu.Lock()
		base := j.finishedAt
		if base.IsZero() {
			base = j.Created
		}
		expired := j.done && base.Before(cutoff)
		if expired && j.progress.Status == JobStatusInterrupted {
			expired = base.Before(interruptedCutoff)
		}
		j.mu.Unlock()
		if expired {
			delete(r.jobs, id)
			removed = true
		}
	}
	r.mu.Unlock()
	if removed {
		r.persistJobs()
	}
}

// ErrTooManyJobs 表示在册（未终结）任务数已达上限。
var ErrTooManyJobs = errors.New("too many running jobs")

// TryCreate 在容量允许时注册新任务；超限返回 ErrTooManyJobs 且不产生任何副作用。
//
// 上限只统计「未终结」任务：已 done/cancelled/interrupted 的历史任务不应
// 永久占满名额（否则恢复出的中断任务会让服务再也无法接受新任务）。
func (r *JobRegistry) TryCreate(total int, cancel context.CancelFunc) (*Job, error) {
	j := &Job{
		ID:       uuid.NewString(),
		Created:  time.Now(),
		Total:    total,
		progress: JobProgress{Total: total, Status: JobStatusRunning},
		subs:     make(map[chan JobProgress]struct{}),
		cancel:   cancel,
		lastSave: time.Now(),
	}
	if r.persister != nil {
		j.persist = r.persistJobs
	}
	r.mu.Lock()
	if r.runningCountLocked() >= r.maxJobs {
		r.mu.Unlock()
		return nil, ErrTooManyJobs
	}
	r.jobs[j.ID] = j
	r.mu.Unlock()
	r.persistJobs()
	return j, nil
}

// runningCountLocked 统计未终结任务数（调用方须持有 r.mu）。
func (r *JobRegistry) runningCountLocked() int {
	n := 0
	for _, j := range r.jobs {
		j.mu.Lock()
		done := j.done
		j.mu.Unlock()
		if !done {
			n++
		}
	}
	return n
}

// ActiveCount 返回在册（未终结）任务数，与 TryCreate 的上限判定共用同一口径。
// 供 /api/metrics 的 `s3c_jobs_active` 暴露（ROADMAP #18）。
func (r *JobRegistry) ActiveCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.runningCountLocked()
}

// Get 按 id 取任务。
func (r *JobRegistry) Get(id string) (*Job, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	j, ok := r.jobs[id]
	return j, ok
}

// maxSubscribersPerJob 单个任务的并发订阅上限。
//
// 为什么需要：SSE 路由的 withStreamLimit 管的是「全局并发流」，同一任务仍可被反复订阅；
// 而 Finish 对每个订阅者都要投递（带上限），订阅数无界时终态关闭的最坏耗时随之线性增长。
const maxSubscribersPerJob = 16

// Subscribe 订阅进度；已结束则立即推送终态并关闭 channel。
//
// 第二个返回值为 false 表示该任务的订阅位已满（见 maxSubscribersPerJob），
// 调用方应回 503 而不是继续接受——Finish 对每个订阅者都要投递（带超时），
// 无上限会让终态关闭的最坏耗时随订阅数线性增长。
func (j *Job) Subscribe() (chan JobProgress, bool) {
	ch := make(chan JobProgress, 16)
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.done {
		ch <- j.progress
		close(ch)
		return ch, true
	}
	if len(j.subs) >= maxSubscribersPerJob {
		return nil, false
	}
	j.subs[ch] = struct{}{}
	ch <- j.progress
	return ch, true
}

// Unsubscribe 取消订阅。
func (j *Job) Unsubscribe(ch chan JobProgress) {
	j.mu.Lock()
	delete(j.subs, ch)
	j.mu.Unlock()
}

// Emit 广播中间进度（慢订阅者可丢中间帧）；落盘按 jobProgressPersistEvery 节流。
//
// **投递必须在锁内完成**：Finish 在同一把锁下清空订阅表、解锁后才 close 这些 channel。
// 旧实现「锁内快照、解锁后发送」与 Finish 的 close 之间没有互斥，一旦并发就会
// send on closed channel → panic（进程不 recover，直接退出）。
// 发送都是非阻塞的（default 分支丢帧），持锁时间有界。
func (j *Job) Emit(p JobProgress) {
	j.mu.Lock()
	// Finish 之后迟到的进度帧必须丢弃（review R9）：RunBatch 每帧都带
	// Status:"running"，异步批量 goroutine 收尾时该帧可能落在 Finish 之后；
	// 改写终态会让内存与落盘快照退回非终态，重启后 restore() 把已完成任务
	// 误标 interrupted，触发虚假对账告警。
	if j.done {
		j.mu.Unlock()
		return
	}
	j.progress = p
	shouldPersist := j.persist != nil && time.Since(j.lastSave) >= jobProgressPersistEvery
	if shouldPersist {
		j.lastSave = time.Now()
	}
	for ch := range j.subs {
		select {
		case ch <- p:
		default:
		}
	}
	j.mu.Unlock()

	// 落盘在锁外做：I/O 不应拖长与 Finish/Subscribe 的互斥窗口。
	if shouldPersist {
		j.persist()
	}
}

// Finish 写入终态并阻塞投递（短超时）后关闭订阅。
func (j *Job) Finish(out JobResult, status string) {
	j.mu.Lock()
	if j.done {
		j.mu.Unlock()
		return
	}
	j.result = out
	j.done = true
	j.finishedAt = time.Now() // Reap 的 TTL 起点：完成时刻（review R8）
	if status == "" {
		status = "done"
	}
	j.progress = JobProgress{
		Done: j.Total, Total: j.Total,
		Migrated: out.Migrated, Failed: out.Failed, Status: status,
	}
	subs := make([]chan JobProgress, 0, len(j.subs))
	for ch := range j.subs {
		subs = append(subs, ch)
	}
	final := j.progress
	j.subs = make(map[chan JobProgress]struct{})
	persist := j.persist
	j.mu.Unlock()

	// 终态必须落盘：这是「任务是否完成」的唯一持久证据。
	if persist != nil {
		persist()
	}

	for _, ch := range subs {
		select {
		case ch <- final:
		case <-time.After(finishSendTimeout):
		}
		close(ch)
	}
}

// Snapshot 返回当前进度/结果/是否完成。
func (j *Job) Snapshot() (JobProgress, JobResult, bool) {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.progress, j.result, j.done
}

// Cancel 取消未完成任务；已完成返回 false。
func (j *Job) Cancel() (cancelled bool, alreadyDone bool) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.done {
		return false, true
	}
	if j.cancel != nil {
		j.cancel()
	}
	return true, false
}

// ProgressFrom 将批量 Progress 转为 JobProgress（OK→migrated）。
func ProgressFrom(p Progress) JobProgress {
	return JobProgress{
		Done: p.Done, Total: p.Total, Migrated: p.OK, Failed: p.Failed,
		Key: p.Key, Error: p.Error, Status: p.Status,
	}
}

// ResultFromBatch 将 BatchResult 转为 JobResult。
func ResultFromBatch(out BatchResult) JobResult {
	return JobResult{
		Migrated: out.OK, Failed: out.Failed, FirstError: out.FirstError, FailKeys: out.FailKeys,
	}
}
