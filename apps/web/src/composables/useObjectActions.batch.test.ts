// useObjectActions.batch.test.ts —— 自 useObjectActions.test.ts 拆出（KNOWN_ISSUES #60）：
// 批量签名链接、复制/移动、删除文件夹（异步任务 + SSE）、批量改元数据。断言逐字搬移。
import { computed, defineComponent, ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useObjectActions, type ObjectBrowserCtx } from './useObjectActions'
import { uploadObject, type UploadTarget } from '../upload'
import { api, s3api, subscribeMigrateEvents, type MigrateProgress } from '../api'
import { toast, createProgressToast } from '../store'
import { confirmDialog } from '../confirm'
import { promptDialog } from '../prompt'
import { copyText } from '../clipboard'
import type { Account, Entry, ObjectItem } from '../types'

vi.mock('../api', () => ({
  s3api: {
    deleteObjects: vi.fn(),
    presign: vi.fn(),
    headObject: vi.fn(),
    renameObject: vi.fn(),
    mkdirObject: vi.fn(),
    downloadZipToDisk: vi.fn(),
    deletePrefixAsync: vi.fn(),
    copyPrefixAsync: vi.fn(),
    copyFilesAsync: vi.fn(),
  },
  api: { base: '', isTauri: false, getActiveServer: () => null, token: '' },
  subscribeMigrateEvents: vi.fn(),
}))
vi.mock('../upload', () => ({ uploadObject: vi.fn() }))
vi.mock('../store', () => ({
  toast: vi.fn(),
  createProgressToast: vi.fn(() => vi.fn()),
}))
vi.mock('../confirm', () => ({ confirmDialog: vi.fn(async () => true) }))
vi.mock('../prompt', () => ({ promptDialog: vi.fn(async () => 'x') }))
vi.mock('../clipboard', () => ({ copyText: vi.fn(async () => {}) }))
vi.mock('../i18n', () => ({ t: (k: string) => k, tf: (k: string) => k }))

/** C1：代理取回必须走带 Bearer 的 fetch（`<a href>` 直连会 401 且把错误体存盘）。 */
const fetchMock = vi.fn()
vi.stubGlobal('fetch', fetchMock)

/** 默认上传实现：挂起直到手动推进（inFlight）。 */
type UploadImpl = (file: File, target: UploadTarget, onProgress?: (p: number) => void, signal?: AbortSignal) => Promise<void>
const defaultUploadImpl: UploadImpl = (_file, _target, _p, signal) => {
  return new Promise<void>((resolve, reject) => {
    if (signal?.aborted) {
      reject(new DOMException('Aborted', 'AbortError'))
      return
    }
    signal?.addEventListener('abort', () => reject(new DOMException('Aborted', 'AbortError')), { once: true })
    inFlight.push({ resolve, reject })
  })
}
vi.mocked(uploadObject).mockImplementation(defaultUploadImpl)

/** 挂起中的上传，测试里手动推进完成。 */
const inFlight: { resolve: () => void; reject: (e: unknown) => void }[] = []

const acc: Account = {
  id: 'a1',
  name: 'A',
  endpoint: '',
  region: '',
  accessKey: '',
  secretSet: false,
  bucket: 'b1',
  pathStyle: true,
  useSSL: true,
    createdAt: '2024-01-01T00:00:00Z',
    publicEndpoint: '',
    updatedAt: '2024-01-01T00:00:00Z',
}

function fileObj(key: string): ObjectItem {
  return { key, size: 1, lastModified: '2024-01-01', etag: 'e1', isDir: false, storageClass: 'STANDARD' }
}

function makeCtx(overrides: Partial<ObjectBrowserCtx> = {}): ObjectBrowserCtx {
  const ctx: ObjectBrowserCtx = {
    account: computed(() => acc),
    currentBucket: ref('b1'),
    selected: ref(new Set<string>()),
    fileObjects: computed(() => []),
    prefix: ref(''),
    opsBusy: ref(false),
    error: ref(''),
    ctxMenu: ref(null),
    load: vi.fn(async () => {}),
    enterPrefix: vi.fn(),
    closeCtx: vi.fn(),
    ...overrides,
  }
  lastCtx = ctx
  return ctx
}

