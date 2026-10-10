package handler

// schedules.go —— 计划任务 5 端点（ROADMAP §三 #6「计划任务与持续同步」）。
//
// 把一次性 migrate/sync 升级为「桶 → 桶定时备份」：CRUD 管理 cron 计划，
// 手动 run 与到点触发共用同一个 ScheduleTrigger（runScheduleJob）——解析
// 账号 → 建异步任务 → 后台 SyncKeys，进度/落盘/重启恢复复用既有 JobRegistry。
//
// 防叠加口径：schedActive 以计划 id 记录「在跑」标记，check-and-set 在同一把
// 锁内完成——手动 run 与调度 tick 并发也只会开一轮（上一轮未结束 → 409 / 跳过）。

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/weilai1949/s3client/apps/server/internal/model"
	"github.com/weilai1949/s3client/apps/server/internal/service"
	"github.com/weilai1949/s3client/apps/server/internal/store"
)

// errSchedConfig 表示计划引用的账号客户端无法构建（HTTP 映射 400，提示修复账号配置）。
var errSchedConfig = errors.New("invalid account configuration")

// scheduleRequest 是 POST/PUT /api/schedules 的请求体（端点专属 DTO，
// 注册表字段集与本结构 json tag 由 openapi_request_fields 门禁双向钉住）。
type scheduleRequest struct {
	SourceAccountID string `json:"sourceAccountId"`
	SourceBucket    string `json:"sourceBucket"`
	SourcePrefix    string `json:"sourcePrefix"`
	TargetAccountID string `json:"targetAccountId"`
	TargetBucket    string `json:"targetBucket"`
	TargetPrefix    string `json:"targetPrefix"`
	Mode            string `json:"mode"`
	Cron            string `json:"cron"`
	// Enabled 缺省为 true（nil 判定，非空串默认，不触发 required 门禁的默认值检测）。
	Enabled *bool `json:"enabled"`
}

// scheduleAccount 读取计划引用的账号；不存在回 404，store 故障回 500（复用
// migrate 口径：读取故障误报 404 会让用户去查不存在的账号）。
func (h *Handler) scheduleAccount(w http.ResponseWriter, id, which string) (*model.Account, bool) {
	acc, err := h.store.Get(id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			h.writeErr(w, http.StatusNotFound, which+" account not found")
			return nil, false
		}
		h.writeInternalErr(w, err, "failed to load "+which+" account")
		return nil, false
	}
	return acc, true
}

// parseSchedule 解码并校验计划请求（创建与更新共用；委托解码点，见
// openapi_inputsource / openapi_request_fields 门禁对 h.xxx() 调用闭包的解析）。
// ok=false 时响应已写出。返回的 Schedule 尚无 id/时间/排期，由 Scheduler 补齐。
func (h *Handler) parseSchedule(w http.ResponseWriter, r *http.Request) (service.Schedule, bool) {
	var req scheduleRequest
	if err := h.readJSON(r, &req); err != nil {
		h.writeBadJSON(w, err)
		return service.Schedule{}, false
	}
	if req.SourceAccountID == "" || req.TargetAccountID == "" {
		h.writeErr(w, http.StatusBadRequest, "sourceAccountId and targetAccountId are required")
		return service.Schedule{}, false
	}
	src, ok := h.scheduleAccount(w, req.SourceAccountID, "source")
	if !ok {
		return service.Schedule{}, false
	}
	dst, ok := h.scheduleAccount(w, req.TargetAccountID, "target")
	if !ok {
		return service.Schedule{}, false
	}
	if _, err := h.clients.get(src); err != nil {
		h.log.Debug("s3 client init", "err", err)
		h.writeErr(w, http.StatusBadRequest, "invalid source account configuration")
		return service.Schedule{}, false
	}
	if _, err := h.clients.get(dst); err != nil {
		h.log.Debug("s3 client init", "err", err)
		h.writeErr(w, http.StatusBadRequest, "invalid target account configuration")
		return service.Schedule{}, false
	}
	srcBucket, ok := h.bucketOr(w, src, req.SourceBucket)
	if !ok {
		return service.Schedule{}, false
	}
	dstBucket, ok := h.bucketOr(w, dst, req.TargetBucket)
	if !ok {
		return service.Schedule{}, false
	}
	// mode 枚举与注册表 EnumStr 双向钉住（enumContracts 登记抽取自本 switch）。
	mode := service.CompareMode(req.Mode)
	switch mode {
	case "":
		mode = service.CompareETag
	case service.CompareETag, service.CompareSizeTime, service.CompareAlways:
		// OK
	default:
		h.writeErr(w, http.StatusBadRequest, "mode must be etag, size_mtime or always")
		return service.Schedule{}, false
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	return service.Schedule{
		SourceAccountID: req.SourceAccountID,
		SourceBucket:    srcBucket,
		SourcePrefix:    req.SourcePrefix,
		TargetAccountID: req.TargetAccountID,
		TargetBucket:    dstBucket,
		TargetPrefix:    req.TargetPrefix,
		Mode:            mode,
		Cron:            req.Cron,
		Enabled:         enabled,
	}, true
}

// listSchedules 返回全部计划（最新在前）。
func (h *Handler) listSchedules(w http.ResponseWriter, r *http.Request) {
	h.writeJSON(w, http.StatusOK, map[string]any{"schedules": h.sched.List()})
}

// createSchedules 创建计划；校验失败回 400（cron/mode/账号/桶逐项点名字段）。
func (h *Handler) createSchedule(w http.ResponseWriter, r *http.Request) {
	s, ok := h.parseSchedule(w, r)
	if !ok {
		return
	}
	created, err := h.sched.Create(s)
	if err != nil {
		h.log.Debug("schedule validation failed", "err", err)
		h.writeErr(w, http.StatusBadRequest, service.ScheduleValidationMessage(err))
		return
	}
	h.writeJSON(w, http.StatusCreated, map[string]any{"schedule": created})
}

// updateSchedule 整体替换计划（保留 id/createdAt/运行态，排期按 cron 变更重算）。
func (h *Handler) updateSchedule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	s, ok := h.parseSchedule(w, r)
	if !ok {
		return
	}
	updated, err := h.sched.Update(id, s)
	if err != nil {
		if errors.Is(err, service.ErrScheduleNotFound) {
			h.writeErr(w, http.StatusNotFound, "schedule not found")
			return
		}
		h.log.Debug("schedule validation failed", "err", err)
		h.writeErr(w, http.StatusBadRequest, service.ScheduleValidationMessage(err))
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"schedule": updated})
}

