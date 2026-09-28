// useObjectBrowser.test.ts —— 对象浏览器主流程：初始化、排序、选择、快捷键、导航、加载与删除。
// KNOWN_ISSUES #60 拆分后的分支范围见 useObjectBrowser.branches.test.ts（断言逐字搬移）。
import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { defineComponent } from 'vue'
import { useObjectBrowser } from './useObjectBrowser'
import type { BucketItem, Entry, ObjectItem } from '../types'
import { s3api } from '../api'
import { selectAccount, toast } from '../store'

/** 被测 API 的真实返回类型：用于给 mock 数据做类型断言（保持运行时数据不变）。 */
type ListBucketsResult = Awaited<ReturnType<typeof s3api.listBuckets>>
type ListObjectsResult = Awaited<ReturnType<typeof s3api.listObjects>>

vi.mock('../api', () => ({
  s3api: {
    listBuckets: vi.fn(),
    deleteBucket: vi.fn(),
    listObjects: vi.fn(),
  },
}))

vi.mock('../store', async () => {
  const { reactive } = await import('vue')
  return {
    state: reactive({ currentAccountId: 'acc-1' }),
    currentAccount: vi.fn(() => ({ id: 'acc-1', bucket: 'b1' })),
    toast: vi.fn(),
    selectAccount: vi.fn(),
  }
})

vi.mock('../i18n', () => ({
  t: (k: string) => k,
  tf: vi.fn((k: string) => k),
}))

vi.mock('../confirm', () => ({
  confirmDialog: vi.fn(async () => true),
}))

// Mock localStorage
const localStorageMock = {
  getItem: vi.fn(() => null),
  setItem: vi.fn(),
  removeItem: vi.fn(),
}
Object.defineProperty(window, 'localStorage', { value: localStorageMock })
Object.defineProperty(globalThis, 'localStorage', { value: localStorageMock, configurable: true })

