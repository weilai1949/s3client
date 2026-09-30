package main

// grafana_dashboard_gate_test.go —— `deploy/grafana/s3clinet.dashboard.json` 的源码门禁。
//
// 背景（2026-09-30）：OPERATIONS.md §4 此前明说「未提供 SLO 仪表盘」，SLI/SLO 只以表格与
// `deploy/prometheus/s3clinet.rules.yml` 的记录规则存在。补上仪表盘后，它与规则文件一样属于
// **可直接进生产** 的配置：面板表达式里的指标名 / `code` 取值 / 记录规则名都是契约，写错一个
// 字母 Grafana **不会报错**，只会让面板永远空白（或让 recording rule 永远 no data），
// 而全部 Go 门禁照样全绿。本文件把这几类契约变成红灯。
//
// 断言范围：
//   - JSON 可解析、顶层元信息与 templating 数据源变量形态正常（可被 Grafana 导入）；
//   - 面板引用的每个 `s3c_*` 指标都真实存在于**发射点**（引号紧邻 `s3c_` 的正则口径，
//     与 `TestPrometheusRulesReferenceRealMetrics` 相同——只在注释里出现的名字不算）；
//   - 每个 `code` 取值都在 `s3wrap` 的错误码白名单（或 errorClass 的非 API 分类）内；
//   - 面板引用的每个 `s3clinet:*` 记录规则都真实存在于 `deploy/prometheus/s3clinet.rules.yml`
//     ——保证「仪表盘与规则文件同源」不是口头约定。
//
// 刻意不做（盲区，避免后来者误判覆盖面）：
//   - 不校验面板布局 / 阈值 / 描述文案是否合理——自然语言与设计偏好，属人工审查；
//   - 不连 Grafana / Prometheus 实际导入或查询（需要外部服务，放进单测只会 flaky）；
//   - 不校验表达式语义（如聚合维度是否正确），只校验名字与标签取值这两类可机械提取的契约。
//
// 变异验证（复核步骤，2026-09-30 实测）：把 `s3c_store_up` 打成 `s3c_store_upp` →
// `TestGrafanaDashboardReferencesRealMetrics` 红灯并点名该指标 → 还原后绿灯。
//
// 相关：`repo_infra_gate_test.go` 的 `TestPrometheusRulesReferenceRealMetrics`（规则文件侧）、
// `docs/OPERATIONS.md` §4（人类可读的 SLO / 告警基线）。

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// grafanaDashboardRel 是仪表盘 JSON 相对仓库根的路径。
const grafanaDashboardRel = "deploy/grafana/s3clinet.dashboard.json"

// grafanaDatasourceVarRef 是面板数据源必须引用的 templating 变量（Grafana 约定形态）。
const grafanaDatasourceVarRef = "${DS_PROMETHEUS}"

// grafanaPanel 是仪表盘面板的可解析子集（只取门禁需要的字段）。
type grafanaPanel struct {
	Type       string          `json:"type"`
	Title      string          `json:"title"`
	Datasource *grafanaDatasrc `json:"datasource"`
	Targets    []grafanaTarget `json:"targets"`
	Options    map[string]any  `json:"options"`
	Panels     []grafanaPanel  `json:"panels"`
	Collapsed  bool            `json:"collapsed"`
}

// grafanaDatasrc 是 Grafana 面板 / 变量的数据源引用。
type grafanaDatasrc struct {
	Type string `json:"type"`
	UID  string `json:"uid"`
}

// grafanaTarget 是面板的一个查询目标（PromQL 在 expr 里）。
type grafanaTarget struct {
	Expr         string          `json:"expr"`
	LegendFormat string          `json:"legendFormat"`
	RefID        string          `json:"refId"`
	Datasource   *grafanaDatasrc `json:"datasource"`
}

// grafanaTemplatingVar 是 templating.list 的一项（只取门禁需要的字段）。
type grafanaTemplatingVar struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Query any    `json:"query"`
}

