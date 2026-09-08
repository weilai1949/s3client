import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import BucketInfoDialog from './BucketInfoDialog.vue'
import { s3api } from '../api'
import { toast } from '../store'
import { fmtDate } from '../format'

vi.mock('../api', () => ({
  s3api: {
    getBucketInfo: vi.fn(async () => ({
      bucket: 'b1',
      region: 'us-east-1',
      createdAt: '2024-01-02T00:00:00Z',
      versioning: '',
    })),
    putBucketVersioning: vi.fn(async () => ({ versioning: 'Enabled' })),
  },
}))
vi.mock('../store', () => ({ toast: vi.fn() }))
vi.mock('../i18n', () => ({
  t: (k: string) => k,
  tf: (k: string) => k,
  locale: () => 'en-US',
}))

// ModalDialog 用 stub 渲染默认插槽，避免 Teleport 干扰断言
const ModalDialogStub = {
  name: 'ModalDialog',
  template: '<div><slot /></div>',
}

function mountDialog() {
  return mount(BucketInfoDialog, {
    props: { open: false, accountId: 'acc-1', bucket: 'b1' },
    global: { stubs: { ModalDialog: ModalDialogStub } },
  })
}

function toggleBtn(w: ReturnType<typeof mountDialog>) {
  return w.findAll('button').find(
    (b) =>
      b.text() === 'overview.enable' ||
      b.text() === 'overview.suspend' ||
      b.text() === 'buckets.processing',
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(s3api.getBucketInfo).mockImplementation(async () => ({
    bucket: 'b1', region: 'us-east-1', createdAt: '2024-01-02T00:00:00Z', versioning: '',
  }))
  vi.mocked(s3api.putBucketVersioning).mockImplementation(async () => ({ versioning: 'Enabled' }))
})

