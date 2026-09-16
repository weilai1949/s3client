# Agents 开发规范

> ⚠️ **本文档已迁移**：开发规范（TDD 优先 / 测试分层 / 必验门禁 / 验收清单 / Red Flags / 技术债）已合并至
> **[docs/development.md](docs/development.md)**。
>
> 本文件保留为跳转入口，以兼容旧链接与既有引用；后续版本将删除。

## 强制规则：改完代码必须同步文档

**任何一次「修复 bug」或「新增功能」完成之后，必须更新相关文档。文档未同步 = 改动未完成**——
不得提交、不得合并、不得宣称完成。文档同步与代码改动属于**同一个 commit**（或同一 PR）。

### 按改动类型对照要更新的文档

| 改动类型 | 必须同步的文档 |
|----------|----------------|
| 前端使用方式 / 界面 / 快捷键 / 截图 | [`README.md`](README.md)（必要时补截图） |
| 后端接口、请求体、响应字段、状态码 | [`docs/API.md`](docs/API.md) + `server/internal/handler/openapi_register_*.go`（并跑契约测试） |
| 错误码 / 错误文案 | [`docs/ERRORS.md`](docs/ERRORS.md) |
| **任何**新功能或 bug 修复 | [`CHANGELOG.md`](CHANGELOG.md) 的 `[Unreleased]` 段（Keep a Changelog：Added / Fixed / Changed） |
| 已实现 / 已修复能力的台账 | [`docs/FEATURES.md`](docs/FEATURES.md) |
| 待办事项状态变化 | [`docs/todolist.md`](docs/todolist.md)（单一待办来源） |
| 版本级规划 / 优先级 | [`ROADMAP.md`](ROADMAP.md)（不做逐条流水账） |
| 分层 / 模块边界 / 目录结构 | [`docs/architecture.md`](docs/architecture.md)；重大决策另加 [`docs/decisions/`](docs/decisions/index.md) ADR |
| 环境变量 / 配置项 | [`.env.example`](.env.example) + [`docs/deployment.md`](docs/deployment.md) + `README.md` |
| 部署 / 镜像 / compose / 发布流程 | [`docs/deployment.md`](docs/deployment.md) |
| 安全策略 / 威胁模型 / 加固 | [`docs/security.md`](docs/security.md) 与 [`SECURITY.md`](SECURITY.md) |
| 开发流程 / 门禁 / 测试命令 | [`docs/development.md`](docs/development.md) + [`CONTRIBUTING.md`](CONTRIBUTING.md) + 本文件 |

### 落地要求

1. **先找文档面**：动手前先确认这次改动会触碰上表哪些文档，与代码一起改。
2. **写清「为什么」**：文档记录行为与决策依据（用户可见行为、边界、兼容性影响），不是复述 diff。
3. **链接不悬空**：新增 / 移动文档时同步修复引用，别留死链。
4. **收尾自检**：提交前逐项确认上表对应文档已更新；有意识地不更新时，在 PR 描述里写明原因。
5. **文档改动也要验证**：命令、路径、示例需与实际一致（能跑就跑一遍）。

> 完整规范与验收清单以 [`docs/development.md`](docs/development.md) 为准（本文档将随迁移计划删除）。
