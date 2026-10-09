<script setup lang="ts">
// SchedulesSection —— 计划任务（ROADMAP #6：cron 定时增量备份）。
// 挂在 MigratePanel 内：计划自带源/目标账号引用，不依赖面板当前选中的源账号，
// 因此渲染在「需要先选账号」的空态之外（始终可见）。增删改查 + 立即运行全部
// 走 s3api 的 schedules 端点，运行态（下次/上次运行、错误）由后端维护。
defineOptions({ name: 'SchedulesSection' })

import { onMounted, reactive, ref } from 'vue'
import { toErrorMessage } from '../errors'
import { s3api } from '../api'
import { state, toast } from '../store'
import { confirmDialog } from '../confirm'
import { t, tf } from '../i18n'
import type { Schedule, ScheduleInput } from '../types'

const schedules = ref<Schedule[]>([])
const loading = ref(false)
const loadError = ref('')
const busy = ref(false)

// 表单状态：editing 为 null 表示新建，非空表示编辑（保留 id/createdAt 由后端负责）。
const formOpen = ref(false)
const editing = ref<Schedule | null>(null)
const formError = ref('')
const form = reactive({
  sourceAccountId: '',
  sourceBucket: '',
  sourcePrefix: '',
  targetAccountId: '',
  targetBucket: '',
  targetPrefix: '',
  mode: 'etag' as ScheduleInput['mode'],
  cron: '',
  enabled: true,
})

async function load() {
  loading.value = true
  loadError.value = ''
  try {
    const res = await s3api.listSchedules()
    schedules.value = res.schedules ?? []
  } catch (e) {
    loadError.value = toErrorMessage(e)
  } finally {
    loading.value = false
  }
}

function resetForm() {
  form.sourceAccountId = ''
  form.sourceBucket = ''
  form.sourcePrefix = ''
  form.targetAccountId = ''
  form.targetBucket = ''
  form.targetPrefix = ''
  form.mode = 'etag'
  form.cron = ''
  form.enabled = true
  formError.value = ''
}

function openCreate() {
  editing.value = null
  resetForm()
  formOpen.value = true
}

function openEdit(s: Schedule) {
  editing.value = s
  form.sourceAccountId = s.sourceAccountId
  form.sourceBucket = s.sourceBucket
  form.sourcePrefix = s.sourcePrefix ?? ''
  form.targetAccountId = s.targetAccountId
  form.targetBucket = s.targetBucket
  form.targetPrefix = s.targetPrefix ?? ''
  form.mode = s.mode
  form.cron = s.cron
  form.enabled = s.enabled
  formError.value = ''
  formOpen.value = true
}

function closeForm() {
  formOpen.value = false
  editing.value = null
}

async function save() {
  formError.value = ''
  if (!form.sourceAccountId || !form.targetAccountId) {
    formError.value = t('schedule.accountRequired')
    return
  }
  if (!form.cron.trim()) {
    formError.value = t('schedule.cronRequired')
    return
  }
  const body: ScheduleInput = {
    sourceAccountId: form.sourceAccountId,
    sourceBucket: form.sourceBucket,
    sourcePrefix: form.sourcePrefix,
    targetAccountId: form.targetAccountId,
    targetBucket: form.targetBucket,
    targetPrefix: form.targetPrefix,
    mode: form.mode,
    cron: form.cron,
    enabled: form.enabled,
  }
  busy.value = true
  try {
    if (editing.value) {
      await s3api.updateSchedule(editing.value.id, body)
      toast(t('schedule.updated'))
    } else {
      await s3api.createSchedule(body)
      toast(t('schedule.created'))
    }
    closeForm()
    await load()
  } catch (e) {
    // 400（cron 非法 / 永不触发 / mode 非法）、404（账号不存在）、403（作用域）等
    // 具名错误都回显在表单内联横幅（role=alert），避免只进 toast 被错过。
    formError.value = toErrorMessage(e)
  } finally {
    busy.value = false
  }
}

async function toggle(s: Schedule) {
  busy.value = true
  try {
    await s3api.updateSchedule(s.id, {
      sourceAccountId: s.sourceAccountId,
      sourceBucket: s.sourceBucket,
      sourcePrefix: s.sourcePrefix,
      targetAccountId: s.targetAccountId,
      targetBucket: s.targetBucket,
      targetPrefix: s.targetPrefix,
      mode: s.mode,
      cron: s.cron,
      enabled: !s.enabled,
    })
    toast(t('schedule.toggled'))
    await load()
  } catch (e) {
    toast(toErrorMessage(e), 'err')
  } finally {
    busy.value = false
  }
}

async function runNow(s: Schedule) {
  busy.value = true
  try {
    const res = await s3api.runScheduleNow(s.id)
    toast(tf('schedule.runStarted', { id: res.jobId }))
    await load()
  } catch (e) {
    toast(toErrorMessage(e), 'err')
  } finally {
    busy.value = false
  }
}

async function remove(s: Schedule) {
  const route = `${s.sourceBucket}/${s.targetBucket}`
  const ok = await confirmDialog({ title: t('schedule.delete'), message: tf('schedule.confirmDelete', { route }) })
  if (!ok) return
  busy.value = true
  try {
    await s3api.deleteSchedule(s.id)
    toast(t('schedule.deleted'))
    await load()
  } catch (e) {
    toast(toErrorMessage(e), 'err')
  } finally {
    busy.value = false
  }
}

onMounted(load)
</script>

