// Package service 承载与 HTTP 无关的领域编排（迁移流式复制、批量任务等），
// 避免 handler 上帝包继续膨胀。handler 仅做鉴权/解析/响应映射。
package service

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/weilai1949/s3client/apps/server/internal/s3wrap"
)

// MaxSinglePutBytes 为 S3 PutObject / CopyObject 真实上限。
const MaxSinglePutBytes int64 = 5_000_000_000

// multipartPartSize 是**大小未知**时的分段大小（64MB），也是段长下限；var 以便单测注入小值。
//
// 它只约束「源以 chunked 返回、拿不到 Content-Length」的对象：那条路径必须整段驻留内存
// （见 MultipartStreamCopy），段长放大即按比例放大缓冲，峰值受 512MB 容器预算约束
// （见 maxIdlePartBufs），故该路径的单对象上限 = 64MB × 10000 = 640GB 是刻意取舍，
// 已登记为 KNOWN_ISSUES #63（已决策 ➖）。
//
// **已知大小**的对象不走这条路径：StreamCopy 走 directMultipartCopy（ADR-014），
// 段体是限定长度的流、不缓冲，段长由 planParts 随对象放大到 5GiB，上限由 S3 单对象
// 5TiB 决定。同端点大对象走 copyLargeSameEndpoint（UploadPartCopy），同样不受此约束。
var multipartPartSize int64 = 64 << 20

// maxMultipartParts 是 S3 分段上传的段号上限（段号 1..10000，第 10000 段合法）。
// var 以便单测注入小值精确覆盖边界；默认值即协议上限（测试断言其未被改动）。
var maxMultipartParts int32 = 10_000

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

// copyBufSize 是流式复制使用的中等缓冲大小（256 KiB）。
//
// io.Copy 的通用缓冲只有 32 KiB，GB 级下载 / 打包时写调用数偏高（每 32 KiB 一次 Write，
// handler 侧每次写还会刷一次写 deadline）。用 256 KiB 把写调用数降到 1/8
// （口径见 docs/archive/server-review-2026-10-12.md S1）。
const copyBufSize = 256 << 10

// maxIdleCopyBufs 是小缓冲池的驻留上限（与 maxIdlePartBufs 同款「可证明有界」口径，
// 不用 sync.Pool 的无界语义）。池驻留是**常量**内存：2 × 256 KiB = 512 KiB，
// 相对 512 MB 容器预算可忽略。
const maxIdleCopyBufs = 2

// copyBufPool 是有界的小缓冲池；channel 容量即驻留上限。
var copyBufPool = make(chan []byte, maxIdleCopyBufs)

// acquireCopyBuf 取一块长度恰为 copyBufSize 的缓冲；池空则新分配。
func acquireCopyBuf() []byte {
	select {
	case b := <-copyBufPool:
		return b[:copyBufSize]
	default:
		return make([]byte, copyBufSize)
	}
}

// releaseCopyBuf 归还缓冲；池满时直接丢弃交给 GC，保证驻留上限。
// 契约：只接受 acquireCopyBuf / CopyBuffered 产出的定长缓冲（长度恒为 copyBufSize）。
func releaseCopyBuf(b []byte) {
	select {
	case copyBufPool <- b[:copyBufSize]:
	default:
	}
}

// CopyBuffered 用有界池化的大缓冲做 io.CopyBuffer：复制语义与 io.Copy 完全一致
// （含 WriterTo / ReaderFrom 捷径与错误原样传播），只是把默认 32 KiB 缓冲换成 256 KiB。
// 流式下载（handler.copyStream）与 ZIP 打包（WriteObjectsZip）共用，避免各自复制一份池。
func CopyBuffered(dst io.Writer, src io.Reader) (int64, error) {
	buf := acquireCopyBuf()
	defer releaseCopyBuf(buf)
	return io.CopyBuffer(dst, src, buf)
}

// singlePutLimit 单次 PUT 阈值（>5GB 或未知大小走分段）；var 以便单测注入小值覆盖直传路径。
var singlePutLimit int64 = MaxSinglePutBytes

// maxPartBytes 单个分段的上限（S3 单段 5GiB）；var 以便单测注入小值。
var maxPartBytes int64 = 5 << 30

