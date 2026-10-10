package main

// adr_format_gate_test.go —— `docs/decisions/` 的 ADR **体例**不变量。
//
// 背景（2026-10-10 交接快照 §5 未做第 12 项）：ADR 覆盖面已由 `adr_coverage_gate_test.go`
// （architecture.md §7 每行有决策链接）与 `TestADRIndexListsEveryADRFile`（文件 ⇄ index 双向）
// 钉住，但**体例**没有：ADR-013 写成 `Accepted（已采纳）`（其余 12 篇是裸 `Accepted`）、
// `## Alternatives Considered` 用编号列表 + 缩进 Pros/Cons（其余 12 篇用 `### 方案名` 标题）、
// 正文混用中文弯引号“…”与「」。这类漂移不会让任何既有门禁变红，却让「复制模板起新篇」
// 的约定逐渐失效——本篇把模板里写明的形态变成红灯。
//
// 断言范围（只钉**可机械判定**的形态，不评判决策内容与 Pros/Cons 是否充分——那是人工评审）：
//   - 每篇 `docs/decisions/NNNN-*.md`（除 `0000-template.md`）的前六个 H2 依次必须是
//     `Status` / `Date` / `Context` / `Decision` / `Alternatives Considered` / `Consequences`
//     （与模板一致），其后的 H2 只允许 `## Update（YYYY-MM-DD）`（ADR 不归档、只追加）；
//   - `Status` 正文只允许裸枚举值 `Proposed` / `Accepted` / `Deprecated` /
//     `Superseded by ADR-NNN`，或取代形态 `Accepted（Supersedes ADR-NNN）`（**仅此一种括注**）；
//   - `Alternatives Considered` 至少有 1 个 `### ` 方案标题（防退回编号列表体例）；
//   - 正文不出现中文弯引号 `“` / `”`（仓库统一用「」）。
// 自检纪律：扫到的 ADR 文件数少于 `minADRFiles`（13）→ Fatal，不允许「没扫到就全绿」。
//
// 盲区（刻意不做）：
//   - 不断言 Pros / Cons 的逐块完整性——ADR-007 有两段「两者都实现 / 只是取舍」的分号式说明，
//     强行要求每块 Pros+Cons 会误伤；
//   - 不断言 `index.md`「状态」列的括注文案（那是给人读的，且 ADR-003 / ADR-004 各自补了
//     不同注记），只保证正文**不**混入括注；
//   - 不断言 Date 是否为回溯、是否与 git 历史一致（人工审查范畴）。
//
// 变异验证（2026-10-10 实测）：
//   - 把 ADR-013 的 Status 改回 `Accepted（已采纳）` → TestADRBodyFollowsTemplateShape 点名
//     「非裸枚举值」；
//   - 把 ADR-013 的首个 `### ` 方案标题改回编号列表行 `1. **…**` → 同测试点名
//     「用编号列表列方案」（首版只断言「至少一个 `### `」，四条 `### ` 里改一条仍全绿，已补负向断言）；
//   - 删掉 ADR-013 全部 `### ` 方案标题 → 同测试点名「没有 `### ` 方案标题」；
//   - 在 ADR-013 正文插入一个弯引号 `“` → TestADRBodyHasNoCurlyQuotes 点名行号；
//   - 删除 ADR-013 的 `## Consequences` 小节 → 同测试点名「缺少 `## Consequences`」；
//   全部还原后全绿（13 篇 ADR 扫描面）。
//
// 相关门禁：`adr_coverage_gate_test.go`（architecture.md §7 覆盖 + index 双向）、
// `doc_link_gate_test.go`（`## Update` 锚点）、`doc_index_gate_test.go`（文档导航入口）。

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// adrH2Re 匹配 ADR 里的 H2 标题（捕获标题文本）。
var adrH2Re = regexp.MustCompile(`(?m)^##\s+(.+?)\s*$`)

// adrUpdateH2Re 是允许出现在六个约定小节之后的 H2 形态。
var adrUpdateH2Re = regexp.MustCompile(`^Update（\d{4}-\d{2}-\d{2}）$`)

// adrStatusAllowedRe 是 `Status` 正文允许的形态（裸枚举值 + 唯一允许的括注：取代关系）。
var adrStatusAllowedRe = regexp.MustCompile(`^(Proposed|Accepted|Deprecated|Superseded by ADR-\d{3}|Accepted（Supersedes ADR-\d{3}）)$`)

// adrNumberedAltRe 匹配 `## Alternatives Considered` 里的编号列表方案行（`1. **方案**`）。
var adrNumberedAltRe = regexp.MustCompile(`^\d+\.\s`)

// adrTemplateSections 是模板约定的前六个 H2（顺序敏感）。
var adrTemplateSections = []string{"Status", "Date", "Context", "Decision", "Alternatives Considered", "Consequences"}

// minADRFiles 是扫描面自检下限（当前 13 篇：ADR-001..ADR-013）。
const minADRFiles = 13

