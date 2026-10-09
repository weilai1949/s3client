# 贡献指南（Contributing）

感谢你愿意为 s3client 贡献！本仓库包含 Go 后端、Vue 前端与 Tauri 桌面壳。参与即表示你同意
[行为准则](CODE_OF_CONDUCT.md)；发现安全问题请走 [安全策略](SECURITY.md) 的私有渠道，不要开 public issue。

## 开发规范速览

- **TDD 优先**、必验门禁、验收清单、Red Flags：见 [docs/DEVELOPMENT.md](../docs/DEVELOPMENT.md)
- **文档同步**：修复 bug 或新增功能完成后**必须**更新相关文档（README / `docs/api.md` / `CHANGELOG.md` / `docs/KNOWN_ISSUES.md` 等），文档未同步视为改动未完成；对照表见 [docs/DEVELOPMENT.md](../docs/DEVELOPMENT.md) §4 与 [AGENTS.md](../AGENTS.md)
- **架构**与关键决策：见 [docs/architecture.md](../docs/architecture.md) 与 [docs/decisions/](../docs/decisions/index.md)
- **API 契约**：见 [docs/api.md](../docs/api.md)（与 OpenAPI 注册表一致；改 handler 请求体时同步更新 `openapi_register_*.go`）

## 分支与提交流程

采用 `develop`（主开发）→ `main`（稳定发布）工作流：

```bash
git checkout develop
git pull
git checkout -b feat/<你的功能>
# ... 开发（TDD：先红灯，再绿灯，再重构）...
cd apps/server && go vet ./... && go test ./...
cd apps/web && pnpm test && pnpm build
git commit -m "feat: ..."
```

## 开发环境

- Go 1.26+ / Node 26 / pnpm 9+ / Rust（桌面端）
- 后端本地启动 + 本机 RustFS 联调：见 [docs/DEVELOPMENT.md](../docs/DEVELOPMENT.md) §2「真实 RustFS 联调」
- 改到前端界面 / 后端接口 / 预签名直传时，额外跑 `make e2e-real`：真实 Go 后端 + 真实 RustFS + 真实构建产物的浏览器联调（自动 docker 起 RustFS、跑完清理；不 mock `/api`）

## 提交规范

- 使用 [Conventional Commits](https://www.conventionalcommits.org/zh-CN/)：`feat:`、`fix:`、`docs:`、`refactor:`、`test:` 等。
- 每个 commit 只做一件事；重构与功能分开。**成文例外**：同日并行、且共享同一组生成物 / 文档的多批改动可合并为一次提交，但提交信息必须按批切片（逐批列出改动点），详见 [docs/DEVELOPMENT.md](../docs/DEVELOPMENT.md) §1。
- 改动前先跑 `go test ./...`、`go vet ./...`、`pnpm build`（或 `make test-all`）。

## 发布流程

见 [docs/DEPLOYMENT.md](../docs/DEPLOYMENT.md) §5 与 `scripts/release-version.sh`（实测同步 **20 个文件**：
Makefile / Go main / Dockerfile / openapi.go / 两套 compose / npm ×2 / Cargo.toml / tauri.conf.json /
Cargo.lock（仅本包）/ README / docs（api·DEPLOYMENT·ROADMAP·FEATURES·**OPERATIONS**·**compatibility**）/
**`docs/api/openapi.json`** / issue 模板）。[`SECURITY.md`](SECURITY.md) 刻意不 pin 具体版本号
（只描述支持窗口与版本方案），故不在同步清单内。

## 联系与支持

支持渠道一览（含「本仓库不提供什么」）见 [SUPPORT.md](SUPPORT.md)；治理与发布权见 [GOVERNANCE.md](GOVERNANCE.md)。
下面是最常用的几步。本仓库**不使用邮件列表**，全部沟通走 GitHub：

| 场景 | 渠道 |
|---|---|
| 缺陷报告 | [新建 Bug Issue](https://github.com/weilai1949/s3client/issues/new?template=bug_report.md)（附版本 / 部署方式 / 复现步骤） |
| 功能建议 | [新建 Feature Issue](https://github.com/weilai1949/s3client/issues/new?template=feature_request.md) |
| 安全漏洞 | **不要开 public issue**——走 [SECURITY.md](SECURITY.md) 的私有漏洞报告渠道 |
| 行为准则投诉 | 见 [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) §执行；同样走私密渠道给维护者 |
| 维护者 | [@weilai1949](https://github.com/weilai1949) |
| 开发流程 / 门禁问题 | 先查 [docs/DEVELOPMENT.md](../docs/DEVELOPMENT.md)；仍不清楚再开 Issue |
| AI 辅助 / 代理权限问题 | 见 [docs/AI_POLICY.md](../docs/AI_POLICY.md) |
