package store

// review_20260924_test.go —— docs/code-review-2026-09-24.md 分派到 store 域的
// 死代码门禁（R19d）：deriveKeyLegacy 生产零引用；envelope 的 S3C2（无参数头）
// 写入分支生产不可达（写路径恒为 S3C3，旧格式只读不写）。

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestCryptoDeadCodeNotInProductionFiles 源码级门禁：本包**非测试** .go 文件中
// 出现 deriveKeyLegacy 或 envelope(encMagicV2 即红。
// 旧格式读取能力不受影响：parseEnvelope 的 S3C2 分支（连同 legacyParams）保留。
func TestCryptoDeadCodeNotInProductionFiles(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	dir := filepath.Dir(file)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		checked++
		src := string(b)
		if strings.Contains(src, "deriveKeyLegacy") {
			t.Errorf("%s 仍引用 deriveKeyLegacy：生产零引用（review R19d），"+
				"测试侧应写 deriveKey(password, salt, legacyParams)", name)
		}
		if strings.Contains(src, "envelope(encMagicV2") {
			t.Errorf("%s 仍在构造 S3C2 写入信封：V2 写分支生产不可达（review R19d），"+
				"旧格式只读不写", name)
		}
	}
	if checked == 0 {
		t.Fatal("no production .go files scanned; gate is not working")
	}
}
