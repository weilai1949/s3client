package main

// repo_infra_gate_test.go —— 仓库级基础设施配置的源码门禁。
//
// 这些都不是 Go 代码，因此既有的单测 / lint / 覆盖率门禁全都看不见它们，
// 但都会直接决定「照 README 跑起来的生产实例」的安全与发布基线：
//
//   - S2：`.env.example` 的占位 token 必须**不能**通过 `config.MinTokenLength`，
//     否则 `cp .env.example .env && docker compose up -d` 会以一个**公开已知**的
//     token 上线（compose 的 `${S3C_TOKEN:?}` 只校验非空）。
//   - S1：运行镜像的基础版必须是仍受安全支持的 alpine。`--ignore-unfixed` 让 Trivy
//     在 EOL 分支上「所有未来 CVE 永远无修复版」，安全门禁结构性失明。
//   - R2/R3/R4：发布链的 tag↔清单一致性、SHA256SUMS 平台内唯一命名、Trivy DB 缓存与重试。
//     这三项此前只存在于 YAML 里，没有回归断言——改动 workflow 时静默退化。
//
// 它们都是「配置正确但断言缺席」的盲区，故用本文件把断言补上。

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/weilai1949/s3clinet/apps/server/internal/config"
)

// repoRoot 返回仓库根目录（apps/server -> apps -> 仓库根）。
func repoRoot(t *testing.T) string {
	t.Helper()
	return filepath.Join(serverRoot(t), "..", "..")
}

// envTokenRe 匹配 `.env.example` 中 S3C_TOKEN 的赋值行。
var envTokenRe = regexp.MustCompile(`(?m)^[ \t]*S3C_TOKEN[ \t]*=[ \t]*(.*)$`)

// TestEnvExampleTokenIsNotAValidCredential 断言仓库根 `.env.example` 的 S3C_TOKEN
// 是空值（或至少短于 MinTokenLength，保证 compose 守卫之外还有一层失败）。
//
// 背景：`change-me-use-openssl-rand-hex-32` 长 33 ≥ MinTokenLength=16，满足
// `${S3C_TOKEN:?}` 的非空校验 → 快速开始会以公开已知口令上线。正确写法见
// `apps/server/.env.example`（`S3C_TOKEN=` 空值 + 注释提示生成命令）。
func TestEnvExampleTokenIsNotAValidCredential(t *testing.T) {
	path := filepath.Join(repoRoot(t), ".env.example")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取 %s: %v", path, err)
	}
	m := envTokenRe.FindStringSubmatch(string(data))
	if m == nil {
		t.Fatalf("%s 缺少 S3C_TOKEN 赋值行（compose 的 ${S3C_TOKEN:?} 依赖它）", path)
	}
	value := strings.TrimSpace(m[1])
	// 允许行尾注释：`S3C_TOKEN=  # 生成：openssl rand -hex 32`
	if i := strings.Index(value, "#"); i >= 0 {
		value = strings.TrimSpace(value[:i])
	}
	if value == "" {
		return // 正确：留空，强制使用者显式填写。
	}
	if len(value) >= config.MinTokenLength {
		t.Errorf("%s 的 S3C_TOKEN 占位值 %q 长 %d ≥ MinTokenLength=%d：它是一枚**有效口令**，"+
			"照 README 快速开始会以公开已知 token 上线。请置空（对齐 apps/server/.env.example）。",
			path, value, len(value), config.MinTokenLength)
	}
}

// alpineFromRe 匹配 Dockerfile 的运行镜像行 `FROM alpine:3.24`（可带 AS 别名）。
var alpineFromRe = regexp.MustCompile(`(?m)^[ \t]*FROM[ \t]+alpine:(\d+)\.(\d+)`)

// minSupportedAlpineMinor 是 alpine 3.x 的最低受支持 minor。
//
// 依据 endoflife.date（2026-09-18）：3.20 的安全支持已于 2026-04-01 结束，故基线
// 抬到 3.21；3.21 于 2026-11-01 EOL、3.22 于 2027-05-01、3.23 于 2027-11-01、
// 3.24 于 2028-06-01。**3.21 EOL 时须把该常量抬到 3.22**，否则本门禁会在 EOL 分支上
// 继续放行——这正是它要防的失效模式。
const minSupportedAlpineMinor = 21

