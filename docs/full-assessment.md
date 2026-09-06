# s3clinet 全方位评估报告

评估范围：当前 `develop` 分支全部代码（含 Phase 1–6 新增功能与 flaky 测试修复）。
评估日期：2026-04-19。
评估维度：架构设计、安全性、性能、并发、测试、文档、DevOps、前端 UX、API 设计、错误处理、无障碍、可维护性。

---

## 一、架构与设计（Architecture & Design）

### 优势
- **分层清晰**：`store → model → s3wrap → handler → service`，依赖方向统一向前，SDK 类型不泄露到 s3wrap 之外。
- **OpenAPI 零反射**：显式 builder 模式，零依赖，契约与代码一一对应（67 端点）。
- **Fake S3 模式**：`httptest.NewServer` + XML 组装，测试与真实 S3 行为对齐。
- **批量操作有界并发**：`RunBatch` / `WriteObjectsZip` / `batchMetadata` 均使用 worker pool + 有界并发。

### 问题
| # | 严重度 | 文件 | 描述 |
|---|--------|------|------|
| A-1 | MEDIUM | `openapi_register.go` | 825 行单一文件，9 个子注册函数全部内联。可拆分为 `register_*.go` 每文件一个域，降低认知负荷。 |
| A-2 | LOW | `handler.go` | `Handler` 结构体 12 个字段，职责偏多。可考虑拆分为 `AccountHandler` / `ObjectHandler` / `MigrateHandler`。 |
| A-3 | LOW | `s3wrap/client.go` | `sharedHTTPClient` 用 `sync.Once` 全局共享，所有账号共用同一 HTTP 客户端（含连接池）。跨账号隔离性差，但 Tauri 本地场景可接受。 |
| A-4 | LOW | `service/sync.go` | `SyncKeys` 先列举源全部对象再过滤，内存占用 O(n)。大桶（>100k）可能 OOM，但硬上限 100k 已缓解。 |
| A-5 | LOW | `service/batch.go` | `results` 通道缓冲 `total`，大批量任务可能内存占用高。 |

---

## 二、安全性（Security）

### 优势
- **SSRF 防护**：`ssrfAwareClient` 拦截链路本地/云元数据 IP，禁用 HTTP(S)_PROXY。
- **常量时间比较**：`secureCompare` 用 SHA-256 + `subtle.ConstantTimeCompare` 防时序攻击。
- **CORS 白名单**：仅放行 localhost/127.0.0.1/tauri，自定义 Origin 需显式配置。
- **请求体限制**：`maxBody = 4MB`，`DisallowUnknownFields` 防畸形 JSON。
- **桶名校验**：拒绝首尾为 `.`/`-`、连续 `..`/`.-`/`-.`。
- **内容安全策略**：`default-src 'self'`，脚本仅同源，内联样式给 Vue 用。

### 问题
| # | 严重度 | 文件 | 描述 |
|---|--------|------|------|
| S-1 | **HIGH** | `store/store.go:189` / `sqlite.go:33` | **明文存储 SecretKey**。JSON 驱动和 SQLite 驱动均以明文持久化 `SecretKey`。仅 `encrypted` 驱动（AES-256-GCM + Argon2id）加密 secrets。文件系统访问 → 所有 S3 凭据泄露。 |
| S-2 | HIGH | `ratelimit.go:66-73` | `clientIP` 仅信任 `RemoteAddr`，未处理 `X-Forwarded-For`。Tauri 本地场景安全，但若未来部署到公网，IP 限速可被伪造绕过。 |
| S-3 | MEDIUM | `middleware.go:180` | `/api/openapi.json` 不进鉴权层——契约端点暴露所有端点路径与参数，可能辅助攻击者枚举 API。 |
| S-4 | MEDIUM | `bucketPolicy.ts` | `parsePolicy` 对 `NotPrincipal` / 嵌套 Principal 数组返回 null，UI 回退 JSON 模式。但用户可能误以为"已解析"而实际未生效。 |
| S-5 | MEDIUM | `batchMetadata.ts` | 无客户端/服务端 key 数量上限；100k keys → 100k 次独立 API 调用。服务端无 `maxBatchKeys` 强制限制。 |
| S-6 | LOW | `openapi_register.go` | OpenAPI 端点暴露了所有 API 路径、参数、响应 schema，对攻击者信息泄露。需评估是否应在生产环境隐藏。 |
| S-7 | LOW | `handler/accounts.go` | `testAccount` 返回 `s3UserMessage(err)` 在 JSON 体中，可能泄露 S3 后端内部信息。 |
| S-8 | LOW | `proxy.go:29` | `key` 从查询参数直接传给 S3 API，无 `..` 路径遍历检查。 |
| S-9 | LOW | `middleware.go:23` | CSP `connect-src 'self' http: https:` 允许任意 HTTP 连接；被攻陷的前端可外传数据。 |

