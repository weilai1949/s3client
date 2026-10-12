import { defineComponent, nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useUploadQueue } from './useUploadQueue'
import type { UploadQueueItem, UploadQueueOptions } from './useUploadQueue'
import { uploadObject } from '../upload'
import { activities } from '../store'

/**
 * 直传并发上限（产品约定 2，见 useUploadQueue 注释）。
 * 字面量写死在测试侧、不从实现导入：实现改了并发数而测试没改，这里就会红灯。
 */
const MAX_CONCURRENCY = 2

vi.mock('../upload', () => ({ uploadObject: vi.fn() }))

vi.mocked(uploadObject).mockImplementation((_file, _target, onProgress, signal) => {
  return new Promise<void>((resolve, reject) => {
    if (signal?.aborted) {
      reject(new DOMException('Aborted', 'AbortError'))
      return
    }
    signal?.addEventListener('abort', () => reject(new DOMException('Aborted', 'AbortError')), { once: true })
    inFlight.push({ resolve, reject, progress: onProgress })
  })
})

/** 挂起中的上传，测试里手动推进完成；progress 用于手动投递字节进度回调。 */
const inFlight: {
  resolve: () => void
  reject: (e: unknown) => void
  progress?: (pct: number) => void
}[] = []

/** 挂一个宿主组件：setup 内调用组合式，把返回值存到外层变量供断言。 */
function mountQueue(options: Partial<UploadQueueOptions> = {}): ReturnType<typeof useUploadQueue> {
  let captured!: ReturnType<typeof useUploadQueue>
  const Host = defineComponent({
    setup() {
      captured = useUploadQueue({ target: (it) => ({ accId: 'a1', key: it.key }), ...options })
      return () => null
    },
  })
  mount(Host)
  return captured
}

function enqueue(q: ReturnType<typeof useUploadQueue>, names: string[]): UploadQueueItem[] {
  return names.map((name) => {
    const it: UploadQueueItem = { file: new File(['x'], name), key: name, pct: 0, status: 'pending' }
    q.items.value.push(it)
    return it
  })
}

