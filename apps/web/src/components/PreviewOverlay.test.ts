import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import PreviewOverlay from './PreviewOverlay.vue'
import type { PreviewState } from './PreviewOverlay.vue'
import { api } from '../api'
import { toast } from '../store'

vi.mock('../api', () => ({ api: { base: 'http://localhost', token: '' } }))
vi.mock('../store', () => ({ toast: vi.fn() }))
vi.mock('../i18n', () => ({
  t: (k: string) => k,
  // 带 msg 的插值保留原因文本（断言「错误不静默」需要看到 401 等详情）
  tf: (k: string, params?: Record<string, unknown>) =>
    params && 'msg' in params ? `${k}:${String(params.msg)}` : k,
}))

const fetchMock = vi.fn()
vi.stubGlobal('fetch', fetchMock)

function resp(text: string, truncated = ''): Partial<Response> {
  return {
    ok: true,
    status: 200,
    statusText: 'OK',
    text: async () => text,
    headers: { get: (h: string) => (h === 'X-Preview-Truncated' ? truncated : null) } as unknown as Headers,
  }
}

/** 媒体/PDF 取回的字节响应（fetchProxyBlob 调 res.blob()）。 */
function blobResp(): Partial<Response> {
  return { ok: true, status: 200, statusText: 'OK', blob: async () => new Blob(['media']) }
}

function preview(overrides: Partial<PreviewState> = {}): PreviewState {
  return { key: 'a.txt', kind: 'text', ...overrides }
}

let mounted: ReturnType<typeof mount> | undefined
afterEach(() => {
  mounted?.unmount()
  mounted = undefined
  vi.restoreAllMocks()
})

function mountOverlay() {
  mounted = mount(PreviewOverlay, {
    props: { preview: null, accountId: 'acc-1', bucket: 'b1' },
    attachTo: document.body,
    global: { stubs: { Teleport: true } },
  })
  return mounted
}

async function open(p: PreviewState) {
  const w = mountOverlay()
  await w.setProps({ preview: p })
  await nextTick()
  return w
}

/** 找到动作区按钮（下载 / 关闭）。 */
function actionBtn(w: ReturnType<typeof mountOverlay>, text: string) {
  const b = w.findAll('.pv-actions button').find((x) => x.text() === text)
  expect(b, `action button ${text}`).toBeTruthy()
  return b!
}

beforeEach(() => {
  vi.clearAllMocks()
  fetchMock.mockReset()
  api.token = ''
})