// TestRuntimeBaseImageIsSupportedAlpine 断言运行镜像不是已 EOL 的 alpine 分支。
// 只校验运行阶段（最后一个 FROM alpine:...）：构建阶段用 golang/node 镜像，与运行时漏洞面无关。
func TestRuntimeBaseImageIsSupportedAlpine(t *testing.T) {
	path := filepath.Join(repoRoot(t), "apps", "server", "Dockerfile")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取 %s: %v", path, err)
	}
	matches := alpineFromRe.FindAllStringSubmatch(string(data), -1)
	if len(matches) == 0 {
		t.Fatalf("%s 未找到 `FROM alpine:<major>.<minor>` 运行镜像行（解析口径需同步）", path)
	}
	// 取最后一次出现的 alpine FROM（多阶段构建的最终运行阶段）。
	last := matches[len(matches)-1]
	major, _ := strconv.Atoi(last[1])
	minor, _ := strconv.Atoi(last[2])
	if major != 3 {
		t.Fatalf("%s 运行镜像 alpine major=%d，门禁口径只覆盖 3.x", path, major)
	}
	if minor < minSupportedAlpineMinor {
		t.Errorf("%s 运行镜像为 alpine:%s：该分支安全支持已结束，叠加 Trivy 的 --ignore-unfixed "+
			"会让漏洞门禁结构性失明。请升级到 alpine:3.%d 或更高（当前最新 3.24，支持至 2028-06-01）。",
			path, last[1]+"."+last[2], minSupportedAlpineMinor)
	}
}

// ---- 供应链 pin（#32 Trivy / #33 Rust + Node + pnpm）----

var (
	// trivyPinnedRe 匹配「tag + digest 双 pin」的 Trivy 镜像引用。
	trivyPinnedRe = regexp.MustCompile(`aquasec/trivy:(\d+\.\d+\.\d+)@sha256:([0-9a-f]{64})`)
	// rustToolchainUseRe 匹配 `uses: dtolnay/rust-toolchain@<ref>` 及其行尾版本注释。
	rustToolchainUseRe = regexp.MustCompile(`(?m)^[ \t]*-?[ \t]*uses:[ \t]*dtolnay/rust-toolchain@([^\s#]+)[ \t]*(?:#[ \t]*(.*))?$`)
	// rustTomlChannelRe 匹配 rust-toolchain.toml 的 `channel = "<version>"`。
	rustTomlChannelRe = regexp.MustCompile(`(?m)^[ \t]*channel[ \t]*=[ \t]*"([^"]+)"`)
	// nodeVersionRe 匹配 GitHub Actions 的 `node-version: <v>`。
	nodeVersionRe = regexp.MustCompile(`(?m)^[ \t]*node-version:[ \t]*(\S+)[ \t]*$`)
	// nodeImageRe 匹配容器镜像 tag `node:<version>-<suffix>`。
	nodeImageRe = regexp.MustCompile(`node:(\d+(?:\.\d+)*)-`)
	// sha40Re / semverRe 是 pin 形状的判据：不可变 commit SHA、完整 X.Y.Z。
	sha40Re  = regexp.MustCompile(`^[0-9a-f]{40}$`)
	semverRe = regexp.MustCompile(`^\d+\.\d+\.\d+$`)
	// stableRustToolchainRe 匹配残留的浮动 pin `dtolnay/rust-toolchain@…# stable`。
	stableRustToolchainRe = regexp.MustCompile(`dtolnay/rust-toolchain@[^\s#]+[ \t]*#[ \t]*stable`)
	// nodesourceExactRe 匹配 NodeSource 的精确 apt 版本 pin `nodejs=<X.Y.Z>-…`。
	nodesourceExactRe = regexp.MustCompile(`nodejs=(\d+\.\d+\.\d+)-`)
)

// stableRustToolchainSHA 是 dtolnay/rust-toolchain `stable` 分支在 2026-09-03 的 tip
// （commit message 即 "toolchain: stable"）。它语义上仍是「跟随 stable」的浮动引用：
// 同样的 SHA 在不同时间会装出不同 rustc，故禁止回退到它。
const stableRustToolchainSHA = "6bed0761d98439e5a578e2877258200ad565ba87"

// TestTrivyImageIsVersionAndDigestPinned（#32）：两套 CI 的 Trivy 必须同版本、同 digest，
// 且不再使用已弃用的 `--vuln-type`（0.74.0 起为 `--pkg-types` 的 deprecated 别名）。
func TestTrivyImageIsVersionAndDigestPinned(t *testing.T) {
	files := []string{
		filepath.Join(".github", "workflows", "ci.yml"),
		".gitlab-ci.yml",
	}
	var refs []string
	for _, rel := range files {
		text := readRepoFile(t, rel)
		matches := trivyPinnedRe.FindAllStringSubmatch(text, -1)
		if len(matches) == 0 {
			t.Errorf("%s 未找到 `aquasec/trivy:<semver>@sha256:<digest>` 引用（#32：tag 可被重新推送，必须叠加 digest）", rel)
		}
		for _, m := range matches {
			refs = append(refs, m[1]+"@sha256:"+m[2])
		}
		// 不允许「只 pin tag」的裸引用：`aquasec/trivy:` 出现次数必须全部带 digest。
		if n := strings.Count(text, "aquasec/trivy:"); n != len(matches) {
			t.Errorf("%s 有 %d 处 `aquasec/trivy:` 引用但只有 %d 处带 digest——裸 tag 引用不可复现", rel, n, len(matches))
		}
		if strings.Contains(text, "--vuln-type") {
			t.Errorf("%s 仍在使用已弃用的 `--vuln-type`，应改为 `--pkg-types`", rel)
		}
		if !strings.Contains(text, "--pkg-types os,library") {
			t.Errorf("%s 的 Trivy 扫描未显式使用 `--pkg-types os,library`", rel)
		}
	}
	if len(refs) == 0 {
		return
	}
	for _, ref := range refs[1:] {
		if ref != refs[0] {
			t.Errorf("两套 CI 的 Trivy 引用不一致：%q vs %q（必须同版本同 digest）", refs[0], ref)
		}
	}
}

