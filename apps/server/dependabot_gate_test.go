package main

// dependabot_gate_test.go —— `.github/dependabot.yml` 的**清单路径**门禁。
//
// 背景（2026-09-29 文档失真收口时实测）：该文件的 Go / npm / Rust 三条更新规则曾把目录写成
// `/server`、`/web`、`/desktop/src-tauri`——三个目录在仓库根**都不存在**（真实路径在 `apps/` 下）。
// Dependabot 在配置目录下找不到清单时会**静默跳过**：表面上「依赖更新配好了」，实际从不开 PR，
// 安全补丁不会自动跟进，而流水线不会因此变红。
//
// 为什么此前没人发现：`repo_infra_gate_test.go` 覆盖的是镜像 pin / workflow pin / 供应链 / 契约，
// 不看 dependabot；YAML 又不在 Go 的编译与 lint 保护范围内——属典型的「配置存在但断言缺席」。
//
// 断言范围（刻意不做的事）：
//   - 只校验「每个 ecosystem 的 directory 存在，且该目录下含对应清单文件」；
//   - **不**校验 schedule / open-pull-requests-limit / 分组与标签策略——那些是策略偏好，
//     不是「配错了」，写死它们会让门禁在正常调整策略时误报；
//   - 出现未登记的 ecosystem 时**红灯**（防止新增一条不受保护的规则后门禁静默放行）。

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// dependabotEcosystemRe / dependabotDirectoryRe 按行解析，避免为一道门禁引入 YAML 依赖
// （与 `internal/openapi` 不引额外依赖的取舍一致）。
var (
	dependabotEcosystemRe = regexp.MustCompile(`(?m)^\s*-\s*package-ecosystem:\s*(\S+)\s*$`)
	dependabotDirectoryRe = regexp.MustCompile(`(?m)^\s*directory:\s*(\S+)\s*$`)
)

// dependabotManifest 登记每个 ecosystem 在其 directory 下**必须存在**的清单文件；
// 值为空串表示只要求目录存在（`github-actions` 看 `.github/workflows/`，`docker` 看仓库根）。
// 新增一条 dependabot 规则时必须在此登记，否则门禁红灯点名。
var dependabotManifest = map[string]string{
	"gomod":          "go.mod",
	"npm":            "package.json",
	"cargo":          "Cargo.toml",
	"github-actions": "",
	"docker":         "",
}

func TestDependabotDirectoriesExistAndHoldManifests(t *testing.T) {
	t.Parallel()
	b, err := os.ReadFile(filepath.Join(repoRoot(t), ".github", "dependabot.yml"))
	if err != nil {
		t.Fatalf("读取 .github/dependabot.yml: %v", err)
	}
	src := string(b)
	ecos := dependabotEcosystemRe.FindAllStringSubmatch(src, -1)
	dirs := dependabotDirectoryRe.FindAllStringSubmatch(src, -1)
	// 命中 0 条即红灯：文案 / 缩进漂移后正则会静默失配，门禁形同虚设（口径同 doc_number_gate_test.go）。
	if len(ecos) == 0 {
		t.Fatal("未解析到任何 package-ecosystem，疑似解析口径失效——门禁已形同虚设")
	}
	if len(ecos) != len(dirs) {
		t.Fatalf("解析结果不成对：ecosystem %d 条 / directory %d 条（每条更新规则应各有一个 directory）", len(ecos), len(dirs))
	}
	for i := range ecos {
		ecosystem, dir := ecos[i][1], dirs[i][1]
		if !strings.HasPrefix(dir, "/") {
			t.Errorf("第 %d 条（%s）的 directory %q 不是以 / 开头的仓库根相对路径", i+1, ecosystem, dir)
			continue
		}
		abs := filepath.Join(repoRoot(t), filepath.FromSlash(dir))
		st, err := os.Stat(abs)
		if err != nil || !st.IsDir() {
			t.Errorf("%s 的 directory %q 指向的目录不存在——Dependabot 会静默跳过该规则，依赖更新失效",
				ecosystem, dir)
			continue
		}
		want, known := dependabotManifest[ecosystem]
		if !known {
			t.Errorf("出现未登记的 package-ecosystem %q：请在本文件的 dependabotManifest 里补上其清单文件名，"+
				"否则该规则不受本门禁保护", ecosystem)
			continue
		}
		if want == "" {
			continue
		}
		if _, err := os.Stat(filepath.Join(abs, want)); err != nil {
			t.Errorf("%s 的 directory %q 下找不到清单文件 %s——Dependabot 会静默跳过",
				ecosystem, dir, want)
		}
	}
}
