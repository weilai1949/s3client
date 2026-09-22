# s3clinet 路线图（Roadmap）

> 本文件只描述 s3clinet 的**版本规划与优先级**（战略层）：每个里程碑的目标、验收标准与**尚未完成**的条目。
> 已实现 / 已修复的内容不在此流水账，见 [`docs/features.md`](features.md)；
> 逐条待办的**唯一来源**仍是 [`docs/todolist.md`](todolist.md)（战术层）；
> 逐字发布历史见 [`CHANGELOG.md`](../CHANGELOG.md)；
> 评分与问题证据见 [`docs/assessment.md`](assessment.md)（2026-09-16 五维度评估：代码质量 82 /
> 漏洞 72 / 死代码 70 / 服务降级 74 / 自我迭代 90，总分 78）。
>
> 状态图例：⬜ 未开始 · ⏳ 进行中（部分已落地） · ➖ 已决策（不做 / 维持现状）
>
> 本文件的 `#N` 编号**独立于** `docs/todolist.md` 的 `#N`，两者不可互指。
> 代码注释与历史提交里的 `roadmap #N` 指 **2026-09-17 收口前的旧编号**（条目已归档至
> [`features.md`](features.md) §M 与 [`CHANGELOG.md`](../CHANGELOG.md)），**不要按当前编号回读**。
>
> 版本命名：稳定里程碑 **v1.0.0** 后日常发版用时间戳（`v1.0.0-YYYYMMDDHHmmss`），预发布用 `v1.0.0-rcN`。
> 当前版本 **`v1.0.0`**（已打 tag `v1.0.0`）。最后更新：2026-09-22。

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

**结论**：**P0 与 P1 均已清零**，`v1.0.0-rc1` 候选版已收口——4 项发布前必修缺陷（P0，提交 `e03a15e`）与
5 项稳定版门槛项（P1：Go 1.26.6 + `govulncheck`、幽灵 action SHA、OpenAPI 契约对齐、缺失 i18n 键、
异步任务持久化）全部修复并归档。**v1.0.0 / v1.0.x / v1.1.0 三个里程碑的开放条目也已于 2026-09-17
全部完成并从本文件移除**（`docs/api.md` 漂移校验、存储加密、安全审计、可观测性、前端长列表与恢复、
覆盖率去注水），当前唯一未完成项是长期性质的第 1 条：**桌面端分发与签名**——该项为**外部凭证阻塞**
（E6 未获取），代码层面已无剩余工作。

> P0 / P1 的逐条修复记录与验证证据见 [`docs/features.md`](features.md)「H」「I」段，
> 2026-09-17 一轮的验收证据见同文件 §M，不在此重复。本文件只列**未完成**项。

---

## 二、里程碑总览

| 里程碑 | 主题 | 关键验收 | 依赖 |
|---|---|---|---|
| **v1.0.0-rc1** | 候选版收口：P0 四项缺陷清零 | ✅ 已收口（`e03a15e`）；门禁全绿 | — |
| **v1.0.0** | 首个稳定版：P1 清零 + 契约可信 | ✅ **已打 tag**（2026-09-22）：`docs/api.md` 漂移校验落地；分支状态审查 P0/P1/P2 全部闭环 | rc1 |
| **v1.0.x** | 可靠性加固：存储 / 安全审计 / 可观测性 | ✅ 已收口（2026-09-17）：密钥落盘加密；安全事件日志；S3 上游指标与流式失败可见 | v1.0.0 |
| **v1.1.0** | 体验与性能：长列表、错误恢复、i18n | ✅ 已收口（2026-09-17）：grid 窗口化；健康轮询自动恢复；门禁消除注水 | v1.0.x |
| **v1.2+** | 长期：桌面分发、可选增强 | 桌面端签名与自动更新；按需评估 | v1.1.0 |

