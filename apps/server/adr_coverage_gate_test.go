package main

// adr_coverage_gate_test.go —— docs/architecture.md §7「关键取舍」表的 ADR 覆盖门禁。
//
// 背景（2026-09-30 ADR 覆盖补足）：
// `docs/architecture.md` §7「关键取舍（详见 ADR）」表原本只有 4 行（ADR-001..004），而 §2
// 「关键机制」表里还有大量已落地决策（SSE 异步任务、账号存储三驱动、预签名直传、批量与
// 流式并发、ZIP 流式打包、单实例约束、REST 无版本前缀）没有对应 ADR——读者想知道「当初
// 为什么这么定」时无据可查。
//
// 补 ADR 容易，难的是防止取舍表与 `docs/decisions/` **再次分叉**：在 §7 新增一行取舍却
// 忘了补决策链接，不会让任何既有门禁变红——`doc_link_gate_test.go` 只保证「已有的链接
// 不悬空」，不保证「每行都有链接」；`doc_index_gate_test.go` 只保证「每篇文档有导航入口」，
// 不保证「取舍行被登记进 ADR」。本门禁把「每行都有决策链接」变成红灯。
//
// 断言范围（只做这一件事）：
//   - 定位 architecture.md 的 `## 7. 关键取舍` 小节（H2、编号 7、标题含「关键取舍」），
//     解析小节内的 Markdown 表格**数据行**（跳过表头与 `|---|` 分隔行、忽略代码围栏内的
//     管道行），断言每一行至少含一个指向 `docs/decisions/` 的 Markdown 链接
//     （等价相对写法 `decisions/`、`./decisions/`、`../docs/decisions/` 均视为命中）。
//   - 链接目标文件是否存在由 `doc_link_gate_test.go` 负责，本门禁只断言「有链接」。
//
// 已知盲区（刻意不做的，避免后来者误判覆盖面）：
//   - 不断言「文档正文里的 ADR 计数文字」（如「ADR 索引（13 篇）」）——文件 ↔ index.md 的
//     完整性已由 TestADRIndexListsEveryADRFile 双向钉住，计数文字仍需人工随增删同步；
//     不做编号连续性 / 题材分布断言（题材覆盖靠人工审查）。
//   - 不覆盖 §2「关键机制」表：该表部分行是机制描述而非取舍，逐行强制会误伤；只对 §7 行。
//   - 不校验链接目标语义是否与该行匹配（人工审查范畴）。
//   - 表格行按「行首（可含缩进）以 `|` 开头」识别：若 §7 小节内出现第二个表格，其数据行
//     也会纳入断言（当前 §7 只有一个表格，如需再开第二个表请同步本门禁口径）。

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// tradeoffHeadingRe 匹配 §7 小节标题：H2、编号 7、标题含「关键取舍」。
var tradeoffHeadingRe = regexp.MustCompile(`^##\s+7\.[^\n]*关键取舍`)

// tableSepRe 匹配 Markdown 表格分隔行（`|---|`、`|:---:|---|` 等）。
var tableSepRe = regexp.MustCompile(`^\s*\|(?:\s*:?-+:?\s*\|)+\s*$`)

// adrLinkTargetRe 抽取行内链接目标 `](target)`（目标可为 <...> 形态，不取 title）。
// 只需目标部分，故不复制 doc_link_gate_test.go 的 mdLinkRe 全形。
var adrLinkTargetRe = regexp.MustCompile(`\]\(\s*(?:<([^>]*)>|([^)\s]*))`)

// stripFencedCode 把代码围栏（``` 或 ~~~）内容替换为空行、保留总行数（行号对齐），
// 围栏内的管道行因此不参与表格解析。围栏自身所在行同样置空（不可能是表格行）。
func stripFencedCode(md string) string {
	lines := strings.Split(md, "\n")
	fence := ""
	out := make([]string, 0, len(lines))
	for _, ln := range lines {
		t := strings.TrimSpace(ln)
		if fence == "" {
			if strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~") {
				fence = t[:3]
				out = append(out, "")
				continue
			}
			out = append(out, ln)
			continue
		}
		if strings.HasPrefix(t, fence) {
			fence = ""
		}
		out = append(out, "")
	}
	return strings.Join(out, "\n")
}

