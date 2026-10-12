# apps/desktop 优化评审快照（2026-10-12）

> **评审时点**：2026-10-12 ｜ **评审对象**：[`apps/desktop`](../../apps/desktop/)（Tauri 2 桌面壳，B/S 架构、无 IPC，见 [ADR-001](../decisions/0001-desktop-no-ipc.md)）
> **结论**：1 项死代码（技术债）+ 6 项优化 + 1 道防复发门禁，全部在本批次（2026-10-12）内闭环。
> **性质**：时点性**子树评审快照**——结论绑定评审日期；条目全部闭环后按 [archive/index.md](index.md) 的「归档四步」`git mv` 进 `docs/archive/` 冻结（命名先例 `code-review-2026-09-24.md` / `code-review-2026-10-09.md` / `code-review-2026-10-10.md`，三者均已归档）。
> **证据口径**：本文件所有 `file:line` 均为评审当天实读所得；§3 的命令与输出为实跑原文（含失败/无结论的项，不代偿、不美化）。

> **归档冻结（2026-10-12）**：本快照条目（D1 / O1–O6 / G1）**全部闭环**，已按「归档四步」`git mv` 冻结入 `docs/archive/`。**落地实况与评审拟稿的差异**（正文 §2.5 保留为落地前的文本真值，按归档纪律不回写）：
> - **O6 依赖升级已落地**：`tauri` `2.11.5 → 2.12.3`、`tauri-build` `2.6.3 → 2.7.2`、`@tauri-apps/cli` `2.11.4 → 2.12.2`（`apps/desktop/package.json` 范围同步收紧为 `^2.12.2`）。故 **O3 的 CI pin 直接钉到 `2.12.2`**（非拟稿写的 `2.11.4`——拟稿先于 D1 / O6 落地，钉旧版会与锁文件分叉）。
> - **O6 连带：`cargo audit` 告警 7 条 → 2 条**。tauri 2.12 移除了 `unic-*` 传递依赖，`unmaintained / unsound` 告警只剩 `proc-macro-error` + `glib 0.18.5`；[`threat-model.md`](../threat-model.md) §5 与 [`ROADMAP.md`](../ROADMAP.md) 的「7 条」同批更新为「2 条」，`THIRD_PARTY_LICENSES.md` 已重生成（Rust crates **428 → 416**）。
> - **O5 的 CSP 收紧待人类确认，且 `form-action` 取值按门禁调整**：拟稿写 `frame-ancestors 'none'` / `form-action 'none'`；落地时 [`TestServerCSPSharesTauriDirectives`](../../apps/server/internal/handler/review_20260924_test.go)（server CSP ⇄ 壳 CSP **共享指令必须一致**）报漂移——server 侧 `form-action 'self'`，壳侧改 `'none'` 即红线。**共享指令不得单侧收紧**，而把 server 一并改 `'none'` 属服务端安全变更（另需人类复核，且前端 `SchedulesSection.vue` 确有 `<form @submit.prevent>` 面）。故壳侧落地为 **`frame-ancestors 'none'`（非共享，新增）+ `form-action 'self'`（与 server 对齐，较原先「无该指令」仍是收紧——`form-action` 不回退到 `default-src`）**。CSP 变更本身按 [`AI_POLICY.md`](../AI_POLICY.md) §3 须人类专家复核，**推送 / 合并前须人类专家确认**（其余 O5 子项——元数据与窗口 `label`——为纯增量，无需确认）。


## 1. 结论摘要

