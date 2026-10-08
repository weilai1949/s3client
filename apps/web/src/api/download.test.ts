import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import {
  DOWNLOAD_PART_SIZE,
  PARALLEL_DOWNLOAD_MIN_BYTES,
  downloadObjectToDisk,
} from './download'

// 并行分段的最小可测尺寸：恰好 4 段（16 MiB / 4 MiB）。
const SIZE = PARALLEL_DOWNLOAD_MIN_BYTES

const baseReq = {
  accountId: 'acc1',
  bucket: 'b',
  key: 'dir/big.bin',
  apiBase: 'https://api.example.com',
  token: 'tok',
  filename: 'big.bin',
}

interface MockRes {
  ok: boolean
  status: number
  statusText: string
  headers: Map<string, string>
  body?: { pipeTo: (w: unknown) => Promise<void>; cancel: () => Promise<void> } | undefined
  blob: () => Promise<Blob>
}

function partBlob(start: number, end: number, marker: number): Blob {
  return new Blob([new Uint8Array(end - start + 1).fill(marker)])
}

/** 206 分段响应：内容首字节为段序号（用于断言顺序）。 */
function rangedResponse(range: string, contentRange?: string | null): MockRes {
  const m = /bytes=(\d+)-(\d+)/.exec(range)!
  const start = Number(m[1])
  const end = Number(m[2])
  const headers = new Map<string, string>()
  if (contentRange !== null) headers.set('Content-Range', contentRange ?? `bytes ${start}-${end}/${SIZE}`)
  return {
    ok: true,
    status: 206,
    statusText: 'Partial Content',
    headers,
    blob: async () => partBlob(start, end, Math.floor(start / DOWNLOAD_PART_SIZE) + 1),
  }
}

/** 全量（Range 被忽略）响应。 */
function fullResponse(body?: MockRes['body']): MockRes {
  return { ok: true, status: 200, statusText: 'OK', headers: new Map(), body, blob: async () => new Blob(['whole']) }
}

function errorResponse(status: number): MockRes {
  return { ok: false, status, statusText: 'Forbidden', headers: new Map(), blob: async () => new Blob() }
}

const fetchMock = vi.fn()

let captured: Blob | undefined
let clicked: string[] = []

beforeEach(() => {
  fetchMock.mockReset()
  captured = undefined
  clicked = []
  vi.stubGlobal('fetch', fetchMock)
  vi.spyOn(URL, 'createObjectURL').mockImplementation((b) => {
    captured = b as Blob
    return 'blob:test'
  })
  vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => {})
  vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (this: HTMLAnchorElement) {
    clicked.push(this.download)
  })
})

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
  delete (window as { showSaveFilePicker?: unknown }).showSaveFilePicker
})

function installSaveFilePicker(handle: { createWritable: () => Promise<unknown> }) {
  const picker = vi.fn().mockResolvedValue(handle)
  Object.defineProperty(window, 'showSaveFilePicker', { value: picker, configurable: true })
  return picker
}

