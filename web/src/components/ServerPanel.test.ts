import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import ServerPanel from './ServerPanel.vue'
import { api } from '../api'
import { toast } from '../store'
import { confirmDialog } from '../confirm'
import type { ServerProfile } from '../types'

vi.mock('../api', () => ({
  api: {
    isTokenPersistent: false,
    listServers: vi.fn(() => []),
    activeServerId: vi.fn(() => ''),
    setTokenPersistent: vi.fn(),
    upsertServer: vi.fn(),
    deleteServer: vi.fn(),
    selectServer: vi.fn(),
  },
}))

vi.mock('../store', () => ({
  toast: vi.fn(),
}))

vi.mock('../confirm', () => ({
  confirmDialog: vi.fn(async () => true),
}))

vi.mock('../i18n', () => ({
  t: (k: string) => k,
  tf: (k: string) => k,
}))

const s1: ServerProfile = { id: 's1', name: 'S1', base: 'http://x:9000/', token: 't1' }
const s2: ServerProfile = { id: 's2', name: 'S2', base: '', token: '' }

const modalStub = {
  name: 'ModalDialog',
  props: ['open', 'title', 'width'],
  template: '<div v-if="open" class="modal-stub"><slot /></div>',
}

let fetchMock: ReturnType<typeof vi.fn>
let reloadSpy: ReturnType<typeof vi.fn>
let origReload: () => void

function okResponse(body: unknown) {
  return { ok: true, status: 200, statusText: 'OK', json: async () => body }
}

function mountPanel() {
  return mount(ServerPanel, { global: { stubs: { ModalDialog: modalStub } } })
}

function findButton(w: ReturnType<typeof mount>, text: string) {
  const btn = w.findAll('button').find((b) => b.text() === text)
  expect(btn, `button "${text}" should exist`).toBeTruthy()
  return btn!
}

beforeEach(() => {
  vi.clearAllMocks()
  fetchMock = vi.fn()
  vi.stubGlobal('fetch', fetchMock)
  origReload = window.location.reload
  reloadSpy = vi.fn()
  ;(window.location as { reload: unknown }).reload = reloadSpy
})

afterEach(() => {
  ;(window.location as { reload: unknown }).reload = origReload
  vi.unstubAllGlobals()
  vi.useRealTimers()
})

