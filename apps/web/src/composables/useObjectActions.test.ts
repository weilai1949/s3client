import { computed, defineComponent, ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useObjectActions, type ObjectBrowserCtx } from './useObjectActions'
import { uploadObject, type UploadTarget } from '../upload'
import { s3api, subscribeMigrateEvents, type MigrateProgress } from '../api'
import { toast, createProgressToast } from '../store'
import { confirmDialog } from '../confirm'
import { promptDialog } from '../prompt'
import { copyText } from '../clipboard'
import { proxyUrl } from '../proxy'
import type { UploadItem } from '../components/UploadQueue.vue'
import type { Account, Entry, ObjectItem, ObjectMeta } from '../types'

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
vi.mock('../proxy', () => ({ proxyUrl: vi.fn(() => 'https://p') }))
vi.mock('../i18n', () => ({ t: (k: string) => k, tf: (k: string) => k }))

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
}

function fileObj(key: string): ObjectItem {
  return { key, size: 1, lastModified: '2024-01-01', etag: 'e1', contentType: 'text/plain', isDir: false }
}

function meta(key: string): ObjectMeta {
  return { key, size: 1, lastModified: '2024-01-01', etag: 'e1', contentType: 'text/plain' }
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

function mountActions(): ReturnType<typeof useObjectActions> {
  return makeActions()
}

/** 设置当前右键菜单条目（或清空）。 */
function setEntry(entry: Entry | null) {
  lastCtx!.ctxMenu.value = entry ? { x: 0, y: 0, entry } : null
}

function enqueue(actions: ReturnType<typeof useObjectActions>, name: string): UploadItem {
  const it: UploadItem = { file: new File(['x'], name), key: name, bucket: 'b1', pct: 0, status: 'pending' }
  actions.uploadQueue.value.push(it)
  return it
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
})

describe('上传取消状态机（cancelled 为终态）', () => {
  beforeEach(() => {
    inFlight.length = 0
    vi.mocked(uploadObject).mockClear()
  })

  it('中止上传中的条目：置 cancelled，不回 pending，也不重新组批上传', async () => {
    const actions = mountActions()
    const a = enqueue(actions, 'a.txt')
    const b = enqueue(actions, 'b.txt')
    const run = actions.runUpload()
    await vi.waitFor(() => expect(inFlight.length).toBe(2))

    actions.abortUploadItem(a)
    expect(a.status).toBe('cancelled')
    inFlight[1].resolve()
    await run

    // cancelled 终态：不被 catch 送回 pending，也不会被外层 for(;;) 重新组批
    expect(a.status).toBe('cancelled')
    expect(uploadObject).toHaveBeenCalledTimes(2)
    expect(uploadObject).toHaveBeenNthCalledWith(1, a.file, expect.objectContaining({ key: 'a.txt' }), expect.anything(), expect.anything())
    expect(b.status).toBe('done')
  })

  it('取消等待中的条目：跳过上传，不占用 worker', async () => {
    const actions = mountActions()
    enqueue(actions, 'a.txt')
    enqueue(actions, 'b.txt')
    const c = enqueue(actions, 'c.txt')
    const run = actions.runUpload()

    await vi.waitFor(() => expect(inFlight.length).toBe(2))
    actions.abortUploadItem(c)
    expect(c.status).toBe('cancelled')

    inFlight[0].resolve()
    inFlight[1].resolve()
    await run

    // 第三条从未发起上传（组批跳过 cancelled）
    expect(uploadObject).toHaveBeenCalledTimes(2)
    expect(c.status).toBe('cancelled')
  })
})

describe('useObjectActions 账号校验', () => {
  it('requireAccId 无账号时抛错并被调用方转化为错误', async () => {
    const actions = makeActions({
      account: computed(() => undefined),
      selected: ref(new Set(['a.txt'])),
    })
    await actions.removeSelected()
    expect(vi.mocked(s3api.deleteObjects)).not.toHaveBeenCalled()
    expect(lastCtx!.error.value).toBe('no active account')
    expect(vi.mocked(toast)).toHaveBeenCalledWith('objects.toastDeleteFailed', 'err')
  })
})

