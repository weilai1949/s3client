import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import ObjectList from './ObjectList.vue'
import type { Entry, SortKey } from '../types'

vi.mock('../i18n', () => ({
  t: (k: string) => k,
  tf: (k: string) => k,
}))

// happy-dom 也提供 ResizeObserver，但统一用可控 stub 保证 clientHeight=0 → viewportH=480
class RO {
  observe() {}
  unobserve() {}
  disconnect() {}
}
vi.stubGlobal('ResizeObserver', RO)

function file(key: string, overrides: Partial<Entry> = {}): Entry {
  return {
    kind: 'file',
    key,
    name: key,
    size: 2048,
    lastModified: '2024-06-01T10:00:00Z',
    object: {
      key,
      size: 2048,
      lastModified: '2024-06-01T10:00:00Z',
      etag: 'e1',
      contentType: 'application/octet-stream',
      isDir: false,
    },
    ...overrides,
  }
}

const folder: Entry = { kind: 'folder', key: 'dir/', name: 'dir' }

const defaults = {
  entries: [] as Entry[],
  bucketView: 'list' as 'list' | 'grid',
  selected: new Set<string>(),
  sortKey: 'name' as SortKey,
  sortDir: 1 as 1 | -1,
  filter: '',
  filterActive: false,
  loading: false,
  totalCount: 0,
  visibleCount: 0,
  nextToken: '',
  isTruncated: false,
  loadingAll: false,
}

function mountList(props: Partial<typeof defaults> = {}) {
  return mount(ObjectList, { props: { ...defaults, ...props } })
}

