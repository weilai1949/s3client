# apps/server 优化评审快照（2026-10-12）

> **评审时点**：2026-10-12 ｜ **评审对象**：[`apps/server`](../../apps/server/)（Go 后端 + 静态托管；分层 `config/atomicfile → model → store → s3wrap → service → handler → openapi`，见 [architecture.md](../architecture.md)）
> **结论**：**无结构性 / 算法级优化缺口**——`go vet` 0、`golangci-lint` **0 issues**、`go test ./...` 的 **9 个 `internal/*` 包全绿**。登记 **6 项微优化候选（S1–S6）**，均为**真实但收益有限**的取舍，另记 1 项**跨子树**门禁现状（S0，非本子树引入）。本快照本身**不含代码改动**，只固化结论与复现口径。
> **性质**：时点性**子树评审快照**——命名先例 [desktop-review-2026-10-12.md](desktop-review-2026-10-12.md)（apps/desktop，同一评审日）。条目全部闭环后按 [archive/index.md](index.md) 的「归档四步」`git mv` 进 `docs/archive/` 冻结。
> **证据口径**：本文件所有 `file:line` 均为评审当天实读所得；§3 的命令与输出为实跑原文（含失败 / 无结论的项，不代偿、不美化）。
> **归档冻结（2026-10-12）**：本快照条目——**S1–S5 全部闭环**（由同一批次「apps/server 优化评审 S1–S5 闭环」落地，见 [`FEATURES.md`](../FEATURES.md) §CN、[`PERFORMANCE.md`](../PERFORMANCE.md) §3.5 与 [`CHANGELOG.md`](../../CHANGELOG.md) `[Unreleased]`）、**S0 已解除**（同日 `apps/web` 批次同步 [`i18n.md`](../i18n.md)，根包 `TestI18nModuleInventoryMatchesDoc` 转绿）、**S6 维持现状**（无代码改动）——已按「归档四步」`git mv` 冻结入 `docs/archive/`。
> **落地实况与评审拟改动的差异**（正文按归档纪律**不回写**，保留评审当天的判断与实测数字）：
> - **S1**：拟稿写「有界缓冲池」，落地为 [`internal/service/stream_copy.go`](../../apps/server/internal/service/stream_copy.go) 的 `CopyBuffered`（256 KiB 定长缓冲 + 容量 2 的 channel 池，`maxIdleCopyBufs`，与 `maxIdlePartBufs` 同款「可证明有界」口径），handler 的 `copyStream` 与 ZIP 打包共用；写 deadline 刷新按 `writeDeadlineRefreshInterval = 30s` 节流（远小于 `streamIdleTimeout` 5min，慢网大文件不被误杀，由「间隔置 0 每次刷新」的用例钉住续期语义）。
> - **S2**：落地为 16 个分片（`rateLimitShards`），且**补上旧实现没有的硬上限**：单分片超过 `maxBucketsPerShard`（=10000/16=625）时先清过期桶、仍超限再淘汰任意桶，总量硬上限 10_000 与分片前等值（拟稿只要求「阈值同步收窄」，实测旧实现「超过 10_000 才尝试清理」对全活跃 IP 并无硬上限）。
> - **S3**：直方图桶与 `Calls` 改为 `atomic` 增量、`Calls` 由桶计数**求和**得出（`+Inf == _count` 结构性成立），错误码分类仍用互斥锁（失败路径、低频）。
> - **S5**：`runtime.ReadMemStats` 换成 `runtime/metrics` 的 `/memory/classes/heap/objects:bytes`（无 STW），响应先写 `bytes.Buffer` 再一次 `Write`（新增 `TestMetricsBufferedSingleWrite` 断言 `Write` 调用次数为 1）。
> - **S4** 与拟改动一致：`buildCSP` 纯函数 + `Handler.cspHeader` 预计算，零值 Handler 回退 `defaultCSPHeader`。
> - **门禁**：`go vet` 0 / `golangci-lint` **0 issues** / `go test ./...` 全绿 / 覆盖率门禁 **100.0% 且零未覆盖块**（`go test -race -count=1 -coverprofile` 复跑）。

## 1. 结论摘要

