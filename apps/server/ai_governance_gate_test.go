package main

// ai_governance_gate_test.go —— 第 10 层「AI 时代」资产的**可执行不变量**。
//
// 背景（2026-09-29 盘点）：本仓库已有 根/子树 `AGENTS.md`、`llms.txt`、`docs/AI_POLICY.md`
// （五档模式 + 权限矩阵 + 披露 + DoD）、PR 模板披露块、机器可读契约与门禁族。但这些 AI 治理资产
// 此前**全靠人工维持**，而它们恰好是最容易被静默破坏的一类：
//
//   - 根 `AGENTS.md` 会被工具**整体注入模型上下文且受字节预算约束**（DEVELOPMENT.md §4.1），
//     超长会被截断——于是硬约束悄悄失效，而没有任何门禁看得见；
//   - 新增一个 `apps/` 子树却忘了写子树 `AGENTS.md` → 该子树规则对 agent 不可见；
//   - `AI_POLICY.md` 正文里的**事实声明**（如「当前未提交 MCP 配置」）会随仓库演进而失真。
//
// 断言范围：只钉「**结构不变量**」（体积 / 覆盖 / 回指 / 声明一致性），不评判规则内容是否合理；
// 政策里**没有**机械保证的条款由 AI_POLICY §11 如实列出，不在本门禁假装覆盖。
//
// 相关：`doc_index_gate_test.go`（文档导航覆盖）、`doc_link_gate_test.go`（链接与锚点）。

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const (
	// maxRootAgentsBytes 是根 AGENTS.md 的注入预算（当前实测 7995 B，2026-09-29）。
	// 依据 DEVELOPMENT.md §4.1：文件被**整体注入**且超长会被省略 / 截断，故必须保持短小；
	// 提高本阈值等于承认「注入后可能被截断」，需同时更新该节说明。
	maxRootAgentsBytes = 10 * 1024
	// maxSubtreeAgentsBytes 是单个子树 AGENTS.md 的上限（当前最大 1960 B）。
	maxSubtreeAgentsBytes = 4 * 1024
	// maxPointerBytes 是 AI 工具「指针文件」的上限——它必须只指向 SSOT，不得自成事实源。
	maxPointerBytes = 2 * 1024
	// minRootAgentsBytes 是自检下限：低于它说明扫描 / 读取口径失效。
	minRootAgentsBytes = 1024
)

// TestRootAgentsMdStaysWithinInjectionBudget 断言根 AGENTS.md 不会因超长被注入截断。
func TestRootAgentsMdStaysWithinInjectionBudget(t *testing.T) {
	t.Parallel()
	b, err := os.ReadFile(filepath.Join(repoRoot(t), "AGENTS.md"))
	if err != nil {
		t.Fatalf("读取根 AGENTS.md: %v", err)
	}
	if len(b) < minRootAgentsBytes {
		t.Fatalf("根 AGENTS.md 只有 %d 字节（下限 %d）：读取口径失效", len(b), minRootAgentsBytes)
	}
	if len(b) > maxRootAgentsBytes {
		t.Errorf("根 AGENTS.md 已 %d 字节，超出注入预算 %d：它会被工具**整体注入**，"+
			"超长将导致硬约束被截断而静默失效。请把可推导的细节移到 docs/DEVELOPMENT.md，只留指针",
			len(b), maxRootAgentsBytes)
	}
	if !strings.Contains(string(b), "docs/DEVELOPMENT.md") {
		t.Error("根 AGENTS.md 未指向规范正文 docs/DEVELOPMENT.md——「只留硬约束与指针」的纪律需要可机检")
	}
}

