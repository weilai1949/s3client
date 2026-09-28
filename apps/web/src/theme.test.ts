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

const { readTheme, resolvedTheme, cycleTheme, systemThemeTick } = await import('./theme')

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

// applyTheme 已是模块私有（导出面死代码，见 deadcode_gate.test.ts 源码形态门禁）：
// 它的对外可见效果（data-theme + localStorage 持久化）一律经 cycleTheme 公开入口断言。
describe('applyTheme（经 cycleTheme 公开入口）', () => {
  beforeEach(() => {
    memLocal.clear()
    delete dataset['theme']
    darkMatches = true
  })

  it('应用新主题到 <html data-theme> 并持久化到 localStorage', () => {
    expect(cycleTheme()).toBe('light') // auto → light
    expect(dataset['theme']).toBe('light')
    expect(memLocal.get('s3c.theme')).toBe('light')
  })

  it('存储 light 时切换 dark：非 auto 分支也走同一套写入 + 持久化', () => {
    memLocal.set('s3c.theme', 'light')
    expect(cycleTheme()).toBe('dark')
    expect(dataset['theme']).toBe('dark')
    expect(memLocal.get('s3c.theme')).toBe('dark')
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
  it('auto 且系统偏好 light 时解析为 light（applyTheme 的 mq.matches false 侧，经 cycleTheme 走到 auto）', () => {
    darkMatches = false
    memLocal.clear()
    memLocal.set('s3c.theme', 'dark') // cycleTheme: dark → auto → 按系统偏好解析
    expect(cycleTheme()).toBe('auto')
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
      delete dataset['theme']
      // 模块级 applyTheme()（默认参数 + setItem 抛错的 catch 侧）过去会直接抛出 → 整个前端启动失败
      const mod = await import('./theme')
      expect(dataset['theme']).toBe('dark') // 初始化已按 auto+暗色系统写入，存储全坏也不崩
      expect(mod.readTheme()).toBe('auto')
      expect(mod.resolvedTheme()).toBe('dark')
      // 写入路径（cycleTheme → applyTheme → setItem 抛错 → catch 降级）同样不许抛
      expect(() => mod.cycleTheme()).not.toThrow()
      expect(dataset['theme']).toBe('light')
    } finally {
      getSpy.mockRestore()
      setSpy.mockRestore()
    }
  })
})

describe('模块初始化（applyTheme 默认参数的唯一公开观察点）', () => {
  it('启动时按 readTheme() 写入 data-theme（存储 light 而系统偏好暗色时不得误按 auto 解析）', async () => {
    memLocal.clear()
    memLocal.set('s3c.theme', 'light')
    delete dataset['theme']
    darkMatches = true // 默认参数若没读存储、误走 auto，这里会解析成 dark 而红灯
    vi.resetModules()
    await import('./theme')
    expect(dataset['theme']).toBe('light')
    expect(memLocal.get('s3c.theme')).toBe('light')
  })
})
