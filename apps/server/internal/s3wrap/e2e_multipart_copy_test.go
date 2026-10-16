package s3wrap

// e2e_multipart_copy_test.go —— ADR-014 新增两条路径的真实对端验证：
// 流式 UploadPart（不可 seek 的 body + 显式 ContentLength）与 UploadPartCopy（服务端分段落拷）。
// 仅当 S3CLIENT_E2E=1 时运行；口径同 e2e_test.go。

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"
	"time"
)

// TestE2EStreamPartAndPartCopy 覆盖无段缓冲直传的底层原语与同端点服务端分段落拷。
func TestE2EStreamPartAndPartCopy(t *testing.T) {
	e2eSkip(t)
	acc, _ := e2eAccount("e2e-mpcopy")
	ctx := context.Background()
	c, err := New(acc)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	bucket := fmt.Sprintf("s3c-e2e-mpc-%d", time.Now().UnixNano())
	if err := c.CreateBucket(ctx, bucket, "us-east-1", "private"); err != nil {
		t.Fatalf("create bucket: %v", err)
	}
	defer func() { _ = cleanupBucket(ctx, c, bucket) }()

	partA := bytes.Repeat([]byte("A"), 5<<20)
	partB := bytes.Repeat([]byte("B"), 5<<20)
	want := append(append([]byte{}, partA...), partB...)

	// 1) 流式 UploadPart：body 不可 seek（io.LimitReader 不是 Seeker），长度必须显式给出。
	//    这是 directMultipartCopy 的底层原语——若显式长度缺失，真实 RustFS 会拒绝该请求。
	uploadID, err := c.CreateMultipartUpload(ctx, bucket, "stream.bin", "application/octet-stream")
	if err != nil {
		t.Fatalf("create multipart: %v", err)
	}
	etagA, err := c.UploadPart(ctx, bucket, "stream.bin", uploadID, 1, io.LimitReader(bytes.NewReader(partA), int64(len(partA))), int64(len(partA)))
	if err != nil {
		t.Fatalf("stream upload part 1: %v", err)
	}
	etagB, err := c.UploadPart(ctx, bucket, "stream.bin", uploadID, 2, io.LimitReader(bytes.NewReader(partB), int64(len(partB))), int64(len(partB)))
	if err != nil {
		t.Fatalf("stream upload part 2: %v", err)
	}
	if err := c.CompleteMultipartUpload(ctx, bucket, "stream.bin", uploadID, []UploadPartSpec{
		{PartNumber: 1, ETag: etagA},
		{PartNumber: 2, ETag: etagB},
	}); err != nil {
		t.Fatalf("complete stream multipart: %v", err)
	}
	if got := getObjectBytes(t, ctx, c, bucket, "stream.bin"); !bytes.Equal(got, want) {
		t.Fatalf("stream multipart body mismatch: got %d bytes, want %d", len(got), len(want))
	}

	// 2) UploadPartCopy：把上面的对象按区间服务端拷成新对象，同时验证
	//    分段初始化时带入的用户元数据在完成后仍然保留。
	up2, err := c.CreateMultipartUploadWithMeta(ctx, bucket, "copied.bin", "application/octet-stream", map[string]string{"color": "red"})
	if err != nil {
		t.Fatalf("create multipart 2: %v", err)
	}
	cp1, err := c.UploadPartCopy(ctx, bucket, "stream.bin", bucket, "copied.bin", up2, 1, 0, int64(len(partA)))
	if err != nil {
		t.Fatalf("upload part copy 1: %v", err)
	}
	cp2, err := c.UploadPartCopy(ctx, bucket, "stream.bin", bucket, "copied.bin", up2, 2, int64(len(partA)), int64(len(partB)))
	if err != nil {
		t.Fatalf("upload part copy 2: %v", err)
	}
	if cp1 == "" || cp2 == "" {
		t.Fatalf("empty copy part etag: %q / %q", cp1, cp2)
	}
	if err := c.CompleteMultipartUpload(ctx, bucket, "copied.bin", up2, []UploadPartSpec{
		{PartNumber: 1, ETag: cp1},
		{PartNumber: 2, ETag: cp2},
	}); err != nil {
		t.Fatalf("complete copied multipart: %v", err)
	}
	if got := getObjectBytes(t, ctx, c, bucket, "copied.bin"); !bytes.Equal(got, want) {
		t.Fatalf("UploadPartCopy body mismatch: got %d bytes, want %d", len(got), len(want))
	}
	meta, err := c.HeadObjectMeta(ctx, bucket, "copied.bin", "")
	if err != nil {
		t.Fatalf("head copied: %v", err)
	}
	if meta.Metadata["color"] != "red" {
		t.Fatalf("copied metadata = %v, want color=red", meta.Metadata)
	}

	// 3) 副本不存在时 UploadPartCopy 必须报错，不得静默成功。
	up3, err := c.CreateMultipartUpload(ctx, bucket, "missing.bin", "")
	if err != nil {
		t.Fatalf("create multipart 3: %v", err)
	}
	if _, err := c.UploadPartCopy(ctx, bucket, "does-not-exist", bucket, "missing.bin", up3, 1, 0, 1024); err == nil {
		t.Fatal("源对象不存在时 UploadPartCopy 必须返回错误")
	}
	_ = c.AbortMultipartUpload(ctx, bucket, "missing.bin", up3)
}
