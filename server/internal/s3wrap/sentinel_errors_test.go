package s3wrap

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// TestUserMessageSentinelMatching 验证「5GB 单次上限」与「复制成功但删源失败」两类
// 应用层错误改为 sentinel（errors.Is）匹配，而非依赖错误文本子串。
func TestUserMessageSentinelMatching(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"object too large", fmt.Errorf("copy a: %w", ErrObjectTooLarge), "object exceeds 5GB single-put limit; use multipart upload"},
		{"source delete failed", fmt.Errorf("move a: %w", ErrSourceDeleteFailed), "copied but failed to delete source"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := UserMessage(c.err); got != c.want {
				t.Fatalf("UserMessage(%v) = %q, want %q", c.err, got, c.want)
			}
		})
	}

	// 纯文本不再触发映射（证明字符串匹配已移除，避免 SDK/后端文案变化导致静默失效）。
	for _, plain := range []string{
		"put failed: object exceeds 5GB limit",
		"copied ok, failed to delete source object",
	} {
		if got := UserMessage(errors.New(plain)); got != "storage operation failed" {
			t.Fatalf("plain text %q must not be string-matched, got %q", plain, got)
		}
	}
}

// TestPutObjectEntityTooLargeWrapped 真 S3 返回 EntityTooLarge（单次 PutObject >5GB）时，
// 防腐层应把错误归一为 ErrObjectTooLarge：errors.Is 可识别、用户文案为 5GB 提示、HTTP 400。
func TestPutObjectEntityTooLargeWrapped(t *testing.T) {
	c, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeS3Error(w, http.StatusBadRequest, "EntityTooLarge", "Your proposed upload exceeds the maximum allowed size")
	}))
	err := c.PutObject(context.Background(), "bkt", "big.bin", strings.NewReader("payload"), "", nil)
	if err == nil {
		t.Fatal("expected EntityTooLarge error")
	}
	if !errors.Is(err, ErrObjectTooLarge) {
		t.Fatalf("err = %v, want wrapped ErrObjectTooLarge", err)
	}
	if !IsEntityTooLarge(err) {
		t.Fatalf("IsEntityTooLarge(%v) = false", err)
	}
	if got := UserMessage(err); got != "object exceeds 5GB single-put limit; use multipart upload" {
		t.Fatalf("UserMessage = %q", got)
	}
	if got := HTTPStatus(err); got != http.StatusBadRequest {
		t.Fatalf("HTTPStatus = %d, want 400", got)
	}
}

// TestCopyObjectEntityTooLargeWrapped 同上，覆盖 CopyObject 路径（同端点 >5GB 复制）。
func TestCopyObjectEntityTooLargeWrapped(t *testing.T) {
	c, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeS3Error(w, http.StatusBadRequest, "EntityTooLarge", "Copy Source exceeds maximum allowed size")
	}))
	err := c.CopyObject(context.Background(), "src", "big.bin", "dst", "big.bin")
	if err == nil {
		t.Fatal("expected EntityTooLarge error")
	}
	if !errors.Is(err, ErrObjectTooLarge) {
		t.Fatalf("err = %v, want wrapped ErrObjectTooLarge", err)
	}
}
