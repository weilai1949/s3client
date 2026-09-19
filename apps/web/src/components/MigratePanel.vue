<script setup lang="ts">
defineOptions({ name: 'MigratePanel' })

import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { toErrorMessage } from '../errors'

import { s3api, subscribeMigrateEvents } from '../api'
import { state, currentAccount, toast, selectAccount, requestTab } from '../store'
import { fmtSize } from '../format'
import { MIGRATE_MAX_KEYS_PER_REQUEST, batchKeys } from '../limits'
import { DEFAULT_VIEWPORT_H, OVERSCAN, ROW_HEIGHT, virtualWindow } from '../virtualList'
import { t, tf } from '../i18n'
import ModalDialog from './ModalDialog.vue'
import type { BucketItem, JobRecord, ObjectItem } from '../types'

// 大对象列表（listAll 上限 200×1000）走窗口化渲染，避免数十万行直接 v-for 冻结页面。
// 行高常量与模板 CSS（.tbl-virtual .v-row { height: 42px }）共用同一来源，避免漂移。
const scrollEl = ref<HTMLElement | null>(null)
const scrollTop = ref(0)
const viewportH = ref(DEFAULT_VIEWPORT_H)
const windowed = computed(() => {
  const win = virtualWindow(objects.value.length, scrollTop.value, viewportH.value, ROW_HEIGHT, OVERSCAN)
  return { ...win, items: objects.value.slice(win.start, win.end) }
})
function onListScroll() {
  if (scrollEl.value) scrollTop.value = scrollEl.value.scrollTop
}
function measureViewport() {
  if (scrollEl.value) viewportH.value = scrollEl.value.clientHeight || DEFAULT_VIEWPORT_H
}
let resizeObs: ResizeObserver | undefined
// 虚拟列表在组件挂载后才随对象数据渲染，scrollEl 的 ref 绑定晚于 onMounted：
// 用 watch 监听 ref 绑定时机，自动测量可视区并注册 ResizeObserver（修复原 onMounted 恒空失效）。
watch(scrollEl, (el) => {
  resizeObs?.disconnect()
  resizeObs = undefined
  if (!el || typeof ResizeObserver === 'undefined') return
  measureViewport()
  resizeObs = new ResizeObserver(measureViewport)
  resizeObs.observe(el)
})

// 窗口起点必须随 objects 变化重置：重新列出/切换前缀后旧的 scrollTop 会让
// objects.slice(start, end) 为空 → 空白表（review §F2）。同步写回真实 DOM scrollTop
// （浏览器在内容缩短时也会钳制，此处显式归零避免依赖钳制时机）。
function resetWindowScroll() {
  scrollTop.value = 0
  if (scrollEl.value) scrollEl.value.scrollTop = 0
}

onBeforeUnmount(() => {
  resizeObs?.disconnect()
  // 组件卸载时若仍有进行中的 SSE 订阅，立即断开（避免后台 goroutine 持续推事件）。
  if (activeUnsub) activeUnsub()
  if (activeJobId.value) {
    // 后端 job 不主动取消（用户离开后任务可能仍在 server 端进行；
    // 短期同步进度由 JobRegistry reap 处理）。
  }
})

const sourceBucket = ref('')
const sourcePrefix = ref('')
const targetAccountId = ref('')
const targetBucket = ref('')
const targetPrefix = ref('')
const objects = ref<ObjectItem[]>([])
const selected = ref<Set<string>>(new Set())
const loading = ref(false)
const busy = ref(false)
const error = ref('')
const sourceBuckets = ref<BucketItem[]>([])
const targetBuckets = ref<BucketItem[]>([])
const loadingBuckets = ref(false)

// 见上：objects 变化（重新列出/切换前缀）后虚拟窗口必须回到顶部。
watch(objects, resetWindowScroll)

/* ---- 迁移进度 ---- */
const progress = reactive({ done: 0, total: 0 })
/** 进度百分比。
 *
 * 不变量：`migrate()` 在等待事件流之前就把 progress.total 设为本次选中数，
 * 且进度条仅在 busy 时渲染，因此渲染期 total 恒 > 0。
 */
