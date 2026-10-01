# 事故复盘：预签名失败返回 `200 {"url":""}`（PM-20260916-presign-empty-url）

> **本文件是 [`POSTMORTEM_TEMPLATE.md`](../POSTMORTEM_TEMPLATE.md) 的首份已填写实例**，按模板 §1–§10 逐节填写
> （§0「适用范围与填写须知」、§7.1「演练补充字段」、§11「刻意不定义的事」属模板元条款与演练专用节，未复制）。
> 对象是一桩**真实代码险情（near-miss）**：2026-09-05 引入 → 2026-09-16 评估发现 → 2026-09-17 修复闭环。
>
> **取证纪律**：全部字段以 **git 取证与既有台账**为证据（commit SHA / `assessment` §二 L1 / `FEATURES` §J /
> `CHANGELOG` / 待办编号台账）；没有运行期实测值的字段一律写「**未留存 + 原因**」——
> **不编造时间戳、日志、指标与报障记录**。
>
> **归档状态（2026-09-30）**：事件已闭环、行动项全部完成，按 [`archive/index.md`](index.md)「归档操作」四步
> `git mv` 冻结进 `docs/archive/`——**不删除、不回写、不改写历史结论**；当前事实以
> [`FEATURES.md`](../FEATURES.md) §J 与 [`../../CHANGELOG.md`](../../CHANGELOG.md) 为准。

## 1. 元信息

| 字段 | 填写 |
|---|---|
| 复盘编号 | `PM-20260916-presign-empty-url` |
| 事件等级 | **P2（near-miss 险情）**——按 [`OPERATIONS.md`](../OPERATIONS.md) §9.1「单功能受损」的**潜在**影响定级；实际由 2026-09-16 例行评估实测发现，**无用户报障、无线上故障记录** |
| 发生 / 发现 / 恢复时间 | 发生（缺陷合入）`2026-09-05 12:51 +08:00`（`b3b287c`）· 发现 `2026-09-16`（评估日，**具体时刻未留存**）· 恢复（修复合入）`2026-09-17 12:26 +08:00`（`5954bfa`） |
| 影响时长 | 缺陷在库 **11 天 23 小时**（09-05 12:51 → 09-17 12:26，+08:00）；发现 → 修复 **≤ 1 天**；**无停机、无数据回滚、人工止损耗时 0**（无运行期止血动作，修复即恢复） |
| 发现方式 | **例行评估实测**（2026-09-16 五维度综合评估，[`assessment.md`](assessment.md) §二 L1）——非告警、非用户报障 → 决定 §6 必须补「为什么没有自动检测」这一环 |
| 记录人 / 复核人 | @weilai1949（单人维护：记录人即维护者本人，模板「建议」口径） |
| 关联条目 | `KNOWN_ISSUES` #23（现已闭环移除，见其编号台账「#1–#24 已闭环移除」行）· `ASSESSMENT` L1 · commit `b3b287c`（引入）/ `5954bfa`（修复）/ `1f32dca`（文档同步）/ `0fbd560`（#23 余项）· [`FEATURES.md`](../FEATURES.md) §J P2-1 · [`../../CHANGELOG.md`](../../CHANGELOG.md) `[Unreleased]`「P2 加固修复（2026-09-16 评估）」段 |

## 2. 摘要（≤3 行）

预签名调用的错误被 `u, _ :=` 吞掉，签名失败时接口仍返回 `200 {"url":""}`——账号 presign（get / post / put）与分段 presign 四处调用点在异常路径下「显示成功却拿不到链接」，调用方无法区分服务端故障与自身渲染问题。
根因是 `b3b287c` 为把语句覆盖率推到 100%，把**测试触达不到**的错误分支当作「不可达死代码」删除，并留下「过期已钳制、`PresignXxx` 实际不可能失败」的错误断言。
2026-09-17 以统一出口 `writePresignResult`（失败 500）修复，并加源码级门禁 `TestPresignErrorsNotSwallowed` 防复发。

## 3. 影响面

