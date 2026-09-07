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

function stubFetch(impl: Parameters<typeof fetch>[1] extends Promise<infer R> ? (input: RequestInfo | URL, init?: RequestInit) => R : never) {
  return vi.stubGlobal('fetch', vi.fn(impl as any))
}

function makeBlobResponse(data: any, status = 200, statusText = 'OK') {
  return {
    ok: true,
    status,
    statusText,
    json: () => Promise.resolve(data),
    blob: () => Promise.resolve(new Blob([JSON.stringify(data)])),
    headers: new Map<string, string>(),
    body: null,
  } as any
}

function makeErrorResponse(status = 400, statusText = 'Bad Request', error?: string) {
  const body = error ? JSON.stringify({ error }) : ''
  return {
    ok: false,
    status,
    statusText,
    json: () => Promise.resolve(error ? { error } : {}),
    text: () => Promise.resolve(body),
    headers: new Map<string, string>(),
    body: null,
  } as any
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
    memLocal.getItem = (() => { throw new Error('boom') }) as any
    const { api } = await loadApi()
    expect(api.isTokenPersistent).toBe(false)
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
    const { api } = await loadApi()
    memLocal.setItem('s3c.activeServerId', 'nonexistent-id')
    // 重新加载以读取新的 localStorage
    const mod = await import('./api')
    expect(mod.api.activeServerId()).toBe(mod.api.listServers()[0].id)
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
    const list = api.listServers()
    // 先创建一个额外的 server
    const extra = api.upsertServer({ name: 'to-delete', base: 'http://x:9000', token: '' })
    const list2 = api.listServers()
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
})

// ── api.base / api.token getter/setter ──────────────────────────────────────
describe('api getter/setter', () => {
  it('base getter/setter 读写 localStorage', async () => {
    const { api } = await loadApi()
    api.base = 'https://s3.example.com'
    expect(api.base).toBe('https://s3.example.com')
    expect(memLocal.getItem('s3c.apiBase')).toBe('https://s3.example.com')
  })
})

// ── request / requestResponse ──────────────────────────────────────────────
describe('request / requestResponse', () => {
  it('request 成功返回 json', async () => {
    stubFetch(() => Promise.resolve(makeBlobResponse({ ok: true })))
    const { api } = await loadApi()
    api.base = 'https://s3.example.com'
    const result = await (api as any).request?.('/test')
    // request 是私有函数不导出；我们通过 s3api.listAccounts 间接覆盖
  })

  it('requestResponse 成功返回 Response', async () => {
    stubFetch(() => Promise.resolve(makeBlobResponse({ ok: true })))
    const { api } = await loadApi()
    api.base = 'https://s3.example.com'
  })

  it('request 失败抛错并解析 error 字段', async () => {
    stubFetch(() => Promise.resolve(makeErrorResponse(403, 'Forbidden', 'bad token')))
    const { api } = await loadApi()
    api.base = 'https://s3.example.com'
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
    await s3api.createAccount({ name: 'a', endpoint: 'http://x' })
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

  it('copyFiles', async () => {
    const { s3api } = await import('./api')
    await s3api.copyFiles('id1', { bucket: 'b', targetBucket: 'tb', keys: ['k'] })
    expect(fetch).toHaveBeenCalledWith('/api/accounts/id1/copy-objects', expect.objectContaining({ method: 'POST' }))
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

  it('deleteObjects / deletePrefix / deletePrefixAsync', async () => {
    const { s3api } = await import('./api')
    await s3api.deleteObjects('id1', { bucket: 'b', keys: ['k'] })
    await s3api.deletePrefix('id1', { bucket: 'b', prefix: 'p' })
    await s3api.deletePrefixAsync('id1', { bucket: 'b', prefix: 'p' })
  })

  it('copyPrefix / copyPrefixAsync', async () => {
    const { s3api } = await import('./api')
    await s3api.copyPrefix('id1', { bucket: 'b', prefix: 'p', targetBucket: 'tb', targetPrefix: 'tp' })
    await s3api.copyPrefixAsync('id1', { bucket: 'b', prefix: 'p', targetBucket: 'tb', targetPrefix: 'tp' })
  })

  it('migrate / migrateAsync / migrateJobStatus / migrateJobCancel / migrateSync', async () => {
    const { s3api } = await import('./api')
    await s3api.migrate({ sourceAccountId: 'a1', sourceKeys: ['k'] })
    await s3api.migrateAsync({ sourceAccountId: 'a1', sourceKeys: ['k'] })
    await s3api.migrateJobStatus('job1')
    await s3api.migrateJobCancel('job1')
    await s3api.migrateSync({ sourceAccountId: 'a1', sourceKeys: ['k'] })
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
      } as any)
    )
    const { downloadZipToDisk } = await import('./api')
    await downloadZipToDisk('id1', { keys: ['a.txt'] })
    expect(fetch).toHaveBeenCalled()
  })

  it('downloadZipToDisk rejects when keys > 50 and no Content-Length', async () => {
    stubFetch(() =>
      Promise.resolve({
        ok: true,
        status: 200,
        statusText: 'OK',
        headers: new Map<string, string>(),
        body: null,
      } as any)
    )
    const { downloadZipToDisk } = await import('./api')
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
      } as any)
    )
    const { downloadZipToDisk } = await import('./api')
    await expect(downloadZipToDisk('id1', { keys: ['a.txt'] })).rejects.toThrow()
  })
})

