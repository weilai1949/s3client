# 全仓代码质量评审（2026-10-09）

> **状态**：**活跃文档**（非归档）。结论绑定提交 `e965e54`（评审期间工作区的在途改动已于评审结束前
> 落为 `5f0d555`——该提交只改 Go 工具链版本、`e2e-real` 用例与文档，**未触碰本报告涉及的任何代码与门禁**）。
> 文中 `file:line` 与门禁数字均为**评审时点快照**；未闭环项见 §4/§5，全部闭环后按
> [`archive/index.md`](archive/index.md)「归档操作」四步 `git mv` 冻结。
>
> **范围**：全仓代码——`apps/server` Go 后端（288 个 `.go`，生产约 1.6 万行）、`apps/web` Vue 3 + TS
> 前端（143 个 `.ts` + 40 个 `.vue`，生产约 2.1 万行）；含门禁、CI 与文档一致性，不含文档文风评审。
> **方法**：机械化门禁**独立复跑** + 6 路五轴深潜（正确性 / 可读性 / 架构 / 安全 / 性能）+ 关键结论回到
> 源码逐条复核（含独立复算 CRC-64/NVME 与两组受控实验：`apps/web/node_modules` 有无、浅克隆 tag 数）。
>
> **结论**：**Request changes**。代码质量显著高于同类项目（分层干净、安全基线扎实、门禁工程罕见地严格，
> 自研 CRC-64/NVME 经独立复算正确），但存在一条**阻断级**问题：仓库自己宣称的「全绿门禁」在 CI 上
> **无法复现**——两个 CI 平台的 Go job 都是「构造性红灯」，且被本地环境掩盖。

## 1. 机械门禁实测（评审时点，全部独立复跑）

| 门禁 | 结果 | 说明 |
|---|---|---|
| `gofmt -l` / `go vet ./...` / `go build ./...` | 0 输出 / exit 0 / exit 0 | 干净 |
| `go test ./...`（10 包） | **10/10 PASS** | 在「真 `.git` + 已装 `apps/web/node_modules`」的等价 checkout 上 |
| Go 覆盖率（`count==0` 块检查） | **100.0%，零未覆盖块** | 与提交信息一致 |
| `golangci-lint run` | **0 issues** | |
| `pnpm lint` / `typecheck` / `typecheck:e2e` / `build` | 0 告警 / exit 0 / exit 0 / OK | |
| `pnpm test` | **82 文件 / 1272 例全过** | |
| `pnpm test:coverage` | **100%（4849 / 3227 / 1229 / 4195）** | |
| `node scripts/gen-api.mjs --check` | exit 0 | 生成物与 `docs/api/openapi.json` 同步 |
| **GitHub CI `ci.yml` server job** | **必然失败** | §3 C1 + C2，受控实验复现 |
| **GitLab CI `server` job** | **必然失败** | §3 C1 |
| `make e2e-real` / `S3CLIENT_E2E=1 go test ./internal/s3wrap/` | **未复跑** | 需 docker + RustFS；见 §7 |

> 基线与本机限制：`5f0d555` 把 `apps/server/go.mod` 升到 `go 1.26.9`，而本机工具链为 go1.26.6 且
> 1.26.9 的工具链下载不可达（`go version` 在模块内超时）——因此主工作区**无法直接跑 Go 门禁**，
> 上述 Go 数字在「同内容、`go.mod` 回退到 1.26.6」的等价 checkout 上取得。这本身不影响结论：
> 代码内容未变。

## 2. 处置状态总览

