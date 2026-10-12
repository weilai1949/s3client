# 全仓代码质量评审（2026-10-10）

> **状态**：**已归档**（2026-10-10，按 [index.md](index.md)「归档操作」四步冻结——§2 状态表的 N1–N3
> **全部同日闭环**，收口证据见 [`../FEATURES.md`](../FEATURES.md) §CK 与 [`../../CHANGELOG.md`](../../CHANGELOG.md) `[Unreleased]`）。
> 结论绑定提交 `22b79e1`。文中 `file:line` 与门禁数字均为**评审时点快照**，**不回写**。
>
> **范围**：全仓代码——`apps/server` Go 后端（302 个 `.go`；生产 16 636 行 / 测试 47 830 行，
> 10 个包）、`apps/web` Vue 3 + TS 前端（146 个 `.ts` + 40 个 `.vue`，`src/` 45 686 行）；含门禁、
> CI 与文档一致性，**不含**文档文风评审。
>
> **方法**：机械化门禁**独立复跑** + 五轴逐文件复核本批 delta（`git diff e965e54..HEAD` 的 42 个
> Go 生产文件、29 个前端生产文件与 5 份 CI / compose / 脚本）+ 关键结论回到源码逐条复核（手工推演
> cron 的春 / 秋 DST 用例、逐条核对新锁序不变式、逐个核对端点级作用域闸的**路由面**是否与其
> `HasPrefix` / `==` 判据一致）。
>
> **结论**：**Approve**（可合并）。未发现 Critical / Required 项；三道 Nit（N1–N3）按仓库口径
> 转登记 [KNOWN_ISSUES.md](../KNOWN_ISSUES.md) #84–#86，不阻断本分支。

## 1. 机械门禁实测（评审时点，全部本机独立复跑）

| 门禁 | 结果 | 说明 |
|---|---|---|
| `gofmt -l` | 0 输出 | 干净 |
| `go vet ./...` | exit 0 | 0 告警 |
| `go build ./...` | exit 0 | 干净 |
| `go test ./...`（10 包） | **10/10 PASS** | 见下方「沙箱口径」——必须在**非沙箱**下跑 |
| Go 覆盖率（`count==0` 块检查） | **100.0%，零未覆盖块** | `go tool cover -func` + `count==0` 双查 |
| `golangci-lint run ./...`（v2.13.2，与 CI 同版本） | **0 issues** | |
| `pnpm lint`（`eslint src e2e e2e-real --max-warnings 0`） | **0 告警** | |
| `pnpm test` | **83 文件 / 1292 例全过** | 上次评审时点 82 文件 / 1272 例 |
| `pnpm build`（`vue-tsc --noEmit` + `vite build`） | OK | 163 模块，`dist` 1.19 kB html + 417.73 kB js |
| `pnpm typecheck:e2e` | 未单独跑 | 已由 `pnpm lint`（含 `e2e-real`）与 CI 覆盖 |
| `S3CLIENT_E2E=1 go test ./internal/s3wrap/` | **未复跑** | 需 docker + RustFS，见 §4 |
| `make e2e-real` | **未复跑** | 同上（本评审环境无 docker / 无外网拉镜像） |
| `govulncheck` / `cargo audit` / Trivy / 两套 CI 远端 | **未复跑** | 需外网或 CI，见 §4 |

> **沙箱口径（本轮新增注意事项，供后续代理复现）**：本评审环境的沙箱**禁止 `listen()`**
> （`socket: operation not permitted`），而 `httptest.NewServer` 必须绑定回环端口——因此在沙箱内
> 直接跑 `go test ./...` 会得到**假红灯**（`internal/handler` 与 `internal/tracing` 在
> `httptest.newLocalListener()` 处 panic，报 `socket: operation not permitted`）。**这不是代码缺陷**：
> 以 `require_escalated` 在非沙箱下复跑即 10/10 全绿。`golangci-lint` 同理（沙箱内报
> `context loading failed: no go files to analyze`，非沙箱下 0 issues）。
>
> **工具链**：本机 `go version` = go1.26.9，与 `apps/server/go.mod` 的 pin 一致，故上次评审的
> 「本机工具链与 `go.mod` 不匹配、需回退版本才能跑」告警**本轮不成立**，Go 门禁数字为直接实测。

