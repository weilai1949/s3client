package main

// config_doc_gate_test.go —— 配置文档与源码的一致性门禁。
//
// 起因（2026-09-29）：对照通用 AGENTS.md 模板清点配置面时，发现 README 的配置矩阵**漏登 3 项**
// （`S3C_LOG_JSON` / `S3C_EXPOSE_OPENAPI` / `S3C_CSP_CONNECT_SRC`）——它们只在
// `internal/config/config.go` 与 `.env.example` 里存在，文档读者完全看不到。
// 配置矩阵已抽成 SSOT [`docs/CONFIGURATION.md`]，但「抽表」本身不防漂移：新增一个 `S3C_*`
// 变量而忘改文档，仍然只会静默漏登。故补一道机械门禁把对照关系钉住。
//
// 断言范围与残留（口径同 doc_number_gate_test.go，写在这里以免后来者误判覆盖面）：
//
//   - **只**断言「变量名是否被收录」，**不**校验默认值 / 取值 / 语义描述是否与代码一致
//     ——后者是自然语言，静态门禁只会被文案漂移骗过，属人工审查范围。
//   - 扫描范围限 `internal/config` 的**生产代码**（跳过 `_test.go`：测试里出现的
//     `S3C_TEST_TIMEOUT` 之类是夹具名，不是对外配置项）。
//   - **反向不检查**：文档里多写、而源码不读的变量不会红灯（保留「已废弃项」说明是合法的）。
//   - **已知残留（非假设，实测存在）**：`internal/store/store.go` 的 `New` 也直接读
//     `S3C_STORE_KEY`（`os.Getenv`）。这是**有意设计**而非漏网——`openJSON` 在入参 storeKey
//     为空时回退到 `New`，服务「不传 key、靠环境配置」的调用方，即 `Open` 注释里写明的
//     StoreKey 契约（KNOWN_ISSUES #64 那轮修复的产物）。因此它读的变量名已在上面被覆盖，
//     门禁不会漏；但**将来若有第三个包读一个全新的 `S3C_*` 变量**，它会掉出扫描范围而
//     门禁全绿——届时须把该包纳入扫描（或先把它收口进 `config.FromEnv`）。

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// s3cEnvVarRe 匹配 Go 源码里的 `"S3C_XXX"` 字符串字面量。
var s3cEnvVarRe = regexp.MustCompile(`"(S3C_[A-Z0-9_]+)"`)

// configEnvVarsFromSource 从 `internal/config` 的生产代码抽取全部 `S3C_*` 变量名（去重升序）。
func configEnvVarsFromSource(t *testing.T) []string {
	t.Helper()
	dir := filepath.Join(serverRoot(t), "internal", "config")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("读取 %s: %v", dir, err)
	}
	seen := map[string]bool{}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("读取 %s: %v", name, err)
		}
		for _, m := range s3cEnvVarRe.FindAllStringSubmatch(string(b), -1) {
			seen[m[1]] = true
		}
	}
	out := make([]string, 0, len(seen))
	for v := range seen {
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

// TestConfigDocCoversAllEnvVars 断言配置 SSOT 文档收录了 config 包读取的每一个 `S3C_*` 变量。
func TestConfigDocCoversAllEnvVars(t *testing.T) {
	vars := configEnvVarsFromSource(t)
	// 抽不到任何变量说明解析口径失效（如配置改从结构体 tag 读取）。此时门禁会「全绿但失明」，
	// 必须红灯提示同步口径，而不是安静通过。
	if len(vars) == 0 {
		t.Fatal("未从 internal/config 抽到任何 S3C_* 变量（解析口径需同步）")
	}
	path := filepath.Join(repoRoot(t), "docs", "CONFIGURATION.md")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取 %s: %v", path, err)
	}
	doc := string(b)
	var missing []string
	for _, v := range vars {
		// 词边界匹配：`\bS3C_TOKEN\b` 不会命中 `S3C_TOKEN_EXTRA`（`_` 属词字符，故无边界）。
		if !regexp.MustCompile(`\b` + regexp.QuoteMeta(v) + `\b`).MatchString(doc) {
			missing = append(missing, v)
		}
	}
	if len(missing) > 0 {
		t.Errorf("docs/CONFIGURATION.md（配置 SSOT）漏登 %d 个环境变量：%s\n"+
			"新增 / 改名配置项时必须同 PR 同步该文档与 README 摘要（见 docs/DEVELOPMENT.md §4 文档同步门禁）",
			len(missing), strings.Join(missing, ", "))
	}
}