const progressPct = computed(() => Math.round((progress.done / progress.total) * 100))
const activeJobId = ref('')
// 组件级 SSE 取消器：保证 onBeforeUnmount 一定能断开正在进行的迁移事件流。
let activeUnsub: (() => void) | undefined
const cancelling = ref(false)

/* ---- 未完成任务（跨重启恢复）---- */
// 服务端持久化任务清单后，进程重启会把运行中的任务标记为 interrupted。
// 这类任务（尤其移动语义「复制成功但源未删除」）必须让用户看到并对账，
// 否则重启即静默丢失：源对象可能被重复迁移或遗留。
const unfinished = ref<JobRecord[]>([])
const unfinishedLoading = ref(false)
const unfinishedError = ref('')

async function loadUnfinishedJobs() {
  unfinishedLoading.value = true
  unfinishedError.value = ''
  try {
    const { jobs } = await s3api.migrateJobs()
    // 只展示需要人工介入的：interrupted（重启中断）与 running（仍在进行）。
    unfinished.value = (jobs ?? []).filter((j) => j.status === 'interrupted' || j.status === 'running')
  } catch (e) {
    unfinishedError.value = toErrorMessage(e)
  } finally {
    unfinishedLoading.value = false
  }
}

/** 忽略一条中断记录：仅从本地列表移除（服务端记录仍按 TTL 保留，供审计）。 */
function dismissUnfinished(id: string) {
  unfinished.value = unfinished.value.filter((j) => j.id !== id)
}

/* ---- 迁移结果弹窗 ---- */
const resultDialog = reactive({
  open: false,
  migrated: 0,
  failed: 0,
  failedKeys: [] as string[],
  lastError: '',
})
const resultTargetId = ref('')

const sourceAccount = computed(() => currentAccount())
const targetAccount = computed(() => state.accounts.find((a) => a.id === targetAccountId.value))
const selectedSize = computed(() => objects.value.filter((o) => selected.value.has(o.key)).reduce((s, o) => s + o.size, 0))

async function loadSourceBuckets() {
  const acc = sourceAccount.value
  if (!acc) return
  loadingBuckets.value = true
  try {
    const res = await s3api.listBuckets(acc.id)
    sourceBuckets.value = res.buckets ?? []
  } catch {
    sourceBuckets.value = []
  } finally {
    loadingBuckets.value = false
  }
}

async function loadTargetBuckets() {
  if (!targetAccountId.value) {
    targetBuckets.value = []
    return
  }
  loadingBuckets.value = true
  try {
    const res = await s3api.listBuckets(targetAccountId.value)
    targetBuckets.value = res.buckets ?? []
  } catch {
    targetBuckets.value = []
  } finally {
    loadingBuckets.value = false
  }
}

/** 目标账号默认值：优先另一个账号，其次源账号（同账号迁移=复制到其他桶/前缀）。 */
function ensureTargetAccount() {
  if (targetAccountId.value) return
  if (!state.accounts.length) return
  const srcId = sourceAccount.value?.id
  const other = state.accounts.find((a) => a.id !== srcId)
  targetAccountId.value = other?.id ?? srcId ?? state.accounts[0].id
}

/** 列出前缀下全部文件（含子目录）：循环分页，delimiter 置空不分目录。 */
const loadingAll = ref(false)
const MAX_ALL_PAGES = 200 // 单页 1000 → 最多 20 万个对象

/** 列举代次：每次列举（本函数 / loadAllSourceObjects）自增，用于丢弃过期响应
 *  （否则先后两次列举的结果会互相覆盖）。非响应式：只作比较，不驱动渲染。 */
let listGen = 0

async function loadSourceObjects() {
  const acc = sourceAccount.value
  if (!acc) return
  const gen = ++listGen
  loading.value = true
  error.value = ''
  const bucket = sourceBucket.value
  const prefix = sourcePrefix.value
  try {
    const res = await s3api.listObjects(acc.id, { bucket, prefix, delimiter: '/', maxKeys: '200' })
    if (gen !== listGen) return
    objects.value = res.objects.filter((o) => !o.isDir)
    selected.value = new Set()
  } catch (e) {
    if (gen !== listGen) return
    error.value = toErrorMessage(e)
  } finally {
    if (gen === listGen) loading.value = false
  }
}

