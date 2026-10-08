package service

import (
	"context"
	"fmt"

	"github.com/weilai1949/s3client/apps/server/internal/s3wrap"
)

// 本文件是删除族的编排层：记账、前缀递归分片、按批删除。
// 与 batch.go 的 RunBatch / CopyKeys 同层——此前删除族编排散在 handler
// （KNOWN_ISSUES #62，2026-09-24 审查 Nit），现与同类编排口径一致。
//
// 分层口径：service 只产出结果与进度；HTTP 校验（bucket/keys/prefix 必填、key 数上限）、
// 状态码、审计、任务注册、**响应体字段拼装**与 failedKeys 的对外裁剪仍留在 handler。

// DeleteCounts 批量删除的累计结果。
//
// S3 的 DeleteObjects 对逐 key 失败仍返回 200，只在响应体内列 <Error>；把「请求数」当作
// 「已删除数」会向用户误报（review §B3）。此处统一按「请求数 − 逐 key 失败数」记账。
//
// 与 BatchResult / SyncResult 同口径：service 只产出结果，JSON 字段由 handler
// 在写出点拼装（见 handler.deleteObjects / deletePrefix）。
type DeleteCounts struct {
	Deleted   int
	Failed    int
	LastError string
}

// Observe 记入一批删除结果：requested 为本批提交的 key 数，failures 为服务端逐 key 拒绝的条目。
func (c *DeleteCounts) Observe(requested int, failures []s3wrap.DeleteFailure) {
	c.Deleted += requested - len(failures)
	c.Failed += len(failures)
	if c.LastError == "" && len(failures) > 0 {
		c.LastError = s3wrap.UserMessageForCode(failures[0].Code)
	}
}

// DeletePrefixResult 是前缀递归删除的汇总（比 DeleteCounts 多一个 truncated）。
type DeletePrefixResult struct {
	Deleted   int
	Failed    int
	Truncated bool
	LastError string
}

// DeleteKeys 单次批量删除（≤1000 key，S3 DeleteObjects 单次上限）并记账。
// 传输层 / 整批失败返回 err（调用方按 S3 错误映射状态码），此时 counts 无意义。
func DeleteKeys(ctx context.Context, client *s3wrap.Client, bucket string, keys []string) (DeleteCounts, error) {
	var counts DeleteCounts
	failures, err := client.DeleteObjects(ctx, bucket, keys)
	if err != nil {
		return counts, err
	}
	counts.Observe(len(keys), failures)
	return counts, nil
}

// RunDeletePrefix 循环列举 + 批量删除前缀下的全部对象（同步）。
// 达到上限即截断并停止，避免畸形/恶意对端把一次同步请求拖成无界循环。
func RunDeletePrefix(
	ctx context.Context, client *s3wrap.Client, bucket, prefix string,
) (DeletePrefixResult, error) {
	counts, truncated, err := deletePrefix(ctx, client, bucket, prefix)
	return DeletePrefixResult{
		Deleted: counts.Deleted, Failed: counts.Failed, Truncated: truncated, LastError: counts.LastError,
	}, err
}

