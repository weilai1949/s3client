# s3clinet 待办清单（To-do List）

> 本文件汇总散落在各评估 / 快照文档中的**待处理（pending）**事项，作为后续迭代的单一待办来源。
> 「已完成 / 已评估」记录见 [`FEATURES.md`](FEATURES.md)；发版历史见 [`CHANGELOG.md`](../CHANGELOG.md)。
>
> 状态图例：⬜ 待办 · ⏳ 已排期 / 进行中 · ✅ 已完成 · ➖ 已决策（不做 / 维持现状）
>
> 最后更新：2026-09-15

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
| 2 | 契约里 `secretKey` 恒为 `"******"` 占位，可改 `secretSet: boolean` | `code-review-v1.0.0-rc1.md` §4 | ⬜ | 评审降级建议（原指控「回传明文 SecretKey」为误报，所有出口均 `Sanitized()`）。改为布尔字段可让客户端区分「已设置 / 未设置」，属于对外契约变更，需与前端联调后决定。 |
| 3 | OpenAPI `components.schemas` / `parameters` / `responses` 目前 **0 个 `$ref`** | `FEATURES.md` 已知边界 | ➖ | 已决策：接线 `$ref` 会改变对外契约，刻意不为凑引用而改动。保留片段供后续按需引用。 |
| 4 | `POST /api/accounts/preview-buckets` 使用临时凭据，不落库 | `FEATURES.md` 已知边界 | ➖ | 设计决策：预览桶仅用表单凭据临时 `ListBuckets`，不写入存储。 |