// TestRustToolchainIsVersionPinned（#33）：dtolnay/rust-toolchain 必须按不可变 SHA pin，
// 行尾注释给出具体版本，且与 apps/desktop/src-tauri/rust-toolchain.toml 的 channel 一致。
func TestRustToolchainIsVersionPinned(t *testing.T) {
	files := []string{
		filepath.Join(".github", "workflows", "ci.yml"),
		filepath.Join(".github", "workflows", "release-desktop.yml"),
	}
	var versions []string
	for _, rel := range files {
		text := readRepoFile(t, rel)
		matches := rustToolchainUseRe.FindAllStringSubmatch(text, -1)
		if len(matches) == 0 {
			t.Errorf("%s 未找到 dtolnay/rust-toolchain 用法（解析口径需同步）", rel)
			continue
		}
		for _, m := range matches {
			ref, comment := m[1], strings.TrimSpace(m[2])
			if !sha40Re.MatchString(ref) {
				t.Errorf("%s 的 dtolnay/rust-toolchain 引用 %q 不是 40 位 commit SHA（浮动 tag/branch 不可复现）", rel, ref)
				continue
			}
			if ref == stableRustToolchainSHA {
				t.Errorf("%s 仍指向 rust-toolchain `stable` 分支 tip（%s）：它跟随 stable，不是版本 pin", rel, ref)
			}
			if !semverRe.MatchString(comment) {
				t.Errorf("%s 的 dtolnay/rust-toolchain 行缺少具体版本注释（形如 `# 1.98.1`），实际为 %q", rel, comment)
				continue
			}
			versions = append(versions, comment)
		}
		if stableRustToolchainRe.MatchString(text) {
			t.Errorf("%s 仍残留 `dtolnay/rust-toolchain@…# stable` 浮动 pin（#33）", rel)
		}
	}

	toml := readRepoFile(t, filepath.Join("apps", "desktop", "src-tauri", "rust-toolchain.toml"))
	m := rustTomlChannelRe.FindStringSubmatch(toml)
	if m == nil {
		t.Fatalf("apps/desktop/src-tauri/rust-toolchain.toml 缺少 `channel = \"<version>\"`（#33：本地与 CI 必须同版本）")
	}
	if !semverRe.MatchString(m[1]) {
		t.Errorf("rust-toolchain.toml 的 channel=%q 不是具体版本（X.Y.Z）", m[1])
	}
	for _, v := range versions {
		if v != m[1] {
			t.Errorf("workflow 的 rust 版本注释 %q 与 rust-toolchain.toml channel %q 不一致（本地与 CI 必须同版本）", v, m[1])
		}
	}
}

// TestNodePinnedToPatchVersion（#33）：GitHub 的 node-version、容器镜像 tag 与 NodeSource
// 安装都必须落到完整 patch 版本，不能只 pin 大版本。
func TestNodePinnedToPatchVersion(t *testing.T) {
	for _, rel := range []string{
		filepath.Join(".github", "workflows", "ci.yml"),
		filepath.Join(".github", "workflows", "release-desktop.yml"),
		filepath.Join(".github", "workflows", "e2e-playwright.yml"),
		filepath.Join(".github", "workflows", "e2e-real.yml"),
	} {
		text := readRepoFile(t, rel)
		matches := nodeVersionRe.FindAllStringSubmatch(text, -1)
		if len(matches) == 0 {
			t.Errorf("%s 未找到 node-version 配置（解析口径需同步）", rel)
		}
		for _, m := range matches {
			if !semverRe.MatchString(m[1]) {
				t.Errorf("%s 的 node-version=%q 只 pin 到非完整版本（应为 X.Y.Z）", rel, m[1])
			}
		}
	}
	for _, rel := range []string{".gitlab-ci.yml", filepath.Join("apps", "server", "Dockerfile")} {
		text := readRepoFile(t, rel)
		matches := nodeImageRe.FindAllStringSubmatch(text, -1)
		if len(matches) == 0 {
			t.Errorf("%s 未找到 node 镜像引用（解析口径需同步）", rel)
		}
		for _, m := range matches {
			if !semverRe.MatchString(m[1]) {
				t.Errorf("%s 的 node 镜像 tag 只 pin 到 %q（应为 X.Y.Z）", rel, m[1])
			}
		}
	}
	// NodeSource 的 setup_<major>.x 只负责加仓库，安装必须显式指定精确版本，
	// 否则同一 commit 在不同日期会装出不同 Node。
	for _, rel := range []string{".gitlab-ci.yml"} {
		text := readRepoFile(t, rel)
		if strings.Contains(text, "deb.nodesource.com/setup_") && !nodesourceExactRe.MatchString(text) {
			t.Errorf("%s 用 NodeSource 安装 Node 但未 pin 精确版本（`nodejs=<X.Y.Z>-…`）", rel)
		}
	}
}

