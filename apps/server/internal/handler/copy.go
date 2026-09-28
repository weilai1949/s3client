package handler

import (
	"context"
	"net/http"
	"strings"

	"github.com/weilai1949/s3clinet/apps/server/internal/model"
	"github.com/weilai1949/s3clinet/apps/server/internal/s3wrap"
	"github.com/weilai1949/s3clinet/apps/server/internal/service"
)

// copyObject 复制单个对象到目标桶/目标 key（不删除源）。
func (h *Handler) copyObject(w http.ResponseWriter, r *http.Request) {
	client, acc, ok := h.accountClient(w, r)
	if !ok {
		return
	}
	var req struct {
		Bucket    string `json:"bucket"`
		Key       string `json:"key"`
		NewKey    string `json:"newKey"`
		NewBucket string `json:"newBucket"`
	}
	if err := h.readJSON(r, &req); err != nil {
		h.writeBadJSON(w, err)
		return
	}
	if req.Key == "" || req.NewKey == "" {
		h.writeErr(w, http.StatusBadRequest, "key and newKey are required")
		return
	}
	bucket := req.Bucket
	if bucket, ok = h.bucketOr(w, acc, bucket); !ok {
		return
	}
	targetBucket := req.NewBucket
	if targetBucket == "" {
		targetBucket = bucket
	}
	if targetBucket == bucket && req.NewKey == req.Key {
		h.writeErr(w, http.StatusBadRequest, "newKey must differ from key in the same bucket")
		return
	}
	if err := client.CopyObject(r.Context(), bucket, req.Key, targetBucket, req.NewKey); err != nil {
		h.writeInternalErr(w, err, "copy operation failed")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"copied": req.NewKey, "bucket": targetBucket})
}

// copyMany 批量复制/移动所选文件到目标桶 + 前缀。
func (h *Handler) copyMany(w http.ResponseWriter, r *http.Request) {
	client, acc, ok := h.accountClient(w, r)
	if !ok {
		return
	}
	var req struct {
		Bucket       string   `json:"bucket"`
		TargetBucket string   `json:"targetBucket"`
		TargetPrefix string   `json:"targetPrefix"`
		Keys         []string `json:"keys"`
		DeleteSource bool     `json:"deleteSource"`
	}
	if err := h.readJSON(r, &req); err != nil {
		h.writeBadJSON(w, err)
		return
	}
	if len(req.Keys) == 0 {
		h.writeErr(w, http.StatusBadRequest, "keys are required")
		return
	}
	if len(req.Keys) > maxBatchKeys {
		h.writeErr(w, http.StatusBadRequest, "too many keys (max 10000 per request)")
		return
	}
	bucket := req.Bucket
	if bucket, ok = h.bucketOr(w, acc, bucket); !ok {
		return
	}
	targetBucket := req.TargetBucket
	if targetBucket == "" {
		targetBucket = bucket
	}
	pairs := make([][2]string, 0, len(req.Keys))
	for _, k := range req.Keys {
		if k == "" {
			continue
		}
		pairs = append(pairs, [2]string{k, service.BaseKey(k, req.TargetPrefix)})
	}
	if !req.DeleteSource {
		out := service.CopyKeys(r.Context(), client, bucket, targetBucket, pairs, 4, nil)
		h.writeJSON(w, http.StatusOK, copyBatchJSON(out, len(pairs), false))
		return
	}
	out := service.MoveKeys(r.Context(), client, bucket, targetBucket, pairs, 4, nil)
	// 移动是「复制成功后删源」的组合动作，源 key 已永久消失：必须与异步版一样留下
	// objects.move 审计（review R3 的同步对齐，threat-model.md R 抵赖缓解）。
	// 移动与纯复制共用 copyBatchJSON：响应形状不得随 deleteSource 漂移（review Nit），
	// truncated 恒写 false（本端点不截断 keys 列表，openapi 契约要求该键存在）。
	h.audit(r, auditObjectsMove, "bucket", bucket, "targetBucket", targetBucket,
		"total", len(pairs), "moved", out.OK, "failed", out.Failed)
	h.writeJSON(w, http.StatusOK, copyBatchJSON(out, len(pairs), false))
}

