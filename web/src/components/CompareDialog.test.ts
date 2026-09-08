import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import CompareDialog from './CompareDialog.vue'
import { s3api } from '../api'
import type { CompareVersion } from './CompareDialog.vue'

vi.mock('../api', () => ({
  s3api: {
    presign: vi.fn(async () => ({ url: 'https://cdn.test/v', method: 'GET' })),
  },
  api: { base: 'http://localhost:8080' },
}))

vi.mock('../i18n', () => ({
  t: (k: string) => k,
  tf: (k: string) => k,
  locale: () => 'en-US',
}))

const VERSIONS: CompareVersion[] = [
  { versionId: 'v2', size: 10, etag: 'e2', lastModified: '2024-01-02', storageClass: 'STANDARD', isLatest: true },
  { versionId: 'v1', size: 12, etag: 'e1', lastModified: '2024-01-01', storageClass: 'GLACIER', isLatest: false },
]

function mountDialog(versions: CompareVersion[] = VERSIONS) {
  return track(mount(CompareDialog, {
    props: {
      open: false,
      accountId: 'acc-1',
      bucket: 'b',
      objectKey: 'k',
      versions,
    },
    attachTo: document.body,
  }))
}

async function openDialog(w: ReturnType<typeof mountDialog>) {
  await w.setProps({ open: true })
  await flushPromises()
}

function bodyBtn(text: string): HTMLButtonElement {
  const b = Array.from(document.body.querySelectorAll('button')).find(
    (x) => (x.textContent ?? '').trim() === text,
  )
  expect(b, `body button "${text}"`).toBeTruthy()
  return b as unknown as HTMLButtonElement
}

function clickBody(text: string) {
  bodyBtn(text).dispatchEvent(new MouseEvent('click', { bubbles: true }))
}

function selectEl(): HTMLSelectElement {
  const el = document.body.querySelector('select') as HTMLSelectElement | null
  expect(el).toBeTruthy()
  return el!
}

const fetchMock = vi.fn()
vi.stubGlobal('fetch', fetchMock)

let mounted: Array<{ unmount: () => void }> = []
function track<T extends { unmount: () => void }>(w: T): T {
  mounted.push(w)
  return w
}

afterEach(() => {
  for (const m of mounted) m.unmount()
  mounted = []
  document.body.innerHTML = ''
})
beforeEach(() => {
  vi.mocked(s3api.presign).mockResolvedValue({
      method: 'get',
      bucket: 'b',
      key: 'k',
      url: 'https://cdn.test/v',
      expiresIn: 900,
    })
})

afterEach(() => {
  vi.clearAllMocks()
  vi.restoreAllMocks()
  fetchMock.mockReset()
})

