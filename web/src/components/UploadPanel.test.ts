import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { nextTick, ref } from 'vue'
import UploadPanel from './UploadPanel.vue'
import { s3api } from '../api'
import { currentAccount, requestTab, toast } from '../store'
import { copyText } from '../clipboard'
import { useUploadQueue } from '../composables/useUploadQueue'
import type { UploadQueueItem } from '../composables/useUploadQueue'
import type { Account } from '../types'

vi.mock('../api', () => ({
  s3api: { presign: vi.fn() },
}))
vi.mock('../store', () => ({
  currentAccount: vi.fn(),
  toast: vi.fn(),
  requestTab: vi.fn(),
}))
vi.mock('../clipboard', () => ({
  copyText: vi.fn(async () => {}),
}))
vi.mock('../composables/useUploadQueue', () => ({
  useUploadQueue: vi.fn(),
}))
vi.mock('../i18n', () => ({
  t: (k: string) => k,
  tf: (k: string) => k,
}))

const acc: Account = {
  id: 'acc-1',
  name: 'my-account',
  endpoint: 'http://minio:9000',
  region: 'us-east-1',
  accessKey: 'ak',
  secretKey: 'sk',
  bucket: 'b1',
  pathStyle: true,
  useSSL: false,
}

/** useUploadQueue mock：items/running 用可驱动的 ref，enqueue 模拟真实入队语义。 */
function makeQueue() {
  const items = ref<UploadQueueItem[]>([])
  const running = ref(false)
  return {
    items,
    running,
    enqueue: vi.fn((files: FileList | null, make?: (f: File) => Partial<UploadQueueItem>) => {
      if (!files || !files.length) return
      for (const f of Array.from(files)) {
        items.value.push({ id: items.value.length + 1, file: f, key: '', pct: 0, status: 'pending', ...make?.(f) })
      }
    }),
    run: vi.fn(async () => [] as UploadQueueItem[]),
    abortItem: vi.fn(),
    abortAll: vi.fn(),
  }
}

function item(overrides: Partial<UploadQueueItem> = {}): UploadQueueItem {
  return { id: 1, file: new File(['x'], 'a.txt'), key: 'a.txt', pct: 0, status: 'pending', ...overrides }
}

let mounted: ReturnType<typeof mount> | undefined
afterEach(() => {
  mounted?.unmount()
  mounted = undefined
})

function mountPanel(account: Account | null = acc) {
  const q = makeQueue()
  // 用 ref 承载账号：让 computed(currentAccount) 像真实 store 一样可失效
  const accState = ref<Account | undefined>(account ?? undefined)
  vi.mocked(currentAccount).mockImplementation(() => accState.value)
  vi.mocked(useUploadQueue).mockReturnValue(q as unknown as ReturnType<typeof useUploadQueue>)
  mounted = mount(UploadPanel)
  return { w: mounted, q, accState }
}

function findButton(w: ReturnType<typeof mount>, text: string) {
  const btn = w.findAll('button').find((b) => b.text() === text)
  expect(btn, `button "${text}" should exist`).toBeTruthy()
  return btn!
}

beforeEach(() => {
  vi.clearAllMocks()
})

