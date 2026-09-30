package main

// changelog_tag_gate_test.go —— 「CHANGELOG 版本段 ↔ git tag」一致性的源码门禁。
//
// 背景（KNOWN_ISSUES #69）：CHANGELOG 曾与 git tag 断裂——
//   ① 3 个时间戳 tag（v1.0.0-20260902164154 / -20260902170212 / -20260902170253）
//     没有任何对应版本段；
//   ② 3 个 2026-09-01 快照段（[v1.0.0-20260901182023] / [20260901.2] / [20260901]）与
//     [0.1.0] / [0.2.0] 段没有对应 tag；
//   ③ `## [1.0.0]` 段日期与 tag 日期不符、顺序非倒序。
// 版本段是「发布记录」，tag 是「发布事实」——二者不一致时，照 CHANGELOG 找 commit 与照 tag
// 找记录会互相矛盾。人工核对只能发现一次，本门禁把它变成红灯。
//
// 断言范围（只钉结构不变量，不评判段内容）：
//   - `## [Unreleased]` 必须是 CHANGELOG 的第一个版本段（倒序排列的锚点）；
//   - 每个 `v*` git tag 都有对应版本段（`## [<tag>]` 或去掉前导 `v` 的 `## [<tag[1:]>]`），
//     或在 CHANGELOG 顶部「tag ↔ 版本段对应关系」表中登记为快照 tag（段列 = 无）；
//   - 反向：每个 `## [version]` 段（Unreleased 除外）都有对应 tag（补前导 `v` 也算），
//     或在该表中登记为快照段（tag 列 = 无）；
//   - `scripts/release-version.sh` 收尾必须对 `## [$DISPLAY]` 做硬检查（缺段即 exit 1）；
//   - 扫描面自检阈值：tag / 段 / 映射行数量低于基线即红灯（防解析口径塌缩后「全绿但失明」）。
//
// 已知盲区（刻意不做）：
//   - tag 从 `.git/packed-refs` 与 `.git/refs/tags` 读文件获取（不依赖 git 命令）；
//     CI 不带 .git 时按阈值红灯，不静默放行；
//   - 不校验段**日期**与 tag 提交日期的吻合（属人工审查；#69 的日期修正是一次性事实核对）；
//   - 不校验映射表「说明」列的内容是否属实（自然语言）。

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// changelogSectionRe 匹配 CHANGELOG 的版本段标题行，组 1 = 段名（不含方括号）。
var changelogSectionRe = regexp.MustCompile(`(?m)^## \[([^\]]+)\]`)

// changelogMappingRowRe 匹配「tag ↔ 版本段对应关系」表的正文行：
// 首列为 tag（`无` = 该行只登记版本段）、第二列为版本段（`无` = 该行只登记 tag）。
// 第三、四列（性质 / 说明）不参与解析；表头行与 `---` 分隔行由调用方过滤。
var changelogMappingRowRe = regexp.MustCompile("^\\|\\s*`?([^`|]*)`?\\s*\\|\\s*`?([^`|]*)`?\\s*\\|")

// 扫描面自检基线（实测值，2026-09-30）：
//
//	20 个 v* tag · 23 个版本段（含 [Unreleased]）· 8 行快照登记。
//
// 低于阈值说明解析 / 读取口径塌缩，必须红灯而不是安静通过。
const (
	minChangelogTags     = 15
	minChangelogSections = 15
	minChangelogMapRows  = 6
)

// changelogSections 按出现顺序返回 CHANGELOG 的全部版本段名。
func changelogSections(text string) []string {
	var out []string
	for _, m := range changelogSectionRe.FindAllStringSubmatch(text, -1) {
		out = append(out, m[1])
	}
	return out
}