## 2. 处置状态总览

| 编号 | 级别 | 一句话 | 状态 |
|---|---|---|---|
| — | Critical / Required | 全仓五轴未发现阻断或必修项 | — |
| N1 | Nit（安全卫生） | `warnPlaintextEndpoints` 原样记录用户提交的 endpoint——若带 userinfo（`http://key:secret@host`）会把凭据写进日志 | ✅ 已修（2026-10-10 同批次 → #84 闭环，见 §5） |
| N2 | Nit（指标命名） | 新指标 `s3c_volume_inode_total` 是 **gauge**，却带 Prometheus 约定**保留给 counter** 的 `_total` 后缀 | ✅ 已修（2026-10-10 同批次 → #85 闭环：指标改名 `s3c_volume_inode_capacity`，见 §5） |
| N3 | Nit（注释卫生） | `apps/server/main.go` 的 `runHealthcheck` 文档注释有一行**重复**（非本批引入，历史遗留） | ✅ 已修（2026-10-10 同批次 → #86 闭环，见 §5） |
| O1 | 观察（非缺陷） | 新增的 `persistMu` 把「快照 + Save」串行化——正确性收益明确，代价是并发落盘排队 | ➖ 维持（注释已写明取舍） |

> 状态由**处置批次**回写（2026-10-10：N1–N3 登记即闭环并移除，编号台账见 [`../KNOWN_ISSUES.md`](../KNOWN_ISSUES.md) §四，证据见 [`../FEATURES.md`](../FEATURES.md) §CK）。

## 3. 五轴结论

本批 delta 的主体是 **2026-10-09 评审 C1–C2 / R1–R10 与 O1–O12 的闭环提交**
（`eca2342` / `a68ff0b` / `30599f0` / `dd9ab88` / `55102a9`）与随后的文档收口。五轴逐条复核如下。

### 3.1 正确性（逐条复核，未发现问题）

- **cron 的 DST 语义**（`apps/server/internal/service/schedule_cron.go`）：新实现改为「绝对时间轴构造
  候选 + 回验本地字段」。我手工推演了 `America/New_York` 两个方向——春季 `2026-03-08 02:30` 间隙：
  偏移集合 `{-18000, -14400}` 产出的两个候选本地墙钟分别为 `03:30 EDT` 与 `01:30 EST`，**均不**
  等于请求墙钟，`valid` 为空即返回最晚实例 `03:30 EDT`，与注释所称「按跳变前偏移解释、当天触发一次」
  一致；秋季 `2026-11-01 01:30` 重复小时：两个候选均回验通过，都进入 `Next` 的 `cand.After(after)`
  比较，不漏跑。跨日错位候选被 `Date()` / `Clock()` 回验剔除。结论：实现与文档一致，逻辑正确。
- **R5「旧快照覆盖新状态」**（`scheduler.go` / `job.go`）：新增 `persistMu` 后才真正把「取快照 +
  Save」变成整体。我逐条核对了**全部** `sc.persist()` / `r.persistJobs()` 调用点——均**先 `Unlock`
  再调用**，且不存在「持有 `sc.mu` / `r.mu` 时取 `persistMu`」的路径，故注释声明的锁序
  `persistMu → mu` 成立、无环、无死锁。
- **R6 前端加载竞态**（`BucketsPanel.vue` / `StorageReportPanel.vue` / `useObjectBrowser.ts`）：
  seq 代际守卫写法与既有 canonical 口径一致，`finally` 也做 seq 判定，避免「过期 finally 抢清在途
  请求的 loading」——这正是「loading 卡死」类缺陷的常见来源，处理正确。
