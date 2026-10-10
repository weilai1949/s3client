package main

// agent_evals_gate_test.go —— 第 11 层「AI 效果证据」资产的结构不变量。
//
// 背景（2026-09-30）：AI 治理此前只有**过程约束**——`docs/AI_POLICY.md` 的权限矩阵 / 披露 / DoD
// 管的是「谁能做什么、必须声明什么」，但同一份 DoD 可以全绿而输出把契约改坏，仓库没有任何
// **可复现的效果测量**。2026-09-30 新增三件套：`docs/AGENT_EVALS.md`（黄金任务集 + 加权评分卡 +
// 贡献度量口径）、`scripts/agent-eval.sh`（机械评测）、PR 披露模板的「占比」字段。这类资产与
// AI_POLICY 一样「最容易被静默破坏」：GT 表被删空、评测脚本退化成空壳、政策与评测文档脱钩、
// 披露字段集漂移——全部既有门禁照常全绿。
//
// 断言范围（只钉**结构**，不评判评测结果好坏 / 任务选题是否合理——那是人工评审）：
//   - `docs/AGENT_EVALS.md` 存在、非空壳，含「黄金任务集 / 评分卡 / 度量」三节标记，
//     且黄金任务集表格的 `GT-<n>` 行 ≥ 阈值；
//   - `scripts/agent-eval.sh` 存在、可执行、非空壳（含 `set -euo pipefail`、`go vet`、
//     `go test`、`EVAL_RESULT` 关键词——脚本被掏空等于机械评测失效）；
//   - `scripts/evals/golden-tasks.yaml` 存在、可解析、恰好 GT-1..GT-4，且 id/标题与文档总览表
//     逐字一致、每任务 `verification_commands` 非空（黄金任务从散文升级为机器可执行规格）；
//   - `scripts/evals/run-golden-task.sh` 存在、可执行、非空壳（按 GT id 实跑判据命令，
//     未知 id fail closed）；
//   - `docs/AI_POLICY.md` 正文**回引** `AGENT_EVALS.md`（过程约束与效果证据两文不得脱钩）；
//   - `.github/PULL_REQUEST_TEMPLATE.md` 的「AI 使用披露」块含「占比」字段（贡献度量的披露载体），
//     且 AI_POLICY §5 的披露模板与 PR 模板**字段集一致**（防「政策说一套、模板是另一套」）。
// 自检纪律（同 doc_number_gate 等）：文档缺失 / GT 表解析出 0 行 / 脚本缺失 → Fatal，
// 不允许「全绿但失明」。
//
// 变异验证（2026-09-30 实测，均红灯点名后还原全绿）：
//   - 从 agent-eval.sh 删除 `go vet` 行 → TestAgentEvalsScriptIsNotAStub 点名「缺少关键词 go vet」；
//   - 删除 AGENT_EVALS.md 的 GT-3 / GT-4 两行（4 → 2 条）→ TestAgentEvalsDocHasGoldenTaskSet
//     点名「只剩 2 条（阈值 3）」；
//   - 把 AI_POLICY.md 中的 `AGENT_EVALS.md` 全部替换为 `AGENT_EVALS_TMP.md` →
//     TestAgentEvalsPolicyReferencesDoc 点名「未引用 AGENT_EVALS.md」；
//   - 从 PR 模板披露块删除「- 占比：」行 → TestPRTemplateDisclosureCarriesShareField 点名缺字段；
//   - 从 scripts/evals/golden-tasks.yaml 删除 GT-4 任务 → TestAgentEvalsGoldenTaskSetIsMachineReadable
//     点名「有 3 个任务，期望恰好 4 个」（还原后全绿）；
//   - 把 golden-tasks.yaml 的 GT-2 标题改成与文档不一致 → 同测试点名「与总览表不一致」。

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// gtRowRe 匹配黄金任务集表格里的任务行（`| GT-1 | … |`）。只认 `GT-<数字>` 前缀的表格行，
// 避免把正文里提到的 `GT-1` 当成任务登记。
var gtRowRe = regexp.MustCompile(`(?m)^\|\s*GT-\d+\s*\|`)

