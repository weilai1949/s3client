package main

// en_docs_gate_test.go —— 英文文档面（`docs/en/`）门禁。
//
// 背景（2026-09-30）：`docs/en/` 此前只有 `index.md`（根 `README.md` 的全量英文翻译），
// `docs/i18n.md` 只管界面文案 key，对**文档翻译**没有任何策略，也没有门禁保证：
//   - 英文页能否追溯到中文源（翻的是哪一版、源是谁）；
//   - 英文页有没有导航入口（`doc_index_gate_test.go` 只认子目录 `index.md` 或 `docs/README.md`
//     的直接链接，管不到「英文页之间是否互相可达」）；
//   - 根 `README.md` 变更后英文翻译快照里的关键事实（版本号 / 端点计数）是否跟着变。
//
// 断言范围（只做这四件事）：
//   - (a) `docs/en/` 下每个页面（`index.md` 除外，它是根 README 的快照、源在仓库根）都
//     声明中文源（`**Source (Chinese SSOT)**:` + 相对链接）与源修订（`**Source revision**:` +
//     hash + 日期），且源文件存在于 `docs/` 下、不在 `docs/en/` 内；
//   - (b) `docs/en/` 下每个页面都能从英文导航 `docs/en/README.md` 或中文 docs 落地页
//     `docs/README.md` 到达（复用 `doc_index_gate_test.go` 的 `navTargets` 口径）；
//   - (c) `docs/en/index.md` 仍跟随根 `README.md`：版本字面量与「N 个 `/api/*` 端点」机械一致，
//     且链回根 README（SSOT 回指）；
//   - (d) 扫描面自检阈值：英文页数 / 导航链接数低于基线即红灯，防「空扫变绿」。
//
// 另有一条**翻译完整性**校验：已翻译页必须保留中文源的每个相对链接目标（路径 / ADR 链接逐字保留）；
// 漏译整段往往同时漏链，因此这条能把「只翻了一半」变成红灯。
//
// 已知盲区（刻意不做）：
//   - **不校验正文语义等价**——那是自然语言，属人工审查；本门禁只钉结构与可机械推导的事实。
//   - **不校验 `Source revision` 是否过期**——git 历史与工作区内容的口径难以在单测里稳定复现；
//     漂移处理靠「同 PR 更新」的人工纪律 + 本节 (c) 的两条机械比对（见 `docs/i18n.md` §7.4）。
//   - **不校验英文页之间的锚点**——`doc_link_gate_test.go` 已全仓校验链接与锚点，本门禁不重复。
//
// 变异验证（复核步骤，实测见任务报告）：
//   摘掉 `docs/en/architecture.md` 的 `**Source (Chinese SSOT)**:` 行 → 本门禁红灯并点名该文件；
//   摘掉 `docs/en/README.md` 里指向 `index.md` 的链接 → 可达性门禁红灯并点名 `docs/en/index.md`；
//   还原后绿灯。
//
// 相关：`doc_index_gate_test.go`（导航覆盖）、`doc_link_gate_test.go`（链接与锚点）、
// `docs_naming_gate_test.go`（命名登记）——四者分别保证「英文页有源」「有入口」「链接不死」「目录登记」。

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

const (
	// enDocsDir 是英文文档目录（相对仓库根）。
	enDocsDir = "docs/en"
	// enDocsNav 是英文导航落地页：GitHub 渲染目录 `README.md`，读者从它进入全部英文页。
	enDocsNav = "docs/en/README.md"
	// enIndexFile 是根 `README.md` 的翻译快照；它是唯一豁免「声明中文源」的英文页（源在仓库根）。
	enIndexFile = "docs/en/index.md"
	// enRootReadme 是英文翻译面要跟随的根 README。
	enRootReadme = "README.md"

	// enSourceMarker / enRevMarker 是每个英文页文首必须声明的内容（格式见 docs/i18n.md §7.3）。
	enSourceMarker = "**Source (Chinese SSOT)**:"
	enRevMarker    = "**Source revision**:"
)

// 扫描面自检阈值（实测 2026-09-30：英文页 3 个、en/README.md 链接目标 20+、根 README 链接目标 30+）。
// 低于阈值说明扫描口径塌缩（目录改名 / WalkDir 失效 / 正则被改坏），此时必须红灯而不是安静通过。
const (
	minEnDocsScanned            = 3
	minEnSourceDeclaredPages    = 2
	minEnNavLinkTargets         = 8
	minEnTranslationSourceLinks = 5
	minRootReadmeLinkTargets    = 20
	minEnIndexLinkTargets       = 15
	minEnScanSurfaceLinkTargets = 20
)

