<script setup lang="ts">
import { computed, ref } from 'vue'
import { promptState, settlePrompt } from '../prompt'
import { useFocusTrap } from '../composables/useFocusTrap'
import { useKeydownStack } from '../composables/useKeydownStack'
import { t } from '../i18n'

const card = ref<HTMLElement>()
const inputEl = ref<HTMLInputElement>()
const open = computed(() => promptState.open)

const { trapTab } = useFocusTrap(card, open, {
  // 初始聚焦落在输入框（它是第一个可聚焦元素）；聚焦后全选，便于直接替换掉默认值。
  afterOpen: () => inputEl.value?.select(),
})

function onKey(e: KeyboardEvent) {
  if (!promptState.open) return
  if (e.key === 'Escape') {
    e.preventDefault()
    settlePrompt(false)
  } else if (e.key === 'Enter') {
    e.preventDefault()
    settlePrompt(true)
  } else {
    trapTab(e)
  }
}

useKeydownStack(onKey, open)
</script>

<template>
  <Teleport to="body">
    <Transition name="modal-fade">
      <div v-if="promptState.open" class="modal-backdrop" @click.self="settlePrompt(false)">
        <div ref="card" class="modal-card" role="dialog" aria-modal="true" :aria-label="promptState.title">
          <h3 class="modal-title">{{ promptState.title }}</h3>
          <label class="field">
            {{ promptState.label }}
            <input
              ref="inputEl"
              v-model="promptState.value"
              type="text"
              :placeholder="promptState.placeholder"
              autocomplete="off"
              spellcheck="false"
              @keydown.enter.prevent="settlePrompt(true)"
            />
          </label>
          <p v-if="promptState.error" class="modal-err" role="alert">{{ promptState.error }}</p>
          <div class="modal-actions">
            <button class="btn sm" @click="settlePrompt(true)">{{ promptState.confirmText }}</button>
            <button class="btn secondary sm" @click="settlePrompt(false)">{{ t('common.cancel') }}</button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.modal-backdrop {
  position: fixed; inset: 0; z-index: 300;
  background: rgba(8, 14, 11, .45);
  backdrop-filter: blur(3px);
  display: flex; align-items: center; justify-content: center;
  padding: 20px;
}
.modal-card {
  width: min(420px, 100%);
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: var(--radius-xl);
  padding: 24px;
  box-shadow: var(--shadow-lg);
}
.modal-title { margin: 0 0 14px; font-size: 16px; font-weight: 700; }
.modal-err { margin: 8px 0 0; color: var(--danger); font-size: 12px; }
.modal-actions { display: flex; justify-content: flex-end; gap: 10px; margin-top: 18px; }

.modal-fade-enter-active, .modal-fade-leave-active { transition: opacity .16s ease; }
.modal-fade-enter-active .modal-card, .modal-fade-leave-active .modal-card { transition: transform .16s ease; }
.modal-fade-enter-from, .modal-fade-leave-to { opacity: 0; }
.modal-fade-enter-from .modal-card, .modal-fade-leave-to .modal-card { transform: scale(.96) translateY(6px); }
</style>
