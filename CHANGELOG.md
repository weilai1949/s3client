# Changelog

遵循 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)。v1.0.0 之前采用 [SemVer](https://semver.org/lang/zh-CN/)；稳定里程碑后日常发版用 **`v1.0.0-YYYYMMDDHHmmss`**，预发布可用 **`v1.0.0-rcN`**（例如 `v1.0.0-rc0`）。

> 「已实现 / 已修复 / 已完善」功能的合并视图（含 0.1.0 起的全量台账）见 [`docs/FEATURES.md`](docs/FEATURES.md)；本文件保留逐字发布历史。

## tag ↔ 版本段对应关系（唯一台账）

> 口径：本表是 git tag 与 CHANGELOG 版本段的**唯一对应台账**，由
> [`apps/server/changelog_tag_gate_test.go`](apps/server/changelog_tag_gate_test.go) 校验
> （新增 / 改名 tag 或段必须同 PR 同步本表）。首列 = git tag（`无` 表示该行只登记版本段）；
> 第二列 = CHANGELOG 版本段（`无` 表示该行只登记 tag）；正式发布（tag 与段一一对应）**不**登记
> 在本表，由门禁正向校验。
> 历史事实（2026-09-30 取证，KNOWN_ISSUES #69 闭环证据）：仓库 2026-09-02 以单 commit
> （`12f39b1`）导入全栈历史，导入前的快照 tag 未随迁，故出现「有段无 tag / 有 tag 无段」
> 两类快照登记——均为**历史事实台账**，不是待办。

| tag | 版本段 | 性质 | 说明 |
|---|---|---|---|
| `无` | `[v1.0.0-20260901182023]` | 快照段（无 tag） | 2026-09-01 快照段；导入时对应 tag 未随迁 |
| `无` | `[20260901.2]` | 快照段（无 tag） | 2026-09-01 快照段；导入时对应 tag 未随迁 |
| `无` | `[20260901]` | 快照段（无 tag） | 2026-09-01 快照段；导入时对应 tag 未随迁 |
| `无` | `[0.1.0]` | 导入前历史段（无 tag） | 2026-08-22 历史段；git tag 自 v0.3.0 起存在 |
| `无` | `[0.2.0]` | 导入前历史段（无 tag） | 2026-08-22 历史段；git tag 自 v0.3.0 起存在 |
| `v1.0.0-20260902164154` | `无` | 快照 tag（无独立段） | 2026-09-02 16:41 打在 `feat(web)` 提交（c33cc07）上，非 release 提交；同日 rc0/rc1 取代 |
| `v1.0.0-20260902170212` | `无` | 快照 tag（无独立段） | 2026-09-02 17:02 打在 `feat` 提交（a77b878）上；同日 rc0/rc1 取代 |
| `v1.0.0-20260902170253` | `无` | 快照 tag（无独立段） | 2026-09-02 17:02 打在 `fix(docs)` 提交（7d8ccec）上；同日 rc0/rc1 取代 |

> `[1.0.0]` 段 ↔ tag `v1.0.0`：段日期已按 tag 事实修正为 2026-09-22（tag 指向提交 0cfd4ef），
> 段内容未改写（历史结论不改写）；该段已移到 `[Unreleased]` 之后恢复倒序。

## [Unreleased]

### 修复（2026-10-09 KNOWN_ISSUES #73 / #80 / #81 / #83 闭环：评审 Optional 项收口）

来源 [`docs/KNOWN_ISSUES.md`](docs/KNOWN_ISSUES.md) #73 / #80 / #81 / #83（评审 O2 / O9 / O10 / O12）；同批 #72 / #74–#79 / #82 仍开放。

- **#73** `s3wrap` 4 个签名带 AWS SDK 类型、仅供包内调用的转换函数降为小写（`fromS3Object` / `formatBuckets` / `describeACL` / `granteeLabel`），收窄 SDK 类型外泄到 handler 的入口。
- **#80** 计划校验错误不再透传 `err.Error()`：新增类型化 `service.ScheduleValidationError`（固定 `Msg`、不含用户输入，`Cause` 仅落日志）与 `ScheduleValidationMessage` 通用回退；handler 改用它；`error_echo_gate_test.go` 新增对 `writeErr(..., 400, x.Error())` 形态的匹配与正则自检。
- **#81** `POST /api/accounts`（创建账号，无 `{id}` 可判）对 `accounts` 作用域 token 一律 403（`reason=accounts_create`）——堵住「`accounts:[A]` 的 token 铸造任意新账号」的凭证面；`GET /api/accounts`（列表，不含 `SecretKey` 的 `AccountView`）语义写入 [`docs/threat-model.md`](docs/threat-model.md) 最小权限节。
- **#83** 散点缺陷群 12 处全修：`wrapObjectTooLarge` 漏包两处、`PurgeObject` 变量遮蔽、`dialContextSSRF` 可返回 `(nil,nil)`、`store` / `sqlite` 吞 `encryptAESGCM` 错误、`atomicfile` 固定 `.tmp` 并发互删、storage_report 金额未取整、`Scheduler.List` O(n²)、损坏 `schedules.json` 静默丢弃后被空列表覆盖、非法 cron 停摆不写 `LastError`、`Job.Total` 无锁读、计划 / 任务落盘 `Save` 失败静默（新增指标 `s3c_persist_failures_total`，见 [`docs/OPERATIONS.md`](docs/OPERATIONS.md)）。
- **验证**：Go——`gofmt` 干净 / `go vet` 0 告警 / `go build ./...` OK / `golangci-lint` 0 issues / `go test ./...` 10/10 包 / `go test ./... -coverprofile` 全包 100.0% statements 且 `count==0` 零块。

### 修复（2026-10-09 / 10-10 KNOWN_ISSUES #82 闭环：R10 残留 · `make check` 缺项 / E2E 路径过滤 / DEVELOPMENT 漂移）

来源 [`docs/KNOWN_ISSUES.md`](docs/KNOWN_ISSUES.md) #82（评审 §5 O11，亦即 R10「工具链 / 文档不一致」的残留半边）；同批 #72 / #74–#79 仍开放。

- **`make check` 补全**：`Makefile` 新增 `build`（`go build ./...`）与 `web-lint`（`pnpm lint`）目标；**2026-10-10 再处置补齐 `web-typecheck`（`pnpm typecheck`）与 `web-build`（`pnpm build`）**，`check` 聚合改为 `vet govulncheck lint build test-cover web-lint web-typecheck web-typecheck-e2e web-test-cover web-build`，`.PHONY` 同步——本地聚合入口逐项覆盖 CI `server`（gofmt 除外）与 `web` 两个 job 的全部静态门禁，并**真跑 `make check` exit 0** 复核。
- **E2E 路径过滤**：真 RustFS Go E2E 的 PR 触发面从 `apps/server/internal/s3wrap/**` 放宽到 `apps/server/**`（GitHub `e2e.yml` 与 GitLab `.e2e-rustfs-trigger` 双侧）——只改 `internal/handler/**` / `service` / `config` 的后端 PR 不再跳过唯一的真 S3 对端门禁。
- **DEVELOPMENT 三处漂移**：`perf.yml` 的 job 名 `perf-budget`（不存在）→ 真实 id `bench`；覆盖率排除项补全为 `src/main.ts` / `src/env.d.ts` / `src/**/*.test.ts` / `src/i18n/messages/**` / `src/assets/**`（原只写 `i18n/**`）；workflow_dispatch 触发行的 job 计数纠正（`ci.yml` 共 6 个 job；GitLab 7 个自动 job + `semgrep-sast`）。
- **防回退门禁**：新增 `apps/server/doc_ci_drift_gate_test.go`（文档引用的 CI job id 必须真实存在 + 覆盖率排除项必须完整写出），`repo_infra_gate_test.go` 加 `TestMakefileCheckMirrorsCIStaticGates`，`ci_consistency_gate_test.go` 加 `TestRustFSE2ETriggersCoverWholeBackend`——三道路径门禁均 TDD 先红后绿。
- **验证**：`go test .`（含全部门禁）ok；`make check` 逐项覆盖 CI `server`（gofmt 除外）与 `web` 两 job 的全部静态门禁。

### 修复（2026-10-09 KNOWN_ISSUES #72 / #74 / #75 / #77 / #78 闭环：评审 Optional 项第二批）

来源 [`docs/KNOWN_ISSUES.md`](docs/KNOWN_ISSUES.md) #72 / #74 / #75 / #77 / #78（评审 O1 / O3 / O4 / O6 / O7）；至此评审 Optional 项仅剩 #76 / #79。

- **#72** 数据面 `UNSIGNED-PAYLOAD` 收窄：`s3wrap/client.go` 仅在请求带 stream（`PutObject` / `UploadPart`）时注入，无 body 的 GET/HEAD/DELETE/List 恢复 SigV4 空体哈希签名（此前无条件注入，令这些请求也无谓放弃可计算的载荷哈希）；`http://` endpoint 带 body 操作的残留风险记入 [`docs/threat-model.md`](docs/threat-model.md) §6.2。
- **#74** 前端传输层保留 HTTP 状态：`api/http.ts` 新增导出类 `ApiError{status, body}`（错误响应归一携带状态与已解析体），成功响应的非 JSON body 由 `request` 边界校验上抛 `ApiError`；`api/jobs.ts` 的 SSE 失败改抛 `ApiError`。
- **#75** `api/generated.gate.test.ts` 补双向穷尽性断言：生产源码 `opPath('<id>')` 引用的 opId 必须存在，且未被引用的 opId 必须正好等于 8 个「前端不消费」白名单（已变异验证）。
- **#77** 右键菜单 a11y：`ObjectContextMenu.vue` 关闭还原来源焦点、Escape 改走 `useKeydownStack`（LIFO，删掉 `useObjectBrowser.ts` 的独立 window Escape 监听）；`ObjectList.vue` 的「⋯」触发器补 `aria-haspopup` / `aria-expanded`（新增 `ctxEntryKey` prop）。
- **#78** i18n 覆盖测试补「模板 / 数据驱动键逐个有定义」断言（堵 `storageReport.kind.${kind}` 等盲区），并清理 3 处硬编码可见文案为 key（`accounts.accessKeyId` / `accounts.accessKeySecret` / `batchEdit.tagsReplace`）。
- **验证**：Go——`gofmt` 干净 / `go vet` 0 告警 / `go build ./...` OK / `golangci-lint` 0 issues / `go test ./...` 10/10 包 / 全包 100.0% statements 且 `count==0` 零块；前端——`pnpm lint` 0 告警 / `pnpm typecheck` exit 0 / `pnpm test:coverage` **82 文件 1290 例、四指标 100%（4895 / 3260 / 1231 / 4230）**。

### 修复（2026-10-09 KNOWN_ISSUES #76 / #79 闭环：评审 Optional 项第三批 · O1–O12 归零）

来源 [`docs/KNOWN_ISSUES.md`](docs/KNOWN_ISSUES.md) #76 / #79（评审 O5 / O8）；至此 2026-10-09 评审的 O1–O12（#72–#83）**全部闭环**。

- **#76** 前端结构性重复收敛：`newRowKey` 的 7 份逐字复制提取为 `src/rowKey.ts` 的 `createRowKey` 工厂；虚拟滚动管线（ObjectList / MigratePanel / VersionsDialog / RecycleBinPanel 的 4 份）提取为 `src/composables/useVirtualRows.ts`，组件只传响应式数组。
- **#79** OpenAPI 契约漂移收口：新增 `openapi_status_declared_test.go`（handler 直接写出的状态码——含 `h.xxx()` 委托闭包——必须在该 operation 声明，通用码由共享 `components.responses` 放行）并补齐 migrate jobs 404 / trash 409 / 4 个异步端点 503 / proxy 400+416 / copy-object 409；`putObjectLock` 用新增的 `Schema.AnyOf` 表达 days/years 二选一；新增 `TestOpenAPIBucketNotRequiredInRequestBody`，把 `bucket` 从全部请求体 required 移除（对齐 `bucketOr` 回退账号默认桶与共享 Bucket query「可省略」语义）；重生成 `docs/api/openapi.json` 与前端 `schema.d.ts`。
- **验证**：Go——`gofmt` 干净 / `go vet` 0 告警 / `go build ./...` OK / `golangci-lint` 0 issues / `go test ./...` 10/10 包 / 全包 100.0% statements 且 `count==0` 零块；前端——`pnpm lint` 0 告警 / `pnpm typecheck` exit 0 / `pnpm test:coverage` **83 文件 1291 例、四指标 100%（4829 / 3241 / 1218 / 4180）** / `pnpm gen:api --check` 同步。

### 修复（2026-10-09 全仓代码评审批次：C1–C2 + R1–R9 全修，O1–O12 登记）

来源 [`docs/code-review-2026-10-09.md`](docs/code-review-2026-10-09.md)（状态表与 §8 已回写）。

- **C1** CI Go job 补真实前端依赖前置：GitHub `server` job 加 pnpm/setup-node（与 web job 同 pin：pnpm 9.15.0 / node 26.10.0 + pnpm cache）+ `pnpm install --frozen-lockfile`；GitLab `server` 用 nodejs tarball + `npm install -g pnpm@9.15.0` + `.pnpm-store/` cache override。根包 `agent_evals` 门禁的判据前置是 `apps/web/node_modules` 存在（gitignore 构建产物，**不 `mkdir` 伪造**）。
- **C2** 两套 CI 全部 checkout 改深克隆（GitHub 十处 `fetch-depth: 0`，GitLab `GIT_DEPTH: "0"`）——`changelog_tag` 门禁从 `.git` 读 v* tag，浅克隆必红。新增 `apps/server/ci_consistency_gate_test.go` 把两侧 job / 命令 / 版本 pin 逐项一致机械钉住（`gitlab-ci-local` 本地实测可用性见 `docs/DEVELOPMENT.md` §CI 双平台一致性）。
- **R1（安全）** `/api/migrate/jobs*` 端点级作用域闸：任务记录不带归属、通用桶/账号判定落空，声明 `prefixes` / `accounts` 的 token 一律 403（审计 `reason=migrate_jobs`），`readonly` 读放行（本就可读全量桶）、POST 取消仍由方法闸拦——此前前缀 token 可枚举全量任务、读别桶 `failedKeys`、取消他人迁移。
- **R2（安全）** `POST /api/accounts/preview-buckets` 端点级作用域闸：自带 endpoint/凭据由服务端拨号的 SSRF 拨号面，声明 `readonly` / `prefixes` / `accounts` 任一的 token 一律 403（`reason=preview_buckets`，仅 `expiresAt` 不受影响）；`accountIDFromPath` 特例死分支随之删除。`docs/api.md` 与 `docs/threat-model.md` 同步两闸与新 reason。
- **R3** cron DST 重写为「绝对时间轴 + 本地字段回验」：`cronCandidates` 以墙钟 UTC 锚定，Go 归一化落点 ±2h 探针收集偏移、逐一回验本地字段——春季跳变间隙当天触发一次（跳变前偏移解释、落点为跳变后本地时刻，晚触发优于漏跑）、秋季重复小时两次都触发、日锚点取正午防跨日误判；测试自嵌 `time/tzdata`（新增智利午夜跳变「跨日候选剔除」用例）。
- **R4** trash purge 错误映射收窄：仅 `s3wrap.HasErrorCode(err, "ObjectLocked")` → 409，其余上游错误落 500（此前一切 S3 API 错误都映射 409，掩盖真实故障）；连带删除零引用的 `s3wrap.IsAPIError`。
- **R5** 计划 / 任务清单落盘串行化：`persistMu` 把「生成快照 + Save」整体串行（Scheduler 与 JobRegistry 各一，锁序 persistMu → mu），旧快照不再可能覆盖新状态；并发回归测试重构为真并发（Finish 入 goroutine + release 信号，100ms 有界等待）。
- **R6** 前端四个 bucket 面板补请求代际守卫（BucketsPanel / StorageReportPanel / RecycleBinPanel / AccountsPanel 各自内联 `loadSeq`：乱序成功不渲染、过期失败不触底）；RecycleBin 补 `loadingBuckets` 标志 + 桶选择器 `:disabled` + finally 复位（异常不再永久卡 loading，测试钉住「当前请求异常必须复位」）。
- **R7** 删除 `types.ts` 三个死类型导出（`StorageClassUsage` / `PrefixUsage` / `StorageRecommendation`）；**连带**把 `deadcode_gate.test.ts` 声明的「类型导出」盲区升级为真断言（`export type` / `interface` 生产代码零引用即红，含合成口径用例），收口「类型维度无机械保证」；新口径扫出生成物 `operations.ts → Operation` 零引用，由 `gen-api.mjs` 改 `as const satisfies Record<string, Operation>` 真正消费并重新生成（`gen:api --check` 同步）。
- **R8** `endpoints.ts` 的 `multipartParts` 改走 `opPath('multipartParts', { id })`、删过期注释（手写 URL 模板绕过生成契约）；`api.gaps.test.ts` 加整路径断言守回归。
- **R9（测试）** e2e OpenAPI 断言不再「不可能失败」：mock 预览无后端（确定性 502/503）时**条件跳过**，其余状态严格断言 200 + JSON + openapi 字段；常驻真断言迁入 `apps/web/e2e-real/real-backend.spec.ts` 新用例（`scripts/e2e-real.sh` 补 `S3C_EXPOSE_OPENAPI=1`——生产默认 404 不暴露规范）。
- **登记**：评审 O1–O12 → [`docs/KNOWN_ISSUES.md`](docs/KNOWN_ISSUES.md) **#72–#83**（12 条全属技术债 / 缺陷，按「同一事项只登记一处」不进 ROADMAP）；R10 工具链残留同日解除（本机 `/usr/local/go1.26.9` 与 CI 同版本）。
- **验证**：Go——`gofmt` 干净 / `go vet` 0 告警 / `golangci-lint` 0 issues / `go test ./...` 10 包 / `make test-cover` 100.0% statements + `count==0` 零块（`-race`）/ `govulncheck` 0 可达；前端——`pnpm lint` 0 告警、`typecheck` + `typecheck:e2e` exit 0、`test:coverage` 四指标 100%（4879 / 3251 / 1229 / 4213，82 文件 1283 例）、`pnpm build`、mock Playwright 21 passed + 1 skipped（R9 条件跳过项）；真实联调 `make e2e-real`（真后端 + 真 RustFS + 真浏览器）**5 passed**（含新增 OpenAPI 契约真断言）。

### 文档（2026-10-09 提交粒度规范补「同日并行多批一次提交」成文例外）

- **`docs/DEVELOPMENT.md` §1.2 与 `.github/CONTRIBUTING.md` 提交规范**：此前规范只写「一个逻辑改动一个 commit / 每个 commit 只做一件事」，与仓库「同日并行多批一次提交」的既有先例（`28218f2` 四批、`e965e54` 三批）长期存在张力，导致每次评审重提。现补一条**成文例外**：同日并行、且共享同一组生成物 / 文档（`docs/api/openapi.json`、`docs/FEATURES.md`、计数类文档等）的多批改动可合并为一次提交，但提交信息必须**按批切片**（逐批列出改动点），以便评审按批阅读。

### 修复（2026-10-09 s3wrap E2E 清桶对被 Object Lock 锁定的版本不收敛 → 共享实例残留桶泄漏）

- **`cleanupBucket` 遇到被锁版本时空转 100 轮后放弃**（`apps/server/internal/s3wrap/e2e_test.go`）：
  版本删除被 GOVERNANCE 保留 / 法定保留拒绝（403）时原先只吞错不重试，循环空转到上限报
  「did not converge」，桶连同对象留在对端——这正是 2026-10-08 共享 RustFS 上
  `s3c-e2el-*` / `s3c-probe-*-lock` 残留桶（含跨重启存活的对象锁 1 天默认保留）的泄漏机制。
  现增 `e2eForceDeleteVersion`：普通删除被拒后先**关法定保留**（`PutObjectLegalHold` OFF，
  非锁定桶上报错属预期、吞掉），再带 **`x-amz-bypass-governance-retention` 头**重删版本；
  删没删掉仍由下一轮列表收敛判定，COMPLIANCE 保留不可绕过时照常走「did not converge」报错。
  无锁桶路径不变：普通删除成功即不发任何强删请求。
- **`TestE2EObjectLock` 清理不再等到期**（`e2e_features_test.go`）：原「10s 短保留窗 + defer
  睡到到期再清」被强删路径取代（且短窗下 E2E 每跑必等 10s+、真对端永远走不到强删路径）——
  现 defer 直接 `cleanupBucket`，Object Lock E2E 由 ≥11s 降至 **0.17s**，强删路径每次真跑。
- 验证：新增假 S3 回归测试 [`cleanup_fake_test.go`](apps/server/internal/s3wrap/cleanup_fake_test.go)
  （被锁版本「403 → 法保留 OFF → bypass 删除 → 删桶收敛」+ 无锁版本「不发强删请求」，
  **先红后绿**：旧实现报 `cleanup did not converge after 100 rounds: 1 versions`）；`S3CLIENT_E2E=1 go test
  ./internal/s3wrap/ -run TestE2E -v` **7/7 通过**且共享 RustFS `ListBuckets` 复核为 `[]`（零残留）；
  `go vet` / `golangci-lint run`（0 issues）/ `go test ./...`（10 包）全绿。

### 修复（2026-10-09 e2e-real 用例 seed 中途失败泄漏账号/桶 + 建桶失败诊断 + 限速节流）

- **`seedAccountAndBucket(...)` 被放在 `try` 之外 → 失败运行留垃圾**（`apps/web/e2e-real/real-backend.spec.ts`）：
  两个用例都在 `try` 之前调用 seed，而 seed 在**建桶**步骤抛错时不会返回，`finally` 根本不执行——
  账号（以及已建成的桶）就留在真实后端与 RustFS 上，这正是失败运行留垃圾的机制。
  修法：新增**调用方持有的资源登记簿** `SeededResources`，seed 改为「边创建边登记」；seed 移入 `try`，
  `finally` 统一走 `cleanupSeeded`——对未创建的部分判空跳过，账号按**名称**清理（覆盖
  「`createAccountViaUI` 在返回 id 之前失败、`accId` 未登记」这一更隐蔽的泄漏面）。
- **建桶失败不再挂满测试超时**：`createBucketViaUI` 原先只等「建桶成功才会触发的 `GET /buckets`」，
  而该 `waitForResponse` 在本套配置下**没有 30s 上限**（trace 实测 `Page.__waitInfo__` 挂满 135s
  test timeout，且报错行号指向后续无关语句，排查体验极差）。现同时等 `POST /bucket` 的响应，
  失败时立即抛出后端错误（状态码 + 响应体摘要）。
- **限速节流**：新增回归用例后，套件在秒级打出 60+ 个 `/api` 请求，会抽干后端令牌桶
  （`ratelimit.go`：120 req/min、突发 30 = 回填 2 token/s）；而页面自身的请求（对象/桶列表刷新）
  不像本文件那样带退避重试——「直传」用例上传成功后列表刷新撞 429、行断言 15s 超时（可复现）。
  按回填速率在用例之间补 5s 间隔，让每个用例从**接近满桶**起步；**不关限速、不改后端**，保真度不变。
- 验证：`scripts/e2e-real.sh`（自管 RustFS 容器）**4 passed（29.6s）**；回归用例**先红后绿**
  （旧结构下残留 1 个账号：`Expected length: 0, Received length: 1`）；`pnpm typecheck:e2e` / `pnpm lint` exit 0。

### 文档（2026-10-09 全仓代码质量评审快照建档）

- **新增活跃评审文档 [`docs/code-review-2026-10-09.md`](docs/code-review-2026-10-09.md)**：对 `e965e54`
  的全仓五轴评审（Go 后端 + Vue/TS 前端；6 路深潜 + 机械门禁**独立复跑**）：结论 **Request changes**
  ——2 Critical（CI 的 Go job 缺 `apps/web/node_modules` 前置、GitHub checkout 浅克隆读不到 tag，
  两处叠加使仓库自述的「全绿门禁」在 CI 上不可复现）+ 10 Required（迁移任务作用域越权、
  `preview-buckets` 成 SSRF 跳板、cron DST 错时/漏跑、trash purge 全 API 错误→409、清单落盘乱序、
  前端加载竞态、死类型导出、契约绕过等，均带 `file:line` 证据与复现口径）。按 §4 文档同步门禁同 PR
  登记：`docs/README.md` 导航 + 本文件 §4 命名约定与「文档登记表」+ 根 `llms.txt` + 根 `AGENTS.md`。
  报告中的未闭环项**本次未登记**事项台账，待修复时按报告 §8 处置计划登记
  [`KNOWN_ISSUES.md`](docs/KNOWN_ISSUES.md)（开放技术债）/ [`ROADMAP.md`](docs/ROADMAP.md) §三（排期项）。

### 修复（2026-10-09 Go 工具链 1.26.6 → 1.26.9（10 个可达 stdlib 漏洞）+ e2e-real 用例对残留桶不幂等）

- **Go 工具链 1.26.6 → 1.26.9**：`govulncheck ./...` 实测 go1.26.6 上有 **10 个可达** stdlib 漏洞
  （GO-2026-6617 等，net/http / net/textproto / crypto/tls，均 go1.26.9 修复；调用链如
  `service.WriteObjectsZip → io.WriteString → http.response.WriteString`），另 24 个
  「被 require 但代码未调用」的模块漏洞不构成可达面。按 ROADMAP E1 口径**四处同步**：
  `apps/server/go.mod` / `apps/server/Dockerfile`（`golang:1.26.9-alpine`）/
  `.gitlab-ci.yml` 两处（`golang:1.26.9-bookworm`）；GitHub 侧经 `go-version-file` 自动跟随。
- **e2e-real「建桶 → 列桶」对残留桶不幂等**（`--no-rustfs` 复用长期 RustFS 即红）：所有运行共用
  同一组凭据时桶列表全局可见，`BucketsPanel.loadBuckets` 在 selectedBucket 为空时自动钻进
  `buckets[0]`——有残留桶时用例一进桶管理就停在旧桶详情页，建桶后按行断言 15s 超时。两处修复
  （`e2e-real/real-backend.spec.ts`）：① `createBucketViaUI` 先等建桶触发的 `GET /buckets` 刷新
  落定再点「返回列表」后断言列表行——消除「回列表被在途刷新重新钻走」的竞态，同时修掉原先直接按行
  断言**误中详情页概览行**的假绿（干净环境下断言命中的根本不是桶列表行）；② `cleanupBucket` 改走
  「列对象 → 批量 `/delete` → 删桶」：原 `delete-prefix` 空前缀被 handler 有意拒绝（400「拒绝空前缀
  以免误删全桶」），非空桶清不掉、直传用例每跑一次泄漏一个桶。
- 验证：共享 RustFS（残留桶在场）`--no-rustfs` 单跑「建桶 → 列桶」由红转绿，全量 3 用例通过；
  `pnpm typecheck:e2e` / `pnpm lint` exit 0；`govulncheck ./...` 升级后 0 可达。
- 顺带修复：`docs/FEATURES.md` 被上个提交（e965e54）整块复制出的第二份「三、质量与覆盖率现状」
  （109 行逐行重复，仅尾部少一个 `### 已知边界与取舍` 子节）——删除首份副本，保留含完整尾节的第二份。

### 新增（2026-10-08 ROADMAP §三 3.2 #7 FinOps 存储分析与成本看板，全栈）

> 证据台账 [`docs/FEATURES.md`](docs/FEATURES.md) **§BV**；[`docs/ROADMAP.md`](docs/ROADMAP.md)
> §三 3.2 的 #7 行**整行移出**（转空号，编号不重排）。范围由交付人拍板为**全栈**：后端聚合端点 +
> OpenAPI 契约 + 前端独立顶层「成本看板」Tab + 文档。

- **后端**：新增只读端点 `GET /api/accounts/{id}/storage-report?bucket=&prefix=`（缺桶 400 / 未知账号
  404；**端点总数 83 → 84**）。列举桶（可限定前缀）后由 `service.AggregateStorageReport` 聚合：按存储类
  与「列举前缀下首层前缀」的用量、`USD/GiB/月` 成本估算，以及低频（30–89 天）与归档（≥90 天）建议。
  沿用 `sync` / `delete-prefix` 的列举硬上限（100 页 / 10 万对象 / token 不前进即停），超额或多页游标
  异常置 `truncated=true`；未知存储类按 STANDARD 单价估算。AWS SDK 类型不越过 `s3wrap`。
- **前端**：新增顶层 `finops` Tab（`StorageReportPanel.vue`）——选桶 + 可选前缀 → 生成报告，展示总量
  卡片、按存储类 / 前缀用量表与优化建议表；`storageReport.ts` 提供字节 / 金额格式化；i18n 中英双语。
  响应类型派生自 OpenAPI 生成物（`operations['storageReport']`），漂移由 `vue-tsc` 拦。
- **文档**：`api.md` 增端点小节与 curl 示例；`openapi.json` 重生成（61 → 62 paths / 83 → 84 operations）；
  `architecture.md` / `README.md` / `docs/en/index.md` / `docs/README.md` 端点计数同步；`user-guide.md`
  补「成本看板」用法。

### 新增（2026-10-08 ROADMAP §三 3.2 #6：计划任务——cron 定时增量备份）

> 证据台账 [`docs/FEATURES.md`](docs/FEATURES.md) **§BU**；[`docs/ROADMAP.md`](docs/ROADMAP.md) §三 3.2
> **#6 整行移出**（转空号，编号不重排）。本批新增 **5 个** `/api/*` 端点（78 → 83，以提交版
> `docs/api/openapi.json` 为准），自研 5 字段 cron 解析（零新增依赖，`go.mod` 无变化）、
> 调度循环与计划落盘复用 `JobRegistry` / `atomicfile` 既有口径；提交版规范与前端生成物已重新生成。

- **计划任务 5 端点**：`GET/POST /api/schedules`、`PUT/DELETE /api/schedules/{id}`、
  `POST /api/schedules/{id}/run`。自研 cron 解析（标准 5 字段、仅数字、Vixie dom/dow OR 语义、
  40 年视界拒绝永不触发表达式）；三条调度语义有测试钉住：停机补跑**只补一次**、
  上一轮未结束**不叠加**（手动 run → 409）、触发失败**记录 `lastError` 且排期照常前移**（防重试轰炸）。
  计划落盘 `S3C_DATA_DIR/schedules.json`（0600 原子写，重启自动恢复）；手动 run 复用
  `JobRegistry` 异步任务链路（进度 / SSE / 取消与迁移任务同口径），列举失败与截断均记入
  `result.lastError`（不静默报「无事可做」）。
- **前端**：迁移面板新增「计划任务」区块（`SchedulesSection`）——列表 / 新建 / 编辑 /
  启停 / 立即运行 / 删除，cron 与账号校验错误行内回显；i18n 中英双语键与字典门禁同步。
- **作用域**：`S3C_TOKEN_SCOPES` 的 `prefixes` 对计划的 `run`/`DELETE` 按**已存计划的桶/前缀**判定
  （请求无 body 时从计划注入引用，越界计划不可触发/删除）；`readonly` 拦全部写方法。

### 新增（2026-10-08 ROADMAP §三 3.2 #5：S3 条件写 / 端到端校验和 / Object Lock）

> 证据台账 [`docs/FEATURES.md`](docs/FEATURES.md) **§BT**；[`docs/ROADMAP.md`](docs/ROADMAP.md) §三 3.2
> **#5 整行移出**（转空号，编号不重排）。本批新增 **7 个** `/api/*` 端点（71 → 78；同日并行批次
> （#6 计划任务 / #7 FinOps）收口后仓库合计 **84**——以提交版 `docs/api/openapi.json` 为准），
> 提交版规范与前端 `schema.d.ts` / `operations.ts` 均已重新生成。
> 三件套均经 `s3wrap` 单边界接入、按厂商支持度降级；真实 RustFS E2E 实测通过（7/7）。

- **条件写（If-Match / If-None-Match）**：`POST /api/accounts/{id}/presign`（`method=put`）新增
  `ifMatch` / `ifNoneMatch`，响应新增 `headers` 回显（浏览器直传必须原样携带，条件头参与签名）；
  `mkdir` / `copy-object` 同字段（复制条件作用于**目标**对象）。条件不满足 → **412**
  （`PreconditionFailed`）、并发冲突 → **409**（`ConditionalRequestConflict`）；前端上传支持
  「仅当对象不存在时创建」，防并发覆盖 / 丢更新。
- **端到端校验和**：`GET .../head` 新增 `checksums`（CRC64NVME / CRC32C / SHA256 / SHA1 + `type`，
  无则 `null`）；新端点 `POST .../verify-checksum` 流式拉取全对象本地重算与存储端比对
  （阶梯 crc64nvme → crc32c → sha256 → sha1 → etag-md5；合成校验和跳过；无来源 → `method="none"`
  如实降级）；自研流式 **CRC-64/NVME** 实现（参数与 AWS `aws_checksums_crc64nvme` 一致，
  独立向量 + 真实 RustFS 返回值双向对齐）。
- **Object Lock（WORM 保留）**：新端点 `GET/PUT .../bucket/object-lock`（桶默认保留策略）、
  `GET/PUT .../object-retention`（对象保留期）、`GET/PUT .../object-legal-hold`（法定保留）；
  读侧降级（未启用 → `enabled/configured=false`、未设置 → `status=OFF`），错误映射
  **400**（未启用 / 保留期违规）/ **403**（GOVERNANCE 拒绝）/ **409**（`ObjectLocked`、既有桶启用被拒）/
  **501**（厂商未实现）。前端：对象详情显示 / 校验 / 编辑校验和与保留状态，桶设置新增 Object Lock 区块。

### 变更（2026-10-08 ROADMAP §三 3.2 #8 / #11 / #13 三路并行落地：大文件续传 + 并行分段 / 零依赖 OTLP trace / Token 作用域）

> 证据台账 [`docs/FEATURES.md`](docs/FEATURES.md) **§BQ**（#8）/ **§BR**（#11）/ **§BS**（#13）；
> [`docs/ROADMAP.md`](docs/ROADMAP.md) §三 3.2 三行**整行移出**（#8 / #11 / #13 转空号，编号不重排）。
> 三条设计口径在开工前由人类拍板：**零新增依赖**自研 OTLP（不引 `opentelemetry-go`）、新增 **1 个**
> 只读 `ListParts` 端点、作用域用独立 env `S3C_TOKEN_SCOPES`（`S3C_TOKEN` 语义不变）。
> `apps/server/go.mod` 的 `require` 段**零变化**（无新增直接 / 传递依赖），故许可证清单无需重生成。

- **① 大文件体验（#8，原 `KNOWN_ISSUES` #51）**：新增只读端点
  `GET /api/accounts/{id}/multipart/parts`（`bucket` / `key` / `uploadId` → `{"parts":[…]}`，缺参 400；
  **端点总数 70 → 71**）与 [`s3wrap`](apps/server/internal/s3wrap/multipart.go) `ListParts`（自有 DTO，
  AWS SDK 类型不外泄，自动翻页）。前端 [`multipartResume.ts`](apps/web/src/multipartResume.ts) 以
  `文件名 + 大小 + 修改时间` 指纹在 localStorage 记录 uploadId 与已完成分段（**零凭证**，上限 20 条），
  刷新 / 重选同一文件时先与服务端真实清单对齐、只补缺段，会话失效则干净重 init；
  [`api/download.ts`](apps/web/src/api/download.ts) 对已知 size ≥16MB 的对象走 **4 路 × 4MB Range GET**
  （有界并发，ADR-009），逐段校验 206 + 字节数 + `Content-Range` 后按序聚合落盘，服务端忽略 Range /
  大小未知 / 小文件自动回退单流——失败即报错，**不落损坏文件**。
- **② 零依赖 OTLP tracing（#11，原 `KNOWN_ISSUES` #54）**：新包
  [`internal/tracing`](apps/server/internal/tracing/) 用标准库实现 W3C `traceparent` 解析 / 生成 +
  OTLP/HTTP JSON `POST {Endpoint}/v1/traces`（`resourceSpans→scopeSpans→spans`；有界队列 512 / 批 64 /
  1s ticker / `Close` 刷出；导出失败与队列满只 WARN，绝不影响请求）。[`main.go`](apps/server/main.go)
  用中间件包住全部路由，`presign` / `proxy` / `migrate`（含 sync / async）打子 span 并以既有
  `X-Request-ID` 关联（span 属性 `request.id`）。**默认关闭**（`S3C_OTEL_ENDPOINT` 为空即零开销）；
  新增 `S3C_OTEL_SAMPLE_RATIO`（默认 1，仅 [0,1]，非法**拒绝启动**）与 `S3C_OTEL_SERVICE_NAME`
  （默认 `s3client`）。决策与替代方案见 [ADR-013](docs/decisions/0013-zero-dep-otlp-tracing.md)。
- **③ Token 作用域与最小权限（#13，原 `KNOWN_ISSUES` #56）**：新增独立 env `S3C_TOKEN_SCOPES`
  （JSON：token → `{readonly, prefixes, accounts, expiresAt}`，严格解析 `DisallowUnknownFields`），
  `S3C_TOKEN` 语义不变、**未登记 token 仍为全权**。`readonly` 仅放行 GET / HEAD（预签名 POST 能铸造
  写 URL，同样 403）；`prefixes` 对 query 与 JSON body 的桶 / 键**各自**校验（桶级操作需整桶授权，
  无法判定即 fail-closed）；`accounts` 限路径 `{id}`；`expiresAt` 过期 401。越权写审计
  `auth.scope_denied`（reason 分类，**不含 token 明文**）并返回 403；非法配置（未知字段 / 未登记
  token / 空元素 / 坏时间 / `prefixes` 以 `/` 开头）启动即失败。OpenAPI `bearerAuth` 补作用域语义。
- **门禁实跑（2026-10-08，全部本机实跑）**：`gofmt -l` 干净 / `go vet ./...` 0 告警 /
  `golangci-lint run` **0 issues** / `make test-cover` **10/10 包 100.0%**（profile `count==0` 零块；
  新增 `internal/tracing` 为第 10 包）/ `go build ./...` OK / 包根文档门禁 `go test . -count=1`
  **ok 5.182s**；`docs/api/openapi.json` 重生成（70 → **71** operations）+ `pnpm gen:api` 重生成两份
  前端产物（`--check` 口径由 `generated.gate.test.ts` 守住）；前端 `pnpm test` **78 文件 / 1189 例**、
  `pnpm test:coverage` 四指标 **100%**（4481 / 3007 / 1151 / 3853）、`pnpm build` OK
  （**381.41 kB / gzip 116.15 kB**，CSS 32.40 kB）、`pnpm lint` 0 告警、`pnpm typecheck:e2e` exit 0；
  真实对端三条：`S3CLIENT_E2E=1 … -run TestE2E` **4/4 PASS**、`make e2e-real` **3 passed**、
  `pnpm e2e` **22 passed**。
- **文档同步**：[`docs/FEATURES.md`](docs/FEATURES.md) 新增 **§BQ** / **§BR** / **§BS**（头部摘要 +
  目录行同步）；[`docs/ROADMAP.md`](docs/ROADMAP.md) §三 3.2 三行移出 + 空号注记 + §四 基线按本轮复测刷新；
  `docs/CONFIGURATION.md`（4 个新 env）、`docs/api.md`（新端点 + 鉴权 / 403 语义）、`docs/OPERATIONS.md`
  §3.4（trace 开启与失败模式）、`docs/architecture.md` 与 `docs/en/architecture.md`、`docs/threat-model.md`
  （最小权限段）、`docs/user-guide.md`（续传与并发下载）、`docs/decisions/0013-*.md` + `index.md`、
  `apps/server/.env.example`；端点计数 70 → 71 四处联动（`README.md` / `docs/en/index.md` /
  `docs/ROADMAP.md` / `docs/FEATURES.md`，由 `doc_number_gate_test.go` + `en_docs_gate_test.go` 守住）。

### 变更（2026-10-08 KNOWN_ISSUES #71 第二批：产品名 `s3clinet` → `s3client` 全量统一）

> 证据台账 [`docs/FEATURES.md`](docs/FEATURES.md) **§BP**；上接同日 **§BO**（仓库 slug 统一）。§BO 当时把
> 品牌 / 运行时 / 监控命名空间列为「刻意不动」的**范围边界**，本批按要求**把该边界一并取消**——
> `KNOWN_ISSUES` #71 的闭环口径由「只统一 slug」扩为「slug + 产品名一次到底」。

- **① 文本与配置**：`s3clinet` / `S3Clinet` / `S3CLINET` → `s3client` / `S3Client` / `S3CLIENT`，**86 个受版本
  控制的文件、358 处**（`.md` 37 / `.go` 18 / `.yml` 8 / `.json` 6 / `.conf` 6 / `.sh` 5 …，含根 `AGENTS.md`、
  `llms.txt`、`CITATION.cff`、两份 `.env.example`、`Makefile`、`.gitlab-ci.yml` 与 4 个 GitHub workflow）。
- **② 文件改名 5 个（`git mv` 保留历史）**：`deploy/prometheus/s3clinet.rules.yml` → `s3client.rules.yml`、
  `deploy/grafana/s3clinet.dashboard.json` → `s3client.dashboard.json`、`deploy/nginx/conf.d/s3clinet-{tls.example,docker,local}.conf`
  → `s3client-*`；全仓引用同批改（含 `grafana_dashboard_gate_test.go` 的路径常量与记录规则正则、
  `repo_infra_gate_test.go` 的 `rulesRel`）。
- **③ 运行时与产物（含行为变更）**：单写者锁文件 `.s3clinet.lock` → **`.s3client.lock`**（旧锁文件在数据目录里
  只是空文件、锁由内核持有，残留无影响）；启动日志 `msg="s3clinet server"` → `s3client server`；二进制
  `s3client-server`（Dockerfile `ENTRYPOINT` / compose / `scripts/*.sh`）；镜像与 `container_name` 的
  `s3client/server` 系；Cargo 包 `s3client` + Tauri `productName` / `identifier`；npm `s3client-web` /
  `s3client-desktop`；E2E 环境变量 `S3CLIENT_E2E` / `S3CLIENT_{ENDPOINT,ACCESS_KEY,SECRET_KEY}`
  （脚本 / 测试 / 文档 / 根 `AGENTS.md` 同批改）。
- **④ 契约与生成物**：`info.title` `s3clinet API` → `s3client API`（`description` 同步）→
  `go test ./internal/handler/ -run TestCommittedOpenAPISpecMatchesRuntime -update-openapi-spec` 重生成
  [`docs/api/openapi.json`](docs/api/openapi.json) → `pnpm gen:api` 重生成 `schema.d.ts` / `operations.ts`
  （`pnpm gen:api --check` **exit 0**）。
- **⑤ 监控命名空间**：记录规则 `s3clinet:*` → `s3client:*`、告警 `S3Clinet*` → `S3Client*`；
  `docs/OPERATIONS.md` §4 告警表 / SLI 表与 `docs/README.md` / `docs/DEVELOPMENT.md` 的文件引用同步。
- **⑥ 豁免（历史不回写）**：**本文件历史条目**只修正指向改名文件的**路径链接**（10 处），叙述里的旧名照旧；
  `docs/archive/` 冻结件整份不动。故全仓仍可见旧写法的位置**仅这两类**。
- **门禁实跑（2026-10-08，与同日 ROADMAP #8 / #11 / #13 批次同树实测）**：`go vet ./...` **0 告警**、
  `go build ./...` OK、`go test -race -count=1 -coverprofile=coverage.out ./...` **10/10 包 100.0%**
  （profile `count==0` 零块；同批新增 `internal/tracing`，故后端包数 9 → 10）、`golangci-lint run` **0 issues**、
  包根文档门禁 `go test . -count=1` **ok 4.198s**；前端 `pnpm test` **78 文件 / 1189 例** 全绿、
  `pnpm lint` **0 告警**、`pnpm gen:api --check` **exit 0**、`pnpm build` OK（**381.41 kB / gzip 116.15 kB**）。

- **文档同步**：[`docs/FEATURES.md`](docs/FEATURES.md) 新增 **§BP**（头部摘要 + 目录行同步，§BO ③ 标注
  「同日已被 §BP 取代」）；[`docs/KNOWN_ISSUES.md`](docs/KNOWN_ISSUES.md) 头部 2026-10-08 段改写为
  「同日两批」并把 #71 台账行同步为两批口径。

### 文档（2026-10-08 根 README 新增「已知限制」小节：跨 endpoint 迁移单对象 640GB 上限）

- **新增 `README.md` §已知限制**：把 [`docs/KNOWN_ISSUES.md`](docs/KNOWN_ISSUES.md) **#63**（已决策 ➖）
  从「只写在代码注释 / 台账 / 运维表」推到**用户可见面**——`64MB × 10000 段 = 640GB`、超限**明确拒绝并
  abort（不静默截断）**、不放大的内存账（512MB 容器预算，一块分段缓冲即 64MB），并链到
  `service/stream_copy.go` 注释、5 个钉默认值 / 边界的测试与 #63 登记行；小节同时写明
  「已知限制的唯一登记处是 `KNOWN_ISSUES.md`，本节只登用户可见的那一条」（防双源漂移）。
- **文档同步**：`docs/KNOWN_ISSUES.md` #63 行补一句「2026-10-08 已补登根 README」（登记行自证可见性）。
- **门禁实跑（2026-10-08）**：`cd apps/server && go test . -count=1` → 包根文档门禁（`doc_link` /
  `doc_number` / `docs_naming` / `doc_index` 等）**ok 5.291s**。

### 变更（2026-10-08 KNOWN_ISSUES #71 闭环：仓库 slug 统一为 `github.com/weilai1949/s3client`）

> 证据台账 [`docs/FEATURES.md`](docs/FEATURES.md) **§BO**；[`docs/KNOWN_ISSUES.md`](docs/KNOWN_ISSUES.md)
> **#71 整行移除**（编号台账改「已闭环移除」）。**推翻 2026-09-30 的 ➖「维持现状」决策**——登记时的理由
> （改模块路径 = 全仓 import 重命名、与当批正交）没有变，变的是**成本已被量化为一次纯机械替换**。

- **① 模块路径**：[`apps/server/go.mod`](apps/server/go.mod) `module github.com/weilai1949/s3clinet/apps/server`
  → `…/s3client/apps/server`；全仓 Go import 同步改写（含门禁测试里以**字符串字面量**出现的示例 import），
  合计 **120 个受版本控制的文件**（Go 110 + `go.mod` 1 + 非 Go 9）。
- **② 仓库 URL**：`.github/SECURITY.md` · `.github/ISSUE_TEMPLATE/config.yml` · `.github/SUPPORT.md` ·
  `.github/CONTRIBUTING.md` · `docs/AI_POLICY.md` · `docs/en/index.md` · `docs/api/accounts.schema.json` 的
  `$id` · `deploy/grafana/s3client.dashboard.json` 面板 `url` · 根 `README.md` 的 GitHub Release 链接 ——
  统一到与 `git remote` / `.well-known/security.txt` / `CITATION.cff` 一致的 `s3client`。
- **③ 范围边界（本批写死；⚠️ 同日下方「第二批」已把该边界取消、品牌一并统一）**：**品牌 / 运行时 /
  监控命名空间不属仓库 slug、本批刻意不动**——产品名 `s3clinet`（文档标题、OpenAPI `title`、启动日志、`CITATION.cff` 标题）、
  运行时工件 `.s3clinet.lock` / 二进制 `s3clinet-server` / 镜像 `s3clinet/server` / `container_name` /
  Cargo 包 `s3clinet` / npm `s3clinet-web`、监控命名空间 `s3clinet:*` 记录规则 + `S3Clinet*` 告警 +
  `deploy/{prometheus,grafana}/s3clinet.*` 文件名；本文件的 monorepo 迁移历史条目按「历史条目不改写」
  保留旧路径时点叙述。
- **④ 残留核对**：全仓按旧 slug `grep -rl` → 除**本条与 [`docs/FEATURES.md`](docs/FEATURES.md) §BO
  记录「改前值」的叙述性引用**（以及本文件 monorepo 迁移历史条目）外 **0 处**，**活引用已清零**；
  两个被 `.gitignore` 忽略的本地旧构建产物已删除，下次构建按新路径重新产出。
- **门禁实跑（2026-10-08）**：`go vet ./... && go build ./... && go test ./... -count=1` → **9/9 包 ok**；
  `golangci-lint run` → **0 issues**；`go test . -count=1` → **ok 4.939s**、文档写回后复跑 **ok 4.235s**（包根文档门禁）；
  前端 `pnpm test` **76 文件 / 1158 例**、`pnpm lint` **0 告警**、`pnpm build` OK（377.34 kB / gzip 114.83 kB）。

### 变更（2026-10-08 批次交接快照 `handoff-20260930.md` 归档冻结：归档四步）

> 所记批次四条目（`ROADMAP` #19 / `KNOWN_ISSUES` #70 / `ROADMAP` #18 / #17）已于 2026-10-01
> 全部收口（快照 §2 表），故按其自身生命周期规则与 [`docs/DEVELOPMENT.md`](docs/DEVELOPMENT.md) §4
> 归档条款执行冻结——**归档 = 不删除、不回写、不改写历史结论**，正文数字与结论零改动。

- **① `git mv`**：`docs/handoff-20260930.md` → [`docs/archive/handoff-20260930.md`](docs/archive/handoff-20260930.md)
  （保留重命名历史），仅在头部**追加**一行「归档冻结（2026-10-08）」状态行（与 `assessment.md` /
  `review-2026-09-19.md` 归档先例同款：附加状态、非回写）。
- **② 引用收敛**：报告自身 15 条出链按新位置改相对路径（`ROADMAP.md` / `KNOWN_ISSUES.md` /
  `FEATURES.md` / `DEVELOPMENT.md` / `accessibility.md` / `README.md` → `../…`、`../CHANGELOG.md` /
  `../AGENTS.md` → `../../…`、`archive/index.md` → `index.md`）；`docs/README.md` 的「接手中断的进行中
  批次」行并入相邻的「找冻结的历史快照」行；三处命名清单摘除——[`AGENTS.md`](AGENTS.md)、
  [`docs/DEVELOPMENT.md`](docs/DEVELOPMENT.md) §4 命名段、[`llms.txt`](llms.txt) 改为只保留
  「时点性批次交接快照同用小写、收口后归档」的规则本身（不再登记活文件名）。
- **③ 索引与登记表**：[`docs/archive/index.md`](docs/archive/index.md) 归档清单补一行、「已归档 5 份」
  → **6 份**、新增「批次交接快照收口后同 PR 冻结」触发规则（首例即本件）；`docs/DEVELOPMENT.md` §4
  文档登记表该行改为**冻结件**（`不复审——归档 = 冻结`，`2026-10-08（归档登记）`）并补归档触发规则一句。
- **豁免口径**（沿用 `assessment.md` 归档先例）：`archive/index.md`「原路径」列、`CHANGELOG` 本条与
  快照正文里的时点性旧路径叙述**不改写**——它们是历史记录，不是活链接。
- **门禁实跑（2026-10-08）**：`cd apps/server && go test . -count=1` **ok（4.669s）**（含
  `doc_index` / `doc_link` / `docs_naming` / `doc_review` / `doc_number` / `llms_size` 等包根文档门禁）。

### 变更（2026-10-01 ROADMAP #10 OpenAPI → 前端类型 / 客户端代码生成：schema-first + 生成物新鲜度门禁）

> 证据台账 [`docs/FEATURES.md`](docs/FEATURES.md) **§BN**；`docs/ROADMAP.md` §三 3.2 **#10 已移出**
> （转空号，前言空号清单 +1）。`dependencies` 仍只有 `vue`，生成器与解析库均为 devDependency，
> 不违 [ADR-004](docs/decisions/0004-minimal-frontend-deps.md)。

- **① 生成器与两份提交产物**：新增 [`apps/web/scripts/gen-api.mjs`](apps/web/scripts/gen-api.mjs)
  与 `pnpm gen:api` 脚本，从黄金契约 [`docs/api/openapi.json`](docs/api/openapi.json) 生成
  [`apps/web/src/api/schema.d.ts`](apps/web/src/api/schema.d.ts)（**4506 行**类型，`openapi-typescript@7.13.0`
  Node API，**无时间戳横幅**故字节稳定）与 [`apps/web/src/api/operations.ts`](apps/web/src/api/operations.ts)
  （**70 条** `operationId → {method, path, params}`）；`--check` 模式逐字节比对，不一致即非零退出。
- **② URL / method 真正来自 spec**（消除漂移的那半边）：[`http.ts`](apps/web/src/api/http.ts) 新增
  `opPath()` + 条件剩余元组——**无占位符的操作可省参、有占位符的漏参即编译错误**，替换一律
  `encodeURIComponent`；[`endpoints.ts`](apps/web/src/api/endpoints.ts) 整体重写，**58 处 URL / method**
  全部由 `operations` 派生；另收编 `download.ts` / `jobs.ts`（含 SSE）/ `index.ts` / `proxy.ts` /
  `ServerPanel.vue` 五处硬编码 `/api/…`。query 串刻意仍手写（query 名不在 path 模板内，已由后端
  `openapi_query_params_test.go` 钉住）。
- **③ 黄金 spec `required` 缺口修复**：`AccountView` / `objectItem` / `listObjectsResponse` **无 `omitempty`**
  （字段恒定序列化），但 `sharedSchemas()` 的 `required` 只列了 4/12、4/6、3/4 —— 补齐后重生成
  `docs/api/openapi.json`（**+13/-2**）。响应契约测试只比字段集合不比 `required`，故收紧后仍全绿。
  **57 条内联 2xx 响应仍无 `required`**，未并入本条完成口径。
- **④ `types.ts` 改为从 spec 派生** 4 个共享实体（`Account` / `ObjectItem` / `ListObjectsResponse` /
  `BucketItem`），使 `schema.d.ts` 成为真实消费者而非孤儿模块（死代码门禁曾因此红灯）。
- **⑤ 新鲜度门禁落地**：新增 [`apps/web/src/api/generated.gate.test.ts`](apps/web/src/api/generated.gate.test.ts)
  （**3 例**）——生成物与 spec 逐字节一致、结构自检（条数 ≥60 防「空表永远绿」+ 占位符 ⇔ `params` 逐个对齐）、
  `opPath` 编码行为。这是 #10「生成物入 CI diff 门禁」的字面落地：改 spec 忘了 `pnpm gen:api` 即红灯。
- **门禁实跑（2026-10-01）**：前端 `pnpm test` **76 文件 / 1158 例**、`pnpm test:coverage` 四指标
  **100%**（4327 / 2932 / 1131 / 3712）、`pnpm lint` **0 告警**、`pnpm typecheck` + `typecheck:e2e` 均 exit 0、
  `pnpm build` OK（377.34 kB / gzip 114.83 kB）、`pnpm gen:api --check` **exit 0**；
  后端 `gofmt -l .` 干净、`go vet` 干净、`golangci-lint run` **0 issues**、`go build` OK、
  `go test ./... -count=1` **9/9 包**、`make test-cover` **9/9 包 100%**、`govulncheck` **0 可达**；
  `make check` / `make bench` **exit 0**。
- **真实链路复跑**（本批改了前端，按 AGENTS 必跑）：`pnpm e2e` **22 passed / 0 skipped**、
  `make e2e-real` **3 passed**（`SERVER_PORT=8081`）、`S3CLINET_E2E=1` **4/4 PASS**（首轮 4/4 全红为
  环境性——本机 `127.0.0.1:9000` 当时无 RustFS；临时起 `rustfs/rustfs:1.0.0-rc.3` 后复跑通过，跑完即删）。
  第三方许可证清单重生成：Go 43 模块 / Rust 428 crates / npm 1 包，UNKNOWN 0 项。
  **唯一未能实跑**：`pnpm audit`——镜像不提供 audit 端点（`ERR_PNPM_AUDIT_ENDPOINT_NOT_EXISTS`），沿用 CI 结果。

### 变更（2026-09-30 ROADMAP #17 可访问性补强：焦点陷阱 / 播报 / 表格与标签 / 组件级 axe / 对比度修色收尾）

> 证据台账 [`docs/FEATURES.md`](docs/FEATURES.md) **§BM**；本批次为 2026-09-30 会话中断批的收尾项
> （⑤ CHANGELOG 记录 + ⑥ 移出 #17 + ⑦ 全量门禁复跑）。文档 SSOT：[`docs/accessibility.md`](docs/accessibility.md)。

- **① 焦点陷阱提取为可复用组合式**：新增 [`apps/web/src/composables/useFocusTrap.ts`](apps/web/src/composables/useFocusTrap.ts)，
  `ModalDialog` / `ConfirmDialog` / `PromptDialog` / `PreviewOverlay` **四模态共用**「记住并恢复焦点 +
  初始移入 + `Tab` 首尾回卷」——此前只有 `ModalDialog` 有陷阱，键盘用户可 Tab 逃出另外三个。
  `Tab` 回卷由各模态自己的 `useKeydownStack` 栈顶 handler 调用 `trapTab(e)`，不破坏 LIFO 语义。
  初始聚焦必须 `await nextTick()` 而非 `flush: 'post'`（`v-model` 是运行时指令，其 mounted 钩子排在
  post 队列里更晚，先跑会让 `PromptDialog` 全选落空），故续体开头有一道空容器守卫。
  新增 7 条用例（三个模态各补焦点恢复 + `Tab` 回卷，`ConfirmDialog` 另补「打开后同 tick 内卸载不抛错」）。
- **② 操作成功 / 失败统一播报 + `aria-live` 补断言**：成功统一走 `toast()` → `Toasts` 的
  `aria-live="polite"` 容器（新增「成功 / 失败同区播报」用例）；失败的**内联横幅**统一
  `role="alert"`（`ObjectsPanel` / `AccountsPanel` ×2 / `BucketsPanel` / `CompareDialog` /
  `MigratePanel` / `RecycleBinPanel` / `ServerPanel` 的 `msg err` + `PromptDialog` 校验失败），
  并由 `src/a11y_gate.test.ts` **源码门禁**钉住「每个 `msg err` / `modal-err` 开标签必须带
  `role="alert"`」（变异验证：去掉 `ServerPanel` 的 `role` → 红灯点名该标签 → 还原绿）。
  `BatchMetadataDialog` 状态区的 `aria-live` 也补了行为断言。
- **③ 表格 `caption` / 选中态语义 / 可见标签**：`src/components` 下 **15 张 `<table>` 全部带
  sr-only `<caption>`**（`styles.css` 新增全局 `.sr-only` 工具类）；四张有行选中的表给数据行加
  `:aria-selected`（`aria-multiselectable` **刻意不用**——组件级 axe 判定它在原生 `<table>` 上
  非法，见下）；**18 个只有 placeholder / `title` 的控件补可见标签**，三种形态：包裹
  `<label class="field">`（批量元数据 3 个，原硬编码英文 `aria-label` 改为与可见标签**同一条 i18n 键**）、
  行内 `<label for>`（标签 / HTTP 头 / CORS 键值行、账号与桶切换下拉——紧邻的可见徽标直接改成
  `<label for>`，**零视觉改动**、工具栏路径与过滤框）、`aria-labelledby` 指向**可见列头**
  （`BucketTags` / `LifecycleDialog`——一个 `<th>` 无法 `for` 到 N 行输入）。
  现存可见表单控件 100% 有可关联标签，仅剩 2 个 `display:none` 的文件选择框（不可见、由带标签的
  拖放区触发）。
- **④ 组件级 `vitest-axe` 扫描**：新增 **devDependency** `vitest-axe`（`dependencies` 仍只有 `vue`，
  不违 [ADR-004](docs/decisions/0004-minimal-frontend-deps.md)）与 [`apps/web/src/a11y_axe.test.ts`](apps/web/src/a11y_axe.test.ts)
  ——对**挂载后的组件 DOM** 跑与 `e2e/a11y.spec.ts` 同口径的 WCAG 2.0 / 2.1 A + AA 规则集，
  覆盖 E2E 到不了的边界态（弹窗、toast 堆叠、带选中行的列表），**serious / critical 即红灯**，
  另有「注入 `image-alt` 必须被报出」的空跑自检；happy-dom 无 CSS 级联故**显式关掉 `color-contrast`**。
  **它在落地当天就抓到一条真问题**：`aria-multiselectable` 写在原生 `<table>` 上属
  `aria-allowed-attr`（critical）违规，已按其判定移除。
- **⑤ 对比度修色收尾**：`apps/web/src/styles.css` 修 6 组低 AA 配对，`docs/accessibility.md` §5.5
  表**同步重算**为 **13 行、双主题各 0 行低于 AA**——`--muted` 4.44→4.57、`--primary` 3.21 / 3.02→4.85 / 4.56、
  `--placeholder` 浅 2.58→4.54 深 3.81→4.54、`--danger` 深 4.25→4.54、白字 on `--brand` 2.37 / 4.23→4.84 / 4.80。
  **`--brand` 拆 token**：该渐变同时是「白字背景」与「header logo 渐变文字」，方向相反——
  `:root` 把 `--brand-from/to` 加深到白字达标，新增 `--brand-mark-*` 专供 logo（浅色沿用加深值、
  **深色块回亮值** `#2dbd98` / `#138e69`），避免「为白字加深」反把深色主题 logo 拖到 4.1:1。
  由 `apps/server/contrast_gate_test.go` 从 `styles.css` 机械重算钉住（变异验证：把 `--muted` 改回
  `#64786f` → 红灯点名两格「记 4.57，重算 4.44」→ 还原绿）。
- **门禁实跑（2026-10-01）**：前端 `pnpm test` **75 文件 / 1155 例**、`pnpm test:coverage` 四指标
  **100%**（4321 / 2932 / 1130 / 3706）、`pnpm lint` **0 告警**、`pnpm build` OK（371.05 kB / gzip 113.84 kB）；
  后端 `gofmt -l .` 干净、`go vet ./...` 干净、`golangci-lint run` **0 issues**、
  `go test ./... -count=1` **9/9 包**、`make test-cover` **9/9 包 100%**、`govulncheck` **0 可达**、
  包根文档门禁 `go test . -count=1` ok、`make check` / `make bench` **exit 0**。
- **渲染态与真实链路复跑**（本批次改了前端，按 AGENTS 必跑）：`pnpm e2e` **22 passed / 0 skipped**
  ——其中 `e2e/a11y.spec.ts` 的 5 条 axe 扫描在**真实 Chromium + 真实构建产物**上对本批配色 /
  `role` / `caption` 改动**零 serious/critical**；`make e2e-real` **3 passed**（真实后端 + RustFS +
  真实产物；本机 8080 被 `haproxy` 占用，用 `SERVER_PORT=8081`）；`S3CLINET_E2E=1`
  **4/4 PASS**；`cargo audit --no-fetch` **0 漏洞**。
  **唯一未能实跑**：`pnpm audit`——所用镜像不提供 audit 端点
  （`ERR_PNPM_AUDIT_ENDPOINT_NOT_EXISTS`），沿用 CI 结果，已在 `ROADMAP.md` §四 如实标注。

### 变更（2026-09-30 ROADMAP #18 可观测性补全：五个缺失指标补齐 + 告警规则 + 仪表盘 + 文档同步）

> 证据台账 [`docs/FEATURES.md`](docs/FEATURES.md) **§BL**；本批次为 2026-09-30 会话中断批的
> 收尾段（⑤ CHANGELOG 记录 + ⑥ 移出 #18 + ⑦ 包根门禁复跑）。

- **五个指标补齐**（发射点 `apps/server/internal/handler/metrics.go`，仍为无 prometheus 客户端库的
  手写 `fmt.Fprintf`）：① `s3c_store_write_failures_total`（counter，只数真实落盘 / SQL 写失败——
  重复 ID、`NotFound` 等业务拒绝不计数；`json` / `encrypted` 驱动 `Ping` 恒 nil，这是它们唯一的主动
  故障信号）；② `s3c_jobs_active`（gauge，未终结任务数，与 `JobRegistry` 256 在册上限同口径）；
  ③ `s3c_http_request_duration_seconds`（histogram，桶上界 `0.005 … 30` + `+Inf`，`+Inf` 与 `_count`
  都取同一次 `recordHTTPMetric` 的 `s3c_http_requests_total`，结构上不漂移）；④
  `s3c_volume_size_bytes` / `s3c_volume_free_bytes`（gauge，`S3C_DATA_DIR` 的 statfs /
  `GetDiskFreeSpaceExW`；**取不到就不发序列**，不用 0 冒充）；⑤ `s3c_last_shutdown_duration_seconds`
  （gauge，**上一次**关停耗时——关停在进程退出前发生、scrape 赶不上 → 落盘 `data/shutdown.json`
  `{"durationUs":N}`，下次启动载入后暴露，0 = 尚无记录；单位取微秒，避免亚毫秒关停被截断成 0
  与「无记录」撞车）。
- **代码落点**：`handler/metrics.go` 新增 `recordHTTPMetric(status, dur)` 与五个输出块、
  `middleware.go` 的 `withLogging` 传入 `dur`（计数与直方图同源）、`handler.go` 新增 `dataDir` /
  `lastShutdown` + `SetDataDir`；新增 `handler/shutdown_metric.go`、`volume_unix.go`（`linux ||
  darwin || freebsd`）、`volume_windows.go`（kernel32 `GetDiskFreeSpaceExW` 延迟绑定——标准库
  `syscall` **没有**该 API，`golang.org/x/sys` 违 ADR-004）、`volume_other.go`；`store/metrics.go`
  包级 `writeFailures` + `WriteFailureCount()`，`filestore.go` `persistLocked` 与 `sqlite.go` 三处
  写失败分支各记一次；`service/job.go` 新增 `JobRegistry.ActiveCount()`；`main.go` 接线
  `SetDataDir` / `LoadLastShutdown` 并在 `srv.Shutdown` 返回后 `RecordShutdown`。
- **告警与仪表盘**：[`deploy/prometheus/s3client.rules.yml`](deploy/prometheus/s3client.rules.yml)
  新增记录规则 `s3clinet:http_latency_p95:rate5m` 与 4 条告警（`S3ClinetStoreWriteFailures` /
  `S3ClinetJobsNearCapacity` / `S3ClinetVolumeSpaceLow` / `S3ClinetHTTPLatencyHigh`）；
  [`deploy/grafana/s3client.dashboard.json`](deploy/grafana/s3client.dashboard.json) 行⑥由「没有面板」
  改写为真面板 + 新行⑦，面板 31 → 40（两道既有门禁 `TestPrometheusRulesReferenceRealMetrics` /
  `TestGrafanaDashboardReferencesRealMetrics` 同步钉住「规则与面板引用的指标必须真实发射」）。
- **测试（先红后绿）**：`metrics_test.go` +5 用例与助手、`TestMetricsEndpointExposed` 指标名清单扩至
  五个新指标；`store/gaps_test.go` `TestStoreWriteFailureCounter`；`service/job_cap_test.go`
  `TestJobRegistryActiveCount`；`main_test.go` `TestMainServerSubprocess` 断言子进程退出后
  `shutdown.json` 存在且 `durationUs ∈ (0, 5e6)`。
- **文档同步**：[`docs/OPERATIONS.md`](docs/OPERATIONS.md)（§3.2 表 +6 行、直方图桶上界、观测缺口段改
  「已补齐」、§4.1 +3 SLI、§4.2 +4 告警、§4.3、§10.3、R-9、组件表、备份表 +`shutdown.json`）、
  [`docs/api.md`](docs/api.md) 指标条目、[`docs/DEPLOYMENT.md`](docs/DEPLOYMENT.md) §6.3、
  [`docs/CONFIGURATION.md`](docs/CONFIGURATION.md) `S3C_DATA_DIR` 行。
- **门禁实跑**（2026-09-30）：`go vet ./...` 干净 / `golangci-lint run` **0 issues**（含
  `GOOS=windows` 同为 0）/ `go test ./... -count=1` **9/9 包** / `-race -coverprofile` awk `count==0`
  **9/9 包 100.0%、无未覆盖块** / 交叉编译 windows·darwin·freebsd·openbsd build OK / 包根文档门禁
  `go test . -count=1` ok。**变异验证 5/5**（规则错名 → 红、仪表盘 `expr` 错名 → 红、去掉
  `_sum` 累加 → 红、去掉 `RecordShutdown` → 红、去掉 `noteWriteFailure` → 红，均点名后还原绿）。
  前端本轮未改，`pnpm test` / `pnpm build` **未跑**（Task D 需跑）。

### 修复（2026-09-30 KNOWN_ISSUES #70 闭环：`NormalizeEndpoint` 幂等修复，推翻 ➖ 决策）

> 证据台账 [`docs/FEATURES.md`](docs/FEATURES.md) **§BK**；问题登记见
> [`docs/KNOWN_ISSUES.md`](docs/KNOWN_ISSUES.md) #70（已闭环移除，编号不重排）。

- **缺陷**：`NormalizeEndpoint("00  /")` → `http://00  `（尾随空白），再归一 → `http://00`，**不幂等**；
  `"host/a  /"` / `"http://host  /"` 同理。根因是 `strings.TrimRight(rest, "/")` 在 host/path 切分
  **之前**执行，把尾斜杠后的内部空白顶到结果末尾，而入口 `TrimSpace` 只做一次——函数头承诺的
  「去首尾空白」与幂等契约（`SameEndpoint` 与建 client 共用）自违。原登记为 ➖「fail-closed 维持
  现状」，本批**复核 4 处调用点影响面后推翻该决策**。
- **修法**（登记内已写死的最小改动，未改其它语义）：`internal/s3wrap/client.go` 的
  `NormalizeEndpoint` 改为**先切分 host/path** → host `TrimSpace` → path 去尾部斜杠与空白
  （`TrimRightFunc`，`/` ∥ `unicode.IsSpace`），保输出不以空白结尾、对自身幂等。
- **TDD 与验证**：`ssrf_fuzz_test.go` 新增 `TestNormalizeEndpointIdempotent`（7 例，先**红灯 3 例**
  点名 → 实现后绿）；变异验证（去掉 host `TrimSpace` → 红灯点名两例 → 还原绿，命令写在测试头）；
  既有用例不改一字即绿；两目标各 10s 有界 fuzz **PASS**（678,909 + 246,990 次执行）；fuzz 内
  修复后不可达的「尾随空白短路分支」按死代码零容忍删除，两枚退化种子保留在 corpus 作回归。
- **门禁实跑**（2026-09-30）：`gofmt` 干净 / `go vet` 0 告警 / `go build` OK /
  `golangci-lint` **0 issues** / `go test ./... -count=1` **9/9 包** / `internal/s3wrap` **100.0%** 覆盖率。

### 变更（2026-09-30 ROADMAP #19 收尾：文档可读性与流程机械化 4 项 + 两道新门禁）

> 证据台账 [`docs/FEATURES.md`](docs/FEATURES.md) **§BJ**；本批次为 2026-09-30 会话中断批的收尾段
> （⑤ CHANGELOG 记录 + ⑥ 移出 #19 + ⑦ 门禁全量复跑）。

- **4 项缺口落地**：① [`docs/architecture.md`](docs/architecture.md) §5 数据流补 **mermaid 时序图**
  （对象上传直传 + multipart 两段，图中链路逐条对 `handler/multipart.go` 核对）；②
  [`llms.txt`](llms.txt) 两行（本文件 / `docs/FEATURES.md`）就地补 `⚠️ 超大`（>200 KB）+ 按
  `## [<版本>]` 段 / §字母章节的检索建议；③ 复审周期口径由「建议值、未在 CI 强制」改为
  **「有机械提醒」**——新门禁 [`apps/server/doc_review_gate_test.go`](apps/server/doc_review_gate_test.go)
  按登记表「N 个月」周期到期即红灯点名（[`docs/README.md`](docs/README.md) 维护表同口径改写）；
  ④ [`.github/PULL_REQUEST_TEMPLATE.md`](.github/PULL_REQUEST_TEMPLATE.md) 验收清单新增
  「决策（补 ADR）」勾选行（与 `adr_coverage_gate_test.go` 取舍行断言同一口径）。
- **两道新门禁红→绿 + 变异实跑**：`llms_size_gate_test.go` 首轮**红灯点名两行** → 补标记转绿；
  **变异**删 CHANGELOG 行标记 → 红灯「目标 319 KB 超过 200 KB 阈值却无体量警示」→ 还原绿灯
  （扫描面 43 个文件链接）。`doc_review_gate_test.go` 首轮绿（21 行 / 14 行带月数 / 0 过期）；
  **变异**把一行日期改成 `2025-01-01` → 红灯「+ 3 个月 = 2025-04-01，今天 2026-09-30」→ 还原绿灯。
- **台账收口**：[`docs/ROADMAP.md`](docs/ROADMAP.md) §三 3.2 按 §六 第 1 条**移出 #19**（#19 转空号、
  编号不重排，证据指针写入 3.2 前言空号清单）；[`docs/DEVELOPMENT.md`](docs/DEVELOPMENT.md) §3
  （两道新门禁落点）与 §4（复审周期机械提醒）同批同步。

### 变更（2026-09-30 文档归档：首份已填写的事故复盘冻结入 `docs/archive/` + 归档触发规则入档）

- **新增首份事故复盘并按「归档操作」四步冻结**：按 [`docs/POSTMORTEM_TEMPLATE.md`](docs/POSTMORTEM_TEMPLATE.md)
  §1–§10 填写 [`docs/archive/incident-20260916-presign-empty-url.md`](docs/archive/incident-20260916-presign-empty-url.md)
  （`PM-20260916-presign-empty-url`，**near-miss 示例**）——`b3b287c`（2026-09-05 为达 100% 覆盖率删除被误判
  「不可达」的预签名错误分支）→ 签名失败回 `200 {"url":""}`；2026-09-16 评估实测发现（`ASSESSMENT` L1 /
  `KNOWN_ISSUES` #23）、2026-09-17 `5954bfa` 统一 `writePresignResult` + 源码门禁 `TestPresignErrorsNotSwallowed`
  闭环、2026-09-19 `0fbd560` 收口 #23 余项。四步执行：① `git mv` 落位 `docs/archive/`（**新文件无既往历史可保**，
  等价于直接落位，不复制、不删了重加）；② 引用收敛——[`docs/POSTMORTEM_TEMPLATE.md`](docs/POSTMORTEM_TEMPLATE.md)
  （原「尚无已填写的复盘实例」改指首例 + 取证纪律指针）、[`docs/OPERATIONS.md`](docs/OPERATIONS.md) §9.3 第 6 步、
  [`docs/README.md`](docs/README.md)「找冻结的历史快照」行与文档维护表；③
  [`docs/archive/index.md`](docs/archive/index.md) 归档清单登记一行、「已归档 4 份」→ **5 份**；④ 本条。
  **正文字段全挂 git 取证，无运行期实测值的字段写「未留存 + 原因」，未编造时间戳 / 日志 / 指标 / 报障记录**。
- **归档触发规则入档**（三处同口径）：[`docs/archive/index.md`](docs/archive/index.md) 头注、
  [`docs/DEVELOPMENT.md`](docs/DEVELOPMENT.md) §4「归档」条、[`docs/README.md`](docs/README.md) 文档维护表——
  **已填写的事故复盘在事件闭环、行动项登记完成后按「归档操作」四步冻结**；首例已于 2026-09-30 执行，
  当前**无待归档例外**。
- **闭环复核实测**（记录在该复盘 §8）：`go test ./internal/handler/ -run 'TestWritePresignResult|TestPresignErrorsNotSwallowed' -count=1`
  **3/3 PASS**；变异把 `objects.go` get 分支改回 `u, _ := ...` → `TestPresignErrorsNotSwallowed` **红灯点名**
  「`objects.go:418` 忽略了预签名错误（会返回 200 + 空 url）」→ 还原后绿灯、`git status` 无残留。

### 修复（2026-09-30 第二轮收口：ROADMAP ✅ 行移出 + §AU 过期注记 + 评估残留缺口补登记 + §四 基线复测）

> 背景：第一轮清完「未实跑」标记后继续逐台账回读，发现 4 类残留——台账合规、过期状态、
> 从未落账的评估缺口与过期的基线日期（证据见 [`docs/FEATURES.md`](docs/FEATURES.md) **§BI**）。

- **ROADMAP §六 第 3 条合规**：§三 3.2 **#14**（原生 fuzz，2026-09-30 已落地 ✅）按「禁止保留已完成
  条目的 ✅ 行」移出，编号不重排、#14 转空号，证据指针（[`docs/FEATURES.md`](docs/FEATURES.md) §BG）
  保留在 3.2 前言空号清单——与 #3 / #12 同口径。
- **FEATURES §AU 过期状态加注**：「未推送、未合并」为落笔时点状态，该批改动已随 `a984df7` 进入
  `develop` 且与 `origin/develop` 一致；原文保留（历史值不改写纪律）+ 事实注记，「需确认」流程是否
  履行不代答。
- **补登记 4 项评估残留缺口** → [`docs/ROADMAP.md`](docs/ROADMAP.md) §三 3.2 新行 **#19**：
  ① `docs/` 0 张图表（`mermaid` / `plantuml` 全目录零命中）；② 超大文档无体量警示（CHANGELOG 323 KB /
  FEATURES 247 KB 平铺进 `llms.txt`）；③ 文档复审周期「建议值，未在 CI 强制」；④ PR 模板验收清单缺
  「新增 / 变更决策是否补 ADR」勾选项——此前评估识别但从未落账，按「不留口头待办」补账。
- **§四 门禁基线复测回填**：改写为「2026-09-30 复测」并明示复测范围（`gofmt` / `vet` / lint / test /
  build / `govulncheck` 0 可达 + 前端 lint / typecheck / test / build 全绿）与未重跑项（覆盖率 / E2E /
  audit 类沿用行内最近实测值），不虚报覆盖范围。

### 修复（2026-09-30 门禁实跑回填：4 处「未实跑」标记清零 + FEATURES §BG 结构归位）

> 背景：[`docs/FEATURES.md`](docs/FEATURES.md) §BD / §BF 及本文件同批两条目落笔时会话无 shell，
> 留下 4 处「⚠️ 门禁未实跑待人类实跑」标记；同批并行落笔的 §BG 被追加到 FEATURES 文件末尾、
> 错挂在 §三「质量与覆盖率现状」**之后**（字母台账应全部位于 §二），目录行与头部摘要也缺
> §BF / §BG 登记——属「登记两处同改」要防的同类漂移。

- **门禁全量实跑并回填实测值**：后端 `go vet` / `go build` / `go test ./...`（**9/9 包全绿**）+
  `golangci-lint`（**0 issues**）；前端 `pnpm lint` + `pnpm test`（**74 文件 / 1132 例全绿**）+
  `typecheck:e2e` + `pnpm build`（366.72 kB，gzip 112.72 kB）全绿。FEATURES §BD / §BF 与本文件
  两条目的 ⚠️ 标记改为实测值，
  [`apps/server/docs_naming_gate_test.go`](apps/server/docs_naming_gate_test.go) 文件头同步去掉「未实跑」。
- **两处变异复核实跑**（完整证据见 [`docs/FEATURES.md`](docs/FEATURES.md) §BH）：
  ① `docs_naming_gate` 摘 `AGENTS.md` 命名清单的 `AGENT_EVALS.md` →
  `TestDocsNamingConventionRegistersEveryDocsFile` 红灯点名 → 还原绿灯；
  ② `contrast_gate` 把 §5.5 计数 `共 **11 行**` 改 10 → `TestContrastRecordCountClaimsMatch`
  红灯点名 → 还原绿灯。
- **FEATURES 结构归位**：§BG 整块移回 §二 末尾（§BF 与 §三 之间，**纯移动 33 行、内容零改动**，
  `git diff --stat` 33 insertions / 33 deletions）；目录行补 BF / BG / BH，头部摘要补 §BG / §BH。

### 新增（2026-09-30 AI 时代文档补强：六路并行收口 P0–P2 缺口）

> 起因：以「AI 时代成熟仓库文档基线」对本仓做实测盘点，逐层核对后确认散文层已饱和，
> 剩余缺口集中在「机器可读深度 + AI 效果证据 + 英文覆盖」。本批由 6 条并行工作流收口，
> **每条新增能力都配机械门禁，且门禁全部做过变异验证**。

- **安全**：新增 [`.well-known/security.txt`](.well-known/security.txt)（RFC 9116 机器可读漏洞披露入口，
  `Expires` 到期即红灯强制续期），门禁 `apps/server/security_txt_gate_test.go`。
- **接口契约**：`docs/api/openapi.json` 补全示例——37/37 有请求体的 operation 均有请求示例、
  70/70 operation 均有 2xx 示例、5 个共享 schema 均有 `example`；[`docs/api.md`](docs/api.md) 新增
  「请求示例（curl）」12 条覆盖全部 10 个 tag；门禁 `apps/server/openapi_examples_gate_test.go`
  （含示例字段的 schema 形状校验与空扫自检）。
- **数据模型**：新增 [`docs/data-model.md`](docs/data-model.md)——`model.Account` 12 字段、`json` /
  `sqlite` / `encrypted` 三驱动、`S3C2`/`S3C3` 信封字节布局、原子写与权限、fail-closed 与单写者约束；
  §0 明确「非 SSOT，冲突时回退 schema / ADR / 代码」；门禁 `apps/server/data_model_gate_test.go`
  （反射 `model.Account` + 解析 `store.Open` 的 switch 驱动名）。
- **可观测性**：新增 [`deploy/grafana/s3client.dashboard.json`](deploy/grafana/s3client.dashboard.json)
  （31 面板，覆盖 3 条 recording rule 与 §4.1 的 SLI），门禁 `apps/server/grafana_dashboard_gate_test.go`
  校验指标 / `code` / recording rule 真实存在；[`docs/OPERATIONS.md`](docs/OPERATIONS.md) §4 由
  「未提供仪表盘」改为落地路径与导入步骤，并新增 §6.5 密钥轮换 Runbook。
- **性能**：新增 `apps/server/bench_budget_test.go`（分配确定性预算 + 极宽耗时兜底）、
  `.github/workflows/perf.yml`（周一 03:00 UTC 留存原始基准）与 `make bench`；口径与「刻意不测什么」
  写在 [`docs/PERFORMANCE.md`](docs/PERFORMANCE.md) §4.2。
- **AI 效果证据**：新增机器可读黄金任务集 [`scripts/evals/golden-tasks.yaml`](scripts/evals/golden-tasks.yaml)
  与 runner [`scripts/evals/run-golden-task.sh`](scripts/evals/run-golden-task.sh)；
  [`docs/AGENT_EVALS.md`](docs/AGENT_EVALS.md) §六首次落台账（含一次 `fail` 基线如实保留），
  并显式声明「无 golden task 端到端实跑，五维记为 N/A ≠ 满分」。
- **英文覆盖**：新增 [`docs/en/README.md`](docs/en/README.md)（英文导航落地页）与
  [`docs/en/architecture.md`](docs/en/architecture.md)（`architecture.md` 全文翻译，页头声明中文 SSOT 与
  来源 revision）；[`docs/i18n.md`](docs/i18n.md) 新增 §7「文档翻译覆盖政策」（SSOT / 优先级 / 快照标记 /
  漂移处理 / 诚实覆盖现状）；门禁 `apps/server/en_docs_gate_test.go`。
- **无障碍**：新增 [`apps/web/e2e/a11y.spec.ts`](apps/web/e2e/a11y.spec.ts)——axe 在**真实浏览器 + 真实
  构建产物**下扫 4 个界面状态（浅色初始态 / 新增登录对话框 / 服务器设置面板 / 深色主题），
  serious / critical 违规即红灯，另有 1 条「注入已知违规必须被报出」的空跑防护；
  同轮修掉 axe 实测出的 2 组对比度不达标（`--ok` 2.55→4.95、`--danger` 4.41→5.91），
  并把 §5.5 静态记录钉在源码上（`apps/server/contrast_gate_test.go`，表值与计数重算比对）。
- **原生 fuzz**：[`apps/server/internal/{s3wrap,store,handler}`](apps/server/internal/) 新增 fuzz 目标
  （端点归一化 / SSRF、`S3C2`/`S3C3` 信封、桶策略 JSON 与文件名边界），
  `.github/workflows/fuzz.yml` 做有界探索（PR 只跑种子语料，语义不变）。
- **元信息**：新增 [`CITATION.cff`](CITATION.cff)（GitHub 引用元数据，根目录工具固定名）与
  [`.github/ISSUE_TEMPLATE/documentation.md`](.github/ISSUE_TEMPLATE/documentation.md)。
- **修正**：`internal/store` 中把当前写入格式误写为 `S3C2` 的注释改为 `S3C3`（实际
  `envelope()` 恒写 `S3C3`，`S3C2` 仅只读兼容），与 [`docs/data-model.md`](docs/data-model.md) 对齐。


### 修复（2026-09-30 docs 内记录的待修项收口：文档失真 1 处 + 对比度静态核查 + 改进项登记 SSOT）

> 背景：按「docs 里记录了待修的也要处理」逐篇清点非冻结文档，三类记录在案的待修项：
> ① [`docs/accessibility.md`](docs/accessibility.md) §5.1 第 9 条仍写「动效偏好是已知缺口、仍会播放」，
> 与同文件 §4 第 5 条（#67② 已修）**自相矛盾**；② §4 第 4 条声明「没有对比度专项核查记录」；
> ③ §5.3/§5.4 的改进项与 [`docs/OPERATIONS.md`](docs/OPERATIONS.md) 观测缺口在文档正文里当
> **口头待办**挂着，违反「两源分工 / 不留文档内待办」纪律。

- **文档失真**：`accessibility.md` §5.1 第 9 条改为**回归项**表述（#67② 已修；若仍播放即回退，
  先看 `apps/web/src/a11y_gate.test.ts` 是否被绕过）。
- **对比度静态核查记录 + 数值门禁**：新增 [`docs/accessibility.md`](docs/accessibility.md) §5.5——按
  WCAG 2.1 相对亮度对 `styles.css` 设计 token 做前景 / 背景配对计算（深色 `rgba()` 覆层按 alpha
  合成到 `--panel`），11 行双主题比值入表；同轮新增
  [`apps/server/contrast_gate_test.go`](apps/server/contrast_gate_test.go)，把「表值 / 计数声明 ↔
  `styles.css`」**机械重算比对**（改色不重算即红灯；含 `--brand-from` 注释数字；变异复核步骤见文件头）。
  对账中纠偏 2 处口径失真：表里 `--danger` 5.98 / `--ok` 5.02 是渲染态读数、不是表中声明的 token
  公式值（应为 **5.91 / 4.95**，`styles.css` 注释同改）；「4 组低于 AA」计数漏行，按表格实况钉为
  **共 11 行、浅色 5 行 / 深色 3 行低于 AA**。
- **改进项登记收口（两源分工）**：`accessibility.md` §5.3/§5.4 与 `OPERATIONS.md` 观测缺口的待办迁入
  [`docs/ROADMAP.md`](docs/ROADMAP.md) §三 **#17**（可访问性补强与自动化扫描）/ **#18**（可观测性
  补全 5 指标），原文档改为指针；`ROADMAP` §三 #12「供应链证明」已 2026-09-29 落地、按 §六 第 1 条
  移出候选池（编号不重排、#12 空号），3.2 节「全部 ⬜ 未排期」等口径随之修正。
- **新文档登记补全**：并行批次新增的 [`docs/data-model.md`](docs/data-model.md)（数据模型与存储格式
  地图 + `apps/server/data_model_gate_test.go`）落地时**四个导航面未登记**（`doc_index_gate` 与命名
  同步门禁会红）——同轮补进 [`docs/README.md`](docs/README.md) 导航、`AGENTS.md` / `DEVELOPMENT.md`
  §4 命名清单与登记表、[`llms.txt`](llms.txt)，并把「账号存储格式」同步表行补上该地图。
- **两条尾巴结清**：① `data-model.md` 登记**去重核对**完成——`docs/README.md` 导航 / `AGENTS.md` /
  [`docs/DEVELOPMENT.md`](docs/DEVELOPMENT.md) §4 / [`llms.txt`](llms.txt) 四面各登记一次、无重复；
  ②「改 `styles.css` 后 §5.5 表须重算」由口头约定升级为 `contrast_gate_test.go` 机械保证（见上条）。
- ✅ **门禁实跑回填（2026-09-30 16:08 CST）**：`cd apps/server && go test . -count=1` 全绿（含
  `contrast_gate`）；前端 `pnpm lint` + `pnpm test`（74 文件 / 1132 例）+ `pnpm build` 全绿；
  `contrast_gate` 计数变异复核已实跑（§5.5 `共 **11 行**` 改 10 → 红灯点名 → 还原绿灯）。

### 修复（2026-09-30 导航收口残留：命名约定两处分叉 + `llms.txt` 目录摘要 + README AI 入口 + 机械门禁）

> 背景：2026-09-30 文档基线补缺六项（[`docs/FEATURES.md`](docs/FEATURES.md) §AY–§BC、§BE）落地后，
> 逐面回读比对仍剩三处「只差一条登记」的分叉（均为本轮实读发现）：`AGENT_EVALS.md` 进了
> [`docs/DEVELOPMENT.md`](docs/DEVELOPMENT.md) §4 命名清单却漏了 [`AGENTS.md`](AGENTS.md) 的清单
> （「同 PR 两处同改防分叉」是硬规则），`AGENTS.md` 小写清单另漏 `en/` 子目录；[`llms.txt`](llms.txt)
> 「目录」段命名摘要落后一代名单（缺 5 个元文档、5 个内容文档与 `en/`）；根 [`README.md`](README.md)
> 「贡献与治理」缺 `AGENT_EVALS.md` 入口（英文快照 [`docs/en/index.md`](docs/en/index.md)、
> [`docs/README.md`](docs/README.md)、`llms.txt` 均有）。

- [`AGENTS.md`](AGENTS.md) 命名约定两处补齐（大写清单 + `AGENT_EVALS.md`、小写清单 + `en/`）；
  [`llms.txt`](llms.txt)「目录」段命名摘要重写为全量名单（元文档 12 + 内容文档 9 + 子目录 4，
  与 AGENTS / DEVELOPMENT §4 同口径；稍后落地的 `data-model.md` 由下方「新文档登记补全」追加）；
  根 [`README.md`](README.md) 补「AI 代理评测与贡献度量」入口。
- 新增 [`apps/server/docs_naming_gate_test.go`](apps/server/docs_naming_gate_test.go)：`docs/` 顶层
  每个 `.md` 与每个**含 `.md` 的子目录**，名字必须同时登记进三处命名口径（AGENTS 命名约定段 /
  DEVELOPMENT §4 命名约定段 / `llms.txt` 目录段；**只认反引号登记形态**，散文提及不算），含扫描面
  自检阈值。**变异复核步骤**：摘掉 AGENTS 命名清单的 `AGENT_EVALS.md` → 红灯点名该文件 → 还原绿灯。
- [`apps/server/AGENTS.md`](apps/server/AGENTS.md) 的门禁清单补限定语（「常用几道，全量以包根
  `*_gate_test.go` 为准」）并登记新门禁——此前该清单只列 5 道、实有 13 道，读起来像全量。
- ✅ **门禁实跑回填（2026-09-30 16:08 CST）**：`cd apps/server && go test . -count=1` 全绿、
  `golangci-lint run` **0 issues**；`docs_naming_gate` 变异复核已实跑（摘 AGENTS 命名清单的
  `AGENT_EVALS.md` → `TestDocsNamingConventionRegistersEveryDocsFile` 红灯点名 → 还原绿灯）。

### 新增（2026-09-30 客户端支持矩阵：浏览器 / 桌面 OS 支持范围与证据等级）

> 背景：10 层文档基线盘点（FEATURES.md §AN）时，第 6 层「用户文档」的唯一缺口是**没有客户端
> 支持矩阵**——「哪些浏览器 / 操作系统能用、哪些只是没测过」此前无文档可查。

- 新增 [`docs/compatibility.md`](docs/compatibility.md) §6.2「客户端支持矩阵（浏览器与操作系统）」
  （§6 改为「兼容矩阵」，原 S3 服务端内容降为 §6.1）：Web 端 Chromium 系 ✅（唯一有自动化覆盖）、
  Firefox / Safari ⚠️（blob 兜底、全站未实测）、移动端 ❓ 未验证、旧浏览器 / IE ❌；「语法与特性
  基线」给出建议最低版本（Chrome ≥87 / Edge ≥88 / Firefox ≥78 / Safari ≥14，标注**构建目标推导、
  未逐版本实测**）；桌面端按 OS 列出产物与架构口径。每条声明带文件与行号依据，证据等级沿用
  OPERATIONS.md §1.1（代码 / CI 文件 · 构建目标推导 · 上游行为）。
- [`docs/user-guide.md`](docs/user-guide.md) §十 桌面端新增一行指向矩阵的链接。
- 刻意**不重编号** §7/§8：POSTMORTEM_TEMPLATE.md 与 GOVERNANCE.md 对 §7 / §8 有散文引用
  （链接门禁不覆盖），重编号会使其失真——矩阵以 §6.2 小节落地。

### 新增（2026-09-30 ADR 覆盖补足：8 篇决策记录 + 取舍表覆盖门禁）

> 背景：docs/architecture.md §7「关键取舍」表原只有 4 行（ADR-001..004），§2「关键机制」表里大量
> 已落地决策（SSE 异步任务 / 账号存储三驱动 / 预签名直传 / 有界并发 / ZIP 流式 / 单实例 / REST 无
> 版本前缀）没有对应 ADR，查「当初为什么这么定」无据可查。

- 新增 [`docs/decisions/0005-sse-async-jobs.md`](docs/decisions/0005-sse-async-jobs.md) 等 **8 篇 ADR**
  （0005–0012），每条「现状」回读源码逐条核实并给出路径，不可核实处显式标「未验证 / 回溯补记」；
  Date 取自 `git log --diff-filter=A` 首次引入日期。
- [`docs/architecture.md`](docs/architecture.md) §2 关键机制表 8 行补 ADR 链接、§7 关键取舍表扩至
  12 行且每行带决策链接；[`docs/decisions/index.md`](docs/decisions/index.md) 登记 8 行。
- 新增 [`apps/server/adr_coverage_gate_test.go`](apps/server/adr_coverage_gate_test.go)：断言 §7 取舍表
  每一行都含指向 `docs/decisions/` 的链接（纯函数解析 + 5 条合成源码口径用例；TDD 先红后绿 +
  变异验证：摘 ADR-007 链接 → 红灯点名该行 → 还原绿灯）。

### 新增（2026-09-30 供应链收口：OpenSSF Scorecard + PR 依赖审查）

> 背景：第 8 层「安全与供应链」盘点后的两块空白——仓库内部门禁齐全（Trivy / govulncheck /
> cargo audit / CodeQL / cosign / SBOM / provenance），但仓库外部健康度评分（OpenSSF Scorecard）
> 与 PR 时点的依赖 diff 审查此前完全没有。

- 新增 [`.github/workflows/scorecard.yml`](.github/workflows/scorecard.yml)：OpenSSF Scorecard——
  `schedule` 每周六 02:00 UTC + `workflow_dispatch`；checkout（`persist-credentials: false`）→
  `ossf/scorecard-action`（`results_format: sarif` + `publish_results: true`）→ `actions/upload-artifact`
  （SARIF 保留 5 天）→ `github/codeql-action/upload-sarif`（code scanning 面板）。权限：顶层
  `permissions: read-all`，仅 analysis job 持 `security-events: write` + `id-token: write`
  （发布到 api.scorecard.dev 所需 OIDC）。
- 新增 [`.github/workflows/dependency-review.yml`](.github/workflows/dependency-review.yml)：
  每个 `pull_request`（含 dependabot PR）对依赖 diff 做漏洞 / 许可证审查
  （`actions/dependency-review-action` v5；默认 fail-on-severity: low / fail-on-scopes: runtime /
  license-check: true）；权限 `contents: read`。
- 两枚新 SHA 经 GitHub API 三重核验（2026-09-30）：`ossf/scorecard-action` v2.4.4 →
  `2d1146689b8cda280b9bc96326124645441f03bc`、`actions/dependency-review-action` v5.0.0 →
  `a1d282b36b6f3519aa1f3fc636f609c47dddb294`；其余 action 复用仓库内既有 SHA。
- 文档同步：[`docs/threat-model.md`](docs/threat-model.md) §5（新增 §5.5、§5.4 补缺口、开头 CI 门禁行更新）。
- 门禁实测：`TestWorkflowActionsAreShaPinned` 8 workflow / 56 引用全 SHA；链接 / 锚点门禁全绿。

### 修复（2026-09-30 状态台账 #69 收口：CHANGELOG ↔ git tag 一致性 + 机械门禁）

> 背景：`git tag` 有 3 个时间戳 tag 无对应版本段、3 个 09-01 快照段与 0.1.0 / 0.2.0 段无对应 tag、
> `[1.0.0]` 段日期与 tag 不符且顺序非倒序（[`docs/KNOWN_ISSUES.md`](docs/KNOWN_ISSUES.md) #69，
> 2026-09-29 盘点时发现）。

- CHANGELOG 顶部新增「tag ↔ 版本段对应关系（唯一台账）」：8 行快照登记（5 个快照段 + 3 个快照 tag），
  正式版本由门禁正向校验；`[1.0.0]` 段日期按 tag 事实修正为 **2026-09-22**（tag 指向提交 0cfd4ef，
  **内容未改写**），并移到 `[Unreleased]` 之后恢复倒序。
- 新增 [`apps/server/changelog_tag_gate_test.go`](apps/server/changelog_tag_gate_test.go)：
  tag ↔ 段双向一致 + `[Unreleased]` 必须居首 + 映射表解析口径（合成用例）+ 扫描面自检阈值；
  tag 从 `.git/packed-refs` / `.git/refs/tags` 读文件获取（不依赖 git 命令）。TDD 先红（映射表缺失
  红灯）后绿；变异验证：删映射行 / 改段名 → 红灯点名。
- [`scripts/release-version.sh`](scripts/release-version.sh) 收尾由 echo 提醒升级为**硬检查**：
  `grep -q "^## \[$DISPLAY\]" CHANGELOG.md` 缺段即 exit 1（#69 建议处置 (c)）。
- [`docs/KNOWN_ISSUES.md`](docs/KNOWN_ISSUES.md) #69 闭环移除（§二 表行移除、§四 台账改「已闭环」、
  头部「最后更新」改 2026-09-30）。
- 门禁实测：`go test . -run 'TestChangelog' -count=1` 绿灯；`bash -n scripts/release-version.sh` 通过。

### 新增（2026-09-30 AI 产出效果证据层收口：代理评测与贡献度量）

> 背景：第 11 层「AI 时代」此前只有**过程约束**（[`docs/AI_POLICY.md`](docs/AI_POLICY.md) 的权限矩阵 /
> 披露 / DoD），没有**效果证据**——同一份 DoD 可以全绿而输出把契约改坏，仓库没有任何可复现的测量。

- 新增 [`docs/AGENT_EVALS.md`](docs/AGENT_EVALS.md)：黄金任务集 **4 条**（GT-1 新增 `/api/*` 端点 ·
  GT-2 修复 store bug · GT-3 新增 `S3C_*` 配置项 · GT-4 前端 UI 小改；每条含目标 / 前置条件 / 完成判据
  （全部映射真实门禁）/ 预期红→绿轨迹 / 失败典型形态）+ 加权评分卡（门禁全绿 40% / 契约文档同步 20% /
  TDD 证据 15% / 死代码安全 15% / 可读性分层 10%，0–4 锚点；硬门槛：门禁全绿或死代码安全任一 0 分
  直接失败）+ AI 贡献度量口径与台账（**基线 2026-09-30，不追溯编造历史**；PR 收口回填、发版前对账）+
  评测结果台账（唯一来源，不通过项转 [`docs/KNOWN_ISSUES.md`](docs/KNOWN_ISSUES.md)）。
- 新增 [`scripts/agent-eval.sh`](scripts/agent-eval.sh)：机械评测（后端 go vet / 门禁 / test / build +
  前端 lint / typecheck / test，依赖缺失显式 SKIP）+ JSON 报告 + `EVAL_RESULT` 摘要，失败非零退出；
  刻意不跑 docker / e2e-real（CI 承担）。
- 新增 [`apps/server/agent_evals_gate_test.go`](apps/server/agent_evals_gate_test.go)：五条结构不变量——
  文档三节标记 + GT 表 ≥ 3 / 脚本存在且非空壳 / AI_POLICY 回引 / PR 披露块含「占比」/ 两模板字段集一致。
  **TDD 先红后绿 + 三条变异验证**（脚本删 `go vet` 关键词、GT 表 4→2、摘除 AI_POLICY 回引 →
  均红灯点名 → 还原全绿）。
- 文档同步：[`docs/AI_POLICY.md`](docs/AI_POLICY.md) §5（「披露 → 度量 → 评测」闭环指向）与 §11
  机械保证表（含如实列出的人工部分）；[`.github/PULL_REQUEST_TEMPLATE.md`](.github/PULL_REQUEST_TEMPLATE.md)
  披露块新增 **`- 占比：`** 字段（字段集由门禁钉住两处一致）。

### 新增（2026-09-30 英文文档入口：docs/en/index.md）

> 背景：第 13 层「质量合规」的文档面只有中文；根目录只允许 4 个约定文件（README / AGENTS /
> CHANGELOG / llms.txt），英文入口按命名约定落 `docs/en/`（目录小写）。

- 新增 [`docs/en/index.md`](docs/en/index.md)：根 README 的完整英文翻译（项目概览 / 特性全量 / 架构 /
  目录 / 环境要求 / 配置摘要 / Quick Start（命令逐字保留）/ 测试 / CI / SDK 接口清单 / 中文文档索引 /
  安全说明 / License），文首声明**中文 README 为 SSOT、本页为翻译快照**，并含「Scope of English docs」
  小节（如实声明当前唯一英文文档；机器可读契约本身英文友好）。
- 导航同步：根 [`README.md`](README.md) 顶部语言切换入口、[`docs/README.md`](docs/README.md)、
  [`llms.txt`](llms.txt)、[`docs/DEVELOPMENT.md`](docs/DEVELOPMENT.md) §4（同步表 / 命名约定 / 登记表）。
- 门禁实测：链接 / 锚点门禁 0 失效；导航覆盖由登记行转绿。

### 新增（2026-09-29 AI 时代层收口：AI 治理的机械保证 + Copilot 指针入口）

> 背景：第 10 层「AI 时代新增层」此前已有 根/子树 `AGENTS.md`、`llms.txt`、`AI_POLICY.md`、
> PR 披露块、机器可读契约与门禁族，但这些 AI 治理资产**全靠人工维持**——而它们恰好最容易被静默破坏：
> 根 `AGENTS.md` 会被工具**整体注入**、超长即被截断（硬约束悄悄失效），新增子树漏写 `AGENTS.md`
> 则该子树规则对 agent 不可见，政策正文里的**事实声明**会随仓库演进失真。

- 新增 [`apps/server/ai_governance_gate_test.go`](apps/server/ai_governance_gate_test.go)：三条结构不变量——
  ① 根 `AGENTS.md` ≤ 10 KiB 且必须指向规范正文；② 每个 `apps/*` 子树必须有 ≤ 4 KiB、**回指根文件**的
  `AGENTS.md`；③ `AI_POLICY.md` 的**事实声明**与实际一致（「未提交 MCP 配置」↔ 根无 `.mcp.json`；
  PR 模板含「AI 使用披露」块；AI 工具指针文件 ≤ 2 KiB 且必须指向 `AGENTS.md`）。
  **四条变异全部实测**：把 `AGENTS.md` 灌到 20 KB → 点名超预算；新建 `apps/probe/` 无子树文件 → 点名；
  改掉 PR 模板披露块措辞 → 点名；把指针文件灌到 7 KB → 点名；还原后全绿。
- 新增 [`.github/copilot-instructions.md`](.github/copilot-instructions.md)：GitHub Copilot 的**纯指针**入口
  （指向根 `AGENTS.md` / `DEVELOPMENT.md` / `AI_POLICY.md` / 三个子树 AGENTS），**不复制任何规则**——
  复制会形成第二事实源，故由门禁把「它只是指针」钉住。
- [`docs/AI_POLICY.md`](docs/AI_POLICY.md) 新增 **§11「AI 治理的机械保证」**：把可机检的条款与门禁一一对应，
  并**如实列出没有机械保证的部分**（权限矩阵的「需确认 / 禁止」档、发布模式授权、不自动合并、不得读密钥——
  均为人工约束）。同节写明**刻意不引入**的 AI 工具文件与理由（不加 `CLAUDE.md`、不加 `llms-full.txt`、
  不提交 `.mcp.json`）。
- 文档同步：[`DEVELOPMENT.md`](docs/DEVELOPMENT.md) §4.1 加载机制（+ 指针入口与本节门禁）、§4 位置与命名、
  「文档登记表」；[`llms.txt`](llms.txt)；[`docs/README.md`](docs/README.md) 文档维护表；
  [`docs/FEATURES.md`](docs/FEATURES.md) §AX。
- 门禁实测：`go test . -count=1` 全绿（含新门禁）；链接门禁 0 失效；`golangci-lint` 0 issues。

### 新增（2026-09-29 安全与供应链收口：自动生成的许可证清单 + 依赖覆盖门禁 + 产物核验指南）

> 背景：10 层文档基线盘点时，第 7 层「安全与供应链」的唯一硬缺口是**仓库内没有任何第三方依赖 /
> 许可证清单**（只有 CI 侧 SBOM：`sbom-cyclonedx` 产物 + OCI attestation）。清单属事实，
> 手写必然漂移——故本次**用脚本从权威来源生成**，并加门禁钉住枚举完整性。

- 新增 [`scripts/gen-third-party-licenses.sh`](scripts/gen-third-party-licenses.sh)：从三处权威来源生成清单——
  Go `go list -m -json all` + 模块内 `LICENSE` 文本机械识别；Rust `cargo metadata` 的 `license` 字段
  （离线兜底：`Cargo.lock` ∩ 本地 registry 源码）；npm `apps/web/package.json` 的运行时依赖。
  识别不出**不猜**：记 `UNKNOWN` 并给出文件路径。`go mod download` 补全带 **90s 超时**（默认 GOPROXY 不可达时不会挂死）。
- 新增 [`docs/THIRD_PARTY_LICENSES.md`](docs/THIRD_PARTY_LICENSES.md)（生成物）：许可证汇总 +
  三段明细（Go 43 模块 / Rust 428 crates / npm 1 包）。实测 **Go 侧全部为宽松许可**
  （BSD-3-Clause 21 / Apache-2.0 19 / MIT 3）；Rust 侧 `MPL-2.0` 5 个与含 `LGPL-2.1-or-later` 的表达式 2 个
  已列为「需人工阅读原文」（机械关键词提示，**不是合规结论**）；当前 `UNKNOWN` **0** 项。
- 新增 [`apps/server/third_party_licenses_gate_test.go`](apps/server/third_party_licenses_gate_test.go)：
  覆盖门禁——依赖图 / `Cargo.lock` / `package.json` 里的每个包都必须出现在清单里（含扫描面自检阈值）。
  **变异验证**：删掉清单里的 `golang.org/x/sys` 与 `serde` 两行 → 红灯逐条点名；重新生成 → 绿灯。
  首版运行即抓出 4 处不一致（3 个未下载模块被生成器丢弃、1 个本地 path crate 被误当第三方），
  两侧口径均已修正并写进注释。
- [`docs/threat-model.md`](docs/threat-model.md) §5 扩写为四小节：**5.1** 锁定策略（Go/npm/Rust/Actions SHA/
  镜像 digest 与各自门禁）· **5.2** 依赖与许可证清单 · **5.3**「**消费者如何验证产物**」——
  镜像 `gh attestation verify oci://…`、`docker buildx imagetools inspect`、`cosign verify <ref>@<digest>` 与
  CycloneDX SBOM 产物；桌面端 `sha256sum -c SHA256SUMS.txt`、`gh attestation verify <file>` 与
  `cosign verify-blob`（参数逐字取自工作流注释）· **5.4** 已知缺口（桌面产物未签名不假承诺、SBOM 不入库的理由、
  OS 包不在清单范围）。
- 文档同步：根 [`README.md`](README.md) 运维与安全段、[`llms.txt`](llms.txt)、[`docs/README.md`](docs/README.md)
  导航、[`AGENTS.md`](AGENTS.md) 入口表与命名约定、[`docs/DEVELOPMENT.md`](docs/DEVELOPMENT.md) §4 同步表 +
  登记表 + 命名约定、[`docs/FEATURES.md`](docs/FEATURES.md) §AW。
- 门禁实测：`go test . -count=1`（含新门禁与导航覆盖门禁）全绿；链接门禁 0 失效；`golangci-lint` 0 issues。

### 新增（2026-09-29 元信息 / 导航收口：docs 落地页 + 导航覆盖门禁 + `.gitattributes`）

> 背景：10 层文档基线盘点时，第 2 层「元信息 / 导航」判定为部分达标——台账与 AI 入口齐全，
> 但**缺 `docs/` 目录的落地页**，且 4 个导航面（根 README「文档」段、`AGENTS.md` 入口表、
> `llms.txt`、`DEVELOPMENT.md` §4 登记表）之间**没有一致性保证**；它们已经漂移过
> （README 归档清单漏 2 项、AGENTS 入口表曾落后、DEVELOPMENT 文档清单曾漏 `SUPPORT.md`）。
> 链接门禁只能保证「已有的链接不悬空」，**不能**保证「新文档被登记进导航」。

- 新增 [`docs/README.md`](docs/README.md)：**`docs/` 目录落地页与人类导航 SSOT**（GitHub 按字面名渲染）。
  按「我要做什么」分五组（用起来 / 部署运维 / 开发贡献 / 状态规划 / 机器可读面），逐条一句话说明；
  另含「文档维护」节（命名规则、登记表、归档纪律、各文档门禁的落点），使读者不必先猜文档在哪。
- 新增 [`apps/server/doc_index_gate_test.go`](apps/server/doc_index_gate_test.go)：**导航覆盖门禁**——
  `docs/*.md` 必须登记进 `docs/README.md`；子目录文档必须出现在落地页或该子目录 `index.md`；含扫描面自检阈值。
  **TDD 先红**：落地页不存在时红灯；**变异验证**：临时新建 `docs/_nav_probe.md`（模拟「新文档忘登记」）
  → 红灯点名 `docs/_nav_probe.md（应登记进 docs/README.md）`；删除 → 绿灯。
- 新增 [`.gitattributes`](.gitattributes)：`* text=auto eol=lf` + Windows 原生脚本（`.bat`/`.cmd`/`.ps1`）
  保留 CRLF + 图片按二进制。此前仓库没有该文件，行尾取决于各人 `core.autocrlf`——Windows 检出可能整文件
  变 CRLF，甚至让 shell 脚本 / Dockerfile 因 `\r` 执行失败。入库核查：538 个跟踪文件中**无** CRLF 文本文件，
  故不触发批量重新规范化。
- 导航面同步：根 [`README.md`](README.md)「文档」段顶部加落地页指针；[`llms.txt`](llms.txt) 索引补一条；
  [`AGENTS.md`](AGENTS.md) 文档入口表加一行；[`docs/DEVELOPMENT.md`](docs/DEVELOPMENT.md) §4 命名约定
  补 `docs/README.md` 的工具固定名属性、登记表补一行。
- 门禁实测：`go test . -count=1`（含新门禁）全绿；链接门禁扫描面 **1335 条相对链接 / 73 条锚点，0 失效**（写入时实测）。

### 新增（2026-09-29 文档缺口收口：链接/锚点门禁 + 文档登记表 + ADR 模板 + 子树 AGENTS + 隐私声明）

> 背景：10 层文档基线盘点的 P1 缺口清单——此前「链接悬空」「文档无 owner」「ADR 索引与真实文件不一致」
> 「子树无 AGENTS.md」「无隐私 / 遥测声明」「DR 演练无记录格式」全靠人工纪律，没有机械保障。

- 新增 [`apps/server/doc_link_gate_test.go`](apps/server/doc_link_gate_test.go)：全仓 md **相对链接目标存在性 +
  GitHub slug 锚点存在性**门禁，含扫描面自检阈值（链接 ≥1000 / 锚点 ≥50，命中不足即红灯，防「全绿但失明」）。
  **变异验证**：把 `POSTMORTEM_TEMPLATE.md` 的链接改成不存在的路径 → 红灯点名 `文件:行号`；把
  `archive/review-2026-09-19.md` 的锚点改一个字符（等长替换保行号）→ 锚点红灯点名；两次均以 sha256 校验还原。
  现扫描面：**1231 条相对链接 + 73 条带锚点链接，0 失效**。
- 新增 ADR 模板 [`docs/decisions/0000-template.md`](docs/decisions/0000-template.md)（英文 H2 + 中文正文，
  含 `Superseded` 生命周期与 `## Update（YYYY-MM-DD）` 约定）；[`docs/decisions/index.md`](docs/decisions/index.md)
  补**日期列**与「最新更新」列，登记 ADR-003 / ADR-004 篇内的 `## Update（2026-09-19）`（原索引**漏登 0004**）。
- 新增子树规则 [apps/server/AGENTS.md](apps/server/AGENTS.md) · [apps/web/AGENTS.md](apps/web/AGENTS.md) ·
  [apps/desktop/AGENTS.md](apps/desktop/AGENTS.md)：只写子树专属硬约束（Go 分层 / 门禁入口与覆盖率口径；
  Web 零运行时依赖与覆盖率排除项；Desktop 无 IPC 与 toolchain pin），仓库级规则仍只在根 `AGENTS.md`（不复制、防分叉）。
- [`docs/DEVELOPMENT.md`](docs/DEVELOPMENT.md) §4 新增**文档登记表**（owner / 复审周期建议值 / 最后复审）：
  「最后复审」一律填「登记时基线（2026-09-29）」——**不追溯编造**历史日期；同处修掉陈旧表述
  「将来若加 `SUPPORT.md` 同理」（该文件已存在）。
- [`docs/user-guide.md`](docs/user-guide.md) 新增「十四、数据与隐私」：**不发送遥测**（全仓生产代码检索
  sentry / posthog / analytics / telemetry 等零命中；web 生产依赖仅 `vue`）、凭据存放位置（`secretKey` 仅服务端
  且响应脱敏为 `secretSet`；`S3C_TOKEN` 默认 `sessionStorage`，勾选后才写 `localStorage`）、日志不含密钥。
- [`docs/POSTMORTEM_TEMPLATE.md`](docs/POSTMORTEM_TEMPLATE.md) 增加 **§7.1 DR 演练复盘补充字段**
  （场景 / 触发方式 / 实测 RTO·RPO 对照 §7.1 **建议值** / 未覆盖路径 / 密钥可用性）——复用同一模板、不新增文件。
- [`apps/server/.env.example`](apps/server/.env.example) 补 `S3C_EXPOSE_METRICS` / `S3C_EXPOSE_OPENAPI` /
  `S3C_CSP_CONNECT_SRC`；[`docs/CONFIGURATION.md`](docs/CONFIGURATION.md) 写明两份 `.env.example` 的**分工口径**
  （根 = compose 透传项，server = 服务端全量可选项），消除「同名两份、内容各半」的双源。
- 新增 [`apps/server/dependabot_gate_test.go`](apps/server/dependabot_gate_test.go)：断言每条 Dependabot 规则的
  `directory` 存在且含对应清单（gomod→`go.mod` / npm→`package.json` / cargo→`Cargo.toml`），未登记的 ecosystem 红灯；
  **变异验证**：把 `/apps/server` 改回 `/server` → 红灯点名（对应上面「文档失真收口」的 Dependabot 修复）。

### 变更（2026-09-29 接口契约表达鉴权：文档级 `security` + 逐端点豁免 + `tags` 分组）

> 背景：[`docs/api/openapi.json`](docs/api/openapi.json) 早已定义 `components.securitySchemes.bearerAuth`，但
> **全局 `security` 为空、0/70 个 operation 声明 `security`、`tags` 为空**——代码生成器与 AI 代理从机器可读
> 契约里读不出「哪些端点要 Bearer」，而 [`docs/api.md`](docs/api.md) 明确要求。属「契约存在但鉴权意图不可判定」。

- `apps/server/internal/openapi/openapi.go`：新增 `Tag` 类型与 `Registry.AddTag`，`buildSpec` 输出文档级
  `security: [{bearerAuth: []}]`；`Op` 新增 `NoAuth`，渲染为逐 operation 的 `security: []`（覆盖文档级默认）。
- `apps/server/internal/handler/openapi_register.go` 声明 **10 个 tag** 分组（accounts / buckets / bucket-settings /
  objects / object-meta / multipart / versions / trash / migrate / system），与 `api.md` 章节对应；
  `openapi_register_system.go` 给 `/api/health`、`/api/metrics` 标 `NoAuth`（真值来源是 `middleware.go` 的
  `withAuth` 豁免名单），`/api/openapi.json` **不在**豁免名单（配 `S3C_TOKEN` 后仍需 Bearer）。
- 新增 [`apps/server/internal/handler/openapi_auth_test.go`](apps/server/internal/handler/openapi_auth_test.go)
  （TDD：先红后绿）断言：文档级 security 存在、豁免端点必须显式 `security: []`、顶层 `tags` 非空且无孤儿 / 未声明分组。
- 重新生成 [`docs/api/openapi.json`](docs/api/openapi.json)：`security` 覆盖 68 op、显式豁免 2 op、`tags` 10 组、
  70/70 op 可判定鉴权意图；[`docs/api.md`](docs/api.md) 同步该事实。
- ⚠️ **属「需确认」改动**：按 [`docs/AI_POLICY.md`](docs/AI_POLICY.md) §3，修改公共 API 契约（`openapi_register_*.go`）
  需人类复核后再落地；本轮**未推送、未合并**。

### 修复（2026-09-29 文档失真收口：11 处文档与实现 / 自身不一致 + 1 项登记）

> 背景：10 层文档基线盘点时逐条回读源码复核，发现一批**读者会被直接误导**的失真。
> 以下全部为「文档与实现不符」或「文档自相矛盾」，逐条给出核实依据；重建历史类问题不擅改，只登记。

- **`.github/dependabot.yml` 三个目录不存在**（写成 `/server`、`/web`、`/desktop/src-tauri`，实际在 `apps/` 下）
  → Dependabot 找不到清单，依赖更新**静默失效**（安全补丁不会自动开 PR）。已改为 `apps/...` 并加注释防回归。
- **`.github/SUPPORT.md` 自相矛盾**：称「本仓库当前没有独立的用户手册 / FAQ 文档」，而 `docs/user-guide.md`
  （654 行，含 FAQ 与排障）早已存在且被 README / `llms.txt` / `DEVELOPMENT.md` 引用 → 改为指向用户手册。
- **`.github/SECURITY.md` 的版本号会随发版漂移**：硬编码「当前版本 `v1.0.0`」，而 `scripts/release-version.sh`
  的同步目标里**没有它**（发版后必然失真）→ 改为指向 `Makefile` 的 `VERSION` 与 `GET /api/health` 的 `version`
  字段（SSOT），支持窗口表改为**不 pin 具体版本号**；`CONTRIBUTING.md` 发版清单与脚本注释同步写明「刻意不同步」。
- **`docs/architecture.md` 四处**：① 引用不存在的 `store/atomic.go`（实为 `internal/atomicfile/atomicfile.go`，
  调用点 `store/filestore.go:197`、`service/job_persist.go:128`）；② 预签名「钳制 [1h, 24h]」与实现不符
  （`handler/objects.go` 仅 ≤0 默认 1h、>24h 钳 24h，**无 1h 下限**，`threat-model.md` 亦自述「无 1h 下限」）；
  ③ `endpoints.ts`「~70 个方法」实为 **59** 个（70 是后端 `routes.go` 的 `mux.HandleFunc` 数，量纲不同）；
  ④ 配置矩阵指针仍指 README，而 README 已声明 `CONFIGURATION.md` 为 SSOT。
- **`docs/api/accounts.schema.json`**：描述「11 个字段恒存在」，`required` 实为 **12** 个。
- **`docs/errors.md`**：「`apps/web/src/errors.ts`（若存在）」——该文件确实存在，去掉陈旧对冲。
- **`README.md` 归档清单漏 2 项**：`code-review-2026-09-24` 与 `code-review-summary` 已归档，README 只列 3 项 → 补齐。
- **`docs/threat-model.md`**：页首「最后更新 2026-09-28」，而同文 §7 已记录 2026-09-29 引入 SAST → 更新为 09-29。
- **登记未修**：CHANGELOG 与 git tag 断裂（3 个时间戳 tag 无版本段、`[1.0.0]` 段日期与 tag 不一致）属
  **重建历史发布记录**，需人类确认口径 → 登记 [`KNOWN_ISSUES.md`](docs/KNOWN_ISSUES.md) **#69**，未擅改。

### 新增（2026-09-29 运维可靠性收口：新增事故复盘模板 `docs/POSTMORTEM_TEMPLATE.md`）

> 背景：10 层文档基线盘点（`docs/FEATURES.md` §AN）时，第 6 层「运维可靠」唯一未收口的缺口是**事故复盘模板**——
> `docs/OPERATIONS.md` §9.3 只定义了「如何记录」的六步，没有可复制的记录骨架；全仓检索 `复盘` / `postmortem` 命中 **0**。

- 新增 [`docs/POSTMORTEM_TEMPLATE.md`](docs/POSTMORTEM_TEMPLATE.md)：11 节空模板——元信息（P0–P3 定级对齐 §9.1）、
  影响面（含 `interrupted` 半成品）、带 `X-Request-ID` 的时间线、**取证清单**（逐条对齐 §9.3 与 §3.3 的检索命令）、
  根因（触发因素 vs 根因）、检测与响应评估（对照 §4.1 / §7.1 的**建议值**并显式标注未强制）、
  恢复验证（逐条对齐 §6.4 六步）、行动项（强制 owner + 截止 + 跟踪位置）、文档同步表（§4 门禁）、
  以及「本模板刻意不定义的事」。
- **边界**：安全漏洞报告仍走 [`SECURITY.md`](.github/SECURITY.md) 私有渠道，不进公开复盘；值班轮换 / SLA /
  对外披露由部署方自定——沿用 §9.2 / §9.4 既有口径，**不为模板发明**仓库里没有的规则（「不用于绩效追责」
  「5 个工作日」等一律标注为**建议**）。头部如实写明**尚无已填写的复盘实例**，模板不冒充台账。
- **存档不新造目录**：填写完成的复盘复用既有归档纪律（`git mv` 进 `docs/archive/`，命名
  `incident-YYYYMMDD-<短名>.md`，并在 [`archive/index.md`](docs/archive/index.md) 登记一行）。
- 文档同步：[`AGENTS.md`](AGENTS.md) 与 [`DEVELOPMENT.md`](docs/DEVELOPMENT.md) §4（命名约定 + 文档同步表两处）、
  [`OPERATIONS.md`](docs/OPERATIONS.md) §9.3 与 §11、[`README.md`](README.md) 文档段、[`llms.txt`](llms.txt)、
  [`archive/index.md`](docs/archive/index.md)、[`FEATURES.md`](docs/FEATURES.md) §AR。

### 修复（2026-09-29 前端单测假红：`App.corruptStorage` 首个用例在「覆盖率 + 全量并发」下击穿 5s 默认超时）

> 现象：`make check` 的 `web-test-cover` 稳定失败——`App.corruptStorage.test.ts` 的
> `s3c.servers = [null] 时正常渲染（不白屏）` 报 `Test timed out in 5000ms`，而
> `pnpm test`（不带覆盖率）与单独跑该文件（1.4s）都是绿的。

- **先判定「慢」还是「卡死」**（决定性实验）：同一次全量跑加 `--testTimeout=20000` →
  **74 文件 / 1132 例全绿**，该文件总耗时 **2612ms**。故是**资源竞争**而非死循环。
- **根因**：该文件的每个用例都 `vi.resetModules()` 后**重新 import 整张 App 模块图**
  （这正是它能复现「渲染期读坏存储」的原因），是全仓最重的单测文件；默认 5s 在
  istanbul 插桩 + 74 文件并行的叠加下会被**首个用例**击穿（它承担首次动态 import 成本）。
- **修法**：只给该文件放宽到 `{ timeout: 20_000 }`（4× 余量），**不动全局 `testTimeout`**
  ——其它文件不需要这么长的窗口，全局放宽会让真正卡死的用例多等 15s 才报错。
- **未掩盖任何东西**：断言一行未改，仍是「顶栏存在 + 侧栏有导航按钮 + Server 入口可达」。
  该文件此前也已被 `docs/archive/review-2026-09-19.md` §F1 记为 P0-2 验收用例。

### 新增（2026-09-29 供应链收口：桌面 SBOM + cosign 签名；告警规则落地；账号库 JSON Schema；#66 闭环）

> 逐层盘点后收口五类剩余工作——**P0 供应链**、**P1 运维**、**P1 契约**、「只登记不实施」、
> 「未闭环但可处理」。五项各配一道机械门禁且**全部做过变异验证**；证据台账见
> [`FEATURES.md`](docs/FEATURES.md) §AQ。

- **P0 · 桌面产物 SBOM**：安装包此前**没有 SBOM**（只有容器镜像有），「这个安装包装了哪些第三方
  依赖」无从查起。`release-desktop.yml` 新增 `sbom-desktop` job——把 `Cargo.lock` 与
  `pnpm-lock.yaml` 放进同一目录，用**已 pin 的 Trivy**（tag + digest）一次扫出合并 CycloneDX，
  产出 `sbom-desktop.cdx.json`，做**组件数自检**后 `attest-build-provenance` 并上传 Release。
  锁文件与平台无关，故**单 job** 即可、不必三平台各跑。**本地 docker 实测**：Cargo.lock →
  **429** 个 crate、pnpm-lock → **24** 个运行期包（Trivy 的 pnpm 解析器只取运行期图、自动排除
  devDependencies——正是「随包分发」的口径）。门禁 `TestReleaseWorkflowProducesDesktopSBOM`。
- **P0 · cosign 签名**：此前签名核验被绑在 `gh` / `docker buildx` 上，策略引擎（Kyverno / Ratify）
  与镜像准入无法验。镜像在 `publish` job 内按 **digest** 无密钥签名（keyless：OIDC 短期令牌 +
  Rekor 透明日志，**无长期私钥**）；桌面侧对 `SHA256SUMS.txt` 做 `cosign sign-blob`——**签清单
  而非逐个安装包**，一次签名传递性覆盖三平台全部产物，Release 只多 `.sig` / `.pem` 两个文件。
  安装器 SHA 经**三步核验**（`refs/tags` → 解引用 annotated tag → `GET /commits/<sha>` 200）：
  `v4.1.2` → `6f9f1778…`。门禁 `TestCosignSigningIsWiredAndConsistentlyPinned`。
- **P1 · 运维**：OPERATIONS.md §4 有 SLO 与告警**表格**，但自述「仓库未提供告警规则文件」——
  落地要靠运维手抄表达式，而**抄错一个字母 Prometheus 不报错，告警只是永不触发**。新增
  [`deploy/prometheus/s3client.rules.yml`](deploy/prometheus/s3client.rules.yml)：3 条 recording rule
  （把 §4.1 的 SLI 固化，告警与仪表盘共用同一表达式）+ 9 条 alerting rule（与 §4.2 逐行对应）。
  门禁 `TestPrometheusRulesReferenceRealMetrics` 校验「引用的指标真实存在于发射点」且
  「`code` 取值在 `s3wrap` 白名单内」（白名单外的码会被折叠成 `other`，表达式即永不命中）。
- **P1 · 契约**：账号库 `accounts.json` 的格式此前只有散文描述。新增
  [`docs/api/accounts.schema.json`](docs/api/accounts.schema.json)（JSON Schema 2020-12）：
  `Array<Account>`、12 个字段**全 required**（`model.Account` 无 `omitempty`，写出的文件里字段
  恒存在）、`id` 标 `uuid`、时间标 `date-time`；并写明「加密后是 S3C3 二进制信封，本 schema
  不适用」。门禁 `TestAccountStoreSchemaMatchesModel` 以**反射**取真值，断言字段集**双向相等** +
  类型 + required。
- **闭环 KNOWN_ISSUES #66（GitLab 侧缺 SAST），依据是「真跑通」而不是「能列出来」**：
  该条登记的理由是「另一套工具链 + 无法在 `make gcl` 本地验证」——**实测推翻**。
  - **接入**：`.gitlab-ci.yml` 加 `include: template: Jobs/SAST.gitlab-ci.yml`（GitLab 原生
    SAST，**Free 档即可用**；Advanced SAST / MR 内联标注才是 Ultimate），并因其 `stage: test`
    的 job 要求、把 `test` 补进自定义的 `stages`。Go 与 TS/JS 都由 semgrep 分析器覆盖，
    故只新增 `semgrep-sast` 一个 job（其余分析器按各自 `exists:` 规则不匹配本仓库）。
  - **实跑证据**（不是 `--list`）：`make gcl GCL_JOBS=semgrep-sast` → `finished in 23 s` →
    `exported artifacts` → **`PASS` / `EXIT=0`**，产出 `gl-sast-report.json`。
    ⚠️ 更正本条目此前的一个错误结论：**「实现早已落地、只是台账没关」不成立**——
    `include:` 是本轮（2026-09-29 18:02）才加的，本轮开始时该文件既无 `include:` 也无
    `test` stage；`make gcl-list` 只能证明「模板能解析、job 排得进流水线」，
    它**不**证明镜像拉得下来、job 跑得起来、报告产得出来——这三件事只有真跑才知道。
  - **实跑立刻暴露出配置缺陷**：默认 `$DEFAULT_SAST_EXCLUDED_PATHS`（`spec, test, tests, tmp`）
    是**目录名**式的，与本仓库命名约定不匹配，首跑 **66 项发现**里有 50 项在测试代码
    （Go 的 `*_test.go`、Vitest 的 `*.test.ts`）、6 项在**生成的** Istanbul HTML 报告资产
    （`apps/web/coverage/` 里的第三方 `prettify.js` / `sorter.js`）。已按覆盖率 instrumentation
    的同一口径补齐 `SAST_EXCLUDED_PATHS`，**66 → 10 项**（4 High + 6 Medium），全部落在生产源码。
  - **triage 落盘**：10 项逐条判定为**误报 / 设计使然**（如 3 处 `int32(n)` 紧邻上一行就是
    `n > 0 && n <= 1000` 守卫；SSRF 三项是「用户配置自己的后端地址」，即产品功能本身），
    依据记入 [`threat-model.md`](docs/threat-model.md) **§7**。**不做规则级抑制**：`.gitlab/sast-ruleset.toml`
    与 `SAST_RULESET_GIT_REFERENCE` 属 **Ultimate 专属**，本项目按 Free 档设计，故没有 Trivy
    `.trivyignore` 那样可移植的集中 ignore；`// nosemgrep` 全档可用但会把工具专有注释写进生产代码。
  - **顺带修掉一个扫描盲区**：`apps/web/src/deadcode_gate.test.ts` 里的 `/<!--…/ ` 正则字面量
    被 Semgrep 判成语法错误（JS Annex B 把 `<!--` 当行注释起始）并**整份跳过该文件**；
    改为 `new RegExp('<!--[\\s\\S]*?-->', 'g')`，行为等价且回到扫描范围。
  - **产物不入库**：`gl-sast-report.json` 补进 `.gitignore`（gcl 会把 artifacts 拷回工作目录根）。
  - **文档同步**：[`DEVELOPMENT.md`](docs/DEVELOPMENT.md) §3 CI 对照表与触发事件表（补 `semgrep-sast`
    行、并在门禁落点登记新门禁）；[`KNOWN_ISSUES.md`](docs/KNOWN_ISSUES.md) #66 闭环并移除；
    [`threat-model.md`](docs/threat-model.md) §7 新增。
- **顺带修掉一个「文档里的命令其实跑不通」的既有缺陷（本轮实测发现）**：Makefile 的
  `gcl` / `gcl-docker` 直接把 `$(GCL_JOBS)` / `$(GCL_EXTRA)` 传给 gitlab-ci-local，但
  gitlab-ci-local 用 yargs 的 `.env('GCL')` 做**前缀映射**——任何 `GCL_*` 环境变量都会被当成
  同名 CLI 选项（`GCL_JOBS=web` → `--jobs web` → `Unknown argument: jobs`；`GCL_FOO=bar` →
  `Unknown argument: foo`，实测确认是前缀映射而非个别键）。而 **GNU make 会把命令行传入的
  变量导出进 recipe 环境**，于是 `make gcl GCL_JOBS=web` **必然失败**——而 Makefile / `README.md` /
  `docs/DEVELOPMENT.md` / `.gitlab-ci.yml` 曾共 **6 处**宣传这条「跑单个 job」的唯一姿势。
  已在 recipe 里用 `env -u GCL_JOBS -u GCL_EXTRA -u GCL` 把它们摘出环境（make 展开仍照常作为
  位置参数，用户可见行为不变），并加门禁 `TestMakefileGclStripsEnvPrefixVars` 防回退
  （变异验证：去掉 `env -u` → 红灯点名三个变量）。
  **本轮的 SAST 实跑正是靠这个修复才跑得起来**——这条缺陷是 #66 验证路上的直接阻塞。
- **门禁自我修正（与「不留尾巴」直接相关）**：桌面 SBOM 门禁**首版是弱门禁**——只断言
  `gh release upload` 字符串存在，而该 workflow 里另有两处同名字符串，于是**把 SBOM 上传整段
  删掉仍全绿**（变异验证时实测暴露）。已收紧为断言**带 SBOM 文件名的那条完整命令**并复验。
  不做变异，这条门禁会一直以「已覆盖」的姿态存在。
- **文档同步**：[`ROADMAP.md`](docs/ROADMAP.md) §三 #12 由 ⏳ 改 **✅**；
  [`OPERATIONS.md`](docs/OPERATIONS.md) §4 指向规则文件；[`DEVELOPMENT.md`](docs/DEVELOPMENT.md)
  §3（门禁落点 +5、CI 对照表、触发事件表、删掉作废的「未镜像理由」段）；
  [`KNOWN_ISSUES.md`](docs/KNOWN_ISSUES.md) #66 闭环并移除。

### 修复（2026-09-29 KNOWN_ISSUES #67 / #68 闭环：前端可访问性三处 + nginx 跨层日志关联）

> 两条都是上一批写文档时**实测发现**的真实缺陷（不是文档问题），当时「只记录未修」。
> 本轮一并闭环，**四项改动各配一道回归门禁**——没有一条靠「改完看一眼」结束。
> 证据台账见 [`FEATURES.md`](docs/FEATURES.md) §AP。

- **#67① `<html lang>` 不随界面语言更新**：`index.html` 硬编码 `lang="zh-CN"` 且全仓无任何运行时
  赋值，切到英文后屏幕阅读器仍按中文发音、浏览器翻译提示也会误判。`i18n/index.ts` 新增
  `applyDocumentLang(loc)`——**模块初始化时同步一次**（让持久化的语言在渲染前就生效），
  `setLocale()` 内再同步。`i18n/index.test.ts` 补 **2 条行为用例**（`setLocale` 与 `cycleLocale`
  两条路径都断言 `document.documentElement.lang`），先红后绿实测。
- **#67② 未处理 `prefers-reduced-motion`**：此前全仓 0 命中，`styles.css` 的 `rise` / `toast-in` /
  `skel` 关键帧与 `modal-fade` / `pop-in` 过渡在「减少动效」偏好下**照常播放**。现于 `styles.css`
  **末尾**加 `@media (prefers-reduced-motion: reduce)`，对 `*` / `*::before` / `*::after` 关闭
  `animation` 与 `transition`（`!important` 必要：动效分散在组件级选择器与内联过渡上，逐条提优先级
  既易漏又难维护）。
- **#67③ `textarea` 不在统一焦点样式内**：`:focus-visible` 选择器列表缺 `textarea`，而表单基础规则
  写了 `outline: none` ⇒ `BucketPolicyVisualEditor.vue` 的 3 个 `<textarea>` 拿不到与其它控件一致的
  2px 轮廓。已纳入选择器列表。
- **新增门禁 `apps/web/src/a11y_gate.test.ts`（3 例）**：CSS 是纯样式表，vitest 跑在 happy-dom 上
  **不做级联**，故这两条此前没有任何测试能观测。门禁用**源码形态**断言——媒体查询存在**且确实关闭
  animation / transition**、`:focus-visible` 规则块选择器含 `textarea` 且声明含 `outline: 2px`；
  并带一条「门禁前置成立」防空跑断言。
- **#68 nginx 访问日志缺请求 ID**：`log_format main` 不含 ID ⇒「用户报错 → 查 nginx 日志 → 定位后端
  `req`」在反向代理层断开。两份被挂载的配置（`nginx.conf` / `nginx.docker.conf`）补**两个**字段：
  `rid=$http_x_request_id`（客户端请求值）与 `req=$upstream_http_x_request_id`（**后端在响应上回显的
  值**，与后端访问日志的 `req` 同源——这才是用来对齐的）。**只记后者**会在客户端传了 ID 时丢掉用户
  手里的原始值；**只记前者**会在客户端没传时记成空，而那正是最常见的情形（浏览器不会自己加这个头）。
  由 `repo_infra_gate_test.go` 的 `TestNginxAccessLogCarriesRequestID` 守住（变异验证：删字段 → 红灯
  点名两份配置）。
- **写测试时实测到的两处工具行为（记录以免重蹈）**：① **`?raw` 读不到 CSS**——vitest 默认
  `css: false` 把 CSS 模块替换为空模块，`import.meta.glob('./*.css', { query: '?raw' })` 能匹配到
  `./styles.css` 这个**键**但**取值长度为 0**，静态导入同样得到空串，故改用 `node:fs`（打开
  `css: true` 会改变**全部**用例的 CSS 处理方式，代价不成比例）；② **vitest 下 `import.meta.url`
  是 `http://` 形式**，`fileURLToPath` 抛「URL must be of scheme file」。两处都先被「门禁前置成立」
  的防空跑断言拦住，没有静默变绿。
- **连带清掉一处既有 workaround**：`apps/web/e2e/screenshots.spec.ts` 的 `waitForTimeout(500)`
  硬等——它当时的注释写着「本应用**未处理** `prefers-reduced-motion`，故 `emulateMedia` 压不掉它」，
  而它本身就是 flaky 来源（机器慢就截到半透明、文字重叠）。现改为
  `test.use({ reducedMotion: 'reduce' })` 并删除该等待：截图拿到稳定终态，用例耗时降到 **881 ms**
  （实测 `2 passed / 3.0s`）。这同时让该 spec 在**真实浏览器**里验证了媒体查询确实生效
  （vitest 的 happy-dom 不做 CSS 级联，`a11y_gate.test.ts` 只能验源码形态，两者互补）。
- **顺带更正一处 HEAD 就存在的现状失真**：`docs/ROADMAP.md` §四 的「E2E（mock 版）」行，
  其**当前状态**列填的是「全 action SHA 经 GitHub API 核验（5 个 SHA 实测 200）」——那是 action
  pin 校验的结论（`TestWorkflowActionsAreShaPinned` 的职责），**与 E2E 通过数无关**。该行现改为
  实跑的 **17 passed / 0 skipped**（不是 15：`screenshots.spec.ts` 的 2 例从未同步进任何通过数）；
  `docs/FEATURES.md` §三 的同一数字一并修正。
- **文档同步**：[`accessibility.md`](docs/accessibility.md)（§2.4 改写、§4 第 5/6/12 条标注已修复
  并保留原编号、§5.3 / §5.4 勾除已完成项）、[`OPERATIONS.md`](docs/OPERATIONS.md) §3 日志关联段重写、
  [`KNOWN_ISSUES.md`](docs/KNOWN_ISSUES.md) #67 / #68 闭环（§二 移除 + §四 台账 + 摘要段）、
  [`ROADMAP.md`](docs/ROADMAP.md) §四 门禁基线复测值。
- **门禁**（全绿实测）：`pnpm lint` **0 告警** / `pnpm typecheck` + `typecheck:e2e` exit 0 /
  `pnpm test` **74 文件 1132 例**（1127 → 1132，新增 5 例）/ `pnpm test:coverage`
  **四指标 100%**（宿主 `NODE_ENV` 未设 4296 / 2932 / 1130 / 3679；`NODE_ENV=production`
  4294 / 2932 / 1130 / 3677）/ `pnpm build` OK——**366.72 kB，gzip 112.72 kB**（JS 较 §AO 的
  366.66 kB 增 0.06 kB）、CSS 31.84 kB（增 0.16 kB，来自 `prefers-reduced-motion` 块与注释，
  属预期）；`gofmt -l` 干净 / `go vet` 0 告警 / 后端 `go test` **9/9 包通过** /
  `golangci-lint` **0 issues**。

### 修复（2026-09-29 KNOWN_ISSUES #65 闭环：前端死代码门禁改用 **TS AST** 判定引用，注释 / 字符串不再算引用）

> #65 是上一批文档收口时**实测发现**的门禁漏报口径：`apps/web/src/deadcode_gate.test.ts` 的
> `usageBody()` 用正则剥掉 import 与导出声明后按**裸词**计数，于是「导出零调用，但别处有一句
> 提到它的**注释**」即被判为已引用。真实事故：`useKeydownStack.ts` 的 `isTopKeydown` 唯一调用点
> 早已删除，仅因 `ConfirmDialog.vue` 留了一句说明注释，门禁 8 例全绿——**漏报的正是本门禁要拦的东西**。
> #65 当时写明「未在本轮修」的理由是「TS 无等价 AST 依赖」；本轮核实该前提**不成立**
> （`typescript@5.9.3` 已是 `apps/web` 的**直接 devDependency**），故闭环。

- **先红后绿**：先把计数逻辑抽成纯函数 `findUnreferencedRuntimeExports(sources)`，补一条**合成源码**
  口径用例（名字只出现在注释 / 字符串 / 模板字面量文本里 → 必须判死；真实调用与 `${…}` 插值 →
  不得误杀）。未修前该用例实测 **FAIL**（`expected [] to deeply equal [ …(3) ]`）——三种「只被提及、
  未被引用」的情形当时全被判成「有引用」。与后端 `deadcode_gate_test.go`「用合成源码写口径测试」
  同做法，**不依赖仓库里正好有反例**。
- **一次失败的尝试（已废弃，记录以免重蹈）**：先用 `ts.createScanner` 做词法剥离（按 token 类型
  把注释 / 字符串替换为空格），直觉上最稳。**实测被证伪**——`i18n/index.ts` 的
  `` s.replaceAll(`{${k}}`, String(v)) `` 之后，扫描器把**其后 393 字节（到文件尾）**当成一个模板
  token 吞掉（裸 `scan()` 循环不会在模板替换后 `reScanTemplateToken`），**一次误报 18 个真实导出
  为死代码**（`setLocale` / `useHealthPoll` / `parsePolicy` …）。这正是 #65 警告过的「误伤」。
- **最终做法（AST + 按文件类型分派）**：`.ts` 用 `createSourceFile` 遍历 AST 收集标识符；
  `.vue` **只**把 `<script>` 块交给 AST，`<template>` 原样保留（模板里的组件名 / 表达式是真实引用）、
  `<style>` 丢弃、HTML 注释剥掉。排除三类非引用：`import` 语句、`export { X }` / `export * from`、
  **被导出声明自身的名字**（否则每个导出自证被引用）；保留类型位置 / 属性名 / 对象键 / `${…}` 插值，
  沿用旧口径以免误杀。
- **变异验证**：往 `theme.ts` 注入「只被注释提到」的导出 `mutationProbeDead` → 门禁**红灯并点名**
  `./theme.ts → mutationProbeDead`；删除后绿灯。证明新口径确实拦得住旧口径漏掉的那一类。
- **残留不隐藏**：文件头「盲区 3」改为已收口，并注明 `.vue` 的 `<template>` 仍按裸词计数
  （Vue 模板不是 TS 语法，精确判定需引入 Vue 编译器）、**同名标识符跨文件抵消仍然存在**
  （与后端 Gate 1 同向，只会漏报）——这些都是裸词口径的结构性特征，不是本次引入。
- **门禁**（全绿实测）：`pnpm lint` 0 告警 / `pnpm typecheck` + `typecheck:e2e` exit 0 /
  `pnpm test` **73 文件 1127 例**（1126 → 1127，新增 1 条口径用例）/ `pnpm test:coverage`
  **四指标 100%**（宿主 `NODE_ENV` 未设 4293 / 2932 / 1129 / 3676；`NODE_ENV=production`
  4291 / 2932 / 1129 / 3674）/ `pnpm build` OK（**366.66 kB，gzip 112.71 kB**）；本轮未改后端。
- **文档同步**：[`KNOWN_ISSUES.md`](docs/KNOWN_ISSUES.md) #65 闭环（§二 移除 + §四 台账改「已闭环
  移除」+ 摘要段改写）、[`ROADMAP.md`](docs/ROADMAP.md) §四 前端例数与覆盖率复测值、
  [`FEATURES.md`](docs/FEATURES.md) 新增 **§AO** 与 §三 复测段。

### 新增（2026-09-29 文档覆盖矩阵收口：11 个新文档 + 3 项供应链门禁 + 机器可读契约 + 性能基线）

> **起因**：以「AI 时代成熟仓库文档基线」10 层清单（社区健康 / 元信息 / 开发协作 / 架构决策 /
> 接口契约 / 运维可靠 / 安全供应链 / 用户文档 / AI 时代新增层）逐层盘点本仓库，得出诊断——
> **对贡献者与 agent 已很完善，但对运维者与终端用户几乎没有专门文档，供应链证据也缺**。
> 本条目是逐项补齐的台账。**逐条证据与取舍见 [`docs/FEATURES.md`](docs/FEATURES.md) §AN。**

**P0 · 供应链可验证性**（对应 OpenSSF Scorecard #14 SAST / #15 SBOM / #17 Signed-Releases）

- **新增 [`.github/workflows/codeql.yml`](.github/workflows/codeql.yml)**：CodeQL SAST，矩阵
  `go` + `javascript-typescript`，查询集 `security-and-quality`。Go 侧此前只有规则/漏洞库驱动的
  golangci-lint(gosec)+govulncheck，**JS/TS 侧此前完全没有 SAST**——而前端直接处理预签名 URL、
  Token 与对象 key，是真实攻击面。单独成 workflow（而非并入 `ci.yml`）是为了不把
  `security-events: write` 带进主 CI 的最小权限面（Scorecard #18）。
- **镜像供应链证据**（`ci.yml` · `publish`）：BuildKit `provenance: mode=max` + `sbom: true`
  产出 OCI attestation manifest；另用**已 pin 的 Trivy**（不引入第二套 SBOM 工具）产出可下载的
  CycloneDX SBOM 文件并作为 artifact；再用 `actions/attest-build-provenance` 出具 GitHub 产物证明
  （`gh attestation verify` 可核验）。按 **digest** 而非 tag 扫描，保证扫的就是刚推的那份。
- **桌面产物证明**（`release-desktop.yml`）：三平台安装包出具产物证明。**与代码签名不互相替代**——
  它证明「产物 ← 哪个 commit ← 哪个 workflow」，但不解决 SmartScreen / Gatekeeper 的**发布者信誉**
  问题（后者仍是 roadmap E6 的外部凭证阻塞）。
- **新增门禁 `TestWorkflowActionsAreShaPinned`**：本仓库「Actions 全部 pin SHA」此前**只有文字约定**，
  没有任何机械门禁——新增一个 `uses: foo@v1` 可静默通过全部测试，而 tag 可被上游重新指向
  （供应链投毒入口）。现把该纪律变成红灯（当前 6 个 workflow / 46 个引用全绿）。

**P1 · 可运维性与机器可读契约**

- **新增 [`docs/OPERATIONS.md`](docs/OPERATIONS.md)**（元文档）：可观测性（16 个指标逐个列明）、
  SLO 与告警基线（**显式标注为建议值、未在代码/CI 中强制**）、Runbook R-1..R-10、
  备份与恢复、灾难恢复、升级回滚、事故响应、容量与单实例约束。
  单列了「本版本没有的观测面」（账号写入失败、在册任务数、HTTP 延迟、卷容量、关停耗时），
  避免读者误以为有。
- **提交机器可读契约 [`docs/api/openapi.json`](docs/api/openapi.json)**（4 579 行）：此前 API 契约
  SSOT 只在**运行时** `GET /api/openapi.json`（默认 404），意味着 Swagger UI / 客户端代码生成 /
  契约 diff / AI 编码代理都必须**先把服务跑起来**才能读到契约。现落盘为仓库内文件。
  采用 JSON 而非 YAML：`internal/openapi` 明确「不引入额外依赖」，且 JSON 是 YAML 1.2 子集。
  新增 **golden 门禁** `TestCommittedOpenAPISpecMatchesRuntime`（改 handler 后必须
  `go test ./internal/handler/ -run TestCommittedOpenAPISpecMatchesRuntime -update-openapi-spec`
  重新生成，否则红灯；已做变异验证）+ `TestCommittedOpenAPISpecIsDiscoverable`（位置 / 可解析 /
  `info.version` 与 `main.go` 一致）。`release-version.sh` 同步该文件（同步清单 17 → 20 个文件）。

**P2 · 用户文档与治理**

- **新增 [`docs/user-guide.md`](docs/user-guide.md)**（654 行）：面向**使用者**的操作指南。
  此前 README「功能」是密集 bullet、`FEATURES.md` 是**台账**（记录「做了什么」），
  **没有任何一页回答「怎么用」**。快捷键 16 行逐条抄自源码，界面文案与 i18n 字典逐条比对。
- **README 补界面截图**（`docs/images/`）：由新增的
  [`apps/web/e2e/screenshots.spec.ts`](apps/web/e2e/screenshots.spec.ts) 从**真实构建产物**生成，
  该 spec 同时是冒烟用例——先断言界面渲染成功且**不含**「无法连接后端」这类环境噪声再截图，
  因此不会把白屏或错误页存成文档配图。只桩 `/api/health` 与 `/api/accounts` 两个端点
  （复用 `features.spec.ts` 的大 fake 会重复维护），无需维护整套 in-memory 后端。
- **新增 [`.github/SUPPORT.md`](.github/SUPPORT.md) / [`.github/GOVERNANCE.md`](.github/GOVERNANCE.md)**：
  支持渠道与「本仓库不提供什么」（无 SLA / 无商业支持 / **无安全邮箱**）、治理现状
  （**如实写明单人维护**，无委员会 / 选举 / 时间表）、角色与 CODEOWNERS 关系、决策流程
  （架构级变更须先写 ADR）、发布权（**代理与自动化不得自动发版**）。
- **新增 [`docs/compatibility.md`](docs/compatibility.md)**：版本命名与节奏、支持窗口、`/api/*` 演进
  承诺（**如实写明当前没有 URL 版本前缀 / 无 Accept 协商 / 无功能开关式降级**）、
  `S3C2 → S3C3` 存储格式兼容、配置增删口径、弃用政策（**尚无任何已弃用项，且不预先承诺时间窗**）、
  桌面端未签名现状。

**P3 · 工程完善**

- **新增 Go 基准测试**（`s3wrap` / `service` / `store` 各一个 `bench_test.go`）+ 
  **[`docs/PERFORMANCE.md`](docs/PERFORMANCE.md)**。此前全仓 `func Benchmark` 为**零**。
  实测出三条值得记录的结论：① 加密写入单次约 **30 ms / 瞬时 64 MiB**（Argon2id t=2,m=64MiB,p=4
  ——**有意设计**，不该被「优化」掉，但并发写时的内存预算要算进去）；② 账号写入是 **O(n)**
  （`persistLocked` 每次重写整个文件，`-benchtime=200x` 335 µs vs `-benchtime=1s` 11.2 ms
  ——账号量级是「配置」级，换来实现简单 + 落盘原子）；③ `PresignPut` 约 **33 µs / 362 allocs**，
  发生在点击路径上无感，但**不可**放进按对象循环的路径。
- **新增 [`docs/glossary.md`](docs/glossary.md)**（64 条：S3 领域 35 + 本项目自造词 29）、
  **[`docs/i18n.md`](docs/i18n.md)**（消息模块结构 / 新增一门语言的 8 步 / 文案覆盖率门禁 /
  5 类常见错误）、**[`docs/accessibility.md`](docs/accessibility.md)**（ARIA 与键盘现状、
  **明确声明未做正式 WCAG 审计**、12 条「未做」清单）。

**顺带修复与登记**

- **删除死代码 `isTopKeydown`**（`apps/web/src/composables/useKeydownStack.ts`）：其唯一调用点早已
  移除，仅靠 `ConfirmDialog.vue` 里一句**说明注释**与测试「续命」。测试改为**派发真实 keydown 观测
  行为**（同时消除白盒断言——此前用 `isTopKeydown()` 直读内部栈，证明不了「真的有且只有栈顶收到
  事件」）。**门禁漏报口径未修**（`usageBody()` 剥掉声明后按裸词计数，注释里的同名词即算引用），
  已登记 **KNOWN_ISSUES #65**。
- **写文档时实测发现、本轮只记录未修的四处**：**#66** GitLab 侧缺 SAST（CodeQL 有意未镜像）；
  **#67** 前端可访问性三处——`<html lang>` 不随语言切换更新、未处理 `prefers-reduced-motion`、
  `styles.css` 的 `:focus-visible` 缺 `textarea`（`BucketPolicyVisualEditor.vue` 的 3 个 textarea
  聚焦无统一轮廓）；**#68** `deploy/nginx/` 的 `log_format main` 缺 `$http_x_request_id`，
  导致「用户报错 → nginx 日志 → 后端 `req`」这条链路在反代层断开。四条均写进
  [`docs/KNOWN_ISSUES.md`](docs/KNOWN_ISSUES.md) §二，**不留在对话或文档正文里当口头待办**。
- **文档命名**：新增文档一律按 [`docs/DEVELOPMENT.md`](docs/DEVELOPMENT.md) §4 的
  「元文档大写 / 内容文档小写」二分落位。

### 变更（2026-09-29 文档命名规则改为「元文档大写 / 内容文档小写」：`docs/` 下 6 个元文档恢复大写）

> **起因**：`docs/` 下 21 个文件里只有 `KNOWN_ISSUES.md` 一个大写名，而这个大写是 2026-09-24
> 作为「白名单例外」单独登记的——即**同一类文档被拆成两套规则**。本轮把它收敛成一条可判定的
> 标准：**描述「仓库自身如何运作」的元文档用大写；描述「产品是什么 / 怎么用」的内容文档用小写
> kebab-case**。这正是 2026-09-17「台账类大写」与 2026-09-24「单个例外」之后的第三代规则。

**改名（一律两步 `git mv`／`mv`，纯大小写改名在 macOS / Windows 上单步会静默失败）：**

| 旧名 | 新名 | 归类 |
|---|---|---|
| `docs/configuration.md` | `docs/CONFIGURATION.md` | 元（配置） |
| `docs/deployment.md` | `docs/DEPLOYMENT.md` | 元（部署） |
| `docs/development.md` | `docs/DEVELOPMENT.md` | 元（开发规范） |
| `docs/ai-policy.md` | `docs/AI_POLICY.md` | 元（政策） |
| `docs/features.md` | `docs/FEATURES.md` | 元（台账） |
| `docs/roadmap.md` | `docs/ROADMAP.md` | 元（规划） |

**保持小写（内容文档）**：`docs/api.md`、`docs/architecture.md`、`docs/errors.md`、
`docs/threat-model.md`，以及 `docs/archive/`、`docs/decisions/` 下全部文件
（`KNOWN_ISSUES.md` 已是大写，不变）。

- **引用同步 607 处 / 44 个文件**：含 **可执行路径**——`scripts/release-version.sh` 的 4 处 sed 目标
  （`docs/DEPLOYMENT.md` / `docs/ROADMAP.md` / `docs/FEATURES.md`，漏改会让发版脚本的版本号静默不更新）、
  `doc_number_gate_test.go` 的 `file:` 字段、`config_doc_gate_test.go` 的 SSOT 路径、
  以及 `internal/config`、`internal/handler`、`internal/s3wrap`、`internal/store`、`apps/web/src` 的注释。
- **历史条目按惯例不改写**：`[Unreleased]` 下方 **2026-09-17「5 个文档名小写化，命名规范收口」** 那条
  **整行冻结**——它记的是当时的小写化映射，改写会让历史自相矛盾；**2026-09-24「文档命名规则登记例外」**
  那条只把其中的 `docs/development.md` 实时路径更新，其括号内的旧名列表保持原样。
  **这是本轮唯一保留小写名的两处**（按标题检索，不写行号——行号会随新增条目腐烂）。
- **规则本体两处同 PR 重写**：[`AGENTS.md`](AGENTS.md) 与 [`docs/DEVELOPMENT.md`](docs/DEVELOPMENT.md)
  §4 命名约定——由「白名单 + 不得新增大写名」改为「按性质二分」，并写入**沿革**（三代规则的取舍）
  与一句防回退提示：**不要凭直觉把某一类「修正」回去**。另把原「目录」条明确为只约束**目录名**，
  避免与新文件名规则混淆。
- **归档件只修链接、不改结论**：`docs/archive/` 是冻结快照，本轮只更新其指向被改名文档的
  **链接目标**（否则归档件 404），正文结论一字未动。

### 新增（2026-09-29 补齐 AI 代理治理与配置 SSOT 文档：`docs/AI_POLICY.md` / `docs/CONFIGURATION.md` / 根 `llms.txt` / `.github/CODEOWNERS`）

> 起因：对照一份通用 AGENTS.md 模板清点本仓库文档面，发现「代理权限边界」「AI 使用披露」
> 「配置项单一事实来源」「LLM 导航索引」「默认审查者」五处缺口。同时模板要求的另一些文档
> （提示词 / 模型卡 / 数据集卡 / 评估基准 / 护栏）在本仓库**结构性不适用**——产品无 AI 功能、
> 无自训练模型、无数据集——故**有意不创建**，并在 [`docs/AI_POLICY.md`](docs/AI_POLICY.md) §0
> 写明不适用的理由与「何时才需要」，避免留下无人维护的空壳文档。

- **[`docs/AI_POLICY.md`](docs/AI_POLICY.md)（新）**：代理五档模式（默认**补丁模式**）、权限矩阵
  （允许 / 需确认 / 禁止）、MCP 工具权限口径、AI 使用披露模板、完成定义（DoD）、多代理协作、
  故障排查、升级渠道。
- **[`docs/CONFIGURATION.md`](docs/CONFIGURATION.md)（新）**：全部 **18 个** `S3C_*` 环境变量的
  **SSOT**、启动期 fail-closed 清单、客户端存储键与 token 持久化策略。清点中发现 README 旧矩阵
  **漏登 3 项**（`S3C_LOG_JSON` / `S3C_EXPOSE_OPENAPI` / `S3C_CSP_CONNECT_SRC`），已在新表补全；
  README 的长矩阵改为「最常用 5 项 + 指向 SSOT」，消除双源漂移。
- **[`llms.txt`](llms.txt)（新，仓根）**：给 LLM / 编码代理的仓库地图（硬约束 + 门禁命令 + 文档索引
  + 禁区）。llms.txt 约定把位置固定为 `/llms.txt`，故**根目录约定文件由 3 个扩为 4 个**——
  [`AGENTS.md`](AGENTS.md) 与 [`docs/DEVELOPMENT.md`](docs/DEVELOPMENT.md) §4 两处同步。
- **[`.github/CODEOWNERS`](.github/CODEOWNERS)（新）**：默认审查者 + 安全敏感面 / 供应链 / 治理文件
  分区。文件名由 GitHub 按字面固定（小写不生效），属命名白名单 ① 类，已**同 PR 登记进两处白名单**。
- **[`.github/PULL_REQUEST_TEMPLATE.md`](.github/PULL_REQUEST_TEMPLATE.md)**：补「AI 使用披露」段
  （含可直接粘贴的模板）与「风险与回滚」段；验收清单加「文档同步」一项；头部链接指向新政策。
- **[`.github/CONTRIBUTING.md`](.github/CONTRIBUTING.md)**：新增「联系与支持」段——此前
  [`SECURITY.md`](.github/SECURITY.md) 的「**邮件**：维护者邮箱（见 CONTRIBUTING.md）」与
  [`CODE_OF_CONDUCT.md`](.github/CODE_OF_CONDUCT.md) §执行的「CONTRIBUTING.md 中列出的渠道」
  都是**死引用**（该文件从未列过邮箱或任何渠道），现改为 GitHub Issue 模板 / 私有漏洞报告 /
  维护者账号等**真实存在**的渠道，并显式声明本仓库不公开安全邮箱。
- **新增门禁 [`apps/server/config_doc_gate_test.go`](apps/server/config_doc_gate_test.go)**：断言配置 SSOT
  文档收录了 `internal/config` 生产代码读取的**每一个** `S3C_*` 变量——正是本次抓到「README 旧矩阵
  漏登 3 项」的那类漂移，靠人工比对必漏。已做**变异验证**（把 `docs/CONFIGURATION.md` 里的
  `S3C_CSP_CONNECT_SRC` 改名 → 红灯并点名该变量；还原 → 绿灯）。文件头写明**断言范围与残留**：
  只查「变量名是否被收录」，不查默认值 / 取值 / 语义（自然语言，静态门禁会被文案漂移骗过）。
  起草时曾另加一条「`S3C_*` 只能由 `config` 包读取」的门禁，**实测被证伪后删除**：
  `internal/store/store.go` 的 `New` 确实直读 `S3C_STORE_KEY`，但那是 `openJSON` 在入参为空时的
  **有意回退**（`Open` 注释写明的 StoreKey 契约，KNOWN_ISSUES #64 那轮修复的产物），
  不是漏网——不该为它发明一条仓库并不存在的规则；该残留改为在门禁文件头如实记录。
- **门禁与台账同步**：[`docs/DEVELOPMENT.md`](docs/DEVELOPMENT.md) §4 文档同步表新增「代理权限边界」
  「仓库导航」「配置项」三行并把配置项的 SSOT 指向新文件、§3 门禁落点补登新门禁；
  [`README.md`](README.md) 文档列表补两行。

### 变更（2026-09-29 工具链：Node `24.21.0` → `26.10.0`，全链路精确 patch pin）

> `26.10.0` 是 2026-09-21 发布的最新 patch，经**三源核对**：nodejs.org `dist/index.json`、
> `docker run --rm node:26-alpine node --version` → `v26.10.0`、NodeSource `node:26.x` 的
> nodistro 仓库 `Packages` → `26.10.0-1nodesource1`。
>
> ⚠️ **Node 26 目前是 Current 而非 LTS**（`dist/index.json` 中 `"lts": false`；按 Node 发布节奏
> 偶数版于同年 10 月转入 LTS）。也就是说运行镜像从 Active LTS（24 Krypton）换到了 Current——
> 这是有意的取舍，`apps/server/Dockerfile` 注释已如实标注「Current；2026-10 转入 LTS」，
> 需要更保守时可回退 24.x 并把本段一并改回。既有实测证据表明 26 与当前前端工具链兼容
> （见 [`docs/FEATURES.md`](docs/FEATURES.md) §AK：`node:26.5.0-alpine` 下 72 文件 / 1126 例全绿）。

- 改动面：4 个 GitHub workflow 的 `node-version`（`ci.yml` ×2、`e2e-real.yml`、`e2e-playwright.yml`、
  `release-desktop.yml`）、`.gitlab-ci.yml` 的 `node:26.10.0-bookworm` 镜像 + NodeSource
  `setup_26.x` + `nodejs=26.10.0-1nodesource1`（两处）+ 相关注释、`apps/server/Dockerfile` 的
  `node:26.10.0-alpine`、`README.md` 与 `.github/CONTRIBUTING.md` 的环境要求、`docs/ROADMAP.md` E9。
- **门禁**：`TestNodePinnedToPatchVersion`（`apps/server/repo_infra_gate_test.go`）继续守住
  「workflow / 镜像 / NodeSource 三处都必须精确到 `X.Y.Z`」。历史台账（`FEATURES.md` §E / §T / §AK
  记录的是当时状态）按「不追溯篡改」纪律**保持原样**。

### 修复（2026-09-29 前端测试与宿主 `NODE_ENV` 解耦：修复 33 文件 / 246 例全红）

> 本机 `pnpm test` 长期报 **33 文件 / 246 例全红**，此前一度怀疑是 Node 26 与 vitest
> 不兼容——实测**否**：同一份代码在 `node:24.21.0-alpine` 与 `node:26.5.0-alpine` 容器里
> 均为 **72 文件 / 1126 例全绿**。真凶是宿主环境变量 **`NODE_ENV=production`**。
> 证据台账见 [`docs/FEATURES.md`](docs/FEATURES.md) **§AK**。

- **根因**：Vue 的 node 入口 `vue/index.js` 在运行时按 `process.env.NODE_ENV` 二选一加载
  CJS 产物（`production` → `dist/vue.cjs.prod.js`，其余 → `dist/vue.cjs.js`）。宿主带着
  `NODE_ENV=production` 时，vitest 把 Vue 解析到 **prod 构建**，而 `@vitejs/plugin-vue`
  编译 SFC 产出的绑定属于 **dev 构建** ⇒ 进程内两份 Vue 实例、reactive 状态不连通。
- **症状极具误导性**（排查成本的主要来源）：不是「找不到模块」，而是
  `[vitest] No "toasts" export is defined on the "./store" mock` —— 而 `src/store.ts`
  的 `toasts` 导出**确实存在**。连带 `Cannot call text on an empty DOMWrapper` /
  `w.vm.xxx is not a function` 等 **246 例**次生失败，根因却只有 2 条。
- **修法**：在 `apps/web/vite.config.ts` 顶部、`import` 之前执行
  `if (process.env.VITEST) { process.env.NODE_ENV = 'test' }`。
  位置必须在 import 之前——`test.env` / `resolve.alias` / `test.define` /
  `server.deps.inline` 四种写法均已实测无效。
  **条件 `process.env.VITEST` 不可省**：无条件改写会让 `vite build` 也读到非 production
  值，把 Vue 的 dev/warn 分支打进产物（实测 bundle 366.66 kB → **424.72 kB**，+58 kB）。
- **守卫**：新增 `src/vite_env_guard.test.ts`（2 例）钉住不变量——删掉那行立刻变红，
  报错文案直接给出根因与修法位置（红灯已实测）。
- **门禁**（全绿实测）：`pnpm test` **73 文件 / 1128 例**（宿主带 `NODE_ENV=production`
  与未设两种环境均全绿）/ `pnpm test:coverage` 四指标 **100%** / `pnpm lint` 0 告警 /
  `pnpm typecheck` + `typecheck:e2e` exit 0 / `pnpm build` OK 且产物
  **366.66 kB（gzip 112.71 kB）**、hash 与改动前一致（无体积回归）。

### 修复（2026-09-28 KNOWN_ISSUES #64 闭环：`store.Open` 的 json 分支不再丢弃入参 `storeKey`）

> #64 是三路五轴复审 19 条中**唯一未闭环**的一条，此前登记为「Nit，实测后挂起」。
> 本次按先红后绿闭环，清单开放项归零。证据台账见 [`docs/FEATURES.md`](docs/FEATURES.md) **§AJ**。

- **缺陷**：`store.Open(dataDir, driver, storeKey)` 的 json 分支无条件
  `return New(path)`，而 `New` 回头读环境变量 `S3C_STORE_KEY` ⇒ **入参被丢弃**，
  与 sqlite / encrypted 两个分支「显式透传 `storeKey`」的契约不一致：非 `FromEnv`
  的调用方传了 key 仍会明文落盘且无任何报错。今天唯一的生产调用方
  （`main.go:85`）传的 `cfg.StoreKey` 恰恰来自同一个环境变量，故**零生产影响**。
- **修法**：抽 `openJSON(path, storeKey)` —— **入参非空即用入参**（`newStore(path, storeKey, false)`），
  **入参为空才回退** `New`（保留环境变量这条既有用法的调用路径）。
  **不**改 `New` 签名：那要动 ~100 处调用、24 个测试文件，而
  `crossdriver_test.go` / `encrypt_at_rest_test.go` 刻意依赖环境变量读 key，机械替换会静默改语义。
- **红灯 → 绿灯**（`internal/store/open_storekey_test.go` + `openjson_env_test.go`）：
  - `TestOpenJSONStoreKeyBeatsEmptyEnv`：环境变量清空 + 显式入参 ⇒ 仍须加密
    （修前失败于「入参被丢弃，secretKey 明文落盘且无报错」）；
  - `TestOpenJSONFallsBackToEnvWhenKeyEmpty` / `TestOpenJSONEnvFallbackKeyRecovers`：
    反向钉住「入参为空仍回退环境变量」，防止把「入参优先」误读成「只用入参」；
  - `TestOpenJSONNoKeyAnywhereStaysPlaintext`：两处都为空时仍明文（permissive 向后兼容不变）。
- **门禁**（全绿实测）：`gofmt -l` 干净 / `go vet ./...` 0 告警 / `go build ./...` OK /
  `golangci-lint run ./...` **0 issues** / `go test -count=1 ./...` **9/9 包** /
  `internal/store` **100.0% statements、零 `count==0` 块** /
  `TestNoUnusedExportedProdSymbols` 绿（`New` 仍有生产引用，不被判死代码）。

### 变更（2026-09-28 文档归档：`code-review-2026-09-24.md` 与 `code-review-summary.md` 冻结入 `docs/archive/`）

- 按 [`docs/archive/index.md`](docs/archive/index.md) 的 4 步规程 `git mv` 归档两份**时点性审查文档**
  （保留重命名历史，不复制、不删了重加）：
  - [`docs/code-review-2026-09-24.md`](docs/archive/code-review-2026-09-24.md) → `docs/archive/`：
    2026-09-24 全仓代码审查，2 Critical + 20 Required 全清、Nit 31/35 闭环；转登记的 #61/#62
    已于 2026-09-28 闭环，**再无未了结的发现**，只余 #63（已决策 ➖）。
  - [`docs/code-review-summary.md`](docs/archive/code-review-summary.md) → `docs/archive/`：
    审查总结与改进建议，正文数字全是**审查时点值**（「1042 用例 / 66 文件」），且其头部本就标注
    「**归档前收口**」——本次把这一步做完。
- **归档前先做成干净状态**：其「⚠️ 待解决的技术问题」里已闭环的第 2 条（**CORS 配置**）与第 3 条
  （**单文件超限**）**直接移除**——那是在办事项清单、不是时点快照，留着已解决项只会误导；
  现只剩第 1 条 #25（外部凭证）。移除理由写在清单下方引注里。
- **链接修正（不悬空）**：移动后两份文件的出链按归档约定加 `../`（`docs/*.md`）/ `../../`（根目录）
  前缀、`archive/` 前缀就地化；全仓 **10 个文件**的入链与源码注释同步改指 `docs/archive/…`
  （含 **8 个 Go 测试文件**注释里的 `docs/code-review-2026-09-24.md`）。
  **自检：26 个 md 悬空链接 0 处、源码 `docs/*.md` 路径 0 处悬空、旧路径 0 处残留。**
- [`docs/archive/index.md`](docs/archive/index.md) 归档清单 **2 → 4 份**（含各自的原路径 /
  归档日期 / 冻结的结论）。
- **顺带同步两处「当前态」门禁基线的过期计数**（此前停在 #60 拆分当时的时点值，而这是**当前事实**
  表、不是历史快照）：[`docs/ROADMAP.md`](docs/ROADMAP.md) §四 前端测试 **1110 → 1126 例**、
  前端覆盖率 **4255/2908/1124/3653 → 4294/2934/1130/3677**；[`docs/FEATURES.md`](docs/FEATURES.md) §三
  同两行同步，并注明 1110 → 1126 系 §AF / §AI 修复新增 16 条红灯用例。
- **`docs/KNOWN_ISSUES.md` #64 收敛**：从「19 条复审处置清单」缩为它**唯一未闭环的那 1 条**
  （`store.Open` json 分支丢 `storeKey`，Nit 实测后挂起）——已闭环的 18 条属
  [`docs/FEATURES.md`](docs/FEATURES.md) §AD–§AI 的台账，不该继续占着「问题唯一来源」的位置。

### 修复（2026-09-28 前端另四条：版本列举代次守卫 / 追加重置滚动 / `loadingAll` 提前可点 / 桶列举共用标志——#64 4 条先红后绿）

> `KNOWN_ISSUES #64`「待处置」表前端第二行的 4 条，**逐条亲自读码复核**后修复。至此 #64 的 19 条
> **只剩 1 条挂起的 Nit**。证据台账见 [`docs/FEATURES.md`](docs/FEATURES.md) **§AI**。

- **`VersionsDialog.load()` 无代次守卫**：关闭再开另一个对象时上一个对象的列举仍在飞，迟到响应会把
  `rows` 覆盖成旧对象的版本（标题已是新对象、列表却是旧的），`finally` 还会提前清掉新对象的 `loading`、
  `catch` 会把旧对象的错误报到新对象头上。
  **修法**：加 `loadSeq`，每个 `await` 后判代次、过期静默丢弃；`catch` / `finally` 都按
  `seq === loadSeq` 收口（与 `RecycleBinPanel.loadSeq` 同口径）。
  **红灯**：`过期的版本列举不得覆盖新对象的列表: expected … not to contain 'stale'`。
- **`ObjectList` 只 `watch(() => props.entries)` 就归零窗口**：`entries` 是**过滤+排序后的 computed，
  每次重算都是新数组身份**，「加载更多」的追加同样换身份 ⇒ 滚到第 300 行点「更多」视口立刻跳回第 1 行。
  （`RecycleBinPanel` 没这问题，因为它的 `markers` 是 `push` 追加、数组身份稳定。）
  **修法**：`useObjectBrowser` 暴露**换源代次 `listGen`**（`load(reset=true)` 才递增，
  `loadMore` / `loadAll` 续页不递增），`ObjectsPanel` 透传给 `ObjectList`；归零只在
  **① `listGen` 变 ② 过滤 / 排序变 ③ 条目数变少（兜底防空白表 §F2）** 三处触发。
  **红灯**：`追加不得把用户滚到的位置清零: expected +0 to be 1260`；同批补「换源即使条目变多也归零」
  的回归守卫，既有两条（换源归零 / 缩短归零）保持绿。
- **`useObjectBrowser.load()` 的 `if (reset) loadingAll.value = false`**：`loadAll` 的首轮正是以
  `reset=true` 加载第一页 ⇒ 批次刚起步就清掉 `loadingAll`，「加载全部」按钮中途重新可点、可重复触发
  把当前批次顶掉。
  **修法**：`if (reset && seqOverride === undefined)`——`loadAll` 通过 `seqOverride` **认领**
  `loadingAll` 并由自己的 `finally` 归位；外部导航 / 刷新（不带 seq）仍照常清。
  **红灯**：`loadAll 自己发起的 reset 不得清掉自己的 loadingAll: expected false to be true`。
- **`MigratePanel` 源 / 目标桶列举共用 `loadingBuckets` 且都无代次守卫**：先完成的一方在 `finally`
  把标志清掉，另一个 `select` 在请求未完成时就被解除禁用；切账号后旧响应仍会落地。
  **修法**：拆成 `loadingSourceBuckets` / `loadingTargetBuckets` + `sourceBucketGen` / `targetBucketGen`，
  `catch` / `finally` 均按代次收口，账号被清空的早退路径也递增代次作废在飞请求。
  **红灯**：`目标完成不得解锁仍在飞的源 select: expected undefined to be defined`、
  `过期列举不得覆盖新账号的桶`。
- **覆盖率补强（门禁拦下来的）**：新增的 **stale / catch 分支**必须逐条可执行，否则四指标掉到
  **99.97% / 99.82%** 被 `test:coverage` 拦下。补 6 条用例（源/目标列举失败清空、源过期成功不落地、
  **源过期失败不清空**、**目标过期失败不清空**、**过期版本列举失败不上抛**）→ 回到**四指标 100%**。
- **两处连带修正**：① `ObjectsPanel.test.ts` 的 `makeBrowser()` 未提供 `listGen`，运行时刷
  `Invalid prop … got Undefined` → mock 补 `listGen: ref(0)`；② helper `opts()` 被 prettier 折成
  `findAll(…)\n[i]` 触发 `no-unexpected-multiline`，**曾让 `pnpm lint` 红灯** → 改取中间变量。
- **文档同 commit 同步**：[`docs/FEATURES.md`](docs/FEATURES.md) 新增 **§AI**；§AD「待处置」表移出前端行
  （5 → **1**，TOC 加 AI）。**并清理该表既有的重复行**——前端行与 `store` 行各有一条重复（其中
  `store` 还新旧两版并存，旧版缺挂起理由），系此前某次替换只命中一处所致，本次一并删除。
  [`docs/KNOWN_ISSUES.md`](docs/KNOWN_ISSUES.md) **#64** 改「19 条已闭环 18、余 1 条挂起 Nit」；本条记录。
- **门禁复跑（全绿）**：前端 `pnpm lint` **0 告警** / `pnpm typecheck` + `pnpm typecheck:e2e` exit 0 /
  `pnpm test` **72 文件 1126 例**（1120 → 1126，连跑两轮均全绿）/
  `pnpm test:coverage` **四指标 100%（4294 / 2934 / 1130 / 3677）** / `pnpm build` OK；
  后端本轮未改，`gofmt -l` 干净 / `go vet` 0 / `go test -race -count=1` **9/9 包、每包 100.0%**、
  **零未覆盖块** / `golangci-lint` **0 issues**。
- **真实 E2E 验收（2026-09-28 收口，三项全过）**：`S3CLINET_E2E=1 go test ./internal/s3wrap/ -run TestE2E -v`
  **4/4 PASS**（`TestE2ERustFS` / `TestE2EBatch1` / `TestE2EBucketSettings` / `TestE2ETrash`；自起
  `rustfs/rustfs:1.0.0-rc.3` 于 `127.0.0.1:9000`，跑完已拆除）；`SERVER_PORT=18090 make e2e-real`
  **3 passed**（真实后端 + 真实 RustFS + 真实 `pnpm build` 产物，`S3C_TOKEN` 开启的生产同构形态；
  本机 **8080 被系统 `haproxy` 占用**故改端口；`e2e-real` 自管 RustFS 且 9000 被占会直接 `die`，
  故两套**串行**跑，跑完容器与端口已核验自动回收）；`pnpm e2e` mock 版 **15 passed**。

### 修复（2026-09-28 `s3wrap` 两条：metadata 值控制字符致 400→500 / IDN 端点建得成却永远连不上——#64 2 条先红后绿）

> `KNOWN_ISSUES #64`「待处置」表 `后端 s3wrap` 一行的 2 条，**逐条亲自读码复核**后修复。
> 证据台账见 [`docs/FEATURES.md`](docs/FEATURES.md) **§AH**。同轮第 3 条（`store` Nit）**实测后挂起**，见下。

- **`ValidateUserMetadata` 的值只查 UTF-8 与长度，控制字符只有键在查** ⇒ 值含 `CRLF` / CTL 能一路过边界，
  走到 Go transport 才被 `invalid header field value` 拒发，**边界该给的 400 变成传输期 500**——恰好违背该
  文件自述的「提前在 API 边界校验，避免落到 S3 端再以 500 形式返回」。
  **修法**：值侧按 `httpguts.ValidHeaderFieldValue` 的同一口径拒绝 `<0x20`（**HTAB 除外**）与 `0x7F`；
  `>=0x80` 属 obs-text，仍由既有 UTF-8 校验把关。
  **已确认不存在头注入**（Go transport 对 `Header.Set` 与直接 map 赋值两条路径都硬拒），所以这是 400/500 的
  **可用性**问题、不是注入面。
  **红灯**：`value with CRLF` / `lone LF` / `control char` / `DEL` 四条 → 全绿；同批钉住 `tab` / `space` /
  UTF-8 文本三个**放行**用例，防误伤正常的多行备注写法。
- **`ValidateEndpoint` 对非 ASCII 主机名（IDN）按「DNS 失败按设计 fail-open」放行** ⇒ 账号建得成，而 Go 的
  `net.Resolver` / `http.Transport` **都不自带 IDNA 转换**，`münchen.de` 每次调用都 `no such host`，
  用户只看到莫名其妙的网络错误、看不出是端点主机名非法。
  **修法**：在 `isBlockedHostname` 之前加边界拒绝，错误串直接给出出路（`use punycode xn-- instead`）。
  **不引入 `golang.org/x/net/idna`**——完整 IDNA 映射表不值得为这一处加依赖（仓库依赖纪律：能用标准库就不用
  第三方）；punycode 形式是纯 ASCII，仍走原流程。
  **红灯**：`ValidateEndpoint("http://münchen.de:9000") = nil, want error`。
- **`store.Open` json 分支丢 `storeKey`（Nit）——实测后挂起，非遗漏**：
  ① 把 `Open` 的 json 分支改走 `newStore` 会让 `store.New` 变成**零生产引用**，被
  `deadcode_gate_test.go` 的 `TestNoUnusedExportedProdSymbols` **红灯拦住**（已实际跑出该报错）；
  ② 要解开就得改 `New` 签名，而 `store.New` / 裸 `New` 共 **~100 处调用、24 个测试文件**，其中
  `crossdriver_test.go` 与 `encrypt_at_rest_test.go` **刻意依赖 `S3C_STORE_KEY` 环境变量**读 key，
  `+ ""` 机械替换会静默改掉它们的语义；
  ③ 缺陷**零生产影响**（唯一生产调用方传的 `cfg.StoreKey` 就来自同一环境变量），而代价是一次跨 24 文件的
  重构——**比例失衡**，留到有真实非 `FromEnv` 调用方时再做。实验已完整回滚，理由记入 §AD 该行与 #64。
- **门禁复跑（全绿）**：`gofmt -l` 干净 / `go vet` 0 告警 / `go build` 干净 /
  `golangci-lint run ./...` **0 issues** / `go test -race -count=1 -coverprofile` **9/9 包、每包 100.0%
  statements** 且**零未覆盖块**（正确口径）；前端本轮未改，`pnpm lint` 0 告警 /
  `pnpm test` **72 文件 1114 例** / `pnpm build` OK。
- **文档同 commit 同步**：[`docs/FEATURES.md`](docs/FEATURES.md) 新增 **§AH**、§AD 待处置表移出 `s3wrap` 行
  （7 → **5**，`store` 行补挂起理由，TOC 加 AH）；[`docs/KNOWN_ISSUES.md`](docs/KNOWN_ISSUES.md) **#64**
  改「已闭环 14 条 / 余 5 条」；本条记录。

### 修复（2026-09-28 `config` 三条：显式 env 文件静默回退改 fail-closed / 关停超时加上界 / 预建数据目录收紧 0700——#64 3 条先红后绿）

> `KNOWN_ISSUES #64`「待处置」表 `后端 config / main` 一行的 3 条，**逐条亲自读码复核**后修复。
> 第 1 条是这一批里唯一接近 Required 的。证据台账见 [`docs/FEATURES.md`](docs/FEATURES.md) **§AG**。

- **显式 `S3C_ENV_FILE` 不可读时静默回退默认值（接近 Required）**：`loadDotEnv()` 对缺失候选直接
  `continue`，而 `envFileCandidates()` 在显式设置时**只返回这一项** ⇒ 整个 `.env` 被跳过、无任何信号。
  与 `config.go` 自述口径直接矛盾（「静默回退默认值会让运维误以为配置已生效 → 改为拒绝启动」）。
  实际危害：写在该文件里的 `S3C_TOKEN` / `S3C_SSRF_DENY_PRIVATE` / `S3C_TRUSTED_PROXIES` /
  `S3C_CSP_CONNECT_SRC` **静默失效**；`S3C_DATA_DIR` 更会静默丢失 → 启动后账号列表「凭空清空」
  （原数据没坏，极易引发重复建号）。
  **修法**：`loadDotEnvFile` / `loadDotEnv` 返回 error；**仅在显式路径**下 `Stat` 失败或 `ReadFile`
  失败一律返回新哨兵 `ErrInvalidEnvFile`，经既有 `Config.envErr` → `Validate` **首查**上抛使启动失败。
  **未显式指定时行为完全不变**（缺 `.env` 是零配置可启动的常态）。
  **红灯**：`Validate() = <nil>, want ErrInvalidEnvFile`——两条用例，一条路径不存在、一条路径存在但
  读不到（用**目录**触发 `EISDIR`，不依赖 `chmod 000`：容器里 root 会绕过权限位）。
- **`S3C_SHUTDOWN_TIMEOUT` 无上界 ⇒ 优雅关停被静默清零**：`envOrInt` 只校验 `>=1`，而 `main.go` 用
  `time.Duration(n) * time.Second` 换算，`n > MaxInt64/1e9`（≈92.2 亿）会**溢出成负时长** →
  `context.WithTimeout` 立即过期 → `srv.Shutdown` 秒回、错误被 `_ =` 丢弃、**退出码仍 0**，日志只有
  `shutting down…` / `shutdown complete`，在途大文件流式传输被硬切断却无从察觉
  （代码注释「ctx 超时在生产不可达」也随之失真）。
  **修法**：`Validate` 加上界 `maxShutdownTimeoutSec = 3600`，超界复用既有 `ErrInvalidEnvValue`
  拒绝启动（与数值校验同一条错误链）。
  **红灯**：`Validate(shutdown=9999999999) = <nil>, want ErrInvalidEnvValue`。
- **预建 `0755` 数据目录永不收紧，与威胁模型声明不符**：`os.MkdirAll(dir, 0o700)` 的 mode **只在新建时
  生效**（umask 只减位不补位），Docker volume / systemd `StateDirectory` 预建的 `0755` 此后永远保持，
  而 `threat-model.md` 边界 A 明写「数据目录 0700 + 文件 0600」；文件侧早有 `chmodSQLitePerms` 主动收紧到
  `0600` 的先例，目录侧没有。
  **修法**：新增 `store.ensureDataDirPerm`，在 `store.Open` 与 `store.AcquireDataDirLock` 建目录后各调一次，
  **尽力而为**、Chmod 失败一律忽略（对齐 `chmodSQLitePerms` 的「只读挂载 / 非本进程属主不影响开库拿锁」口径）。
  **红灯**：`Open 后 data dir perm = 0755, want 0700` 与 `AcquireDataDirLock 后 … = 0755, want 0700`。
- **既有用例不动**：`TestFromEnvExplicitEnvFileMissingUsesDefaults` 仍然成立（取值层面确实回落默认值），
  只补注释指向新的 fail-closed 用例——启动层面失败与取值回落是两件事。
- **文档同 commit 同步**：[`README.md`](README.md) 的 `S3C_ENV_FILE` / `S3C_SHUTDOWN_TIMEOUT` 两行与
  `.env` 查找顺序说明、`apps/server/.env.example` 两处注释、[`docs/architecture.md`](docs/architecture.md)
  环境变量段、[`docs/DEPLOYMENT.md`](docs/DEPLOYMENT.md) SIGTERM 段、
  [`docs/threat-model.md`](docs/threat-model.md) 边界 A（注明「启动时主动 chmod 收紧」）；台账
  [`docs/FEATURES.md`](docs/FEATURES.md) 新增 **§AG**、§AD 待处置表移出该行（10 → **7**，TOC 加 AG）；
  [`docs/KNOWN_ISSUES.md`](docs/KNOWN_ISSUES.md) **#64** 改「已闭环 12 条 / 余 7 条」；本条记录。
- **覆盖率口径修正（流程）**：零覆盖块必须用**默认空白分隔**的 `awk 'NR>1 && $NF+0==0'` 判断；
  用 `-F,` 会取到 `line.col` 字段而**永远数出 0**（形同虚设的空信号）。另注意 `go test` 打印的
  `100.0%` 只保留 1 位小数，几千条语句漏 1 条仍显示 100.0%，**零覆盖块检查才是权威信号**。
- **门禁复跑（全绿）**：`gofmt -l` 干净 / `go vet` 0 告警 / `go build` 干净 /
  `golangci-lint run ./...` **0 issues** / `go test -race -count=1 -coverprofile` **9/9 包、每包 100.0%
  statements** 且**零未覆盖块**（正确口径复核）；前端本轮未改，`pnpm lint` 0 告警 /
  `pnpm test` **72 文件 1114 例** / `pnpm build` OK。

### 修复（2026-09-28 前端四条：sticky error 两处 / `DestDialog` 并发重复提交 / `signing` 死状态——#64 前端 4 条先红后绿）

> `KNOWN_ISSUES #64`「待处置」表前端第一行的 4 条，**逐条亲自读码复核**后修复，每条都先写失败测试。
> 证据台账见 [`docs/FEATURES.md`](docs/FEATURES.md) **§AF**。

- **`RecycleBinPanel.vue` 的 `error` 只写不清（接近 Required）**：4 处赋值、**0 处清空**，而模板
  `v-if="!account()"` → `v-else-if="error"` → `v-else-if="bucketSel"` 三分支**互斥** ⇒ 一次网络抖动就用横幅
  **取代整页**，点「重试」成功后 `loadMarkers` 仍不清 `error` ⇒ 必须刷新页面。另 `loadBuckets` 失败时
  `bucketSel` 为空、而重试只调 `loadMarkers(true)`（缺 bucket 直接 return）⇒ 横幅永久卡死且无任何反馈。
  **修法**：`loadBuckets` / `loadMarkers` **成功即清**；重试改为 `retry()` = 先 `loadBuckets()` 补桶选择
  再 `loadMarkers(true)`。**红灯**：`重试成功后错误横幅必须消失: expected true to be false`。
- **`BucketsPanel.vue` 同样 sticky error + 展示上一个账号的桶（接近 Required）**：`error` 无清空点，
  且 `loadBuckets` 失败分支**不重置 `buckets`**，而 `watch(accSel)` 只清 `selectedBucket` ⇒ 切到凭据失效的
  账号，会把**上一个账号的桶**渲染在当前账号选择器之下。
  **修法**：成功即清 `error`、失败时 `buckets = []`。
  **红灯**：`失败后不得渲染上一个账号的桶表: expected true to be false`。
- **`DestDialog.vue` 并发重复提交（接近 Required）**：`watch(props.open)` 里**无条件 `busy.value = false`**，
  而 `ModalDialog` 的 Esc / ✕ / 背景点击三条关闭路径都不看 `busy`，组件又是常驻（`ObjectsPanel` 只绑 `:open`、
  无 `v-if`）⇒ 飞行中任务期间关掉再开，第二次 `submitDest` 不被挡；先到的那次 `emit('submit')` 还会在第二个
  任务运行中把弹窗关掉。**修法**：**删掉这行复位**——`busy` 本就由 `submitDest` 的 `finally` 在
  成功 / 失败 / 中止三条路径归位，打开时已是 false，这行复位是多余且有害的（删代码而非加条件）。
  **红灯**：`在途任务期间 busy 不得被复位: expected false to be true`。
- **`useUploadQueue.ts` 的 `signing` 死状态（死代码红线）**：`it.status = 'signing'` 与 `it.status = 'uploading'`
  写在**同一个同步块**（中间无 `await`）⇒ 渲染永远插不进来，`UploadPanel` 的「签名中…」标签与 `abortItem` 的
  `signing` 分支**永不可达**——正是 AGENTS.md 所说「被覆盖率掩盖的死代码」（测试直接构造该状态把它们盖绿）。
  **修法**：把过渡挪到**首次字节进度回调**（`if (it.status === 'signing') it.status = 'uploading'`），presign 的
  一次网络往返期间该状态停得住；该守卫同时保证不会把已 `cancelled` 的条目改回 `uploading`。
  **选「让它可达」而不是「删掉状态」**：删除要连带动 `UploadPanel` / `i18n` / 3 个混合状态夹具共 6 个文件的
  无关断言，且会废掉 AGENTS.md 明确豁免的枚举成员；让它可达只改 1 个生产文件 3 行，既有测试从 gap 变为真实覆盖。
  **红灯**：`在途且尚无字节进度应停在 signing: expected 'uploading' to be 'signing'`。
- **三处既有夹具随正确行为更新（意图未变）**：①「加载失败显示错误与重试」原本**靠错误粘住**才看得到横幅
  （mount 触发 2 次 `loadBuckets`，第 1 次失败、第 2 次成功——正确行为本就该清掉），夹具改为两个初始调用
  都失败；② `requeue` 用例的中间断言由 `uploading` 改为 `signing`；③「桶列表加载失败…」由「重试为空操作」
  改为「重试先重拉桶」（并补断言 `listBuckets` 调用数增加）。
- **门禁复跑（全绿）**：前端 `pnpm lint` **0 告警** / `pnpm typecheck` + `pnpm typecheck:e2e` exit 0 /
  `pnpm test` **72 文件 1114 例**（1110 → 1114，净增 4 条红灯用例）/
  `pnpm test:coverage` **四指标 100%（4260 / 2908 / 1124 / 3658）** / `pnpm build` OK；
  后端本轮未改，`gofmt -l` 干净 / `go vet` 0 / `go test -race -count=1` **9/9 包、每包 100.0%** /
  `golangci-lint` **0 issues**。
- **文档同 commit 同步**：[`docs/FEATURES.md`](docs/FEATURES.md) 新增 **§AF**、§AD 待处置表移出该行（14 → **10**，
  TOC 加 AF）；[`docs/KNOWN_ISSUES.md`](docs/KNOWN_ISSUES.md) **#64** 改「已闭环 9 条 / 余 10 条」（§二 + §四台账）；
  本条记录。

### 修复（2026-09-28 破坏性操作审计覆盖补齐：同步批量移动 / 重命名删源 / 版本永久删除三处无审计——#64 Security 3 条先红后绿）

> `KNOWN_ISSUES #64`「待处置」表的后端 `handler` 一条，**亲自读码复核后确认属实**。
> 依据是 [`docs/threat-model.md`](docs/threat-model.md) 边界 A 的 **R（抵赖）** 缓解声明——
> 它明写覆盖「对象删除与前缀删除」，而这三处会**真实删掉数据**却没有任何 `objects.*` 审计事件，
> 持有效 token 的调用方可不留痕迹地删数据。证据台账见 [`docs/FEATURES.md`](docs/FEATURES.md) **§AE**。

- **同步 `POST /copy-objects` + `deleteSource`（移动）无审计**：移动是「复制成功后删源」的组合动作，
  **异步版已按 review R3 补了 `objects.move`，同步版漏了**——同一动作只因走同步 / 异步两条路由就
  可审计性不同。补 `objects.move`（`bucket` / `targetBucket` / `total` / `moved` / `failed`）。
  **红灯**：`TestCopyManySyncMoveAudits` → `同步移动未写 objects.move 审计（源被删却无痕迹，与异步版口径不一致）`。
- **`POST /rename` 全程无审计**：复制成功后 `DeleteObject` 删源，源 key 永久消失——把删除藏进
  「重命名」即可绕开已有的 `objects.delete` 事件。补 `objects.move`（带 `key` / `newKey`）。
  **红灯**：`TestRenameObjectAudits` → `重命名未写 objects.move 审计（源 key 被永久删除却无痕迹，可绕开 objects.delete）`。
- **`DELETE /version` 永久版本删除无审计**：数据不可恢复，而回收站 `trash.purge` 反而有审计。
  补 `objects.delete`（带 `versionId`）。
  **红灯**：`TestDeleteObjectVersionAudits` → `永久版本删除未写 objects.delete 审计（数据不可恢复却无痕迹）`。
- **三处全部复用 `audit.go` 既有稳定事件名，不新增契约名**——事件名是日志检索与告警的依赖；
  同时保留**负向断言**：纯复制（`deleteSource` 缺省）仍**不得**记 `objects.move`，避免把只读复制记成移动。
- **文档同 commit 同步**：[`docs/threat-model.md`](docs/threat-model.md) 边界 A 的 R 行补「移动与重命名
  （同步批量 / 异步批量 / `rename`）与版本永久删除（带 `versionId`）」、闭环指向 §AE、
  「最后更新」推到 2026-09-28；[`docs/FEATURES.md`](docs/FEATURES.md) 新增 **§AE** 并把 §AD 待处置
  表的该条移出（15 → **14**，TOC 加 AE）；[`docs/KNOWN_ISSUES.md`](docs/KNOWN_ISSUES.md) **#64**
  同步为「已闭环 5 条 / 余 14 条」（§二 + §四台账）；本条记录。
- **门禁复跑（全绿）**：`gofmt -l` 干净 / `go vet` 0 告警 / `go build` 干净 /
  `golangci-lint run ./...` **0 issues** / `go test -race -count=1 -coverprofile` **9/9 包、每包 100.0% statements**
  且**零未覆盖块**；前端本轮无代码改动，`pnpm lint` 0 告警、`pnpm test` **72 文件 1110 例**、`pnpm build` OK。

### 修复（2026-09-28 三路五轴复审：列举循环无界 ×2 / 收满上限误报截断 / `-healthcheck` IPv6 恒失败——4 条先红后绿，余 15 条转登记 #64）

> 对全仓重新跑一轮五轴复审（后端 handler+service / 后端 s3wrap+store+config / 前端 web 三路并行）。
> 发现**逐条亲自读码复核**后，只修已完成红绿的 4 条；其余 **15 条未复核的不直接采信、也不静默丢弃**，
> 登记为 [`docs/KNOWN_ISSUES.md`](docs/KNOWN_ISSUES.md) **#64**，清单与首要 / 次要标注见
> [`docs/FEATURES.md`](docs/FEATURES.md) **§AD**「待处置」表。

- **`service.deletePrefix` 列举循环无界（Required）**：退出条件只有「`maxDelete` 计数到顶」与「token 为空」，
  而**空页 / token 不前进都不推进计数**，同步 `POST /api/accounts/{id}/delete-prefix` 会一直空转到客户端断连，
  并占住 `withStreamLimit` 的 32 个流槽位（占满后 proxy / zip / copy / delete-prefix / migrate / jobs SSE 全 503）。
  修复 = 移植 `sync.go` `listAll` / `indexDst` 已有的 §B6 三道守卫：`listMaxPages` 页数上限 +
  `NextToken == ""` / `NextToken == token` 停止并标截断 + 循环首行 `ctx.Err()`。
  **红灯**：`TestRunDeletePrefixStopsAtPageCap` 报 **`未终止：空页 + 前进 token 导致死循环`**、
  `TestRunDeletePrefixStopsOnNonAdvancingToken` 报 `deleted = 100000, want 2000`。
  连带把 `TestRunDeletePrefixTruncateExact` / `CrossLimit` / `ProgressesOnAllFailures` 三个 fixture 的
  `NextContinuationToken` 从常量 `"t"` 改成**逐页前进**——真实对端不会重复同一 token，而「不前进」正是本轮守卫对象，
  常量 token 会让循环在第 2 页就停（三个用例的**断言与意图均未变**）。
- **`handler.listPrefixKeys` 同款无界（Required）**：`copy-prefix`（同步 / 异步）与 `delete-prefix` 异步列举
  **三处共用**，而 `copyPrefixAsync` 的列举跑在 2h 任务超时**之外**。修复同上（`listPrefixMaxPages` +
  `ctx.Err()` + token 守卫），循环结构对齐 `indexDst`。
  **红灯**：`TestOlListPrefixKeysStopsOnNonAdvancingToken` / `TestOlListPrefixKeysStopsAtPageCap`
  报 **`listPrefixKeys 未终止：列举循环空转`**（各 5s 超时判死）。
- **收满上限误报 `truncated=true`（Optional，边界差一）**：`len(keys) >= maxCopy` 即置截断、不看 `p.IsTruncated`，
  前缀下**正好** 100 000 个对象时会误报，而 `docs/api.md` 把该字段定义为「第 limit+1 个起未参与本次操作」，
  客户端据此会去重试一个已完成的操作。**红灯**：`TestOlListPrefixKeysExactlyLimitIsNotTruncated`
  报 `正好收满 limit 且对端声明列举完成 ⇒ 不是截断`。修复 = `indexDst` 的三段式（先判本页丢弃 → 再判收满 → 最后看
  `p.IsTruncated`）；`deletePrefix` 的同一边界本就正确，未改动。
- **`-healthcheck` 对 IPv6 字面量必然失败（Required）**：`S3C_ADDR="[::1]:8080"` 拼成
  `http://::1:8080/api/health`，`url.Parse` 报 `invalid port "::1:8080" after host` → `client.Get` 失败 →
  **恒返回 1**，Docker `HEALTHCHECK` / systemd watchdog 会把一个完全健康的服务判死并反复重启
  （`[::1]:port` 是 `IsLoopbackAddr` 认可、允许不设 token 的合法配置，属会真实用到的一类）。
  **红灯**：`TestRunHealthcheck/ipv6_loopback_literal` 报 `= 1, want 0`。修复 = `net.JoinHostPort(host, port)`
  产出 `http://[::1]:8080/api/health`（`url.Parse` 实测通过）；无 IPv6 的环境自动跳过用例，
  `no-port-in-here` 的 fail-closed 语义未变（该用例仍绿）。
- **门禁复跑（全绿）**：`gofmt -l` 干净 / `go vet` 0 告警 / `go build` 干净 /
  `golangci-lint run ./...` **0 issues** / `go test -race -count=1 -coverprofile` **9/9 包、每包 100.0% statements**
  且**零未覆盖块**——新增的 `ctx.Err()` 守卫分支各补 1 个用例，否则 `internal/service` 会跌到 **99.8%** 门禁线以下；
  前端本轮无代码改动，`pnpm lint` 0 告警、`pnpm test` **72 文件 1110 例**。
- **文档同 commit 同步**：[`docs/FEATURES.md`](docs/FEATURES.md) 新增 **§AD**（已闭环 4 条 + 待处置 15 条 + TOC）·
  [`docs/KNOWN_ISSUES.md`](docs/KNOWN_ISSUES.md) 新开 **#64**（⬜ 待复核，§二 + §四编号台账）· 本条记录。

### 修复（2026-09-28 KNOWN_ISSUES #60–#63 收口：前端测试拆分 / `SameEndpoint` 纳入 `useSSL` / 删除编排下沉 `service` / 640GB 上限证据补齐）

> 承接 2026-09-24 全仓审查转登记的 #61 / #62（本段起 **⬜ → ✅**）与既有 #60 / #63；四项**同一个提交**完成。
> 证据台账见 [`docs/FEATURES.md`](docs/FEATURES.md) §AB；条目状态见 [`docs/KNOWN_ISSUES.md`](docs/KNOWN_ISSUES.md)
> （#60–#62 已从正文移除并登记「已闭环移除」，编号不回收；#63 维持 ➖ 已决策）。下文 2026-09-24 各段里的
> 「#61 / #62 ⬜」「67 文件 1110 例」等为**该段时点值**，按「快照不回写」纪律保留原样。

- **#60 前端 4 个超 1000 行测试文件拆分（4 → 9 文件，只动文件归属、断言逐字搬移）**：`src/api.test.ts` 1986 → `api.test.ts` 887 / `api.transfer.test.ts` 677 / `api.gaps.test.ts` 573；`src/components/MigratePanel.test.ts` 1230 → 871 / `MigratePanel.branches.test.ts` 463；`src/composables/useObjectActions.test.ts` 1164 → 872 / `useObjectActions.batch.test.ts` 447；`src/composables/useObjectBrowser.test.ts` 1147 → 558 / `useObjectBrowser.branches.test.ts` 641——**最大 887 行 < 1000**。每个新文件复制完整头部后按 `vue-tsc`（`noUnusedLocals` / `noUnusedParameters`）逐条裁掉未用 import 与只被对侧使用的辅助函数（`makeObj` / `findButtonStartsWith` / `meta` / `mountActions` / `enqueue` / `captureSaves` 等），文件头注释改为「本文件范围 + 指向兄弟文件」。**等价性证据**：`vitest --reporter=json` 拆分前后全量对比 → **测试名清单 diff 为空、1110 → 1110 例**；`pnpm test:coverage` **四指标 100%（4255 / 2908 / 1124 / 3653）完全不变**（67 → 72 文件）。
- **#61 `SameEndpoint` 精确判定纳入 `useSSL`（跨包契约变更）**：签名改 6 参 `(aEndpoint, aRegion string, aUseSSL bool, bEndpoint, bRegion string, bUseSSL bool)`——裸端点按各自账号 `UseSSL` 补 scheme 后再互比，**显式 scheme 优先于开关**、端点为空时不看开关（与建 client 时 `BaseEndpoint` 同一口径），消除「裸端点 + 两侧 `UseSSL` 不同」时的判定盲区；新增 `(*s3wrap.Client).UseSSL()` 暴露建 client 时的同一值。调用点 3 处同步：`handler/migrate.go`、`handler/migrate_async.go`、`service/sync.go`。先红后绿用例：`service/zip_test.go` `TestSameEndpoint` 表驱动 **10 → 17 组**、`service/gaps_test.go` `TestSameEndpointNormalizeEdges` **7 → 9 断言**（`useSSL` 由固定 `false` 改为可变入参钉死口径）。
- **#62 批量删除编排下沉 `service`（2026-09-24 审查 Nit「本轮未完成」→ 闭环）**：新增 `internal/service/delete.go`（`DeleteCounts` / `DeletePrefixResult` / `DeleteKeys` / `RunDeletePrefix` / `deletePrefix` / `DeleteKeysBatched` / `MoveKeys`），与 `batch.go` 的 `RunBatch` / `CopyKeys` 同层；`handler/objects.go` **538 → 432 行**、`handler/copy.go` 删 `copyKeysThenDelete`（净 -21 行），handler 只剩 HTTP 边界（入参校验 → 调 service → 状态码 / 响应体 → 审计）。**纯重构、外部可见行为逐字不变**：响应体仍为 `{"deleted","failed","lastError"}` 与前缀递归口径，按 `copyBatchJSON` 既有先例在 `writeJSON` 处内联构造 map，`TestOpenAPI_EndpointResponseSchemasMatchHandlers` 全量核对不受影响。新增 `internal/service/delete_test.go`（436 行）逐条搬移原 3 个 handler 白盒用例（`TestOlRunDeletePrefix*`），并删除随之死掉的 `handler.s3UserMessageForCode` 与测试替身 `olListPagesFake`。
- **#63 维持现状，只补证据**：新增 `TestMultipartStreamCopyPartSizeIs64MB` 钉住**分段默认值 64MB**——640GB 上限是「分段大小 × 段号上限」两个默认值之积，此前只有 `TestMaxMultipartPartsIsProtocolLimit` 钉段数，证据缺一半；复核 `docker-compose.yml` / `docker-compose.prod.yml` 的 server 服务确为 `deploy.resources.limits.memory: 512M`（一块分段缓冲即 64MB，放大分段即突破内存预算）。口径三处一致：`service/stream_copy.go` 注释（段号在**上传前**判定，不误杀第 10000 段的合法对象）→ [`docs/KNOWN_ISSUES.md`](docs/KNOWN_ISSUES.md) #63 → [`docs/FEATURES.md`](docs/FEATURES.md) §AB。
- **门禁复测（全绿）**：后端 `gofmt -l` 干净 / `go vet` 0 告警 / `go test -race -count=1 ./...` **9/9 包、每包 100.0% statements** / `go build` 干净 / `golangci-lint run ./...` **0 issues**；前端 `pnpm lint` **0 告警** / `pnpm typecheck` + `pnpm typecheck:e2e` exit 0 / `pnpm test` **72 文件 1110 例**（测试名清单与拆分前逐条一致）/ `pnpm test:coverage` **四指标 100%（4255 / 2908 / 1124 / 3653）** / `pnpm build` OK。
- **文档同 commit 同步**：[`docs/KNOWN_ISSUES.md`](docs/KNOWN_ISSUES.md)（#60–#62 闭环移除 + §二 只留 #63 并补证据 + 编号台账三行改「已闭环移除」+ 最后更新 2026-09-28）+ 台账 [`docs/FEATURES.md`](docs/FEATURES.md) 新增 §AB（含 §三 门禁表「67 文件 → 72 文件」与 2026-09-28 复测段）+ [`docs/ROADMAP.md`](docs/ROADMAP.md) §四 基线日期与前端测试行 + [`docs/architecture.md`](docs/architecture.md) §2 `service` 行补「删除」编排 + 本条记录。

### 变更（2026-09-28 `docs/DEVELOPMENT.md` §7 三条历史技术债收口：已闭环项改写为「现行守卫」，消除与 KNOWN_ISSUES 的双源）

- **§7 原题「已知技术债（来自 2026-09-16 综合评估）」与「问题唯一来源 `KNOWN_ISSUES.md`」构成双源**，且三条**逐条复核后实际均已闭环**、只是文档没跟上。本条不新增问题登记——§7 改写为「**闭环状态 + 现行守卫 + 开发规则**」三列表格，原登记出处仍指 [`docs/archive/assessment.md`](docs/archive/assessment.md)（H1 / S7 / D5）。台账见 [`docs/FEATURES.md`](docs/FEATURES.md) §AC。
- **H1（OpenAPI 注册表与真实 handler 字段不一致）**：已从「人工比对」变成**机械门禁**，本轮实跑 `go test ./internal/handler/ -run 'TestOpenAPI|TestAPIDoc'` 全绿——`TestOpenAPIRequestFieldsMatchHandlerDTOs` / `TestOpenAPI_ContractRequestBodyMatchesHandlers` / `TestOpenAPIRequestDeclarationMatchesHandlerInput` / `TestAPIDocMatchesRoutes` / `TestAPIDocDocumentsRequestBodyFields` / `TestOpenAPI_EndpointResponseSchemasMatchHandlers` 等（连同 `openapi_path_params` / `openapi_query_params` / `openapi_semantics` / `openapi_shape` 等，共 11 个 `api_doc_test.go` + `openapi_*_test.go` 契约测试文件）。§7 保留的规则是「改 handler 请求 / 响应体时同 PR 同步 `openapi_register_*.go`（§4 文档同步门禁）」。
- **S7 / P0-4（SSE 终态检测分叉）**：已收敛为单一实现 `apps/web/src/api/jobs.ts` 的 `subscribeMigrateEvents`（EOF 后轮询回读直到终态 + 合成终态事件 + 心跳空闲超时 + 连续回读失败快速 `onError`），`MigratePanel` / `DestDialog` / `useObjectActions` 三调用方共用；实跑 `pnpm test src/api.transfer.test.ts src/api.gaps.test.ts` **64 例全绿**。§7 保留的规则是「新增异步任务消费方只调 `subscribeMigrateEvents`，不得自己开 SSE 流 / 判终态 / 加超时」。
- **D5（endpoint 归一化多份实现）**：已收敛为单一 helper `s3wrap.NormalizeEndpoint`，建 client（`s3wrap/client.go`）/ 预签名（`s3wrap/presign.go`）/ SSRF 拨号校验（`s3wrap/ssrf.go`）/ 同端判定（`service/migrate.go`）四处共用；实跑 `go test ./internal/s3wrap/ -run TestNormalizeEndpoint` → `TestNormalizeEndpoint` 与 `TestNormalizeEndpointNeverDoubleScheme` **PASS**。§7 保留的规则是「改端点逻辑一律走 `NormalizeEndpoint`，不要在调用点重写补全 / 去尾斜杠 / scheme 判断」。
- **顺带修正两处未跟上的「最后更新」**：[`docs/FEATURES.md`](docs/FEATURES.md) 与 [`docs/ROADMAP.md`](docs/ROADMAP.md) 的头部日期停在 2026-09-24，而 §AB 台账与 §四 门禁基线已于 2026-09-28 改写，一并推到 2026-09-28。
- **[`docs/archive/code-review-summary.md`](docs/archive/code-review-summary.md) 头部「状态更新」活块追加 ⑥**（该块的既定用法就是「正文时点值不回写、只在头部追加后续事实」）：记 #60–#62 闭环 / #63 补证据维持 ➖、前端 67 → **72 文件**、Go 文件 199 → **201**（74 生产 + 127 `_test.go`，`go list ./...` **9 包**、41410 行）、§7 双源消除；并把正文「⚠️ 待解决的技术问题」里第 2 条 **CORS**（`corsAllowedOrigin` 显式白名单 + `isTrustedDefaultOrigin` 已放行 tauri 自定义协议 + CSRF 双防）与第 3 条 **单文件超限**（Go 侧由 ② 拆完、前端侧本轮拆完）标为**已在块内追平**——两条正文条目本身按快照纪律保留。
- **门禁复跑（只改 `docs/`，零回归）**：`gofmt -l` 干净 / `go vet` 0 告警 / `go build` 干净 / `go test -race -count=1` **9/9 包、每包 100.0% statements**（含 `doc_number_gate_test.go` 文档数字门禁）/ `golangci-lint` **0 issues**；前端 `pnpm lint` 0 告警 / `pnpm test` **72 文件 1110 例** / `pnpm build` OK。

### 修复（2026-09-24 全仓代码审查 [`docs/archive/code-review-2026-09-24.md`](docs/archive/code-review-2026-09-24.md)：2 Critical + 20 Required 全清，Nit 31/35 闭环 + 3 项转登记 + 1 项判定不成立）
- **2 Critical**：**C1** 前端代理 URL 不带凭证——`S3C_TOKEN` 开启的部署下预览 / 下载全部 401（且 401 的错误 JSON 被当文件静默存盘）。`proxy.ts` 改带 `Authorization` 的 `fetch` → blob → objectURL（`downloadProxyObject`），**未**走「豁免 proxy 鉴权」的捷径；`scripts/e2e-real.sh` 注入 `S3C_TOKEN`、`e2e-real/real-backend.spec.ts` 全部 `/api` 调用带 Bearer，`make e2e-real` 在该形态下实跑 **3 passed** 即验收。**C2** `IsLoopbackAddr` 把 `:8080`（空 host）判为回环 → 非回环强制鉴权被绕过；删 `config.go` 的 `|| host == ""`，`main_test.go` 缺陷预期翻转（`:8080` → `false`）并新增「`S3C_ADDR=":8080"` 无 token 必须启动失败」用例。
- **后端 Required R1–R20**：安全四连（XFF 取末段、`withRateLimit` 移到 `withAuth` 外层、破坏性异步操作写 202 前补审计、`StoreDriver` 归一化 + 白名单）；正确性五连（mode=text 丢 `versionId`、同步列举静默截断透出 `truncated`、`RelKey` 改用 `stripPrefix` 内核、Reap TTL 改按 `finishedAt`、`Emit` 加终态保护）；边界与持久化（同端点跨账号迁移 `AccessDenied` 也回退 `StreamCopy`、原子写 `Sync()` + rename 后 fsync 父目录并两份实现收敛为 `internal/atomicfile`、建表 / 迁移错误上抛走启动失败、`List()` 循环后检查 `rows.Err()`）；性能 / 契约 / s3wrap（`List()` 查询不解密 SecretKey、`ListenAndServe` 失败退出码非 0 + `components.responses` 可序列化 + `objectItem.ContentType` 补齐、presign 不挂 `metricsMiddleware`、`HTTPStatus` 补 `ErrPartialDelete → 409`）；死代码三件（`expvar` 两指标、`errTestPresign` / `JobRegistry.Create` / `BucketOrDefault` / `deriveKeyLegacy` 与 `envelope` V2 分支——**源码级门禁测试先红后绿** + 迁移 / 内联）；同步 `copy-objects` 挂 `withStreamLimit`。
- **前端 Required F1–F5**：`setTokenPersistent` 迁移并清理 `s3c.token.<id>`（默认 sessionStorage，跨会话保留开关同时作用于 per-server token）；requeue 模式可取消已入批未开始条目；`MigratePanel` 虚拟列表行高回归 `ROW_HEIGHT=42`；`UploadQueue` 改 `:key="it.id"`；四个从未 emit 的 `error` 事件与零使用 props 删除 + 四组件异步提交防重复守卫（双击只发一次）。
- **Nit 35 项（后端 18 + 前端 17）——31 项修复、3 项转登记、1 项判定不成立**，逐项见审查报告「处置明细」表：已修含 `%w` 格式化 nil、WAL/`-shm` 0600、DSN `busy_timeout(5000)`、KDF 参数上界、`envOrInt` 显式报错、migrate 仅 `ErrNotFound` → 404、`getBucketInfo` 去 `ListBuckets` 全量拉取、`copyMany` 两分支共用 `copyBatchJSON`、sync `etag` 对分段 ETag 回退 size 比对（新增 `isMultipartETag`）、`LastError → FirstError` 全量改名、CSP `connect-src` 单源化、`EnumStr` 冗余 `string()` 清零，前端 `PAGE_SIZE` 单源、abort 后 Promise settle、0 字节文件放行、`validateDoc` i18n 化、7 处可删除行改稳定行键（改回 `:key="i"` 即 6 红）、账号回退三段复制提取 `composables/useAccountSelect.ts`、`RecycleBinPanel` / `VersionsDialog` 接 `virtualList.ts`、`selectedSize` 增量化、`t` 遮蔽改名、`withDefaults` 空 no-op 与 `MigratePanel` 空 `if` 删除、五处死导出去 `export`（保留实现）+ `deadcode_gate.test.ts` 源码形态门禁（先红后绿）。**4 项未按原样修复（3 项转登记、1 项判定不成立），一律登记不静默略过**：`SameEndpoint useSSL` → [`KNOWN_ISSUES.md`](docs/KNOWN_ISSUES.md) **#61**（技术债 ⬜）、**batch 删除编排下沉 `service` → #62（技术债 ⬜，本轮未完成**：`handler/objects.go` 删除族与 `handler/copy.go` `copyKeysThenDelete` 仍在 handler，`service` 侧只有 `RunBatch` / `CopyKeys`，已在 `deleteObjects` 注释处指路）、`stream_copy` 640GB 上限 → **#63**（已决策 ➖，S3 段号上限 × 512MB 容器内存预算的刻意取舍，注释 + 测试钉住默认值）；`App.vue` 双 `JSON.parse` 经复核**不成立**（全文 0 处），不改。
- **门禁复测（全绿）**：后端 `gofmt -l` 干净 / `go vet` 0 告警 / `go test ./...` **9/9 包**（R11 新增 `internal/atomicfile`，故由 8 包增至 9 包；**每包 100.0% statements**） / `go build` 干净 / `golangci-lint run` **0 issues**；前端 `pnpm lint` 0 告警 / `pnpm test` **67 文件 1110 例** / `pnpm test:coverage` **四指标 100%（4255 / 2908 / 1124 / 3653）** / `pnpm build` + `pnpm typecheck:e2e` exit 0；真实 E2E 两项——`S3CLINET_E2E=1 go test ./internal/s3wrap/ -run 'TestE2E'` **4/4 PASS**、`SERVER_PORT=18090 make e2e-real` **3 passed**（本机 8080 被系统 `haproxy` 占用，改端口重跑；后端 `S3C_TOKEN` 开启的生产同构形态）。
- **测试纪律**：全部按 TDD——先写会失败的测试再改实现，断言外部可见行为（返回值 / HTTP 状态码 / 渲染结果 / DOM 保留）；被旧测试固化的缺陷行为按审查结论**改预期**。新增测试文件 `internal/handler/review_20260924_test.go`、`internal/service/{review_20260924,job_finish_reap,sync_truncated,sync_multipart_etag,export}_test.go`、`internal/model/review_20260924_test.go`、`internal/store/review_20260924_test.go`、`internal/atomicfile/`、`apps/web/src/composables/useAccountSelect.test.ts`，前端 +67 例（1043 → 1110）。
- **文档同 commit 同步**：审查报告 [`docs/archive/code-review-2026-09-24.md`](docs/archive/code-review-2026-09-24.md)（含处置状态块、逐条 ✅ / ⚠️ / ℹ️、**处置明细 35 行**、修复优先级建议执行标注）+ 台账 [`docs/FEATURES.md`](docs/FEATURES.md) §AA + [`docs/KNOWN_ISSUES.md`](docs/KNOWN_ISSUES.md) 新增 #62 / #63 与编号台账两行 + [`docs/api.md`](docs/api.md) / [`docs/errors.md`](docs/errors.md)（`ErrPartialDelete → 409`、`RequestTimeout` 分类一致）+ [`docs/ROADMAP.md`](docs/ROADMAP.md) §四 门禁基线 + [`docs/archive/code-review-summary.md`](docs/archive/code-review-summary.md) 头部追加 ⑤ 条 + 本条记录。

### 变更（2026-09-24 待办来源拆分：`docs/todolist.md` → `docs/KNOWN_ISSUES.md`，功能候选归位 `roadmap`）
- **重命名 `docs/todolist.md` → `docs/KNOWN_ISSUES.md`（`git mv`，保留重命名历史）**：文件语义从「待办清单」收敛为「**已知问题**」——只收**缺陷 / 外部阻塞 / 技术债**。
- **按「问题 / 方向」两源分工重组正文**：保留 #25（⛔ 桌面端签名与公证，外部凭证阻塞）与 #60（4 个超 1000 行前端测试文件拆分，技术债）；新增「三、分类归零凭证」与「四、编号台账」两节，原六类分类的闭环证据与全部已移除编号（#1–#24 / #26–#39 / #41 / #45 / #46 及保留空号 #40 / #42 / #43 / #44）逐条可回溯，编号**不重排、不回收**，代码注释与历史提交里的旧引用仍可解析。
- **#47–#59 迁出至 `docs/ROADMAP.md` §三 3.2**：这 13 条是 ROADMAP 派生的**功能候选（非问题）**，唯一来源回归 roadmap 本身；该表「对应」列改名「原编号」并标注**已停用**（仅作历史映射），roadmap 文件头、§一结论、§三 3.2 前言同步改写。
- **SSOT 条款改写为「两源分工」**：`AGENTS.md` 文档入口表（一行 → 问题 / 方向两行）、`docs/DEVELOPMENT.md` §4「文档同步门禁」对照表、`docs/ROADMAP.md` §六 第 1 条（「单一来源」→「两源分工」，明确问题记 `KNOWN_ISSUES.md`、方向留 roadmap，同一事项只在一处登记）。
- **文档命名规则登记例外**：`KNOWN_ISSUES.md` 计入大写文件名**白名单**——`AGENTS.md` 与 `docs/DEVELOPMENT.md` 命名约定两处同步（此前规则为「仓库内不存在白名单之外的大写文件名」，2026-09-17 曾把 `API.md` / `ASSESSMENT.md` / `ERRORS.md` / `FEATURES.md` / `ROADMAP.md` 全部小写化）。理由：沿用社区通用名，便于外部工具与贡献者按字面检索；并补「新增例外必须同 PR 同时改这两处」，避免白名单分叉。
- **全仓引用收敛（51 个跟踪文件 / 128 处）**：带路径链接 29 处、裸编号引用 41 处、历史叙述 21 处。源码 / 脚本 / 配置注释（Go / Vue / TS / GitHub Actions / GitLab CI / `Makefile` / compose / nginx 示例配置）里 `todolist #N` → `KNOWN_ISSUES #N`；`README.md`、`docs/FEATURES.md`、`docs/threat-model.md`、`docs/errors.md`、`docs/archive/code-review-summary.md`、`.github/CONTRIBUTING.md` 与归档索引按新名改写。`docs/FEATURES.md` 的历史台账条目加「（时名 `todolist.md`）」限定，避免读者以为当时就叫新名。
- **豁免与死链处理**：`CHANGELOG.md` 历史条目与 `docs/archive/` 冻结正文的**时点叙述不改写**（沿用仓内「快照不回写」纪律），仅把 CHANGELOG 两条历史条目的 markdown 链接**目标**改指新文件（标签保留当时文件名）以免死链；全仓 markdown 死链自检 0 条。
- **门禁影响已确认**：无门禁枚举 `docs/` 目录或硬编码 `todolist.md`——`apps/server/doc_number_gate_test.go` 只登记 `README.md` / `docs/api.md` / `docs/ROADMAP.md` / `docs/FEATURES.md` 四个文件的数字声明，`repo_infra_gate_test.go` 与 `review_20260924_test.go` 的目录扫描范围是源码目录；故本次重命名不触达任何机械门禁。

### 变更（2026-09-24 收尾：`assessment.md` 归档冻结 + 2 个超 1000 行 Go 测试文件拆分 + 仓根游离脚手架清除）
- **`docs/assessment.md` → `docs/archive/assessment.md`（归档四步，「待归档」政策例外清零）**：① `git mv`（保留重命名历史）；② 全仓引用收敛——入链 14 处（README / `DEVELOPMENT.md` ×2 / `todolist.md` / `threat-model.md` ×2 / `FEATURES.md` ×6 / `ROADMAP.md` ×3 / `review-2026-09-19.md` 改**同目录**相对链）与报告自身 5 条出链按新位置改相对路径（`todolist.md` / `FEATURES.md` → `../…`、`../CHANGELOG.md` → `../../CHANGELOG.md`、`archive/index.md` → `index.md`）；③ [`docs/archive/index.md`](docs/archive/index.md) 归档清单补一行、「已归档 1 份」→「2 份」、**删除整段「待归档（当前政策例外）」**并把「除待归档的 assessment.md 外」的例外句改为「`docs/` 下（本目录之外）」；④ 本条记录。`docs/DEVELOPMENT.md` §4 归档条款的「当前例外：assessment.md 暂按活跃文档维护」同步改写为「当前无待归档例外」（两份均已归档）。快照**不回写**：正文历史结论、file:line、评分数字零改动，仅在头部追加**归档冻结状态行**（2026-09-24，附加状态、非回写——与 `review-2026-09-19.md` 归档前先例同款纪律）。收口：全仓旧路径残留 0（index 清单「原路径」列的历史记录、CHANGELOG 历史条目的时点叙述、源码注释的裸 `ASSESSMENT M4` 等**编号引用**三类豁免）。
- **2 个超 1000 行 `handler` 测试文件三路拆分（1038 / 1001 → 6 个，均 <1000 行）**：`openapi_contract_test.go` 按节边界拆为本体（569 行，夹具 + 分节 1–4 路由↔规范 / operation 完整性 / 路径参数 / `$ref`）+ [`openapi_shape_test.go`](apps/server/internal/handler/openapi_shape_test.go)（209 行，分节 5–6 序列化确定性与顶层形状）+ [`openapi_requestbody_test.go`](apps/server/internal/handler/openapi_requestbody_test.go)（286 行，分节 7 requestBody↔handler DTO 对齐）；`objects_test.go` 按函数边界拆为本体（300 行，HEAD / 建目录 / 重命名 / 列表与批量删除）+ [`objects_copy_test.go`](apps/server/internal/handler/objects_copy_test.go)（440 行，复制族）+ [`objects_download_test.go`](apps/server/internal/handler/objects_download_test.go)（297 行，ZIP / 代理下载 / 文件名消毒 / SetHeaders）。**只动文件归属、零断言改动**：新文件复制完整 import 块后按编译器逐条裁剪，三个文件头注释改写为「本文件范围 + 指向拆出文件」；拆分等价性以 `git worktree` 出 HEAD 与拆分后 `-list 'Test'` 全清单 diff——**254 个测试名完全一致**（md5 `7da012fa521c559b29bdd6ab4b193df9` 双侧相同），`go test ./...` 8/8 包全绿。
- **仓根 3 个游离脚手架文件删除**：`go.mod`（仓根无任何 `.go` 文件、模块路径错、且与 `apps/server/go.mod` 冲突，`go work` / `go build` 均会报错）、`package.json`（`npm init -y` 默认产物，无 scripts、无依赖，`pnpm lint` / `build` 不读取）、`pnpm-lock.yaml`（0 依赖的空锁文件）。三者均为工具误初始化残留，删除不产生行为变化、不进 `.gitignore`（根目录本就不该有此类文件，加白名单反而掩盖再次误建）。

### 新增（2026-09-24 roadmap §三 #3 死代码纪律收口：前后端两道「零生产引用」导出门禁）
- **后端 `apps/server/deadcode_gate_test.go` 扩展「生产代码导出符号零引用」判定（Gate 1）**：AST 扫描全仓 Go 文件，导出包级符号 / 导出方法一旦**零生产引用**（仅测试引用同样算死）即红灯，报出文件与处置指引；按「归一化包 clause + 符号名」分组（`foo_test` 与 `foo` 同作用域）、方法裸名归并、`//nolint` 消音豁免、门禁自身反射用例（`reflectiveMethodNames` 7 项）配合成用例兜底、const 表成员按枚举契约豁免。**变异验证**：注入 `MutantDeadExportedFunc` → 精确红灯（184 文件扫描、定位 `internal/service/job.go`、给出处置指引）→ 撤回 → 复绿且 grep 残留 0。
- **前端 `apps/web/src/deadcode_gate.test.ts` 新增非 API 模块半边（Gate 2，关闭文件头盲区 #2）**：非 `src/api` 模块的每个运行期导出必须被生产代码引用（剥掉 import 与导出声明后裸词计数，仅测试引用 = 死）；每个生产源模块必须被生产代码 import——`.vue` 组件要求默认导入（`import type` 不算在用）、`export { … } from` 再导出边计入 import 图、`main.ts` 入口（`index.html` 引用）豁免；`default` 导出与 `as` 重命名导入以前置断言拦死（出现即红灯，先改写再进门禁）；79 文件 / ≥90 导出 / ≥37 组件下限防空跑。**变异验证**：注入 `mutantDeadExport` + `MutantOrphan.vue` → 两半同时红灯并指名 → 撤回 → 7/7 复绿。
- **两道门禁首跑清出的死代码全部删除**：后端 5 个生产死导出——`openapi.OpBuilder` 链式配置器（`OpBuilder` / `Param` / `Respond`，端点已全部改为 `r.Operation(...)` 内联 `Op{}` 直注册，链只剩测试引用）与 `service.RegistryOption` / `WithMaxJobs` 选项接缝（原 `SetMaxJobsForTest` 的再包装，零生产调用），`Registry.Operation` 改无返回值、`NewJobRegistry*` 去掉空转选项参数；测试接缝统一为 `FillJobSlotsForTest(t, h)`（`export_test.go`，生产文件零 `*ForTest` 钩子）。前端 2 个真死导出——`i18nKeyCount`（`i18n/index.ts`，仅测试引用的字典计数器）与 `shouldUseMultipart`（`upload.ts`，`uploadObject` 已内联 `file.size < MULTIPART_THRESHOLD` 同义判断），引用方测试同步改写：中英键数一致由 `coverage.test.ts` 集合级比对承担、阈值分支由 `uploadObject` 用例覆盖。全量复测：后端 `gofmt` 干净 / `go vet` 0 告警 / `go test` **8/8 包** / `go build` 干净 / `golangci-lint` **0 issues**；前端 `pnpm lint` 0 告警 / `typecheck` + `typecheck:e2e` exit 0 / **66 文件 1043 例**、四指标 **100%（4072 / 2843 / 1093 / 3501）** / `pnpm build` OK。
- **文档同 commit 同步**：`docs/ROADMAP.md` §三 #3 行移出（编号不重排、#3 留空号并在 3.2 注中说明；证据指针入 `docs/FEATURES.md` §Z）+ §四 基线日期 2026-09-24 与前端数字实测更新 + R2 守卫注补两半口径；`docs/DEVELOPMENT.md` §3 门禁落点补生产导出符号与前端落点；`docs/architecture.md` §3 提法扩为两半。

### 新增（2026-09-24 仓根 `.editorconfig`：补报告「开发工具·IDE 集成」缺口）
- **新增 [`.editorconfig`](.editorconfig)**：`docs/archive/code-review-summary.md` 改进建议里唯一非桌面、可立即落地的缺口（代码质量工具 golangci-lint/staticcheck/eslint/vue-tsc 本已齐备）。只声明**已被现有工具链强制**的规则——Go / Makefile = tab（gofmt / make 强制）、TS / Vue / YAML / JSON / shell = 2 空格、Rust = 4 空格（rustfmt 默认）、统一 UTF-8 / LF / 末行换行；`*.md` 关闭行尾空格裁剪（3 份文档依赖两空格硬换行，含本报告自身）。**不新增 CI 门禁**、只影响编辑器内即时行为，与 `gofmt` / `rustfmt` 无冲突。[`docs/DEVELOPMENT.md`](docs/DEVELOPMENT.md) §3 同步登记（文档与代码同 commit）。

### 修复（2026-09-24 `docs/archive/code-review-summary.md` 事实性错误 + 已完成项误列待办 + 编造指标）
- **可核实的事实错误逐项修正**：架构分层图缺 `service` / `openapi` / `config` 三包且把「对象存储」记在 `store` 名下（实际 `store` 只管账号 / 配置落盘，对象字节读写全走 `s3wrap`，依赖方向 `handler → service → s3wrap`、`handler/store → model`）；前端「Svelte + TypeScript」→ **Vue 3 + Vite**；「179 个 Go 文件 / 约 15 万行 / 25 个包」→ 实测 **184（71 生产 + 113 测试）/ 约 3.84 万行 / 8 包**；「约 120 个 TS/JS」→ **113 个 TS/JS + 37 个 `.vue` = 150**；「所有文件 <1000 行」「大部分 500-800 行」→ **168/184 <500 行，2 个测试文件超 1000**（`handler/openapi_contract_test.go` 1038、`handler/objects_test.go` 1002，触及 AGENTS.md「约 1000 行」约束）——该两项拆分**待 handler 包 15 个测试文件的在途外部改动收敛后再动**，本轮只如实记录不硬改；测试类型「四种」→ 按 `DEVELOPMENT.md` §2 口径「三类测试 + 前端构建检查」；发布日期 2026-09-02 → `v1.0.0` 实际 **2026-09-22**。
- **已完成却列为「待解决 / 改进建议」的改判**：监控指标（`/api/metrics` 已有 54 项 `s3c_*`，默认 404 需 `S3C_EXPOSE_METRICS=1`）、安全审计（双 CI 已自动化 govulncheck + Trivy + cargo audit，非人工定期扫描）、API 文档自动生成（`/api/openapi.json` 由 9 个 `openapi_register_*.go` 运行时生成 SSOT + `api_doc_test.go` 双向门禁）、代码质量检查工具（已齐）、蓝绿部署（单实例 Compose + 桌面安装包无流量切换对象，`graceful-restart.sh` 已够，判为过度设计）、性能优化（0 个 `func Benchmark`、无已识别瓶颈，先建基准再谈）；**分布式追踪改指已立项的 `todolist` #54 / `roadmap` §三 #11，报告不另立项**。
- **编造指标与无来源内容移除**：「成功指标」的 测试通过率≥99.9% / CI<10min / 用户满意度≥4.5/5 / 可用性≥99.99% / 安全事件≤0.5次/年 在仓内**无任何采集来源**（无 SLO、无反馈通道、无事件台账），改为「已有门禁钉住的 / 可测量但未纳入 / 需先建采集通道」三档如实标注；「未来展望」的 K8s + AI 运维 + 插件架构在 `roadmap` 无对应条目，替换为真实 §3.2 候选池 `#47`–`#59`；「行动计划」的「本周内 / 下两周 / 1个月内 / 季度级」无排期来源且与正文结论矛盾，改为只指向唯一排期来源 `todolist`；「两套 CI 完全一致」改判为门禁一致、`publish` / `release-desktop` 为 GitHub 侧有意单边。
- **门禁现状如实标注**：`go test ./...` 当前 7/8 包 ok、`handler` 包 FAIL 系 15 个 staged 测试文件插入自引用 import（`t"github.com/.../internal/handler"`，缺空格）形成 import cycle，**非本轮工作所引入**，收敛后即恢复全绿；报告原「完全通过」标题与该事实矛盾，改为「除 `handler` 包外全通过」。

### 修复（2026-09-24 `docs/DEVELOPMENT.md` CI 双平台一致性表漏登 `publish` job）
- **`ci.yml` 的 `publish`（推 GHCR）在一致性表里两头不落**：表首声明「两边 job、命令、门禁阈值必须一致；改任一侧都要同步另一侧（含下表）」，但 `publish` 既没列 GitLab 对应 job、也没像 `release-desktop.yml` 那样标「不镜像」——读者会误以为 `ci.yml` 的 job 已全部对齐。现补一行：`ci.yml · publish` | **不镜像** | 推 GHCR（`needs: docker`，Trivy 通过才推；`if: != 'pull_request'` 即 push / dispatch 才推），GitLab 侧未配置 registry，门禁 `TestGitHubWorkflowPushesImage`。
- **触发表三行口径同步**：「push 到 main/develop | `ci.yml` 四个 job」实为**五个**（`publish` 也在 push 上运行）；补注「pull_request 时 `publish` 不跑」；「workflow_dispatch 全部四套」补注 `ci.yml` 含 `desktop-build` 共 6 个 job。均为文档与 workflow 现状对齐，无代码改动、无门禁读取该表（已 grep 确认零引用，不会影响现有门禁）。

### 新增（2026-09-24 roadmap 趋势展望候选池 · 13 条迭代方向）
- **`docs/ROADMAP.md` §三 拆为 3.1（已立项 / 已决策）+ 3.2「趋势展望候选池」**，新增 **#4–#16** 共 13 条候选迭代方向（均 ⬜ 未排期、非发布承诺）：**#4** MCP Server（AI 代理工具面）· **#5** S3 条件写 / CRC64 校验和 / Object Lock · **#6** 增量同步升级为 cron 定时备份 · **#7** FinOps 存储分析看板 · **#8** 上传断点续传 + 下载并行分段 · **#9** 本地文件夹 ↔ 桶同步 + PWA · **#10** OpenAPI → 前端类型 / 客户端代码生成 · **#11** OpenTelemetry trace 贯穿 + SLO 仪表盘 · **#12** SBOM + SLSA provenance + cosign 供应链证明（不依赖 E6 证书，补 R5 缺口的一半）· **#13** Token 作用域与最小权限 · **#14** 原生 fuzz 入门禁 · **#15** 多副本 / HA 评估（涉推翻 R4 / ADR-002，先 ADR 后动码）· **#16** 多平台差异化体验评估（同日入册，原表尾孤立行并入候选池并解 #4 编号冲突）。每条均写明**现有代码可承接点**与趋势依据；§一结论、§二里程碑行同步。
- **按 roadmap §六 第 1 条同步唯一待办来源 `docs/todolist.md`**：补登记 **#47–#59**（来源标 `ROADMAP §三 #N`，按章归入功能 / API 契约 / 代码质量 / 安全供应链 / 可观测性五类，状态同为 ⬜ 未排期），头部新增 2026-09-24 补登记说明，「最后更新」→ 2026-09-24。roadmap 与 todolist 两套 `#N` 编号保持独立、互指均带前缀（§六 第 6 条）。

### 变更（2026-09-23 归档：`review-2026-09-19.md` 移入 `docs/archive/` 冻结）
- **桌面端签名与公证只在唯一待办来源记录**：该报告正文 3 处记载（归档冻结块 / 执行摘要 / §9.1）全部移除，开放项仅由 [`docs/todolist.md`](docs/KNOWN_ISSUES.md) **#25**（⛔ 外部阻塞，来源 ROADMAP §三 #1）承接；归档快照不再携带开放项跟踪职责。
- **归档前全面排查（零阻塞项）**：P0 / §三 正确性（B3–B11 / F2–F10）/ P1 / P2 / S1–S6 / R2–R10 / D1–D10 逐类核销全部闭环（证据节 `FEATURES.md` §Q–§Y 齐全、todolist 零开放缺陷）；过期状态句「HEAD `c58c5e9`、这批复核修复尚未提交」随 `3d50052` 入库后以 2026-09-23 归档冻结块更正；11 个内部锚点按 GitHub slug 算法逐一核验、当前值（1042 例 / 66 文件 / 360.52 kB / 12 SHA / 70 端点）与实跑一致。
- **归档操作四步**：① `git mv`（保留重命名历史）；② 全仓引用收敛——源码 / 脚本注释 49 处（`docs/` 前缀 21 + 裸引用 28）改指 `docs/archive/`，README / features / todolist / development / 本文件同步，报告自身 32 条出链按新位置改相对路径（`FEATURES.md` → `../FEATURES.md`、`../CHANGELOG.md` → `../../CHANGELOG.md`）；③ [`docs/archive/index.md`](docs/archive/index.md) 归档清单登记、「待归档」例外仅剩 `assessment.md`；④ 本条记录。收口：全仓旧路径残留 0（index 清单「原路径」列的历史记录除外）。

### 修复（2026-09-23 `docs/todolist.md` 收口：编号台账漏登 + #18 移出）
- **头部「已移除的已完成编号」两段系统性漏登**：列表创建时（2026-09-17 `93fc632`）未回填**早期收口的 #1–#7**（#1 于 `5fdf64f` 关闭，#1–#4 于 `c7fdb18`、#5–#7 于 `d9d0bd7` 归档进 features），2026-09-22 后闭环的 **#35–#39 / #41 / #45 / #46**（§U / §V 证据，见 §五 / §六 说明）也从未入列——读者无法解释这些编号空号，误以为登记丢失。现补齐为完整序列（保留空号 #40 / #42 / #43 / #44 与在册的 #25 不入列），并把「最后更新」修正为 2026-09-23。
- **#18（`/api/health` 暴露 version）按「只收录尚未完成」约定移出清单**：2026-09-23 复审维持原行为——开源项目版本与依赖本就公开、指纹价值≈0，而该响应是安全补丁验证与运维定位的廉价通道；且代码事实为 `withAuth` **显式跳过** `/api/health`（Docker HEALTHCHECK 需无 token 探测），原「端点通常在鉴权后」表述失真、随迁一并修正。决策记录由 `docs/threat-model.md` §6.2 单点承载，`#18` 入「已移除」台账；移出后清单仅剩 #25（⛔ 外部凭证阻塞）。

### 修复（2026-09-22 威胁模型相邻失真收尾 · `.github/SECURITY.md` 三处 + `FEATURES.md` 现状表四行）
- **`.github/SECURITY.md` 与本轮修好的 `docs/threat-model.md` 相互矛盾**（该文件把 threat-model 引为权威，自身却没有任何机械门禁）：① 支持版本表仍停在「`v1.0.0-rc` 预发布阶段」——实际 `main.go` `version = "v1.0.0"` 且 `v1.0.0` tag 已存在，改为 `v1.0.0` / 时间戳版为受支持、`v1.0.0-rc*` 已被取代；② 部署建议称「`sqlite` 驱动当前明文落盘密钥」与实现相反（配 `S3C_STORE_KEY` 时 `secret_key` 列以 AES-256-GCM 密文落盘；无 key 时进程默认拒绝启动，`S3C_ALLOW_PLAINTEXT_STORE=1` 仅限联调），按实现重写并注明其余列仍为明文——故生产仍推荐 `encrypted`（整库加密）；③ CSRF 条目「强制 `application/json`」过强（实现为**非空** `Content-Type` 必须为 json、缺省放行），与 threat-model 边界 A 同款措辞对齐。
- **`docs/FEATURES.md` §一 现状表（§9 / §10）四行同款过期**：①「落盘加密 … 格式 `S3C2｜salt｜ciphertext`」→ 当前写入 `S3C3`（Argon2id 参数随文件头保存），`S3C2` 仅为兼容只读；② `encrypted`「严格 S3C2」→ 严格模式只接受受支持的加密信封（`S3C3` 当前 / `S3C2` 兼容）；③「CSRF 双防 … 强制 `application/json`」→ 改为非空 `Content-Type` 必须为 json（与上条 ③ 同源）；④「限速 … 优先 `X-Forwarded-For`」→ 仅直连对端命中 `S3C_TRUSTED_PROXIES` 时才采信 XFF 首段（默认不信任，防伪造绕过），与 threat-model 边界 A 的 DoS 行一致。
- **范围纪律**：`docs/archive/assessment.md` / `docs/archive/review-2026-09-19.md` 里的同款措辞**不改**——二者是审计 / 审查时点快照，按「不追溯篡改」保留原样；`docs/FEATURES.md` §二 历史台账（§C 的 `S-1 明文 SecretKey` / `S-2 XFF 限速绕过` 两行写的是 2026-04-19 当轮的处置事实，S3C2 与「优先采信 XFF」在当时均成立）同样保留，本轮只改 §一 现状表。收口方式：对全部活文档按「同一事实的多种措辞」逐一 grep 对源复核（`明文落盘密钥` / `X-Forwarded-For` / `S3C2` / `v1.0.0-rc` / `强制 application/json`），除上述 7 处外**零残留**（README / deployment / roadmap / api 早已与实现一致）。

### 修复（2026-09-22 子审查 724461ee 复核 · 异步任务契约统一 + 门禁口径补充）
- **异步任务失败 key 契约分裂（High，前端实际丢数据）**：`service.JobResult` 落盘/清单接口用 `failKeys`，而 `GET /api/migrate/jobs/{id}` 与 `migrate_exec.go` 用 `failedKeys`，OpenAPI / 前端 `types.ts` / `MigratePanel.vue` / `docs/api.md` 也读 `failedKeys`——重启后的 `interrupted` 任务在「未完成任务」视图里读不到失败对象列表。现统一公共字段为 `failedKeys`（`JobResult` json tag），并在 `job_persist.go` 的 `UnmarshalJSON` 里兼容旧落盘名 `failKeys`；OpenAPI `jobResultSchema` 与 `docs/api.md` 的 list 示例同步。新增 `TestOlMigrateJobsListUsesFailedKeys`（清单接口 result 必须含 `failedKeys`、不得泄露 `failKeys`）与 `TestJobResultUnmarshalAcceptsLegacyFailKeys`（新旧两种落盘格式都能读出）。
- **query 字面量读取的空白盲区（Medium）**：`q.Get( "x" )` / `q[ "x" ]` 既不被字面量正则抽取，也被动态键正则误判为「非空白后接引号」而跳过（fail-open）。四条抽取正则允许括号/方括号内空白，动态下标正则补 `\s` 排除；新增空白字面量与空白动态键的合成用例。
- **请求体解码 receiver 校验（Medium）**：`parseBodyCalls` 只按 selector 名判定 `readJSON`/`NewDecoder`，`other.readJSON` / `other.NewDecoder` 会误报。现限制为 `h.readJSON(...)` 与 `json.NewDecoder(...)`；`findReadJSONTarget` 同样要求 receiver 为 `h`。新增 `TestParseBodyCallsDecodeReceivers` 与 `other.readJSON` 负例。
- **死代码符号计数作用域（Medium）**：`findUnusedExportedTestSymbols` 原按全仓标识符名计数，不同包同名导出符号会互相抵消。现按「归一化包名 + 符号名」计数（`foo_test` 与 `foo` 同作用域），新增 `TestFindUnusedExportedTestSymbolsScopesByPackage` 合成跨包同名用例。
- **文档收口**：`docs/archive/review-2026-09-19.md` 状态块 HEAD 更新为 `c58c5e9`；`docs/archive/index.md` / `docs/DEVELOPMENT.md` 明确 `assessment.md` / `docs/archive/review-2026-09-19.md` 为「被大量引用的待归档例外」；`docs/ROADMAP.md` / `docs/todolist.md` 图例补 `⛔`；`docs/todolist.md` #28 日期改 2026-09-22 / §W；`docs/FEATURES.md` §G 标题改为「已记录的 Unreleased 项（已完成）」；`docs/DEVELOPMENT.md` #37 标注已闭环。

### 修复（2026-09-22 子审查复核 · 调用闭包 AST + 报告断言范围收口）
- **契约门禁的调用闭包仍走裸正则（fail-open）**：path / query / request-fields / semantics 四处在沿 `h.xxx()` 找委托闭包时用 `callRe = h\.(\w+)\(` 扫方法体，注释或字符串字面量里的 `h.ghost()` 会被当成真实调用，污染闭包并掩盖漂移。现统一改为 AST：path/query/request-fields 用 `parseBodyCalls(...).calls`，semantics 用新增的 `handlerMethodCalls(fd)`，删除 `callRe`。新增 `TestParseBodyCallsIgnoresCommentsAndStrings` 钉住「注释/字符串里的 `h.x()` 不进入闭包」。
- **报告对响应门禁范围的表述收口**：`docs/archive/review-2026-09-19.md` / `docs/FEATURES.md` 明确「自由体」指**顶层** property 自由体——顶层自由体仅 `/api/openapi.json` 一个；`metadata` / `fields` 等嵌套 `openapi.Obj()` 只比对到顶层键，不计入端点级 `untyped`。同时把「md 叙述性数字门禁」明确为「md 端点数量叙述数字门禁」（当前仅登记这一类），并把响应覆盖限缩为「顶层 property 名级」——required / type / enum / 嵌套语义与 helper 内部直接 `writeJSON` 未覆盖，`openapi_response_contract_test.go` 文件头同步写明。
- **门禁实跑**：`go test ./...` 8/8 包通过 / `golangci-lint run ./...` 0 issues / `gofmt -l .` 干净。

### 修复（2026-09-22 门禁自身追加复核 · query 动态键 fail-open + AST 口径 + async 请求体自由体）
- **query 绑定变量动态键此前会 fail-open**：`openapi_query_params_test.go` 的动态键检测只匹配 `r.URL.Query().Get(name)` / `r.URL.Query()[k]`，而抽取又只认字面量 `q.Get("x")`；于是 `q := r.URL.Query(); q.Get(name)`（或 `q[k]`）既不被抽成参数、也不被标记为动态读取，注册表漏声明时可以全绿。已补 `hasDynamicQueryRead` 对所有绑定变量的检查，并新增合成用例覆盖绑定后的 `Get(name)` / `q[k]` 与「字面量不得误报」。
- **`PathValue` 动态键检测改 AST**：`openapi_path_params_test.go` 旧实现用裸正则扫整段方法体，注释/字符串里的 `PathValue(name)` 会误报红灯。新增 `hasDynamicPathValueRead` 走 `go/parser`，并用 `TestPathParamDynamicReadUsesAST` 钉住「真实动态键必报、注释/字符串/字面量不误报」，删除旧正则变量。
- **`readJSON` 定位改 AST**：`openapi_request_fields_test.go` 旧实现用逐行正则找 `readJSON(r, &x)`，注释或字符串字面量里的调用会被当成真实解码点。新增 `findReadJSONTarget`（AST 定位真实调用并返回行号，`findDecodeSite` 改为消费它），并加 `TestFindReadJSONTargetIgnoresCommentsAndStrings` 口径测试。
- **`delete-prefix/async`、`copy-prefix/async` 顶层请求体从自由体升级**：注册表此前用 `openapi.Obj()`，请求体字段门禁跳过字段集比对。现与同步端点共用具体 `properties`（`deletePrefixBody` / `copyPrefixBody`），请求体字段门禁覆盖这两个端点；剩余 `metadata` / `fields` 等嵌套自由体仍只比对到顶层键（已在门禁文件头登记）。
- **文档同步**：`docs/archive/review-2026-09-19.md` 加当前状态块（HEAD / 数字 / 工作区非干净）、修正 P2 未闭环的旧摘要、更新过期数字与行号、明确附录 C 只描述 2026-09-19 当轮；`docs/DEVELOPMENT.md` / `docs/ROADMAP.md` / `docs/FEATURES.md` 同步 AST 口径、golangci linter 列表与门禁落点。
- **门禁实跑**：后端 `go test -count=1 ./...` 8/8 包通过 / `golangci-lint run ./...` **0 issues**；前端 `pnpm test` **66 文件 / 1042 用例**全绿 / `pnpm build` OK（360.52 kB，gzip 110.85 kB）。

### 新增（docs 归档目录：为时点性文档提供冻结存放处）
- **新增 `docs/archive/` 与索引 [`docs/archive/index.md`](docs/archive/index.md)**：此前时点性文档（综合评估、分支 / 版本审查等）没有约定的归档位置——要么留在 `docs/` 根下与活跃 SSOT 混放，要么在文档收敛时被直接移除（上一轮「文档收敛」即删掉了三份历史评估快照，追溯证据随之丢失）。现明确：结论被后续工作取代的时点性文档用 `git mv` 移入 `docs/archive/` **冻结**——不删除、不回写、不改写历史结论；`decisions/` 的 ADR **不归档、不删除**（被取代时新写一篇引用旧篇并标 `Superseded`）。索引内写明「什么该归档 / 什么不该归档 / 归档操作四步」。
- **约定登记**：[`docs/DEVELOPMENT.md`](docs/DEVELOPMENT.md) §4 补「归档」条目、把 `docs/archive/` 纳入目录清单、并新增一行「文档命名 / 存放位置 / 归档」的文档同步门禁；[`README.md`](README.md)「文档」段加归档入口。**当前无归档文件**，本批只落约定，供后续归档使用。
- **验证**：纯文档改动，不触碰代码与门禁；`docs/archive/index.md` 及本批改动文件内的全部相对链接已用脚本逐一核验可解析（无死链）。

### 修复（2026-09-22 审查 §4.3「维持」项复核 · 门禁自身可被绕过）
- **`openapi_inputsource_test.go` 用裸字符串匹配，注释即可骗过**：该门禁判断「handler 是否真的解码请求体」时用 `strings.Contains(body, "readJSON(")`，于是 `// 这里提到 readJSON(` 与 `s := "readJSON("` 都被判为「解码了」。实测把 `createAccount` 的真实 `readJSON` 调用删掉、只在注释里留一句 `readJSON(`，**旧实现仍全绿**（假绿）。现改为 **AST 判定**（`parseBodyCalls` 解析语法树，只认真实调用表达式），新增 `TestDecodesBodyIgnoresCommentsAndStrings` 钉住口径。**变异验证**：删真实解码留注释 → 新实现红灯。
- **`api_doc_test.go` 的缩进盲区让陈旧条目静默留存（fail-open）**：解析正则只认行首，缩进行在双向 diff 中**两个方向都不匹配**。实测在 `docs/api.md` 插入一条缩进的陈旧路由 `DELETE /api/nonexistent-stale`，`TestAPIDocMatchesRoutes` **静默通过**。现新增 `TestAPIDocRoutesAreFlushLeft`：路由声明行必须顶格，缩进即红灯（含扫描下限自检）。**变异验证**：插入缩进陈旧条目 → 新门禁红灯。
- **`deadcode_gate_test.go` 的整行正则对行内形状全部逃逸**：旧实现匹配 `^_ = ident` / `^var _ = expr`，实测 `x := 1; _ = x`、`if true { _ = x }`、`_ = x.Field`、`_ = s[0]` **全部不被拦**（fail-open）。现改为 **AST 判定**（`findSilencingDeadCode`）：按语句判定，只有「左侧全为 `_`」才算消音；`_, ok := m[k]`（comma-ok 惯用法）与 `_, _ = w.Write(b)`（显式丢弃返回值）放行，`var _ Iface = (*T)(nil)`（编译期接口断言）放行。新增 `TestFindSilencingDeadCodeCoverage` 断言「AST 命中数严格大于旧正则」（实测 9 vs 5），防止退回旧口径。
- **`//nolint` 消音此前无任何门禁**：`AGENTS.md` / `DEVELOPMENT.md` 明令禁止用 `//nolint` 消音（「那只是把死代码藏起来」），但实测在函数上方加一行无理由的 `//nolint:all`，`golangci-lint run` 仍报 **0 issues**——`//nolint` 是 golangci 自身的指令，默认不受任何 linter 审查。现 `.golangci.yml` 启用 `nolintlint`（`require-explanation` + `require-specific` + `allow-unused: false`）。**变异验证**：植入 `//nolint:all` → 红灯。
- **path 参数缺 handler 侧门禁（新发现的缺口）**：`TestOpenAPI_ContractPathParamsDeclared` 只校验注册表**内部**自洽（模板 `{x}` ⇔ 声明 `in:path`），**没有任何一条**比对 handler 是否真的 `r.PathValue("x")` 读取它。现新增 `openapi_path_params_test.go`：注册表 `in:path` 参数名集 ⇔ handler 沿调用闭包 `PathValue` 读取集**双向**比对（70 个端点 / 60 个带 path 参数），含非字面量键检测。**变异验证**：把 `getAccount` 的 `PathValue("id")` 改为 `PathValue("accountId")` → 同时报「漏声明」与「幻影参数」。
- **门禁实跑**：后端 `go vet ./...` 0 告警 / `golangci-lint run ./...` **0 issues**（含新启用的 `nolintlint`）/ `go build ./...` OK / `go test ./...` 全绿（8/8 包，每包 100.0% 覆盖且零未覆盖块）。

### 修复（2026-09-22 审查 §7.4 门禁盲区收尾 · 响应契约 + 4 处真实漂移）
- **`s3wrap.CorsRule` 缺 `json` tag（CORS 规则在浏览器里被静默清空）**：该结构体经 handler **直接序列化**进 `GET /api/accounts/{id}/bucket/cors` 的响应体，却没有 json tag → Go 输出 PascalCase（`AllowedMethods` / `MaxAgeSeconds`），而前端 `types.ts` 的 `CorsRule` 与 `docs/api.md` 都是 camelCase。Go 的 `json.Unmarshal` 大小写不敏感，**服务端既有测试一直全绿**；但 JavaScript 严格区分大小写，浏览器读 `x.allowedMethods` 恒为 `undefined` → `BucketCors.vue` 把方法与来源归一化成空数组。已补全 6 个 camelCase tag（`id` 带 `omitempty`）。回归测试：`TestCorsRuleJSONFieldNames`（双向：必须含 camelCase、不得再有 PascalCase）、`TestCorsRuleJSONRoundTrip`（前端发来的 camelCase 能被解析）。**变异验证**：去掉 tag → 两条红灯并列出实际 PascalCase 键。
- **`POST /api/migrate/async` 注册表误声明 `200`**：handler 写 `http.StatusAccepted`（202），`docs/api.md` 也写 `202`，只有注册表写 200。已改为 202，并由端点级响应门禁钉住。
- **`docs/api.md` 称 `/api/openapi.json`「不进鉴权层」**：与实现相反——`withAuth` 只豁免 `health` / `metrics`；配了 `S3C_TOKEN` 时无凭证访问返回 401，且需 `S3C_EXPOSE_OPENAPI=1` 否则 404（既有 `TestOpenAPI_GateAndAuth` 早已钉住该行为，纯文档失真）。已改为「经过鉴权层」并写明两个前置条件。
- **`docs/FEATURES.md` 称 69 个 `/api/*` 端点**：实际为 70（`routes.go` 70 条 `mux.HandleFunc`，README / `api.md` / `ROADMAP.md` 与 `apiRouteCount` 常量均为 70）。已改为 70。
- **`docs/api.md` 的 `PUT /api/accounts/{id}` 缺请求体 JSON**：此前只有散文描述（「字段同创建」），导致 `TestAPIDocDocumentsRequestBodyFields` 无法机械比对字段集而红灯。已补显式 JSON 示例（必填 `name`/`endpoint`/`accessKey`，`secretKey` 可省略保留原值），与注册表 required 声明一致。
- **门禁实跑**：后端 `go vet ./...` 0 告警 / `golangci-lint run ./...` **0 issues** / `go build ./...` OK / `go test ./...` 全绿（含新增门禁）。

### 新增（2026-09-22 审查 §7.4 门禁盲区收尾 · 补门禁而非只补缺陷）
- **端点级响应门禁全量收敛（自由体 `openapi.Obj()` → 具体 `properties`）**：把 accounts / buckets / bucket-settings / objects / object-meta / multipart / versions / trash / migrate / system 十个注册表的自由体响应全部升级为 `openapi.BuildObj`，嵌套形状抽为共享构造器（`corsRuleSchema` / `tagRowSchema` / `lifecycleRuleSchema` / `versionEntrySchema` / `deleteMarkerSchema` / `jobProgressSchema` / `jobResultSchema` / `jobRecordSchema`）。端点级门禁覆盖面 **15 → 61** 个成功响应，自由体从约 45 个降到 **1 个**（仅 `/api/openapi.json`：响应体即规范本身，由 `Registry.HTTPHandler()` 直接写而非 `writeJSON`，机械抽取无意义）。自检从 `checked ≥ 15` 收紧为 `checked ≥ 60` 且 `untyped ≤ 1`——**注册表回退成自由体会直接红灯**。
- **query 参数门禁扩到四种读取口径**：抽取器从只认 `Get()` 扩到 `Get` / `Has` / `Values` / `Query()["x"]`，并覆盖 `q := r.URL.Query()` 绑定后的同名形式；新增**非字面量键检测**（`q.Get(name)` / `q[k]` 命中即红灯），杜绝动态键读取静默逃逸。新增口径测试 `TestQueryReadExtractorCoversAllForms`（8 种形态逐一断言 + 「无读取不得抽出参数」的反向断言）。
- **新增 md 叙述性数字门禁**（审查 §7.4 矩阵里唯一标「否」的行）：`apps/server/doc_number_gate_test.go` 以 `routes.go` 的 `mux.HandleFunc` 注册数为唯一真值，校验 README / `api.md` / `ROADMAP.md` / `FEATURES.md` 中「N 个 `/api/*` 端点」的 N；要求每条声明**至少命中一次**（文案漂移致正则失配即红灯，防门禁静默失效）。**变异验证**：把 `FEATURES.md` 改回 69 → 红灯。
- **新增 `_test.go` 导出符号死代码门禁**：`golangci-lint unused` 对**导出**符号因「可能被包外引用」而豁免，但 `_test.go` 不参与库构建、永远无包外引用——实测给 `_test.go` 加无人调用的导出函数/类型，`golangci-lint run` 报 **0 issues**。新增 `TestNoUnusedExportedTestSymbols` 扫描全部 `_test.go` 的导出包级符号，无引用即红灯（`export_test.go` 作为约定的测试接缝整体豁免）；检测逻辑抽为纯函数 `findUnusedExportedTestSymbols`，由 `TestFindUnusedExportedTestSymbols` 用**合成源码**做口径测试，不依赖「仓库里正好有个死符号」来证明门禁有效。

### 修复（2026-09-22 文档失真批次：roadmap 门禁基线 · threat-model · 活文档行号引用）
- **`docs/ROADMAP.md` §四 门禁基线整体过期**：表头标注「实测 2026-09-19」，但 09-20 / 09-22 两批改动（审查 P2、#37 真实联调）合入后数字未刷新，违反本仓库「门禁数字必须来自实跑」约定。本轮**全部门禁实跑取真数**并同步：前端测试 986 例（64 文件）→ **1042 例（66 文件）**；前端覆盖率补实测值（4074 / 2844 / 1095 / 3503 四指标 100%）；E2E 行补上 2026-09-22 新增的第三套工作流 `e2e-real.yml`（`make e2e-real` 实跑 3 passed），并把 mock 版与真实联调版拆成两行；新增此前漏登记的 `golangci-lint run ./...`（v2.13.2，0 issues——它是 AGENTS.md 硬门禁与 R2 守卫，却一直不在基线表内）；前端 lint / 类型行补上 `eslint src e2e e2e-real` 与 `typecheck:e2e` 口径。5 个 E2E workflow action SHA 重新经 GitHub API 核验（均 200）。
- **`docs/FEATURES.md` §三 现状表同步**：前端测试 1039 → **1042**，并补 2026-09-22 复测注记段（历史注记按「不追溯篡改」保留原值追加，不改写）。
- **`docs/threat-model.md` 状态与语义失真（5 处状态翻转 + 3 处语义 + 行号漂移）**：① 安全审计日志仍标「❌ 未缓解（todo #17）」、DoS 行仍以「XFF 伪造、job 无上限」为缺口——`handler/audit.go`、`S3C_TRUSTED_PROXIES` 可信代理白名单、JobRegistry ≤256 均已闭环，R/D 两行改 ✅；② §3「`s3c.servers` 把 token 写 localStorage（待修复）」与实现相反（`storage.ts` `writeServers` 只存 `{id,name,base}`，#15 已归档），改为已闭环描述；③ §6 两条开放项（审计日志缺失 / XFF 伪造）划线；④ presign「过期钳制 [1h, 24h]」改为真实的「≤0 默认 1h、>24h 钳 24h、无 1h 下限」并补 `multipart.go` 同款钳制；⑤ 桶名校验补漏禁 `--` 规则；⑥ metadata「字节总长」不存在——实际为 key≤128 / value≤256 字符 + ≤10 标签；⑦ action SHA「10 个」→ **12 个**（全量 workflows 去重，本轮逐一经 GitHub API 复验均 200）；⑧ 全文 9 处 `file:line` 行号引用（`config.go:120`、`main.go:74`、`objects.go:433` 等已全部漂移）改为**符号引用**（函数 / 常量名）——与「测试名不得硬编码源码行号」同一教训，根治行号腐烂；顺手修 `.trivyignore` 注释里的过期 `alpine:3.20`（Dockerfile 实为 3.24）。
- **`docs/threat-model.md` 二次全量对源复核（3 处事实错误 + 2 处精确度 + §2/§6 结构补全）**：① 边界 D 的 IMDS 归属错误——`100.96.0.2` 是**火山引擎**内网元数据而非阿里云（`ssrf.go` 注释原文即为两家），拆分归属；② §4 元数据 tags 单位错误——前一条 bullet 所述「字节总长」修正后写的「≤256 **字符**」实为 `len()` **字节**口径（错误消息文案 chars 属既有措辞），改正并注明；③ §边界 C 的「todolist #29/#31」死指针（编号已从 todolist 正文移除）改指 features §T 归档证据；④ Tampering 行「强制 application/json」过强——实现为**非空时**必须 json、缺省放行（安全结论不变，Bearer 头本就强制预检），按实现措辞；⑤ §6 拆为「6.1 已闭环（证据归档）+ 6.2 仍接受的风险（health 暴露 version / metrics 免鉴权 / SSRF 私网放行 / 明文 store 放行开关，共 4 项）」并去掉无实义的「与路线」标题——原节清一色已闭环项，读者无从知晓仍被有意接受的风险；⑥ §2 安全默认值表补 3 行（`/api/health` 免鉴权、`S3C_TRUSTED_PROXIES` 默认空、`S3C_SSRF_DENY_PRIVATE` 默认关），ReadHeaderTimeout 位置改符号 `runServer`；⑦ `.trivyignore`「空清单」改「仅策略注释、0 忽略条目」；⑧ 文件头删 CHANGELOG 式括注（明细归本文件）。STRIDE 覆盖边界 B–E 的补表**另开一轮**（改动大，不在本批）。
- **`docs/threat-model.md` STRIDE 补全（边界 B–E 各 6 行状态表，承接上一条的「另开一轮」）**：§1 标题自称「STRIDE × 边界」，但此前仅边界 A 有完整 STRIDE×状态表，B–E 只有缓解 bullet，读者无法逐类核对。本轮把 B/C/D/E 各补成与 A 同构的 6 行表（不适用 / 已决策放行的类别逐行给理由，不用「没写」来回避），新增 §1 状态图例（✅/⚠️/❌/➖）。新增事实全部先对源复核：presign 方法白名单 `get|put|post`（缺省 put、非法 400）与签发端点在 Bearer 鉴权内（`routes.go`）；密文篡改链路（`decryptAESGCM` 认证失败 → `store.Open` 失败 → `main` return 1 硬退出，ADR-002）；数据目录 flock 单写者锁（`AcquireDataDirLock`，`LOCK_NB` 第二实例立即失败）与 0700/0600 权限；Tauri 无 `invoke_handler`、capabilities 源真值 `capabilities/default.json` `permissions: []`（`gen/schemas/capabilities.json` 系未跟踪的过期构建产物——mtime 早于源文件——不作真值）。原 B/C/D/E 的 bullet 全部并入表格，无信息丢失。
- **活文档 file:line 引用腐烂与 ADR / 快照指针失真（threat-model 之外的残留清零）**：① `docs/decisions/0002`：`main.go:61-66` / `health.go:9-18` 均已漂移（`store.Open` 现在 `main.go:81` 附近、503 由 `health.go` `health` 回）——ADR 按 `docs/archive/` 约定**不归档、属活文档**，改符号引用（`main.go` `runServer` · `health.go` `health` / `store.Ping`）；② `decisions/0003`：`config.go:120` 改符号（`FromEnv`，`S3C_ADDR` 默认回环），并修 IMDS 归属同款错误——`100.96.0.2` 是**火山引擎**而非阿里（全仓第 2 处也是最后一处，另一处在 threat-model 已修，源注释 `ssrf.go` 原文即为两家）；③ `FEATURES.md` 两处：§E 台账行 `metadata.go:40` 去行号（遮蔽循环已改名，行号无从对证）、§U row 9 的例证行号「实际」改「当时」时态（历史证据保留数字，只消歧现在时误读）；④ `assessment.md` **仅头部处置指针**补「§N 起持续归档见 features / CHANGELOG」与「正文 file:line、版本号、测试数字均为审计时点值」——正文历史结论零改动（快照纪律）；⑤ `docs/archive/review-2026-09-19.md` 整份不动：审查对象 pinned 到 `develop@0fbd560`，时点自证。收口方式：对全部活文档（README / AGENTS / architecture / api / development / deployment / errors / features / roadmap / todolist / threat-model / decisions / archive / .github）按 `(go|ts|vue|js|yml|yaml|json|sh|conf|mod|Dockerfile):数字` 模式全量扫描 → **0 处无时点限定的引用**（两处纪律豁免：快照 `assessment` / `review` 整份不动；`features` §U row 9 的历史例证行号保留原数字、已加「当时」时点限定——它是「测试名硬编码行号」修复的证据本体，抹掉数字即抹掉证据）。

### 新增（2026-09-22 真实后端 + 真实 RustFS 浏览器联调 · todolist #37）
- **新增真实联调浏览器冒烟（不 mock `/api`）**：此前 `e2e-playwright.yml` / `playwright-e2e` 的 `/api/**` 全被 `page.route` mock，没有任何门禁验证「真实 Go 后端 + 真实 RustFS + 真实构建产物」拼在一起时的行为——尤其**浏览器直传**（预签名 PUT 是浏览器 → S3 的跨源请求），mock 时代在结构上不可能覆盖。现新增 `apps/web/e2e-real/real-backend.spec.ts` + 专用 `apps/web/playwright.real.config.ts`（不自拉 webServer，`PLAYWRIGHT_BASE_URL` 指向真实后端），3 条用例：账号 CRUD 真实落库（`secretKey` 不回传、UI「测试连接」走真实 S3）、建桶 → 列桶（真实 `CreateBucket`/`ListBuckets`）、**浏览器真实直传 → 列对象 → 预签名 GET 回读逐字节校验**。用例先清空后端账号表，保证每个用例从真实空状态出发。
- **本地一键运行（docker 自动起 RustFS）**：新增 `scripts/e2e-real.sh` + `make e2e-real`，自动 docker 起一份真实 RustFS（显式配 `RUSTFS_CORS_ALLOWED_ORIGINS`，否则浏览器直传被 CORS 拦下）→ `pnpm build` 真实产物 → 起真实 Go 后端（`S3C_STATIC_DIR` 托管产物）→ 跑 `pnpm e2e:real` → `trap` 自动清理；支持 `--keep` / `--skip-build`。
- **两套 CI 接入**：GitHub 新增 `.github/workflows/e2e-real.yml`（PR 按路径 + 手动 + 每周五定时）；GitLab 新增 `e2e-real` job（RustFS 用 GitLab service，与 `rustfs-e2e` 同款，service 变量里配好 CORS）。触发规则与另两套 E2E 对齐，不阻塞普通 push。
- **编排收敛为单一来源**：初版把「起 RustFS → 构建产物与后端 → 起后端 → 跑用例」在脚本 / GitHub / GitLab 各写了一份（改一处漏两处即漂移）。现 `scripts/e2e-real.sh` 是唯一编排：两套 CI 都 `bash scripts/e2e-real.sh`（GitHub 自起容器；GitLab 用 `--no-rustfs` + `RUSTFS_ENDPOINT` 复用 service），各自只留「装工具链 / 装浏览器系统依赖」。脚本同时补上 `curl` 依赖检查、`--no-rustfs` 时只清自己起的容器、以及幂等的 `playwright install chromium`（修掉新克隆下 `make e2e-real` 直接失败的 UX 缺口）。
- **机械门禁防漂移**：`TestRealE2EUsesSharedScript`（两侧 CI 必须**实际调用** `bash scripts/e2e-real.sh`——仅出现在 `paths:`/`changes:` 里不算）、`TestRustFSImageIsConsistentlyPinned`（脚本默认值 / compose / GitLab service 三处镜像版本必须一致，禁 `latest`）、`TestRealE2EArtifactsExist`（联调 spec 出现 `page.route(` 即红灯）、`TestE2ESourcesAreTypechecked`、`TestLocalRealE2ETargetExists`。全部门禁均经变异验证（改镜像版本 / 删 CI 里的脚本调用 / 删 `typecheck:e2e` 都会红灯）。
- **补齐 E2E 源码的静态检查**：`e2e/` 与 `e2e-real/` 此前不在主 `tsconfig.json` 的 `include` 内，**零类型检查与 lint**（`pnpm lint` 只扫 `src`）。现新增 `apps/web/tsconfig.e2e.json` + `pnpm typecheck:e2e`，并把 `pnpm lint` 扩到 `eslint src e2e e2e-real`（顺带修掉一个此前未被发现的未使用变量）；两套 CI 的 `web` job 与本地 `make check` 都纳入。
- **变异验证**：去掉 RustFS 的 `RUSTFS_CORS_ALLOWED_ORIGINS` 后重跑——浏览器直传用例红灯，前两条不经浏览器的用例仍绿，证明该用例确非空断言。

### 修复（2026-09-22 审查 §9.3 P2 收尾 · 正确性 + 发布链 + 门禁质量）
- **D1 Prometheus 直方图语义违规（`/api/metrics` 输出全错）**：`s3wrap` 内部按「每次调用只 +1 个桶」记录**增量**，但 `handler/metrics.go` 直接把它当**累积**值按 `_bucket{le=…}` 输出——实测 `le="0.01"`=3、`le="+Inf"`=0、`_count`=3，违反 Prometheus 契约（`le` 必须单调不减且 `+Inf == _count`），`histogram_quantile()` 结果全错。现 `MetricsSnapshot` 在快照时把增量累积为真累积值；并把 `calls` 从 `atomic.Int64` 移入同一把锁（此前 `calls` 与 `buckets` 分属两处更新，快照可读到 `+Inf`=0 而 `_count`=3 的不一致）。注释同步改为准确描述。回归测试：`TestS3MetricsHistogramIsCumulativeAcrossBuckets`（多耗时累积值逐桶断言）、`TestS3MetricsHistogramConcurrentContract`（8×50 并发下契约不变）、`TestMetricsHistogramSatisfiesPrometheusContract`（端点级：解析 `le` 单调性与 `+Inf == _count`）。已用「还原增量语义」变异验证三条均红灯。
- **P3 `/api/openapi.json` 每次请求全量 marshal**：70 个 operation 此前每个请求都重新渲染。现 `Registry` 增加结果缓存，**任何注册动作（`Operation` / `Param` / `Respond` / `SetInfo` / `AddServer`）都置空缓存**，语义与「每次重新 marshal」完全一致；并发调用双检加锁只计算一次，且返回缓存副本（调用方改写不污染后续请求）。回归测试：`TestRegistry_MarshalJSONCached`（含副本隔离）、`TestRegistry_MarshalJSONCacheInvalidated`（四类注册动作逐一验证失效）、`TestRegistry_MarshalJSONConcurrent`。
- **D4 `failKeys ≤ 200` 承诺与实现不符**：`docs/FEATURES.md` / `docs/api.md` 承诺「failKeys ≤ 200」，但服务端只有异步 `delete-prefix` 在 handler 内裁剪，`copy-objects`（同步/异步）、`migrate`（同步/异步）、`migrate/sync` 全部原样回传——10k 全失败会回约 10 MB 并落盘 `jobs.json`。现把上限提为 handler 层共享常量 `maxFailKeys = 200` + `capFailKeys`，**所有回传 `failedKeys` 的端点统一裁剪**；异步路径经新增的 `jobResultFromBatch` 在 `Job.Finish`（落盘）**之前**裁剪。`failed` 计数不受裁剪影响。`service/batch.go` 与 `copy.go` 里两处互相矛盾的注释（「由调用方裁剪」vs「RunBatch 已裁剪」）改为与实现一致。回归测试：`TestOlCopyManyFailKeysAll`、`TestOlMigrateSyncFailKeysCapped`、`TestOlMigrateAsyncFailKeysCappedBeforePersist`（直接检查 `jobs.json`，证明裁剪发生在持久化前）。
- **R7 本地 `make` 与 CI 不等价**：`Makefile` 缺 `lint`（golangci-lint）/ `govulncheck` 目标，`test-all` 不含覆盖率门禁（本地全绿不代表 CI 会绿）；裸 `pnpm install` 会改写锁文件，导致 CI 的 `--frozen-lockfile` 失败；`test-cover` / `install-hooks` 未列入 `.PHONY`。现新增 `lint` / `govulncheck` / `check`（聚合静态检查 + 双侧覆盖率）目标，全部 `pnpm install` 改为 `--frozen-lockfile`，`test-all: test-cover web-test-cover`，`.PHONY` 补全全部目标。回归门禁 `TestMakefileMirrorsCIGates`（含「所有已定义目标必须在 .PHONY」的机械检查）。
- **R9 GitHub 构建不传 `VERSION` build-arg**：镜像内版本回落到 Dockerfile 的过期字面量，而 GitLab 传 `VERSION=ci`、Makefile 传真实版本——同一个 Dockerfile 三种行为。现 GitHub 的构建与推送两处都显式注入 `VERSION=ci`。门禁 `TestGitHubWorkflowInjectsVersionBuildArg`。
- **R10 无 workflow 推送镜像**：「发布」此前只有 tag + 桌面安装包，用户无法 `docker pull` 到 CI 产物。新增 `publish` job：`needs: docker`（Trivy 扫描通过后才推送）、`if: github.event_name != 'pull_request'`、`permissions: packages: write`、`docker/login-action` 登录 GHCR、按 commit SHA 与分支双标签推送。因 buildx 的 `push` 与 `load` 互斥且扫描需 `load`，故拆为独立 job（也避免把未扫描的镜像推出去）。门禁 `TestGitHubWorkflowPushesImage`。
- **D5 `git tag v1.0.0` 不存在但 roadmap 称已收口**：用 `scripts/release-version.sh v1.0.0` 把 15 处版本串从 `1.0.0-rc1` 同步到 `1.0.0`（Makefile / Dockerfile / compose×2 / web+desktop `package.json` / `tauri.conf.json` / `Cargo.toml` / `Cargo.lock` / `main.go` / README / docs），并打上 `v1.0.0` tag。历史性提及（CHANGELOG 已发布区、roadmap 里程碑行、review 报告）按「不追溯篡改」纪律保留。
- **D10 `CHANGELOG.md` 内过期计数**：`[Unreleased]` 段内两处「当时实测值」（覆盖率 3883/2769/1059/3339、前端 63 文件 / 983 测试）后续用例增加后已失真。按本文件「不追溯篡改」的纪律**保留原值并加注**当时的时点与新值，不改写历史数字。
- **测试质量：21 个用例名硬编码源码行号且已过期**（如 `api.test.ts` 写 `line 125 else`，实际 `listServers` 在 `storage.ts:301`；`bucketPolicy.test.ts` 写 `line 111`，实际在 `:108`）——测试在描述一个不存在的版本，属实现耦合而非行为断言。已从全部用例名中移除行号（改为描述行为，行号留在注释里），并新增机械门禁 `deadcode_gate.test.ts` 的「测试名不得硬编码源码行号」（含扫描下限自检防解析失效）。
- **测试质量：i18n 门禁两处盲区**：① `usedKeyTexts` 对整份源码做正则，**注释里的键形字符串也算「已使用」**——删掉真实引用、只在注释留个键名，死键门禁仍全绿。现先按字符扫描剥离注释（正确跳过字符串字面量，避免把 `'https://x'` 的 `//` 当注释）再匹配。② 中英双语只比对**键数量**，`zh-CN` 缺 `a` 而 `en-US` 多 `b` 时数量相等、门禁仍绿，用户切到英文会看到原始 key。现按语言分别解析键集合做**集合级双向比对**，并新增「每个键的两种语言取值都非空」。两条均经变异验证（改名一个 en-US 键使数量不变 → 集合门禁红灯；删真实引用只留注释 → 死键门禁红灯）。
- **文档失真修正**：`openapi_contract_test.go` 的 `apiRouteCount` 注释算错（写「69 = 68 + 1」而常量与实际均为 70，应为「70 = 69 + 1」）；`docs/todolist.md` 补记 #40 / #42 / #43 / #44 为**从未启用的保留空号**（避免被误读为漏登记），并把 #15 补入已完成编号。
- **门禁实跑**：后端 `gofmt -l` 干净 / `go vet ./...` 0 告警 / `go test -race` 8/8 包通过且**每包 100.0%**；前端 `pnpm lint` 0 告警 / `pnpm typecheck` 通过 / `pnpm test:coverage` **66 文件 1042 用例全绿，四项覆盖率 100%**（statements 4074 / branches 2844 / functions 1095 / lines 3503）。

### 修复（2026-09-20 审查 §9.3 P2 · 契约 + 安全 / 供应链）
- **#26 OpenAPI 注册表漏声明 query 参数（含 1 个幻影参数）**：注册表未声明 4 个 handler 真实读取的 query 参数——`head.versionId`、`proxy.maxBytes`、`trash.prefix`、`objects.startAfter`（后者此前连 `docs/api.md` 都未记录），按 OpenAPI 生成的客户端根本不会发送它们。已全部补声明（`maxBytes` 为整数并注明默认 1 MiB / 上限 2 MiB）。**新增全量门禁** `openapi_query_params_test.go`：解析 `routes.go` → handler 方法调用闭包 → 机械抽取 `r.URL.Query().Get("x")` / `q.Get("x")`，与注册表 `in:query` 集合**双向**比对（含自检计数防解析口径失效）。上线即抓到**第 5 处漂移**：`GET /api/accounts/{id}/versions` 声明了 handler 从不读取的幻影参数 `key`——已移除。`docs/api.md` 同步补 `startAfter`。
- **#27 类型 / required / 枚举语义无门禁**：新增 `openapi_semantics_test.go`。① **枚举门禁**：每个注册表 `enum`（请求体字段或 query）必须在 `enumContracts` 登记，并从 handler 的 `switch` case 字符串 / `map[string]bool` 字面量 / 跨包 `service` 常量机械抽取真实接受值做双向比对（空串除外）；上线即发现 `PUT object-acl` 注册表漏了 handler 实际接受的 `authenticated-read` / `aws-exec-read`。② **required 门禁**：注册表声明 required 的请求字段必须是 handler 真实解码、且非「空值→默认」的字段；据此修正 `presign.method`、`copy-object.newBucket`、`copy-prefix.targetBucket` 三处「注册表 required 但 handler 对空值取默认」的错误声明。残留范围（未识别的 `bucketOr` 默认桶模式）写入文件头。
- **#28 响应门禁残留范围（端点内联 / `map[string]any`）**：扩展 `openapi_response_contract_test.go`，新增**端点级响应门禁**：对每个声明了具体 `properties` 的 2xx 响应，机械抽取 handler `h.writeJSON` 的表达式键（字面量 map 键、命名 struct 的 `json` tag、变量、同包 helper 返回值递归），与注册表属性做双向比对（`omitempty` 记为可选）。同时把 4 个注册表文件里约 30 个 `openapi.Obj()` 响应升级为 `BuildObj` 具体属性以纳入门禁，并修正 3 个异步端点误声明 `200`（handler 实为 `202`）。仍未覆盖的残留（其他注册表文件的自由体响应、运行时动态键 map）已在文件头写明。
- **#29 / #31 明文密钥落盘改为安全默认（硬失败）**：此前 `json` / `sqlite` 驱动在 `S3C_STORE_KEY` 为空时把 `secretKey` 明文落盘，仅启动 WARN。现 `Config.Validate` 直接返回新的 `ErrPlaintextStoreNotAllowed` **拒绝启动**，除非显式设置 `S3C_ALLOW_PLAINTEXT_STORE=1`（仅本地联调；此时仍打「明文落盘」WARN）。错误信息给出两条出路（设 `S3C_STORE_KEY` ≥ 16 字符 / 显式 opt-in），且不泄露任何密钥值。base `docker-compose.yml` 同步改为 `S3C_STORE_KEY: "${S3C_STORE_KEY:?…}"` 强制非空（缺失即 compose 启动失败）；`.env.example`（根与 `apps/server`）补 `S3C_ALLOW_PLAINTEXT_STORE` 说明；`e2e.yml` 因 `docker compose up` 会插值整个文件而补一次性 dummy key。回归测试：`TestValidatePlaintextStoreRejected`、`TestMainServerRejectsPlaintextStoreWithoutOptIn`、`TestMainServerAllowsPlaintextStoreWithOptIn`、`TestFromEnvAllowPlaintextStore`。
- **#30 `/api/metrics` 不受 `S3C_TOKEN` 保护（文档显式声明）**：行为不变（有意为内网 Prometheus 免 token scrape），但此前文档未说明「即使配了 token 也匿名可读」。已在 `README.md`（变量表 + 安全默认值）、`docs/api.md`、`docs/DEPLOYMENT.md` §6.3、`docs/threat-model.md` §2 显式声明，并新增 `TestMetricsEndpointUnauthenticatedEvenWithToken` 钉住该行为（配 token + 开 metrics 时无 `Authorization` 仍 200，而 `/api/accounts` 无 token 为 401）。
- **#32 Trivy 0.58.1 → 0.74.0 + digest 双 pin**：两套 CI 由 `aquasec/trivy:0.58.1` 升级到 `aquasec/trivy:0.74.0@sha256:62b1e65e…1969`（tag + digest 不可变 pin，digest 经 `docker pull` 与 ghcr.io / public.ecr.aws 清单三源核对）。CI 依赖的 `--vuln-type` 自 0.58 起 deprecated，改为 `--pkg-types os,library`（0.74.0 的 `trivy image --help` 与 `pkg/flag/package_flags.go` 已核实）。回归门禁 `TestTrivyImageIsVersionAndDigestPinned`：两套 CI 必须版本一致、必须带 digest、必须用 `--pkg-types` 且不得残留 `--vuln-type`。
- **#33 工具链 pin（Rust / Node / desktop pnpm）**：`dtolnay/rust-toolchain` 此前 pin 的 SHA 实为 `stable` **浮动分支 tip**（commit 即 `toolchain: stable`，并非版本 pin）——改为版本分支 SHA `ce678459…`（`# 1.98.1`）；新增 `apps/desktop/src-tauri/rust-toolchain.toml`（`channel = "1.98.1"`）；GitLab 桌面镜像 `rust:1-bookworm` → `rust:1.98.1-bookworm`。Node 由只 pin 大版本（`24`）收紧到精确 patch `24.21.0`（workflow `node-version`、`node:24.21.0-alpine`、`node:24.21.0-bookworm`、NodeSource `nodejs=24.21.0-1nodesource1`）。`apps/desktop/package.json` 补 `packageManager: pnpm@9.15.0`（与 web / CI 一致；已实测 `pnpm@9.15.0 --frozen-lockfile` 不破坏锁文件）。门禁：`TestRustToolchainIsVersionPinned`、`TestNodePinnedToPatchVersion`、`TestPackageManagerIsPinnedToCIPnpm`。
- **#34 `.dockerignore` 漏覆盖率 / 缓存目录**：此前漏掉 `apps/web/coverage`、`.pnpm-store`、`test-results`、`.run/`、`.cargo/` 等（本机实测 >4 MB，另有 `.gitlab-ci-local` 体积更大）→ 同一 commit 因本地是否跑过覆盖率而产出不同镜像层。已补齐上述目录及 `playwright-report`、`.trivy-cache/`、`coverage.out`/`.txt`、`*.tsbuildinfo`、`*.log` 等，并把过窄的 `apps/server/*.log` 换成 `**/*.log`。门禁 `TestDockerignoreExcludesBuildArtifacts`（11 条必需模式，注释行不计入）。
- **门禁实跑**：`gofmt -l` 干净 / `go vet ./...` 0 告警 / `golangci-lint run ./...` **0 issues** / `go test -race -count=1 -coverprofile` 8/8 包通过且**每包 100.0% 语句覆盖**（`awk '$NF==0'` 零块）/ `go build ./...` OK；前端 `pnpm lint` 0 告警 / `pnpm test` 66 文件 1039 用例全绿 / `pnpm build` OK；`docker compose config` 在缺 `S3C_STORE_KEY` 时退出 1、补齐后退出 0。新增门禁均经「先红后绿」验证，供应链 pin 均经上游实测核验（Trivy / Node / Rust 版本与 digest）。

### 修复（2026-09-19 审查 P1 · §9.2 全部闭环）
- **§7.2 响应契约失真：`components.schemas.Account` 与真实 DTO 不符**：schema 声明了 `provider` / `forcePathStyle` / `insecureSkipVerify` 三个**幻影字段**（`model.AccountView` 无此字段），却**漏掉真实的 `useSSL`**——而它是 `GET /api/accounts`、`POST /api/accounts`、`GET|PUT /api/accounts/{id}` 全部 200/201 响应的契约，按此 schema 生成的客户端会读到永远为 null 的字段、并丢失 `useSSL`。已按 `model.AccountView` 逐字段修正。回归测试：新增 `TestOpenAPI_AccountSchemaIsAccountView`（逐字段断言 + `secretKey` 不得出现）。
- **新增响应契约门禁（handler DTO ⇔ OpenAPI schema 双向比对）**：此前所有门禁只覆盖**请求**方向，**没有任何门禁把「响应 schema」与真实响应 DTO 做比对**——这正是上一条长期存活的根因，也是审查 §4.3/§7.4 点名的唯一缺失门禁类型。新增 `apps/server/internal/handler/openapi_response_contract_test.go`：对 `components.schemas` 的每个共享 schema（`Account` / `Bucket` / `ObjectItem` / `ListObjectsResp` / `Error`）用 `go/parser` 机械抽取对应 Go DTO 的 `json` tag，做**双向**比对（schema 多写 = 幻影字段；漏写 = 客户端丢字段），并强制新共享 schema 必须在 `schemaDTOs` 登记（防逃逸）。上线即对 `Account` 红灯，修正后转绿。文件头写明断言范围：仅共享 schema，不含各端点内联 / `map[string]any` 拼装的响应。
- **§4.3 文档字段门禁改双向并去掉子串匹配**：`api_doc_test.go` 的 `TestAPIDocDocumentsRequestBodyFields` 原用 `strings.Contains(body, name)` **单向**校验（注册表 ⊆ 文档），有两处结构性盲区：① 文档多写的幻影字段从不检查；② 子串碰撞（字段 `key` 被 `keys`/`secretKey` 满足、`newKey` 被 `newKeys` 满足）。现改为机械抽取 `docs/api.md` 请求体 JSON 的**顶层键**（自写容错扫描器：剥离 `//` 注释、支持嵌套对象、区分 `200 {…}` 响应体）后**双向**比对，彻底移除子串匹配。别名段同时支持「请求体与 `POST /api/x` 相同」与省略方法名的「请求体与 `copy-prefix` 相同」。
- **§4.3 请求体字段门禁从「硬编码端点族」改为「全量遍历」**：原 `openapi_contract_test.go` 的字段断言是**逐端点族硬编码**（2026-09 修过的那批），新端点写错字段仍可 100% 全绿。新增 `apps/server/internal/handler/openapi_request_fields_test.go`：解析 `routes.go` → handler 方法调用闭包 → 定位真正 `readJSON` 的方法（覆盖 `parseMigrateRequest` / `parseCopyPrefix` 委托解码）→ 抽取解码结构体字段集，与注册表 requestBody schema **全量**双向比对（端点专属 DTO 双向相等；共享模型走 `serverManagedFields` 白名单；注册表自由体跳过并计数自检）。**上线即抓到 3 处客户端可见漂移**：`POST …/storage-class` 注册表漏 `versionId`、`POST /api/accounts/preview-buckets` 注册表与 `docs/api.md` 漏 `publicEndpoint`/`useSSL`——已修注册表与文档。
- **R2 无 tag ↔ 清单一致性门禁（P0 级）**：`release-desktop.yml` 解析出 tag 后**从不**与 `tauri.conf.json` / `Cargo.toml` 比对，给 `v1.0.0-rc2` 打 tag 会发布**标着 rc1** 的安装包。新增 `Verify tag matches manifest versions` 步骤：tag（去前缀 `v`）必须等于两处清单版本，否则 `exit 1`。本地实测 rc1 通过、rc2 正确失败。
- **R3 `SHA256SUMS` 跨平台竞态**：三平台矩阵此前各自 `gh release upload SHA256SUMS.txt --clobber` **同一资产名**，并行执行下「最后写入者胜」，发布出去的校验清单只覆盖一个平台。现改为各平台上传唯一命名的 `SHA256SUMS-<bundle>.txt`，新增 `aggregate-checksums` job（`needs: publish`）拉取三份合并去重为唯一 `SHA256SUMS.txt`（含行数自检防漏拉），并清理中间资产。
- **R4 Trivy DB 下载无缓存/重试**：仓库自带日志记录过 `FATAL … connection timed out`（6m44s 后失败），网络抖动会让安全门禁以「构建失败」形式变红。两套 CI 均拆出 `trivy image --download-db-only` + 4 次指数退避重试，扫描阶段加 `--skip-db-update` 复用已就绪 DB；GitHub 侧加 `actions/cache`（`.trivy-cache`），GitLab 侧加 `cache: paths`。
- **S1 运行镜像基础版 EOL**：`apps/server/Dockerfile` 运行阶段 `alpine:3.20`（安全支持已于 2026-04-01 结束）→ `alpine:3.24`（支持至 2028-06-01）。EOL 分支叠加 Trivy 的 `--ignore-unfixed` 会让「所有未来 CVE 永远无修复版」，安全门禁结构性失明。
- **S2 `.env.example` 占位 token 是「有效口令」**：根 `.env.example` 的 `S3C_TOKEN=change-me-use-openssl-rand-hex-32` 长 33 ≥ `MinTokenLength=16`，满足 compose 的 `${S3C_TOKEN:?}` 非空守卫——照 README 快速开始 `cp .env.example .env && docker compose up -d` 会以**公开已知 token** 上线。已置空（对齐 `apps/server/.env.example`）。
- **配置类回归门禁（防复发）**：新增 `apps/server/repo_infra_gate_test.go`，把上述四项配置断言变成会变红的测试——`.env.example` 的 token 不得 ≥ `MinTokenLength`；运行镜像不得是 EOL alpine（基线常量 `minSupportedAlpineMinor`，附 3.21 EOL 时须抬到 3.22 的说明）；发布 workflow 必须做 tag↔清单比对与平台内唯一 checksum；两套 CI 的 Trivy 必须缓存 + 重试。这四项都是「YAML/配置正确但断言缺席」的盲区。
- **门禁实跑**：`gofmt -l` 干净 / `go vet ./...` 0 告警 / `golangci-lint run ./...` **0 issues** / `go test -race -count=1 ./...` 8/8 包通过且**每包 100.0% 语句覆盖**（`awk '$NF==0'` 零块）。新增门禁均经「先红后绿」验证。

### 文档（2026-09-19 审查 P1 同批：§7.3 文档失真 D2 / D3 / D6 / D8 / D9）
- **D2 修正 `openapi_handler.go` 的鉴权注释**：原注释称 `/api/openapi.json`「不经鉴权层」，与实际相反——配置 `S3C_TOKEN` 时无 token 返回 **401**（`withAuth` 只豁免 health/metrics），未开启 `S3C_EXPOSE_OPENAPI` 时返回 404。注释改为与实测一致。
- **D3 修发布脚本 `scripts/release-version.sh`**：正则原为 `^v1\.0\.0-<时间戳|rcN>$`，会**拒绝纯 `v1.0.0`**（roadmap 首个稳定里程碑）与 `v1.0.1` / `v1.1.0`；现放宽到通用 `vMAJOR.MINOR.PATCH` + `-rcN`/`-alphaN`/`-betaN`/时间戳。同步文件从 8 处补到 12 处（新增 `docs/DEPLOYMENT.md` / `docs/ROADMAP.md` / `docs/FEATURES.md` / `.github/ISSUE_TEMPLATE/bug_report.md` / `internal/openapi/openapi.go`）；`Cargo.lock` 改用 awk **只改本包** `name = "s3clinet"`，避免全局 `sed` 把 400+ 依赖包的 version 行改坏。已实测 `v1.0.0` / `v1.1.0` / rc 三种输入均正确同步。
- **D6 修 presign `expiresIn` 文档**：`docs/api.md` 原写「范围 1s–7 天、默认 15 分钟」，代码是**缺省默认 1 小时、超过 24 小时钳到 24 小时**（`objects.go`）；已改为与代码及 `architecture.md` 一致。
- **D8 修「3 路并发」失真**：现行能力描述（`README.md` / `docs/architecture.md` / `docs/FEATURES.md` 能力表）改为实际值 **2 路**（`UPLOAD_CONCURRENCY = 2`）；0.3.0 历史条目保留原样（不追溯篡改历史台账）。
- **D9 修 `docs/DEVELOPMENT.md` 存放约定自相矛盾**：原段先说 `CHANGELOG.md` 是根目录约定文件，随后又写「其余文档（含 `CHANGELOG.md`）统一放 `docs/`」；改为「除根目录约定文件与 `.github/` 社区健康文件外的其余文档放 `docs/`」。
- **SSOT 修复：`docs/todolist.md` 补登记本轮全部开放项**：此前该文件写着「无待办」而审查仍有大量开放发现，违反「本文件是唯一待办来源」的约定。现按 #26–#46 补登记 P2 全部条目（契约残留 / 安全 / 发布链 / 可观测性 / 文档失真），并注明编号稳定不重排。

### 修复（2026-09-19 审查 §三 正确性 · 后端 B3–B11）
- **B3 `DeleteObjects` 吞掉 200 响应体内的逐 key 失败**：S3 对「部分 key 删不掉」（桶策略 / 保留期 / MFA Delete）**仍返回 200**，只在响应体内列 `<Error>`；旧实现只看顶层 `err`，于是 4 个调用点全部把「请求数」当成「已删除数」——受保护对象删不掉，UI 却报「已删除 N 个」。现在 `s3wrap.DeleteObjects` 返回 `[]DeleteFailure{Key,Code,Message}`（`err` 只表示传输/协议失败，两者可同时非空），`POST …/delete` 回 `{"deleted":成功数,"failed":失败数,"lastError":"access denied"}`，`delete-prefix`（同步/异步）与回收站 `PurgeObject` 同样按「请求数 − 逐 key 失败数」记账，`PurgeObject` 遇部分失败上抛新的 `ErrPartialDelete`（回收站不再显示「已彻底清除」而版本仍在）。回归测试 `TestDeleteObjectsReportsPerKeyFailures` / `TestDeleteObjectsPartialFailureAcrossBatches` / `TestPurgeObjectRefusesPartialDelete` + handler 三端点 `TestOlDeleteObjectsReportsPartialFailure` / `TestOlDeletePrefixReportsPartialFailure` / `TestOlDeletePrefixAsyncReportsPartialFailure`。
- **B3 连带：递归删除的进度口径改为「已处理数」**：`runDeletePrefix` 原来只按 `deleted` 前进，桶策略让删除全部失败时同一页会被反复列出、反复失败直到 2h 任务超时。现按 `deleted + failed` 前进并在 `truncated` 处收口，`TestOlRunDeletePrefixProgressesOnAllFailures`（100 页全失败仍正常终止）锁死该行为。
- **B4 ZIP 打包遇首个拷贝错误即 `break` → 永久 goroutine + 连接泄漏**：`results` 是无缓冲通道，一旦停止消费，其余 3 个 worker 永久阻塞在发送上（`wg.Wait(); close(results)` 协程永不返回，已取回未写出的 body 不关闭）。触发路径包含**用户取消 ZIP 下载**（`ctxCancelReader` → `io.Copy` 报错）。改为 `continue` 后失败 key 照常进入清单、在途 body 全部关闭。测试 `TestWriteObjectsZipCopyErrorDoesNotLeak`（断言所有已取回 body 最终被关闭且 5 个 key 全部上报；旧实现会泄漏 3 个 body）。`FEATURES.md` 中「P-3 zip goroutine 泄漏 ✅」的结论至此才真正成立。
- **B5 增量同步吞掉列举错误**：源端 403/5xx 时 `SyncKeys` 回 `200 {scanned:0}`，用户读到的是「没有需要同步的对象」。现 `listAll`/`indexDst` 返回 `error`，`SyncKeys` 签名改为 `(SyncResult, error)`，handler 走 `writeInternalErr`（源端 AccessDenied → **403**）；`cancelOrErr` 把「ctx 取消（客户端放弃）」与「真实列举错误」分开：前者仍返回已完成的部分结果且不报错，后者必须上抛。测试 `TestMigrateSync_SourceListFailureIsReported`（端点级，断言 403 且响应体不含 `scanned`）、`TestSync_SourceListErrorIsReturned` / `TestSync_DstListErrorIsReturned`。
- **B6 `indexDst` 无页数上限且循环内不查 ctx**：旧的循环条件是「已收集数 < 10 万」，对端只要返回**不前进的** `NextContinuationToken` 就会空转到 2h 任务超时。现在 `listAll`/`indexDst` 共用一组硬上限（100 页 × 1000 key × 10 万总量）+ 每轮 ctx 检查 + 「NextToken 未前进即停」。测试 `TestIndexDstStopsOnNonAdvancingToken` / `TestListAllStopsOnNonAdvancingToken`（带 5s 超时护栏，旧实现会挂死）/ `TestIndexDstStopsAtPageCap` / `TestIndexDstStopsOnCancelledContext`。
- **B7 `Job.Emit` 在锁外投递、`Finish` 关闭同一批 channel**：`Emit` 先在锁内快照订阅者、解锁后才发送，`Finish` 在同一把锁下清空订阅表、解锁后 `close` 这些 channel——两者之间没有互斥，一旦并发就是 **send on closed channel → panic，进程直接退出**（无 recover）。现改为**在锁内非阻塞投递**（`select` + `default`，持锁时间有界），落盘 I/O 移到锁外。测试 `TestJobEmitFinishConcurrentDoesNotPanic` 用「节流落盘钩子」把 Emit 卡在「快照之后、投递之前」再调用 `Finish`——旧实现**确定性 panic**，新实现通过；另加 20 轮 × 50 次并发压测。
- **B9 8 MB 请求体上限与「10 000 key 批量」的承诺冲突**：`maxBody` 原为 8 MiB，而 `maxBatchKeys` 允许 10 000 个 key（1 KB/key ≈ 10.3 MB）——`io.LimitReader` 把合法请求截断成 JSON 语法错误，回的是 400「invalid request body」而不是 413。现上限提到 16 MiB，并用 `io.LimitedReader{N: maxBody+1}` 区分「超限」与「JSON 无效」：超限时 `writeBadJSON` 回 **413** + `request body too large (max 16MB)`。测试 `TestReadJSONAcceptsDocumentedMaxBatch`（10 000 × 1 KB 载荷可解析）/ `TestReadJSONRejectsOverLimitBody` / `TestReadJSONRejectsMaxBodyPlusOne` / `TestDeleteObjectsBodyTooLargeIs413`。
- **B10① 分段上传第 10000 段被误判超限**：`partNum++` 之后才判 `partNum > 10000`，而**段号 10000 是合法的最后一段**——正好 10000 段的对象会被 abort 并报「exceeds part limit」。改为**上传前**判断（`partNum > maxMultipartParts`），并把手写常量提为可注入的 `maxMultipartParts`（默认 10000，测试断言其未被改动）。测试 `TestMultipartStreamCopyAcceptsExactlyMaxParts`（旧实现必失败）/ `TestMultipartStreamCopyRejectsPartOverLimit`（第 N+1 段根本不发出）。
- **B10② `Content-Disposition` 的 `filename*` 用 `QueryEscape`**：空格被编成 `+`，而 RFC 5987 的百分号编码里 `+` 是字面加号——浏览器会把 `my file.txt` 存成 `my+file.txt`。新增 `rfc5987Escape`（按 attr-char 白名单逐字节 `%XX`，非 ASCII 走 UTF-8 字节）。测试 `TestOlProxyContentDispositionRFC5987`（含中文文件名）。
- **B10③ download-zip 不校验空 key**：`GET /bucket/?key=""` 会被 S3 当成「列举桶」，把 ListBucket XML 当成对象内容塞进 ZIP 包。现在空 key 直接 400。测试 `TestOlDownloadZipRejectsEmptyKey`。
- **B10④ `mode=text` 忽略读错误仍回 200**：`io.ReadFull` 的错误被丢弃，传输中断会被渲染成「内容只有这么多」。现在非 `EOF`/`ErrUnexpectedEOF` 的读错误走 `proxyErr`；对 `ErrUnexpectedEOF` 再用 `Content-Length` 判定是否短读（短读同样报错）。测试 `TestOlProxyTextMalformedChunkIsNot200`（畸形 chunked 编码）/ `TestOlProxyTextShortBodyIsNot200`。
- **B10⑤ 分段顺序不校验**：`CompleteMultipartUpload` 的 `Parts` 必须按 `PartNumber` 升序，乱序时 S3 回 `InvalidPartOrder`，而它没进错误映射表 → 用户拿到 **500**（其实是自己的请求错了）。现在 handler 在提交前拒绝降序/重复段号（400），`HTTPStatus`/`UserMessageForCode` 也补上 `InvalidPartOrder → 400 / invalid request`。测试 `TestOlMultipartCompleteRequiresAscendingParts`（乱序 400、升序 200）、`TestHTTPStatusCoversAllBranches` 新增用例。
- **B10⑥ 指标标签取自服务端原始 `<Code>`，基数无界**（同 §S5）：`errorClass` 直接返回 `ErrorCode(err)`，而错误码来自用户配置的**不可信**端点——对端每次返回不同的 `Code` 即可让 `s3c_s3_call_errors_total` 的序列数无限增长。现改为白名单（20 个已识别的 S3 错误码）内的原样返回，其余一律归 `other`。测试 `TestErrorClassUnknownCodeFoldsToOther`。
- **B11 `statusRecorder` 未覆写 `Write`**：只 `Write` 不 `WriteHeader` 时 Go 隐式发出 200，而 recorder 的 `written` 仍为 false、后续显式 `WriteHeader` 会再次透传（Go 打 superfluous 日志），并把日志/指标里的状态改写成与实际响应不符的值。现覆写 `Write` 标记「已写出」。测试 `TestStatusRecorderWriteMarksWritten` / `TestStatusRecorderWriteHeaderFirst`。

### 修复（2026-09-19 审查 §三 正确性 · 前端 F2–F10）
- **F2 虚拟列表窗口不随 `entries` 重置 → 渲染 0 行空白表**：`scrollTop` 只由滚动事件赋值，切换目录/过滤后 `start` 仍取旧偏移，`entries.slice(start,end)` 为空，而空态占位因子项非空被抑制。现抽出共用的窗口化数学 `src/virtualList.ts`（`virtualWindow` + 单一 `ROW_HEIGHT`/`OVERSCAN`），`ObjectList` 与 `MigratePanel` 在 `entries` 变化时把 `scrollTop` 归零并同步写回真实 DOM（不再依赖浏览器钳制反复触发 scroll 事件收敛）。
- **F3 客户端不做批量分片，撞服务端硬上限后整批失败**：`useObjectActions` 一次提交全部选中 key（服务端上限 1000）、`MigratePanel` 同理（上限 10 000），而选中量可达 2 万–20 万——「加载全部 → 全选 → 删除」在 >1000 时返回英文 400，**一个都没删**。新增 `src/limits.ts`（`DELETE_MAX_KEYS_PER_REQUEST=1000` / `MIGRATE_MAX_KEYS_PER_REQUEST=10000` + `batchKeys`），两处改为**分片串行提交并聚合**：单片传输失败计入该片但不中断后续分片，`deleted/failed/lastError` 汇总后按「全部成功 / 部分成功 / 整体失败」分别提示（新增 i18n 键 `objects.toastDeletePartial`），选中态在 `finally` 中复位。
- **F4 复制文件夹忽略 `truncated`**：>10 万对象时静默只复制 10 万却提示成功（同一弹窗的删除分支有检查，属复制粘贴分叉）。现在按 `start.truncated` 提示「仅复制一部分」（i18n 键 `dest.toastCopiedTruncated`）。
- **F5 `loadAllSourceObjects` 在分页循环内读实时 `sourcePrefix`**：续页 token 会串到新前缀上，列表混两个前缀。现在进入循环时快照 bucket/prefix，并以 `listGen` 代次守卫丢弃过期响应（成功/失败/finally 三处）。
- **F6 迁移 SSE 无空闲超时**：`reader.read()` 可永久挂起 → EOF 后的补偿轮询根本不会启动 → 按钮永久禁用。现在 45s 空闲超时（任何数据/心跳到达即重置），超时经既有 `onError` 上报并主动 `cancel` reader + `abort` 释放连接；EOF 补偿轮询语义不变。（未加总时长上限：空闲超时已消除永久挂起，硬上限会误杀仍在正常推进的长迁移。）
- **F7 复制分享链接对全部选中并发 presign**：5000 选中 = 5000 并发请求。现在复用 `batchMetadata` 的 4 路有界池（`boundedPool` + `BATCH_META_CONCURRENCY`）。
- **F8 `AccountsPanel.load` 无 try/catch**：后端不可用时 unhandled rejection（健康轮询正是为此设计）。现在捕获并渲染带「重试」按钮的错误条（i18n 键 `accounts.loadFailed`）。
- **F9 四处可靠性缺陷**：① `useHealthPoll` 改为代次 + `disposed` 守卫，`stop()`/卸载后在途探测不再续跑定时器；② `ObjectList` 的 ResizeObserver 改为 `watch(scrollEl)` 挂载（首屏是骨架屏，原 `onMounted` 路径下 `viewportH` 恒 480），与 `MigratePanel` 做法对齐；③ 行高统一为 **42px**（`virtualList.ts` 常量与 `.tbl-virtual .v-row` CSS 同源，此前常量 38 低于真实行高导致窗口与垫片错位）；④ `theme.ts`（模块级调用）/`store.ts`/`useObjectBrowser.ts` 的 `localStorage` 读写全部加 try/catch 降级（隐私模式/配额异常不再抛错，`theme.ts` 抛出会导致整站启动失败）。
- **F10 `request()/requestResponse()` 的 headers 覆盖陷阱**：`fetch(url, { headers, ...opts })` 中 `opts` 在后，调用方一旦传 `headers` 就会把合并好的 `Authorization`/`Content-Type` 整体覆盖掉。现改为先合成默认 headers 再 `Object.assign` 调用方 headers（同名以调用方为准，默认值不丢）。
- **门禁实跑**：后端 `go vet` / `golangci-lint run ./...` **0 issues** / `gofmt -l` 干净 / 8/8 包 `-race` 通过且**每包 100.0% 语句覆盖**（`awk '$NF==0'` 零块）；前端 `pnpm lint`（0 警告）/ `pnpm typecheck` / `pnpm test:coverage`（66 文件 / 1039 用例，四项覆盖率 100%：statements 4074、branches 2844、functions 1095、lines 3503）/ `pnpm build`（360.52 kB / gzip 110.85 kB）全绿；**真对端 RustFS E2E 4/4 通过**（`S3CLINET_E2E=1`，覆盖 12 MiB 分段组装、批量删除、桶配置、回收站 purge）。

### 修复（2026-09-19 审查 P0 清零）
- **P0 SSE 自旋：`interrupted` 任务的 `/events` 流现在收到终态帧后立即关闭**。`migrate_async.go` 的 SSE 循环用 `case p := <-ch:`（缺 `ok` 判断），而 `Job.Subscribe` 对已结束任务「推一帧后立即 close」——**已关闭的 channel 永远就绪**，于是终态帧之后死循环刷零值帧（实测 3 秒吐 37 MB 且流永不结束）；终态判定又只认 `done`/`cancelled`，漏掉了同属终态的 `interrupted`（重启恢复标记）。现改为 `case p, open := <-ch: if !open { return }` + `service.IsTerminalJobStatus(p.Status)`；并给 SSE 路由补上 `withStreamLimit`（此前唯一未受并发保护的流式端点），`Job.Subscribe` 增加**每任务订阅上限 16**（`Finish` 对每个订阅者都要投递，无上限会让终态关闭的最坏耗时线性增长），饱和时端点回 503。回归测试 `TestMigrateJobEventsInterruptedStreamCloses`（用真实 `jobs.json` 走恢复路径造 interrupted 任务，断言只收到 1 帧且流 EOF）与 `TestMigrateJobEventsSubscriberCap`；前者已用「还原旧循环」变异验证会红灯。
- **P0 前端白屏：`s3c.servers` 被写坏不再导致整站白屏**。`readServers` 直接把 localStorage 解析结果当可信数组用（`rawList.map((s) => s.id ?? '')`），`[null]` 直接抛 `TypeError: Cannot read properties of null (reading 'id')`；而 `App.vue` 在**渲染期**调用 `api.getActiveServer()`，异常打断渲染 → 白屏，连唯一能修复存储的 Server 面板也不可达。`[1]`/`["x"]`/`[{}]` 不抛错但会生成 `id:''` 的幽灵 server 且不修复存储；`[{"base":{}}]` 会在 `applyProfile` 抛 `replace is not a function`。现新增 `sanitizeServer()` 逐元素做形状校验（非对象 / 无 id 一律丢弃并回写修复后的清单），`getActiveServer()` 兜底为渲染期安全访问器（存储被禁用时返回 undefined 而非抛出）。测试：`api.test.ts` 覆盖 5 种坏存储 + 混合条目，新增 `App.corruptStorage.test.ts` 用**真实** `./api` 模块挂载真实 App 断言三种坏存储下 header/侧边栏正常渲染（白屏即挂不上 header）。原「条目只有 token 时补空串成 id:''」的用例改判为「丢弃并修复」——幽灵 server 正是本缺陷的一部分。
- **P0 增量同步永不收敛：`SyncKeys` 的目标 key 映射与比对口径统一**。过滤阶段用相对 key 判定（`dstPrefix + stripPrefix(so.Key, srcPrefix)`），复制阶段却把**完整源 key** 交给 `MigrateKeys`（后者裸拼接 `targetPrefix + k`）——源 `p/a.txt`、`dstPrefix=q/` 实际写到 `q/p/a.txt`，比较仍在看 `q/a.txt`，于是二次同步依旧判定缺失，重复 10 万次也不收敛。现抽出复制内核 `migrateKeys(..., dstKeyFor func(string) string, ...)`，`MigrateKeys` 与 `SyncKeys` 共用；`SyncKeys` 只保留**一个** `dstKeyFor` 表达式，比对与复制不可能再漂移。附带修 `stripPrefix` 的段边界：前缀 `p` 不再命中 `prefix/x.txt` 并削成 `refix/x.txt`（该 key 在目标侧永不存在，同样导致反复重拷）。测试：`TestSync_PrefixMappingConverges`（目标 key 正确 + 二次同步 `copied:0`）、`TestSync_PrefixSegmentBoundary`；handler 侧 `TestMigrateSync_PrefixFilter` 增加收敛断言。**行为变更**：`sourcePrefix` 非空时目标 key 现在是「targetPrefix + 相对路径」（此前是 targetPrefix + 完整源 key）——这正是 `docs/api.md` 对 `copy-prefix` 已声明的「targetPrefix 直接前置到相对 key」语义。
- **P0 CI docker job 结构性失败：`build-push-action` 补 `load: true`**。`setup-buildx-action` 默认 `driver=docker-container`，镜像只留在 builder 缓存；而 Trivy 通过 `docker.sock` 扫的是**本地 daemon**，因此该 job 在 GitHub 侧不可能通过（GitLab 侧用原生 `docker build`，故一直正常——两侧门禁并不等价）。现补 `load: true`，并新增 `docker image inspect s3clinet/server:ci` 前置步骤，把「镜像没进 daemon」这一结构性失败与「真的扫出漏洞」区分开。已用本地 `docker-container` builder 复现：不加 `--load` 时构建 exit 0 但 `docker image inspect` 失败，加 `--load` 后成功。
- **P0 OpenAPI 契约幻影字段清零**：注册表 + `docs/api.md` 声明、handler 的请求结构体却**不含**的字段共 6 个——`POST /api/accounts` 的 `provider`、`POST …/rename` 的 `replaceTags`、`POST …/set-headers` 的 `cacheControl`/`contentEnc`/`contentLang`/`disposition`、`POST …/multipart/init` 的 `metadata`。因 `readJSON` 用 `DisallowUnknownFields`，**按文档示例体调用必然 400**；`set-headers` 更严重——文档承诺的「编辑 Cache-Control / Content-Language / Content-Encoding / Content-Disposition」这个功能并不存在（前端 `HeadersDialog` 也只发 contentType/metadata）。三处均无实现意图，故从注册表与文档删除；同时补上真漏记的 `useSSL`（前端 `AccountInput` 一直在发，只是没被文档声明）。`TestOpenAPI_ContractRequestBodyMatchesHandlers` 新增第 8 项：这四个端点的注册表字段集必须与 handler 结构体字段集**完全相等**（两端任一方向漂移都会变红，含 `useSSL` 缺失这类反向漏记）。

### 重构（前端 API 模块拆分）
- **`src/api.ts`（838 行单文件）拆分为 `src/api/` 目录 7 个模块**：`storage.ts`（浏览器凭据 / 多服务器 profile，339 行）、`endpoints.ts`（`s3api` 领域端点，192）、`jobs.ts`（异步任务 SSE + EOF 状态回读，119）、`download.ts`（ZIP 流式落盘，85）、`index.ts`（公开面 barrel，82）、`upload.ts`（预签名直传，48）、`http.ts`（传输层，38）。原文件把凭据存储、传输、~70 个领域端点、SSE、XHR 上传混在一处，内聚性已失。模块间为单向依赖 `index → {endpoints, jobs, download, upload} → http → storage`，无环；`./api` 与 `../api` 仍解析到 `api/index.ts`，**全部既有 import 路径与对外契约不变**。顺带消除 `migrateJobStatus` 的重复定义（拆分时 endpoints 与 jobs 各写一份 URL 与返回类型），统一由 `jobs.ts` 实现、`s3api` 引用同一函数。本次为**纯搬运**：`fetch` 的 headers 合并顺序等既有语义原样保留（`docs/archive/review-2026-09-19.md` §F10 的潜伏缺陷不混入本次改动）。验证：`vue-tsc` / `eslint` 通过、986 例（64 文件）单测全绿、覆盖率四指标 100%（statements 3941 / branches 2787 / functions 1082 / lines 3392，7 个新模块全部满覆盖）。

### 新增（前端死代码门禁）
- **`apps/web/src/deadcode_gate.test.ts`：API 公开面「导出后无人引用」门禁**（对应后端 `apps/server/deadcode_gate_test.go` 的前端半边）。`s3api.*` / `api.*` / 具名导出的每个成员必须在**至少一个非测试源文件**中被引用——eslint 与 `vue-tsc` 只报未使用的**局部变量**，看不见导出符号，而测试文件的引用会让死代码一直"活着"（真实事故见下条移除项）。实现要点：用 `import.meta.glob(?raw)` 读源码（不引入 `node:fs` / `@types/node`，维持前端依赖最小化），按 **import 别名精确匹配 `别名.成员`**，不做裸词或子串匹配——避免 `'migrate'` 这类字符串字面量与 `migrateAsync` 这类前缀碰撞造成假绿。三重自检防"空跑变绿"（扫描文件数 ≥50、公开面解析 ≥50、引用总数 ≥30）+ `INTENTIONAL_UNUSED` 豁免清单（附"豁免不得腐烂"检查）。**变异验证**：注入 `s3api.zzDeadProbe` → 红灯；再复刻历史缺陷（`s3api.copyFiles` 仅被 `api.test.ts` 调用）→ **仍红灯**，即"只被测试引用"不被算作使用。

### 移除（前端死 API 与生产测试接缝）
- **删除 5 个零生产调用的 `s3api` 方法**：`copyFiles` / `deletePrefix` / `copyPrefix` / `migrate` / `migrateSync`——全仓仅 `api.test.ts` 引用，属"被测试引用掩盖的死代码"。对应后端端点保留，仅前端 UI 未使用。同时删除 2 个零生产引用的 barrel 导出：`requestResponse`（仅 `download.ts` 内部使用）、`downloadZipToDisk`（生产只走 `s3api.downloadZipToDisk`）；相关测试改为从 `./api/http`、`./api/download` 直接导入。
- **生产包不再携带测试专用接缝**：`s3wrap.ResetMetrics`（导出且注释自称"仅测试使用"）移入 `metrics_test.go` 内的 `resetMetrics()`，不再进入生产二进制；`service.SetMaxJobsForTest` 删除，在册任务上限由**可变全局** `var maxJobs` 改为**每实例**构造期选项 `NewJobRegistry(WithMaxJobs(n))`——原写法会跨用例、跨注册表实例相互污染（并发测试下即数据竞争），handler 侧接缝移到 `internal/handler/export_test.go`（仅 `go test` 时编译），并以行为用例（上限耗尽 → `ErrTooManyJobs`、非正上限回落默认值）替换原「钩子返回旧值」用例。

### 测试与质量门禁（2026-09-19 浏览器 E2E 空转用例）
- **消除 Playwright 的 2 个「空转」用例：14 passed / 1 skipped → 15 passed / 0 skipped**。「账号管理：空状态 → 新增 → 列表」用 `getByRole('link')` 找侧边栏入口，而 `App.vue` 的导航渲染为 `<button>`（实测 link 命中 0 / button 命中 1）→ `isVisible()` 恒为 false → `test.skip(true, '侧边栏没有账号入口（可能是 Tauri-only 视图）')`。**它不是 Tauri-only 场景，是定位器写错被 skip 掩盖**：该用例的空状态断言 `/暂无|empty|no account/i` 与中英两版真实文案均不匹配（实测 `false`），即真正执行也会失败——说明它从未跑过。现改为 role=button 硬断言入口可见（找不到即失败，不再 skip）+ 校验真实空态文案 + 走完「新增登录 → 填表 → 保存」，断言表单字段进入 POST 请求体、新增行渲染、弹窗关闭。第二个用例原本只 `page.evaluate(fetch)` 一次并断言 fixture 自己写死的 `version`（前端完全不参与，app 全坏也会绿），现改为断言用户可见的恢复行为：`/api/accounts` 先 500 → 错误横幅 + 「连接异常」徽标，后端恢复后 `useHealthPoll` 自动重拉账号并清除错误态。两个用例均经变异验证（还原旧定位器 / 打断 `onRecover` 都会红灯）。

### 修复（OpenAPI 契约字段级漂移）
- **字段级契约门禁 + 三处客户端可见的注册表失真**：新增 `TestAPIDocDocumentsRequestBodyFields`——OpenAPI 每个 `requestBody` 的字段名必须出现在该端点 `docs/api.md` 正文（支持「请求体与 `POST /api/x` 相同」别名）。门禁上线即抓到：`POST …/mkdir` 注册表写 `prefix`（handler / 前端 / 文档一直用 `key`）、`POST …/copy-objects` 与 `/async` 写 `items`（真实字段是 `keys`，异步侧此前只声明空对象）、`DELETE /api/accounts/{id}/version` 把 query 参数误声明为 requestBody。按 OpenAPI 生成的客户端用这些字段会被 `DisallowUnknownFields` 直接 400 / 收不到参数。已修注册表并加回归断言（`openapi_contract_test.go` 第 7 项 + 新增 `requestQueryParams` 助手），并新增通用输入源门禁 `TestOpenAPIRequestDeclarationMatchesHandlerInput`（routes.go → handler 调用闭包找 `readJSON` ⇔ 注册表 `Request:` 双向比对，含 `withStreamLimit` 包装与委托解码；经变异测试验证会红灯）；`docs/api.md` 补齐 `provider` / `publicEndpoint` / `replaceTags` / `cacheControl` / `contentEnc` / `contentLang` / `disposition` / `metadata` 等漏记字段。
- **消音式死代码门禁**：`golangci-lint`（`run.tests: true`）实测**能**报未引用的测试函数/方法，盲区只有显式消音 `_ = ident` / `var _ = expr`。新增 `apps/server/deadcode_gate_test.go` 扫全仓 Go 源码禁止这两种形状（放行 `_ = f()` 这类显式忽略返回值），并自检扫描文件数避免路径写错静默变绿；本轮清掉 11 处遗留（含 `var _ = errors.New // 保持 errors 引用位`、只声明不用的 `deletes`、`PurgeObject` 里的冗余 `_ = keyMarker`）。

### 新增（可靠性与安全）
- **R8 完善：存储硬失败可观测 + SSRF 生效策略可见**：`/api/metrics` 新增 `s3c_store_up`（store 掉线为 0，与既有 `/api/health` 503 一起构成 ADR-002 的可告警面）与 `s3c_ssrf_deny_private`（反映 `S3C_SSRF_DENY_PRIVATE` 生效值）；启动日志新增 `ssrfDenyPrivate` 字段；新增 `s3wrap.DenyPrivateNetworks()` getter。测试 `TestMetricsStoreUpAndSSRFPolicy` 覆盖两个 gauge 的 1/0 两态。文档同步 [api.md](docs/api.md) 指标清单、[DEPLOYMENT.md](docs/DEPLOYMENT.md) §6.1（503 处置顺序 + 告警建议）与 §6.3、[threat-model.md](docs/threat-model.md) 边界 D；`ROADMAP.md` §5.1 把 R8 移入「已决策接受（ADR 兜底）」索引，登记表只剩 R5（唯一开放项）。
- **DataDir 单写者锁**：文件型 store（json/sqlite）+ 内存 JobRegistry 只支持单副本，此前只靠文档约束。新增 `store.AcquireDataDirLock`（unix `flock(LOCK_EX|LOCK_NB)` + `<DataDir>/.s3clinet.lock`），第二个实例启动即失败返回 1，内核在进程退出时释放锁（无陈旧锁文件）。测试覆盖互斥、释放后重加、目录不可建 / 锁文件被占、`runServer` 端到端拒绝启动；`GOOS=windows` 构建通过（非 unix 为文档化 no-op）。
- **`S3C_SSRF_DENY_PRIVATE` 可选 SSRF 加固**：置 `1` 后创建期与拨号期校验连 RFC1918 / ULA / 回环 / 未指定地址一并拒绝；默认关闭，保持 [ADR-003](docs/decisions/0003-ssrf-private-allow.md) 的自托管主场景（MinIO / RustFS / 局域网放行），ADR-003 增加 Update 段。测试：`TestDenyPrivateNetworksOptIn` / `TestDenyPrivateNetworksDialGuard` / `TestFromEnvSSRFDenyPrivate`。文档同步 `.env.example`（根 + server）、README 配置表、[DEPLOYMENT.md](docs/DEPLOYMENT.md) §2.1、[threat-model.md](docs/threat-model.md) 边界 D、[architecture.md](docs/architecture.md) 关键机制 / 取舍、[decisions/index.md](docs/decisions/index.md)。
- **RustSec 审计入 CI**：`cargo audit`（pin `cargo-audit 0.22.2`）加入 GitHub `ci.yml` 与 GitLab `desktop` job，新增本地 `make rust-audit`。实跑 **0 漏洞**；7 条 unmaintained / unsound 告警（`proc-macro-error`、5 个 `unic-*`、`glib 0.18.5`）逐条 triage，均为上游无修复版本的传递依赖，不用 ignore 清单掩盖。
- **分段上传缺 ETag 的失败路径可见 + 多厂商兼容矩阵**：新增 `upload.test.ts` 用例断言「2xx 但读不到 ETag → 报错并在组装前 abort 清理分段」；README 补 RustFS / MinIO / AWS S3 / 阿里 OSS / 腾讯 COS 的 CORS `ExposeHeader: ETag` 要求矩阵（RustFS 为唯一自动化真对端 E2E）。

### 变更（安全：明文落盘告警）
- **`json` / `sqlite` 驱动空 `S3C_STORE_KEY` 时启动打 WARN 告警**：此前 base compose 默认 `sqlite` + 空 key，`secretKey` 明文落盘却只在文档里提示，运行时没有任何信号。新增 `config.Config.StorePlaintextWarning()`（返回可执行文案：驱动名 / `DataDir` / 改用 `encrypted` 或设 ≥16 字符的 `S3C_STORE_KEY`），`runServer` 在配置校验后 `logger.Warn` 输出；`encrypted` 或非空 key 不告警。测试：`TestStorePlaintextWarning` 六例表驱动（先失败后实现）+ `TestMainServerWarnsPlaintextStore` 子进程断言启动日志确实出现该告警。文档同步 [DEPLOYMENT.md](docs/DEPLOYMENT.md) §2.1、[threat-model.md](docs/threat-model.md) 边界 C、[ROADMAP.md](docs/ROADMAP.md) §5.1 R3（🟡 → 🟢）。
- **旧 `roadmap #N` 引用清零（27 处 / 25 文件）**：2026-09-17 路线图收口后，代码注释仍指向已作废的旧编号。带 `ASSESSMENT` 编号的只保留该编号，纯 roadmap 编号改为「已闭环：FEATURES.md §M」，`docker-compose.yml` 的密钥加密说明改指 §M + §5.1 R3；并给 [ROADMAP.md](docs/ROADMAP.md) §六 增加第 6 条编号引用规则。涉及 `apps/server`（config / store / handler / s3wrap）与 `apps/web`（App.vue、useHealthPoll、useBucketSetting、vite.config.ts 等）。
- **桌面端分发与签名正式立项（暂不处理）**：登记为 [todolist.md](docs/KNOWN_ISSUES.md) #25，阻塞依赖为 Windows 代码签名证书 / Apple Developer ID + 公证（[ROADMAP.md](docs/ROADMAP.md) §5.2 E6）。

### 文档（路线图风险登记）
- **风险登记只留「还需要人看的项」**：R1/R2/R3/R4/R6/R7 已收敛为自动化门禁，从 §5.1 登记表移入同节末尾的「已收敛」索引（编号不重排，todolist #25 / §三 #1 / ADR / 代码注释的引用保持有效）；§5.1 现在只剩 R5（桌面签名，🟡 开放）与 R8（已决策接受，➖）。§5 说明、§四 守卫段与 §六 维护约定同步改写。
- **`docs/ROADMAP.md` §五 重构为「风险登记表 + 依赖清单」**：原表 6 行中 5 行写的是**已闭环**的缓解措施（与本文件「只列未完成项」的约定冲突，历史证据本应归 `FEATURES.md`），且标题含「依赖」却没有依赖内容。现拆为 §5.1 风险登记（新增 `R*` 编号、**可观测触发信号**、等级、状态图例 🟢 有门禁 / 🟡 开放 / ➖ 已决策接受、守卫与跟踪）与 §5.2 依赖清单（Go 工具链、AWS SDK v2、modernc sqlite、RustFS 镜像、actions pin SHA、签名证书、GitHub Release 通道、S3 厂商 CORS/ETag、S3C2/S3C3 兼容承诺），并补「复审规则」。同时补登此前缺失的开放风险：base compose 明文密钥（R3）、单副本假设（R4）、桌面未签名且无自动更新（R5）、上游 ETag/CORS 差异（R7）、Rust 依赖审计缺口（R6）。另修正编号说明：代码注释里的 `roadmap #N` 指 2026-09-17 收口前的**旧编号**，不可按当前编号回读。
- **`docs/ROADMAP.md` 全文与新 §五 对齐**：§二 澄清「依赖」列仅指里程碑前置关系并把外部依赖指向 §5.2；§三 #1 补 `R5` / `E6` 交叉引用；§四 标明本表即 §5.1 中 🟢 风险的生效守卫；§六 新增第 6 条「编号引用」规则（新引用必须带 `§三 #N` / `R*` / `E*` 前缀）。[`docs/threat-model.md`](docs/threat-model.md) 边界 C 的旧 `roadmap #2` 引用改为指向 `FEATURES.md` §M 与 §5.1 R3（残留风险：base compose 仍可空 `S3C_STORE_KEY`）。

### 变更（本地 CI 体验）
- **gitlab-ci-local 本地跑法收口**：提交 [`.gitlab-ci-local-env`](.gitlab-ci-local-env) 默认挂 `/var/run/docker.sock`（对齐正式 runner，免每次 `--volume`）；`Makefile` 增加 `gcl` / `gcl-list` / `gcl-docker` 并 pin `gitlab-ci-local@4.75.1`；新增 [`.gitlab-ci-local-variables.yml.example`](.gitlab-ci-local-variables.yml.example) 说明国内可覆盖的 `GOPROXY` / `NPM_REGISTRY` / `TRIVY_DB_REPOSITORY`（复制为 gitignore 的 `.gitlab-ci-local-variables.yml` 即生效）。Trivy 默认 DB（`mirror.gcr.io`）超时无需改 script——设 `TRIVY_DB_REPOSITORY=ghcr.io/aquasecurity/trivy-db:2` 即可。文档入口：[`docs/DEVELOPMENT.md`](docs/DEVELOPMENT.md) §3、README、`.gitlab-ci.yml` 头部注释。

### 变更（目录结构）
- **monorepo 布局：`server/` / `web/` / `desktop/` 统一迁入 `apps/`**：三个应用此前平铺在仓库根目录，与 `docs/` / `deploy/` / `scripts/` 混在同一层，根目录随应用增多持续膨胀。现统一为 `apps/server`、`apps/web`、`apps/desktop`（一律 `git mv`，保留历史）。Go module 路径随之由 `github.com/weilai1949/s3clinet/server` 改为 `github.com/weilai1949/s3clinet/apps/server`（84 个文件、121 处 import 同步改写）。构建与运行链路全部对齐：`Makefile`、`apps/server/Dockerfile`（构建上下文仍为仓库根，`COPY apps/...`）、三套 compose、`.dockerignore`、`.gitignore`、`scripts/*.sh`（`run-dev` / `graceful-restart` / `release-version` / `nginx-local`）、`.githooks/pre-commit`、4 个 GitHub workflow、`.gitlab-ci.yml`。Tauri 的 `tauri.conf.json` 用相对路径（`../web`、`../../web/dist`），随目录整体移动后无需改动即保持有效。
- **修复迁移后 `S3C_STATIC_DIR` 默认值失效**：默认值原为 `./web/dist`，是相对**进程工作目录**解析的。所有文档化的启动方式（`make server`、README 的 `cd apps/server && go run .`）都以 `apps/server` 为 CWD，迁移后该默认值会解析到不存在的 `apps/server/web/dist`，静态托管静默 404。已改为 `../web/dist`，并同步 `apps/server/.env.example` 与 README 配置表说明（先写失败断言再改实现）。

### 移除（不留尾巴）
- **`acc_helper_test.go` 中一整套失效的 S3 错误注入机制**：`accFailInjector.set` 是 `rules` 的**唯一写入方**，却从未被任何测试调用——于是 `rules` 恒为空 map、`take` 恒返回 false，`accErrXML` 与 `accSettingsFake` 也随之不可达（`accSettingsFake` 本身也无调用点）。更关键的是它**被覆盖率掩盖**：测试文件不参与 instrumentation，100% 门禁只看生产代码，所以这段死代码在门禁全绿的情况下长期存在。桶级配置（encryption/cors/website/policy/tagging）已由后写的 `bucket_settings_test.go` 用自带假 S3 完整覆盖，注入器属于被取代的遗留物。已删除 `accS3Fail` / `set` / `accErrXML` / `accSettingsOp` / `accSettingsFake` 与 4 个仅它使用的 XML 常量，`accBucketsFake` 去掉无人可填的 `inj` 参数（改为 `accBucketsFake()`），`newAccFailInjector` 一并移除。
- **前端 lint 门禁由 `--max-warnings 50` 收紧为 `--max-warnings 0`**：实测 `eslint src` 当前为 **0 errors / 0 warnings**，`50` 的阈值等于给后续改动留了 50 条告警的沉默额度，与新增的「机械检查零告警」规则冲突。收紧后 `pnpm lint` 与 `pnpm typecheck`（`vue-tsc --noEmit`）均通过。
- **测试代码中的 12 处 `_ = x` 消音与假断言**：按新增的「死代码零容忍」硬约束（见 `AGENTS.md` 第 5 条）清理——`gaps_test.go` 的 `_ = res`（`res` 已在上一行 `defer res.Body.Close()` 使用）、`ctx2`/`cancel2`（未使用，改用 `func() {}`）、`_ = r`（`r` 下一行即用）、`_ = in`（循环变量被丢弃，改为真正断言每个输入）、`sub`/`release`（订阅与信号通道均未被消费）、`job_persist_test.go` 的 `_ = job`、`cover_extra_test.go` 的 `_ = newCancelReader(...)`（Go 允许直接丢弃返回值）、`stream_copy_test.go` 的 `var _ = StreamCopy`（编译期形状断言，对已被常量断言覆盖的符号是冗余）、`metadata_test.go` 的 `_ = httptest.NewRecorder // 兼容 import 暂未使用`（该文件已用 `httptest` 7 次，纯属死 shim）、`acc_core_test.go` 的 `_ = up.String() // 触发 uptime 闭包`（改为断言非空）。这些模式 `golangci-lint` **看不见**（`_ =` 正是它的消音手段），只能靠人工审查与显式规则拦住。
- **26 个从未被引用的 i18n 死键**（`app.name`、`common.back` / `confirm` / `danger` / `default` / `done` / `empty` / `failed` / `none` / `search` / `selected` / `test` / `upload` / `waiting`、`objects.bucket` / `prefix`、`toolbar.delete` / `downloadZip` / `filter` / `refresh`、`trash.empty`、`upload.queue`、`server.save`、`batchEdit.aclNoChange` / `storageNoChange`、`bucketTags.errEmptyKey`）：中英各 26 行，字典键数 695 → 669。此前只校验「被引用 → 有定义」，「有定义 → 被引用」无人校验，而 i18n 目录又被排除在覆盖率统计外，死键因此长期堆积（2026-09-16 评估 D7）。现于 `apps/web/src/i18n/coverage.test.ts` 增加**反向门禁**：字典中任何既非字面量引用、也不匹配动态拼接模式的键都会让测试变红。动态键（`` `provider.${p}.label` `` 等 4 处）按「字面命名空间前缀 + 插值」形状生成豁免正则，避免把 `/api/accounts/${id}` 这类非 i18n 模板串误当豁免而让门禁恒真。
- **`@vitest/coverage-v8` 依赖**：`vite.config.ts` 的覆盖率 provider 是 `istanbul`，v8 provider 从未启用，属重复依赖。已从 `apps/web/package.json` 移除并重算 `pnpm-lock.yaml`（`pnpm install --frozen-lockfile` 仍通过；锁文件里残留的 `coverage-v8` 条目仅为 vitest 的 optional peer 声明，不再安装）。覆盖率仍为 100%（statements 3883/3883、branches 2769/2769、functions 1059/1059、lines 3339/3339）。<sup>†</sup>
  > <sup>†</sup> 上述四项计数是**该条目撰写当时的实测值**，后续用例增加后已变化（2026-09-21 实测：statements 4074 / branches 2844 / functions 1095 / lines 3503）。按本文件「不追溯篡改已发布区」的纪律保留原值并加注，不回写（docs/archive/review-2026-09-19.md §7.3 D10）。
- **`ObjectList.vue` 的 `visibleCount` prop**：父组件 `ObjectsPanel.vue` 传入、组件内从不读取（空态判断实际用 `totalCount`），属迁移遗留的无效契约。已连同父组件绑定与测试 fixture 一并删除。`ObjectToolbar` 的同名 prop **保留**——它真实用于 `{{ visibleCount }}/{{ totalCount }}` 计数展示。
- **`.gitignore` 由 202 行精简至 81 行**（其中非注释的有效规则 45 条）：删去 28 行与本站技术栈无关的脚手架模板条目（`bower_components`、`jspm_packages/`、`web_modules/`、`.next`、`out`、`.nuxt`、`.output`、`.parcel-cache`、`.svelte-kit/`、`.docusaurus`、`.serverless/`、`.fusebox/`、`.dynamodb/`、`.firebase/`、`.tern-port`、`.vscode-test`、`lib-cov`、`.grunt`、`.lock-wscript`、`build/Release`、`*.tgz`、`.yarn-integrity`、`.node_repl_history`、`.stylelintcache`、`.npm`、Gatsby / vuepress / vitepress 段等），并按**实测**（逐条注释掉该规则后 `git check-ignore` 复查）删除 4 条被通用规则覆盖的冗余项（`apps/server/data/` ← `data/`、`apps/web/node_modules/` 与 `apps/desktop/node_modules/` ← `node_modules/`、`apps/web/dist/` ← `dist/`）。删除前后 `git status --ignored` 的忽略集合逐行比对一致，唯一差异是把过窄的 `.cache/` 收敛为 `.cargo/`（前者会顺带忽略任意层级的 `.cache/`，后者才是桌面端构建真实产生的目录）。已确认无任何**已跟踪**文件落入新规则。
- **`apps/web/src/styles.css` 中 2 个零引用设计 token**：`--primary-h`（浅 `#0b8563` / 深 `#2fdca6`）与 `--warn-bg`（浅 `#fffbeb` / 深 `rgba(217,119,6,.12)`）在 `src/` 中**无任何 `var()` 消费**（全仓 grep 仅命中定义本身，且不存在 `setProperty` 或动态 `var(--${...})` 拼接）。二者是早期「主色 hover 加深」「警告底色」方案的遗留——现在主色 hover 靠 `box-shadow` + `translateY` 表达，警告仅用前景色 `--warn`（`PreviewOverlay.vue:112`），配套的 `.msg.warn` / `.tag.warn` 规则从未存在。已按两套主题**成对删除**，删后调色板仍是浅/深 1:1 镜像（43 个颜色 token 两侧一致），`--warn` 与 `--primary` 保留（各有消费方）。
- **`apps/web/public/logo.svg` 死资源**：与 `apps/web/src/assets/logo.svg` **字节相同**（`md5 fd267cb2…`），但只有后者被 `App.vue` 以 `import logoUrl from './assets/logo.svg'` 引用。前者属 Vite 的 `public/` 直拷目录，会被**原样复制进每次构建产物**（`dist/logo.svg`，942 B），而 `dist/index.html` 与全部 chunk 都不引用它——早期 `public/` 方案的遗留副本。已 `git rm`（`public/` 目录随之消失）；品牌图仍由 `src/assets/logo.svg` 经 Vite 内联为 data URI 正常渲染（重建后 `dist/` 不再出现游离的 `logo.svg`，`pnpm build` 通过）。
- **`apps/web/src/styles.css` 中失效的 `.popover*` 规则块**：设置弹层已改为整页 `ServerPanel.vue`，`popover-wrap` / `.popover`（含 `h4` / `.field + .field` / `.actions`）与 `.popover-backdrop` 共 6 条规则，以及媒体查询里的 `.popover` 宽度覆盖，在 `src/` 中**已无任何模板引用**（全仓 grep `popover` 仅命中这些定义本身）。已删除该块；同段的 `@keyframes pop-in` **保留**——它仍被 `ObjectContextMenu.vue:105` 复用（该组件测试 9 项通过）。


### 新增（2026-09-17 路线图迭代 #1–#8、#10、#11）
- **`docs/api.md` 自动化校验（roadmap #1 / todolist #8）**：新增 `apps/server/internal/handler/api_doc_test.go`，从 `docs/api.md` 解析 `METHOD /api/...` 行并与 `routes.go` 注册表做**双向 diff**——文档多写一个端点或漏写一个端点都会让 `go test ./...` 变红。此前 `docs/api.md` 与路由的同步完全靠人工维护，70 个端点里任何一个改名/新增都可能静默漂移。已注入两侧漂移实测变红后还原，当前 70 ↔ 70 完全一致。
- **S3C3 加密格式：KDF 参数写入文件头（roadmap #2 / todolist #16）**：S3C2 信封只存 `magic + salt`，Argon2 参数硬编码（`t=1`），因此直接调参会让既有加密库**永久无法解密**。新格式 `S3C3` 为 `magic(4) + time(4,BE) + memory(4,BE) + threads(1) + salt(16) + ciphertext`，读取时按**文件头里的参数**派生密钥，从而可以在不影响旧库的前提下逐步加强。`argonTimeV3 = 2`（OWASP 建议 Argon2id time cost ≥ 2）。**双版本读取**：`parseEnvelope` 同时识别 S3C2（沿用 `legacyParams = {1, 64MiB, 4}`）与 S3C3；S3C3 头部参数为 0 视为损坏并报错（不静默降级为弱密钥）。新写入一律 S3C3。
- **SQLite 驱动 `secret_key` 列加密（roadmap #2 / ASSESSMENT M1）**：`SQLiteStore` 新增 `storeKey`，写入时 `encryptSecret` 以 S3C3 密文落盘、读取时 `decryptSecret` 解密；**历史明文行仍可读**（按魔数判别），写回时自动加密。库中已是密文但进程未配置 `S3C_STORE_KEY` 时显式报错，绝不把密文当明文返回。`config.MinStoreKeyLength = 16` 对 `S3C_STORE_KEY` 做最短长度校验（`ErrShortStoreKey`）。
- **安全审计日志（roadmap #3 / ASSESSMENT M3）**：新增 `apps/server/internal/handler/audit.go`，以稳定事件常量 + `h.audit(r, event, extra...)` 记录 `audit` / `ip` / `method` / `path`。覆盖：鉴权失败（401，含 `malformed` / `bad_token` 原因）、账号创建/更新/删除、桶策略设置/清除、对象删除、前缀删除、回收站清空、限速命中。
- **可信代理 XFF 解析（roadmap #3 / ASSESSMENT M5）**：新增配置 `S3C_TRUSTED_PROXIES`（默认空）。`clientIPWithProxies` **仅当直连对端命中白名单**时才采信 `X-Forwarded-For` 首段，否则一律回退 `RemoteAddr`。此前无条件信任 XFF，直连部署下任何客户端都能为每个请求伪造一个新 IP 绕过限速。
- **S3 上游调用指标（roadmap #5 / todolist #21）**：新增 `apps/server/internal/s3wrap/metrics.go`，用 smithy `Finalize` 中间件统一采集（覆盖全部 SDK 调用，无需逐方法埋点）：`s3c_s3_calls_total`、`s3c_s3_call_errors_total{code=...}`（API 错误码，非 API 错误归 `canceled` / `timeout` / `transport`，基数有界）、`s3c_s3_call_duration_seconds` 直方图（11 桶）、`s3c_s3_stream_bytes_total`。经 `/api/metrics` 输出。
- **ZIP 部分失败可见（roadmap #4 / ASSESSMENT S6）**：`zip.go` 不再丢弃 `service.WriteObjectsZip` 返回的 `failKeys`——部分失败落 Warn 日志（含失败 key 清单）并计入 `s3c_zip_partial_failures_total` / `s3c_zip_failed_keys_total`，整体失败计入 `s3c_zip_failed_total`。此前失败信息只写进包内 `_下载失败清单.txt`，服务端完全无法观测批量下载失败率。
- **前端后端健康轮询与自动恢复（roadmap #8 / todolist #22）**：新增 `apps/web/src/composables/useHealthPoll.ts`，后端不可用后每 5s 探测 `/api/health`，一旦恢复即回调 `loadAccounts` 重新拉取数据并清除错误横幅；未出错时不轮询（不做无谓请求）。新增 `api.health()`。
- **grid 视图渲染上限（roadmap #7 / ASSESSMENT D8）**：`ObjectList.vue` 的 grid 分支改为渲染 `gridItems`（上限 300 条），超出时显示截断提示（新增 i18n 键 `objects.gridTruncated`）。此前 grid 的 `v-for` 对万级条目全量渲染 DOM（列表视图早已窗口化）。
- **错误文案源码级门禁（roadmap #6 / todolist #23）**：新增 `apps/server/internal/handler/error_echo_gate_test.go`，扫描生产 `.go` 文件（剔除注释与 `_test.go`），禁止 `writeErr(..., StatusBadRequest, "..." +` 这类把用户输入拼进响应文案的写法。

### 修复（2026-09-17 路线图迭代）
- **客户端错误消息不再回显用户输入（roadmap #6 / ASSESSMENT L2）**：`headers.go` 把 `ValidateUserMetadata` 的 `err.Error()` 直接回传客户端，而该错误串含用户提交的 metadata key（形如 `key %q length %d > %d`）。现改为固定文案 `invalid user metadata` + 服务端 `Debug` 日志（记录 bucket / key）。同类问题一并修复：`metadata.go` 的 `unsupported acl: <值>` → `unsupported acl`、`duplicate tag key`、`duplicate rule id`；`objects.go` 的 `unsupported storageClass`；`multipart.go` 的 `duplicate partNumber`。`TestSetHeadersInvalidUserMetadata400` 增加 `echo` 断言，确保响应体不含用户输入（含中文与超长 key/value）。
- **`useBucketSetting.reload()` 竞态守卫（roadmap #8 / ASSESSMENT S8）**：快速切桶或保存后刷新会让多个 reload 并发，旧请求的响应/失败会覆盖新请求的状态。现加 `reloadSeq` 序号，只有最后一次发起的请求才允许写 `loading` 或上报错误。
- **`entries` / `visibleEntries` 重复排序（roadmap #10 / ASSESSMENT D9）**：`useObjectBrowser.ts` 抽出 `compareEntries`（文件夹恒在前、文件按当前列与方向），`entries` 一次排序到位，`visibleEntries` 只做过滤并保持顺序。此前过滤态下会先排文件夹、再排一次文件。

### 测试与质量门禁（2026-09-17 路线图迭代）
- **覆盖率门禁去「注水」：前端纳入 `src/i18n/index.ts`（roadmap #11 / todolist #12）**：`vite.config.ts` 不再整体排除 `src/i18n/**`，改为只排除纯数据模块 `src/i18n/messages/**`（其完整性由 `i18n/coverage.test.ts` 的键门禁保证）。纳入后 `index.ts` 的 `readLocale` 回退/异常、`setLocale` 写入失败、`cycleLocale`、`locale()`、`i18nKeyCount` 缺省参数等分支补齐了**行为测试**（非 gap 测试），四指标仍为 100%。
- **后端删除不可达防御分支，`count==0` 检查归零**：`SQLiteStore.encryptSecret` 的 AES 加密错误分支（`deriveKey` 恒返回 32 字节合法密钥）与 `openSQLite` 中冗余的 `os.MkdirAll` 判断属确实不可达的防御代码，已删除而非写测试凑覆盖；`registerMiddlewares` 从内联闭包抽为命名函数以便直接测试锚点缺失时的错误上抛。`make test-cover` 的 `count==0` 扫描现无任何输出，8 个包语句覆盖率均 100%。
- **新增前端测试**：`useHealthPoll.test.ts`（轮询节奏 / 恢复回调 / start 幂等 / 卸载停止）、`useBucketSetting` 竞态守卫用例、`ObjectList` grid 截断用例、`App.vue` 恢复路径集成用例、`api.health()` 用例、i18n 分支用例。前端 63 文件 / 983 测试全绿。<sup>†</sup>
  > <sup>†</sup> 「63 文件 / 983 测试」是**该条目撰写当时的实测值**；后续用例增加后已变化（2026-09-21 实测：66 文件 / 1039 用例）。按本文件「不追溯篡改已发布区」的纪律保留原值并加注，不回写（docs/archive/review-2026-09-19.md §7.3 D10）。

### 文档（2026-09-17 路线图迭代）
- **路线图 v1.0.0 / v1.0.x / v1.1.0 收口（roadmap #1–#8、#10、#11）**：三个里程碑的开放条目全部完成并从 `docs/ROADMAP.md` 移除（长期项重编号为 #1–#3），`docs/todolist.md` 五个分类均归零，完成证据归档至 `docs/FEATURES.md` §M。`docs/api.md` 补充 `/api/metrics` 的 S3 上游指标与 ZIP 失败指标说明、ZIP 端点部分失败可观测说明。`README.md` / `.env.example` / `apps/server/.env.example` / `docs/DEPLOYMENT.md` 补充 `S3C_STORE_KEY`（≥16）与 `S3C_TRUSTED_PROXIES`；`docs/threat-model.md` 边界 C 表与已知风险同步更新。

### 新增
- **GitLab CI（`.gitlab-ci.yml`），与 GitHub Actions 同门禁**：把 `.github/workflows/` 的 `ci.yml`、`e2e.yml`、`e2e-playwright.yml` 逐 job 镜像为 `server` / `web` / `docker` / `desktop` / `desktop-build`（`when: manual`）与 `rustfs-e2e` / `playwright-e2e`，命令与阈值完全一致（gofmt、`go vet`、govulncheck v1.8.0、golangci-lint v2.13.2、`go test -race` + 覆盖率 100%、`pnpm lint/typecheck/test:coverage/build`、`docker build` + Trivy CRITICAL/HIGH、`cargo check --locked`、RustFS 真对端 E2E、Playwright chromium E2E）。触发规则对应 GitHub 的 push（main/develop）+ pull_request + 手动 + 定时。`release-desktop.yml` **有意不镜像**——它发布到 GitHub Release（tauri-action + `gh release upload`）且需 Windows/macOS runner 与 `GITHUB_TOKEN`，属发版设计而非 CI 一致性。`rustfs-e2e` 用 GitLab service 容器替代 compose 起对端，镜像/端口/凭据不变，并照搬 GitHub 的 `/health` 轮询等待（正式 runner 的 service healthcheck 只认镜像自带 HEALTHCHECK，rustfs 镜像没有）。新增 `.gitlab-ci-local/` 到 `.gitignore`（本地执行状态目录）。本地无 GitLab 实例即可用 `npx --yes gitlab-ci-local` 跑真实 job；两侧对照表与执行器差异（tracked-only 同步、`docker` job 需挂宿主 socket）记入 [`docs/DEVELOPMENT.md`](docs/DEVELOPMENT.md) §3。

### 修复
- **`server` job 的 16 条 golangci-lint 问题清零，GitHub 与 GitLab 两侧 `server` 门禁转绿**：此前 `server` job 在 `develop` 上连续失败，原因不是代码缺陷而是**门禁本身**——4 个 `unused` 指向已失效的注入器（见「移除」首条），12 个 `staticcheck` 里 8 个是 `s3wrap/*_fake_test.go` 的 `QF1002`（假 S3 的 `switch { case r.Method == http.MethodGet: }` 应为 `switch r.Method { case http.MethodGet: }`，纯风格），另有 `QF1006`（`internal/service/sync.go` 的 `for { if len(out) >= maxTotal { break } }` 提升为循环条件）、`QF1008`（`internal/store/store.go` 冗余的 `s.fileStore.load()` 选择器）、`S1040`（`internal/store/gaps_test.go` 把 `st` 断言成它已是的 `AccountStore` 类型，恒真的空断言）。逐一修掉后 `golangci-lint run` 输出 **0 issues**，`go vet` / `gofmt` 均干净。
- **`main_test.go` 的 `result.err` 字段从未被写入**：并发结果结构体带 `err`，但 goroutine 只填 `code`，`err` 永远为零值，属「看似在断言错误、实则从不校验」的假保障。已把通道简化为 `chan int`，直接断言 `runServer` 返回值。
- **`ha_gap_test.go` 用例循环重复构造请求与 recorder**：循环内先按 `"/nope-suffix"` 建 `req`/`rr`，紧接着在 `if/else` 里**无条件覆盖** `req`、随后又覆盖 `rr`，另有一个恒被丢弃的 `_ = i`。净效果虽等同于直接按 `a.ID` 构造，但读起来像在覆盖「非法后缀」分支。已改为按 `c.name` 一次性决定路径，去掉全部无效赋值（行为不变，`-race` 下仍全绿）。

- **CI 的 Trivy 镜像名错误导致 `docker` job 一直失败（GitHub 侧同样中招）**：`ci.yml` 的 Trivy 步骤引用 `aquasecurity/trivy:0.58.1`，但该路径**只存在于 `ghcr.io`**，Docker Hub 上的官方仓库是 [`aquasec/trivy`](https://github.com/aquasecurity/trivy/blob/v0.58.1/docs/getting-started/installation.md)。`docker run` 拉不到镜像会以 **exit 125** 退出，而漏洞门禁的失败码是 **exit 1**——两者混淆后看起来像「扫出了 CRITICAL/HIGH 漏洞」，实际是镜像名写错。GitHub 上 `docker` job 自引入起连续多次失败均为此因（check-run annotation 为 "Process completed with exit code 125"）。已把两套 CI 统一改为 `aquasec/trivy:0.58.1`（与 `ghcr.io/aquasecurity/trivy:0.58.1` 为同一镜像，config digest 一致）。
- **GitLab 侧 `-race` 在 alpine 镜像下必然失败**：`.go-base` 原用 `golang:1.26.6-alpine`，而 `go test -race` 需要 cgo（`CGO_ENABLED=1`）+ C 编译器，alpine 镜像两者皆无，会直接报 `-race requires cgo`。已改用 `golang:1.26.6-bookworm`（自带 gcc，也更贴近 GitHub 的 `ubuntu-latest`）并显式设 `CGO_ENABLED=1`。
- **GitLab 侧 `docker` job 的 `.trivyignore` 挂载路径不可达**：`docker run -v "$CI_PROJECT_DIR/.trivyignore:..."` 里的 `$CI_PROJECT_DIR` 是 **job 容器内**路径，宿主 daemon 看不到（正式 runner 上 `/builds/...` 为容器卷，宿主无此路径），Trivy 会以 `ignore file not found: /trivy/.trivyignore` 失败。已改为把 Trivy 二进制从固定版本镜像 `docker cp` 到 job 容器内执行——trivy 经挂载的 `docker.sock` 扫描宿主镜像，`--ignorefile` 用容器内可见的相对路径，GitLab 与本地 gitlab-ci-local 行为一致。
- **补完 `apps/` 迁移漏改的 `working-directory`（4 个 GitHub workflow 全部受影响）**：目录迁移把 `paths` / `go-version-file` / `cache-dependency-path` / artifact `path` 改成了 `apps/...`，但 `working-directory:` 与 tauri-action 的 `projectPath:` 仍是迁移前的裸名（`server` / `web` / `desktop`）。这些目录已不存在，相关步骤会以 "No such file or directory" 失败——即迁移后 GitHub CI 实际是红的。已统一改为 `apps/server` / `apps/web` / `apps/desktop`（含 `release-desktop.yml` 的 `projectPath: apps/desktop`）。
- **GitLab 两个 E2E job 的就绪轮询会「假绿」**：GitHub 的写法是 `curl … && exit 0`，但 GitLab 把整个 `script` 拼成**一个 shell 脚本**执行，`exit 0` 会结束**整个 job**——照搬过来会让 `rustfs-e2e` / `playwright-e2e` 在服务就绪后立即退出，**一条测试都没跑却报 PASS**。已改为「置标志位 + `break`」跳出循环，失败路径的 `exit 1` 保留（本就该中断 job）。GitHub 侧不受影响（每个 `run:` 是独立 step）。
- **GitLab 触发规则与 GitHub 逐 job 对齐**：三套 GitHub workflow 的 `on:` 并不相同——`ci.yml` 有 `push` 但**没有** `schedule`；两个 E2E 有 `schedule` 与 `pull_request.paths` 但**没有** `push`。此前 GitLab 用一条全局 `schedule` 规则，会让定时任务把 `server`/`web`/`docker`/`desktop` 全量重跑，且 `push` 时也会跑 E2E。现按 job 拆成 `.ci-trigger` / `.e2e-rustfs-trigger` / `.e2e-playwright-trigger`，E2E 侧用 `changes:` 复刻 GitHub 的 `paths` 过滤。实测四类事件选出的 job 集合与 GitHub 一致。
- **`desktop-build` 的 `when: manual` 被 rules 覆盖**：该 job 一旦引入 `rules:`，job 级 `when: manual` 即失效，会变成自动执行的重负载 Tauri 构建。已改为在 `.ci-manual-only-trigger` 的 rules 里写 `when: manual`，并限定 `CI_PIPELINE_SOURCE == "web"`——对应 GitHub 的 `if: github.event_name == 'workflow_dispatch'`（push / PR 流水线里该 job 根本不存在）。

### 测试与质量门禁
- **后端覆盖率门禁 90% → 100%，并按「删除死代码 / 行为断言」补齐**：`make test-cover` 与 CI 现在直接检查 `coverage.out` 中是否存在 `count==0` 的语句块，不再比较只有 1 位小数的 `total` 百分比——99.96% 会被四舍五入显示成 `100.0%`，用它做门禁会漏过回退。补齐过程中删除了 `jobsList` 中确实不可达的 `recs == nil` 兜底（`JobRegistry.List` 以 `make(..., 0, n)` 构造，空清单也非 nil，序列化本就是 `[]`）；`record()` 里「状态为空时按 done 反推」的兜底则**保留**——它可由公开的 `Emit` 触发（`JobProgress.Status` 是 omitempty，进度帧允许不带状态），而空状态一旦落盘，恢复流程会把它当成非终态、将已完成任务误标为 `interrupted`，属于真实防线而非死代码，因此改为用行为断言锁定。其余缺口全部改用行为断言覆盖：四个异步端点在在册任务达上限时经**真实路由**返回 503 且不注册新任务；`Create` 在容量耗尽时仍返回已终结任务且不占用名额；`restore` 跳过无 ID 的损坏记录；`List` 在 `Created` 相同时按 ID 升序稳定排序；无状态进度帧不得让清单/落盘快照出现空状态；`Emit` 的中间进度按 `jobProgressPersistEvery` 节流落盘（该间隔改为包级变量以便测试快速触发，与既有 `reapInterval` / `finishSendTimeout` 同一模式）；`PublicURL` 在端点不可用（未配置或退化为 scheme-only）时返回空串，而不是拼出损坏链接。8 个包（main / config / handler / model / openapi / s3wrap / service / store）语句覆盖率现均为 100%。

### 文档
> 本节按时间顺序记录本次文档结构重构的每一步。**各条目中的路径名反映该步骤当时的真实位置**——文件名小写化与迁至 `.github/` 发生在后，最终布局以本节最后一条为准。
- **根目录文档统一收敛至 `docs/`**：除 `README.md` 外，`CHANGELOG.md` / `ROADMAP.md` / `CONTRIBUTING.md` / `SECURITY.md` / `CODE_OF_CONDUCT.md` / `agents.md` 全部移入 `docs/`（GitHub 的 `docs/` 目录同样被识别为社区健康文件位置），根目录只保留 `README.md`（代理入口 `AGENTS.md` 随后回归根目录，见下一条）。同步修正 README、`docs/` 内部互链、`scripts/release-version.sh` 与 release workflow 中的路径引用，全仓相对链接已脚本校验无死链。
- **待办清单只保留未完成项**：`docs/todolist.md` 移除 6 项已完成条目（#9 / #13 / #14 / #19 / #20 / #24）以及仅描述已完成工作的归档叙述；⏳ 条目收敛为「剩余工作」，已落地部分不再重复记录（证据在 `FEATURES.md`）。编号保持稳定、不重排——`server/` 代码注释仍以 `todolist #N` 引用本清单，并据此修正 `ROADMAP.md` 中指向已移除编号的悬空引用。
- **文档命名约定 + 修复大小写冲突**：约定普通文档用小写 kebab-case，仅名字被外部约定固定的用大写（`README.md` / `CHANGELOG.md` / `CONTRIBUTING.md` / `SECURITY.md` / `CODE_OF_CONDUCT.md` / `LICENSE`），Agent 指令用 `AGENTS.md`；禁止任意两个路径仅大小写不同，规则写入 `docs/DEVELOPMENT.md` §4。据此把威胁模型 `docs/security.md` 重命名为 `docs/threat-model.md`——它由本次移动带入 `docs/` 后与 GitHub 认的漏洞披露策略 `docs/SECURITY.md` 仅差大小写，在 macOS / Windows 的大小写不敏感文件系统上检出会互相覆盖；README / FEATURES / development / deployment / SECURITY / agents 中的引用同步修正。；同时补充**目录命名**约定——目录一律小写，大写只由工具强制决定（`.github/`、`.github/ISSUE_TEMPLATE/` 已符合，若将来引入 REUSE 规范则为 `LICENSES/`），`docs/` 改成 `Docs/` 会使 GitHub 的社区健康文件查找与 Pages 发布源失效。
- **Agent 指令文件归位到仓库根目录**：删除已废弃的 `docs/agents.md`（其 TDD 约定与「文档同步门禁」对照表早已合并进 [`docs/DEVELOPMENT.md`](docs/DEVELOPMENT.md) §4，文件自身即声明为待删的跳转入口），并在**仓库根目录**新建 [`AGENTS.md`](AGENTS.md)。原因是 agent 工具的发现机制：候选名精确匹配 `AGENTS.md`（区分大小写），项目根由 `.git` 标记确定，且只注入根文件与子树文件——放在 `docs/` 下即使改成大写也不会被加载。新文件只保留仓库级硬约束、门禁命令与文档入口指针（详细规范仍在 `docs/DEVELOPMENT.md`，避免此前的两处维护漂移），并新增 `docs/DEVELOPMENT.md` §4.1 记录加载机制（候选名 / `.git` 根判定 / 子树惰性加载 / 字节预算截断）、`AGENTS.local.md` 加入 `.gitignore`。同步恢复 `docs/CONTRIBUTING.md`、`docs/ROADMAP.md` 对代理入口的引用，并清理 `scripts/release-version.sh` 里针对旧文件、已无匹配目标的两条版本号 sed。
- **5 个文档名小写化，命名规范收口**：`docs/API.md` / `ASSESSMENT.md` / `ERRORS.md` / `FEATURES.md` / `ROADMAP.md` → `api.md` / `assessment.md` / `errors.md` / `features.md` / `roadmap.md`。它们此前是本仓库自己的「台账类大写」习惯，并无任何工具按文件名匹配，属白名单之外的豁免项；本轮按唯一正确处理方向（**小写化**，而非继续大写）消除豁免，命名规范自此**不存在非白名单例外**——大写仅限 `README.md`、`AGENTS.md`、`CHANGELOG.md`、`CONTRIBUTING.md`、`CODE_OF_CONDUCT.md`、`SECURITY.md`、`LICENSE`。改名一律用两步 `git mv`（纯大小写改名在 macOS / Windows 上单步会静默失败），并同步更新引用：README / AGENTS.md / `docs/` 内部互链与正文路径，以及 `scripts/release-version.sh` 中 `docs/api.md` 的 sed 目标（漏改会让 `set -e` 直接中断发版脚本）。`docs/CHANGELOG.md` 的历史条目按惯例不改写，旧名↔新名对照见本条。
- **社区健康文件迁至 `.github/`（规避「静默顶掉」）**：`CONTRIBUTING.md` / `SECURITY.md` / `CODE_OF_CONDUCT.md` 由 `docs/` 移至 `.github/`。依据 [GitHub 官方规则](https://docs.github.com/en/communities/setting-up-your-project-for-healthy-contributions/creating-a-default-community-health-file)：这类文件在 `.github/`、根目录、`docs/` 三处都被识别，但**查找优先级为 `.github/` > 根目录 > `docs/`**，且命中即独占、多份并存不报错——放在最低优先级的 `docs/` 会被将来的根目录副本静默忽略。`CHANGELOG.md` 不在 GitHub 的社区健康文件名单内（位置纯属 Keep a Changelog 约定），故留在 `docs/`。因相对链接方向反转，同步更新 README / AGENTS.md / `docs/DEVELOPMENT.md` / `docs/threat-model.md` 的引用，以及三个文件自身指向 `docs/` 的链接。
- **`CHANGELOG.md` 迁至仓库根目录（类别归位）**：它此前与 `api.md` / `DEVELOPMENT.md` 等**叙述文档**混放在 `docs/`，但类别上属于**约定文件**——Keep a Changelog 钉死了文件名（`CHANGELOG.md`）并把它与 README / CONTRIBUTING 并列为「典型大写文件」，主流 changelog 工具（release-please / semantic-release / standard-version / git-cliff）默认路径也都是 `./CHANGELOG.md`。GitHub 对 CHANGELOG **没有任何查找行为**（它不在社区健康文件名单内），故位置纯由约定决定；未选 `.github/` 是因为那里放的是 GitHub 配置与社区健康文件，放进去没有任何工具会读。共迁移 11 处引用：`docs/` 内 9 处同目录链接升为 `../CHANGELOG.md`、`.github/SECURITY.md` 1 处、CI Release notes 1 处 URL（反而更短）；`AGENTS.md` 与 `docs/DEVELOPMENT.md` §4 的根目录存放规则同步改写，`AGENTS.md` 入口表新增「每个 PR 都要补发版记录」一行。根目录现为 `README.md` / `AGENTS.md` / `CHANGELOG.md` 三个约定文件。
- **补齐 `.github/` 下文件的发现入口**：GitHub 会自动在「新建 issue / PR 页面」「`/contribute` 页面」「仓库概览的 Contributing 与 Code of conduct 标签」「Security 标签页的 Security policy」等处暴露这三个社区健康文件，但仓库还缺两个**显式**入口，本轮补上：① `.github/PULL_REQUEST_TEMPLATE.md` 顶部新增指向 `CONTRIBUTING.md` 与 `docs/DEVELOPMENT.md` 的提示行——PR 模板会被**自动填充**到每个 PR 的描述框，是触达率最高的位置；② 新增 `.github/ISSUE_TEMPLATE/config.yml`，用 `contact_links` 在 issue 模板选择页底部给出「贡献指南」与「报告安全漏洞（请勿公开）」两个入口，并设 `blank_issues_enabled: false` 隐藏空白 issue（Write 及以上权限的维护者仍可见 Maintainers only 入口）。

### 仓库卫生
- **补齐 `.gitignore` 缺口**：审计工作区（`git ls-files -i -c` 无命中、未跟踪且未忽略文件仅 `AGENTS.md`）后补 4 类未覆盖项——① `coverage.out` / `coverage.txt`：`make test-cover` 与 CI 的 Go 覆盖率产物，**属真实缺口**（通用 `coverage` 规则只匹配同名目录，匹配不到这两个文件）；② OS 垃圾 `.DS_Store` / `Thumbs.db` / `Desktop.ini`（本项目同时发布 macOS `.dmg` 与 Windows `.exe`）；③ 编辑器临时文件 `.idea/` / `*.iml` / `*.swp` / `*.swo` / `*~`；④ Agent 工具本地目录 `.claude/` / `.cursor/`。`.vscode/` 采用 `.vscode/*` 加白名单 `extensions.json` / `settings.json`，保留将来共享团队配置的位置（已用退出码与真实建文件端到端验证：仅这两个文件出现在 `git status`）。
- **本轮按最小改动未处理的已知项**（留待专项）：5 条被通用规则覆盖的冗余条目（`web/node_modules/`、`desktop/node_modules/`、`web/dist/`、`server/data/`、`web/playwright/.cache/`）、过宽规则 `data/`（裸名会静默忽略任意层级的 `data/` 目录，如将来的 `web/src/data/`）与 Next.js 遗留的 `out`，以及 `.gitignore` 中 143 行与本技术栈无关的 Node 通用模板。
- **已核实无需改动**：账号存储文件（`accounts.json` / `accounts.db` / `accounts.json.enc`，`S3C_DATA_DIR` 默认 `./data`）均在忽略范围内；nginx 日志写容器内 `/var/log/nginx/` 或 stderr；compose 使用命名卷；Trivy 仅只读挂载 `.trivyignore`、不落报告文件。

### 开发规范
- **硬约束第 5 条细化为「不留尾巴（死代码零容忍）」**：明确列举四类禁止项——死代码（定义后无人引用的函数/类型/常量/变量/字段、不可达分支）、未定义即使用的标识符、定义了却未使用的变量（含只写不读、赋值后即被覆盖、恒真/恒假的空断言）；**枚举值除外**（枚举/常量表成员即使暂无引用也属契约，保留）。同时写入两条此前缺失的认知：① **警惕「被覆盖率掩盖的死代码」**——测试文件不参与 instrumentation，`go test -cover` 只看生产代码，测试辅助里的死代码能让 100% 门禁全绿（本仓库即因此让一套失效的错误注入器长期存活），覆盖率达标与无死代码必须分别验证；② 机械检查必须开启并保持**零告警**（Go: `golangci-lint` 的 `unused`/`staticcheck` + `go vet`；TS/JS: `pnpm lint` + `vue-tsc`），不接受「历史遗留」作为例外。同步写入 [`AGENTS.md`](AGENTS.md) 硬约束与 [`docs/DEVELOPMENT.md`](docs/DEVELOPMENT.md) §5 验收清单、§6 Red Flags（含「不要用 `_ = x` / `//nolint` / 导出为 `_test` 辅助来消音」）。
- **新增「改完代码必须同步文档」强制规则**：修复 bug 或新增功能完成后**必须**更新相关文档，文档未同步视为改动未完成、不得提交/合并。新增「改动类型 → 必须更新的文档」对照表（README / `docs/API.md` + `openapi_register_*.go` / `CHANGELOG.md` / `docs/FEATURES.md` / `docs/todolist.md` / `ROADMAP.md` / `docs/architecture.md` / `docs/DEPLOYMENT.md` / `docs/security.md` / `docs/ERRORS.md` / `CONTRIBUTING.md` 等），落地到 `agents.md`（后已删除）与 [`docs/DEVELOPMENT.md`](docs/DEVELOPMENT.md) §4「文档同步门禁」，并写入验收清单与 Red Flags；[`CONTRIBUTING.md`](.github/CONTRIBUTING.md) 开发规范速览同步补充。

### P1 稳定版门槛修复（2026-09-16 评估）
- **异步任务清单持久化 + 重启恢复（v1.0.0 最后一项门槛）**：`JobRegistry` 此前纯内存，进程重启即丢失全部任务记录；而「复制→删源」的移动语义是两阶段操作，中途崩溃会留下「已复制但源未删除」的中间态且无任何痕迹可对账。现在任务清单经 `service/job_persist.go` 的 `JobPersister` 落盘（临时文件 → rename → 0600；`Create`/`Finish` 必写、中间进度 2s 节流），启动时把非终态任务标记为 `interrupted` 并回写；新增 `GET /api/migrate/jobs` 返回任务清单；前端 `MigratePanel` 新增「未完成任务」区块，明确提示移动任务「可能已复制但源未删除」并支持逐条忽略。`interrupted` 采用独立 7 天保留期（`JobInterruptedTTL`），修复了「恢复后首次 reap 就被 30 分钟 TTL 清除、恢复功能形同虚设」的缺陷（附回归测试）。落盘未复用 `store/atomic.go`——`service→store` 会造成分层倒置，两处技术策略一致。`NewJobRegistry()` 保持纯内存语义，既有调用方与测试不受影响。
- **Go 工具链 1.26.5 → 1.26.6，并新增 `govulncheck` 门禁**：`server/go.mod`、`server/Dockerfile` 同步至 1.26.6（CI 经 `go-version-file` 自动跟随）。修复 go1.26.5 中 6 个**可达** stdlib 漏洞（net/url 二次方复杂度、crypto/tls 握手 DoS、net/http HTTP/2 探测、encoding/xml 递归、encoding/asn1 递归、net/http Punycode），`govulncheck ./...` 实测由「6 个可达」降为 **0**。CI 在 `go vet` 后新增固定版本 `golang.org/x/vuln/cmd/govulncheck@v1.8.0` 门禁——Trivy 只扫容器 OS/库，拦不住标准库 CVE，此前是盲区。
- **修正 2 处幽灵 action SHA**：`e2e-playwright.yml` 的 `pnpm/action-setup` 与 `actions/upload-artifact` 指向的 commit 在 GitHub 上不存在（API 404），会导致工作流失败，且若上游伪造同名 tag 存在执行恶意 action 的供应链风险。已改为与其它 workflow 一致的正确 SHA；全仓 10 个 action SHA 经 GitHub API 逐一核验均有效。
- **补齐 3 个缺失 i18n 键**：`objects.toastCopyFailed`、`batchEdit.tagsNeedKey`、`common.working` 此前被引用但未定义，用户界面会直接显示原始 key。已补齐 zh-CN / en-US 文案；新增 `src/i18n/coverage.test.ts` 静态扫描「被引用但未定义」的字面量键（用 `import.meta.glob` 读源码，不引入 `node:*` 依赖），作为该类缺陷的常驻门禁。
- **OpenAPI 契约收尾**：`POST /api/accounts/{id}/delete-marker/restore` 请求体由错误的 `deleteMarkerId` 修正为 handler 实际解析的 `versionId`（此前客户端按文档调用会因缺 `versionId` 直接 400）。契约测试同步扩展至 `delete-marker/restore` / `version`(DELETE) / `version/restore`，断言「含 `versionId` 且不含 `deleteMarkerId`」，并已验证该测试在回退修复时确实失败。

### P2 加固修复（2026-09-16 评估）
- **预签名失败不再被静默吞掉**：`presign`（get/put/post）与 `multipart/part` 三处原为 `u, _ := client.PresignXxx(...)`，失败时返回 `200 {"url":""}`——调用方拿到空 URL 却看到成功状态。预签名并非不可能失败：AWS SDK 在取凭证、输入序列化等阶段都会报错（实测空 key、context 取消均可触发）。现统一经 `writePresignResult` 处理，失败返回 500 `failed to create presigned url`，并新增源码级门禁 `TestPresignErrorsNotSwallowed` 防止该写法复发（已回退验证门禁会失败）。
- **流式传输中断不再无痕**：`copyStream` 原以 `_, _ = io.Copy(...)` 丢弃返回值，大文件下载被上游读失败或写超时打断时，日志与指标里都没有任何记录。现返回 `(int64, error)`，由 `recordStreamOutcome` 分流：真实中断记 Warn 并累加新指标 `s3c_stream_interrupted_total`；客户端主动断开（用户取消/关页面）仅记 Debug、不计入指标，避免污染告警。
- **异步任务加上限**：`JobRegistry` 新增在册任务上限（256 个未终结任务），超限时异步端点返回 503 `too many running jobs; retry later`，避免短时间内大量请求持续堆积 goroutine、SSE 订阅与落盘条目。上限只统计未终结任务，因此恢复出的 `interrupted` 历史任务不会永久占满名额。`Create` 保持原签名以不波及 70+ 处调用点，新增 `TryCreate` 供需要感知容量的路径使用。
- **TLS 站点补齐 HSTS 与 Permissions-Policy**：`deploy/nginx/conf.d/s3client-tls.example.conf` 增加 `Strict-Transport-Security`（180 天，暂不带 preload）与 `Permissions-Policy`（关闭定位/麦克风/摄像头/支付/USB/interest-cohort）。这两项只能由 TLS 终止层表达，后端已有的 CSP 等头无法替代。

### P2 加固修复（续）
- **清理后端死代码**：删除 `ctxReader`（与 `ctxCancelReader` 职责重复且零生产引用）、`batchItemError`（生产零引用；该格式串实际在 `service/batch.go` 内联）、`Client.S3()`（导出但零生产调用，E2E 清理逻辑改用包装层已有的 `DeleteObjectVersion`/`DeleteObject`）。核验后**保留** `isNoSuchBucketSetting`——它有 5 处生产调用，todolist 将其列为死代码属描述有误。当时覆盖率为 99.6%（该轮之后门禁已提升至 100%，见上方「测试与质量门禁」）。
- **compose 默认结构化日志**：`docker-compose.yml` 与 `docker-compose.prod.yml` 注入 `S3C_LOG_JSON: "${S3C_LOG_JSON:-1}"`，容器日志可直接被采集器按字段检索；设 `S3C_LOG_JSON=0` 可退回纯文本。已用 `docker compose config` 验证默认值与覆盖行为。
- **端点归一化合并为唯一实现，修掉 URL 损坏**：`s3wrap` 与 `service` 各有一份归一化且行为不一致——`s3wrap` 只做大小写敏感的 `http://`/`https://` 前缀判断，把用户手填的 `"HTTP://MinIO:9000"` 当成裸主机，拼成损坏的 `"http://HTTP://MinIO:9000"`。现 `s3wrap.NormalizeEndpoint` 为唯一实现（去首尾空白、scheme 大小写不敏感、host 小写、**保留路径前缀**），`service.SameEndpoint` 与 `PublicURL` 均改为复用；`PublicURL` 同时修掉 `" http://a.com/ /b/k"` 这类带空格的链接。
- **`ValidateEndpoint` 不再把合法端点误判为非法**：该函数此前 `strings.TrimSpace` 只用于判空、`url.Parse` 仍用未 trim 的原串，用户填 `" http://127.0.0.1:9000 "` 会收到 `invalid endpoint URL`（且消息里是难懂的 `first path segment in URL cannot contain colon`）。现改为归一化后解析；**保持 fail-closed**——非空输入归一化后退化（`"http://"`、`"/"`、`"https:///"`）仍报错，不因归一化静默放行。E2E 实证：畸形端点建账号后 `preview-buckets` 由 400 变为通过校验进入网络调用，日志中 BaseEndpoint 为干净的 `http://127.0.0.1:9000`。
- **删除 handler 侧 4 个死代码包装**：`derefString` / `derefInt64` / `timeOrZero` / `boolOrFalse` 生产零引用，仅被 `TestAccDerefHelpers` 引用（覆盖率注水样本），连同该测试一并删除。`s3wrap` 内的同名 helper 保留——各有 29/1/5/2/4 处生产调用。

### P0 发布阻塞修复（2026-09-16 评估）
- **OpenAPI 契约与真实 handler 字段级对齐**：`/api/migrate`、`/api/migrate/async` 请求体字段改为 handler 实际解析的 `sourceAccountId` / `sourceBucket` / `sourceKeys` / `targetAccountId` / `targetBucket` / `targetPrefix`（删除此前虚构的 `deleteSource` / `storageClass`）；presign `method` 枚举由 `GET/PUT/DELETE/HEAD` 改为实际支持的 `get/put/post`；delete 请求体移除 handler 不解析的 `versionId`；multipart/part 补上 handler 支持的 `expiresIn`。新增 `TestOpenAPI_ContractRequestBodyMatchesHandlers`，对「注册表 schema ↔ handler DTO」做字段级一致性断言（含 `$ref` 解引用 / enum 检查），补上 routes↔spec 检查覆盖不到的漂移面。
- **前端 Token 不再明文落 `localStorage`**：`s3c.servers` 只存 `{id,name,base}`；token 单副本按 `s3c.token.<serverId>` 存储（默认 sessionStorage，仅显式开启「跨会话保留」时写 localStorage）。旧版本内嵌 token 首次读取时一次性迁移并回填活动服务器 profile；删除服务器同步清理 per-server token。修复「token 仅 sessionStorage」策略被多服务器列表旁路的问题。
- **`loadAll()` 至少在 `nextToken` 为空时也发起一次请求**：改为 do-while，空 token 时以 reset 语义加载第一页（替换旧列表、避免重复项），不再出现「一次请求都没发却提示已加载全部」；某页失败即停止续页且不再发成功提示。
- **迁移 SSE 终态检测兜底，消除 Promise 永久悬挂**：流以 EOF 结束且未收到终态时，轮询 `migrateJobStatus`（500ms 间隔、30s 上限）直到 done/cancelled 并合成终态 `status`；连续 3 次回读失败快速 `onError`。`ctxDeleteFolder` / `DestDialog` / `MigratePanel` 三个调用方由此保证拿到终态或错误，`opsBusy` 不再永不复位、按钮不再永久禁用。

### 安全与可靠性
- **运行时基镜像换成 Alpine 3.20**：`debian:bookworm-slim`（~80MB、90+ 包、glibc + openssl 3.0.x）→ `alpine:3.20`（~5MB、19 个包、musl + openssl 3.3.7）。体积从 ~80MB 降到 41.5MB；Trivy 报告的 CRITICAL/HIGH 漏洞面收窄 ~80%。
- **`/api/metrics` 默认关闭**：新增 `S3C_EXPOSE_METRICS=1` 显式开启；默认返回 404，假装端点不存在，避免公网被 scrape 运行指标。
- **S3C_TOKEN 短口令硬失败**：长度 < 16 字符（含多 token 中最短者）直接拒绝启动，不再仅告警。
- **S3 出站禁 HTTP(S)_PROXY**：transport 显式 `Proxy=nil`，防止环境变量代理绕过 `dialContextSSRF` 校验。
- **桶名校验收紧**：拒绝首尾为 `.`/`-`、连续 `..`/`.-`/`-.`。
- **user metadata 边界 400**：键值长度 / 字节总长超 S3 限制直接 400，不再落到 S3 端以 500 返回。
- **delete-objects 单次 ≤1000**：与 S3 `DeleteObjects` 上限对齐；超限返回 400 并提示走 `delete-prefix`。
- **migrate SSE 写超时**：复用 `streamIdleTimeout`（每次成功写出后刷新），慢客户端不会让连接无限挂着。
- **store 失败回滚**：`EncryptedStore.Update` 持久化失败回滚内存；`SQLiteStore.Update` 显式传播 `Exec` 错误；`JSON Store.Update` 补测。
- **错误映射去字符串匹配**：新增 sentinel `ErrObjectTooLarge` / `ErrSourceDeleteFailed` 并用 `errors.Is` 判断；`PutObject`/`CopyObject` 把 S3 `EntityTooLarge` 归一为 sentinel，不再依赖 SDK 文案。
- **X-Request-ID 回显加固**：仅回显长度 ≤128 且全为可见 ASCII 的值，超长/含控制字符改由服务端生成 UUID，阻挡日志与响应头注入。
- **Bearer scheme 大小写不敏感**：按 RFC 7235 接受 `bearer`/`BEARER` 等，凭证仍常量时间比较。
- **`.env` 解析解耦进程 CWD**：`S3C_ENV_FILE`（设置后为唯一来源）→ CWD `.env` → 可执行文件同目录 `.env`；真实环境变量始终优先。
- **nginx 内存上限**：两个 compose 文件为 nginx 显式设置 `deploy.resources.limits.memory: 128M`。

### 工程化（v1.0.0-rc1 评估 P1/P2 路线落地）
- **OpenAPI 自动生成**：`/api/openapi.json` 端点（无依赖显式 builder），70 个 `/api/*` 端点按域（accounts/buckets/bucket-settings/objects/object-meta/multipart/versions/trash/migrate/system）集中登记；与 routes.go 一一对应；端点不进鉴权层（契约非业务）。
- **Playwright 浏览器 E2E 基建**：chromium + vite preview + `page.route()` 拦截 `/api/*` 回放 fixture；`smoke.spec.ts` 3 例 + `account-flow.spec.ts` 2 例；新增 `.github/workflows/e2e-playwright.yml`（PR/dispatch + 每周三 02:00 UTC 冒烟）；`pnpm e2e:install` 安装 chromium 与系统依赖；vitest exclude `e2e/**` 避免冲突。
- **增量同步**：`POST /api/migrate/sync`（withStreamLimit 保护），按 `etag`（默认）/ `size_mtime` / `always` 三种 mode 比对源/目标，仅复制差异对象，跳过完全一致的对象；internal/service.SyncKeys 复用 MigrateKeys 完成实际复制；`s3wrap.Client.Endpoint()` 访问器供 SameEndpoint 比对。
- **桶策略可视化编辑器**：web/src/bucketPolicy.ts + BucketPolicyVisualEditor.vue：Statement（Effect/Principal/Actions/Resources/Sid）表单式编辑，4 个常用模板（公共读 / 公共读写 / 拒绝 List / 清空），实时 JSON 预览 + validateDoc 校验；不支持的结构（NotPrincipal/嵌套）自动回退原始 JSON 模式，避免覆盖用户已写的高级策略。
- **对象元数据批量编辑**：web/src/batchMetadata.ts 4 路有界并发 worker-pool（BATCH_META_CONCURRENCY=4 与 copy/migrate 同上限）；BatchMetadataDialog：ACL 下拉 / 标签替换或清空 / 存储类型输入，3 个 fieldset 独立勾选；运行中进度 + 失败明细可滚动展示；ObjectToolbar 新增「批量改元数据」按钮。

### 工程化
- **CI 工具链对齐**：所有 workflow 的 pnpm 统一为 11；`desktop-build` 拆为「先装 tauri-cli（CI 缓存）→ 再 tauri build」，去掉 `|| true` 静默失败。
- **OpenAPI 生成器死代码清理**：删除零引用的 `Prop()` 与从未使用的 `Request.Example` / `Response.Headers` / `Response.Content` / `Schema.Additional`；`internal/openapi` 覆盖率仍 100%，`/api/openapi.json` 输出字节级不变（对外契约不动）。
- **文档收敛**：三份已迁移的历史评估快照（`full-assessment.md` / `code-review.md` / `code-review-v1.0.0-rc1.md`）移除；散落的待处理事项统一汇总到 `docs/todolist.md`（单一待办来源），`FEATURES.md` / `README.md` 同步更新链接。
- **存储驱动再收敛**：`EncryptedStore` / `encryptedCodec` 并入统一 `Store` + 单一 `storeCodec`（`strict` 区分 json permissive / encrypted 严格语义），`encrypted.go` 删除，`NewEncrypted` 返回 `*Store`；磁盘格式、错误文案、盐策略与覆盖率 100% 不变，解决历史「双驱动功能重叠」遗留项。
- **账号响应契约收敛**：账号响应从 `secretKey: "******"` 占位改为 `AccountView`（新增 `secretSet: boolean`，**不再回传 `secretKey`**）；请求仍以 `secretKey` 提交（编辑留空 = 保持不变）。后端（`model.AccountView` / handler 视图转换）、前端（`types.ts` `Account.secretSet`）、OpenAPI 与 `docs/API.md` 同步更新；`internal/model` 覆盖率 100%。
- **OpenAPI components 接线 `$ref`**：新增 `openapi.Ref()` 与 `Param.Ref` / `Response.Ref` 渲染分支；共享 `schemas`（Error/Account/Bucket/ObjectItem/ListObjectsResp）、`parameters`（AccountID/Bucket/Prefix/MaxKeys/ContinuationToken）、`responses`（BadRequest/NotFound）经 `refSchema` / `refParam` / `refResp` 全部接线为 `$ref`（109 处引用、12 个唯一目标），消灭「components 0 引用」死代码状态；契约测试改为先解析 `$ref` 再校验路径参数 / 响应描述，新增「components 无死片段」断言（`Unauthorized`/`TooManyRequests`/`InternalError` 为全局错误词汇除外）；`internal/openapi` 覆盖率保持 100%。`/api/openapi.json` 输出形状从内联改为 `$ref`（对外契约结构性更新，运行时 API 行为不变，前端不消费该文档）。
- **待办清理**：`docs/todolist.md` 移除「三、历史评审遗留」整节（7 项均已复核关闭，详见 `FEATURES.md` C / D / E），待办清单仅保留仍未决事项。
- **待办归档**：`docs/todolist.md` 全部 4 项（存储驱动再收敛 / `secretSet` 账号契约 / OpenAPI `$ref` 接线 / `preview-buckets` 预览桶契约）均已完成，从待办清单移除并归档至 `FEATURES.md`「一、产品功能」与「二、已完成修复与优化」（A / B 段补记存储驱动收敛完成态），待办清单当前为空。
- **综合评估 + 文档结构重构**：4 路并行深度审查（后端 Go / 前端 Vue-TS / 安全威胁与依赖 / SRE 可靠性）产出 `docs/ASSESSMENT.md`（五维度评分：代码质量 82 / 漏洞 72 / 死代码 70 / 降级 74 / 自我迭代 90），发现 OpenAPI 契约失真、Go 1.26.6 修复线、幽灵 SHA、异步任务无恢复等 20+ 项并回写 `todolist.md`；文档按开源项目常见结构重构——新增 `SECURITY.md` / `CODE_OF_CONDUCT.md` / `.github/ISSUE_TEMPLATE` / `PULL_REQUEST_TEMPLATE.md` / `docs/architecture.md` / `docs/DEVELOPMENT.md`（合并原 `agents.md`）/ `docs/DEPLOYMENT.md` / `docs/security.md` / `docs/decisions/`（ADR-001~004），README / FEATURES 链接同步更新。
- **Trivy 镜像扫描**：CI 在 Docker 构建后跑 `aquasecurity/trivy:0.58.1`，CRITICAL/HIGH 漏洞硬失败；新增 `.trivyignore` 与 `--ignorefile` 集中收纳可忽略的 CVE。
- **构建层升级 Node 24**：Dockerfile 与所有 workflow 的 `node-version` 升到 24；pnpm 锁回 9.15.0（与 `package.json` 的 `packageManager` 声明一致，兼容现有 `pnpm-lock.yaml`）。
- **配置校验函数 `Config.Validate()`**：`MinTokenLength=16`、非回环无 token 拒绝启动。
- **`go mod tidy` 显式化**：Makefile 移除每次 `server` 启动的隐式 tidy，新增 `make tidy` 显式入口。
- **SSE 卸载中断**：`useObjectActions` `showDetail` 加 `detailSeq` 序号守卫；`MigratePanel` 组件级 `activeUnsub`，`onBeforeUnmount` 断开进行中的 SSE。
- **列表请求 AbortController**：`useObjectBrowser.load` 每次新请求取消旧的；`onBeforeUnmount` 取消进行中的 fetch。
- **双文件存储驱动去重**：`json` / `encrypted` 驱动共享 unexported `fileStore` + 注入式 `fileCodec`，锁 / CRUD / 持久化回滚 / 快照排序 / JSON 中间表示单点实现（净减约 331 行），文件名、磁盘格式、错误文案与加密语义不变。
- **OpenAPI 契约测试**：`routes.go` ↔ 规范双向一致（漏登记与陈旧条目都红灯）、operation 完整性 / 路径参数 / `$ref` 可解析 / `MarshalJSON` 确定性；`internal/openapi` 包与 handler 内全部 OpenAPI 函数达 100% statement 覆盖。
- **全仓 gofmt 对齐 Go 1.26**：修正结构体字段对齐与文档注释空行规则，CI `gofmt -l .` 门禁恢复干净。

### 前端
- **Token 默认 sessionStorage**：`s3c_token_persistent='1'` 显式开启才写 localStorage；旧版本 localStorage 遗留值首次读取时自动迁移到 sessionStorage 并清空 localStorage 副本。
- **MigratePanel 虚拟滚动**：`listAll` 上限 200×1000 不再 `<tr v-for>` 全量渲染，按 ObjectList 同款窗口化方案。
- **showDetail seq 守卫**：快速切对象时过期响应丢弃。
- **lint 收紧**：`@typescript-eslint/no-explicit-any` 由 off 改 warn；新增 `isTauri` 类型明确化。
- **集中式 `requireAccId()`**：`useObjectActions` 13 处 `ctx.account.value!.id` 收敛为单一异常入口。
- **SSE 卸载中断补齐**：`DestDialog` 迁移进度流与 `useObjectActions` 前缀删除流在 `onBeforeUnmount` 主动 abort，不再后台悬挂推送。
- **VersionsDialog 分页**：按 `keyMarker`/`versionIdMarker` 翻页（≤20 页），仍被截断时显示提示，不再静默丢弃第 1000 条之后的版本。
- **消除非空断言**：剩余 3 处 `ctx.account.value!` 与 `ObjectsPanel` 的 `{} as KeyBindings` 改为守卫 / 可选类型（`KeyBindings` 字段可选 + `?.` 调用）。
- **blob 下载延迟回收**：`revokeObjectURL` 由同步改为延迟 60s，避免中断浏览器尚未开始的下载。
- **lint 再收紧**：`@typescript-eslint/no-explicit-any` 由 warn 改 error（当前 0 违规）。


## [1.0.0] - 2026-09-22

首个稳定版（v1）。整合 v0.16.0 之后的全部迭代，此后按 `v1.0.0-年月日十分秒` 版本发布。

### 复制 / 移动 / 删除（跨桶，全闭环）
- 单文件复制 `POST /copy-object`（不删源）与批量复制/移动 `POST /copy-objects`（保留文件名，`deleteSource` 移动）；文件/文件夹统一「复制到… / 移动到…」对话框（目标桶下拉 + 目标路径/前缀）。
- 移动语义：文件 = 复制成功后删除源；文件夹 = 复制成功后删除源前缀（副本有失败则保留源）。
- 修复 `rename` 跨桶同名移动被误拒（同桶内 `newKey == key` 才拒绝）。
- 未知类型文件行内「预览」按钮常显。

### 大文件分段上传（Multipart Upload 直传）
- 后端 `multipart/init | part | complete | abort` 四端点；前端 `≥100MB` 自动切 10MB 段、4 路并发直传，任一段失败自动 abort；小文件仍单 PUT。
- 要求 Bucket CORS 暴露 `ETag` 响应头（见 README）。

### 对象元数据管理（HTTP 头 / ACL / 标签）
- 对象权限 ACL：`GET/PUT /object-acl`，私有/公共读切换 + 复制公开链接。
- 对象标签 Tagging：`GET/PUT /object-tags`，键值编辑、一键清空；无标签返回空列表（兼容 `NoSuchTagSet`）。

### 导航与交互
- 左侧菜单更名（账号管理 / 文件上传 / 服务器设置等），按依赖分组「数据操作 / 配置」，首次进入按账号存在路由到「对象管理」。

### 后端与测试
- 新增 S3 接口：`CreateMultipartUpload` / `Complete/AbortMultipartUpload` / `PresignUploadPart` / `Get/PutObjectAcl` / `Get/Put/DeleteObjectTagging`。
- 假 S3 单元测试覆盖上述能力与参数校验；`go test ./...`、`pnpm build` 均通过。

## [v1.0.0-rc1] - 2026-09-02

相对 `v1.0.0-rc0`：桌面安装包由 CI 在打 tag 时交叉构建并挂到 GitHub Release。

### 工程化
- **桌面发版 CI**：`release-desktop.yml` 在 `v*` tag（或手动指定 tag）时交叉构建 NSIS `.exe` / `.deb` / `.dmg`，并上传到对应 GitHub Release。

## [v1.0.0-rc0] - 2026-09-02

首次候选发布（`develop` 全栈快照）。相对时间戳构建的主要增量：CI（gofmt / golangci-lint v2 / desktop cargo / race）打通；不做服务降级与无历史加密/账号文件格式兼容。

### 安全与可靠性
- **不做服务降级 / 无历史格式兼容**：`/api/health` store 失败返回 `503` + `status:error`；`encrypted` 仅 `S3C2`；JSON 账号文件写入 `0600`（加载时不再改旧权限）；移除 `sameEndpoint` 测试 shim 与前端「旧后端无 version」兼容注释。

### 安全与可靠性（ANALYSIS 整改·续2）
- **copyPrefix 异步**：`POST .../copy-prefix/async` 复用 job+SSE；前端文件夹复制改走异步。
- **copy-objects / delete-prefix 异步**：`POST .../copy-objects/async`、`POST .../delete-prefix/async`；多选移动与文件夹删除走 SSE。
- **多 token**：`S3C_TOKEN` 逗号分隔支持轮换。
- **防腐层**：`s3wrap.UserMessage`/`HTTPStatus`/`IsNotFound`/`ObjectStream`；handler 生产代码不再直接依赖 smithy/types；`service.CopyKeys`/`MigrateKeys`/`WriteObjectsZip`/`JobRegistry`；i18n≈570 键。
- **service 收口**：迁移引擎、ZIP、异步 Job 注册表迁出 handler；短写校验；`Entry`/`SortKey` 提到 `types.ts`。
- **可观测**：`GET /api/metrics`（Prometheus 文本）、`X-Request-ID`、可选 `S3C_LOG_JSON=1`。
- **TLS**：`docker-compose.tls.yml` 叠加 prod；a11y（aria-current/aria-sort/焦点恢复/网格键盘）；对比度与死资源清理；ANALYSIS §七整改对照。
- **长尾**：ZIP 4-worker 并行拉取；`docs/ERRORS.md`；coverage_gaps 拆分；`noUnusedLocals`；ModalDialog 组件测；i18n≈640；ServerPanel 多 token 提示；desktop tauri build 仅手动 workflow_dispatch。

### 安全与可靠性（ANALYSIS 整改·续）
- **>5GB 迁移 multipart**：跨端点流式分段；同端点 CopyObject EntityTooLarge 自动回退 multipart（`internal/service`）。
- **copyPrefix**：4 worker 并发 + `withStreamLimit`；delete-prefix 同样纳入流限。
- **加密 KDF**：`encrypted` 存储改为 magic+盐派生（现为 Argon2id / `S3C2`）。
- **API 限速**：每 IP 令牌桶约 120/min（health/静态除外）。
- **错误映射**：`writeInternalErr` 统一走 `s3HTTPStatus`（NoSuchBucket→404 等）。
- **生产**：TLS nginx 示例、README token 必填说明、`service` 包抽离流式复制。

### 前端（续）
- ObjectList 列表虚拟滚动；i18n≈308 键；ESLint 基线；token 可选 sessionStorage（`s3c_token_ephemeral=1`）。

### 安全与可靠性（ANALYSIS 整改）
- **SSRF**：S3 HTTP 客户端禁重定向 + Dial/创建时拦截链路本地与云元数据地址。
- **鉴权**：非回环监听且无 `S3C_TOKEN` 时**拒绝启动**；compose 强制 `${S3C_TOKEN:?}`。
- **RustFS 联调**：口令改为环境变量必填；CORS 收紧；镜像 pin `1.0.0-rc.3`；健康检查不依赖 curl。
- **流式写超时**：`statusRecorder.Unwrap` 修复静默失效；改为 5 分钟滚动空闲超时。
- **异步迁移**：引擎级超时/取消 API、SSE 心跳与终态可靠投递、关停取消任务；前端 EOF 回读 + 取消按钮。
- **其他**：`sameEndpoint` 保留 scheme；presign 未知 method→400、默认 1h/上限 24h；ZIP 已压缩 Store + 断开感知；SVG 排除 inline；zip-slip NTFS 消毒；5GB 用 `5e9` 字节；标签上限 10。
- **运维**：`docker-compose.prod.yml`、nginx `proxy_buffering off`、health 探测 store、SQLite `user_version`、dependabot、CI gofmt/race/cover/desktop。

### 前端
- Escape 键栈（多层弹窗只关顶层）；KeepAlive max=5；分段上传段级重试；ZIP 优先 File System Access 流式落盘；i18n 高频键扩展；回收站减少空页扫库。

### 工程化与质量
- **s3wrap 拆分**：989 行单文件拆为 `client` / `bucket` / `object` / `presign` / `multipart` / `helpers` 六文件（单文件 <400 行）。
- **错误处理收敛**：`writeInternalErr` / `writeBadJSON` / `s3UserMessage` / `batchItemError` / `s3HTTPStatus`。
- **CI**：Web `pnpm test` + lint；E2E 独立 workflow（PR 路径触发 + 每周定时）；gofmt/race/cover/golangci 门禁。
- **前端**：Hash 深链接；`upload` / `i18n` 单测；主导航中英文切换（i18n 脚手架）。
- **发版**：`scripts/release-version.sh`；`make test-all`。
- **账号存储 SQLite**：`S3C_STORE_DRIVER=sqlite`（纯 Go `modernc.org/sqlite`，无 CGO），WAL 模式 `accounts.db`。
- **账号存储加密**：`S3C_STORE_DRIVER=encrypted` + `S3C_STORE_KEY`，AES-256-GCM 静态加密 `accounts.json.enc`。
- **移除遗留迁移**：不再从明文 `accounts.json` 自动导入到 sqlite/encrypted；前端不再迁移旧版单地址 localStorage 配置。
- **异步迁移 SSE**：`POST /api/migrate/async` + `GET /api/migrate/jobs/{id}/events` 实时进度；前端迁移面板改用 SSE。

## [v1.0.0-20260901182023] - 2026-09-01

> 版本命名改为 `v1.0.0-年月日十分秒`（如 `v1.0.0-20260901182023`）。本版本整合「对象存储类型 / 版本比较 / 一键还原、桶管理、回收站」三块迭代。

### 对象存储类型 + 版本比较 + 一键还原（Batch1）
- 存储类型（StorageClass）展示与切换：`HeadObject` 返回 `storageClass`（支持 `versionId`）；对象列表/网格与「对象详情」展示；新增 `POST /api/accounts/{id}/storage-class`（`CopyObject` 副本到自身 + `x-amz-storage-class`）一键切换（标准/低频/单区/智能分层/归档等）。
- 版本比较/详情：`presign` 支持 `versionId`（`PresignGetVersion` 对历史版本生成签名 GET）；「版本」对话框新增「版本比较」——选两个内容版本并排比元数据（大小/时间/存储类型/ETag）+ 逐行内容差异高亮（二进制或 >2MB 仅比元数据，可分别下载）。
- 一键还原已删除对象：新增 `POST /api/accounts/{id}/delete-marker/restore`（`DeleteObject` 删除标记版本）撤销删除；「版本」对话框删除标记行新增「还原」。

### 桶管理菜单（Batch2）
- 新顶层菜单「桶管理」：列出/新建/删除/管理桶；桶详情含 **概览（版本控制开关）、生命周期、加密（SSE）、CORS、网站托管、桶策略、桶标签** 7 个页签。
- 后端新增 `Get/Put/DeleteBucketEncryption`、`Get/Put/DeleteBucketCors`、`Get/Put/DeleteBucketWebsite`、`Get/Put/DeleteBucketPolicy`、`Get/Put/DeleteBucketTagging`（15 个端点）；`isNoSuchBucketSetting` 把「未配置」错误映射为空响应。

### 回收站菜单（Batch3）
- 新顶层菜单「回收站」：遍历列出桶内全部删除标记（分页游标，空页自动翻页）；每项支持 **一键还原** 与 **彻底清除**（`POST /api/accounts/{id}/trash/purge`，`PurgeObject` 永删该 key 全部版本+标记）。
- 新增端点：`GET /api/accounts/{id}/trash`、`POST /api/accounts/{id}/trash/purge`。

### 测试
- 假 S3 单测：`TestChangeStorageClass` / `TestRestoreDeleteMarker` / `TestPresignGetVersion` / `TestHeadObject` / `TestListObjectVersions` / `TestBucketSettings` / `TestTrash`，覆盖各读/写/校验与未配置分支。
- 真实 MinIO E2E：`TestE2EBatch1`（删除标记还原 + 版本预签名）、`TestE2EBucketSettings`（桶策略/标签）、`TestE2ETrash`（彻底清除 3 个版本+标记、无残留）均通过（MinIO 对扩展存储类型/CORS/网站托管/SSE 的 S3 API 支持受限，已做容错说明）。

### 四轴评审反馈修复（安全 / 正确性 / 可维护性）
- **前端修复**：Vue `key` 为保留属性、不复用为 prop——7 个弹窗（ACL/HTTP头/标签/存储类型/版本/版本比较/复制移动）的对象 key 全部改为 `objectKey`，此前这些功能实际不可用（已用 Vue 3.5 SSR 复现验证）；桶管理 6 个设置页签的 `watch` 增加 `immediate`（此前永不加载、保存会用默认值覆盖真实配置）。
- **安全加固**：CORS 中间件对非白名单 Origin 的普通请求直接 403（此前仅删响应头、请求照常执行），`readJSON` 强制 `application/json`；代理 `mode=inline` 增加 MIME 白名单（恶意 HTML/SVG 强制 attachment），代理支持 `versionId` 并按版本下载、补充 `Accept-Ranges`/416；创建账号忽略客户端提交的 `id`（杜绝覆盖已有账号）；ZIP 条目名同时按 `/` 与 `\` 消毒；compose 默认仅回环发布 + 未鉴权非回环监听启动警告 + Token 过短警告；实现 `.env` 加载（自带 30 行极简解析器）。
- **正确性/健壮性**：`PurgeObject` 改 `DeleteObjects` 批量（1000/批）消除 N+1；`deletePrefix` 上限改为页前检查+跨页切分（不再超删整页）；存储类型切换映射 `InvalidRequest`→400；multipart 段号校验（1..10000 且唯一）；（copy/migrate 改为 4 路有界并发，migrate 限 10k key、failKeys 上限 200；桶属性不再静默吞错（记录日志）。
- **前端健壮性**：对象浏览/桶/回收站/迁移的列表加载增加导航序号防过期响应；上传入队时捕获所属桶（防止切换桶串桶）；迁移面板列表补传 source bucket；shift 范围选择改为从 mousedown 捕获；数据面板用 `KeepAlive` 保持状态；`catch (e: any)` 全面改为 `catch (e)` + `toErrorMessage`；ModalDialog 增加焦点陷阱与初始焦点；版本比较下载改走服务端代理（不再 `window.open` 可执行内容）。
- **测试**：handler 覆盖率 60.3% → **70.8%**，原先 0% 的主端点全部补测（列表对象/批量删除/账号 GET-PUT-DELETE/连通性/预览桶/列出桶/桶级 5 个 DELETE/CORS 拒跨域/JSON Content-Type）；CORS 拒绝行为新增断言；E2E 空标签探针改为真实断言；新增前端 vitest 测试（`versionDiff`/`format`/`storageClass`/`errors`，17 例）并提取公共 `versionDiff` 模块（修复公共后缀被标成 `context` 的语义错误）。
- **清理**：删除死代码（`s3wrap.PresignGet`、`s3api.health`/`getAccount`、`genSign`/`SignUrlDialog`）；`.gitignore` 加入根 `data/`；文档同步（API.md 补 `preview-buckets`、README SDK 接口列表补全纠错、E2E 命令改为 `-run 'TestE2E'` 全量）。
- **第二轮补漏**：AWS SDK 类型外泄清理（对象/桶/ACL 转换下沉 `s3wrap`：`FromS3Object`/`FormatBuckets`/`DescribeACL`/`GranteeLabel`，handler 不再直接依赖 `s3.ListBucketsOutput` 等）；`streamCopy` 增加 5GB 单次上传上限提示；`filename*` 改用 RFC 5987 编码；store 持久化临时文件改 `O_CREATE|O_EXCL`（防抢占/符号链接重定向）且目录 0700、掩码标记语义化（`model.IsMaskedSecret`）；PreviewOverlay 支持 Escape 关闭；对象右键菜单支持方向键导航与初始焦点；抽取 `useBucketSetting` 组合式（桶管理 6 个设置页签去重，剩各自取数/提交逻辑）；api.ts 补充 localStorage 令牌安全权衡说明。

## [20260901.2] - 2026-09-01

### 对象版本：删除指定版本 / 版本回滚
- 后端 `s3wrap` 新增 `DeleteObjectVersion`（`DeleteObject` 带 `versionId`）与 `RestoreObjectVersion`（`CopyObject` 从 `?versionId=` 复制回当前 key；版本控制下写出一条新版本，返回新 VersionId）。
- 新增端点：`DELETE /api/accounts/{id}/version`（删除指定版本）与 `POST /api/accounts/{id}/version/restore`（把某历史版本恢复为当前）。
- 「对象版本」对话框每行新增「恢复」（删除标记禁用）与「删除」操作，均带确认弹窗与结果提示，操作后自动刷新版本列表。
- 假 S3 单元测试覆盖：删除版本（传 `versionId`）、恢复版本（`X-Amz-Version-Id` 回读）、缺 key / versionId 校验 400。
- 真实 MinIO 端到端验证：恢复历史版本（`CopyObject` 带 `?versionId=`）成功并写出一条新版本、删除指定版本成功。

## [20260901] - 2026-09-01

### 桶属性 + 对象版本管理
- 后端 `s3wrap` 新增 `GetBucketLocation` / `GetBucketVersioning` / `PutBucketVersioning` / `ListObjectVersions`。
- 新增端点：`GET /bucket-info`（区域 / 创建时间 / 版本控制状态）、`PUT /bucket-versioning`（`Enabled|Suspended`）、`GET /versions`（对象各版本 + 删除标记 + 分页游标）。
- 对象浏览页位置栏新增「桶属性」：展示区域、创建时间、版本控制状态，并可一键**开启/暂停版本控制**。
- 对象右键新增「版本」：列出该对象的历史版本（最新 / 历史 / 删除标记，含 VersionId/时间/大小）。
- **真实 MinIO 端到端联调**（新增 `s3wrap` E2E 测试，`S3CLINET_E2E=1` 运行）：验证建桶、PutObject/GetObject/HeadObject、`CopyObject`、**预签名 PUT 直传**、**三段式 Multipart（预签名单段 PUT + 组装）**（12MB 组装回读一致）、对象标签、`PutBucketVersioning` + 多版本覆盖写 + `ListObjectVersions`。
- 假 S3 单元测试覆盖：桶属性（区域/创建时间/版本状态）、版本控制开关与非法状态 400、版本列表（版本 + 删除标记）。

### 代码质量（Review 驱动重构）
- **后端 handler 拆分**：`handler.go` 2186 行 → 131 行，按领域拆出 `routes.go`/`middleware.go`/`accounts.go`/`buckets.go`/`objects.go`/`copy.go`/`proxy.go`/`zip.go`/`headers.go`/`multipart.go`/`metadata.go`/`migrate.go`/`health.go`（同包，行为不变）。
- **去除冗余 `/copy` 端点**（`copyObjects`，前端未使用），保留 `copy-object` / `copy-objects` / `copy-prefix` 三件套；相应移除其单测与文档。
- **默认桶可空语义**：`BucketOrDefault()` 空时返回 `""`（不再臆造 `"default"`）；新增 `bucketOr` 助手，请求缺省桶且账号无默认桶时返回明确的 `400 bucket is required`（替代静默打到不存在的桶）。
- **`downloadZip`**：zip 条目名脱敏（路径分隔符、`..` 上跳），单次打包上限 `1000` 个对象。
- **安全强化**：新增 `Content-Security-Policy`（脚本仅同源等）；`http.Server` 增加 `ReadTimeout`（不设 `WriteTimeout` 以免截断流式大文件）；`main.go` 版本默认值同步为 `20260901`。
- **前端 ObjectsPanel 拆分**：2041 行 → **442 行的编排层**。把 11 个弹窗/覆盖层与对象列表、工具栏、上传队列、桶列表、右键菜单抽为独立 `*.vue` 子组件；再把约 900 行编排逻辑抽到 `web/src/composables/`（`useObjectBrowser` / `useObjectActions` / `usePreview`）。行为与视觉不变。
- **`proxyUrl` 去重**：抽到共享 `web/src/proxy.ts`，`ObjectsPanel` 与 `PreviewOverlay` 共用一份实现。
- **handler 测试拆分**：`handler_test.go` 1861 行 → 按领域拆成 `helpers_test.go`/`accounts_test.go`/`buckets_test.go`/`objects_test.go`/`multipart_test.go`/`metadata_test.go`/`migrate_test.go`（33 个测试函数 + 6 个助手，无重复丢失）。
- 新增 `agents.md`：TDD 优先的开发规范与验收清单（与本改动一并落地）。

## [0.16.0] - 2026-08-28

### 文件迁移优化
- **源/目标 Bucket 下拉选择**：不再手输，加载账号下全部桶（含默认桶选项）。
- **目标账号默认值**：打开面板自动预选（优先另一个账号，其次同账号=复制到其他桶/前缀），避免忘选；下拉标注「（同账号）」。
- **迁移进度**：分批执行（每批 50 个）并实时显示「迁移中 x/y」与进度条。
- **迁移结果弹窗**：成功/失败/总计统计 + **失败对象清单**（后端新增 `failedKeys` 字段，最多展示 200 个，可滚动）+ 首个错误详情 + 「**去目标账号查看**」一键跳转（切换账号并进入对象管理）。
- **已选统计**：显示已选文件数与合计大小。
- 真实 MinIO 端到端验证：同账号迁移（CopyObject 路径）成功、目标桶结构正确、清理干净；`failedKeys` 字段经真实失败场景验证。

## [0.15.0] - 2026-08-28

### 行内「⋯ 更多操作」（互联网表格习惯）
- 对象列表每行右侧新增**常显「⋯」按钮**：点击弹出与右键一致的完整菜单（文件：下载/预览/复制签名链接/复制 Key/重命名/详情/删除；文件夹：打开/复制路径/复制/移动/删除文件夹），锚定按钮下方右对齐。
- 文件行操作列精简为「下载 / 预览 + ⋯」：签名、详情等低频操作统一收入更多菜单，行更干净（其他按钮仍 hover 显示，⋯ 常显）。
- 文件夹行操作列不再空白，同样提供 ⋯ 入口。
- CDP 真实浏览器验证：⋯ 弹出菜单、菜单项完整、菜单内操作（详情）联动正常。

## [0.14.0] - 2026-08-28

### 查看对象（显式化，云控制台习惯）
- **行操作新增「预览」按钮**：hover 行即可看到「下载 / 预览 / 签名 / 详情」，预览入口不再深藏右键菜单。
- **双击 = 查看**：双击文件打开预览弹窗（图片/视频/音频/PDF/文本/代码；未知类型自动转下载），双击文件夹进入——与 OSS/COS 控制台一致。
- **Enter = 查看选中文件**：快捷键语义从「下载」改为「查看」，未知类型自动转下载；快捷键提示条文案同步。
- 预览面板本身不变（v0.9.0 起的安全代理预览，文本转义渲染、PDF 沙箱）。

## [0.13.0] - 2026-08-28

### 对象属性与生命周期（参考 OSS/COS/TOS 控制台共性）
- **编辑对象 HTTP 头**：详情弹窗「编辑 HTTP 头」——修改 Content-Type 与自定义元数据（键值行编辑、可增删）；后端 `CopyObject` 复制到自己并 `MetadataDirective: REPLACE`，保存后详情即时刷新。
- **生命周期规则**：Bucket 列表每行「生命周期」——规则管理弹窗（规则 ID/前缀/过期天数，增删后整体保存）；后端 `Get/PutBucketLifecycleConfiguration`（简化版前缀过期删除）。
  - 未配置规则正确返回空列表（兼容 `NoSuchLifecycleConfiguration` 错误码）；
  - 清空规则走 `DeleteBucketLifecycle`（空 PUT 会被 MinIO 等实现拒绝，真实环境验证后修正）。
- 假 S3 单元测试：REPLACE 指令与 Content-Type/元数据头、生命周期读写 XML、非法规则（天数/重复 ID）校验；真实 MinIO 端到端验证（set-headers 后 head 确认、规则保存/读取/清空）。

## [0.12.0] - 2026-08-28

### 控制台化：Bucket 管理（参考 OSS/COS/TOS 网页控制台共性）
- **Bucket 列表页**：对象管理首层展示 Bucket 表格（名称/创建时间/进入/删除），点击「进入」管理对象；有默认桶的账号自动进入默认桶。
- **创建 Bucket**：弹窗表单（名称 + 读写权限：私有/公共读/公共读写），后端 `CreateBucket`（自动附带地域 LocationConstraint）；创建后自动进入。
- **删除 Bucket**：确认框提示须先清空；桶非空时后端返回 409 语义化提示。
- **统计条**：对象页顶部显示当前目录对象数/总大小、选中项数与合计大小（控制台习惯）。
- **列表 / 网格视图切换**：网格卡片（类型图标/名称/大小），单击选中、双击打开、右键菜单与列表一致。
- **返回桶列表**：对象页操作栏「← 返回桶列表」。
- 后端：`POST /api/accounts/{id}/bucket`、`DELETE /api/accounts/{id}/bucket`（含假 S3 测试：创建路径/ACL 头/命名校验/删除/非空 409）+ 真实 MinIO 端到端验证（创建/列表/删除/非法名 400）。

## [0.11.0] - 2026-08-28

### UI 交互：点击内容统一弹窗化
- 新增通用 **ModalDialog** 组件（标题栏 + 关闭按钮 + Esc/遮罩关闭 + 内容滚动）。
- **对象详情**、**签名 URL 结果**：由列表下方内嵌面板改为弹窗，不再被滚动带走，查看/复制更聚焦。
- **账号新增/编辑表单**、**服务端新增/编辑表单**：由内嵌表单改为弹窗（模态操作更清晰，主流 SaaS 习惯）。
- 保留内嵌的：上传进度（需持续可见）、迁移结果（消息条足够）、Toast、确认/输入/预览弹窗（已是弹窗）。

## [0.10.0] - 2026-08-28

### UI 交互重新设计
- **对象管理工具条重构**：拆分为「位置栏」（Bucket + 面包屑 + 路径编辑 + 过滤）与「操作栏」（刷新/上级 | 上传/新建文件夹 | 全选/批量操作），分组分隔线，主操作更突出。
- **文件管理器式行交互**：单击行=切换选中（文件夹=进入），双击=打开，操作按钮（下载/签名/详情）hover 时显示（窄屏常显），行内新增「详情」入口。
- **键盘快捷键**：`Enter` 下载选中文件、`F2` 重命名、`Delete` 删除所选、`Ctrl/Cmd+A` 全选（输入框/按钮聚焦时不抢占）；首次展示可关闭的快捷键提示条（localStorage 记忆）。
- **账号快速切换**：对象管理面板头部「当前账号」下拉，直接切换账号无需回账号面板。
- **统一 busy 态**：递归复制/移动/删除文件夹等耗时操作加防重复提交，相关按钮联动禁用。
- **空状态引导**：无账号时提供「+ 创建第一个账号」按钮，一键跳转账号面板并自动打开新增表单。
- **错误提示可重试**：对象管理错误条附「重试」按钮。
- **跨面板联动**：前端直传完成 Toast 带「查看对象」动作按钮，一键跳转对象管理；Toast 组件支持动作按钮。
- 上传入口更名为「上传文件」，操作栏按钮文案精简。

## [0.9.0] - 2026-08-28

### 新增
- **常见格式安全预览**：对象管理右键「预览」按类型分发——图片（含 SVG，`<img>` 上下文脚本不执行）、视频 / 音频（原生播放器，代理支持 Range 拖动）、PDF（`sandbox` iframe 禁脚本）、文本与代码（txt/md/json/csv/log/源码等 50+ 扩展名，转义文本展示，超大文件截断提示）；未知格式提示下载。

### 安全加固（展示侧攻击面收敛）
- **预览 / 下载统一走服务端代理**（`GET /api/accounts/{id}/proxy`）：
  - `mode=download` 强制 `Content-Disposition: attachment`，恶意 HTML/SVG/JS 内容**永不进入浏览器渲染管道**；
  - `mode=text` 强制 `text/plain + nosniff` 并服务端截断（默认 1MB），前端以转义文本渲染，根除 HTML 注入 / XSS；
  - 文件名清洗（去路径分隔符 / 引号 / 控制字符），防 `Content-Disposition` 头注入。
- 移除预览面板「在新窗口打开」；签名 URL 面板提示勿在浏览器直接打开（复制分享场景保留）。
- 对象管理所有「下载」入口（行内按钮 / 右键 / 双击）改为代理下载。
- 文本内容全程 `{{ }}` 转义展示，不渲染任何 HTML/Markdown。

### 后端
- 新增 `GET /api/accounts/{id}/proxy`（download/inline/text 三模式，Range 透传，404/400 语义化）。
- 假 S3 单元测试：attachment/inline 头、Content-Type 透传、Range 转发、文本强制纯文本与截断标记、nosniff、404/400、文件名清洗；真实 MinIO 端到端验证。

## [0.8.0] - 2026-08-28

### 新增
- **上传到当前目录**：对象管理工具栏直接选择文件上传（复用 presign PUT 直传，2 路并发），内嵌进度条与逐文件状态，完成后自动刷新列表；无需再切换到「前端直传」面板。
- **面包屑地址栏可编辑**：点击 ✎ 将路径变为输入框，直接输入完整前缀回车跳转（Esc 取消），深目录直达。
- **图片预览**：右键图片文件「预览」，弹层内嵌大图展示（缩放自适应），可一键在新窗口打开原图。
- **Shift 范围多选**：按住 Shift 点击复选框，连续选中区间内所有文件（文件管理器习惯）。

### 修复与改进
- 对象管理面板工具栏布局调整：上传/刷新/上级目录分组，右侧主操作（新建文件夹）保持。
- 预览、上传进度等新组件样式与深色模式自动适配。

## [0.7.0] - 2026-08-28

### 新增
- **国内主流对象存储预设**：登录页服务商增加腾讯云 COS、华为云 OBS、火山引擎 TOS、百度智能云 BOS、京东云 OSS、七牛云 Kodo；选区域自动填充 Endpoint / 公网 Endpoint（S3 兼容域名）。
- **海外常见对象存储预设**：AWS S3、Cloudflare R2、Wasabi、Backblaze B2、DigitalOcean Spaces、Linode/Akamai、Scaleway、Hetzner。
- **服务商三行分组**：兼容 / 国内 / 国外，便于快速选择。
- **批量下载（ZIP 打包）**：对象管理选中多个文件一键打包下载；后端 `POST /download-zip` 流式打包（不落盘、不占内存），获取失败的对象写入包内 `_下载失败清单.txt`。
- **删除文件夹（递归）**：右键文件夹「删除文件夹（含全部内容）」，后端循环 `ListObjectsV2` + 批量删除，上限 10 万对象保护；空前缀拒绝（防误删全桶）。
- **复制 / 移动文件夹（递归）**：右键输入目标前缀，后端逐 key `CopyObject`（同桶目标前缀与源重叠时拒绝，防无限复制）；移动 = 全部复制成功后才删除源。
- 真实 MinIO 端到端验证：复制结构正确、ZIP 条目正确、递归删除干净。

### 后端
- 新增 `POST /api/accounts/{id}/delete-prefix`、`POST /api/accounts/{id}/copy-prefix`、`POST /api/accounts/{id}/download-zip` 三个端点。
- 假 S3 单元测试：ZIP 内容与失败清单、递归删除分页计数、递归复制计数与重叠前缀拒绝、跨桶同前缀放行。

### 修复与改进
- 前端 `requestBlob` 支持二进制响应（错误时仍解析 JSON error）。
- 对象管理工具栏新增「下载所选(ZIP)」（打包中有 loading 态）。

## [0.6.0] - 2026-08-28

### 新增
- **对象列表「加载全部」**：循环分页直到末尾（上限 200 页保护），完成后汇总提示已加载的文件/文件夹数。
- **迁移面板「列出全部文件」**：列出源前缀下**所有文件（含子目录）**，循环分页（单页 1000、上限 20 万对象），支持全选一键迁移；已用真实 MinIO 验证递归与单层列出的差异。
- **上传完成项「复制链接」**：上传完成后一键复制 1 小时签名下载链接，即传即分享。
- 剪贴板能力抽取为共享模块 `web/src/clipboard.ts`（Clipboard API + textarea 降级），对象面板与上传面板统一使用。

### 修复与改进
- 对象列表与迁移面板的批量加载均有页数上限保护，避免超大桶长时间卡死；超出上限时明确提示已加载数量。
- 迁移面板「列出全部」与「列出对象」互斥禁用，防止并发请求交错。

## [0.5.0] - 2026-08-28

### 新增
- **对象管理增强**：
  - **新建文件夹**：工具栏一键创建（服务端 PUT 空对象，key 自动补全 `/` 结尾），真实 S3 验证通过。
  - **重命名 / 移动**：右键菜单操作，输入新 Key（可含路径即移动）；后端先 `CopyObject` 成功后才删除源，复制失败不丢数据；支持跨桶移动（`newBucket`）。
  - **对象详情**：右键「详情」调 `HeadObject`，展示 Key / 大小 / 修改时间 / Content-Type / ETag / 元数据；对象不存在返回 404。
  - **批量复制签名链接**：选中多个文件一键复制 1 小时签名链接（每行一个）。
- 通用输入对话框 `PromptDialog`（自动聚焦全选、Enter 确认、Esc 取消、内置校验），与确认框视觉统一。

### 后端
- 新增 `GET /api/accounts/{id}/head`（HeadObject 详情，404 语义化）、`POST /api/accounts/{id}/mkdir`（空对象建目录）、`POST /api/accounts/{id}/rename`（copy+delete）。
- 假 S3 单元测试：head 详情字段与 404/400、mkdir 路径规范化与空 body、rename 先复制后删除及复制失败不删源、同 key 拒绝。

### 修复与改进
- 前端交互文案与空状态保持一致；详情面板与签名面板同风格（深色模式自动适配）。

## [0.4.0] - 2026-08-28

### 新增
- **深色模式**：支持「跟随系统 / 浅色 / 深色」三态循环切换（顶栏主题按钮），选择持久化；全部颜色 token 化，深色下表格、表单、弹层、Toast、骨架屏等一并适配。
- **自定义确认对话框**：替换原生 `confirm()`（删除账号/对象/服务端统一体验），支持 Esc 取消、Enter 确认、遮罩点击关闭，危险操作红色警示图标。
- **对象管理右键菜单**：文件 → 下载 / 复制签名链接（1 小时）/ 复制 Key / 删除；文件夹 → 打开 / 复制路径。
- **对象管理双击交互**（文件管理器习惯）：双击文件=下载，双击文件夹=进入。
- **对象列表列排序**：名称 / 大小 / 修改时间表头可点击排序（升/降序切换），文件夹恒置顶。
- **对象列表本地即时过滤**：输入关键字过滤当前目录已加载条目，显示命中数，一键清除。
- **记住上次选中的账号**：刷新 / 重开后自动恢复上次登录的账号。
- 上传面板新增「清除已完成」：只移除已完成项，保留等待 / 失败项便于重试。

### 修复与改进
- 顶栏新增主题切换按钮（☀️ / 🌙 图标随实际主题变化，系统主题变化时自动刷新）。
- 表格可排序表头带 hover 反馈与排序指示箭头。
- 全局按钮 / 输入框 / 表格 / 面板等硬编码颜色全部收敛为 CSS 变量，为深色主题与后续定制提供统一入口。
- 对象管理空状态区分「无对象」与「无匹配项」两种提示。

## [0.3.0] - 2026-08-28

### 新增
- **OSS 式登录**（参考阿里云 OSS Browser）：账号表单新增「服务商」选择（阿里云 OSS / AWS S3 / S3 兼容）+「区域」预设下拉（OSS 21 个地域、AWS 20 个区域，选中自动填充 Endpoint 与公网 Endpoint），保存后自动切换为当前账号；新增 `web/src/regions.ts` 预设数据。
- 对象管理：新增「下载」操作（短时效 presign GET 直接打开）；「加载更多」改为追加分页并显示已加载数量。
- 前端直传：3 路并发上传 + 「重试失败」一键重传。
- CI：新增 GitHub Actions 工作流（Go vet/test/build、Web typecheck/build、Docker 镜像构建）。
- Web：`pnpm typecheck`（vue-tsc），`pnpm build` 前置类型检查。
- Go 测试：SPA fallback、鉴权路径边界、JSON 尾部数据、copy 部分失败（假 S3）、账号文件权限。
- **`/api/health` 返回服务端版本号**（`version` 字段，ldflags 注入）；「服务端配置」连通性检测后展示后端版本标签。
- 所有响应（含静态资源）新增基础安全头：`X-Content-Type-Options: nosniff`、`X-Frame-Options: DENY`、`Referrer-Policy: no-referrer`。

### UI 改版
- **C 端视觉风格**：浅色网格渐变背景、玻璃拟态顶栏（backdrop-blur）、品牌渐变 Logo 与渐变标题、侧边栏激活项渐变胶囊、浮动卡片（大圆角 + 柔和投影）、渐变主按钮/进度条、标签页切换过渡动画、卡片入场动效、细圆角滚动条。
- 设计系统重写：语义化 CSS token、统一间距/圆角/阴影、按钮尺寸体系（`btn sm`）、表格粘性表头与选中行高亮、`focus-visible` 焦点环。
- 头部重构：服务器地址/Token 收进「⚙ 设置」弹层；新增后端连接状态指示灯（已连接/连接异常）。
- 全局 Toast 反馈（右下角、自动消失）：账号增删改、对象删除、复制、上传完成等操作结果。
- 对象管理：面包屑导航（根目录 / 逐级可点）、骨架屏加载、目录名改为可键盘聚焦的按钮、空状态带图标。
- 前端直传：拖拽上传区（点击/拖拽/键盘均可触发）、整体进度条、单文件移除。
- 账号表单：编辑时 SecretKey 占位提示「留空则保持不变」；连通性状态简化为 未测试/测试中/正常/失败。
- 迁移面板：Bucket 输入框占位显示账号默认桶；大小列格式化。
- 响应式：窄屏（<900px）侧边栏折叠为顶部横向导航；`index.html` 增加 favicon 与 theme-color。
- 主题色：品牌渐变与主色调由科技蓝切换为**薄荷绿**（`--brand` `#2dbd98→#0e8c66`、`--primary` `#10a37c`），并联动阴影、焦点环与各 tint。

### 修复与改进
- **修复跨 endpoint 流式迁移在明文 HTTP 端点失败**：AWS SDK for Go v2 1.107 默认对 `PutObject` 计算请求 CRC32 校验与 payload SHA-256，而迁移的流式 body 不可 seek，非 TLS 端点直接报错（`unseekable stream is not supported...`）。现改为 `RequestChecksumCalculation=WhenRequired`，并通过 finalize 中间件在 payload hash 计算前预置 `UNSIGNED-PAYLOAD`（与 HTTPS 下 SDK 行为一致），流式迁移恢复正常（新增假 S3 集成测试覆盖）。
- **修复 SPA fallback 失效**：静态文件未命中时此前直接 404，现对无扩展名的页面路由正确回退 `index.html`；带扩展名的缺失资源仍 404。
- **安全**：Bearer Token 校验改为常量时间比较（SHA-256 摘要 + `subtle.ConstantTimeCompare`），降低时序侧信道风险；`accounts.json` 写入权限 `0600`（含明文密钥）；`Update` 持久化失败回滚内存状态。
- **健壮性**：S3 HTTP client 增加建连/TLS/响应头超时与连接池上限（不设整体超时，避免截断大文件流式迁移）。
- 前端「下载」由 `window.open` 改为程序化锚点点击，避免异步请求后被浏览器弹窗拦截（新标签页无法打开）。
- `/api/accounts/{id}/copy` 与 migrate 行为对齐：逐 key 复制、失败继续，响应新增 `failed`/`lastError` 字段。
- 鉴权路径判断精确化（`/api` 或 `/api/` 前缀），不再误伤 `/apiary` 之类路径；JSON 请求体拒绝尾部多余数据；健康检查日志降为 debug。
- 日志状态记录器忽略多余的 `WriteHeader` 调用，避免状态与实际响应不一致。
- 前端：对象列表分隔符默认 `/`（与占位提示一致，目录可浏览）；全选 checkbox 状态修正；迁移列表大小格式化；GET 请求不再携带多余 `Content-Type`（避免跨域预检）。
- 重构：新增 `web/src/format.ts` 共享格式化工具（`fmtSize`/`fmtDate`），对象/直传/迁移三个面板去重。
- 构建：Dockerfile 拷贝 `pnpm-lock.yaml` 并改 `--frozen-lockfile`（可复现）；Go 构建通过 ldflags 注入版本号；Makefile 补全 `.PHONY` 并新增 `test`/`vet`/`web-typecheck`/`docker` 目标；移除 pnpm 11 已忽略的 `package.json#pnpm` 字段。
- 测试：新增 config 包测试（默认值/覆盖/列表解析）、健康检查版本与安全头断言、错误 Token（含长度不同）鉴权用例、migrate 跨 endpoint 流式复制假 S3 集成测试。

## [0.2.0] - 2026-08-22

### 新增
- 服务端集中配置（`internal/config`），支持 `S3C_*` 环境变量与 `.env`（见 `server/.env.example`）。
- **安全加固**：默认绑定回环 `127.0.0.1`；CORS 白名单（`S3C_CORS_ORIGINS`）；可选 Bearer 鉴权（`S3C_TOKEN`）。
- **容器化**：多阶段 `server/Dockerfile`（web + Go + `debian:bookworm-slim` 运行时，非 root `app`，自带 `HEALTHCHECK`，`/data` 已授权）；`docker-compose.yml`（server + MinIO）；构建参数 `GOPROXY`/`NPM_REGISTRY` 可覆盖；新增 `/s3clinet-server -healthcheck` 自检子命令。
- Go 单元测试：store / s3wrap / handler（CRUD、持久化、脱敏、原子写、CORS 策略、鉴权、迁移端点判断）。
- 文档：REST API 参考（`docs/API.md`）、配置矩阵、运行/部署说明、容器直传可达性说明。

### 修复与改进
- 修复跨 provider 迁移误用 `CopyObject`：`sameEndpoint` 仅在两端点一致且非空时判定为同端点，否则流式复制。
- 跨 endpoint 迁移保留源对象 `Content-Type` 与元数据。
- 预签名有效期限制在 1s–7 天；对象 `maxKeys` 限制 1–1000。
- 账号更新增加必填校验，避免清空 endpoint/accessKey。
- 账号存储改为原子写（临时文件 + rename）；`Create` 写盘失败回滚内存状态。
- 账号缺省 region 为空时回退 `us-east-1`。
- 前端：上传队列直接持有 `File` 引用（避免同名误匹配）；设置区新增 Token；签名复制支持降级；对象列表加载状态。

## [0.1.0] - 2026-08-22

### 新增
- 首个版本：Go 后端（AWS SDK for Go v2，封装 11 个 S3 接口）、Vue3+Vite+TS 前端（账号/列对象/直传/签名/删除/迁移/加前缀）、Tauri 2 桌面壳（无 IPC，B/S）。
