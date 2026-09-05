<script setup lang="ts">
import { ref, watch } from 'vue'

import { s3api } from '../api'
import { confirmDialog } from '../confirm'
import { t, tf } from '../i18n'
import { useBucketSetting } from '../composables/useBucketSetting'
import BucketPolicyVisualEditor from './BucketPolicyVisualEditor.vue'

const props = defineProps<{
  accountId: string
  bucket: string
}>()

const emit = defineEmits<{
  (e: 'error', msg: string): void
  (e: 'changed'): void
}>()

const configured = ref(false)
const policy = ref('')
const draft = ref('') // 可视化编辑器产出的最新 JSON；保存按钮按下时用它。

const { loading, saving, save } = useBucketSetting({
  bucket: () => props.bucket,
  onError: (m) => emit('error', m),
  onChanged: () => emit('changed'),
  load: async () => {
    const r = await s3api.getBucketPolicy(props.accountId, props.bucket)
    configured.value = r.configured
    policy.value = r.policy || ''
    draft.value = r.policy || ''
  },
})

// 当父组件重置（切桶等），让 draft 跟着最新 policy 同步。
watch(
  () => policy.value,
  (v) => {
    draft.value = v
  },
)

async function remove() {
  const ok = await confirmDialog({
    title: t('policy.removeTitle'),
    message: tf('policy.removeConfirm', { bucket: props.bucket }),
    confirmText: t('common.remove'),
    danger: true,
  })
  if (!ok) return
  await save(async () => {
    await s3api.deleteBucketPolicy(props.accountId, props.bucket)
    policy.value = ''
  }, t('policy.toastRemoved'))
}

async function savePolicy() {
  await save(async () => {
    const v = draft.value.trim()
    if (!v) throw new Error(t('policy.emptyErr'))
    try {
      JSON.parse(v)
    } catch {
      throw new Error(t('policy.invalidJson'))
    }
    await s3api.putBucketPolicy(props.accountId, { bucket: props.bucket, policy: v })
    policy.value = v
  }, t('policy.toastSaved'))
}
</script>

<template>
  <div v-if="loading" class="empty" style="padding:20px">{{ t('policy.loading') }}</div>
  <div v-else>
    <div class="badge" style="color:var(--muted)">{{ t('policy.hint') }}</div>
    <BucketPolicyVisualEditor
      :bucket="bucket"
      :raw="policy"
      @update="draft = $event"
      @error="(m) => emit('error', m)"
    />
    <div class="row" style="margin-top:12px">
      <button class="btn sm" :disabled="saving" @click="savePolicy">
        {{ saving ? t('common.saving') : t('common.save') }}
      </button>
      <button class="btn danger sm" :disabled="saving" @click="remove">
        {{ t('policy.removeBtn') }}
      </button>
    </div>
  </div>
</template>
