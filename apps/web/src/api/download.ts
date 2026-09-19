import { requestResponse } from './http'
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
  const path = `/api/accounts/${id}/download-zip`
  const res = await requestResponse(path, { method: 'POST', body: JSON.stringify(body) })
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
