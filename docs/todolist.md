# s3clinet 待办清单（To-do List）

> 本文件是后续迭代的**单一待办来源**，只收录**尚未完成**的事项。
> 「已完成 / 已评估」记录见 [`features.md`](features.md)；发版历史见 [`CHANGELOG.md`](../CHANGELOG.md)。
> 最近一轮综合评估（2026-09-16，五维度：代码质量 / 漏洞 / 死代码 / 降级 / 自我迭代）见
> [`assessment.md`](assessment.md)。
>
> 状态图例：⬜ 待办 · ⏳ 已排期 / 进行中（仅列剩余工作） · ➖ 已决策（不做 / 维持现状）
>
> **编号约定**：编号保持稳定、不因条目移除而重排——`apps/server/` 代码注释与历史提交仍以
> `todolist #N` 引用本清单，重排会使这些引用失真。已移除的已完成编号：
> #9 / #10 / #13 / #14 / #19 / #20 / #24，修复证据统一归档至 [`features.md`](features.md)。
>
> 最后更新：2026-09-17

## 目录

- [一、功能 / 架构待办](#一功能--架构待办)
- [二、API / 契约待办](#二api--契约待办)
- [三、代码质量 / 死代码待办](#三代码质量--死代码待办)
- [四、安全 / 供应链待办](#四安全--供应链待办)
- [五、可靠性 / 可观测性待办](#五可靠性--可观测性待办)

---

## 一、功能 / 架构待办

> 当前无待办事项。

| # | 项 | 来源 | 状态 | 说明 |
|---|----|------|------|------|
| — | （空） | — | — | 无待办事项 |

## 二、API / 契约待办

| # | 项 | 来源 | 状态 | 说明 |
|---|----|------|------|------|
| 8 | docs/api.md 无自动化校验（OpenAPI 契约测试只校验 routes↔注册表） | ASSESSMENT I2 | ⬜ | CI 加文档-路由 diff 检查或文档生成 |

## 三、代码质量 / 死代码待办

> 覆盖率 100% 反而掩盖了死代码（测试引用使死符号「活着」、i18n 被排除统计）。

| # | 项 | 来源 | 状态 | 说明 |
|---|----|------|------|------|
| 11 | 前端约 40 个 i18n 死键（`common.back/confirm/...`、`toolbar.*`、`trash.empty` 等）；`ObjectList` grid 视图无窗口化 + `visibleCount` prop 未用；`entries`/`visibleEntries` 重复排序 | ASSESSMENT D7-D9 | ⏳ | **已完成（2026-09-17）**：① 死键实为 **26 个**（非「约 40」），中英各 26 行已删，字典 695 → 669，并在 `i18n/coverage.test.ts` 增加**反向门禁**（字典中既无字面量引用、也不匹配动态拼接模式的键会让测试变红），`storage.class.*` / `provider.*` 动态键按模板形状豁免；② `visibleCount` prop 已从 `ObjectList.vue`、父组件绑定与测试 fixture 中移除（`ObjectToolbar` 的同名 prop 保留，它真实用于计数展示）；③ `@vitest/coverage-v8` 冗余依赖已删。**剩余**：grid 视图窗口化（`v-for` 仍全量渲染）、`entries`/`visibleEntries` 重复排序 |
| 12 | 覆盖率 100% 门禁的「注水」：前端排除 `i18n/**` 统计 | ASSESSMENT I1 | ⏳ | 后端已闭环（2026-09：不可达分支删除死代码而非写 gap 测试；门禁改为精确检查 profile 中 `count==0` 的语句块，不再用四舍五入的 total 百分比）；剩前端：纳入 i18n 或调整门禁至 95% + 聚焦行为测试 |

## 四、安全 / 供应链待办

| # | 项 | 来源 | 状态 | 说明 |
|---|----|------|------|------|
| 16 | SQLite 驱动 `secret_key` 明文落盘且 compose 默认即该驱动；Argon2 参数偏弱（t=1）；`S3C_STORE_KEY` 无最短长度校验 | ASSESSMENT M1/M2 / 安全审计 | ⬜ | compose 默认改 encrypted + 磁盘级加密文档；Argon2 t≥2；StoreKey 加长度校验 |
| 17 | 无安全审计日志（401、账号 CRUD、策略/删除变更）；XFF 完全信任可绕过限速 | ASSESSMENT M3/M5 / 安全审计 | ⏳ | 安全事件日志（401 / 账号 CRUD / 策略与删除变更）；XFF 仅信任已知代理 |
| 18 | `/api/health` 暴露 version | ASSESSMENT L3 / 安全审计 | ➖ | 维持现状：运维定位版本需要，且该端点通常在内网 / 鉴权后；移除属行为变更 |

## 五、可靠性 / 可观测性待办

| # | 项 | 来源 | 状态 | 说明 |
|---|----|------|------|------|
| 21 | 指标不足：缺 S3 上游延迟/错误分类/存储状态/流字节数 | ASSESSMENT S3 / SRE 审查 | ⏳ | `s3wrap` 层调用耗时直方图 / 错误按码分类 / 流字节数 metric |
| 22 | 前端对后端不可用恢复弱（仅挂载 load 一次，无健康轮询/自动重试）；`useBucketSetting.reload()` 无竞态守卫 | ASSESSMENT S5/S8 / SRE+前端审查 | ⬜ | 健康轮询 + 自动恢复；reload 加 seq/AbortController |
| 23 | 错误消息回显用户输入 | ASSESSMENT L2 / 后端审查 | ⏳ | `headers.go:35` 把 `ValidateUserMetadata` 的 `err.Error()` 回传客户端，消息含用户提交的 metadata key；应改固定文案 + 服务端日志 |
