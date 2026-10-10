package main

// doc_status_drift_gate_test.go —— 文档「状态陈述」的源码门禁（跨文档语义一致性）。
//
// 背景（2026-10-10 实测发现）：既有文档门禁分别钉住「链接 / 锚点可达」（doc_link_gate）、
// 「叙述性数字」（doc_number_gate）、「文档引用的 CI 事实」（doc_ci_drift_gate），但**状态陈述**
// ——「某条目还开着 / 某能力未接入」——没有任何机械校验。本门禁上线即抓到两处纯语义漂移：
//   ① `docs/OPERATIONS.md` §4.3 写「跨服务 trace … 未接入（ROADMAP §三 3.2 #11 ⬜）」，
//      而 #11 已于 2026-10-08 落地并按 ROADMAP §六 第 1 条移出转空号（FEATURES §BR），
//      同文件 §3.4 就是已实现的 OTel tracing——一处悬空编号 + 一处与本文件自相矛盾；
//   ② `docs/FEATURES.md` 三处章节头注用**无限定时点的现在时**写「仍开放 / 仍未处置」，
//      与同文件 §Q / §S / §T / §CC / §CD 的闭环记录直接矛盾（读者会把写作时点当成现状）。
//
// 断言范围（三条，各带扫描面自检——命中低于基线即 Fatal，防「正则塌缩后全绿失明」）：
//   1. TestOpenStatusRoadmapRefsPointToLiveEntries：非历史文档中，与**开放状态标记**
//      （⬜ / ⏳ / 未排期 / 未接入）**同一行**的 `§三 [3.2] #N` 引用，N 必须是 ROADMAP
//      §3.1 / §3.2 表内的现存条目。语义：标记 = 「现在还开着」，落地即移出转空号；
//      **不带标记**的引用是历史出处指针（如 `§三 #7` 指向已落地的 FinOps），允许指向空号——
//      故只校验「开放声明」，不校验「引用存在性」。
//   2. TestOperationsTraceRowTracksOTelSection：`docs/OPERATIONS.md` 存在 §3.4 OTel tracing
//      小节时，§4.3 的「跨服务 trace」行不得写「未接入」，且必须点名 `S3C_OTEL_ENDPOINT`
//      （与同文件实现小节保持同一口径）。
//   3. TestFeaturesOpenClaimsCarryTimeAnchor：`docs/FEATURES.md`（**已完成**台账）中出现
//      「仍开放 / 仍未处置」的行必须自带日期（时点锚）；否则读者会把章节写作时点的状态
//      当成当前事实。
//   4. TestDocPageDateClaimNotStale（2026-10-10 交接快照 §5 未做第 10 项）：页头的**时点声明**
//      （`最后更新：YYYY-MM-DD` / `统计时点：YYYY-MM-DD` / `盘点时点`）不得**早于**文内最新的
//      **时点锚**（`上一轮更新` / `复测` / `实跑` / `归档日期` / `评审时点` 等声明式短语旁的日期）——
//      堵「正文改了、页头日期没动」的日期滞后。
//
// 扫描范围与刻意不做的（盲区，避免后来者误判覆盖面）：
//   - 断言 1 排除**历史 / 时点台账**：`docs/archive/**`（归档冻结）、`docs/FEATURES.md` 与
//     `CHANGELOG.md`（「不追溯篡改 / 快照不回写」纪律保留原文）——它们的 `§三 #N + ⏳` 是
//     当时的记录，不是现状声明；
//   - 断言 1 只认 `§三` 前缀的跨文件引用格式（DEVELOPMENT §4 约定），故 `KNOWN_ISSUES #N`
//     与 `ROADMAP §三 #N` 两套编号空间互不误伤；**同行粒度**：同一行既有历史引用又另有
//     开放标记时会误判（当前 6 处实测均无此形态）——遇到就把历史引用换行或去掉标记；
//   - 断言 1 不校验裸编号与 `#a–#b` 范围引用（无 `§三` 前缀无法区分编号空间）；
//   - 断言 3 只认「仍开放 / 仍未处置」两种措辞；行内有日期即通过，**不校验日期本身是否正确
//     （那需要人工读上下文）。
//   - 断言 4 只认**声明式时点锚**（短语旁 12 字内的日期），**刻意不扫正文里的普通日期**——
//     示例载荷（如 `retainUntilDate: 2031-02-03`、`Version: "2012-10-17"`）与计划性未来日期
//     （如「下次审查 2027-03-29」）都不是内容新鲜度声明，扫进来会大面积误报；因此
//     「正文写了个新日期但旁边没有时点短语」仍是盲区（当前 8 篇声明页 / 51 处锚，实测无漂移）；
//     另排除 `CHANGELOG.md`——其历史条目里的「最后更新」是**叙述**（记录当年把哪篇文档的
//     页头推到哪天），不是它自己的页头声明。
//
// 变异验证（2026-10-10 实跑，复核命令：
// `cd apps/server && go test . -run 'TestOpenStatusRoadmapRefs|TestOperationsTraceRow|TestFeaturesOpenClaims|TestDocPageDateClaimNotStale' -count=1 -v`）：
//   - 把 §4.3 的 trace 行改回 `未接入（… §三 3.2 #11 ⬜）` → 断言 1 与 2 双红；
//   - 把 §4.3 的 trace 行改成引用 `§三 3.2 #99 ⬜` → 断言 1 红（编号不在表内）；
//   - 删掉 FEATURES 任一「仍开放 / 仍未处置」行里的日期 → 断言 3 红；
//   - 把 KNOWN_ISSUES 页头 `最后更新：2026-10-10` 改成 `2026-10-08` → 断言 4 红（文内锚 2026-10-09/10）；
//   - 还原后四条全绿。
//
// 相关：`doc_link_gate_test.go`（链到没有）、`doc_number_gate_test.go`（数字对不对）、
// `doc_ci_drift_gate_test.go`（文档引用的 CI 事实真不真）——四者分别管
// 「可达 / 数字 / CI 配置 / 状态」四个维度，互不重复。

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

