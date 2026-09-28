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
func SameEndpoint(aEndpoint, aRegion, bEndpoint, bRegion string) bool {
	na, nb := normalizeForCompare(aEndpoint), normalizeForCompare(bEndpoint)
	if na != "" && nb != "" {
		return na == nb
	}
	if na == "" && nb == "" {
		return normalizeRegion(aRegion) == normalizeRegion(bRegion)
	}
	return false
}

// normalizeForCompare 归一化端点，用于 SameEndpoint 的「是否同一服务端」比较。
//
// useSSL 固定传 false 只改变**无 scheme 输入**的补全结果（补 http），对带 scheme
// 的输入无影响；两侧用同一规则归一化，比较是对称的（裸端点一律按 http 互比，
// 不会出现「A 按 http、B 按 https」的单边错配）——这正是建 client 之外的比较口径
// 所需（review Nit「SameEndpoint 硬编码 useSSL=false」的口径说明）。
//
// 已知边界：账号真实 UseSSL 不在 SameEndpoint 签名里（model.Account.UseSSL 只在
// 建 client 时参与 NormalizeEndpoint）。两端点字符串相同/一侧裸写而实际 TLS 配置
// 不同时，会被判为同端而走 CopyObject（写错服务端）或被判异端而走流式（慢但正确）。
// 精确判定需把 useSSL 纳入签名——那是 handler 调用点的跨包契约变更，另行处理。
func normalizeForCompare(endpoint string) string {
	return s3wrap.NormalizeEndpoint(endpoint, false)
}

// normalizeRegion 归一化 region 用于比较（trim + 忽略大小写）。
func normalizeRegion(r string) string {
	return strings.ToLower(strings.TrimSpace(r))
}
