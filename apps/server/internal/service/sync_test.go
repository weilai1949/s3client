package service

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/weilai1949/s3clinet/apps/server/internal/model"
	"github.com/weilai1949/s3clinet/apps/server/internal/s3wrap"
)

// fakeS3 构造支持 ListObjectsV2 + PutObject + CopyObject 的最小 fake S3。
// src/dst 用不同 bucket 模拟跨桶迁移；端点相同（同 fake server）。
func makeFakePair(t *testing.T) (src, dst *s3wrap.Client, closer func()) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bucket := firstSeg(r.URL.Path)
		key := strings.TrimPrefix(r.URL.Path, "/"+bucket)
		key = strings.TrimPrefix(key, "/")
		switch {
		case r.Method == "GET" && r.URL.Query().Get("list-type") == "2":
			prefix := r.URL.Query().Get("prefix")
			keys := s3FakeStore[bucket]
			var sb strings.Builder
			sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?><ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">`)
			for _, k := range keys {
				if prefix == "" || strings.HasPrefix(k, prefix) {
					tag := s3FakeEtag[bucket+"/"+k]
					fmt.Fprintf(&sb, `<Contents><Key>%s</Key><Size>%d</Size><ETag>"%x"</ETag><LastModified>%s</LastModified><StorageClass>STANDARD</StorageClass></Contents>`,
						k, s3FakeSize[bucket+"/"+k], tag, time.Unix(int64(tag%100000)+1_700_000_000, 0).UTC().Format(time.RFC3339))
				}
			}
			sb.WriteString(`<IsTruncated>false</IsTruncated></ListBucketResult>`)
			w.Header().Set("Content-Type", "application/xml")
			_, _ = w.Write([]byte(sb.String()))
		case r.Method == "PUT":
			if cs := r.Header.Get("X-Amz-Copy-Source"); cs != "" {
				// CopyObject：从源 bucket 复制到当前 bucket。
				srcRef := strings.ReplaceAll(cs, "%2F", "/")
				if idx := strings.Index(srcRef, "/"); idx >= 0 {
					srcBucket := srcRef[:idx]
					srcKey := srcRef[idx+1:]
					if body, ok := s3FakeStore[srcBucket]; ok {
						for _, k := range body {
							if k == srcKey {
								s3FakeStore[bucket] = appendUnique(s3FakeStore[bucket], key)
								s3FakeSize[bucket+"/"+key] = s3FakeSize[srcBucket+"/"+srcKey]
								s3FakeEtag[bucket+"/"+key] = s3FakeEtag[srcBucket+"/"+srcKey]
								w.Header().Set("Content-Type", "application/xml")
								_, _ = w.Write([]byte(`<?xml version="1.0"?><CopyObjectResult><ETag>"x"</ETag></CopyObjectResult>`))
								return
							}
						}
					}
				}
				http.NotFound(w, r)
				return
			}
			s3FakeStore[bucket] = appendUnique(s3FakeStore[bucket], key)
			s3FakeSize[bucket+"/"+key] = 42
			s3FakeEtag[bucket+"/"+key] = simpleHash(bucket + "/" + key)
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	acc1 := &model.Account{
		Endpoint:  srv.URL,
		AccessKey: "AK",
		SecretKey: "SK",
		Region:    "us-east-1",
		Bucket:    "src-bucket",
		PathStyle: true,
	}
	acc2 := *acc1
	acc2.Bucket = "dst-bucket"
	c1, err := s3wrap.New(acc1)
	if err != nil {
		srv.Close()
		t.Fatalf("src client: %v", err)
	}
	c2, err := s3wrap.New(&acc2)
	if err != nil {
		srv.Close()
		t.Fatalf("dst client: %v", err)
	}
	closer = func() {
		srv.Close()
		s3FakeMu.Lock()
		s3FakeStore = map[string][]string{}
		s3FakeSize = map[string]int64{}
		s3FakeEtag = map[string]uint64{}
		s3FakeMu.Unlock()
	}
	return c1, c2, closer
}

