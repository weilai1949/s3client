import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { downloadProxyObject, fetchProxy, fetchProxyBlob, proxyUrl } from './proxy'

const fetchMock = vi.fn()
vi.stubGlobal('fetch', fetchMock)

/** 已捕获的附件保存：每次 downloadProxyObject 触发的锚点点击。 */
interface SavedFile { href: string; download: string }
let saved: SavedFile[] = []

beforeEach(() => {
  fetchMock.mockReset()
  saved = []
  vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (this: HTMLAnchorElement) {
    saved.push({ href: this.href, download: this.download })
  })
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe('proxyUrl', () => {
  it('constructs proxy URL with required params', () => {
    const url = proxyUrl('acc1', 'mybucket', 'download', 'key/to/file', 'https://api.example.com')
    expect(url).toBe('https://api.example.com/api/accounts/acc1/proxy?bucket=mybucket&key=key%2Fto%2Ffile&mode=download')
  })

  it('includes versionId when provided', () => {
    const url = proxyUrl('acc1', 'mybucket', 'inline', 'key', 'https://api.example.com', 'v123')
    expect(url).toContain('versionId=v123')
  })

  it('excludes versionId when not provided', () => {
    const url = proxyUrl('acc1', 'mybucket', 'text', 'key', 'https://api.example.com')
    expect(url).not.toContain('versionId')
  })

  it('supports all modes', () => {
    expect(proxyUrl('a', 'b', 'download', 'k', 'https://api.example.com')).toContain('mode=download')
    expect(proxyUrl('a', 'b', 'inline', 'k', 'https://api.example.com')).toContain('mode=inline')
    expect(proxyUrl('a', 'b', 'text', 'k', 'https://api.example.com')).toContain('mode=text')
  })
})

const baseReq = {
  accountId: 'acc1',
  bucket: 'mybucket',
  key: 'dir/a.txt',
  apiBase: 'https://api.example.com',
}

describe('fetchProxy', () => {
  it('带 Bearer 头请求代理 URL，2xx 原样返回 Response', async () => {
    const resp = { ok: true, status: 200, statusText: 'OK' }
    fetchMock.mockResolvedValueOnce(resp)
    const got = await fetchProxy({ ...baseReq, mode: 'inline', token: 'tok123' })
    expect(got).toBe(resp)
    expect(fetchMock).toHaveBeenCalledWith(
      'https://api.example.com/api/accounts/acc1/proxy?bucket=mybucket&key=dir%2Fa.txt&mode=inline',
      expect.objectContaining({ headers: { Authorization: 'Bearer tok123' } }),
    )
  })

  it('token 为空时不携带 Authorization 头，并透传 signal/versionId', async () => {
    fetchMock.mockResolvedValueOnce({ ok: true })
    const ctrl = new AbortController()
    await fetchProxy({ ...baseReq, mode: 'text', token: '', versionId: 'v9', signal: ctrl.signal })
    const [, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit]
    expect(init.headers).toEqual({})
    expect(init.signal).toBe(ctrl.signal)
    expect(fetchMock.mock.calls[0][0]).toContain('versionId=v9')
  })

  it('非 2xx 抛错（错误响应体不当作内容返回）', async () => {
    fetchMock.mockResolvedValueOnce({ ok: false, status: 401, statusText: 'Unauthorized' })
    await expect(fetchProxy({ ...baseReq, mode: 'download', token: 'x' })).rejects.toThrow('401 Unauthorized')
  })
})

describe('fetchProxyBlob', () => {
  it('返回响应的 Blob（媒体预览用）', async () => {
    const blob = new Blob(['bytes'])
    fetchMock.mockResolvedValueOnce({ ok: true, blob: async () => blob })
    await expect(fetchProxyBlob({ ...baseReq, mode: 'inline', token: 't' })).resolves.toBe(blob)
  })
})

describe('downloadProxyObject', () => {
  it('成功：带 Bearer 取回字节后以附件名保存，并释放 objectURL', async () => {
    const revoked: string[] = []
    vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:mock-url')
    vi.spyOn(URL, 'revokeObjectURL').mockImplementation((u: string) => { revoked.push(u) })
    fetchMock.mockResolvedValueOnce({ ok: true, blob: async () => new Blob(['x']) })

    await downloadProxyObject({ ...baseReq, token: 'tok' }, 'a.txt')

    expect(fetchMock.mock.calls[0][0]).toContain('mode=download')
    expect((fetchMock.mock.calls[0][1] as RequestInit).headers).toEqual({ Authorization: 'Bearer tok' })
    expect(saved).toEqual([{ href: 'blob:mock-url', download: 'a.txt' }])
    expect(revoked).toEqual(['blob:mock-url'])
  })

  it('失败（如 401）：抛错且不保存任何文件', async () => {
    const createSpy = vi.spyOn(URL, 'createObjectURL')
    fetchMock.mockResolvedValueOnce({ ok: false, status: 401, statusText: 'Unauthorized' })
    await expect(downloadProxyObject({ ...baseReq, token: 'x' }, 'a.txt')).rejects.toThrow('401 Unauthorized')
    expect(createSpy).not.toHaveBeenCalled()
    expect(saved).toEqual([])
  })
})
