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
 */
export const BATCH_META_CONCURRENCY = 4

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
}

export interface BatchMetaError {
  key: string
  step: 'acl' | 'tags' | 'storageClass'
  message: string
}

export interface BatchMetaResult {
  ok: number
  failed: number
  errors: BatchMetaError[]
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

  // 简易 worker-pool（4 路并发）。
  const queue = input.keys.slice()
  const allWorkers: Promise<BatchMetaResult>[] = [] // 收集所有 worker 以便聚合
  const inflight = new Set<Promise<BatchMetaResult>>()

  async function worker(key: string): Promise<BatchMetaResult> {
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

  async function spawn(): Promise<BatchMetaResult> {
    while (queue.length > 0) {
      if (inflight.size >= BATCH_META_CONCURRENCY) {
        await Promise.race(inflight)
        continue
      }
      const key = queue.shift()!
      const p = worker(key).finally(() => inflight.delete(p))
      inflight.add(p)
      allWorkers.push(p)
    }
    await Promise.all(inflight)
    // 聚合所有 worker 结果。
    let ok = 0
    let failed = 0
    const errors: BatchMetaError[] = []
    for (const p of allWorkers) {
      const r = await p
      ok += r.ok
      failed += r.failed
      errors.push(...r.errors)
    }
    return { ok, failed, errors }
  }

  return await spawn()
}
