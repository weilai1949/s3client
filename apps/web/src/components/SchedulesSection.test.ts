// SchedulesSection.test.ts —— 计划任务区块（ROADMAP #6）：加载 / 表单校验 / 增改 / 启停 / 立即运行 / 删除。
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import SchedulesSection from './SchedulesSection.vue'
import { s3api } from '../api'
import { state, toast } from '../store'
import { confirmDialog } from '../confirm'
import type { Account, Schedule } from '../types'

vi.mock('../api', () => ({
  s3api: {
    listSchedules: vi.fn(),
    createSchedule: vi.fn(),
    updateSchedule: vi.fn(),
    deleteSchedule: vi.fn(),
    runScheduleNow: vi.fn(),
  },
}))

vi.mock('../store', async () => {
  const { reactive } = await import('vue')
  return {
    state: reactive({ accounts: [] as Account[], currentAccountId: '' }),
    toast: vi.fn(),
  }
})

vi.mock('../confirm', () => ({
  confirmDialog: vi.fn(async () => true),
}))

vi.mock('../i18n', () => ({
  t: (k: string) => k,
  tf: (k: string, vars?: Record<string, string | number>) =>
    vars ? `${k}:${Object.values(vars).join('|')}` : k,
}))

const acc1: Account = {
  id: 'acc-1',
  name: 'src',
  endpoint: 'http://localhost:9000',
  publicEndpoint: 'http://localhost:9000',
  region: 'us-east-1',
  accessKey: 'ak',
  secretSet: true,
  bucket: 'b1',
  pathStyle: true,
  useSSL: false,
  createdAt: '2026-01-01T00:00:00Z',
  updatedAt: '2026-01-01T00:00:00Z',
}

const acc2: Account = { ...acc1, id: 'acc-2', name: 'dst', bucket: 'b2' }

const sched1: Schedule = {
  id: 'sched-1',
  sourceAccountId: 'acc-1',
  sourceBucket: 'b1',
  sourcePrefix: 'data/',
  targetAccountId: 'acc-2',
  targetBucket: 'b2',
  targetPrefix: 'backup/',
  mode: 'etag',
  cron: '0 2 * * *',
  enabled: true,
  createdAt: '2026-10-08T10:00:00Z',
  nextRunAt: '2026-10-09T02:00:00Z',
}

async function mountSection() {
  const w = mount(SchedulesSection)
  await flushPromises()
  return w
}

beforeEach(() => {
  state.accounts = [acc1, acc2]
  vi.mocked(s3api.listSchedules).mockResolvedValue({ schedules: [sched1] })
  vi.mocked(s3api.createSchedule).mockResolvedValue({ schedule: sched1 })
  vi.mocked(s3api.updateSchedule).mockResolvedValue({ schedule: { ...sched1, enabled: false } })
  vi.mocked(s3api.deleteSchedule).mockResolvedValue({ deleted: 'sched-1' })
  vi.mocked(s3api.runScheduleNow).mockResolvedValue({ jobId: 'job-1', scheduleId: 'sched-1' })
  vi.mocked(confirmDialog).mockResolvedValue(true)
})

afterEach(() => {
  vi.clearAllMocks()
})