- **`Job.total` 数据竞争**（`job.go`）：字段私有化 + `Total()` 加锁读，`Emit` 的「首学」分支只在
  `total == 0` 时写入，与注释声明的不变量一致；`record()` / `Finish()` 在锁内直读字段，无重复加锁。
- **`store` 加密失败不再吞错**（`store.go` / `sqlite.go` / `filestore.go`）：`encode` 改为返回 error
  并逐层上抛，错误分支由 `encryptAESGCMFn` 注入覆盖——「生产不可达」不等于「可以不测」，这里两者都做到了。
- **R4 trash 409 收窄**（`trash.go`）：从「一切 S3 API 错误 → 409」收窄为仅 `ObjectLocked → 409`，
  其余交由 `writeInternalErr` 的映射表，与 `GET /trash` 口径一致；删掉的 `s3wrap.IsAPIError` 全仓零引用。

### 3.2 安全

- **R1 / R2 端点级作用域闸**（`handler/scope.go`）：三段闸的判据与其**实际路由注册**逐条比对通过——
  `POST /api/accounts/preview-buckets`（`handler/routes.go:23`）、`GET|POST /api/migrate/jobs*`
  （`handler/routes.go:95-98`）与 `POST /api/accounts` 的路径 / 方法完全对应；Go 1.22+ `ServeMux`
  先对路径做 clean 与 301 归一，`//`、尾斜杠等变体进不到 handler，故 `==` / `HasPrefix` 判据不会被绕过。
  拒绝路径统一走 `denyScope`（写 `auth.scope_denied` 审计 + 不回显 token），符合硬约束 #4。
- **`UNSIGNED-PAYLOAD` 收窄**（`s3wrap/client.go`）：只在请求**带 stream** 时注入，无 body 的
  `GET/HEAD/DELETE/List` 交回 SigV4 的真空体哈希——把「可计算的完整性」还给无体请求。属签名路径行为
  变更，其最佳验证是真实 RustFS E2E（本评审**未复跑**，见 §4）。
- **`atomicfile` 临时名唯一化**（`atomicfile.go`）：`crypto/rand` 后缀消除「同名并发写互相删除临时
  文件」的窗口，并顺带回收历史固定名残骸；`O_EXCL` + 0600 + 父目录 fsync 的既有不变量未破坏。
- **SSRF 拨号**（`s3wrap/ssrf.go`）：`last` 初始化为非 nil，杜绝「解析器返回空列表 + nil error」时把
  `(nil, nil)` 交给 `net/http`（会被当作拿到空连接）——fail-closed 正确。
- 新增的明文端点 WARN **不记录任何密钥**（只记 id / name / 端点字段名）——除 N1 指出的 userinfo 面外无泄漏。

### 3.3 架构与可读性

- **分层未外泄**（硬约束 #3）：本批 handler 新增对 `internal/service` 的依赖（`PersistFailureCount` /
  `ScheduleValidationMessage`）与既有 `handler → service` 方向一致；AWS SDK 类型仍止步于 `s3wrap`；
  最大生产文件 `internal/openapi/openapi.go` 709 行，全仓无文件越过约 1000 行的拆分红线。
- **死代码零容忍**（硬约束 #5）：`describeACL` / `formatBuckets` 的导出面降级、`types.ts` 三个死类型
  导出删除、`IsAPIError` 删除，全部经 `unused` / `staticcheck` 与前端 TS AST 门禁验证；本批**新增**的
  `service/persist_metrics.go` 的 `notePersistFailure` / `PersistFailureCount` 均有真实调用点。
