package handler

// checksum.go —— 端到端校验和验证端点（POST /verify-checksum）。
//
// 流程：Head 选定算法阶梯（CRC64NVME → CRC32C → SHA256 → SHA1 → ETag-MD5）
// → 流式 GET 全对象本地重算 → 与存储端比对。无可验证来源（厂商未存校验和、
// 分段合成校验和、非单段 ETag）→ method="none"、match=false——如实降级，不是错误。

import (
	"net/http"
)

// verifyChecksum 本地重算对象内容校验和并与存储端值比对。
func (h *Handler) verifyChecksum(w http.ResponseWriter, r *http.Request) {
	client, acc, ok := h.accountClient(w, r)
	if !ok {
		return
	}
	var req struct {
		Bucket    string `json:"bucket"`
		Key       string `json:"key"`
		VersionID string `json:"versionId"`
	}
	if err := h.readJSON(r, &req); err != nil {
		h.writeBadJSON(w, err)
		return
	}
	if req.Key == "" {
		h.writeErr(w, http.StatusBadRequest, "key is required")
		return
	}
	bucket := req.Bucket
	if bucket, ok = h.bucketOr(w, acc, bucket); !ok {
		return
	}
	v, err := client.VerifyObjectChecksum(r.Context(), bucket, req.Key, req.VersionID)
	if err != nil {
		h.writeInternalErr(w, err, "checksum verification failed")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"bucket": bucket, "key": req.Key, "versionId": req.VersionID,
		"method": v.Method, "local": v.Local, "remote": v.Remote, "match": v.Match,
	})
}
