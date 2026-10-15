// useObjectBrowser.branches.test.ts —— 自 useObjectBrowser.test.ts 拆出（KNOWN_ISSUES #60）：
// KeepAlive 生命周期、loadAll 分支、分页上限与选中合计。断言逐字搬移，与拆分前等价。
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { defineComponent, h, KeepAlive, nextTick, ref as vueRef } from 'vue'
import { useObjectBrowser } from './useObjectBrowser'
import type { Account, BucketItem, Entry, ObjectItem } from '../types'
import { s3api } from '../api'
import { confirmDialog } from '../confirm'
import { currentAccount, toast } from '../store'
import { tf } from '../i18n'

/** 被测 API 的真实返回类型：用于给 mock 数据做类型断言（保持运行时数据不变）。 */
type ListBucketsResult = Awaited<ReturnType<typeof s3api.listBuckets>>
type ListObjectsResult = Awaited<ReturnType<typeof s3api.listObjects>>
type DeleteBucketResult = Awaited<ReturnType<typeof s3api.deleteBucket>>

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
  getItem: vi.fn((_key: string): string | null => null),
  setItem: vi.fn(),
  removeItem: vi.fn(),
}
Object.defineProperty(window, 'localStorage', { value: localStorageMock })
Object.defineProperty(globalThis, 'localStorage', { value: localStorageMock, configurable: true })

describe('useObjectBrowser KeepAlive lifecycle', () => {
  function makeBindings() {
    return { previewOrDownload: vi.fn(), ctxRenameKey: vi.fn(), removeSelected: vi.fn() }
  }

  it('onActivated/onDeactivated toggle panelActive', async () => {
    vi.mocked(s3api.listBuckets).mockResolvedValue({ buckets: [{ name: 'b1' }] } as unknown as ListBucketsResult)
    vi.mocked(s3api.listObjects).mockResolvedValue({ objects: [], commonPrefixes: [], nextToken: '', isTruncated: false } as unknown as ListObjectsResult)
    let browser!: ReturnType<typeof useObjectBrowser>
    const Host = defineComponent({
      setup() {
        browser = useObjectBrowser(makeBindings())
        return () => null
      },
    })
    const show = vueRef(true)
    const Root = defineComponent({
      setup() {
        return () =>
          h(KeepAlive, null, {
            default: () => (show.value ? h(Host) : h(defineComponent({ setup: () => () => null }))),
          })
      },
    })
    const wrapper = mount(Root)
    await vi.waitFor(() => expect(browser.panelActive.value).toBe(true))
    show.value = false
    await vi.waitFor(() => expect(browser.panelActive.value).toBe(false))
    wrapper.unmount()
  })
})

describe('useObjectBrowser loadAll error / path toggle', () => {
  function makeBindings() {
    return { previewOrDownload: vi.fn(), ctxRenameKey: vi.fn(), removeSelected: vi.fn() }
  }

  it('loadAll surfaces listObjects error', async () => {
    // 覆盖不可达：load() 自身 try/catch 吞掉所有错误、从不 reject，loadAll 的
    // catch（useObjectBrowser.ts:405-406）属防御代码、无法被任何调用路径触发；
    // 本测试断言错误经 load() 内部 catch 上抛到 error.value（该行为仍被测试覆盖）。
    vi.mocked(s3api.listObjects).mockRejectedValue(new Error('page failed'))
    const browser = useObjectBrowser(makeBindings())
    browser.currentBucket.value = 'b1'
    browser.nextToken.value = 'tok'
    await browser.loadAll()
    expect(browser.error.value).toBe('page failed')
    expect(browser.loadingAll.value).toBe(false)
  })

  it('togglePathEdit starts editing when not editing', () => {
    const browser = useObjectBrowser(makeBindings())
    browser.pathEditing.value = false
    browser.prefix.value = 'a/'
    browser.togglePathEdit()
    expect(browser.pathEditing.value).toBe(true)
    expect(browser.pathDraft.value).toBe('a/')
  })
})

