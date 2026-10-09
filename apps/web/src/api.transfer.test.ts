// api.transfer.test.ts —— 自 api.test.ts 拆出（KNOWN_ISSUES #60）：迁移事件 SSE、直传与边界分支。
// 断言逐字搬移，与拆分前等价；其余范围见 api.test.ts（基础/请求/s3api）与 api.gaps.test.ts。
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

  it('EOF 未收到终态且回读失败：应 onError（不得静默悬挂）', async () => {
    // /events 立即 EOF；状态回读请求持续失败（网络抖动）
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
      // 状态回读失败
      return Promise.reject(new Error('network down'))
    })
    const { subscribeMigrateEvents } = await import('./api')
    const onProgress = vi.fn()
    const onError = vi.fn()
    const abort = subscribeMigrateEvents('job1', onProgress, onError)
    // 等待回读重试（3 次 × 500ms 间隔）耗尽
    await new Promise((r) => setTimeout(r, 2500))
    abort()
    // 回读重试耗尽后必须 onError，让调用方 Promise reject（opsBusy 复位），而非永久悬挂
    expect(onError).toHaveBeenCalled()
  })

  it('EOF 未收到终态且任务仍在运行：轮询直到 done 后回调终态（不得悬挂）', async () => {
    let statusCalls = 0
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
      // 状态回读：前两次返回「仍在运行」，第三次返回完成
      statusCalls++
      if (statusCalls < 3) {
        return Promise.resolve({
          ok: true,
          status: 200,
          statusText: 'OK',
          json: () => Promise.resolve({ done: false, progress: { done: 3, total: 10, migrated: 3 } }),
          blob: () => Promise.resolve(new Blob(['ok'])),
          headers: new Map<string, string>(),
          body: null,
        })
      }
      return Promise.resolve({
        ok: true,
        status: 200,
        statusText: 'OK',
        json: () => Promise.resolve({ done: true, progress: { done: 10, total: 10, migrated: 10, failed: 0 } }),
        blob: () => Promise.resolve(new Blob(['ok'])),
        headers: new Map<string, string>(),
        body: null,
      })
    })
    const { subscribeMigrateEvents } = await import('./api')
    const onProgress = vi.fn()
    const onError = vi.fn()
    const abort = subscribeMigrateEvents('job1', onProgress, onError)
    // 等待 3 次状态回读（前两次 running、第三次 done；间隔 500ms）
    await new Promise((r) => setTimeout(r, 2500))
    abort()
    // 必须收到终态（status='done'），调用方 Promise 才能 resolve
    const terminal = onProgress.mock.calls.find(([p]) => p.status === 'done')
    expect(terminal).toBeTruthy()
    expect(onError).not.toHaveBeenCalled()
  })

  it('SSE 静默挂起（连接不断但无任何事件）：空闲超时触发 onError，不永久占用按钮', async () => {
    vi.useFakeTimers()
    try {
      let cancelled = false
      stubFetch(() =>
        Promise.resolve({
          ok: true,
          status: 200,
          statusText: 'OK',
          headers: new Map<string, string>(),
          body: {
            getReader: () => ({
              read: () => new Promise(() => {}), // 永久挂起
              cancel: () => {
                cancelled = true
                return Promise.resolve()
              },
            }),
          },
        }),
      )
      const { subscribeMigrateEvents } = await import('./api')
      const onProgress = vi.fn()
      const onError = vi.fn()
      subscribeMigrateEvents('job1', onProgress, onError)
      await vi.advanceTimersByTimeAsync(46_000)
      expect(onError).toHaveBeenCalled()
      expect(onProgress).not.toHaveBeenCalled()
      expect(cancelled).toBe(true) // 超时必须主动断开流，不留悬挂连接
    } finally {
      vi.useRealTimers()
    }
  })

  it('SSE 空闲超时会主动关闭底层流（不留悬挂连接）', async () => {
    vi.useFakeTimers()
    try {
      const cancel = vi.fn(() => Promise.resolve())
      stubFetch(() =>
        Promise.resolve({
          ok: true,
          status: 200,
          statusText: 'OK',
          headers: new Map<string, string>(),
          body: {
            getReader: () => ({ read: () => new Promise(() => {}), cancel }), // 永久挂起
          },
        }),
      )
      const { subscribeMigrateEvents } = await import('./api')
      const onError = vi.fn()
      subscribeMigrateEvents('job1', vi.fn(), onError)
      await vi.advanceTimersByTimeAsync(46_000)
      expect(onError).toHaveBeenCalled()
      expect(cancel).toHaveBeenCalled()
    } finally {
      vi.useRealTimers()
    }
  })

  it('SSE 心跳持续到达时不误判空闲；终态到达后不触发空闲超时', async () => {
    let cancelled = false
    let reads = 0
    stubFetch(() =>
      Promise.resolve({
        ok: true,
        status: 200,
        statusText: 'OK',
        headers: new Map<string, string>(),
        body: {
          getReader: () => ({
            read: async () => {
              reads++
              if (reads === 1) return { done: false, value: new TextEncoder().encode('event: ping\ndata: {}\n\n') }
              if (reads === 2) {
                // 两次心跳间隔 40ms < 60ms 空闲阈值 → 不得超时
                await new Promise((r) => setTimeout(r, 40))
                return { done: false, value: new TextEncoder().encode('event: ping\ndata: {}\n\n') }
              }
              await new Promise((r) => setTimeout(r, 40))
              return {
                done: false,
                value: new TextEncoder().encode('event: progress\ndata: {"done":1,"total":1,"migrated":1,"status":"done"}\n\n'),
              }
            },
            cancel: () => {
              cancelled = true
              return Promise.resolve()
            },
          }),
        },
      }),
    )
    const { subscribeMigrateEvents } = await import('./api')
    const onProgress = vi.fn()
    const onError = vi.fn()
    const abort = subscribeMigrateEvents('job1', onProgress, onError, 60)
    await vi.waitFor(() => expect(onProgress.mock.calls.some(([p]) => p.status === 'done')).toBe(true))
    abort()
    expect(onError).not.toHaveBeenCalled()
    expect(cancelled).toBe(false)
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

  it('条件写 headers 逐个 setRequestHeader（条件头参与签名，漏带即签名不匹配）', async () => {
    const XHRMock = createXhrMockClass()
    vi.stubGlobal('XMLHttpRequest', XHRMock)
    const { directUpload } = await import('./api')
    const promise = directUpload(
      'https://x',
      new Blob(['a']) as File,
      undefined,
      undefined,
      { 'If-None-Match': '*', 'X-Amz-SignedHeaders': 'host;if-none-match' },
    )
    const inst = XHRMock.getInstances()[0]
    expect(inst.setRequestHeader).toHaveBeenCalledWith('If-None-Match', '*')
    expect(inst.setRequestHeader).toHaveBeenCalledWith('X-Amz-SignedHeaders', 'host;if-none-match')
    inst._fire('load', {})
    await promise
  })

  it('412 条件不满足 → 返回可读的条件写失败提示（而非裸状态码）', async () => {
    const XHRMock = createXhrMockClass()
    vi.stubGlobal('XMLHttpRequest', XHRMock)
    const { directUpload } = await import('./api')
    const promise = directUpload('https://x', new Blob(['a']) as File, undefined, undefined, { 'If-None-Match': '*' })
    const inst = XHRMock.getInstances()[0]
    inst.status = 412
    inst._fire('load', {})
    await expect(promise).rejects.toThrow('upload.conditionalFailed')
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
    const { api } = await loadApi()
    const { downloadZipToDisk } = await import('./api/download')
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
    const { api } = await loadApi()
    const { downloadZipToDisk } = await import('./api/download')
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
    const { api } = await loadApi()
    const { downloadZipToDisk } = await import('./api/download')
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
    const { api } = await loadApi()
    const { downloadZipToDisk } = await import('./api/download')
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