// headingLevel 返回行的标题级别（H1=1 / H2=2，其余 0）。
func headingLevel(line string) int {
	t := strings.TrimSpace(line)
	switch {
	case strings.HasPrefix(t, "## "):
		return 2
	case strings.HasPrefix(t, "# "):
		return 1
	default:
		return 0
	}
}

// tableCell 返回表格行第 n 列（0 起）的文本（去掉首尾管道与空白）。
func tableCell(line string, n int) string {
	parts := strings.Split(line, "|")
	if n+1 < len(parts) {
		return strings.TrimSpace(parts[n+1])
	}
	return ""
}

// rowHasDecisionLink 判定表格行是否含指向 docs/decisions/ 的链接。
func rowHasDecisionLink(line string) bool {
	for _, m := range adrLinkTargetRe.FindAllStringSubmatch(line, -1) {
		tgt := m[1]
		if tgt == "" {
			tgt = m[2]
		}
		tgt = strings.TrimSpace(tgt)
		if strings.HasPrefix(tgt, "decisions/") ||
			strings.HasPrefix(tgt, "./decisions/") ||
			strings.HasPrefix(tgt, "docs/decisions/") ||
			strings.HasPrefix(tgt, "../docs/decisions/") {
			return true
		}
	}
	return false
}

// adrRow 是 §7 取舍表的一行数据行。
type adrRow struct {
	line    int    // 在 architecture.md 中的 1 起行号
	cell    string // 第一列文本（决策列），用于红灯点名
	hasLink bool   // 是否含指向 docs/decisions/ 的链接
}

// tradeoffRows 从 architecture.md 全文解析 §7「关键取舍」表的数据行。
// 返回 (rows, found)：found=false 表示没找到该小节、或该小节内没有表格数据行。
func tradeoffRows(md string) ([]adrRow, bool) {
	md = stripFencedCode(md)
	lines := strings.Split(md, "\n")
	var out []adrRow
	inSection := false
	sawHeaderSep := false
	for i, ln := range lines {
		if inSection {
			if lvl := headingLevel(ln); lvl == 1 || lvl == 2 {
				break // 下一个 H1/H2：§7 结束
			}
		} else {
			if tradeoffHeadingRe.MatchString(ln) {
				inSection = true
			}
			continue
		}
		if !strings.HasPrefix(strings.TrimSpace(ln), "|") {
			continue
		}
		if tableSepRe.MatchString(ln) {
			sawHeaderSep = true
			continue
		}
		if !sawHeaderSep {
			continue // 分隔行之前是表头
		}
		out = append(out, adrRow{
			line:    i + 1,
			cell:    tableCell(ln, 0),
			hasLink: rowHasDecisionLink(ln),
		})
	}
	if !inSection || len(out) == 0 {
		return nil, false
	}
	return out, true
}

// shortenCell 截断单元格文本用于红灯点名（长文案不淹没输出）。
func shortenCell(cell string, max int) string {
	r := []rune(cell)
	if len(r) <= max {
		return cell
	}
	return string(r[:max]) + "…"
}

// TestArchitectureTradeoffRowsHaveDecisionLinks 断言 §7 取舍表每一行都有决策链接。
func TestArchitectureTradeoffRowsHaveDecisionLinks(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	b, err := os.ReadFile(filepath.Join(root, "docs", "architecture.md"))
	if err != nil {
		t.Fatalf("读取 docs/architecture.md: %v", err)
	}
	rows, found := tradeoffRows(string(b))
	if !found {
		t.Fatal("architecture.md 未找到 §7「关键取舍」小节或该小节没有表格数据行：解析口径失效")
	}
	for _, r := range rows {
		if !r.hasLink {
			t.Errorf("architecture.md:%d 取舍表行「%s」缺少指向 docs/decisions/ 的决策链接——"+
				"新增取舍行必须同 PR 补 ADR 并在本行给出链接",
				r.line, shortenCell(r.cell, 24))
		}
	}
}

