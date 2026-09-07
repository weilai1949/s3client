import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import BatchMetadataDialog from './BatchMetadataDialog.vue'
import { batchSetMetadata, type BatchMetaResult } from '../batchMetadata'
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
