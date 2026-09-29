package main

// doc_link_gate_test.go —— 全仓 Markdown **相对链接目标存在性 + 页内锚点存在性**的源码门禁。
//
// 背景（docs/archive/review-2026-09-19.md §7.3「其余文档失真」）：
// `doc_number_gate_test.go` 只钉住「N 个端点」这类数字；`config_doc_gate_test.go` 只钉住
// `S3C_*` 变量名。而**文档之间的链接**长期没有机械校验——重命名 / 移动一个文档后，
// 引用它的 md 会留下死链（GitHub 上表现为 404 或跳转失败），而全部 Go/TS 门禁照样全绿。
// 本门禁把「链接可达」变成红灯。
//
// 断言范围（本门禁**只**做这两件事）：
//   - 全仓 `*.md` 中每一个**相对**链接（`[text](target)`）解析出的目标文件存在；
//     目标是目录时，断言该目录存在（GitHub 会把目录链接渲染成目录列表页）。
//   - 带 `#fragment` 的链接：锚点必须在目标文件（无路径片段时为本文件）的标题里真实存在，
//     按 GitHub slug 规则做机械推导（见 ghSlug / ghSlugger，含重复标题的 `-1` / `-2` 后缀）。
//
// 跳过：外链（`http://` / `https://`）、`mailto:` / `tel:`、页内 `#frag` 以外的纯锚点也校验。
// 扫描跳过目录：`node_modules/`、`.git/`、`dist/`、`coverage/`、`.gitlab-ci-local/`、
// `test-results/`（第三方与生成物，不属于本仓库维护的文档面）。
//
// 已知盲区（刻意不做的，避免后来者误判覆盖面）：
//   - **不校验外链可达性**：需要网络，且上游页面变动与本仓库改动无关，放进单测只会 flaky。
//   - **不校验大小写不敏感文件系统上的冲突**：本门禁按运行时文件系统语义解析——在
//     macOS / Windows 上 `[x](Docs/DEVELOPMENT.md)` 能「找到」`docs/DEVELOPMENT.md`，
//     在 Linux / CI 上则红灯。仓库级禁令见根 `AGENTS.md`（任何两个路径不得仅大小写不同），
//     需要跨平台一致性时应另设静态检查，而不是依赖本门禁。
//   - **不校验引用式链接**（`[text][ref]` / `[ref]: url`）与 HTML `<a href>`：当前仓库未使用；
//     一旦引入，本门禁会静默漏掉它们（`scannedLinks` 自检只能发现「总量塌缩」，不能发现个别漏网）。
//   - **不校验 anchor 之外的目标语义**（如链接指向的文件内容是否相关），属人工审查。
//   - **slug 规则不实现 Unicode NFKC 归一化**：GitHub 会先做 NFKC；本仓库标题若引入全角字符
//     等需要归一化的形态，slug 推导可能与 GitHub 不一致——届时需同步实现（Go 标准库无 NFKC）。

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"unicode"
)

// mdSkipDirs 是链接扫描跳过的目录名（第三方依赖、VCS 元数据与构建/测试产物）。
var mdSkipDirs = map[string]bool{
	"node_modules":     true,
	".git":             true,
	"dist":             true,
	"coverage":         true,
	".gitlab-ci-local": true,
	"test-results":     true,
}

// mdLinkRe 匹配行内 Markdown 链接 `[text](target)` 或 `[text](<target>)`，可带 `"title"`。
// 组 1 = 链接文本，组 2 = 目标（不含尖括号），组 3 = 目标之后的 title（可为空）。
// 刻意不匹配图片 `![alt](src)`：`!` 已由前一个字符判别（见 mdLinkSite.img）。
var mdLinkRe = regexp.MustCompile(`\[([^\]]*)\]\(\s*(?:<([^>]*)>|([^)\s]*))((?:\s+"[^"]*")?\s*)\)`)

// mdHeadingRe 匹配 ATX 标题（`## text`，可带行尾闭合 `#`）。
var mdHeadingRe = regexp.MustCompile(`(?m)^[ \t]{0,3}(#{1,6})[ \t]+(.*?)[ \t]*#*[ \t]*$`)

