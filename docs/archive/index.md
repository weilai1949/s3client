# 文档归档（Archive）

> 本目录存放**时点性文档**的历史快照：它们记录某次评估 / 审查 / 迁移在**当时**的结论，
> 结论被后续工作取代后即**冻结**，不再随代码演进更新。归档 = 冻结——**不删除、不回写、
> 不改写历史结论**；后续处置写进 [`FEATURES.md`](../FEATURES.md) 与
> [`CHANGELOG.md`](../../CHANGELOG.md)，本目录只保留原文。
>
> 命名与存放约定见 [`DEVELOPMENT.md`](../DEVELOPMENT.md) §4；当前**已归档 4 份**（见下表）。
> 目前**无待归档例外**——时点性文档在引用收敛后即按下方「归档操作」`git mv` 归档。

## 归档清单

| 归档文件 | 原路径 | 归档日期 | 冻结的结论 |
|---|---|---|---|
| [review-2026-09-19.md](review-2026-09-19.md) | `docs/review-2026-09-19.md` | 2026-09-23 | 2026-09-19 分支整体状态审查（七维度实跑复核）：P0 / P1 / P2、S1–S6、R2–R10、D1–D10 全部闭环 |
| [assessment.md](assessment.md) | `docs/assessment.md` | 2026-09-24 | 2026-09-16 五维度综合评估（代码质量 / 漏洞 / 死代码 / 降级 / 自我迭代）：评分 78/100，P0 / P1 / P2 与路线图 v1.0.0–v1.1.0 收口证据见 [`FEATURES.md`](../FEATURES.md) §H 起 |
| [code-review-2026-09-24.md](code-review-2026-09-24.md) | `docs/code-review-2026-09-24.md` | 2026-09-28 | 2026-09-24 全仓代码审查（六路五轴）：2 Critical + 20 Required **全部修复**，Nit 35 项中 31 闭环 / 3 转登记（→ `KNOWN_ISSUES` #61 #62 #63）/ 1 判定不成立；转登记的 #61 #62 已于 2026-09-28 闭环（见 [`FEATURES.md`](../FEATURES.md) §AB） |
| [code-review-summary.md](code-review-summary.md) | `docs/code-review-summary.md` | 2026-09-28 | 代码审查总结与改进建议（总体评分、质量指标、待解决项、改进建议）；头部「状态更新」活块 ①–⑥ 冻结 2026-09-24 → 2026-09-28 的收口事实，正文数字均为**审查时点值**（如「1042 用例 / 66 文件」）。其「⚠️ 待解决的技术问题」在归档前已清空到只剩 #25——该清单是在办事项、已闭环项直接移除 |

> `docs/` 下（本目录之外）的文档均为**活跃文档**，须随代码同步维护
> （§4 文档同步门禁）。本目录约定随归档实践持续生效。

## 什么该归档

- **时点性快照**：综合评估、分支 / 版本审查、迁移对照表、已完成的一次性方案——结论绑定在某个
  commit 或日期上，不随后续代码变化而更新。
- **已填写的事故复盘**：`incident-YYYYMMDD-<短名>.md`（空模板见
  [`POSTMORTEM_TEMPLATE.md`](../POSTMORTEM_TEMPLATE.md)）——事件闭环、行动项已登记进
  [`KNOWN_ISSUES.md`](../KNOWN_ISSUES.md) 后即冻结；正文是**当时**的取证与判断，不随后续修复回写。
- **已被取代、且不再作为当前事实来源**的文档：仍有追溯价值（证据链、当时的判断依据），
  但其数字与结论已不代表现状。
- 归档后如需引用，**只作为历史证据**引用；当前事实一律以活跃文档为准。

## 什么不该归档

- **活跃 SSOT**：[`api.md`](../api.md) / [`architecture.md`](../architecture.md) /
  [`errors.md`](../errors.md) / [`FEATURES.md`](../FEATURES.md) / [`KNOWN_ISSUES.md`](../KNOWN_ISSUES.md) /
  [`ROADMAP.md`](../ROADMAP.md) / [`DEPLOYMENT.md`](../DEPLOYMENT.md) /
  [`DEVELOPMENT.md`](../DEVELOPMENT.md) / [`threat-model.md`](../threat-model.md)，
  以及根目录的 [`README.md`](../../README.md) / [`CHANGELOG.md`](../../CHANGELOG.md) /
  [`AGENTS.md`](../../AGENTS.md)——这些必须随代码同步更新。
- **ADR**：见 [`decisions/`](../decisions/index.md)。ADR 同样**不删除、不归档**——决策被取代时
  **新写一篇** ADR 引用旧篇，并把旧篇状态标为 `Superseded`，保留「当时为什么这么定」的原始理由。

## 归档操作

1. 用 `git mv` 移动（保留重命名历史，**不复制**、不删了重加）；
2. 修正全仓指向旧路径的引用（含源码注释里的 `docs/xxx.md` 路径），**链接不悬空**；
3. 在本文件「归档清单」补一行（归档文件 / 原路径 / 归档日期 / 冻结的结论）；
4. 在 [`CHANGELOG.md`](../../CHANGELOG.md) `[Unreleased]` 记一条 `变更`。