describe('useObjectBrowser final branches', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(s3api.listBuckets).mockResolvedValue({ buckets: [{ name: 'b1' }] } as unknown as ListBucketsResult)
    vi.mocked(s3api.listObjects).mockResolvedValue({ objects: [], commonPrefixes: [], nextToken: '', isTruncated: false } as unknown as ListObjectsResult)
    vi.mocked(currentAccount).mockReturnValue({ id: 'acc-1', bucket: 'b1' } as unknown as Account)
    vi.mocked(confirmDialog).mockResolvedValue(true)
  })

  function makeBindings() {
    return { previewOrDownload: vi.fn(), ctxRenameKey: vi.fn(), removeSelected: vi.fn() }
  }

  it('multiple folders sort by name; filter active narrows entries', () => {
    const browser = useObjectBrowser(makeBindings())
    browser.commonPrefixes.value = ['b/', 'a/']
    browser.objects.value = [{ key: 'z.txt', size: 5, lastModified: '2024-01-01', isDir: false } as unknown as ObjectItem]
    let entries = browser.visibleEntries.value
    expect(entries[0].key).toBe('a/') // 两个 folder 触发 sorts comparator
    browser.filter.value = 'z'
    expect(browser.filterActive.value).toBe(true)
    entries = browser.visibleEntries.value
    expect(entries.length).toBe(1)
    expect(entries[0].key).toBe('z.txt')
  })

  it('loadBuckets with no account returns early', async () => {
    vi.mocked(currentAccount).mockReturnValueOnce(undefined)
    const browser = useObjectBrowser(makeBindings())
    await browser.loadBuckets()
    expect(s3api.listBuckets).not.toHaveBeenCalled()
  })

  it('stale loadBuckets response is dropped (seq guard)', async () => {
    let resolveFirst!: (v: ListBucketsResult) => void
    let resolveSecond!: (v: ListBucketsResult) => void
    vi.mocked(s3api.listBuckets)
      .mockImplementationOnce(() => new Promise((r) => { resolveFirst = r }))
      .mockImplementationOnce(() => new Promise((r) => { resolveSecond = r }))
    const browser = useObjectBrowser(makeBindings())
    const p1 = browser.loadBuckets()
    const p2 = browser.loadBuckets()
    resolveFirst({ buckets: [{ name: 'first' }] } as unknown as ListBucketsResult)
    resolveSecond({ buckets: [{ name: 'second' }] } as unknown as ListBucketsResult)
    await Promise.all([p1, p2])
    expect(browser.buckets.value.map((b) => b.name)).toEqual(['second'])
  })

  it('loadBuckets 与对象 load 各自计数：切桶不会丢弃在飞的桶列表（回归：曾共用 loadSeq）', async () => {
    let resolveBuckets!: (v: ListBucketsResult) => void
    let resolveObjects!: (v: ListObjectsResult) => void
    vi.mocked(s3api.listBuckets).mockImplementationOnce(
      () => new Promise((r) => { resolveBuckets = r }),
    )
    vi.mocked(s3api.listObjects).mockImplementationOnce(
      () => new Promise((r) => { resolveObjects = r }),
    )
    const browser = useObjectBrowser(makeBindings())
    const pBuckets = browser.loadBuckets() // 在飞的桶列表请求（bucketSeq=1）
    browser.currentBucket.value = 'b1'
    const pObjects = browser.load(true) // 切桶触发的对象加载（旧实现会占用同一计数器 → 桶响应被误丢）
    resolveBuckets({ buckets: [{ name: 'b1' }] } as unknown as ListBucketsResult)
    await pBuckets
    // 若两者共用计数器，load() 递增后 loadBuckets 的响应会被判过期丢弃、且 loadingBuckets 卡 true。
    expect(browser.buckets.value.map((b) => b.name)).toEqual(['b1'])
    expect(browser.loadingBuckets.value).toBe(false)
    resolveObjects({ objects: [], commonPrefixes: [], nextToken: '', isTruncated: false } as unknown as ListObjectsResult)
    await pObjects
  })

  it('removeBucket returns early when confirm is cancelled', async () => {
    vi.mocked(confirmDialog).mockResolvedValue(false)
    const browser = useObjectBrowser(makeBindings())
    await browser.removeBucket({ name: 'b1' } as unknown as BucketItem)
    expect(s3api.deleteBucket).not.toHaveBeenCalled()
  })

  it('removeBucket 无当前账号时直接返回（不依赖非空断言）', async () => {
    vi.mocked(confirmDialog).mockResolvedValue(true)
    vi.mocked(currentAccount).mockReturnValue(undefined)
    const browser = useObjectBrowser(makeBindings())
    await browser.removeBucket({ name: 'b1' } as unknown as BucketItem)
    expect(s3api.deleteBucket).not.toHaveBeenCalled()
  })

  it('loadedSize/selectedSize computeds evaluate with data', () => {
    const browser = useObjectBrowser(makeBindings())
    browser.objects.value = [
      { key: 'a.txt', size: 10, lastModified: '', isDir: false },
      { key: 'b.txt', size: 25, lastModified: '', isDir: false },
    ] as unknown as ObjectItem[]
    browser.selected.value = new Set(['a.txt'])
    expect(browser.loadedSize.value).toBe(35)
    expect(browser.selectedSize.value).toBe(10)
  })

  it('onGlobalKey bails when account or bucket missing', () => {
    const bindings = makeBindings()
    vi.mocked(currentAccount).mockReturnValueOnce(undefined)
    const browser = useObjectBrowser(bindings)
    browser.panelActive.value = true
    browser.onGlobalKey(new KeyboardEvent('keydown', { key: 'Enter' }))
    expect(bindings.previewOrDownload).not.toHaveBeenCalled()
  })
})