// htmlTagRe 匹配标题文本里的 HTML 标签（GitHub 先剥离标签再算 slug）。
var htmlTagRe = regexp.MustCompile(`<[^>]*>`)

// mdLinkSite 是一条待校验的链接（保留文件与行号，供红灯点名）。
type mdLinkSite struct {
	file string // 相对仓库根的路径（/ 分隔）
	line int    // 1 起
	tgt  string // 原始目标文本
	img  bool   // 图片链接（`![alt](src)`）：同样校验目标存在
}

// repoMarkdownFiles 返回仓库根下全部 `.md`（相对路径，/ 分隔，升序）。
func repoMarkdownFiles(t *testing.T) []string {
	t.Helper()
	root := repoRoot(t)
	var files []string
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if mdSkipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		rel, relErr := filepath.Rel(root, p)
		if relErr != nil {
			return relErr
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Fatalf("遍历仓库 md 文件: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("未扫到任何 .md 文件（扫描口径失效）")
	}
	return files
}

// lineOf 返回 off 之前出现的换行数 + 1（供红灯点名行号）。
func lineOf(b []byte, off int) int {
	return 1 + strings.Count(string(b[:off]), "\n")
}

// collectMdLinks 抽取全部 md 文件的链接（含图片）与全部标题锚点。
func collectMdLinks(t *testing.T) (links []mdLinkSite, anchors map[string]map[string]bool) {
	t.Helper()
	root := repoRoot(t)
	anchors = map[string]map[string]bool{}
	for _, rel := range repoMarkdownFiles(t) {
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("读取 %s: %v", rel, err)
		}
		text := string(b)
		for _, loc := range mdLinkRe.FindAllStringSubmatchIndex(text, -1) {
			tgt := ""
			if loc[4] >= 0 {
				tgt = text[loc[4]:loc[5]] // `<...>` 形态
			} else {
				tgt = text[loc[6]:loc[7]]
			}
			links = append(links, mdLinkSite{
				file: rel,
				line: lineOf(b, loc[0]),
				tgt:  strings.TrimSpace(tgt),
				img:  loc[0] > 0 && text[loc[0]-1] == '!',
			})
		}
		anchors[rel] = headingAnchors(text)
	}
	return links, anchors
}

// headingAnchors 推导一个 md 文件里全部标题的 GitHub slug（含重复标题的 `-1` 后缀）。
func headingAnchors(text string) map[string]bool {
	out := map[string]bool{}
	sl := newGHSlugger()
	for _, m := range mdHeadingRe.FindAllStringSubmatch(text, -1) {
		out[sl.slug(m[2])] = true
	}
	return out
}

// ghSlug 按 GitHub 的 slug 规则把标题文本转成锚点。
func ghSlug(heading string) string {
	s := strings.ToLower(strings.TrimSpace(heading))
	s = htmlTagRe.ReplaceAllString(s, "")
	s = strings.Map(func(r rune) rune {
		if r == ' ' || r == '-' || r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return -1
	}, s)
	return strings.ReplaceAll(s, " ", "-")
}

// ghSlugger 复刻 GitHub 的标题锚点生成器：同名 slug 追加 `-1` / `-2`…（从第二次出现起计数）。
type ghSlugger struct {
	seen map[string]int
}

// newGHSlugger 构造一个空的 slug 生成器。
func newGHSlugger() *ghSlugger { return &ghSlugger{seen: map[string]int{}} }

// slug 返回标题文本对应的 GitHub 锚点。
func (s *ghSlugger) slug(heading string) string {
	base := ghSlug(heading)
	n := s.seen[base]
	s.seen[base] = n + 1
	if n == 0 {
		return base
	}
	return base + "-" + strconv.Itoa(n)
}

// isExternalLink 判定目标是否属于跳过面（外链 / mailto / tel）。
func isExternalLink(tgt string) bool {
	for _, p := range []string{"http://", "https://", "mailto:", "tel:"} {
		if strings.HasPrefix(tgt, p) {
			return true
		}
	}
	return strings.Contains(tgt, "://")
}

