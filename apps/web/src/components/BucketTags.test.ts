import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import BucketTags from './BucketTags.vue'
import { s3api } from '../api'
import { toast } from '../store'

vi.mock('../api', () => ({
  s3api: {
    getBucketTags: vi.fn(async () => ({ bucket: 'b1', tags: [] })),
    putBucketTags: vi.fn(async () => ({ updated: 1 })),
    deleteBucketTags: vi.fn(async () => ({ deleted: 'b1' })),
  },
}))
vi.mock('../store', () => ({ toast: vi.fn() }))
vi.mock('../i18n', () => ({
  t: (k: string) => k,
  tf: (k: string) => k,
  locale: () => 'en-US',
}))

function mountTags() {
  return mount(BucketTags, { props: { accountId: 'acc-1', bucket: 'b1' } })
}

function addBtn(w: ReturnType<typeof mountTags>) {
  const b = w.findAll('button').find((x) => x.text() === 'bucketTags.add')
  expect(b, 'add button should exist').toBeTruthy()
  return b!
}

function saveBtn(w: ReturnType<typeof mountTags>) {
  const b = w.findAll('button').find((x) => x.text() === 'common.save')
  expect(b, 'save button should exist').toBeTruthy()
  return b!
}

function clearBtn(w: ReturnType<typeof mountTags>) {
  const b = w.findAll('button').find((x) => x.text() === 'bucketTags.clearAll')
  expect(b, 'clear button should exist').toBeTruthy()
  return b!
}

function rowInputs(w: ReturnType<typeof mountTags>, i: number) {
  const row = w.findAll('tbody tr')[i]!
  const inputs = row.findAll('input')
  return { key: inputs[0]!, value: inputs[1]! }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(s3api.getBucketTags).mockImplementation(async () => ({ bucket: 'b1', tags: [] }))
  vi.mocked(s3api.putBucketTags).mockImplementation(async () => ({ updated: 1 }))
  vi.mocked(s3api.deleteBucketTags).mockImplementation(async () => ({ deleted: 'b1' }))
})

describe('BucketTags', () => {
  it('loading → 渲染已有标签行', async () => {
    vi.mocked(s3api.getBucketTags).mockResolvedValueOnce({
      bucket: 'b1',
      tags: [
        { key: 'env', value: 'prod' },
        { key: 'team', value: 'platform' },
      ],
    })
    const w = mountTags()
    expect(w.text()).toContain('bucketTags.loading')
    await flushPromises()
    const rows = w.findAll('tbody tr')
    expect(rows).toHaveLength(2)
    expect((rowInputs(w, 0).key.element as HTMLInputElement).value).toBe('env')
    expect((rowInputs(w, 0).value.element as HTMLInputElement).value).toBe('prod')
  })

  it('add / remove 行', async () => {
    const w = mountTags()
    await flushPromises()
    expect(w.findAll('tbody tr')).toHaveLength(0)
    await addBtn(w).trigger('click')
    expect(w.findAll('tbody tr')).toHaveLength(1)
    await addBtn(w).trigger('click')
    await w.findAll('tbody tr')[0]!.find('button').trigger('click')
    expect(w.findAll('tbody tr')).toHaveLength(1)
  })

  it('保存：trim key → put → toast → changed', async () => {
    vi.mocked(s3api.getBucketTags).mockResolvedValue({
      bucket: 'b1',
      tags: [{ key: 'env', value: 'prod' }],
    })
    const w = mountTags()
    await flushPromises()
    await rowInputs(w, 0).key.setValue('  env  ')
    await rowInputs(w, 0).value.setValue('prod')
    await saveBtn(w).trigger('click')
    await flushPromises()
    expect(vi.mocked(s3api.putBucketTags)).toHaveBeenCalledWith('acc-1', {
      bucket: 'b1',
      tags: [{ key: 'env', value: 'prod' }],
    })
    expect(toast).toHaveBeenCalledWith('bucketTags.toastSaved')
    expect(w.emitted('changed')).toHaveLength(1)
  })

  it('全空 key 行：提交空 tags 并成功', async () => {
    vi.mocked(s3api.getBucketTags).mockResolvedValue({ bucket: 'b1', tags: [] })
    const w = mountTags()
    await flushPromises()
    await addBtn(w).trigger('click')
    await saveBtn(w).trigger('click')
    await flushPromises()
    expect(vi.mocked(s3api.putBucketTags)).toHaveBeenCalledWith('acc-1', {
      bucket: 'b1',
      tags: [],
    })
    expect(toast).toHaveBeenCalledWith('bucketTags.toastSaved')
    expect(w.emitted('changed')).toHaveLength(1)
  })

  it('clear：delete → toast → 表格清空 → changed', async () => {
    // 首次载入有 1 条，clear 后的 reload 返回空列表
    let seen = 0
    vi.mocked(s3api.getBucketTags).mockImplementation(async () => ({
      bucket: 'b1',
      tags: seen++ === 0 ? [{ key: 'env', value: 'prod' }] : [],
    }))
    const w = mountTags()
    await flushPromises()
    expect(w.findAll('tbody tr')).toHaveLength(1)
    await clearBtn(w).trigger('click')
    expect(vi.mocked(s3api.deleteBucketTags)).toHaveBeenCalledWith('acc-1', 'b1')
    await flushPromises()
    expect(toast).toHaveBeenCalledWith('bucketTags.toastCleared')
    expect(w.findAll('tbody tr')).toHaveLength(0)
    expect(w.emitted('changed')).toHaveLength(1)
  })

  it('load 失败 → emit error', async () => {
    vi.mocked(s3api.getBucketTags).mockRejectedValueOnce(new Error('load fail'))
    const w = mountTags()
    await flushPromises()
    expect(w.emitted('error')).toEqual([['load fail']])
  })

  it('tags 缺省(null/undefined)→ `(r.tags ?? [])` 兜底渲染空表', async () => {
    vi.mocked(s3api.getBucketTags).mockResolvedValue({ bucket: 'b1' } as unknown as Awaited<ReturnType<typeof s3api.getBucketTags>>)
    const w = mountTags()
    await flushPromises()
    expect(w.emitted('error')).toBeUndefined()
    expect(w.findAll('tbody tr')).toHaveLength(0)
    expect(w.text()).not.toContain('bucketTags.loading')
  })

  it('put 失败 → emit error 且不 toast', async () => {
    vi.mocked(s3api.getBucketTags).mockResolvedValue({
      bucket: 'b1',
      tags: [{ key: 'env', value: 'prod' }],
    })
    vi.mocked(s3api.putBucketTags).mockRejectedValue(new Error('put fail'))
    const w = mountTags()
    await flushPromises()
    await saveBtn(w).trigger('click')
    await flushPromises()
    expect(w.emitted('error')).toEqual([['put fail']])
    expect(toast).not.toHaveBeenCalled()
  })
})