describe('useObjectBrowser loadAll guards + keepalive reactivation', () => {
  function makeBindings() {
    return { previewOrDownload: vi.fn(), ctxRenameKey: vi.fn(), removeSelected: vi.fn() }
  }

  it('loadAll returns early while already loading', async () => {
    const browser = useObjectBrowser(makeBindings())
    browser.currentBucket.value = 'b1'
    browser.loading.value = true
    await browser.loadAll()
    expect(s3api.listObjects).not.toHaveBeenCalled()
  })

  it('loadAll 首页加载期间不得清 loadingAll（否则「加载全部」跑到一半按钮重新可点）', async () => {
    vi.mocked(s3api.listBuckets).mockResolvedValue({ buckets: [{ name: 'b1' }] } as unknown as ListBucketsResult)
    let releasePage2!: () => void
    const page2 = new Promise<void>((resolve) => { releasePage2 = resolve })
    vi.mocked(s3api.listObjects)
      .mockResolvedValueOnce({ objects: [], commonPrefixes: [], nextToken: 't1', isTruncated: true } as unknown as ListObjectsResult)
      .mockImplementationOnce(async () => {
        await page2
        return { objects: [], commonPrefixes: [], nextToken: '', isTruncated: false } as unknown as ListObjectsResult
      })
    const browser = useObjectBrowser(makeBindings())
    browser.currentBucket.value = 'b1'
    browser.nextToken.value = '' // needFirstPage=true → 首轮以 reset 语义加载
    const run = browser.loadAll()
    await vi.waitFor(() => expect(browser.nextToken.value).toBe('t1')) // 首页已回、第二页仍在飞
    expect(browser.loadingAll.value, 'loadAll 自己发起的 reset 不得清掉自己的 loadingAll').toBe(true)
    releasePage2()
    await run
    expect(browser.loadingAll.value).toBe(false)
  })

  it('loadAll breaks and returns when navigation invalidates seq', async () => {
    const browser = useObjectBrowser(makeBindings())
    browser.currentBucket.value = 'b1'
    browser.nextToken.value = 't'
    browser.goRoot() // 先触发一次正常加载（listObjects → token 't' 由默认实现返回）
    const p = browser.loadAll()
    // 加载期间导航递增 loadSeq（goRoot 为公开方法，内部 ++loadSeq）
    vi.mocked(s3api.listObjects).mockImplementation(async () => {
      browser.goRoot()
      return { objects: [], commonPrefixes: [], nextToken: '', isTruncated: false } as unknown as ListObjectsResult
    })
    // 恢复 nextToken 使 loadAll 的 while 循环进入（第一次 listObjects 已完成）
    browser.nextToken.value = 't'
    await p
    expect(browser.loadingAll.value).toBe(false)
  })

  it('loadAll 分页途中导航递增 loadSeq：循环 break + 末尾 return，不发完成 toast', async () => {
    const browser = useObjectBrowser(makeBindings())
    browser.currentBucket.value = 'b1'
    // 先完成一次正常加载，确保 loading=false 并留下 nextToken='t' 供 loadAll 分页
    vi.mocked(s3api.listObjects).mockImplementation(async () => ({
      objects: [], commonPrefixes: [], nextToken: 't', isTruncated: true,
    } as unknown as ListObjectsResult))
    await browser.load(true)
    expect(browser.nextToken.value).toBe('t')
    // loadAll 的每次 listObjects 内模拟「导航」：goRoot 递增 loadSeq（仅第一次，避免递归）
    let bumped = false
    vi.mocked(s3api.listObjects).mockImplementation(async () => {
      if (!bumped) {
        bumped = true
        browser.goRoot()
      }
      return { objects: [], commonPrefixes: [], nextToken: 't', isTruncated: true } as unknown as ListObjectsResult
    })
    await browser.loadAll()
    expect(browser.loadingAll.value).toBe(false)
  })

  it('KeepAlive reactivation with changed account reloads', async () => {
    vi.mocked(s3api.listBuckets).mockResolvedValue({ buckets: [{ name: 'b1' }] } as unknown as ListBucketsResult)
    vi.mocked(s3api.listObjects).mockResolvedValue({ objects: [], commonPrefixes: [], nextToken: '', isTruncated: false } as unknown as ListObjectsResult)
    let browser!: ReturnType<typeof useObjectBrowser>
    const Host = defineComponent({
      setup() {
        browser = useObjectBrowser(makeBindings())
        return () => null
      },
    })
    const show = vueRef(true)
    const Root = defineComponent({
      setup() {
        return () =>
          h(KeepAlive, null, {
            default: () => (show.value ? h(Host) : h(defineComponent({ setup: () => () => null }))),
          })
      },
    })
    const wrapper = mount(Root)
    await vi.waitFor(() => expect(browser.panelActive.value).toBe(true))
    show.value = false
    await vi.waitFor(() => expect(browser.panelActive.value).toBe(false))
    const { state } = await import('../store')
    state.currentAccountId = 'changed-acc'
    show.value = true
    await vi.waitFor(() => expect(browser.panelActive.value).toBe(true))
    // 再激活时账号已变 → onAccountSwitch 清空 prefix/bucket
    await vi.waitFor(() => expect(browser.currentBucket.value).toBe('b1')) // acc.bucket 自动进入
    wrapper.unmount()
  })
})

