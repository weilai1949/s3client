package main

// doc_alert_drift_gate_test.go —— 「文档告警表 ⇔ 可执行规则文件」双向等集门禁。
//
// 背景（2026-10-10 文档「明确没做」盘点，A3）：`docs/OPERATIONS.md` §4.2 是**人类可读**
// 的告警基线，`deploy/prometheus/s3client.rules.yml` 是**可执行形态**，两者靠人工同改。
// 实测表 13 行、规则 13 条**数字相同纯属巧合**——两边集合根本不一样：表里有
// 「健康探测失败」（外部黑盒探测，不属 Prometheus 规则），规则里有
// `S3ClientZipPartialFailures`（表里没有），即「抄漏一条 + 多写一条」的典型漂移，
// 而当时**没有任何门禁**看得见。既有的
// `TestPrometheusRulesReferenceRealMetrics` 只校验指标名与 `code` 取值是否真实，
// 不校验「表与规则是不是同一套」。
//
// 解析口径（零依赖、纯文本扫描，与 doc_ci_drift_gate_test.go 同源）：
//   - 文档侧：定位 `### 4.2 建议告警规则` 到下一个 `### ` 之间的表格数据行，
//     第一列必须含**恰好一个**反引号包裹的告警名（`S3ClientXxx`）；
//   - 规则侧：`s3client.rules.yml` 里全部 `- alert: <name>`；
//   - 断言：两个集合**双向相等**，且各自不低于基线（防止正则塌缩后静默全绿）。
//
// 相关：`repo_infra_gate_test.go` 的 `TestPrometheusRulesReferenceRealMetrics`
// （指标名 / `code` 取值）、`grafana_dashboard_gate_test.go`（仪表盘引用的指标与记录规则）。

import (
	"regexp"
	"strings"
	"testing"
)

const (
	// alertOpsDocRel 是告警表所在文档。
	alertOpsDocRel = "docs/OPERATIONS.md"
	// alertRulesRel 是可执行规则文件（Prometheus `rule_files` 直接挂载）。
	alertRulesRel = "deploy/prometheus/s3client.rules.yml"
	// alertSectionHeading 定位 §4.2 小节起点。
	alertSectionHeading = "### 4.2 建议告警规则"
)

// minAlertRows 是扫描面自检基线：表数据行与规则条数都不得低于它
// （2026-10-10 实测各 13 条）。低于基线即解析口径塌缩，直接红灯。
const minAlertRows = 13

// alertNameCellRe 匹配表格第一列里的反引号告警名。
var alertNameCellRe = regexp.MustCompile("`(S3Client[A-Za-z0-9]+)`")

// alertRuleNameRe 匹配规则文件里的 `- alert: <name>`。
var alertRuleNameRe = regexp.MustCompile(`(?m)^\s*-\s+alert:\s+(\S+)`)

// separatorCellRe 匹配 Markdown 表格分隔行首列（`---` / `:---:` 等）。
var separatorCellRe = regexp.MustCompile(`^:?-+:?$`)

// opsAlertTableRows 提取 §4.2 表格的数据行（跳过表头与分隔行），返回每行第一列文本。
func opsAlertTableRows(doc string) []string {
	start := strings.Index(doc, alertSectionHeading)
	if start < 0 {
		return nil
	}
	rest := doc[start+len(alertSectionHeading):]
	// 小节边界：下一个 `### `（三级标题）即本节结束。
	if next := strings.Index(rest, "\n### "); next >= 0 {
		rest = rest[:next]
	}
	var firstCells []string
	for _, line := range strings.Split(rest, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "|") {
			continue
		}
		cells := strings.Split(strings.TrimPrefix(line, "|"), "|")
		if len(cells) < 2 {
			continue
		}
		first := strings.TrimSpace(cells[0])
		// 分隔行（`---`）不是数据行，由调用方按内容跳过。
		if separatorCellRe.MatchString(first) {
			continue
		}
		firstCells = append(firstCells, first)
	}
	return firstCells
}

// TestOperationsAlertTableMatchesRulesFile 断言 §4.2 表行与 rules.yml 的 alert 集合**双向相等**。
func TestOperationsAlertTableMatchesRulesFile(t *testing.T) {
	t.Parallel()

	doc := readRepoFile(t, alertOpsDocRel)
	rows := opsAlertTableRows(doc)
	if len(rows) < minAlertRows+1 { // +1：含表头行
		t.Fatalf("§4.2 只解析到 %d 行（含表头，基线 ≥%d）——小节定位或表格结构变化需同步本门禁",
			len(rows), minAlertRows+1)
	}

	rules := readRepoFile(t, alertRulesRel)
	ruleAlerts := map[string]bool{}
	for _, m := range alertRuleNameRe.FindAllStringSubmatch(rules, -1) {
		ruleAlerts[m[1]] = true
	}
	if len(ruleAlerts) < minAlertRows {
		t.Fatalf("%s 只解析到 %d 条 alert（基线 ≥%d）——规则文件结构变化需同步本门禁",
			alertRulesRel, len(ruleAlerts), minAlertRows)
	}

	// ① 表行 → 规则：每一行都必须点名一个真实存在的 alert。
	docAlerts := map[string]bool{}
	for i, cell := range rows {
		names := alertNameCellRe.FindAllStringSubmatch(cell, -1)
		if len(names) == 0 {
			// 表头行允许没有告警名（第一列是「告警」）；数据行必须有。
			if i == 0 {
				if cell != "告警" {
					t.Errorf("%s §4.2 表头第一列是 %q，期望「告警」（解析口径需同步）", alertOpsDocRel, cell)
				}
				continue
			}
			t.Errorf("%s §4.2 行 %d（%q）没有反引号告警名——表行必须与 rules.yml 的 alert 一一对应",
				alertOpsDocRel, i+1, cell)
			continue
		}
		if len(names) > 1 {
			t.Errorf("%s §4.2 行 %d（%q）出现 %d 个告警名，应恰好 1 个",
				alertOpsDocRel, i+1, cell, len(names))
			continue
		}
		name := names[0][1]
		docAlerts[name] = true
		if !ruleAlerts[name] {
			t.Errorf("%s §4.2 点名 %s，但 %s 没有该 alert——文档写了永不触发的告警",
				alertOpsDocRel, name, alertRulesRel)
		}
	}

	// ② 规则 → 表行：每条 alert 都必须出现在 §4.2（否则运维在文档里看不到它）。
	for name := range ruleAlerts {
		if !docAlerts[name] {
			t.Errorf("%s 定义了 %s，但 %s §4.2 表里没有它——可执行规则缺文档说明",
				alertRulesRel, name, alertOpsDocRel)
		}
	}
}