| 编号 | 级别 | 一句话 | 状态 |
|---|---|---|---|
| C1 | Critical | CI Go job 缺 `apps/web/node_modules` 前置 → `agent_evals` 门禁自毁 | ✅ 已修（本批次） |
| C2 | Critical | GitHub checkout 无 `fetch-depth` → `changelog_tag` 门禁在浅克隆上必红 | ✅ 已修（本批次） |
| R1 | Required（安全） | 前缀作用域 token 可读/取消**任意**迁移任务 | ✅ 已修（本批次） |
| R2 | Required（安全） | `preview-buckets` 绕过作用域，成为认证后的 SSRF 跳板 | ✅ 已修（本批次） |
| R3 | Required（正确性） | cron 的 DST 处理：春季跳变错时/漏跑、秋季重复小时漏跑 | ✅ 已修（本批次） |
| R4 | Required（正确性） | trash purge 把所有 S3 API 错误映射成 409 | ✅ 已修（本批次） |
| R5 | Required（正确性） | 计划/任务清单「先快照后落盘」可让旧快照覆盖新状态 | ✅ 已修（本批次） |
| R6 | Required（正确性） | 前端 4 个 bucket 加载器竞态 + 1 处 `loading` 卡死 | ✅ 已修（本批次，落地口径见 §8 回写） |
| R7 | Required（规范） | 3 个死类型导出（违反 AGENTS.md 硬约束 #5） | ✅ 已修（本批次，连带补类型维度门禁） |
| R8 | Required（契约） | `multipartParts` 手写路径绕过生成契约 + 注释失效 | ✅ 已修（本批次） |
| R9 | Required（测试） | e2e OpenAPI 断言不可能失败 | ✅ 已修（本批次，落地口径见 §8 回写） |
| R10 | Required（工程） | 门禁清单/CI 路径过滤/文档漂移（详见 §5 O11） | ✅ 工具链残留已解除（本机 go1.26.9）；门禁清单/路径过滤/文档漂移半边并入 O11（#82） |
| O1–O12 | Optional | 载荷完整性、契约漂移、重复实现、可访问性、i18n 等 | ⬜ 已登记（[`KNOWN_ISSUES.md`](KNOWN_ISSUES.md) #72–#83，待排期） |

> 状态由 **2026-10-09 修复批次**回写（逐条落地与例外口径见 §8 末尾「本批次回写」）；§1 门禁数字仍为评审时点快照，不改写。

## 3. Critical（阻断合并）

### C1. CI 的 Go job 缺 `apps/web/node_modules` 前置 → 门禁自毁

`TestAgentEvalsGoldenTaskSetIsMachineReadable` 对 `scripts/evals/golden-tasks.yaml` 里每个
`requires_path` 做 `os.Stat`，缺失即 `t.Errorf`（`apps/server/agent_evals_gate_test.go:279-283`）；
GT-4 的四个判据（`golden-tasks.yaml:196/201/206/211`）都要求 `apps/web/node_modules`。
而 GitHub 的 `ci.yml` `server` job（`.github/workflows/ci.yml:17-73`）与 GitLab 的 `server` job
（`.gitlab-ci.yml:214-244`，`extends: .go-base`，`before_script` 只有 `go mod download`）
**都不安装 web 依赖**，却执行 `go test -race ... ./...`。

受控实验（同一 checkout，删除/恢复软链）：

```text
无 apps/web/node_modules → FAIL github.com/.../apps/server 0.005s（agent_evals_gate_test.go:281）
有 apps/web/node_modules → ok   github.com/.../apps/server 0.006s
```

**后果**：提交信息里「`go test ./...` 10/10 包全绿」是本地观测，两个 CI 平台的必备检查都是红的。
**建议修复**：给 Go job 补真实前置（`pnpm install --frozen-lockfile`，复用 pnpm 缓存），
或把该门禁的「前置存在性」判据与「当前环境是否提供 web 工具链」解耦（例如按 `requires_path`
声明其提供方，未提供的环境跳过该条而非 `Errorf`）——**不要**用 `mkdir` 伪造目录。

### C2. GitHub CI 另有一处红灯：`changelog_tag_gate` 依赖 tag，而 checkout 是浅克隆