describe('useUploadQueue 共享状态机', () => {
  beforeEach(() => {
    inFlight.length = 0
    vi.mocked(uploadObject).mockClear()
  })

  it('cancel 策略：取消等待中的条目直接跳过（不发起请求、不占用并发位），cancelled 保持终态', async () => {
    const q = mountQueue() // 默认 cancel 策略 + drain（对象面板语义）
    // 并发位之外再多排 2 条：1 条待取消、1 条正常等待
    const items = enqueue(q, Array.from({ length: MAX_CONCURRENCY + 2 }, (_, i) => `f${i}.txt`))
    const run = q.run()
    await vi.waitFor(() => expect(inFlight.length).toBe(MAX_CONCURRENCY))

    const cancelled = items[MAX_CONCURRENCY]
    q.abortItem(cancelled)
    expect(cancelled.status).toBe('cancelled')

    inFlight.splice(0).forEach((f) => f.resolve())
    // 并发位空出后，批内下一条（未被取消）继续上传，被取消的那条被跳过
    await vi.waitFor(() => expect(inFlight.length).toBe(1))
    inFlight[0].resolve()
    await run

    expect(uploadObject).toHaveBeenCalledTimes(MAX_CONCURRENCY + 1)
    expect(cancelled.status).toBe('cancelled')
    expect(q.items.value.filter((it) => it.status === 'done').length).toBe(MAX_CONCURRENCY + 1)
  })

  it('requeue 策略（上传面板语义）：中止在途条目回 pending，可再次上传到 done', async () => {
    const q = mountQueue({ onAbort: 'requeue', drain: false })
    const [a] = enqueue(q, ['a.txt'])
    const run = q.run()
    await vi.waitFor(() => expect(inFlight.length).toBe(1))

    q.abortItem(a) // 仅中止，不标记 cancelled（状态仍 signing，待 catch 回 pending）
    expect(a.status).toBe('signing')
    inFlight.splice(0).forEach((f) => f.reject(new DOMException('Aborted', 'AbortError')))
    await run
    expect(a.status).toBe('pending')
    expect(a.pct).toBe(0)

    const run2 = q.run()
    await vi.waitFor(() => expect(inFlight.length).toBe(1))
    inFlight[0].resolve()
    const processed = await run2
    expect(a.status).toBe('done')
    expect(a.pct).toBe(100)
    expect(processed.map((it) => it.key)).toContain('a.txt')
  })

  it('尚无字节进度时状态停在 signing（不是赋值后立即被覆盖）', async () => {
    const q = mountQueue()
    const [a] = enqueue(q, ['a.txt'])
    const run = q.run()
    await vi.waitFor(() => expect(inFlight.length).toBe(1))

    // presign 是一次网络往返：期间必须停在 signing。旧实现把
    // `status='signing'` 与 `status='uploading'` 写在同一个同步块里（中间无 await），
    // 渲染永远插不进来 ⇒「签名中…」标签与 abortItem 的 signing 分支全是死代码。
    expect(a.status, '在途且尚无字节进度应停在 signing').toBe('signing')

    // 首次收到字节进度 → 离开签名阶段进入上传阶段
    inFlight[0].progress?.(10)
    await nextTick()
    expect(a.status, '首次字节进度后转 uploading').toBe('uploading')
    expect(a.pct).toBe(10)

    inFlight.splice(0).forEach((f) => f.resolve())
    await run
    expect(a.status).toBe('done')
  })

  it('requeue 策略：取消已入批未开始（pending）条目标 cancelled 终态，不再被后台上传', async () => {
    const q = mountQueue({ onAbort: 'requeue', drain: false })
    const items = enqueue(q, Array.from({ length: MAX_CONCURRENCY + 1 }, (_, i) => `f${i}.txt`))
    const run = q.run()
    await vi.waitFor(() => expect(inFlight.length).toBe(MAX_CONCURRENCY))

    const queued = items[MAX_CONCURRENCY]
    expect(queued.status).toBe('pending') // 已入批、未轮到并发位
    q.abortItem(queued)
    expect(queued.status).toBe('cancelled') // requeue 只豁免在途条目；未开始的取消 = 终态

    inFlight.splice(0).forEach((f) => f.resolve())
    await run
    // 被取消的那条从未发起上传（批次快照持有它，靠 cancelled 终态挡住）
    expect(uploadObject).toHaveBeenCalledTimes(MAX_CONCURRENCY)
    expect(queued.status).toBe('cancelled')
  })

  it('abortAll 中止运行中的队列：批内剩余 pending 不再被后台捞起（清空/切走后不残留）', async () => {
    const q = mountQueue({ onAbort: 'requeue' }) // 对象面板式 drain：单批 + 继续吸收
    const items = enqueue(q, Array.from({ length: MAX_CONCURRENCY + 2 }, (_, i) => `f${i}.txt`))
    const run = q.run()
    await vi.waitFor(() => expect(inFlight.length).toBe(MAX_CONCURRENCY))

    q.abortAll()
    inFlight.splice(0).forEach((f) => f.reject(new DOMException('Aborted', 'AbortError')))
    await new Promise((r) => setTimeout(r, 10)) // 让 worker 的 catch 与后续取件全部走完

    // 只有最初两条发起过上传：剩余 pending 被停住，不在后台续传
    expect(uploadObject).toHaveBeenCalledTimes(MAX_CONCURRENCY)
    await run
    expect(q.running.value).toBe(false)
    // requeue 语义：在途两条中止后回 pending，未开始的保持 pending，全部可再次上传
    expect(items.every((it) => it.status === 'pending')).toBe(true)
  })

  it('enqueue 入队 0 字节无 MIME 的空文件（不得当目录占位丢弃）', () => {
    const q = mountQueue()
    const emptyFile = new File([''], '', { type: '' })
    const normalFile = new File(['x'], 'a.txt', { type: 'text/plain' })
    const fileList = [emptyFile, normalFile] as unknown as FileList
    q.enqueue(fileList)
    expect(q.items.value.length).toBe(2)
    expect(q.items.value[0].file.name).toBe('')
    expect(q.items.value[1].file.name).toBe('a.txt')
  })

  it('enqueueItems 直接入队已算好 key 的条目（拖拽上传：逐条不同路径）', () => {
    const q = mountQueue()
    q.enqueueItems([
      { file: new File(['1'], 'a.jpg'), key: 'photos/a.jpg', bucket: 'b1' },
      { file: new File(['2'], 'b.jpg'), key: 'photos/2024/b.jpg', bucket: 'b1' },
    ])
    expect(q.items.value.map((it) => it.key)).toEqual(['photos/a.jpg', 'photos/2024/b.jpg'])
    expect(q.items.value.every((it) => it.status === 'pending' && it.pct === 0 && it.bucket === 'b1')).toBe(true)
    // id 单调递增：UploadQueue 用 it.id 做 v-for key（同名前缀的两条不得碰撞）
    expect(q.items.value[1].id).toBeGreaterThan(q.items.value[0].id!)
  })

  it('abortAll aborts all in-flight items', async () => {
    const q = mountQueue()
    const [a] = enqueue(q, ['a.txt'])
    const [b] = enqueue(q, ['b.txt'])
    const run = q.run()
    await vi.waitFor(() => expect(inFlight.length).toBe(2))
    q.abortAll()
    await run
    expect(a.status).toBe('cancelled')
    expect(b.status).toBe('cancelled')
  })

  it('abortAll on empty queue is a no-op', () => {
    const q = mountQueue()
    expect(() => q.abortAll()).not.toThrow()
  })

  it('worker catch sets status=err and err message on non-AbortError', async () => {
    const q = mountQueue()
    const [item] = enqueue(q, ['a.txt'])
    // Make uploadObject throw immediately so the worker catches it
    vi.mocked(uploadObject).mockImplementationOnce(async () => {
      throw new Error('upload failed')
    })
    const run = q.run()
    // No inFlight entry because uploadObject threw before pushing
    await run
    expect(item.status).toBe('err')
    expect(item.err).toBe('upload failed')
  })

  it('默认参数：进度回调闭包经默认 writePct 落盘 it.pct（onProgress fallback）', async () => {
    const q = mountQueue({ drain: false }) // 不带 onProgress/selectBatch/onItemStart
    const [a] = enqueue(q, ['a.txt'])
    let gotPct: ((p: number) => void) | undefined
    vi.mocked(uploadObject).mockImplementationOnce(async (_f, _t, pct) => {
      gotPct = pct
    })
    await q.run()
    expect(a.status).toBe('done')
    // 强制调用传给 uploadObject 的进度回调：验证 `(p) => writePct(it, p)` → `it.pct = p`
    expect(gotPct).toBeTypeOf('function')
    gotPct!(42)
    expect(a.pct).toBe(42)
  })

  it('enqueue 传入 null/空 FileList 直接返回（early-return，不 push 条目）', () => {
    const q = mountQueue()
    q.enqueue(null)
    q.enqueue([] as unknown as FileList)
    expect(q.items.value.length).toBe(0)
  })

  it('abortItem 取消 signing 状态条目：默认 cancel 策略标 cancelled', () => {
    const q = mountQueue()
    const it: UploadQueueItem = { file: new File(['x'], 's.txt'), key: 's.txt', pct: 0, status: 'signing' }
    q.items.value.push(it)
    q.abortItem(it)
    expect(it.status).toBe('cancelled')
  })

  it('abortItem 对终态条目（done/err）不改变状态（非 pending/signing/uploading 的 false 侧）', () => {
    const q = mountQueue()
    const done: UploadQueueItem = { file: new File(['x'], 'd.txt'), key: 'd.txt', pct: 100, status: 'done' }
    const err: UploadQueueItem = { file: new File(['x'], 'e.txt'), key: 'e.txt', pct: 0, status: 'err' }
    q.items.value.push(done, err)
    q.abortItem(done)
    q.abortItem(err)
    expect(done.status).toBe('done')
    expect(err.status).toBe('err')
  })

  it('running 中再次 run 直接返回空数组', async () => {
    const q = mountQueue({ drain: false })
    enqueue(q, ['a.txt'])
    const first = q.run()
    await vi.waitFor(() => expect(inFlight.length).toBe(1))
    const second = await q.run()
    expect(second).toEqual([])
    inFlight.splice(0).forEach((f) => f.resolve())
    await first
    expect(q.items.value[0].status).toBe('done')
  })
})