> 「依赖」列只表示**里程碑前置关系**（上一个里程碑须先收口），不含外部技术 / 供应链 / 凭证依赖——
> 后者见 [§5.2 依赖清单](#52-依赖清单)。

**发布约束**：任一里程碑发布前，[质量门禁基线](#四质量门禁基线)必须全绿；P0 未清零不得进入 v1.0.0。

---

## 三、v1.2+ / 长期

> v1.0.0 / v1.0.x / v1.1.0 三个里程碑的开放条目已于 2026-09-17 全部完成并从本文件移除，
> 逐条验收证据见 [`features.md`](features.md) §M。本节只列仍未完成的长期项。

| # | 条目 | 来源 | 状态 | 说明 |
|---|---|---|---|---|
| 1 | 桌面端分发与签名 | 长期 | ⛔ | **已立项**（todolist #25），**外部阻塞**：Windows 代码签名证书 / Apple Developer ID + 公证均为外部凭证（**E6**，⬜ 未获取），未获取前无法完成签名与公证；自动更新通道依赖签名产物。**代码层面无剩余工作**——打包与发布链（tag↔清单校验、平台内唯一 `SHA256SUMS`、聚合 job）已收口，未签名产物以 `SHA256SUMS` + 手动放行说明过渡（[deployment.md](deployment.md) §5）；风险 **R5** 见 §五 |
| 2 | 增量同步与批量能力的体验增强 | FEATURES | ➖ | 现有 `etag` / `size_mtime` / `always` 三模式满足需求，按用户反馈再评估 |
| 3 | 死代码纪律 | ASSESSMENT §三 | ➖ | 维持现状：由覆盖率门禁调整一并治理，不单独立项 |

---

## 四、质量门禁基线

任一版本发布前必须全绿（当前实测状态，2026-09-19）：

| 门禁 | 命令 | 当前状态 |
|---|---|---|
| Go 格式 | `gofmt -l .`（`apps/server/`） | ✅ 干净 |
| Go 静态检查 | `go vet ./...` | ✅ 0 告警 |
| Go 测试 | `go test -race -count=1 ./...` | ✅ 8/8 包通过 |
| Go 覆盖率 | `make test-cover`（检查 profile 中 `count==0` 语句块） | ✅ 每包 + 汇总均 100.0% statements；CI 硬门禁 100% |
| Go 漏洞 | `govulncheck ./...` | ✅ 0 可达漏洞（go1.26.6；已入 CI 门禁） |
| 前端 lint | `pnpm lint` | ✅ 0 error / 0 warning |
| 前端类型 | `vue-tsc --noEmit` | ✅ exit 0 |
| 前端测试 | `pnpm test` | ✅ 986 例全绿（64 文件） |
| 前端覆盖率 | statements / branches / functions / lines | ✅ 100%（含 `src/i18n/index.ts`） |
| 依赖审计 | `pnpm audit` / Trivy | ✅ npm 0 漏洞；镜像 CRITICAL/HIGH 硬失败 |
| E2E | Playwright（`e2e.yml` + `e2e-playwright.yml`） | ✅ 全 action SHA 经 GitHub API 核验 |
| Rust 依赖审计 | `cargo audit`（两套 CI 的 desktop job + `make rust-audit`） | ✅ 0 漏洞；7 条 unmaintained/unsound 告警已 triage |

> 后端覆盖率已补齐至**每包 100%**（2026-09 删除了确实不可达的防御分支，其余缺口改用行为断言，
> 不做 gap 测试），CI 与 `make test-cover` 以 100% 为硬阈值——直接检查 profile 中的 `count==0`
> 语句块，避免 1 位小数的百分比四舍五入掩盖回退。前端已把 `src/i18n/index.ts` 纳入统计（仅排除
> `src/i18n/messages/**` 纯数据），四指标仍 100%。
>
> 本表同时是 [§5.1](#51-风险登记) 末尾「已收敛」索引中各守卫的落地：R1（契约漂移）由
> `handler/api_doc_test.go`（端点 + 请求体字段级**双向**）· `openapi_inputsource_test.go`（输入源）·
> `openapi_request_fields_test.go`（注册表字段集 ⇔ handler 解码结构体字段集**全量遍历**）·
> `openapi_response_contract_test.go`（共享 schema ⇔ Go DTO **响应**字段双向）守住，R2（覆盖率掩盖
> 死代码）由 `count==0` + `golangci-lint` 零告警 + `deadcode_gate_test.go` 守住，R3（明文落盘）由
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
> 逐条处置证据归档在 [`features.md`](features.md) §N / §O）。
>
> **可管理性要求**：每条登记风险都要给出**可观测的触发信号**（或写清「缺口」）；没有信号的条目
> 只是叙述，不进本表。
>
> 编号 `R*`（风险）/ `E*`（依赖）独立于 `docs/todolist.md` 的 `#N` 与 §三 的 `#`，不可互指；
> **编号不重排**——已收敛 / 已接受的编号保留在索引里，避免 todolist / ADR / 代码注释的引用失真。
>
> 状态：🟡 开放（人工跟踪）；➖ 已决策接受只出现在索引中（ADR 是唯一来源）
> 等级 = 影响面 × 发生概率（高 / 中 / 低），仅用于排序。

### 5.1 风险登记

| # | 风险 | 触发信号（可观测） | 影响 | 等级 | 状态 | 守卫 / 缓解 | 跟踪 |
|---|---|---|---|---|---|---|---|
| R5 | 桌面产物未签名 / macOS 未公证、无自动更新通道 | Release 产物无签名；用户需手动放行；修复版无法自动触达旧客户端 | 分发摩擦 + 安全修复触达不了存量用户 | 高 | 🟡 | 附 `SHA256SUMS`；[deployment.md](deployment.md) §5 明示未公证。**缺口**：无证书、无更新通道 | §三 #1 · todolist #25 · E6 |

**已收敛为自动化门禁（不再占用登记行，2026-09-19 收敛）**

| # | 原风险 | 现守卫（回归即红灯） | 处置证据 |
|---|---|---|---|
| R1 | OpenAPI 契约漂移（字段名 / 输入源位置 / **响应 schema**） | `api_doc_test.go`（端点双向 diff + 请求体字段级双向）· `openapi_inputsource_test.go`（输入源）· `openapi_request_fields_test.go`（注册表 ⇔ handler 字段集全量遍历）· `openapi_response_contract_test.go`（共享 schema ⇔ Go DTO 响应字段双向） | [`features.md`](features.md) §O-1 · §S |
| R2 | 覆盖率掩盖死代码 / `_ = x` 消音 | 覆盖率门禁 `count==0` · `golangci-lint`（`unused`/`staticcheck`）· `deadcode_gate_test.go` | [`features.md`](features.md) §O-2 |
| R3 | 明文密钥落盘（`sqlite`/`json` + 空 `S3C_STORE_KEY`） | `Config.Validate` 硬失败（`ErrPlaintextStoreNotAllowed`；仅 `S3C_ALLOW_PLAINTEXT_STORE=1` 放行）+ base compose `${S3C_STORE_KEY:?}` 强制 key + `StorePlaintextWarning` 启动告警（opt-in 时）+ 表驱动单测 + 子进程断言 | [`features.md`](features.md) §N-1 |
| R4 | 多副本共享同一 `DataDir` | `store.AcquireDataDirLock`（unix flock；**非 unix 为 no-op，已知残留**） | [`features.md`](features.md) §O-3 |
| R6 | Rust 依赖审计缺口 | 两套 CI 的 `cargo audit` + `make rust-audit`（当前 0 漏洞，7 条告警已 triage） | [`features.md`](features.md) §O-4 |
| R7 | 分段上传缺 `ETag` 不可见 | `upload.test.ts`（缺 ETag → 报错 + `multipartAbort`）· README 兼容性矩阵 | [`features.md`](features.md) §O-6 |
| R9 | 发布产物版本与 tag 不一致（tag↔清单无门禁） | `release-desktop.yml` 的 tag↔`tauri.conf.json`/`Cargo.toml` 比对 + `repo_infra_gate_test.go` | [`features.md`](features.md) §S |
| R10 | 三平台校验清单互相覆盖（`SHA256SUMS` 竞态） | 平台内唯一 `SHA256SUMS-<bundle>.txt` + `aggregate-checksums` 聚合 job + `repo_infra_gate_test.go` | [`features.md`](features.md) §S |
| R11 | 安全门禁因 Trivy DB 下载失败而结构性变红/失明 | 两套 CI 的 DB 预下载 + 退避重试 + `--skip-db-update` + 缓存 + `repo_infra_gate_test.go` | [`features.md`](features.md) §S |
| R12 | 运行镜像基础版 EOL（叠加 `--ignore-unfixed` 致漏洞门禁失明） | `Dockerfile` 抬到受支持 alpine + `repo_infra_gate_test.go`（含 EOL 时抬基线的说明） | [`features.md`](features.md) §S |
| R13 | `.env.example` 占位 token 是「有效口令」 | token 置空 + `TestEnvExampleTokenIsNotAValidCredential` | [`features.md`](features.md) §S |

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
| E9 | pnpm `9.15.0` / Node `24.21.0` / Rust `1.98.1` / Playwright chromium | 构建 | 前端 / 桌面构建与 E2E | ✅ 由 `packageManager`（web + desktop）+ 锁文件 + `rust-toolchain.toml` + CI 精确 patch 版本固定 | 构建 / E2E 结果漂移 | `--frozen-lockfile`；CI 与本地同命令；`repo_infra_gate_test.go` 断言 Node/pnpm/Rust 均为精确 pin |
| E10 | 加密存储文件格式 S3C2 / S3C3 向后兼容承诺 | 内部契约 | `store` 加解密与既有账号库 | ✅ S3C3 头部随文件携带 Argon2id 参数 | 直接改 KDF 参数会让既有库不可解密 | 只增版本、不改既有语义；升级路径已有实跑证据（[features.md](features.md)） |

**复审规则**：状态随每次五维度评估（[assessment.md](assessment.md)）与里程碑收口同步更新；登记项（🟡）必须写清「缺口」，➖ 由 ADR 兜底并只登记在索引；风险收敛为自动化门禁后移入上方「已收敛」索引（**编号不重排**），闭环证据按 §六 第 1 条归档 `features.md`。新增外部服务 / 凭证 / 分发通道必须在同一 PR 补 E 表一行。

---

## 六、维护约定

1. **单一来源**：本文件只维护**版本级规划与优先级**，且**只列未完成项**；新增 / 关闭具体条目时，同步更新
   [`docs/todolist.md`](todolist.md)（唯一待办来源），完成后归档至 [`docs/features.md`](features.md)
   并**从本文件移除该条目**（必要时重编号）；风险条目收敛为自动化门禁后移入 §5.1「已收敛」索引、**编号不重排**。
2. **评估驱动**：每次五维度评估（见 [`docs/assessment.md`](assessment.md)）产出后，按 P0/P1/P2 回写
   本路线图与 todolist，形成「评估 → 修复 → 再评估」闭环。
3. **状态真实性**：标 ⏳ 必须是工作区/分支已有代码变更，并在合并后改为 ✅（或按第 1 条移除）；禁止保留
   已完成条目的 ✅ 行——历史证据归 `features.md`。所有门禁数字必须来自实跑。
4. **发版触发**：里程碑验收标准全部满足后，由 `scripts/release-version.sh` 同步版本号并更新
   [`CHANGELOG.md`](../CHANGELOG.md)，再打 tag。
5. **文档同步**：修复 bug 或新增功能完成后必须更新相关文档（对照表见
   [`docs/development.md`](development.md) §4 与 [`AGENTS.md`](../AGENTS.md)）；本路线图的版本级条目随改动同步。
6. **编号引用**：引用本文件条目必须带前缀或节号——`§三 #N`、`R*`（风险）、`E*`（依赖）；**禁止**新写
   无前缀的 `roadmap #N`（旧编号已作废，见文件头说明）。历史遗留的无前缀引用按该映射说明理解，
   触及对应文件时随手改为正确指向。
