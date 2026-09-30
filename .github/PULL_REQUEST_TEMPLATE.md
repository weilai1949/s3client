---
name: Pull Request
about: 提交代码变更
title: ''
labels: ''
assignees: ''
---

> 提交前请先读 **[贡献指南](CONTRIBUTING.md)**（分支 / 提交规范 / 开发环境）、**[开发规范](../docs/DEVELOPMENT.md)**（TDD 优先 / 门禁 / 验收清单 / Red Flags）与 **[AI 使用与代理治理政策](../docs/AI_POLICY.md)**（权限矩阵 / 披露要求 / DoD）。
> 改动涉及的文档必须与本 PR **同一个提交**同步更新，对照表见 [docs/DEVELOPMENT.md](../docs/DEVELOPMENT.md) §4。

## 变更内容

描述本次变更做了什么、为什么。

## 关联

- Closes #（issue 编号）

## 验收清单（按 [docs/DEVELOPMENT.md](../docs/DEVELOPMENT.md) 五轴）

- [ ] **正确性**：需求/边界/错误路径覆盖；测试断言行为而非实现
- [ ] **可读性**：命名规范；无死代码；无 rest 兼容 shim
- [ ] **架构**：沿用 `store → model → s3wrap → handler` 分层；单文件 ≤1000 行
- [ ] **安全**：用户输入在边界校验；敏感字段不落 localStorage / 不写日志；禁 `v-html`
- [ ] **性能**：分页；有界并发；无 N+1 / 无界循环
- [ ] **文档**：已按 [docs/DEVELOPMENT.md](../docs/DEVELOPMENT.md) §4 同步相关文档；`CHANGELOG.md` 的 `[Unreleased]` 已更新
- [ ] **AI 度量**：实质性 AI 贡献已按 [docs/AGENT_EVALS.md](../docs/AGENT_EVALS.md) §四 度量表回填一行（无 AI 贡献写「无」）
- [ ] **验证**：`go test ./...` + `pnpm build` 全绿；UI 改动附截图/手测记录

## 门禁（CI 自动检查）

- [ ] `go vet ./...` / `gofmt`
- [ ] 后端覆盖率 100%（`make test-cover`，profile 中不得有 `count==0` 语句块）/ 前端 ≥ 100%
- [ ] Docker 构建 + Trivy（CRITICAL/HIGH 失败）
- [ ] Playwright E2E（web 变更时）

## 测试

- 新增/修改测试：`xxx_test.go` / `xxx.test.ts`（列出）
- 本地验证命令与结果：

## 风险与回滚

- 风险：（本变更可能影响什么 / 最坏情况）
- 回滚：（如何退回，例如 `git revert`、镜像 tag 回指、数据备份位置）

## AI 使用披露

> 允许使用 AI 辅助，但**贡献者必须审查并理解**所提交的每一行改动；**实质性** AI 生成内容必须披露，
> 纯格式化 / 拼写修正可写「无」。详见 [docs/AI_POLICY.md](../docs/AI_POLICY.md) §5。

```text
AI 使用披露：
- 工具：<工具 / 模型名称，或「无」>
- 用途：<如生成初版测试用例、辅助重构、文档润色>
- 范围：<涉及文件或模块>
- 占比：<AI 生成 / 辅助比例的估计值，如 30–50%；纯格式化 / 拼写修正写 0>
- 人工审查：<已审查并验证；说明验证方式，如门禁实跑结果>
- 敏感数据：<未向外部模型发送任何密钥 / 用户数据 / 未公开源码>
- 许可证：<确认未引入第三方受限代码>
```

