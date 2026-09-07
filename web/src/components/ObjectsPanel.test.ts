import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
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
    const createBtn = w.findAll('button').find((b) => b.text() === 'objects.createFirst')
    expect(createBtn).toBeTruthy()
    await createBtn!.trigger('click')
    expect(requestAccountForm).toHaveBeenCalledTimes(1)
  })
})