describe('useObjectBrowser', () => {
  function makeBindings() {
    return {
      previewOrDownload: vi.fn(),
      ctxRenameKey: vi.fn(),
      removeSelected: vi.fn(),
    }
  }

  it('initializes with empty state', () => {
    const browser = useObjectBrowser(makeBindings())
    expect(browser.prefix.value).toBe('')
    expect(browser.currentBucket.value).toBe('')
    expect(browser.objects.value).toEqual([])
    expect(browser.selected.value).toEqual(new Set())
  })

  it('toggleSort toggles direction when same key, resets when different', () => {
    const browser = useObjectBrowser(makeBindings())
    browser.sortKey.value = 'name'
    browser.sortDir.value = 1
    browser.toggleSort('name')
    expect(browser.sortDir.value).toBe(-1)
    browser.toggleSort('size')
    expect(browser.sortKey.value).toBe('size')
    expect(browser.sortDir.value).toBe(1)
  })

  it('openCtx / closeCtx sets ctxMenu', () => {
    const browser = useObjectBrowser(makeBindings())
    const entry = { kind: 'file', key: 'a.txt' }
    browser.openCtx(new MouseEvent('click'), entry as unknown as Entry)
    expect(browser.ctxMenu.value).toEqual({ x: 0, y: 0, entry })
    browser.closeCtx()
    expect(browser.ctxMenu.value).toBeNull()
  })

  it('onKey closes ctx on Escape', () => {
    const browser = useObjectBrowser(makeBindings())
    browser.openCtx(new MouseEvent('click'), { kind: 'file', key: 'a.txt' } as unknown as Entry)
    expect(browser.ctxMenu.value).not.toBeNull()
    browser.onKey(new KeyboardEvent('keydown', { key: 'Escape' }))
    expect(browser.ctxMenu.value).toBeNull()
  })

  it('toggle adds/removes key from selected', () => {
    const browser = useObjectBrowser(makeBindings())
    browser.toggle('a.txt')
    expect(browser.selected.value.has('a.txt')).toBe(true)
    browser.toggle('a.txt')
    expect(browser.selected.value.has('a.txt')).toBe(false)
  })

  it('selectAll selects all when not all selected', () => {
    const browser = useObjectBrowser(makeBindings())
    browser.objects.value = [
      { key: 'a.txt', size: 1, lastModified: '', etag: 'e1', isDir: false },
      { key: 'b.txt', size: 2, lastModified: '', etag: 'e2', isDir: false },
    ]
    browser.selectAll()
    expect(browser.selected.value.size).toBe(2)
  })

  it('selectAll clears when all selected', () => {
    const browser = useObjectBrowser(makeBindings())
    browser.objects.value = [
      { key: 'a.txt', size: 1, lastModified: '', etag: 'e1', isDir: false },
    ]
    browser.selected.value = new Set(['a.txt'])
    browser.selectAll()
    expect(browser.selected.value.size).toBe(0)
  })

  it('onRowClick enters prefix for folder, toggles for file', () => {
    const browser = useObjectBrowser(makeBindings())
    browser.currentBucket.value = 'b1'
    browser.onRowClick({ kind: 'folder', key: 'dir/' } as Entry)
    expect(browser.prefix.value).toBe('dir/')
    browser.onRowClick({ kind: 'file', key: 'a.txt' } as Entry)
    expect(browser.selected.value.has('a.txt')).toBe(true)
  })

  it('onGlobalKey handles Enter, F2, Delete, Ctrl+A', () => {
    const bindings = makeBindings()
    const browser = useObjectBrowser(bindings)
    browser.panelActive.value = true
    browser.currentBucket.value = 'b1'
    browser.objects.value = [
      { key: 'a.txt', size: 1, lastModified: '', etag: 'e1', isDir: false },
    ]
    browser.selected.value = new Set(['a.txt'])

    // Enter
    browser.onGlobalKey(new KeyboardEvent('keydown', { key: 'Enter' }))
    expect(bindings.previewOrDownload).toHaveBeenCalled()

    // F2
    browser.onGlobalKey(new KeyboardEvent('keydown', { key: 'F2' }))
    expect(bindings.ctxRenameKey).toHaveBeenCalled()

    // Delete
    browser.onGlobalKey(new KeyboardEvent('keydown', { key: 'Delete' }))
    expect(bindings.removeSelected).toHaveBeenCalled()

    // Ctrl+A：选中项已是全部 → 反选清空
    browser.onGlobalKey(new KeyboardEvent('keydown', { key: 'a', ctrlKey: true }))
    expect(browser.selected.value.size).toBe(0)
    // 再次 Ctrl+A → 全选恢复
    browser.onGlobalKey(new KeyboardEvent('keydown', { key: 'a', ctrlKey: true }))
    expect(browser.selected.value.size).toBe(1)
  })

  it('onGlobalKey skips when panel not active', () => {
    const bindings = makeBindings()
    const browser = useObjectBrowser(bindings)
    browser.panelActive.value = false
    browser.onGlobalKey(new KeyboardEvent('keydown', { key: 'Enter' }))
    expect(bindings.previewOrDownload).not.toHaveBeenCalled()
  })

  it('onGlobalKey skips when input focused', () => {
    const bindings = makeBindings()
    const browser = useObjectBrowser(bindings)
    browser.panelActive.value = true
    const ev = new KeyboardEvent('keydown', { key: 'Enter' })
    Object.defineProperty(ev, 'target', { value: document.createElement('input') })
    browser.onGlobalKey(ev)
    expect(bindings.previewOrDownload).not.toHaveBeenCalled()
  })

  it('bindings 缺省（面板初始化期间）：快捷键与双击不抛错', () => {
    // 不传 bindings：命中默认参数与可选链的 undefined 分支。
    const browser = useObjectBrowser()
    browser.panelActive.value = true
    browser.currentBucket.value = 'b1'
    browser.objects.value = [
      { key: 'a.txt', size: 1, lastModified: '', etag: 'e1', isDir: false },
    ]
    browser.selected.value = new Set(['a.txt'])

    expect(() => browser.onGlobalKey(new KeyboardEvent('keydown', { key: 'Enter' }))).not.toThrow()
    expect(() => browser.onGlobalKey(new KeyboardEvent('keydown', { key: 'F2' }))).not.toThrow()
    expect(() => browser.onGlobalKey(new KeyboardEvent('keydown', { key: 'Delete' }))).not.toThrow()
    expect(() =>
      browser.onRowDblClick({
        kind: 'file',
        key: 'a.txt',
        name: 'a.txt',
        object: { key: 'a.txt', size: 1, lastModified: '', etag: 'e1', isDir: false },
      } as Entry),
    ).not.toThrow()
  })

  it('enterBucket / goRoot / goUp', () => {
    const browser = useObjectBrowser(makeBindings())
    browser.enterBucket('my-bucket')
    expect(browser.currentBucket.value).toBe('my-bucket')
    expect(browser.prefix.value).toBe('')

    browser.prefix.value = 'a/b/'
    browser.goUp()
    expect(browser.prefix.value).toBe('a/')

    browser.goRoot()
    expect(browser.prefix.value).toBe('')
  })

  it('onBucketChange / enterPrefix / onBucketSelect', () => {
    const browser = useObjectBrowser(makeBindings())
    browser.currentBucket.value = 'b1'
    browser.onBucketChange()
    expect(browser.prefix.value).toBe('')

    browser.enterPrefix('dir/')
    expect(browser.prefix.value).toBe('dir/')
    expect(vi.mocked(s3api.listObjects)).toHaveBeenCalled()

    browser.onBucketSelect('b2')
    expect(browser.currentBucket.value).toBe('b2')
    expect(browser.prefix.value).toBe('')
  })

  it('backToBuckets clears bucket and loads', () => {
    const browser = useObjectBrowser(makeBindings())
    browser.backToBuckets()
    expect(browser.currentBucket.value).toBe('')
  })

  it('toggleView toggles bucketView', () => {
    const browser = useObjectBrowser(makeBindings())
    expect(browser.bucketView.value).toBe('list')
    browser.toggleView()
    expect(browser.bucketView.value).toBe('grid')
    browser.toggleView()
    expect(browser.bucketView.value).toBe('list')
  })

  it('toggleWithShift selects range', () => {
    const browser = useObjectBrowser(makeBindings())
    browser.objects.value = [
      { key: 'a.txt', size: 1, lastModified: '', etag: 'e1', isDir: false },
      { key: 'b.txt', size: 2, lastModified: '', etag: 'e2', isDir: false },
      { key: 'c.txt', size: 3, lastModified: '', etag: 'e3', isDir: false },
    ]
    browser.toggleWithShift('a.txt', false)
    expect(browser.selected.value.has('a.txt')).toBe(true)
    browser.toggleWithShift('c.txt', true)
    expect(browser.selected.value.has('b.txt')).toBe(true)
    expect(browser.selected.value.has('c.txt')).toBe(true)
  })

  it('onRowDblClick enters prefix for folder, downloads for file', () => {
    const bindings = makeBindings()
    const browser = useObjectBrowser(bindings)
    browser.currentBucket.value = 'b1'
    browser.onRowDblClick({ kind: 'folder', key: 'dir/' } as Entry)
    expect(browser.prefix.value).toBe('dir/')
    browser.onRowDblClick({ kind: 'file', key: 'a.txt', object: {} } as Entry)
    expect(bindings.previewOrDownload).toHaveBeenCalled()
  })

  it('startPathEdit / commitPath / cancelPathEdit / togglePathEdit', () => {
    const browser = useObjectBrowser(makeBindings())
    browser.prefix.value = 'dir/'
    browser.startPathEdit()
    expect(browser.pathEditing.value).toBe(true)
    expect(browser.pathDraft.value).toBe('dir/')

    browser.pathDraft.value = 'newdir/'
    browser.commitPath()
    expect(browser.pathEditing.value).toBe(false)
    expect(browser.prefix.value).toBe('newdir/')

    browser.startPathEdit()
    browser.cancelPathEdit()
    expect(browser.pathEditing.value).toBe(false)

    browser.startPathEdit()
    browser.togglePathEdit()
    expect(browser.pathEditing.value).toBe(false) // committed
  })

  it('openCreateBucket / onCreateBucket / removeBucket', async () => {
    const browser = useObjectBrowser(makeBindings())
    browser.openCreateBucket()
    expect(browser.creatingBucket.value).toBe(true)

    await browser.onCreateBucket({ name: 'new-bucket', region: 'us', acl: 'private' })
    expect(browser.creatingBucket.value).toBe(false)
    expect(browser.currentBucket.value).toBe('new-bucket')

    // removeBucket with confirm ok: 当前桶等于被删桶时应重置
    browser.currentBucket.value = 'old-bucket'
    await browser.removeBucket({ name: 'old-bucket' } as unknown as BucketItem)
    expect(browser.currentBucket.value).toBe('')
  })

  it('cancelPathEdit resets pathEditing', () => {
    const browser = useObjectBrowser(makeBindings())
    browser.pathEditing.value = true
    browser.cancelPathEdit()
    expect(browser.pathEditing.value).toBe(false)
  })

  it('loadMore calls load(false)', () => {
    const browser = useObjectBrowser(makeBindings())
    browser.currentBucket.value = 'b1'
    browser.loadMore()
    expect(vi.mocked(s3api.listObjects)).toHaveBeenCalled()
  })
})

