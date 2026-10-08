import { s3api, directUpload } from './api'
import { t, tf } from './i18n'
import { clearResume, loadResume, saveResume } from './multipartResume'

/** 上传目标（账号 + 桶 + key）。 */
export interface UploadTarget {
  accId: string
  bucket?: string
  key: string
}

/** 超过该大小的文件自动走 S3 分段上传（单 PUT 上限 5GB，分段对超大文件更稳、可并行）。 */
export const MULTIPART_THRESHOLD = 100 * 1024 * 1024 // 100MB

// S3 分段：除最后一段外每段最小 5MB；这里取 10MB 平衡并发数与段数量。
const PART_SIZE = 10 * 1024 * 1024
const PART_CONCURRENCY = 4

/** 分段 PUT 重试退避（ms）：约 500 → 1500 → 3500。 */
export const PART_RETRY_DELAYS_MS = [500, 1500, 3500] as const

function isAbortError(e: unknown): boolean {
  return e instanceof DOMException && e.name === 'AbortError'
}

function sleep(ms: number, signal?: AbortSignal): Promise<void> {
  return new Promise((resolve, reject) => {
    if (signal?.aborted) {
      reject(new DOMException('Aborted', 'AbortError'))
      return
    }
    const timer = setTimeout(() => {
      signal?.removeEventListener('abort', onAbort)
      resolve()
    }, ms)
    const onAbort = () => {
      clearTimeout(timer)
      reject(new DOMException('Aborted', 'AbortError'))
    }
    signal?.addEventListener('abort', onAbort, { once: true })
  })
}

/**
 * 带指数退避的重试：不重试 AbortError；最多尝试 delays.length + 1 次。
 * 导出供单测与其它弱网路径复用。
 */
export async function withRetries<T>(
  fn: () => Promise<T>,
  delaysMs: readonly number[] = PART_RETRY_DELAYS_MS,
  signal?: AbortSignal,
): Promise<T> {
  let lastErr: unknown
  const attempts = delaysMs.length + 1
  for (let i = 0; i < attempts; i++) {
    if (signal?.aborted) throw new DOMException('Aborted', 'AbortError')
    try {
      return await fn()
    } catch (e) {
      lastErr = e
      if (isAbortError(e)) throw e
      if (i >= delaysMs.length) break
      await sleep(delaysMs[i], signal)
    }
  }
  throw lastErr
}

/** 计算分段上传的段数（供测试与 UI 预估）。 */
export function calcMultipartParts(fileSize: number, partSize = PART_SIZE): number {
  if (fileSize <= 0) return 0
  return Math.ceil(fileSize / partSize)
}

/**
 * 上传单个文件：小文件走单次 PUT（presign），大文件自动分段上传。
 * onProgress 回调整数百分比（0-100）。signal 触发时中止 XHR 并尽力 abort 分段会话。
 */
export async function uploadObject(
  file: File,
  target: UploadTarget,
  onProgress?: (pct: number) => void,
  signal?: AbortSignal,
): Promise<void> {
  if (signal?.aborted) throw new DOMException('Aborted', 'AbortError')
  if (file.size < MULTIPART_THRESHOLD) {
    const presign = await s3api.presign(target.accId, {
      method: 'put',
      key: target.key,
      bucket: target.bucket,
      expiresIn: 3600,
    })
    if (signal?.aborted) throw new DOMException('Aborted', 'AbortError')
    return directUpload(presign.url, file, onProgress, signal)
  }
  return multipartUpload(file, target, onProgress, signal)
}

/** PUT 一个分段并返回其 ETag（complete 需要）。 */
function putPartReturnEtag(
  url: string,
  blob: Blob,
  onProgress?: (pct: number) => void,
  signal?: AbortSignal,
): Promise<string> {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest()
    let onAbort: (() => void) | undefined
    if (signal) {
      onAbort = () => {
        xhr.abort()
        reject(new DOMException('Aborted', 'AbortError'))
      }
      // 调用方（multipartUpload 的 worker）在 PUT 分段前已检查 signal.aborted，
      // 此处无需重复防御；直接挂监听即可。
      signal.addEventListener('abort', onAbort, { once: true })
    }
    const cleanup = () => {
      if (signal && onAbort) signal.removeEventListener('abort', onAbort)
    }
    xhr.open('PUT', url)
    xhr.upload.onprogress = (e) => {
      if (e.lengthComputable && onProgress) onProgress(Math.round((e.loaded / e.total) * 100))
    }
    xhr.onload = () => {
      cleanup()
      if (xhr.status >= 200 && xhr.status < 300) {
        resolve(xhr.getResponseHeader('ETag') ?? '')
      } else {
        reject(new Error(tf('upload.partHttpError', { status: xhr.status })))
      }
    }
    xhr.onerror = () => {
      cleanup()
      reject(new Error(t('upload.partNetworkError')))
    }
    xhr.onabort = () => {
      cleanup()
      reject(new DOMException('Aborted', 'AbortError'))
    }
    xhr.send(blob)
  })
}

