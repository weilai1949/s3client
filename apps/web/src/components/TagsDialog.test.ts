import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import TagsDialog from './TagsDialog.vue'
import { s3api } from '../api'
import { toast } from '../store'

vi.mock('../api', () => ({
  s3api: {
    getObjectTags: vi.fn(),
    putObjectTags: vi.fn(async () => ({})),
  },
}))

vi.mock('../store', () => ({ toasts: [], toast: vi.fn() }))

vi.mock('../i18n', () => ({
  t: (k: string) => k,
  tf: (k: string) => k,
  locale: () => 'en-US',
}))

function mountDialog() {
  return track(mount(TagsDialog, {
    props: { open: false, accountId: 'acc-1', bucket: 'b', objectKey: 'k' },
    attachTo: document.body,
  }))
}

async function openDialog(w: ReturnType<typeof mountDialog>, tags: Array<{ key: string; value: string }> = []) {
  vi.mocked(s3api.getObjectTags).mockResolvedValue({ tags } as never)
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

function keyInputs(): HTMLInputElement[] {
  return Array.from(document.body.querySelectorAll('input[placeholder="tags.keyPh"]')) as HTMLInputElement[]
}

function valueInputs(): HTMLInputElement[] {
  return Array.from(document.body.querySelectorAll('input[placeholder="tags.valuePh"]')) as HTMLInputElement[]
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

describe('TagsDialog', () => {
  it('shows loading, then empty state when object has no tags', async () => {
    let resolveLoad!: (v: Awaited<ReturnType<typeof s3api.getObjectTags>>) => void
    vi.mocked(s3api.getObjectTags).mockImplementation(
      () => new Promise((resolve) => { resolveLoad = resolve }),
    )
    const w = mountDialog()
    await w.setProps({ open: true })
    await flushPromises()
    expect(document.body.textContent).toContain('tags.loading')

    resolveLoad({ tags: [] })
    await flushPromises()
    expect(document.body.textContent).toContain('tags.empty')
    expect(keyInputs()).toHaveLength(0)
  })

  it('renders loaded tags as rows', async () => {
    const w = mountDialog()
    await openDialog(w, [
      { key: 'env', value: 'prod' },
      { key: 'team', value: 's3' },
    ])
    expect(keyInputs().map((i) => i.value)).toEqual(['env', 'team'])
    expect(valueInputs().map((i) => i.value)).toEqual(['prod', 's3'])
  })

  it('emits error when loading tags fails', async () => {
    vi.mocked(s3api.getObjectTags).mockRejectedValue(new Error('tags-load-boom'))
    const w = mountDialog()
    await w.setProps({ open: true })
    await flushPromises()
    expect(w.emitted('error')).toEqual([['tags-load-boom']])
  })

  it('adds and removes tag rows, empty state returns', async () => {
    const w = mountDialog()
    await openDialog(w)
    clickBody('tags.add')
    await flushPromises()
    expect(keyInputs()).toHaveLength(1)
    expect(keyInputs()[0].value).toBe('')
    clickBody('tags.add')
    await flushPromises()
    expect(keyInputs()).toHaveLength(2)

    const removeBtns = Array.from(document.body.querySelectorAll('button')).filter(
      (b) => !b.classList.contains('dlg-x') && (b.textContent ?? '').trim() === '✕',
    )
    removeBtns[0].dispatchEvent(new MouseEvent('click', { bubbles: true }))
    await flushPromises()
    expect(keyInputs()).toHaveLength(1)
    clickBody('common.cancel')
  })

  it('submits tags (empty keys filtered), toasts updated and closes', async () => {
    const w = mountDialog()
    await openDialog(w)
    clickBody('tags.add')
    await flushPromises()
    fill(keyInputs()[0], ' env ')
    fill(valueInputs()[0], 'prod')
    clickBody('tags.add')
    await flushPromises()
    fill(keyInputs()[1], '  ')
    fill(valueInputs()[1], 'ignored')
    await flushPromises()

    clickBody('common.save')
    await flushPromises()
    expect(vi.mocked(s3api.putObjectTags)).toHaveBeenCalledWith('acc-1', {
      bucket: 'b',
      key: 'k',
      tags: [{ key: 'env', value: 'prod' }],
    })
    expect(toast).toHaveBeenCalledWith('tags.toastUpdated')
    expect(w.emitted('close')).toBeTruthy()
  })

  it('clearing all tags toasts cleared and closes', async () => {
    const w = mountDialog()
    await openDialog(w, [{ key: 'env', value: 'prod' }])
    // 删除唯一一行 → tags 为空
    const removeBtns = Array.from(document.body.querySelectorAll('button')).filter(
      (b) => !b.classList.contains('dlg-x') && (b.textContent ?? '').trim() === '✕',
    )
    removeBtns[0].dispatchEvent(new MouseEvent('click', { bubbles: true }))
    await flushPromises()
    clickBody('common.save')
    await flushPromises()
    expect(vi.mocked(s3api.putObjectTags)).toHaveBeenCalledWith('acc-1', {
      bucket: 'b',
      key: 'k',
      tags: [],
    })
    expect(toast).toHaveBeenCalledWith('tags.toastCleared')
    expect(w.emitted('close')).toBeTruthy()
  })

  it('saving guard: pending save blocks a second submit', async () => {
    type TagsResult = Awaited<ReturnType<typeof s3api.putObjectTags>>
    let resolvePut!: (v: TagsResult) => void
    const pending = new Promise<TagsResult>((resolve) => { resolvePut = resolve })
    vi.mocked(s3api.putObjectTags).mockImplementation(async () => pending)
    const w = mountDialog()
    await openDialog(w, [{ key: 'env', value: 'prod' }])
    clickBody('common.save')
    await flushPromises()
    expect(bodyBtn('common.save').disabled).toBe(true)
    await (w.vm as unknown as { submitTags: () => Promise<void> }).submitTags()
    expect(vi.mocked(s3api.putObjectTags)).toHaveBeenCalledTimes(1)
    resolvePut({ tags: [] })
    await flushPromises()
    expect(toast).toHaveBeenCalledWith('tags.toastUpdated')
  })

  it('emits error when saving tags fails', async () => {
    vi.mocked(s3api.putObjectTags).mockRejectedValue(new Error('tags-save-boom'))
    const w = mountDialog()
    await openDialog(w, [{ key: 'env', value: 'prod' }])
    clickBody('common.save')
    await flushPromises()
    expect(w.emitted('error')).toEqual([['tags-save-boom']])
    expect(w.emitted('close')).toBeUndefined()
  })

  it('cancel button emits close', async () => {
    const w = mountDialog()
    await openDialog(w)
    clickBody('common.cancel')
    expect(w.emitted('close')).toBeTruthy()
  })

  it('tags 缺省(null/undefined)→ `r.tags ?? []` 兜底渲染空态', async () => {
    const w = mountDialog()
    vi.mocked(s3api.getObjectTags).mockResolvedValue({} as never)
    await w.setProps({ open: true })
    await flushPromises()
    expect(w.emitted('error')).toBeUndefined()
    expect(document.body.textContent).toContain('tags.empty')
    expect(keyInputs()).toHaveLength(0)
  })

  it('open→false 走 watcher 的 early-return：不再重新拉取标签', async () => {
    const w = mountDialog()
    await openDialog(w, [{ key: 'env', value: 'prod' }])
    expect(vi.mocked(s3api.getObjectTags)).toHaveBeenCalledTimes(1)
    await w.setProps({ open: false })
    await flushPromises()
    expect(vi.mocked(s3api.getObjectTags)).toHaveBeenCalledTimes(1)
    expect(w.emitted('error')).toBeUndefined()
  })

  it('ModalDialog 自身的 close 事件转发为父级 close', async () => {
    const w = mountDialog()
    await openDialog(w)
    w.findComponent({ name: 'ModalDialog' }).vm.$emit('close')
    expect(w.emitted('close')).toBeTruthy()
  })
})
