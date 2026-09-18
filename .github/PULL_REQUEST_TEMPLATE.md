---
name: Pull Request
about: 提交代码变更
title: ''
labels: ''
assignees: ''
---

> 提交前请先读 **[贡献指南](CONTRIBUTING.md)**（分支 / 提交规范 / 开发环境）与 **[开发规范](../docs/development.md)**（TDD 优先 / 门禁 / 验收清单 / Red Flags）。
> 改动涉及的文档必须与本 PR **同一个提交**同步更新，对照表见 [docs/development.md](../docs/development.md) §4。

## 变更内容

描述本次变更做了什么、为什么。

## 关联

- Closes #（issue 编号）

## 验收清单（按 [docs/development.md](../docs/development.md) 五轴）

- [ ] **正确性**：需求/边界/错误路径覆盖；测试断言行为而非实现
- [ ] **可读性**：命名规范；无死代码；无 rest 兼容 shim
- [ ] **架构**：沿用 `store → model → s3wrap → handler` 分层；单文件 ≤1000 行
- [ ] **安全**：用户输入在边界校验；敏感字段不落 localStorage / 不写日志；禁 `v-html`
- [ ] **性能**：分页；有界并发；无 N+1 / 无界循环
- [ ] **验证**：`go test ./...` + `pnpm build` 全绿；UI 改动附截图/手测记录

## 门禁（CI 自动检查）

- [ ] `go vet ./...` / `gofmt`
- [ ] 后端覆盖率 100%（`make test-cover`，profile 中不得有 `count==0` 语句块）/ 前端 ≥ 100%
- [ ] Docker 构建 + Trivy（CRITICAL/HIGH 失败）
- [ ] Playwright E2E（web 变更时）

## 测试

- 新增/修改测试：`xxx_test.go` / `xxx.test.ts`（列出）
- 本地验证命令与结果：