`apps/server/changelog_tag_gate_test.go:68-101` 只从 `.git/refs/tags/*` 与 `.git/packed-refs`
读 tag，少于 15 个即 `t.Fatalf`（`:187-189`，`minChangelogTags=15`）；而全部 18 处
`actions/checkout` 都是裸用，**没有任何 `fetch-depth`/`fetch-tags`**
（`grep -rn "fetch-depth\|fetch-tags" .github/` 为空）。用 `git clone --depth=1 --no-tags`
复现：`refs/tags` 0 项、`git tag` 0 个。
**建议修复**：`with: { fetch-depth: 0 }`（或 `fetch-tags: true`）。GitLab 侧是全量克隆，不受影响。

## 4. Required

### R1. 作用域 token 可越权读/取消任意迁移任务（安全）

`apps/server/internal/handler/scope.go:296-305` 只为 `/api/schedules/{id}` 注入桶引用（注释明确写了
「否则前缀作用域 token 可触发/删除越界计划」），但 `routes.go:95-98` 的
`GET /api/migrate/jobs`、`/{id}`、`/{id}/cancel`、`/{id}/events` **没有任何注入**：
`requestScopeRefs` 返回空 → `authorize` 的循环空转放行。`jobs.go:30` 全量列举，
`migrate_async.go:53-68` 无归属校验即 `job.Cancel()`。
**影响**：仅授权 `prefixes:["bucket-a"]` 的 token 可枚举全部任务、读取别桶进度与 `failedKeys`
（对象键名）、取消他人进行中的迁移。
**建议修复**：任务记录携带 source/target 桶并在 status/cancel/events 校验，或对带作用域的 token
直接拒绝这些路由；补行为级回归测试（作用域 token 取消他桶任务 → 403）。

### R2. `preview-buckets` 绕过作用域，成为认证后的 SSRF 跳板（安全）

`scope.go:99-108` 把该路径的 account id 记为 `""`（跳过 accounts 约束），body 不主动带 `bucket`
时 refs 为空（跳过 prefixes 约束）；`accounts.go:138-153` 用**调用方自带**的
endpoint/credential 直接 `s3wrap.New` + `ListBuckets`，而 `S3C_SSRF_DENY_PRIVATE` 默认 false
（`config.go:151`，ADR-003 自托管放行）。`scope_test.go:193-195` 把「不被当作越界账号」钉成了
期望行为——即豁免是有意的，但**未加任何作用域闸**。
**建议修复**：对已设任何作用域（`Readonly` / `Prefixes` / `Accounts`）的 token 拒绝该端点，
或让它走 SSRF 白名单策略。

### R3. cron 的 DST 处理有两处真实缺陷（正确性）

`apps/server/internal/service/schedule_cron.go:225-232` 用 `time.Date(...)` 构造候选；注释
（`:222-224`）称间隙时刻会「落到跳变后的合法时刻」，实测**相反**（我用独立探针复现，
另一路差分实现结论一致）：

| 表达式 | after | 代码返回 | 正确行为 |
|---|---|---|---|
| `30 2 * * *` | NY 2026-03-08 01:00 EST | **01:30 EST（小时=1，违反表达式）** | 03:30 EDT 或跳过 |
| `30 2 * * *` | NY 2026-03-08 01:45 EST | **当天不跑**（候选 01:30 已过期，跳到次日） | 当天应触发一次 |
| `30 1 * * *` | NY 2026-11-01 01:45 EDT | **2026-11-02 01:30 EST（漏跑）** | 2026-11-01 01:30 EST（45 分钟后） |

**影响**：按非 UTC 时区的一次/日备份，每年一次错时或漏跑。
**建议修复**：在绝对时间轴上构造候选并回验本地字段等于表达式（或显式限定 UTC 并加门禁断言）；
同步修正注释并补两例 DST 测试。
> 解析本体（区间 / 列表 / 步进 / dom-dow OR 语义 / 不可能表达式）经 4000 例差分测试**零偏差**，
> 问题只在 DST。

### R4. trash purge 把所有 S3 API 错误映射成 409（正确性）