/** 最近一次 makeCtx 的句柄：测试里设置 selected / prefix / ctxMenu / opsBusy 等。 */
let lastCtx: ObjectBrowserCtx | null = null

/** 挂一个宿主组件：setup 内调用组合式，把返回值存到外层变量供断言。 */
function makeActions(overrides: Partial<ObjectBrowserCtx> = {}): ReturnType<typeof useObjectActions> {
  return mountActionsHost(overrides).actions
}

/** 同上，但保留宿主组件句柄以便测试卸载（onBeforeUnmount）行为。 */
function mountActionsHost(overrides: Partial<ObjectBrowserCtx> = {}): {
  actions: ReturnType<typeof useObjectActions>
  unmount: () => void
} {
  let captured!: ReturnType<typeof useObjectActions>
  const Host = defineComponent({
    setup() {
      captured = useObjectActions(makeCtx(overrides))
      return () => null
    },
  })
  const wrapper = mount(Host)
  return { actions: captured, unmount: () => wrapper.unmount() }
}

/** 设置当前右键菜单条目（或清空）。 */
function setEntry(entry: Entry | null) {
  lastCtx!.ctxMenu.value = entry ? { x: 0, y: 0, entry } : null
}

beforeEach(() => {
  inFlight.length = 0
  vi.mocked(uploadObject).mockClear()
  vi.mocked(s3api.deleteObjects).mockReset()
  vi.mocked(s3api.presign).mockReset()
  vi.mocked(s3api.headObject).mockReset()
  vi.mocked(s3api.renameObject).mockReset()
  vi.mocked(s3api.mkdirObject).mockReset()
  vi.mocked(s3api.downloadZipToDisk).mockReset()
  vi.mocked(s3api.deletePrefixAsync).mockReset()
  vi.mocked(subscribeMigrateEvents).mockReset()
  vi.mocked(confirmDialog).mockReset()
  vi.mocked(confirmDialog).mockResolvedValue(true)
  vi.mocked(promptDialog).mockReset()
  vi.mocked(promptDialog).mockResolvedValue(null)
  vi.mocked(copyText).mockReset()
  vi.mocked(copyText).mockResolvedValue(undefined)
  vi.mocked(toast).mockClear()
  vi.mocked(createProgressToast).mockClear()
  vi.mocked(createProgressToast).mockReturnValue(vi.fn())
  // fetch 桩（下游可能触发代理下载）；token 默认开启以断言 Bearer 头
  fetchMock.mockReset()
  api.token = 'tok123'
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe('批量签名链接', () => {
  it('copySelectedLinks 无选中直接返回', async () => {
    const actions = makeActions()
    await actions.copySelectedLinks()
    expect(vi.mocked(s3api.presign)).not.toHaveBeenCalled()
    expect(vi.mocked(copyText)).not.toHaveBeenCalled()
  })

  it('copySelectedLinks 全部成功：每行一个 URL 复制', async () => {
    const o1 = fileObj('a.txt')
    const o2 = fileObj('b.txt')
    const actions = makeActions({
      fileObjects: computed(() => [o1, o2]),
      selected: ref(new Set(['a.txt', 'b.txt'])),
    })
    vi.mocked(s3api.presign).mockImplementation((_id, body) =>
      Promise.resolve({
        method: 'get',
        bucket: 'b1',
        key: body.key,
        url: `https://s/${body.key}`,
        expiresIn: 3600,
      }),
    )
    await actions.copySelectedLinks()
    await flushPromises()
    expect(vi.mocked(copyText)).toHaveBeenCalledWith('https://s/a.txt\nhttps://s/b.txt')
    expect(vi.mocked(toast)).toHaveBeenCalledWith('objects.toastCopiedLinks')
    expect(lastCtx!.error.value).toBe('')
  })

  it('copySelectedLinks 部分失败：成功的复制，失败写错误', async () => {
    const o1 = fileObj('a.txt')
    const o2 = fileObj('b.txt')
    const actions = makeActions({
      fileObjects: computed(() => [o1, o2]),
      selected: ref(new Set(['a.txt', 'b.txt'])),
    })
    vi.mocked(s3api.presign).mockImplementation((_id, body) =>
      body.key === 'b.txt' ? Promise.reject(new Error('no')) : Promise.resolve({ url: 'u1' } as Awaited<ReturnType<typeof s3api.presign>>),
    )
    await actions.copySelectedLinks()
    await flushPromises()
    expect(vi.mocked(copyText)).toHaveBeenCalledWith('u1')
    expect(vi.mocked(toast)).toHaveBeenCalledWith('objects.toastCopiedLinks')
    // 断言用的是该 key；「该 key 确有定义」由 i18n/coverage.test.ts 静态扫描兜底
    // （本文件的 i18n mock 回显 key，无法在此发现缺失键）。
    expect(lastCtx!.error.value).toBe('objects.toastCopyFailed')
  })

  it('copySelectedLinks 全部失败：不复制、仅写错误', async () => {
    const o1 = fileObj('a.txt')
    const actions = makeActions({
      fileObjects: computed(() => [o1]),
      selected: ref(new Set(['a.txt'])),
    })
    vi.mocked(s3api.presign).mockRejectedValue(new Error('no'))
    await actions.copySelectedLinks()
    await flushPromises()
    expect(vi.mocked(copyText)).not.toHaveBeenCalled()
    expect(lastCtx!.error.value).toBe('objects.toastCopyFailed')
  })

  it('copySelectedLinks 账号丢失：走外层 catch', async () => {
    const o1 = fileObj('a.txt')
    const actions = makeActions({
      account: computed(() => undefined),
      fileObjects: computed(() => [o1]),
      selected: ref(new Set(['a.txt'])),
    })
    await actions.copySelectedLinks()
    expect(lastCtx!.error.value).toBe('no active account')
  })

  it('copySelectedLinks 大量选中：presign 并发不超过 4（复用有界池）', async () => {
    const objs = Array.from({ length: 12 }, (_, i) => fileObj(`k${i}.txt`))
    const actions = makeActions({
      fileObjects: computed(() => objs),
      selected: ref(new Set(objs.map((o) => o.key))),
    })
    let active = 0
    let peak = 0
    vi.mocked(s3api.presign).mockImplementation(async (_id, body) => {
      active++
      peak = Math.max(peak, active)
      await new Promise((r) => setTimeout(r, 5))
      active--
      return { method: 'get', bucket: 'b1', key: body.key, url: `https://s/${body.key}`, expiresIn: 3600 }
    })
    await actions.copySelectedLinks()
    await flushPromises()
    expect(peak).toBeLessThanOrEqual(4)
    expect(vi.mocked(s3api.presign)).toHaveBeenCalledTimes(12)
    expect(vi.mocked(copyText)).toHaveBeenCalledWith(objs.map((o) => `https://s/${o.key}`).join('\n'))
  })
})

describe('复制 / 移动', () => {
  it('ctxCopyFolder / ctxMoveFolder：仅文件夹条目', () => {
    const actions = makeActions()
    setEntry({ kind: 'file', key: 'a.txt', name: 'a.txt', object: fileObj('a.txt') })
    actions.ctxCopyFolder()
    expect(actions.destOpen.value).toBe(false)
    actions.ctxMoveFolder()
    expect(actions.destOpen.value).toBe(false)
    setEntry(null)
    actions.ctxCopyFolder()
    expect(actions.destOpen.value).toBe(false)

    setEntry({ kind: 'folder', key: 'dir/', name: 'dir' })
    actions.ctxCopyFolder()
    expect(actions.destCtx.value).toEqual({ mode: 'copy', kind: 'folder', key: 'dir/' })
    expect(actions.destOpen.value).toBe(true)
    actions.ctxMoveFolder()
    expect(actions.destCtx.value).toEqual({ mode: 'move', kind: 'folder', key: 'dir/' })
  })

  it('ctxCopyFile / ctxMoveFile：仅文件条目', () => {
    const actions = makeActions()
    setEntry({ kind: 'folder', key: 'dir/', name: 'dir' })
    actions.ctxCopyFile()
    expect(actions.destOpen.value).toBe(false)
    actions.ctxMoveFile()
    expect(actions.destOpen.value).toBe(false)
    setEntry(null)
    actions.ctxMoveFile()
    expect(actions.destOpen.value).toBe(false)

    setEntry({ kind: 'file', key: 'a.txt', name: 'a.txt', object: fileObj('a.txt') })
    actions.ctxCopyFile()
    expect(actions.destCtx.value).toEqual({ mode: 'copy', kind: 'file', key: 'a.txt' })
    actions.ctxMoveFile()
    expect(actions.destCtx.value).toEqual({ mode: 'move', kind: 'file', key: 'a.txt' })
  })

  it('openDest / openDestMulti / onDestSubmit', () => {
    const o1 = fileObj('a.txt')
    const o2 = fileObj('b.txt')
    const actions = makeActions({
      fileObjects: computed(() => [o1, o2]),
      selected: ref(new Set(['a.txt'])),
    })
    actions.openDest('move', 'file', 'a.txt')
    expect(actions.destCtx.value).toEqual({ mode: 'move', kind: 'file', key: 'a.txt' })
    expect(actions.destOpen.value).toBe(true)

    actions.openDestMulti('copy')
    expect(actions.destCtx.value).toEqual({ mode: 'copy', kind: 'multi', key: '', keys: ['a.txt'] })

    actions.onDestSubmit()
    expect(actions.destOpen.value).toBe(false)
    expect(lastCtx!.load).toHaveBeenCalledWith(true)

    // 无选中：不打开
    const empty = makeActions()
    empty.openDestMulti('copy')
    expect(empty.destOpen.value).toBe(false)
  })
})

describe('删除文件夹（异步任务 + SSE 进度）', () => {
  it('非文件夹 / 无条目 / 忙时忽略', () => {
    const actions = makeActions()
    setEntry({ kind: 'file', key: 'a.txt', name: 'a.txt' })
    actions.ctxDeleteFolder()
    expect(vi.mocked(s3api.deletePrefixAsync)).not.toHaveBeenCalled()

    setEntry({ kind: 'folder', key: 'dir/', name: 'dir' })
    lastCtx!.opsBusy.value = true
    actions.ctxDeleteFolder()
    expect(vi.mocked(s3api.deletePrefixAsync)).not.toHaveBeenCalled()
    lastCtx!.opsBusy.value = false

    setEntry(null)
    actions.ctxDeleteFolder()
    expect(vi.mocked(s3api.deletePrefixAsync)).not.toHaveBeenCalled()
  })

  it('确认取消则不删除', async () => {
    const actions = makeActions()
    setEntry({ kind: 'folder', key: 'dir/', name: 'dir' })
    vi.mocked(confirmDialog).mockResolvedValueOnce(false)
    await actions.ctxDeleteFolder()
    expect(vi.mocked(s3api.deletePrefixAsync)).not.toHaveBeenCalled()
  })

  it('成功：进度事件更新 toast，done 后汇总 + 刷新', async () => {
    let onProgress: ((p: Partial<MigrateProgress>) => void) | undefined
    let onError: ((err: Error) => void) | undefined
    const stop = vi.fn()
    vi.mocked(subscribeMigrateEvents).mockImplementation((_jobId, p, e) => {
      onProgress = p as (p: Partial<MigrateProgress>) => void
      onError = e
      return stop
    })
    vi.mocked(s3api.deletePrefixAsync).mockResolvedValueOnce({ jobId: 'j1', total: 10, truncated: true })
    const delProgress = vi.fn()
    vi.mocked(createProgressToast).mockReturnValue(delProgress)

    const actions = makeActions()
    setEntry({ kind: 'folder', key: 'dir/', name: 'dir' })
    const run = actions.ctxDeleteFolder()
    await vi.waitFor(() => expect(onProgress).toBeTruthy())
    expect(onError).toBeTruthy()
    expect(vi.mocked(toast)).toHaveBeenCalledWith('dest.toastDeletingFolder')
    expect(lastCtx!.opsBusy.value).toBe(true)

    // 进行中事件：更新进度 toast
    onProgress!({ migrated: 5, done: 5, total: 10, status: 'running' })
    expect(delProgress).toHaveBeenCalledWith('dest.toastDeleteProgress')
    // total=0 不更新进度
    onProgress!({ migrated: 0, done: 0, total: 0, status: 'running' })
    expect(delProgress.mock.calls.length).toBe(1)
    // done 终态：resolve
    onProgress!({ migrated: 5, done: 5, total: 10, status: 'done' })
    await run
    expect(stop).toHaveBeenCalled()
    expect(vi.mocked(toast)).toHaveBeenCalledWith('dest.toastDeletedTruncated')
    expect(lastCtx!.opsBusy.value).toBe(false)
    expect(lastCtx!.load).toHaveBeenCalledWith(true)
  })

  it('cancelled 终态：按删除数量 toast（非 truncated）', async () => {
    let onProgress: ((p: Partial<MigrateProgress>) => void) | undefined
    vi.mocked(subscribeMigrateEvents).mockImplementation((_jobId, p) => {
      onProgress = p as (p: Partial<MigrateProgress>) => void
      return () => {}
    })
    vi.mocked(s3api.deletePrefixAsync).mockResolvedValueOnce({ jobId: 'j2', total: 3 })
    const actions = makeActions()
    setEntry({ kind: 'folder', key: 'dir/', name: 'dir' })
    const run = actions.ctxDeleteFolder()
    await vi.waitFor(() => expect(onProgress).toBeTruthy())
    onProgress!({ status: 'cancelled' }) // 无 migrated 字段 → ?? 0
    await run
    expect(vi.mocked(toast)).toHaveBeenCalledWith('dest.toastDeletedN')
    expect(lastCtx!.opsBusy.value).toBe(false)
  })

  it('SSE 错误：reject 并写入错误', async () => {
    let onError: ((err: Error) => void) | undefined
    vi.mocked(subscribeMigrateEvents).mockImplementation((_jobId, _p, e) => {
      onError = e
      return () => {}
    })
    vi.mocked(s3api.deletePrefixAsync).mockResolvedValueOnce({ jobId: 'j3', total: 1 })
    const actions = makeActions()
    setEntry({ kind: 'folder', key: 'dir/', name: 'dir' })
    const run = actions.ctxDeleteFolder()
    await vi.waitFor(() => expect(onError).toBeTruthy())
    onError!(new Error('sse fail'))
    await run
    expect(lastCtx!.error.value).toBe('sse fail')
    expect(lastCtx!.opsBusy.value).toBe(false)
  })

  it('组件卸载时断开进行中的 SSE 订阅，并让等待中的 Promise settle', async () => {
    let onProgress: ((p: Partial<MigrateProgress>) => void) | undefined
    const stop = vi.fn()
    vi.mocked(subscribeMigrateEvents).mockImplementation((_jobId, p) => {
      onProgress = p as (p: Partial<MigrateProgress>) => void
      return stop
    })
    vi.mocked(s3api.deletePrefixAsync).mockResolvedValueOnce({ jobId: 'j5', total: 2 })
    const { actions, unmount } = mountActionsHost()
    setEntry({ kind: 'folder', key: 'dir/', name: 'dir' })

    const run = actions.ctxDeleteFolder()
    await vi.waitFor(() => expect(onProgress).toBeTruthy())
    expect(stop).not.toHaveBeenCalled()

    unmount()
    // 卸载必须让等待中的 Promise 以中止收尾（旧实现 await 永远挂起 = 帧泄漏）
    await expect(run).resolves.toBeUndefined()
    expect(stop).toHaveBeenCalledTimes(1)
    expect(lastCtx!.error.value).toBe('') // 中止不算错误，不写 ctx.error
    expect(lastCtx!.opsBusy.value).toBe(false) // finally 照常复位
  })
})

describe('批量改元数据', () => {
  it('openBatch：无选中忽略；onBatchDone 关闭并刷新', async () => {
    const actions = makeActions()
    actions.openBatch()
    expect(actions.batchOpen.value).toBe(false)

    const a2 = makeActions({ selected: ref(new Set(['a.txt'])) })
    a2.openBatch()
    expect(a2.batchOpen.value).toBe(true)
    await a2.onBatchDone()
    expect(a2.batchOpen.value).toBe(false)
    expect(lastCtx!.load).toHaveBeenCalledWith(true)
  })
})
