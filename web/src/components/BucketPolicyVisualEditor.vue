<script setup lang="ts">
import { computed, ref, watch } from 'vue'

import {
  POLICY_TEMPLATES,
  parsePolicy,
  type PolicyDoc,
  type PolicyStatement,
  serializePolicy,
  validateDoc,
} from '../bucketPolicy'
import { t, tf } from '../i18n'

const props = defineProps<{
  /** 当前桶名，用于模板中 ARN 填充。 */
  bucket: string
  /** 来自父组件的策略字符串（可能为空）。 */
  raw: string
}>()

const emit = defineEmits<{
  (e: 'update', json: string): void
  (e: 'error', msg: string): void
}>()

const mode = ref<'visual' | 'json'>('visual')

const doc = ref<PolicyDoc>({ Version: '2012-10-17', Statement: [] })
const parseError = ref(false)
const dirty = ref(false)

watch(
  () => props.raw,
  (raw) => {
    const parsed = parsePolicy(raw)
    if (parsed) {
      doc.value = parsed
      parseError.value = false
      dirty.value = false
    } else if (raw.trim() === '') {
      // 空 -> 清空 doc；不报错。
      doc.value = { Version: '2012-10-17', Statement: [] }
      parseError.value = false
      dirty.value = false
    } else {
      // 父组件已加载的 JSON 含可视化不支持的结构（NotPrincipal / 嵌套）。
      parseError.value = true
      mode.value = 'json'
    }
  },
  { immediate: true },
)

const validationError = computed(() => validateDoc(doc.value))
const previewJSON = computed(() => {
  try {
    return serializePolicy(doc.value)
  } catch (e) {
    return String((e as Error)?.message ?? e)
  }
})

watch(
  doc,
  () => {
    dirty.value = true
    emit('update', previewJSON.value)
  },
  { deep: true },
)

function applyTemplate(id: string) {
  const tpl = POLICY_TEMPLATES.find((x) => x.id === id)
  if (!tpl) return
  doc.value = tpl.build(props.bucket)
  dirty.value = true
  emit('update', previewJSON.value)
}

function addStatement() {
  doc.value = {
    ...doc.value,
    Statement: [
      ...doc.value.Statement,
      {
        sid: '',
        effect: 'Allow',
        principal: '*',
        actions: ['s3:GetObject'],
        resources: [`arn:aws:s3:::${props.bucket}/*`],
      },
    ],
  }
}

function removeStatement(idx: number) {
  const next = [...doc.value.Statement]
  next.splice(idx, 1)
  doc.value = { ...doc.value, Statement: next }
}

function toLines(arr: string[]): string {
  return arr.join('\n')
}

function parseLines(s: string): string[] {
  return s
    .split('\n')
    .map((x) => x.trim())
    .filter((x) => x !== '')
}

function onActionsChange(idx: number, v: string) {
  const next = [...doc.value.Statement]
  next[idx] = { ...next[idx], actions: parseLines(v) }
  doc.value = { ...doc.value, Statement: next }
}

function onResourcesChange(idx: number, v: string) {
  const next = [...doc.value.Statement]
  next[idx] = { ...next[idx], resources: parseLines(v) }
  doc.value = { ...doc.value, Statement: next }
}

function onField<K extends keyof PolicyStatement>(
  idx: number,
  field: K,
  value: PolicyStatement[K],
) {
  const next = [...doc.value.Statement]
  next[idx] = { ...next[idx], [field]: value }
  doc.value = { ...doc.value, Statement: next }
}

const templateLabels: Record<string, string> = {
  'public-read': 'policy.tplPublicRead',
  'public-read-write': 'policy.tplPublicReadWrite',
  'deny-list': 'policy.tplDenyList',
  clear: 'policy.tplClear',
}
</script>