var (
	// enRevisionRe 匹配 `**Source revision**: \`<hex>\` (YYYY-MM-DD)`。
	enRevisionRe = regexp.MustCompile("\\*\\*Source revision\\*\\*:\\s*`([0-9a-f]{7,40})`\\s*\\(([0-9]{4}-[0-9]{2}-[0-9]{2})\\)")
	// zhVersionRe / enVersionRe 分别抓根 README 与英文快照里的版本字面量，用于跨语言比对。
	zhVersionRe = regexp.MustCompile("当前版本\\s*`([^`\\n]+)`")
	enVersionRe = regexp.MustCompile("Current version:?\\s*`([^`\\n]+)`")
	// zhEndpointCountRe / enEndpointCountRe 抓「N 个 `/api/*` 端点」的中英两种写法。
	zhEndpointCountRe = regexp.MustCompile("(\\d+) 个 `/api/\\*` 端点")
	enEndpointCountRe = regexp.MustCompile("(\\d+) `/api/\\*` endpoints?")
)

// enDocsMarkdownFiles 返回 `docs/en/` 下全部 `.md`（相对仓库根，`/` 分隔，升序）。
func enDocsMarkdownFiles(t *testing.T) []string {
	t.Helper()
	root := repoRoot(t)
	var files []string
	err := filepath.WalkDir(filepath.Join(root, filepath.FromSlash(enDocsDir)), func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
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
		t.Fatalf("遍历 %s: %v", enDocsDir, err)
	}
	if len(files) == 0 {
		t.Fatalf("未扫到任何 %s/*.md——扫描口径已失效", enDocsDir)
	}
	sort.Strings(files)
	return files
}

// readEnDocsFile 读取仓库根相对路径的文本；读取失败即 Fatal（文件缺失不应被当成「没违规」）。
func readEnDocsFile(t *testing.T, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repoRoot(t), filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("读取 %s: %v", rel, err)
	}
	return string(b)
}

// enDeclaredSource 解析英文页的 `Source (Chinese SSOT)` 声明行，返回中文源（相对仓库根）。
// 声明行缺失 / 无链接 / 目标非法或不存在时，红灯点名该英文页并返回空串。
func enDeclaredSource(t *testing.T, rel, text string) string {
	t.Helper()
	var line string
	for _, l := range strings.Split(text, "\n") {
		if strings.Contains(l, enSourceMarker) {
			line = l
			break
		}
	}
	if line == "" {
		t.Errorf("%s 缺 %q 声明行：英文页必须声明中文源（SSOT），格式见 docs/i18n.md §7.3", rel, enSourceMarker)
		return ""
	}
	m := mdLinkRe.FindStringSubmatch(line)
	if m == nil {
		t.Errorf("%s 的 %q 行没有相对链接：必须给出指向 docs/ 下中文源文件的链接", rel, enSourceMarker)
		return ""
	}
	tgt := m[3]
	if m[2] != "" {
		tgt = m[2] // `<...>` 形态
	}
	tgt, _, _ = strings.Cut(tgt, "#")
	tgt, _, _ = strings.Cut(tgt, "?")
	if tgt == "" || isExternalLink(tgt) {
		t.Errorf("%s 的源声明链接无效：%q", rel, tgt)
		return ""
	}
	src := filepath.ToSlash(filepath.Clean(filepath.Join(filepath.Dir(rel), filepath.FromSlash(tgt))))
	if !strings.HasPrefix(src, "docs/") || strings.HasPrefix(src, enDocsDir+"/") {
		t.Errorf("%s 声明的源 %s 必须位于 docs/ 下且不在 %s/ 内", rel, src, enDocsDir)
		return ""
	}
	if _, err := os.Stat(filepath.Join(repoRoot(t), filepath.FromSlash(src))); err != nil {
		t.Errorf("%s 声明的中文源 %s 不存在：%v", rel, src, err)
		return ""
	}
	return src
}