// ---- 口径用例（合成源码，不依赖仓库里正好有一个反例） ----

// TestTradeoffRowsOnlyCollectSectionSeven 断言其它小节的表格行不参与收集。
func TestTradeoffRowsOnlyCollectSectionSeven(t *testing.T) {
	t.Parallel()
	md := "# 架构设计\n\n" +
		"## 2. 后端分层\n\n" +
		"| 机制 | 说明 |\n|---|---|\n| 错误映射 | 无链接行 |\n\n" +
		"## 7. 关键取舍（详见 ADR）\n\n" +
		"| 决策 | 理由 |\n|---|---|\n" +
		"| 存储不可用时硬失败 | 见 [ADR-002](decisions/0002-store-fail-closed.md) |\n"
	rows, found := tradeoffRows(md)
	if !found {
		t.Fatal("未解析到 §7 表格：口径失效")
	}
	if len(rows) != 1 {
		t.Fatalf("期望只收集到 §7 的 1 行，实际 %d 行", len(rows))
	}
	if rows[0].cell != "存储不可用时硬失败" || !rows[0].hasLink {
		t.Fatalf("收集结果不符：%+v", rows[0])
	}
}

// TestTradeoffRowsSkipHeaderAndSeparator 断言表头行与分隔行不作为数据行。
func TestTradeoffRowsSkipHeaderAndSeparator(t *testing.T) {
	t.Parallel()
	md := "## 7. 关键取舍（详见 ADR）\n\n" +
		"| 决策 | 理由 |\n|:---|---:|\n" +
		"| A | 无链接 |\n" +
		"| B | [ADR](decisions/0001-desktop-no-ipc.md) |\n"
	rows, found := tradeoffRows(md)
	if !found {
		t.Fatal("未解析到 §7 表格：口径失效")
	}
	if len(rows) != 2 {
		t.Fatalf("期望 2 行数据行（表头/分隔行不算），实际 %d", len(rows))
	}
	if rows[0].cell != "A" || rows[0].hasLink {
		t.Fatalf("行 A 应为「无链接」数据行：%+v", rows[0])
	}
	if rows[1].cell != "B" || !rows[1].hasLink {
		t.Fatalf("行 B 应为「有链接」数据行：%+v", rows[1])
	}
	if rows[0].line != 5 {
		t.Fatalf("行 A 的行号应为 5（1 起：标题/空行/表头/分隔行在前），实际 %d", rows[0].line)
	}
}

// TestTradeoffRowsIgnoreFencedCode 断言代码围栏内的管道行不参与收集（含围栏内伪标题）。
func TestTradeoffRowsIgnoreFencedCode(t *testing.T) {
	t.Parallel()
	md := "## 7. 关键取舍（详见 ADR）\n\n" +
		"```\n" +
		"## 9. 假标题\n" +
		"| 伪 | 表格行 |\n" +
		"```\n\n" +
		"| 决策 | 理由 |\n|---|---|\n" +
		"| C | [ADR](decisions/0004-minimal-frontend-deps.md) |\n"
	rows, found := tradeoffRows(md)
	if !found {
		t.Fatal("未解析到 §7 表格：口径失效")
	}
	if len(rows) != 1 {
		t.Fatalf("围栏内管道行与伪标题应被忽略，期望 1 行，实际 %d 行", len(rows))
	}
	if rows[0].cell != "C" {
		t.Fatalf("收集结果不符：%+v", rows[0])
	}
}

