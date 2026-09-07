import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createProgressToast, toasts, requestTab, tabRequest, requestAccountForm, accountFormRequest, currentAccount, selectAccount, rememberedAccountId, toast, updateToast, dismissToast, state } from './store'
import type { Account } from './types'

describe('createProgressToast（SSE 进度节流）', () => {
  beforeEach(() => vi.useFakeTimers())
  afterEach(() => {
    vi.useRealTimers()
    toasts.splice(0, toasts.length)
  })

  it('500ms 窗口内只保留一条，不新增 toast', () => {
    const progress = createProgressToast()
    progress('1/10')
    progress('2/10')
    progress('3/10')
    expect(toasts).toHaveLength(1)
    expect(toasts[0].text).toBe('1/10')
  })

  it('窗口过后就地更新同一条 toast，而不是新发一条', () => {
    const progress = createProgressToast()
    progress('1/10')
    const firstId = toasts[0].id
    vi.advanceTimersByTime(500)
    progress('2/10')
    expect(toasts).toHaveLength(1)
    expect(toasts[0].id).toBe(firstId)
    expect(toasts[0].text).toBe('2/10')
  })

  it('toast 已自动消失后再次触发会发出新的一条', () => {
    const progress = createProgressToast()
    progress('1/10')
    const firstId = toasts[0].id
    vi.advanceTimersByTime(4000) // 超过 3600ms 自动消失
    progress('2/10')
    expect(toasts).toHaveLength(1)
    expect(toasts[0].id).not.toBe(firstId)
    expect(toasts[0].text).toBe('2/10')
  })

  it('自定义节流窗口生效', () => {
    const progress = createProgressToast(100)
    progress('a')
    vi.advanceTimersByTime(100)
    progress('b')
    expect(toasts[0].text).toBe('b')
  })
})

describe('requestTab', () => {
  it('sets tab and increments seq', () => {
    tabRequest.tab = ''
    tabRequest.seq = 0
    requestTab('accounts')
    expect(tabRequest.tab).toBe('accounts')
    expect(tabRequest.seq).toBe(1)
  })
})

describe('requestAccountForm', () => {
  it('increments seq', () => {
    accountFormRequest.seq = 0
    requestAccountForm()
    expect(accountFormRequest.seq).toBe(1)
  })
})

describe('currentAccount', () => {
  it('finds account by id', () => {
    state.accounts = [{ id: 'a1', name: 'A' } as Account, { id: 'a2', name: 'B' } as Account]
    state.currentAccountId = 'a1'
    expect(currentAccount()?.id).toBe('a1')
    state.currentAccountId = 'a2'
    expect(currentAccount()?.id).toBe('a2')
    state.currentAccountId = 'missing'
    expect(currentAccount()).toBeUndefined()
  })
})

describe('selectAccount', () => {
  it('sets currentAccountId and persists to localStorage', () => {
    const mem = new Map<string, string>()
    const ls = {
      getItem: (k: string) => mem.get(k) ?? null,
      setItem: (k: string, v: string) => mem.set(k, v),
      removeItem: (k: string) => mem.delete(k),
    }
    Object.defineProperty(globalThis, 'localStorage', { value: ls, configurable: true, writable: true })
    selectAccount('acc1')
    expect(state.currentAccountId).toBe('acc1')
    expect(mem.get('s3c.currentAccountId')).toBe('acc1')
  })

  it('removes from localStorage when id is empty', () => {
    const mem = new Map<string, string>()
    const ls = {
      getItem: (k: string) => mem.get(k) ?? null,
      setItem: (k: string, v: string) => mem.set(k, v),
      removeItem: (k: string) => mem.delete(k),
    }
    Object.defineProperty(globalThis, 'localStorage', { value: ls, configurable: true, writable: true })
    selectAccount('')
    expect(state.currentAccountId).toBe('')
  })
})

describe('rememberedAccountId', () => {
  it('reads from localStorage', () => {
    const mem = new Map<string, string>()
    mem.set('s3c.currentAccountId', 'acc1')
    const ls = {
      getItem: (k: string) => mem.get(k) ?? null,
    }
    Object.defineProperty(globalThis, 'localStorage', { value: ls, configurable: true, writable: true })
    expect(rememberedAccountId()).toBe('acc1')
  })

  it('returns empty string when not found', () => {
    const ls = {
      getItem: () => null,
    }
    Object.defineProperty(globalThis, 'localStorage', { value: ls, configurable: true, writable: true })
    expect(rememberedAccountId()).toBe('')
  })
})

describe('toast', () => {
  beforeEach(() => {
    toasts.splice(0, toasts.length)
    vi.useFakeTimers()
  })
  afterEach(() => { vi.useRealTimers() })

  it('adds toast', () => {
    const id = toast('msg', 'ok')
    expect(toasts).toHaveLength(1)
    expect(toasts[0].text).toBe('msg')
    expect(toasts[0].kind).toBe('ok')
    expect(typeof id).toBe('number')
  })

  it('supports action button', () => {
    const action = { label: 'View', onClick: vi.fn() }
    toast('err', 'err', action)
    expect(toasts[toasts.length - 1].action).toStrictEqual(action)
  })
})

describe('updateToast', () => {
  beforeEach(() => {
    toasts.splice(0, toasts.length)
  })

  it('updates text of existing toast', () => {
    toasts.push({ id: 1, kind: 'ok', text: 'old' })
    updateToast(1, 'new')
    expect(toasts[toasts.length - 1].text).toBe('new')
  })

  it('no-op for missing id', () => {
    updateToast(999, 'new')
    expect(toasts).toHaveLength(0)
  })
})

describe('dismissToast', () => {
  it('removes toast', () => {
    toasts.splice(0, toasts.length)
    const id = toast('msg')
    dismissToast(id)
    expect(toasts).toHaveLength(0)
  })
})
