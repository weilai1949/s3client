import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, shallowMount } from '@vue/test-utils'
import { nextTick } from 'vue'
import type { Component } from 'vue'
import BucketsPanel from './BucketsPanel.vue'
import BucketOverview from './BucketOverview.vue'
import BucketEncryption from './BucketEncryption.vue'
import BucketCors from './BucketCors.vue'
import BucketWebsite from './BucketWebsite.vue'
import BucketPolicy from './BucketPolicy.vue'
import BucketTags from './BucketTags.vue'
import CreateBucketDialog from './CreateBucketDialog.vue'
import LifecycleDialog from './LifecycleDialog.vue'
import { s3api } from '../api'
import { state, selectAccount, rememberedAccountId, toast } from '../store'
import { confirmDialog } from '../confirm'
import { fmtDate } from '../format'
import type { Account, BucketItem } from '../types'

vi.mock('../api', () => ({
  s3api: {
    listBuckets: vi.fn(async () => ({ buckets: [] })),
    createBucket: vi.fn(async () => ({ created: 'b1', region: 'us-east-1', acl: 'private' })),
    deleteBucket: vi.fn(async () => ({ deleted: 'b1' })),
    getBucketInfo: vi.fn(async () => ({ bucket: 'b1', region: '', createdAt: '', versioning: '' })),
    getBucketPolicy: vi.fn(async () => ({ bucket: 'b1', configured: false, policy: '' })),
    putBucketPolicy: vi.fn(async () => ({ configured: true })),
    getBucketTags: vi.fn(async () => ({ bucket: 'b1', tags: [] })),
    putBucketTags: vi.fn(async () => ({ updated: 0 })),
    getBucketCors: vi.fn(async () => ({ bucket: 'b1', rules: [] })),
    putBucketCors: vi.fn(async () => ({ updated: 0 })),
    getBucketEncryption: vi.fn(async () => ({ bucket: 'b1', configured: false, algorithm: 'AES256', kmsKeyId: '', bucketKeyEnabled: true })),
    putBucketEncryption: vi.fn(async () => ({ configured: true, algorithm: 'AES256' })),
    getBucketWebsite: vi.fn(async () => ({ bucket: 'b1', configured: false, indexDocument: '', errorDocument: '', redirectAllRequestsTo: '' })),
    putBucketWebsite: vi.fn(async () => ({ configured: true })),
  },
  api: { token: '', base: '' },
  subscribeMigrateEvents: vi.fn(() => () => {}),
}))
vi.mock('../store', async () => {
  const { reactive } = await import('vue')
  return {
    state: reactive({ accounts: [] as Account[], currentAccountId: '' }),
    selectAccount: vi.fn(),
    rememberedAccountId: vi.fn(() => ''),
    toast: vi.fn(),
  }
})
vi.mock('../i18n', () => ({
  t: (k: string) => k,
  tf: (k: string) => k,
  locale: () => 'en-US',
}))
vi.mock('../confirm', () => ({ confirmDialog: vi.fn(async () => true) }))

const account1: Account = {
  id: 'acc-1', name: 'Alpha', endpoint: 'http://minio:9000', region: 'us-east-1',
  accessKey: 'ak', secretKey: 'sk', bucket: 'b1', pathStyle: true, useSSL: false,
}
const account2: Account = {
  id: 'acc-2', name: 'Beta', endpoint: 'http://minio:9000', region: 'us-east-1',
  accessKey: 'ak2', secretKey: 'sk2', bucket: 'b2', pathStyle: true, useSSL: false,
}

const buckets: BucketItem[] = [
  { name: 'alpha', creationDate: '2024-01-02T00:00:00Z' },
  { name: 'beta', creationDate: '' },
]

let wrappers: { unmount: () => void }[] = []

function mountPanel() {
  const w = shallowMount(BucketsPanel)
  wrappers.push(w)
  return w
}

beforeEach(() => {
  vi.clearAllMocks()
  state.accounts = [account1, account2]
  state.currentAccountId = 'acc-1'
  vi.mocked(rememberedAccountId).mockReturnValue('')
  vi.mocked(confirmDialog).mockImplementation(async () => true)
  vi.mocked(s3api.listBuckets).mockImplementation(async () => ({ buckets }))
  vi.mocked(s3api.deleteBucket).mockImplementation(async () => ({ deleted: 'b1' }))
  vi.mocked(s3api.createBucket).mockImplementation(async () => ({ created: 'b1', region: 'us-east-1', acl: 'private' }))
})

