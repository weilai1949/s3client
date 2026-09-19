import type { ServerProfile } from '../types'
import { t } from '../i18n'

/**
 * 浏览器侧凭据与多服务器 profile 存储（从 `api.ts` 拆出，行为不变）。
 *
 * 安全默认：Bearer Token 存 sessionStorage（关标签即清），避免 XSS 持久窃取。
 * 显式开启「跨会话保留」后才落 localStorage（`s3c_token_persistent = '1'`）。
 *
 * 迁移策略：从 v1.0.0-rc2 之前的版本升级时，旧版 token 在 localStorage；
 * 首次读取若 sessionStorage 为空但 localStorage 有值，自动迁移到 sessionStorage
 * 并清空 localStorage 中的副本（一次性、不破坏既有用户）。
 *
 * 本模块**不依赖 http/endpoints**，是依赖图的最底层，避免与 `request()` 形成环。
 */
const LS_SERVERS = 's3c.servers'
const LS_ACTIVE = 's3c.activeServerId'
const LS_BASE = 's3c.apiBase'
const LS_TOKEN = 's3c.token'
const LS_TOKEN_PERSISTENT = 's3c_token_persistent'

export function tokenPersistent(): boolean {
  try {
    return localStorage.getItem(LS_TOKEN_PERSISTENT) === '1'
  } catch {
    return false
  }
}

export function readToken(): string {
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

export function writeToken(v: string) {
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

export function isTauri(): boolean {
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

export function defaultBase(): string {
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

/** 归一后的存储条目：字段齐备（缺失一律补空串），供迁移与 hydration 直接消费。 */
interface StoredServerEntry {
  id: string
  name: string
  base: string
  token: string
}

/** 单条服务器记录的形状校验与归一。
 *
 * `s3c.servers` 是外部可写的 localStorage 键（旧版本、手工编辑、其它页面都可能写坏），
 * 因此读取路径必须把每个元素当成**不可信输入**：`[null]`/`[1]`/`[{}]` 这类条目此前会
 * 抛 TypeError（整站白屏）或退化成 id:'' 的幽灵 server（见 review-2026-09-19.md §F1）。
 * 返回 null 表示该条无法寻址（没有 id），调用方应丢弃并修复存储。
 */
function sanitizeServer(s: unknown): StoredServerEntry | null {
  if (typeof s !== 'object' || s === null) return null
  const o = s as Record<string, unknown>
  const str = (v: unknown): string => (typeof v === 'string' ? v : '')
  const id = str(o.id)
  if (!id) return null
  return { id, name: str(o.name), base: str(o.base), token: str(o.token) }
}

function readServers(): ServerProfile[] {
  // 原始存储条目（含旧版本内嵌 token，供一次性迁移）；落盘形态不含 token。
  const rawList: StoredServerEntry[] = []
  let hasValidArray = false
  let repaired = false
  try {
    const raw = localStorage.getItem(LS_SERVERS)
    if (raw !== null) {
      const list = JSON.parse(raw) as unknown
      if (Array.isArray(list)) {
        hasValidArray = true
        for (const item of list) {
          const s = sanitizeServer(item)
          if (s) rawList.push(s)
          else repaired = true
        }
      }
    }
  } catch {
    /* ignore */
  }
  // 兼容迁移：旧版本 s3c.servers 内嵌 token → 迁移到 per-server 存储并重写 localStorage（一次性）。
  let migrated = false
  const hydrated: ServerProfile[] = rawList.map((s) => {
    if (s.token) {
      writeServerToken(s.id, s.token)
      migrated = true
    }
    return { id: s.id, name: s.name, base: s.base, token: readServerToken(s.id) }
  })
  // 迁移与「坏条目修复」都要回写：把清洗后的清单落盘，避免坏数据被反复读取。
  if (migrated || repaired) {
    writeServers(hydrated)
  }
  if (migrated) {
    // 保持旧行为：若活动服务器带 token 且全局 token 为空（如 sessionStorage 已清），应用之，
    // 使升级后当前会话仍能直接请求。
    const activeId = readActiveServerId()
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

function readActiveServerId(): string {
  try {
    return localStorage.getItem(LS_ACTIVE) ?? ''
  } catch {
    return ''
  }
}

/** 当前 API base（去掉尾部斜杠）；未设置时回退到默认（Tauri → 本地后端）。 */
export function getBase(): string {
  const stored = localStorage.getItem(LS_BASE)
  return (stored !== null ? stored : defaultBase()).replace(/\/+$/, '')
}

export function setBase(v: string) {
  localStorage.setItem(LS_BASE, v.replace(/\/+$/, ''))
}

/** 切换「跨会话保留」：开启时把 token 同时落到 localStorage；关闭时清除 localStorage 副本。 */
export function setTokenPersistent(enabled: boolean) {
  const current = readToken()
  try {
    if (enabled) localStorage.setItem(LS_TOKEN_PERSISTENT, '1')
    else localStorage.removeItem(LS_TOKEN_PERSISTENT)
  } catch {
    /* ignore */
  }
  writeToken(current)
}

export function listServers(): ServerProfile[] {
  return readServers()
}

export function activeServerId(): string {
  const id = readActiveServerId()
  const list = readServers()
  if (list.find((s) => s.id === id)) return id
  return list[0]?.id ?? ''
}

/** 当前生效的服务器配置。
 *
 * 这是**渲染期安全**访问器：`App.vue` 在模板里直接调用它（顶栏显示服务器名），
 * 一旦抛出就打断渲染 → 整站白屏，而唯一能修复存储的 Server 面板也随之不可达
 * （见 review-2026-09-19.md §F1）。因此这里兜底为 undefined，让 UI 至少可用。
 * 可达路径：浏览器禁用存储时 `writeServers` 的 setItem 会抛 SecurityError。
 */
export function getActiveServer(): ServerProfile | undefined {
  try {
    const id = activeServerId()
    return listServers().find((s) => s.id === id)
  } catch {
    return undefined
  }
}

/** 设为当前生效并同步 base/token */
export function selectServer(id: string): ServerProfile | undefined {
  const p = listServers().find((s) => s.id === id)
  if (!p) return undefined
  applyProfile(p)
  return p
}

export function upsertServer(input: { id?: string; name: string; base: string; token: string }): ServerProfile {
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
      if (activeServerId() === input.id) applyProfile(list[i])
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
}

export function deleteServer(id: string): void {
  // 删除前判定：被删服务器是否为当前生效项（删除后 activeServerId 已无法回退到它）。
  const wasActive = activeServerId() === id
  clearServerToken(id)
  let list = readServers().filter((s) => s.id !== id)
  if (!list.length) {
    const fresh: ServerProfile = {
      id: newId(),
      name: isTauri() ? t('server.localBackend') : t('server.sameOriginDefault'),
      base: defaultBase(),
      token: '',
    }
    list = [fresh]
  }
  writeServers(list)
  if (wasActive) {
    applyProfile(list[0])
  }
}
