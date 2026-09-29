# AGENTS.md（apps/web 子树）

> 仅列 **前端专属**约束与入口；仓库级规则见根 [`AGENTS.md`](../../AGENTS.md)（已被注入，不复制）。

- **零运行时依赖原则**：`dependencies` 只有 `vue`。状态 / 路由（hash 深链接）/ HTTP（原生
  `fetch`）/ i18n / 主题全部自制——**不要**引入 Pinia、vue-router、axios、vue-i18n。
  缘由与代价见 [`ADR-004`](../../docs/decisions/0004-minimal-frontend-deps.md)。
- **提交前**（在 `apps/web/` 内）：`pnpm lint`（eslint，`--max-warnings 0`）、`pnpm typecheck`
  （`vue-tsc --noEmit`）、`pnpm test`（vitest）、`pnpm build`（`vue-tsc` + `vite build`）。
- **覆盖率门禁**：`pnpm test:coverage` 需**四指标 100%**；排除项只有 `src/main.ts` / `src/env.d.ts` /
  `src/**/*.test.ts` / `src/i18n/messages/**`（纯数据字典，另有键完整性门禁）/ `src/assets/**`
  ——`src/i18n/index.ts` 的读写与回退逻辑**纳入统计**。另有 `src/deadcode_gate.test.ts` 死代码门禁。
- **E2E 源码也要类型检查**：`pnpm typecheck:e2e`（`tsconfig.e2e.json` 覆盖 `e2e/` 与 `e2e-real/`）。
  - `pnpm e2e`：mock `/api` 的浏览器用例；`pnpm e2e:real`：真实后端 + 真实 RustFS，**不得** mock。
- **安全**：禁 `v-html`；`secretKey` 不得进 localStorage（凭据与存储位置的现状见
  [`docs/user-guide.md`](../../docs/user-guide.md)「数据与隐私」）；外部数据一律不可信。
- **产物**：构建输出 `dist/`，由服务端 `S3C_STATIC_DIR` 托管、桌面端 `frontendDist` 内嵌——
  桌面端**不是**独立应用，不能在 Tauri 侧另起服务。