// maxPartAttempts 单段最多尝试次数：一次主流 + maxPartAttempts-1 次按偏移 Range 重读。
// SDK 内置重试对不可 seek 的流式 body 无效（第二次尝试会因 RewindStream 失败而报错，ADR-014），
// 因此重试必须由这里自己做。
const maxPartAttempts = 3

// planParts 依对象大小算出段长：段数不超过协议上限（10000），段长落在 [64MiB, 5GiB]。
// 返回的段长乘上段数即为可复制的最大对象体积——上限由 S3 单对象 5TiB 决定，不再由内存约束决定
// （直传不缓冲段，见 directMultipartCopy）。size <= 0（未知）时用默认段长。
func planParts(size int64) (int64, error) {
	if size <= 0 {
		return multipartPartSize, nil
	}
	partSize := (size + int64(maxMultipartParts) - 1) / int64(maxMultipartParts)
	if partSize < multipartPartSize {
		partSize = multipartPartSize
	}
	if partSize > maxPartBytes {
		partSize = maxPartBytes
	}
	if n := (size + partSize - 1) / partSize; n > int64(maxMultipartParts) {
		return 0, errPartLimit(maxMultipartParts, partSize)
	}
	return partSize, nil
}

// runParts 是分段复制/直传共用的骨架：建上传 → 逐段调用 upload → 完成；任何失败都 abort。
// upload 收到 uploadID、段号与 [offset, offset+length)，返回该段 ETag。
func runParts(ctx context.Context, dst *s3wrap.Client, bucket, key, contentType string, metadata map[string]string, size, partSize int64,
	upload func(uploadID string, partNum int32, offset, length int64) (string, error)) error {
	uploadID, err := dst.CreateMultipartUploadWithMeta(ctx, bucket, key, contentType, metadata)
	if err != nil {
		return err
	}
	var parts []s3wrap.UploadPartSpec
	abort := func() { _ = dst.AbortMultipartUpload(context.Background(), bucket, key, uploadID) }
	var partNum int32 = 1
	var offset int64
	for size < 0 || offset < size {
		want := partSize
		if size >= 0 && size-offset < want {
			want = size - offset
		}
		etag, uerr := upload(uploadID, partNum, offset, want)
		if uerr != nil {
			abort()
			return uerr
		}
		parts = append(parts, s3wrap.UploadPartSpec{PartNumber: partNum, ETag: etag})
		offset += want
		partNum++
	}
	if len(parts) == 0 {
		abort()
		return dst.PutObject(ctx, bucket, key, bytes.NewReader(nil), contentType, metadata)
	}
	if err := dst.CompleteMultipartUpload(ctx, bucket, key, uploadID, parts); err != nil {
		abort()
		return err
	}
	return nil
}

// directMultipartCopy 把已知大小的源以分段**直传**到目标：段体是限定长度的流式读，
// 显式带 Content-Length，进程内不驻留整段缓冲（ADR-014）——因此段长可以随对象放大到 5GiB，
// 而内存不随段长增长。主流顺序供给各段；某段失败后主流已不可信，该段及之后的段
// 一律改用 Range GET 按偏移重读。
func directMultipartCopy(ctx context.Context, src, dst *s3wrap.Client, srcBucket, srcKey, dstBucket, dstKey, contentType string, metadata map[string]string, size int64, source io.Reader) error {
	partSize, err := planParts(size)
	if err != nil {
		return err
	}
	stream := source
	upload := func(uploadID string, partNum int32, offset, length int64) (string, error) {
		var lastErr error
		for attempt := 1; attempt <= maxPartAttempts; attempt++ {
			var (
				lr     io.Reader
				closer io.Closer
			)
			if stream != nil {
				lr = stream
			} else {
				// 主流已失效：按偏移 Range 重读该段（源必须是可重读的 S3 对象）。
				st, gerr := src.GetObjectStream(ctx, srcBucket, srcKey, "", fmt.Sprintf("bytes=%d-%d", offset, offset+length-1))
				if gerr != nil {
					lastErr = gerr
					if ctx.Err() != nil {
						return "", ctx.Err()
					}
					continue
				}
				lr, closer = st.Body, st.Body
			}
			// body 是限定长度的流：不可 seek，长度由 ContentLength 显式给出；
			// 源端短于声明长度时由传输层报错（LengthRequired / unexpected EOF），不会静默成功。
			etag, uerr := dst.UploadPart(ctx, dstBucket, dstKey, uploadID, partNum, io.LimitReader(lr, length), length)
			if closer != nil {
				_ = closer.Close()
			}
			if uerr == nil {
				return etag, nil
			}
			lastErr = uerr
			stream = nil // 主流已被部分消费，后续段一律 Range 重读
			if ctx.Err() != nil {
				return "", ctx.Err()
			}
		}
		return "", lastErr
	}
	return runParts(ctx, dst, dstBucket, dstKey, contentType, metadata, size, partSize, upload)
}