// linkTargetExists 断言相对链接目标（文件或目录）存在；返回目标相对仓库根的路径。
func linkTargetExists(t *testing.T, root, fromFile, tgt string) (string, bool) {
	t.Helper()
	// 去掉查询串与锚点；GitHub 对 md 不使用 `?query`，但为稳妥剥掉。
	pathPart, _, _ := strings.Cut(tgt, "#")
	pathPart, _, _ = strings.Cut(pathPart, "?")
	if pathPart == "" {
		return fromFile, true // 纯锚点：目标即本文件
	}
	abs := filepath.Join(root, filepath.FromSlash(filepath.Dir(fromFile)), filepath.FromSlash(pathPart))
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		rel = abs
	}
	if _, err := os.Stat(abs); err != nil {
		return filepath.ToSlash(rel), false
	}
	return filepath.ToSlash(rel), true
}

// minScannedMarkdownLinks 是扫描面自检阈值。
//
// 自检纪律同 `doc_number_gate_test.go`（命中 0 次即红灯）/ `config_doc_gate_test.go`
// （抽不到变量即红灯）：实测基线（2026-09-29）为 41 个 md / 1199 条链接（其中 24 条外链）
// / 70 条带锚点。阈值取 1000 留出正常增删余量；一旦低于它，说明扫描口径塌缩
// （正则被改坏、跳过目录被误加、repoRoot 定位漂移），此时**必须红灯**而不是安静通过。
const minScannedMarkdownLinks = 1000

// minScannedMarkdownAnchors 是带锚点链接的扫描面自检阈值（实测基线 70 条，阈值取 50）。
const minScannedMarkdownAnchors = 50

// TestMarkdownRelativeLinksResolve 断言全仓 md 的每个相对链接目标（文件或目录）存在。
func TestMarkdownRelativeLinksResolve(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	links, _ := collectMdLinks(t)
	if len(links) < minScannedMarkdownLinks {
		t.Fatalf("只扫到 %d 条 Markdown 链接（阈值 %d）：扫描口径已失效或被误缩窄，"+
			"门禁会「全绿但失明」——请先修口径再谈结果", len(links), minScannedMarkdownLinks)
	}
	checked := 0
	for _, l := range links {
		if isExternalLink(l.tgt) {
			continue
		}
		checked++
		target, ok := linkTargetExists(t, root, l.file, l.tgt)
		if !ok {
			t.Errorf("%s:%d 相对链接目标不存在：%s（解析为 %s）", l.file, l.line, l.tgt, target)
		}
	}
	t.Logf("校验 %d 条相对链接目标均存在（另有 %d 条外链跳过）", checked, len(links)-checked)
}

// TestMarkdownAnchorsResolve 断言全仓 md 中每个 `路径#锚点` 的锚点在目标文件真实存在。
func TestMarkdownAnchorsResolve(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	links, anchors := collectMdLinks(t)
	if len(links) < minScannedMarkdownLinks {
		t.Fatalf("只扫到 %d 条 Markdown 链接（阈值 %d）：扫描口径已失效或被误缩窄",
			len(links), minScannedMarkdownLinks)
	}
	checked := 0
	for _, l := range links {
		if isExternalLink(l.tgt) {
			continue
		}
		_, frag, hasFrag := strings.Cut(l.tgt, "#")
		if !hasFrag || frag == "" {
			continue
		}
		checked++
		// 锚点目标：无路径片段时是本文件；带路径时按同一解析口径定位（目标必须已存在于
		// 上一条用例，这里只关心是否有该标题；目标不存在时上一条用例已红灯）。
		target, ok := linkTargetExists(t, root, l.file, l.tgt)
		if !ok {
			continue
		}
		avail, scanned := anchors[target]
		if !scanned {
			if strings.HasSuffix(target, ".md") {
				t.Errorf("%s:%d 锚点 #%s 指向 %s，但该文件未被扫描（口径需同步）", l.file, l.line, frag, target)
			}
			continue
		}
		if !avail[frag] {
			t.Errorf("%s:%d 锚点不存在：#%s → %s（按 GitHub slug 规则推导后无此标题）",
				l.file, l.line, frag, target)
		}
	}
	if checked < minScannedMarkdownAnchors {
		t.Fatalf("只校验到 %d 条带锚点链接（阈值 %d）：锚点扫描口径已失效",
			checked, minScannedMarkdownAnchors)
	}
	t.Logf("校验 %d 条带锚点链接", checked)
}
