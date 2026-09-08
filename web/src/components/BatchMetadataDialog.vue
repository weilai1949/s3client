<script setup lang="ts">
import { computed, reactive, ref } from 'vue'

import { batchSetMetadata, type BatchMetaError } from '../batchMetadata'
import { s3api } from '../api'
import { toErrorMessage } from '../errors'
import ModalDialog from './ModalDialog.vue'
import { toast } from '../store'
import { t, tf } from '../i18n'

const props = defineProps<{
  accountId: string
  bucket: string
  keys: string[]
  /** 受控显隐：父级用 :open 而非 v-if 挂载，避免关闭时销毁组件丢失进行中状态。 */
  open?: boolean
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'done', result: { ok: number; failed: number }): void
}>()

const open = computed(() => props.open ?? true)
const running = ref(false)
const progress = reactive({ done: 0, total: 0 })

// 用户勾选 + 选择的字段值。
const applyAcl = ref(false)
const acl = ref<'private' | 'public-read' | 'public-read-write'>('private')

const applyTags = ref(false)
const tagsMode = ref<'replace' | 'clear' | 'none'>('none')
const tags = ref<{ key: string; value: string }[]>([])

// storageClass 选项来自 HEAD 当前桶内一个样本对象的可能值；前端不强校验，
// 后端 changeStorageClass 已对非法值返 400。
const applyStorage = ref(false)
const storageClass = ref('STANDARD_IA')

const errors = ref<BatchMetaError[]>([])
const result = ref<{ ok: number; failed: number } | null>(null)
const showAllErrors = ref(false)

// 是否实际发生了标签修改：clear 恒为改；replace 需至少有一行非空（key 或 value 有内容）。
const hasTagChange = computed(() => {
  if (!applyTags.value) return false
  if (tagsMode.value === 'clear') return true
  if (tagsMode.value === 'replace') return tags.value.some((tg) => tg.key || tg.value)
  return false
})

const noChange = computed(() => !applyAcl.value && !hasTagChange.value && !applyStorage.value)

// 错误明细：默认只展示前 50 条，可展开显示全部。
const shownErrors = computed(() => (showAllErrors.value ? errors.value : errors.value.slice(0, 50)))

function close() {
  if (running.value) return
  emit('close')
}

function warnToast(msg: string): void {
  toast(msg, 'err')
}

function addTagRow() {
  tags.value.push({ key: '', value: '' })
}
function removeTagRow(i: number) {
  const next = tags.value.slice()
  next.splice(i, 1)
  tags.value = next
}

async function onConfirm() {
  if (running.value) return
  if (props.keys.length === 0) {
    warnToast(t('batchEdit.empty'))
    return
  }
  if (noChange.value) {
    warnToast(t('batchEdit.needStep'))
    return
  }
  // 校验：标签替换模式下至少有一个非空 key。
  if (applyTags.value && tagsMode.value === 'replace') {
    const filled = tags.value.filter((tg) => tg.key)
    if (filled.length === 0) {
      warnToast(t('batchEdit.tagsNeedKey'))
      return
    }
  }

  running.value = true
  errors.value = []
  result.value = null
  showAllErrors.value = false
  progress.done = 0
  progress.total = props.keys.length

  // 构造输入：tags 仅在「replace」模式注入；「clear」传空数组；「none」不传。
  let tagsArg: { key: string; value: string }[] | undefined
  if (applyTags.value) {
    if (tagsMode.value === 'replace') tagsArg = tags.value.filter((tg) => tg.key)
    else if (tagsMode.value === 'clear') tagsArg = []
  }

  try {
    const out = await batchSetMetadata({
      accountId: props.accountId,
      bucket: props.bucket,
      keys: props.keys,
      acl: applyAcl.value ? acl.value : undefined,
      tags: tagsArg,
      storageClass: applyStorage.value ? storageClass.value : undefined,
      onProgress: (done, total) => {
        progress.done = done
        progress.total = total
      },
    })
    result.value = { ok: out.ok, failed: out.failed }
    errors.value = out.errors
    progress.done = props.keys.length
    if (out.failed === 0) {
      toast(tf('batchEdit.done', { ok: out.ok, failed: 0 }), 'ok')
      emit('done', { ok: out.ok, failed: 0 })
    } else {
      toast(tf('batchEdit.done', { ok: out.ok, failed: out.failed }), 'err')
      emit('done', { ok: out.ok, failed: out.failed })
    }
  } catch (e) {
    // batchSetMetadata 只会对 per-key 错误返回部分结果；此处 catch 覆盖真正的 throw
    // （如 key 超限等输入量级错误 / 网络层异常）。step 用通用 'batch'，不再硬编码 'acl'。
    const msg = toErrorMessage(e)
    result.value = { ok: 0, failed: props.keys.length }
    errors.value = [{ key: '*', step: 'batch', message: msg }]
    toast(tf('batchEdit.fatalError', { msg }), 'err')
    emit('done', { ok: 0, failed: props.keys.length })
  } finally {
    running.value = false
  }
}

// 静默导入以避免 lint 报「未使用」（保留扩展点：未来若需按桶探测可用存储类型）。
void s3api
</script>

