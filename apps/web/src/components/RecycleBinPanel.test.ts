import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import RecycleBinPanel from './RecycleBinPanel.vue'
import { s3api } from '../api'
import { state, toast, selectAccount, rememberedAccountId } from '../store'
import { confirmDialog } from '../confirm'
import { ROW_HEIGHT } from '../virtualList'
import type { Account } from '../types'

type ListTrashResult = Awaited<ReturnType<typeof s3api.listTrash>>

// happy-dom 也提供 ResizeObserver，但统一用可控 stub 保证 clientHeight=0 → viewportH=480
class RO {
  observe() {}
  unobserve() {}
  disconnect() {}
}
vi.stubGlobal('ResizeObserver', RO)

vi.mock('../api', () => ({
  s3api: {
    listBuckets: vi.fn(async () => ({ buckets: [] })),
    listTrash: vi.fn(async () => ({ deleteMarkers: [], isTruncated: false, nextKeyMarker: '', nextVersionIdMarker: '' })),
    restoreDeleteMarker: vi.fn(async () => ({ restored: 'k1', versionId: 'v1' })),
    purgeTrashObject: vi.fn(async () => ({ deleted: 1 })),
  },
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

vi.mock('../confirm', () => ({
  confirmDialog: vi.fn(async () => true),
}))

vi.mock('../i18n', () => ({
  t: (k: string) => k,
  tf: (k: string) => k,
}))

const acc1: Account = {
  id: 'acc-1',
  name: 'A1',
  endpoint: '127.0.0.1:9000',
  region: 'us-east-1',
  accessKey: 'ak',
  secretSet: true,
  bucket: 'b1',
  pathStyle: true,
  useSSL: false,
}

const acc2: Account = { ...acc1, id: 'acc-2', name: 'A2' }

const b1 = { name: 'b1', creationDate: '2024-01-01' }
const b2 = { name: 'b2', creationDate: '2024-02-01' }

const m1: TrashMarker = { key: 'k1', versionId: 'v1', isLatest: true, lastModified: '2024-01-01T00:00:00Z' }
const m2: TrashMarker = { key: 'k2', versionId: 'v2', isLatest: false, lastModified: '2024-02-02T00:00:00Z' }

interface TrashMarker {
  key: string
  versionId: string
  isLatest: boolean
  lastModified: string
}

function page(
  markers: TrashMarker[] = [],
  isTruncated = false,
  nextKeyMarker = '',
  nextVersionIdMarker = '',
): { deleteMarkers: TrashMarker[]; isTruncated: boolean; nextKeyMarker: string; nextVersionIdMarker: string } {
  return { deleteMarkers: markers, isTruncated, nextKeyMarker, nextVersionIdMarker }
}

function manyMarkers(n: number): TrashMarker[] {
  return Array.from({ length: n }, (_, i) => ({
    key: `mk${String(i).padStart(2, '0')}`,
    versionId: `v${i}`,
    isLatest: true,
    lastModified: '2024-01-01T00:00:00Z',
  }))
}

function mountPanel() {
  return mount(RecycleBinPanel)
}

function findButton(w: ReturnType<typeof mount>, text: string) {
  const btn = w.findAll('button').find((b) => b.text() === text)
  expect(btn, `button "${text}" should exist`).toBeTruthy()
  return btn!
}

beforeEach(() => {
  // 必须 resetAllMocks：clearAllMocks 不会清空 mockResolvedValueOnce 队列，
  // 上一个用例未消费的 Once 会污染后续用例（listTrash 数据错位）。
  vi.resetAllMocks()
  vi.mocked(s3api.listBuckets).mockResolvedValue({ buckets: [] })
  vi.mocked(s3api.listTrash).mockResolvedValue({ deleteMarkers: [], isTruncated: false, nextKeyMarker: '', nextVersionIdMarker: '' })
  vi.mocked(s3api.restoreDeleteMarker).mockResolvedValue({ restored: 'k1', versionId: 'v1' })
  vi.mocked(s3api.purgeTrashObject).mockResolvedValue({ deleted: 1 } as Awaited<ReturnType<typeof s3api.purgeTrashObject>>)
  vi.mocked(confirmDialog).mockResolvedValue(true)
  vi.mocked(rememberedAccountId).mockReturnValue('')
  state.accounts = [acc1]
  state.currentAccountId = ''
})

describe('RecycleBinPanel', () => {
  it('无账号：显示 needAccount 空态，不请求桶', async () => {
    state.accounts = []
    const w = mountPanel()
    await flushPromises()
    expect(w.find('.empty').text()).toContain('trash.needAccount')
    expect(selectAccount).toHaveBeenCalledWith('')
    expect(s3api.listBuckets).not.toHaveBeenCalled()
    expect(s3api.listTrash).not.toHaveBeenCalled()
  })

  it('remembered 账号优先；失效时回退 currentAccountId / 第一个账号', async () => {
    // remembered 不存在 → 回退 currentAccountId
    vi.mocked(rememberedAccountId).mockReturnValue('gone-acc')
    state.currentAccountId = 'acc-1'
    let w = mountPanel()
    await flushPromises()
    expect(selectAccount).toHaveBeenCalledWith('acc-1')
    w.unmount()

    // 既无 remembered 也无 current → 第一个账号
    state.currentAccountId = ''
    vi.mocked(rememberedAccountId).mockReturnValue('')
    w = mountPanel()
    await flushPromises()
    expect(selectAccount).toHaveBeenLastCalledWith('acc-1')
  })

  it('加载桶与回收站标记并渲染表格；从未被选桶自动选中', async () => {
    vi.mocked(rememberedAccountId).mockReturnValue('acc-1')
    vi.mocked(s3api.listBuckets).mockResolvedValue({ buckets: [b1, b2] })
    vi.mocked(s3api.listTrash).mockResolvedValueOnce(page([m1]))
    const w = mountPanel()
    await flushPromises()

    expect(s3api.listBuckets).toHaveBeenCalledWith('acc-1')
    // 第二个 select 为桶选择器，自动选中第一项 b1
    const sels = w.findAll('select.acc-select')
    expect(sels.length).toBe(2)
    expect((sels[1].element as HTMLSelectElement).value).toBe('b1')
    expect(s3api.listTrash).toHaveBeenCalledWith('acc-1', { bucket: 'b1', keyMarker: '', versionIdMarker: '', maxKeys: 1000 })
    expect(w.text()).toContain('k1')
    expect(w.text()).toContain('v1')
    expect(w.text()).toContain('trash.endReached')
    // 未截断 → more 禁用
    expect(findButton(w, 'common.more').attributes('disabled')).toBeDefined()
    // 行内 restore/purge 按钮
    expect(w.findAll('tbody tr')[0].find('button.danger.sm').text()).toBe('trash.purge')
    expect(w.findAll('tbody tr')[0].findAll('button')[0].text()).toBe('trash.restore')
  })

  it('切换账号：重置桶与标记并重新拉取', async () => {
    state.accounts = [acc1, acc2]
    vi.mocked(rememberedAccountId).mockReturnValue('acc-1')
    vi.mocked(s3api.listBuckets).mockImplementation(
      async (id: string) => ({ buckets: id === 'acc-1' ? [b1, b2] : [] }),
    )
    vi.mocked(s3api.listTrash).mockResolvedValueOnce(page([m1]))
    const w = mountPanel()
    await flushPromises()
    await w.findAll('select.acc-select')[0].setValue('acc-2')
    await flushPromises()
    expect(selectAccount).toHaveBeenLastCalledWith('acc-2')
    expect(s3api.listBuckets).toHaveBeenLastCalledWith('acc-2')
    // acc-2 无桶 → bucketSel 空 → 提示选择桶且无标记
    expect((w.findAll('select.acc-select')[1].element as HTMLSelectElement).value).toBe('')
    expect(w.text()).toContain('trash.pickBucket')
    expect(w.text()).not.toContain('k1')
    expect(w.text()).not.toContain('server.any')
  })

  it('恢复：取消不调用 API；成功移除标记并提示', async () => {
    vi.mocked(rememberedAccountId).mockReturnValue('acc-1')
    vi.mocked(s3api.listBuckets).mockResolvedValue({ buckets: [b1] })
    vi.mocked(s3api.listTrash).mockResolvedValueOnce(page([m1, m2]))
    vi.mocked(confirmDialog).mockResolvedValueOnce(false)
    const w = mountPanel()
    await flushPromises()

    await findButton(w, 'trash.restore').trigger('click')
    await flushPromises()
    expect(s3api.restoreDeleteMarker).not.toHaveBeenCalled()
    expect(confirmDialog).toHaveBeenCalledWith(expect.objectContaining({ confirmText: 'trash.restore', danger: false }))

    vi.mocked(confirmDialog).mockResolvedValueOnce(true)
    await findButton(w, 'trash.restore').trigger('click')
    await flushPromises()
    expect(s3api.restoreDeleteMarker).toHaveBeenCalledWith('acc-1', { bucket: 'b1', key: 'k1', versionId: 'v1' })
    expect(toast).toHaveBeenCalledWith('trash.restored')
    // k1 行被移除，k2 保留
    expect(w.text()).toContain('k2')
    expect(w.text()).not.toContain('k1')
  })

  it('恢复失败：展示错误信息', async () => {
    vi.mocked(rememberedAccountId).mockReturnValue('acc-1')
    vi.mocked(s3api.listBuckets).mockResolvedValue({ buckets: [b1] })
    vi.mocked(s3api.listTrash).mockResolvedValueOnce(page([m1]))
    vi.mocked(s3api.restoreDeleteMarker).mockRejectedValueOnce(new Error('restore boom'))
    const w = mountPanel()
    await flushPromises()
    await findButton(w, 'trash.restore').trigger('click')
    await flushPromises()
    expect(w.find('.msg.err').text()).toContain('restore boom')
  })

  it('清空：取消不调用 API；成功移除并按 key 过滤', async () => {
    vi.mocked(rememberedAccountId).mockReturnValue('acc-1')
    vi.mocked(s3api.listBuckets).mockResolvedValue({ buckets: [b1] })
    vi.mocked(s3api.listTrash).mockResolvedValueOnce(page([m1, m2]))
    vi.mocked(s3api.purgeTrashObject).mockResolvedValueOnce({ deleted: 2 } as Awaited<ReturnType<typeof s3api.purgeTrashObject>>)
    vi.mocked(confirmDialog).mockResolvedValueOnce(false)
    const w = mountPanel()
    await flushPromises()

    await findButton(w, 'trash.purge').trigger('click')
    await flushPromises()
    expect(s3api.purgeTrashObject).not.toHaveBeenCalled()

    vi.mocked(confirmDialog).mockResolvedValueOnce(true)
    await findButton(w, 'trash.purge').trigger('click')
    await flushPromises()
    expect(s3api.purgeTrashObject).toHaveBeenCalledWith('acc-1', { bucket: 'b1', key: 'k1' })
    // 同一 key 的所有版本被过滤（k2 保留）
    expect(w.text()).toContain('k2')
    expect(w.text()).not.toContain('k1')
  })

  it('清空失败：展示错误信息', async () => {
    vi.mocked(rememberedAccountId).mockReturnValue('acc-1')
    vi.mocked(s3api.listBuckets).mockResolvedValue({ buckets: [b1] })
    vi.mocked(s3api.listTrash).mockResolvedValueOnce(page([m1]))
    vi.mocked(s3api.purgeTrashObject).mockRejectedValueOnce(new Error('purge boom'))
    const w = mountPanel()
    await flushPromises()
    await findButton(w, 'trash.purge').trigger('click')
    await flushPromises()
    expect(w.find('.msg.err').text()).toContain('purge boom')
  })

  it('加载更多：传递游标并追加标记，页尾禁用 more', async () => {
    vi.mocked(rememberedAccountId).mockReturnValue('acc-1')
    vi.mocked(s3api.listBuckets).mockResolvedValue({ buckets: [b1] })
    vi.mocked(s3api.listTrash)
      .mockResolvedValueOnce(page([m1], true, 'k1', 'v1'))
      .mockResolvedValueOnce(page([m2], false))
    const w = mountPanel()
    await flushPromises()
    expect(w.text()).toContain('trash.hasMore')
    await findButton(w, 'common.more').trigger('click')
    await flushPromises()
    expect(s3api.listTrash).toHaveBeenNthCalledWith(2, 'acc-1', { bucket: 'b1', keyMarker: 'k1', versionIdMarker: 'v1', maxKeys: 1000 })
    expect(w.text()).toContain('k1')
    expect(w.text()).toContain('k2')
    expect(w.text()).toContain('trash.endReached')
    expect(findButton(w, 'common.more').attributes('disabled')).toBeDefined()
  })

  it('空页自动翻页跳过（reset 跳过 2 页；loadMore 跳过 3 页）', async () => {
    vi.mocked(rememberedAccountId).mockReturnValue('acc-1')
    vi.mocked(s3api.listBuckets).mockResolvedValue({ buckets: [b1] })
    vi.mocked(s3api.listTrash).mockResolvedValue(page([], true, '', ''))
    const w = mountPanel()
    await flushPromises()
    expect(s3api.listTrash).toHaveBeenCalledTimes(3)
    expect(w.text()).toContain('trash.hasMore')
    await findButton(w, 'common.more').trigger('click')
    await flushPromises()
    expect(s3api.listTrash).toHaveBeenCalledTimes(7)
  })

  it('快速切换桶：过期请求静默终止，不污染标记列表', async () => {
    vi.mocked(rememberedAccountId).mockReturnValue('acc-1')
    vi.mocked(s3api.listBuckets).mockResolvedValue({ buckets: [b1, b2] })
    let ra!: (v: ListTrashResult) => void
    let rb!: (v: ListTrashResult) => void
    vi.mocked(s3api.listTrash)
      .mockImplementationOnce(() => new Promise((r) => { ra = r }))
      .mockImplementationOnce(() => new Promise((r) => { rb = r }))
    const w = mountPanel()
    await flushPromises()
    expect(s3api.listTrash).toHaveBeenCalledTimes(1)
    // 第一次 loadMarkers（b1）仍在途
    await w.findAll('select.acc-select')[1].setValue('b2')
    await flushPromises()
    expect(s3api.listTrash).toHaveBeenCalledTimes(2)
    // 新请求挂起期间展示 loading 空态
    expect(w.text()).toContain('trash.loading')
    // 新请求先完成
    rb(page([m2]))
    await flushPromises()
    expect(w.text()).toContain('k2')
    // 旧请求后完成 → seq 不匹配被丢弃
    ra(page([m1], true, 'old', 'old'))
    await flushPromises()
    expect(w.text()).not.toContain('k1')
  })

  it('过期请求失败：catch 中 seq 不匹配 → 不显示错误', async () => {
    vi.mocked(rememberedAccountId).mockReturnValue('acc-1')
    vi.mocked(s3api.listBuckets).mockResolvedValue({ buckets: [b1, b2] })
    let raj!: (e: unknown) => void
    let rb!: (v: ListTrashResult) => void
    vi.mocked(s3api.listTrash)
      .mockImplementationOnce(() => new Promise((_res, rej) => { raj = rej }))
      .mockImplementationOnce(() => new Promise((res) => { rb = res }))
    const w = mountPanel()
    await flushPromises()
    // 第一次（b1）在途；切到 b2 发起第二次
    await w.findAll('select.acc-select')[1].setValue('b2')
    await flushPromises()
    expect(s3api.listTrash).toHaveBeenCalledTimes(2)
    // 新请求成功
    rb(page([m2]))
    await flushPromises()
    expect(w.text()).toContain('k2')
    // 旧请求以 reject 失败 → catch 中 seq 不匹配，不设置 error
    raj(new Error('stale boom'))
    await flushPromises()
    expect(w.find('.msg.err').exists()).toBe(false)
  })

  it('标记加载失败显示错误；重试重新拉取', async () => {
    vi.mocked(rememberedAccountId).mockReturnValue('acc-1')
    vi.mocked(s3api.listBuckets).mockResolvedValue({ buckets: [b1] })
    vi.mocked(s3api.listTrash)
      .mockRejectedValueOnce(new Error('trash boom'))
      .mockResolvedValueOnce(page([m1]))
    const w = mountPanel()
    await flushPromises()
    expect(w.find('.msg.err').text()).toContain('trash boom')
    await w.find('.msg.err button.link').trigger('click')
    await flushPromises()
    expect(s3api.listTrash).toHaveBeenCalledTimes(2)
    expect(toast).not.toHaveBeenCalled()
  })

  it('桶列表加载失败显示错误；重试先重拉桶，桶仍缺则标记不重拉', async () => {
    state.accounts = [acc1]
    vi.mocked(rememberedAccountId).mockReturnValue('acc-1')
    vi.mocked(s3api.listBuckets).mockRejectedValue(new Error('list boom'))
    const w = mountPanel()
    await flushPromises()
    expect(w.find('.msg.err').text()).toContain('list boom')
    const listBucketsCalls = vi.mocked(s3api.listBuckets).mock.calls.length
    await w.find('.msg.err button.link').trigger('click')
    await flushPromises()
    // 重试必须重新拉桶（否则 bucketSel 永远为空、横幅永久卡死且无任何反馈）
    expect(vi.mocked(s3api.listBuckets).mock.calls.length).toBeGreaterThan(listBucketsCalls)
    // 桶仍然没拉到 → bucketSel 为空 → loadMarkers 提前返回
    expect(s3api.listTrash).not.toHaveBeenCalled()
  })

  it('重试成功后错误横幅必须消失并恢复列表（横幅不得永久遮蔽整页）', async () => {
    vi.mocked(rememberedAccountId).mockReturnValue('acc-1')
    vi.mocked(s3api.listBuckets).mockResolvedValue({ buckets: [b1] })
    let fail = true
    vi.mocked(s3api.listTrash).mockImplementation(async () => {
      if (fail) throw new Error('trash down')
      return page([m1])
    })
    const w = mountPanel()
    await flushPromises()
    expect(w.find('.msg.err').text()).toContain('trash down')

    fail = false
    await w.find('.msg.err button.link').trigger('click')
    await flushPromises()
    expect(w.find('.msg.err').exists(), '重试成功后错误横幅必须消失').toBe(false)
    expect(w.text(), '成功后列表必须恢复渲染').toContain('k1')
  })

  it('刷新按钮重新加载标记（reset 语义）', async () => {
    vi.mocked(rememberedAccountId).mockReturnValue('acc-1')
    vi.mocked(s3api.listBuckets).mockResolvedValue({ buckets: [b1] })
    vi.mocked(s3api.listTrash)
      .mockResolvedValueOnce(page([m1]))
      .mockResolvedValueOnce(page([m1, m2]))
    const w = mountPanel()
    await flushPromises()
    expect(w.text()).toContain('k1')
    expect(w.text()).not.toContain('k2')
    await findButton(w, 'common.refresh').trigger('click')
    await flushPromises()
    expect(s3api.listTrash).toHaveBeenCalledTimes(2)
    expect(w.text()).toContain('k2')
  })

  it('busy 中 restore/purge 直接返回：不弹确认、不发请求', async () => {
    vi.mocked(rememberedAccountId).mockReturnValue('acc-1')
    vi.mocked(s3api.listBuckets).mockResolvedValue({ buckets: [b1] })
    vi.mocked(s3api.listTrash).mockResolvedValueOnce(page([m1]))
    const w = mountPanel()
    await flushPromises()
    // 直接置 busy（行内按钮已禁用，改用 vm 直调验证 busy 守卫）
    ;(w.vm as unknown as { busy: boolean }).busy = true
    await (w.vm as unknown as { restore: (m: TrashMarker) => Promise<void> }).restore(m1)
    await (w.vm as unknown as { purge: (m: TrashMarker) => Promise<void> }).purge(m1)
    expect(confirmDialog).not.toHaveBeenCalled()
    expect(s3api.restoreDeleteMarker).not.toHaveBeenCalled()
    expect(s3api.purgeTrashObject).not.toHaveBeenCalled()
  })

  it('virtual 行高由 ROW_HEIGHT 绑定到行内样式（防 CSS 字面量 38/42 漂移回归）', async () => {
    vi.mocked(rememberedAccountId).mockReturnValue('acc-1')
    vi.mocked(s3api.listBuckets).mockResolvedValue({ buckets: [b1] })
    vi.mocked(s3api.listTrash).mockResolvedValue(page([m1, m2]))
    const w = mountPanel()
    await flushPromises()
    const row = w.find('tbody tr.v-row')
    expect(row.exists()).toBe(true)
    // 渲染行高必须来自虚拟窗口同一常量（此前 CSS 字面量 38 vs ROW_HEIGHT=42 漂移）
    expect(row.attributes('style')).toContain(`height: ${ROW_HEIGHT}px`)
  })

  it('virtualizes long marker lists：只渲染窗口行 + spacer，滚动后 padTop 出现且首行正确', async () => {
    vi.mocked(rememberedAccountId).mockReturnValue('acc-1')
    vi.mocked(s3api.listBuckets).mockResolvedValue({ buckets: [b1] })
    vi.mocked(s3api.listTrash).mockResolvedValue(page(manyMarkers(50)))
    const w = mountPanel()
    await flushPromises()
    // viewportH=480 → ceil(480/ROW_HEIGHT)+24 = 36 行窗口
    expect(w.findAll('tbody tr.v-row')).toHaveLength(36)
    expect(w.findAll('tbody tr.v-spacer')).toHaveLength(1) // 初始仅 padBottom（50-36 行）
    const wrap = w.find('.tbl-wrap')
    // 「加载更多」工具条留在滚动容器之外（不受滚动影响）
    expect(wrap.element.contains(findButton(w, 'common.more').element)).toBe(false)
    ;(wrap.element as HTMLElement).scrollTop = ROW_HEIGHT * 30
    await wrap.trigger('scroll')
    await nextTick()
    // 滚动后 padTop 出现（18*42=756px），首行是第 18 条
    const spacer = w.find('tbody tr.v-spacer')
    expect(spacer.exists()).toBe(true)
    expect(spacer.find('td').attributes('style')).toContain('756px')
    expect(w.findAll('tbody tr.v-row')[0].text()).toContain('mk18')
  })

  it('切换桶后虚拟窗口回到顶部（不残留旧 scrollTop）', async () => {
    vi.mocked(rememberedAccountId).mockReturnValue('acc-1')
    vi.mocked(s3api.listBuckets).mockResolvedValue({ buckets: [b1, b2] })
    vi.mocked(s3api.listTrash).mockImplementation(async (_id, opts) =>
      opts.bucket === 'b1' ? page(manyMarkers(50)) : page([m1, m2]),
    )
    const w = mountPanel()
    await flushPromises()
    const wrap = w.find('.tbl-wrap')
    ;(wrap.element as HTMLElement).scrollTop = ROW_HEIGHT * 30
    await wrap.trigger('scroll')
    expect(w.findAll('tbody tr.v-row')[0].text()).toContain('mk18')

    // 切到只有 2 条标记的桶：窗口必须从头渲染，不能残留 start=18 的空窗口
    await w.findAll('select.acc-select')[1].setValue('b2')
    await flushPromises()
    const rows = w.findAll('tbody tr.v-row')
    expect(rows).toHaveLength(2)
    expect(rows[0].text()).toContain('k1')
    // 真实 DOM 滚动位置同步归零，避免下一次滚动事件把陈旧偏移写回
    expect((w.find('.tbl-wrap').element as HTMLElement).scrollTop).toBe(0)
  })

  it('scrollEl 未绑定时空安全早退；容器缺席/出现时数据重置与视口测量都正确', async () => {
    // 空标记列表 → 表格不渲染（scrollEl 为 null）
    vi.mocked(rememberedAccountId).mockReturnValue('acc-1')
    vi.mocked(s3api.listBuckets).mockResolvedValue({ buckets: [b1] })
    let w = mountPanel()
    await flushPromises()
    expect(w.find('.tbl-wrap').exists()).toBe(false)
    const idle = w.vm as unknown as { onListScroll: () => void; measureViewport: () => void; viewportH: number }
    idle.onListScroll()
    idle.measureViewport()
    expect(idle.viewportH).toBe(480)
    // 空态点刷新：markers 重置触发归零，但无滚动容器 → 守卫不抛错
    await findButton(w, 'common.refresh').trigger('click')
    await flushPromises()
    expect(w.text()).toContain('trash.emptyHint')
    w.unmount()

    // 表格渲染后 measureViewport 取实测 clientHeight（非 0 时不走 480 兜底）
    vi.mocked(s3api.listTrash).mockResolvedValue(page(manyMarkers(5)))
    w = mountPanel()
    await flushPromises()
    const vm = w.vm as unknown as { measureViewport: () => void; viewportH: number }
    Object.defineProperty(w.find('.tbl-wrap').element, 'clientHeight', { value: 600, configurable: true })
    vm.measureViewport()
    expect(vm.viewportH).toBe(600)
  })

  it('环境无 ResizeObserver 时列表仍正常渲染（不注册观察者）', async () => {
    vi.stubGlobal('ResizeObserver', undefined)
    try {
      vi.mocked(rememberedAccountId).mockReturnValue('acc-1')
      vi.mocked(s3api.listBuckets).mockResolvedValue({ buckets: [b1] })
      vi.mocked(s3api.listTrash).mockResolvedValue(page(manyMarkers(3)))
      const w = mountPanel()
      await flushPromises()
      expect(w.findAll('tbody tr.v-row')).toHaveLength(3)
      expect(() => w.unmount()).not.toThrow()
    } finally {
      vi.stubGlobal('ResizeObserver', RO)
    }
  })

  it('unmount 时安全断开 ResizeObserver（onBeforeUnmount 分支）', async () => {
    vi.mocked(rememberedAccountId).mockReturnValue('acc-1')
    vi.mocked(s3api.listBuckets).mockResolvedValue({ buckets: [b1] })
    vi.mocked(s3api.listTrash).mockResolvedValue(page(manyMarkers(3)))
    const w = mountPanel()
    await flushPromises()
    expect(w.find('.tbl-wrap').exists()).toBe(true)
    expect(() => w.unmount()).not.toThrow()
  })
})