// deletePrefix 是 RunDeletePrefix 的循环内核。
//
// 三道守卫与 sync.go 的 listAll / indexDst 同一形态（review §B6）：
//  1. 页数上限 listMaxPages——空页 + 前进 token 时 maxDelete 的计数永远不增长，
//     只靠计数守卫会一直空转到客户端断连（同步端点还会占住 withStreamLimit 的流槽位）；
//  2. token 缺失或不前进——对端异常时立即停，避免同一页被反复删除、反复计数；
//  3. 循环首行查 ctx——不依赖 SDK 调用间接响应取消。
//
// 进度以「已处理数」（成功 + 逐 key 失败）为准而不是「已删除数」：桶策略/保留期让删除全部
// 失败时，只看 deleted 会让循环永不前进（同一页反复列出、反复失败直到 2h 任务超时）。
func deletePrefix(
	ctx context.Context, client *s3wrap.Client, bucket, prefix string,
) (counts DeleteCounts, truncated bool, err error) {
	const maxDelete = 100_000
	token := ""
	for page := 0; page < listMaxPages; page++ {
		if err := ctx.Err(); err != nil {
			return counts, false, err
		}
		p, listErr := client.ListObjectsPage(ctx, bucket, prefix, "", token, "", listMaxKeys)
		if listErr != nil {
			return counts, false, listErr
		}
		keys := make([]string, 0, len(p.Objects))
		for _, o := range p.Objects {
			keys = append(keys, o.Key)
		}
		if len(keys) > 0 {
			if counts.Deleted+counts.Failed+len(keys) > maxDelete {
				keys = keys[:maxDelete-counts.Deleted-counts.Failed]
				truncated = true
			}
			failures, delErr := client.DeleteObjects(ctx, bucket, keys)
			if delErr != nil {
				return counts, truncated, delErr
			}
			counts.Observe(len(keys), failures)
		}
		if truncated {
			break // 本页有对象被裁掉 ⇒ 没枚举完
		}
		if !p.IsTruncated {
			return counts, false, nil
		}
		// 对端声称未完：已达总量上限、token 缺失或不前进，都必须停并标记截断——
		// 否则要么静默截断（review R6），要么循环空转（review §B6）。
		if counts.Deleted+counts.Failed >= maxDelete || p.NextToken == "" || p.NextToken == token {
			return counts, true, nil
		}
		token = p.NextToken
	}
	return counts, true, nil
}

// DeleteKeysBatched 按 1000 一批删除已列举出的 keys（异步删除任务体）。
//
// 每批结束后回调 onProgress（Status 恒为 running）；终态由调用方在 ctx 结果确定后
// 自行 Finish（与 handler 侧 copy/migrate 任务同一收口方式）。
//
// failKeysLimit 是 API 层的对外回传上限（handler.maxFailKeys）：删除任务的失败 key 数
// 与分片数同量级（可达 10 万），在 service 侧边收集边截断可避免无界增长；上限值本身
// 仍由 API 层注入，service 不发明数字。
func DeleteKeysBatched(
	ctx context.Context, client *s3wrap.Client,
	bucket string, keys []string, failKeysLimit int,
	onProgress func(Progress),
) (DeleteCounts, []string) {
	var counts DeleteCounts
	var failKeys []string
	const batch = 1000
	for i := 0; i < len(keys); i += batch {
		if ctx.Err() != nil {
			break
		}
		end := i + batch
		if end > len(keys) {
			end = len(keys)
		}
		chunk := keys[i:end]
		// chunk ≤ 1000 → 单次 SDK 调用，failures 只可能属于本批。
		failures, delErr := client.DeleteObjects(ctx, bucket, chunk)
		if delErr != nil {
			// 传输层失败：本批 key 的结果未知，全部计入失败。
			counts.Failed += len(chunk)
			if counts.LastError == "" {
				counts.LastError = s3wrap.UserMessage(delErr)
			}
			for _, k := range chunk {
				if len(failKeys) < failKeysLimit {
					failKeys = append(failKeys, k)
				}
			}
		} else {
			counts.Observe(len(chunk), failures)
			for _, f := range failures {
				if len(failKeys) < failKeysLimit {
					failKeys = append(failKeys, f.Key)
				}
			}
		}
		if onProgress != nil {
			onProgress(Progress{
				Done: counts.Deleted + counts.Failed, Total: len(keys),
				OK: counts.Deleted, Failed: counts.Failed,
				Error: counts.LastError, Status: "running",
			})
		}
	}
	return counts, failKeys
}

// MoveKeys 复制成功后再删源（移动），进度回调与 CopyKeys 同形。
func MoveKeys(
	ctx context.Context, client *s3wrap.Client,
	srcBucket, dstBucket string,
	pairs [][2]string, // [srcKey, dstKey]
	workers int,
	onProgress func(Progress),
) BatchResult {
	return RunBatch(ctx, pairs, workers,
		func(p [2]string) string { return p[0] },
		func(ctx context.Context, p [2]string) error {
			if err := client.CopyObject(ctx, srcBucket, p[0], dstBucket, p[1]); err != nil {
				return err
			}
			if err := client.DeleteObject(ctx, srcBucket, p[0]); err != nil {
				// sentinel 包装：批量结果用 errors.Is 识别「移动半成功」，不依赖错误文案。
				return fmt.Errorf("%w: %s: %w", s3wrap.ErrSourceDeleteFailed, p[0], err)
			}
			return nil
		},
		onProgress)
}