describe('downloadObjectToDisk 单流路径', () => {
  it('大小未知：单流带 Bearer 与 versionId，经 objectURL 触发保存', async () => {
    fetchMock.mockResolvedValueOnce({ ...fullResponse(), blob: async () => new Blob(['x']) })
    await downloadObjectToDisk({ ...baseReq, versionId: 'v1' })
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit]
    expect(url).toContain('/api/accounts/acc1/proxy?')
    expect(url).toContain('versionId=v1')
    expect(url).toContain('mode=download')
    expect((init.headers as Record<string, string>).Authorization).toBe('Bearer tok')
    expect((init.headers as Record<string, string>).Range).toBeUndefined()
    expect(clicked).toEqual(['big.bin'])
  })

  it('小文件（< 阈值）走单流；无 token 时不带 Authorization', async () => {
    fetchMock.mockResolvedValueOnce({ ...fullResponse(), blob: async () => new Blob(['x']) })
    await downloadObjectToDisk({ ...baseReq, token: '', size: 1024 })
    const [, init] = fetchMock.mock.calls[0] as [string, RequestInit]
    expect(init.headers).toEqual({})
    expect(clicked).toEqual(['big.bin'])
  })

  it('单流 FSA 可用：流式落盘', async () => {
    const pipeTo = vi.fn().mockResolvedValue(undefined)
    const cancel = vi.fn().mockResolvedValue(undefined)
    fetchMock.mockResolvedValueOnce(fullResponse({ pipeTo, cancel }))
    installSaveFilePicker({ createWritable: async () => ({ abort: vi.fn() }) })
    await downloadObjectToDisk({ ...baseReq, size: 1024 })
    expect(pipeTo).toHaveBeenCalled()
    expect(clicked).toEqual([])
  })

  it('单流 FSA 选择器被拒 → cancel body 并抛错（不落盘）', async () => {
    const cancel = vi.fn().mockRejectedValue(new Error('cancel failed'))
    fetchMock.mockResolvedValueOnce(fullResponse({ pipeTo: vi.fn(), cancel }))
    Object.defineProperty(window, 'showSaveFilePicker', {
      value: vi.fn().mockRejectedValue(new Error('user cancelled')),
      configurable: true,
    })
    await expect(downloadObjectToDisk({ ...baseReq, size: 1024 })).rejects.toThrow('user cancelled')
    expect(cancel).toHaveBeenCalled()
    expect(clicked).toEqual([])
  })

  it('单流 FSA 存在但响应无 body → 退回 blob 保存', async () => {
    fetchMock.mockResolvedValueOnce({ ...fullResponse(), blob: async () => new Blob(['x']) })
    installSaveFilePicker({ createWritable: async () => ({ abort: vi.fn() }) })
    await downloadObjectToDisk({ ...baseReq, size: 1024 })
    expect(clicked).toEqual(['big.bin'])
  })

  it('单流非 2xx → 抛错且不落盘（错误体绝不当文件保存）', async () => {
    fetchMock.mockResolvedValueOnce(errorResponse(403))
    await expect(downloadObjectToDisk({ ...baseReq, size: 1024 })).rejects.toThrow('403 Forbidden')
    expect(clicked).toEqual([])
  })
})