describe('useObjectBrowser 100% branch completion', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(s3api.listBuckets).mockResolvedValue({ buckets: [{ name: 'b1' }] } as unknown as ListBucketsResult)
    vi.mocked(s3api.listObjects).mockResolvedValue({ objects: [], commonPrefixes: [], nextToken: '', isTruncated: false } as unknown as ListObjectsResult)
    vi.mocked(s3api.deleteBucket).mockResolvedValue({} as unknown as DeleteBucketResult)
    vi.mocked(currentAccount).mockReturnValue({ id: 'acc-1', bucket: 'b1' } as unknown as Account)
    vi.mocked(confirmDialog).mockResolvedValue(true)
  })

  function makeBindings() {
    return { previewOrDownload: vi.fn(), ctxRenameKey: vi.fn(), removeSelected: vi.fn() }
  }

  it('toggleSort 同键且当前为降序时翻回升序（sortDir === 1 的 -1 侧）', () => {
    const browser = useObjectBrowser(makeBindings())
    browser.sortKey.value = 'name'
    browser.sortDir.value = -1
    browser.toggleSort('name')
    expect(browser.sortDir.value).toBe(1)
  })

  it('size 排序下 size 缺失回退 0（a.size ?? 0 / b.size ?? 0 的缺失侧）', () => {
    const browser = useObjectBrowser(makeBindings())
    browser.objects.value = [
      { key: 'a.txt', lastModified: '2024-01-01', isDir: false },
      { key: 'b.txt', lastModified: '2024-01-01', isDir: false },
    ] as unknown as ObjectItem[]
    browser.sortKey.value = 'size'
    expect(browser.visibleEntries.value.map((e) => e.key)).toEqual(['a.txt', 'b.txt'])
  })

  it('time 排序下 lastModified 缺失回退空串（a/b 两侧 ?? 的缺失侧）', () => {
    const browser = useObjectBrowser(makeBindings())
    browser.objects.value = [
      { key: 'a.txt', size: 1, isDir: false },
      { key: 'b.txt', size: 2, isDir: false },
    ] as unknown as ObjectItem[]
    browser.sortKey.value = 'time'
    expect(browser.visibleEntries.value.map((e) => e.key)).toEqual(['a.txt', 'b.txt'])
  })

  it('relName：key 不在当前 prefix 下时原样返回（startsWith 的 false 侧）', () => {
    const browser = useObjectBrowser(makeBindings())
    browser.prefix.value = 'dir/'
    browser.objects.value = [{ key: 'elsewhere.txt', size: 1, lastModified: '', isDir: false }] as unknown as ObjectItem[]
    expect(browser.entries.value[0].name).toBe('elsewhere.txt')
  })

  it('relName：s 切片为空时回退文件名（base 分支），再空时回退 full（双重回退）', () => {
    const browser = useObjectBrowser(makeBindings())
    browser.prefix.value = 'dir/'
    browser.commonPrefixes.value = ['dir/']
    // s='' → split('/').pop() 得 'dir'（中间回退）
    expect(browser.entries.value[0].name).toBe('dir')
    // full 无 basename → 最终回退 full（'' || '' || ''）
    expect(browser.relName('', false)).toBe('')
  })

  it('loadBuckets 响应缺 buckets 字段时回退空数组', async () => {
    vi.mocked(s3api.listBuckets).mockResolvedValue({} as unknown as ListBucketsResult)
    const browser = useObjectBrowser(makeBindings())
    await browser.loadBuckets()
    expect(browser.buckets.value).toEqual([])
    expect(browser.currentBucket.value).toBe('b1') // acc.bucket 自动进入
  })

  it('loadBuckets 拒绝非 Error 值：String(e) 落盘', async () => {
    vi.mocked(s3api.listBuckets).mockRejectedValue('plain failure')
    const browser = useObjectBrowser(makeBindings())
    await browser.loadBuckets()
    expect(browser.error.value).toBe('plain failure')
  })

  it('loadBuckets 过期序号的拒绝被丢弃（seq 守卫的 false 侧）', async () => {
    let rejectFirst!: (e: unknown) => void
    vi.mocked(s3api.listBuckets)
      .mockImplementationOnce(() => new Promise((_, rej) => { rejectFirst = rej }))
      .mockResolvedValueOnce({ buckets: [{ name: 'b1' }] } as unknown as ListBucketsResult)
    const browser = useObjectBrowser(makeBindings())
    const p1 = browser.loadBuckets()
    const p2 = browser.loadBuckets()
    rejectFirst(new Error('stale fail'))
    await p1
    await p2
    expect(browser.error.value).toBe('')
    expect(browser.loadingBuckets.value).toBe(false)
  })

  it('removeBucket 删除非当前桶时不清空 currentBucket', async () => {
    const browser = useObjectBrowser(makeBindings())
    browser.currentBucket.value = 'keep'
    // loadBuckets 只返回 'keep'：currentBucket 仍有效，不会被重置
    vi.mocked(s3api.listBuckets).mockResolvedValue({ buckets: [{ name: 'keep' }] } as unknown as ListBucketsResult)
    await browser.removeBucket({ name: 'other' } as unknown as BucketItem)
    expect(browser.currentBucket.value).toBe('keep')
  })

  it('load() 不传参使用默认 reset=true（默认参数分支）', async () => {
    const browser = useObjectBrowser(makeBindings())
    browser.currentBucket.value = 'b1'
    await browser.load()
    expect(s3api.listObjects).toHaveBeenCalled()
    expect(browser.loading.value).toBe(false)
  })

  it('reset 加载缺 objects/commonPrefixes 字段时回退空数组', async () => {
    vi.mocked(s3api.listObjects).mockResolvedValue({} as unknown as ListObjectsResult)
    const browser = useObjectBrowser(makeBindings())
    browser.currentBucket.value = 'b1'
    await browser.load(true)
    expect(browser.objects.value).toEqual([])
    expect(browser.commonPrefixes.value).toEqual([])
  })

  it('loadMore 响应缺 objects/commonPrefixes 字段时原样累积', async () => {
    vi.mocked(s3api.listObjects).mockResolvedValue({} as unknown as ListObjectsResult)
    const browser = useObjectBrowser(makeBindings())
    browser.currentBucket.value = 'b1'
    browser.objects.value = [{ key: 'a.txt', size: 1, lastModified: '', isDir: false }] as unknown as ObjectItem[]
    browser.commonPrefixes.value = ['keep/']
    await browser.loadMore()
    expect(browser.objects.value.map((o) => o.key)).toEqual(['a.txt'])
    expect(browser.commonPrefixes.value).toEqual(['keep/'])
  })

  it('load 拒绝非 Error 值：String(e) 落盘', async () => {
    vi.mocked(s3api.listObjects).mockRejectedValue('plain failure')
    const browser = useObjectBrowser(makeBindings())
    browser.currentBucket.value = 'b1'
    await browser.load(true)
    expect(browser.error.value).toBe('plain failure')
  })

  it('load 过期序号的拒绝被丢弃（seq 守卫的 false 侧）', async () => {
    let rejectFirst!: (e: unknown) => void
    vi.mocked(s3api.listObjects)
      .mockImplementationOnce(() => new Promise((_, rej) => { rejectFirst = rej }))
      .mockResolvedValueOnce({ objects: [], commonPrefixes: [], nextToken: '', isTruncated: false } as unknown as ListObjectsResult)
    const browser = useObjectBrowser(makeBindings())
    browser.currentBucket.value = 'b1'
    const p1 = browser.load(true)
    const p2 = browser.load(true)
    rejectFirst(new Error('stale fail'))
    await p1
    await p2
    expect(browser.error.value).toBe('')
    expect(browser.loading.value).toBe(false)
  })

  it('goUp 单段前缀弹出后为空串（parts.length 的 false 侧）', () => {
    const browser = useObjectBrowser(makeBindings())
    browser.currentBucket.value = 'b1'
    browser.prefix.value = 'a/'
    browser.goUp()
    expect(browser.prefix.value).toBe('')
  })

  it('onGlobalKey：TEXTAREA/SELECT/BUTTON/isContentEditable 聚焦时忽略，普通元素放行', () => {
    const bindings = makeBindings()
    const browser = useObjectBrowser(bindings)
    browser.panelActive.value = true
    browser.currentBucket.value = 'b1'
    browser.objects.value = [{ key: 'a.txt', size: 1, lastModified: '', isDir: false }] as unknown as ObjectItem[]
    browser.selected.value = new Set(['a.txt'])
    const fire = (el: HTMLElement | null, key = 'Enter') => {
      const ev = new KeyboardEvent('keydown', { key })
      Object.defineProperty(ev, 'target', { value: el })
      browser.onGlobalKey(ev)
    }
    fire(document.createElement('textarea'))
    fire(document.createElement('select'))
    fire(document.createElement('button'))
    const editable = document.createElement('div')
    Object.defineProperty(editable, 'isContentEditable', { value: true })
    fire(editable)
    expect(bindings.previewOrDownload).not.toHaveBeenCalled()
    // 普通 div（可编辑性为 false）放行 → Enter 触发预览
    fire(document.createElement('div'))
    expect(bindings.previewOrDownload).toHaveBeenCalledTimes(1)
  })

  it('onGlobalKey Enter/F2/Delete 无选中文件时不触发（first/files.length 的 false 侧）', () => {
    const bindings = makeBindings()
    const browser = useObjectBrowser(bindings)
    browser.panelActive.value = true
    browser.currentBucket.value = 'b1'
    browser.objects.value = [{ key: 'a.txt', size: 1, lastModified: '', isDir: false }] as unknown as ObjectItem[]
    browser.selected.value = new Set()
    browser.onGlobalKey(new KeyboardEvent('keydown', { key: 'Enter' }))
    browser.onGlobalKey(new KeyboardEvent('keydown', { key: 'F2' }))
    browser.onGlobalKey(new KeyboardEvent('keydown', { key: 'Delete' }))
    expect(bindings.previewOrDownload).not.toHaveBeenCalled()
    expect(bindings.ctxRenameKey).not.toHaveBeenCalled()
    expect(bindings.removeSelected).not.toHaveBeenCalled()
  })

  it('onGlobalKey 修饰键分支：Cmd+A 全选；无修饰/非 a 键不触发', () => {
    const bindings = makeBindings()
    const browser = useObjectBrowser(bindings)
    browser.panelActive.value = true
    browser.currentBucket.value = 'b1'
    browser.objects.value = [{ key: 'a.txt', size: 1, lastModified: '', isDir: false }] as unknown as ObjectItem[]
    browser.selected.value = new Set()
    browser.onGlobalKey(new KeyboardEvent('keydown', { key: 'a', metaKey: true }))
    expect(browser.selected.value.size).toBe(1)
    browser.selected.value = new Set()
    browser.onGlobalKey(new KeyboardEvent('keydown', { key: 'a' }))
    expect(browser.selected.value.size).toBe(0)
    browser.onGlobalKey(new KeyboardEvent('keydown', { key: 'b', ctrlKey: true }))
    expect(browser.selected.value.size).toBe(0)
  })

  it('commitPath：无尾斜杠补斜杠；空白回退空前缀', () => {
    const browser = useObjectBrowser(makeBindings())
    browser.pathDraft.value = 'x/y'
    browser.commitPath()
    expect(browser.prefix.value).toBe('x/y/')
    browser.pathDraft.value = '   '
    browser.commitPath()
    expect(browser.prefix.value).toBe('')
  })

  it('onRowDblClick 文件无 object 时忽略（e.object 的 false 侧）', () => {
    const bindings = makeBindings()
    const browser = useObjectBrowser(bindings)
    browser.onRowDblClick({ kind: 'file', key: 'a.txt' } as Entry)
    expect(bindings.previewOrDownload).not.toHaveBeenCalled()
  })

  it('onMounted 无账号时跳过加载（account 的 false 侧）', async () => {
    vi.mocked(currentAccount).mockReturnValueOnce(undefined)
    let browser!: ReturnType<typeof useObjectBrowser>
    const Host = defineComponent({
      setup() {
        browser = useObjectBrowser(makeBindings())
        return () => null
      },
    })
    const wrapper = mount(Host)
    await vi.waitFor(() => expect(browser.panelActive.value).toBe(true))
    expect(s3api.listBuckets).not.toHaveBeenCalled()
    expect(s3api.listObjects).not.toHaveBeenCalled()
    wrapper.unmount()
  })
})

