<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
import { t } from '../i18n'
import { useKeydownStack } from '../composables/useKeydownStack'
import type { Entry } from '../types'

const props = defineProps<{
  menu: { x: number; y: number; entry: Entry } | null
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'open'): void
  (e: 'preview'): void
  (e: 'copy-link'): void
  (e: 'copy-key'): void
  (e: 'copy-folder'): void
  (e: 'move-folder'): void
  (e: 'delete-folder'): void
  (e: 'copy-file'): void
  (e: 'move-file'): void
  (e: 'rename'): void
  (e: 'detail'): void
  (e: 'acl'): void
  (e: 'tags'): void
  (e: 'versions'): void
  (e: 'delete'): void
}>()

const menuEl = ref<HTMLElement>()

// 打开前的焦点元素：关闭时还原，避免键盘用户「菜单一关心就丢焦点」（KNOWN_ISSUES #77）。
let previousFocus: HTMLElement | null = null

// 打开时记录来源焦点并聚焦第一项；关闭时还原焦点。方向键在菜单内循环导航。
watch(
  () => props.menu,
  async (m) => {
    if (m) {
      const active = document.activeElement
      previousFocus = active instanceof HTMLElement ? active : null
      await nextTick()
      menuEl.value?.querySelector<HTMLElement>('button')?.focus()
    } else {
      const target = previousFocus
      previousFocus = null
      if (target && target.isConnected) target.focus()
    }
  },
)

// Escape 走全局 LIFO 键栈（KNOWN_ISSUES #77）：仅当菜单是最上层时接收 Escape，
// 不会越过在其之上打开的对话框——旧实现用独立 window 监听无条件关菜单，破坏 LIFO。
useKeydownStack(
  (e) => {
    if (e.key === 'Escape') {
      e.preventDefault()
      emit('close')
    }
  },
  () => props.menu !== null,
)

function onMenuKeydown(e: KeyboardEvent) {
  // 模板 v-if="menu" 已保证 menuEl 非空、菜单恒含按钮；无需判空守卫。
  const btns = Array.from(menuEl.value!.querySelectorAll<HTMLElement>('button'))
  const idx = btns.indexOf(document.activeElement as HTMLElement)
  if (e.key === 'ArrowDown') { e.preventDefault(); btns[(idx + 1) % btns.length].focus() }
  else if (e.key === 'ArrowUp') { e.preventDefault(); btns[(idx - 1 + btns.length) % btns.length].focus() }
  else if (e.key === 'Home') { e.preventDefault(); btns[0].focus() }
  else if (e.key === 'End') { e.preventDefault(); btns[btns.length - 1].focus() }
}

function menuStyle() {
  // 模板仅在 v-if="menu" 时调用，m 恒非空。
  const m = props.menu!
  return {
    left: Math.max(8, Math.min(m.x, window.innerWidth - 200)) + 'px',
    // 靠近底部时上移，保证菜单项在视口内；菜单自身另有 max-height + 滚动兜底。
    top: Math.max(8, Math.min(m.y, window.innerHeight - 240)) + 'px',
  }
}
</script>

<template>
  <Teleport to="body">
    <div v-if="menu" ref="menuEl" class="ctx-menu" :style="menuStyle()" role="menu" tabindex="-1" @click.stop @keydown="onMenuKeydown">
      <template v-if="menu.entry.kind === 'folder'">
        <button role="menuitem" @click="emit('open')">{{ t('ctx.openFolder') }}</button>
        <button role="menuitem" @click="emit('copy-key')">{{ t('ctx.copyPath') }}</button>
        <div class="ctx-sep" />
        <button role="menuitem" @click="emit('copy-folder')">{{ t('toolbar.copyTo') }}</button>
        <button role="menuitem" @click="emit('move-folder')">{{ t('toolbar.moveTo') }}</button>
        <button role="menuitem" class="danger" @click="emit('delete-folder')">{{ t('ctx.deleteFolder') }}</button>
      </template>
      <template v-else>
        <button role="menuitem" @click="emit('open')">{{ t('common.download') }}</button>
        <button role="menuitem" @click="emit('preview')">{{ t('objects.preview') }}</button>
        <button role="menuitem" @click="emit('copy-link')">{{ t('ctx.copySignedLink') }}</button>
        <button role="menuitem" @click="emit('copy-key')">{{ t('ctx.copyKey') }}</button>
        <div class="ctx-sep" />
        <button role="menuitem" @click="emit('copy-file')">{{ t('toolbar.copyTo') }}</button>
        <button role="menuitem" @click="emit('move-file')">{{ t('toolbar.moveTo') }}</button>
        <button role="menuitem" @click="emit('rename')">{{ t('common.rename') }}</button>
        <button role="menuitem" @click="emit('detail')">{{ t('ctx.detail') }}</button>
        <button role="menuitem" @click="emit('acl')">{{ t('ctx.acl') }}</button>
        <button role="menuitem" @click="emit('tags')">{{ t('ctx.tags') }}</button>
        <button role="menuitem" @click="emit('versions')">{{ t('ctx.versions') }}</button>
        <div class="ctx-sep" />
        <button role="menuitem" class="danger" @click="emit('delete')">{{ t('common.delete') }}</button>
      </template>
    </div>
  </Teleport>
</template>

<style scoped>
/* 右键菜单 */
.ctx-menu {
  position: fixed; z-index: 150;
  min-width: 190px;
  max-height: calc(100vh - 16px);
  overflow-y: auto;
  padding: 6px;
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-lg);
  animation: pop-in .12s ease;
  display: flex; flex-direction: column;
}
.ctx-menu button {
  display: flex; align-items: center; gap: 8px;
  padding: 8px 12px;
  border: none; border-radius: var(--radius-sm);
  background: none;
  color: var(--text); font-size: 13px; font-weight: 500;
  cursor: pointer; text-align: left;
  white-space: nowrap;
}
.ctx-menu button:hover { background: var(--row-hover); color: var(--primary); }
.ctx-menu button.danger { color: var(--danger); }
.ctx-menu button.danger:hover { background: var(--danger-bg); color: var(--danger); }
.ctx-sep { height: 1px; margin: 5px 8px; background: var(--border); }
</style>