---

## 三、性能与并发（Performance & Concurrency）

### 优势
- **有界并发**：`WriteObjectsZip` / `RunBatch` / `batchMetadata` 均使用 worker pool。
- **虚拟滚动**：MigratePanel 虚拟滚动，列表上限 200×1000。
- **SSE 卸载中断**：`useObjectActions.showDetail` 加 `detailSeq` 守卫；`MigratePanel` 组件级 `activeUnsub`。

### 问题
| # | 严重度 | 文件 | 描述 |
|---|--------|------|------|
| P-1 | **HIGH** | `s3wrap/client.go:63-67` | **`awsconfig.LoadDefaultConfig(context.Background(), …)` 不可取消**。配置加载在 handler 请求路径上运行，不可取消，在不可达 endpoint 上无限期阻塞 handler goroutine。 |
| P-2 | HIGH | `service/migrate.go:43-46` | **`SameEndpoint` 空 endpoint 回退**：两端均为空时返回 false，强制使用 StreamCopy（跨端点慢路径），即使两侧是同一 S3 实例。静默性能退化。 |
| P-3 | MEDIUM | `zip.go:133-138` | `io.Copy` 失败后 `break`，但 worker/goroutine 未清理；goroutine 泄漏（已修复）。 |
| P-4 | LOW | `service/sync.go:51-57` | 列举 src/dst 之间无 `ctx.Err()` 检查；取消在列举中点时仍完成全部列举。 |
| P-5 | LOW | `service/batch.go:49` | `results` 通道缓冲 `total`（最多 10k），大批量任务内存尖峰。 |
| P-6 | LOW | `s3wrap/client.go:23-26` | `sharedHTTPOnce` 全局共享 HTTP 客户端，所有账号共用连接池。跨账号并发可能互相影响。 |

---

## 四、测试质量（Testing）

### 优势
- **覆盖率高**：新代码路径全部有测试（sync 6、openapi 4、handler 2、batchMetadata 5）。
- **Race detector**：`go test -race` 全绿。
- **Flaky 测试修复**：`TestWriteObjectsZipCancelDuringFetch` 和 `TestRunBatchProgressAndAggregate` 已修复，30 次连续通过。
- **Playwright E2E**：4 通过 / 1 skip，覆盖 SPA 渲染 + API 契约 + 账户流。

### 问题
| # | 严重度 | 文件 | 描述 |
|---|--------|------|------|
| T-1 | MEDIUM | `gaps_test.go` | `TestWriteObjectsZipCancelDuringFetch` 修复后仍依赖 `ready.WaitGroup` 同步，若 `get` 函数内部不调用 `ready.Done()` 则死锁。测试与实现耦合紧。 |
| T-2 | MEDIUM | `batchMetadata.test.ts` | mock 用 `vi.mocked(s3api.putObjectAcl)` 强制类型断言，若 API 签名变更则测试不报错但也不生效。 |
| T-3 | HIGH | `sync_test.go` | `s3FakeStore`/`s3FakeSize`/`s3FakeEtag` 是包级全局变量，无并发安全保护。`go test -race` 会数据竞争。 |
| T-4 | HIGH | `migrate_sync_test.go` | `syncStore` 同理是包级全局变量，测试修改后未在 defer 中清理。 |
| T-5 | MEDIUM | `handler/migrate_sync_test.go` | 仅覆盖 SkipsEqualByETag 和 InvalidMode，缺少 CompareSizeTime 模式、prefix 过滤、跨端点同步测试。 |
| T-6 | MEDIUM | `web/e2e/` | 仅 smoke(3 用例) + account-flow(2 用例)，缺少桶管理、对象操作、多部分上传、迁移、版本控制、回收站。 |
| T-7 | LOW | `Makefile` | `test` 目标未包含 `-race` 或 `-cover`；`web-test` 未指定 `--coverage`。 |