// grafanaDashboard 是仪表盘 JSON 的可解析子集。
type grafanaDashboard struct {
	Title         string                    `json:"title"`
	UID           string                    `json:"uid"`
	SchemaVersion int                       `json:"schemaVersion"`
	Refresh       string                    `json:"refresh"`
	Time          struct{ From, To string } `json:"time"`
	Tags          []string                  `json:"tags"`
	Templating    struct {
		List []grafanaTemplatingVar `json:"list"`
	} `json:"templating"`
	Panels []grafanaPanel `json:"panels"`
}

// loadGrafanaDashboard 读取并解析仪表盘 JSON（解析失败即 Fatal）。
func loadGrafanaDashboard(t *testing.T) grafanaDashboard {
	t.Helper()
	raw := readRepoFile(t, grafanaDashboardRel)
	var d grafanaDashboard
	if err := json.Unmarshal([]byte(raw), &d); err != nil {
		t.Fatalf("%s 不是合法 JSON（Grafana 无法导入）：%v", grafanaDashboardRel, err)
	}
	return d
}

// walkGrafanaPanels 递归收集全部面板（含 row 内嵌面板，防将来改成 collapsed row 后漏扫）。
func walkGrafanaPanels(panels []grafanaPanel, fn func(grafanaPanel)) {
	for _, p := range panels {
		fn(p)
		walkGrafanaPanels(p.Panels, fn)
	}
}

// grafanaPanelExprs 返回全部面板查询表达式（保持顺序，供稳定报错）。
func grafanaPanelExprs(d grafanaDashboard) []string {
	var out []string
	walkGrafanaPanels(d.Panels, func(p grafanaPanel) {
		for _, tg := range p.Targets {
			if s := strings.TrimSpace(tg.Expr); s != "" {
				out = append(out, s)
			}
		}
	})
	return out
}

// grafanaMetricRe 从 PromQL 表达式中提取 `s3c_*` 指标名（词法级）。
var grafanaMetricRe = regexp.MustCompile(`\bs3c_[a-z0-9_]+`)

// grafanaRecordingRuleRe 提取 `s3clinet:*` 记录规则名（rules.yml 的命名空间）。
var grafanaRecordingRuleRe = regexp.MustCompile(`\bs3clinet:[a-z0-9_:]+`)

// promRecordRuleRe 匹配 rules.yml 里的 `record: <name>`。
var promRecordRuleRe = regexp.MustCompile(`(?m)^[ \t]*-[ \t]*record:[ \t]*(\S+)[ \t]*$`)

