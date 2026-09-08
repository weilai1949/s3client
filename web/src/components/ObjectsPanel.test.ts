import { beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick, ref } from 'vue'
import { shallowMount } from '@vue/test-utils'
import ObjectsPanel from './ObjectsPanel.vue'
import ObjectList from './ObjectList.vue'
import ObjectToolbar from './ObjectToolbar.vue'
import { useObjectBrowser } from '../composables/useObjectBrowser'
import { useObjectActions } from '../composables/useObjectActions'
import { usePreview } from '../composables/usePreview'
import { requestAccountForm } from '../store'
import type { Account, Entry } from '../types'

vi.mock('../composables/useObjectBrowser', () => ({ useObjectBrowser: vi.fn() }))
vi.mock('../composables/useObjectActions', () => ({ useObjectActions: vi.fn() }))
vi.mock('../composables/usePreview', () => ({ usePreview: vi.fn() }))
vi.mock('../store', () => ({
  state: { accounts: [], currentAccountId: 'acc-1' },
  requestAccountForm: vi.fn(),
  toast: vi.fn(),
}))
vi.mock('../i18n', () => ({
  t: (k: string) => k,
  tf: (k: string) => k,
}))

const account: Account = {
  id: 'acc-1',
  name: 'my-account',
  endpoint: 'http://minio:9000',
  region: 'us-east-1',
  accessKey: 'ak',
  secretKey: 'sk',
  bucket: 'b1',
  pathStyle: true,
  useSSL: false,
}

const sampleEntries: Entry[] = [
  { kind: 'file', key: 'a.txt', name: 'a.txt', size: 10, lastModified: '2024-01-01', object: { key: 'a.txt', size: 10, lastModified: '2024-01-01', etag: 'e1', contentType: 'text/plain', isDir: false } },
  { kind: 'folder', key: 'dir/', name: 'dir' },
]

function makeBrowser(overrides: Record<string, unknown> = {}) {
  return {
    accSel: ref('acc-1'),
    account: ref<Account | null>(account),
    currentBucket: ref('b1'),
    ctxMenu: ref(null),
    buckets: ref([{ name: 'b1', creationDate: '2024-01-01' }]),
    loadingBuckets: ref(false),
    loading: ref(false),
    prefix: ref(''),
    crumbs: ref([]),
    pathEditing: ref(false),
    filterActive: ref(false),
    visibleEntries: ref(sampleEntries),
    entries: ref(sampleEntries),
    allSelected: ref(false),
    selected: ref(new Set<string>()),
    selectedSize: ref(0),
    fileObjects: ref([]),
    loadedSize: ref(0),
    bucketView: ref('list'),
    opsBusy: ref(false),
    filter: ref(''),
    pathDraft: ref(''),
    hintsHidden: ref(false),
    error: ref(''),
    sortKey: ref('name'),
    sortDir: ref(1),
    nextToken: ref(''),
    isTruncated: ref(false),
    loadingAll: ref(false),
    creatingBucket: ref(false),
    onBucketSelect: vi.fn(),
    goRoot: vi.fn(),
    enterPrefix: vi.fn(),
    goUp: vi.fn(),
    startPathEdit: vi.fn(),
    commitPath: vi.fn(),
    cancelPathEdit: vi.fn(),
    togglePathEdit: vi.fn(),
    refreshAll: vi.fn(),
    backToBuckets: vi.fn(),
    selectAll: vi.fn(),
    toggleView: vi.fn(),
    loadMore: vi.fn(),
    loadAll: vi.fn(),
    onRowClick: vi.fn(),
    onRowDblClick: vi.fn(),
    openCtx: vi.fn(),
    openCtxFromButton: vi.fn(),
    toggleWithShift: vi.fn(),
    toggleSort: vi.fn(),
    openCreateBucket: vi.fn(),
    enterBucket: vi.fn(),
    removeBucket: vi.fn(),
    onCreateBucket: vi.fn(),
    hideHints: vi.fn(),
    ...overrides,
  }
}

