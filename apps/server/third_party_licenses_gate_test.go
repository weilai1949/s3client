package main

// third_party_licenses_gate_test.go —— `docs/THIRD_PARTY_LICENSES.md` 的**覆盖门禁**。
//
// 背景（2026-09-29 第 7 层「安全与供应链」盘点）：仓库此前没有任何第三方依赖 / 许可证清单。
// 手写必然漂移，故清单由 `scripts/gen-third-party-licenses.sh` 从权威来源**生成**：
//   Go   ← `go list -m -json all`（模块图）+ 模块目录内 LICENSE 文本机械识别
//   Rust ← `apps/desktop/src-tauri/Cargo.lock`（配合 cargo metadata / registry 缓存取 license）
//   npm  ← `apps/web/package.json` 的 dependencies
//
// 但「有生成脚本」不等于「清单是最新的」——加了依赖却忘记重新生成，清单照样缺项而无人察觉。
// 本门禁把「依赖图 ↔ 清单」的对应关系钉住。
//
// 断言范围（刻意不做的事）：
//   - **不**做逐字节比对：清单里的 license 列依赖本地 registry / 模块缓存（cargo metadata 失败时
//     退回离线兜底），不同机器上文字可能不同；逐字节比会 flaky。这里只断言**枚举完整性**：
//     依赖图里的每一个「名字 + 版本」都必须出现在清单里（版本变了 = 必须重新生成）。
//   - **不**校验许可证判断是否正确（那是对 LICENSE 文本的机械匹配，需人工复核，清单顶部已声明）；
//   - 不校验 devDependencies（不随产物分发，清单已显式说明不在范围）。
//
// 另有 §1 汇总的**同义归并**断言（2026-10-10 交接快照 §5 未做第 6 项）：
// `TestThirdPartyLicenseSummaryCanonicalizesSynonyms` 要求汇总表每行都是**规范化 SPDX 表达式**
// （`/` ≡ `OR`、同级操作数排序、括号与 `WITH` 例外保留），且不存在两行规范化后相同的同义分裂
// （历史实测把 `MIT OR Apache-2.0` / `Apache-2.0 OR MIT` / `MIT/Apache-2.0` / `Apache-2.0/MIT`
// 拆成了四行）。规则与生成脚本 `scripts/gen-third-party-licenses.sh` 的 `canonical_spdx` 一致，
// 两处必须同改（脚本产出、门禁守住）。

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// thirdPartyDoc 是清单相对仓库根的路径（由生成脚本写入）。
const thirdPartyDoc = "docs/THIRD_PARTY_LICENSES.md"

// 扫描面自检阈值（实测基线：Go 40 / Rust 428 / npm 1）。低于阈值 = 扫描口径塌缩，必须红灯。
const (
	minGoModules  = 35
	minRustCrates = 400
	minNpmRuntime = 1
)

// goModule 是从 `go list -m -json` 抽出的第三方模块。
type goModule struct {
	Path    string
	Version string
	Main    bool
	Std     bool
}

func TestThirdPartyLicensesAreComplete(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	docPath := filepath.Join(root, filepath.FromSlash(thirdPartyDoc))
	raw, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("读取 %s: %v（请先跑 ./scripts/gen-third-party-licenses.sh）", thirdPartyDoc, err)
	}
	doc := string(raw)
	if !strings.Contains(doc, "本文件由脚本自动生成") {
		t.Fatalf("%s 缺少「自动生成」标记——疑似被手工改写；清单必须由生成脚本产出", thirdPartyDoc)
	}

	// ---- Go：模块图里的每个第三方模块（含版本）都必须在清单里 ----
	goMods := goModulesFromSource(t, root)
	if len(goMods) < minGoModules {
		t.Fatalf("只解析到 %d 个 Go 第三方模块（阈值 %d）：解析口径已失效", len(goMods), minGoModules)
	}
	for _, m := range goMods {
		if !strings.Contains(doc, "| `"+m.Path+"` | `"+m.Version+"` |") {
			t.Errorf("Go 模块 %s@%s 未出现在 %s —— 依赖已变更但清单未重新生成（跑 ./scripts/gen-third-party-licenses.sh）",
				m.Path, m.Version, thirdPartyDoc)
		}
	}

	// ---- Rust：Cargo.lock 里的每个 crate（含版本）都必须在清单里 ----
	rust := rustCratesFromLock(t, root)
	if len(rust) < minRustCrates {
		t.Fatalf("只解析到 %d 个 crate（阈值 %d）：Cargo.lock 解析口径已失效", len(rust), minRustCrates)
	}
	for _, c := range rust {
		if !strings.Contains(doc, "| `"+c[0]+"` | `"+c[1]+"` |") {
			t.Errorf("crate %s@%s 未出现在 %s —— Cargo.lock 已变更但清单未重新生成", c[0], c[1], thirdPartyDoc)
		}
	}

	// ---- npm：前端运行时依赖必须都在清单里 ----
	npm := npmRuntimeDeps(t, root)
	if len(npm) < minNpmRuntime {
		t.Fatalf("只解析到 %d 个前端运行时依赖（阈值 %d）：package.json 解析口径已失效", len(npm), minNpmRuntime)
	}
	for _, name := range npm {
		if !strings.Contains(doc, "| `"+name+"` |") {
			t.Errorf("前端运行时依赖 %s 未出现在 %s —— package.json 已变更但清单未重新生成", name, thirdPartyDoc)
		}
	}
}

