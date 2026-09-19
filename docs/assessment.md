# s3clinet 项目综合评估报告（2026-09-16）

> 本报告从**代码质量、代码漏洞、死代码、服务降级、自我迭代**五个维度对 s3clinet 进行整体评估。
> 评估方法：4 个并行深度审查（后端 Go / 前端 Vue-TS / 安全威胁与依赖 / SRE 可靠性）× 全部质量门禁实跑
> （`go vet`、`go test -race`、`pnpm lint`、`vue-tsc`、覆盖率）× 关键发现人工复核。
> 评估结论已回写至 [`todolist.md`](todolist.md) 作为后续迭代的待办来源。
>
> **后续处置（不改写本报告的历史结论）**：2026-09-17 完成 P0/P1/P2 与路线图 v1.0.0–v1.1.0 收口
> （证据见 [`features.md`](features.md) §H–§M）；2026-09-19 完成风险登记集中处置 R1/R2/R4/R6/R7/R8
> （证据见 [`features.md`](features.md) §O），其中 R1 的字段级门禁又发现并修复了 mkdir / copy-objects /
> `DELETE /version` 三处 OpenAPI 注册表失真。

---

## 〇、执行摘要

**总体评分：78 / 100（良好，可发布但需修复若干发布前缺陷）**

s3clinet 是一个工程质量**显著高于平均水平**的项目：所有声明的质量门禁均**属实**（go vet 干净、8 包测试通过、前端 935 测试全绿、覆盖率 100%、eslint 0 违规、vue-tsc 通过、`-race` 全绿），无 XSS / 内存泄漏 / 竞态三大硬伤，安全基础（鉴权 / CSRF / SSRF / 错误脱敏）设计严谨，自我迭代闭环（评估→修复→再评估台账）完整。

主要短板集中在四类：**OpenAPI 契约与真实 handler 不一致（SSOT 失真）**、**异步任务无可观测性且丢失后无恢复**、**前端 3 个发布前缺陷（token 明文落 localStorage / loadAll 静默漏载 / SSE 终态卡死）**、**供应链与运行时漏洞（Go 1.26.6 修复线、幽灵 SHA、SQLite 明文密钥）**。

| 维度 | 评分 | 一句话结论 |
|---|---|---|
| 代码质量 | 82/100 | 分层清晰、命名规范、错误处理严谨；但存在死代码与重复实现 |
| 代码漏洞 | 72/100 | 无 Critical 可利用漏洞；SSRF/CSRF/鉴权防护到位；但 OpenAPI 契约失真 + 6 个可达 stdlib CVE |
| 死代码 | 70/100 | 后端 5 处 + 前端约 40 个 i18n 死键，覆盖率 100% 掩盖了死代码 |
| 服务降级 | 74/100 | 流式/并发/上限设计良好；但异步任务丢失无恢复、指标盲区、前端无自动恢复 |
| 自我迭代 | 90/100 | FEATURES 58 项台账、todolist 单一待办源、CI 门禁完备——行业标杆 |

---

## 一、代码质量（82/100）

### 1.1 独立验证（声明全部属实）

| 门禁 | 结果 |
|---|---|
| `go vet ./...` | ✅ 0 告警 |
| `go test -race -count=1 ./...` | ✅ 8/8 包通过（含 147s handler 并发套件） |
| 后端覆盖率 | ✅ 8 包均 100.0% statements（真实，非造假，通过可编程假 S3 httptest server 达成） |
| `pnpm lint` | ✅ 0 error 0 warning |
| `vue-tsc --noEmit` | ✅ exit 0（strict + noUnusedLocals/Parameters 全开） |
| `pnpm test` | ✅ 61 文件 / 935 测试全绿 |
| 前端覆盖率 | ✅ statements/branches/functions/lines 均 100% |

### 1.2 优点（已验证）