// adrFiles 返回 `docs/decisions/` 下的 ADR 文件（相对仓库根的斜杠路径，已排除模板与索引）。
func adrFiles(t *testing.T) []string {
	t.Helper()
	dir := filepath.Join(repoRoot(t), "docs", "decisions")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("读取 %s: %v", dir, err)
	}
	var out []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".md") {
			continue
		}
		if name == "0000-template.md" || name == "index.md" {
			continue
		}
		out = append(out, "docs/decisions/"+name)
	}
	if len(out) < minADRFiles {
		t.Fatalf("只扫到 %d 篇 ADR（下限 %d）——目录被清空 / 改名或本门禁解析口径塌缩，"+
			"不允许「没扫到就全绿」", len(out), minADRFiles)
	}
	return out
}

// TestADRBodyFollowsTemplateShape 断言六节顺序、Status 裸枚举值、Alternatives 用 `###` 方案标题。
func TestADRBodyFollowsTemplateShape(t *testing.T) {
	for _, rel := range adrFiles(t) {
		text := readRepoFile(t, filepath.FromSlash(rel))

		var sections []string
		for _, m := range adrH2Re.FindAllStringSubmatch(text, -1) {
			sections = append(sections, m[1])
		}
		if len(sections) < len(adrTemplateSections) {
			t.Errorf("%s 只有 %d 个 H2 小节（模板要求前六个为 %v）",
				rel, len(sections), adrTemplateSections)
			continue
		}
		for i, want := range adrTemplateSections {
			if sections[i] != want {
				t.Errorf("%s 第 %d 个 H2 = %q，模板要求 %q——H2 用英文且顺序固定（见 0000-template.md）",
					rel, i+1, sections[i], want)
			}
		}
		for _, extra := range sections[len(adrTemplateSections):] {
			if !adrUpdateH2Re.MatchString(extra) {
				t.Errorf("%s 出现模板外的小节 %q——ADR 只允许追加 `## Update（YYYY-MM-DD）`（不归档、不改写历史）",
					rel, extra)
			}
		}

		// Status 正文：`## Status` 与下一个 H2 之间的第一个非空行。
		status := sectionFirstLine(text, "Status")
		if status == "" {
			t.Errorf("%s 的 `## Status` 小节为空", rel)
		} else if !adrStatusAllowedRe.MatchString(status) {
			t.Errorf("%s 的 Status = %q 不是裸枚举值——只允许 %v 或 `Accepted（Supersedes ADR-NNN）`；"+
				"「（已采纳）」这类语义括注只属于 index.md 的状态列", rel, status, []string{
				"Proposed", "Accepted", "Deprecated", "Superseded by ADR-NNN"})
		}

		// Alternatives：至少一个 `### ` 方案标题（防退回编号列表体例），且不得出现编号列表方案行。
		alt := sectionBody(text, "Alternatives Considered")
		heads := 0
		for _, line := range strings.Split(alt, "\n") {
			if strings.HasPrefix(line, "### ") {
				heads++
			}
			if adrNumberedAltRe.MatchString(line) {
				t.Errorf("%s 的 `## Alternatives Considered` 用编号列表列方案（%q）——"+
					"改用 `### <方案名>`（编号随增删漂移，标题才是稳定锚点）", rel, strings.TrimSpace(line))
			}
		}
		if heads == 0 {
			t.Errorf("%s 的 `## Alternatives Considered` 没有 `### ` 方案标题——"+
				"替代方案必须用 `### <方案名>` 起头（编号列表会随增删漂移）", rel)
		}
	}
}

// TestADRBodyHasNoCurlyQuotes 断言 ADR 正文不混用中文弯引号（仓库统一「」）。
func TestADRBodyHasNoCurlyQuotes(t *testing.T) {
	for _, rel := range adrFiles(t) {
		text := readRepoFile(t, filepath.FromSlash(rel))
		for i, line := range strings.Split(text, "\n") {
			if strings.ContainsAny(line, "“”") {
				t.Errorf("%s:%d 出现中文弯引号（“ / ”）——正文统一用「」，避免同一篇里两种引号混排：%s",
					rel, i+1, strings.TrimSpace(line))
			}
		}
	}
}

// sectionBody 返回 `## <title>` 到下一个 H2 之间的正文（不含标题行）；小节不存在返回 ""。
// 先给全文补一个前导换行，使「文件以该 H2 开头」与「H2 在行内」两种情形用同一条查找路径。
func sectionBody(md, title string) string {
	head := "\n## " + title + "\n"
	md = "\n" + md
	i := strings.Index(md, head)
	if i < 0 {
		return ""
	}
	rest := md[i+len(head):]
	if j := strings.Index(rest, "\n## "); j >= 0 {
		rest = rest[:j]
	}
	return rest
}

// sectionFirstLine 返回 `## <title>` 小节的第一个非空行（去掉可能的列表 / 标题符号）。
func sectionFirstLine(md, title string) string {
	for _, line := range strings.Split(sectionBody(md, title), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
}
