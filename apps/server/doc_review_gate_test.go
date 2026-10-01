package main

// doc_review_gate_test.go —— `docs/DEVELOPMENT.md` §4「文档登记表」复审**到期**门禁。
//
// 背景（2026-09-30，ROADMAP §三 3.2 #19③）：登记表的「复审周期」自认是**建议值、未在 CI 强制、
// 没有任何自动化提醒**——「多久不看就该有人看」没人看就等于没登记。本门禁把有明确月数周期的行
// 做机械到期检查：`最后复审 + 周期` 早于今天即红灯，逼出一次**真实复审**（复审后把该行日期更新为
// 复核当天，红灯即转绿）。
//
// 断言范围：
//   - 登记表每一行的「最后复审」必须含可解析日期（`YYYY-MM-DD`，`登记时基线（2026-09-29）` 同样可解析）；
//   - 周期含「N 个月」的行：`日期 + N 个月` 不得早于今天；
//   - 扫描面自检：表格行数与「带月数周期」行数低于阈值 → Fatal（防解析口径塌缩后全绿但失明）。
//
// 不断言：无数值周期的行（「每个版本发版前」「由门禁强制」「每次归档操作时同 PR」「不复审——冻结」
// 等无固定时点的口径）、复审**质量**（那是人做的，门禁只管「到没到点」）。
//
// 变异验证（复核步骤，2026-09-30 实跑）：把 `docs/DEVELOPMENT.md` §4 中
// `docs/DEVELOPMENT.md · CONFIGURATION.md` 行的「最后复审」改成 `2025-01-01` → 本门禁红灯点名
// 「+ 3 个月 = 2025-04-01，今天 2026-09-30」→ 还原后绿灯。
// 复核命令：`cd apps/server && go test . -run TestDocReviewRegisterDueDates -count=1`。
//
// 相关：`docs_naming_gate_test.go`（命名清单三处分叉）、`doc_index_gate_test.go`（导航覆盖）——
// 三者分别保证「名字不分叉」「有入口」「到点有人看」。

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	docReviewSurface = "docs/DEVELOPMENT.md"
	// docReviewStart / docReviewEnd 圈定 §4「文档登记表」的文本段（措辞变更需同步本门禁）。
	docReviewStart = "**文档登记表**"
	docReviewEnd   = "落地要求："
	// minReviewRows / minReviewMonthRows 是扫描面自检阈值
	//（实测 2026-09-30：登记表 21 行，其中 14 行带「N 个月」周期）。
	minReviewRows      = 15
	minReviewMonthRows = 8
)

var (
	reviewMonthsRe = regexp.MustCompile(`(\d+)\s*个月`)
	reviewDateRe   = regexp.MustCompile(`(\d{4})-(\d{2})-(\d{2})`)
)

// TestDocReviewRegisterDueDates 断言：文档登记表里带月数周期的行没有超过复审到期日。
func TestDocReviewRegisterDueDates(t *testing.T) {
	text := readRepoFile(t, docReviewSurface)
	seg := reviewTableSlice(t, text)

	now := time.Now()
	rows, monthRows, overdue := 0, 0, 0
	for _, line := range strings.Split(seg, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "|") ||
			strings.Contains(line, "| 文档 |") ||
			isReviewSeparator(line) {
			continue
		}
		cells := strings.Split(line, "|")
		if len(cells) < 5 {
			t.Errorf("登记表行解析失败（不足 3 列）：%q", line)
			continue
		}
		name := strings.TrimSpace(cells[1])
		cycle := strings.TrimSpace(cells[2])
		last := strings.TrimSpace(cells[3])
		rows++

		d := reviewDateRe.FindStringSubmatch(last)
		if d == nil {
			t.Errorf("%s：「最后复审」缺可解析日期（YYYY-MM-DD）：%q", name, last)
			continue
		}
		y, _ := strconv.Atoi(d[1])
		mo, _ := strconv.Atoi(d[2])
		day, _ := strconv.Atoi(d[3])
		lastAt := time.Date(y, time.Month(mo), day, 0, 0, 0, 0, time.UTC)

		mm := reviewMonthsRe.FindStringSubmatch(cycle)
		if mm == nil {
			continue // 无数值周期的口径不参与到期判断
		}
		monthRows++
		months, _ := strconv.Atoi(mm[1])
		due := lastAt.AddDate(0, months, 0)
		if now.After(due) {
			overdue++
			t.Errorf("%s：复审已过期——最后复审 %s + %d 个月 = %s，今天 %s；"+
				"请做一次真实复核后把该行日期更新为复核当天（登记表约定：只有真复核过才更新日期）",
				name, lastAt.Format("2006-01-02"), months,
				due.Format("2006-01-02"), now.Format("2006-01-02"))
		}
	}

	if rows < minReviewRows {
		t.Fatalf("扫描面塌缩：登记表解析出 %d 行，低于自检阈值 %d", rows, minReviewRows)
	}
	if monthRows < minReviewMonthRows {
		t.Fatalf("扫描面塌缩：带月数周期的行只有 %d 行，低于自检阈值 %d", monthRows, minReviewMonthRows)
	}
	t.Logf("登记表 %d 行，带月数周期 %d 行，过期 %d 行", rows, monthRows, overdue)
}

// reviewTableSlice 截取登记表文本段；标记缺失即 Fatal（措辞变更需同步本门禁）。
func reviewTableSlice(t *testing.T, text string) string {
	t.Helper()
	i := strings.Index(text, docReviewStart)
	if i < 0 {
		t.Fatalf("%s 未找到起始标记 %q：措辞变更需同步本门禁", docReviewSurface, docReviewStart)
	}
	rest := text[i+len(docReviewStart):]
	j := strings.Index(rest, docReviewEnd)
	if j < 0 {
		t.Fatalf("%s 未找到终止标记 %q：措辞变更需同步本门禁", docReviewSurface, docReviewEnd)
	}
	return rest[:j]
}

// isReviewSeparator 判断是否为 markdown 表格分隔行（`|---|` 一族）。
func isReviewSeparator(line string) bool {
	return strings.TrimLeft(strings.ReplaceAll(line, "|", ""), " -") == ""
}