describe('编辑 HTTP 头 / 标签 / 桶属性 / 版本 / 存储类型 / 生命周期对话框', () => {
  it('openHeadersDialog；onHeadersSaved 同 key 时刷新详情', async () => {
    const actions = makeActions()
    const got = meta('a.txt')
    vi.mocked(s3api.headObject).mockResolvedValue(got)

    actions.openHeadersDialog('a.txt')
    expect(actions.headersKey.value).toBe('a.txt')
    expect(actions.headersOpen.value).toBe(true)

    // 详情正是该对象 → 刷新详情
    actions.detail.value = { ...got }
    actions.onHeadersSaved()
    expect(actions.headersOpen.value).toBe(false)
    await flushPromises()
    expect(vi.mocked(s3api.headObject)).toHaveBeenCalledWith('a1', { bucket: 'b1', key: 'a.txt' })
    expect(actions.detail.value).toEqual(got)
  })

  it('onHeadersSaved 详情为其他对象时不刷新', async () => {
    const actions = makeActions()
    actions.detail.value = meta('other.txt')
    actions.openHeadersDialog('a.txt')
    actions.onHeadersSaved()
    await flushPromises()
    expect(vi.mocked(s3api.headObject)).not.toHaveBeenCalled()
  })

  it('ctxTags/openTagsDialog：仅文件条目打开；文件夹/空关闭菜单', () => {
    const actions = makeActions()
    setEntry({ kind: 'folder', key: 'dir/', name: 'dir' })
    actions.ctxTags()
    expect(actions.tagsOpen.value).toBe(false)
    expect(lastCtx!.closeCtx).toHaveBeenCalled()

    setEntry(null)
    actions.ctxTags()
    expect(actions.tagsOpen.value).toBe(false)

    setEntry({ kind: 'file', key: 'a.txt', name: 'a.txt', object: fileObj('a.txt') })
    actions.ctxTags()
    expect(actions.tagsKey.value).toBe('a.txt')
    expect(actions.tagsOpen.value).toBe(true)

    actions.openTagsDialog('b.txt')
    expect(actions.tagsKey.value).toBe('b.txt')
    expect(actions.tagsOpen.value).toBe(true)
  })

  it('openBucketInfo', () => {
    const actions = makeActions()
    expect(actions.bucketInfoOpen.value).toBe(false)
    actions.openBucketInfo()
    expect(actions.bucketInfoOpen.value).toBe(true)
  })

  it('ctxVersions/openVersions：仅文件条目打开', () => {
    const actions = makeActions()
    setEntry({ kind: 'folder', key: 'dir/', name: 'dir' })
    actions.ctxVersions()
    expect(actions.versionsOpen.value).toBe(false)
    setEntry(null)
    actions.ctxVersions()
    expect(actions.versionsOpen.value).toBe(false)
    setEntry({ kind: 'file', key: 'a.txt', name: 'a.txt', object: fileObj('a.txt') })
    actions.ctxVersions()
    expect(actions.versionsKey.value).toBe('a.txt')
    expect(actions.versionsOpen.value).toBe(true)
    actions.openVersions('b.txt')
    expect(actions.versionsKey.value).toBe('b.txt')
  })

  it('openStorageClass；onStorageClassSaved 同对象刷新详情', async () => {
    const actions = makeActions()
    const got = meta('a.txt')
    vi.mocked(s3api.headObject).mockResolvedValue(got)
    actions.detail.value = { ...got }

    actions.openStorageClass('a.txt')
    expect(actions.storageClassKey.value).toBe('a.txt')
    expect(actions.storageClassOpen.value).toBe(true)

    await actions.onStorageClassSaved()
    expect(actions.storageClassOpen.value).toBe(false)
    expect(lastCtx!.load).toHaveBeenCalledWith(true)
    await flushPromises()
    expect(vi.mocked(s3api.headObject)).toHaveBeenCalledWith('a1', { bucket: 'b1', key: 'a.txt' })
    expect(actions.detail.value).toEqual(got)
  })

  it('onStorageClassSaved 详情为其他对象时只刷新列表', async () => {
    const actions = makeActions()
    actions.detail.value = meta('other.txt')
    actions.openStorageClass('a.txt')
    await actions.onStorageClassSaved()
    expect(lastCtx!.load).toHaveBeenCalledWith(true)
    expect(vi.mocked(s3api.headObject)).not.toHaveBeenCalled()
  })

  it('openLifecycle', () => {
    const actions = makeActions()
    actions.openLifecycle('b1')
    expect(actions.lifecycleBucket.value).toBe('b1')
    expect(actions.lifecycleOpen.value).toBe(true)
  })
})