describe('加载与渲染', () => {
  it('挂载即加载并渲染行（含 cron、路线、下次运行）', async () => {
    const w = await mountSection()
    expect(s3api.listSchedules).toHaveBeenCalledTimes(1)
    expect(w.text()).toContain('0 2 * * *')
    expect(w.text()).toContain('b1|b2') // routeArrow 模板
    expect(w.text()).toContain('2026-10-09T02:00:00Z')
  })

  it('空清单显示空态提示', async () => {
    vi.mocked(s3api.listSchedules).mockResolvedValue({ schedules: [] })
    const w = await mountSection()
    expect(w.text()).toContain('schedule.empty')
  })

  it('加载失败显示行内横幅（role=alert）', async () => {
    vi.mocked(s3api.listSchedules).mockRejectedValue(new Error('boom'))
    const w = await mountSection()
    const alert = w.find('[role="alert"]')
    expect(alert.exists()).toBe(true)
    expect(alert.text()).toContain('boom')
  })

  it('从未运行显示 never，lastError 显示在行内', async () => {
    vi.mocked(s3api.listSchedules).mockResolvedValue({
      schedules: [{ ...sched1, lastRunAt: undefined, lastError: 'cron boom' }],
    })
    const w = await mountSection()
    expect(w.text()).toContain('schedule.never')
    expect(w.text()).toContain('cron boom')
  })

  it('响应缺 schedules 字段时回退空数组（?? [] 分支）', async () => {
    vi.mocked(s3api.listSchedules).mockResolvedValue({} as never)
    const w = await mountSection()
    expect(w.text()).toContain('schedule.empty')
  })

  it('停用且已运行过的计划：enabled=false 与 lastRunAt 的渲染分支', async () => {
    vi.mocked(s3api.listSchedules).mockResolvedValue({
      schedules: [{ ...sched1, enabled: false, lastRunAt: '2026-10-08T02:00:00Z' }],
    })
    const w = await mountSection()
    expect(w.text()).toContain('schedule.statusOff')
    expect(w.text()).toContain('2026-10-08T02:00:00Z')
    expect(w.text()).not.toContain('schedule.never')
    // 停用态的启停按钮显示「启用」
    const toggleBtn = w.findAll('tbody button')[1]
    expect(toggleBtn?.text()).toBe('schedule.statusOn')
  })

  it('编辑省略前缀的计划：sourcePrefix/targetPrefix 走 ?? 回退', async () => {
    vi.mocked(s3api.listSchedules).mockResolvedValue({
      schedules: [{ ...sched1, sourcePrefix: undefined, targetPrefix: undefined }],
    })
    const w = await mountSection()
    await w.findAll('tbody button')[2]?.trigger('click')
    const inputs = w.findAll('form input:not([type])')
    expect((inputs[1].element as HTMLInputElement).value).toBe('') // 源前缀
    expect((inputs[3].element as HTMLInputElement).value).toBe('') // 目标前缀
  })

  it('刷新按钮重新加载', async () => {
    const w = await mountSection()
    await w.find('.sched-head button').trigger('click')
    await flushPromises()
    expect(s3api.listSchedules).toHaveBeenCalledTimes(2)
  })
})

describe('新建表单校验', () => {
  it('缺源/目标账号 → 表单错误（不发请求）', async () => {
    const w = await mountSection()
    await w.find('.sched-head button:last-child').trigger('click')
    await w.find('form').trigger('submit')
    await flushPromises()
    expect(w.find('form [role="alert"]').text()).toBe('schedule.accountRequired')
    expect(s3api.createSchedule).not.toHaveBeenCalled()
  })

  it('缺 cron → 表单错误（不发请求）', async () => {
    const w = await mountSection()
    await w.find('.sched-head button:last-child').trigger('click')
    const selects = w.findAll('form select')
    await selects[0].setValue('acc-1')
    await selects[1].setValue('acc-2')
    await w.find('form').trigger('submit')
    await flushPromises()
    expect(w.find('form [role="alert"]').text()).toBe('schedule.cronRequired')
    expect(s3api.createSchedule).not.toHaveBeenCalled()
  })

  it('创建成功：填满全表单 → toast + 关表单 + 重载（请求体逐字段断言）', async () => {
    const w = await mountSection()
    await w.find('.sched-head button:last-child').trigger('click')
    const selects = w.findAll('form select')
    await selects[0].setValue('acc-1') // 源账号
    await selects[1].setValue('acc-2') // 目标账号
    await selects[2].setValue('size_mtime') // 比对模式
    const inputs = w.findAll('form input:not([type])')
    await inputs[0].setValue('b1') // 源桶
    await inputs[1].setValue('data/') // 源前缀
    await inputs[2].setValue('b2') // 目标桶
    await inputs[3].setValue('backup/') // 目标前缀
    await inputs[4].setValue('0 3 * * *') // cron
    await w.find('form input[type="checkbox"]').setValue(false) // 启用 → 停用
    await w.find('form').trigger('submit')
    await flushPromises()
    expect(s3api.createSchedule).toHaveBeenCalledWith({
      sourceAccountId: 'acc-1',
      sourceBucket: 'b1',
      sourcePrefix: 'data/',
      targetAccountId: 'acc-2',
      targetBucket: 'b2',
      targetPrefix: 'backup/',
      mode: 'size_mtime',
      cron: '0 3 * * *',
      enabled: false,
    })
    expect(toast).toHaveBeenCalledWith('schedule.created')
    expect(w.find('form').exists()).toBe(false)
  })

  it('服务端 400（cron 非法）回显在表单横幅', async () => {
    vi.mocked(s3api.createSchedule).mockRejectedValue(new Error('cron: bad'))
    const w = await mountSection()
    await w.find('.sched-head button:last-child').trigger('click')
    const selects = w.findAll('form select')
    await selects[0].setValue('acc-1')
    await selects[1].setValue('acc-2')
    const inputs = w.findAll('form input:not([type])')
    await inputs[inputs.length - 1]?.setValue('bad')
    await w.find('form').trigger('submit')
    await flushPromises()
    expect(w.find('form [role="alert"]').text()).toContain('cron: bad')
    expect(w.find('form').exists()).toBe(true) // 出错不关表单
  })

  it('取消按钮关闭表单', async () => {
    const w = await mountSection()
    await w.find('.sched-head button:last-child').trigger('click')
    expect(w.find('form').exists()).toBe(true)
    await w.findAll('form button')[1]?.trigger('click')
    expect(w.find('form').exists()).toBe(false)
  })
})

