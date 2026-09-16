package handler

import (
	"net/http"

	"github.com/weilai1949/s3clinet/server/internal/service"
)

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
