import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'

vi.mock('./api', () => ({
  s3api: {
    presign: vi.fn(),
    multipartInit: vi.fn(),
    multipartPart: vi.fn(),
    multipartComplete: vi.fn(),
    multipartAbort: vi.fn(),
  },
  directUpload: vi.fn(),
}))

import { MULTIPART_THRESHOLD, PART_RETRY_DELAYS_MS, calcMultipartParts, withRetries, uploadObject } from './upload'
import { s3api, directUpload } from './api'

// Auto-fire mock XHR: send() completes immediately (default 'load'), tests can
// pre-configure the next instance via nextXHRConfig (status/error/etag).
type XhrAutofire = 'load' | 'error' | 'abort' | 'none'
type XhrHandler = () => void
interface MockProgressEvent { lengthComputable: boolean; loaded: number; total: number }
interface MockXhr {
  open: (...args: unknown[]) => unknown
  send: (...args: unknown[]) => unknown
  setRequestHeader: (...args: unknown[]) => unknown
  upload: { onprogress: ((e: MockProgressEvent) => void) | null }
  status: number
  getResponseHeader: (name: string) => string | null
  onload: XhrHandler | null
  onerror: XhrHandler | null
  onabort: XhrHandler | null
  abort: (...args: unknown[]) => unknown
  _autofire?: XhrAutofire
}
const XHR_INSTANCES: MockXhr[] = []
let nextXHRConfig: (inst: MockXhr) => void = () => {}
function createMockXHR() {
  const handlers: Record<'onload' | 'onerror' | 'onabort', XhrHandler | null> = {
    onload: null,
    onerror: null,
    onabort: null,
  }
  const inst: MockXhr = {
    open: vi.fn(),
    send: vi.fn(() => {
      inst.upload.onprogress?.({ lengthComputable: true, loaded: 1, total: 2 })
      const map: Record<XhrAutofire, 'onload' | 'onerror' | 'onabort' | null> = {
        load: 'onload',
        error: 'onerror',
        abort: 'onabort',
        none: null,
      }
      const key = map[inst._autofire ?? 'load']
      if (key) handlers[key]?.()
    }),
    setRequestHeader: vi.fn(),
    upload: { onprogress: null },
    status: 200,
    getResponseHeader: vi.fn(() => 'etag-1'),
    get onload() { return handlers.onload },
    set onload(fn) { handlers.onload = fn },
    get onerror() { return handlers.onerror },
    set onerror(fn) { handlers.onerror = fn },
    get onabort() { return handlers.onabort },
    set onabort(fn) { handlers.onabort = fn },
    abort: vi.fn(() => { handlers.onabort?.() }),
  }
  nextXHRConfig(inst)
  XHR_INSTANCES.push(inst)
  return inst
}


/** 100MB+ 代理文件（避免真实分配大内存）。 */
function largeFile(): File {
  const real = new File(['x'], 'big.bin', { type: 'application/octet-stream' })
  return new Proxy(real, {
    get(target, prop) {
      if (prop === 'size') return MULTIPART_THRESHOLD
      return Reflect.get(target, prop)
    },
  }) as File
}

/** 100MB+ 且无 content-type 的代理文件（覆盖 file.type || undefined 的右分支）。 */
function largeFileNoType(): File {
  const real = new File(['x'], 'big.bin')
  return new Proxy(real, {
    get(target, prop) {
      if (prop === 'size') return MULTIPART_THRESHOLD
      return Reflect.get(target, prop)
    },
  }) as File
}

function mockMultipartParts() {
  vi.mocked(s3api.multipartInit).mockResolvedValue({ uploadId: 'up1', key: 'k.bin', bucket: 'b' })
  vi.mocked(s3api.multipartPart).mockResolvedValue({ partNumber: 1, url: 'https://part.url', expiresIn: 3600 })
  vi.mocked(s3api.multipartComplete).mockResolvedValue({ completed: 'ok' })
  vi.mocked(s3api.multipartAbort).mockResolvedValue(undefined as unknown as Awaited<ReturnType<typeof s3api.multipartAbort>>)
}