describe('删除对象', () => {
  it('removeSelected 无选中直接返回', async () => {
    const actions = makeActions()
    await actions.removeSelected()
    expect(vi.mocked(confirmDialog)).not.toHaveBeenCalled()
    expect(vi.mocked(s3api.deleteObjects)).not.toHaveBeenCalled()
  })

  it('removeSelected 确认取消', async () => {
    const actions = makeActions({ selected: ref(new Set(['a.txt'])) })
    vi.mocked(confirmDialog).mockResolvedValueOnce(false)
    await actions.removeSelected()
    expect(vi.mocked(s3api.deleteObjects)).not.toHaveBeenCalled()
  })

  it('removeSelected 成功：toast、清空选中、刷新列表', async () => {
    const actions = makeActions({ selected: ref(new Set(['a.txt', 'b.txt'])) })
    vi.mocked(s3api.deleteObjects).mockResolvedValue({ deleted: 2 })
    await actions.removeSelected()
    expect(vi.mocked(s3api.deleteObjects)).toHaveBeenCalledWith('a1', { bucket: 'b1', keys: ['a.txt', 'b.txt'] })
    expect(vi.mocked(toast)).toHaveBeenCalledWith('objects.toastDeleted')
    expect(lastCtx!.selected.value.size).toBe(0)
    expect(lastCtx!.load).toHaveBeenCalledWith(true)
  })

  it('removeSelected 失败：写错误并 toast err', async () => {
    const actions = makeActions({ selected: ref(new Set(['a.txt'])) })
    vi.mocked(s3api.deleteObjects).mockRejectedValue(new Error('boom'))
    await actions.removeSelected()
    expect(lastCtx!.error.value).toBe('boom')
    expect(vi.mocked(toast)).toHaveBeenCalledWith('objects.toastDeleteFailed', 'err')
  })

  it('removeSelected 超过服务端单次上限（1000）时按 1000 分片串行提交', async () => {
    const keys = Array.from({ length: 2500 }, (_, i) => `f${String(i).padStart(4, '0')}.bin`)
    const actions = makeActions({ selected: ref(new Set(keys)) })
    vi.mocked(s3api.deleteObjects).mockResolvedValue({ deleted: 1000, failed: 0 })
    await actions.removeSelected()
    const calls = vi.mocked(s3api.deleteObjects).mock.calls
    expect(calls).toHaveLength(3)
    // 每片都不超上限，且三片不重不漏（顺序与选中顺序一致）
    for (const [, body] of calls) expect(body.keys.length).toBeLessThanOrEqual(1000)
    expect(calls.flatMap(([, body]) => body.keys)).toEqual(keys)
    // 全部成功 → 走整体成功的提示（分片不改变既有提示语义）
    expect(vi.mocked(toast)).toHaveBeenCalledWith('objects.toastDeleted')
    expect(lastCtx!.error.value).toBe('')
    expect(lastCtx!.selected.value.size).toBe(0)
    expect(lastCtx!.load).toHaveBeenCalledWith(true)
  })

  it('removeSelected 分片中途传输失败：已成功的分片不被吞掉（部分成功 + 失败原因）', async () => {
    // 2500 个 key → 3 片（1000 / 1000 / 500）：第 2 片失败时第 3 片仍应继续
    const keys = Array.from({ length: 2500 }, (_, i) => `f${String(i).padStart(4, '0')}.bin`)
    const actions = makeActions({ selected: ref(new Set(keys)) })
    vi.mocked(s3api.deleteObjects).mockReset()
    vi.mocked(s3api.deleteObjects)
      .mockResolvedValueOnce({ deleted: 998, failed: 2, lastError: 'access denied' })
      .mockRejectedValueOnce(new Error('boom'))
      .mockResolvedValueOnce({ deleted: 1, failed: 0 })
    await actions.removeSelected()
    expect(vi.mocked(s3api.deleteObjects).mock.calls).toHaveLength(3)
    // 已成功 999 个 + 失败 2 个：必须报「部分成功」而不是只报失败
    expect(vi.mocked(toast)).toHaveBeenCalledWith('objects.toastDeletePartial', 'err')
    // 失败原因同时进入面板错误条，便于用户复核
    expect(lastCtx!.error.value).toBe('access denied')
    expect(lastCtx!.selected.value.size).toBe(0)
    expect(lastCtx!.load).toHaveBeenCalledWith(true)
  })

  it('removeSelected 单片即传输失败：报删除失败并刷新列表', async () => {
    const actions = makeActions({ selected: ref(new Set(['a.txt'])) })
    vi.mocked(s3api.deleteObjects).mockRejectedValue(new Error('boom'))
    await actions.removeSelected()
    expect(vi.mocked(toast)).toHaveBeenCalledWith('objects.toastDeleteFailed', 'err')
    expect(lastCtx!.error.value).toBe('boom')
    expect(lastCtx!.selected.value.size).toBe(0)
    expect(lastCtx!.load).toHaveBeenCalledWith(true)
  })

  it('removeSelected 逐 key 被拒但无异常：不误报全部成功', async () => {
    const actions = makeActions({ selected: ref(new Set(['a.txt', 'b.txt'])) })
    vi.mocked(s3api.deleteObjects).mockResolvedValue({ deleted: 1, failed: 1, lastError: 'access denied' })
    await actions.removeSelected()
    expect(vi.mocked(toast)).toHaveBeenCalledWith('objects.toastDeletePartial', 'err')
    expect(vi.mocked(toast)).not.toHaveBeenCalledWith('objects.toastDeleted', expect.anything())
    expect(lastCtx!.error.value).toBe('access denied')
    expect(lastCtx!.load).toHaveBeenCalledWith(true)
  })

  it('removeOne 取消 / 成功 / 失败', async () => {
    const actions = makeActions()
    vi.mocked(confirmDialog).mockResolvedValueOnce(false)
    await actions.removeOne('a.txt')
    expect(vi.mocked(s3api.deleteObjects)).not.toHaveBeenCalled()

    vi.mocked(s3api.deleteObjects).mockResolvedValue({ deleted: 1 })
    await actions.removeOne('a.txt')
    expect(vi.mocked(s3api.deleteObjects)).toHaveBeenCalledWith('a1', { bucket: 'b1', keys: ['a.txt'] })
    expect(vi.mocked(toast)).toHaveBeenCalledWith('objects.toastDeleted')
    expect(lastCtx!.selected.value.size).toBe(0)
    expect(lastCtx!.load).toHaveBeenCalledWith(true)

    vi.mocked(s3api.deleteObjects).mockRejectedValue(new Error('del fail'))
    await actions.removeOne('b.txt')
    expect(lastCtx!.error.value).toBe('del fail')
  })
})

