<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { toErrorMessage } from '../errors'

import { s3api } from '../api'
import { toast } from '../store'
import { confirmDialog } from '../confirm'
import { fmtDate, fmtSize } from '../format'
import { DEFAULT_VIEWPORT_H, OVERSCAN, ROW_HEIGHT, virtualWindow } from '../virtualList'
import { t, tf } from '../i18n'
import ModalDialog from './ModalDialog.vue'
import CompareDialog from './CompareDialog.vue'
import type { CompareVersion } from './CompareDialog.vue'

interface VersionRow {
  key: string
  versionId: string
  isLatest: boolean
  lastModified: string
  size: number
  etag: string
  storageClass: string
  isDeleteMarker: boolean
}

const props = defineProps<{
  open: boolean
  accountId: string
  bucket: string
  objectKey: string
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'error', msg: string): void
}>()

const rows = ref<VersionRow[]>([])
const loading = ref(false)
const busy = ref(false)
const compareOpen = ref(false)
/** 达到分页上限后服务端仍有更多版本（此时不再静默丢弃，UI 明确提示）。 */
const truncated = ref(false)
const truncatedPages = ref(0)

/* 版本大列表窗口化：仅渲染可视区 + overscan，避免上万条版本冻结弹窗（review Nit：大表无虚拟滚动）。
   行高单一来源：ROW_HEIGHT 直接绑定到 v-row 行内样式（:style），CSS 不再另存字面量——
   此前 CSS 38px 与 ROW_HEIGHT=42 漂移导致滚动窗口错位（review §F3，同类 bug 见 §F9③）。 */
const scrollEl = ref<HTMLElement | null>(null)
const scrollTop = ref(0)
const viewportH = ref(DEFAULT_VIEWPORT_H)

const windowed = computed(() => {
  const win = virtualWindow(rows.value.length, scrollTop.value, viewportH.value, ROW_HEIGHT, OVERSCAN)
  return { ...win, items: rows.value.slice(win.start, win.end) }
})

function onListScroll() {
  if (scrollEl.value) scrollTop.value = scrollEl.value.scrollTop
}

function measureViewport() {
  if (scrollEl.value) viewportH.value = scrollEl.value.clientHeight || DEFAULT_VIEWPORT_H
}

let resizeObs: ResizeObserver | undefined

// 弹窗内容随 open/loading 才渲染，scrollEl 的 ref 绑定晚于 onMounted：
// 用 watch 监听 ref 绑定时机，自动测量可视区并注册 ResizeObserver（同 ObjectList/MigratePanel）。
watch(scrollEl, (el) => {
  resizeObs?.disconnect()
  resizeObs = undefined
  if (!el || typeof ResizeObserver === 'undefined') return
  measureViewport()
  resizeObs = new ResizeObserver(measureViewport)
  resizeObs.observe(el)
})

/* 数据整体更换（重开弹窗/重新加载/删除恢复后 rows 被整体重新赋值）必须把窗口起点归零，
   否则残留的旧 scrollTop 会让 rows.slice(start, end) 为空 → 空白表（review §F2）。
   只监听数组身份：rows 每次 load 都是整体替换，不存在 push 追加场景。
   同步写回真实 DOM scrollTop，避免下一次滚动事件把陈旧偏移写回。 */
function resetWindowScroll() {
  scrollTop.value = 0
  if (scrollEl.value) scrollEl.value.scrollTop = 0
}
watch(() => rows.value, () => resetWindowScroll())

onBeforeUnmount(() => resizeObs?.disconnect())

/** 单次 load 最多翻多少页（页大小由后端固定 ≤1000），避免极端桶把弹窗拖死。 */
const MAX_VERSION_PAGES = 20

/** 可参与内容比较的版本（排除删除标记）。 */
const contentVersions = computed<CompareVersion[]>(() =>
  rows.value
    .filter((v) => !v.isDeleteMarker)
    .map((v) => ({
      versionId: v.versionId,
      size: v.size,
      etag: v.etag,
      lastModified: v.lastModified,
      storageClass: v.storageClass,
      isLatest: v.isLatest,
    })),
)

// 加载代次：每次 load +1。关闭再打开另一个对象时，上一个对象的列举可能仍在飞，
// 过期响应不得写进 rows（否则标题已是新对象、列表却还是旧对象的版本）。
let loadSeq = 0