| # | 项 | 类型 | 证据位置 | 状态 |
|---|---|---|---|---|
| S0 | 当日工作区：根包 i18n 文档数字门禁红（同日 `apps/web` 在办批次新增 i18n key，`docs/i18n.md` 未同步） | 跨子树（**非** `apps/server` 代码） | `apps/server/doc_number_gate_test.go` 的 `TestI18nModuleInventoryMatchesDoc` | ✅ 已解除（web 批次同步 `docs/i18n.md`；见冻结横幅） |
| S1 | 流式下载 / ZIP 打包走 `io.Copy` 默认 **32KB** 缓冲，且**每次写**都刷新写超时 → GB 级传输的写调用数偏高 | 优化（吞吐 / syscall 数） | [`internal/handler/stream.go`](../../apps/server/internal/handler/stream.go) 第 51 行 · 第 84–85 行 · [`internal/service/zip.go`](../../apps/server/internal/service/zip.go) 第 133 行 | ✅ 已闭环（冻结横幅） |
| S2 | 限速器 `ipLimiter` 用**一把全局 `sync.Mutex`**：所有 `/api/` 请求串行过锁 | 优化（并发争用） | [`internal/handler/ratelimit.go`](../../apps/server/internal/handler/ratelimit.go) 第 18–35 行 | ✅ 已闭环（冻结横幅） |
| S3 | S3 调用指标 `observe` 每次调用取**全局互斥锁**，且每次调用前后各调一次 `time.Now()` | 优化（并发争用） | [`internal/s3wrap/metrics.go`](../../apps/server/internal/s3wrap/metrics.go) 第 114 行 · 第 192–194 行 | ✅ 已闭环（冻结横幅） |
| S4 | 安全头中间件**每请求**拼接 CSP 字符串（内容恒定，可预计算） | 优化（每请求分配） | [`internal/handler/middleware.go`](../../apps/server/internal/handler/middleware.go) 第 20 行 · 第 32 行 | ✅ 已闭环（冻结横幅） |
| S5 | `/api/metrics` 用 `runtime.ReadMemStats`（**Stop-The-World**）并多次 `fmt.Fprintf` 直写响应 | 优化（采集开销） | [`internal/handler/metrics.go`](../../apps/server/internal/handler/metrics.go) 第 68 行 · 第 70 行 | ✅ 已闭环（冻结横幅） |
| S6 | 低优先 / **不建议**单独动手：`/api/openapi.json` 每请求拷贝已缓存 spec；`clientCache` 无容量上限；`s3wrap.New` 每次缓存 miss 调 `awsconfig.LoadDefaultConfig` | 说明（避免后来者重复排查） | [`internal/openapi/openapi.go`](../../apps/server/internal/openapi/openapi.go) 第 269 行 · [`internal/handler/client_cache.go`](../../apps/server/internal/handler/client_cache.go) 第 26 行 · [`internal/s3wrap/client.go`](../../apps/server/internal/s3wrap/client.go) 第 86 行 | ➖ 维持现状 |

## 2. 逐项发现

### S0 当日根包门禁唯一失败项：i18n 文档数字（跨子树）

- **现象**：`cd apps/server && go test . -count=1` 唯一失败的是 `TestI18nModuleInventoryMatchesDoc`——同日 `apps/web` 在办批次新增了若干组件（`ActivityCenter` 等）的 i18n key，源码键数与 [`i18n.md`](../i18n.md) §1 / §2 / 末尾统计口径记录的数字不再一致。
- **归属**：门禁代码在 `apps/server`（根包 `*_gate_test.go`），但它**读的是 `apps/web` 的字典**——这是「文档 ↔ 源码数字」的跨子树契约，修复点在 `docs/i18n.md`，属该 web 批次（与 `apps/server` 生产代码无关）。
- **判定**：不属本次 `apps/server` 优化范畴；此处登记只为说明「当日 `apps/server` 门禁并非全绿，且失败项不由本子树引入」。
- **拟改动**：由 web 批次同步 `docs/i18n.md`（§1 的 key 总数、§2 模块表、末尾统计口径三处）——注意本轮工作区键数仍在变动，改前先重跑该门禁取真值。

### S1 流式下载 / ZIP 打包：32KB 通用缓冲 + 每写刷新 deadline