describe('useObjectBrowser gaps', () => {
  function makeBindings() {
    return { previewOrDownload: vi.fn(), ctxRenameKey: vi.fn(), removeSelected: vi.fn() }
  }

  it('loadBuckets resets stale currentBucket and auto-enters default bucket', async () => {
    vi.mocked(s3api.listBuckets).mockResolvedValue({ buckets: [{ name: 'b9' }] } as unknown as ListBucketsResult)
    const browser = useObjectBrowser(makeBindings())
    browser.buckets.value = [{ name: 'b9', createdAt: '' }] as unknown as BucketItem[]
    browser.currentBucket.value = 'stale'
    await browser.loadBuckets()
    // stale(currentBucket) 失效被重置，随即自动进入 acc.bucket（line 140/143 同轮触发）
    expect(browser.currentBucket.value).toBe('b1')
  })

  it('loadBuckets surfaces error', async () => {
    vi.mocked(s3api.listBuckets).mockRejectedValue(new Error('list failed'))
    const browser = useObjectBrowser(makeBindings())
    await browser.loadBuckets()
    expect(browser.error.value).toBe('list failed')
  })

  it('load reads nextToken/isTruncated and clears selected on reset', async () => {
    vi.mocked(s3api.listObjects).mockResolvedValue({
      objects: [{ key: 'a.txt' }], commonPrefixes: [], nextToken: 'tok1', isTruncated: true,
    } as unknown as ListObjectsResult)
    const browser = useObjectBrowser(makeBindings())
    browser.currentBucket.value = 'b1'
    browser.selected.value = new Set(['old'])
    await browser.load(true)
    expect(browser.nextToken.value).toBe('tok1')
    expect(browser.isTruncated.value).toBe(true)
    expect(browser.selected.value.size).toBe(0)
  })

  it('loadMore appends objects without clearing selection', async () => {
    vi.mocked(s3api.listObjects).mockResolvedValue({
      objects: [{ key: 'b.txt' }], commonPrefixes: [], nextToken: '', isTruncated: false,
    } as unknown as ListObjectsResult)
    const browser = useObjectBrowser(makeBindings())
    browser.currentBucket.value = 'b1'
    browser.objects.value = [{ key: 'a.txt', size: 1, lastModified: '', isDir: false }] as unknown as ObjectItem[]
    browser.selected.value = new Set(['a.txt'])
    await browser.loadMore()
    expect(browser.objects.value.map((o) => o.key)).toEqual(['a.txt', 'b.txt'])
    expect(browser.selected.value.has('a.txt')).toBe(true)
  })

  it('refreshAll reloads buckets then objects', async () => {
    vi.mocked(s3api.listBuckets).mockResolvedValue({ buckets: [{ name: 'b1' }] } as unknown as ListBucketsResult)
    vi.mocked(s3api.listObjects).mockResolvedValue({ objects: [], commonPrefixes: [], nextToken: '', isTruncated: false } as unknown as ListObjectsResult)
    const browser = useObjectBrowser(makeBindings())
    browser.currentBucket.value = 'b1'
    await browser.refreshAll()
    expect(s3api.listBuckets).toHaveBeenCalled()
    expect(s3api.listObjects).toHaveBeenCalled()
  })

  it('hideHints writes localStorage and clears error on load', async () => {
    const browser = useObjectBrowser(makeBindings())
    browser.hideHints()
    expect(localStorageMock.setItem).toHaveBeenCalledWith('s3c.hintsHidden', '1')
    expect(browser.hintsHidden.value).toBe(true)
  })

  it('存储不可用时提示条读写降级（不抛错，默认未隐藏）', () => {
    const boom = () => { throw new DOMException('SecurityError', 'SecurityError') }
    const throwing = { getItem: boom, setItem: boom, removeItem: boom }
    Object.defineProperty(globalThis, 'localStorage', { value: throwing, configurable: true })
    Object.defineProperty(window, 'localStorage', { value: throwing, configurable: true })
    try {
      const browser = useObjectBrowser(makeBindings())
      expect(browser.hintsHidden.value).toBe(false)
      expect(() => browser.hideHints()).not.toThrow()
      expect(browser.hintsHidden.value).toBe(true)
    } finally {
      Object.defineProperty(globalThis, 'localStorage', { value: localStorageMock, configurable: true })
      Object.defineProperty(window, 'localStorage', { value: localStorageMock, configurable: true })
    }
  })

  it('openCtxFromButton anchors below the button', () => {
    const browser = useObjectBrowser(makeBindings())
    const fakeTarget = { getBoundingClientRect: () => ({ right: 200, bottom: 100, left: 0, top: 0, width: 0, height: 0 }) }
    const ev = { currentTarget: fakeTarget } as unknown as MouseEvent
    browser.openCtxFromButton(ev, { kind: 'file', key: 'a.txt' } as Entry)
    expect(browser.ctxMenu.value).not.toBeNull()
    expect(browser.ctxMenu.value?.entry.key).toBe('a.txt')
  })

  it('sort comparator handles name/size/lastModified + dirs first', () => {
    const browser = useObjectBrowser(makeBindings())
    browser.commonPrefixes.value = ['dir/']
    browser.objects.value = [
      { key: 'zz.txt', size: 10, lastModified: '2024-02-01', isDir: false },
      { key: 'aa.txt', size: 2, lastModified: '2024-01-01', isDir: false },
    ] as unknown as ObjectItem[]
    // default sort name asc
    let entries = browser.visibleEntries.value
    expect(entries[0].key).toBe('dir/') // dirs first
    expect(entries[1].key).toBe('aa.txt')
    expect(entries[2].key).toBe('zz.txt')
    browser.toggleSort('size')
    entries = browser.visibleEntries.value
    expect(entries[1].key).toBe('aa.txt') // size asc after dirs
    browser.toggleSort('time')
    entries = browser.visibleEntries.value
    expect(entries[1].key).toBe('aa.txt')
  })

  it('account switch watch clears prefix and reloads', async () => {
    vi.mocked(s3api.listBuckets).mockResolvedValue({ buckets: [{ name: 'b1' }] } as unknown as ListBucketsResult)
    vi.mocked(s3api.listObjects).mockResolvedValue({ objects: [], commonPrefixes: [], nextToken: '', isTruncated: false } as unknown as ListObjectsResult)
    let browser!: ReturnType<typeof useObjectBrowser>
    const Host = defineComponent({
      setup() {
        browser = useObjectBrowser(makeBindings())
        return () => null
      },
    })
    const wrapper = mount(Host)
    await vi.waitFor(() => expect(browser.panelActive.value).toBe(true))
    browser.prefix.value = 'x/'
    const { state } = await import('../store')
    state.currentAccountId = 'other'
    await vi.waitFor(() => expect(browser.prefix.value).toBe(''))
    wrapper.unmount()
  })

  it('mounted host registers window listeners and unmount cleans up', async () => {
    vi.mocked(s3api.listBuckets).mockResolvedValue({ buckets: [{ name: 'b1' }] } as unknown as ListBucketsResult)
    vi.mocked(s3api.listObjects).mockResolvedValue({ objects: [], commonPrefixes: [], nextToken: '', isTruncated: false } as unknown as ListObjectsResult)
    let browser!: ReturnType<typeof useObjectBrowser>
    const Host = defineComponent({
      setup() {
        browser = useObjectBrowser(makeBindings())
        return () => null
      },
    })
    const wrapper = mount(Host)
    await vi.waitFor(() => expect(browser.panelActive.value).toBe(true))
    wrapper.unmount()
    expect(browser.error.value).toBe('')
  })
})

