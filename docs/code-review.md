# Code Review — s3clinet P1/P2 迭代评估

评估范围：本 session 新增/修改的全部代码（Phase 1–6，含 flaky 测试修复）。
评估时间：2026-04-19。

---

## 严重（Critical / High）

### 1. `batchMetadata.ts` worker 池未处理 rejection（HIGH）
**文件**：`web/src/batchMetadata.ts:93-104`
**现象**：`worker(key).finally(() => inflight.delete(p))` — 当 `worker` 内部 `catch` 住错误后，promise 变为 resolved（不会 reject），所以 `Promise.race` 不会因为 rejection 抛出。
**实际风险**：`catch` 已在 worker 内部吞掉错误并 `return`，所以 `Promise.race` 不会因 rejection 崩溃。**此项误报**。但 worker 内的 `catch` 会 `failed++` 并 `return`，这意味着该 key 的后续 steps 不再执行（正确），但 `ok` 不会自增（也正确）。不过 `failed++` 是非原子操作，在并发下可能漏计。
**结论**：低风险，但 `failed`/`ok` 计数器在并发下非原子，需用 `Atomic` 或串行化。

### 2. `gaps_test.go` `release` channel 从未关闭（CRITICAL）
**文件**：`server/internal/service/gaps_test.go:547`
**现象**：`release := make(chan struct{})` 创建后从未 `close(release)`。测试注释声称"用 channel 显式确保两个 key 都已进入 fetch 再 cancel"，但实际同步靠的是 `ready.WaitGroup` + `cancel()`，`release` 是死代码。
**风险**：worker 中的 `select { case <-release; case <-c.Done(): }` 之所以没挂死，是因为 `cancel()` 在 `ready.Wait()` 之后触发，`c.Done()` 总是先于 `<-release` 触发。一旦有人按注释意图给 `release` 加 `close`，或两个 worker 同时选到 `release` 分支（Go select 随机性），worker 会在 ctx 已取消的情况下走"成功路径"返回 body，导致 `failKeys` 变成 0，测试静默失败。现有通过是偶然的。
**修复**：在 goroutine 里 `close(release)`（放在 `cancel()` 之后），与 `TestWriteObjectsZipPostGetCtxCheck` 保持一致。

---

## 中等（Medium）

### 3. `zip.go` `CreateHeader` 失败时 `item.body` 泄漏（MEDIUM）
**文件**：`server/internal/service/zip.go:101-106`
**现象**：`f, _ := zw.CreateHeader(hdr)` 忽略错误；当 `CreateHeader` 失败时 `f == nil`，后续 `io.Copy(f, ...)` 会 panic，且 `item.body` 永远不会被关闭。
**修复**：检查 `f` 的错误，`item.body.Close()` 后记入 `failKeys`。

### 4. `zip.go` manifest 文件句柄未关闭（MEDIUM）
**文件**：`server/internal/service/zip.go:110-113`
**现象**：`f, ferr := zw.Create("_下载失败清单.txt")` 后未 `defer f.Close()`；若 `io.WriteString` 失败，文件句柄泄漏。
**修复**：`if ferr == nil { defer f.Close(); ... }`。

### 5. `gaps_test.go` `done` 关闭与 `WriteObjectsZip` 返回之间竞争（MEDIUM）
**文件**：`server/internal/service/gaps_test.go:568/572`
**现象**：`close(done)` 在 `cancel()` 之后，但两个 worker 不一定已向 `results` 发完结果。主 goroutine `<-done` 解除后立即读 `failKeys`，而 `WriteObjectsZip` 可能还在收第二个 worker 的结果。
**实际**：虽实际安全（`results` 无缓冲，worker 在 send 时阻塞，直到主 goroutine 接收），但依赖 channel 顺序的生命周期耦合很脆弱，属于"碰巧正确"。
**修复**：去掉 `done` channel，直接 `<-` 一个 signaling mechanism，或等 `WriteObjectsZip` 返回后再检查。

