package main

// ci_consistency_gate_test.go —— 「CI 必须给仓库门禁提供可复现前置」的源码门禁
// （背景：docs/archive/code-review-2026-10-09.md §3 C1 / C2，2026-10-09）。
//
// 评审发现：仓库宣称的「全绿门禁」在两个 CI 平台的 Go job 上是**构造性红灯**，
// 且被本地环境掩盖——本地 checkout 有 .git 全量历史 + node_modules，CI 都没有：
//
//   - C1：跑全量 `go test ./...` 的 job 未装 apps/web/node_modules。根包
//     agent_evals 门禁断言黄金任务 GT-4 的判据前置
//     requires_path=apps/web/node_modules 存在（os.Stat 失败即红）；该目录是
//     gitignore 构建产物，CI 不装必缺 → Go job 必红。修法只能是装真实前置
//     （pnpm install --frozen-lockfile），**不得 mkdir 伪造目录**。
//   - C2：actions/checkout 默认浅克隆（fetch-depth: 1，不带 tag）、GitLab 默认
//     GIT_DEPTH=20——而 changelog_tag_gate 刻意不调 git 命令、直读
//     .git/refs/tags 与 .git/packed-refs，要求 ≥15 个 `v*` tag。浅历史里旧 tag
//     缺失 → Go job 必红。修法是全量历史：`fetch-depth: 0` / `GIT_DEPTH: "0"`。
//
// 两类问题都只能靠读 CI 配置本身来钉（本地跑不了 Actions / Runner），因此把
// 「跑全量 Go 测试的 job 必须装前端前置」「所有 checkout 必须全量历史 + tag」
// 写成源码门禁——CI 配置改回去立即红灯，而不是等下次 CI 绿灯假象再骗一次人。
//
// 解析口径（刻意零依赖、纯行扫描；阈值自检防口径塌缩）：
//   - GitHub：`jobs:` 下两空格缩进的 job 键界定块；块内出现同时含 `go test`、
//     `./...` 且不含 `-run` 的行 = 跑全量测试（`-run` 过滤过的 job 不跑根包门禁）；
//   - GitLab：顶格 job 键界定块，同上判据；
//   - checkout 全量历史：`uses: actions/checkout` 行到下一个列表项（step 的 `- `）
//     或 job / 顶层键之间必须出现 `fetch-depth: 0`。

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// 扫描面自检基线（实测 2026-10-09）：
//
//	10 个 GitHub workflow · 16 个 actions/checkout 步 · 两平台各 1 个全量 go test job。
//
// 低于阈值说明解析口径塌缩或 CI 结构大改，必须红灯而不是安静通过。
const (
	minCIWorkflowFiles    = 8
	minCICheckoutSteps    = 10
	minCIFullGoTestJobsGH = 1
	minCIFullGoTestJobsGL = 1
)