describe('upload multipart helpers', () => {
  it('calcMultipartParts rounds up', () => {
    expect(calcMultipartParts(0)).toBe(0)
    expect(calcMultipartParts(1)).toBe(1)
    expect(calcMultipartParts(10 * 1024 * 1024)).toBe(1)
    expect(calcMultipartParts(10 * 1024 * 1024 + 1)).toBe(2)
  })
})

describe('withRetries', () => {
  it('retries on failure then succeeds', async () => {
    let n = 0
    const result = await withRetries(
      async () => {
        n++
        if (n < 3) throw new Error('transient')
        return 'ok'
      },
      [1, 1, 1],
    )
    expect(result).toBe('ok')
    expect(n).toBe(3)
  })

  it('does not retry AbortError', async () => {
    let n = 0
    await expect(
      withRetries(async () => {
        n++
        throw new DOMException('Aborted', 'AbortError')
      }, [1, 1]),
    ).rejects.toMatchObject({ name: 'AbortError' })
    expect(n).toBe(1)
  })

  it('retries DOMException with a non-AbortError name like any other error', async () => {
    let n = 0
    await expect(
      withRetries(async () => {
        n++
        throw new DOMException('boom', 'OtherError')
      }, [1, 1]),
    ).rejects.toMatchObject({ name: 'OtherError' })
    expect(n).toBe(3)
  })

  it('exhausts retries and throws last error', async () => {
    let n = 0
    await expect(
      withRetries(async () => {
        n++
        throw new Error(`fail-${n}`)
      }, [1, 1]),
    ).rejects.toThrow('fail-3')
    expect(n).toBe(3)
  })

  it('respects abort signal between attempts', async () => {
    const ctrl = new AbortController()
    let n = 0
    const p = withRetries(
      async () => {
        n++
        if (n === 1) {
          ctrl.abort()
          throw new Error('first')
        }
        return 'never'
      },
      [5, 5],
      ctrl.signal,
    )
    await expect(p).rejects.toMatchObject({ name: 'AbortError' })
    expect(n).toBe(1)
  })

  it('aborts during sleep delay', async () => {
    const ctrl = new AbortController()
    let n = 0
    const p = withRetries(
      async () => {
        n++
        throw new Error(`fail-${n}`)
      },
      [100],
      ctrl.signal,
    )
    setTimeout(() => ctrl.abort(), 10)
    await expect(p).rejects.toMatchObject({ name: 'AbortError' })
    expect(n).toBe(1)
  })

  it('uses the default delays when delaysMs omitted', async () => {
    const fn = vi.fn(async () => 'ok-default')
    await expect(withRetries(fn)).resolves.toBe('ok-default')
    expect(fn).toHaveBeenCalledTimes(1)
  })
})

