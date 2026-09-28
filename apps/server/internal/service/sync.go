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
	Scanned int `json:"scanned"` // 源侧扫描数
	Skipped int `json:"skipped"` // 因 equal 跳过
	Copied  int `json:"copied"`  // 实际复制数
	Failed  int `json:"failed"`  // 复制失败数
	// Truncated 表示列举结果被安全上限截断（有对象未被枚举）。恒为 bool、
	// 不 omitempty：让「没枚举完」与「对端确实没有更多对象」可区分，否则源侧
	// 第 100_001 个对象静默漏拷、超限的目标对象每次被误判缺失（review R6）。
	Truncated bool     `json:"truncated"`
	FailKeys  []string `json:"failedKeys,omitempty"`
	// FirstError 是首个失败的错误——名副其实：仅在空时写入，永远取第一条
	// （旧名 LastError 名不副实，review Nit）。JSON 名保持历史契约 `lastError`
	// 不变，改的只是 Go 字段名。
	FirstError string `json:"lastError,omitempty"`
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
	srcList, srcTruncated, err := listAll(ctx, src, srcBucket, srcPrefix)
	if err != nil || ctx.Err() != nil {
		return cancelOrErr(ctx, SyncResult{Scanned: len(srcList), Truncated: srcTruncated}, err)
	}
	if onProgress != nil {
		onProgress(Progress{Total: len(srcList), Done: 0, Failed: 0})
	}

	// 2. 列举目标 prefix 同名 key + 元数据。
	dstMeta, dstTruncated, err := indexDst(ctx, dst, dstBucket, dstPrefix)
	if err != nil || ctx.Err() != nil {
		return cancelOrErr(ctx, SyncResult{Scanned: len(srcList), Truncated: srcTruncated || dstTruncated}, err)
	}
	// 任一端截断都必须透出：源侧截断 ⇒ 有对象没被扫描（漏拷）；目标侧截断 ⇒
	// 索引不全（已存在的对象被误判缺失而重拷）。不标记就与「全部比对完」不可区分。
	truncated := srcTruncated || dstTruncated

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
		return SyncResult{Scanned: len(srcList), Skipped: skipped, Truncated: truncated}, nil
	}

	// 4. 复用复制内核完成复制（目标 key 由同一个 dstKeyFor 决定）。
	sameEP := SameEndpoint(src.Endpoint(), src.Region(), dst.Endpoint(), dst.Region())
	out := migrateKeys(ctx, src, dst, srcBucket, dstBucket, toCopy, dstKeyFor, sameEP, workers, onProgress)
	return SyncResult{
		Scanned:    len(srcList),
		Skipped:    skipped,
		Copied:     out.OK,
		Failed:     out.Failed,
		Truncated:  truncated,
		FailKeys:   out.FailKeys,
		FirstError: out.FirstError,
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
//
// truncated=true 表示列举结果被截断（有对象未被枚举），共三种触发：
//  1. 累计超过 listMaxTotal——防御异常/恶意分页让内存与延迟无界；
//  2. 对端声称还有下一页却给不出有效 token（空串，或与上一页相同的 token——
//     后者会让循环空转）；
//  3. 页数上限 listMaxPages 耗尽而对端仍称有下一页。
//
// 截断必须透出（review R6）：静默截断会让「没枚举完」与「对端确实没有更多
// 对象」不可区分——源侧第 100_001 个对象永不同步且无任何信号。
func listAll(ctx context.Context, c *s3wrap.Client, bucket, prefix string) ([]s3wrap.ObjectItem, bool, error) {
	out := make([]s3wrap.ObjectItem, 0, 256)
	token := ""
	for page := 0; page < listMaxPages; page++ {
		if err := ctx.Err(); err != nil {
			return out, false, err
		}
		p, err := c.ListObjectsPage(ctx, bucket, prefix, "", token, "", listMaxKeys)
		if err != nil {
			return out, false, err
		}
		out = append(out, p.Objects...)
		if len(out) > listMaxTotal {
			return out[:listMaxTotal], true, nil
		}
		if !p.IsTruncated {
			return out, false, nil
		}
		// 对端声称未完：已达总量上限（无法安全续页）或 token 缺失/不前进，
		// 都必须停并标记截断——否则要么静默截断，要么循环空转。
		if len(out) >= listMaxTotal || p.NextToken == "" || p.NextToken == token {
			return out, true, nil
		}
		token = p.NextToken
	}
	return out, true, nil
}

// indexDst 列举目标 prefix 全部对象元数据；返回 key → ObjectMeta（仅 ETag/Size/LastModified）
// 以及是否截断（语义同 listAll：有对象因安全上限未被收录即为截断）。
//
// 与 listAll 用同一组硬上限：此前的循环条件是「已收集数 < 10 万」且循环内不查 ctx，
// 对端只要返回不前进的 NextToken，任务就会一直挂到 2h 超时（review §B6）。
// 目标侧截断同样必须透出（review R6）：索引不全会让已存在的对象每次同步被重拷。
func indexDst(ctx context.Context, c *s3wrap.Client, bucket, prefix string) (map[string]*s3wrap.ObjectMeta, bool, error) {
	out := make(map[string]*s3wrap.ObjectMeta)
	token := ""
	for page := 0; page < listMaxPages; page++ {
		if err := ctx.Err(); err != nil {
			return out, false, err
		}
		p, err := c.ListObjectsPage(ctx, bucket, prefix, "", token, "", listMaxKeys)
		if err != nil {
			return out, false, err
		}
		dropped := false
		for _, o := range p.Objects {
			if len(out) >= listMaxTotal {
				dropped = true // 本页还有对象装不进上限：明确截断，不得静默丢弃
				break
			}
			out[o.Key] = &s3wrap.ObjectMeta{
				Size:         o.Size,
				ETag:         o.ETag,
				LastModified: o.LastModified,
			}
		}
		if dropped {
			return out, true, nil
		}
		if len(out) >= listMaxTotal {
			// 正好收满且本页无丢弃：对端还声称有下一页才算截断（正好等于上限且
			// 对端声明列举完成 ⇒ 全部收录，不是截断）。
			return out, p.IsTruncated, nil
		}
		if !p.IsTruncated {
			return out, false, nil
		}
		// 同 listAll：token 缺失/不前进必须停并标记截断，否则空转到 2h 超时。
		if p.NextToken == "" || p.NextToken == token {
			return out, true, nil
		}
		token = p.NextToken
	}
	return out, true, nil
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
		if src.ETag == dst.ETag {
			return true
		}
		// 分段上传的 ETag 形如 "hash-N"（各段 MD5 再 MD5），不是内容的直接摘要：
		// 跨分段数/跨实现（复制通路会重新分段）必然不等 → 旧实现每次同步都判
		// 「不等」而重拷，永不收敛（review Nit）。任一侧为分段形态时退化为 size
		// 比较——与空 ETag 同一保守口径：内容一致 ⇒ 大小一致（收敛）；大小不等
		// 仍照常复制（宁可多传一次，绝不漏同步）。
		if isMultipartETag(src.ETag) || isMultipartETag(dst.ETag) {
			return src.Size == dst.Size
		}
		return false
	case CompareSizeTime:
		if src.Size != dst.Size {
			return false
		}
		return src.LastModified.Unix() == dst.LastModified.Unix()
	}
	return false
}

// isMultipartETag 判断 ETag 是否为分片上传产物（形如 "<hash>-<partCount>"，
// 如 `"0aa1bb-3"`）。partCount 必须非空且全为数字，避免把普通 ETag 里的连字符
// （或畸形输入 "-3"、"abc-"）误判为分片标记而错误退化为 size 比较。
func isMultipartETag(etag string) bool {
	s := strings.Trim(etag, `"`)
	i := strings.LastIndexByte(s, '-')
	if i <= 0 || i == len(s)-1 {
		return false
	}
	for _, ch := range s[i+1:] {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
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