afterEach(() => {
  wrappers.forEach((w) => w.unmount())
  wrappers = []
})

function btnByText(w: ReturnType<typeof mountPanel>, text: string) {
  const b = w.findAll('button').find((x) => x.text() === text)
  expect(b, `button ${text} should exist`).toBeTruthy()
  return b!
}

describe('BucketsPanel', () => {
  it('无账号：needAccount 空态，不发起请求，创建按钮禁用', async () => {
    state.accounts = []
    const w = mountPanel()
    await nextTick()
    expect(w.text()).toContain('buckets.needAccount')
    expect(vi.mocked(s3api.listBuckets)).not.toHaveBeenCalled()
    expect(btnByText(w, 'buckets.createBtn').attributes('disabled')).toBeDefined()
  })

  it('记住的账号生效：accSel=remembered，加载并自动打开首个桶', async () => {
    vi.mocked(rememberedAccountId).mockReturnValue('acc-2')
    vi.mocked(s3api.listBuckets).mockResolvedValue({ buckets } as any)
    const w = mountPanel()
    await flushPromises()
    expect(selectAccount).toHaveBeenCalledWith('acc-2')
    expect(vi.mocked(s3api.listBuckets)).toHaveBeenCalledWith('acc-2')
    // 自动进入第一个桶详情
    expect(w.findComponent(BucketOverview).exists()).toBe(true)
    expect(w.text()).toContain('alpha')
  })

  it('默认账号：加载并渲染桶详情，backList → 表格 → manage 回详情', async () => {
    vi.mocked(s3api.listBuckets).mockResolvedValue({ buckets } as any)
    const w = mountPanel()
    await flushPromises()
    expect(selectAccount).toHaveBeenCalledWith('acc-1')
    expect(w.findComponent(BucketOverview).exists()).toBe(true)
    await btnByText(w, 'buckets.backList').trigger('click')
    // 表格列出两个桶
    const rows = w.findAll('tbody tr')
    expect(rows).toHaveLength(2)
    expect(rows[0].text()).toContain('alpha')
    expect(rows[0].text()).toContain(fmtDate('2024-01-02T00:00:00Z'))
    expect(rows[1].text()).toContain('beta')
    // 重新 manage → 详情
    await rows[0].findAll('button').find((b) => b.text() === 'common.manage')!.trigger('click')
    expect(w.findComponent(BucketOverview).exists()).toBe(true)
  })

  it('加载中显示 loading 状态', async () => {
    let resolveList!: (v: unknown) => void
    vi.mocked(s3api.listBuckets).mockImplementationOnce(
      () => new Promise((r) => { resolveList = r }) as any,
    )
    const w = mountPanel()
    await nextTick()
    expect(w.text()).toContain('buckets.loading')
    resolveList({ buckets })
    await flushPromises()
    expect(w.findComponent(BucketOverview).exists()).toBe(true)
  })

  it('空桶列表显示 empty 状态', async () => {
    vi.mocked(s3api.listBuckets).mockResolvedValue({ buckets: [] } as any)
    const w = mountPanel()
    await flushPromises()
    expect(w.text()).toContain('buckets.empty')
  })

  it('加载失败显示错误与重试', async () => {
    vi.mocked(s3api.listBuckets)
      .mockRejectedValueOnce(new Error('list failed'))
      .mockResolvedValueOnce({ buckets } as any)
    const w = mountPanel()
    await flushPromises()
    expect(w.find('.msg.err').text()).toContain('list failed')
    await btnByText(w, 'common.retry').trigger('click')
    await flushPromises()
    // mount 时 onMounted + accSel watch 各触发一次 loadBuckets，重试为第三次
    expect(vi.mocked(s3api.listBuckets)).toHaveBeenCalledTimes(3)
  })

  it('页签切换渲染对应设置组件并转发 error，lifecycle 打开对话框', async () => {
    vi.mocked(s3api.listBuckets).mockResolvedValue({ buckets } as any)
    const w = mountPanel()
    await flushPromises()
    const tabs: [string, Component][] = [
      ['buckets.tabEncryption', BucketEncryption],
      ['buckets.tabCors', BucketCors],
      ['buckets.tabWebsite', BucketWebsite],
      ['buckets.tabPolicy', BucketPolicy],
      ['buckets.tabTags', BucketTags],
      ['buckets.tabOverview', BucketOverview],
    ]
    for (const [label, comp] of tabs) {
      await btnByText(w, label).trigger('click')
      await nextTick()
      // VTU 对动态组件 id 的 findComponent 类型不精确，此处以 any 使用 stub API
      const stub: any = w.findComponent(comp as never)
      expect(stub.exists(), `tab ${label}`).toBe(true)
      // 每个设置组件收到当前账号/桶 props，且 @error 事件上浮为错误横幅
      expect(stub.props('accountId'), `${label} accountId`).toBe('acc-1')
      expect(stub.props('bucket'), `${label} bucket`).toBe('alpha')
      stub.vm.$emit('error', `${label}-err`)
      await nextTick()
      expect(w.find('.msg.err').text(), `${label} error banner`).toContain(`${label}-err`)
    }
    // lifecycle：占位提示 + 编辑按钮 → 对话框
    await btnByText(w, 'buckets.tabLifecycle').trigger('click')
    expect(w.text()).toContain('buckets.lifecycleHint')
    await btnByText(w, 'buckets.editLifecycle').trigger('click')
    const lifecycle = w.findComponent(LifecycleDialog)
    expect(lifecycle.props('open')).toBe(true)
    lifecycle.vm.$emit('error', 'lifecycle-err')
    await nextTick()
    expect(w.find('.msg.err').text()).toContain('lifecycle-err')
    lifecycle.vm.$emit('close')
    await nextTick()
    expect(w.findComponent(LifecycleDialog).props('open')).toBe(false)
  })

  it('创建桶：按钮弹窗 → created → toast + 重新加载 → error 上浮 → close 关闭', async () => {
    vi.mocked(s3api.listBuckets).mockResolvedValue({ buckets } as any)
    const w = mountPanel()
    await flushPromises()
    await btnByText(w, 'buckets.createBtn').trigger('click')
    const create = w.findComponent(CreateBucketDialog)
    expect(create.props('open')).toBe(true)
    expect(create.props('accountId')).toBe('acc-1')
    create.vm.$emit('created')
    await flushPromises()
    expect(toast).toHaveBeenCalledWith('buckets.created')
    // mount 时两次 + created 后一次
    expect(vi.mocked(s3api.listBuckets)).toHaveBeenCalledTimes(3)
    create.vm.$emit('error', 'create-err')
    await nextTick()
    expect(w.find('.msg.err').text()).toContain('create-err')
    create.vm.$emit('close')
    expect(w.findComponent(CreateBucketDialog).props('open')).toBe(false)
  })

  it('确认删除桶：delete → toast → 重新加载', async () => {
    vi.mocked(s3api.listBuckets).mockResolvedValue({ buckets } as any)
    const w = mountPanel()
    await flushPromises()
    await btnByText(w, 'buckets.backList').trigger('click')
    const row = w.findAll('tbody tr')[0]!
    await row.findAll('button').find((b) => b.text() === 'common.delete')!.trigger('click')
    expect(confirmDialog).toHaveBeenCalledWith(
      expect.objectContaining({ title: 'buckets.deleteTitle', danger: true }),
    )
    expect(vi.mocked(s3api.deleteBucket)).toHaveBeenCalledWith('acc-1', 'alpha')
    await flushPromises()
    expect(toast).toHaveBeenCalledWith('buckets.deleted')
    // mount 时两次 + 删除后 reload 一次
    expect(vi.mocked(s3api.listBuckets)).toHaveBeenCalledTimes(3)
  })

  it('删除确认取消：不调用 delete', async () => {
    vi.mocked(confirmDialog).mockResolvedValue(false)
    vi.mocked(s3api.listBuckets).mockResolvedValue({ buckets } as any)
    const w = mountPanel()
    await flushPromises()
    await btnByText(w, 'buckets.backList').trigger('click')
    await w.findAll('tbody tr')[0]!.findAll('button').find((b) => b.text() === 'common.delete')!.trigger('click')
    expect(vi.mocked(s3api.deleteBucket)).not.toHaveBeenCalled()
  })

  it('删除失败：错误上浮', async () => {
    vi.mocked(s3api.listBuckets).mockResolvedValue({ buckets } as any)
    vi.mocked(s3api.deleteBucket).mockRejectedValue(new Error('del fail'))
    const w = mountPanel()
    await flushPromises()
    await btnByText(w, 'buckets.backList').trigger('click')
    await w.findAll('tbody tr')[0]!.findAll('button').find((b) => b.text() === 'common.delete')!.trigger('click')
    await flushPromises()
    expect(w.find('.msg.err').text()).toContain('del fail')
  })

  it('账号切换：accSel watch 重新加载并清空选中', async () => {
    vi.mocked(s3api.listBuckets).mockResolvedValue({ buckets } as any)
    const w = mountPanel()
    await flushPromises()
    expect(vi.mocked(s3api.listBuckets).mock.calls[0][0]).toBe('acc-1')
    await w.find('select.acc-select').setValue('acc-2')
    await flushPromises()
    expect(selectAccount).toHaveBeenCalledWith('acc-2')
    expect(vi.mocked(s3api.listBuckets)).toHaveBeenCalledWith('acc-2')
    // 重新加载后仍自动进入新列表首个桶
    expect(w.findComponent(BucketOverview).exists()).toBe(true)
  })

  it('账号列表变化：失效的 accSel 被重置', async () => {
    vi.mocked(s3api.listBuckets).mockResolvedValue({ buckets } as any)
    const w = mountPanel()
    await flushPromises()
    state.accounts = []
    await nextTick()
    expect(selectAccount).toHaveBeenCalledWith('')
    expect(w.text()).toContain('buckets.needAccount')
  })

  it('账号列表变化：accSel 为空时 watch 短路；accSel 失效时重置为列表首个账号', async () => {
    vi.mocked(s3api.listBuckets).mockResolvedValue({ buckets } as any)
    // 场景 1：初始无账号（accSel=''），随后列表出现 → watch 短路，不重复 selectAccount
    state.accounts = []
    const w = mountPanel()
    await flushPromises()
    state.accounts = [account1]
    await nextTick()
    expect(selectAccount).toHaveBeenCalledTimes(1) // 仅 onMounted 的 selectAccount('')

    // 场景 2：accSel 指向已删除账号 → some() 回调执行并重置到第一个账号
    await w.find('select.acc-select').setValue('acc-1')
    await flushPromises()
    vi.mocked(selectAccount).mockClear()
    state.accounts = [account2]
    await nextTick()
    expect(selectAccount).toHaveBeenCalledWith('acc-2')
  })

  it('子页签 changed：重新加载并在列表变化时重新选中', async () => {
    vi.mocked(s3api.listBuckets).mockResolvedValue({ buckets } as any)
    const w = mountPanel()
    await flushPromises()
    // 当前选中 alpha（在列表中）→ 列表变化为仅 beta → 自动切到 beta
    vi.mocked(s3api.listBuckets).mockResolvedValue({ buckets: [buckets[1]] } as any)
    w.findComponent(BucketOverview).vm.$emit('changed')
    await flushPromises()
    // mount 时两次 + changed 后一次
    expect(vi.mocked(s3api.listBuckets)).toHaveBeenCalledTimes(3)
    expect(w.text()).toContain('beta')
  })

  it('子页签 error 事件显示错误横幅', async () => {
    vi.mocked(s3api.listBuckets).mockResolvedValue({ buckets } as any)
    const w = mountPanel()
    await flushPromises()
    w.findComponent(BucketOverview).vm.$emit('error', 'boom')
    await nextTick()
    expect(w.find('.msg.err').text()).toContain('boom')
  })
})

describe('BucketsPanel empty account guard', () => {
  it('loadBuckets with empty accSel clears buckets', async () => {
    state.currentAccountId = ''
    const w = mountPanel()
    await flushPromises()
    // 空账号：列表为空（loadBuckets 提前返回清空，不依赖 API 结果）
    expect(w.find('.empty').exists() || w.find('.buckets').exists()).toBe(false)
    state.currentAccountId = 'acc-1' // 恢复，避免影响后续
  })
})
