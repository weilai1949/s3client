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

  it('异步提交防重复：在途时双击「保存规则」只发一次 putLifecycle', async () => {
    vi.mocked(s3api.putLifecycle).mockImplementationOnce(() => new Promise<never>(() => {})) // 请求挂起
    const w = mountDialog()
    await openDialog(w, [{ id: 'r1', prefix: 'logs', days: 30 }])
    clickBody('lifecycle.saveRules')
    await flushPromises()
    expect(bodyBtn('lifecycle.saveRules').disabled).toBe(true) // 在途时提交按钮禁用
    clickBody('lifecycle.saveRules') // 第二次点击必须被守卫拦住
    await flushPromises()
    expect(vi.mocked(s3api.putLifecycle)).toHaveBeenCalledTimes(1)
  })

  it('提交在途时直调 submitLifecycle 被守卫拦截；失败后发出 error 且按钮恢复', async () => {
    let rejectSave!: (e: Error) => void
    vi.mocked(s3api.putLifecycle).mockImplementationOnce(
      () =>
        new Promise<never>((_, rej) => {
          rejectSave = rej
        }),
    )
    const w = mountDialog()
    await openDialog(w, [{ id: 'r1', prefix: 'logs', days: 30 }])
    clickBody('lifecycle.saveRules')
    await flushPromises()
    expect(bodyBtn('lifecycle.saveRules').disabled).toBe(true) // 在途时提交按钮禁用
    // 按钮 disabled 绕过点击后，提交入口自身必须仍拦住第二次提交
    await (w.vm as unknown as { submitLifecycle: () => Promise<void> }).submitLifecycle()
    expect(vi.mocked(s3api.putLifecycle)).toHaveBeenCalledTimes(1)
    // 失败 settle：error 事件、未发 close、按钮恢复
    rejectSave(new Error('lifecycle-save-boom'))
    await flushPromises()
    expect(w.emitted('error')).toEqual([['lifecycle-save-boom']])
    expect(w.emitted('close')).toBeUndefined()
    expect(bodyBtn('lifecycle.saveRules').disabled).toBe(false)
  })
})

describe('LifecycleDialog 稳定行键', () => {
  it('删除中间规则行后其余行保留原 DOM 节点（v-for 键用行 id 而非 index）', async () => {
    const w = mountDialog()
    await openDialog(w, [
      { id: 'r1', prefix: 'logs', days: 30 },
      { id: 'r2', prefix: 'tmp', days: 10 },
      { id: 'r3', prefix: 'arch', days: 7 },
    ])
    const rows = () => Array.from(document.body.querySelectorAll('table.tbl tbody tr'))
    const before = rows()
    expect(before).toHaveLength(3)

    const middleDelete = before[1].querySelector('button') as HTMLButtonElement
    middleDelete.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    await flushPromises()

    const after = rows()
    expect(after).toHaveLength(2)
    // index 作 key 时：Vue 复用第 2 个节点承载第 3 行并卸载原第 3 个节点
    expect(after[0]).toBe(before[0])
    expect(after[1]).toBe(before[2])
    // 存活行的输入值仍是原第 1、3 行（防串行）
    expect(after.map((r) => (r.querySelector('input[placeholder="lifecycle.prefixPh"]') as HTMLInputElement).value))
      .toEqual(['logs', 'arch'])
    expect(after.map((r) => (r.querySelector('input[type="number"]') as HTMLInputElement).value))
      .toEqual(['30', '7'])
  })
})