async function loadAllSourceObjects() {
  const acc = sourceAccount.value
  if (!acc) return
  const gen = ++listGen
  // 快照：分页循环内不得读实时 sourceBucket/sourcePrefix——用户在续页之间改前缀会把
  // 旧前缀的 continuationToken 带到新前缀上，列表混两个前缀（review §F5）。
  const bucket = sourceBucket.value
  const prefix = sourcePrefix.value
  loadingAll.value = true
  error.value = ''
  try {
    const all: ObjectItem[] = []
    let token = ''
    let guard = 0
    for (;;) {
      const q: Record<string, string> = { bucket, prefix, maxKeys: '1000' }
      if (token) q.continuationToken = token
      const res = await s3api.listObjects(acc.id, q)
      // 代次守卫：期间用户重新列出（或切换账号/前缀）→ 丢弃本次全部结果。
      if (gen !== listGen) return
      all.push(...res.objects.filter((o) => !o.isDir))
      if (!res.isTruncated || !res.nextToken) break
      if (++guard >= MAX_ALL_PAGES) {
        toast(tf('migrate.listedCap', { n: all.length }), 'err')
        break
      }
      token = res.nextToken
    }
    objects.value = all
    selected.value = new Set()
    toast(tf('migrate.listedAll', { n: all.length }))
  } catch (e) {
    if (gen !== listGen) return
    error.value = toErrorMessage(e)
  } finally {
    if (gen === listGen) loadingAll.value = false
  }
}

function toggle(k: string) {
  const s = new Set(selected.value)
  if (s.has(k)) s.delete(k)
  else s.add(k)
  selected.value = s
}

function selectAll() {
  if (selected.value.size === objects.value.length) selected.value = new Set()
  else selected.value = new Set(objects.value.map((o) => o.key))
}

async function migrate() {
  const src = sourceAccount.value
  if (!src) return
  if (selected.value.size === 0) return
  if (!targetAccountId.value) {
    error.value = t('migrate.selectTarget')
    return
  }
  busy.value = true
  error.value = ''
  progress.done = 0
  progress.total = selected.value.size
  const keys = [...selected.value]
  // 服务端单次请求最多 10000 个 key（handler/migrate.go），选中量可远超（loadAll 20 万）：
  // 不分片会整批 400、一个都不迁移（review §F3）。分片串行提交并聚合结果。
  const chunks = batchKeys(keys, MIGRATE_MAX_KEYS_PER_REQUEST)
  let unsub: (() => void) | undefined
  let finalStatus = ''
  let migrated = 0
  let failed = 0
  let lastError = ''
  const failedKeys: string[] = []
  try {
    for (let i = 0; i < chunks.length; i++) {
      const doneBase = progress.done
      const chunk = chunks[i]
      const { jobId } = await s3api.migrateAsync({
        sourceAccountId: src.id,
        sourceBucket: sourceBucket.value || undefined,
        sourceKeys: chunk,
        targetAccountId: targetAccountId.value,
        targetBucket: targetBucket.value || undefined,
        targetPrefix: targetPrefix.value,
      })
      activeJobId.value = jobId
      activeUnsub = undefined
      finalStatus = ''
      await new Promise<void>((resolve, reject) => {
        unsub = subscribeMigrateEvents(
          jobId,
          (p) => {
            // 多分片：进度按已完成分片数累加，进度条不因切换 job 而回退。
            progress.done = doneBase + p.done
            progress.total = keys.length
            if (p.status === 'done' || p.status === 'cancelled') {
              finalStatus = p.status
              resolve()
            }
          },
          reject,
        )
        activeUnsub = unsub
      })
      unsub?.()
      unsub = undefined
      activeUnsub = undefined
      const st = await s3api.migrateJobStatus(jobId)
      const r = st.result ?? { migrated: 0, failed: 0 }
      migrated += r.migrated
      failed += r.failed
      if (!lastError && r.lastError) lastError = r.lastError
      for (const k of r.failedKeys ?? []) {
        if (failedKeys.length < 200) failedKeys.push(k)
      }
      if (finalStatus === 'cancelled' || st.progress.status === 'cancelled') {
        finalStatus = 'cancelled'
        break
      }
    }
    resultTargetId.value = targetAccountId.value
    resultDialog.open = true
    resultDialog.migrated = migrated
    resultDialog.failed = failed
    resultDialog.failedKeys = failedKeys
    resultDialog.lastError = lastError
    if (finalStatus === 'cancelled') {
      toast(tf('migrate.toastCancelled', { ok: migrated, fail: failed }), 'err')
    } else if (failed) {
      toast(tf('migrate.toastPartial', { ok: migrated, fail: failed }), 'err')
    } else {
      toast(tf('migrate.toastOk', { n: migrated }))
    }
  } catch (e) {
    // 分片失败：不吞掉已完成分片的结果，让用户看到真实的「已迁移 N / 失败 M」。
    if (migrated || failed) {
      resultTargetId.value = targetAccountId.value
      resultDialog.open = true
      resultDialog.migrated = migrated
      resultDialog.failed = failed
      resultDialog.failedKeys = failedKeys
      resultDialog.lastError = lastError || toErrorMessage(e)
      toast(tf('migrate.toastPartial', { ok: migrated, fail: failed }), 'err')
    }
    error.value = toErrorMessage(e)
  } finally {
    unsub?.()
    activeUnsub = undefined
    activeJobId.value = ''
    cancelling.value = false
    busy.value = false
  }
}