function makeActions(overrides: Record<string, unknown> = {}) {
  return {
    uploading: ref(false),
    zipLoading: ref(false),
    uploadQueue: ref([]),
    uploadInput: ref(null),
    abortUploadItem: vi.fn(),
    detail: ref(null),
    headersOpen: ref(false),
    headersKey: ref(''),
    tagsOpen: ref(false),
    tagsKey: ref(''),
    lifecycleOpen: ref(false),
    lifecycleBucket: ref(''),
    destOpen: ref(false),
    destCtx: ref(null),
    aclOpen: ref(false),
    aclKey: ref(''),
    bucketInfoOpen: ref(false),
    versionsOpen: ref(false),
    versionsKey: ref(''),
    openBucketInfo: vi.fn(),
    pickUploadFiles: vi.fn(),
    mkdir: vi.fn(),
    openDestMulti: vi.fn(),
    copySelectedLinks: vi.fn(),
    downloadSelectedZip: vi.fn(),
    removeSelected: vi.fn(),
    onPickUpload: vi.fn(),
    download: vi.fn(),
    openLifecycle: vi.fn(),
    onHeadersSaved: vi.fn(),
    openHeadersDialog: vi.fn(),
    openAcl: vi.fn(),
    openTagsDialog: vi.fn(),
    openVersions: vi.fn(),
    storageClassOpen: ref(false),
    storageClassKey: ref(''),
    openStorageClass: vi.fn(),
    onStorageClassSaved: vi.fn(),
    onDestSubmit: vi.fn(),
    ctxOpen: vi.fn(),
    ctxCopyLink: vi.fn(),
    ctxCopyKey: vi.fn(),
    ctxCopyFolder: vi.fn(),
    ctxMoveFolder: vi.fn(),
    ctxDeleteFolder: vi.fn(),
    ctxCopyFile: vi.fn(),
    ctxMoveFile: vi.fn(),
    ctxRename: vi.fn(),
    ctxDetail: vi.fn(),
    ctxAcl: vi.fn(),
    ctxTags: vi.fn(),
    batchOpen: ref(false),
    openBatch: vi.fn(),
    onBatchDone: vi.fn(),
    ctxVersions: vi.fn(),
    ctxDelete: vi.fn(),
    ...overrides,
  }
}

const toolbarStub = {
  name: 'ObjectToolbar',
  props: ['bucket'],
  inheritAttrs: false,
  template: '<div class="toolbar-stub" />',
}

function shallowPanel() {
  return shallowMount(ObjectsPanel, {
    global: {
      stubs: {
        // 显式 stub：浅渲染的默认 stub 会把 prefix 作为 DOM property 写回
        // 自定义元素，而 Node.prefix 是只读 getter，触发 Vue warn。
        ObjectToolbar: toolbarStub,
      },
    },
  })
}

function mountPanel() {
  vi.mocked(useObjectBrowser).mockReturnValue(makeBrowser() as any)
  vi.mocked(useObjectActions).mockReturnValue(makeActions() as any)
  vi.mocked(usePreview).mockReturnValue({ preview: ref(null), showPreview: vi.fn(), previewOrDownload: vi.fn(), ctxPreview: vi.fn() } as any)
  return shallowPanel()
}

beforeEach(() => {
  vi.clearAllMocks()
})