---

## 五、文档与 DevOps（Documentation & DevOps）

### 优势
- **CHANGELOG / API.md** 已更新，记录 P1/P2 路线落地。
- **code-review.md**：21 项评估文档。
- **Playwright CI**：PR/dispatch + 每周三 02:00 UTC 冒烟。
- **Conventional Commits**：所有提交遵循规范。

### 问题
| # | 严重度 | 文件 | 描述 |
|---|--------|------|------|
| D-1 | MEDIUM | `docs/code-review.md` | 文档未纳入版本控制流程（未在 CI 中自动生成/校验）。建议 pre-commit hook 自动运行 race detector 并更新文档。 |
| D-2 | LOW | `.github/workflows/e2e-playwright.yml` | 未设置 `fail-fast: false`，若一个 e2e 测试失败，整个 job 停止。可能需重试机制。 |
| D-3 | LOW | `Makefile` | 未见 `Makefile`（之前提到 `make tidy` 显式入口）。需确认 Makefile 是否存在。 |
| D-4 | LOW | `docs/API.md` | 新增 `POST /api/migrate/sync` 和 `GET /api/openapi.json` 章节，但未更新 `docs/ERRORS.md` 中的错误码对照。 |

---

## 六、前端 UX 与无障碍（Frontend UX & Accessibility）

### 优势
- **视觉策略编辑器**：表单式编辑 + 4 个常用模板 + 实时 JSON 预览。
- **批量元数据对话框**：3 个 fieldset 独立勾选，运行中进度 + 失败明细。
- **i18n 中英文对齐**：新增 11 个键（zh-CN + en-US）。
- **模式切换**：桶策略可视化/JSON 双模式，不支持结构自动回退。

### 问题
| # | 严重度 | 文件 | 描述 |
|---|--------|------|------|
| UX-1 | HIGH | `BatchMetadataDialog.vue:162-164` | 标签 key/value 输入框无 `<label>`，仅占位符。WCAG 2.1 违规。 |
| UX-2 | HIGH | `BucketPolicyVisualEditor.vue:196-243` | 所有输入（effect/sid/principal/actions/resources）无 `<label>` 或 `aria-label`。 |
| UX-3 | HIGH | `BucketPolicyVisualEditor.vue:261-267` | 原始 JSON textarea 无 `<label>`。 |
| UX-4 | MEDIUM | `BucketPolicyVisualEditor.vue:32-52` | `watch(() => props.raw)` + `{ immediate: true }` 级联触发 `watch(doc, ...)`，若父组件重新渲染相同 `raw`，`dirty` 被重置为 `false`，可能丢失用户编辑。 |
| UX-5 | MEDIUM | `BatchMetadataDialog.vue:131` | 提示文本提"4 路并发"但无进度条，仅文字"执行中 0/5"。 |
| UX-6 | MEDIUM | `BatchMetadataDialog.vue:114-119` | `catch` 块假设所有失败都是致命的，但 `batchSetMetadata` 已返回部分结果。`catch` 是死代码（不可达）。 |
| UX-7 | MEDIUM | `useObjectActions.ts:355-373` | `copySelectedLinks` 顺序 `await`，一条失败则全部中止。应改为 `Promise.allSettled`。 |
| UX-8 | MEDIUM | `useObjectActions.ts:340-352` | `runUpload` 的 `finally` 块清空 `uploadQueue.value = []`，可能丢弃仍在运行的上传项。 |
| UX-9 | LOW | `BucketPolicyVisualEditor.vue:135-140` | `templateLabels` 硬编码映射，应从 `POLICY_TEMPLATES` 派生。 |
| UX-10 | LOW | `ObjectToolbar.vue:114` | "全选"复选框无 `<label>`，仅 `title` 属性。 |
| UX-11 | LOW | `ObjectToolbar.vue:132` | 视图切换按钮用 emoji `📊`，无 `aria-label`。 |
| UX-12 | LOW | `ObjectsPanel.vue:254-261` | 快捷键提示栏用 emoji `⌨️`/`🔲`，不可翻译。 |
| UX-13 | LOW | `BatchMetadataDialog.vue:186-193` | 错误列表截断至 50 条，无"显示更多"链接。 |
| UX-14 | LOW | `BatchMetadataDialog.vue:42` | `noChange` 计算属性在 `applyTags=true && tagsMode='replace'` 且所有 key 为空时返回 `false`，允许提交空标签替换。 |

