# s3clinet 待办清单（To-do List）

> 本文件汇总散落在各评估 / 快照文档中的**待处理（pending）**事项，作为后续迭代的单一待办来源。
> 「已完成 / 已评估」记录见 [`FEATURES.md`](FEATURES.md)；发版历史见 [`CHANGELOG.md`](../CHANGELOG.md)。
>
> 状态图例：⬜ 待办 · ⏳ 已排期 / 进行中 · ✅ 已完成 · ➖ 已决策（不做 / 维持现状）
>
> 最后更新：2026-09-16

## 目录

- [一、功能 / 架构待办](#一功能--架构待办)
- [二、API / 契约待办](#二api--契约待办)
---

## 一、功能 / 架构待办

| # | 项 | 来源 | 状态 | 说明 |
|---|----|------|------|------|
| 1 | `EncryptedStore` 与「带 `S3C_STORE_KEY` 的 JSON Store」功能重叠 | `full-assessment.md` T-8 注 | ✅ | 已收敛：两驱动统一为 `Store` + 单一 `storeCodec`（strict 区分 json/encrypted），删除 `EncryptedStore` / `encryptedCodec`；磁盘格式与错误文案不变，`internal/store` 覆盖率 100%。 |

---

## 二、API / 契约待办

| # | 项 | 来源 | 状态 | 说明 |
|---|----|------|------|------|
| 2 | 契约里 `secretKey` 恒为 `"******"` 占位，可改 `secretSet: boolean` | `code-review-v1.0.0-rc1.md` §4 | ✅ | 已收敛：响应改为 `AccountView`（无 `secretKey`，新增 `secretSet: boolean`）；请求仍用 `secretKey` 提交。后端/前端/OpenAPI/文档同步更新，`internal/model` 覆盖率 100%。 |
| 3 | OpenAPI `components.schemas` / `parameters` / `responses` 目前 **0 个 `$ref`** | `FEATURES.md` 已知边界 | ✅ | 已接线：新增 `openapi.Ref()` 与 `Param.Ref` / `Response.Ref`，共享 schema / parameter / response 全部通过 `refSchema` / `refParam` / `refResp` 接线为 `$ref`（109 处引用、12 个唯一目标）；契约测试同步支持 `$ref` 解析，并新增「components 无死片段」断言；`internal/openapi` 覆盖率保持 100%。 |
| 4 | `POST /api/accounts/preview-buckets` 使用临时凭据，不落库 | `FEATURES.md` 已知边界 | ✅ | 已确认并补文档：仅用表单临时凭据 `ListBuckets`、不写入存储（请求体已细化，400 缺字段/500 上游失败均有测试）；FEATURES/API 描述已同步。 |