describe('CompareDialog', () => {
  it('does nothing with fewer than two versions', async () => {
    const w = mountDialog([VERSIONS[0]])
    await openDialog(w)
    expect(vi.mocked(s3api.presign)).not.toHaveBeenCalled()
    expect(fetchMock).not.toHaveBeenCalled()
    expect(document.body.textContent).toContain('compare.hint')
  })

  it('auto-runs compare on open: defaults latest vs previous, renders diff lines', async () => {
    vi.mocked(s3api.presign).mockImplementation(async (_id, body) => ({
      url: `https://cdn.test/v?versionId=${body.versionId}`,
      method: 'get',
      bucket: 'b',
      key: 'k',
      expiresIn: 900,
    }))
    fetchMock.mockImplementation(async (url: string) => ({
      ok: true,
      text: async () => (url.includes('versionId=v2') ? 'a\nB\nc' : 'a\nb\nc'),
    }))
    const w = mountDialog()
    await openDialog(w)
    // 第一个下拉是 base（默认前一个 v1 → idx1），第二个是 target（最新 v2 → idx0）
    expect(selectEl().selectedIndex).toBe(1)
    expect(document.body.querySelectorAll('select')[1].selectedIndex).toBe(0)
    expect(vi.mocked(s3api.presign).mock.calls.map((c) => c[1])).toEqual([
      { method: 'get', bucket: 'b', key: 'k', versionId: 'v1', expiresIn: 900 },
      { method: 'get', bucket: 'b', key: 'k', versionId: 'v2', expiresIn: 900 },
    ])
    const text = document.body.textContent ?? ''
    expect(text).toContain('compare.versionId')
    expect(text).toContain('v1')
    expect(text).toContain('v2')
    expect(document.body.querySelector('.diff-removed')?.textContent).toBe('b')
    expect(document.body.querySelector('.diff-added')?.textContent).toBe('B')
  })

  it('re-runs comparison via button', async () => {
    fetchMock.mockImplementation(async () => ({ ok: true, text: async () => 'same' }))
    const w = mountDialog()
    await openDialog(w)
    const before = vi.mocked(s3api.presign).mock.calls.length
    clickBody('compare.run')
    await flushPromises()
    expect(vi.mocked(s3api.presign).mock.calls.length).toBe(before + 2)
    expect(fetchMock).toHaveBeenCalledTimes(4)
  })

  it('base/target selects update v-model indexes', async () => {
    fetchMock.mockImplementation(async () => ({ ok: true, text: async () => 'same' }))
    const w = mountDialog()
    await openDialog(w)
    const selects = document.body.querySelectorAll('select')
    const sel = (i: number, v: string) => {
      ;(selects[i] as HTMLSelectElement).value = v
      ;(selects[i] as HTMLSelectElement).dispatchEvent(new Event('change'))
    }
    sel(0, '0') // base → v2
    sel(1, '1') // target → v1
    await flushPromises()
    expect((w.vm as unknown as { baseIdx: number }).baseIdx).toBe(0)
    expect((w.vm as unknown as { targetIdx: number }).targetIdx).toBe(1)
    expect((selects[0] as HTMLSelectElement).selectedIndex).toBe(0)
    expect((selects[1] as HTMLSelectElement).selectedIndex).toBe(1)
  })

  it('skips content download for too-large versions', async () => {
    const w = mountDialog([
      { versionId: 'big', size: 3 * 1024 * 1024, etag: 'e', lastModified: '2024-01-01', storageClass: 'STANDARD', isLatest: true },
      { versionId: 'v1', size: 12, etag: 'e1', lastModified: '2024-01-01', storageClass: 'STANDARD', isLatest: false },
    ])
    await openDialog(w)
    expect(fetchMock).not.toHaveBeenCalled()
    expect(document.body.textContent).toContain('compare.tooLarge')
  })

  it('marks binary content without diffing', async () => {
    fetchMock.mockImplementation(async () => ({ ok: true, text: async () => 'a\u0000b' }))
    const w = mountDialog()
    await openDialog(w)
    expect(document.body.textContent).toContain('compare.binary')
    expect(document.body.querySelector('.diff')).toBeNull()
  })

  it('shows error when presign fails', async () => {
    vi.mocked(s3api.presign).mockRejectedValue(new Error('presign-boom'))
    const w = mountDialog()
    await openDialog(w)
    expect(document.body.textContent).toContain('compare.error')
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('never fetches when base and target are the same version', async () => {
    const v = VERSIONS[0]
    const w = mountDialog([{ ...v }, { ...v }])
    await openDialog(w)
    expect(vi.mocked(s3api.presign)).not.toHaveBeenCalled()
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('downloads base and target versions through proxy anchors', async () => {
    fetchMock.mockImplementation(async () => ({ ok: true, text: async () => 'same' }))
    const hrefs: string[] = []
    vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (this: HTMLAnchorElement) {
      hrefs.push(this.href)
    })
    const w = mountDialog()
    await openDialog(w)
    clickBody('compare.dlBase')
    clickBody('compare.dlTarget')
    expect(hrefs).toHaveLength(2)
    expect(hrefs[0]).toContain('mode=download')
    expect(hrefs[0]).toContain('versionId=v1')
    expect(hrefs[0]).toContain('/api/accounts/acc-1/proxy?')
    expect(hrefs[1]).toContain('versionId=v2')
  })

  it('dialog X button emits close', async () => {
    const w = mountDialog()
    await openDialog(w)
    const x = document.body.querySelector('button.dlg-x') as HTMLButtonElement
    expect(x).toBeTruthy()
    x.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    expect(w.emitted('close')).toBeTruthy()
  })

  it('ignores a stale runCompare response when a newer run exists', async () => {
    fetchMock.mockImplementation(async () => ({ ok: true, text: async () => 'same\ntext' }))
    const pend: Array<(v: Awaited<ReturnType<typeof s3api.presign>>) => void> = []
    vi.mocked(s3api.presign).mockImplementation(
      () => new Promise((resolve) => { pend.push(resolve) }),
    )
    const w = mountDialog()
    await w.setProps({ open: true })
    await flushPromises()
    expect(pend).toHaveLength(2)
    // 直接再触发一次比较（模拟并发）；此时有 4 个挂起的 presign
    const run2 = (w.vm as unknown as { runCompare: () => Promise<void> }).runCompare()
    expect(pend).toHaveLength(4)

    const pkg = (u: string): Awaited<ReturnType<typeof s3api.presign>> => ({
      method: 'get',
      bucket: 'b',
      key: 'k',
      url: u,
      expiresIn: 900,
    })
    // 先完成的旧请求应被忽略（compareCtrl 已指向新的一次）
    pend[0](pkg('https://cdn.test/old-1'))
    pend[1](pkg('https://cdn.test/old-2'))
    await flushPromises()
    expect(document.body.querySelector('.diff')).toBeNull()
    // 新请求完成 → 渲染差异
    pend[2](pkg('https://cdn.test/new-1'))
    pend[3](pkg('https://cdn.test/new-2'))
    await run2
    await flushPromises()
    expect(document.body.querySelector('.diff')).toBeTruthy()
  })

  it('stale responses and aborts are ignored', async () => {
    const pend: Array<(v: Awaited<ReturnType<typeof s3api.presign>>) => void> = []
    vi.mocked(s3api.presign).mockImplementation(
      () => new Promise((resolve) => { pend.push(resolve) }),
    )
    const w = mountDialog()
    await w.setProps({ open: true })
    await flushPromises()
    expect(pend).toHaveLength(2)

    // 关闭对话框 → 取消当前比较
    await w.setProps({ open: false })
    await flushPromises()
    const stale = (u: string): Awaited<ReturnType<typeof s3api.presign>> => ({
      method: 'get',
      bucket: 'b',
      key: 'k',
      url: u,
      expiresIn: 900,
    })
    pend[0](stale('https://cdn.test/stale-1'))
    pend[1](stale('https://cdn.test/stale-2'))
    await flushPromises()
    // 已中止的旧请求不应产生结果渲染
    expect(document.body.textContent ?? '').not.toContain('compare.error')

    // 重新打开触发新的比较
    await w.setProps({ open: true })
    await flushPromises()
    expect(vi.mocked(s3api.presign).mock.calls.length).toBeGreaterThanOrEqual(4)
  })

  it('内容下载返回非 ok → fetchVersionContent 抛错并展示 compare.error', async () => {
    fetchMock.mockImplementation(async () => ({
      ok: false,
      status: 500,
      statusText: 'boom',
      text: async () => '',
    }))
    const w = mountDialog()
    await openDialog(w)
    expect(document.body.textContent ?? '').toContain('compare.error')
  })

  it('versions 长度非常规时 baseIdx 走三元假分支（回退 0）', async () => {
    // NaN.length：< 2 比较为 false，> 1 也为 false → baseIdx = 0
    const w = mountDialog(NaN as unknown as CompareVersion[])
    await openDialog(w)
    expect((w.vm as unknown as { baseIdx: number }).baseIdx).toBe(0)
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('空 versions 渲染空表格（base/target 缺失的 ?? 0 / — 分支）', async () => {
    const w = mountDialog([])
    await openDialog(w)
    const text = document.body.textContent ?? ''
    expect(text).toContain('compare.versionId')
    // base/target 均 undefined → fmtSize(??0)、storageClass '—'、etag '—'
    expect(text).toContain('compare.hint')
    expect(text).toContain('—')
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('对象名以斜杠结尾时 pop 为空回退下载名 object', async () => {
    fetchMock.mockImplementation(async () => ({ ok: true, text: async () => 'same' }))
    const downloads: string[] = []
    vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (this: HTMLAnchorElement) {
      downloads.push(this.download)
    })
    const w = track(mount(CompareDialog, {
      props: {
        open: false,
        accountId: 'acc-1',
        bucket: 'b',
        objectKey: 'dir/',
        versions: VERSIONS,
      },
      attachTo: document.body,
    }))
    await openDialog(w)
    clickBody('compare.dlBase')
    // '' → || 'object' 兜底
    expect(downloads).toContain('object')
  })

  it('差异中的空行渲染占位符 ␣', async () => {
    vi.mocked(s3api.presign).mockImplementation(async (_id, body) => ({
      url: `https://cdn.test/v?versionId=${body.versionId}`,
      method: 'get',
      bucket: 'b',
      key: 'k',
      expiresIn: 900,
    }))
    fetchMock.mockImplementation(async (url: string) => ({
      ok: true,
      text: async () => (url.includes('versionId=v2') ? 'a\n\nb' : 'a\nb'),
    }))
    const w = mountDialog()
    await openDialog(w)
    // lineDiff 产生 added 空行 → line.text 为 '' → 渲染 ␣
    expect(document.body.querySelector('.diff-added')?.textContent).toBe('␣')
  })
})