// disclosureFieldLabels 是披露块的 7 个字段标签（PR 模板与 AI_POLICY §5 模板必须一致）。
var disclosureFieldLabels = []string{
	"- 工具：",
	"- 用途：",
	"- 范围：",
	"- 占比：",
	"- 人工审查：",
	"- 敏感数据：",
	"- 许可证：",
}

// minGoldenTasks 是黄金任务集的自检阈值：低于它说明任务集被整体删空或解析口径塌缩。
const minGoldenTasks = 3

// agentEvalsDocPath / agentEvalScriptRel 是三件套的固定路径。
const (
	agentEvalsDocPath = "docs/AGENT_EVALS.md"
	agentEvalScript   = "scripts/agent-eval.sh"
)

// agentEvalsTaskSetPath / agentEvalsRunnerPath 是「机器可读黄金任务集 + 执行器」的固定路径。
const (
	agentEvalsTaskSetPath = "scripts/evals/golden-tasks.yaml"
	agentEvalsRunnerPath  = "scripts/evals/run-golden-task.sh"
)

// minGoldenTaskSetTasks / minGoldenTaskSetCommands / minGoldenTaskSetAnchors 是任务集的
// 「扫描面自检」阈值：解析出的任务数 / 判据命令总数 / 五维锚点总数低于阈值，说明解析口径
// 塌缩或任务集被掏空——Fatal 而非静默全绿（同 doc_number_gate 的自检纪律）。
const (
	minGoldenTaskSetTasks    = 4
	minGoldenTaskSetCommands = 10
	minGoldenTaskSetAnchors  = 20
)

// agentEvalScriptKeywords 是评测脚本不得缺失的关键词：脚本变空壳 = 机械评测失效。
var agentEvalScriptKeywords = []string{
	"set -euo pipefail",
	"go vet",
	"go test",
	"EVAL_RESULT",
}

// TestAgentEvalsDocHasGoldenTaskSet 断言 AGENT_EVALS.md 存在、含三大节标记、GT 集 ≥ 阈值。
func TestAgentEvalsDocHasGoldenTaskSet(t *testing.T) {
	path := filepath.Join(repoRoot(t), filepath.FromSlash(agentEvalsDocPath))
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取 %s: %v——第 11 层「AI 效果证据」的载体缺失，删除它等于把评测框架整个移除", path, err)
	}
	text := string(b)
	if len(b) < 2000 {
		t.Fatalf("%s 只有 %d 字节（下限 2000）：文档被截断或只剩空壳（口径需同步）", path, len(b))
	}
	for _, marker := range []string{"黄金任务集", "评分卡", "度量"} {
		if !strings.Contains(text, marker) {
			t.Errorf("%s 缺少节标记 %q——评测框架的三大件（任务集 / 评分卡 / 度量）缺一不可", path, marker)
		}
	}
	rows := gtRowRe.FindAllString(text, -1)
	if len(rows) == 0 {
		t.Fatal("AGENT_EVALS.md 未解析出任何 GT-<n> 表格行（解析口径需同步：任务行形如 `| GT-1 | … |`）")
	}
	if len(rows) < minGoldenTasks {
		t.Errorf("AGENT_EVALS.md 的黄金任务集只剩 %d 条（阈值 %d）：任务集被整体删空后，"+
			"「AI 是否值得信任」失去可复现的测量基准", len(rows), minGoldenTasks)
	}
}