- **现象**：[`internal/handler/stream.go`](../../apps/server/internal/handler/stream.go) 的 `copyStream`（第 51 行）用 `io.Copy(dw, &contextReader{…})`；`dw` 是 `*deadlineWriter`（第 84–85 行，**每次都 `SetWriteDeadline`**），源被 `contextReader` 包装。两侧都不实现 `io.WriterTo` / `io.ReaderFrom` ⇒ `io.Copy` 落入 32KB 通用缓冲。[`internal/service/zip.go`](../../apps/server/internal/service/zip.go) 第 133 行对**每个** ZIP 条目同样走裸 `io.Copy`。
- **依据**：§3 ③ 的独立微基准（不改仓库）——同负载下 64 MiB：默认 `io.Copy` = **2048 次写**（平均 32768 B/次），`io.CopyBuffer`(256 KiB) = **256 次写**（平均 262144 B/次），即写调用数降为 1/8；真实路径每次写还额外付一次 `SetWriteDeadline`。
- **判定**：真实但收益有限——只在 **GB 级**大文件下载 / 大 ZIP 打包时可见，小响应无差别。
- **拟改动**：改用 `io.CopyBuffer` + **有界**缓冲池（沿用 [`internal/service/stream_copy.go`](../../apps/server/internal/service/stream_copy.go) 第 38 行 `maxIdlePartBufs` 的「可证明有界」口径，不要 `sync.Pool` 的无界语义）；并把写 deadline 刷新按「时间 / 字节阈值」节流（`streamIdleTimeout` = 5min，见 `stream.go` 第 17 行，留有很大余量）。
- **风险**：池驻留是**常量**内存，须计入 512MB 容器预算；放宽 deadline 刷新节奏必须验证慢网大文件不被误杀（否则会回归成「大文件下载被截断」）。

### S2 限速器：一把全局锁串行所有 `/api/` 请求

- **现象**：[`internal/handler/ratelimit.go`](../../apps/server/internal/handler/ratelimit.go) 的 `ipLimiter`（第 18–20 行）以单把 `sync.Mutex` 保护一张 `map[string]*tokenBucket`；`allow`（第 32–35 行）在锁内完成桶查找 / 前进 / 补充令牌。
- **放大点**：`withRateLimit` 挂在**鉴权外层**（`routes.go` 的组合串），因此**每一个** `/api/` 请求都会进这个临界区。
- **判定**：临界区极短（纳秒级），只有在「高并发 + 多核」下才成为可测瓶颈；作为管理面工具，典型负载无感。
- **拟改动**：按 IP 分片（如 16 / 64 个 shard，各自锁 + 各自 map），或改 `sync.Map`；**保持有界性**——现「`len(byIP) > 10_000` 触发 reap」的内存上限在分片后变成 `分片数 × 阈值`，需同步收窄单分片阈值以守住总量。

### S3 S3 调用指标：全局锁 + 双 `time.Now()`

- **现象**：[`internal/s3wrap/metrics.go`](../../apps/server/internal/s3wrap/metrics.go) 的 `s3Metrics.observe`（第 114 行）在 `mu.Lock()` 下更新 `calls` / `buckets` / `byCode`；`metricsMiddleware.HandleFinalize`（第 192–194 行）在调用前后各调一次 `time.Now()`。
- **放大点**：[`internal/service/sync.go`](../../apps/server/internal/service/sync.go) / `migrate` 的批量路径**逐 key** 发起多次 S3 调用，并发 worker 下所有在飞调用共享同一把锁。
- **判定**：同样是纳秒级临界区，且被 S3 网络往返（毫秒级）完全淹没——**收益比 S2 更小**，仅在高并发纯内网小对象场景理论上可见。
- **拟改动（若要做）**：计时桶改 `atomic` 数组（与 [`internal/handler/metrics.go`](../../apps/server/internal/handler/metrics.go) 的 `metricHTTPDurBuckets` 同款），`byCode` 保留锁或独立分片。建议与 S2 **合并评估、分开实施**（同源模式，风险与收益同构）。

### S4 安全头中间件：每请求拼接 CSP

- **现象**：[`internal/handler/middleware.go`](../../apps/server/internal/handler/middleware.go) 的 `withSecurityHeaders`（第 20 行）在**每个**响应上执行 `"default-src 'self'; …; connect-src "+connect+"; …"` 拼接（第 32 行）。
- **判定**：`cspConnectSrc` 在 `Routes()` 前设定、之后恒定（`SetCSPConnectSrc` 只在启动期可调用），故整串 CSP 是**不变量**；每请求拼接 = 1–2 次分配 + 常量拼接。
- **拟改动**：在 `New` / `SetCSPConnectSrc` 时预计算整串，中间件只 `Header().Set`。属「顺手可做」，非必须。

