package handler

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/weilai1949/s3clinet/apps/server/internal/service"
)

// TestJobsListEndpoint 验证 GET /api/migrate/jobs 返回任务清单（含未完成/中断任务）。
func TestJobsListEndpoint(t *testing.T) {
	h, _ := gapStoreHandler(t)
	h.SetJobPersister(service.NewFileJobPersister(filepath.Join(t.TempDir(), "jobs.json")))
	t.Cleanup(h.Shutdown)

	// 造一个已完成任务，确保清单能真实反映注册表内容。
	_, cancel := context.WithCancel(context.Background())
	job := h.migrateJobs.Create(2, cancel)
	job.Finish(service.JobResult{Migrated: 2}, service.JobStatusDone)

	rr := httptest.NewRecorder()
	h.jobsList(rr, httptest.NewRequest(http.MethodGet, "/api/migrate/jobs", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rr.Code, rr.Body)
	}

	var resp struct {
		Jobs []service.JobRecord `json:"jobs"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v; body=%s", err, rr.Body)
	}
	if len(resp.Jobs) != 1 {
		t.Fatalf("jobs = %+v, want 1", resp.Jobs)
	}
	if resp.Jobs[0].ID != job.ID || resp.Jobs[0].Status != service.JobStatusDone {
		t.Errorf("job = %+v, want %s/done", resp.Jobs[0], job.ID)
	}
	if resp.Jobs[0].Result.Migrated != 2 {
		t.Errorf("result.migrated = %d, want 2", resp.Jobs[0].Result.Migrated)
	}
}

// TestJobsListEmptyIsArray 空清单必须序列化为 []，不能是 null（前端直接 .length / map）。
func TestJobsListEmptyIsArray(t *testing.T) {
	h, _ := gapStoreHandler(t)
	t.Cleanup(h.Shutdown)

	rr := httptest.NewRecorder()
	h.jobsList(rr, httptest.NewRequest(http.MethodGet, "/api/migrate/jobs", nil))

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(rr.Body.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	if string(raw["jobs"]) != "[]" {
		t.Errorf("jobs = %s, want []", raw["jobs"])
	}
}

// TestSetJobPersisterRecoversInterruptedJobs 验证启动装配路径：
// 注入 persister 后，历史 running 任务对前端呈现为 interrupted。
func TestSetJobPersisterRecoversInterruptedJobs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "jobs.json")

	// 模拟「上一次进程崩溃时留下 running 任务」。
	seed := service.NewFileJobPersister(path)
	if err := seed.Save([]service.JobRecord{{
		ID: "crashed", Total: 3, Status: service.JobStatusRunning,
		Progress: service.JobProgress{Done: 1, Total: 3, Migrated: 1, Status: service.JobStatusRunning},
	}}); err != nil {
		t.Fatal(err)
	}

	h, _ := gapStoreHandler(t)
	h.SetJobPersister(service.NewFileJobPersister(path))
	t.Cleanup(h.Shutdown)

	// 轮询接口必须能查到该任务，且为终态（前端据此提示「已中断，需对账」）。
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/migrate/jobs/crashed", nil)
	req.SetPathValue("id", "crashed")
	h.migrateJobStatus(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rr.Code, rr.Body)
	}
	var resp struct {
		Done     bool                `json:"done"`
		Progress service.JobProgress `json:"progress"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if !resp.Done {
		t.Error("done = false, want true for interrupted job")
	}
	if resp.Progress.Status != service.JobStatusInterrupted {
		t.Errorf("status = %q, want interrupted", resp.Progress.Status)
	}
}

// TestSetJobPersisterReplacesRegistry 替换 persister 时旧注册表必须停止（避免 goroutine 泄漏）。
func TestSetJobPersisterReplacesRegistry(t *testing.T) {
	h, _ := gapStoreHandler(t)
	old := h.migrateJobs

	_, cancel := context.WithCancel(context.Background())
	job := old.Create(1, cancel)
	job.Finish(service.JobResult{Migrated: 1}, service.JobStatusDone)

	h.SetJobPersister(service.NewFileJobPersister(filepath.Join(t.TempDir(), "j.json")))
	t.Cleanup(h.Shutdown)

	if h.migrateJobs == old {
		t.Fatal("registry not replaced")
	}
	// 旧注册表已停止：其 reap 循环退出，Stop 幂等不 panic。
	old.Stop()
}

// TestNewJobReturns503AtCapacity 在册任务达上限时异步端点返回 503 而非无限接受
// （todolist #17 / ASSESSMENT M4）。用 ctx 取消状态间接确认 cancel() 被调用，
// 避免注册失败时泄漏 WithTimeout 定时器。
func TestNewJobReturns503AtCapacity(t *testing.T) {
	h, _ := gapStoreHandler(t)
	t.Cleanup(h.Shutdown)
	SetJobCapForTest(h, 1)

	// 占满唯一名额（首个 job 的返回值不参与断言，只需注册成功）。
	if _, ok := h.newJob(httptest.NewRecorder(), 1, func() {}); !ok {
		t.Fatal("first job should be accepted")
	}

	// 第二个必须 503，且传入的 cancel 必须已被调用（否则定时器泄漏）。
	cancelled := false
	rr := httptest.NewRecorder()
	job, ok := h.newJob(rr, 1, func() { cancelled = true })
	if ok || job != nil {
		t.Fatalf("job=%v ok=%v, want nil/false at capacity", job, ok)
	}
	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503; body=%s", rr.Code, rr.Body)
	}
	if !cancelled {
		t.Error("cancel() must be called when registration fails")
	}
}

