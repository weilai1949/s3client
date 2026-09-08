import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import BucketCors from './BucketCors.vue'
import { s3api } from '../api'
import { toast } from '../store'

vi.mock('../api', () => ({
  s3api: {
    getBucketCors: vi.fn(async () => ({ bucket: 'b1', rules: [] })),
    putBucketCors: vi.fn(async () => ({ updated: 1 })),
    deleteBucketCors: vi.fn(async () => ({ deleted: 'b1' })),
  },
}))
vi.mock('../store', () => ({ toast: vi.fn() }))
vi.mock('../i18n', () => ({
  t: (k: string) => k,
  tf: (k: string) => k,
  locale: () => 'en-US',
}))

const sampleRules = [
  {
    id: 'r1',
    allowedMethods: ['GET', 'POST'],
    allowedOrigins: ['https://a.com'],
    allowedHeaders: ['x-a', 'x-b'],
    exposeHeaders: ['ETag'],
    maxAgeSeconds: 600,
  },
]

function mountCors() {
  return mount(BucketCors, { props: { accountId: 'acc-1', bucket: 'b1' } })
}

function ruleBox(w: ReturnType<typeof mountCors>, i = 0) {
  return w.findAll('.cors-rule')[i]!
}

function chip(w: ReturnType<typeof mountCors>, i: number, method: string) {
  const label = ruleBox(w, i).findAll('.chip').find((l) => l.text().trim() === method)
  expect(label, `chip ${method} should exist`).toBeTruthy()
  return label!.find('input')
}

function saveBtn(w: ReturnType<typeof mountCors>) {
  const b = w.findAll('button').find((x) => x.text() === 'common.save')
  expect(b, 'save button should exist').toBeTruthy()
  return b!
}

function clearBtn(w: ReturnType<typeof mountCors>) {
  const b = w.findAll('button').find((x) => x.text() === 'cors.clearAll')
  expect(b, 'clear button should exist').toBeTruthy()
  return b!
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(s3api.getBucketCors).mockImplementation(async () => ({ bucket: 'b1', rules: [] }))
  vi.mocked(s3api.putBucketCors).mockImplementation(async () => ({ updated: 1 }))
  vi.mocked(s3api.deleteBucketCors).mockImplementation(async () => ({ deleted: 'b1' }))
})

