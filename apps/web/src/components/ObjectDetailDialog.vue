<script setup lang="ts">
import { computed, ref, watch } from 'vue'

import { s3api } from '../api'
import { toErrorMessage } from '../errors'
import { fmtDate, fmtSize } from '../format'
import { t } from '../i18n'
import ModalDialog from './ModalDialog.vue'
import { storageClassLabel } from '../storageClass'
import type { ObjectLegalHold, ObjectMeta, ObjectRetention, RetentionMode, VerifyResult } from '../types'

const props = defineProps<{
  open: boolean
  detail: ObjectMeta | null
  /** 账号 / 桶：读写对象保护（保留期、法定保留、校验和复核）所需的寻址参数。 */
  accountId: string
  bucket: string
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'editHeaders', key: string): void
  (e: 'openAcl', key: string): void
  (e: 'openTags', key: string): void
  (e: 'openVersions', key: string): void
  (e: 'openStorageClass', key: string): void
}>()

/* ---- 校验和：HeadObject 的 checksums 展示 + 服务端复核 ---- */
const verifying = ref(false)
const verifyResult = ref<VerifyResult | null>(null)
const verifyError = ref('')

const checksumRows = computed<Array<{ name: string; value: string }>>(() => {
  const checksums = props.detail?.checksums
  if (!checksums) return []
  const rows: Array<{ name: string; value: string }> = []
  for (const [name, value] of Object.entries(checksums)) {
    if (value) rows.push({ name, value })
  }
  return rows
})

async function verify(key: string): Promise<void> {
  verifying.value = true
  verifyError.value = ''
  verifyResult.value = null
  try {
    verifyResult.value = await s3api.verifyChecksum(props.accountId, { bucket: props.bucket, key })
  } catch (e) {
    verifyError.value = toErrorMessage(e)
  } finally {
    verifying.value = false
  }
}

/* ---- 对象保护：保留期 + 法定保留（详情打开时读取，改完就地回显） ---- */
const retention = ref<ObjectRetention | null>(null)
const legalHold = ref<'ON' | 'OFF'>('OFF')
const protectionError = ref('')
const retentionSaving = ref(false)
const retentionError = ref('')
const holdSaving = ref(false)
const holdError = ref('')
/** 编辑用的保留期表单字段：加载时由 retention 回填，保存前本地校验。 */
const mode = ref<RetentionMode>('GOVERNANCE')
const until = ref('')

