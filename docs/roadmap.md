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
- [三、v1.0.0：首个稳定版](#三v100首个稳定版)
- [四、v1.0.x：可靠性加固](#四v10x可靠性加固)
- [五、v1.1.0：体验与性能](#五v110体验与性能)
- [六、v1.2+ / 长期](#六v12--长期)
- [七、质量门禁基线](#七质量门禁基线)
- [八、风险与依赖](#八风险与依赖)
- [九、维护约定](#九维护约定)

---

## 一、当前定位

**产品形态**：S3 兼容对象存储客户端，Web 端 + Tauri 2 桌面端（B/S 架构、无 IPC、全 HTTP）；Go 后端
（AWS SDK for Go v2）+ Vue 3 / Vite / TS 前端，70 个 `/api/*` 端点、OpenAPI 3.0.3 自动生成。

**结论**：**P0 与 P1 均已清零**，`v1.0.0-rc1` 候选版已收口——4 项发布前必修缺陷（P0，提交 `e03a15e`）与
5 项稳定版门槛项（P1：Go 1.26.6 + `govulncheck`、幽灵 action SHA、OpenAPI 契约对齐、缺失 i18n 键、
异步任务持久化）全部修复并归档。因此当前第一优先级是**收口 v1.0.0**（仅剩 `docs/api.md` 漂移校验），
随后进入 `v1.0.x` 的可靠性加固：**存储加密、安全审计、可观测性**三条主线。

> P0 / P1 的逐条修复记录与验证证据见 [`docs/features.md`](features.md)「H」「I」段，
> 不在此重复。本文件只列**未完成**项。

---

## 二、里程碑总览

| 里程碑 | 主题 | 关键验收 | 依赖 |
|---|---|---|---|
| **v1.0.0-rc1**（当前） | 候选版收口：P0 四项缺陷清零 | ✅ 已收口（`e03a15e`）；门禁全绿 | — |
| **v1.0.0** | 首个稳定版：P1 清零 + 契约可信 | ✅ P1 已清零；仅剩 `docs/api.md` 漂移校验（#1） | rc1 |
| **v1.0.x** | 可靠性加固：存储 / 安全审计 / 可观测性 | 密钥落盘加密；安全事件日志；S3 上游指标与流式失败可见 | v1.0.0 |
| **v1.1.0** | 体验与性能：长列表、错误恢复、i18n | grid 窗口化；健康轮询自动恢复；门禁消除注水 | v1.0.x |
| **v1.2+** | 长期：桌面分发、可选增强 | 桌面端签名与自动更新；按需评估 | v1.1.0 |

**发布约束**：任一里程碑发布前，[质量门禁基线](#七质量门禁基线)必须全绿；P0 未清零不得进入 v1.0.0。

---

## 三、v1.0.0：首个稳定版

**目标**：关闭最后一项契约可信度缺口，使「文档即契约」不再依赖人工维护。发布后打 `v1.0.0` tag。

| # | 条目 | 来源 | 状态 | 验收标准 |
|---|---|---|---|---|
| 1 | `docs/api.md` 自动化校验 | ASSESSMENT I2 / todolist #8 | ⬜ | CI 增加文档-路由 diff 检查，或改为由 OpenAPI 生成该文档；文档与 `routes.go` 漂移即红灯 |

---

## 四、v1.0.x：可靠性加固

**目标**：解决评估中「服务降级 74 分」暴露的存储与可观测性短板，不改变对外契约。

### 存储与安全

| # | 条目 | 来源 | 状态 | 验收标准 |
|---|---|---|---|---|
| 2 | SQLite 驱动密钥加密 + Argon2 加强 | ASSESSMENT M1/M2 / todolist #16 | ⬜ | compose 默认 `encrypted` 驱动或提供磁盘加密文档；Argon2 t≥2（**须先引入带 KDF 参数的新格式版本并支持双版本读取**——现有 `S3C2` 格式不存参数，直接调参将使既有加密库不可解密）；`S3C_STORE_KEY` 最短长度校验；升级路径兼容旧库 |
| 3 | 安全审计日志 + XFF 可信代理 | ASSESSMENT M3/M5 / todolist #17 | ⏳ | 401 / 账号 CRUD / 策略与删除变更留事件日志；`clientIP` 仅信任已知代理。**已完成**：JobRegistry 总上限（256 个未终结任务，超限 503） |

### 可观测性

| # | 条目 | 来源 | 状态 | 验收标准 |
|---|---|---|---|---|
| 4 | ZIP 部分失败可见（S6） | ASSESSMENT S6 | ⬜ | ZIP 打包中单 key 失败需有服务端日志与指标（当前仅写入包内 `_下载失败清单.txt`，handler 忽略 `failKeys`，无法观测失败率）。**已完成**：`copyStream` 返回并记录 error + `s3c_stream_interrupted` 计数 |
| 5 | S3 上游指标 | ASSESSMENT S3 / todolist #21 | ⏳ | `s3wrap` 层记录调用延迟直方图 / 错误按码分类 / 流字节数。**已完成**：compose 默认 `S3C_LOG_JSON=1` 结构化日志 |
| 6 | 错误文案不回显用户输入 | ASSESSMENT L2 / todolist #23 | ⬜ | `headers.go:35` 把 `ValidateUserMetadata` 的 `err.Error()` 直接回传客户端，消息中含用户提交的 metadata key（如 `key %q length %d > %d`）；应改为固定文案 + 服务端日志。**已完成**：预签名失败返回 500（原为 `200 {"url":""}`），并有源码级门禁防复发（L1） |

---

## 五、v1.1.0：体验与性能

**目标**：前端长列表与恢复能力，以及让「100% 覆盖率」门禁回归行为价值。

| # | 条目 | 来源 | 状态 | 验收标准 |
|---|---|---|---|---|
| 7 | grid 视图窗口化 + `ObjectList` 清理 | ASSESSMENT D8 / todolist #11 | ⏳ | **已完成**：未使用的 `visibleCount` prop 已从 `ObjectList.vue`（含父组件绑定与测试 fixture）移除。**剩余**：grid 视图 `v-for` 仍全量渲染，需加渲染上限或窗口化 |
| 8 | 前端健康轮询与自动恢复 | ASSESSMENT S5/S8 / todolist #22 | ⬜ | 后端不可用后自动重试 / 恢复；`useBucketSetting.reload()` 加 seq 或 AbortController 竞态守卫 |
| 9 | 后端死代码清理（收尾） | ASSESSMENT D5/D6 / todolist #10 | ✅ | **已完成**：`ctxReader` / `batchItemError` / `Client.S3()` 已删。① endpoint 归一化已合并为唯一实现 `s3wrap.NormalizeEndpoint`（`service.SameEndpoint` 与建 client 的 BaseEndpoint 都调它；`HTTP://Host` → `http://HTTP://Host` 的损坏已修）。② 四个 helper 经复核**并非死代码**：`derefString` 29 处、`timeOrZero` 4 处、`boolOrFalse` 5 处、`derefInt64` 2 处，全部在 `s3wrap` 内有真实生产引用（`bucket.go` / `object.go` / `multipart.go` / `s3wrap_dto.go`），`handler` 侧本就零引用也无需引用；`helpers.go` 与 `s3wrap_dto.go` 承载不同职责，不构成重复。原「应删」判断有误，已更正 |
| 10 | i18n 死键清理与重复排序 | ASSESSMENT D7/D9 / todolist #11 | ⏳ | **已完成**：26 个未引用键已删（中英各 26 行，695 → 669），`storage.class.*` / `provider.*` 等动态模板键按形状豁免；`i18n/coverage.test.ts` 新增反向门禁防回潮。**剩余**：`entries` / `visibleEntries` 单次排序 |

---

## 六、v1.2+ / 长期

| # | 条目 | 来源 | 状态 | 说明 |
|---|---|---|---|---|
| 11 | 覆盖率门禁去「注水」 | ASSESSMENT I1 / todolist #12 | ⬜ | 前端纳入 `i18n/**` 统计或调整为 95% + 行为测试；后端减少为打满分支而写的 gap 测试（如 `handler` 侧仅由测试引用的 deref 包装） |
| 12 | 桌面端分发与签名 | 长期 | ⬜ | Windows / Linux / macOS 产物签名与自动更新策略（现仅打包挂 Release） |
| 13 | 增量同步与批量能力的体验增强 | FEATURES | ➖ | 现有 `etag` / `size_mtime` / `always` 三模式满足需求，按用户反馈再评估 |
| 14 | 死代码纪律 | ASSESSMENT §三 | ➖ | 维持现状：由覆盖率门禁调整（#11）一并治理，不单独立项 |

---

## 七、质量门禁基线

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
| 前端测试 | `pnpm test` | ✅ 967 例全绿（62 文件） |
| 前端覆盖率 | statements / branches / functions / lines | ✅ 100% |
| 依赖审计 | `pnpm audit` / Trivy | ✅ npm 0 漏洞；镜像 CRITICAL/HIGH 硬失败 |
| E2E | Playwright（`e2e.yml` + `e2e-playwright.yml`） | ✅ 全 action SHA 经 GitHub API 核验 |

> 后端覆盖率已补齐至**每包 100%**（2026-09 删除了确实不可达的防御分支，其余缺口改用行为断言，
> 不做 gap 测试），CI 与 `make test-cover` 以 100% 为硬阈值——直接检查 profile 中的 `count==0`
> 语句块，避免 1 位小数的百分比四舍五入掩盖回退。去注水见 #12。

---

## 八、风险与依赖

| 风险 | 影响 | 缓解 |
|---|---|---|
| OpenAPI 契约作为 SSOT 仍可能漂移 | 客户端按文档调用 400 | 契约测试已覆盖全部 `requestBody` 端点；剩余文档侧漂移由 #1 自动化校验兜底 |
| 覆盖率门禁掩盖死代码 | 维护成本高、真实覆盖失真 | 门禁调整与死代码清理同批做（#9 / #11） |
| SQLite 默认驱动密钥明文 | 落盘密钥泄露 | `docker-compose.prod.yml` 已用 `encrypted`；默认 compose 与磁盘加密文档见 #2 |
| Argon2 参数升级的兼容性 | 直接调参将使既有加密库不可解密 | #2 要求先引入带 KDF 参数的新格式版本 + 双版本读取，再调参 |
| ZIP 部分失败不可观测 | 无法发现批量下载的失败率 | 补服务端日志与指标（#4） |

---

## 九、维护约定

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
