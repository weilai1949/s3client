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

  it('无桶时显示 noBucket 占位选项；loadingBuckets 禁用选择器；空桶禁用桶属性/向上', () => {
    const w = mount(ObjectToolbar, {
      props: { ...baseProps, buckets: [], bucket: '', prefix: '', loadingBuckets: true },
    })
    const sel = w.find('select')
    expect(sel.attributes('disabled')).toBeDefined()
    const opts = sel.findAll('option').map((o) => o.text())
    expect(opts).toEqual(['toolbar.noBucket'])
    // 无 bucket → 桶属性/向上按钮禁用
    expect(findButton(w, 'toolbar.bucketProps').attributes('disabled')).toBeDefined()
    expect(findButton(w, 'toolbar.goUp').attributes('disabled')).toBeDefined()
    // 路径根按钮显示占位文案 Bucket
    expect(w.findAll('button.link').some((b) => b.text() === 'Bucket')).toBe(true)
  })

  it('面包屑：根按钮 go-root，非当前路径 crumb 进入前缀，当前路径为纯文本', async () => {
    const w = mount(ObjectToolbar, {
      props: {
        ...baseProps,
        crumbs: [
          { name: 'dir', path: 'dir/' },
          { name: 'sub', path: 'dir/sub/' },
        ],
      },
    })
    await findButton(w, 'my-bucket').trigger('click')
    expect(w.emitted('go-root')).toBeTruthy()
    // 当前前缀为纯文本（无按钮）
    expect(w.findAll('span.cur').map((s) => s.text())).toEqual(['dir'])
    await findButton(w, 'sub').trigger('click')
    expect(w.emitted('enter-prefix')?.[0]).toEqual(['dir/sub/'])
  })

  it('桶属性按钮触发 open-bucket-info；向上/返回桶列表触发对应事件', async () => {
    const w = mount(ObjectToolbar, { props: baseProps })
    await findButton(w, 'toolbar.bucketProps').trigger('click')
    expect(w.emitted('open-bucket-info')).toBeTruthy()
    await findButton(w, 'toolbar.goUp').trigger('click')
    expect(w.emitted('go-up')).toBeTruthy()
    await findButton(w, 'toolbar.backBuckets').trigger('click')
    expect(w.emitted('back-to-buckets')).toBeTruthy()
  })

  it('上传中显示 uploading 文案并禁用上传；loading/opsBusy 禁用刷新/上传', async () => {
    const w = mount(ObjectToolbar, { props: baseProps })
    await w.setProps({ uploading: true })
    const upBtn = findButton(w, 'toolbar.uploading')
    expect(upBtn.attributes('disabled')).toBeDefined()
    await w.setProps({ loading: true })
    expect(findButton(w, 'common.refresh').attributes('disabled')).toBeDefined()
    expect(findButton(w, 'toolbar.backBuckets').attributes('disabled')).toBeDefined()
    await w.setProps({ uploading: false, loading: false, opsBusy: true })
    expect(findButton(w, 'common.refresh').attributes('disabled')).toBeDefined()
    expect(findButton(w, 'toolbar.upload').attributes('disabled')).toBeDefined()
  })

  it('全选 checkbox 触发 toggle-select-all', async () => {
    const w = mount(ObjectToolbar, { props: baseProps })
    const cb = w.find('input[type="checkbox"]')
    expect((cb.element as HTMLInputElement).checked).toBe(false)
    await cb.setValue(true)
    expect(w.emitted('toggle-select-all')).toBeTruthy()
  })

  it('批量操作按钮逐一触发对应事件', async () => {
    const w = mount(ObjectToolbar, { props: { ...baseProps, selectedCount: 2, selectedSize: 64 } })
    await findButton(w, 'toolbar.copyTo').trigger('click')
    expect(w.emitted('open-dest-multi')?.[0]).toEqual(['copy'])
    await findButton(w, 'toolbar.moveTo').trigger('click')
    expect(w.emitted('open-dest-multi')?.[1]).toEqual(['move'])
    await findButton(w, 'toolbar.copyLink').trigger('click')
    expect(w.emitted('copy-links')).toBeTruthy()
    await findButton(w, 'toolbar.zipDownload').trigger('click')
    expect(w.emitted('download-zip')).toBeTruthy()
    await findButton(w, 'toolbar.batchEdit').trigger('click')
    expect(w.emitted('open-batch-edit')).toBeTruthy()
    await findButton(w, 'common.delete').trigger('click')
    expect(w.emitted('remove-selected')).toBeTruthy()
    // 选中统计 badge
    expect(w.text()).toContain('toolbar.selectedFiles')
  })

  it('zipLoading 显示 zipping 文案并禁用下载', () => {
    const w = mount(ObjectToolbar, { props: { ...baseProps, selectedCount: 1, zipLoading: true } })
    const zip = findButton(w, 'toolbar.zipping')
    expect(zip.attributes('disabled')).toBeDefined()
  })

  it('路径输入 v-model 更新 pathDraft；blur 提交路径', async () => {
    const w = mount(ObjectToolbar, { props: { ...baseProps, pathEditing: true } })
    const input = w.find('.path-input')
    await input.setValue('newdir/')
    expect(w.emitted('update:pathDraft')?.[0]).toEqual(['newdir/'])
    await input.trigger('blur')
    expect(w.emitted('commit-path')).toBeTruthy()
  })

  it('grid 视图切换按钮显示 listView 文案', () => {
    const w = mount(ObjectToolbar, { props: { ...baseProps, bucketView: 'grid' } })
    const viewBtn = w.findAll('button').find((b) => b.text().includes('toolbar.listView'))
    expect(viewBtn).toBeTruthy()
  })

  it('编辑态路径按钮显示 ✓ 并触发 toggle-path-edit', async () => {
    const w = mount(ObjectToolbar, { props: { ...baseProps, pathEditing: true } })
    await findButton(w, '✓').trigger('click')
    expect(w.emitted('toggle-path-edit')).toBeTruthy()
  })
})
