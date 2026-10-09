import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import BucketObjectLock from './BucketObjectLock.vue'
import { s3api } from '../api'
import { toast } from '../store'

vi.mock('../api', () => ({
  s3api: {
    getObjectLock: vi.fn(),
    putObjectLock: vi.fn(),
  },
}))
vi.mock('../store', () => ({ toast: vi.fn() }))
vi.mock('../i18n', () => ({
  t: (k: string) => k,
  tf: (k: string) => k,
  locale: () => 'en-US',
}))

interface Config {
  enabled?: boolean
  defaultRetentionMode?: '' | 'GOVERNANCE' | 'COMPLIANCE'
  defaultRetentionDays?: number
  defaultRetentionYears?: number
}

function lockConfig(cfg: Config) {
  return {
    bucket: 'b1',
    enabled: false,
    defaultRetentionMode: '' as const,
    defaultRetentionDays: 0,
    defaultRetentionYears: 0,
    ...cfg,
  }
}

function mountLock() {
  return mount(BucketObjectLock, { props: { accountId: 'acc-1', bucket: 'b1' } })
}

/** 保留策略表单控件：第一个 select 是模式，第二个是单位，number 输入是时长。 */
function form(w: ReturnType<typeof mountLock>) {
  const selects = w.findAll('select')
  expect(selects.length, '模式与单位两个 select').toBe(2)
  return {
    mode: selects[0],
    unit: selects[1],
    amount: w.find('input[type="number"]'),
  }
}

function saveBtn(w: ReturnType<typeof mountLock>) {
  const b = w.findAll('button').find((x) => x.text().trim() === 'objlock.save')
  expect(b, 'save button should exist').toBeTruthy()
  return b!
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(s3api.getObjectLock).mockResolvedValue(lockConfig({}))
  vi.mocked(s3api.putObjectLock).mockResolvedValue(
    lockConfig({ enabled: true, defaultRetentionMode: 'GOVERNANCE', defaultRetentionDays: 30 }),
  )
})

describe('BucketObjectLock', () => {
  it('loading → 读取配置；未启用时显示状态与不可补开提示', async () => {
    const w = mountLock()
    expect(w.text()).toContain('objlock.loading')
    await flushPromises()
    expect(s3api.getObjectLock).toHaveBeenCalledWith('acc-1', 'b1')
    const text = w.text()
    expect(text).toContain('objlock.disabled')
    expect(text).toContain('objlock.notEnabledHint')
    expect(text).toContain('objlock.hint')
    const f = form(w)
    expect((f.mode.element as HTMLSelectElement).value).toBe('GOVERNANCE')
    expect((f.unit.element as HTMLSelectElement).value).toBe('days')
    expect((f.amount.element as HTMLInputElement).value).toBe('1')
  })

  it('已启用（天数）：回填模式与时长，不显示不可补开提示', async () => {
    vi.mocked(s3api.getObjectLock).mockResolvedValue(
      lockConfig({ enabled: true, defaultRetentionMode: 'GOVERNANCE', defaultRetentionDays: 30 }),
    )
    const w = mountLock()
    await flushPromises()
    expect(w.text()).toContain('objlock.enabled')
    expect(w.text()).not.toContain('objlock.notEnabledHint')
    const f = form(w)
    expect((f.mode.element as HTMLSelectElement).value).toBe('GOVERNANCE')
    expect((f.unit.element as HTMLSelectElement).value).toBe('days')
    expect((f.amount.element as HTMLInputElement).value).toBe('30')
  })

  it('已启用（年数 + COMPLIANCE）：回填年数与模式', async () => {
    vi.mocked(s3api.getObjectLock).mockResolvedValue(
      lockConfig({ enabled: true, defaultRetentionMode: 'COMPLIANCE', defaultRetentionYears: 2 }),
    )
    const w = mountLock()
    await flushPromises()
    const f = form(w)
    expect((f.mode.element as HTMLSelectElement).value).toBe('COMPLIANCE')
    expect((f.unit.element as HTMLSelectElement).value).toBe('years')
    expect((f.amount.element as HTMLInputElement).value).toBe('2')
  })

  it('保存（天）：putObjectLock 携带 defaultRetentionDays → toast → changed', async () => {
    const w = mountLock()
    await flushPromises()
    const f = form(w)
    await f.mode.setValue('COMPLIANCE')
    await f.amount.setValue('45')
    await saveBtn(w).trigger('click')
    await flushPromises()
    expect(s3api.putObjectLock).toHaveBeenCalledWith('acc-1', {
      bucket: 'b1',
      defaultRetentionMode: 'COMPLIANCE',
      defaultRetentionDays: 45,
    })
    expect(toast).toHaveBeenCalledWith('objlock.toastSaved')
    expect(w.emitted('changed')).toHaveLength(1)
  })

  it('保存（年）：putObjectLock 携带 defaultRetentionYears', async () => {
    const w = mountLock()
    await flushPromises()
    const f = form(w)
    await f.unit.setValue('years')
    await f.amount.setValue('3')
    await saveBtn(w).trigger('click')
    await flushPromises()
    expect(s3api.putObjectLock).toHaveBeenCalledWith('acc-1', {
      bucket: 'b1',
      defaultRetentionMode: 'GOVERNANCE',
      defaultRetentionYears: 3,
    })
  })

  it('时长非法（<1 或非整数）→ 本地拦截，不发请求', async () => {
    const w = mountLock()
    await flushPromises()
    await form(w).amount.setValue('0')
    await saveBtn(w).trigger('click')
    await flushPromises()
    expect(s3api.putObjectLock).not.toHaveBeenCalled()
    expect(w.text()).toContain('objlock.invalidAmount')
    expect(toast).not.toHaveBeenCalled()
  })

  it('保存中：按钮切换为保存中文案并禁用，完成后复原', async () => {
    let release: () => void = () => {}
    vi.mocked(s3api.putObjectLock).mockImplementation(
      () =>
        new Promise((resolve) => {
          release = () => resolve(lockConfig({ enabled: true, defaultRetentionMode: 'GOVERNANCE', defaultRetentionDays: 7 }))
        }),
    )
    const w = mountLock()
    await flushPromises()
    await saveBtn(w).trigger('click')
    await nextTick()
    const savingBtn = w.findAll('button').find((b) => b.text().trim() === 'common.saving')
    expect(savingBtn, '保存中按钮文案').toBeTruthy()
    expect((savingBtn!.element as HTMLButtonElement).disabled).toBe(true)
    release()
    await flushPromises()
    expect(w.emitted('changed')).toHaveLength(1)
  })

  it('putObjectLock 失败（409/501）→ 后端固定文案原样上浮，不 toast', async () => {
    vi.mocked(s3api.putObjectLock).mockRejectedValue(new Error('409 bucket was created without Object Lock'))
    const w = mountLock()
    await flushPromises()
    await saveBtn(w).trigger('click')
    await flushPromises()
    expect(w.emitted('error')).toEqual([['409 bucket was created without Object Lock']])
    expect(toast).not.toHaveBeenCalled()
  })

  it('读取失败 → emit error', async () => {
    vi.mocked(s3api.getObjectLock).mockRejectedValue(new Error('501 not implemented'))
    const w = mountLock()
    await flushPromises()
    expect(w.emitted('error')).toEqual([['501 not implemented']])
  })
})
