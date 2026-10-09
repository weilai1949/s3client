package s3wrap

// conditional_test.go —— 条件写（If-Match / If-None-Match）在 s3wrap 边界的透传行为。
// 断言外部可见行为：发给 S3 的请求头、预签名结果的必带头、以及 412/409 错误归类。

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"
)

// TestPutObjectCondForwardsConditionalHeaders 条件写 PUT：条件必须原样变成请求头；
// 无条件时不携带任何条件头（不得有幽灵 If-Match）。
func TestPutObjectCondForwardsConditionalHeaders(t *testing.T) {
	var gotIfMatch, gotIfNone string
	c, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotIfMatch = r.Header.Get("If-Match")
		gotIfNone = r.Header.Get("If-None-Match")
		w.WriteHeader(http.StatusOK)
	}))
	if err := c.PutObjectCond(context.Background(), "bkt", "k.txt", strings.NewReader("x"), "text/plain", nil,
		Conditions{IfMatch: "\"e1\"", IfNoneMatch: "*"}); err != nil {
		t.Fatalf("PutObjectCond: %v", err)
	}
	if gotIfMatch != "\"e1\"" || gotIfNone != "*" {
		t.Fatalf("conditional headers = If-Match %q / If-None-Match %q, want \"e1\" / *", gotIfMatch, gotIfNone)
	}
}

// TestPutObjectWithoutConditionsOmitsHeaders 无条件上传不得携带条件头。
func TestPutObjectWithoutConditionsOmitsHeaders(t *testing.T) {
	var gotIfMatch, gotIfNone string
	c, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotIfMatch = r.Header.Get("If-Match")
		gotIfNone = r.Header.Get("If-None-Match")
		w.WriteHeader(http.StatusOK)
	}))
	if err := c.PutObject(context.Background(), "bkt", "k.txt", strings.NewReader("x"), "", nil); err != nil {
		t.Fatalf("PutObject: %v", err)
	}
	if gotIfMatch != "" || gotIfNone != "" {
		t.Fatalf("unexpected conditional headers: If-Match %q / If-None-Match %q", gotIfMatch, gotIfNone)
	}
}

// TestCopyObjectCondForwardsDestinationConditions 复制的条件作用于**目标**对象。
func TestCopyObjectCondForwardsDestinationConditions(t *testing.T) {
	var gotIfMatch, gotIfNone, gotSource string
	c, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotIfMatch = r.Header.Get("If-Match")
		gotIfNone = r.Header.Get("If-None-Match")
		gotSource = r.Header.Get("x-amz-copy-source")
		w.WriteHeader(http.StatusOK)
	}))
	if err := c.CopyObjectCond(context.Background(), "bkt", "src.txt", "bkt", "dst.txt",
		CopyOptions{Conditions: Conditions{IfMatch: "\"d1\"", IfNoneMatch: "*"}}); err != nil {
		t.Fatalf("CopyObjectCond: %v", err)
	}
	if gotIfNone != "*" || gotIfMatch != "\"d1\"" {
		t.Fatalf("conditional headers = If-Match %q / If-None-Match %q", gotIfMatch, gotIfNone)
	}
	if !strings.Contains(gotSource, "src.txt") {
		t.Fatalf("copy source = %q, want src.txt", gotSource)
	}
}

// TestCopyObjectCondWithChecksumAlgorithm 复制时物化全对象校验和：
// checksumAlgorithm 非空 → x-amz-checksum-algorithm 头必须带上（RustFS 实测会真算并存储）。
func TestCopyObjectCondWithChecksumAlgorithm(t *testing.T) {
	var gotAlg, gotIfNone string
	c, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAlg = r.Header.Get("x-amz-checksum-algorithm")
		gotIfNone = r.Header.Get("If-None-Match")
		w.WriteHeader(http.StatusOK)
	}))
	if err := c.CopyObjectCond(context.Background(), "bkt", "src.txt", "bkt", "dst.txt",
		CopyOptions{Conditions: Conditions{IfNoneMatch: "*"}, ChecksumAlgorithm: "CRC64NVME"}); err != nil {
		t.Fatalf("CopyObjectCond: %v", err)
	}
	if gotAlg != "CRC64NVME" {
		t.Fatalf("x-amz-checksum-algorithm = %q, want CRC64NVME", gotAlg)
	}
	if gotIfNone != "*" {
		t.Fatalf("If-None-Match = %q, want * (条件与校验和可叠加)", gotIfNone)
	}
	// 空 checksumAlgorithm → 不带该头（服务端默认行为不变）
	gotAlg = ""
	c2, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAlg = r.Header.Get("x-amz-checksum-algorithm")
		w.WriteHeader(http.StatusOK)
	}))
	if err := c2.CopyObjectCond(context.Background(), "bkt", "src.txt", "bkt", "dst2.txt",
		CopyOptions{Conditions: Conditions{IfMatch: "\"e1\""}}); err != nil {
		t.Fatalf("CopyObjectCond: %v", err)
	}
	if gotAlg != "" {
		t.Fatalf("empty checksumAlgorithm must omit header, got %q", gotAlg)
	}
}

