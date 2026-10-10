# s3client 功能大全（Features）

> 本文件是 s3client 的**单一事实来源**：产品能力总览 + 全部已完成修复 / 优化记录。
> 已把散落在各评估文档与 [`CHANGELOG.md`](../CHANGELOG.md) 中的「已实现 / 已修复 / 已完善」功能统一汇总于此；CHANGELOG 仍保留逐字发布历史。
>
> - 已知问题：[`KNOWN_ISSUES.md`](KNOWN_ISSUES.md)（缺陷 / 阻塞 / 技术债） · 迭代方向：[`ROADMAP.md`](ROADMAP.md) §三 · 发版历史：[`CHANGELOG.md`](../CHANGELOG.md) · 综合评估：[`archive/assessment.md`](archive/assessment.md)
> - 接口细节：[`api.md`](api.md) · 错误约定：[`errors.md`](errors.md) · 开发规范：[`DEVELOPMENT.md`](DEVELOPMENT.md) · 安全设计：[`threat-model.md`](threat-model.md) · Nginx 部署：[`deploy/nginx/README.md`](../deploy/nginx/README.md)
>
> 最后更新：2026-10-10（`v1.0.0` 之后的 Unreleased 区间；含分支状态审查 P0 / §三 / P1 / P2 四轮处置 + §7.4 门禁盲区收尾 + §AA 全仓代码审查处置 + §AB KNOWN_ISSUES #60–#63 收口 + §AJ #64 闭环 + §AK 前端测试与宿主 `NODE_ENV` 解耦 + §AR 补齐事故复盘模板、§AS 文档失真收口（11 处 + Dependabot 路径）、§AT 文档缺口收口（链接/锚点门禁 + 登记表 + 子树 AGENTS + 隐私）、§AU 接口契约表达鉴权（security/tags）、§AV 元信息/导航收口（docs 落地页 + 导航覆盖门禁 + `.gitattributes`）、§AW 安全与供应链收口（自动生成许可证清单 + 依赖覆盖门禁 + 产物核验）、§AX AI 时代层收口（AI 治理机械保证 + Copilot 指针入口）+ 2026-09-30 文档基线补缺与导航收口（§AY–§BF 共八节）：§AY 客户端支持矩阵（浏览器 / 桌面 OS）、§AZ ADR 覆盖补足（8 篇 + 取舍表门禁）、§BA 供应链收口（Scorecard + Dependency Review）、§BB AI 产出效果证据层（评测 + 度量）、§BC 英文文档入口（docs/en/index.md）、§BD 导航收口残留（命名两处分叉 + `docs_naming_gate`）、§BE 状态台账 #69 收口（CHANGELOG ↔ git tag 一致性）、§BF docs 内待修项收口（对比度静态核查 + 改进项登记 ROADMAP #17/#18）、§BG AI 时代文档补强（六路并行：P0 效果证据 / P1 机器可读 / P2 元信息）、§BH 门禁实跑回填与 §BG 结构归位、§BI 第二轮收口（ROADMAP ✅ 行移出 / §AU 注记 / 评估缺口补登记 / §四 复测）、§BJ ROADMAP #19 文档可读性与流程机械化（mermaid 时序图 / 超大体量预警 / 复审到期门禁 / PR 模板 ADR 勾选）、§BK KNOWN_ISSUES #70 闭环（`NormalizeEndpoint` 幂等修复，推翻 ➖ 决策）、§BL ROADMAP #18 可观测性补全（五个观测指标 + 告警规则 + 仪表盘）、§BM ROADMAP #17 可访问性补强（焦点陷阱 / 统一播报 / 表格与可见标签 / 组件级 vitest-axe / 对比度修色收尾）、§BN ROADMAP #10 OpenAPI → 前端类型 / 客户端代码生成（schema-first + diff 门禁）、§BO KNOWN_ISSUES #71 闭环（仓库 slug 统一 `github.com/weilai1949/s3client`，推翻 ➖「维持现状」决策）、§BP KNOWN_ISSUES #71 第二批（产品名 `s3clinet` → `s3client` 全量统一，含锁文件 / 镜像 / 指标命名空间）、§BQ ROADMAP #8 大文件体验（上传断点续传 + 下载并行分段，71 端点）、§BR ROADMAP #11 零依赖 OTLP tracing（W3C traceparent + OTLP/HTTP JSON，默认关）、§BS ROADMAP #13 Token 作用域与最小权限（`S3C_TOKEN_SCOPES`）、§BT ROADMAP #5 S3 新协议特性（条件写 / 端到端校验和 / Object Lock）、§BU ROADMAP #6 计划任务（cron 定时增量备份）、§BV ROADMAP #7 FinOps 存储分析与成本看板（全栈：聚合端点 83→84 + 独立「成本看板」Tab）、§BW Go 工具链 1.26.6 → 1.26.9（10 个可达 stdlib 漏洞归零）+ e2e-real 用例幂等化（断言前返回列表 + 清桶走对象级批量删除）+ 本文件 e965e54 重复章节去重、§BX e2e-real `seedAccountAndBucket` 失败泄漏修复（登记簿判空清理 + 建桶失败快诊 + 限速节流）、§BY s3wrap E2E 清桶强删路径（Object Lock 锁定版本根治共享实例残留桶泄漏）、§BZ 全仓代码评审批次（C1–C2 + R1–R9 全修 + O1–O12 登记 #72–#83）、§CA KNOWN_ISSUES #73 / #80 / #81 / #83 闭环（s3wrap 导出面降级 / 计划校验定值文案 + 回显门禁 / accounts 创建面收口 / 散点缺陷群 12 处）、§CB KNOWN_ISSUES #82 闭环（R10 残留：`make check` 缺项 / E2E 路径过滤 / DEVELOPMENT 漂移 + 防回退门禁）、§CC KNOWN_ISSUES #72 / #74 / #75 / #77 / #78 闭环（数据面 UNSIGNED-PAYLOAD 收窄 / 前端 ApiError / generated.gate 穷尽性 / 右键菜单 a11y / i18n 盲区与硬编码文案）、§CD KNOWN_ISSUES #76 / #79 闭环（newRowKey ×7 + 虚拟滚动 ×4 canonical 化 / OpenAPI 状态码与 bucket-required 门禁；2026-10-09 评审 O1–O12 归零）、§CE 2026-10-10 根 `.env.example` ⇔ compose 透传面双向一致（新增双向门禁 / README 配置项计数 18 → 22）、§CF 2026-10-10 `make dev` 受管 web PID 指向修复 + 启动流程去自动 tidy（新增门禁）、§CG 2026-10-10 两处文档状态漂移收口（悬空 `ROADMAP #11` 引用 / trace 观测面自相矛盾 / FEATURES 无锚点开放陈述；新增状态陈述门禁）、§CH 2026-10-10 A 组六项可机械优化（内置卷 inode 指标 + `S3ClientVolumeInodeLow` / 告警表 ⇔ `rules.yml` 双向门禁 / AI 政策「不自动合并」升级代码强制 / 网格态 axe 扫描判定为非缺陷 / 明文 `http://` 端点告知）、§CI 2026-10-10 全 docs 通读问题清单收口（表渲染断裂 / 过期数字与日期 / 悬空指针 / 安全契约缺失 / 通用状态码未接线 / 英文页漏改 / 升级回滚步骤缺失，+ 三道门禁））

## 目录

- [一、产品功能](#一产品功能)
  - [1. 账号管理](#1-账号管理) · [2. 存储桶与桶属性](#2-存储桶与桶属性) · [3. 对象浏览与检索](#3-对象浏览与检索)
  - [4. 上传](#4-上传) · [5. 下载与预览](#5-下载与预览) · [6. 对象操作](#6-对象操作)
  - [7. 版本与回收站](#7-版本与回收站) · [8. 迁移与增量同步](#8-迁移与增量同步)
  - [9. 存储驱动与数据安全](#9-存储驱动与数据安全) · [10. 服务端安全与鉴权](#10-服务端安全与鉴权)
  - [11. API 与契约](#11-api-与契约) · [12. 前端体验与无障碍](#12-前端体验与无障碍)
  - [13. 桌面端](#13-桌面端) · [14. 部署、CI 与工程化](#14-部署ci-与工程化)
- [二、已完成修复与优化](#二已完成修复与优化) — A 本轮增量 · B 驱动去重明细 · C 全方位评估 58 项 · D v1.0.0-rc1 评估 21 项 · E Optional/Nit 长尾 · F 历史版本全量台账（0.1.0→v1.0.0-rc1） · G Unreleased · H–Z 各轮处置台账 · AA 2026-09-24 全仓代码审查处置 · AB 2026-09-28 KNOWN_ISSUES #60–#63 收口 · AC 2026-09-28 DEVELOPMENT.md §7 历史技术债收口 · AD 2026-09-28 三路五轴复审（闭环 4 条 + 15 条转 #64） · AE 2026-09-28 破坏性操作审计覆盖补齐 · AF 2026-09-28 前端四条（sticky error / DestDialog 并发 / signing 死状态） · AG 2026-09-28 config 三条（显式 env 文件 fail-closed / 关停超时上界 / 数据目录 0700） · AH 2026-09-28 s3wrap 两条（metadata 值控制字符 / IDN 端点） · AI 2026-09-28 前端另四条（代次守卫 / 追加重置滚动 / loadingAll / 桶列举标志） · AJ 2026-09-28 KNOWN_ISSUES #64 闭环（`store.Open` json 分支丢 `storeKey`） · AK 2026-09-29 前端测试与宿主 `NODE_ENV` 解耦（`vite.config.ts` 隔离 + `vite_env_guard.test.ts` 守卫） · AL 2026-09-29 对照通用 AGENTS.md 模板补齐代理治理与配置 SSOT · AM 2026-09-29 文档命名规则收敛为「元文档大写 / 内容文档小写」 · AN 2026-09-29 文档覆盖矩阵收口（11 个新文档 + 3 项供应链门禁 + 机器可读契约 + 性能基线） · AO 2026-09-29 死代码门禁改用 TS AST 判定引用 · AP 2026-09-29 可访问性三处 + nginx 跨层日志关联 · AQ 2026-09-29 供应链收口（桌面 SBOM + cosign）+ 告警规则 + 账号库 Schema · AR 2026-09-29 补齐事故复盘模板（`docs/POSTMORTEM_TEMPLATE.md`） · AS 2026-09-29 文档失真收口（11 处「文档与实现 / 自身不一致」+ Dependabot 路径失效） · AT 2026-09-29 文档缺口收口（链接/锚点门禁 + 文档登记表 + ADR 模板 + 子树 AGENTS + 隐私声明） · AU 2026-09-29 接口契约表达鉴权（文档级 security + 逐端点豁免 + tags 分组） · AV 2026-09-29 元信息/导航收口（docs 落地页 + 导航覆盖门禁 + `.gitattributes`） · AW 2026-09-29 安全与供应链收口（自动生成许可证清单 + 依赖覆盖门禁 + 产物核验指南） · AX 2026-09-29 AI 时代层收口（AI 治理机械保证 + Copilot 指针入口） · AY 2026-09-30 客户端支持矩阵收口 · AZ 2026-09-30 ADR 覆盖补足（8 篇 + 覆盖门禁） · BA 2026-09-30 供应链收口（Scorecard + Dependency Review） · BB 2026-09-30 AI 产出效果证据层收口 · BC 2026-09-30 英文文档入口（docs/en/index.md） · BD 2026-09-30 导航收口残留（命名两处分叉 + `docs_naming_gate`） · BE 2026-09-30 状态台账 #69 收口（CHANGELOG ↔ tag 一致性） · BF 2026-09-30 docs 内待修项收口（文档失真 + 对比度静态核查 + 改进项登记 SSOT） · BG 2026-09-30 AI 时代文档补强（六路并行：效果证据 / 机器可读 / 元信息） · BH 2026-09-30 门禁实跑回填与 §BG 结构归位 · BI 2026-09-30 第二轮收口（ROADMAP ✅ 行移出 + 评估缺口补登记） · BJ 2026-09-30 ROADMAP #19 文档可读性与流程机械化（mermaid / 体量预警 / 复审到期门禁 / PR ADR 勾选） · BK 2026-09-30 KNOWN_ISSUES #70 闭环（NormalizeEndpoint 幂等修复） · BL 2026-09-30 ROADMAP #18 可观测性补全（五个观测指标 + 告警规则 + 仪表盘） · BM 2026-09-30 ROADMAP #17 可访问性补强（焦点陷阱 / 统一播报 / 表格与可见标签 / 组件级 vitest-axe / 对比度修色收尾） · BN 2026-10-01 ROADMAP #10 OpenAPI → 前端类型 / 客户端代码生成（schema-first + 生成物新鲜度门禁） · BO 2026-10-08 KNOWN_ISSUES #71 闭环（仓库 slug 统一 `github.com/weilai1949/s3client` + 全仓 import / 仓库 URL） · BP 2026-10-08 KNOWN_ISSUES #71 第二批（产品名 `s3clinet` → `s3client` 全量统一 + 5 文件改名 + 生成物重生成） · BQ 2026-10-08 ROADMAP #8 大文件体验（上传断点续传 + 下载并行分段，71 端点） · BR 2026-10-08 ROADMAP #11 零依赖 OTLP tracing（traceparent + OTLP/HTTP JSON，默认关） · BS 2026-10-08 ROADMAP #13 Token 作用域与最小权限（`S3C_TOKEN_SCOPES`） · BT 2026-10-08 ROADMAP #5 S3 新协议特性（条件写 / 端到端校验和 / Object Lock） · BU 2026-10-08 ROADMAP #6 计划任务（cron 定时增量备份） · BV 2026-10-08 ROADMAP #7 FinOps 存储分析与成本看板（全栈） · BW 2026-10-09 Go 工具链 1.26.6 → 1.26.9（10 可达 stdlib 漏洞归零）+ e2e-real 用例幂等化（含本文件去重） · BX 2026-10-09 e2e-real seed 失败泄漏修复（登记簿判空清理 + 建桶失败快诊 + 限速节流） · BY 2026-10-09 s3wrap E2E 清桶强删路径（Object Lock 锁定版本根治共享实例残留桶泄漏） · BZ 2026-10-09 全仓代码评审批次（C1–C2 + R1–R9 全修 + O1–O12 登记 #72–#83） · CA 2026-10-09 KNOWN_ISSUES #73 / #80 / #81 / #83 闭环（s3wrap 导出面降级 / 计划校验定值文案 + 回显门禁 / accounts 创建面收口 / 散点缺陷群 12 处） · CB 2026-10-09 KNOWN_ISSUES #82 闭环（R10 残留：make check 缺项 / E2E 路径过滤 / DEVELOPMENT 漂移） · CC 2026-10-09 KNOWN_ISSUES #72 / #74 / #75 / #77 / #78 闭环（UNSIGNED-PAYLOAD 收窄 / 前端 ApiError / generated.gate 穷尽性 / 右键菜单 a11y / i18n 盲区与硬编码文案） · CD 2026-10-09 KNOWN_ISSUES #76 / #79 闭环（newRowKey + 虚拟滚动 canonical 化 / OpenAPI 状态码与 bucket-required 门禁；评审 O1–O12 归零） · CE 2026-10-10 根 `.env.example` ⇔ compose 透传面双向一致（+ 双向门禁）+ README 配置项计数订正 · CF 2026-10-10 `make dev` 受管 web PID 指向修复 + 启动流程去自动 tidy（+ 门禁） · CG 2026-10-10 两处文档状态漂移收口（悬空 `ROADMAP #11` 引用 / trace 观测面自相矛盾 / FEATURES 无锚点开放陈述）（+ 状态陈述门禁） · CH 2026-10-10 A 组六项可机械优化（内置卷 inode 指标 + `S3ClientVolumeInodeLow` / 告警表 ⇔ `rules.yml` 双向门禁 / AI 政策「不自动合并」升级代码强制 / 网格态 axe 扫描 / 明文 `http://` 端点告知）（+ 一道门禁） · CI 2026-10-10 全 docs 通读问题清单收口（含 openapi 通用状态码接线 / en 页数字 / docs 导航表格修复）（+ 三道门禁）
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

### A. 2026-09 本轮增量（store 去重 / OpenAPI 契约 / 长尾）

| 项 | 状态 | 内容 |
|---|---|---|
| 双存储驱动重叠 | ✅ | 抽取共享 `fileStore` + `fileCodec`（净减约 331 行）后**再收敛完成**：`EncryptedStore`/`encryptedCodec` 并入统一 `Store` + 单一 `storeCodec`（strict 区分 json permissive / encrypted 严格），`encrypted.go` 删除、`NewEncrypted` 返回 `*Store`；磁盘格式 / 错误文案 / 加密语义不变 |
| `Create` 跨驱动一致性 | ✅ | 统一存防御性副本（此前 json 驱动别名调用方指针，encrypted 已复制） |
| 跨驱动对照测试 | ✅ | `crossdriver_test.go`：三驱动 CRUD 等价、遗留 S3C2 信封可读、落盘字节布局 |
| OpenAPI 覆盖率 | ✅ | `internal/openapi` 与 handler 内全部 OpenAPI 函数 **100.0%** statement |
| OpenAPI 契约强度 | ✅ | `openapi_contract_test.go`：69 条路由↔规范双向一致（漏登记/陈旧都红灯）、operation 完整性 + 唯一 operationId、`{param}` 必填 path 参数、`$ref` 可解析（含解析器自检）、`MarshalJSON` 确定性（含并发）、顶层 3.0.3 形状 |
| OpenAPI `$ref` 接线 | ✅ | 新增 `openapi.Ref()` / `Param.Ref` / `Response.Ref`；共享 schema / parameter / response 经 `refSchema` / `refParam` / `refResp` 全部接线（109 处 `$ref`、12 个唯一目标），契约测试支持 `$ref` 解析并新增「components 无死片段」断言（全局错误词汇除外） |
| 错误映射去字符串匹配 | ✅ | sentinel `ErrObjectTooLarge` / `ErrSourceDeleteFailed` + `errors.Is`；`PutObject`/`CopyObject` 归一 S3 `EntityTooLarge` |
| `X-Request-ID` 加固 | ✅ | ≤128 且仅可见 ASCII，非法值服务端生成 UUID |
| Bearer scheme | ✅ | RFC 7235 大小写不敏感，凭证仍常量时间比较 |
| `.env` 解耦 CWD | ✅ | `S3C_ENV_FILE`（唯一来源）→ CWD → 可执行文件同目录；真实环境变量优先 |
| 流式复制缓冲 | ✅ | 有界复用（常驻上限 1×64MB = 512M 容器预算 1/8），成功与 abort 路径都归还 |
| `metadata.go` 变量遮蔽 | ✅ | 循环变量 `r` → `row`（不再遮蔽 `*http.Request`） |
| SSE 卸载中断 | ✅ | `DestDialog` 迁移进度流、`useObjectActions` 前缀删除流 `onBeforeUnmount` 主动 abort |
| VersionsDialog 分页 | ✅ | `keyMarker`/`versionIdMarker` 翻页（≤20 页），其余部分显式截断提示 |
| 非空断言清理 | ✅ | 剩余 3 处 `ctx.account.value!` + `{} as KeyBindings` 改为守卫 / 可选类型 |
| ESLint 收紧 | ✅ | `@typescript-eslint/no-explicit-any` warn → error（0 违规） |
| gofmt 对齐 Go 1.26 | ✅ | 7 个既有文件按新字段对齐 / 文档注释规则格式化，CI `gofmt -l .` 恢复干净 |

### B. 双存储驱动去重明细

| 文件 | 变化 |
|---|---|
| `apps/server/internal/store/filestore.go` | 新增（213 行）：共享内存状态 + 锁 + CRUD + 回滚 + `snapshotLocked`/`persistLocked` |
| `apps/server/internal/store/crypto.go` | 新增（66 行）：S3C2 常量、`deriveKey`、`envelope`、AES-256-GCM 加解密 |
| `apps/server/internal/store/store.go` | 244 → 93 行：仅选 `storeCodec`（明文 + 兼容 S3C2） |
| `apps/server/internal/store/encrypted.go` | 267 → 80 行（去重后仅选 `encryptedCodec`）；**再收敛后已删除**，逻辑并入 `store.go` 的 `storeCodec`（strict） |
| `apps/server/internal/store/account_store.go` | `ErrNotFound` 移到接口旁 |
| `open.go` / `sqlite.go` / `atomic.go` | 未改动（文件路径、返回类型、故障注入 seams 全部保留） |

> 再收敛已完成（本轮）：`encryptedCodec` / `EncryptedStore` 并入统一 `storeCodec`（`strict` 区分
> permissive/严格语义），`encrypted.go` 删除，`NewEncrypted` 返回 `*Store`；行为、错误文案、磁盘格式不变。

### C. 全方位评估 58 项（2026-04-19 全部落地）

#### 安全性
| 项 | 状态 | 修复 |
|---|---|---|
| S-1 明文 SecretKey | ✅ | JSON Store 落盘 AES-256-GCM（`S3C_STORE_KEY` 派生，S3C2｜salt｜ciphertext，兼容明文旧文件） |
| S-2 XFF 限速绕过 | ✅ | `clientIP` 优先 `X-Forwarded-For`（逗号分割），回退 `RemoteAddr` |
| S-3/S-6 openapi.json 泄露 | ✅ | `S3C_EXPOSE_OPENAPI` 门控（默认 404），开启后需 Bearer 鉴权；移出 `withAuth` 绕过 |
| S-4 parsePolicy 误判 | ✅ | UI 明确提示「不支持结构需用原始 JSON 编辑」，消除静默误解析 |
| S-5 批量 key 无上限 | ✅ | `BATCH_META_MAX_KEYS=10000`（超限抛错） |
| S-7 testAccount 泄露 | ➖ | `UserMessage` 防腐层已映射为通用消息，无内部细节泄露 |
| S-8 proxy key 遍历 | ✅ | 拒绝含控制字符的 key（S3 对 `..` 字面段安全） |
| S-9 CSP connect-src | ✅ | 默认收紧为 `'self'` + 本地 Tauri 后端；`S3C_CSP_CONNECT_SRC` 可显式放宽 |

#### 架构与设计
| 项 | 状态 | 修复 |
|---|---|---|
| A-1/M-1 openapi_register 拆分 | ✅ | 拆为 9 个 `openapi_register_*.go` 按域文件 |
| A-2/M-2 Handler 拆分 | ➖ | 评估后维持：方法已按域分文件，拆子 handler 收益低、风险高 |
| A-3/P-6/C-7 共享 HTTP 客户端 | ✅ | `MaxConnsPerHost=32`，批量操作不无限占 fd |
| A-4 SyncKeys 内存 | ✅ | `indexDst` 增加 100k 硬上限（与 listAll 一致） |
| A-5/P-5/C-6 results 缓冲 | ✅ | `RunBatch` results 通道改无缓冲：内存 O(workers)，进度即时回调 |

#### 性能与并发
| 项 | 状态 | 修复 |
|---|---|---|
| P-1 awsconfig 不可取消 | ✅ | 10s timeout context（`cfgCtx`） |
| P-2 SameEndpoint 回退 | ✅ | region-aware：两端均为空时仅当 region 相同才视为同端 |
| P-3 zip goroutine 泄漏 | ✅ | `ctxCancelReader` + `context.AfterFunc` 中断阻塞读 |
| P-4/C-5 列举间无 ctx 检查 | ✅ | `SyncKeys` 列举后检查 `ctx.Err()`，取消即返回 |

#### 测试质量
| 项 | 状态 | 修复 |
|---|---|---|
| T-1 zip cancel 时序 | ✅ | `ready.WaitGroup` 同步 + 取消路径确定性触发 |
| T-2 batchMetadata mock | ✅ | `vi.mocked` 类型化 + 签名对齐 |
| T-3/T-4 测试全局变量 | ✅ | `syncStore` 加 mutex（`sync_test` 已有 `s3FakeMu`） |
| T-5 同步测试缺口 | ✅ | 新增 CompareSizeTime / prefix 过滤 / 跨端点 StreamCopy 三测试 |
| T-6 e2e 覆盖缺口 | ✅ | `e2e/features.spec.ts` 10 用例；15 passed / 0 skipped |
| T-7 Makefile | ✅ | `test` 加 `-race`；新增 `test-cover` / `web-test-cover` |
| T-8 store 编译回归 | ✅ | 复用 `encrypted.go` 共享加密助手；store 覆盖率提升 |
| T-9 前端覆盖率 | ✅ | 11.28% → 95.27% statements；36 组件 + 全部 composables 覆盖；顺带修 3 个缺陷（dropzone 点击递归、MigratePanel ResizeObserver 时机、deleteServer 当前服务器 applyProfile） |

#### 文档与 DevOps
| 项 | 状态 | 修复 |
|---|---|---|
| D-1 pre-commit | ✅ | `.githooks/pre-commit`（gofmt + go vet + fe typecheck）+ `make install-hooks` |
| D-2 e2e 重试 | ✅ | Playwright CI retries 2 |
| D-3 Makefile 缺失 | ➖ | 根 Makefile 存在且已强化 |
| D-4 errors.md 未更新 | ✅ | 补充 migrate/sync 与 openapi.json 的错误约定 |

#### 前端 UX 与无障碍
| 项 | 状态 | 修复 |
|---|---|---|
| UX-1/3 tag 输入 label | ✅ | `sr-only` label + i18n（tagKey/tagValue） |
| UX-2 策略编辑器 label | ➖ | 已由外层 `<label>` 包裹（核查确认满足） |
| UX-4 watcher 级联 | ✅ | 移除 deep `watch(doc)`，`prevRaw` 守卫防 dirty 重置 |
| UX-5 进度条 | ✅ | `<progress>` + onProgress 实时推进 |
| UX-6/E-1 死代码 | ✅ | catch 改通用 `step:'batch'` |
| UX-7 copySelectedLinks | ✅ | `Promise.allSettled` 部分成功 |
| UX-8 runUpload 队列 | ✅ | 清空移入 try（完成后清理） |
| UX-9/M-4 templateLabels | ✅ | 从 `POLICY_TEMPLATES` 派生 |
| UX-10 全选 label | ➖ | 已由 `<label>` 包裹 |
| UX-11 emoji 按钮 | ✅ | `📊` + `aria-label` |
| UX-12 emoji 提示 | ✅ | `aria-hidden` + i18n |
| UX-13 错误截断 | ✅ | 「显示全部 n 条 / 收起」 |
| UX-14 空标签提交 | ✅ | `hasTagChange`（全空禁止提交） |
| M-3 缩进 | ✅ | 统一 4 空格 |

#### API 设计 / 错误处理
| 项 | 状态 | 修复 |
|---|---|---|
| API-1/4 批量元数据端点 | ➖ | 前端编排用的单对象端点（object-acl / tags / storage-class）已在 OpenAPI 登记 |
| API-2 migrate/sync 登记 | ➖ | 已在 OpenAPI 登记 |
| API-3 maxBody 偏小 | ✅ | 4MB → 8MB |
| E-2 removeSelected toast | ✅ | `objects.toastDeleteFailed` |
| E-3 copySignLink toast | ✅ | 成功带 key、失败新 toast |
| E-4 错误横幅 | ✅ | `role="alert"` + 关闭按钮 |
| E-5 错误泄露 | ✅ | `toErrorMessage` 防腐确认 |

#### 附带修复的真实缺陷
- **ModalDialog 缺少 footer 插槽**：BatchMetadataDialog 确定/取消按钮永不渲染 → 已加 `<slot name="footer" />`。
- **ConfirmDialog / PromptDialog z-index 相同**：弹出在 ModalDialog 背后不可见 → z-index 200 → 300。
- **ObjectContextMenu 溢出视口**：低处菜单项不可点击 → `max-height` + 滚动 + 防负值。
- **BatchMetadataDialog open 受控化**：父级 `:open` 而非 `v-if`，保留进行中状态。

### D. v1.0.0-rc1 评估 21 项（含后续补修）

| 严重度 | 项 | 状态 |
|---|---|---|
| CRITICAL | `gaps_test.go` `release` channel 未关闭 / `done` channel 竞争 / 冗余 `sync.Once` | ✅ |
| HIGH | BatchMetadataDialog 空 tags 在 replace 模式静默清空服务端标签 | ✅ |
| HIGH | BatchMetadataDialog 异常无用户反馈 | ✅ |
| MEDIUM | `zip.go` `CreateHeader` 失败时 `item.body` 泄漏 | ✅ |
| MEDIUM | `zip.go` manifest 文件句柄无效 `defer` | ✅ |
| MEDIUM | `zip.go` `io.Copy` 失败后 ZIP 流状态未定义 | ✅ |
| MEDIUM | `batchMetadata.ts` `acl` 类型过宽 | ✅ |
| MEDIUM | `batchMetadata.test.ts` 未使用参数 | ✅ |
| LOW | BatchMetadataDialog 无障碍（aria-label / aria-live） | ✅ |
| LOW | `batchMetadata.ts` `accId` 别名 / 冗余 `as` / 空 keys 守卫 | ✅ |
| LOW | `zip.go` `ctxReader` 无法中断阻塞式底层 Read | ✅（后续 `ctxCancelReader` + `context.AfterFunc` 专项修复） |
| LOW | `zip.go` producer goroutine 泄漏 | ➖（复核为无泄漏） |
| MEDIUM | `batchMetadata.ts` 计数器并发非原子 | ➖（JS 单线程事件循环保证原子性） |
| INFO | 「GET /api/accounts 返回明文 SecretKey」指控 | ➖（误报：所有出口均 `Sanitized()`；后续契约收敛为 `AccountView.secretSet`，见 §10「账号密钥不出口」） |
| — | 其余 Info 项 | ✅ / ➖（见 `CHANGELOG.md` 对应版本段） |

### E. Optional / Nit 长尾采纳情况（v1.0.0-rc1 评估）

| 项 | 状态 | 说明 |
|---|---|---|
| encrypted Update 失败回滚内存 | ✅ | 已与 json 驱动对齐 |
| 流式复制 64MB/worker 缓冲 | ✅ | 改为有界复用（常驻 ≤64MB） |
| `/api/metrics` 鉴权暴露 | ✅ | `withMetricsGate` 默认 404，`S3C_EXPOSE_METRICS=1` 显式开启 |
| migrate SSE 写超时 | ✅ | 滚动 `SetWriteDeadline`（`streamIdleTimeout`） |
| worker-pool 仓内重复 4 份 | ✅ | 收敛为 `service.RunBatch` |
| `ProxyFromEnvironment` 绕过 SSRF | ✅ | transport 显式 `Proxy=nil`，测试 `ssrf_gap_test.go` |
| deleteObjects key 数上限 | ✅ | `maxDeleteKeys=1000`，超限 400 |
| user metadata 以 500 返回 | ✅ | `ValidateUserMetadata` → 400 |
| `ssrf.go` `To4` 死分支 | ➖ | 代码中已无 `To4`；拨号循环分支互异 |
| showDetail 无 seq 守卫 | ✅ | `useObjectActions` `detailSeq` |
| SSE 订阅未在卸载 abort | ✅ | DestDialog / useObjectActions 已补 |
| VersionsDialog 忽略分页截断 | ✅ | 翻页 + 截断提示 |
| copySelectedLinks 串行 presign N+1 | ✅ | `Promise.allSettled` 并行 |
| 面板挂载隐式切全局账号 | ➖ | 复核：onMounted/onActivated 仅 reload，不 `selectAccount` |
| `no-explicit-any:'off'` | ✅ | off → warn → **error**（0 违规） |
| X-Request-ID 超长回显 | ✅ | 长度 + 可见 ASCII 校验 |
| Bearer scheme 大小写敏感 | ✅ | `strings.Cut` + `EqualFold` |
| `metadata.go` 循环变量遮蔽 | ✅ | 循环变量改名 |
| `.env` 按 CWD 相对加载 | ✅ | `S3C_ENV_FILE` + 可执行文件同目录候选 |
| UserMessage 字符串匹配 `"exceeds 5GB"` | ✅ | sentinel + `errors.Is` |
| `validBucketName` 首/尾字符 | ✅ | 拒绝首尾 `.`/`-` 与连续分隔符 |
| 前端 `api.ts` 死代码 | ✅ | `requestBlob` / `downloadZip` 已删 |
| 同批 files 排序两次 | ✅ | 已移除重复排序 |
| 13 处 `ctx.account.value!.id` | ✅ | 收敛入口，剩余 3 处本轮清理 |
| blob 下载同步 `revokeObjectURL` | ✅ | 延迟 60s |
| `{} as KeyBindings` cast hack | ✅ | 改 `const bindings: KeyBindings = {}` + 可选字段 |
| pnpm 版本口径不一（9 vs 11） | ✅ | 全仓统一 `9.15.0`（`packageManager` + 各 workflow + Dockerfile）；`apps/desktop/package.json` 亦补 `packageManager`，由 `repo_infra_gate_test.go` 断言两个 package.json 与 CI 的 pnpm 版本一致 |
| Node 只 pin 大版本 | ✅ | 统一到 `24.21.0`（`node-version` / `node:24.21.0-alpine` / `node:24.21.0-bookworm` / NodeSource `nodejs=24.21.0-1nodesource1`），由 `TestNodePinnedToPatchVersion` 守住 |
| Rust 工具链未 pin | ✅ | `dtolnay/rust-toolchain` 由 `stable` 浮动分支 tip 改为版本分支 SHA（`ce678459… # 1.98.1`），新增 `apps/desktop/src-tauri/rust-toolchain.toml`（`channel = "1.98.1"`），GitLab 镜像 `rust:1.98.1-bookworm`；由 `TestRustToolchainIsVersionPinned` 守住 |
| Trivy 固定 0.58.1（落后 + `--vuln-type` deprecated） | ✅ | 升到 `0.74.0` 并加 digest 双 pin（`aquasec/trivy:0.74.0@sha256:62b1e65e…`），`--vuln-type` → `--pkg-types os,library`；由 `TestTrivyImageIsVersionAndDigestPinned` 守住 |
| `.dockerignore` 漏覆盖率/缓存目录 | ✅ | 补 `apps/web/coverage`、`.pnpm-store`、`test-results`、`playwright-report`、`.run/`、`.cargo/`、`.trivy-cache/`、`.gitlab-ci-local/`、`coverage.out`/`.txt`、`*.tsbuildinfo`、`*.log` 等；由 `TestDockerignoreExcludesBuildArtifacts` 守住（注释行不计入） |
| README 示例绑回环 | ✅ | `-p 127.0.0.1:8080:8080` + 说明 |
| nginx 容器缺内存限制 | ✅ | 两个 compose 均 `memory: 128M` |
| `\|\| true` 吞安装失败 | ➖ | 已无 install 命令使用；仅保留 read/kill 的良性回退 |
| Makefile 每次启动 `go mod tidy` | ✅ | 移除隐式 tidy，新增 `make tidy` |

### F. 历史版本功能与修复全量台账（0.1.0 → v1.0.0-rc1）

> 从 [`CHANGELOG.md`](../CHANGELOG.md) 的 22 个版本段完整汇总；逐字发布历史仍以 CHANGELOG 为准。

#### v1.0.0-rc1（2026-09-02）
- 桌面发版 CI：`release-desktop.yml` 在 `v*` tag 交叉构建 NSIS `.exe` / `.deb` / `.dmg` 并上传 GitHub Release。

#### v1.0.0-rc0（2026-09-02）
- 不做服务降级 / 无历史格式兼容：`/api/health` store 失败 503 + `status:error`；`encrypted` 仅 `S3C2`；JSON 账号文件 0600；移除兼容 shim 与旧后端注释。
- 异步 SSE：`copy-prefix/async`、`copy-objects/async`、`delete-prefix/async`；多 token 轮换（`S3C_TOKEN` 逗号分隔）。
- 防腐层与 service 收口：`UserMessage` / `HTTPStatus` / `IsNotFound` / `ObjectStream`；迁移引擎、ZIP、`JobRegistry` 迁出 handler；短写校验。
- 可观测：`GET /api/metrics`（Prometheus 文本）、`X-Request-ID`、可选 `S3C_LOG_JSON=1`。
- 安全：SSRF 禁重定向 + 拦截链路本地 / 云元数据；非回环无 token 拒绝启动；IP 令牌桶限速约 120/min；错误映射统一 `s3HTTPStatus`。
- 流式：写超时改 5 分钟滚动空闲；>5GB 迁移 multipart；同端点 `EntityTooLarge` 自动回退 multipart。
- 存储：`S3C_STORE_DRIVER=sqlite`（纯 Go `modernc.org/sqlite`，WAL）；`encrypted` + `S3C_STORE_KEY`（AES-256-GCM，Argon2id / `S3C2`）；移除遗留明文自动导入。
- 工程化：`s3wrap` 拆六文件（各 <400 行）；错误处理收敛；CI gofmt / race / cover / golangci + Web test / lint + E2E workflow；`release-version.sh`；`make test-all`。
- 前端：Escape 键栈、`KeepAlive max=5`、分段上传段级重试、ZIP File System Access 流式落盘、Hash 深链接、i18n 脚手架、token 可选 sessionStorage。
- 运维：`docker-compose.prod.yml`、TLS 叠加、nginx `proxy_buffering off`、health 探测 store、SQLite `user_version`、dependabot。

#### v1.0.0-20260901182023（2026-09-01）
- **Batch1 存储类型 + 版本比较 + 一键还原**：`HeadObject` 返回 `storageClass`；`POST /storage-class` 一键切换（标准 / 低频 / 单区 / 智能分层 / 归档）；版本比较（并排元数据 + 逐行内容差异，二进制或 >2MB 仅比元数据）；`POST /delete-marker/restore` 撤销删除。
- **Batch2 桶管理菜单**：7 页签（概览 / 生命周期 / SSE / CORS / 网站托管 / 桶策略 / 桶标签）+ 15 个 `Get/Put/DeleteBucket*` 端点；`isNoSuchBucketSetting` 把「未配置」映射为空响应。
- **Batch3 回收站菜单**：`GET /trash` + `POST /trash/purge`，列全部删除标记、一键还原、彻底清除（永删该 key 全部版本）。
- **四轴评审修复**：Vue `key` 为保留属性导致 7 个弹窗实际不可用 → 统一 `objectKey`（Vue 3.5 SSR 复现验证）；桶设置页签 `watch` 补 `immediate`（此前永不加载、保存会用默认值覆盖）；CORS 非白名单普通请求直接 403 + `readJSON` 强制 JSON；`inline` 代理 MIME 白名单；代理支持 `versionId` + `Accept-Ranges`/416；创建账号忽略客户端 id；ZIP 条目 `/` 与 `\` 双消毒；`.env` 加载；`PurgeObject` 改 `DeleteObjects` 批量 1000/批；`deletePrefix` 页前检查 + 跨页切分；multipart 段号 1..10000 且唯一校验；copy/migrate 4 路有界并发、migrate 限 10k key、failKeys ≤200<sup>†</sup>；列表导航序号防过期响应；上传入队捕获所属桶；焦点陷阱与初始焦点；版本比较下载走服务端代理。
  > <sup>†</sup> **该条当时并不成立**（2026-09-21 复核，docs/archive/review-2026-09-19.md §7.3 D4）：服务端 `BatchResult.FailKeys` 不做截断，只有异步 `delete-prefix` 在 handler 内裁剪，copy/migrate/sync 原样回传——10k 全失败会回约 10 MB 并落盘 `jobs.json`。**2026-09-22 已真正实现**：上限提为 handler 层共享常量 `maxFailKeys = 200` + `capFailKeys`，所有回传 `failedKeys` 的端点（copy-objects 同步/异步、migrate 同步/异步、migrate/sync、delete-prefix 同步/异步）统一裁剪，异步路径在 `jobResultFromBatch` 中于 `Finish`（落盘）之前裁剪；回归测试 `TestOlCopyManyFailKeysAll` / `TestOlMigrateSyncFailKeysCapped` / `TestOlMigrateAsyncFailKeysCappedBeforePersist`。
- 测试与清理：handler 覆盖率 60.3% → 70.8%；AWS SDK 类型外泄清理（`FromS3Object` / `FormatBuckets` / `DescribeACL` / `GranteeLabel`）；`filename*` RFC 5987；store 临时文件 `O_CREATE|O_EXCL`；删除死代码（`PresignGet`、`genSign`、`SignUrlDialog` 等）。

#### 20260901.2（2026-09-01）
- 对象版本：删除指定版本 `DELETE /version`、版本回滚 `POST /version/restore`（`CopyObject` 带 `?versionId=`，版本控制下写出一条新版本）；版本对话框「恢复 / 删除」+ 确认与结果提示；MinIO E2E 验证。

#### 20260901（2026-09-01）
- 桶属性 `GET /bucket-info`（区域 / 创建时间 / 版本控制）+ `PUT /bucket-versioning`（Enabled|Suspended）+ `GET /versions`（版本 + 删除标记 + 游标）；对象右键「版本」。
- 真实 MinIO E2E：预签名 PUT 直传、三段式 Multipart（12MB 组装回读一致）、`PutBucketVersioning` + 多版本覆盖写 + `ListObjectVersions`。
- 重构：`handler.go` 2186 → 131 行按域拆分；`ObjectsPanel.vue` 2041 → 442 行 + `useObjectBrowser` / `useObjectActions` / `usePreview`；`proxyUrl` 抽共享 `proxy.ts`；handler 测试拆分；新增 `agents.md`。
- 正确性：默认桶可空语义（`bucketOr`，缺省桶缺失返回 400）；`downloadZip` 条目脱敏 + 单次上限 1000；CSP；`ReadTimeout`（不设 `WriteTimeout`）。

#### 1.0.0（2026-08-28）
- 复制 / 移动 / 删除跨桶全闭环（`copy-object` / `copy-objects`；移动 = 复制成功后删源，文件夹副本有失败则保留源）；修复跨桶同名重命名被误拒。
- 大文件分段上传四端点（`init | part | complete | abort`），前端 ≥100MB 自动切 10MB/段、4 路并发，任一段失败自动 abort；小文件仍单 PUT。
- 对象 ACL（私有 ↔ 公共读 + 公开链接）、对象标签（键值编辑 / 一键清空 / 兼容 `NoSuchTagSet`）。
- 左侧菜单按「数据操作 / 配置」分组，首次按账号存在路由到对象管理。

#### 0.16.0（2026-08-28）
- 迁移优化：源 / 目标 Bucket 下拉、目标账号默认预选（含「（同账号）」标注）、每批 50 分批进度条、结果弹窗（成功 / 失败 / 总计 + 失败清单 ≤200 + 首个错误 + 「去目标账号查看」跳转）、已选数量与合计大小。

#### 0.15.0（2026-08-28）
- 行内常显「⋯ 更多操作」菜单（与右键菜单一致）；文件行操作列精简为「下载 / 预览 + ⋯」；文件夹行同样提供入口。

#### 0.14.0（2026-08-28）
- 行内「预览」按钮；双击 = 查看（文件预览 / 文件夹进入）；`Enter` 语义由下载改为查看（未知类型自动转下载）。

#### 0.13.0（2026-08-28）
- 编辑对象 HTTP 头（`CopyObject` 到自己 + `MetadataDirective: REPLACE`，保存后详情即时刷新）；生命周期规则管理（规则 ID / 前缀 / 过期天数整体保存；未配置返回空；清空走 `DeleteBucketLifecycle`）。

#### 0.12.0（2026-08-28）
- Bucket 列表页（名称 / 创建时间 / 进入 / 删除）；创建 Bucket（权限 + LocationConstraint，创建后自动进入）；删除 Bucket（非空 409）；对象页统计条；列表 / 网格切换；返回桶列表。

#### 0.11.0（2026-08-28）
- 通用 `ModalDialog` 组件（标题栏 + 关闭 + Esc / 遮罩关闭）；对象详情、签名 URL 结果、账号表单、服务端表单弹窗化。

#### 0.10.0（2026-08-28）
- 工具条拆「位置栏 + 操作栏」；文件管理器式行交互；键盘快捷键 `Enter` / `F2` / `Delete` / `Ctrl/Cmd+A` + 可关闭提示条；账号快速切换；统一 busy 防重复提交；空状态引导；错误条「重试」；跨面板联动 Toast；上传入口更名。

#### 0.9.0（2026-08-28）
- 安全预览：图片（含 SVG，`<img>` 上下文不执行脚本）/ 视频 / 音频（Range 拖动）/ PDF（sandbox iframe）/ 文本与代码 50+ 扩展名（转义展示、超大截断），未知格式转下载。
- 预览 / 下载统一服务端代理 `GET /proxy`：`download` 强制 attachment、`text` 强制 `text/plain + nosniff` 且服务端截断、文件名清洗防 `Content-Disposition` 注入；移除「在新窗口打开」；文本全程 `{{ }}` 转义。

#### 0.8.0（2026-08-28）
- 上传到当前目录（复用 presign PUT，2 路并发，内嵌进度）；面包屑可编辑直达深目录；图片预览；Shift 范围多选。

#### 0.7.0（2026-08-28）
- 服务商预设扩展（国内 COS / OBS / TOS / BOS / OSS / Kodo；海外 AWS / R2 / Wasabi / B2 / Spaces / Linode / Scaleway / Hetzner）+ 兼容 / 国内 / 国外三行分组。
- 批量 ZIP 打包下载（流式、不落盘，失败对象写入 `_下载失败清单.txt`）；递归删除文件夹（≤10 万对象、空前缀拒绝）；递归复制 / 移动文件夹（重叠前缀拒绝）。
- 新增 `delete-prefix` / `copy-prefix` / `download-zip` 端点。

#### 0.6.0（2026-08-28）
- 对象列表「加载全部」（循环分页 ≤200 页）；迁移面板「列出全部文件」（单页 1000、≤20 万对象、全选迁移）；上传完成一键复制 1 小时签名链接；`clipboard.ts` 共享模块。

#### 0.5.0（2026-08-28）
- 新建文件夹（PUT 空对象 + `/` 结尾）；重命名 / 移动（先复制后删源、支持跨桶）；对象详情（`HeadObject`）；批量复制签名链接；通用 `PromptDialog`。

#### 0.4.0（2026-08-28）
- 深色模式三态（跟随系统 / 浅色 / 深色）；自定义确认对话框替换原生 `confirm()`；对象右键菜单；双击交互；列排序（文件夹恒置顶）；本地即时过滤；记住上次账号；「清除已完成」。

#### 0.3.0（2026-08-28）
- OSS 式登录：服务商 + 区域预设（OSS 21 地域 / AWS 20 区域）自动填充 Endpoint 与公网 Endpoint；下载操作；3 路并发上传 + 失败重试。
- **修复跨 endpoint 明文 HTTP 流式迁移失败**：SDK 默认 CRC32 / payload SHA-256 对不可 seek 流报错 → `RequestChecksumCalculation=WhenRequired` + 预置 `UNSIGNED-PAYLOAD`；修复 SPA fallback 失效。
- 安全：Bearer 常量时间比较、`accounts.json` 0600、Update 持久化失败回滚；S3 client 连接 / TLS / 响应头超时 + 连接池上限。
- UI 改版：C 端视觉、CSS token 设计系统、`⚙ 设置` 弹层、连接状态灯、全局 Toast、面包屑 + 骨架屏、拖拽上传、响应式折叠；主题色科技蓝 → 薄荷绿。
- CI 首次接入；`/api/health` 返回版本号；基础安全响应头。

#### 0.2.0（2026-08-22）
- `internal/config` 集中配置 + `.env`；默认回环绑定 / CORS 白名单 / 可选 Bearer 鉴权。
- 容器化：多阶段 `Dockerfile`（非 root、HEALTHCHECK、`/data`）、`docker-compose.yml`、`-healthcheck` 子命令。
- 修复跨 provider 迁移误用 `CopyObject`；跨 endpoint 保留 Content-Type 与元数据；预签名 1s–7 天、`maxKeys` 1–1000；账号更新必填校验；账号存储原子写 + `Create` 失败回滚；region 缺省回退 `us-east-1`。
- 文档：`docs/api.md`、配置矩阵、运行 / 部署说明。

#### 0.1.0（2026-08-22）
- 首个版本：Go 后端（AWS SDK for Go v2，封装 11 个 S3 接口）、Vue3 + Vite + TS 前端（账号 / 列对象 / 直传 / 签名 / 删除 / 迁移 / 加前缀）、Tauri 2 桌面壳（无 IPC，B/S）。

### G. 已记录的 Unreleased 项（已完成）
运行时基镜像 Alpine 3.20、`/api/metrics` 默认关闭、S3C_TOKEN 短口令硬失败、S3 出站禁代理、桶名校验收紧、user metadata 400、delete-objects ≤1000、migrate SSE 写超时、store 失败回滚、错误 sentinel 化、X-Request-ID 加固、Bearer scheme、`.env` 解耦、nginx 内存上限、OpenAPI 自动生成 + 契约测试、**OpenAPI components 接线 `$ref`（109 处引用，消灭 0 引用死代码）**、双驱动去重、存储驱动再收敛（统一 `storeCodec`）、账号响应契约收敛（`secretSet` 替代 `"******"` 占位）、全仓 gofmt 对齐、Playwright E2E、增量同步、桶策略可视化编辑器、批量元数据编辑、存储类型切换、回收站、桌面端 SHA 校验与 actions pin SHA、**gitlab-ci-local 本地体验收口**（`.gitlab-ci-local-env` 默认挂 docker.sock、`make gcl*` pin `@4.75.1`、Trivy DB / 镜像源变量示例）等。完整逐条见 [`CHANGELOG.md`](../CHANGELOG.md)。

### H. 2026-09-16 评估 P0 发布阻塞修复

> 来源：[`archive/assessment.md`](archive/assessment.md) §六 P0（H1 / H4 / M8+C2 / S7+C3）。对应 KNOWN_ISSUES #5 / #6 / #7 / #15，均已归档。

| # | 项 | 状态 | 修复内容 |
|---|----|------|----------|
| P0-1 | OpenAPI 契约与真实 handler 字段级不一致（H1） | ✅ | `/api/migrate`、`/api/migrate/async` 请求体字段改为 handler 实际解析的 `sourceAccountId` / `sourceBucket` / `sourceKeys` / `targetAccountId` / `targetBucket` / `targetPrefix`（删除虚构的 `deleteSource` / `storageClass`）；presign `method` 枚举 `GET/PUT/DELETE/HEAD` → `get/put/post`；delete 移除 `versionId`；multipart/part 补 `expiresIn`。新增 `TestOpenAPI_ContractRequestBodyMatchesHandlers` 做「registry schema ↔ handler DTO」字段级三方一致性兜底（含 `$ref` 解引用与 enum 断言） |
| P0-2 | 前端 Token 明文双份落 `localStorage['s3c.servers']`（H4） | ✅ | `s3c.servers` 只存 `{id,name,base}`；token 单副本按 `s3c.token.<serverId>` 独立存储（默认 sessionStorage，仅显式「跨会话保留」时写 localStorage）。旧版本内嵌 token 首次读取时一次性迁移并回填活动服务器 profile；删除服务器同步清理 per-server token |
| P0-3 | `loadAll()` 在 `nextToken` 为空时不发请求却误报「已加载全部」（M8 / C2） | ✅ | 改为 do-while：`nextToken` 为空时仍加载第一页（reset 语义，替换旧列表避免重复项）；某页失败即停止续页且不再发成功 toast；`MAX_ALL_PAGES` 上限保留 |
| P0-4 | 批量复制/移动/删除 SSE 终态检测导致 Promise 悬挂、`opsBusy` 永不复位（S7 / C3） | ✅ | `subscribeMigrateEvents` 在流 EOF 且未收到终态时轮询 `migrateJobStatus`（500ms 间隔、30s 上限）直到 done/cancelled，并在回读结果上合成终态 `status`；连续 3 次回读失败快速 `onError`。三个调用方（`ctxDeleteFolder` / `DestDialog` / `MigratePanel`）由此保证拿到终态或错误，不再永久禁用按钮 |

### I. 2026-09-16 P1 稳定版门槛修复

> 来源：[`archive/assessment.md`](archive/assessment.md) §六 P1 与 §二 L4 / S1。对应 KNOWN_ISSUES #13 / #14 / #19 / #24，以及 P0-1 的契约收尾。
> 修复后 `govulncheck ./...` 由 6 个可达 stdlib 漏洞降为 **0**；全仓 action SHA 经 GitHub API 核验均有效。
> **P1 至此清零**，`v1.0.0` 稳定版门槛达成。

| # | 项 | 状态 | 修复内容 |
|---|----|------|----------|
| P1-1 | Go 1.26.5 的 6 个可达 stdlib 漏洞（H2） | ✅ | `apps/server/go.mod` 与 `apps/server/Dockerfile` 升至 1.26.6（CI 经 `go-version-file` 自动跟随）。CI 在 `go vet` 后新增固定版本 `govulncheck@v1.8.0` 门禁，补上 Trivy 只扫 OS/库、拦不住标准库 CVE 的盲区 |
| P1-2 | workflow 幽灵 action SHA（H3，实际 2 处） | ✅ | `e2e-playwright.yml` 的 `pnpm/action-setup` 与 `actions/upload-artifact` 改为正确 SHA；全仓 10 个 action SHA 经 GitHub API 逐一核验 |
| P1-3 | 3 个 i18n 键被引用但未定义（L4 / R1） | ✅ | 补齐 `objects.toastCopyFailed` / `batchEdit.tagsNeedKey` / `common.working` 的 zh-CN / en-US 文案；新增 `src/i18n/coverage.test.ts` 静态扫描（`import.meta.glob` 读源码，不引入 `node:*` 依赖），已验证删键即失败 |
| P1-4 | `delete-marker/restore` 契约字段漂移（P0-1 收尾） | ✅ | 请求体 `deleteMarkerId` → `versionId`（对齐 `restoreDeleteMarker` handler 与 `docs/api.md`）；契约测试扩展至 `delete-marker/restore` / `version`(DELETE) / `version/restore`，并验证回退修复时测试确实失败 |
| P1-5 | 异步任务丢失无法恢复（S1，v1.0.0 最后一项门槛） | ✅ | ① `service/job_persist.go`：`JobPersister` 抽象 + `FileJobPersister` 原子写（临时文件 → rename → 0600），`Create`/`Finish` 必落盘、中间进度 2s 节流；② 启动恢复：非终态任务标记 `interrupted` 并回写，`NewJobRegistry()` 保持纯内存语义不破坏既有测试；③ `interrupted` 独立 7 天保留期（`JobInterruptedTTL`），修复「恢复后首次 reap 即被 30 分钟 TTL 清除」的缺陷（有回归测试）；④ 新增 `GET /api/migrate/jobs`（路由数 69 → 70，OpenAPI 同步）；⑤ 前端 `MigratePanel` 新增「未完成任务」区块，提示移动任务「已复制但源未删除」并可逐条忽略；⑥ `main.go` 注入 `dataDir/jobs.json`。落盘未复用 `store/atomic.go`：`service→store` 会形成分层倒置（技术策略一致，已在 `architecture.md` 记录） |

### J. 2026-09-16 P2 加固修复（第一批）

> 来源：[`archive/assessment.md`](archive/assessment.md) §二 L1/L2、§二 S2、§二 M4/M6。对应 KNOWN_ISSUES #17（部分）/ #18（部分）/ #20 / #23（部分）。

| # | 项 | 状态 | 修复内容 |
|---|----|------|----------|
| P2-1 | 预签名错误被吞（L1） | ✅ | `objects.go` 三处 + `multipart.go` 一处 `u, _ := client.PresignXxx(...)` → 统一 `writePresignResult`，失败 500 而非 `200 {"url":""}`；新增源码级门禁 `TestPresignErrorsNotSwallowed`（逐行剔除注释后匹配，回退即失败） |
| P2-2 | 流式传输错误被静默吞（S2） | ✅ | `copyStream` 返回 `(int64, error)`；`recordStreamOutcome` 区分「真实中断」（Warn + `s3c_stream_interrupted_total`）与「客户端主动断开」（Debug，不计数） |
| P2-3 | JobRegistry 无总上限（M4） | ✅ | 新增 `TryCreate` + `ErrTooManyJobs`；上限 `maxJobs = 256` 只统计未终结任务（避免恢复的 interrupted 任务永久占满）；4 个异步端点超限返回 503 并释放 ctx；`Create` 保持原签名 |
| P2-4 | TLS 前置无 HSTS / Permissions-Policy（M6） | ✅ | TLS 示例配置补 `Strict-Transport-Security`（180 天）+ `Permissions-Policy`（关闭定位/麦克风/摄像头/支付/USB/interest-cohort） |
| P2-5 | 后端死代码（D1-D4） | ✅ | 删 `ctxReader`（与 `ctxCancelReader` 职责重复、零生产引用）、`batchItemError`（生产零引用）、`Client.S3()`（导出零生产调用，E2E 清理改用已有 `DeleteObjectVersion`/`DeleteObject`）；**`isNoSuchBucketSetting` 经核验有 5 处生产调用，非死代码，保留** |
| P2-6 | compose 未启用结构化日志（S4） | ✅ | `docker-compose.yml` / `docker-compose.prod.yml` 注入 `S3C_LOG_JSON: "${S3C_LOG_JSON:-1}"`，可设 0 覆盖；`.env.example` 同步说明；经 `docker compose config` 验证默认 1、可覆盖 0 |

**未纳入本轮**（评估为需更大改动或属行为变更）：
- **Argon2 `t=1` → `t≥2`**：加密文件格式只存 `S3C2` magic + salt，**不存 KDF 参数**。直接改 `argonTime` 会让所有既有 `accounts.json.enc` / `accounts.db` 无法解密。需先引入带参数的新格式版本（如 `S3C3`）并实现双版本读取迁移，属独立工作项。
- **`/api/health` 移除 `version`**：前端不消费该字段，但移除会改变既有响应契约（`accounts_test.go` 已断言其存在）。保留是运维定位版本的实际需要，且该端点通常位于内网或鉴权之后。
- **compose 默认改 encrypted**：会改变默认部署行为并涉及密钥管理（`S3C_STORE_KEY` 分发），需配套部署文档与升级说明。

### K. 2026-09-17 P2 加固修复（第二批）

> 来源：[`archive/assessment.md`](archive/assessment.md) §二 D5/D6。对应 KNOWN_ISSUES #10。

| # | 项 | 状态 | 修复内容 |
|---|----|------|----------|
| P2-7 | endpoint 归一化三份实现且行为不一致（D5） | ✅ | `s3wrap/client.go` 的 `normalizeEndpoint` 改为导出的 `NormalizeEndpoint` 并修正语义：**去首尾空白 + scheme 大小写不敏感 + host 小写 + 保留路径前缀**。旧实现只做大小写敏感前缀判断，把 `"HTTP://MinIO:9000"` 拼成损坏的 `"http://HTTP://MinIO:9000"`。`service/migrate.go` 删除本地副本改为复用；`s3wrap/presign.go` 的 `PublicURL` 同样复用，修掉 `" http://a.com/ /b/k"` 这类带空格的链接。`TestNormalizeEndpoint` / `TestPublicURL` 表驱动锁定，含路径前缀保留用例 |
| P2-8 | `ValidateEndpoint` 用未 trim 原串解析（D5 连带） | ✅ | 该函数此前 `strings.TrimSpace` 仅用于判空、解析仍用原串，导致用户手填的 `" http://127.0.0.1:9000 "` 被误判 `invalid endpoint URL`。改为先归一化再解析。**同时保持 fail-closed**：非空输入归一化后退化（`"http://"`、`"/"`、`"https:///"`）仍报错，不因归一化静默放行 —— 已加回归用例。E2E 实证：畸形 endpoint 建账号后 `preview-buckets` 由 400 `invalid account configuration` 变为通过校验进入网络调用，日志中 BaseEndpoint 为干净的 `http://127.0.0.1:9000` |
| P2-9 | handler 侧 `derefString`/`derefInt64`/`timeOrZero`/`boolOrFalse` 死代码（D6） | ✅ | 4 个包装在生产代码零引用，仅 `TestAccDerefHelpers` 引用（覆盖率注水样本）。连同该测试一并删除。`s3wrap/helpers.go` 与 `s3wrap_dto.go` 中的同名 helper **保留** —— 分别有 29/1/5/2/4 处生产调用（`derefString`/`derefInt32`/`boolOrFalse`/`derefInt64`/`timeOrZero`），属包内合理复用，非重复 |

### L. 2026-09-17 目录迁移后的可移除项清理

> 来源：[`archive/assessment.md`](archive/assessment.md) §二 D7/D8/D10。对应 KNOWN_ISSUES #11。

| # | 项 | 状态 | 处理内容 |
|---|----|------|----------|
| L1 | i18n 死键（D7） | ✅ | 全量扫描确认 **26 个**键中英各 26 行从未被引用（`app.name`、`common.back/confirm/danger/default/done/empty/failed/none/search/selected/test/upload/waiting`、`objects.bucket/prefix`、`toolbar.delete/downloadZip/filter/refresh`、`trash.empty`、`upload.queue`、`server.save`、`batchEdit.aclNoChange/storageNoChange`、`bucketTags.errEmptyKey`），字典键数 695 → 669。**根因**：既有门禁只校验「被引用 → 有定义」，无人校验反向，且 `i18n/**` 被排除在覆盖率统计外。已在 `i18n/coverage.test.ts` 增加**反向门禁**——字典中任何既非字面量引用、也不匹配动态拼接模式的键都会让测试变红；`storage.class.*` / `provider.*` 动态键按「字面命名空间前缀 + 插值」形状生成豁免正则。**关键点**：豁免必须限定该形状，否则 `/api/accounts/${id}`、`Bearer ${token}` 这类非 i18n 模板串会生成几乎匹配任意键的正则，让门禁恒真（初版即踩此坑，实测由「26 死键」退化为「0 死键」） |
| L2 | `@vitest/coverage-v8` 冗余依赖（D10） | ✅ | `vite.config.ts` 的 provider 为 `istanbul`，v8 provider 从未启用。已从 `package.json` 移除并重算锁文件（`pnpm install --frozen-lockfile` 通过；锁内残留条目仅为 vitest 的 optional peer 声明，不再安装）。四指标覆盖率复测仍 100% |
| L3 | `ObjectList.vue` 未使用的 `visibleCount` prop（D8） | ✅ | 父组件 `ObjectsPanel.vue` 传入、组件内从不读取（空态判断实际用 `totalCount`）。连同父组件绑定与 `ObjectList.test.ts` fixture 一并删除。`ObjectToolbar` 的同名 prop **保留**——真实用于 `{{ visibleCount }}/{{ totalCount }}` 计数展示。**剩余**：grid 视图 `v-for` 全量渲染未窗口化（D8 另一半） |
| L4 | `.gitignore` 冗余与无关条目 | ✅ | 202 行 → 80 行。删去 28 行与本站技术栈无关的脚手架模板条目（Bower / jspm / Snowpack、Next/Nuxt/Gatsby、SvelteKit、Docusaurus、VitePress、Serverless、FuseBox、DynamoDB、Firebase、Yarn、`.tern-port` 等），并按**实测**（逐条注释该规则后 `git check-ignore` 复查）删除 4 条被通用规则覆盖的冗余项（`apps/server/data/` ← `data/`；`apps/web/node_modules/`、`apps/desktop/node_modules/` ← `node_modules/`；`apps/web/dist/` ← `dist/`）。删除前后 `git status --ignored` 忽略集合逐行比对一致，唯一差异是把过窄的 `.cache/` 收敛为 `.cargo/`（前者会顺带忽略任意层级 `.cache/`，后者才是桌面端构建真实产物；已确认 `.cargo/` 下无可提交的 `config.toml`）。已确认无任何**已跟踪**文件落入新规则 |

### M. 2026-09-17 路线图 v1.0.0 / v1.0.x / v1.1.0 收口（#1–#8、#10、#11）

> 来源：[`ROADMAP.md`](ROADMAP.md) 三～五章。除 #12（桌面端签名）外的开放条目全部落地。

| # | 条目 | 状态 | 实现与验证 |
|---|------|------|------------|
| 1 | `docs/api.md` 自动化校验 | ✅ | 新增 `handler/api_doc_test.go`：从 `docs/api.md` 解析 `METHOD /api/...` 行，与 `routes.go` 注册表做**双向 diff**（文档多写 / 少写均红灯）。已注入漂移实测两侧变红后还原 |
| 2 | SQLite 密钥加密 + Argon2 加强 | ✅ | 新加密格式 **S3C3**：`magic + time(4,BE) + memory(4,BE) + threads(1) + salt(16) + ciphertext`，把 KDF 参数写进文件头，读取按文件参数派生 → 可在不影响既有库的前提下调参。`argonTimeV3=2`（OWASP 建议 ≥2）。**双版本读取**：S3C2 旧库（t=1）仍可解。`SQLiteStore` 新增 `storeKey`，`secret_key` 列以 S3C3 密文落盘、读时解密、历史明文行原样兼容；无 key 读密文行显式报错。`config.MinStoreKeyLength = 16` 校验 `S3C_STORE_KEY` |
| 3 | 安全审计日志 + XFF 可信代理 | ✅ | 新增 `handler/audit.go`：稳定事件常量 + `h.audit(r, event, ...)`，覆盖 401（含 malformed/bad_token 原因）、账号 CRUD、桶策略设置/清除、对象删除/前缀删除、回收站清空、限速命中。`clientIPWithProxies` 仅在直连对端命中 `S3C_TRUSTED_PROXIES` 时才采信 XFF 首段，否则回退 `RemoteAddr`（防伪造 XFF 绕过限速，已有集成用例）。`S3C_TRUSTED_PROXIES` 默认空 |
| 4 | ZIP 部分失败可见 | ✅ | `zip.go` 不再丢弃 `WriteObjectsZip` 的 `failKeys`：部分失败落 Warn 日志并计入 `s3c_zip_partial_failures_total` / `s3c_zip_failed_keys_total`，整体失败计入 `s3c_zip_failed_total` |
| 5 | S3 上游指标 | ✅ | 新增 `s3wrap/metrics.go`：smithy `Finalize` 中间件采集调用数、耗时直方图（11 桶）、错误按码分类（API code / `canceled` / `timeout` / `transport`，基数有界）、流字节数。`/api/metrics` 输出 `s3c_s3_*` 系列；`copyStream` 回传字节数计入 `s3c_s3_stream_bytes_total` |
| 6 | 错误文案不回显用户输入 | ✅ | `headers.go` 的 `ValidateUserMetadata` 错误改为固定文案 + `Debug` 日志（原样回传含用户 key 的错误串）；`metadata.go` / `objects.go` / `multipart.go` 的 `unsupported acl: <值>`、`duplicate tag key`、`unsupported storageClass` 等一并改为固定文案。新增 `error_echo_gate_test.go` 源码级门禁防复发 |
| 7 | grid 视图窗口化 | ✅ | `ObjectList.vue` 的 grid 分支改为渲染 `gridItems`（上限 300 条），超出时显示截断提示（`objects.gridTruncated`）；列表视图此前已有窗口化 |
| 8 | 前端健康轮询与自动恢复 | ✅ | 新增 `composables/useHealthPoll.ts`：后端出错后每 5s 探测 `/api/health`，成功即回调 `loadAccounts` 重载并清除错误横幅；未出错时不轮询。`useBucketSetting.reload()` 加 seq 竞态守卫（旧请求的失败/loading 不覆盖新请求） |
| 10 | `entries` / `visibleEntries` 单次排序 | ✅ | `useObjectBrowser.ts` 抽出 `compareEntries`，`entries` 一次排序（文件夹恒在前），`visibleEntries` 只做过滤、保持顺序，消除过滤态下的重复排序 |
| 11 | 覆盖率门禁去「注水」 | ✅ | 前端 `vite.config.ts` 不再整体排除 `src/i18n/**`，只排除纯数据模块 `src/i18n/messages/**`；`index.ts` 纳入统计后补齐 `readLocale` 回退/异常、`setLocale` 写入失败、`cycleLocale`、`locale()`、`i18nKeyCount` 缺省参数等行为测试，四指标仍 100%。后端把确实不可达的防御分支删除（`encryptSecret` 的 AES 错误分支、`os.MkdirAll` 冗余判断），改为行为断言而非 gap 测试；`make test-cover` 的 `count==0` 检查归零 |

---

### N. 2026-09-19 明文落盘启动告警与旧编号引用清理

| # | 条目 | 状态 | 实现与验证 |
|---|------|------|------------|
| 1 | 落盘明文启动告警 | ✅ | `config.Config.StorePlaintextWarning()`：`json` / `sqlite` 驱动且 `S3C_STORE_KEY` 为空时返回可执行文案（含驱动名、`DataDir`、`encrypted` 建议与最短长度），其余配置（含未知驱动）返回空串；`runServer` 在 `Validate` 之后以 `logger.Warn` 输出。先写失败测试（`TestStorePlaintextWarning` 六例表驱动）再实现；`TestMainServerWarnsPlaintextStore` 用 fork 子进程跑完整 `main()`，断言 `sqlite` + 空 key 启动日志确实出现「明文落盘」，加密配置不告警。对应风险 [ROADMAP.md](ROADMAP.md) §5.1「已收敛」索引 R3（启动告警守卫）。**2026-09-20 强化**：该告警仅在显式 `S3C_ALLOW_PLAINTEXT_STORE=1` 放行时才会出现——默认改为硬失败，见 §T |
| 2 | 旧 `roadmap #N` 引用清零 | ✅ | 2026-09-17 收口后，代码注释里仍留 **27 处**（25 个文件）指向已归档编号的 `roadmap #N`（`docker-compose.yml`、`config.go`、`store/*`、`handler/*`、`s3wrap/*`、`apps/web/src/*`）。按语义改写：带 `ASSESSMENT` 编号的只保留该编号；纯 roadmap 编号的改为「已闭环：FEATURES.md §M」；`docker-compose.yml` 的密钥加密说明改指 `FEATURES.md` §M + `ROADMAP.md` §5.1「已收敛」索引 R3。全仓 `grep -rn "roadmap #[0-9]"`（除 `CHANGELOG.md` 的历史记录）为零，编号引用规则见 [ROADMAP.md](ROADMAP.md) §六 第 6 条 |

---

### O. 2026-09-19 风险登记集中处置（R1 / R2 / R4 / R6 / R7 / R8）

> 来源：[`ROADMAP.md`](ROADMAP.md) §5.1 风险登记（R5 桌面签名经决策暂不处理）。每条都补了
> 可复现的门禁或测试，收敛后移入该节末尾的「已收敛」索引（R8 维持 ➖ + 可选加固）。

| # | 条目 | 状态 | 实现与验证 |
|---|------|------|------------|
| 1 | R1 契约字段级门禁 + 三处注册表失真修复 | ✅ | 新增 `TestAPIDocDocumentsRequestBodyFields`（`api_doc_test.go`）：OpenAPI 每个 `requestBody` 的字段名必须出现在该端点 `docs/api.md` 正文，支持「请求体与 `POST /api/x` 相同」别名（`apiDocSections` / `sectionText`）。门禁上线即抓到三处**客户端可见**失真：`mkdir` 注册表写 `prefix`（handler/frontend/文档都是 `key`）、`copy-objects` 写 `items`（真实是 `keys`）、`DELETE /api/accounts/{id}/version` 把 query 参数误声明成 requestBody。已修注册表并加回归断言（`TestOpenAPI_ContractRequestBodyMatchesHandlers` 第 7 项 + `requestQueryParams`），`docs/api.md` 补齐当时判定为漏记的字段。**事后更正**：其中只有 `publicEndpoint` / `metadata` 是真漏记，`provider`/`replaceTags`/`cacheControl`/`contentEnc`/`contentLang`/`disposition` 六个字段在 handler 中**并不存在**（反向漂移：文档与注册表凭空多写），按文档示例体调用会因 `DisallowUnknownFields` 直接 400。2026-09-19 P0-5 已把这六个幻影字段从注册表与 `docs/api.md` 删除，并把该端点族的断言改成**字段集完全相等**（第 8 项），同时补上真漏记的 `useSSL`。另加**通用输入源门禁** `TestOpenAPIRequestDeclarationMatchesHandlerInput`：解析 `routes.go`（含 `h.withStreamLimit(h.x)` 包装）→ handler 方法调用闭包找 `readJSON` / `json.NewDecoder` ⇔ 注册表 `Request:` 双向比对，覆盖全部 70 条路由；已用「给 `GET /api/health` 注入假 requestBody」的变异验证过会红灯后还原 |
| 2 | R2 消音式死代码门禁 | ✅ | 实测 `golangci-lint`（`run.tests: true`）**能**报未引用的测试函数/方法，盲区只有显式消音。新增 `apps/server/deadcode_gate_test.go` 扫全仓 Go 源码禁止 `_ = ident` 与 `var _ = expr`（允许 `_ = f()` 这类显式忽略返回值），并自检扫描文件数避免路径写错静默变绿。清掉 11 处遗留消音（`acc_ratelimit_test.go` 的 `var _ = errors.New`、`bs_gap_test.go` 的只声明不用的 `deletes`、`object.go` 里 `PurgeObject` 的冗余 `_ = keyMarker` 等） |
| 3 | R4 DataDir 单写者锁 | ✅ | 新增 `store.AcquireDataDirLock`：unix `flock(LOCK_EX\|LOCK_NB)` 锁 `<DataDir>/.s3client.lock`，第二个实例启动即失败（`runServer` 返回 1）；内核在进程退出时释放，无陈旧锁。`main.go` 在开 store 前加锁。测试：`TestDataDirLockExcludesSecondInstance`（互斥 + 释放后可重加）、`TestDataDirLockErrors`（目录不可建 / 锁文件被占）、`TestRunServerRejectsLockedDataDir`（端到端拒绝启动）；`GOOS=windows go build ./...` 通过（非 unix 为文档化 no-op） |
| 4 | R6 RustSec 审计入 CI | ✅ | `cargo audit`（pin `cargo-audit 0.22.2`）加入 GitHub `ci.yml` 与 GitLab `desktop` job，新增本地 `make rust-audit`。实跑结果：**0 个漏洞**，7 条 unmaintained / unsound 告警（`proc-macro-error`、5 个 `unic-*`、`glib 0.18.5`）逐条 triage——均为上游未发修复版本的传递依赖，默认退出码策略是「漏洞红灯、告警可见」，不用 ignore 清单掩盖 |
| 5 | R8 可选 SSRF 加固 | ✅ | 新增 `S3C_SSRF_DENY_PRIVATE`（默认关闭，保持 [ADR-003](decisions/0003-ssrf-private-allow.md) 自托管放行）：置位后 `isBlockedIP` 连 RFC1918 / ULA / 回环 / 未指定地址一并拒绝，创建期与拨号期双重校验同时生效。测试：`TestDenyPrivateNetworksOptIn`（默认放行 vs 开启后 6 类地址全拒、公网仍放行）、`TestDenyPrivateNetworksDialGuard`（拨号期）、`TestFromEnvSSRFDenyPrivate`。ADR-003 增加 Update 段记录该开关 |
| 6 | R7 分段上传 ETag 失败路径可见 | ✅ | 新增 `upload.test.ts` 用例：分段 PUT 返回 2xx 但缺 `ETag` → 报「未读取到 ETag」并在组装前 `multipartAbort`（不留半成品）。README 补多厂商 CORS 兼容性矩阵（RustFS 有真对端 E2E，MinIO / AWS S3 / 阿里 OSS / 腾讯 COS 标注手动），明确 `ExposeHeader: ETag` 是分段组装硬前提 |
| 7 | R8 存储硬失败可观测 | ✅ | `/api/metrics` 新增 `s3c_store_up` gauge（store 掉线为 0），与既有 `/api/health` 503、`TestHealthStoreUnavailable` 一起构成 ADR-002 的可告警面；`TestMetricsStoreUpAndSSRFPolicy` 覆盖 1/0 两态。[DEPLOYMENT.md](DEPLOYMENT.md) §6.1 补 503 处置顺序与告警建议、§6.3 与 [api.md](api.md) 指标清单同步 |
| 8 | R8 SSRF 生效策略可见 | ✅ | 启动日志新增 `ssrfDenyPrivate` 字段；`/api/metrics` 新增 `s3c_ssrf_deny_private` gauge（读新增的 `s3wrap.DenyPrivateNetworks()`），运维可核对 `S3C_SSRF_DENY_PRIVATE` 是否生效；`TestMetricsStoreUpAndSSRFPolicy` 覆盖 1/0 两态 |

---

### P. 2026-09-19 分支状态审查 A1–A3 处置（前端 API 拆分 / 死代码门禁 / 测试接缝）

> 来源：2026-09-19 分支状态审查 [`docs/archive/review-2026-09-19.md`](archive/review-2026-09-19.md) 的架构项 A1–A3 与
> 浏览器 E2E 空转用例（已在审查文档中销项——该文档按约定只保留**未闭环**的发现）。归档证据即本表。
> 该审查登记的 P0/P1 缺陷（B1 SSE 自旋、B2 SyncKeys 前缀、F1 白名单 crash、R1 CI docker job、
> §7.2 契约幻影字段等）**在本节写作时点（2026-09-19）仍未处置**，不在本节范围内——
> 其后由 §Q（P0）、§R（§三 正确性）、§S（P1）逐节闭环，归档审查亦记「P0 / P1 全部清零」。

| # | 条目 | 状态 | 实现与验证 |
|---|------|------|------------|
| 1 | A1 `api.ts` 838 行单文件拆分 | ✅ | 拆为 `apps/web/src/api/` 下 7 个模块：`storage.ts`(339) / `endpoints.ts`(192) / `jobs.ts`(119) / `download.ts`(85) / `index.ts`(82) / `upload.ts`(48) / `http.ts`(38)。单向依赖 `index → {endpoints,jobs,download,upload} → http → storage`，无环；`./api`、`../api` 仍解析到 `api/index.ts`，**import 路径与对外契约不变**（用 `git mv api.ts api/index.ts` 保留历史）。顺带消除 `migrateJobStatus` 在 endpoints/jobs 各写一份的重复定义。**纯搬运**：`fetch` headers 合并顺序等既有语义原样保留（审查 §F10 潜伏缺陷不混入）。验证：`vue-tsc` / `eslint` 通过、986 例（64 文件）单测全绿、覆盖率四指标 100%（statements 3941 / branches 2787 / functions 1082 / lines 3392，7 个新模块全部满覆盖）、Playwright 15 passed / 0 skipped |
| 2 | A3 前端死代码门禁 | ✅ | 新增 `apps/web/src/deadcode_gate.test.ts`：API 公开面（`s3api.*` / `api.*` / 具名导出）每个成员必须在**至少一个非测试源文件**中被引用。eslint/vue-tsc 只报未使用的局部变量，看不见导出符号，而测试引用会让死代码"活着"——这正是本项要堵的洞。用 `import.meta.glob(?raw)` 读源码（不引入 `node:fs`/`@types/node`）；按 **import 别名精确匹配 `别名.成员`**，不做裸词/子串匹配（避免 `'migrate'` 字面量与 `migrateAsync` 前缀碰撞假绿）；三重自检（扫描文件数 ≥50 / 公开面 ≥50 / 引用总数 ≥30）防"空跑变绿"；`INTENTIONAL_UNUSED` 豁免清单附"豁免不得腐烂"检查。**变异验证**：注入 `s3api.zzDeadProbe` → 红灯；复刻历史缺陷（`s3api.copyFiles` 仅被 `api.test.ts` 调用）→ **仍红灯**（测试引用不算使用） |
| 3 | A3 清理零生产调用的公开面 | ✅ | 删除 5 个 `s3api` 方法（`copyFiles` / `deletePrefix` / `copyPrefix` / `migrate` / `migrateSync`，全仓仅测试引用；对应后端端点保留）与 2 个无用 barrel 导出（`requestResponse` 仅 `download.ts` 内部使用、`downloadZipToDisk` 生产只走 `s3api.downloadZipToDisk`） |
| 4 | A2 生产包剔除测试专用接缝 | ✅ | `s3wrap.ResetMetrics`（导出且自称"仅测试使用"）→ 移入 `metrics_test.go` 内的 `resetMetrics()`，不再进入生产二进制；`service.SetMaxJobsForTest` 删除，在册上限由**可变全局** `var maxJobs` 改为**每实例**构造期选项 `NewJobRegistry(WithMaxJobs(n))`（原写法跨用例/跨实例污染，并发下即数据竞争），handler 侧接缝移到 `internal/handler/export_test.go`（仅 `go test` 编译）。以行为用例替换原「钩子返回旧值」用例：`TestWithMaxJobsLimitsRegistry`（上限生效 + 实例间互不影响）、`TestWithMaxJobsIgnoresNonPositive`（非正上限回落默认） |
| 5 | 浏览器 E2E 空转用例 | ✅ | `account-flow.spec.ts` 两个用例由「1 skipped + 1 个断言自身 fixture」改为真实 UI 驱动（`role=button` 入口、空态文案、新增表单 → POST 请求体 → 列表渲染；以及 `/api/accounts` 失败 → 断连徽标 → `/api/health` 恢复后自动重连）。Playwright 15 passed / **0 skipped** |

---

### Q. 2026-09-19 分支状态审查 P0 处置（5 项全部闭环）

> 来源：[`docs/archive/review-2026-09-19.md`](archive/review-2026-09-19.md)。该审查登记的 P0 已全部处置并从审查文档移除
> （该文档只保留未闭环发现）；本表即归档证据，逐条改动见 [`CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]`。
> 剩余的 P1/P2 发现**在本节写作时点（2026-09-19）仍开放**——其后 P1 由 §S（2026-09-19）、
> P2 由 §T（2026-09-20）与 §U（2026-09-22）全部闭环，归档审查记「P2 已清零」。

| # | 条目 | 状态 | 实现与验证 |
|---|------|------|------------|
| 1 | P0 SSE 对 `interrupted` 任务无限自旋 | ✅ | `case p, open := <-ch: if !open { return }` + `service.IsTerminalJobStatus(p.Status)`；SSE 路由补 `withStreamLimit`；`Job.Subscribe` 加每任务订阅上限 16（饱和回 503）。测试 `TestMigrateJobEventsInterruptedStreamCloses`（真实 `jobs.json` 恢复路径 + 真 HTTP 流，断言只 1 帧且 EOF）/ `TestMigrateJobEventsSubscriberCap`；前者经「还原旧循环」变异验证会红灯 |
| 2 | P0 `s3c.servers` 坏数据整站白屏 | ✅ | `sanitizeServer()` 逐元素形状校验 + 回写修复清单；`getActiveServer()` 改为渲染期安全访问器。测试：`api.test.ts` 5 种坏存储 + 混合条目，`App.corruptStorage.test.ts` 用真实 `./api` 挂载真实 App 断言不白屏 |
| 3 | P0 `SyncKeys` 目标 key 映射错误、永不收敛 | ✅ | 抽出复制内核 `migrateKeys(..., dstKeyFor, ...)`，`MigrateKeys` / `SyncKeys` 共用；`SyncKeys` 只留一个映射表达式；`stripPrefix` 补段边界校验。测试 `TestSync_PrefixMappingConverges` / `TestSync_PrefixSegmentBoundary` / `TestMigrateSync_PrefixFilter`（含二次同步 `copied:0`） |
| 4 | P0 GitHub CI `docker` job 结构性失败 | ✅ | `build-push-action` 加 `load: true` + 新增 `docker image inspect` 前置步骤。本地 `docker-container` builder 复现：不加 `--load` 构建 exit 0 但镜像不在 daemon，加 `--load` 后可见 |
| 5 | P0 OpenAPI 契约幻影字段（6 个） | ✅ | `provider` / `replaceTags` / `cacheControl` / `contentEnc` / `contentLang` / `disposition` 从注册表与 `docs/api.md` 删除（无实现意图，前端亦无对应入参）；补上真漏记的 `useSSL`。`TestOpenAPI_ContractRequestBodyMatchesHandlers` 第 8 项断言四个端点的**字段集完全相等** |

---

### R. 2026-09-19 分支状态审查 §三 正确性处置（后端 B3–B11 / 前端 F2–F10）

> 来源：[`docs/archive/review-2026-09-19.md`](archive/review-2026-09-19.md) 的「阶段 2 · 正确性」全部发现项。该审查文档按约定
> 只保留**未闭环**发现，本节即归档证据，逐条改动见 [`CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]`。
> 原则：**补门禁而不只是补缺陷**——每条修复都配了会先失败的回归测试，覆盖「分支/语义正确但断言缺席」这类盲区。

| # | 条目 | 状态 | 实现与验证 |
|---|------|------|------------|
| 1 | B3 `DeleteObjects` 吞掉 200 响应体内的逐 key 失败 | ✅ | `s3wrap.DeleteObjects` 改返回 `[]DeleteFailure{Key,Code,Message}`（`err` 仅表示传输/协议失败，两者可同时非空）；`POST …/delete` 回 `{deleted,failed,lastError}`、`delete-prefix`（同步/异步）与回收站 `PurgeObject` 一律按「请求数 − 逐 key 失败数」记账，新增 sentinel `ErrPartialDelete`。连带修 `runDeletePrefix` 的进度口径（只看 `deleted` 会在「全部删除失败」时空转到 2h 超时）。测试：s3wrap 3 例 + handler 4 例（含 100 页全失败仍终止） |
| 2 | B4 ZIP 打包 `break` → 永久 goroutine + 连接泄漏 | ✅ | `service/zip.go` 的 `break` → `continue`：无缓冲 `results` 必须继续消费，否则其余 worker 永久阻塞在发送上、已取回 body 不关闭（用户取消下载即触发）。测试 `TestWriteObjectsZipCopyErrorDoesNotLeak` 断言所有取回 body 最终关闭且全部失败 key 上报（旧实现泄漏 3 个 body）。`FEATURES.md` 早先「P-3 zip goroutine 泄漏 ✅」至此才成立 |
| 3 | B5 增量同步吞掉列举错误 | ✅ | `listAll`/`indexDst` 返回 `error`，`SyncKeys` 签名改为 `(SyncResult, error)`；`cancelOrErr` 区分「客户端取消（返回部分结果、不报错）」与「真实列举错误（上抛）」；handler 走 `writeInternalErr`（源端 AccessDenied → 403，不再回 `200 {scanned:0}`）。测试：端点级 1 例 + service 2 例 |
| 4 | B6 `indexDst` 无页数上限、循环内不查 ctx | ✅ | `listAll`/`indexDst` 共用硬上限（100 页 × 1000 key × 10 万总量）+ 每轮 ctx 检查 + 「NextToken 未前进即停」。测试 4 例（含 5s 超时护栏：旧实现会挂死） |
| 5 | B7 `Job.Emit` 锁外投递 vs `Finish` 关闭 channel | ✅ | 改为锁内非阻塞投递（`select`+`default`），落盘 I/O 移到锁外。测试用「节流落盘钩子」把 Emit 卡在「快照之后、投递之前」再 `Finish`：旧实现**确定性 panic（send on closed channel → 进程退出）**，新实现通过；另加 20×50 并发压测 |
| 6 | B9 8 MB body 上限 vs 10 000 key 批量 | ✅ | 上限提到 16 MiB（10 000×1 KB ≈ 10.3 MB 必须通过），并用 `LimitedReader{N: maxBody+1}` 把「超限」与「JSON 无效」分开：超限回 **413** 而非 400。测试 4 例 |
| 7 | B10① 第 10000 段被误判超限 | ✅ | 上限判断移到**上传前**（段号 10000 合法）；常量提为可注入的 `maxMultipartParts`（默认 10000 有断言保护），测试用小上限精确覆盖边界，避免真跑 1 万次 UploadPart |
| 8 | B10② `filename*` 用 `QueryEscape`（空格 → `+`） | ✅ | 新增 `rfc5987Escape`（attr-char 白名单 + UTF-8 逐字节 `%XX`）。测试含中文文件名与「不得出现 `+`」断言 |
| 9 | B10③ download-zip 不校验空 key | ✅ | 空 key 直接 400（否则 S3 把 `GET /bucket/?key=""` 当列举桶，ListBucket XML 被塞进 ZIP）。测试 1 例 |
| 10 | B10④ `mode=text` 忽略读错误仍回 200 | ✅ | 非 `EOF`/`ErrUnexpectedEOF` 的读错误走 `proxyErr`；`ErrUnexpectedEOF` 再用 `Content-Length` 判定短读。测试 2 例（畸形 chunked / 声明长度大于实发） |
| 11 | B10⑤ 分段顺序不校验（`InvalidPartOrder` → 500） | ✅ | handler 提交前拒绝降序/重复段号（400）；`HTTPStatus`/`UserMessageForCode` 补 `InvalidPartOrder → 400 / invalid request`。测试 2 处 |
| 12 | B10⑥ 指标标签取服务端原始 `<Code>`（基数无界，同 §S5） | ✅ | `errorClass` 改走 20 个已识别错误码的白名单，其余归 `other`。测试 `TestErrorClassUnknownCodeFoldsToOther`（断言原始码不出现在 `ErrorsByCode`） |
| 13 | B11 `statusRecorder` 未覆写 `Write` | ✅ | 覆写 `Write` 标记「已写出」，隐式 200 之后的多余 `WriteHeader` 不再透传、日志状态不被改写。测试 2 例 |
| 14 | F2 虚拟列表窗口不随 `entries` 重置 → 0 行空白表 | ✅ | 抽出共用 `src/virtualList.ts`（`virtualWindow` + 单一 `ROW_HEIGHT`），两组件在 `entries` 变化时归零 `scrollTop` 并同步写回 DOM |
| 15 | F3 客户端不分片，撞服务端上限整批失败 | ✅ | 新增 `src/limits.ts`（1000 / 10000 上限来自后端契约）+ `batchKeys`；删除与迁移改为分片串行 + 结果聚合（单片失败不中断后续），新增 i18n 键 `objects.toastDeletePartial` |
| 16 | F4 复制文件夹忽略 `truncated` | ✅ | 按 `start.truncated` 提示「仅复制一部分」（i18n 键 `dest.toastCopiedTruncated`） |
| 17 | F5 `loadAllSourceObjects` 分页内读实时前缀 | ✅ | 循环外快照 bucket/prefix + `listGen` 代次守卫（成功/失败/finally 三处） |
| 18 | F6 迁移 SSE 无空闲超时 | ✅ | 45s 空闲超时（任何数据/心跳重置），超时经既有 `onError` 上报并主动释放连接；EOF 补偿轮询语义不变 |
| 19 | F7 分享链接对全部选中并发 presign | ✅ | 复用 `batchMetadata` 的 4 路有界池 |
| 20 | F8 `AccountsPanel.load` 无 try/catch | ✅ | 捕获并渲染带「重试」按钮的错误条（i18n 键 `accounts.loadFailed`） |
| 21 | F9 健康轮询续跑 / RO 首屏失效 / 行高错位 / localStorage 无保护 | ✅ | ① 代次 + `disposed` 守卫；② `ObjectList` 的 ResizeObserver 改 `watch(scrollEl)`；③ 行高常量与 CSS 统一 42px；④ `theme.ts`/`store.ts`/`useObjectBrowser.ts` 全部 try/catch 降级 |
| 22 | F10 headers 合并顺序陷阱 | ✅ | 先合成默认 headers 再 `Object.assign` 调用方 headers（默认 Authorization/Content-Type 不再被整体覆盖） |

**门禁**：`go vet` / `golangci-lint run ./...` **0 issues** / `go test -race -count=1` 8/8 包 + **每包 100.0%**（
`awk '$NF==0'` 零块）/ `gofmt -l` 干净；前端 `pnpm lint`（0 警告）/ `pnpm typecheck` /
`pnpm test:coverage`（66 文件 / 1039 用例，四指标 100%）/ `pnpm build` 全绿；
**真对端 RustFS E2E 4/4 通过**（`S3CLIENT_E2E=1`，覆盖 12 MiB 分段组装、批量删除、桶配置、回收站 purge——
本轮改动直接触及 `DeleteObjects` / `PurgeObject` / 分段上传，故按 `AGENTS.md` 要求追加实跑）。

---

### S. 2026-09-19 分支状态审查 P1 处置（§9.2 全部闭环）

> 来源：[`docs/archive/review-2026-09-19.md`](archive/review-2026-09-19.md) §9.2 的 8 项 P1。该审查文档按约定只保留
> **未闭环**发现，本节即归档证据，逐条改动见 [`CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]`。
> 原则同 §R：**补门禁而不只是补缺陷**——每项都配了会先失败的回归测试；新增/改造的门禁在文件头
> 写明「断言范围」，避免下轮把"有门禁"误读为"已全量收敛"。

| # | 条目 | 状态 | 实现与验证 |
|---|------|------|------------|
| 1 | §7.2 修正 `components.schemas.Account` | ✅ | `internal/openapi/openapi.go` 删幻影字段 `provider`/`forcePathStyle`/`insecureSkipVerify`、补真实 `useSSL`；与 `model.AccountView` 逐字段一致 |
| 2 | §7.2 新增**响应契约门禁** | ✅ | 新增 `openapi_response_contract_test.go`：对 `components.schemas` 每个共享 schema 用 `go/parser` 抽取对应 Go DTO 的 `json` tag，做**双向**比对（多写=幻影字段，漏写=客户端丢字段）；含 `Account` 逐字段断言与 `secretKey` 不得出现断言。上线即红灯，修正后转绿 |
| 3 | §4.3 文档字段门禁改**双向**、去子串匹配 | ✅ | `api_doc_test.go` 的 `TestAPIDocDocumentsRequestBodyFields` 改为机械抽取 `docs/api.md` 请求体 JSON 的**顶层键**（自写容错扫描器，支持注释剥离 / 嵌套对象 / `200 {…}` 响应体区分），与注册表双向比对；彻底去掉 `strings.Contains`（原实现下 `key` 会被 `keys`/`secretKey` 满足）。别名段支持「与 `POST /api/x` 相同」与省略方法名的「与 `copy-prefix` 相同」 |
| 4 | §4.3 请求体字段门禁改**全量遍历** | ✅ | 新增 `openapi_request_fields_test.go`：routes.go → handler 方法调用闭包 → `readJSON` 目标变量 → 结构体字段集，与注册表 requestBody schema 全量双向比对。端点专属 DTO 双向相等；共享模型走 `serverManagedFields` 白名单。**上线即抓到 3 处新漂移**：`storage-class` 漏 `versionId`、`preview-buckets` 漏 `publicEndpoint`/`useSSL`——已修注册表与 `docs/api.md` |
| 5 | R2 tag ↔ 清单一致性 | ✅ | `release-desktop.yml` 新增 `Verify tag matches manifest versions`：tag（去 `v`）必须等于 `tauri.conf.json` 与 `Cargo.toml` 的 `version`，否则 `exit 1`。本地实测 rc1 通过、rc2 正确失败 |
| 6 | R3 `SHA256SUMS` 竞态 | ✅ | 三平台各自上传**平台内唯一**的 `SHA256SUMS-<bundle>.txt`；新增 `aggregate-checksums` job（`needs: publish`）拉取三份合并去重为唯一 `SHA256SUMS.txt`（含行数自检），并清理中间资产 |
| 7 | R4 Trivy DB 缓存 / 重试 | ✅ | 两套 CI 均拆出 `--download-db-only` + 4 次指数退避重试，扫描阶段加 `--skip-db-update` 复用已就绪 DB；GitHub 侧加 `actions/cache`（`.trivy-cache`），GitLab 侧加 `cache: paths`。把「网络抖动」与「真的扫出漏洞」区分开 |
| 8 | S1 运行镜像 alpine EOL | ✅ | `apps/server/Dockerfile` 运行镜像 `alpine:3.20`（2026-04-01 EOL）→ `alpine:3.24`（支持至 2028-06-01） |
| 9 | S2 `.env.example` 占位 token | ✅ | 根 `.env.example` 的 `S3C_TOKEN` 由 `change-me-use-openssl-rand-hex-32`（33 字符，是**有效口令**）改为置空，对齐 `apps/server/.env.example` |
| 10 | 配置类门禁（防复发） | ✅ | 新增 `apps/server/repo_infra_gate_test.go`：`.env.example` token 不得 ≥ `MinTokenLength`、运行镜像不得是 EOL alpine、发布 workflow 必须做 tag↔清单比对与平台内唯一 checksum、两套 CI 的 Trivy 必须缓存+重试。四项均为「YAML/配置正确但断言缺席」的盲区 |
| 11 | §7.3 文档失真 D2 / D3 / D6 / D8 / D9（同批修正） | ✅ | D2：`openapi_handler.go` 鉴权注释改为与实测一致（401/404）；D3：`release-version.sh` 正则放宽到通用 semver（接受纯 `v1.0.0`/`v1.1.0`）、同步文件补到 12 处、`Cargo.lock` 只改本包；D6：`api.md` 的 presign `expiresIn` 改为「默认 1h、上限 24h」；D8：现行能力描述「3 路并发」→ **2 路**；D9：`DEVELOPMENT.md` 存放约定改为「除根目录约定文件与 `.github/` 外统一放 `docs/`」 |
| 12 | SSOT：`docs/KNOWN_ISSUES.md`（时名 `todolist.md`）补登记开放项 | ✅ | 此前写着「无待办」而审查仍有大量开放发现，违反「唯一待办来源」约定；现按 #26–#46 补登记 P2 全部条目（契约残留 / 安全 / 发布链 / 可观测性 / 文档失真），编号稳定不重排 |

**门禁实跑**：`gofmt -l` 干净 / `go vet ./...` 0 告警 / `golangci-lint run ./...` **0 issues** /
`go test -race -count=1 ./...` 8/8 包通过且**每包 100.0% 语句覆盖**（`awk '$NF==0'` 零块）；
新增门禁均已做「先红后绿」验证（Account schema 与 3 处注册表漂移均先失败再修绿）。

---

### T. 2026-09-20 审查 §9.3 P2 处置（契约 #26–#28 + 安全 / 供应链 #29–#34）

> 来源：[`docs/archive/review-2026-09-19.md`](archive/review-2026-09-19.md) §9.3 P2。该审查文档按约定只保留**未闭环**发现，
> 本节即归档证据，逐条改动见 [`CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]`。原则同 §R / §S：
> **补门禁而不只是补缺陷**——每项都配了会先失败的回归测试，新增门禁在文件头写明「断言范围」。
> #25（桌面端签名）经确认是**外部凭证阻塞**（roadmap E6），代码侧无剩余工作，转入 ⛔ 外部阻塞。

| # | 条目 | 状态 | 实现与验证 |
|---|------|------|------------|
| 1 | #26 query 参数漏声明 | ✅ | 补 `head.versionId` / `proxy.maxBytes` / `trash.prefix` / `objects.startAfter`（+ `docs/api.md` 补 `startAfter`）。新增 `openapi_query_params_test.go`：routes.go → handler 调用闭包 → 机械抽取 `q.Get(...)`，与注册表 `in:query` **双向**比对（含自检计数）。**上线即抓到第 5 处漂移**：`GET /versions` 的幻影参数 `key`（handler 从不读取）→ 移除 |
| 2 | #27 类型 / required / 枚举语义门禁 | ✅ | 新增 `openapi_semantics_test.go`。**枚举**：每个 `EnumStr` 站点须在 `enumContracts` 登记，真实值从 `switch` case / `map[string]bool` / 跨包 `service` 常量抽取后双向比对 → 抓到 `PUT object-acl` 漏 `authenticated-read`/`aws-exec-read`。**required**：注册表 required 必须是 handler 真实解码且非「空值→默认」的字段 → 修正 `presign.method`、`copy-object.newBucket`、`copy-prefix.targetBucket` 三处错误 required。残留（`bucketOr` 默认桶模式）写入文件头 |
| 3 | #28 响应门禁残留范围 | ✅ | `openapi_response_contract_test.go` 新增**端点级响应门禁**：对声明具体 `properties` 的 2xx 响应，抽取 handler `writeJSON` 表达式的键（字面量 map / 命名 struct tag / 变量 / 同包 helper 递归）双向比对；4 个注册表文件约 30 个 `Obj()` 响应升级为 `BuildObj` 纳入门禁；修正 3 个异步端点 `200`→`202`。残留（其他注册表自由体、运行时动态键 map）写入文件头 |
| 4 | #29 / #31 明文落盘安全默认 | ✅ | `Config.Validate` 对 `json`/`sqlite` + 空 `S3C_STORE_KEY` 返回新 sentinel `ErrPlaintextStoreNotAllowed` **硬失败**，除非 `S3C_ALLOW_PLAINTEXT_STORE=1`（opt-in 时保留 WARN）；报错给出两条出路且不泄露密钥。base compose 改 `${S3C_STORE_KEY:?}` 强制非空；`.env.example`（根 + server）补说明；`e2e.yml` 补 dummy key（`docker compose up` 会插值整个文件）。测试 `TestValidatePlaintextStoreRejected` / `TestMainServerRejectsPlaintextStoreWithoutOptIn` / `TestMainServerAllowsPlaintextStoreWithOptIn` / `TestFromEnvAllowPlaintextStore` |
| 5 | #30 metrics 免鉴权文档声明 | ✅ | 行为不变（有意内网 scrape），在 `README.md` / `docs/api.md` / `docs/DEPLOYMENT.md` §6.3 / `docs/threat-model.md` §2 显式声明「即使配了 `S3C_TOKEN` 也不受保护、匿名可读、勿直接暴露公网」；`TestMetricsEndpointUnauthenticatedEvenWithToken` 钉住行为（metrics 无 token 200 / accounts 无 token 401） |
| 6 | #32 Trivy 升级 + digest pin | ✅ | `aquasec/trivy:0.58.1` → `0.74.0@sha256:62b1e65e…1969`（两套 CI；digest 经 `docker pull` + ghcr.io + public.ecr.aws 三源核对）；`--vuln-type` → `--pkg-types os,library`（0.74.0 `--help` + `pkg/flag/package_flags.go` 核实）。门禁 `TestTrivyImageIsVersionAndDigestPinned`（两套 CI 版本一致 + 必须带 digest + 禁用 `--vuln-type`） |
| 7 | #33 工具链 pin | ✅ | **关键发现**：原 `dtolnay/rust-toolchain` SHA `6bed0761…` 是 `stable` **浮动分支 tip**（commit 即 `toolchain: stable`），并非版本 pin——改为版本分支 SHA `ce678459… # 1.98.1`；新增 `rust-toolchain.toml`（`channel = "1.98.1"`）；GitLab `rust:1.98.1-bookworm`。Node `24` → 精确 `24.21.0`（workflow / alpine / bookworm / NodeSource）。desktop `packageManager: pnpm@9.15.0`（实测不破坏 lockfile）。门禁 `TestRustToolchainIsVersionPinned` / `TestNodePinnedToPatchVersion` / `TestPackageManagerIsPinnedToCIPnpm` |
| 8 | #34 `.dockerignore` 补齐 | ✅ | 补 `apps/web/coverage`、`.pnpm-store`、`test-results`、`playwright-report`、`.run/`、`.cargo/`、`.trivy-cache/`、`.gitlab-ci-local/`、`coverage.out`/`.txt`、`*.tsbuildinfo`、`*.log` 等；`apps/server/*.log` → `**/*.log`。门禁 `TestDockerignoreExcludesBuildArtifacts`（11 条必需模式，注释行不计入） |

**门禁实跑**：`gofmt -l` 干净 / `go vet ./...` 0 告警 / `golangci-lint run ./...` **0 issues** /
`go test -race -count=1 -coverprofile` 8/8 包通过且**每包 100.0% 语句覆盖**（`awk '$NF==0'` 零块）/
`go build ./...` OK；前端 `pnpm lint` 0 告警 / `pnpm test` 66 文件 1039 用例全绿 / `pnpm build` OK；
`docker compose config` 缺 `S3C_STORE_KEY` → exit 1、补齐 → exit 0。供应链 pin 均经上游实测核验
（Trivy 版本 + digest、Node 24.21.0、Rust 1.98.1），新门禁均做「先红后绿」与变异验证。

---

### U. 2026-09-22 审查 §9.3 P2 收尾（D1 / D4 / D5 / D10 + P3 + R7 / R9 / R10 + 门禁质量）

> 来源：[`docs/archive/review-2026-09-19.md`](archive/review-2026-09-19.md) §7.3 与 §9.3 的余项；逐条改动见
> [`CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]`。原则同 §R / §S / §T：**补门禁而不只是补缺陷**，
> 每项都配了会先失败的回归测试，关键修复做「还原旧实现」变异验证。

| # | 条目 | 状态 | 实现与验证 |
|---|------|------|------------|
| 1 | D1 Prometheus 直方图语义违规 | ✅ | 根因：`s3wrap` 记的是**每桶增量**，`handler/metrics.go` 直接当**累积**值按 `_bucket{le=…}` 输出 → `le` 非单调、`+Inf`≠`_count`（实测 `le="0.01"`=3 / `le="+Inf"`=0 / `_count`=3），`histogram_quantile()` 全错。现 `MetricsSnapshot` 快照时累积为真累积值；并把 `calls` 由 `atomic.Int64` 移入同一把锁（消除「`+Inf`=0 而 `_count`=3」的读写不一致窗口）。测试：`TestS3MetricsHistogramIsCumulativeAcrossBuckets`（3 个耗时逐桶断言累积值）、`TestS3MetricsHistogramConcurrentContract`（8×50 并发）、`TestMetricsHistogramSatisfiesPrometheusContract`（端点级解析 `le` 单调性与 `+Inf == _count`）。**变异验证**：把 `cumulative += c` 还原为 `cumulative = c` → 三条全部红灯，且复现报告中 `+Inf`=0 / `_count`=3 的原始症状 |
| 2 | P3 `/api/openapi.json` 无缓存 | ✅ | `Registry` 增加 `spec` 缓存：`Operation` / `Param` / `Respond` / `SetInfo` / `AddServer` 全部置空（语义与「每次重算」等价）；`MarshalJSON` 双检加锁只算一次并返回**副本**（调用方改写不污染缓存）。测试：`TestRegistry_MarshalJSONCached`（含副本隔离）、`TestRegistry_MarshalJSONCacheInvalidated`（四类注册动作逐一验证）、`TestRegistry_MarshalJSONConcurrent`。既有 `TestOpenAPI_ContractMarshalDeterministic`（8×8 并发字节一致）继续通过 |
| 3 | D4 `failKeys ≤ 200` 承诺不成立 | ✅ | 根因：承诺写在 `FEATURES.md` / `api.md`，但服务端只有异步 `delete-prefix` 在 handler 内裁剪；`copy-objects`（同步/异步）、`migrate`（同步/异步）、`migrate/sync` 全部原样回传 → 10k 全失败约 10 MB 并落盘 `jobs.json`。现上限提为 handler 层共享常量 `maxFailKeys = 200` + `capFailKeys`，**所有**回传 `failedKeys` 的端点统一裁剪；异步路径经新增 `jobResultFromBatch` 在 `Job.Finish`（落盘）**之前**裁剪；`failed` 计数不受裁剪影响。测试：`TestOlCopyManyFailKeysAll`（201 失败 → 列表 200、计数 201）、`TestOlMigrateSyncFailKeysCapped`、`TestOlMigrateAsyncFailKeysCappedBeforePersist`（**直接读 `jobs.json`** 断言裁剪发生在持久化前）。同时修正 `service/batch.go` 与 `copy.go` 中两处互相矛盾的注释 |
| 4 | D5 `git tag v1.0.0` 不存在 | ✅ | `scripts/release-version.sh v1.0.0` 同步 15 处版本串（Makefile / Dockerfile / compose×2 / web+desktop `package.json` / `tauri.conf.json` / `Cargo.toml` / `Cargo.lock`（只改本包）/ `main.go` / README / docs），并打 `v1.0.0` tag。历史性提及（CHANGELOG 已发布区、roadmap 里程碑行、本审查报告）按「不追溯篡改」纪律保留 |
| 5 | D10 CHANGELOG 段内过期计数 | ✅ | `[Unreleased]` 内两处「当时实测值」（覆盖率 3883/2769/1059/3339、前端 63 文件 / 983 测试）已失真。按本文件「不追溯篡改已发布区」的纪律**保留原值并加注**当时时点与新值（4074/2844/1095/3503、66 文件 / 1039——后者是 §W 追加用例前的值，§W 后为 1042），不改写历史数字 |
| 6 | R7 本地 `make` ≠ CI | ✅ | 新增 `lint`（golangci-lint）/ `govulncheck` / `check`（聚合静态检查 + 双侧覆盖率）目标；全部 `pnpm install` → `--frozen-lockfile`；`test-all: test-cover web-test-cover`（此前无覆盖率门禁）；`.PHONY` 补全全部已定义目标。门禁 `TestMakefileMirrorsCIGates`（含「所有已定义目标必须在 .PHONY」的机械检查；**变异验证**：还原裸 `pnpm install` → 红灯） |
| 7 | R9 GitHub 不传 `VERSION` build-arg | ✅ | 构建与推送两处都显式注入 `VERSION=ci`（此前回落到 Dockerfile 过期字面量，而 GitLab 传 `ci`、Makefile 传真实版本——一个 Dockerfile 三种行为）。门禁 `TestGitHubWorkflowInjectsVersionBuildArg`（要求 ≥2 处，保证扫描与推送版本一致；**变异验证**：改掉一处 → 红灯） |
| 8 | R10 无 workflow 推送镜像 | ✅ | 新增 `publish` job：`needs: docker`（Trivy 通过才推）、`if: github.event_name != 'pull_request'`、`permissions: packages: write`、`docker/login-action` 登录 GHCR、按 commit SHA + 分支双标签推送。因 buildx `push` 与 `load` 互斥且扫描需 `load`，拆为独立 job（同时避免推出未扫描镜像）。门禁 `TestGitHubWorkflowPushesImage` |
| 9 | 测试质量：21 个用例名硬编码过期行号 | ✅ | 如 `api.test.ts` 写 `line 125 else`（当时 `listServers` 在 `storage.ts:301`）、`bucketPolicy.test.ts` 写 `line 111`（当时 `:108`）——测试在描述一个不存在的版本。已从全部用例名移除行号（改为描述行为，行号留在注释），新增门禁：`deadcode_gate.test.ts` 的「测试名不得硬编码源码行号」（含扫描下限自检防空跑） |
| 10 | 测试质量：i18n 门禁两处盲区 | ✅ | ① `usedKeyTexts` 对整份源码做正则 → **注释里的键名也算「已使用」**；现先按字符扫描剥离注释（正确跳过字符串字面量，避免把 `'https://x'` 的 `//` 当注释）再匹配。② 双语只比**键数量** → `zh-CN` 缺 `a` 而 `en-US` 多 `b` 时数量相等仍绿；现按语言解析键集合做**集合级双向比对**，并新增「每个键两种语言取值都非空」。**变异验证**：改名一个 en-US 键（数量不变）→ 集合门禁红灯；删真实引用只留注释 → 死键门禁红灯 |
| 11 | 文档失真小项 | ✅ | `openapi_contract_test.go` 的 `apiRouteCount` 注释算错（写「69 = 68 + 1」，常量与实际均为 70 → 改「70 = 69 + 1」）；`docs/KNOWN_ISSUES.md`（时名 `todolist.md`）补记 #40 / #42 / #43 / #44 为**从未启用的保留空号**（避免被误读为漏登记），#15 补入已完成编号 |

**门禁实跑**：后端 `gofmt -l` 干净 / `go vet ./...` 0 告警 / `go test -race -count=1 -coverprofile`
8/8 包通过且**每包 100.0% 语句覆盖**；前端 `pnpm lint` 0 告警 / `pnpm typecheck` 通过 /
`pnpm test:coverage` **66 文件 1042 用例全绿，四项覆盖率 100%**（statements 4074 / branches 2844 /
functions 1095 / lines 3503）。

> **仍未纳入机械门禁的残留**（须人工跟踪，已写入各门禁文件头）：端点内联 / 运行时动态键的响应体、
> query 参数的 `Has()` / `Values()` 读取口径、md 表格中的叙述性数字。
> （**真实 Go 后端 + 真实前端产物**的浏览器联调冒烟已于 §V 闭环，不再属于本残留清单。）
>
> **2026-09-22 收尾（§W）**：上述三项残留**全部闭环**——端点级响应门禁升级为全量覆盖（15→61 个
> 成功响应）、query 门禁扩到四种读取口径、新增 md 叙述性数字门禁与 `_test.go` 导出符号死代码门禁。
> 现顶层自由体响应仅剩 `/api/openapi.json` 一个（响应体即 OpenAPI 规范本身，非 `writeJSON` 写出）；
> `metadata` / `fields` 等嵌套自由体只比对到顶层键。

---

### V. 2026-09-22 真实后端 + 真实 RustFS 浏览器联调（KNOWN_ISSUES #37）

> 来源：[`docs/archive/review-2026-09-19.md`](archive/review-2026-09-19.md) §9.3 的**唯一保留项**。此前
> `e2e-playwright.yml` / `playwright-e2e` 的 `/api/**` 全被 `page.route` mock，没有任何门禁
> 验证「真实 Go 后端 + 真实 RustFS + 真实构建产物」拼在一起时的行为——尤其**浏览器直传**
> （预签名 PUT 是浏览器 → S3 的跨源请求），mock 时代在结构上不可能被覆盖。本轮补齐后
> 审查 §9.3 清零。

| # | 条目 | 状态 | 实现与验证 |
|---|------|------|------------|
| 1 | 真实联调用例（不 mock `/api`） | ✅ | 新增 `apps/web/e2e-real/real-backend.spec.ts` + 专用 `playwright.real.config.ts`（不自拉 webServer，`PLAYWRIGHT_BASE_URL` 指向真实后端）。3 条用例：① 账号 CRUD 真实落库（后端 `GET /api/accounts/{id}` 可见、`secretKey` 不回传、UI「测试连接」走真实 S3 成功）；② 建桶 → 列桶（真实 `CreateBucket`/`ListBuckets`）；③ **浏览器真实直传**：UI 选文件 → 前端取预签名 PUT → 浏览器 XHR PUT 到 RustFS → 列对象 → 预签名 GET 回读逐字节校验。用例先 `resetBackend` 清空账号表，保证每个用例从真实空状态出发（失败重试也不会串状态） |
| 2 | 本地一键运行（docker 自动起 RustFS） | ✅ | 新增 `scripts/e2e-real.sh` + `make e2e-real`：自动 docker 起一份真实 RustFS（**显式配 `RUSTFS_CORS_ALLOWED_ORIGINS`**，否则浏览器直传被 CORS 拦下）→ `pnpm build` 真实产物 → 起真实 Go 后端（`S3C_STATIC_DIR` 托管产物）→ 跑 `pnpm e2e:real` → `trap` 自动清理；`--keep` / `--skip-build` / `--no-rustfs`（复用外部对端）便于排查与 CI 复用；幂等 `playwright install chromium` 修掉新克隆直接失败的 UX 缺口 |
| 3 | 两套 CI 接入（GitHub + GitLab） | ✅ | GitHub 新增 `.github/workflows/e2e-real.yml`（PR 按路径 + 手动 + 每周五定时）；GitLab 新增 `e2e-real` job（RustFS 用 GitLab **service**，与 `rustfs-e2e` 同款；service 变量里配好 `RUSTFS_CORS_ALLOWED_ORIGINS`）。触发规则与另两套 E2E 对齐，不阻塞普通 push |
| 4 | 编排收敛为单一来源 | ✅ | 初版三处各写一份编排（改一处漏两处即漂移）。现两套 CI 都 `bash scripts/e2e-real.sh`（GitHub 自起容器；GitLab `--no-rustfs` + `RUSTFS_ENDPOINT=http://rustfs:9000` 复用 service），各自只留「装工具链 / 装浏览器系统依赖」 |
| 5 | 机械门禁防漂移 | ✅ | `TestRealE2EUsesSharedScript`（两侧 CI 必须**实际调用** `bash scripts/e2e-real.sh`——仅出现在 `paths:`/`changes:` 里不算）、`TestRustFSImageIsConsistentlyPinned`（脚本默认值 / compose / GitLab service 三处镜像版本一致，禁 `latest`）、`TestRealE2EArtifactsExist`（**联调 spec 出现 `page.route(` 即红灯**，防退化成第二个 mock 版）、`TestE2ESourcesAreTypechecked`、`TestLocalRealE2ETargetExists` |
| 6 | E2E 源码静态检查 | ✅ | `e2e/` 与 `e2e-real/` 此前不在主 `tsconfig.json` 的 `include` 内，**零类型检查与 lint**。现新增 `apps/web/tsconfig.e2e.json` + `pnpm typecheck:e2e`，`pnpm lint` 扩到 `eslint src e2e e2e-real`（顺带修掉一个未使用变量）；两套 CI 的 `web` job 与本地 `make check` 都纳入 |

**变异验证**：① 去掉 RustFS 容器/CORS 配置后重跑本地真实联调——第 3 条（浏览器直传）**红灯**
（跨源 PUT 被拦，超时），前两条不经浏览器的用例仍绿，证明该用例确实依赖真实对端 + 真 CORS；
② 门禁侧逐条变异：改 GitLab service 镜像版本 → `TestRustFSImageIsConsistentlyPinned` 红灯；
把 GitLab 的 `bash scripts/e2e-real.sh` 换成 `pnpm e2e:real` → `TestRealE2EUsesSharedScript` 红灯
（该门禁最初用子串 `scripts/e2e-real.sh`，被 `changes:` 路径命中而假绿，已改为要求实际调用）；
删 CI 里的 `typecheck:e2e` → `TestE2ESourcesAreTypechecked` 红灯；往联调 spec 塞类型错误 → 类型检查红灯。

**门禁实跑**：`make e2e-real` 3 passed（默认 docker 模式与 `--no-rustfs` 外部对端模式各跑一遍）；后端新增 5 条门禁纳入 `go test` 100% 覆盖率。

---

### W. 2026-09-22 审查 §7.4 门禁盲区收尾（响应门禁全量 + query 读取口径 + 叙述性数字 + 导出测试符号）

> 来源：[`docs/archive/review-2026-09-19.md`](archive/review-2026-09-19.md) §4.3 / §7.4 / §9.1 列出的**最后四类残留**
> ——即门禁矩阵里标「部分」与「否」的行。原则同 §R / §S / §T / §U：**补门禁而不只是补缺陷**；
> 本轮新增/扩展的门禁全部做了变异验证，并在升级过程中**抓到 4 处真实漂移**（见条目 5–8）。

| # | 条目 | 状态 | 实现与验证 |
|---|------|------|------------|
| 1 | 端点级响应门禁：**顶层**自由体 `openapi.Obj()` 全量收敛 | ✅ | 把 accounts / buckets / bucket-settings / objects / object-meta / multipart / versions / trash / migrate / system 十个注册表的**顶层**自由体响应全部升级为 `openapi.BuildObj` 具体 `properties`（嵌套形状抽为共享构造器：`corsRuleSchema` / `tagRowSchema` / `lifecycleRuleSchema` / `versionEntrySchema` / `deleteMarkerSchema` / `jobProgressSchema` / `jobResultSchema` / `jobRecordSchema`）。端点级门禁覆盖面 **15 → 61** 个成功响应，**顶层**自由体从 ~45 个降到 **1 个**（`/api/openapi.json`：响应体即规范本身，由 `Registry.HTTPHandler()` 直接写，非 `writeJSON`，机械抽取无意义）。自检从 `checked ≥ 15` 收紧为 `checked ≥ 60` 且 `untyped ≤ 1`——**回退成顶层自由体会直接红灯**。注意：`metadata` / `fields` 等**嵌套**自由体仍只比对到顶层键 |
| 2 | query 参数门禁：读取口径扩展 | ✅ | 抽取器从只认 `Get()` 扩到四种字面量口径：`Get` / `Has` / `Values` / `Query()["x"]`，并覆盖 `q := r.URL.Query()` 绑定后的同名形式。新增**非字面量键检测**（`q.Get(name)` / `q[k]`）——命中即红灯要求改字面量，杜绝「动态键读取静默逃逸」。新增口径测试 `TestQueryReadExtractorCoversAllForms`（8 种形态逐一断言，含「无读取不得抽出参数」的反向断言） |
| 3 | md 叙述性数字门禁（矩阵唯一标「否」的行） | ✅ | 新增 `apps/server/doc_number_gate_test.go`：以 `routes.go` 的 `mux.HandleFunc` 注册数为**唯一真值**，校验 README / `api.md` / `ROADMAP.md` / `FEATURES.md` 里「N 个 `/api/*` 端点」的 N。要求每条声明**至少命中一次**（文案漂移导致正则失配即红灯，防门禁静默失效）。变异验证：把 `FEATURES.md` 改回 69 → 红灯 |
| 4 | `_test.go` 导出符号死代码门禁 | ✅ | `golangci-lint unused` 对**导出**符号因「可能被包外引用」而豁免，但 `_test.go` 不参与库构建、永远无包外引用——实测给 `_test.go` 加无人调用的导出函数/类型，`golangci-lint run` 报 **0 issues**。新增 `TestNoUnusedExportedTestSymbols`：扫描全部 `_test.go` 的导出包级符号，无引用即红灯；`export_test.go`（约定的测试接缝）整体豁免。检测逻辑抽为纯函数 `findUnusedExportedTestSymbols`，由 `TestFindUnusedExportedTestSymbols` 用**合成源码**做口径测试（死符号必报、被引用/非导出/Test 入口/接缝文件不得误伤）——不依赖「仓库里正好有个死符号」来证明门禁有效 |
| 5 | **真实漂移（升级中抓到）**：`s3wrap.CorsRule` 缺 `json` tag | ✅ | 该结构体经 handler **直接序列化**进 `GET /bucket/cors` 响应，却无 tag → Go 输出 PascalCase（`AllowedMethods` / `MaxAgeSeconds`），而前端 `types.ts` 与 `docs/api.md` 都是 camelCase。Go 的 `json.Unmarshal` 大小写不敏感，**服务端既有测试一直全绿**；但 JavaScript 严格区分大小写，浏览器读 `x.allowedMethods` 恒为 `undefined` → CORS 规则的方法/来源被静默归一化成空数组。已补全 6 个 camelCase tag；新增 `TestCorsRuleJSONFieldNames`（双向：必须有 camelCase、不得再有 PascalCase）与 `TestCorsRuleJSONRoundTrip`（前端发来的 camelCase 能解析）。**变异验证**：去掉 tag → 两条红灯并列出实际 PascalCase 键 |
| 6 | **真实漂移**：`POST /api/migrate/async` 误声明 200 | ✅ | handler 写 `StatusAccepted`（202），`docs/api.md` 也写 `202`，只有注册表写 200。已改 202 并由端点级门禁钉住 |
| 7 | **真实漂移**：`docs/api.md` 称 openapi.json「不进鉴权层」 | ✅ | 与实现相反：`withAuth` 只豁免 `health` / `metrics`，openapi.json 配了 token 时无凭证返回 401，且需 `S3C_EXPOSE_OPENAPI=1` 否则 404（既有 `TestOpenAPI_GateAndAuth` 早已钉住该行为，仅文档失真）。已改为「经过鉴权层」并写明两个前置条件 |
| 8 | **真实漂移**：`docs/FEATURES.md` 称 69 个端点 | ✅ | 实际为 70（`routes.go` 70 条 `mux.HandleFunc`，README / `api.md` / `ROADMAP.md` 与 `apiRouteCount` 常量均为 70）。已改为 70，并纳入条目 3 的数字门禁 |
| 9 | 文档请求体缺失：`PUT /api/accounts/{id}` | ✅ | 该端点请求体在 `docs/api.md` 只有散文描述（「字段同创建」），导致 `TestAPIDocDocumentsRequestBodyFields` 无法机械比对字段集而红灯。已补显式 JSON 示例（并注明 `secretKey` 可省略、必填为 `name`/`endpoint`/`accessKey`），与注册表的 required 声明一致 |

**门禁实跑**：后端 `go vet ./...` 0 告警 / `golangci-lint run ./...` **0 issues** / `go build ./...` OK /
`go test ./...` 全绿（含新增门禁）；四条新门禁与两处缺陷修复均经**变异验证**（改回旧写法必红灯）。

> **本节后的残留**（已写入各门禁文件头）：**顶层**自由体响应仅 `/api/openapi.json` 一个；
> `metadata` / `fields` 等**嵌套**自由体与嵌套对象/数组的**深层字段**只比对到顶层键（元素形状由
> 注册表共享构造器统一维护）；md 中依赖运行环境或时刻的叙述性数字（用例数 / 覆盖率 / 行数）
> 按「历史记录」保留，不做静态门禁；md 端点数量叙述数字由 `doc_number_gate_test.go` 覆盖。

---

### X. 2026-09-22 审查 §4.3「维持」项复核收尾（门禁自身口径 + path 参数门禁）

> 来源：[`docs/archive/review-2026-09-19.md`](archive/review-2026-09-19.md) §4.3 表格里最后五行标「维持」的项。
> 复核后发现**每一项都藏着真实缺口**（不是「已足够好」，而是「没人验过它能不能被绕过」）——
> 全部经**实测确认缺口存在**，再改为 AST 判定或新增门禁，并逐条做变异验证（改回旧写法必红灯）。

| # | 门禁 | 实测缺口 | 处置与验证 |
|---|------|----------|------------|
| 1 | `openapi_inputsource_test.go` | 用裸字符串匹配 `readJSON(` 判断「handler 是否解码请求体」→ **注释与字符串字面量都能骗过它**。实测 `// 这里提到 readJSON(` 与 `s := "readJSON("` 均被判定为「解码了」 | 改为 **AST 判定**（`parseBodyCalls` 解析语法树，只认真实调用表达式）。新增 `TestDecodesBodyIgnoresCommentsAndStrings` 用合成源码钉住口径（注释/字面量不算、真调用与委托算）。**变异验证**：删掉 `createAccount` 的真实 `readJSON` 调用、只在注释留 `readJSON(` → 旧实现假绿、新实现红灯 |
| 2 | `api_doc_test.go` `TestAPIDocMatchesRoutes` | 解析正则只认行首，缩进行在两个方向上都被漏。实测「文档里有**陈旧**路由、且写成缩进」时双向 diff **静默通过**（fail-open）——陈旧条目会永久留在文档里 | 新增 `TestAPIDocRoutesAreFlushLeft`：路由声明行必须顶格，缩进即红灯（含扫描下限自检）。**变异验证**：插入缩进的 `DELETE /api/nonexistent-stale` → 新门禁红灯，旧双向 diff 无反应 |
| 3 | `deadcode_gate_test.go` | 旧实现是整行正则 `^_ = ident` / `^var _ = expr`。实测 `x := 1; _ = x`、`if true { _ = x }`、`_ = x.Field`、`_ = s[0]` **全部逃逸**（fail-open） | 改为 **AST 判定**（`findSilencingDeadCode`）：按语句而非按行，判据是「左侧是否全为 `_`」。`_, ok := m[k]`（comma-ok 惯用法）与 `_, _ = w.Write(b)`（显式丢弃返回值）放行；`var _ Iface = (*T)(nil)`（编译期接口断言）放行。新增 `TestFindSilencingDeadCodeCoverage` 断言「AST 命中数严格大于旧正则」（实测 9 vs 5），防止退回旧口径。**变异验证**：四种逃逸形状现全部被拦 |
| 4 | `golangci-lint` 配置 | **未启用 `nolintlint`**：在函数上方加一行无理由的 `//nolint:all`，`golangci-lint run` 仍报 **0 issues**——`//nolint` 是 golangci 自身的指令，默认不受任何 linter 审查；而 `AGENTS.md` / `DEVELOPMENT.md` 明令禁止用它消音 | `.golangci.yml` 启用 `nolintlint`（`require-explanation: true` + `require-specific: true` + `allow-unused: false`）。既有唯一一处 `//nolint:staticcheck // SA1019: ...` 已合规。**变异验证**：植入 `//nolint:all` → 红灯（"should mention specific linter"） |
| 5 | path 参数（新发现的缺口） | `TestOpenAPI_ContractPathParamsDeclared` 只校验注册表**内部**自洽（模板 `{x}` ⇔ 声明 `in:path`），**没有任何一条**比对 handler 是否真的 `r.PathValue("x")` 读取它——注册表与 handler 同时改名才一致，只改一边无人拦 | 新增 `openapi_path_params_test.go`：注册表 `in:path` 参数名集 ⇔ handler 沿调用闭包 `PathValue` 读取集，**双向**比对（70 个端点 / 60 个带 path 参数），含非字面量键检测与口径测试。**变异验证**：把 `getAccount` 的 `PathValue("id")` 改成 `PathValue("accountId")` → 同时报「漏声明」与「幻影参数」 |

**门禁实跑**：后端 `go vet ./...` 0 告警 / `golangci-lint run ./...` **0 issues**（含新启用的 `nolintlint`）/
`go build ./...` OK / `go test ./...` 全绿（8/8 包，**每包 100.0% 覆盖且零未覆盖块**）。

> **本节后的残留**：§4.3 表格已无「维持」项（每行都是 ✅ 且带回归门禁）。仅剩的**结构性**无法
> 机械拦截项同 §W——`/api/openapi.json` 顶层自由体响应、`metadata`/`fields` 嵌套自由体与嵌套结构深层字段、依赖运行环境/时刻的
> 叙述性数字；均已在对应门禁文件头写明断言范围。
>
> **2026-09-22 追加复核（门禁自身再检查）**：发现并关闭三处新缺口——① `openapi_query_params_test.go`
> 的动态键检测只认 `r.URL.Query().Get(name)`，绑定变量后的 `q := r.URL.Query(); q.Get(name)` / `q[k]`
> 会同时逃过抽取与动态检测（fail-open，已补 `hasDynamicQueryRead` 与合成用例）；
> ② `openapi_path_params_test.go` 的动态键检测还是裸正则（注释/字符串会误报），已改 AST
> （`hasDynamicPathValueRead`）；③ `openapi_request_fields_test.go` 的 `readJSON` 定位仍是逐行正则
> （注释/字符串里的调用会被当成解码点），已改 AST（`findReadJSONTarget`）。同时把
> `delete-prefix/async`、`copy-prefix/async` 的请求体从 `openapi.Obj()` 自由体升级为具体
> `properties`，使请求体字段门禁覆盖这两个端点。

---

### Y. 2026-09-22 子审查复核：异步任务契约统一 + 门禁口径补充

> 来源：子审查 724461ee 的复核发现。逐条修复证据见 [`CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]`。

| # | 条目 | 状态 | 实现与验证 |
|---|------|------|------------|
| 1 | 异步任务失败 key 契约分裂（`failKeys` vs `failedKeys`） | ✅ | 公共契约统一为 `failedKeys`（`JobResult` json tag + OpenAPI `jobResultSchema` + `docs/api.md` list 示例）；`job_persist.go` 的 `UnmarshalJSON` 兼容旧落盘 `failKeys`。新增 `TestOlMigrateJobsListUsesFailedKeys`（清单接口 result 必须含 `failedKeys`、不得泄露 `failKeys`）与 `TestJobResultUnmarshalAcceptsLegacyFailKeys`（新旧格式 + 非法 JSON 错误路径） |
| 2 | query 字面量空白盲区 | ✅ | `q.Get( "x" )` / `q[ "x" ]` 既可被抽取、也不会被误判动态；动态下标正则补 `\s` 排除。合成用例覆盖空白字面量与空白动态键 |
| 3 | 请求体解码 receiver 不敏感 | ✅ | `parseBodyCalls` 只认 `h.readJSON` / `json.NewDecoder`；`findReadJSONTarget` 要求 receiver 为 `h`。新增 `TestParseBodyCallsDecodeReceivers` 与 `other.readJSON` 负例 |
| 4 | 死代码符号计数跨包互相抵消 | ✅ | 按「归一化包名 + 符号名」计数（`foo_test` 与 `foo` 同作用域）；新增 `TestFindUnusedExportedTestSymbolsScopesByPackage` 跨包同名合成用例 |

> **仍未机械覆盖**：仍保留 §X 列出的结构性残留；本节的 query 抽取本身仍是正则（动态键已显式红灯），
> 如需彻底 AST 化，应另立任务而不是把「有门禁」读成「已无裸正则」。

---

### Z. 2026-09-24 roadmap §三 #3 死代码纪律收口：前后端两道「零生产引用」导出门禁

> 来源：roadmap §三 #3（原 ➖「维持现状：由覆盖率门禁调整一并治理」）评估后改判为**两道显式门禁**收口；
> 逐条改动与实现细节见 [`CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]` 同日条目。

| # | 条目 | 状态 | 实现与验证 |
|---|------|------|------------|
| 1 | 后端生产导出符号零引用门禁（Gate 1） | ✅ | `apps/server/deadcode_gate_test.go` 扩展：AST 扫描全仓 Go 文件，导出包级符号与导出方法一旦**零生产引用**（仅测试引用同样算死）即红灯并报出文件与处置指引。口径：按「归一化包 clause + 符号名」分组（`foo_test` 与 `foo` 同作用域）、方法裸名归并；豁免 `//nolint` 消音符号、门禁自身反射用例（`reflectiveMethodNames` 7 项，配合成用例兜底）、const 表成员（枚举契约）。**变异验证**：注入 `MutantDeadExportedFunc` → 精确红灯（184 文件扫描、定位 `internal/service/job.go`、给出处置指引）→ 撤回 → 复绿且 grep 残留 0 |
| 2 | 前端非 API 模块导出门禁（Gate 2，关闭文件头盲区 #2） | ✅ | `apps/web/src/deadcode_gate.test.ts` 新增 B 半边：非 `src/api` 模块的每个运行期导出必须被生产代码引用（剥掉 import 与导出声明后裸词计数，仅测试引用 = 死）；每个生产源模块必须被生产代码 import——`.vue` 组件要求**默认导入**（`import type` 不算在用）、`export { … } from` 再导出边计入 import 图（`api/index.ts → ./upload` 实例）、`main.ts` 入口豁免（`index.html` 引用）；`default` 导出与 `as` 重命名导入用**前置断言**拦死（出现即红灯，先改写再进门禁）；自检下限 79 文件 / ≥90 导出 / ≥37 组件防空跑。**变异验证**：注入 `mutantDeadExport` + `MutantOrphan.vue` → 两半同时红灯并指名 → 撤回 → 7/7 复绿 |
| 3 | 门禁红灯清出的死代码清零 | ✅ | 后端 5 个生产死导出删除：`openapi.OpBuilder` 链式配置器（`OpBuilder` / `Param` / `Respond`——端点已全部改为 `r.Operation(...)` 内联 `Op{}` 直注册，链只剩测试引用）与 `service.RegistryOption` / `WithMaxJobs` 选项接缝（原 `SetMaxJobsForTest` 的再包装，零生产调用），`Registry.Operation` 改无返回值、`NewJobRegistry*` 去掉空转选项参数；测试接缝统一为 `FillJobSlotsForTest(t, h)`（`export_test.go`，生产文件零 `*ForTest` 钩子）。前端 2 个真死导出删除：`i18nKeyCount`（`i18n/index.ts`）、`shouldUseMultipart`（`upload.ts`，`uploadObject` 已内联同义判断）；引用方测试同步改写（中英键数一致由 `coverage.test.ts` 集合级比对承担、阈值分支由 `uploadObject` 用例覆盖） |
| 4 | 文档同步（与代码同 commit） | ✅ | `roadmap` §三 #3 行移除（编号不重排，#3 留空号并在 3.2 注中说明）、§四 基线日期与前端数字实测更新；`DEVELOPMENT.md` §3 门禁落点补两半口径与前端落点；`architecture.md` §3 提法扩为两半；`CHANGELOG` `[Unreleased]` 同日条目；本节即证据台账 |

> **仍未机械覆盖**：门禁文件头保留 3 条盲区——类型导出（`vue-tsc` 承担）、`s3api[name]` 动态成员（全仓无此写法）、
> 裸词计数的同名抵消与注释 / 字符串命中（只漏报、不误报，与后端 Gate 1 同向）。`as` 重命名导入与 `default`
> 导出属**前置断言**而非盲区：出现即红灯。

### AA. 2026-09-24 全仓代码审查处置（2 Critical + 20 Required 全清；Nit 31/35 闭环、3 项转登记、1 项判定不成立）

> 来源：[`code-review-2026-09-24.md`](archive/code-review-2026-09-24.md)（该报告即唯一条目清单，逐条状态见其正文）。
> 纪律：TDD——每项先写会失败的测试再改实现，断言外部可见行为（返回值 / HTTP 状态码 / 渲染结果）；
> 被旧测试固化的缺陷行为按审查结论**改预期**（如 `main_test.go` 的 `:8080` 期望由 `true` 翻转为 `false`）。
> 逐条改动与实现细节见 [`CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]` 同日条目。

| # | 条目 | 状态 | 实现与验证 |
|---|------|------|------------|
| C1 | 前端代理 URL 不带凭证 → S3C_TOKEN 部署下预览 / 下载全 401 | ✅ | `proxy.ts` 改带 `Authorization` 的 `fetch` → blob → objectURL（`downloadProxyObject`），401 不再把错误 JSON 当文件静默存盘；`scripts/e2e-real.sh` 注入 `S3C_TOKEN`、`e2e-real/real-backend.spec.ts` 全部 `/api` 调用带 Bearer。**未**用豁免 proxy 鉴权的捷径。实跑 `make e2e-real` **3 passed**（后端 `S3C_TOKEN` 开启、生产同构形态） |
| C2 | `IsLoopbackAddr` 把 `:8080`（空 host）判为回环 → 非回环强制鉴权被绕过 | ✅ | 删 `config.go` 的 `\|\| host == ""`；`main_test.go` 缺陷预期翻转（`:8080` → `false`）+ 新增 `S3C_ADDR=":8080"` 无 token 必须启动失败的测试 |
| R1–R4 | 安全四连：XFF 取首段可绕过限速 / 鉴权失败请求不过限速器 / 破坏性异步操作无审计 / StoreDriver 大小写与未知值绕过明文闸 | ✅ | 限速与审计 IP 改取可信代理追加的**末段**；`withRateLimit` 移到 `withAuth` 外层；`deletePrefixAsync` / `copyManyAsync` 移动模式写 202 前补 `h.audit`（含 jobId）；`FromEnv` 归一化 + `Validate` 白名单拒绝未知驱动 |
| R5–R9 | 正确性：mode=text 丢 versionId / 同步静默截断 / `RelKey` 裸 TrimPrefix / Reap TTL 用 Created / Emit 无终态保护 | ✅ | text 分支传 `versionID` + 契约测试；`SyncResult` 透出 `truncated`（`sync_truncated_test.go`）；`RelKey` 改用 `stripPrefix` 作内核；`Job` 加 `finishedAt` 按其算 TTL（`job_finish_reap_test.go`）；`Emit` 锁内 `if j.done { return }` |
| R10–R13 | 边界与持久化：AccessDenied 不回退流式 / 原子写无 fsync / 建表错误被吞 / `rows.Err()` 未查 | ✅ | 同端点跨账号迁移 `AccessDenied` 也回退 `StreamCopy`；原子写 `Sync()` + rename 后 fsync 父目录、两份实现收敛为 `internal/atomicfile`；`sqlite.go` 建表与迁移错误上抛走启动失败；`List()` 循环后检查并包装 `rows.Err()` |
| R14–R17 | 性能 / 契约 / s3wrap：`List()` 每行 Argon2 解密即脱敏 / 三项契约违背 / presign 挂 metrics / `ErrPartialDelete` 未映射 | ✅ | `List` 查询不解密（合成 `secretSet`）；`ListenAndServe` 失败退出码非 0 + `components.responses` 字段可序列化 + `objectItem.ContentType` 补齐；presign client 不注册 `metricsMiddleware`；`HTTPStatus` 补 `ErrPartialDelete → 409`（已入 `docs/errors.md`） |
| R18–R20 | 死代码三件 + 同步 copy-objects 缺背压 | ✅ | `expvar` 两指标删除；`errTestPresign` / `JobRegistry.Create` / `BucketOrDefault` / `deriveKeyLegacy` 与 `envelope` V2 分支——**源码级门禁测试（先红后绿）+ 迁移 / 内联**：`Create` 迁入 `service/export_test.go`（handler 8 处改 `TryCreate`）、`BucketOrDefault` 删除改内联 `acc.Bucket`、`deriveKeyLegacy` 删除且 `envelope()` 收敛为 V3-only；`copy-objects` 挂 `withStreamLimit` |
| F1–F5 | 前端五项：per-server token 持久化失效 / requeue 无法取消 / 虚拟列表行高漂移 / 组合键可碰撞 / 死事件与防重复守卫 | ✅ | `setTokenPersistent` 迁移并清理 `s3c.token.<id>`；`abortItem` 覆盖 pending 条目；MigratePanel 行高改 `ROW_HEIGHT`；`UploadQueue` 改 `:key="it.id"`；四个死 `error` 事件与零使用 props 删除、四组件加异步提交防重复守卫（双击只发一次） |
| Nit | 后端 18 项 + 前端 17 项（共 35；`withDefaults` 空 no-op 与 MigratePanel 空 if 原排在后端段、实为前端） | ⚠️ **31 ✅ + 3 转登记 + 1 不成立** | 后端已修：`%w` 格式化 nil、RequestTimeout 两表一致、IsEntityTooLarge 冗余匹配、WAL/-shm 0600、DSN `busy_timeout`、KDF 参数上界、`envOrInt` 显式报错、healthcheck 注释、migrate store 故障→500（非 404）、`getBucketInfo` 去 ListBuckets 全量拉取、copyMany 响应形状对齐、sync etag multipart 收敛、`LastError→FirstError`、CSP `connect-src` 单源化、EnumStr 冗余转换。**3 项未按原样修复、转登记 [`KNOWN_ISSUES.md`](KNOWN_ISSUES.md)**：`SameEndpoint useSSL` 硬编码 → **#61**（技术债 ⬜）；**batch 删除编排下沉 `service` → #62（技术债 ⬜，本轮未完成——`objects.go` 删除族与 `copy.go` `copyKeysThenDelete` 仍在 handler，`service` 侧只有 `RunBatch`/`CopyKeys` 无删除编排，注释已在 `objects.go` `deleteObjects` 处指路）**；`stream_copy` 640GB 上限 → **#63**（已决策 ➖，S3 段号上限 × 512MB 容器内存预算的刻意取舍，注释 + 测试钉住默认值）。前端已修：五处导出面死代码清零（`updateToast` / `applyTheme` / `UPLOAD_CONCURRENCY` / `polling` / `normalizeStringArray`，**源码形态门禁先红后绿**）、`requestTab` 收窄 `TabKey`、`regions ?? []` 删除、`getBase`/`setBase` 降级 try/catch、分页 `PAGE_SIZE` 单源、abort 后 Promise settle、0 字节文件放行、`validateDoc` i18n 化、`selectedSize` 增量化、`t` 遮蔽改名、`withDefaults` 空 no-op 删除、MigratePanel 空 if 删除、**7 处可删除行 `v-for` 稳定行键**（含 7 个组件「删除中间行保留原 DOM 节点」测试，改回 `:key="i"` 即 6 红）、**账号回退三段复制提取 `useAccountSelect`**、**RecycleBinPanel / VersionsDialog 虚拟滚动**（复用 `virtualList.ts`）。`App.vue` 双 `JSON.parse` 经复核**不成立**（`App.vue` 全文 0 处 `JSON.parse`），不改（1 项 ℹ️）。**逐项处置明细（35 行）见审查报告「处置明细」表** |
| 门禁 | 全绿实测 | ✅ | 后端 `gofmt -l` 干净 / `go vet` 0 告警 / `go test` **9/9 包**（`go list ./...` 共 9 个，含 R11 新建的 `internal/atomicfile`，**每包 100.0% statements**） / `go build` 干净 / `golangci-lint` **0 issues**；前端 `pnpm lint` 0 告警 / `pnpm test` **67 文件 1110 例** / `pnpm test:coverage` **四指标 100%（4255 / 2908 / 1124 / 3653）** / `pnpm build` + `typecheck:e2e` exit 0；`S3CLIENT_E2E=1 go test ./internal/s3wrap/ -run TestE2E` **4/4 PASS**；`SERVER_PORT=18090 make e2e-real` **3 passed**（本机 8080 被系统 `haproxy` 占用，改端口重跑） |

### AB. 2026-09-28 KNOWN_ISSUES #60–#63 收口（前端测试拆分 / `SameEndpoint` 纳入 `useSSL` / 删除编排下沉 / 640GB 上限证据补齐）

> 来源：[`KNOWN_ISSUES.md`](KNOWN_ISSUES.md) §二。**一个提交**完成 #60 / #61 / #62 三项闭环 + #63 补证据；
> 三者已从 §二 移出、在 §四 编号台账登记「已闭环移除」，编号不回收。
> 纪律：TDD——先写会失败的用例再改实现；#62 为**纯重构**，外部可见行为（HTTP 状态码 / 响应体 / 审计行）逐字不变。
> 逐条改动与实现细节见 [`CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]` 同日条目。

| # | 条目 | 状态 | 实现与验证 |
|---|------|------|------------|
| 60 | 4 个超 1000 行的前端测试文件拆分 | ✅ | **4 → 9 文件，断言逐字搬移**（每个拆出文件头部带「本文件范围 + 指向兄弟文件」注释）：`src/api.test.ts` 1986 行 → `api.test.ts` 887 + `api.transfer.test.ts` 677 + `api.gaps.test.ts` 573；`src/components/MigratePanel.test.ts` 1230 → `MigratePanel.test.ts` 871 + `MigratePanel.branches.test.ts` 463；`src/composables/useObjectActions.test.ts` 1164 → `useObjectActions.test.ts` 872 + `useObjectActions.batch.test.ts` 447；`src/composables/useObjectBrowser.test.ts` 1147 → `useObjectBrowser.test.ts` 558 + `useObjectBrowser.branches.test.ts` 641。**最大 887 行 < 1000**。等价性以 `vitest --reporter=json` 双侧全量对比：拆分前后**测试名清单 diff 为空**、**1110 → 1110 例**、覆盖率**四指标 100%（4255 / 2908 / 1124 / 3653）完全不变**（67 文件 → 72 文件） |
| 61 | `SameEndpoint` 精确判定纳入 `useSSL` | ✅ | `service.SameEndpoint` 改为 6 参签名 `(aEndpoint, aRegion string, aUseSSL bool, bEndpoint, bRegion string, bUseSSL bool)`——裸端点按各自账号 `UseSSL` 补 scheme 后再互比，**显式 scheme 优先于开关**、端点为空时不看开关；新增 `(*s3wrap.Client).UseSSL()`（`s3wrap/client.go`，100% 覆盖）。调用点 3 处同步：`handler/migrate.go`、`handler/migrate_async.go`、`service/sync.go`。用例：`service/zip_test.go` `TestSameEndpoint` 表驱动 **10 → 17 组**（裸端点同/异 TLS、显式 scheme 优先、`useSSL` 对空端点无效等）、`service/gaps_test.go` `TestSameEndpointNormalizeEdges` **7 → 9 断言**（`useSSL` 由固定 `false` 改为可变入参钉死口径） |
| 62 | 批量删除编排下沉 `service`（§AA 遗留 Nit，本轮未完成） | ✅ | 新增 `internal/service/delete.go`（189 行）承载 `DeleteCounts` / `DeletePrefixResult` / `DeleteKeys` / `RunDeletePrefix` / `DeleteKeysBatched` / `MoveKeys`，与 `batch.go` 的 `RunBatch` / `CopyKeys` 同层；`handler/objects.go` **538 → 432 行**、`handler/copy.go` 删 `copyKeysThenDelete`（净 -21 行），handler 只剩 HTTP 边界（入参校验 → 调 service → 写状态码/响应体 → 审计）。**响应体形状不变**（`{"deleted","failed","lastError"}` 与前缀递归口径），故仍按既有先例在 `writeJSON` 处内联构造 map，`TestOpenAPI_EndpointResponseSchemasMatchHandlers` 全量核对不受影响。新增 `internal/service/delete_test.go`（436 行）逐条搬移原 handler 3 个白盒用例（`TestOlRunDeletePrefix*`），并删掉随之死掉的 `handler.s3UserMessageForCode` 与 `olListPagesFake` |
| 63 | 流式复制单对象 640GB 上限（64MB × 10000 段） | ➖ **维持现状，证据补齐** | 新增 `TestMultipartStreamCopyPartSizeIs64MB` 钉住**分段默认值 64MB**——640GB 是「分段 × 段数」两个默认值的乘积，此前只有 `TestMaxMultipartPartsIsProtocolLimit` 钉段数，证据缺一半；复核 `docker-compose.yml` / `docker-compose.prod.yml` 的 server 服务确为 `deploy.resources.limits.memory: 512M`（一块分段缓冲即 64MB）。口径三处一致：`service/stream_copy.go` 注释（段号在**上传前**判定，不误杀第 10000 段的合法对象）→ [`KNOWN_ISSUES.md`](KNOWN_ISSUES.md) #63 → 本行 |
| 门禁 | 全绿实测 | ✅ | 后端 `gofmt -l` 干净 / `go vet` 0 告警 / `go test -race -count=1` **9/9 包、每包 100.0% statements** / `go build` 干净 / `golangci-lint run ./...` **0 issues**；前端 `pnpm lint` **0 告警** / `pnpm typecheck` + `typecheck:e2e` exit 0 / `pnpm test` **72 文件 1110 例**（与拆分前测试名清单逐条一致）/ `pnpm test:coverage` **四指标 100%（4255 / 2908 / 1124 / 3653）** / `pnpm build` OK |

### AC. 2026-09-28 `DEVELOPMENT.md` §7 历史技术债收口（H1 / S7 / D5 三条均已闭环）

> 来源：[`DEVELOPMENT.md`](DEVELOPMENT.md) §7 原题为「已知技术债（来自 2026-09-16 综合评估）」，
> 与 [`KNOWN_ISSUES.md`](KNOWN_ISSUES.md)「问题唯一来源」的约定构成**双源**；逐条复核后三条**实际均已闭环**，
> 只是文档没跟上。本节**不新增问题登记**——§7 改写为「闭环状态 + 现行守卫 + 开发规则」，
> 原登记出处仍指 [`archive/assessment.md`](archive/assessment.md)（H1 / S7 / D5）。以下为本轮实跑验证。

| 原条目 | 复核结论 | 闭环证据（本轮实跑） |
|---|---|---|
| **H1** OpenAPI 注册表与真实 handler 字段不一致（SSOT 失真） | ✅ 已从「人工比对」变成**机械门禁**，不再是待修技术债，只剩流程规则 | `go test ./internal/handler/ -run 'TestOpenAPI\|TestAPIDoc'` 全绿：`TestOpenAPIRequestFieldsMatchHandlerDTOs`（注册表 ⇔ handler 字段全量遍历）、`TestOpenAPI_ContractRequestBodyMatchesHandlers`、`TestOpenAPIRequestDeclarationMatchesHandlerInput`、`TestAPIDocMatchesRoutes`、`TestAPIDocDocumentsRequestBodyFields`、`TestOpenAPI_EndpointResponseSchemasMatchHandlers`（连同 `openapi_path_params` / `openapi_query_params` / `openapi_semantics` / `openapi_shape` 等，共 11 个 `api_doc_test.go` + `openapi_*_test.go` 契约测试文件） |
| **S7 / P0-4** SSE 终态检测在 MigratePanel 与 `useObjectActions` / `DestDialog` 各一份 | ✅ 已收敛为单一实现 `apps/web/src/api/jobs.ts` 的 `subscribeMigrateEvents`，三调用方共用 | `pnpm test src/api.transfer.test.ts src/api.gaps.test.ts` **64 例全绿**，含 `subscribeMigrateEvents: ping+status+Authorization+EOF-done fallback`（EOF 回读兜底 / 合成终态 / 心跳超时） |
| **D5** endpoint 归一化多份实现（行为不一致） | ✅ 已收敛为单一 helper `s3wrap.NormalizeEndpoint`，建 client / 预签名 / SSRF 拨号校验 / 同端判定四处共用 | `go test ./internal/s3wrap/ -run TestNormalizeEndpoint` → `TestNormalizeEndpoint` 与 `TestNormalizeEndpointNeverDoubleScheme` **PASS** |
| 文档活状态块同步 | ✅ | [`code-review-summary.md`](archive/code-review-summary.md) 头部「状态更新」活块追加 **⑥**：记录 #60–#62 闭环 / #63 补证据维持 ➖、前端 67 → **72 文件**、Go 文件 199 → **201**（74 生产 + 127 `_test.go`，9 包 41410 行）、§7 双源消除；正文「⚠️ 待解决的技术问题」的 CORS 与单文件超限两条已在块内追平，**正文时点值仍不回写** |
| 门禁 | ✅ | 只改 `docs/`，复跑 `gofmt -l` 干净 / `go vet` 0 / `go build` 干净 / `go test -race -count=1` **9/9 包、每包 100.0%** / `golangci-lint` **0 issues** / `pnpm lint` 0 告警 / `pnpm test` **72 文件 1110 例** —— 零回归 |

### AD. 2026-09-28 三路五轴复审（后端 handler+service / 后端 s3wrap+store+config / 前端 web）——本轮闭环 4 条，另 15 条转登记 [`KNOWN_ISSUES.md`](KNOWN_ISSUES.md) #64

> 按 `code-review-and-quality` 五轴方法对全仓重新审一轮（三路并行），发现经**逐条亲自读码复核**后只修
> **有把握且已完成红绿**的 4 条；其余**未复核的不直接采信、也不静默丢弃**——登记为
> [`KNOWN_ISSUES.md`](KNOWN_ISSUES.md) **#64**，清单见下方「待处置」表。

#### 已闭环（4 条，全部先红后绿）

| # | 条目 | 状态 | 证据（红 → 绿） |
|---|---|---|---|
| 1 | `service.deletePrefix` 列举循环**无界**：退出条件只有「`maxDelete` 计数到顶」与「token 为空」，而**空页 / token 不前进都不推进计数**，同步 `delete-prefix` 会一直空转到客户端断连并占住 `withStreamLimit` 的 32 个流槽位 | ✅ | 红：`TestRunDeletePrefixStopsAtPageCap` → **`未终止：空页 + 前进 token 导致死循环`**、`TestRunDeletePrefixStopsOnNonAdvancingToken` → `deleted = 100000, want 2000`。绿：移植 §B6 三道守卫——`listMaxPages` 页数上限 + `NextToken == ""` / `NextToken == token` 停止并标截断 + 循环首行 `ctx.Err()`；`TruncateExact` / `CrossLimit` / `ProgressesOnAllFailures` 三个 fixture 的 token 改为**逐页前进**（真实对端不会重复同一 token，常量 token 恰是本轮守卫对象），另补 `TestRunDeletePrefixReturnsOnCancelledContext` |
| 2 | `handler.listPrefixKeys` 同款无界：`copy-prefix`（同步 / 异步）与 `delete-prefix` 异步列举**三处共用**；`copyPrefixAsync` 的列举还跑在 2h 任务超时**之外** | ✅ | 红：`TestOlListPrefixKeysStopsOnNonAdvancingToken` 与 `TestOlListPrefixKeysStopsAtPageCap` → **`listPrefixKeys 未终止：列举循环空转`**（各 5s 超时）。绿：`listPrefixMaxPages` + `ctx.Err()` + token 守卫，结构对齐 `indexDst`；补 `TestOlListPrefixKeysReturnsOnCancelledContext` |
| 3 | **边界差一**：正好收满上限（`listPrefixKeys` 的 `maxCopy` / 100 000）且对端声明列举完成时仍报 `truncated=true`，与 `docs/api.md`「第 limit+1 个起未参与本次操作」的定义冲突，客户端会去重试一个已完成的操作 | ✅ | 红：`TestOlListPrefixKeysExactlyLimitIsNotTruncated` → **`正好收满 limit 且对端声明列举完成 ⇒ 不是截断`**。绿：改用 `indexDst` 的「先判本页丢弃、再判收满、最后看 `p.IsTruncated`」三段式。`deletePrefix` 本就正确（上限检查在 `!IsTruncated` 之后），未改动 |
| 4 | `-healthcheck` 对 **IPv6 字面量监听地址**必然失败：`[::1]:8080` 拼成 `http://::1:8080/api/health`，`url.Parse` 报 `invalid port` → 恒返回 1，Docker `HEALTHCHECK` 会把**完全健康**的服务判死并反复重启（`[::1]:port` 是 `IsLoopbackAddr` 认可、允许不设 token 的合法配置） | ✅ | 红：`TestRunHealthcheck/ipv6_loopback_literal` → **`= 1, want 0`**。绿：`net.JoinHostPort(host, port)` 产出 `http://[::1]:8080/api/health`（实测 `url.Parse` 通过）；无 IPv6 的环境自动 `t.Skip`，`no-port-in-here` 的 fail-closed 用例仍绿 |
| 门禁 | 全绿实测 | ✅ | `gofmt -l` 干净 / `go vet` 0 告警 / `go build` 干净 / `golangci-lint run ./...` **0 issues** / `go test -race -count=1 -coverprofile` **9/9 包、每包 100.0% statements**、**零未覆盖块**（新增守卫分支各补 1 个用例，否则 `service` 会跌到 99.8%）/ `pnpm lint` 0 告警、`pnpm test` **72 文件 1110 例**（前端本轮无改动） |

#### ⬜ 待处置（原 15 条 —— `handler` 审计见 §AE、前端 4 条见 §AF、`config`/`main` 3 条见 §AG、`s3wrap` 2 条见 §AH、前端另 4 条见 §AI、**后端 `store` 1 条见 §AJ**；**2026-09-28 已全部闭环，本表现余 0 条**）

| 区域 | 条目 | 闭环去向 |
|---|---|---|
| 后端 `store` | `store.Open` 的 json 分支**丢弃入参 `storeKey`** 改读环境变量 ⇒ 契约与 sqlite / encrypted 分支不一致，非 `FromEnv` 调用方传了 key 仍明文落盘且无报错（今天唯一生产调用方传的 `cfg.StoreKey` 就来自同一环境变量，**零生产影响**）。曾于 2026-09-28 实测后挂起：直接把 json 分支改走 `newStore` 会让 `store.New` 变成零生产引用、被 `TestNoUnusedExportedProdSymbols` 拦红，而解开需跨 24 个测试文件改 `New` 签名 | ✅ **§AJ**：抽 `openJSON(path, storeKey)` —— 入参非空走 `newStore`、**为空才回退 `New`**，既让入参生效又保住 `New` 的生产引用，**不动签名**即完成修复。至此 #64 的 19 条全部闭环，[`KNOWN_ISSUES.md`](KNOWN_ISSUES.md) #64 移除 |

### AE. 2026-09-28 破坏性操作审计覆盖补齐（#64 Security，3 条先红后绿）

> 来源：§AD「待处置」表的后端 `handler` 一条，亲自读码复核后确认属实并修复。依据是
> [`threat-model.md`](threat-model.md) 边界 A 的 **R（抵赖）** 缓解声明——它明写覆盖
> 「对象删除与前缀删除」，而下面三处会**真实删掉数据**却没有任何 `objects.*` 审计事件，
> 持有效 token 的调用方可不留痕迹地删数据。

| # | 缺口 | 落的事件 | 红 → 绿 |
|---|---|---|---|
| 1 | 同步 `POST /copy-objects` + `deleteSource`（移动 = 复制后删源）；**异步版已按 review R3 补了 `objects.move`，同步版漏了**——同一动作只因走两条路由就可审计性不同 | `objects.move`（`bucket` / `targetBucket` / `total` / `moved` / `failed`） | 红 `同步移动未写 objects.move 审计（源被删却无痕迹，与异步版口径不一致）` |
| 2 | `POST /rename`：复制成功后 `DeleteObject` 删源，源 key 永久消失、全程无审计——把删除藏进「重命名」即可绕开已有的 `objects.delete` 事件 | `objects.move`（带 `key` / `newKey`） | 红 `重命名未写 objects.move 审计（源 key 被永久删除却无痕迹，可绕开 objects.delete）` |
| 3 | `DELETE /version`：永久版本删除，数据不可恢复；回收站 `trash.purge` 反而有审计 | `objects.delete`（带 `versionId`） | 红 `永久版本删除未写 objects.delete 审计（数据不可恢复却无痕迹）` |
| 负向断言 | 纯复制（`deleteSource` 缺省）**仍不得**记 `objects.move`，与异步范式同一口径，避免把只读复制记成移动 | — | `TestCopyManySyncMoveAudits` 内的前后计数比对 |
| 事件名 | 全部复用 `audit.go` **既有稳定事件名**，**不新增契约名**——事件名是日志检索与告警的依赖 | — | — |
| 新测试 | [`handler/audit_coverage_test.go`](../apps/server/internal/handler/audit_coverage_test.go)：`TestCopyManySyncMoveAudits` / `TestRenameObjectAudits` / `TestDeleteObjectVersionAudits` | — | 3 红 → 3 绿；既有 `TestCopyManyAsyncMoveAuditsJobStart` / `TestAudit*` / `TestRenameObject` / `TestTrash` 全部保持绿 |
| 文档 | [`threat-model.md`](threat-model.md) 边界 A 的 R 行补「移动与重命名（同步批量 / 异步批量 / `rename`）与版本永久删除（带 `versionId`）」，闭环指向本节；「最后更新」推到 2026-09-28 | — | — |
| 门禁 | 全绿实测 | ✅ | `gofmt -l` 干净 / `go vet` 0 / `go build` 干净 / `golangci-lint run ./...` **0 issues** / `go test -race -count=1 -coverprofile` **9/9 包、每包 100.0% statements**、**零未覆盖块** / 前端 `pnpm lint` 0 告警、`pnpm test` **72 文件 1110 例** |

### AF. 2026-09-28 前端四条（#64）：sticky error 两处 / `DestDialog` 并发 / `signing` 死状态——4 条先红后绿

> §AD「待处置」表前端第一行的 4 条，逐条亲自读码复核后修复。**每条都先写失败测试再改实现。**

| # | 缺陷 | 修法 | 红 → 绿 |
|---|---|---|---|
| 1 | `RecycleBinPanel.vue` 的 `error` **只写不清**（4 处赋值、0 处清空），而模板 `v-if="!account()"` → `v-else-if="error"` → `v-else-if="bucketSel"` 三分支**互斥** ⇒ 一次失败就用横幅**取代整页**，「重试」成功后仍不清除，必须刷新页面；另 `loadBuckets` 失败时 `bucketSel` 为空，而重试只调 `loadMarkers(true)`（缺 bucket 直接 return）⇒ 横幅永久卡死 | `loadBuckets` / `loadMarkers` **成功即清 `error`**；重试改为 `retry()` = 先 `loadBuckets()` 补桶选择再 `loadMarkers(true)` | 红 `重试成功后错误横幅必须消失: expected true to be false` |
| 2 | `BucketsPanel.vue` 的 `error` 同样无清空点，且 `loadBuckets` 失败分支**不重置 `buckets`**，而 `watch(accSel)` 只清 `selectedBucket` ⇒ 切到凭据失效的账号会把**上一个账号的桶**渲染在当前账号选择器之下 | 成功即清 `error`；失败时 `buckets = []` | 红 `失败后不得渲染上一个账号的桶表: expected true to be false` |
| 3 | `DestDialog.vue` 在 `watch(props.open)` 里**无条件 `busy.value = false`**，而 `ModalDialog` 的 Esc / ✕ / 背景点击三条关闭路径都不看 `busy`，组件又是常驻（`ObjectsPanel` 只绑 `:open`、无 `v-if`）⇒ 飞行中任务期间关掉再开，第二次 `submitDest` 不被挡；先到的那次 `emit('submit')` 还会在第二个任务运行中把弹窗关掉 | **删掉 open watcher 里的复位**——`busy` 本就由 `submitDest` 的 `finally` 在成功 / 失败 / 中止三条路径归位，打开时已是 false，这行复位是多余且有害的 | 红 `在途任务期间 busy 不得被复位: expected false to be true` |
| 4 | `useUploadQueue.ts` 把 `it.status = 'signing'` 与 `it.status = 'uploading'` 写在**同一个同步块**（中间无 `await`）⇒ 渲染永远插不进来，`UploadPanel` 的「签名中…」标签与 `abortItem` 的 `signing` 分支**永不可达**——典型的「被覆盖率掩盖的死状态」 | 把过渡挪到**首次字节进度回调**（`if (it.status === 'signing') it.status = 'uploading'`），presign 的一次网络往返期间该状态停得住；守卫保证不会把已 `cancelled` 的条目改回 `uploading` | 红 `在途且尚无字节进度应停在 signing: expected 'uploading' to be 'signing'` |
| 连带 | 既有用例「`加载失败显示错误与重试」原本靠**错误粘住**才看得到横幅（mount 触发 2 次 `loadBuckets`，第 1 次失败、第 2 次成功——正确行为本就该清掉），夹具改为两个初始调用都失败；`requeue` 用例的中间断言由 `uploading` 改为 `signing`；`桶列表加载失败…` 由「重试为空操作」改为「重试先重拉桶」 | 三处**均为夹具 / 断言随正确行为更新，测试意图未变** | — |
| 事件名 | 不涉及后端契约 | — | — |
| 文档 | [`threat-model.md`](threat-model.md) 无涉 | — | — |
| 门禁 | 全绿实测 | ✅ | `pnpm lint` **0 告警** / `pnpm typecheck` + `pnpm typecheck:e2e` exit 0 / `pnpm test` **72 文件 1114 例**（1110 → 1114，净增 4 条红灯用例）/ `pnpm test:coverage` **四指标 100%（4260 / 2908 / 1124 / 3658）** / `pnpm build` OK；后端本轮未改，`gofmt -l` 干净 / `go vet` 0 / `go test -race` **9/9 包、每包 100.0%** / `golangci-lint` **0 issues** |

### AG. 2026-09-28 `config` / `main` 三条（#64）：显式 env 文件静默回退 / 关停超时无上界 / 预建数据目录不收紧——3 条先红后绿

> §AD「待处置」表 `后端 config / main` 一行的 3 条，逐条亲自读码复核后修复。
> 第 1 条是这 10 条里唯一接近 Required 的：写在 `S3C_ENV_FILE` 里的加固项会**静默失效**。

| # | 缺陷 | 修法 | 红 → 绿 |
|---|---|---|---|
| 1 | 显式 `S3C_ENV_FILE` 路径缺失 / 不可读时 `loadDotEnv()` 直接 `continue` → **静默回退默认值**，与 `config.go` 自述的口径（「静默回退默认值会让运维误以为配置已生效 → 改为拒绝启动」）矛盾。写在该文件里的 `S3C_TOKEN` / `S3C_SSRF_DENY_PRIVATE` / `S3C_TRUSTED_PROXIES` / `S3C_CSP_CONNECT_SRC` 静默失效；`S3C_DATA_DIR` 更会静默丢失 → 启动后账号列表「凭空清空」（原数据没坏，极易引发重复建号） | `loadDotEnvFile` 返回 error、`loadDotEnv` 返回 error；**显式路径**下 `Stat` 失败或 `ReadFile` 失败一律返回新哨兵 `ErrInvalidEnvFile`，经既有 `Config.envErr` → `Validate` 首查上抛使启动失败。**未显式指定时行为不变**（缺 `.env` 是零配置可启动的常态） | 红 `Validate() = <nil>, want ErrInvalidEnvFile`（两条：路径不存在、路径存在但读不到——后者用**目录**触发 EISDIR，不依赖 `chmod 000`，容器里 root 会绕过权限位） |
| 2 | `S3C_SHUTDOWN_TIMEOUT` 只校验 `>=1`、**无上界**：`main.go` 用 `time.Duration(n) * time.Second` 换算，`n > MaxInt64/1e9`（≈92.2 亿）会**溢出成负时长** → `context.WithTimeout` 立即过期 → `srv.Shutdown` 秒回、错误被 `_ =` 丢弃、**退出码仍 0**，日志只有 `shutting down…` / `shutdown complete`，在途大文件流式传输被硬切断却无从察觉（代码注释「ctx 超时在生产不可达」也随之失真） | `Validate` 加上界 `maxShutdownTimeoutSec = 3600`，超界复用 `ErrInvalidEnvValue` 拒绝启动（与既有数值校验同一条错误链） | 红 `Validate(shutdown=9999999999) = <nil>, want ErrInvalidEnvValue` |
| 3 | `os.MkdirAll(dir, 0o700)` 的 mode **只在新建时生效**：Docker volume / systemd `StateDirectory` 预建的 `0755` 此后永远保持（umask 只减不补），与 `threat-model.md` 边界 A「数据目录 0700 + 文件 0600」的声明不符；文件侧早有 `chmodSQLitePerms` 主动收紧到 `0600` 的先例，目录侧没有 | 新增 `ensureDataDirPerm`（`store.Open` 与 `store.AcquireDataDirLock` 建目录后各调一次），**尽力而为**、Chmod 失败一律忽略（对齐 `chmodSQLitePerms` 的「只读挂载 / 非本进程属主不影响开库拿锁」口径） | 红 `Open 后 data dir perm = 0755, want 0700` 与 `AcquireDataDirLock 后 … = 0755, want 0700` |
| 文档 | [`README.md`](../README.md) `S3C_ENV_FILE` / `S3C_SHUTDOWN_TIMEOUT` 两行与 `.env` 查找顺序说明补 fail-closed 与取值上界；[`architecture.md`](architecture.md) 环境变量段、[`DEPLOYMENT.md`](DEPLOYMENT.md) SIGTERM 段同步；[`threat-model.md`](threat-model.md) 边界 A 的 Info disclosure 行注明「启动时主动 chmod 收紧」；`apps/server/.env.example` 两处注释同步 | — | — |
| 门禁 | 全绿实测 | ✅ | `gofmt -l` 干净 / `go vet` 0 / `go build` 干净 / `golangci-lint run ./...` **0 issues** / `go test -race -count=1 -coverprofile` **9/9 包、每包 100.0% statements**、**零未覆盖块**（`awk 'NR>1 && $NF+0==0'`——**必须用默认空白分隔**；用 `-F,` 会取到 `line.col` 字段而永远数出 0）/ 前端本轮未改，`pnpm test` **72 文件 1114 例** |

### AH. 2026-09-28 `s3wrap` 两条（#64）：metadata 值控制字符 400→500 / IDN 端点「建得成却永远连不上」——2 条先红后绿

> §AD「待处置」表 `后端 s3wrap` 一行的 2 条，逐条亲自读码复核后修复。

| # | 缺陷 | 修法 | 红 → 绿 |
|---|---|---|---|
| 1 | `ValidateUserMetadata` 的**值**只查 UTF-8 合法性与长度，**控制字符只有键在查** ⇒ 值含 `CRLF` / CTL 能一路过边界，走到 Go transport 才被 `invalid header field value` 拒发，**边界该给的 400 变成传输期 500**——恰好违背该文件自述的「提前在 API 边界校验，避免落到 S3 端再以 500 形式返回」 | 值侧按 `httpguts.ValidHeaderFieldValue` 的同一口径拒绝 `<0x20`（**HTAB 除外**，空格本就 `<0x20` 之外）与 `0x7F`；`>=0x80` 属 obs-text，仍由既有 UTF-8 校验把关。**已确认不存在头注入**（Go transport 对 `Header.Set` 与直接赋值两条路径都硬拒），所以这是 400/500 的**可用性**问题，不是注入面 | 红 4 条：`value with CRLF` / `lone LF` / `control char` / `DEL` → 全绿；`tab` / `space` / UTF-8 文本三个**放行**用例同批钉住，防误伤正常多行备注 |
| 2 | `ValidateEndpoint` 对非 ASCII 主机名（IDN）按「DNS 失败**按设计 fail-open**」放行 ⇒ 账号建得成，而 Go 的 `net.Resolver` / `http.Transport` **都不自带 IDNA 转换**，`münchen.de` 每次调用都 `no such host`，用户只看到莫名其妙的网络错误、看不出是端点主机名非法 | 在 `isBlockedHostname` 之前加边界拒绝，错误串直接给出出路（`use punycode xn-- instead`）。**不引入 `golang.org/x/net/idna`**——完整 IDNA 映射表不值得为这一处加依赖（仓库依赖纪律：能用标准库就不用第三方）；punycode 形式是纯 ASCII，仍走原流程 | 红 `ValidateEndpoint("http://münchen.de:9000") = nil, want error`；同批钉住 punycode 端点仍放行 |
| 门禁 | 全绿实测 | ✅ | `gofmt -l` 干净 / `go vet` 0 / `go build` 干净 / `golangci-lint run ./...` **0 issues** / `go test -race -count=1 -coverprofile` **9/9 包、每包 100.0%**、**零未覆盖块**（正确口径）/ 前端未改，`pnpm test` **72 文件 1114 例** |

### AI. 2026-09-28 前端另四条（#64）：代次守卫 / 追加重置滚动 / `loadingAll` 提前可点 / 桶列举共用标志——4 条先红后绿

> §AD「待处置」表前端第二行的 4 条，逐条亲自读码复核后修复。至此 §AD 的 15 条**只剩 1 条挂起的 Nit**。

| # | 缺陷 | 修法 | 红 → 绿 |
|---|---|---|---|
| 1 | `VersionsDialog.vue` 的 `load()` **无代次守卫**：关闭再开另一个对象时，上一个对象的列举仍在飞，迟到响应会把 `rows` 覆盖成旧对象的版本（标题已是新对象、列表却是旧的），`finally` 还会把新对象的 `loading` 提前清掉、`catch` 会把旧对象的错误报到新对象头上 | 加 `loadSeq`：每个 `await` 后判代次，过期则**静默丢弃**；`catch` 与 `finally` 都按 `seq === loadSeq` 收口（与 `RecycleBinPanel.loadSeq` 同口径） | 红 `过期的版本列举不得覆盖新对象的列表: expected '…stale…' not to contain 'stale'`；另补「过期请求失败不得上抛」用例 |
| 2 | `ObjectList.vue` 只 `watch(() => props.entries)` 就归零窗口——`entries` 是**过滤+排序后的 computed，每次重算都是新数组身份**，「加载更多」的追加同样换身份 ⇒ 滚到第 300 行点「更多」视口立刻跳回第 1 行（`RecycleBinPanel` 的 `markers` 是 `push` 追加、身份稳定，所以没这问题） | 由 `useObjectBrowser` 暴露**换源代次 `listGen`**（`load(reset=true)` 才递增，`loadMore`/`loadAll` 续页不递增），`ObjectsPanel` 透传给 `ObjectList`；归零只在 **① `listGen` 变 ② 过滤/排序变 ③ 条目数变少（兜底防空白表 §F2）** 三处触发 | 红 `追加不得把用户滚到的位置清零: expected +0 to be 1260`；同批补「换源即使条目变多也归零」的**回归守卫**，既有两条（换源归零 / 缩短归零）保持绿 |
| 3 | `useObjectBrowser.load()` 里 `if (reset) loadingAll.value = false`，而 `loadAll` 的首轮正是以 `reset=true` 加载第一页 ⇒ 批次刚起步就把 `loadingAll` 清掉，「加载全部」按钮中途重新可点、能重复触发把当前批次顶掉 | 改成 `if (reset && seqOverride === undefined)`——`loadAll` 通过 `seqOverride` **认领** `loadingAll` 并由自己的 `finally` 归位；外部导航/刷新（不带 seq）仍然清 | 红 `loadAll 自己发起的 reset 不得清掉自己的 loadingAll: expected false to be true` |
| 4 | `MigratePanel.vue` 源 / 目标桶列举**共用一个 `loadingBuckets` 且都无代次守卫**：先完成的一方在 `finally` 里把标志清掉，另一个 `select` 在请求未完成时就被解除禁用；切账号后旧响应仍会落地 | 拆成 `loadingSourceBuckets` / `loadingTargetBuckets` + `sourceBucketGen` / `targetBucketGen`，`catch`/`finally` 均按代次收口，账号被清空的早退路径也递增代次作废在飞请求 | 红 `目标完成不得解锁仍在飞的源 select: expected undefined to be defined`、`过期列举不得覆盖新账号的桶` |
| 覆盖率补强 | 上述修复新增的 **stale / catch 分支**必须逐条可执行（否则四指标掉到 99.97% / 99.82% 被门禁拦下） | 补 6 条：源列举失败清空、目标列举失败清空、源过期成功不落地、**源过期失败不清空**、**目标过期失败不清空**、**过期版本列举失败不上抛** | 红 → 绿，`pnpm test:coverage` 回到**四指标 100%** |
| 连带 | `ObjectsPanel.test.ts` 的 `makeBrowser()` 未提供 `listGen` → 运行时刷 `Invalid prop … got Undefined`；helper `opts()` 被 prettier 折成 `findAll(…)\n[i]` 触发 `no-unexpected-multiline` | mock 补 `listGen: ref(0)`；helper 改取中间变量（**该 error 曾让 `pnpm lint` 红灯**） | lint 0 告警 |
| 门禁 | 全绿实测 | ✅ | `pnpm lint` **0 告警** / `pnpm typecheck` + `pnpm typecheck:e2e` exit 0 / `pnpm test` **72 文件 1126 例**（1120 → 1126）/ `pnpm test:coverage` **四指标 100%（4294 / 2934 / 1130 / 3677）** / `pnpm build` OK；后端未改，`gofmt -l` 干净 / `go vet` 0 / `go test -race` **9/9 包、每包 100.0%** / `golangci-lint` **0 issues**；**真实 E2E 三项实跑（2026-09-28 收口验收）**——`S3CLIENT_E2E=1 go test ./internal/s3wrap/ -run TestE2E` **4/4 PASS**（`TestE2ERustFS` / `TestE2EBatch1` / `TestE2EBucketSettings` / `TestE2ETrash`，自起 `rustfs/rustfs:1.0.0-rc.3` 于 `127.0.0.1:9000`）、`SERVER_PORT=18090 make e2e-real` **3 passed**（真实后端 + 真实 RustFS + 真实产物，`S3C_TOKEN` 生产同构；本机 8080 被系统 `haproxy` 占用故改端口；跑完容器与端口已核验自动回收）、`pnpm e2e` mock 版 **15 passed** |

### AJ. 2026-09-28 KNOWN_ISSUES #64 闭环：`store.Open` 的 json 分支不再丢弃入参 `storeKey`

> #64 是三路五轴复审 19 条中**唯一未闭环**的一条，此前登记为「Nit，实测后挂起」，
> 挂起理由写在 §AD「待处置」表：改 json 分支走 `newStore` 会让 `store.New` 变成零生产引用，
> 被 `TestNoUnusedExportedProdSymbols` 拦成红灯（已实测）；而解开它需要改 `New` 签名，
> 代价是跨 24 个测试文件的重构。本次换了个切口——**保留 `New` 这条生产调用路径**——
> 于是不动签名也修好了。至此 #64 的 19 条（§AD 4 + §AE 3 + §AF 4 + §AG 3 + §AH 2 + §AI 4）**全部闭环**。

| # | 项 | 证据（红 → 绿） |
|---|---|---|
| 1 | **缺陷**：`Open(dataDir, driver, storeKey)` 的 json 分支无条件 `return New(path)`，`New` 回头读 `S3C_STORE_KEY` ⇒ **入参被丢弃**，与 sqlite / encrypted 两分支「显式透传 `storeKey`」的契约不一致；非 `FromEnv` 调用方传了 key 仍明文落盘且无报错。今天唯一生产调用方（`main.go:85`）传的 `cfg.StoreKey` 恰来自同一环境变量，**零生产影响**（属契约一致性，非线上缺陷） | — |
| 2 | **修法**：抽 `openJSON(path, storeKey)` —— 入参非空走 `newStore(path, storeKey, false)`、**为空才回退 `New`**。回退分支让 `New` 仍有一条生产调用路径（`TestNoUnusedExportedProdSymbols` 保持绿），同时也保留了「不传 key、靠环境变量」的既有用法 | 先红：`internal/store/store.go: 导出func "New" 零生产引用`（把 json 分支直接改成 `newStore` 时的实测报错）→ 采用「非空入参走 `newStore`、空入参走 `New`」后 `internal/store/open.go:56` 保留该引用，门禁转绿 |
| 3 | **决定性红灯**：环境变量清空 + 显式入参 ⇒ 仍须加密。修前 `New` 读到空 env ⇒ 明文落盘且无报错，正是「入参被丢弃」的可观测形态 | `TestOpenJSONStoreKeyBeatsEmptyEnv` 红：`Open(json) 丢弃了入参 storeKey：S3C_STORE_KEY 为空时 secretKey 明文落盘且无报错` → 绿；附 `Get` 读回断言 `sk-secret` 微妙不变 |
| 4 | **反向守卫**（防「入参优先」被误读成「只用入参」）：入参为空但环境变量非空 ⇒ 仍加密；且用错的 key 重开必须解不开，证明加密用的是环境变量那把 | `TestOpenJSONFallsBackToEnvWhenKeyEmpty` / `TestOpenJSONEnvFallbackStillEncrypted` / `TestOpenJSONEnvFallbackKeyRecovers` 三条同批上，**新用例直接绿**（它们守的是未被改动的既有语义） |
| 5 | **向后兼容**：两处（入参与环境变量）都为空时仍明文落盘，permissive 读旧明文库的语义不变 | `TestOpenJSONNoKeyAnywhereStaysPlaintext` 新用例直接绿 |
| 6 | 文件：`open.go`（+`openJSON`、注释写明 StoreKey 契约）/ 新增 `open_storekey_test.go`（4 例，含决定性红灯）/ 新增 `openjson_env_test.go`（2 例，反向守卫） | 测试文件本身也在导出符号门禁扫描范围内，`TestNoUnusedExportedTestSymbols` 绿 |
| 门禁 | 全绿实测 | ✅ | `gofmt -l .` 干净 / `go vet ./...` **0 告警** / `go build ./...` OK / `golangci-lint run ./...` **0 issues** / `go test -count=1 ./...` **9/9 包** / `internal/store` **100.0% statements、零 `count==0` 块**（`awk 'NR>1 && $NF+0==0'` 为 0）/ `go test -race ./internal/store/` 通过 / `TestNoUnusedExportedProdSymbols` 绿；前端本轮未改 |

> **文档同步**：[`KNOWN_ISSUES.md`](KNOWN_ISSUES.md) #64 移除（§二 表格行 + §四 台账 + 头部已闭环编号区间收紧至 `#60–#64`）；
> [`CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]` 记一条「修复」；本节为证据台账落点。

### AK. 2026-09-29 前端测试与宿主 `NODE_ENV` 解耦（修复 33 文件 / 246 例全红，新增 2 例守卫）

> **背景**：本机跑 `pnpm test` 得到 **33 文件 / 246 例全红**，曾被怀疑是 Node 26 与 vitest
> 不兼容。实测**否**：同一份代码在 `node:24.21.0-alpine` 与 `node:26.5.0-alpine` 容器里
> 都是 **72 文件 / 1126 例全绿**。真正的变量是宿主环境变量 **`NODE_ENV=production`**。

| # | 项 | 证据 |
|---|---|---|
| 1 | **根因**：Vue 的 node 入口 `vue/index.js` 在运行时按 `process.env.NODE_ENV` 二选一加载 CJS 产物（`production` → `dist/vue.cjs.prod.js`，其余 → `dist/vue.cjs.js`）。宿主带 `NODE_ENV=production` 时 vitest 把 Vue 解析到 **prod 构建**，而 `@vitejs/plugin-vue` 编译 SFC 产出的绑定属于 **dev 构建** ⇒ 进程内两份 Vue 实例、reactive 状态不连通 | 对照实验：`NODE_ENV=production` → 16 failed；`test` / `development` / 未设 → 全绿（同 node_modules、同 pnpm、同 Node） |
| 2 | **误导性症状**（排查成本的主要来源）：不是「找不到模块」，而是 `[vitest] No "toasts" export is defined on the "./store" mock` —— 而 `src/store.ts:64` 的 `toasts` 导出**确实存在**，只是属于另一份 Vue 实例。连带 `Cannot call text on an empty DOMWrapper` / `w.vm.xxx is not a function` 等 246 例次生失败 | 全量错误分类：根因仅 2 条 `No "x" export is defined`，其余 244 条均为其连带 |
| 3 | **修法**：在 `apps/web/vite.config.ts` 顶部、`import` 之前执行 `if (process.env.VITEST) { process.env.NODE_ENV = 'test' }`。位置必须在 import 之前——vitest 的 `test.env` 注入时机晚于依赖解析，**实测无效**（试过 `test.env` / `resolve.alias` / `test.define` / `server.deps.inline` 四种写法，只有顶部改写有效） | 见下表「为什么必须是 VITEST 条件」 |
| 4 | **守卫**：新增 `src/vite_env_guard.test.ts`（2 例）把不变量钉住——删掉那行即变红，且报错文案直接给出根因与修法位置 | **红灯实测**：移除 config 里的改写后，2 例均红（`expected 'production' to be 'test'`）→ 恢复后绿 |
| 5 | **为什么必须带 `if (process.env.VITEST)` 条件**：无条件改写会让 `vite build` 也读到非 production 值，把 Vue 的 dev/warn 分支打进产物 —— 实测 bundle **366.66 kB → 424.72 kB（+58 kB，gzip 112.71 → 129.45 kB）**。加条件后回到基线 | bundle 实测：无条件钉值 424.72 kB；加 `VITEST` 条件后 **366.66 kB**（与原始 config 逐字节同 hash `index-BH1Mf0FU.js`） |
| 6 | **纠正一处早前误判**：初稿注释曾写「'development' 在 --coverage 下更慢」；补跑证实那次 `App.corruptStorage.test.ts` 的 5s 超时是**并发跑批时的资源竞争**，与取值无关。最终 config full+coverage 连跑 7 次全绿 | baseline 3/3 绿、修后 unset 3/3 绿、修后 prod 4/4 绿 |

| 门禁 | 全绿实测 | ✅ |
|---|---|---|
| 前端 `pnpm test`（宿主 `NODE_ENV=production`） | **73 文件 / 1128 例全绿**（修复前 33 文件 / 246 例全红） | ✅ |
| 前端 `pnpm test`（宿主 `NODE_ENV` 未设） | **73 文件 / 1128 例全绿** | ✅ |
| `pnpm test:coverage`（宿主 `NODE_ENV=production`） | 四指标 **100%（4292 / 2934 / 1130 / 3675）** | ✅ |
| `pnpm test:coverage`（宿主 `NODE_ENV` 未设） | 四指标 **100%（4294 / 2934 / 1130 / 3677）** | ✅ |
| `pnpm lint` / `pnpm typecheck` / `pnpm typecheck:e2e` | 0 告警 / exit 0 / exit 0 | ✅ |
| `pnpm build`（宿主 `NODE_ENV=production`） | OK，**366.66 kB / gzip 112.71 kB**，产物 hash 与改动前一致（无体积回归） | ✅ |
| 后端（本轮未改） | `gofmt -l` 干净 / `go vet ./...` 0 告警 / `go build ./...` OK | ✅ |

> **覆盖率差值的说明**：宿主带 `NODE_ENV` 时统计为 4292/3675、未设时 4294/3677，
> 差值来自 `ObjectList.vue` 与 `PreviewOverlay.vue` 各一行 Vue dev/prod 分支插桩差异
> （已逐文件比对确认，**两种环境下四指标均为 100%**，无任何 uncovered 行）。

> **文档同步**：[`ROADMAP.md`](ROADMAP.md) §四 门禁基线两行更新（1126 → 1128 例、73 文件；
> 覆盖率补注两种宿主环境的口径）与 §5.2 新增依赖 **E11**；[`CHANGELOG.md`](../CHANGELOG.md)
> `[Unreleased]` 记一条「修复」；本节为证据台账落点。

---

### AL. 2026-09-29 对照通用 AGENTS.md 模板补齐代理治理与配置 SSOT（新增 4 个文件 / 改 8 处引用）

> **背景**：拿到一份通用 `AGENTS.md` 模板（22 节）作为「文档规范」清点本仓库文档面。
> 结论分两类：**该补的补**（代理权限边界 / AI 披露 / 配置 SSOT / LLM 导航 / 默认审查者），
> **不该建的明确不建**（提示词 / 模型卡 / 数据集卡 / 评估基准 / 护栏——本仓库无 AI 功能、
> 无自训练模型、无数据集，建了就是无人维护的空壳）。后者的判断与理由写在
> [`AI_POLICY.md`](AI_POLICY.md) §0，并写明「何时才需要」。

| # | 项 | 处置 | 证据 |
|---|---|---|---|
| 1 | **代理权限边界缺失** | 新增 [`docs/AI_POLICY.md`](AI_POLICY.md)：五档代理模式（**默认补丁模式**，禁止自行进入发布模式）、权限矩阵（允许 / 需确认 / 禁止三档）、MCP 工具权限口径、AI 披露模板、DoD、多代理协作、故障排查、升级渠道 | 与 [`AGENTS.md`](../AGENTS.md) 硬约束逐条对齐（TDD / 文档同步 / 分层 / SecretKey / 死代码）；`AGENTS.md` 只加两行指针，**保持短小**（该文件受注入字节预算约束） |
| 2 | **配置项无 SSOT，且旧矩阵漏登** | 新增 [`docs/CONFIGURATION.md`](CONFIGURATION.md)：18 个 `S3C_*` 全量矩阵（真值取自 `config.FromEnv` / `Validate`）、`.env` 查找顺序、**fail-closed 清单 8 条**、客户端存储键 | 清点时抓到 README 旧矩阵**漏登 3 项**：`S3C_LOG_JSON`、`S3C_EXPOSE_OPENAPI`、`S3C_CSP_CONNECT_SRC`（三者都只在 `.env.example` / 代码里存在）。README 改为「最常用 5 项 + 指向 SSOT」，消除双源 |
| 3 | **无 LLM 导航索引** | 新增仓根 [`llms.txt`](../llms.txt)（硬约束 / 门禁命令 / 文档索引 / 禁区 / 需确认清单） | llms.txt 约定固定查 `/llms.txt`，故根目录约定文件 **3 → 4**；[`AGENTS.md`](../AGENTS.md) 与 [`DEVELOPMENT.md`](DEVELOPMENT.md) §4 命名约定**两处同 PR 同步**（防白名单分叉） |
| 4 | **无默认审查者** | 新增 [`.github/CODEOWNERS`](../.github/CODEOWNERS)：默认 owner + 安全敏感面（`store` / `s3wrap` / `middleware` / `SECURITY.md` / `threat-model.md`）+ 供应链（两套 CI / Dockerfile）+ 治理文件分区 | 文件名为 GitHub 按字面固定（小写不生效），属命名白名单 **①**，已同 PR 登记进 `AGENTS.md` 与 `DEVELOPMENT.md` §4 两处 |
| 5 | **PR 模板缺 AI 披露与回滚** | `.github/PULL_REQUEST_TEMPLATE.md` 补「AI 使用披露」段（可直接粘贴的模板）与「风险与回滚」段，验收清单加「文档同步」一项 | 对齐 [`AI_POLICY.md`](AI_POLICY.md) §5 与模板 §17「PR 必须包含」 |
| 6 | **两处死引用（顺带发现）** | [`SECURITY.md`](../.github/SECURITY.md) 的「**邮件**：维护者邮箱（见 CONTRIBUTING.md）」与 [`CODE_OF_CONDUCT.md`](../.github/CODE_OF_CONDUCT.md) §执行的「CONTRIBUTING.md 中列出的渠道」**都指向不存在的内容**——`CONTRIBUTING.md` 从未列过邮箱或任何沟通渠道 | `CONTRIBUTING.md` 新增「联系与支持」段，列**真实存在**的渠道（Issue 模板 / GitHub 私有漏洞报告 / `@weilai1949`），并显式声明**本仓库不公开安全邮箱**；SECURITY.md 改为唯一保证可达的私有报告渠道 |
| 7 | **工具链：Node 24 → 26** | `24.21.0` → `26.10.0`，覆盖 4 个 GitHub workflow + `.gitlab-ci.yml`（镜像 / NodeSource ×2）+ `apps/server/Dockerfile` + README / CONTRIBUTING 环境要求 + roadmap E9 | **三源核对**：nodejs.org `dist/index.json` / `docker run node:26-alpine` → `v26.10.0` / NodeSource nodistro `Packages` → `26.10.0-1nodesource1`。⚠️ Node 26 当前 `"lts": false`（**Current 非 LTS**，2026-10 转 LTS），Dockerfile 注释已如实标注。门禁 `TestNodePinnedToPatchVersion` 继续守住精确 patch |
| 8 | **补门禁而不只补文档** | 新增 [`config_doc_gate_test.go`](../apps/server/config_doc_gate_test.go)：断言配置 SSOT 收录 `internal/config` 读取的**每一个** `S3C_*` 变量 | 本次抓到「README 旧矩阵漏登 3 项」正是这类漂移，人工比对必漏。**变异验证**：把 `docs/CONFIGURATION.md` 的 `S3C_CSP_CONNECT_SRC` 改名 → 红灯并点名该变量；还原 → 绿灯。起草时另加的「`S3C_*` 只能由 config 包读取」**实测被证伪后删除**——`internal/store/store.go` 的 `New` 直读 `S3C_STORE_KEY` 是 `openJSON` 入参为空时的**有意回退**（`Open` 注释写明的 StoreKey 契约，#64 那轮修复的产物），不为它发明仓库并不存在的规则，改在门禁文件头如实记录该残留 |

> **有意不建**（模板 §16 要求，但本仓库结构性不适用）：`prompts/`、`docs/ai/prompts.md`、
> `docs/ai/evaluation.md`、`docs/ai/model-card.md`、`docs/ai/dataset-card.md`、
> `docs/ai/guardrails.md`。模板自身写的是「**如果**仓库包含 AI 功能，必须管理」——本仓库是
> S3 客户端，不含任何 AI 功能。判断与「何时才需要」见 [`AI_POLICY.md`](AI_POLICY.md) §0 的对照表。

> **文档同步**：[`AGENTS.md`](../AGENTS.md)（两根指针 + 根目录 4 约定 + 白名单加 `CODEOWNERS`）、
> [`README.md`](../README.md)（文档列表 + 配置段改指向 SSOT + Node 要求）、
> [`DEVELOPMENT.md`](DEVELOPMENT.md) §4（命名规则 + 文档同步表 3 行）、
> [`ROADMAP.md`](ROADMAP.md) E9、[`CONTRIBUTING.md`](../.github/CONTRIBUTING.md)（环境要求 + 联系与支持）、
> [`SECURITY.md`](../.github/SECURITY.md)、[`CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]` 两条；
> 本节为证据台账落点。**历史台账不追溯篡改**：`FEATURES.md` §E / §T / §AK 中记载「Node 统一到
> `24.21.0`」（以及 §AK 的 `node:24.21.0` vs `node:26.5.0` 对照实验）的条目记录的是**当时状态**，
> 保持原样；`ROADMAP.md` 的 E9 是**现状表**，故已同步为 `26.10.0`。

---

### AM. 2026-09-29 文档命名规则收敛为「元文档大写 / 内容文档小写」（`docs/` 下 6 个元文档恢复大写，引用同步 607 处）

> **起因**：`docs/` 下 21 个文件里**只有 `KNOWN_ISSUES.md` 一个大写名**，而它是 2026-09-24 作为
> 「白名单例外」单独登记的——等于**同一类文档被拆成两套规则**，读者无法从名字判断该不该大写。
> 本轮把它收敛成一条**可判定**的标准：描述「**仓库自身如何运作**」的元文档用大写；描述
> 「**产品是什么 / 怎么用**」的内容文档用小写 kebab-case。

| # | 项 | 处置 | 证据 |
|---|---|---|---|
| 1 | **6 个元文档恢复大写** | `configuration.md→CONFIGURATION.md`、`deployment.md→DEPLOYMENT.md`、`ai-policy.md→AI_POLICY.md`、`development.md→DEVELOPMENT.md`、`features.md→FEATURES.md`、`roadmap.md→ROADMAP.md` | 全部走**两步** `git mv`／`mv`（纯大小写改名在 macOS / Windows 上单步会静默失败——2026-09-17 那条历史记录自己就写明了这一点）。未跟踪的新文件（`configuration.md` / `ai-policy.md`）用 `mv`，`git mv` 对未跟踪文件会 `fatal: not under version control` |
| 2 | **内容文档维持小写** | `api.md`、`architecture.md`、`errors.md`、`threat-model.md` + `archive/`、`decisions/` 全部文件 | 与通用模板一致：模板自身把 `docs/architecture.md`、`docs/api/openapi.yaml` 写作小写，而把 `CONFIGURATION.md` / `DEPLOYMENT.md` / `AI_POLICY.md` 写作大写 |
| 3 | **引用同步 607 处 / 44 个文件** | 脚本化替换 + 逐条核对 | **可执行路径是重点**：`scripts/release-version.sh` 4 处 sed 目标（漏改则发版脚本版本号静默不更新）、`doc_number_gate_test.go` 的 `file:` 字段（该门禁自带「命中 0 次即红灯」，漏改会直接失败而非静默失明）、`config_doc_gate_test.go` 的 SSOT 路径、以及 `internal/{config,handler,s3wrap,store}` 与 `apps/web/src` 的注释 |
| 4 | **历史条目不改写（唯一的豁免）** | `CHANGELOG.md` 的 **2026-09-17「5 个文档名小写化，命名规范收口」** 那条**整行冻结**；**2026-09-24「文档命名规则登记例外」** 那条只更新其中的 `docs/development.md` 实时路径，括号内旧名列表保持原样 | 若一并替换，那条小写化记录会变成「`FEATURES.md` → `FEATURES.md`」这类自相矛盾的文本。改完全仓复检：**残留小写名有且仅有这两处**。**按标题检索、不写行号**——行号会随新增条目腐烂（本仓库已有同类教训） |
| 5 | **归档件只修链接** | `docs/archive/` 冻结快照中指向被改名文档的**链接目标**同步更新，正文结论一字未动 | 归档纪律是「不移除、不回写、不改写历史结论」，但**链接不悬空**同样成立；沿用 2026-09-24 `todolist.md → KNOWN_ISSUES.md` 时的同一处置 |
| 6 | **规则本体两处同 PR 重写** | `AGENTS.md` 与 `DEVELOPMENT.md` §4 命名约定：由「白名单 + 不得新增大写名」改为「按性质二分」，写入**三代规则沿革**与防回退提示；原「目录」条明确为只约束**目录名** | 两处必须同改（防分叉）是本仓库既有硬约束；命名规则本体属第 4 类「文档命名 / 存放位置 / 归档」同步项 |

> **沿革（三代规则，历史条目均不改写）**：① 2026-09-17 把当时的「台账类大写」全部**小写化**
> （理由：无任何工具按文件名匹配）；② 2026-09-24 单独登记 `KNOWN_ISSUES.md` 为**白名单例外**；
> ③ **2026-09-29（本节）改为「元文档大写 / 内容文档小写」二分**——即元文档恢复大写、
> 内容文档维持小写。三轮取舍逐条记在 [`CHANGELOG.md`](../CHANGELOG.md)。

> **文档同步**：[`AGENTS.md`](../AGENTS.md) + [`DEVELOPMENT.md`](DEVELOPMENT.md) §4（命名规则两处）、
> [`CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]` 一条、以及上表第 3 项列出的全部引用点；
> 本节为证据台账落点。

---

### AN. 2026-09-29 文档覆盖矩阵收口（11 个新文档 + 3 项供应链门禁 + 机器可读契约 + 性能基线）

> **诊断**：以 10 层基线（社区健康 / 元信息 / 开发协作 / 架构决策 / 接口契约 / 运维可靠 /
> 安全供应链 / 用户文档 / AI 时代新增层）逐层盘点，结论是——**对贡献者与 agent 已很完善，
> 但对运维者（1.5/9）与终端用户（2/7）几乎没有专门文档，供应链证据（SBOM / 签名 / SAST）也缺**。
> 逐项补齐的取舍与证据如下。所有「建议值」类内容（SLO 目标、RTO/RPO、告警阈值）在文档内
> **均显式标注为未强制**，不冒充既成事实。

| # | 层 | 缺口 | 处置 | 证据 |
|---|---|---|---|---|
| 1 | 供应链 | **无 SAST**（JS/TS 侧完全没有） | 新增 `codeql.yml`：`go` + `javascript-typescript`，`security-and-quality`；独立 workflow 以隔离 `security-events: write` 权限 | 新增 SHA 经 GitHub API **三重核验**：`refs/tags` → 若 `type=tag` 解引用到 commit → `GET /commits/<sha>` 200（annotated tag 的 tag-object SHA 用在这里就是**幽灵 SHA**，本仓库记录过该失效模式） |
| 2 | 供应链 | **无 SBOM** | BuildKit `sbom: true`（OCI attestation）+ 用**已 pin 的 Trivy** 产出可下载 CycloneDX 文件；不引入第二套 SBOM 工具 | 按 **digest** 扫而非 tag，避免 tag 并发覆盖导致「扫的不是推的那份」 |
| 3 | 供应链 | **产物无证明** | `actions/attest-build-provenance`：镜像（`subject-name`+`subject-digest`，`push-to-registry`）与三平台桌面产物（`subject-path`）各一份 | 与 SHA256SUMS 互补：后者证明「文件没被改」，前者钉住「文件 ← commit ← workflow」；**均不解决** SmartScreen/Gatekeeper 发布者信誉（仍 roadmap E6） |
| 4 | 供应链 | 「Actions 全 pin SHA」**只有文字约定** | 新增门禁 `TestWorkflowActionsAreShaPinned`（豁免 `./` 本地 action 与 `docker://`） | 变异验证：把 `codeql-action/init@<sha>` 改成 `@v3` → 红灯点名；还原 → 绿灯。当前 **6 workflow / 46 引用** |
| 5 | 接口契约 | 契约只在**运行时**可得（默认 404），外部工具 / AI 代理必须先跑服务 | 提交 `docs/api/openapi.json`（4 579 行）+ golden 门禁 | `TestCommittedOpenAPISpecMatchesRuntime`（比对前 `json.Compact` 归一化——buildSpec 走 `map[string]any`+`json.Marshal`，key 有序确定，故归一化后逐字节可比）+ `TestCommittedOpenAPISpecIsDiscoverable`（位置 / 可解析 / version 对齐 `main.go`）。**变异验证**：改 title → 红灯并打印再生成命令。用 JSON 非 YAML：`internal/openapi` 明确不引额外依赖，且 JSON ⊂ YAML 1.2 |
| 6 | 运维 | **无 Runbook / SLO / 备份恢复 / DR / 事故响应**（最大缺口 1.5/9） | 新增 `OPERATIONS.md`（722 行 / 11 节），含 16 个指标逐条、Runbook R-1..R-10（症状 / 判据 / 处置 / 升级）、备份恢复、DR、容量 | 指标名逐字取自 `handler/metrics.go` + `s3wrap/metrics.go`。**顺带查明两条易误判的事实**：① `json`/`encrypted` 驱动的 `Ping()` 恒 `nil`，故 `/api/health` 503 与 `s3c_store_up 0` **只对 sqlite 真实有效**；② 仓库**无密钥轮换 / 导出工具**且 `AccountView` 不回传 `secretKey` ⇒ **换 `S3C_STORE_KEY` = 重新录入全部账号**。另显式列出「本版本没有的观测面」 |
| 7 | 用户文档 | **无用户手册 / FAQ / 排障**（2/7） | 新增 `user-guide.md`（654 行）；README「功能」是 bullet、`FEATURES.md` 是**台账**（做了什么）而非**指南**（怎么用），不可互替。另补 **README 界面截图**（`docs/images/`） | 16 条快捷键逐条抄自源码；308 条中文引文与 `i18n/messages/*.ts` 逐条比对全命中。截图由 `apps/web/e2e/screenshots.spec.ts` 从**真实构建产物**生成，spec 本身先断言「渲染成功且无『无法连接后端』噪声」再截图——**首版正因缺这道断言，把带 503 红条的图当成了配图**（已修正为只桩 `/api/health` + `/api/accounts`）。**未能核实即不写**：`POST /api/migrate/sync` 前端全仓无调用 → 明确标注「仅 API、界面无入口」 |
| 8 | 治理 | 无 `SUPPORT.md` / `GOVERNANCE.md` / 兼容与弃用政策 | 新增三者 | **如实写明「当前没有」**：无 SLA / 商业支持 / 支持邮箱 / **安全邮箱**；治理为**单人维护**，无委员会 / 选举 / 投票 / 时间表；`/api/*` 无版本前缀、无 Accept 协商、无 LTS 线；弃用政策**尚无一例弃用且不预先承诺时间窗**（「预先不做做不到的数字」） |
| 9 | 工程 | 全仓 `func Benchmark` **为 0** | 三个 `bench_test.go` + `PERFORMANCE.md` | 三条实测结论：① 加密写入 **~30 ms / 瞬时 64 MiB**（Argon2id t=2,m=64MiB,p=4，**有意设计**；并发写须计入内存预算）；② 账号写入 **O(n)**（`persistLocked` 重写整文件：`-benchtime=200x` 335 µs vs `-benchtime=1s` 11.2 ms，两个数量级）；③ `PresignPut` **33 µs / 362 allocs**（点击路径无感，**不可**放进按对象循环） |
| 10 | 工程 | 无术语表 / i18n 指南 / 无障碍说明 | 新增 `glossary.md`（64 条，逐个 grep 复核 0 编造）、`i18n.md`、`accessibility.md` | i18n 步骤与真实文件结构逐一对应；无障碍**零合规断言**——全文 6 处「WCAG/合规」均为否定句，并列出 12 条「未做」（未审计 / 无 axe/pa11y / 无屏幕阅读器实测 / `prefers-reduced-motion` 未处理 / `<html lang>` 不随语言更新 / `textarea` 不在统一焦点样式内等） |
| 11 | 硬约束 | **死代码 `isTopKeydown`**（生产零调用，靠注释与测试「续命」） | 删除导出 + 仅测它的用例；测试改为**派发真实 keydown 观测行为**（同时消除白盒断言：此前用 `isTopKeydown()` 直接读内部栈，证明不了「真的有且只有栈顶收到事件」） | 门禁**漏报口径未修**（`usageBody()` 剥掉声明后按裸词计数，注释里的同名词即算引用）→ 登记 **KNOWN_ISSUES #65** |
| 12 | 写文档时实测发现、**只记录未修** | 4 处真实问题（非文档问题） | 全部登记进 [`KNOWN_ISSUES.md`](KNOWN_ISSUES.md) §二，**不留在文档正文里当口头待办** | **#66** GitLab 侧缺 SAST；**#67** 前端可访问性三处（`<html lang>` 不随语言更新 / 未处理 `prefers-reduced-motion` / `textarea` 缺统一焦点轮廓）；**#68** nginx `log_format main` 缺 `$http_x_request_id`，跨层日志关联断裂 |
| 13 | CI 一致性 | 新增 GitHub-only workflow 会让双平台漂移 | `DEVELOPMENT.md` §3 对照表补 `codeql.yml` 行 + 触发事件表更新 + 写明**未镜像的理由与代价** | 沿用 `release-desktop.yml` / `publish` 的「未镜像必须显式登记」先例；GitLab 侧缺口进 KNOWN_ISSUES（#66）而非装作不存在 |

> **文档同步**：[`README.md`](../README.md) 文档段按「用法 / 运维与安全 / 贡献与治理 / 归档」重组、
> [`AGENTS.md`](../AGENTS.md) 文档入口表 +5 行、[`DEVELOPMENT.md`](DEVELOPMENT.md) §3（门禁落点 + CI 对照表 + 触发表）
> 与 §4（文档同步表 +5 行、命名规则清单更新）、[`llms.txt`](../llms.txt) 索引、
> [`CONTRIBUTING.md`](../.github/CONTRIBUTING.md)（发版同步文件数 17 → **20**、指向 SUPPORT/GOVERNANCE）、
> [`CODEOWNERS`](../.github/CODEOWNERS) +7 条敏感面、[`KNOWN_ISSUES.md`](KNOWN_ISSUES.md) #65–#68、
> [`CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]` 一条、`scripts/release-version.sh`（+3 个 sed 目标）、
> [`README.md`](../README.md) 新增「界面」段（截图由 `e2e/screenshots.spec.ts` 生成）；
> 本节为证据台账落点。

---

### AO. 2026-09-29 收口 KNOWN_ISSUES #65：前端死代码门禁改用 **TS AST** 判定引用（注释 / 字符串不再算引用）

> **背景**：#65 是 §AN 写文档时实测发现的**门禁漏报口径**——`usageBody()` 用正则剥掉 import 与
> 导出声明后按**裸词**计数，于是「导出零调用，但别处有一句提到它的**注释**」即被判为已引用。
> 真实事故：`useKeydownStack.ts` 的 `isTopKeydown` 唯一调用点早已删除，仅因 `ConfirmDialog.vue`
> 留了一句说明注释，门禁 8 例全绿。#65 当时写明「**未在本轮修的原因**：TS 无等价 AST 依赖，
> 正则剥离注释 / 字符串有误伤风险」——本轮核实该前提**不成立**：`typescript@5.9.3` 已是
> `apps/web` 的**直接 devDependency**，AST 路线可行。

| # | 步骤 | 做法 | 证据 |
|---|---|---|---|
| 1 | **先红** | 先把计数逻辑抽成**纯函数** `findUnreferencedRuntimeExports(sources)`，再补一条**合成源码**口径用例（名字只出现在注释 / 字符串 / 模板字面量文本里 → 必须判死；真实调用与 `${…}` 插值 → 不得误杀） | 未修前该用例 **FAIL**：`expected [] to deeply equal [ …(3) ]`——门禁确实把三种「只被提及、未被引用」的情形全判成「有引用」。与后端 `deadcode_gate_test.go`「每条判定都配一个用合成源码写的口径测试」同做法，不依赖仓库里正好有反例 |
| 2 | **一次失败的尝试（已废弃，记录以免重蹈）** | 先用 `ts.createScanner` 做**词法**剥离：按 token 类型把注释与字符串替换为空格。直觉上最稳（词法不需要语法正确，看似对 `.vue` 也安全） | **实测被证伪**：`i18n/index.ts` 的 `` s.replaceAll(`{${k}}`, String(v)) `` 之后，扫描器把**其后 393 字节**（到文件尾）当成一个模板 token 吞掉——`createScanner` 的裸 `scan()` 循环不会在模板替换后 `reScanTemplateToken`，含 `${…}` 插值的模板字面量因此吃掉后续整段代码。后果是**一次误报 18 个真实导出为死代码**（`setLocale` / `useHealthPoll` / `parsePolicy` …），这正是 #65 警告过的「误伤」 |
| 3 | **回到 AST，并按文件类型分派** | `.ts`：`createSourceFile` + 遍历 AST 收集标识符；`.vue`：**只**把 `<script>` 块交给 AST，`<template>` 原样保留（模板里的组件名 / 表达式是真实引用）、`<style>` 丢弃、HTML 注释剥掉 | 排除三类非引用：`import` 语句（说明符是字符串、绑定名是本地别名）、`export { X }` / `export * from`（导出声明本身）、**被导出声明自身的名字**（否则每个导出自证被引用）。保留类型位置标识符 / 属性名 / 对象键 / `${…}` 插值——沿用旧口径以免误杀 |
| 4 | **变异验证** | 往 `theme.ts` 注入一个「只被注释提到」的导出 `mutationProbeDead` | 门禁**红灯并点名** `./theme.ts → mutationProbeDead`；删除后绿灯。证明新口径确实拦得住旧口径漏掉的那一类 |
| 5 | **口径与残留写明** | 文件头「盲区 3」由「注释/字符串里的同名整词会被算作引用」改为已收口，并注明 `.vue` 模板仍按裸词计数、**同名标识符跨文件抵消仍然存在**（与后端 Gate 1 同向，只会漏报） | 残留不隐藏：跨文件同名抵消是裸词口径的结构性特征，不是本次引入 |

> **未做（有意）**：`.vue` 的 `<template>` 未走 AST（Vue 模板不是 TS 语法，需引入 Vue 编译器才能
> 精确判定）。模板里的裸词计数会保留「模板文本提到某名字即算引用」的宽松度——**只会漏报**，
> 与 §3 之前的口径一致，且模板中出现的名字绝大多数确实是真实引用。

> **文档同步**：[`KNOWN_ISSUES.md`](KNOWN_ISSUES.md) #65 闭环并从 §二 移除、§四 台账改「已闭环」、
> 摘要段改写；[`ROADMAP.md`](ROADMAP.md) §四 前端例数 1126 → **1127** 与覆盖率复测值；
> 本节与 §三 复测段为证据台账落点；[`CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]` 一条。

---

### AP. 2026-09-29 收口 KNOWN_ISSUES #67 / #68：可访问性三处 + nginx 跨层日志关联

> **背景**：两条都是上一批写文档时**实测发现**的真实缺陷（不是文档问题），当时「只记录未修」。
> 本轮一并闭环，四项改动各自配一道回归门禁——**没有一条是靠「改完看一眼」结束的**。

| # | 缺陷 | 修法 | 回归门禁 |
|---|---|---|---|
| 1 | **#67①** `<html lang>` 不随界面语言更新：`index.html` 硬编码 `lang="zh-CN"`，全仓无任何运行时赋值，切到英文后屏幕阅读器仍按中文发音、浏览器翻译提示误判 | `i18n/index.ts` 新增 `applyDocumentLang(loc)`，**模块初始化时同步一次**（让持久化的语言在渲染前就生效），`setLocale()` 内再同步 | `i18n/index.test.ts` 新增 **2 条行为用例**（`setLocale` 与 `cycleLocale` 两条路径都断言 `document.documentElement.lang`）——先红后绿实测 |
| 2 | **#67②** 未处理 `prefers-reduced-motion`：全仓 0 命中，`rise` / `toast-in` / `skel` 关键帧与 `modal-fade` / `pop-in` 过渡在前庭敏感用户声明「减少动效」后**照常播放** | `styles.css` **末尾**加 `@media (prefers-reduced-motion: reduce)`，对 `*` / `*::before` / `*::after` 关闭 `animation` 与 `transition`（`!important` 必要：动效分散在组件级选择器与内联过渡上，逐条提优先级既易漏又难维护） | 新增 `src/a11y_gate.test.ts`（3 例）：断言媒体查询存在**且确实关闭两者** |
| 3 | **#67③** `textarea` 不在统一焦点样式内：`:focus-visible` 列表缺 `textarea`，而表单基础规则写了 `outline: none` ⇒ `BucketPolicyVisualEditor.vue` 的 3 个 `<textarea>` 拿不到与其它控件一致的 2px 轮廓 | 选择器列表纳入 `textarea:focus-visible` | 同一个 `a11y_gate.test.ts`：解析 `:focus-visible` 规则块，断言选择器含 `textarea` 且声明含 `outline: 2px` |
| 4 | **#68** nginx `log_format main` 不含请求 ID ⇒「用户报错 → 查 nginx 日志 → 定位后端 `req`」在反向代理层断开 | `deploy/nginx/{nginx,nginx.docker}.conf` 补 **两个**字段：`rid=$http_x_request_id`（客户端请求值）与 `req=$upstream_http_x_request_id`（**后端回显值**，与后端访问日志 `req` 同源） | `repo_infra_gate_test.go` 新增 `TestNginxAccessLogCarriesRequestID`；变异验证：删掉字段 → 红灯点名两份配置，还原 → 绿灯 |

> **#68 为什么记两个变量（这是本轮唯一的判断点）**：只记 `$http_x_request_id` 会在
> **客户端没传**时记成空——而那正是最常见的情形（浏览器不会自己加这个头）；只记
> `$upstream_http_x_request_id` 则会在客户端传了 ID 时丢掉用户手里那个原始值（用户从报错提示
> 抄下来的通常是它）。二者互补，缺一即该场景下链路断裂。

> **连带的既有 workaround 清理**：#67② 修好后，`apps/web/e2e/screenshots.spec.ts` 里那个
> **`waitForTimeout(500)` 硬等**失去了存在理由——它当时的注释白纸黑字写着「本应用**未处理**
> `prefers-reduced-motion`，故 `emulateMedia` 压不掉它」，而它本身就是个 flaky 来源
> （机器慢就截到按钮半透明、文字重叠）。现改为 `test.use({ reducedMotion: 'reduce' })` 并删掉
> 该等待：截图拿到的是稳定终态，用例耗时降到 **881 ms**（实测 `2 passed / 3.0s`）。
> 这同时让该 spec 顺带在**真实浏览器**里验证了媒体查询确实生效——vitest 的 happy-dom 不做
> 级联，`a11y_gate.test.ts` 只能验源码形态，两者互补。

> **顺带更正一处 HEAD 就存在的现状失真**：`ROADMAP.md` §四 的「E2E（mock 版）」行，其
> **当前状态**列填的是「全 action SHA 经 GitHub API 核验（5 个 SHA 实测 200）」——那是
> action pin 校验的结论，`TestWorkflowActionsAreShaPinned` 的职责，**与 E2E 通过数无关**。
> 该行现改为实跑的 **17 passed / 0 skipped**（不是 15：§AN 新增的 `screenshots.spec.ts` 2 例
> 未同步进任何一处通过数）。同时修正 `FEATURES.md` §三 的同一数字。

> **写测试时实测到的两处工具行为（记录以免重蹈）**：
> ① **`?raw` 读不到 CSS**——vitest 默认 `css: false` 会把 CSS 模块替换为空模块；
> `import.meta.glob('./*.css', { query: '?raw' })` 能匹配到 `./styles.css` 这个**键**，但**取值长度为 0**，
> 静态 `import css from './styles.css?raw'` 同样得到空串。故 `a11y_gate.test.ts` 改用 `node:fs`
> （打开 `css: true` 会改变**全部**用例的 CSS 处理方式，代价不成比例）。
> ② **vitest 下 `import.meta.url` 是 `http://` 形式**，`fileURLToPath` 直接抛
> 「URL must be of scheme file」（`e2e/screenshots.spec.ts` 能用是因为那是 Playwright 的真 Node ESM）。
> 两处都由「门禁前置成立」这条防空跑断言先拦住，**没有静默变绿**。

> **文档同步**：[`accessibility.md`](accessibility.md)（§2.4 焦点可见性改写、§4 第 5/6/12 条改标注为
> 已修复并保留原编号、§5.3 / §5.4 勾除已完成项）、[`OPERATIONS.md`](OPERATIONS.md) §3 日志关联段重写、
> [`KNOWN_ISSUES.md`](KNOWN_ISSUES.md) #67 / #68 闭环（§二 移除 + §四 台账 + 摘要段）、
> [`ROADMAP.md`](ROADMAP.md) §四 门禁基线复测值、[`CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]` 一条。

---

### AQ. 2026-09-29 供应链收口（桌面 SBOM + cosign）+ 告警规则落地 + 账号库 Schema + #66 闭环

> **来源**：逐层盘点后按指令收口五类剩余工作——P0 供应链、P1 运维、P1 契约、
> 「只登记不实施」、以及「未闭环但可处理」。**五项各配一道机械门禁，且全部做过变异验证。**

| # | 桶 | 事项 | 处置 | 门禁与证据 |
|---|---|---|---|---|
| 1 | **P0 供应链** | 桌面安装包**没有 SBOM**（此前只有容器镜像有）——「这个安装包装了哪些第三方依赖」无从查起 | `release-desktop.yml` 新增 `sbom-desktop` job：把 `Cargo.lock` + `pnpm-lock.yaml` 放进同一目录，用**已 pin 的 Trivy**（tag + digest）一次扫出合并 CycloneDX，产出 `sbom-desktop.cdx.json`，做**组件数自检**后 `attest-build-provenance` 并上传 Release。锁文件与平台无关，故**单 job** 即可、不必三平台各跑 | `TestReleaseWorkflowProducesDesktopSBOM`。**本地 docker 实测**：Cargo.lock → **429** 个 crate、pnpm-lock → **24** 个运行期包（Trivy 的 pnpm 解析器只取运行期图、自动排除 devDependencies——正是「随包分发」的口径）。变异：删组件数自检 → 红灯；删上传命令 → 红灯 |
| 2 | **P0 供应链** | **cosign 未接入**，签名核验被绑在 `gh` / `docker buildx` 上，策略引擎（Kyverno / Ratify）与镜像准入无法验 | 镜像在 `publish` job 内按 **digest** 无密钥签名（keyless：OIDC 短期令牌 + Rekor 透明日志，无长期私钥）；桌面侧对 `SHA256SUMS.txt` 做 `cosign sign-blob`——**签清单而非逐个安装包**，一次签名传递性覆盖三平台全部产物，Release 只多 `.sig` / `.pem` 两个文件 | `TestCosignSigningIsWiredAndConsistentlyPinned`。安装器 SHA **三步核验**（`refs/tags` → 解引用 annotated tag → `GET /commits/<sha>` 200）：`v4.1.2` → `6f9f1778…`。变异：两处 pin 不同 SHA → 红灯；镜像改成按 tag 签 → 红灯 |
| 3 | **P1 运维** | OPERATIONS.md §4 有 SLO 与告警**表格**，但自述「仓库未提供告警规则文件」——落地要靠运维手抄表达式，而**抄错一个字母 Prometheus 不报错，告警只是永不触发** | 新增 [`deploy/prometheus/s3client.rules.yml`](../deploy/prometheus/s3client.rules.yml)：3 条 recording rule（把 §4.1 的 SLI 固化，告警与仪表盘共用同一表达式）+ 9 条 alerting rule（与 §4.2 逐行对应），挂进 `rule_files` 即用 | `TestPrometheusRulesReferenceRealMetrics`：① 规则引用的每个 `s3c_*` 必须存在于**发射点**（正则要求引号紧邻指标名，故只在注释里出现的名字不算）；② 每个 `code` 取值必须在 `s3wrap` 白名单内（白名单外的码会被折叠成 `other`，表达式即永不命中）。变异：指标名打错一个字母 → 红灯点名；`SlowDown` 写成 `Slowdown` → 红灯。该文件与 OPERATIONS §4 **必须同改**，已写进两处 |
| 4 | **P1 契约** | 账号库 `accounts.json` 的格式只有散文描述，**无机器可读 Schema**——外部工具无法校验 / 生成 | 新增 [`docs/api/accounts.schema.json`](api/accounts.schema.json)（JSON Schema 2020-12）：`Array<Account>`、12 个字段**全 required**（`model.Account` 无 `omitempty`，写出的文件里字段恒存在）、`id` 标 `uuid`、时间标 `date-time`；描述里写明「加密后是 S3C3 二进制信封，本 schema 不适用」与「明文驱动的 `secretKey` 即明文」 | `TestAccountStoreSchemaMatchesModel`：真值取自**反射** `reflect.TypeOf(model.Account{})`（不是再抄一份字段表），断言**字段集双向相等** + 每字段 JSON 类型 + required 覆盖全字段。变异三个方向全中：删字段 / 改类型 / 加多余字段 |
| 5 | **未闭环** | **#66**（GitLab 侧缺 SAST）登记为开放，理由是「另一套工具链 + 无法在 `make gcl` 本地验证」 | **实测推翻该理由**，且**闭环依据是「真跑通」而非「能列出来」**：`.gitlab-ci.yml` 加 `include: template: Jobs/SAST.gitlab-ci.yml`（**Free 档即可用**）+ 把 `test` 补进自定义 `stages`（模板的 job 是 `stage: test`，不补会直接配置报错）；`DEVELOPMENT.md` §3 补对照表 / 触发事件表 | **实跑证据**：`make gcl GCL_JOBS=semgrep-sast` → `finished in 23 s` → `exported artifacts` → **`PASS` / `EXIT=0`** + `gl-sast-report.json`。⚠️ **更正一处错误结论**：本条此前写「实现早已落地、只是台账没关」——**不成立**，`include:` 是本轮 18:02 才加的（本轮开始时该文件既无 `include:` 也无 `test` stage）；`make gcl-list` 只证明「模板能解析、job 排得进流水线」，**不**证明镜像拉得下来、job 跑得起来、报告产得出来——**实跑立刻暴露了配置缺陷**：默认 `SAST_EXCLUDED_PATHS`（`spec, test, tests, tmp`）是**目录名**式的，与本仓库命名约定不匹配，首跑 66 项里 50 项在测试代码、6 项在**生成的** `apps/web/coverage/` 第三方 JS；补齐后 **66 → 10 项**，全部落在生产源码，逐条 triage 记入 [`threat-model.md`](threat-model.md) §7 |
| 6 | **门禁转绿** | `make check` 稳定失败：`App.corruptStorage` 首个用例在「覆盖率插桩 + 全量并发」下击穿 5s 默认超时（`pnpm test` 与单跑该文件均绿，属**假红**） | 先判定「慢 vs 卡死」：同一次全量跑加 `--testTimeout=20000` → **74 文件 / 1132 例全绿**、该文件总耗时 **2612ms**，故是资源竞争。只给**该文件**放宽到 `{ timeout: 20_000 }`（4× 余量），**不动全局 `testTimeout`** | 断言一行未改（顶栏存在 + 侧栏有导航按钮 + Server 入口可达）。理由写在用例上方注释里：该文件每个用例都 `vi.resetModules()` 后重新 import 整张 App 模块图，是全仓最重的单测文件 |

> **一处自我修正（与「不留尾巴」直接相关）**：第 1 项的门禁**首版是弱门禁**——它只断言
> `gh release upload` 字符串存在，而该 workflow 里另有两处同名字符串（publish 的
> `SHA256SUMS-<bundle>.txt`、aggregate 的 `SHA256SUMS.txt`），于是**把 SBOM 上传整段删掉仍全绿**
> （变异验证时实测暴露）。已收紧为断言**带 SBOM 文件名的那条完整命令**并复验通过。
> 这正是「门禁必须变异验证」的价值：不做变异，这条门禁会一直以「已覆盖」的姿态存在。

> **文档同步**：[`ROADMAP.md`](ROADMAP.md) §三 #12 由 ⏳ 改 **✅**（并写明 keyless 签名不解决
> 发布者信誉、那仍属 E6）；[`OPERATIONS.md`](OPERATIONS.md) §4 指向规则文件并删掉「未提供规则文件」
> 的旧表述；[`DEVELOPMENT.md`](DEVELOPMENT.md) §3（门禁落点 +5、CI 对照表 `semgrep-sast` 行、
> 触发事件表、删掉已作废的「未镜像理由」段）；[`KNOWN_ISSUES.md`](KNOWN_ISSUES.md) #66 闭环并移除；
> [`CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]` 一条。

---

### AR. 2026-09-29 补齐运维事故复盘模板（10 层文档基线第 6 层缺口收口）

> **诊断**：10 层文档覆盖矩阵盘点（§AN）时，第 6 层「运维可靠」唯一未收口的缺口是**事故复盘模板**——
> [`OPERATIONS.md`](OPERATIONS.md) §9.3 只定义了「如何记录」的六步，没有可复制的记录骨架；
> 全仓检索 `复盘` / `postmortem` 命中 **0**。

| # | 缺口 | 处置 | 证据 |
|---|---|---|---|
| 1 | 无复盘模板 | 新增 [`POSTMORTEM_TEMPLATE.md`](POSTMORTEM_TEMPLATE.md)：11 节空模板 | 元信息（P0–P3 定级对齐 §9.1）/ 影响面（含 `interrupted` 半成品）/ 带 `X-Request-ID` 的时间线 / **取证清单**（逐条对齐 §9.3 与 §3.3 的检索命令）/ 根因（触发因素 vs 根因）/ 检测与响应评估（对照 §4.1、§7.1 建议值并显式标注未强制）/ 恢复验证（逐条对齐 §6.4 六步）/ 行动项（强制 owner + 截止 + 跟踪位置）/ §4 文档同步表 / §11「刻意不定义的事」 |
| 2 | 存档路径未定 | **不新造目录**：填写完成的复盘 `git mv` 进 `docs/archive/`，命名 `incident-YYYYMMDD-<短名>.md`，并在 [`archive/index.md`](archive/index.md) 登记一行 | 复用既有归档纪律（§4「归档」条 + `archive/index.md`「什么该归档」新增该类），避免为一次性需求新增目录与命名规则 |
| 3 | 边界易越界 | 明确「安全漏洞报告走 SECURITY.md 私有渠道，不进公开复盘」「值班轮换 / SLA / 对外披露由部署方自定」 | 对齐 §9.2、§9.4 既有口径；**不为模板发明**仓库里没有的规则——「不用于绩效追责」「5 个工作日」等一律标注（**建议**） |
| 4 | 模板不得冒充台账 | 头部如实写明**尚无已填写的复盘实例** | 沿用本仓惯例：没有的东西不假装有（同 [`accessibility.md`](accessibility.md) 的自陈「未做」、[`../.github/SUPPORT.md`](../.github/SUPPORT.md) 的「不提供什么」） |

> **文档同步**：命名约定两处（[`AGENTS.md`](../AGENTS.md) + [`DEVELOPMENT.md`](DEVELOPMENT.md) §4）、
> [`OPERATIONS.md`](OPERATIONS.md) §9.3 与 §11、[`README.md`](../README.md) 文档段、[`llms.txt`](../llms.txt)、
> [`archive/index.md`](archive/index.md)、[`CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]` 一条；
> 本节为证据台账落点。

---

### AS. 2026-09-29 文档失真收口（11 处「文档与实现 / 自身不一致」+ Dependabot 路径失效）

> **诊断**：10 层文档基线盘点时逐条**回读源码复核**（不是只读文档），发现一批会直接误导读者的失真；
> 其中 1 处（Dependabot 路径）是**已入库、现网生效**的功能性缺陷——依赖更新静默失效。

| # | 位置 | 失真 | 核实依据 | 处置 |
|---|---|---|---|---|
| 1 | `.github/dependabot.yml` | 目录写成 `/server`、`/web`、`/desktop/src-tauri` | 三者均不存在，实际为 `apps/server`、`apps/web`、`apps/desktop/src-tauri` | 改为 `apps/...` + 防回归注释 |
| 2 | `.github/SUPPORT.md` | 「本仓库当前没有独立的用户手册 / FAQ」 | [`user-guide.md`](user-guide.md) 存在（654 行，含 §12 FAQ / §13 排障），且被 README / `llms.txt` / [`DEVELOPMENT.md`](DEVELOPMENT.md) 引用 | 改为指向用户手册 |
| 3 | `.github/SECURITY.md` | 硬编码「当前版本 `v1.0.0`」 | `scripts/release-version.sh` 的同步目标里**没有该文件** → 发版必然漂移 | 改为指向 `Makefile` `VERSION` / `/api/health`（SSOT），支持窗口表不 pin 版本 |
| 4 | [`architecture.md`](architecture.md) | 引用 `store/atomic.go` | 该文件不存在；实为 `internal/atomicfile/atomicfile.go`，调用点 `store/filestore.go:197`、`service/job_persist.go:128` | 修正路径与调用点 |
| 5 | [`architecture.md`](architecture.md) | 预签名「过期钳制 [1h, 24h]」 | `handler/objects.go` 仅 ≤0 默认 1h、>24h 钳 24h，**无 1h 下限**；[`threat-model.md`](threat-model.md) §1 自述「无 1h 下限」 | 按实现改写 |
| 6 | [`architecture.md`](architecture.md) | `endpoints.ts`「~70 个方法」 | 实测 `s3api` 顶层方法 **59** 个；70 是后端 `mux.HandleFunc` 数（量纲不同） | 改为 59 并写明量纲差异 |
| 7 | [`architecture.md`](architecture.md) | 配置指针指向 `README.md#配置服务端` | README 已声明 [`CONFIGURATION.md`](CONFIGURATION.md) 为 SSOT | 改指 SSOT |
| 8 | `api/accounts.schema.json` | 「11 个字段恒存在」 | `required` 实为 **12** 个（python 解析核对） | 改为 12 |
| 9 | [`errors.md`](errors.md) | 「`apps/web/src/errors.ts`（若存在）」 | 文件确实存在 | 去掉陈旧对冲 |
| 10 | [`../README.md`](../README.md) | 归档清单只有 3 项 | `docs/archive/` 实有 4 份归档；缺 `code-review-2026-09-24`、`code-review-summary` | 补齐 2 项 |
| 11 | [`threat-model.md`](threat-model.md) | 页首「最后更新 2026-09-28」 | 同文 §7 已记 2026-09-29 引入 SAST | 更新为 09-29 |

> **只登记未修**：CHANGELOG 与 git tag 断裂（3 个时间戳 tag 无版本段、`[1.0.0]` 段日期与 tag 不一致）——
> 补段等于**重建历史发布记录**，需人类确认口径，故登记 [`KNOWN_ISSUES.md`](KNOWN_ISSUES.md) **#69**，本轮**不擅自回写历史**。
> 判据：本仓对「历史结论」的纪律是**不回写**（见 [`archive/index.md`](archive/index.md)）。

> **文档同步**：[`CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]` 一条、
> [`KNOWN_ISSUES.md`](KNOWN_ISSUES.md) #69（§二 / §四 / 页首口径 / 归零口径四处）、本节为证据台账落点。

---

### AT. 2026-09-29 文档缺口收口（链接/锚点门禁 + 文档登记表 + ADR 模板 + 子树 AGENTS + 隐私声明）

> **诊断**：§AS 修的是「文档说错了」，本节补的是「**没人机械保证文档不会说错**」。此前链接悬空 /
> 文档无 owner / ADR 索引与真实文件不一致 / 子树无规则文件 / 无隐私声明 / DR 演练无记录格式，
> 全靠人工纪律——本次把它们变成**文件 + 门禁**。

| # | 缺口 | 处置 | 证据 |
|---|---|---|---|
| 1 | 相对链接与页内锚点**无机械校验**（重命名文档后死链只在 GitHub 上可见） | 新增 [`doc_link_gate_test.go`](../apps/server/doc_link_gate_test.go) | 扫描面自检阈值（链接 ≥1000 / 锚点 ≥50，命中不足即红灯，防「全绿但失明」）；**变异验证**：改 `POSTMORTEM_TEMPLATE.md` 的链接 → 红灯点名 `文件:行号`；改 `archive/review-2026-09-19.md` 的锚点（等长替换保行号）→ 锚点红灯；两次以 sha256 校验还原。现测 **1231 条相对链接 + 73 条带锚点链接，0 失效**；盲区（外链可达性 / 大小写冲突 / 引用式链接 / slug 无 NFKC）写在文件头 |
| 2 | 文档**无 owner / 复审周期** | [`DEVELOPMENT.md`](DEVELOPMENT.md) §4 新增文档登记表 | 11 类文档 × owner（单人维护 `@weilai1949`）× 复审周期**建议值**；「最后复审」一律填「登记时基线（2026-09-29）」，**不追溯编造**历史日期 |
| 3 | ADR **无模板**；索引与真实文件不一致 | 新增 [`decisions/0000-template.md`](decisions/0000-template.md)；重写 [`decisions/index.md`](decisions/index.md) 索引表 | 声明真实格式（英文 H2 + 中文正文）；补**日期列**与「最新更新」列；登记 ADR-003 / ADR-004 篇内的 `## Update（2026-09-19）`（原索引**漏登 0004**） |
| 4 | 子树**无 `AGENTS.md`**（根文件自称子树规则应放子树） | 新增 `apps/server`、`apps/web`、`apps/desktop` 三份子树 `AGENTS.md` | 只写子树专属硬约束（Go 分层 / 门禁入口与覆盖率口径；Web 零运行时依赖 + 覆盖率排除项；Desktop 无 IPC + toolchain pin），仓库级规则仍只在根文件——**不复制、防分叉** |
| 5 | **无隐私 / 遥测声明** | [`user-guide.md`](user-guide.md) 新增「十四、数据与隐私」 | 实测：全仓生产代码检索 sentry / posthog / analytics / telemetry 等**零命中**，web 生产依赖仅 `vue`；凭据存放（`secretKey` 仅服务端、响应脱敏 `secretSet`；`S3C_TOKEN` 默认 `sessionStorage`）；日志不含密钥 |
| 6 | DR 演练**无记录格式** | [`POSTMORTEM_TEMPLATE.md`](POSTMORTEM_TEMPLATE.md) 新增 §7.1 | 复用同一模板（**不新增文件**）：场景 / 触发方式 / 实测 RTO·RPO 对照 §7.1 **建议值** / 未覆盖路径 / 密钥可用性；全文标注「建议值、未强制、不是 SLA」 |
| 7 | `.env.example` 漏登 3 个可选项 | `apps/server/.env.example` 补 `S3C_EXPOSE_METRICS` / `S3C_EXPOSE_OPENAPI` / `S3C_CSP_CONNECT_SRC` | [`CONFIGURATION.md`](CONFIGURATION.md) 写明两份 `.env.example` 的**分工口径**（根 = compose 透传项；server = 服务端全量可选项），消除「同名两份、内容各半」的双源；根文件保持只列 compose 透传项 |
| 8 | §AS 的 Dependabot 修复**无回归门禁** | 新增 [`dependabot_gate_test.go`](../apps/server/dependabot_gate_test.go) | 断言每条规则的 directory 存在且含对应清单（gomod→`go.mod` / npm→`package.json` / cargo→`Cargo.toml`），未登记的 ecosystem 红灯；**变异验证**：把 `/apps/server` 改回 `/server` → 红灯点名 |

> **文档同步**：[`CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]` 一条、[`DEVELOPMENT.md`](DEVELOPMENT.md) §4 文档登记表
> （新增文件已登记）；本节为证据台账落点。

---

### AU. 2026-09-29 接口契约表达鉴权（文档级 `security` + 逐端点豁免 + `tags` 分组）

> **诊断**：`api/openapi.json` 早已定义 `components.securitySchemes.bearerAuth`，但**全局 `security` 为空、
> 0/70 个 operation 声明 `security`、`tags` 为空**——代码生成器与 AI 代理从机器可读契约里读不出
> 「哪些端点要 Bearer」，而 [`api.md`](api.md) 明确要求。属「契约存在但**鉴权意图不可判定**」。

| # | 处置 | 证据 |
|---|---|---|
| 1 | `internal/openapi`：新增 `Tag` 与 `Registry.AddTag`；`buildSpec` 输出文档级 `security: [{bearerAuth: []}]`；`Op` 新增 `NoAuth` → 渲染为逐 operation 的 `security: []`（覆盖文档级默认） | 改动集中在 `internal/openapi/openapi.go`、`handler/openapi_register{,_system}.go` |
| 2 | 声明 **10 个 tag**（accounts / buckets / bucket-settings / objects / object-meta / multipart / versions / trash / migrate / system），与 [`api.md`](api.md) 章节对应 | 断言「无孤儿声明、无未声明引用」 |
| 3 | `/api/health`、`/api/metrics` 标 `NoAuth`；`/api/openapi.json` **不豁免** | 真值来源 `handler/middleware.go` 的 `withAuth` 豁免名单，不照抄文档 |
| 4 | 新增 `handler/openapi_auth_test.go`（**先红后绿**） | 红灯原文：文档级 security 缺失 / 豁免端点未显式 `security: []` / 顶层 tags 为空；实现后 3 例全绿，并跑通 `TestCommittedOpenAPISpecMatchesRuntime` 等既有契约门禁 |
| 5 | 重新生成 [`api/openapi.json`](api/openapi.json) | python3 解析核对：**68 op 继承文档级 security + 2 op 显式豁免 = 70/70 可判定**；`tags` 10 组、70/70 op 至少归 1 组 |

> ⚠️ **需人类复核**：按 [`AI_POLICY.md`](AI_POLICY.md) §3，修改公共 API 契约（`openapi_register_*.go`）属「**需确认**」，
> 本轮只准备改动与证据，**未推送、未合并**。
> **2026-09-30 事实注记（§BI）**：该批改动已随后续提交（`git log -S AddTag` → `a984df7`）进入
> `develop` 且当前与 `origin/develop` 一致——上句「未推送、未合并」是落笔时点状态，按「历史值不改写、
> 只加注」纪律保留原文；「需确认」是否已由人类履行属流程记录，本表不代答。

> **文档同步**：[`api.md`](api.md)、[`CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]` 一条；本节为证据台账落点。

---

### AV. 2026-09-29 元信息 / 导航收口（docs 落地页 + 导航覆盖门禁 + `.gitattributes`）

> **诊断**：第 2 层「元信息 / 导航」此前为部分达标——台账（CHANGELOG / ROADMAP / KNOWN_ISSUES / FEATURES）
> 与 AI 入口（`AGENTS.md` / `llms.txt`）齐全，但**缺 `docs/` 落地页**，且 4 个导航面之间**无一致性保证**：
> `doc_link_gate` 只能证明「已有链接不悬空」，证明不了「新文档被登记进导航」。

| # | 缺口 | 处置 | 证据 |
|---|---|---|---|
| 1 | `docs/` 无落地页（浏览 30 个文件没有入口） | 新增 [`README.md`](README.md)：人类导航 SSOT，按「我要做什么」分五组 + 机器可读面 + 文档维护规则 | 每篇文档一句话说明；指向各 SSOT（`CONFIGURATION` / `api.md` / `KNOWN_ISSUES` / `ROADMAP`）与各文档门禁的落点 |
| 2 | 导航面靠人工同步，**已漂移过**（README 归档清单漏 2 项、AGENTS 入口表落后、DEVELOPMENT 清单漏 `SUPPORT.md`） | 新增 [`doc_index_gate_test.go`](../apps/server/doc_index_gate_test.go)：`docs/*.md` 必须登记进落地页；子目录文档必须出现在落地页或该子目录 `index.md` | **TDD 先红**（落地页不存在 → 红灯）；**变异验证**：新建 `docs/_nav_probe.md` → 红灯点名「应登记进 docs/README.md」→ 删除后绿灯；含扫描面自检阈值（≥25 篇） |
| 3 | **无 `.gitattributes`**：行尾取决于各人 `core.autocrlf` | 新增 [`.gitattributes`](../.gitattributes)：`* text=auto eol=lf` + Windows 脚本（`.bat`/`.cmd`/`.ps1`）CRLF + 图片二进制 | 入库核查 **538 个跟踪文件中 0 个 CRLF 文本文件**，故不触发批量重新规范化；避免 Windows 检出整文件变 CRLF 或脚本因 `\r` 执行失败 |
| 4 | 四个导航面各写各的 | 同步：根 [`README.md`](../README.md) 文档段顶部加落地页指针、[`llms.txt`](../llms.txt) 补一条、根 [`AGENTS.md`](../AGENTS.md) 入口表加一行、[`DEVELOPMENT.md`](DEVELOPMENT.md) §4 命名约定与「文档登记表」 | 链接门禁实测扫描面 **1335 条相对链接 / 73 条锚点，0 失效**（写入时实测） |

> **未纳入本层**：CHANGELOG 与 git tag 的对应关系属「发布元信息」缺口，已登记 [`KNOWN_ISSUES.md`](KNOWN_ISSUES.md) **#69**
> （需人类确认口径，不在本轮回写历史）。

> **文档同步**：[`CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]` 一条、[`DEVELOPMENT.md`](DEVELOPMENT.md) §4 登记表；
> 本节为证据台账落点。

---

### AW. 2026-09-29 安全与供应链收口（自动生成的许可证清单 + 依赖覆盖门禁 + 产物核验指南）

> **诊断**：第 7 层「安全与供应链」的自动审计（Trivy / govulncheck / cargo audit / CodeQL / SAST）、
> SBOM、签名与 provenance 都已就位，唯一硬缺口是**仓库内没有第三方依赖与许可证清单**——
> 而清单属事实，手写必然漂移。故做成「**脚本生成 + 门禁钉住枚举完整性**」。

| # | 缺口 | 处置 | 证据 |
|---|---|---|---|
| 1 | 无依赖 / 许可证清单 | 新增 [`../scripts/gen-third-party-licenses.sh`](../scripts/gen-third-party-licenses.sh) + 生成物 [`THIRD_PARTY_LICENSES.md`](THIRD_PARTY_LICENSES.md) | 三处权威来源：Go `go list -m -json all` + 模块内 `LICENSE` 文本机械识别；Rust `cargo metadata` 的 `license` 字段（离线兜底 `Cargo.lock` ∩ registry 源码）；npm `apps/web/package.json`。识别不出**不猜** → `UNKNOWN` + 文件路径；补全用 `go mod download` 且带 **90s 超时**（默认 GOPROXY 不可达时不挂死） |
| 2 | 生成物会过期 | 新增 [`../apps/server/third_party_licenses_gate_test.go`](../apps/server/third_party_licenses_gate_test.go) | 依赖图 / `Cargo.lock` / `package.json` 里**每个包**都必须出现在清单里（含扫描面自检阈值）；**变异验证**：删掉清单里的 `golang.org/x/sys` 与 `serde` 两行 → 红灯逐条点名，重新生成 → 绿灯 |
| 3 | 两侧口径各有 bug | 由门禁**首次运行**抓出并修正 | ① 生成器把「`Dir` 为空的未下载模块」当非依赖丢弃 → 漏掉 x/net、x/term、x/text；② 门禁把本地 path crate `s3client` 当第三方；③ Cargo.lock 分块正则隔块漏读（429 → 214）——Go RE2 **不支持前瞻断言**，改用 `strings.Split` |
| 4 | 消费者不知道**怎么验**产物 | [`threat-model.md`](threat-model.md) §5 扩为四小节 | **5.1** 锁定策略（Go / npm / Rust / Actions SHA / 镜像 digest 与各自门禁）· **5.2** 依赖与许可证清单 · **5.3**「消费者如何验证产物」· **5.4** 已知缺口 |
| 5 | 核验命令必须真实 | 命令**逐字取自工作流注释**，不凭记忆写 | `gh attestation verify oci://…`、`docker buildx imagetools inspect <ref> --format '{{json .Provenance}}'`、`cosign verify <ref>@<digest>`、`sha256sum -c SHA256SUMS.txt`、`cosign verify-blob`（含 `--certificate` / `--signature` / `--certificate-identity-regexp`） |
| 6 | 合规口径不越界 | 清单顶部声明「机械识别、**不构成法律意见**」；copyleft **关键词**命中单列供人工阅读 | 实测：Go **43** 模块全部宽松（BSD-3-Clause 21 / Apache-2.0 19 / MIT 3）、npm 仅 `vue`、Rust 428 crates 中 `MPL-2.0` 5 个与含 `LGPL-2.1-or-later` 的表达式 2 个；`UNKNOWN` **0** 项 |

> **文档同步**：[`CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]` 一条、[`AGENTS.md`](../AGENTS.md) 入口表与命名约定、
> [`llms.txt`](../llms.txt)、[`README.md`](../README.md) 运维与安全段、本页导航 [`README.md`](README.md)、
> [`DEVELOPMENT.md`](DEVELOPMENT.md) §4 同步表 / 登记表 / 命名约定；本节为证据台账落点。

---

### AX. 2026-09-29 AI 时代层收口（AI 治理的机械保证 + Copilot 指针入口）

> **诊断**：第 10 层此前「**资产齐全但不可执行**」——根/子树 `AGENTS.md`、`llms.txt`、`AI_POLICY.md`、
> PR 披露块、机器可读契约与门禁族都在，但这些 AI 治理资产**本身没有任何门禁**。而这一层恰好最容易被
> 静默破坏：注入截断、子树漏建、政策声明失真。

| # | 缺口 | 处置 | 证据 |
|---|---|---|---|
| 1 | 根 `AGENTS.md` 会被**整体注入**、超长即被截断，硬约束静默失效 | 门禁断言 ≤ **10 KiB**（实测 7995 B）且必须指向规范正文 | `TestRootAgentsMdStaysWithinInjectionBudget`；**变异**：灌到 20 KB → 点名「超出注入预算」 |
| 2 | 新增 `apps/` 子树漏写 `AGENTS.md` → 子树规则对 agent 不可见 | 门禁逐个断言 `apps/*` 存在子树文件、≤ 4 KiB、**含 `../../AGENTS.md` 回指** | `TestEveryAppSubtreeHasAgentsMd`；**变异**：新建 `apps/probe/` → 点名 |
| 3 | `AI_POLICY.md` 的**事实声明**会随仓库演进失真 | 门禁把声明与现状对上（「未提交 MCP 配置」↔ 根无 `.mcp.json`；PR 模板含披露块） | `TestAiPolicyClaimsMatchRepoState`；**变异**：改掉披露块措辞 → 点名 |
| 4 | 其它 AI 工具入口可能变成「第二事实源」 | 新增 [`.github/copilot-instructions.md`](../.github/copilot-instructions.md) **纯指针**（Copilot 唯一入口），门禁断言 ≤ 2 KiB 且必须指向 `AGENTS.md` | **变异**：灌到 7 KB → 点名「超出指针预算」 |
| 5 | 政策与门禁的对应关系无人知晓 | [`AI_POLICY.md`](AI_POLICY.md) 新增 **§11**：可机检条款 ↔ 门禁一一对应，并**如实列出没有机械保证的部分**（权限矩阵「需确认 / 禁止」档、发布授权、不自动合并、不得读密钥 → 人工约束） | 同节另写明刻意不引入的文件：不加 `CLAUDE.md`、不加 `llms-full.txt`、不提交 `.mcp.json`（各附理由） |

> **文档同步**：[`CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]` 一条、[`DEVELOPMENT.md`](DEVELOPMENT.md) §4.1 / §4 位置与命名 / 登记表、
> [`llms.txt`](../llms.txt)、[`README.md`](README.md) 文档维护表；本节为证据台账落点。

---

### AY. 2026-09-30 客户端支持矩阵收口（10 层基线第 6 层「用户文档」唯一缺口）

> **诊断**：10 层文档基线盘点（§AN）时，第 6 层「用户文档」唯一未收口的是**没有客户端支持矩阵**——
> 「哪些浏览器 / 操作系统能用、哪些只是没测过」无文档可查。

| # | 步骤 | 做法 | 证据 |
|---|---|---|---|
| 1 | 先收集硬证据再动笔 | 回读 vite.config.ts（未覆盖 `build.target` → Vite 6 默认 'modules'）、tsconfig.json（ES2021）、download.ts（File System Access + blob 兜底）、upload.ts（XHR）、storage.ts（存储降级）、styles.css（`:focus-visible` / `prefers-reduced-motion` / 900px 断点）、两套 playwright.config.ts（仅 chromium）、release-desktop.yml（三平台矩阵） | 矩阵每行带文件与行号；版本区间标**建议值（未逐版本实测）** |
| 2 | compatibility.md §6 改「兼容矩阵」+ 新增 §6.2 | **不重编号 §7/§8**（POSTMORTEM_TEMPLATE.md / GOVERNANCE.md 对 §7 / §8 有散文引用，重编号会失真） | user-guide 新链接锚点由 doc_link_gate 校验 |
| 3 | user-guide §十 加一行指针 | 最小改动（+2 行） | 同上 |
| 4 | 门禁实测 | `go test . -count=1` 全绿（含链接 / 锚点 / 导航覆盖三闸） | 0 失效 |

> **文档同步**：[`CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]` 一条；[`docs/README.md`](README.md) 导航行、根 [`README.md`](../README.md)、[`llms.txt`](../llms.txt)、[`DEVELOPMENT.md`](DEVELOPMENT.md) §4 同步表行由 Lead 统一收口；本节为证据台账落点。

---

### AZ. 2026-09-30 ADR 覆盖补足（8 篇新 ADR + 取舍表覆盖门禁）

> **诊断**：architecture.md §7 取舍表仅 4 行有 ADR（均 2026-09-16 回溯），§2 关键机制表中 SSE 异步任务、
> 存储三驱动、预签名直传、有界并发、ZIP 流式、单实例、REST 无版本前缀等 8 项已落地决策无决策记录；
> 且「新增取舍行忘补链接」无任何门禁可见。

| # | 缺口 | 处置 | 证据 |
|---|---|---|---|
| 1 | 8 项已落地决策无 ADR | 新增 ADR-005..012（每条现状回读源码核实并注明路径；WebSocket / 外部 DB / 逐库拒绝理由等无原始记录处显式标「未验证」）；Date 取自 `git log --diff-filter=A` 首次引入日期 | `docs/decisions/0005`..`0012` + index 登记 8 行 |
| 2 | architecture.md §2 / §7 无决策链接 | §2 机制表 8 行补链接、§7 取舍表扩至 12 行且每行带链接 | doc_link_gate 校验全部可达 |
| 3 | 「取舍行忘补 ADR 链接」无门禁 | 新增 `adr_coverage_gate_test.go`：断言 §7 每行含 `docs/decisions/` 链接（纯函数解析 + 5 条合成口径用例） | TDD 先红（8 行无链接逐行点名）后绿；变异：摘 ADR-007 链接 → 红灯点名该行 → 还原绿灯 |

> **文档同步**：[`CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]` 一条；[`docs/README.md`](README.md) / [`llms.txt`](../llms.txt) / [`DEVELOPMENT.md`](DEVELOPMENT.md) §4 行由 Lead 统一收口；本节为证据台账落点。

---

### BA. 2026-09-30 供应链收口（OpenSSF Scorecard + PR 依赖审查）

> **诊断**：第 8 层「安全与供应链」盘点后仍剩两块空白——仓库外部健康度评分（OpenSSF Scorecard，
> 此前只在 ci.yml 注释里被提及）与 PR 时点的依赖 diff 审查（此前 dependabot 只在事后开 PR、
> 无进入 main 前的拦截）。

| # | 缺口 | 处置 | 证据 |
|---|---|---|---|
| 1 | 无 Scorecard workflow | 新增 `.github/workflows/scorecard.yml`：schedule 周六 + `workflow_dispatch`；官方模板四步；顶层 `read-all` + job 级 `security-events: write` / `id-token: write` | SHA `2d1146689b8cda280b9bc96326124645441f03bc`（v2.4.4）API 三重核验；门禁 8 workflow / 56 引用全 SHA |
| 2 | 无 PR 依赖审查 | 新增 `.github/workflows/dependency-review.yml`：每个 PR 跑 dependency-review-action v5（默认 low / runtime / license-check）；权限 `contents: read` | SHA `a1d282b36b6f3519aa1f3fc636f609c47dddb294`（v5.0.0）三重核验；与 Trivy 阈值差异在 threat-model §5.5 写明是有意的 |
| 3 | GitLab 侧无等价 | 不镜像并显式登记：DEVELOPMENT §3 对照表 + threat-model §5.5 | GitLab SAST / Dependency Scanning 模板均非同一检查 |

> **文档同步**：[`threat-model.md`](threat-model.md) §5.5 新增 + §5.4 缺口四条 + §5 开头行；[`CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]` 一条；本节为证据台账落点。

---

### BB. 2026-09-30 AI 产出效果证据层收口（黄金任务集 / 评分卡 / 贡献度量）

> **诊断**：第 11 层「AI 时代」只有过程约束（权限 / 披露 / DoD），没有效果证据——同一份 DoD 可以全绿
> 而输出把契约改坏，仓库没有任何可复现的测量。

| # | 缺口 | 处置 | 证据 |
|---|---|---|---|
| 1 | 无黄金任务集 | `docs/AGENT_EVALS.md`：GT-1..GT-4（新增端点 / store bug / 配置项 / 前端 UI），判据全部映射真实门禁，含红→绿轨迹与失败形态 | GT 判据逐一取自现有 gate 文件名；门禁断言 GT 表 ≥ 3 |
| 2 | 无评分卡 | 加权 40/20/15/15/10，0–4 锚点，硬门槛（门禁全绿或死代码安全任一 0 分直接失败） | 每档锚点绑定可复核证据 |
| 3 | 无贡献度量 | 「占比」披露字段 + 度量台账（基线 2026-09-30，不编造历史；PR 收口回填、发版前对账） | PR 模板 + AI_POLICY §5 字段集由门禁钉住一致 |
| 4 | 无机械评测 | `scripts/agent-eval.sh`（vet/门禁/test/build + 前端三件套，依赖缺失显式 SKIP；JSON + EVAL_RESULT，失败非零退出） | 冒烟实测 vet/build pass、失败阶段回显、exit=1 行为正确 |
| 5 | 结构无人守 | `agent_evals_gate_test.go` 五条不变量 | TDD 先红后绿 + 三条变异验证红灯点名 |

> **文档同步**：[`AI_POLICY.md`](AI_POLICY.md) §5 / §11、PR 模板披露块；[`CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]` 一条；本节为证据台账落点。

---

### BC. 2026-09-30 英文文档入口收口（docs/en/index.md）

> **诊断**：第 13 层「质量合规」的文档面只有中文；根目录只允许 4 个约定文件，英文入口按命名约定
> 落 `docs/en/`（目录小写）。

| # | 缺口 | 处置 | 证据 |
|---|---|---|---|
| 1 | 无英文文档 | 新增 `docs/en/index.md`：根 README 完整英文翻译（概览 / 特性 / 架构 / Quick Start 命令逐字保留 / 文档索引 / 安全 / License），文首声明**中文 SSOT、翻译快照** | doc_link_gate 对 docs/en/ 0 红灯 |
| 2 | 存在两份英文入口（readme.en.md 与 en/） | 收敛为单一入口 `docs/en/index.md`，删除 readme.en.md 并把全部登记重指向 | 链接门禁机械化收口（残留引用即红） |
| 3 | 导航未登记 | docs/README.md / llms.txt / 根 README 语言切换 / DEVELOPMENT §4（同步表 / 命名约定 / 登记表） | 由 Lead 统一收口，doc_index 转绿 |

> **文档同步**：[`CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]` 一条；本节为证据台账落点。

---

### BD. 2026-09-30 导航收口残留（命名约定两处分叉 + `llms.txt` 目录摘要 + README AI 入口 + 机械门禁）

> **诊断**：2026-09-30 文档基线补缺六项（§AY–§BC、§BE）落地后逐面回读比对，三处导航仍有
> 「只差一条登记」的分叉——它们同属「新增文档漏登记命名清单」这一类，此前全靠人工纪律维持，
> 正因没有门禁才漏。

| # | 缺口 | 处置 | 证据 |
|---|---|---|---|
| 1 | `AGENT_EVALS.md` 只进了 [`DEVELOPMENT.md`](DEVELOPMENT.md) §4 命名清单，[`AGENTS.md`](../AGENTS.md) 清单漏登（「同 PR 两处同改防分叉」是硬规则）；AGENTS 小写清单另漏 `en/` 子目录 | AGENTS 命名约定两处补齐（大写 + `AGENT_EVALS.md`、小写 + `en/`） | 与 DEVELOPMENT §4 逐名对齐 |
| 2 | [`llms.txt`](../llms.txt)「目录」段命名摘要落后一代名单（缺 `OPERATIONS` / `PERFORMANCE` / `AGENT_EVALS` / `POSTMORTEM_TEMPLATE` / `THIRD_PARTY_LICENSES` 与 5 个内容文档、`en/` 子目录） | 重写为全量名单（元文档 12 + 内容文档 9 + 子目录 4；稍后落地的 `data-model.md` 见 §BF 第 4 条），与 AGENTS / DEVELOPMENT §4 同口径 | 三处命名口径逐名一致 |
| 3 | 根 [`README.md`](../README.md)「贡献与治理」缺 `AGENT_EVALS.md` 入口（英文快照 [`en/index.md`](en/index.md)、[`README.md`](README.md) 导航、`llms.txt` 均有） | 补一行「AI 代理评测与贡献度量」 | SSOT 与翻译快照覆盖一致 |
| 4 | 上述分叉**无机械保证**（正因无门禁才漏登） | 新增 [`docs_naming_gate_test.go`](../apps/server/docs_naming_gate_test.go)：`docs/` 顶层 `.md` 与含 `.md` 子目录的名字必须同时登记进三处命名口径（只认反引号登记形态，散文提及不算），含扫描面自检阈值 | 变异复核步骤见文件头：摘 `AGENT_EVALS.md` 登记 → 红灯点名 → 还原绿灯 |

> **文档同步**：[`CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]` 一条；本节为证据台账落点。
> **门禁实跑回填（2026-09-30 16:08 CST，§BH）**：`cd apps/server && go test . -count=1` 全绿；
> 变异复核已实跑——摘掉 `AGENTS.md` 命名清单的 `AGENT_EVALS.md` →
> `TestDocsNamingConventionRegistersEveryDocsFile` 红灯点名「AGENTS.md 命名约定段 缺 AGENT_EVALS.md」→
> 还原后绿灯。

---

### BE. 2026-09-30 状态台账 #69 收口（CHANGELOG ↔ git tag 一致性 + 机械门禁）

> **背景**：`git tag` 有 3 个时间戳 tag 无对应版本段、3 个 09-01 快照段与 0.1.0 / 0.2.0 段无对应 tag、
> `[1.0.0]` 段日期与 tag 不符且顺序非倒序（KNOWN_ISSUES #69，2026-09-29 盘点时发现）。

| # | 步骤 | 做法 | 证据 |
|---|---|---|---|
| 1 | git 取证定性 | 3 个时间戳 tag 均打在 2026-09-02 的 `feat`/`fix` 提交（c33cc07 / a77b878 / 7d8ccec）上、**非** `release:` 提交；同日稍后 rc0（19:10）/ rc1（19:16）两个 `release: prepare` tag 取代 → 定性**内部快照**（#69 处置 (a) 成立） | `git tag -l` + `git log -1 --format='%ci %s' <tag>` 逐一取证 |
| 2 | CHANGELOG 顶部建「tag ↔ 版本段对应关系（唯一台账）」 | 8 行快照登记：5 个「有段无 tag」（09-01 快照段 ×3 + 0.1.0/0.2.0 导入前历史段）+ 3 个「有 tag 无段」快照 tag；正式版本由门禁正向校验不登记 | 表格式机器可解析（首列 tag / 次列段，`无` 表示该侧缺失） |
| 3 | `[1.0.0]` 段修正 | 日期按 tag 事实改 2026-09-22（tag 指向 0cfd4ef），整段移到 Unreleased 之后恢复倒序；**内容未改写**（历史结论不改写纪律） | 迁移前后 diff 只改日期与位置 |
| 4 | 门禁（TDD） | 新增 `changelog_tag_gate_test.go`：tag↔段双向 + Unreleased 居首 + 映射解析口径（合成用例）+ 扫描阈值；tag 从 .git 读文件获取 | 先红（映射表缺失）→ 绿；变异验证：删映射行 / 改段名 → 红灯点名 |
| 5 | 发版脚本硬检查 | `release-version.sh` 收尾 `grep -q "^## \[$DISPLAY\]" CHANGELOG.md` 缺段 exit 1 | `bash -n` 通过；由 TestReleaseScriptHardChecksChangelogSection 钉住 |

> **文档同步**：本节为证据台账落点；CHANGELOG `[Unreleased]` 已由任务 D 自行追加一条「修复」；
> KNOWN_ISSUES #69 闭环；目录索引由 Lead 统一收口。

### BF. 2026-09-30 docs 内待修项收口（文档失真 + 对比度静态核查 + 改进项登记 SSOT）

> **诊断**：按「docs 里记录了待修的也要处理」逐篇清点非冻结文档（`archive/` 冻结件不回写），
> 记录在案的待修项集中在 `accessibility.md` 与 `OPERATIONS.md`：一处文档自相矛盾、一处「只声明
> 缺口无记录」、一批挂在正文里的口头待办。

| # | 缺口 | 处置 | 证据 |
|---|---|---|---|
| 1 | `accessibility.md` §5.1 第 9 条与 §4 第 5 条**自相矛盾**（#67② 已修仍写「动效偏好缺口仍在」） | 第 9 条改为回归项表述（仍播放即回退，先看 `a11y_gate.test.ts`） | 与 §4 第 5 条、`styles.css` 的 `prefers-reduced-motion` 块一致 |
| 2 | §4 第 4 条「没有对比度专项核查记录」 | 新增 §5.5 静态核查记录（11 行 token 配对、WCAG 2.1、深色 `rgba()` 覆层按 alpha 合成）+ `contrast_gate_test.go` 数值门禁 | 对账纠偏 2 处口径失真：`--danger` 5.98 / `--ok` 5.02 系渲染态读数 → token 公式值 **5.91 / 4.95**（`styles.css` 注释同改）；「4 组」计数漏行 → 钉为**共 11 行、浅色 5 / 深色 3 行低于 AA**。修色（`--ok` / `--danger`）由并行 axe 批次完成 |
| 3 | §5.3/§5.4 改进项 + `OPERATIONS.md` 观测缺口在正文当**口头待办** | 迁入 [`ROADMAP.md`](ROADMAP.md) §三 **#17 / #18**（唯一来源），原文档改指针；`ROADMAP` #12 已落地按 §六 第 1 条移出、3.2 口径修正 | 两源分工合规；#12 证据在 §AQ / §AW 与 threat-model §5.4，移出不丢信息 |
| 4 | 并行批次新增 [`data-model.md`](data-model.md)（+ `data_model_gate_test.go`）**四个导航面未登记**（`doc_index_gate` 与命名同步门禁会红） | 补进 [`README.md`](README.md) 导航、`AGENTS.md` / [`DEVELOPMENT.md`](DEVELOPMENT.md) §4 命名清单与登记表、[`llms.txt`](../llms.txt)；「账号存储格式」同步表行挂上该地图 | 四面逐名对齐；**去重核对已完成**（2026-09-30 复核：各登记一次、无重复） |
| 5 | 「改 `styles.css` 后 §5.5 表须重算」只是口头约定（本轮即实测抓到改色未按公式重算） | 新增 [`contrast_gate_test.go`](../apps/server/contrast_gate_test.go)：表格数值 / 计数声明 / `--brand-from` 注释数字 ↔ `styles.css` **机械重算比对**，改色不重算即红灯 | 变异复核步骤见文件头：改 `--ok` 色阶 → 红灯点名该行；把结论句 5 改 4 → 计数红灯 |

> **文档同步**：[`CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]` 一条；本节为证据台账落点。
> **门禁实跑回填（2026-09-30 16:08 CST，§BH）**：`cd apps/server && go test . -count=1` 全绿
> （链接 / 锚点 / 导航 / 命名 / `contrast_gate`）；前端 `pnpm lint` 0 告警、`pnpm test` **74 文件 /
> 1132 例全绿**、`pnpm build` OK（366.72 kB，gzip 112.72 kB）；`contrast_gate` 计数变异复核已实跑——
> 把 §5.5 结论句 `共 **11 行**` 改 10 → `TestContrastRecordCountClaimsMatch` 红灯点名 → 还原绿灯
> （步骤见其文件头）。

---

### BG. 2026-09-30 AI 时代文档补强（六路并行：P0 效果证据 / P1 机器可读 / P2 元信息）

> **诊断**：以「AI 时代成熟仓库文档基线」逐层盘点（承接 §AN 的 10 层矩阵），确认散文文档层已饱和，
> 剩余缺口集中于三类：① **AI 效果证据**只有制度没有可执行任务集与台账；② **机器可读深度**不足
> （OpenAPI 0 示例、无 `security.txt`、无 SLO 仪表盘、性能无回归门禁、无障碍无自动检测）；
> ③ **英文覆盖**只有 README、数据模型无单一地图。本批 6 条并行工作流逐条收口，**每条能力配一道
> 机械门禁并做变异验证**（破坏 → 红灯点名 → 还原 → 绿灯）。

| # | 工作流 | 缺口 | 交付 | 门禁 / 证据 |
|---|---|---|---|---|
| 1 | 接口契约 | `openapi.json` 5 schema / 0 示例、无 requestBody 示例；`api.md` 0 条 curl | 37/37 请求体 operation 有请求示例、70/70 有 2xx 示例、5/5 schema 有 `example`；`api.md` 12 条 curl 覆盖 10 tag；非 JSON 2xx 用真实 media type | `openapi_examples_gate_test.go`（含示例字段 schema 形状校验、curl 覆盖与空扫自检）；5 处变异：删响应示例 / 删请求示例 / 加未声明字段 / 删 migrate curl / 空 paths 全部红灯点名 |
| 2 | 运维与性能 | OPERATIONS 自述「未提供仪表盘」；有基线无回归门禁；密钥轮换无 runbook | `deploy/grafana/s3client.dashboard.json`（31 面板）；`bench_budget_test.go` + `perf.yml` + `make bench`；OPERATIONS §4 改指仪表盘 + 新增 §6.5 密钥轮换 Runbook；PERFORMANCE §4 预算表与 flakiness 口径 | `grafana_dashboard_gate_test.go`（18/18 指标真实存在、7/7 `code` 在白名单、3/3 recording rule 匹配）；`bench_budget_test.go`（实测 PresignPut 362 allocs、加密写 151–155 allocs、账号写 O(n) 比值 23–27×，分配断言确定性、时间上限刻意宽松）；4 处变异 |
| 3 | 安全与输入空间 | 无机器可读漏洞披露入口；解析面无输入空间探索 | `.well-known/security.txt`（RFC 9116）；`internal/{s3wrap,store,handler}` 的 stdlib `testing.F` 目标；`fuzz.yml` 有界探索；`CITATION.cff`；`documentation.md` issue 模板 | `security_txt_gate_test.go`（必填字段 + `Expires` 必须未过期，过期即红灯强制续期）+ `TestWorkflowActionsAreShaPinned` |
| 4 | AI 效果证据 | 黄金任务集是散文、台账为空 | `scripts/evals/golden-tasks.yaml`（4 任务 / 16 命令 / 20 锚点）+ `run-golden-task.sh`（fail-closed）；AGENT_EVALS §六 首次台账（4 行，含一次 `fail` 基线如实保留） | `agent_evals_gate_test.go` 新增机器可读规格与 runner 断言；2 处变异（删任务 / 篡改标题）。**诚实边界**：无 golden task 端到端实跑，五维记 N/A ≠ 满分 |
| 5 | 英文与无障碍 | `docs/en/` 只有 README 翻译、无文档翻译政策与门禁；无障碍无自动检测 | `en/README.md`（英文导航）+ `en/architecture.md`（全文翻译，声明中文 SSOT + revision）；`i18n.md` §7 翻译覆盖政策；`e2e/a11y.spec.ts`（axe，4 状态 + 1 自检）；修 `--ok` / `--danger` 对比度 | `en_docs_gate_test.go`（来源声明 / revision / 可达性；3 处变异）；axe 实测先红（3 个状态 serious color-contrast）→ 修色 → 5/5 绿；`contrast_gate_test.go` 把 §5.5 表值与计数钉在 `styles.css`（2 处变异） |
| 6 | 数据模型 | 账号 / 驱动 / 信封事实散落四处 | `docs/data-model.md`（含 §0 SSOT 裁决表）；`internal/store` 注释把当前写入格式误标 `S3C2` 的 6 处改为 `S3C3` | `data_model_gate_test.go`（反射 `model.Account` 字段 + 解析 `store.Open` switch 驱动名；2 处变异） |

> **一处自我修正（记以免重蹈）**：`accessibility.md` §5.5 首版把 `--danger` / `--ok` 的浅色值写成
> **渲染态 axe 读数**（5.98 / 5.02），与表头声明的「token 公式值」口径不符；`contrast_gate_test.go`
> 对账时按 WCAG 2.1 重算为 **5.91 / 4.95** 并纠正，同时发现「4 组低于 AA」的计数漏了
> `--primary on --panel` 一行（实为 **共 11 行、浅色 5 行 / 深色 3 行**）。教训：静态记录必须由
> 机械重算钉住，手算数字会漂。

> **文档同步**：[`docs/data-model.md`](data-model.md)（新）、[`docs/en/README.md`](en/README.md) ·
> [`docs/en/architecture.md`](en/architecture.md)（新）、[`docs/README.md`](README.md)（导航 + 机器可读面）、
> [`docs/DEVELOPMENT.md`](DEVELOPMENT.md) §2（fuzz）/ §3（门禁落点 + CI 对照表 + 触发表）/ §4（同步表 +
> 登记表 + 命名）、[`AGENTS.md`](../AGENTS.md)、[`llms.txt`](../llms.txt)、[`CHANGELOG.md`](../CHANGELOG.md)、
> [`docs/OPERATIONS.md`](OPERATIONS.md)、[`docs/PERFORMANCE.md`](PERFORMANCE.md)、
> [`docs/AGENT_EVALS.md`](AGENT_EVALS.md)、[`docs/i18n.md`](i18n.md)、[`docs/accessibility.md`](accessibility.md)、
> [`README.md`](../README.md)；本节为证据台账落点。

---

### BH. 2026-09-30 门禁实跑回填与 §BG 结构归位（「未实跑」标记清零）

> **诊断**：§BD / §BF 与 CHANGELOG 两条目落笔时会话无 shell，留有 4 处「⚠️ 门禁未实跑待人类实跑」
> 标记；同批并行落笔的 §BG 被追加到文件末尾、错挂在 §三「质量与覆盖率现状」**之后**（字母台账
> 应全部位于 §二），目录行与头部摘要也停在 §BE / §BF、未含 §BG——正是命名 / 登记「同 PR 两处同改」
> 要防的那类漂移，只是这次发生在 FEATURES 自身。

| # | 缺口 | 处置 | 证据 |
|---|---|---|---|
| 1 | 4 处「门禁未实跑」标记（本文件 §BD / §BF、CHANGELOG 两条） | 按 AGENTS 提交前门禁全量实跑并把实测值写回原标记位：后端 `go vet` / `go build` / `go test ./...` **9/9 包全绿**、`golangci-lint` **0 issues**；前端 `pnpm lint` 0 告警 / `pnpm test` **74 文件 1132 例全绿** / `typecheck:e2e` exit 0 / `pnpm build` OK（366.72 kB，gzip 112.72 kB） | 各处原 ⚠️ 标记改为实测值；`docs_naming_gate_test.go` 文件头同步去掉「未实跑」 |
| 2 | §BD / §BF 留下两处**变异复核**未跑 | 亲手复跑两道：① `docs_naming_gate` 摘掉 `AGENTS.md` 命名清单的 `AGENT_EVALS.md` → `TestDocsNamingConventionRegistersEveryDocsFile` 红灯点名「AGENTS.md 命名约定段 缺 AGENT_EVALS.md」→ 还原绿灯；② `contrast_gate` 把 §5.5 结论句 `共 **11 行**` 改 10 → `TestContrastRecordCountClaimsMatch` 红灯点名「按 4.5:1 从表格重算为 11」→ 还原绿灯 | 复核命令与红灯文案见两门禁文件头；本节即实跑记录 |
| 3 | §BG 错挂在 §三 之后；目录 / 头部摘要缺 §BF / §BG 登记 | §BG 整块移回 §二 末尾（§BF 与 §三 之间，**纯移动 33 行、内容零改动**）；目录行补 BF / BG / BH，头部摘要补 §BG / §BH | `git diff --stat` 33 insertions / 33 deletions；`doc_link_gate` 校验目录新锚点可达 |

> **文档同步**：[`CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]` 一条；本节为证据台账落点。

---

### BI. 2026-09-30 第二轮收口（ROADMAP ✅ 行移出 / §AU 过期注记 / 评估残留缺口补登记 / §四 基线复测）

> **诊断**：第一轮（§BH）清完「未实跑」标记后继续逐台账回读，发现 4 类残留：① ROADMAP §六 第 3 条
> 「禁止保留已完成条目的 ✅ 行」被 §三 3.2 **#14** 违反（fuzz 已于同日落地仍留表内）；② 本文件 §AU 的
> 「未推送、未合并」随事实过期；③ 文档基线评估识别的 4 项缺口（图表 / 体量 / 评审周期 / PR ADR 勾选）
> **从未落账**——评估会话曾计划登记，但拟用编号 #69 被 CHANGELOG ↔ tag 问题占用并闭环，这批缺口随后无人认领；
> ④ ROADMAP §四 基线日期停在 2026-09-28。

| # | 缺口 | 处置 | 证据 |
|---|---|---|---|
| 1 | §三 3.2 #14 ✅ 行违反 §六 第 3 条 | 移出该行（编号不重排、#14 转空号），3.2 前言空号清单补 #14 并保留证据指针（§BG） | 移出前 `grep '§三 #14'` 仓内零引用，不产生悬空指针 |
| 2 | §AU「未推送、未合并」过期 | 原文保留（历史值不改写纪律）+ 2026-09-30 事实注记（已随后续提交进入 `develop`、与 `origin` 一致） | `git log -S AddTag` → `a984df7`；`git status` 显示与 `origin/develop` 一致 |
| 3 | 4 项评估缺口未落账 | ROADMAP §三 3.2 新增 **#19**：① `docs/` 0 图表 ② 超大文档无体量警示 ③ 评审周期未 CI 强制 ④ PR 模板缺 ADR 勾选 | 实测：`grep mermaid\|plantuml` 全 `docs/` 零命中、`docs/images/` 仅 2 图；CHANGELOG 323 KB / FEATURES 247 KB 平铺进 `llms.txt`（0 体量警示）；`docs/README.md` 自认「建议值，未在 CI 强制」；PR 模板验收清单无 ADR 项 |
| 4 | §四 基线日期停在 2026-09-28 | 改写为「2026-09-30 复测」并明示复测范围与未重跑项（覆盖率 / E2E / audit 类沿用行内最近实测值） | 同轮实跑：`gofmt` 干净、`govulncheck` 0 可达，叠加 §BH 已回填的 vet / lint / test / build / 前端全绿 |

> **文档同步**：[`ROADMAP.md`](ROADMAP.md) §三 3.2（移出 #14、新增 #19）与 §四、[`CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]` 一条；本节为证据台账落点。

---

### BJ. 2026-09-30 ROADMAP #19 文档可读性与流程机械化（4 项落地，**已收尾**）

> 来源：[`ROADMAP.md`](ROADMAP.md) §三 3.2 **#19**（2026-09-30 补登记、同日开工）。
> 收尾状态：4 项实现与文档同步先落地（两道新门禁红→绿 + 变异实跑通过），同日会话完成
> ⑤ CHANGELOG 记录、⑥ 移出 #19（转空号、证据指针入 3.2 前言空号清单）与 ⑦ 门禁全量复跑，
> **本批次闭环**。

| # | 缺口 | 处置 | 证据 |
|---|---|---|---|
| ① | `docs/` **0 张图表**（`mermaid` / `plantuml` 零命中），关键机制只有散文与 ASCII | [`architecture.md`](architecture.md) §5 数据流（对象上传）补 **mermaid 时序图**（直传 + multipart 两段），与四步文字版同段互指 | 图中链路逐条对 `handler/multipart.go` 核对：`init` / `complete` 走后端、分段由浏览器直传 |
| ② | 超大文档平铺进 [`../llms.txt`](../llms.txt) **无体量警示**（CHANGELOG ~319 KB / FEATURES ~244 KB） | 两行就地补 `⚠️ 超大`（>200 KB）+ 检索建议（按 `## [<版本>]` 段 / §字母章节）；新增 **`llms_size_gate_test.go`**：目标 >200 KB 的链接所在行必须含该标记 | 首轮**红灯**点名两行 → 补标记转绿；**变异实跑**删 CHANGELOG 行标记 → 红灯「目标 319 KB 超过 200 KB 阈值却无体量警示」→ 还原绿灯（扫描面 43 个文件链接，自检 ≥30） |
| ③ | 文档复审周期自认「建议值、未在 CI 强制、无任何自动化提醒」 | 新增 **`doc_review_gate_test.go`**：登记表带「N 个月」周期的行超过 `最后复审 + 周期` 即红灯点名，复核后更新日期即绿；无数值周期的口径仍靠人工 | 首轮绿（21 行 / 14 行带月数 / 0 过期）；**变异实跑**把一行日期改成 `2025-01-01` → 红灯「+ 3 个月 = 2025-04-01，今天 2026-09-30」→ 还原绿灯 |
| ④ | PR 模板验收清单缺「新增 / 变更决策是否补 ADR」勾选项 | [`../.github/PULL_REQUEST_TEMPLATE.md`](../.github/PULL_REQUEST_TEMPLATE.md) 验收清单新增「决策」行（补 ADR + `architecture.md` §7 取舍行同步 + 无取舍写「不适用」） | 与 `adr_coverage_gate_test.go` 的取舍行断言同一口径 |

> **文档同步（已完成）**：[`DEVELOPMENT.md`](DEVELOPMENT.md) §3（两道新门禁落点）与 §4（复审周期
> 「建议值」→ 带月数到期**有机械提醒**）、[`README.md`](README.md)（机器可读面 llms 行 + owner /
> 复审周期行）、[`../llms.txt`](../llms.txt)、[`../.github/PULL_REQUEST_TEMPLATE.md`](../.github/PULL_REQUEST_TEMPLATE.md)、
> 本节 + 目录行 + 头部摘要。**本文件即证据台账落点。**
>
> **收尾三步（2026-09-30 已完成）**：⑤ [`../CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]` 顶部
> 已记「ROADMAP #19 收尾」一条；⑥ [`ROADMAP.md`](ROADMAP.md) §三 3.2 已按 §六 第 1 条**移出 #19**
> （#19 转空号、编号不重排，前言空号清单已同步）；⑦ 提交前门禁全量复跑**全绿**——
> `go vet ./...` 干净、`go test ./...` **9/9 包**（`-count=1`）、`go build ./...` OK、
> 前端 `pnpm test` **74 文件 / 1132 例**、`pnpm build` OK（366.72 kB / gzip 112.72 kB）、
> 两道新门禁 `go test . -run 'TestLLMSTextWarns|TestDocReviewRegister' -count=1`
> **2/2 PASS**（21 行 / 14 行带月数 / 0 过期；43 个文件链接 / 2 个超大）。

---

### BK. 2026-09-30 KNOWN_ISSUES #70 闭环：`NormalizeEndpoint` 幂等修复（推翻 ➖ 决策）

> 来源：[`KNOWN_ISSUES.md`](KNOWN_ISSUES.md) **#70**（2026-09-30 §BG 原生 fuzz 实测发现，原登记 ➖
> 「fail-closed、维持现状」）。本批**推翻该决策**并闭环——推翻依据（4 处调用点影响面复核 +
> 函数头契约自违 + 登记内已写死的最小修法）写入登记的「处理中」版本，闭环后整行按约定移除、
> §四 台账补「已闭环移除」行（编号不重排）。本文件即证据台账落点。

| # | 项 | 证据 |
|---|---|---|
| 1 | **缺陷**：`NormalizeEndpoint("00  /")` → `"http://00  "`（尾随空白），再归一 → `"http://00"`，**不幂等**；`"host/a  /"` / `"http://host  /"` 同理。根因：`strings.TrimRight(rest, "/")` 在 host/path 切分**之前**执行，把尾斜杠后的内部空白顶到结果末尾，而入口 `TrimSpace` 只做一次。函数头明文承诺「去首尾空白」与幂等契约（`service.SameEndpoint` 与建 client 共用同一套规则），现状自违；调用点 4 处（`client.go` / `presign.go` / `ssrf.go` / `service/migrate.go`），输出进 `url.Parse` 与同端比较 | — |
| 2 | **推翻 ➖ 依据（动手前影响面复核）**：① fail-closed 只是「不可利用」，不豁免「比较判定同一端点、建出的 URL 却不同」的契约破坏；② 修法是登记内**已写死的最小改动**（先切分 host/path，再对 host `TrimSpace`），不碰 scheme 补全 / 大小写 / 路径保留语义；③ 两个退化种子已在 corpus、属性断言已具备，可 TDD 收口 | `KNOWN_ISSUES` #70 ➖ → 🔄 处理中（同日推翻并写依据）→ 闭环移除 |
| 3 | **TDD 先红**：`ssrf_fuzz_test.go` 新增 `TestNormalizeEndpointIdempotent`（7 例：斜杠前空格暴露 / 路径尾斜杠前空格 / scheme 后 host 尾斜杠前空格 / host 边缘空白 / host 内部空白保留 / 入口首尾空白 / 普通端点；断言**外部可见行为**——输出不以空白结尾 + 对自身幂等） | 首轮**红灯 3 例点名**：`"00  /" = "http://00  " 以空白结尾`、`"host/a  /" = "http://host/a  "`、`"http://host  /" = "http://host  "` → 实现后 **7/7 绿** |
| 4 | **修法**（`client.go` `NormalizeEndpoint`）：先切分 host/path → host `strings.TrimSpace` → path `TrimRightFunc`（`'/'` ∥ `unicode.IsSpace`，保输出不以空白结尾）→ host 与 path 全空返回 `""`（`"/"` 等输入与旧行为一致）。scheme 补全 / scheme 与 host 小写 / 路径内部原样均不变 | `go test ./internal/s3wrap/ -count=1` **PASS**、`-cover` **100.0% statements**；既有 `TestNormalizeEndpoint` / `TestNormalizeEndpointNeverDoubleScheme` **不改一字即绿** |
| 5 | **fuzz 死分支清理**：原「输出带尾随空白就跳过幂等断言」的短路分支修复后**不可达**（死代码零容忍）→ 删除；两枚退化种子**保留**在 corpus 作回归（原登记注释建议「修好后与种子一并删除」，此处**刻意不删**：保留 corpus 回归价值高于删除，确定性断言另由 `TestNormalizeEndpointIdempotent` 承担），注释同步改写为已修复口径 | `FuzzNormalizeEndpoint` / `FuzzValidateEndpoint` 各 **10s 有界 fuzz PASS**（678,909 + 246,990 次执行、0 失败，2026-09-30 实跑） |
| 6 | **变异验证**：把 `client.go` 中 host 的 `strings.TrimSpace(host)` 改为不生效（`_ = strings.TrimSpace(host)`） | **红灯点名** `"00  /"` 与 `"http://host  /"` 两例（path 尾空格例不归 host trim 管，故未点名——符合预期）→ 还原后 7/7 绿；复核命令写在测试文件头 |
| 门禁 | 全绿实测（2026-09-30） | `gofmt -l .` 干净 / `go vet ./...` **0 告警** / `go build ./...` OK / `golangci-lint run` **0 issues** / `go test ./... -count=1` **9/9 包** / `internal/s3wrap` **100.0% statements**；前端本轮未改 |

> **文档同步**：[`KNOWN_ISSUES.md`](KNOWN_ISSUES.md) #70 闭环移除（§二 表格行 + §四 台账「已闭环移除」
> + 头部已闭环编号区间补 `#70` + 前言注记改「已于同日闭环并移除」）；[`CHANGELOG.md`](../CHANGELOG.md)
> `[Unreleased]` 记一条「修复」；本节 + 头部摘要 + 目录行三处同步。`docs/archive/` 冻结件**不回写**。

---

### BL. 2026-09-30 ROADMAP #18 可观测性补全：五个缺失指标（发射点 + 告警 + 仪表盘 + 文档同步）

> 来源：[`ROADMAP.md`](ROADMAP.md) §三 3.2 **#18**（2026-09-30 会话中断批 Task C）。
> 收尾状态：五个指标的代码、测试、prometheus 规则、grafana 仪表盘与文档同步**先落地且门禁全绿**；
> 同日会话完成 ⑤ CHANGELOG 记录、⑥ 移出 #18（转空号、证据指针入 3.2 前言空号清单）与 ⑦ 包根门禁
> 复跑，**本批次闭环**。本文件即证据台账落点。

| 指标（逐字） | 类型 | 口径 / 关键取舍 | 发射点与证据 |
|---|---|---|---|
| `s3c_store_write_failures_total` | counter | 只数**真实写入失败**（落盘 / SQL 写错、已回滚）；重复 ID、`NotFound` 等业务拒绝**不计数**。`json` / `encrypted` 驱动 `Ping` 恒 nil，这是它们唯一的主动故障信号 | `store/metrics.go` 包级 `writeFailures` + `filestore.go` `persistLocked` / `sqlite.go` Create·Update·Delete 写失败分支各记一次；`gaps_test.go` `TestStoreWriteFailureCounter` |
| `s3c_jobs_active` | gauge | **未终结**任务数，与 `JobRegistry` 在册上限（256）同口径 | `service/job.go` `JobRegistry.ActiveCount()`（锁内复用 `runningCountLocked`）；`job_cap_test.go` `TestJobRegistryActiveCount` |
| `s3c_http_request_duration_seconds` | histogram | 桶上界 `0.005 … 30` + `+Inf`；`+Inf` 与 `_count` 都取 `s3c_http_requests_total`（同一次 `recordHTTPMetric` 记录，结构上不会漂移） | `handler/metrics.go` `recordHTTPMetric(status, dur)`；`middleware.go` 的 `withLogging` 传入 `dur`（计数与直方图同源）；`metrics_test.go` `TestMetricsHTTPDurationHistogram` |
| `s3c_volume_size_bytes` / `s3c_volume_free_bytes` | gauge | `S3C_DATA_DIR` 的 statfs / `GetDiskFreeSpaceExW`；**取不到就不发序列**（平台不支持 / 目录未配置 / statfs 失败），不用 0 冒充 | 新增 `volume_unix.go`（`linux \|\| darwin \|\| freebsd`，`syscall.Statfs`）、`volume_windows.go`（kernel32 延迟绑定——标准库 `syscall` **没有**该 API，`golang.org/x/sys` 违 ADR-004）、`volume_other.go`（恒 `ok=false`） |
| `s3c_volume_inode_total` / `s3c_volume_inode_free` | gauge | 同一 statfs 的 `Files` / `Ffree`；**口径与容量序列一致**——取不到就不发序列（0 会让 `free/total` = 0/0 = NaN，告警静默失效）。2026-10-10 补上，收口 OPERATIONS §4.3 原「仍无内置指标」行 | `volume_unix.go` `volumeInodes()`、`volume_windows.go`（Windows 无 inode 概念 → 恒 `ok=false`）、`volume_other.go`（恒 `ok=false`）；`metrics.go` 在 `h.dataDir != ""` 内输出；`metrics_test.go` `TestMetricsVolumeInodes` |
| `s3c_last_shutdown_duration_seconds` | gauge | **上一次**关停耗时：关停在进程退出前发生、scrape 赶不上 → 落盘 `data/shutdown.json` `{"durationUs":N}`，**下次启动载入**后暴露；0 = 尚无记录（单位取微秒，避免亚毫秒关停截断成 0 与「无记录」撞车） | 新增 `handler/shutdown_metric.go`（`LoadLastShutdown` / `RecordShutdown`）；`main.go` 在 `srv.Shutdown` 返回后 `RecordShutdown`；`main_test.go` `TestMainServerSubprocess` 断言子进程退出后文件存在且 `durationUs ∈ (0, 5e6)` |

**告警与仪表盘**（同批落地，两道既有门禁同步钉住「引用的指标必须真实发射」）：

- `deploy/prometheus/s3client.rules.yml`：新增记录规则 `s3client:http_latency_p95:rate5m` + 4 条告警
  `S3ClientStoreWriteFailures` / `S3ClientJobsNearCapacity` / `S3ClientVolumeSpaceLow` / `S3ClientHTTPLatencyHigh`。
- `deploy/grafana/s3client.dashboard.json`：行⑥由「没有面板」改写为真面板 + 新行⑦，面板 **31 → 40**。
- 变异验证（5/5，改坏 → 红灯点名 → 还原 → 绿）：① 规则里 `s3c_jobs_active` → `s3c_jobs_active_broken`
  → `TestPrometheusRulesReferenceRealMetrics` 红并点名；② 仪表盘 **`expr`** 里的
  `s3c_volume_size_bytes` → `…_bytess` → `TestGrafanaDashboardReferencesRealMetrics` 红并点名
  （注意：改 `description` **不会**红——门禁只扫 `expr`）；③ `metrics.go` 去掉 `metricHTTPDurSumNs.Add`
  → `TestMetricsHTTPDurationHistogram` 红（`_sum 未随请求增长`）；④ `main.go` 去掉
  `h.RecordShutdown(...)` → `TestMainServerSubprocess` 红（`shutdown.json` 不存在）；⑤
  `filestore.go` 去掉 `noteWriteFailure()` → `TestStoreWriteFailureCounter` 红（`0 -> 0`）。

> **门禁实跑（2026-09-30，本会话实测）**：`go vet ./...` 干净 / `golangci-lint run` **0 issues**
> （`GOOS=windows golangci-lint run ./internal/handler/` 同为 0）/ `go test ./... -count=1`
> **9/9 包 ok**（root 4.3s / handler 135.9s / s3wrap 43.2s / service 28.9s / store 3.6s）/
> `go test -race -coverprofile` + awk `count==0` → **9/9 包 100.0%、无未覆盖块** /
> 交叉编译 `GOOS=windows · darwin · freebsd · openbsd` build OK（`netbsd` / `dragonfly` 在依赖侧
> `modernc.org/libc` 即编不过，与本批次无关）/ 包根文档门禁 `go test . -count=1` ok（4.7s）。
> **前端 `pnpm test` / `pnpm build` 未跑**——本批次未改 `apps/web`（Task D 必须跑）。

>
> **文档同步**：[`OPERATIONS.md`](OPERATIONS.md)（§3.2 表 +6 行、两套直方图桶上界、观测缺口段改
> 「已补齐」、§4.1 +3 SLI、§4.2 +4 告警、§4.3、§10.3、R-9、组件表、备份表 +`shutdown.json`）、
> [`api.md`](api.md) 指标条目、[`DEPLOYMENT.md`](DEPLOYMENT.md) §6.3、[`CONFIGURATION.md`](CONFIGURATION.md)
> `S3C_DATA_DIR` 行、[`../CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]` 一条「变更」、
> 本节 + 头部摘要 + 目录行三处同步。

---

### BM. 2026-09-30 ROADMAP #17 可访问性补强与自动化扫描（5 项全部落地，**已收尾**）

> 来源：[`ROADMAP.md`](ROADMAP.md) §三 3.2 **#17**（2026-09-30 自 [`accessibility.md`](accessibility.md)
> §5.3–§5.5 收口迁入）。5 个子项同批完成：焦点陷阱 / 播报 / 表格与标签 / 组件级 axe / 对比度修色；
> 同日会话完成 ⑤ CHANGELOG 记录、⑥ 移出 #17（转空号、证据指针入 3.2 前言空号清单）与 ⑦ 全量门禁复跑，
> **本批次（含 #19 / #70 / #18 / #17 四项）闭环**。文档 SSOT：[`accessibility.md`](accessibility.md)。

| # | 子项 | 处置 | 证据 |
|---|---|---|---|
| ① | 焦点陷阱只覆盖 `ModalDialog`，`ConfirmDialog` / `PromptDialog` / `PreviewOverlay` 可被 Tab 逃出 | 新增 **`composables/useFocusTrap.ts`**（记住 / 恢复焦点 + 初始移入 + `Tab` 首尾回卷），四模态共用；`Tab` 回卷仍由各模态 `useKeydownStack` 栈顶 handler 调 `trapTab(e)`，不破坏 LIFO | 新增 **7 条用例**（三个模态各补焦点恢复 + 首尾回卷 / 中间不干预；`ConfirmDialog` 另补「打开后同 tick 内卸载不抛错」）；`ModalDialog` 原 14 例**不改一字即绿**。`useFocusTrap.ts` 覆盖率 **100%**。**关键取舍**：初始聚焦必须 `await nextTick()`（`flush:'post'` 会跑在 `v-model` 指令 mounted 之前 → 全选落空），故续体开头有空容器守卫 |
| ② | 两处 `aria-live` 无测试断言；操作失败横幅多数不经 live region | 成功统一走 `toast()` → `Toasts` 的 `aria-live="polite"`；失败的 8 处内联横幅统一 `role="alert"`（`ObjectsPanel` / `AccountsPanel` ×2 / `BucketsPanel` / `CompareDialog` / `MigratePanel` / `RecycleBinPanel` / `ServerPanel` 的 `msg err` + `PromptDialog` 校验失败） | `a11y_gate.test.ts` **新源码门禁**「每个 `msg err` / `modal-err` 开标签必须带 `role="alert`」+ **变异实跑**（去掉 `ServerPanel` 的 `role` → 红灯列出该标签 → 还原绿）；行为断言 3 条：`Toasts` 成功 / 失败同区、`BatchMetadataDialog` 状态区 `aria-live` 承载 running / done、`PromptDialog` 校验失败 `role="alert"` |
| ③ | 15 张表无 `caption`、行选中无语义；18 个控件只有 placeholder / `title` | 15 张 `<table>` 全部加 **sr-only `<caption>`**（`styles.css` 新增全局 `.sr-only`，裁剪到 1px 但不 `display:none`）；四张行选中表的数据行加 **`:aria-selected`**；18 个控件补**可见标签**（包裹 `<label class="field">` / 行内 `<label for>` / `aria-labelledby` 指向可见列头三种形态，账号与桶切换下拉把**紧邻的可见徽标**直接改成 `<label for>`——**零视觉改动**） | `a11y_gate.test.ts` 两条新源码门禁：每张 `<table>` 紧跟 `<caption class="sr-only">`、行选中表声明 `:aria-selected`（**自检**：全仓恰好 4 个组件）；行为断言 3 条：`ObjectList` 行 `aria-selected` 随选中集合变化、`TagsDialog` `label[for]` 能解析到目标输入框、`BatchMetadataDialog` 三控件被可见 `<label>` 包裹。扫描复核：可见表单控件 **77 / 77** 有可关联标签，仅剩 2 个 `display:none` 文件选择框（不可见、由带标签的拖放区触发）。**顺带修掉**：批量元数据 3 处 `aria-label` 是**硬编码英文**（中文界面朗读英文），改为与可见标签同一条 i18n 键（+4 个 i18n 键：`batchEdit.tagsModeLabel` / `headers.colKey` / `toolbar.pathLabel` / `toolbar.filterLabel`，zh / en 各一） |
| ④ | 组件级 axe 扫描缺失（渲染态只有 E2E 侧 4 个初始状态） | 新增 **devDependency `vitest-axe`**（`dependencies` 仍只有 `vue`，不违 ADR-004）+ **`src/a11y_axe.test.ts`**：对挂载后的组件 DOM 跑 WCAG 2.0 / 2.1 A + AA 规则集，serious / critical 即红灯；非阻塞项打印供人工判断；happy-dom 无 CSS 级联故显式关掉 `color-contrast` | **6 例**：`ModalDialog` 带 footer / `ConfirmDialog` 危险态 / `PromptDialog` 带校验失败 / `Toasts` 成功+失败堆叠 / `ObjectList` 带 caption 与选中行 / **空跑自检**（注入 `image-alt` 必须被报出）。**落地当天即抓到真问题**：`aria-multiselectable` 写在原生 `<table>` 上 → `aria-allowed-attr`（critical），已按其判定移除（详见下表第 5 行） |
| ⑤ | 对比度修色收尾：§5.4「剩余」6 组低 AA | `styles.css` 修色 + §5.5 表**同步重算**：`--muted` `#64786f`→`#62766d`（4.44→4.57）、`--primary` `#10a37c`→`#0d8162`（3.21 / 3.02→4.85 / 4.56）、`--placeholder` 浅 `#8aa89a`→`#657b71`（2.58→4.54）、深 `#5f7a6d`→`#6e867a`（3.81→4.54）、`--danger` 深 `#ef4444`→`#f05050`（4.25→4.54）、`--brand-from/to` `#2dbd98` / `#0e8c66`→`#1f8067` / `#0d825e`（白字 2.37 / 4.23→4.84 / 4.80） | §5.5 现为 **13 行**（原 11 + 新增 `--brand-mark-from/to` on `--bg` 两行）、**双主题各 0 行低于 AA**；`--brand` **拆 token**：同一渐变既是白字背景又是 logo 文字、方向相反，故新增 `--brand-mark-*` 专供 header logo（浅色沿用加深值 4.56 / 4.52，**深色块回亮值** `#2dbd98` / `#138e69` → 7.87 / 4.54），避免「为白字加深」反把深色主题 logo 拖到 4.1:1。**变异实跑**：把 `--muted` 改回 `#64786f` → `contrast_gate` 红灯点名两格「记 4.57，重算 4.44」→ 还原绿 |

> **门禁实跑（2026-10-01，全部为本会话实测）**：前端 `pnpm test` **75 文件 / 1155 例**（较批次前 +23）、
> `pnpm test:coverage` 四指标 **100%**（statements 4321 / branches 2932 / functions 1130 / lines 3706）、
> `pnpm lint` **0 告警**、`pnpm build` OK（`index.js` 371.05 kB / gzip 113.84 kB、`index.css` 32.40 kB）；
> 后端 `gofmt -l .` **干净**（顺带修正上批遗留的两处注释对齐）、`go vet ./...` 干净、
> `golangci-lint run` **0 issues**、`go test ./... -count=1` **9/9 包**（root 4.4s / handler 131.8s /
> s3wrap 33.3s / service 29.5s / store 3.0s）、包根文档门禁 `go test . -count=1` **ok**、
> `go test . -run TestContrast -count=1` **PASS**、`make check` **exit 0**、`make bench` **exit 0**。
> **同轮补跑的渲染态 / 真实链路验证**（本批次改了前端，按 AGENTS 必跑）：
> `pnpm e2e` **22 passed / 0 skipped**（含 `e2e/a11y.spec.ts` 5 条 axe 扫描——本批配色 / `role` /
> `caption` 改动在**真实 Chromium + 真实构建产物**上零 serious/critical）；
> `make e2e-real` **3 passed**（真实后端 + RustFS + 真实产物，`S3C_TOKEN` 生产同构形态；
> 本机 8080 被 `haproxy` 占用，用 `SERVER_PORT=8081`，脚本编排不变）；
> `S3CLIENT_E2E=1` **4/4 PASS**（临时起 RustFS 跑完即删）；
> `govulncheck ./...` **0 可达**、`cargo audit --no-fetch` **0 漏洞**。
> **唯一未能实跑**：`pnpm audit`——所用镜像 `registry.npmmirror.com` 不提供
> `/-/npm/v1/security/audits` 端点（`ERR_PNPM_AUDIT_ENDPOINT_NOT_EXISTS`），沿用 CI 结果。

> **文档同步**：[`accessibility.md`](accessibility.md)（§1.2 属性分布 +2 行 / `role` 分布 `alert` 1→9、
> §1.3、§1.4 重写为「统一播报」口径、**新增 §1.6** 表格与可见标签、§2.2 重写为 `useFocusTrap`、
> §4 第 2 / 7 / 8 / 9 / 10 条改 ✅ 并补 `vitest-axe`、§5.2 表 +3 行、§5.3 `vitest-axe` 划掉、
> §5.4 中 / 低两行收口、**§5.5 整表重算 + 结论改写**）、[`../CHANGELOG.md`](../CHANGELOG.md)
> `[Unreleased]` 一条「变更」、[`../AGENTS.md`](../AGENTS.md) / [`DEVELOPMENT.md`](DEVELOPMENT.md) §4 /
> [`../llms.txt`](../llms.txt) 三处命名清单 + [`README.md`](README.md) 导航行（`handoff-20260930.md` 登记）、
> [`../apps/web/package.json`](../apps/web/package.json) devDependencies +1、本节 + 头部摘要 + 目录行三处同步。

### BN. 2026-10-01 ROADMAP #10 OpenAPI → 前端类型 / 客户端代码生成（schema-first + diff 门禁，**已收尾**）

> 来源：[`ROADMAP.md`](ROADMAP.md) §三 3.2 **#10**（原编号 #53）。目标是把「靠契约测试**测出**漂移」
> 升级为「结构上**产生不出**漂移」：`/api/openapi.json` 成为前端 URL / method / 实体类型的唯一来源，
> 生成物提交进仓库并由**新鲜度门禁**钉死。生成器与解析库均为 **devDependency**，`dependencies`
> 仍只有 `vue`，不违 [ADR-004](decisions/0004-minimal-frontend-deps.md)。

| # | 子项 | 处置 | 证据 |
|---|---|---|---|
| ① | 生成器与两份提交产物 | 新增 **`apps/web/scripts/gen-api.mjs`**（读 `docs/api/openapi.json`）+ `package.json` 脚本 **`pnpm gen:api`**，产出 `src/api/schema.d.ts`（**4506 行**，`openapi-typescript@7.13.0` Node API 生成，**无时间戳横幅**故字节稳定）与 `src/api/operations.ts`（**88 行 / 70 条** `operationId → {method, path, params}`）；`--check` 逐字节比对两份产物，不一致即退出 1 并提示跑 `pnpm gen:api`。缺 `operationId` 或 id 冲突时生成器直接失败 | `pnpm gen:api --check` **exit 0**（本批实跑）；解析到 **70 个操作 / 53 条 path / 5 个共享 `components.schemas`** |
| ② | URL 与 method 真正来自 spec（消除漂移的那半边） | `src/api/http.ts` 新增 **`opPath()`** + 条件剩余元组 `PathArgs<Id>`：无占位符的操作可省参，**有占位符的操作漏参即编译错误**，替换一律 `encodeURIComponent`；`src/api/endpoints.ts` **整体重写**——**58 处 URL / method** 全部改走 `opPath('<opId>')` / `operations.<opId>.method` | `src/api.test.ts` 全部 URL / method 断言**通过**（证明 operationId ⇔ wrapper 映射正确）。**三处 wrapper 与 operationId 不同名**已记录：`listVersions`→`listObjectVersions`、`copyFilesAsync`→`copyObjectsAsync`、`downloadZipToDisk`→`downloadZip` |
| ③ | 散落的硬编码 `/api/…` 收编 | 另 5 处一并改造：`src/api/download.ts`、`src/api/jobs.ts`（含 SSE `migrateJobEvents` 的 `method`）、`src/api/index.ts`、`src/proxy.ts`（保留自有 `apiBase` 前缀）、`src/components/ServerPanel.vue` | 全仓生产代码**不再手写端点路径**；query 串**刻意仍手写**——query 名不在 path 模板里，且已由后端 `openapi_query_params_test.go` 钉住 |
| ④ | 黄金 spec 的 `required` 缺口（前端类型变严格的前置修复） | 后端 `internal/openapi/openapi.go` `sharedSchemas()`：`AccountView` / `objectItem` / `listObjectsResponse` **均无 `omitempty`**（字段恒定序列化），但 `required` 只列了 4/12、4/6、3/4 —— 补齐 `Account` 12 项、`ObjectItem` 6 项、`ListObjectsResp` 4 项；黄金契约 `-update-openapi-spec` 重生成（`docs/api/openapi.json` **+13/-2**） | **这是 spec 的缺陷，不是前端类型的问题**；响应契约测试只比**字段集合**不比 `required`，故收紧后 `TestOpenAPI\|TestCommitted\|TestAPI` 仍 ok。**57 条内联 2xx 响应仍无 `required`**，端点级响应类型继续手写——**未并入 #10 的完成口径** |
| ⑤ | 生成物成为真实消费者 + 新鲜度门禁 | `src/types.ts` 头部改为从 spec 派生 4 个共享实体（`Account` / `ObjectItem` / `ListObjectsResponse` / `BucketItem`），使 `schema.d.ts` 不是孤儿模块；新增 **`src/api/generated.gate.test.ts`**（**3 例**）：新鲜度逐字节比对、结构自检（条数 ≥60 防「空表永远绿」+ 占位符 ⇔ `params` 逐个对齐）、`opPath` 编码行为 | 死代码门禁曾红报 `./api/schema.d.ts（模块无人 import）`，派生后 `pnpm test` 全绿——**门禁倒逼 codegen 必须有真实消费者**。新鲜度门禁是 #10「生成物入 CI diff 门禁」的**字面落地** |

> **门禁实跑（2026-10-01，全部为本会话实测）**：前端 `pnpm test` **76 文件 / 1158 例**（较批次前 +3，
> 新增 `src/api/generated.gate.test.ts`）、`pnpm test:coverage` 四指标 **100%
> （4327 / 2932 / 1131 / 3712）**、`pnpm lint` **0 告警**、`pnpm typecheck` + `pnpm typecheck:e2e`
> 均 exit 0、`pnpm build` OK（`index.js` **377.34 kB / gzip 114.83 kB**、CSS 32.40 kB）、
> `pnpm gen:api --check` **exit 0**；后端 `gofmt -l .` **干净**、`go vet ./...` **0 告警**、
> `go build ./...` OK、`golangci-lint run` **0 issues**、`go test ./... -count=1` **9/9 包**、
> `make test-cover` **9/9 包 100.0%（`count==0` 零块）**、`govulncheck ./...` **0 可达漏洞**
> （另 21 个「被 require 但未调用」的模块级漏洞，不构成可达面）。
> **仓库级**：`make check` **exit 0**、`make bench` **exit 0**。
> **渲染态与真实链路**（本批改了前端，按 AGENTS 必跑）：`pnpm e2e` **22 passed / 0 skipped**、
> `make e2e-real` **3 passed**（真实后端 + RustFS + 真实产物；本机 8080 被 `haproxy` 占用故用
> `SERVER_PORT=8081`）、`S3CLIENT_E2E=1 go test ./internal/s3wrap/ -run TestE2E -v` **4/4 PASS**
> （临时起 `rustfs/rustfs:1.0.0-rc.3` 于 `127.0.0.1:9000`，跑完 `docker rm -f` 并核验端口已关闭）。
> 第三方许可证清单已按新 devDependency 重生成：`./scripts/gen-third-party-licenses.sh` →
> **Go 43 模块 / Rust 428 crates / npm 1 包，UNKNOWN 0 项**（devDependency 不入清单，内容无变化）。
> **唯一未能实跑**：`pnpm audit`——镜像 `registry.npmmirror.com` 不提供 audit 端点
> （`ERR_PNPM_AUDIT_ENDPOINT_NOT_EXISTS`），沿用 CI 结果。

> **文档同步**：[`../CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]` 一条「变更」、
> [`ROADMAP.md`](ROADMAP.md) §三 3.2 **删除 #10 行**（转空号，前言空号清单 +1）与 §四 门禁基线
> （例数 / 覆盖率 / **新增「前端生成物新鲜度」一行**）、[`DEVELOPMENT.md`](DEVELOPMENT.md) §3
> 门禁落点补 `src/api/generated.gate.test.ts`、[`api.md`](api.md) 「前端可基于此生成 TypeScript
> client」由设想改为既成事实、本节 + 头部摘要 + 目录行三处同步。

### §BO 2026-10-08 KNOWN_ISSUES #71 闭环：仓库 slug 统一为 `github.com/weilai1949/s3client`

> 来源：[`KNOWN_ISSUES.md`](KNOWN_ISSUES.md) **#71**（2026-09-30 登记为 ➖「维持现状」，理由是
> 「改模块路径 = 全仓 import 路径重命名、与当时批次正交」；2026-10-08 **推翻该决策**——理由没有变，
> 成本已被量化为一次纯机械替换，一次做完即可）。`git remote` / [`.well-known/security.txt`](../.well-known/security.txt)
> / [`CITATION.cff`](../CITATION.cff) 本就用 `s3client`，本批把两侧对齐到该取值（`s3client` 实测 301 → `s3client`）。

| # | 项 | 改动 |
|---|---|---|
| 1 | 模块路径 | [`apps/server/go.mod`](../apps/server/go.mod) `module github.com/weilai1949/s3clinet/apps/server`（**改前值**，全仓已无此写法）→ `…/s3client/apps/server`；全仓 Go import（含 `deadcode_gate_test.go` / `third_party_licenses_gate_test.go` 等**字符串字面量里的示例 import**）同步改写，**120 个受版本控制的文件**（Go 110 + `go.mod` 1 + 非 Go 9；另有 2 个被 `.gitignore` 忽略的本地旧构建产物同批删除） |
| 2 | 仓库 URL | `.github/SECURITY.md` · `.github/ISSUE_TEMPLATE/config.yml` · `.github/SUPPORT.md` · `.github/CONTRIBUTING.md` · [`docs/AI_POLICY.md`](AI_POLICY.md) · [`docs/en/index.md`](en/index.md) · [`docs/api/accounts.schema.json`](api/accounts.schema.json) 的 `$id` · `deploy/grafana/s3client.dashboard.json` 面板 `url` · 根 [`README.md`](../README.md) 的 GitHub Release 链接 —— 统一 `github.com/weilai1949/s3clinet`（**改前值**）→ `…/s3client` |
| 3 | **刻意不动（本批范围边界；同日已被 §BP 取代）** | 本批**只统一仓库 slug**：品牌（旧写法 `s3clinet` 的文档标题 / OpenAPI `title` / 启动日志 / `CITATION.cff` 标题）、运行时工件（旧锁文件名 / 二进制 / 镜像 / `container_name` / Cargo 与 npm 包名）、监控命名空间（旧记录规则前缀与告警名、`deploy/{prometheus,grafana}/s3clinet.*` 文件名）**当时刻意不动**；**同日第二批已全部统一为 `s3client`，见 §BP**。`CHANGELOG.md` 历史条目与 `docs/archive/` 冻结件按「历史不回写」惯例仍保留旧写法 |
| 4 | 残留核对 | 改前写法 `github.com/weilai1949/s3clinet` 全仓 `grep` → **活引用 0 处**（import / 模块路径 / 仓库 URL / `$id` / 面板链接全清零）；剩余仅为**叙述性「改前值」**——本节与 §BP、[`../CHANGELOG.md`](../CHANGELOG.md) 历史条目、[`archive/`](archive/) 冻结件（豁免口径同 `assessment.md` 归档先例）；`apps/server/s3client-server`、`.run/…/s3client-server` 两个本地构建产物（均 `.gitignore` 忽略）已删除，下次构建即用新路径重新产出 |

> **门禁实跑（2026-10-08，全部本机实跑，未编造）**：
>
> | 门禁 | 结果 |
> |---|---|
> | `cd apps/server && go vet ./... && go build ./... && go test ./... -count=1` | **EXIT=0，9/9 包 ok**（root 3.395s / handler 125.530s / s3wrap 32.742s / service 32.536s / store 2.859s / config / model / openapi / atomicfile 各 <10ms） |
> | `golangci-lint run` | **0 issues** |
> | `cd apps/server && go test . -count=1`（包根文档门禁族） | **ok 4.939s**（改名后首跑）→ 文档同步（本节 / `CHANGELOG` / `KNOWN_ISSUES` 写回）后**复跑 ok 4.235s** |
> | `cd apps/web && pnpm test` | **76 文件 / 1158 例全绿**（7.45s） |
> | `cd apps/web && pnpm lint` | **0 告警** |
> | `cd apps/web && pnpm build` | **OK**（vue-tsc + vite，377.34 kB / gzip 114.83 kB） |
>
> **本批不需要重生成的产物**：`docs/api/openapi.json` 里没有仓库 URL（`title` 属产品名、本批不动），
> `pnpm gen:api --check` 面不受影响；`docs/THIRD_PARTY_LICENSES.md` 与依赖无关，未重跑。
> **同日 §BP 已重生成前两者**（`title` 改为 `s3client API` → `-update-openapi-spec` + `pnpm gen:api`）。

> **文档同步**：`CHANGELOG.md` `[Unreleased]` 一条「变更」；[`KNOWN_ISSUES.md`](KNOWN_ISSUES.md)
> **#71 整行移除**（§二 表 + 「故本表当前为」一句 + 编号台账改「已闭环移除」+ 头部新增
> **2026-10-08 段**并把上一轮降为「上一轮更新」+ 闭环编号清单补 #71）；本节 + 头部摘要 + 目录行三处同步。


### §BP 2026-10-08 KNOWN_ISSUES #71 第二批：产品名 `s3clinet` → `s3client` 全量统一

> 来源：接同日 **§BO**（仓库 slug 统一）。§BO 把品牌 / 运行时 / 监控命名空间列为「刻意不动」的范围边界，
> **本批按要求把该边界一并取消**——#71 的闭环口径由「只统一 slug」扩为「slug + 产品名一次到底」。
> 改前值（旧写法，**全仓活引用已清零**）：`s3clinet` / `S3Clinet` / `S3CLINET`。

| # | 项 | 改动 |
|---|---|---|
| 1 | 文本与配置 | **86 个受版本控制的文件、358 处**写法替换（`s3clinet` → `s3client`、`S3Clinet` → `S3Client`、`S3CLINET` → `S3CLIENT`）：`.md` 30 · `.go` 18 · `.yml` 8 · `.json` 6 · `.conf` 6 · `.sh` 5 · `.ts` 2 · `.example` 2 · `txt/toml/rs/lock/.gitignore/.dockerignore/cff/Dockerfile/Makefile` 各 1；含根 `AGENTS.md`（`S3CLIENT_E2E`）、`llms.txt`、`CITATION.cff`（`title: "S3 Client (s3client)"`）、两份 `.env.example`、`Makefile`、`.gitlab-ci.yml`、4 个 GitHub workflow |
| 2 | 文件改名 5 个 | **`git mv` 保留历史**：`deploy/prometheus/s3clinet.rules.yml` → `s3client.rules.yml`、`deploy/grafana/s3clinet.dashboard.json` → `s3client.dashboard.json`、`deploy/nginx/conf.d/s3clinet-{tls.example,docker,local}.conf` → `s3client-*`；全仓引用同批改（`grafana_dashboard_gate_test.go` 的路径常量 + `s3client:*` 记录规则正则、`repo_infra_gate_test.go` 的 `rulesRel`、compose 与 `nginx*.conf` 的 include、`docs/README.md` / `OPERATIONS.md` / `DEVELOPMENT.md` / `CHANGELOG` 的链接） |
| 3 | 运行时与产物（**含行为变更**） | 单写者锁文件 `.s3clinet.lock` → **`.s3client.lock`**（旧锁文件在数据目录只是空文件、锁由内核持有，残留无影响）；启动日志 `msg="s3clinet server"` → `s3client server`；二进制 `s3client-server`（Dockerfile `ENTRYPOINT` / compose / `scripts/*.sh` / `-healthcheck`）；镜像与 `container_name` 的 `s3client/server` 系；Cargo 包 `s3client` + Tauri `productName` / `identifier`（`com.weilai1949.s3client`）；npm `s3client-web` / `s3client-desktop`；E2E 环境变量 `S3CLIENT_E2E` / `S3CLIENT_{ENDPOINT,ACCESS_KEY,SECRET_KEY}` |
| 4 | 契约与生成物 | `info.title` `s3clinet API` → **`s3client API`**（`description` 同步）→ `go test ./internal/handler/ -run TestCommittedOpenAPISpecMatchesRuntime -update-openapi-spec` 重生成 [`api/openapi.json`](api/openapi.json)（3 行）→ `pnpm gen:api` 重生成 `schema.d.ts`（`gen:api --check` exit 0）；`docs/api.md` 示例响应与 `accounts.schema.json` `title` 同步 |
| 5 | 监控命名空间 | 记录规则 `s3clinet:*` → **`s3client:*`**、告警 `S3Clinet*` → **`S3Client*`**（`S3ClientStoreWriteFailures` / `S3ClientJobsNearCapacity` / `S3ClientVolumeSpaceLow` / `S3ClientHTTPLatencyHigh`）；`OPERATIONS.md` §4.1 / §4.2 表与 `docs/README.md` 机器可读面、`DEVELOPMENT.md` §4 对照表同步 |
| 6 | **豁免（历史不回写）** | ① `CHANGELOG.md` **历史条目**：只修正**指向改名文件的路径链接**（10 处，否则 `doc_link_gate` 红灯），叙述里的旧名照旧；② `docs/archive/` **冻结件整份不动**。故全仓仍可见旧写法的位置**仅这两类**——`grep` 计数：`CHANGELOG` 28 + `archive` 20（改前全仓 416，其中 358 已替换、10 处为链接目标改写） |

> **门禁实跑（2026-10-08，全部本机实跑）**：
>
> | 门禁 | 结果 |
> |---|---|
> | `cd apps/server && go vet ./... && go build ./... && go test ./... -count=1` | **0 告警 / OK / 10/10 包 ok**（`go test -race -count=1 -coverprofile=coverage.out ./...`，各包均 **100.0%** 语句覆盖，profile `count==0` 零块；同树含同日 ROADMAP #8 / #11 / #13 批次新增的 `internal/tracing`，故后端包数 **9 → 10**） |
> | `golangci-lint run` | **0 issues**（v2.13.2，与 CI 同版本） |
> | `cd apps/server && go test . -count=1`（包根文档门禁，含 `doc_link` 对 10 处改链的校验） | **ok 4.198s**（2183 条相对链接目标均存在、25 条外链跳过；同树另含 #8 / #11 / #13 台账与新增 ADR-013） |
> | `cd apps/web && pnpm test` | **78 文件 / 1189 例** 全绿（含 `generated.gate` / `deadcode_gate` / i18n 覆盖；同树含 #8 新增 `multipartResume.test.ts` / `api/download.test.ts`） |
> | `cd apps/web && pnpm lint` / `pnpm gen:api --check` / `pnpm build` | **0 告警**（`--max-warnings 0`）/ **exit 0** / OK（**381.41 kB，gzip 116.15 kB**，CSS 32.40 kB） |

> **文档同步**：[`../CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]` 一条「变更（…第二批：产品名全量统一）」；
> [`KNOWN_ISSUES.md`](KNOWN_ISSUES.md) 头部 2026-10-08 段改写为「同日两批」+ #71 台账行同步；
> 本节 `§BO` ③ 标注「同日已被 §BP 取代」；本节 + 头部摘要 + 目录行三处同步。


### §BQ 2026-10-08 ROADMAP #8 大文件体验：上传断点续传 + 下载并行分段

> 来源：ROADMAP §三 3.2 **#8**（原 `KNOWN_ISSUES` #51，本条落地后按 §六 第 1 条移出、编号转空号）。
> 设计口径由人类在开工前拍板：**允许新增 1 个只读端点**（列出已上传分段），续传以**服务端真实清单**为准。

| # | 交付 | 关键取舍 / 口径 |
|---|---|---|
| 1 | 后端只读端点 `GET /api/accounts/{id}/multipart/parts` | query `bucket` / `key` / `uploadId` → `{"parts":[{partNumber,etag,size,lastModified}]}`；缺 `key` / `uploadId` → 400；`bucket` 缺省回退账号默认桶；**端点总数 70 → 71**；AWS SDK 类型止步 `s3wrap`（自有 DTO） |
| 2 | [`s3wrap.ListParts`](../apps/server/internal/s3wrap/multipart.go) | `MaxParts=1000` + `NextPartNumberMarker` 自动翻页；ETag 去引号；`NoSuchUpload` 原样透传（前端据此判定「会话失效」而非「清单为空」） |
| 3 | 前端续传 [`multipartResume.ts`](../apps/web/src/multipartResume.ts) + [`upload.ts`](../apps/web/src/upload.ts) | 指纹 `name:size:lastModified` → localStorage 单键 `s3c.multipart.resume.v1`（上限 20 条）；**只存 accId / bucket / key / uploadId / parts，零凭证**（测试断言 raw 文本不含 `secretKey` / `accessKey`）；上传前调 `multipartParts` 对齐服务端真实清单 → 跳过已传段、只补缺段、每段完成即落记录；清单为空 → abort 旧会话并重 init；查询抛错（uploadId 失效）→ 清记录干净重 init；账号 / 桶 / key 变化 → 不复用；complete 或明确中止 → 清记录 |
| 4 | 下载并行分段 [`api/download.ts`](../apps/web/src/api/download.ts) | 已知 size ≥16MB → **4 路并发 × 4MB Range GET**（经既有 `/proxy`，带 Bearer；有界并发，ADR-009）；逐段校验 206 + 字节数 === 请求区间 + `Content-Range`（若有）一致 → 按段号索引顺序聚合；FSA 可用则逐段写盘（写失败 abort 后抛错），否则 objectURL；服务端忽略 Range（回 200）、size 未知、<16MB → **回退单流**；任何校验失败即抛错，**不落损坏文件** |

> **TDD**：`multipart_listparts_test.go` 先写 → 编译红（`c.ListParts undefined`）→ 实现后 `-run TestListParts` 5/5 PASS；
> handler 测试先于实现编写（首次执行被并发 teammate 的瞬时编译错误阻断，未能观察到属于本条的编译红，如实记录）。
>
> **门禁实跑（2026-10-08，全部本机实跑）**：`go test ./internal/s3wrap/ -cover` **100.0%**（`ListParts` 100.0%）、
> `go test ./internal/handler/ -cover` **100.0%**（`multipart_parts.go` 100.0%，137s 全量）；`go vet ./...` 0 告警、
> `golangci-lint run` **0 issues**；前端 `pnpm test` **78 文件 / 1189 例**、`pnpm test:coverage` 四指标 **100%**
> （4481 / 3007 / 1151 / 3853）、`pnpm build` OK（**381.41 kB / gzip 116.15 kB**）、`pnpm lint` 0 告警；
> 真实对端 `S3CLIENT_E2E=1 go test ./internal/s3wrap/ -run TestE2E` **4/4 PASS**
> （`rustfs/rustfs:1.0.0-rc.3` @ `127.0.0.1:9000`，含 12 MiB multipart 组装）、`make e2e-real` **3 passed**、
> `pnpm e2e` **22 passed**。
>
> **残留（已登记，非缺陷）**：真实对端 E2E 未新增 `ListParts` 断言（既有 E2E 只覆盖 multipart 组装）；
> `ListParts` 由 fake 测试 100% 覆盖。

### §BR 2026-10-08 ROADMAP #11 零依赖 OTLP tracing（W3C traceparent + OTLP/HTTP JSON，默认关闭）

> 来源：ROADMAP §三 3.2 **#11**（原 `KNOWN_ISSUES` #54，本条落地后移出转空号）。人类拍板
> **零新增依赖、自研最小 tracer**，故**不**引入官方 `opentelemetry-go`（替代方案与取舍见
> [ADR-013](decisions/0013-zero-dep-otlp-tracing.md)）。

| # | 交付 | 关键取舍 / 口径 |
|---|---|---|
| 1 | 新包 [`internal/tracing`](../apps/server/internal/tracing/) | 标准库实现，`require` 段零新增依赖；W3C `traceparent`（`00-<32hex>-<16hex>-<2hex>`）解析 / 生成——非 `00` 版本、字段数 / 长度不符、全 0 一律**不继承、新建**；入站 `sampled=1` 即使 `SampleRatio=0` 也继承采样，`sampled=0` 不重采样（head sampling，无尾部采样） |
| 2 | OTLP/HTTP JSON 导出 | `POST {Endpoint}/v1/traces`，`resourceSpans→scopeSpans→spans`（resource attribute `service.name`）；**有界队列 512 + 批 64 + 1s ticker + `Close` 刷出**；导出失败 / 队列满只 WARN + 计数，**绝不影响请求路径**；`Endpoint` 为空 = 禁用（透传、不注入响应头、零开销） |
| 3 | 接线 | [`main.go`](../apps/server/main.go) 用中间件包住 `h.Routes()`（含 fail-closed 的初始化失败拒绝启动、退出前 5s `Close`）；`presign` / `proxy` / `migrate` / `migrate/sync` / `migrate/async` 打子 span，server span 带既有 `X-Request-ID` 关联属性 `request.id` |
| 4 | 配置与文档 | `S3C_OTEL_ENDPOINT`（默认空 = 关）、`S3C_OTEL_SAMPLE_RATIO`（默认 1，仅 [0,1]，NaN / 越界 / 非数字 → `envErr` → `Validate()` **拒绝启动**）、`S3C_OTEL_SERVICE_NAME`（默认 `s3client`）；`docs/OPERATIONS.md` §3.4 写开启方式 / 采集端期望 / 失败模式 |

> **TDD**：先写 3 个测试文件 → 红（`undefined: Config/Tracer/span`，build failed）→ 实现后绿、100%；
> config 三例先红（`cfg.OTelEndpoint undefined`）后绿。接线用例（`handler/tracing_wiring_test.go` +
> `TestRunServerTracingInitFails`）为接线后补写，如实记录。
>
> **门禁实跑（2026-10-08）**：`go test ./internal/tracing/ -cover` **100.0%**（22 例，含 httptest 假 collector
> 报文断言与 traceparent 表驱动）、`go test ./internal/config/ -cover` **100.0%**；`go vet ./...` 0 告警、
> `golangci-lint run` **0 issues**、`gofmt -l` 干净；`make test-cover` **10/10 包 100.0%**（新增本包为第 10 包，
> profile `count==0` 零块）。**变异验证**：把 `parseTraceparent` 的 `len(parts) != 4` 改成 `< 4` →
> `TestParseTraceparent` 精确点名「多余字段」红灯，还原后复跑绿。
>
> **残留（已登记，非缺陷）**：只实现 OTLP 字段子集（无 events / links / span status，resource 仅 `service.name`，
> 仅 HTTP/JSON 无 gRPC / protobuf）；队列 / 批量参数是代码内变量（无 env）；`Close` 5s 窗口内采集端不可用会丢
> 最后一批；handler panic 时该请求 span 不导出（无 recover，保持既有行为）；新增响应头 `traceparent` 仅启用时注入，
> 未加入 CORS `Access-Control-Expose-Headers`（浏览器 JS 默认读不到，服务端到服务端 / gateway 可读）。

### §BS 2026-10-08 ROADMAP #13 Token 作用域与最小权限（`S3C_TOKEN_SCOPES`）

> 来源：ROADMAP §三 3.2 **#13**（原 `KNOWN_ISSUES` #56，本条落地后移出转空号）。人类拍板
> **新增独立 env `S3C_TOKEN_SCOPES`，`S3C_TOKEN` 语义完全不变**（未登记 token 仍为全权，向后兼容）。

| # | 交付 | 关键取舍 / 口径 |
|---|---|---|
| 1 | [`config/scopes.go`](../apps/server/internal/config/scopes.go) | `TokenScope{Readonly, Prefixes, Accounts, ExpiresAt}` + `Config.ScopeFor`；严格 JSON（`DisallowUnknownFields`）；token 必须在 `S3C_TOKEN` 逗号列表内、`prefixes` / `accounts` 元素非空、`expiresAt` 为 RFC3339、`prefixes` 不得以 `/` 开头；任一非法 → `Validate()` 返回 `ErrInvalidTokenScopes` **拒绝启动**（fail-closed，文案不含 token 明文） |
| 2 | [`handler/scope.go`](../apps/server/internal/handler/scope.go) + `withAuth` | 仍走 `secureCompare` 常量时间比较；命中后按 scope 判定：`readonly` 仅放行 GET / HEAD（**预签名 POST 同样 403**——它能铸造写 URL）；`prefixes` 对 query 与 JSON body 的桶 / 键**各自校验（去重）**，列表 `prefix` 同判，桶级操作需整桶授权，body 非法 JSON / 读失败 / >16MB（无法判定）→ fail-closed 403；`accounts` 限路径 `{id}`；`expiresAt` 过期 → 401（`token_expired`）；越权 → 403 + 审计 `auth.scope_denied`（reason = readonly / prefix / account / unparsable_body，**不含 token 明文**） |
| 3 | 契约与文档 | OpenAPI `bearerAuth.description` 补作用域语义（`docs/api/openapi.json` 由统一重生成刷新）；`docs/CONFIGURATION.md`（SSOT 行含 JSON 示例 + fail-closed 行）、`docs/api.md`（403 语义）、`docs/threat-model.md`（最小权限段 + 遗留项）、`apps/server/.env.example` |

> **TDD**：先写 `config/scopes_test.go` → 编译红（`cfg.ScopeFor undefined` / `undefined: ErrInvalidTokenScopes`）→
> 实现后绿；再先写 `handler/scope_test.go` → 编译红 → 实现后 21 组用例转绿。
>
> **自审抓到并修掉的两个真问题**：① `newBucket==""` 时 `newKey` 既并入主桶又被当成空桶独立引用 → 误判 403；
> ② query `bucket` 原先优先于 body `bucket`，只在一处塞越界桶即可绕过 → 改为两处桶引用**各自校验**并补三态用例。
>
> **门禁实跑（2026-10-08）**：`go test ./internal/config/ -cover` **100.0%**、`go test ./internal/handler/ -cover`
> **100.0%**（`scope.go` 全函数 100.0%，`count==0` 零块）；`go vet ./...` 0 告警、`golangci-lint run` **0 issues**。
> **变异验证**：摘掉 `readonly` 判定 → 4 个写方法断言红灯（`POST presign = 200, want 403` / `PUT = 400, want 403` /
> `DELETE = 200, want 403` / `PATCH = 404, want 403`）；摘掉「token 必须已登记」校验 → 2 条配置用例红；均已还原绿。
> 契约：重生成后 `TestCommittedOpenAPISpecMatchesRuntime` 绿（`bearerAuth.description` 入提交版 spec）。
>
> **残留（有意，已写入 [`threat-model.md`](threat-model.md)）**：作用域是 **token 级粗粒度**而非 S3 IAM——
> `prefixes` 只约束请求显式给出的桶 / 键；桶列表、账号列表、`lifecycle` 规则内的 per-rule prefix、
> `preview-buckets`（body 自带凭证）不在约束内。配了 `prefixes` 的 token 其请求体在鉴权层按 `maxBody`(16MB)
> 缓存后原样还原，故超限对这类 token 返回 403 而非 413；`defaultBucket` 对每个未解析桶引用各做一次
> `store.Get`（量级小，未做 per-request 缓存）。

### §BT 2026-10-08 ROADMAP #5 S3 新协议特性：条件写 / 端到端校验和 / Object Lock

> 来源：ROADMAP §三 3.2 **#5**（原 `KNOWN_ISSUES` #48，本条落地后整行移出转空号）。开工前三条
> 口径由人类拍板：**全栈含最小 UI**、条件写覆盖**直传预签名 + 服务端写路径**、**不提交 git**
> （改动留工作树待人工审阅；同分支另有并行代理推进 #6 计划任务 / #7 FinOps，端点计数与共享
> 产物（`routes.go` / `docs/api/openapi.json` / 计数声明）按各自批次滚动更新，收口时点为 **83**）。

| # | 交付 | 关键取舍 / 口径 |
|---|---|---|
| 1 | 条件写·s3wrap 边界 [`s3wrap/conditions.go`](../apps/server/internal/s3wrap/conditions.go) + [`object.go`](../apps/server/internal/s3wrap/object.go) + [`presign.go`](../apps/server/internal/s3wrap/presign.go) | `Conditions{IfMatch,IfNoneMatch}` + `PutObjectCond` / `CopyObjectCond`（`CopyOptions` 含目标端条件与校验和算法）/ `DeleteObjectCond`；`PresignPut` 改返 `PresignPutResult{URL,Headers}`（**失败也返非 nil 空结果**，handler 无需错误分支）；`PreconditionFailed→412`、`ConditionalRequestConflict→409` 入 `errors.go` 映射表 |
| 2 | 条件写·handler 边界 [`handler/conditions.go`](../apps/server/internal/handler/conditions.go) | `presign(put)` / `mkdir` / `copy-object` 三处字段 + 统一边界校验：`ifNoneMatch` 仅 `*`（map 字面量供 enum 契约机械抽取）、`ifMatch` 可见 ASCII ≤512（CRLF/空格在边界拦截，错误文案固定不回显输入）；`get`/`post` 携带条件显式 400；预签名 put 响应 `headers` 回显——RustFS 实测条件头**参与签名**（`X-Amz-SignedHeaders=host;if-none-match`），浏览器直传必须原样携带 |
| 3 | 端到端校验和·读侧与验证 [`s3wrap/checksum.go`](../apps/server/internal/s3wrap/checksum.go) | `head` 响应新增 `checksums`（CRC64NVME / CRC32C / SHA256 / SHA1 + `FULL_OBJECT\|COMPOSITE_*`，无则 `null`）；新端点 `POST /api/accounts/{id}/verify-checksum`：Head 选阶梯 **crc64nvme → crc32c → sha256 → sha1 → etag-md5** → 流式 GET 本地重算 → 逐字节比对；合成校验和（类型前缀 / 值内 `-` 后缀）跳过、无来源 `method="none"` 如实降级 |
| 4 | 自研流式 CRC-64/NVME [`s3wrap/crc64nvme.go`](../apps/server/internal/s3wrap/crc64nvme.go) | poly `0xad93d23594c93659` 反射式 / init=xorout=全 1，查表实现（`hash.Hash`+`Hash64`）；**三方对齐**：CRC RevEng check 向量 `0xAE8B14860A799888`、Python 独立实现逐值比对、真实 RustFS 存储值 `"N4bktbEKNg8="`（E2E 绑定断言）；**零新增依赖**（`go.mod` require 段零变化，许可证清单无需重生成） |
| 5 | 端到端校验和·写侧物化 | `copy-object` 可选 `checksumAlgorithm`（CRC64NVME / SHA256 / CRC32C / SHA1，map 字面量 enum 契约）→ `CopyOptions.ChecksumAlgorithm` → S3 复制时计算并存储全对象校验和（RustFS 实测：复制后 `head` 可读、`verify` match）；批量复制 / 迁移与预签名直传不物化（见残留） |
| 6 | Object Lock [`s3wrap/objectlock.go`](../apps/server/internal/s3wrap/objectlock.go) + [`handler/objectlock.go`](../apps/server/internal/handler/objectlock.go) | 6 端点：`GET/PUT bucket/object-lock`（默认保留策略，天/年二选一 + 模式必填）、`GET/PUT object-retention`（RFC3339 未来时刻）、`GET/PUT object-legal-hold`（ON/OFF）；读侧降级链：`ObjectLockConfigurationNotFoundError` / `InvalidRequest`（RustFS 非锁定桶实测码）→ `enabled/configured=false`、`LegalHoldNotFoundError` → `OFF`、**空 `ObjectRetention` 元素整体归一为未设置**；错误映射 `ObjectLocked→409`、`RetentionPeriodTooShort→400`、`ObjectLockConfigurationNotFoundError→400`、`NotImplemented→501`、`InvalidBucketState→409`（handler 特判「只能建桶时启用」固定文案） |
| 7 | 前端最小 UI [`apps/web/src/`](../apps/web/src/) | 上传「仅当对象不存在时创建（If-None-Match: \*）」（presign `ifNoneMatch:"*"` → 响应 `headers` 逐个上 XHR；412 文案「上传被拒绝：目标对象已存在」；≥100MB 分段路径前置拒绝并提示）；对象详情（`ObjectDetailDialog`）显示校验和 + 「校验校验和」（method/本地值/远端值/一致与否）+ 保留期设置 + 法定保留开关（PUT 真实接线）；桶设置新增 **Object Lock 页签**（`BucketObjectLock.vue`，状态 + 默认保留策略编辑，未建时启用显示固定提示）；`endpoints.ts` 新增 7 方法 + presign/mkdir/copy 条件字段；i18n zh+en 全键 |
| 8 | 契约与文档 | OpenAPI 注册表 **+7 operation**（`getObjectLock`/`putObjectLock`/`getObjectRetention`/`putObjectRetention`/`getObjectLegalHold`/`putObjectLegalHold`/`verifyChecksum`）+ 3 处字段扩展（presign / mkdir / copy-object 条件与校验和）+ 7 份 examples（根门禁机械校验「字段 ∈ schema、required 齐全、enum 吻合」）；提交版 `docs/api/openapi.json` 重生成、前端 `schema.d.ts` / `operations.ts` 经 `pnpm gen:api` 重生成；`api.md` / `errors.md` / `compatibility.md` §6.1 第 4 条（含 RustFS 实测差异）/ `CHANGELOG` / 计数声明同步 |

> **TDD**：三件套均「先红后绿」——s3wrap 条件写 / 校验和 / Object Lock 先落测试（编译红：`undefined: PresignPutResult`
> 等）→ 实现转绿；handler 层同样先写 `conditional_write_test.go` / `objectlock_api_test.go` / `checksum_api_test.go`
> 再实现；E2E 探测先行（临时 probe 实测 RustFS 能力矩阵后删除，结论固化进正式 `e2e_features_test.go`）。
>
> **自审抓到并修掉的真问题**：① `CopyObject` 线格式实为 **PUT + x-amz-copy-source**（测试按 SDK 真值修正）；
> ② RustFS 对「锁桶无保留对象」返回**空 `ObjectRetention` 元素**——不归一会让 `configured=true` 却无字段；
> ③ `cleanupBucket` 曾**无界循环**挂死 10 分钟（对象被保留期拒删），加 100 轮上限 + 收敛报错；
> ④ 预签名条件头**参与签名**（必须回显，漏传会 403）；⑤ GOVERNANCE 保留期内**版本删除**被 403 强制
> （首轮探测只看 err 未复查列表，误判为「不强制」，`compatibility.md` 已据实修正）；
> ⑥ **跨批次规范抖动（收口时抓出并修复）**：同日并行 #6 的 `POST /api/schedules` 与
> `PUT /api/schedules/{id}` **共享同一个 `*openapi.Request` 指针**，`applyExamples` 遍历 `apiExamples`
> （Go map，顺序按进程随机）经指针写示例 → 两 operation 的请求示例互相覆盖，提交版规范门禁
> `TestCommittedOpenAPISpecMatchesRuntime` **10 次里 7 次随机红**（语义相同、字节抖动，循环复跑定位）。
> 修复：`openapi_register_schedules.go` 每次注册构造**独立** `Request`（附共享指针禁忌注释），
> 修复后连跑 **8/8 稳定通过**——属并行批次文件上的**一行级跨范围修复**，PR 描述已明示。
>
> **门禁实跑（2026-10-08 晚，终局合并态）**：`make test-cover` **10/10 包 100.0%、`count==0` 零块**
> （含同日并行 #6 的 `schedules` 全量）；`gofmt -l` 干净 / `go vet ./...` 0 告警 /
> `golangci-lint run ./...` **0 issues**（本批 gosec 弱哈希 4 处 + G115 截断 1 处按「仅比对 S3 存储
> 校验和、非安全用途」带理由登记，nolintlint 全过）；`go test ./...` 10/10 包；包根文档门禁
> （`doc_number` / `doc_link` / `en_docs` / `deadcode` / `examples`）全绿；`pnpm e2e` **22 passed**。
>
> **前端实跑（2026-10-08 晚，与 #6 SchedulesSection 合并态）**：`pnpm lint` 0 告警、
> `pnpm typecheck` / `pnpm typecheck:e2e` exit 0、`pnpm test` **80 文件 / 1254 例全绿**、
> `pnpm test:coverage` **四指标 100%**（4770 语句 / 3185 分支 / 1217 函数 / 4128 行）、
> `pnpm build` OK（408.09 kB，gzip 122.74 kB）、`pnpm gen:api --check` exit 0（83 操作）；
> **`make e2e-real` 3 passed**（真实浏览器 → 真实后端 + `S3C_TOKEN` → 真实 RustFS
> 账号/建桶/预签名直传回读）。
>
> **真对端 E2E（RustFS 1.0.0-rc.3，`S3CLIENT_E2E=1 -run TestE2E`）**：**7/7 PASS**——新增
> `TestE2EConditionalWrite`（服务端 Put/Copy/Delete 条件 + 预签名条件头全链路）、`TestE2EChecksumVerify`
> （etag-md5 回退、CRC64NVME/SHA256 物化、绑定已知向量的 match 断言）、`TestE2EObjectLock`
> （配置/保留/法定保留读写 + 版本删除 403 强制 + 非锁定桶降级），既有 4 条回归全绿。
>
> **残留（有意，见 `compatibility.md` §6.1 第 4 条与 `api.md` 注记）**：① 预签名 **POST 表单**不支持条件
> （S3 POST policy 无条件头，UI 对 ≥100MB 分段路径不提供条件写）；② 批量复制 / 迁移不物化校验和
> （`checksumAlgorithm` 仅单文件 `copy-object`，避免改变存量迁移行为）；③ RustFS 差异：保留期一经设置
> 不可修改（405）、GOVERNANCE→COMPLIANCE 升级被拒（`AccessDenied`）——以 AWS 语义为准的行为不在
> RustFS 断言；④ 对象保留期编辑 UI 只支持新增 / 延长语义（缩短由服务端按厂商规则裁决）；
> ⑤ mkdir 条件写仅到 API 层（`endpoints.ts` `mkdirObject` 类型含 `ifNoneMatch`，UI 走共享
> `promptDialog` 单值契约未动，调用只传 `bucket/key`）；⑥ 复制对话框条件字段仅到类型层
> （`DestDialog.vue` 不传 `ifMatch/ifNoneMatch`，`checksumAlgorithm` 不在前端 endpoints 类型里）。

### §BU 2026-10-08 ROADMAP #6 计划任务：cron 定时增量备份

> 来源：ROADMAP §三 3.2 **#6**（原 `KNOWN_ISSUES` #49，本条落地后整行移出转空号）。
> 同批并行落地为 #5（S3 新协议特性）与 #7（FinOps 存储分析与成本看板）——共享产物（`routes.go` /
> `docs/api/openapi.json` / 端点计数声明）按各自批次滚动更新，**收口合计 84 个 `/api/*` 端点**
> （71 基线 + 5 #6 + 7 #5 + 1 #7）。
> 口径：**全栈含最小 UI**、**零新增依赖**（自研 5 字段 cron 解析，`go.mod` require 段零变化）、
> **不提交 git**（改动留工作树待人工审阅）。

| # | 交付 | 关键取舍 / 口径 |
|---|---|---|
| 1 | 自研 cron 解析 [`service/schedule_cron.go`](../apps/server/internal/service/schedule_cron.go) | 标准 5 字段（分 时 日 月 周）、**仅数字**（`MON`/`JAN` 名字显式拒收，文档写明）；`a/s` 步进 = `a..max`、升序区间、**40 年视界**拒绝永不触发的表达式（`0 0 30 2 *` → 400）；dom/dow 双受限时按 **Vixie OR 语义**（`0 0 15 * 0` = 15 号**或**周日）——19 例表驱动 + 模糊测试钉住 |
| 2 | 计划模型与校验 [`service/schedule.go`](../apps/server/internal/service/schedule.go) | `Schedule` 15 字段（含运行态 `lastRunAt/lastJobId/lastError`，`omitempty/omitzero`）；`Validate` 点名字段错误（API 400 直接回传）；`mode` 空 = etag（与 sync 同口径） |
| 3 | 落盘 [`service/schedule_persist.go`](../apps/server/internal/service/schedule_persist.go) | `schedules.json` 0600 **原子写**（`atomicfile`，同 `jobs.json` 口径），损坏文件静默降级空清单（计划是辅助信息，不阻塞启动）；`SchedulePersister` 接口 + `FileSchedulePersister`，setter 注入（同 `SetJobPersister` 模式） |
| 4 | 调度器 [`service/scheduler.go`](../apps/server/internal/service/scheduler.go) | 30s 评估循环（cron 粒度 1 分钟 → 到点 ≤60s 内触发）；三条语义**各有测试钉住**：① **补跑一次**——停机错过的多个槽只触发一次并直接排到未来（防恢复瞬间洪水）；② **不叠加**——trigger 返回 `ErrScheduleRunning` 时跳过本轮、不覆盖运行态、排期照常前移；③ **失败即前移**——记录 `lastError` 后同样前移（坏配置不会每 tick 重试轰炸），成功或更新计划时清除；手动 `RunNow` 不改自动排期 |
| 5 | handler 5 端点 [`handler/schedules.go`](../apps/server/internal/handler/schedules.go) | `GET/POST /api/schedules`、`PUT/DELETE /api/schedules/{id}`、`POST /api/schedules/{id}/run`；错误映射：cron/mode/桶/配置 → 400、计划或账号不存在 → 404、上一轮未结束 → **409**、在册任务满 → 503、store 故障 → 500（仅 `ErrNotFound` 才 404，与 migrate 同修复口径）；**防叠加 check-and-set 在同一把锁内**完成（手动 run 与调度 tick 并发也只开一轮）；触发复用 `JobRegistry` 异步任务（进度 / SSE / 取消 / 重启恢复与迁移任务同链路），`total=0` 由 SyncKeys 首帧学习；**列举失败与截断均记入 `result.lastError`**（不静默报「无事可做」/「已全部完成」） |
| 6 | 作用域 [`handler/scope.go`](../apps/server/internal/handler/scope.go) | `prefixes` 令牌对计划 `run`/`DELETE`（无请求体）按**已存计划的桶/前缀**注入引用判定（越界计划不可触发/删除）；`POST` 建计划走既有 body refs；`readonly` 拦全部写方法；GET 列表不带桶引用 → 放行（与桶列表同类残留，threat-model 口径） |
| 7 | 前端 [`SchedulesSection.vue`](../apps/web/src/components/SchedulesSection.vue) | 挂迁移面板（计划自带源/目标引用，不依赖当前选中账号）：列表 / 新建 / 编辑 / 启停 / 立即运行 / 删除；cron 与账号校验错误行内回显（`role=alert`）；i18n 新增 `schedule.*` 中英双语键（字典一致性 + 死键门禁全绿）；`endpoints.ts` +5 方法（路径/method 取自 `gen:api` 生成物） |
| 8 | 契约与文档 | OpenAPI 注册表 **+5 operation**（`listSchedules`/`createSchedule`/`updateSchedule`/`deleteSchedule`/`runScheduleNow`）+ 新 tag `schedules` + 7 份 examples（根门禁机械校验）；`mode` enum 双站点登记 `enumContracts`（handler switch 机械抽取双向钉住）；提交版 `openapi.json` 与前端 `schema.d.ts` / `operations.ts` 重生成；`api.md`（5 路由 + 请求体字段双向比对 + curl 示例）/ `user-guide.md` §九 / `CONFIGURATION` / `OPERATIONS` / `CHANGELOG` / 计数声明同步 |

> **TDD**：service 层先红后绿——cron 解析测试先行（19 例 + fuzz）→ 实现；handler 层先写
> `schedules_test.go`（编译红：`h.SetSchedulePersister undefined`）→ 接线转绿；覆盖缺口逐行
> lcov 映射补齐（store 故障 500、目标侧配置/桶、空 mode 归一、列举失败、截断、trigger 双侧
> client 失败等 11 处分支各有具名用例）。
>
> **自审抓到并修掉的真问题**：① `Cron.Expr()` 只被测试引用 → 违反死代码零容忍，连同只写不读的
> `expr` 字段一并删除（回显以 `Schedule.Cron` 为唯一来源）；② 消音式死代码（`_ = trig` 等 4 处）
> 与两个失去引用的测试 helper → 删除；③ `SetSchedulePersister`/`NewFileSchedulePersister` 缺
> `main.go` 生产接线 → 补齐（deadcode 门禁抓出）；④ 409 测试依赖列举完成时序，`-race` 并行下
> SDK 重试拖过 5s 超时 → 收尾等待放宽至 30s（仅失败路径等待）；⑤ `SchedulesSection` 首次挂载位置
> 插进了 `v-if`/`v-else` 之间导致模板编译崩 → 移到 `</template>` 之后。
>
> **门禁实跑（2026-10-08 晚）**：`go vet ./...` 0 告警；`golangci-lint run` **0 issues**；
> `make test-cover` **10/10 包 100.0%（`count==0` 零块）**；包根门禁（`doc_number` / `doc_link` /
> `en_docs` / `deadcode` / `examples` / `changelog_tag`）全绿；`apiRouteCount` 78 → **83**。
> **前端实跑**：`pnpm lint` 0 告警、`pnpm typecheck` / `pnpm typecheck:e2e` exit 0、
> `pnpm test` **80 文件 / 1254 例全绿**、`pnpm test:coverage` **四指标 100%**（4770 语句 /
> 3185 分支 / 1217 函数 / 4128 行）、`pnpm build` OK（408.09 kB，gzip 122.74 kB）、
> `pnpm gen:api --check` exit 0（83 操作）、`pnpm e2e` **22 passed**；`make e2e-real`
> **3 passed**（脚本自带 RustFS，实跑端口 `RUSTFS_PORT=9006 SERVER_PORT=18090`——默认
> 9000/8080 被常驻开发容器与 haproxy 占用）。
>
> **真对端 E2E（`S3CLIENT_E2E=1 -run TestE2E`）：7/7 PASS**——计划任务不改 s3wrap，复制内核
> 与既有回归一并复跑确认无副作用。
>
> **残留（有意）**：① GET 计划列表不携带桶引用 → `prefixes` 令牌可**读**到越界计划的桶名
> （与桶列表同类残留，不可触发/删除——写路径已注入判定）；② 账号作用域仍按路径 `{id}` 判定，
> 计划 body 的 `sourceAccountId/targetAccountId` 不受 accounts 约束（与 migrate 同口径）；
> ③ 一次性增量同步仍只有 API 入口（计划任务是周期性入口；单次手动同步见 `api.md`）；
> ④ cron 仅数字 5 字段（不支持 `MON`/`JAN` 名字与 `L`/`W`/`#` 扩展）——文档已写明。

### §BV 2026-10-08 ROADMAP #7 FinOps 存储分析与成本看板

> 来源：ROADMAP §三 3.2 **#7**（原 `KNOWN_ISSUES` #50，本条落地后移出转空号）。交付人拍板范围：
> **全栈**——后端聚合端点 + OpenAPI 契约 + 前端独立顶层「成本看板」Tab + 文档同步。设计约束沿用
> 既有口径：**零新增依赖**、聚合纯函数下沉 `service`、列举三层硬上限（100 页 / 100k 对象 / token
> 不前进即停）与 `sync` / `delete-prefix` 同形、AWS SDK 类型不越过 `s3wrap`。

| # | 交付 | 关键取舍 / 口径 |
|---|---|---|
| 1 | [`handler/storage_report.go`](../apps/server/internal/handler/storage_report.go) | `GET /api/accounts/{id}/storage-report?bucket=&prefix=`（**端点总数 83 → 84**）；缺桶 400 / 未知账号 404 / S3 错误按既有映射；100 页 / 10 万对象硬上限，超额或多页游标异常置 `truncated=true`（不静默截断） |
| 2 | [`service/storage_report.go`](../apps/server/internal/service/storage_report.go) | 纯聚合：按存储类与「列举前缀下首层前缀」聚合用量与 `USD/GiB/月` 成本；未知存储类回退 STANDARD 价；年龄 30–89 天建议下沉 `STANDARD_IA`、≥90 天建议归档 `GLACIER_IR`（按 kind+源类+目标类聚合、确定排序）；定价与阈值是常量表 |
| 3 | 契约 | [`openapi_register_storage_report.go`](../apps/server/internal/handler/openapi_register_storage_report.go)（响应 10 字段 + 嵌套三数组）+ 示例；`docs/api/openapi.json` 统一重生成（61 → 62 paths / 83 → 84 operations） |
| 4 | 前端 | 新增顶层 `finops` Tab：[`StorageReportPanel.vue`](../apps/web/src/components/StorageReportPanel.vue)（选桶 + 前缀 → 报告卡片 / 存储类表 / 前缀表 / 建议表）+ [`storageReport.ts`](../apps/web/src/storageReport.ts) 纯格式化 + i18n 双语 28 键；类型派生自 spec（`types.ts` 的 `operations['storageReport']`） |
| 5 | 文档 | 本 §BV、[`ROADMAP.md`](ROADMAP.md) 3.2 行移出、[`CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]`、[`api.md`](api.md)、[`user-guide.md`](user-guide.md)、[`architecture.md`](architecture.md) 计数、[`README.md`](../README.md) 与 [`docs/en/index.md`](en/index.md) 计数 |

> **TDD**：先落地人类侧红灯用例 [`storage_report_test.go`](../apps/server/internal/handler/storage_report_test.go)
> （6 组：聚合 + 估算 + 建议 / 前缀透传分组 / 空桶 / 错误分支 / token 停 / 页数顶 / 100k 上限），
> 再写实现转绿；补 `storage_report_extra_test.go`（单页超 10 万的对象触发页内裁断）与
> `storage_report_cancel_test.go`（进入 handler 前 ctx 已取消）补齐全部分支。
>
> **门禁实跑（2026-10-08）**：后端 `go test -count=1 ./...` **9/9 包全绿**、逐包 **100.0% statements**
> （`awk '$NF==0'` 零未覆盖块，含 `openapi` 包——number schema 由 handler 侧局部构造，不向基础包
> 添加无独立测试面覆盖的导出符号）；`go vet ./...` 干净、`golangci-lint run ./...` **0 issues**、
> `gofmt -l` 干净。前端 `pnpm test:coverage` **80 文件 / 1208 例全绿、四指标 100%**
> （`StorageReportPanel.vue` / `storageReport.ts` / `endpoints.ts` 均 100%）、`pnpm build` + `pnpm lint`
> 干净、`pnpm gen:api --check` exit 0。README 截图重生成（`docs/images/*.png` 2 张）。
> 契约门禁（`doc_number_gate` / `openapi_examples_gate` / `api_doc` / `en_docs_gate` /
> `changelog_tag_gate` 等）全绿。
>
> **残留（有意）**：成本为公开牌价的量级估算（非账单真值，页内已声明）；聚合基于**当前版本**对象
> （不含历史版本 / 未完成分段的存储占用）；`byPrefix` 只到列举前缀下的首层，更细粒度需多次请求。

### §BW 2026-10-09 Go 工具链 1.26.6 → 1.26.9（10 个可达 stdlib 漏洞归零）+ e2e-real 用例幂等化

> 来源：质量画像评审复跑门禁发现——`govulncheck ./...` 在 go1.26.6 上 **10 个可达** stdlib 漏洞
> （GO-2026-6617 等，net/http / net/textproto / crypto/tls，均 go1.26.9 修复，调用链含
> `service.WriteObjectsZip → io.WriteString → http.response.WriteString`）；`make e2e-real --no-rustfs`
> 复用长期 RustFS（残留桶在场）时「建桶 → 列桶」红灯。与 [`../CHANGELOG.md`](../CHANGELOG.md)
> `[Unreleased]` 同日条目同一 commit。

| # | 项 | 改动 |
|---|---|---|
| 1 | Go 工具链 **1.26.6 → 1.26.9** | 按 ROADMAP E1「四处同步」口径：[`../apps/server/go.mod`](../apps/server/go.mod) / [`../apps/server/Dockerfile`](../apps/server/Dockerfile)（`golang:1.26.9-alpine`）/ [`../.gitlab-ci.yml`](../.gitlab-ci.yml) 两处（`golang:1.26.9-bookworm`）；GitHub 侧经 `go-version-file` 自动跟随。`govulncheck` **10 可达 → 0** |
| 2 | e2e-real 建桶断言（`e2e-real/real-backend.spec.ts`） | `createBucketViaUI` 先 `waitForResponse` 等建桶触发的 `GET /buckets` 落定 → 点「返回列表」→ 断言**列表行**。修两层：① 残留桶时 `BucketsPanel.loadBuckets` 在 selectedBucket 为空时自动钻进 `buckets[0]` 详情页，建桶后按行断言 15s 超时（共享实例实测红）；② 干净环境原断言命中的其实是**详情页概览行**而非列表行（假绿）。等待刷新同时消除「回列表被在途刷新重新钻走」竞态 |
| 3 | e2e-real 清桶幂等 | `cleanupBucket` 改「列对象 → 批量 `POST /delete` → `DELETE /bucket`」：原 `delete-prefix` 空前缀被 handler 有意拒绝（400「拒绝空前缀以免误删全桶」，`objects.go`），非空桶清不掉——直传用例每跑一次泄漏一个桶。清理失败仍仅 `console.warn`，不掩盖用例结论 |
| 4 | 本文件去重 | 上个提交 e965e54 把「三、质量与覆盖率现状」整块复制（109 行逐行重复，首份仅尾部少 `### 已知边界与取舍`）——删首份、保留含完整尾节的第二份 |

> **门禁实跑（2026-10-09，全部本机实跑）**：
>
> | 门禁 | 结果 |
> |---|---|
> | `go vet ./... && go build ./... && gofmt -l .` | 0 告警 / OK / 干净（go1.26.9） |
> | `go test ./... -count=1` | **10/10 包 ok** |
> | `make test-cover`（`-race`） | **10/10 包 100.0% statements** + profile `count==0` 零块 |
> | `golangci-lint run ./...` | **0 issues** |
> | `govulncheck ./...` | **0 可达漏洞**（修复前 **10**） |
> | `pnpm lint` / `pnpm typecheck` / `pnpm typecheck:e2e` | 0 告警 / exit 0 / exit 0 |
> | `pnpm test` | **82 文件 / 1272 例**全绿 |
> | `pnpm test:coverage` | **四指标 100%**（statements 4849 / branches 3227 / functions 1229 / lines 4195） |
> | `pnpm build` | OK（`index.js` 416.90 kB，gzip 124.98 kB；CSS 33.66 kB，gzip 7.05 kB） |
> | 共享脏 RustFS `--no-rustfs` 实跑 | 「建桶 → 列桶」**由红转绿**；全量 2 passed / 1 failed——第 3 条败于共享实例缺 `RUSTFS_CORS_ALLOWED_ORIGINS`（`scripts/e2e-real.sh` 注明必需的环境配置缺口，非代码问题） |
> | 洁净实例 + **预埋残留桶**（legacy-a 空桶 / legacy-b 非空桶）全量 | **3 passed（10.4s）**；跑后仅剩预埋桶——用例桶（含非空直传桶）**0 泄漏**，③ 的清理修复实证 |
> | `make e2e-real`（自管容器规范路径） | **3 passed（9.5s）** |

> **环境注记**：本机经模块代理拉工具链的路径长期卡死（mod cache 内 1.26.8 半截 tmp 自 09-16 停滞），
> 本次改由 `dl.google.com` 直下 tarball（SHA256 与官方一致）装 `/usr/local/go1.26.9`，门禁均以
> `PATH=/usr/local/go1.26.9/bin:$PATH` 执行；CI 侧 `setup-go` 读 `go-version-file`、镜像按 tag 拉取，不受影响。

> **文档同步**：[`../CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]` 同日条目、[`ROADMAP.md`](ROADMAP.md) §四
> govulncheck 行 + E1 行、[`threat-model.md`](threat-model.md) CI 门禁行；本节 + 头部摘要 + 目录行三处同步。

---

### §BX 2026-10-09 e2e-real `seedAccountAndBucket` 失败泄漏账号/桶修复（含建桶失败快诊 + 限速节流）

> 来源：评审批次之后**新发现**（不在 [`code-review-2026-10-09.md`](archive/code-review-2026-10-09.md) 内）——
> `e2e-real/real-backend.spec.ts` 两个用例把 `seedAccountAndBucket(...)` 放在 `try` **之外**：seed 在
> 建桶步骤抛错时不会返回，`finally` 根本不执行，账号/桶就留在真实后端与 RustFS 上（失败运行留垃圾的机制）。
> 与 [`../CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]` 同日条目同一 commit。

| # | 项 | 改动 |
|---|---|---|
| 1 | seed 失败泄漏（`e2e-real/real-backend.spec.ts`） | 新增调用方持有的资源登记簿 `SeededResources`：seed **边创建边登记**（`accId` / `bucket`）并移入 `try`，`finally` 统一走 `cleanupSeeded`——未创建项判空跳过；账号按**名称**清理，覆盖「`createAccountViaUI` 在返回 id 前失败、`accId` 未登记」的隐蔽泄漏面。两处用例 + 回归用例三处同改 |
| 2 | 建桶失败诊断（`createBucketViaUI`） | 原先只等「建桶成功才触发的 `GET /buckets`」，而该 `waitForResponse` 本套配置下**无 30s 上限**（trace 实测挂满 135s test timeout，报错行号指向后续无关语句）。现同时等 `POST /bucket`，非 2xx 立即抛「状态码 + 响应体摘要」 |
| 3 | 限速节流（4 用例后） | 套件秒级打出 60+ 个 `/api` 请求抽干令牌桶（`ratelimit.go`：120 req/min、突发 30）；页面自身的列表刷新不带退避重试 → 「直传」用例上传后列表刷新撞 429、行断言 15s 超时（可复现）。按回填速率（2 token/s）在用例间补 5s 间隔，令每个用例从接近满桶起步；**不关限速、不改后端** |
| 4 | 回归用例（新增第 4 条） | 「seed 中途失败不泄漏」：以非法桶名（含大写）让后端 400 → seed 抛错；先断言前提（账号确已落库，防失败点漂移导致空转），`finally` 清理后再断言账号表为空 |

> **门禁实跑（2026-10-09，本机实跑）**：
>
> | 门禁 | 结果 |
> |---|---|
> | `pnpm typecheck:e2e` / `pnpm lint` | exit 0 / 0 告警 |
> | `scripts/e2e-real.sh --skip-build`（自管 RustFS 容器 19011，后端 18111） | **4 passed（29.6s）** |
> | 回归用例**先红后绿** | 旧结构（seed 在 try 外）：`Expected length: 0, Received length: 1`（账号残留 1 个，2.2s）；修后通过 |
> | 无节流对照（诊断） | 4 用例无间隔时「直传」用例稳定复现 429 → 行断言超时；节流后两轮全绿 |

> **文档同步**：[`../CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]` 同日条目；本节 + 头部摘要 + 目录行三处同步。

### §BY 2026-10-09 s3wrap E2E 清桶强删路径（根治 Object Lock 锁定版本导致的共享实例残留桶泄漏）

> 来源：2026-10-08 共享 RustFS（`s3c-dev-rustfs`，持久卷）实测发现 3 个残留桶
> （`s3c-e2el-1791466318612938664` 带 GOVERNANCE 保留 + 法定保留、`s3c-probe-*` 两个）——
> 均为更早 s3wrap E2E / 能力探测跑批的遗留：被 Object Lock 锁住的版本删除被 403 拒绝时，
> `cleanupBucket` 只吞错不重试，空转 100 轮报「did not converge」后放弃，桶泄漏且跨容器重启存活。
> 与 [`../CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]` 同日条目同一 commit。

| # | 项 | 改动 |
|---|---|---|
| 1 | 强删路径（`e2e_test.go`） | 新增 `e2eForceDeleteVersion`：普通版本删除被拒 → `PutObjectLegalHold` OFF（非锁定桶上报错属预期，吞掉）→ 带 `x-amz-bypass-governance-retention` 头重删版本；`cleanupBucket` 版本循环接入。删没删掉仍由下一轮列表收敛判定；COMPLIANCE 保留不可绕过时照常「did not converge」报错。无锁桶路径不变（普通删除成功即不发强删请求） |
| 2 | Object Lock E2E 收尾（`e2e_features_test.go`） | `TestE2EObjectLock` defer 原「睡到 10s 保留窗到期再清」被强删路径取代——短窗只为断言「保留期内删不掉」，清理不再等；E2E 由 ≥11s 降至 **0.17s**，强删路径每次真跑 |
| 3 | 假 S3 回归测试（新增 `cleanup_fake_test.go`） | 不依赖 `S3CLIENT_E2E`，普通 `go test` 即钉两条外部行为：被锁版本「403 → 法保留 OFF → bypass 删除 → 删桶收敛」；无锁版本+删除标记「一轮清干净、零强删请求」 |

> **门禁实跑（2026-10-09，本机实跑，go1.26.9）**：
>
> | 门禁 | 结果 |
> |---|---|
> | 新增回归测试先红后绿 | 旧实现：`cleanup did not converge after 100 rounds: 1 versions, 0 markers remain`；修后 2/2 通过 |
> | `S3CLIENT_E2E=1 go test ./internal/s3wrap/ -run TestE2E -v`（真 RustFS） | **7/7 通过**（1.18s）；跑后 `ListBuckets` 复核 `[]`（零残留） |
> | `go vet ./... && go build ./... && gofmt -l .` | 干净 / OK / 无输出 |
> | `go test ./... -count=1` | **10/10 包 ok** |
> | `golangci-lint run` | **0 issues** |

> **文档同步**：[`../CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]` 同日条目；本节 + 头部摘要 + 目录行三处同步。

---

### §BZ 2026-10-09 全仓代码评审批次：C1–C2 + R1–R9 全修 + O1–O12 登记

> 来源：[`code-review-2026-10-09.md`](archive/code-review-2026-10-09.md)（2026-10-09 全仓五轴评审）——
> 2 Critical + 10 Required 当日全修、每项附行为级回归测试；12 条 Optional 登记入
> [`KNOWN_ISSUES.md`](KNOWN_ISSUES.md) **#72–#83**（全属技术债 / 缺陷，按「同一事项只登记一处」不进 ROADMAP）。
> 评审快照状态表与 §8 已回写；与 [`../CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]` 同日条目同一 commit。

| # | 项 | 改动 |
|---|---|---|
| C1 | CI Go job 补前端依赖前置 | GitHub `server` job 加 pnpm 9.15.0 / node 26.10.0（与 web job 同 pin + pnpm cache）+ `pnpm install --frozen-lockfile`；GitLab `server` 用 nodejs tarball + `npm install -g pnpm@9.15.0` + `.pnpm-store/` cache override——`agent_evals` 门禁判据前置是 `apps/web/node_modules` 存在（**不 `mkdir` 伪造**） |
| C2 | 深克隆 + CI 双侧一致性门禁 | GitHub 十处 checkout `fetch-depth: 0` + GitLab `GIT_DEPTH: "0"`（`changelog_tag` 从 `.git` 读 tag，浅克隆必红）；新增 `apps/server/ci_consistency_gate_test.go` 钉两侧 job / 命令 / 版本 pin 逐项一致 |
| R1（安全） | migrate jobs 端点级作用域闸 | `/api/migrate/jobs*` 对声明 `prefixes` / `accounts` 的 token 一律 403（审计 `reason=migrate_jobs`；`readonly` 读放行、POST 取消仍由方法闸拦）——堵住「前缀 token 枚举全量任务 / 读别桶 `failedKeys` / 取消他人迁移」 |
| R2（安全） | preview-buckets 端点级作用域闸 | 对声明 `readonly` / `prefixes` / `accounts` 任一的 token 一律 403（`reason=preview_buckets`，仅 `expiresAt` 不受影响）——自带 endpoint/凭据由服务端拨号的 SSRF 跳板收口；`accountIDFromPath` 特例死分支随之删除 |
| R3 | cron DST 绝对时间轴 + 本地字段回验 | `cronCandidates`：墙钟 UTC 锚 + Go 归一化落点 ±2h 探针偏移集合 + 逐一回验——春季间隙当天一次（跳变前偏移解释、落点为跳变后本地时刻）、秋季重复小时两次都触发、日锚取正午防跨日误判；测试自嵌 `time/tzdata`，新增智利午夜跳变「跨日候选剔除」用例 |
| R4 | trash purge 409 收窄 | 仅 `ObjectLocked` → 409，其余上游错误落 500（此前一切 S3 API 错误都 409、掩盖真实故障）；连带删除零引用的 `s3wrap.IsAPIError` |
| R5 | 落盘串行化 `persistMu` | 「生成快照 + Save」整体串行（Scheduler / JobRegistry 各一，锁序 persistMu → mu），旧快照不再可能覆盖新状态；并发回归测试重构为真并发（Finish 入 goroutine + release 信号） |
| R6 | 四面板请求代际守卫 + loading 复位 | BucketsPanel / StorageReportPanel / RecycleBinPanel / AccountsPanel 各自内联 `loadSeq`（乱序成功不渲染、过期失败不触底）；RecycleBin 补 `loadingBuckets` 标志 + 桶选择器 `:disabled` + finally 复位（异常不再永久卡 loading） |
| R7 | 死类型导出清零 + 类型维度门禁 | 删 `types.ts` 三个死类型（`StorageClassUsage` / `PrefixUsage` / `StorageRecommendation`）；`deadcode_gate.test.ts` 把声明的「类型导出」盲区升级为真断言（`export type` / `interface` 生产零引用即红 + 合成口径用例）；生成物 `operations.ts → Operation` 零引用改 `gen-api.mjs` 输出 `as const satisfies Record<string, Operation>` 真正消费 |
| R8 | multipartParts 回归生成契约 | `endpoints.ts` 改 `opPath('multipartParts', { id })`、删过期注释；`api.gaps.test.ts` 加整路径断言守回归 |
| R9 | e2e OpenAPI 断言可失败化 | smoke 在静态预览无后端（确定性 502/503）时**条件跳过**、其余状态严格断言 200 + JSON + openapi 字段；常驻真断言迁入 `e2e-real/real-backend.spec.ts`（`scripts/e2e-real.sh` 补 `S3C_EXPOSE_OPENAPI=1`——生产默认 404 不暴露规范） |
| 登记 | O1–O12 → `KNOWN_ISSUES.md` #72–#83 | 载荷完整性 / s3wrap 导出面 / 传输层 status / generated.gate 穷尽性 / 重复 helper / a11y 焦点 / i18n 盲区 / OpenAPI 漂移门禁 / 错误回显 / accounts 列表面 / `make check` 缺项 / 散点缺陷群；R10 工具链残留同日解除（本机 `/usr/local/go1.26.9` 与 CI 同版本） |

> **门禁实跑（2026-10-09，本机实跑，go1.26.9）**：
>
> | 门禁 | 结果 |
> |---|---|
> | `gofmt -l` / `go vet` / `go build ./...` | 干净 / 0 告警 / OK |
> | `go test ./... -count=1` | **10/10 包 ok** |
> | `make test-cover`（`-race`） | **10/10 包 100.0% statements + `count==0` 零块**（R3 补智利午夜跳变跨日剔除例后全绿） |
> | `golangci-lint run` | **0 issues** |
> | `govulncheck ./...` | **0 可达漏洞** |
> | `pnpm lint` / `typecheck` + `typecheck:e2e` / `pnpm build` | 0 告警 / 均 exit 0 / OK |
> | `pnpm test:coverage` | **82 文件 1283 例全绿、四指标 100%（4879 / 3251 / 1229 / 4213）** |
> | `pnpm e2e`（mock 预览） | **21 passed + 1 skipped**（跳过项即 R9 条件跳过——OpenAPI 真断言在下一行） |
> | `make e2e-real`（真后端 + 真 RustFS + 真产物） | **5 passed**（35.1s，含新增「OpenAPI 规范 200 + JSON spec」真断言） |
> | `pnpm gen:api --check` | exit 0（R7 重生成 `operations.ts` 后，由 `generated.gate.test.ts` 内跑） |

> **文档同步**：`docs/api.md` / `docs/threat-model.md`（两闸与新 reason 枚举）、`docs/user-guide.md`（计划任务 DST 语义）、`docs/DEVELOPMENT.md`（门禁清单补 `ci_consistency_gate` / 类型导出半边 + CI 一致性节 + e2e-real 描述）、`docs/KNOWN_ISSUES.md` #72–#83 + 台账、评审快照状态表与 §8 回写、[`../CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]` 同日条目；本节 + 头部摘要 + 目录行三处同步。

### §CA 2026-10-09 KNOWN_ISSUES #73 / #80 / #81 / #83 闭环（评审 Optional 项收口）

> 来源：[`KNOWN_ISSUES.md`](KNOWN_ISSUES.md) #73 / #80 / #81 / #83（2026-10-09 全仓评审 O2 / O9 / O10 / O12）。
> 同批 **#72 / #74–#79** 在本节写作时点（2026-10-09）仍开放（前端传输层 / 契约漂移 / 重复实现 / a11y / i18n 等；#82 见 §CB）——
> 同日分两批闭环：#72 / #74 / #75 / #77 / #78 见 §CC，#76 / #79 见 §CD。

| # | 项 | 改动 |
|---|---|---|
| #73 | s3wrap 导出面收窄 | `s3wrap_dto.go` 4 个签名带 AWS SDK 类型、仅供包内调用的转换函数降为小写（`fromS3Object` / `formatBuckets` / `describeACL` / `granteeLabel`），不再给新 handler 作者留下「合法」的 SDK 类型依赖入口 |
| #80 | 计划校验错误不回显 | `service` 新增 `ScheduleValidationError{Msg, Cause}`（`Msg` 固定、不含用户输入，`Cause` 仅落日志）+ `ScheduleValidationMessage` 通用回退；handler 改用它；`error_echo_gate_test.go` 新增对 `writeErr(..., 400, x.Error())` 形态的匹配与正则自检用例 |
| #81 | accounts 作用域堵创建面 | `scope.go` 端点级闸：`POST /api/accounts`（无 `{id}` 可判）对 `accounts` 作用域 token 一律 403（`reason=accounts_create`）——`accounts:[A]` 的 token 不再能铸造任意新账号；`GET /api/accounts`（列表，不含 `SecretKey` 的 `AccountView`）仍放行，语义写入 [`threat-model.md`](threat-model.md) 最小权限节 |
| #83 | 散点缺陷群 12 处 | ① `CopyObjectWithMeta` / `ChangeObjectStorageClass` 补 `wrapObjectTooLarge`；② `PurgeObject` 内层变量改名去遮蔽；③ `dialContextSSRF` 的 `last` 预置为错误（不再可能返回 `(nil,nil)`）；④ `store` / `sqlite` 的 `encryptAESGCM` 错误上抛（`encryptAESGCMFn` 注入测失败分支）；⑤ `atomicfile` 临时文件唯一命名 + 回收旧版固定名残骸；⑥ storage_report 金额取整到 1e-6；⑦ `Scheduler.List` 选择排序改 `sort.Slice`；⑧ 损坏 `schedules.json` 改名 `<path>.corrupt` 保留现场；⑨ 非法 cron 停摆写 `LastError` 并落盘；⑩ `Job.Total` 改 `total` 字段 + 加锁 `Total()`；⑪ 计划 / 任务落盘 `Save` 失败计数并入新指标 `s3c_persist_failures_total` |

> **门禁实跑（2026-10-09，本机 go1.26.9）**：`gofmt -l` 干净 / `go vet ./...` 0 告警 / `go build ./...` OK /
> `golangci-lint run` **0 issues** / `go test ./...` **10/10 包 ok** / `go tool cover` **全包 100.0% statements、
> `count==0` 零块**（`go test ./... -coverprofile` 逐块核查）。
>
> **文档同步**：[`KNOWN_ISSUES.md`](KNOWN_ISSUES.md)（§二 移除三行 + 台账 + 头部摘要）、
> [`OPERATIONS.md`](OPERATIONS.md)（新增 `s3c_persist_failures_total` 指标行）、
> [`../CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]`、本节 + 头部摘要 + 目录行。

### §CB 2026-10-09 / 10-10 KNOWN_ISSUES #82 闭环（R10 残留：`make check` 缺项 / E2E 路径过滤 / DEVELOPMENT 漂移）

> 来源：[`KNOWN_ISSUES.md`](KNOWN_ISSUES.md) #82（2026-10-09 全仓评审 §5 O11，亦即 R10「工具链 / 文档不一致」的残留半边）。
> **2026-10-10 再处置**：`make check` 原先仍漏 CI `web` job 的 `pnpm typecheck` 与 `pnpm build`
> 两步（注释却声称「web job 的全部」）；本日补齐并**真跑 `make check` 全绿**复核。

| 面 | 改动 |
|---|---|
| `make check` 缺项 | `Makefile` 新增 `build`（`go build ./...`）与 `web-lint`（`pnpm lint`）目标；**2026-10-10 补齐 `web-typecheck`（`pnpm typecheck`）与 `web-build`（`pnpm build`）**，`check` 聚合改为 `vet govulncheck lint build test-cover web-lint web-typecheck web-typecheck-e2e web-test-cover web-build`，`.PHONY` 同步——逐项覆盖 CI `server`（gofmt 除外）与 `web` 两个 job 的全部静态门禁 |
| E2E 路径过滤 | 真 RustFS Go E2E 的 PR 触发面从 `apps/server/internal/s3wrap/**` 放宽到 `apps/server/**`（GitHub `e2e.yml` 与 GitLab `.e2e-rustfs-trigger` 双侧）——只改 `internal/handler/**` / `service` / `config` 的后端 PR 不再跳过唯一的真 S3 对端门禁 |
| DEVELOPMENT 三处漂移 | ① `perf.yml` 的 job 名 `perf-budget` → 真实 id `bench`；② 覆盖率排除项补全为 `src/main.ts` / `src/env.d.ts` / `src/**/*.test.ts` / `src/i18n/messages/**` / `src/assets/**`（原只写 `i18n/**`）；③ workflow_dispatch 触发行的 job 计数纠正（`ci.yml` 共 6 个 job；GitLab 7 个自动 job + `semgrep-sast`） |
| 防回退门禁 | 新增 `doc_ci_drift_gate_test.go`（文档引用的 CI job id 必须真实存在 + 覆盖率排除项必须完整写出）；`repo_infra_gate_test.go` 的 `TestMakefileCheckMirrorsCIStaticGates` 断言 `check` 前置含 `web-typecheck` / `web-build` 且对应目标真的调用 `pnpm typecheck` / `pnpm build`；`ci_consistency_gate_test.go` 加 `TestRustFSE2ETriggersCoverWholeBackend` |

> **验证（2026-10-09 TDD 先红后绿；2026-10-10 真跑复核）**：`go test .`（含全部门禁）ok；
> **`make check` 实跑 exit 0**——后端 vet / govulncheck / golangci-lint / `go build ./...` /
> `test-cover`（-race，全包 100%）与前端 lint / typecheck / typecheck:e2e / test:coverage（100%）/
> build 逐项通过。
>
> **文档同步**：[`KNOWN_ISSUES.md`](KNOWN_ISSUES.md)（§二 移除 #82 行 + 台账 + 头部摘要）、
> [`DEVELOPMENT.md`](DEVELOPMENT.md)（三处漂移 + 门禁清单）、
> [`code-review-2026-10-09.md`](archive/code-review-2026-10-09.md)（R10 状态回写）、
> [`../CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]`、本节 + 头部摘要 + 目录行。

### §CC 2026-10-09 KNOWN_ISSUES #72 / #74 / #75 / #77 / #78 闭环（评审 Optional 项第二批）

> 来源：[`KNOWN_ISSUES.md`](KNOWN_ISSUES.md) #72 / #74 / #75 / #77 / #78（2026-10-09 全仓评审 O1 / O3 / O4 / O6 / O7）。
> 至此 §5 的 Optional 项仅剩 **#76 / #79** 开放。

| # | 项 | 改动 |
|---|---|---|
| #72 | 数据面载荷完整性收窄 | `s3wrap/client.go` 的 `unsignedPayloadSetter` 只在**带 stream** 的请求（PutObject/UploadPart）注入 `UNSIGNED-PAYLOAD`：无 body 的 GET/HEAD/DELETE/List 交回 SigV4 默认空体哈希签名（此前无条件注入，令这些请求也无谓放弃可计算的载荷哈希）。`http://` endpoint 带 body 操作的残留风险记入 [`threat-model.md`](threat-model.md) §6.2 |
| #74 | 前端传输层保留 status | `api/http.ts` 新增导出类 `ApiError{status, body}`：错误响应归一携带 HTTP 状态与已解析响应体（`toError` 返回它），成功响应的非 JSON body 由 `request` 边界校验上抛 `ApiError`；`api/jobs.ts` 的 SSE 失败也改抛 `ApiError` |
| #75 | generated.gate 双向穷尽 | `api/generated.gate.test.ts` 新增：扫描生产源码所有 `opPath('<id>')`，① 引用的 opId 必须存在于 `operations`（spec 删路径后的 stale ref 即红）；② `operations` 中未被引用的 opId 必须**正好等于** 8 个「前端不消费」白名单（spec 新增操作漏接 / 白名单腐烂即红）。已变异验证 |
| #77 | 右键菜单 a11y | `ObjectContextMenu.vue`：打开记录 `previousFocus`、关闭还原；Escape 改经 `useKeydownStack`（LIFO，仅最上层接收），删掉 `useObjectBrowser.ts` 里绕过 LIFO 的独立 window Escape 监听；`ObjectList.vue` 的「⋯」触发器补 `aria-haspopup="menu"` 与随状态变化的 `aria-expanded`（新增 `ctxEntryKey` prop） |
| #78 | i18n 覆盖盲区 + 硬编码文案 | `i18n/coverage.test.ts` 新增「模板 / 数据驱动键逐个有定义」（按 `STORAGE_CLASS_VALUES` / `PROVIDERS` / `PROVIDER_GROUPS` / `storageReport.kind.*` 枚举）；清理 3 处硬编码可见文案为 key（`accounts.accessKeyId` / `accounts.accessKeySecret` / `batchEdit.tagsReplace`） |

> **门禁实跑（2026-10-09，本机 go1.26.9 + node 26）**：
> 后端——`gofmt` 干净 / `go vet` 0 告警 / `go build ./...` OK / `golangci-lint` 0 issues /
> `go test ./...` 10/10 包 / `go tool cover` **全包 100.0% statements、`count==0` 零块**；
> 前端——`pnpm lint` 0 告警 / `pnpm typecheck` exit 0 / `pnpm test:coverage` **82 文件 1290 例全绿、
> 四指标 100%（4895 / 3260 / 1231 / 4230）**。
>
> **文档同步**：[`KNOWN_ISSUES.md`](KNOWN_ISSUES.md)（§二 移除 5 行 + 台账 + 头部摘要）、
> [`threat-model.md`](threat-model.md) §6.2（#72 残留风险）、[`../CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]`、
> [`code-review-2026-10-09.md`](archive/code-review-2026-10-09.md)（O 状态回写）、本节 + 头部摘要 + 目录行。

### §CD 2026-10-09 KNOWN_ISSUES #76 / #79 闭环（评审 Optional 项第三批；O1–O12 归零）

> 来源：[`KNOWN_ISSUES.md`](KNOWN_ISSUES.md) #76 / #79（2026-10-09 全仓评审 O5 / O8）。
> 至此 2026-10-09 评审的 **O1–O12（#72–#83）全部闭环**。

| # | 项 | 改动 |
|---|---|---|
| #76 | 前端结构性重复收敛 | ① `newRowKey` 的 7 份逐字复制 → `src/rowKey.ts` 的 `createRowKey(prefix?)` 工厂（TagsDialog / BucketTags / HeadersDialog / BatchMetadataDialog / LifecycleDialog / BucketCors / BucketPolicyVisualEditor 各自 `const newRowKey = createRowKey()`）；② 虚拟滚动管线 4 份 → `src/composables/useVirtualRows.ts`（scrollTop/viewportH → `windowed` 切片 + `onListScroll` + ResizeObserver 测量 + `resetWindowScroll`），ObjectList / MigratePanel / VersionsDialog / RecycleBinPanel 只传响应式数组 |
| #79 | OpenAPI 契约漂移收口 | ① 新增门禁 `openapi_status_declared_test.go`：handler 直接写出的 `http.StatusXxx`（含 `h.xxx()` 委托闭包）必须在该 operation 声明，通用码（400/401/403/404/405/413/429/500）由共享 `components.responses` 放行；② 补齐缺失声明：migrate jobs 404、trash purge 409、4 个异步端点 503、proxy 400+416、copy-object 409；③ `putObjectLock` 用新增的 `Schema.AnyOf` 表达 days/years 二选一；④ 新增门禁 `TestOpenAPIBucketNotRequiredInRequestBody`——`bucket` 从全部请求体 required 移除（`bucketOr` 回退账号默认桶，与共享 Bucket query 参数「可省略」一致）；⑤ 重生成 `docs/api/openapi.json` 与前端 `schema.d.ts` |

> **门禁实跑（2026-10-09，本机 go1.26.9 + node 26）**：
> 后端——`gofmt` 干净 / `go vet` 0 告警 / `go build ./...` OK / `golangci-lint` 0 issues /
> `go test ./...` 10/10 包 / `go tool cover` **全包 100.0% statements、`count==0` 零块**；
> 前端——`pnpm lint` 0 告警 / `pnpm typecheck` exit 0 / `pnpm test:coverage` **83 文件 1291 例全绿、
> 四指标 100%（4829 / 3241 / 1218 / 4180）** / `pnpm gen:api --check` 同步（`generated.gate`）。
>
> **文档同步**：[`KNOWN_ISSUES.md`](KNOWN_ISSUES.md)（§二 移除末两行 + 台账 + 头部）、
> [`api.md`](api.md)（Object Lock 请求体二选一 / bucket 可省略口径已一致，核对无需改）、
> [`../CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]`、
> [`code-review-2026-10-09.md`](archive/code-review-2026-10-09.md)（O 状态归零回写）、本节 + 头部摘要 + 目录行。

---

### §CE 2026-10-10 根 `.env.example` ⇔ compose 透传面双向一致（新增门禁）+ README 配置项计数订正

> **来源**：核对「本地 `.env` 是否缺环境变量」时，把根 `.env.example` 与三个 `docker-compose*.yml`
> 的 `${VAR}` 插值面做集合比对，发现**双向漂移**（漏 7 / 多 3）；顺带抓到 README 的配置项计数停在 18。

| # | 项 | 改动 | 证据 |
|---|---|---|---|
| 1 | 根 `.env.example` 漏列 compose 实际插值项 | 补 `S3C_REGION` / `S3C_LOG_LEVEL` / `S3C_CORS_ORIGINS` / `S3C_SHUTDOWN_TIMEOUT` / `S3C_IMAGE_TAG` + 构建参数 `GOPROXY` / `NPM_REGISTRY`；按「必填 / 可选覆盖 / RustFS 凭据」分区重排 | 三文件 `${VAR}` 插值集共 14 键，改前根模板只覆盖 7 个 |
| 2 | 根 `.env.example` 列了 compose **不透传**项 | 移除 `S3C_ALLOW_PLAINTEXT_STORE` / `S3C_TRUSTED_PROXIES` / `S3C_SSRF_DENY_PRIVATE`，改为注释指引 `apps/server/.env.example` | compose `environment:` 是显式白名单且无 `env_file:`，写进根 `.env` 不生效；`docker-compose.yml` 明文存储注释同步订正为「需加入 `environment:` 白名单」 |
| 3 | README 配置项计数陈旧 | 「含全部 18 个 `S3C_*` 变量」→ **22** | 与 `internal/config` 读取数、[`CONFIGURATION.md`](CONFIGURATION.md) 表格一致 |
| 4 | 防回退门禁（TDD 先红后绿） | 新增 [`env_example_gate_test.go`](../apps/server/env_example_gate_test.go)：双向断言（compose 每个 `${VAR}` 必须在根模板有落脚点 + 根模板每个赋值行都必须是 compose 插值项）；`doc_number_gate_test.go` 的 `docNumberClaims` 新增 README「N 个 `S3C_*` 变量」声明（真值复用 `configEnvVarsFromSource`） | 先红：根模板漏 7 / 多 3、README 18≠22；后绿；`go test .` 全部门禁通过 |
| 5 | 模板未在文件内说明「只填未注释项」与行内注释陷阱 | 两份 `.env.example` 顶部补用法说明（`#` 仅行首是注释；`S3C_LOG_LEVEL=info # 说明` 的值是整串 `info # 说明`）；[`CONFIGURATION.md`](CONFIGURATION.md) §1 同步解析规则 | 新增 `TestEnvExamplesAvoidInlineComments`（两份模板）；**变异验证**：注入 `S3C_LOG_LEVEL=info # mutation-check` → 红并点名 `.env.example:53`，还原 → 绿 |

> **文档同步**：[`../CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]`、[`CONFIGURATION.md`](CONFIGURATION.md) §1
> （补「显式白名单」口径）、[`DEPLOYMENT.md`](DEPLOYMENT.md) §2.2、[`../README.md`](../README.md) 配置段、
> [`DEVELOPMENT.md`](DEVELOPMENT.md) §3 门禁清单、本节 + 头部摘要 + 目录行。

---

### §CF 2026-10-10 `make dev` 受管 web PID 指向修复 + 启动流程去自动 tidy（新增门禁）

> **来源**：`make dev` 打印 `[web] 警告: pid=… 存活但不是预期的 'vite' 进程（PID 已被复用？）`，
> 随后 Vite 静默落到 1950；每次启动泄漏一个 vite 进程（实测两条 vite 链并存）。

| # | 项 | 改动 | 证据 |
|---|---|---|---|
| 1 | 受管 web 记录的是 pnpm 包装进程，与 `expected_process_pattern web=vite` 失配 | `run-dev.sh` / `graceful-restart.sh` 改为直接启动 `./node_modules/.bin/vite`（该 shim 末行 `exec node …/vite/bin/vite.js`，不换 PID） | 实际进程链 `pnpm.mjs → pnpm.cjs → vite.js`；实测 SIGTERM 包装进程可级联杀链，故问题纯在**身份校验失配**而非信号传播 |
| 2 | 端口被占时 Vite 静默换端口，`wait_http 1949` 被旧实例「骗过」 | 启动加 `--strictPort`（显式失败）；`graceful-restart.sh` 增 `.bin/vite` 缺失预检 | 修复前 1949 / 1950 各一条 vite 链；修复后 1949 单一 owner = `.run/web.pid` 所指进程、1950 空闲 |
| 3 | 防回退门禁（TDD 先红后绿） | 新增 [`dev_scripts_gate_test.go`](../apps/server/dev_scripts_gate_test.go) 的 `TestManagedWebServerRecordsVitePid`（4 条断言） | 修复前红（两脚本各 3 条点名）；修复后绿；真实 `make dev` 无告警、`/api/health` ok |
| 4 | 启动流程每次自动 `go mod tidy`，静默改写提交文件 | `run-dev.sh` / `graceful-restart.sh` 移除全部 `go mod tidy`，依赖同步只留显式 `make tidy`（`go build` 默认 `-mod=readonly`：不一致会显式报错） | 改动前每次 `make dev` 删 `go.sum` 6 行；改动后 `make dev` 前后 `go.mod` / `go.sum` sha256 **完全一致**；门禁 `TestDevScriptsLeaveDependencySyncToExplicitTargets` 先红（两脚本点名）后绿 |

> **文档同步**：[`../CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]`、
> [`DEVELOPMENT.md`](DEVELOPMENT.md) §3 门禁清单、本节 + 头部摘要 + 目录行。

---

### §CG 2026-10-10 两处文档状态漂移收口（悬空 `ROADMAP #11` 引用 + trace 观测面自相矛盾；新增状态陈述门禁）

> **来源**：全 docs「未处理问题」盘点时**实测发现**（非评审登记项）——既有文档门禁只钉
> 「链接可达 / 叙述性数字 / CI 事实」，**状态陈述**无机械校验，两处漂移全绿潜伏。

| # | 项 | 改动 | 证据 |
|---|---|---|---|
| 1 | `OPERATIONS.md` §4.3 trace 行写 `未接入（ROADMAP §三 3.2 #11 ⬜）`——#11 已于 2026-10-08 落地转空号（本文件 §BR），**同文件 §3.4 就是已实现的 OTel tracing**：悬空编号 + 自相矛盾 | 该行改为「导出已内置、采集端在服务外（§3.4，`S3C_OTEL_ENDPOINT` 留空 = 关闭）」+ 配置指引；§3.2 尾部「仍只能外部采集的观测面」两句同步为「trace 的**采集端**（span 导出已内置 §3.4）」 | 运维照旧行配不出 trace；门禁断言 1+2 上线即红 |
| 2 | `FEATURES.md` §P / §Q / §CA 三处章节头注在 2026-10-10 修复前用**无日期锚的现在时**写「仍未处置 / 仍开放」，与同文件闭环章节矛盾 | 三行补**日期时点锚** + 闭环指针（§P→§Q/§R/§S、§Q→§S/§T/§U、§CA→§CC/§CD），原文事实不改写 | FEATURES 是**已完成**台账，无锚点历史状态会被读成现状；门禁断言 3 上线即红（3 行点名） |
| 3 | 防回退门禁（TDD 先红后绿，三条断言均带扫描面自检基线） | 新增 [`doc_status_drift_gate_test.go`](../apps/server/doc_status_drift_gate_test.go)：① 带 ⬜/⏳/未排期/未接入 的 `§三 #N` 引用 ⇒ 必须指向 ROADMAP §3.1/§3.2 现存条目（历史出处指针与三类时点台账除外）；② `OPERATIONS.md` §3.4 存在时 trace 行不得写「未接入」且必须点名 `S3C_OTEL_ENDPOINT`；③ `FEATURES.md` 的开放陈述行必须带日期（格式 `YYYY-MM-DD`，如 2026-10-10） | 先红（两处漂移 + FEATURES 三行）后绿；**变异验证**三轮：改回旧文案 → 1+2 双红、改引 `#99 ⬜` → 1 红、删日期锚 → 3 红，还原全绿；全量 `go test ./...` 10/10 包 **100.0%**（`count==0` 零块）、`golangci-lint` **0 issues** |

> **文档同步**：[`../CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]`、
> [`DEVELOPMENT.md`](DEVELOPMENT.md) §3 门禁清单、本节 + 头部摘要 + 目录行。

---

### §CH 2026-10-10 A 组六项「可机械优化」收口（inode 指标内置 + 三道门禁 + 网格态 a11y + 明文端点告知）

> **来源**：全 docs「明确没做 / 无机械保证」盘点归类出的 **A 组**——仓内闭环、有先例可循，
> 指令 A1–A6（A6 与 A2 同一门禁文件，全并入 A2）。原则不变：**TDD 先红后绿**，
> 把「文档里的一句话」换成「跑得起来的断言」或「真实存在的信号」。

| # | 项 | 改动 | 证据 / 门禁 |
|---|---|---|---|
| A1 | `OPERATIONS.md` §4.3「数据卷 **inode** 仍无内置指标」 | `internal/handler` 新增 `volumeInodes()`（`volume_unix.go` 取 statfs `Files` / `Ffree`、`volume_windows.go` 恒 `ok=false`——Windows 无 inode 概念、`volume_other.go` 恒 `ok=false`），`/api/metrics` 发 `s3c_volume_inode_total` / `s3c_volume_inode_free`，**沿用「取不到就不发序列」口径**（总数为 0 会让 `free/total` = 0/0 = NaN，告警静默失效）；`OPERATIONS.md` §3.2 表 + §3.2 尾注 + §4.3 行转「仅序列缺失平台」+ §10.2 资源表、`api.md` 指标清单、`DEPLOYMENT.md` 同步；`rules.yml` 新增告警 `S3ClientVolumeInodeLow`（空闲 < 20% 持续 10 分钟）+ §4.2 对应行 | `metrics_test.go` `TestMetricsVolumeInodes`（三态：未配目录 / 正常 / statfs 失败）先红后绿；`TestPrometheusRulesReferenceRealMetrics` 校新指标真实发射；四平台 `GOOS=darwin/freebsd/windows/openbsd go build` 全过 |
| A2 / A6 | `AI_POLICY.md` §11「不自动合并 PR / 不自动删分支或数据」原标 **⚠️ 人工**、**且无门禁守着**（零命中只是现状） | 新增 `TestNoAutoMergeOrPrivilegedCITriggers`：扫 `.github/workflows/*` + `.gitlab-ci.yml` + `scripts/*.sh`，命中 `gh pr merge` / `enable-auto-merge` / `mergify` / `merge_when_pipeline_succeeds` / `auto_merge` / `pull_request_target` / `git push --delete` / `git branch -D` / `delete-branch` 任一即红；**双向钉**：§11 必须回指测试名。A6 的「禁止类」可机检部分并入本条，§11 对应行改 **代码强制**；「进入发布模式需人类授权」等仓库外判定项仍如实标 ⚠️ 人工 | `ai_governance_gate_test.go`（含扫描面自检 `minCIScannedFiles=18`）；`docs/AI_POLICY.md` §11 回指 |
| A3 | `OPERATIONS.md` §4.2 表 ⇔ `rules.yml` 双向同步此前**只靠人工**：实测表 13 行 = 13 条 alert **纯属巧合**——表多「健康探测失败」（外部黑盒探测，不属 Prometheus 规则）、规则多 `S3ClientZipPartialFailures`，两边集合根本不同 | §4.2 每行第一列补**反引号 alert 名**；外部探测项移入表外注记（保住阈值信息）；补 `S3ClientZipPartialFailures` 行；新增 `doc_alert_drift_gate_test.go` 断言「表行 ⇔ `- alert:` 双向等集」（各带 ≥13 条扫描面基线）；§4 导语与 `rules.yml` 头注同步登记该门禁 | `TestOperationsAlertTableMatchesRulesFile`：上线即红并**点名漂移**（表无 `S3ClientZipPartialFailures`、规则多两条），补行 + 加名后转绿；`DEVELOPMENT.md` §3 门禁清单登记 |
| A4 | `accessibility.md` §4 第 9 条「网格单元格 `role=button` 无 `aria-label`」**从没被 axe 扫过**（四个扫描态都不含网格视图），真伪未判 | `e2e/a11y.spec.ts` 新增第 5 个扫描态「对象网格视图」（`installObjectsStub` 桩出带默认桶的账号 + 3 个对象 → 切网格）；同批把文件头承诺的「非阻塞级违规打印供人工判断」真正接上（此前代码直接丢弃）。**判定结果：axe 0 违规（阻塞级与非阻塞级均无）→ 非缺陷，不改组件**——`button-name` 认可文本节点派生的可访问名，加 `aria-label` 反而会覆盖「文件名 + 大小」并造出第二事实源 | `pnpm exec playwright test a11y.spec.ts` **6 passed**；`accessibility.md` §4.9 由「仍未做」转「已判定 + 受 CI 约束」，§4 第 2 / 4 条与 §5.2 / §5.3 状态数 4 → 5 |
| A5 | `threat-model.md` §6.2 `http://` 载荷完整性**残留**（本体只能靠 TLS）且账号创建 / 更新时**零提示** | 对齐 `main.go` 明文落盘告警先例：`handler/accounts.go` 新增 `warnPlaintextEndpoints`——`endpoint` / `publicEndpoint` 经 `s3wrap.NormalizeEndpoint` 归一化后为 `http://` 即打 WARN（创建 + 更新两路；日志只记 id / 名称 / 端点，**不含密钥**）；前端 `AccountsPanel.vue` 在「使用 HTTPS/TLS」**未勾选**时于勾选框下给出同一句提示（`accounts.useSSLWarn`，中英双语），勾选即消失 | `accounts_plaintext_test.go` 三条（显式 `http://` / 大小写 scheme / 裸端点 + `useSSL=false` 打 WARN，`https://` 与裸端点 + `useSSL=true` **不打**（防噪声），更新路径与 `publicEndpoint` 各一条，外加「日志不含 `secretKey` 哨兵」）；`AccountsPanel.test.ts`「明文端点提示」用例断言展示 / 勾选消失 / 取消恢复 |
| 相邻 | **上一批遗留**：后端默认端口 8080 → 5000 之后，`e2e/smoke.spec.ts` 的「无后端 → 条件跳过」判据失效——旧口径按状态码猜（注释写 503），而 vite 代理在目标拒绝连接时实测回 **500 + `text/plain` + 空 body**（vite 源码 `res.writeHead(500, …)`），于是 `pnpm e2e` 在**没起后端的常态下**必然红灯；更糟的是真实后端的 404 恰好也是 `text/plain`，按 content-type 猜会把「端点坏了」误判成「没后端」 | 改用**后端自己写的头**作判据：`X-Request-ID`（`withRequestID` 中间件覆盖全部路由，含 404 / 5xx，见 `OPERATIONS.md` §3.3）——vite 自己应答时不带该头。判据来自后端而非代理实现，不随 vite 版本漂移 | **双向实测**：不起后端 → `1 skipped`（原本红）；起后端但未开 `S3C_EXPOSE_OPENAPI` → `1 failed`（R9 严格断言**未被削弱**）；随后关后端复跑 → `22 passed + 1 skipped` |

> **文档同步**：[`../CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]`、
> [`DEVELOPMENT.md`](DEVELOPMENT.md) §3 门禁清单（`doc_alert_drift_gate_test.go`）、
> [`AI_POLICY.md`](AI_POLICY.md) §11（A2/A6）、[`OPERATIONS.md`](OPERATIONS.md) §3.2 / §4.2 / §4.3 / §10.2、
> [`api.md`](api.md)、[`DEPLOYMENT.md`](DEPLOYMENT.md)、[`threat-model.md`](threat-model.md) §6.2、
> [`user-guide.md`](user-guide.md)、[`accessibility.md`](accessibility.md) §4 / §5.2 / §5.3、
> 本节 + 头部摘要 + 目录行。

---

### §CI 2026-10-10 全 docs 通读问题清单收口（表渲染断裂 / 过期数字与悬空指针 / 安全契约缺失 / 通用状态码接线 + 三道门禁）

> **来源**：四路并行通读 `docs/` 全量（23 篇顶层 + `decisions/` 14 + `archive/` 9 + `en/` 3），
> 逐条与代码 / 实跑数字交叉核验后的问题清单；按严重度排序处置。原则不变：**能机械化的换成门禁，
> 不能机械化的当 PR 同步改掉**。

| # | 项 | 改动 | 证据 / 门禁 |
|---|---|---|---|
| 1 | [`README.md`](README.md) 机器可读面表格被空行劈断（后 4 行无表头，GitHub 渲染成裸文本） | 删空行并回同一张表；顺带「三条实测结论」→ 四条（[`PERFORMANCE.md`](PERFORMANCE.md) §3 实为 3.1–3.4） | 目测渲染 + `doc_link_gate` 全绿 |
| 2 | [`ROADMAP.md`](ROADMAP.md) 页头日期停 2026-09-29、§四 基线落后两轮（82 文件 / 21 e2e）、里程碑表候选池口径过时 | 页头 → 2026-10-10；**§四 全量实跑回填**（见本节末「2026-10-10 基线」）；候选池改「现存 #4 / #9 / #15 / #16」 | 逐数字实跑复现；`go test .` 包根门禁全绿 |
| 3 | [`FEATURES.md`](FEATURES.md) §三 唯一无日期「现状表」整节过期（9/9 包 / 76 文件 / go1.26.6 等 6 处与仓库矛盾），与 ROADMAP §四 双基线互相打架 | §三 改为**指针**（「发布前基线唯一来源 = ROADMAP §四」）+ 新增 2026-10-10 全量复测块；只保留 §四 未单列的补充项 | 两处「现状表」漂移的根因处置 |
| 4 | [`data-model.md`](data-model.md) **安全级缺失**：`json`/`sqlite` 无 `S3C_STORE_KEY` 拒绝启动（`S3C_ALLOW_PLAINTEXT_STORE`）在 SSOT 地图里完全不存在，§3/§4.1 措辞反导向 | §0 补 SSOT 行、§3 补「明文闸」段、§4.1 块引用补警示、§7 fail-closed 表新增一行 | 与 `config.go:339` / `CONFIGURATION.md` §3 / `threat-model.md` 对齐 |
| 5 | [`api.md`](api.md) + `docs/api/openapi.json` 契约失真（8 中 + 4 低）：「全是 JSON」错、sync `truncated` 语义误导（**重跑仍从头列举、100k 后永远漏**）、漏 `lastError`、429/413 通用码缺席、proxy 漏 `versionId`、`accounts_create` 作用域闸漏登、24h 钳制位置指错 | 逐条改写；**通用状态码 `401/413/429/500` 逐 operation 接线**（见 #6）；补 `s3c_persist_failures_total` 指标 | `api_doc` / `openapi_*` 门禁全绿；黄金契约重生成 |
| 6 | `components.responses` 三个共享响应（`Unauthorized` / `TooManyRequests` / `InternalError`）**定义后 0 引用**、84 个 operation 无一声明 401/413/429/500（codegen / Swagger UI 看不到） | **TDD 先红后绿**：`TestOpenAPIUniversalResponsesAreWiredPerOperation`（逐 op 通用码 + 孤儿组件双向断言）→ 新增 `Registry.ForEachOperation` + `applyUniversalResponses` 统一补挂 + 新共享组件 `PayloadTooLarge`；`info.description` 的 JSON 例外清单同步订正；重生成 `openapi.json`（+891 行）与前端 `schema.d.ts`（+302 行） | 先红（84 op 全缺 + 3 孤儿）→ 后绿；`pnpm test` 83 文件 1292 例全绿（生成物门禁含内） |
| 7 | [`errors.md`](errors.md)：引用不存在的符号 `s3UserMessageForCode`、漏 429 行、`ErrPartialDelete` 文案不全 | 改为 `service/delete.go` 复用 `s3wrap.UserMessageForCode` 的真实链路；补 429 行（`withRateLimit` 在 `withAuth` 外层）与带计数文案 | 逐条对照 `errors.go` / `ratelimit.go` |
| 8 | [`architecture.md`](architecture.md) 分层图与真实 import 不符（画了不存在的 `s3wrap→store`、漏 `handler→store` 与 `openapi` 层）、§5 写 `fetch` 实为 XHR、ADR 编号体例分裂 | 分层图改**真实边集**（实测 import 后重画）+ 栈序简化的口径说明；`fetch` → XHR；全仓 `ADR-0013`→`ADR-013`、`ADR-0006`→`ADR-006` 统一（含 `en/`、`OPERATIONS`、`CONFIGURATION`），模板补「文件名四位 / 标题三位」体例 | `grep -rho internal/…` 实测边集；`adr_coverage` / `doc_link` 全绿 |
| 9 | [`glossary.md`](glossary.md) 分层术语与 architecture / 子树 AGENTS 三处互不一致 + 3 处小瑕疵 | 分层条目改指 architecture §2 权威；`expiresIn` 指针、标签双层语义、`IsTerminalJobStatus` 文件名同步 | 与 #8 同批 |
| 10 | [`i18n.md`](i18n.md) 漂移最重：漏登 `schedules.ts` / `storageReport.ts`（6→8 模块、682→818 key）、合并顺序与「同名 key 赢家」结论错、`<html lang>` 结论被 `applyDocumentLang` 推翻、用例 6→7、入口按钮位次错 | 按实测数字与源码逐处改写（§1 / §2 表 / §3 步骤与 bullet / §4 / §5 / §7.5 / 统计口径） | 复现命令实跑（8 模块 818 key） |
| 11 | [`accessibility.md`](accessibility.md) 自称「数字全部由命令跑出」但四处过期 | `role=alert` 9→19（补 4 组件逐文件）、补 `aria-expanded`/`aria-haspopup` 行、`aria-hidden`/`aria-label` 更新、表 15→19、表单控件 77→94、`useKeydownStack` 9→7 例、§5.2 的 `aria-multiselectable` 旧口径修正；§1.1 加**统计时点** | 按 §1.1 自带命令实跑 |
| 12 | `docs/en/`：`en/index.md` `all 18 S3C_*` 漏改（真值 22）、`en/architecture` 60/72 双错（真 72/84）、缺 FinOps / 已知限制 / 4 个文档索引项、revision 全部过期、缺中文源的 mermaid 时序图 | 数字与内容同步；revision → `30599f0`（2026-10-10）；补英文时序图与分层边集、`en/README` 补 4 个中文入口 | **门禁扩面**（见 #13） |
| 13 | 数字类漂移无门禁：`doc_number_gate` 只扫中文 3 处 | **TDD**：`docNumberClaims` 新增 4 条（`en/index` 的 S3C 与端点数、`en/architecture` 端点数、中英 architecture 的 `s3api` 方法数 ⇔ `endpoints.ts`）；新增 `frontendMethodCountFromSource` 口径 | 先红（en 页 18≠22、60≠72、72≠84）→ 改页后绿 |
| 14 | 导航计数无门禁：ADR「12 篇」（实为 13）在 [`README.md`](README.md) 与 `llms.txt` 双双失真 | 计数改 13；**TDD 新增 `TestADRIndexListsEveryADRFile`**（index ⇄ `decisions/` 文件双向等集）；`adr_coverage` 头注「已知盲区」同步 | **变异验证**：删 index 一行 → 红灯点名 `0007-…md` → 还原绿 |
| 15 | `llms.txt` 把产物核验命令挂到 `THIRD_PARTY_LICENSES.md` §5.3（该文件无此节，真在 `threat-model.md` §5.3） | 改指 `docs/threat-model.md` §5.3（与根 README 同口径） | 纯文本引用，`doc_link` 不可见故人工修 |
| 16 | [`DEPLOYMENT.md`](DEPLOYMENT.md) §6.4 升级 / §7 回滚过薄、与 [`OPERATIONS.md`](OPERATIONS.md) §8 循环互指（**关键操作步骤实际缺失**——本次唯一「缺内容」项） | §6.4 补 5 步可执行序列（备份 → `S3C_IMAGE_TAG` → pull/stop/up → 六项验证 → 回滚）+ 格式降级警告；§7 补 tag 改法 / 回滚后验证 / 备份硬要求 | 与 `docker-compose.prod.yml` 实测对照 |
| 17 | 页头日期集体滞后 + 悬空 `§三 #N` 指针（`doc_status_drift` 只拦带状态标记的，无标记引用全漏网） | [`KNOWN_ISSUES.md`](KNOWN_ISSUES.md) → 2026-10-10 并**补齐 §四 缺失的 #72/#74/#75/#77/#78 行**、#47–#59 指针改「落地转 FEATURES」；[`AI_POLICY.md`](AI_POLICY.md) → 2026-10-10（§11 升级代码强制）；[`threat-model.md`](threat-model.md) → 2026-10-10；[`DEVELOPMENT.md`](DEVELOPMENT.md) #10/#19 悬空引用修正 + §3 补齐 5 道漏列门禁（`doc_link` / `doc_index` / `docs_naming` / `dependabot` / `third_party_licenses`）；[`CONFIGURATION.md`](CONFIGURATION.md) #13 悬空改 §BS | `go test .` 全绿（含 `doc_status_drift`） |
| 18 | [`CONFIGURATION.md`](CONFIGURATION.md) §5 漏 2 个真实存储键 | 补 `s3c.token.<serverId>`（主形态）与 `s3c.multipart.resume.v1`（零凭证、上限 20 条）；[`compatibility.md`](compatibility.md) 客户端键行改为「以 §5 为准」不再复制 | 对照 `storage.ts` / `multipartResume.ts` |
| 19 | [`compatibility.md`](compatibility.md) §6.2「文件 + 行号可查」表 **12 处行号漂移**（该节可信度全建立在行号可查上） | 逐条订正（`styles.css` 186–194 / 617、`i18n/index.ts` 69、`upload.ts` 19、`download.ts` 53–66 / 68–85 / 5+69–75、`jobs.ts` 45–72、`package.json` 18、`ci.yml` 376/414、`release-desktop.yml` 147） | 逐文件实测行号 |
| 20 | ADR 事实过期与裸引用 | [`decisions/0004`](decisions/0004-minimal-frontend-deps.md) 追加 Update（死债已闭环 / 行数不再回填）；[`0008`](decisions/0008-frontend-zero-dep-stack.md) 追加 Update（7→8 tab）；[`0013`](decisions/0013-zero-dep-otlp-tracing.md) 补 #11 移出注；`0005/0006/0009/0010` 的 `ASSESSMENT` / `review R8·R11·§B4` 裸引用补归档链接与闭环注 | 按 ADR「Update 追加不改写」纪律 |
| 21 | [`archive/index.md`](archive/index.md)：冻结件 `S3CLINET_*` 命令照抄会**静默 skip**、`code-review-2026-10-09` 索引行（终态）与正文 §8（中间态）冲突 | 头注补**改名总注**（复现须替换 `S3CLIENT_*`）；该行补「正文停在同日中间态，终态见台账」 | 冻结件不回写，索引层兜底 |
| 22 | [`AGENT_EVALS.md`](AGENT_EVALS.md) 度量表 10 天零回填无状态、「20+ 门禁」过时 | 表下加 ⚠️ 状态行（0 条已回填 + 无门禁强制）；基线事实改「30+ 道（27 包根 + 3 子包 + 3 前端）」 | 实数清点 |
| 23 | [`user-guide.md`](user-guide.md) 3 处误导操作 + [`POSTMORTEM_TEMPLATE.md`](POSTMORTEM_TEMPLATE.md) 裸 `§N` 撞自身章节号 | 页签顺序改实序（Object Lock 第 4 位，表格行同步移位并去重）；Esc 出处改 `ObjectContextMenu.vue` 键栈；顶栏连接状态改「优先显示服务端名」；模板 11 处裸 `§N` 全部限定为 `OPERATIONS.md §N`（含最严重的 `§7.1` 撞车） | 对照 `BucketsPanel.vue` `tabs` 与键栈实现 |
| 24 | [`OPERATIONS.md`](OPERATIONS.md) §4.3 标题「本服务不提供指标」与表内 2 行已内置矛盾；[`PERFORMANCE.md`](PERFORMANCE.md) 采集环境 go1.26.6 违反自订 §4.3 规则 | §4.3 标题改「需外部补齐 / 平台受限」；PERFORMANCE §2 加「未在 go1.26.9 复测」诚实注记（§3 改四条） | — |

> **2026-10-10 基线（本批实跑，同步为 [`ROADMAP.md`](ROADMAP.md) §四 当前值）**：后端 `gofmt` 干净 /
> `go vet` 0 / `golangci-lint` 0 issues / `go test` **10/10 包** / `make test-cover` **10/10 包 100.0%
> （`count==0` 零块）** / `govulncheck` **0 可达**（21 不可达模块漏洞）；前端 `pnpm lint` 0 /
> `typecheck` + `typecheck:e2e` exit 0 / `gen:api --check` exit 0 / `pnpm test` **83 文件 1292 例** /
> `test:coverage` **4829 / 3243 / 1218 / 4180 四指标 100%** / `pnpm build` OK（**417.73 kB，gzip 125.30 kB，
> CSS 33.76 kB**）/ `pnpm e2e` **22 passed + 1 skipped**；真实 E2E 两项——`make e2e-real` **5 passed**
> （`SERVER_PORT=8081` + 外部共享 RustFS `--no-rustfs`）、`S3CLIENT_E2E=1` **7/7 PASS**；
> `cargo audit --no-fetch` **0 漏洞**（7 告警）。`pnpm audit` 仍无法实跑（镜像无 audit 端点，沿用 CI）。

> **文档同步**：[`../CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]`、[`ROADMAP.md`](ROADMAP.md) §四 /
> 页头、[`README.md`](README.md)、[`DEPLOYMENT.md`](DEPLOYMENT.md) §6.4 / §7、[`OPERATIONS.md`](OPERATIONS.md) §4.3、
> [`DEVELOPMENT.md`](DEVELOPMENT.md) §3、[`CONFIGURATION.md`](CONFIGURATION.md) §2 / §5、[`api.md`](api.md)、
> [`errors.md`](errors.md)、[`data-model.md`](data-model.md)、[`architecture.md`](architecture.md)、[`glossary.md`](glossary.md)、
> [`compatibility.md`](compatibility.md)、[`i18n.md`](i18n.md)、[`accessibility.md`](accessibility.md)、[`user-guide.md`](user-guide.md)、
> [`threat-model.md`](threat-model.md)、[`AI_POLICY.md`](AI_POLICY.md)、[`AGENT_EVALS.md`](AGENT_EVALS.md)、
> [`KNOWN_ISSUES.md`](KNOWN_ISSUES.md)、[`PERFORMANCE.md`](PERFORMANCE.md)、[`POSTMORTEM_TEMPLATE.md`](POSTMORTEM_TEMPLATE.md)、
> `docs/en/`（index / README / architecture）、`docs/decisions/`（0004 / 0008 / 0013 / 0000-template / 0005 / 0006 / 0009 / 0010 / index）、
> [`archive/index.md`](archive/index.md)、[`llms.txt`](../llms.txt)、`docs/api/openapi.json` + `apps/web/src/api/schema.d.ts`（重生成）、
> 本节 + 头部摘要 + 目录行。

---

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