describe('ObjectsPanel', () => {
  it('渲染对象列表（ObjectList 收到可见条目）', () => {
    const w = mountPanel()
    const list = w.findComponent(ObjectList)
    expect(list.exists()).toBe(true)
    expect(list.props('entries')).toHaveLength(2)
    expect(list.props('selected')).toBeInstanceOf(Set)
    // 工具栏 stub 收到选中统计
    const toolbar = w.findComponent(ObjectToolbar)
    expect(toolbar.exists()).toBe(true)
    expect(toolbar.props('bucket')).toBe('b1')
  })

  it('错误横幅渲染错误信息，点击关闭按钮可清空', async () => {
    vi.mocked(useObjectBrowser).mockReturnValue(
      makeBrowser({ error: ref('upload failed: 403') }) as any,
    )
    vi.mocked(useObjectActions).mockReturnValue(makeActions() as any)
    vi.mocked(usePreview).mockReturnValue({ preview: ref(null), showPreview: vi.fn(), previewOrDownload: vi.fn(), ctxPreview: vi.fn() } as any)
    const w = shallowPanel()

    const banner = w.find('.msg.err')
    expect(banner.exists()).toBe(true)
    expect(banner.text()).toContain('upload failed: 403')
    // 关闭按钮（关闭/重试两个 link 按钮，第一个为 close）
    const closeBtn = banner.findAll('button')[0]
    expect(closeBtn.text()).toBe('common.close')
    await closeBtn.trigger('click')
    expect(w.find('.msg.err').exists()).toBe(false)
  })

  it('无账号时显示空态，可引导创建账号', async () => {
    vi.mocked(useObjectBrowser).mockReturnValue(
      makeBrowser({ account: ref<Account | null>(null) }) as any,
    )
    vi.mocked(useObjectActions).mockReturnValue(makeActions() as any)
    vi.mocked(usePreview).mockReturnValue({ preview: ref(null), showPreview: vi.fn(), previewOrDownload: vi.fn(), ctxPreview: vi.fn() } as any)
    const w = shallowPanel()

    expect(w.find('.empty').exists()).toBe(true)
    // 无账号分支：acc-select 显示空账号占位 option（另一个分支由 accounts 选项测试覆盖）
    const emptyOption = w.find('select.acc-select option')
    expect(emptyOption.exists()).toBe(true)
    expect(emptyOption.attributes('value')).toBe('')
    expect(emptyOption.text()).toBe('objects.noAccountOption')
    const createBtn = w.findAll('button').find((b) => b.text() === 'objects.createFirst')
    expect(createBtn).toBeTruthy()
    await createBtn!.trigger('click')
    expect(requestAccountForm).toHaveBeenCalledTimes(1)
  })
})

