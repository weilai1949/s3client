# 术语表（Glossary）

> 本文件统一 s3client 仓库里的两套词汇：**A. S3 / 对象存储领域术语**（产品面向的外部概念）
> 与 **B. 本项目自造词 / 内部术语**（代码、测试、门禁与文档里反复出现、但外部无标准定义的叫法）。
> 收录原则：**只收真能在本仓库代码或文档里找到依据的词**，并在条目里点名出现位置；
> 找不到依据的一律不收，宁少勿编。S3 语义细节以 [`api.md`](api.md) 为准，
> 错误映射以 [`errors.md`](errors.md) 为准，开发规范以 [`DEVELOPMENT.md`](DEVELOPMENT.md) 为准。

---

## A. S3 / 对象存储领域术语

| 术语 | 定义 | 本仓库依据 |
|---|---|---|
| **桶（Bucket）** | S3 的顶层命名容器，账号内唯一；对象都以「桶 + key」定位 | [`api.md`](api.md) §账号「列出桶 / 创建桶 / 删除桶」；桶名严格校验（长度、首尾字符、连续 `..` / `.-` / `-.`）见 [`FEATURES.md`](FEATURES.md) §一.2 |
| **对象（Object）** | 桶内的数据单元，由 key、内容、元数据、ETag、存储类型等组成 | [`FEATURES.md`](FEATURES.md) §一.3「对象详情」；`apps/server/internal/s3wrap/object.go` |
| **key（对象键）** | 对象在桶内的完整路径字符串。本控制台**不做**「目录」实体，目录是 key 前缀的呈现 | [`FEATURES.md`](FEATURES.md) §一.3「目录浏览」；`handler/objects.go` 的 `key` / `prefix` 参数 |
| **前缀（prefix）** | key 的字符串前缀，用于列举与批量操作的作用域。空前缀在本仓库被**显式拒绝**（防误删全桶），删除前缀上限 10 万个对象 | [`api.md`](api.md) §删除文件夹（递归） |
| **分隔符（delimiter）** | 列举时用于「折叠」层级的分隔字符，控制台默认 `/`；服务端透传给 S3 | `s3wrap/object.go`（`listObjectsV2` 的 `in.Delimiter`）、`handler/objects.go` 的 `q.Get("delimiter")` |
| **分页游标 `continuationToken` / `startAfter`** | `ListObjectsV2` 的续页与「从某 key 之后开始」两种口径；两者**互斥使用** | [`api.md`](api.md) §列出对象 |
| **版本控制状态（versioning）** | 桶级开关。开启后同一 key 保留多版本，删除对象只写删除标记 | [`FEATURES.md`](FEATURES.md) §一.2「版本控制」；`api.md` §桶版本控制开关 |
| **`versionId`** | 对象某一版本的标识。详情、删除、回滚、预签名 GET 都可带上它定位历史版本 | [`api.md`](api.md) §对象详情 / 删除指定版本 / 版本回滚 |
| **`isLatest`** | 标记某版本是否为该 key 的当前版本 | [`api.md`](api.md) §对象版本列表（`ListObjectVersions`） |
| **删除标记（delete marker）** | 版本控制桶里「删除对象」产生的**无数据版本**；撤销删除 = 删除该标记本身 | [`api.md`](api.md) §一键还原已删除对象（恢复删除标记） |
| **回收站（trash）** | 本控制台对「桶内删除标记」的用户可见称呼，由 `ListObjectVersions` 过滤出删除标记得到 | [`api.md`](api.md) §回收站（列出删除标记）；`RecycleBinPanel.vue` |
| **彻底清除 / purge** | 删除某 key 的**全部版本与删除标记**，不可再还原；非版本控制桶则兜底删除当前对象 | [`api.md`](api.md) §彻底清除（永久删除对象） |
| **分段上传（multipart upload）** | 大文件拆段上传后组装：`CreateMultipartUpload` → 逐段 `UploadPart` → `CompleteMultipartUpload` | [`api.md`](api.md) §分段上传（大文件直传）；[`DEVELOPMENT.md`](DEVELOPMENT.md) §2 的「分段 Multipart 组装」E2E |
| **段号（`partNumber`）** | 分段序号，`CompleteMultipartUpload` 要求**升序**；乱序在本仓库被 handler 先行拦下 | [`errors.md`](errors.md)「分段顺序不是升序」→ 400 `parts must be ordered by ascending partNumber` |
| **ETag** | 对象内容的实体标签，也是分段直传后需要从响应头读回的值（要求桶 CORS **暴露** `ETag`） | [`api.md`](api.md) §分段上传；[`DEVELOPMENT.md`](DEVELOPMENT.md) §2「浏览器直传」 |
| **预签名 URL（presigned URL）** | 服务端用临时凭证签出的、有时效的直连 S3 地址，用于浏览器直传 / 直下，**不经**后端代理 | [`api.md`](api.md) §生成签名；[`threat-model.md`](threat-model.md) 边界 B |
| **`expiresIn`** | 预签名有效期（秒）。缺省或 ≤0 时默认 **1 小时**，超过 **24 小时**被钳到 24 小时（S3 协议上限 7 天，控制台刻意收紧） | [`api.md`](api.md) §生成签名（默认与钳制在 `handler/objects.go` 的 `presign` 与 `handler/multipart.go`；`s3wrap/presign.go` 只负责签名与 `≤0` 拒绝） |
| **path-style** | 端点寻址方式之一：把桶名放进路径（`/{bucket}/{key}`）。兼容 MinIO / OSS 等自建实现，账号参数里由 `pathStyle` 开关控制 | [`api.md`](api.md) §创建账号（`pathStyle` 用于 MinIO/OSS 等第三方）；[`DEVELOPMENT.md`](DEVELOPMENT.md) §2 假 S3 模式 |
| **virtual-hosted-style** | 另一种寻址方式：桶名进主机名（`{bucket}.endpoint`）。本仓库把它作为 path-style 的**对立项**存在（`pathStyle` 为假时由 AWS SDK 走该风格） | 同上（`pathStyle` 参数的语义对称面；`s3wrap/client.go` 中按该标志构建 client） |
| **`publicEndpoint`** | 供**浏览器直传 / 预签名**使用的对外端点（与后端内部访问用的 `endpoint` 分离，留空则回落到 `endpoint`） | [`api.md`](api.md) §创建账号；[`CONFIGURATION.md`](CONFIGURATION.md) 客户端设置 |
| **区域（region）** | 桶所在的 S3 区域；账号可设缺省区域，桶属性里可读实际区域 | [`CONFIGURATION.md`](CONFIGURATION.md) `S3C_REGION`；[`FEATURES.md`](FEATURES.md) §一.2「桶属性」 |
| **ACL** | 对象的访问控制列表。控制台的 ACL 编辑器列出固定枚举项（如 `public-read`），不做自由文本 | [`api.md`](api.md) §对象权限（ACL）；`AclDialog.vue`、`i18n/messages/objectDialogs.ts` 的 `acl.*` |
| **标签（Tagging / tags）** | key-value 形式的对象或桶元数据。**两层语义**：前端 UI 层的「空标签行提交」被显式禁止（`hasTagChange` 判定，不发无意义请求）；API 层 `tags` 传空数组则**删除全部标签**（`api.md` §对象标签） | [`api.md`](api.md) §对象标签（Tagging）/ §桶标签；[`FEATURES.md`](FEATURES.md)「UX-14 空标签提交 → `hasTagChange`」（`BatchMetadataDialog.vue`） |
| **对象 HTTP 头** | 可写的响应头族：`Cache-Control` / `Content-Disposition` / `Content-Encoding` / `Content-Language` / `Content-Type` 以及自定义元数据 | [`api.md`](api.md) §设置对象 HTTP 头；`HeadersDialog.vue` |
| **生命周期规则（lifecycle）** | 桶级「前缀过期删除」规则，本控制台只做前缀 + 过期天数这一子集 | [`api.md`](api.md) §生命周期规则（前缀过期删除）；`LifecycleDialog.vue` |
| **SSE（服务端加密）** | 桶级服务端加密开关与算法配置 | [`api.md`](api.md) §桶服务端加密（SSE）；`BucketEncryption.vue` |
| **CORS（桶级）** | 桶的跨源规则（允许方法 / 来源 / 头 / 暴露头 / 缓存时长）。**浏览器直传**依赖它：缺 CORS 会被浏览器拦下，而 curl / Playwright `APIRequestContext` **不经** CORS，只用它们验证会「假绿」 | [`api.md`](api.md) §桶 CORS 规则；[`DEVELOPMENT.md`](DEVELOPMENT.md) §2「必须给 RustFS 配 `RUSTFS_CORS_ALLOWED_ORIGINS`」 |
| **静态网站托管（website）** | 桶级静态网站配置（首页 / 错误页等）的读写开关 | [`api.md`](api.md) §桶静态网站托管；`BucketWebsite.vue` |
| **桶策略（bucket policy）** | 桶级 JSON 访问策略。控制台提供可视化编辑器（Statement 表单 + 模板 + 实时 JSON 预览）与原始 JSON 回退 | [`FEATURES.md`](FEATURES.md) §一.2「桶策略」；`BucketPolicy.vue` / `BucketPolicyVisualEditor.vue`；[`api.md`](api.md) §桶策略 |
| **存储类型（StorageClass）** | 对象的存储层级。切换靠 `CopyObject` 副本写回自身并携带 `x-amz-storage-class`，因此版本控制桶下会**产生新版本** | [`api.md`](api.md) §切换对象存储类型（支持的取值枚举同节）；`StorageClassDialog.vue` |
| **`ListObjectsV2`** | 列举桶内**当前对象**（每 key 只出现一次），支持前缀 / 分隔符 / 分页游标；控制台的对象列表与批量操作都基于它 | [`api.md`](api.md) §列出对象；`s3wrap/object.go` 的 `listObjectsV2` |
| **`ListObjectVersions`** | 列举桶内**全部版本 + 删除标记**，游标是 `keyMarker` + `versionIdMarker` 两个。与 `ListObjectsV2` 的关键差别：**能看见被删除与历史版本**，代价是同一 key 会重复出现多次 | [`api.md`](api.md) §对象版本列表（`ListObjectVersions`）；回收站即「`ListObjectVersions` 过滤为删除标记」 |
| **增量同步（sync）** | 跨账号迁移的子模式：按 **ETag / size+mtime / always** 三种口径比对，只复制有差异的对象 | [`api.md`](api.md) §增量同步（按 ETag / size+mtime 比对，仅复制差异对象） |
| **异步任务（job）与 SSE 进度** | 迁移 / 复制 / 删除等长任务在服务端登记为 job，进度经 SSE 推送；job 有终态与保留期 | [`api.md`](api.md) §异步迁移（SSE 进度）；`service/job.go`、`web/src/api/jobs.ts` |
| **S3 兼容存储 / RustFS** | 上游对端。本仓库用 **RustFS** 作自托管测试对端（默认 `rustfsadmin/rustfsadmin`，S3 API 9000、控制台 9001） | [`DEVELOPMENT.md`](DEVELOPMENT.md) §2「真实 RustFS 联调」 |

