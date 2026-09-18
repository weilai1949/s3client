package service

import (
	"context"

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
//   - 复用 MigrateKeys 完成实际复制（同/异端点自动适配）
func SyncKeys(
	ctx context.Context,
	src, dst *s3wrap.Client,
	srcBucket, srcPrefix, dstBucket, dstPrefix string,
	mode CompareMode,
	workers int,
	onProgress func(Progress),
) SyncResult {
	if workers < 1 {
		workers = 4
	}
	if mode == "" {
		mode = CompareETag
	}

	// 1. 列举源 prefix 全部 key + 元数据（递归）。
	srcList := listAll(ctx, src, srcBucket, srcPrefix)
	if onProgress != nil {
		onProgress(Progress{Total: len(srcList), Done: 0, Failed: 0})
	}
	// 取消时不再列举目标、不复制：返回当前扫描结果即可。
	if ctx.Err() != nil {
		return SyncResult{Scanned: len(srcList)}
	}

	// 2. 列举目标 prefix 同名 key + 元数据。
	dstMeta := indexDst(ctx, dst, dstBucket, dstPrefix)
	if ctx.Err() != nil {
		return SyncResult{Scanned: len(srcList)}
	}

	// 3. 过滤出「需要复制」的 key。
	toCopy := make([]string, 0, len(srcList))
	skipped := 0
	for _, so := range srcList {
		rel := stripPrefix(so.Key, srcPrefix)
		dstKey := dstPrefix + rel
		if isEqual(mode, so, dstMeta[dstKey]) {
			skipped++
			continue
		}
		toCopy = append(toCopy, so.Key)
	}

	if len(toCopy) == 0 {
		if onProgress != nil {
			onProgress(Progress{Total: len(srcList), Done: len(srcList), Failed: 0})
		}
		return SyncResult{Scanned: len(srcList), Skipped: skipped}
	}

	// 4. 复用 MigrateKeys 完成复制。
	sameEP := SameEndpoint(src.Endpoint(), src.Region(), dst.Endpoint(), dst.Region())
	out := MigrateKeys(ctx, src, dst, srcBucket, dstBucket, toCopy, dstPrefix, sameEP, workers, onProgress)
	return SyncResult{
		Scanned:   len(srcList),
		Skipped:   skipped,
		Copied:    out.OK,
		Failed:    out.Failed,
		FailKeys:  out.FailKeys,
		LastError: out.LastError,
	}
}

// listAll 递归列举 prefix 下全部对象 key（不含 CommonPrefix）。
// 硬上限 100k 防止大桶卡死。
func listAll(ctx context.Context, c *s3wrap.Client, bucket, prefix string) []s3wrap.ObjectItem {
	const maxPages = 100
	const maxKeys = 1000
	const maxTotal = 100_000

	out := make([]s3wrap.ObjectItem, 0, 256)
	token := ""
	for page := 0; page < maxPages; page++ {
		p, err := c.ListObjectsPage(ctx, bucket, prefix, "", token, "", int32(maxKeys))
		if err != nil || p == nil {
			break
		}
		for _, o := range p.Objects {
			out = append(out, o)
			if len(out) >= maxTotal {
				return out
			}
		}
		if !p.IsTruncated || p.NextToken == "" {
			break
		}
		token = p.NextToken
	}
	return out
}

// indexDst 列举目标 prefix 全部对象元数据；返回 key → ObjectMeta（仅 ETag/Size/LastModified）。
// 与 listAll 保持一致的硬上限 100k，防止大桶内存膨胀。
func indexDst(ctx context.Context, c *s3wrap.Client, bucket, prefix string) map[string]*s3wrap.ObjectMeta {
	out := map[string]*s3wrap.ObjectMeta{}
	const maxKeys = 1000
	const maxTotal = 100_000
	token := ""
	for len(out) < maxTotal {
		p, err := c.ListObjectsPage(ctx, bucket, prefix, "", token, "", int32(maxKeys))
		if err != nil || p == nil {
			break
		}
		for _, o := range p.Objects {
			out[o.Key] = &s3wrap.ObjectMeta{
				Size:         o.Size,
				ETag:         o.ETag,
				LastModified: o.LastModified,
			}
			if len(out) >= maxTotal {
				break
			}
		}
		if !p.IsTruncated || p.NextToken == "" {
			break
		}
		token = p.NextToken
	}
	return out
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

// stripPrefix 去掉 key 起始的 prefix，保留相对路径；prefix 为空时返回原 key。
func stripPrefix(key, prefix string) string {
	if prefix == "" {
		return key
	}
	if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
		return key[len(prefix):]
	}
	return key
}