| # | 项 | 类型 | 证据位置 | 状态 |
|---|---|---|---|---|
| D1 | `serde` / `serde_json` 是**未被引用的直接依赖**（死代码，违反根 [`AGENTS.md`](../../AGENTS.md) 硬约束 5） | 技术债 | [`Cargo.toml`](../../apps/desktop/src-tauri/Cargo.toml) 第 13–14 行 vs [`main.rs`](../../apps/desktop/src-tauri/src/main.rs) 全文 | ✅ 已闭环 |
| O1 | 缺 `[profile.release]`：发布产物未做体积优化（默认 `lto=false` / `codegen-units=16` / 未 strip） | 优化（分发体积） | [`Cargo.toml`](../../apps/desktop/src-tauri/Cargo.toml) 无 `[profile.release]` | ✅ 已闭环 |
| O2 | `bundle.targets: "all"` 超出实际发布面（Linux 会连带 rpm / AppImage，而 AppImage 打包**需联网**下载 linuxdeploy） | 优化（构建可靠性） | [`tauri.conf.json`](../../apps/desktop/src-tauri/tauri.conf.json) 第 30 行 | ✅ 已闭环 |
| O3 | 两套 CI 的 `cargo install tauri-cli --version "^2"` 浮动（本地是锁文件钉住的 `2.11.4`） | 优化（可复现性，ROADMAP E9） | [`.github/workflows/ci.yml`](../../.github/workflows/ci.yml) 第 420 行 · [`.gitlab-ci.yml`](../../.gitlab-ci.yml) 第 437 行 | ✅ 已闭环 |
| O4 | `make desktop-dev` 不装 web 依赖，而 `beforeDevCommand` 会执行 `cd ../web && pnpm dev` → 全新检出必失败 | 优化（DX） | [`Makefile`](../../Makefile) 第 72 行 · `tauri.conf.json` 的 `beforeDevCommand` | ✅ 已闭环 |
| O5 | 打包元数据（`publisher` / `copyright` / `category`）缺失；窗口未显式 `label: "main"`；WebView CSP 缺 `frame-ancestors` / `form-action` | 优化（元数据 / 加固） | [`tauri.conf.json`](../../apps/desktop/src-tauri/tauri.conf.json) 第 14–31 行 | ✅ 已闭环 |
| O6 | Tauri 依赖停在 `2.11.5`（最新 `2.12.3`）、CLI `2.11.4`（最新 `2.12.2`） | 优化（依赖新鲜度） | `Cargo.lock` · [`pnpm-lock.yaml`](../../apps/desktop/pnpm-lock.yaml) | ✅ 已闭环 |
| G1 | 为 D1 补一道机械门禁：`cargo check` 加 `-D unused_crate_dependencies` | 门禁（防复发） | `ci.yml` / `.gitlab-ci.yml` 的 desktop job | ✅ 已闭环 |

## 2. 逐项发现

> 逐项的**拟改动内容**（可照抄落地）见 §2.5；**本批次不纳入的候选优化**清单见 §5。

### D1 死依赖：`serde` / `serde_json`

- **现象**：[`Cargo.toml`](../../apps/desktop/src-tauri/Cargo.toml) 声明了 `serde`（`features = ["derive"]`）与 `serde_json`，但 [`main.rs`](../../apps/desktop/src-tauri/src/main.rs) 只有 `tauri::Builder::default().run(tauri::generate_context!())`——**没有任何 `#[tauri::command]`、没有结构体派生、没有 JSON 解析**。全仓 grep 也确认只有该文件提到这两个 crate。
- **来由**：Tauri 官方模板带这两个依赖，是因为**模板自身**有示例 command（如 `greet`）需要序列化；本项目按 [ADR-001](../decisions/0001-desktop-no-ipc.md) 刻意不做 IPC，模板依赖被原样留下。
- **判定**：属根 [`AGENTS.md`](../../AGENTS.md) 硬约束 5「死代码零容忍」——「定义了却未使用」。它不改变行为（`serde` 仍作为 `tauri` 的传递依赖存在于 `Cargo.lock`，只是**本项目不再直接依赖**），但会虚增「直接依赖」面与审计口径。
- **拟改动**：删除两行；`cargo check --offline` 重写 `Cargo.lock`（仅少 root package 的两条直接依赖边，包集合不变，故 [`THIRD_PARTY_LICENSES.md`](../THIRD_PARTY_LICENSES.md) **无需**重生成——已核对：`serde` / `serde_json` 仍以传递依赖身份出现在清单里）。

