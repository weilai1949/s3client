package main

// doc_ci_drift_gate_test.go —— 「文档里引用的 CI 事实必须与配置文件一致」的源码门禁
// （背景：docs/code-review-2026-10-09.md §5 O11 / KNOWN_ISSUES #82，2026-10-09）。
//
// 评审点名的三处 DEVELOPMENT 漂移此前**无门禁**，只能靠人肉比对：
//   - 「CI 双平台一致性」表把 perf.yml 的 job 写成不存在的 `perf-budget`（真实 id 是
//     `bench`）——引用了一个永远跑不起来的 job；
//   - 覆盖率段落只写了「前端 `i18n/**` 被排除统计」，而 `vite.config.ts` 实际排除
//     5 项（main.ts / env.d.ts / *.test.ts / i18n/messages/** / assets/**），
//     低估了门禁的「去水分」面；
//   - 触发事件的 job 计数差一。
//
// 前两类可机械校验，故写成门禁：文档引用的 job 必须真实存在、覆盖率排除项必须被
// 完整写出。第三类（自由叙述的计数）无法稳定解析，改由本文件头注释登记口径，
// 不设断言以免变成「为数字写脆测试」。
//
// 解析口径（零依赖、纯文本扫描，与 ci_consistency_gate_test.go 同源）：
//   - job 引用：`X.yml` · `Y`（Y 为 job id），在 `.github/workflows/X.yml` 的
//     `jobs:` 下必须存在；扫描面自检（至少解析到 6 处引用）防正则塌缩；
//   - 覆盖率排除项：定位 `vite.config.ts` 的 `coverage:` 块内 `exclude: [` 列表，
//     每个字面量必须在 DEVELOPMENT 的覆盖率段落出现。

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// devDocCIPlatformJobRefRe 匹配 DEVELOPMENT「CI 双平台一致性」表里的 “ `X.yml` · `Y` “。
var devDocCIPlatformJobRefRe = regexp.MustCompile("`([a-z0-9-]+\\.ya?ml)` · `([a-z0-9_-]+)`")

// minDevDocJobRefs 是扫描面自检基线（实测 2026-10-09：ci / codeql / scorecard /
// dependency-review / perf / fuzz 共 6 处以上 job 引用）。低于它说明正则塌缩。
const minDevDocJobRefs = 6

// TestDevelopmentCIPlatformTableJobRefsExist（O11 #82）：DEVELOPMENT 的 CI 表里
// 每个 “ `X.yml` · `Y` “ 的 Y 必须是我仓库里真实存在的 job id（perf-budget 事故）。
func TestDevelopmentCIPlatformTableJobRefsExist(t *testing.T) {
	doc := readRepoFile(t, filepath.Join("docs", "DEVELOPMENT.md"))
	matches := devDocCIPlatformJobRefRe.FindAllStringSubmatch(doc, -1)
	if len(matches) < minDevDocJobRefs {
		t.Fatalf("只解析到 %d 处 `` `X.yml` · `Y` `` 引用（基线 ≥%d）——正则塌缩", len(matches), minDevDocJobRefs)
	}
	for _, m := range matches {
		wf, job := m[1], m[2]
		path := filepath.Join(".github", "workflows", wf)
		data, err := os.ReadFile(filepath.Join(repoRoot(t), path))
		if err != nil {
			t.Errorf("DEVELOPMENT 引用了不存在的 workflow %q（%v）", wf, err)
			continue
		}
		if !ciJobExists(string(data), job) {
			t.Errorf("DEVELOPMENT 引用 `%s` · `%s`，但 %s 里没有这个 job id——文档引用了跑不起来的 job（O11 #82）",
				wf, job, path)
		}
	}
}

// ciJobExists 判断 GitHub workflow 文本的 `jobs:` 下是否存在该 job id。
func ciJobExists(workflow, job string) bool {
	for _, j := range ciJobBlocks(workflow, ciGitHubJobKeyRe) {
		if j.name == job {
			return true
		}
	}
	return false
}

// viteCoverageExcludeRe 匹配 vite.config.ts 覆盖率 exclude 数组里的单引号字面量。
var viteCoverageExcludeRe = regexp.MustCompile(`'([^']+)'`)

// TestDevelopmentDocumentsAllCoverageExclusions（O11 #82）：DEVELOPMENT 覆盖率段落
// 必须完整写出 vite.config.ts 实际排除的每一项——只写 i18n 会让读者误以为业务代码
// 全在统计内，低估门禁的去水分面。
func TestDevelopmentDocumentsAllCoverageExclusions(t *testing.T) {
	vite := readRepoFile(t, filepath.Join("apps", "web", "vite.config.ts"))
	covIdx := strings.Index(vite, "coverage:")
	if covIdx < 0 {
		t.Fatal("vite.config.ts 未找到 coverage 配置块（解析口径需同步）")
	}
	tail := vite[covIdx:]
	excIdx := strings.Index(tail, "exclude: [")
	if excIdx < 0 {
		t.Fatal("vite.config.ts 的 coverage 块未找到 exclude 数组（解析口径需同步）")
	}
	tail = tail[excIdx:]
	end := strings.Index(tail, "]")
	if end < 0 {
		t.Fatal("vite.config.ts 的 coverage.exclude 数组未闭合（解析口径需同步）")
	}
	excludes := viteCoverageExcludeRe.FindAllStringSubmatch(tail[:end], -1)
	if len(excludes) < 3 {
		t.Fatalf("只解析到 %d 项覆盖率排除（基线 ≥3）——解析口径塌缩", len(excludes))
	}

	doc := readRepoFile(t, filepath.Join("docs", "DEVELOPMENT.md"))
	for _, e := range excludes {
		entry := e[1]
		if !strings.Contains(doc, entry) {
			t.Errorf("DEVELOPMENT 的覆盖率段落漏写排除项 %q——低估门禁去水分面（O11 #82）", entry)
		}
	}
}
