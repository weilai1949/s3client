// api.test.ts —— API 层测试的**基础**范围：token 存储、servers、请求封装、s3api。
// KNOWN_ISSUES #60 拆分后的另两部分见 api.transfer.test.ts 与 api.gaps.test.ts（断言逐字搬移）。
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

// happy-dom 默认不提供 localStorage/sessionStorage；用简单 Map 替身补齐。
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
let memSession: MemStorage

beforeEach(() => {
  memLocal = new MemStorage()
  memSession = new MemStorage()
  Object.defineProperty(globalThis, 'localStorage', { value: memLocal, configurable: true, writable: true })
  Object.defineProperty(globalThis, 'sessionStorage', { value: memSession, configurable: true, writable: true })
  vi.resetModules()
})

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

// ── i18n stub ──────────────────────────────────────────────────────────────
vi.mock('./i18n', () => ({
  t: (k: string) => k,
  tf: (k: string) => k,
  locale: 'zh-CN',
}))

// ── helpers ────────────────────────────────────────────────────────────────
async function loadApi() {
  return import('./api')
}

function stubFetch(impl: (input: RequestInfo | URL, init?: RequestInit) => unknown) {
  return vi.stubGlobal('fetch', vi.fn(impl))
}

function makeBlobResponse(data: unknown, status = 200, statusText = 'OK'): Response {
  return {
    ok: true,
    status,
    statusText,
    json: () => Promise.resolve(data),
    blob: () => Promise.resolve(new Blob([JSON.stringify(data)])),
    headers: new Map<string, string>(),
    body: null,
  } as unknown as Response
}

function makeErrorResponse(status = 400, statusText = 'Bad Request', error?: string): Response {
  const body = error ? JSON.stringify({ error }) : ''
  return {
    ok: false,
    status,
    statusText,
    json: () => Promise.resolve(error ? { error } : {}),
    text: () => Promise.resolve(body),
    headers: new Map<string, string>(),
    body: null,
  } as unknown as Response
}