// TestPackageManagerIsPinnedToCIPnpm（#33）：apps/desktop 与 apps/web 的 packageManager
// 必须存在且等于两套 CI 实际使用的 pnpm 版本，否则本地（如 pnpm 11）会改写 lockfile 格式，
// CI 的 `--frozen-lockfile` 随即失败。
func TestPackageManagerIsPinnedToCIPnpm(t *testing.T) {
	// 两套 CI 都通过 pnpm/action-setup 的 `version:` / `corepack prepare pnpm@` 指定版本。
	// 行首锚定是必须的：`node-version: 24.21.0` 里也含子串 `version:`，不锚定会把它当成 pnpm 版本。
	pnpmVerRe := regexp.MustCompile(`(?m)^[ \t]*(?:-[ \t]+)?version:[ \t]*(\d+\.\d+\.\d+)[ \t]*$|pnpm@(\d+\.\d+\.\d+)`)
	var ciVersions []string
	for _, rel := range []string{
		filepath.Join(".github", "workflows", "ci.yml"),
		".gitlab-ci.yml",
	} {
		text := readRepoFile(t, rel)
		for _, m := range pnpmVerRe.FindAllStringSubmatch(text, -1) {
			v := m[1]
			if v == "" {
				v = m[2]
			}
			ciVersions = append(ciVersions, v)
		}
	}
	if len(ciVersions) == 0 {
		t.Fatal("未能从 CI 配置解析出 pnpm 版本（解析口径需同步）")
	}
	for _, v := range ciVersions[1:] {
		if v != ciVersions[0] {
			t.Errorf("两套 CI 的 pnpm 版本不一致：%q vs %q", ciVersions[0], v)
		}
	}
	want := "pnpm@" + ciVersions[0]

	for _, rel := range []string{
		filepath.Join("apps", "desktop", "package.json"),
		filepath.Join("apps", "web", "package.json"),
	} {
		var pkg struct {
			PackageManager string `json:"packageManager"`
		}
		if err := json.Unmarshal([]byte(readRepoFile(t, rel)), &pkg); err != nil {
			t.Fatalf("解析 %s: %v", rel, err)
		}
		if pkg.PackageManager == "" {
			t.Errorf("%s 缺少 packageManager 字段（#33：本地与 CI 的 pnpm 版本会漂移）", rel)
			continue
		}
		if pkg.PackageManager != want {
			t.Errorf("%s 的 packageManager=%q，与 CI 使用的 %q 不一致（--frozen-lockfile 会失败）", rel, pkg.PackageManager, want)
		}
	}
}

// TestDockerignoreExcludesBuildArtifacts（#34）：本地跑过覆盖率 / 缓存后，同一 commit
// 不得因此产出不同镜像层——所有构建/测试产物都必须进 .dockerignore。
func TestDockerignoreExcludesBuildArtifacts(t *testing.T) {
	text := readRepoFile(t, ".dockerignore")
	var lines []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		lines = append(lines, line)
	}
	for _, want := range []string{
		"apps/web/coverage",
		".pnpm-store",
		"test-results",
		".run/",
		".cargo/",
		"playwright-report",
		"blob-report",
		"coverage.out",
		"*.tsbuildinfo",
		"**/*.log",
		".gitlab-ci-local",
	} {
		found := false
		for _, line := range lines {
			if strings.Contains(line, want) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf(".dockerignore 缺少 %q：本地跑过覆盖率/缓存后同一 commit 会产出不同镜像层（#34）", want)
		}
	}
}

