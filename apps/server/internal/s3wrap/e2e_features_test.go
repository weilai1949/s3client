package s3wrap

// e2e_features_test.go —— ROADMAP §三 #5 三件套的真对端 E2E（RustFS，S3CLIENT_E2E=1 才跑）：
//
//   1. TestE2EConditionalWrite        条件写：服务端 Put/Copy/Delete 条件 + 预签名直传条件头；
//   2. TestE2EChecksumVerify          端到端校验和：etag-md5 回退、复制物化 CRC64NVME/SHA256、
//                                     本地重算与存储端比对（绑定真实服务端返回值）；
//   3. TestE2EObjectLock              Object Lock：桶配置 / 保留期 / 法定保留读写与降级形态。
//
// 断言口径（与 compatibility.md §6.1 第 4 条一致）：只断言**协议交互与降级形态**，
// 不在 RustFS 上断言 AWS S3 官方行为（实测 RustFS 删除不强制保留、GOVERNANCE→COMPLIANCE
// 保留升级被拒——属厂商差异，不在本文件做行为假设）。

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// e2eHeadETag 读取对象当前 ETag（条件写用例必须用**新鲜** ETag，避免覆盖后陈旧）。
func e2eHeadETag(t *testing.T, ctx context.Context, c *Client, bucket, key string) string {
	t.Helper()
	meta, err := c.HeadObjectMeta(ctx, bucket, key, "")
	if err != nil {
		t.Fatalf("head %s: %v", key, err)
	}
	return meta.ETag
}

// e2ePutStatus 对预签名 URL 发 PUT（可携带任意请求头），返回状态码与响应体。
func e2ePutStatus(t *testing.T, url string, body []byte, headers map[string]string) (int, string) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("put %s: %v", url, err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

// e2eCreateBucketWithLock 建桶时启用 Object Lock（CreateBucketInput 的专用字段）。
func e2eCreateBucketWithLock(ctx context.Context, c *Client, bucket string) error {
	_, err := c.s3.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(bucket), ObjectLockEnabledForBucket: aws.Bool(true),
	})
	return err
}

// wantPrecondition 断言错误为 PreconditionFailed（条件写不满足的稳定语义）。
func wantPrecondition(t *testing.T, err error, op string) {
	t.Helper()
	if !HasErrorCode(err, "PreconditionFailed") {
		t.Fatalf("%s: err = %v, want PreconditionFailed", op, err)
	}
}