const (
	docStatusRoadmap  = "docs/ROADMAP.md"
	docStatusOps      = "docs/OPERATIONS.md"
	docStatusFeatures = "docs/FEATURES.md"
	// docStatusMinRefs / docStatusMinOpenRefs 是断言 1 的扫描面自检阈值。
	// 实测基线（2026-10-10，作用域 58 个 md，排除历史台账后）：`§三 #N` 引用 26 处，
	// 其中与开放标记同行 6 处（断言 1 修复后为 5 处）。低于阈值即正则或排除清单被改坏。
	docStatusMinRefs     = 20
	docStatusMinOpenRefs = 4
	// docStatusMinFeatureClaims 是断言 3 的基线：FEATURES 中「仍开放 / 仍未处置」3 行
	// （§P / §Q / §CA 头注，均已加时点锚）。归零说明措辞已收敛，需同步本阈值。
	docStatusMinFeatureClaims = 3
	// docStatusMinLiveEntries 是 ROADMAP §3.1 + §3.2 现存条目数的下限
	//（实测 6：#1 / #2 / #4 / #9 / #15 / #16）。
	docStatusMinLiveEntries = 4
	// docStatusMinDateClaimDocs / docStatusMinDateAnchors 是断言 4 的扫描面自检阈值
	//（实测 2026-10-10：8 篇页头时点声明、51 处文内时点锚）。
	docStatusMinDateClaimDocs = 5
	docStatusMinDateAnchors   = 25
)

var (
	// docStatusRefRe 匹配跨文件 ROADMAP 条目引用：`§三 #N` 或 `§三 3.1/3.2 #N`。
	// 允许 `#17②` 这类子项后缀（只取数字部分）。
	docStatusRefRe = regexp.MustCompile(`§三\s*(?:3\.[12]\s*)?#\s*(\d+)`)
	// docStatusOpenRe 是「现在还开着」的标记集合：状态图例符号 + 常见开放措辞。
	docStatusOpenRe = regexp.MustCompile(`⬜|⏳|未排期|未接入`)
	// docStatusFeatureClaimRe 是 FEATURES（已完成台账）里会被误读成现状的现在时开放陈述。
	docStatusFeatureClaimRe = regexp.MustCompile(`仍开放|仍未处置`)
	// docStatusDateRe 是时点锚：行内出现任一 `YYYY-MM-DD` 即可。
	docStatusDateRe = regexp.MustCompile(`\d{4}-\d{2}-\d{2}`)
	// docStatusLiveRowRe 匹配 ROADMAP 表格的数字首列行（`| 4 | …`）。
	docStatusLiveRowRe = regexp.MustCompile(`^\|\s*(\d+)\s*\|`)
	// docStatusDateClaimRe 匹配页头**时点声明**（最后更新 / 统计时点 / 盘点时点 + 日期）。
	docStatusDateClaimRe = regexp.MustCompile(`(?:最后更新|统计时点|盘点时点)[^\d\n]{0,8}(\d{4}-\d{2}-\d{2})`)
	// docStatusDateAnchorRe 匹配文内的**声明式时点锚**（时点短语 + 12 字内的日期）。
	// 只认声明式短语，刻意不扫普通正文日期（示例载荷 / 计划性未来日期会误报）。
	docStatusDateAnchorRe = regexp.MustCompile(
		`(?:最后更新|上一轮更新|上一轮|统计时点|盘点时点|写作时点|复核时点|复测|实跑|归档日期|评审时点|评审日期|冻结|收口)` +
			`[^\d\n]{0,12}(\d{4}-\d{2}-\d{2})`)
)