describe('ObjectList', () => {
  it('shows skeleton while loading with no entries', () => {
    const w = mountList({ loading: true, totalCount: 0 })
    expect(w.find('[aria-busy="true"]').exists()).toBe(true)
    expect(w.findAll('.skel-row')).toHaveLength(6)
  })

  it('renders filtered empty state with filter query', () => {
    const w = mountList({ filter: 'abc', filterActive: true })
    const empty = w.find('.empty')
    expect(empty.exists()).toBe(true)
    expect(empty.text()).toContain('objects.noMatch')
    expect(empty.find('.empty-icon').text()).toBe('🔍')
  })

  it('renders empty dir state when list has no entries', () => {
    const w = mountList({})
    const empty = w.find('.empty')
    expect(empty.exists()).toBe(true)
    expect(empty.text()).toContain('objects.emptyDir')
    expect(empty.find('.empty-icon').text()).toBe('📭')
  })

  it('grid 空目录态（无过滤）：📭 + emptyDir（filterActive=false 分支）', () => {
    const w = mountList({ bucketView: 'grid' })
    const empty = w.find('.empty')
    expect(empty.exists()).toBe(true)
    expect(empty.text()).toContain('objects.emptyDir')
    expect(empty.find('.empty-icon').text()).toBe('📭')
  })

  it('grid meta：size 缺省 → 0 兜底；storageClass 存在 → ` · sc` 追加', () => {
    const w = mountList({
      bucketView: 'grid',
      entries: [
        file('a.bin', {
          size: undefined,
          object: { key: 'a.bin', size: 0, lastModified: '', etag: 'e', contentType: 'x', isDir: false, storageClass: 'STANDARD' } as Entry['object'] & { storageClass: string },
        }),
      ],
    })
    const meta = w.find('.grid-item .gi-meta').text()
    expect(meta).toContain(' · STANDARD')
  })

  it('列表行 size/lastModified 缺失：`?? 0` / `?? \'\'` 兜底渲染', () => {
    const w = mountList({ entries: [file('x.txt', { size: undefined, lastModified: undefined, object: undefined })] })
    const row = w.find('.v-row')
    expect(row.exists()).toBe(true)
    expect(row.findAll('td')[1].text()).toContain('x.txt')
    // 不抛错即可（fmtSize(0)、fmtDate('')）
  })

  it('checkbox click 带 .stop：不冒泡触发 rowClick', async () => {
    const w = mountList({ entries: [file('a.txt')] })
    const cb = w.find('input[type="checkbox"]')
    await cb.trigger('click')
    expect(w.emitted('rowClick')).toBeUndefined()
  })

  it('time 表头点击 → toggleSort(time),sortKey=time 时显示指示器', async () => {
    const w = mountList({ sortKey: 'name', sortDir: 1, entries: [file('a.txt')] })
    await w.findAll('thead th')[3].trigger('click')
    expect(w.emitted('toggleSort')).toEqual([['time']])
    await w.setProps({ sortKey: 'time' })
    await w.vm.$nextTick()
    expect(w.findAll('thead th')[3].find('.sort-ind').text()).toBe('▲')
  })

  it('onListScroll 未绑定 scrollEl：早退不写 scrollTop', async () => {
    const w = mountList({ entries: [file('a.txt')] })
    const vm = w.vm as unknown as { scrollEl: unknown; onListScroll: () => void }
    vm.scrollEl = null
    vm.onListScroll()
    // 不抛错即通过（scrollTop 未更新）
    expect((w.vm as unknown as { scrollTop: number }).scrollTop).toBe(0)
  })

  it('renders grid view with per-type icons and meta, and empty grid state', async () => {
    const entries = [
      file('pic.png'),
      file('movie.mp4', { size: 100 }),
      file('song.mp3', { size: 200 }),
      file('doc.pdf', { size: 300 }),
      file('arch.zip', { size: 400 }),
      file('notes.txt', { size: 500 }),
      file('data.bin', { size: 600 }),
      folder,
    ]
    const w = mountList({ bucketView: 'grid', entries, selected: new Set(['pic.png']), filterActive: true, filter: 'zz' })
    const items = w.findAll('.grid-item')
    expect(items).toHaveLength(8)
    expect(items[0].find('.gi-icon').text()).toBe('🖼️')
    expect(items[1].find('.gi-icon').text()).toBe('🎬')
    expect(items[2].find('.gi-icon').text()).toBe('🎵')
    expect(items[3].find('.gi-icon').text()).toBe('📕')
    expect(items[4].find('.gi-icon').text()).toBe('📦')
    expect(items[5].find('.gi-icon').text()).toBe('📄')
    expect(items[6].find('.gi-icon').text()).toBe('📎')
    expect(items[7].find('.gi-icon').text()).toBe('📁')
    // 文件 meta：size + storageClass；文件夹 meta
    expect(items[0].find('.gi-meta').text()).toContain('2.0 KB')
    expect(items[7].find('.gi-meta').text()).toBe('objects.folder')
    // 选中状态只作用于 file
    expect(items[0].classes()).toContain('selected')
    expect(items[7].classes()).not.toContain('selected')

    // 网格空态（有 filterActive）
    await w.setProps({ entries: [] })
    expect(w.find('.empty').text()).toContain('objects.noMatch')
  })

  it('grid interactions: click/dblclick/contextmenu/enter/space emit events', async () => {
    const w = mountList({ bucketView: 'grid', entries: [file('a.txt', { size: 10 }), folder] })
    const [fi, di] = w.findAll('.grid-item')
    await fi.trigger('click')
    expect(w.emitted('rowClick')?.[0]).toEqual([w.props('entries')[0]])
    await fi.trigger('dblclick')
    expect(w.emitted('rowDbl')?.[0]).toEqual([w.props('entries')[0]])
    await fi.trigger('contextmenu')
    expect(w.emitted('ctx')?.[0]?.[1]).toEqual(w.props('entries')[0])
    expect(w.emitted('ctx')?.[0]?.[0]).toBeInstanceOf(MouseEvent)
    await fi.trigger('keydown.enter')
    expect(w.emitted('rowDbl')).toHaveLength(2)
    await fi.trigger('keydown.space')
    expect(w.emitted('rowClick')).toHaveLength(2)
    // 文件夹同样可点击进入
    await di.trigger('click')
    const rowEvents = w.emitted('rowClick')!
    expect(rowEvents[rowEvents.length - 1]).toEqual([folder])
  })

  it('list view: folder row renders folder-link and emits enterDir; more button emits ctxButton', async () => {
    const w = mountList({ entries: [folder] })
    const row = w.find('tbody tr.v-row')
    expect(row.classes()).toContain('row-folder')
    expect(row.text()).toContain('dir')
    expect(row.text()).toContain('—')
    const link = row.find('button.folder-link')
    await link.trigger('click')
    expect(w.emitted('enterDir')).toEqual([['dir/']])
    // .stop 防止触发 rowClick
    expect(w.emitted('rowClick')).toBeUndefined()
    await row.find('button.more-btn').trigger('click')
    expect(w.emitted('ctxButton')).toHaveLength(1)
    expect(w.emitted('ctxButton')?.[0]?.[1]).toEqual(folder)
  })

  it('list view: file row renders checkbox/storageClass/actions and emits events', async () => {
    const withSc = file('data.txt', { size: 3 })
    withSc.object = { key: 'data.txt', size: 3, lastModified: '2024-06-01', etag: 'e1', contentType: 'text/plain', isDir: false, storageClass: 'STANDARD' }
    const w = mountList({ entries: [withSc], selected: new Set(['data.txt']) })
    const row = w.find('tbody tr.v-row')
    expect(row.classes()).toContain('selected')
    expect(row.text()).toContain('3 B')
    expect(row.find('.sc-badge').text()).toBe('STANDARD')

    await row.trigger('click')
    expect(w.emitted('rowClick')?.[0]).toEqual([withSc])
    await row.trigger('dblclick')
    expect(w.emitted('rowDbl')?.[0]).toEqual([withSc])
    await row.trigger('contextmenu')
    expect(w.emitted('ctx')?.[0]?.[1]).toEqual(withSc)

    // shift + change → checkToggle(key, shift)
    const cb = row.find('input[type="checkbox"]')
    await cb.trigger('mousedown', { shiftKey: true })
    await cb.trigger('change')
    expect(w.emitted('checkToggle')).toEqual([['data.txt', true]])

    // 按钮事件
    const btns = row.findAll('button')
    await btns[0].trigger('click') // download
    expect(w.emitted('download')?.[0]).toEqual([withSc.object])
    await btns[1].trigger('click') // preview
    expect(w.emitted('preview')?.[0]).toEqual([withSc.object])
    await btns[2].trigger('click') // more
    expect(w.emitted('ctxButton')?.[0]?.[1]).toEqual(withSc)
  })

  it('list view: checkbox without shift emits checkToggle(key, false)', async () => {
    const w = mountList({ entries: [file('a.txt', { size: 1 })] })
    const cb = w.find('tbody input[type="checkbox"]')
    await cb.trigger('mousedown', { shiftKey: false })
    await cb.trigger('change')
    expect(w.emitted('checkToggle')).toEqual([['a.txt', false]])
  })

  it('sort headers show aria-sort/indicator and emit toggleSort on click/keyboard', async () => {
    const w = mountList({ sortKey: 'size', sortDir: -1 })
    const ths = w.findAll('thead th')
    expect(ths[1].attributes('aria-sort')).toBe('none')
    expect(ths[2].attributes('aria-sort')).toBe('descending')
    expect(ths[2].find('.sort-ind').text()).toBe('▼')
    await ths[1].trigger('click')
    expect(w.emitted('toggleSort')?.[0]).toEqual(['name'])
    await ths[1].trigger('keydown.enter')
    await ths[1].trigger('keydown.space')
    expect(w.emitted('toggleSort')).toHaveLength(3)
    await ths[2].trigger('click')
    expect(w.emitted('toggleSort')?.[3]).toEqual(['size'])
    await ths[3].trigger('click')
    expect(w.emitted('toggleSort')?.[4]).toEqual(['time'])
    // sortDir=1 → ascending
    await w.setProps({ sortKey: 'name', sortDir: 1 })
    expect(w.findAll('thead th')[1].attributes('aria-sort')).toBe('ascending')
    expect(w.findAll('thead th')[1].find('.sort-ind').text()).toBe('▲')
  })

  it('size/time sort headers respond to keyboard activation (enter/space)', async () => {
    const w = mountList({})
    const ths = w.findAll('thead th')
    await ths[2].trigger('keydown.enter')
    await ths[2].trigger('keydown.space')
    await ths[3].trigger('keydown.enter')
    expect(w.emitted('toggleSort')).toEqual([['size'], ['size'], ['time']])
  })

  it('loadMore/loadAll buttons disabled without nextToken, enabled with; badges reflect truncation', async () => {
    const w = mountList({ nextToken: '', isTruncated: true })
    const btns = w.findAll('.toolbar button')
    expect(btns).toHaveLength(2)
    expect(btns[0].attributes('disabled')).toBeDefined()
    expect(btns[1].attributes('disabled')).toBeDefined()
    expect(w.text()).toContain('objects.hasMore')

    await w.setProps({ nextToken: 'tok1', isTruncated: false })
    const btns2 = w.findAll('.toolbar button')
    expect(btns2[0].attributes('disabled')).toBeUndefined()
    expect(btns2[1].attributes('disabled')).toBeUndefined()
    await btns2[0].trigger('click')
    expect(w.emitted('loadMore')).toBeTruthy()
    await btns2[1].trigger('click')
    expect(w.emitted('loadAll')).toBeTruthy()
    expect(w.text()).toContain('objects.endReached')
  })

  it('loading/loadingAll states change toolbar button labels', async () => {
    const w = mountList({ nextToken: 'tok', loading: true })
    expect(w.findAll('.toolbar button')[0].text()).toBe('common.loading')
    await w.setProps({ loading: false, loadingAll: true })
    expect(w.findAll('.toolbar button')[1].text()).toBe('common.loading')
  })

  it('virtualizes long lists: only window of rows rendered, scroll reveals spacer rows', async () => {
    const many = Array.from({ length: 60 }, (_, i) => file(`f${String(i).padStart(2, '0')}.txt`, { size: i + 1 }))
    const w = mountList({ entries: many })
    // viewportH=480 → count = ceil(480/38)+24 = 37
    expect(w.findAll('tbody tr.v-row')).toHaveLength(37)
    const spacers = () => w.findAll('tbody tr.v-spacer')
    expect(spacers()).toHaveLength(1) // padBottom
    const el = w.find('.tbl-wrap').element as HTMLElement
    el.scrollTop = 38 * 30
    await w.find('.tbl-wrap').trigger('scroll')
    await w.find('.tbl-wrap').trigger('scroll') // 幂等
    expect(spacers().length).toBeGreaterThanOrEqual(1)
    const firstRow = w.findAll('tbody tr.v-row')[0].text()
    expect(firstRow).toContain('f18.txt')
  })

  it('unmount 时安全断开 ResizeObserver（onBeforeUnmount 分支）', () => {
    const w = mountList({ entries: [file('a.txt')] })
    expect(() => w.unmount()).not.toThrow()
  })
})
