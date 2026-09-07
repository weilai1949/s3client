import { describe, expect, it, vi } from 'vitest'
import { useObjectBrowser } from './useObjectBrowser'
import type { Entry } from '../types'
import { s3api } from '../api'

vi.mock('../api', () => ({
  s3api: {
    listBuckets: vi.fn(),
    deleteBucket: vi.fn(),
    listObjects: vi.fn(),
  },
}))

vi.mock('../store', () => ({
  state: { currentAccountId: 'acc-1' },
  currentAccount: vi.fn(() => ({ id: 'acc-1', bucket: 'b1' })),
  toast: vi.fn(),
  selectAccount: vi.fn(),
}))

vi.mock('../i18n', () => ({
  t: (k: string) => k,
  tf: (k: string, _vars: Record<string, unknown>) => k,
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
    browser.openCtx(new MouseEvent('click'), entry as any)
    expect(browser.ctxMenu.value).toEqual({ x: 0, y: 0, entry })
    browser.closeCtx()
    expect(browser.ctxMenu.value).toBeNull()
  })

  it('onKey closes ctx on Escape', () => {
    const browser = useObjectBrowser(makeBindings())
    browser.openCtx(new MouseEvent('click'), { kind: 'file', key: 'a.txt' } as any)
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
      { key: 'a.txt', size: 1, lastModified: '', etag: 'e1', contentType: 'text/plain', isDir: false },
      { key: 'b.txt', size: 2, lastModified: '', etag: 'e2', contentType: 'text/plain', isDir: false },
    ]
    browser.selectAll()
    expect(browser.selected.value.size).toBe(2)
  })

  it('selectAll clears when all selected', () => {
    const browser = useObjectBrowser(makeBindings())
    browser.objects.value = [
      { key: 'a.txt', size: 1, lastModified: '', etag: 'e1', contentType: 'text/plain', isDir: false },
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
      { key: 'a.txt', size: 1, lastModified: '', etag: 'e1', contentType: 'text/plain', isDir: false },
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
      { key: 'a.txt', size: 1, lastModified: '', etag: 'e1', contentType: 'text/plain', isDir: false },
      { key: 'b.txt', size: 2, lastModified: '', etag: 'e2', contentType: 'text/plain', isDir: false },
      { key: 'c.txt', size: 3, lastModified: '', etag: 'e3', contentType: 'text/plain', isDir: false },
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
    await browser.removeBucket({ name: 'old-bucket' } as any)
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
