import { getBase, readToken } from './storage'

/** 传输层：统一注入 base / Bearer / JSON Content-Type，并把错误响应归一为 Error。 */

/**
 * 合成 fetch init：默认 header 与调用方 header **合并**，调用方同名项优先，
 * 但 Authorization / Content-Type 等默认值不会被整体覆盖掉。
 *
 * 拆分前是 `{ headers, ...opts }`（opts 在后）——调用方一旦传 headers，合成的
 * Authorization/Content-Type 会被整个替换掉（review §F10，潜伏陷阱）。
 */
function buildInit(opts: RequestInit): RequestInit {
  const headers: Record<string, string> = {}
  if (opts.body != null) headers['Content-Type'] = 'application/json'
  const token = readToken()
  if (token) headers['Authorization'] = `Bearer ${token}`
  Object.assign(headers, (opts.headers as Record<string, string>) ?? {})
  return { ...opts, headers }
}

async function toError(res: Response): Promise<Error> {
  let msg = res.statusText
  try {
    const j = await res.json()
    if (j?.error) msg = j.error
  } catch {
    /* ignore */
  }
  return new Error(`${res.status} ${msg}`)
}

export async function request<T>(path: string, opts: RequestInit = {}): Promise<T> {
  const res = await fetch(getBase() + path, buildInit(opts))
  if (!res.ok) throw await toError(res)
  return res.json() as Promise<T>
}

/** 原始 Response（流式下载用）；失败时解析 JSON error。 */
export async function requestResponse(path: string, opts: RequestInit = {}): Promise<Response> {
  const res = await fetch(getBase() + path, buildInit(opts))
  if (!res.ok) throw await toError(res)
  return res
}