async function load() {
  const seq = ++loadSeq
  rows.value = []
  truncated.value = false
  truncatedPages.value = 0
  loading.value = true
  try {
    const merged: VersionRow[] = []
    // 按 keyMarker/versionIdMarker 翻页直到不再截断（或达到上限），
    // 避免后端 isTruncated=true 时静默丢失 1000 条之后的版本。
    let isTruncated = false
    let keyMarker: string | undefined
    let versionIdMarker: string | undefined
    let pages = 0
    do {
      const r = await s3api.listVersions(props.accountId, {
        bucket: props.bucket,
        prefix: props.objectKey,
        keyMarker,
        versionIdMarker,
      })
      if (seq !== loadSeq) return // 过期请求：新对象已接管 rows / loading，静默丢弃
      for (const m of r.deleteMarkers ?? []) {
        if (m.key !== props.objectKey) continue
        merged.push({
          key: m.key,
          versionId: m.versionId,
          isLatest: m.isLatest,
          lastModified: m.lastModified,
          size: 0,
          etag: '',
          storageClass: '',
          isDeleteMarker: true,
        })
      }
      for (const v of r.versions ?? []) {
        if (v.key !== props.objectKey) continue
        merged.push({
          key: v.key,
          versionId: v.versionId,
          isLatest: v.isLatest,
          lastModified: v.lastModified,
          size: v.size,
          etag: v.etag,
          storageClass: v.storageClass ?? '',
          isDeleteMarker: false,
        })
      }
      isTruncated = !!r.isTruncated
      keyMarker = r.nextKeyMarker
      versionIdMarker = r.nextVersionIdMarker
      pages++
      // 服务端说还有更多，但没给游标（异常实现）时也停止，避免死循环。
    } while (isTruncated && keyMarker && pages < MAX_VERSION_PAGES)

    if (isTruncated) {
      truncated.value = true
      truncatedPages.value = pages
    }
    merged.sort((a, b) => (b.lastModified || '').localeCompare(a.lastModified || ''))
    rows.value = merged
  } catch (err) {
    // 过期请求的失败不该报到已切走的对象头上
    if (seq === loadSeq) emit('error', toErrorMessage(err))
  } finally {
    // 同理：过期请求不得把新对象的 loading 提前清掉（否则骨架屏闪走、列表未到位）
    if (seq === loadSeq) loading.value = false
  }
}

watch(() => props.open, (o) => {
  if (o) load()
})

async function restore(v: VersionRow) {
  if (busy.value || v.isDeleteMarker) return
  const ok = await confirmDialog({
    title: t('versions.restoreTitle'),
    message: tf('versions.restoreConfirm', { key: props.objectKey, versionId: v.versionId }),
    confirmText: t('versions.restore'),
    danger: false,
  })
  if (!ok) return
  busy.value = true
  try {
    const r = await s3api.restoreObjectVersion(props.accountId, { bucket: props.bucket, key: props.objectKey, versionId: v.versionId })
    toast(tf('versions.restoreOk', { key: props.objectKey, versionId: r.versionId }))
    await load()
  } catch (err) {
    emit('error', toErrorMessage(err))
  } finally {
    busy.value = false
  }
}

/** 一键还原删除标记（撤销删除）：移除该删除标记版本，对象回到被删除前的状态。 */
async function restoreDeleteMarker(v: VersionRow) {
  if (busy.value || !v.isDeleteMarker) return
  const ok = await confirmDialog({
    title: t('versions.restoreDmTitle'),
    message: tf('versions.restoreDmConfirm', { key: props.objectKey }),
    confirmText: t('trash.restore'),
    danger: false,
  })
  if (!ok) return
  busy.value = true
  try {
    await s3api.restoreDeleteMarker(props.accountId, { bucket: props.bucket, key: props.objectKey, versionId: v.versionId })
    toast(tf('versions.restoreDmOk', { key: props.objectKey }))
    await load()
  } catch (err) {
    emit('error', toErrorMessage(err))
  } finally {
    busy.value = false
  }
}