// TestEveryAppSubtreeHasAgentsMd 断言每个 apps 子树都有子树 AGENTS.md 且回指根文件。
func TestEveryAppSubtreeHasAgentsMd(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	entries, err := os.ReadDir(filepath.Join(root, "apps"))
	if err != nil {
		t.Fatalf("读取 apps/: %v", err)
	}
	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e.Name())
		}
	}
	if len(dirs) < 3 {
		t.Fatalf("apps/ 下只识别到 %d 个子树（期望 ≥3：server / web / desktop）：扫描口径失效", len(dirs))
	}
	for _, d := range dirs {
		p := filepath.Join(root, "apps", d, "AGENTS.md")
		b, err := os.ReadFile(p)
		if err != nil {
			t.Errorf("apps/%s 缺少子树 AGENTS.md：该子树的专属规则对 agent 不可见（新增子树必须同 PR 补上）", d)
			continue
		}
		if len(b) > maxSubtreeAgentsBytes {
			t.Errorf("apps/%s/AGENTS.md 已 %d 字节，超出子树预算 %d：子树文件只写**专属**约束，"+
				"仓库级规则留在根文件", d, len(b), maxSubtreeAgentsBytes)
		}
		if !strings.Contains(string(b), "../../AGENTS.md") {
			t.Errorf("apps/%s/AGENTS.md 未回指根 AGENTS.md：子树规则必须挂在仓库级规则之下（防两处分叉）", d)
		}
	}
}

// TestAiPolicyClaimsMatchRepoState 断言 AI_POLICY 正文里的**事实声明**与仓库现状一致。
func TestAiPolicyClaimsMatchRepoState(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	policyRaw, err := os.ReadFile(filepath.Join(root, "docs", "AI_POLICY.md"))
	if err != nil {
		t.Fatalf("读取 docs/AI_POLICY.md: %v", err)
	}
	policy := string(policyRaw)

	// 声明①：§4「本仓库当前未提交 MCP 配置」——一旦根目录出现 .mcp.json，政策必须同步改写。
	if strings.Contains(policy, "未提交 MCP 配置") {
		if _, err := os.Stat(filepath.Join(root, ".mcp.json")); err == nil {
			t.Error("AI_POLICY §4 声明「本仓库当前未提交 MCP 配置」，但根目录已有 .mcp.json——请同步政策（含权限口径）")
		}
	} else {
		t.Error("AI_POLICY 未找到「未提交 MCP 配置」声明：该声明受本门禁保护，措辞变更需同步本文件")
	}

	// 声明②：§5「实质性 AI 生成内容必须披露」——载体是 PR 模板的披露块，块被删则要求落空。
	tpl, err := os.ReadFile(filepath.Join(root, ".github", "PULL_REQUEST_TEMPLATE.md"))
	if err != nil {
		t.Fatalf("读取 PR 模板: %v", err)
	}
	if !strings.Contains(string(tpl), "AI 使用披露") {
		t.Error("PR 模板缺少「AI 使用披露」块：AI_POLICY §5 的披露要求失去载体（块本身受本门禁保护）")
	}

	// 声明③：若存在其它 AI 工具入口，它必须是**纯指针**（不得复制规则形成第二事实源）。
	pointer := filepath.Join(root, ".github", "copilot-instructions.md")
	if b, err := os.ReadFile(pointer); err == nil {
		if len(b) > maxPointerBytes {
			t.Errorf(".github/copilot-instructions.md 已 %d 字节，超出指针预算 %d："+
				"AI 工具入口只应指向 AGENTS.md 等 SSOT，复制规则会产生第二事实源", len(b), maxPointerBytes)
		}
		if !strings.Contains(string(b), "AGENTS.md") {
			t.Error(".github/copilot-instructions.md 未指向 AGENTS.md：指针文件失去唯一价值")
		}
	}
}