describe('useUploadQueue 全局活动登记（顶栏角标）', () => {
  const KEY = 'upload:panel'

  beforeEach(() => {
    inFlight.length = 0
    vi.mocked(uploadObject).mockClear()
    vi.mocked(uploadObject).mockImplementation((_file, _target, onProgress, signal) => {
      return new Promise<void>((resolve, reject) => {
        if (signal?.aborted) {
          reject(new DOMException('Aborted', 'AbortError'))
          return
        }
        signal?.addEventListener('abort', () => reject(new DOMException('Aborted', 'AbortError')), { once: true })
        inFlight.push({ resolve, reject, progress: onProgress })
      })
    })
    delete activities[KEY]
  })

  /** 挂宿主组件并同时拿到组合式返回值与组件包装器（卸载用）。 */
  function mountHost(options: Partial<UploadQueueOptions>) {
    let q!: ReturnType<typeof useUploadQueue>
    const Host = defineComponent({
      setup() {
        q = useUploadQueue({ target: (it) => ({ accId: 'a1', key: it.key }), ...options })
        return () => null
      },
    })
    const wrapper = mount(Host)
    return { q, wrapper }
  }

  function push(q: ReturnType<typeof useUploadQueue>, name: string): UploadQueueItem {
    const it: UploadQueueItem = { file: new File(['x'], name), key: name, pct: 0, status: 'pending' }
    q.items.value.push(it)
    return it
  }

  it('开启 activity：入队后登记、进度更新、全部完成后撤下', async () => {
    const { q } = mountHost({ activity: { key: KEY, label: () => '上传' }, drain: false })
    // immediate：空队列 → 不登记
    expect(activities[KEY]).toBeUndefined()

    push(q, 'a.txt')
    await nextTick()
    expect(activities[KEY]).toMatchObject({ key: KEY, label: '上传', done: 0, total: 1, pct: 0 })

    const run = q.run()
    await vi.waitFor(() => expect(inFlight.length).toBe(1))
    inFlight[0].progress?.(40)
    await nextTick()
    expect(activities[KEY].pct).toBe(40)

    inFlight.splice(0).forEach((f) => f.resolve())
    await run
    await nextTick()
    // 全部 done：没有进行中的条目 → 撤下登记（角标不长期停留）
    expect(activities[KEY]).toBeUndefined()
  })

  it('全部条目被取消（cancelled 不算进行中）→ 撤下登记', async () => {
    const { q } = mountHost({ activity: { key: KEY, label: () => '上传' } })
    push(q, 'a.txt')
    await nextTick()
    expect(activities[KEY]).toBeDefined()

    const run = q.run()
    await vi.waitFor(() => expect(inFlight.length).toBe(1))
    // 传队列里的响应式条目（v-for / 模板路径也是这样拿到的）：裸对象改状态不触发响应式
    q.abortItem(q.items.value[0]) // 默认 cancel 策略 → 终态 cancelled
    await run
    await nextTick()
    expect(activities[KEY]).toBeUndefined()
  })

  it('未开启 activity 时不登记（默认选项无 activity）', async () => {
    const { q, wrapper } = mountHost({ drain: false })
    push(q, 'a.txt')
    await nextTick()
    expect(activities[KEY]).toBeUndefined()
    // 卸载路径同样不碰活动表（无 activity 选项时 clearActivity 不被调用）
    wrapper.unmount()
    expect(activities[KEY]).toBeUndefined()
  })

  it('卸载时中止上传并撤下活动登记（卸载后没有东西再更新它）', async () => {
    const { q, wrapper } = mountHost({ activity: { key: KEY, label: () => '上传' } })
    const it = push(q, 'a.txt')
    const run = q.run()
    await vi.waitFor(() => expect(inFlight.length).toBe(1))
    expect(activities[KEY]).toBeDefined()

    wrapper.unmount()
    expect(it.status).toBe('cancelled') // abortAll：在途按 cancel 策略落终态
    expect(activities[KEY]).toBeUndefined()
    await run
  })
})
