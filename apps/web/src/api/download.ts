import { opPath, requestResponse } from './http'
import { operations } from './operations'
import { t } from '../i18n'

const ZIP_BLOB_MAX_BYTES = 500 * 1024 * 1024 // 500MB blob 兜底上限

/**
 * blob 下载用的 object URL 回收延迟：click() 之后部分浏览器才异步读取 blob，
 * 同步 revokeObjectURL 会中断下载（0 字节/失败），因此推迟到下载启动之后再回收。
 */
const BLOB_URL_REVOKE_DELAY_MS = 60_000

type SaveFilePickerWindow = Window & {
  showSaveFilePicker?: (options?: {
    suggestedName?: string
    types?: { description: string; accept: Record<string, string[]> }[]
  }) => Promise<FileSystemFileHandle>
}

/** 将 ReadableStream 写入 File System Access API 的 writable。 */
async function streamBodyToFile(body: ReadableStream<Uint8Array>, handle: FileSystemFileHandle): Promise<void> {
  const writable = await handle.createWritable()
  try {
    await body.pipeTo(writable)
  } catch (e) {
    try {
      await writable.abort()
    } catch {
      /* ignore */
    }
    throw e
  }
}

/**
 * ZIP 下载：优先 File System Access API 流式落盘（避免整包进内存）；
 * 否则 blob 兜底，但 Content-Length > 500MB 或缺失且 keys>50 时拒绝。
 */
export async function downloadZipToDisk(
  id: string,
  body: { bucket?: string; keys: string[] },
  suggestedName?: string,
): Promise<void> {
  const keys = body.keys
  const path = opPath('downloadZip', { id })
  const res = await requestResponse(path, { method: operations.downloadZip.method, body: JSON.stringify(body) })
  const clHeader = res.headers.get('Content-Length')
  const contentLength = clHeader ? Number(clHeader) : NaN
  const filename =
    suggestedName ||
    `objects-${new Date().toISOString().slice(0, 19).replace(/[:T]/g, '-')}.zip`

  const w = window as SaveFilePickerWindow
  if (typeof w.showSaveFilePicker === 'function' && res.body) {
    try {
      const handle = await w.showSaveFilePicker({
        suggestedName: filename,
        types: [{ description: 'ZIP archive', accept: { 'application/zip': ['.zip'] } }],
      })
      await streamBodyToFile(res.body, handle)
      return
    } catch (e) {
      res.body.cancel().catch(() => {})
      throw e
    }
  }

  // blob 兜底：大包或未知大小且文件多时拒绝，避免 OOM
  if (
    (Number.isFinite(contentLength) && contentLength > ZIP_BLOB_MAX_BYTES) ||
    (!Number.isFinite(contentLength) && keys.length > 50)
  ) {
    res.body?.cancel().catch(() => {})
    throw new Error(t('api.zipTooLarge'))
  }
  const blob = await res.blob()
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  // 不在同一任务内 revoke：延迟回收，避免中断浏览器尚未开始的下载。
  setTimeout(() => URL.revokeObjectURL(url), BLOB_URL_REVOKE_DELAY_MS)
}

// ---- 对象下载：已知大小的大对象走有界并发 Range（ADR-009），其余单流 ----

/** Range 分段大小：4 MiB（减少往返，又足够细以摊平长尾）。 */
export const DOWNLOAD_PART_SIZE = 4 * 1024 * 1024
/** 并行分段并发上限（ADR-009：所有批量 / 流式路径有界；此处取 4 与分段上传一致）。 */
export const DOWNLOAD_CONCURRENCY = 4
/** 达到该大小才值得并行；更小的文件单流更快、更省往返。 */
export const PARALLEL_DOWNLOAD_MIN_BYTES = 16 * 1024 * 1024

/** 对象下载落盘请求（经 `/proxy`，带 Bearer；`size` 已知才可能走并行分段）。 */
export interface ObjectDownloadRequest {
  accountId: string
  bucket: string
  key: string
  apiBase: string
  token: string
  filename: string
  /** 对象字节数（来自列表 / HEAD）；未知则不并行。 */
  size?: number
  versionId?: string
  signal?: AbortSignal
}

/** 构造 `/proxy` 下载 URL（mode=download，服务端强制 `Content-Disposition: attachment`）。 */
function downloadProxyUrl(req: ObjectDownloadRequest): string {
  const p = new URLSearchParams({ bucket: req.bucket, key: req.key, mode: 'download' })
  if (req.versionId) p.set('versionId', req.versionId)
  return req.apiBase + `${opPath('proxyObject', { id: req.accountId })}?${p.toString()}`
}