/** `datetime-local` 要本地时区的 `YYYY-MM-DDTHH:mm`，而 retainUntilDate 是 UTC RFC3339。 */
function toLocalInput(iso: string): string {
  const d = new Date(iso)
  const pad = (n: number): string => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

/** 序号守卫：快速切换对象时丢弃过期的读取结果（与 useObjectActions 的 detailSeq 同口径）。 */
let protectionSeq = 0

async function loadProtection(): Promise<void> {
  if (!props.detail) return
  const seq = ++protectionSeq
  protectionError.value = ''
  // 读取失败也归一为返回值：错误必须先过 seq 检查，不能绕过守卫写入过期状态。
  const outcome = await Promise.all([
    s3api.getObjectRetention(props.accountId, { bucket: props.bucket, key: props.detail.key }),
    s3api.getObjectLegalHold(props.accountId, { bucket: props.bucket, key: props.detail.key }),
  ]).then(
    (v): { ok: [ObjectRetention, ObjectLegalHold] } => ({ ok: v }),
    (e: unknown): { err: string } => ({ err: toErrorMessage(e) }),
  )
  if (seq !== protectionSeq) return
  if ('err' in outcome) {
    protectionError.value = outcome.err
    return
  }
  const [gotRetention, gotHold] = outcome.ok
  retention.value = gotRetention
  legalHold.value = gotHold.status === 'ON' ? 'ON' : 'OFF'
  mode.value = gotRetention.mode === 'COMPLIANCE' ? 'COMPLIANCE' : 'GOVERNANCE'
  until.value = gotRetention.retainUntilDate ? toLocalInput(gotRetention.retainUntilDate) : ''
}

watch(
  [() => props.open, () => props.detail?.key],
  ([open]) => {
    if (open) void loadProtection()
  },
  { immediate: true },
)

async function saveRetention(key: string): Promise<void> {
  retentionError.value = ''
  const ts = Date.parse(until.value)
  // 后端只接受未来时刻（否则 400）：先在本地拦下，省一次往返也少一句后端术语。
  if (Number.isNaN(ts) || ts <= Date.now()) {
    retentionError.value = t('detail.retentionInvalid')
    return
  }
  retentionSaving.value = true
  try {
    retention.value = await s3api.putObjectRetention(props.accountId, {
      bucket: props.bucket,
      key,
      mode: mode.value,
      retainUntilDate: new Date(ts).toISOString(),
    })
  } catch (e) {
    // 400 输入违规 / 403 GOVERNANCE 拒绝 / 409 ObjectLocked：原样展示后端固定文案。
    retentionError.value = toErrorMessage(e)
  } finally {
    retentionSaving.value = false
  }
}

async function toggleLegalHold(key: string): Promise<void> {
  const status = legalHold.value === 'ON' ? 'OFF' : 'ON'
  holdSaving.value = true
  holdError.value = ''
  try {
    const updated = await s3api.putObjectLegalHold(props.accountId, { bucket: props.bucket, key, status })
    legalHold.value = updated.status === 'ON' ? 'ON' : 'OFF'
  } catch (e) {
    holdError.value = toErrorMessage(e)
  } finally {
    holdSaving.value = false
  }
}
</script>

<template>
  <ModalDialog :open="open" :title="t('detail.title')" width="min(640px, 100%)" @close="emit('close')">
    <template v-if="detail">
      <table class="tbl detail-tbl">
        <caption class="sr-only">{{ t('detail.tableAria') }}</caption>
        <tbody>
          <tr><th>{{ t('detail.key') }}</th><td class="mono" style="word-break:break-all">{{ detail.key }}</td></tr>
          <tr><th>{{ t('common.size') }}</th><td>{{ fmtSize(detail.size) }}</td></tr>
          <tr><th>{{ t('detail.mtime') }}</th><td>{{ fmtDate(detail.lastModified) }}</td></tr>
          <tr><th>{{ t('detail.contentType') }}</th><td class="mono">{{ detail.contentType || '—' }}</td></tr>
          <tr><th>{{ t('objects.storageClass') }}</th><td>
            {{ detail.storageClass ? storageClassLabel(detail.storageClass) : '—' }}
            <span v-if="detail.storageClass" class="mono badge" style="margin-left:6px">{{ detail.storageClass }}</span>
            <button class="btn secondary sm" style="margin-left:10px" @click="emit('openStorageClass', detail.key)">{{ t('detail.switch') }}</button>
          </td></tr>
          <tr><th>{{ t('detail.etag') }}</th><td class="mono" style="word-break:break-all">{{ detail.etag || '—' }}</td></tr>
          <tr v-if="detail.metadata && Object.keys(detail.metadata).length">
            <th>{{ t('detail.metadata') }}</th>
            <td>
              <div v-for="(v, k) in detail.metadata" :key="k" class="mono">{{ k }}: {{ v }}</div>
            </td>
          </tr>
          <tr>
            <th>{{ t('detail.checksums') }}</th>
            <td>
              <template v-if="checksumRows.length">
                <div v-for="row in checksumRows" :key="row.name" class="mono" style="word-break:break-all">{{ row.name }}: {{ row.value }}</div>
              </template>
              <span v-else>{{ t('detail.checksumsNone') }}</span>
              <div style="margin-top:6px">
                <button class="btn sm" :disabled="verifying" @click="verify(detail.key)">{{ t('detail.verify') }}</button>
              </div>
              <div v-if="verifyResult" class="badge" style="display:block; margin-top:6px">
                <template v-if="verifyResult.method === 'none'">{{ t('detail.verifyNone') }}</template>
                <template v-else>
                  <div>{{ t('detail.verifyMethod') }}: {{ verifyResult.method }}</div>
                  <div class="mono" style="word-break:break-all">{{ t('detail.verifyLocal') }}: {{ verifyResult.local }}</div>
                  <div class="mono" style="word-break:break-all">{{ t('detail.verifyRemote') }}: {{ verifyResult.remote }}</div>
                  <div>{{ verifyResult.match ? t('detail.verifyMatch') : t('detail.verifyMismatch') }}</div>
                </template>
              </div>
              <div v-if="verifyError" class="badge" role="alert" style="color:var(--danger)">{{ verifyError }}</div>
            </td>
          </tr>
          <tr>
            <th>{{ t('detail.retention') }}</th>
            <td>
              <template v-if="retention?.configured">
                <span class="badge">{{ retention.mode }}</span>
                <span style="margin-left:6px">{{ fmtDate(retention.retainUntilDate) }}</span>
              </template>
              <span v-else>{{ t('detail.retentionNone') }}</span>
              <div v-if="protectionError" class="badge" role="alert" style="color:var(--danger)">{{ protectionError }}</div>
              <label class="field">
                {{ t('detail.retentionMode') }}
                <select v-model="mode">
                  <option value="GOVERNANCE">GOVERNANCE</option>
                  <option value="COMPLIANCE">COMPLIANCE</option>
                </select>
              </label>
              <label class="field">
                {{ t('detail.retentionUntil') }}
                <input v-model="until" type="datetime-local" />
              </label>
              <button class="btn sm" :disabled="retentionSaving" @click="saveRetention(detail.key)">{{ t('detail.retentionSave') }}</button>
              <div v-if="retentionError" class="badge" role="alert" style="color:var(--danger)">{{ retentionError }}</div>
            </td>
          </tr>
          <tr>
            <th>{{ t('detail.legalHold') }}</th>
            <td>
              <span class="badge">{{ legalHold === 'ON' ? t('detail.legalHoldOn') : t('detail.legalHoldOff') }}</span>
              <button class="btn sm" style="margin-left:10px" :disabled="holdSaving" @click="toggleLegalHold(detail.key)">
                {{ legalHold === 'ON' ? t('detail.legalHoldTurnOff') : t('detail.legalHoldTurnOn') }}
              </button>
              <div v-if="holdError" class="badge" role="alert" style="color:var(--danger)">{{ holdError }}</div>
            </td>
          </tr>
        </tbody>
      </table>
      <div class="row" style="margin-top:12px">
        <button class="btn sm" @click="emit('editHeaders', detail.key)">{{ t('detail.editHeaders') }}</button>
        <button class="btn sm" @click="emit('openAcl', detail.key)">{{ t('detail.acl') }}</button>
        <button class="btn sm" @click="emit('openTags', detail.key)">{{ t('detail.tags') }}</button>
        <button class="btn sm" @click="emit('openVersions', detail.key)">{{ t('detail.versions') }}</button>
        <span class="badge">{{ t('detail.metaBadge') }}</span>
      </div>
    </template>
  </ModalDialog>
</template>

<style scoped>
.detail-tbl th { width: 120px; background: none; border-bottom: 1px solid var(--border); }
.detail-tbl th, .detail-tbl td { padding: 7px 10px; }
</style>