// TestE2EConditionalWrite 真对端条件写全链路。
func TestE2EConditionalWrite(t *testing.T) {
	e2eSkip(t)
	acc, _ := e2eAccount("e2e-cond")
	ctx := context.Background()
	c, err := New(acc)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	bucket := fmt.Sprintf("s3c-e2ec-%d", time.Now().UnixNano())
	defer func() {
		if err := cleanupBucket(ctx, c, bucket); err != nil {
			t.Logf("cleanup %s: %v", bucket, err)
		}
	}()
	if err := c.CreateBucket(ctx, bucket, "us-east-1", "private"); err != nil {
		t.Fatalf("create bucket: %v", err)
	}

	// 1) If-None-Match:"*" 只创建
	if err := c.PutObjectCond(ctx, bucket, "cond.txt", strings.NewReader("v1"), "", nil,
		Conditions{IfNoneMatch: "*"}); err != nil {
		t.Fatalf("create-only put (new): %v", err)
	}
	err = c.PutObjectCond(ctx, bucket, "cond.txt", strings.NewReader("v2"), "", nil,
		Conditions{IfNoneMatch: "*"})
	wantPrecondition(t, err, "create-only put on existing")

	// 2) If-Match 乐观锁（每次使用前重新读 ETag；wrong etag 不需要真实值）
	err = c.PutObjectCond(ctx, bucket, "cond.txt", strings.NewReader("v3"), "", nil,
		Conditions{IfMatch: `"deadbeef"`})
	wantPrecondition(t, err, "If-Match wrong etag")
	etag := e2eHeadETag(t, ctx, c, bucket, "cond.txt")
	if err := c.PutObjectCond(ctx, bucket, "cond.txt", strings.NewReader("v4"), "", nil,
		Conditions{IfMatch: etag}); err != nil {
		t.Fatalf("If-Match right etag: %v", err)
	}

	// 3) 目标端条件复制
	if err := c.CopyObjectCond(ctx, bucket, "cond.txt", bucket, "cond-copy.txt",
		CopyOptions{Conditions: Conditions{IfNoneMatch: "*"}, ChecksumAlgorithm: "CRC64NVME"}); err != nil {
		t.Fatalf("conditional copy (new dest): %v", err)
	}
	err = c.CopyObjectCond(ctx, bucket, "cond.txt", bucket, "cond-copy.txt",
		CopyOptions{Conditions: Conditions{IfNoneMatch: "*"}})
	wantPrecondition(t, err, "conditional copy on existing dest")

	// 4) 条件删除（新鲜 ETag；wrong etag 不需要真实值）
	err = c.DeleteObjectCond(ctx, bucket, "cond.txt", Conditions{IfMatch: `"deadbeef"`})
	wantPrecondition(t, err, "conditional delete wrong etag")
	etag = e2eHeadETag(t, ctx, c, bucket, "cond.txt")
	if err := c.DeleteObjectCond(ctx, bucket, "cond.txt", Conditions{IfMatch: etag}); err != nil {
		t.Fatalf("conditional delete right etag: %v", err)
	}

	// 5) 预签名直传条件：条件头回显 + 参与签名（X-Amz-SignedHeaders）
	res, err := c.PresignPut(ctx, bucket, "presign-cond.txt", 10*time.Minute,
		Conditions{IfNoneMatch: "*"})
	if err != nil {
		t.Fatalf("presign put: %v", err)
	}
	if !strings.Contains(res.URL, "if-none-match") {
		t.Fatalf("signed headers must include if-none-match: %s", res.URL)
	}
	if status, body := e2ePutStatus(t, res.URL, []byte("first"), res.Headers); status != http.StatusOK {
		t.Fatalf("presigned conditional put (new) status=%d body=%s", status, body)
	}
	if status, body := e2ePutStatus(t, res.URL, []byte("second"), res.Headers); status != http.StatusPreconditionFailed {
		t.Fatalf("presigned conditional put (existing) status=%d, want 412 body=%s", status, body)
	}

	t.Logf("E2E conditional write OK (bucket=%s)", bucket)
}

// TestE2EChecksumVerify 真对端校验和：读侧暴露 + 复制物化 + 本地重算比对。
func TestE2EChecksumVerify(t *testing.T) {
	e2eSkip(t)
	acc, _ := e2eAccount("e2e-ck")
	ctx := context.Background()
	c, err := New(acc)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	bucket := fmt.Sprintf("s3c-e2ec-ck-%d", time.Now().UnixNano())
	defer func() {
		if err := cleanupBucket(ctx, c, bucket); err != nil {
			t.Logf("cleanup %s: %v", bucket, err)
		}
	}()
	if err := c.CreateBucket(ctx, bucket, "us-east-1", "private"); err != nil {
		t.Fatalf("create bucket: %v", err)
	}
	payload := "checksum probe payload"

	// 1) 未物化校验和的单段对象 → etag-md5 尽力比对（RustFS 单 PUT 的 ETag = MD5）
	if err := c.PutObject(ctx, bucket, "plain.txt", strings.NewReader(payload), "", nil); err != nil {
		t.Fatalf("put plain: %v", err)
	}
	v, err := c.VerifyObjectChecksum(ctx, bucket, "plain.txt", "")
	if err != nil {
		t.Fatalf("verify plain: %v", err)
	}
	if v.Method != "etag-md5" || !v.Match {
		t.Fatalf("plain verify = %+v, want etag-md5 match", v)
	}

	// 2) 复制物化 CRC64NVME → Head 暴露 → 端到端比对（绑定真实服务端返回值）
	if err := c.CopyObjectCond(ctx, bucket, "plain.txt", bucket, "crc.txt",
		CopyOptions{ChecksumAlgorithm: "CRC64NVME"}); err != nil {
		t.Fatalf("copy with crc64nvme: %v", err)
	}
	meta, err := c.HeadObjectMeta(ctx, bucket, "crc.txt", "")
	if err != nil {
		t.Fatalf("head crc: %v", err)
	}
	if meta.Checksums == nil || meta.Checksums.CRC64NVME == "" {
		t.Fatalf("head checksums = %+v, want crc64nvme exposed", meta.Checksums)
	}
	if meta.Checksums.CRC64NVME != "N4bktbEKNg8=" {
		t.Fatalf("server crc64nvme = %q, want known vector N4bktbEKNg8=", meta.Checksums.CRC64NVME)
	}
	v, err = c.VerifyObjectChecksum(ctx, bucket, "crc.txt", "")
	if err != nil {
		t.Fatalf("verify crc: %v", err)
	}
	if v.Method != "crc64nvme" || !v.Match || v.Local != "N4bktbEKNg8=" {
		t.Fatalf("crc64nvme verify = %+v, want match with known vector", v)
	}

	// 3) SHA256 阶梯
	if err := c.CopyObjectCond(ctx, bucket, "plain.txt", bucket, "sha.txt",
		CopyOptions{ChecksumAlgorithm: "SHA256"}); err != nil {
		t.Fatalf("copy with sha256: %v", err)
	}
	v, err = c.VerifyObjectChecksum(ctx, bucket, "sha.txt", "")
	if err != nil {
		t.Fatalf("verify sha: %v", err)
	}
	if v.Method != "sha256" || !v.Match {
		t.Fatalf("sha256 verify = %+v, want match", v)
	}

	t.Logf("E2E checksum verify OK (bucket=%s)", bucket)
}

