<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import { toErrorMessage } from '../errors'

import { s3api } from '../api'
import { t, tf } from '../i18n'
import { toast } from '../store'
import ModalDialog from './ModalDialog.vue'
import type { ObjectMeta } from '../types'

const props = defineProps<{
  open: boolean
  accountId: string
  bucket: string
  objectKey: string
  detail?: ObjectMeta | null
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'saved'): void
  (e: 'error', msg: string): void
}>()

const contentType = ref('')
/** 行模型带前端生成的 rowKey：v-for 用它做稳定键（index 作 key 会让删除中间行复用错行 DOM）。 */
const meta = reactive<{ rowKey: string; key: string; value: string }[]>([])

/** 行稳定键：组件内自增序列（仅需在同一实例的 v-for 内唯一，不使用会碰撞的业务字段）。 */
let rowSeq = 0
const newRowKey = () => `row-${++rowSeq}`

watch(() => props.open, (o) => {
  if (!o) return
  contentType.value = ''
  meta.splice(0, meta.length)
  const d = props.detail
  if (d && d.key === props.objectKey) {
    contentType.value = d.contentType || ''
    for (const [k, v] of Object.entries(d.metadata ?? {})) {
      meta.push({ rowKey: newRowKey(), key: k, value: v })
    }
  }
})

function addMetaRow() {
  meta.push({ rowKey: newRowKey(), key: '', value: '' })
}

function removeMetaRow(i: number) {
  meta.splice(i, 1)
}

/** 提交在途：防双击重复 setHeaders。 */
const saving = ref(false)

async function submitHeaders() {
  if (saving.value) return
  const m: Record<string, string> = {}
  for (const item of meta) {
    const k = item.key.trim()
    if (k && item.value) m[k] = item.value
  }
  saving.value = true
  try {
    await s3api.setHeaders(props.accountId, {
      bucket: props.bucket,
      key: props.objectKey,
      contentType: contentType.value.trim(),
      metadata: m,
    })
    toast(tf('headers.toastUpdated', { key: props.objectKey }))
    emit('saved')
  } catch (err) {
    emit('error', toErrorMessage(err))
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <ModalDialog :open="open" :title="t('headers.title')" width="min(540px, 100%)" @close="emit('close')">
    <div class="grid" style="grid-template-columns:1fr">
      <label class="field">{{ t('headers.object') }} <span class="mono badge">{{ objectKey }}</span></label>
      <label class="field">
        {{ t('headers.contentType') }}
        <input v-model="contentType" :placeholder="t('headers.contentTypePh')" autocomplete="off" spellcheck="false" />
      </label>
      <div class="field">
        <div class="row" style="justify-content:space-between; margin-bottom:4px">
          <span>{{ t('headers.customMeta') }}</span>
          <button class="btn secondary sm" @click="addMetaRow">{{ t('headers.add') }}</button>
        </div>
        <div v-for="(m, i) in meta" :key="m.rowKey" class="row" style="margin-top:6px">
          <input v-model="m.key" :placeholder="t('headers.keyPh')" style="flex:1" autocomplete="off" spellcheck="false" />
          <input v-model="m.value" :placeholder="t('headers.valuePh')" style="flex:1" autocomplete="off" spellcheck="false" />
          <button class="btn secondary sm" :aria-label="t('common.delete')" @click="removeMetaRow(i)">✕</button>
        </div>
      </div>
    </div>
    <div class="row" style="margin-top:16px">
      <button class="btn sm" :disabled="saving" @click="submitHeaders">{{ t('common.save') }}</button>
      <button class="btn secondary sm" @click="emit('close')">{{ t('common.cancel') }}</button>
    </div>
  </ModalDialog>
</template>
