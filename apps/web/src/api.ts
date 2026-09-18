import type {
  Account,
  AccountInput,
  BucketInfo,
  CorsRule,
  JobRecord,
  LifecycleRule,
  ListObjectsResponse,
  ListVersionsResponse,
  MigrationResult,
  ObjectMeta,
  PresignResponse,
  ServerProfile,
  BucketItem,
} from './types'
import { t } from './i18n'

// 安全默认：Bearer Token 存 sessionStorage（关标签即清），避免 XSS 持久窃取。
// 显式开启「跨会话保留」后才落 localStorage（`s3c_token_persistent = '1'`）。
//
// 迁移策略：从 v1.0.0-rc2 之前的版本升级时，旧版 token 在 localStorage；
// 首次读取若 sessionStorage 为空但 localStorage 有值，自动迁移到 sessionStorage
// 并清空 localStorage 中的副本（一次性、不破坏既有用户）。
const LS_SERVERS = 's3c.servers'
const LS_ACTIVE = 's3c.activeServerId'
const LS_BASE = 's3c.apiBase'
const LS_TOKEN = 's3c.token'
const LS_TOKEN_PERSISTENT = 's3c_token_persistent'

function tokenPersistent(): boolean {
  try {
    return localStorage.getItem(LS_TOKEN_PERSISTENT) === '1'
  } catch {
    return false
  }
}

function readToken(): string {
  // 1) 优先 sessionStorage
  try {
    const fromSession = sessionStorage.getItem(LS_TOKEN)
    if (fromSession) return fromSession
  } catch {
    /* ignore */
  }
  // 2) 用户显式开启「跨会话保留」→ localStorage
  if (tokenPersistent()) {
    try {
      return localStorage.getItem(LS_TOKEN) ?? ''
    } catch {
      return ''
    }
  }
  // 3) 一次性迁移：旧版本（默认 localStorage）的 token 升级时迁到 sessionStorage。
  let fromLS = ''
  try {
    fromLS = localStorage.getItem(LS_TOKEN) ?? ''
  } catch {
    return ''
  }
  if (fromLS) {
    try {
      sessionStorage.setItem(LS_TOKEN, fromLS)
      localStorage.removeItem(LS_TOKEN)
    } catch {
      /* ignore */
    }
  }
  return fromLS
}

function writeToken(v: string) {
  const trimmed = v.trim()
  // 总是清空 sessionStorage（无论持久化模式如何，保证两存储不会同时存在）
  try {
    if (trimmed) sessionStorage.setItem(LS_TOKEN, trimmed)
    else sessionStorage.removeItem(LS_TOKEN)
  } catch {
    /* ignore */
  }
  if (tokenPersistent()) {
    // 持久化模式：同时写 localStorage
    try {
      if (trimmed) localStorage.setItem(LS_TOKEN, trimmed)
      else localStorage.removeItem(LS_TOKEN)
    } catch {
      /* ignore */
    }
  } else {
    // 默认：清空 localStorage 的残留（避免历史值被静默恢复）
    try {
      localStorage.removeItem(LS_TOKEN)
    } catch {
      /* ignore */
    }
  }
}

function isTauri(): boolean {
  // Tauri 在 window 上挂的内部标志只用于探测；用 unknown 避免 any 逃逸类型检查。
  const w = window as unknown as {
    __TAURI_INTERNALS__?: unknown
    __TAURI__?: unknown
  }
  return (
    !!w.__TAURI_INTERNALS__ ||
    !!w.__TAURI__ ||
    navigator.userAgent.toLowerCase().includes('tauri') ||
    /^(?:tauri|app\.tauri)(?:\.localhost)?$/.test(location.hostname)
  )
}

function defaultBase(): string {
  return isTauri() ? 'http://127.0.0.1:8080' : ''
}