<template>
  <ModalDialog :open="open" :title="t('batchEdit.title')" data-testid="batch-edit-dialog" @close="close">
    <div class="badge" style="color:var(--muted)">
      {{ tf('batchEdit.hint', { n: keys.length }) }}
    </div>

    <fieldset>
      <legend>
        <label>
          <input v-model="applyAcl" type="checkbox" data-testid="batch-edit-acl-toggle" />
          {{ t('batchEdit.aclLabel') }}
        </label>
      </legend>
      <select v-model="acl" :disabled="!applyAcl" class="full" aria-label="ACL">
        <option value="private">private</option>
        <option value="public-read">public-read</option>
        <option value="public-read-write">public-read-write</option>
      </select>
    </fieldset>

    <fieldset>
      <legend>
        <label>
          <input v-model="applyTags" type="checkbox" data-testid="batch-edit-tags-toggle" />
          {{ t('batchEdit.tagsLabel') }}
        </label>
      </legend>
      <select v-model="tagsMode" :disabled="!applyTags" class="full" aria-label="Tag mode">
        <option value="none">{{ t('batchEdit.tagsNoChange') }}</option>
        <option value="replace">替换</option>
        <option value="clear">{{ t('batchEdit.tagsClear') }}</option>
      </select>
      <div v-if="applyTags && tagsMode === 'replace'" style="margin-top:8px">
        <div v-for="(tg, i) in tags" :key="i" class="tag-row">
          <label class="sr-only" :for="'batch-tag-key-' + i">{{ t('batchEdit.tagKey') }}</label>
          <input :id="'batch-tag-key-' + i" v-model="tg.key" type="text" :placeholder="t('batchEdit.tagKey')" />
          <label class="sr-only" :for="'batch-tag-val-' + i">{{ t('batchEdit.tagValue') }}</label>
          <input :id="'batch-tag-val-' + i" v-model="tg.value" type="text" :placeholder="t('batchEdit.tagValue')" />
          <button class="btn sm danger" type="button" @click="removeTagRow(i)">×</button>
        </div>
        <button class="btn sm" type="button" @click="addTagRow">+</button>
      </div>
    </fieldset>

    <fieldset>
      <legend>
        <label>
          <input v-model="applyStorage" type="checkbox" data-testid="batch-edit-storage-toggle" />
          {{ t('batchEdit.storageLabel') }}
        </label>
      </legend>
      <input v-model="storageClass" :disabled="!applyStorage" type="text" class="full" aria-label="Storage class" />
    </fieldset>

    <div v-if="running || result" class="status" aria-live="polite">
      <span v-if="running">{{ tf('batchEdit.running', { done: progress.done, total: progress.total }) }}</span>
      <span v-else>{{ tf('batchEdit.done', { ok: result!.ok, failed: result!.failed }) }}</span>
      <progress v-if="running" class="progress bar" :max="progress.total || 1" :value="progress.done"></progress>
    </div>

    <div v-if="errors.length" class="errors">
      <strong>{{ tf('batchEdit.errors', { n: shownErrors.length }) }}</strong>
      <ul>
        <li v-for="(e, i) in shownErrors" :key="i">
          <code>{{ e.key }}</code> · {{ e.step }} · {{ e.message }}
        </li>
      </ul>
      <button v-if="errors.length > 50" class="link" type="button" @click="showAllErrors = !showAllErrors">
        {{ showAllErrors ? t('batchEdit.showLess') : tf('batchEdit.showMore', { n: errors.length }) }}
      </button>
    </div>

    <template #footer>
      <button class="btn sm" type="button" :disabled="running" @click="close">{{ t('common.cancel') }}</button>
      <button class="btn primary sm" type="button" :disabled="running || noChange" @click="onConfirm">
        {{ running ? t('common.working') : t('batchEdit.confirm') }}
      </button>
    </template>
  </ModalDialog>
</template>

<style scoped>
.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}
fieldset {
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 8px 12px;
  margin: 8px 0;
}
legend {
  padding: 0 6px;
  color: var(--muted);
  font-size: 12px;
}
.full {
  width: 100%;
  padding: 6px 10px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--panel);
  color: var(--text);
  font-size: 13px;
}
.tag-row {
  display: grid;
  grid-template-columns: 1fr 1fr auto;
  gap: 6px;
  margin: 4px 0;
}
.tag-row input {
  padding: 4px 8px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--panel);
  color: var(--text);
  font-size: 13px;
}
.status {
  margin: 8px 0;
  color: var(--brand);
  font-size: 13px;
}
.progress.bar {
  width: 100%;
  margin-top: 4px;
  height: 6px;
}
.errors {
  margin: 8px 0;
  padding: 8px 12px;
  border: 1px solid #ef4444;
  border-radius: var(--radius);
  background: rgba(239, 68, 68, 0.05);
  font-size: 12px;
  max-height: 160px;
  overflow: auto;
}
.errors ul {
  margin: 4px 0 0;
  padding-left: 16px;
}
.errors code {
  font-family: var(--font-mono);
  color: var(--muted);
}
</style>
