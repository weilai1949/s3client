import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import BucketEncryption from './BucketEncryption.vue'
import { s3api } from '../api'
import { toast } from '../store'
import { confirmDialog } from '../confirm'

vi.mock('../api', () => ({
  s3api: {
    getBucketEncryption: vi.fn(async () => ({
      bucket: 'b1',
      configured: false,
      algorithm: 'AES256',
      kmsKeyId: '',
      bucketKeyEnabled: true,
    })),
    putBucketEncryption: vi.fn(async () => ({ configured: true, algorithm: 'AES256' })),
    deleteBucketEncryption: vi.fn(async () => ({ deleted: 'b1' })),
  },
}))
vi.mock('../store', () => ({ toast: vi.fn() }))
vi.mock('../confirm', () => ({ confirmDialog: vi.fn(async () => true) }))
vi.mock('../i18n', () => ({
  t: (k: string) => k,
  tf: (k: string) => k,
  locale: () => 'en-US',
}))

function mountEncryption() {
  return mount(BucketEncryption, { props: { accountId: 'acc-1', bucket: 'b1' } })
}

function saveBtn(w: ReturnType<typeof mountEncryption>) {
  const b = w.findAll('button').find((x) => x.text() === 'common.save')
  expect(b, 'save button should exist').toBeTruthy()
  return b!
}

function disableBtn(w: ReturnType<typeof mountEncryption>) {
  const b = w.findAll('button').find((x) => x.text() === 'encryption.disableBtn')
  expect(b, 'disable button should exist').toBeTruthy()
  return b!
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(s3api.getBucketEncryption).mockImplementation(async () => ({
    bucket: 'b1', configured: false, algorithm: 'AES256', kmsKeyId: '', bucketKeyEnabled: true,
  }))
  vi.mocked(s3api.putBucketEncryption).mockImplementation(async () => ({ configured: true, algorithm: 'AES256' }))
  vi.mocked(s3api.deleteBucketEncryption).mockImplementation(async () => ({ deleted: 'b1' }))
  vi.mocked(confirmDialog).mockImplementation(async () => true)
})

describe('BucketEncryption', () => {
  it('loading → AES256 默认渲染（无 KMS 输入）', async () => {
    const w = mountEncryption()
    expect(w.text()).toContain('encryption.loading')
    await flushPromises()
    const select = w.find('select')
    expect((select.element as HTMLSelectElement).value).toBe('AES256')
    expect(w.find('input[placeholder="encryption.kmsKeyPh"]').exists()).toBe(false)
    const bucketKey = w.find('input[type="checkbox"]')
    expect((bucketKey.element as HTMLInputElement).checked).toBe(true)
    expect(w.text()).toContain('encryption.hint')
  })

  it('algorithm 缺省(/空)时回退 AES256 渲染', async () => {
    vi.mocked(s3api.getBucketEncryption).mockResolvedValue({
      bucket: 'b1',
      configured: true,
      algorithm: undefined,
      kmsKeyId: undefined,
      bucketKeyEnabled: false,
    } as any)
    const w = mountEncryption()
    await flushPromises()
    // algorithm 回退 AES256 → KMS 输入框不显示
    expect((w.find('select').element as HTMLSelectElement).value).toBe('AES256')
    expect(w.find('input[placeholder="encryption.kmsKeyPh"]').exists()).toBe(false)
  })

  it('已配置 KMS：回显算法/KMS key/桶密钥开关', async () => {
    vi.mocked(s3api.getBucketEncryption).mockResolvedValue({
      bucket: 'b1',
      configured: true,
      algorithm: 'aws:kms',
      kmsKeyId: 'arn:aws:kms:us-east-1:123:key/abc',
      bucketKeyEnabled: false,
    } as any)
    const w = mountEncryption()
    await flushPromises()
    expect((w.find('select').element as HTMLSelectElement).value).toBe('aws:kms')
    expect((w.find('input[placeholder="encryption.kmsKeyPh"]').element as HTMLInputElement).value).toBe(
      'arn:aws:kms:us-east-1:123:key/abc',
    )
    expect((w.find('input[type="checkbox"]').element as HTMLInputElement).checked).toBe(false)
  })

  it('切换算法到 dsse 显示 KMS 输入并保存 payload', async () => {
    const w = mountEncryption()
    await flushPromises()
    await w.find('select').setValue('aws:kms:dsse')
    const kms = w.find('input[placeholder="encryption.kmsKeyPh"]')
    expect(kms.exists()).toBe(true)
    await kms.setValue('key/123')
    await w.find('input[type="checkbox"]').setValue(false)
    await saveBtn(w).trigger('click')
    await flushPromises()
    expect(vi.mocked(s3api.putBucketEncryption)).toHaveBeenCalledWith('acc-1', {
      bucket: 'b1',
      algorithm: 'aws:kms:dsse',
      kmsKeyId: 'key/123',
      bucketKeyEnabled: false,
    })
    expect(toast).toHaveBeenCalledWith('encryption.toastSaved')
    expect(w.emitted('changed')).toHaveLength(1)
  })

  it('disable：确认 → delete → toast → changed', async () => {
    vi.mocked(s3api.getBucketEncryption).mockResolvedValue({
      bucket: 'b1',
      configured: true,
      algorithm: 'aws:kms',
      kmsKeyId: 'key/1',
      bucketKeyEnabled: true,
    } as any)
    const w = mountEncryption()
    await flushPromises()
    await disableBtn(w).trigger('click')
    expect(confirmDialog).toHaveBeenCalledWith(
      expect.objectContaining({ title: 'encryption.disableTitle', danger: true }),
    )
    expect(vi.mocked(s3api.deleteBucketEncryption)).toHaveBeenCalledWith('acc-1', 'b1')
    await flushPromises()
    expect(toast).toHaveBeenCalledWith('encryption.toastDisabled')
    expect(w.emitted('changed')).toHaveLength(1)
  })

  it('disable 取消：不调用 delete', async () => {
    vi.mocked(confirmDialog).mockResolvedValue(false)
    const w = mountEncryption()
    await flushPromises()
    await disableBtn(w).trigger('click')
    expect(vi.mocked(s3api.deleteBucketEncryption)).not.toHaveBeenCalled()
    expect(toast).not.toHaveBeenCalled()
  })

  it('load 失败 → emit error', async () => {
    vi.mocked(s3api.getBucketEncryption).mockRejectedValueOnce(new Error('load fail'))
    const w = mountEncryption()
    await flushPromises()
    expect(w.emitted('error')).toEqual([['load fail']])
  })

  it('put 失败 → emit error 且不 toast', async () => {
    vi.mocked(s3api.getBucketEncryption).mockResolvedValue({
      bucket: 'b1', configured: false, algorithm: 'AES256', kmsKeyId: '', bucketKeyEnabled: true,
    } as any)
    vi.mocked(s3api.putBucketEncryption).mockRejectedValue(new Error('put fail'))
    const w = mountEncryption()
    await flushPromises()
    await saveBtn(w).trigger('click')
    await flushPromises()
    expect(w.emitted('error')).toEqual([['put fail']])
    expect(toast).not.toHaveBeenCalled()
  })
})
