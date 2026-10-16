package main

// web_xss_gate_test.go —— P-6（2026-10-15 渗透测试快照）的前端 XSS **机械门禁**。
//
// 为什么需要它：`v-html` 的禁令此前只写在 AGENTS.md / apps/web/AGENTS.md 的硬约束与
// 代码评审口径里，**没有任何门禁**。2026-10-15 的对抗性复核确认：ESLint 的
// `pluginVue.configs['flat/essential']` 不含 `vue/no-v-html`（实测该 config 的规则集里
// 0 处命中），因而「禁 v-html」当时只是文字约定——新增一处 `v-html` 不会让任何流水线变红。
// 本文件把它变成两道可执行的断言：
//
//  1. `apps/web/eslint.config.js` 必须**显式**启用 `'vue/no-v-html': 'error'`；
//  2. `apps/web/src/**` 的**生产源码**在剥离注释后，六类 HTML 注入汇聚点必须为 0。
//
// 口径（与 apps/server/internal/handler/review_20260924_test.go 的源码级扫描同一路数）：
//   - 只扫 `apps/web/src/**` 的 `*.vue` / `*.ts`；`*.test.ts` / `*.spec.ts` 是测试代码，
//     不算生产面（与覆盖率 instrumentation 排除 `*.test.ts` 同一口径）——其中
//     `document.body.innerHTML = ''` 是测试里清理 DOM 的常规写法，不构成 XSS 面；
//   - **剥离注释**：`//`、`/* */`、`<!-- -->` 里的词不算违规——注释里解释「为什么不用
//     v-html」恰恰应该保留（现状 `components/HighlightText.vue` 的注释就提到 `v-html`）；
//   - 只做**文本级**断言，不解析 AST：AST 级判定由 ESLint 的 `vue/no-v-html` 承担，
//     本门禁额外覆盖「规则被删掉」与「ESLint 覆盖不到的文件」两种情况，两者互补。
//
// 变异验证（2026-10-16 实测，两次都变红后恢复）：
//   - 临时删掉 `eslint.config.js` 的 `'vue/no-v-html': 'error'` 一行 →
//     `TestWebESLintEnablesNoVHtml` 红；
//   - 临时新建含 `v-html` 的 `apps/web/src/__mutant__.vue` →
//     `TestWebProductionSourceHasNoHTMLSinks` 红。

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// webXSSSinkRe 匹配六类 HTML 注入汇聚点（P-6 待核对清单，逐字对应）：
// Vue 模板的 `v-html`、DOM 的 `innerHTML` / `outerHTML` / `insertAdjacentHTML` /
// `srcdoc`、以及 `document.write`（含 `writeln`）。
//
// 说明：`eval` / `new Function` 同样被 2026-10-15 快照核为 0，但它们是**代码注入**而非
// HTML 注入，且不在本门禁的清单口径内，故不在此列（列进来会让「0 命中」的语义变模糊）。
var webXSSSinkRe = regexp.MustCompile(
	`v-html|innerHTML|outerHTML|insertAdjacentHTML|srcdoc|document\s*\.\s*write(?:ln)?\b`)

// webNoVHtmlRuleRe 匹配 eslint 配置里**显式**启用 `vue/no-v-html` 的行（单双引号都收）。
var webNoVHtmlRuleRe = regexp.MustCompile(`['"]vue/no-v-html['"]\s*:\s*['"]error['"]`)

// webXSSHit 是一次命中：汇聚点的字面量 + 行号（1 起，剥离注释时保留换行故与原文件一致）。
type webXSSHit struct {
	kind string
	line int
}

// TestWebESLintEnablesNoVHtml 断言 `apps/web/eslint.config.js` 显式把 `vue/no-v-html`
// 设为 error。先剥离注释再匹配，避免「注释里写着这条规则」骗过门禁（规则实际被注释掉时
// `pnpm lint` 不会拦任何 `v-html`）。
func TestWebESLintEnablesNoVHtml(t *testing.T) {
	cfg := stripSourceComments(readRepoFile(t, filepath.Join("apps", "web", "eslint.config.js")))
	if !webNoVHtmlRuleRe.MatchString(cfg) {
		t.Error("apps/web/eslint.config.js 未显式启用 `'vue/no-v-html': 'error'`：" +
			"`pluginVue.configs['flat/essential']` 不含该规则（实测），不写这一行则新增 v-html " +
			"不会让 `pnpm lint`（--max-warnings 0）失败（P-6）")
	}
}

// TestWebProductionSourceHasNoHTMLSinks 断言生产源码在剥离注释后不含六类 HTML 汇聚点。
func TestWebProductionSourceHasNoHTMLSinks(t *testing.T) {
	root := repoRoot(t)
	files := webXSSSrcFiles(t)
	for _, path := range files {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("读取 %s: %v", path, err)
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			rel = path
		}
		for _, hit := range scanWebXSSSinks(string(raw)) {
			t.Errorf("%s:%d 出现 HTML 注入汇聚点 %q：生产代码必须走文本插值（{{ }}）/"+
				"文本绑定，不得把外部数据当 HTML 渲染（P-6，硬约束「禁 v-html」）",
				rel, hit.line, hit.kind)
		}
	}
}

