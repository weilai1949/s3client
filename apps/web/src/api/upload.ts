/**
 * 预签名直传（从 `api.ts` 拆出，行为不变）：用 XHR 而非 fetch —— 只有 XHR 能提供
 * 上传进度事件（`upload.onprogress`）。字节不经过 Go 服务，直接 PUT 到 S3。
 */
export function directUpload(
  url: string,
  file: File,
  onProgress?: (pct: number) => void,
  signal?: AbortSignal,
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
    xhr.upload.onprogress = (e) => {
      if (e.lengthComputable && onProgress) onProgress(Math.round((e.loaded / e.total) * 100))
    }
    xhr.onload = () => {
      cleanup()
      if (xhr.status >= 200 && xhr.status < 300) resolve()
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
