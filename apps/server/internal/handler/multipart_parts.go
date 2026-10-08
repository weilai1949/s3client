package handler

import (
	"net/http"
	"time"
)

// multipartParts 列出某次分段上传已在服务端真实存在的分段（只读）。
//
// 断点续传的**唯一真值**在服务端：前端只用本端点对齐清单、跳过已上传段，
// 本地持久化记录（含 uploadId）仅是候选，服务端返回空 / uploadId 失效时前端重新 init。
func (h *Handler) multipartParts(w http.ResponseWriter, r *http.Request) {
	client, acc, ok := h.accountClient(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	key := q.Get("key")
	uploadID := q.Get("uploadId")
	if key == "" || uploadID == "" {
		h.writeErr(w, http.StatusBadRequest, "key and uploadId are required")
		return
	}
	bucket := q.Get("bucket")
	if bucket, ok = h.bucketOr(w, acc, bucket); !ok {
		return
	}
	parts, err := client.ListParts(r.Context(), bucket, key, uploadID)
	if err != nil {
		h.writeInternalErr(w, err, "multipart operation failed")
		return
	}
	items := make([]map[string]any, 0, len(parts))
	for _, p := range parts {
		items = append(items, map[string]any{
			"partNumber":   p.PartNumber,
			"etag":         p.ETag,
			"size":         p.Size,
			"lastModified": p.LastModified.UTC().Format(time.RFC3339),
		})
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"parts": items})
}
