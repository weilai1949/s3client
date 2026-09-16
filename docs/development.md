# 开发指南

> 面向贡献者与 AI 代理的开发规范。本文件合并了原 `agents.md` 的 TDD 约定与 `CONTRIBUTING.md` 的开发环境部分。
> 贡献流程、提交规范、环境搭建见 [CONTRIBUTING.md](../CONTRIBUTING.md)；代码架构见 [architecture.md](architecture.md)。

## 1. 核心原则（TDD 优先）

1. **先写会失败的测试，再写让它通过的实现。**
2. **小步提交**：一个逻辑改动一个 commit；改动前后都要 `go test ./...` 与 `pnpm build` 全绿。
3. **行为驱动，不测内部实现**：断言「外部可见的行为 / 返回值 / HTTP 状态码」，不要断言私有函数或内部变量。
4. **红灯-绿灯-重构（红绿蓝）**：先看到测试因缺实现而失败（红），再让实现通过（绿），最后在测试保护下优化结构（重构）。
5. **改完代码必须同步文档**：任何一次**修复 bug** 或**新增功能**完成之后，必须更新相关文档；文档未同步 = 改动未完成，不得提交 / 合并。文档与代码属于同一个 commit（或同一 PR）。

## 2. 三类测试（按项目现有基建落地）

| 层 | 位置 | 运行方式 | 目的 |
|----|------|----------|------|
| **单元 / 行为测试** | `server/internal/.../*_test.go` | `cd server && go test ./...` | 用 `httptest.NewServer` 的**假 S3** 验证 handler 层逻辑（路由、参数校验、正/反例、错误码映射） |
| **真实对端 E2E** | `server/internal/s3wrap/e2e_test.go` | `S3CLINET_E2E=1 go test ./internal/s3wrap/ -run 'TestE2E' -v` | 验证最硬核路径：**SigV4 签名 / 预签名直传 / 分段 Multipart 组装 / 跨 bucket 复制 / 标签 / 版本控制**。默认指向本地 RustFS |
| **前端类型 + 构建** | `web` | `cd web && pnpm build`（含 `vue-tsc --noEmit`） | 类型安全与可构建性；UI 改动同时保留手测/截图证据 |

### 假 S3 模式（handler 测试）
- 用 `httptest.NewServer` 模拟 S3，采用 **path-style**（请求落在 `/{bucket}/{key}`）。
- 依 `r.URL.Query().Has("acl"|"tagging"|"versions"|"location"|"versioning")` 与 HTTP 方法分发返回的 XML。
- 返回标准 S3 XML（`<ListVersionsResult>`、`<AccessControlPolicy>`、`<Tagging>` 等），并按需返回特定错误码（如 `NoSuchTagSet`）验证容错。

### 真实 RustFS 联调
- 需要时用 `docker compose up -d rustfs`（默认 `rustfsadmin/rustfsadmin`，S3 API 9000、控制台 9001）。
- E2E 测试用 `S3CLINET_E2E=1` 门控，普通 `go test ./...` 不会执行，CI 因此不受影响。

## 3. 必验门禁（每次改动提交前）

```bash
cd server && go vet ./... && go test ./...   # 后端
cd server && go build ./...                   # 后端可构建
cd web && pnpm test && pnpm build             # 前端单测 + 类型检查 + 构建
# 涉及签名/直传/分段/复制/标签/版本时，额外跑真实 RustFS E2E
cd server && S3CLINET_E2E=1 go test ./internal/s3wrap/ -run 'TestE2E' -v
# 或 make test-all（后端 + 前端单测）
```

> ⚠️ **关于覆盖率 100% 门禁**：项目 CI 对后端（90%）与前端（100%）设有覆盖率门禁。注意
> 100% 是「达到」而非「自然覆盖」——前端 `i18n/**` 被排除统计、后端存在为补分支而写的
> `gap` 测试。**写测试请以行为价值为先**（断言外部行为），不要为凑覆盖率而写与实现耦合的测试。

## 4. 文档同步门禁（改完代码必须更新文档）

任何「修复 bug」或「新增功能」完成之后，必须更新对应文档，否则视为改动未完成。动手前先按下表确认本次改动会触碰哪些文档面，与代码一起改。

