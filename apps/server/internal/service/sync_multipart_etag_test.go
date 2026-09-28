package service

// sync_multipart_etag_test.go —— docs/code-review-2026-09-24.md Nit：
// etag 模式对 multipart 对象永不收敛（每次都重拷）。
//
// 分段上传的 ETag 形如 "hash-N"（各段 MD5 的 MD5），跨实现/跨分段大小不可比；
// 我们的复制通路会重新分段并必然产生不同 ETag → 旧实现每次同步都判定「不等」而重拷。
// 修复口径：任一侧 ETag 是分段形态时回退到**大小比较**（内容一致 ⇒ 大小一致，可收敛）。

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/weilai1949/s3clinet/apps/server/internal/model"
	"github.com/weilai1949/s3clinet/apps/server/internal/s3wrap"
)

// multipartObj 是 etag fake 的对象条目：ETag 原样回显（可带引号/分段后缀），
// 不走 makeFakePair 的 uint64 哈希推导——那推不出 "-N" 分段形态。
type multipartObj struct {
	key  string
	size int64
	etag string
}

// makeEtagPair 起一个同时服务 src/dst 两个 bucket 的 fake S3：
// ListObjectsV2 按 bucket 原样回显 ETag/Size；带 Copy-Source 的 PUT 视为服务端复制成功。
// 足以覆盖「内容一致不重拷」（无复制请求）与「该拷则拷」（CopyObject 成功）。
func makeEtagPair(t *testing.T, objs map[string][]multipartObj) (src, dst *s3wrap.Client, closer func()) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bucket := firstSeg(r.URL.Path)
		switch {
		case r.Method == http.MethodGet && r.URL.Query().Get("list-type") == "2":
			prefix := r.URL.Query().Get("prefix")
			var sb strings.Builder
			sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?><ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">`)
			for _, o := range objs[bucket] {
				if prefix != "" && !strings.HasPrefix(o.key, prefix) {
					continue
				}
				fmt.Fprintf(&sb,
					`<Contents><Key>%s</Key><Size>%d</Size><ETag>%s</ETag><LastModified>2024-01-01T00:00:00Z</LastModified><StorageClass>STANDARD</StorageClass></Contents>`,
					o.key, o.size, o.etag)
			}
			sb.WriteString(`<IsTruncated>false</IsTruncated></ListBucketResult>`)
			w.Header().Set("Content-Type", "application/xml")
			_, _ = w.Write([]byte(sb.String()))
		case r.Method == http.MethodPut && r.Header.Get("X-Amz-Copy-Source") != "":
			w.Header().Set("Content-Type", "application/xml")
			_, _ = w.Write([]byte(`<?xml version="1.0"?><CopyObjectResult><ETag>"copied"</ETag></CopyObjectResult>`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	srcAcc := model.Account{
		Endpoint: srv.URL, AccessKey: "AK", SecretKey: "SK",
		Region: "us-east-1", Bucket: "src-bucket", PathStyle: true,
	}
	dstAcc := srcAcc
	dstAcc.Bucket = "dst-bucket"
	c1, err := s3wrap.New(&srcAcc)
	if err != nil {
		t.Fatalf("src client: %v", err)
	}
	c2, err := s3wrap.New(&dstAcc)
	if err != nil {
		t.Fatalf("dst client: %v", err)
	}
	return c1, c2, func() { srv.Close() }
}

// TestIsMultipartETag 分支覆盖：分段 ETag 形如 "<hash>-<partCount>"，
// partCount 必须非空且全为数字；裸连字符、空后缀、前导连字符都不是分段形态。
func TestIsMultipartETag(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{`"0aa1bb-3"`, true},                          // 带引号的分段 ETag（列表原样形态）
		{`9e1b3ca2-11`, true},                         // 多位段数、无引号
		{`"d41d8cd98f00b204e9800998ecf8427e"`, false}, // 单段 ETag
		{`abc`, false},                                // 无连字符
		{`-3`, false},                                 // 连字符在首位（无 hash）
		{`abc-`, false},                               // 段数为空
		{`abc-x`, false},                              // 段数非数字
		{``, false},                                   // 空串
	}
	for _, c := range cases {
		if got := isMultipartETag(c.in); got != c.want {
			t.Errorf("isMultipartETag(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

// TestSync_MultipartETagEqualSizeSkipsCopy 两侧内容一致（同大小）但 ETag 不可比分段对象：
// 不得重拷，且二次同步同样收敛（旧实现 copied=1 且每次都拷）。
func TestSync_MultipartETagEqualSizeSkipsCopy(t *testing.T) {
	src, dst, closer := makeEtagPair(t, map[string][]multipartObj{
		"src-bucket": {{key: "big.bin", size: 100, etag: `"0aa1bb-3"`}},
		"dst-bucket": {{key: "big.bin", size: 100, etag: `"7cc9dd-5"`}},
	})
	defer closer()

	out := mustSync(t, context.Background(), src, dst, "src-bucket", "", "dst-bucket", "", CompareETag, 2)
	if out.Skipped != 1 || out.Copied != 0 {
		t.Fatalf("内容一致的 multipart 对象不得重拷: %+v", out)
	}
	if again := mustSync(t, context.Background(), src, dst, "src-bucket", "", "dst-bucket", "", CompareETag, 2); again.Copied != 0 {
		t.Fatalf("二次同步 copied=%d, want 0（etag 不可比导致永不收敛）", again.Copied)
	}
}

// TestSync_SinglePartSrcMultipartDstSkipsWhenSizeEqual 最常见形态：源是单段对象，
// 目标被我们的分段复制通路写成了 "-N" ETag——两边 ETag 永远不同，大小一致即视为相等。
func TestSync_SinglePartSrcMultipartDstSkipsWhenSizeEqual(t *testing.T) {
	src, dst, closer := makeEtagPair(t, map[string][]multipartObj{
		"src-bucket": {{key: "f.bin", size: 64, etag: `"d41d8cd98f00b204e9800998ecf8427e"`}},
		"dst-bucket": {{key: "f.bin", size: 64, etag: `"9e1b3ca2-11"`}},
	})
	defer closer()

	out := mustSync(t, context.Background(), src, dst, "src-bucket", "", "dst-bucket", "", CompareETag, 2)
	if out.Skipped != 1 || out.Copied != 0 {
		t.Fatalf("单段 vs 分段 ETag、大小一致时应跳过: %+v", out)
	}
}

// TestSync_MultipartETagSizeDiffCopies 大小不同 ⇒ 内容不等：回退大小比较后仍必须复制。
func TestSync_MultipartETagSizeDiffCopies(t *testing.T) {
	src, dst, closer := makeEtagPair(t, map[string][]multipartObj{
		"src-bucket": {{key: "big.bin", size: 100, etag: `"0aa1bb-3"`}},
		"dst-bucket": {{key: "big.bin", size: 200, etag: `"7cc9dd-5"`}},
	})
	defer closer()

	out := mustSync(t, context.Background(), src, dst, "src-bucket", "", "dst-bucket", "", CompareETag, 2)
	if out.Copied != 1 || out.Skipped != 0 {
		t.Fatalf("大小不同的分段对象必须复制: %+v", out)
	}
}