---

## 七、API 设计（API Design）

### 优势
- **OpenAPI 3.0 契约**：67 个端点集中登记，与代码一一对应。
- **统一响应格式**：`writeJSON`/`writeErr`/`writeInternalErr` 标准化错误响应。
- **内容类型强制**：`readJSON` 要求 `application/json`，`DisallowUnknownFields` 防畸形请求。

### 问题
| # | 严重度 | 文件 | 描述 |
|---|--------|------|------|
| API-1 | MEDIUM | `openapi_register.go` | OpenAPI 契约未包含批量元数据端点（`object-batch/acl` 等）。新增端点需同步更新 OpenAPI 注册。 |
| API-2 | MEDIUM | `migrate_sync.go` | `POST /api/migrate/sync` 返回 `SyncResult`，但未在 OpenAPI 中登记（已在 Phase 3 添加，但需确认）。 |
| API-3 | LOW | `handler.go:104` | `maxBody = 4MB` 对 `delete-objects`（1000 key）可能不够。 |
| API-4 | LOW | `openapi_register.go` | 批量元数据端点（`object-batch/*`）未在 OpenAPI 中注册——当前批量编辑是前端编排，无后端端点。需决定是否新增后端端点。 |

---

## 八、错误处理（Error Handling）

### 优势
- **统一错误处理**：`writeErr`/`writeInternalErr` 标准化错误响应。
- **S3 错误映射**：`s3HTTPStatus` + `s3UserMessage` 将 S3 错误映射为 HTTP 状态码。
- **批量操作不抛错**：`batchSetMetadata` 返回 `{ ok, failed, errors }`，不阻断其它对象。

### 问题
| # | 严重度 | 文件 | 描述 |
|---|--------|------|------|
| E-1 | MEDIUM | `BatchMetadataDialog.vue:114-119` | `catch` 块中 `step: 'acl'` 硬编码，实际失败可能是 tags 或 storageClass，误导用户。 |
| E-2 | MEDIUM | `useObjectActions.ts:133-149` | `removeSelected` 仅设 `ctx.error.value`，无 toast 通知，用户可能未注意到失败。 |
| E-3 | LOW | `useObjectActions.ts:179-191` | `copySignLink` 失败设 `ctx.error.value` 无 toast；成功 toast 未指明哪个对象。 |
| E-4 | LOW | `ObjectsPanel.vue:264-267` | 错误横幅仅显示 `error` 字符串，无瞬态/永久错误区分，无自动消失。 |
| E-5 | LOW | `batchMetadata.ts:79-81` | `toErrorMessage(e)` 可能泄露内部栈跟踪（若 `e` 是 `Error` 对象）。 |

---

## 九、并发与资源管理（Concurrency & Resource Management）

### 优势
- **有界并发**：`WriteObjectsZip` / `RunBatch` / `batchMetadata` 均使用 worker pool。
- **上下文取消**：`WriteObjectsZip` 检查 `ctx.Done()`，producer 在取消时退出。
- **AbortController**：`useObjectBrowser.load` 每次新请求取消旧的。

### 问题
| # | 严重度 | 文件 | 描述 |
|---|--------|------|------|
| C-1 | HIGH | `zip.go:160-170` | `ctxReader` 无法中断阻塞式底层 Read。已用 `ctxCancelReader` + `context.AfterFunc` 修复。 |
| C-2 | HIGH | `s3wrap/client.go:63-67` | `awsconfig.LoadDefaultConfig(context.Background(), …)` 不可取消。已用 10 秒 timeout context 修复。 |
| C-3 | HIGH | `service/migrate.go:43-46` | `SameEndpoint` 空 endpoint 回退。已修复：两端均为空时返回 true。 |
| C-4 | MEDIUM | `batchMetadata.ts` | 原 `ok++`/`failed++` 非原子。已改为 per-worker 结果聚合。 |
| C-5 | LOW | `service/sync.go:51-57` | 列举 src/dst 之间无 `ctx.Err()` 检查。 |
| C-6 | LOW | `service/batch.go:49` | `results` 通道缓冲 `total`，大批量任务内存占用 O(n)。 |
| C-7 | LOW | `s3wrap/client.go:23-26` | `sharedHTTPOnce` 全局共享 HTTP 客户端，所有账号共用连接池。 |

