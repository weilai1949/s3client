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
>
> 版本命名：稳定里程碑 **v1.0.0** 后日常发版用时间戳（`v1.0.0-YYYYMMDDHHmmss`），预发布用 `v1.0.0-rcN`。
> 当前版本 **`v1.0.0-rc1`**。最后更新：2026-09-17。

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
覆盖率去注水），当前唯一未完成项是长期性质的第 1 条：**桌面端分发与签名**。

> P0 / P1 的逐条修复记录与验证证据见 [`docs/features.md`](features.md)「H」「I」段，
> 2026-09-17 一轮的验收证据见同文件 §M，不在此重复。本文件只列**未完成**项。

---

## 二、里程碑总览

| 里程碑 | 主题 | 关键验收 | 依赖 |
|---|---|---|---|
| **v1.0.0-rc1** | 候选版收口：P0 四项缺陷清零 | ✅ 已收口（`e03a15e`）；门禁全绿 | — |
| **v1.0.0** | 首个稳定版：P1 清零 + 契约可信 | ✅ 已收口（2026-09-17）：`docs/api.md` 漂移校验落地 | rc1 |
| **v1.0.x** | 可靠性加固：存储 / 安全审计 / 可观测性 | ✅ 已收口（2026-09-17）：密钥落盘加密；安全事件日志；S3 上游指标与流式失败可见 | v1.0.0 |
| **v1.1.0** | 体验与性能：长列表、错误恢复、i18n | ✅ 已收口（2026-09-17）：grid 窗口化；健康轮询自动恢复；门禁消除注水 | v1.0.x |
| **v1.2+** | 长期：桌面分发、可选增强 | 桌面端签名与自动更新；按需评估 | v1.1.0 |

**发布约束**：任一里程碑发布前，[质量门禁基线](#四质量门禁基线)必须全绿；P0 未清零不得进入 v1.0.0。

---

## 三、v1.2+ / 长期

> v1.0.0 / v1.0.x / v1.1.0 三个里程碑的开放条目已于 2026-09-17 全部完成并从本文件移除，
> 逐条验收证据见 [`features.md`](features.md) §M。本节只列仍未完成的长期项。

| # | 条目 | 来源 | 状态 | 说明 |
|---|---|---|---|---|
| 1 | 桌面端分发与签名 | 长期 | ⬜ | Windows / Linux / macOS 产物签名与自动更新策略（现仅打包挂 Release） |
| 2 | 增量同步与批量能力的体验增强 | FEATURES | ➖ | 现有 `etag` / `size_mtime` / `always` 三模式满足需求，按用户反馈再评估 |
| 3 | 死代码纪律 | ASSESSMENT §三 | ➖ | 维持现状：由覆盖率门禁调整一并治理，不单独立项 |

---

## 四、质量门禁基线

任一版本发布前必须全绿（当前实测状态，2026-09-17）：

| 门禁 | 命令 | 当前状态 |
|---|---|---|
| Go 格式 | `gofmt -l .`（`apps/server/`） | ✅ 干净 |
| Go 静态检查 | `go vet ./...` | ✅ 0 告警 |
| Go 测试 | `go test -race -count=1 ./...` | ✅ 8/8 包通过 |
| Go 覆盖率 | `make test-cover`（检查 profile 中 `count==0` 语句块） | ✅ 每包 + 汇总均 100.0% statements；CI 硬门禁 100% |
| Go 漏洞 | `govulncheck ./...` | ✅ 0 可达漏洞（go1.26.6；已入 CI 门禁） |
| 前端 lint | `pnpm lint` | ✅ 0 error / 0 warning |
| 前端类型 | `vue-tsc --noEmit` | ✅ exit 0 |
| 前端测试 | `pnpm test` | ✅ 983 例全绿（63 文件） |
| 前端覆盖率 | statements / branches / functions / lines | ✅ 100%（含 `src/i18n/index.ts`） |
| 依赖审计 | `pnpm audit` / Trivy | ✅ npm 0 漏洞；镜像 CRITICAL/HIGH 硬失败 |
| E2E | Playwright（`e2e.yml` + `e2e-playwright.yml`） | ✅ 全 action SHA 经 GitHub API 核验 |

> 后端覆盖率已补齐至**每包 100%**（2026-09 删除了确实不可达的防御分支，其余缺口改用行为断言，
> 不做 gap 测试），CI 与 `make test-cover` 以 100% 为硬阈值——直接检查 profile 中的 `count==0`
> 语句块，避免 1 位小数的百分比四舍五入掩盖回退。前端已把 `src/i18n/index.ts` 纳入统计（仅排除
> `src/i18n/messages/**` 纯数据），四指标仍 100%。

---

## 五、风险与依赖

| 风险 | 影响 | 缓解 |
|---|---|---|
| OpenAPI 契约作为 SSOT 仍可能漂移 | 客户端按文档调用 400 | 契约测试覆盖全部 `requestBody` 端点；`api_doc_test.go` 对 `docs/api.md` ↔ `routes.go` 做双向 diff |
| 覆盖率门禁掩盖死代码 | 维护成本高、真实覆盖失真 | 已删除不可达分支、纳入 i18n 统计、清空 `count==0`；`golangci-lint` 的 `unused` / `staticcheck` 保持零告警 |
| SQLite 驱动密钥明文 | 落盘密钥泄露 | `secret_key` 列已按 S3C3 加密落盘（`S3C_STORE_KEY` ≥16）；`docker-compose.prod.yml` 用 `encrypted` |
| Argon2 参数升级的兼容性 | 直接调参将使既有加密库不可解密 | S3C3 头部携带 KDF 参数、S3C2 双版本读取，升级路径已实证 |
| ZIP 部分失败不可观测 | 无法发现批量下载的失败率 | 已补 `s3c_zip_partial_failures_total` / `s3c_zip_failed_keys_total` / `s3c_zip_failed_total` 与服务端日志 |
| 伪造 XFF 绕过限速 | 单机可无限请求 | `S3C_TRUSTED_PROXIES` 白名单外一律用 `RemoteAddr`；默认空即不信任 XFF |

---

## 六、维护约定

1. **单一来源**：本文件只维护**版本级规划与优先级**，且**只列未完成项**；新增 / 关闭具体条目时，同步更新
   [`docs/todolist.md`](todolist.md)（唯一待办来源），完成后归档至 [`docs/features.md`](features.md)
   并**从本文件移除该条目**（必要时重编号）。
2. **评估驱动**：每次五维度评估（见 [`docs/assessment.md`](assessment.md)）产出后，按 P0/P1/P2 回写
   本路线图与 todolist，形成「评估 → 修复 → 再评估」闭环。
3. **状态真实性**：标 ⏳ 必须是工作区/分支已有代码变更，并在合并后改为 ✅（或按第 1 条移除）；禁止保留
   已完成条目的 ✅ 行——历史证据归 `features.md`。所有门禁数字必须来自实跑。
4. **发版触发**：里程碑验收标准全部满足后，由 `scripts/release-version.sh` 同步版本号并更新
   [`CHANGELOG.md`](../CHANGELOG.md)，再打 tag。
5. **文档同步**：修复 bug 或新增功能完成后必须更新相关文档（对照表见
   [`docs/development.md`](development.md) §4 与 [`AGENTS.md`](../AGENTS.md)）；本路线图的版本级条目随改动同步。
