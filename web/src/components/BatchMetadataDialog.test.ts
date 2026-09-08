import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import BatchMetadataDialog from './BatchMetadataDialog.vue'
import { batchSetMetadata, type BatchMetaError, type BatchMetaInput, type BatchMetaResult } from '../batchMetadata'
import { toast } from '../store'

vi.mock('../batchMetadata', () => ({
  batchSetMetadata: vi.fn(),
}))

vi.mock('../store', () => ({
  toast: vi.fn(),
}))

vi.mock('../i18n', () => ({
  t: (k: string) => k,
  tf: (k: string) => k,
}))

// ModalDialog 组件缺少 footer slot（生产缺陷）：测试用自定义 stub 同时渲染
// 默认主体与 footer 插槽，以便驱动确定/取消按钮。
const ModalDialogStub = {
  name: 'ModalDialog',
  template: '<div><slot /><slot name="footer" /></div>',
}

function mountDialog(keys: string[]) {
  return mount(BatchMetadataDialog, {
    props: { accountId: 'acc-1', bucket: 'b1', keys },
    global: { stubs: { ModalDialog: ModalDialogStub } },
  })
}

function confirmBtn(w: ReturnType<typeof mountDialog>) {
  const btn = w.findAll('button').find((b) => b.text() === 'batchEdit.confirm')
  expect(btn, 'confirm button should exist').toBeTruthy()
  return btn!
}

beforeEach(() => {
  vi.clearAllMocks()
})

describe('BatchMetadataDialog', () => {
  it('noChange 判定：全空 tag 时提交禁用，填写后启用', async () => {
    const w = mountDialog(['k1', 'k2'])
    // 初始未勾选任何修改 → 禁用
    expect(confirmBtn(w).attributes('disabled')).toBeDefined()

    // 勾选 ACL → 启用
    await w.find('[data-testid="batch-edit-acl-toggle"]').setValue(true)
    expect(confirmBtn(w).attributes('disabled')).toBeUndefined()

    // 取消 ACL，改勾选标签（clear 是修改 → 启用）
    await w.find('[data-testid="batch-edit-acl-toggle"]').setValue(false)
    await w.find('[data-testid="batch-edit-tags-toggle"]').setValue(true)
    const modeSel = w.find('select[aria-label="Tag mode"]')
    await modeSel.setValue('clear')
    expect(confirmBtn(w).attributes('disabled')).toBeUndefined()

    // replace 模式 + 全空 tag 行 → 无实际修改 → 禁用
    await modeSel.setValue('replace')
    const addBtn = w.findAll('button').find((b) => b.text() === '+')
    expect(addBtn).toBeTruthy()
    await addBtn!.trigger('click')
    const keyInput = w.find('.tag-row input[id^="batch-tag-key-"]')
    expect((keyInput.element as HTMLInputElement).value).toBe('')
    expect(confirmBtn(w).attributes('disabled')).toBeDefined()

    // 填入 key → 启用
    await keyInput.setValue('env')
    expect(confirmBtn(w).attributes('disabled')).toBeUndefined()
  })

  it('提交后渲染进度条，完成后 emit done 并 toast', async () => {
    let resolveFn!: (v: BatchMetaResult) => void
    const pending = new Promise<BatchMetaResult>((resolve) => {
      resolveFn = resolve
    })
    vi.mocked(batchSetMetadata).mockImplementation(async (input) => {
      input.onProgress?.(1, input.keys.length)
      return pending
    })

    const w = mountDialog(['k1', 'k2'])
    await w.find('[data-testid="batch-edit-acl-toggle"]').setValue(true)
    await confirmBtn(w).trigger('click')
    await nextTick()

    // 进行中：进度条 1/2
    const bar = w.find('progress.progress.bar')
    expect(bar.exists()).toBe(true)
    expect(bar.attributes('value')).toBe('1')
    expect(bar.attributes('max')).toBe('2')

    // 完成：结果文案 + done 事件 + ok toast
    resolveFn({ ok: 2, failed: 0, errors: [] })
    await flushPromises()
    await nextTick()
    expect(w.text()).toContain('batchEdit.done')
    expect(w.emitted('done')).toEqual([[{ ok: 2, failed: 0 }]])
    expect(toast).toHaveBeenCalledWith('batchEdit.done', 'ok')
  })

  it('keys 为空时提示且不发起请求', async () => {
    const w = mountDialog([])
    await w.find('[data-testid="batch-edit-acl-toggle"]').setValue(true)
    await confirmBtn(w).trigger('click')
    expect(toast).toHaveBeenCalledWith('batchEdit.empty', 'err')
    expect(batchSetMetadata).not.toHaveBeenCalled()
  })
})