---

## B. 本项目自造词 / 内部术语

> 这一节的词是本仓库长期使用、但**外部无标准定义**的叫法。定义尽量给出「为什么这么叫」，
> 并点名唯一来源文件，避免多处各写一份造成漂移。

### B.1 流程与门禁

| 术语 | 定义 | 出现位置 |
|---|---|---|
| **门禁** | 任何**机械可判**的硬性检查（CI job、`*_gate_test.go`、覆盖率阈值、lint 0 issues）。门禁只认「绿 / 红」，不接受「历史遗留」作为例外；新增规则要配「防门禁静默失效」的自检（扫描文件数下限、声明必须命中） | [`DEVELOPMENT.md`](DEVELOPMENT.md) §3 必验门禁；[`AGENTS.md`](../AGENTS.md)「提交前门禁（必须全绿）」 |
| **红灯 / 绿灯** | 测试失败 / 通过的拟物说法。**先红灯后绿灯**＝先看到测试因缺实现而失败，再写实现让它通过；每个修复项都要求「先红后绿」的证据 | [`DEVELOPMENT.md`](DEVELOPMENT.md) §1.4「红灯-绿灯-重构（红绿蓝）」；[`FEATURES.md`](FEATURES.md) §二 各项 |
| **变异验证** | 主动把实现或文档改坏（删一行、改回旧写法、注入假符号），确认门禁**确实会红**，再撤回并复绿——用来证明门禁不是恒真的摆设 | [`FEATURES.md`](FEATURES.md)（如 Gate 1「注入 `MutantDeadExportedFunc` → 精确红灯」、i18n 死键门禁「改名一个 en-US 键 → 集合门禁红灯」） |
| **覆盖率门禁（100% 且查 `count==0`）** | 后端**不比较百分比**而是直接检查 profile 里是否存在 `count==0` 的语句块——`total` 只有 1 位小数，99.96% 会被四舍五入显示成 100.0% 而漏过回退；前端用四指标阈值 100% | [`DEVELOPMENT.md`](DEVELOPMENT.md) §3 的块引用；`apps/web/vite.config.ts` 的 `thresholds`；[`ROADMAP.md`](ROADMAP.md) |
| **gap 测试** | 为**凑覆盖率**而写、与实现耦合、并不验证外部行为的测试。本仓库把它列为 Red Flag：遇到确实不可达的分支，正确做法是**删死代码**，而不是写 gap 测试把它「测活」 | [`DEVELOPMENT.md`](DEVELOPMENT.md) §3 块引用、§6 Red Flags；[`AI_POLICY.md`](AI_POLICY.md) §9「覆盖率门禁红灯」处置 |
| **假 S3（模式）** | handler 层测试里用 `httptest.NewServer` 起的最小 S3 替身：按 query（`acl` / `tagging` / `versions` / `location` / `versioning`）与方法分发标准 XML，并可返回特定错误码验证容错 | [`DEVELOPMENT.md`](DEVELOPMENT.md) §2「假 S3 模式」 |
| **真实对端 E2E** | 不 mock、连真实 S3 兼容实现的测试层：`S3CLIENT_E2E=1 go test ./internal/s3wrap/ -run 'TestE2E'`（默认指向本地 RustFS），普通 `go test ./...` 不会执行 | [`DEVELOPMENT.md`](DEVELOPMENT.md) §2 / §3 |
| **真实联调（`make e2e-real`）** | 真实 Go 后端（托管真实 `vite build` 产物）+ 真实 RustFS + 真实浏览器，**不 mock `/api`**；编排唯一来源是 `scripts/e2e-real.sh`，两套 CI 与本地共用 | [`DEVELOPMENT.md`](DEVELOPMENT.md) §2 / §3；`scripts/e2e-real.sh` |
| **真值来源（SSOT）** | 「某个事实只有一处权威定义」：配置项以 [`CONFIGURATION.md`](CONFIGURATION.md) 为 SSOT；md 正文的「N 个 `/api/*` 端点」以 `routes.go` 的注册数为唯一真值 | [`CONFIGURATION.md`](CONFIGURATION.md) 标题块；[`DEVELOPMENT.md`](DEVELOPMENT.md) §4 |
| **文档同步门禁** | 「改完代码必须同 PR 更新对应文档，否则视为改动未完成」的对照表机制（改动类型 → 必须同步的文档） | [`DEVELOPMENT.md`](DEVELOPMENT.md) §4 |
| **Fail-closed（拒绝启动）** | 配置可疑时**宁可起不来**也不静默降级。例如显式 `S3C_ENV_FILE` 缺失即返回哨兵 `ErrInvalidEnvFile` 使启动失败；未知 `S3C_STORE_DRIVER` 拒绝启动 | [`CONFIGURATION.md`](CONFIGURATION.md) §1 块引用与 §3 清单；ADR [`decisions/0002-store-fail-closed.md`](decisions/0002-store-fail-closed.md) |

