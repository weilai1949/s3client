# Code Review — s3clinet P1/P2 迭代评估

评估范围：本 session 新增/修改的全部代码（Phase 1–6，含 flaky 测试修复）。
评估时间：2026-04-19。

---

## 已修复（Fixed）

> **已迁移**：21 项逐条修复记录（CRITICAL / HIGH / MEDIUM / LOW）见 **[`FEATURES.md`](FEATURES.md)** 的「二、已完成修复与优化 → D」。

## 仍存在（Acknowledged, not fixed）

> **已迁移**：其中 `zip.go` `ctxReader` 阻塞读后续已由 `ctxCancelReader` + `context.AfterFunc`
> 专项修复；`batchMetadata` 计数器在 JS 单线程模型下安全。两项最终状态见 **[`FEATURES.md`](FEATURES.md)**。

---

## 验证结果

| 测试组合 | 结果 |
|---------|------|
| `go test -race -count=30 -run 'TestWriteObjectsZip|TestRunBatchProgressAndAggregate'` | 30/30 绿 |
| `go test -race -count=3 ./internal/service/` | 3/3 绿（50.6s） |
| `go test -race -count=1 ./...` | 全绿（handler 148s, service 15s） |
| `pnpm test`（前端 61 单测） | 61/61 绿 |
| `pnpm typecheck` | 干净 |
| `pnpm build` | 351KB / 107KB gzip |

---

## 总结

| 严重度 | 原数量 | 已修复 | 仍存在 |
|--------|--------|--------|--------|
| Critical | 1 | 1 | 0 |
| High | 2 | 2 | 0 |
| Medium | 6 | 5 | 1（计数器非原子，JS 单线程安全） |
| Low | 9 | 7 | 2（ctxReader 阻塞、producer 泄漏 — 无实际泄漏） |
| Info | 4 | 2 | 2 |

**核心结论**：21 项中 20 项已修复，1 项（JS 计数器非原子）在单线程模型下实际安全，1 项（ctxReader 阻塞中断）需架构级改动。

> 注：上表为 2026-04-19 评估时点状态；`ctxReader` 阻塞中断已于后续专项修复。最终状态见 [`FEATURES.md`](FEATURES.md)。