- **架构分层清晰**：`store → model → s3wrap → handler → openapi` 单向依赖，AWS 类型不外泄。
- **错误处理严谨**：`UserMessage` 防腐层 + sentinel（`ErrObjectTooLarge`/`ErrSourceDeleteFailed`）+ `errors.Is`，错误文案不依赖 SDK。
- **竞态处理教科书级**：前端 `loadSeq` + `AbortController` 双保险；SSE 订阅在卸载路径全部断开。
- **文件规模健康**：最大 api.ts 694 行、handler.go 拆分后 131 行，均未触 1000 行红线。
- **测试覆盖真实到位**：82 个 useObjectBrowser 测试覆盖竞态/KeepAlive/卸载清理——正是易漏测的点。

### 1.3 需改进（详见 §三）

- 覆盖率 100% 有「注水」：前端 `vite.config.ts` 排除 `i18n/**`（1432 行）统计；后端有大量 `gap_cover_test.go` 为打满分支而写，行为验证价值有限且与实现强耦合。
- 代码重复：endpoint 归一化三份实现（行为不一致）、`timeOrZero`/`deref*` 跨包重复、三处相似批量 JSON 组装。

---

## 二、代码漏洞（72/100）

### 2.1 已缓解（做得好的部分，验证无绕道）

| 威胁 | 缓解措施 | 证据 |
|---|---|---|
| 鉴权仿冒 | Bearer 常量时间比较（sha256+subtle）、scheme 大小写不敏感、多 token 轮换、最短 16 字符 | `middleware.go:228-243`、`config.go:180` |
| CSRF | CORS 非白名单 Origin 直接 403 + 强制 `application/json` 使跨域变更必预检 | `middleware.go:105-172`、`handler.go:161-177` |
| SSRF | 创建时拦 IMDS/链路本地 + **拨号期二次解析逐 IP 校验（消除 DNS 重绑定 TOCTOU）** + 禁重定向 + 禁代理 | `ssrf.go:21-50,82-107`、`client.go:144-151` |
| 敏感数据 | AccountView 不含 secretKey、错误脱敏、S3C2 加密 + Argon2id + 原子写 + 0600 | `model/account.go:43-78`、`store/crypto.go`、`atomic.go` |
| XSS | 全库 0 处 `v-html`/`innerHTML`/`eval`；预览全走服务端代理 + sandbox + 类型白名单 | `apps/web/src` 全量 grep |
| 输入防护 | 路径遍历/控制字符拒绝、zip-slip 消毒、批量 key 上限（1000/10000）、8MB body cap | `proxy.go:36-41`、`zip.go:149-168`、`handler.go:116` |

### 2.2 待修复（按严重度）

#### 🔴 High

| # | 问题 | 证据 | 影响 |
|---|---|---|---|
| **H1** | **OpenAPI 契约与真实 handler 大面积不一致（SSOT 失真）**：`/api/migrate` 文档字段 `srcAccountId/srcBucket/...` vs 真实 `sourceAccountId/sourceBucket/sourceKeys/...`（字段名全对不上）；presign 声明 `GET/PUT/DELETE/HEAD` vs 实际仅 `get\|put\|post`；delete 声明 `versionId` 但 handler 无此字段；`deleteMarkerId` vs `versionId` | `openapi_register_migrate.go:11-19` vs `migrate_exec.go:5-12`、`openapi_register_objects.go:86` vs `objects.go:401-456` | 客户端按文档调用必 400；OpenAPI 作为 SSOT 不可信。抽查 5 处 100% 命中 |
| **H2** | **Go 1.26.5 存在 6 个可达的 stdlib 漏洞**（修复版 1.26.6）：net/url 二次方复杂度、crypto/tls 握手 DoS、net/http HTTP/2 探测、encoding/xml 递归、encoding/asn1 递归、net/http Punycode | `go.mod:3`、`Dockerfile:25`；`govulncheck` 实跑 | 均有明确调用链（`s3wrap/client.go:129` Do→URL.Parse 等） |
| **H3** | **e2e-playwright.yml 的 pnpm/action-setup 幽灵 SHA**：`b906affcce14559ad1aafd4ab0d3e1d3ed4c0f0` 在 GitHub 上不存在（API 422），与其它 3 处正确 SHA `...0e942779e9f58b1` 仅差 8 位 | `.github/workflows/e2e-playwright.yml:35` | 供应链风险：工作流会失败或（若 fork 伪造 tag）被恶意 action 执行 |
| **H4** | **前端 Token 明文双份落 localStorage**：`ServerProfile.token` 随 `s3c.servers` 整体明文写入 localStorage，旁路了代码自己声明的「token 仅 sessionStorage」安全策略 | `apps/web/src/api.ts:143-151,213-244` | 任何 XSS 或同源恶意脚本可持久窃取全部后端 token |