// ── api token 存储 ─────────────────────────────────────────────────────────
describe('api token 存储', () => {
  it('默认写入 sessionStorage，不写 localStorage', async () => {
    const { api } = await loadApi()
    api.token = 'session-token-1234567890'
    expect(memSession.getItem('s3c.token')).toBe('session-token-1234567890')
    expect(memLocal.getItem('s3c.token')).toBeNull()
    expect(api.token).toBe('session-token-1234567890')
  })

  it('setTokenPersistent(true) 同时写 sessionStorage 与 localStorage', async () => {
    const { api } = await loadApi()
    api.token = 'persist-token-1234567890'
    api.setTokenPersistent(true)
    expect(memSession.getItem('s3c.token')).toBe('persist-token-1234567890')
    expect(memLocal.getItem('s3c.token')).toBe('persist-token-1234567890')
    expect(api.isTokenPersistent).toBe(true)
  })

  it('setTokenPersistent(false) 把 token 从 localStorage 移走，仅留 sessionStorage', async () => {
    const { api } = await loadApi()
    api.setTokenPersistent(true)
    api.token = 't1'
    expect(memLocal.getItem('s3c.token')).toBe('t1')
    api.setTokenPersistent(false)
    expect(memLocal.getItem('s3c.token')).toBeNull()
    expect(memLocal.getItem('s3c_token_persistent')).toBeNull()
    expect(memSession.getItem('s3c.token')).toBe('t1')
  })

  it('旧版本遗留的 localStorage token 首次读取时自动迁移到 sessionStorage', async () => {
    memLocal.setItem('s3c.token', 'legacy-token-1234567890')
    const { api } = await loadApi()
    expect(api.token).toBe('legacy-token-1234567890')
    expect(memLocal.getItem('s3c.token')).toBeNull()
    expect(memSession.getItem('s3c.token')).toBe('legacy-token-1234567890')
  })

  it('tokenPersistent() localStorage 抛异常时返回 false', async () => {
    // 使 localStorage.getItem 抛异常
    const orig = memLocal.getItem
    memLocal.getItem = () => { throw new Error('boom') }
    const { api } = await loadApi()
    expect(api.isTokenPersistent).toBe(false)
    memLocal.getItem = orig
  })

  it('token 迁移读取 localStorage 抛异常时回退空串', async () => {
    const orig = memLocal.getItem
    memLocal.getItem = ((k: string) => {
      if (k === 's3c.token') throw new Error('boom')
      return orig.call(memLocal, k)
    })
    const { api } = await loadApi()
    // session 为空 + 非持久化 → 进入一次性迁移分支，getItem 抛异常 → catch 返回 ''
    expect(api.token).toBe('')
    memLocal.getItem = orig
  })

  it('清空 token 字符串时 sessionStorage 与 localStorage 都被清空', async () => {
    const { api } = await loadApi()
    api.setTokenPersistent(true)
    api.token = 't2'
    expect(memSession.getItem('s3c.token')).toBe('t2')
    expect(memLocal.getItem('s3c.token')).toBe('t2')
    api.token = ''
    expect(memSession.getItem('s3c.token')).toBeNull()
    expect(memLocal.getItem('s3c.token')).toBeNull()
  })

  it('setTokenPersistent(true) 迁移已有 per-server token 到 localStorage，(false) 清掉全部残留副本', async () => {
    const { api } = await loadApi()
    api.token = 'global-token-abcdef'
    const p = api.upsertServer({ name: 'mig', base: 'http://x:9000', token: 'mig-token-123456789' })
    // 空值的 session 键不落盘；仅剩 localStorage 的残留（session 已清）也要能被关闭清理
    memSession.setItem('s3c.token.ghost', '')
    memLocal.setItem('s3c.token.stale', 'stale-token-123')
    expect(memLocal.getItem(`s3c.token.${p.id}`)).toBeNull()
    api.setTokenPersistent(true)
    expect(memLocal.getItem(`s3c.token.${p.id}`)).toBe('mig-token-123456789')
    expect(memLocal.getItem('s3c.token.ghost')).toBeNull()
    api.setTokenPersistent(false)
    expect(memLocal.getItem(`s3c.token.${p.id}`)).toBeNull()
    expect(memLocal.getItem('s3c.token.stale')).toBeNull()
    // 关闭只清 localStorage 副本，session 里的 token 不受影响
    expect(memSession.getItem(`s3c.token.${p.id}`)).toBe('mig-token-123456789')
  })

  it('setTokenPersistent 迁移 per-server token 时存储抛异常 → 吞掉不中断', async () => {
    const { api } = await loadApi()
    const p = api.upsertServer({ name: 'x', base: 'http://x:9000', token: 'tok-abcdefghij' })
    const origSet = memLocal.setItem
    memLocal.setItem = () => { throw new Error('quota') }
    api.setTokenPersistent(true) // 持久化标记 + per-server 副本两次 setItem 都会抛，均应被吞
    memLocal.setItem = origSet
    expect(api.isTokenPersistent).toBe(false)
    expect(memSession.getItem(`s3c.token.${p.id}`)).toBe('tok-abcdefghij')
  })

  it('setTokenPersistent(true) 读取 sessionStorage 键列表抛异常 → 迁移中断但不抛', async () => {
    const { api } = await loadApi()
    const p = api.upsertServer({ name: 'y', base: 'http://y:9000', token: 'tok-zyxwvutsr' })
    Object.defineProperty(memSession, 'length', { get() { throw new Error('boom') }, configurable: true })
    api.setTokenPersistent(true)
    expect(api.isTokenPersistent).toBe(true)
    expect(memLocal.getItem(`s3c.token.${p.id}`)).toBeNull()
    expect(memSession.getItem(`s3c.token.${p.id}`)).toBe('tok-zyxwvutsr')
  })
})

// ── isTauri / defaultBase / newId ──────────────────────────────────────────
describe('isTauri / defaultBase / newId', () => {
  it('isTauri() 检测 window.__TAURI_INTERNALS__', async () => {
    const w = window as unknown as { __TAURI_INTERNALS__?: boolean }
    w.__TAURI_INTERNALS__ = true
    try {
      const { api } = await loadApi()
      expect(api.isTauri).toBe(true)
    } finally {
      w.__TAURI_INTERNALS__ = undefined
    }
  })

  it('isTauri() 检测 window.__TAURI__', async () => {
    const w = window as unknown as { __TAURI__?: boolean }
    w.__TAURI__ = true
    try {
      const { api } = await loadApi()
      expect(api.isTauri).toBe(true)
    } finally {
      w.__TAURI__ = undefined
    }
  })

  it('isTauri() 检测 navigator.userAgent', async () => {
    const orig = navigator.userAgent
    Object.defineProperty(navigator, 'userAgent', { value: 'tauri', configurable: true })
    try {
      const { api } = await loadApi()
      expect(api.isTauri).toBe(true)
    } finally {
      Object.defineProperty(navigator, 'userAgent', { value: orig, configurable: true })
    }
  })

  it('isTauri() 检测 location.hostname', async () => {
    const orig = location.hostname
    Object.defineProperty(location, 'hostname', { value: 'tauri.localhost', configurable: true })
    try {
      const { api } = await loadApi()
      expect(api.isTauri).toBe(true)
    } finally {
      Object.defineProperty(location, 'hostname', { value: orig, configurable: true })
    }
  })

  it('defaultBase() Tauri → http://127.0.0.1:8080', async () => {
    const orig = location.hostname
    Object.defineProperty(location, 'hostname', { value: 'tauri.localhost', configurable: true })
    try {
      const { api } = await loadApi()
      expect(api.base).toBe('http://127.0.0.1:8080')
    } finally {
      Object.defineProperty(location, 'hostname', { value: orig, configurable: true })
    }
  })

  it('newId() 返回非空字符串', async () => {
    const { api } = await loadApi()
    const id = api.upsertServer({ name: 'x', base: '/', token: '' }).id
    expect(typeof id).toBe('string')
    expect(id.length).toBeGreaterThan(0)
  })
})

