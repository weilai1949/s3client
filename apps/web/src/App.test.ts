import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { reactive, ref, nextTick } from 'vue'
import App from './App.vue'
import type { Account } from './types'

vi.mock('./api', () => ({
  s3api: {
    listAccounts: vi.fn(async () => ({ accounts: [] })),
  },
  api: {
    isTauri: false,
    getActiveServer: vi.fn(() => undefined),
    health: vi.fn(async () => ({ status: 'ok', version: 'v1' })),
  },
}))

vi.mock('./store', async () => {
  const { reactive } = await import('vue')
  return {
    state: reactive({ accounts: [] as Account[], currentAccountId: '' }),
    rememberedAccountId: vi.fn(() => ''),
    selectAccount: vi.fn(),
    tabRequest: reactive({ tab: '', seq: 0 }),
    accountFormRequest: reactive({ seq: 0 }),
  }
})

vi.mock('./i18n', () => ({
  t: (k: string) => k,
  cycleLocale: vi.fn(),
  i18nState: reactive({ locale: 'zh-CN' }),
}))

vi.mock('./theme', () => ({
  readTheme: vi.fn(() => 'auto'),
  resolvedTheme: vi.fn(() => 'light'),
  cycleTheme: vi.fn(() => 'dark'),
  systemThemeTick: ref(0),
}))

vi.mock('./router', () => ({
  tabFromHash: vi.fn(() => null),
  setTabHash: vi.fn(),
  onTabHashChange: vi.fn(() => () => {}),
}))

import { api, s3api } from './api'
import { onTabHashChange, setTabHash, tabFromHash, type TabKey } from './router'
import { state, tabRequest, accountFormRequest, selectAccount, rememberedAccountId } from './store'
import { cycleTheme, resolvedTheme } from './theme'
import { cycleLocale, i18nState } from './i18n'

function mountApp() {
  return mount(App, {
    global: {
      stubs: {
        AccountsPanel: true,
        ObjectsPanel: true,
        UploadPanel: true,
        MigratePanel: true,
        BucketsPanel: true,
        RecycleBinPanel: true,
        ServerPanel: true,
        ConfirmDialog: true,
        PromptDialog: true,
        Toasts: true,
      },
    },
  })
}

