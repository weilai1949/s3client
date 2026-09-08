import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import LifecycleDialog from './LifecycleDialog.vue'
import { s3api } from '../api'
import { toast } from '../store'

vi.mock('../api', () => ({
  s3api: {
    getLifecycle: vi.fn(),
    putLifecycle: vi.fn(async () => ({ updated: 1 })),
  },
}))

vi.mock('../store', () => ({ toasts: [], toast: vi.fn() }))

vi.mock('../i18n', () => ({
  t: (k: string) => k,
  tf: (k: string) => k,
  locale: () => 'en-US',
}))

function mountDialog() {
  return track(mount(LifecycleDialog, {
    props: { open: false, accountId: 'acc-1', bucket: 'b' },
    attachTo: document.body,
  }))
}

async function openDialog(w: ReturnType<typeof mountDialog>, rules: unknown[] = []) {
  vi.mocked(s3api.getLifecycle).mockResolvedValue({ rules } as never)
  await w.setProps({ open: true })
  await flushPromises()
}

function bodyBtn(text: string): HTMLButtonElement {
  const b = Array.from(document.body.querySelectorAll('button')).find(
    (x) => (x.textContent ?? '').trim() === text,
  )
  expect(b, `body button "${text}"`).toBeTruthy()
  return b as unknown as HTMLButtonElement
}

function clickBody(text: string) {
  bodyBtn(text).dispatchEvent(new MouseEvent('click', { bubbles: true }))
}

function prefixInputs(): HTMLInputElement[] {
  return Array.from(document.body.querySelectorAll('input[placeholder="lifecycle.prefixPh"]')) as HTMLInputElement[]
}

function daysInputs(): HTMLInputElement[] {
  return Array.from(document.body.querySelectorAll('input[type="number"]')) as HTMLInputElement[]
}

function fill(el: HTMLInputElement, v: string) {
  el.value = v
  el.dispatchEvent(new Event('input'))
}

let mounted: Array<{ unmount: () => void }> = []
function track<T extends { unmount: () => void }>(w: T): T {
  mounted.push(w)
  return w
}

afterEach(() => {
  for (const m of mounted) m.unmount()
  mounted = []
  document.body.innerHTML = ''
})
afterEach(() => {
  vi.clearAllMocks()
})

describe('LifecycleDialog', () => {
  it('shows loading skeleton, then renders rules', async () => {
    let resolveLoad!: (v: Awaited<ReturnType<typeof s3api.getLifecycle>>) => void
    vi.mocked(s3api.getLifecycle).mockImplementation(
      () => new Promise((resolve) => { resolveLoad = resolve }),
    )
    const w = mountDialog()
    await w.setProps({ open: true })
    await flushPromises()
    expect(document.body.querySelector('[aria-busy="true"]')).toBeTruthy()
    expect(document.body.textContent).not.toContain('lifecycle.empty')

    resolveLoad({ rules: [{ id: 'r1', prefix: 'logs', days: 30 }] })
    await flushPromises()
    expect(document.body.querySelector('[aria-busy="true"]')).toBeNull()
    expect(document.body.textContent).toContain('r1')
    expect(prefixInputs().map((i) => i.value)).toEqual(['logs'])
    expect(daysInputs().map((i) => i.value)).toEqual(['30'])
  })

  it('renders empty state and disables save when no rules', async () => {
    const w = mountDialog()
    await openDialog(w)
    expect(document.body.textContent).toContain('lifecycle.empty')
    expect(bodyBtn('lifecycle.saveRules').disabled).toBe(true)
  })

  it('rules 字段缺失(null/undefined)→ `res.rules ?? []` 兜底渲染空态', async () => {
    const w = mountDialog()
    vi.mocked(s3api.getLifecycle).mockResolvedValue({} as never)
    await w.setProps({ open: true })
    await flushPromises()
    expect(w.emitted('error')).toBeUndefined()
    expect(document.body.textContent).toContain('lifecycle.empty')
    expect(bodyBtn('lifecycle.saveRules').disabled).toBe(true)
  })

  it('emits error when loading fails', async () => {
    vi.mocked(s3api.getLifecycle).mockRejectedValue(new Error('lifecycle-load-boom'))
    const w = mountDialog()
    await w.setProps({ open: true })
    await flushPromises()
    expect(w.emitted('error')).toEqual([['lifecycle-load-boom']])
    // 加载失败后回到空列表态（loading 复位、无规则）
    expect(document.body.textContent).toContain('lifecycle.empty')
  })

  it('adds and removes rules with defaults', async () => {
    const w = mountDialog()
    await openDialog(w)
    clickBody('lifecycle.addRule')
    await flushPromises()
    expect(prefixInputs()).toHaveLength(1)
    expect(daysInputs().map((i) => i.value)).toEqual(['30'])
    expect(bodyBtn('lifecycle.saveRules').disabled).toBe(false)

    clickBody('common.delete')
    await flushPromises()
    expect(prefixInputs()).toHaveLength(0)
    expect(document.body.textContent).toContain('lifecycle.empty')
  })

  it('submits only valid rules and closes', async () => {
    const w = mountDialog()
    await openDialog(w, [
      { id: 'r1', prefix: 'logs', days: 30 },
      { id: 'r2', prefix: '', days: 5 }, // 无前缀 → 非法
    ])
    clickBody('lifecycle.addRule')
    await flushPromises()
    fill(prefixInputs()[2], 'tmp')
    fill(daysInputs()[2], '0') // days < 1 → 非法
    await flushPromises()
    clickBody('lifecycle.saveRules')
    await flushPromises()
    expect(vi.mocked(s3api.putLifecycle)).toHaveBeenCalledWith('acc-1', {
      bucket: 'b',
      rules: [{ id: 'r1', prefix: 'logs', days: 30 }],
    })
    expect(toast).toHaveBeenCalledWith('lifecycle.toastSaved')
    expect(w.emitted('close')).toBeTruthy()
  })

  it('emits error when saving fails', async () => {
    vi.mocked(s3api.putLifecycle).mockRejectedValue(new Error('lifecycle-save-boom'))
    const w = mountDialog()
    await openDialog(w, [{ id: 'r1', prefix: 'logs', days: 30 }])
    clickBody('lifecycle.saveRules')
    await flushPromises()
    expect(w.emitted('error')).toEqual([['lifecycle-save-boom']])
    expect(w.emitted('close')).toBeUndefined()
  })

  it('cancel button emits close', async () => {
    const w = mountDialog()
    await openDialog(w, [{ id: 'r1', prefix: 'logs', days: 30 }])
    clickBody('common.cancel')
    expect(w.emitted('close')).toBeTruthy()
  })

  it('open→false 走 watcher 的 early-return：不再重新拉取规则', async () => {
    const w = mountDialog()
    await openDialog(w, [{ id: 'r1', prefix: 'logs', days: 30 }])
    expect(vi.mocked(s3api.getLifecycle)).toHaveBeenCalledTimes(1)
    await w.setProps({ open: false })
    await flushPromises()
    expect(vi.mocked(s3api.getLifecycle)).toHaveBeenCalledTimes(1)
    expect(w.emitted('error')).toBeUndefined()
  })

  it('ModalDialog 自身的 close 事件转发为父级 close', async () => {
    const w = mountDialog()
    await openDialog(w)
    w.findComponent({ name: 'ModalDialog' }).vm.$emit('close')
    expect(w.emitted('close')).toBeTruthy()
  })
})
