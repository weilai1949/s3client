import { onBeforeUnmount, onDeactivated, ref, toRaw } from 'vue'
import type { Ref } from 'vue'
import { toErrorMessage } from '../errors'
import { uploadObject } from '../upload'

/**
 * 直传并发数（两套上传入口统一取原对象面板值 2）：
 * 过低慢，过高浏览器/对端都吃力；如需调整只改这一个常量。
 */
const UPLOAD_CONCURRENCY = 2

/** 上传条目状态：cancelled 是用户主动中止的终态——不再回 pending 重新组批上传。 */
export type UploadItemStatus = 'pending' | 'signing' | 'uploading' | 'done' | 'err' | 'cancelled'

/** 统一的上传队列条目结构（UploadPanel 与对象面板共享）。 */
export interface UploadQueueItem {
  id?: number
  file: File
  key: string
  /** 入队时所属桶（防止上传途中切换桶导致串桶）。 */
  bucket?: string
  pct: number
  status: UploadItemStatus
  err?: string
}

export interface UploadQueueOptions {
  /** 单条上传目标（账号/桶/key），run 时逐条求值。 */
  target: (it: UploadQueueItem) => { accId: string; bucket?: string; key: string }
  /** 选中本轮参与上传的条目；缺省取全部 pending。 */
  selectBatch?: (items: UploadQueueItem[]) => UploadQueueItem[]
  /** 单批传完后继续吸收新入队的 pending（对象面板边传边加）；false = 单批快照（上传面板）。默认 true。 */
  drain?: boolean
  /** 单条中止后的去向：requeue=在途回 pending（可再次上传，上传面板）；
   * cancel=在途先标 cancelled 终态再 abort（对象面板，默认）。
   * 未开始的 pending 条目两种策略下都直接标 cancelled 终态。 */
  onAbort?: 'requeue' | 'cancel'
  /** 单条开始上传前钩子（如按当前前缀重算 key）。 */
  onItemStart?: (it: UploadQueueItem) => void
  /** 进度回调（缺省写入 item.pct）。 */
  onProgress?: (it: UploadQueueItem, pct: number) => void
}

export interface UploadQueue {
  items: Ref<UploadQueueItem[]>
  running: Ref<boolean>
  /** 入队：全部文件逐个入队（0 字节空文件也入队，不当目录占位丢弃）；make 供调用方补充 key/bucket 等字段。 */
  enqueue: (files: FileList | null, make?: (file: File) => Partial<UploadQueueItem>) => void
  /** 运行共享状态机；返回本轮实际选中的条目（按选中顺序）。 */
  run: () => Promise<UploadQueueItem[]>
  abortItem: (it: UploadQueueItem) => void
  abortAll: () => void
}

/**
 * 上传队列组合式：统一条目结构、并发控制、状态机与 abort 支持，
 * 供 UploadPanel（独立上传页）与 useObjectActions（对象面板内嵌上传）共用。
 * 两边只保留 UI 层差异（分组/展示/触发方式/完成后的提示与刷新）。
 */
export function useUploadQueue(options: UploadQueueOptions): UploadQueue {
  const items = ref<UploadQueueItem[]>([])
  const running = ref(false)
  /** 在途条目 → AbortController（toRaw 作键，避免响应式代理污染键）。 */
  const aborts = new Map<UploadQueueItem, AbortController>()
  let seq = 0
  /** abortAll 置位：停住 run 的取件循环（run 启动时复位），杜绝后台续传。 */
  let stopped = false

  const selectBatch = options.selectBatch ?? ((all: UploadQueueItem[]) => all.filter((it) => it.status === 'pending'))
  const writePct = options.onProgress ?? ((it: UploadQueueItem, pct: number) => (it.pct = pct))

  function enqueue(files: FileList | null, make?: (file: File) => Partial<UploadQueueItem>) {
    if (!files || !files.length) return
    for (const f of Array.from(files)) {
      items.value.push({ id: ++seq, file: f, key: '', pct: 0, status: 'pending', ...make?.(f) })
    }
  }

  /** 单条中止：pending（已入批未开始）两种策略下都直接标 cancelled 终态——
   * 批次快照仍持有它，不标终态 worker 会照常捞起来在后台上传（UI 已移除/清空也拦不住）。
   * requeue 仅豁免在途条目：abort 后回 pending 供重试（由 run 的 catch 落实）。 */
  function abortItem(it: UploadQueueItem) {
    const inflight = it.status === 'signing' || it.status === 'uploading'
    if (it.status === 'pending' || (inflight && options.onAbort !== 'requeue')) it.status = 'cancelled'
    const raw = toRaw(it)
    aborts.get(raw)?.abort()
    aborts.delete(raw)
  }

  /** 全量中止（切后台 / 卸载 / 清空队列）：abort 在途条目，并停住状态机——
   * 否则 requeue 把在途条目送回 pending 后，worker 会继续消费批次快照在后台续传。 */
  function abortAll() {
    stopped = true
    for (const it of [...aborts.keys()]) abortItem(it)
  }

  async function run(): Promise<UploadQueueItem[]> {
    if (running.value) return []
    stopped = false
    running.value = true
    const processed: UploadQueueItem[] = []
    try {
      // 循环直到队列中没有待上传项（drain 模式下期间可继续追加文件）
      for (;;) {
        if (stopped) break
        const batch = selectBatch(items.value)
        if (!batch.length) break
        processed.push(...batch)
        let i = 0
        const worker = async () => {
          while (i < batch.length) {
            if (stopped) break // abortAll 后批内剩余条目不再启动
            const it = batch[i++]
            // 批内已被取消的条目直接跳过：不发起请求、不占用并发位
            if (it.status === 'cancelled') continue
            const ctrl = new AbortController()
            aborts.set(toRaw(it), ctrl)
            try {
              options.onItemStart?.(it)
              it.status = 'signing'
              it.err = undefined
              // 首次收到字节进度才离开签名阶段：presign 是一次网络往返，期间该状态
              // 必须停得住。此前把 `status='uploading'` 直接写在下一行（同一同步块、
              // 中间无 await）→ 渲染永远插不进来，「签名中…」标签与 abortItem 的
              // signing 分支都成了被覆盖率掩盖的死状态。
              await uploadObject(
                it.file,
                options.target(it),
                (p) => {
                  if (it.status === 'signing') it.status = 'uploading'
                  writePct(it, p)
                },
                ctrl.signal,
              )
              it.status = 'done'
              it.pct = 100
            } catch (err) {
              // status 可能已被 abortItem 外部置为 cancelled（TS 无法看到跨函数可变状态）
              const st = it.status as UploadItemStatus
              if (st === 'cancelled') {
                // cancelled 是终态：保持，外层组批与批内 worker 都会跳过
              } else if (err instanceof DOMException && err.name === 'AbortError') {
                it.status = 'pending'
                it.pct = 0
              } else {
                it.status = 'err'
                it.err = toErrorMessage(err)
              }
            } finally {
              aborts.delete(toRaw(it))
            }
          }
        }
        await Promise.all(Array.from({ length: Math.min(UPLOAD_CONCURRENCY, batch.length) }, worker))
        if (options.drain === false) break
      }
      return processed
    } finally {
      running.value = false
    }
  }

  // 面板卸载 / KeepAlive 切走时中止上传并停住状态机（对象面板：在途标 cancelled 终态；
  // 上传面板：在途回 pending 可续传，批内未开始条目不被后台捞起）
  onDeactivated(abortAll)
  onBeforeUnmount(abortAll)

  return { items, running, enqueue, run, abortItem, abortAll }
}
