package main

// dev_scripts_gate_test.go —— 本地开发脚本「受管 web dev server 的 PID 必须指向 vite 本身」门禁。
//
// 背景（2026-10-10 实测，`make dev` 暴露）：`run-dev.sh` / `graceful-restart.sh` 曾用
// `pnpm dev &` 再 `write_pid web $!`。`$!` 是 **pnpm 包装进程**，实际链为
// `pnpm.mjs → pnpm.cjs → vite.js` 三层；而 `process.sh` 的 `expected_process_pattern web`
// 只认关键字 `vite`。于是 `validated_pid` 把「存活且确实是本次启动的 vite」误判为
// 「PID 已被复用」，**拒绝停止**：旧 vite 继续占用 1949，新 vite 静默退到 1950，
// `wait_http 1949` 又因旧实例在响应而通过 → `make dev` 报「已就绪」但实际跑在 1950，
// 且每次 `make dev` 泄漏一个 vite 进程（实测 `ps` 里两条 vite 链并存）。
//
// 修法：直接启动 `./node_modules/.bin/vite`——该 shim（pnpm 生成）末行是
// `exec node …/vite/bin/vite.js`，`exec` 不换 PID，故 `$!` 就是 vite 进程、命令行含 `vite`，
// 与校验口径一致（信号直达 vite，不再依赖包装进程转发）；再配 `--strictPort`，
// 把「端口被占 → 静默换端口」变成显式失败，杜绝「已就绪但不在 1949」的假绿。
//
// 断言（任一回退即红灯）：
//  1. `process.sh` 对 web 的预期进程名仍是 `vite`；
//  2. `apps/web` 的 `dev` 脚本就是裸 `vite`（直接启动与 `pnpm dev` 等价的前提）；
//  3. 两个受管脚本都以 `./node_modules/.bin/vite` 启动 web，且带 `--strictPort`；
//  4. 两个受管脚本都不再用 `pnpm dev`（防回退到多层包装进程）。

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"
)

// managedDevScripts 是通过 `.run/*.pid` 托管本地进程、且被 `make dev` / 优雅重启触达的脚本
// （新增受管入口时在此登记）。
var managedDevScripts = []string{"scripts/run-dev.sh", "scripts/graceful-restart.sh"}

// webPatternRe 匹配 process.sh 里 `web)` 分支返回的预期进程名。
var webPatternRe = regexp.MustCompile(`(?m)^\s*web\)\s*echo\s+"vite"`)

// pnpmDevRe 匹配脚本里以 `pnpm dev` / `pnpm run dev` 启动进程的行。
var pnpmDevRe = regexp.MustCompile(`(?m)^\s*pnpm (run )?dev\b`)

// goModTidyRe 匹配脚本里**真正执行** `go mod tidy` 的行（`#` 注释行不匹配——注释不执行）。
var goModTidyRe = regexp.MustCompile(`(?m)^\s*go mod tidy\b`)

// TestManagedWebServerRecordsVitePid 钉住「.run/web.pid 记录的就是 vite 进程」这一不变量。
func TestManagedWebServerRecordsVitePid(t *testing.T) {
	t.Parallel()

	// 1) 校验口径本身没被改掉。
	if !webPatternRe.MatchString(readRepoFile(t, "scripts/lib/process.sh")) {
		t.Errorf("scripts/lib/process.sh 的 `expected_process_pattern web` 不再是 \"vite\"；" +
			"改口径必须同步本门禁与「被记录的启动方式」，否则会重新出现「误判 PID 复用 → 拒绝停止」")
	}

	// 2) 直接启动 .bin/vite 与 `pnpm dev` 等价的前提：dev 脚本就是裸 `vite`。
	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if err := json.Unmarshal([]byte(readRepoFile(t, "apps/web/package.json")), &pkg); err != nil {
		t.Fatalf("解析 apps/web/package.json: %v", err)
	}
	if got := strings.TrimSpace(pkg.Scripts["dev"]); got != "vite" {
		t.Errorf("apps/web 的 scripts.dev = %q，期望裸 \"vite\"；直接启动 .bin/vite 依赖二者等价，"+
			"带额外参数/前置步骤时必须改回用 pnpm 并同步 process.sh 的校验口径", got)
	}

	// 3)+4) 受管脚本的 web 启动方式。
	for _, rel := range managedDevScripts {
		body := readRepoFile(t, rel)
		if !strings.Contains(body, "./node_modules/.bin/vite") {
			t.Errorf("%s 未用 `./node_modules/.bin/vite` 启动 web：用包装进程（如 `pnpm dev`）会让 "+
				".run/web.pid 记录的 PID 命令行不含 \"vite\"，validated_pid 将误判「PID 已被复用」而拒绝停止，"+
				"导致旧实例占着 1949、新实例静默退到 1950", rel)
		}
		if !strings.Contains(body, "--strictPort") {
			t.Errorf("%s 启动 web 未带 --strictPort：端口被占时 vite 会静默换端口，"+
				"`make dev` 仍报「已就绪」但实际不在 1949", rel)
		}
		if pnpmDevRe.MatchString(body) {
			t.Errorf("%s 仍在用 `pnpm dev` 启动受管 web（回退到多层包装进程，见文件头背景）", rel)
		}
	}
}

// TestDevScriptsLeaveDependencySyncToExplicitTargets 钉住「受管开发脚本不在启动流程里自动同步依赖」。
//
// `go mod tidy` 会改写 go.mod/go.sum，而仓库对它的定位是**显式**动作：Makefile `server` 目标
// 的注释写明「不在每次启动时跑 `go mod tidy`，避免依赖被无意识升级导致开发与 CI 漂移」。
// 2026-10-10 实测：`make dev` 每次都会删除 `apps/server/go.sum` 的 6 行（`run-dev.sh` 的无条件
// tidy），提交文件被开发动作悄悄改写。故启动路径一律不 tidy——`go build` 默认 `-mod=readonly`，
// 依赖不一致会**报错**（loud failure）而不是静默改写；显式同步走 `make tidy`。
func TestDevScriptsLeaveDependencySyncToExplicitTargets(t *testing.T) {
	t.Parallel()
	for _, rel := range managedDevScripts {
		if goModTidyRe.MatchString(readRepoFile(t, rel)) {
			t.Errorf("%s 在启动流程里执行 `go mod tidy`：会静默改写 go.mod/go.sum（实测 `make dev` 每次删 go.sum 6 行），"+
				"与 Makefile `server` 目标「不在每次启动时跑 go mod tidy」的口径相悖；依赖同步请留给显式入口 `make tidy`", rel)
		}
	}
	if !regexp.MustCompile(`(?m)^tidy:`).MatchString(readRepoFile(t, "Makefile")) {
		t.Errorf("Makefile 缺少显式 `tidy` 目标：移除自动 tidy 后必须保留依赖同步的显式入口")
	}
}