// TestGitHubWorkflowInjectsVersionBuildArg（R9）：GitHub 侧构建镜像时必须显式传 VERSION
// build-arg。此前不传，镜像内版本回落到 Dockerfile 的过期字面量，而 GitLab 传 VERSION=ci、
// Makefile 传真实版本——同一个 Dockerfile 三种行为。
func TestGitHubWorkflowInjectsVersionBuildArg(t *testing.T) {
	text := readRepoFile(t, filepath.Join(".github", "workflows", "ci.yml"))
	if !strings.Contains(text, "build-args:") || !strings.Contains(text, "VERSION=ci") {
		t.Error("ci.yml 构建镜像时未显式注入 VERSION build-arg（R9：版本会回落到 Dockerfile 字面量）")
	}
	// 扫描与推送必须用同一个 build-arg，否则推出去的镜像与扫描过的不是同一个版本。
	if n := strings.Count(text, "VERSION=ci"); n < 2 {
		t.Errorf("ci.yml 中 `VERSION=ci` 出现 %d 次：构建/扫描与推送两处都必须注入（R9）", n)
	}
}

// TestGitHubWorkflowPushesImage（R10）：必须有 workflow 推送镜像，否则「发布」只有 tag 与
// 桌面安装包，用户无法 docker pull 到 CI 构建的产物。
func TestGitHubWorkflowPushesImage(t *testing.T) {
	text := readRepoFile(t, filepath.Join(".github", "workflows", "ci.yml"))
	if !strings.Contains(text, "push: true") {
		t.Error("ci.yml 无任何 push: true——镜像只构建不发布（R10）")
	}
	if !strings.Contains(text, "packages: write") {
		t.Error("推送 GHCR 需要 packages: write 权限（R10）")
	}
	if !strings.Contains(text, "docker/login-action@") {
		t.Error("推送镜像前必须登录 registry（R10）")
	}
	// 发布 job 必须依赖扫描 job，避免把带漏洞的镜像推出去。
	if !strings.Contains(text, "needs: docker") {
		t.Error("publish job 必须 needs: docker，确保 Trivy 扫描通过后才推送（R10）")
	}
	// PR 不得推送镜像。
	if !strings.Contains(text, "github.event_name != 'pull_request'") {
		t.Error("publish job 必须排除 pull_request，避免 PR 污染 registry（R10）")
	}
}

// TestMakefileMirrorsCIGates（R7）：本地 make 必须能跑与 CI 等价的静态检查与覆盖率门禁，
// 否则「本地全绿」不代表 CI 会绿。
func TestMakefileMirrorsCIGates(t *testing.T) {
	text := readRepoFile(t, "Makefile")
	// ① 必须有 lint / govulncheck 目标（CI 有，此前本地无）。
	for _, target := range []string{"lint:", "govulncheck:"} {
		if !strings.Contains(text, target) {
			t.Errorf("Makefile 缺少 %q 目标：本地无法复现 CI 的该门禁（R7）", target)
		}
	}
	if !strings.Contains(text, "golangci-lint run") {
		t.Error("Makefile 的 lint 目标必须真的调用 golangci-lint（R7）")
	}
	if !strings.Contains(text, "govulncheck") {
		t.Error("Makefile 的 govulncheck 目标必须真的调用 govulncheck（R7）")
	}
	// ② test-all 必须含覆盖率门禁（此前只有 test + web-test，覆盖率完全没跑）。
	if !strings.Contains(text, "test-all: test-cover web-test-cover") {
		t.Error("test-all 必须包含 test-cover 与 web-test-cover（R7：此前无覆盖率门禁）")
	}
	// ③ 裸 `pnpm install` 会改写锁文件；必须用 --frozen-lockfile。
	if strings.Contains(text, "pnpm install &&") {
		t.Error("Makefile 使用了裸 `pnpm install`：会改写锁文件导致 CI 的 --frozen-lockfile 失败（R7）")
	}
	// ④ 所有定义的目标都必须进 .PHONY（test-cover / install-hooks 此前遗漏）。
	phony := phonyTargets(text)
	for _, target := range definedTargets(text) {
		if !phony[target] {
			t.Errorf("Makefile 目标 %q 不在 .PHONY 中（R7）", target)
		}
	}
}

// phonyTargets 解析 .PHONY 行声明的目标集合。
func phonyTargets(makefile string) map[string]bool {
	out := map[string]bool{}
	for _, line := range strings.Split(makefile, "\n") {
		if !strings.HasPrefix(line, ".PHONY:") {
			continue
		}
		for _, f := range strings.Fields(strings.TrimPrefix(line, ".PHONY:")) {
			out[f] = true
		}
	}
	return out
}

// definedTargets 解析 Makefile 中定义的目标名（行首 `name:`，排除变量赋值与 .PHONY）。
func definedTargets(makefile string) []string {
	var out []string
	for _, line := range strings.Split(makefile, "\n") {
		if line == "" || strings.HasPrefix(line, "\t") || strings.HasPrefix(line, "#") {
			continue
		}
		name, rest, ok := strings.Cut(line, ":")
		if !ok || strings.Contains(name, "=") || strings.Contains(name, " ") {
			continue
		}
		if strings.HasPrefix(name, ".") || name == "" {
			continue
		}
		// 排除 `A ?= B` 之类被 Cut 误判的情形。
		if strings.HasPrefix(rest, "=") {
			continue
		}
		out = append(out, name)
	}
	return out
}