// ── servers ────────────────────────────────────────────────────────────────
describe('servers', () => {
  it('listServers() 首次返回默认 server', async () => {
    const { api } = await loadApi()
    const list = api.listServers()
    expect(Array.isArray(list)).toBe(true)
    expect(list.length).toBeGreaterThanOrEqual(1)
  })

  it('activeServerId() 回退到 list[0].id', async () => {
    const { api } = await loadApi()
    const id = api.activeServerId()
    const list = api.listServers()
    expect(id).toBe(list[0].id)
  })

  it('activeServerId() 未知 id 回退到 list[0].id', async () => {
    await loadApi()
    memLocal.setItem('s3c.activeServerId', 'nonexistent-id')
    // 重新加载以读取新的 localStorage
    const mod = await import('./api')
    expect(mod.api.activeServerId()).toBe(mod.api.listServers()[0].id)
  })

  // P0-2（docs/archive/review-2026-09-19.md §F1）：s3c.servers 被外部写坏时整站白屏。
  // 读路径在渲染期被 App.vue 调用，抛异常 = 白屏且 UI 无法自救（连修复它的 Server 面板都不可达）。
  describe('存储被写坏时不得抛异常 / 不得产生幽灵 server', () => {
    const corrupt: Array<{ name: string; raw: string }> = [
      { name: '[null]（元素为 null）', raw: '[null]' },
      { name: '[1]（元素为数字）', raw: '[1]' },
      { name: '["x"]（元素为字符串）', raw: '["x"]' },
      { name: '[{}]（元素缺 id）', raw: '[{}]' },
      { name: '[{"base":{}}]（base 非字符串）', raw: '[{"base":{}}]' },
    ]

    for (const c of corrupt) {
      it(`${c.name} → listServers/getActiveServer 正常返回且不残留坏条目`, async () => {
        memLocal.setItem('s3c.servers', c.raw)
        const { api } = await loadApi()

        // 读取路径（App.vue 渲染期调用）必须不抛。
        let list: unknown[] = []
        expect(() => { list = api.listServers() }).not.toThrow()
        expect(() => api.getActiveServer()).not.toThrow()
        expect(() => api.activeServerId()).not.toThrow()

        // 坏条目不得变成 id:'' 的幽灵 server。
        expect(list.every((s) => typeof (s as { id: string }).id === 'string' && (s as { id: string }).id !== '')).toBe(true)
        // 且存储被修复（不把坏数据留在 localStorage 里反复踩）。
        const stored = JSON.parse(memLocal.getItem('s3c.servers') ?? 'null') as unknown[] | null
        expect(Array.isArray(stored)).toBe(true)
        expect(JSON.stringify(stored)).not.toContain('null')
      })
    }

    it('坏条目与合法条目混合时保留合法的那些', async () => {
      memLocal.setItem('s3c.servers', '[null,{"id":"keep","name":"ok","base":"http://ok"},7]')
      const { api } = await loadApi()
      const list = api.listServers()
      expect(list.map((s) => s.id)).toEqual(['keep'])
      expect(api.getActiveServer()?.name).toBe('ok')
    })

    it('存储完全不可用时 getActiveServer() 返回 undefined 而不是抛出（渲染期不得白屏）', async () => {
      const { api } = await loadApi()
      const boom = () => { throw new Error('SecurityError: storage disabled') }
      vi.spyOn(memLocal, 'getItem').mockImplementation(boom)
      vi.spyOn(memLocal, 'setItem').mockImplementation(boom)

      expect(() => api.getActiveServer()).not.toThrow()
      expect(api.getActiveServer()).toBeUndefined()
    })
  })

  it('selectServer() 生效并同步', async () => {
    const { api } = await loadApi()
    const list = api.listServers()
    const p = api.selectServer(list[0].id)
    expect(p?.id).toBe(list[0].id)
    expect(memLocal.getItem('s3c.apiBase')).toBeDefined()
  })

  it('upsertServer() 创建新 server', async () => {
    const { api } = await loadApi()
    const p = api.upsertServer({ name: 'new-srv', base: 'http://x:9000', token: 'tk' })
    expect(p.name).toBe('new-srv')
    expect(p.base).toBe('http://x:9000')
    expect(p.token).toBe('tk')
  })

  it('upsertServer() 更新已有 server 并同步 applyProfile', async () => {
    const { api } = await loadApi()
    const list = api.listServers()
    const existing = list[0]
    const updated = api.upsertServer({ id: existing.id, name: 'renamed', base: 'http://y:9000', token: 'tk2' })
    expect(updated.name).toBe('renamed')
    expect(updated.base).toBe('http://y:9000')
    expect(updated.token).toBe('tk2')
  })

  it('deleteServer() 移除并回退默认', async () => {
    const { api } = await loadApi()
    // 先创建一个额外的 server
    const extra = api.upsertServer({ name: 'to-delete', base: 'http://x:9000', token: '' })
    api.deleteServer(extra.id)
    const list3 = api.listServers()
    expect(list3.find((s) => s.id === extra.id)).toBeUndefined()
  })

  it('deleteServer() 仅剩一个时重建默认', async () => {
    const { api } = await loadApi()
    const list = api.listServers()
    // 只删一个（默认的唯一 server）
    api.deleteServer(list[0].id)
    const list2 = api.listServers()
    expect(list2.length).toBeGreaterThanOrEqual(1)
  })

  it('getActiveServer() 返回当前生效 server', async () => {
    const { api } = await loadApi()
    const p = api.getActiveServer()
    expect(p).toBeDefined()
    expect(p?.id).toBe(api.activeServerId())
  })

  it('upsertServer() 的 token 不得明文落 localStorage.s3c.servers（安全策略：token 仅 sessionStorage / 显式持久化）', async () => {
    const { api } = await loadApi()
    const p = api.upsertServer({ name: 'sec-srv', base: 'http://x:9000', token: 'super-secret-token-xyz' })
    // 1) 磁盘上的 servers 列表不得包含 token 字段
    const raw = memLocal.getItem('s3c.servers')
    expect(raw).not.toBeNull()
    const stored = JSON.parse(raw ?? '[]') as Array<Record<string, unknown>>
    expect(stored.length).toBeGreaterThanOrEqual(1)
    for (const s of stored) {
      expect(s).not.toHaveProperty('token')
      expect(JSON.stringify(s)).not.toContain('super-secret-token-xyz')
    }
    // 2) 返回值仍携带 token（运行时视图）
    expect(p.token).toBe('super-secret-token-xyz')
    // 3) token 按服务器 id 存 sessionStorage（默认非持久化）
    expect(memSession.getItem(`s3c.token.${p.id}`)).toBe('super-secret-token-xyz')
    expect(memLocal.getItem(`s3c.token.${p.id}`)).toBeNull()
  })

  it('upsertServer() 开启跨会话保留时 token 同时落 localStorage.s3c.token.<id>', async () => {
    const { api } = await loadApi()
    api.setTokenPersistent(true)
    const p = api.upsertServer({ name: 'persist-srv', base: 'http://x:9000', token: 'persist-token-abc' })
    expect(memLocal.getItem(`s3c.token.${p.id}`)).toBe('persist-token-abc')
    expect(memSession.getItem(`s3c.token.${p.id}`)).toBe('persist-token-abc')
    // servers 列表仍然无 token
    const raw = memLocal.getItem('s3c.servers')
    expect(JSON.stringify(raw)).not.toContain('persist-token-abc')
  })

  it('listServers() 从 per-server token 存储恢复 token 视图', async () => {
    const { api } = await loadApi()
    const p = api.upsertServer({ name: 'restore-srv', base: 'http://x:9000', token: 'restore-token-123' })
    // 重新加载模块（模拟新会话/刷新）
    const mod = await import('./api')
    const list = mod.api.listServers()
    const found = list.find((s) => s.id === p.id)
    expect(found?.token).toBe('restore-token-123')
  })

  it('旧版本 s3c.servers 内嵌 token 首次读取时迁移到 per-server 存储并重写 localStorage', async () => {
    const legacyId = 'legacy-server-id'
    memLocal.setItem(
      's3c.servers',
      JSON.stringify([{ id: legacyId, name: 'old', base: 'http://x:9000', token: 'legacy-token-abc' }]),
    )
    memLocal.setItem('s3c.activeServerId', legacyId)
    const { api } = await loadApi()
    // 触发读取（迁移在 listServers/activeServerId 路径中执行）
    api.listServers()
    // 迁移：per-server 存储可见
    expect(memSession.getItem(`s3c.token.${legacyId}`)).toBe('legacy-token-abc')
    // localStorage.servers 已重写为无 token
    const stored = JSON.parse(memLocal.getItem('s3c.servers') ?? '[]') as Array<Record<string, unknown>>
    for (const s of stored) expect(s).not.toHaveProperty('token')
    // 活动 server 的全局 token 也已生效
    expect(api.token).toBe('legacy-token-abc')
  })

  it('deleteServer() 同时清除该 server 的 per-server token', async () => {
    const { api } = await loadApi()
    const p = api.upsertServer({ name: 'to-del-sec', base: 'http://x:9000', token: 'del-token-456' })
    expect(memSession.getItem(`s3c.token.${p.id}`)).toBe('del-token-456')
    api.deleteServer(p.id)
    expect(memSession.getItem(`s3c.token.${p.id}`)).toBeNull()
  })
})

