<script setup lang="ts">
import { createRowKey } from '../rowKey'
import { ref } from 'vue'

import { s3api } from '../api'
import { t } from '../i18n'
import { useBucketSetting } from '../composables/useBucketSetting'

const props = defineProps<{
  accountId: string
  bucket: string
}>()

const emit = defineEmits<{
  (e: 'error', msg: string): void
  (e: 'changed'): void
}>()

const tags = ref<{ rowKey: string; key: string; value: string }[]>([])

/** 行稳定键：组件内自增序列（v-for key；不使用可能碰撞的业务字段）。 */
const newRowKey = createRowKey()

const { loading, saving, save } = useBucketSetting({
  bucket: () => props.bucket,
  onError: (m) => emit('error', m),
  onChanged: () => emit('changed'),
  load: async () => {
    const r = await s3api.getBucketTags(props.accountId, props.bucket)
    tags.value = (r.tags ?? []).map((tag) => ({ rowKey: newRowKey(), key: tag.key, value: tag.value }))
  },
})

function addRow() {
  tags.value.push({ rowKey: newRowKey(), key: '', value: '' })
}

function removeRow(i: number) {
  tags.value.splice(i, 1)
}

async function clear() {
  await save(async () => {
    await s3api.deleteBucketTags(props.accountId, props.bucket)
    tags.value = []
  }, t('bucketTags.toastCleared'))
}

async function saveTags() {
  await save(async () => {
    // 已按 key.trim() 过滤出有效行；空 key 行不会进入提交（原循环校验为死代码）。
    const valid = tags.value.filter((row) => row.key.trim())
    await s3api.putBucketTags(props.accountId, {
      bucket: props.bucket,
      tags: valid.map((row) => ({ key: row.key.trim(), value: row.value })),
    })
  }, t('bucketTags.toastSaved'))
}
</script>

<template>
  <div v-if="loading" class="empty" style="padding:20px">{{ t('bucketTags.loading') }}</div>
  <div v-else>
    <table class="tbl">
      <caption class="sr-only">{{ t('bucketTags.tableAria') }}</caption>
      <thead><tr><th id="bucket-tags-key-h" style="width:40%">{{ t('bucketTags.colKey') }}</th><th id="bucket-tags-val-h">{{ t('bucketTags.colValue') }}</th><th style="width:60px"></th></tr></thead>
      <tbody>
        <tr v-for="(row, i) in tags" :key="row.rowKey">
          <td><input v-model="row.key" class="mono" placeholder="key" aria-labelledby="bucket-tags-key-h" /></td>
          <td><input v-model="row.value" class="mono" placeholder="value" aria-labelledby="bucket-tags-val-h" /></td>
          <td><button class="btn secondary sm" @click="removeRow(i)">{{ t('common.remove') }}</button></td>
        </tr>
      </tbody>
    </table>
    <div class="row" style="margin-top:12px">
      <button class="btn secondary sm" @click="addRow">{{ t('bucketTags.add') }}</button>
      <button class="btn sm" :disabled="saving" @click="saveTags">{{ saving ? t('common.saving') : t('common.save') }}</button>
      <button class="btn danger sm" :disabled="saving" @click="clear">{{ t('bucketTags.clearAll') }}</button>
    </div>
  </div>
</template>
