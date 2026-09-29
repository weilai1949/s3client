# GitHub Copilot 仓库指令（**指针，不复制规则**）

> 本文件只做一件事：把 Copilot 指到**唯一事实来源**。仓库规则一律不在此复述——复制会产生第二事实源
> 并随之漂移。该约束由 `apps/server/ai_governance_gate_test.go` 机械校验（体积上限 + 必须指向 `AGENTS.md`）。

- **仓库级硬约束**（必须先读；支持 `AGENTS.md` 约定的工具会自动注入）：[`../AGENTS.md`](../AGENTS.md)
- **规范正文**（TDD / 三类测试 / 门禁 / 验收清单 / 文档同步门禁 / 命名与归档）：[`../docs/DEVELOPMENT.md`](../docs/DEVELOPMENT.md)
- **代理权限边界**（五档模式、什么必须人类确认、AI 披露要求、DoD）：[`../docs/AI_POLICY.md`](../docs/AI_POLICY.md)
- **子树规则**（按你改动的目录读）：[`apps/server`](../apps/server/AGENTS.md) · [`apps/web`](../apps/web/AGENTS.md) · [`apps/desktop`](../apps/desktop/AGENTS.md)
- **文档导航**：[`../docs/README.md`](../docs/README.md) · **LLM 扁平索引**：[`../llms.txt`](../llms.txt)

提交前门禁（完整清单见根 `AGENTS.md`）：

```bash
cd apps/server && go vet ./... && go test ./... && go build ./...
cd apps/web && pnpm test && pnpm build
```