// ---- 发布链（R2 / R3 / R4）----

// readRepoFile 读取仓库根下的文件。
func readRepoFile(t *testing.T, rel string) string {
	t.Helper()
	path := filepath.Join(repoRoot(t), rel)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取 %s: %v", path, err)
	}
	return string(data)
}

// TestReleaseWorkflowVerifiesTagMatchesManifest（R2）：发布 workflow 必须把解析出的 tag
// 与 tauri.conf.json / Cargo.toml 的版本比对，否则给 rc2 打 tag 会发布标着 rc1 的安装包。
func TestReleaseWorkflowVerifiesTagMatchesManifest(t *testing.T) {
	text := readRepoFile(t, filepath.Join(".github", "workflows", "release-desktop.yml"))
	for _, want := range []string{
		"tauri.conf.json", // 必须读取前端/桌面清单
		"Cargo.toml",      // 必须读取 Rust 清单
	} {
		if !strings.Contains(text, want) {
			t.Errorf("release-desktop.yml 未在 tag 校验中读取 %s（R2：tag 必须与清单版本一致）", want)
		}
	}
	// 必须真的做「比对 + 失败退出」，而不只是打印。
	if !strings.Contains(text, "!=") || !strings.Contains(text, "exit 1") {
		t.Error("release-desktop.yml 缺少 tag↔清单版本的实际比对/失败逻辑（R2）")
	}
}

// TestReleaseWorkflowChecksumsArePerPlatform（R3）：三平台矩阵不得各自 clobber 同一个
// SHA256SUMS.txt；必须平台内唯一命名，并由聚合 job 合并。
func TestReleaseWorkflowChecksumsArePerPlatform(t *testing.T) {
	text := readRepoFile(t, filepath.Join(".github", "workflows", "release-desktop.yml"))
	// 平台内唯一命名（SHA256SUMS-<bundle>.txt）。
	if !strings.Contains(text, "SHA256SUMS-") {
		t.Error("release-desktop.yml 未使用平台内唯一的校验清单名（R3：三平台并行 clobber 会互相覆盖）")
	}
	// 必须存在聚合 job 合并出唯一 SHA256SUMS.txt。
	if !strings.Contains(text, "aggregate-checksums") {
		t.Error("release-desktop.yml 缺少聚合校验清单的 job（R3）")
	}
	if !strings.Contains(text, "needs: publish") {
		t.Error("聚合 job 必须 needs: publish，否则会与三平台上传竞态（R3）")
	}
}

// TestTrivyScansUseCacheAndRetry（R4）：两套 CI 的 Trivy 都必须缓存 DB 且带重试，
// 否则 mirror.gcr.io 抖动会让安全门禁以网络失败的形式变红。
func TestTrivyScansUseCacheAndRetry(t *testing.T) {
	files := []string{
		filepath.Join(".github", "workflows", "ci.yml"),
		".gitlab-ci.yml",
	}
	for _, rel := range files {
		text := readRepoFile(t, rel)
		if !strings.Contains(text, "--download-db-only") {
			t.Errorf("%s 的 Trivy 未做 DB 预下载（R4：无法与扫描失败区分）", rel)
		}
		if !strings.Contains(text, "--skip-db-update") {
			t.Errorf("%s 的 Trivy 扫描未用 --skip-db-update 复用已就绪 DB（R4）", rel)
		}
		if !strings.Contains(text, "attempt") {
			t.Errorf("%s 的 Trivy DB 下载缺少重试循环（R4）", rel)
		}
	}
	// GitHub 侧必须显式缓存 DB 目录（GitLab 侧由 cache: paths 承担，见 .gitlab-ci.yml）。
	gh := readRepoFile(t, filepath.Join(".github", "workflows", "ci.yml"))
	if !strings.Contains(gh, "actions/cache@") || !strings.Contains(gh, ".trivy-cache") {
		t.Error("ci.yml 未缓存 Trivy DB 目录（R4）")
	}
}

// ---- 真实后端 + RustFS 浏览器联调（todolist #37）----

// rustfsImageRe 匹配 `rustfs/rustfs:<version>` 镜像引用。
// 版本部分只取 `[0-9A-Za-z.-]`，避免把脚本里 `${RUSTFS_IMAGE:-rustfs/rustfs:1.0.0-rc.3}` 的 `}` 吞进来。
var rustfsImageRe = regexp.MustCompile(`rustfs/rustfs:([0-9][0-9A-Za-z.-]*)`)

