<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { toErrorMessage } from '../errors'

import { state, selectAccount } from '../store'
import { resolveAccountSelect } from '../composables/useAccountSelect'
import { s3api } from '../api'
import { toast } from '../store'
import { confirmDialog } from '../confirm'
import { fmtDate } from '../format'
import { DEFAULT_VIEWPORT_H, OVERSCAN, ROW_HEIGHT, virtualWindow } from '../virtualList'
import { t, tf } from '../i18n'
import type { BucketItem } from '../types'

interface TrashMarker {
  key: string
  versionId: string
  isLatest: boolean
  lastModified: string
}

const accSel = ref('')
const bucketSel = ref('')
const buckets = ref<BucketItem[]>([])
const markers = ref<TrashMarker[]>([])
const loading = ref(false)
const loadingMore = ref(false)
const loadingBuckets = ref(false)
const error = ref('')
const isTruncated = ref(false)
const nextKeyMarker = ref('')
const nextVersionIdMarker = ref('')
const busy = ref(false)

const account = () => state.accounts.find((a) => a.id === accSel.value)

/* markers 大列表窗口化：仅渲染可视区 + overscan，避免上万条删除标记冻结 DOM。
   行高单一来源：ROW_HEIGHT 直接绑定到 v-row 行内样式（:style），CSS 不再另存字面量——
   此前 CSS 38px 与 ROW_HEIGHT=42 漂移导致滚动窗口错位（review §F3，同类 bug 见 §F9③）。 */
const scrollEl = ref<HTMLElement | null>(null)
const scrollTop = ref(0)
const viewportH = ref(DEFAULT_VIEWPORT_H)

const windowed = computed(() => {
  const win = virtualWindow(markers.value.length, scrollTop.value, viewportH.value, ROW_HEIGHT, OVERSCAN)
  return { ...win, items: markers.value.slice(win.start, win.end) }
})

function onListScroll() {
  if (scrollEl.value) scrollTop.value = scrollEl.value.scrollTop
}

function measureViewport() {
  if (scrollEl.value) viewportH.value = scrollEl.value.clientHeight || DEFAULT_VIEWPORT_H
}

let resizeObs: ResizeObserver | undefined

// markers 为空时容器不渲染（首屏/空桶），scrollEl 的 ref 绑定晚于 onMounted：
// 用 watch 监听 ref 绑定时机，自动测量可视区并注册 ResizeObserver（同 ObjectList/MigratePanel）。
watch(scrollEl, (el) => {
  resizeObs?.disconnect()
  resizeObs = undefined
  if (!el || typeof ResizeObserver === 'undefined') return
  measureViewport()
  resizeObs = new ResizeObserver(measureViewport)
  resizeObs.observe(el)
})

/* 数据源整体更换（切桶/切账号/刷新/恢复与清除后的重新赋值）必须把窗口起点归零，
   否则残留的旧 scrollTop 会让 markers.slice(start, end) 为空 → 空白表（review §F2）。
   只监听数组身份、不做深监听：「加载更多」是原数组 push 追加，不重置用户滚动位置。
   同步写回真实 DOM scrollTop，避免下一次滚动事件把陈旧偏移写回。 */
function resetWindowScroll() {
  scrollTop.value = 0
  if (scrollEl.value) scrollEl.value.scrollTop = 0
}
watch(() => markers.value, () => resetWindowScroll())

onBeforeUnmount(() => resizeObs?.disconnect())

// 桶加载代际（评审 R6）：切账号会连续发起两次 listBuckets，乱序返回时旧响应会把
// 上一个账号的桶选择写进当前账号——写入 / 清错 / 清 loading 都须先验 seq
//（口径同 useObjectBrowser.loadBuckets）。loadingBuckets 配 finally 复位：
// 异常时不得永久为 true（否则桶选择器永久禁用）。
let bucketSeq = 0

async function loadBuckets() {
  if (!accSel.value) {
    buckets.value = []
    return
  }
  const seq = ++bucketSeq
  loadingBuckets.value = true
  try {
    const r = await s3api.listBuckets(accSel.value)
    if (seq !== bucketSeq) return // 过期响应：当前账号的加载已接管状态，静默丢弃
    buckets.value = r.buckets
    if (!bucketSel.value || !r.buckets.some((b) => b.name === bucketSel.value)) bucketSel.value = r.buckets[0]?.name ?? ''
    // 成功即清：横幅不得永久遮蔽整页（v-if 三分支互斥，error 一旦置位列表就再也出不来）。
    error.value = ''
  } catch (e) {
    if (seq !== bucketSeq) return // 过期失败：不污染当前账号的横幅
    error.value = toErrorMessage(e)
  } finally {
    if (seq === bucketSeq) loadingBuckets.value = false // 过期 finally 不得抢清在途请求的 loading
  }
}