describe('uploadObject', () => {
  beforeEach(() => {
    XHR_INSTANCES.length = 0
    nextXHRConfig = () => {}
    vi.mocked(s3api.multipartAbort).mockResolvedValue(undefined as unknown as Awaited<ReturnType<typeof s3api.multipartAbort>>)
    vi.stubGlobal('XMLHttpRequest', vi.fn(createMockXHR))
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    vi.useRealTimers()
  })

  it('small file uses presign + directUpload', async () => {
    const file = new File(['small'], 'small.txt', { type: 'text/plain' })
    vi.mocked(s3api.presign).mockResolvedValue({
      method: 'put',
      bucket: 'mybucket',
      key: 'small.txt',
      url: 'https://presigned.url',
      expiresIn: 3600,
    })
    vi.mocked(directUpload).mockResolvedValue(undefined)
    await uploadObject(file, { accId: 'acc1', key: 'small.txt' })
    expect(s3api.presign).toHaveBeenCalled()
    expect(directUpload).toHaveBeenCalled()
  })

  it('large file uses multipart upload and completes after part PUT', async () => {
    mockMultipartParts()
    const onProgress = vi.fn()
    await uploadObject(largeFile(), { accId: 'acc1', bucket: 'b', key: 'k.bin' }, onProgress)
    expect(s3api.multipartComplete).toHaveBeenCalledWith('acc1', expect.objectContaining({ uploadId: 'up1' }))
    expect(s3api.multipartAbort).not.toHaveBeenCalled()
    expect(onProgress).toHaveBeenCalled()
    expect(XHR_INSTANCES.length).toBeGreaterThan(0)
  })

  it('part PUT 2xx 但缺 ETag → partNoEtag 报错并中止会话（Bucket CORS 未暴露 ETag）', async () => {
    vi.useFakeTimers()
    mockMultipartParts()
    nextXHRConfig = (inst) => { inst.getResponseHeader = vi.fn(() => null) }
    const p = uploadObject(largeFile(), { accId: 'acc1', bucket: 'b', key: 'k.bin' })
    const rejection = expect(p).rejects.toThrow(/未读取到 ETag/)
    await vi.advanceTimersByTimeAsync(PART_RETRY_DELAYS_MS[0])
    await vi.advanceTimersByTimeAsync(PART_RETRY_DELAYS_MS[1])
    await vi.advanceTimersByTimeAsync(PART_RETRY_DELAYS_MS[2])
    await rejection
    await vi.waitFor(() => expect(s3api.multipartAbort).toHaveBeenCalled())
  })

  it('part PUT non-2xx → partHttpError after retries, abort session', async () => {
    vi.useFakeTimers()
    vi.mocked(s3api.multipartInit).mockResolvedValue({ uploadId: 'up1', key: 'k.bin', bucket: 'b' })
    vi.mocked(s3api.multipartPart).mockResolvedValue({ partNumber: 1, url: 'https://part.url', expiresIn: 3600 })
    nextXHRConfig = (inst) => { inst.status = 500 }
    const p = uploadObject(largeFile(), { accId: 'acc1', bucket: 'b', key: 'k.bin' })
    const rejection = expect(p).rejects.toThrow(/分段上传失败/) // 先挂 handler 再推进
    await vi.advanceTimersByTimeAsync(PART_RETRY_DELAYS_MS[0])
    await vi.advanceTimersByTimeAsync(PART_RETRY_DELAYS_MS[1])
    await vi.advanceTimersByTimeAsync(PART_RETRY_DELAYS_MS[2])
    await rejection
    await vi.waitFor(() => expect(s3api.multipartAbort).toHaveBeenCalled())
  })

  it('part PUT network error → partNetworkError after retries', async () => {
    vi.useFakeTimers()
    vi.mocked(s3api.multipartInit).mockResolvedValue({ uploadId: 'up1', key: 'k.bin', bucket: 'b' })
    vi.mocked(s3api.multipartPart).mockResolvedValue({ partNumber: 1, url: 'https://part.url', expiresIn: 3600 })
    nextXHRConfig = (inst) => { inst._autofire = 'error' }
    const p = uploadObject(largeFile(), { accId: 'acc1', bucket: 'b', key: 'k.bin' })
    const rejection = expect(p).rejects.toThrow(/分段上传网络错误/)
    await vi.advanceTimersByTimeAsync(PART_RETRY_DELAYS_MS[0])
    await vi.advanceTimersByTimeAsync(PART_RETRY_DELAYS_MS[1])
    await vi.advanceTimersByTimeAsync(PART_RETRY_DELAYS_MS[2])
    await rejection
  })

  it('part PUT abort → AbortError, session aborted', async () => {
    vi.mocked(s3api.multipartInit).mockResolvedValue({ uploadId: 'up1', key: 'k.bin', bucket: 'b' })
    vi.mocked(s3api.multipartPart).mockResolvedValue({ partNumber: 1, url: 'https://part.url', expiresIn: 3600 })
    nextXHRConfig = (inst) => { inst._autofire = 'abort' }
    await expect(uploadObject(largeFile(), { accId: 'acc1', bucket: 'b', key: 'k.bin' }))
      .rejects.toMatchObject({ name: 'AbortError' })
    expect(s3api.multipartAbort).toHaveBeenCalled()
  })

  it('signal aborted during part PUT → xhr.abort() + AbortError', async () => {
    const ctrl = new AbortController()
    nextXHRConfig = (inst) => { inst._autofire = 'none' } // send 后保持 pending，不发完成回调
    vi.mocked(s3api.multipartInit).mockResolvedValue({ uploadId: 'up1', key: 'k.bin', bucket: 'b' })
    vi.mocked(s3api.multipartPart).mockResolvedValue({ partNumber: 1, url: 'https://part.url', expiresIn: 3600 })
    const p = uploadObject(largeFile(), { accId: 'acc1', bucket: 'b', key: 'k.bin' }, undefined, ctrl.signal)
    // 10MB 分段 + 4 并发 worker：每个 worker 的首个分段 PUT 挂起（无 autofire），
    // 因此恰好 4 个 XHR 创建并已 send（send 前监听器已注册）。
    await vi.waitFor(() => expect(XHR_INSTANCES.length).toBe(4))
    for (const xhr of XHR_INSTANCES) expect(xhr.send).toHaveBeenCalled()
    ctrl.abort() // 触发 putPartReturnEtag 的 signal 监听（onAbort → xhr.abort + reject）
    await expect(p).rejects.toMatchObject({ name: 'AbortError' })
    for (const xhr of XHR_INSTANCES) expect(xhr.abort).toHaveBeenCalled()
    await vi.waitFor(() => expect(s3api.multipartAbort).toHaveBeenCalled())
  })

  it('signal aborted after multipart init aborts session', async () => {
    const ctrl = new AbortController()
    vi.mocked(s3api.multipartInit).mockResolvedValue({ uploadId: 'up1', key: 'k.bin', bucket: 'b' })
    const p = uploadObject(largeFile(), { accId: 'acc1', bucket: 'b', key: 'k.bin' }, undefined, ctrl.signal)
    await vi.waitFor(() => expect(s3api.multipartInit).toHaveBeenCalled())
    ctrl.abort()
    await expect(p).rejects.toMatchObject({ name: 'AbortError' })
    await vi.waitFor(() => expect(s3api.multipartAbort).toHaveBeenCalled())
  })

  it('signal aborted right after init → immediate abort branch', async () => {
    const ctrl = new AbortController()
    vi.mocked(s3api.multipartInit).mockResolvedValue({ uploadId: 'up1', key: 'k.bin', bucket: 'b' })
    const p = uploadObject(largeFile(), { accId: 'acc1', bucket: 'b', key: 'k.bin' }, undefined, ctrl.signal)
    ctrl.abort() // line 162 的 signal.aborted 分支：abort 后直接抛
    await expect(p).rejects.toMatchObject({ name: 'AbortError' })
    await vi.waitFor(() => expect(s3api.multipartAbort).toHaveBeenCalled())
  })

  it('part response without ETag → partNoEtag error', async () => {
    vi.useFakeTimers()
    mockMultipartParts()
    nextXHRConfig = (inst) => { inst.getResponseHeader = vi.fn(() => '') }
    const p = uploadObject(largeFile(), { accId: 'acc1', bucket: 'b', key: 'k.bin' })
    const rejection = expect(p).rejects.toThrow(/分段上传完成但未读取到 ETag/)
    await vi.advanceTimersByTimeAsync(PART_RETRY_DELAYS_MS[0])
    await vi.advanceTimersByTimeAsync(PART_RETRY_DELAYS_MS[1])
    await vi.advanceTimersByTimeAsync(PART_RETRY_DELAYS_MS[2])
    await rejection
    await vi.waitFor(() => expect(s3api.multipartAbort).toHaveBeenCalled())
  })

  it('aborts when signal is already aborted', async () => {
    const ctrl = new AbortController()
    ctrl.abort()
    const file = new File(['small'], 'small.txt', { type: 'text/plain' })
    await expect(uploadObject(file, { accId: 'acc1', key: 'small.txt' }, undefined, ctrl.signal))
      .rejects.toMatchObject({ name: 'AbortError' })
  })

  it('large file without content-type → multipartInit receives contentType undefined', async () => {
    mockMultipartParts()
    await uploadObject(largeFileNoType(), { accId: 'acc1', bucket: 'b', key: 'k.bin' })
    expect(s3api.multipartInit).toHaveBeenCalledWith('acc1', expect.objectContaining({ contentType: undefined }))
  })

  it('part 响应缺 ETag 头（null）→ ?? 空串 → partNoEtag 错误并中止会话', async () => {
    vi.useFakeTimers()
    mockMultipartParts()
    nextXHRConfig = (inst) => { inst.getResponseHeader = vi.fn(() => null) }
    const p = uploadObject(largeFile(), { accId: 'acc1', bucket: 'b', key: 'k.bin' })
    const rejection = expect(p).rejects.toThrow(/分段上传完成但未读取到 ETag/)
    await vi.advanceTimersByTimeAsync(PART_RETRY_DELAYS_MS[0])
    await vi.advanceTimersByTimeAsync(PART_RETRY_DELAYS_MS[1])
    await vi.advanceTimersByTimeAsync(PART_RETRY_DELAYS_MS[2])
    await rejection
    await vi.waitFor(() => expect(s3api.multipartAbort).toHaveBeenCalled())
  })

  it('分段进度事件 lengthComputable=false 被忽略，不回调 onProgress', async () => {
    mockMultipartParts()
    const onProgress = vi.fn()
    await uploadObject(largeFile(), { accId: 'acc1', bucket: 'b', key: 'k.bin' }, onProgress)
    const before = onProgress.mock.calls.length
    XHR_INSTANCES[0].upload.onprogress?.({ lengthComputable: false, loaded: 1, total: 2 })
    expect(onProgress.mock.calls.length).toBe(before)
  })

  it('signal aborted right after init 且 abort 请求失败 → catch 回调吞掉', async () => {
    const ctrl = new AbortController()
    vi.mocked(s3api.multipartInit).mockResolvedValue({ uploadId: 'up1', key: 'k.bin', bucket: 'b' })
    vi.mocked(s3api.multipartAbort).mockRejectedValue(new Error('abort failed'))
    const p = uploadObject(largeFile(), { accId: 'acc1', bucket: 'b', key: 'k.bin' }, undefined, ctrl.signal)
    ctrl.abort()
    await expect(p).rejects.toMatchObject({ name: 'AbortError' })
    await vi.waitFor(() => expect(s3api.multipartAbort).toHaveBeenCalled())
  })

  it('signal 中止引发 abort() 且 abort 请求失败 → catch 回调吞掉', async () => {
    const ctrl = new AbortController()
    nextXHRConfig = (inst) => { inst._autofire = 'none' }
    vi.mocked(s3api.multipartInit).mockResolvedValue({ uploadId: 'up1', key: 'k.bin', bucket: 'b' })
    vi.mocked(s3api.multipartPart).mockResolvedValue({ partNumber: 1, url: 'https://part.url', expiresIn: 3600 })
    vi.mocked(s3api.multipartAbort).mockRejectedValue(new Error('abort failed'))
    const p = uploadObject(largeFile(), { accId: 'acc1', bucket: 'b', key: 'k.bin' }, undefined, ctrl.signal)
    await vi.waitFor(() => expect(XHR_INSTANCES.length).toBe(4))
    ctrl.abort()
    await expect(p).rejects.toMatchObject({ name: 'AbortError' })
    await vi.waitFor(() => expect(s3api.multipartAbort).toHaveBeenCalled())
  })
})