describe('BatchMetadataDialog extra branches', () => {
  it('fatal throw → 通用错误 + done{ok:0} + errors 渲染', async () => {
    vi.mocked(batchSetMetadata).mockRejectedValue(new Error('too many keys'))
    const w = mountDialog(['k1', 'k2'])
    await w.find('[data-testid="batch-edit-acl-toggle"]').setValue(true)
    await confirmBtn(w).trigger('click')
    await flushPromises()
    expect(w.emitted('done')).toEqual([[{ ok: 0, failed: 2 }]])
    expect(toast).toHaveBeenCalledWith('batchEdit.fatalError', 'err')
    expect(w.find('.errors').exists()).toBe(true)
    expect(w.text()).toContain('too many keys')
  })

  it('done with failures → err toast + done with failed count', async () => {
    vi.mocked(batchSetMetadata).mockResolvedValue({
      ok: 1, failed: 1, errors: [{ key: 'k2', step: 'acl', message: 'denied' }],
    })
    const w = mountDialog(['k1', 'k2'])
    await w.find('[data-testid="batch-edit-acl-toggle"]').setValue(true)
    await confirmBtn(w).trigger('click')
    await flushPromises()
    expect(w.emitted('done')).toEqual([[{ ok: 1, failed: 1 }]])
    expect(toast).toHaveBeenCalledWith('batchEdit.done', 'err')
    expect(w.text()).toContain('denied')
  })

  it('tags clear → 传空 tags；replace → 过滤带 key 的行', async () => {
    const input: BatchMetaInput[] = []
    vi.mocked(batchSetMetadata).mockImplementation(async (i) => {
      input.push(i)
      return { ok: 2, failed: 0, errors: [] }
    })
    const w = mountDialog(['k1', 'k2'])
    await w.find('[data-testid="batch-edit-tags-toggle"]').setValue(true)
    const modeSel = w.find('select[aria-label="Tag mode"]')
    await modeSel.setValue('clear')
    await confirmBtn(w).trigger('click')
    await flushPromises()
    expect(input[0].tags).toEqual([])

    // replace：一行有 key、一行只有 value → 只传有 key 的行
    await modeSel.setValue('replace')
    const addBtn = w.findAll('button').find((b) => b.text() === '+')!
    await addBtn.trigger('click')
    await w.find('.tag-row input[id^="batch-tag-key-"]').setValue('env')
    await addBtn.trigger('click')
    await w.find('.tag-row input[id^="batch-tag-key-"][id$="1"]').setValue('')
    await w.find('.tag-row input[id^="batch-tag-val-"][id$="1"]').setValue('orphan')
    await confirmBtn(w).trigger('click')
    await flushPromises()
    expect(input[1].tags).toEqual([{ key: 'env', value: '' }])
  })

  it('替换模式仅有 value 无 key → tagsNeedKey 校验', async () => {
    const w = mountDialog(['k1'])
    await w.find('[data-testid="batch-edit-tags-toggle"]').setValue(true)
    const modeSel = w.find('select[aria-label="Tag mode"]')
    await modeSel.setValue('replace')
    const addBtn = w.findAll('button').find((b) => b.text() === '+')!
    await addBtn.trigger('click')
    await w.find('.tag-row input[id^="batch-tag-val-"]').setValue('orphan')
    await confirmBtn(w).trigger('click')
    expect(toast).toHaveBeenCalledWith('batchEdit.tagsNeedKey', 'err')
    expect(batchSetMetadata).not.toHaveBeenCalled()
  })

  it('noChange（未勾选任何修改，防御分支）→ 提示需要选择', async () => {
    const w = mountDialog(['k1'])
    // UI 上按钮 disabled；防御分支直接用 vm 触发
    await (w.vm as unknown as { onConfirm: () => Promise<void> }).onConfirm()
    expect(toast).toHaveBeenCalledWith('batchEdit.needStep', 'err')
    expect(batchSetMetadata).not.toHaveBeenCalled()
  })

  it('removeTagRow 删除行 + 关闭时 running 守卫', async () => {
    let resolveFn!: (v: BatchMetaResult) => void
    vi.mocked(batchSetMetadata).mockImplementation(async () => new Promise((resolve) => { resolveFn = resolve }))
    const w = mountDialog(['k1'])
    await w.find('[data-testid="batch-edit-tags-toggle"]').setValue(true)
    await w.find('select[aria-label="Tag mode"]').setValue('replace')
    const addBtn = w.findAll('button').find((b) => b.text() === '+')!
    await addBtn.trigger('click')
    await addBtn.trigger('click')
    expect(w.findAll('.tag-row')).toHaveLength(2)
    await w.findAll('.tag-row button.danger')[0].trigger('click')
    expect(w.findAll('.tag-row')).toHaveLength(1)
    await w.find('.tag-row input[id^="batch-tag-key-"]').setValue('env')

    await w.find('[data-testid="batch-edit-acl-toggle"]').setValue(true)
    await confirmBtn(w).trigger('click')
    await nextTick()
    // running 中 close 无效（防御分支走 vm）
    ;(w.vm as unknown as { close: () => void }).close()
    expect(w.emitted('close')).toBeUndefined()
    resolveFn({ ok: 1, failed: 0, errors: [] })
    await flushPromises()
    ;(w.vm as unknown as { close: () => void }).close()
    expect(w.emitted('close')).toBeTruthy()
  })

  it('错误超过 50 条可展开', async () => {
    const errors: BatchMetaError[] = Array.from({ length: 60 }, (_, i) => ({ key: `k${i}`, step: 'acl', message: `m${i}` }))
    vi.mocked(batchSetMetadata).mockResolvedValue({ ok: 0, failed: 60, errors })
    const w = mountDialog(Array.from({ length: 60 }, (_, i) => `k${i}`))
    await w.find('[data-testid="batch-edit-acl-toggle"]').setValue(true)
    await confirmBtn(w).trigger('click')
    await flushPromises()
    expect(w.text()).toContain('batchEdit.errors')
    const toggle = w.findAll('button').find((b) => b.text() === 'batchEdit.showMore')
    expect(toggle).toBeTruthy()
    await toggle!.trigger('click')
    expect(w.text()).toContain('batchEdit.showLess')
  })

  it('ACL select 与 storageClass 输入 v-model 绑定进入提交 payload', async () => {
    const inputs: BatchMetaInput[] = []
    vi.mocked(batchSetMetadata).mockImplementation(async (i) => {
      inputs.push(i)
      return { ok: 1, failed: 0, errors: [] }
    })
    const w = mountDialog(['k1'])
    // 勾选 ACL → 下拉可选；改选 public-read
    await w.find('[data-testid="batch-edit-acl-toggle"]').setValue(true)
    await w.find('select[aria-label="ACL"]').setValue('public-read')
    // 勾选存储类切换 → 输入框可编辑
    await w.find('[data-testid="batch-edit-storage-toggle"]').setValue(true)
    await w.find('input[aria-label="Storage class"]').setValue('GLACIER')
    await confirmBtn(w).trigger('click')
    await flushPromises()
    expect(inputs[0]).toMatchObject({ acl: 'public-read', storageClass: 'GLACIER' })
    expect(inputs[0].keys).toEqual(['k1'])
  })

  it('running 中再次 onConfirm 直接返回：不发起第二次 batch', async () => {
    let resolveFn!: (v: BatchMetaResult) => void
    vi.mocked(batchSetMetadata).mockImplementation(async (input) => {
      input.onProgress?.(1, input.keys.length)
      return new Promise<BatchMetaResult>((resolve) => { resolveFn = resolve })
    })
    const w = mountDialog(['k1', 'k2'])
    await w.find('[data-testid="batch-edit-acl-toggle"]').setValue(true)
    await confirmBtn(w).trigger('click') // running → true
    await nextTick()
    // 进行中直调 onConfirm：running 守卫直接返回
    await (w.vm as unknown as { onConfirm: () => Promise<void> }).onConfirm()
    expect(vi.mocked(batchSetMetadata)).toHaveBeenCalledTimes(1)
    resolveFn({ ok: 2, failed: 0, errors: [] })
    await flushPromises()
    await nextTick()
  })

  it('open 显式传 undefined 时 computed 回退为 true', () => {
    const w = mount(BatchMetadataDialog, {
      props: { accountId: 'acc-1', bucket: 'b1', keys: ['k'], open: undefined },
      global: { stubs: { ModalDialog: ModalDialogStub } },
    })
    expect((w.vm as unknown as { open: boolean }).open).toBe(true)
  })

  it('open 显式传 false 时 computed 不回退为 true', () => {
    const w = mount(BatchMetadataDialog, {
      props: { accountId: 'acc-1', bucket: 'b1', keys: ['k'], open: false },
      global: { stubs: { ModalDialog: ModalDialogStub } },
    })
    expect((w.vm as unknown as { open: boolean }).open).toBe(false)
  })

  it('tags 勾选但模式为 none 时不传 tags 参数', async () => {
    const input: BatchMetaInput[] = []
    vi.mocked(batchSetMetadata).mockImplementation(async (i) => {
      input.push(i)
      return { ok: 1, failed: 0, errors: [] }
    })
    const w = mountDialog(['k1'])
    // 勾选 tags 但保持默认 none（配合 ACL 使 noChange=false）
    await w.find('[data-testid="batch-edit-tags-toggle"]').setValue(true)
    await w.find('[data-testid="batch-edit-acl-toggle"]').setValue(true)
    await confirmBtn(w).trigger('click')
    await flushPromises()
    expect(input[0].tags).toBeUndefined()
  })

  it('运行中 total 为 0 时进度条 max 回退 1', async () => {
    let resolveFn!: (v: BatchMetaResult) => void
    vi.mocked(batchSetMetadata).mockImplementation(async () => new Promise<BatchMetaResult>((resolve) => { resolveFn = resolve }))
    const w = mountDialog(['k1'])
    await w.find('[data-testid="batch-edit-acl-toggle"]').setValue(true)
    await confirmBtn(w).trigger('click')
    await nextTick()
    // 强制 total=0 渲染：progress.total || 1 → max=1
    ;(w.vm as unknown as { progress: { total: number } }).progress.total = 0
    await nextTick()
    expect(w.find('progress.progress.bar').attributes('max')).toBe('1')
    resolveFn({ ok: 1, failed: 0, errors: [] })
    await flushPromises()
  })
})