// TestRustFSImageIsConsistentlyPinned（#37）：RustFS 镜像版本在「脚本默认值 / compose /
// GitLab service」三处必须**完全一致**且带具体版本。
//
// 为什么需要门禁：RustFS 是这套联调唯一的真实 S3 对端，版本不一致意味着本地与 CI 验证的
// 不是同一个实现。GitLab 的 service 必须自带镜像引用（不能引用脚本变量），因此它是最容易
// 与脚本默认值漂移的一处。
//
// 说明：GitHub workflow **不再**引用镜像——它整体委托给 `scripts/e2e-real.sh`（由脚本起容器）。
// 若将来又出现直接引用，也必须与其余各处一致（本门禁会把任何出现的引用纳入比对）。
func TestRustFSImageIsConsistentlyPinned(t *testing.T) {
	// 必须自带引用的三处（GitHub 可选：委托给脚本后不再需要）。
	required := []string{
		filepath.Join("scripts", "e2e-real.sh"),
		"docker-compose.yml",
		".gitlab-ci.yml",
	}
	optional := []string{filepath.Join(".github", "workflows", "e2e-real.yml")}

	seen := map[string][]string{}
	collect := func(rel string, mustExist bool) {
		text := readRepoFile(t, rel)
		matches := rustfsImageRe.FindAllStringSubmatch(text, -1)
		if len(matches) == 0 {
			if mustExist {
				t.Errorf("%s 未找到 `rustfs/rustfs:<version>` 引用（#37：真实对端镜像需 pin）", rel)
			}
			return
		}
		for _, m := range matches {
			seen[m[1]] = append(seen[m[1]], rel)
		}
	}
	for _, rel := range required {
		collect(rel, true)
	}
	for _, rel := range optional {
		collect(rel, false)
	}

	if len(seen) > 1 {
		t.Errorf("RustFS 镜像版本不一致：%v（脚本默认值 / compose / GitLab service 必须同版本，#37）", seen)
	}
	// 不允许浮动 tag：`latest` 会让同一 commit 在不同时间拉出不同对端。
	for version := range seen {
		if version == "latest" {
			t.Errorf("RustFS 使用了浮动 tag `latest`（%v）：E2E 会因上游漂移而 flaky", seen[version])
		}
	}
}

// TestRealE2EUsesSharedScript（#37）：本地与两套 CI 必须跑**同一段编排**（scripts/e2e-real.sh），
// 而不是各抄一份「起 RustFS + 构建 + 起后端 + 跑用例」。
//
// 为什么需要门禁：此前三处各写一份编排，改一处漏两处就会出现「本地跑通但 CI 跑不通」
// （或反之）。收敛到共享脚本后，本门禁钉住这个结构，防止将来有人图省事又复制一份。
func TestRealE2EUsesSharedScript(t *testing.T) {
	for _, rel := range []string{
		filepath.Join(".github", "workflows", "e2e-real.yml"),
		".gitlab-ci.yml",
	} {
		text := readRepoFile(t, rel)
		// 必须**实际调用**（`bash scripts/e2e-real.sh`），而不是只在 `changes:` 路径过滤里
		// 提到该文件——后者曾被本门禁误判为「已复用脚本」（变异验证发现）。
		if !strings.Contains(text, "bash scripts/e2e-real.sh") {
			t.Errorf("%s 未实际调用 `bash scripts/e2e-real.sh`（#37：编排必须单一来源，仅出现在 paths/changes 里不算）", rel)
		}
	}
	// 脚本必须是可执行且语法合法的 bash（`bash -n` 由开发者在 CI 跑；这里只校验关键要素）。
	script := readRepoFile(t, filepath.Join("scripts", "e2e-real.sh"))
	for _, want := range []string{
		"docker run",                  // 自动起 RustFS（默认分支）
		"RUSTFS_CORS_ALLOWED_ORIGINS", // 浏览器直传所需 CORS
		"pnpm build",                  // 真实构建产物
		"S3C_STATIC_DIR",              // 后端托管产物
		"e2e:real",                    // 真实联调入口
		"trap cleanup EXIT",           // 跑完自动清理
		"--no-rustfs",                 // 复用外部对端（GitLab service 用）
	} {
		if !strings.Contains(script, want) {
			t.Errorf("scripts/e2e-real.sh 缺少 %q（#37：共享编排须自包含、可复用外部对端、可清理）", want)
		}
	}
	// 脚本不能写死单一后端地址：CI 用 service 别名，本地用回环。
	if strings.Contains(script, "S3CLINET_ENDPOINT=http://127.0.0.1:9000") {
		t.Error("scripts/e2e-real.sh 写死了 S3CLINET_ENDPOINT：应经 RUSTFS_ENDPOINT 变量支持外部对端（#37）")
	}
}