async function multipartUpload(
  file: File,
  target: UploadTarget,
  onProgress?: (pct: number) => void,
  signal?: AbortSignal,
): Promise<void> {
  const totalParts = calcMultipartParts(file.size)
  const parts: UploadedPart[] = new Array(totalParts)
  let uploadId = ''
  let doneBytes = 0
  let aborted = false
  /** 已向服务端发起过 abort（幂等）：会话只在拿到 uploadId 后才能真正清理。 */
  let sessionAborted = false
  const abort = () => {
    aborted = true
    clearResume(file)
    if (!sessionAborted && uploadId) {
      sessionAborted = true
      s3api.multipartAbort(target.accId, { bucket: target.bucket, key: target.key, uploadId }).catch(() => {})
    }
  }
  signal?.addEventListener('abort', abort, { once: true })

  try {
    // 1) 断点续传：本地记录只是候选，能否复用由服务端 ListParts 清单决定。
    const saved = loadResume(file)
    if (saved) {
      if (saved.accId === target.accId && saved.key === target.key && saved.bucket === target.bucket) {
        try {
          const server = await s3api.multipartParts(target.accId, {
            bucket: target.bucket,
            key: target.key,
            uploadId: saved.uploadId,
          })
          if (server.parts.length > 0) {
            // 服务端真实清单为准：只采纳落在本文件段号范围内的分段。
            uploadId = saved.uploadId
            for (const p of server.parts) {
              if (p.partNumber >= 1 && p.partNumber <= totalParts) {
                parts[p.partNumber - 1] = { partNumber: p.partNumber, etag: p.etag }
              }
            }
            doneBytes = countDoneBytes(file.size, parts)
            onProgress?.(Math.min(100, Math.round((doneBytes / file.size) * 100)))
          } else {
            // 服务端清单为空 = 旧会话没有可复用分段：清理后走全新 init。
            s3api.multipartAbort(target.accId, { bucket: target.bucket, key: target.key, uploadId: saved.uploadId }).catch(() => {})
            clearResume(file)
          }
        } catch {
          // uploadId 失效（NoSuchUpload）/ 服务端不可达：清掉本地记录，干净地重新 init。
          clearResume(file)
        }
      } else {
        // 同文件指纹但目标账号 / 桶 / key 已变：旧记录不可复用。
        clearResume(file)
      }
    }
    // 2) 无可复用会话则 init，并落一条续传记录（供刷新 / 断电后对齐）。
    if (!uploadId) {
      const init = await s3api.multipartInit(target.accId, {
        bucket: target.bucket,
        key: target.key,
        contentType: file.type || undefined,
      })
      uploadId = init.uploadId
      saveResume(file, { accId: target.accId, key: target.key, bucket: target.bucket, uploadId, parts: [] })
    }
    if (signal?.aborted || aborted) {
      // 中止若发生在 init 期间，abort 监听还拿不到 uploadId；这里补一次会话清理（幂等）。
      abort()
      throw new DOMException('Aborted', 'AbortError')
    }

    let idx = 0
    const worker = async () => {
      while (idx < totalParts) {
        if (signal?.aborted || aborted) throw new DOMException('Aborted', 'AbortError')
        const n = idx++ + 1
        // 服务端清单已确认的段直接跳过：只补缺段。
        if (parts[n - 1]) continue
        const start = (n - 1) * PART_SIZE
        const end = Math.min(start + PART_SIZE, file.size)
        const blob = file.slice(start, end)
        const etag = await withRetries(async () => {
          const presign = await s3api.multipartPart(target.accId, {
            bucket: target.bucket,
            key: target.key,
            uploadId,
            partNumber: n,
            expiresIn: 3600,
          })
          if (signal?.aborted || aborted) throw new DOMException('Aborted', 'AbortError')
          return putPartReturnEtag(presign.url, blob, (p) => {
            const partLoaded = Math.round((p / 100) * blob.size)
            const overall = Math.min(100, Math.round(((doneBytes + partLoaded) / file.size) * 100))
            onProgress?.(overall)
          }, signal)
        }, PART_RETRY_DELAYS_MS, signal)
        if (!etag) {
          throw new Error(t('upload.partNoEtag'))
        }
        parts[n - 1] = { partNumber: n, etag }
        doneBytes += blob.size
        // 每段完成即更新续传记录：刷新 / 断电后从该点继续。
        saveResume(file, {
          accId: target.accId,
          key: target.key,
          bucket: target.bucket,
          uploadId,
          parts: parts.filter(isUploadedPart),
        })
        onProgress?.(Math.min(100, Math.round((doneBytes / file.size) * 100)))
      }
    }
    await Promise.all(Array.from({ length: Math.min(PART_CONCURRENCY, totalParts) }, worker))
    if (signal?.aborted || aborted) throw new DOMException('Aborted', 'AbortError')
    await s3api.multipartComplete(target.accId, {
      bucket: target.bucket,
      key: target.key,
      uploadId,
      parts,
    })
    clearResume(file)
    onProgress?.(100)
  } catch (e) {
    if (!isAbortError(e)) abort()
    throw e
  } finally {
    signal?.removeEventListener('abort', abort)
  }
}

/** 已完成分段（服务端清单 / 本地 PUT 回执统一形状）。 */
type UploadedPart = { partNumber: number; etag: string }

/** 过滤稀疏分段数组中的空位（`new Array(n)` 的空洞会被 filter 跳过）。 */
function isUploadedPart(p: UploadedPart | undefined): p is UploadedPart {
  return p !== undefined
}

/** 第 n 段的字节数（续传进度按本地文件大小折算已确认段）。 */
function partBytes(fileSize: number, partNumber: number): number {
  const start = (partNumber - 1) * PART_SIZE
  return Math.min(start + PART_SIZE, fileSize) - start
}

/** 统计分段数组中已确认段的总字节数（用于续传起始进度）。 */
function countDoneBytes(fileSize: number, parts: (UploadedPart | undefined)[]): number {
  let total = 0
  for (let i = 0; i < parts.length; i++) {
    if (parts[i]) total += partBytes(fileSize, i + 1)
  }
  return total
}