// deleteSchedule 删除计划；不存在回 404，成功回显被删 id。
func (h *Handler) deleteSchedule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.sched.Delete(id); err != nil {
		h.writeErr(w, http.StatusNotFound, "schedule not found")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"deleted": id})
}

// runSchedule 立即触发一次计划（不改自动排期）；错误映射：
// 计划/账号不存在 404、上一轮未结束 409、在册任务满 503、账号配置损坏 400、
// 其余（store 读取故障等）走 writeInternalErr → 500。
func (h *Handler) runSchedule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	jobID, err := h.sched.RunNow(id)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrScheduleNotFound):
			h.writeErr(w, http.StatusNotFound, "schedule not found")
		case errors.Is(err, service.ErrScheduleRunning):
			h.writeErr(w, http.StatusConflict, "schedule run already in progress")
		case errors.Is(err, service.ErrTooManyJobs):
			h.writeErr(w, http.StatusServiceUnavailable, "too many running jobs; retry later")
		case errors.Is(err, errSchedConfig):
			h.writeErr(w, http.StatusBadRequest, "invalid account configuration")
		case errors.Is(err, store.ErrNotFound):
			h.writeErr(w, http.StatusNotFound, "schedule account not found")
		default:
			h.writeInternalErr(w, err, "failed to start schedule run")
		}
		return
	}
	h.writeJSON(w, http.StatusAccepted, map[string]any{"jobId": jobID, "scheduleId": id})
}

// runScheduleJob 是 service.ScheduleTrigger 实现：解析账号 → 防叠加 check-and-set
// → 建异步任务 → 后台 SyncKeys。出错时按契约返回空 jobID。
func (h *Handler) runScheduleJob(s service.Schedule) (string, error) {
	// 先解析（无锁）：计划创建后账号可能被删/改坏，错误由调用方映射状态码。
	src, err := h.store.Get(s.SourceAccountID)
	if err != nil {
		return "", fmt.Errorf("load source account: %w", err)
	}
	dst, err := h.store.Get(s.TargetAccountID)
	if err != nil {
		return "", fmt.Errorf("load target account: %w", err)
	}
	srcClient, err := h.clients.get(src)
	if err != nil {
		return "", fmt.Errorf("%w: source: %v", errSchedConfig, err)
	}
	dstClient, err := h.clients.get(dst)
	if err != nil {
		return "", fmt.Errorf("%w: target: %v", errSchedConfig, err)
	}

	// 防叠加 + 建任务在同一把锁内完成：并发的手动 run 与调度 tick 只会开一轮。
	h.schedMu.Lock()
	if h.schedActive[s.ID] {
		h.schedMu.Unlock()
		return "", service.ErrScheduleRunning
	}
	ctx, cancel := context.WithTimeout(context.Background(), migrateJobTimeout)
	job, err := h.migrateJobs.TryCreate(0, cancel) // total 未知：SyncKeys 首帧学习
	if err != nil {
		h.schedMu.Unlock()
		cancel()
		return "", err
	}
	h.schedActive[s.ID] = true
	h.schedMu.Unlock()

	go func() {
		defer cancel()
		defer h.endScheduleRun(s.ID)
		out, syncErr := service.SyncKeys(ctx, srcClient, dstClient,
			s.SourceBucket, s.SourcePrefix, s.TargetBucket, s.TargetPrefix,
			s.Mode, 4, func(p service.Progress) {
				job.Emit(service.ProgressFrom(p))
			})
		status := "done"
		if ctx.Err() != nil {
			status = "cancelled"
		}
		job.Finish(scheduleJobResult(out, syncErr), status)
	}()
	return job.ID, nil
}

// endScheduleRun 清除计划的在跑标记（幂等；goroutine 收尾调用）。
func (h *Handler) endScheduleRun(id string) {
	h.schedMu.Lock()
	delete(h.schedActive, id)
	h.schedMu.Unlock()
}

// scheduleJobResult 把 SyncResult 映射为任务结果：
// 列举失败（syncErr）与截断（Truncated）都必须在 FirstError 里可见——否则
// 定时备份把「源端 403/5xx」或「只拷了一部分」静默报成 done（同 sync 端点 review §B5 口径）。
func scheduleJobResult(out service.SyncResult, syncErr error) service.JobResult {
	res := service.JobResult{
		Migrated:   out.Copied,
		Failed:     out.Failed,
		FirstError: out.FirstError,
		FailKeys:   capFailKeys(out.FailKeys),
	}
	if res.FirstError == "" && syncErr != nil {
		res.FirstError = syncErr.Error()
	}
	if out.Truncated && res.FirstError == "" {
		res.FirstError = "source listing truncated; not all objects were synced"
	}
	return res
}