function newId(): string {
  return crypto.randomUUID?.() ?? `s-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

// 多服务器 token 存储：`s3c.servers` 只存 {id,name,base}（不落 token，见 writeServers），
// 每个服务器的 token 按 `s3c.token.<serverId>` 独立存储（默认 sessionStorage；
// 显式「跨会话保留」时同时写 localStorage），与单服务器 `s3c.token` 策略一致。
function serverTokenKey(id: string): string {
  return `${LS_TOKEN}.${id}`
}

function readServerToken(id: string): string {
  try {
    const fromSession = sessionStorage.getItem(serverTokenKey(id))
    if (fromSession) return fromSession
  } catch {
    /* ignore */
  }
  if (tokenPersistent()) {
    try {
      return localStorage.getItem(serverTokenKey(id)) ?? ''
    } catch {
      return ''
    }
  }
  return ''
}

function writeServerToken(id: string, token: string) {
  const trimmed = token.trim()
  try {
    if (trimmed) sessionStorage.setItem(serverTokenKey(id), trimmed)
    else sessionStorage.removeItem(serverTokenKey(id))
  } catch {
    /* ignore */
  }
  if (tokenPersistent()) {
    try {
      if (trimmed) localStorage.setItem(serverTokenKey(id), trimmed)
      else localStorage.removeItem(serverTokenKey(id))
    } catch {
      /* ignore */
    }
  } else {
    try {
      localStorage.removeItem(serverTokenKey(id))
    } catch {
      /* ignore */
    }
  }
}

function clearServerToken(id: string) {
  try {
    sessionStorage.removeItem(serverTokenKey(id))
  } catch {
    /* ignore */
  }
  try {
    localStorage.removeItem(serverTokenKey(id))
  } catch {
    /* ignore */
  }
}

// StoredServerProfile 是 s3c.servers 的实际落盘形态：**不含 token**（安全策略）。
type StoredServerProfile = Omit<ServerProfile, 'token'>

function readServers(): ServerProfile[] {
  // 原始存储条目（含旧版本内嵌 token，供一次性迁移）；StoredServerProfile 不含 token。
  let rawList: Array<Partial<StoredServerProfile> & { token?: string }> = []
  let hasValidArray = false
  try {
    const raw = localStorage.getItem(LS_SERVERS)
    if (raw !== null) {
      const list = JSON.parse(raw) as unknown as Array<Partial<StoredServerProfile> & { token?: string }>
      if (Array.isArray(list)) {
        rawList = list
        hasValidArray = true
      }
    }
  } catch {
    /* ignore */
  }
  // 兼容迁移：旧版本 s3c.servers 内嵌 token → 迁移到 per-server 存储并重写 localStorage（一次性）。
  let migrated = false
  const hydrated: ServerProfile[] = rawList.map((s) => {
    const id = s.id ?? ''
    const legacy = (s as { token?: string }).token ?? ''
    if (legacy) {
      writeServerToken(id, legacy)
      migrated = true
    }
    return { id, name: s.name ?? '', base: s.base ?? '', token: readServerToken(id) }
  })
  if (migrated) {
    writeServers(hydrated)
    // 保持旧行为：若活动服务器带 token 且全局 token 为空（如 sessionStorage 已清），应用之，
    // 使升级后当前会话仍能直接请求。
    const activeId = (() => {
      try {
        return localStorage.getItem(LS_ACTIVE) ?? ''
      } catch {
        return ''
      }
    })()
    const active = hydrated.find((s) => s.id === activeId)
    if (active && active.token && !readToken()) {
      applyProfile(active)
    }
  }
  // 已存在合法数组键（即便为空数组）：保持原样，不自动创建默认 server（activeServerId 会回退到 ''）。
  if (hasValidArray) return hydrated
  // 首次 / 键损坏：默认一条
  const p: ServerProfile = {
    id: newId(),
    name: isTauri() ? t('server.localBackend') : t('server.sameOriginDefault'),
    base: defaultBase(),
    token: '',
  }
  writeServers([p])
  localStorage.setItem(LS_ACTIVE, p.id)
  applyProfile(p)
  return [p]
}

function writeServers(list: ServerProfile[]) {
  // 落盘时剥离 token：s3c.servers 永远不包含密钥。
  const stored: StoredServerProfile[] = list.map(({ id, name, base }) => ({ id, name, base }))
  localStorage.setItem(LS_SERVERS, JSON.stringify(stored))
}

function applyProfile(p: ServerProfile) {
  localStorage.setItem(LS_BASE, (p.base || '').replace(/\/+$/, ''))
  writeToken(p.token || '')
  localStorage.setItem(LS_ACTIVE, p.id)
}

export const api = {
  get base(): string {
    const stored = localStorage.getItem(LS_BASE)
    return (stored !== null ? stored : defaultBase()).replace(/\/+$/, '')
  },
  set base(v: string) {
    localStorage.setItem(LS_BASE, v.replace(/\/+$/, ''))
  },
  get token(): string {
    return readToken()
  },
  set token(v: string) {
    writeToken(v)
  },
  /** Token 是否「跨会话保留」（开启后写 localStorage）。默认 false = 仅 sessionStorage。 */
  get isTokenPersistent(): boolean {
    return tokenPersistent()
  },
  /**
   * 切换「跨会话保留」：开启时把 token 同时落到 localStorage；
   * 关闭时从 localStorage 清除并保留在 sessionStorage。
   */
  setTokenPersistent(enabled: boolean) {
    const current = readToken()
    try {
      if (enabled) localStorage.setItem(LS_TOKEN_PERSISTENT, '1')
      else localStorage.removeItem(LS_TOKEN_PERSISTENT)
    } catch {
      /* ignore */
    }
    writeToken(current)
  },
  get isTauri(): boolean {
    return isTauri()
  },

  listServers(): ServerProfile[] {
    return readServers()
  },

  activeServerId(): string {
    const id = localStorage.getItem(LS_ACTIVE) ?? ''
    const list = readServers()
    if (list.find((s) => s.id === id)) return id
    return list[0]?.id ?? ''
  },

  getActiveServer(): ServerProfile | undefined {
    const id = this.activeServerId()
    return this.listServers().find((s) => s.id === id)
  },

  /** 设为当前生效并同步 base/token */
  selectServer(id: string): ServerProfile | undefined {
    const p = this.listServers().find((s) => s.id === id)
    if (!p) return undefined
    applyProfile(p)
    return p
  },

  upsertServer(input: { id?: string; name: string; base: string; token: string }): ServerProfile {
    const list = readServers()
    const base = (input.base || '').replace(/\/+$/, '')
    const token = (input.token || '').trim()
    const name = input.name.trim() || (base || t('server.sameOriginShort'))
    if (input.id) {
      const i = list.findIndex((s) => s.id === input.id)
      if (i >= 0) {
        const updated: ServerProfile = { ...list[i], name, base, token }
        // token 不随 servers 列表落 localStorage；按服务器 id 独立存 sessionStorage/持久化。
        writeServerToken(input.id, token)
        list[i] = { ...updated, token: readServerToken(input.id) }
        writeServers(list)
        if (this.activeServerId() === input.id) applyProfile(list[i])
        return list[i]
      }
    }
    const p: ServerProfile = { id: newId(), name, base, token }
    // 同上：token 独立存储，servers 列表不含密钥。
    writeServerToken(p.id, token)
    p.token = readServerToken(p.id)
    list.push(p)
    writeServers(list)
    return p
  },

  deleteServer(id: string): void {
    // 删除前判定：被删服务器是否为当前生效项（删除后 activeServerId 已无法回退到它）。
    const wasActive = this.activeServerId() === id
    clearServerToken(id)
    let list = readServers().filter((s) => s.id !== id)
    if (!list.length) {
      const fresh: ServerProfile = { id: newId(), name: isTauri() ? t('server.localBackend') : t('server.sameOriginDefault'), base: defaultBase(), token: '' }
      list = [fresh]
    }
    writeServers(list)
    if (wasActive) {
      applyProfile(list[0])
    }
  },
}

async function request<T>(path: string, opts: RequestInit = {}): Promise<T> {
  const headers: Record<string, string> = { ...((opts.headers as Record<string, string>) ?? {}) }
  if (opts.body != null) headers['Content-Type'] = 'application/json'
  if (api.token) headers['Authorization'] = `Bearer ${api.token}`
  const res = await fetch(api.base + path, { headers, ...opts })
  if (!res.ok) {
    let msg = res.statusText
    try {
      const j = await res.json()
      if (j?.error) msg = j.error
    } catch {
      /* ignore */
    }
    throw new Error(`${res.status} ${msg}`)
  }
  return res.json() as Promise<T>
}

/** 原始 Response（流式下载用）；失败时解析 JSON error。 */
export async function requestResponse(path: string, opts: RequestInit = {}): Promise<Response> {
  const headers: Record<string, string> = { ...((opts.headers as Record<string, string>) ?? {}) }
  if (opts.body != null) headers['Content-Type'] = 'application/json'
  if (api.token) headers['Authorization'] = `Bearer ${api.token}`
  const res = await fetch(api.base + path, { headers, ...opts })
  if (!res.ok) {
    let msg = res.statusText
    try {
      const j = await res.json()
      if (j?.error) msg = j.error
    } catch {
      /* ignore */
    }
    throw new Error(`${res.status} ${msg}`)
  }
  return res
}

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

export const s3api = {

  listAccounts: () => request<{ accounts: Account[] }>('/api/accounts'),
  createAccount: (a: AccountInput) => request<Account>('/api/accounts', { method: 'POST', body: JSON.stringify(a) }),
  updateAccount: (id: string, a: Partial<AccountInput>) =>
    request<Account>(`/api/accounts/${id}`, { method: 'PUT', body: JSON.stringify(a) }),
  deleteAccount: (id: string) => request<{ deleted: string }>(`/api/accounts/${id}`, { method: 'DELETE' }),
  testAccount: (id: string) => request<{ ok: boolean; bucket: string; error?: string }>(`/api/accounts/${id}/test`, { method: 'POST' }),
  listBuckets: (id: string) => request<{ buckets: BucketItem[] }>(`/api/accounts/${id}/buckets`),
  createBucket: (id: string, body: { name: string; region?: string; acl?: string }) =>
    request<{ created: string; region: string; acl: string }>(`/api/accounts/${id}/bucket`, { method: 'POST', body: JSON.stringify(body) }),
  deleteBucket: (id: string, name: string) =>
    request<{ deleted: string }>(`/api/accounts/${id}/bucket?name=${encodeURIComponent(name)}`, { method: 'DELETE' }),
  previewBuckets: (a: AccountInput) =>
    request<{ buckets: BucketItem[] }>('/api/accounts/preview-buckets', { method: 'POST', body: JSON.stringify(a) }),

  listObjects: (id: string, q: Record<string, string>, opts: { signal?: AbortSignal } = {}) => {
    const qs = new URLSearchParams(q).toString()
    return request<ListObjectsResponse>(`/api/accounts/${id}/objects?${qs}`, { signal: opts.signal })
  },
  headObject: (id: string, q: Record<string, string>) => {
    const qs = new URLSearchParams(q).toString()
    return request<ObjectMeta>(`/api/accounts/${id}/head?${qs}`)
  },
  mkdirObject: (id: string, body: { bucket?: string; key: string }) =>
    request<{ created: string; bucket: string }>(`/api/accounts/${id}/mkdir`, { method: 'POST', body: JSON.stringify(body) }),
  renameObject: (id: string, body: { bucket?: string; key: string; newKey: string; newBucket?: string }) =>
    request<{ renamed: string }>(`/api/accounts/${id}/rename`, { method: 'POST', body: JSON.stringify(body) }),
  copyObject: (id: string, body: { bucket?: string; key: string; newKey: string; newBucket?: string }) =>
    request<{ copied: string; bucket: string }>(`/api/accounts/${id}/copy-object`, { method: 'POST', body: JSON.stringify(body) }),
  copyFiles: (id: string, body: { bucket?: string; targetBucket?: string; targetPrefix?: string; keys: string[]; deleteSource?: boolean }) =>
    request<{ copied: number; failed: number; lastError?: string; failedKeys?: string[] }>(`/api/accounts/${id}/copy-objects`, { method: 'POST', body: JSON.stringify(body) }),
  /** 异步批量复制/移动：立即返回 jobId，进度走 migrate jobs SSE；`deleteSource` 由服务端在任务内删源。 */
  copyFilesAsync: (id: string, body: { bucket?: string; targetBucket?: string; targetPrefix?: string; keys: string[]; deleteSource?: boolean }) =>
    request<{ jobId: string; total: number }>(`/api/accounts/${id}/copy-objects/async`, { method: 'POST', body: JSON.stringify(body) }),
  getObjectAcl: (id: string, query: { bucket?: string; key: string }) => {
    const qs = new URLSearchParams()
    qs.set('key', query.key)
    if (query.bucket) qs.set('bucket', query.bucket)
    return request<{
      bucket: string
      key: string
      owner?: string
      public: boolean
      grants: { grantee: string; permission: string }[]
      url: string
    }>(`/api/accounts/${id}/object-acl?${qs.toString()}`)
  },
  putObjectAcl: (id: string, body: { bucket?: string; key: string; acl: string }) =>
    request<{ acl: string }>(`/api/accounts/${id}/object-acl`, { method: 'PUT', body: JSON.stringify(body) }),
  getObjectTags: (id: string, query: { bucket?: string; key: string }) => {
    const qs = new URLSearchParams()
    qs.set('key', query.key)
    if (query.bucket) qs.set('bucket', query.bucket)
    return request<{ tags: { key: string; value: string }[] }>(`/api/accounts/${id}/object-tags?${qs.toString()}`)
  },
  putObjectTags: (id: string, body: { bucket?: string; key: string; tags: { key: string; value: string }[] }) =>
    request<{ tags: { key: string; value: string }[] }>(`/api/accounts/${id}/object-tags`, { method: 'PUT', body: JSON.stringify(body) }),
  getBucketInfo: (id: string, bucket?: string) =>
    request<BucketInfo>(`/api/accounts/${id}/bucket-info?bucket=${encodeURIComponent(bucket ?? '')}`),
  putBucketVersioning: (id: string, body: { bucket?: string; status: 'Enabled' | 'Suspended' }) =>
    request<{ versioning: string }>(`/api/accounts/${id}/bucket-versioning`, { method: 'PUT', body: JSON.stringify(body) }),
  getBucketEncryption: (id: string, bucket?: string) =>
    request<{ bucket: string; configured: boolean; algorithm: string; kmsKeyId: string; bucketKeyEnabled: boolean }>(`/api/accounts/${id}/bucket/encryption?bucket=${encodeURIComponent(bucket ?? '')}`),
  putBucketEncryption: (id: string, body: { bucket?: string; algorithm: string; kmsKeyId?: string; bucketKeyEnabled?: boolean }) =>
    request<{ configured: boolean; algorithm: string }>(`/api/accounts/${id}/bucket/encryption`, { method: 'PUT', body: JSON.stringify(body) }),
  deleteBucketEncryption: (id: string, bucket?: string) =>
    request<{ deleted: string }>(`/api/accounts/${id}/bucket/encryption?bucket=${encodeURIComponent(bucket ?? '')}`, { method: 'DELETE' }),
  getBucketCors: (id: string, bucket?: string) =>
    request<{ bucket: string; rules: CorsRule[] }>(`/api/accounts/${id}/bucket/cors?bucket=${encodeURIComponent(bucket ?? '')}`),
  putBucketCors: (id: string, body: { bucket?: string; rules: CorsRule[] }) =>
    request<{ updated: number; deleted?: string }>(`/api/accounts/${id}/bucket/cors`, { method: 'PUT', body: JSON.stringify(body) }),
  deleteBucketCors: (id: string, bucket?: string) =>
    request<{ deleted: string }>(`/api/accounts/${id}/bucket/cors?bucket=${encodeURIComponent(bucket ?? '')}`, { method: 'DELETE' }),
  getBucketWebsite: (id: string, bucket?: string) =>
    request<{ bucket: string; configured: boolean; indexDocument: string; errorDocument: string; redirectAllRequestsTo: string }>(`/api/accounts/${id}/bucket/website?bucket=${encodeURIComponent(bucket ?? '')}`),
  putBucketWebsite: (id: string, body: { bucket?: string; indexDocument?: string; errorDocument?: string; redirectAllRequestsTo?: string }) =>
    request<{ configured: boolean }>(`/api/accounts/${id}/bucket/website`, { method: 'PUT', body: JSON.stringify(body) }),
  deleteBucketWebsite: (id: string, bucket?: string) =>
    request<{ deleted: string }>(`/api/accounts/${id}/bucket/website?bucket=${encodeURIComponent(bucket ?? '')}`, { method: 'DELETE' }),
  getBucketPolicy: (id: string, bucket?: string) =>
    request<{ bucket: string; configured: boolean; policy: string }>(`/api/accounts/${id}/bucket/policy?bucket=${encodeURIComponent(bucket ?? '')}`),
  putBucketPolicy: (id: string, body: { bucket?: string; policy: string }) =>
    request<{ configured: boolean; deleted?: string }>(`/api/accounts/${id}/bucket/policy`, { method: 'PUT', body: JSON.stringify(body) }),
  deleteBucketPolicy: (id: string, bucket?: string) =>
    request<{ deleted: string }>(`/api/accounts/${id}/bucket/policy?bucket=${encodeURIComponent(bucket ?? '')}`, { method: 'DELETE' }),
  getBucketTags: (id: string, bucket?: string) =>
    request<{ bucket: string; tags: { key: string; value: string }[] }>(`/api/accounts/${id}/bucket/tags?bucket=${encodeURIComponent(bucket ?? '')}`),
  putBucketTags: (id: string, body: { bucket?: string; tags: { key: string; value: string }[] }) =>
    request<{ updated: number; deleted?: string }>(`/api/accounts/${id}/bucket/tags`, { method: 'PUT', body: JSON.stringify(body) }),
  deleteBucketTags: (id: string, bucket?: string) =>
    request<{ deleted: string }>(`/api/accounts/${id}/bucket/tags?bucket=${encodeURIComponent(bucket ?? '')}`, { method: 'DELETE' }),
  listVersions: (id: string, q: { bucket?: string; prefix?: string; keyMarker?: string; versionIdMarker?: string }) => {
    const qs = new URLSearchParams()
    if (q.bucket) qs.set('bucket', q.bucket)
    if (q.prefix) qs.set('prefix', q.prefix)
    if (q.keyMarker) qs.set('keyMarker', q.keyMarker)
    if (q.versionIdMarker) qs.set('versionIdMarker', q.versionIdMarker)
    return request<ListVersionsResponse>(`/api/accounts/${id}/versions?${qs.toString()}`)
  },
  deleteObjectVersion: (id: string, q: { bucket?: string; key: string; versionId: string }) => {
    const qs = new URLSearchParams()
    qs.set('key', q.key)
    qs.set('versionId', q.versionId)
    if (q.bucket) qs.set('bucket', q.bucket)
    return request<{ deleted: string; versionId: string }>(`/api/accounts/${id}/version?${qs.toString()}`, { method: 'DELETE' })
  },
  restoreObjectVersion: (id: string, body: { bucket?: string; key: string; versionId: string }) =>
    request<{ restored: string; versionId: string }>(`/api/accounts/${id}/version/restore`, { method: 'POST', body: JSON.stringify(body) }),
  restoreDeleteMarker: (id: string, body: { bucket?: string; key: string; versionId: string }) =>
    request<{ restored: string; versionId: string }>(`/api/accounts/${id}/delete-marker/restore`, { method: 'POST', body: JSON.stringify(body) }),
  listTrash: (id: string, q: { bucket?: string; prefix?: string; keyMarker?: string; versionIdMarker?: string; maxKeys?: number }) => {
    const qs = new URLSearchParams()
    if (q.bucket) qs.set('bucket', q.bucket)
    if (q.prefix) qs.set('prefix', q.prefix)
    if (q.keyMarker) qs.set('keyMarker', q.keyMarker)
    if (q.versionIdMarker) qs.set('versionIdMarker', q.versionIdMarker)
    if (q.maxKeys) qs.set('maxKeys', String(q.maxKeys))
    return request<{ deleteMarkers: { key: string; versionId: string; isLatest: boolean; lastModified: string }[]; isTruncated: boolean; nextKeyMarker: string; nextVersionIdMarker: string }>(`/api/accounts/${id}/trash?${qs.toString()}`)
  },
  purgeTrashObject: (id: string, body: { bucket?: string; key: string }) =>
    request<{ purged: string; deleted: number }>(`/api/accounts/${id}/trash/purge`, { method: 'POST', body: JSON.stringify(body) }),
  changeStorageClass: (id: string, body: { bucket?: string; key: string; versionId?: string; storageClass: string }) =>
    request<{ changed: string; versionId: string; storageClass: string }>(`/api/accounts/${id}/storage-class`, { method: 'POST', body: JSON.stringify(body) }),
  setHeaders: (id: string, body: { bucket?: string; key: string; contentType?: string; metadata?: Record<string, string> }) =>
    request<{ updated: string }>(`/api/accounts/${id}/set-headers`, { method: 'POST', body: JSON.stringify(body) }),
  getLifecycle: (id: string, bucket?: string) =>
    request<{ rules: LifecycleRule[] }>(`/api/accounts/${id}/lifecycle?bucket=${encodeURIComponent(bucket ?? '')}`),
  putLifecycle: (id: string, body: { bucket?: string; rules: LifecycleRule[] }) =>
    request<{ updated: number }>(`/api/accounts/${id}/lifecycle`, { method: 'PUT', body: JSON.stringify(body) }),
  presign: (id: string, body: { method?: string; key: string; bucket?: string; versionId?: string; expiresIn?: number }) =>
    request<PresignResponse>(`/api/accounts/${id}/presign`, { method: 'POST', body: JSON.stringify(body) }),
  multipartInit: (id: string, body: { bucket?: string; key: string; contentType?: string }) =>
    request<{ uploadId: string; key: string; bucket: string }>(`/api/accounts/${id}/multipart/init`, { method: 'POST', body: JSON.stringify(body) }),
  multipartPart: (id: string, body: { bucket?: string; key: string; uploadId: string; partNumber: number; expiresIn?: number }) =>
    request<{ partNumber: number; url: string; expiresIn: number }>(`/api/accounts/${id}/multipart/part`, { method: 'POST', body: JSON.stringify(body) }),
  multipartComplete: (id: string, body: { bucket?: string; key: string; uploadId: string; parts: { partNumber: number; etag: string }[] }) =>
    request<{ completed: string }>(`/api/accounts/${id}/multipart/complete`, { method: 'POST', body: JSON.stringify(body) }),
  multipartAbort: (id: string, body: { bucket?: string; key: string; uploadId: string }) =>
    request<{ aborted: boolean }>(`/api/accounts/${id}/multipart/abort`, { method: 'POST', body: JSON.stringify(body) }),
  deleteObjects: (id: string, body: { bucket?: string; keys: string[] }) =>
    request<{ deleted: number }>(`/api/accounts/${id}/delete`, { method: 'POST', body: JSON.stringify(body) }),
  deletePrefix: (id: string, body: { bucket?: string; prefix: string }) =>
    request<{ deleted: number; truncated: boolean }>(`/api/accounts/${id}/delete-prefix`, { method: 'POST', body: JSON.stringify(body) }),
  /** 异步前缀删除：立即返回 jobId，进度走 migrate jobs SSE。 */
  deletePrefixAsync: (id: string, body: { bucket?: string; prefix: string }) =>
    request<{ jobId: string; total: number; truncated?: boolean }>(`/api/accounts/${id}/delete-prefix/async`, { method: 'POST', body: JSON.stringify(body) }),
  copyPrefix: (id: string, body: { bucket?: string; prefix: string; targetBucket?: string; targetPrefix: string }) =>
    request<{ copied: number; failed: number; total: number; lastError?: string }>(`/api/accounts/${id}/copy-prefix`, { method: 'POST', body: JSON.stringify(body) }),
  copyPrefixAsync: (id: string, body: { bucket?: string; prefix: string; targetBucket?: string; targetPrefix: string }) =>
    request<{ jobId: string; total: number; truncated?: boolean }>(`/api/accounts/${id}/copy-prefix/async`, { method: 'POST', body: JSON.stringify(body) }),
  /** 流式 ZIP 落盘（优先 File System Access API）。 */
  downloadZipToDisk: (id: string, body: { bucket?: string; keys: string[] }, suggestedName?: string) =>
    downloadZipToDisk(id, body, suggestedName),
  migrate: (body: {
    sourceAccountId: string
    sourceBucket?: string
    sourceKeys: string[]
    targetAccountId: string
    targetBucket?: string
    targetPrefix?: string
  }) => request<MigrationResult>('/api/migrate', { method: 'POST', body: JSON.stringify(body) }),

  migrateAsync: (body: {
    sourceAccountId: string
    sourceBucket?: string
    sourceKeys: string[]
    targetAccountId: string
    targetBucket?: string
    targetPrefix?: string
  }) =>
    request<{ jobId: string; total: number }>('/api/migrate/async', { method: 'POST', body: JSON.stringify(body) }),

  /** 异步任务清单（含进程重启后中断的任务，用于「未完成任务」对账视图）。 */
  migrateJobs: () => request<{ jobs: JobRecord[] }>('/api/migrate/jobs'),

  migrateJobStatus: (jobId: string) =>
    request<{ jobId: string; done: boolean; progress: MigrateProgress; result?: MigrationResult }>(
      `/api/migrate/jobs/${encodeURIComponent(jobId)}`,
    ),

  migrateJobCancel: (jobId: string) =>
    request<{ jobId: string; cancelled: boolean; done?: boolean }>(
      `/api/migrate/jobs/${encodeURIComponent(jobId)}/cancel`,
      { method: 'POST' },
    ),

  /** 增量同步：按 ETag / size+mtime / always 比对，仅复制差异对象。 */
  migrateSync: (body: {
    sourceAccountId: string
    sourceBucket?: string
    sourcePrefix?: string
    targetAccountId: string
    targetBucket?: string
    targetPrefix?: string
    mode?: 'etag' | 'size_mtime' | 'always'
  }) =>
    request<{
      scanned: number
      skipped: number
      copied: number
      failed: number
      failedKeys?: string[]
      lastError?: string
    }>('/api/migrate/sync', { method: 'POST', body: JSON.stringify(body) }),
}

export interface MigrateProgress {
  done: number
  total: number
  migrated: number
  failed: number
  key?: string
  error?: string
  status?: string
}

// 流 EOF 后回读 job 状态的重试参数：任务仍在运行则轮询直到终态（防 Promise 永久悬挂）；
// 连续多次回读失败视为网络不可用，快速 onError 让调用方复位（三个调用方均无自身超时兜底）。
const JOB_STATUS_POLL_MS = 30_000 // 任务仍在运行时的最长轮询
const JOB_STATUS_POLL_INTERVAL_MS = 500
const JOB_STATUS_MAX_CONSECUTIVE_FAILURES = 3

/** 订阅迁移 SSE 进度（fetch 流式，支持 Bearer）。返回 abort 函数。 */
export function subscribeMigrateEvents(
  jobId: string,
  onProgress: (p: MigrateProgress) => void,
  onError: (err: Error) => void,
): () => void {
  const ctrl = new AbortController()
  const headers: Record<string, string> = { Accept: 'text/event-stream' }
  if (api.token) headers['Authorization'] = `Bearer ${api.token}`
  ;(async () => {
    let lastStatus: string | undefined
    try {
      const res = await fetch(`${api.base}/api/migrate/jobs/${encodeURIComponent(jobId)}/events`, {
        headers,
        signal: ctrl.signal,
      })
      if (!res.ok || !res.body) {
        throw new Error(`${res.status} ${res.statusText}`)
      }
      const reader = res.body.getReader()
      const dec = new TextDecoder()
      let buf = ''
      for (;;) {
        const { done, value } = await reader.read()
        if (done) break
        buf += dec.decode(value, { stream: true })
        let idx: number
        while ((idx = buf.indexOf('\n\n')) >= 0) {
          const block = buf.slice(0, idx)
          buf = buf.slice(idx + 2)
          let eventName = ''
          for (const line of block.split('\n')) {
            if (line.startsWith('event:')) {
              eventName = line.slice(6).trim()
            } else if (line.startsWith('data:')) {
              // 心跳 ping 忽略
              if (eventName === 'ping') continue
              const raw = line.startsWith('data: ') ? line.slice(6) : line.slice(5).trimStart()
              try {
                const p = JSON.parse(raw) as MigrateProgress
                if (p.status) lastStatus = p.status
                onProgress(p)
              } catch {
                /* ignore partial */
              }
            }
          }
        }
      }
      // 流正常 EOF 但未收到终态：回读并轮询 job 状态，直到终态或超时，
      // 避免 UI 永久卡在「迁移中」（旧实现只回读一次：回读失败或任务仍未完成即悬挂，
      // 三个调用方 ctxDeleteFolder / DestDialog / MigratePanel 均无超时兜底）。
      if (lastStatus !== 'done' && lastStatus !== 'cancelled' && !ctrl.signal.aborted) {
        const deadline = Date.now() + JOB_STATUS_POLL_MS
        let consecutiveFailures = 0
        for (;;) {
          if (ctrl.signal.aborted) break
          try {
            const st = await s3api.migrateJobStatus(jobId)
            consecutiveFailures = 0
            if (st.done) {
              // 终态：无论回读进度是否自带 status，都合成终态事件让调用方 resolve。
              const s = st.progress.status
              onProgress({ ...st.progress, status: s === 'cancelled' ? 'cancelled' : 'done' })
              break
            }
            // 任务仍在运行：上报一次当前进度（非终态），继续轮询。
            onProgress({ ...st.progress, status: st.progress.status || 'running' })
          } catch {
            // 网络抖动：连续失败达上限即判网络不可用，快速 onError（不再死等 deadline）。
            consecutiveFailures++
            if (consecutiveFailures >= JOB_STATUS_MAX_CONSECUTIVE_FAILURES) {
              onError(new Error(`migrate job ${jobId} status unavailable`))
              break
            }
          }
          if (Date.now() >= deadline) {
            onError(new Error(`migrate job ${jobId} status timeout`))
            break
          }
          await new Promise((r) => setTimeout(r, JOB_STATUS_POLL_INTERVAL_MS))
        }
      }
    } catch (e) {
      if (!ctrl.signal.aborted) onError(e instanceof Error ? e : new Error(String(e)))
    }
  })()
  return () => ctrl.abort()
}

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
