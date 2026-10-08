package handler

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/weilai1949/s3client/apps/server/internal/s3wrap"
	"github.com/weilai1949/s3client/apps/server/internal/service"
	"github.com/weilai1949/s3client/apps/server/internal/tracing"
)

func (h *Handler) listObjects(w http.ResponseWriter, r *http.Request) {
	client, acc, ok := h.accountClient(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	bucket := q.Get("bucket")
	if bucket, ok = h.bucketOr(w, acc, bucket); !ok {
		return
	}
	maxKeys, _ := strconv.Atoi(q.Get("maxKeys"))
	if maxKeys < 1 {
		maxKeys = 1000
	}
	if maxKeys > 1000 {
		maxKeys = 1000
	}

	page, err := client.ListObjectsPage(r.Context(), bucket, q.Get("prefix"), q.Get("delimiter"), q.Get("continuationToken"), q.Get("startAfter"), int32(maxKeys))
	if err != nil {
		h.writeInternalErr(w, err, "list objects failed")
		return
	}
	resp := listObjectsResponse{
		Objects:        make([]objectItem, 0, len(page.Objects)),
		CommonPrefixes: make([]string, 0, len(page.CommonPrefixes)),
		IsTruncated:    page.IsTruncated,
		NextToken:      page.NextToken,
	}
	for _, o := range page.Objects {
		resp.Objects = append(resp.Objects, fromS3Object(o))
	}
	resp.CommonPrefixes = append(resp.CommonPrefixes, page.CommonPrefixes...)
	h.writeJSON(w, http.StatusOK, resp)
}

// headObject 返回单个对象的元数据详情（HeadObject）。
func (h *Handler) headObject(w http.ResponseWriter, r *http.Request) {
	client, acc, ok := h.accountClient(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	bucket := q.Get("bucket")
	if bucket, ok = h.bucketOr(w, acc, bucket); !ok {
		return
	}
	key := q.Get("key")
	if key == "" {
		h.writeErr(w, http.StatusBadRequest, "key is required")
		return
	}
	out, err := client.HeadObjectMeta(r.Context(), bucket, key, q.Get("versionId"))
	if err != nil {
		if s3wrap.IsNotFound(err) {
			h.writeErr(w, http.StatusNotFound, "object not found")
			return
		}
		h.writeInternalErr(w, err, "head object failed")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"key":          key,
		"size":         out.Size,
		"lastModified": out.LastModified,
		"etag":         out.ETag,
		"contentType":  out.ContentType,
		"storageClass": out.StorageClass,
		"metadata":     out.Metadata,
	})
}

// allowedStorageClasses 允许切换到的存储类型（S3 标准枚举的常用子集）。
var allowedStorageClasses = map[string]bool{
	"STANDARD":            true,
	"REDUCED_REDUNDANCY":  true,
	"STANDARD_IA":         true,
	"ONEZONE_IA":          true,
	"INTELLIGENT_TIERING": true,
	"GLACIER":             true,
	"GLACIER_IR":          true,
	"DEEP_ARCHIVE":        true,
	"EXPRESS_ONEZONE":     true,
}

// changeStorageClass 切换对象存储类型（服务端 CopyObject 复制到自身，并携带新 StorageClass）。
// 版本控制桶下写出一条新版本；versionId 为空表示切换当前对象。
func (h *Handler) changeStorageClass(w http.ResponseWriter, r *http.Request) {
	client, acc, ok := h.accountClient(w, r)
	if !ok {
		return
	}
	var req struct {
		Bucket       string `json:"bucket"`
		Key          string `json:"key"`
		VersionID    string `json:"versionId"`
		StorageClass string `json:"storageClass"`
	}
	if err := h.readJSON(r, &req); err != nil {
		h.writeBadJSON(w, err)
		return
	}
	if req.Key == "" {
		h.writeErr(w, http.StatusBadRequest, "key is required")
		return
	}
	if req.StorageClass == "" || !allowedStorageClasses[req.StorageClass] {
		// 不回显用户提交的 storageClass（L2）。
		h.log.Debug("unsupported storageClass", "storageClass", req.StorageClass, "key", req.Key)
		h.writeErr(w, http.StatusBadRequest, "unsupported storageClass")
		return
	}
	bucket, ok := h.bucketOr(w, acc, req.Bucket)
	if !ok {
		return
	}
	newVersion, err := client.ChangeObjectStorageClass(r.Context(), bucket, req.Key, req.VersionID, req.StorageClass)
	if err != nil {
		if s3wrap.HasErrorCode(err, "InvalidRequest", "InvalidStorageClass") {
			h.writeErr(w, http.StatusBadRequest, s3UserMessage(err))
			return
		}
		h.writeInternalErr(w, err, "change storage class failed")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"changed":      req.Key,
		"versionId":    newVersion,
		"storageClass": req.StorageClass,
	})
}