<template>
  <section class="schedules" :aria-label="t('schedule.title')">
    <div class="sched-head">
      <h3 style="margin:0">{{ t('schedule.title') }}</h3>
      <span class="muted" style="font-size:12px">{{ t('schedule.hint') }}</span>
      <span class="spacer" />
      <button class="btn secondary sm" :disabled="loading" @click="load">
        {{ loading ? t('schedule.loading') : t('schedule.refresh') }}
      </button>
      <button class="btn sm" @click="openCreate">{{ t('schedule.add') }}</button>
    </div>

    <div v-if="loadError" class="msg err" role="alert">{{ loadError }}</div>

    <!-- 新建 / 编辑表单 -->
    <form v-if="formOpen" class="sched-form" @submit.prevent="save">
      <div class="badge">{{ editing ? t('schedule.editTitle') : t('schedule.newTitle') }}</div>
      <div class="toolbar">
        <label class="field">
          {{ t('schedule.sourceAccount') }}
          <select v-model="form.sourceAccountId" required>
            <option v-for="a in state.accounts" :key="a.id" :value="a.id">{{ a.name }}</option>
          </select>
        </label>
        <label class="field">
          {{ t('schedule.sourceBucket') }}
          <input v-model="form.sourceBucket" />
        </label>
        <label class="field">
          {{ t('schedule.sourcePrefix') }}
          <input v-model="form.sourcePrefix" :placeholder="t('migrate.prefixPlaceholder')" />
        </label>
      </div>
      <div class="toolbar">
        <label class="field">
          {{ t('schedule.targetAccount') }}
          <select v-model="form.targetAccountId" required>
            <option v-for="a in state.accounts" :key="a.id" :value="a.id">{{ a.name }}</option>
          </select>
        </label>
        <label class="field">
          {{ t('schedule.targetBucket') }}
          <input v-model="form.targetBucket" />
        </label>
        <label class="field">
          {{ t('schedule.targetPrefix') }}
          <input v-model="form.targetPrefix" :placeholder="t('migrate.prefixPlaceholder')" />
        </label>
      </div>
      <div class="toolbar">
        <label class="field">
          {{ t('schedule.mode') }}
          <select v-model="form.mode">
            <option value="etag">{{ t('schedule.modeEtag') }}</option>
            <option value="size_mtime">{{ t('schedule.modeSize') }}</option>
            <option value="always">{{ t('schedule.modeAlways') }}</option>
          </select>
        </label>
        <label class="field">
          {{ t('schedule.cron') }}
          <input v-model="form.cron" :placeholder="t('schedule.cronPh')" required />
        </label>
        <label class="field">
          <input v-model="form.enabled" type="checkbox" />
          {{ t('schedule.enabled') }}
        </label>
      </div>
      <div class="badge" style="font-size:12px">{{ t('schedule.cronHint') }}</div>
      <div v-if="formError" class="msg err" role="alert">{{ formError }}</div>
      <div class="row">
        <button class="btn sm" type="submit" :disabled="busy">{{ editing ? t('schedule.save') : t('schedule.create') }}</button>
        <button class="btn secondary sm" type="button" @click="closeForm">{{ t('schedule.cancel') }}</button>
      </div>
    </form>

    <div v-if="!loading && !schedules.length && !loadError" class="empty">{{ t('schedule.empty') }}</div>

    <div v-else-if="schedules.length" class="tbl-wrap">
      <table class="tbl">
        <caption class="sr-only">{{ t('schedule.tableAria') }}</caption>
        <thead>
          <tr>
            <th>{{ t('schedule.route') }}</th>
            <th>{{ t('schedule.cron') }}</th>
            <th>{{ t('schedule.mode') }}</th>
            <th>{{ t('schedule.enabled') }}</th>
            <th>{{ t('schedule.nextRun') }}</th>
            <th>{{ t('schedule.lastRun') }}</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="s in schedules" :key="s.id">
            <td class="mono">{{ tf('schedule.routeArrow', { src: s.sourceBucket, dst: s.targetBucket }) }}</td>
            <td class="mono">{{ s.cron }}</td>
            <td>{{ s.mode }}</td>
            <td>
              <span class="tag" :class="s.enabled ? 'ok' : ''">{{ s.enabled ? t('schedule.statusOn') : t('schedule.statusOff') }}</span>
            </td>
            <td class="muted">{{ s.nextRunAt }}</td>
            <td class="muted">
              <template v-if="s.lastRunAt">{{ s.lastRunAt }}</template>
              <template v-else>{{ t('schedule.never') }}</template>
              <div v-if="s.lastError" class="msg err" role="alert">{{ s.lastError }}</div>
            </td>
            <td>
              <div class="row">
                <button class="btn secondary sm" :disabled="busy" @click="runNow(s)">{{ t('schedule.run') }}</button>
                <button class="btn secondary sm" :disabled="busy" :aria-label="t('schedule.toggle')" @click="toggle(s)">
                  {{ s.enabled ? t('schedule.statusOff') : t('schedule.statusOn') }}
                </button>
                <button class="btn secondary sm" :disabled="busy" @click="openEdit(s)">{{ t('schedule.save') }}</button>
                <button class="btn secondary sm" :disabled="busy" @click="remove(s)">{{ t('schedule.delete') }}</button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>

<style scoped>
.schedules {
  margin-top: 14px;
  border-top: 1px solid var(--border);
  padding-top: 10px;
}
.sched-head {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  margin-bottom: 8px;
}
.sched-form {
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--panel-2);
  padding: 10px 12px;
  margin-bottom: 10px;
}
</style>
