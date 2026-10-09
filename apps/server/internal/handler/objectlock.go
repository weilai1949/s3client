package handler

// objectlock.go —— Object Lock（WORM 保留）三组端点：
//
//	bucket/object-lock   桶级配置（默认保留策略）——读 / 写
//	object-retention     对象保留期（GOVERNANCE / COMPLIANCE + 到期时间）——读 / 写
//	object-legal-hold    法定保留（ON / OFF）——读 / 写
//
// 降级口径（厂商支持度差异，见 s3wrap/objectlock.go 与 compatibility.md §6.1）：
// 桶未启用 Object Lock → 读回 enabled=false；对象无保留期 / 未设法定保留 →
// configured=false / status=OFF；写入失败按错误码映射（未启用 400、桶状态冲突 409、
// 锁定中 409、保留期违规 400、厂商未实现 501）。

import (
	"net/http"
	"time"

	"github.com/weilai1949/s3client/apps/server/internal/s3wrap"
)

// allowedRetentionModes 合法保留模式（S3 Object Lock 枚举）。
var allowedRetentionModes = map[string]bool{
	"GOVERNANCE": true,
	"COMPLIANCE": true,
}

// allowedLegalHoldStatuses 合法法定保留状态。
var allowedLegalHoldStatuses = map[string]bool{
	"ON":  true,
	"OFF": true,
}

// getObjectLock 读取桶 Object Lock 配置（未启用 → enabled=false，非错误）。
func (h *Handler) getObjectLock(w http.ResponseWriter, r *http.Request) {
	client, acc, ok := h.accountClient(w, r)
	if !ok {
		return
	}
	bucket, ok := h.bucketOr(w, acc, r.URL.Query().Get("bucket"))
	if !ok {
		return
	}
	cfg, err := client.GetObjectLockConfiguration(r.Context(), bucket)
	if err != nil {
		h.writeInternalErr(w, err, "object lock operation failed")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"bucket": bucket, "enabled": cfg.Enabled,
		"defaultRetentionMode":  cfg.DefaultRetentionMode,
		"defaultRetentionDays":  cfg.DefaultRetentionDays,
		"defaultRetentionYears": cfg.DefaultRetentionYears,
	})
}

// putObjectLock 设置桶默认保留策略（模式 + 天/年二选一；桶须在创建时启用 Object Lock）。
func (h *Handler) putObjectLock(w http.ResponseWriter, r *http.Request) {
	client, acc, ok := h.accountClient(w, r)
	if !ok {
		return
	}
	var req struct {
		Bucket                string `json:"bucket"`
		DefaultRetentionMode  string `json:"defaultRetentionMode"`
		DefaultRetentionDays  int32  `json:"defaultRetentionDays"`
		DefaultRetentionYears int32  `json:"defaultRetentionYears"`
	}
	if err := h.readJSON(r, &req); err != nil {
		h.writeBadJSON(w, err)
		return
	}
	if !allowedRetentionModes[req.DefaultRetentionMode] {
		h.writeErr(w, http.StatusBadRequest, "defaultRetentionMode must be GOVERNANCE or COMPLIANCE")
		return
	}
	if req.DefaultRetentionDays < 0 || req.DefaultRetentionYears < 0 {
		h.writeErr(w, http.StatusBadRequest, "default retention days and years must not be negative")
		return
	}
	if req.DefaultRetentionDays > 0 && req.DefaultRetentionYears > 0 {
		h.writeErr(w, http.StatusBadRequest, "specify only one of defaultRetentionDays or defaultRetentionYears")
		return
	}
	if req.DefaultRetentionDays == 0 && req.DefaultRetentionYears == 0 {
		h.writeErr(w, http.StatusBadRequest, "defaultRetentionDays or defaultRetentionYears is required")
		return
	}
	bucket := req.Bucket
	if bucket, ok = h.bucketOr(w, acc, bucket); !ok {
		return
	}
	cfg := s3wrap.ObjectLockConfig{
		DefaultRetentionMode:  req.DefaultRetentionMode,
		DefaultRetentionDays:  req.DefaultRetentionDays,
		DefaultRetentionYears: req.DefaultRetentionYears,
	}
	if err := client.PutObjectLockConfiguration(r.Context(), bucket, cfg); err != nil {
		// RustFS：在既有桶上启用 Object Lock → InvalidBucketState（409）。
		// 文案固定说明「只能建桶时启用」，不回显错误详情。
		if s3wrap.HasErrorCode(err, "InvalidBucketState") {
			h.writeErr(w, http.StatusConflict, "object lock cannot be enabled on an existing bucket")
			return
		}
		h.writeInternalErr(w, err, "object lock operation failed")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"bucket": bucket, "enabled": true,
		"defaultRetentionMode":  cfg.DefaultRetentionMode,
		"defaultRetentionDays":  cfg.DefaultRetentionDays,
		"defaultRetentionYears": cfg.DefaultRetentionYears,
	})
}

