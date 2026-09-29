# 获取支持（Support）

> 本文件说明**遇到问题时该去哪里、以及本仓库明确不承诺什么**。它是 [`CONTRIBUTING.md`](CONTRIBUTING.md)
> §联系与支持 的展开版：那里是**渠道一览**（唯一来源），这里补「提问前自查」「提交时给什么」
> 「不提供什么」三块，渠道本身不重复维护第二份。安全漏洞走 [SECURITY.md](SECURITY.md) 的私密渠道，
> 参与讨论需遵守 [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)。

## 1. 渠道一览

全部沟通走 **GitHub**——本仓库**不使用邮件列表，也不提供支持邮箱**（见 §4）。

| 场景 | 渠道 | 链接 |
|---|---|---|
| 缺陷（可复现的 bug） | 新建 Bug Issue | [新建 Bug Issue](https://github.com/weilai1949/s3clinet/issues/new?template=bug_report.md) |
| 功能建议 | 新建 Feature Issue | [新建 Feature Issue](https://github.com/weilai1949/s3clinet/issues/new?template=feature_request.md) |
| 安全问题（漏洞 / 加固绕过） | **走私密渠道，勿开 public issue**——GitHub 私有漏洞报告 | [SECURITY.md](SECURITY.md) §报告漏洞 |
| 行为准则投诉 | **私密**——按执行渠道私信维护者，勿开 public issue | [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) §执行 |
| 使用提问（怎么配 / 怎么用） | 公开 Issue（用 Feature Issue 模板或普通 issue 描述即可；**当前没有** Discussions / 论坛 / 群组等其它讨论区） | [新建 Issue](https://github.com/weilai1949/s3clinet/issues/new/choose) |
| 开发流程 / 门禁 / 提交规范问题 | 先读文档，仍不清楚再开 Issue | [`CONTRIBUTING.md`](CONTRIBUTING.md) · [`docs/DEVELOPMENT.md`](../docs/DEVELOPMENT.md) |
| AI 辅助 / 代理权限问题 | 见仓库级政策 | [`docs/AI_POLICY.md`](../docs/AI_POLICY.md) |

> **公开 issue 是公开记录**：粘贴日志前请先脱敏——**不要**贴出 `S3C_TOKEN`、`S3C_STORE_KEY`、
> 账号 `AccessKey` / `SecretKey`、真实桶名与业务对象 key。凭据一旦出现在公开 issue 里即视为已泄露，
> 请立即轮换（Token 轮换与配置方式见 [`docs/CONFIGURATION.md`](../docs/CONFIGURATION.md) §2）。

## 2. 提问前自查清单

绝大多数问题在开 Issue 前就能解决——请按顺序自查，并在 Issue 里写明**已经查过哪些**：

1. **版本是否是最新的**：`GET /api/health` 的 `version` 字段即服务端版本；版本命名与支持窗口见
   [`README.md`](../README.md) 与 [SECURITY.md](SECURITY.md) §支持的版本（旧版本请先升级再复现）。
2. **读快速开始与功能说明**：[`README.md`](../README.md)（部署方式、CORS / `ExposeHeader: ETag`
   等硬前提）与 [`docs/CONFIGURATION.md`](../docs/CONFIGURATION.md)（全部 `S3C_*` 环境变量 SSOT）。
   > 完整使用说明见用户手册 [`docs/user-guide.md`](../docs/user-guide.md)（首次配置 / 上传下载 /
   > 对象与桶操作 / 版本与回收站 / 快捷键 / FAQ / 排障）；接口字段见 [`docs/api.md`](../docs/api.md)。
3. **查已知问题**：[`docs/KNOWN_ISSUES.md`](../docs/KNOWN_ISSUES.md) 是**缺陷 / 外部阻塞 / 技术债的唯一来源**，
   当前开放项很少（例如桌面端未签名属外部凭证阻塞）；不要开已有条目重复的 issue。
4. **查功能候选与规划**：如果你要的不是修 bug 而是「以后会不会有 X」，先看
   [`docs/ROADMAP.md`](../docs/ROADMAP.md) §三——那里的候选池条目**是方向不是承诺**（⬜ 未排期），
   已有条目请直接评论原条目而不是另开。
5. **搜既有 issue / PR**：用关键字（错误文案、端点名、S3 服务商名）搜
   [Issues](https://github.com/weilai1949/s3clinet/issues) 与
   [Pull Requests](https://github.com/weilai1949/s3clinet/pulls)，确认没有重复。
6. **错误码先查对照表**：[`docs/errors.md`](../docs/errors.md)（S3 错误 → HTTP 状态映射）与
   [`docs/api.md`](../docs/api.md)（通用码 `400` / `401` / `404` / `500`）。
7. **启动就失败**：多半命中 [`docs/CONFIGURATION.md`](../docs/CONFIGURATION.md) §3「启动期硬失败清单」
   ——非回环监听未设 `S3C_TOKEN`、`json` / `sqlite` 未设 `S3C_STORE_KEY`、显式 `S3C_ENV_FILE` 不可读
   等情形都是**有意拒绝启动**，不是 bug。
8. **直传 / 分段失败**：先确认目标桶的 CORS 规则暴露了 `ETag`（分段组装硬前提），见
   [`README.md`](../README.md) 的兼容性矩阵。

## 3. 提交 Issue 时请提供

Bug Issue 模板（[`bug_report.md`](ISSUE_TEMPLATE/bug_report.md)）已列出必填项，核心是这四类信息：

| 类别 | 具体内容 |
|---|---|
| **版本** | `/api/health` 的 `version`（如 `v1.0.0` 或时间戳版 `v1.0.0-YYYYMMDDHHmmss`）；桌面端另附发布产物名 |
| **部署方式** | Docker Compose（`docker-compose.yml` / `docker-compose.prod.yml`）/ 本机二进制 / 桌面端；是否经反向代理与 TLS（若经代理，附代理类型） |
| **复现步骤** | 最小步骤序列 + **预期行为** + **实际行为**；涉及界面时附浏览器与操作系统；涉及 S3 侧时说明服务商（RustFS / MinIO / AWS S3 / 阿里 OSS / 腾讯 COS …） |
| **日志** | 服务端日志（建议 `S3C_LOG_LEVEL=debug`、`S3C_LOG_JSON=1` 便于采集）；浏览器控制台 / 网络面板中的失败请求；请**脱敏后**粘贴 |

可选但很有帮助：`/api/health` 与（开启 `S3C_EXPOSE_METRICS=1` 时）`/api/metrics` 的相关片段、
请求的 `X-Request-ID`（访问日志字段 `req` 与之对应，可直接定位一次请求）。

## 4. 本仓库不提供什么

出于「**单人维护的开源项目**」这一现实，以下内容**当前没有**，请不要按有来预期：

| 不提供 | 现状说明 |
|---|---|
| **SLA / 响应时限承诺** | 无任何形式的可用性或响应时间承诺；issue 与 PR 均按维护者个人时间处理。唯一写明的时限在安全渠道：[SECURITY.md](SECURITY.md) §报告漏洞 的「48 小时内确认收到」，仅适用于私密漏洞报告 |
| **商业支持 / 付费服务 / 优先级通道** | 无付费支持、无商业授权、无加急通道，也没有「购买支持」这类入口 |
| **邮件 / 即时通讯支持** | **本仓库不公开支持邮箱，也没有 IM 群或论坛**；所有公开讨论都在 GitHub Issues |
| **安全邮箱** | **不公开**——漏洞请走 GitHub 私有漏洞报告（[SECURITY.md](SECURITY.md) §报告漏洞 明确写明本仓库未公开安全邮箱） |
| **代运维 / 代部署 / 环境排障** | 不代办部署、不接入你的环境；请你自行按 [`docs/DEPLOYMENT.md`](../docs/DEPLOYMENT.md) 与 [`docs/threat-model.md`](../docs/threat-model.md) 加固 |
| **第三方服务（S3 服务商、云厂商、代理、证书）的责任** | 上游故障、计费、限流、CORS / ETag 行为差异属对方范畴；本仓库只保证自身的兼容性要求已文档化（见 [`README.md`](../README.md) 兼容性矩阵） |
| **数据恢复 / 找回已删除对象** | 不提供恢复服务。对象删除的后果由 S3 侧版本控制 / 生命周期决定；回收站与版本还原能力见 [`README.md`](../README.md) 功能列表，但**必须在删除前**开启对应能力 |
| **为旧版本长期供血** | 仅维护最新版本，安全修复随下一个版本发布；超出支持窗口的版本请升级（见 [SECURITY.md](SECURITY.md) §支持的版本） |
| **自定义功能开发 / 私有分支** | 功能需求走 Issue 讨论或自行 fork（MIT 许可，见 [`LICENSE`](../LICENSE)） |

## 5. 相关文档

- 渠道与提交规范（**渠道唯一来源**）：[`CONTRIBUTING.md`](CONTRIBUTING.md) §联系与支持
- 漏洞披露与支持版本表：[`SECURITY.md`](SECURITY.md)
- 行为准则与执行：[`CODE_OF_CONDUCT.md`](CODE_OF_CONDUCT.md)
- 已知问题（缺陷 / 阻塞 / 技术债唯一来源）：[`docs/KNOWN_ISSUES.md`](../docs/KNOWN_ISSUES.md)
- 功能候选与版本规划：[`docs/ROADMAP.md`](../docs/ROADMAP.md) §三
- 部署与运维：[`docs/DEPLOYMENT.md`](../docs/DEPLOYMENT.md)
- 配置项 SSOT：[`docs/CONFIGURATION.md`](../docs/CONFIGURATION.md)
- 治理、决策与发布权：[`GOVERNANCE.md`](GOVERNANCE.md)
- 兼容性承诺与弃用政策：[`docs/compatibility.md`](../docs/compatibility.md)