`apps/server/internal/handler/trash.go:73-79`：`if s3wrap.IsAPIError(err) { 409 ... }` →
`AccessDenied` / `NoSuchBucket` / `SlowDown` 全变 409 Conflict；而
`s3wrap/errors.go:128-139` 已有正确映射表。更糟的是 `handler/ol_trash_test.go:85-88` 把这个
错误行为写成了断言（同一 fake 上 `GET /trash` 的 AccessDenied 却是 403，自相矛盾）。
**建议修复**：只把 `ObjectLocked` → 409，其余走 `writeInternalErr`；同步修测试。

### R5. 计划/任务清单「先快照、后落盘」可让旧快照覆盖新状态（正确性）

`apps/server/internal/service/scheduler.go:119-127` 与 `job.go:157-162`：快照在锁内生成、
`Save` 在锁外执行。persister 自带的 mu 只串行化写、不保证**顺序**：P1 拿旧快照 S1 → 状态更新 →
P2 拿新快照 S2 并先写 → P1 后写 S1 → 磁盘回退到旧状态（进程重启后丢一次更新）。内存态正确，
普通测试发现不了。
**建议修复**：用一个独立 persist mutex 把「生成快照 + 写盘」整体串行化（仍不占 `sc.mu` / `r.mu`）。

### R6. 前端 4 个 bucket 加载器竞态 + 1 处 `loading` 卡死（正确性）

`BucketsPanel.vue:47-70`、`StorageReportPanel.vue:23-40`、`RecycleBinPanel.vue:81-95`、
`AccountsPanel.vue:137` 都没有请求代际守卫，而 `useObjectBrowser.ts:139-157` 与 `MigratePanel`
已有规范的 `loadSeq`。切账号时两次请求乱序返回 → A 的桶渲染在 B 的账号下；`RecycleBinPanel`
同时只有 try/catch、**无 finally** → 异常时 `loadingBuckets` 永久为 true。
**建议修复**：抽 `useBucketList(accountIdRef)` 复用 `loadSeq` 口径。

### R7. 死类型导出（违反 AGENTS.md 硬约束 #5）

`apps/web/src/types.ts:37-39` 的 `StorageClassUsage` / `PrefixUsage` / `StorageRecommendation`
全仓零引用（含本文件），而前端 `deadcode_gate.test.ts:26-28` 明确把「类型导出」列为盲区，
`vue-tsc` 也不报未使用导出 → 硬约束在类型维度上无机械保证。**建议修复**：删除或真正使用。

### R8. 生成契约被绕过 + 过期注释（契约）

`apps/web/src/api/endpoints.ts:241-248` 手写 `/api/accounts/${id}/multipart/parts`，注释称
「操作 id 要等 Lead 重新生成后才进入联合类型」，但 `operations.ts:70` 早已有 `multipartParts`
——注释失效且该端点绕过 `opPath` 契约层。**建议修复**：改走 `opPath('multipartParts', { id })`，
删掉过期注释。

### R9. e2e 断言不可能失败（测试）

`apps/web/e2e/smoke.spec.ts:19-31` 的 OpenAPI 用例接受 `[200,404,500,502,503]`，仅在 200 分支
校验 content-type——端点彻底坏掉也绿。**建议修复**：在真实后端模式下断言 200 + JSON，或把该用例
改为条件跳过而非「接受一切」。

### R10. 工作区（评审时点）工具链与文档不一致（已在 `5f0d555` 基本解除）

评审进行中，工作区曾有 `go.mod` 升 1.26.9、而 `Dockerfile` 注释仍写「go.mod 要求 1.26.6」的
不一致；该批次已于 `5f0d555` 提交（注释同步、四处对齐）。**残留提醒**：该提交把 `go.mod` 提到
1.26.9 后，本机（1.26.6 且工具链下载不可达）**跑不了 Go 门禁**——请在 CI 与本地都确认
`golang:1.26.9-*` 与工具链可用，否则门禁形同虚设。

## 5. Optional

