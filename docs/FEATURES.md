# s3client 功能大全（Features）

> 本文件是 s3client 的**单一事实来源**：产品能力总览 + 全部已完成修复 / 优化记录。
> 已把散落在各评估文档与 [`CHANGELOG.md`](../CHANGELOG.md) 中的「已实现 / 已修复 / 已完善」功能统一汇总于此；CHANGELOG 仍保留逐字发布历史。
>
> - 已知问题：[`KNOWN_ISSUES.md`](KNOWN_ISSUES.md)（缺陷 / 阻塞 / 技术债） · 迭代方向：[`ROADMAP.md`](ROADMAP.md) §三 · 发版历史：[`CHANGELOG.md`](../CHANGELOG.md) · 综合评估：[`archive/assessment.md`](archive/assessment.md)
> - 接口细节：[`api.md`](api.md) · 错误约定：[`errors.md`](errors.md) · 开发规范：[`DEVELOPMENT.md`](DEVELOPMENT.md) · 安全设计：[`threat-model.md`](threat-model.md) · Nginx 部署：[`deploy/nginx/README.md`](../deploy/nginx/README.md)
>
> 最后更新：2026-10-16（`v1.0.0` 之后的 Unreleased 区间）。**逐批次索引见下方[§二](#二已完成修复与优化)（8 个主题区块 · 104 行逐批索引）**，逐批明细证据表见 [`BATCHES.md`](BATCHES.md)（本页头不再重复枚举，避免同一份清单两处维护）；最近批次：§CR（O1–O3 闭环 + 门禁全量实跑全绿 + `changelog_tag` 门禁克隆口径校准）· §CS（安全评审 R2a/R2b/R3a + CSP 配置校验 + 限流可观测收口）· §CT（作用域对迁移 / 计划的账号引用 + 计划列表裁剪）· §CU（防御纵深：nginx / GitLab Node / 下载名）· §CV（对抗性渗透测试 High 收口：作用域零引用 fail-closed + 账号写入面 + body 解析器归一）· §CW（`prefixes` 必须与 `accounts` 同时声明）· **§CX**（2026-10-16 渗透测试 **P-3..P-15** 收口 + SAST 台账刷新为 20 条）· **§CY**（2026-10-16 大对象迁移无段缓冲直传与同端点分段落拷 ADR-014：`UploadPart` 显式长度 + 段级 Range 重试 + `UploadPartCopy`，#63 收窄至「大小未知」路径）· **§CZ**（2026-10-16 未知大小路径单对象上限 640GB → 1024GB：段长 64MiB → 105MiB + 容器预算 512M → 1G，#63 闭环移除）。

## 目录

- [一、产品功能](#一产品功能)
  - [1. 账号管理](#1-账号管理) · [2. 存储桶与桶属性](#2-存储桶与桶属性) · [3. 对象浏览与检索](#3-对象浏览与检索)
  - [4. 上传](#4-上传) · [5. 下载与预览](#5-下载与预览) · [6. 对象操作](#6-对象操作)
  - [7. 版本与回收站](#7-版本与回收站) · [8. 迁移与增量同步](#8-迁移与增量同步)
  - [9. 存储驱动与数据安全](#9-存储驱动与数据安全) · [10. 服务端安全与鉴权](#10-服务端安全与鉴权)
  - [11. API 与契约](#11-api-与契约) · [12. 前端体验与无障碍](#12-前端体验与无障碍)
  - [13. 桌面端](#13-桌面端) · [14. 部署、CI 与工程化](#14-部署ci-与工程化)
- [二、已完成修复与优化](#二已完成修复与优化)
  - 1. 建档、评估与历史台账 · 2. 稳定版修复与 KNOWN_ISSUES 收口 · 3. 审查与全仓评审处置 · 4. ROADMAP 功能与能力收口
  - 5. 文档、导航与代理治理 · 6. 门禁与质量机械化 · 7. 安全、供应链与加固 · 8. 工具链与三端体验优化
  - 逐批一行索引（§A–§CZ 共 104 批），明细证据表见 [`BATCHES.md`](BATCHES.md)
- [三、质量与覆盖率现状](#三质量与覆盖率现状)

---

## 一、产品功能

### 1. 账号管理
| 能力 | 说明 |
|---|---|
| 账号 CRUD | 增删改查；ID 由服务端生成，忽略客户端提交的 id；`Create` 存防御性副本 |
| 连通性测试 | `HeadBucket` 探测，返回 `ok` / `error` |
| 预览桶（不落库） | 用临时凭据 `ListBuckets`，不写入存储 |
| 服务商预设 | 按「兼容 / 国内 / 国外」分组（MinIO、阿里 OSS、腾讯 COS、华为 OBS、火山 TOS、AWS、R2、Wasabi、B2、Spaces 等） |
| 连接参数 | endpoint / publicEndpoint / region / AK / SK / bucket / pathStyle / useSSL |
| 密钥不出口 | 响应为 `AccountView`：不包含 `secretKey`，仅 `secretSet: boolean`；请求侧仍以 `secretKey` 提交 |

### 2. 存储桶与桶属性
| 能力 | 说明 |
|---|---|
| 桶列表 / 创建 / 删除 | 含桶名严格校验（长度、首尾字符、连续 `..`/`.-`/`-.`） |
| 桶属性 | 区域、创建时间、版本控制状态 |
| 版本控制 | 一键开启 / 暂停 |
| 生命周期 | 前缀过期删除规则读写 |
| 服务端加密（SSE） | 读写 / 开关 |
| CORS 规则 | 读写 / 开关 |
| 静态网站托管 | 读写 / 开关 |
| 桶策略 | 可视化编辑器（Statement 表单 + 4 模板 + 实时 JSON 预览）+ 原始 JSON 回退 |
| 桶标签 | 读写 / 开关 |

### 3. 对象浏览与检索
| 能力 | 说明 |
|---|---|
| 列表分页 | `ListObjectsV2`，「加载更多」追加 |
| 目录浏览 | 前缀 + 分隔符（默认 `/`）；20 万行虚拟滚动（窗口化渲染） |
| 对象详情 | 大小 / ETag / Content-Type / 存储类型 / 元数据 |
| 存储类型 | 列表 / 详情 / 版本中展示并一键切换（`CopyObject` 副本到自身） |
| 新建文件夹 / 重命名 / 移动 | 前缀级操作 |

### 4. 上传
| 能力 | 说明 |
|---|---|
| 小文件直传 | 服务端 v4 签名 PUT URL，浏览器直传 S3（2 路并发、进度、失败重试） |
| 大文件分段 | `≥100MB` 自动切 10MB/段、4 路并发直传，任一段失败即 abort 清理 |
| 上传队列 | 面板与对象区共享同一状态机；支持取消 / 重试；`cancelled` 终态不自动重启 |
| 前缀追加 | 上传时可给 key 追加前缀 |

### 5. 下载与预览
| 能力 | 说明 |
|---|---|
| 一键下载 | 短时效签名 GET URL |
| 分享链接 | 1 小时签名 URL 复制；批量复制链接并行 presign（`Promise.allSettled`） |
| 安全代理 | 下载 / 预览走服务端代理（三模式 + 类型拒渲染 + sandbox），key 拒绝控制字符 |
| ZIP 打包 | 批量下载流式打包（`io.Copy` 零整对象缓冲，滚动写超时） |
| blob 回收 | `revokeObjectURL` 延迟 60s，避免中断尚未开始的下载 |

### 6. 对象操作
| 能力 | 说明 |
|---|---|
| 删除 | 单个 / 批量（`≤1000` 一次，超限 400）/ 前缀递归 |
| 复制 / 移动 | 跨桶、文件夹递归；同桶文件走 `CopyObject`；移动 = 复制成功后删源 |
| 设置 HTTP 头 | `set-headers`（Cache-Control / Content-Disposition 等） |
| ACL | 读 / 写，私有 ↔ 公共读，复制公开访问链接 |
| 标签 | 读 / 写 / 清空，键值行编辑 |
| 批量改元数据 | ACL / 标签（替换或清空）/ 存储类型；4 路有界并发，进度 + 失败明细；全空标签禁止提交 |
| 批量 key 上限 | `BATCH_META_MAX_KEYS=10000`，超限抛错 |

### 7. 版本与回收站
| 能力 | 说明 |
|---|---|
| 版本列表 | `ListObjectVersions`，含删除标记；`keyMarker`/`versionIdMarker` 翻页（≤20 页），仍截断时显式提示 |
| 删除指定版本 | `DeleteObject` 带 `versionId` |
| 版本回滚 | 恢复某版本为当前 |
| 版本比较 / 详情 | 选两个内容版本做差异比对 |
| 回收站 | 独立菜单，列出全部删除标记，一键还原（撤销删除）/ 彻底清除（永久删除该 key 全部版本） |

### 8. 迁移与增量同步
| 能力 | 说明 |
|---|---|
| 同 endpoint 复制 | 服务端 `CopyObject` |
| 跨 endpoint 复制 | `GetObject` → `PutObject` 流式转发（保留 Content-Type 与元数据，64MB 分段有界复用） |
| 单 / 批量迁移 | 逐 key 执行，失败继续并汇总；异步任务 + SSE 进度 |
| 增量同步 | `POST /api/migrate/sync`，按 `etag`（默认）/ `size_mtime` / `always` 比对，仅复制差异对象 |
| 前缀追加 | 迁移时可给目标 key 追加前缀 |
| 进度 | `RunBatch` 无缓冲 results 通道，边收边回调（内存 O(workers)） |

### 9. 存储驱动与数据安全
| 能力 | 说明 |
|---|---|
| 三驱动 | `json`（默认）/ `sqlite` / `encrypted`，统一 `store.Open` 入口 |
| 共享实现 | `json` 与 `encrypted` 统一为 `Store` + 单一 `storeCodec`（strict 区分），复用 `fileStore`（锁 / CRUD / 回滚 / 快照单点实现） |
| 落盘加密 | `S3C_STORE_KEY` → Argon2id + AES-256-GCM，当前写入格式 `S3C3`（Argon2id 参数随文件头保存，故可在不破坏既有库的前提下调参）；兼容读旧 `S3C2` |
| 原子写 | 临时文件 + 写后 rename；写前清残骸（`O_EXCL`）；0600 权限 |
| 兼容性 | `json` 驱动同时读明文与历史 S3C2 文件（permissive，写盘换新盐）；`encrypted` 严格模式只接受加密信封（S3C3 当前 / S3C2 兼容），盐建时随机并随文件复用 |
| 失败回滚 | `Create`/`Update`/`Delete` 持久化失败回滚内存，避免内存/磁盘漂移 |

### 10. 服务端安全与鉴权
| 能力 | 说明 |
|---|---|
| 鉴权 | Bearer 多 token（`S3C_TOKEN` 逗号分隔，支持轮换）；凭证常量时间比较，scheme 大小写不敏感（RFC 7235） |
| 启动校验 | 短 token（< 16 字符）拒绝启动；非回环未设 token 拒绝启动 |
| CSRF 双防 | CORS 白名单外 Origin 直接 403 + `readJSON` 要求 **非空** `Content-Type` 必须为 `application/json`（缺省放行，仍受 JSON 解码器约束）触发预检 |
| SSRF 双校验 | 创建时校验 + 拨号时二次校验（防 DNS rebinding）；禁重定向；`Proxy=nil` 防环境变量代理绕过 |
| 响应头 | CSP（默认 `connect-src 'self'`，`S3C_CSP_CONNECT_SRC` 可放宽）+ 安全响应头 |
| 限速 | 按客户端 IP 限速；`X-Forwarded-For` 仅当直连对端命中 `S3C_TRUSTED_PROXIES` 时采信首段（默认空 = 不信任，防直连伪造绕过） |
| 端点门控 | `/api/openapi.json`（`withOpenAPIGate`）与 `/api/metrics`（`withMetricsGate`）默认 404 |
| 输入防护 | 路径遍历 / 控制字符 key 拒绝；user metadata 边界 400（非 500） |
| 请求 ID | 回显 `X-Request-ID` 仅限 ≤128 可见 ASCII，否则服务端生成，防日志/响应头注入 |
| 错误脱敏 | `UserMessage` 防腐层统一映射，sentinel（`ErrObjectTooLarge` / `ErrSourceDeleteFailed`）替代文案匹配 |
| 账号密钥不出口 | 账号响应为 `AccountView`：**不包含 `secretKey`**，仅 `secretSet: boolean` 表示是否已设置；存储层 `Sanitized()` 占位不再进入 HTTP 响应 |

### 11. API 与契约
| 能力 | 说明 |
|---|---|
| REST 端点 | **84** 个 `/api/*` 端点（含 health/metrics/openapi.json） |
| OpenAPI 3.0.3 | `/api/openapi.json` 自动生成，按域登记（accounts / buckets / bucket-settings / objects / object-meta / multipart / versions / trash / migrate / system） |
| 契约测试 | `routes.go` ↔ 规范双向一致、operation 完整性、路径参数、`$ref` 可解析、`MarshalJSON` 确定性 |
| 错误约定 | 统一 JSON 错误体，见 [`errors.md`](errors.md) |
| 静态托管 | SPA fallback，`/api/*` 未匹配直接 404（不返回 HTML） |

### 12. 前端体验与无障碍
| 能力 | 说明 |
|---|---|
| 技术栈 | Vue 3 + Vite + TS；`dependencies` 仅 `vue` |
| 国际化 | zh / en 按域拆分（`i18n/messages/*`） |
| 主题 | 明暗主题切换 |
| 快捷键 | 全局键盘 + 面板激活守卫（KeepAlive 下不误触） |
| 竞态防护 | `loadSeq` / `detailSeq` 序号守卫 + `AbortController`；SSE 卸载主动 abort |
| 无障碍 | `label` / `aria-label` / `aria-live` / `role="alert"` / 焦点陷阱 |
| 预览管线 | 三模式代理 + 类型拒渲染 + `sandbox` |
| 弹窗 | `ModalDialog` footer 插槽、z-index 分层、右键菜单溢出滚动 |
| 加载态 | 骨架 / 进度条 / 错误横幅与重试 |

### 13. 桌面端
| 能力 | 说明 |
|---|---|
| Tauri 2 | B/S 架构，**无 IPC**，前后端全 HTTP |
| 权限收敛 | `withGlobalTauri` 关闭 + capability 裁剪 |
| 分发 | CI 交叉构建 `.exe`(NSIS) / `.deb` / `.dmg`，挂 GitHub Release，附 SHA256SUMS |

### 14. 部署、CI 与工程化
| 能力 | 说明 |
|---|---|
| 容器 | Alpine 运行镜像；非 root；healthcheck；compose base / prod / tls 三套 |
| 安全默认 | 回环发布、token 强制注入、server/rustfs/nginx 显式内存上限（nginx 128M） |
| CI | `gofmt` / `go vet` / `go test -race` / 覆盖率门禁 / Docker 构建 / Trivy / Playwright E2E |
| CI 双平台 | 同一套门禁同时落 GitHub Actions 与 GitLab CI（`.gitlab-ci.yml`）：server / web / docker / desktop / desktop-build（手动）/ rustfs-e2e / playwright-e2e 逐 job 对应，命令与阈值一致；`release-desktop.yml` 有意不镜像（GitHub Release 专属）。本地用 `make gcl` / `make gcl-docker`（pin `gitlab-ci-local@4.75.1`，`.gitlab-ci-local-env` 默认挂 docker.sock；Trivy DB 可用 `TRIVY_DB_REPOSITORY` / `.gitlab-ci-local-variables.yml.example`） |
| 供应链 | GitHub Actions 全部 pin SHA；Trivy 用官方镜像 `aquasec/trivy`（`aquasecurity/trivy` 仅存在于 ghcr.io，写错会以 exit 125 失败而非漏洞失败）；发布产物附 SHA256SUMS |
| 本地 | Makefile（`test` / `test-cover` / `web-test` / `vet` / `install-hooks` / `gcl` / `gcl-list` / `gcl-docker`）；pre-commit（gofmt + vet + 前端 typecheck） |
| 版本 | `scripts/release-version.sh` 同步 8 处版本号 |

---

## 二、已完成修复与优化

> 状态图例：✅ 已完成 · ➖ 已评估 / 无需改动 · ⏳ 说明见备注。
>
> 本节是 **104 个批次（§A–§CZ）的区块索引**：每批一行（批次 · 日期 · 要点）。逐批明细证据表（`file:line` 改动点、门禁实跑数字、测试与文档同步记录）已外移到 [`BATCHES.md`](BATCHES.md) 的**同字母**章节——历史引用「`FEATURES.md §X`」即指本节对应索引行，批次互引（如「见 §S」）在 [`BATCHES.md`](BATCHES.md) 内按字母解析。
>
> 历史批次头注中的「仍开放 / 仍未处置」均为**写作时点**状态、已带日期锚（2026-10-16 复核：见 §P / §Q / §CA 三行）。

### 1. 建档、评估与历史台账

| 批次 | 日期 | 要点 |
|---|---|---|
| [§A](BATCHES.md#a-2026-09-建档批次增量store-去重--openapi-契约--长尾) | 2026-09 | 建档批次增量（store 去重 / OpenAPI 契约 / 长尾） |
| [§B](BATCHES.md#b-双存储驱动去重明细) | 2026-09 | 双存储驱动去重明细 |
| [§C](BATCHES.md#c-全方位评估-58-项2026-04-19-全部落地) | 2026-04-19 | 全方位评估 58 项（2026-04-19 全部落地） |
| [§D](BATCHES.md#d-v100-rc1-评估-21-项含后续补修) | 2026-09-02 | v1.0.0-rc1 评估 21 项（含后续补修） |
| [§E](BATCHES.md#e-optional--nit-长尾采纳情况v100-rc1-评估) | 2026-09-02 | Optional / Nit 长尾采纳情况（v1.0.0-rc1 评估） |
| [§F](BATCHES.md#f-历史版本功能与修复全量台账010--v100-rc1) | 2026-08-22 → 2026-09-02 | 历史版本功能与修复全量台账（0.1.0 → v1.0.0-rc1） |
| [§G](BATCHES.md#g-已记录的-unreleased-项已完成) | — | 已记录的 Unreleased 项（已完成） |

### 2. 稳定版修复与 KNOWN_ISSUES 收口

| 批次 | 日期 | 要点 |
|---|---|---|
| [§H](BATCHES.md#h-2026-09-16-评估-p0-发布阻塞修复) | 2026-09-16 | 评估 P0 发布阻塞修复 |
| [§I](BATCHES.md#i-2026-09-16-p1-稳定版门槛修复) | 2026-09-16 | P1 稳定版门槛修复 |
| [§J](BATCHES.md#j-2026-09-16-p2-加固修复第一批) | 2026-09-16 | P2 加固修复（第一批） |
| [§K](BATCHES.md#k-2026-09-17-p2-加固修复第二批) | 2026-09-17 | P2 加固修复（第二批） |
| [§L](BATCHES.md#l-2026-09-17-目录迁移后的可移除项清理) | 2026-09-17 | 目录迁移后的可移除项清理 |
| [§N](BATCHES.md#n-2026-09-19-明文落盘启动告警与旧编号引用清理) | 2026-09-19 | 明文落盘启动告警与旧编号引用清理 |
| [§O](BATCHES.md#o-2026-09-19-风险登记集中处置r1--r2--r4--r6--r7--r8) | 2026-09-19 | 风险登记集中处置（R1 / R2 / R4 / R6 / R7 / R8） |
| [§AB](BATCHES.md#ab-2026-09-28-known_issues-6063-收口前端测试拆分--sameendpoint-纳入-usessl--删除编排下沉--640gb-上限证据补齐) | 2026-09-28 | KNOWN_ISSUES #60–#63 收口（前端测试拆分 / `SameEndpoint` 纳入 `useSSL` / 删除编排下沉 / 640GB 上限证据补齐） |
| [§AC](BATCHES.md#ac-2026-09-28-developmentmd-7-历史技术债收口h1--s7--d5-三条均已闭环) | 2026-09-28 | `DEVELOPMENT.md` §7 历史技术债收口（H1 / S7 / D5 三条均已闭环） |
| [§AP](BATCHES.md#ap-2026-09-29-收口-known_issues-67--68可访问性三处--nginx-跨层日志关联) | 2026-09-29 | 收口 KNOWN_ISSUES #67 / #68：可访问性三处 + nginx 跨层日志关联 |
| [§BK](BATCHES.md#bk-2026-09-30-known_issues-70-闭环normalizeendpoint-幂等修复推翻--决策) | 2026-09-30 | KNOWN_ISSUES #70 闭环：`NormalizeEndpoint` 幂等修复（推翻 ➖ 决策） |
| [§BO](BATCHES.md#bo-2026-10-08-known_issues-71-闭环仓库-slug-统一为-githubcomweilai1949s3client) | 2026-10-08 | KNOWN_ISSUES #71 闭环：仓库 slug 统一为 `github.com/weilai1949/s3client` |
| [§BP](BATCHES.md#bp-2026-10-08-known_issues-71-第二批产品名-s3clinet--s3client-全量统一) | 2026-10-08 | KNOWN_ISSUES #71 第二批：产品名 `s3clinet` → `s3client` 全量统一 |
| [§CA](BATCHES.md#ca-2026-10-09-known_issues-73--80--81--83-闭环评审-optional-项收口) | 2026-10-09 | KNOWN_ISSUES #73 / #80 / #81 / #83 闭环（评审 Optional 项收口）；该批头注时点锚：同批 #72 / #74–#79 **在写作时点（2026-10-09）仍开放**，同日由 §CC / §CD 闭环 |
| [§CB](BATCHES.md#cb-2026-10-09--10-10-known_issues-82-闭环r10-残留make-check-缺项--e2e-路径过滤--development-漂移) | 2026-10-09 / 10-10 | KNOWN_ISSUES #82 闭环（R10 残留：`make check` 缺项 / E2E 路径过滤 / DEVELOPMENT 漂移） |
| [§CC](BATCHES.md#cc-2026-10-09-known_issues-72--74--75--77--78-闭环评审-optional-项第二批) | 2026-10-09 | KNOWN_ISSUES #72 / #74 / #75 / #77 / #78 闭环（评审 Optional 项第二批） |
| [§CD](BATCHES.md#cd-2026-10-09-known_issues-76--79-闭环评审-optional-项第三批o1o12-归零) | 2026-10-09 | KNOWN_ISSUES #76 / #79 闭环（评审 Optional 项第三批；O1–O12 归零） |

### 3. 审查与全仓评审处置

| 批次 | 日期 | 要点 |
|---|---|---|
| [§P](BATCHES.md#p-2026-09-19-分支状态审查-a1a3-处置前端-api-拆分--死代码门禁--测试接缝) | 2026-09-19 | 分支状态审查 A1–A3 处置（前端 API 拆分 / 死代码门禁 / 测试接缝）；该批头注时点锚：P0/P1 缺陷**在写作时点（2026-09-19）仍未处置**，其后由 §Q / §R / §S 闭环 |
| [§Q](BATCHES.md#q-2026-09-19-分支状态审查-p0-处置5-项全部闭环) | 2026-09-19 | 分支状态审查 P0 处置（5 项全部闭环）；该批头注时点锚：剩余 P1/P2 **在写作时点（2026-09-19）仍开放**，其后由 §S / §T / §U 闭环 |
| [§R](BATCHES.md#r-2026-09-19-分支状态审查-三-正确性处置后端-b3b11--前端-f2f10) | 2026-09-19 | 分支状态审查 §三 正确性处置（后端 B3–B11 / 前端 F2–F10） |
| [§S](BATCHES.md#s-2026-09-19-分支状态审查-p1-处置92-全部闭环) | 2026-09-19 | 分支状态审查 P1 处置（§9.2 全部闭环） |
| [§T](BATCHES.md#t-2026-09-20-审查-93-p2-处置契约-2628--安全--供应链-2934) | 2026-09-20 | 审查 §9.3 P2 处置（契约 #26–#28 + 安全 / 供应链 #29–#34） |
| [§U](BATCHES.md#u-2026-09-22-审查-93-p2-收尾d1--d4--d5--d10--p3--r7--r9--r10--门禁质量) | 2026-09-22 | 审查 §9.3 P2 收尾（D1 / D4 / D5 / D10 + P3 + R7 / R9 / R10 + 门禁质量） |
| [§W](BATCHES.md#w-2026-09-22-审查-74-门禁盲区收尾响应门禁全量--query-读取口径--叙述性数字--导出测试符号) | 2026-09-22 | 审查 §7.4 门禁盲区收尾（响应门禁全量 + query 读取口径 + 叙述性数字 + 导出测试符号） |
| [§X](BATCHES.md#x-2026-09-22-审查-43维持项复核收尾门禁自身口径--path-参数门禁) | 2026-09-22 | 审查 §4.3「维持」项复核收尾（门禁自身口径 + path 参数门禁） |
| [§Y](BATCHES.md#y-2026-09-22-子审查复核异步任务契约统一--门禁口径补充) | 2026-09-22 | 子审查复核：异步任务契约统一 + 门禁口径补充 |
| [§AA](BATCHES.md#aa-2026-09-24-全仓代码审查处置2-critical--20-required-全清nit-3135-闭环3-项转登记1-项判定不成立) | 2026-09-24 | 全仓代码审查处置（2 Critical + 20 Required 全清；Nit 31/35 闭环、3 项转登记、1 项判定不成立） |
| [§AD](BATCHES.md#ad-2026-09-28-三路五轴复审后端-handlerservice--后端-s3wrapstoreconfig--前端-web本轮闭环-4-条另-15-条转登记-known_issuesmdknown_issuesmd-64) | 2026-09-28 | 三路五轴复审（后端 handler+service / 后端 s3wrap+store+config / 前端 web）——本轮闭环 4 条，另 15 条转登记 [`KNOWN_ISSUES.md`](KNOWN_ISSUES.md) #64 |
| [§AE](BATCHES.md#ae-2026-09-28-破坏性操作审计覆盖补齐64-security3-条先红后绿) | 2026-09-28 | 破坏性操作审计覆盖补齐（#64 Security，3 条先红后绿） |
| [§AF](BATCHES.md#af-2026-09-28-前端四条64sticky-error-两处--destdialog-并发--signing-死状态4-条先红后绿) | 2026-09-28 | 前端四条（#64）：sticky error 两处 / `DestDialog` 并发 / `signing` 死状态——4 条先红后绿 |
| [§AG](BATCHES.md#ag-2026-09-28-config--main-三条64显式-env-文件静默回退--关停超时无上界--预建数据目录不收紧3-条先红后绿) | 2026-09-28 | `config` / `main` 三条（#64）：显式 env 文件静默回退 / 关停超时无上界 / 预建数据目录不收紧——3 条先红后绿 |
| [§AH](BATCHES.md#ah-2026-09-28-s3wrap-两条64metadata-值控制字符-400500--idn-端点建得成却永远连不上2-条先红后绿) | 2026-09-28 | `s3wrap` 两条（#64）：metadata 值控制字符 400→500 / IDN 端点「建得成却永远连不上」——2 条先红后绿 |
| [§AI](BATCHES.md#ai-2026-09-28-前端另四条64代次守卫--追加重置滚动--loadingall-提前可点--桶列举共用标志4-条先红后绿) | 2026-09-28 | 前端另四条（#64）：代次守卫 / 追加重置滚动 / `loadingAll` 提前可点 / 桶列举共用标志——4 条先红后绿 |
| [§AJ](BATCHES.md#aj-2026-09-28-known_issues-64-闭环storeopen-的-json-分支不再丢弃入参-storekey) | 2026-09-28 | KNOWN_ISSUES #64 闭环：`store.Open` 的 json 分支不再丢弃入参 `storeKey` |
| [§BZ](BATCHES.md#bz-2026-10-09-全仓代码评审批次c1c2--r1r9-全修--o1o12-登记) | 2026-10-09 | 全仓代码评审批次：C1–C2 + R1–R9 全修 + O1–O12 登记 |
| [§CK](BATCHES.md#ck-2026-10-10-全仓代码质量评审结论-approve-三道-nit-8486-闭环--快照归档) | 2026-10-10 | 全仓代码质量评审（结论 Approve）+ 三道 Nit #84–#86 闭环 + 快照归档 |
| [§CQ](BATCHES.md#cq-2026-10-14-全仓代码质量评审结论-approvef1f7--a1a2--c2-闭环--c1-有意维持--o1o3-转登记) | 2026-10-14 | 全仓代码质量评审（结论 Approve）：F1–F7 + A1–A2 + C2 闭环 + C1 有意维持 + O1–O3 转登记 |
| [§CR](BATCHES.md#cr-2026-10-15-o1o3-闭环有测试环境实跑-changelog_tag-门禁克隆口径校准) | 2026-10-15 | O1–O3 闭环（有测试环境实跑）+ `changelog_tag` 门禁克隆口径校准 |

### 4. ROADMAP 功能与能力收口

| 批次 | 日期 | 要点 |
|---|---|---|
| [§M](BATCHES.md#m-2026-09-17-路线图-v100--v10x--v110-收口181011) | 2026-09-17 | 路线图 v1.0.0 / v1.0.x / v1.1.0 收口（#1–#8、#10、#11） |
| [§BJ](BATCHES.md#bj-2026-09-30-roadmap-19-文档可读性与流程机械化4-项落地已收尾) | 2026-09-30 | ROADMAP #19 文档可读性与流程机械化（4 项落地，**已收尾**） |
| [§BL](BATCHES.md#bl-2026-09-30-roadmap-18-可观测性补全五个缺失指标发射点--告警--仪表盘--文档同步) | 2026-09-30 | ROADMAP #18 可观测性补全：五个缺失指标（发射点 + 告警 + 仪表盘 + 文档同步） |
| [§BM](BATCHES.md#bm-2026-09-30-roadmap-17-可访问性补强与自动化扫描5-项全部落地已收尾) | 2026-09-30 | ROADMAP #17 可访问性补强与自动化扫描（5 项全部落地，**已收尾**） |
| [§BN](BATCHES.md#bn-2026-10-01-roadmap-10-openapi--前端类型--客户端代码生成schema-first--diff-门禁已收尾) | 2026-10-01 | ROADMAP #10 OpenAPI → 前端类型 / 客户端代码生成（schema-first + diff 门禁，**已收尾**） |
| [§BQ](BATCHES.md#bq-2026-10-08-roadmap-8-大文件体验上传断点续传--下载并行分段) | 2026-10-08 | ROADMAP #8 大文件体验：上传断点续传 + 下载并行分段 |
| [§BR](BATCHES.md#br-2026-10-08-roadmap-11-零依赖-otlp-tracingw3c-traceparent--otlphttp-json默认关闭) | 2026-10-08 | ROADMAP #11 零依赖 OTLP tracing（W3C traceparent + OTLP/HTTP JSON，默认关闭） |
| [§BS](BATCHES.md#bs-2026-10-08-roadmap-13-token-作用域与最小权限s3c_token_scopes) | 2026-10-08 | ROADMAP #13 Token 作用域与最小权限（`S3C_TOKEN_SCOPES`） |
| [§BT](BATCHES.md#bt-2026-10-08-roadmap-5-s3-新协议特性条件写--端到端校验和--object-lock) | 2026-10-08 | ROADMAP #5 S3 新协议特性：条件写 / 端到端校验和 / Object Lock |
| [§BU](BATCHES.md#bu-2026-10-08-roadmap-6-计划任务cron-定时增量备份) | 2026-10-08 | ROADMAP #6 计划任务：cron 定时增量备份 |
| [§BV](BATCHES.md#bv-2026-10-08-roadmap-7-finops-存储分析与成本看板) | 2026-10-08 | ROADMAP #7 FinOps 存储分析与成本看板 |

### 5. 文档、导航与代理治理

| 批次 | 日期 | 要点 |
|---|---|---|
| [§AL](BATCHES.md#al-2026-09-29-对照通用-agentsmd-模板补齐代理治理与配置-ssot新增-4-个文件--改-8-处引用) | 2026-09-29 | 对照通用 AGENTS.md 模板补齐代理治理与配置 SSOT（新增 4 个文件 / 改 8 处引用） |
| [§AM](BATCHES.md#am-2026-09-29-文档命名规则收敛为元文档大写--内容文档小写docs-下-6-个元文档恢复大写引用同步-607-处) | 2026-09-29 | 文档命名规则收敛为「元文档大写 / 内容文档小写」（`docs/` 下 6 个元文档恢复大写，引用同步 607 处） |
| [§AN](BATCHES.md#an-2026-09-29-文档覆盖矩阵收口11-个新文档--3-项供应链门禁--机器可读契约--性能基线) | 2026-09-29 | 文档覆盖矩阵收口（11 个新文档 + 3 项供应链门禁 + 机器可读契约 + 性能基线） |
| [§AR](BATCHES.md#ar-2026-09-29-补齐运维事故复盘模板10-层文档基线第-6-层缺口收口) | 2026-09-29 | 补齐运维事故复盘模板（10 层文档基线第 6 层缺口收口） |
| [§AS](BATCHES.md#as-2026-09-29-文档失真收口11-处文档与实现--自身不一致-dependabot-路径失效) | 2026-09-29 | 文档失真收口（11 处「文档与实现 / 自身不一致」+ Dependabot 路径失效） |
| [§AT](BATCHES.md#at-2026-09-29-文档缺口收口链接锚点门禁--文档登记表--adr-模板--子树-agents--隐私声明) | 2026-09-29 | 文档缺口收口（链接/锚点门禁 + 文档登记表 + ADR 模板 + 子树 AGENTS + 隐私声明） |
| [§AV](BATCHES.md#av-2026-09-29-元信息--导航收口docs-落地页--导航覆盖门禁--gitattributes) | 2026-09-29 | 元信息 / 导航收口（docs 落地页 + 导航覆盖门禁 + `.gitattributes`） |
| [§AX](BATCHES.md#ax-2026-09-29-ai-时代层收口ai-治理的机械保证--copilot-指针入口) | 2026-09-29 | AI 时代层收口（AI 治理的机械保证 + Copilot 指针入口） |
| [§AY](BATCHES.md#ay-2026-09-30-客户端支持矩阵收口10-层基线第-6-层用户文档唯一缺口) | 2026-09-30 | 客户端支持矩阵收口（10 层基线第 6 层「用户文档」唯一缺口） |
| [§AZ](BATCHES.md#az-2026-09-30-adr-覆盖补足8-篇新-adr--取舍表覆盖门禁) | 2026-09-30 | ADR 覆盖补足（8 篇新 ADR + 取舍表覆盖门禁） |
| [§BB](BATCHES.md#bb-2026-09-30-ai-产出效果证据层收口黄金任务集--评分卡--贡献度量) | 2026-09-30 | AI 产出效果证据层收口（黄金任务集 / 评分卡 / 贡献度量） |
| [§BC](BATCHES.md#bc-2026-09-30-英文文档入口收口docsenindexmd) | 2026-09-30 | 英文文档入口收口（docs/en/index.md） |
| [§BD](BATCHES.md#bd-2026-09-30-导航收口残留命名约定两处分叉--llmstxt-目录摘要--readme-ai-入口--机械门禁) | 2026-09-30 | 导航收口残留（命名约定两处分叉 + `llms.txt` 目录摘要 + README AI 入口 + 机械门禁） |
| [§BE](BATCHES.md#be-2026-09-30-状态台账-69-收口changelog--git-tag-一致性--机械门禁) | 2026-09-30 | 状态台账 #69 收口（CHANGELOG ↔ git tag 一致性 + 机械门禁） |
| [§BF](BATCHES.md#bf-2026-09-30-docs-内待修项收口文档失真--对比度静态核查--改进项登记-ssot) | 2026-09-30 | docs 内待修项收口（文档失真 + 对比度静态核查 + 改进项登记 SSOT） |
| [§BG](BATCHES.md#bg-2026-09-30-ai-时代文档补强六路并行p0-效果证据--p1-机器可读--p2-元信息) | 2026-09-30 | AI 时代文档补强（六路并行：P0 效果证据 / P1 机器可读 / P2 元信息） |
| [§BH](BATCHES.md#bh-2026-09-30-门禁实跑回填与-bg-结构归位未实跑标记清零) | 2026-09-30 | 门禁实跑回填与 §BG 结构归位（「未实跑」标记清零） |
| [§BI](BATCHES.md#bi-2026-09-30-第二轮收口roadmap--行移出--au-过期注记--评估残留缺口补登记--四-基线复测) | 2026-09-30 | 第二轮收口（ROADMAP ✅ 行移出 / §AU 过期注记 / 评估残留缺口补登记 / §四 基线复测） |
| [§CG](BATCHES.md#cg-2026-10-10-两处文档状态漂移收口悬空-roadmap-11-引用--trace-观测面自相矛盾新增状态陈述门禁) | 2026-10-10 | 两处文档状态漂移收口（悬空 `ROADMAP #11` 引用 + trace 观测面自相矛盾；新增状态陈述门禁） |
| [§CI](BATCHES.md#ci-2026-10-10-全-docs-通读问题清单收口表渲染断裂--过期数字与悬空指针--安全契约缺失--通用状态码接线--三道门禁) | 2026-10-10 | 全 docs 通读问题清单收口（表渲染断裂 / 过期数字与悬空指针 / 安全契约缺失 / 通用状态码接线 + 三道门禁） |
| [§CJ](BATCHES.md#cj-2026-10-10-交接快照-5未做清单收口14-项5-道新门禁--9-处文档结构--adr-体例) | 2026-10-10 | 交接快照 §5「未做清单」收口（14 项：5 道新门禁 + 9 处文档结构 + ADR 体例） |
| [§CO](BATCHES.md#co-2026-10-12-docs-复盘快照归档--其沿用的-14-项建议逐项复核) | 2026-10-12 | docs 复盘快照归档 + 其沿用的 14 项建议逐项复核 |

### 6. 门禁与质量机械化

| 批次 | 日期 | 要点 |
|---|---|---|
| [§V](BATCHES.md#v-2026-09-22-真实后端--真实-rustfs-浏览器联调known_issues-37) | 2026-09-22 | 真实后端 + 真实 RustFS 浏览器联调（KNOWN_ISSUES #37） |
| [§Z](BATCHES.md#z-2026-09-24-roadmap-三-3-死代码纪律收口前后端两道零生产引用导出门禁) | 2026-09-24 | roadmap §三 #3 死代码纪律收口：前后端两道「零生产引用」导出门禁 |
| [§AO](BATCHES.md#ao-2026-09-29-收口-known_issues-65前端死代码门禁改用-ts-ast-判定引用注释--字符串不再算引用) | 2026-09-29 | 收口 KNOWN_ISSUES #65：前端死代码门禁改用 **TS AST** 判定引用（注释 / 字符串不再算引用） |
| [§BX](BATCHES.md#bx-2026-10-09-e2e-real-seedaccountandbucket-失败泄漏账号桶修复含建桶失败快诊--限速节流) | 2026-10-09 | e2e-real `seedAccountAndBucket` 失败泄漏账号/桶修复（含建桶失败快诊 + 限速节流） |
| [§BY](BATCHES.md#by-2026-10-09-s3wrap-e2e-清桶强删路径根治-object-lock-锁定版本导致的共享实例残留桶泄漏) | 2026-10-09 | s3wrap E2E 清桶强删路径（根治 Object Lock 锁定版本导致的共享实例残留桶泄漏） |
| [§CE](BATCHES.md#ce-2026-10-10-根-envexample--compose-透传面双向一致新增门禁-readme-配置项计数订正) | 2026-10-10 | 根 `.env.example` ⇔ compose 透传面双向一致（新增门禁）+ README 配置项计数订正 |
| [§CH](BATCHES.md#ch-2026-10-10-a-组六项可机械优化收口inode-指标内置--三道门禁--网格态-a11y--明文端点告知) | 2026-10-10 | A 组六项「可机械优化」收口（inode 指标内置 + 三道门禁 + 网格态 a11y + 明文端点告知） |

### 7. 安全、供应链与加固

| 批次 | 日期 | 要点 |
|---|---|---|
| [§AQ](BATCHES.md#aq-2026-09-29-供应链收口桌面-sbom--cosign-告警规则落地--账号库-schema--66-闭环) | 2026-09-29 | 供应链收口（桌面 SBOM + cosign）+ 告警规则落地 + 账号库 Schema + #66 闭环 |
| [§AU](BATCHES.md#au-2026-09-29-接口契约表达鉴权文档级-security--逐端点豁免--tags-分组) | 2026-09-29 | 接口契约表达鉴权（文档级 `security` + 逐端点豁免 + `tags` 分组） |
| [§AW](BATCHES.md#aw-2026-09-29-安全与供应链收口自动生成的许可证清单--依赖覆盖门禁--产物核验指南) | 2026-09-29 | 安全与供应链收口（自动生成的许可证清单 + 依赖覆盖门禁 + 产物核验指南） |
| [§BA](BATCHES.md#ba-2026-09-30-供应链收口openssf-scorecard--pr-依赖审查) | 2026-09-30 | 供应链收口（OpenSSF Scorecard + PR 依赖审查） |
| [§BW](BATCHES.md#bw-2026-10-09-go-工具链-1266--126910-个可达-stdlib-漏洞归零-e2e-real-用例幂等化) | 2026-10-09 | Go 工具链 1.26.6 → 1.26.9（10 个可达 stdlib 漏洞归零）+ e2e-real 用例幂等化 |
| [§CP](BATCHES.md#cp-2026-10-12-晚pnpm-audit-官方-registry-实跑清零12--0-a11y-扫描盲区补齐413-三态-dev-strictport) | 2026-10-12 | `pnpm audit` 官方 registry 实跑清零（12 → 0）+ a11y 扫描盲区补齐（§4.13 三态）+ dev `strictPort` |
| [§CS](BATCHES.md#cs-2026-10-15-安全评审-r2a--r2b--r3a--两项加固收口) | 2026-10-15 | 安全评审 R2a / R2b / R3a + 两项加固收口 |
| [§CT](BATCHES.md#ct-2026-10-15-安全评审残留收口作用域对迁移--计划的账号引用--计划列表裁剪) | 2026-10-15 | 安全评审残留收口：作用域对迁移 / 计划的账号引用 + 计划列表裁剪 |
| [§CU](BATCHES.md#cu-2026-10-15-安全评审防御纵深收口nginx--gitlab-node--下载名) | 2026-10-15 | 安全评审防御纵深收口：nginx / GitLab Node / 下载名 |
| [§CV](BATCHES.md#cv-2026-10-16-对抗性渗透测试-high-收口作用域零引用-fail-closed--账号写入面--body-解析器归一) | 2026-10-16 | 对抗性渗透测试 High 收口：作用域零引用 fail-closed + 账号写入面 + body 解析器归一 |
| [§CW](BATCHES.md#cw-2026-10-16-收口账号维度残留prefixes-必须与-accounts-同时声明) | 2026-10-16 | 收口账号维度残留：`prefixes` 必须与 `accounts` 同时声明 |
| [§CX](BATCHES.md#cx-2026-10-16-渗透测试-p-3p-15-收口端点凭据--csp--zip--cors--密钥注入--镜像与容器加固--sast-台账) | 2026-10-16 | 渗透测试 P-3..P-15 收口（端点凭据 / CSP / ZIP / CORS / 密钥注入 / 镜像与容器加固 / SAST 台账） |

### 8. 工具链与三端体验优化

| 批次 | 日期 | 要点 |
|---|---|---|
| [§AK](BATCHES.md#ak-2026-09-29-前端测试与宿主-node_env-解耦修复-33-文件--246-例全红新增-2-例守卫) | 2026-09-29 | 前端测试与宿主 `NODE_ENV` 解耦（修复 33 文件 / 246 例全红，新增 2 例守卫） |
| [§CF](BATCHES.md#cf-2026-10-10-make-dev-受管-web-pid-指向修复--启动流程去自动-tidy新增门禁) | 2026-10-10 | `make dev` 受管 web PID 指向修复 + 启动流程去自动 tidy（新增门禁） |
| [§CL](BATCHES.md#cl-2026-10-12-appsweb-页面-ui-与交互优化批次h1h3--m1m3--l1l2--d1) | 2026-10-12 | apps/web 页面 UI 与交互优化批次（H1–H3 / M1–M3 / L1–L2 + D1） |
| [§CM](BATCHES.md#cm-2026-10-12-appsdesktop-优化评审批次d1--o1o6--g1) | 2026-10-12 | apps/desktop 优化评审批次（D1 + O1–O6 + G1） |
| [§CN](BATCHES.md#cn-2026-10-12-appsserver-优化评审-s1s5-闭环批次) | 2026-10-12 | apps/server 优化评审 S1–S5 闭环批次 |
| [§CY](BATCHES.md#cy-2026-10-16-大对象迁移无段缓冲直传与同端点分段落拷adr-014) | 2026-10-16 | 大对象迁移无段缓冲直传与同端点分段落拷（ADR-014）：`UploadPart` 显式长度 + 段级 Range 重试 + `UploadPartCopy`；#63 收窄至「大小未知」路径 |
| [§CZ](BATCHES.md#cz-2026-10-16-未知大小路径大对象上限提升至-1024gb-与容器内存预算-512m-抬至-1g) | 2026-10-16 | 未知大小路径单对象上限 640GB → 1024GB（段长 64MiB → 105MiB）+ 容器预算 512M → 1G（告警阈值 410 → 820MB）；KNOWN_ISSUES #63 闭环移除 |

## 三、质量与覆盖率现状

> 2026-09-15 本机实测；2026-09-16 P0 + P1 修复后复测：`go vet ./...` 干净、`go test -race ./...` 8/8 包通过
> （`handler` 覆盖率 100.0%）、`govulncheck ./...` **0 可达漏洞**、前端 62 文件 / **967** 测试全绿且四指标均 100%、
> `vue-tsc` / `vite build` / `eslint` 干净。
>
> 2026-09-19 风险登记集中处置后复测：`make test-cover` 8/8 包 **100.0%**（含新增 `deadcode_gate_test.go`）、
> `golangci-lint run ./...` **0 issues**、前端 64 文件 / **986** 测试全绿、`cargo audit` **0 漏洞**（7 条告警已 triage）。
>
> 2026-09-19 审查 §三（正确性 B3–B11 / F2–F10）处置后复测见 §R 末段：后端 8/8 包 **100.0%**、前端 66 文件 / **1039** 测试全绿。
>
> 2026-09-20 审查 §9.3 P2（契约 #26–#28 + 安全 / 供应链 #29–#34）处置后复测见 §T 末段：
> 后端 8/8 包 **100.0%**（`awk '$NF==0'` 零块）、`golangci-lint` **0 issues**、前端 66 文件 / **1039** 测试全绿、
> `docker compose config` 在缺 `S3C_STORE_KEY` 时拒绝启动。
>
> 2026-09-22 §37 真实联调收口后复测（roadmap §四 门禁基线同步为此轮实跑值）：后端 8/8 包 **100.0%**、
> `golangci-lint` **0 issues**、前端 66 文件 / **1042** 测试全绿（覆盖率 4074 / 2844 / 1095 / 3503 四指标 100%）、
> `pnpm audit` **0 漏洞**、`make e2e-real` **3 passed**。
>
> 2026-09-24 roadmap §三 #3 死代码纪律收口（前后端两道「零生产引用」导出门禁，见 §Z）后复测
> （roadmap §四 门禁基线同步为此轮实跑值）：`gofmt -l` 干净 / `go vet` 0 告警 / 后端 `go test` **8/8 包通过**、
> `golangci-lint` **0 issues**、`go build` 干净；前端 `pnpm lint` 0 告警 / `pnpm typecheck` + `typecheck:e2e` exit 0 /
> **66 文件 1043 例全绿**（覆盖率 **4072 / 2843 / 1093 / 3501 四指标 100%**）/ `pnpm build` OK。
>
> 2026-09-24 全仓代码审查处置（2 Critical + 20 Required 全清；Nit 31/35 闭环 + 3 项转登记 + 1 项判定不成立，见 §AA）后复测：
> `gofmt -l` 干净 / `go vet` 0 告警 / `go build` 干净 / 后端 `go test` **9/9 包通过**（R11 新增 `internal/atomicfile`，
> 故由 8 包增至 9 包；**每包 100.0% statements**）、`golangci-lint` **0 issues**；
> 前端 `pnpm lint` 0 告警 / `pnpm typecheck` + `typecheck:e2e` exit 0 / **67 文件 1110 例全绿**（覆盖率
> **4255 / 2908 / 1124 / 3653 四指标 100%**）/ `pnpm build` OK；真实 E2E 两项——
> `S3CLIENT_E2E=1 go test ./internal/s3wrap/ -run 'TestE2E'` **4/4 PASS**、`make e2e-real` **3 passed**
> （后端 `S3C_TOKEN` 开启的生产同构形态，C1 修复的验收实跑）。
>
> 2026-09-28 KNOWN_ISSUES #60–#63 收口（见 §AB）后复测：`gofmt -l` 干净 / `go vet` 0 告警 / `go build` 干净 /
> 后端 `go test` **9/9 包、每包 100.0% statements**、`golangci-lint` **0 issues**；前端 `pnpm lint` 0 告警 /
> `pnpm typecheck` + `typecheck:e2e` exit 0 / **72 文件 1110 例全绿**（#60 拆分只动文件归属，
> 测试名清单与拆分前**逐条一致**；覆盖率 **4255 / 2908 / 1124 / 3653 四指标 100% 不变**）/ `pnpm build` OK。
>
> **2026-09-29 §AN / §AO / §AP 复测**（roadmap §四 门禁基线同步为此轮实跑值）：
> `gofmt -l` 干净 / `go vet` **0 告警** / 后端 `go test` **9/9 包通过** / `golangci-lint` **0 issues**；
> 前端 `pnpm lint` 0 告警 / `pnpm typecheck` + `typecheck:e2e` exit 0 / **74 文件 1132 例全绿** /
> `pnpm test:coverage` **四指标 100%（宿主 `NODE_ENV` 未设 4296 / 2932 / 1130 / 3679；
> `NODE_ENV=production` 4294 / 2932 / 1130 / 3677）** / `pnpm build` OK（**366.72 kB，gzip 112.72 kB**；
> CSS 31.84 kB——§AP 的 `prefers-reduced-motion` 块与注释带来的 +0.16 kB，属预期）。
>
> ⚠️ **例数轨迹 1128 → 1126 → 1127 → 1132，没有一次是「丢测试」**：§AN 删除了死代码
> `isTopKeydown`（生产零调用，靠注释与测试「续命」）及**仅**测它的白盒用例，把断言改为
> **派发真实 `keydown` 观测行为**——旧断言读内部栈状态，只能证明「栈里有谁」，证明不了
> 「真的有且只有栈顶收到事件」；§AO 为死代码门禁补 1 条**合成源码口径用例**；§AP 为
> #67 / #68 补 5 条（`a11y_gate.test.ts` 3 条源码形态 + `i18n` 2 条行为）。故一降三升之间，
> **行为覆盖面是净增的**，覆盖率四指标始终 100%。`App.test.ts` / `i18n/index.test.ts` 中
> 只改注释里文档路径（`features.md` → `FEATURES.md`）的部分不涉及例数。
>
> **2026-10-01 §BM 复测**（ROADMAP #17 可访问性批次收口后全量重跑；roadmap §四 门禁基线同步为此轮实跑值）：
> `gofmt -l` 干净 / `go vet` **0 告警** / `golangci-lint` **0 issues** / 后端 `go test ./...`
> **9/9 包** + `make test-cover` **9/9 包 100.0%（`count==0` 零块）** / `go build` OK /
> `govulncheck ./...` **0 可达漏洞**；前端 `pnpm lint` 0 告警 / `pnpm typecheck` + `typecheck:e2e`
> 均 exit 0 / **75 文件 1155 例全绿**（§BM 新增 23 例，文件 74 → 75）/
> `pnpm test:coverage` **四指标 100%（4321 / 2932 / 1130 / 3706）** / `pnpm build` OK
> （**371.05 kB，gzip 113.84 kB**；CSS 32.40 kB——§BM 新增全局 `.sr-only` 工具类与
> `--brand-mark-*` 注释）；**真实浏览器 E2E 首次在本环境实跑** `pnpm e2e` **22 passed / 0 skipped**
> （含 `e2e/a11y.spec.ts` 5 条 axe 扫描——本批配色 / `role` / `caption` 改动的渲染态验证）；
> `cargo audit --no-fetch` **0 漏洞**（7 条告警已 triage）。
> 同轮补跑**两项真实 E2E**（本轮改了前端，按 AGENTS 必跑）：`make e2e-real` **3 passed / 0 skipped**
> （真实后端 + RustFS + 真实产物，`S3C_TOKEN` 生产同构形态；本机 8080 被 `haproxy` 占用，用
> `SERVER_PORT=8081`，脚本编排不变）；`S3CLIENT_E2E=1 go test ./internal/s3wrap/ -run 'TestE2E'`
> **4/4 PASS**（临时起 `rustfs/rustfs:1.0.0-rc.3` 于 `127.0.0.1:9000`，跑完即删）；另
> `make check` **exit 0**（vet / lint / 后端覆盖率 / 前端覆盖率 / `typecheck:e2e` 一把跑齐）、
> `make bench` **exit 0**（确定性预算门禁 + 原始基准数字通过）。
> **未能实跑**：`pnpm audit`——镜像 `registry.npmmirror.com` 不提供 audit 端点
> （`ERR_PNPM_AUDIT_ENDPOINT_NOT_EXISTS`），沿用 CI 与 2026-09-22 值。

> **2026-10-01 §BN 复测**（ROADMAP #10 代码生成批次收口后全量重跑；roadmap §四 门禁基线同步为此轮实跑值）：
> `gofmt -l` 干净 / `go vet` **0 告警** / `golangci-lint` **0 issues** / `go build ./...` OK /
> 后端 `go test ./... -count=1` **9/9 包** + `make test-cover` **9/9 包 100.0%（`count==0` 零块）** /
> `govulncheck ./...` **0 可达漏洞**；前端 `pnpm lint` **0 告警** / `pnpm typecheck` + `typecheck:e2e`
> 均 exit 0 / **76 文件 1158 例全绿**（§BN 新增 3 例，文件 75 → 76，新增 `src/api/generated.gate.test.ts`）/
> `pnpm test:coverage` **四指标 100%（4327 / 2932 / 1131 / 3712）** / `pnpm build` OK
> （**377.34 kB，gzip 114.83 kB**；CSS 32.40 kB——`src/api/operations.ts` 进入生产包）/
> **`pnpm gen:api --check` exit 0**（§BN 新登记的生成物新鲜度门禁）。
> **仓库级**：`make check` **exit 0**、`make bench` **exit 0**。
> **渲染态与真实链路**（本批改了前端，按 AGENTS 必跑）：`pnpm e2e` **22 passed / 0 skipped**、
> `make e2e-real` **3 passed**（`SERVER_PORT=8081`）、`S3CLIENT_E2E=1 go test ./internal/s3wrap/
> -run 'TestE2E' -v` **4/4 PASS**——**首轮 4/4 全红是环境性的**（本机 `127.0.0.1:9000` 当时无
> RustFS，`connection refused`），临时起 `rustfs/rustfs:1.0.0-rc.3` 后复跑 4/4 PASS，
> 跑完 `docker rm -f` 并核验端口已关闭。
> 第三方许可证清单重生成：**Go 43 模块 / Rust 428 crates / npm 1 包，UNKNOWN 0 项**（devDependency 不入清单）。
> **未能实跑**：`pnpm audit`——同上，镜像不提供 audit 端点，沿用 CI 结果。

> **2026-10-10 全量复测**（§CE–§CH 四批合并态；roadmap §四 门禁基线同步为此轮实跑值）：
> `gofmt -l` 干净 / `go vet` **0 告警** / `golangci-lint` **0 issues** / `go build ./...` OK /
> 后端 `go test ./... -count=1` **10/10 包** + `make test-cover` **10/10 包 100.0%（`count==0` 零块）** /
> `govulncheck ./...` **0 可达漏洞**（21 个不可达模块漏洞）；前端 `pnpm lint` **0 告警** /
> `pnpm typecheck` + `typecheck:e2e` 均 exit 0 / **83 文件 1292 例全绿**（`a68ff0b` 新增
> `src/composables/useVirtualRows.test.ts`）/ `pnpm test:coverage` **四指标 100%（4829 / 3243 / 1218 / 4180）** /
> `pnpm build` OK（**417.73 kB，gzip 125.30 kB；CSS 33.76 kB**）/ `pnpm gen:api --check` **exit 0** /
> `docker compose config` base / prod / tls 均通过；渲染态与真实链路：`pnpm e2e` **22 passed + 1 skipped**、
> `make e2e-real` **5 passed / 0 skipped**（`SERVER_PORT=8081` + 外部共享 RustFS `--no-rustfs`）、
> `S3CLIENT_E2E=1 go test ./internal/s3wrap/ -run 'TestE2E'` **7/7 PASS**；
> `cargo audit --no-fetch` **0 漏洞**（7 条告警已 triage）。
> **未能实跑**：`pnpm audit`——镜像不提供 audit 端点，沿用 CI 结果。
>
> **门禁清单与发布前基线以 [`ROADMAP.md`](ROADMAP.md) §四 为唯一来源**（本节上表不再重复维护整套现状，
> 避免两处「现状表」互相漂移）；本节上方各带日期的复测块是逐批实跑历史。下表只列 §四 未单列的补充门禁：

| 门禁 | 结果 |
|---|---|
| `docker compose config` | base / prod / tls 均通过（2026-10-10 实跑） |

### 已知边界与取舍
- **单实例**：文件型 store（`json` / `sqlite`）+ 内存任务表不支持多副本，启动时对 `S3C_DATA_DIR` 加 `flock` 单写者锁，第二实例启动即失败；非 unix（Windows）无跨进程文件锁，为文档化 no-op。
- **SSRF 默认放行私网 / 回环**（自托管主场景，[ADR-003](decisions/0003-ssrf-private-allow.md)）；需要更严策略时设 `S3C_SSRF_DENY_PRIVATE=1`。
- OpenAPI `components.schemas` / `parameters` / `responses` 已全部接线为 `$ref`（`refSchema` / `refParam` / `refResp`）；中间件级通用状态码 `401 / 413 / 429 / 500` 自 2026-10-10 起在全部注册完成后由 `applyUniversalResponses` 逐 operation 补挂到共享 `Unauthorized` / `PayloadTooLarge`（新增）/ `TooManyRequests` / `InternalError`——此前三个组件**定义后 0 引用**、84 个 operation 无一声明这些状态，由 `TestOpenAPIUniversalResponsesAreWiredPerOperation` 钉住（原「全局错误词汇、不绑定单个端点」的表达已过时）。
- `POST /api/accounts/preview-buckets` 使用表单临时凭据只读 `ListBuckets` 并立即返回，**不落库、不校验对已有账号**；缺 `endpoint`/`accessKey`/`secretKey` 返回 400，上游 `ListBuckets` 失败返回 500。仍受 `s3wrap.New` 的 endpoint 格式校验与拨号期 SSRF（禁 IMDS/链路本地）防护。
- 桌面端仅做壳与分发，不使用 Tauri IPC。
