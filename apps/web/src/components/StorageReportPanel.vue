<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'

import { toErrorMessage } from '../errors'
import { s3api } from '../api'
import { state } from '../store'
import { t } from '../i18n'
import { fmtBytes, fmtCost } from '../storageReport'
import type { BucketItem, StorageReport } from '../types'

// FinOps 成本看板（ROADMAP §三 #7）：选桶（可加前缀）→ 调 /storage-report → 渲染用量与建议。
// 账号沿用全局当前账号；组件在账号切换时重新拉取桶列表。

const accountId = computed(() => state.currentAccountId)
const buckets = ref<BucketItem[]>([])
const bucket = ref('')
const prefix = ref('')
const loadingBuckets = ref(false)
const loading = ref(false)
const error = ref('')
const report = ref<StorageReport | null>(null)

// 桶加载代际（评审 R6）：切账号会连续发起两次 listBuckets，乱序返回时旧响应会把
// 上一个账号的桶选择写进当前账号——写入 / 清错 / 清 loading 都须先验 seq
//（口径同 useObjectBrowser.loadBuckets）。
let bucketSeq = 0

async function loadBuckets() {
  if (!accountId.value) {
    buckets.value = []
    bucket.value = ''
    return
  }
  const seq = ++bucketSeq
  loadingBuckets.value = true
  try {
    const r = await s3api.listBuckets(accountId.value)
    if (seq !== bucketSeq) return // 过期响应：当前账号的加载已接管状态，静默丢弃
    buckets.value = r.buckets
    bucket.value = r.buckets[0]?.name ?? ''
    error.value = ''
  } catch (err) {
    if (seq !== bucketSeq) return // 过期失败：不污染当前账号的横幅
    buckets.value = []
    bucket.value = ''
    error.value = toErrorMessage(err)
  } finally {
    if (seq === bucketSeq) loadingBuckets.value = false // 过期 finally 不得抢清在途请求的 loading
  }
}

async function run() {
  loading.value = true
  error.value = ''
  try {
    report.value = await s3api.storageReport(accountId.value, { bucket: bucket.value, prefix: prefix.value })
  } catch (err) {
    error.value = toErrorMessage(err)
  } finally {
    loading.value = false
  }
}

onMounted(loadBuckets)
watch(accountId, loadBuckets)
</script>

