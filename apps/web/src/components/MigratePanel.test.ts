// MigratePanel.test.ts —— 迁移面板主流程（加载 / 选择 / 授权 / 桶与前缀 / 进度与结果）。
// KNOWN_ISSUES #60 拆分后的防御与分支范围见 MigratePanel.branches.test.ts（断言逐字搬移）。
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import MigratePanel from './MigratePanel.vue'
import { s3api, subscribeMigrateEvents } from '../api'
import type { MigrateProgress } from '../api'
import { currentAccount, requestTab, selectAccount, state, toast } from '../store'
import { ROW_HEIGHT } from '../virtualList'
import type { Account, ListObjectsResponse, ObjectItem } from '../types'

/** MigratePanel 通过 defineExpose 暴露给测试的成员（组件 setup 状态无公开类型）。 */
interface MigrateVm {
  loadAllSourceObjects: () => Promise<void>
  loadSourceBuckets?: () => Promise<void> | void
  migrate: () => Promise<void>
  cancelMigrate?: () => Promise<void>
  cancelling: boolean
  ensureTargetAccount: () => void
  targetAccountId: string | undefined
  error: unknown
  selected?: Set<string>
  onListScroll: () => void
  measureViewport: () => void
  viewportH: number
}

vi.mock('../api', () => ({
  s3api: {
    listBuckets: vi.fn(),
    listObjects: vi.fn(),
    migrateAsync: vi.fn(),
    migrateJobs: vi.fn(),
    migrateJobStatus: vi.fn(),
    migrateJobCancel: vi.fn(),
  },
  subscribeMigrateEvents: vi.fn(() => () => {}),
}))

vi.mock('../store', async () => {
  const { reactive } = await import('vue')
  return {
    state: reactive({ accounts: [] as Account[], currentAccountId: '' }),
    currentAccount: vi.fn(),
    toast: vi.fn(),
    selectAccount: vi.fn(),
    requestTab: vi.fn(),
  }
})

vi.mock('../i18n', () => ({
  t: (k: string) => k,
  tf: vi.fn((k: string) => k),
}))

const acc1: Account = {
  id: 'acc-1', name: 'acc-one', endpoint: 'http://minio:9000', region: 'r',
  accessKey: 'ak', secretSet: true, bucket: 'src-bucket', pathStyle: true, useSSL: false,
}
const acc2: Account = { ...acc1, id: 'acc-2', name: 'acc-two', bucket: 'dst-bucket' }

const objA: ObjectItem = { key: 'a.txt', size: 10, lastModified: '2024-01-01', etag: 'e1', isDir: false }
const objB: ObjectItem = { key: 'b.bin', size: 20, lastModified: '2024-01-02', etag: 'e2', isDir: false }
const objDir: ObjectItem = { key: 'dir/', size: 0, lastModified: '', etag: '', isDir: true }

const ModalDialogStub = {
  name: 'ModalDialog',
  props: ['open', 'title', 'width'],
  template: '<div class="dlg-stub"><slot /></div>',
}

let mounted: ReturnType<typeof mount> | undefined
afterEach(() => {
  mounted?.unmount()
  mounted = undefined
})

function mountPanel() {
  mounted = mount(MigratePanel, { global: { stubs: { ModalDialog: ModalDialogStub } } })
  return mounted
}

function findButton(w: ReturnType<typeof mount>, text: string) {
  const btn = w.findAll('button').find((b) => b.text() === text)
  expect(btn, `button "${text}" should exist`).toBeTruthy()
  return btn!
}

/* 与组件共用同一行高来源（src/virtualList.ts）。 */
const ROW = ROW_HEIGHT

/** 迁移测试用对象（分页 / 分片场景批量构造）。 */
function makeObj(key: string): ObjectItem {
  return { key, size: 1, lastModified: '2024-01-01', etag: 'e', isDir: false }
}

function findButtonStartsWith(w: ReturnType<typeof mount>, prefix: string) {
  const btn = w.findAll('button').find((b) => b.text().startsWith(prefix))
  expect(btn, `button starting with "${prefix}" should exist`).toBeTruthy()
  return btn!
}

beforeEach(() => {
  vi.clearAllMocks()
  state.accounts = [acc1, acc2]
  state.currentAccountId = 'acc-1'
  vi.mocked(currentAccount).mockImplementation(() =>
    state.accounts.find((a) => a.id === state.currentAccountId),
  )
  vi.mocked(s3api.listBuckets).mockResolvedValue({ buckets: [{ name: 'b-one', creationDate: '2024-01-01' }] })
  vi.mocked(s3api.migrateJobs).mockResolvedValue({ jobs: [] })
  vi.mocked(s3api.listObjects).mockResolvedValue({
    objects: [objA, objB, objDir], commonPrefixes: [], isTruncated: false, nextToken: '',
  })
})

