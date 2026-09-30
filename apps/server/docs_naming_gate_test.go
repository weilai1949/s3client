package main

// docs_naming_gate_test.go —— 「docs/ 命名约定两处分叉」门禁。
//
// 背景（2026-09-30）：`AGENTS.md` 与 `docs/DEVELOPMENT.md` §4 的命名约定要求**两处同改防分叉**
// （“新增或变更任何大写文件名，必须在同一个 PR 里同时改本处与 AGENTS.md”），但两处清单
// **已经分叉过**：`AGENT_EVALS.md` 2026-09-30 落地时只进了 DEVELOPMENT 的清单，AGENTS 的清单漏登；
// AGENTS 的小写清单也漏了 `docs/en/` 子目录；`llms.txt`「目录」段的命名摘要更落后一代名单
// （缺 5 个元文档与 5 个内容文档、`en/` 子目录）。人类纪律拦不住这类漏登记，且没有门禁看得见。
//
// 断言范围（只钉「登记覆盖」，不评判命名规则内容是否合理）：`docs/` 下每个顶层 `*.md` 与每个
// **含 `.md` 的子目录**，其名字必须同时出现在三处命名口径里——
//   - `AGENTS.md` 的命名约定段（“docs/ 下按性质二分命名” ～ “任何两个路径不得仅大小写不同”）；
//   - `docs/DEVELOPMENT.md` §4 的命名约定段（“命名”条 ～ “目录”条）；
//   - `llms.txt` 的「目录」段。
//
// 解析口径：截取文本段后**只认反引号里的登记名**（如 `AGENT_EVALS.md`、`docs/en/`）——散文里
// 提到的名字不算登记（防「提到过就算数」的漏报）；路径前缀 `docs/` 与目录尾 `/` 归一化后比较。
//
// 自检纪律（同 doc_number_gate 等）：docs 顶层 `.md` / 含 `.md` 子目录低于阈值 → Fatal，
// 防止解析口径塌缩后「全绿但失明」。
//
// 变异验证（复核步骤）：摘掉 `AGENTS.md` 命名清单里的 `AGENT_EVALS.md` → 本门禁红灯点名
// 「AGENTS.md 命名约定段缺 AGENT_EVALS.md」→ 还原后绿灯。
// ⚠️ 本轮落笔会话无 shell，门禁**未实跑**：复核命令 `cd apps/server && go test . -count=1`。
//
// 相关：`doc_index_gate_test.go`（导航覆盖）、`doc_link_gate_test.go`（链接与锚点）——
// 三者分别保证「有入口」「链接不死」「命名清单不分叉」。

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// namingSurface 是一处「命名口径」文本段的截取定位（起 / 止标记均为各文件原文，措辞变更需同步本门禁）。
type namingSurface struct {
	name       string
	rel        string
	start, end string
}

// namingSurfaces 是三处必须同步的命名口径。
var namingSurfaces = []namingSurface{
	{"AGENTS.md 命名约定段", "AGENTS.md",
		"- **`docs/` 下按性质二分命名**", "- **任何两个路径不得仅大小写不同**"},
	{"docs/DEVELOPMENT.md §4 命名约定段", filepath.Join("docs", "DEVELOPMENT.md"),
		"- **命名**：`docs/` 下按**文档性质**二分", "- **目录**：一律小写"},
	{"llms.txt 目录段", "llms.txt", "## 目录", "## 禁区"},
}

// backtickTokenRe 匹配反引号片段——登记名的唯一合法形态。
var backtickTokenRe = regexp.MustCompile("`[^`\n]+`")

// 命名登记扫描面自检阈值（实测 2026-09-30：docs 顶层 .md 23 个、含 .md 子目录 3 个）。
const (
	minDocsTopLevelMD = 20
	minDocsSubdirs    = 3
)

// namingSurfaceSlice 截取某处命名口径的文本段；标记缺失即 Fatal（措辞变更需同步本门禁）。
func namingSurfaceSlice(t *testing.T, s namingSurface, text string) string {
	t.Helper()
	i := strings.Index(text, s.start)
	if i < 0 {
		t.Fatalf("%s（%s）未找到起始标记 %q：措辞变更需同步本门禁", s.name, s.rel, s.start)
	}
	rest := text[i+len(s.start):]
	j := strings.Index(rest, s.end)
	if j < 0 {
		t.Fatalf("%s（%s）未找到终止标记 %q：措辞变更需同步本门禁", s.name, s.rel, s.end)
	}
	return rest[:j]
}

// surfaceMentions 判断文本段是否以反引号形态登记了 name（`docs/en/` 的尾斜杠形态归一化后比较）。
func surfaceMentions(slice, name string) bool {
	for _, tok := range backtickTokenRe.FindAllString(slice, -1) {
		got := strings.Trim(tok, "`")
		got = strings.TrimPrefix(got, "docs/")
		got = strings.TrimSuffix(got, "/")
		if got == name {
			return true
		}
	}
	return false
}

// assertNamesRegisteredInAllSurfaces 断言 names 里的每个名字都登记进三处命名口径。
func assertNamesRegisteredInAllSurfaces(t *testing.T, names []string) {
	t.Helper()
	root := repoRoot(t)
	for _, s := range namingSurfaces {
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(s.rel)))
		if err != nil {
			t.Fatalf("读取 %s: %v（命名口径文件缺失即分叉无人可见）", s.rel, err)
		}
		slice := namingSurfaceSlice(t, s, string(b))
		for _, n := range names {
			if !surfaceMentions(slice, n) {
				t.Errorf("%s 缺 %s：docs/ 命名清单要求 AGENTS.md / DEVELOPMENT.md §4 / llms.txt 三处同改"+
					"（防分叉），漏登记即分叉", s.name, n)
			}
		}
	}
}

// TestDocsNamingConventionRegistersEveryDocsFile 断言 docs/ 顶层每个 .md 都在三处命名口径里登记。
func TestDocsNamingConventionRegistersEveryDocsFile(t *testing.T) {
	t.Parallel()
	entries, err := os.ReadDir(filepath.Join(repoRoot(t), "docs"))
	if err != nil {
		t.Fatalf("读取 docs/: %v", err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		names = append(names, e.Name())
	}
	if len(names) < minDocsTopLevelMD {
		t.Fatalf("docs/ 顶层只识别到 %d 个 .md（阈值 %d）：扫描口径失效", len(names), minDocsTopLevelMD)
	}
	assertNamesRegisteredInAllSurfaces(t, names)
}

// TestDocsNamingConventionRegistersEveryDocsSubdir 断言 docs/ 下每个「含 .md 的子目录」都被三处
// 命名口径登记。`api/`（只有机器可读 JSON）与 `images/`（只有截图）不属命名约定管辖，故按「含 .md」过滤。
func TestDocsNamingConventionRegistersEveryDocsSubdir(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	entries, err := os.ReadDir(filepath.Join(root, "docs"))
	if err != nil {
		t.Fatalf("读取 docs/: %v", err)
	}
	var dirs []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		sub, err := os.ReadDir(filepath.Join(root, "docs", e.Name()))
		if err != nil {
			t.Fatalf("读取 docs/%s/: %v", e.Name(), err)
		}
		for _, f := range sub {
			if !f.IsDir() && strings.HasSuffix(f.Name(), ".md") {
				dirs = append(dirs, e.Name())
				break
			}
		}
	}
	if len(dirs) < minDocsSubdirs {
		t.Fatalf("docs/ 下只识别到 %d 个含 .md 的子目录（阈值 %d）：扫描口径失效", len(dirs), minDocsSubdirs)
	}
	assertNamesRegisteredInAllSurfaces(t, dirs)
}