// seedInterruptedJobHandler 造一个「上次进程崩溃留下 running 任务」的 handler：
// 经 SetJobPersister 恢复后该任务为 interrupted（终态）。返回其 job id。
func seedInterruptedJobHandler(t *testing.T) (*Handler, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "jobs.json")
	seed := service.NewFileJobPersister(path)
	if err := seed.Save([]service.JobRecord{{
		ID: "interrupted-job-1", Total: 3, Status: service.JobStatusRunning,
		Progress: service.JobProgress{Done: 1, Total: 3, Migrated: 1, Status: service.JobStatusRunning},
	}}); err != nil {
		t.Fatal(err)
	}
	h, _ := gapStoreHandler(t)
	h.SetJobPersister(service.NewFileJobPersister(path))
	t.Cleanup(h.Shutdown)
	return h, "interrupted-job-1"
}

// TestMigrateJobEventsInterruptedStreamCloses 复现并锁死 P0-1：
//
// interrupted 是 IsTerminalJobStatus 认可的终态，但 SSE 循环只认 done/cancelled，
// 且 `case p := <-ch` 不检查通道关闭——而 Subscribe 对已结束任务「推一帧后立即 close」，
// **已关闭的 channel 永远就绪** → 死循环刷零值帧（实测 3 秒 37 MB 且流永不结束）。
//
// 验收：收到 interrupted 终态帧后流必须 EOF（不是靠客户端超时断开）。
func TestMigrateJobEventsInterruptedStreamCloses(t *testing.T) {
	h, jobID := seedInterruptedJobHandler(t)

	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/api/migrate/jobs/"+jobID+"/events", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("sse request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("sse status = %d, want 200", resp.StatusCode)
	}

	// 只应有一帧（任务快照）；出现第 2 帧即证明 handler 在已关闭的 channel 上自旋。
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	frames := 0
	var last service.JobProgress
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		frames++
		if frames > 1 {
			t.Fatalf("P0-1 复现：收到 %d 个 data 帧，流未在终态帧后关闭（自旋）", frames)
		}
		if err := json.Unmarshal([]byte(line[6:]), &last); err != nil {
			t.Fatalf("frame %d: %v", frames, err)
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("read stream: %v", err)
	}
	if frames != 1 {
		t.Fatalf("data frames = %d, want 1（终态快照）", frames)
	}
	if last.Status != service.JobStatusInterrupted {
		t.Fatalf("frame status = %q, want %q", last.Status, service.JobStatusInterrupted)
	}
}