// TestAgentEvalsScriptIsNotAStub 断言 scripts/agent-eval.sh 存在、可执行且不是空壳。
func TestAgentEvalsScriptIsNotAStub(t *testing.T) {
	path := filepath.Join(repoRoot(t), filepath.FromSlash(agentEvalScript))
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("读取 %s: %v——机械评测脚本缺失，「效果证据」只能靠人工跑", path, err)
	}
	if info.Mode()&0111 == 0 {
		t.Errorf("%s 不可执行（mode %v）：评测脚本必须能直接运行", path, info.Mode())
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取 %s: %v", path, err)
	}
	if len(b) < 500 {
		t.Fatalf("%s 只有 %d 字节（下限 500）：脚本疑似空壳（口径需同步）", path, len(b))
	}
	for _, kw := range agentEvalScriptKeywords {
		if !strings.Contains(string(b), kw) {
			t.Errorf("%s 缺少关键词 %q——脚本被掏空等于机械评测失效", path, kw)
		}
	}
}

// docGTRowRe 从 AGENT_EVALS.md 的黄金任务总览表解析 `| GT-1 | 标题 | … |` 的 id 与标题，
// 用于钉住「文档总览表 ↔ 机器可读任务集」的 id/标题一致（两处漂移 = 出现两个口径）。
var docGTRowRe = regexp.MustCompile(`(?m)^\|\s*(GT-\d+)\s*\|\s*([^|]*?)\s*\|`)

// goldenTaskSetDoc 是 scripts/evals/golden-tasks.yaml 的结构镜像。该文件刻意用「JSON 内容 +
// .yaml 后缀」：JSON 是 YAML 1.2 的严格子集，PyYAML 与 Go 标准库 encoding/json 都能解析，
// 这样门禁不必为评测给 go.mod 增加一个 YAML 模块依赖（go.mod / go.sum 不在本改动范围）。
type goldenTaskSetDoc struct {
	SchemaVersion       int                   `json:"schema_version"`
	Doc                 string                `json:"doc"`
	ScorecardDimensions []goldenTaskDimension `json:"scorecard_dimensions"`
	Tasks               []goldenTaskSpec      `json:"tasks"`
}

// goldenTaskDimension 是 §三 评分卡的一个维度（权重合计必须为 100）。
type goldenTaskDimension struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	WeightPct int    `json:"weight_pct"`
}

// goldenTaskSpec 是一道黄金任务的完整机器可执行规格。
type goldenTaskSpec struct {
	ID                   string              `json:"id"`
	Title                string              `json:"title"`
	Scope                string              `json:"scope"`
	Prompt               string              `json:"prompt"`
	FilesLikelyTouched   []string            `json:"files_likely_touched"`
	VerificationCommands []goldenTaskCommand `json:"verification_commands"`
	ExpectedArtifacts    []string            `json:"expected_artifacts"`
	ScoringAnchors       map[string]string   `json:"scoring_anchors"`
}

// goldenTaskCommand 是一条判据命令；RequiresPath 非空时表示「该路径不存在则记为 skip（partial）」。
type goldenTaskCommand struct {
	Cmd          string `json:"cmd"`
	Desc         string `json:"desc"`
	RequiresPath string `json:"requires_path"`
}