// mkdirObject 通过 PUT 空对象创建"文件夹"（S3 无真实目录，key 以 / 结尾）。
func (h *Handler) mkdirObject(w http.ResponseWriter, r *http.Request) {
	client, acc, ok := h.accountClient(w, r)
	if !ok {
		return
	}
	var req struct {
		Bucket string `json:"bucket"`
		Key    string `json:"key"`
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
	key := strings.TrimRight(req.Key, "/") + "/"
	if err := client.PutObject(r.Context(), bucket, key, nil, "", nil); err != nil {
		h.writeInternalErr(w, err, "create folder failed")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"created": key, "bucket": bucket})
}

// renameObject 复制到新 key 后删除源（跨桶移动需目标桶参数，默认同桶）。
func (h *Handler) renameObject(w http.ResponseWriter, r *http.Request) {
	client, acc, ok := h.accountClient(w, r)
	if !ok {
		return
	}
	var req struct {
		Bucket    string `json:"bucket"`
		Key       string `json:"key"`
		NewKey    string `json:"newKey"`
		NewBucket string `json:"newBucket"` // 可选：跨桶移动
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
	// 同桶内 newKey 与 key 相同视为无效（跨桶同名移动允许）。
	if targetBucket == bucket && req.Key == req.NewKey {
		h.writeErr(w, http.StatusBadRequest, "newKey must differ from key in the same bucket")
		return
	}
	// 先复制，成功后才删除源，避免复制失败丢数据。
	if err := client.CopyObject(r.Context(), bucket, req.Key, targetBucket, req.NewKey); err != nil {
		h.writeInternalErr(w, err, "rename copy failed")
		return
	}
	if err := client.DeleteObject(r.Context(), bucket, req.Key); err != nil {
		// 复制已成功但删除失败：源与目标同时存在，提示用户手动处理。
		h.writeInternalErr(w, err, "copied to new key but failed to delete source")
		return
	}
	// 重命名 = 复制成功后永久删除源 key：记 objects.move（带 key/newKey），
	// 否则把删除藏进「重命名」即可绕开 objects.delete 的审计覆盖（threat-model.md R）。
	h.audit(r, auditObjectsMove, "bucket", bucket, "key", req.Key,
		"targetBucket", targetBucket, "newKey", req.NewKey)
	h.writeJSON(w, http.StatusOK, map[string]any{"renamed": req.NewKey})
}

// maxDeleteKeys 限制单次 delete-objects 的 key 数。
// S3 DeleteObjects 本身最大 1000 一次；此处不放大以保持「一次 HTTP 一次 SDK 调用」的简单契约。
// 大量删除请走 delete-prefix（异步 + 自动分页）。
const maxDeleteKeys = 1000

// maxFailKeys 是 API 层返回 / 落盘的失败 key 上限：避免 10 万个 key 全失败时把响应
// （同步路径）与 jobs.json（异步路径）撑到约 10 MB。
//
// 这是对外的**承诺**（docs/FEATURES.md、docs/api.md）：所有回传 failedKeys 的端点都必须
// 经 capFailKeys 裁剪。此前只有 delete-prefix 异步路径裁剪，copy/migrate/sync 全部原样
// 回传——承诺与实现不符（docs/archive/review-2026-09-19.md §7.3 D4）。
const maxFailKeys = 200

// capFailKeys 把失败 key 列表裁剪到 maxFailKeys。未超限时原样返回（不复制）。
func capFailKeys(keys []string) []string {
	if len(keys) <= maxFailKeys {
		return keys
	}
	out := make([]string, maxFailKeys)
	copy(out, keys[:maxFailKeys])
	return out
}

// deleteObjects 批量删除：校验 / 审计 / 响应在本层，记账编排在 service.DeleteKeys。
func (h *Handler) deleteObjects(w http.ResponseWriter, r *http.Request) {
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
	if len(req.Keys) > maxDeleteKeys {
		h.writeErr(w, http.StatusBadRequest, "too many keys (max 1000); use delete-prefix for larger batches")
		return
	}
	bucket := req.Bucket
	if bucket, ok = h.bucketOr(w, acc, bucket); !ok {
		return
	}
	counts, err := service.DeleteKeys(r.Context(), client, bucket, req.Keys)
	if err != nil {
		h.writeInternalErr(w, err, "delete objects failed")
		return
	}
	h.audit(r, auditObjectsDelete, "bucket", bucket, "deleted", counts.Deleted, "failed", counts.Failed)
	resp := map[string]any{"deleted": counts.Deleted, "failed": counts.Failed}
	if counts.LastError != "" {
		resp["lastError"] = counts.LastError
	}
	h.writeJSON(w, http.StatusOK, resp)
}

// deletePrefix 递归删除前缀下的全部对象（编排在 service.RunDeletePrefix）。
func (h *Handler) deletePrefix(w http.ResponseWriter, r *http.Request) {
	client, acc, ok := h.accountClient(w, r)
	if !ok {
		return
	}
	var req struct {
		Bucket string `json:"bucket"`
		Prefix string `json:"prefix"`
	}
	if err := h.readJSON(r, &req); err != nil {
		h.writeBadJSON(w, err)
		return
	}
	if req.Prefix == "" {
		h.writeErr(w, http.StatusBadRequest, "prefix is required (拒绝空前缀以免误删全桶)")
		return
	}
	bucket := req.Bucket
	if bucket, ok = h.bucketOr(w, acc, bucket); !ok {
		return
	}
	res, err := service.RunDeletePrefix(r.Context(), client, bucket, req.Prefix)
	if err != nil {
		h.writeInternalErr(w, err, "delete prefix failed")
		return
	}
	h.audit(r, auditDeletePrefix, "bucket", bucket, "prefix", req.Prefix,
		"deleted", res.Deleted, "failed", res.Failed)
	resp := map[string]any{"deleted": res.Deleted, "failed": res.Failed, "truncated": res.Truncated}
	if res.LastError != "" {
		resp["lastError"] = res.LastError
	}
	h.writeJSON(w, http.StatusOK, resp)
}

// deletePrefixAsync 异步递归删除前缀；进度复用 migrate jobs（migrated=已删除数）。
func (h *Handler) deletePrefixAsync(w http.ResponseWriter, r *http.Request) {
	client, acc, ok := h.accountClient(w, r)
	if !ok {
		return
	}
	var req struct {
		Bucket string `json:"bucket"`
		Prefix string `json:"prefix"`
	}
	if err := h.readJSON(r, &req); err != nil {
		h.writeBadJSON(w, err)
		return
	}
	if req.Prefix == "" {
		h.writeErr(w, http.StatusBadRequest, "prefix is required (拒绝空前缀以免误删全桶)")
		return
	}
	bucket := req.Bucket
	if bucket, ok = h.bucketOr(w, acc, bucket); !ok {
		return
	}
	// 先列举以拿到 total（与 copy-prefix/async 一致），再异步删除。
	keys, truncated, err := h.listPrefixKeys(r.Context(), client, bucket, req.Prefix, 100_000)
	if err != nil {
		h.writeInternalErr(w, err, "delete prefix list failed")
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), migrateJobTimeout)
	job, ok := h.newJob(w, len(keys), cancel)
	if !ok {
		return
	}
	// 任务启动即审计（与同步路径对齐）：异步删除是不可逆动作，202 之后只能靠日志追溯。
	h.audit(r, auditDeletePrefix, "jobId", job.ID, "bucket", bucket, "prefix", req.Prefix,
		"total", job.Total, "truncated", truncated)
	go func() {
		defer cancel()
		counts, failKeys := service.DeleteKeysBatched(ctx, client, bucket, keys, maxFailKeys, func(p service.Progress) {
			job.Emit(service.ProgressFrom(p))
		})
		status := "done"
		if ctx.Err() != nil {
			status = "cancelled"
		}
		job.Finish(service.JobResult{
			Migrated: counts.Deleted, Failed: counts.Failed, FirstError: counts.LastError, FailKeys: failKeys,
		}, status)
	}()
	h.writeJSON(w, http.StatusAccepted, map[string]any{
		"jobId": job.ID, "total": job.Total, "truncated": truncated,
	})
}