/** 代理请求头：有 token 才带 Bearer（与 `proxy.ts` 同口径）。 */
function authHeaders(token: string): Record<string, string> {
  return token ? { Authorization: `Bearer ${token}` } : {}
}

/** objectURL → 触发浏览器保存 → 立即回收（与 `proxy.ts` 同口径；错误体绝不作为内容落盘）。 */
function saveBlobViaAnchor(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob)
  try {
    const a = document.createElement('a')
    a.href = url
    a.download = filename
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
  } finally {
    URL.revokeObjectURL(url)
  }
}

/**
 * 单流下载：整对象流式落盘（Range 不可用 / 大小未知 / 小文件的路径，也是并行分支的回退）。
 * 非 2xx 一律抛错——绝不把错误 JSON 当文件静默存盘。
 */
async function downloadSingle(req: ObjectDownloadRequest): Promise<void> {
  const res = await fetch(downloadProxyUrl(req), { headers: authHeaders(req.token), signal: req.signal })
  if (!res.ok) throw new Error(`${res.status} ${res.statusText}`)
  const w = window as SaveFilePickerWindow
  if (typeof w.showSaveFilePicker === 'function' && res.body) {
    try {
      const handle = await w.showSaveFilePicker({ suggestedName: req.filename })
      await streamBodyToFile(res.body, handle)
    } catch (e) {
      res.body.cancel().catch(() => {})
      throw e
    }
    return
  }
  saveBlobViaAnchor(await res.blob(), req.filename)
}

/**
 * 有界并发 Range GET → 按段号顺序聚合落盘。
 *
 * 每段校验：状态必须 206、字节数必须等于请求区间、`Content-Range`（若有）必须与请求区间一致；
 * 任一段不符即抛错且不产生任何落盘（长度 / 顺序校验，绝不静默产出损坏文件）。
 * 服务端忽略 Range（返回 200 全量）时干净回退单流。
 */
async function downloadParallel(req: ObjectDownloadRequest, size: number): Promise<void> {
  const total = Math.ceil(size / DOWNLOAD_PART_SIZE)
  const chunks: (Blob | undefined)[] = new Array(total)
  const headers = authHeaders(req.token)
  let next = 0
  let rangeSupported = true

  const worker = async () => {
    while (rangeSupported && next < total) {
      const index = next++
      const start = index * DOWNLOAD_PART_SIZE
      const end = Math.min(start + DOWNLOAD_PART_SIZE, size) - 1
      const res = await fetch(downloadProxyUrl(req), {
        headers: { ...headers, Range: `bytes=${start}-${end}` },
        signal: req.signal,
      })
      if (!res.ok) throw new Error(`${res.status} ${res.statusText}`)
      if (res.status !== 206) {
        // 服务端忽略 Range（返回 200 全量）：并行前提不成立，停掉其余并发并回退单流。
        rangeSupported = false
        res.body?.cancel().catch(() => {})
        return
      }
      const blob = await res.blob()
      if (blob.size !== end - start + 1) throw new Error(t('api.downloadPartMismatch'))
      const contentRange = res.headers.get('Content-Range')
      if (contentRange && contentRange !== `bytes ${start}-${end}/${size}`) {
        throw new Error(t('api.downloadPartMismatch'))
      }
      chunks[index] = blob
    }
  }

  await Promise.all(Array.from({ length: Math.min(DOWNLOAD_CONCURRENCY, total) }, worker))
  if (!rangeSupported) {
    await downloadSingle(req)
    return
  }

  // 每段按 index 落位：数组顺序即段号顺序；全部并发成功即无空洞（空洞会使 new Blob 静默缺段，
  // 因此上面的逐段校验是唯一的完整性防线，失败即抛错）。
  const ordered = chunks as Blob[]
  const w = window as SaveFilePickerWindow
  if (typeof w.showSaveFilePicker === 'function') {
    const handle = await w.showSaveFilePicker({ suggestedName: req.filename })
    const writable = await handle.createWritable()
    try {
      for (const c of ordered) await writable.write(c)
      await writable.close()
    } catch (e) {
      try {
        await writable.abort()
      } catch {
        /* 已尽力清理；原始错误优先上报 */
      }
      throw e
    }
    return
  }
  saveBlobViaAnchor(new Blob(ordered), req.filename)
}

/** 对象下载统一入口：已知大小的大对象走并行 Range，其余走单流。 */
export async function downloadObjectToDisk(req: ObjectDownloadRequest): Promise<void> {
  if (req.size !== undefined && req.size >= PARALLEL_DOWNLOAD_MIN_BYTES) {
    await downloadParallel(req, req.size)
    return
  }
  await downloadSingle(req)
}