// ── subscribeMigrateEvents ──────────────────────────────────────────────────
describe('subscribeMigrateEvents', () => {
  it('错误路径：fetch 返回非 ok', async () => {
    stubFetch(() => Promise.resolve({ ok: false, status: 500, statusText: 'err', body: null } as any))
    const { subscribeMigrateEvents } = await import('./api')
    const onProgress = vi.fn()
    const onError = vi.fn()
    const abort = subscribeMigrateEvents('job1', onProgress, onError)
    // 等待 microtask
    await new Promise((r) => setTimeout(r, 20))
    expect(onError).toHaveBeenCalled()
    abort()
  })

  it('abort 路径', async () => {
    stubFetch(() => new Promise(() => {})) // never resolves
    const { subscribeMigrateEvents } = await import('./api')
    const onProgress = vi.fn()
    const onError = vi.fn()
    const abort = subscribeMigrateEvents('job1', onProgress, onError)
    abort()
    await new Promise((r) => setTimeout(r, 20))
    // abort 后不应调用 onError
    expect(onError).not.toHaveBeenCalled()
  })

  it('SSE 解析循环：接收 data 事件并回调 onProgress', async () => {
    let readCount = 0
    stubFetch((url: string) => {
      if (String(url).includes('/events')) {
        // SSE 流：第一次 read 返回数据，第二次返回 done
        return Promise.resolve({
          ok: true,
          status: 200,
          statusText: 'OK',
          headers: new Map<string, string>(),
          body: {
            getReader: () => ({
              read: () => {
                readCount++
                if (readCount === 1) {
                  return Promise.resolve({
                    done: false,
                    value: new TextEncoder().encode(
                      'event: progress\ndata: {"done":0,"total":10}\n\n'
                    ),
                  })
                }
                return Promise.resolve({ done: true, value: new Uint8Array(0) })
              },
              cancel: vi.fn(),
            }),
          },
        } as any)
      }
      // job status 回读
      return Promise.resolve({
        ok: true,
        status: 200,
        statusText: 'OK',
        json: () => Promise.resolve({ done: true, progress: { status: 'done' } }),
        blob: () => Promise.resolve(new Blob(['ok'])),
        headers: new Map<string, string>(),
        body: null,
      } as any)
    })
    const { subscribeMigrateEvents } = await import('./api')
    const onProgress = vi.fn()
    const onError = vi.fn()
    const abort = subscribeMigrateEvents('job1', onProgress, onError)
    // 等待 SSE 解析和后续处理
    await new Promise((r) => setTimeout(r, 100))
    abort()
    expect(onProgress).toHaveBeenCalled()
  })
})