// presign 生成 v4 签名 URL（get/put/post）。
func (h *Handler) presign(w http.ResponseWriter, r *http.Request) {
	_, endSpan := tracing.Start(r.Context(), "presign")
	defer endSpan()
	client, acc, ok := h.accountClient(w, r)
	if !ok {
		return
	}
	var req struct {
		Method    string `json:"method"`
		Key       string `json:"key"`
		Bucket    string `json:"bucket"`
		VersionID string `json:"versionId"`
		ExpiresIn int64  `json:"expiresIn"`
	}
	if err := h.readJSON(r, &req); err != nil {
		h.writeBadJSON(w, err)
		return
	}
	if req.Key == "" {
		h.writeErr(w, http.StatusBadRequest, "key is required")
		return
	}
	if req.Method == "" {
		req.Method = "put"
	}
	bucket := req.Bucket
	if bucket, ok = h.bucketOr(w, acc, bucket); !ok {
		return
	}
	expires := time.Duration(req.ExpiresIn) * time.Second
	if expires <= 0 {
		expires = time.Hour // 默认 1h（慢网大文件友好；上限见下方校验）
	}
	// 上限 24h（S3 允许最长 7 天；控制台场景收紧）。
	maxExpires := 24 * time.Hour
	if expires > maxExpires {
		expires = maxExpires
	}

	switch strings.ToLower(req.Method) {
	case "get":
		u, err := client.PresignGetVersion(r.Context(), bucket, req.Key, req.VersionID, expires)
		h.writePresignResult(w, err, map[string]any{"method": "get", "bucket": bucket, "key": req.Key, "url": u, "expiresIn": int64(expires.Seconds())})
	case "post":
		post, err := client.PresignPost(r.Context(), bucket, req.Key, expires)
		var body map[string]any
		if err == nil {
			body = map[string]any{
				"method": "post", "bucket": bucket, "key": req.Key,
				"url": post.URL, "fields": post.Fields, "expiresIn": int64(expires.Seconds()),
			}
		}
		h.writePresignResult(w, err, body)
	case "put":
		u, err := client.PresignPut(r.Context(), bucket, req.Key, expires)
		h.writePresignResult(w, err, map[string]any{"method": "put", "bucket": bucket, "key": req.Key, "url": u, "expiresIn": int64(expires.Seconds())})
	default:
		h.writeErr(w, http.StatusBadRequest, "invalid method (get|put|post)")
	}
}