// TestRealE2EArtifactsExist（#37）：联调用例、专用 config、npm 入口必须齐备，
// 且用例**不得** mock `/api`（否则退化成第二个 playwright-e2e）。
func TestRealE2EArtifactsExist(t *testing.T) {
	if _, err := os.Stat(filepath.Join(repoRoot(t), "apps", "web", "e2e-real", "real-backend.spec.ts")); err != nil {
		t.Errorf("缺少 apps/web/e2e-real/real-backend.spec.ts（#37 的联调用例本体）：%v", err)
	}
	if _, err := os.Stat(filepath.Join(repoRoot(t), "apps", "web", "playwright.real.config.ts")); err != nil {
		t.Errorf("缺少 apps/web/playwright.real.config.ts（#37 专用配置）：%v", err)
	}
	pkg := readRepoFile(t, filepath.Join("apps", "web", "package.json"))
	if !strings.Contains(pkg, `"e2e:real"`) || !strings.Contains(pkg, "playwright.real.config.ts") {
		t.Error("apps/web/package.json 的 e2e:real 未指向 playwright.real.config.ts（#37）")
	}
	// 联调用例**不得** mock /api：出现 page.route 调用即说明它退化成第二个 playwright-e2e。
	// 逐行扫描并跳过注释行——文件头的说明性注释里会**提到** `page.route()`（解释为何不用它），
	// 直接对全文做子串匹配会把这段注释误判为真实调用。
	spec := readRepoFile(t, filepath.Join("apps", "web", "e2e-real", "real-backend.spec.ts"))
	for _, line := range strings.Split(spec, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "*") || strings.HasPrefix(trimmed, "/*") {
			continue
		}
		if strings.Contains(trimmed, "page.route(") {
			t.Error("e2e-real/real-backend.spec.ts 使用了 page.route：真实联调不得 mock /api（#37）")
			break
		}
	}
}

// TestE2ESourcesAreTypechecked（#37）：两套 CI 与本地 `make check` 都必须对 E2E 源码
// （`e2e/` 与 `e2e-real/`）做类型检查。
//
// 为什么需要门禁：主 `tsconfig.json` 只 include `src/**`，两套 E2E 长期**零静态检查**——
// 而 `e2e-real` 是 #37 的核心门禁，其中的用例写错类型（如误用 APIRequestContext 的返回）
// 只会在运行时才炸。本门禁钉住「E2E 有独立 tsconfig 且两侧 CI 都跑它」。
func TestE2ESourcesAreTypechecked(t *testing.T) {
	// 独立配置必须存在，且 include 两套 E2E 目录与 playwright 配置。
	cfg := readRepoFile(t, filepath.Join("apps", "web", "tsconfig.e2e.json"))
	for _, want := range []string{"e2e/**/*.ts", "e2e-real/**/*.ts", "playwright.real.config.ts"} {
		if !strings.Contains(cfg, want) {
			t.Errorf("apps/web/tsconfig.e2e.json 未 include %q（#37：E2E 源码需类型检查）", want)
		}
	}
	// package.json 必须有入口。
	pkg := readRepoFile(t, filepath.Join("apps", "web", "package.json"))
	if !strings.Contains(pkg, `"typecheck:e2e"`) || !strings.Contains(pkg, "tsconfig.e2e.json") {
		t.Error("apps/web/package.json 缺少指向 tsconfig.e2e.json 的 typecheck:e2e 脚本（#37）")
	}
	// 两套 CI 的 web job 都要跑它，否则 CI 仍看不见 E2E 类型错误。
	for _, rel := range []string{
		filepath.Join(".github", "workflows", "ci.yml"),
		".gitlab-ci.yml",
	} {
		if !strings.Contains(readRepoFile(t, rel), "typecheck:e2e") {
			t.Errorf("%s 的 web job 未跑 `pnpm typecheck:e2e`（#37：E2E 类型错误会在 CI 漏网）", rel)
		}
	}
	// 本地 check 聚合也要含它，保证「本地绿 ≈ CI 绿」。
	if !strings.Contains(readRepoFile(t, "Makefile"), "web-typecheck-e2e") {
		t.Error("Makefile 的 check 未包含 web-typecheck-e2e（#37：本地无法复现 CI 的 E2E 类型门禁）")
	}
}

// TestLocalRealE2ETargetExists（#37）：本地必须能一条命令起「真实 RustFS + 真实产物 +
// 真实后端」跑通联调，否则「本地测试也用 docker 自动运行一份 RustFS」的诉求落空。
func TestLocalRealE2ETargetExists(t *testing.T) {
	mk := readRepoFile(t, "Makefile")
	if !strings.Contains(mk, "e2e-real:") {
		t.Error("Makefile 缺少 e2e-real 目标（#37：本地一键真实联调）")
	}
	if !strings.Contains(mk, "scripts/e2e-real.sh") {
		t.Error("Makefile 的 e2e-real 必须调用 scripts/e2e-real.sh（#37）")
	}
}