describe('downloadObjectToDisk 并行 Range 路径', () => {
  it('有界并发 Range 分段并按段号顺序聚合（无 FSA 时 objectURL 保存）', async () => {
    const ranges: string[] = []
    fetchMock.mockImplementation(async (_url: string, init: RequestInit) => {
      const range = (init.headers as Record<string, string>).Range
      ranges.push(range)
      return rangedResponse(range)
    })
    await downloadObjectToDisk({ ...baseReq, size: SIZE })
    expect(ranges).toHaveLength(4)
    expect([...ranges].sort()).toEqual([
      `bytes=0-${DOWNLOAD_PART_SIZE - 1}`,
      `bytes=${DOWNLOAD_PART_SIZE}-${2 * DOWNLOAD_PART_SIZE - 1}`,
      `bytes=${2 * DOWNLOAD_PART_SIZE}-${3 * DOWNLOAD_PART_SIZE - 1}`,
      `bytes=${3 * DOWNLOAD_PART_SIZE}-${SIZE - 1}`,
    ].sort())
    const bytes = new Uint8Array(await captured!.arrayBuffer())
    expect(bytes.length).toBe(SIZE)
    expect(bytes[0]).toBe(1)
    expect(bytes[DOWNLOAD_PART_SIZE]).toBe(2)
    expect(bytes[2 * DOWNLOAD_PART_SIZE]).toBe(3)
    expect(bytes[3 * DOWNLOAD_PART_SIZE]).toBe(4)
    expect(clicked).toEqual(['big.bin'])
  })

  it('FSA 可用：逐段顺序写入并 close', async () => {
    const written: Blob[] = []
    const writable = {
      write: vi.fn(async (b: Blob) => { written.push(b) }),
      close: vi.fn().mockResolvedValue(undefined),
      abort: vi.fn().mockResolvedValue(undefined),
    }
    installSaveFilePicker({ createWritable: async () => writable })
    fetchMock.mockImplementation(async (_url: string, init: RequestInit) =>
      rangedResponse((init.headers as Record<string, string>).Range),
    )
    await downloadObjectToDisk({ ...baseReq, size: SIZE })
    expect(written).toHaveLength(4)
    expect(written[0]!.size).toBe(DOWNLOAD_PART_SIZE)
    const first = new Uint8Array(await written[0]!.arrayBuffer())
    const second = new Uint8Array(await written[1]!.arrayBuffer())
    expect(first[0]).toBe(1)
    expect(second[0]).toBe(2)
    expect(writable.close).toHaveBeenCalled()
  })

  it('FSA 写入失败 → abort 并抛错（不留下半截文件）', async () => {
    const abort = vi.fn().mockResolvedValue(undefined)
    const writable = {
      write: vi.fn().mockRejectedValue(new Error('disk full')),
      close: vi.fn(),
      abort,
    }
    installSaveFilePicker({ createWritable: async () => writable })
    fetchMock.mockImplementation(async (_url: string, init: RequestInit) =>
      rangedResponse((init.headers as Record<string, string>).Range),
    )
    await expect(downloadObjectToDisk({ ...baseReq, size: SIZE })).rejects.toThrow('disk full')
    expect(abort).toHaveBeenCalled()
  })

  it('服务端忽略 Range（200 全量）→ 取消并发并回退单流', async () => {
    const cancel = vi.fn().mockRejectedValue(new Error('cancel failed'))
    fetchMock.mockImplementation(async (_url: string, init: RequestInit) => {
      const headers = init.headers as Record<string, string>
      return headers.Range ? fullResponse({ pipeTo: vi.fn(), cancel }) : { ...fullResponse(), blob: async () => new Blob(['whole']) }
    })
    await downloadObjectToDisk({ ...baseReq, size: SIZE })
    expect(cancel).toHaveBeenCalled()
    expect(clicked).toEqual(['big.bin'])
  })

  it('服务端忽略 Range 且无 body → 仍回退单流', async () => {
    fetchMock.mockImplementation(async (_url: string, init: RequestInit) => {
      const headers = init.headers as Record<string, string>
      return headers.Range ? fullResponse() : { ...fullResponse(), blob: async () => new Blob(['whole']) }
    })
    await downloadObjectToDisk({ ...baseReq, size: SIZE })
    expect(clicked).toEqual(['big.bin'])
  })

  it('分段非 2xx → 抛错且不落盘', async () => {
    fetchMock.mockResolvedValue(errorResponse(500))
    await expect(downloadObjectToDisk({ ...baseReq, size: SIZE })).rejects.toThrow('500 Forbidden')
    expect(clicked).toEqual([])
  })

  it('分段长度与请求区间不符 → 抛错（长度校验）', async () => {
    fetchMock.mockImplementation(async (_url: string, init: RequestInit) => {
      const range = (init.headers as Record<string, string>).Range
      const r = rangedResponse(range)
      return { ...r, blob: async () => new Blob(['short']) }
    })
    await expect(downloadObjectToDisk({ ...baseReq, size: SIZE })).rejects.toThrow(/长度或范围/)
    expect(clicked).toEqual([])
  })

  it('Content-Range 与请求区间不符 → 抛错（顺序 / 范围校验）', async () => {
    fetchMock.mockImplementation(async (_url: string, init: RequestInit) =>
      rangedResponse((init.headers as Record<string, string>).Range, 'bytes 0-1/999'),
    )
    await expect(downloadObjectToDisk({ ...baseReq, size: SIZE })).rejects.toThrow(/长度或范围/)
    expect(clicked).toEqual([])
  })

  it('无 Content-Range 头时仅按长度校验，可正常落盘', async () => {
    fetchMock.mockImplementation(async (_url: string, init: RequestInit) =>
      rangedResponse((init.headers as Record<string, string>).Range, null),
    )
    await downloadObjectToDisk({ ...baseReq, size: SIZE })
    expect(clicked).toEqual(['big.bin'])
  })
})