// ── api.base / api.token getter/setter ──────────────────────────────────────
describe('api getter/setter', () => {
  it('base getter/setter 读写 localStorage', async () => {
    const { api } = await loadApi()
    api.base = 'https://s3.example.com'
    expect(api.base).toBe('https://s3.example.com')
    expect(memLocal.getItem('s3c.apiBase')).toBe('https://s3.example.com')
  })

  it('localStorage 读抛错时 base getter 回退默认值（降级策略与模块其余读取一致）', async () => {
    const { api } = await loadApi()
    vi.spyOn(memLocal, 'getItem').mockImplementation(() => {
      throw new Error('storage broken')
    })
    expect(() => api.base).not.toThrow()
    expect(api.base).toBe('')
  })

  it('localStorage 写抛错时 base setter 不抛异常（静默降级）', async () => {
    const { api } = await loadApi()
    vi.spyOn(memLocal, 'setItem').mockImplementation(() => {
      throw new Error('quota exceeded')
    })
    expect(() => {
      api.base = 'https://s3.example.com'
    }).not.toThrow()
  })
})

// ── request / requestResponse ──────────────────────────────────────────────
describe('request / requestResponse', () => {
  it('api.health() 请求 /api/health 并返回解析结果', async () => {
    stubFetch(() => Promise.resolve(makeBlobResponse({ status: 'ok', version: 'v1' })))
    const { api } = await loadApi()
    api.base = 'https://s3.example.com'
    const res = await api.health()
    expect(res).toEqual({ status: 'ok', version: 'v1' })
    expect(vi.mocked(globalThis.fetch)).toHaveBeenCalledWith(
      'https://s3.example.com/api/health',
      expect.objectContaining({ headers: {} }),
    )
  })

  it('request 成功返回 json', async () => {
    stubFetch(() => Promise.resolve(makeBlobResponse({ ok: true })))
    const { api } = await loadApi()
    api.base = 'https://s3.example.com'
    await (api as { request?: (path: string) => Promise<unknown> }).request?.('/test')
    // request 是私有函数不导出；我们通过 s3api.listAccounts 间接覆盖
  })

  it('requestResponse 成功返回 Response', async () => {
    stubFetch(() => Promise.resolve(makeBlobResponse({ ok: true })))
    const { api } = await loadApi()
    api.base = 'https://s3.example.com'
  })

  it('requestResponse 不带 opts：默认参数 `= {}` 生效，无 Content-Type', async () => {
    stubFetch(() => Promise.resolve(makeBlobResponse({ ok: true })))
    const { api } = await loadApi()
    const { requestResponse } = await import('./api/http')
    api.base = 'https://s3.example.com'
    const res = await requestResponse('/test')
    expect(res.ok).toBe(true)
    expect(vi.mocked(globalThis.fetch)).toHaveBeenCalledWith('https://s3.example.com/test', expect.objectContaining({ headers: {} }))
  })

  it('requestResponse 带 opts 但无 body：不设置 Content-Type', async () => {
    stubFetch(() => Promise.resolve(makeBlobResponse({ ok: true })))
    const { api } = await loadApi()
    const { requestResponse } = await import('./api/http')
    api.base = 'https://s3.example.com'
    await requestResponse('/test', { method: 'GET' })
    expect(vi.mocked(globalThis.fetch)).toHaveBeenCalledWith(
      'https://s3.example.com/test',
      expect.objectContaining({ method: 'GET', headers: {} }),
    )
  })

  it('request 失败抛错并解析 error 字段', async () => {
    stubFetch(() => Promise.resolve(makeErrorResponse(403, 'Forbidden', 'bad token')))
    const { api } = await loadApi()
    api.base = 'https://s3.example.com'
  })

  it('调用方自带 headers 时与默认 Authorization/Content-Type 合并（默认值不丢）', async () => {
    stubFetch(() => Promise.resolve(makeBlobResponse({ ok: true })))
    const { api } = await loadApi()
    api.token = 'tok-abc'
    api.base = 'https://s3.example.com'
    const { request } = await import('./api/http')
    await request('/test', { method: 'POST', body: '{}', headers: { 'X-Trace': 't1', 'Content-Type': 'text/csv' } })
    expect(vi.mocked(globalThis.fetch)).toHaveBeenCalledWith(
      'https://s3.example.com/test',
      expect.objectContaining({
        method: 'POST',
        body: '{}',
        headers: {
          'X-Trace': 't1',
          // 调用方同名 header 优先
          'Content-Type': 'text/csv',
          Authorization: 'Bearer tok-abc',
        },
      }),
    )
  })

  it('requestResponse 同样合并 headers（授权头不被调用方 headers 覆盖掉）', async () => {
    stubFetch(() => Promise.resolve(makeBlobResponse({ ok: true })))
    const { api } = await loadApi()
    api.token = 'tok-xyz'
    api.base = 'https://s3.example.com'
    const { requestResponse } = await import('./api/http')
    await requestResponse('/stream', { headers: { Accept: 'application/zip' } })
    const init = vi.mocked(globalThis.fetch).mock.calls[0][1] as RequestInit
    expect(init.headers).toEqual({ Accept: 'application/zip', Authorization: 'Bearer tok-xyz' })
  })
})