// copyManyAsync 异步批量复制/移动：立即返回 jobId，进度复用 migrate jobs SSE。
func (h *Handler) copyManyAsync(w http.ResponseWriter, r *http.Request) {
	client, acc, ok := h.accountClient(w, r)
	if !ok {
		return
	}
	var req struct {
		Bucket       string   `json:"bucket"`
		TargetBucket string   `json:"targetBucket"`
		TargetPrefix string   `json:"targetPrefix"`
		Keys         []string `json:"keys"`
		DeleteSource bool     `json:"deleteSource"`
	}
	if err := h.readJSON(r, &req); err != nil {
		h.writeBadJSON(w, err)
		return
	}
	if len(req.Keys) == 0 {
		h.writeErr(w, http.StatusBadRequest, "keys are required")
		return
	}
	if len(req.Keys) > maxBatchKeys {
		h.writeErr(w, http.StatusBadRequest, "too many keys (max 10000 per request)")
		return
	}
	bucket := req.Bucket
	if bucket, ok = h.bucketOr(w, acc, bucket); !ok {
		return
	}
	targetBucket := req.TargetBucket
	if targetBucket == "" {
		targetBucket = bucket
	}
	pairs := make([][2]string, 0, len(req.Keys))
	for _, k := range req.Keys {
		if k == "" {
			continue
		}
		pairs = append(pairs, [2]string{k, service.BaseKey(k, req.TargetPrefix)})
	}
	ctx, cancel := context.WithTimeout(context.Background(), migrateJobTimeout)
	job, ok := h.newJob(w, len(pairs), cancel)
	if !ok {
		return
	}
	if req.DeleteSource {
		// 移动是「复制成功后删源」的组合动作，半成功（复制了但没删/删了没复制）需按
		// 任务审计对账；纯复制不记该事件（review R3）。
		h.audit(r, auditObjectsMove, "jobId", job.ID, "bucket", bucket,
			"targetBucket", targetBucket, "total", job.Total)
	}
	go func() {
		defer cancel()
		var out service.BatchResult
		if !req.DeleteSource {
			out = service.CopyKeys(ctx, client, bucket, targetBucket, pairs, 4, func(p service.Progress) {
				job.Emit(service.ProgressFrom(p))
			})
		} else {
			out = service.MoveKeys(ctx, client, bucket, targetBucket, pairs, 4, func(p service.Progress) {
				job.Emit(service.ProgressFrom(p))
			})
		}
		status := "done"
		if ctx.Err() != nil {
			status = "cancelled"
		}
		job.Finish(jobResultFromBatch(out), status)
	}()
	h.writeJSON(w, http.StatusAccepted, map[string]any{"jobId": job.ID, "total": job.Total})
}

type copyPrefixReq struct {
	Bucket       string `json:"bucket"`
	Prefix       string `json:"prefix"`
	TargetBucket string `json:"targetBucket"`
	TargetPrefix string `json:"targetPrefix"`
}

func (h *Handler) parseCopyPrefix(w http.ResponseWriter, r *http.Request) (
	req copyPrefixReq, client *s3wrap.Client, bucket, targetBucket string, ok bool,
) {
	var acc *model.Account
	client, acc, ok = h.accountClient(w, r)
	if !ok {
		return
	}
	if err := h.readJSON(r, &req); err != nil {
		h.writeBadJSON(w, err)
		ok = false
		return
	}
	if req.Prefix == "" {
		h.writeErr(w, http.StatusBadRequest, "prefix is required")
		ok = false
		return
	}
	bucket = req.Bucket
	if bucket, ok = h.bucketOr(w, acc, bucket); !ok {
		return
	}
	targetBucket = req.TargetBucket
	if targetBucket == "" {
		targetBucket = bucket
	}
	if targetBucket == bucket &&
		(req.TargetPrefix == req.Prefix || strings.HasPrefix(req.TargetPrefix, req.Prefix)) {
		h.writeErr(w, http.StatusBadRequest, "targetPrefix must not overlap source prefix in the same bucket")
		ok = false
		return
	}
	ok = true
	return
}

// listPrefixMaxPages 是 listPrefixKeys 的页数硬上限，与 service 的 listMaxPages 同值。
// 空页 + 每次都前进的 token 时 maxCopy 的计数永不增长，只有页数上限拦得住（否则同步
// copy-prefix / delete-prefix 会一直占着 withStreamLimit 的流槽位，见 service.deletePrefix）。
const listPrefixMaxPages = 100