// copyLargeSameEndpoint 同端点复制超大对象（CopyObject 报 EntityTooLarge 后的路径）：
// 各段走服务端 UploadPartCopy，数据不经过本进程，内存与带宽均为零。
func copyLargeSameEndpoint(ctx context.Context, src, dst *s3wrap.Client, srcBucket, srcKey, dstBucket, dstKey string) error {
	meta, err := src.HeadObjectMeta(ctx, srcBucket, srcKey, "")
	if err != nil {
		return err
	}
	partSize, err := planParts(meta.Size)
	if err != nil {
		return err
	}
	upload := func(uploadID string, partNum int32, offset, length int64) (string, error) {
		return dst.UploadPartCopy(ctx, srcBucket, srcKey, dstBucket, dstKey, uploadID, partNum, offset, length)
	}
	return runParts(ctx, dst, dstBucket, dstKey, meta.ContentType, meta.Metadata, meta.Size, partSize, upload)
}

// StreamCopy 跨客户端复制对象。已知大小走无缓冲直传（>5GiB 时分段，段长随对象放大）；
// 未知大小走有缓冲的 multipart（段长固定 64MiB，受段数上限约束）。
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
	// 已知大小且超过单 PUT 阈值：直传分段，内存不随段长增长。
	if size > singlePutLimit {
		return directMultipartCopy(ctx, src, dst, srcBucket, srcKey, dstBucket, dstKey, stream.ContentType, stream.Metadata, size, stream.Body)
	}
	// 未知大小（源以 chunked 返回、无 Content-Length）：走有缓冲分段（段长 64MiB），
	// 单对象上限 640GiB 由段数上限约束，超限拒绝而非静默截断（KNOWN_ISSUES #63）。
	if size >= 0 && size <= singlePutLimit {
		return dst.PutObject(ctx, dstBucket, dstKey, stream.Body, stream.ContentType, stream.Metadata)
	}
	return MultipartStreamCopy(ctx, dst, dstBucket, dstKey, stream.ContentType, stream.Metadata, stream.Body)
}

// errPartLimit 段数上限错误。信息带上「段数 × 段大小」，让用户能直接算出可复制的最大对象体积。
func errPartLimit(parts int32, partSize int64) error {
	return fmt.Errorf("object exceeds multipart stream limit (%d parts x %d bytes)", parts, partSize)
}

// MultipartStreamCopy 大小未知时的有缓冲分段路径：段长固定 64MiB，必须整段读入内存，
// 因此受容器内存预算约束（见 maxIdlePartBufs），单对象上限 = 64MiB × 10000。
// 已知大小的对象不走这里（走 directMultipartCopy）。
func MultipartStreamCopy(ctx context.Context, dst *s3wrap.Client, bucket, key, contentType string, metadata map[string]string, body io.Reader) error {
	uploadID, err := dst.CreateMultipartUploadWithMeta(ctx, bucket, key, contentType, metadata)
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
			// 段号上限必须在**上传前**判断：段号 10000 是合法的最后一段，
			// 上传后再判 partNum > 10000 会把「正好 10000 段」的合法对象误判为超限并 abort。
			if partNum > maxMultipartParts {
				abort()
				return errPartLimit(maxMultipartParts, multipartPartSize)
			}
			etag, uerr := dst.UploadPart(ctx, bucket, key, uploadID, partNum, bytes.NewReader(buf[:n]), int64(n))
			if uerr != nil {
				abort()
				return uerr
			}
			parts = append(parts, s3wrap.UploadPartSpec{PartNumber: partNum, ETag: etag})
			partNum++
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
		return dst.PutObject(ctx, bucket, key, bytes.NewReader(nil), contentType, metadata)
	}
	if err := dst.CompleteMultipartUpload(ctx, bucket, key, uploadID, parts); err != nil {
		abort()
		return err
	}
	return nil
}
