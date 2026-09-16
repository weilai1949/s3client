# s3clinet 待办清单（To-do List）

> 本文件汇总散落在各评估 / 快照文档中的**待处理（pending）**事项，作为后续迭代的单一待办来源。
> 「已完成 / 已评估」记录见 [`FEATURES.md`](FEATURES.md)；发版历史见 [`CHANGELOG.md`](../CHANGELOG.md)。
> 最近一轮综合评估（2026-09-16，五维度：代码质量 / 漏洞 / 死代码 / 降级 / 自我迭代）见
> [`ASSESSMENT.md`](ASSESSMENT.md)。
>
> 状态图例：⬜ 待办 · ⏳ 已排期 / 进行中 · ✅ 已完成 · ➖ 已决策（不做 / 维持现状）
>
> 最后更新：2026-09-16

## 目录

- [一、功能 / 架构待办](#一功能--架构待办)
- [二、API / 契约待办](#二api--契约待办)
- [三、代码质量 / 死代码待办](#三代码质量--死代码待办)
- [四、安全 / 供应链待办](#四安全--供应链待办)
- [五、可靠性 / 可观测性待办](#五可靠性--可观测性待办)

---

## 一、功能 / 架构待办

> 2026-09-16 评估新增（来源：[`ASSESSMENT.md`](ASSESSMENT.md)）。

| # | 项 | 来源 | 状态 | 说明 |
|---|----|------|------|------|
| 5 | `loadAll()` 在 `nextToken` 为空时静默只加载第一页并误报「已加载全部」 | ASSESSMENT M8 / 前端审查 C2 | ⬜ | `while (nextToken.value && ...)` 改为 do-while 或本地 token 至少执行一次请求；补「nextToken 为空」路径测试 |
| 6 | 批量复制/移动/删除（文件夹）SSE 终态检测：流以 EOF 结束时 Promise 悬挂、`opsBusy` 永不复位、按钮永久禁用 | ASSESSMENT S7 / 前端审查 C3 | ⬜ | 收敛为 MigratePanel 已实现的正确版本（共享 composable），`onProgress` 同时处理流结束回读 `done:true` |

## 二、API / 契约待办

| # | 项 | 来源 | 状态 | 说明 |
|---|----|------|------|------|
| 7 | **OpenAPI 契约与真实 handler 不一致（SSOT 失真）**：`/api/migrate` 字段名 `srcAccountId/...` vs 真实 `sourceAccountId/...` 全对不上；presign 声明 `GET/PUT/DELETE/HEAD` vs 实际仅 `get\|put\|post`；delete 声明 `versionId` 但 handler 无此字段；`deleteMarkerId` vs `versionId` | ASSESSMENT H1 / 后端审查 R1 | ⬜ | 抽查 5 处 100% 命中；建议契约测试增加「注册表 schema ↔ handler 实际解析字段」三方一致性断言 |
| 8 | docs/API.md 无自动化校验（OpenAPI 契约测试只校验 routes↔注册表） | ASSESSMENT I2 | ⬜ | CI 加文档-路由 diff 检查或文档生成 |

## 三、代码质量 / 死代码待办

> 覆盖率 100% 反而掩盖了死代码（测试引用使死符号「活着」、i18n 被排除统计）。

| # | 项 | 来源 | 状态 | 说明 |
|---|----|------|------|------|
| 9 | 后端死代码：`ctxReader`（仅测试引用，与 `ctxCancelReader` 重复）、`batchItemError`（生产零引用）、`Client.S3()`（导出零生产调用）、`isNoSuchBucketSetting`（转发别名） | ASSESSMENT D1-D4 | ⬜ | 删除/内联；若 `Client.S3()` 需保留供测试则加注释 |
| 10 | endpoint 归一化三份实现且行为不一致（`s3wrap/client.go:157` vs `service/migrate.go:62`）；`timeOrZero`/`deref*` 跨包重复 | ASSESSMENT D5-D6 | ⬜ | 合并为单一 helper；`SameEndpoint` 与 `s3wrap.New` 端点判定需一致 |
| 11 | 前端约 40 个 i18n 死键（`common.back/confirm/...`、`toolbar.*`、`trash.empty` 等）；`ObjectList` grid 视图无窗口化 + `visibleCount` prop 未用；`entries`/`visibleEntries` 重复排序 | ASSESSMENT D7-D9 | ⬜ | 脚本清理死键（保留 `storage.class.*`/`provider.*` 动态键）；grid 加渲染上限或窗口化 |
| 12 | 覆盖率 100% 门禁的「注水」：前端排除 `i18n/**` 统计；后端大量 gap 测试为打满分支而写 | ASSESSMENT I1 | ⬜ | 前端纳入 i18n 或调整门禁至 95% + 聚焦行为测试；后端减少 gap 测试并补行为断言 |

## 四、安全 / 供应链待办

| # | 项 | 来源 | 状态 | 说明 |
|---|----|------|------|------|
| 13 | **Go 1.26.5 有 6 个可达 stdlib 漏洞**（修复版 1.26.6）：net/url、crypto/tls、net/http、encoding/xml、encoding/asn1、Punycode | ASSESSMENT H2 / 安全审计 | ⬜ | `go.mod:3` + `server/Dockerfile:25` + CI 同步；CI 增补 `govulncheck ./...` 门禁 |
| 14 | **e2e-playwright.yml:35 pnpm/action-setup 幽灵 SHA**（GitHub API 422 不存在，与正确 SHA 差 8 位） | ASSESSMENT H3 / 安全审计 | ⬜ | 改为与其它 workflow 一致的 `b906affcce14559ad1aafd4ab0e942779e9f58b1` |
| 15 | **前端 Token 明文双份落 `localStorage['s3c.servers']`**，旁路「token 仅 sessionStorage」策略 | ASSESSMENT H4 / 前端审查 C1 | ⬜ | `s3c.servers` 只存 `{id,name,base}`，token 单副本按服务器 id 前缀存 sessionStorage |
| 16 | SQLite 驱动 `secret_key` 明文落盘且 compose 默认即该驱动；Argon2 参数偏弱（t=1）；`S3C_STORE_KEY` 无最短长度校验 | ASSESSMENT M1/M2 / 安全审计 | ⬜ | compose 默认改 encrypted + 磁盘级加密文档；Argon2 t≥2；StoreKey 加长度校验 |
| 17 | 无安全审计日志（401、账号 CRUD、策略/删除变更）；JobRegistry 无总 job 上限；XFF 完全信任可绕过限速 | ASSESSMENT M3/M4/M5 / 安全审计 | ⬜ | 安全事件日志；JobRegistry 上限；XFF 仅信任已知代理 |
| 18 | TLS 前置无 HSTS / Permissions-Policy；`/api/health` 暴露 version | ASSESSMENT M6/L3 / 安全审计 | ⬜ | nginx TLS conf 加 `add_header`；health 考虑去 version |

## 五、可靠性 / 可观测性待办

| # | 项 | 来源 | 状态 | 说明 |
|---|----|------|------|------|
| 19 | **异步任务丢失恢复缺失**：JobRegistry 纯内存无持久化，「复制→删源」两阶段可能半途中断且无对账 | ASSESSMENT S1 / SRE 审查 | ⬜ | 任务清单落盘（复用 store 原子写）+ 启动标记 interrupted + 前端展示未完成任务 |
| 20 | 流式传输错误被静默吞（`copyStream` 忽略 `io.Copy` 返回值），大文件下载中断无日志/指标 | ASSESSMENT S2 / SRE 审查 | ⬜ | `copyStream` 返回 error 记日志 + `s3c_stream_interrupted` 计数 |
| 21 | 指标不足：缺 S3 上游延迟/错误分类/存储状态/流字节数；compose 未启用 `S3C_LOG_JSON=1` | ASSESSMENT S3/S4 / SRE 审查 | ⬜ | s3wrap 层加调用耗时/错误/字节 metric；compose 默认结构化日志 |
| 22 | 前端对后端不可用恢复弱（仅挂载 load 一次，无健康轮询/自动重试）；`useBucketSetting.reload()` 无竞态守卫 | ASSESSMENT S5/S8 / SRE+前端审查 | ⬜ | 健康轮询 + 自动恢复；reload 加 seq/AbortController |
| 23 | 预签名错误被吞（`u, _ :=`，失败返回空 url 的 200）；错误消息回显用户输入 | ASSESSMENT L1/L2 / 后端审查 | ⬜ | 检查并 500；去掉拼接回显 |
| 24 | 3 个 i18n 键被引用但未定义（`objects.toastCopyFailed`/`batchEdit.tagsNeedKey`/`common.working`） | ASSESSMENT L4 / 前端审查 R1 | ⬜ | 补齐定义；测试断言改为真实文案而非原始 key |

---

## 历史归档

> 此前 4 项（存储驱动收敛 / `secretSet` 契约 / OpenAPI `$ref` 接线 / 预览桶契约）均已完成并归档至
> [`FEATURES.md`](FEATURES.md) 的「一、产品功能」与「二、已完成修复与优化」。