// getObjectRetention 读取对象保留期（无保留期 / 未启用 → configured=false）。
func (h *Handler) getObjectRetention(w http.ResponseWriter, r *http.Request) {
	client, acc, ok := h.accountClient(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	bucket, ok := h.bucketOr(w, acc, q.Get("bucket"))
	if !ok {
		return
	}
	key := q.Get("key")
	if key == "" {
		h.writeErr(w, http.StatusBadRequest, "key is required")
		return
	}
	versionID := q.Get("versionId")
	ret, err := client.GetObjectRetention(r.Context(), bucket, key, versionID)
	if err != nil {
		h.writeInternalErr(w, err, "object retention operation failed")
		return
	}
	resp := map[string]any{
		"bucket": bucket, "key": key, "versionId": versionID,
		"configured": ret != nil, "mode": "", "retainUntilDate": "",
	}
	if ret != nil {
		resp["mode"] = ret.Mode
		resp["retainUntilDate"] = ret.RetainUntil.Format(time.RFC3339)
	}
	h.writeJSON(w, http.StatusOK, resp)
}

// putObjectRetention 设置对象保留期（模式 + 未来时刻的到期时间，RFC3339）。
func (h *Handler) putObjectRetention(w http.ResponseWriter, r *http.Request) {
	client, acc, ok := h.accountClient(w, r)
	if !ok {
		return
	}
	var req struct {
		Bucket          string `json:"bucket"`
		Key             string `json:"key"`
		VersionID       string `json:"versionId"`
		Mode            string `json:"mode"`
		RetainUntilDate string `json:"retainUntilDate"`
	}
	if err := h.readJSON(r, &req); err != nil {
		h.writeBadJSON(w, err)
		return
	}
	if req.Key == "" {
		h.writeErr(w, http.StatusBadRequest, "key is required")
		return
	}
	if !allowedRetentionModes[req.Mode] {
		h.writeErr(w, http.StatusBadRequest, "mode must be GOVERNANCE or COMPLIANCE")
		return
	}
	if req.RetainUntilDate == "" {
		h.writeErr(w, http.StatusBadRequest, "retainUntilDate is required")
		return
	}
	until, err := time.Parse(time.RFC3339, req.RetainUntilDate)
	if err != nil {
		h.writeErr(w, http.StatusBadRequest, "retainUntilDate must be RFC3339, e.g. 2031-02-03T04:05:06Z")
		return
	}
	if !until.After(time.Now()) {
		h.writeErr(w, http.StatusBadRequest, "retainUntilDate must be in the future")
		return
	}
	bucket := req.Bucket
	if bucket, ok = h.bucketOr(w, acc, bucket); !ok {
		return
	}
	until = until.UTC()
	if err := client.PutObjectRetention(r.Context(), bucket, req.Key, req.VersionID, req.Mode, until); err != nil {
		h.writeInternalErr(w, err, "object retention operation failed")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"bucket": bucket, "key": req.Key, "versionId": req.VersionID,
		"configured": true, "mode": req.Mode, "retainUntilDate": until.Format(time.RFC3339),
	})
}

// getObjectLegalHold 读取法定保留状态（未设置 → OFF）。
func (h *Handler) getObjectLegalHold(w http.ResponseWriter, r *http.Request) {
	client, acc, ok := h.accountClient(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	bucket, ok := h.bucketOr(w, acc, q.Get("bucket"))
	if !ok {
		return
	}
	key := q.Get("key")
	if key == "" {
		h.writeErr(w, http.StatusBadRequest, "key is required")
		return
	}
	versionID := q.Get("versionId")
	status, err := client.GetObjectLegalHold(r.Context(), bucket, key, versionID)
	if err != nil {
		h.writeInternalErr(w, err, "legal hold operation failed")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"bucket": bucket, "key": key, "versionId": versionID, "status": status,
	})
}

// putObjectLegalHold 设置法定保留（ON / OFF）。
func (h *Handler) putObjectLegalHold(w http.ResponseWriter, r *http.Request) {
	client, acc, ok := h.accountClient(w, r)
	if !ok {
		return
	}
	var req struct {
		Bucket    string `json:"bucket"`
		Key       string `json:"key"`
		VersionID string `json:"versionId"`
		Status    string `json:"status"`
	}
	if err := h.readJSON(r, &req); err != nil {
		h.writeBadJSON(w, err)
		return
	}
	if req.Key == "" {
		h.writeErr(w, http.StatusBadRequest, "key is required")
		return
	}
	if !allowedLegalHoldStatuses[req.Status] {
		h.writeErr(w, http.StatusBadRequest, "status must be ON or OFF")
		return
	}
	bucket := req.Bucket
	if bucket, ok = h.bucketOr(w, acc, bucket); !ok {
		return
	}
	if err := client.PutObjectLegalHold(r.Context(), bucket, req.Key, req.VersionID, req.Status); err != nil {
		h.writeInternalErr(w, err, "legal hold operation failed")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"bucket": bucket, "key": req.Key, "versionId": req.VersionID, "status": req.Status,
	})
}
