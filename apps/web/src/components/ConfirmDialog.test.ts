import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { nextTick, reactive } from 'vue'
import ConfirmDialog from './ConfirmDialog.vue'
import { confirmState, settleConfirm } from '../confirm'

vi.mock('../confirm', () => ({
  confirmState: reactive({
    open: false,
    title: '',
    message: '',
    confirmText: undefined as string | undefined,
    danger: true,
    resolve: null,
  }),
  confirmDialog: vi.fn(async () => true),
  settleConfirm: vi.fn(),
}))

vi.mock('../i18n', () => ({
  t: (k: string) => k,
  tf: (k: string) => k,
  locale: () => 'en-US',
}))

function bodyBtn(text: string): HTMLButtonElement {
  const b = Array.from(document.body.querySelectorAll('button')).find(
    (x) => (x.textContent ?? '').trim() === text,
  )
  expect(b, `body button "${text}"`).toBeTruthy()
  return b as unknown as HTMLButtonElement
}

function clickBody(text: string) {
  bodyBtn(text).dispatchEvent(new MouseEvent('click', { bubbles: true }))
}

afterEach(() => {
  vi.clearAllMocks()
  confirmState.open = false
})

describe('ConfirmDialog', () => {
  it('renders nothing while closed', () => {
    const w = mount(ConfirmDialog, { attachTo: document.body })
    expect(document.body.querySelector('.modal-card')).toBeNull()
    expect(document.body.textContent ?? '').not.toContain('Delete file?')
    w.unmount()
  })

  it('renders title/message and danger confirm button; confirm settles true', async () => {
    const w = mount(ConfirmDialog, { attachTo: document.body })
    confirmState.title = 'Delete file?'
    confirmState.message = 'This cannot be undone.'
    confirmState.danger = true
    confirmState.confirmText = 'Delete'
    confirmState.open = true
    await nextTick()

    const text = document.body.textContent ?? ''
    expect(text).toContain('Delete file?')
    expect(text).toContain('This cannot be undone.')
    expect(document.body.querySelector('.modal-icon.danger')).toBeTruthy()
    expect(document.body.querySelector('[role="alertdialog"]')?.getAttribute('aria-label')).toBe('Delete file?')

    clickBody('Delete')
    expect(settleConfirm).toHaveBeenCalledWith(true)

    clickBody('common.cancel')
    expect(settleConfirm).toHaveBeenLastCalledWith(false)
    w.unmount()
  })

  it('non-danger mode: info icon and default common.ok confirm text', async () => {
    const w = mount(ConfirmDialog, { attachTo: document.body })
    confirmState.danger = false
    confirmState.confirmText = undefined
    confirmState.open = true
    await nextTick()

    expect(document.body.querySelector('.modal-icon.danger')).toBeNull()
    expect(document.body.textContent).toContain('common.ok')
    clickBody('common.ok')
    expect(settleConfirm).toHaveBeenCalledWith(true)
    w.unmount()
  })

  it('danger 且 confirmText 缺省：默认 common.delete 文案', async () => {
    const w = mount(ConfirmDialog, { attachTo: document.body })
    confirmState.danger = true
    confirmState.confirmText = undefined
    confirmState.open = true
    await nextTick()

    expect(document.body.querySelector('.modal-icon.danger')).toBeTruthy()
    expect(document.body.textContent).toContain('common.delete')
    clickBody('common.delete')
    expect(settleConfirm).toHaveBeenCalledWith(true)
    w.unmount()
  })

  it('cancel button and backdrop self-click settle false', async () => {
    const w = mount(ConfirmDialog, { attachTo: document.body })
    confirmState.danger = true
    confirmState.confirmText = 'Go'
    confirmState.open = true
    await nextTick()

    const backdrop = document.body.querySelector('.modal-backdrop') as HTMLElement
    expect(backdrop).toBeTruthy()
    backdrop.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    expect(settleConfirm).toHaveBeenLastCalledWith(false)
    w.unmount()
  })

  it('Escape cancels, Enter confirms via keydown stack', async () => {
    const w = mount(ConfirmDialog, { attachTo: document.body })
    confirmState.open = true
    await nextTick()

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    expect(settleConfirm).toHaveBeenLastCalledWith(false)

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter' }))
    expect(settleConfirm).toHaveBeenLastCalledWith(true)
    w.unmount()
  })

  it('open 时其他按键（非 Escape/Enter）：走 else-if 假分支，不 settle', async () => {
    const w = mount(ConfirmDialog, { attachTo: document.body })
    confirmState.open = true
    await nextTick()
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'z' }))
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'F8' }))
    expect(settleConfirm).not.toHaveBeenCalled()
    w.unmount()
  })

  it('keys ignored while closed', async () => {
    const w = mount(ConfirmDialog, { attachTo: document.body })
    confirmState.open = false
    await nextTick()
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter' }))
    expect(settleConfirm).not.toHaveBeenCalled()
    w.unmount()
  })

  it('关闭瞬间（handler 尚未弹出 keydown 栈）按键走 early-return 被忽略', async () => {
    const w = mount(ConfirmDialog, { attachTo: document.body })
    confirmState.open = true
    await nextTick() // watcher 已 flush：handler 已入栈
    confirmState.open = false
    // 立即派发：open 变化尚未 flush，handler 仍在栈顶 → onKey 读到 closed → early-return
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    expect(settleConfirm).not.toHaveBeenCalled()
    await nextTick() // flush 后 handler 弹出，避免污染后续用例
    w.unmount()
  })

  it('打开后焦点移入确认按钮，关闭后恢复到打开前的元素', async () => {
    const outside = document.createElement('button')
    outside.textContent = 'outside'
    document.body.appendChild(outside)
    outside.focus()

    const w = mount(ConfirmDialog, { attachTo: document.body })
    confirmState.confirmText = 'Delete'
    confirmState.open = true
    await flushPromises()
    expect(document.activeElement).toBe(bodyBtn('Delete'))

    confirmState.open = false
    await flushPromises()
    expect(document.activeElement).toBe(outside)
    outside.remove()
    w.unmount()
  })

  it('Tab 焦点陷阱：首尾回卷、中间不干预，焦点不逃出对话框', async () => {
    const w = mount(ConfirmDialog, { attachTo: document.body })
    confirmState.confirmText = 'Delete'
    confirmState.open = true
    await flushPromises()
    const confirmBtn = bodyBtn('Delete')
    const cancelBtn = bodyBtn('common.cancel')
    expect(document.activeElement).toBe(confirmBtn)

    // Shift+Tab 在第一个元素上 → 回卷到最后一个
    let ev = new KeyboardEvent('keydown', { key: 'Tab', shiftKey: true, cancelable: true })
    window.dispatchEvent(ev)
    expect(ev.defaultPrevented).toBe(true)
    expect(document.activeElement).toBe(cancelBtn)

    // Tab 在最后一个元素上 → 回卷到第一个
    ev = new KeyboardEvent('keydown', { key: 'Tab', cancelable: true })
    window.dispatchEvent(ev)
    expect(ev.defaultPrevented).toBe(true)
    expect(document.activeElement).toBe(confirmBtn)

    // Tab 在第一个元素上（不带 shift）→ 透传，不回卷
    ev = new KeyboardEvent('keydown', { key: 'Tab', cancelable: true })
    window.dispatchEvent(ev)
    expect(ev.defaultPrevented).toBe(false)
    expect(document.activeElement).toBe(confirmBtn)

    // Shift+Tab 在最后一个元素上 → 透传
    cancelBtn.focus()
    ev = new KeyboardEvent('keydown', { key: 'Tab', shiftKey: true, cancelable: true })
    window.dispatchEvent(ev)
    expect(ev.defaultPrevented).toBe(false)
    expect(document.activeElement).toBe(cancelBtn)

    confirmState.open = false
    await flushPromises()
    w.unmount()
  })

  it('打开后同 tick 内卸载：初始聚焦的续体遇到已置空的容器不抛错、焦点不被劫持', async () => {
    const outside = document.createElement('button')
    outside.textContent = 'outside'
    document.body.appendChild(outside)
    outside.focus()

    const w = mount(ConfirmDialog, { attachTo: document.body })
    confirmState.open = true
    await nextTick() // 打开 watcher 已运行并挂起在 await nextTick()
    w.unmount() // 卡片随组件卸载，模板 ref 置空
    await flushPromises() // 续体恢复 → 走空容器守卫（无 unhandled rejection）
    expect(document.activeElement).toBe(outside)
    outside.remove()
  })
})