describe('下载', () => {
  it('download 生成代理 URL 并触发点击', () => {
    const actions = makeActions()
    actions.download(fileObj('dir/a.txt'))
    expect(vi.mocked(proxyUrl)).toHaveBeenCalledWith('a1', 'b1', 'download', 'dir/a.txt', '')
    // 无斜杠 key 分支
    actions.download(fileObj('rootfile.txt'))
    expect(vi.mocked(proxyUrl)).toHaveBeenCalledWith('a1', 'b1', 'download', 'rootfile.txt', '')
  })

  it('download key 以斜杠结尾时 basename 为空 → || 回退 object', () => {
    const actions = makeActions()
    const origCreate = document.createElement.bind(document)
    let created: HTMLAnchorElement | undefined
    const spy = vi.spyOn(document, 'createElement').mockImplementation(((tag: string) => {
      const el = origCreate(tag)
      if (tag === 'a') created = el as HTMLAnchorElement
      return el
    }) as unknown as typeof document.createElement)
    try {
      actions.download(fileObj('dir/'))
      expect(created).toBeTruthy()
      // 'dir/'.split('/').pop() === ''：|| 兜底为 'object'
      expect(created!.download).toBe('object')
    } finally {
      spy.mockRestore()
    }
  })

  it('copySignLink 成功：复制 URL 并 toast', async () => {
    const actions = makeActions()
    vi.mocked(s3api.presign).mockResolvedValue({
      method: 'get',
      bucket: 'b1',
      key: 'a.txt',
      url: 'https://s/u1',
      expiresIn: 3600,
    })
    await actions.copySignLink(fileObj('a.txt'))
    expect(vi.mocked(s3api.presign)).toHaveBeenCalledWith('a1', {
      method: 'get',
      key: 'a.txt',
      bucket: 'b1',
      expiresIn: 3600,
    })
    await flushPromises()
    expect(vi.mocked(copyText)).toHaveBeenCalledWith('https://s/u1')
    expect(vi.mocked(toast)).toHaveBeenCalledWith('objects.copySignLinkOk')
  })

  it('copySignLink 失败：写错误并 toast err', async () => {
    const actions = makeActions()
    vi.mocked(s3api.presign).mockRejectedValue(new Error('presign fail'))
    await actions.copySignLink(fileObj('a.txt'))
    await flushPromises()
    expect(lastCtx!.error.value).toBe('presign fail')
    expect(vi.mocked(toast)).toHaveBeenCalledWith('objects.copySignLinkFail', 'err')
  })

  it('copyTextAndToast 默认提示文案', async () => {
    const actions = makeActions()
    actions.copyTextAndToast('hello')
    await flushPromises()
    expect(vi.mocked(copyText)).toHaveBeenCalledWith('hello')
    expect(vi.mocked(toast)).toHaveBeenCalledWith('objects.copiedClipboard')
  })

  it('downloadSelectedZip 无选中直接返回', async () => {
    const actions = makeActions()
    await actions.downloadSelectedZip()
    expect(vi.mocked(s3api.downloadZipToDisk)).not.toHaveBeenCalled()
  })

  it('downloadSelectedZip 成功：toast 并复位 loading', async () => {
    const o = fileObj('a.txt')
    const actions = makeActions({ fileObjects: computed(() => [o]), selected: ref(new Set(['a.txt'])) })
    vi.mocked(s3api.downloadZipToDisk).mockResolvedValue(undefined)
    await actions.downloadSelectedZip()
    expect(vi.mocked(s3api.downloadZipToDisk)).toHaveBeenCalledWith(
      'a1',
      { bucket: 'b1', keys: ['a.txt'] },
      expect.stringMatching(/^objects-\d{4}-\d{2}-\d{2}-\d{2}-\d{2}-\d{2}\.zip$/),
    )
    expect(vi.mocked(toast)).toHaveBeenCalledWith('objects.toastZipped')
    expect(actions.zipLoading.value).toBe(false)
  })

  it('downloadSelectedZip AbortError 静默', async () => {
    const o = fileObj('a.txt')
    const actions = makeActions({ fileObjects: computed(() => [o]), selected: ref(new Set(['a.txt'])) })
    vi.mocked(s3api.downloadZipToDisk).mockRejectedValue(new DOMException('Aborted', 'AbortError'))
    await actions.downloadSelectedZip()
    expect(lastCtx!.error.value).toBe('')
    expect(actions.zipLoading.value).toBe(false)
  })

  it('downloadSelectedZip 其他错误写入 error', async () => {
    const o = fileObj('a.txt')
    const actions = makeActions({ fileObjects: computed(() => [o]), selected: ref(new Set(['a.txt'])) })
    vi.mocked(s3api.downloadZipToDisk).mockRejectedValue(new Error('zip fail'))
    await actions.downloadSelectedZip()
    expect(lastCtx!.error.value).toBe('zip fail')
    expect(actions.zipLoading.value).toBe(false)
  })
})

