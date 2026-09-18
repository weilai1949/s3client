import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import HeadersDialog from './HeadersDialog.vue'
import { s3api } from '../api'
import { toast } from '../store'

vi.mock('../api', () => ({
  s3api: {
    setHeaders: vi.fn(async () => ({ ok: true })),
  },
}))

vi.mock('../store', () => ({ toasts: [], toast: vi.fn() }))

vi.mock('../i18n', () => ({
  t: (k: string) => k,
  tf: (k: string) => k,
  locale: () => 'en-US',
}))

const DETAIL = {
  key: 'k',
  size: 10,
  lastModified: '2024-01-01',
  etag: 'e1',
  contentType: 'text/plain',
  metadata: { a: '1', b: '2' },
}

function mountDialog(detail: unknown = undefined) {
  return track(mount(HeadersDialog, {
    props: {
      open: false,
      accountId: 'acc-1',
      bucket: 'b',
      objectKey: 'k',
      detail: detail as never,
    },
    attachTo: document.body,
  }))
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

function typeInput(): HTMLInputElement {
  const el = document.body.querySelector('input[placeholder="headers.contentTypePh"]') as HTMLInputElement | null
  expect(el).toBeTruthy()
  return el!
}

function keyInputs(): HTMLInputElement[] {
  return Array.from(document.body.querySelectorAll('input[placeholder="headers.keyPh"]')) as HTMLInputElement[]
}

function valueInputs(): HTMLInputElement[] {
  return Array.from(document.body.querySelectorAll('input[placeholder="headers.valuePh"]')) as HTMLInputElement[]
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

describe('HeadersDialog', () => {
  it('renders empty form without detail', async () => {
    const w = mountDialog()
    await openDialog(w)
    expect(document.body.textContent).toContain('headers.title')
    expect(typeInput().value).toBe('')
    expect(keyInputs()).toHaveLength(0)
  })

  it('prefills contentType and metadata rows from matching detail', async () => {
    const w = mountDialog(DETAIL)
    await openDialog(w)
    expect(typeInput().value).toBe('text/plain')
    const keys = keyInputs()
    const vals = valueInputs()
    expect(keys.map((k) => k.value)).toEqual(['a', 'b'])
    expect(vals.map((v) => v.value)).toEqual(['1', '2'])
  })

  it('ignores detail for a different object key', async () => {
    const w = mountDialog({ ...DETAIL, key: 'other' })
    await openDialog(w)
    expect(typeInput().value).toBe('')
    expect(keyInputs()).toHaveLength(0)
  })

  it('detail 的 contentType 空/metadata 缺失：`|| \'\'` 与 `?? {}` 兜底渲染空表单', async () => {
    const w = mountDialog({ ...DETAIL, contentType: '', metadata: undefined })
    await openDialog(w)
    expect(w.emitted('error')).toBeUndefined()
    expect(typeInput().value).toBe('')
    expect(keyInputs()).toHaveLength(0)
    expect(document.body.textContent).toContain('headers.title')
  })

  it('adds and removes meta rows', async () => {
    const w = mountDialog()
    await openDialog(w)
    clickBody('headers.add')
    await flushPromises()
    expect(keyInputs()).toHaveLength(1)
    clickBody('headers.add')
    await flushPromises()
    expect(keyInputs()).toHaveLength(2)

    const removeBtns = Array.from(document.body.querySelectorAll('button')).filter(
      (b) => !b.classList.contains('dlg-x') && (b.textContent ?? '').trim() === '✕',
    )
    expect(removeBtns).toHaveLength(2)
    removeBtns[0].dispatchEvent(new MouseEvent('click', { bubbles: true }))
    await flushPromises()
    expect(keyInputs()).toHaveLength(1)
  })

  it('submits trimmed keys, skips empty values, toasts and emits saved', async () => {
    const w = mountDialog()
    await openDialog(w)
    fill(typeInput(), ' text/html ')
    clickBody('headers.add')
    await flushPromises()
    fill(keyInputs()[0], '  env ')
    fill(valueInputs()[0], 'prod')
    clickBody('headers.add')
    await flushPromises()
    // 第二行只有 key 没有 value → 被跳过
    fill(keyInputs()[1], 'empty')

    await flushPromises()
    clickBody('common.save')
    await flushPromises()
    expect(vi.mocked(s3api.setHeaders)).toHaveBeenCalledWith('acc-1', {
      bucket: 'b',
      key: 'k',
      contentType: 'text/html',
      metadata: { env: 'prod' },
    })
    expect(toast).toHaveBeenCalledWith('headers.toastUpdated')
    expect(w.emitted('saved')).toBeTruthy()
  })

  it('emits error when saving fails', async () => {
    vi.mocked(s3api.setHeaders).mockRejectedValue(new Error('headers-boom'))
    const w = mountDialog()
    await openDialog(w)
    clickBody('common.save')
    await flushPromises()
    expect(w.emitted('error')).toEqual([['headers-boom']])
    expect(w.emitted('saved')).toBeUndefined()
  })

  it('cancel button emits close', async () => {
    const w = mountDialog()
    await openDialog(w)
    clickBody('common.cancel')
    expect(w.emitted('close')).toBeTruthy()
  })

  it('open→false 走 watcher 的 early-return；重开后重新预填', async () => {
    const w = mountDialog(DETAIL)
    await openDialog(w)
    await w.setProps({ open: false })
    await flushPromises()
    await w.setProps({ open: true })
    await flushPromises()
    expect(typeInput().value).toBe('text/plain')
    expect(keyInputs()).toHaveLength(2)
  })

  it('ModalDialog 自身的 close 事件转发为父级 close', async () => {
    const w = mountDialog()
    await openDialog(w)
    w.findComponent({ name: 'ModalDialog' }).vm.$emit('close')
    expect(w.emitted('close')).toBeTruthy()
  })
})