### 6. `BatchMetadataDialog.vue` 空 tags 数组在 "replace" 模式下静默清空服务端标签（HIGH）
**文件**：`web/src/components/BatchMetadataDialog.vue:73-93`
**现象**：`applyTags=true`、`tagsMode='replace'`，但 `tags.value` 为空（用户从未添加行），验证循环零次通过，`tagsArg = []` 发给 API，清空所有对象标签。
**修复**：`tagsMode === 'replace' && tags.value.length === 0` 时拒绝执行并 toast 提示。

### 7. `BatchMetadataDialog.vue` 异常无用户反馈（MEDIUM）
**文件**：`web/src/components/BatchMetadataDialog.vue:95-116`
**现象**：`batchSetMetadata` 抛异常时，`finally` 重置 `running` 但无 toast/result 展示；UI 卡住无错误指示。
**修复**：`catch` 中 `toast(err, 'err')` 或 `result.value = { ok: 0, failed: input.keys.length }`。

### 8. `batchMetadata.ts` `acl` 类型过于宽松（MEDIUM）
**文件**：`web/src/batchMetadata.ts:23`
**现象**：`acl?: string` 应为 `'private' | 'public-read' | 'public-read-write'`（与 JSDoc 和 Vue 组件一致）。
**修复**：改为联合类型。

### 9. `batchMetadata.ts` `failed`/`ok` 计数器并发非原子（MEDIUM）
**文件**：`web/src/batchMetadata.ts:44-46, 84-90`
**现象**：`failed++` / `ok++` 在并发 worker 中非原子操作，可能漏计或重复计数。
**修复**：用 `Atomic`（如 `atomic.AddInt32`）或串行化计数器更新。

---

## 低等（Low）

### 10. `zip.go` `ctxReader` 无法中断阻塞式底层 Read（LOW）
**文件**：`server/internal/service/zip.go:159-163`
**现象**：`ctxReader.Read` 仅在调用前检查 `ctx.Err()`；若底层 Read 在 syscall 级别阻塞（如 S3 SDK 的网络读），取消 ctx 无法中断阻塞，`io.Copy` 会 hang。
**修复**：用 `io.CopyN` + timeout 或改用带 context 的 reader 包装。

### 11. `zip.go` `io.Copy` 失败后 ZIP 流处于未定义状态（LOW）
**文件**：`server/internal/service/zip.go:102-106`
**现象**：`io.Copy` 失败后，该 ZIP 条目只写了部分数据，后续条目继续写入，ZIP 流已损坏。
**修复**：记录失败后立即 `return` 或 `break`，不再写入后续条目。

### 12. `zip.go` producer goroutine 在 ctx cancel 时泄漏（LOW）
**文件**：`server/internal/service/zip.go:72-81`
**现象**：producer 在 `jobs <- k` 阻塞时若 ctx cancel，`select` 选中 `ctx.Done()` 返回，但 worker 可能仍在等待 `jobs` 通道的数据。worker 最终在 `range jobs` 结束时退出（通道关闭），但如果 producer 已退出且通道未关闭，worker 会永久阻塞。
**修复**：producer 退出前 `close(jobs)`，确保 worker 能退出。

### 13. `batchMetadata.ts` 空 `keys` 数组无守卫（LOW）
**文件**：`web/src/batchMetadata.ts:42`
**现象**：`keys.length === 0` 时返回 `{ok: 0, failed: 0}`，但语义上应视为"全部成功"（0 个对象需要修改）。
**修复**：返回 `{ok: 0, failed: 0, errors: []}`（当前已正确），或改为 `{ok: input.keys.length, failed: 0}` 以保持一致性。

### 14. `batchMetadata.ts` `accId` 别名多余（LOW）
**文件**：`web/src/batchMetadata.ts:43`
**现象**：`const accId = input.accountId` 在闭包中未使用，直接用 `input.accountId` 即可。
**修复**：删除别名。