### B.2 代码与测试模式

| 术语 | 定义 | 出现位置 |
|---|---|---|
| **死代码** | 定义后**无人引用**的函数 / 类型 / 常量 / 变量 / 字段、不可达分支，以及未定义即使用的标识符。零容忍，**枚举值除外**（枚举/常量表成员算契约，即使暂无引用也保留） | [`AGENTS.md`](../AGENTS.md) 硬约束 5；[`DEVELOPMENT.md`](DEVELOPMENT.md) §5 / §6 |
| **消音式死代码** | 用 `_ = x` / `var _ = f` / `//nolint` / 导出成 `_test` 辅助等手段把死代码「藏起来」让它不再报警。`apps/server/deadcode_gate_test.go` 用 AST 判定并禁止这类写法（允许 `_ = f()` 这种显式忽略返回值），`nolintlint` 要求 `//nolint` 必须写明 linter 与理由 | `apps/server/deadcode_gate_test.go`（文件头注释与 `silenceViolation` 用例）；[`DEVELOPMENT.md`](DEVELOPMENT.md) §6 |
| **被覆盖率掩盖的死代码** | 测试文件不参与 instrumentation，`go test -cover` 只看生产代码——于是「测试辅助里的死代码」能让 100% 覆盖率门禁全绿。结论：覆盖率达标 ≠ 无死代码，两者必须**分别验证** | [`DEVELOPMENT.md`](DEVELOPMENT.md) §6 最后一条；[`AGENTS.md`](../AGENTS.md) 硬约束 5 的同名警告 |
| **前向门禁 / 反向门禁** | 前向＝「被引用 → 必须有定义」；反向＝「有定义 → 必须被引用」。i18n 字典曾只做前向，导致约 40 个死键长期堆积（键数虚高、译者重复劳动），后补反向门禁 | `apps/web/src/i18n/coverage.test.ts` 的两个用例注释；[`FEATURES.md`](FEATURES.md) §二 L1（D7） |
| **豁免腐烂检查** | 门禁的「白名单」本身也要被检查：`INTENTIONAL_UNUSED` 里若登记了**已不存在**的符号即红灯，防止豁免清单越积越假 | `apps/web/src/deadcode_gate.test.ts`「INTENTIONAL_UNUSED 不得登记已不存在的符号」 |
| **空跑变绿** | 门禁因为路径写错 / 扫到 0 个文件而「没有违规所以通过」。对策是自检下限：如扫描文件数 ≥ 50、公开面 ≥ 50、引用总数 ≥ 30 | `apps/web/src/deadcode_gate.test.ts`「门禁自检」；`apps/server/*_gate_test.go` 同类自检 |
| **哨兵错误（sentinel error）** | 用 `errors.New` 定义的、可被 `errors.Is` / `errors.As` 识别的**具名错误值**，用来替代「按错误文案 `strings.Contains` 匹配」。即使上游 SDK / 各 S3 实现的文案变化也不会静默失效 | `apps/server/internal/s3wrap/errors.go`（`ErrObjectTooLarge` / `ErrSourceDeleteFailed` / `ErrPartialDelete`）；[`errors.md`](errors.md)「已移除字符串匹配」块引用 |
| **终态（terminal state）** | 异步任务 / 上传项不再变化的收尾状态（`done` / `cancelled` / `interrupted`…）。终态必须**可判定且有界**：SSE 流若以 EOF 结束却没收到终态，Promise 会永久悬挂、按钮永久禁用——故用「EOF 后回读轮询直到终态」终结它 | `apps/server/internal/service/job_persist.go`（`IsTerminalJobStatus`）；`apps/web/src/api/jobs.ts`；[`DEVELOPMENT.md`](DEVELOPMENT.md) §7 的 S7 / P0-4 行 |
| **兜底** | 主路径之外的那条**保底分支**：它可能几乎不触发，但缺了就会挂起 / 白屏。与「死代码」的区别是**能否由公开 API 触发**——能触发就必须保留并用行为断言覆盖，不能为了覆盖率或「没人走」删掉 | [`DEVELOPMENT.md`](DEVELOPMENT.md) §3 块引用（`jobsList` nil 兜底被删、`JobProgress.Status` 兜底被保留的判例）；`api/jobs.ts` 的 EOF 回读兜底 |
| **代次守卫（seq guard / 代次）** | 每个「可被新请求顶掉」的异步加载持有一个自增序号（`loadSeq` / `detailSeq` / `sourceBucketGen` …），每个 `await` 之后比对序号：**过期响应静默丢弃**，`catch` 与 `finally` 都按「序号仍相等」收口。防的是迟到响应覆盖新数据、把新对象的 loading 提前清掉、把旧错误报到新对象头上 | `apps/web/src/components/VersionsDialog.vue`（`loadSeq`）、`RecycleBinPanel.vue`、`composables/useObjectActions.ts`（`detailSeq`）、`MigratePanel.vue`（`sourceBucketGen` / `targetBucketGen`）；[`FEATURES.md`](FEATURES.md) §AI；[`architecture.md`](architecture.md) |
| **在途请求（in-flight）** | 已发出、尚未返回的请求。切账号 / 换前缀 / 关弹窗后，在途请求的返回都属于「过期」；本仓库要求早退路径也**递增代次**把它作废 | [`FEATURES.md`](FEATURES.md) §AI 第 4 条；[`DEVELOPMENT.md`](DEVELOPMENT.md) §7 的 S7 行（「流以 EOF 结束时 Promise 悬挂」） |
| **分层（`store → model → s3wrap → handler`，栈序简化）** | 后端单向分层的口头简称；**逐条 import 边以 [`architecture.md`](architecture.md) §2 为权威**（`s3wrap` 只依赖 `model`、不依赖 `store`；`handler` 直接依赖 `store`）。不变的硬约束：AWS SDK 类型只在 `s3wrap` 内出现，handler / 前端不得外泄；单文件不超过约 1000 行 | [`AGENTS.md`](../AGENTS.md) 硬约束 3；[`architecture.md`](architecture.md) §2；[`DEVELOPMENT.md`](DEVELOPMENT.md) §5 |
| **防腐层** | 把外部系统（AWS SDK、S3 错误码、上游文案）的差异挡在边界内、对外只暴露**稳定**状态与短消息的那一层。本仓库是 `s3wrap`（错误 → 稳定 HTTP 状态 + 短英文 `UserMessage`，前端再 i18n） | [`errors.md`](errors.md) 首段；[`FEATURES.md`](FEATURES.md) §一.10「`UserMessage` 防腐层」 |
| **有界并发** | 并发度有显式上限、不会随输入规模无限增长。批量任务、ZIP 拉取、分段上传都走这一模式 | `service/batch.go`（`RunBatch` / `CopyKeys`）、`service/zip.go`（拉取最多 4）、`service/stream_copy.go`；[`DEVELOPMENT.md`](DEVELOPMENT.md) §5「性能」 |
| **虚拟滚动（窗口化）** | 只渲染视口内的行 + 上下垫片撑出真实滚动高度，用于 20 万行量级的列表；`virtualList.ts` 提供窗口计算，`ObjectList` / `RecycleBinPanel` / `VersionsDialog` / `MigratePanel` 复用 | `apps/web/src/virtualList.ts`；[`FEATURES.md`](FEATURES.md) §一.3「20 万行虚拟滚动」 |
| **反模式（Red Flag）** | 仓库明确要求「遇到即停下修正」的写法清单（没看到红灯就写实现、用 `any` 掩盖不变量、一次升级一批依赖、功能逻辑渗入共享工具模块、「以后再说」的清理等） | [`DEVELOPMENT.md`](DEVELOPMENT.md) §6；[`AI_POLICY.md`](AI_POLICY.md) §0 边界声明 |
| **技术债的现行守卫** | 已闭环的历史债不再当问题登记，而是转写成「什么动作会把它改坏」的规则 + 回归即红灯的门禁。例如 endpoint 归一化只允许走 `s3wrap.NormalizeEndpoint`，SSE 终态只允许走 `subscribeMigrateEvents` | [`DEVELOPMENT.md`](DEVELOPMENT.md) §7（H1 / S7 / D5 三条）；[`KNOWN_ISSUES.md`](KNOWN_ISSUES.md) |
| **分叉（同一逻辑多份实现）** | 同一件事被两处以上各写一遍，行为随后漂移（S7 的 SSE 终态检测、D5 的 endpoint 归一化都是实例）。处置方式是**收敛为单一实现**并让所有调用方共用 | [`DEVELOPMENT.md`](DEVELOPMENT.md) §7；[`architecture.md`](architecture.md) |

---

## 相关文档

- 接口与字段：[`api.md`](api.md) · 错误码与文案：[`errors.md`](errors.md)
- 开发规范 / 门禁 / Red Flags：[`DEVELOPMENT.md`](DEVELOPMENT.md) · 代理入口：[`AGENTS.md`](../AGENTS.md)
- 国际化与本地化：[`i18n.md`](i18n.md) · 可访问性：[`accessibility.md`](accessibility.md)
- 架构与决策：[`architecture.md`](architecture.md) · [`decisions/index.md`](decisions/index.md)
- 功能台账：[`FEATURES.md`](FEATURES.md) · 已知问题：[`KNOWN_ISSUES.md`](KNOWN_ISSUES.md)