- **结构性去重**：`rowKey.ts` 收敛 7 处逐字复制的 3 行工厂、`useVirtualRows.ts` 收敛 4 处复制粘贴的
  虚拟滚动管线——**减少**了读者需同时持有的概念（行高单一来源、ResizeObserver 生命周期、换源归零
  三件事从「4 份各自维护」变成「1 处 canonical + 各组件只调用」），符合「重构要减少概念而非搬迁
  复杂度」。`useVirtualRows` 的 `watch(scrollEl)` + 兜底 `onMounted` 组合我按 Vue 生命周期核对过：
  首屏 ref 未绑定、弹窗懒渲染、卸载三种时序均覆盖；`ObjectContextMenu` 的 `useKeydownStack`
  第二参数 `{ immediate: true }` 已确认存在（否则「挂载即开菜单」时 Escape 永不注册，会是真的 a11y 回归）。
- **门禁自身可证伪**：本批新增 / 扩面的门禁（`archive_index_gate_test` / `openapi_status_declared_test` /
  `doc_alert_drift_gate_test` / `doc_status_drift_gate_test` / `ci_consistency_gate_test` 等）均带
  「扫描面自检阈值」（如 `len(routes) < 60` 即 Fatal），避免解析口径塌缩后「全绿但失明」；头注释写明
  断言范围、盲区与**变异验证**口径，符合 [DEVELOPMENT.md](../DEVELOPMENT.md) §4 的纪律。

### 3.4 契约与文档一致性

- **OpenAPI**：84 个 operation 的 `description` 由**已声明的契约事实**合成（summary + tag 说明 +
  声明的 2xx），显式值优先——不引入人工维护的散文，因此不会漂移；通用状态码（400/401/403/404/405/
  413/429/500 之外需逐端点声明）改为注册完成后由 `applyUniversalResponses` + `ForEachOperation`
  统一补挂，避免 84×4 处手写漂移。`openapi_spec_file_test` 保证提交版 `docs/api/openapi.json` 与
  运行时规范语义一致（我复跑通过）。
- **默认端口 8080 → 5000**：我全仓 grep 复核过迁移完整性——代码 / 脚本 / compose / nginx /
  `.env.example` / CI **无 8080 残留**；`docs/` 与 `CHANGELOG.md` 中残留的 8080 全部是**历史台账 /
  历史复跑记录**（按仓库「不回写历史结论」纪律应保留），非漂移。
- **指标 ⇔ 告警 ⇔ 文档三方**：新指标 `s3c_volume_inode_total` / `s3c_volume_inode_free` 在
  `handler/metrics.go` 发射点、`deploy/prometheus/s3client.rules.yml` 的 `S3ClientVolumeInodeLow`、
  [OPERATIONS.md](../OPERATIONS.md) §3.2 / §4.2 / §4.3 与 [DEPLOYMENT.md](../DEPLOYMENT.md) 四处一致，
  `for: 10m` 与文档口径相符（命名问题见 N2）。

## 4. 未复跑项与证据缺口（诚实披露）

| 未复跑 | 原因 | 影响 |
|---|---|---|
| `S3CLIENT_E2E=1 go test ./internal/s3wrap/ -run 'TestE2E' -v` | 需 docker + RustFS 镜像（本环境无 docker / 外网受限） | §3.2 的 `UNSIGNED-PAYLOAD` 收窄属**签名路径变更**，其真验证在此；本地 compose 内单测只覆盖了「无体请求不注入」这一点 |
| `make e2e-real` | 同上 | 前端直传 / 预签名 / OpenAPI 真断言三条结合部未在本轮实测 |
| `govulncheck` / `cargo audit` / Trivy 镜像扫描 | 需外网 | 「10 个可达 stdlib 漏洞归零」的结论本轮**未独立复验**（依赖 CI 的 `server` / `desktop` job 与既有批次证据） |
| 两套 CI（GitHub / GitLab）远端执行 | 无远端权限 | C1 / C2 的修复（Go job 补前端依赖前置、`fetch-depth: 0`）我只做了**静态复核**：全部 `actions/checkout` 的深克隆面已补齐（`rg -n 'fetch-depth' .github/` 命中 10 处），`ci_consistency_gate_test` 已把该不变式机械化 |