describe('upload final branches', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    XHR_INSTANCES.length = 0
    nextXHRConfig = () => {}
    vi.mocked(s3api.multipartAbort).mockResolvedValue(undefined as unknown as Awaited<ReturnType<typeof s3api.multipartAbort>>)
    vi.stubGlobal('XMLHttpRequest', vi.fn(createMockXHR))
  })
  afterEach(() => {
    vi.unstubAllGlobals()
    vi.useRealTimers()
  })

  it('withRetries with pre-aborted signal throws immediately', async () => {
    const ctrl = new AbortController()
    ctrl.abort()
    const fn = vi.fn(async () => 'never')
    await expect(withRetries(fn, [1, 1], ctrl.signal)).rejects.toMatchObject({ name: 'AbortError' })
    expect(fn).not.toHaveBeenCalled()
  })

  it('uploadObject small file: abort between presign and directUpload', async () => {
    const ctrl = new AbortController()
    vi.mocked(s3api.presign).mockResolvedValue({ method: 'put', bucket: 'b', key: 'k.txt', url: 'https://p', expiresIn: 3600 })
    vi.mocked(directUpload).mockResolvedValue(undefined)
    const file = new File(['x'], 'k.txt')
    const p = uploadObject(file, { accId: 'acc1', bucket: 'b', key: 'k.txt' }, undefined, ctrl.signal)
    ctrl.abort() // presign 期间被中止 → line 97 检查抛 AbortError
    await expect(p).rejects.toMatchObject({ name: 'AbortError' })
    expect(directUpload).not.toHaveBeenCalled()
  })

  // abort() 幂等守卫（170 行 `if (aborted) return`）：见下方「最后一组分段完成前后」两用例。
  it('首段 onload 时 signal 中止：下一轮 worker 迭代的守卫抛 AbortError', async () => {
    mockMultipartParts()
    const ctrl = new AbortController()
    let count = 0
    nextXHRConfig = (inst) => {
      count++
      if (count === 1) {
        // 第一个分段 onload 内中止 signal：worker A 完成第 1 段后进入下一轮迭代时命中 180 行
        inst.getResponseHeader = vi.fn(() => {
          ctrl.abort()
          return 'etag-1'
        })
      }
    }
    const p = uploadObject(largeFile(), { accId: 'acc1', bucket: 'b', key: 'k.bin' }, undefined, ctrl.signal)
    await expect(p).rejects.toMatchObject({ name: 'AbortError' })
    expect(s3api.multipartComplete).not.toHaveBeenCalled()
  })

  it('全部分段完成后 signal 才中止：末尾守卫抛 AbortError 且不 complete', async () => {
    mockMultipartParts()
    const ctrl = new AbortController()
    let count = 0
    nextXHRConfig = (inst) => {
      count++
      if (count === 10) {
        // 最后一个分段的 onload 内中止：其余 worker 已耗尽（idx==totalParts），不会先抛 180/193 守卫
        inst.getResponseHeader = vi.fn(() => {
          ctrl.abort()
          return 'etag-10'
        })
      }
    }
    const p = uploadObject(largeFile(), { accId: 'acc1', bucket: 'b', key: 'k.bin' }, undefined, ctrl.signal)
    await expect(p).rejects.toMatchObject({ name: 'AbortError' })
    expect(s3api.multipartComplete).not.toHaveBeenCalled()
    await vi.waitFor(() => expect(s3api.multipartAbort).toHaveBeenCalledTimes(1))
  })

  it('abort 幂等守卫：catch 对非 AbortError 二次调用 abort() 时直接 return', async () => {
    mockMultipartParts()
    const ctrl = new AbortController()
    let count = 0
    nextXHRConfig = (inst) => {
      count++
      if (count === 10) {
        // 最后一个分段：onload 内先中止（listener → abort() 第一次），返回空 ETag →
        // partNoEtag（非 AbortError）→ catch 第二次调用 abort() → 170 行 `if (aborted) return`
        inst.getResponseHeader = vi.fn(() => {
          ctrl.abort()
          return ''
        })
      }
    }
    const p = uploadObject(largeFile(), { accId: 'acc1', bucket: 'b', key: 'k.bin' }, undefined, ctrl.signal)
    await expect(p).rejects.toThrow(/未读取到 ETag/)
    expect(s3api.multipartComplete).not.toHaveBeenCalled()
    // 第二次 abort() 被幂等守卫拦截：multipartAbort 只被调用一次
    await vi.waitFor(() => expect(s3api.multipartAbort).toHaveBeenCalledTimes(1))
  })
})