| 维度 | 填写 |
|---|---|
| 账号 / 桶 / 对象 | **无对象侧影响**——签名失败发生在任何 S3 写入之前。受影响的是预签名相关**功能的失败路径**：`POST /api/accounts/{id}/presign`（`method` = get / post / put）与 `POST /api/accounts/{id}/multipart/part` |
| 数据完整性 | **无丢失 / 覆盖 / 明文落盘**；无半成品——签名失败 → 前端拿不到 URL → 不发起写请求（`GET /api/migrate/jobs` 的 `interrupted` 对账不适用） |
| 对外可用性 | `/api/health` 正常、无 5xx 上升；**故障路径此前被计为 2xx**，观测上是「零错误」。修复后同路径返回 500：全路由经 `withMetricsGate` 全局包装（`apps/server/internal/handler/routes.go`），计入 `s3c_http_responses_total{class="5xx"}`（[`OPERATIONS.md`](../OPERATIONS.md) §3.2）。**无实测 5xx 比例**：事发时未开启 `S3C_EXPOSE_METRICS` 采集，事件亦非线上运行故障 |
| 安全影响 | **无**未授权访问，**无** `S3C_TOKEN` / `S3C_STORE_KEY` 泄露。但吞错**掩盖了上游凭证类故障**（取凭证失败、`SignatureDoesNotMatch` 等本应进日志与 `s3c_s3_call_errors_total` 的信号被静默丢弃）——属观测风险，已随修复消除 |

## 4. 时间线

> 每行带时间戳与证据；本事件无线上请求，故无 `X-Request-ID` 行（[`OPERATIONS.md`](../OPERATIONS.md) §3.3 的
> `rid=` / `req=` 跨层关联字段在事发时**尚不存在**，`#68` 于 2026-09-29 才补）。

| 时间（+08:00） | 类型 | 内容 | 证据 |
|---|---|---|---|
| 2026-09-05 12:51 | 动作（**引入回归**） | `b3b287c`「test: 后端覆盖率提升至 100%」把 `objects.go` 的 get / post / put 与 `multipart.go` 的 part 四处 `u, err := client.PresignXxx(...)` 改为 `u, _ := ...` 并删除错误分支，注释断言「过期被钳制到 [1h, 24h]，`PresignGetVersion` 实际不可能失败」；同批还删了 store / service / s3wrap 多处防御性检查 | `git show b3b287c`（diff 内 `-` 行是被删的错误处理、`+` 行是吞错写法）；`git log -S 'u, _ := client.Presign'` 确认该写法**此前不存在** |
| 2026-09-16（时刻未留存） | **发现** | 五维度综合评估实测：**空 key、context 取消**均可触发预签名失败 → 响应 `200 {"url":""}` | [`assessment.md`](assessment.md) §二 L1（证据列 `objects.go:441-453`、`multipart.go:80`——**行号为审计时点值，已漂移**，现以符号为准） |
| 2026-09-16 | 决策 | 登记 `todolist` #23（今 `KNOWN_ISSUES` #23），纳入 P2 加固批次 | [`FEATURES.md`](../FEATURES.md) §J 表头「来源」行 |
| 2026-09-17 12:26:19 | 动作（**修复**） | `5954bfa`：新增 `writePresignResult` 统一四处调用点，失败返回 `500 failed to create presigned url`；新增源码级门禁 `TestPresignErrorsNotSwallowed`，**回退 `multipart.go` 一行验证门禁确实失败** | `git log -1 5954bfa`（提交说明含触发条件实测、否决方案与变异验证） |
| 2026-09-17 12:26:30 | 动作（**文档同步**） | `1f32dca`：待办 #23 **部分关闭**（预签名已修；「错误消息回显用户输入」余项仍开放） | `git log -1 1f32dca` 的「文档同步」段 |
| 2026-09-19 | 动作（**余项闭环**） | `0fbd560` 落地 `error_echo_gate_test.go`（`CHANGELOG` 记为 roadmap #6 / todolist #23），#23 随后整条从清单移除 | `git log --diff-filter=A -- '*error_echo_gate_test.go'`；[`KNOWN_ISSUES.md`](../KNOWN_ISSUES.md) 编号台账 |
| 2026-09-30 16:1x | 观测（**闭环复核**） | 复跑三条相关测试 **3/3 PASS**；变异把 `objects.go` get 分支改回 `u, _ := ...` → 门禁红灯点名「`objects.go:418` 忽略了预签名错误（会返回 200 + 空 url）」→ 还原后绿灯 | 本文件 §8 第 2 条 |