describe('UploadPanel', () => {
  it('shows empty state without account', () => {
    const { w } = mountPanel(null)
    expect(w.find('.empty').text()).toContain('upload.needAccount')
    expect(w.find('.dropzone').exists()).toBe(false)
    // tf mock 回显 key，不拼接变量名
    expect(w.text()).toContain('upload.currentAccount')
  })

  it('binds prefix, adds files via change event with trimmed key prefix, resets input', async () => {
    const { w, q } = mountPanel()
    // 前缀尾部斜杠被 keyFor 清除
    await w.find('input[placeholder="upload.prefixPlaceholder"]').setValue('docs//')
    const input = w.find('input[type="file"]')
    const files = [new File(['a'], 'a.txt'), new File(['b'], 'b.bin')]
    Object.defineProperty(input.element, 'files', { value: files })
    await input.trigger('change')
    await nextTick()
    expect(q.enqueue).toHaveBeenCalledTimes(1)
    const make = q.enqueue.mock.calls[0][1] as (f: File) => Partial<UploadQueueItem>
    expect(make(files[0])).toEqual({ key: 'docs/a.txt' })
    expect(q.items.value).toHaveLength(2)
    expect(q.items.value[0].key).toBe('docs/a.txt')
    expect(q.items.value[0].status).toBe('pending')
  })

  it('dropzone: dragover/dragleave classes and drop enqueues files', async () => {
    const { w, q } = mountPanel()
    const dz = w.find('.dropzone')
    await dz.trigger('dragover')
    expect(w.find('.dropzone').classes()).toContain('over')
    await dz.trigger('dragleave')
    expect(w.find('.dropzone').classes()).not.toContain('over')
    const dt = { files: [new File(['x'], 'd.txt')] } as unknown as DataTransfer
    await dz.trigger('drop', { dataTransfer: dt })
    expect(q.enqueue).toHaveBeenCalledTimes(1)
    expect(w.find('.dropzone').classes()).not.toContain('over')
  })

  it('drop 无 dataTransfer：`e.dataTransfer?.files ?? null` 兜底为 null,以 null 入队（队列侧自行忽略）', async () => {
    const { w, q } = mountPanel()
    const dz = w.find('.dropzone')
    await dz.trigger('drop', {})
    expect(q.enqueue).toHaveBeenCalledWith(null, expect.any(Function))
  })

  it('addFiles 在 fileInput 未绑定时：`if (fileInput.value)` 守卫跳过重置', async () => {
    const { w } = mountPanel()
    const vm = w.vm as unknown as { fileInput: unknown; addFiles: (f: FileList | null) => void }
    vm.fileInput = null
    expect(() => vm.addFiles(null)).not.toThrow()
  })

  it('dropzone click opens the picker exactly once (no recursion), enter/space too', async () => {
    const { w } = mountPanel()
    // 隐藏 input 位于 dropzone 内：input 上的 @click.stop 阻止程序化 click 上抛回 dropzone，
    // 避免真实浏览器中的递归（issue: input.click() 冒泡 → dropzone @click → input.click() …）。
    const input = w.find('input[type="file"]')
    const clickSpy = vi.spyOn(input.element as unknown as { click: () => void }, 'click')
    const dz = w.find('.dropzone')
    await dz.trigger('click')
    expect(clickSpy).toHaveBeenCalledTimes(1) // 若无 @click.stop 这里会递归/无限调用
    await dz.trigger('keydown.enter')
    await dz.trigger('keydown.space')
    expect(clickSpy).toHaveBeenCalledTimes(3)
  })

  it('renders statuses/progress/rows and drives toolbar disabled state', async () => {
    const { w, q } = mountPanel()
    q.items.value = [
      item({ id: 1, key: 'p.txt', status: 'pending', pct: 0 }),
      item({ id: 2, key: 's.txt', status: 'signing', pct: 5 }),
      item({ id: 3, key: 'u.txt', status: 'uploading', pct: 50, file: new File(['x'], 'u.txt') }),
      item({ id: 4, key: 'd.txt', status: 'done', pct: 100 }),
      item({ id: 5, key: 'e.txt', status: 'err', pct: 10, err: 'boom' }),
    ]
    await nextTick()
    // 总进度 (0+5+50+100+10)/5 = 33
    const bar = w.find('.row .progress')
    expect(bar.attributes('role')).toBe('progressbar')
    expect(bar.attributes('aria-valuenow')).toBe('33')
    expect(w.text()).toContain('1/5 · 33%')
    const lis = w.findAll('.list-item')
    expect(lis[0].text()).toContain('upload.statusPending')
    expect(lis[1].text()).toContain('upload.statusSigning')
    expect(lis[2].text()).toContain('50%')
    expect(lis[3].text()).toContain('upload.statusDone')
    expect(lis[4].text()).toContain('upload.statusErr')
    expect(lis[4].text()).toContain('boom')
    // start/retry/clear 按钮可用性
    const start = findButton(w, 'upload.start')
    expect(start.attributes('disabled')).toBeUndefined()
    const retry = findButton(w, 'upload.retryFailed')
    expect(retry.attributes('disabled')).toBeUndefined()
    const clearDone = findButton(w, 'upload.clearDone')
    expect(clearDone.attributes('disabled')).toBeUndefined()
    // running 时全部禁用（除 start 文案仍在）
    q.running.value = true
    await nextTick()
    expect(findButton(w, 'upload.start').attributes('disabled')).toBeDefined()
    expect(w.findAll('.list-item')[0].find('button[aria-label="common.remove"]').exists()).toBe(false)
  })

  it('uploadAll toasts ok with action, toasts partial with errors', async () => {
    const { w, q } = mountPanel()
    q.items.value = [item({ id: 1, key: 'a.txt' }), item({ id: 2, key: 'b.txt' })]
    await nextTick()
    vi.mocked(q.run).mockResolvedValueOnce([
      { ...item({ id: 1, key: 'a.txt' }), status: 'done' },
      { ...item({ id: 2, key: 'b.txt' }), status: 'done' },
    ])
    await findButton(w, 'upload.start').trigger('click')
    await flushPromises()
    expect(q.run).toHaveBeenCalledTimes(1)
    expect(toast).toHaveBeenCalledTimes(1)
    const [text, kind, action] = vi.mocked(toast).mock.calls[0] as unknown as [
      string, string, { onClick: () => void },
    ]
    expect(text).toBe('upload.toastOk')
    expect(kind).toBe('ok')
    action.onClick()
    expect(requestTab).toHaveBeenCalledWith('objects')

    // 部分失败
    vi.mocked(q.run).mockResolvedValueOnce([
      { ...item({ id: 3, key: 'c.txt' }), status: 'done' },
      { ...item({ id: 4, key: 'd.txt' }), status: 'err', err: 'x' },
    ])
    await findButton(w, 'upload.start').trigger('click')
    await flushPromises()
    expect(toast).toHaveBeenLastCalledWith('upload.toastPartial', 'err', expect.anything())
  })

  it('uploadAll with empty processed batch does not toast', async () => {
    const { w, q } = mountPanel()
    q.items.value = [item({ id: 1, key: 'a.txt' })]
    vi.mocked(q.run).mockResolvedValueOnce([])
    await findButton(w, 'upload.start').trigger('click')
    await flushPromises()
    expect(toast).not.toHaveBeenCalled()
  })

  it('uploadAll guards against account disappearing mid-flight', async () => {
    const { w, q, accState } = mountPanel()
    q.items.value = [item({ id: 1, key: 'a.txt' })]
    await nextTick()
    // 按钮已渲染后账号被移除（computed 失效）→ uploadAll 直接返回，不跑队列
    accState.value = undefined
    await findButton(w, 'upload.start').trigger('click')
    await flushPromises()
    expect(q.run).not.toHaveBeenCalled()
    expect(toast).not.toHaveBeenCalled()
  })

  it('retryFailed resets err items to pending and reruns', async () => {
    const { w, q } = mountPanel()
    q.items.value = [
      item({ id: 1, key: 'a.txt', status: 'err', pct: 40, err: 'boom' }),
      item({ id: 2, key: 'b.txt', status: 'done', pct: 100 }),
    ]
    await nextTick()
    await findButton(w, 'upload.retryFailed').trigger('click')
    await flushPromises()
    const first = q.items.value[0]
    expect(first.status).toBe('pending')
    expect(first.pct).toBe(0)
    expect(first.err).toBeUndefined()
    expect(q.run).toHaveBeenCalledTimes(1)
  })

  it('clearDone removes only done items; clearList aborts all and empties', async () => {
    const { w, q } = mountPanel()
    q.items.value = [
      item({ id: 1, key: 'a.txt', status: 'done', pct: 100 }),
      item({ id: 2, key: 'b.txt', status: 'err', pct: 0, err: 'x' }),
      item({ id: 3, key: 'c.txt', status: 'pending', pct: 0 }),
    ]
    await nextTick()
    await findButton(w, 'upload.clearDone').trigger('click')
    await nextTick()
    expect(q.items.value.map((i) => i.key)).toEqual(['b.txt', 'c.txt'])
    await findButton(w, 'upload.clearAll').trigger('click')
    await nextTick()
    expect(q.abortAll).toHaveBeenCalledTimes(1)
    expect(q.items.value).toHaveLength(0)
  })

  it('removeItem aborts and splices the item when not running', async () => {
    const { w, q } = mountPanel()
    const pending = item({ id: 1, key: 'a.txt', status: 'pending' })
    q.items.value = [pending]
    await nextTick()
    const rm = w.find('.list-item button[aria-label="common.remove"]')
    expect(rm.exists()).toBe(true)
    await rm.trigger('click')
    await nextTick()
    expect(q.abortItem).toHaveBeenCalledWith(pending)
    expect(q.items.value).toHaveLength(0)
  })

  it('copyDoneLink presigns, copies the url and toasts; surfaces presign errors', async () => {
    const { w, q } = mountPanel()
    const done = item({ id: 1, key: 'a.txt', status: 'done', pct: 100 })
    q.items.value = [done]
    await nextTick()
    vi.mocked(s3api.presign).mockResolvedValueOnce({ url: 'http://signed/1' } as unknown as Awaited<ReturnType<typeof s3api.presign>>)
    await findButton(w, 'upload.copyLink').trigger('click')
    await flushPromises()
    expect(s3api.presign).toHaveBeenCalledWith('acc-1', { method: 'get', key: 'a.txt', expiresIn: 3600 })
    expect(copyText).toHaveBeenCalledWith('http://signed/1')
    expect(toast).toHaveBeenLastCalledWith('upload.copiedLink')

    vi.mocked(s3api.presign).mockRejectedValueOnce(new Error('presign failed'))
    await findButton(w, 'upload.copyLink').trigger('click')
    await flushPromises()
    expect(toast).toHaveBeenLastCalledWith('presign failed', 'err')
  })

  it('start button disabled without items and no remove button for done items', async () => {
    const { w, q } = mountPanel()
    expect(findButton(w, 'upload.start').attributes('disabled')).toBeDefined()
    q.items.value = [item({ id: 1, key: 'a.txt', status: 'done', pct: 100 })]
    await nextTick()
    // done 行：无 remove 按钮
    expect(w.find('.list-item button[aria-label="common.remove"]').exists()).toBe(false)
    // 无 err → retry 禁用
    expect(findButton(w, 'upload.retryFailed').attributes('disabled')).toBeDefined()
  })
})

