# s3clinet 路线图（Roadmap）

> 本文件只描述 s3clinet 的**版本规划与优先级**（战略层）：每个里程碑的目标、验收标准与**尚未完成**的条目。
> 已实现 / 已修复的内容不在此流水账，见 [`docs/FEATURES.md`](FEATURES.md)；
> 逐条待办自 2026-09-24 起**分两处维护、互不重复**：**问题**（缺陷 / 外部阻塞 / 技术债）的唯一来源为
> [`docs/KNOWN_ISSUES.md`](KNOWN_ISSUES.md)；**功能候选与版本级规划**（本文件 §三）的唯一来源为本文件本身；
> 逐字发布历史见 [`CHANGELOG.md`](../CHANGELOG.md)；
> 评分与问题证据见 [`docs/archive/assessment.md`](archive/assessment.md)（2026-09-16 五维度评估：代码质量 82 /
> 漏洞 72 / 死代码 70 / 服务降级 74 / 自我迭代 90，总分 78）。
>
> 状态图例：⬜ 未开始 · ⏳ 进行中（部分已落地） · ➖ 已决策（不做 / 维持现状） · ⛔ 外部阻塞（外部凭证未获取等非代码工作）
>
> 本文件的 `#N` 编号**独立于** `docs/KNOWN_ISSUES.md` 的 `#N`，两者不可互指。
> 代码注释与历史提交里的 `roadmap #N` 指 **2026-09-17 收口前的旧编号**（条目已归档至
> [`FEATURES.md`](FEATURES.md) §M 与 [`CHANGELOG.md`](../CHANGELOG.md)），**不要按当前编号回读**。
>
> 版本命名：稳定里程碑 **v1.0.0** 后日常发版用时间戳（`v1.0.0-YYYYMMDDHHmmss`），预发布用 `v1.0.0-rcN`。
> 当前版本 **`v1.0.0`**（已打 tag `v1.0.0`）。最后更新：2026-09-29。

## 目录