// goModulesFromSource 跑 `go list -m -json all`（离线、走本地模块缓存）并返回第三方模块。
func goModulesFromSource(t *testing.T, root string) []goModule {
	t.Helper()
	cmd := exec.Command("go", "list", "-m", "-json", "all")
	cmd.Dir = filepath.Join(root, "apps", "server")
	cmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod", "GOPROXY=off")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list -m all 失败（需要本地模块缓存；CI 在 go build 之后运行本门禁）: %v", err)
	}
	dec := json.NewDecoder(strings.NewReader(string(out)))
	var mods []goModule
	for dec.More() {
		var m goModule
		if err := dec.Decode(&m); err != nil {
			t.Fatalf("解析 go list 输出: %v", err)
		}
		if m.Main || m.Std || m.Version == "" {
			continue // 本仓库主模块 / 标准库
		}
		mods = append(mods, m)
	}
	return mods
}

// rustLockSep 是 Cargo.lock 里每个 package 段的起始标记。
// 用 `strings.Split` 而不是正则切块：Go 的 RE2 **不支持前瞻断言**（`(?=…)`），
// 而「消费式」的收尾会把下一段的 `[[` 吃掉、隔块漏读（首版两种写法都踩过）。
const rustLockSep = "[[package]]"

var (
	rustLockNameRe    = regexp.MustCompile(`(?m)^name = "([^"]+)"`)
	rustLockVersionRe = regexp.MustCompile(`(?m)^version = "([^"]+)"`)
	rustLockSourceRe  = regexp.MustCompile(`(?m)^source = "`)
)

// rustCratesFromLock 解析 Cargo.lock 里的**第三方** crate（name, version）。
//
// 刻意跳过没有 `source` 的条目：那是本工作区的 path 依赖（`s3client` 自身），不是第三方组件——
// 首版门禁把它当成第三方并要求出现在清单里，是**门禁自身的口径 bug**（已由本函数修正）。
func rustCratesFromLock(t *testing.T, root string) [][2]string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, "apps", "desktop", "src-tauri", "Cargo.lock"))
	if err != nil {
		t.Fatalf("读取 Cargo.lock: %v", err)
	}
	var out [][2]string
	for _, body := range strings.Split(string(b), rustLockSep)[1:] {
		if !rustLockSourceRe.MatchString(body) {
			continue // 本地 path 依赖（本仓库自己的 crate）
		}
		n := rustLockNameRe.FindStringSubmatch(body)
		v := rustLockVersionRe.FindStringSubmatch(body)
		if n == nil || v == nil {
			t.Fatalf("Cargo.lock 存在无法解析的 %s 段：\n%s", rustLockSep, body)
		}
		out = append(out, [2]string{n[1], v[1]})
	}
	return out
}

// npmRuntimeDeps 返回 `apps/web/package.json` 里 dependencies 的包名。
func npmRuntimeDeps(t *testing.T, root string) []string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, "apps", "web", "package.json"))
	if err != nil {
		t.Fatalf("读取 apps/web/package.json: %v", err)
	}
	var pkg struct {
		Dependencies map[string]string `json:"dependencies"`
	}
	if err := json.Unmarshal(b, &pkg); err != nil {
		t.Fatalf("解析 apps/web/package.json: %v", err)
	}
	names := make([]string, 0, len(pkg.Dependencies))
	for name := range pkg.Dependencies {
		names = append(names, name)
	}
	return names
}

// thirdPartySummaryRowRe 匹配 §1 汇总表行（首列 = 反引号包裹的许可证表达式）。
var thirdPartySummaryRowRe = regexp.MustCompile("^\\|\\s*`([^`]+)`\\s*\\|")

// spdxSplitTop 按**顶层**分隔符切分（括号内的分隔符不切），返回去空白的片段。
// 与 `scripts/gen-third-party-licenses.sh` 的 `_split_top` 同一规则，两处必须同改。
func spdxSplitTop(s, sep string) []string {
	var out []string
	depth, cur := 0, strings.Builder{}
	for i := 0; i < len(s); {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
		}
		if depth == 0 && strings.HasPrefix(s[i:], sep) {
			out = append(out, strings.TrimSpace(cur.String()))
			cur.Reset()
			i += len(sep)
			continue
		}
		cur.WriteByte(s[i])
		i++
	}
	out = append(out, strings.TrimSpace(cur.String()))
	return out
}

