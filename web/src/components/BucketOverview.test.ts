import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import BucketOverview from './BucketOverview.vue'
import { s3api } from '../api'
import { toast } from '../store'
import { confirmDialog } from '../confirm'
import { fmtDate } from '../format'

vi.mock('../api', () => ({
  s3api: {
    getBucketInfo: vi.fn(async () => ({ bucket: 'b1', region: '', createdAt: '', versioning: '' })),
    putBucketVersioning: vi.fn(async () => ({ versioning: 'Enabled' })),
  },
}))
vi.mock('../store', () => ({ toast: vi.fn() }))
vi.mock('../confirm', () => ({ confirmDialog: vi.fn(async () => true) }))
vi.mock('../i18n', () => ({
  t: (k: string) => k,
  tf: (k: string) => k,
  locale: () => 'en-US',
}))

function mountOverview() {
  return mount(BucketOverview, { props: { accountId: 'acc-1', bucket: 'b1' } })
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(s3api.getBucketInfo).mockImplementation(async () => ({
    bucket: 'b1', region: '', createdAt: '', versioning: '',
  }))
  vi.mocked(s3api.putBucketVersioning).mockImplementation(async () => ({ versioning: 'Enabled' }))
  vi.mocked(confirmDialog).mockImplementation(async () => true)
})

describe('BucketOverview', () => {
  it('loading → 渲染区域/创建时间/版本控制状态', async () => {
    let resolveInfo!: (v: Awaited<ReturnType<typeof s3api.getBucketInfo>>) => void
    vi.mocked(s3api.getBucketInfo).mockImplementationOnce(
      () => new Promise((r) => { resolveInfo = r }),
    )
    const w = mountOverview()
    expect(w.text()).toContain('overview.loading')
    resolveInfo({ region: 'us-east-1', createdAt: '2024-01-02T00:00:00Z', versioning: '' } as unknown as Awaited<ReturnType<typeof s3api.getBucketInfo>>)
    await flushPromises()
    const text = w.text()
    expect(text).toContain('b1')
    expect(text).toContain('us-east-1')
    expect(text).toContain(fmtDate('2024-01-02T00:00:00Z'))
    // 未启用 → versioningOff 徽标 + enable 按钮
    expect(text).toContain('overview.versioningOff')
    expect(text).toContain('overview.enable')
    expect(text).toContain('overview.hint')
  })

  it('Empty createdAt 时显示占位符', async () => {
    const w = mountOverview()
    await flushPromises()
    // 两处占位：region 为空 + createdAt 为空
    expect(w.text()).toContain('—')
  })

  it('启用版本控制：确认 → put → toast → changed', async () => {
    let n = 0
    vi.mocked(s3api.getBucketInfo).mockImplementation(async () =>
      (n++ === 0
        ? { region: 'us-east-1', createdAt: '', versioning: '' }
        : { region: 'us-east-1', createdAt: '', versioning: 'Enabled' }) as unknown as Awaited<ReturnType<typeof s3api.getBucketInfo>>,
    )
    const w = mountOverview()
    await flushPromises()
    // 首次加载后按钮应为 enable
    const btn = w.findAll('button').find((b) => b.text() === 'overview.enable')!
    await btn.trigger('click')
    expect(confirmDialog).toHaveBeenCalledWith(
      expect.objectContaining({ title: 'overview.enableTitle', danger: false }),
    )
    expect(vi.mocked(s3api.putBucketVersioning)).toHaveBeenCalledWith('acc-1', {
      bucket: 'b1',
      status: 'Enabled',
    })
    await flushPromises()
    expect(toast).toHaveBeenCalledWith('overview.toastEnabled')
    expect(w.emitted('changed')).toHaveLength(1)
    expect(w.text()).toContain('overview.suspend')
  })

  it('已启用版本控制：suspend 按钮 → danger 确认 → Suspended toast', async () => {
    vi.mocked(s3api.getBucketInfo).mockResolvedValue({
      region: 'us-east-1',
      createdAt: '2024-01-02T00:00:00Z',
      versioning: 'Enabled',
    } as unknown as Awaited<ReturnType<typeof s3api.getBucketInfo>>)
    const w = mountOverview()
    await flushPromises()
    const btn = w.findAll('button').find((b) => b.text() === 'overview.suspend')!
    await btn.trigger('click')
    expect(confirmDialog).toHaveBeenCalledWith(
      expect.objectContaining({ title: 'overview.suspendTitle', danger: true }),
    )
    expect(vi.mocked(s3api.putBucketVersioning)).toHaveBeenCalledWith('acc-1', {
      bucket: 'b1',
      status: 'Suspended',
    })
    await flushPromises()
    expect(toast).toHaveBeenCalledWith('overview.toastSuspended')
    expect(w.emitted('changed')).toHaveLength(1)
  })

  it('取消确认：不调用 put', async () => {
    vi.mocked(confirmDialog).mockResolvedValue(false)
    const w = mountOverview()
    await flushPromises()
    await w.find('button').trigger('click')
    expect(vi.mocked(s3api.putBucketVersioning)).not.toHaveBeenCalled()
    expect(toast).not.toHaveBeenCalled()
  })

  it('load 失败 → emit error', async () => {
    vi.mocked(s3api.getBucketInfo).mockRejectedValueOnce(new Error('load fail'))
    const w = mountOverview()
    await flushPromises()
    expect(w.emitted('error')).toEqual([['load fail']])
  })

  it('put 失败 → emit error 且 toggling 复位', async () => {
    vi.mocked(s3api.getBucketInfo).mockResolvedValue({
      region: '', createdAt: '', versioning: '',
    } as unknown as Awaited<ReturnType<typeof s3api.getBucketInfo>>)
    vi.mocked(s3api.putBucketVersioning).mockRejectedValue(new Error('put fail'))
    const w = mountOverview()
    await flushPromises()
    await w.find('button').trigger('click')
    await flushPromises()
    expect(w.emitted('error')).toEqual([['put fail']])
    expect(toast).not.toHaveBeenCalled()
    const btn = w.findAll('button').find((b) => b.text() === 'overview.enable')!
    expect(btn.attributes('disabled')).toBeUndefined()
  })

  it('put 以非 Error 抛错(字符串)→ String(err) 兜底', async () => {
    vi.mocked(s3api.getBucketInfo).mockResolvedValue({
      region: '', createdAt: '', versioning: 'Enabled',
    } as unknown as Awaited<ReturnType<typeof s3api.getBucketInfo>>)
    vi.mocked(s3api.putBucketVersioning).mockRejectedValue('boom-string')
    const w = mountOverview()
    await flushPromises()
    await w.findAll('button').find((b) => b.text() === 'overview.suspend')!.trigger('click')
    await flushPromises()
    expect(w.emitted('error')).toEqual([['boom-string']])
  })
})

describe('BucketOverview toggle busy guard (vm)', () => {
  it('toggle while already toggling returns early (UI-disabled 守卫)', async () => {
    const w = mountOverview()
    await flushPromises()
    const vm = w.vm as unknown as { toggling: boolean; toggleVersioning: () => Promise<void> }
    vm.toggling = true
    await expect(vm.toggleVersioning()).resolves.toBeUndefined()
    expect(s3api.putBucketVersioning).not.toHaveBeenCalled()
  })
})