// listPrefixKeys 列举 prefix 下最多 maxCopy 个 key。
//
// truncated 表示「有对象未被收录」：或本页装不下、或收满后对端仍称有下一页、或列举因
// 安全上限（页数上限 / token 缺失或不前进）提前停止——口径与 service 的 listAll /
// indexDst / deletePrefix 一致（review §B6 与 R6）。
func (h *Handler) listPrefixKeys(ctx context.Context, client *s3wrap.Client, bucket, prefix string, maxCopy int) ([]string, bool, error) {
	var keys []string
	token := ""
	for page := 0; page < listPrefixMaxPages; page++ {
		if err := ctx.Err(); err != nil {
			return nil, false, err
		}
		p, err := client.ListObjectsPage(ctx, bucket, prefix, "", token, "", 1000)
		if err != nil {
			return nil, false, err
		}
		dropped := false
		for _, o := range p.Objects {
			if len(keys) >= maxCopy {
				dropped = true // 本页还有对象装不进上限：明确截断，不得静默丢弃
				break
			}
			keys = append(keys, o.Key)
		}
		if dropped {
			return keys, true, nil
		}
		if len(keys) >= maxCopy {
			// 正好收满且本页无丢弃：对端还声称有下一页才算截断（正好等于上限且
			// 对端声明列举完成 ⇒ 全部收录，不是截断）。
			return keys, p.IsTruncated, nil
		}
		if !p.IsTruncated {
			return keys, false, nil
		}
		// 同 service.deletePrefix：token 缺失或不前进必须停并标记截断，否则空转。
		if p.NextToken == "" || p.NextToken == token {
			return keys, true, nil
		}
		token = p.NextToken
	}
	return keys, true, nil
}

func copyBatchJSON(out service.BatchResult, total int, truncated bool) map[string]any {
	resp := map[string]any{"copied": out.OK, "failed": out.Failed, "total": total, "truncated": truncated}
	if out.FirstError != "" {
		resp["lastError"] = out.FirstError
	}
	if keys := capFailKeys(out.FailKeys); len(keys) > 0 {
		resp["failedKeys"] = keys
	}
	return resp
}

// copyPrefix 同步递归复制前缀。
func (h *Handler) copyPrefix(w http.ResponseWriter, r *http.Request) {
	req, client, bucket, targetBucket, ok := h.parseCopyPrefix(w, r)
	if !ok {
		return
	}
	keys, truncated, err := h.listPrefixKeys(r.Context(), client, bucket, req.Prefix, 100_000)
	if err != nil {
		h.writeInternalErr(w, err, "copy operation failed")
		return
	}
	pairs := make([][2]string, 0, len(keys))
	for _, k := range keys {
		pairs = append(pairs, [2]string{k, service.RelKey(k, req.Prefix, req.TargetPrefix)})
	}
	out := service.CopyKeys(r.Context(), client, bucket, targetBucket, pairs, 4, nil)
	h.writeJSON(w, http.StatusOK, copyBatchJSON(out, len(keys), truncated))
}

// copyPrefixAsync 异步前缀复制：立即返回 jobId，进度复用 migrate jobs SSE/轮询。
func (h *Handler) copyPrefixAsync(w http.ResponseWriter, r *http.Request) {
	req, client, bucket, targetBucket, ok := h.parseCopyPrefix(w, r)
	if !ok {
		return
	}
	keys, truncated, err := h.listPrefixKeys(r.Context(), client, bucket, req.Prefix, 100_000)
	if err != nil {
		h.writeInternalErr(w, err, "copy operation failed")
		return
	}
	pairs := make([][2]string, 0, len(keys))
	for _, k := range keys {
		pairs = append(pairs, [2]string{k, service.RelKey(k, req.Prefix, req.TargetPrefix)})
	}
	ctx, cancel := context.WithTimeout(context.Background(), migrateJobTimeout)
	job, ok := h.newJob(w, len(pairs), cancel)
	if !ok {
		return
	}
	go func() {
		defer cancel()
		out := service.CopyKeys(ctx, client, bucket, targetBucket, pairs, 4, func(p service.Progress) {
			job.Emit(service.ProgressFrom(p))
		})
		status := "done"
		if ctx.Err() != nil {
			status = "cancelled"
		}
		job.Finish(jobResultFromBatch(out), status)
	}()
	h.writeJSON(w, http.StatusAccepted, map[string]any{
		"jobId": job.ID, "total": job.Total, "truncated": truncated,
	})
}