// spdxSortedJoin 对同级操作数去重 + 字典序排序后按 sep 连接（OR / AND 同级可交换、可去重）。
func spdxSortedJoin(parts []string, sep string) string {
	seen := map[string]bool{}
	var uniq []string
	for _, p := range parts {
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		uniq = append(uniq, p)
	}
	sort.Strings(uniq)
	return strings.Join(uniq, sep)
}

// canonicalSpdx 把 SPDX 表达式规范化为**同义归并**形态：`/` ≡ `OR`、按 SPDX 优先级
// （WITH > AND > OR）递归规范、同级操作数排序去重、括号与 `WITH` 例外保留。
// `UNKNOWN…` 原样返回（它不是 SPDX 表达式）。
func canonicalSpdx(expr string) string {
	if strings.HasPrefix(expr, "UNKNOWN") {
		return expr
	}
	s := regexp.MustCompile(`\s*/\s*`).ReplaceAllString(strings.TrimSpace(expr), " OR ")
	s = strings.Join(strings.Fields(s), " ")
	return spdxCanonOr(s)
}

func spdxCanonOr(s string) string {
	parts := make([]string, 0, 4)
	for _, p := range spdxSplitTop(s, " OR ") {
		parts = append(parts, spdxCanonAnd(p))
	}
	return spdxSortedJoin(parts, " OR ")
}

func spdxCanonAnd(s string) string {
	parts := make([]string, 0, 4)
	for _, p := range spdxSplitTop(s, " AND ") {
		parts = append(parts, spdxCanonAtom(p))
	}
	return spdxSortedJoin(parts, " AND ")
}

func spdxCanonAtom(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")") {
		return "(" + spdxCanonOr(strings.TrimSpace(s[1:len(s)-1])) + ")"
	}
	return strings.Join(strings.Fields(s), " ")
}

// TestThirdPartyLicenseSummaryCanonicalizesSynonyms 断言 §1 汇总每行都是规范化表达式，
// 且没有两个同义表达式被拆成两行——同义分裂会让「按许可证看依赖分布」失真。
func TestThirdPartyLicenseSummaryCanonicalizesSynonyms(t *testing.T) {
	t.Parallel()

	// 规范化口径的样例护栏（防函数被改坏后门禁静默恒真）。
	for expr, want := range map[string]string{
		"MIT/Apache-2.0":                      "Apache-2.0 OR MIT",
		"MIT OR Apache-2.0":                   "Apache-2.0 OR MIT",
		"Apache-2.0/MIT":                      "Apache-2.0 OR MIT",
		"Apache-2.0 / MIT":                    "Apache-2.0 OR MIT",
		"Apache-2.0 AND MIT":                  "Apache-2.0 AND MIT",
		"(MIT OR Apache-2.0) AND Unicode-3.0": "(Apache-2.0 OR MIT) AND Unicode-3.0",
		"UNKNOWN（模块内无 LICENSE / COPYING 文件）": "UNKNOWN（模块内无 LICENSE / COPYING 文件）",
	} {
		if got := canonicalSpdx(expr); got != want {
			t.Fatalf("canonicalSpdx(%q) = %q，期望 %q——规范化口径被改坏", expr, got, want)
		}
	}

	text := readRepoFile(t, thirdPartyDoc)
	inSummary := false
	rows := 0
	seen := map[string]string{}
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "## 1.") {
			inSummary = true
			continue
		}
		if inSummary && strings.HasPrefix(line, "## 2.") {
			break
		}
		if !inSummary {
			continue
		}
		m := thirdPartySummaryRowRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		label := m[1]
		rows++
		canon := canonicalSpdx(label)
		if canon != label {
			t.Errorf("%s §1 汇总行 `%s` 未规范化（应为 `%s`）——生成脚本的 canonical_spdx 失效或文档被手工改写",
				thirdPartyDoc, label, canon)
		}
		if prev, dup := seen[canon]; dup {
			t.Errorf("%s §1 把同义表达式拆成了两行：`%s` 与 `%s`（规范化后同为 `%s`）",
				thirdPartyDoc, prev, label, canon)
		}
		seen[canon] = label
	}
	if rows < 10 {
		t.Fatalf("扫描面塌缩：%s §1 只解析到 %d 行（基线 ≥10）——表格形态或小节标题被改坏",
			thirdPartyDoc, rows)
	}
	t.Logf("§1 汇总行 %d 行，规范化后唯一 %d 个表达式", rows, len(seen))
}