#### 🟠 Medium

| # | 问题 | 证据 |
|---|---|---|
| M1 | SQLite 驱动 `secret_key` **明文落盘**，且 compose 默认驱动即 sqlite + `S3C_STORE_KEY` 空 | `store/sqlite.go:33,85`、`docker-compose.yml:33-34` |
| M2 | Argon2 参数偏弱（t=1/m=64MB，OWASP 建议 t≥2）；`S3C_STORE_KEY` 无最短长度校验 | `store/crypto.go:19-25`、`config.go` |
| M3 | 无安全审计日志（401、账号 CRUD、策略/删除变更均无事件日志） | `middleware.go:53-71` 仅通用 access log |
| M4 | `clientIP` 完全信任 `X-Forwarded-For` 首段，直连部署可伪造 XFF 绕过限速 | `ratelimit.go:66-80` |
| M5 | JobRegistry 无总 job 数上限，认证用户可无限开 async job 消耗内存 | `service/job.go` |
| M6 | TLS 前置无 HSTS / Permissions-Policy | `deploy/nginx/conf.d/s3clinet-tls.example.conf:27-31` |
| M7 | CI 未跑 `govulncheck`（仅 Trivy 扫容器 OS/库），stdlib 漏洞不拦截 | `ci.yml:142-149` |
| M8 | 前端 `loadAll()` 在 `nextToken` 为空时 while 一次不执行却 toast「已加载全部」——**实际只加载第一页 100 条**（数据完整性） | `useObjectBrowser.ts:393-413` |

#### 🟡 Low

| # | 问题 | 证据 |
|---|---|---|
| L1 | 预签名错误被吞（`u, _ :=`），签名失败返回空 url 的 200 | `objects.go:441-453`、`multipart.go:80` |
| L2 | 错误消息回显用户输入（`"unsupported acl: "+req.ACL`） | `metadata.go:74`、`handler.go:121` |
| L3 | `/api/health` 暴露 version（辅助漏洞匹配）；metrics 暴露时无鉴权 | `health.go:13-23`、`metrics.go:40` |
| L4 | 3 个 i18n 键被引用但未定义（`objects.toastCopyFailed`/`batchEdit.tagsNeedKey`/`common.working`），用户可见原始 key | `useObjectActions.ts:393`、`BatchMetadataDialog.vue:90,224` |

### 2.3 依赖审计结果

| 生态 | 结果 |
|---|---|
| Go（govulncheck 实跑） | 6 个可达 stdlib 漏洞（H2）；22 个 module 漏洞全在未导入的 `x/crypto/ssh` 与 `x/sys/windows`，不可达（Low）；modernc.org/sqlite、aws-sdk-go-v2 无告警 |
| npm（pnpm audit 实跑） | ✅ **0 已知漏洞**；生产依赖仅 vue；`allowBuilds` 只放行 esbuild（良好实践） |
| Rust（Cargo.toml 人工） | tauri 2.11.5 / wry 0.55.1 较新，无直接适用 RUSTSEC；无 IPC、面小；cargo-audit 未装（工具限制） |
| 供应链 | actions 全部 pin SHA（除 H3 幽灵 SHA）；dependabot 覆盖 5 个生态；`.trivyignore` 空清单合理 |

---

## 三、死代码（70/100）

### 3.1 后端（已验证）