describe('PreviewOverlay', () => {
  it('renders nothing when preview is null', () => {
    const w = mountOverlay()
    expect(w.find('.pv-backdrop').exists()).toBe(false)
    expect(w.text()).toBe('')
  })

  it('image：带 Bearer 取回 inline 字节后以 objectURL 渲染；关闭/背景/img 错误仍生效', async () => {
    api.token = 'tok123'
    vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:img')
    vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => {})
    let resolveFetch!: (r: Partial<Response>) => void
    fetchMock.mockReturnValueOnce(new Promise<Partial<Response>>((res) => (resolveFetch = res)))
    const w = await open(preview({ key: 'pic.png', kind: 'image' }))
    await nextTick()
    // 取回中：媒体占位（`<img src>` 直连无法带 Authorization，故先取字节）
    expect(w.find('.pv-body .empty').text()).toContain('preview.loadingText')

    resolveFetch(blobResp())
    await flushPromises()

    const [url, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit]
    expect(url).toContain('/api/accounts/acc-1/proxy?')
    expect(url).toContain('mode=inline')
    expect(url).toContain('key=pic.png')
    expect(init.headers).toEqual({ Authorization: 'Bearer tok123' })
    const img = w.find('img')
    expect(img.exists()).toBe(true)
    expect(img.attributes('src')).toBe('blob:img')
    expect(img.attributes('alt')).toBe('pic.png')

    // 下载与关闭按钮
    actionBtn(w, 'common.download')
    // 点击卡片内部不关闭；点击背景（self）关闭
    await w.find('.pv-card').trigger('click')
    expect(w.emitted('close')).toBeUndefined()
    await w.find('.pv-backdrop').trigger('click')
    expect(w.emitted('close')).toHaveLength(1)
    await actionBtn(w, 'common.close').trigger('click')
    expect(w.emitted('close')).toHaveLength(2)
    // img error → close
    await img.trigger('error')
    expect(w.emitted('close')).toHaveLength(3)
  })

  it('media 取回失败：显示失败占位，不渲染媒体元素（C1 错误不静默）', async () => {
    fetchMock.mockResolvedValueOnce({ ok: false, status: 401, statusText: 'Unauthorized' })
    const w = await open(preview({ key: 'pic.png', kind: 'image' }))
    await flushPromises()
    expect(w.find('img').exists()).toBe(false)
    expect(w.find('.pv-body').text()).toContain('preview.fail')
    expect(w.find('.pv-body').text()).toContain('401 Unauthorized')
  })

  it('renders video/audio/pdf via objectURL, unknown kind renders placeholder without fetching', async () => {
    vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:m')
    fetchMock.mockResolvedValue(blobResp())
    const w = await open(preview({ key: 'v.mp4', kind: 'video' }))
    await flushPromises()
    expect(w.find('video').attributes('src')).toBe('blob:m')
    await w.setProps({ preview: preview({ key: 'a.mp3', kind: 'audio' }) })
    await flushPromises()
    expect(w.find('audio').exists()).toBe(true)
    expect(w.find('audio').attributes('src')).toBe('blob:m')
    await w.setProps({ preview: preview({ key: 'd.pdf', kind: 'pdf' }) })
    await flushPromises()
    const iframe = w.find('iframe')
    expect(iframe.exists()).toBe(true)
    expect(iframe.attributes('sandbox')).toBe('')
    expect(iframe.attributes('src')).toBe('blob:m')
    await w.setProps({ preview: preview({ key: 'x.xyz', kind: 'none' }) })
    await flushPromises()
    expect(w.find('.pv-body .empty').text()).toContain('preview.unsupported')
  })

  it('下载按钮成功：经带 Bearer 的代理取回后以附件名保存并释放 objectURL', async () => {
    vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:dl')
    const revoked: string[] = []
    vi.spyOn(URL, 'revokeObjectURL').mockImplementation((u: string) => { revoked.push(u) })
    const saved: string[] = []
    vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (this: HTMLAnchorElement) {
      saved.push(this.download)
    })
    fetchMock.mockResolvedValueOnce({ ok: true, blob: async () => new Blob(['d']) })

    const w = await open(preview({ key: 'dir/a.txt', kind: 'none' }))
    await flushPromises()
    expect(fetchMock).not.toHaveBeenCalled() // 未知类型不预取媒体
    api.token = 'tok123'
    await actionBtn(w, 'common.download').trigger('click')
    await flushPromises()

    const [url, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit]
    expect(url).toContain('mode=download')
    expect(url).toContain('key=dir%2Fa.txt')
    expect(init.headers).toEqual({ Authorization: 'Bearer tok123' })
    expect(saved).toEqual(['a.txt'])
    expect(revoked).toEqual(['blob:dl'])
  })

  it('下载失败（如 401）：toast 错误提示，不保存任何文件', async () => {
    vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:dl')
    const saved: string[] = []
    vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (this: HTMLAnchorElement) {
      saved.push(this.download)
    })
    fetchMock.mockResolvedValueOnce({ ok: false, status: 401, statusText: 'Unauthorized' })

    const w = await open(preview({ key: 'a.txt', kind: 'none' }))
    await flushPromises()
    await actionBtn(w, 'common.download').trigger('click')
    await flushPromises()

    expect(saved).toEqual([])
    expect(URL.createObjectURL).not.toHaveBeenCalled()
    expect(vi.mocked(toast)).toHaveBeenCalledWith('preview.downloadFail:401 Unauthorized', 'err')
  })

  it('切换/关闭/卸载预览时释放已创建的 objectURL（媒体 blob 不泄漏）', async () => {
    let n = 0
    vi.spyOn(URL, 'createObjectURL').mockImplementation(() => `blob:${++n}`)
    const revoked: string[] = []
    vi.spyOn(URL, 'revokeObjectURL').mockImplementation((u: string) => { revoked.push(u) })
    fetchMock.mockResolvedValue(blobResp())

    const w = await open(preview({ key: 'pic.png', kind: 'image' }))
    await flushPromises()
    expect(w.find('img').attributes('src')).toBe('blob:1')
    // 切到文本预览：旧 objectURL 释放
    await w.setProps({ preview: preview({ key: 'a.txt', kind: 'text' }) })
    await flushPromises()
    expect(revoked).toContain('blob:1')
    // 再开媒体并卸载：卸载时释放
    fetchMock.mockResolvedValue(blobResp())
    await w.setProps({ preview: preview({ key: 'pic2.png', kind: 'image' }) })
    await flushPromises()
    expect(w.find('img').attributes('src')).toBe('blob:2')
    w.unmount()
    mounted = undefined
    expect(revoked).toContain('blob:2')
  })

  it('fetches text content and shows truncated badge', async () => {
    fetchMock.mockResolvedValueOnce(resp('hello world', '1'))
    const w = await open(preview({ key: 'a.txt', kind: 'text' }))
    await flushPromises()
    expect(fetchMock).toHaveBeenCalledTimes(1)
    const args = fetchMock.mock.calls[0] as unknown as [string, RequestInit]
    expect(args[0]).toContain('mode=text') // 代理统一入口
    expect(args[0]).toContain('/api/accounts/acc-1/proxy?')
    expect(args[1].headers).toEqual({})
    expect(args[1].signal).toBeInstanceOf(AbortSignal)
    expect(w.find('.pv-pre').text()).toBe('hello world')
    expect(w.text()).toContain('preview.truncated')
  })

  it('sends bearer token when api.token is set', async () => {
    api.token = 'tok123'
    fetchMock.mockResolvedValueOnce(resp('x'))
    await open(preview({ key: 'a.txt', kind: 'text' }))
    await flushPromises()
    const args = fetchMock.mock.calls[0] as unknown as [string, RequestInit]
    expect(args[1].headers).toEqual({ Authorization: 'Bearer tok123' })
  })

  it('shows loading state while fetch pending, then renders fail text on HTTP error', async () => {
    let resolveFetch!: (r: Partial<Response>) => void
    fetchMock.mockReturnValueOnce(new Promise<Partial<Response>>((res) => (resolveFetch = res)))
    const w = await open(preview({ key: 'a.txt', kind: 'text' }))
    await nextTick()
    expect(w.find('.pv-text-wrap').text()).toContain('preview.loadingText')
    resolveFetch({ ok: false, status: 404, statusText: 'Not Found' })
    await flushPromises()
    expect(w.find('.pv-pre').text()).toContain('preview.fail')
  })

  it('ignores aborted/overridden fetches and keeps showing current preview', async () => {
    // 第一次（文本）fetch 挂起；切到 image 后 abort；image 自身的取回正常完成
    let rejectFetch!: (e: unknown) => void
    fetchMock.mockImplementationOnce(() => new Promise((_res, rej) => (rejectFetch = rej)))
    fetchMock.mockResolvedValueOnce(blobResp())
    vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:img')
    const w = await open(preview({ key: 'a.txt', kind: 'text' }))
    await w.setProps({ preview: preview({ key: 'pic.png', kind: 'image' }) })
    await flushPromises()
    rejectFetch(new DOMException('Aborted', 'AbortError'))
    await flushPromises()
    // 文本失败未写入：当前预览仍是图片
    expect(w.find('img').exists()).toBe(true)
    expect(w.text()).not.toContain('preview.fail')
  })

  it('drops stale text result when preview changed mid-fetch', async () => {
    let resolveText!: (t: string) => void
    fetchMock.mockReturnValueOnce(
      new Promise<Partial<Response>>((res) => {
        resolveText = (t) => res({ ok: true, text: async () => t, headers: { get: () => null } as unknown as Headers })
      }),
    )
    const w = await open(preview({ key: 'a.txt', kind: 'text' }))
    // 切换到一个新文本预览（新 fetch），旧 fetchCtrl 失效
    fetchMock.mockResolvedValueOnce(resp('new content'))
    await w.setProps({ preview: preview({ key: 'b.txt', kind: 'text' }) })
    resolveText('stale content')
    await flushPromises()
    expect(w.find('.pv-pre').text()).toBe('new content')
    expect(w.text()).not.toContain('stale content')
  })

  it('drops stale text read (post-fetch guard) when preview switched while res.text() pending', async () => {
    // 第一次 fetch 先完成（通过 fetchCtrl 与 ctrl 一致检查），但其 res.text() 挂起；
    // 此时切到新预览，旧 text 稍后返回 → 守卫拦截后续处理
    let resolveFetch!: (r: Partial<Response>) => void
    let resolveText!: (t: string) => void
    fetchMock.mockReturnValueOnce(
      new Promise<Partial<Response>>((res) => {
        resolveFetch = () => res({
          ok: true,
          // stale 响应标记 truncated=1：若守卫未拦截，truncated badge 会被写入
          headers: { get: (h: string) => (h === 'X-Preview-Truncated' ? '1' : null) } as unknown as Headers,
          text: () => new Promise<string>((r) => (resolveText = r)),
        })
      }),
    )
    const w = await open(preview({ key: 'a.txt', kind: 'text' }))
    resolveFetch({} as Partial<Response>) // fetch 完成 → 通过第一个 fetchCtrl 检查，进入 res.text() 等待
    await flushPromises()
    // 新预览启动第二段 fetch（立即完成），旧 fetchCtrl 失效
    fetchMock.mockResolvedValueOnce(resp('new content'))
    await w.setProps({ preview: preview({ key: 'b.txt', kind: 'text' }) })
    await flushPromises()
    expect(w.find('.pv-pre').text()).toBe('new content')
    // 旧 text 迟到：先写回文本，但守卫拦截后续处理（truncated 不写入）
    resolveText('stale content')
    await flushPromises()
    expect(w.text()).not.toContain('preview.truncated')
  })

  it('closes to null: clears text/loading and stops fetching', async () => {
    const w = await open(preview({ key: 'a.txt', kind: 'text' }))
    let rejectFetch!: (e: unknown) => void
    fetchMock.mockReturnValueOnce(new Promise((_res, rej) => (rejectFetch = rej)))
    await w.setProps({ preview: preview({ key: 'b.txt', kind: 'text' }) })
    await w.setProps({ preview: null })
    await nextTick()
    expect(w.find('.pv-backdrop').exists()).toBe(false)
    rejectFetch(new DOMException('Aborted', 'AbortError'))
    await flushPromises()
    expect(w.find('.pv-pre').exists()).toBe(false)
  })

  it('closes on Escape via keydown stack', async () => {
    const w = await open(preview({ key: 'a.txt', kind: 'image' }))
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await nextTick()
    expect(w.emitted('close')).toHaveLength(1)
  })

  it('Escape 在 preview 为空时不触发 close；其他按键一律忽略', async () => {
    const w = await open(preview({ key: 'a.txt', kind: 'image' }))
    // preview 为 null：Escape 短路（props.preview 为假）
    await w.setProps({ preview: null })
    await nextTick()
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await nextTick()
    expect(w.emitted('close')).toBeUndefined()
    // 有 preview 但按键非 Escape：e.key === 'Escape' 为假
    await w.setProps({ preview: preview({ key: 'b.txt', kind: 'text' }) })
    await nextTick()
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'z' }))
    await nextTick()
    expect(w.emitted('close')).toBeUndefined()
  })

  it('下载名以斜杠结尾：pop 为空回退 object 并释放 objectURL', async () => {
    vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:dl')
    const revoked: string[] = []
    vi.spyOn(URL, 'revokeObjectURL').mockImplementation((u: string) => { revoked.push(u) })
    const saved: string[] = []
    vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (this: HTMLAnchorElement) {
      saved.push(this.download)
    })
    fetchMock.mockResolvedValueOnce({ ok: true, blob: async () => new Blob(['d']) })

    const w = await open(preview({ key: 'dir/', kind: 'none' }))
    await flushPromises()
    expect(fetchMock).not.toHaveBeenCalled() // 未知类型不预取媒体
    await actionBtn(w, 'common.download').trigger('click')
    await flushPromises()

    // '' → || 'object' 兜底
    expect(saved).toEqual(['object'])
    expect(revoked).toEqual(['blob:dl'])
  })

  it('媒体取回在切换后才完成：post-await 守卫丢弃旧结果（不建 objectURL、不写错误）', async () => {
    const createSpy = vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:stale')
    let resolveA!: (r: Partial<Response>) => void
    fetchMock.mockReturnValueOnce(new Promise<Partial<Response>>((res) => (resolveA = res)))
    fetchMock.mockResolvedValueOnce(resp('hello'))
    const w = await open(preview({ key: 'pic.png', kind: 'image' }))
    await flushPromises()
    // 切到文本预览（新 fetchCtrl），旧媒体取回稍后才返回
    await w.setProps({ preview: preview({ key: 'a.txt', kind: 'text' }) })
    await flushPromises()
    expect(w.find('.pv-pre').text()).toBe('hello')
    resolveA(blobResp())
    await flushPromises()
    expect(createSpy).not.toHaveBeenCalled()
    expect(w.find('img').exists()).toBe(false)
    expect(w.text()).not.toContain('preview.fail')
    expect(w.find('.pv-pre').text()).toBe('hello')
  })

  it('媒体取回在切换后才失败：catch 守卫忽略迟到错误，不覆盖当前预览', async () => {
    vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:img')
    let rejectA!: (e: Error) => void
    fetchMock.mockReturnValueOnce(new Promise<Partial<Response>>((_res, rej) => { rejectA = rej }))
    fetchMock.mockResolvedValueOnce(blobResp())
    const w = await open(preview({ key: 'a.png', kind: 'image' }))
    await flushPromises()
    // 切到另一张图片（新的 fetchCtrl），旧取回稍后才 reject
    await w.setProps({ preview: preview({ key: 'b.png', kind: 'image' }) })
    await flushPromises()
    expect(w.find('img').attributes('src')).toBe('blob:img')
    rejectA(new Error('late failure'))
    await flushPromises()
    expect(w.find('img').attributes('src')).toBe('blob:img')
    expect(w.text()).not.toContain('preview.fail')
  })
})