describe('useObjectBrowser remaining', () => {
  function makeBindings() {
    return { previewOrDownload: vi.fn(), ctxRenameKey: vi.fn(), removeSelected: vi.fn() }
  }

  it('removeBucket surfaces deleteBucket error', async () => {
    vi.mocked(s3api.deleteBucket).mockRejectedValue(new Error('delete failed'))
    const browser = useObjectBrowser(makeBindings())
    browser.currentBucket.value = 'b1'
    await browser.removeBucket({ name: 'b1' } as unknown as BucketItem)
    expect(browser.error.value).toBe('delete failed')
  })

  it('accSel getter 读取当前账号 id，setter 委托 selectAccount', async () => {
    const browser = useObjectBrowser(makeBindings())
    // 前序测试可能把 mocked store 的 currentAccountId 改成 'other'，先归位
    const { state } = await import('../store')
    state.currentAccountId = 'acc-1'
    expect(browser.accSel.value).toBe('acc-1') // 触发 getter（line 326）
    browser.accSel.value = 'other-acc' // 触发 setter（line 327）
    expect(selectAccount).toHaveBeenCalledWith('other-acc')
  })

  it('crumbs builds breadcrumb paths', () => {
    const browser = useObjectBrowser(makeBindings())
    browser.prefix.value = 'a/b/'
    expect(browser.crumbs.value).toEqual([
      { name: 'a', path: 'a/' },
      { name: 'b', path: 'a/b/' },
    ])
  })

  it('loadAll paginates until exhausted and toasts', async () => {
    vi.mocked(s3api.listObjects).mockResolvedValue({
      objects: [{ key: 'x.txt', size: 1, lastModified: '', isDir: false }],
      commonPrefixes: [], nextToken: 'tok', isTruncated: true,
    } as unknown as ListObjectsResult)
    const browser = useObjectBrowser(makeBindings())
    browser.currentBucket.value = 'b1'
    browser.nextToken.value = 'tok1'
    await browser.loadAll()
    expect(toast).toHaveBeenCalled()
  })

  it('loadAll 在 nextToken 为空时也至少加载第一页（不得静默只加载 0 条并误报「已加载全部」）', async () => {
    // 单页场景：首轮请求返回无 nextToken，随后不应再发分页请求。
    vi.mocked(s3api.listObjects).mockResolvedValue({
      objects: [{ key: 'only.txt', size: 1, lastModified: '', isDir: false }],
      commonPrefixes: [], nextToken: '', isTruncated: false,
    } as unknown as ListObjectsResult)
    const browser = useObjectBrowser(makeBindings())
    browser.currentBucket.value = 'b1'
    // 模拟「刚进目录、尚未加载过任何一页」：nextToken 为空，objects 也为空。
    browser.objects.value = []
    browser.nextToken.value = ''
    await browser.loadAll()
    // 必须真正发起过列表请求并拿到数据
    expect(s3api.listObjects).toHaveBeenCalled()
    expect(browser.objects.value.length).toBe(1)
    expect(browser.objects.value[0].key).toBe('only.txt')
    expect(browser.loadingAll.value).toBe(false)
  })

  it('loadAll 在 nextToken 为空但已有旧列表时重置为第一页（不重复追加导致重复项）', async () => {
    vi.mocked(s3api.listObjects).mockResolvedValue({
      objects: [{ key: 'p1.txt', size: 1, lastModified: '', isDir: false }],
      commonPrefixes: [], nextToken: '', isTruncated: false,
    } as unknown as ListObjectsResult)
    const browser = useObjectBrowser(makeBindings())
    browser.currentBucket.value = 'b1'
    // 旧列表已有数据但 nextToken 为空（例如此前加载失败残留）
    browser.objects.value = [{ key: 'stale.txt', size: 9, lastModified: '', isDir: false }] as unknown as ObjectItem[]
    browser.nextToken.value = ''
    await browser.loadAll()
    // 应重置为第一页，而不是在旧列表上追加（stale 被替换，且无重复 p1）
    const keys = browser.objects.value.map((o) => o.key)
    expect(keys).toEqual(['p1.txt'])
  })

  it('togglePathEdit commits when editing', () => {
    const browser = useObjectBrowser(makeBindings())
    browser.pathEditing.value = true
    browser.prefix.value = 'old/'
    browser.pathDraft.value = 'new/'
    browser.togglePathEdit()
    expect(browser.pathEditing.value).toBe(false)
    expect(browser.prefix.value).toBe('new/')
  })
})