// TestTradeoffRowsDetectDecisionLinkShapes 断言各等价链接写法判「命中」、非 decisions 目标判「未命中」。
func TestTradeoffRowsDetectDecisionLinkShapes(t *testing.T) {
	t.Parallel()
	md := "## 7. 关键取舍（详见 ADR）\n\n" +
		"| 决策 | 理由 |\n|---|---|\n" +
		"| 直写 | [a](decisions/0001-desktop-no-ipc.md) |\n" +
		"| 点斜杠 | [b](./decisions/0002-store-fail-closed.md) |\n" +
		"| 全路径 | [c](docs/decisions/0003-ssrf-private-allow.md) |\n" +
		"| 上溯全路径 | [d](../docs/decisions/0004-minimal-frontend-deps.md) |\n" +
		"| 外部文件 | [e](../README.md) |\n" +
		"| 同目录模板 | [f](0000-template.md) |\n" +
		"| 无链接 | 没有任何链接 |\n"
	rows, found := tradeoffRows(md)
	if !found {
		t.Fatal("未解析到 §7 表格：口径失效")
	}
	if len(rows) != 7 {
		t.Fatalf("期望 7 行，实际 %d", len(rows))
	}
	for i, want := range []bool{true, true, true, true, false, false, false} {
		if rows[i].hasLink != want {
			t.Errorf("第 %d 行（%s）hasLink=%v，期望 %v",
				i+1, rows[i].cell, rows[i].hasLink, want)
		}
	}
}

// TestTradeoffRowsMissingSectionFails 断言没有 §7 小节时返回 found=false（防口径静默失效）。
func TestTradeoffRowsMissingSectionFails(t *testing.T) {
	t.Parallel()
	md := "# 架构设计\n\n## 2. 后端分层\n\n| 机制 | 说明 |\n|---|---|\n| 错误映射 | 无 |\n"
	rows, found := tradeoffRows(md)
	if found {
		t.Fatalf("文档没有 §7 小节时不应解析出数据行，实际 %d 行", len(rows))
	}
	// 有 §7 标题但无表格时同样 found=false。
	md2 := "## 7. 关键取舍（详见 ADR）\n\n这里没有表格。\n"
	if rows2, found2 := tradeoffRows(md2); found2 {
		t.Fatalf("§7 无表格时应 found=false，实际解析出 %d 行", len(rows2))
	}
}

// TestADRIndexListsEveryADRFile 断言 docs/decisions/index.md 的索引与 docs/decisions/ 下的
// ADR 文件**双向一致**（`0000-template.md` 与 `index.md` 除外）：新增 ADR 忘登记、或索引残留
// 已删除的文件，都会让「ADR 索引（N 篇）」这类导航计数漂移复发（2026-10-10 实测
// docs/README.md 与 llms.txt 在 ADR-013 新增后仍写「12 篇」，计数文字此前无任何机械校验）。
// 变异验证：从 index.md 删一行 → 红灯点名缺失文件；登记不存在的目标 → 反向红灯。
func TestADRIndexListsEveryADRFile(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(repoRoot(t), "docs", "decisions")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("读取 decisions 目录: %v", err)
	}
	onDisk := map[string]bool{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		if e.Name() == "index.md" || e.Name() == "0000-template.md" {
			continue
		}
		onDisk[e.Name()] = true
	}
	if len(onDisk) < 5 {
		t.Fatalf("decisions/ 只解析到 %d 个 ADR 文件，疑似扫描口径失效", len(onDisk))
	}

	idx := readRepoFile(t, "docs/decisions/index.md")
	inIndex := map[string]bool{}
	for _, m := range adrLinkTargetRe.FindAllStringSubmatch(idx, -1) {
		target := m[1]
		if target == "" {
			target = m[2]
		}
		target = strings.TrimPrefix(target, "./")
		if strings.Contains(target, "/") || !strings.HasSuffix(target, ".md") {
			continue // 只关心同目录（decisions/ 内）的文件链接
		}
		inIndex[target] = true
	}
	delete(inIndex, "0000-template.md")
	delete(inIndex, "index.md")

	for f := range onDisk {
		if !inIndex[f] {
			t.Errorf("docs/decisions/%s 存在，但 index.md 索引表未登记——导航与真实文件分叉", f)
		}
	}
	for f := range inIndex {
		if !strings.HasPrefix(f, "0") {
			continue // 非 ADR 编号形文件不属本断言范围
		}
		if !onDisk[f] {
			t.Errorf("index.md 登记了 decisions/ 下不存在的 ADR 文件 %s", f)
		}
	}
}
