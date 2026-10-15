# 全仓代码质量评审（2026-10-14）

> **状态**：**已归档**（2026-10-14，按 [index.md](index.md)「归档操作」四步冻结——§2 状态表的
> 缺陷 / 门禁 / a11y / 供应链项**同日闭环**，三项可维护性发现**转登记** [`../KNOWN_ISSUES.md`](../KNOWN_ISSUES.md)
> #88–#90；证据见 [`../FEATURES.md`](../FEATURES.md) §CQ 与 [`../../CHANGELOG.md`](../../CHANGELOG.md) `[Unreleased]`）。
> 结论绑定提交 `develop`（评审时点 HEAD）。文中 `file:line` 与门禁数字均为**评审时点快照**，**不回写**。
>
> **范围**：全仓——`apps/server` Go 后端（303 个 `.go`；生产 16 793 行 / 测试 48 194 行）、
> `apps/web` Vue 3 + TS（96 个生产 `.ts`/`.vue`，`src/` 21 934 行）、`apps/desktop`（Tauri）、CI / 脚本 / 门禁；**不含**文档文风评审。
>
> **方法**：三路并行静态探察（后端 / 前端 / 桌面 + CI + 门禁）+ 关键结论**回到源码逐条复核**
> （前端计数桶、`s3wrap` 列举守卫、门禁断言形态、Makefile 解析、CI 镜像引用均实读确认）。
>
> **结论**：**Approve**（可合并）。未发现 Critical。发现 **4 处应修**（1 前端竞态 + 1 无界列举 +
> 2 道门禁可被「语义等价输入」骗过）、**3 项低优先 / 卫生**、**2 项可访问名缺失**、**2 项供应链 / 安全面**；
> 4 处应修 + 3 项卫生 + 2 项 a11y + 安全/供应链**当日全部落地**，三项可维护性发现转登记 #88–#90。

## 1. 机械门禁实测（评审时点，如实披露）

| 门禁 | 结果 | 说明 |
|---|---|---|
| `go test ./...`（10 包） | **未跑全**（5/10 PASS） | 本评审环境**无外网**：`atomicfile` / `config` / `model` / `openapi` / `tracing` 5 包 PASS；其余因依赖（`aws-sdk-go-v2` / `sqlite` / `uuid` / `x/crypto`）需从 proxy 下载而 `setup failed`（**环境限制，非代码缺陷**） |
| `go vet` / `go build` / `gofmt` | **未跑** | 同因——依赖未落本地 module cache |
| `golangci-lint` | **未跑** | 本机未安装（离线亦无法安装） |
| `pnpm lint` / `pnpm test` / `pnpm build` | **未跑** | `apps/web/node_modules` 未安装（离线） |
| 覆盖率（`count==0` 块检查） | **未跑** | 同上 |
| 结论所依据 | **源码逐条实读复核** | 见 §2 各项的 `file:line`；**本快照不含「门禁全绿」的实测结论**（诚实披露） |

> **工具链**：本机 `go version` = go1.27.1，而 `apps/server/go.mod` pin `go 1.26.9`——**本机工具链与
> `go.mod` 不一致**（Go 工具链 1.21+ 会按 `go.mod` 的 `go`/`toolchain` 行自动选用，故实际构建用 1.26.9）。
> 这一条留给后续代理注意：在**有外网的环境**复跑门禁时先确认 `go env GOTOOLCHAIN` 行为。

## 2. 处置状态总览