// roadmapLiveEntries 解析 ROADMAP §3.1 + §3.2 表内的现存条目编号集合。
// 段落边界标记缺失即 Fatal（措辞变更需同步本门禁），扫描面低于阈值同样 Fatal。
func roadmapLiveEntries(t *testing.T) map[int]bool {
	t.Helper()
	text := readRepoFile(t, docStatusRoadmap)
	start := strings.Index(text, "### 3.1")
	if start < 0 {
		t.Fatalf("%s 未找到起始标记 %q——措辞变更需同步本门禁", docStatusRoadmap, "### 3.1")
	}
	end := strings.Index(text, "## 四")
	if end < 0 || end < start {
		t.Fatalf("%s 未找到终止标记 %q（须位于 §3.1 之后）——措辞变更需同步本门禁",
			docStatusRoadmap, "## 四")
	}

	live := map[int]bool{}
	for _, line := range strings.Split(text[start:end], "\n") {
		m := docStatusLiveRowRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		n, err := strconv.Atoi(m[1])
		if err != nil {
			t.Fatalf("ROADMAP 表行编号解析失败 %q: %v", line, err)
		}
		live[n] = true
	}
	if len(live) < docStatusMinLiveEntries {
		t.Fatalf("扫描面塌缩：ROADMAP §3.1+§3.2 只解析到 %d 个条目（基线 ≥%d）",
			len(live), docStatusMinLiveEntries)
	}
	return live
}

