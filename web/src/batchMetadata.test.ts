import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { batchSetMetadata, type BatchMetaInput } from './batchMetadata'

// 用 vi.mock 隔离 s3api；避免引入 happy-dom 与真 fetch。
const callsPutObjectAcl: Array<{ id: string; body: unknown }> = []
const callsPutObjectTags: Array<{ id: string; body: unknown }> = []
const callsChangeStorageClass: Array<{ id: string; body: unknown }> = []

vi.mock('./api', () => ({
  s3api: {
    putObjectAcl: vi.fn(async (id: string, body: unknown) => {
      callsPutObjectAcl.push({ id, body })
    }),
    putObjectTags: vi.fn(async (id: string, body: unknown) => {
      callsPutObjectTags.push({ id, body })
    }),
    changeStorageClass: vi.fn(async (id: string, body: unknown) => {
      callsChangeStorageClass.push({ id, body })
    }),
  },
}))

describe('batchSetMetadata', () => {
  beforeEach(() => {
    callsPutObjectAcl.length = 0
    callsPutObjectTags.length = 0
    callsChangeStorageClass.length = 0
  })
  afterEach(() => {
    vi.clearAllMocks()
  })

  it('runs all steps per key when all three are provided', async () => {
    const input: BatchMetaInput = {
      accountId: 'acc-1',
      bucket: 'b1',
      keys: ['a.txt', 'b.txt'],
      acl: 'public-read',
      tags: [{ key: 'k', value: 'v' }],
      storageClass: 'STANDARD_IA',
    }
    const out = await batchSetMetadata(input)
    expect(out.ok).toBe(2)
    expect(out.failed).toBe(0)
    expect(callsPutObjectAcl).toHaveLength(2)
    expect(callsPutObjectTags).toHaveLength(2)
    expect(callsChangeStorageClass).toHaveLength(2)
  })

  it('returns 0/0 when no steps specified (no-op)', async () => {
    const out = await batchSetMetadata({
      accountId: 'acc-1',
      keys: ['a.txt', 'b.txt'],
    })
    expect(out.ok).toBe(2)
    expect(out.failed).toBe(0)
    expect(callsPutObjectAcl).toHaveLength(0)
  })

  it('only runs the provided steps', async () => {
    const out = await batchSetMetadata({
      accountId: 'acc-1',
      keys: ['x'],
      acl: 'private',
    })
    expect(out.ok).toBe(1)
    expect(callsPutObjectAcl).toHaveLength(1)
    expect(callsPutObjectTags).toHaveLength(0)
    expect(callsChangeStorageClass).toHaveLength(0)
  })

  it('reports failure per (key, step) without aborting the batch', async () => {
    // 第一次 acl 抛错 → a.txt 标记失败但 b.txt 继续。
    const { s3api } = await import('./api')
    let n = 0
    vi.mocked(s3api.putObjectAcl).mockImplementation((async (_id, _body) => {
      n++
      if (n === 1) throw new Error('boom')
      return { acl: 'public-read' }
    }) as typeof s3api.putObjectAcl)
    const out = await batchSetMetadata({
      accountId: 'acc-1',
      keys: ['a.txt', 'b.txt'],
      acl: 'public-read',
    })
    expect(out.failed).toBe(1)
    expect(out.ok).toBe(1)
    expect(out.errors[0]).toMatchObject({ key: 'a.txt', step: 'acl', message: 'boom' })
  })

  it('clears tags when empty array provided', async () => {
    await batchSetMetadata({
      accountId: 'acc-1',
      keys: ['x'],
      tags: [],
    })
    expect(callsPutObjectTags).toHaveLength(1)
    const body = callsPutObjectTags[0].body as { tags: unknown[] }
    expect(body.tags).toEqual([])
  })
})