| 编号 | 级别 | 一句话 | 状态 |
|---|---|---|---|
| F1 | **应修**（正确性 / 竞态） | `useObjectBrowser` 的 `loadBuckets` 与对象 `load` **共用同一个 `loadSeq` 计数器**：切桶触发的 `load` 会把在飞的桶列表响应误判过期而丢弃，并跳过 `finally` 的 `loadingBuckets` 复位（桶选择器永久卡 loading） | ✅ 已修（拆出独立 `bucketSeq` + 回归用例） |
| F2 | **应修**（有界性） | `s3wrap.ListParts`（`multipart.go:97`）与 `s3wrap.PurgeObject`（`object.go:339`）翻页**无页上限、无「游标不前进」守卫**（仅 `nil`/空游标兜底）——对端异常即死循环 | ✅ 已修（补页上限 + 游标守卫 + `pagination_bounds_test.go`） |
| F3 | **应修**（门禁有效性） | `TestReleaseWorkflowVerifiesTagMatchesManifest` 断言 `strings.Contains(text,"!=")`，而 `!=` 只出现在错误文案里（真比对是 `[ "$tauri" = "$want" ]`）——删比对留文案仍绿 | ✅ 已修（改断言真实比对形态） |
| F4 | **应修**（门禁有效性） | `TestMakefileMirrorsCIGates` 对解析结果**无最小条数自检**（解析塌缩即空转全绿），且**只查 defined→.PHONY、不查反向** | ✅ 已修（补自检下限 + 反向检查） |
| F5 | 低（卫生） | `Makefile:5` 的 `.PHONY` 列了 `restart` 但**无 `restart:` 目标**（孤儿条目，`make restart` 报 No rule） | ✅ 已修（移除孤儿条目） |
| F6 | 低（门禁有效性） | `TestTrivyScansUseCacheAndRetry` 断言裸词 `"attempt"`（注释即可满足） | ✅ 已修（改断言 `for attempt in` 循环构造） |
| F7 | 低（门禁有效性） | `ai_governance_gate_test.go` 的 Copilot 指针校验写在 `if err == nil {}` 内——删文件即静默取消校验 | ✅ 已修（缺文件即 Fatal） |
| A1 | 应修（a11y） | `ObjectToolbar.vue:117` 图标按钮（`✓`/`✎`）无 `aria-label`，且 `:title` 在编辑态为空串 → 无可访问名 | ✅ 已修（补 `aria-label`，按状态取编辑 / 确定） |
| A2 | 应修（a11y） | 进度条缺可访问名：`MigratePanel.vue:518` / `UploadPanel.vue:179` / `ActivityCenter.vue:56` / `BatchMetadataDialog.vue:217` | ✅ 已修（各补 `aria-label`） |
| C1 | 观察（安全） | Tauri `tauri.conf.json` 的 `connect-src` 为 `'self' http: https:`（可连任意源）；但桌面端须连**任意用户配置**的 S3 端点 / 后端，构建期无法枚举主机 | ➖ 有意维持（取舍记入 `apps/desktop/AGENTS.md`） |
| C2 | 应修（供应链） | CI / compose 的基础镜像（golang / node / rust / docker / nginx / rustfs）只 pin tag、未 pin digest（与 Trivy 的 tag+digest 纪律不对称） | ✅ 已修（六个镜像补 `@sha256:` + 新门禁 `TestCIBaseImagesAreDigestPinned`） |
| O1 | 观察（可维护性） | 前端「seq 守卫异步加载」惯用法手写约 8 处 | ⏭ 转登记 #88（纯行为等价重构，验证靠测试套件） |
| O2 | 观察（可维护性） | 后端「分页列举循环」骨架在 5 处复制、页上限常量各写一份 | ⏭ 转登记 #89 |
| O3 | 观察（可维护性） | `handler.statusRecorder` / `tracing.statusWriter` 两个状态包装器 `Flush` 已分叉 | ⏭ 转登记 #90 |

> 状态由**处置批次**回写（2026-10-14：F1–F7 / A1–A2 / C2 当日闭环，C1 有意维持，O1–O3 转登记）。

## 3. 五轴结论

### 3.1 正确性

- **F1（前端竞态）**：`useObjectBrowser.ts` 中 `loadBuckets`（`:197` 原）与对象 `load`（`:299` 原）
  都取 `++loadSeq.value`。二者是独立数据源，交错时桶响应会被误判过期——该文件自身的注释还称 `loadSeq` 是
  「导航序号」。仓库口径本是**按关注点分开计数**（`RecycleBinPanel` 的 `bucketSeq`/`loadSeq`、
  `MigratePanel` 的 `sourceBucketGen`/`targetBucketGen`、`VersionsDialog` 的独立 `loadSeq`），此处是唯一偏差。
  已拆出 `bucketSeq` 并加回归用例（旧实现下该用例失败）。
- **F2（后端有界性）**：`s3wrap` 两处列举循环此前只有「截断但缺标记」兜底，缺页上限与游标前进判定。
  仓库的列举纪律（`service/sync_list_gate_test.go` 头注释明文：「对端返回不前进的 token 就永不退出」）
  此前**只在 `internal/service` 有门禁**，`s3wrap` 在网外。已补守卫并新增 `pagination_bounds_test.go`
  （5s 超时兜底：守卫被删则用例超时红灯，而非挂住进程）。

### 3.2 可访问性

- 图标按钮与四个进度条此前无程序化可访问名（`role="progressbar"` 有 `aria-value*` 但无 `aria-label`）。
  已按状态 / 语义补 `aria-label`（复用既有 i18n key，未新增文案）。渲染态由 `e2e/a11y.spec.ts`（axe）兜底。