/** 重试：先补桶选择再重拉标记——loadMarkers 在无 bucketSel 时直接返回，
 *  只重拉标记会让「loadBuckets 失败」那次的横幅永久卡死、且无任何反馈。 */
async function retry() {
  await loadBuckets()
  await loadMarkers(true)
}

// 加载序号：每次 loadMarkers +1；过期循环在首个 await 后静默终止，
// 避免「快速切桶时两个并发循环互相 push 覆盖 marker」的竞态（仿 useObjectBrowser 的 loadSeq）。
let loadSeq = 0

async function loadMarkers(reset: boolean) {
  if (!accSel.value || !bucketSel.value) return
  const seq = ++loadSeq
  if (reset) {
    markers.value = []
    nextKeyMarker.value = ''
    nextVersionIdMarker.value = ''
    isTruncated.value = false
  }
  loading.value = reset
  loadingMore.value = !reset
  try {
    // 空页自动向后翻少量页（版本列表可能夹杂非删除标记）；避免一次拉满 51 页撑爆 DOM
    let guard = 0
    const emptyPageSkip = reset ? 2 : 3
    for (;;) {
      const r = await s3api.listTrash(accSel.value, {
        bucket: bucketSel.value,
        keyMarker: nextKeyMarker.value,
        versionIdMarker: nextVersionIdMarker.value,
        maxKeys: 1000,
      })
      if (seq !== loadSeq) return // 过期请求：新加载已接管 markers / 游标，静默退出
      markers.value.push(...r.deleteMarkers)
      nextKeyMarker.value = r.nextKeyMarker
      nextVersionIdMarker.value = r.nextVersionIdMarker
      isTruncated.value = r.isTruncated
      if (r.deleteMarkers.length || !r.isTruncated || guard++ >= emptyPageSkip) break
    }
    // 加载成功即清：横幅不得永久遮蔽整页（v-if / v-else-if 三分支互斥）。
    error.value = ''
  } catch (e) {
    if (seq === loadSeq) error.value = toErrorMessage(e)
  } finally {
    if (seq === loadSeq) {
      loading.value = false
      loadingMore.value = false
    }
  }
}

onMounted(() => {
  accSel.value = resolveAccountSelect()
  selectAccount(accSel.value)
  loadBuckets()
})

watch(accSel, (v) => {
  selectAccount(v)
  bucketSel.value = ''
  markers.value = []
  loadBuckets()
})

watch(bucketSel, (b) => {
  if (b) loadMarkers(true)
  else markers.value = []
})

async function restore(m: TrashMarker) {
  if (busy.value) return
  const ok = await confirmDialog({
    title: t('trash.restoreTitle'),
    message: tf('trash.restoreConfirm', { key: m.key }),
    confirmText: t('trash.restore'),
    danger: false,
  })
  if (!ok) return
  busy.value = true
  try {
    await s3api.restoreDeleteMarker(accSel.value, { bucket: bucketSel.value, key: m.key, versionId: m.versionId })
    toast(tf('trash.restored', { key: m.key }))
    markers.value = markers.value.filter((x) => !(x.key === m.key && x.versionId === m.versionId))
  } catch (e) {
    error.value = toErrorMessage(e)
  } finally {
    busy.value = false
  }
}

