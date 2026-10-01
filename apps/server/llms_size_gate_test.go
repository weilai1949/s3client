package main

// llms_size_gate_test.go —— `llms.txt` 对**超大文档**必须给出体量警示的门禁。
//
// 背景（2026-09-30，ROADMAP §三 3.2 #19②）：`CHANGELOG.md`（约 323 KB）与 `docs/FEATURES.md`
// （约 247 KB）被平铺直链进 `llms.txt`——「链接可达性」门禁（`doc_link_gate_test.go`）只管链到没有，
// 不管链进来的是一篇会把上下文撑爆的巨文件。LLM / 代理按索引整读会直接截断，故「超大必须预警」
// 与「链接必须可达」是两件事，各配一道门禁。
//
// 断言范围：
//   - 扫描 `llms.txt` 中形如 `[标签](相对路径)` 的**文件**链接（跳过外链、锚点与目录）；
//   - 目标文件字节数 > llmsOversizeBytes（200 KB）→ 该行必须含 llmsOversizeMarker（`⚠️ 超大`）；
//   - 扫描面自检：解析出的文件链接低于阈值 → Fatal（防解析口径塌缩后「全绿但失明」）。
//
// 不断言：具体字节数（随编辑漂移，断言了必红，故只断言「超阈值需预警」这一定性事实）、
// 链接文案与顺序、目标低于阈值的行。
//
// 变异验证（复核步骤，2026-09-30 实跑）：删掉 `llms.txt` 中 `CHANGELOG.md` 行的 `⚠️ 超大` →
// 本门禁红灯点名「目标 319 KB 超过 200 KB 阈值却无体量警示」→ 还原后绿灯。
// 复核命令：`cd apps/server && go test . -run TestLLMSTextWarns -count=1`。
//
// 相关：`doc_link_gate_test.go`（链接可达）、`ai_governance_gate_test.go`（根 AGENTS.md 注入体积预算）——
// 三者分别保证「链得对」「链得到」「进得去」。

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const (
	// llmsOversizeBytes 是「超大」阈值：超过它的单文件链接必须在 llms.txt 就地预警。
	// 200 KB 是经验值（LLM 单文件检索的舒适量级上限），只用于**定性**判断。
	llmsOversizeBytes = 200 << 10
	// llmsOversizeMarker 是预警标记，须出现在超大目标所在行。
	llmsOversizeMarker = "⚠️ 超大"
	// minLLMSFileLinks 是扫描面自检阈值（实测 2026-09-30：llms.txt 43 个相对文件链接）。
	minLLMSFileLinks = 30
)

// llmsLinkRe 匹配 markdown 链接的目标部分。
var llmsLinkRe = regexp.MustCompile(`\[[^\]]*\]\(([^)]+)\)`)

// TestLLMSTextWarnsAboutOversizedDocs 断言：`llms.txt` 里指向超大文件的链接行必须带体量警示。
func TestLLMSTextWarnsAboutOversizedDocs(t *testing.T) {
	text := readRepoFile(t, "llms.txt")
	root := repoRoot(t)

	seen := 0
	oversize := 0
	for _, m := range llmsLinkRe.FindAllStringSubmatch(text, -1) {
		target := m[1]
		if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") ||
			strings.HasPrefix(target, "#") {
			continue // 外链与纯锚点不涉及仓库文件体量
		}
		if i := strings.IndexByte(target, '#'); i >= 0 {
			target = target[:i] // 去掉页内锚点
		}
		info, err := os.Stat(filepath.Join(root, target))
		if err != nil || info.IsDir() {
			continue // 非文件目标（不可达由 doc_link_gate 负责）
		}
		seen++
		if info.Size() <= llmsOversizeBytes {
			continue
		}
		oversize++
		line := lineContaining(text, m[0])
		if !strings.Contains(line, llmsOversizeMarker) {
			t.Errorf("llms.txt 链接 %q：目标 %d KB 超过 %d KB 阈值，所在行缺体量警示标记 %q"+
				"（LLM 按索引整读会被截断，须就地预警）",
				target, info.Size()>>10, llmsOversizeBytes>>10, llmsOversizeMarker)
		}
	}

	if seen < minLLMSFileLinks {
		t.Fatalf("扫描面塌缩：llms.txt 解析出 %d 个文件链接，低于自检阈值 %d", seen, minLLMSFileLinks)
	}
	t.Logf("llms.txt 文件链接 %d 个，其中超大（> %d KB）%d 个", seen, llmsOversizeBytes>>10, oversize)
}

// lineContaining 返回包含子串的那一整行（按行扫描，取第一处）。
func lineContaining(text, sub string) string {
	for _, line := range strings.Split(text, "\n") {
		if strings.Contains(line, sub) {
			return line
		}
	}
	return ""
}