func firstSeg(path string) string {
	path = strings.TrimPrefix(path, "/")
	for i := 0; i < len(path); i++ {
		if path[i] == '/' {
			return path[:i]
		}
	}
	return path
}

var (
	s3FakeMu    sync.Mutex
	s3FakeStore = map[string][]string{}
	s3FakeSize  = map[string]int64{}
	s3FakeEtag  = map[string]uint64{}
)

func appendUnique(s []string, v string) []string {
	for _, x := range s {
		if x == v {
			return s
		}
	}
	return append(s, v)
}

func simpleHash(s string) uint64 {
	var h uint64 = 1469598103934665603
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= 1099511628211
	}
	return h
}

// TestSync_PrefixMappingConverges 复现并锁死 P0-3（docs/review-2026-09-19.md §B2）。
//
// 过滤阶段用相对 key 判定（dstPrefix + stripPrefix(so.Key, srcPrefix)），复制阶段却把
// **完整源 key** 交给 MigrateKeys（后者裸拼接 targetPrefix + k）——两处映射表达式不一致：
// 源 p/a.txt、srcPrefix=p/、dstPrefix=q/ 会实际写到 q/p/a.txt，而比较仍在看 q/a.txt，
// 于是二次同步依旧认为「缺失」→ 永远 copied>0，增量同步永不收敛。
func TestSync_PrefixMappingConverges(t *testing.T) {
	s3FakeMu.Lock()
	s3FakeStore = map[string][]string{
		"src-bucket": {"p/a.txt", "p/dir/b.txt"},
		"dst-bucket": {},
	}
	s3FakeSize = map[string]int64{
		"src-bucket/p/a.txt": 10, "src-bucket/p/dir/b.txt": 20,
	}
	s3FakeEtag = map[string]uint64{
		"src-bucket/p/a.txt": 0xaa, "src-bucket/p/dir/b.txt": 0xbb,
	}
	s3FakeMu.Unlock()

	src, dst, closer := makeFakePair(t)
	defer closer()

	run := func() SyncResult {
		return mustSync(t, context.Background(), src, dst, "src-bucket", "p/", "dst-bucket", "q/", CompareETag, 2)
	}

	out := run()
	if out.Copied != 2 || out.Failed != 0 {
		t.Fatalf("首次同步 copied=%d failed=%d, want 2/0", out.Copied, out.Failed)
	}

	// 目标 key 必须是「相对 key 映射」的结果，而不是 q/ + 完整源 key。
	s3FakeMu.Lock()
	got := append([]string(nil), s3FakeStore["dst-bucket"]...)
	s3FakeMu.Unlock()
	sort.Strings(got)
	want := []string{"q/a.txt", "q/dir/b.txt"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("目标 key = %v, want %v（源 key 未被正确映射）", got, want)
	}

	// 验收：二次同步必须收敛（copied == 0）。
	if second := run(); second.Copied != 0 {
		t.Fatalf("二次同步 copied=%d, want 0（增量同步未收敛）", second.Copied)
	}
}