## 5. 取证清单

> 勾选「已留存」；未留存的写原因——**本事件是代码险情（near-miss），证据在仓库而非运行现场**，
> 运行期取证项在事发时均无对应实例可取。

| 证据 | 已留存 | 位置 / 命令与**未留存原因** |
|---|---|---|
| `/api/health` 响应体 | ☐ | 未留存：非运行期事件，无故障实例可取证（发现方式为评估实测，非告警 / 报障） |
| `/api/metrics` 全文 | ☐ | 未留存：当时未开启 `S3C_EXPOSE_METRICS`（默认 404），且无运行期故障窗口 |
| 最近 15 分钟 server 日志 | ☐ | 未留存：无用户报障、无故障窗口可回溯 |
| nginx 访问日志 | ☐ | 未留存：同上；且跨层 `rid=` / `req=` 关联字段事发时**尚不存在**（`#68`，2026-09-29 补） |
| 数据目录状态 | ☐ | 未留存：与本事件无关——签名失败不触碰数据目录，无数据侧影响（`ls -l "$S3C_DATA_DIR"` 无取证价值） |
| `jobs.json` 副本 | ☐ | 未留存：本事件不涉及异步任务与半成品 |
| 审计事件检索结果 | ☐ | 未留存：签名失败非破坏性操作，不产生 `msg="audit"` 事件 |

**实际留存的证据**（仓库内，可逐条复跑）：`git show b3b287c`（引入 diff）· [`assessment.md`](assessment.md) §二 L1（发现记录）· `git log -1 5954bfa`（修复 + 变异验证说明）· `git log -1 1f32dca`（文档同步）· `apps/server/internal/handler/presign_error_test.go`（3 条测试）· [`FEATURES.md`](../FEATURES.md) §J P2-1 · [`../../CHANGELOG.md`](../../CHANGELOG.md)「P2 加固修复（2026-09-16 评估）」段 · [`KNOWN_ISSUES.md`](../KNOWN_ISSUES.md) 编号台账（#23 已闭环移除）。

## 6. 根因分析

- **触发因素**（导火索）：`b3b287c` 为把 7 个包的语句覆盖率推到 100%，按「**测试不可达 = 不可达**」的口径删除防御性错误分支——预签名四处调用改为 `u, _ := ...`，并写入注释断言「过期被钳制到 [1h, 24h]，`PresignXxx` 实际不可能失败」。
- **根因**（5 Why，追问到可改动的一层）：
  1. 为什么签名失败会回 `200 {"url":""}`？→ 错误值被 `_` 丢弃，handler 恒走成功分支。
  2. 为什么敢丢？→ 注释断言「不可能失败」——把**测试可达性**错当成**运行时可达性**；AWS SDK 在取凭证、输入序列化阶段都会返回错误（修复时实测**空 key、context 取消**均可触发，断言不成立）。
  3. 为什么这条断言没人拦？→ 当时**没有**「不得吞掉外部调用错误」的机械门禁；覆盖率目标反而成了删防御的理由。
  4. 为什么行为漂移没人拦？→ 失败路径**无测试**：`200 {"url":""}` 从未被断言为错误；前端把空 URL 当成功渲染，也不报错。
  5. **可改动的一层** → **代码 + 测试门禁**：统一错误出口（`writePresignResult`）+ 行为测试（`TestWritePresignResultError` / `Success`）+ 源码级门禁（`TestPresignErrorsNotSwallowed`）。三层均已落地（§9）。
