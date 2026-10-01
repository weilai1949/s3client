<script setup lang="ts">
import { ref, toRef, watch } from 'vue'
import { useFocusTrap } from '../composables/useFocusTrap'
import { useKeydownStack } from '../composables/useKeydownStack'
import { t } from '../i18n'

const props = defineProps<{
  open: boolean
  title: string
  /** 卡片宽度（CSS 值），默认 min(560px, 100%) */
  width?: string
}>()

const emit = defineEmits<{ (e: 'close'): void }>()

const card = ref<HTMLElement>()
const { trapTab } = useFocusTrap(card, toRef(props, 'open'))

let previousOverflow = ''

// 打开时锁定滚动、关闭时还原（焦点的保存 / 恢复与初始聚焦由 useFocusTrap 负责）。
watch(() => props.open, (o) => {
  if (o) {
    previousOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
  } else {
    document.body.style.overflow = previousOverflow
  }
})

function onKey(e: KeyboardEvent) {
  if (!props.open) return
  if (e.key === 'Escape') {
    e.preventDefault()
    emit('close')
    return
  }
  trapTab(e)
}

useKeydownStack(onKey, toRef(props, 'open'))
</script>

<template>
  <Teleport to="body">
    <Transition name="modal-fade">
      <div v-if="open" class="dlg-backdrop" @click.self="emit('close')">
        <div
          ref="card"
          class="dlg-card"
          :style="{ width: width ?? 'min(560px, 100%)' }"
          role="dialog"
          aria-modal="true"
          :aria-label="title"
          tabindex="-1"
        >
          <div class="dlg-head">
            <h3 class="dlg-title">{{ title }}</h3>
            <button class="dlg-x" :aria-label="t('common.close')" @click="emit('close')">✕</button>
          </div>
          <div class="dlg-body">
            <slot />
          </div>
          <div v-if="$slots.footer" class="dlg-footer">
            <slot name="footer" />
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.dlg-backdrop {
  position: fixed; inset: 0; z-index: 200;
  background: rgba(8, 14, 11, .45);
  backdrop-filter: blur(3px);
  display: flex; align-items: center; justify-content: center;
  padding: 20px;
}
.dlg-card {
  max-width: 100%;
  max-height: 88vh;
  display: flex; flex-direction: column;
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: var(--radius-xl);
  box-shadow: var(--shadow-lg);
  overflow: hidden;
}
.dlg-head {
  display: flex; align-items: center; justify-content: space-between; gap: 12px;
  padding: 14px 18px 0;
}
.dlg-title { margin: 0; font-size: 16px; font-weight: 700; }
.dlg-x {
  flex: none;
  width: 28px; height: 28px;
  border: none; border-radius: 8px;
  background: none;
  color: var(--muted); font-size: 14px;
  cursor: pointer;
  transition: all .15s ease;
}
.dlg-x:hover { background: var(--row-hover); color: var(--text); }
.dlg-body { padding: 14px 18px 18px; overflow: auto; }
.dlg-footer {
  display: flex; justify-content: flex-end; gap: 10px;
  padding: 12px 18px 16px;
  border-top: 1px solid var(--border);
}

.modal-fade-enter-active, .modal-fade-leave-active { transition: opacity .16s ease; }
.modal-fade-enter-active .dlg-card, .modal-fade-leave-active .dlg-card { transition: transform .16s ease; }
.modal-fade-enter-from, .modal-fade-leave-to { opacity: 0; }
.modal-fade-enter-from .dlg-card, .modal-fade-leave-to .dlg-card { transform: scale(.96) translateY(6px); }
</style>
