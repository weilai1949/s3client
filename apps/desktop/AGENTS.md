# AGENTS.md（apps/desktop 子树）

> 仅列 **Tauri 桌面端专属**约束与入口；仓库级规则见根 [`AGENTS.md`](../../AGENTS.md)（已被注入，不复制）。

- **无 IPC 是硬约束**：`src-tauri/src/main.rs` 只 `tauri::Builder::default().run(...)`，**不得**加
  `invoke_handler` / 自定义 `command` / 插件；`capabilities/default.json` 的 `permissions` 必须保持
  空数组，`tauri.conf.json` 的 `withGlobalTauri` 必须保持 `false`。理由见
  [`ADR-001`](../../docs/decisions/0001-desktop-no-ipc.md)。
- **替代原生命令的通道**：文件选择走 HTML5 `<input type="file">` / 拖放；请求走 HTTP 到 Go 后端
  与 S3 预签名直传——**不要**用 IPC 绕回 Rust 侧实现。
- **前端来源**：`build.beforeBuildCommand` / `frontendDist` 直接取 `../../web/dist`；改前端行为改
  `apps/web/`，不要在这里复制一份。
- **工具链**：Rust 版本固定于 `rust-toolchain.toml`（`channel` 需与两套 CI 的
  `dtolnay/rust-toolchain` pin 一致，由 `apps/server/repo_infra_gate_test.go` 校验）；pnpm 版本
  固定在 `package.json` 的 `packageManager`。
- **提交前**（在仓库根）：`make rust-audit`（cargo audit，RustSec 漏洞）+ `make desktop-build`
  （`tauri build`；默认 feature `custom-protocol` 用于打包内嵌资源）。
- **发布一致性**：`tauri.conf.json` 与 `Cargo.toml` 的版本必须与发布 tag 一致（`release-desktop.yml`
  会比对并 `exit 1`，见 `repo_infra_gate_test.go`）。