<template>
  <div class="panel finops">
    <h3>{{ t('storageReport.title') }}</h3>
    <p class="hint">{{ t('storageReport.desc') }}</p>

    <div v-if="!accountId" class="empty">{{ t('storageReport.needAccount') }}</div>
    <template v-else>
      <div class="controls">
        <label class="fld">
          <span>{{ t('storageReport.bucket') }}</span>
          <select v-model="bucket" :disabled="loadingBuckets">
            <option v-for="b in buckets" :key="b.name" :value="b.name">{{ b.name }}</option>
          </select>
        </label>
        <label class="fld">
          <span>{{ t('storageReport.prefix') }}</span>
          <input v-model="prefix" :placeholder="t('storageReport.prefixPlaceholder')" />
        </label>
        <button class="btn" :disabled="!bucket || loading" @click="run">{{ t('storageReport.generate') }}</button>
      </div>

      <div v-if="error" class="msg err" role="alert">{{ error }}</div>
      <div v-if="loading" class="empty">{{ t('storageReport.loading') }}</div>
      <div v-else-if="!report" class="empty">{{ t('storageReport.idle') }}</div>
      <div v-else class="report">
        <div class="cards">
          <div class="card"><div class="label">{{ t('storageReport.objectCount') }}</div><div class="value">{{ report.objectCount }}</div></div>
          <div class="card"><div class="label">{{ t('storageReport.totalSize') }}</div><div class="value">{{ fmtBytes(report.totalSize) }}</div></div>
          <div class="card"><div class="label">{{ t('storageReport.monthlyCost') }}</div><div class="value">{{ fmtCost(report.monthlyCost) }}</div></div>
          <div class="card"><div class="label">{{ t('storageReport.prefixGroupCount') }}</div><div class="value">{{ report.prefixGroupCount }}</div></div>
        </div>

        <div v-if="report.truncated" class="msg warn" role="alert">{{ t('storageReport.truncated') }}</div>

        <h3>{{ t('storageReport.byClass') }}</h3>
        <table class="tbl">
          <caption class="sr-only">{{ t('storageReport.byClass') }}</caption>
          <thead><tr>
            <th>{{ t('storageReport.colClass') }}</th><th>{{ t('storageReport.colCount') }}</th>
            <th>{{ t('storageReport.colSize') }}</th><th>{{ t('storageReport.colCost') }}</th>
          </tr></thead>
          <tbody>
            <tr v-for="c in report.byStorageClass" :key="c.storageClass">
              <td class="mono">{{ c.storageClass }}</td>
              <td>{{ c.count }}</td>
              <td>{{ fmtBytes(c.size) }}</td>
              <td>{{ fmtCost(c.monthlyCost) }}</td>
            </tr>
          </tbody>
        </table>

        <h3>{{ t('storageReport.byPrefix') }}</h3>
        <table class="tbl">
          <caption class="sr-only">{{ t('storageReport.byPrefix') }}</caption>
          <thead><tr>
            <th>{{ t('storageReport.colPrefix') }}</th><th>{{ t('storageReport.colCount') }}</th>
            <th>{{ t('storageReport.colSize') }}</th>
          </tr></thead>
          <tbody>
            <tr v-for="p in report.byPrefix" :key="p.prefix">
              <td class="mono">{{ p.prefix }}</td>
              <td>{{ p.count }}</td>
              <td>{{ fmtBytes(p.size) }}</td>
            </tr>
          </tbody>
        </table>

        <h3>{{ t('storageReport.recommendations') }}</h3>
        <table v-if="report.recommendations.length" class="tbl">
          <caption class="sr-only">{{ t('storageReport.recommendations') }}</caption>
          <thead><tr>
            <th>{{ t('storageReport.colKind') }}</th><th>{{ t('storageReport.colFrom') }}</th>
            <th>{{ t('storageReport.colTo') }}</th><th>{{ t('storageReport.colCount') }}</th>
            <th>{{ t('storageReport.colSize') }}</th><th>{{ t('storageReport.colSaving') }}</th>
          </tr></thead>
          <tbody>
            <tr v-for="(r, i) in report.recommendations" :key="i">
              <td>{{ t(`storageReport.kind.${r.kind}`) }}</td>
              <td class="mono">{{ r.fromStorageClass }}</td>
              <td class="mono">{{ r.toStorageClass }}</td>
              <td>{{ r.count }}</td>
              <td>{{ fmtBytes(r.size) }}</td>
              <td>{{ fmtCost(r.estimatedMonthlySaving) }}</td>
            </tr>
          </tbody>
        </table>
        <div v-else class="empty">{{ t('storageReport.noRecommendations') }}</div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.hint { color: var(--muted); font-size: 12px; margin: -8px 0 14px; }
.controls { display: flex; flex-wrap: wrap; gap: 12px; align-items: flex-end; margin-bottom: 14px; }
.fld { display: flex; flex-direction: column; gap: 4px; font-size: 12px; color: var(--muted); }
.fld select, .fld input {
  padding: 7px 9px; border-radius: 8px; border: 1px solid var(--border);
  background: var(--panel-2); color: inherit; min-width: 160px;
}
.cards { display: flex; flex-wrap: wrap; gap: 12px; margin-bottom: 14px; }
.card {
  flex: 1 1 140px; padding: 12px 14px; border: 1px solid var(--border);
  border-radius: 10px; background: var(--panel-2);
}
.card .label { font-size: 12px; color: var(--muted); }
.card .value { font-size: 20px; font-weight: 700; margin-top: 4px; }
.msg.warn { color: var(--danger); border-color: var(--danger-border); background: var(--danger-bg); }
</style>