| 改动类型 | 必须同步的文档 |
|----------|----------------|
| 前端使用方式 / 界面 / 快捷键 / 截图 | [`README.md`](../README.md)（必要时补截图） |
| 后端接口、请求体、响应字段、状态码 | [`API.md`](API.md) + `server/internal/handler/openapi_register_*.go`（并跑契约测试） |
| 错误码 / 错误文案 | [`ERRORS.md`](ERRORS.md) |
| **任何**新功能或 bug 修复 | [`CHANGELOG.md`](../CHANGELOG.md) 的 `[Unreleased]` 段（Keep a Changelog：Added / Fixed / Changed） |
| 已实现 / 已修复能力的台账 | [`FEATURES.md`](FEATURES.md) |
| 待办事项状态变化 | [`todolist.md`](todolist.md)（单一待办来源） |
| 版本级规划 / 优先级 | [`ROADMAP.md`](../ROADMAP.md)（不做逐条流水账） |
| 分层 / 模块边界 / 目录结构 | [`architecture.md`](architecture.md)；重大决策另加 [`decisions/`](decisions/index.md) ADR |
| 环境变量 / 配置项 | [`.env.example`](../.env.example) + [`deployment.md`](deployment.md) + `README.md` |
| 部署 / 镜像 / compose / 发布流程 | [`deployment.md`](deployment.md) |
| 安全策略 / 威胁模型 / 加固 | [`security.md`](security.md) 与 [`SECURITY.md`](../SECURITY.md) |
| 开发流程 / 门禁 / 测试命令 | 本文件 + [`CONTRIBUTING.md`](../CONTRIBUTING.md) + [`agents.md`](../agents.md) |

落地要求：

- **写清「为什么」**：文档记录用户可见行为、边界与决策依据，不复述 diff。
- **链接不悬空**：新增 / 移动文档时同步修复引用。
- **有意识地不更新**：若某类文档确实无需改动，在 PR 描述中写明原因。
- **文档改动也要验证**：命令、路径、示例需与实际一致（能跑就跑一遍）。

## 5. 验收清单（按 review 五轴）

提交前逐项核对并在 PR 描述中说明：

- **正确性**：符合需求；边界（空值/空列表/截断/大文件）覆盖；错误路径有测试；测试断言的是行为不是实现。
- **可读性**：命名贴近领域且与项目一致；控制流平直；无死代码 / 无 rest 兼容 shim。
- **架构**：沿用 `store → model → s3wrap → handler` 分层；`AWS` 类型不外泄到 handler/前端；**单文件不超过约 1000 行**，超过先拆再改；Feature 逻辑不进共享模块。
- **账号存储**：`S3C_STORE_DRIVER` 支持 `json`（默认）/ `sqlite` / `encrypted`；`store.Open` 统一入口。
- **安全**：用户输入在边界校验；敏感字段（`SecretKey`）不落地 localStorage、不写日志；输出转义（禁 `v-html`）；外部数据视为不可信。
- **性能**：列表端点分页；分段上传有界并发；无 N+1 / 无界循环。
- **文档**：本次改动涉及的文档已按 §4「文档同步门禁」更新（README / API.md / CHANGELOG / todolist 等），链接无死链，命令与路径实测一致。
- **验证**：测试绿 + 构建绿 + 涉及 UI 的保留截图/手测记录。

## 6. Red Flags（遇到即停下修正）

- 没看到红灯就直接写实现；
- 大而全的单文件改动（>1000 行）；
- 用 `any`/断言私有成员「掩盖」不清晰的不变量；
- 一次升级一批依赖 / 手改 lockfile；
- 功能逻辑渗入共享工具模块；
- 「以后再说」的清理不会发生——提交前就清干净；
- 改完代码不更新文档（README / API.md / CHANGELOG / todolist 等相关文档漏更，见 §4）；
- 为凑覆盖率而写与实现耦合的测试（gap 测试模式）。

## 7. 已知技术债（来自 2026-09-16 综合评估）

> 详见 [ASSESSMENT.md](ASSESSMENT.md) 与 [todolist.md](todolist.md)。开发时**避免扩大**以下模式：

- OpenAPI 注册表与真实 handler 字段不一致（改 handler 请求体时同步更新 `openapi_register_*.go`）。
- 复制粘贴式逻辑分叉（如 SSE 终态检测在 MigratePanel 与 useObjectActions 各一份）——优先收敛为共享实现。
- endpoint 归一化多份实现——改端点逻辑时统一到单一 helper。
