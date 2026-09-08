import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import StorageClassDialog from './StorageClassDialog.vue'
import { s3api } from '../api'
import { toast } from '../store'
import { STORAGE_CLASS_VALUES } from '../storageClass'

vi.mock('../api', () => ({
  s3api: {
    changeStorageClass: vi.fn(async () => ({ storageClass: 'GLACIER' })),
  },
}))

vi.mock('../store', () => ({ toasts: [], toast: vi.fn() }))

vi.mock('../i18n', () => ({
  t: (k: string) => k,
  tf: (k: string) => k,
  locale: () => 'en-US',
}))

function mountDialog(currentClass = 'STANDARD') {
  return mount(StorageClassDialog, {
    props: {
      open: false,
      accountId: 'acc-1',
      bucket: 'b',
      objectKey: 'k',
      currentClass,
    },
    attachTo: document.body,
  })
}

async function openDialog(w: ReturnType<typeof mountDialog>) {
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

function selectEl(): HTMLSelectElement {
  const el = document.body.querySelector('select') as HTMLSelectElement | null
  expect(el).toBeTruthy()
  return el!
}

function selectIndex(sel: HTMLSelectElement, i: number) {
  sel.selectedIndex = i
  sel.dispatchEvent(new Event('change'))
}

afterEach(() => {
  vi.clearAllMocks()
})

describe('StorageClassDialog', () => {
  it('opens: all storage options, current badge, default selection', async () => {
    const w = mountDialog('STANDARD')
    await openDialog(w)
    const text = document.body.textContent ?? ''
    expect(text).toContain('storage.title')
    expect(text).toContain('storage.current')
    expect(text).toContain('storage.hint')
    const sel = selectEl()
    expect(sel.options.length).toBe(STORAGE_CLASS_VALUES.length)
    expect(sel.selectedIndex).toBe(STORAGE_CLASS_VALUES.indexOf('STANDARD'))
    // 未变更时按钮禁用
    expect(bodyBtn('storage.switch').disabled).toBe(true)
    w.unmount()
  })

  it('empty currentClass defaults to STANDARD', async () => {
    const w = mountDialog('')
    await openDialog(w)
    expect(selectEl().selectedIndex).toBe(STORAGE_CLASS_VALUES.indexOf('STANDARD'))
    expect(bodyBtn('storage.switch').disabled).toBe(false)
    w.unmount()
  })

  it('submit with unchanged class only reports storage.unchanged', async () => {
    const w = mountDialog('STANDARD')
    await openDialog(w)
    await (w.vm as unknown as { submit: () => Promise<void> }).submit()
    await flushPromises()
    expect(toast).toHaveBeenCalledWith('storage.unchanged')
    expect(vi.mocked(s3api.changeStorageClass)).not.toHaveBeenCalled()
    expect(w.emitted('saved')).toBeUndefined()
    w.unmount()
  })

  it('switches class: changeStorageClass + toast + saved + busy guard', async () => {
    type ChangeResult = Awaited<ReturnType<typeof s3api.changeStorageClass>>
    let resolveChange!: (v: ChangeResult) => void
    const pending = new Promise<ChangeResult>((resolve) => { resolveChange = resolve })
    vi.mocked(s3api.changeStorageClass).mockImplementation(async () => pending)
    const w = mountDialog('STANDARD')
    await openDialog(w)
    selectIndex(selectEl(), STORAGE_CLASS_VALUES.indexOf('GLACIER'))
    await flushPromises()

    clickBody('storage.switch')
    await flushPromises()
    expect(bodyBtn('storage.switching').disabled).toBe(true)
    // saving 防重入：直接再调一次方法
    await (w.vm as unknown as { submit: () => Promise<void> }).submit()
    expect(vi.mocked(s3api.changeStorageClass)).toHaveBeenCalledTimes(1)
    expect(vi.mocked(s3api.changeStorageClass)).toHaveBeenCalledWith('acc-1', {
      bucket: 'b',
      key: 'k',
      storageClass: 'GLACIER',
    })

    resolveChange({ changed: 'k', versionId: 'v2', storageClass: 'GLACIER' })
    await flushPromises()
    expect(toast).toHaveBeenCalledWith('storage.toastChanged')
    expect(w.emitted('saved')).toBeTruthy()
    // saving 复位后按钮重新可用（chosen 仍不等于 currentClass）
    expect(bodyBtn('storage.switch').disabled).toBe(false)
    w.unmount()
  })

  it('emits error when changeStorageClass fails', async () => {
    vi.mocked(s3api.changeStorageClass).mockRejectedValue(new Error('storage-boom'))
    const w = mountDialog('STANDARD')
    await openDialog(w)
    selectIndex(selectEl(), STORAGE_CLASS_VALUES.indexOf('GLACIER'))
    await flushPromises()
    clickBody('storage.switch')
    await flushPromises()
    expect(w.emitted('error')).toEqual([['storage-boom']])
    expect(w.emitted('saved')).toBeUndefined()
    w.unmount()
  })

  it('close button emits close', async () => {
    const w = mountDialog('STANDARD')
    await openDialog(w)
    clickBody('common.close')
    expect(w.emitted('close')).toBeTruthy()
    w.unmount()
  })

  it('open→false 走 watcher 的 early-return；重开回到默认选中', async () => {
    const w = mountDialog('GLACIER')
    await openDialog(w)
    expect(selectEl().selectedIndex).toBe(STORAGE_CLASS_VALUES.indexOf('GLACIER'))
    await w.setProps({ open: false })
    await flushPromises()
    await w.setProps({ open: true })
    await flushPromises()
    expect(selectEl().selectedIndex).toBe(STORAGE_CLASS_VALUES.indexOf('GLACIER'))
    w.unmount()
  })

  it('ModalDialog 自身的 close 事件转发为父级 close', async () => {
    const w = mountDialog('STANDARD')
    await openDialog(w)
    w.findComponent({ name: 'ModalDialog' }).vm.$emit('close')
    expect(w.emitted('close')).toBeTruthy()
    w.unmount()
  })
})
