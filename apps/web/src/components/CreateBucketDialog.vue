<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import { toErrorMessage } from '../errors'

import { s3api } from '../api'
import { t } from '../i18n'
import ModalDialog from './ModalDialog.vue'

const props = defineProps<{
  open: boolean
  accountId: string
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'created', info: { name: string; region: string; acl: string }): void
  (e: 'error', msg: string): void
}>()

const form = reactive({ name: '', acl: 'private' })
/** 提交在途：防双击重复 createBucket（含 Enter 键路径）。 */
const saving = ref(false)

watch(() => props.open, (o) => {
  if (o) {
    form.name = ''
    form.acl = 'private'
  }
})

async function submitCreateBucket() {
  if (saving.value) return
  const name = form.name.trim()
  if (!name) {
    emit('error', t('buckets.nameRequired'))
    return
  }
  saving.value = true
  try {
    const r = await s3api.createBucket(props.accountId, { name, acl: form.acl })
    emit('created', { name: r.created, region: r.region, acl: r.acl })
  } catch (err) {
    emit('error', toErrorMessage(err))
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <ModalDialog :open="open" :title="t('buckets.create')" width="min(440px, 100%)" @close="emit('close')">
    <div class="grid" style="grid-template-columns:1fr">
      <label class="field">
        {{ t('common.name') }}
        <input v-model="form.name" :placeholder="t('buckets.namePh')" autocomplete="off" spellcheck="false" @keydown.enter.prevent="submitCreateBucket" />
      </label>
      <label class="field">
        {{ t('buckets.aclLabel') }}
        <select v-model="form.acl">
          <option value="private">{{ t('buckets.aclPrivateRecommend') }}</option>
          <option value="public-read">{{ t('buckets.aclPublicRead') }}</option>
          <option value="public-read-write">{{ t('buckets.aclPublicWrite') }}</option>
        </select>
      </label>
    </div>
    <div class="row" style="margin-top:16px">
      <button class="btn sm" :disabled="saving" @click="submitCreateBucket">{{ t('common.create') }}</button>
      <button class="btn secondary sm" @click="emit('close')">{{ t('common.cancel') }}</button>
    </div>
  </ModalDialog>
</template>
