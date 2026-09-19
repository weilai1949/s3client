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

describe('theme remaining branches', () => {
  it('applyTheme auto 且系统偏好 light 时解析为 light（mq.matches 的 false 侧）', () => {
    darkMatches = false
    memLocal.clear()
    applyTheme('auto')
    expect(dataset['theme']).toBe('light')
    expect(memLocal.get('s3c.theme')).toBe('auto')
  })

  it('系统主题变化但存储非 auto 时不重复应用（readTheme === auto 的 false 侧）', () => {
    memLocal.set('s3c.theme', 'dark')
    const before = dataset['theme']
    const tick = systemThemeTick.value
    mqListeners.forEach(fn => fn())
    expect(systemThemeTick.value).toBe(tick + 1)
    // applyTheme 未被调用：存储未被改写为 auto，data-theme 也未变化
    expect(memLocal.get('s3c.theme')).toBe('dark')
    expect(dataset['theme']).toBe(before)
  })
})

describe('存储不可用（隐私模式/配额异常）时主题模块不抛错', () => {
  it('模块初始化 + 读/写主题都在 localStorage 抛异常时降级而不是崩溃', async () => {
    const getSpy = vi.spyOn(globalThis.localStorage, 'getItem').mockImplementation(() => {
      throw new DOMException('SecurityError', 'SecurityError')
    })
    const setSpy = vi.spyOn(globalThis.localStorage, 'setItem').mockImplementation(() => {
      throw new DOMException('QuotaExceededError', 'QuotaExceededError')
    })
    try {
      vi.resetModules()
      darkMatches = true
      // 模块级 applyTheme() 过去会直接抛出 → 整个前端启动失败
      const mod = await import('./theme')
      expect(mod.readTheme()).toBe('auto')
      expect(mod.resolvedTheme()).toBe('dark')
      expect(() => mod.applyTheme('light')).not.toThrow()
      expect(dataset['theme']).toBe('light')
      expect(() => mod.cycleTheme()).not.toThrow()
    } finally {
      getSpy.mockRestore()
      setSpy.mockRestore()
    }
  })
})