| 编号 | 位置 | 问题与建议 |
|---|---|---|
| O1 | `s3wrap/client.go:120-143` | `UNSIGNED-PAYLOAD` **无条件**注入数据面，且 `RequestChecksumCalculation=WhenRequired`：PutObject/UploadPart/Copy 的载荷既无 SigV4 完整性也无 SDK 校验和；对已放行的 `http://` endpoint 可被中间人改写。建议按「带 body 的操作」收窄，或对 http endpoint 强制显式开关，并在 [`threat-model.md`](threat-model.md) 记一笔。 |
| O2 | `s3wrap/s3wrap_dto.go:23/40/55/69` | 4 个导出函数签名带 AWS SDK 类型，但只在 s3wrap 内调用 → 降为小写，避免给下一个 handler 作者留下「合法」的 SDK 依赖入口。 |
| O3 | `web/src/api/http.ts:47-61` | 传输层丢弃 HTTP status（只拼进 message），调用方无法区分 401/403/404/409/412/501——而本轮新功能正需要（Object Lock 409 vs 501、copy 412）；`res.json() as Promise<T>` 对不可信响应零校验。建议 `ApiError{status, body}` + 边界校验（或从 `operations[...]['responses'][200]` 派生类型）。 |
| O4 | `web/src/api/generated.gate.test.ts:38` | 只有 `>= 60`（实际 84），无「`s3api` 覆盖 `operations` 全部 opId」的穷尽性断言 → 删掉一个 spec 路径再重新生成仍全绿。 |
| O5 | 前端 7 文件 | `newRowKey` 在 TagsDialog / BucketTags / HeadersDialog / BatchMetadataDialog / LifecycleDialog / BucketCors / BucketPolicyVisualEditor 逐字复制 7 份；虚拟滚动管线在 ObjectList / MigratePanel / VersionsDialog / RecycleBinPanel 重复 4 份。各自应有一个 canonical helper。 |
| O6 | `web/src/components/ObjectContextMenu.vue:31`、`ObjectList.vue:234` | 菜单关闭后焦点丢失（无 `previousFocus` 还原）、触发器缺 `aria-haspopup`/`aria-expanded`；`useObjectBrowser.ts:514-515` 两套 window keydown，Escape 绕过 `useKeydownStack` 的 LIFO 语义。 |
| O7 | `web/src/i18n/coverage.test.ts:75` | 键覆盖正则看不见数据驱动键与模板键（`storageReport.kind.${kind}`）；当前 12 个间接键恰好都已定义（潜在风险）。另有 6–7 处硬编码可见文案（`AccessKey ID/Secret`、`BatchMetadataDialog.vue:185` 的「替换」等）。 |
| O8 | `handler/openapi_register_*.go` | required/状态码漂移**无门禁**：`putObjectLock` 声明只要求 `bucket+mode`，但 handler 还要求 days/years 之一；`bucket` 在部分对象端点标 required、在 schedules 处又明确不标；migrate jobs 的 404、trash 的 409、copy 的 409、proxy 的 400/416 均未声明。 |
| O9 | `handler/schedules.go:136/155` | `writeErr(400, err.Error())` 回显服务层错误（含用户 cron 原文）；`error_echo_gate_test.go:24` 只匹配 `"字面量"+var`，`err.Error()` 形式从缝里漏过。 |
| O10 | `handler/scope.go:57-62` | `accounts:[A]` 的 token 仍可 `GET/POST /api/accounts`（无 `{id}` 即跳过约束）。测试已写成「预期行为」，但按最小权限语义是越权面，建议至少在文档明确。 |
| O11 | Makefile / docs / CI | `make check`（`Makefile:111`）缺 `govulncheck`、`pnpm lint`、`go build ./...`；`e2e.yml` 的 paths 过滤使只改 `internal/handler/**` 的 PR 跳过真 RustFS 的 Go E2E；`docs/DEVELOPMENT.md:151` 的 `perf-budget` job 名不存在、`:211` 低估前端覆盖率排除项、`:179` job 计数差一。 |
| O12 | 散点 | `wrapObjectTooLarge` 漏包两处（sentinel 不成立）；`PurgeObject` 遮蔽 `out`；`ssrf.go:142` 可返回 `(nil,nil)`；`store.go:128/136`、`sqlite.go:56` 吞 `encryptAESGCM` 错误；`atomicfile.go:45` 固定 `.tmp` + `Save` 错误被丢弃；`storage_report` 金额未取整；`Scheduler.List` O(n²) 选择排序；**损坏的 `schedules.json` 静默丢弃后被空列表覆盖**（`scheduler.go:99-114`）；**非法 cron 让计划永久停摆却不写 `LastError`**（`:285-293`）；`Job.Total` 存在潜在无锁读。 |