// blockingWriter 在第一次 Write 时阻塞，直到测试放行：用于让 SSE handler 停止消费订阅通道，
// 从而制造「缓冲被灌满」的条件。
type blockingWriter struct {
	started chan struct{}
	release chan struct{}
	once    sync.Once
	mu      sync.Mutex
	buf     bytes.Buffer
}

func (b *blockingWriter) Header() http.Header { return http.Header{} }
func (b *blockingWriter) WriteHeader(int)     {}
func (b *blockingWriter) Flush()              {}

func (b *blockingWriter) Write(p []byte) (int, error) {
	b.once.Do(func() {
		close(b.started)
		<-b.release
	})
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

// TestMigrateJobEventsClosedWithoutTerminalFrame 覆盖 SSE 循环的「通道已关闭但从未收到终态帧」分支。
//
// 这是 P0-1 的深水区：`Finish` 对每个订阅者的终态投递带超时（finishSendTimeout），
// 订阅者缓冲已满时**终态帧会被丢弃**，随后 channel 关闭——此时唯一可用的信号就是「通道已关闭」。
// 缺 `ok` 判断的实现会在这里退化成死循环（已关闭的 channel 永远就绪）。
func TestMigrateJobEventsClosedWithoutTerminalFrame(t *testing.T) {
	h, _ := gapStoreHandler(t)
	t.Cleanup(h.Shutdown)

	_, cancel := context.WithCancel(context.Background())
	defer cancel()
	job := h.migrateJobs.Create(1, cancel)

	w := &blockingWriter{started: make(chan struct{}), release: make(chan struct{})}
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.SetPathValue("id", job.ID)

	done := make(chan struct{})
	go func() {
		defer close(done)
		h.migrateJobEvents(w, req)
	}()

	select {
	case <-w.started: // handler 已订阅并阻塞在写第一帧上
	case <-time.After(5 * time.Second):
		t.Fatal("handler 未开始写第一帧")
	}

	// handler 阻塞期间灌满 16 个缓冲（超出部分被 Emit 丢弃），
	// 于是 Finish 的终态投递必然超时 → 终态帧丢失，channel 关闭。
	for i := 0; i < 64; i++ {
		job.Emit(service.JobProgress{Done: i, Total: 64, Status: service.JobStatusRunning})
	}
	job.Finish(service.JobResult{}, service.JobStatusDone)

	close(w.release) // 放行写入：handler 排空缓冲后必须从「通道已关闭」退出
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("通道已关闭且无终态帧时 handler 没有退出（退化为 P0-1 的自旋）")
	}
}

// TestMigrateJobEventsSubscriberCap 单个任务的 SSE 订阅数必须有上限：
// SSE 路由的 withStreamLimit 约束的是「全局并发流」，同一任务仍可被反复订阅，
// 而 Finish 对每个订阅者都要投递（带超时）——无上限会让终态关闭的最坏耗时随订阅数线性增长。
func TestMigrateJobEventsSubscriberCap(t *testing.T) {
	h, _ := gapStoreHandler(t)
	t.Cleanup(h.Shutdown)

	_, cancel := context.WithCancel(context.Background())
	defer cancel()
	job := h.migrateJobs.Create(1, cancel)

	var held []chan service.JobProgress
	for {
		ch, ok := job.Subscribe()
		if !ok {
			break
		}
		held = append(held, ch)
		if len(held) > 1024 {
			t.Fatal("Subscribe 无上限：已接受 1024 个订阅者")
		}
	}
	if len(held) == 0 {
		t.Fatal("首个订阅必须成功")
	}

	// 订阅位已满 → 端点必须回 503，而不是挂起或继续接受。
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.SetPathValue("id", job.ID)
	rr := httptest.NewRecorder()
	h.migrateJobEvents(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503 when subscribers saturated; body=%s", rr.Code, rr.Body)
	}
}
