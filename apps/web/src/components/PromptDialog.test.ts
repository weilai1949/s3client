import { afterEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
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
})