describe('ObjectsPanel wiring', () => {
  it('bucket-list view renders when no currentBucket and forwards bucket events', async () => {
    vi.mocked(useObjectBrowser).mockReturnValue(makeBrowser({ currentBucket: ref('') }) as any)
    vi.mocked(useObjectActions).mockReturnValue(makeActions() as any)
    vi.mocked(usePreview).mockReturnValue({ preview: ref(null), showPreview: vi.fn(), previewOrDownload: vi.fn(), ctxPreview: vi.fn() } as any)
    const w = shallowPanel()
    const list = w.findComponent({ name: 'BucketList' })
    expect(list.exists()).toBe(true)
    ;(list.vm as any).$emit('enter', 'b2')
    expect(vi.mocked(useObjectBrowser).mock.results[0].value.enterBucket).toHaveBeenCalledWith('b2')
    ;(list.vm as any).$emit('create', { name: 'nb' })
    expect(vi.mocked(useObjectBrowser).mock.results[0].value.openCreateBucket).toHaveBeenCalled()
    ;(list.vm as any).$emit('remove', { name: 'b2' })
    expect(vi.mocked(useObjectBrowser).mock.results[0].value.removeBucket).toHaveBeenCalled()
    ;(list.vm as any).$emit('lifecycle', 'b2')
    expect(vi.mocked(useObjectActions).mock.results[0].value.openLifecycle).toHaveBeenCalled()
  })

  it('toolbar events forward to browser/actions', async () => {
    const w = mountPanel()
    const tb = w.findComponent(ObjectToolbar)
    const b = vi.mocked(useObjectBrowser).mock.results[0].value
    const a = vi.mocked(useObjectActions).mock.results[0].value
    tb.vm.$emit('open-bucket-info'); expect(a.openBucketInfo).toHaveBeenCalled()
    tb.vm.$emit('go-root'); expect(b.goRoot).toHaveBeenCalled()
    tb.vm.$emit('refresh'); expect(b.refreshAll).toHaveBeenCalled()
    tb.vm.$emit('upload'); expect(a.pickUploadFiles).toHaveBeenCalled()
    tb.vm.$emit('mkdir'); expect(a.mkdir).toHaveBeenCalled()
    tb.vm.$emit('open-dest-multi'); expect(a.openDestMulti).toHaveBeenCalled()
    tb.vm.$emit('download-zip'); expect(a.downloadSelectedZip).toHaveBeenCalled()
    tb.vm.$emit('remove-selected'); expect(a.removeSelected).toHaveBeenCalled()
    tb.vm.$emit('copy-links'); expect(a.copySelectedLinks).toHaveBeenCalled()
    tb.vm.$emit('open-batch-edit'); expect(a.openBatch).toHaveBeenCalled()
    tb.vm.$emit('toggle-view'); expect(b.toggleView).toHaveBeenCalled()
    tb.vm.$emit('toggle-select-all'); expect(b.selectAll).toHaveBeenCalled()
    tb.vm.$emit('start-path-edit'); expect(b.startPathEdit).toHaveBeenCalled()
    tb.vm.$emit('commit-path'); expect(b.commitPath).toHaveBeenCalled()
    tb.vm.$emit('back-to-buckets'); expect(b.backToBuckets).toHaveBeenCalled()
    tb.vm.$emit('toggle-path-edit'); expect(b.togglePathEdit).toHaveBeenCalled()
    tb.vm.$emit('enter-prefix', 'd/'); expect(b.enterPrefix).toHaveBeenCalledWith('d/')
    tb.vm.$emit('bucket-change', 'b2'); expect(b.onBucketSelect).toHaveBeenCalled()
  })

  it('object list events forward to browser/actions', async () => {
    const w = mountPanel()
    const list = w.findComponent(ObjectList)
    const b = vi.mocked(useObjectBrowser).mock.results[0].value
    const a = vi.mocked(useObjectActions).mock.results[0].value
    const p = vi.mocked(usePreview).mock.results[0].value
    list.vm.$emit('row-click', { kind: 'file', key: 'a.txt' })
    expect(b.onRowClick).toHaveBeenCalled()
    list.vm.$emit('ctx', { entry: {} })
    expect(b.openCtx).toHaveBeenCalled()
    list.vm.$emit('check-toggle', 'a.txt', true)
    expect(b.toggleWithShift).toHaveBeenCalled()
    list.vm.$emit('load-more')
    expect(b.loadMore).toHaveBeenCalled()
    list.vm.$emit('load-all')
    expect(b.loadAll).toHaveBeenCalled()
    list.vm.$emit('preview', { key: 'a.txt' })
    expect(p.showPreview).toHaveBeenCalled()
    list.vm.$emit('download', { key: 'a.txt' })
    expect(a.download).toHaveBeenCalled()
    list.vm.$emit('enter-dir', 'd/')
    expect(b.enterPrefix).toHaveBeenCalledWith('d/')
    list.vm.$emit('toggle-sort', 'size')
    expect(b.toggleSort).toHaveBeenCalled()
  })

  it('upload queue, hints bar, error retry and dialogs wiring', async () => {
    const actions = makeActions({ uploadQueue: ref([{ id: 1, key: 'a.txt', pct: 10, status: 'uploading' }] as any) })
    vi.mocked(useObjectActions).mockReturnValue(actions as any)
    vi.mocked(useObjectBrowser).mockReturnValue(makeBrowser({ error: ref('x'), hintsHidden: ref(false) }) as any)
    vi.mocked(usePreview).mockReturnValue({ preview: ref(null), showPreview: vi.fn(), previewOrDownload: vi.fn(), ctxPreview: vi.fn() } as any)
    const w = shallowPanel()

    const q = w.findComponent({ name: 'UploadQueue' })
    expect(q.exists()).toBe(true)
    q.vm.$emit('cancel', 1)
    expect(actions.abortUploadItem).toHaveBeenCalled()

    // hints bar
    const dismiss = w.findAll('button').find((b) => b.text() === 'objects.hintsDismiss')!
    await dismiss.trigger('click')
    expect(vi.mocked(useObjectBrowser).mock.results[0].value.hideHints).toHaveBeenCalled()

    // error banner retry
    const retry = w.findAll('button').find((b) => b.text() === 'common.retry')!
    await retry.trigger('click')
    expect(vi.mocked(useObjectBrowser).mock.results[0].value.refreshAll).toHaveBeenCalled()
  })

  it('dialogs get open props and forward errors', async () => {
    const actions = makeActions({
      headersOpen: ref(true), tagsOpen: ref(true), lifecycleOpen: ref(true),
      aclOpen: ref(true), storageClassOpen: ref(true), destOpen: ref(true),
      batchOpen: ref(true), versionsOpen: ref(true), bucketInfoOpen: ref(true),
      detail: ref({ key: 'a.txt', size: 1 } as any),
    })
    vi.mocked(useObjectActions).mockReturnValue(actions as any)
    vi.mocked(useObjectBrowser).mockReturnValue(makeBrowser({ creatingBucket: ref(true) }) as any)
    vi.mocked(usePreview).mockReturnValue({ preview: ref(null), showPreview: vi.fn(), previewOrDownload: vi.fn(), ctxPreview: vi.fn() } as any)
    const w = shallowPanel()

    for (const name of ['ObjectDetailDialog', 'HeadersDialog', 'TagsDialog', 'LifecycleDialog', 'DestDialog', 'AclDialog', 'StorageClassDialog', 'BatchMetadataDialog', 'CreateBucketDialog']) {
      expect(w.findComponent({ name }).exists()).toBe(true)
    }
    const detail = w.findComponent({ name: 'ObjectDetailDialog' })
    expect(detail.props('open')).toBe(true)
    // error 转发
    detail.vm.$emit('error', 'boom')
    expect(vi.mocked(useObjectBrowser).mock.results[0].value.error.value).toBe('boom')
  })

  it('toolbar v-model 更新 filter / pathDraft', async () => {
    const w = mountPanel()
    const tb = w.findComponent(ObjectToolbar)
    const b = vi.mocked(useObjectBrowser).mock.results[0].value
    tb.vm.$emit('update:filter', 'abc')
    expect(b.filter.value).toBe('abc')
    tb.vm.$emit('update:path-draft', 'dir/sub')
    expect(b.pathDraft.value).toBe('dir/sub')
  })

  it('hintsHidden=true 时快捷键提示条不渲染', () => {
    vi.mocked(useObjectBrowser).mockReturnValue(makeBrowser({ hintsHidden: ref(true) }) as any)
    vi.mocked(useObjectActions).mockReturnValue(makeActions() as any)
    vi.mocked(usePreview).mockReturnValue({ preview: ref(null), showPreview: vi.fn(), previewOrDownload: vi.fn(), ctxPreview: vi.fn() } as any)
    const w = shallowPanel()
    expect(w.find('.hints-bar').exists()).toBe(false)
  })

  it('getCtxEntry 读取右键菜单 entry（usePreview 注入的闭包）', () => {
    const browser = makeBrowser({ ctxMenu: ref(null) })
    vi.mocked(useObjectBrowser).mockReturnValue(browser as any)
    vi.mocked(useObjectActions).mockReturnValue(makeActions() as any)
    let cfg: { getCtxEntry: () => unknown } | undefined
    vi.mocked(usePreview).mockImplementation((c: any) => {
      cfg = c
      return { preview: ref(null), showPreview: vi.fn(), previewOrDownload: vi.fn(), ctxPreview: vi.fn() } as any
    })
    shallowPanel()
    expect(cfg).toBeDefined()
    // 无右键菜单 → undefined
    expect(cfg!.getCtxEntry()).toBeUndefined()
    // 有右键菜单 → 返回 entry
    const entry = { kind: 'file', key: 'a.txt', name: 'a.txt', object: { key: 'a.txt' } }
    ;(browser.ctxMenu as any).value = { x: 1, y: 1, entry }
    expect(cfg!.getCtxEntry()).toEqual(entry)
  })

  it('dialogs close/error 事件回写面板状态（全部弹窗 + 预览 overlay）', async () => {
    const browser = makeBrowser({ creatingBucket: ref(true) })
    const actions = makeActions({
      detail: ref({ key: 'a.txt', size: 1, storageClass: 'STANDARD' } as any),
      headersOpen: ref(true), headersKey: ref('a.txt'),
      tagsOpen: ref(true), tagsKey: ref('a.txt'),
      lifecycleOpen: ref(true), lifecycleBucket: ref('b1'),
      destOpen: ref(true), destCtx: ref({ kind: 'file', mode: 'copy', key: 'a.txt' } as any),
      aclOpen: ref(true), aclKey: ref('a.txt'),
      storageClassOpen: ref(true), storageClassKey: ref('a.txt'),
      batchOpen: ref(true), versionsOpen: ref(true), versionsKey: ref('a.txt'), bucketInfoOpen: ref(true),
    })
    vi.mocked(useObjectActions).mockReturnValue(actions as any)
    vi.mocked(useObjectBrowser).mockReturnValue(browser as any)
    const preview = ref<unknown>({ key: 'a.txt', kind: 'text' })
    vi.mocked(usePreview).mockReturnValue({ preview, showPreview: vi.fn(), previewOrDownload: vi.fn(), ctxPreview: vi.fn() } as any)
    const w = shallowPanel()

    const detail = w.findComponent({ name: 'ObjectDetailDialog' })
    expect(detail.props('open')).toBe(true)
    expect(detail.props('detail')).toBeTruthy()
    detail.vm.$emit('close')
    expect(actions.detail.value).toBeNull()

    const create = w.findComponent({ name: 'CreateBucketDialog' })
    expect(create.props('open')).toBe(true)
    expect(create.props('accountId')).toBe('acc-1')
    create.vm.$emit('close')
    expect(browser.creatingBucket.value).toBe(false)
    create.vm.$emit('error', 'create-err')
    expect(browser.error.value).toBe('create-err')

    const headers = w.findComponent({ name: 'HeadersDialog' })
    expect(headers.props('open')).toBe(true)
    expect(headers.props('objectKey')).toBe('a.txt')
    headers.vm.$emit('close')
    expect(actions.headersOpen.value).toBe(false)
    headers.vm.$emit('error', 'headers-err')
    expect(browser.error.value).toBe('headers-err')

    const tags = w.findComponent({ name: 'TagsDialog' })
    expect(tags.props('open')).toBe(true)
    expect(tags.props('objectKey')).toBe('a.txt')
    tags.vm.$emit('close')
    expect(actions.tagsOpen.value).toBe(false)
    tags.vm.$emit('error', 'tags-err')
    expect(browser.error.value).toBe('tags-err')

    const lifecycle = w.findComponent({ name: 'LifecycleDialog' })
    expect(lifecycle.props('open')).toBe(true)
    expect(lifecycle.props('bucket')).toBe('b1')
    lifecycle.vm.$emit('close')
    expect(actions.lifecycleOpen.value).toBe(false)
    lifecycle.vm.$emit('error', 'lifecycle-err')
    expect(browser.error.value).toBe('lifecycle-err')

    const dest = w.findComponent({ name: 'DestDialog' })
    expect(dest.props('open')).toBe(true)
    expect(dest.props('kind')).toBe('file')
    expect(dest.props('mode')).toBe('copy')
    expect(dest.props('objectKey')).toBe('a.txt')
    dest.vm.$emit('close')
    expect(actions.destOpen.value).toBe(false)
    dest.vm.$emit('error', 'dest-err')
    expect(browser.error.value).toBe('dest-err')

    const acl = w.findComponent({ name: 'AclDialog' })
    expect(acl.props('open')).toBe(true)
    expect(acl.props('objectKey')).toBe('a.txt')
    acl.vm.$emit('close')
    expect(actions.aclOpen.value).toBe(false)
    acl.vm.$emit('error', 'acl-err')
    expect(browser.error.value).toBe('acl-err')

    const storage = w.findComponent({ name: 'StorageClassDialog' })
    expect(storage.props('open')).toBe(true)
    expect(storage.props('objectKey')).toBe('a.txt')
    expect(storage.props('currentClass')).toBe('STANDARD')
    // detail 无 storageClass 时 currentClass 回退为空串（ObjectMeta.storageClass 可选）
    actions.detail.value = { key: 'a.txt', size: 1 } as any
    await nextTick()
    expect(storage.props('currentClass')).toBe('')
    storage.vm.$emit('close')
    expect(actions.storageClassOpen.value).toBe(false)
    storage.vm.$emit('error', 'storage-err')
    expect(browser.error.value).toBe('storage-err')

    const batch = w.findComponent({ name: 'BatchMetadataDialog' })
    // BatchMetadataDialog 未声明 open prop（直接透传），校验数据 props
    expect(batch.props('accountId')).toBe('acc-1')
    expect(batch.props('keys')).toEqual([])
    batch.vm.$emit('close')
    expect(actions.batchOpen.value).toBe(false)

    const info = w.findComponent({ name: 'BucketInfoDialog' })
    expect(info.props('open')).toBe(true)
    expect(info.props('bucket')).toBe('b1')
    info.vm.$emit('close')
    expect(actions.bucketInfoOpen.value).toBe(false)
    info.vm.$emit('error', 'info-err')
    expect(browser.error.value).toBe('info-err')

    const versions = w.findComponent({ name: 'VersionsDialog' })
    expect(versions.props('open')).toBe(true)
    expect(versions.props('objectKey')).toBe('a.txt')
    versions.vm.$emit('close')
    expect(actions.versionsOpen.value).toBe(false)
    versions.vm.$emit('error', 'versions-err')
    expect(browser.error.value).toBe('versions-err')

    const overlay = w.findComponent({ name: 'PreviewOverlay' })
    expect(overlay.props('preview')).toEqual({ key: 'a.txt', kind: 'text' })
    overlay.vm.$emit('close')
    expect(preview.value).toBeNull()
    overlay.vm.$emit('error', 'preview-err')
    expect(browser.error.value).toBe('preview-err')
  })
})