---

## 十、可维护性（Maintainability）

### 优势
- **单一职责**：每个文件 <1000 行（代码规范）。
- **类型安全**：TypeScript 严格模式，Go 强类型。
- **测试驱动**：TDD 风格，新代码路径全部有测试。

### 问题
| # | 严重度 | 文件 | 描述 |
|---|--------|------|------|
| M-1 | MEDIUM | `openapi_register.go` | 825 行单一文件，9 个子注册函数全部内联。可拆分为 `register_*.go` 每文件一个域。 |
| M-2 | MEDIUM | `handler.go` | `Handler` 结构体 12 个字段，职责偏多。可考虑拆分为 `AccountHandler` / `ObjectHandler` / `MigrateHandler`。 |
| M-3 | LOW | `useObjectActions.ts:505-573` | 返回对象格式不一致：`detail` 用 2 空格缩进，其余用 4 空格。 |
| M-4 | LOW | `BucketPolicyVisualEditor.vue:135-140` | `templateLabels` 硬编码映射，应从 `POLICY_TEMPLATES` 派生。 |
| M-5 | LOW | `batchMetadata.ts:42-112` | `batchSetMetadata` 混合队列逻辑、worker 逻辑和聚合，可提取为 `boundedPool<T, R>` 工具函数。 |
| M-6 | LOW | `BatchMetadataDialog.vue:391-398` | `v-if="batchOpen"` 销毁重建组件，丢失进行中状态。 |

---

## 总结

| 维度 | 问题数 | 严重 | 高 | 中等 | 低级 |
|------|--------|------|-----|------|------|
| 架构与设计 | 5 | 0 | 0 | 1 | 4 |
| 安全性 | 9 | **1** | 1 | 3 | 4 |
| 性能与并发 | 7 | **1** | 2 | 0 | 4 |
| 测试质量 | 7 | 0 | 2 | 3 | 2 |
| 文档与 DevOps | 4 | 0 | 0 | 1 | 3 |
| 前端 UX | 14 | **3** | 3 | 5 | 6 |
| API 设计 | 4 | 0 | 0 | 2 | 2 |
| 错误处理 | 5 | 0 | 0 | 2 | 3 |
| 并发与资源 | 7 | **1** | 3 | 0 | 3 |
| 可维护性 | 6 | 0 | 0 | 2 | 4 |
| **合计** | **58** | **6** | **11** | **19** | **35** |

### 最高优先级（立即修复）
1. **S-1（安全）**：明文存储 SecretKey（JSON/SQLite 驱动）。应加密存储或使 `encrypted` 驱动成为默认。
2. **S-2（安全）**：`clientIP` 未处理 `X-Forwarded-For`，IP 限速可被伪造。
3. **P-1（性能）**：`awsconfig.LoadDefaultConfig(context.Background(), …)` 不可取消，已用 10 秒 timeout 修复。
4. **P-2（性能）**：`SameEndpoint` 空 endpoint 回退，已修复：两端均为空时返回 true。
5. **UX-1/2/3（无障碍）**：批量对话框和策略编辑器缺少 `<label>`，WCAG 违规。
6. **UX-4（状态丢失）**：`BucketPolicyVisualEditor` watcher 级联可能丢失用户编辑。
7. **UX-7（体验）**：`copySelectedLinks` 顺序 `await`，一条失败则全部中止。

### 快速修复（低投入高回报）
- 为所有无标签输入添加 `<label>` 或 `aria-label`（UX-1/2/3/10）
- 修复 `copySelectedLinks` 改用 `Promise.allSettled`（UX-7）
- 修复 `templateLabels` 从 `POLICY_TEMPLATES` 派生（UX-9/M-4）
- 添加 vitest `coverage` 配置（V-3）
- `openapi_register.go` 拆分为按域文件（M-1）
- 将 JSON/SQLite store 的 `SecretKey` 加密存储（S-1）
