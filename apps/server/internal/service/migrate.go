package service

import (
	"context"
	"strings"

	"github.com/weilai1949/s3clinet/apps/server/internal/s3wrap"
)

// MigrateKeys 跨账号/桶迁移：同端点优先 CopyObject（EntityTooLarge 回退流式），异端点走 StreamCopy。
// 目标 key 为裸前缀拼接（targetPrefix + 源 key）；需要别的映射语义请用 migrateKeys。
func MigrateKeys(
	ctx context.Context,
	src, dst *s3wrap.Client,
	srcBucket, dstBucket string,
	keys []string,
	targetPrefix string,
	sameEP bool,
	workers int,
	onProgress func(Progress),
) BatchResult {
	return migrateKeys(ctx, src, dst, srcBucket, dstBucket, keys,
		func(k string) string { return targetPrefix + k }, sameEP, workers, onProgress)
}

// migrateKeys 是 MigrateKeys 与 SyncKeys 共用的复制内核。
//
// 目标 key 必须由 dstKeyFor 决定（而不是在此处裸拼接目标前缀）：增量同步的目标 key 是
// 「目标前缀 + 相对路径」，与源 key 不是同一个字符串——把两处映射写成两个表达式正是
// P0-3（review-2026-09-19.md §B2）永不收敛的根因。
func migrateKeys(
	ctx context.Context,
	src, dst *s3wrap.Client,
	srcBucket, dstBucket string,
	keys []string,
	dstKeyFor func(string) string,
	sameEP bool,
	workers int,
	onProgress func(Progress),
) BatchResult {
	if workers < 1 {
		workers = 4
	}
	return RunBatch(ctx, keys, workers,
		func(k string) string { return k },
		func(ctx context.Context, k string) error {
			dstKey := dstKeyFor(k)
			var merr error
			if sameEP {
				merr = src.CopyObject(ctx, srcBucket, k, dstBucket, dstKey)
				if merr != nil && s3wrap.IsEntityTooLarge(merr) {
					merr = StreamCopy(ctx, src, dst, srcBucket, k, dstBucket, dstKey)
				}
			} else {
				merr = StreamCopy(ctx, src, dst, srcBucket, k, dstBucket, dstKey)
			}
			return merr
		},
		onProgress)
}

// SameEndpoint 判断两个 S3 配置是否指向同一服务端：
//   - 端点均显式配置：比较端点（去尾斜杠、补全 scheme、忽略大小写），等则视为同端。
//   - 端点均为空（使用 S3 默认端点）：仅当 region 相同才视为同端（不同 region 是不同的 S3 服务）。
//   - 其余情况视为异端。
func SameEndpoint(aEndpoint, aRegion, bEndpoint, bRegion string) bool {
	na, nb := s3wrap.NormalizeEndpoint(aEndpoint, false), s3wrap.NormalizeEndpoint(bEndpoint, false)
	if na != "" && nb != "" {
		return na == nb
	}
	if na == "" && nb == "" {
		return normalizeRegion(aRegion) == normalizeRegion(bRegion)
	}
	return false
}

// normalizeRegion 归一化 region 用于比较（trim + 忽略大小写）。
func normalizeRegion(r string) string {
	return strings.ToLower(strings.TrimSpace(r))
}