async function removeVersion(v: VersionRow) {
  if (busy.value) return
  const ok = await confirmDialog({
    title: t('versions.deleteTitle'),
    message: tf('versions.deleteConfirm', { key: props.objectKey, versionId: v.versionId }),
    confirmText: t('common.delete'),
    danger: true,
  })
  if (!ok) return
  busy.value = true
  try {
    await s3api.deleteObjectVersion(props.accountId, { bucket: props.bucket, key: props.objectKey, versionId: v.versionId })
    toast(tf('versions.deleteOk', { key: props.objectKey, versionId: v.versionId }))
    await load()
  } catch (err) {
    emit('error', toErrorMessage(err))
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <ModalDialog :open="open" :title="tf('versions.title', { key: objectKey })" width="min(820px, 100%)" @close="emit('close')">
    <div style="margin-bottom:10px" class="mono badge" v-if="objectKey">{{ tf('versions.object', { key: objectKey }) }}</div>
    <div v-if="loading" class="empty" style="padding:22px">{{ t('versions.loading') }}</div>
    <div v-else-if="!rows.length" class="empty" style="padding:22px">
      {{ t('versions.empty') }}
    </div>
    <template v-else>
      <div class="row" style="margin-bottom:10px">
        <button class="btn secondary sm" :disabled="contentVersions.length < 2" @click="compareOpen = true">
          {{ contentVersions.length < 2 ? t('versions.compareNeed') : t('versions.compare') }}
        </button>
        <span class="badge" v-if="rows.some((v) => v.isDeleteMarker)">{{ t('versions.hasDeleteMarker') }}</span>
        <span class="badge" v-if="truncated" style="color:#d64545">{{ tf('versions.truncated', { pages: truncatedPages }) }}</span>
      </div>
      <div ref="scrollEl" class="tbl-wrap tbl-virtual" @scroll.passive="onListScroll">
        <table class="tbl">
          <thead><tr><th style="width:96px">{{ t('versions.colType') }}</th><th>{{ t('versions.colVersionId') }}</th><th style="width:140px">{{ t('versions.colMtime') }}</th><th style="width:70px">{{ t('versions.colSize') }}</th><th style="width:96px">{{ t('versions.colStorage') }}</th><th style="width:190px">{{ t('versions.colActions') }}</th></tr></thead>
          <tbody>
            <tr v-if="windowed.padTop" class="v-spacer" aria-hidden="true">
              <td :colspan="6" :style="{ height: windowed.padTop + 'px' }" />
            </tr>
            <tr v-for="v in windowed.items" :key="v.versionId || 'del-' + v.lastModified" class="v-row" :style="{ height: `${ROW_HEIGHT}px` }">
              <td>
                <span v-if="v.isDeleteMarker" class="badge" style="color:#d64545">{{ t('versions.typeDeleteMarker') }}</span>
                <span v-else-if="v.isLatest" class="badge" style="color:var(--primary)">{{ t('versions.typeLatest') }}</span>
                <span v-else class="badge">{{ t('versions.typeHistory') }}</span>
              </td>
              <td class="mono" style="word-break:break-all">{{ v.versionId || 'null' }}</td>
              <td>{{ fmtDate(v.lastModified) }}</td>
              <td>{{ v.isDeleteMarker ? '—' : fmtSize(v.size) }}</td>
              <td>{{ v.storageClass || '—' }}</td>
              <td>
                <template v-if="v.isDeleteMarker">
                  <button class="btn secondary sm" :disabled="busy" style="color:var(--primary)" @click="restoreDeleteMarker(v)">{{ t('trash.restore') }}</button>
                  <button class="btn danger sm" :disabled="busy" style="margin-left:6px" @click="removeVersion(v)">{{ t('versions.deleteMarker') }}</button>
                </template>
                <template v-else>
                  <button class="btn secondary sm" :disabled="busy" style="margin-right:4px" @click="restore(v)">{{ t('versions.restore') }}</button>
                  <button class="btn danger sm" :disabled="busy" @click="removeVersion(v)">{{ t('common.delete') }}</button>
                </template>
              </td>
            </tr>
            <tr v-if="windowed.padBottom" class="v-spacer" aria-hidden="true">
              <td :colspan="6" :style="{ height: windowed.padBottom + 'px' }" />
            </tr>
          </tbody>
        </table>
      </div>
    </template>
    <div class="row" style="margin-top:14px">
      <button class="btn secondary sm" @click="emit('close')">{{ t('common.close') }}</button>
    </div>

    <CompareDialog
      :open="compareOpen"
      :account-id="accountId"
      :bucket="bucket"
      :object-key="objectKey"
      :versions="contentVersions"
      @close="compareOpen = false"
    />
  </ModalDialog>
</template>

<style scoped>
/* 虚拟滚动：固定可视高度，行高由 v-row 行内 ROW_HEIGHT 绑定（不在此存字面量）。 */
.tbl-virtual {
  max-height: min(60vh, 640px);
  overflow: auto;
}
.tbl-virtual .v-spacer td { padding: 0; border: 0; }
</style>