// TestGrafanaDashboardParsesAndIsSane：JSON 必须可解析，且具备可被 Grafana 导入的
// 基本形态——顶层元信息、数据源模板变量、以及「每个查询面板都真的绑了数据源与表达式」。
func TestGrafanaDashboardParsesAndIsSane(t *testing.T) {
	d := loadGrafanaDashboard(t)

	if strings.TrimSpace(d.Title) == "" {
		t.Errorf("%s 缺少顶层 title（Grafana 导入后无法识别仪表盘名）", grafanaDashboardRel)
	}
	if strings.TrimSpace(d.UID) == "" {
		t.Errorf("%s 缺少顶层 uid（导入会随机生成，重导入会重复建盘）", grafanaDashboardRel)
	}
	if d.SchemaVersion < 36 {
		t.Errorf("%s 的 schemaVersion=%d 过低（本门禁按 Grafana 10+ 的 JSON 形态校验，建议 ≥ 36）",
			grafanaDashboardRel, d.SchemaVersion)
	}
	if d.Time.From == "" || d.Time.To == "" {
		t.Errorf("%s 缺少 time.from / time.to（导入后默认时间范围不可控）", grafanaDashboardRel)
	}
	if strings.TrimSpace(d.Refresh) == "" {
		t.Errorf("%s 缺少 refresh（单实例 + 无历史存储，建议 30s–1m 轮询）", grafanaDashboardRel)
	}

	// ① templating：必须有一个 Prometheus 类型的数据源变量，供面板引用。
	var dsVar *grafanaTemplatingVar
	for i := range d.Templating.List {
		v := &d.Templating.List[i]
		if v.Name == "DS_PROMETHEUS" {
			dsVar = v
		}
	}
	if dsVar == nil {
		t.Fatalf("%s templating.list 缺少名为 DS_PROMETHEUS 的数据源变量（导入时无法选择数据源）",
			grafanaDashboardRel)
	}
	if dsVar.Type != "datasource" {
		t.Errorf("%s 的 DS_PROMETHEUS 变量 type=%q，应为 datasource", grafanaDashboardRel, dsVar.Type)
	}
	if q, _ := dsVar.Query.(string); q != "prometheus" {
		t.Errorf("%s 的 DS_PROMETHEUS 变量 query=%v，应为 \"prometheus\"", grafanaDashboardRel, dsVar.Query)
	}

	// ② 面板形态：每个面板有标题；查询面板绑到数据源变量且有非空 expr；text 面板有内容。
	panels, queryPanels, textPanels, rows := 0, 0, 0, 0
	walkGrafanaPanels(d.Panels, func(p grafanaPanel) {
		panels++
		if strings.TrimSpace(p.Title) == "" {
			t.Errorf("%s 存在无标题面板（type=%q）——无法在告警上下文里定位", grafanaDashboardRel, p.Type)
		}
		if p.Type == "row" {
			rows++
			return
		}
		if p.Datasource == nil || p.Datasource.UID != grafanaDatasourceVarRef || p.Datasource.Type != "prometheus" {
			t.Errorf("%s 面板 %q 的 datasource=%+v，应统一绑定 prometheus 的 %s",
				grafanaDashboardRel, p.Title, p.Datasource, grafanaDatasourceVarRef)
		}
		if p.Type == "text" {
			textPanels++
			if content, _ := p.Options["content"].(string); strings.TrimSpace(content) == "" {
				t.Errorf("%s 文本面板 %q 的 options.content 为空——面板会渲染成空白",
					grafanaDashboardRel, p.Title)
			}
			return
		}
		queryPanels++
		hasExpr := false
		for _, tg := range p.Targets {
			if strings.TrimSpace(tg.Expr) != "" {
				hasExpr = true
			}
		}
		if !hasExpr {
			t.Errorf("%s 面板 %q（type=%q）没有任何非空 expr——导入后是无数据的空面板",
				grafanaDashboardRel, p.Title, p.Type)
		}
	})

	// 自检阈值：数量塌缩说明解析口径失效（如面板被挪进 collapsed row 而 walk 没跟），必须红灯而非安静通过。
	if panels < 20 {
		t.Fatalf("%s 只解析出 %d 个面板（阈值 20）：结构或解析口径已塌缩", grafanaDashboardRel, panels)
	}
	if queryPanels < 10 {
		t.Fatalf("%s 只有 %d 个查询面板（阈值 10）：仪表盘退化成说明文档", grafanaDashboardRel, queryPanels)
	}
	if textPanels == 0 {
		t.Errorf("%s 没有任何 text 面板——sqlite 探测口径 / 观测缺口这类无法用 PromQL 表达的事实无处承载",
			grafanaDashboardRel)
	}
	if rows == 0 {
		t.Errorf("%s 没有任何 row 分组——面板超过 20 个时应按主题分组", grafanaDashboardRel)
	}
	t.Logf("%s：%d 个面板（查询 %d / 文本 %d / 分组 %d）", grafanaDashboardRel, panels, queryPanels, textPanels, rows)
}

