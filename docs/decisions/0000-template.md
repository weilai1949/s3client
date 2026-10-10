# ADR-NNN：<决策的一句话标题>

> **用法**：复制本文件为 `docs/decisions/NNNN-<短横线短名>.md`（NNNN 为四位递增编号、短名小写
> kebab-case），逐节填写，并在 [`index.md`](index.md) 的索引表登记一行。**编号体例**：文件名用
> **四位**（`0013-…`），标题与正文 / 索引引用一律用**三位** `ADR-0NN`（如 `ADR-013`，与 `ADR-001..ADR-012`
> 及代码注释口径一致）。**不要**在 ADR 里记录
> 待办 / 缺陷——那些走 [`KNOWN_ISSUES.md`](../KNOWN_ISSUES.md)（唯一来源）。
>
> **章节约定**：H2 用**英文**（`Status` / `Date` / `Context` / `Decision` /
> `Alternatives Considered` / `Consequences`）——与既有的
> ADR-001..0004 一致，便于跨仓库检索与工具解析；正文用中文。
>
> **生命周期**：ADR **不归档、不删除**。决策变化时新写一篇：新篇 `Status` 写
> `Accepted（Supersedes ADR-NNN）`，旧篇改标 `Superseded by ADR-MMM`；只是补充事实 / 收紧配置而决策
> 本身不变时，**追加**一节 `## Update（YYYY-MM-DD）`（见 ADR-003 / ADR-004），不要改写历史结论。

## Status

Proposed | Accepted | Deprecated | Superseded by ADR-NNN

> **填写形态**（2026-10-10 统一，门禁 [`../../apps/server/adr_format_gate_test.go`](../../apps/server/adr_format_gate_test.go)）：
> 正文只写**裸枚举值**——`Proposed` / `Accepted` / `Deprecated`；被取代的旧篇写 `Superseded by ADR-MMM`，
> 取代旧篇的新篇写 `Accepted（Supersedes ADR-NNN）`（**仅此两种括注形态**）。**不要**在正文追加
> 「（已采纳）」这类**语义括注**：括注只属于 [`index.md`](index.md) 的「状态」列（`Accepted（已采纳）` /
> `Accepted（已采纳；默认策略未变）`）——正文供工具解析、索引给人读，两处形态不同是有意的。

## Date

YYYY-MM-DD（若为回溯记录，注明「回溯记录；决策自 <时点> 沿用」）

## Context

要解决的**问题**与当时的约束：现状是什么、为什么非决定不可、有哪些不可迁就的前提
（安全边界 / 兼容性 / 依赖预算 / 上游限制）。只写与本次决策相关的背景，不复述整份架构。

## Decision

**做出的是什么决定**（一句话可复述），以及它落在哪些代码 / 配置 / 文档上（给出具体路径或
变量名）。决定必须足够具体，让读者不必猜测边界条件。

## Alternatives Considered

逐个列出**认真考虑过**的替代方案（含「什么都不做」），每个都要写 **Pros / Cons**。
方案用 `### <方案名>` 起头（**不要**用编号列表：编号会随增删漂移，标题才是稳定锚点；
体例由 `adr_format_gate_test.go` 钉住）。
不要写明显荒谬的稻草人——替代方案的价值在于解释「为什么没选它」。

## Consequences

- **正面**：获得什么（可验证的收益，而非口号）。
- **代价 / 风险**：失去什么、引入了什么新的失败模式、哪些约束从此必须长期遵守。
- **后续**：需要同步的文档 / 门禁 / 测试；若留下未决问题，登记到
  [`KNOWN_ISSUES.md`](../KNOWN_ISSUES.md) 而不是留在这里。