async function cancelMigrate() {
  if (!activeJobId.value || cancelling.value) return
  cancelling.value = true
  try {
    await s3api.migrateJobCancel(activeJobId.value)
    toast(t('migrate.cancelRequested'))
  } catch (e) {
    error.value = toErrorMessage(e)
    cancelling.value = false
  }
}

/** 迁移完成 → 去目标账号对象管理查看。 */
function gotoTargetObjects() {
  const target = state.accounts.find((a) => a.id === resultTargetId.value)
  if (target) selectAccount(target.id)
  resultDialog.open = false
  requestTab('objects')
}

watch(() => state.currentAccountId, () => {
  sourceBucket.value = ''
  targetAccountId.value = ''
  objects.value = []
  selected.value = new Set()
  ensureTargetAccount()
  // onMounted 已 await 过这两个加载，这里无需再等（watcher 里的 Promise 无人消费）。
  void loadSourceBuckets()
  void loadTargetBuckets()
  loadSourceObjects()
})

watch(targetAccountId, () => {
  targetBucket.value = ''
  loadTargetBuckets()
})

onMounted(async () => {
  ensureTargetAccount()
  loadUnfinishedJobs()
  await loadSourceBuckets()
  await loadTargetBuckets()
  loadSourceObjects()
})
</script>

