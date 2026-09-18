package handler

import (
	"net/http"

	"github.com/weilai1949/s3clinet/apps/server/internal/service"
)

// migrateSyncRequest 增量同步请求体（与 migrate 一致 + 增量判定字段）。
type migrateSyncRequest struct {
	SourceAccountID string `json:"sourceAccountId"`
	SourceBucket    string `json:"sourceBucket"`
	SourcePrefix    string `json:"sourcePrefix"`
	TargetAccountID string `json:"targetAccountId"`
	TargetBucket    string `json:"targetBucket"`
	TargetPrefix    string `json:"targetPrefix"`
	// Mode：etag（默认）/ size_mtime / always。CompareMode 字符串值。
	Mode string `json:"mode"`
}

// migrateSyncResponse 同步结果，scanned / skipped / copied / failed 直观反映比对结果。
type migrateSyncResponse struct {
	Scanned   int      `json:"scanned"`
	Skipped   int      `json:"skipped"`
	Copied    int      `json:"copied"`
	Failed    int      `json:"failed"`
	FailKeys  []string `json:"failedKeys,omitempty"`
	LastError string   `json:"lastError,omitempty"`
}

// syncHandler 同步迁移（按 ETag / size+mtime 比对，仅复制差异对象）。
// 该端点为 P1/P2 路线「增量同步」的基础实现；后续可加 SSE 异步进度。
func (h *Handler) syncHandler(w http.ResponseWriter, r *http.Request) {
	var req migrateSyncRequest
	if err := h.readJSON(r, &req); err != nil {
		h.writeBadJSON(w, err)
		return
	}
	if req.SourceAccountID == "" || req.TargetAccountID == "" {
		h.writeErr(w, http.StatusBadRequest, "sourceAccountId and targetAccountId are required")
		return
	}
	src, err := h.store.Get(req.SourceAccountID)
	if err != nil {
		h.writeErr(w, http.StatusNotFound, "source account not found")
		return
	}
	dst, err := h.store.Get(req.TargetAccountID)
	if err != nil {
		h.writeErr(w, http.StatusNotFound, "target account not found")
		return
	}
	srcClient, err := h.clients.get(src)
	if err != nil {
		h.writeErr(w, http.StatusBadRequest, "invalid source account configuration")
		return
	}
	dstClient, err := h.clients.get(dst)
	if err != nil {
		h.writeErr(w, http.StatusBadRequest, "invalid target account configuration")
		return
	}
	srcBucket := req.SourceBucket
	var ok bool
	if srcBucket, ok = h.bucketOr(w, src, srcBucket); !ok {
		return
	}
	targetBucket := req.TargetBucket
	if targetBucket, ok = h.bucketOr(w, dst, targetBucket); !ok {
		return
	}
	mode := service.CompareMode(req.Mode)
	switch mode {
	case "", service.CompareETag, service.CompareSizeTime, service.CompareAlways:
		// OK
	default:
		h.writeErr(w, http.StatusBadRequest, "mode must be etag, size_mtime or always")
		return
	}
	out := service.SyncKeys(r.Context(), srcClient, dstClient, srcBucket, req.SourcePrefix, targetBucket, req.TargetPrefix, mode, 4, nil)
	resp := migrateSyncResponse{
		Scanned: out.Scanned, Skipped: out.Skipped, Copied: out.Copied, Failed: out.Failed,
		FailKeys: out.FailKeys, LastError: out.LastError,
	}
	h.writeJSON(w, http.StatusOK, resp)
}
