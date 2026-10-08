import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import {
  RESUME_STORAGE_KEY,
  MAX_RESUME_RECORDS,
  clearResume,
  fileFingerprint,
  loadResume,
  saveResume,
} from './multipartResume'

// happy-dom 默认不提供 localStorage；用 Map 替身补齐（同 api.transfer.test.ts）。
class MemStorage implements Storage {
  private m = new Map<string, string>()
  get length(): number { return this.m.size }
  clear() { this.m.clear() }
  key(i: number): string | null { return [...this.m.keys()][i] ?? null }
  getItem(k: string): string | null { return this.m.get(k) ?? null }
  setItem(k: string, v: string) { this.m.set(k, String(v)) }
  removeItem(k: string) { this.m.delete(k) }
}

let memLocal: MemStorage

function makeFile(name = 'big.bin', size = 100, lastModified = 1000): File {
  const f = new File(['x'], name)
  Object.defineProperty(f, 'size', { value: size })
  Object.defineProperty(f, 'lastModified', { value: lastModified })
  return f
}

const record = {
  accId: 'acc1',
  key: 'k.bin',
  bucket: 'b',
  uploadId: 'up1',
  parts: [{ partNumber: 1, etag: 'e1' }],
}

describe('multipartResume', () => {
  beforeEach(() => {
    memLocal = new MemStorage()
    Object.defineProperty(globalThis, 'localStorage', { value: memLocal, configurable: true, writable: true })
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
  })

  it('fileFingerprint 只用 name/size/lastModified，不含路径或凭证', () => {
    expect(fileFingerprint(makeFile('a.bin', 7, 42))).toBe('a.bin:7:42')
  })

  it('无记录时 loadResume 返回 undefined', () => {
    expect(loadResume(makeFile())).toBeUndefined()
  })

  it('saveResume → loadResume 往返（含 bucket 可选字段）', () => {
    const f = makeFile()
    saveResume(f, record)
    expect(loadResume(f)).toEqual(record)
    // 记录中不得出现任何凭证字段。
    const raw = localStorage.getItem(RESUME_STORAGE_KEY) ?? ''
    expect(raw).not.toContain('SecretKey')
    expect(raw).not.toContain('secretKey')
    expect(raw).not.toContain('AccessKey')
    expect(raw).not.toContain('accessKey')
  })

  it('clearResume 删除对应记录', () => {
    const f = makeFile()
    saveResume(f, record)
    clearResume(f)
    expect(loadResume(f)).toBeUndefined()
  })

  it('记录损坏（非法 JSON / null / 非对象）时降级为空表', () => {
    const f = makeFile()
    for (const bad of ['{oops', 'null', '1']) {
      localStorage.setItem(RESUME_STORAGE_KEY, bad)
      expect(loadResume(f)).toBeUndefined()
    }
  })

  it('超出 MAX_RESUME_RECORDS 时丢最早记录，保留最新', () => {
    const seeded: Record<string, typeof record> = {}
    for (let i = 0; i < MAX_RESUME_RECORDS; i++) {
      seeded[`old-${i}.bin:1:1`] = { ...record, uploadId: `up-${i}` }
    }
    localStorage.setItem(RESUME_STORAGE_KEY, JSON.stringify(seeded))
    const fresh = makeFile('fresh.bin', 2, 2)
    saveResume(fresh, { ...record, uploadId: 'up-fresh' })
    const all = JSON.parse(localStorage.getItem(RESUME_STORAGE_KEY) ?? '{}') as Record<string, unknown>
    expect(Object.keys(all)).toHaveLength(MAX_RESUME_RECORDS)
    expect(all['old-0.bin:1:1']).toBeUndefined()
    expect(loadResume(fresh)?.uploadId).toBe('up-fresh')
  })

  it('localStorage 不可用（未定义）时全部降级为 no-op', () => {
    vi.stubGlobal('localStorage', undefined)
    const f = makeFile()
    expect(loadResume(f)).toBeUndefined()
    expect(() => saveResume(f, record)).not.toThrow()
    expect(() => clearResume(f)).not.toThrow()
  })

  it('访问 localStorage 抛 SecurityError 时全部降级为 no-op', () => {
    const original = Object.getOwnPropertyDescriptor(globalThis, 'localStorage')
    Object.defineProperty(globalThis, 'localStorage', {
      configurable: true,
      get() {
        throw new Error('denied')
      },
    })
    try {
      const f = makeFile()
      expect(loadResume(f)).toBeUndefined()
      expect(() => saveResume(f, record)).not.toThrow()
      expect(() => clearResume(f)).not.toThrow()
    } finally {
      if (original) Object.defineProperty(globalThis, 'localStorage', original)
      else delete (globalThis as { localStorage?: unknown }).localStorage
    }
  })

  it('setItem 抛配额错误时静默放弃（不影响上线路径）', () => {
    const f = makeFile()
    vi.spyOn(memLocal, 'setItem').mockImplementation(() => {
      throw new Error('quota exceeded')
    })
    expect(() => saveResume(f, record)).not.toThrow()
    expect(() => clearResume(f)).not.toThrow()
  })
})