| # | 死代码 | 证据 | 建议 |
|---|---|---|---|
| D1 | ~~`ctxReader`（与 `ctxCancelReader` 重复，仅测试引用）~~ ✅ 已删除 | `service/zip.go` + `gaps_test.go` | ~~删除~~ 已删 |
| D2 | ~~`batchItemError`（生产零引用）~~ ✅ 已删除 | `handler/s3_errors.go` | ~~删除~~ 已删（该格式串实际在 `service/batch.go` 内联，无需并入） |
| D3 | ~~`Client.S3()`（导出但生产零调用，仅测试白盒）~~ ✅ 已删除 | `s3wrap/client.go` | ~~标注或删除~~ 已删；E2E 清理改用已有 `DeleteObjectVersion`/`DeleteObject` |
| D4 | ~~`isNoSuchBucketSetting`（纯转发别名）~~ ❌ **误判：实为 5 处生产调用的在用代码**（`bucket_settings.go:34,108,179,253,328`） | `bucket_settings.go:11-13` | ~~内联~~ 保留（内联会重复 5 次 `s3wrap.HasErrorCode` 调用，反而降低可读性） |
| D5 | ~~endpoint 归一化三份实现（行为不一致）~~ ✅ 已合并为单一 `s3wrap.NormalizeEndpoint`，并修掉大小写 scheme / 首尾空白导致的 URL 损坏 | `s3wrap/client.go`、`service/migrate.go`、`s3wrap/presign.go` | ~~合并~~ 已合并（`service` 与 `PublicURL` 均改为复用；`ValidateEndpoint` 一并改为归一化后解析） |
| D6 | ~~`timeOrZero`/`derefString`/`derefInt64`/`boolOrFalse` 跨包重复~~ ✅ handler 侧 4 个包装为纯死代码，已删；`s3wrap` 内的同名 helper 有 29/1/5/2/4 处生产调用，保留 | `handler.go`（已删）、`s3wrap/helpers.go`、`s3wrap_dto.go` | ~~去重~~ 删死代码，保留在用 helper |

### 3.2 前端（已验证）

| # | 死代码 | 证据 |
|---|---|---|
| D7 | ~~约 40 个 i18n 键「定义但从未引用」~~ ✅ **已清理**：实为 **26 个**（中英各 26 行），字典 695 → 669；并在 `i18n/coverage.test.ts` 增加**反向门禁**（既无字面量引用、也不匹配动态拼接模式的键会让测试变红），`storage.class.*` / `provider.*` 动态模板键按「字面命名空间前缀 + 插值」形状豁免 | `i18n/messages/*.ts` 全量扫描 |
| D8 | ~~`ObjectList.vue` 的 `visibleCount` prop 声明但模板不使用~~ ✅ **已删除**（连同父组件绑定与测试 fixture；`ObjectToolbar` 同名 prop 保留，真实用于计数展示）。**剩余**：grid 视图 `v-for` 全量渲染未窗口化（list 有虚拟滚动） | `ObjectList.vue` |
| D9 | `entries` 与 `visibleEntries` 重复排序（O(n log n)×2） | `useObjectBrowser.ts:81-115` |
| D10 | ~~`@vitest/coverage-istanbul` 与 `coverage-v8` 双装冗余~~ ✅ **已删除** `coverage-v8`（provider 为 istanbul，v8 从未启用），锁文件已重算 | `apps/web/package.json` |

### 3.3 结论

覆盖率 100% 反而**掩盖**了死代码：D1/D2/D3 因测试引用而「活着」，D7 因 i18n 被排除统计而漏网。死代码清理建议与覆盖率门禁调整（前端纳入 i18n、后端减少 gap 测试）一起做。

---

## 四、服务降级（74/100）

### 4.1 做得好的部分（已验证）