<template>
  <div class="panel">
    <div class="toolbar">
      <h3 style="margin:0">{{ t('migrate.title') }}</h3>
      <span class="spacer" />
      <span class="badge">{{ tf('migrate.sourceAccount', { name: sourceAccount?.name || t('common.noAccount') }) }}</span>
    </div>

    <!-- 未完成任务（跨重启恢复）：仅在有 interrupted/running 任务时出现 -->
    <div v-if="unfinished.length || unfinishedLoading || unfinishedError" class="unfinished">
      <div class="unfinished-head">
        <span class="tag bad">{{ t('migrate.statusInterrupted') }}</span>
        <strong>{{ t('migrate.unfinishedTitle') }}</strong>
        <span class="badge">{{ tf('migrate.unfinishedCount', { n: unfinished.length }) }}</span>
        <span class="spacer" />
        <button class="btn secondary sm" :disabled="unfinishedLoading" @click="loadUnfinishedJobs">
          {{ unfinishedLoading ? t('common.working') : t('common.refresh') }}
        </button>
      </div>
      <p class="badge" style="margin:6px 0">{{ t('migrate.unfinishedHint') }}</p>
      <div v-if="unfinishedError" class="badge" style="color:var(--danger)">{{ unfinishedError }}</div>
      <div v-for="j in unfinished" :key="j.id" class="unfinished-item">
        <span class="mono">{{ j.id }}</span>
        <span class="tag" :class="j.status === 'interrupted' ? 'bad' : 'ok'">
          {{ j.status === 'interrupted' ? t('migrate.statusInterrupted') : t('migrate.statusRunning') }}
        </span>
        <span class="badge">{{ tf('migrate.unfinishedProgress', { done: j.progress.done, total: j.total }) }}</span>
        <span v-if="j.result.failed" class="badge" style="color:var(--danger)">{{ tf('migrate.resultFail', { n: j.result.failed }) }}</span>
        <span class="spacer" />
        <button class="btn secondary sm" @click="dismissUnfinished(j.id)">{{ t('migrate.dismiss') }}</button>
      </div>
    </div>

    <div v-if="!sourceAccount" class="empty">
      <span class="empty-icon" aria-hidden="true">🗂️</span>
      {{ t('migrate.needAccount') }}
    </div>

    <template v-else>
      <!-- 源：Bucket / 前缀 / 列出 -->
      <div class="toolbar">
        <label class="field">
          {{ t('migrate.sourceBucket') }}
          <select v-model="sourceBucket" :disabled="loadingBuckets" style="min-width:180px">
            <option value="">{{ tf('migrate.defaultBucket', { name: sourceAccount.bucket || 'default' }) }}</option>
            <option v-for="b in sourceBuckets" :key="b.name" :value="b.name">{{ b.name }}</option>
          </select>
        </label>
        <label class="field">
          {{ t('migrate.sourcePrefix') }}
          <input v-model="sourcePrefix" :placeholder="t('migrate.prefixPlaceholder')" @keyup.enter="loadSourceObjects" />
        </label>
        <button class="btn secondary sm" style="align-self:flex-end" :disabled="loading || busy" @click="loadSourceObjects">
          {{ loading ? t('migrate.listing') : t('migrate.listObjects') }}
        </button>
        <button class="btn secondary sm" style="align-self:flex-end" :disabled="loading || loadingAll || busy" @click="loadAllSourceObjects">
          {{ loadingAll ? t('migrate.collecting') : t('migrate.listAll') }}
        </button>
      </div>

      <!-- 目标：账号 / Bucket / 前缀 -->
      <div class="toolbar" style="margin-bottom:14px">
        <label class="field">
          {{ t('migrate.targetAccount') }}
          <select v-model="targetAccountId" style="min-width:180px">
            <option v-for="a in state.accounts" :key="a.id" :value="a.id">{{ a.name }}{{ a.id === sourceAccount.id ? t('migrate.sameAccount') : '' }}</option>
          </select>
        </label>
        <label class="field">
          {{ t('migrate.targetBucket') }}
          <select v-model="targetBucket" :disabled="!targetAccountId || loadingBuckets" style="min-width:180px">
            <option value="">{{ tf('migrate.defaultBucket', { name: targetAccount?.bucket || 'default' }) }}</option>
            <option v-for="b in targetBuckets" :key="b.name" :value="b.name">{{ b.name }}</option>
          </select>
        </label>
        <label class="field">
          {{ t('migrate.targetPrefix') }}
          <input v-model="targetPrefix" :placeholder="t('migrate.targetPrefixPh')" />
        </label>
      </div>

      <!-- 选择与操作 -->
      <div class="toolbar">
        <label><input type="checkbox" :checked="objects.length > 0 && selected.size === objects.length" @change="selectAll" /> {{ t('common.selectAll') }}</label>
        <span class="badge">{{ tf('toolbar.selectedFiles', { n: selected.size, size: fmtSize(selectedSize) }) }}</span>
        <span class="spacer" />
        <button class="btn sm" :disabled="!selected.size || !targetAccountId || busy" @click="migrate">
          {{ busy ? `${t('migrate.running')} ${progress.done}/${progress.total}` : t('migrate.start') }}
        </button>
        <button
          v-if="busy && activeJobId"
          class="btn secondary sm"
          :disabled="cancelling"
          @click="cancelMigrate"
        >
          {{ cancelling ? t('migrate.cancelling') : t('migrate.cancel') }}
        </button>
      </div>

      <!-- 迁移进度条 -->
      <div v-if="busy" class="progress" style="margin:10px 0" role="progressbar" :aria-valuenow="progressPct" aria-valuemin="0" aria-valuemax="100">
        <div class="bar" :style="{ width: progressPct + '%' }" />
      </div>

      <div v-if="error" class="msg err" style="margin:10px 0">{{ error }}</div>

      <div v-if="loading" aria-busy="true" :aria-label="t('migrate.listingAria')">
        <div v-for="i in 4" :key="i" class="skel-row" />
      </div>
      <div v-else-if="objects.length" ref="scrollEl" class="tbl-wrap tbl-virtual" @scroll.passive="onListScroll">
        <table class="tbl">
          <thead><tr><th style="width:30px"></th><th>Key</th><th style="width:100px">{{ t('common.size') }}</th></tr></thead>
          <tbody>
            <tr v-if="windowed.padTop" class="v-spacer" aria-hidden="true">
              <td :colspan="3" :style="{ height: windowed.padTop + 'px' }" />
            </tr>
            <tr v-for="o in windowed.items" :key="o.key" class="v-row" :class="{ selected: selected.has(o.key) }">
              <td><input type="checkbox" :aria-label="tf('objects.selectItem', { name: o.key })" :checked="selected.has(o.key)" @change="toggle(o.key)" /></td>
              <td class="mono">{{ o.key }}</td>
              <td class="muted">{{ fmtSize(o.size) }}</td>
            </tr>
            <tr v-if="windowed.padBottom" class="v-spacer" aria-hidden="true">
              <td :colspan="3" :style="{ height: windowed.padBottom + 'px' }" />
            </tr>
          </tbody>
        </table>
      </div>
      <div v-else class="empty">
        <span class="empty-icon" aria-hidden="true">⇄</span>
        {{ t('migrate.emptyHint') }}
      </div>
    </template>

    <!-- 迁移结果弹窗 -->
    <ModalDialog :open="resultDialog.open" :title="t('migrate.resultTitle')" width="min(560px, 100%)" @close="resultDialog.open = false">
      <div class="row" style="margin-bottom:12px">
        <span class="tag ok">{{ tf('migrate.resultOk', { n: resultDialog.migrated }) }}</span>
        <span class="tag" :class="resultDialog.failed ? 'bad' : 'ok'">{{ tf('migrate.resultFail', { n: resultDialog.failed }) }}</span>
        <span class="badge">{{ tf('migrate.resultTotal', { n: resultDialog.migrated + resultDialog.failed }) }}</span>
      </div>
      <div v-if="resultDialog.failed" class="result-fail">
        <div class="badge" style="margin-bottom:6px">{{ t('migrate.failList') }}</div>
        <div class="fail-list">
          <div v-for="k in resultDialog.failedKeys" :key="k" class="mono fail-item">{{ k }}</div>
        </div>
        <div v-if="resultDialog.lastError" class="badge" style="margin-top:6px; color:var(--danger)">{{ tf('migrate.firstError', { msg: resultDialog.lastError }) }}</div>
      </div>
      <div class="row" style="margin-top:16px">
        <button class="btn sm" @click="gotoTargetObjects">{{ t('migrate.gotoTarget') }}</button>
        <button class="btn secondary sm" @click="resultDialog.open = false">{{ t('common.close') }}</button>
      </div>
    </ModalDialog>
  </div>
</template>

<style scoped>
.result-fail { margin-top: 4px; }
.fail-list {
  max-height: 180px; overflow: auto;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--panel-2);
  padding: 8px 10px;
}
.fail-item { font-size: 12px; padding: 2px 0; }

/* 虚拟滚动：固定行高 + 上下垫片让滚动条反映真实总高度。 */
.tbl-virtual {
  max-height: 60vh;
  overflow: auto;
}
.tbl-virtual .v-row { height: 38px; }
.tbl-virtual .v-spacer td { padding: 0; border: 0; }
</style>
