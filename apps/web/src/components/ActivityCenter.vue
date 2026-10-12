<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { activities } from '../store'
import { t, tf } from '../i18n'
import { useKeydownStack } from '../composables/useKeydownStack'

/**
 * 顶栏「进行中的任务」角标。
 *
 * 存在理由：上传/迁移的状态机挂在面板组件上（KeepAlive 缓存）。用户切到别的页面后
 * 任务仍在推进，但界面上看不到任何进度——把活动快照提到 store 后，这里提供一个
 * 全局可见的进度出口（含展开明细）。无活动时整体不渲染，不占顶栏空间。
 */
const list = computed(() => Object.values(activities).filter((a) => a.total > 0))

/** 整体百分比：多条任务取平均，只用于角标一句话概览。
 *  模板的 v-if 已保证 list 非空（空即不渲染），这里无需判空守卫。 */
const overallPct = computed(() => Math.round(list.value.reduce((s, a) => s + a.pct, 0) / list.value.length))

const open = ref(false)
const root = ref<HTMLElement>()

function onDocClick(e: MouseEvent) {
  if (!open.value) return
  if (root.value?.contains(e.target as Node)) return
  open.value = false
}

useKeydownStack((e) => {
  if (e.key !== 'Escape') return
  e.preventDefault()
  open.value = false
}, open)

onMounted(() => document.addEventListener('click', onDocClick))
onBeforeUnmount(() => document.removeEventListener('click', onDocClick))
</script>

<template>
  <div v-if="list.length" ref="root" class="activity">
    <button
      class="theme-btn activity-btn"
      :title="t('activity.title')"
      :aria-label="tf('activity.aria', { n: list.length })"
      aria-haspopup="true"
      :aria-expanded="open"
      @click="open = !open"
    >
      <span class="act-spinner" aria-hidden="true" />
      <span class="act-pct">{{ overallPct }}%</span>
    </button>
    <div v-if="open" class="activity-pop" role="region" :aria-label="t('activity.title')">
      <div class="act-head">{{ t('activity.title') }}</div>
      <div v-for="a in list" :key="a.key" class="act-item">
        <div class="act-name">{{ a.label }}</div>
        <div class="progress" role="progressbar" :aria-valuenow="a.pct" aria-valuemin="0" aria-valuemax="100">
          <div class="bar" :style="{ width: a.pct + '%' }" />
        </div>
        <div class="badge">{{ a.done }}/{{ a.total }} · {{ a.pct }}%</div>
      </div>
      <div class="act-hint">{{ t('activity.hint') }}</div>
    </div>
  </div>
</template>

<style scoped>
.activity { position: relative; }
.activity-btn { width: auto; gap: 6px; padding: 0 10px; }
.act-spinner {
  width: 12px; height: 12px; flex: none;
  border: 2px solid var(--border-strong);
  border-top-color: var(--primary);
  border-radius: 50%;
  animation: act-spin .8s linear infinite;
}
.act-pct { font-size: 12px; font-weight: 600; }
@keyframes act-spin { to { transform: rotate(360deg); } }

.activity-pop {
  position: absolute; right: 0; top: calc(100% + 8px); z-index: 60;
  width: min(320px, 78vw);
  padding: 12px 14px;
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-lg);
  animation: pop-in .12s ease;
}
.act-head { font-size: 13px; font-weight: 700; margin-bottom: 10px; }
.act-item + .act-item { margin-top: 10px; }
.act-name { font-size: 12px; color: var(--text); margin-bottom: 5px; }
.act-hint { margin-top: 10px; font-size: 11px; color: var(--muted); }
</style>