var (
	// ciStepItemRe 匹配 step 列表项（`- uses:` / `- name:`），用作 checkout 步窗口右界。
	ciStepItemRe = regexp.MustCompile(`^\s+-\s+`)
	// ciGitHubJobKeyRe 界定 GitHub `jobs:` 下两空格缩进的 job 键。
	ciGitHubJobKeyRe = regexp.MustCompile(`^  [A-Za-z0-9_-]+:\s*$`)
	// ciGitLabJobKeyRe 界定 GitLab 顶格的 job / 模板键。
	ciGitLabJobKeyRe = regexp.MustCompile(`^[A-Za-z0-9_.-]+:\s*$`)
	// ciTopLevelKeyRe 界定顶格顶层键（`jobs:` / `on:` …），同为窗口右界。
	ciTopLevelKeyRe = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*:\s*$`)
	// gitLabDepthRe 匹配 GIT_DEPTH: "0"（GitLab 克隆深度 0 = 不限深度，含全部 tag）。
	gitLabDepthRe = regexp.MustCompile(`(?m)^\s*GIT_DEPTH:\s*"?0"?\s*$`)
)

// ciJob 是切分出来的单个 CI job（或 GitLab 模板）块。
type ciJob struct {
	name string
	body string
}

// ciJobBlocks 按 keyRe 把 CI 配置文本切成 job 块（键行本身计入下一块的 name）。
func ciJobBlocks(text string, keyRe *regexp.Regexp) []ciJob {
	var jobs []ciJob
	var cur *ciJob
	for _, ln := range strings.Split(text, "\n") {
		if keyRe.MatchString(ln) {
			if cur != nil {
				jobs = append(jobs, *cur)
			}
			cur = &ciJob{name: strings.TrimSuffix(strings.TrimSpace(ln), ":")}
			continue
		}
		if cur != nil {
			cur.body += ln + "\n"
		}
	}
	if cur != nil {
		jobs = append(jobs, *cur)
	}
	return jobs
}

// runsFullGoTest 判断 job 是否跑「全量 go test ./...」：单行同时含 `go test`、
// `./...` 且不含 `-run`（带 -run 的只跑子集，不触根包门禁，无需前置）。
// 单行判据的口径限制（如续行拆分）由上方扫描面自检兜底——job 数掉下基线即红。
func runsFullGoTest(body string) bool {
	for _, ln := range strings.Split(body, "\n") {
		if strings.Contains(ln, "go test") && strings.Contains(ln, "./...") && !strings.Contains(ln, "-run") {
			return true
		}
	}
	return false
}

// ciWorkflowFiles 返回 .github/workflows 下全部 workflow 文件（*.yml / *.yaml）。
func ciWorkflowFiles(t *testing.T) []string {
	t.Helper()
	dir := filepath.Join(repoRoot(t), ".github", "workflows")
	var out []string
	for _, pat := range []string{"*.yml", "*.yaml"} {
		matches, err := filepath.Glob(filepath.Join(dir, pat))
		if err != nil {
			t.Fatalf("glob %s: %v", pat, err)
		}
		out = append(out, matches...)
	}
	if len(out) == 0 {
		t.Fatal("未找到任何 GitHub workflow 文件（解析口径需同步）")
	}
	return out
}

// ciCheckoutWindow 返回 checkout 的 `uses:` 行之后、到下一个 step / job 边界之前的文本
// （即该 step 自己的 `with:` 块），供 fetch-depth 断言使用。
func ciCheckoutWindow(lines []string, i int, jobKeyRe *regexp.Regexp) string {
	var b []string
	for j := i + 1; j < len(lines); j++ {
		ln := lines[j]
		if ciStepItemRe.MatchString(ln) || jobKeyRe.MatchString(ln) || ciTopLevelKeyRe.MatchString(ln) {
			break
		}
		b = append(b, ln)
	}
	return strings.Join(b, "\n")
}

// TestFullGoTestJobsInstallWebPrecondition（C1）：每个跑全量 `go test ./...` 的
// CI job 必须真实安装 apps/web/node_modules 前置（agent_evals 门禁 GT-4 的
// requires_path 会 os.Stat 它）。装不了就别跑全量——判据是二选一，不接受 mkdir 伪造。
func TestFullGoTestJobsInstallWebPrecondition(t *testing.T) {
	var files, ghFull int
	for _, p := range ciWorkflowFiles(t) {
		files++
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("读 %s: %v", p, err)
		}
		rel, relErr := filepath.Rel(repoRoot(t), p)
		if relErr != nil {
			rel = p
		}
		for _, j := range ciJobBlocks(string(data), ciGitHubJobKeyRe) {
			if !runsFullGoTest(j.body) {
				continue
			}
			ghFull++
			if !strings.Contains(j.body, "pnpm install --frozen-lockfile") {
				t.Errorf("%s 的 job %q 跑全量 `go test ./...` 却不装前端依赖（C1：根包 agent_evals 门禁断言 requires_path=apps/web/node_modules 存在，CI 不装必红——须 pnpm install --frozen-lockfile 装真实前置，不得 mkdir 伪造）",
					filepath.ToSlash(rel), j.name)
			}
		}
	}
	if files < minCIWorkflowFiles {
		t.Errorf("只解析到 %d 个 workflow（基线 ≥%d）——解析口径塌缩", files, minCIWorkflowFiles)
	}
	if ghFull < minCIFullGoTestJobsGH {
		t.Errorf("GitHub 侧只找到 %d 个全量 go test job（基线 ≥%d）——解析口径塌缩", ghFull, minCIFullGoTestJobsGH)
	}

	glFull := 0
	for _, j := range ciJobBlocks(readRepoFile(t, ".gitlab-ci.yml"), ciGitLabJobKeyRe) {
		if !runsFullGoTest(j.body) {
			continue
		}
		glFull++
		if !strings.Contains(j.body, "pnpm install --frozen-lockfile") {
			t.Errorf(".gitlab-ci.yml 的 job %q 跑全量 `go test ./...` 却不装前端依赖（C1 同上）", j.name)
		}
	}
	if glFull < minCIFullGoTestJobsGL {
		t.Errorf("GitLab 侧只找到 %d 个全量 go test job（基线 ≥%d）——解析口径塌缩", glFull, minCIFullGoTestJobsGL)
	}
}

// TestCIRepoCheckoutsFetchFullHistory（C2）：所有 actions/checkout 必须
// fetch-depth: 0、GitLab 必须 GIT_DEPTH: "0"——changelog_tag_gate 直读
// .git/refs/tags / packed-refs 要求 ≥15 个 v* tag，浅克隆不带旧 tag 即构造性红灯。
func TestCIRepoCheckoutsFetchFullHistory(t *testing.T) {
	steps := 0
	for _, p := range ciWorkflowFiles(t) {
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("读 %s: %v", p, err)
		}
		rel, relErr := filepath.Rel(repoRoot(t), p)
		if relErr != nil {
			rel = p
		}
		lines := strings.Split(string(data), "\n")
		for i, ln := range lines {
			// 注释里的示例行不是真 checkout（dependency-review / scorecard 有此类注释）。
			if strings.HasPrefix(strings.TrimSpace(ln), "#") {
				continue
			}
			if !strings.Contains(ln, "uses: actions/checkout@") {
				continue
			}
			steps++
			window := ciCheckoutWindow(lines, i, ciGitHubJobKeyRe)
			if !strings.Contains(window, "fetch-depth: 0") {
				t.Errorf("%s:%d 的 actions/checkout 未声明 fetch-depth: 0（C2：浅克隆不带 tag，changelog_tag_gate 要求 ≥15 个 v* tag，CI 必红）",
					filepath.ToSlash(rel), i+1)
			}
		}
	}
	if steps < minCICheckoutSteps {
		t.Errorf("只解析到 %d 个 actions/checkout 步（基线 ≥%d）——解析口径塌缩", steps, minCICheckoutSteps)
	}
	if !gitLabDepthRe.MatchString(readRepoFile(t, ".gitlab-ci.yml")) {
		t.Error(".gitlab-ci.yml 未设置 GIT_DEPTH: \"0\"（C2：默认 GIT_DEPTH=20 浅克隆缺旧 tag，changelog_tag_gate 必红）")
	}
}

// TestRustFSE2ETriggersCoverWholeBackend（评审 2026-10-09 R10 / O11 #82）：Go 端真
// RustFS E2E（e2e.yml / .e2e-rustfs-trigger）的 PR 触发路径此前只列
// `internal/s3wrap/**`——只改 `internal/handler/**`（或 service / config）的 PR 会
// 静默跳过这条唯一的真 S3 对端门禁。两套 CI 都必须把**整个后端**纳入触发面并保持一致。
func TestRustFSE2ETriggersCoverWholeBackend(t *testing.T) {
	gh := readRepoFile(t, filepath.Join(".github", "workflows", "e2e.yml"))
	if !strings.Contains(gh, "'apps/server/**'") {
		t.Error("GitHub e2e.yml 的 pull_request paths 必须含 'apps/server/**'（否则非 s3wrap 的后端改动跳过真 RustFS Go E2E，O11 #82）")
	}
	if strings.Contains(gh, "'apps/server/internal/s3wrap/**'") {
		t.Error("GitHub e2e.yml 仍残留仅 s3wrap 的窄路径 'apps/server/internal/s3wrap/**'——应被 'apps/server/**' 取代（O11 #82）")
	}

	var block string
	for _, j := range ciJobBlocks(readRepoFile(t, ".gitlab-ci.yml"), ciGitLabJobKeyRe) {
		if j.name == ".e2e-rustfs-trigger" {
			block = j.body
		}
	}
	if block == "" {
		t.Fatal("未在 .gitlab-ci.yml 找到 .e2e-rustfs-trigger 块（解析口径需同步）")
	}
	if !strings.Contains(block, "apps/server/**") {
		t.Error("GitLab .e2e-rustfs-trigger 的 changes 必须含 apps/server/**（与 GitHub 侧一致，O11 #82）")
	}
	if strings.Contains(block, "apps/server/internal/s3wrap/**") {
		t.Error("GitLab .e2e-rustfs-trigger 仍残留仅 s3wrap 的窄路径——应被 apps/server/** 取代（O11 #82）")
	}
}