## 6. 值得肯定（经独立复核）

- **CRC-64/NVME 自研实现正确**：`reflect64(0xad93d23594c93659)==0x9a6c9329ac4bc9b5`，与 Go 生态
  参考实现的反射多项式一致；`check("123456789")==0xAE8B14860A799888`（我用两套独立实现复算一致）；
  分块/整体等价有测试、真实 RustFS E2E 绑定了对端返回值。零新增依赖手写 CRC 而做到位，少见。
- **安全卫生**：SecretKey 从不落 localStorage / 日志 / URL（`Omit<ServerProfile,'token'>` 把持久化
  写成类型错误）；生产代码零 `v-html`；CSP / XFO / Referrer-Policy / nosniff；常量时间 token 比较；
  XFF 仅信可信代理且取最后一段；SSRF 在 Dial 时复验；错误按 code（非文本）映射；参数化 SQL，
  唯一 `Sprintf` 是常量 PRAGMA。
- **工程化门禁**：21 个 `*_gate_test.go` 全部**没有 `t.Skip`**、绝大多数有「扫描面自检」，
  `deadcode_gate_test.go` 用 AST 抓 `_ = x` 消音并对检测器本身做合成源码用例——这是我见过最认真的
  门禁族。文档契约（链接 / 锚点 / 命名 / 索引 / 计数）双向机械校验。
- **架构**：`store → model → s3wrap → handler` 干净，生产 Go 文件全部 < 700 行（上限 1000），
  AWS SDK 类型不外泄，条件写边界校验 + 412/409 映射集中在 `errors.go`，异步/计划任务共用一条链路
  并做了 check-and-set 防叠加。
- 未复现任何「覆盖率掩盖的死代码」或「未定义标识符」：`golangci-lint` 0 issues、`vue-tsc` 干净、
  前端覆盖率四指标 100% 且未排除业务文件。

## 7. 未覆盖面与不确定性

- `make e2e-real` / `S3CLIENT_E2E=1` 真 RustFS E2E **未复跑**（需 docker + RustFS）；且评审时点
  另一写入方正在改 `apps/web/e2e-real/real-backend.spec.ts`，此时运行会得到误导性结果。
- §3 的 CI 结论是**读配置 + 受控实验**推出的，未在 GitHub Actions 真机上执行；
  `golang:1.26.9-*` 镜像与工具链是否存在离线不可验。
- 前端未逐行读的渲染路径：`BucketPolicyVisualEditor`、`CompareDialog`、`BucketWebsite`、
  `BucketObjectLock`、`BucketEncryption`、`CreateBucketDialog` 等（以 grep + 测试存在性覆盖）。
- 后端 `sync` / `delete` / `batch` / `migrate` / `zip` 等 service 文件为抽样覆盖；
  未执行 `-race` 全量（CI 有，但当前 CI 红）。

## 8. 处置计划

1. 修 CI（C1 + C2）——CI 绿之前，其余改动缺少可信基线；
2. 修作用域缺口（R1 / R2）与 trash 409（R4）——小改动、大风险面，均补行为级回归测试；
3. 修 cron DST（R3）——优先「绝对时间轴 + 本地字段回验」，修正注释；
4. 修落盘乱序（R5）与前端竞态（R6），补并发 / 乱序回归测试；
5. 清死类型（R7）、契约绕过（R8）、不可失败断言（R9），再按 §5 表收口结构性重复与文档漂移；
6. 未闭环项在收口后登记进 [`KNOWN_ISSUES.md`](KNOWN_ISSUES.md)（开放技术债）/
   [`ROADMAP.md`](ROADMAP.md) §三（排期项），并同步 [`../CHANGELOG.md`](../CHANGELOG.md)
   `[Unreleased]`；本报告在引用收敛后按 [`archive/index.md`](archive/index.md)「归档操作」冻结。

