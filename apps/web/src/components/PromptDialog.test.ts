import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { nextTick, reactive } from 'vue'
import PromptDialog from './PromptDialog.vue'
import { promptState, settlePrompt } from '../prompt'

vi.mock('../prompt', () => ({
  promptState: reactive({
    open: false,
    title: '',
    label: '',
    value: '',
    placeholder: '',
    confirmText: '',
    error: '',
    resolve: null,
    validate: undefined,
  }),
  promptDialog: vi.fn(async () => 'x'),
  settlePrompt: vi.fn(),
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

function inputEl(): HTMLInputElement {
  const el = document.body.querySelector('input[type="text"]') as HTMLInputElement | null
  expect(el).toBeTruthy()
  return el!
}

let mounted: Array<{ unmount: () => void }> = []
function track<T extends { unmount: () => void }>(w: T): T {
  mounted.push(w)
  return w
}

afterEach(() => {
  for (const m of mounted) m.unmount()
  mounted = []
  document.body.innerHTML = ''
})
afterEach(() => {
  vi.clearAllMocks()
  promptState.open = false
  promptState.error = ''
})

describe('PromptDialog', () => {
  it('renders nothing while closed', () => {
    track(mount(PromptDialog, { attachTo: document.body }))
    expect(document.body.querySelector('.modal-card')).toBeNull()
  })

  it('renders title, label, placeholder, value, confirm text and error', async () => {
    track(mount(PromptDialog, { attachTo: document.body }))
    promptState.title = 'Rename'
    promptState.label = 'New name'
    promptState.placeholder = 'type here'
    promptState.value = 'old.txt'
    promptState.confirmText = 'Rename now'
    promptState.error = 'invalid'
    promptState.open = true
    await nextTick()

    const text = document.body.textContent ?? ''
    expect(text).toContain('Rename')
    expect(text).toContain('New name')
    expect(text).toContain('invalid')
    expect(document.body.querySelector('.modal-err')?.textContent).toBe('invalid')
    expect(document.body.querySelector('[role="dialog"]')?.getAttribute('aria-label')).toBe('Rename')
    expect(inputEl().value).toBe('old.txt')
    expect(inputEl().placeholder).toBe('type here')
    expect(bodyBtn('Rename now')).toBeTruthy()
  })

  it('typing updates promptState.value via v-model', async () => {
    track(mount(PromptDialog, { attachTo: document.body }))
    promptState.open = true
    await nextTick()
    inputEl().value = 'typed!'
    inputEl().dispatchEvent(new Event('input'))
    await nextTick()
    expect(promptState.value).toBe('typed!')
  })

  it('confirm and cancel settle the prompt', async () => {
    track(mount(PromptDialog, { attachTo: document.body }))
    promptState.open = true
    await nextTick()
    clickBody('common.cancel')
    expect(settlePrompt).toHaveBeenLastCalledWith(false)
    // settlePrompt 不改变 open — 直接重新点击确认（mock 无默认 confirmText，需显式设置）
    promptState.confirmText = 'common.ok'
    promptState.open = true
    await nextTick()
    clickBody('common.ok')
    expect(settlePrompt).toHaveBeenLastCalledWith(true)
  })

  it('Enter in input and Escape via keydown stack settle accordingly', async () => {
    track(mount(PromptDialog, { attachTo: document.body }))
    promptState.open = true
    await nextTick()
    inputEl().dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter' }))
    expect(settlePrompt).toHaveBeenLastCalledWith(true)

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    expect(settlePrompt).toHaveBeenLastCalledWith(false)

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter' }))
    expect(settlePrompt).toHaveBeenLastCalledWith(true)
  })

  it('backdrop self-click cancels', async () => {
    track(mount(PromptDialog, { attachTo: document.body }))
    promptState.open = true
    await nextTick()
    const backdrop = document.body.querySelector('.modal-backdrop') as HTMLElement
    backdrop.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    expect(settlePrompt).toHaveBeenLastCalledWith(false)
  })

  it('keys are ignored while closed', async () => {
    track(mount(PromptDialog, { attachTo: document.body }))
    await nextTick()
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter' }))
    expect(settlePrompt).not.toHaveBeenCalled()
  })

  it('open 时其他按键（非 Escape/Enter）不触发 settle', async () => {
    track(mount(PromptDialog, { attachTo: document.body }))
    promptState.open = true
    await nextTick()
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'z' }))
    expect(settlePrompt).not.toHaveBeenCalled()
  })

  it('关闭瞬间（handler 尚未弹出 keydown 栈）按键走 early-return 被忽略', async () => {
    track(mount(PromptDialog, { attachTo: document.body }))
    promptState.open = true
    await nextTick() // watcher 已 flush：handler 已入栈
    promptState.open = false
    // 立即派发：open 变化尚未 flush，handler 仍在栈顶 → onKey 读到 closed → early-return
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter' }))
    expect(settlePrompt).not.toHaveBeenCalled()
    await nextTick() // flush 后 handler 弹出，避免污染后续用例
  })

  it('打开后焦点移入输入框并全选，关闭后恢复到打开前的元素', async () => {
    const outside = document.createElement('button')
    outside.textContent = 'outside'
    document.body.appendChild(outside)
    outside.focus()

    track(mount(PromptDialog, { attachTo: document.body }))
    promptState.value = 'old.txt'
    promptState.confirmText = 'common.ok'
    promptState.open = true
    await flushPromises()
    const input = inputEl()
    expect(document.activeElement).toBe(input)
    expect(input.selectionStart).toBe(0)
    expect(input.selectionEnd).toBe('old.txt'.length)

    promptState.open = false
    await flushPromises()
    expect(document.activeElement).toBe(outside)
    outside.remove()
  })

  it('Tab 焦点陷阱：首尾回卷、中间不干预，焦点不逃出对话框', async () => {
    track(mount(PromptDialog, { attachTo: document.body }))
    promptState.confirmText = 'common.ok'
    promptState.open = true
    await flushPromises()
    const input = inputEl()
    const confirmBtn = bodyBtn('common.ok')
    const cancelBtn = bodyBtn('common.cancel')
    expect(document.activeElement).toBe(input)

    // Shift+Tab 在第一个元素上 → 回卷到最后一个
    let ev = new KeyboardEvent('keydown', { key: 'Tab', shiftKey: true, cancelable: true })
    window.dispatchEvent(ev)
    expect(ev.defaultPrevented).toBe(true)
    expect(document.activeElement).toBe(cancelBtn)

    // Tab 在最后一个元素上 → 回卷到第一个
    ev = new KeyboardEvent('keydown', { key: 'Tab', cancelable: true })
    window.dispatchEvent(ev)
    expect(ev.defaultPrevented).toBe(true)
    expect(document.activeElement).toBe(input)

    // 中间位置（确认按钮）不干预
    confirmBtn.focus()
    ev = new KeyboardEvent('keydown', { key: 'Tab', cancelable: true })
    window.dispatchEvent(ev)
    expect(ev.defaultPrevented).toBe(false)
    expect(document.activeElement).toBe(confirmBtn)

    promptState.open = false
    await flushPromises()
  })

  it('校验失败文案带 role="alert"：内联失败横幅进 live region 播报', async () => {
    track(mount(PromptDialog, { attachTo: document.body }))
    promptState.error = 'invalid name'
    promptState.open = true
    await flushPromises()
    const err = document.body.querySelector('.modal-err')
    expect(err, '应渲染校验失败文案').toBeTruthy()
    expect(err!.getAttribute('role')).toBe('alert')
    expect(err!.textContent).toBe('invalid name')
  })
})