// TestSync_PrefixSegmentBoundary 前缀必须落在段边界上：
// srcPrefix="p" 不得命中 prefix/x.txt（旧实现 stripPrefix 会产出 refix/x.txt 这种被削掉首字母的 key）。
func TestSync_PrefixSegmentBoundary(t *testing.T) {
	s3FakeMu.Lock()
	s3FakeStore = map[string][]string{
		"src-bucket": {"p/a.txt", "prefix/x.txt"},
		"dst-bucket": {},
	}
	s3FakeSize = map[string]int64{
		"src-bucket/p/a.txt": 10, "src-bucket/prefix/x.txt": 30,
	}
	s3FakeEtag = map[string]uint64{
		"src-bucket/p/a.txt": 0xaa, "src-bucket/prefix/x.txt": 0xcc,
	}
	s3FakeMu.Unlock()

	src, dst, closer := makeFakePair(t)
	defer closer()

	out := mustSync(t, context.Background(), src, dst, "src-bucket", "p", "dst-bucket", "q/", CompareETag, 2)
	if out.Copied != 2 || out.Failed != 0 {
		t.Fatalf("copied=%d failed=%d, want 2/0", out.Copied, out.Failed)
	}

	s3FakeMu.Lock()
	got := append([]string(nil), s3FakeStore["dst-bucket"]...)
	s3FakeMu.Unlock()
	sort.Strings(got)
	// p/a.txt → q/a.txt（段边界命中，剥掉前缀）；prefix/x.txt 不在 "p" 段下 → 原样保留，绝不被削成 "refix/x.txt"。
	want := []string{"q/a.txt", "q/prefix/x.txt"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("目标 key = %v, want %v", got, want)
	}

	// 旧实现下 prefix/x.txt 的比较 key 被削成 q/refix/x.txt（永不存在）→ 每次都重拷。
	if second := mustSync(t, context.Background(), src, dst, "src-bucket", "p", "dst-bucket", "q/", CompareETag, 2); second.Copied != 0 {
		t.Fatalf("二次同步 copied=%d, want 0（段边界前缀导致永不收敛）", second.Copied)
	}
}

func TestSync_SkipsEqualByETag(t *testing.T) {
	s3FakeMu.Lock()
	s3FakeStore = map[string][]string{
		"src-bucket": {"a.txt", "b.txt"},
		"dst-bucket": {"a.txt"}, // a.txt 已存在
	}
	s3FakeSize = map[string]int64{
		"src-bucket/a.txt": 10, "dst-bucket/a.txt": 10,
		"src-bucket/b.txt": 20,
	}
	// ETag 一致（a.txt）→ 跳过；b.txt 不在 dst → 复制
	s3FakeEtag = map[string]uint64{
		"src-bucket/a.txt": 0xdead, "dst-bucket/a.txt": 0xdead,
		"src-bucket/b.txt": 0xbeef,
	}
	s3FakeMu.Unlock()

	src, dst, closer := makeFakePair(t)
	defer closer()

	out := mustSync(t, context.Background(), src, dst, "src-bucket", "", "dst-bucket", "", CompareETag, 2)
	if out.Scanned != 2 {
		t.Errorf("scanned = %d, want 2", out.Scanned)
	}
	if out.Skipped != 1 {
		t.Errorf("skipped = %d, want 1 (a.txt equal)", out.Skipped)
	}
	if out.Copied != 1 {
		t.Errorf("copied = %d, want 1 (b.txt new)", out.Copied)
	}
	if out.Failed != 0 {
		t.Errorf("failed = %d, want 0", out.Failed)
	}
}

func TestSync_AlwaysCopies(t *testing.T) {
	s3FakeMu.Lock()
	s3FakeStore = map[string][]string{
		"src-bucket": {"a.txt"},
		"dst-bucket": {"a.txt"},
	}
	s3FakeSize = map[string]int64{
		"src-bucket/a.txt": 10, "dst-bucket/a.txt": 10,
	}
	s3FakeEtag = map[string]uint64{
		"src-bucket/a.txt": 1, "dst-bucket/a.txt": 2, // ETag 不同
	}
	s3FakeMu.Unlock()

	src, dst, closer := makeFakePair(t)
	defer closer()

	out := mustSync(t, context.Background(), src, dst, "src-bucket", "", "dst-bucket", "", CompareETag, 2)
	if out.Copied != 1 {
		t.Errorf("ETag differ → copied = %d, want 1", out.Copied)
	}

	out2 := mustSync(t, context.Background(), src, dst, "src-bucket", "", "dst-bucket", "", CompareAlways, 2)
	if out2.Copied != 1 {
		t.Errorf("CompareAlways → copied = %d, want 1", out2.Copied)
	}
}

