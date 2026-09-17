package handler

import (
	"context"
	"net/http"

	"github.com/weilai1949/s3clinet/server/internal/service"
)

// newJob 注册异步任务；在册任务数达上限时返回 503 并释放调用方的 ctx。
//
// 返回 ok=false 表示已写出错误响应（调用方应直接 return）。注册失败时必须
// 调用 cancel()，否则 WithTimeout 派生的定时器要等超时才释放（goroutine 泄漏）。
func (h *Handler) newJob(w http.ResponseWriter, total int, cancel context.CancelFunc) (*service.Job, bool) {
	job, err := h.migrateJobs.TryCreate(total, cancel)
	if err != nil {
		cancel()
		h.writeErr(w, http.StatusServiceUnavailable, "too many running jobs; retry later")
		return nil, false
	}
	return job, true
}

// jobsList 返回异步任务清单（最新在前），含跨重启恢复的 interrupted 任务。
//
// 用途：前端「未完成任务」视图。任务清单持久化后，进程重启不会丢失记录，
// 「复制成功但源未删除」的移动任务可据此被发现并对账（todolist #19）。
func (h *Handler) jobsList(w http.ResponseWriter, r *http.Request) {
	recs := h.migrateJobs.List()
	if recs == nil {
		recs = []service.JobRecord{} // 序列化为 []，避免前端拿到 null
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"jobs": recs})
}
