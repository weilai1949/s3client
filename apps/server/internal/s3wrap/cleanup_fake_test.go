package s3wrap

// cleanup_fake_test.go —— cleanupBucket（e2e_test.go）的假 S3 收敛测试。
//
// 背景（2026-10-08 真对端事故）：共享 RustFS 上残留 s3c-e2el-* / s3c-probe-*-lock 桶——
// Object Lock 桶里被 GOVERNANCE 保留 / 法定保留拦住的版本，普通版本删除被 403 后
// cleanupBucket 只吞错不重试，循环空转 100 轮报「did not converge」，桶泄漏（对象锁
// 默认保留 1 天，跨重启存活）。本文件把「法保留 OFF + GOVERNANCE bypass 强删」的收敛
// 行为钉在普通 `go test` 里，不依赖 S3CLIENT_E2E。

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

// TestCleanupBucketForceDeletesLockedVersion 被锁版本：先普通删除（403）→ 法定保留 OFF →
// 带 x-amz-bypass-governance-retention 头删版本 → 下一轮列表为空 → 删桶成功。
func TestCleanupBucketForceDeletesLockedVersion(t *testing.T) {
	const vid = "vid-locked"
	var (
		plainDeletes, legalHoldOFF, bypassDeletes int
		versionGone, bucketDeleted                bool
	)
	c, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		switch {
		case r.Method == http.MethodGet && q.Has("versions"):
			w.Header().Set("Content-Type", "application/xml")
			if versionGone {
				_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><ListVersionsResult ` + xmlNS + `><MaxKeys>1000</MaxKeys><IsTruncated>false</IsTruncated></ListVersionsResult>`))
				return
			}
			_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><ListVersionsResult ` + xmlNS + `>
<MaxKeys>1000</MaxKeys><IsTruncated>false</IsTruncated>
<Version><Key>locked.txt</Key><VersionId>` + vid + `</VersionId><IsLatest>true</IsLatest><LastModified>2026-10-08T00:00:00.000Z</LastModified><ETag>&quot;e&quot;</ETag><Size>1</Size></Version>
</ListVersionsResult>`))
		case r.Method == http.MethodPut && q.Has("legal-hold"):
			b, _ := io.ReadAll(r.Body)
			if q.Get("versionId") == vid && strings.Contains(string(b), "<Status>OFF</Status>") {
				legalHoldOFF++
			}
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodDelete && q.Get("versionId") != "":
			if r.Header.Get("x-amz-bypass-governance-retention") == "true" {
				if q.Get("versionId") == vid {
					bypassDeletes++
					versionGone = true
				}
				w.WriteHeader(http.StatusNoContent)
				return
			}
			plainDeletes++
			writeS3Error(w, http.StatusForbidden, "AccessDenied", "retention")
		case r.Method == http.MethodGet && q.Get("list-type") == "2":
			w.Header().Set("Content-Type", "application/xml")
			_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><ListBucketResult ` + xmlNS + `><Name>bkt</Name><KeyCount>0</KeyCount><MaxKeys>1000</MaxKeys><IsTruncated>false</IsTruncated></ListBucketResult>`))
		case r.Method == http.MethodDelete:
			bucketDeleted = true
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	if err := cleanupBucket(context.Background(), c, "bkt"); err != nil {
		t.Fatalf("cleanupBucket: %v", err)
	}
	if plainDeletes != 1 {
		t.Fatalf("plain version deletes = %d, want 1 (普通删除先行)", plainDeletes)
	}
	if legalHoldOFF != 1 {
		t.Fatalf("legal hold OFF = %d, want 1 (删前关法定保留)", legalHoldOFF)
	}
	if bypassDeletes != 1 {
		t.Fatalf("bypass deletes = %d, want 1", bypassDeletes)
	}
	if !bucketDeleted {
		t.Fatal("bucket not deleted")
	}
}

// TestCleanupBucketPlainVersionsNoForce 无锁版本：普通删除一轮清干净，不发法定保留 / bypass 请求。
func TestCleanupBucketPlainVersionsNoForce(t *testing.T) {
	var plainDeletes, legalHoldOFF, bypassDeletes, bucketDeleted int
	c, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		switch {
		case r.Method == http.MethodGet && q.Has("versions"):
			w.Header().Set("Content-Type", "application/xml")
			if plainDeletes >= 3 {
				_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><ListVersionsResult ` + xmlNS + `><MaxKeys>1000</MaxKeys><IsTruncated>false</IsTruncated></ListVersionsResult>`))
				return
			}
			_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><ListVersionsResult ` + xmlNS + `>
<MaxKeys>1000</MaxKeys><IsTruncated>false</IsTruncated>
<Version><Key>t.txt</Key><VersionId>v1</VersionId><IsLatest>true</IsLatest><LastModified>2026-10-08T00:00:00.000Z</LastModified><ETag>&quot;e&quot;</ETag><Size>1</Size></Version>
<Version><Key>t.txt</Key><VersionId>v2</VersionId><IsLatest>false</IsLatest><LastModified>2026-10-08T00:00:00.000Z</LastModified><ETag>&quot;e&quot;</ETag><Size>1</Size></Version>
<DeleteMarker><Key>t.txt</Key><VersionId>dm1</VersionId><IsLatest>true</IsLatest><LastModified>2026-10-08T00:00:01.000Z</LastModified></DeleteMarker>
</ListVersionsResult>`))
		case r.Method == http.MethodPut && q.Has("legal-hold"):
			legalHoldOFF++
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodDelete && q.Get("versionId") != "":
			if r.Header.Get("x-amz-bypass-governance-retention") == "true" {
				bypassDeletes++
			}
			plainDeletes++
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && q.Get("list-type") == "2":
			w.Header().Set("Content-Type", "application/xml")
			_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><ListBucketResult ` + xmlNS + `><Name>bkt</Name><KeyCount>0</KeyCount><MaxKeys>1000</MaxKeys><IsTruncated>false</IsTruncated></ListBucketResult>`))
		case r.Method == http.MethodDelete:
			bucketDeleted++
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	if err := cleanupBucket(context.Background(), c, "bkt"); err != nil {
		t.Fatalf("cleanupBucket: %v", err)
	}
	if plainDeletes != 3 {
		t.Fatalf("plain deletes = %d, want 3 (2 版本 + 1 删除标记)", plainDeletes)
	}
	if legalHoldOFF != 0 || bypassDeletes != 0 {
		t.Fatalf("force requests: hold=%d bypass=%d, want 0/0 (无锁桶不走强删)", legalHoldOFF, bypassDeletes)
	}
	if bucketDeleted != 1 {
		t.Fatalf("bucket deletes = %d, want 1", bucketDeleted)
	}
}
