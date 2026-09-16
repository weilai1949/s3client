# s3clinet 路线图（Roadmap）

> 本文件描述 s3clinet 的**版本规划与优先级**（战略层）：每个里程碑的目标、验收标准与包含的条目。
> 逐条待办的**唯一来源**仍是 [`docs/todolist.md`](docs/todolist.md)（战术层）；已实现 / 已修复台账见
> [`docs/FEATURES.md`](docs/FEATURES.md)；逐字发布历史见 [`CHANGELOG.md`](CHANGELOG.md)；
> 评分与问题证据见 [`docs/ASSESSMENT.md`](docs/ASSESSMENT.md)（2026-09-16 五维度评估：代码质量 82 /
> 漏洞 72 / 死代码 70 / 服务降级 74 / 自我迭代 90，总分 78）。
>
> 状态图例：⬜ 未开始 · ⏳ 进行中 · ✅ 已完成 · ➖ 已决策（不做 / 维持现状）
>
> 版本命名：稳定里程碑 **v1.0.0** 后日常发版用时间戳（`v1.0.0-YYYYMMDDHHmmss`），预发布用 `v1.0.0-rcN`。
> 当前版本 **`v1.0.0-rc1`**。最后更新：2026-09-16。

## 目录

- [一、当前定位](#一当前定位)
- [二、里程碑总览](#二里程碑总览)
- [三、v1.0.0-rc1：候选版收口](#三v100-rc1候选版收口)
- [四、v1.0.0：首个稳定版](#四v100首个稳定版)
- [五、v1.0.x：可靠性加固](#五v10x可靠性加固)
- [六、v1.1.0：体验与性能](#六v110体验与性能)
- [七、v1.2+ / 长期](#七v12--长期)
- [八、质量门禁基线](#八质量门禁基线)
- [九、风险与依赖](#九风险与依赖)
- [十、维护约定](#十维护约定)

---

## 一、当前定位

**产品形态**：S3 兼容对象存储客户端，Web 端 + Tauri 2 桌面端（B/S 架构、无 IPC、全 HTTP）；Go 后端
（AWS SDK for Go v2）+ Vue 3 / Vite / TS 前端，70 个 `/api/*` 端点、OpenAPI 3.0.3 自动生成。

**结论**：工程质量显著高于平均水平，**已具备发布条件**：2026-09-16 评估发现的 4 项发布前必修缺陷
（P0）已全部修复并提交（`e03a15e`），并归档至 [`docs/FEATURES.md`](docs/FEATURES.md)「H」段与
[`docs/todolist.md`](docs/todolist.md)。因此路线图当前的第一优先级不是加功能，而是**关闭 P1
（供应链 / 契约可信 / 异步任务不丢），把候选版收口成可信的稳定版**。

**已提交并归档的 P0 修复**（均为 2026-09-16 评估发现；代码提交 `e03a15e`，文档归档 `d9d0bd7`）：

| 评估编号 | 内容 | 状态 | 验证证据 |
|---|---|---|---|
| H1 | OpenAPI 契约对齐真实 handler（migrate 字段名 / presign method 枚举 / delete 去掉 `versionId` / multipart 补 `expiresIn`） | ✅ | 新增 `TestOpenAPI_ContractRequestBodyMatchesHandlers` 全绿 |
| H4 | 前端 token 移出 `localStorage['s3c.servers']`（含旧数据一次性迁移，仅存 `{id,name,base}`） | ✅ | `api.test.ts` 130 例全绿 |
| M8 | `loadAll()` 至少执行一次请求（do-while），失败不再误报「已加载全部」 | ✅ | `useObjectBrowser.test.ts` 75 例全绿 |
| S7 | SSE EOF 终态检测：轮询回读 job 状态直到终态 / 连续失败快速 `onError`，消除 Promise 悬挂 | ✅ | `api.test.ts` 轮询与 onError 用例全绿 |

> 注：H1 收尾时遗留 1 处未对齐——`POST /api/accounts/{id}/delete-marker/restore` 契约声明
> `deleteMarkerId`，真实 handler 解析 `versionId`（`metadata.go:373-404`），契约测试亦未覆盖该端点。
> 该项已并入 §四「契约与文档一致性」#10 收尾，修复后关闭。

---

## 二、里程碑总览

| 里程碑 | 主题 | 关键验收 | 依赖 |
|---|---|---|---|
| **v1.0.0-rc1**（当前） | 候选版收口：P0 四项缺陷清零 | ✅ 四项 P0 已合并入 `develop`（`e03a15e`）；门禁全绿 | — |
| **v1.0.0** | 首个稳定版：P1 清零 + 契约可信 | Go 1.26.6；无幽灵 SHA；异步任务可恢复；契约测试覆盖全部端点 | rc1 |
| **v1.0.x** | 可靠性加固：存储 / 可观测性 / 安全审计 | 密钥落盘加密；S3 上游指标；流式错误可见 | v1.0.0 |
| **v1.1.0** | 体验与性能：长列表、错误恢复、i18n | grid 窗口化；健康轮询自动恢复；门禁消除注水 | v1.0.x |
| **v1.2+** | 长期：桌面分发、可选增强 | 桌面端签名与自动更新；按需评估 | v1.1.0 |

**发布约束**：任一里程碑发布前，[质量门禁基线](#八质量门禁基线)必须全绿；P0 未清零不得进入 v1.0.0。

---

## 三、v1.0.0-rc1：候选版收口

**目标**：把评估发现的 4 项「发布前必修」（ASSESSMENT §六 P0）落地、提交、验证，形成可发布的
`v1.0.0` 前置基线。**不含新功能**。

> **状态：已收口（2026-09-16）**。四项均以提交 `e03a15e` 合并入 `develop`，门禁全绿
> （`go vet` / `go test -race` / `pnpm lint` / `vue-tsc` / 前端 966 测试 + 覆盖率 100%）。
> 逐条归档见 [`docs/FEATURES.md`](docs/FEATURES.md)「H. 2026-09-16 评估 P0 发布阻塞修复」。
> 仅 #1 的 `deleteMarkerId`↔`versionId` 子项按计划转入 §四 #10 收尾。

| # | 条目 | 来源 | 状态 | 验收标准 |
|---|---|---|---|---|
| 1 | OpenAPI 契约对齐真实 handler（H1） | ASSESSMENT H1 / todolist #7 | ✅ | ① migrate / migrate/async 请求体字段与 `migrateRequest` 一致 ✅；② presign `method` 枚举为 `get\|put\|post` ✅；③ delete 不再声明 `versionId` ✅；④ multipart/part 补 `expiresIn` ✅；⑤ 契约测试三方一致断言通过 ✅；⑥ 剩余 `deleteMarkerId`↔`versionId` 对齐 → **转入 §四 #10** |
| 2 | 前端 token 移出 `localStorage.servers`（H4） | ASSESSMENT H4 / todolist #15 | ✅ | ① `s3c.servers` 只含 `{id,name,base}`；② token 按 `s3c.token.<serverId>` 单副本存 sessionStorage（「跨会话保留」才写 localStorage）；③ 旧版本内嵌 token 一次性迁移且不重复落盘；④ 测试断言 localStorage 中无 token |
| 3 | `loadAll()` 至少请求一次（M8） | ASSESSMENT M8 / todolist #5 | ✅ | ① `nextToken` 为空时仍加载第一页（reset 语义，避免重复项）；② 任一分页失败停止续页且不提示「已加载全部」；③ 页面守卫与 `loadSeq` 竞态测试覆盖 |
| 4 | SSE 终态检测统一（S7） | ASSESSMENT S7 / todolist #6 | ✅ | ① 流 EOF 未收到终态时轮询 job 状态至终态；② 连续 ≥3 次回读失败快速 `onError`，调用方 `opsBusy` 复位；③ `ctxDeleteFolder` / `DestDialog` / `MigratePanel` 三处行为一致、无 Promise 悬挂 |

**退出条件**：✅ 四项合并入 `develop`（`e03a15e`），`CHANGELOG.md` 的 `[Unreleased]` 记录归档，
`todolist.md` 对应条目关闭并移入 `FEATURES.md`。

---

## 四、v1.0.0：首个稳定版

**目标**：P0 清零后关闭全部 P1，使「契约可信、供应链干净、异步任务不丢」。发布后打 `v1.0.0` tag。

### P1（稳定版门槛）

> **进度（2026-09-16）**：#5 / #6 / #7 / #8 / #10 **全部完成并验证**，见 [`FEATURES.md`](docs/FEATURES.md)「I」段；
> P1 已清零，`v1.0.0` 稳定版门槛达成。

| # | 条目 | 来源 | 状态 | 验收标准 |
|---|---|---|---|---|
| 5 | Go 1.26.5 → 1.26.6（6 个可达 stdlib CVE） | ASSESSMENT H2 / todolist #13 | ✅ | `server/go.mod`、`server/Dockerfile` 升至 1.26.6（CI 经 `go-version-file` 跟随）；CI 增补 `govulncheck@v1.8.0` 门禁；实测 **0 可达漏洞** |
| 6 | 修正 workflow 中的**幽灵 action SHA（2 处）**：`e2e-playwright.yml:35` 的 `pnpm/action-setup`、`e2e-playwright.yml:74` 的 `actions/upload-artifact`（GitHub API 均 404） | ASSESSMENT H3 / todolist #14 + 本轮新发现 | ✅ | 已改为 `b906affcce14559ad1aafd4ab0e942779e9f58b1` / `ea165f8d65b6e75b540449e92b4886f43607fa02`；全仓 10 个 action SHA 经 GitHub API 逐一核验均 200 |
| 7 | 异步任务持久化 + 重启恢复 | ASSESSMENT S1 / todolist #19 | ✅ | `service/job_persist.go` 任务清单落盘（临时文件 + rename + 0600；未复用 `store/atomic.go` 以避免 `service→store` 分层倒置，见架构文档）；启动恢复将非终态任务标记 `interrupted` 并回写；新增 `GET /api/migrate/jobs`；前端 `MigratePanel` 展示未完成任务并提示「移动任务可能已复制未删源」；`interrupted` 保留 7 天（`JobInterruptedTTL`）避免被 30 分钟 TTL 误清 |
| 8 | 补齐 3 个缺失 i18n 键 | ASSESSMENT L4 / todolist #24 | ✅ | `objects.toastCopyFailed` / `batchEdit.tagsNeedKey` / `common.working` 已补 zh/en 定义；新增 `src/i18n/coverage.test.ts` 静态扫描防复发 |

### 契约与文档一致性

| # | 条目 | 来源 | 状态 | 验收标准 |
|---|---|---|---|---|
| 9 | `docs/API.md` 自动化校验 | ASSESSMENT I2 / todolist #8 | ⬜ | CI 增加文档-路由 diff 检查或由 OpenAPI 生成；文档与 `routes.go` 漂移即红灯 |
| 10 | 契约测试覆盖全部 `requestBody` 端点 | ASSESSMENT H1 收尾 / ROADMAP #1 ⑥ | ✅ | `TestOpenAPI_ContractRequestBodyMatchesHandlers` 已扩展至 `delete-marker/restore` / `version`(DELETE) / `version/restore`；`deleteMarkerId`↔`versionId` 已对齐并断言（回退即失败） |

---

## 五、v1.0.x：可靠性加固

**目标**：解决评估中「服务降级 74 分」暴露的存储与可观测性短板，不改变对外契约。

### 存储与安全（P2）

| # | 条目 | 来源 | 状态 | 验收标准 |
|---|---|---|---|---|
| 11 | SQLite 驱动密钥加密 + Argon2 加强 | ASSESSMENT M1/M2 / todolist #16 | ⬜ | compose 默认 `encrypted` 驱动或提供磁盘加密文档；Argon2 t≥2；`S3C_STORE_KEY` 最短长度校验；升级路径兼容旧库 |
| 12 | 安全审计日志 + JobRegistry 上限 + XFF 可信代理 | ASSESSMENT M3/M4/M5 / todolist #17 | ⬜ | 401 / 账号 CRUD / 策略与删除变更留事件日志；JobRegistry 总 job 上限并可观测；`clientIP` 仅信任已知代理 |
| 13 | TLS 前置补安全响应头 | ASSESSMENT M6 / todolist #18 | ⬜ | nginx TLS 模板补 `Strict-Transport-Security` / `Permissions-Policy`；`/api/health` 评估是否隐藏 version |

### 可观测性（P2）

| # | 条目 | 来源 | 状态 | 验收标准 |
|---|---|---|---|---|
| 14 | 流式传输错误可见 | ASSESSMENT S2 / todolist #20 | ⬜ | `copyStream` 返回并记录 error；新增 `s3c_stream_interrupted` 计数；ZIP 部分失败有服务端日志（S6） |
| 15 | S3 上游指标 | ASSESSMENT S3 / todolist #21 | ⬜ | `s3wrap` 层记录调用延迟直方图 / 错误按码分类 / 流字节数；存储状态与 `S3C_LOG_JSON=1` 默认开启 |
| 16 | 预签名错误与错误文案 | ASSESSMENT L1/L2 / todolist #23 | ⬜ | 签名失败返回 5xx 而非空 url 的 200；错误消息不回显用户输入 |

---

## 六、v1.1.0：体验与性能

**目标**：前端长列表与恢复能力，以及让「100% 覆盖率」门禁回归行为价值。

| # | 条目 | 来源 | 状态 | 验收标准 |
|---|---|---|---|---|
| 17 | grid 视图窗口化 + `ObjectList` 清理 | ASSESSMENT D8 / todolist #11 | ⬜ | grid `v-for` 加渲染上限或窗口化；移除未使用的 `visibleCount` prop |
| 18 | 前端健康轮询与自动恢复 | ASSESSMENT S5 / todolist #22 | ⬜ | 后端不可用后自动重试 / 恢复；`useBucketSetting.reload()` 加 seq 或 AbortController 竞态守卫（S8） |
| 19 | 死代码清理 | ASSESSMENT D1-D6 / todolist #9-10 | ⬜ | `ctxReader` / `batchItemError` / `Client.S3()` / `isNoSuchBucketSetting` 删除或内联；endpoint 归一化合并为单一 helper；`timeOrZero` / `deref*` 去重 |
| 20 | i18n 死键清理与重复排序 | ASSESSMENT D7/D9 / todolist #11 | ⬜ | 清理约 40 个未引用键（保留 `storage.class.*` / `provider.*` 等动态模板键）；`entries` / `visibleEntries` 单次排序 |

---

## 七、v1.2+ / 长期

| # | 条目 | 来源 | 状态 | 说明 |
|---|---|---|---|---|
| 21 | 覆盖率门禁去「注水」 | ASSESSMENT I1 / todolist #12 | ⬜ | 前端纳入 `i18n/**` 统计或调整为 95% + 行为测试；后端减少为打满分支而写的 gap 测试 |
| 22 | 桌面端分发与签名 | 长期 | ⬜ | Windows / Linux / macOS 产物签名与自动更新策略（现仅打包挂 Release） |
| 23 | 增量同步与批量能力的体验增强 | FEATURES | ➖ | 现有 `etag` / `size_mtime` / `always` 三模式满足需求，按用户反馈再评估 |
| 24 | 死代码纪律 | ASSESSMENT §三 | ➖ | 维持现状：由覆盖率门禁调整（#21）一并治理，不单独立项 |

---

## 八、质量门禁基线

任一版本发布前必须全绿（当前实测状态）：

| 门禁 | 命令 | 当前状态 |
|---|---|---|
| Go 格式 | `gofmt -l .`（`server/`） | ✅ 干净 |
| Go 静态检查 | `go vet ./...` | ✅ 0 告警 |
| Go 测试 | `go test -race -count=1 ./...` | ✅ 8/8 包通过 |
| Go 覆盖率 | 各包 statements | ✅ 100%（去注水见 #21） |
| Go 漏洞 | `govulncheck ./...` | ✅ 0 可达漏洞（go1.26.6；已入 CI 门禁） |
| 前端 lint | `pnpm lint` | ✅ 0 error / 0 warning |
| 前端类型 | `vue-tsc --noEmit` | ✅ exit 0 |
| 前端测试 | `pnpm test` | ✅ 966 例全绿（62 文件） |
| 前端覆盖率 | statements / branches / functions / lines | ✅ 100% |
| 依赖审计 | `pnpm audit` / Trivy | ✅ npm 0 漏洞；镜像 CRITICAL/HIGH 硬失败 |
| E2E | Playwright（`e2e.yml` + `e2e-playwright.yml`） | ✅ 幽灵 SHA 已修（#6，2 处） |

---

## 九、风险与依赖

| 风险 | 影响 | 缓解 |
|---|---|---|
| OpenAPI 契约作为 SSOT 仍可能漂移 | 客户端按文档调用 400 | 契约测试升级为「注册表 ↔ handler 解析字段」三方断言（#10），并入 CI 必过项 |
| 异步任务纯内存 | 重启丢任务、产生中间态 | 已解决（#7）：任务清单落盘 + 启动标记 `interrupted` + 前端对账视图 |
| `localStorage` 中残留历史 token | 升级用户仍明文落盘 | ✅ H4 迁移逻辑一次性清除内嵌 token（已随 rc1 提交 `e03a15e`） |
| Go stdlib CVE 修复线滞后 | 运行时 DoS / 复杂度攻击面 | ✅ 已升级 1.26.6 + `govulncheck` 门禁（#5），dependabot 已覆盖 |
| 幽灵 action SHA（2 处） | 工作流失败，或 fork 伪造 tag 时执行恶意 action | ✅ 已按 GitHub API 核验结果替换（#6），全仓 SHA 复核通过 |
| 覆盖率 100% 掩盖死代码 | 维护成本高、真实覆盖失真 | 门禁调整与死代码清理同批做（#21 / #19） |
| SQLite 默认驱动密钥明文 | 落盘密钥泄露 | v1.0.x 默认加密或文档化磁盘加密（#11） |

---

## 十、维护约定

1. **单一来源**：本文件只维护**版本级规划与优先级**；新增 / 关闭具体条目时，同步更新
   [`docs/todolist.md`](docs/todolist.md)（唯一待办来源），完成后归档至 [`docs/FEATURES.md`](docs/FEATURES.md)。
2. **评估驱动**：每次五维度评估（见 [`docs/ASSESSMENT.md`](docs/ASSESSMENT.md)）产出后，按 P0/P1/P2 回写
   本路线图与 todolist，形成「评估 → 修复 → 再评估」闭环。
3. **状态真实性**：标 ✅ 必须附门禁实跑或测试证据；标 ⏳ 必须是工作区/分支已有代码变更，并在合并后改为 ✅。
4. **发版触发**：里程碑验收标准全部满足后，由 `scripts/release-version.sh` 同步版本号并更新
   [`CHANGELOG.md`](CHANGELOG.md)，再打 tag。
5. **文档同步**：修复 bug 或新增功能完成后必须更新相关文档（对照表见
   [`docs/development.md`](docs/development.md) §4 与 [`agents.md`](agents.md)）；本路线图的版本级条目随改动同步。