- **流式资源管理**：`streamSlots` 全局 32 并发 + 滚动空闲写超时 5min（优于绝对截止）；`partBufPool` 64MB 有界复用；zip 4 worker 有界并发 + 无缓冲结果通道。
- **批量上限完备**：listAll/indexDst/deletePrefix/copyPrefix 全部 100k 硬上限；maxZipKeys=1000、maxDeleteKeys=1000、maxBatchKeys=10000。
- **优雅关闭**：SIGTERM → 先 cancel 异步任务再 `srv.Shutdown`（超时 `S3C_SHUTDOWN_TIMEOUT`）；nginx `SIGQUIT` 零停机 reload。
- **存储决策合理**：store 不可用 = 硬失败 + health 503（降级只读会导致写丢失，决策正确但未文档化，见 ADR 建议）。
- **部署可靠性**：非 root、HEALTHCHECK、内存上限（nginx 128M）、`proxy_buffering off` + 3600s 超时。

### 4.2 需改进（按严重度）

| # | 问题 | 证据 | 影响 |
|---|---|---|---|
| **S1** | **异步任务丢失恢复缺失**：JobRegistry 纯内存、无持久化、无重启恢复；「复制→删源」两阶段可能半途中断且无对账 | `service/job.go:51-56,117-130`、`handler.go:77-81` | 进程重启后进行中的迁移/删除**永久丢失且无痕迹**，可能产生「复制完成但源未删」的中间态 |
| **S2** | **流式传输错误被静默吞**：`copyStream` 忽略 `io.Copy` 返回值，大文件下载中断无错误日志/指标 | `stream.go:43-47`、`proxy.go:115`、`zip.go:50` | 可观测性黑洞：客户端断连后日志仍显示 200 |
| S3 | **指标严重不足**：只有 HTTP 计数/uptime/goroutine/内存/build_info；缺 S3 上游延迟、错误率按码分类、存储状态、流字节数 | `metrics.go:41-65` | S3 上游挂掉只能靠 5xx 间接推断 |
| S4 | ~~compose 未启用结构化日志（`S3C_LOG_JSON=1` 被注释）~~ ✅ 已启用 | `docker-compose.yml`、`docker-compose.prod.yml`、`.env.example` | ~~容器日志难采集检索~~ 已注入 `S3C_LOG_JSON: "${S3C_LOG_JSON:-1}"` |
| S5 | 前端对后端不可用恢复弱：仅挂载时 load 一次，无健康轮询/自动重试 | `App.vue:85-109,166-169` | 后端恢复后需手动刷新 |
| S6 | ZIP 部分失败无服务端日志，handler 忽略 failKeys/err | `service/zip.go:141-145` | 无法观测 zip 部分失败率 |
| S7 | 批量操作 SSE 终态检测缺陷：流以 EOF 结束时 Promise 悬挂、`opsBusy` 永不复位、按钮永久禁用（MigratePanel 用另一套正确实现，属复制粘贴分叉） | `useObjectActions.ts:451-474`、`DestDialog.vue:77-99`、`api.ts:630-643` | 复制/移动/删除文件夹操作可永久卡死 |
| S8 | `useBucketSetting.reload()` 无竞态守卫（切桶旧响应可覆盖新状态） | `useBucketSetting.ts:22-39` | 快速切桶时显示错误桶配置 |

### 4.3 Top 5 可靠性改进

1. 异步任务持久化 + 重启恢复（S1，数据一致性）
2. S3 上游指标：延迟直方图/错误分类/字节数（S3，最大可观测性盲区）
3. 流式错误日志 + 中断指标（S2）
4. 前端健康轮询自动恢复 + compose 默认 `S3C_LOG_JSON=1`（S5/S4）
5. SSE 终态检测统一为 MigratePanel 的共享实现（S7）

---

## 五、自我迭代（90/100）

### 5.1 做得好的部分（已验证，行业标杆）

