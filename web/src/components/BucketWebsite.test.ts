import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import BucketWebsite from './BucketWebsite.vue'
import { s3api } from '../api'
import { toast } from '../store'
import { confirmDialog } from '../confirm'

vi.mock('../api', () => ({
  s3api: {
    getBucketWebsite: vi.fn(async () => ({
      bucket: 'b1',
      configured: false,
      indexDocument: '',
      errorDocument: '',
      redirectAllRequestsTo: '',
    })),
    putBucketWebsite: vi.fn(async () => ({ configured: true })),
    deleteBucketWebsite: vi.fn(async () => ({ deleted: 'b1' })),
  },
}))
vi.mock('../store', () => ({ toast: vi.fn() }))
vi.mock('../confirm', () => ({ confirmDialog: vi.fn(async () => true) }))
vi.mock('../i18n', () => ({
  t: (k: string) => k,
  tf: (k: string) => k,
  locale: () => 'en-US',
}))

function mountWebsite() {
  return mount(BucketWebsite, { props: { accountId: 'acc-1', bucket: 'b1' } })
}

function saveBtn(w: ReturnType<typeof mountWebsite>) {
  const b = w.findAll('button').find((x) => x.text() === 'common.save')
  expect(b, 'save button should exist').toBeTruthy()
  return b!
}

function disableBtn(w: ReturnType<typeof mountWebsite>) {
  const b = w.findAll('button').find((x) => x.text() === 'website.disableBtn')
  expect(b, 'disable button should exist').toBeTruthy()
  return b!
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(s3api.getBucketWebsite).mockImplementation(async () => ({
    bucket: 'b1', configured: false, indexDocument: '', errorDocument: '', redirectAllRequestsTo: '',
  }))
  vi.mocked(s3api.putBucketWebsite).mockImplementation(async () => ({ configured: true }))
  vi.mocked(s3api.deleteBucketWebsite).mockImplementation(async () => ({ deleted: 'b1' }))
  vi.mocked(confirmDialog).mockImplementation(async () => true)
})

describe('BucketWebsite', () => {
  it('loading → 回显已配置的三个文档字段', async () => {
    vi.mocked(s3api.getBucketWebsite).mockResolvedValue({
      bucket: 'b1',
      configured: true,
      indexDocument: 'index.html',
      errorDocument: '404.html',
      redirectAllRequestsTo: 'https://example.com',
    })
    const w = mountWebsite()
    expect(w.text()).toContain('website.loading')
    await flushPromises()
    expect((w.find('input[placeholder="index.html"]').element as HTMLInputElement).value).toBe('index.html')
    expect((w.find('input[placeholder="website.errorDocPh"]').element as HTMLInputElement).value).toBe('404.html')
    expect((w.find('input[placeholder="website.redirectPh"]').element as HTMLInputElement).value).toBe('https://example.com')
    expect(w.text()).toContain('website.hint')
  })

  it('index 文档保存：put → toast → changed', async () => {
    const w = mountWebsite()
    await flushPromises()
    await w.find('input[placeholder="index.html"]').setValue('index.html')
    await w.find('input[placeholder="website.errorDocPh"]').setValue('404.html')
    await saveBtn(w).trigger('click')
    await flushPromises()
    expect(vi.mocked(s3api.putBucketWebsite)).toHaveBeenCalledWith('acc-1', {
      bucket: 'b1',
      indexDocument: 'index.html',
      errorDocument: '404.html',
      redirectAllRequestsTo: '',
    })
    expect(toast).toHaveBeenCalledWith('website.toastSaved')
    expect(w.emitted('changed')).toHaveLength(1)
  })

  it('redirect 单独保存也允许（else 分支）', async () => {
    const w = mountWebsite()
    await flushPromises()
    await w.find('input[placeholder="website.redirectPh"]').setValue('https://example.com')
    await saveBtn(w).trigger('click')
    await flushPromises()
    expect(vi.mocked(s3api.putBucketWebsite)).toHaveBeenCalled()
    expect(toast).toHaveBeenCalledWith('website.toastSaved')
  })

  it('index 与 redirect 均空：报错且不调用 put', async () => {
    const w = mountWebsite()
    await flushPromises()
    await saveBtn(w).trigger('click')
    await flushPromises()
    expect(w.emitted('error')).toEqual([['website.needIndexOrRedirect']])
    expect(vi.mocked(s3api.putBucketWebsite)).not.toHaveBeenCalled()
    expect(toast).not.toHaveBeenCalled()
  })

  it('disable：确认 → delete → toast → changed', async () => {
    const w = mountWebsite()
    await flushPromises()
    await disableBtn(w).trigger('click')
    expect(confirmDialog).toHaveBeenCalledWith(
      expect.objectContaining({ title: 'website.disableTitle', danger: true }),
    )
    expect(vi.mocked(s3api.deleteBucketWebsite)).toHaveBeenCalledWith('acc-1', 'b1')
    await flushPromises()
    expect(toast).toHaveBeenCalledWith('website.toastDisabled')
    expect(w.emitted('changed')).toHaveLength(1)
  })

  it('disable 取消：不调用 delete', async () => {
    vi.mocked(confirmDialog).mockResolvedValue(false)
    const w = mountWebsite()
    await flushPromises()
    await disableBtn(w).trigger('click')
    expect(vi.mocked(s3api.deleteBucketWebsite)).not.toHaveBeenCalled()
    expect(toast).not.toHaveBeenCalled()
  })

  it('load 失败 → emit error', async () => {
    vi.mocked(s3api.getBucketWebsite).mockRejectedValueOnce(new Error('load fail'))
    const w = mountWebsite()
    await flushPromises()
    expect(w.emitted('error')).toEqual([['load fail']])
  })

  it('put 失败 → emit error 且不 toast', async () => {
    vi.mocked(s3api.getBucketWebsite).mockResolvedValue({
      bucket: 'b1', configured: false, indexDocument: '', errorDocument: '', redirectAllRequestsTo: '',
    })
    vi.mocked(s3api.putBucketWebsite).mockRejectedValue(new Error('put fail'))
    const w = mountWebsite()
    await flushPromises()
    await w.find('input[placeholder="index.html"]').setValue('index.html')
    await saveBtn(w).trigger('click')
    await flushPromises()
    expect(w.emitted('error')).toEqual([['put fail']])
    expect(toast).not.toHaveBeenCalled()
  })
})
