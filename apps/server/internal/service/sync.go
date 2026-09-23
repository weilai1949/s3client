package service

import (
	"context"
	"strings"

	"github.com/weilai1949/s3clinet/apps/server/internal/s3wrap"
)

// CompareMode 选择「源/目标对象视为相等」的判定方式。
//   - etag：MD5/CRC（取决于 S3 后端是否计算）；同字节内容必同 ETag。
//   - size_mtime：大小相同 + LastModified 时间戳相同（精确到秒）。
//   - always：视为不相等，总是复制（强制覆盖）。
type CompareMode string

const (
	CompareETag     CompareMode = "etag"
	CompareSizeTime CompareMode = "size_mtime"
	CompareAlways   CompareMode = "always"
)

// SyncResult 增量同步结果。
type SyncResult struct {
	Scanned   int      `json:"scanned"` // 源侧扫描数
	Skipped   int      `json:"skipped"` // 因 equal 跳过
	Copied    int      `json:"copied"`  // 实际复制数
	Failed    int      `json:"failed"`  // 复制失败数
	FailKeys  []string `json:"failedKeys,omitempty"`
	LastError string   `json:"lastError,omitempty"`
}

// SyncKeys 增量同步 src prefix → dst prefix：
//   - 列举 src 全部对象（递归）
//   - 对每个 src 对象，HEAD dst 对应 key：相等则跳过，不等/缺失则复制
//   - 复用复制内核完成实际复制（同/异端点自动适配）
//
// 列举失败（源/目标端 4xx/5xx、ctx 取消）必须上抛 error：静默返回「扫描 0 个」会让用户
// 以为「无事可做」，而实际上一次对象都没比对过（docs/archive/review-2026-09-19.md §B5）。
func SyncKeys(
	ctx context.Context,
	src, dst *s3wrap.Client,
	srcBucket, srcPrefix, dstBucket, dstPrefix string,
	mode CompareMode,
	workers int,
	onProgress func(Progress),
) (SyncResult, error) {
	if workers < 1 {
		workers = 4
	}
	if mode == "" {
		mode = CompareETag
	}

	// 1. 列举源 prefix 全部 key + 元数据（递归）。
	srcList, err := listAll(ctx, src, srcBucket, srcPrefix)
	if err != nil || ctx.Err() != nil {
		return cancelOrErr(ctx, SyncResult{Scanned: len(srcList)}, err)
	}
	if onProgress != nil {
		onProgress(Progress{Total: len(srcList), Done: 0, Failed: 0})
	}

	// 2. 列举目标 prefix 同名 key + 元数据。
	dstMeta, err := indexDst(ctx, dst, dstBucket, dstPrefix)
	if err != nil || ctx.Err() != nil {
		return cancelOrErr(ctx, SyncResult{Scanned: len(srcList)}, err)
	}

	// 3. 过滤出「需要复制」的 key。
	//
	// dstKeyFor 是**唯一**的「源 key → 目标 key」映射表达式：过滤（比对该 key 是否已一致）
	// 与实际复制必须共用它。此前复制侧把完整源 key 交给 MigrateKeys 裸拼接，等于用了第二个
	// 表达式（dstPrefix + 完整 key），于是「比较看的 key」与「写出的 key」不一致 → 永不收敛。
	dstKeyFor := func(k string) string { return dstPrefix + stripPrefix(k, srcPrefix) }
	toCopy := make([]string, 0, len(srcList))
	skipped := 0
	for _, so := range srcList {
		if isEqual(mode, so, dstMeta[dstKeyFor(so.Key)]) {
			skipped++
			continue
		}
		toCopy = append(toCopy, so.Key)
	}

	if len(toCopy) == 0 {
		if onProgress != nil {
			onProgress(Progress{Total: len(srcList), Done: len(srcList), Failed: 0})
		}
		return SyncResult{Scanned: len(srcList), Skipped: skipped}, nil
	}

	// 4. 复用复制内核完成复制（目标 key 由同一个 dstKeyFor 决定）。
	sameEP := SameEndpoint(src.Endpoint(), src.Region(), dst.Endpoint(), dst.Region())
	out := migrateKeys(ctx, src, dst, srcBucket, dstBucket, toCopy, dstKeyFor, sameEP, workers, onProgress)
	return SyncResult{
		Scanned:   len(srcList),
		Skipped:   skipped,
		Copied:    out.OK,
		Failed:    out.Failed,
		FailKeys:  out.FailKeys,
		LastError: out.LastError,
	}, nil
}

// 列举硬上限：最多 listMaxPages 页、每页 listMaxKeys 个、总量不超过 listMaxTotal。
// 三层上限共同保证「畸形/恶意对端返回不前进的 NextToken」时循环仍然终止。
const (
	listMaxPages = 100
	listMaxKeys  = 1000
	listMaxTotal = 100_000
)

