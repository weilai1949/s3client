import { reactive, ref } from 'vue'
import type { Account } from './types'
import type { TabKey } from './router'

/** 记住上次选中的账号（互联网应用习惯：刷新/重开后恢复现场）。 */
const LS_CURRENT_ACCOUNT = 's3c.currentAccountId'

export const state = reactive({
  accounts: [] as Account[],
  currentAccountId: '',
})

/**
 * 数据世代：切换后端服务器后自增。
 *
 * App 用它给面板容器换 key，等于把各面板的缓存一次性丢弃（桶、对象、选中项都属于
 * 旧服务器）。这取代了此前 `window.location.reload()` 的做法——整页刷新虽然也能清干净，
 * 但会连窗口里与服务器无关的东西一起丢（未保存的表单、上传队列、滚动位置），
 * 且要重新走一遍首屏加载。
 */
export const serverEpoch = ref(0)

export function bumpServerEpoch() {
  serverEpoch.value++
}

/** 跨面板跳转信号：任何组件可请求切换到某 tab（App 监听执行）。
 *  '' 表示「无目标」（seq 只用于触发 watcher，消费方按空值守卫跳过）。 */
export const tabRequest = reactive<{ tab: TabKey | ''; seq: number }>({ tab: '', seq: 0 })

export function requestTab(tab: TabKey) {
  tabRequest.tab = tab
  tabRequest.seq++
}

/** 请求账号面板自动打开「新增登录」表单。 */
export const accountFormRequest = reactive({ seq: 0 })

export function requestAccountForm() {
  accountFormRequest.seq++
}

export function currentAccount(): Account | undefined {
  return state.accounts.find((a) => a.id === state.currentAccountId)
}

/** 切换当前账号并持久化，下次打开自动恢复。
 *  存储不可用（隐私模式/配额异常）只影响「记住上次选择」，不得中断切换本身。 */
export function selectAccount(id: string) {
  state.currentAccountId = id
  try {
    if (id) localStorage.setItem(LS_CURRENT_ACCOUNT, id)
    else localStorage.removeItem(LS_CURRENT_ACCOUNT)
  } catch {
    /* ignore */
  }
}

/** 读取上次选中的账号 id（可能已不存在，由调用方校验）。 */
export function rememberedAccountId(): string {
  try {
    return localStorage.getItem(LS_CURRENT_ACCOUNT) ?? ''
  } catch {
    return ''
  }
}

/* ---- 全局 Toast 反馈（支持操作按钮） ---- */

/* ---- 全局活动中心（长任务进度：直传 / 迁移） ---- */

/** 一条长任务的进度快照。按 `key` 唯一：同一入口复用一条，避免进度堆积成噪声。 */
export interface Activity {
  /** 唯一键（`upload:objects` / `upload:panel` / `migrate`）。 */
  key: string
  /** 面向用户的任务名（已本地化，渲染时直接展示）。 */
  label: string
  /** 已完成条目数。 */
  done: number
  /** 条目总数。 */
  total: number
  /** 整体百分比（0-100）。 */
  pct: number
}

/**
 * 活动表：面板把在跑的耗时任务登记进来，顶栏角标据此展示「切走也看得见」的进度。
 *
 * 为什么放 store：上传/迁移的状态机挂在**面板组件**上（KeepAlive 缓存），面板被切走时
 * 组件仍活着、进度仍在推进，但用户看不到任何东西——只有把快照提升到全局，
 * 顶栏才能在整个应用范围内展示它（KNOWN_ISSUES 记录的「长任务进度不可见」）。
 */
export const activities = reactive<Record<string, Activity>>({})

/** 上报（或更新）一条活动进度。 */
export function reportActivity(a: Activity) {
  activities[a.key] = a
}

/** 移除一条活动（任务结束 / 无可展示进度时调用，避免角标长期停留）。 */
export function clearActivity(key: string) {
  delete activities[key]
}

export interface ToastItem {
  id: number
  kind: 'ok' | 'err'
  text: string
  /** 可选动作按钮（例如「查看对象」跳转）。 */
  action?: { label: string; onClick: () => void }
}

