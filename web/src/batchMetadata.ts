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
  acl?: string
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
  const accId = input.accountId
  const errors: BatchMetaError[] = []
  let ok = 0
  let failed = 0

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
  const inflight = new Set<Promise<void>>()

  async function worker(key: string): Promise<void> {
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
            await s3api.putObjectAcl(accId, body as { bucket?: string; key: string; acl: string })
            break
          case 'tags':
            await s3api.putObjectTags(accId, body as { bucket?: string; key: string; tags: { key: string; value: string }[] })
            break
          case 'storageClass':
            await s3api.changeStorageClass(accId, body as { bucket?: string; key: string; storageClass: string })
            break
        }
      } catch (e) {
        failed++
        errors.push({ key, step, message: toErrorMessage(e) })
        // 单条失败立即中断后续 step（保持对象状态一致）。
        return
      }
    }
    ok++
  }

  async function spawn(): Promise<void> {
    while (queue.length > 0) {
      if (inflight.size >= BATCH_META_CONCURRENCY) {
        await Promise.race(inflight)
        continue
      }
      const key = queue.shift()!
      const p = worker(key).finally(() => inflight.delete(p))
      inflight.add(p)
    }
    await Promise.all(inflight)
  }

  await spawn()
  return { ok, failed, errors }
}
