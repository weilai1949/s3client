import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import ObjectDetailDialog from './ObjectDetailDialog.vue'
import ModalDialog from './ModalDialog.vue'
import type { ObjectMeta } from '../types'

vi.mock('../i18n', () => ({
  t: (k: string) => k,
  tf: (k: string) => k,
}))

// ModalDialog stub：原地渲染默认插槽，便于从 wrapper 断言对话框内容并驱动 close。
const ModalDialogStub = {
  name: 'ModalDialog',
  template: '<div class="dlg-stub"><slot /></div>',
}

function mountDialog(detail: ObjectMeta | null) {
  return mount(ObjectDetailDialog, {
    props: { open: true, detail },
    global: { stubs: { ModalDialog: ModalDialogStub } },
  })
}

const fullDetail: ObjectMeta = {
  key: 'docs/report.pdf',
  size: 2048,
  lastModified: '2024-06-01T10:00:00Z',
  etag: '"abc123"',
  contentType: 'application/pdf',
  storageClass: 'STANDARD',
  metadata: { author: 'tester', env: 'prod' },
}

describe('ObjectDetailDialog', () => {
  it('renders nothing inside dialog when detail is null', () => {
    const w = mountDialog(null)
    expect(w.find('.detail-tbl').exists()).toBe(false)
  })

  it('renders all fields and metadata for a full detail', () => {
    const w = mountDialog(fullDetail)
    const table = w.find('.detail-tbl')
    expect(table.exists()).toBe(true)
    expect(table.text()).toContain('docs/report.pdf')
    expect(table.text()).toContain('2.0 KB')
    // storageClassLabel('STANDARD')：i18n mock 回显 key → 原样展示
    expect(table.text()).toContain('STANDARD')
    expect(table.text()).toContain('application/pdf')
    expect(table.text()).toContain('abc123')
    // metadata 行
    expect(table.text()).toContain('author: tester')
    expect(table.text()).toContain('env: prod')
  })

  it('falls back to em dash for missing fields and hides metadata row', () => {
    const w = mountDialog({
      key: 'plain',
      size: 0,
      lastModified: '',
      etag: '',
      contentType: '',
    })
    const table = w.find('.detail-tbl')
    expect(table.text()).toContain('—')
    const metaRow = table.findAll('tr').find((tr) => tr.text().includes('detail.metadata'))
    expect(metaRow).toBeFalsy()
  })

  it('emits editHeaders/openAcl/openTags/openVersions/openStorageClass with detail key', async () => {
    const w = mountDialog(fullDetail)
    const byText = (text: string) => {
      const btn = w.findAll('button').find((b) => b.text() === text)
      expect(btn, `button ${text}`).toBeTruthy()
      return btn!
    }
    await byText('detail.editHeaders').trigger('click')
    await byText('detail.acl').trigger('click')
    await byText('detail.tags').trigger('click')
    await byText('detail.versions').trigger('click')
    await byText('detail.switch').trigger('click')
    expect(w.emitted('editHeaders')).toEqual([[fullDetail.key]])
    expect(w.emitted('openAcl')).toEqual([[fullDetail.key]])
    expect(w.emitted('openTags')).toEqual([[fullDetail.key]])
    expect(w.emitted('openVersions')).toEqual([[fullDetail.key]])
    expect(w.emitted('openStorageClass')).toEqual([[fullDetail.key]])
  })

  it('emits close when ModalDialog emits close', async () => {
    const w = mountDialog(fullDetail)
    const dlg = w.findComponent(ModalDialog)
    expect(dlg.exists()).toBe(true)
    dlg.vm.$emit('close')
    await w.vm.$nextTick()
    expect(w.emitted('close')).toBeTruthy()
  })

  it('不声明从未 emit 的 error 事件与零使用的 accountId/bucket props（F5）', () => {
    const w = mountDialog(fullDetail)
    const emits = w.vm.$options.emits as string[]
    const props = Object.keys(w.vm.$options.props as Record<string, unknown>)
    expect(emits).not.toContain('error')
    expect(props).not.toContain('accountId')
    expect(props).not.toContain('bucket')
    expect(props).toEqual(['open', 'detail'])
  })
})