describe('UploadPanel keyFor', () => {
  it('keyFor: bare filename with empty prefix, prefixed after input', async () => {
    const { w } = mountPanel()
    expect((w.vm as unknown as { keyFor: (filename: string) => string }).keyFor('a.txt')).toBe('a.txt')
    const prefixInput = w.find('input[placeholder="upload.prefixPlaceholder"]')
    await prefixInput.setValue('dir/')
    expect((w.vm as unknown as { keyFor: (filename: string) => string }).keyFor('a.txt')).toBe('dir/a.txt')
  })
})

type QueueOptionsProbe = {
  onItemStart: (it: { id: number; file: File; key: string }) => void
  target: (it: { id: number; file: File; key: string }) => { accId: string; bucket?: string; key: string }
  selectBatch: (items: Array<{ status: string }>) => Array<{ status: string }>
}

describe('UploadPanel queue option callbacks', () => {
  it('drives target/onItemStart/selectBatch callbacks passed to useUploadQueue', async () => {
    const q = makeQueue()
    let opts!: QueueOptionsProbe
    vi.mocked(useUploadQueue).mockImplementation((o) => {
      opts = o as unknown as QueueOptionsProbe
      return q as unknown as ReturnType<typeof useUploadQueue>
    })
    const accState = ref<Account | undefined>(acc)
    vi.mocked(currentAccount).mockImplementation(() => accState.value)
    mounted = mount(UploadPanel)
    await mounted.find('input[placeholder="upload.prefixPlaceholder"]').setValue('docs//')

    expect(opts).toBeTruthy()
    // onItemStart：用当前前缀重写 key（keyFor 的 prefix 分支）
    const file = new File(['x'], 'b.txt')
    const it = { id: 1, file, key: '' }
    opts.onItemStart(it)
    expect(it.key).toBe('docs/b.txt')
    // target：当前账号 id + 更新后的 key
    expect(opts.target(it)).toEqual({ accId: 'acc-1', key: 'docs/b.txt' })
    // selectBatch：只保留 done/cancelled 之外的项
    expect(opts.selectBatch([
      { status: 'done' }, { status: 'cancelled' }, { status: 'pending' },
    ])).toEqual([{ status: 'pending' }])
  })

  it('无条目时 totalPct 为 0（不触发空数组 reduce）', () => {
    const { w } = mountPanel()
    expect((w.vm as unknown as { totalPct: number }).totalPct).toBe(0)
  })

  it('copyDoneLink 无账号时直接返回：不发起 presign', async () => {
    const { w } = mountPanel(null)
    await (w.vm as unknown as { copyDoneLink: (it: UploadQueueItem) => Promise<void> }).copyDoneLink(item())
    expect(s3api.presign).not.toHaveBeenCalled()
  })
})