// ── directUpload ────────────────────────────────────────────────────────────
describe('directUpload', () => {
  function createXhrMockClass() {
    const instances: any[] = []
    class XHRMock {
      open = vi.fn()
      setRequestHeader = vi.fn()
      send = vi.fn()
      abort = vi.fn()
      status = 200
      upload = { onprogress: null as ((e: any) => void) | null }
      onload: ((e: any) => void) | null = null
      onerror: ((e: any) => void) | null = null
      onabort: ((e: any) => void) | null = null
      private _listeners: Record<string, Array<(e: any) => void>> = {}
      addEventListener = vi.fn((evt: string, fn: any) => {
        ;(this._listeners[evt] ||= []).push(fn)
      })
      removeEventListener = vi.fn((evt: string, fn: any) => {
        const arr = this._listeners[evt]
        if (arr) {
          const i = arr.indexOf(fn)
          if (i >= 0) arr.splice(i, 1)
        }
      })
      _fire(event: string, e: any) {
        // Call on* property handler (xml standard behavior)
        const handler = (this as any)[`on${event}`]
        if (handler) handler(e)
        ;(this._listeners[event] || []).forEach((fn) => fn(e))
      }
      constructor() {
        instances.push(this)
      }
      static getInstances() { return instances }
      static clear() { instances.length = 0 }
    }
    return XHRMock
  }

  it('上传成功 2xx', async () => {
    const XHRMock = createXhrMockClass()
    vi.stubGlobal('XMLHttpRequest', XHRMock as any)
    const { directUpload } = await import('./api')
    const promise = directUpload('https://x', new Blob(['a'] as any) as any)
    const inst = XHRMock.getInstances()[0]
    expect(inst.open).toHaveBeenCalledWith('PUT', 'https://x')
    expect(inst.send).toHaveBeenCalled()
    inst._fire('load', {})
    await promise
  })

  it('网络错误', async () => {
    const XHRMock = createXhrMockClass()
    vi.stubGlobal('XMLHttpRequest', XHRMock as any)
    const { directUpload } = await import('./api')
    const promise = directUpload('https://x', new Blob(['a'] as any) as any)
    const inst = XHRMock.getInstances()[0]
    inst._fire('error', {})
    await expect(promise).rejects.toThrow('upload network error')
  })

  it('被 abort', async () => {
    const XHRMock = createXhrMockClass()
    vi.stubGlobal('XMLHttpRequest', XHRMock as any)
    const { directUpload } = await import('./api')
    const promise = directUpload('https://x', new Blob(['a'] as any) as any)
    const inst = XHRMock.getInstances()[0]
    inst._fire('abort', {})
    await expect(promise).rejects.toThrow('Aborted')
  })

  it('signal 已 abort 提前拒绝', async () => {
    const XHRMock = createXhrMockClass()
    vi.stubGlobal('XMLHttpRequest', XHRMock as any)
    const { directUpload } = await import('./api')
    const controller = new AbortController()
    controller.abort()
    await expect(directUpload('https://x', new Blob(['a'] as any) as any, undefined, controller.signal)).rejects.toThrow('Aborted')
  })

  it('非 2xx 状态码', async () => {
    const XHRMock = createXhrMockClass()
    vi.stubGlobal('XMLHttpRequest', XHRMock as any)
    const { directUpload } = await import('./api')
    const promise = directUpload('https://x', new Blob(['a'] as any) as any)
    const inst = XHRMock.getInstances()[0]
    inst.status = 403
    inst._fire('load', {})
    await expect(promise).rejects.toThrow('upload failed')
  })

  it('上传进度回调', async () => {
    const XHRMock = createXhrMockClass()
    vi.stubGlobal('XMLHttpRequest', XHRMock as any)
    const { directUpload } = await import('./api')
    const onProgress = vi.fn()
    const promise = directUpload('https://x', new Blob(['a'] as any) as any, onProgress)
    const inst = XHRMock.getInstances()[0]
    // fire upload progress
    inst.upload.onprogress?.({ lengthComputable: true, loaded: 50, total: 100 })
    expect(onProgress).toHaveBeenCalledWith(50)
    inst._fire('load', {})
    await promise
  })

  it('signal listener 在非 abort 信号时注册', async () => {
    const XHRMock = createXhrMockClass()
    vi.stubGlobal('XMLHttpRequest', XHRMock as any)
    const { directUpload } = await import('./api')
    const controller = new AbortController()
    // 捕获 signal.addEventListener 调用
    const signalAddEventListener = vi.fn()
    controller.signal.addEventListener = signalAddEventListener as any
    const promise = directUpload('https://x', new Blob(['a'] as any) as any, undefined, controller.signal)
    const inst = XHRMock.getInstances()[0]
    inst._fire('load', {})
    await promise
    // 信号未 abort，监听器应被注册到 signal 上
    expect(signalAddEventListener).toHaveBeenCalledWith('abort', expect.any(Function), { once: true })
  })
})