describe('MigratePanel', () => {
  it('shows empty state without an account and skips loading', async () => {
    state.accounts = []
    state.currentAccountId = ''
    const w = mountPanel()
    await flushPromises()
    expect(w.find('.empty').text()).toContain('migrate.needAccount')
    expect(s3api.listBuckets).not.toHaveBeenCalled()
    expect(s3api.listObjects).not.toHaveBeenCalled()
  })

  it('无源账号时 loadAllSourceObjects / migrate 走守卫直接返回', async () => {
    state.accounts = []
    state.currentAccountId = ''
    const w = mountPanel()
    await flushPromises()
    await (w.vm as unknown as { loadAllSourceObjects: () => Promise<void> }).loadAllSourceObjects()
    await (w.vm as unknown as { migrate: () => Promise<void> }).migrate()
    expect(s3api.listObjects).not.toHaveBeenCalled()
    expect(s3api.migrateAsync).not.toHaveBeenCalled()
  })

  it('renders source/target selects and lists objects excluding folders', async () => {
    const w = mountPanel()
    await flushPromises()
    // 源加载 + 目标加载（onMounted 显式调用 + targetAccountId watch 兜底）
    expect(s3api.listBuckets).toHaveBeenCalledWith('acc-1')
    expect(s3api.listBuckets).toHaveBeenCalledWith('acc-2')
    expect(s3api.listObjects).toHaveBeenCalledWith('acc-1', {
      bucket: '', prefix: '', delimiter: '/', maxKeys: '200',
    })
    const selects = w.findAll('select')
    expect(selects).toHaveLength(3)
    // 源 bucket 下拉：默认 + 桶列表
    const srcOpts = selects[0].findAll('option').map((o) => o.text())
    expect(srcOpts).toEqual(['migrate.defaultBucket', 'b-one'])
    // 目标账号下拉：两个账号，源账号带 sameAccount 后缀
    const tgtOpts = selects[1].findAll('option').map((o) => o.text())
    expect(tgtOpts).toEqual(['acc-one' + 'migrate.sameAccount', 'acc-two'])
    // 行渲染：目录被过滤
    const rows = w.findAll('.v-row')
    expect(rows).toHaveLength(2)
    expect(w.text()).toContain('a.txt')
    expect(w.text()).not.toContain('dir/')
  })

  it('toggle/selectAll sync selection and migrate button enablement', async () => {
    const w = mountPanel()
    await flushPromises()
    const start = findButton(w, 'migrate.start')
    expect(start.attributes('disabled')).toBeDefined()
    // 勾选一行
    await w.findAll('.v-row input[type="checkbox"]')[0].setValue(true)
    await nextTick()
    expect(findButton(w, 'migrate.start').attributes('disabled')).toBeUndefined()
    // 全选 → 2 行选中；取消勾选 → 清空
    const allCb = w.find('.toolbar input[type="checkbox"]')
    await allCb.setValue(true)
    await nextTick()
    expect(w.findAll('tr.selected')).toHaveLength(2)
    await allCb.setValue(false)
    await nextTick()
    expect(w.findAll('tr.selected')).toHaveLength(0)
  })

  it('prefix input + list button/keyup.enter reload listings', async () => {
    const w = mountPanel()
    await flushPromises()
    const pInput = w.find('input[placeholder="migrate.prefixPlaceholder"]')
    await pInput.setValue('sub/')
    await findButton(w, 'migrate.listObjects').trigger('click')
    await flushPromises()
    expect(s3api.listObjects).toHaveBeenLastCalledWith('acc-1', {
      bucket: '', prefix: 'sub/', delimiter: '/', maxKeys: '200',
    })
    await pInput.trigger('keyup.enter')
    await flushPromises()
    expect(s3api.listObjects).toHaveBeenCalledTimes(3)
  })

  it('shows loading skeleton while listing and clears error afterwards', async () => {
    let resolveObjects!: (v: ListObjectsResponse) => void
    const w = mountPanel()
    await flushPromises()
    vi.mocked(s3api.listObjects).mockReturnValueOnce(new Promise((res) => (resolveObjects = res)))
    await findButton(w, 'migrate.listObjects').trigger('click')
    await nextTick()
    expect(w.find('[aria-busy="true"]').exists()).toBe(true)
    expect(w.findAll('.skel-row')).toHaveLength(4)
    resolveObjects({ objects: [objA], commonPrefixes: [], isTruncated: false, nextToken: '' })
    await flushPromises()
    expect(w.findAll('.v-row')).toHaveLength(1)
    expect(w.find('[aria-busy="true"]').exists()).toBe(false)
  })

  it('surfaces listObjects error in the error banner', async () => {
    vi.mocked(s3api.listObjects).mockRejectedValueOnce(new Error('list failed'))
    const w = mountPanel()
    await flushPromises()
    expect(w.find('.msg.err').text()).toContain('list failed')
  })

  it('loadSourceBuckets/loadTargetBuckets catch clears their bucket lists', async () => {
    // 第一次调用（源）失败 → sourceBuckets 被清空
    vi.mocked(s3api.listBuckets).mockRejectedValueOnce(new Error('src boom'))
    const w = mountPanel()
    await flushPromises()
    const srcSel = w.findAll('select')[0]
    expect(srcSel.findAll('option')).toHaveLength(1) // 仅默认

    // 重新挂载：源成功、目标失败 → targetBuckets 被清空
    w.unmount()
    vi.mocked(s3api.listBuckets)
      .mockResolvedValueOnce({ buckets: [{ name: 'b-one', creationDate: '2024-01-01' }] })
      .mockRejectedValueOnce(new Error('dst boom'))
    const w2 = mountPanel()
    await flushPromises()
    const tgtSel = w2.findAll('select')[1]
    expect(tgtSel.findAll('option')).toHaveLength(2) // 仅两个账号 option，无桶
  })

  it('listAll paginates with continuationToken and toasts listedAll', async () => {
    const w = mountPanel()
    await flushPromises()
    vi.mocked(s3api.listObjects).mockReset()
    vi.mocked(s3api.listObjects)
      .mockResolvedValueOnce({ objects: [objA], commonPrefixes: [], isTruncated: true, nextToken: 't1' })
      .mockResolvedValueOnce({ objects: [objB], commonPrefixes: [], isTruncated: false, nextToken: '' })
    await findButton(w, 'migrate.listAll').trigger('click')
    await flushPromises()
    expect(vi.mocked(s3api.listObjects).mock.calls[0][1]).toEqual({
      bucket: '', prefix: '', maxKeys: '1000',
    })
    expect(vi.mocked(s3api.listObjects).mock.calls[1][1]).toEqual({
      bucket: '', prefix: '', maxKeys: '1000', continuationToken: 't1',
    })
    expect(toast).toHaveBeenCalledWith('migrate.listedAll')
  })

  it('listAll caps at MAX_ALL_PAGES and toasts listedCap', async () => {
    const w = mountPanel()
    await flushPromises()
    vi.mocked(s3api.listObjects).mockReset()
    let page = 0
    vi.mocked(s3api.listObjects).mockImplementation(async () => ({
      objects: [{ ...objA, key: `f${page++}.dat` }], commonPrefixes: [], isTruncated: true, nextToken: 't',
    }))
    await findButton(w, 'migrate.listAll').trigger('click')
    await flushPromises()
    expect(toast).toHaveBeenCalledWith('migrate.listedCap', 'err')
    // 200 页后仍提示成功（现有行为）
    expect(toast).toHaveBeenCalledWith('migrate.listedAll')
  })

  it('listAll 续页用进入循环时的 prefix 快照，不被中途编辑污染', async () => {
    const w = mountPanel()
    await flushPromises()
    vi.mocked(s3api.listObjects).mockReset()
    let firstPage = true
    vi.mocked(s3api.listObjects).mockImplementation(async () => {
      if (firstPage) {
        firstPage = false
        // 第一页仍在途时用户改前缀：续页 token 属于旧前缀
        await w.find('input[placeholder="migrate.prefixPlaceholder"]').setValue('new/')
        return { objects: [objA], commonPrefixes: [], isTruncated: true, nextToken: 'tok-1' }
      }
      return { objects: [objB], commonPrefixes: [], isTruncated: false, nextToken: '' }
    })
    await w.find('input[placeholder="migrate.prefixPlaceholder"]').setValue('old/')
    await findButton(w, 'migrate.listAll').trigger('click')
    await flushPromises()
    const calls = vi.mocked(s3api.listObjects).mock.calls
    expect(calls).toHaveLength(2)
    expect(calls[0][1].prefix).toBe('old/')
    expect(calls[1][1]).toEqual({ bucket: '', prefix: 'old/', maxKeys: '1000', continuationToken: 'tok-1' })
  })

  it('listAll 期间用户重新列出：过期响应被丢弃，不混入新前缀', async () => {
    const w = mountPanel()
    await flushPromises()
    vi.mocked(s3api.listObjects).mockReset()
    let releaseAll!: (v: ListObjectsResponse) => void
    vi.mocked(s3api.listObjects)
      .mockImplementationOnce(() => new Promise((res) => (releaseAll = res)))
      .mockResolvedValueOnce({ objects: [objA, objB], commonPrefixes: [], isTruncated: false, nextToken: '' })
    const allClick = findButton(w, 'migrate.listAll').trigger('click')
    // listAll 第一页仍在途 → 用户点「列出对象」切换到新前缀
    await findButton(w, 'migrate.listObjects').trigger('click')
    await flushPromises()
    releaseAll({ objects: [{ ...objA, key: 'stale.dat' }], commonPrefixes: [], isTruncated: false, nextToken: '' })
    await allClick
    await flushPromises()
    expect(w.text()).toContain('a.txt')
    expect(w.text()).not.toContain('stale.dat')
  })

  it('载入完成前切换账号：过期响应被丢弃，不覆盖新账号的列表', async () => {
    vi.mocked(s3api.listObjects).mockReset()
    vi.mocked(s3api.listObjects).mockResolvedValue({ objects: [], commonPrefixes: [], isTruncated: false, nextToken: '' })
    const w = mountPanel()
    await flushPromises()

    let resolveStale!: (v: ListObjectsResponse) => void
    vi.mocked(s3api.listObjects).mockImplementationOnce(() => new Promise((res) => (resolveStale = res)))
    const btn = findButton(w, 'migrate.listObjects')
    await btn.trigger('click')
    // 切换账号触发第二次列举（此时第一次仍在途）
    vi.mocked(s3api.listObjects).mockResolvedValue({ objects: [objA], commonPrefixes: [], isTruncated: false, nextToken: '' })
    state.currentAccountId = 'acc-2'
    await flushPromises()
    expect(w.findAll('.v-row')).toHaveLength(1)

    // 旧账号的响应迟到：必须被丢弃，不得覆盖新账号的列表
    resolveStale({ objects: [{ ...objA, key: 'stale.dat' }], commonPrefixes: [], isTruncated: false, nextToken: '' })
    await flushPromises()
    expect(w.text()).toContain('a.txt')
    expect(w.text()).not.toContain('stale.dat')
  })

  it('列举全部期间切换账号：过期分页请求的失败被丢弃，不污染错误条', async () => {
    vi.mocked(s3api.listObjects).mockReset()
    vi.mocked(s3api.listObjects).mockResolvedValue({ objects: [], commonPrefixes: [], isTruncated: false, nextToken: '' })
    const w = mountPanel()
    await flushPromises()

    let rejectStale!: (e: Error) => void
    vi.mocked(s3api.listObjects).mockImplementationOnce(() => new Promise((_res, rej) => (rejectStale = rej)))
    await findButton(w, 'migrate.listAll').trigger('click')
    // 切换账号 → 新的列举代次，旧请求作废
    state.currentAccountId = 'acc-2'
    await flushPromises()

    rejectStale(new Error('stale page failure'))
    await flushPromises()
    expect(w.find('.msg.err').exists()).toBe(false)
    expect(w.find('[aria-busy="true"]').exists()).toBe(false)
  })

  it('载入完成前切换账号：过期发起方的失败被丢弃，不覆盖新账号的列表', async () => {
    vi.mocked(s3api.listObjects).mockReset()
    vi.mocked(s3api.migrateAsync).mockReset()
    vi.mocked(subscribeMigrateEvents).mockReset()
    vi.mocked(s3api.migrateJobStatus).mockReset()
    vi.mocked(s3api.listObjects).mockResolvedValue({ objects: [], commonPrefixes: [], isTruncated: false, nextToken: '' })
    const w = mountPanel()
    await flushPromises()

    let rejectStale!: (e: Error) => void
    vi.mocked(s3api.listObjects).mockImplementationOnce(() => new Promise((_res, rej) => (rejectStale = rej)))
    await findButton(w, 'migrate.listObjects').trigger('click')
    // 切换账号 → 新的列举代次，旧请求作废并返回新账号的列表
    vi.mocked(s3api.listObjects).mockResolvedValue({ objects: [objA], commonPrefixes: [], isTruncated: false, nextToken: '' })
    state.currentAccountId = 'acc-2'
    await flushPromises()
    expect(w.text()).toContain('a.txt')

    // 旧请求此刻才失败：必须被丢弃，不得清空/污染新账号的列表
    rejectStale(new Error('stale failure'))
    await flushPromises()
    expect(w.text()).toContain('a.txt')
    expect(w.find('.msg.err').exists()).toBe(false)
  })

  it('迁移选中超过服务端单次上限（10000）时按 10000 分片串行提交并聚合结果', async () => {
    // 分页由 continuationToken 驱动（不依赖第几次调用）：每页 1000 条直到列举完。
    const objs = Array.from({ length: 20001 }, (_, i) => makeObj(`f${String(i).padStart(5, '0')}.bin`))
    vi.mocked(s3api.listObjects).mockImplementation(async (_id, q) => {
      const from = Number((q as { continuationToken?: string }).continuationToken ?? 0)
      const to = Math.min(objs.length, from + 1000)
      return {
        objects: objs.slice(from, to), commonPrefixes: [],
        isTruncated: to < objs.length, nextToken: to < objs.length ? String(to) : '',
      }
    })
    let job = 0
    vi.mocked(s3api.migrateAsync).mockImplementation(async () => ({ jobId: `job-${++job}`, total: 1 }))
    vi.mocked(s3api.migrateJobStatus).mockResolvedValue({
      progress: { status: 'done' }, result: { migrated: 1, failed: 0 },
    } as Awaited<ReturnType<typeof s3api.migrateJobStatus>>)
    vi.mocked(subscribeMigrateEvents).mockImplementation((_id, onP) => {
      queueMicrotask(() => onP({ status: 'done', done: 1, total: 1, migrated: 1, failed: 0 }))
      return () => {}
    })

    const w = mountPanel()
    await flushPromises()
    await findButton(w, 'migrate.listAll').trigger('click')
    await flushPromises()
    await w.find('.toolbar input[type="checkbox"]').setValue(true)
    await findButton(w, 'migrate.start').trigger('click')
    await flushPromises()

    const calls = vi.mocked(s3api.migrateAsync).mock.calls
    expect(calls).toHaveLength(3)
    for (const [body] of calls) expect(body.sourceKeys!.length).toBeLessThanOrEqual(10000)
    expect(calls[0][0].sourceKeys).toEqual(objs.slice(0, 10000).map((o) => o.key))
    expect(calls[1][0].sourceKeys).toEqual(objs.slice(10000, 20000).map((o) => o.key))
    expect(calls[2][0].sourceKeys).toEqual(objs.slice(20000).map((o) => o.key))
    // 分片结果聚合进结果弹窗与 toast：3 片 × 1 条
    expect(toast).toHaveBeenCalledWith('migrate.toastOk')
    expect(w.text()).toContain('migrate.resultTotal')
  })

  it('分片中途失败：已完成分片的结果不被吞掉（结果弹窗 + 部分成功提示）', async () => {
    const objs = Array.from({ length: 10001 }, (_, i) => makeObj(`f${String(i).padStart(5, '0')}.bin`))
    vi.mocked(s3api.listObjects).mockImplementation(async (_id, q) => {
      const from = Number((q as { continuationToken?: string }).continuationToken ?? 0)
      const to = Math.min(objs.length, from + 1000)
      return {
        objects: objs.slice(from, to), commonPrefixes: [],
        isTruncated: to < objs.length, nextToken: to < objs.length ? String(to) : '',
      }
    })
    let job = 0
    vi.mocked(s3api.migrateAsync).mockImplementation(async () => {
      job++
      if (job === 2) throw new Error('batch 2 rejected')
      return { jobId: `job-${job}`, total: 1 }
    })
    vi.mocked(s3api.migrateJobStatus).mockResolvedValue({
      progress: { status: 'done' }, result: { migrated: 1, failed: 0 },
    } as Awaited<ReturnType<typeof s3api.migrateJobStatus>>)
    vi.mocked(subscribeMigrateEvents).mockImplementation((_id, onP) => {
      queueMicrotask(() => onP({ status: 'done', done: 1, total: 1, migrated: 1, failed: 0 }))
      return () => {}
    })

    const w = mountPanel()
    await flushPromises()
    await findButton(w, 'migrate.listAll').trigger('click')
    await flushPromises()
    await w.find('.toolbar input[type="checkbox"]').setValue(true)
    await findButton(w, 'migrate.start').trigger('click')
    await flushPromises()

    expect(vi.mocked(s3api.migrateAsync).mock.calls).toHaveLength(2)
    // 第 1 片成功（1 条）：结果必须如实展示，而不是只报「迁移失败」
    expect(toast).toHaveBeenCalledWith('migrate.toastPartial', 'err')
    expect(w.text()).toContain('migrate.resultOk')
    expect(w.find('.msg.err').text()).toContain('batch 2 rejected')
  })

  it('多分片迁移中取消：停止提交后续分片', async () => {
    const objs = Array.from({ length: 15000 }, (_, i) => makeObj(`f${String(i).padStart(5, '0')}.bin`))
    vi.mocked(s3api.listObjects).mockReset()
    vi.mocked(s3api.listObjects).mockImplementation(async (_id, q) => {
      const from = Number((q as { continuationToken?: string }).continuationToken ?? 0)
      const to = Math.min(objs.length, from + 1000)
      return {
        objects: objs.slice(from, to), commonPrefixes: [],
        isTruncated: to < objs.length, nextToken: to < objs.length ? String(to) : '',
      }
    })
    vi.mocked(s3api.migrateAsync).mockReset()
    vi.mocked(s3api.migrateAsync).mockResolvedValue({ jobId: 'job-cancel', total: 10000 } as Awaited<ReturnType<typeof s3api.migrateAsync>>)
    vi.mocked(s3api.migrateJobStatus).mockReset()
    vi.mocked(s3api.migrateJobStatus).mockResolvedValue({
      progress: { status: 'cancelled' }, result: { migrated: 0, failed: 0 },
    } as Awaited<ReturnType<typeof s3api.migrateJobStatus>>)
    vi.mocked(subscribeMigrateEvents).mockReset()
    vi.mocked(subscribeMigrateEvents).mockImplementation((_id, onP) => {
      queueMicrotask(() => onP({ status: 'cancelled', done: 0, total: 10000, migrated: 0, failed: 0 }))
      return () => {}
    })

    const w = mountPanel()
    await flushPromises()
    await findButton(w, 'migrate.listAll').trigger('click')
    await flushPromises()
    await w.find('.toolbar input[type="checkbox"]').setValue(true)
    await findButton(w, 'migrate.start').trigger('click')
    await flushPromises()

    // 第 1 片被取消 → 不再提交第 2 片
    expect(vi.mocked(s3api.migrateAsync).mock.calls).toHaveLength(1)
    expect(toast).toHaveBeenCalledWith('migrate.toastCancelled', 'err')
  })

  it('失败 key 超过 200 条时结果弹窗只保留前 200 条', async () => {
    const failedKeys = Array.from({ length: 250 }, (_, i) => `bad-${String(i).padStart(3, '0')}.txt`)
    vi.mocked(s3api.migrateAsync).mockReset()
    vi.mocked(s3api.migrateAsync).mockResolvedValue({ jobId: 'job-fails', total: 250 } as Awaited<ReturnType<typeof s3api.migrateAsync>>)
    vi.mocked(s3api.migrateJobStatus).mockReset()
    vi.mocked(s3api.migrateJobStatus).mockResolvedValue({
      progress: { status: 'done' },
      result: { migrated: 0, failed: 250, failedKeys },
    } as Awaited<ReturnType<typeof s3api.migrateJobStatus>>)
    vi.mocked(subscribeMigrateEvents).mockReset()
    vi.mocked(subscribeMigrateEvents).mockImplementation((_id, onP) => {
      queueMicrotask(() => onP({ status: 'done', done: 250, total: 250, migrated: 0, failed: 250 }))
      return () => {}
    })
    const w = mountPanel()
    await flushPromises()
    await w.find('.toolbar input[type="checkbox"]').setValue(true)
    await findButton(w, 'migrate.start').trigger('click')
    await flushPromises()

    const fails = w.findAll('.fail-item')
    expect(fails).toHaveLength(200)
    expect(fails[0].text()).toBe('bad-000.txt')
    // 超出的失败 key 被截断，不渲染进弹窗
    expect(w.text()).not.toContain('bad-249.txt')
  })

  it('多分片全部正常完成：继续提交后续分片（不提前中断）', async () => {
    const objs = Array.from({ length: 15000 }, (_, i) => makeObj(`f${String(i).padStart(5, '0')}.bin`))
    vi.mocked(s3api.listObjects).mockReset()
    // 分页由 continuationToken 驱动，不依赖第几次调用
    vi.mocked(s3api.listObjects).mockImplementation(async (_id, q) => {
      const from = Number((q as { continuationToken?: string }).continuationToken ?? 0)
      const to = Math.min(objs.length, from + 1000)
      return {
        objects: objs.slice(from, to), commonPrefixes: [],
        isTruncated: to < objs.length, nextToken: to < objs.length ? String(to) : '',
      }
    })
    let job = 0
    vi.mocked(s3api.migrateAsync).mockReset()
    vi.mocked(s3api.migrateAsync).mockImplementation(async () => ({ jobId: `job-${++job}`, total: 1 }))
    vi.mocked(s3api.migrateJobStatus).mockReset()
    vi.mocked(s3api.migrateJobStatus).mockResolvedValue({
      progress: { status: 'done' }, result: { migrated: 1, failed: 0 },
    } as Awaited<ReturnType<typeof s3api.migrateJobStatus>>)
    vi.mocked(subscribeMigrateEvents).mockReset()
    vi.mocked(subscribeMigrateEvents).mockImplementation((_id, onP) => {
      queueMicrotask(() => onP({ status: 'done', done: 1, total: 1, migrated: 1, failed: 0 }))
      return () => {}
    })

    const w = mountPanel()
    await flushPromises()
    await findButton(w, 'migrate.listAll').trigger('click')
    await flushPromises()
    await w.find('.toolbar input[type="checkbox"]').setValue(true)
    await findButton(w, 'migrate.start').trigger('click')
    await flushPromises()

    // 第 1 片（10000 条）完成且未取消 → 第 2 片（5000 条）必须继续提交
    expect(vi.mocked(s3api.migrateAsync).mock.calls.map((c) => c[0].sourceKeys!.length)).toEqual([10000, 5000])
    expect(w.find('.msg.err').exists()).toBe(false)
  })

  it('migrate runs progress, opens result dialog and goto targets', async () => {
    let progressCb!: (p: MigrateProgress) => void
    const unsub = vi.fn()
    vi.mocked(s3api.migrateAsync).mockResolvedValue({ jobId: 'j1' } as Awaited<ReturnType<typeof s3api.migrateAsync>>)
    vi.mocked(s3api.migrateJobStatus).mockResolvedValue({
      progress: { status: 'done', done: 2, total: 2, migrated: 1, failed: 1 },
      result: { migrated: 1, failed: 1, failedKeys: ['k1', 'k2', 'k3'], lastError: 'boom' },
    } as Awaited<ReturnType<typeof s3api.migrateJobStatus>>)
    vi.mocked(subscribeMigrateEvents).mockImplementation((_id, onP) => {
      progressCb = onP
      return unsub
    })

    const w = mountPanel()
    await flushPromises()
    await w.find('.toolbar input[type="checkbox"]').setValue(true)
    await findButton(w, 'migrate.start').trigger('click')
    await flushPromises()

    expect(s3api.migrateAsync).toHaveBeenCalledWith({
      sourceAccountId: 'acc-1', sourceBucket: undefined, sourceKeys: ['a.txt', 'b.bin'],
      targetAccountId: 'acc-2', targetBucket: undefined, targetPrefix: '',
    })
    expect(subscribeMigrateEvents).toHaveBeenCalledWith('j1', expect.any(Function), expect.any(Function))

    // 进度 1/2 → 50%
    progressCb({ done: 1, total: 2, migrated: 0, failed: 0, status: 'running' })
    await nextTick()
    expect(findButtonStartsWith(w, 'migrate.running').text()).toBe('migrate.running 1/2')
    expect(w.find('.progress .bar').attributes('style')).toContain('50%')

    // 完成
    progressCb({ done: 2, total: 2, migrated: 1, failed: 1, status: 'done' })
    await flushPromises()
    expect(s3api.migrateJobStatus).toHaveBeenCalledWith('j1')
    expect(unsub).toHaveBeenCalled()
    // 结果弹窗内容（ModalDialog stub 恒渲染 slot）
    expect(w.text()).toContain('k1')
    expect(w.text()).toContain('k3')
    expect(w.text()).toContain('migrate.firstError')
    expect(toast).toHaveBeenCalledWith('migrate.toastPartial', 'err')
    // 跳转目标账号
    await findButton(w, 'migrate.gotoTarget').trigger('click')
    expect(selectAccount).toHaveBeenCalledWith('acc-2')
    expect(requestTab).toHaveBeenCalledWith('objects')
    // 关闭按钮
    const closeBtn = w.findAll('button').find((b) => b.text() === 'common.close')
    expect(closeBtn).toBeTruthy()
    await closeBtn!.trigger('click')
  })

  it('migrate success without failures toasts toastOk', async () => {
    let progressCb!: (p: MigrateProgress) => void
    vi.mocked(s3api.migrateAsync).mockResolvedValue({ jobId: 'j2' } as Awaited<ReturnType<typeof s3api.migrateAsync>>)
    vi.mocked(s3api.migrateJobStatus).mockResolvedValue({
      progress: { status: 'done' }, result: { migrated: 2, failed: 0 },
    } as Awaited<ReturnType<typeof s3api.migrateJobStatus>>)
    vi.mocked(subscribeMigrateEvents).mockImplementation((_id, onP) => {
      progressCb = onP
      return () => {}
    })
    const w = mountPanel()
    await flushPromises()
    await w.find('.toolbar input[type="checkbox"]').setValue(true)
    await findButton(w, 'migrate.start').trigger('click')
    await flushPromises()
    progressCb({ done: 2, total: 2, migrated: 2, failed: 0, status: 'done' })
    await flushPromises()
    expect(toast).toHaveBeenCalledWith('migrate.toastOk')
  })

  it('cancel in-flight migration requests cancel and toasts cancelled on completion', async () => {
    let progressCb!: (p: MigrateProgress) => void
    const unsub = vi.fn()
    vi.mocked(s3api.migrateAsync).mockResolvedValue({ jobId: 'j3' } as Awaited<ReturnType<typeof s3api.migrateAsync>>)
    vi.mocked(s3api.migrateJobStatus).mockResolvedValue({
      progress: { status: 'cancelled' }, result: { migrated: 0, failed: 0 },
    } as Awaited<ReturnType<typeof s3api.migrateJobStatus>>)
    vi.mocked(s3api.migrateJobCancel).mockResolvedValue({} as Awaited<ReturnType<typeof s3api.migrateJobCancel>>)
    vi.mocked(subscribeMigrateEvents).mockImplementation((_id, onP) => {
      progressCb = onP
      return unsub
    })
    const w = mountPanel()
    await flushPromises()
    await w.find('.toolbar input[type="checkbox"]').setValue(true)
    await findButton(w, 'migrate.start').trigger('click')
    await flushPromises()
    // 在途：取消按钮出现
    const cancel = findButton(w, 'migrate.cancel')
    await cancel.trigger('click')
    await flushPromises()
    expect(s3api.migrateJobCancel).toHaveBeenCalledWith('j3')
    expect(toast).toHaveBeenCalledWith('migrate.cancelRequested')
    // 服务端最终取消状态 → toastCancelled
    progressCb({ done: 0, total: 2, migrated: 0, failed: 0, status: 'done' })
    await flushPromises()
    expect(toast).toHaveBeenCalledWith('migrate.toastCancelled', 'err')
  })

  it('cancelMigrate surfaces cancel errors', async () => {
    let progressCb!: (p: MigrateProgress) => void
    vi.mocked(s3api.migrateAsync).mockResolvedValue({ jobId: 'j4' } as Awaited<ReturnType<typeof s3api.migrateAsync>>)
    vi.mocked(s3api.migrateJobCancel).mockRejectedValueOnce(new Error('cancel failed'))
    vi.mocked(s3api.migrateJobStatus).mockResolvedValue({ progress: { status: 'done' }, result: { migrated: 0, failed: 0 } } as Awaited<ReturnType<typeof s3api.migrateJobStatus>>)
    vi.mocked(subscribeMigrateEvents).mockImplementation((_id, onP) => {
      progressCb = onP
      return () => {}
    })
    const w = mountPanel()
    await flushPromises()
    await w.find('.toolbar input[type="checkbox"]').setValue(true)
    await findButton(w, 'migrate.start').trigger('click')
    await flushPromises()
    await findButton(w, 'migrate.cancel').trigger('click')
    await flushPromises()
    expect(w.find('.msg.err').text()).toContain('cancel failed')
    // 收尾，避免泄漏未决 promise
    progressCb({ done: 1, total: 1, migrated: 0, failed: 0, status: 'done' })
    await flushPromises()
  })

  it('migrateAsync rejection surfaces error', async () => {
    vi.mocked(s3api.migrateAsync).mockRejectedValueOnce(new Error('migrate failed'))
    const w = mountPanel()
    await flushPromises()
    await w.find('.toolbar input[type="checkbox"]').setValue(true)
    await findButton(w, 'migrate.start').trigger('click')
    await flushPromises()
    expect(w.find('.msg.err').text()).toContain('migrate failed')
    // 按钮恢复可用
    expect(findButton(w, 'migrate.start').attributes('disabled')).toBeUndefined()
  })

  it('SSE onError rejects migration and surfaces the error', async () => {
    let errorCb!: (e: Error) => void
    vi.mocked(s3api.migrateAsync).mockResolvedValue({ jobId: 'j5' } as Awaited<ReturnType<typeof s3api.migrateAsync>>)
    vi.mocked(subscribeMigrateEvents).mockImplementation((_id, _onP, onErr) => {
      errorCb = onErr
      return () => {}
    })
    const w = mountPanel()
    await flushPromises()
    await w.find('.toolbar input[type="checkbox"]').setValue(true)
    await findButton(w, 'migrate.start').trigger('click')
    await flushPromises()
    errorCb(new Error('sse down'))
    await flushPromises()
    expect(w.find('.msg.err').text()).toContain('sse down')
  })

  it('unmount during in-flight migration: 断开订阅且 migrate() 以中止 settle（不悬挂）', async () => {
    const unsub = vi.fn()
    vi.mocked(s3api.migrateAsync).mockResolvedValue({ jobId: 'j6' } as Awaited<ReturnType<typeof s3api.migrateAsync>>)
    vi.mocked(subscribeMigrateEvents).mockReturnValue(unsub)
    const w = mountPanel()
    await flushPromises()
    await w.find('.toolbar input[type="checkbox"]').setValue(true)
    const run = (w.vm as unknown as MigrateVm).migrate()
    await flushPromises()
    // 仍在途 → unmount 触发 activeUnsub
    expect(unsub).not.toHaveBeenCalled()
    w.unmount()
    // 卸载必须让等待中的 Promise 以中止收尾（旧实现 await 永远挂起 = 帧泄漏）
    await expect(run).resolves.toBeUndefined()
    expect(unsub).toHaveBeenCalledTimes(1)
    expect((w.vm as unknown as MigrateVm).error).toBe('')
  })

  it('account switch watch resets state and reloads', async () => {
    const w = mountPanel()
    await flushPromises()
    state.currentAccountId = 'acc-2'
    await flushPromises()
    expect(s3api.listObjects).toHaveBeenLastCalledWith('acc-2', expect.anything())
    expect(s3api.listBuckets).toHaveBeenCalledWith('acc-2')
    // 目标账号切换 → 目标桶重置并重新加载（此时另一账号为 acc-1）
    const tgtSel = w.findAll('select')[1]
    await tgtSel.setValue('acc-2')
    await flushPromises()
    expect(s3api.listBuckets).toHaveBeenLastCalledWith('acc-2')
  })

  it('same-account migration: only one account present defaults target to source', async () => {
    state.accounts = [acc1]
    const w = mountPanel()
    await flushPromises()
    await w.find('.toolbar input[type="checkbox"]').setValue(true)
    vi.mocked(s3api.migrateAsync).mockResolvedValue({ jobId: 'j7' } as Awaited<ReturnType<typeof s3api.migrateAsync>>)
    vi.mocked(s3api.migrateJobStatus).mockResolvedValue({ progress: { status: 'done' }, result: { migrated: 1, failed: 0 } } as Awaited<ReturnType<typeof s3api.migrateJobStatus>>)
    let progressCb!: (p: MigrateProgress) => void
    vi.mocked(subscribeMigrateEvents).mockImplementation((_id, onP) => {
      progressCb = onP
      return () => {}
    })
    await findButton(w, 'migrate.start').trigger('click')
    await flushPromises()
    expect(s3api.migrateAsync).toHaveBeenCalledWith(
      expect.objectContaining({ targetAccountId: 'acc-1' }),
    )
    progressCb({ done: 1, total: 1, migrated: 1, failed: 0, status: 'done' })
    await flushPromises()
  })

  it('listAll surfaces errors in the error banner', async () => {
    const w = mountPanel()
    await flushPromises()
    vi.mocked(s3api.listObjects).mockReset()
    vi.mocked(s3api.listObjects).mockRejectedValue(new Error('collect failed'))
    await findButton(w, 'migrate.listAll').trigger('click')
    await flushPromises()
    expect(w.find('.msg.err').text()).toContain('collect failed')
  })

  it('goto targets with missing account skips selectAccount and requests tab', async () => {
    let progressCb!: (p: MigrateProgress) => void
    vi.mocked(s3api.migrateAsync).mockResolvedValue({ jobId: 'j8' } as Awaited<ReturnType<typeof s3api.migrateAsync>>)
    vi.mocked(s3api.migrateJobStatus).mockResolvedValue({
      progress: { status: 'done' }, result: { migrated: 1, failed: 0 },
    } as Awaited<ReturnType<typeof s3api.migrateJobStatus>>)
    vi.mocked(subscribeMigrateEvents).mockImplementation((_id, onP) => {
      progressCb = onP
      return () => {}
    })
    const w = mountPanel()
    await flushPromises()
    await w.find('.toolbar input[type="checkbox"]').setValue(true)
    await findButton(w, 'migrate.start').trigger('click')
    await flushPromises()
    progressCb({ done: 1, total: 1, migrated: 1, failed: 0, status: 'done' })
    await flushPromises()
    // 目标账号在迁移完成后被移除 → found 分支跳过 selectAccount
    state.accounts = [acc1]
    await findButton(w, 'migrate.gotoTarget').trigger('click')
    expect(selectAccount).not.toHaveBeenCalled()
    expect(requestTab).toHaveBeenCalledWith('objects')
  })

  it('virtual 行高由 ROW_HEIGHT 绑定到行内样式（防 CSS 字面量 38/42 漂移回归）', async () => {
    vi.mocked(s3api.listObjects).mockResolvedValue({
      objects: [
        { key: 'a.dat', size: 1, lastModified: '2024-01-01', etag: 'e', isDir: false },
        { key: 'b.dat', size: 2, lastModified: '2024-01-01', etag: 'e', isDir: false },
      ],
      commonPrefixes: [], isTruncated: false, nextToken: '',
    })
    const w = mountPanel()
    await flushPromises()
    const row = w.find('.v-row')
    expect(row.exists()).toBe(true)
    // 渲染出的行高必须来自虚拟窗口同一个常量（此前 CSS 38px vs ROW_HEIGHT=42 漂移）
    expect(row.attributes('style')).toContain(`height: ${ROW_HEIGHT}px`)
  })

  it('virtualizes long object lists and scrolls with spacer rows', async () => {
    const many = Array.from({ length: 50 }, (_, i) => ({
      key: `f${String(i).padStart(2, '0')}.dat`, size: i, lastModified: '2024-01-01',
      etag: 'e', isDir: false,
    }))
    vi.mocked(s3api.listObjects).mockReset()
    vi.mocked(s3api.listObjects).mockResolvedValue({
      objects: many, commonPrefixes: [], isTruncated: false, nextToken: '',
    })
    const w = mountPanel()
    await flushPromises()
    // viewportH=480 → ceil(480/42)+24 = 36（ROW_HEIGHT 与 CSS 行高一致）
    expect(w.findAll('.v-row')).toHaveLength(36)
    // 初始 padBottom spacer（50-36 行）
    expect(w.findAll('tbody tr.v-spacer')).toHaveLength(1)
    const wrap = w.find('.tbl-wrap')
    ;(wrap.element as HTMLElement).scrollTop = 42 * 30
    await wrap.trigger('scroll')
    await nextTick()
    // 滚动后 padTop 出现（18*42=756px）
    const spacer = w.find('tbody tr.v-spacer')
    expect(spacer.exists()).toBe(true)
    expect(spacer.find('td').attributes('style')).toContain('756px')
    expect(w.findAll('.v-row')[0].text()).toContain('f18.dat')
  })

  it('重新列出对象后虚拟窗口回到顶部（不残留旧 scrollTop）', async () => {
    const many = Array.from({ length: 50 }, (_, i) => ({
      key: `f${String(i).padStart(2, '0')}.dat`, size: i, lastModified: '2024-01-01',
      etag: 'e', isDir: false,
    }))
    vi.mocked(s3api.listObjects).mockResolvedValue({
      objects: many, commonPrefixes: [], isTruncated: false, nextToken: '',
    })
    const w = mountPanel()
    await flushPromises()
    const wrap = w.find('.tbl-wrap')
    // 滚到第 30 行：窗口起点 = 30 - OVERSCAN(12) = 18
    ;(wrap.element as HTMLElement).scrollTop = ROW * 30
    await wrap.trigger('scroll')
    expect(w.findAll('.v-row')[0].text()).toContain('f18.dat')

    // 切到只有 3 条的另一前缀 → 窗口必须从头渲染，不能空白
    vi.mocked(s3api.listObjects).mockResolvedValue({
      objects: [objA, objB], commonPrefixes: [], isTruncated: false, nextToken: '',
    })
    await findButton(w, 'migrate.listObjects').trigger('click')
    await flushPromises()
    const rows = w.findAll('.v-row')
    expect(rows).toHaveLength(2)
    expect(rows[0].text()).toContain('a.txt')
    // 注意：happy-dom 不会像真实浏览器那样在内容缩短时钳制 scrollTop，
    // 因此这里只断言「窗口回到顶部」这一外部可见行为（上面的 rows[0]）。
    expect(w.findAll('.v-row')).toHaveLength(2)
  })
})

