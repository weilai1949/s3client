# 文档归档（Archive）

> 本目录存放**时点性文档**的历史快照：它们记录某次评估 / 审查 / 迁移在**当时**的结论，
> 结论被后续工作取代后即**冻结**，不再随代码演进更新。归档 = 冻结——**不删除、不回写、
> 不改写历史结论**；后续处置写进 [`features.md`](../features.md) 与
> [`CHANGELOG.md`](../../CHANGELOG.md)，本目录只保留原文。
>
> 命名与存放约定见 [`development.md`](../development.md) §4；当前**已归档 1 份**（见下表）。
>
> **待归档（当前政策例外）**：[`assessment.md`](../assessment.md)（2026-09-16 评估）是时点性快照，
> 仍被代码注释与活跃文档大量引用；在引用收敛前按活跃文档维护（可追加状态更新块，不回写历史
> 结论），待引用收敛后再按下方「归档操作」`git mv` 归档。原同列的 `review-2026-09-19.md` 已于
> 2026-09-23 完成全仓引用收敛并归档（见下表）。

## 归档清单

| 归档文件 | 原路径 | 归档日期 | 冻结的结论 |
|---|---|---|---|
| [review-2026-09-19.md](review-2026-09-19.md) | `docs/review-2026-09-19.md` | 2026-09-23 | 2026-09-19 分支整体状态审查（七维度实跑复核）：P0 / P1 / P2、S1–S6、R2–R10、D1–D10 全部闭环 |

> 除上述**待归档**的 `assessment.md` 外，`docs/` 下的文档均为**活跃文档**，须随代码同步维护
> （§4 文档同步门禁）。本目录约定随归档实践持续生效。

## 什么该归档

- **时点性快照**：综合评估、分支 / 版本审查、迁移对照表、已完成的一次性方案——结论绑定在某个
  commit 或日期上，不随后续代码变化而更新。
- **已被取代、且不再作为当前事实来源**的文档：仍有追溯价值（证据链、当时的判断依据），
  但其数字与结论已不代表现状。
- 归档后如需引用，**只作为历史证据**引用；当前事实一律以活跃文档为准。

## 什么不该归档

- **活跃 SSOT**：[`api.md`](../api.md) / [`architecture.md`](../architecture.md) /
  [`errors.md`](../errors.md) / [`features.md`](../features.md) / [`todolist.md`](../todolist.md) /
  [`roadmap.md`](../roadmap.md) / [`deployment.md`](../deployment.md) /
  [`development.md`](../development.md) / [`threat-model.md`](../threat-model.md)，
  以及根目录的 [`README.md`](../../README.md) / [`CHANGELOG.md`](../../CHANGELOG.md) /
  [`AGENTS.md`](../../AGENTS.md)——这些必须随代码同步更新。
- **ADR**：见 [`decisions/`](../decisions/index.md)。ADR 同样**不删除、不归档**——决策被取代时
  **新写一篇** ADR 引用旧篇，并把旧篇状态标为 `Superseded`，保留「当时为什么这么定」的原始理由。

## 归档操作

1. 用 `git mv` 移动（保留重命名历史，**不复制**、不删了重加）；
2. 修正全仓指向旧路径的引用（含源码注释里的 `docs/xxx.md` 路径），**链接不悬空**；
3. 在本文件「归档清单」补一行（归档文件 / 原路径 / 归档日期 / 冻结的结论）；
4. 在 [`CHANGELOG.md`](../../CHANGELOG.md) `[Unreleased]` 记一条 `变更`。
