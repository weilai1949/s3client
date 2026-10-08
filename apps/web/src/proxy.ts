import { opPath } from './api/http'
import { downloadObjectToDisk, PARALLEL_DOWNLOAD_MIN_BYTES } from './api/download'

/** 共享的「服务端代理 / 下载 URL」构造器（安全预览与下载统一入口）。 */

/**
 * 构造安全代理 URL：
 *   download=强制 attachment 下载（内容不进渲染管道）
 *   inline=透传 Content-Type（图片/PDF/媒体预览）
 *   text=强制 text/plain + nosniff（文本预览）
 */
export function proxyUrl(
  accountId: string,
  bucket: string,
  mode: 'download' | 'inline' | 'text',
  key: string,
  apiBase: string,
  versionId?: string,
): string {
  const p = new URLSearchParams({ bucket, key, mode })
  if (versionId) p.set('versionId', versionId)
  return apiBase + `${opPath('proxyObject', { id: accountId })}?${p.toString()}`
}

/**
 * 代理请求参数：`<a href>` / `<img src>` / `<iframe src>` 这类资源加载**无法携带
 * Authorization 头**，在启用 S3C_TOKEN 的部署下会 401（预览全挂 / 下载把 401 的
 * JSON 错误体当文件静默存盘）。因此所有代理取回统一走下面的 fetch 辅助。
 */
export interface ProxyRequest {
  accountId: string
  bucket: string
  mode: 'download' | 'inline' | 'text'
  key: string
  apiBase: string
  /** Bearer Token（`api.token`）；空串时不携带鉴权头。 */
  token: string
  versionId?: string
  signal?: AbortSignal
}

/**
 * 统一带鉴权的代理请求：Bearer 头 + `res.ok` 校验，非 2xx 一律抛错。
 * 错误响应体（如鉴权失败的 JSON）绝不作为内容返回——调用方拿到的要么是
 * 真正的对象字节，要么是一个会走错误提示分支的异常。
 */
export async function fetchProxy(req: ProxyRequest): Promise<Response> {
  const res = await fetch(proxyUrl(req.accountId, req.bucket, req.mode, req.key, req.apiBase, req.versionId), {
    headers: req.token ? { Authorization: `Bearer ${req.token}` } : {},
    signal: req.signal,
  })
  if (!res.ok) throw new Error(`${res.status} ${res.statusText}`)
  return res
}

/** 代理取回对象字节（媒体/PDF 预览与下载落盘共用）。失败抛错。 */
export async function fetchProxyBlob(req: ProxyRequest): Promise<Blob> {
  return (await fetchProxy(req)).blob()
}

/**
 * 经代理下载对象为附件：带 Bearer 取回字节 → objectURL → 触发保存 → 释放 URL。
 * 失败抛错且不产生任何落盘动作（错误体绝不当文件保存），由调用方转成错误提示。
 *
 * `sizeBytes` 已知且达到并行阈值时改走有界并发 Range 分段（`api/download.ts`，ADR-009），
 * 服务端不支持 Range / 返回非 206 时内部自动回退本单流路径。
 */
export async function downloadProxyObject(
  req: Omit<ProxyRequest, 'mode' | 'signal'>,
  filename: string,
  sizeBytes?: number,
): Promise<void> {
  if (sizeBytes !== undefined && sizeBytes >= PARALLEL_DOWNLOAD_MIN_BYTES) {
    await downloadObjectToDisk({ ...req, size: sizeBytes, filename })
    return
  }
  const blob = await fetchProxyBlob({ ...req, mode: 'download' })
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