export const toasts = reactive<ToastItem[]>([])

/** 同时可见的 toast 上限：满了先挤掉最旧一条，新反馈不会被旧反馈挡在后面。 */
const MAX_TOASTS = 4
/** 基准停留时长；长文案按字数加时（见 toastDuration），上限 +3s。 */
const TOAST_MS = 3600

/**
 * 文案越长停留越久：3600ms 够读一句短提示，但迁移/批量删除这类长句（含文件名与计数）
 * 常常读到一半就消失，用户只能靠重试复现。每超出 60 字一档、每字 +40ms，封顶 +3s。
 */
function toastDuration(text: string): number {
  return TOAST_MS + Math.min(3000, Math.max(0, text.length - 60) * 40)
}

/** 自动消失计时器的可暂停状态：hover / 焦点停留时冻结，移开后按剩余时长续跑。 */
interface ToastTimer {
  handle?: ReturnType<typeof setTimeout>
  remainingMs: number
  startedAt: number
}
let toastSeq = 0
const toastTimers = new Map<number, ToastTimer>()

function armTimer(id: number, ms: number) {
  const t: ToastTimer = { handle: setTimeout(() => dismissToast(id), ms), remainingMs: ms, startedAt: Date.now() }
  toastTimers.set(id, t)
}

export function toast(text: string, kind: 'ok' | 'err' = 'ok', action?: ToastItem['action']): number {
  const id = ++toastSeq
  toasts.push({ id, kind, text, action })
  while (toasts.length > MAX_TOASTS) dismissToast(toasts[0].id)
  armTimer(id, toastDuration(text))
  return id
}

/**
 * 悬停 / 聚焦暂停自动消失：鼠标停在 toast 上（或用 Tab 把焦点移到它的按钮上）时冻结
 * 倒计时——否则用户正准备点「撤销 / 查看对象」，提示就没了。
 * 传入 id 不存在（已 dismiss / 未持计时器）时静默忽略。
 */
export function setToastHovered(id: number, hovered: boolean) {
  const t = toastTimers.get(id)
  if (!t) return
  if (hovered) {
    if (t.handle === undefined) return
    clearTimeout(t.handle)
    t.handle = undefined
    t.remainingMs = Math.max(0, t.remainingMs - (Date.now() - t.startedAt))
  } else {
    if (t.handle !== undefined) return
    t.startedAt = Date.now()
    t.handle = setTimeout(() => dismissToast(id), t.remainingMs)
  }
}

/** 就地更新一条 toast 的文本（进度型反馈复用同一条，避免灌满 toast 栈）。
 *  模块私有：唯一入口是 createProgressToast 的进度节流路径。
 *  返回是否命中：id 已随自动 dismiss 消失时返回 false、不碰 toast 栈，由调用方改发新条
 *  —— 存在性判断只此一处，调用方不再重复查一遍（原来两处各查一次）。 */
function updateToast(id: number, text: string): boolean {
  const i = toasts.findIndex((t) => t.id === id)
  if (i < 0) return false
  toasts[i].text = text
  return true
}

/**
 * 进度型 toast 工厂（SSE 迁移/删除进度）：至多每 throttleMs 一条；
 * 未过期直接就地更新同一条文本（已自动消失则新发）。终态反馈请直接调 toast()，不经过节流。
 */
export function createProgressToast(throttleMs = 500): (text: string) => void {
  let id: number | null = null
  let lastAt = 0
  return (text) => {
    const now = Date.now()
    if (id !== null && now - lastAt < throttleMs) return
    if (id === null || !updateToast(id, text)) id = toast(text)
    lastAt = now
  }
}

export function dismissToast(id: number) {
  const timer = toastTimers.get(id)
  if (timer !== undefined) {
    if (timer.handle !== undefined) clearTimeout(timer.handle)
    toastTimers.delete(id)
  }
  const i = toasts.findIndex((t) => t.id === id)
  if (i >= 0) toasts.splice(i, 1)
}
