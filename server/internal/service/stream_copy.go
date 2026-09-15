// Package service 承载与 HTTP 无关的领域编排（迁移流式复制、批量任务等），
// 避免 handler 上帝包继续膨胀。handler 仅做鉴权/解析/响应映射。
package service

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/weilai1949/s3clinet/server/internal/s3wrap"
)

// MaxSinglePutBytes 为 S3 PutObject / CopyObject 真实上限。
const MaxSinglePutBytes int64 = 5_000_000_000

// multipartPartSize 分段大小（64MB）；var 以便单测注入小值覆盖多段路径。
var multipartPartSize int64 = 64 << 20

// maxIdlePartBufs 是池中可驻留的空闲分段缓冲数量上限。
//
// 内存预算：server 容器上限 512M（docker-compose*.yml 的 deploy.resources.limits.memory），
// 一块分段缓冲即 64MB。因此刻意只保留 **1** 块：最坏情况下池的常驻开销 = 64MB
// （容器预算的 1/8），且 channel 池不会被 GC 提前回收，必须用固定小上限来保证可证明的有界性。
// 复用的收益来自「先后复制多个大对象」时省掉每对象一次 64MB 分配；
// 峰值内存仍由同时在飞的 MultipartStreamCopy 数量决定（RunBatch / 迁移有界并发），池不放大峰值。
const maxIdlePartBufs = 1

// partBufPool 是有界的分段缓冲池（channel 容量即上限）。
// 选择 channel（而非 sync.Pool）是为了让「最多驻留 1 块 64MB」可证明、可断言。
var partBufPool = make(chan []byte, maxIdlePartBufs)

// acquirePartBuf 获取一块长度恰为 size 的缓冲：优先复用池中容量足够的块，
// 尺寸不合适的旧块直接丢弃（不回收，避免反复挑拣）。
// 池中只会有 1 块，因此循环最多迭代 1 次即返回新分配。
func acquirePartBuf(size int64) []byte {
	for {
		select {
		case b := <-partBufPool:
			if int64(cap(b)) >= size {
				return b[:size]
			}
			// 旧尺寸块不适用：丢弃并继续找下一块。
		default:
			return make([]byte, size)
		}
	}
}

// releasePartBuf 归还缓冲；池满（已有 1 块空闲）时直接丢弃交给 GC，保证驻留上限。
func releasePartBuf(b []byte) {
	if cap(b) == 0 {
		return
	}
	select {
	case partBufPool <- b[:cap(b)]:
	default:
	}
}

// StreamCopy 跨客户端复制对象；>5GB 或未知大小走 multipart。
func StreamCopy(ctx context.Context, src, dst *s3wrap.Client, srcBucket, srcKey, dstBucket, dstKey string) error {
	stream, err := src.GetObjectStream(ctx, srcBucket, srcKey, "", "")
	if err != nil {
		return err
	}
	defer stream.Body.Close()
	size := int64(-1)
	if stream.ContentLength != nil {
		size = *stream.ContentLength
	}
	if size >= 0 && size <= MaxSinglePutBytes {
		return dst.PutObject(ctx, dstBucket, dstKey, stream.Body, stream.ContentType, stream.Metadata)
	}
	return MultipartStreamCopy(ctx, dst, dstBucket, dstKey, stream.ContentType, stream.Body)
}

// MultipartStreamCopy 将 reader 以 multipart 写入目标。
func MultipartStreamCopy(ctx context.Context, dst *s3wrap.Client, bucket, key, contentType string, body io.Reader) error {
	uploadID, err := dst.CreateMultipartUpload(ctx, bucket, key, contentType)
	if err != nil {
		return err
	}
	var parts []s3wrap.UploadPartSpec
	buf := acquirePartBuf(multipartPartSize)
	defer releasePartBuf(buf) // 所有返回路径（含 abort）都归还
	var partNum int32 = 1
	abort := func() { _ = dst.AbortMultipartUpload(context.Background(), bucket, key, uploadID) }
	for {
		// 上游 ReadFull 不会感知 ctx；UploadPart 用 ctx；这里仅在 UploadPart 返回后判断退出。
		n, readErr := io.ReadFull(body, buf)
		if n > 0 {
			etag, uerr := dst.UploadPart(ctx, bucket, key, uploadID, partNum, bytes.NewReader(buf[:n]))
			if uerr != nil {
				abort()
				return uerr
			}
			parts = append(parts, s3wrap.UploadPartSpec{PartNumber: partNum, ETag: etag})
			partNum++
			if partNum > 10_000 {
				abort()
				return fmt.Errorf("object exceeds multipart part limit (10000 parts)")
			}
		}
		if readErr == io.EOF || readErr == io.ErrUnexpectedEOF {
			break
		}
		if readErr != nil {
			abort()
			return readErr
		}
	}
	if len(parts) == 0 {
		abort()
		return dst.PutObject(ctx, bucket, key, bytes.NewReader(nil), contentType, nil)
	}
	if err := dst.CompleteMultipartUpload(ctx, bucket, key, uploadID, parts); err != nil {
		abort()
		return err
	}
	return nil
}
