package handler

// error_echo_gate_test.go —— 错误文案不回显用户输入（ASSESSMENT L2）。
//
// 背景：`headers.go` 曾把 `ValidateUserMetadata` 的 `err.Error()` 直接回传客户端，
// 消息里带着用户提交的 metadata key（如 `key %q length %d > %d`）；metadata.go /
// objects.go / multipart.go 也有 `"unsupported acl: "+req.ACL` 这类拼接。
// 这些回显会把不可信输入反射进响应体，便于探测与日志污染。
//
// 行为断言（TestSetHeadersInvalidUserMetadata400）只覆盖 user metadata 一条路径；
// 本文件用源码级门禁统一防复发：生产代码里 `writeErr(..., 400, "字面量"+变量)`
// 这类「固定前缀 + 用户输入拼接」一律红灯。确需回显时必须改用固定文案并落日志。

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// clientErrConcatRe 匹配 writeErr 的 400 消息里出现的字符串拼接
// （形如 `writeErr(w, http.StatusBadRequest, "xxx: "+someVar)`）。
var clientErrConcatRe = regexp.MustCompile(`writeErr\([^)]*StatusBadRequest[^)]*"[^"]*"\s*\+`)

func TestClientErrorMessagesDoNotEchoInput(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, e := range entries {
		name := e.Name()
		// 只查生产代码：测试文件允许构造带用户输入的期望消息。
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(".", name))
		if err != nil {
			t.Fatal(err)
		}
		checked++
		// 逐行剔除注释后再匹配，避免注释里举例说明被误判。
		for i, line := range strings.Split(string(b), "\n") {
			code := line
			if idx := strings.Index(code, "//"); idx >= 0 {
				code = code[:idx]
			}
			if clientErrConcatRe.MatchString(code) {
				t.Errorf("%s:%d 400 错误文案拼接了变量（可能回显用户输入）：%s",
					name, i+1, strings.TrimSpace(line))
			}
		}
	}
	if checked == 0 {
		t.Fatal("no production .go files scanned; gate is not working")
	}
}