### S5 `/api/metrics`：`ReadMemStats` 的 STW 与多次直写

- **现象**：[`internal/handler/metrics.go`](../../apps/server/internal/handler/metrics.go) 第 70 行每次抓取都调 `runtime.ReadMemStats`——该调用会**暂停所有 goroutine**；其上还有 30+ 次 `fmt.Fprintf` 直接写 `http.ResponseWriter`。
- **判定**：抓取频率通常是秒级（Prometheus 15–30s），STW 时长为微秒～毫秒量级，对业务无实质影响；属「规范化 / 降低采集副作用」，不是救火项。
- **拟改动**：改用 `runtime/metrics`（无 STW）读等价指标（如 `/memory/classes/heap/objects:bytes`）；响应先写入单个 `bytes.Buffer` 再一次性 `Write`，减少多次写与 `Content-Type` 之外的重复调用。

### S6 已核对：**不建议**单独动的三处

- **`/api/openapi.json` 每请求拷贝已缓存 spec**（[`internal/openapi/openapi.go`](../../apps/server/internal/openapi/openapi.go) 第 269 行的 `MarshalJSON` 双检锁）：spec 只构建一次并缓存，请求侧 `copy` 是**刻意的缓存隔离**（防调用方改写污染缓存）；该端点流量极低。删拷贝会引入数据竞争风险，收益 ≈ 0 ⇒ 维持现状。
- **`clientCache` 无容量上限**（[`internal/handler/client_cache.go`](../../apps/server/internal/handler/client_cache.go) 第 26 行）：条目数 = 账号数（人工维护，量级 < 百），且账号增删改会 `evict`。加 LRU 是**过度设计** ⇒ 维持现状。
- **`s3wrap.New` 每次缓存 miss 调 `awsconfig.LoadDefaultConfig`**（[`internal/s3wrap/client.go`](../../apps/server/internal/s3wrap/client.go) 第 86 行）：只在账号新建 / 变更（缓存 miss）时触发，不在请求热路径；10s 超时上下文是**防不可达 endpoint 阻塞**的刻意设计 ⇒ 维持现状。

## 3. 复现与验证口径（实跑记录）

```bash
# ① 基线（cwd = apps/server/，评审当天）
go vet ./...                              # 无输出（0 issues）
golangci-lint run                         # 0 issues.
go build ./...                            # OK
go test ./... -count=1                    # internal/{atomicfile,config,handler,model,
                                          #   openapi,s3wrap,service,store,tracing} 9 包全 ok；
                                          #   根包唯一失败 = S0（见 ②）

# ② 当日根包唯一失败项（S0，属 web 批次文档同步）
go test . -count=1 -run TestI18nModuleInventoryMatchesDoc
#   失败点名：docs/i18n.md 声称的 key 总数 / §2 模块表 与 apps/web 源码实测不符

# ③ 既有性能基线复跑（结果与 docs/PERFORMANCE.md 的记录同量级）
go test ./internal/service/ -run '^$' -bench 'Benchmark(RelKey|BaseKey)$' -benchmem
#   BenchmarkRelKey/段边界命中-16   77613883   15.50 ns/op   0 B/op   0 allocs/op
#   BenchmarkRelKey/深前缀-16       72206844   14.75 ns/op   0 B/op   0 allocs/op
#   BenchmarkRelKey/含中文与空格-16  35132247   35.40 ns/op  48 B/op   1 allocs/op
#   BenchmarkBaseKey/段边界命中-16   45949402   29.80 ns/op  16 B/op   1 allocs/op
#   BenchmarkBaseKey/无前缀-16       37673164   32.75 ns/op  24 B/op   1 allocs/op

# ④ S1 取证：/tmp 下独立微基准（**不改仓库**）——同负载的写调用数对比
#    io.Copy（默认缓冲）：          67108864 字节 → 2048 次写（avg 32768 B/write）
#    io.CopyBuffer(buf 256 KiB)：   67108864 字节 →  256 次写（avg 262144 B/write）
```

## 4. 已核对、确认**无需**改动（避免后来者重复排查）

