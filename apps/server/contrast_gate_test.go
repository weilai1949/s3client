package main

// contrast_gate_test.go —— `docs/accessibility.md` §5.5 对比度核查记录的「记录 ↔ 源码」一致性门禁。
//
// 背景（2026-09-30）：§5.5 的对比度表是**静态计算**记录，计算输入是 `apps/web/src/styles.css` 的
// 设计 token，方法（WCAG 2.1 相对亮度）写在表头。改任何一个颜色而不重算，记录立刻变成假数字；
// 「改色后记得重算」只是口头约定——本轮对账就实测抓到两处：表里 `--danger` 5.98 / `--ok` 5.02
// 是渲染态 axe 读数、不是表中声明的 token 公式值（应为 5.91 / 4.95），以及「4 组低于 AA」
// 的计数漏行。故把记录钉在源码上：改色不重算即红灯。
//
// 断言范围：
//   - §5.5 表格每一行的浅色 / 深色两格数值 == 按 WCAG 2.1 相对亮度从 styles.css 重算的比值
//     （四舍五入 2 位小数，容差 0.011）；深色 `rgba()` 覆层按 alpha 合成到 `--panel`
//     （与 §5.5 方法注释同一口径）；
//   - 「白字 on `--brand` 渐变」行的注释数字（浅端 `--brand-from` 比值）同样被重算钉住；
//   - 结论句的计数声明（`共 **N** 行` / `浅色主题 **N** 行低于 AA` / `深色主题 **N** 行低于 AA`）
//     与按 4.5:1 阈值重算的行数一致；
//   - 扫描面自检：解析出的表格行数 / token 数低于阈值即 Fatal（防解析口径塌缩后全绿但失明）。
//
// 不断言：判定列的自然语言措辞、渲染态 axe 读数（`e2e/a11y.spec.ts` 承担）、组件临时配色与
// 图片 / 阴影上的对比度——「扫描通过 ≠ 全站达标」的口径写在 §5.5 结论。
//
// 变异验证（复核步骤，2026-09-30 已实跑）：① 把 styles.css 的 `--ok` 由 `#047857` 改成 `#10b981`
// → 红灯点名「--ok on --ok-bg 浅色格记 4.95，重算为 2.29」；② 把 §5.5 结论句的 `共 **11 行**`
// 改成 `共 **10 行**` → 计数红灯点名；两者还原后绿灯。复核命令：
// `cd apps/server && go test . -run TestContrast -count=1`。
// ⚠️ 注意容差：本门禁按 `contrastTolerance`（±0.011）比对，**单通道 ±1 的微调不会触发**——
// 它拦的是「改色后忘记重算」，不是「必须逐位重算」。