// TestE2EObjectLock 真对端 Object Lock：配置读写、保留期、法定保留与降级形态。
func TestE2EObjectLock(t *testing.T) {
	e2eSkip(t)
	acc, _ := e2eAccount("e2e-lock")
	ctx := context.Background()
	c, err := New(acc)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	bucket := fmt.Sprintf("s3c-e2el-%d", time.Now().UnixNano())
	var retentionUntil time.Time
	// 保留期取「短窗 + 到期收尾」：RustFS 对 GOVERNANCE 保留做**真实强制**（版本删除 403）
	// 且保留期一经设置**不可修改**（再 PUT → 405 MethodNotAllowed，2026-10-08 实测），
	// 所以用 10 秒短窗：断言强制行为后等到期再清理，否则对象永不可删、桶必泄漏。
	retentionWindow := 10 * time.Second
	defer func() {
		if wait := time.Until(retentionUntil.Add(time.Second)); wait > 0 {
			time.Sleep(wait)
		}
		if err := cleanupBucket(ctx, c, bucket); err != nil {
			t.Logf("cleanup %s: %v", bucket, err)
		}
	}()

	// 1) 建桶即启用 Object Lock → 读到配置（此时未设默认保留）
	if err := e2eCreateBucketWithLock(ctx, c, bucket); err != nil {
		t.Fatalf("create lock bucket: %v", err)
	}
	cfg, err := c.GetObjectLockConfiguration(ctx, bucket)
	if err != nil {
		t.Fatalf("get lock config: %v", err)
	}
	if !cfg.Enabled {
		t.Fatalf("config = %+v, want Enabled", cfg)
	}

	// 2) 对象先创建（尚未继承默认保留）→ 读保留期必须降级为 nil
	if err := c.PutObject(ctx, bucket, "locked.txt", strings.NewReader("x"), "", nil); err != nil {
		t.Fatalf("put obj: %v", err)
	}
	ret, err := c.GetObjectRetention(ctx, bucket, "locked.txt", "")
	if err != nil {
		t.Fatalf("get retention (none): %v", err)
	}
	if ret != nil {
		t.Fatalf("retention = %+v, want nil before set", ret)
	}

	// 3) 默认保留策略写 → 读回
	want := ObjectLockConfig{DefaultRetentionMode: "GOVERNANCE", DefaultRetentionDays: 1}
	if err := c.PutObjectLockConfiguration(ctx, bucket, want); err != nil {
		t.Fatalf("put lock config: %v", err)
	}
	cfg, err = c.GetObjectLockConfiguration(ctx, bucket)
	if err != nil {
		t.Fatalf("get lock config after put: %v", err)
	}
	if !cfg.Enabled || cfg.DefaultRetentionMode != want.DefaultRetentionMode ||
		cfg.DefaultRetentionDays != want.DefaultRetentionDays {
		t.Fatalf("config = %+v, want %+v", cfg, want)
	}

	// 4) 对象保留期写 → 读回（短窗）
	retentionUntil = time.Now().UTC().Add(retentionWindow).Truncate(time.Second)
	if err := c.PutObjectRetention(ctx, bucket, "locked.txt", "", "GOVERNANCE", retentionUntil); err != nil {
		t.Fatalf("put retention: %v", err)
	}
	ret, err = c.GetObjectRetention(ctx, bucket, "locked.txt", "")
	if err != nil {
		t.Fatalf("get retention: %v", err)
	}
	if ret == nil || ret.Mode != "GOVERNANCE" || !ret.RetainUntil.Equal(retentionUntil) {
		t.Fatalf("retention = %+v, want GOVERNANCE until %s", ret, retentionUntil)
	}

	// 5) 法定保留 ON → 读回 → OFF（OFF 路径也进真对端；须在删除动作之前——删除后
	//    最新版本是删除标记，未带 versionId 的读会 404）
	if err := c.PutObjectLegalHold(ctx, bucket, "locked.txt", "", "ON"); err != nil {
		t.Fatalf("put legal hold: %v", err)
	}
	if status, err := c.GetObjectLegalHold(ctx, bucket, "locked.txt", ""); err != nil || status != "ON" {
		t.Fatalf("legal hold = %q err=%v, want ON", status, err)
	}
	if err := c.PutObjectLegalHold(ctx, bucket, "locked.txt", "", "OFF"); err != nil {
		t.Fatalf("clear legal hold: %v", err)
	}
	if status, err := c.GetObjectLegalHold(ctx, bucket, "locked.txt", ""); err != nil || status != "OFF" {
		t.Fatalf("legal hold = %q err=%v, want OFF", status, err)
	}

	// 6) 删除强制的两条真实形态（RustFS 实测 / 与 S3 版本化语义一致，E8 实测差异点）：
	//    删除**版本**被 GOVERNANCE 拒绝（403）；当前对象的 DeleteObject 只产生删除标记（200）。
	vid := ""
	page, err := c.ListObjectVersions(ctx, bucket, "", "", "", 100)
	if err != nil {
		t.Fatalf("list versions: %v", err)
	}
	for _, v := range page.Versions {
		if v.Key == "locked.txt" {
			vid = v.VersionID
			break
		}
	}
	if vid == "" {
		t.Fatal("locked.txt version not found for retention enforcement check")
	}
	delErr := c.DeleteObjectVersion(ctx, bucket, "locked.txt", vid)
	if !HasErrorCode(delErr, "AccessDenied") && !HasErrorCode(delErr, "ObjectLocked") {
		t.Fatalf("version delete under retention = %v, want AccessDenied/ObjectLocked", delErr)
	}
	if err := c.DeleteObject(ctx, bucket, "locked.txt"); err != nil {
		t.Fatalf("plain delete (delete marker) = %v, want nil", err)
	}

	// 7) 普通桶降级：读配置 → Enabled=false 且无错误
	plain := bucket + "-p"
	defer func() {
		if err := cleanupBucket(ctx, c, plain); err != nil {
			t.Logf("cleanup %s: %v", plain, err)
		}
	}()
	if err := c.CreateBucket(ctx, plain, "us-east-1", "private"); err != nil {
		t.Fatalf("create plain bucket: %v", err)
	}
	pcfg, err := c.GetObjectLockConfiguration(ctx, plain)
	if err != nil {
		t.Fatalf("get lock config (plain): %v", err)
	}
	if pcfg.Enabled {
		t.Fatalf("plain config = %+v, want Enabled=false", pcfg)
	}
	// 8) 普通桶上写保留 → 真实错误上抛（各厂商码不同，只要求非 nil）
	if err := c.PutObject(ctx, plain, "a.txt", strings.NewReader("x"), "", nil); err != nil {
		t.Fatalf("put plain obj: %v", err)
	}
	if err := c.PutObjectRetention(ctx, plain, "a.txt", "", "GOVERNANCE", time.Now().UTC().Add(time.Hour)); err == nil {
		t.Fatal("put retention on plain bucket must error")
	}

	t.Logf("E2E object lock OK (bucket=%s)", bucket)
}