- **无 N+1**：分段上传走预签名直传——[`internal/handler/multipart.go`](../../apps/server/internal/handler/multipart.go) 的 `multipartPart` 只回一个预签名 URL，服务端**不代理**分片体；对象列表是**单次**分页调用（[`internal/handler/objects.go`](../../apps/server/internal/handler/objects.go) 第 26 行），非「每对象再发一次请求」。
- **并发与内存处处有界**：流式并发 32（`streamSlots`）、ZIP 抓取 4 worker、分段缓冲池上限 1 块 64MB（`maxIdlePartBufs`）、账号库全量常驻内存且 `Get` 为 O(1) map 命中。
- **列举有硬上限**：页数 / 对象数上限（`listPrefixMaxPages`、`storageReportMaxObjects` 等）保证恶意对端返回「不前进的 nextToken」时空转被截断并标记 `truncated`。
- **OpenAPI spec 已缓存**：`Registry.MarshalJSON` 双检锁，`/api/openapi.json` 不重新构建规范（仅做一次缓存拷贝，见 S6）。
- **客户端的失效语义正确**：`clientCache` 按 `Account.UpdatedAt` 比对失效，账号增删改走 `evict`，不会用到陈旧凭据。
- **门禁零告警**：`go vet` / `golangci-lint`（`unused` / `staticcheck` / `gosec` / `nolintlint`）/ 根包死代码门禁 [`deadcode_gate_test.go`](../../apps/server/deadcode_gate_test.go) 均为 0——本子树**无死代码可清**（与同日的 desktop 子树不同）。
- **既有基准无退化**：`RelKey` / `BaseKey` 与 [`PERFORMANCE.md`](../PERFORMANCE.md) 记录同量级（15–35 ns/op，0–1 alloc/op）。

## 5. 刻意不做

- **不引入新依赖**：S2 / S5 用手写分片与 stdlib（`sync` / `runtime/metrics`）即可，不引第三方 rate limiter / Prometheus client——本仓库「零第三方」是既有取舍（见 [decisions/](../decisions/index.md)）。
- **不动 640GB 流式复制上限**：`KNOWN_ISSUES #63` 已决策「➖ 维持现状」，放宽受 512MB 容器内存预算约束（`multipartPartSize × maxMultipartParts`），本快照不重开该结论。
- **不为 S1–S5 新增门禁**：它们是**性能取舍**，没有可机械判定的契约；强行加断言只会测到内部实现（违反 [DEVELOPMENT.md](../DEVELOPMENT.md) §1「行为驱动，不测内部实现」），故不设防复发门禁——与 desktop 的 G1（死依赖是**可机械判定**的事实）性质不同。

## 6. 未复跑 / 未实测（如实披露）

- **未做端到端压测**：S1–S5 的收益均为**定性**判断；S1 只取证了「写调用数」，**未**测真实吞吐 / 延迟 p99。
- **未采集 pprof**：S2 / S3 的「锁争用是否为可测热点」未在真实负载下用 `pprof`（`mutex` / `block` profile）验证。
- **未做改动验证**：S1–S6 均**未**做任何验证性代码改动（本快照只读）。
- **未触发远端 CI**：全部证据为本地实跑，两套 CI 的实际结果以本批次推送后为准。

## 7. 登记归属与后续

- **待办唯一来源**：S1–S5 目前是**本文件**的待办条目（`apps/server` 优化候选）。按 [DEVELOPMENT.md](../DEVELOPMENT.md) §4 的两源分工，**未闭环**的问题（含技术债）的唯一来源是 [`KNOWN_ISSUES.md`](../KNOWN_ISSUES.md)——若本批次内**不**落地，应在同 PR 转登 `KNOWN_ISSUES.md` 并在上表状态列标「转登记」；落地则随代码写入 [`CHANGELOG.md`](../../CHANGELOG.md) `[Unreleased]` 与 [`FEATURES.md`](../FEATURES.md)。
- **S0 归同日 `apps/web` 批次**：修复点是 `docs/i18n.md` 的数字同步，不建议在 `apps/server` 侧改动。
- **闭环后归档**：本文件为时点性快照，S1–S5 全部闭环（或全部转登记）后，按 [archive/index.md](index.md) 的「归档四步」`git mv` 进 `docs/archive/` 冻结。
