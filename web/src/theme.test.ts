import { describe, expect, it, beforeEach, vi } from 'vitest'

// Mock localStorage before importing theme
const memLocal = new Map<string, string>()
Object.defineProperty(globalThis, 'localStorage', {
  value: {
    getItem: (k: string) => memLocal.get(k) ?? null,
    setItem: (k: string, v: string) => memLocal.set(k, v),
    removeItem: (k: string) => memLocal.delete(k),
    clear: () => memLocal.clear(),
    key: (i: number) => [...memLocal.keys()][i] ?? null,
    get length() { return memLocal.size },
  },
  configurable: true,
  writable: true,
})

// Mock matchMedia
let darkMatches = true
const mqListeners: (() => void)[] = []
Object.defineProperty(window, 'matchMedia', {
  value: vi.fn((query: string) => ({
    get matches() { return query.includes('dark') ? darkMatches : false },
    media: query,
    onchange: null,
    addEventListener: vi.fn((_, fn: () => void) => { mqListeners.push(fn) }),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  })),
  configurable: true,
})

// Mock document.documentElement.dataset
const dataset: Record<string, string> = {}
Object.defineProperty(document.documentElement, 'dataset', {
  value: dataset,
  configurable: true,
  writable: true,
})

const { readTheme, resolvedTheme, applyTheme, cycleTheme, systemThemeTick } = await import('./theme')

describe('readTheme', () => {
  beforeEach(() => memLocal.clear())

  it('returns auto by default', () => {
    expect(readTheme()).toBe('auto')
  })

  it('returns stored light', () => {
    memLocal.set('s3c.theme', 'light')
    expect(readTheme()).toBe('light')
  })

  it('returns stored dark', () => {
    memLocal.set('s3c.theme', 'dark')
    expect(readTheme()).toBe('dark')
  })

  it('returns auto for invalid stored value', () => {
    memLocal.set('s3c.theme', 'invalid')
    expect(readTheme()).toBe('auto')
  })
})

describe('resolvedTheme', () => {
  it('returns dark when auto and prefers dark', () => {
    darkMatches = true
    memLocal.clear()
    expect(resolvedTheme()).toBe('dark')
  })

  it('returns light when auto and prefers light', () => {
    darkMatches = false
    memLocal.clear()
    expect(resolvedTheme()).toBe('light')
  })

  it('returns stored theme when not auto', () => {
    memLocal.set('s3c.theme', 'dark')
    expect(resolvedTheme()).toBe('dark')
  })
})

describe('applyTheme', () => {
  beforeEach(() => {
    memLocal.clear()
    delete dataset['theme']
    darkMatches = true
  })

  it('sets data-theme and persists to localStorage', () => {
    applyTheme('dark')
    expect(dataset['theme']).toBe('dark')
    expect(memLocal.get('s3c.theme')).toBe('dark')
  })

  it('defaults to readTheme result', () => {
    memLocal.set('s3c.theme', 'light')
    applyTheme()
    expect(dataset['theme']).toBe('light')
  })
})

describe('cycleTheme', () => {
  beforeEach(() => {
    memLocal.clear()
    delete dataset['theme']
    darkMatches = true
  })

  it('cycles auto -> light -> dark -> auto', () => {
    expect(cycleTheme()).toBe('light')
    expect(cycleTheme()).toBe('dark')
    expect(cycleTheme()).toBe('auto')
  })
})

describe('system theme listener', () => {
  it('increments tick on matchMedia change', () => {
    const initial = systemThemeTick.value
    mqListeners.forEach(fn => fn())
    expect(systemThemeTick.value).toBe(initial + 1)
  })
})