// TestAgentEvalsGoldenTaskSetIsMachineReadable 断言机器可读黄金任务集存在、可解析、恰好
// GT-1..GT-4，且 id/标题与文档总览表逐字一致、每任务判据命令与五维锚点非空。
// 它把「黄金任务只有散文」这一最高优先缺口钉死：删任务 / 改标题不同步 / 掏空判据 → 红灯。
func TestAgentEvalsGoldenTaskSetIsMachineReadable(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, filepath.FromSlash(agentEvalsTaskSetPath))
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取 %s: %v——黄金任务集只有散文、没有机器可执行规格，评测无法复现", path, err)
	}
	if len(raw) < 1500 {
		t.Fatalf("%s 只有 %d 字节（下限 1500）：任务集被截断或只剩空壳", path, len(raw))
	}
	var set goldenTaskSetDoc
	if err := json.Unmarshal(raw, &set); err != nil {
		t.Fatalf("%s 不是合法 JSON（该文件刻意用 JSON 内容 + .yaml 后缀，见文件 _format 字段）：%v", path, err)
	}
	if set.SchemaVersion != 1 {
		t.Errorf("%s schema_version = %d，期望 1（口径变更必须显式升版并同步门禁）", path, set.SchemaVersion)
	}
	if set.Doc != agentEvalsDocPath {
		t.Errorf("%s 的 doc = %q，期望 %q（机器可读集必须指向其 SSOT 文档）", path, set.Doc, agentEvalsDocPath)
	}

	// 评分卡五维：与 §三 的五个维度一一对应，权重合计 100，标签必须能在文档里找到。
	if len(set.ScorecardDimensions) != 5 {
		t.Fatalf("%s scorecard_dimensions = %d 条，期望 5（§三 五维评分卡）", path, len(set.ScorecardDimensions))
	}
	docText := readRepoFile(t, agentEvalsDocPath)
	dimIDs := map[string]bool{}
	weightSum := 0
	for _, d := range set.ScorecardDimensions {
		if d.ID == "" || d.Label == "" || d.WeightPct <= 0 {
			t.Errorf("%s 维度 %+v 不完整：id / label / weight_pct 均必填", path, d)
		}
		if dimIDs[d.ID] {
			t.Errorf("%s 维度 id %q 重复", path, d.ID)
		}
		dimIDs[d.ID] = true
		weightSum += d.WeightPct
		if !strings.Contains(docText, d.Label) {
			t.Errorf("%s 维度标签 %q 在 %s §三 中找不到——机器可读集与评分卡口径漂移", path, d.Label, agentEvalsDocPath)
		}
	}
	if weightSum != 100 {
		t.Errorf("%s 五维权重合计 = %d，期望 100（§三 评分卡口径）", path, weightSum)
	}

	// 恰好 GT-1..GT-4，且标题与文档总览表逐字一致。
	byID := map[string]goldenTaskSpec{}
	for _, task := range set.Tasks {
		if byID[task.ID].ID != "" {
			t.Errorf("%s 任务 id %q 重复", path, task.ID)
		}
		byID[task.ID] = task
	}
	if len(set.Tasks) != minGoldenTaskSetTasks {
		t.Fatalf("%s 有 %d 个任务，期望恰好 %d 个（GT-1..GT-4）", path, len(set.Tasks), minGoldenTaskSetTasks)
	}
	docTitles := map[string]string{}
	for _, m := range docGTRowRe.FindAllStringSubmatch(docText, -1) {
		docTitles[m[1]] = strings.TrimSpace(m[2])
	}
	cmdTotal, anchorTotal := 0, 0
	for _, id := range []string{"GT-1", "GT-2", "GT-3", "GT-4"} {
		task, ok := byID[id]
		if !ok {
			t.Errorf("%s 缺少任务 %s", path, id)
			continue
		}
		if wantTitle, found := docTitles[id]; !found {
			t.Errorf("%s 的总览表未解析出 %s 的标题行（解析口径需同步）", agentEvalsDocPath, id)
		} else if task.Title != wantTitle {
			t.Errorf("%s 的 %s 标题 = %q，与 %s 总览表的 %q 不一致——两处必须逐字同改",
				path, id, task.Title, agentEvalsDocPath, wantTitle)
		}
		if task.Scope == "" || task.Prompt == "" {
			t.Errorf("%s 的 %s 缺 scope / prompt——任务不可复现", path, id)
		}
		if len(task.FilesLikelyTouched) == 0 {
			t.Errorf("%s 的 %s 缺 files_likely_touched", path, id)
		}
		if len(task.ExpectedArtifacts) == 0 {
			t.Errorf("%s 的 %s 缺 expected_artifacts", path, id)
		}
		if len(task.VerificationCommands) == 0 {
			t.Errorf("%s 的 %s 没有 verification_commands——判据缺失的任务无法评测", path, id)
		}
		for _, c := range task.VerificationCommands {
			if strings.TrimSpace(c.Cmd) == "" {
				t.Errorf("%s 的 %s 有空的判据命令", path, id)
			}
			if strings.TrimSpace(c.Desc) == "" {
				t.Errorf("%s 的 %s 判据命令 %q 缺 desc（人看不懂的命令等于不可复核）", path, id, c.Cmd)
			}
			if c.RequiresPath != "" {
				if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(c.RequiresPath))); err != nil {
					t.Errorf("%s 的 %s 判据前置 %q 不存在：%v（写错前置会让判据静默 skip）", path, id, c.RequiresPath, err)
				}
			}
			cmdTotal++
		}
		for dim := range dimIDs {
			if strings.TrimSpace(task.ScoringAnchors[dim]) == "" {
				t.Errorf("%s 的 %s 缺维度 %q 的 scoring_anchors 锚点——五维必须逐任务落到可复核证据", path, id, dim)
			}
		}
		for dim := range task.ScoringAnchors {
			if !dimIDs[dim] {
				t.Errorf("%s 的 %s 出现未知维度 %q（不在 §三 五维内）", path, id, dim)
			}
		}
		anchorTotal += len(task.ScoringAnchors)
	}

	// 扫描面自检：解析量低于阈值 = 解析口径塌缩或任务集被掏空，Fatal 而非静默全绿。
	if cmdTotal < minGoldenTaskSetCommands {
		t.Fatalf("%s 只解析出 %d 条判据命令（阈值 %d）：任务集疑似被掏空或解析塌缩", path, cmdTotal, minGoldenTaskSetCommands)
	}
	if anchorTotal < minGoldenTaskSetAnchors {
		t.Fatalf("%s 只解析出 %d 条五维锚点（阈值 %d）：评分锚点被删空", path, anchorTotal, minGoldenTaskSetAnchors)
	}
}