func TestSync_EmptySrc(t *testing.T) {
	s3FakeMu.Lock()
	s3FakeStore = map[string][]string{
		"src-bucket": {},
		"dst-bucket": {"existing.txt"},
	}
	s3FakeSize = map[string]int64{}
	s3FakeEtag = map[string]uint64{}
	s3FakeMu.Unlock()

	src, dst, closer := makeFakePair(t)
	defer closer()

	out := mustSync(t, context.Background(), src, dst, "src-bucket", "", "dst-bucket", "", CompareETag, 2)
	if out.Scanned != 0 || out.Skipped != 0 || out.Copied != 0 {
		t.Errorf("empty src: %+v", out)
	}
}

func TestSync_DefaultModeIsETag(t *testing.T) {
	s3FakeMu.Lock()
	s3FakeStore = map[string][]string{"src-bucket": {"x"}, "dst-bucket": {"x"}}
	s3FakeSize = map[string]int64{"src-bucket/x": 1, "dst-bucket/x": 1}
	s3FakeEtag = map[string]uint64{"src-bucket/x": 99, "dst-bucket/x": 99}
	s3FakeMu.Unlock()

	src, dst, closer := makeFakePair(t)
	defer closer()

	out := mustSync(t, context.Background(), src, dst, "src-bucket", "", "dst-bucket", "", "", 2)
	if out.Skipped != 1 {
		t.Errorf("default mode skipped = %d, want 1", out.Skipped)
	}
}

func TestSync_SizeMTimeMode(t *testing.T) {
	s3FakeMu.Lock()
	// 同 size + 同 LastModified → 视为相等；不同则复制。
	s3FakeStore = map[string][]string{"src-bucket": {"a.txt", "b.txt"}, "dst-bucket": {"a.txt", "b.txt"}}
	s3FakeSize = map[string]int64{
		"src-bucket/a.txt": 10, "dst-bucket/a.txt": 10,
		"src-bucket/b.txt": 20, "dst-bucket/b.txt": 21, // size 不同
	}
	// LastModified 由 LastModified%100000+1.7e9 推导 → 同 tag ⇒ 同 mtime
	s3FakeEtag = map[string]uint64{
		"src-bucket/a.txt": 11, "dst-bucket/a.txt": 11, // 同 tag ⇒ 同 mtime ⇒ 视为相等
		"src-bucket/b.txt": 22, "dst-bucket/b.txt": 33, // size 不同 ⇒ 直接不等
	}
	s3FakeMu.Unlock()

	src, dst, closer := makeFakePair(t)
	defer closer()

	out := mustSync(t, context.Background(), src, dst, "src-bucket", "", "dst-bucket", "", CompareSizeTime, 2)
	if out.Skipped != 1 {
		t.Errorf("size_mtime skipped = %d, want 1", out.Skipped)
	}
	if out.Copied != 1 {
		t.Errorf("size_mtime copied = %d, want 1 (b.txt size differ)", out.Copied)
	}
}

func TestStripPrefix(t *testing.T) {
	cases := []struct{ in, prefix, want string }{
		{"a/b.txt", "a/", "b.txt"},
		{"a/b/c.txt", "a/", "b/c.txt"},
		{"x.txt", "", "x.txt"},
		{"a/b.txt", "", "a/b.txt"},
	}
	for _, c := range cases {
		if got := stripPrefix(c.in, c.prefix); got != c.want {
			t.Errorf("stripPrefix(%q, %q) = %q, want %q", c.in, c.prefix, got, c.want)
		}
	}
}

// mustSync 调 SyncKeys 并在列举失败时终止测试；多数用例只关心复制/比对语义。
func mustSync(
	t *testing.T, ctx context.Context,
	src, dst *s3wrap.Client,
	srcBucket, srcPrefix, dstBucket, dstPrefix string,
	mode CompareMode, workers int,
) SyncResult {
	t.Helper()
	out, err := SyncKeys(ctx, src, dst, srcBucket, srcPrefix, dstBucket, dstPrefix, mode, workers, nil)
	if err != nil {
		t.Fatalf("SyncKeys: %v", err)
	}
	return out
}