// assertEnTranslationKeepsLinks 断言英文翻译页保留了中文源的每个相对链接目标。
// 判据是「链接目标集合」而非链接文本：目标路径必须逐字保留，文本可翻译。
func assertEnTranslationKeepsLinks(t *testing.T, links []mdLinkSite, rel, src string) {
	t.Helper()
	zhTargets := navTargets(links, src)
	if len(zhTargets) < minEnTranslationSourceLinks {
		t.Fatalf("%s 的中文源 %s 只解析到 %d 个相对链接目标（阈值 %d）：链接解析口径可能已失效",
			rel, src, len(zhTargets), minEnTranslationSourceLinks)
	}
	enTargets := navTargets(links, rel)
	var missing []string
	for tgt := range zhTargets {
		if !enTargets[tgt] {
			missing = append(missing, tgt)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("%s 未保留中文源 %s 的 %d 个链接目标（漏译整段常伴随漏链，逐条补回或确认该链接为何不适用）：\n  %s",
			rel, src, len(missing), strings.Join(missing, "\n  "))
	}
}

// TestEnDocsDeclareChineseSource 断言英文页可追溯到中文源，并保留其链接目标（断言 a）。
func TestEnDocsDeclareChineseSource(t *testing.T) {
	t.Parallel()
	files := enDocsMarkdownFiles(t)
	if len(files) < minEnDocsScanned {
		t.Fatalf("只扫到 %d 个 %s/*.md（阈值 %d）：英文文档扫描口径已失效", len(files), enDocsDir, minEnDocsScanned)
	}
	links, _ := collectMdLinks(t)
	declared, translated := 0, 0
	for _, rel := range files {
		if rel == enIndexFile {
			continue // index.md 是根 README 的快照，源在仓库根，按历史约定豁免。
		}
		text := readEnDocsFile(t, rel)
		if !enRevisionRe.MatchString(text) {
			t.Errorf("%s 缺 %q 修订声明（短 commit hash + YYYY-MM-DD 日期）：格式见 docs/i18n.md §7.3",
				rel, enRevMarker)
		}
		src := enDeclaredSource(t, rel, text)
		if src == "" {
			continue
		}
		declared++
		if rel == enDocsNav {
			continue // 导航改编页：只要求声明源，不做逐段翻译的链接奇偶校验。
		}
		assertEnTranslationKeepsLinks(t, links, rel, src)
		translated++
	}
	if declared < minEnSourceDeclaredPages {
		t.Fatalf("只校验到 %d 个声明了中文源的英文页（阈值 %d）：门禁扫描口径已失效",
			declared, minEnSourceDeclaredPages)
	}
	if translated < 1 {
		t.Fatalf("没有任何英文页做了「链接目标保留」校验：翻译完整性检查形同虚设")
	}
}

// TestEnDocsAreReachableFromEnglishNav 断言英文页都能从英文导航或中文 docs 落地页到达（断言 b）。
func TestEnDocsAreReachableFromEnglishNav(t *testing.T) {
	t.Parallel()
	if _, err := os.Stat(filepath.Join(repoRoot(t), filepath.FromSlash(enDocsNav))); err != nil {
		t.Fatalf("英文导航页 %s 不存在：%v（没有它，英文页无路可寻）", enDocsNav, err)
	}
	files := enDocsMarkdownFiles(t)
	if len(files) < minEnDocsScanned {
		t.Fatalf("只扫到 %d 个 %s/*.md（阈值 %d）：英文文档扫描口径已失效", len(files), enDocsDir, minEnDocsScanned)
	}
	links, _ := collectMdLinks(t)
	enNav := navTargets(links, enDocsNav)
	zhNav := navTargets(links, docsNavIndex)
	if len(enNav) < minEnNavLinkTargets {
		t.Fatalf("%s 只解析到 %d 个链接目标（阈值 %d）：英文导航扫描口径已失效",
			enDocsNav, len(enNav), minEnNavLinkTargets)
	}
	var unreachable []string
	for _, f := range files {
		if f == enDocsNav {
			continue // 导航页自身不需要被自己登记。
		}
		if enNav[f] || zhNav[f] {
			continue
		}
		unreachable = append(unreachable, f+"（应登记进 "+enDocsNav+" 或 "+docsNavIndex+"）")
	}
	if len(unreachable) > 0 {
		t.Errorf("以下 %d 个英文页没有任何导航入口——新增英文页必须同时登记进英文导航，"+
			"否则英文读者找不到它：\n  %s", len(unreachable), strings.Join(unreachable, "\n  "))
	}
}

// TestEnDocsIndexTracksRootReadme 断言英文 README 快照仍跟随根 README（断言 c）。
func TestEnDocsIndexTracksRootReadme(t *testing.T) {
	t.Parallel()
	readme := readEnDocsFile(t, enRootReadme)
	index := readEnDocsFile(t, enIndexFile)

	if !strings.Contains(index, "SSOT") {
		t.Errorf("%s 未声明 SSOT：翻译快照必须写清「中文是单一事实来源」", enIndexFile)
	}

	links, _ := collectMdLinks(t)
	readmeTargets := navTargets(links, enRootReadme)
	indexTargets := navTargets(links, enIndexFile)
	if len(readmeTargets) < minRootReadmeLinkTargets {
		t.Fatalf("根 %s 只解析到 %d 个链接目标（阈值 %d）：链接扫描口径已失效",
			enRootReadme, len(readmeTargets), minRootReadmeLinkTargets)
	}
	if len(indexTargets) < minEnIndexLinkTargets {
		t.Fatalf("%s 只解析到 %d 个链接目标（阈值 %d）：链接扫描口径已失效",
			enIndexFile, len(indexTargets), minEnIndexLinkTargets)
	}
	if !indexTargets[enRootReadme] {
		t.Errorf("%s 未链回根 %s：翻译快照必须能从英文页回到中文 SSOT", enIndexFile, enRootReadme)
	}

	zhVersion := enFirstCapture(t, enRootReadme, zhVersionRe, readme)
	enVersion := enFirstCapture(t, enIndexFile, enVersionRe, index)
	if zhVersion != enVersion {
		t.Errorf("版本字面量漂移：根 %s 是 %q，英文快照 %s 是 %q——根 README 变更时英文快照必须同 PR 同步",
			enRootReadme, zhVersion, enIndexFile, enVersion)
	}

	zhCount := enFirstCaptureInt(t, enRootReadme, zhEndpointCountRe, readme)
	enCount := enFirstCaptureInt(t, enIndexFile, enEndpointCountRe, index)
	if zhCount != enCount {
		t.Errorf("端点计数漂移：根 %s 声称 %d 个 `/api/*` 端点，英文快照 %s 声称 %d 个——英文侧未同步",
			enRootReadme, zhCount, enIndexFile, enCount)
	}
}

// TestEnDocsScanSurfaceIsNotCollapsed 是扫描面自检（断言 d）：空扫 / 口径塌缩不得变绿。
func TestEnDocsScanSurfaceIsNotCollapsed(t *testing.T) {
	t.Parallel()
	files := enDocsMarkdownFiles(t)
	if len(files) < minEnDocsScanned {
		t.Fatalf("只扫到 %d 个 %s/*.md（阈值 %d）：英文文档扫描口径已失效", len(files), enDocsDir, minEnDocsScanned)
	}
	links, _ := collectMdLinks(t)
	total := 0
	for _, f := range files {
		total += len(navTargets(links, f))
	}
	if total < minEnScanSurfaceLinkTargets {
		t.Fatalf("%s 下全部英文页只解析到 %d 个链接目标（阈值 %d）：链接解析口径已失效或被误缩窄",
			enDocsDir, total, minEnScanSurfaceLinkTargets)
	}
}

// enFirstCapture 返回 re 在 text 中的第一个捕获组；未命中即 Fatal——措辞漂移会让门禁静默失效。
func enFirstCapture(t *testing.T, file string, re *regexp.Regexp, text string) string {
	t.Helper()
	m := re.FindStringSubmatch(text)
	if m == nil {
		t.Fatalf("%s 未匹配到 %q：文案已漂移，门禁失效——请同步 en_docs_gate_test.go 的正则", file, re.String())
	}
	return m[1]
}

// enFirstCaptureInt 同 enFirstCapture，但要求捕获组是整数。
func enFirstCaptureInt(t *testing.T, file string, re *regexp.Regexp, text string) int {
	t.Helper()
	raw := enFirstCapture(t, file, re, text)
	n, err := strconv.Atoi(raw)
	if err != nil {
		t.Fatalf("%s 捕获组 %q 不是整数: %v", file, raw, err)
	}
	return n
}
