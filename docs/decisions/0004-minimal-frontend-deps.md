# ADR-004：前端生产依赖仅保留 vue

## Status

Accepted

## Date

2026-09-16（回溯记录；决策自项目初版沿用）

## Context

s3clinet 前端（Vue 3 + Vite + TS）需要状态管理、路由、HTTP 客户端、国际化、主题等常见能力。生态惯例是引入 Pinia、vue-router、axios、vue-i18n 等库。

## Decision

**生产依赖仅 `vue`**：状态用轻量 `reactive` 模块（`store.ts`）、路由用 hash 深链接（`router.ts`，无 vue-router）、HTTP 用原生 `fetch` 封装（`api.ts`）、i18n 用自制消息字典（`i18n/messages/`）、主题用 CSS 变量 + `theme.ts`。

## Alternatives Considered

### Pinia / vue-router / axios / vue-i18n 全套
- Pros：生态成熟、功能丰富。
- Cons：为一个小型工具引入 4+ 运行时依赖，扩大供应链面（依赖审计、CVE 跟踪、包体积）；本项目前端功能边界清晰，无需复杂路由/全局状态。
- Rejected：收益低于成本。

## Consequences

- 供应链最小化：`pnpm audit` 0 已知漏洞，生产依赖仅 vue 一个运行时包。
- 自研封装需自行维护边界（HTTP 错误解析、i18n 键管理）——`api.ts` 694 行为最大文件，仍在 1000 行红线内。
- 已知债：i18n 出现约 40 个死键与 3 个引用未定义键（todo #11 / #24），需要脚本化清理。
