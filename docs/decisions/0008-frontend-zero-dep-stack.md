# ADR-008：前端零运行时依赖下的自研路由 / 状态 / i18n / HTTP（ADR-004 姊妹篇）

## Status

Accepted

## Date

2026-09-18（回溯记录；`router.ts` / `store.ts` / `i18n/index.ts` 于 2026-09-18 首次引入；
`api/http.ts` 于 2026-09-19 从 `api.ts` 拆分引入）

## Context

[ADR-004](0004-minimal-frontend-deps.md) 决定前端生产依赖只有 `vue`（供应链最小化）。
其配套问题是：路由 / 状态 / i18n / HTTP 这些通常由第三方库承担的职能怎么办。本 ADR 记录
这些**配套自研**的取舍，是 ADR-004 的姊妹篇；「是否引入依赖」的总体权衡见 ADR-004，本文
不复述。逐库拒绝理由的当时原始讨论未在仓库文档中找到记录，下列对比为 2026-09-30 回溯补记
（**未验证**），以 ADR-004 为准。

现状（2026-09-30 回读源码核实）：

- [`apps/web/package.json`](../../apps/web/package.json)：`dependencies` 仅
  `{"vue": "^3.5.13"}`。
- 路由：[`apps/web/src/router.ts`](../../apps/web/src/router.ts) hash 深链接（`#/tab`），
  `tabFromHash` / `setTabHash` / `onTabHashChange`，无 vue-router。
- 状态：[`apps/web/src/store.ts`](../../apps/web/src/store.ts) 全局 `reactive` state +
  `tabRequest` / `accountFormRequest` 跨面板信号，无 Pinia。
- HTTP：[`apps/web/src/api/http.ts`](../../apps/web/src/api/http.ts) 原生 `fetch` 封装
  （base / Bearer / JSON / 错误归一），无 axios。
- i18n：[`apps/web/src/i18n/index.ts`](../../apps/web/src/i18n/index.ts) 自研字典合并 +
  `t` / `tf` 插值 + localStorage 持久化 + `<html lang>` 同步，无 vue-i18n。
- 主题、快捷键栈（`useKeydownStack`）等其余能力同属此列。

## Decision

**路由 / 状态 / i18n / HTTP / 主题全部自研，运行时依赖只有 `vue`；新增依赖必须先挑战
必要性并回指 ADR-004。** 该原则由 [`apps/web/AGENTS.md`](../../apps/web/AGENTS.md)
「零运行时依赖原则」与覆盖率 / 死代码门禁共同守住。

## Alternatives Considered

### vue-router
- Pros：嵌套路由 / 守卫 / devtools 生态成熟。
- Cons：本应用只有 7 个 tab 的扁平导航，hash 深链接约 30 行即可覆盖；引入增加体积与攻击面。
- 被拒。

### Pinia
- Pros：模块化 store、devtools。
- Cons：当前状态面小（账号 / tab / toast），`reactive` + 跨面板信号已足够。
- 被拒。

### axios
- Pros：拦截器 / 取消 / 兼容层生态。
- Cons：`fetch` + `AbortController` 已覆盖需求；上传进度用 XHR 是唯一例外（`upload.ts` 注释）。
- 被拒。

### vue-i18n
- Pros：复数 / 复杂插值 / 懒加载。
- Cons：两语言（zh-CN / en-US）字典 + `t` / `tf` 插值足够；键完整性另有机房门禁。
- 被拒。

## Consequences

- 获得：体积最小、零供应链依赖、全部行为可测（100% 覆盖率门禁可落实）。
- 代价：框架能力（嵌套路由、devtools、复杂复数规则等）缺失；新增功能需要先自研等价物，
  团队必须长期遵守零依赖原则。
- 历史教训：自研 i18n 曾漏同步 `<html lang>`（KNOWN_ISSUES #67①），已闭环并由
  `a11y_gate.test.ts` 钉住——自研意味着边界行为要自己写门禁，这是本决策的隐性成本。
