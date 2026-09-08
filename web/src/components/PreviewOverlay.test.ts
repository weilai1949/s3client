import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import PreviewOverlay from './PreviewOverlay.vue'
import type { PreviewState } from './PreviewOverlay.vue'
import { api } from '../api'
import { proxyUrl } from '../proxy'

vi.mock('../api', () => ({ api: { base: 'http://localhost', token: '' } }))
vi.mock('../proxy', () => ({ proxyUrl: vi.fn(() => 'http://proxy/url') }))
vi.mock('../i18n', () => ({
  t: (k: string) => k,
  tf: (k: string) => k,
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

function preview(overrides: Partial<PreviewState> = {}): PreviewState {
  return { key: 'a.txt', kind: 'text', url: 'http://proxy/url', ...overrides }
}

let mounted: ReturnType<typeof mount> | undefined
afterEach(() => {
  mounted?.unmount()
  mounted = undefined
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

  it('renders image with proxy download link, close on button/backdrop/img error', async () => {
    const w = await open(preview({ key: 'pic.png', kind: 'image' }))
    const img = w.find('img')
    expect(img.exists()).toBe(true)
    expect(img.attributes('src')).toBe('http://proxy/url')
    expect(img.attributes('alt')).toBe('pic.png')
    const link = w.find('.pv-actions a')
    expect(link.attributes('href')).toBe('http://proxy/url')
    expect(link.text()).toBe('common.download')

    // 点击卡片内部不关闭；点击背景（self）关闭
    await w.find('.pv-card').trigger('click')
    expect(w.emitted('close')).toBeUndefined()
    await w.find('.pv-backdrop').trigger('click')
    expect(w.emitted('close')).toHaveLength(1)
    // close 按钮
    const closeBtn = w.findAll('.pv-actions button').find((b) => b.text() === 'common.close')
    await closeBtn!.trigger('click')
    expect(w.emitted('close')).toHaveLength(2)
    // img error → close
    await img.trigger('error')
    expect(w.emitted('close')).toHaveLength(3)
  })

  it('renders video/audio/pdf/unknown kinds', async () => {
    const w = await open(preview({ key: 'v.mp4', kind: 'video' }))
    expect(w.find('video').attributes('src')).toBe('http://proxy/url')
    await w.setProps({ preview: preview({ key: 'a.mp3', kind: 'audio' }) })
    expect(w.find('audio').exists()).toBe(true)
    await w.setProps({ preview: preview({ key: 'd.pdf', kind: 'pdf' }) })
    const iframe = w.find('iframe')
    expect(iframe.exists()).toBe(true)
    expect(iframe.attributes('sandbox')).toBe('')
    await w.setProps({ preview: preview({ key: 'x.xyz', kind: 'none' }) })
    expect(w.find('.pv-body .empty').text()).toContain('preview.unsupported')
  })

  it('fetches text content and shows truncated badge', async () => {
    fetchMock.mockResolvedValueOnce(resp('hello world', '1'))
    const w = await open(preview({ key: 'a.txt', kind: 'text' }))
    await flushPromises()
    expect(fetchMock).toHaveBeenCalledTimes(1)
    const args = fetchMock.mock.calls[0] as unknown as [string, RequestInit]
    expect(args[0]).toBe('http://proxy/url') // proxyUrl mock 固定值
    expect(proxyUrl).toHaveBeenCalledWith('acc-1', 'b1', 'text', 'a.txt', 'http://localhost')
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
    // 第一次 fetch 挂起；切到 image 后 abort；再切回文本
    let rejectFetch!: (e: unknown) => void
    fetchMock.mockReturnValueOnce(new Promise((_res, rej) => (rejectFetch = rej)))
    const w = await open(preview({ key: 'a.txt', kind: 'text' }))
    await w.setProps({ preview: preview({ key: 'pic.png', kind: 'image' }) })
    await nextTick()
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
    // 此时切到新预览，旧 text 稍后返回 → 第 76 行 `if (fetchCtrl !== ctrl) return` 拦截后续处理
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
    // 旧 text 迟到：行 75 先写回文本，但行 76 的守卫拦截后续处理（truncated 不写入）
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
})