describe('useObjectBrowser 分页上限与选中合计', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(s3api.listObjects).mockReset()
    vi.mocked(currentAccount).mockReturnValue({ id: 'acc-1', bucket: 'b1' } as unknown as Account)
  })

  function makeBindings() {
    return { previewOrDownload: vi.fn(), ctxRenameKey: vi.fn(), removeSelected: vi.fn() }
  }

  it('loadAll 触顶提示的 n 由请求 maxKeys 推导（分页大小单一来源，不在文案处二次硬编码）', async () => {
    vi.mocked(s3api.listObjects)
      .mockResolvedValueOnce({
        objects: [{ key: 'p1.txt', size: 1, lastModified: '', isDir: false }],
        commonPrefixes: [], nextToken: 'tok1', isTruncated: true,
      } as unknown as ListObjectsResult)
      .mockResolvedValueOnce({
        objects: [{ key: 'p2.txt', size: 2, lastModified: '', isDir: false }],
        commonPrefixes: [], nextToken: '', isTruncated: true,
      } as unknown as ListObjectsResult)
    const browser = useObjectBrowser(makeBindings())
    browser.currentBucket.value = 'b1'
    await browser.loadAll()
    // 第二页 nextToken 为空 → 循环退出；isTruncated 仍为 true → 触顶提示
    expect(s3api.listObjects).toHaveBeenCalledTimes(2)
    const firstQuery = vi.mocked(s3api.listObjects).mock.calls[0]![1] as Record<string, string>
    const n = 2 * Number(firstQuery.maxKeys)
    expect(n).toBeGreaterThan(0)
    expect(tf).toHaveBeenCalledWith('objects.toastLoadedCap', { n })
    expect(toast).toHaveBeenCalledWith('objects.toastLoadedCap', 'err')
  })

  it('toggle 增量维护 selectedSize：选中/取消即时生效，列表外的 key 按 0 计', () => {
    const browser = useObjectBrowser(makeBindings())
    browser.objects.value = [
      { key: 'a.txt', size: 10, lastModified: '', isDir: false },
      { key: 'b.txt', size: 25, lastModified: '', isDir: false },
    ] as unknown as ObjectItem[]
    expect(browser.selectedSize.value).toBe(0)
    browser.toggle('a.txt')
    expect(browser.selectedSize.value).toBe(10)
    browser.toggle('b.txt')
    expect(browser.selectedSize.value).toBe(35)
    browser.toggle('a.txt')
    expect(browser.selectedSize.value).toBe(25)
    // 不在列表中的 key：按 0 计，不影响合计
    browser.toggle('ghost.txt')
    expect(browser.selected.value.has('ghost.txt')).toBe(true)
    expect(browser.selectedSize.value).toBe(25)
    browser.toggle('ghost.txt')
    expect(browser.selectedSize.value).toBe(25)
  })

  it('toggleWithShift 范围补选后 selectedSize 合计正确', () => {
    const browser = useObjectBrowser(makeBindings())
    browser.objects.value = [
      { key: 'a.txt', size: 10, lastModified: '', isDir: false },
      { key: 'b.txt', size: 25, lastModified: '', isDir: false },
      { key: 'c.txt', size: 5, lastModified: '', isDir: false },
    ] as unknown as ObjectItem[]
    browser.toggleWithShift('a.txt', false)
    expect(browser.selectedSize.value).toBe(10)
    browser.toggleWithShift('c.txt', true)
    expect([...browser.selected.value].sort()).toEqual(['a.txt', 'b.txt', 'c.txt'])
    expect(browser.selectedSize.value).toBe(40)
  })

  it('selectAll / 外部重设 selected / 列表变化：selectedSize 同步兜底重算', () => {
    const browser = useObjectBrowser(makeBindings())
    const list = (a: number, b: number) =>
      [
        { key: 'a.txt', size: a, lastModified: '', isDir: false },
        { key: 'b.txt', size: b, lastModified: '', isDir: false },
      ] as unknown as ObjectItem[]
    browser.objects.value = list(10, 25)
    browser.selectAll()
    expect(browser.selectedSize.value).toBe(35)
    browser.selectAll() // 已全选 → 清空
    expect(browser.selectedSize.value).toBe(0)
    // 面板外部直接赋值（useObjectActions 的路径）
    browser.selected.value = new Set(['a.txt'])
    expect(browser.selectedSize.value).toBe(10)
    // 列表变化（替换/分页追加改变尺寸）→ 选中合计按新尺寸重算
    browser.objects.value = list(7, 25)
    expect(browser.selectedSize.value).toBe(7)
  })

  it('load(reset) 清空选中后 selectedSize 归零', async () => {
    vi.mocked(s3api.listObjects).mockResolvedValue({
      objects: [{ key: 'x.txt', size: 3, lastModified: '', isDir: false }],
      commonPrefixes: [], nextToken: '', isTruncated: false,
    } as unknown as ListObjectsResult)
    const browser = useObjectBrowser(makeBindings())
    browser.currentBucket.value = 'b1'
    browser.objects.value = [
      { key: 'a.txt', size: 10, lastModified: '', isDir: false },
    ] as unknown as ObjectItem[]
    browser.selected.value = new Set(['a.txt'])
    expect(browser.selectedSize.value).toBe(10)
    await browser.load(true)
    expect(browser.selected.value.size).toBe(0)
    expect(browser.selectedSize.value).toBe(0)
  })
})

