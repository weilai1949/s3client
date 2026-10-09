import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import StorageReportPanel from './StorageReportPanel.vue'
import { s3api } from '../api'
import { state } from '../store'
import type { Account, BucketItem, StorageReport } from '../types'

vi.mock('../api', () => ({
  s3api: {
    listBuckets: vi.fn(async () => ({ buckets: [] })),
    storageReport: vi.fn(),
  },
  api: { token: '', base: '' },
  subscribeMigrateEvents: vi.fn(() => () => {}),
}))
vi.mock('../store', async () => {
  const { reactive } = await import('vue')
  return {
    state: reactive({ accounts: [] as Account[], currentAccountId: '' }),
  }
})
vi.mock('../i18n', () => ({ t: (k: string) => k }))

const buckets: BucketItem[] = [
  { name: 'alpha', creationDate: '2024-01-02T00:00:00Z' },
  { name: 'beta', creationDate: '' },
]

function reportFixture(over: Partial<StorageReport> = {}): StorageReport {
  return {
    bucket: 'alpha',
    prefix: '',
    objectCount: 6,
    totalSize: 32212254720,
    truncated: false,
    monthlyCost: 0.53,
    prefixGroupCount: 3,
    byStorageClass: [{ storageClass: 'STANDARD', count: 3, size: 17179869184, monthlyCost: 0.368 }],
    byPrefix: [{ prefix: 'photos/', count: 3, size: 17179869184 }],
    recommendations: [{
      kind: 'infrequent', fromStorageClass: 'STANDARD', toStorageClass: 'STANDARD_IA',
      count: 1, size: 5368709120, estimatedMonthlySaving: 0.0525,
    }],
    ...over,
  }
}

let wrappers: { unmount: () => void }[] = []

function mountPanel() {
  const w = mount(StorageReportPanel)
  wrappers.push(w)
  return w
}

function btnByText(w: ReturnType<typeof mountPanel>, text: string) {
  const b = w.findAll('button').find((x) => x.text() === text)
  expect(b, `button ${text} should exist`).toBeTruthy()
  return b!
}

beforeEach(() => {
  vi.clearAllMocks()
  state.accounts = []
  state.currentAccountId = ''
  vi.mocked(s3api.listBuckets).mockResolvedValue({ buckets })
  vi.mocked(s3api.storageReport).mockResolvedValue(reportFixture())
})

afterEach(() => {
  wrappers.forEach((w) => w.unmount())
  wrappers = []
})