describe('ServerPanel', () => {
  it('空态：无服务器时显示空提示与活动占位，不发起探测', async () => {
    vi.mocked(api.listServers).mockReturnValue([])
    vi.mocked(api.activeServerId).mockReturnValue('')
    const w = mountPanel()
    await flushPromises()
    expect(w.find('.empty').text()).toContain('server.empty')
    expect(w.text()).toContain('server.active')
    expect(w.text()).toBeDefined()
    expect(fetchMock).not.toHaveBeenCalled()
    const cb = w.find('input[type="checkbox"]').element as HTMLInputElement
    expect(cb.checked).toBe(false)
  })

  it('渲染服务器行：活动标记、版本 badge、探测去尾斜杠与 Bearer 头', async () => {
    vi.mocked(api.listServers).mockReturnValue([s1, s2])
    vi.mocked(api.activeServerId).mockReturnValue('s1')
    fetchMock
      .mockResolvedValueOnce(okResponse({ version: '1.4.0' }))
      .mockResolvedValueOnce(okResponse({}))
    const w = mountPanel()
    await flushPromises()

    expect(fetchMock).toHaveBeenNthCalledWith(1, 'http://x:9000/api/health', { headers: { Authorization: 'Bearer t1' } })
    expect(fetchMock).toHaveBeenNthCalledWith(2, '/api/health', { headers: {} })
    expect(w.text()).toContain('S1')
    expect(w.text()).toContain('S2')
    expect(w.text()).toContain('v1.4.0')
    expect(w.text()).toContain('server.tokenSet')
    expect(w.text()).toContain('server.sameOrigin')
    expect(w.text()).toContain('server.activeTag')
    expect(w.text()).toContain('server.healthOk')
    expect(w.findAll('tbody tr').length).toBe(2)
    expect(w.findAll('tbody tr')[0].classes()).toContain('selected')
  })

  it('健康检查失败：非 ok 响应与网络异常展示错误信息', async () => {
    vi.mocked(api.listServers).mockReturnValue([s1])
    vi.mocked(api.activeServerId).mockReturnValue('s1')
    fetchMock.mockResolvedValueOnce({ ok: false, status: 500, statusText: 'Internal Server Error', json: async () => ({}) })
    const w = mountPanel()
    await flushPromises()
    const bad = w.find('span.tag.bad')
    expect(bad.text()).toBe('server.healthFail')
    expect(bad.attributes('title')).toBe('500 Internal Server Error')
    w.unmount()

    fetchMock.mockRejectedValueOnce(new Error('net down'))
    const w2 = mountPanel()
    await flushPromises()
    expect(w2.find('span.tag.bad').attributes('title')).toBe('net down')
  })

  it('健康检查：非 JSON 响应体被容忍（无版本 badge）', async () => {
    vi.mocked(api.listServers).mockReturnValue([s1])
    vi.mocked(api.activeServerId).mockReturnValue('s1')
    fetchMock.mockResolvedValueOnce({ ok: true, status: 200, statusText: 'OK', json: async () => { throw new Error('bad json') } })
    const w = mountPanel()
    await flushPromises()
    expect(w.text()).toContain('server.healthOk')
    expect(w.findAll('.tag').some((t) => (t.text() || '').startsWith('v'))).toBe(false)
  })

  it('探测中显示 checking 状态', async () => {
    vi.mocked(api.listServers).mockReturnValue([s1])
    vi.mocked(api.activeServerId).mockReturnValue('s1')
    fetchMock.mockReturnValue(new Promise(() => {}))
    const w = mountPanel()
    await flushPromises()
    expect(w.text()).toContain('server.healthChecking')
  })

  it('单个探测按钮与全部探测按钮再次发起请求', async () => {
    vi.mocked(api.listServers).mockReturnValue([s1, s2])
    vi.mocked(api.activeServerId).mockReturnValue('s1')
    fetchMock.mockResolvedValue(okResponse({ version: '1' }))
    const w = mountPanel()
    await flushPromises()
    expect(fetchMock).toHaveBeenCalledTimes(2)
    await findButton(w, 'server.probe').trigger('click')
    await flushPromises()
    expect(fetchMock).toHaveBeenCalledTimes(3)
    await findButton(w, 'server.probeAll').trigger('click')
    await flushPromises()
    expect(fetchMock).toHaveBeenCalledTimes(5)
  })

  it('持久化开关调用 api.setTokenPersistent', async () => {
    vi.mocked(api.listServers).mockReturnValue([])
    vi.mocked(api.activeServerId).mockReturnValue('')
    const w = mountPanel()
    await flushPromises()
    const cb = w.find('input[type="checkbox"]')
    await cb.setValue(true)
    expect(api.setTokenPersistent).toHaveBeenCalledWith(true)
    await cb.setValue(false)
    expect(api.setTokenPersistent).toHaveBeenCalledWith(false)
  })

  it('新增按钮打开表单（默认名称/base）并可取消', async () => {
    vi.mocked(api.listServers).mockReturnValue([])
    vi.mocked(api.activeServerId).mockReturnValue('')
    const w = mountPanel()
    await flushPromises()
    await findButton(w, 'server.add').trigger('click')
    expect(w.find('.modal-stub').exists()).toBe(true)
    expect((w.find('input[placeholder="server.namePh"]').element as HTMLInputElement).value).toBe('server.defaultName')
    expect((w.find('input[placeholder="server.basePh"]').element as HTMLInputElement).value).toBe('http://127.0.0.1:8080')
    await findButton(w, 'common.cancel').trigger('click')
    expect(w.find('.modal-stub').exists()).toBe(false)
  })

  it('名称为空时提交显示错误且不调用 API', async () => {
    vi.mocked(api.listServers).mockReturnValue([])
    vi.mocked(api.activeServerId).mockReturnValue('')
    const w = mountPanel()
    await flushPromises()
    await findButton(w, 'server.add').trigger('click')
    await w.find('input[placeholder="server.namePh"]').setValue('')
    await findButton(w, 'common.save').trigger('click')
    await flushPromises()
    expect(w.find('.msg.err').text()).toBe('server.nameRequired')
    expect(api.upsertServer).not.toHaveBeenCalled()
    expect(w.find('.modal-stub').exists()).toBe(true)
  })

  it('ModalDialog 的 close 事件关闭表单（@close 绑定）', async () => {
    vi.mocked(api.listServers).mockReturnValue([])
    vi.mocked(api.activeServerId).mockReturnValue('')
    const w = mountPanel()
    await flushPromises()
    await findButton(w, 'server.add').trigger('click')
    expect(w.find('.modal-stub').exists()).toBe(true)
    const dlg = w.findComponent({ name: 'ModalDialog' })
    expect(dlg.exists()).toBe(true)
    dlg.vm.$emit('close')
    await nextTick()
    expect(w.find('.modal-stub').exists()).toBe(false)
  })

  it('新建保存流程：upsert + toast + 关闭表单（非当前项不刷新页面）', async () => {
    vi.mocked(api.listServers).mockReturnValue([s1])
    vi.mocked(api.activeServerId).mockReturnValue('s1')
    vi.mocked(api.upsertServer).mockReturnValue({ id: 'new-1', name: 'NewS', base: 'http://n:1', token: 'tok' })
    const w = mountPanel()
    await flushPromises()
    await findButton(w, 'server.add').trigger('click')
    await w.find('input[placeholder="server.namePh"]').setValue('NewS')
    await w.find('input[placeholder="server.basePh"]').setValue('http://n:1/')
    await w.find('input[placeholder="server.tokenPh"]').setValue('tok')
    await findButton(w, 'common.save').trigger('click')
    await flushPromises()
    expect(api.upsertServer).toHaveBeenCalledWith({ id: undefined, name: 'NewS', base: 'http://n:1/', token: 'tok' })
    expect(toast).toHaveBeenCalledWith('server.toastAdded')
    expect(w.find('.modal-stub').exists()).toBe(false)
    // p.id !== activeId → 不安排页面刷新
    expect(reloadSpy).not.toHaveBeenCalled()
  })

  it('编辑当前生效服务器：保存后刷新页面', async () => {
    vi.mocked(api.listServers).mockReturnValue([s1])
    vi.mocked(api.activeServerId).mockReturnValue('s1')
    vi.mocked(api.upsertServer).mockReturnValue({ ...s1, name: 'S1x' })
    vi.useFakeTimers()
    const w = mountPanel()
    await flushPromises()
    await findButton(w, 'server.edit').trigger('click')
    expect((w.find('input[placeholder="server.namePh"]').element as HTMLInputElement).value).toBe('S1')
    expect((w.find('input[placeholder="server.basePh"]').element as HTMLInputElement).value).toBe('http://x:9000/')
    expect((w.find('input[placeholder="server.tokenPh"]').element as HTMLInputElement).value).toBe('t1')
    await w.find('input[placeholder="server.namePh"]').setValue('S1x')
    await findButton(w, 'common.save').trigger('click')
    await flushPromises()
    expect(api.upsertServer).toHaveBeenCalledWith({ id: 's1', name: 'S1x', base: 'http://x:9000/', token: 't1' })
    expect(toast).toHaveBeenCalledWith('server.toastUpdated')
    await vi.advanceTimersByTimeAsync(300)
    expect(reloadSpy).toHaveBeenCalledTimes(1)
  })

  it('删除：取消不删除；删除当前项触发切换提示与刷新', async () => {
    vi.mocked(api.listServers).mockReturnValue([s1])
    vi.mocked(api.activeServerId).mockReturnValue('s1')
    vi.mocked(confirmDialog).mockResolvedValueOnce(false)
    vi.useFakeTimers()
    const w = mountPanel()
    await flushPromises()
    await findButton(w, 'server.delete').trigger('click')
    await flushPromises()
    expect(api.deleteServer).not.toHaveBeenCalled()

    vi.mocked(confirmDialog).mockResolvedValueOnce(true)
    await findButton(w, 'server.delete').trigger('click')
    await flushPromises()
    expect(api.deleteServer).toHaveBeenCalledWith('s1')
    expect(toast).toHaveBeenNthCalledWith(1, 'server.toastDeleted')
    expect(toast).toHaveBeenNthCalledWith(2, 'server.toastSwitchedActive')
    await vi.advanceTimersByTimeAsync(300)
    expect(reloadSpy).toHaveBeenCalledTimes(1)
  })

  it('删除非当前服务器不提示切换、不刷新页面', async () => {
    vi.mocked(api.listServers).mockReturnValue([s1, s2])
    vi.mocked(api.activeServerId).mockReturnValue('s1')
    const w = mountPanel()
    await flushPromises()
    await w.findAll('tbody tr')[1].find('button.danger.sm').trigger('click')
    await flushPromises()
    expect(api.deleteServer).toHaveBeenCalledWith('s2')
    expect(toast).toHaveBeenCalledTimes(1)
    expect(reloadSpy).not.toHaveBeenCalled()
    // 行仍渲染（mock 未删除），且 healthMap 条目已被清理 → healthUntested
    expect(w.text()).toContain('server.healthUntested')
    expect(w.text()).toContain('S2')
  })

  it('选择服务器：当前项 no-op，切换项调用 selectServer 并刷新', async () => {
    vi.mocked(api.listServers).mockReturnValue([s1, s2])
    vi.mocked(api.activeServerId).mockReturnValue('s1')
    vi.useFakeTimers()
    const w = mountPanel()
    await flushPromises()
    const radios = w.findAll('input[name="server"]')
    expect(radios.length).toBe(2)
    // 点击当前生效项 → 提前返回
    await radios[0].setValue()
    expect(api.selectServer).not.toHaveBeenCalled()
    // 切换到 s2 → selectServer + toast + 刷新
    await radios[1].setValue()
    await flushPromises()
    expect(api.selectServer).toHaveBeenCalledWith('s2')
    expect(toast).toHaveBeenCalledWith('server.toastSelect')
    await vi.advanceTimersByTimeAsync(300)
    expect(reloadSpy).toHaveBeenCalledTimes(1)
  })

  it('选择当前已激活服务器时 short-circuit：不调 api、不提示', async () => {
    vi.mocked(api.listServers).mockReturnValue([s1, s2])
    vi.mocked(api.activeServerId).mockReturnValue('s1')
    fetchMock.mockResolvedValue(okResponse({}))
    const w = mountPanel()
    await flushPromises()
    await (w.vm as unknown as { select: (s: ServerProfile) => void }).select(s1)
    expect(api.selectServer).not.toHaveBeenCalled()
    expect(toast).not.toHaveBeenCalled()
    expect(reloadSpy).not.toHaveBeenCalled()
  })
})
