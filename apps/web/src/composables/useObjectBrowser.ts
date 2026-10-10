import { computed, onActivated, onBeforeUnmount, onDeactivated, onMounted, ref, watch } from 'vue'
import { toErrorMessage } from '../errors'

import { s3api } from '../api'
import { state, currentAccount, toast, selectAccount } from '../store'
import { confirmDialog } from '../confirm'
import { t, tf } from '../i18n'
import type { BucketItem, ObjectItem } from '../types'
import type { Entry, SortKey } from '../types'

/** 右键菜单状态（面板传给 ObjectContextMenu，动作/预览经此读取当前条目）。 */
export interface CtxMenu {
  x: number
  y: number
  entry: Entry
}

/**
 * 跨组合式注入的键盘/双击回调：由面板在创建 actions/preview 后回填。
 * 由于监听器在 onMounted 注册、且只在用户输入时触发，回填时刻不影响行为；
 * 全部可选——未回填时对应快捷键静默忽略（面板初始化期间不会 panic）。
 */
export interface KeyBindings {
  previewOrDownload?: (o: ObjectItem) => void
  ctxRenameKey?: (key: string) => void
  removeSelected?: () => void
}

/** 单页列举条数：请求 maxKeys 与「加载全部」上限文案共用同一来源，避免两处硬编码漂移。 */
const PAGE_SIZE = 100