- **为什么没被更早发现**：
  - **告警面不存在**：事发时（09-05 → 09-16）仓库尚无 SLO / 告警基线与 Runbook——[`OPERATIONS.md`](../OPERATIONS.md) §4 / §5 于 **2026-09-29** 才建立（`a984df7`）。
  - **指标看不见**：HTTP 侧故障被记成 2xx（正是本缺陷的表象）；S3 侧 `s3c_s3_call_errors_total` **不含**预签名（presign client 刻意不挂 `metricsMiddleware`，见 [`FEATURES.md`](../FEATURES.md) R14–R17 行）——两头都没有「预签名失败」信号。
  - **测试看不见**：`e2e-real` 浏览器直传联调 2026-09-22 才落地（`todolist` #37），且只覆盖**成功路径**。
  - **唯一防线是周期性人工评估**——本事件正是被 2026-09-16 五维度评估实测抓到。
- **同类风险是否仍在**：本类写法已被 `TestPresignErrorsNotSwallowed` 堵死（2026-09-30 复核：改回 `u, _ :=` 即红灯点名，见 §8）；失败现返回 500 → 进 `s3c_http_responses_total{class="5xx"}`，被 [`OPERATIONS.md`](../OPERATIONS.md) §4.2 的 5xx 告警建议值覆盖。**当前无开放项**，故不在 [`KNOWN_ISSUES.md`](../KNOWN_ISSUES.md) 新登记（该清单只收**尚未闭环**的项）；「为覆盖率删防御」的治理约束见仓库硬约束的死代码纪律与 `deadcode_gate`。

## 7. 检测与响应评估

| 指标 | 实测 | 对照 | 差距 |
|---|---|---|---|
| 检测时延（发生 → 发现） | **≥ 10 天**（2026-09-05 12:51 → 2026-09-16 评估；评估具体时刻未留存） | [`OPERATIONS.md`](../OPERATIONS.md) §4.1 建议 SLI / SLO 只覆盖**运行期**错误率，不覆盖此类缺陷 | **无自动检测**——防线只有周期性人工评估；告警基线当时尚不存在（2026-09-29 才建） |
| 止损时延（发现 → 止血） | **≤ 1 天**（09-16 发现 → 09-17 12:26 修复合入） | 无既定基线（§4 只有检测侧建议值） | 无独立止血动作：缺陷只影响**失败路径**、未扩散，**修复即止血** |
| 恢复时延（止血 → 恢复） | **0**（修复与门禁同一提交完成；无停机、无数据回滚） | §7.1 建议 RTO / RPO **不适用**（无运行中断、无数据丢失） | — |
| 既有 Runbook 是否命中 | **全部未命中**——事发时仓库尚无 Runbook（[`OPERATIONS.md`](../OPERATIONS.md) §5 的 R-1..R-10 于 2026-09-29 才建立） | §5 的 R-1..R-10 | 按模板「未命中须补一条」处置：症状类 Runbook **已随后落地**——**R-6**（S3 上游错误率上升）/ **R-7**（浏览器直传失败）即本类症状的处置入口；本缺陷属**代码缺陷**，其专属防线是源码门禁而非运行手册 |

## 8. 处置与验证

1. **实际执行的动作**（与 §4 时间线一一对应；被否决的方案也列出）：
   - **采用**：新增 `writePresignResult(w, err, body)` 统一出口——`err != nil` → `500 failed to create presigned url` 且不回 body；`err == nil` → 200 原 body。四处调用点（`objects.go` 的 get / post / put、`multipart.go` 的 part）全部改用。
   - **否决 A**：保留 `u, _ :=` 只补一条注释 → **无机械保证**，等于把同一个错误断言再留一遍（`b3b287c` 正是这么错的）。
   - **否决 B**：在测试里用「取消请求上下文」注入失败 → AWS SDK 会缓存已解析凭证，同一 client 首次失败后后续调用不再失败，据此写的测试会 **flaky**（`5954bfa` 提交说明）。改为**单测统一错误处理函数 + 源码级门禁**。
2. **恢复验证**：本事件**无运行期恢复动作**，[`OPERATIONS.md`](../OPERATIONS.md) §6.4 的六步**不适用**（无实例可验，理由同 §5）。以门禁复跑替代，**2026-09-30 实测**：
   - `cd apps/server && go test ./internal/handler/ -run 'TestWritePresignResult|TestPresignErrorsNotSwallowed' -count=1` → **3/3 PASS**；
   - **变异复核**：把 `objects.go` get 分支改回 `u, _ := ...`（并把错误参数写死 `nil`）→ `TestPresignErrorsNotSwallowed` **红灯点名**「`objects.go:418` 忽略了预签名错误（会返回 200 + 空 url）」→ 还原后**绿灯**，`git status` 无残留。
