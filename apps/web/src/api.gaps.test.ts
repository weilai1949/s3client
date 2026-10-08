// api.gaps.test.ts —— 自 api.test.ts 拆出（KNOWN_ISSUES #60）：api gaps 补充分支。
// 断言逐字搬移，与拆分前等价；其余范围见 api.test.ts（基础/请求/s3api）与 api.transfer.test.ts。
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
describe('api gaps: token/servers 分支', () => {
  it('readToken 持久化模式下 localStorage 无 token → 返回空串', async () => {
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

  it('listServers() 本地 JSON 非数组时回退默认 server', async () => {
    memLocal.setItem('s3c.servers', JSON.stringify({ not: 'array' }))
    const { api } = await loadApi()
    const list = api.listServers()
    expect(Array.isArray(list)).toBe(true)
    expect(list[0].name).toBe('server.sameOriginDefault')
  })

  it('listServers() 首次默认 server 在 Tauri 下用 localBackend 名称', async () => {
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

  it('activeServerId() 服务器列表为空 → 返回空串', async () => {
    memLocal.setItem('s3c.servers', '[]')
    const { api } = await loadApi()
    expect(api.activeServerId()).toBe('')
    expect(api.getActiveServer()).toBeUndefined()
  })

  it('upsertServer() base 为空 → base 归一化为空串', async () => {
    const { api } = await loadApi()
    const p = api.upsertServer({ name: 'srv', base: '', token: '' })
    expect(p.base).toBe('')
  })

  it('upsertServer() 名称为空且 base 非空 → 以 base 为名', async () => {
    const { api } = await loadApi()
    const p = api.upsertServer({ name: '', base: 'http://x:9001', token: '' })
    expect(p.name).toBe('http://x:9001')
  })

  it('upsertServer() 名称、base 均为空 → 回退 sameOriginShort', async () => {
    const { api } = await loadApi()
    const p = api.upsertServer({ name: '   ', base: '', token: '' })
    expect(p.name).toBe('server.sameOriginShort')
    expect(p.base).toBe('')
  })

  it('upsertServer() id 不存在 → 创建新 server', async () => {
    const { api } = await loadApi()
    const p = api.upsertServer({ id: 'ghost-id', name: 'new', base: 'http://x:9002', token: '' })
    expect(p.id).not.toBe('ghost-id')
    expect(p.name).toBe('new')
  })

  it('upsertServer() 更新非当前 server → 不触发 applyProfile', async () => {
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

  it('deleteServer() 在 Tauri 下重建默认 server → localBackend', async () => {
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

  it('multipartParts：path id 编码、key/uploadId 必带、bucket 可选', async () => {
    const { s3api } = await import('./api')
    await s3api.multipartParts('id/1', { key: 'dir/a b.bin', uploadId: 'U 1' })
    await s3api.multipartParts('id2', { bucket: 'bkt', key: 'k', uploadId: 'U2' })
    const urls = vi.mocked(fetch).mock.calls.map((c) => String(c[0]))
    expect(urls[0]).toContain('/api/accounts/id%2F1/multipart/parts?')
    expect(urls[0]).toContain('key=dir%2Fa+b.bin')
    expect(urls[0]).toContain('uploadId=U+1')
    expect(urls[0]).not.toContain('bucket=')
    expect(urls[1]).toContain('bucket=bkt')
  })
})

// ── api.ts 覆盖率补全：requestResponse 错误体与 downloadZipToDisk catch ──────
describe('api gaps: requestResponse / downloadZipToDisk catch', () => {
  it('requestResponse 错误体无 error 字段 → 回退 statusText', async () => {
    stubFetch(() => Promise.resolve(makeErrorResponse(500, 'boom')))
    const { api } = await loadApi()
    const { downloadZipToDisk } = await import('./api/download')
    api.base = 'https://s3.example.com'
    await expect(downloadZipToDisk('id1', { keys: ['a.txt'] })).rejects.toThrow('500 boom')
  })

  it('FSA 失败且 body.cancel() 也失败 → cancel().catch 回调执行', async () => {
    const cancel = vi.fn().mockRejectedValue(new Error('cancel failed'))
    const body = { pipeTo: vi.fn(), cancel }
    stubFetch(() => Promise.resolve({
      ok: true, status: 200, body,
      headers: new Map<string, string>(),
    }))
    Object.defineProperty(window, 'showSaveFilePicker', { value: vi.fn().mockRejectedValue(new Error('user cancelled')), configurable: true })
    const { api } = await loadApi()
    const { downloadZipToDisk } = await import('./api/download')
    api.base = 'https://s3.example.com'
    await expect(downloadZipToDisk('id1', { keys: ['a.txt'] })).rejects.toThrow('user cancelled')
    expect(cancel).toHaveBeenCalledTimes(1)
    delete (window as unknown as { showSaveFilePicker?: unknown }).showSaveFilePicker
  })

  it('超限且 body 存在 → res.body?.cancel() 执行且其失败被吞', async () => {
    const cancel = vi.fn().mockRejectedValue(new Error('cancel failed'))
    stubFetch(() => Promise.resolve({
      ok: true, status: 200, statusText: 'OK',
      headers: new Map<string, string>([['Content-Length', '600000000']]),
      body: { cancel },
    }))
    const { downloadZipToDisk } = await import('./api/download')
    await expect(downloadZipToDisk('id1', { keys: ['a.txt'] })).rejects.toThrow('api.zipTooLarge')
    expect(cancel).toHaveBeenCalledTimes(1)
  })
})

// ── api.ts 覆盖率补全：subscribeMigrateEvents SSE 分支 ───────────────────────
describe('api gaps: subscribeMigrateEvents SSE 分支', () => {
  it('无空格 data 行 + 未知字段行 + 终态 done → 不触发回读', async () => {
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

  it('回读时 progress 无 status 且任务完成 → status 取 done', async () => {
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

  it('回读时 progress 无 status 且任务未完成 → status 为 running（非终态，调用方继续轮询等待）', async () => {
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
    // 旧实现 status=undefined 会让调用方（ctxDeleteFolder/DestDialog）的
    // `p.status === 'done'|'cancelled'` 判断永不成立 → Promise 悬挂；
    // 新实现合成非终态 'running'，调用方知道仍在运行、继续等待轮询终态。
    expect(onProgress.mock.calls[onProgress.mock.calls.length - 1]![0].status).toBe('running')
    off()
  })

  it('fetch 抛非 Error 字符串 → onError 收到包装后的 Error', async () => {
    stubFetch(() => Promise.reject('plain failure'))
    const { subscribeMigrateEvents } = await import('./api')
    const onError = vi.fn()
    const off = subscribeMigrateEvents('job-1', vi.fn(), onError)
    await vi.waitFor(() => expect(onError).toHaveBeenCalled())
    expect(onError.mock.calls[0][0]).toBeInstanceOf(Error)
    off()
  })

  it('abort 后 fetch 仍 reject → 不回调 onError', async () => {
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

  it('abort 后流才 EOF → 跳过回读且无回调', async () => {
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

  it('进度事件 lengthComputable=false → 不回调 onProgress', async () => {
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

// ── api.ts 覆盖率补全：per-server token 持久化读取/清理 + 迁移回退 + 轮询超时 ──
describe('api gaps: per-server token 持久化与迁移回退', () => {
  it('readServerToken 持久化开启时从 localStorage 恢复 per-server token', async () => {
    memLocal.setItem('s3c_token_persistent', '1')
    const { api } = await loadApi()
    const srv = api.upsertServer({ name: 'a', base: 'http://a.example:9000', token: 'srv-tok-123456' })
    // 清掉 session 副本，仅留 localStorage（模拟重开浏览器：sessionStorage 已清、持久化仍在）
    memSession.removeItem(`s3c.token.${srv.id}`)
    const found = api.listServers().find((s) => s.id === srv.id)
    expect(found?.token).toBe('srv-tok-123456')
  })

  it('readServerToken 持久化读取 localStorage 抛异常 → 返回空串', async () => {
    memLocal.setItem('s3c_token_persistent', '1')
    const { api } = await loadApi()
    const srv = api.upsertServer({ name: 'a', base: 'http://a.example:9000', token: 'srv-tok-abcdef' })
    memSession.removeItem(`s3c.token.${srv.id}`)
    const orig = memLocal.getItem
    memLocal.getItem = ((k: string) => {
      if (k.startsWith('s3c.token.')) throw new Error('boom')
      return orig.call(memLocal, k)
    }) as typeof memLocal.getItem
    const found = api.listServers().find((s) => s.id === srv.id)
    expect(found?.token).toBe('')
    memLocal.getItem = orig
  })

  it('writeServerToken 持久化下清空 token → 同步移除 localStorage 副本', async () => {
    memLocal.setItem('s3c_token_persistent', '1')
    const { api } = await loadApi()
    const srv = api.upsertServer({ name: 'a', base: 'http://a.example:9000', token: 'srv-tok-xyz' })
    expect(memLocal.getItem(`s3c.token.${srv.id}`)).toBe('srv-tok-xyz')
    api.upsertServer({ id: srv.id, name: 'a', base: 'http://a.example:9000', token: '' })
    expect(memLocal.getItem(`s3c.token.${srv.id}`)).toBeNull()
  })

  it('迁移内嵌 token 时读取 activeServerId 抛异常 → 回退空串且迁移不中断', async () => {
    memLocal.setItem(
      's3c.servers',
      JSON.stringify([{ id: 'srv-1', name: 'old', base: 'http://x:9000', token: 'legacy-1' }]),
    )
    memLocal.setItem('s3c.activeServerId', 'srv-1')
    const orig = memLocal.getItem
    memLocal.getItem = ((k: string) => {
      if (k === 's3c.activeServerId') throw new Error('boom')
      return orig.call(memLocal, k)
    }) as typeof memLocal.getItem
    const { api } = await loadApi()
    api.listServers() // 触发 readServers() 内的一次性迁移
    // activeId 回读抛异常被吞（catch 返回 ''），迁移仍完成：per-server token 落到 sessionStorage
    expect(memSession.getItem('s3c.token.srv-1')).toBe('legacy-1')
    memLocal.getItem = orig
  })

  it('迁移旧条目缺 name/base 时补空串；缺 id 的条目无法寻址 → 丢弃并修复存储', async () => {
    // 极旧格式：条目有 id + token 但缺 name/base，应补空串并完成 token 迁移。
    memLocal.setItem('s3c.servers', JSON.stringify([{ id: 'legacy-partial', token: 'legacy-partial-tok' }]))
    const { api } = await loadApi()
    const list = api.listServers()
    expect(list.map((s) => s.id)).toEqual(['legacy-partial'])
    expect(list[0]?.name).toBe('')
    expect(list[0]?.base).toBe('')
    // token 已迁移到 per-server 存储
    expect(list[0]?.token).toBe('legacy-partial-tok')
    // s3c.servers 已重写为无 token 形态
    expect(memLocal.getItem('s3c.servers')).not.toContain('legacy-partial-tok')

    // 缺 id 的条目无法被 activeServerId / per-server token 寻址，保留只会变成幽灵 server（id:''）→ 丢弃。
    memLocal.setItem('s3c.servers', JSON.stringify([{ token: 'orphan-tok' }]))
    const orphan = api.listServers()
    expect(orphan.map((s) => s.id)).not.toContain('')
    expect(memLocal.getItem('s3c.servers')).not.toContain('orphan-tok')
  })
})

// ── api.ts 覆盖率补全：迁移轮询超时（防 Promise 永久悬挂的兜底） ──────────────
describe('api gaps: 迁移轮询超时', () => {
  it('EOF 后任务长期未完成 → 轮询超时 onError（不永久悬挂）', async () => {
    vi.useFakeTimers()
    try {
      stubFetch((input: RequestInfo | URL) => {
        if (String(input).includes('/events')) {
          return Promise.resolve({
            ok: true,
            status: 200,
            statusText: 'OK',
            headers: new Map<string, string>(),
            body: {
              getReader: () => ({
                read: () => Promise.resolve({ done: true, value: new Uint8Array(0) }),
                cancel: vi.fn(),
              }),
            },
          })
        }
        // 状态回读永远返回「未完成」→ 轮询到 deadline 后必须 onError
        return Promise.resolve(makeBlobResponse({ done: false, progress: {} }))
      })
      const { subscribeMigrateEvents } = await import('./api')
      const onProgress = vi.fn()
      const onError = vi.fn()
      subscribeMigrateEvents('job-1', onProgress, onError)
      await vi.advanceTimersByTimeAsync(31_000)
      expect(onError).toHaveBeenCalled()
    } finally {
      vi.useRealTimers()
    }
  })

  it('回读终态为 cancelled → 合成 status=cancelled 回调（取消不被误报为 done）', async () => {
    stubFetch((input: RequestInfo | URL) => {
      if (String(input).includes('/events')) {
        return Promise.resolve({
          ok: true,
          status: 200,
          statusText: 'OK',
          headers: new Map<string, string>(),
          body: {
            getReader: () => ({
              read: () => Promise.resolve({ done: true, value: new Uint8Array(0) }),
              cancel: vi.fn(),
            }),
          },
        })
      }
      return Promise.resolve(makeBlobResponse({ done: true, progress: { status: 'cancelled' } }))
    })
    const { subscribeMigrateEvents } = await import('./api')
    const onProgress = vi.fn()
    const onError = vi.fn()
    const abort = subscribeMigrateEvents('job-1', onProgress, onError)
    await vi.waitFor(() =>
      expect(onProgress.mock.calls.some(([p]) => p.status === 'cancelled')).toBe(true),
    )
    abort()
    expect(onError).not.toHaveBeenCalled()
  })

  it('轮询等待期间 abort → 循环静默退出（不再 onError）', async () => {
    stubFetch((input: RequestInfo | URL) => {
      if (String(input).includes('/events')) {
        return Promise.resolve({
          ok: true,
          status: 200,
          statusText: 'OK',
          headers: new Map<string, string>(),
          body: {
            getReader: () => ({
              read: () => Promise.resolve({ done: true, value: new Uint8Array(0) }),
              cancel: vi.fn(),
            }),
          },
        })
      }
      return Promise.resolve(makeBlobResponse({ done: false, progress: {} }))
    })
    const { subscribeMigrateEvents } = await import('./api')
    const onError = vi.fn()
    const abort = subscribeMigrateEvents('job-1', vi.fn(), onError)
    // 完成至少一轮回读后中止，使循环在下一次迭代顶部检测到 aborted 而 break
    await new Promise((r) => setTimeout(r, 700))
    abort()
    await new Promise((r) => setTimeout(r, 600))
    expect(onError).not.toHaveBeenCalled()
  })
})
