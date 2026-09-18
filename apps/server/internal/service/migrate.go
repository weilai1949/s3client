package service

import (
	"context"
	"strings"

	"github.com/weilai1949/s3clinet/apps/server/internal/s3wrap"
)

// MigrateKeys 跨账号/桶迁移：同端点优先 CopyObject（EntityTooLarge 回退流式），异端点走 StreamCopy。
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
	if workers < 1 {
		workers = 4
	}
	return RunBatch(ctx, keys, workers,
		func(k string) string { return k },
		func(ctx context.Context, k string) error {
			dstKey := targetPrefix + k
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