- **评估→修复→再评估闭环**：`docs/features.md` 有完整 58 项评估台账（C 段）、v1.0.0-rc1 评估 21 项（D 段）、Optional/Nit 长尾（E 段）——每次评估结果都归档并回写。
- **单一待办来源**：`docs/todolist.md` 汇总散落待办，本报告发现将回写至此。
- **CI 门禁完备**：gofmt/vet/golangci-lint/race/覆盖率（评估时后端 90%、前端 100%；2026-09-17 起后端门禁亦提升至 100%）/Docker 构建/Trivy/E2E×2/桌面 cargo check，actions 全部 pin SHA（除 H3）。
- **版本发布流程**：`scripts/release-version.sh` 同步 8 处版本号；CHANGELOG 逐版本记录（Keep a Changelog 规范）。
- **提交规范**：81/96 提交符合 Conventional Commits；pre-commit hook 覆盖 gofmt/vet/typecheck。
- **文档-代码同步抽查**：docs/api.md 与 routes.go 抽查 6 端点全一致。

### 5.2 需改进

| # | 问题 | 建议 |
|---|---|---|
| I1 | 覆盖率 100% 门禁边际收益低、维护成本高（gap 测试、i18n 排除） | 前端纳入 i18n 或调整门禁到 95% + 聚焦行为测试；后端减少「为打满分支而写」的测试 |
| I2 | docs/api.md 无自动化校验（OpenAPI 契约测试只校验 routes↔注册表，不校验手写文档） | CI 加文档-路由 diff 检查或文档生成 |
| I3 | 无 ADR（架构决策如「存储不降级」「SSRF 私网放行」「无 IPC 桌面」均无书面理由） | 补 `docs/decisions/`，见本次文档重构 |
| I4 | e2e-playwright.yml 幽灵 SHA 说明「更新未同步」 | 核对所有 workflow 的 action SHA 一致性 |

---

## 六、优先修复清单（Roadmap）

### P0（发布前必修）
1. **H1** OpenAPI 契约对齐真实 handler（migrate 字段名、presign method 枚举、delete/restore 字段）
2. **H4** 前端 token 明文移出 localStorage.servers
3. **M8/C2** `loadAll()` 至少执行一次请求（do-while）
4. **S7/C3** SSE 终态检测统一为共享实现（消除 Promise 悬挂）

### P1（尽快）
5. **H2** Go 1.26.5 → 1.26.6（go.mod + Dockerfile + CI）
6. **H3** 修正 e2e-playwright.yml 幽灵 SHA
7. **S1** 异步任务持久化 + 重启恢复
8. **L4/R1** 补齐 3 个缺失 i18n 键

### P2（规划）
9. **M1/M2** SQLite 驱动密钥加密 + Argon2 参数加强 + S3C_STORE_KEY 长度校验
10. **M3/M5** 安全审计日志 + JobRegistry 上限
11. **S2/S3** 流式错误日志 + S3 上游指标
12. **D1-D10** 死代码清理（后端 6 处 + 前端 i18n 死键 + grid 窗口化）——**进度（2026-09-17）**：D1-D6 已闭环；D7 死键实为 26 个，已删并加反向门禁；D8 的 `visibleCount` prop 已删、grid 窗口化仍待做；D10 `coverage-v8` 冗余依赖已删。剩 D9 重复排序与 D8 窗口化
13. **M6** TLS 前置补 HSTS
14. **I2** docs/api.md 自动化校验

---

## 附：评估方法

- **深度审查子代理**：后端 Go（27 生产文件 + 78 测试文件）、前端 Vue/TS（apps/web/src 全量）、安全威胁模型（STRIDE × 5 边界 + govulncheck/pnpm audit 实跑）、SRE 可靠性（降级/可观测性/CI）。
- **人工复核**：所有 Critical/High 发现均经 `grep`/`sed`/`git` 独立验证（含 H1 字段对比、H3 GitHub API 422、C1 localStorage 写入路径、M8 loadAll 循环逻辑、R1 i18n 键缺失）。
- **门禁实跑**：`go vet`、`go test -race -count=1`、`pnpm lint`、`vue-tsc`、覆盖率。
- **限制**：cargo-audit 未安装无法在线审计 Rust 依赖；部分性能评估基于静态分析未做基准测试。
