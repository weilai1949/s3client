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
> #8 / #9 / #10 / #11 / #12 / #13 / #14 / #16 / #17 / #19 / #20 / #21 / #22 / #23 / #24，
> 修复证据统一归档至 [`features.md`](features.md)。
>
> **2026-09-19 补登记**：#26–#37 来自 [`review-2026-09-19.md`](review-2026-09-19.md) 的 P2（§9.3），
> 此前该轮审查的开放发现**未登记入本清单**（本文件当时写着「无待办」），违反「本文件是唯一待办来源」
> 的约定——本次一并补登记，恢复 SSOT。

> 最后更新：2026-09-19

## 目录

- [一、功能 / 架构待办](#一功能--架构待办)
- [二、API / 契约待办](#二api--契约待办)
- [三、代码质量 / 死代码待办](#三代码质量--死代码待办)
- [四、安全 / 供应链待办](#四安全--供应链待办)
- [五、可靠性 / 可观测性待办](#五可靠性--可观测性待办)
- [六、文档失真待办](#六文档失真待办)

---

## 一、功能 / 架构待办

> #25 为**已正式立项、暂不处理**的条目：桌面端分发与签名受外部凭证阻塞（roadmap §5.2 E6）。

| # | 项 | 来源 | 状态 | 说明 |
|---|----|------|------|------|
| 25 | 桌面端分发与签名 | ROADMAP §三 #1 / 长期 | ⬜ | **已立项，暂不处理**：Windows 代码签名证书 / Apple Developer ID + 公证是阻塞依赖（roadmap E6），未获取前产物被 SmartScreen / Gatekeeper 拦截且无自动更新通道（roadmap R5）；实现入口 `.github/workflows/release-desktop.yml` |

## 二、API / 契约待办

> 请求体/响应契约门禁已于 2026-09-19 补齐（`api_doc_test.go` 双向、`openapi_request_fields_test.go`
> 全量遍历、`openapi_response_contract_test.go` 响应级）；#8（`docs/api.md` 无自动化校验）早前闭环。
> 残留的是**注册表未声明的 query 参数**与**类型/required 语义无门禁**。

| # | 项 | 来源 | 状态 | 说明 |
|---|----|------|------|------|
| 26 | 注册表漏声明 4 个 query 参数 | review §7.3 D7 | ⬜ | `head.versionId`、`proxy.maxBytes`、`trash.prefix`（文档有）、`objects.startAfter`（文档也没有——未公开的公开特性）；按 OpenAPI 生成的客户端不会发送这些参数 |
| 27 | 类型 / required / 枚举语义无门禁 | review §7.4 | ⬜ | 现有门禁只比对字段名集合；presign 的 method 枚举有逐条断言，其余端点无 |

## 三、代码质量 / 死代码待办

> #11（i18n 死键 / grid 无窗口化 / 重复排序）与 #12（覆盖率门禁注水）均已闭环。
> 新增门禁的**断言范围**已在文件头写明（见 #28）。

| # | 项 | 来源 | 状态 | 说明 |
|---|----|------|------|------|
| 28 | 响应门禁残留范围（端点内联 / `map[string]any` 响应） | review §4.3 / §7.4 | ⬜ | `openapi_response_contract_test.go` 只覆盖 `components.schemas` 的共享 schema；各端点内联或手工拼装的响应无机械门禁（范围已写入文件头注释） |

## 四、安全 / 供应链待办

> #16（SQLite 密钥明文 / Argon2 t=1 / StoreKey 无长度校验）与 #17（无审计日志 / XFF 全信任）
> 均已闭环；#18 为已决策的维持现状项。S1（alpine EOL）/ S2（占位 token）已于 2026-09-19 闭环。

| # | 项 | 来源 | 状态 | 说明 |
|---|----|------|------|------|
| 18 | `/api/health` 暴露 version | ASSESSMENT L3 / 安全审计 | ➖ | 维持现状：运维定位版本需要，且该端点通常在内网 / 鉴权后；移除属行为变更 |
| 29 | base compose 明文落盘 | review §5.2 S3 | ⬜ | 默认 `sqlite` + 空 `S3C_STORE_KEY` → secretKey 明文落盘，仅启动 WARN；`prod.yml` 才强制 |
| 30 | `/api/metrics` 配了 token 仍免鉴权（文档未明说） | review §5.2 S4 | ⬜ | 设计如此，但需在文档中显式声明「即使配了 `S3C_TOKEN` 也不受保护」 |
| 31 | `S3C_STORE_KEY` 无非默认值校验 | review §5.2 S6 | ⬜ | 明文落盘只降级为 WARN |
| 32 | Trivy 固定在 0.58.1（落后 16 个小版本，`--vuln-type` 已 deprecation） | review §8.1 R5 | ⬜ | 升级并核验官方镜像 digest |
| 33 | Rust 工具链未 pin；desktop 缺 `packageManager`；Node 只 pin 大版本 | review §8.1 R6 | ⬜ | `dtolnay/rust-toolchain@…# stable` + `rust:1-bookworm`，无 `rust-toolchain.toml` |
| 34 | `.dockerignore` 漏覆盖率/缓存目录（~5.4 MB） | review §8.1 R8 | ⬜ | 漏 `apps/web/coverage`、`.pnpm-store`、`test-results`、`.run/`、`.cargo/` → 同一 commit 因本地是否跑过覆盖率而产出不同镜像层 |

## 五、可靠性 / 可观测性待办

> #21（缺 S3 上游指标）与 #22（前端恢复弱 / reload 无竞态守卫）均已闭环；
> #23（错误消息回显用户输入）已闭环并有源码级门禁防复发。

| # | 项 | 来源 | 状态 | 说明 |
|---|----|------|------|------|
| 35 | Prometheus 直方图语义违规 | review §7.3 D1 | ⬜ | `s3wrap/metrics.go` 注释称「累积」但实现每次只 +1 个桶，`handler/metrics.go` 却按 `_bucket{le=…}` 输出 → `le` 非单调、`+Inf ≠ _count`，`histogram_quantile()` 全错。要么改真累积，要么改语义（同步注释与测试） |
| 36 | `/api/openapi.json` 每次请求全量 marshal | review §6.2 P3 | ⬜ | 70 个 operation 无缓存（默认 404，影响低–中） |
| 37 | 缺「真实后端 + 真实构建产物」浏览器冒烟 | review §9.3 | ⬜ | 当前 `/api/**` 全被 `page.route` mock，无前后端联调门禁 |
| 38 | 本地 `make` ≠ CI（无 lint/govulncheck 目标、`test-all` 无覆盖率门禁） | review §8.1 R7 | ⬜ | `Makefile` 缺 `lint` / `golangci-lint` / `govulncheck`；`make` 用裸 `pnpm install` 会改写锁文件；`test-cover` / `install-hooks` 不在 `.PHONY` |
| 39 | GitHub 不传 `VERSION` build-arg；无 workflow 推送镜像 | review §8.1 R9/R10 | ⬜ | GitHub 回落到过期字面量，GitLab 传 `VERSION=ci`，Makefile 传实版本；`push: false` |

## 六、文档失真待办

> 来自 [`review-2026-09-19.md`](review-2026-09-19.md) §7.3；均为「数字/承诺与代码不符」，须与文档同 PR。
> D2（注释）/ D3（发布脚本）/ D6（presign 参数）/ D8（3 路并发）/ D9（存放约定）已随 2026-09-19
> 文档同步一并修正（见 [`features.md`](features.md) §S）；余下条目见下表。

| # | 项 | 来源 | 状态 | 说明 |
|---|----|------|------|------|
| 41 | `git tag v1.0.0` 不存在但 roadmap 称已收口 | review §7.3 D5 | ⬜ | 只有 `rc0/rc1`；HEAD 已 `rc1+114`，多处版本串仍为 `1.0.0-rc1`。**需人工决策**：打 tag 或修正 roadmap 状态 |
| 45 | `features.md` 的 `failKeys ≤200` 与代码不符 | review §7.3 D4 | ⬜ | 服务端 `BatchResult.FailKeys` 不裁剪（10k 全失败约 10 MB 并落盘 `jobs.json`），只有前端裁剪 |
| 46 | `CHANGELOG.md` 的 `[Unreleased]` 内过期覆盖率计数 | review §7.3 D10 | ⬜ | 历史段落的 3883/2769/1059/3339 与「983 测试」是当时实测值；按 CHANGELOG「不追溯篡改已发布区」的纪律，需决定是否加注「当时值」而非改写 |