// TestDeleteObjectCondForwardsIfMatch 条件删除：仅当 ETag 匹配才删除。
func TestDeleteObjectCondForwardsIfMatch(t *testing.T) {
	var gotIfMatch string
	c, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotIfMatch = r.Header.Get("If-Match")
		w.WriteHeader(http.StatusNoContent)
	}))
	if err := c.DeleteObjectCond(context.Background(), "bkt", "k.txt", Conditions{IfMatch: "\"e9\""}); err != nil {
		t.Fatalf("DeleteObjectCond: %v", err)
	}
	if gotIfMatch != "\"e9\"" {
		t.Fatalf("If-Match = %q, want \"e9\"", gotIfMatch)
	}
}

// TestPresignPutEchoesConditionalHeaders 预签名直传：URL 可用，条件头通过 Headers 回显
// 供客户端随 PUT 发送（直传路径浏览器必须显式带上，签名不含这些标准头也会被 S3 求值）。
func TestPresignPutEchoesConditionalHeaders(t *testing.T) {
	c, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	res, err := c.PresignPut(context.Background(), "bkt", "k.txt", time.Hour,
		Conditions{IfMatch: "\"e1\"", IfNoneMatch: "*"})
	if err != nil {
		t.Fatalf("PresignPut: %v", err)
	}
	if res.URL == "" {
		t.Fatal("empty presigned url")
	}
	if res.Headers["If-None-Match"] != "*" || res.Headers["If-Match"] != "\"e1\"" {
		t.Fatalf("echoed headers = %v, want both conditions", res.Headers)
	}
}

// TestPresignPutWithoutConditions 无条件预签名：URL 照常，回显头为空表（响应体形状稳定）。
func TestPresignPutWithoutConditions(t *testing.T) {
	c, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	res, err := c.PresignPut(context.Background(), "bkt", "k.txt", time.Hour, Conditions{})
	if err != nil {
		t.Fatalf("PresignPut: %v", err)
	}
	if res.URL == "" {
		t.Fatal("empty presigned url")
	}
	if len(res.Headers) != 0 {
		t.Fatalf("headers = %v, want empty", res.Headers)
	}
}

// TestPresignPutErrorKeepsEmptyResult 失败路径形状：非 nil 空结果（URL 空、headers 照常回显）
// + err——调用方无需对 err 单开分支（与改造前 (url string, err) 语义一致）。
func TestPresignPutErrorKeepsEmptyResult(t *testing.T) {
	c, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	res, err := c.PresignPut(context.Background(), "bkt", "k.txt", 0, Conditions{IfNoneMatch: "*"})
	if err == nil {
		t.Fatal("want errInvalidExpiry for zero expiry")
	}
	if res == nil || res.URL != "" {
		t.Fatalf("res = %+v, want non-nil empty result", res)
	}
	if res.Headers["If-None-Match"] != "*" {
		t.Fatalf("headers = %v, want echoed condition even on error", res.Headers)
	}
}

// TestConditionalErrorMapping 条件不满足的 S3 错误码 → HTTP 状态与用户文案（对外契约）。
func TestConditionalErrorMapping(t *testing.T) {
	err := fakeAPIError{code: "PreconditionFailed"}
	if got := HTTPStatus(err); got != http.StatusPreconditionFailed {
		t.Fatalf("HTTPStatus(PreconditionFailed) = %d, want 412", got)
	}
	if msg := UserMessage(err); !strings.Contains(msg, "precondition") {
		t.Fatalf("UserMessage(PreconditionFailed) = %q, want mention precondition", msg)
	}
	conflict := fakeAPIError{code: "ConditionalRequestConflict"}
	if got := HTTPStatus(conflict); got != http.StatusConflict {
		t.Fatalf("HTTPStatus(ConditionalRequestConflict) = %d, want 409", got)
	}
	if msg := UserMessage(conflict); !strings.Contains(msg, "conflict") {
		t.Fatalf("UserMessage(ConditionalRequestConflict) = %q, want mention conflict", msg)
	}
}
