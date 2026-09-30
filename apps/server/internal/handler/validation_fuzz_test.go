package handler

// validation_fuzz_test.go —— 名称/文件名边界校验的原生 fuzz（stdlib `testing.F`，无新依赖）。
//
// 覆盖两个用户输入校验点：
//   - `validBucketName`：桶名 / 前缀输入的第一道边界（文档契约见其头注释）；
//   - `sanitizeFilename`：Content-Disposition 注入防线，输出必须不含路径分隔符 / 引号 / 换行。
//
// 断言方式：对「被接受」的输入断言文档契约成立（拒绝集不参与，避免把实现抄成测试）；
// 以及 sanitize 的净化后不变量。这能抓到 fail-open 的边界（如长度/首尾字符/连续分隔符）。
//
// 运行（有界；CI 见 .github/workflows/fuzz.yml）：
//
//	cd apps/server && go test ./internal/handler/ -run '^$' -fuzz '^FuzzValidBucketName$' -fuzztime=10s
//	cd apps/server && go test ./internal/handler/ -run '^$' -fuzz '^FuzzSanitizeFilename$' -fuzztime=10s

import (
	"strings"
	"testing"
)

// FuzzValidBucketName 验证被接受的桶名满足文档契约。
func FuzzValidBucketName(f *testing.F) {
	for _, seed := range []string{
		"abc", "my-bucket", "a.b.c", "ab", "aBc", "-abc", "abc-", "a..b", "a-.b",
		strings.Repeat("a", 63), strings.Repeat("a", 64),
		"bj-1", "0.0.0.0", "xn--bucket", "a-b", "a_b", "名为桶",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, name string) {
		if !validBucketName(name) {
			return
		}
		if len(name) < 3 || len(name) > 63 {
			t.Fatalf("接受越界长度 %d 的桶名: %q", len(name), name)
		}
		for i, r := range name {
			allowed := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '.'
			if !allowed {
				t.Fatalf("接受非法字符 %q 的桶名: %q", r, name)
			}
			if (i == 0 || i == len(name)-1) && (r == '-' || r == '.') {
				t.Fatalf("接受首尾为 -/. 的桶名: %q", name)
			}
			if i > 0 && (r == '.' || r == '-') && (name[i-1] == '.' || name[i-1] == '-') {
				t.Fatalf("接受连续分隔符的桶名: %q", name)
			}
		}
	})
}

// FuzzSanitizeFilename 验证净化后输出永不含注入字符且非空。
func FuzzSanitizeFilename(f *testing.F) {
	for _, seed := range []string{
		"", "a.txt", "a/b\\c\"d.txt", "line\nbreak.txt", "carriage\rreturn.txt",
		"../etc/passwd", "正常名字.txt", "\r\n", "`;$()|<>", "a b\tc.txt",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, name string) {
		got := sanitizeFilename(name)
		if got == "" {
			t.Fatal("sanitizeFilename 返回空串——Content-Disposition 需要非空回退值")
		}
		if strings.ContainsAny(got, "/\\\"\n\r") {
			t.Fatalf("sanitizeFilename(%q)=%q 仍含路径分隔符/引号/换行", name, got)
		}
	})
}