### 15. `batchMetadata.ts` 冗余 `as` 类型断言（LOW）
**文件**：`web/src/batchMetadata.ts:65-81`
**现象**：`body` 对象已匹配各 `s3api` 调用的参数类型，`as` 断言多余。
**修复**：移除 `as` 断言。

### 16. `gaps_test.go` `sync.Once` 包装 `cancel()` 语义误用（LOW）
**文件**：`server/internal/service/gaps_test.go:546/567`
**现象**：`releaseOnce.Do(cancel)` 暗示"防止 double-cancel"，但 `context.CancelFunc` 本身可安全幂等调用。`sync.Once` 的真正作用是保证 `ready.Wait()` 之后 cancel 只执行一次，但用 `sync.Once` 表达"幂等 cancel"会让读者误以为 cancel 不幂等。
**修复**：去掉 `sync.Once`，直接 `cancel()`，或更正注释。

### 17. `BatchMetadataDialog.vue` 无障碍缺失（LOW）
**文件**：`web/src/components/BatchMetadataDialog.vue`
**现象**：(a) 表单控件无程序化标签（ACL select、tag-mode select、storage input 无 `aria-label`）；(b) 动态 status 区域无 `aria-live`；(c) tag key/value 输入框仅占位符无标签。
**修复**：添加 `aria-label` 和 `aria-live="polite"`。

### 18. `batchMetadata.test.ts` 76 行未使用参数（LOW）
**文件**：`web/src/batchMetadata.test.ts:76`
**现象**：`vi.mocked(s3api.putObjectAcl).mockImplementation((async () => { ... }) as typeof s3api)` 中 `id` 和 `body` 参数未使用。
**修复**：用 `_id` 和 `_body` 或移除参数。

---

## 信息（Info）

### 19. `zip.go` `defer zw.Close()` 错误抑制是设计意图（INFO）
**文件**：`server/internal/service/zip.go:30-34`
**现象**：`defer` 中 `zw.Close()` 错误仅在 `err == nil` 时覆盖返回值。这是设计意图——ZIP 写入已失败时不再覆盖错误。

### 20. `batchMetadata.test.ts` 并发测试覆盖不足（INFO）
**文件**：`web/src/batchMetadata.test.ts`
**现象**：测试仅用 2 keys 1 step，未触达 `BATCH_META_CONCURRENCY=4` 的并发路径。
**建议**：增加并发场景测试。

### 21. `gaps_test.go` Goroutine 泄漏（LOW）
**文件**：`server/internal/service/gaps_test.go:564-569`
**现象**：测试结束后，`WriteObjectsZip` 内部的 goroutine（producer、3 个闲住 worker、wg.Wait 关闭 results 的 goroutine）仍阻塞在 `jobs <- k` 或 `<-release` 上，直到进程退出。Go test 进程在所有 non-daemon goroutine 结束前不会退出——若后续测试依赖进程级清理或 `t.Parallel()`，会 hang。单测试单独跑不受影响。
**修复**：加 `t.Cleanup` 或让 worker 在 `ctx.Done()` 时退出 `jobs` 循环。

---

## 总结

| 严重度 | 数量 | 主要文件 |
|--------|------|----------|
| Critical/High | 2 | `batchMetadata.ts`, `gaps_test.go` |
| Medium | 6 | `zip.go`, `gaps_test.go`, `BatchMetadataDialog.vue`, `batchMetadata.ts` |
| Low | 9 | `zip.go`, `batchMetadata.ts`, `gaps_test.go`, `BatchMetadataDialog.vue` |
| Info | 4 | `zip.go`, `batchMetadata.test.ts`, `gaps_test.go` |

**总计 21 项**，其中 2 项 Critical/High 需立即修复，6 项 Medium 需修复，9 项 Low 建议修复。
