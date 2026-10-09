import { reactive, readonly } from 'vue'
import { accountsMessages } from './messages/accounts'
import { bucketsMessages } from './messages/buckets'
import { commonMessages } from './messages/common'
import { objectDialogsMessages } from './messages/objectDialogs'
import { objectsMessages } from './messages/objects'
import { schedulesMessages } from './messages/schedules'
import { storageReportMessages } from './messages/storageReport'
import { uploadMessages } from './messages/upload'
import type { Locale, MessageBundle } from './messages/types'

export type { Locale } from './messages/types'

const LS_LOCALE = 's3c.locale'

function readLocale(): Locale {
  try {
    const v = localStorage.getItem(LS_LOCALE) as Locale
    if (v === 'zh-CN' || v === 'en-US') return v
  } catch {
    /* vitest / SSR */
  }
  return 'zh-CN'
}

/**
 * 把界面语言同步到 `<html lang>`。
 *
 * 为什么必须做：`index.html` 只能硬编码一个初始 `lang="zh-CN"`，而用户可能持久化过 `en-US`。
 * 此前没有任何运行时赋值，于是切到英文后文档语言标记仍是 `zh-CN`——屏幕阅读器会按中文读音
 * 念英文内容，浏览器的翻译提示也会误判（KNOWN_ISSUES #67①，由 `i18n/index.test.ts` 钉住）。
 */
function applyDocumentLang(loc: Locale): void {
  document.documentElement.lang = loc
}

const state = reactive({ locale: readLocale() })

// 首屏即同步：让持久化的语言在渲染前就反映到 <html lang>，而不是等用户手动切一次。
applyDocumentLang(state.locale)

/** 按命名空间拆分的消息模块（见 messages/ 目录），在此汇总为运行时字典。 */
function mergeBundles(...bundles: MessageBundle[]): Record<Locale, Record<string, string>> {
  return {
    'zh-CN': Object.assign({}, ...bundles.map((b) => b['zh-CN'])),
    'en-US': Object.assign({}, ...bundles.map((b) => b['en-US'])),
  }
}

const messages = mergeBundles(
  commonMessages,
  objectsMessages,
  objectDialogsMessages,
  bucketsMessages,
  uploadMessages,
  accountsMessages,
  schedulesMessages,
  storageReportMessages,
)

export function t(key: string): string {
  return messages[state.locale][key] ?? messages['zh-CN'][key] ?? key
}

/** Simple `{name}` placeholder replacement for wired UI strings. */
export function tf(key: string, vars: Record<string, string | number>): string {
  let s = t(key)
  for (const [k, v] of Object.entries(vars)) {
    s = s.replaceAll(`{${k}}`, String(v))
  }
  return s
}

export function locale(): Locale {
  return state.locale
}

export function setLocale(loc: Locale) {
  state.locale = loc
  applyDocumentLang(loc)
  try {
    localStorage.setItem(LS_LOCALE, loc)
  } catch {
    /* ignore */
  }
}

export function cycleLocale(): Locale {
  const order: Locale[] = ['zh-CN', 'en-US']
  const i = order.indexOf(state.locale)
  const next = order[(i + 1) % order.length]
  setLocale(next)
  return next
}

export const i18nState = readonly(state)