describe('ObjectsPanel account select + upload input', () => {
  it('renders account options and updates accSel on select', async () => {
    const store = (await import('../store')) as { state: { accounts: any[]; currentAccountId: string } }
    store.state.accounts = [account, { id: 'acc-2', name: 'other' } as Account]
    const w = mountPanel()
    const sel = w.find('select.acc-select')
    expect(sel.exists()).toBe(true)
    const opts = sel.findAll('option')
    expect(opts).toHaveLength(2)
    expect(opts[0].text()).toBe('my-account')
    await sel.setValue('acc-2')
    expect(vi.mocked(useObjectBrowser).mock.results[0].value.accSel.value).toBe('acc-2')
    store.state.accounts = []
  })

  it('hidden file input change delegates to onPickUpload', () => {
    const actions = makeActions()
    vi.mocked(useObjectActions).mockReturnValue(actions as any)
    vi.mocked(useObjectBrowser).mockReturnValue(makeBrowser() as any)
    vi.mocked(usePreview).mockReturnValue({ preview: ref(null), showPreview: vi.fn(), previewOrDownload: vi.fn(), ctxPreview: vi.fn() } as any)
    const w = shallowPanel()
    const input = w.find('input[type="file"]')
    expect(input.exists()).toBe(true)
    // happy-dom: 直接触发 change 事件
    const files = [new File(['x'], 'a.txt')]
    Object.defineProperty(input.element, 'files', { value: files, configurable: true })
    ;(input.element as HTMLInputElement).dispatchEvent(new Event('change'))
    expect(actions.onPickUpload).toHaveBeenCalled()
  })
})
