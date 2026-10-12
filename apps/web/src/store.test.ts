import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { bumpServerEpoch, serverEpoch } from './store'
import { createProgressToast, setToastHovered, toasts, requestTab, tabRequest, requestAccountForm, accountFormRequest, currentAccount, selectAccount, rememberedAccountId, toast, dismissToast, state } from './store'
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

  it('窗口过后就地更新同一条 toast，而不是新发一条（私有 updateToast 的「命中已有 id → 就地改文本」路径）', () => {
    const progress = createProgressToast()
    progress('1/10')
    const firstId = toasts[0].id
    vi.advanceTimersByTime(500)
    progress('2/10')
    expect(toasts).toHaveLength(1)
    expect(toasts[0].id).toBe(firstId)
    expect(toasts[0].text).toBe('2/10')
  })

  it('toast 已自动消失后再次触发会发出新的一条（私有 updateToast 的「id 不存在 → 不碰旧栈」路径）', () => {
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

describe('serverEpoch', () => {
  it('bumpServerEpoch 自增（切换服务器后 App 据此丢弃各面板的旧服务器缓存）', () => {
    const before = serverEpoch.value
    bumpServerEpoch()
    expect(serverEpoch.value).toBe(before + 1)
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

  it('存储不可用（写入抛异常）时切换账号仍生效，不抛出', () => {
    const ls = {
      getItem: () => null,
      setItem: () => { throw new DOMException('QuotaExceededError', 'QuotaExceededError') },
      removeItem: () => { throw new DOMException('SecurityError', 'SecurityError') },
    }
    Object.defineProperty(globalThis, 'localStorage', { value: ls, configurable: true, writable: true })
    expect(() => selectAccount('acc9')).not.toThrow()
    expect(state.currentAccountId).toBe('acc9')
    expect(() => selectAccount('')).not.toThrow()
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

  it('存储不可用（读取抛异常）时降级为空串而不是中断启动', () => {
    const ls = {
      getItem: () => { throw new DOMException('SecurityError', 'SecurityError') },
    }
    Object.defineProperty(globalThis, 'localStorage', { value: ls, configurable: true, writable: true })
    expect(() => rememberedAccountId()).not.toThrow()
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

describe('dismissToast', () => {
  it('removes toast', () => {
    toasts.splice(0, toasts.length)
    const id = toast('msg')
    dismissToast(id)
    expect(toasts).toHaveLength(0)
  })

  it('no-op for unknown id (timer undefined + index < 0 两侧)', () => {
    toasts.splice(0, toasts.length)
    dismissToast(424242)
    expect(toasts).toHaveLength(0)
  })
})

describe('toast 栈上限 / 长文案时长 / 悬停暂停', () => {
  beforeEach(() => {
    toasts.splice(0, toasts.length)
    vi.useFakeTimers()
  })
  afterEach(() => {
    vi.useRealTimers()
    toasts.splice(0, toasts.length)
  })

  it('同时可见上限 4 条：超出时挤掉最旧一条（新反馈不被挡）', () => {
    for (let i = 1; i <= 6; i++) toast(`msg-${i}`)
    expect(toasts.map((t) => t.text)).toEqual(['msg-3', 'msg-4', 'msg-5', 'msg-6'])
  })

  it('长文案停留更久（封顶 +3s），短文案按基准 3600ms', () => {
    toast('short')
    vi.advanceTimersByTime(3599)
    expect(toasts).toHaveLength(1)
    vi.advanceTimersByTime(2)
    expect(toasts).toHaveLength(0)

    toast('x'.repeat(200)) // 3600 + 封顶 3000 = 6600ms
    vi.advanceTimersByTime(5000)
    expect(toasts).toHaveLength(1)
    vi.advanceTimersByTime(1700)
    expect(toasts).toHaveLength(0)
  })

  it('悬停冻结倒计时，移开后按剩余时长续跑', () => {
    const id = toast('hover me')
    vi.advanceTimersByTime(2000) // 剩余 1600ms
    setToastHovered(id, true)
    vi.advanceTimersByTime(60_000)
    expect(toasts).toHaveLength(1)
    setToastHovered(id, false)
    vi.advanceTimersByTime(1599)
    expect(toasts).toHaveLength(1)
    vi.advanceTimersByTime(2)
    expect(toasts).toHaveLength(0)
  })

  it('重复进入 / 重复离开悬停都是 no-op；未知 id 静默忽略', () => {
    const id = toast('x')
    setToastHovered(id, true)
    setToastHovered(id, true) // 已暂停：不重复扣减剩余时长
    setToastHovered(id, false)
    setToastHovered(id, false) // 已在跑：不重复挂计时器
    vi.advanceTimersByTime(3601)
    expect(toasts).toHaveLength(0)
    expect(() => setToastHovered(424242, true)).not.toThrow()
  })

  it('暂停中手动 dismiss：计时器一并清掉，不留悬挂回调', () => {
    const id = toast('paused')
    setToastHovered(id, true)
    dismissToast(id)
    vi.advanceTimersByTime(10_000)
    expect(toasts).toHaveLength(0)
  })
})