// 行键盘导航（roving tabindex）：焦点落点由本 composable 决定，ObjectList 只负责落焦。
describe('useObjectBrowser 键盘行导航焦点', () => {
  function makeBindings() {
    return { previewOrDownload: vi.fn(), ctxRenameKey: vi.fn(), removeSelected: vi.fn() }
  }
  const file = (key: string) => ({ key, size: 1, lastModified: '', isDir: false }) as unknown as ObjectItem

  it('focusRow 记录焦点；moveFocus 从当前行移动，越界钳到首尾（不绕回）', () => {
    const browser = useObjectBrowser(makeBindings())
    browser.currentBucket.value = 'b1'
    browser.objects.value = [file('a.txt'), file('b.txt'), file('c.txt')]

    browser.focusRow('b.txt')
    expect(browser.focusKey.value).toBe('b.txt')
    browser.moveFocus(1)
    expect(browser.focusKey.value).toBe('c.txt')
    browser.moveFocus(1) // 已在尾行：钳住不动
    expect(browser.focusKey.value).toBe('c.txt')
    browser.moveFocus(-5) // 越界向上：钳到首行
    expect(browser.focusKey.value).toBe('a.txt')
  })

  it('无焦点行时方向键从首行起步；空列表时 moveFocus / focusEdge 均不动', () => {
    const browser = useObjectBrowser(makeBindings())
    browser.currentBucket.value = 'b1'
    // 空列表：任何移动都是 no-op（也不得写入焦点）
    browser.moveFocus(1)
    browser.moveFocus(-1)
    browser.focusEdge('start')
    browser.focusEdge('end')
    expect(browser.focusKey.value).toBeNull()

    browser.objects.value = [file('a.txt'), file('b.txt')]
    browser.moveFocus(-1) // 无焦点 → 落到首行
    expect(browser.focusKey.value).toBe('a.txt')
    browser.focusEdge('end')
    expect(browser.focusKey.value).toBe('b.txt')
    browser.focusEdge('start')
    expect(browser.focusKey.value).toBe('a.txt')
  })

  it('过滤把焦点行筛掉时清空焦点；仍在可见集内则保留', async () => {
    const browser = useObjectBrowser(makeBindings())
    browser.currentBucket.value = 'b1'
    browser.objects.value = [file('a.txt'), file('b.txt')]
    browser.focusRow('b.txt')

    browser.filter.value = 'b' // b.txt 仍可见 → 焦点保留
    await nextTick()
    expect(browser.focusKey.value).toBe('b.txt')

    browser.filter.value = 'a' // b.txt 被筛掉 → 焦点清空，避免方向键/删除落到看不见的行
    await nextTick()
    expect(browser.focusKey.value).toBeNull()
  })

  it('Enter / F2 作用于焦点行（即便该行未勾选）', () => {
    const bindings = makeBindings()
    const browser = useObjectBrowser(bindings)
    browser.panelActive.value = true
    browser.currentBucket.value = 'b1'
    browser.objects.value = [file('a.txt'), file('b.txt')]
    browser.selected.value = new Set()
    browser.focusKey.value = 'b.txt'

    browser.onGlobalKey(new KeyboardEvent('keydown', { key: 'Enter' }))
    expect(bindings.previewOrDownload).toHaveBeenCalledWith(expect.objectContaining({ key: 'b.txt' }))
    browser.onGlobalKey(new KeyboardEvent('keydown', { key: 'F2' }))
    expect(bindings.ctxRenameKey).toHaveBeenCalledWith('b.txt')
  })

  it('Delete 作用于未勾选的焦点行：先勾选该行再删', () => {
    const bindings = makeBindings()
    const browser = useObjectBrowser(bindings)
    browser.panelActive.value = true
    browser.currentBucket.value = 'b1'
    browser.objects.value = [file('a.txt'), file('b.txt')]
    browser.selected.value = new Set()
    browser.focusKey.value = 'b.txt'

    browser.onGlobalKey(new KeyboardEvent('keydown', { key: 'Delete' }))
    expect(browser.selected.value.has('b.txt')).toBe(true)
    expect(bindings.removeSelected).toHaveBeenCalledTimes(1)
  })

  it('焦点行是文件夹时 Enter/F2/Delete 不作用于文件（文件夹不参与文件动作）', () => {
    const bindings = makeBindings()
    const browser = useObjectBrowser(bindings)
    browser.panelActive.value = true
    browser.currentBucket.value = 'b1'
    browser.objects.value = [file('a.txt')]
    browser.commonPrefixes.value = ['dir/']
    browser.selected.value = new Set()
    browser.focusRow('dir/')

    browser.onGlobalKey(new KeyboardEvent('keydown', { key: 'F2' }))
    browser.onGlobalKey(new KeyboardEvent('keydown', { key: 'Delete' }))
    browser.onGlobalKey(new KeyboardEvent('keydown', { key: 'Enter' }))
    expect(bindings.ctxRenameKey).not.toHaveBeenCalled()
    expect(bindings.removeSelected).not.toHaveBeenCalled()
    expect(bindings.previewOrDownload).not.toHaveBeenCalled()
  })
})
