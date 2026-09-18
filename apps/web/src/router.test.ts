import { describe, expect, it, beforeEach, vi } from 'vitest'
import { tabFromHash, setTabHash, onTabHashChange } from './router'

/** 测试用的全局替身：location/history 在 jsdom 中不可直接赋值，先 delete 再挂载。 */
type MutableGlobal = {
  location?: { hash: string }
  history?: { replaceState: (data: unknown, unused: string, url?: string | URL | null) => void }
}
const g = globalThis as unknown as MutableGlobal

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
    delete g.location
    g.location = { hash: '' }
    g.history = { replaceState: vi.fn() }
  })

  it('writes hash via history.replaceState', () => {
    setTabHash('objects')
    expect(g.history!.replaceState).toHaveBeenCalledWith(null, '', '#/objects')
  })

  it('does not call replaceState when hash already matches', () => {
    g.location!.hash = '#/objects'
    setTabHash('objects')
    expect(g.history!.replaceState).not.toHaveBeenCalled()
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
    }) as unknown as typeof window.addEventListener
    window.removeEventListener = ((type: string, fn: () => void) => {
      if (type === 'hashchange') removeListener(fn)
    }) as unknown as typeof window.removeEventListener

    const cb = vi.fn()
    const off = onTabHashChange(cb)
    expect(addListener).toHaveBeenCalled()

    off()
    expect(removeListener).toHaveBeenCalled()

    window.addEventListener = origAdd
    window.removeEventListener = origRemove
  })

  it('invokes callback when hashchange fires', () => {
    const cb = vi.fn()
    let handler: (() => void) | null = null
    const origAdd = window.addEventListener
    const origRemove = window.removeEventListener
    window.addEventListener = ((type: string, fn: () => void) => {
      if (type === 'hashchange') handler = fn
    }) as unknown as typeof window.addEventListener
    window.removeEventListener = vi.fn() as unknown as typeof window.removeEventListener

    const off = onTabHashChange(cb)
    handler!()
    expect(cb).toHaveBeenCalled()

    off()
    window.addEventListener = origAdd
    window.removeEventListener = origRemove
  })
})