// TestOpenStatusRoadmapRefsPointToLiveEntries 断言 1：带开放标记的 `§三 #N` 引用必须指向
// ROADMAP §3.1 / §3.2 表内的现存条目——落地即移出转空号，「还开着」的声明不能挂在空号上。
func TestOpenStatusRoadmapRefsPointToLiveEntries(t *testing.T) {
	t.Parallel()
	live := roadmapLiveEntries(t)

	total, openRefs := 0, 0
	for _, rel := range repoMarkdownFiles(t) {
		if docStatusIsHistoricalLedger(rel) {
			continue
		}
		b, err := os.ReadFile(filepath.Join(repoRoot(t), filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("读取 %s: %v", rel, err)
		}
		for i, line := range strings.Split(string(b), "\n") {
			ms := docStatusRefRe.FindAllStringSubmatch(line, -1)
			if len(ms) == 0 {
				continue
			}
			total += len(ms)
			if !docStatusOpenRe.MatchString(line) {
				continue // 无开放标记 = 历史出处指针，允许指向空号
			}
			for _, m := range ms {
				openRefs++
				n, err := strconv.Atoi(m[1])
				if err != nil {
					t.Fatalf("%s:%d 引用编号解析失败 %q: %v", rel, i+1, m[1], err)
				}
				if !live[n] {
					t.Errorf("%s:%d 引用 `§三 #%d` 并标为开放（⬜/⏳/未排期/未接入），"+
						"但该编号已不在 ROADMAP §3.1/§3.2 表内（落地即按 §六 第 1 条移出转空号）——"+
						"开放声明必须指向现存条目：%s",
						rel, i+1, n, strings.TrimSpace(line))
				}
			}
		}
	}

	if total < docStatusMinRefs {
		t.Fatalf("扫描面塌缩：全仓只解析到 %d 处 `§三 #N` 引用（基线 ≥%d）——"+
			"正则或历史台账排除清单被改坏", total, docStatusMinRefs)
	}
	if openRefs < docStatusMinOpenRefs {
		t.Fatalf("扫描面塌缩：带开放标记的引用只有 %d 处（基线 ≥%d）——"+
			"开放标记集合或排除清单被改坏", openRefs, docStatusMinOpenRefs)
	}
	t.Logf("`§三 #N` 引用 %d 处，其中带开放标记 %d 处", total, openRefs)
}

// TestOperationsTraceRowTracksOTelSection 断言 2：OPERATIONS §4.3 的 trace 观测面行
// 不得与同文件 §3.4（OTel tracing 已实现）自相矛盾——「未接入」是漂移，且必须点名开关变量。
func TestOperationsTraceRowTracksOTelSection(t *testing.T) {
	t.Parallel()
	ops := readRepoFile(t, docStatusOps)

	if !strings.Contains(ops, "### 3.4 OTel tracing") {
		t.Fatalf("%s 未找到 %q 小节——若 OTel tracing 被移除，本断言前提失效，需同步本门禁",
			docStatusOps, "### 3.4 OTel tracing")
	}
	var row string
	for _, line := range strings.Split(ops, "\n") {
		if strings.HasPrefix(line, "| 跨服务 trace |") {
			row = line
			break
		}
	}
	if row == "" {
		t.Fatalf("%s 未找到「| 跨服务 trace |」表行——行被改名或移除，需同步本门禁", docStatusOps)
	}
	if strings.Contains(row, "未接入") {
		t.Errorf("%s 的跨服务 trace 行声称「未接入」，但同文件 §3.4 已实现 OTel tracing"+
			"（`S3C_OTEL_ENDPOINT` 开关、ADR-013）——同一文件自相矛盾：%s",
			docStatusOps, strings.TrimSpace(row))
	}
	if !strings.Contains(row, "S3C_OTEL_ENDPOINT") {
		t.Errorf("%s 的跨服务 trace 行未点名实现开关 `S3C_OTEL_ENDPOINT`——"+
			"观测面行必须与 §3.4 同一口径，否则运维照着旧行配不出 trace：%s",
			docStatusOps, strings.TrimSpace(row))
	}
}

// TestFeaturesOpenClaimsCarryTimeAnchor 断言 3：FEATURES（已完成台账）里现在时的
// 开放陈述必须带日期——章节头注描述的是**写作时点**，无锚点就会被读成现状。
func TestFeaturesOpenClaimsCarryTimeAnchor(t *testing.T) {
	t.Parallel()
	text := readRepoFile(t, docStatusFeatures)

	claims := 0
	for i, line := range strings.Split(text, "\n") {
		if !docStatusFeatureClaimRe.MatchString(line) {
			continue
		}
		claims++
		if !docStatusDateRe.MatchString(line) {
			t.Errorf("%s:%d 无限定时点的开放陈述——FEATURES 是**已完成**台账，"+
				"写作时点的状态必须带日期锚（如「2026-09-19 写作时点」）并给出闭环指针，"+
				"否则读者会把历史状态当成现状：%s",
				docStatusFeatures, i+1, strings.TrimSpace(line))
		}
	}
	if claims < docStatusMinFeatureClaims {
		t.Fatalf("扫描面塌缩：FEATURES 中「仍开放 / 仍未处置」只剩 %d 行（基线 ≥%d）——"+
			"措辞收敛后需同步本阈值，别让门禁静默失效",
			claims, docStatusMinFeatureClaims)
	}
	t.Logf("FEATURES 现在时开放陈述 %d 行（缺日期锚的行数见上方错误）", claims)
}

// docStatusIsHistoricalLedger 判断该 md 是否为**历史 / 时点台账**（不参与断言 1）：
// 归档冻结件、FEATURES 已完成台账、CHANGELOG 逐字发布历史——三者按「不追溯篡改 /
// 快照不回写」纪律保留原文，其中的 `§三 #N + ⏳` 是当时记录，不是现状声明。
func docStatusIsHistoricalLedger(rel string) bool {
	if strings.HasPrefix(rel, "docs/archive/") {
		return true
	}
	return rel == docStatusFeatures || rel == "CHANGELOG.md"
}

// TestDocPageDateClaimNotStale 断言 4：页头时点声明不得早于文内最新时点锚——
// 「正文改了、页头日期没动」的日期滞后（本批修过的四类正属此类）必须被机械拦住。
func TestDocPageDateClaimNotStale(t *testing.T) {
	t.Parallel()

	claimDocs, anchors := 0, 0
	for _, rel := range repoMarkdownFiles(t) {
		if strings.HasPrefix(rel, "docs/archive/") || rel == "CHANGELOG.md" {
			// 归档 = 冻结不回写；CHANGELOG 的历史条目里「最后更新」是**叙述**（记录当年把
			// 哪篇文档的页头推到哪天），不是它自己的页头声明——按历史台账排除。
			continue
		}
		text := readRepoFile(t, rel)
		claim := docStatusDateClaimRe.FindStringSubmatch(text)
		if claim == nil {
			continue
		}
		claimDocs++
		maxAnchor := ""
		for _, m := range docStatusDateAnchorRe.FindAllStringSubmatch(text, -1) {
			anchors++
			if m[1] > maxAnchor { // ISO 日期字典序即时间序
				maxAnchor = m[1]
			}
		}
		if maxAnchor > claim[1] {
			t.Errorf("%s 页头时点声明「%s」早于文内最新时点锚 %s——"+
				"正文已更新但页头日期未同步（改完正文必须把页头日期推到当天）",
				rel, claim[1], maxAnchor)
		}
	}

	if claimDocs < docStatusMinDateClaimDocs {
		t.Fatalf("扫描面塌缩：全仓只解析到 %d 篇带页头时点声明的文档（基线 ≥%d）——"+
			"声明措辞或扫描面被改坏", claimDocs, docStatusMinDateClaimDocs)
	}
	if anchors < docStatusMinDateAnchors {
		t.Fatalf("扫描面塌缩：全仓只解析到 %d 处文内时点锚（基线 ≥%d）——"+
			"锚短语集合被改坏", anchors, docStatusMinDateAnchors)
	}
	t.Logf("页头时点声明 %d 篇，文内时点锚 %d 处", claimDocs, anchors)
}
