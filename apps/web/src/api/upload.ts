/**
 * 预签名直传（从 `api.ts` 拆出，行为不变）：用 XHR 而非 fetch —— 只有 XHR 能提供
 * 上传进度事件（`upload.onprogress`）。字节不经过 Go 服务，直接 PUT 到 S3。
 */
import { t } from '../i18n'

export function directUpload(
  url: string,
  file: File,
  onProgress?: (pct: number) => void,
  signal?: AbortSignal,
  /**
   * 由后端签进 URL 的条件写请求头（presign 响应的 `headers`）——S3 会对条件求值，
   * 条件头也参与签名，**必须原样带上**，否则签名不匹配直接 403。
   */
  headers?: Record<string, string>,
): Promise<void> {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest()
    let onAbort: (() => void) | undefined
    if (signal) {
      onAbort = () => {
        xhr.abort()
        reject(new DOMException('Aborted', 'AbortError'))
      }
      if (signal.aborted) {
        onAbort()
        return
      }
      signal.addEventListener('abort', onAbort, { once: true })
    }
    const cleanup = () => {
      if (signal && onAbort) signal.removeEventListener('abort', onAbort)
    }
    xhr.open('PUT', url)
    xhr.setRequestHeader('Content-Type', file.type || 'application/octet-stream')
    for (const [name, value] of Object.entries(headers ?? {})) xhr.setRequestHeader(name, value)
    xhr.upload.onprogress = (e) => {
      if (e.lengthComputable && onProgress) onProgress(Math.round((e.loaded / e.total) * 100))
    }
    xhr.onload = () => {
      cleanup()
      if (xhr.status >= 200 && xhr.status < 300) resolve()
      // 412 = 条件写不满足（If-None-Match 命中已有对象 / If-Match 不匹配）：
      // 给用户一句能看懂的话，而不是裸状态码。
      else if (xhr.status === 412) reject(new Error(t('upload.conditionalFailed')))
      else reject(new Error(`upload failed: HTTP ${xhr.status}`))
    }
    xhr.onerror = () => {
      cleanup()
      reject(new Error('upload network error'))
    }
    xhr.onabort = () => {
      cleanup()
      reject(new DOMException('Aborted', 'AbortError'))
    }
    xhr.send(file)
  })
}