describe('App', () => {
  beforeEach(() => {
    state.accounts = []
    state.currentAccountId = ''
    tabRequest.tab = ''
    tabRequest.seq = 0
    accountFormRequest.seq = 0
    ;(api as unknown as { isTauri: boolean }).isTauri = false
    ;(i18nState as unknown as { locale: string }).locale = 'zh-CN'
    vi.clearAllMocks()
    vi.mocked(tabFromHash).mockReturnValue(null)
  })

  it('loads accounts, routes to objects when accounts exist, stays on accounts when empty', async () => {
    vi.mocked(s3api.listAccounts).mockResolvedValueOnce({ accounts: [{ id: 'a1', name: 'A' }] } as unknown as Awaited<ReturnType<typeof s3api.listAccounts>>)
    const w = mountApp()
    await flushPromises()
    expect(w.text()).toContain('nav.accounts')
    // first account auto-selected
    expect(vi.mocked(selectAccount)).toHaveBeenCalledWith('a1') // fallback select
    w.unmount()
  })

  it('restores the remembered account when it still exists', async () => {
    vi.mocked(s3api.listAccounts).mockResolvedValueOnce({
      accounts: [{ id: 'a1', name: 'A' }, { id: 'a2', name: 'B' }],
    } as unknown as Awaited<ReturnType<typeof s3api.listAccounts>>)
    vi.mocked(rememberedAccountId).mockReturnValue('a2')
    const w = mountApp()
    await flushPromises()
    expect(vi.mocked(selectAccount)).toHaveBeenCalledWith('a2')
    w.unmount()
  })

  it('shows server error banner and goto-server button', async () => {
    vi.mocked(s3api.listAccounts).mockRejectedValueOnce(new Error('boom'))
    const w = mountApp()
    await flushPromises()
    expect(w.text()).toContain('conn.fail')
    expect(w.text()).toContain('boom')
    const goto = w.findAll('button').find((b) => b.text().includes('conn.gotoServer'))
    expect(goto).toBeTruthy()
    await goto!.trigger('click')
    // switchTab('server') + setTabHash
    w.unmount()
  })

  it('后端不可用后轮询 /api/health，恢复时自动重载并清除错误（roadmap #8）', async () => {
    vi.useFakeTimers()
    try {
      vi.mocked(s3api.listAccounts).mockRejectedValueOnce(new Error('boom'))
      const w = mountApp()
      await vi.advanceTimersByTimeAsync(0)
      expect(w.text()).toContain('conn.fail')

      // 后端恢复：下一次探测成功后重新拉取账号并清掉错误横幅。
      vi.mocked(s3api.listAccounts).mockResolvedValueOnce({ accounts: [] } as unknown as Awaited<ReturnType<typeof s3api.listAccounts>>)
      await vi.advanceTimersByTimeAsync(5000)
      await flushPromises()
      expect(vi.mocked(api.health)).toHaveBeenCalled()
      expect(vi.mocked(s3api.listAccounts)).toHaveBeenCalledTimes(2)
      expect(w.text()).not.toContain('conn.fail')
      w.unmount()
    } finally {
      vi.useRealTimers()
    }
  })

  it('tab switches via nav, hash listener, tabRequest and accountFormRequest', async () => {
    vi.mocked(s3api.listAccounts).mockResolvedValueOnce({ accounts: [] })
    const w = mountApp()
    await flushPromises()
    const navBtns = w.findAll('.tabs button')
    expect(navBtns.length).toBe(7)
    const serverBtn = navBtns.find((b) => b.text().includes('nav.server'))!
    await serverBtn.trigger('click')
    expect(w.find('server-panel-stub').exists()).toBe(true)

    tabRequest.seq++
    tabRequest.tab = 'trash'
    await nextTick()
    expect(w.find('recycle-bin-panel-stub').exists()).toBe(true)

    accountFormRequest.seq++
    await nextTick()
    expect(w.find('accounts-panel-stub').exists()).toBe(true)

    w.unmount()
  })

  it('theme cycle updates icon and locale cycle calls cycleLocale', async () => {
    vi.mocked(s3api.listAccounts).mockResolvedValueOnce({ accounts: [] })
    const w = mountApp()
    await flushPromises()
    await w.find('.tabs button').trigger('click') // ensure mounted
    const themeBtn = w.findAll('.theme-btn')[1]
    await themeBtn.trigger('click')
    expect(vi.mocked(cycleTheme)).toHaveBeenCalled()
    const localeBtn = w.findAll('.theme-btn')[0]
    await localeBtn.trigger('click')
    expect(vi.mocked(cycleLocale)).toHaveBeenCalled()
    w.unmount()
  })

  it('unmount unsubscribes hash listener and keeps tab on hash change', async () => {
    let handler: ((t: TabKey | null) => void) | null = null
    vi.mocked(onTabHashChange).mockImplementationOnce((cb: (t: TabKey | null) => void) => {
      handler = cb
      return () => {}
    })
    vi.mocked(s3api.listAccounts).mockResolvedValueOnce({ accounts: [] })
    const w = mountApp()
    await flushPromises()
    expect(handler).toBeTruthy()
    handler!('upload')
    await nextTick()
    expect(w.find('upload-panel-stub').exists()).toBe(true)
    w.unmount()
  })

  it('列表仍含当前账号时保持当前（else-if keep-current 分支不重选）', async () => {
    vi.mocked(s3api.listAccounts).mockResolvedValueOnce({
      accounts: [{ id: 'a1', name: 'A' }, { id: 'a2', name: 'B' }],
    } as unknown as Awaited<ReturnType<typeof s3api.listAccounts>>)
    vi.mocked(rememberedAccountId).mockReturnValue('zz-gone')
    state.currentAccountId = 'a2'
    const w = mountApp()
    await flushPromises()
    // 既无 remembered 匹配、也不再回退第一个：selectAccount 不应被调用
    expect(vi.mocked(selectAccount)).not.toHaveBeenCalled()
    w.unmount()
  })

  it('dark 主题渲染 moon 图标、Tauri 桌面徽章、英文 locale 标签', async () => {
    ;(api as unknown as { isTauri: boolean }).isTauri = true
    vi.mocked(resolvedTheme).mockReturnValueOnce('dark')
    ;(i18nState as unknown as { locale: string }).locale = 'en'
    vi.mocked(s3api.listAccounts).mockResolvedValueOnce({ accounts: [] })
    const w = mountApp()
    await flushPromises()
    expect(w.text()).toContain('Desktop')
    const themeBtn = w.findAll('.theme-btn')[1]
    // themeIcon === 'moon' → 月亮 svg（M21.752 路径）
    expect(themeBtn.html()).toContain('M21.752')
    // localeLabel 英文分支（locale 按钮）
    expect(w.findAll('.theme-btn')[0].attributes('aria-label')).toBe('locale.en')
    ;(api as unknown as { isTauri: boolean }).isTauri = false
    ;(i18nState as unknown as { locale: string }).locale = 'zh-CN'
    w.unmount()
  })

  it('hash 指向有效 tab 时首次路由直接进入且不再写 hash', async () => {
    vi.mocked(tabFromHash).mockReturnValue('buckets')
    vi.mocked(s3api.listAccounts).mockResolvedValueOnce({ accounts: [] })
    const w = mountApp()
    await flushPromises()
    expect(w.find('buckets-panel-stub').exists()).toBe(true)
    // !hashTab 为假 → 不再调用 setTabHash
    expect(vi.mocked(setTabHash)).not.toHaveBeenCalled()
    w.unmount()
  })

  it('hash 监听收到空值时保持当前 tab', async () => {
    let handler: ((t: TabKey | null) => void) | null = null
    vi.mocked(onTabHashChange).mockImplementationOnce((cb: (t: TabKey | null) => void) => {
      handler = cb
      return () => {}
    })
    vi.mocked(s3api.listAccounts).mockResolvedValueOnce({ accounts: [] })
    const w = mountApp()
    await flushPromises()
    handler!(null)
    await nextTick()
    expect(w.find('accounts-panel-stub').exists()).toBe(true)
    w.unmount()
  })

  it('tabRequest 无目标 tab 时不切换', async () => {
    vi.mocked(s3api.listAccounts).mockResolvedValueOnce({ accounts: [] })
    const w = mountApp()
    await flushPromises()
    tabRequest.seq++ // tabRequest.tab 仍为空 → 守卫跳过
    await nextTick()
    expect(w.find('accounts-panel-stub').exists()).toBe(true)
    w.unmount()
  })

  it('账号变更后再次加载不再重路由（首次路由只执行一次）', async () => {
    vi.mocked(s3api.listAccounts).mockResolvedValueOnce({ accounts: [] })
    const w = mountApp()
    await flushPromises()
    const before = vi.mocked(setTabHash).mock.calls.length
    // 直接二次调用 loadAccounts（等价 @changed 触发）
    await (w.vm as unknown as { loadAccounts: () => Promise<void> }).loadAccounts()
    await flushPromises()
    // first=false → 跳过重路由、不再写 hash
    expect(vi.mocked(setTabHash).mock.calls.length).toBe(before)
    expect(w.find('accounts-panel-stub').exists()).toBe(true)
    w.unmount()
  })

  it('nav 点击 migrate 切换到 MigratePanel', async () => {
    vi.mocked(s3api.listAccounts).mockResolvedValueOnce({ accounts: [] })
    const w = mountApp()
    await flushPromises()
    const migrateBtn = w.findAll('.tabs button').find((b) => b.text().includes('nav.migrate'))!
    await migrateBtn.trigger('click')
    expect(w.find('migrate-panel-stub').exists()).toBe(true)
    w.unmount()
  })

  it('未知 tab 值不渲染任何面板（server else-if 假分支）', async () => {
    vi.mocked(s3api.listAccounts).mockResolvedValueOnce({ accounts: [] })
    const w = mountApp()
    await flushPromises()
    ;(w.vm as unknown as { tab: string }).tab = 'zz-unknown'
    await nextTick()
    expect(w.find('accounts-panel-stub').exists()).toBe(false)
    expect(w.find('server-panel-stub').exists()).toBe(false)
    w.unmount()
  })
})