describe('右键菜单派发', () => {
  it('ctxOpen：文件夹进入 / 文件下载 / 空忽略', () => {
    const actions = makeActions()
    setEntry({ kind: 'folder', key: 'dir/', name: 'dir' })
    actions.ctxOpen()
    expect(lastCtx!.enterPrefix).toHaveBeenCalledWith('dir/')

    setEntry({ kind: 'file', key: 'a.txt', name: 'a.txt', object: fileObj('a.txt') })
    actions.ctxOpen()
    expect(vi.mocked(proxyUrl)).toHaveBeenCalled()

    // 文件但无 object：不下载
    const calls = vi.mocked(proxyUrl).mock.calls.length
    setEntry({ kind: 'file', key: 'b.txt', name: 'b.txt' })
    actions.ctxOpen()
    expect(vi.mocked(proxyUrl).mock.calls.length).toBe(calls)

    setEntry(null)
    actions.ctxOpen()
    expect(vi.mocked(proxyUrl).mock.calls.length).toBe(calls)
  })

  it('ctxCopyKey：复制条目 key；空条目忽略', async () => {
    const actions = makeActions()
    setEntry({ kind: 'file', key: 'a.txt', name: 'a.txt' })
    actions.ctxCopyKey()
    await flushPromises()
    expect(vi.mocked(copyText)).toHaveBeenCalledWith('a.txt')

    setEntry(null)
    actions.ctxCopyKey()
    await flushPromises()
    expect(vi.mocked(copyText).mock.calls.length).toBe(1)
  })

  it('ctxCopyLink：仅文件且有 object 时复制签名链接', () => {
    const actions = makeActions()
    setEntry({ kind: 'folder', key: 'dir/', name: 'dir' })
    actions.ctxCopyLink()
    expect(vi.mocked(s3api.presign)).not.toHaveBeenCalled()

    setEntry({ kind: 'file', key: 'a.txt', name: 'a.txt' })
    actions.ctxCopyLink()
    expect(vi.mocked(s3api.presign)).not.toHaveBeenCalled()

    setEntry({ kind: 'file', key: 'a.txt', name: 'a.txt', object: fileObj('a.txt') })
    actions.ctxCopyLink()
    expect(vi.mocked(s3api.presign)).toHaveBeenCalled()
  })

  it('ctxDelete：仅文件删除', () => {
    const actions = makeActions()
    setEntry({ kind: 'folder', key: 'dir/', name: 'dir' })
    actions.ctxDelete()
    expect(vi.mocked(confirmDialog)).not.toHaveBeenCalled()

    setEntry({ kind: 'file', key: 'a.txt', name: 'a.txt', object: fileObj('a.txt') })
    actions.ctxDelete()
    expect(vi.mocked(confirmDialog)).toHaveBeenCalled()
  })

  it('ctxDetail：有条目则 showDetail', async () => {
    const actions = makeActions()
    setEntry(null)
    actions.ctxDetail()
    expect(vi.mocked(s3api.headObject)).not.toHaveBeenCalled()

    setEntry({ kind: 'file', key: 'a.txt', name: 'a.txt', object: fileObj('a.txt') })
    actions.ctxDetail()
    await flushPromises()
    expect(vi.mocked(s3api.headObject)).toHaveBeenCalledWith('a1', { bucket: 'b1', key: 'a.txt' })
  })
})

