<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import { toErrorMessage } from '../errors'

import { s3api } from '../api'
import { toast } from '../store'
import { t, tf } from '../i18n'
import ModalDialog from './ModalDialog.vue'
import type { LifecycleRule } from '../types'

const props = defineProps<{
  open: boolean
  accountId: string
  bucket: string
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'error', msg: string): void
}>()

const rules = reactive<(LifecycleRule & { rowKey: string })[]>([])
const loading = ref(false)

/** 行稳定键：组件内自增序列（v-for key；不使用可能碰撞的业务字段 id）。 */
let rowSeq = 0
const newRowKey = () => `row-${++rowSeq}`

watch(() => props.open, async (o) => {
  if (!o) return
  rules.splice(0, rules.length)
  loading.value = true
  try {
    const res = await s3api.getLifecycle(props.accountId, props.bucket)
    for (const r of res.rules ?? []) rules.push({ ...r, rowKey: newRowKey() })
  } catch (e) {
    emit('error', toErrorMessage(e))
  } finally {
    loading.value = false
  }
})

function addRule() {
  rules.push({ id: 'rule-' + Date.now().toString(36), prefix: '', days: 30, rowKey: newRowKey() })
}

function removeRule(i: number) {
  rules.splice(i, 1)
}

/** 提交在途：防双击重复 putLifecycle（与初始加载的 loading 分开）。 */
const saving = ref(false)

async function submitLifecycle() {
  if (saving.value) return
  // rowKey 是前端渲染用的行键（v-for key），只用于本地，不进请求体。
  const validRules = rules
    .filter((r) => r.prefix.trim() && r.days >= 1)
    .map(({ rowKey: _rowKey, ...rule }) => rule)
  saving.value = true
  try {
    const r = await s3api.putLifecycle(props.accountId, { bucket: props.bucket, rules: validRules })
    toast(tf('lifecycle.toastSaved', { n: r.updated }))
    emit('close')
  } catch (e) {
    emit('error', toErrorMessage(e))
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <ModalDialog :open="open" :title="tf('lifecycle.title', { bucket })" width="min(640px, 100%)" @close="emit('close')">
    <div class="badge" style="display:block; margin-bottom:10px; line-height:1.7">
      {{ t('lifecycle.hint') }}
    </div>
    <div v-if="loading" aria-busy="true">
      <div v-for="i in 2" :key="i" class="skel-row" />
    </div>
    <table v-else-if="rules.length" class="tbl">
      <caption class="sr-only">{{ t('lifecycle.tableAria') }}</caption>
      <thead>
        <tr>
          <th style="width:110px">{{ t('lifecycle.colId') }}</th>
          <th id="lifecycle-prefix-h">{{ t('lifecycle.colPrefix') }}</th>
          <th id="lifecycle-days-h" style="width:110px">{{ t('lifecycle.colDays') }}</th>
          <th style="width:60px"></th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="(r, i) in rules" :key="r.rowKey">
          <td class="mono">{{ r.id }}</td>
          <td><input v-model="r.prefix" :placeholder="t('lifecycle.prefixPh')" style="width:100%" autocomplete="off" spellcheck="false" aria-labelledby="lifecycle-prefix-h" /></td>
          <td><input v-model.number="r.days" type="number" min="1" style="width:88px" aria-labelledby="lifecycle-days-h" /></td>
          <td><button class="btn danger sm" @click="removeRule(i)">{{ t('common.delete') }}</button></td>
        </tr>
      </tbody>
    </table>
    <div v-else class="empty" style="padding:18px">
      {{ t('lifecycle.empty') }}
    </div>
    <div class="row" style="margin-top:12px">
      <button class="btn secondary sm" @click="addRule">{{ t('lifecycle.addRule') }}</button>
      <span class="spacer" />
      <button class="btn sm" :disabled="!rules.length || loading || saving" @click="submitLifecycle">{{ t('lifecycle.saveRules') }}</button>
      <button class="btn secondary sm" @click="emit('close')">{{ t('common.cancel') }}</button>
    </div>
  </ModalDialog>
</template>
