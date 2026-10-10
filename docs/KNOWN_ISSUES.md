# s3client 已知问题（Known Issues）

> 本文件是**缺陷 / 外部阻塞 / 技术债**的**唯一来源**，只收录**尚未闭环**的项。
> **功能候选与版本规划不在本文件**——见 [`ROADMAP.md`](ROADMAP.md) §三（战略层唯一来源）；
> 已修复 / 已完成台账见 [`FEATURES.md`](FEATURES.md)；发版历史见 [`CHANGELOG.md`](../CHANGELOG.md)。
> 最近一轮综合评估（2026-09-16，五维度：代码质量 / 漏洞 / 死代码 / 降级 / 自我迭代）见
> [`archive/assessment.md`](archive/assessment.md)。
>
> 状态图例：⬜ 待处理 · ⏳ 处理中（仅列剩余工作） · ➖ 已决策（不做 / 维持现状） · ⛔ 外部阻塞（外部凭证未获取等非代码工作）
>
> **两源分工（2026-09-24 起）**：本文件收**问题**（缺陷 / 阻塞 / 技术债）；
> [`ROADMAP.md`](ROADMAP.md) 收**方向**（版本规划 / 功能候选）。同一编号只在一个文件里是「当前条目」，
> 跨文件引用必须带前缀：`KNOWN_ISSUES #N` / `ROADMAP §三 #N`。
>
> **编号约定**：编号保持稳定、不因条目移除而重排——`apps/server/` 代码注释与历史提交仍以
> `KNOWN_ISSUES #N` 引用本清单（2026-09-24 迁移前写作 `todolist #N`，两者同指），重排会使这些引用失真。
> 已闭环移除的编号：#1–#24 / #26–#39 / #41 / #45 / #46 / #60–#64 / #69 / #70 / #71 / #72 / #73 / #74 / #75 / #76 / #77 / #78 / #79 / #80 / #81 / #82 / #83；
> **从未启用（保留空号）**：#40 / #42 / #43 / #44——补登记时跳号，为保持既有编号稳定而**不回收**
> （回收会让历史提交里的 `#N` 指向不同条目）。
> **2026-09-24 迁出**：#47–#59 为 ROADMAP 派生的**功能候选（非问题）**，唯一来源改为
> [`ROADMAP.md`](ROADMAP.md) §三 3.2，本文件不再收录。故本清单编号不连续属预期，不是漏登记。
>
> 最后更新：2026-10-09（第三批次：**#76 / #79 已闭环并移除**，证据见 [`FEATURES.md`](FEATURES.md) §CD）：
> **#76** 前端结构性重复收敛——`newRowKey` 的 7 份逐字复制提取为 `src/rowKey.ts` 的
> `createRowKey` 工厂；虚拟滚动管线（ObjectList / MigratePanel / VersionsDialog /
> RecycleBinPanel 的 4 份）提取为 `src/composables/useVirtualRows.ts`，各组件只传响应式数组；
> **#79** OpenAPI 契约漂移收口——新增「handler 直接写出的状态码必须声明」门禁
> （`openapi_status_declared_test.go`）并补齐 migrate jobs 404 / trash 409 / copy 409 /
> 异步端点 503 / proxy 400+416 等缺失声明；`putObjectLock` 用新增的 `anyOf` 表达
> days/years 二选一；`bucket` 从全部请求体 required 移除（handler 经 `bucketOr` 回退账号
> 默认桶，与共享 Bucket query 参数「可省略」一致）并新增门禁，重生成 `docs/api/openapi.json`
> 与前端 `schema.d.ts`。
> 故 §二 当前存量 = **#63（➖ 已决策）**——**2026-10-09 评审的 O1–O12（#72–#83）已全部闭环移除**。
>
> 上一轮更新：2026-10-09（第二批次：**#72 / #74 / #75 / #77 / #78 已闭环并移除**，证据见 [`FEATURES.md`](FEATURES.md) §CC）：
> **#72** 数据面 `UNSIGNED-PAYLOAD` 收窄为只在**带 stream 的请求**注入（无 body 的
> GET/HEAD/DELETE/List 恢复 SigV4 空体哈希签名；此前无条件注入），`http://` endpoint 的
> 残留载荷完整性风险记入 [`threat-model.md`](threat-model.md) §6.2；
> **#74** 前端传输层新增 `ApiError{status, body}`（错误响应归一携带 HTTP 状态与已解析体），
> 成功响应的非 JSON body 由 `request` 边界校验上抛；`jobs.ts` SSE 失败改抛 `ApiError`；
> **#75** `generated.gate.test.ts` 补双向穷尽性断言：生产源码 `opPath('<id>')` 引用的 opId
> 必须存在，且未被引用的 opId 必须正好等于 8 个「前端不消费」白名单；
> **#77** 右键菜单关闭还原来源焦点、Escape 改走 `useKeydownStack`（LIFO，删掉
> `useObjectBrowser` 的独立 window Escape 监听）、「⋯」触发器补 `aria-haspopup`/`aria-expanded`；
> **#78** i18n 覆盖测试补「模板 / 数据驱动键逐个有定义」断言（堵 `storageReport.kind.${kind}`
> 等盲区），并清理 3 处硬编码可见文案（`AccessKey ID/Secret`、批量标签「替换」）。
> 故 §二 当前存量 = **#63（➖ 已决策）+ #76 / #79（⬜ 开放）**。
>
> 上一轮更新：2026-10-09（**#73 / #80 / #81 / #82 / #83 已闭环并移除**；#73/#80/#81/#83 证据见 [`FEATURES.md`](FEATURES.md) §CA，#82 见 §CB）：
> **#73** `s3wrap` 4 个签名带 AWS SDK 类型、仅供包内调用的转换函数降为小写（`fromS3Object` /
> `formatBuckets` / `describeACL` / `granteeLabel`），收窄 SDK 类型外泄入口；
> **#80** 计划校验错误不再透传 `err.Error()`——`service` 新增类型化 `ScheduleValidationError`
> （`Msg` 固定、`Cause` 仅落日志），handler 改走 `ScheduleValidationMessage`，`error_echo_gate`
> 增加对 `writeErr(..., 400, x.Error())` 形态的捕获与正则自检；
> **#81** `POST /api/accounts`（创建账号，无路径 `{id}` 可判）对 `accounts` 作用域 token
> 一律 403（`reason=accounts_create`）——堵住「`accounts:[A]` 的 token 铸造任意新账号」的
> 凭证面；`GET /api/accounts`（列表，返回不含 `SecretKey` 的 `AccountView`）语义在
> [`threat-model.md`](threat-model.md) 最小权限节明确；
> **#83** 散点缺陷群 12 处全部修复（2 处漏包 `wrapObjectTooLarge`、`PurgeObject` 遮蔽 `out`、
> `dialContextSSRF` 可返回 `(nil,nil)`、`store` / `sqlite` 吞 `encryptAESGCM` 错误、
> `atomicfile` 固定 `.tmp` 并发互删、storage_report 金额未取整、`Scheduler.List` O(n²) 选择排序、
> 损坏 `schedules.json` 静默丢弃后被空列表覆盖、非法 cron 停摆不写 `LastError`、`Job.Total`
> 无锁读、计划/任务落盘 `Save` 失败被静默——**新增 `s3c_persist_failures_total` 指标**）；
> **#82** R10 评审残留闭环——`make check` 补 `govulncheck` / 前端 `pnpm lint` / `go build ./...`，
> 真 RustFS Go E2E 的 PR 触发面从 `internal/s3wrap/**` 放宽到 `apps/server/**`，DEVELOPMENT
> 的 `perf-budget`（真实 job id 为 `bench`）/ 覆盖率排除项 / 触发事件 job 计数三处漂移修正，
> 并加机械门禁防回退（`doc_ci_drift_gate_test.go` + `TestMakefileCheckMirrorsCIStaticGates` +
> `TestRustFSE2ETriggersCoverWholeBackend`），证据见 [`FEATURES.md`](FEATURES.md) §CB。
> 故 §二 当前存量 = **#63（➖ 已决策）+ #72 / #74–#79（⬜ 开放）**。
>
> 上一轮更新：2026-10-08（**#71 已闭环并移除**，同日分两批完成，**推翻原 ➖「维持现状」决策**）：
> **① 仓库 slug 统一**——`github.com/weilai1949/s3clinet` → `…/s3client`（`apps/server/go.mod` 模块路径 +
> 全仓 Go import + 全部仓库 URL：`.github/SECURITY.md` / `ISSUE_TEMPLATE/config.yml` / `SUPPORT.md` /
> `CONTRIBUTING.md` / `AI_POLICY.md` / `docs/en/index.md` / `accounts.schema.json` `$id` /
> Grafana 面板链接 / 根 `README.md` Release 链接），对齐 `git remote` / `.well-known/security.txt` /
> `CITATION.cff` 既有取值，证据见 [`FEATURES.md`](FEATURES.md) **§BO**；
> **② 产品名统一（同日第二批）**——品牌 / 运行时 / 监控命名空间**一并**改为 `s3client`：文档与英文快照标题、
> OpenAPI `title: "s3client API"`（已重生成 `openapi.json` 与前端 `schema.d.ts`）、启动日志 `msg="s3client server"`、
> 单写者锁文件 `.s3client.lock`（原 `.s3clinet.lock`，**行为变更**）、二进制 `s3client-server`、镜像与
> `container_name` `s3client/server` 系、Cargo 包与 Tauri `productName` / `identifier`、npm `s3client-web` /
> `s3client-desktop`、E2E 环境变量 `S3CLIENT_E2E` / `S3CLIENT_{ENDPOINT,ACCESS_KEY,SECRET_KEY}`、
> 记录规则 `s3client:*` 与告警 `S3Client*`、`deploy/{prometheus,grafana}/s3client.*` 与
> `deploy/nginx/conf.d/s3client-*.conf` 文件名，证据见 [`FEATURES.md`](FEATURES.md) **§BP**。
> **仍保留旧写法的只有两处（历史不回写）**：`CHANGELOG.md` 历史条目（**只修正**指向改名文件的路径链接，
> 叙述里的旧名照旧）与 `docs/archive/` 冻结件。
>
> 上一轮更新：2026-09-30（**#70 已闭环并移除**：`NormalizeEndpoint` 幂等修复——推翻原 ➖ 决策、
> 按登记内写死的修法（先切分 host/path，再对 host `TrimSpace`）落地，TDD 先红后绿 + 变异验证 +
> 两目标各 10s 有界 fuzz PASS，证据见 [`FEATURES.md`](FEATURES.md) §BK。同日 **#69 已闭环并移除**：
> CHANGELOG 顶部新增「tag ↔ 版本段对应关系（唯一台账）」，
> 3 个时间戳 tag 定性为**同日内部快照**（打在 feat/fix 提交上、非 release 提交、当日被 rc0/rc1 取代，
> 无独立版本段）；反向补齐 5 个「有段无 tag」的历史段登记；`[1.0.0]` 段日期按 tag 事实修正为
> 2026-09-22（内容未改写）并恢复倒序；新门禁 `apps/server/changelog_tag_gate_test.go` +
> `scripts/release-version.sh` 硬检查，证据见 [`FEATURES.md`](FEATURES.md) §BE。产品功能缺陷仍为零。
> 前一日记录保持原文——2026-09-29 文档覆盖矩阵收口时新开 #65–#68 四条，**均为写文档时实测发现**：
> 门禁口径 / CI 覆盖 / 可访问性 / 日志关联。同日晚些时候 **#65–#68 已全部闭环并移除**：
> #65 / #67 / #68 各补了机械门禁（证据见 [`FEATURES.md`](FEATURES.md) §AO / §AP），
> #66 的 GitLab SAST 以**实跑通过**（非 `--list`）闭环并推翻了它自己登记的「无法本地验证」
> 这一理由（证据见 §AQ；发现项 triage 见 [`threat-model.md`](threat-model.md) §7）。
> 仓内技术债不再为零，产品功能缺陷仍为零。