describe('BucketInfoDialog', () => {
  it('open=true 触发加载：loading → 字段渲染', async () => {
    let resolveInfo!: (v: unknown) => void
    vi.mocked(s3api.getBucketInfo).mockImplementationOnce(
      () => new Promise((r) => { resolveInfo = r }) as any,
    )
    const w = mountDialog()
    // 关闭状态不渲染内容
    expect(w.text().includes('overview.loading')).toBe(false)
    await w.setProps({ open: true })
    expect(w.text()).toContain('overview.loading')
    resolveInfo({
      bucket: 'b1',
      region: 'us-east-1',
      createdAt: '2024-01-02T00:00:00Z',
      versioning: '',
    })
    await flushPromises()
    const text = w.text()
    expect(text).toContain('b1')
    expect(text).toContain('us-east-1')
    expect(text).toContain(fmtDate('2024-01-02T00:00:00Z'))
    expect(text).toContain('buckets.verUnset')
    expect(toggleBtn(w)!.text()).toBe('overview.enable')
    expect(text).toContain('common.close')
  })

  it('close 按钮与 ModalDialog close 事件转发', async () => {
    const w = mountDialog()
    await w.setProps({ open: true })
    await flushPromises()
    const closeBtn = w.findAll('button').find((b) => b.text() === 'common.close')!
    await closeBtn.trigger('click')
    expect(w.emitted('close')).toHaveLength(1)
    // ModalDialog 自身的 close 事件（用户点遮罩/X）
    const modal = w.findComponent({ name: 'ModalDialog' })
    modal.vm.$emit('close')
    expect(w.emitted('close')).toHaveLength(2)
  })

  it('toggle 启用：put → toast → 重新加载并更新状态', async () => {
    let n = 0
    vi.mocked(s3api.getBucketInfo).mockImplementation(async () =>
      n++ === 0
        ? { bucket: 'b1', region: 'us-east-1', createdAt: '', versioning: '' }
        : { bucket: 'b1', region: 'us-east-1', createdAt: '', versioning: 'Enabled' },
    )
    const w = mountDialog()
    await w.setProps({ open: true })
    await flushPromises()
    await toggleBtn(w)!.trigger('click')
    expect(vi.mocked(s3api.putBucketVersioning)).toHaveBeenCalledWith('acc-1', {
      bucket: 'b1',
      status: 'Enabled',
    })
    await flushPromises()
    expect(toast).toHaveBeenCalledWith('buckets.toastVerEnabled')
    expect(w.text()).toContain('buckets.verEnabled')
    expect(toggleBtn(w)!.text()).toBe('overview.suspend')
  })

  it('Suspended 状态渲染，suspend 切换 → toast', async () => {
    let n = 0
    vi.mocked(s3api.getBucketInfo).mockImplementation(async () =>
      n++ === 0
        ? { bucket: 'b1', region: 'us-east-1', createdAt: '', versioning: 'Enabled' }
        : { bucket: 'b1', region: 'us-east-1', createdAt: '', versioning: 'Suspended' },
    )
    const w = mountDialog()
    await w.setProps({ open: true })
    await flushPromises()
    expect(w.text()).toContain('buckets.verEnabled')
    expect(toggleBtn(w)!.text()).toBe('overview.suspend')
    await toggleBtn(w)!.trigger('click')
    expect(vi.mocked(s3api.putBucketVersioning)).toHaveBeenCalledWith('acc-1', {
      bucket: 'b1',
      status: 'Suspended',
    })
    await flushPromises()
    expect(toast).toHaveBeenCalledWith('buckets.toastVerSuspended')
    expect(w.text()).toContain('buckets.verSuspended')
    expect(toggleBtn(w)!.text()).toBe('overview.enable')
  })

  it('createdAt 为空显示占位', async () => {
    vi.mocked(s3api.getBucketInfo).mockResolvedValue({
      bucket: 'b1',
      region: '',
      createdAt: '',
      versioning: 'Suspended',
    } as any)
    const w = mountDialog()
    await w.setProps({ open: true })
    await flushPromises()
    expect(w.text()).toContain('—')
    expect(w.text()).toContain('buckets.verSuspended')
  })

  it('load 失败 → emit error', async () => {
    vi.mocked(s3api.getBucketInfo).mockRejectedValueOnce(new Error('load fail'))
    const w = mountDialog()
    await w.setProps({ open: true })
    await flushPromises()
    expect(w.emitted('error')).toEqual([['load fail']])
  })

  it('put 失败 → emit error 且 saving 复位', async () => {
    vi.mocked(s3api.getBucketInfo).mockResolvedValue({
      bucket: 'b1', region: 'us-east-1', createdAt: '', versioning: '',
    } as any)
    vi.mocked(s3api.putBucketVersioning).mockRejectedValue(new Error('put fail'))
    const w = mountDialog()
    await w.setProps({ open: true })
    await flushPromises()
    await toggleBtn(w)!.trigger('click')
    await flushPromises()
    expect(w.emitted('error')).toEqual([['put fail']])
    expect(toggleBtn(w)!.text()).toBe('overview.enable')
    expect(toggleBtn(w)!.attributes('disabled')).toBeUndefined()
  })

  it('重复打开重新加载；关闭时跳过', async () => {
    const w = mountDialog()
    await w.setProps({ open: true })
    await flushPromises()
    expect(vi.mocked(s3api.getBucketInfo)).toHaveBeenCalledTimes(1)
    await w.setProps({ open: false })
    await w.setProps({ open: true })
    await flushPromises()
    expect(vi.mocked(s3api.getBucketInfo)).toHaveBeenCalledTimes(2)
  })

  it('saving 期间按钮禁用并防重复提交', async () => {
    let resolvePut!: (v: unknown) => void
    vi.mocked(s3api.putBucketVersioning).mockImplementationOnce(
      () => new Promise((r) => { resolvePut = r }) as any,
    )
    const w = mountDialog()
    await w.setProps({ open: true })
    await flushPromises()
    await toggleBtn(w)!.trigger('click')
    await nextTick()
    expect(toggleBtn(w)!.text()).toBe('buckets.processing')
    expect(toggleBtn(w)!.attributes('disabled')).toBeDefined()
    // 保存中再次点击：不应重复发起请求
    await toggleBtn(w)!.trigger('click')
    await nextTick()
    expect(vi.mocked(s3api.putBucketVersioning)).toHaveBeenCalledTimes(1)
    resolvePut({ versioning: 'Enabled' })
    await flushPromises()
    expect(toast).toHaveBeenCalledWith('buckets.toastVerEnabled')
  })

  it('info 未加载时 toggleVersioning 走守卫直接返回', async () => {
    const w = mountDialog() // 未打开：info 为 null
    await (w.vm as unknown as { toggleVersioning: () => Promise<void> }).toggleVersioning()
    expect(vi.mocked(s3api.putBucketVersioning)).not.toHaveBeenCalled()
  })
})
