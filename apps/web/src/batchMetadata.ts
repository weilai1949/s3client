import { s3api } from './api'
import { toErrorMessage } from './errors'

/**
 * 批量改对象元数据：bounded concurrency 的并发执行器，依次 PUT ACL/Tags/StorageClass。
 *
 * 设计取舍：
 *   - 后端没有「批量改 metadata」端点（与 S3 兼容层不匹配：每条对象要走
 *     CopyObject 副本 + REPLACE 元数据指令；逐对象执行天然适合并发）。
 *   - 4 路有界并发：与现有 copy/migrate 走同一并发上限（一致性）。
 *   - 不抛错：返回 { ok, failed, errors } 让调用方汇总展示，不阻断其它对象。
 *   - key 数量上限 BATCH_META_MAX_KEYS：超过则直接抛错（输入量级异常，属误用，
 *     由调用方兜底提示，避免一次性创建海量并发任务）。
 */
export const BATCH_META_CONCURRENCY = 4

/** 单次批量操作允许的最大 key 数量（防御误用：超大批量应拆分为多次）。 */
export const BATCH_META_MAX_KEYS = 10000

export interface BatchMetaInput {
  /** 当前账号 id */
  accountId: string
  /** 当前桶；与 per-key 的 bucket 二选一，per-key 缺省时回落到此 */
  bucket?: string
  /** 待改的对象 key 列表 */
  keys: string[]
  /** ACL：'private' | 'public-read' | 'public-read-write'；undefined=跳过 */
  acl?: 'private' | 'public-read' | 'public-read-write'
  /** 标签：数组形式；空数组=清空；undefined=跳过 */
  tags?: { key: string; value: string }[]
  /** 存储类型；undefined=跳过 */
  storageClass?: string
  /** 进度回调：每完成一个 key 调用一次（可选）；done 为已完成的 key 数，total=keys.length。 */
  onProgress?: (done: number, total: number) => void
}

export interface BatchMetaError {
  key: string
  step: 'acl' | 'tags' | 'storageClass' | 'batch'
  message: string
}

export interface BatchMetaResult {
  ok: number
  failed: number
  errors: BatchMetaError[]
}

/**
 * 有界并发执行器：按 concurrency 启动 worker，每个 worker 从共享队列取 item 处理。
 * 返回结果与输入 items 同序（按 index 填回），便于调用方聚合且顺序可预期。
 */
export function boundedPool<T, R>(
  items: T[],
  concurrency: number,
  worker: (item: T) => Promise<R>,
): Promise<R[]> {
  const results: R[] = new Array(items.length)
  let next = 0
  async function spawn(): Promise<void> {
    for (;;) {
      const i = next++
      if (i >= items.length) break
      results[i] = await worker(items[i])
    }
  }
  const n = Math.max(1, Math.min(concurrency, items.length))
  return Promise.all(Array.from({ length: n }, () => spawn())).then(() => results)
}

export async function batchSetMetadata(input: BatchMetaInput): Promise<BatchMetaResult> {
  // 预计算每个 key 的 steps，避免在循环里反复判断。
  const steps: Array<'acl' | 'tags' | 'storageClass'> = []
  if (input.acl !== undefined) steps.push('acl')
  if (input.tags !== undefined) steps.push('tags')
  if (input.storageClass !== undefined) steps.push('storageClass')
  if (steps.length === 0) {
    // 没有需要修改的字段：视为全部成功。
    return { ok: input.keys.length, failed: 0, errors: [] }
  }

  if (input.keys.length > BATCH_META_MAX_KEYS) {
    throw new Error(`too many keys: ${input.keys.length} > ${BATCH_META_MAX_KEYS}`)
  }

  // 单个 key 依次执行各 step；任一 step 失败即中止该 key 后续 step，并记为 1 个错误。
  async function processKey(key: string): Promise<BatchMetaResult> {
    for (const step of steps) {
      try {
        const body: { bucket?: string; key: string; acl?: string; tags?: { key: string; value: string }[]; storageClass?: string } = {
          bucket: input.bucket,
          key,
        }
        if (step === 'acl') body.acl = input.acl
        if (step === 'tags') body.tags = input.tags
        if (step === 'storageClass') body.storageClass = input.storageClass
        switch (step) {
          case 'acl':
            await s3api.putObjectAcl(input.accountId, body as { bucket?: string; key: string; acl: string })
            break
          case 'tags':
            await s3api.putObjectTags(input.accountId, body as { bucket?: string; key: string; tags: { key: string; value: string }[] })
            break
          case 'storageClass':
            await s3api.changeStorageClass(input.accountId, body as { bucket?: string; key: string; storageClass: string })
            break
        }
      } catch (e) {
        return { ok: 0, failed: 1, errors: [{ key, step, message: toErrorMessage(e) }] }
      }
    }
    return { ok: 1, failed: 0, errors: [] }
  }

  let done = 0
  const results = await boundedPool(input.keys, BATCH_META_CONCURRENCY, async (key) => {
    const r = await processKey(key)
    done++
    input.onProgress?.(done, input.keys.length)
    return r
  })

  // 聚合所有 worker 结果（顺序与 keys 一致）。
  let ok = 0
  let failed = 0
  const errors: BatchMetaError[] = []
  for (const r of results) {
    ok += r.ok
    failed += r.failed
    errors.push(...r.errors)
  }
  return { ok, failed, errors }
}
