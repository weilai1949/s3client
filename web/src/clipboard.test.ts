import { describe, expect, it, beforeEach, vi } from 'vitest'
import { copyText } from './clipboard'

describe('copyText', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it('uses Clipboard API when available', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    Object.defineProperty(navigator, 'clipboard', {
      value: { writeText },
      configurable: true,
    })
    await copyText('hello')
    expect(writeText).toHaveBeenCalledWith('hello')
  })

  it('falls back to execCommand when Clipboard API fails', async () => {
    const writeText = vi.fn().mockRejectedValue(new Error('blocked'))
    Object.defineProperty(navigator, 'clipboard', {
      value: { writeText },
      configurable: true,
    })
    const ta = { value: 'hello', style: {}, select: vi.fn() }
    const appendChild = vi.fn()
    const removeChild = vi.fn()
    const execCommand = vi.fn().mockReturnValue(true)
    vi.stubGlobal('document', {
      createElement: vi.fn(() => ta),
      body: { appendChild, removeChild },
      execCommand,
    })
    await copyText('hello')
    expect(execCommand).toHaveBeenCalledWith('copy')
  })

  it('falls back when navigator.clipboard is absent', async () => {
    const origClipboard = (navigator as any).clipboard
    Object.defineProperty(navigator, 'clipboard', {
      value: undefined,
      configurable: true,
    })
    const execCommand = vi.fn().mockReturnValue(true)
    vi.stubGlobal('document', {
      createElement: vi.fn(() => ({ value: '', style: {}, select: vi.fn() })),
      body: { appendChild: vi.fn(), removeChild: vi.fn() },
      execCommand,
    })
    await copyText('hello')
    expect(execCommand).toHaveBeenCalledWith('copy')
    Object.defineProperty(navigator, 'clipboard', {
      value: origClipboard,
      configurable: true,
    })
  })
})