<template>
  <div>
    <div class="row" style="margin-bottom:10px; gap:6px">
      <button
        class="btn sm"
        :class="{ primary: mode === 'visual' }"
        data-testid="policy-mode-visual"
        @click="mode = 'visual'"
      >
        {{ t('policy.visualMode') }}
      </button>
      <button
        class="btn sm"
        :class="{ primary: mode === 'json' }"
        data-testid="policy-mode-json"
        @click="mode = 'json'"
      >
        {{ t('policy.jsonMode') }}
      </button>
      <span v-if="dirty" class="badge dirty">{{ t('policy.dirty') }}</span>
    </div>

    <div v-if="mode === 'visual'">
      <div v-if="parseError" class="badge error">
        {{ t('policy.parsedFail') }}
      </div>
      <template v-else>
        <div class="badge" style="color:var(--muted)">
          {{ tf('policy.parsedOk', { n: doc.Statement.length }) }}
        </div>

        <fieldset class="tpl-row">
          <legend>{{ t('policy.templates') }}</legend>
          <button
            v-for="tpl in POLICY_TEMPLATES"
            :key="tpl.id"
            class="btn sm"
            type="button"
            @click="applyTemplate(tpl.id)"
          >
            {{ t(templateLabels[tpl.id]) }}
          </button>
        </fieldset>

        <div v-for="(s, i) in doc.Statement" :key="i" class="stmt-card" :data-testid="`policy-stmt-${i}`">
          <div class="row stmt-head">
            <strong>{{ tf('policy.statementN', { n: i + 1 }) }}</strong>
            <button class="btn sm danger" type="button" @click="removeStatement(i)">
              ×
            </button>
          </div>

          <div class="grid-2">
            <label>
              {{ t('policy.effect') }}
              <select
                :value="s.effect"
                @change="onField(i, 'effect', ($event.target as HTMLSelectElement).value as 'Allow' | 'Deny')"
              >
                <option value="Allow">Allow</option>
                <option value="Deny">Deny</option>
              </select>
            </label>
            <label>
              {{ t('policy.sid') }}
              <input
                type="text"
                :value="s.sid ?? ''"
                @input="onField(i, 'sid', ($event.target as HTMLInputElement).value)"
              />
            </label>
          </div>

          <label>
            {{ t('policy.principal') }}
            <input
              type="text"
              :value="s.principal"
              @input="onField(i, 'principal', ($event.target as HTMLInputElement).value)"
            />
          </label>

          <label>
            {{ t('policy.actions') }}
            <textarea
              rows="3"
              :value="toLines(s.actions)"
              spellcheck="false"
              @input="onActionsChange(i, ($event.target as HTMLTextAreaElement).value)"
            ></textarea>
          </label>

          <label>
            {{ t('policy.resources') }}
            <textarea
              rows="3"
              :value="toLines(s.resources)"
              spellcheck="false"
              @input="onResourcesChange(i, ($event.target as HTMLTextAreaElement).value)"
            ></textarea>
          </label>
        </div>

        <button class="btn sm" type="button" @click="addStatement">
          {{ t('policy.addStatement') }}
        </button>

        <div v-if="validationError" class="badge error" style="margin-top:8px">
          {{ validationError }}
        </div>

        <details style="margin-top:10px">
          <summary>{{ t('policy.preview') }}</summary>
          <pre class="mono preview">{{ previewJSON }}</pre>
        </details>
      </template>
    </div>

    <textarea
      v-else
      class="mono policy-area"
      :value="raw"
      spellcheck="false"
      @input="emit('update', ($event.target as HTMLTextAreaElement).value); dirty = true"
    ></textarea>
  </div>
</template>

<style scoped>
.tpl-row {
  border: 1px dashed var(--border);
  border-radius: var(--radius);
  padding: 8px 10px;
  margin: 10px 0;
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  align-items: center;
}
.tpl-row legend {
  padding: 0 6px;
  color: var(--muted);
  font-size: 12px;
}
.stmt-card {
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 10px 12px;
  margin: 10px 0;
  display: grid;
  gap: 8px;
  background: var(--panel-2);
}
.stmt-head {
  justify-content: space-between;
}
.grid-2 {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
}
.badge.dirty {
  color: var(--brand);
  font-size: 12px;
}
.badge.error {
  color: #ef4444;
  font-size: 12px;
}
label {
  display: grid;
  gap: 4px;
  font-size: 12px;
  color: var(--muted);
}
input, select, textarea {
  font-family: var(--font-mono);
  font-size: 13px;
  background: var(--panel);
  color: var(--text);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 6px 8px;
}
textarea {
  resize: vertical;
}
.preview {
  background: var(--panel);
  padding: 10px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  font-size: 12px;
  max-height: 240px;
  overflow: auto;
}
.policy-area {
  width: 100%;
  min-height: 240px;
  margin-top: 10px;
  padding: 10px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  font-family: var(--font-mono);
  font-size: 12px;
  line-height: 1.5;
  background: var(--panel-2);
  color: var(--text);
  resize: vertical;
}
</style>