## 目录

- [一、外部阻塞](#一外部阻塞)
- [二、技术债 / 缺陷](#二技术债--缺陷)
- [三、分类归零凭证](#三分类归零凭证)
- [四、编号台账](#四编号台账)

---

## 一、外部阻塞

> 非代码工作，依赖外部凭证 / 第三方通道；代码侧已无剩余工作，故**不占仓内排期**。

| # | 项 | 来源 | 状态 | 说明 |
|---|----|------|------|------|
| 25 | 桌面端分发与签名 | ROADMAP §三 #1 / 长期 | ⛔ | **外部阻塞**：Windows 代码签名证书 / Apple Developer ID + 公证是**外部凭证**（roadmap E6，⬜ 未获取），未获取前无法完成签名 / 公证，产物被 SmartScreen / Gatekeeper 拦截且无自动更新通道（roadmap R5）。代码侧无剩余工作（发布链已收口）；未签名产物以 `SHA256SUMS` + 手动放行说明过渡（[DEPLOYMENT.md](DEPLOYMENT.md) §5） |

## 二、技术债 / 缺陷

> 与代码 / 仓库硬约束直接相关、可在仓内闭环的项。

| # | 项 | 来源 | 状态 | 说明 |
|---|----|------|------|------|
| 63 | 流式复制单对象 640GB 上限（64MB × 10000 段） | code-review-2026-09-24 Nit（刻意取舍） | ➖ | **已决策维持现状**（2026-09-28 复核并补齐证据）：10000 段是 S3 协议上限，按比例放大分段缓冲会突破容器 512MB 内存预算（`docker-compose.yml` / `docker-compose.prod.yml` 的 server 服务 `deploy.resources.limits.memory: 512M`，一块分段缓冲即 64MB）。超出上限的对象在段号耗尽前被**明确拒绝并 abort**，绝不静默截断。口径与内存账写在 `service/stream_copy.go` 注释（段号在**上传前**判定，不误杀第 10000 段的合法对象）；两个默认值分别由 `TestMultipartStreamCopyPartSizeIs64MB`（分段 64MB）与 `TestMaxMultipartPartsIsProtocolLimit`（段数 10000）钉住，边界行为由 `TestMultipartStreamCopyAcceptsExactlyMaxParts` / `TestMultipartStreamCopyRejectsPartOverLimit` / `TestMultipartStreamCopyByteCeiling` 覆盖。如将来要放宽，先评估内存预算再动。**2026-10-08 补登**：该限制此前只写在 `stream_copy.go` 注释 / 本文件 / [`OPERATIONS.md`](OPERATIONS.md) §性能表，**根 `README.md` 新增「已知限制」小节**把它推到用户可见面 |

> 2026-09-28：#60（前端测试拆分）/ #61（`SameEndpoint` 纳入 `useSSL`）/ #62（批量删除编排下沉 `service`）
> 已闭环移除，证据见 [`FEATURES.md`](FEATURES.md) §AB；同日新开 **#64**（三路复审 19 条的处置清单），
> 其余 18 条当天闭环（§AE–§AI）。**#64 于 2026-09-28 晚些时候闭环**：json 分支抽
> `openJSON(path, storeKey)` 改为「入参非空即用入参、为空才回退 `New`」，
> 不动 `New` 签名（避开了当时预估的 24 文件重构），证据见 [`FEATURES.md`](FEATURES.md) §AJ。
> **2026-09-29**：文档覆盖矩阵收口时新开 **#65**（前端死代码门禁的注释 / 字符串漏报口径，已实测确认）、
> **#66**（GitLab 侧缺 SAST）、**#67**（前端可访问性三处：`<html lang>` 不随语言更新 /
> 未处理 `prefers-reduced-motion` / `textarea` 缺统一焦点轮廓）、**#68**（nginx 访问日志缺
> `$http_x_request_id`，前后端日志无法跨层关联）。四条**均为写文档时实测发现**。
> **#65 已于同日闭环并移除**：`usageBody()` 由「正则裸词计数」改为 **TS AST 收集引用标识符**
> （`.vue` 的 `<script>` 走 AST、`<template>` 原样算引用），注释与字符串字面量里的同名整词
> 不再算引用；配合成源码口径用例 + 真实变异验证（注入「只被注释提到」的导出 → 红灯点名）。
> 证据见 [`FEATURES.md`](FEATURES.md) §AO。途中一次**失败的尝试**也记入该节：先用
> `ts.createScanner` 做词法剥离，实测它会把含 `${…}` 插值的模板字面量**吞掉其后整段文件**，
> 一次误报 18 个真实导出为死代码——故退回 AST 路线。
> **#67 / #68 亦已于同日闭环并移除**（证据见 [`FEATURES.md`](FEATURES.md) §AP）：
> **#67①** `i18n/index.ts` 新增 `applyDocumentLang()`（初始化 + `setLocale` 两处同步 `<html lang>`）；
> **#67②** `styles.css` 末尾加 `@media (prefers-reduced-motion: reduce)` 关闭 animation / transition；
> **#67③** `textarea` 纳入统一 `:focus-visible` 列表；三者由新增的
> `apps/web/src/a11y_gate.test.ts`（源码形态）与 `i18n/index.test.ts`（行为）两道门禁守住。
> **#68** `deploy/nginx/{nginx,nginx.docker}.conf` 的 `log_format main` 补
> `rid=$http_x_request_id req=$upstream_http_x_request_id`（后端回显值才是与后端 `req` 对齐的那个），
> 由 `TestNginxAccessLogCarriesRequestID` 守住。
> **#66 亦已于同日闭环并移除**：GitLab 侧由 `include: template: Jobs/SAST.gitlab-ci.yml` 引入官方
> SAST（`semgrep-sast`，**Free 档可用**）并把 `test` 补进自定义 `stages`。
> **原来「不镜像」的理由（另一套工具链、无法本地验证）经实测作废**——但闭环依据是**真跑通**，
> 不是「能列出来」：`make gcl GCL_JOBS=semgrep-sast` → `finished in 23 s` → `exported artifacts`
> → **`PASS` / `EXIT=0`**，产出 `gl-sast-report.json`。（`make gcl-list` 只证明模板能解析、job
> 排得进流水线；**实跑才暴露出**默认排除路径漏掉 Go/Vitest 测试约定与生成的 `coverage/` 资产，
> 首跑 66 项 → 补齐后 10 项，逐条 triage 见 [`threat-model.md`](threat-model.md) §7。）
> 证据见 [`FEATURES.md`](FEATURES.md) §AQ 与 [`DEVELOPMENT.md`](DEVELOPMENT.md) §3。
> 故本表当前为：**#63 已决策维持现状（➖）**；**产品功能缺陷仍为零**，外部阻塞见 §一 #25。
> **#70 / #71 为 2026-09-30「AI 时代文档补强」批次新增**（见 [`FEATURES.md`](FEATURES.md) §BG）：#70 由本批新增的 fuzz 目标实测发现（不可利用，已留回归种子），**已于同日闭环并移除**（推翻 ➖ 决策、修复证据见 [`FEATURES.md`](FEATURES.md) §BK）；#71 为登记既有命名分裂，不是本批引入，**已于 2026-10-08 闭环并移除**（同样推翻 ➖ 决策、证据见 [`FEATURES.md`](FEATURES.md) §BO）。
> **2026-10-09**：全仓代码质量评审（[`code-review-2026-10-09.md`](archive/code-review-2026-10-09.md)）的 **O1–O12 十二条 Optional 项登记为 #72–#83**（本节表，全部属技术债 / 缺陷，按「同一事项只登记一处」不进 [`ROADMAP.md`](ROADMAP.md) §三）；同批评审的 C1–C2 与 R1–R9 已于**当日修复闭环、不占编号**（证据见 [`FEATURES.md`](FEATURES.md) §BZ 与 [`../CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]`）；R10 的工具链残留同日解除（本机 `/usr/local/go1.26.9` 与 CI 同版本）。其中 **#73 / #80 / #81 / #83 已于同日闭环并移除**（证据见 [`FEATURES.md`](FEATURES.md) §CA；#81 = `POST /api/accounts` 拒绝 `accounts` 作用域 token，见 §二 台账与 [`threat-model.md`](threat-model.md)），**#82**（R10 评审残留：`make check` 缺项 / E2E 路径过滤 / DEVELOPMENT 漂移）亦于同日闭环（证据见 [`FEATURES.md`](FEATURES.md) §CB），**#72 / #74 / #75 / #77 / #78** 同日第二批次闭环（证据见 §CC），**#76 / #79** 同日第三批次闭环（证据见 §CD）。故 §二 当前存量 = **#63（➖ 已决策）**——**2026-10-09 评审的 O1–O12（#72–#83）已全部闭环移除**，产品功能缺陷仍为零，外部阻塞见 §一 #25。

## 三、分类归零凭证

> 迁移前本文件按六类收录待办。以下**原分类当前均无开放项**——保留本表是为了让旧编号
> （`todolist #N` / `KNOWN_ISSUES #N`）仍可回溯到闭环证据，**不是待办**；
> 各分类里属于「功能候选」的条目已按两源分工迁往 [ROADMAP.md](ROADMAP.md) §三 3.2。

| 原分类 | 现状 | 闭环证据 / 迁出去向 |
|--------|------|---------------------|
| 功能 / 架构 | 无开放项 | 迁出：#47–#52 / #58 / #59 → roadmap §三 3.2 |
| API / 契约 | 无开放项 | 迁出：#53 → roadmap §三 3.2。#26 / #27 / #28 已于 2026-09-22 闭环（[FEATURES.md](FEATURES.md) §T / §W）；请求体字段、query 参数、类型 / required / 枚举语义、共享 schema 与端点级响应均有机械门禁，断言范围写在各自文件头注释 |
| 代码质量 / 死代码 | 无开放项 | 迁出：#57 → roadmap §三 3.2。#11 / #12 已闭环；#28 的残留范围已随自由体全量收敛（[FEATURES.md](FEATURES.md) §W） |
| 安全 / 供应链 | 无开放项 | 迁出：#55 / #56 → roadmap §三 3.2。#16 / #17 已闭环；#18 于 2026-09-23 复审维持现状并移出（决策见 [threat-model.md](threat-model.md) §6.2）；#29–#34 于 2026-09-20 闭环（[FEATURES.md](FEATURES.md) §T）；S1 / S2 于 2026-09-19 闭环 |
| 可靠性 / 可观测性 | 无开放项 | 迁出：#54 → roadmap §三 3.2。#21 / #22 / #23 已闭环；#35 / #36 / #38 / #39 于 2026-09-22 闭环（[FEATURES.md](FEATURES.md) §U）；#37 于 2026-09-22 闭环（[FEATURES.md](FEATURES.md) §V） |
| 文档失真 | 无开放项 | D2 / D3 / D6 / D8 / D9 随 2026-09-19 文档同步修正（[FEATURES.md](FEATURES.md) §S）；D1 / D4 / D5 / D10（对应 #35 / #45 / #41 / #46）于 2026-09-22 闭环（[FEATURES.md](FEATURES.md) §U） |

## 四、编号台账

> 历史提交与代码注释里的 `todolist #N` / `KNOWN_ISSUES #N` 均可按下表回溯；**编号不重排、不回收**。

| 编号 | 状态 | 说明 |
|------|------|------|
| #1–#24 | 已闭环移除 | 证据见 [FEATURES.md](FEATURES.md) 与 [threat-model.md](threat-model.md) §6.2 |
| #25 | **开放** ⛔ | 外部阻塞，见 §一 |
| #26–#39 | 已闭环移除 | 2026-09-19 补登记，2026-09-20 / 2026-09-22 分批闭环（[FEATURES.md](FEATURES.md) §S–§W） |
| #40 / #42 / #43 / #44 | 从未启用（保留空号） | 补登记时跳号；回收会让历史提交里的 `#N` 指向不同条目 |
| #41 / #45 / #46 | 已闭环移除 | 对应文档失真 D1 / D4 / D5 / D10，2026-09-22 闭环（[FEATURES.md](FEATURES.md) §U） |
| #47–#59 | **已迁出** | 功能候选（非问题），唯一来源改为 [ROADMAP.md](ROADMAP.md) §三 3.2 |
| #60 | **已闭环移除** | 4 个超 1000 行的前端测试文件拆分，2026-09-28 闭环（[FEATURES.md](FEATURES.md) §AB） |
| #61 | **已闭环移除** | `SameEndpoint` 精确判定纳入 `useSSL`（跨包契约变更），2026-09-28 闭环（[FEATURES.md](FEATURES.md) §AB） |
| #62 | **已闭环移除** | 批量删除编排下沉 `service`（2026-09-24 审查 Nit 本轮未完成），2026-09-28 闭环（[FEATURES.md](FEATURES.md) §AB） |
| #63 | **已决策** ➖ | 已知限制（流式复制单对象 640GB 上限，维持现状），见 §二 |
| #64 | **已闭环移除** | 三路复审 19 条中唯一未闭环项（`store.Open` json 分支丢 `storeKey`）：抽 `openJSON(path, storeKey)` 改为「入参非空即用入参，为空才回退 `New`」，2026-09-28 闭环（[FEATURES.md](FEATURES.md) §AJ）；当天早些时候闭环的 18 条台账见同文件 §AD–§AI |
| #65 | **已闭环移除** | 前端死代码门禁的注释 / 字符串漏报口径：`usageBody()` 改 TS AST 收集引用标识符（`.vue` 的 `<script>` 走 AST、`<template>` 原样算引用），并补合成源码口径用例 + 变异验证，2026-09-29 闭环（[FEATURES.md](FEATURES.md) §AO） |
| #66 | **已闭环移除** | GitLab 侧 SAST：`include: template: Jobs/SAST.gitlab-ci.yml` 引入官方 `semgrep-sast`（Free 档），`stages` 补 `test`；原「无法本地验证」的理由经**实跑**作废（`make gcl GCL_JOBS=semgrep-sast` → `PASS`，非仅 `--list`），2026-09-29 闭环（[FEATURES.md](FEATURES.md) §AQ；发现项 triage 见 [threat-model.md](threat-model.md) §7） |
| #67 | **已闭环移除** | 前端可访问性三处：`applyDocumentLang()` 同步 `<html lang>` / `@media (prefers-reduced-motion: reduce)` 关闭动效 / `textarea` 纳入统一 `:focus-visible` 列表，2026-09-29 闭环（[FEATURES.md](FEATURES.md) §AP） |
| #68 | **已闭环移除** | nginx `log_format main` 补 `rid=$http_x_request_id req=$upstream_http_x_request_id`，跨层日志可关联，2026-09-29 闭环（[FEATURES.md](FEATURES.md) §AP） |
| #69 | **已闭环移除** | CHANGELOG 与 git tag 断裂：顶部新增「tag ↔ 版本段对应关系（唯一台账）」（3 个时间戳 tag 定性为同日内部快照、5 个「有段无 tag」历史段登记）；`[1.0.0]` 段日期按 tag 事实修正为 2026-09-22（内容未改写）并恢复倒序；新门禁 `apps/server/changelog_tag_gate_test.go`（tag↔段双向 + Unreleased 居首 + 映射表解析口径 + 扫描阈值，TDD 先红后绿 + 变异验证）+ `scripts/release-version.sh` 硬检查（缺 `## [<version>]` 段即 exit 1），2026-09-30 闭环（[FEATURES.md](FEATURES.md) §BE） |
| #70 | **已闭环移除** | `NormalizeEndpoint` 对含尾随空白输入不幂等：原 ➖「fail-closed 维持现状」决策于 2026-09-30 推翻，按登记内写死的修法修复（先切分 host/path，再对 host `TrimSpace`、对 path 去尾部斜杠与空白，保输出不以空白结尾），TDD 先红后绿 + 变异验证（删 host `TrimSpace` → 红灯点名 `"00  /"` / `"http://host  /"` → 还原绿）+ 两目标各 10s 有界 fuzz PASS，2026-09-30 闭环（[FEATURES.md](FEATURES.md) §BK） |
| #71 | **已闭环移除** | 仓库 slug `s3clinet` / `s3client` 并存：2026-10-08 **推翻原 ➖「维持现状」决策**，同日分两批统一——**① slug**（`go.mod` 模块路径 + 全仓 Go import + 全部仓库 URL → `github.com/weilai1949/s3client`，与 `git remote` / `.well-known/security.txt` / `CITATION.cff` 对齐）见 [FEATURES.md](FEATURES.md) §BO；**② 产品名**（品牌 / 运行时 / 监控命名空间的全部旧写法 → `s3client` 系，含 `.s3client.lock` 锁文件名等**行为变更面**）见同文件 §BP。仅 `CHANGELOG.md` 历史条目（只修路径链接）与 `docs/archive/` 冻结件保留旧写法 |
| #73 / #80 / #81 / #83 | **已闭环移除** | 2026-10-09 评审 O2 / O9 / O10 / O12：s3wrap 4 个带 SDK 签名的包内转换函数降为小写；计划校验错误改类型化定值文案 + `error_echo_gate` 捕获 `err.Error()` 透传；`POST /api/accounts` 拒绝 `accounts` 作用域 token（`reason=accounts_create`）；散点缺陷群 12 处全修（含新增 `s3c_persist_failures_total`），证据见 [`FEATURES.md`](FEATURES.md) §CA |
| #82 | **已闭环移除** | 2026-10-09 评审 O11（R10 残留）：`make check` 补 `govulncheck` / `web-lint` / `build`；`e2e.yml` 与 `.gitlab-ci.yml` 的真 RustFS Go E2E 触发面放宽到 `apps/server/**`；DEVELOPMENT 的 `perf-budget`→`bench` / 覆盖率排除项 / 触发 job 计数三处修正；新增 `doc_ci_drift_gate_test.go` 与两道一致性门禁防回退。**2026-10-10 再处置**：补齐 `web-typecheck`（`pnpm typecheck`）与 `web-build`（`pnpm build`）以真正覆盖 CI `web` job 全部静态步，并**真跑 `make check` exit 0** 复核，证据见 [`FEATURES.md`](FEATURES.md) §CB |
| #76 / #79 | **已闭环移除** | 2026-10-09 评审 O5 / O8：#76 前端 `newRowKey` ×7 → `src/rowKey.ts`、虚拟滚动管线 ×4 → `src/composables/useVirtualRows.ts`；#79 新增 `openapi_status_declared_test.go`（handler 直接写出的状态码必须声明）+ `bucket` required 门禁，补齐缺失状态码、`putObjectLock` 用 `anyOf` 表达二选一、`bucket` 请求体 required 归零并重生成规范。证据见 [`FEATURES.md`](FEATURES.md) §CD |

---

> **迁移说明**：本文件原为 `docs/todolist.md`，2026-09-24 以 `git mv` 更名（保留重命名历史），
> 并同步按「问题 / 方向」两源分工重组：功能候选 #47–#59 迁至 [ROADMAP.md](ROADMAP.md) §三 3.2，
> 本文件只保留缺陷 / 外部阻塞 / 技术债。