3. **回滚**：不涉及——修复未改变成功路径行为，无存储格式（`S3C2` / `S3C3`）变更，无需按 [`DEPLOYMENT.md`](../DEPLOYMENT.md) §7 回滚。

## 9. 行动项

| # | 行动项 | 类型 | 负责人 | 截止 | 跟踪位置 | 状态 |
|---|---|---|---|---|---|---|
| 1 | 统一预签名错误出口：失败返回 500 而非 `200 {"url":""}`（四处调用点） | 代码 | @weilai1949 | 2026-09-17（已完成） | commit `5954bfa` · [`FEATURES.md`](../FEATURES.md) §J P2-1 | ☑ |
| 2 | 源码级门禁禁止 `,…_ := …Presign` 复发，并做变异验证 | 测试门禁 | @weilai1949 | 2026-09-17（已完成） | `TestPresignErrorsNotSwallowed`（`presign_error_test.go`） | ☑ |
| 3 | 行为测试钉住失败 / 成功两条路径 | 测试门禁 | @weilai1949 | 2026-09-17（已完成） | `TestWritePresignResultError` / `TestWritePresignResultSuccess` | ☑ |
| 4 | 台账与文档同步（评估发现项 → 待办编号 → 发版记录），含 #23 余项「错误文案回显」收口 | 文档 | @weilai1949 | 2026-09-19（已完成） | `1f32dca` · `0fbd560`（`error_echo_gate_test.go`）· [`../../CHANGELOG.md`](../../CHANGELOG.md) | ☑ |
| 5 | 本类症状的观测与手册基线：5xx 告警建议值 + R-6 / R-7 Runbook | 告警 / Runbook | @weilai1949 | 2026-09-29（已完成） | [`OPERATIONS.md`](../OPERATIONS.md) §4.2 / §5 | ☑ |

> 第 5 条是**事后补齐**项（事发时告警基线与 Runbook 尚不存在），登记在此以免读者误以为当时已有；
> 它不是本事件发生时立项的行动项。行动项 1–4 才是 2026-09-16/17 当轮的处置。

## 10. 收口与文档同步

| 面 | 是否需要 | 落点 |
|---|---|---|
| 缺陷 / 技术债 / 外部阻塞 | 是（曾登记） | [`KNOWN_ISSUES.md`](../KNOWN_ISSUES.md) #23——预签名部分 2026-09-17 关闭、余项 2026-09-19 收口，**整条已闭环移除**（编号台账「#1–#24 已闭环移除」行）；当前**无开放项** |
| 已修复能力与证据 | 是 | [`FEATURES.md`](../FEATURES.md) §J P2-1 |
| 发版记录 | 是 | [`../../CHANGELOG.md`](../../CHANGELOG.md) `[Unreleased]`「P2 加固修复（2026-09-16 评估）」段（「预签名失败不再被静默吞掉」条） |
| Runbook / SLO / 告警阈值 | 是（事后基线） | [`OPERATIONS.md`](../OPERATIONS.md) §4.2（5xx 告警建议值）/ §5（R-6 / R-7）；规则文件 `deploy/prometheus/s3clinet.rules.yml` 由 `TestPrometheusRulesReferenceRealMetrics` 校验。本事件**未新增指标** |
| 安全边界 / 已接受风险 | 不需要 | 无安全边界变化、无新接受的风险——[`threat-model.md`](../threat-model.md) 无需登记；吞错的观测风险已随修复消除 |
| 用户可见行为 / 排障 | 是 | 行为变更（失败由 200 → 500）记入 [`../../CHANGELOG.md`](../../CHANGELOG.md)；[`user-guide.md`](../user-guide.md) 不改——正常路径行为不变、FAQ 无对应条目 |
| 本复盘存档 | 是 | `docs/archive/incident-20260916-presign-empty-url.md`（本文件）+ [`index.md`](index.md) 归档清单登记一行 |