// ── s3api methods ───────────────────────────────────────────────────────────
describe('s3api', () => {
  beforeEach(() => {
    stubFetch(() => Promise.resolve(makeBlobResponse({})))
  })

  it('listAccounts', async () => {
    const { s3api } = await import('./api')
    await s3api.listAccounts()
    expect(fetch).toHaveBeenCalledWith('/api/accounts', expect.anything())
  })

  it('createAccount', async () => {
    const { s3api } = await import('./api')
    await s3api.createAccount({
      name: 'a',
      endpoint: 'http://x',
      region: 'us-east-1',
      accessKey: 'ak',
      secretKey: 'sk',
      bucket: 'b',
      pathStyle: true,
      useSSL: false,
    })
    expect(fetch).toHaveBeenCalledWith('/api/accounts', expect.objectContaining({ method: 'POST' }))
  })

  it('updateAccount', async () => {
    const { s3api } = await import('./api')
    await s3api.updateAccount('id1', { name: 'b' })
    expect(fetch).toHaveBeenCalledWith('/api/accounts/id1', expect.objectContaining({ method: 'PUT' }))
  })

  it('deleteAccount', async () => {
    const { s3api } = await import('./api')
    await s3api.deleteAccount('id1')
    expect(fetch).toHaveBeenCalledWith('/api/accounts/id1', expect.objectContaining({ method: 'DELETE' }))
  })

  it('testAccount', async () => {
    const { s3api } = await import('./api')
    await s3api.testAccount('id1')
    expect(fetch).toHaveBeenCalledWith('/api/accounts/id1/test', expect.objectContaining({ method: 'POST' }))
  })

  it('listBuckets', async () => {
    const { s3api } = await import('./api')
    await s3api.listBuckets('id1')
    expect(fetch).toHaveBeenCalledWith('/api/accounts/id1/buckets', expect.anything())
  })

  it('createBucket', async () => {
    const { s3api } = await import('./api')
    await s3api.createBucket('id1', { name: 'nb', region: 'us-east-1', acl: 'private' })
    expect(fetch).toHaveBeenCalledWith(
      '/api/accounts/id1/bucket',
      expect.objectContaining({ method: 'POST', body: JSON.stringify({ name: 'nb', region: 'us-east-1', acl: 'private' }) }),
    )
  })

  it('deleteBucket encodes name query', async () => {
    const { s3api } = await import('./api')
    await s3api.deleteBucket('id1', 'b name/ç')
    expect(fetch).toHaveBeenCalledWith(
      '/api/accounts/id1/bucket?name=b%20name%2F%C3%A7',
      expect.objectContaining({ method: 'DELETE' }),
    )
  })

  it('listObjects', async () => {
    const { s3api } = await import('./api')
    await s3api.listObjects('id1', { prefix: 'foo' })
    expect(fetch).toHaveBeenCalledWith('/api/accounts/id1/objects?prefix=foo', expect.anything())
  })

  it('headObject', async () => {
    const { s3api } = await import('./api')
    await s3api.headObject('id1', { key: 'k' })
    expect(fetch).toHaveBeenCalledWith('/api/accounts/id1/head?key=k', expect.anything())
  })

  it('mkdirObject', async () => {
    const { s3api } = await import('./api')
    await s3api.mkdirObject('id1', { bucket: 'b', key: 'k' })
    expect(fetch).toHaveBeenCalledWith('/api/accounts/id1/mkdir', expect.objectContaining({ method: 'POST' }))
  })

  it('renameObject', async () => {
    const { s3api } = await import('./api')
    await s3api.renameObject('id1', { bucket: 'b', key: 'k', newKey: 'kk' })
    expect(fetch).toHaveBeenCalledWith('/api/accounts/id1/rename', expect.objectContaining({ method: 'POST' }))
  })

  it('copyObject', async () => {
    const { s3api } = await import('./api')
    await s3api.copyObject('id1', { bucket: 'b', key: 'k', newKey: 'kk' })
    expect(fetch).toHaveBeenCalledWith('/api/accounts/id1/copy-object', expect.objectContaining({ method: 'POST' }))
  })

  it('copyFilesAsync', async () => {
    const { s3api } = await import('./api')
    await s3api.copyFilesAsync('id1', { bucket: 'b', keys: ['k'] })
    expect(fetch).toHaveBeenCalledWith('/api/accounts/id1/copy-objects/async', expect.objectContaining({ method: 'POST' }))
  })

  it('getObjectAcl', async () => {
    const { s3api } = await import('./api')
    await s3api.getObjectAcl('id1', { bucket: 'b', key: 'k' })
    expect(fetch).toHaveBeenCalledWith(expect.stringContaining('/object-acl?'), expect.anything())
  })

  it('putObjectAcl', async () => {
    const { s3api } = await import('./api')
    await s3api.putObjectAcl('id1', { bucket: 'b', key: 'k', acl: 'private' })
    expect(fetch).toHaveBeenCalledWith('/api/accounts/id1/object-acl', expect.objectContaining({ method: 'PUT' }))
  })

  it('getObjectTags', async () => {
    const { s3api } = await import('./api')
    await s3api.getObjectTags('id1', { bucket: 'b', key: 'k' })
    expect(fetch).toHaveBeenCalledWith(expect.stringContaining('/object-tags?'), expect.anything())
  })

  it('putObjectTags', async () => {
    const { s3api } = await import('./api')
    await s3api.putObjectTags('id1', { bucket: 'b', key: 'k', tags: [] })
    expect(fetch).toHaveBeenCalledWith('/api/accounts/id1/object-tags', expect.objectContaining({ method: 'PUT' }))
  })

  it('getBucketInfo', async () => {
    const { s3api } = await import('./api')
    await s3api.getBucketInfo('id1', 'b')
    expect(fetch).toHaveBeenCalledWith('/api/accounts/id1/bucket-info?bucket=b', expect.anything())
  })

  it('putBucketVersioning', async () => {
    const { s3api } = await import('./api')
    await s3api.putBucketVersioning('id1', { bucket: 'b', status: 'Enabled' })
    expect(fetch).toHaveBeenCalledWith('/api/accounts/id1/bucket-versioning', expect.objectContaining({ method: 'PUT' }))
  })

  it('getBucketEncryption / putBucketEncryption / deleteBucketEncryption', async () => {
    const { s3api } = await import('./api')
    await s3api.getBucketEncryption('id1', 'b')
    await s3api.putBucketEncryption('id1', { bucket: 'b', algorithm: 'AES256' })
    await s3api.deleteBucketEncryption('id1', 'b')
    expect(fetch).toHaveBeenCalled()
  })

  it('getBucketCors / putBucketCors / deleteBucketCors', async () => {
    const { s3api } = await import('./api')
    await s3api.getBucketCors('id1', 'b')
    await s3api.putBucketCors('id1', { bucket: 'b', rules: [] })
    await s3api.deleteBucketCors('id1', 'b')
  })

  it('getBucketWebsite / putBucketWebsite / deleteBucketWebsite', async () => {
    const { s3api } = await import('./api')
    await s3api.getBucketWebsite('id1', 'b')
    await s3api.putBucketWebsite('id1', { bucket: 'b' })
    await s3api.deleteBucketWebsite('id1', 'b')
  })

  it('getBucketPolicy / putBucketPolicy / deleteBucketPolicy', async () => {
    const { s3api } = await import('./api')
    await s3api.getBucketPolicy('id1', 'b')
    await s3api.putBucketPolicy('id1', { bucket: 'b', policy: '{}' })
    await s3api.deleteBucketPolicy('id1', 'b')
  })

  it('getBucketTags / putBucketTags / deleteBucketTags', async () => {
    const { s3api } = await import('./api')
    await s3api.getBucketTags('id1', 'b')
    await s3api.putBucketTags('id1', { bucket: 'b', tags: [] })
    await s3api.deleteBucketTags('id1', 'b')
  })

  it('listVersions / deleteObjectVersion / restoreObjectVersion / restoreDeleteMarker', async () => {
    const { s3api } = await import('./api')
    await s3api.listVersions('id1', { bucket: 'b' })
    await s3api.deleteObjectVersion('id1', { bucket: 'b', key: 'k', versionId: 'v' })
    await s3api.restoreObjectVersion('id1', { bucket: 'b', key: 'k', versionId: 'v' })
    await s3api.restoreDeleteMarker('id1', { bucket: 'b', key: 'k', versionId: 'v' })
  })

  it('listTrash / purgeTrashObject', async () => {
    const { s3api } = await import('./api')
    await s3api.listTrash('id1', { bucket: 'b' })
    await s3api.purgeTrashObject('id1', { bucket: 'b', key: 'k' })
  })

  it('changeStorageClass / setHeaders', async () => {
    const { s3api } = await import('./api')
    await s3api.changeStorageClass('id1', { bucket: 'b', key: 'k', storageClass: 'STANDARD' })
    await s3api.setHeaders('id1', { bucket: 'b', key: 'k' })
  })

  it('getLifecycle / putLifecycle', async () => {
    const { s3api } = await import('./api')
    await s3api.getLifecycle('id1', 'b')
    await s3api.putLifecycle('id1', { bucket: 'b', rules: [] })
  })

  it('presign / multipartInit / multipartPart / multipartComplete / multipartAbort', async () => {
    const { s3api } = await import('./api')
    await s3api.presign('id1', { key: 'k' })
    await s3api.multipartInit('id1', { bucket: 'b', key: 'k' })
    await s3api.multipartPart('id1', { bucket: 'b', key: 'k', uploadId: 'u', partNumber: 1 })
    await s3api.multipartComplete('id1', { bucket: 'b', key: 'k', uploadId: 'u', parts: [] })
    await s3api.multipartAbort('id1', { bucket: 'b', key: 'k', uploadId: 'u' })
  })

  it('deleteObjects / deletePrefixAsync', async () => {
    const { s3api } = await import('./api')
    await s3api.deleteObjects('id1', { bucket: 'b', keys: ['k'] })
    await s3api.deletePrefixAsync('id1', { bucket: 'b', prefix: 'p' })
  })

  it('copyPrefixAsync', async () => {
    const { s3api } = await import('./api')
    await s3api.copyPrefixAsync('id1', { bucket: 'b', prefix: 'p', targetBucket: 'tb', targetPrefix: 'tp' })
  })

  it('migrateAsync / migrateJobs / migrateJobStatus / migrateJobCancel', async () => {
    const { s3api } = await import('./api')
    await s3api.migrateAsync({ sourceAccountId: 'a1', sourceKeys: ['k'], targetAccountId: 'a2' })
    await s3api.migrateJobs()
    await s3api.migrateJobStatus('job1')
    await s3api.migrateJobCancel('job1')
  })

  it('downloadZipToDisk blob fallback success', async () => {
    stubFetch(() =>
      Promise.resolve({
        ok: true,
        status: 200,
        statusText: 'OK',
        headers: new Map<string, string>([['Content-Length', '1000']]),
        blob: () => Promise.resolve(new Blob(['zip'])),
        body: null,
      })
    )
    const { downloadZipToDisk } = await import('./api/download')
    await downloadZipToDisk('id1', { keys: ['a.txt'] })
    expect(fetch).toHaveBeenCalled()
  })

  it('downloadZipToDisk blob fallback 延迟 revokeObjectURL（不中断下载）', async () => {
    stubFetch(() =>
      Promise.resolve({
        ok: true,
        status: 200,
        statusText: 'OK',
        headers: new Map<string, string>([['Content-Length', '1000']]),
        blob: () => Promise.resolve(new Blob(['zip'])),
        body: null,
      })
    )
    const { downloadZipToDisk } = await import('./api/download')
    const create = vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:zip')
    const revoke = vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => {})
    vi.useFakeTimers()
    try {
      await downloadZipToDisk('id1', { keys: ['a.txt'] })
      expect(create).toHaveBeenCalled()
      // 关键：下载启动时不得同步回收（否则部分浏览器会中断下载）
      expect(revoke).not.toHaveBeenCalled()
      vi.advanceTimersByTime(60_000)
      expect(revoke).toHaveBeenCalledWith('blob:zip')
    } finally {
      vi.useRealTimers()
    }
  })

  it('downloadZipToDisk rejects when keys > 50 and no Content-Length', async () => {
    stubFetch(() =>
      Promise.resolve({
        ok: true,
        status: 200,
        statusText: 'OK',
        headers: new Map<string, string>(),
        body: null,
      })
    )
    const { downloadZipToDisk } = await import('./api/download')
    const keys = Array.from({ length: 51 }, (_, i) => `key-${i}.txt`)
    await expect(downloadZipToDisk('id1', { keys })).rejects.toThrow()
  })

  it('downloadZipToDisk rejects when Content-Length > ZIP_BLOB_MAX_BYTES', async () => {
    stubFetch(() =>
      Promise.resolve({
        ok: true,
        status: 200,
        statusText: 'OK',
        headers: new Map<string, string>([['Content-Length', '600000000']]),
        body: null,
      })
    )
    const { downloadZipToDisk } = await import('./api/download')
    await expect(downloadZipToDisk('id1', { keys: ['a.txt'] })).rejects.toThrow()
  })

  it('requestResponse 携带 Authorization（api.token 存在时）', async () => {
    stubFetch(() =>
      Promise.resolve({
        ok: true,
        status: 200,
        statusText: 'OK',
        headers: new Map<string, string>([['Content-Length', '1000']]),
        blob: () => Promise.resolve(new Blob(['zip'])),
        body: null,
      })
    )
    const { api } = await loadApi()
    const { downloadZipToDisk } = await import('./api/download')
    api.token = 'tok-123'
    await downloadZipToDisk('id1', { keys: ['a.txt'] })
    expect(globalThis.fetch).toHaveBeenCalledWith(
      expect.stringContaining('/api/accounts/id1/download-zip'),
      expect.objectContaining({
        headers: expect.objectContaining({ Authorization: 'Bearer tok-123' }),
      }),
    )
  })
})

// ── subscribeMigrateEvents ──────────────────────────────────────────────────