// TestAgentEvalsGoldenTaskRunnerIsNotAStub 断言 GT 执行器存在、可执行、非空壳，且真的接到任务集。
func TestAgentEvalsGoldenTaskRunnerIsNotAStub(t *testing.T) {
	path := filepath.Join(repoRoot(t), filepath.FromSlash(agentEvalsRunnerPath))
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("读取 %s: %v——机器可读任务集没有执行器，等于又回到「只能人工跑」", path, err)
	}
	if info.Mode()&0111 == 0 {
		t.Errorf("%s 不可执行（mode %v）：GT 执行器必须能直接运行", path, info.Mode())
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取 %s: %v", path, err)
	}
	if len(b) < 1500 {
		t.Fatalf("%s 只有 %d 字节（下限 1500）：执行器疑似空壳", path, len(b))
	}
	for _, kw := range []string{"set -euo pipefail", "golden-tasks.yaml", "EVAL_RESULT", "exit 1"} {
		if !strings.Contains(string(b), kw) {
			t.Errorf("%s 缺少关键词 %q——执行器被掏空等于 GT 判据失效", path, kw)
		}
	}
}

// TestAgentEvalsPolicyReferencesDoc 断言 AI_POLICY.md 回引 AGENT_EVALS.md：
// 过程约束（政策）与效果证据（评测）互为闭环，脱钩即两文各说各话。
func TestAgentEvalsPolicyReferencesDoc(t *testing.T) {
	policy := readRepoFile(t, filepath.Join("docs", "AI_POLICY.md"))
	if !strings.Contains(policy, "AGENT_EVALS.md") {
		t.Error("docs/AI_POLICY.md 未引用 AGENT_EVALS.md——「披露 → 度量 → 评测」闭环断裂（两文必须互链）")
	}
}