- [一、当前定位](#一当前定位)
- [二、里程碑总览](#二里程碑总览)
- [三、v1.2+ / 长期](#三v12--长期)
- [四、质量门禁基线](#四质量门禁基线)
- [五、风险与依赖](#五风险与依赖)
- [六、维护约定](#六维护约定)

---

## 一、当前定位

**产品形态**：S3 兼容对象存储客户端，Web 端 + Tauri 2 桌面端（B/S 架构、无 IPC、全 HTTP）；Go 后端
（AWS SDK for Go v2）+ Vue 3 / Vite / TS 前端，70 个 `/api/*` 端点、OpenAPI 3.0.3 自动生成。

**结论**：**P0 与 P1 均已清零**，`v1.0.0` 已打 tag；`v1.0.0-rc1` → `v1.0.0` → `v1.0.x` → `v1.1.0`
四个里程碑均已收口。已立项的未完成项只有长期性质的第 1 条：**桌面端分发与签名**——外部凭证阻塞
（E6 未获取），代码层面已无剩余工作。2026-09-24 另在 §三 3.2 补录 **13 条趋势展望迭代方向**
（#4–#16，⬜ 候选、未排期；其中 #12 供应链证明已于 2026-09-29 落地并按 §六 第 1 条移出），
2026-09-30 再收口迁入 **#17–#18**（可访问性补强 / 观测指标补全，来源 `accessibility.md` §5.3–§5.5
与 `OPERATIONS.md` 观测缺口）——**两条已于同日全部落地并按 §六 第 1 条移出**（#18 见 §BL、#17 见 §BM），
故两源迁入项当前**均已在候选池之外**；按 §六 第 1 条的两源分工，**§三 3.2 表即这批候选的唯一来源**。
排期进入里程碑后才转 ⏳，评估为不做则转 ➖ 并由 ADR / 决策记录兜底。

> 已修复内容不在此流水账：P0 / P1 逐条记录与验证证据见 [`FEATURES.md`](FEATURES.md)「H」「I」段，
> 2026-09-17 一轮的验收证据见同文件 §M。本文件只列**未完成**项。

---

## 二、里程碑总览

| 里程碑 | 主题 | 关键验收 | 依赖 |
|---|---|---|---|
| **v1.2+** | 长期：桌面分发、可选增强、趋势方向（§三 #4–#16 候选池） | 桌面端签名与自动更新；候选方向按需立项评估 | v1.1.0 |
| **v1.3+** | 长期：增量同步与批量能力的体验增强 | 评估用户反馈下的现有模式扩展能力 | v1.2+ |
| **v2.0+** | 长期：死代码纪律治理 | 升级覆盖率门禁，治理残留死代码（已备选） | v1.3+ |

> rc1 → v1.0.0 → v1.0.x → v1.1.0 四个里程碑已全部收口，验收证据见 [`FEATURES.md`](FEATURES.md)
> 「H」「I」/ §M——按 §六 第 3 条约定**不占 ✅ 行**。

> 「依赖」列只表示**里程碑前置关系**（上一个里程碑须先收口），不含外部技术 / 供应链 / 凭证依赖——
> 后者见 [§5.2 依赖清单](#52-依赖清单)。

**发布约束**：任一里程碑发布前，[质量门禁基线](#四质量门禁基线)必须全绿；P0 未清零不得发布。

---

## 三、v1.2+ / 长期

> v1.0.0 / v1.0.x / v1.1.0 三个里程碑的开放条目已于 2026-09-17 全部完成并从本文件移除，
> 逐条验收证据见 [`FEATURES.md`](FEATURES.md) §M。本节只列仍未完成的长期项。
>
> **#4–#16 为 2026-09-24 补录的趋势展望方向**（来源标「趋势展望」：结合现有代码可承接点与
> 2026 技术趋势评估得出）：全部 ⬜ **未排期候选池**，不是发布承诺——立项排期后状态改 ⏳ 并
> 进入 §二 里程碑验收；评估为不做转 ➖。按 §六 第 1 条的两源分工，**本节即这批条目的唯一来源**；
> 2026-09-24 迁入前曾以 `KNOWN_ISSUES #47`–`#59` 登记，该编号**已停用**、仅作历史映射（见表「原编号」列）。
> **#17–#18 为 2026-09-30 收口迁入的文档内改进项**（来源 [`accessibility.md`](accessibility.md)
> §5.3–§5.5 与 [`OPERATIONS.md`](OPERATIONS.md) 观测缺口声明，按「不留文档内口头待办」纪律迁入），
> 「原编号」列记 `—`；**两条已于同日全部落地**并按 §六 第 1 条移出转空号（#18 证据 §BL、
> #17 证据 §BM），故本表当前不含它们。

### 3.1 已立项 / 已决策项

| # | 条目 | 来源 | 状态 | 说明 |
|---|---|---|---|---|
| 1 | 桌面端分发与签名 | 长期 | ⛔ | **已立项**（[`KNOWN_ISSUES.md`](KNOWN_ISSUES.md) #25），**外部阻塞**：Windows 代码签名证书 / Apple Developer ID + 公证均为外部凭证（**E6**，⬜ 未获取），未获取前无法完成签名与公证；自动更新通道依赖签名产物。**代码层面无剩余工作**——打包与发布链（tag↔清单校验、平台内唯一 `SHA256SUMS`、聚合 job）已收口，未签名产物以 `SHA256SUMS` + 手动放行说明过渡（[DEPLOYMENT.md](DEPLOYMENT.md) §5）；风险 **R5** 见 §五 |
| 2 | 增量同步与批量能力的体验增强 | FEATURES | ➖ | 现有 `etag` / `size_mtime` / `always` 三模式满足需求，按用户反馈再评估 |

### 3.2 候选池（2026-09-24 起补录，⬜ 未排期；已落地条目按 §六 第 1 条移出）

> 编号接续 3.1（§三 内全局唯一；3.1 原 #3「死代码纪律」于 2026-09-24 以前后端两道导出门禁收口移出，
> 编号不重排故 #3 空号；#12「供应链证明」于 2026-09-29 全部落地后同样移出，#12 空号，证据见
> [`FEATURES.md`](FEATURES.md) §AQ / §AW；#14「原生 fuzz 与性质测试」于 2026-09-30 落地后按 §六 第 3 条
> 同样移出，#14 空号，证据见 [`FEATURES.md`](FEATURES.md) §BG、移出动作见 §BI；#19「文档可读性与
> 流程机械化」于 2026-09-30 收口后按 §六 第 1 条同样移出，#19 空号，证据见
> [`FEATURES.md`](FEATURES.md) §BJ；#18「可观测性补全」2026-09-30 落地后同样移出，#18 空号，
> 证据见 [`FEATURES.md`](FEATURES.md) §BL；#17「可访问性补强与自动化扫描」2026-09-30 落地后同样移出，
> #17 空号，证据见 [`FEATURES.md`](FEATURES.md) §BM；#10「OpenAPI → 前端类型 / 客户端代码生成」
> 2026-10-01 落地后同样移出，#10 空号，证据见 [`FEATURES.md`](FEATURES.md) §BN）；「原编号」列为 2026-09-24 迁入前在 `KNOWN_ISSUES.md` 的编号，
> **已停用**、仅作历史映射——新建与引用一律用本表 `§三 #N`。
> 立项时在本表把状态改 ⏳；评估为不做改 ➖ 并写决策依据（本表即唯一来源，无第二处需同步）。

| # | 方向 | 原编号 | 状态 | 现有代码可承接点 + 趋势依据 |
|---|---|---|---|---|
| 4 | MCP Server：把对象存储能力开放给 AI 代理 | #47 | ⬜ | 已有 70 端点 OpenAPI 3.0.3 全量契约 + `handler → service → s3wrap` 分层，工具面可由契约派生并复用既有鉴权 / 限速 / SSRF 防护；MCP 已是 AI 客户端接入外部工具的事实标准，只读工具可先行、写工具复用 Bearer 与危险操作二次确认 |
| 5 | S3 新协议特性：条件写 / 端到端校验和 / Object Lock | #48 | ⬜ | 已有 CopyObject 复制链、`etag` 比对与版本控制；条件写（If-Match / If-None-Match）防并发覆盖、CRC64 全对象校验和、Object Lock / 合规保留是近两年 S3 API 演进主线，经 `s3wrap` 唯一边界接入并按厂商支持度降级（扩 E8 兼容矩阵） |
| 6 | 计划任务与持续同步（增量同步 → 定时备份） | #49 | ⬜ | `migrate/sync` 三模式 + `JobRegistry` 落盘 / 重启恢复 / SSE 进度已是任务框架；补 cron 式计划即可把一次性迁移升级为「桶 → 桶定时备份」，落盘策略沿用 `job_persist.go`，符合 2026 数据保护 / 可持续备份趋势 |
| 7 | FinOps：存储分析与成本洞察 | #50 | ⬜ | 已有列表 size / storageClass、批量改存储类、生命周期规则读写；按前缀 / 存储类聚合用量、给出低频 / 归档 / 生命周期建议即成成本看板（有界并发 + 100k 上限沿用 `RunBatch`），对齐 FinOps 成本优化大趋势 |
| 8 | 大文件体验：上传断点续传 + 下载并行分段 | #51 | ⬜ | 上传侧已有 multipart 四端点与 4 路并发，缺「刷新 / 断电后恢复」——可持久化分段清单实现续传；下载侧代理已支持 Range / 416，补多段并行 GET 聚合。大文件可靠性是网盘类客户端的分水岭能力 |
| 9 | 本地文件夹 ↔ 桶 双向同步 + PWA 离线壳 | #52 | ⬜ | `download.ts` 已用 File System Access API 流式落盘，同 API 的目录句柄 + `etag` 比对可复用为本地目录同步；PWA manifest / service worker 让 Web 端可安装离线启动（密钥仍不落地 localStorage，遵守安全基线） |
| 11 | OpenTelemetry：trace 贯穿签名 / 代理 / 迁移 | #54 | ⬜ | 已有 Prometheus 指标 + `X-Request-ID` + 可选 `S3C_LOG_JSON`；接入 OTLP 导出（开关式、默认关）把请求 ID 升级为跨 presign / proxy / migrate 的 trace，配套 SLO 仪表盘。OTel 已是可观测性事实标准，与既有指标互补不替换。**2026-09-30 部分落地**：SLO 仪表盘已随本批交付（`deploy/grafana/s3clinet.dashboard.json` + `grafana_dashboard_gate_test.go`）；**OTel trace 仍未做**，本条保持 ⬜。 |
| 13 | Token 作用域与最小权限（只读 / 前缀限定） | #56 | ⬜ | 现有多 token（`S3C_TOKEN` 逗号分隔）只有全权一种；补作用域声明（只读、限定账号 / 前缀、过期时间），高危端点按 scope 拒绝并进 OpenAPI 契约。least-privilege / 短时凭证是 API 鉴权演进主流，复用常量时间比较与既有中间件链 |
| 15 | 多副本 / HA 能力评估（store 外置） | #58 | ⬜ | 现状为 `flock` 单副本（R4 已决策接受、ADR-002 fail-closed）；评估引入可外置的 store 后端（如 SQLite 共享卷 / Postgres 驱动）以支撑滚动升级与多副本。**属推翻既有决策的评估项**：先出 ADR 再动代码，结论若维持现状则转 ➖ |
| 16 | 多平台差异化用户体验增强 | #59 | ⬜ | 体验评估计划（桌面 / 移动 / Web 协同体验）；评估产出后按结论拆分或转 ➖ |

---

## 四、质量门禁基线

任一版本发布前必须全绿（实测状态；**2026-10-01 复测**（#17 可访问性批次收口、#10 代码生成批次
落地后全量重跑）：
`gofmt -l` 干净 / `go vet` 0 告警 / `golangci-lint` **0 issues** / `go test ./...` **9/9 包** /
`go build` OK / `make test-cover` **9/9 包 100.0%（`count==0` 零块）** / `govulncheck` **0 可达**，
与前端 `pnpm lint` 0 告警 / `pnpm typecheck` + `typecheck:e2e` 均 exit 0 / `pnpm test`
**76 文件 1158 例** / `pnpm test:coverage` 四指标 **100%** / `pnpm build` OK / `pnpm gen:api --check`
**exit 0** / `pnpm e2e` **22 passed** / `cargo audit`（`--no-fetch` 用缓存 advisory DB）**0 漏洞**，
两项真实 E2E——`make e2e-real` **3 passed**、`S3CLINET_E2E=1` **4/4 PASS** 全绿。
**本轮无法实跑**：`pnpm audit`——所用镜像 `registry.npmmirror.com` 不提供
`/-/npm/v1/security/audits` 端点（`ERR_PNPM_AUDIT_ENDPOINT_NOT_EXISTS`），沿用 CI 与
2026-09-22 实测值）：

| 门禁 | 命令 | 当前状态 |
|---|---|---|
| Go 格式 | `gofmt -l .`（`apps/server/`） | ✅ 干净 |
| Go 静态检查 | `go vet ./...` | ✅ 0 告警 |
| Go lint | `golangci-lint run ./...`（v2.13.2，`errcheck` / `staticcheck` / `govet` / `ineffassign` / `unused` / `gosec` / `nolintlint`） | ✅ 0 issues |
| Go 测试 | `go test -race -count=1 ./...` | ✅ 9/9 包通过（2026-09-24 §AA 后由 8 包增至 9 包，R11 新增 `internal/atomicfile`） |
| Go 覆盖率 | `make test-cover`（检查 profile 中 `count==0` 语句块） | ✅ 每包 + 汇总均 100.0% statements；CI 硬门禁 100% |
| Go 漏洞 | `govulncheck ./...` | ✅ 0 可达漏洞（go1.26.6；已入 CI 门禁。**2026-10-01 复测**：0 可达；另扫出 21 个「被 require 但代码未调用」的模块漏洞，不构成可达面） |
| 前端 lint | `pnpm lint`（`eslint src e2e e2e-real`） | ✅ 0 error / 0 warning |
| 前端类型 | `pnpm typecheck` + `pnpm typecheck:e2e` | ✅ 均 exit 0 |
| 前端测试 | `pnpm test` | ✅ **1158** 例全绿（**76** 文件；**2026-10-01 §BN**：#10 代码生成批次 +3 例（新增 `src/api/generated.gate.test.ts`：生成物新鲜度 + 结构自检 + `opPath` 行为），文件数 75 → 76；**2026-10-01 §BM**：可访问性批次 +23 例（焦点陷阱 7 / live region 3 / 表格与标签 3 / 组件级 axe 6 / 选中态 1 等），文件数 74 → 75（新增 `src/a11y_axe.test.ts`）；2026-09-30 前的轨迹：2026-09-28 KNOWN_ISSUES #60 拆 4 文件为 9 文件，测试数与测试名清单不变，此前为 67 文件；同日 #64 修复新增 16 条红灯用例 1110 → 1126；2026-09-29 新增 `src/vite_env_guard.test.ts` 2 例隔离开宿主 `NODE_ENV` → 1128；**同日 §AN 删除死代码 `isTopKeydown` 及其白盒用例、改写为派发真实 keydown 的行为断言 → 1126**；**同日 §AO 补死代码门禁的合成源码口径用例 → 1127**；**同日 §AP 收口 #67 / #68，新增 `a11y_gate.test.ts` 3 例与 `i18n` 2 例 → 1132、文件数 74**） |
| 前端覆盖率 | `pnpm test:coverage`（statements / branches / functions / lines） | ✅ 100%（**2026-10-01 §BN 复测**：4327 / 2932 / 1131 / 3712，四指标均 100%（新增 `src/api/operations.ts` 被 `endpoints.ts` 全量消费，纳入统计）；此前 2026-10-01 §BM 值 4321 / 2932 / 1130 / 3706、2026-09-29 §AP 值 4296 / 2932 / 1130 / 3679（`NODE_ENV` 未设）与 4294 / 2932 / 1130 / 3677（`NODE_ENV=production`，差值来自 Vue dev/prod 构建各自少/多插桩的那一行）均 100%。含 `src/i18n/index.ts`） |
| 前端生成物新鲜度 | `pnpm gen:api --check`（= `node scripts/gen-api.mjs --check`，由 `src/api/generated.gate.test.ts` 在 `pnpm test` 内调用） | ✅ exit 0（**2026-10-01 §BN 首次登记**）：`src/api/schema.d.ts` / `src/api/operations.ts` 与 [`api/openapi.json`](api/openapi.json) 逐字节一致；改 spec 忘了重跑 `pnpm gen:api` 即红灯 |
| 依赖审计 | `pnpm audit` / Trivy | ✅ npm 0 漏洞；镜像 CRITICAL/HIGH 硬失败。**2026-10-01 本地未能实跑**：所用镜像不提供 audit 端点（`ERR_PNPM_AUDIT_ENDPOINT_NOT_EXISTS`），沿用 CI 结果；新增 devDependency `vitest-axe` 由 Dependabot / CI 覆盖 |
| E2E（mock 版） | Playwright（`e2e.yml` + `e2e-playwright.yml`） | **22 passed / 0 skipped**（**2026-10-01 实跑**：`pnpm e2e`，含 5 条 `a11y.spec.ts` axe 扫描与 2 条截图——本批配色 / aria 改动的渲染态验证；此前 2026-09-29 记为 17 passed，含 `screenshots.spec.ts` 的 2 例）。此前本行「当前状态」误填成 action SHA 校验结果——那是 `TestWorkflowActionsAreShaPinned` 的职责，与 E2E 通过数无关，2026-09-29 更正 |
| E2E（真实联调） | `make e2e-real`（`e2e-real.yml` + GitLab `e2e-real` job，共用 `scripts/e2e-real.sh`） | ✅ 3 passed / 0 skipped（**2026-10-01 复测**——本轮改动前端，按 AGENTS 必跑；本机 8080 被 `haproxy` 占用故用 `SERVER_PORT=8081`，脚本编排不变。真实后端 + RustFS + 真实产物、**`S3C_TOKEN` 开启的生产同构形态**；2026-09-24 审查 C1 验收实跑为上一次） |
| E2E（真实 S3 协议） | `S3CLINET_E2E=1 go test ./internal/s3wrap/ -run 'TestE2E'` | ✅ 4/4 PASS（**2026-10-01 复测**：临时起 `rustfs/rustfs:1.0.0-rc.3` 于 `127.0.0.1:9000` 跑完即删；`TestE2ERustFS` / `TestE2EBatch1` / `TestE2EBucketSettings` / `TestE2ETrash` 四条全 PASS。真实 RustFS：分段 / 复制 / 桶属性 / 回收站） |
| Rust 依赖审计 | `cargo audit`（两套 CI 的 desktop job + `make rust-audit`） | ✅ 0 漏洞；7 条 unmaintained/unsound 告警已 triage（**2026-10-01 `cargo audit --no-fetch` 复测**——`make rust-audit` 默认要从 GitHub 拉 advisory DB，本环境网络不通，改用本地缓存 DB，结论不变） |

> 后端覆盖率已补齐至**每包 100%**（2026-09 删除了确实不可达的防御分支，其余缺口改用行为断言，
> 不做 gap 测试），CI 与 `make test-cover` 以 100% 为硬阈值——直接检查 profile 中的 `count==0`
> 语句块，避免 1 位小数的百分比四舍五入掩盖回退。前端已把 `src/i18n/index.ts` 纳入统计（仅排除
> `src/i18n/messages/**` 纯数据），四指标仍 100%。
>
> 本表同时是 [§5.1](#51-风险登记) 末尾「已收敛」索引中各守卫的落地：R1（契约漂移）由
> `handler/api_doc_test.go`（端点 + 请求体字段级**双向**）· `openapi_inputsource_test.go`（输入源）·
> `openapi_request_fields_test.go`（注册表字段集 ⇔ handler 解码结构体字段集**全量遍历**）·
> `openapi_query_params_test.go`（query 四种读取口径 + 绑定变量动态键）·
> `openapi_path_params_test.go`（path 参数 ⇔ handler `PathValue` 双向）·
> `openapi_semantics_test.go`（类型 / required / 枚举）·
> `openapi_response_contract_test.go`（共享 schema + **端点级**响应字段双向）守住，R2（覆盖率掩盖
> 死代码）由 `count==0` + `golangci-lint` 零告警 + `deadcode_gate_test.go` 守住（2026-09-24 扩至生产导出符号
> 零引用，前端同口径半边见 `apps/web/src/deadcode_gate.test.ts` 与 features §Z），R3（明文落盘）由
> `Config.Validate` 硬失败（`ErrPlaintextStoreNotAllowed`，仅 `S3C_ALLOW_PLAINTEXT_STORE=1` 放行）+
> base compose 的 `${S3C_STORE_KEY:?}` + `StorePlaintextWarning` 单测 + 子进程日志断言守住，R4（单副本）由 `TestDataDirLock*` +
> `TestRunServerRejectsLockedDataDir` 守住，R6（Rust 供应链）由本表的 `cargo audit` 守住。
> 发布链与安全基线（tag↔清单、平台内唯一 checksum、Trivy DB 缓存/重试、alpine 版本、`.env.example`
> token）由 `repo_infra_gate_test.go` 守住。
> 门禁变红即对应风险回归，按 §5 的复审规则回写状态。

---

## 五、风险与依赖

> 登记表**只保留还需要人看的项**：仍开放（🟡）的风险。已收敛为自动化门禁的、以及已决策接受
> （➖，ADR 兜底）的取舍都不占用登记行，统一放在 §5.1 末尾的索引里（门禁见 [§四](#四质量门禁基线)，
> 逐条处置证据归档在 [`FEATURES.md`](FEATURES.md) §N / §O）。
>
> **可管理性要求**：每条登记风险都要给出**可观测的触发信号**（或写清「缺口」）；没有信号的条目
> 只是叙述，不进本表。
>
> 编号 `R*`（风险）/ `E*`（依赖）独立于 `docs/KNOWN_ISSUES.md` 的 `#N` 与 §三 的 `#`，不可互指；
> **编号不重排**——已收敛 / 已接受的编号保留在索引里，避免 KNOWN_ISSUES / ADR / 代码注释的引用失真。
>
> 状态：🟡 开放（人工跟踪）；➖ 已决策接受只出现在索引中（ADR 是唯一来源）
> 等级 = 影响面 × 发生概率（高 / 中 / 低），仅用于排序。

### 5.1 风险登记

| # | 风险 | 触发信号（可观测） | 影响 | 等级 | 状态 | 守卫 / 缓解 | 跟踪 |
|---|---|---|---|---|---|---|---|
| R5 | 桌面产物未签名 / macOS 未公证、无自动更新通道 | Release 产物无签名；用户需手动放行；修复版无法自动触达旧客户端 | 分发摩擦 + 安全修复触达不了存量用户 | 高 | 🟡 | 附 `SHA256SUMS`；[DEPLOYMENT.md](DEPLOYMENT.md) §5 明示未公证。**缺口**：无证书、无更新通道 | §三 #1 · todolist #25 · E6 |

**已收敛为自动化门禁（不再占用登记行，2026-09-19 收敛）**

| # | 原风险 | 现守卫（回归即红灯） | 处置证据 |
|---|---|---|---|
| R1 | OpenAPI 契约漂移（字段名 / 输入源位置 / query / path / **响应 schema**） | `api_doc_test.go`（端点双向 diff + 请求体字段级双向）· `openapi_inputsource_test.go`（输入源）· `openapi_request_fields_test.go`（注册表 ⇔ handler 字段集全量遍历）· `openapi_query_params_test.go`（query 四种读取口径 + 绑定变量动态键）· `openapi_path_params_test.go`（path 参数 ⇔ `PathValue` 双向）· `openapi_semantics_test.go`（类型 / required / 枚举）· `openapi_response_contract_test.go`（共享 schema + 端点级响应字段双向） | [`FEATURES.md`](FEATURES.md) §O-1 · §S · §X |
| R2 | 覆盖率掩盖死代码 / `_ = x` 消音 | 覆盖率门禁 `count==0` · `golangci-lint`（`errcheck`/`staticcheck`/`govet`/`ineffassign`/`unused`/`gosec`/`nolintlint` 零告警）· `deadcode_gate_test.go` | [`FEATURES.md`](FEATURES.md) §O-2 · §X |
| R3 | 明文密钥落盘（`sqlite`/`json` + 空 `S3C_STORE_KEY`） | `Config.Validate` 硬失败（`ErrPlaintextStoreNotAllowed`；仅 `S3C_ALLOW_PLAINTEXT_STORE=1` 放行）+ base compose `${S3C_STORE_KEY:?}` 强制 key + `StorePlaintextWarning` 启动告警（opt-in 时）+ 表驱动单测 + 子进程断言 | [`FEATURES.md`](FEATURES.md) §N-1 |
| R4 | 多副本共享同一 `DataDir` | `store.AcquireDataDirLock`（unix flock；**非 unix 为 no-op，已知残留**） | [`FEATURES.md`](FEATURES.md) §O-3 |
| R6 | Rust 依赖审计缺口 | 两套 CI 的 `cargo audit` + `make rust-audit`（当前 0 漏洞，7 条告警已 triage） | [`FEATURES.md`](FEATURES.md) §O-4 |
| R7 | 分段上传缺 `ETag` 不可见 | `upload.test.ts`（缺 ETag → 报错 + `multipartAbort`）· README 兼容性矩阵 | [`FEATURES.md`](FEATURES.md) §O-6 |
| R9 | 发布产物版本与 tag 不一致（tag↔清单无门禁） | `release-desktop.yml` 的 tag↔`tauri.conf.json`/`Cargo.toml` 比对 + `repo_infra_gate_test.go` | [`FEATURES.md`](FEATURES.md) §S |
| R10 | 三平台校验清单互相覆盖（`SHA256SUMS` 竞态） | 平台内唯一 `SHA256SUMS-<bundle>.txt` + `aggregate-checksums` 聚合 job + `repo_infra_gate_test.go` | [`FEATURES.md`](FEATURES.md) §S |
| R11 | 安全门禁因 Trivy DB 下载失败而结构性变红/失明 | 两套 CI 的 DB 预下载 + 退避重试 + `--skip-db-update` + 缓存 + `repo_infra_gate_test.go` | [`FEATURES.md`](FEATURES.md) §S |
| R12 | 运行镜像基础版 EOL（叠加 `--ignore-unfixed` 致漏洞门禁失明） | `Dockerfile` 抬到受支持 alpine + `repo_infra_gate_test.go`（含 EOL 时抬基线的说明） | [`FEATURES.md`](FEATURES.md) §S |
| R13 | `.env.example` 占位 token 是「有效口令」 | token 置空 + `TestEnvExampleTokenIsNotAValidCredential` | [`FEATURES.md`](FEATURES.md) §S |

**已决策接受（不修，ADR 兜底；不再占用登记行）**

| # | 决策 | 兜底与可切换加固 | 依据 |
|---|---|---|---|
| R8 | 存储不可用硬失败（不降级只读） · SSRF 默认放行私网 / 回环 | 健康检查返回 503 + `/api/metrics` 的 `s3c_store_up 0` 可告警（`TestHealthStoreUnavailable` / `TestMetricsStoreUpAndSSRFPolicy`）；SSRF 可设 `S3C_SSRF_DENY_PRIVATE=1` 切换为拒绝，生效值见启动日志 `ssrfDenyPrivate` 与 `s3c_ssrf_deny_private` 指标 | [ADR-002](decisions/0002-store-fail-closed.md) · [ADR-003](decisions/0003-ssrf-private-allow.md) |

### 5.2 依赖清单

| # | 依赖 | 类型 | 被谁依赖 | 当前状态 | 失效后果 | 兜底 / 约定 |
|---|---|---|---|---|---|---|
| E1 | Go `1.26.6` 工具链 | 构建 | `go.mod` / `Dockerfile` / 两套 CI | ✅ `go.mod`、`Dockerfile`、GitLab 镜像三处显式一致（GitHub 侧读 `go-version-file`） | 本地绿 CI 红，或镜像回退到含 CVE 版本 | 升级须同步这四处，`govulncheck` 兜底 |
| E2 | AWS SDK for Go v2（`v1.43.7` / `s3 v1.107.3`） | 库 | 签名 / 预签名 / 分段 / 复制 | ✅ 已 pin | 破坏性升级使签名或错误映射漂移 | `s3wrap` 是唯一边界；升级必须跑 RustFS 真对端 E2E |
| E3 | `modernc.org/sqlite`（纯 Go、无 cgo） | 库 | `sqlite` store 驱动 | ✅ 已 pin | 换驱动需重做加密与并发验证 | `store.Open` 可按驱动切换（json / encrypted） |
| E4 | RustFS `1.0.0-rc.3` 镜像 | 服务 | 本地 compose 联调 + 两套 CI 的真对端 E2E | ⏳ 上游 RC 版本 | 上游行为变化污染 E2E 结论 | 镜像 pin 版本；E2E 只在 E2E 工作流跑 |
| E5 | GitHub Actions（全部 pin SHA）+ dependabot | 供应链 | CI / 发布 | ✅ SHA 经 GitHub API 核验 | 幽灵 SHA 致工作流失败或被伪造 action 执行（曾发生） | 新增 / 升级 action 必须核验 SHA 有效；dependabot 覆盖 5 个生态 |
| E6 | 代码签名证书（Windows）/ Apple Developer ID + 公证 | 外部凭证 | §三 #1 桌面分发（阻塞项） | ⬜ 未获取 | 产物被 SmartScreen / Gatekeeper 拦截 | 暂以 `SHA256SUMS` + 手动放行说明过渡 |
| E7 | GitHub Release（tauri-action + `gh release upload`）+ Windows / macOS runner | 发布通道 | 桌面端分发 | ✅ 已可用 | 桌面产物无法分发 | 无替代通道（有意不镜像到 GitLab CI） |
| E8 | 目标 S3 服务的 CORS / ETag 行为（阿里 / 腾讯 / RustFS / MinIO） | 外部服务 | 浏览器直传与分段上传 | ⚠️ 因厂商而异 | 直传或分段组装失败 | README 兼容性矩阵明示所需 CORS / `ExposeHeader: ETag`；缺 ETag 即报错并清理分段 |
| E9 | pnpm `9.15.0` / Node `26.10.0` / Rust `1.98.1` / Playwright chromium | 构建 | 前端 / 桌面构建与 E2E | ✅ 由 `packageManager`（web + desktop）+ 锁文件 + `rust-toolchain.toml` + CI 精确 patch 版本固定 | 构建 / E2E 结果漂移 | `--frozen-lockfile`；CI 与本地同命令；`repo_infra_gate_test.go` 断言 Node/pnpm/Rust 均为精确 pin |
| E10 | 加密存储文件格式 S3C2 / S3C3 向后兼容承诺 | 内部契约 | `store` 加解密与既有账号库 | ✅ S3C3 头部随文件携带 Argon2id 参数 | 直接改 KDF 参数会让既有库不可解密 | 只增版本、不改既有语义；升级路径已有实跑证据（[FEATURES.md](FEATURES.md)） |
| E11 | **宿主 `NODE_ENV`（尤其 `production`）** | 环境 | 前端 vitest 单测 | ✅ 由 `apps/web/vite.config.ts` 顶部在 `VITEST` 下改写为 `test` 隔离，并由 `src/vite_env_guard.test.ts` 钉住 | Vue 被解析到 prod 构建 ⇒ 进程内两份 Vue 实例、`vi.mock` 打不进组件，**33 文件 / 246 例全红**，症状却是「导出存在却报 `No "x" export is defined`」这类误导信息 | 隔离必须带 `if (process.env.VITEST)` 条件——否则 `vite build` 也读到非 production 值，把 Vue 的 dev/warn 分支打进产物（实测 bundle 366.66 kB → 424.72 kB）。复发即由 `vite_env_guard.test.ts` 红灯拦下 |

**复审规则**：状态随每次五维度评估（[archive/assessment.md](archive/assessment.md)）与里程碑收口同步更新；登记项（🟡）必须写清「缺口」，➖ 由 ADR 兜底并只登记在索引；风险收敛为自动化门禁后移入上方「已收敛」索引（**编号不重排**），闭环证据按 §六 第 1 条归档 `FEATURES.md`。新增外部服务 / 凭证 / 分发通道必须在同一 PR 补 E 表一行。

---

## 六、维护约定

1. **两源分工**：本文件只维护**版本级规划与优先级**，且**只列未完成项**；**功能候选 / 版本级条目以本文件为唯一来源**，不在别处重复登记。**问题**（缺陷 / 外部阻塞 / 技术债）的唯一来源是 [`docs/KNOWN_ISSUES.md`](KNOWN_ISSUES.md)——评估或实现暴露出的*问题*记入那边、*方向*留在本文件，同一事项只在一个文件里是「当前条目」。条目完成后归档至 [`docs/FEATURES.md`](FEATURES.md) 并**从本文件移除**（必要时重编号）；风险条目收敛为自动化门禁后移入 §5.1「已收敛」索引、**编号不重排**。
2. **评估驱动**：每次五维度评估（见 [`docs/archive/assessment.md`](archive/assessment.md)）产出后，按 P0/P1/P2 回写
   本路线图与 [`KNOWN_ISSUES.md`](KNOWN_ISSUES.md)，形成「评估 → 修复 → 再评估」闭环。
3. **状态真实性**：标 ⏳ 必须是工作区/分支已有代码变更，并在合并后改为 ✅（或按第 1 条移除）；禁止保留
   已完成条目的 ✅ 行——历史证据归 `FEATURES.md`。所有门禁数字必须来自实跑。
4. **发版触发**：里程碑验收标准全部满足后，由 `scripts/release-version.sh` 同步版本号并更新
   [`CHANGELOG.md`](../CHANGELOG.md)，再打 tag。
5. **文档同步**：修复 bug 或新增功能完成后必须更新相关文档（对照表见
   [`docs/DEVELOPMENT.md`](DEVELOPMENT.md) §4 与 [`AGENTS.md`](../AGENTS.md)）；本路线图的版本级条目随改动同步。
6. **编号引用**：引用本文件条目必须带前缀或节号——`§三 #N`、`R*`（风险）、`E*`（依赖）；**禁止**新写
   无前缀的 `roadmap #N`（旧编号已作废，见文件头说明）。历史遗留的无前缀引用按该映射说明理解，
   触及对应文件时随手改为正确指向。