// cancelOrErr 决定「列举未完成」如何回到调用方：调用点在「列举失败」或「ctx 已取消」时进入。
//
// ctx 已取消 = 客户端主动放弃：沿用既有语义返回已完成的部分结果而不报错（响应已无人接收，
// 报错只会污染日志）。其余错误必须上抛——静默返回空列表会把「源端 403/5xx」变成
// 「没有需要同步的对象」，用户看到的是「扫描 0 个，无事可做」（review §B5）。
func cancelOrErr(ctx context.Context, partial SyncResult, err error) (SyncResult, error) {
	if ctx.Err() != nil {
		return partial, nil
	}
	return partial, err
}

// listAll 递归列举 prefix 下全部对象 key（不含 CommonPrefix）。
//
// 错误上抛而不是静默返回空列表：源端 403/5xx 若被吞掉，同步端点会回
// `200 {scanned:0}` 并告诉用户「没有需要复制的内容」（review §B5）。
func listAll(ctx context.Context, c *s3wrap.Client, bucket, prefix string) ([]s3wrap.ObjectItem, error) {
	out := make([]s3wrap.ObjectItem, 0, 256)
	token := ""
	for page := 0; page < listMaxPages; page++ {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		p, err := c.ListObjectsPage(ctx, bucket, prefix, "", token, "", listMaxKeys)
		if err != nil {
			return out, err
		}
		out = append(out, p.Objects...)
		if len(out) >= listMaxTotal {
			return out[:listMaxTotal], nil
		}
		// NextToken 不前进（等于上一页的 token）时必须停，否则外层循环空转。
		if !p.IsTruncated || p.NextToken == "" || p.NextToken == token {
			break
		}
		token = p.NextToken
	}
	return out, nil
}

// indexDst 列举目标 prefix 全部对象元数据；返回 key → ObjectMeta（仅 ETag/Size/LastModified）。
//
// 与 listAll 用同一组硬上限：此前的循环条件是「已收集数 < 10 万」且循环内不查 ctx，
// 对端只要返回不前进的 NextToken，任务就会一直挂到 2h 超时（review §B6）。
func indexDst(ctx context.Context, c *s3wrap.Client, bucket, prefix string) (map[string]*s3wrap.ObjectMeta, error) {
	out := make(map[string]*s3wrap.ObjectMeta)
	token := ""
	for page := 0; page < listMaxPages; page++ {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		p, err := c.ListObjectsPage(ctx, bucket, prefix, "", token, "", listMaxKeys)
		if err != nil {
			return out, err
		}
		for _, o := range p.Objects {
			if len(out) >= listMaxTotal {
				break
			}
			out[o.Key] = &s3wrap.ObjectMeta{
				Size:         o.Size,
				ETag:         o.ETag,
				LastModified: o.LastModified,
			}
		}
		if len(out) >= listMaxTotal {
			break
		}
		if !p.IsTruncated || p.NextToken == "" || p.NextToken == token {
			break
		}
		token = p.NextToken
	}
	return out, nil
}

// isEqual 按 mode 比对 src vs dst 元数据；dst 为 nil 视为不等。
func isEqual(mode CompareMode, src s3wrap.ObjectItem, dst *s3wrap.ObjectMeta) bool {
	if dst == nil {
		return false
	}
	switch mode {
	case CompareAlways:
		return false
	case CompareETag:
		// ETag 为空（罕见）回退到 size 比对。
		if src.ETag == "" || dst.ETag == "" {
			return src.Size == dst.Size
		}
		return src.ETag == dst.ETag
	case CompareSizeTime:
		if src.Size != dst.Size {
			return false
		}
		return src.LastModified.Unix() == dst.LastModified.Unix()
	}
	return false
}

// stripPrefix 去掉 key 起始的 prefix，返回相对路径；prefix 为空时返回原 key。
//
// **段边界校验**：prefix 必须结束在 S3 key 的「/」段边界上才算命中，否则原样返回。
// 否则前缀 "p" 会错误命中 "prefix/x.txt" 并剥成 "refix/x.txt"——该 key 在目标侧永不存在，
// 使增量同步每次都判定「缺失」而反复重拷（docs/archive/review-2026-09-19.md §B2）。
func stripPrefix(key, prefix string) string {
	if prefix == "" {
		return key
	}
	if !strings.HasPrefix(key, prefix) {
		return key
	}
	rest := key[len(prefix):]
	// 恰好等于 prefix：整条就是前缀本身，相对路径为空。
	if rest == "" {
		return ""
	}
	// prefix 自带 "/" 结尾，或紧随其后就是段分隔符，才算段边界命中。
	if !strings.HasSuffix(prefix, "/") && !strings.HasPrefix(rest, "/") {
		return key
	}
	return strings.TrimPrefix(rest, "/")
}