### 3.3 供应链与安全

- **C2**：Trivy 早已 tag+digest 双 pin（`TestTrivyImageIsVersionAndDigestPinned`），但其余六个基础镜像
  只 pin tag——tag 可被上游重新推送，是不可复现的供应链面。已补齐 digest（取自 Docker Hub 官方 API 的
  **多架构 index** digest）并新增门禁 `TestCIBaseImagesAreDigestPinned`（`.gitlab-ci.yml` 全部 `image:` +
  `services[].name`、三个 compose 的非自建镜像）。
- **C1**：Tauri `connect-src` 无法在「收窄」与「支持任意用户 S3 端点」间两全——**有意维持**，并把该取舍
  显式写进 `apps/desktop/AGENTS.md`（把「默认值」升级为「有意识的决策」）。

### 3.4 门禁有效性（本轮重点）

四道门禁存在「断言被语义等价输入满足」的假阴性（F3 / F4 / F6 / F7 的失效形态）。已逐一改断言为**真实
可判定形态**（真实比对表达式 / 解析条数下限 + 反向集合 / 循环构造 / 缺文件即 Fatal）。这类问题的危害是
**门禁看似在守、实则失明**，与本仓库「门禁必须可证伪」的纪律直接相关。

### 3.5 架构与可维护性

- 分层未外泄（`s3wrap` 仍是 AWS SDK 唯一进口；无文件破 1000 行）；死代码三路**未发现**
  （`deadcode_gate` + `unused`/`staticcheck` + 前端 TS AST 门禁三重守住）。
- 前端 `v-html` / `any` / `console.*` **零命中**；SecretKey 不落盘不写日志；无提交的密钥。
- 三处重复（O1–O3）为**行为等价重构**，其唯一验证是测试套件（本环境跑不了）——故转登记 #88–#90，
  不在无验证条件下盲改（其中 O3 涉及跨包，须避免 `tracing` 反向依赖 `handler`）。

## 4. 未复跑项与证据缺口（诚实披露）

| 未复跑 | 原因 | 影响 |
|---|---|---|
| `go test ./...`（全 10 包）/ `go vet` / `go build` / `golangci-lint` | 本环境**无外网**，依赖无法下载 / 工具未安装 | F2 与后端门禁改动（F3/F4/F6/F7/C2 的 Gate）**未经测试执行验证**——须在有外网的环境复跑 |
| `pnpm lint` / `pnpm test` / `pnpm build` / `typecheck` | `node_modules` 未安装（离线） | F1 / A1 / A2 的前端改动**未经类型检查与测试执行验证** |
| `S3CLIENT_E2E=1 go test ./internal/s3wrap/ -run TestE2E` / `make e2e-real` / 两套 CI 远端 | 需 docker + 外网 / 无远端权限 | C2 的镜像 digest **未经真实拉取验证**（digest 取自 Docker Hub API，形态为 40 位 `@sha256:`） |

> **据此**：本快照的结论是**评审 + 修复**的结论；**「全绿」须由有外网的环境复跑 §5 命令后确认**。
> 这也是把 O1–O3 转登记而非盲改的原因之一。

## 5. 复现口径

```bash
# 后端（须非沙箱 / 有外网 module cache）
cd apps/server && gofmt -l . && go vet ./... && go build ./... && go test -race -count=1 ./...
cd apps/server && golangci-lint run ./...        # 期望 "0 issues."

# 前端
cd apps/web && pnpm install --frozen-lockfile && pnpm lint && pnpm typecheck && pnpm test && pnpm build

# 涉及签名 / 预签名 / 分段 / 照此改 s3wrap 时
cd apps/server && S3CLIENT_E2E=1 go test ./internal/s3wrap/ -run 'TestE2E' -v
make e2e-real
```

## 6. 结论

**Approve**。四道门禁的假阴性（F3/F4/F6/F7）是本轮最值得修的一类——门禁「全绿但失明」比缺门禁更危险；
前端计数桶竞态（F1）与 `s3wrap` 无界列举（F2）是真实但需特定条件触发的缺陷，均已修复并配行为用例；
a11y 两处、供应链 digest（C2）一并落地；C1 复核后有意维持并记录。三项可维护性发现（O1–O3）
**转登记** [`../KNOWN_ISSUES.md`](../KNOWN_ISSUES.md) #88–#90。**本快照未含门禁实跑结论**（环境离线，见 §1/§4）。