// TestPRTemplateDisclosureCarriesShareField 断言 PR 披露块含「占比」字段——它是 AI 贡献度量
// （AGENT_EVALS.md §四）的披露载体，块在而字段被删等于度量口径悄悄失效。
func TestPRTemplateDisclosureCarriesShareField(t *testing.T) {
	text := readRepoFile(t, filepath.Join(".github", "PULL_REQUEST_TEMPLATE.md"))
	if !strings.Contains(text, "AI 使用披露") {
		t.Error("PR 模板缺少「AI 使用披露」块（AI_POLICY §5 的披露要求失去载体）")
	}
	for _, label := range disclosureFieldLabels {
		if !strings.Contains(text, label) {
			t.Errorf("PR 模板披露块缺少字段 %q——披露字段集是 AI 贡献度量的口径，删字段即删度量", label)
		}
	}
}

// TestAiPolicyDisclosureTemplateMatchesPRTemplate 断言 AI_POLICY §5 的披露模板与 PR 模板
// 字段集一致：政策正文是 SSOT，PR 模板是载体，两处漂移会让「披露了什么」出现两个口径。
func TestAiPolicyDisclosureTemplateMatchesPRTemplate(t *testing.T) {
	policy := readRepoFile(t, filepath.Join("docs", "AI_POLICY.md"))
	for _, label := range disclosureFieldLabels {
		if !strings.Contains(policy, label) {
			t.Errorf("AI_POLICY.md §5 的披露模板缺少字段 %q——与 PR 模板字段集不一致（两处必须同改）", label)
		}
	}
}

// ---- 度量台账的流程闭环（2026-10-10 交接快照 §5 未做第 4 项）----
//
// 背景：`AGENT_EVALS.md` §四 的度量表自 2026-09-30 建立、2026-10-10 仍为「0 条已回填」，文档自己也
// 承认「回填动作当前**无门禁强制**」。本组三条断言把「约定」变成红灯，覆盖三个可机械化的面：
//   - 表行数 ⇄ 状态行「N 条已回填」（回填了一行却忘改状态行 → 漂移立即暴露）；
//   - PR 模板的「AI 度量」勾选（披露 → 回填的载体）必须存在且点名 AGENT_EVALS §四；
//   - `scripts/release-version.sh` 的「发版前人工复核」清单必须含度量表对账（发版时对账不靠记性）。
// 做不到机械化的部分（评分是否公允、返工次数是否属实）仍是人工评审，不在此假装覆盖。

// agentEvalsMetricStatusRe 解析 §四 的「⚠️ **当前状态（YYYY-MM-DD）：N 条已回填**」行。
var agentEvalsMetricStatusRe = regexp.MustCompile(`\*\*当前状态（(\d{4}-\d{2}-\d{2})）：(\d+) 条已回填\*\*`)

// minAgentEvalsMetricCols 是度量表的最小列数（PR# / 工具 / 任务类型 / 门禁结果 / 返工次数 / 评分）。
const minAgentEvalsMetricCols = 6

// TestAgentEvalsMetricLedgerIsSelfConsistent 断言度量表的数据行数与状态行声明的条数一致。
func TestAgentEvalsMetricLedgerIsSelfConsistent(t *testing.T) {
	text := readRepoFile(t, agentEvalsDocPath)

	start := strings.Index(text, "## 四、")
	end := strings.Index(text, "## 五、")
	if start < 0 || end <= start {
		t.Fatalf("%s 未解析出 §四 / §五 边界（章节标题变更需同步本门禁）", agentEvalsDocPath)
	}
	section := text[start:end]

	status := agentEvalsMetricStatusRe.FindStringSubmatch(section)
	if status == nil {
		t.Fatalf("%s §四 未找到「**当前状态（YYYY-MM-DD）：N 条已回填**」状态行——"+
			"没有显式状态就无法机械对账（口径变更需同步本门禁）", agentEvalsDocPath)
	}
	want, err := strconv.Atoi(status[2])
	if err != nil {
		t.Fatalf("解析状态行条数 %q: %v", status[2], err)
	}

	rows, width := 0, 0
	for _, line := range strings.Split(section, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "|") {
			continue
		}
		cells := strings.Split(strings.Trim(line, "|"), "|")
		first := strings.TrimSpace(cells[0])
		if first == "" || first == "PR#" || strings.HasPrefix(first, "---") {
			continue // 表头 / 分隔行
		}
		if first == "—" || first == "-" {
			continue // 占位行（尚未回填）
		}
		rows++
		width = len(cells)
	}
	if rows != want {
		t.Errorf("%s §四 度量表有 %d 条真实数据行，但状态行声明「%d 条已回填」——"+
			"回填一行必须同步改状态行（两处漂移 = 度量口径失真）", agentEvalsDocPath, rows, want)
	}
	if rows > 0 && width < minAgentEvalsMetricCols {
		t.Errorf("%s §四 度量表列数 = %d，少于 %d 列（PR# / 工具 / 任务类型 / 门禁结果 / 返工次数 / 评分）——"+
			"回填时不得丢列", agentEvalsDocPath, width, minAgentEvalsMetricCols)
	}
	t.Logf("§四 度量台账：%d 条已回填（状态行声明 %d，基准日 %s）", rows, want, status[1])
}