describe('重命名 / 详情 / ACL / 新建文件夹', () => {
  it('ctxRename：仅文件条目弹出输入框', () => {
    const actions = makeActions()
    setEntry({ kind: 'folder', key: 'dir/', name: 'dir' })
    actions.ctxRename()
    expect(vi.mocked(promptDialog)).not.toHaveBeenCalled()

    setEntry(null)
    actions.ctxRename()
    expect(vi.mocked(promptDialog)).not.toHaveBeenCalled()

    setEntry({ kind: 'file', key: 'a.txt', name: 'a.txt', object: fileObj('a.txt') })
    actions.ctxRename()
    expect(vi.mocked(promptDialog)).toHaveBeenCalled()
  })

  it('ctxRenameKey：取消 / 同名 / 成功 / 失败 / 校验函数', async () => {
    const actions = makeActions()
    // 取消
    vi.mocked(promptDialog).mockResolvedValueOnce(null)
    await actions.ctxRenameKey('a.txt')
    expect(vi.mocked(s3api.renameObject)).not.toHaveBeenCalled()

    // 同名 → 直接返回
    vi.mocked(promptDialog).mockResolvedValueOnce('a.txt')
    await actions.ctxRenameKey('a.txt')
    expect(vi.mocked(s3api.renameObject)).not.toHaveBeenCalled()

    // 校验函数：空/纯空白报错，非空通过
    const calls = vi.mocked(promptDialog).mock.calls
    const req = calls[calls.length - 1]![0]
    expect(req.validate!('')).toBe('objects.renameEmpty')
    expect(req.validate!('  ')).toBe('objects.renameEmpty')
    expect(req.validate!('b.txt')).toBeNull()

    // 成功（首尾空白被 trim）
    vi.mocked(s3api.renameObject).mockResolvedValue({ renamed: 'b.txt' })
    vi.mocked(promptDialog).mockResolvedValueOnce('  b.txt  ')
    await actions.ctxRenameKey('a.txt')
    expect(vi.mocked(s3api.renameObject)).toHaveBeenCalledWith('a1', {
      bucket: 'b1',
      key: 'a.txt',
      newKey: 'b.txt',
    })
    expect(vi.mocked(toast)).toHaveBeenCalledWith('objects.toastRenamed')
    expect(lastCtx!.load).toHaveBeenCalledWith(true)

    // 失败
    vi.mocked(s3api.renameObject).mockRejectedValue(new Error('rename fail'))
    vi.mocked(promptDialog).mockResolvedValueOnce('c.txt')
    await actions.ctxRenameKey('a.txt')
    expect(lastCtx!.error.value).toBe('rename fail')
  })

  it('showDetail 成功；过期响应被丢弃', async () => {
    const actions = makeActions()
    const gotA = meta('a.txt')
    const gotB = meta('b.txt')
    let resolveA!: (v: ObjectMeta) => void
    vi.mocked(s3api.headObject)
      .mockImplementationOnce(() => new Promise<ObjectMeta>((r) => (resolveA = r)))
      .mockResolvedValueOnce(gotB)

    const runA = actions.showDetail('a.txt') // seq 1
    const runB = actions.showDetail('b.txt') // seq 2
    await runB
    expect(actions.detail.value).toEqual(gotB)
    resolveA(gotA)
    await runA
    // 过期响应被丢弃，不覆盖新详情
    expect(actions.detail.value).toEqual(gotB)
  })

  it('showDetail 失败：当前序号写错误，过期序号忽略', async () => {
    const actions = makeActions()
    let rejectA!: (e: unknown) => void
    vi.mocked(s3api.headObject)
      .mockImplementationOnce(() => new Promise<ObjectMeta>((_, rej) => (rejectA = rej)))
      .mockRejectedValueOnce(new Error('head fail'))

    const runA = actions.showDetail('a.txt') // seq 1
    const runB = actions.showDetail('b.txt') // seq 2
    await runB
    expect(lastCtx!.error.value).toBe('head fail')
    // 过期错误不写入
    lastCtx!.error.value = ''
    rejectA(new Error('stale fail'))
    await runA
    expect(lastCtx!.error.value).toBe('')
    expect(actions.detail.value).toBeNull()
  })

  it('ctxAcl / openAcl：仅文件条目打开', () => {
    const actions = makeActions()
    setEntry({ kind: 'folder', key: 'dir/', name: 'dir' })
    actions.ctxAcl()
    expect(actions.aclOpen.value).toBe(false)
    setEntry(null)
    actions.ctxAcl()
    expect(actions.aclOpen.value).toBe(false)
    setEntry({ kind: 'file', key: 'a.txt', name: 'a.txt', object: fileObj('a.txt') })
    actions.ctxAcl()
    expect(actions.aclKey.value).toBe('a.txt')
    expect(actions.aclOpen.value).toBe(true)
    actions.openAcl('b.txt')
    expect(actions.aclKey.value).toBe('b.txt')
  })

  it('mkdir：取消 / 校验 / 成功 / 失败', async () => {
    const actions = makeActions({ prefix: ref('dir/') })
    vi.mocked(promptDialog).mockResolvedValueOnce(null)
    await actions.mkdir()
    expect(vi.mocked(s3api.mkdirObject)).not.toHaveBeenCalled()

    const calls = vi.mocked(promptDialog).mock.calls
    const req = calls[calls.length - 1]![0]
    expect(req.validate!('')).toBe('objects.mkdirEmpty')
    expect(req.validate!('  ')).toBe('objects.mkdirEmpty')
    expect(req.validate!('new')).toBeNull()

    vi.mocked(s3api.mkdirObject).mockResolvedValue({ created: 'dir/new/', bucket: 'b1' })
    vi.mocked(promptDialog).mockResolvedValueOnce('new/')
    await actions.mkdir()
    expect(vi.mocked(s3api.mkdirObject)).toHaveBeenCalledWith('a1', { bucket: 'b1', key: 'dir/new/' })
    expect(vi.mocked(toast)).toHaveBeenCalledWith('objects.toastMkdir')
    expect(lastCtx!.load).toHaveBeenCalledWith(true)

    vi.mocked(s3api.mkdirObject).mockRejectedValueOnce(new Error('mkdir fail'))
    vi.mocked(promptDialog).mockResolvedValueOnce('x')
    await actions.mkdir()
    expect(lastCtx!.error.value).toBe('mkdir fail')
  })
})

