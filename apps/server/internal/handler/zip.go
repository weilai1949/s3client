package handler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/weilai1949/s3clinet/apps/server/internal/service"
)

// downloadZip 将所选对象流式打包为 ZIP 下载（不落盘、不占内存）。
func (h *Handler) downloadZip(w http.ResponseWriter, r *http.Request) {
	client, acc, ok := h.accountClient(w, r)
	if !ok {
		return
	}
	var req struct {
		Bucket string   `json:"bucket"`
		Keys   []string `json:"keys"`
	}
	if err := h.readJSON(r, &req); err != nil {
		h.writeBadJSON(w, err)
		return
	}
	if len(req.Keys) == 0 {
		h.writeErr(w, http.StatusBadRequest, "keys are required")
		return
	}
	if len(req.Keys) > maxZipKeys {
		h.writeErr(w, http.StatusBadRequest, fmt.Sprintf("too many keys, max %d", maxZipKeys))
		return
	}
	bucket := req.Bucket
	if bucket, ok = h.bucketOr(w, acc, bucket); !ok {
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="objects-%d.zip"`, time.Now().Unix()))
	beginStreamResponse(w)

	failKeys, zipErr := service.WriteObjectsZip(r.Context(), func(ctx context.Context, key string) (io.ReadCloser, string, error) {
		out, err := client.GetObjectStream(ctx, bucket, key, "", "")
		if err != nil {
			return nil, "", err
		}
		return out.Body, out.ContentType, nil
	}, req.Keys, w)
	// ZIP 部分失败此前不可观测（失败清单只写进包内，handler 忽略返回值）。
	// 这里落服务端日志并计入指标，使批量下载失败率可见（ASSESSMENT S6）。
	h.recordZipOutcome(bucket, len(req.Keys), failKeys, zipErr)
}

// recordZipOutcome 记录一次 ZIP 打包的结果：部分失败与整体失败都留痕。
func (h *Handler) recordZipOutcome(bucket string, total int, failKeys []string, err error) {
	if err != nil {
		metricZipFailed.Add(1)
		h.log.Error("zip packaging failed", "bucket", bucket, "total", total, "err", err)
		return
	}
	if len(failKeys) == 0 {
		return
	}
	metricZipPartialFailures.Add(1)
	metricZipFailedKeys.Add(int64(len(failKeys)))
	h.log.Warn("zip partial failure", "bucket", bucket, "total", total, "failed", len(failKeys), "failedKeys", failKeys)
}
