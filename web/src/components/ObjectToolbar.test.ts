import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import ObjectToolbar from './ObjectToolbar.vue'

vi.mock('../i18n', () => ({
  t: (k: string) => k,
  tf: (k: string) => k,
}))

const baseProps: {
  bucket: string
  buckets: { name: string; creationDate: string }[]
  loadingBuckets: boolean
  loading: boolean
  prefix: string
  crumbs: { name: string; path: string }[]
  pathEditing: boolean
  filterActive: boolean
  visibleCount: number
  totalCount: number
  allSelected: boolean
  selectedCount: number
  selectedSize: number
  fileCount: number
  loadedSize: number
  bucketView: 'list' | 'grid'
  uploading: boolean
  opsBusy: boolean
  zipLoading: boolean
} = {
  bucket: 'my-bucket',
  buckets: [
    { name: 'my-bucket', creationDate: '2024-01-01' },
    { name: 'other-bucket', creationDate: '2024-02-01' },
  ],
  loadingBuckets: false,
  loading: false,
  prefix: 'dir/',
  crumbs: [{ name: 'dir', path: 'dir/' }],
  pathEditing: false,
  filterActive: false,
  visibleCount: 3,
  totalCount: 4,
  allSelected: false,
  selectedCount: 0,
  selectedSize: 0,
  fileCount: 3,
  loadedSize: 1024,
  bucketView: 'list',
  uploading: false,
  opsBusy: false,
  zipLoading: false,
}

function findButton(w: ReturnType<typeof mount>, text: string) {
  const btn = w.findAll('button').find((b) => b.text() === text)
  expect(btn, `button "${text}" should exist`).toBeTruthy()
  return btn!
}

describe('ObjectToolbar', () => {
  it('渲染桶选择器与标题/操作按钮', () => {
    const w = mount(ObjectToolbar, { props: baseProps })
    const opts = w.find('select').findAll('option').map((o) => o.text())
    expect(opts).toEqual(['my-bucket', 'other-bucket'])
    expect((w.find('select').element as HTMLSelectElement).value).toBe('my-bucket')
    // 操作按钮（i18n mock 直接回显 key）
    expect(w.text()).toContain('common.refresh')
    expect(w.text()).toContain('toolbar.upload')
    expect(w.text()).toContain('toolbar.mkdir')
    expect(w.text()).toContain('toolbar.bucketProps')
    expect(w.text()).toContain('toolbar.backBuckets')
  })

  it('切换桶选项触发 bucket-change', async () => {
    const w = mount(ObjectToolbar, { props: baseProps })
    await w.find('select').setValue('other-bucket')
    expect(w.emitted('bucket-change')?.[0]).toEqual(['other-bucket'])
  })

  it('选中状态为 0 时禁用批量操作，选中后启用并显示全选', async () => {
    const w = mount(ObjectToolbar, { props: baseProps })
    for (const text of ['toolbar.copyTo', 'toolbar.moveTo', 'toolbar.copyLink', 'toolbar.zipDownload', 'toolbar.batchEdit', 'common.delete']) {
      expect(findButton(w, text).attributes('disabled')).toBeDefined()
    }
    await w.setProps({ selectedCount: 2, selectedSize: 300, allSelected: true })
    for (const text of ['toolbar.copyTo', 'toolbar.moveTo', 'toolbar.copyLink', 'toolbar.zipDownload', 'toolbar.batchEdit', 'common.delete']) {
      expect(findButton(w, text).attributes('disabled')).toBeUndefined()
    }
    const checkbox = w.find('input[type="checkbox"]').element as HTMLInputElement
    expect(checkbox.checked).toBe(true)
    // 选中统计 badge
    expect(w.text()).toContain('toolbar.selectedStats')
  })

  it('点击上传/刷新/新建目录/视图切换按钮触发对应事件', async () => {
    const w = mount(ObjectToolbar, { props: baseProps })
    await findButton(w, 'toolbar.upload').trigger('click')
    expect(w.emitted('upload')).toBeTruthy()
    await findButton(w, 'common.refresh').trigger('click')
    expect(w.emitted('refresh')).toBeTruthy()
    // mkdir 按钮文案带 "+ " 前缀
    const mkdirBtn = w.findAll('button').find((b) => b.text().includes('toolbar.mkdir'))
    expect(mkdirBtn).toBeTruthy()
    await mkdirBtn!.trigger('click')
    expect(w.emitted('mkdir')).toBeTruthy()
    // list 视图 → 按钮文案为 gridView（带 📊 图标前缀）
    const viewBtn = w.findAll('button').find((b) => b.text().includes('toolbar.gridView'))
    expect(viewBtn).toBeTruthy()
    await viewBtn!.trigger('click')
    expect(w.emitted('toggle-view')).toBeTruthy()
  })

  it('路径编辑模式：Enter 提交、Esc 取消、✎ 切换', async () => {
    const w = mount(ObjectToolbar, { props: { ...baseProps, pathEditing: true } })
    const input = w.find('.path-input')
    expect(input.exists()).toBe(true)
    await input.trigger('keydown.enter')
    expect(w.emitted('commit-path')).toBeTruthy()
    await input.trigger('keydown.esc')
    expect(w.emitted('cancel-path-edit')).toBeTruthy()
    // 非编辑态：✎ 按钮切换
    await w.setProps({ pathEditing: false })
    await findButton(w, '✎').trigger('click')
    expect(w.emitted('toggle-path-edit')).toBeTruthy()
  })

  it('过滤输入 v-model 更新 filter 并显示 badge', async () => {
    const w = mount(ObjectToolbar, { props: { ...baseProps, filterActive: true, visibleCount: 1, totalCount: 4 } })
    await w.find('input[type="search"]').setValue('abc')
    expect(w.emitted('update:filter')?.[0]).toEqual(['abc'])
    expect(w.text()).toContain('1/4')
    // 清除按钮清空过滤
    const clear = w.findAll('button').find((b) => b.text() === 'common.clear')
    expect(clear).toBeTruthy()
    await clear!.trigger('click')
    const filterEvents = w.emitted('update:filter')!
    expect(filterEvents[filterEvents.length - 1]).toEqual([''])
  })
})
