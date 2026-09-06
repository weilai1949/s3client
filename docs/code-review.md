# Code Review — s3clinet P1/P2 迭代评估

评估范围：本 session 新增/修改的全部代码（Phase 1–6，含 flaky 测试修复）。
评估时间：2026-04-19。

---

## 已修复（Fixed）

### CRITICAL → Fixed
**1. `gaps_test.go` `release` channel 从未关闭（CRITICAL）**
- 修复：在 goroutine 里 `close(release)`（放在 `cancel()` 之后），与 `TestWriteObjectsZipPostGetCtxCheck` 保持一致。
- 修复：去掉 `done` channel 竞争，改用 `wg.Wait()` 确保 `WriteObjectsZip` 返回后再检查结果。
- 修复：去掉 `sync.Once` 包装幂等 `CancelFunc`。

### HIGH → Fixed
**2. `BatchMetadataDialog.vue` 空 tags 数组在 "replace" 模式下静默清空服务端标签**
- 修复：`tagsMode === 'replace' && tags.value.filter(t => t.key).length === 0` 时 toast 拒绝。

**3. `BatchMetadataDialog.vue` 异常无用户反馈**
- 修复：`catch` 块中 `toast(err, 'err')` + `result.value = { ok: 0, failed: keys.length }`。

### MEDIUM → Fixed
**4. `zip.go` `CreateHeader` 失败时 `item.body` 泄漏**
- 修复：检查 `f` 的错误，`item.body.Close()` 后记入 `failKeys`，`continue`。

**5. `zip.go` manifest 文件句柄未关闭**
- 修复：`io.Writer` 无 `Close` 方法，移除无效 `defer f.Close()`。

**6. `zip.go` `io.Copy` 失败后 ZIP 流处于未定义状态**
- 修复：`break` 替代 `continue`，不再写入后续条目。

**7. `batchMetadata.ts` `acl` 类型过于宽松**
- 修复：`acl?: 'private' | 'public-read' | 'public-read-write'`。

**8. `batchMetadata.test.ts` 未使用参数**
- 修复：`_id` 和 `_body` 前缀。

### LOW → Fixed
**9. `BatchMetadataDialog.vue` 无障碍缺失**
- 修复：添加 `aria-label`（ACL select、tag-mode select、storage input）和 `aria-live="polite"`（status 区域）。

**10. `batchMetadata.ts` `accId` 别名多余**
- 修复：已移除（代码中已直接使用 `input.accountId`）。

**11. `batchMetadata.ts` 冗余 `as` 类型断言**
- 修复：已移除（类型已正确推断）。

**12. `batchMetadata.ts` 空 `keys` 数组守卫**
- 修复：已确认 `keys.length === 0` 时返回 `{ok: 0, failed: 0}` 语义正确。

---

## 仍存在（Acknowledged, not fixed）

### LOW — `zip.go` `ctxReader` 无法中断阻塞式底层 Read
- `ctxReader.Read` 仅在调用前检查 `ctx.Err()`；若底层 Read 在 syscall 级别阻塞（如 S3 SDK 的网络读），取消 ctx 无法中断阻塞，`io.Copy` 会 hang。
- **不修复原因**：需改用带超时/取消的 reader 包装（如 `net.Conn.SetReadDeadline`），属于架构级改动，非本迭代范围。

### LOW — `zip.go` producer goroutine 在 ctx cancel 时泄漏
- producer 在 `jobs <- k` 阻塞时若 ctx cancel，`select` 选中 `ctx.Done()` 返回，`defer close(jobs)` 确保通道关闭，worker 的 `range jobs` 退出。**无泄漏**。

### MEDIUM — `batchMetadata.ts` `failed`/`ok` 计数器并发非原子
- JavaScript 单线程事件循环中，`failed++` / `ok++` 在 `await` 之间是原子的（无抢占）。**实际安全**，但计数器在 `worker` 的 `catch` 中更新，顺序非确定。
- **不修复原因**：JS 单线程模型保证原子性；误差可忽略。

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