async function purge(m: TrashMarker) {
  if (busy.value) return
  const ok = await confirmDialog({
    title: t('trash.purgeTitle'),
    message: tf('trash.purgeConfirm', { key: m.key }),
    confirmText: t('trash.purge'),
    danger: true,
  })
  if (!ok) return
  busy.value = true
  try {
    const r = await s3api.purgeTrashObject(accSel.value, { bucket: bucketSel.value, key: m.key })
    toast(tf('trash.purged', { key: m.key, n: r.deleted }))
    markers.value = markers.value.filter((x) => x.key !== m.key)
  } catch (e) {
    error.value = toErrorMessage(e)
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="panel">
    <div class="toolbar">
      <h3 style="margin:0">{{ t('trash.title') }}</h3>
      <span class="spacer" />
      <label class="badge" for="trash-acc-select">{{ t('trash.account') }}</label>
      <select id="trash-acc-select" v-model="accSel" class="acc-select" :title="tf('trash.switchAccount', { n: state.accounts.length })">
        <option v-if="!state.accounts.length" value="">{{ t('trash.noAccounts') }}</option>
        <option v-for="a in state.accounts" :key="a.id" :value="a.id">{{ a.name }}</option>
      </select>
      <label class="badge" for="trash-bucket-select">{{ t('trash.bucket') }}</label>
      <select id="trash-bucket-select" v-model="bucketSel" class="acc-select" :disabled="loadingBuckets" :title="t('trash.switchBucket')">
        <option v-if="!buckets.length" value="">{{ t('trash.noBuckets') }}</option>
        <option v-for="b in buckets" :key="b.name" :value="b.name">{{ b.name }}</option>
      </select>
      <button class="btn secondary sm" :disabled="!bucketSel || loading" @click="loadMarkers(true)">{{ t('common.refresh') }}</button>
    </div>

    <div v-if="!account()" class="empty">
      <span class="empty-icon" aria-hidden="true">🗑️</span>
      {{ t('trash.needAccount') }}
    </div>

    <div v-else-if="error" class="msg err" role="alert" style="margin-bottom:10px">
      <span style="flex:1">{{ error }}</span>
      <button class="link" style="flex:none" @click="retry">{{ t('common.retry') }}</button>
    </div>

    <template v-else-if="bucketSel">
      <div v-if="loading" class="empty" style="padding:20px">{{ t('trash.loading') }}</div>
      <div v-else-if="!markers.length" class="empty">
        <span class="empty-icon" aria-hidden="true">🗑️</span>
        {{ t('trash.emptyHint') }}
      </div>
      <div v-else ref="scrollEl" class="tbl-wrap tbl-virtual" @scroll.passive="onListScroll">
        <table class="tbl">
          <caption class="sr-only">{{ t('trash.tableAria') }}</caption>
          <thead><tr><th>{{ t('trash.colKey') }}</th><th style="width:120px">{{ t('trash.colVersion') }}</th><th style="width:160px">{{ t('trash.colDeletedAt') }}</th><th style="width:180px; text-align:right">{{ t('trash.colActions') }}</th></tr></thead>
          <tbody>
            <tr v-if="windowed.padTop" class="v-spacer" aria-hidden="true">
              <td :colspan="4" :style="{ height: windowed.padTop + 'px' }" />
            </tr>
            <tr v-for="m in windowed.items" :key="m.key + ':' + m.versionId" class="v-row" :style="{ height: `${ROW_HEIGHT}px` }">
              <td class="mono" style="word-break:break-all">{{ m.key }}</td>
              <td class="mono" style="word-break:break-all">{{ m.versionId }}</td>
              <td class="muted">{{ fmtDate(m.lastModified) }}</td>
              <td>
                <div class="actions" style="justify-content:flex-end; gap:6px">
                  <button class="btn secondary sm" :disabled="busy" style="color:var(--primary)" @click="restore(m)">{{ t('trash.restore') }}</button>
                  <button class="btn danger sm" :disabled="busy" @click="purge(m)">{{ t('trash.purge') }}</button>
                </div>
              </td>
            </tr>
            <tr v-if="windowed.padBottom" class="v-spacer" aria-hidden="true">
              <td :colspan="4" :style="{ height: windowed.padBottom + 'px' }" />
            </tr>
          </tbody>
        </table>
      </div>
      <div class="toolbar" style="margin-top:12px; margin-bottom:0">
        <button class="btn secondary sm" :disabled="!isTruncated || loadingMore" @click="loadMarkers(false)">
          {{ loadingMore ? t('common.loading') : t('common.more') }}
        </button>
        <span class="badge">{{ isTruncated ? t('trash.hasMore') : t('trash.endReached') }}</span>
        <span class="badge" style="margin-left:auto">{{ tf('trash.shown', { n: markers.length }) }}</span>
      </div>
    </template>

    <div v-else class="empty">
      <span class="empty-icon" aria-hidden="true">🪣</span>
      {{ t('trash.pickBucket') }}
    </div>
  </div>
</template>

<style scoped>
.acc-select { max-width: 220px; padding: 5px 10px; font-size: 13px; }
.actions { display: flex; }

/* 虚拟滚动：固定可视高度，行高由 v-row 行内 ROW_HEIGHT 绑定（不在此存字面量）。 */
.tbl-virtual {
  max-height: min(60vh, 640px);
  overflow: auto;
}
.tbl-virtual .v-spacer td { padding: 0; border: 0; }
</style>
