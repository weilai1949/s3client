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
// P0-3（docs/archive/review-2026-09-19.md §B2）永不收敛的根因。
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
				// 两类错误回退流式，粒度都是逐 key：
				//   - EntityTooLarge：服务端复制的 5GB 单次上限，流式分段可越；
				//   - AccessDenied：CopyObject 用**源账号**凭证签名，目标桶私有时
				//     必回 403；而流式路径 GET/PUT 各用各凭证本可成功（review R10，
				//     两账号同一 MinIO/RustFS 是主场景）。不回退则整批 key 全失败。
				if merr != nil && (s3wrap.IsEntityTooLarge(merr) || s3wrap.HasErrorCode(merr, "AccessDenied")) {
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
//
// aUseSSL / bUseSSL 是两个账号的 useSSL 开关，只对**无 scheme 的裸端点**参与补全
// （显式 scheme 优先），与建 client 时 s3wrap.New → NormalizeEndpoint 的口径逐字一致：
// 因此「判定同端」等价于「两侧 BaseEndpoint 相同」，不会出现比较说同端、建出的 URL
// 却连不上的漂移。端点为空时开关不参与（未设 BaseEndpoint → SDK 恒走默认 https 端点）。
func SameEndpoint(aEndpoint, aRegion string, aUseSSL bool, bEndpoint, bRegion string, bUseSSL bool) bool {
	na, nb := normalizeForCompare(aEndpoint, aUseSSL), normalizeForCompare(bEndpoint, bUseSSL)
	if na != "" && nb != "" {
		return na == nb
	}
	if na == "" && nb == "" {
		return normalizeRegion(aRegion) == normalizeRegion(bRegion)
	}
	return false
}

// normalizeForCompare 归一化端点，用于 SameEndpoint 的「是否同一服务端」比较。
// useSSL 只改变**无 scheme 输入**的补全结果（补 http 或 https），规则由
// s3wrap.NormalizeEndpoint 唯一提供（建 client 与比较共用同一实现）。
func normalizeForCompare(endpoint string, useSSL bool) string {
	return s3wrap.NormalizeEndpoint(endpoint, useSSL)
}

// normalizeRegion 归一化 region 用于比较（trim + 忽略大小写）。
func normalizeRegion(r string) string {
	return strings.ToLower(strings.TrimSpace(r))
}
