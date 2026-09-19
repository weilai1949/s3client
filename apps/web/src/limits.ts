/**
 * 服务端批量接口的硬上限（前端契约常量）。
 *
 * 这些数字来自 `apps/server/internal/handler`：超过上限的请求会被直接 400 拒绝
 * （`too many keys (max …)`）。前端选中量可以远超上限（对象列表 loadAll 2 万、
 * 迁移面板 20 万、`selectAll` 可全选），因此**提交前必须分片**——否则整批失败、
 * 一个都没执行（见 docs/review-2026-09-19.md §F3）。
 *
 * 放在共享模块而不是各自散落的魔数：上限变更时只改一处。
 */

/** `POST /api/accounts/{id}/delete` 单次提交的 key 上限（S3 DeleteObjects 本身一次最多 1000）。 */
export const DELETE_MAX_KEYS_PER_REQUEST = 1000

/** `POST /api/accounts/{id}/migrate` 单次提交的 key 上限。 */
export const MIGRATE_MAX_KEYS_PER_REQUEST = 10_000

/** 按服务端上限把 key 列表切成串行提交的分片；顺序保持与输入一致。 */
export function batchKeys(keys: string[], maxPerRequest: number): string[][] {
  const batches: string[][] = []
  for (let i = 0; i < keys.length; i += maxPerRequest) {
    batches.push(keys.slice(i, i + maxPerRequest))
  }
  return batches
}
