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
