import { describe, expect, it, beforeEach, vi } from 'vitest'
import { tabFromHash, setTabHash, onTabHashChange } from './router'

describe('tab hash router', () => {
  it('parses valid hash tabs', () => {
    expect(tabFromHash('#/objects')).toBe('objects')
    expect(tabFromHash('#objects')).toBe('objects')
    expect(tabFromHash('#/trash?x=1')).toBe('trash')
  })

  it('returns null for invalid hash', () => {
    expect(tabFromHash('')).toBe(null)
    expect(tabFromHash('#/unknown')).toBe(null)
  })
})

describe('setTabHash', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
    delete (globalThis as any).location
    ;(globalThis as any).location = { hash: '' }
    ;(globalThis as any).history = { replaceState: vi.fn() }
  })

  it('writes hash via history.replaceState', () => {
    setTabHash('objects')
    expect((globalThis as any).history.replaceState).toHaveBeenCalledWith(null, '', '#/objects')
  })

  it('does not call replaceState when hash already matches', () => {
    ;(globalThis as any).location.hash = '#/objects'
    setTabHash('objects')
    expect((globalThis as any).history.replaceState).not.toHaveBeenCalled()
  })
})

describe('onTabHashChange', () => {
  it('registers and unregisters hashchange listener', () => {
    const addListener = vi.fn()
    const removeListener = vi.fn()
    const origAdd = window.addEventListener
    const origRemove = window.removeEventListener
    window.addEventListener = ((type: string, fn: () => void) => {
      if (type === 'hashchange') addListener(fn)
    }) as any
    window.removeEventListener = ((type: string, fn: () => void) => {
      if (type === 'hashchange') removeListener(fn)
    }) as any

    const cb = vi.fn()
    const off = onTabHashChange(cb)
    expect(addListener).toHaveBeenCalled()

    off()
    expect(removeListener).toHaveBeenCalled()

    window.addEventListener = origAdd
    window.removeEventListener = origRemove
  })
})