export function useObjectBrowser(bindings: KeyBindings = {}) {
  const prefix = ref('')
  const currentBucket = ref('')
  const buckets = ref<BucketItem[]>([])
  const objects = ref<ObjectItem[]>([])
  const commonPrefixes = ref<string[]>([])
  const nextToken = ref('')
  const isTruncated = ref(false)
  const loading = ref(false)
  const loadingBuckets = ref(false)
  const error = ref('')
  const selected = ref<Set<string>>(new Set())

  /* ---- 本地过滤（只过滤已加载条目，不重新请求） ---- */
  const filter = ref('')

  /* ---- 列排序：文件夹恒在前，文件按列排序 ---- */
  const sortKey = ref<SortKey>('name')
  const sortDir = ref<1 | -1>(1) // 1=升序 -1=降序

  function toggleSort(k: SortKey) {
    if (sortKey.value === k) sortDir.value = sortDir.value === 1 ? -1 : 1
    else {
      sortKey.value = k
      sortDir.value = 1
    }
  }

  /* ---- 右键菜单 ---- */
  const ctxMenu = ref<CtxMenu | null>(null)

  function openCtx(e: MouseEvent, entry: Entry) {
    ctxMenu.value = { x: e.clientX, y: e.clientY, entry }
  }

  /** 从行内「⋯ 更多」按钮打开菜单：锚定按钮下方、右对齐（互联网表格习惯）。 */
  function openCtxFromButton(e: MouseEvent, entry: Entry) {
    const rect = (e.currentTarget as HTMLElement).getBoundingClientRect()
    ctxMenu.value = { x: rect.right - 190, y: rect.bottom + 4, entry }
  }

  function closeCtx() {
    ctxMenu.value = null
  }

  const account = computed(() => currentAccount())
  const fileObjects = computed(() => objects.value.filter((o) => !o.isDir))
  const allSelected = computed(() => fileObjects.value.length > 0 && selected.value.size === fileObjects.value.length)

  /** 条目比较器：文件夹恒在前；文件按当前列/方向排序。 */
  function compareEntries(a: Entry, b: Entry): number {
    const ra = a.kind === 'folder' ? 0 : 1
    const rb = b.kind === 'folder' ? 0 : 1
    if (ra !== rb) return ra - rb
    if (a.kind === 'folder') return a.name.localeCompare(b.name, undefined, { numeric: true })
    let r: number
    if (sortKey.value === 'name') r = a.name.localeCompare(b.name, undefined, { numeric: true })
    else if (sortKey.value === 'size') r = (a.size ?? 0) - (b.size ?? 0)
    else r = (a.lastModified ?? '').localeCompare(b.lastModified ?? '')
    return r * sortDir.value
  }

  // 单次排序：entries 已按当前列排好（文件夹恒在前），visibleEntries 只做过滤。
  // 此前 entries 只排文件夹、visibleEntries 再排文件，过滤态下会重复排序（KNOWN_ISSUES #11）。
  const entries = computed<Entry[]>(() => {
    const folders: Entry[] = commonPrefixes.value.map((p) => ({
      kind: 'folder',
      key: p,
      name: relName(p, true),
    }))
    const files: Entry[] = fileObjects.value.map((o) => ({
      kind: 'file',
      key: o.key,
      name: relName(o.key, false),
      size: o.size,
      lastModified: o.lastModified,
      object: o,
    }))
    return [...folders, ...files].sort(compareEntries)
  })

  /* 过滤后的可见条目：过滤保持 entries 的既有顺序，因此无需二次排序。 */
  const visibleEntries = computed<Entry[]>(() => {
    const kw = filter.value.trim().toLowerCase()
    if (!kw) return entries.value
    return entries.value.filter((e) => e.name.toLowerCase().includes(kw))
  })

  const filterActive = computed(() => filter.value.trim() !== '')

  const crumbs = computed(() => {
    const parts = prefix.value.split('/').filter(Boolean)
    return parts.map((name, i) => ({ name, path: parts.slice(0, i + 1).join('/') + '/' }))
  })

  function relName(full: string, isFolder: boolean): string {
    let s = full.startsWith(prefix.value) ? full.slice(prefix.value.length) : full
    if (isFolder) s = s.replace(/\/$/, '')
    return s || full.replace(/\/$/, '').split('/').pop() || full
  }

  async function loadBuckets() {
    const acc = account.value
    if (!acc) return
    const seq = ++loadSeq.value
    loadingBuckets.value = true
    try {
      const res = await s3api.listBuckets(acc.id)
      if (seq !== loadSeq.value) return
      buckets.value = res.buckets ?? []
      // 控制台习惯：currentBucket 失效则回到 Bucket 列表页；有默认桶时自动进入
      if (currentBucket.value && !buckets.value.some((b) => b.name === currentBucket.value)) {
        currentBucket.value = ''
      }
      if (!currentBucket.value && acc.bucket) {
        currentBucket.value = acc.bucket
      }
    } catch (e: unknown) {
      if (seq === loadSeq.value) error.value = e instanceof Error ? e.message : String(e)
    } finally {
      if (seq === loadSeq.value) loadingBuckets.value = false
    }
  }

  /* ---- Bucket 管理（控制台化：列表页 / 创建 / 删除 / 进入） ---- */
  const bucketView = ref<'list' | 'grid'>('list')
  const creatingBucket = ref(false)

  function enterBucket(name: string) {
    prefix.value = ''
    currentBucket.value = name
    load(true)
  }

  function openCreateBucket() {
    creatingBucket.value = true
  }

  async function onCreateBucket({ name }: { name: string; region: string; acl: string }) {
    creatingBucket.value = false
    toast(tf('buckets.toastCreatedNamed', { name }))
    await loadBuckets()
    enterBucket(name)
  }

  async function removeBucket(b: BucketItem) {
    const ok = await confirmDialog({
      title: t('buckets.deleteTitle'),
      message: tf('buckets.deleteConfirm', { name: b.name }),
    })
    if (!ok) return
    // 账号可能已被切走/清空：此时无可删除目标，直接返回而不是非空断言。
    const accId = account.value?.id
    if (!accId) return
    try {
      await s3api.deleteBucket(accId, b.name)
      toast(tf('buckets.deleted', { name: b.name }))
      if (currentBucket.value === b.name) currentBucket.value = ''
      await loadBuckets()
    } catch (e) {
      error.value = toErrorMessage(e)
    }
  }

  /* ---- 统计（对象数 / 总大小 / 选中合计，控制台习惯） ---- */
  const loadedSize = computed(() => fileObjects.value.reduce((s, o) => s + o.size, 0))
  /** key → size 查找表：选中合计用它增量维护，避免每次勾选都全表扫 fileObjects。 */
  const sizeByKey = computed(() => new Map(fileObjects.value.map((o) => [o.key, o.size])))
  /** 选中合计大小：由 toggle/toggleWithShift 增量维护，赋值型变更与列表变化经下方 watcher 同步兜底。 */
  const selectedSize = ref(0)
  function recomputeSelectedSize() {
    const byKey = sizeByKey.value
    let sum = 0
    for (const k of selected.value) sum += byKey.get(k) ?? 0
    selectedSize.value = sum
  }
  // 原地增删 selected（Set 身份不变）不会触发本 watcher，故 toggle 内就地增减合计；
  // 赋值型变更（selectAll / load 重置 / 面板外部赋值）与列表变化在此同步重算。
  watch([selected, fileObjects], recomputeSelectedSize, { immediate: true, flush: 'sync' })

  // 导航序号：进入目录/切桶/重置时 +1；过期响应直接丢弃，避免快速导航串数据。
  const loadSeq = ref(0)
  // 换源代次：仅在「整体替换列表」（切目录/切桶/刷新/换账号）时 +1，「加载更多」的
  // 追加不递增。ObjectList 据此区分「该把虚拟窗口归零」与「该保住用户滚动位置」——
  // entries 是过滤+排序后的 computed，每次重算都是新数组身份，单看它分不出这两种。
  const listGen = ref(0)
  // 进行中的列表请求 AbortController：每次新 load 取消旧的；卸载时取消所有。
  let loadCtrl: AbortController | undefined

  /** KeepAlive 下仅在面板可见时响应账号切换，避免后台 Tab 重复拉取。 */
  const panelActive = ref(false)
  let syncedAccountId = state.currentAccountId

  async function onAccountSwitch() {
    prefix.value = ''
    currentBucket.value = ''
    syncedAccountId = state.currentAccountId
    if (account.value) {
      await loadBuckets()
      await load(true)
    }
  }

  async function load(reset = true, seqOverride?: number) {
    const acc = account.value
    if (!acc || !currentBucket.value) return
    const seq = seqOverride ?? ++loadSeq.value
    // 只有「不是 loadAll 自己发起的 reset」才清 loadingAll：loadAll 的首轮正是以
    // reset 语义加载第一页（needFirstPage），清了它就等于批次刚起步就把按钮重新打开，
    // 用户能重复点「加载全部」把当前批次顶掉（UI 进度失真、可重复触发）。
    // loadAll 通过 seqOverride 认领 loadingAll，其 finally 负责归位。
    if (reset && seqOverride === undefined) loadingAll.value = false
    if (reset) listGen.value++ // 换源：ObjectList 需要把虚拟窗口归零（追加不走这条路径）
    loading.value = true
    error.value = ''
    // 取消上一次仍在飞的请求，避免过期响应浪费带宽。
    if (loadCtrl) loadCtrl.abort()
    const ctrl = new AbortController()
    loadCtrl = ctrl
    try {
      const q: Record<string, string> = {
        bucket: currentBucket.value,
        prefix: prefix.value,
        delimiter: '/',
        maxKeys: String(PAGE_SIZE),
      }
      if (!reset && nextToken.value) q.continuationToken = nextToken.value
      const res = await s3api.listObjects(acc.id, q, { signal: ctrl.signal })
      if (seq !== loadSeq.value) return // 过期响应，丢弃
      if (reset) {
        objects.value = res.objects ?? []
        commonPrefixes.value = res.commonPrefixes ?? []
        selected.value = new Set()
        lastSelIdx.value = -1
      } else {
        objects.value = [...objects.value, ...(res.objects ?? [])]
        commonPrefixes.value = [...new Set([...commonPrefixes.value, ...(res.commonPrefixes ?? [])])]
      }
      nextToken.value = res.nextToken
      isTruncated.value = res.isTruncated
    } catch (e: unknown) {
      if (seq === loadSeq.value) error.value = e instanceof Error ? e.message : String(e)
    } finally {
      if (seq === loadSeq.value) loading.value = false
    }
  }

  function onBucketChange() {
    prefix.value = ''
    load(true)
  }

  function enterPrefix(p: string) {
    prefix.value = p
    load(true)
  }

  function goRoot() {
    prefix.value = ''
    load(true)
  }

  function goUp() {
    const parts = prefix.value.split('/').filter(Boolean)
    parts.pop()
    prefix.value = parts.length ? parts.join('/') + '/' : ''
    load(true)
  }

  function toggle(k: string) {
    const s = selected.value
    const size = sizeByKey.value.get(k) ?? 0
    if (s.has(k)) {
      s.delete(k)
      selectedSize.value -= size
    } else {
      s.add(k)
      selectedSize.value += size
    }
  }

  /** 行点击：文件=切换选中，文件夹=进入（文件管理器习惯）。 */
  function onRowClick(e: Entry) {
    if (e.kind === 'folder') enterPrefix(e.key)
    else toggle(e.key)
  }

  function selectAll() {
    if (allSelected.value) {
      selected.value = new Set()
    } else {
      selected.value = new Set(fileObjects.value.map((o) => o.key))
    }
  }

  /* ---- 键盘快捷键（文件管理器习惯；输入框聚焦时不触发） ---- */
  function onGlobalKey(e: KeyboardEvent) {
    // KeepAlive 缓存下监听仍存活：面板被切走后不得用旧 selected 触发删除/预览/全选
    if (!panelActive.value) return
    const el = e.target as HTMLElement | null
    if (el && (el.tagName === 'INPUT' || el.tagName === 'TEXTAREA' || el.tagName === 'SELECT' || el.tagName === 'BUTTON' || el.isContentEditable)) return
    if (!account.value || !currentBucket.value) return
    const files = fileObjects.value.filter((o) => selected.value.has(o.key))
    const first = files[0]
    if (e.key === 'Enter') {
      if (first) {
        e.preventDefault()
        bindings.previewOrDownload?.(first)
      }
    } else if (e.key === 'F2') {
      if (first) {
        e.preventDefault()
        bindings.ctxRenameKey?.(first.key)
      }
    } else if (e.key === 'Delete' || e.key === 'Backspace') {
      if (files.length) {
        e.preventDefault()
        bindings.removeSelected?.()
      }
    } else if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'a') {
      e.preventDefault()
      selectAll()
    }
  }

  /* ---- 账号快速切换（面板头部下拉，无需回账号面板） ---- */
  const accSel = computed({
    get: () => state.currentAccountId,
    set: (v: string) => selectAccount(v),
  })

  /* ---- 快捷键提示条（可关闭） ---- */
  const LS_HINTS = 's3c.hintsHidden'
  // 存储不可用（隐私模式/配额异常）时按「未隐藏」处理，不得让面板初始化失败。
  let hintsStored = false
  try {
    hintsStored = localStorage.getItem(LS_HINTS) === '1'
  } catch {
    /* ignore */
  }
  const hintsHidden = ref(hintsStored)

  function hideHints() {
    hintsHidden.value = true
    try {
      localStorage.setItem(LS_HINTS, '1')
    } catch {
      /* ignore：本次会话内仍然隐藏 */
    }
  }

  /* ---- 耗时操作统一 busy 防重复提交 ---- */
  const opsBusy = ref(false)

  /* ---- 面包屑地址栏（可编辑跳转） ---- */
  const pathEditing = ref(false)
  const pathDraft = ref('')

  function startPathEdit() {
    pathDraft.value = prefix.value
    pathEditing.value = true
  }

  function commitPath() {
    pathEditing.value = false
    const p = pathDraft.value.trim()
    enterPrefix(p ? (p.endsWith('/') ? p : p + '/') : '')
  }

  /* ---- Shift 范围多选（文件管理器习惯） ---- */
  const fileList = computed(() => visibleEntries.value.filter((e) => e.kind === 'file'))
  const lastSelIdx = ref(-1)

  function toggleWithShift(k: string, shift: boolean) {
    const idx = fileList.value.findIndex((e) => e.key === k)
    if (shift && lastSelIdx.value >= 0 && idx >= 0) {
      const s = selected.value
      const [a, b] = [Math.min(lastSelIdx.value, idx), Math.max(lastSelIdx.value, idx)]
      for (let i = a; i <= b; i++) s.add(fileList.value[i].key)
      // 原地增删不触发 selected 身份 watcher：范围补齐后手动重算合计
      recomputeSelectedSize()
    } else {
      toggle(k)
    }
    lastSelIdx.value = idx
  }

  /** 双击：文件=查看（预览，未知类型下载），文件夹=进入。 */
  function onRowDblClick(e: Entry) {
    if (e.kind === 'folder') enterPrefix(e.key)
    else if (e.object) bindings.previewOrDownload?.(e.object)
  }

  async function refreshAll() {
    await loadBuckets()
    await load(true)
  }

  /** 加载全部：循环分页直到末尾（上限保护，避免超大桶卡死）。 */
  const loadingAll = ref(false)
  const MAX_ALL_PAGES = 200 // 单页 PAGE_SIZE 条 → 最多 200 页（上限保护）

  async function loadAll() {
    const acc = account.value
    if (!acc || !currentBucket.value || loading.value) return
    const seq = ++loadSeq.value
    loadingAll.value = true
    error.value = ''
    // 进入时 nextToken 是否为空：为空说明「尚未分页」（刚进目录 / 旧列表残留），
    // 首轮需以 reset 语义加载第一页（替换旧列表，避免 append 造成重复项）。
    const needFirstPage = !nextToken.value
    try {
      let guard = 0
      // 至少请求一次：nextToken 为空时也加载第一页。
      // 旧实现 `while (nextToken.value && ...)` 在 nextToken 为空时一次都不请求，
      // 却 toast「已加载全部」——实际只显示旧列表（可能为空），属数据完整性缺陷。
      do {
        if (seq !== loadSeq.value) break // 期间发生导航，中止
        guard++
        await load(needFirstPage && guard === 1, seq)
        if (error.value) break // 某页加载失败：停止续页（load 内部已捕获错误并置 error）
      } while (nextToken.value && guard < MAX_ALL_PAGES)
      if (seq !== loadSeq.value) return
      // 任一分页失败时不发「已加载全部」成功提示（error.value 已在 UI 展示）。
      if (error.value) return
      toast(tf('objects.toastLoadedAll', { files: fileObjects.value.length, folders: commonPrefixes.value.length }))
      if (isTruncated.value) toast(tf('objects.toastLoadedCap', { n: guard * PAGE_SIZE }), 'err')
    } finally {
      // load() 内部已捕获列表错误（不经 reject 上抛），此处无需 catch（原 catch 为死代码）。
      if (seq === loadSeq.value) loadingAll.value = false
    }
  }

  /* ---- 工具栏事件转发（供 ObjectToolbar / ObjectList 调用） ---- */
  function onBucketSelect(v: string) {
    currentBucket.value = v
    onBucketChange()
  }

  function backToBuckets() {
    currentBucket.value = ''
    loadBuckets()
  }

  function cancelPathEdit() {
    pathEditing.value = false
  }

  function togglePathEdit() {
    if (pathEditing.value) commitPath()
    else startPathEdit()
  }

  function toggleView() {
    bucketView.value = bucketView.value === 'list' ? 'grid' : 'list'
  }

  function loadMore() {
    load(false)
  }

  watch(() => state.currentAccountId, async () => {
    if (!panelActive.value) return
    await onAccountSwitch()
  })

  onActivated(async () => {
    panelActive.value = true
    if (state.currentAccountId !== syncedAccountId) await onAccountSwitch()
  })

  onDeactivated(() => {
    panelActive.value = false
  })

  onMounted(async () => {
    panelActive.value = true
    syncedAccountId = state.currentAccountId
    window.addEventListener('click', closeCtx)
    window.addEventListener('keydown', onGlobalKey)
    window.addEventListener('blur', closeCtx)
    window.addEventListener('scroll', closeCtx, true)
    if (account.value) {
      await loadBuckets()
      await load(true)
    }
  })

  onBeforeUnmount(() => {
    window.removeEventListener('click', closeCtx)
    window.removeEventListener('keydown', onGlobalKey)
    window.removeEventListener('blur', closeCtx)
    window.removeEventListener('scroll', closeCtx, true)
    // 卸载时取消仍在飞的列表请求，避免组件销毁后 fetch 回调修改已释放的 ref。
    if (loadCtrl) loadCtrl.abort()
  })

  return {
    prefix,
    currentBucket,
    buckets,
    objects,
    commonPrefixes,
    nextToken,
    isTruncated,
    loading,
    loadingBuckets,
    error,
    selected,
    filter,
    sortKey,
    sortDir,
    toggleSort,
    ctxMenu,
    openCtx,
    openCtxFromButton,
    closeCtx,
    panelActive,
    account,
    fileObjects,
    allSelected,
    entries,
    visibleEntries,
    filterActive,
    crumbs,
    relName,
    loadBuckets,
    bucketView,
    creatingBucket,
    enterBucket,
    openCreateBucket,
    onCreateBucket,
    removeBucket,
    loadedSize,
    selectedSize,
    load,
    onBucketChange,
    enterPrefix,
    goRoot,
    goUp,
    toggle,
    onRowClick,
    selectAll,
    onGlobalKey,
    accSel,
    hintsHidden,
    hideHints,
    opsBusy,
    pathEditing,
    pathDraft,
    startPathEdit,
    commitPath,
    fileList,
    lastSelIdx,
    toggleWithShift,
    onRowDblClick,
    refreshAll,
    loadingAll,
    listGen,
    loadAll,
    onBucketSelect,
    backToBuckets,
    cancelPathEdit,
    togglePathEdit,
    toggleView,
    loadMore,
  }
}