import (
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

const (
	stylesCSSRel    = "apps/web/src/styles.css"
	a11yDocRel      = "docs/accessibility.md"
	contrastSecTag  = "### 5.5 对比度核查记录"
	minContrastRows = 10
	minThemeTokens  = 20
	aaTextRatio     = 4.5
	// contrastTolerance 是「文档格值 vs 重算值」的容差（两侧都四舍五入到 2 位小数）。
	contrastTolerance = 0.011
)

var (
	// cssVarRe 匹配 `--token: <值>;`（值只认 6 位 hex 与 rgba()，其余形态不参与对比度计算）。
	cssVarRe = regexp.MustCompile(`(--[a-z0-9-]+)\s*:\s*(#[0-9a-fA-F]{6}|rgba\([^)]*\))\s*;`)
	// rgbaRe 解析 `rgba(r, g, b, a)`（a 允许 `.8` 形态）。
	rgbaRe = regexp.MustCompile(`^rgba\(\s*(\d+)\s*,\s*(\d+)\s*,\s*(\d+)\s*,\s*([0-9.]+)\s*\)$`)
	// cellNumRe 从表格数值格里取第一个数（格里允许 `**5.91** ✅` 这类标注）。
	cellNumRe = regexp.MustCompile(`([0-9]+(?:\.[0-9]+)?)`)
	// rowTokenRe 取行首格里的反引号 token 名（`--text` → 捕获 `text`）。
	rowTokenRe = regexp.MustCompile("`--([a-z0-9-]+)`")
	// 计数声明（措辞变更需同步本门禁）。
	totalRowsClaimRe = regexp.MustCompile(`共 \*\*(\d+) 行\*\*`)
	lightClaimRe     = regexp.MustCompile(`浅色主题 \*\*(\d+) 行\*\*低于 AA`)
	darkClaimRe      = regexp.MustCompile(`深色主题 \*\*(\d+) 行\*\*低于 AA`)
)

// rgbColor 是 0–255 的 sRGB 通道值。
type rgbColor [3]float64

// contrastDocRow 是 §5.5 表格的一行（fg/bg 为 token 名；fg=="white" 表示白字）。
type contrastDocRow struct {
	fg, bg string
	light  float64
	dark   float64
	line   string
}

func parseHexColor(s string) (rgbColor, bool) {
	if len(s) != 7 || s[0] != '#' {
		return rgbColor{}, false
	}
	var c rgbColor
	for i := 0; i < 3; i++ {
		v, err := strconv.ParseUint(s[1+2*i:3+2*i], 16, 8)
		if err != nil {
			return rgbColor{}, false
		}
		c[i] = float64(v)
	}
	return c, true
}

func parseRGBAColor(s string) (c rgbColor, alpha float64, ok bool) {
	m := rgbaRe.FindStringSubmatch(strings.ReplaceAll(s, " ", ""))
	if m == nil {
		return rgbColor{}, 0, false
	}
	for i := 0; i < 3; i++ {
		v, err := strconv.ParseFloat(m[i+1], 64)
		if err != nil {
			return rgbColor{}, 0, false
		}
		c[i] = v
	}
	aText := m[4]
	if strings.HasPrefix(aText, ".") {
		aText = "0" + aText // css 允许 `.8` 形态的 alpha
	}
	a, err := strconv.ParseFloat(aText, 64)
	if err != nil {
		return rgbColor{}, 0, false
	}
	return c, a, true
}

// overOut 把带 alpha 的前景色合成到不透明背景（深色 rgba() 覆层的口径，见 §5.5 方法注释）。
func overOut(fg rgbColor, alpha float64, bg rgbColor) rgbColor {
	var out rgbColor
	for i := 0; i < 3; i++ {
		out[i] = alpha*fg[i] + (1-alpha)*bg[i]
	}
	return out
}

func relLuminance(c rgbColor) float64 {
	lin := func(v float64) float64 {
		s := v / 255
		if s <= 0.04045 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(c[0]) + 0.7152*lin(c[1]) + 0.0722*lin(c[2])
}

func contrastRatio(a, b rgbColor) float64 {
	la, lb := relLuminance(a), relLuminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }

// cssBlock 截取 marker 之后到第一个顶层 `}` 之前的声明块（`:root {` / `:root[data-theme='dark'] {`）。
func cssBlock(text, marker string) string {
	i := strings.Index(text, marker)
	if i < 0 {
		return ""
	}
	rest := text[i+len(marker):]
	if j := strings.Index(rest, "\n}"); j >= 0 {
		return rest[:j]
	}
	return rest
}

// themeColors 解析一个主题块：hex 直接用，rgba() 按 alpha 合成到同主题的 `--panel`。
func themeColors(block string) map[string]rgbColor {
	raw := map[string]string{}
	for _, m := range cssVarRe.FindAllStringSubmatch(block, -1) {
		raw[m[1]] = m[2]
	}
	panel := rgbColor{255, 255, 255}
	if v, ok := raw["--panel"]; ok {
		if c, ok2 := parseHexColor(v); ok2 {
			panel = c
		}
	}
	out := map[string]rgbColor{}
	for name, v := range raw {
		if c, ok := parseHexColor(v); ok {
			out[name] = c
			continue
		}
		if c, a, ok := parseRGBAColor(v); ok {
			out[name] = overOut(c, a, panel)
		}
	}
	return out
}

// mergeTheme 叠加主题覆盖（CSS 级联：深色块未重定义的 token 继承浅色值，如 `--brand-*`）。
func mergeTheme(base, override map[string]rgbColor) map[string]rgbColor {
	out := make(map[string]rgbColor, len(base)+len(override))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range override {
		out[k] = v
	}
	return out
}

// contrastDocSection 截取 §5.5 小节（到下一个同级 `## ` 标题或文末）。
func contrastDocSection(t *testing.T, doc string) string {
	t.Helper()
	i := strings.Index(doc, contrastSecTag)
	if i < 0 {
		t.Fatalf("docs/accessibility.md 未找到 %q 小节：对比度记录被移除或改名，需同步本门禁", contrastSecTag)
	}
	rest := doc[i:]
	if j := strings.Index(rest[1:], "\n## "); j >= 0 {
		return rest[:j+1]
	}
	return rest
}

// parseContrastRows 解析 §5.5 表格行（表头 / 分隔行跳过）。
func parseContrastRows(t *testing.T, section string) []contrastDocRow {
	t.Helper()
	var rows []contrastDocRow
	for _, line := range strings.Split(section, "\n") {
		if !strings.HasPrefix(line, "|") || strings.Contains(line, "组合（前景") {
			continue
		}
		cells := strings.Split(line, "|")
		if len(cells) < 5 || strings.Contains(cells[1], "---") {
			continue // 表头 / 分隔行
		}
		toks := rowTokenRe.FindAllStringSubmatch(cells[1], -1)
		row := contrastDocRow{line: strings.TrimSpace(line)}
		switch {
		case strings.HasPrefix(strings.TrimSpace(cells[1]), "白字"):
			row.fg = "white"
			if len(toks) == 0 {
				t.Errorf("§5.5 行缺背景 token：%s", row.line)
				continue
			}
			row.bg = toks[len(toks)-1][1]
		default:
			if len(toks) < 2 {
				t.Errorf("§5.5 行未解析出前景/背景 token（需形如 `--text` on `--bg`）：%s", row.line)
				continue
			}
			row.fg, row.bg = toks[0][1], toks[1][1]
		}
		for _, cell := range []struct {
			text string
			dst  *float64
		}{{cells[2], &row.light}, {cells[3], &row.dark}} {
			m := cellNumRe.FindStringSubmatch(cell.text)
			if m == nil {
				t.Errorf("§5.5 行数值格无数值：%s", row.line)
				continue
			}
			v, err := strconv.ParseFloat(m[1], 64)
			if err != nil {
				t.Errorf("§5.5 行数值格无法解析 %q：%s", m[1], row.line)
				continue
			}
			*cell.dst = v
		}
		rows = append(rows, row)
	}
	return rows
}

// tokenColor 取主题里的 token 颜色（fg=="white" 特判为白字）。
func tokenColor(t *testing.T, theme map[string]rgbColor, token string) (rgbColor, bool) {
	t.Helper()
	if token == "white" {
		return rgbColor{255, 255, 255}, true
	}
	c, ok := theme["--"+token]
	if !ok {
		t.Errorf("styles.css 缺少 token --%s（§5.5 表引用了它）", token)
		return rgbColor{}, false
	}
	return c, true
}

// TestContrastRecordMatchesStylesTokens 断言 §5.5 每行两格数值与 styles.css 重算值一致。
func TestContrastRecordMatchesStylesTokens(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	cssB, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(stylesCSSRel)))
	if err != nil {
		t.Fatalf("读取 %s: %v", stylesCSSRel, err)
	}
	css := string(cssB)
	light := themeColors(cssBlock(css, ":root {"))
	dark := mergeTheme(light, themeColors(cssBlock(css, ":root[data-theme='dark'] {")))
	for name, theme := range map[string]map[string]rgbColor{"浅色": light, "深色": dark} {
		if len(theme) < minThemeTokens {
			t.Fatalf("%s主题只解析出 %d 个 token（阈值 %d）：解析口径失效", name, len(theme), minThemeTokens)
		}
	}

	section := contrastDocSection(t, readRepoFile(t, filepath.FromSlash(a11yDocRel)))
	rows := parseContrastRows(t, section)
	if len(rows) < minContrastRows {
		t.Fatalf("§5.5 只解析出 %d 行（阈值 %d）：表格被删或解析口径失效", len(rows), minContrastRows)
	}

	for _, row := range rows {
		for _, th := range []struct {
			name   string
			colors map[string]rgbColor
			doc    float64
		}{{"浅色", light, row.light}, {"深色", dark, row.dark}} {
			fg, ok := tokenColor(t, th.colors, row.fg)
			if !ok {
				continue
			}
			bg, ok := tokenColor(t, th.colors, row.bg)
			if !ok {
				continue
			}
			calc := round2(contrastRatio(fg, bg))
			if math.Abs(th.doc-calc) > contrastTolerance {
				t.Errorf("§5.5「--%s on --%s」%s格记 %.2f，按 WCAG 2.1 从 styles.css 重算为 %.2f："+
					"改色后未重算或记录口径漂移（本表统一取 token 公式值）", row.fg, row.bg, th.name, th.doc, calc)
			}
		}
		// 白字渐变行的注释数字（浅端 `--brand-from`）同样被钉住。
		if row.fg == "white" {
			from, ok := tokenColor(t, light, "brand-from")
			if !ok {
				continue
			}
			want := strconv.FormatFloat(round2(contrastRatio(rgbColor{255, 255, 255}, from)), 'f', 2, 64)
			if !strings.Contains(row.line, want) {
				t.Errorf("§5.5 白字渐变行缺浅端 `--brand-from` 的重算值 %s（或数值已过期）：%s", want, row.line)
			}
		}
	}
}

