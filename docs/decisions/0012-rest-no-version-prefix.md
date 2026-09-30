# ADR-012：REST 契约无版本前缀，破坏性变更靠发布节奏缓冲

## Status

Accepted

## Date

2026-09-18（回溯记录；`routes.go` 自 2026-09-18 起注册 `/api/*` 无版本前缀，决策自项目
初版沿用）

## Context

单产品、单前端随仓库演进，当前没有「同时维护多个不兼容 API 版本」的现实需求。
[`compatibility.md`](../compatibility.md) §3 已把这套演进口径写成自我约束；本 ADR 记录
「**为什么不引入版本机制**」这个决定本身。

现状（2026-09-30 回读核实）：

- [`apps/server/internal/handler/routes.go`](../../apps/server/internal/handler/routes.go)：
  全部端点形如 `GET /api/...`，**无 `/api/v1` 前缀、无 `Accept` 协商、无版本请求头**
  （[`compatibility.md`](../compatibility.md) §3.2）。
- 非破坏性变更允许（新增端点 / 新增可选字段 / 放宽校验等，§3.3）；破坏性变更必须
  `CHANGELOG` 显式标注 + 走 §7 弃用流程 + 靠发布节奏（`-rcN` / 大版本 / 时间戳版）缓冲（§3.4）。
- 机械护栏：OpenAPI 契约门禁族 + `api.md` 双向门禁（[`DEVELOPMENT.md`](../DEVELOPMENT.md)
  §3「契约与文档门禁的落点」）。

## Decision

**不引入 URL 版本前缀与版本协商；兼容性靠「机械门禁 + CHANGELOG 显式标注 + 发布节奏」
三件套；破坏性变更能弃用就不直接删。**

## Alternatives Considered

### `/api/v1` 前缀 + 并行版本
- Pros：破坏性变更零缓冲成本。
- Cons：需要同时维护多版本注册表 / 文档 / 前端；当前单产品无多版本并行需求，引入即永久
  承担版本化开销。
- 被拒（[`compatibility.md`](../compatibility.md) §3.2 现状表述）。

### Accept 协商 / 版本请求头
- Pros：语义化表达。
- Cons：比前缀更隐晦，客户端出错更难排查；同样被「无多版本需求」否决。
- 被拒。

### 完全冻结 API 并承诺 LTS
- Pros：客户端最稳。
- Cons：与「单人维护快速演进」现实不符，会成为无法兑现的承诺（`compatibility.md` §3.2
  明言「当前实践的约束，不是已承诺的硬保证」）。
- 被拒。

## Consequences

- 破坏性变更的缓冲期与发布节奏绑定（预发布 / 大版本 / 时间戳版逐条点出）。
- 机械护栏只钉「文档与实现不漂移」，业务语义稳定靠评审 + CHANGELOG。
- 将来若出现第二类不兼容客户端（如第三方 SDK / 移动端），需新 ADR 引入版本机制并
  Supersede 本篇。