describe('编辑 / 启停 / 运行 / 删除', () => {
  it('编辑：表单回填既有值，保存走 updateSchedule', async () => {
    const w = await mountSection()
    const editBtn = w.findAll('tbody button')[2] // run / toggle / edit / delete
    await editBtn?.trigger('click')
    const inputs = w.findAll('form input:not([type])')
    expect((inputs[inputs.length - 1].element as HTMLInputElement).value).toBe('0 2 * * *')
    await w.find('form').trigger('submit')
    await flushPromises()
    expect(s3api.updateSchedule).toHaveBeenCalledWith('sched-1', expect.objectContaining({ cron: '0 2 * * *' }))
    expect(toast).toHaveBeenCalledWith('schedule.updated')
  })

  it('启停切换走 updateSchedule 并翻转 enabled', async () => {
    const w = await mountSection()
    const toggleBtn = w.findAll('tbody button')[1]
    await toggleBtn?.trigger('click')
    await flushPromises()
    expect(s3api.updateSchedule).toHaveBeenCalledWith('sched-1', expect.objectContaining({ enabled: false }))
    expect(toast).toHaveBeenCalledWith('schedule.toggled')
  })

  it('启停失败 → err toast（不重载）', async () => {
    vi.mocked(s3api.updateSchedule).mockRejectedValue(new Error('403'))
    const w = await mountSection()
    const calls = vi.mocked(s3api.listSchedules).mock.calls.length
    await w.findAll('tbody button')[1]?.trigger('click')
    await flushPromises()
    expect(toast).toHaveBeenCalledWith('403', 'err')
    expect(vi.mocked(s3api.listSchedules).mock.calls.length).toBe(calls)
  })

  it('立即运行：toast 带任务 id 并重载', async () => {
    const w = await mountSection()
    await w.findAll('tbody button')[0]?.trigger('click')
    await flushPromises()
    expect(s3api.runScheduleNow).toHaveBeenCalledWith('sched-1')
    expect(toast).toHaveBeenCalledWith('schedule.runStarted:job-1')
  })

  it('运行失败 → err toast', async () => {
    vi.mocked(s3api.runScheduleNow).mockRejectedValue(new Error('409'))
    const w = await mountSection()
    await w.findAll('tbody button')[0]?.trigger('click')
    await flushPromises()
    expect(toast).toHaveBeenCalledWith('409', 'err')
  })

  it('删除：确认后调用 deleteSchedule + toast + 重载', async () => {
    const w = await mountSection()
    await w.findAll('tbody button')[3]?.trigger('click')
    await flushPromises()
    expect(confirmDialog).toHaveBeenCalled()
    expect(s3api.deleteSchedule).toHaveBeenCalledWith('sched-1')
    expect(toast).toHaveBeenCalledWith('schedule.deleted')
  })

  it('删除：用户取消确认 → 不发请求', async () => {
    vi.mocked(confirmDialog).mockResolvedValueOnce(false)
    const w = await mountSection()
    await w.findAll('tbody button')[3]?.trigger('click')
    await flushPromises()
    expect(s3api.deleteSchedule).not.toHaveBeenCalled()
  })

  it('删除失败 → err toast', async () => {
    vi.mocked(s3api.deleteSchedule).mockRejectedValue(new Error('404'))
    const w = await mountSection()
    await w.findAll('tbody button')[3]?.trigger('click')
    await flushPromises()
    expect(toast).toHaveBeenCalledWith('404', 'err')
  })
})