describe('StorageReportPanel', () => {
  it('无账号：needAccount 空态，不发起请求', async () => {
    const w = mountPanel()
    await nextTick()
    expect(w.text()).toContain('storageReport.needAccount')
    expect(vi.mocked(s3api.listBuckets)).not.toHaveBeenCalled()
    expect(w.find('button.btn').exists()).toBe(false)
  })

  it('有账号：加载桶并默认选中首个；生成报告渲染用量与建议', async () => {
    state.accounts = [{ id: 'acc-1', name: 'Alpha' } as Account]
    state.currentAccountId = 'acc-1'
    const w = mountPanel()
    await flushPromises()
    expect(vi.mocked(s3api.listBuckets)).toHaveBeenCalledWith('acc-1')
    const options = w.findAll('select option')
    expect(options.map((o) => o.text())).toEqual(['alpha', 'beta'])
    await expect(btnByText(w, 'storageReport.generate').attributes('disabled')).toBeUndefined()
    expect(w.text()).toContain('storageReport.idle')

    await btnByText(w, 'storageReport.generate').trigger('click')
    await flushPromises()
    expect(vi.mocked(s3api.storageReport)).toHaveBeenCalledWith('acc-1', { bucket: 'alpha', prefix: '' })
    expect(w.text()).toContain('30.0 GiB')
    expect(w.text()).toContain('$0.53')
    expect(w.text()).toContain('STANDARD')
    expect(w.text()).toContain('photos/')
    expect(w.text()).toContain('storageReport.kind.infrequent')
  })

  it('前缀输入随请求透传', async () => {
    state.accounts = [{ id: 'acc-1', name: 'Alpha' } as Account]
    state.currentAccountId = 'acc-1'
    const w = mountPanel()
    await flushPromises()
    await w.find('input').setValue('logs/')
    await btnByText(w, 'storageReport.generate').trigger('click')
    await flushPromises()
    expect(vi.mocked(s3api.storageReport)).toHaveBeenCalledWith('acc-1', { bucket: 'alpha', prefix: 'logs/' })
  })

  it('加载中显示 loading 文案', async () => {
    state.accounts = [{ id: 'acc-1', name: 'Alpha' } as Account]
    state.currentAccountId = 'acc-1'
    let resolveReport!: (r: StorageReport) => void
    vi.mocked(s3api.storageReport).mockImplementationOnce(
      () => new Promise((r) => { resolveReport = r }),
    )
    const w = mountPanel()
    await flushPromises()
    await btnByText(w, 'storageReport.generate').trigger('click')
    await nextTick()
    expect(w.text()).toContain('storageReport.loading')
    resolveReport(reportFixture())
    await flushPromises()
    expect(w.text()).not.toContain('storageReport.loading')
  })

  it('报告请求失败：显示错误横幅', async () => {
    state.accounts = [{ id: 'acc-1', name: 'Alpha' } as Account]
    state.currentAccountId = 'acc-1'
    vi.mocked(s3api.storageReport).mockRejectedValueOnce(new Error('report boom'))
    const w = mountPanel()
    await flushPromises()
    await btnByText(w, 'storageReport.generate').trigger('click')
    await flushPromises()
    expect(w.find('.msg.err').text()).toContain('report boom')
  })

  it('桶列表加载失败：错误横幅且禁用生成', async () => {
    state.accounts = [{ id: 'acc-1', name: 'Alpha' } as Account]
    state.currentAccountId = 'acc-1'
    vi.mocked(s3api.listBuckets).mockRejectedValueOnce(new Error('list boom'))
    const w = mountPanel()
    await flushPromises()
    expect(w.find('.msg.err').text()).toContain('list boom')
    expect(btnByText(w, 'storageReport.generate').attributes('disabled')).toBeDefined()
  })

  it('空桶列表：无可选桶、生成按钮禁用', async () => {
    state.accounts = [{ id: 'acc-1', name: 'Alpha' } as Account]
    state.currentAccountId = 'acc-1'
    vi.mocked(s3api.listBuckets).mockResolvedValue({ buckets: [] })
    const w = mountPanel()
    await flushPromises()
    expect(w.findAll('select option')).toHaveLength(0)
    expect(btnByText(w, 'storageReport.generate').attributes('disabled')).toBeDefined()
  })

  it('截断与无建议：渲染提示文案', async () => {
    state.accounts = [{ id: 'acc-1', name: 'Alpha' } as Account]
    state.currentAccountId = 'acc-1'
    vi.mocked(s3api.storageReport).mockResolvedValue(reportFixture({ truncated: true, recommendations: [] }))
    const w = mountPanel()
    await flushPromises()
    await btnByText(w, 'storageReport.generate').trigger('click')
    await flushPromises()
    expect(w.text()).toContain('storageReport.truncated')
    expect(w.text()).toContain('storageReport.noRecommendations')
  })

  it('切换账号：watch 重新拉取桶列表', async () => {
    state.accounts = [{ id: 'acc-1', name: 'Alpha' } as Account]
    state.currentAccountId = 'acc-1'
    mountPanel()
    await flushPromises()
    state.currentAccountId = 'acc-2'
    await nextTick()
    await flushPromises()
    expect(vi.mocked(s3api.listBuckets)).toHaveBeenLastCalledWith('acc-2')
  })

  it('桶列表加载中禁用 select；选择变更更新 v-model', async () => {
    state.accounts = [{ id: 'acc-1', name: 'Alpha' } as Account]
    state.currentAccountId = 'acc-1'
    let resolveList!: (v: { buckets: BucketItem[] }) => void
    vi.mocked(s3api.listBuckets).mockImplementationOnce(
      () => new Promise((r) => { resolveList = r }),
    )
    const w = mountPanel()
    await nextTick()
    expect(w.find('select').attributes('disabled')).toBeDefined()
    resolveList({ buckets })
    await flushPromises()
    expect(w.find('select').attributes('disabled')).toBeUndefined()
    await w.find('select').setValue('beta')
    await nextTick()
    await btnByText(w, 'storageReport.generate').trigger('click')
    await flushPromises()
    expect(vi.mocked(s3api.storageReport)).toHaveBeenCalledWith('acc-1', { bucket: 'beta', prefix: '' })
  })
})