describe('BucketCors', () => {
  it('loading → 无规则时自动补一条默认规则', async () => {
    const w = mountCors()
    expect(w.text()).toContain('cors.loading')
    await flushPromises()
    expect(w.findAll('.cors-rule')).toHaveLength(1)
    const box = ruleBox(w)
    expect((box.find('input[placeholder="cors.ruleIdPh"]').element as HTMLInputElement).value).toBe('')
    expect((chip(w, 0, 'GET').element as HTMLInputElement).checked).toBe(true)
    expect((chip(w, 0, 'POST').element as HTMLInputElement).checked).toBe(false)
    expect((box.find('input[placeholder="cors.originsPh"]').element as HTMLInputElement).value).toBe('*')
    expect((box.find('input[placeholder="cors.headersPh"]').element as HTMLInputElement).value).toBe('*')
    expect((box.find('input[type="number"]').element as HTMLInputElement).value).toBe('3600')
  })

  it('渲染已有规则（joinList 回显）', async () => {
    vi.mocked(s3api.getBucketCors).mockResolvedValue({ bucket: 'b1', rules: sampleRules } as any)
    const w = mountCors()
    await flushPromises()
    const box = ruleBox(w)
    expect((box.find('input[placeholder="cors.ruleIdPh"]').element as HTMLInputElement).value).toBe('r1')
    expect((box.find('input[placeholder="cors.originsPh"]').element as HTMLInputElement).value).toBe('https://a.com')
    expect((box.find('input[placeholder="cors.headersPh"]').element as HTMLInputElement).value).toBe('x-a, x-b')
    expect((box.find('input[placeholder="cors.exposePh"]').element as HTMLInputElement).value).toBe('ETag')
    expect((box.find('input[type="number"]').element as HTMLInputElement).value).toBe('600')
    expect((chip(w, 0, 'GET').element as HTMLInputElement).checked).toBe(true)
    expect((chip(w, 0, 'POST').element as HTMLInputElement).checked).toBe(true)
    expect((chip(w, 0, 'PUT').element as HTMLInputElement).checked).toBe(false)
  })

  it('toggle 方法：取消 GET、勾选 HEAD → 保存 payload', async () => {
    vi.mocked(s3api.getBucketCors).mockResolvedValue({ bucket: 'b1', rules: sampleRules } as any)
    const w = mountCors()
    await flushPromises()
    await chip(w, 0, 'GET').setValue(false)
    await chip(w, 0, 'HEAD').setValue(true)
    await saveBtn(w).trigger('click')
    await flushPromises()
    expect(vi.mocked(s3api.putBucketCors)).toHaveBeenCalledWith('acc-1', {
      bucket: 'b1',
      rules: [expect.objectContaining({ allowedMethods: ['POST', 'HEAD'] })],
    })
    expect(toast).toHaveBeenCalledWith('cors.toastSaved')
    expect(w.emitted('changed')).toHaveLength(1)
  })

  it('列表输入 split/join、maxAge 数字绑定 → 保存 payload', async () => {
    vi.mocked(s3api.getBucketCors).mockResolvedValue({ bucket: 'b1', rules: [] } as any)
    const w = mountCors()
    await flushPromises()
    const box = ruleBox(w)
    await box.find('input[placeholder="cors.originsPh"]').setValue('https://a.com, https://b.com')
    await box.find('input[placeholder="cors.exposePh"]').setValue('ETag, x-version')
    await box.find('input[type="number"]').setValue('7200')
    expect((box.find('input[placeholder="cors.originsPh"]').element as HTMLInputElement).value).toBe(
      'https://a.com, https://b.com',
    )
    await saveBtn(w).trigger('click')
    await flushPromises()
    expect(vi.mocked(s3api.putBucketCors)).toHaveBeenCalledWith('acc-1', {
      bucket: 'b1',
      rules: [
        expect.objectContaining({
          allowedOrigins: ['https://a.com', 'https://b.com'],
          exposeHeaders: ['ETag', 'x-version'],
          maxAgeSeconds: 7200, // v-model.number
        }),
      ],
    })
  })

  it('规则 id 与 allowedHeaders 输入 v-model 绑定 → 保存 payload', async () => {
    vi.mocked(s3api.getBucketCors).mockResolvedValue({ bucket: 'b1', rules: [] } as any)
    const w = mountCors()
    await flushPromises()
    const box = ruleBox(w)
    await box.find('input[placeholder="cors.ruleIdPh"]').setValue('rule-1')
    await box.find('input[placeholder="cors.headersPh"]').setValue('x-a, x-b')
    await saveBtn(w).trigger('click')
    await flushPromises()
    expect(vi.mocked(s3api.putBucketCors)).toHaveBeenCalledWith('acc-1', {
      bucket: 'b1',
      rules: [expect.objectContaining({ id: 'rule-1', allowedHeaders: ['x-a', 'x-b'] })],
    })
  })

  it('add / remove 规则', async () => {
    vi.mocked(s3api.getBucketCors).mockResolvedValue({ bucket: 'b1', rules: [] } as any)
    const w = mountCors()
    await flushPromises()
    await w.findAll('button').find((x) => x.text() === 'cors.addRule')!.trigger('click')
    expect(w.findAll('.cors-rule')).toHaveLength(2)
    await ruleBox(w, 0).find('button').trigger('click')
    expect(w.findAll('.cors-rule')).toHaveLength(1)
    // 删除最后一条 → 空列表
    await ruleBox(w, 0).find('button').trigger('click')
    expect(w.findAll('.cors-rule')).toHaveLength(0)
  })

  it('clear：delete → toast → 重置为默认规则 → changed', async () => {
    vi.mocked(s3api.getBucketCors).mockResolvedValue({ bucket: 'b1', rules: sampleRules } as any)
    const w = mountCors()
    await flushPromises()
    await clearBtn(w).trigger('click')
    expect(vi.mocked(s3api.deleteBucketCors)).toHaveBeenCalledWith('acc-1', 'b1')
    await flushPromises()
    expect(toast).toHaveBeenCalledWith('cors.toastCleared')
    expect(w.findAll('.cors-rule')).toHaveLength(1) // clear 后 addRule 兜底
    expect(w.emitted('changed')).toHaveLength(1)
  })

  it('load 失败 → emit error', async () => {
    vi.mocked(s3api.getBucketCors).mockRejectedValueOnce(new Error('load fail'))
    const w = mountCors()
    await flushPromises()
    expect(w.emitted('error')).toEqual([['load fail']])
  })

  it('put 失败 → emit error 且不 toast', async () => {
    vi.mocked(s3api.getBucketCors).mockResolvedValue({ bucket: 'b1', rules: [] } as any)
    vi.mocked(s3api.putBucketCors).mockRejectedValue(new Error('put fail'))
    const w = mountCors()
    await flushPromises()
    await saveBtn(w).trigger('click')
    await flushPromises()
    expect(w.emitted('error')).toEqual([['put fail']])
    expect(toast).not.toHaveBeenCalled()
  })

  it('load 响应缺 rules 键 → 默认补一条规则', async () => {
    vi.mocked(s3api.getBucketCors).mockResolvedValue({ bucket: 'b1' } as any)
    const w = mountCors()
    await flushPromises()
    expect(w.findAll('.cors-rule')).toHaveLength(1)
  })

  it('规则缺 allowedMethods/allowedOrigins 字段时归一化为空数组', async () => {
    vi.mocked(s3api.getBucketCors).mockResolvedValue({
      bucket: 'b1',
      rules: [{ id: 'r1' }],
    } as any)
    const w = mountCors()
    await flushPromises()
    const box = ruleBox(w)
    expect((box.find('input[placeholder="cors.originsPh"]').element as HTMLInputElement).value).toBe('')
    expect((box.find('input[placeholder="cors.headersPh"]').element as HTMLInputElement).value).toBe('')
    expect((chip(w, 0, 'GET').element as HTMLInputElement).checked).toBe(false)
  })

  it('joinList 对 undefined 回退为空串', () => {
    const w = mountCors()
    expect((w.vm as any).joinList(undefined)).toBe('')
  })
})