// gitTagsFromRefs 从 .git/packed-refs 与 .git/refs/tags 读文件收集全部 `v*` tag
// （刻意不调用 git 命令，避免 CI 环境的 git 可用性差异）。
func gitTagsFromRefs(t *testing.T) []string {
	t.Helper()
	root := repoRoot(t)
	tags := map[string]bool{}

	// 松散 refs：.git/refs/tags/<name> 每文件一个 tag。
	looseDir := filepath.Join(root, ".git", "refs", "tags")
	_ = filepath.WalkDir(looseDir, func(p string, d os.DirEntry, err error) error {
		if err == nil && d != nil && !d.IsDir() {
			if rel, relErr := filepath.Rel(looseDir, p); relErr == nil {
				tags[filepath.ToSlash(rel)] = true
			}
		}
		return nil
	})

	// packed refs：`<sha> refs/tags/<name>` 行（剥除 peeled `^<sha>` 行）。
	if b, err := os.ReadFile(filepath.Join(root, ".git", "packed-refs")); err == nil {
		for _, line := range strings.Split(string(b), "\n") {
			if i := strings.Index(line, " refs/tags/"); i >= 0 {
				tags[strings.TrimSpace(line[i+len(" refs/tags/"):])] = true
			}
		}
	}

	var out []string
	for name := range tags {
		if strings.HasPrefix(name, "v") {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

// changelogMappingSectionRe 截取「tag ↔ 版本段对应关系」小节正文（到下一个版本段标题或文末为止）。
// 解析范围刻意限制在该小节内：CHANGELOG 其他地方也可能出现 markdown 表格（如版本段正文里的
// 改名对照表），全文件扫描会把无关表格误当映射行。RE2 不支持前瞻，故用非贪婪 + `\z` 收尾。
var changelogMappingSectionRe = regexp.MustCompile(`(?s)## tag ↔ 版本段对应关系.*?(?:\n## \[|\z)`)

// parseChangelogTagMapping 解析映射表小节，返回两类登记：
//   - sectionOnly：有版本段、无 tag（快照段）的段名集合（已剥方括号）；
//   - tagOnly：有 tag、无版本段（快照 tag）的 tag 集合。
func parseChangelogTagMapping(text string) (sectionOnly, tagOnly map[string]bool) {
	sectionOnly, tagOnly = map[string]bool{}, map[string]bool{}
	block := changelogMappingSectionRe.FindString(text)
	for _, line := range strings.Split(block, "\n") {
		m := changelogMappingRowRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		tagCell := strings.TrimSpace(m[1])
		secCell := strings.TrimSpace(m[2])
		if tagCell == "" || strings.Contains(tagCell, "---") || tagCell == "tag" {
			continue // 表头 / 分隔行 / 空行
		}
		switch {
		case tagCell == "无":
			// 表格里登记为 `[段名]`（带方括号，人读习惯），门禁从标题解析出的是裸段名——
			// 统一剥掉方括号再登记，避免「带括号 vs 不带括号」的假失配。
			sectionOnly[bareSection(secCell)] = true
		case secCell == "无":
			tagOnly[tagCell] = true
		}
	}
	return sectionOnly, tagOnly
}

// bareSection 剥掉版本段两侧的方括号（`[1.0.0]` → `1.0.0`）；不匹配 `[...]` 形态时原样返回。
func bareSection(s string) string {
	if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
		return s[1 : len(s)-1]
	}
	return s
}

// sectionMatchesTag 判断 tag 是否命中段集合：`## [<tag>]` 或去掉前导 v 的 `## [<tag[1:]>]`。
func sectionMatchesTag(sections []string, tag string) bool {
	strip := strings.TrimPrefix(tag, "v")
	for _, s := range sections {
		if s == tag || s == strip {
			return true
		}
	}
	return false
}

// tagMatchesSection 判断版本段是否命中 tag 集合：`<section>` 或补前导 v 的 `v<section>`。
func tagMatchesSection(tags []string, section string) bool {
	withV := section
	if !strings.HasPrefix(section, "v") {
		withV = "v" + section
	}
	for _, tag := range tags {
		if tag == section || tag == withV {
			return true
		}
	}
	return false
}

// TestChangelogTagsAndSectionsReconcile 断言 tag ↔ 版本段双向一致（KNOWN_ISSUES #69 的机械保证）。
func TestChangelogTagsAndSectionsReconcile(t *testing.T) {
	text := readRepoFile(t, "CHANGELOG.md")

	sections := changelogSections(text)
	if len(sections) < minChangelogSections {
		t.Fatalf("CHANGELOG 只解析出 %d 个版本段（阈值 %d）：解析口径已失效", len(sections), minChangelogSections)
	}
	if sections[0] != "Unreleased" {
		t.Errorf("CHANGELOG 第一个版本段是 %q，必须是 [Unreleased]（倒序排列的锚点，KNOWN_ISSUES #69）", sections[0])
	}

	sectionOnly, tagOnly := parseChangelogTagMapping(text)
	if len(sectionOnly)+len(tagOnly) < minChangelogMapRows {
		t.Fatalf("「tag ↔ 版本段对应关系」表只解析出 %d 行（阈值 %d）：表格式或解析口径已失效", len(sectionOnly)+len(tagOnly), minChangelogMapRows)
	}

	tags := gitTagsFromRefs(t)
	if len(tags) < minChangelogTags {
		t.Fatalf("只从 .git 读出 %d 个 v* tag（阈值 %d）：读取口径已失效（CI 必须带 .git，KNOWN_ISSUES #69）", len(tags), minChangelogTags)
	}

	// 正向：每个 tag 有对应段，或在映射表登记为快照 tag。
	for _, tag := range tags {
		if sectionMatchesTag(sections, tag) || tagOnly[tag] {
			continue
		}
		t.Errorf("tag %s 在 CHANGELOG 无对应版本段，也未在「tag ↔ 版本段对应关系」表登记为快照 tag——"+
			"请补段或登记（KNOWN_ISSUES #69）", tag)
	}

	// 反向：每个段（Unreleased 除外）有对应 tag，或在映射表登记为快照段。
	for _, s := range sections[1:] {
		if tagMatchesSection(tags, s) || sectionOnly[s] {
			continue
		}
		t.Errorf("版本段 [%s] 无对应 git tag，也未在「tag ↔ 版本段对应关系」表登记为快照段——"+
			"请补 tag 或登记（KNOWN_ISSUES #69）", s)
	}
}

// TestReleaseScriptHardChecksChangelogSection 断言发版脚本收尾对 `## [$DISPLAY]` 做硬检查：
// 缺段即 exit 1，而不是只在最后 echo 一句提醒（#69 建议处置 (c)）。
func TestReleaseScriptHardChecksChangelogSection(t *testing.T) {
	script := readRepoFile(t, filepath.Join("scripts", "release-version.sh"))
	if !strings.Contains(script, `grep -q "^## \[$DISPLAY\]" CHANGELOG.md`) {
		t.Error("scripts/release-version.sh 收尾缺少对 `## [$DISPLAY]` 版本段的硬检查（grep -q + 缺段 exit 1）：" +
			"否则发版会打出无 CHANGELOG 段的 tag（KNOWN_ISSUES #69）")
	}
}

// TestChangelogMappingParsingIgnoresHeaderAndComments 用合成表格钉住解析口径：
// 表头 / `---` 分隔行 / 非表格行（块引用、裸文本）不得被当成登记行。
func TestChangelogMappingParsingIgnoresHeaderAndComments(t *testing.T) {
	table := "## tag ↔ 版本段对应关系（唯一台账）\n" +
		"| tag | 版本段 | 性质 | 说明 |\n" +
		"|---|---|---|---|\n" +
		"| `无` | `[20260901]` | 快照段 | 说明文字 |\n" +
		"| `v9.9.9-20260101000000` | `无` | 快照 tag | 说明文字 |\n" +
		"> | 块引用里的管道 | 不该被解析 | 也不该 |\n" +
		"## [Unreleased]\n"
	sectionOnly, tagOnly := parseChangelogTagMapping(table)
	if !sectionOnly["20260901"] {
		t.Errorf("sectionOnly 缺 20260901：%v", sectionOnly)
	}
	if !tagOnly["v9.9.9-20260101000000"] {
		t.Errorf("tagOnly 缺 v9.9.9-20260101000000：%v", tagOnly)
	}
	if len(sectionOnly) != 1 || len(tagOnly) != 1 {
		t.Errorf("解析出多余登记行：sectionOnly=%v tagOnly=%v（表头 / 分隔行 / 块引用应被忽略）", sectionOnly, tagOnly)
	}
}
