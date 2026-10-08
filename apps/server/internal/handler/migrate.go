package handler

import (
	"errors"
	"net/http"

	"github.com/weilai1949/s3client/apps/server/internal/model"
	"github.com/weilai1949/s3client/apps/server/internal/s3wrap"
	"github.com/weilai1949/s3client/apps/server/internal/service"
	"github.com/weilai1949/s3client/apps/server/internal/store"
	"github.com/weilai1949/s3client/apps/server/internal/tracing"
)

// migrate 跨账号迁移：源账号对象 → 目标账号/桶 + 前缀（同步）。
func (h *Handler) migrate(w http.ResponseWriter, r *http.Request) {
	ctx, endSpan := tracing.Start(r.Context(), "migrate")
	defer endSpan()
	req, src, dst, srcClient, dstClient, srcBucket, targetBucket, ok := h.parseMigrateRequest(w, r)
	if !ok {
		return
	}
	sameEP := service.SameEndpoint(src.Endpoint, src.Region, src.UseSSL, dst.Endpoint, dst.Region, dst.UseSSL)
	out := service.MigrateKeys(ctx, srcClient, dstClient, srcBucket, targetBucket, req.SourceKeys, req.TargetPrefix, sameEP, 4, nil)
	h.writeJSON(w, http.StatusOK, migrateBatchJSON(out))
}

func (h *Handler) parseMigrateRequest(w http.ResponseWriter, r *http.Request) (
	req migrateRequest,
	src, dst *model.Account,
	srcClient, dstClient *s3wrap.Client,
	srcBucket, targetBucket string,
	ok bool,
) {
	if err := h.readJSON(r, &req); err != nil {
		h.writeBadJSON(w, err)
		return
	}
	if req.SourceAccountID == "" || req.TargetAccountID == "" {
		h.writeErr(w, http.StatusBadRequest, "sourceAccountId and targetAccountId are required")
		return
	}
	if len(req.SourceKeys) == 0 {
		h.writeErr(w, http.StatusBadRequest, "sourceKeys are required")
		return
	}
	if len(req.SourceKeys) > 10_000 {
		h.writeErr(w, http.StatusBadRequest, "too many keys (max 10000 per request)")
		return
	}
	var err error
	src, err = h.store.Get(req.SourceAccountID)
	if err != nil {
		// store 读取故障（磁盘/解码错误）不是「账号不存在」：一律 404 会误导用户
		// 去查一个本来存在的账号，只有 ErrNotFound 才是 404。
		if !errors.Is(err, store.ErrNotFound) {
			h.writeInternalErr(w, err, "failed to load source account")
			return
		}
		h.writeErr(w, http.StatusNotFound, "source account not found")
		return
	}
	dst, err = h.store.Get(req.TargetAccountID)
	if err != nil {
		if !errors.Is(err, store.ErrNotFound) {
			h.writeInternalErr(w, err, "failed to load target account")
			return
		}
		h.writeErr(w, http.StatusNotFound, "target account not found")
		return
	}
	srcClient, err = h.clients.get(src)
	if err != nil {
		h.log.Debug("migrate source client", "err", err)
		h.writeErr(w, http.StatusBadRequest, "invalid source account configuration")
		return
	}
	dstClient, err = h.clients.get(dst)
	if err != nil {
		h.log.Debug("migrate target client", "err", err)
		h.writeErr(w, http.StatusBadRequest, "invalid target account configuration")
		return
	}
	srcBucket = req.SourceBucket
	if srcBucket, ok = h.bucketOr(w, src, srcBucket); !ok {
		return
	}
	targetBucket = req.TargetBucket
	if targetBucket, ok = h.bucketOr(w, dst, targetBucket); !ok {
		return
	}
	ok = true
	return
}