// TestGrafanaDashboardReferencesRealMetrics：面板引用的每个 `s3c_*` 指标都必须在
// `internal/handler/metrics.go` 或 `main.go` 的发射点真实输出（引号紧邻 `s3c_` 的口径，
// 与 `TestPrometheusRulesReferenceRealMetrics` 相同）。
//
// 为什么需要它：PromQL 里的指标名是契约。写错一个字母 Prometheus / Grafana 都不报错，
// 面板只是永远 no data 或恒为 0——对「存储掉线」这类硬失败项，等于监控静默失明。
func TestGrafanaDashboardReferencesRealMetrics(t *testing.T) {
	emitted := emittedMetricNames(t)

	d := loadGrafanaDashboard(t)
	referenced := map[string]bool{}
	for _, expr := range grafanaPanelExprs(d) {
		for _, m := range grafanaMetricRe.FindAllString(expr, -1) {
			referenced[m] = true
		}
	}
	if len(referenced) < 10 {
		t.Fatalf("%s 只解析出 %d 个 `s3c_*` 指标引用（阈值 10）：表达式解析口径已失效",
			grafanaDashboardRel, len(referenced))
	}

	var missing []string
	for name := range referenced {
		if !emitted[name] {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("%s 引用了未发射的指标（Grafana 不报错，面板只是永远无数据）：%v\n实际发射的指标：%v",
			grafanaDashboardRel, missing, sortedKeys(emitted))
	}
	t.Logf("%s 引用的 %d 个 s3c_* 指标全部存在于发射点", grafanaDashboardRel, len(referenced))
}

// TestGrafanaDashboardCodesAreInS3wrapWhitelist：面板里 `code=...` / `code=~"A|B"` 的每个
// 取值都必须在 `s3wrap` 的错误码白名单（或 errorClass 的非 API 分类）内。
//
// 为什么需要它：白名单外的码会被 `errorClass` 折叠成 `other`，于是 `code="Foo"` 永不命中、
// 面板恒为 0；而白名单是会变的（本仓库就为「标签基数无界」改过它）。
func TestGrafanaDashboardCodesAreInS3wrapWhitelist(t *testing.T) {
	allowed := s3wrapMetricErrorCodes(t)

	d := loadGrafanaDashboard(t)
	checked := 0
	var bad []string
	for _, expr := range grafanaPanelExprs(d) {
		for _, m := range promCodeValueRe.FindAllStringSubmatch(expr, -1) {
			for _, code := range strings.Split(m[1], "|") {
				code = strings.TrimSpace(code)
				if code == "" {
					continue
				}
				checked++
				if !allowed[code] {
					bad = append(bad, code)
				}
			}
		}
	}
	if checked < 3 {
		t.Fatalf("%s 只解析出 %d 个 code 取值（阈值 3）：解析口径已失效", grafanaDashboardRel, checked)
	}
	if len(bad) > 0 {
		sort.Strings(bad)
		t.Errorf("%s 的 `code` 取值不在白名单内（会被折叠成 other，面板永不命中）：%v",
			grafanaDashboardRel, bad)
	}
	t.Logf("%s 的 %d 个 code 取值均在白名单 / errorClass 分类内", grafanaDashboardRel, checked)
}

// TestGrafanaDashboardUsesRecordingRulesFromPrometheusRules：面板引用的每个 `s3clinet:*`
// 记录规则都必须在 `deploy/prometheus/s3clinet.rules.yml` 里真实定义，且三条 SLI 记录规则
// 都被仪表盘引用。
//
// 为什么需要它：仪表盘与本仓库分发的记录规则是同一份 SLI 的两个消费面（OPERATIONS.md §4）。
// 规则改名而仪表盘不改 → 面板静默无数据；仪表盘少引一条 → 该 SLI 在盘上没有视图。
func TestGrafanaDashboardUsesRecordingRulesFromPrometheusRules(t *testing.T) {
	const rulesRel = "deploy/prometheus/s3clinet.rules.yml"
	rules := readRepoFile(t, rulesRel)
	defined := map[string]bool{}
	for _, m := range promRecordRuleRe.FindAllStringSubmatch(rules, -1) {
		defined[m[1]] = true
	}
	if len(defined) == 0 {
		t.Fatalf("%s 未解析出任何 record 规则（解析口径需同步）", rulesRel)
	}

	d := loadGrafanaDashboard(t)
	referenced := map[string]bool{}
	for _, expr := range grafanaPanelExprs(d) {
		for _, m := range grafanaRecordingRuleRe.FindAllString(expr, -1) {
			referenced[m] = true
		}
	}
	var missing []string
	for name := range referenced {
		if !defined[name] {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("%s 引用了 %s 里不存在的记录规则（面板会静默无数据）：%v",
			grafanaDashboardRel, rulesRel, missing)
	}

	// 三条 SLI 记录规则都必须有面板，否则该 SLI 在仪表盘上不可见。
	for name := range defined {
		if !referenced[name] {
			t.Errorf("%s 未引用记录规则 %q：该 SLI 在仪表盘上没有视图（OPERATIONS.md §4.1）",
				grafanaDashboardRel, name)
		}
	}
}

// emittedMetricNames 收集生产代码里真实发射的 `s3c_*` 指标名（引号紧邻 `s3c_` 的口径，
// 注释 / HELP 文本里提到的名字不算）——真值取自源码，不是再抄一份清单。
func emittedMetricNames(t *testing.T) map[string]bool {
	t.Helper()
	emitted := map[string]bool{}
	for _, rel := range []string{
		filepath.Join("apps", "server", "internal", "handler", "metrics.go"),
		filepath.Join("apps", "server", "main.go"),
	} {
		for _, m := range emittedMetricRe.FindAllString(readRepoFile(t, rel), -1) {
			emitted[strings.TrimPrefix(m, `"`)] = true
		}
	}
	if len(emitted) < 10 {
		t.Fatalf("只从发射点抽到 %d 个指标（解析口径需同步）", len(emitted))
	}
	return emitted
}

// s3wrapMetricErrorCodes 解析 `s3wrap/metrics.go` 的白名单 + errorClass 的非 API 分类。
func s3wrapMetricErrorCodes(t *testing.T) map[string]bool {
	t.Helper()
	allowed := map[string]bool{"other": true, "canceled": true, "timeout": true, "transport": true}
	sw := readRepoFile(t, filepath.Join("apps", "server", "internal", "s3wrap", "metrics.go"))
	block := regexp.MustCompile(`(?s)var metricErrorCodes = map\[string\]struct\{\}\{(.*?)\n\}`).FindStringSubmatch(sw)
	if block == nil {
		t.Fatal("未能在 s3wrap/metrics.go 定位 metricErrorCodes 白名单（解析口径需同步）")
	}
	for _, m := range regexp.MustCompile(`"([A-Za-z0-9]+)":`).FindAllStringSubmatch(block[1], -1) {
		allowed[m[1]] = true
	}
	if len(allowed) < 10 {
		t.Fatalf("白名单只解析出 %d 项（解析口径需同步）", len(allowed))
	}
	return allowed
}

// TestGrafanaDashboardFileIsImportableJSON 保留一条最直白的验收断言：文件存在且是 JSON 对象。
// （与上面几条共用 loadGrafanaDashboard；单独存在是为了让「文件被误删」的报错最直观。）
func TestGrafanaDashboardFileIsImportableJSON(t *testing.T) {
	path := filepath.Join(repoRoot(t), filepath.FromSlash(grafanaDashboardRel))
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取 %s：%v（Grafana 导入的文件必须随仓库分发）", grafanaDashboardRel, err)
	}
	var obj map[string]any
	if err := json.Unmarshal(b, &obj); err != nil {
		t.Fatalf("%s 不是合法 JSON 对象：%v", grafanaDashboardRel, err)
	}
	if _, ok := obj["panels"]; !ok {
		t.Errorf("%s 缺少顶层 panels 字段——不是 Grafana 仪表盘 JSON", grafanaDashboardRel)
	}
}
