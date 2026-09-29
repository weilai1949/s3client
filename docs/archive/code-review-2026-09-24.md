# 全面代码审查报告(2026-09-24)

> **处置状态(2026-09-24 收口)**:2 Critical + 20 Required(后端 15 + 前端 5)**全部修复**;
> Nit(正文两段共 **35 项**:后端 18 + 前端 17)——**31 项 ✅ 已修复**、**2 项 ⚠️ 转登记开放技术债 ⬜**
> (`SameEndpoint useSSL` → [`KNOWN_ISSUES.md`](../KNOWN_ISSUES.md) **#61**;batch 删除编排下沉 → **#62**,
> **本轮未完成**)、**1 项 ⚠️ 转登记已决策 ➖**(stream_copy 640GB 上限 → **#63**)、**1 项 ℹ️ 复核后判定不成立**
> (App.vue 双 `JSON.parse`)。逐条状态见正文各级标题的 ✅ / ⚠️ / ℹ️ 标记与本报告
> 「Nit」节末尾的**处置明细**表;证据台账见 [`FEATURES.md`](../FEATURES.md) **§AA**,
> 逐项改动见 [`CHANGELOG.md`](../../CHANGELOG.md) `[Unreleased]` 同日条目。
>
> 复测(全绿):后端 `gofmt` 干净 / `go vet` 0 告警 / `go test` **9/9 包**(`go list ./...` 9 个包,**每包 100.0% statements**,
> 含 R11 新建的 `internal/atomicfile`) / `go build` 干净 /
> `golangci-lint` **0 issues**;前端 `pnpm lint` 0 告警 / `pnpm test` **67 文件 1110 例** /
> `pnpm test:coverage` **四指标 100%(4255 / 2908 / 1124 / 3653)** / `pnpm build` + `typecheck:e2e` exit 0;
> 真实 E2E 两项实跑——`S3CLINET_E2E=1 go test ./internal/s3wrap/ -run 'TestE2E'` **4/4 PASS**、
> `make e2e-real` **3 passed**(后端 `S3C_TOKEN` 开启的生产同构形态,即 C1 要求的验收实跑)。
>
> 标记含义:**✅ 已修复** · **⚠️ 未按原样修复——转登记 [`KNOWN_ISSUES.md`](../KNOWN_ISSUES.md)(#61 / #62 开放 ⬜,#63 已决策 ➖)** · **ℹ️ 复核后判定非问题,不改**。
> 未完成项不静默略过:一律在 [`KNOWN_ISSUES.md`](../KNOWN_ISSUES.md) 留条目(本轮 #61 / #62 保持开放,可回溯、可排期)。
> 下方「门禁基线」与正文 file:line 为**审查时点快照**,按仓内「不追溯篡改」纪律保留原样。
>
> **后续状态更新（2026-09-28）**：本报告转登记的 **#61 / #62 已闭环**——`SameEndpoint` 精确判定纳入 `useSSL`
> （6 参签名 + `(*s3wrap.Client).UseSSL()`）、批量删除编排下沉 `service`（新增 `internal/service/delete.go`，
> `handler/objects.go` 538 → 432 行）；**#63 证据补齐**（新增 `TestMultipartStreamCopyPartSizeIs64MB` 钉住 64MB 分段），
> 决策仍为 ➖ 维持现状。三项收口见 [`FEATURES.md`](../FEATURES.md) **§AB** 与 [`CHANGELOG.md`](../../CHANGELOG.md)
> `[Unreleased]` 同日条目；[`KNOWN_ISSUES.md`](../KNOWN_ISSUES.md) §二 现只剩 #63。上文「开放 ⬜」等措辞为
> **2026-09-24 时点快照**，按「不追溯篡改」纪律保留原样。

> 范围:全仓**代码**(apps/server Go 后端 / apps/web Vue3 前端 / apps/desktop Tauri 壳),不含文档评审。
> 方法:机械化门禁(go vet / golangci-lint / go test -count=1 / eslint / vue-tsc / vitest,全部 0 issues)+ 6 路五轴深度审查(正确性/可读性/架构/安全/性能/死代码),全部发现经源码复核或实测验证。

## 门禁基线

| 门禁 | 结果 |
|---|---|
| `go vet ./...` / `go build ./...` | ✅ 通过 |
| `go test -count=1 ./...`(8 包,含 -race 抽查) | ✅ 全绿 |
| `golangci-lint run`(unused/staticcheck) | ✅ 0 issues |
| `eslint src --max-warnings 0` | ✅ 0 issues |
| `vue-tsc --noEmit` | ✅ 0 errors |
| `vitest run` | ✅ 66 文件 / 1042 用例全过 |

机械化门禁全绿,但以下问题均为工具无法捕获的语义级缺陷——覆盖率与 lint 达标 ≠ 无问题。

## 总体评价

代码库整体质量高于平均水平:分层契约(handler→service→s3wrap、store→model 单向无环,零 AWS SDK 类型泄漏)、竞态守卫(loadSeq/代次守卫)、SSRF 三层防护、zip-slip 防御、常量时间 token 比较、注释解释「为什么」等普遍到位。单文件最大 631 行(未破 1000 行红线)。

但审查发现 **2 Critical / 15 Required(后端)/ 5 Required(前端)/ 若干 Nit**,其中两个 Critical 均为安全/核心功能级。

---

## Critical(2)

### C1. ✅ 前端代理 URL 不携带凭证,启用 S3C_TOKEN 的部署下预览与下载全部 401
- 位置:`apps/web/src/proxy.ts:17-19`,波及 `useObjectActions.ts:205-213`(download)、`usePreview.ts:30-34`(inline 预览)、`PreviewOverlay.vue:96`、`CompareDialog.vue:127`
- `<a href>` / `<img>` / `<video>` / `<iframe>` 资源加载无法发送 Authorization 头;服务端 `withAuth`(middleware.go:209-243)对所有 `/api/*` 强制 Bearer 且无 query-token 兜底。docker-compose.yml:41 强制 S3C_TOKEN 非空(生产默认形态)。
- 影响:媒体/PDF 预览 401;「下载对象」把 401 JSON 错误体当文件保存——**静默给用户坏文件**。同一弹窗的文本预览正常(显式带 Bearer fetch),恰好证明其余路径漏了鉴权。e2e-real.sh:158-163 启动后端未设 token,故联调从未暴露。
- 修复:改用带 Authorization 的 fetch + blob objectURL(复用 downloadZipToDisk / 文本预览既有模式);或后端为 proxy 端点签发短期免头凭证。**禁止用豁免 proxy 鉴权来「修复」**。修完必须在 S3C_TOKEN 开启下跑 e2e-real。

### C2. ✅ `IsLoopbackAddr` 把 `:8080`(未指定 host)判为回环,非回环强制鉴权被绕过
- 位置:`apps/server/internal/config/config.go:180`
- `return host == "127.0.0.1" || ... || host == ""`——实测 `net.SplitHostPort(":8080")` 得 host="",而 `net.Listen(":18099")` 绑定 `[::]`(所有接口)。config.go:203「非回环必须 S3C_TOKEN」对最常见的通配写法失效:`S3C_ADDR=":8080"` 无 token 时账号管理 API(含 SecretKey 增删查、S3 代理)无鉴权暴露全部网卡。`0.0.0.0`/`[::]` 均被正确拦截,唯空 host 漏网。
- 修复:删掉 `|| host == ""`;补 `":8080"` 无 token 必须启动失败的测试。

---

## Required — 后端(15)

### 安全(4)

**R1.** ✅ `handler/ratelimit.go:83-88` | X-Forwarded-For 取**首段**(最左),可信代理部署下客户端可伪造 `X-Forwarded-For: 1.2.3.4` 让代理追加真实 IP 后首段为攻击者自选值 → per-IP 限速(120 req/min)完全绕过;audit.go:30 审计 IP 同被污染。修复:取最后一段(可信代理追加的)。gap_cover_test.go:78-79 固化了现行行为需同步改。

**R2.** ✅ `handler/routes.go:94` + `middleware.go:229-246` | 中间件顺序 `withAuth(withMetricsGate(withRateLimit(mux)))`——401 在 withAuth 直接写出,**鉴权失败请求不经过限速器**,Bearer token 可无限速暴力尝试(缓解:MinTokenLength=16 + 常量时间比较)。修复:withRateLimit 移到 withAuth 外层。

**R3.** ✅ `handler/objects.go:344-428`(deletePrefixAsync)、`copy.go:158-175`(copyManyAsync 移动模式) | 可删 10 万对象的破坏性异步操作**整个 goroutine 无 h.audit**(同步 deletePrefix 有审计,异步没有)。修复:写 202 前补审计事件(含 jobId)。

**R4.** ✅ `config.go:209` + `store/open.go:18` | StoreDriver 大小写/空白/未知值绕过「明文落盘」安全闸:Validate 用原始串比较,Open 却 `ToLower(TrimSpace)` 且未知值静默默认 json。`S3C_STORE_DRIVER="JSON"` + 无 STORE_KEY + 未 opt-in 明文时,ErrPlaintextStoreNotAllowed 被跳过仍启动。修复:FromEnv 归一化 + Validate 白名单拒绝未知值。

### 正确性(8)

**R5.** ✅ `handler/proxy.go:57` | mode=text 丢弃 versionId(46 行已解析,57 行传空串)——历史版本文本预览返回最新版本。inline/download 分支正确使用。修复:传 `versionID` + 契约测试。

**R6.** ✅ `service/sync.go:107-111,141-143` | 同步列举静默截断:listMaxTotal=100_000 命中时返回 scanned=100000、err=nil,SyncResult 无 truncated 字段(对比 copyPrefix 有)。源侧第 100001 个对象永不同步且无信号;目标侧超限对象每次同步被重拷。修复:命中上限报错或透出 Truncated。

**R7.** ✅ `service/batch.go:132` | RelKey 裸 `TrimPrefix` 缺段边界校验——sync.go:217-233 的 stripPrefix 已修复并测试过同一 P0(prefix="backup" 命中 "backups/x" → 错位/双斜杠),copy-prefix 路径仍保留第二个裸实现。修复:RelKey 改用 stripPrefix 作内核。

**R8.** ✅ `service/job.go:247` | Reap TTL 以 Created 而非完成时间为准:跑超 30 分钟的任务 Finish 后 ≤5 分钟即被清除(列表丢终态、轮询 404、jobs.json 记录消失)。修复:Job 加 finishedAt 并按其算 TTL。

**R9.** ✅ `service/job.go:380` | Emit 无终态保护:迟到的带 `Status:"running"` 帧(RunBatch 每帧都带)落在 Finish 后会把已完成任务改写为非终态 → 重启后 restore() 误标 interrupted,触发虚假对账告警。record():182-189 只兜底空 Status。修复:Emit 锁内 `if j.done { return }`。

**R10.** ✅ `service/migrate.go:49-56` | 同端点跨账号迁移仅 EntityTooLarge 回退流式;CopyObject 以源账号凭证签名、目标桶私有时每个 key 403 且无 AccessDenied 回退(流式各用各凭证本可成功)。两账号同一 MinIO/RustFS 是主场景。修复:AccessDenied 也回退 StreamCopy。

**R11.** ✅ `store/atomic.go:27-53` | 原子写缺 fsync:write→close→rename 全程无 `f.Sync()`,掉电后 rename 可能先于数据落盘,账号文件损坏/清空。`service/job_persist.go:135` 有同缺陷近重复实现(还无 O_EXCL、无短写检查)。修复:close 前 Sync + rename 后 fsync 父目录;两份实现收敛。

**R12.** ✅ `store/sqlite.go:93-94` | 建表与迁移错误被吞(`_, _ = db.Exec(sqliteSchema)`):坏盘下「启动成功、运行时全挂」。修复:错误上抛走启动失败路径。

**R13.** ✅ `store/sqlite.go:189` | `for rows.Next()` 后未检查 `rows.Err()`:迭代中断(如优雅关停并发 Close)时截断列表当成功返回,UI 静默丢账号。修复:循环后检查并包装返回。

### 性能/契约(3)

**R14.** ✅ `store/sqlite.go:171-190` | List() 每行跑一次 Argon2id(实测 34ms/64MiB)解密 SecretKey,随后 `Sanitized()` 立即脱敏丢弃——只打击启用加密的生产配置,20 账号 ≈0.7s/次。修复:List 查询不解密(SELECT 合成 secretSet)。

**R15.** ✅ `main.go:110-136` + `openapi/openapi.go:465-485` + `handler/handler.go:109` | 三项契约违背:(a) ListenAndServe 失败(端口占用)退出码 0,与自述契约矛盾,systemd/Docker on-failure 不重启(main_test.go:196-212 钉死 want 0 需同步改);(b) components.responses 的 JSON/Ref 字段被 `json:"-"` 静默丢弃,契约 SSOT 里共享响应引用解析到无 schema 空壳(实测验证);(c) objectItem.ContentType 恒为空串而 OpenAPI/前端契约声明该字段。

### s3wrap(2)

**R16.** ✅ `s3wrap/client.go:87-99` | presign client 同样挂 metricsMiddleware(实证:一次 PresignPut 使 Calls delta=1)。PresignUploadPart 每段预签名一次——N 段浏览器直传注入 N 个假「S3 调用」,p99 塌向 0,污染 s3c_s3_calls_total 语义。修复:presign client 不注册 metricsMiddleware。

**R17.** ✅ `s3wrap/errors.go:81-101` | HTTPStatus 未映射 ErrPartialDelete:PurgeObject 部分删除被报成通用 500,精心映射的 UserMessage 生产不可达,已删除计数一并丢失。修复:HTTPStatus 补 `errors.Is(err, ErrPartialDelete) → 409`。

### 死代码(违反仓库硬约束 #5)(3)

**R18.** ✅ `handler/metrics.go:32-37` expvar.Publish 两指标生产不可达(无 /debug/vars 挂载,唯一读取方是测试)——AGENTS.md 警告的「被覆盖率掩盖的死代码」。
**R19.** ✅ `handler/presign.go:8-9` errTestPresign 哨兵仅测试引用却放生产文件;`service/job.go:267` JobRegistry.Create 生产零引用(生产全走 newJob→TryCreate);`model/account.go:81` BucketOrDefault 恒等函数;`store/crypto.go:63-65` deriveKeyLegacy 生产零引用、`crypto.go:88-91` envelope V2 分支生产不可达。
**R20.** ✅ `handler/routes.go:37` 同步 copy-objects(10,000 keys × 每 key 一次 CopyObject)未挂 withStreamLimit,同类长耗时端点均挂——缺失 32 槽上限与 503 背压。

---

## Required — 前端(5)

**F1.** ✅ `api/storage.ts:290-299` | setTokenPersistent 只处理全局 `s3c.token`,完全忽略 per-server 的 `s3c.token.<id>`:开启持久化不迁移已有 token(重启即丢),关闭持久化不清 LS 残留(token 永久在盘,与 api/index.ts:44 自述契约矛盾)。
**F2.** ✅ `composables/useUploadQueue.ts:77-135` | requeue 模式(UploadPanel)无法取消已入批未开始条目:worker 消费 selectBatch 快照,requeue 的 abortItem 对 pending 条目是 no-op——UI 移除/清空列表后文件仍在后台上传。
**F3.** ✅ `components/MigratePanel.vue:571` | 虚拟列表 CSS 行高 38px 与 ROW_HEIGHT=42 漂移——virtualList.ts:5 明载「曾为 38 导致滚动窗口错位(review §F9③)」的**已修复 bug 回归**。OVERSCAN=12 恰好吸收错位,20 万对象时滚动总高度高估 ~10%。修复:改 42px,更稳妥用 CSS 变量绑定 ROW_HEIGHT。
**F4.** ✅ `components/UploadQueue.vue:38` | v-for `:key="bucket|key"` 组合键可碰撞(两个目录选同名文件);useUploadQueue 已分配唯一自增 id 未用。修复:`:key="it.id"`。
**F5.** ✅ 四组件声明了从未 emit 的 `error` 事件且父组件挂了处理器(PreviewOverlay.vue:28 / CompareDialog.vue:33 / ObjectDetailDialog.vue:22 / BucketPolicyVisualEditor.vue:23);ObjectDetailDialog 的 accountId/bucket props 零使用;BatchMetadataDialog.vue:146 `void s3api` 压 lint shim;AccountsPanel/CreateBucketDialog/HeadersDialog/LifecycleDialog 四处异步提交无防重复守卫(双击=重复 createAccount/createBucket)。

---

## Nit(摘选,共 30+ 项)

> **处置状态**:正文两段按语义归属共 **35 项**(后端 18 + 前端 17;其中 `withDefaults` 空 no-op、
> MigratePanel 空 if 两项原排在「后端」段、实为前端)——**31 项 ✅ 已修复**、**2 项 ⚠️ 转登记开放技术债**
> (`SameEndpoint useSSL` → #61、batch 删除编排下沉 → **#62 本轮未完成**)、**1 项 ⚠️ 转登记已决策**
> (stream_copy 640GB 上限 → #63)、**1 项 ℹ️ 复核后判定不成立**(App.vue 双 `JSON.parse`)。逐项见下方「处置明细」。

后端:ssrf.go:48 `%w` 格式化 nil err 产出 `%!w(<nil>)`;errors.go RequestTimeout 两表分类不一致;IsEntityTooLarge 冗余文案匹配;sqlite WAL/-shm 侧车 0644(实测);DSN 无 busy_timeout;parseEnvelope KDF 参数无上界(损坏文件 4TiB OOM);envOrInt 静默吞非法值;healthcheck 注释与行为不符;migrate 系列把 store 故障误报 404;getBucketInfo 串行 3 次上游调用(ListBuckets 拉全量只为取一个桶);copyMany DeleteSource 分支响应形状与 copyBatchJSON 漂移;sync etag 模式对 multipart 对象永不收敛(每次重拷);SameEndpoint 硬编码 useSSL=false;stream_copy 固定 64MB×10000=640GB 上限;LastError 名不副实(实为 FirstError);`withDefaults(defineProps, {})` 空 no-op;MigratePanel 空 if 分支;XFF 之外的 CSP connect-src 字面量两处维护;EnumStr 冗余 string() 转换;batch 删除编排留在 handler(同类均在 service)。

前端:upload.ts:75 shouldUseMultipart 生产零引用;五处导出面死代码(updateToast/applyTheme/UPLOAD_CONCURRENCY/polling/normalizeStringArray,恰在项目自设门禁声明盲区内);requestTab 盲断言 TabKey;regions.ts `?? []` 恒不可达分支;storage.ts 模块内降级策略不一致(getBase/setBase 裸奔);App.vue 渲染热路径双重 JSON.parse;分页大小 100 在 toast 文案二次硬编码;abort 后 Promise 永不 settle(帧泄漏);0 字节无 MIME 文件被当目录占位丢弃;validateDoc 硬编码中文错误串破坏 i18n;7 处可删除行 v-for 用 index 作 key;账号回退初始化逻辑逐字复制(应提取 composable);RecycleBinPanel/VersionsDialog 大表无虚拟滚动;selectedSize computed O(n) 全表扫;`t` 局部变量遮蔽 i18n `t`。

**非问题澄清**(曾怀疑、已排除):crypto/rand nonce 忽略错误——go 1.26 契约 never fails,注释判断成立;dialContextSSRF 空 DNS 结果——实测 resolver 对 NXDOMAIN 返回错误而非空列表,不可达;SSRF 防护、zip-slip、Content-Disposition 注入防护、路径穿越防护均逐项验证通过。

### 处置明细

> 上面两段是**审查时点的原始清单**(快照,不回写);本表是**收口后的逐项处置**,与顶部状态块同口径。

**后端(18)**

| # | 项 | 处置 | 实现与证据 |
|---|----|------|------------|
| B1 | `ssrf.go:48` `%!w(<nil>)` 污染错误串 | ✅ | 无主机名分支改 `fmt.Errorf("invalid endpoint URL: %q (missing host)", endpoint)`,与「解析失败」的 `%w` 分支分开 |
| B2 | `RequestTimeout` 两表分类不一致 | ✅ | `s3wrap/errors.go` 两处 `case` 统一含 `RequestTimeout`;[errors.md](../errors.md) 同步 |
| B3 | `IsEntityTooLarge` 冗余文案匹配 | ✅ | 改「只做结构化判定」(应用层 sentinel / S3 结构化错误码),删字符串包含判断 |
| B4 | sqlite WAL / `-shm` 侧车 0644 | ✅ | 建库后侧车文件 chmod `0600`,与主库文件一致 |
| B5 | DSN 无 `busy_timeout` | ✅ | DSN 补 `busy_timeout(5000)`,避免写锁竞争下立即 `database is locked` |
| B6 | `parseEnvelope` KDF 参数无上界 | ✅ | KDF 参数加上界(默认 10 秒 / 相应内存上限),损坏文件不再 4TiB OOM |
| B7 | `envOrInt` 静默吞非法值 | ✅ | 返回 `ErrInvalidEnvValue`,启动日志显式报错 |
| B8 | healthcheck 注释与行为不符 | ✅ | `main.go` `runHealthcheck` 注释改写为与实现一致的 fail-closed 口径 |
| B9 | migrate 系列把 store 故障误报 404 | ✅ | 仅 `errors.Is(err, store.ErrNotFound)` → 404,其余 500 |
| B10 | `getBucketInfo` 串行 3 次上游调用 | ✅ | 桶不存在即短路返回空属性;不再 `ListBuckets` 拉全量;versioning / 标签并行取 |
| B11 | `copyMany` `DeleteSource` 分支响应形状漂移 | ✅ | 移动与纯复制共用 `copyBatchJSON`,`truncated` 恒写 `false` 并注释 openapi 契约要求 |
| B12 | sync `etag` 模式对 multipart 对象永不收敛 | ✅ | `service/sync.go` 新增 `isMultipartETag` 判定,分段 ETag 回退 size 比对;[sync_multipart_etag_test.go](../../apps/server/internal/service/sync_multipart_etag_test.go) |
| B13 | `SameEndpoint` 硬编码 `useSSL=false` | ⚠️ **#61** | 精确判定需改 `SameEndpoint` 跨包签名与全部调用点;现有归一化对称(裸端点恒按 http 互比)且失败方向偏安全。口径说明写入 `service/migrate.go` 注释 + 钉死用例,登记 [KNOWN_ISSUES.md](../KNOWN_ISSUES.md) #61(开放 ⬜) |
| B14 | `stream_copy` 固定 64MB × 10000 = 640GB 上限 | ⚠️ **#63** | 10000 段是 S3 协议上限,放大分段缓冲会突破 512MB 容器预算——**刻意取舍,维持现状**。已写入 `service/stream_copy.go` 注释(段号在**上传前**判,不误杀第 10000 段的合法对象)+ 测试钉住默认值;登记 [KNOWN_ISSUES.md](../KNOWN_ISSUES.md) #63(已决策 ➖) |
| B15 | `LastError` 名不副实(实为首个错误) | ✅ | 全量改名 `FirstError`(job 结果 / SSE 帧 / 持久化字段 / 调用点),[objects.go](../../apps/server/internal/handler/objects.go) `deletePrefixAsync` 同步 |
| B16 | XFF 之外的 `connect-src` 字面量两处维护 | ✅ | `handler/middleware.go` 抽 `defaultCSPConnectSrc` 为唯一字面量来源,`SetCSPConnectSrc` 空值时共用 |
| B17 | `EnumStr` 冗余 `string()` 转换 | ✅ | 全部调用点改传字符串字面量,`EnumStr(string(...))` 全仓 0 处 |
| B18 | batch 删除编排留在 handler(同类均在 service) | ⚠️ **#62(本轮未完成)** | `handler/objects.go` 的 `deleteObjects` / `deletePrefix` / `deletePrefixAsync`(含 `deleteCounts` 计数)与 `handler/copy.go:177` `copyKeysThenDelete` 仍在 handler;`service` 侧只有 `RunBatch` / `CopyKeys`,**无对应的删除编排**。下沉需把响应形状(`deleteCounts` / `deletePrefixResult`)抽离 `http.ResponseWriter` 并重排测试,属可独立成 PR 的纯重构——本轮未做,登记 [KNOWN_ISSUES.md](../KNOWN_ISSUES.md) #62(开放 ⬜) |

**前端(17)**

| # | 项 | 处置 | 实现与证据 |
|---|----|------|------------|
| F1 | `upload.ts:75` `shouldUseMultipart` 生产零引用 | ✅ | `uploadObject` 内联 `file.size < MULTIPART_THRESHOLD` 同义判断,函数删除(门禁与引用方测试同步改写) |
| F2 | 五处导出面死代码(`updateToast` / `applyTheme` / `UPLOAD_CONCURRENCY` / `polling` / `normalizeStringArray`) | ✅ | 5 处去掉 `export` 关键字**保留实现**(非删除),并新增 `deadcode_gate.test.ts` 源码形态门禁(改回 `export` 即红) |
| F3 | `requestTab` 盲断言 `TabKey` | ✅ | 参数收窄为 `TabKey`,删 `as TabKey` 断言 |
| F4 | `regions.ts` `?? []` 恒不可达分支 | ✅ | 分支删除 |
| F5 | `storage.ts` `getBase` / `setBase` 裸奔(降级策略不一致) | ✅ | 补 try/catch 降级并注释口径(与模块内其余函数一致) |
| F6 | `App.vue` 渲染热路径双重 `JSON.parse` | ℹ️ | **复核不成立**:`App.vue` 全文无 `JSON.parse`(0 处),渲染热路径不存在该写法,不改 |
| F7 | 分页大小 100 在 toast 文案二次硬编码 | ✅ | `useObjectBrowser.ts` 抽 `PAGE_SIZE` 单源,toast 文案经 `PAGE_SIZE` 插值 |
| F8 | abort 后 Promise 永不 settle(帧泄漏) | ✅ | 补 settle 分支 |
| F9 | 0 字节无 MIME 文件被当目录占位丢弃 | ✅ | 0 字节文件放行 |
| F10 | `validateDoc` 硬编码中文错误串破坏 i18n | ✅ | 改 i18n key,随界面语言切换 |
| F11 | 7 处可删除行 `v-for` 用 index 作 key | ✅ | 改稳定行键 + 7 个组件「删除中间行保留原 DOM 节点」测试(改回 `:key="i"` 即 **6 红**,验证门禁有效) |
| F12 | 账号回退初始化逻辑逐字复制 | ✅ | 提取 `composables/useAccountSelect.ts` `resolveAccountSelect()`,`App.vue` / `BucketsPanel` / `RecycleBinPanel` 三处改用;[useAccountSelect.test.ts](../../apps/web/src/composables/useAccountSelect.test.ts) 17 例 |
| F13 | `RecycleBinPanel` / `VersionsDialog` 大表无虚拟滚动 | ✅ | 复用 `virtualList.ts`。**行高沿用 `ROW_HEIGHT=42`**(按钮行实测 50px,滚动条略短 ~19%,与 `ObjectList` 既有行为一致)——已知取舍,不改常量以免重排既有组件 |
| F14 | `selectedSize` computed O(n) 全表扫 | ✅ | 增量化(选中 / 取消时增减) |
| F15 | `t` 局部变量遮蔽 i18n `t` | ✅ | 局部变量改名 |
| F16 | `withDefaults(defineProps, {})` 空 no-op | ✅ | 全仓 `withDefaults(` 0 处 |
| F17 | `MigratePanel` 空 `if` 分支 | ✅ | 空分支删除 |

**小计**:35 = 31 ✅ + 2 ⚠️(开放技术债 #61 / #62)+ 1 ⚠️(已决策 #63)+ 1 ℹ️。

---

## 修复优先级建议

> **执行状态(2026-09-24 收口)**:下列 6 档**已全部按序执行完毕**——2 Critical + 20 Required 全量闭环;
> Nit 31/35 闭环,余下 4 项中 **3 项转登记 [`KNOWN_ISSUES.md`](../KNOWN_ISSUES.md)**(#61 / #62 / #63)、
> **1 项复核后判定不成立**,逐项见「处置明细」。

1. **C2**(一行删除,鉴权绕过)+ **C1**(核心功能损坏,需 blob fetch 重构)
2. R1/R2/R4(安全三连)、R5(一行)、F3/F4(一行 × 2)
3. R7/R8/R9(数据正确性三连,均有测试锚点)
4. R6/R10/R11/R12/R13(边界与持久化)
5. 死代码清理(R18/R19/F5 死代码部分——零容忍硬约束,机械可清)
6. 其余 Required 与 Nit 按模块批量小 PR 清理;每项修复需按 AGENTS.md 同步测试与文档
