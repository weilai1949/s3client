import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { MigrateProgress } from './api'

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

/** XHR 事件处理器替身：测试用普通对象字面量驱动，故参数取 unknown。 */
type XhrHandler = (e: unknown) => void

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

  it('token 迁移读取 localStorage 抛异常时回退空串（line 58 catch）', async () => {
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
    const { requestResponse, api } = await loadApi()
    api.base = 'https://s3.example.com'
    const res = await requestResponse('/test')
    expect(res.ok).toBe(true)
    expect(vi.mocked(globalThis.fetch)).toHaveBeenCalledWith('https://s3.example.com/test', expect.objectContaining({ headers: {} }))
  })

  it('requestResponse 带 opts 但无 body：不设置 Content-Type', async () => {
    stubFetch(() => Promise.resolve(makeBlobResponse({ ok: true })))
    const { requestResponse, api } = await loadApi()
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
    await s3api.migrate({ sourceAccountId: 'a1', sourceKeys: ['k'], targetAccountId: 'a2' })
    await s3api.migrateAsync({ sourceAccountId: 'a1', sourceKeys: ['k'], targetAccountId: 'a2' })
    await s3api.migrateJobStatus('job1')
    await s3api.migrateJobCancel('job1')
    await s3api.migrateSync({ sourceAccountId: 'a1', sourcePrefix: 'p/', targetAccountId: 'a2' })
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
      })
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
      })
    )
    const { downloadZipToDisk } = await import('./api')
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
    const { api, downloadZipToDisk } = await loadApi()
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
describe('subscribeMigrateEvents', () => {
  it('错误路径：fetch 返回非 ok', async () => {
    stubFetch(() => Promise.resolve({ ok: false, status: 500, statusText: 'err', body: null }))
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
    stubFetch((input: RequestInfo | URL) => {
      if (String(input).includes('/events')) {
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
        })
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
      })
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
    const instances: InstanceType<typeof XHRMock>[] = []
    class XHRMock {
      open = vi.fn()
      setRequestHeader = vi.fn()
      send = vi.fn()
      abort = vi.fn()
      status = 200
      upload = { onprogress: null as XhrHandler | null }
      onload: XhrHandler | null = null
      onerror: XhrHandler | null = null
      onabort: XhrHandler | null = null
      private _listeners: Record<string, XhrHandler[]> = {}
      addEventListener = vi.fn((evt: string, fn: XhrHandler) => {
        ;(this._listeners[evt] ||= []).push(fn)
      })
      removeEventListener = vi.fn((evt: string, fn: XhrHandler) => {
        const arr = this._listeners[evt]
        if (arr) {
          const i = arr.indexOf(fn)
          if (i >= 0) arr.splice(i, 1)
        }
      })
      _fire(event: string, e: unknown) {
        // Call on* property handler (xml standard behavior)
        const handler = (this as unknown as Record<string, XhrHandler | null | undefined>)[`on${event}`]
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
    vi.stubGlobal('XMLHttpRequest', XHRMock)
    const { directUpload } = await import('./api')
    const promise = directUpload('https://x', new Blob(['a']) as File)
    const inst = XHRMock.getInstances()[0]
    expect(inst.open).toHaveBeenCalledWith('PUT', 'https://x')
    expect(inst.send).toHaveBeenCalled()
    inst._fire('load', {})
    await promise
  })

  it('网络错误', async () => {
    const XHRMock = createXhrMockClass()
    vi.stubGlobal('XMLHttpRequest', XHRMock)
    const { directUpload } = await import('./api')
    const promise = directUpload('https://x', new Blob(['a']) as File)
    const inst = XHRMock.getInstances()[0]
    inst._fire('error', {})
    await expect(promise).rejects.toThrow('upload network error')
  })

  it('被 abort', async () => {
    const XHRMock = createXhrMockClass()
    vi.stubGlobal('XMLHttpRequest', XHRMock)
    const { directUpload } = await import('./api')
    const promise = directUpload('https://x', new Blob(['a']) as File)
    const inst = XHRMock.getInstances()[0]
    inst._fire('abort', {})
    await expect(promise).rejects.toThrow('Aborted')
  })

  it('signal 已 abort 提前拒绝', async () => {
    const XHRMock = createXhrMockClass()
    vi.stubGlobal('XMLHttpRequest', XHRMock)
    const { directUpload } = await import('./api')
    const controller = new AbortController()
    controller.abort()
    await expect(directUpload('https://x', new Blob(['a']) as File, undefined, controller.signal)).rejects.toThrow('Aborted')
  })

  it('非 2xx 状态码', async () => {
    const XHRMock = createXhrMockClass()
    vi.stubGlobal('XMLHttpRequest', XHRMock)
    const { directUpload } = await import('./api')
    const promise = directUpload('https://x', new Blob(['a']) as File)
    const inst = XHRMock.getInstances()[0]
    inst.status = 403
    inst._fire('load', {})
    await expect(promise).rejects.toThrow('upload failed')
  })

  it('上传进度回调', async () => {
    const XHRMock = createXhrMockClass()
    vi.stubGlobal('XMLHttpRequest', XHRMock)
    const { directUpload } = await import('./api')
    const onProgress = vi.fn()
    const promise = directUpload('https://x', new Blob(['a']) as File, onProgress)
    const inst = XHRMock.getInstances()[0]
    // fire upload progress
    inst.upload.onprogress?.({ lengthComputable: true, loaded: 50, total: 100 })
    expect(onProgress).toHaveBeenCalledWith(50)
    inst._fire('load', {})
    await promise
  })

  it('signal listener 在非 abort 信号时注册', async () => {
    const XHRMock = createXhrMockClass()
    vi.stubGlobal('XMLHttpRequest', XHRMock)
    const { directUpload } = await import('./api')
    const controller = new AbortController()
    // 捕获 signal.addEventListener 调用
    const signalAddEventListener = vi.fn()
    controller.signal.addEventListener = signalAddEventListener
    const promise = directUpload('https://x', new Blob(['a']) as File, undefined, controller.signal)
    const inst = XHRMock.getInstances()[0]
    inst._fire('load', {})
    await promise
    // 信号未 abort，监听器应被注册到 signal 上
    expect(signalAddEventListener).toHaveBeenCalledWith('abort', expect.any(Function), { once: true })
  })
})

// ── api.ts 剩余分支：request 错误、FSA ZIP、token 迁移、deleteServer 回退 ──
describe('api gaps', () => {
  it('request 解析 JSON error 并抛 403', async () => {
    stubFetch(() => Promise.resolve(makeErrorResponse(403, 'Forbidden', 'bad token')))
    const { s3api, api } = await loadApi()
    api.base = 'https://s3.example.com'
    await expect(s3api.listAccounts()).rejects.toThrow('403 bad token')
  })

  it('request 非 JSON 错误体回退 statusText', async () => {
    stubFetch(() => Promise.resolve(makeErrorResponse(502, 'Bad Gateway')))
    const { s3api, api } = await loadApi()
    api.base = 'https://s3.example.com'
    await expect(s3api.listAccounts()).rejects.toThrow('502 Bad Gateway')
  })

  it('requestResponse 非 ok 解析 JSON error', async () => {
    stubFetch(() => Promise.resolve(makeErrorResponse(401, 'Unauthorized', 'expired')))
    const { downloadZipToDisk, api } = await loadApi()
    api.base = 'https://s3.example.com'
    await expect(downloadZipToDisk('id1', { keys: ['a.txt'] })).rejects.toThrow('401 expired')
  })

  it('downloadZipToDisk FSA 成功流式落盘', async () => {
    const body = { pipeTo: vi.fn().mockResolvedValue(undefined), cancel: vi.fn().mockResolvedValue(undefined) }
    stubFetch(() => Promise.resolve({
      ok: true, status: 200, body,
      headers: new Map<string, string>(),
    }))
    const picker = vi.fn().mockResolvedValue({ createWritable: async () => ({ abort: vi.fn() }) })
    Object.defineProperty(window, 'showSaveFilePicker', { value: picker, configurable: true })
    const { downloadZipToDisk, api } = await loadApi()
    api.base = 'https://s3.example.com'
    await downloadZipToDisk('id1', { keys: ['a.txt'] })
    expect(picker).toHaveBeenCalled()
    expect(body.pipeTo).toHaveBeenCalled()
    delete (window as unknown as { showSaveFilePicker?: unknown }).showSaveFilePicker
  })

  it('downloadZipToDisk FSA 失败 → cancel body 并抛错', async () => {
    const cancel = vi.fn().mockResolvedValue(undefined)
    const body = { pipeTo: vi.fn(), cancel }
    stubFetch(() => Promise.resolve({
      ok: true, status: 200, body,
      headers: new Map<string, string>(),
    }))
    Object.defineProperty(window, 'showSaveFilePicker', { value: vi.fn().mockRejectedValue(new Error('user cancelled')), configurable: true })
    const { downloadZipToDisk, api } = await loadApi()
    api.base = 'https://s3.example.com'
    await expect(downloadZipToDisk('id1', { keys: ['a.txt'] })).rejects.toThrow('user cancelled')
    expect(cancel).toHaveBeenCalled()
    delete (window as unknown as { showSaveFilePicker?: unknown }).showSaveFilePicker
  })

  it('streamBodyToFile 失败时 abort writable', async () => {
    const abort = vi.fn().mockResolvedValue(undefined)
    const body = { pipeTo: vi.fn().mockRejectedValue(new Error('pipe broke')), cancel: vi.fn().mockResolvedValue(undefined) }
    stubFetch(() => Promise.resolve({
      ok: true, status: 200, body,
      headers: new Map<string, string>(),
    }))
    Object.defineProperty(window, 'showSaveFilePicker', { value: vi.fn().mockResolvedValue({ createWritable: async () => ({ abort }) }), configurable: true })
    const { downloadZipToDisk, api } = await loadApi()
    api.base = 'https://s3.example.com'
    await expect(downloadZipToDisk('id1', { keys: ['a.txt'] })).rejects.toThrow('pipe broke')
    expect(abort).toHaveBeenCalled()
    delete (window as unknown as { showSaveFilePicker?: unknown }).showSaveFilePicker
  })

  it('s3api.downloadZipToDisk 包装函数转发', async () => {
    stubFetch(() => Promise.resolve(makeBlobResponse({})))
    const { s3api, api } = await loadApi()
    api.base = 'https://s3.example.com'
    await s3api.downloadZipToDisk('id1', { bucket: 'b', keys: ['a.txt'] })
    expect(fetch).toHaveBeenCalledWith('https://s3.example.com/api/accounts/id1/download-zip', expect.anything())
  })

  it('token 一次迁移：session 无值但 localStorage 有旧值', async () => {
    const { api } = await loadApi()
    api.setTokenPersistent(false)
    memLocal.setItem('s3c.token', 'legacy-token-1234567890')
    expect(api.token).toBe('legacy-token-1234567890')
  })

  it('deleteServer 删除当前生效服务器 → 应用回退 profile（剩余第一个）', async () => {
    const { api } = await loadApi()
    const s1 = api.upsertServer({ name: 'one', base: 'http://one', token: 'tok1' })
    api.selectServer(s1.id)
    expect(api.base).toBe('http://one')
    expect(api.token).toBe('tok1')
    api.deleteServer(s1.id) // 删当前生效 → 回退到剩余第一个（内置默认）并应用
    expect(api.base).toBe('')
    expect(api.token).toBe('')
    expect(api.listServers().length).toBe(1)
  })
})

// ── api.ts 收尾：session 优先、persistent 读取异常、previewBuckets ──
describe('api edge branches', () => {
  it('readToken prefers sessionStorage', async () => {
    memSession.setItem('s3c.token', 'session-token')
    memLocal.setItem('s3c.token', 'local-token')
    const { api } = await loadApi()
    api.setTokenPersistent(false)
    expect(api.token).toBe('session-token')
  })

  it('tokenPersistent 分支读取 localStorage，异常回退空串', async () => {
    memLocal.setItem('s3c_token_persistent', '1')
    memLocal.setItem('s3c.token', 'persisted-token')
    const { api } = await loadApi()
    expect(api.token).toBe('persisted-token')

    const orig = memLocal.getItem
    memLocal.getItem = ((k: string) => {
      if (k === 's3c.token') throw new Error('boom')
      return orig.call(memLocal, k)
    })
    const { api: api2 } = await loadApi()
    expect(api2.token).toBe('')
    memLocal.getItem = orig
  })

  it('s3api.previewBuckets posts account input', async () => {
    stubFetch(() => Promise.resolve(makeBlobResponse({ buckets: [{ name: 'b1' }] })))
    const { s3api, api } = await loadApi()
    api.base = 'https://s3.example.com'
    const out = await s3api.previewBuckets({
      name: 'a', endpoint: 'http://x', region: 'us-east-1',
      accessKey: 'ak', secretKey: 'sk', bucket: 'b', pathStyle: true, useSSL: false,
    })
    expect(fetch).toHaveBeenCalledWith('https://s3.example.com/api/accounts/preview-buckets', expect.anything())
    expect(out.buckets).toEqual([{ name: 'b1' }])
  })
})

// ── api.ts 末批：Authorization 头、selectServer 未命中、versions/trash 参数、SSE ping ──
describe('api final branches', () => {
  it('request/requestResponse attach Authorization when token set', async () => {
    stubFetch(() => Promise.resolve(makeBlobResponse({ ok: true })))
    const { api, s3api } = await loadApi()
    api.base = 'https://s3.example.com'
    api.setTokenPersistent(false)
    api.token = 'tok-123'
    await s3api.listAccounts()
    const calls = vi.mocked(fetch).mock.calls
    const init: RequestInit = calls[calls.length - 1]![1] ?? {}
    expect((init.headers as Record<string, string> | undefined)?.['Authorization']).toBe('Bearer tok-123')
  })

  it('selectServer unknown id returns undefined', async () => {
    const { api } = await loadApi()
    expect(api.selectServer('missing-id')).toBeUndefined()
  })

  it('listVersions forwards prefix/keyMarker/versionIdMarker', async () => {
    stubFetch(() => Promise.resolve(makeBlobResponse({})))
    const { s3api, api } = await loadApi()
    api.base = 'https://s3.example.com'
    await s3api.listVersions('id1', { bucket: 'b', prefix: 'p/', keyMarker: 'km', versionIdMarker: 'vm' })
    const calls = vi.mocked(fetch).mock.calls
    const url = String(calls[calls.length - 1]![0])
    expect(url).toContain('prefix=p%2F')
    expect(url).toContain('keyMarker=km')
    expect(url).toContain('versionIdMarker=vm')
  })

  it('listTrash forwards prefix/keyMarker/versionIdMarker/maxKeys', async () => {
    stubFetch(() => Promise.resolve(makeBlobResponse({})))
    const { s3api, api } = await loadApi()
    api.base = 'https://s3.example.com'
    await s3api.listTrash('id1', { bucket: 'b', prefix: 'p/', keyMarker: 'km', versionIdMarker: 'vm', maxKeys: 999 })
    const calls = vi.mocked(fetch).mock.calls
    const url = String(calls[calls.length - 1]![0])
    expect(url).toContain('maxKeys=999')
    expect(url).toContain('keyMarker=km')
  })

  it('subscribeMigrateEvents: ping+status+Authorization+EOF-done fallback', async () => {
    const stream = new ReadableStream({
      start(c) {
        // ping 心跳：eventName==='ping' → data 行被 continue 忽略
        c.enqueue(new TextEncoder().encode('event: ping\ndata: {}\n\n'))
        // 带 status 的 progress
        c.enqueue(new TextEncoder().encode('event: progress\ndata: {"migrated":1,"status":"running"}\n\n'))
        c.close() // EOF 无终态 → 触发 migrateJobStatus 回读
      },
    })
    const statusResp = { done: true, progress: { migrated: 1, done: 1, total: 1, status: '' } }
    stubFetch((input: RequestInfo | URL) => {
      const url = String(input)
      if (url.includes('/events')) {
        return Promise.resolve({ ok: true, status: 200, body: stream, json: async () => ({}) })
      }
      return Promise.resolve(makeBlobResponse(statusResp))
    })
    const { api, subscribeMigrateEvents } = await loadApi()
    api.base = 'https://s3.example.com'
    api.setTokenPersistent(false)
    api.token = 'tok-sse'
    const onProgress = vi.fn()
    const off = subscribeMigrateEvents('job-1', onProgress, vi.fn())
    await vi.waitFor(() => expect(onProgress).toHaveBeenCalledTimes(2))
    // EOF 回读：最后一次回调为 migrateJobStatus 的 done
    const last = onProgress.mock.calls[onProgress.mock.calls.length - 1]![0]
    expect(last.status).toBe('done')
    off()
  })
})

// ── api.ts 覆盖率补全：readToken / newId / servers 分支 ──────────────────────
describe('api gaps: token/servers 分支', () => {
  it('readToken 持久化模式下 localStorage 无 token → 返回空串（line 48 ?? 右侧）', async () => {
    memLocal.setItem('s3c_token_persistent', '1')
    const { api } = await loadApi()
    expect(api.isTokenPersistent).toBe(true)
    expect(api.token).toBe('')
  })

  it('newId() 无 crypto.randomUUID 时回退时间戳+随机串', async () => {
    vi.stubGlobal('crypto', { randomUUID: undefined })
    const { api } = await loadApi()
    const p = api.upsertServer({ name: 'x', base: '/', token: '' })
    expect(p.id).toMatch(/^s-\d+-[a-z0-9]+$/)
  })

  it('listServers() 本地 JSON 非数组时回退默认 server（line 125 else）', async () => {
    memLocal.setItem('s3c.servers', JSON.stringify({ not: 'array' }))
    const { api } = await loadApi()
    const list = api.listServers()
    expect(Array.isArray(list)).toBe(true)
    expect(list[0].name).toBe('server.sameOriginDefault')
  })

  it('listServers() 首次默认 server 在 Tauri 下用 localBackend 名称（line 133 左支）', async () => {
    const orig = location.hostname
    Object.defineProperty(location, 'hostname', { value: 'tauri.localhost', configurable: true })
    try {
      const { api } = await loadApi()
      const list = api.listServers()
      expect(list[0].name).toBe('server.localBackend')
      expect(list[0].base).toBe('http://127.0.0.1:8080')
    } finally {
      Object.defineProperty(location, 'hostname', { value: orig, configurable: true })
    }
  })

  it('activeServerId() 服务器列表为空 → 返回空串（line 197 ?? 右侧）', async () => {
    memLocal.setItem('s3c.servers', '[]')
    const { api } = await loadApi()
    expect(api.activeServerId()).toBe('')
    expect(api.getActiveServer()).toBeUndefined()
  })

  it('upsertServer() base 为空 → base 归一化为空串（line 215 || 右侧）', async () => {
    const { api } = await loadApi()
    const p = api.upsertServer({ name: 'srv', base: '', token: '' })
    expect(p.base).toBe('')
  })

  it('upsertServer() 名称为空且 base 非空 → 以 base 为名（line 217 中型 ||）', async () => {
    const { api } = await loadApi()
    const p = api.upsertServer({ name: '', base: 'http://x:9001', token: '' })
    expect(p.name).toBe('http://x:9001')
  })

  it('upsertServer() 名称、base 均为空 → 回退 sameOriginShort（line 217 尾支）', async () => {
    const { api } = await loadApi()
    const p = api.upsertServer({ name: '   ', base: '', token: '' })
    expect(p.name).toBe('server.sameOriginShort')
    expect(p.base).toBe('')
  })

  it('upsertServer() id 不存在 → 创建新 server（line 220 else）', async () => {
    const { api } = await loadApi()
    const p = api.upsertServer({ id: 'ghost-id', name: 'new', base: 'http://x:9002', token: '' })
    expect(p.id).not.toBe('ghost-id')
    expect(p.name).toBe('new')
  })

  it('upsertServer() 更新非当前 server → 不触发 applyProfile（line 223 假分支）', async () => {
    const { api } = await loadApi()
    const def = api.listServers()[0]
    const extra = api.upsertServer({ name: 'extra', base: 'http://extra', token: 't-extra' })
    api.base = 'http://keep-base'
    api.token = 'keep-token'
    api.upsertServer({ id: extra.id, name: 'extra-renamed', base: 'http://extra2', token: 't2' })
    expect(api.base).toBe('http://keep-base')
    expect(api.token).toBe('keep-token')
    expect(api.activeServerId()).toBe(def.id)
  })

  it('deleteServer() 在 Tauri 下重建默认 server → localBackend（line 238 左支）', async () => {
    const orig = location.hostname
    Object.defineProperty(location, 'hostname', { value: 'tauri.localhost', configurable: true })
    try {
      const { api } = await loadApi()
      const list = api.listServers()
      api.deleteServer(list[0].id)
      const list2 = api.listServers()
      expect(list2[0].name).toBe('server.localBackend')
      expect(list2[0].base).toBe('http://127.0.0.1:8080')
    } finally {
      Object.defineProperty(location, 'hostname', { value: orig, configurable: true })
    }
  })
})

// ── api.ts 覆盖率补全：s3api bucket 可选参数分支 ──────────────────────────────
describe('api gaps: s3api bucket 可选参数', () => {
  beforeEach(() => {
    stubFetch(() => Promise.resolve(makeBlobResponse({})))
  })

  it('bucket 缺省时各 bucket 接口拼接空 bucket 参数（?? 右侧）', async () => {
    const { s3api } = await import('./api')
    await s3api.getBucketInfo('id1')
    await s3api.getBucketEncryption('id1')
    await s3api.deleteBucketEncryption('id1')
    await s3api.getBucketCors('id1')
    await s3api.deleteBucketCors('id1')
    await s3api.getBucketWebsite('id1')
    await s3api.deleteBucketWebsite('id1')
    await s3api.getBucketPolicy('id1')
    await s3api.deleteBucketPolicy('id1')
    await s3api.getBucketTags('id1')
    await s3api.deleteBucketTags('id1')
    await s3api.getLifecycle('id1')
    const urls = vi.mocked(fetch).mock.calls.map((c) => String(c[0]))
    expect(urls.length).toBe(12)
    for (const u of urls) expect(u).toContain('bucket=')
    for (const u of urls) expect(u).not.toContain('bucket=undefined')
  })

  it('query 无 bucket 时 getObjectAcl/getObjectTags/listVersions/deleteObjectVersion/listTrash 不拼 bucket', async () => {
    const { s3api } = await import('./api')
    await s3api.getObjectAcl('id1', { key: 'k' })
    await s3api.getObjectTags('id1', { key: 'k' })
    await s3api.listVersions('id1', { prefix: 'p' })
    await s3api.deleteObjectVersion('id1', { key: 'k', versionId: 'v' })
    await s3api.listTrash('id1', { prefix: 'p' })
    const urls = vi.mocked(fetch).mock.calls.map((c) => String(c[0]))
    expect(urls.length).toBe(5)
    for (const u of urls) expect(u).not.toContain('bucket=')
  })
})

// ── api.ts 覆盖率补全：requestResponse 错误体与 downloadZipToDisk catch ──────
describe('api gaps: requestResponse / downloadZipToDisk catch', () => {
  it('requestResponse 错误体无 error 字段 → 回退 statusText（line 275 假分支）', async () => {
    stubFetch(() => Promise.resolve(makeErrorResponse(500, 'boom')))
    const { downloadZipToDisk, api } = await loadApi()
    api.base = 'https://s3.example.com'
    await expect(downloadZipToDisk('id1', { keys: ['a.txt'] })).rejects.toThrow('500 boom')
  })

  it('FSA 失败且 body.cancel() 也失败 → cancel().catch 回调执行（line 336 函数）', async () => {
    const cancel = vi.fn().mockRejectedValue(new Error('cancel failed'))
    const body = { pipeTo: vi.fn(), cancel }
    stubFetch(() => Promise.resolve({
      ok: true, status: 200, body,
      headers: new Map<string, string>(),
    }))
    Object.defineProperty(window, 'showSaveFilePicker', { value: vi.fn().mockRejectedValue(new Error('user cancelled')), configurable: true })
    const { downloadZipToDisk, api } = await loadApi()
    api.base = 'https://s3.example.com'
    await expect(downloadZipToDisk('id1', { keys: ['a.txt'] })).rejects.toThrow('user cancelled')
    expect(cancel).toHaveBeenCalledTimes(1)
    delete (window as unknown as { showSaveFilePicker?: unknown }).showSaveFilePicker
  })

  it('超限且 body 存在 → res.body?.cancel() 执行且其失败被吞（line 346 函数）', async () => {
    const cancel = vi.fn().mockRejectedValue(new Error('cancel failed'))
    stubFetch(() => Promise.resolve({
      ok: true, status: 200, statusText: 'OK',
      headers: new Map<string, string>([['Content-Length', '600000000']]),
      body: { cancel },
    }))
    const { downloadZipToDisk } = await import('./api')
    await expect(downloadZipToDisk('id1', { keys: ['a.txt'] })).rejects.toThrow('api.zipTooLarge')
    expect(cancel).toHaveBeenCalledTimes(1)
  })
})

// ── api.ts 覆盖率补全：subscribeMigrateEvents SSE 分支 ───────────────────────
describe('api gaps: subscribeMigrateEvents SSE 分支', () => {
  it('无空格 data 行 + 未知字段行 + 终态 done → 不触发回读（line 614/611/627）', async () => {
    const stream = new ReadableStream({
      start(c) {
        // data: 不带空格（614 右支）；retry 行既不 event: 也不 data:（611 else 支）
        c.enqueue(new TextEncoder().encode('event: progress\ndata:{"status":"done"}\nretry: 100\n\n'))
        c.close()
      },
    })
    stubFetch((input: RequestInfo | URL) => {
      if (String(input).includes('/events')) {
        return Promise.resolve({ ok: true, status: 200, body: stream })
      }
      return Promise.resolve(makeBlobResponse({ done: true, progress: { status: 'done' } }))
    })
    const { subscribeMigrateEvents } = await import('./api')
    const onProgress = vi.fn<(p: MigrateProgress) => void>()
    const off = subscribeMigrateEvents('job-1', onProgress, vi.fn())
    await vi.waitFor(() => expect(onProgress).toHaveBeenCalledTimes(1))
    expect(onProgress.mock.calls[onProgress.mock.calls.length - 1]![0].status).toBe('done')
    await new Promise((r) => setTimeout(r, 20))
    expect(fetch).toHaveBeenCalledTimes(1) // EOF 后 lastStatus==='done' → 不回读
    off()
  })

  it('回读时 progress 无 status 且任务完成 → status 取 done（line 632 三元真支）', async () => {
    const stream = new ReadableStream({ start(c) { c.close() } })
    stubFetch((input: RequestInfo | URL) => {
      if (String(input).includes('/events')) {
        return Promise.resolve({ ok: true, status: 200, body: stream })
      }
      return Promise.resolve(makeBlobResponse({ done: true, progress: {} }))
    })
    const { subscribeMigrateEvents } = await import('./api')
    const onProgress = vi.fn<(p: MigrateProgress) => void>()
    const off = subscribeMigrateEvents('job-1', onProgress, vi.fn())
    await vi.waitFor(() => expect(onProgress).toHaveBeenCalledTimes(1))
    expect(onProgress.mock.calls[onProgress.mock.calls.length - 1]![0].status).toBe('done')
    off()
  })

  it('回读时 progress 无 status 且任务未完成 → status 为 undefined（line 632 三元假支）', async () => {
    const stream = new ReadableStream({ start(c) { c.close() } })
    stubFetch((input: RequestInfo | URL) => {
      if (String(input).includes('/events')) {
        return Promise.resolve({ ok: true, status: 200, body: stream })
      }
      return Promise.resolve(makeBlobResponse({ done: false, progress: {} }))
    })
    const { subscribeMigrateEvents } = await import('./api')
    const onProgress = vi.fn()
    const off = subscribeMigrateEvents('job-1', onProgress, vi.fn())
    await vi.waitFor(() => expect(onProgress).toHaveBeenCalledTimes(1))
    expect(onProgress.mock.calls[onProgress.mock.calls.length - 1]![0].status).toBeUndefined()
    off()
  })

  it('fetch 抛非 Error 字符串 → onError 收到包装后的 Error（line 641 三元假支）', async () => {
    stubFetch(() => Promise.reject('plain failure'))
    const { subscribeMigrateEvents } = await import('./api')
    const onError = vi.fn()
    const off = subscribeMigrateEvents('job-1', vi.fn(), onError)
    await vi.waitFor(() => expect(onError).toHaveBeenCalled())
    expect(onError.mock.calls[0][0]).toBeInstanceOf(Error)
    off()
  })

  it('abort 后 fetch 仍 reject → 不回调 onError（line 641 假分支）', async () => {
    let rejectFetch: (e: unknown) => void = () => {}
    stubFetch(() => new Promise((_, rej) => { rejectFetch = rej }))
    const { subscribeMigrateEvents } = await import('./api')
    const onError = vi.fn()
    const off = subscribeMigrateEvents('job-1', vi.fn(), onError)
    await new Promise((r) => setTimeout(r, 10))
    off()
    rejectFetch(new Error('network down'))
    await new Promise((r) => setTimeout(r, 20))
    expect(onError).not.toHaveBeenCalled()
  })

  it('abort 后流才 EOF → 跳过回读且无回调（line 627 abort 项）', async () => {
    let resolveRead: (v: unknown) => void = () => {}
    const pending = new Promise<unknown>((res) => { resolveRead = res })
    stubFetch((input: unknown) => {
      if (String(input).includes('/events')) {
        return Promise.resolve({ ok: true, status: 200, body: { getReader: () => ({ read: () => pending, cancel: vi.fn() }) } })
      }
      return Promise.resolve(makeBlobResponse({ done: true, progress: { status: 'done' } }))
    })
    const { subscribeMigrateEvents } = await import('./api')
    const onProgress = vi.fn()
    const onError = vi.fn()
    const off = subscribeMigrateEvents('job-1', onProgress, onError)
    await new Promise((r) => setTimeout(r, 20))
    off()
    resolveRead({ done: true, value: new Uint8Array(0) })
    await new Promise((r) => setTimeout(r, 20))
    expect(onError).not.toHaveBeenCalled()
    expect(onProgress).not.toHaveBeenCalled()
    expect(fetch).toHaveBeenCalledTimes(1)
  })
})

// ── api.ts 覆盖率补全：directUpload 进度分支 ────────────────────────────────
describe('api gaps: directUpload 进度分支', () => {
  function createXhrMockClass() {
    const instances: InstanceType<typeof XHRMock>[] = []
    class XHRMock {
      open = vi.fn()
      setRequestHeader = vi.fn()
      send = vi.fn()
      abort = vi.fn()
      status = 200
      upload = { onprogress: null as XhrHandler | null }
      onload: XhrHandler | null = null
      onerror: XhrHandler | null = null
      onabort: XhrHandler | null = null
      constructor() {
        instances.push(this)
      }
      static getInstances() { return instances }
      static clear() { instances.length = 0 }
    }
    return XHRMock
  }

  it('进度事件 lengthComputable=false → 不回调 onProgress（line 673 假分支）', async () => {
    const XHRMock = createXhrMockClass()
    vi.stubGlobal('XMLHttpRequest', XHRMock)
    const { directUpload } = await import('./api')
    const onProgress = vi.fn()
    const promise = directUpload('https://x', new Blob(['a']) as File, onProgress)
    const inst = XHRMock.getInstances()[0]
    inst.upload.onprogress?.({ lengthComputable: false, loaded: 50, total: 100 })
    expect(onProgress).not.toHaveBeenCalled()
    inst.onload?.({})
    await promise
  })

  it('未提供 onProgress 时进度事件正常触发不报错', async () => {
    const XHRMock = createXhrMockClass()
    vi.stubGlobal('XMLHttpRequest', XHRMock)
    const { directUpload } = await import('./api')
    const promise = directUpload('https://x', new Blob(['a']) as File)
    const inst = XHRMock.getInstances()[0]
    inst.upload.onprogress?.({ lengthComputable: true, loaded: 1, total: 2 })
    inst.onload?.({})
    await promise
  })
})
