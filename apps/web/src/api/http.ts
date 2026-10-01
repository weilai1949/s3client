import { getBase, readToken } from './storage'
import { operations, type OperationId } from './operations'

/** 传输层：统一注入 base / Bearer / JSON Content-Type，并把错误响应归一为 Error。 */

/**
 * `opPath('deleteBucket', …)` 的实参形状：**由 spec 生成的 path 模板反推**——
 * 模板无占位符时整段参数可省略，有占位符时每个 `{name}` 都必须给（少给 = 编译错，
 * 而不是运行时拼出一个 `…/{id}` 的坏 URL）。
 */
type PathArgs<Id extends OperationId> = (typeof operations)[Id]['params'] extends readonly []
  ? [params?: undefined]
  : [params: { [K in (typeof operations)[Id]['params'][number]]: string }]

/**
 * 按 **spec 生成的** path 模板拼 URL（ROADMAP §三 #10）。
 *
 * `path` 逐字来自 `docs/api/openapi.json`，前端不再手写 URL 字面量——spec 改路径时
 * 生成物同步变、前端跟着变；spec 删操作时 `OperationId` 收窄，`vue-tsc` 直接红。
 * 占位符值统一 `encodeURIComponent`。
 */
export function opPath<Id extends OperationId>(id: Id, ...args: PathArgs<Id>): string {
  const params = (args[0] ?? {}) as Record<string, string>
  let out: string = operations[id].path
  for (const name of operations[id].params) {
    out = out.split(`{${name}}`).join(encodeURIComponent(params[name]))
  }
  return out
}

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