// ---------------------------------------------------------------------------
// A2/A6（2026-10-10 文档「明确没做」盘点）：AI_POLICY §11 把「不自动合并 PR /
// 不自动删分支或数据」如实标为「⚠️ 人工」——实测 `.github/workflows/`、`.gitlab-ci.yml`
// 与 `scripts/` 里零命中自动合并 / 删分支 / 特权触发，但**零命中只是现状，没有任何门禁**
// 阻止下一个改动把 `gh pr merge` 或 `pull_request_target` 加进来。本断言把该政策条款
// 从「人工承诺」升级为「代码强制」，并**双向钉**：政策 §11 必须回指本测试名，
// 仓库侧出现禁用模式即红灯。
// ---------------------------------------------------------------------------

// ciForbiddenRe 匹配「自动合并 / 自动删分支 / 特权触发」形态。
// 只扫 CI 载体（GitHub workflow / GitLab CI / scripts），不扫文档——文档里讨论这些
// 模式的段落（含 AI_POLICY 自己）是合法引用。
var ciForbiddenRe = mustCompileForbidden()

func mustCompileForbidden() *regexp.Regexp {
	return regexp.MustCompile(
		`gh pr merge|` + // GitHub CLI 自动合并
			`enable-auto-merge|` + // peter-evans/enable-auto-merge 等 action
			`mergify|` + // Mergify 配置 / app
			`merge_when_pipeline_succeeds|` + // GitLab 自动合并
			`auto_merge|` + // GitLab auto-merge API
			`pull_request_target|` + // 特权触发（fork PR 亦可写 token）
			`git push --delete|` + // 删远端分支
			`git branch -D|` + // 强删本地分支
			`delete-branch`) // 删分支 action
}

// minCIScannedFiles 是扫描面自检基线：10 个 GitHub workflow + 1 个 .gitlab-ci.yml
// + 9 个 scripts/*.sh = 20（低于 18 即读取口径塌缩）。
const minCIScannedFiles = 18

// TestNoAutoMergeOrPrivilegedCITriggers 断言仓库 CI 载体不含自动合并 / 自动删分支 /
// 特权触发模式，且 AI_POLICY §11 对应行已回指本测试名（双向钉）。
func TestNoAutoMergeOrPrivilegedCITriggers(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)

	// ① 收集全部 CI 载体：GitHub workflows + GitLab CI + scripts 下的 shell。
	files := append([]string{}, ciWorkflowFiles(t)...)
	for _, rel := range []string{filepath.Join(".gitlab-ci.yml")} {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Fatalf("缺少 %s: %v", rel, err)
		}
		files = append(files, filepath.Join(root, rel))
	}
	scripts, err := filepath.Glob(filepath.Join(root, "scripts", "*.sh"))
	if err != nil {
		t.Fatalf("glob scripts/*.sh: %v", err)
	}
	files = append(files, scripts...)

	if len(files) < minCIScannedFiles {
		t.Fatalf("扫描面塌缩：只收集到 %d 个 CI 载体文件（基线 ≥%d）——"+
			"workflow / scripts 目录结构变化需同步本门禁", len(files), minCIScannedFiles)
	}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("读取 %s: %v", f, err)
		}
		for i, line := range strings.Split(string(b), "\n") {
			if ciForbiddenRe.MatchString(line) {
				rel, _ := filepath.Rel(root, f)
				t.Errorf("%s:%d 命中「自动合并 / 自动删分支 / 特权触发」禁用模式——"+
					"AI_POLICY §3 禁止档（自动合并 PR、自动删分支或数据须人类执行）：\n  %s",
					rel, i+1, strings.TrimSpace(line))
			}
		}
	}

	// ② 双向钉：政策 §11 的对应行必须回指本测试名，否则「升级为代码强制」的说法无载体。
	policy := readRepoFile(t, filepath.Join("docs", "AI_POLICY.md"))
	testName := "TestNoAutoMergeOrPrivilegedCITriggers"
	if !strings.Contains(policy, testName) {
		t.Errorf("docs/AI_POLICY.md §11 未回指 %s："+
			"「不自动合并 PR / 不自动删分支」的覆盖强度声明必须与门禁一一对应", testName)
	}
}