// TestContrastRecordCountClaimsMatch 断言 §5.5 结论句的计数声明与按 4.5:1 重算的行数一致。
func TestContrastRecordCountClaimsMatch(t *testing.T) {
	t.Parallel()
	// 计数也按 styles.css 重算（阈值 4.5:1），不取文档格值——计数声明与表值一起绑在源码上。
	cssB, err := os.ReadFile(filepath.Join(repoRoot(t), filepath.FromSlash(stylesCSSRel)))
	if err != nil {
		t.Fatalf("读取 %s: %v", stylesCSSRel, err)
	}
	css := string(cssB)
	light := themeColors(cssBlock(css, ":root {"))
	dark := mergeTheme(light, themeColors(cssBlock(css, ":root[data-theme='dark'] {")))
	section := contrastDocSection(t, readRepoFile(t, filepath.FromSlash(a11yDocRel)))
	rows := parseContrastRows(t, section)
	if len(rows) < minContrastRows {
		t.Fatalf("§5.5 只解析出 %d 行（阈值 %d）：解析口径失效", len(rows), minContrastRows)
	}

	var lightBelow, darkBelow int
	for _, row := range rows {
		for _, th := range []struct {
			colors map[string]rgbColor
			below  *int
		}{{light, &lightBelow}, {dark, &darkBelow}} {
			fg, ok := tokenColor(t, th.colors, row.fg)
			if !ok {
				continue
			}
			bg, ok := tokenColor(t, th.colors, row.bg)
			if !ok {
				continue
			}
			if contrastRatio(fg, bg) < aaTextRatio {
				*th.below++
			}
		}
	}
	for _, cl := range []struct {
		name string
		re   *regexp.Regexp
		want int
	}{
		{"「共 N 行」", totalRowsClaimRe, len(rows)},
		{"「浅色主题 N 行低于 AA」", lightClaimRe, lightBelow},
		{"「深色主题 N 行低于 AA」", darkClaimRe, darkBelow},
	} {
		m := cl.re.FindStringSubmatch(section)
		if m == nil {
			t.Errorf("§5.5 缺计数声明 %s（措辞变更需同步本门禁）", cl.name)
			continue
		}
		got, err := strconv.Atoi(m[1])
		if err != nil {
			t.Errorf("§5.5 计数声明 %s 无法解析：%v", cl.name, err)
			continue
		}
		if got != cl.want {
			t.Errorf("§5.5 记 %s=%d，按 %.1f:1 从表格重算为 %d：计数漏行或改色后未更新",
				cl.name, got, aaTextRatio, cl.want)
		}
	}
}
