import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import CreateBucketDialog from './CreateBucketDialog.vue'
import { s3api } from '../api'

vi.mock('../api', () => ({
  s3api: {
    createBucket: vi.fn(async () => ({ created: 'b1', region: 'r1', acl: 'private' })),
  },
}))

vi.mock('../i18n', () => ({
  t: (k: string) => k,
  tf: (k: string) => k,
  locale: () => 'en-US',
}))

function mountDialog() {
  return mount(CreateBucketDialog, {
    props: { open: false, accountId: 'acc-1' },
    attachTo: document.body,
  })
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

function nameInput(): HTMLInputElement {
  const el = document.body.querySelector('input') as HTMLInputElement | null
  expect(el).toBeTruthy()
  return el!
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

describe('CreateBucketDialog', () => {
  it('opens with reset form (empty name, private ACL)', async () => {
    const w = mountDialog()
    await w.setProps({ open: true })
    await flushPromises()
    expect(document.body.textContent).toContain('buckets.create')
    expect(nameInput().value).toBe('')
    expect(selectEl().selectedIndex).toBe(0)
    w.unmount()
  })

  it('rejects empty name with buckets.nameRequired error', async () => {
    const w = mountDialog()
    await w.setProps({ open: true })
    await flushPromises()
    clickBody('common.create')
    await flushPromises()
    expect(w.emitted('error')).toEqual([['buckets.nameRequired']])
    expect(vi.mocked(s3api.createBucket)).not.toHaveBeenCalled()
    w.unmount()
  })

  it('whitespace-only name is also rejected', async () => {
    const w = mountDialog()
    await w.setProps({ open: true })
    await flushPromises()
    nameInput().value = '   '
    nameInput().dispatchEvent(new Event('input'))
    await flushPromises()
    clickBody('common.create')
    await flushPromises()
    expect(w.emitted('error')).toEqual([['buckets.nameRequired']])
    w.unmount()
  })

  it('submits chosen ACL and emits created', async () => {
    vi.mocked(s3api.createBucket).mockResolvedValue({ created: 'b1', region: 'r1', acl: 'public-read' })
    const w = mountDialog()
    await w.setProps({ open: true })
    await flushPromises()
    nameInput().value = 'b1'
    nameInput().dispatchEvent(new Event('input'))
    selectIndex(selectEl(), 1)
    await flushPromises()
    clickBody('common.create')
    await flushPromises()
    expect(vi.mocked(s3api.createBucket)).toHaveBeenCalledWith('acc-1', {
      name: 'b1',
      acl: 'public-read',
    })
    expect(w.emitted('created')).toEqual([[{ name: 'b1', region: 'r1', acl: 'public-read' }]])
    w.unmount()
  })

  it('Enter key in name field submits', async () => {
    const w = mountDialog()
    await w.setProps({ open: true })
    await flushPromises()
    nameInput().value = 'b-enter'
    nameInput().dispatchEvent(new Event('input'))
    nameInput().dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter' }))
    await flushPromises()
    expect(vi.mocked(s3api.createBucket)).toHaveBeenCalledWith('acc-1', {
      name: 'b-enter',
      acl: 'private',
    })
    w.unmount()
  })

  it('emits error when createBucket fails', async () => {
    vi.mocked(s3api.createBucket).mockRejectedValue(new Error('create-boom'))
    const w = mountDialog()
    await w.setProps({ open: true })
    await flushPromises()
    nameInput().value = 'b1'
    nameInput().dispatchEvent(new Event('input'))
    await flushPromises()
    clickBody('common.create')
    await flushPromises()
    expect(w.emitted('error')).toEqual([['create-boom']])
    expect(w.emitted('created')).toBeUndefined()
    w.unmount()
  })

  it('reopening resets previously typed name and ACL', async () => {
    const w = mountDialog()
    await w.setProps({ open: true })
    await flushPromises()
    nameInput().value = 'b1'
    nameInput().dispatchEvent(new Event('input'))
    selectIndex(selectEl(), 2)
    await flushPromises()

    await w.setProps({ open: false })
    await w.setProps({ open: true })
    await flushPromises()
    expect(nameInput().value).toBe('')
    expect(selectEl().selectedIndex).toBe(0)
    expect(document.body.textContent).toContain('buckets.create')
    w.unmount()
  })

  it('cancel button emits close', async () => {
    const w = mountDialog()
    await w.setProps({ open: true })
    await flushPromises()
    clickBody('common.cancel')
    expect(w.emitted('close')).toBeTruthy()
    w.unmount()
  })

  it('ModalDialog 自身的 close 事件（遮罩/X）转发为父级 close', async () => {
    const w = mountDialog()
    await w.setProps({ open: true })
    await flushPromises()
    w.findComponent({ name: 'ModalDialog' }).vm.$emit('close')
    expect(w.emitted('close')).toBeTruthy()
    w.unmount()
  })

  it('异步提交防重复：在途时双击「创建」只发一次 createBucket', async () => {
    vi.mocked(s3api.createBucket).mockImplementationOnce(() => new Promise<never>(() => {})) // 请求挂起
    const w = mountDialog()
    await w.setProps({ open: true })
    await flushPromises()
    nameInput().value = 'b1'
    nameInput().dispatchEvent(new Event('input'))
    await flushPromises()
    clickBody('common.create')
    await flushPromises()
    expect(bodyBtn('common.create').disabled).toBe(true) // 在途时提交按钮禁用
    clickBody('common.create') // 第二次点击必须被守卫拦住
    await flushPromises()
    expect(vi.mocked(s3api.createBucket)).toHaveBeenCalledTimes(1)
    w.unmount()
  })

  it('提交在途时 Enter 再触发提交被守卫拦截；失败后发出 error 且按钮恢复', async () => {
    let rejectSave!: (e: Error) => void
    vi.mocked(s3api.createBucket).mockImplementationOnce(
      () =>
        new Promise<never>((_, rej) => {
          rejectSave = rej
        }),
    )
    const w = mountDialog()
    await w.setProps({ open: true })
    await flushPromises()
    nameInput().value = 'b1'
    nameInput().dispatchEvent(new Event('input'))
    await flushPromises()
    // Enter 键路径发起提交（按钮 disabled 无法覆盖该路径）
    nameInput().dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true }))
    await flushPromises()
    expect(vi.mocked(s3api.createBucket)).toHaveBeenCalledTimes(1)
    expect(bodyBtn('common.create').disabled).toBe(true)
    // 在途：Enter 路径再次触发提交入口，必须被 saving 守卫拦截
    nameInput().dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true }))
    await flushPromises()
    expect(vi.mocked(s3api.createBucket)).toHaveBeenCalledTimes(1)
    // 失败 settle：error 事件 + 按钮恢复
    rejectSave(new Error('create-boom'))
    await flushPromises()
    expect(w.emitted('error')).toEqual([['create-boom']])
    expect(bodyBtn('common.create').disabled).toBe(false)
    w.unmount()
  })
})