// TestPRTemplateRequiresMetricBackfill 断言 PR 模板的「AI 度量」勾选项存在，且点名
// AGENT_EVALS §四 的度量表——它是「披露 → 回填」的唯一载体，被删则回填无入口。
func TestPRTemplateRequiresMetricBackfill(t *testing.T) {
	text := readRepoFile(t, filepath.Join(".github", "PULL_REQUEST_TEMPLATE.md"))
	var line string
	for _, l := range strings.Split(text, "\n") {
		if strings.Contains(l, "AGENT_EVALS.md") && strings.Contains(l, "回填") {
			line = l
			break
		}
	}
	if line == "" {
		t.Fatalf("PR 模板缺少「AI 度量」勾选项（需同时含 `AGENT_EVALS.md` 与「回填」）——"+
			"§四 度量表的回填入口消失，度量永远停在基线（见 %s）", agentEvalsDocPath)
	}
	if !strings.Contains(line, "[ ]") {
		t.Errorf("PR 模板的 AI 度量行不是可勾选清单项（缺 `[ ]`）：%s", strings.TrimSpace(line))
	}
	if !strings.Contains(line, "§四") {
		t.Errorf("PR 模板的 AI 度量行未点名 `§四`（度量表所在小节）：%s", strings.TrimSpace(line))
	}
}

// TestReleaseScriptRemindsMetricLedgerReview 断言发版脚本印出「发版前人工复核」清单且含度量表对账——
// 发版时对账是 §四 的第二道机制（第一道是 PR 收口回填），只在文档里写「发版前复核」等于靠记性。
func TestReleaseScriptRemindsMetricLedgerReview(t *testing.T) {
	text := readRepoFile(t, filepath.Join("scripts", "release-version.sh"))
	for _, kw := range []string{"发版前人工复核", "AGENT_EVALS.md §四 度量表", "对账"} {
		if !strings.Contains(text, kw) {
			t.Errorf("scripts/release-version.sh 的复核提醒缺少 %q——发版时对账的机械锚点消失", kw)
		}
	}
	if !strings.Contains(text, "REMINDER") {
		t.Error("scripts/release-version.sh 未用 heredoc 印出复核清单（提示只在注释里 = 跑发版的人看不到）")
	}
	// heredoc 起始行必须是**未被注释**的（`# cat <<'REMINDER'` 注解掉 = 跑发版时什么都不印）。
	printed := false
	for _, l := range strings.Split(text, "\n") {
		if strings.TrimSpace(l) == "cat <<'REMINDER'" {
			printed = true
			break
		}
	}
	if !printed {
		t.Error("scripts/release-version.sh 的复核清单 heredoc 未真正执行（`cat <<'REMINDER'` 被注释或改写）——" +
			"注释掉的提醒等于没有提醒")
	}
}