### O1 缺 `[profile.release]`：安装包未做体积优化

- **现象**：`Cargo.toml` 全文没有 `[profile.release]`，发布构建走 Rust 默认档（`opt-level=3`、`lto=false`、`codegen-units=16`、保留调试符号）。
- **依据**：上游 [`tauri-docs` v2 `concept/size`](https://github.com/tauri-apps/tauri-docs/blob/v2/src/content/docs/concept/size.mdx) 明确推荐 `codegen-units = 1` / `lto = true` / `opt-level = "s"` / `panic = "abort"` / `strip = true`；桌面产物（`.exe` / `.deb` / `.dmg`）是**用户要下载**的文件，体积是用户可见指标。
- **判定**：纯优化，无行为变化（本壳无 IPC、无 `catch_unwind` 依赖，`panic = "abort"` 语义差异为零）。代价是 release 链接变慢——发布 job 超时预算 90 分钟，余量充足。
- **拟改动**：按上游推荐新增 `[profile.release]` 五键。

### O2 `bundle.targets: "all"` 超出实际发布面

- **现象**：`targets` 取 `"all"`，而实际发布只做 Windows NSIS `.exe` / Linux `.deb` / macOS `.dmg`（[`release-desktop.yml`](../../.github/workflows/release-desktop.yml) 用 `--bundles` 精确指定）。
- **依据**：Linux 侧 `all` 会连带 rpm 与 AppImage；**AppImage 打包在构建期要联网下载** `linuxdeploy` 与 `AppRun`（上游 `crates/tauri-bundler/src/bundle/linux/appimage/linuxdeploy.rs` 的 `download()`，缓存于 `~/.cache/tauri`）。即本地 `make desktop-build` 在离线/受限网络下会失败，而收益为零。
- **判定**：收窄到与实际发布面一致，并保留各平台合法取值。
- **拟改动**：`targets` 改为显式列表（`nsis` / `deb` / `dmg`）。

### O3 CI 的 `tauri-cli` 版本浮动

- **现象**：`ci.yml`（手动触发的 `desktop-build`）与 `.gitlab-ci.yml` 都执行 `cargo install tauri-cli --version "^2" --locked`——解析到**当时最新的 2.x**，而本地用的是锁文件钉住的 `@tauri-apps/cli 2.11.4`。
- **判定**：与 ROADMAP E9「CI 与本地同命令、精确 pin」的口径有落差（同一 commit 在不同日期可能装出不同 CLI）。
- **不是问题的一侧**：发布路径不受影响——`release-desktop.yml` 用 `tauri-apps/tauri-action`，它检测到项目内 `@tauri-apps/cli` 后走 `pnpm tauri`（即锁文件版本），已核对上游 `dist/index.js` 的 `getRunner()`。
- **拟改动**：两处 pin 到 `2.11.4`（与本地锁文件一致）。

### O4 `make desktop-dev` 缺 web 依赖

- **现象**：目标只 `cd apps/desktop && pnpm install --frozen-lockfile && pnpm tauri dev`；而 Tauri 的 `beforeDevCommand` 是 `cd ../web && pnpm dev`，`apps/web/node_modules` 不存在时 `vite` 直接 `not found`。
- **判定**：全新检出执行 `make desktop-dev` 必失败（对比：`make desktop-build` 已先构建 web，口径不一致）。
- **拟改动**：目标内先 `cd apps/web && pnpm install --frozen-lockfile`。

### O5 打包元数据 / 窗口 label / WebView CSP

- **元数据**：`bundle` 段只有 `active` / `targets` / `icon`，缺 `publisher` / `copyright` / `category`（Linux `.deb` 控制字段与 Windows 文件属性会留空）。
- **窗口 label**：[`capabilities/default.json`](../../apps/desktop/src-tauri/capabilities/default.json) 的 `windows: ["main"]` 依赖 Tauri 的默认窗口 label，`tauri.conf.json` 未显式写 `label`——显式声明可避免将来改窗口配置时 capability **静默脱离**。
- **CSP**：当前值已收紧了 `script-src 'self'` / `object-src 'none'` / `base-uri 'self'`；缺 `frame-ancestors 'none'` 与 `form-action 'none'`（壳内既不需要被嵌套，也没有表单提交面）。
- **判定**：均为纯增量加固 / 元数据补全，不改行为。

### O6 依赖新鲜度

- **现象**（`pnpm tauri info` 实跑，见 §3）：`tauri` 2.11.5（最新 2.12.3）、`tauri-build` 2.6.3（2.7.2）、`wry` 0.55.1（0.57.0）、`tao` 0.35.3（0.37.1）、CLI 2.11.4（2.12.2）。
- **判定**：属升级议题而非缺陷；新版本通常先承载修复。升级须连带重生成 [`THIRD_PARTY_LICENSES.md`](../THIRD_PARTY_LICENSES.md) 并复跑 `cargo audit`。
- **兜底口径**：若升级触发不可接受的风险面（新 advisory / 需要改配置 schema），本项**单独回退**为「登记待办」，不阻塞 D1 / O1–O5 的闭环。

### G1 防复发门禁

- **动机**：Go 侧有 `golangci-lint`（`unused` / `staticcheck`）与 [`deadcode_gate_test.go`](../../apps/server/deadcode_gate_test.go)，TS 侧有 `pnpm lint` + `vue-tsc`；Rust 侧此前**没有任何**「未使用符号 / 未使用依赖」的机械检查（`rust-toolchain.toml` 有意不装 clippy 组件）。D1 因此能长期存活。
- **做法**：不引入新工具、不装新组件——`cargo check` 加 `RUSTFLAGS=-D unused_crate_dependencies`（编译器自带 lint），两套 CI 的 desktop job 同步。
- **代价**：该 lint 对**直接依赖**生效（正是 D1 的形态），保留 `tauri` 这个真实使用项不受影响。

### 2.5 拟改动内容（逐项，落地即按此执行）

> 以下为**文本真值**：落地时代码取此处内容；若落地时发现与代码实际不符，**先改本文档再改代码**（防两处漂移）。

**D1 删除死依赖**（`apps/desktop/src-tauri/Cargo.toml`）：

```diff
 [dependencies]
 tauri = { version = "2", features = [] }
-serde = { version = "1", features = ["derive"] }
-serde_json = "1"
```

落库后执行 `cargo check --offline`（重写 `Cargo.lock`：仅少 root package 的两条直接依赖边），随后 `cargo check --locked` 必须通过（CI 口径）。

**O1 新增 release profile**（同文件 `[features]` 之后）：

```toml
[profile.release]
codegen-units = 1    # 让 LLVM 拿到更大优化单元
lto = true           # 链接期优化
opt-level = "s"      # 体积优先（要速度改回 3）
panic = "abort"      # 本壳无 catch_unwind 依赖
strip = true         # 去掉发布产物调试符号
```

**O2 收窄打包目标**（`apps/desktop/src-tauri/tauri.conf.json`）：

```diff
-    "targets": "all",
+    "targets": ["nsis", "deb", "dmg"],
```

依据（上游源码实读，非推断）：`tauri-bundler` 的 `settings.package_types()` 取到配置里的目标列表后**按当前目标平台过滤**——macOS 只留 `app`/`dmg`、Linux 只留 `deb`/`rpm`/`appimage`、Windows 只留 `msi`/`nsis`，跨平台项被丢弃。因此该列表三平台通用；另注意**过滤后为空集时 bundler 直接 `Ok(())` 返回（静默不打包）**，故列表必须覆盖每个目标平台至少一项（当前列表满足：macOS→dmg、Linux→deb、Windows→nsis）。

**O3 钉住 CI 的 tauri-cli 版本**：

- `.github/workflows/ci.yml`（`Install tauri-cli` 步）与 `.gitlab-ci.yml`（同注释行）：
  `cargo install tauri-cli --version "^2" --locked` → `cargo install tauri-cli --version "2.11.4" --locked`
- 依据：本地 `apps/desktop/pnpm-lock.yaml` 解析出的 `@tauri-apps/cli` 为 `2.11.4`；两处 pin 到同一 patch 版本后，本地与 CI 的 CLI 一致（ROADMAP E9 口径）。

**O4 补 web 依赖**（`Makefile` 的 `desktop-dev` 目标）：

```make
desktop-dev:
	cd apps/web && pnpm install --frozen-lockfile
	cd apps/desktop && pnpm install --frozen-lockfile && pnpm tauri dev
```

**O5 元数据 / 窗口 label / CSP**（`tauri.conf.json`）：

```diff
     "windows": [
       {
+        "label": "main",
         "title": "S3 Client",
@@
-      "csp": "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; media-src 'self' blob:; frame-src 'self' blob:; connect-src 'self' http: https:; object-src 'none'; base-uri 'self'"
+      "csp": "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; media-src 'self' blob:; frame-src 'self' blob:; connect-src 'self' http: https:; object-src 'none'; base-uri 'self'; frame-ancestors 'none'; form-action 'none'"
@@
   "bundle": {
     "active": true,
+    "publisher": "weilai1949",
+    "copyright": "Copyright © 2026 weilai1949",
+    "category": "DeveloperTool",
     "targets": ["nsis", "deb", "dmg"],
```

- `label: "main"` 与 [`capabilities/default.json`](../../apps/desktop/src-tauri/capabilities/default.json) 的 `windows: ["main"]` 对齐（消除「依赖 Tauri 默认 label」的隐式耦合）。
- `frame-ancestors 'none'` / `form-action 'none'` 为纯收紧：壳内既无嵌套需求也无表单提交面。
- ⚠️ **CSP 变更按 [`AI_POLICY.md`](../AI_POLICY.md) 属「需人类确认 + 人类专家复核」**：本项为**收紧**（不改 `connect-src` 的用户可配置后端白名单语义），仍需人工确认后才提交。

**O6 依赖升级**（`Cargo.toml` / `Cargo.lock` / `pnpm-lock.yaml`）：

- `tauri` `2.11.5` → `2.12.3`、`tauri-build` `2.6.3` → `2.7.2`（`cargo update -p tauri --precise 2.12.3` 或等价 `Cargo.toml` 收敛）；`@tauri-apps/cli` `2.11.4` → `2.12.2`（`apps/desktop/package.json` 收紧范围 + `pnpm install` 重写锁文件）。
- 连带必做：`./scripts/gen-third-party-licenses.sh` 重生成 [`THIRD_PARTY_LICENSES.md`](../THIRD_PARTY_LICENSES.md)（门禁 `TestThirdPartyLicensesAreComplete`）+ `cargo audit` 复跑；`tauri` 升到 2.12 后 `capabilities` / `tauri.conf.json` schema 若有变更需同步。
- **兜底**：若升级引入不可接受风险（新 advisory / 需改 schema / 需新插件），本项回退为「转入 [`KNOWN_ISSUES.md`](../KNOWN_ISSUES.md) 登记」，不阻塞其余条目。

**G1 防复发门禁**（两套 CI 的 desktop job，`cargo check` 步）：

```diff
-        run: cargo check --locked
+        env:
+          # 编译器自带 lint：直接依赖被声明却未被引用时**构建失败**（防 D1 复发）。
+          # Rust 侧此前无任何 unused 检查（rust-toolchain.toml 有意不装 clippy 组件），
+          # 故沿用编译器能力，不引入新工具、不改工具链。
+          RUSTFLAGS: -D unused_crate_dependencies
+        run: cargo check --locked
```

（GitLab 侧对应 job 的 `script:` 段落同样加 `RUSTFLAGS: -D unused_crate_dependencies`。）

## 3. 复现与验证口径（实跑记录）

```bash
# ① 基线：改前干净编译（apps/desktop/src-tauri）
cargo check --offline --locked        # Finished `dev` profile in 31.40s

# ② D1 取证：编译器自带的 unused_crate_dependencies
RUSTFLAGS="-W unused_crate_dependencies" cargo check --offline
#   warning: extern crate `serde` is unused in crate `s3client`
#   warning: extern crate `serde_json` is unused in crate `s3client`
#   warning: `s3client` (bin "s3client") generated 2 warnings

# ③ D1 判定验证（临时改动，已还原）：删除两行后
cargo check --offline                 # Finished `dev` profile in 0.47s（0 警告）
# 还原后 `git status` 显示 apps/desktop 无改动

# ④ O6 版本取证
pnpm tauri info                       # tauri 2.11.5 (latest 2.12.3) / @tauri-apps/cli 2.11.4 (2.12.2) …

# ⑤ 工具链 pin 核对：rust-toolchain.toml 生效（而非宿主默认）
cd apps/desktop/src-tauri && rustc --version   # rustc 1.98.1 (48a229cea 2026-09-01)
```

## 4. 已核对、确认**无需**改动（避免后来者重复排查）

- `icons/` 下的 iOS / Android / Windows logo 是 `tauri icon` 的标准产物，`bundle.icon` 只引用其中 5 个——**不是**死文件，属工具约定输出。
- `src-tauri/gen/`（`gen/schemas/*.json`）与 `src-tauri/target/` 已在根 [`.gitignore`](../../.gitignore) 排除；`Cargo.lock` **应**入库（Tauri 二进制可复现），现状正确。
- 发布链已收口：tag ↔ `tauri.conf.json` / `Cargo.toml` 版本比对（[`repo_infra_gate_test.go`](../../apps/server/repo_infra_gate_test.go)）、平台内唯一 `SHA256SUMS-<bundle>.txt` + 聚合 job、Sigstore 产物证明、桌面 SBOM——无可动的缺口。
- 无 IPC 硬约束合规：`main.rs` 无 `invoke_handler`、`capabilities/default.json` 的 `permissions` 为空数组、`withGlobalTauri: false`（与 [`threat-model.md`](../threat-model.md) 边界 B–E 的陈述一致）。
- 工具链 pin 一致：`rust-toolchain.toml` 1.98.1 ↔ 两套 CI 的 `dtolnay/rust-toolchain` / `rust:1.98.1-bookworm`（`repo_infra_gate_test.go` 守住）。

## 5. 本批次不纳入的候选优化清单（评审留档，11 项）

> **登记口径**：本节是**评审留档**，不是待办台账——待办的唯一来源是 [`KNOWN_ISSUES.md`](../KNOWN_ISSUES.md)（技术债）与 [`ROADMAP.md`](../ROADMAP.md)（方向）。下列候选项若要推进，先按 [`DEVELOPMENT.md`](../DEVELOPMENT.md) §4 登记到对应台账，再在本文件标注「已转登记」。
> **证据等级**：`实读` = 本机打开文件/命令确认；`上游源码` = 2026-10-12 实读上游仓库源码；`推断` = 未验证，仅给出假设。

| # | 候选 | 依据（等级） | 收益 | 代价 / 阻塞 | 归属 |
|---|---|---|---|---|---|
| C1 | `apps/web` 的 `vite.config.ts` 加 `strictPort: true` | `devUrl: http://localhost:1949` 与 vite `server.port: 1949`（实读） | 端口被占用时**快速失败**，不再是 Tauri 连不上/白屏 | 改 `apps/web`，牵动 web E2E 起服务方式 | apps/web 另立项 |
| C2 | Rust 侧 `cargo fmt` / `cargo clippy` 门禁 | `rust-toolchain.toml` 注释「仓库未使用 cargo fmt / cargo clippy，CI 以 `--profile minimal` 安装」（实读） | 补上 `unused_crate_dependencies` 之外的可读性 / 正确性告警面 | 改工具链安装口径 + 两套 CI 同改；首次接入大概率有存量告警 | 另行评估 |
| C3 | GitHub 侧 desktop job 加 Rust 依赖缓存 | `ci.yml` 的 `desktop` job 无 cache；GitLab 侧同 job 已有 cache（实读） | PR 反馈提速（每条 PR 现需完整编译 wry/gtk 依赖树） | 需引入第三方 action（必须 SHA pin + 供应链评审）并接受缓存投毒面 | 另行评估 |
| C4 | `bundle.windows.nsis`（`installMode` / `languages` / `displayLanguageSelector`） | `tauri.conf.json` 未配置该项（实读） | 安装体验与本地化 | 与代码签名批次耦合（未签名前用户先被 SmartScreen 拦） | 随 #25 一起 |
| C5 | `bundle.macOS.minimumSystemVersion` 显式声明 | `tauri.conf.json` 未配置（实读）；当前取上游默认 | 系统支持度显式化，不再依赖上游默认值 | 必须同步 [`compatibility.md`](../compatibility.md) §6.2 的桌面行 | 随 #25 一起 |
| C6 | Linux `.deb` 元数据（`depends` / `files` / AppStream） | `tauri.conf.json` 未配置（实读） | 包管理器体验、依赖显式化 | 收益低，且需真实发行版实测 | 待需求触发 |
| C7 | 桌面自动更新（`tauri-plugin-updater` / `createUpdaterArtifacts`） | `tauri.conf.json` 无 updater 配置（实读） | 存量用户可自动获得修复 | **外部凭证阻塞**（签名 / 公证未到位）**且引入插件会触碰 [ADR-001](../decisions/0001-desktop-no-ipc.md) 的「无插件」边界**，需新 ADR + 人类决策 | 需决策，交叉 KNOWN_ISSUES #25 |
| C8 | 窗口状态持久化（`tauri-plugin-window-state`） | 同 C7（实读） | 记住窗口尺寸 / 位置 | 同 C7（插件边界 + 新 ADR） | 需决策 |
| C9 | 系统托盘 / 原生通知 | [ADR-001](../decisions/0001-desktop-no-ipc.md) 明文「当前无此需求」（实读） | 常驻 / 提醒 | 同 C7；且 ADR 已给出「无需求」结论，先要有需求再谈 | 需决策 |
| C10 | 单实例（`tauri-plugin-single-instance`） | 后端已有数据目录 flock 单写者锁（`apps/server/internal/store/lock.go` 的 `AcquireDataDirLock`，实读；[ADR-011](../decisions/0011-single-instance-no-ha.md)） | 双击图标不产生第二个壳进程 | 同 C7；且壳层单实例与后端锁语义重叠，需一并设计 | 需决策 |
| C11 | `tauri` crate 的 feature 裁剪（`default-features = false` 可行性） | 现为 `features = []`，**默认特性仍然开启**（推断，未验证） | 依赖树 / 体积进一步下降 | 未验证，风险高：可能破坏 wry / 自定义协议路径，需实测编译 + 打包 | 待评估 |

**与 ADR-001 的关系（重要）**：C7–C10 都会**引入 Tauri 插件**。ADR-001 的决策是「无 IPC、无插件、capabilities 空权限集」，其代价一栏写明「桌面端无法使用需系统权限的特性（如系统托盘原生通知），当前无此需求」。因此这四项**不是实现细节优化，而是架构边界变更**：推进前需新写 ADR 承接（引用 ADR-001 并标注取代关系），并按 [`AI_POLICY.md`](../AI_POLICY.md) 交人类确认。

## 6. 未复跑 / 未实测（如实披露）

- **release 体积对比未实测**：O1 的收益按上游文档做**定性**判断（未在本地跑 `tauri build --release` 前后对比字节数）。
- **AppImage 打包未实跑**：O2 的「需联网」结论来自上游 bundler 源码（`linuxdeploy.rs` 的 `download()`），未在本地触发该路径。
- **远端 CI 未触发**：全部证据为本地实跑，两套 CI 的实际通过与否以本批次推送后的结果为最终凭据。

## 7. 与本批次代码改动的关系

本快照条目的**待办唯一来源**口径：D1 / O1–O6 / G1 均为**本批次内闭环**项（登记与闭环同一批次），故不进入 [`KNOWN_ISSUES.md`](../KNOWN_ISSUES.md)（该文件的定位是「尚未闭环」的问题）。闭环证据随代码写入 [`CHANGELOG.md`](../../CHANGELOG.md) `[Unreleased]` 与 [`FEATURES.md`](../FEATURES.md)；若其中任一项在本批次内**未能**闭环，则同 PR 转入 `KNOWN_ISSUES.md` 登记，并从本表状态列标为「转登记」。

## 8. 落地实证（2026-10-12 收口实跑）

> 与 §3（评审当天的**改前**记录）相对——本节是**改动落地后**的实跑原文，命令均在 `apps/desktop/src-tauri` 或仓库根执行。

```bash
# ① D1 + O1：删死依赖 → 重写 Cargo.lock（仅少 root package 两条直接依赖边）
cargo check --offline                  # 0 警告
git diff --stat Cargo.lock             # 1 file changed, 2 deletions(-)

# ② G1 门禁生效验证：删依赖后，lint 通过；若把 serde/serde_json 加回则构建失败
RUSTFLAGS="-D unused_crate_dependencies" cargo check --offline --locked   # Finished（0 警告）

# ③ O6：依赖升级（网络可用）
cargo update -p tauri --precise 2.12.3        # tauri 2.11.5 -> 2.12.3（连带 wry 0.57.0 / tao 0.37.1）
cargo update -p tauri-build --precise 2.7.2   # tauri-build 2.6.3 -> 2.7.2
#   apps/desktop：pnpm update @tauri-apps/cli@2.12.2（2.11.4 -> 2.12.2，package.json 收 `^2.12.2`）

# ④ 新版依赖 + G1 lint 回归（工作区已放 frontendDist 占位）
RUSTFLAGS="-D unused_crate_dependencies" cargo check --locked   # Finished `dev` profile（0 警告）

# ⑤ O6 连带：RustSec 审计（0 漏洞；告警 7 条 -> 2 条）
cargo audit                            # 0 vulnerabilities；unmaintained proc-macro-error + unsound glib 0.18.5

# ⑥ O6 连带：许可证清单重生成（门禁 TestThirdPartyLicensesAreComplete）
./scripts/gen-third-party-licenses.sh  # Go 43 / Rust 416 crates / npm 1；UNKNOWN 0
```

- **未在本地实跑而留待远端 CI 的项**：O2 的 `targets` 收窄只改了配置取值（实际三平台打包由 `release-desktop.yml` 的 `--bundles` 精确指定，本地未跑 `tauri build`）；O4 的 `make desktop-dev` 需要在全新检出上验证（本地已装 web 依赖）；两套 CI 的 O3 / G1 改动以推送后的实际结果为最终凭据。
- **O5 CSP 的人类确认**见文首横幅——该子项在人类专家复核前进度记为「改动已备、待确认」；`form-action` 取值由拟稿的 `'none'` 调整为 `'self'` 以通过 [`TestServerCSPSharesTauriDirectives`](../../apps/server/internal/handler/review_20260924_test.go)（理由见横幅）。
