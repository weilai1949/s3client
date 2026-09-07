import { describe, expect, it, beforeEach, vi } from 'vitest'
import { promptState, promptDialog, settlePrompt } from './prompt'

vi.mock('./i18n', () => ({
  t: vi.fn((k: string) => k),
}))

beforeEach(() => {
  promptState.open = false
  promptState.resolve = null
  promptState.title = ''
  promptState.label = ''
  promptState.initial = ''
  promptState.placeholder = ''
  promptState.confirmText = ''
  promptState.validate = undefined
  promptState.value = ''
  promptState.error = ''
})

describe('promptDialog', () => {
  it('opens dialog and returns promise', async () => {
    const p = promptDialog({ title: 'Enter name' })
    expect(promptState.open).toBe(true)
    expect(promptState.title).toBe('Enter name')
    expect(promptState.value).toBe('')
    settlePrompt(false)
    await expect(p).resolves.toBeNull()
  })

  it('uses initial value and custom confirmText', async () => {
    promptDialog({ title: 'T', initial: 'hello', confirmText: 'Save' })
    expect(promptState.value).toBe('hello')
    expect(promptState.confirmText).toBe('Save')
    settlePrompt(false)
  })
})

describe('settlePrompt', () => {
  it('submit with validation error stays open', async () => {
    promptDialog({ title: 'T', validate: (v) => v ? null : 'required' })
    promptState.value = ''
    settlePrompt(true)
    expect(promptState.open).toBe(true)
    expect(promptState.error).toBe('required')
  })

  it('submit with valid value closes and resolves', async () => {
    const p = promptDialog({ title: 'T', validate: () => null })
    promptState.value = 'hello'
    settlePrompt(true)
    expect(promptState.open).toBe(false)
    await expect(p).resolves.toBe('hello')
  })

  it('cancel closes and resolves null', async () => {
    const p = promptDialog({ title: 'T' })
    settlePrompt(false)
    expect(promptState.open).toBe(false)
    await expect(p).resolves.toBeNull()
  })
})
