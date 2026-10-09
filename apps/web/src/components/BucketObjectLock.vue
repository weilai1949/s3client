<script setup lang="ts">
import { ref } from 'vue'

import { s3api } from '../api'
import { t } from '../i18n'
import { useBucketSetting } from '../composables/useBucketSetting'
import type { RetentionMode } from '../types'

const props = defineProps<{
  accountId: string
  bucket: string
}>()

const emit = defineEmits<{
  (e: 'error', msg: string): void
  (e: 'changed'): void
}>()

/** Object Lock 状态与默认保留策略表单（写前在本地拦一次时长合法性）。 */
const enabled = ref(false)
const mode = ref<RetentionMode>('GOVERNANCE')
const amount = ref(1)
const unit = ref<'days' | 'years'>('days')
const formError = ref('')

const { loading, saving, save } = useBucketSetting({
  bucket: () => props.bucket,
  onError: (m) => emit('error', m),
  onChanged: () => emit('changed'),
  load: async () => {
    const r = await s3api.getObjectLock(props.accountId, props.bucket)
    enabled.value = r.enabled
    mode.value = r.defaultRetentionMode === 'COMPLIANCE' ? 'COMPLIANCE' : 'GOVERNANCE'
    // 后端只回一份时长（未配置默认保留时两者都是 0）：按非零项回填表单。
    if (r.defaultRetentionDays >= 1) {
      unit.value = 'days'
      amount.value = r.defaultRetentionDays
    } else if (r.defaultRetentionYears >= 1) {
      unit.value = 'years'
      amount.value = r.defaultRetentionYears
    } else {
      unit.value = 'days'
      amount.value = 1
    }
    formError.value = ''
  },
})

async function savePolicy(): Promise<void> {
  const n = Number(amount.value)
  if (!Number.isInteger(n) || n < 1) {
    formError.value = t('objlock.invalidAmount')
    return
  }
  formError.value = ''
  // 提交成功后 useBucketSetting 会 reload() 重新回填，这里不再重复写状态。
  await save(
    async () => {
      await s3api.putObjectLock(props.accountId, {
        bucket: props.bucket,
        defaultRetentionMode: mode.value,
        ...(unit.value === 'days' ? { defaultRetentionDays: n } : { defaultRetentionYears: n }),
      })
    },
    t('objlock.toastSaved'),
  )
}
</script>

<template>
  <div v-if="loading" class="empty" style="padding:20px">{{ t('objlock.loading') }}</div>
  <div v-else>
    <div class="row" style="margin-bottom:10px">
      <span class="badge">{{ t('objlock.status') }}</span>
      <span class="badge" :style="enabled ? 'color:var(--primary)' : ''">
        {{ enabled ? t('objlock.enabled') : t('objlock.disabled') }}
      </span>
    </div>
    <div v-if="!enabled" class="badge" style="color:var(--muted)">{{ t('objlock.notEnabledHint') }}</div>
    <label class="field">
      {{ t('objlock.mode') }}
      <select v-model="mode">
        <option value="GOVERNANCE">GOVERNANCE</option>
        <option value="COMPLIANCE">COMPLIANCE</option>
      </select>
    </label>
    <div class="row" style="align-items:flex-end; gap:10px">
      <label class="field">
        {{ t('objlock.amount') }}
        <input v-model.number="amount" type="number" min="1" step="1" class="mono" style="width:110px" />
      </label>
      <label class="field">
        {{ t('objlock.unit') }}
        <select v-model="unit">
          <option value="days">{{ t('objlock.unitDays') }}</option>
          <option value="years">{{ t('objlock.unitYears') }}</option>
        </select>
      </label>
    </div>
    <div class="row" style="margin-top:14px">
      <button class="btn sm" :disabled="saving" @click="savePolicy">
        {{ saving ? t('common.saving') : t('objlock.save') }}
      </button>
    </div>
    <div v-if="formError" class="msg err" role="alert" style="margin-top:10px">{{ formError }}</div>
    <div class="badge" style="margin-top:10px; color:var(--muted)">{{ t('objlock.hint') }}</div>
  </div>
</template>