describe('上传入口', () => {
  it('keyForUpload：有/无前缀', () => {
    const a1 = makeActions({ prefix: ref('dir/') })
    expect(a1.keyForUpload('a.txt')).toBe('dir/a.txt')
    const a2 = makeActions({ prefix: ref('') })
    expect(a2.keyForUpload('a.txt')).toBe('a.txt')
  })

  it('pickUploadFiles：上传中不发点击；空闲点击输入框', () => {
    const actions = makeActions()
    const click = vi.fn()
    actions.uploadInput.value = { click } as unknown as HTMLInputElement
    actions.uploading.value = true
    actions.pickUploadFiles()
    expect(click).not.toHaveBeenCalled()
    actions.uploading.value = false
    actions.pickUploadFiles()
    expect(click).toHaveBeenCalled()
  })

  it('onPickUpload：按前缀入队并开始上传', async () => {
    const actions = makeActions({ prefix: ref('dir/') })
    const file = new File(['x'], 'a.txt')
    const input = document.createElement('input')
    Object.defineProperty(input, 'files', { value: [file] })
    const ev = new Event('change')
    Object.defineProperty(ev, 'target', { value: input })

    actions.onPickUpload(ev)
    expect(input.value).toBe('')
    await vi.waitFor(() => expect(inFlight.length).toBe(1))
    expect(actions.uploadQueue.value[0].key).toBe('dir/a.txt')
    expect(actions.uploadQueue.value[0].bucket).toBe('b1')

    inFlight[0].resolve()
    await vi.waitFor(() => expect(vi.mocked(toast)).toHaveBeenCalledWith('objects.toastUploadOkDir'))
    expect(actions.uploadQueue.value.length).toBe(0)
    expect(actions.uploading.value).toBe(false)
    expect(lastCtx!.load).toHaveBeenCalledWith(true)
  })

  it('runUpload：上传中或账号缺失时直接返回', async () => {
    const actions = makeActions()
    actions.uploading.value = true
    await actions.runUpload()
    expect(vi.mocked(uploadObject)).not.toHaveBeenCalled()
    actions.uploading.value = false

    const noAcc = makeActions({ account: computed(() => undefined) })
    await noAcc.runUpload()
    expect(vi.mocked(uploadObject)).not.toHaveBeenCalled()
  })

  it('runUpload：条目未带 bucket 时回落当前桶', async () => {
    const actions = makeActions()
    // 直接入队无 bucket 的条目：target 求值时走 it.bucket || 当前桶
    actions.uploadQueue.value.push({ file: new File(['x'], 'a.txt'), key: 'a.txt', pct: 0, status: 'pending' } as UploadItem)
    const run = actions.runUpload()
    await vi.waitFor(() => expect(inFlight.length).toBe(1))
    inFlight[0].resolve()
    await run
    expect(vi.mocked(toast)).toHaveBeenCalledWith('objects.toastUploadOkDir')
    expect(actions.uploadQueue.value.length).toBe(0)
  })

  it('runUpload：部分失败提示错误 toast', async () => {
    const actions = makeActions()
    vi.mocked(uploadObject).mockImplementation(((file: File) =>
      file.name === 'bad.txt'
        ? Promise.reject(new Error('bad'))
        : Promise.resolve()) as unknown as UploadImpl)
    enqueue(actions, 'good.txt')
    enqueue(actions, 'bad.txt')

    await actions.runUpload()
    expect(vi.mocked(toast)).toHaveBeenCalledWith('upload.toastPartial', 'err')
    expect(actions.uploadQueue.value.length).toBe(0)
    expect(actions.uploading.value).toBe(false)
    expect(lastCtx!.load).toHaveBeenCalledWith(true)
    vi.mocked(uploadObject).mockImplementation(defaultUploadImpl)
  })
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

  it('组件卸载时断开进行中的 SSE 订阅', async () => {
    let onProgress: ((p: Partial<MigrateProgress>) => void) | undefined
    const stop = vi.fn()
    vi.mocked(subscribeMigrateEvents).mockImplementation((_jobId, p) => {
      onProgress = p as (p: Partial<MigrateProgress>) => void
      return stop
    })
    vi.mocked(s3api.deletePrefixAsync).mockResolvedValueOnce({ jobId: 'j5', total: 2 })
    const { actions, unmount } = mountActionsHost()
    setEntry({ kind: 'folder', key: 'dir/', name: 'dir' })

    void actions.ctxDeleteFolder()
    await vi.waitFor(() => expect(onProgress).toBeTruthy())
    expect(stop).not.toHaveBeenCalled()

    unmount()
    expect(stop).toHaveBeenCalledTimes(1)
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