**本批次回写（2026-10-09，同日并行批次之一）**：

- **C1–C2 + R1–R9 全部修复**，每项附行为级回归测试；全量门禁复跑全绿——`gofmt` 干净 / `go vet`
  0 告警 / `golangci-lint` 0 issues / `go test ./...` **10/10 包** / `make test-cover`（`-race`）
  **100.0% statements + `count==0` 零块** / `govulncheck` 0 可达；前端 `pnpm lint` 0 告警 /
  `typecheck` + `typecheck:e2e` exit 0 / `test:coverage` **82 文件 1283 例、四指标 100%
  （4879 / 3251 / 1229 / 4213）** / `pnpm build` OK；mock Playwright **21 passed + 1 skipped**
  （跳过项即 R9 的条件跳过）；`make e2e-real` 真后端 + 真 RustFS **5 passed**（含新增
  「OpenAPI 规范 200 + JSON spec」真断言）。
- **O1–O12** 按第 6 条全部登记为 [`KNOWN_ISSUES.md`](KNOWN_ISSUES.md) **#72–#83**——12 条均属
  技术债 / 缺陷，按「同一事项只登记一处」（[`DEVELOPMENT.md`](DEVELOPMENT.md) §4）不进 ROADMAP §三；
  ROADMAP §四门禁基线已同步本轮实测值。
- **R10**：工具链残留解除——CI 侧随 `5f0d555` 收口，本机已装 `/usr/local/go1.26.9`（与 CI 同版本），
  本批全部 Go 门禁在该版本下复跑；表内 R10 的「门禁清单 / CI 路径过滤 / 文档漂移」半边内容并入
  O11（#82）登记。
- **三处与建议不完全一致的落地口径**（评审原文保留、不改写）：
  - **R6**：原句「`RecycleBinPanel` 只有 try/catch、无 finally → `loadingBuckets` 永久 true」与现场
    不符——该文件此前并无 `loadingBuckets` 标志（桶选择器也不随请求禁用）。落地为「新增
    `loadingBuckets` + `:disabled` + finally 复位」，测试钉住「当前请求异常必须复位 loading」；
    亦未抽 `useBucketList` composable（`AccountsPanel` 依赖 `editingId` / `form`，统一签名不适配），
    改为四面板各自内联 `loadSeq` 守卫。
  - **R9**：采建议的「条件跳过」方案，但真断言不消失——静态预览无后端（确定性 502/503）时跳过，
    其余状态严格断言 200 + JSON + openapi 字段；常驻真断言迁入
    [`../apps/web/e2e-real/real-backend.spec.ts`](../apps/web/e2e-real/real-backend.spec.ts)
    （`scripts/e2e-real.sh` 补 `S3C_EXPOSE_OPENAPI=1`——生产默认 404 不暴露规范）。
  - **R7**：除删 `types.ts` 三个死类型外，连带把 `deadcode_gate.test.ts` 声明的「类型导出」盲区
    升级为真断言（原「类型维度无机械保证」一并收口）；新口径扫出的生成物
    `operations.ts → Operation` 零引用由生成器改 `as const satisfies Record<string, Operation>`
    真正消费，而非删除契约。
- **第 7 条的两条未覆盖面已补**：`make e2e-real` **5 passed**（评审时点未复跑）；`-race` 全量经
  `make test-cover` 复跑全绿（「当前 CI 红」由 C1/C2 修复收口，CI 真机结果随下次流水线确认）。
- 本报告仍为活跃文档（状态表已回写），按第 6 条在引用收敛后执行
  [`archive/index.md`](archive/index.md)「归档操作」冻结。
