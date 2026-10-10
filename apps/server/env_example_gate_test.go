package main

// env_example_gate_test.go —— 根 `.env.example` ⇔ docker compose 透传面的**双向**一致性门禁。
//
// 背景（2026-10-10）：根 `.env.example` 自称「只列 compose 实际透传的变量」
// （见 docs/CONFIGURATION.md §1「两份 .env.example 的口径分工」），但实测两个方向都漂移：
//   - **少列**：5 个 compose 真正插值的键（S3C_SHUTDOWN_TIMEOUT / S3C_REGION /
//     S3C_CORS_ORIGINS / S3C_LOG_LEVEL / S3C_IMAGE_TAG）与两个构建参数
//     （GOPROXY / NPM_REGISTRY）都不在模板里——照 `cp .env.example .env` 的用户
//     无法发现这些可覆盖项；
//   - **多列**：3 个 compose **并不透传**的服务端加固项（S3C_ALLOW_PLAINTEXT_STORE /
//     S3C_TRUSTED_PROXIES / S3C_SSRF_DENY_PRIVATE）——compose 的 `environment:` 是
//     显式白名单且无 `env_file:`，在根 `.env` 里取消注释这些行不会生效。
//
// 断言（两个方向）：
//  1. compose 三个文件里每个 `${VAR}` 插值的键，根 `.env.example` 必须有对应赋值行
//     （注释掉的 `# KEY=` 也算——模板允许保留 TODO 形态）；
//  2. 反向：根 `.env.example` 里每个赋值行 `KEY=` 都必须是 compose 插值过的键；
//  3. 两份模板都**不示范**「值后行内注释」——`#` 只在行首才算注释（见下文
//     TestEnvExamplesAvoidInlineComments）。
//
// 口径说明：只认「赋值行」（`^\s*#?\s*KEY=`）。不含 `=` 的散文说明不参与，因此根模板可以
// 用散文指向 `apps/server/.env.example` 而不被误判为声明。未做严格的是：不校验默认值 /
// 语义描述是否一致（自然语言，属人工审查范围，与 config_doc_gate_test.go 同口径）。

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// composeInterpRe 匹配 compose 文件里的 `${VAR}` / `${VAR:-default}` / `${VAR:?msg}` 插值。
var composeInterpRe = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)`)

// envExampleAssignRe 匹配 `.env.example` 的赋值行；`#` 注释前缀可选（保留 TODO 形态）。
var envExampleAssignRe = regexp.MustCompile(`(?m)^[ \t]*#?[ \t]*([A-Za-z_][A-Za-z0-9_]*)=`)

// composeInterpolatedKeys 汇总三个 compose 文件里全部 `${VAR}` 插值的键（去重升序）。
func composeInterpolatedKeys(t *testing.T) []string {
	t.Helper()
	seen := map[string]bool{}
	for _, name := range []string{"docker-compose.yml", "docker-compose.prod.yml", "docker-compose.tls.yml"} {
		p := filepath.Join(repoRoot(t), name)
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("读取 %s: %v", p, err)
		}
		for _, m := range composeInterpRe.FindAllStringSubmatch(string(b), -1) {
			seen[m[1]] = true
		}
	}
	// 抽不到任何插值说明解析口径失效（如 compose 改用 env_file:），门禁会「全绿但失明」，
	// 必须红灯提示同步口径，而不是安静通过。
	if len(seen) == 0 {
		t.Fatal("未从 compose 抽到任何 ${VAR} 插值，解析口径需同步")
	}
	return sortedKeys(seen)
}

// envExampleAssignedKeys 汇总指定 `.env.example` 的赋值行键（去重升序）。
func envExampleAssignedKeys(t *testing.T, rel string) []string {
	t.Helper()
	p := filepath.Join(repoRoot(t), rel)
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("读取 %s: %v", p, err)
	}
	seen := map[string]bool{}
	for _, m := range envExampleAssignRe.FindAllStringSubmatch(string(b), -1) {
		seen[m[1]] = true
	}
	if len(seen) == 0 {
		t.Fatalf("%s 未解析到任何赋值行，解析口径需同步", rel)
	}
	return sortedKeys(seen)
}

// TestRootEnvExampleCoversEveryComposeKey 断言 compose 里每个 `${VAR}` 都在根模板有落脚点。
func TestRootEnvExampleCoversEveryComposeKey(t *testing.T) {
	t.Parallel()
	have := map[string]bool{}
	for _, k := range envExampleAssignedKeys(t, ".env.example") {
		have[k] = true
	}
	var missing []string
	for _, k := range composeInterpolatedKeys(t) {
		if !have[k] {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		t.Errorf("根 .env.example 漏列 %d 个 compose 实际插值的键：%s\n"+
			"照 `cp .env.example .env` 的用户将无法发现这些可覆盖项（docs/CONFIGURATION.md §1 口径）",
			len(missing), strings.Join(missing, ", "))
	}
}

// TestRootEnvExampleListsOnlyComposeKeys 断言根模板不列 compose 不透传的键。
func TestRootEnvExampleListsOnlyComposeKeys(t *testing.T) {
	t.Parallel()
	compose := map[string]bool{}
	for _, k := range composeInterpolatedKeys(t) {
		compose[k] = true
	}
	var extra []string
	for _, k := range envExampleAssignedKeys(t, ".env.example") {
		if !compose[k] {
			extra = append(extra, k)
		}
	}
	if len(extra) > 0 {
		t.Errorf("根 .env.example 列了 %d 个 compose **不透传**的键：%s\n"+
			"compose 的 environment: 是显式白名单且无 env_file:，在根 .env 里设置它们不会生效；"+
			"服务端专属可选项请只放 apps/server/.env.example",
			len(extra), strings.Join(extra, ", "))
	}
}

// envInlineCommentRe 匹配「值后面跟行内注释」的陷阱行：`KEY=value # 说明`。
var envInlineCommentRe = regexp.MustCompile(`^[ \t]*[A-Za-z_][A-Za-z0-9_]*=[^\n]*[ \t]#`)

// TestEnvExamplesAvoidInlineComments 断言两份模板不示范「值后行内注释」。
//
// `.env` 解析器只在**行首**识别 `#`（`internal/config/loadDotEnvFile`：`strings.TrimSpace(line)`
// 后判断 `strings.HasPrefix(line, "#")`），行内 `#` 会被切进值里——`S3C_LOG_LEVEL=info # 说明`
// 实际得到 `info # 说明`。模板是用户抄写的样本，示范这种写法会把「合法默认」变成非法值
// （`S3C_LOG_LEVEL` 会被判非法 / `S3C_TOKEN` 会带着注释一起当口令），故机械钉住。
func TestEnvExamplesAvoidInlineComments(t *testing.T) {
	t.Parallel()
	for _, rel := range []string{".env.example", "apps/server/.env.example"} {
		rel := rel
		t.Run(rel, func(t *testing.T) {
			t.Parallel()
			p := filepath.Join(repoRoot(t), rel)
			b, err := os.ReadFile(p)
			if err != nil {
				t.Fatalf("读取 %s: %v", p, err)
			}
			for i, line := range strings.Split(string(b), "\n") {
				if strings.HasPrefix(strings.TrimSpace(line), "#") {
					continue // 整行注释（`#` 在行首）是合法写法
				}
				if envInlineCommentRe.MatchString(line) {
					t.Errorf("%s:%d 值后带行内注释（会被当作值的一部分）：%s\n"+
						"`#` 仅在行首才是注释，说明请单独成行", rel, i+1, strings.TrimSpace(line))
				}
			}
		})
	}
}