// TestWebXSSSinkScannerIsSensitive 判别力自检：正则与注释剥离一旦写坏，生产扫描会以
// 「0 命中」永远全绿——这正是「门禁结构性失明」的形态。故用合成样本把判别力钉住：
// 注释里的汇聚点**不**算命中，代码里的六类**必须**全部命中。
func TestWebXSSSinkScannerIsSensitive(t *testing.T) {
	const sample = `<template>
  <div v-html="untrusted" />
  <span :title="a.innerHTML" />
  <iframe srcdoc="x" />
</template>
<script setup lang="ts">
// 注释里的 v-html 与 innerHTML 不算（与 HighlightText.vue 的现状一致）
/**
 * document.write( 也在块注释里
 */
const el = document.createElement('div')
el.innerHTML = payload
el.outerHTML = 'x'
el.insertAdjacentHTML('beforeend', payload)
document.write(payload)
</script>
`
	stripped := stripSourceComments(sample)
	if strings.Contains(stripped, "注释里的") || strings.Contains(stripped, "也在块注释里") {
		t.Fatal("stripSourceComments 未剥离注释：注释里的汇聚点会被误判为违规")
	}
	hits := scanWebXSSSinks(sample)
	// 命中应为 7 处：v-html / a.innerHTML / srcdoc / el.innerHTML / outerHTML /
	// insertAdjacentHTML / document.write（模板里的 `<!-- -->` 与脚本里的两类注释都不含命中）。
	if len(hits) != 7 {
		t.Fatalf("合成样本命中 %d 处，want 7：正则或注释剥离口径塌缩", len(hits))
	}
	for _, h := range hits {
		if h.line <= 1 || h.line > strings.Count(sample, "\n") {
			t.Fatalf("行号 %d 越界：剥离注释时未保留换行对齐", h.line)
		}
	}
}

// webXSSSrcFiles 返回 `apps/web/src` 下的生产源码（`*.vue` / `*.ts`，排除测试与 d.ts）。
func webXSSSrcFiles(t *testing.T) []string {
	t.Helper()
	root := filepath.Join(repoRoot(t), "apps", "web", "src")
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		if strings.HasSuffix(name, ".test.ts") || strings.HasSuffix(name, ".spec.ts") {
			return nil
		}
		if strings.HasSuffix(name, ".vue") || strings.HasSuffix(name, ".ts") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("遍历 %s: %v", root, err)
	}
	// 解析口径塌缩守卫：源文件数骤降（目录改名 / 扩展名变化）时，本门禁会以
	// 「0 文件 0 命中」静默全绿，必须在扫描前就失败（基线 97 个生产源文件，2026-10-16）。
	if len(files) < 50 {
		t.Fatalf("只扫到 %d 个前端生产源文件（基线 ≥50）：扫描口径疑似塌缩", len(files))
	}
	return files
}

// scanWebXSSSinks 返回源码里全部 HTML 汇聚点（先剥离注释，行号与原文件对齐）。
func scanWebXSSSinks(code string) []webXSSHit {
	stripped := stripSourceComments(code)
	var hits []webXSSHit
	for _, loc := range webXSSSinkRe.FindAllStringIndex(stripped, -1) {
		hits = append(hits, webXSSHit{
			kind: stripped[loc[0]:loc[1]],
			line: 1 + strings.Count(stripped[:loc[0]], "\n"),
		})
	}
	return hits
}

// stripSourceComments 把 `//`、`/* */`、`<!-- -->` 三类注释替换为**等量空白**（保留换行），
// 使剥离后的行号与原文件一一对应。
//
// 只在「可开启注释的位置」识别标记：行首、空白之后，或 `; ( ) { } [ ] , > =` 之后——
// 真实注释都满足，而字符串里的 `https://…`（前一个字符是 `:`）与 `'src/**'`（前一个是
// 字母）不满足，故不会被误剥成注释（后者若被吞掉，可能连带吞掉后面真正的汇聚点）。
func stripSourceComments(src string) string {
	var out strings.Builder
	out.Grow(len(src))
	blank := func(s string) {
		for i := 0; i < len(s); i++ {
			if s[i] == '\n' {
				out.WriteByte('\n')
			} else {
				out.WriteByte(' ')
			}
		}
	}
	for i := 0; i < len(src); {
		rest := src[i:]
		switch {
		case strings.HasPrefix(rest, "<!--") && atCommentStart(src, i):
			end := strings.Index(rest[4:], "-->")
			if end < 0 {
				blank(rest)
				i = len(src)
				continue
			}
			blank(rest[:4+end+3])
			i += 4 + end + 3
		case strings.HasPrefix(rest, "/*") && atCommentStart(src, i):
			end := strings.Index(rest[2:], "*/")
			if end < 0 {
				blank(rest)
				i = len(src)
				continue
			}
			blank(rest[:2+end+2])
			i += 2 + end + 2
		case strings.HasPrefix(rest, "//") && atCommentStart(src, i):
			end := strings.IndexByte(rest, '\n')
			if end < 0 {
				blank(rest)
				i = len(src)
				continue
			}
			blank(rest[:end])
			i += end
		default:
			out.WriteByte(src[i])
			i++
		}
	}
	return out.String()
}

// atCommentStart 判断位置 i 的注释标记是否处于可开启注释的位置（行首 / 空白 / 分隔符之后）。
func atCommentStart(src string, i int) bool {
	if i == 0 {
		return true
	}
	switch src[i-1] {
	case ' ', '\t', '\r', '\n', ';', '(', ')', '{', '}', '[', ']', ',', '>', '=':
		return true
	}
	return false
}