## 5. Nit 详述

### N1. 明文端点 WARN 原样记录用户提交的 endpoint（可能带 userinfo 凭据）

`apps/server/internal/handler/accounts.go:122-124` 的 `h.log.Warn(...)` 把 `a.Endpoint` /
`a.PublicEndpoint` **原文**写进日志。而 `s3wrap.ValidateEndpoint`（`internal/s3wrap/ssrf.go`）与
`s3wrap.NormalizeEndpoint`（`internal/s3wrap/client.go:210`）都**不剥离 URL 的 userinfo**——用户填
`http://key:secret@minio.local:9000` 会建号成功，随后这条 WARN 把凭据明文落盘。**建议**：落日志前用
`net/url` 解析并 `Redacted()`（或只记 host）。新增的 `accounts_plaintext_test.go` 目前只断言「是否打了
WARN」，未断言字段脱敏——可一并加一条行为断言。

### N2. gauge 指标用了 counter 的 `_total` 后缀

`apps/server/internal/handler/metrics.go:134-141` 发射的 `s3c_volume_inode_total` 是 **gauge**
（inode 总量），但 `_total` 后缀按 Prometheus 约定**保留给 counter**——同文件相邻的 `s3c_jobs_active`
（gauge）就用了 `_active` 这类中性后缀。风险是下游按约定把 `_total` 当单调计数器做 `rate()` /
`increase()` 而得到无意义结果，部分 lint 工具也会就此告警。该指标**本批新生**，改名成本最低的时点是
现在。**建议**：改为 `s3c_volume_inode_capacity`（或 `_count`），并同 PR 同步 `handler/metrics.go` /
`deploy/prometheus/s3client.rules.yml`（`S3ClientVolumeInodeLow` 的表达式与注释）/ [OPERATIONS.md](../OPERATIONS.md)
§3.2·§4.2·§4.3 / [DEPLOYMENT.md](../DEPLOYMENT.md) / [FEATURES.md](../FEATURES.md)。

### N3. `runHealthcheck` 文档注释重复行（历史遗留）

`apps/server/main.go:203-204` 连续两行以 `// "no-port-in-here"）时` 开头（第二行才是完整句），是本批
端口替换**之前**就存在的重复——本次 diff 只改了 `8080 → 5000`，未引入。属注释卫生，顺手删掉即可
（不影响任何门禁）。

## 6. 复现口径

```bash
# 后端（⚠️ 必须非沙箱：httptest 需要 listen()）
cd apps/server && gofmt -l . && go vet ./... && go build ./... && go test ./...
cd apps/server && go test -race -count=1 -timeout 600s -coverprofile=coverage.out ./... \
  && go tool cover -func=coverage.out | tail -1 \
  && awk 'NR > 1 && $NF == 0 { print "uncovered: " $0; bad=1 } END { if (bad) exit 1 }' coverage.out
cd apps/server && golangci-lint run ./...        # 期望 "0 issues."

# 前端
cd apps/web && pnpm lint && pnpm test && pnpm build

# 涉及签名 / 预签名 / 分段时（本评审未跑）
cd apps/server && S3CLIENT_E2E=1 go test ./internal/s3wrap/ -run 'TestE2E' -v
make e2e-real
```

## 7. 结论

**Approve**。本批 delta（2026-10-09 评审的全量闭环 + 随后的文档收口）在正确性、并发、安全、架构、
契约五个维度上均达到本仓库既有的高标准，且**没有把复杂度搬迁当重构**：新增的门禁都带自检与变异验证，
新增的抽象都收敛了重复而非中转。三道 Nit 均为非阻断的卫生问题，已转登记
[KNOWN_ISSUES.md](../KNOWN_ISSUES.md) #84–#86；建议随下一批次（顺手改）处置，无需为它们单独开分支。
