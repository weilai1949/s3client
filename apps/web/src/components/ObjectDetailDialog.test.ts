import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import ObjectDetailDialog from './ObjectDetailDialog.vue'
import ModalDialog from './ModalDialog.vue'
import { s3api } from '../api'
import type { ObjectMeta, ObjectRetention, VerifyResult } from '../types'

vi.mock('../api', () => ({
  s3api: {
    getObjectRetention: vi.fn(),
    getObjectLegalHold: vi.fn(),
    putObjectRetention: vi.fn(),
    putObjectLegalHold: vi.fn(),
    verifyChecksum: vi.fn(),
  },
}))

vi.mock('../i18n', () => ({
  t: (k: string) => k,
  tf: (k: string) => k,
}))

// ModalDialog stub：原地渲染默认插槽，便于从 wrapper 断言对话框内容并驱动 close。
const ModalDialogStub = {
  name: 'ModalDialog',
  template: '<div class="dlg-stub"><slot /></div>',
}

/** 挂载参数：对象保护读写需要账号 / 桶（后端按此寻址）。 */
interface MountOpts {
  open?: boolean
  detail?: ObjectMeta | null
}

function mountDialog(opts: MountOpts = {}) {
  return mount(ObjectDetailDialog, {
    props: {
      open: opts.open ?? true,
      detail: opts.detail === undefined ? null : opts.detail,
      accountId: 'acc-1',
      bucket: 'b1',
    },
    global: { stubs: { ModalDialog: ModalDialogStub } },
  })
}

/** 按按钮文本找按钮（模板里的插值可能带换行空白）。 */
function btn(w: ReturnType<typeof mountDialog>, text: string) {
  const b = w.findAll('button').find((x) => x.text().trim() === text)
  expect(b, `button "${text}" should exist`).toBeTruthy()
  return b!
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

const plainDetail: ObjectMeta = {
  key: 'plain',
  size: 0,
  lastModified: '',
  etag: '',
  contentType: '',
}

/** 有保留期 + 法定保留 ON 的保护状态。 */
const retentionOn: ObjectRetention = {
  bucket: 'b1',
  key: 'docs/report.pdf',
  versionId: '',
  configured: true,
  mode: 'COMPLIANCE',
  retainUntilDate: '2031-02-03T04:05:06Z',
}

const retentionOff: ObjectRetention = {
  bucket: 'b1',
  key: 'plain',
  versionId: '',
  configured: false,
  mode: '',
  retainUntilDate: '',
}

function verifyWith(patch: Partial<VerifyResult>): VerifyResult {
  return { bucket: 'b1', key: 'docs/report.pdf', versionId: '', method: 'sha256', local: 'aa', remote: 'aa', match: true, ...patch }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(s3api.getObjectRetention).mockResolvedValue(retentionOff)
  vi.mocked(s3api.getObjectLegalHold).mockResolvedValue({ bucket: 'b1', key: '', versionId: '', status: 'OFF' })
  vi.mocked(s3api.putObjectRetention).mockResolvedValue(retentionOn)
  vi.mocked(s3api.putObjectLegalHold).mockResolvedValue({ bucket: 'b1', key: '', versionId: '', status: 'ON' })
  vi.mocked(s3api.verifyChecksum).mockResolvedValue(verifyWith({}))
})

describe('ObjectDetailDialog', () => {
  it('renders nothing inside dialog when detail is null', async () => {
    const w = mountDialog({ detail: null })
    expect(w.find('.detail-tbl').exists()).toBe(false)
    await flushPromises()
    // detail 为空时不发起对象保护读取（避免空 key 打到后端）
    expect(s3api.getObjectRetention).not.toHaveBeenCalled()
    expect(s3api.getObjectLegalHold).not.toHaveBeenCalled()
  })

  it('renders all fields and metadata for a full detail', () => {
    const w = mountDialog({ detail: fullDetail })
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
    const w = mountDialog({ detail: plainDetail })
    const table = w.find('.detail-tbl')
    expect(table.text()).toContain('—')
    const metaRow = table.findAll('tr').find((tr) => tr.text().includes('detail.metadata'))
    expect(metaRow).toBeFalsy()
  })

  it('emits editHeaders/openAcl/openTags/openVersions/openStorageClass with detail key', async () => {
    const w = mountDialog({ detail: fullDetail })
    await btn(w, 'detail.editHeaders').trigger('click')
    await btn(w, 'detail.acl').trigger('click')
    await btn(w, 'detail.tags').trigger('click')
    await btn(w, 'detail.versions').trigger('click')
    await btn(w, 'detail.switch').trigger('click')
    expect(w.emitted('editHeaders')).toEqual([['docs/report.pdf']])
    expect(w.emitted('openAcl')).toEqual([['docs/report.pdf']])
    expect(w.emitted('openTags')).toEqual([['docs/report.pdf']])
    expect(w.emitted('openVersions')).toEqual([['docs/report.pdf']])
    expect(w.emitted('openStorageClass')).toEqual([['docs/report.pdf']])
  })

  it('emits close when ModalDialog emits close', async () => {
    const w = mountDialog({ detail: fullDetail })
    const dlg = w.findComponent(ModalDialog)
    expect(dlg.exists()).toBe(true)
    dlg.vm.$emit('close')
    await w.vm.$nextTick()
    expect(w.emitted('close')).toBeTruthy()
  })

  it('声明的 props 恰为 open/detail/accountId/bucket，且不声明从未 emit 的 error 事件（F5）', () => {
    const w = mountDialog({ detail: fullDetail })
    const emits = w.vm.$options.emits as string[]
    const props = Object.keys(w.vm.$options.props as Record<string, unknown>)
    expect(emits).not.toContain('error')
    expect(props.sort()).toEqual(['accountId', 'bucket', 'detail', 'open'])
  })
})

describe('ObjectDetailDialog 校验和', () => {
  it('有校验和：逐条展示算法与取值', () => {
    const w = mountDialog({ detail: { ...plainDetail, checksums: { crc64nvme: 'c64', sha256: 's256', type: 'FULL_OBJECT' } } })
    const text = w.find('.detail-tbl').text()
    expect(text).toContain('crc64nvme: c64')
    expect(text).toContain('sha256: s256')
    expect(text).toContain('type: FULL_OBJECT')
    expect(text).not.toContain('detail.checksumsNone')
  })

  it('空值字段不渲染，全部为空则回落到「无校验和」', () => {
    const rows = mountDialog({ detail: { ...plainDetail, checksums: { crc64nvme: '', sha256: 's256' } } })
    expect(rows.text()).toContain('sha256: s256')
    expect(rows.text()).not.toContain('crc64nvme:')

    const none = mountDialog({ detail: { ...plainDetail, checksums: null } })
    expect(none.text()).toContain('detail.checksumsNone')
  })

  it('校验校验和：显示算法 / 本地值 / 远端值 / 一致', async () => {
    vi.mocked(s3api.verifyChecksum).mockResolvedValue(verifyWith({ method: 'sha256', local: 'abc', remote: 'abc', match: true }))
    const w = mountDialog({ detail: fullDetail })
    await btn(w, 'detail.verify').trigger('click')
    await flushPromises()
    expect(s3api.verifyChecksum).toHaveBeenCalledWith('acc-1', { bucket: 'b1', key: 'docs/report.pdf' })
    const text = w.find('.detail-tbl').text()
    expect(text).toContain('detail.verifyMethod: sha256')
    expect(text).toContain('detail.verifyLocal: abc')
    expect(text).toContain('detail.verifyRemote: abc')
    expect(text).toContain('detail.verifyMatch')
    expect(text).not.toContain('detail.verifyMismatch')
  })

  it('校验不一致：显示不一致', async () => {
    vi.mocked(s3api.verifyChecksum).mockResolvedValue(verifyWith({ match: false }))
    const w = mountDialog({ detail: fullDetail })
    await btn(w, 'detail.verify').trigger('click')
    await flushPromises()
    expect(w.find('.detail-tbl').text()).toContain('detail.verifyMismatch')
  })

  it('method=none：显示「无可验证来源」（如实降级，不是错误）', async () => {
    vi.mocked(s3api.verifyChecksum).mockResolvedValue(verifyWith({ method: 'none', local: '', remote: '', match: false }))
    const w = mountDialog({ detail: fullDetail })
    await btn(w, 'detail.verify').trigger('click')
    await flushPromises()
    const text = w.find('.detail-tbl').text()
    expect(text).toContain('detail.verifyNone')
    expect(text).not.toContain('detail.verifyMethod')
  })

  it('校验失败：错误就地展示', async () => {
    vi.mocked(s3api.verifyChecksum).mockRejectedValue(new Error('501 not implemented'))
    const w = mountDialog({ detail: fullDetail })
    await btn(w, 'detail.verify').trigger('click')
    await flushPromises()
    expect(w.find('.detail-tbl').text()).toContain('501 not implemented')
  })
})

describe('ObjectDetailDialog 保留期与法定保留读取', () => {
  it('关闭时不读取；打开后按账号/桶/key 读取并回显', async () => {
    const w = mountDialog({ detail: fullDetail, open: false })
    await flushPromises()
    expect(s3api.getObjectRetention).not.toHaveBeenCalled()

    await w.setProps({ open: true })
    await flushPromises()
    expect(s3api.getObjectRetention).toHaveBeenCalledWith('acc-1', { bucket: 'b1', key: 'docs/report.pdf' })
    expect(s3api.getObjectLegalHold).toHaveBeenCalledWith('acc-1', { bucket: 'b1', key: 'docs/report.pdf' })
    // 未设置保留期 + 法定保留 OFF → 空态文案与默认表单
    expect(w.find('.detail-tbl').text()).toContain('detail.retentionNone')
    expect(w.find('.detail-tbl').text()).toContain('detail.legalHoldOff')
    expect((w.find('select').element as HTMLSelectElement).value).toBe('GOVERNANCE')
    expect((w.find('input[type="datetime-local"]').element as HTMLInputElement).value).toBe('')
  })

  it('已设置保留期 + 法定保留 ON：回显模式、到期时间与表单', async () => {
    vi.mocked(s3api.getObjectRetention).mockResolvedValue(retentionOn)
    vi.mocked(s3api.getObjectLegalHold).mockResolvedValue({ bucket: 'b1', key: '', versionId: '', status: 'ON' })
    const w = mountDialog({ detail: fullDetail })
    await flushPromises()
    const text = w.find('.detail-tbl').text()
    expect(text).toContain('detail.retention')
    expect(text).toContain('COMPLIANCE')
    expect(text).not.toContain('detail.retentionNone')
    expect(text).toContain('detail.legalHoldOn')
    expect((w.find('select').element as HTMLSelectElement).value).toBe('COMPLIANCE')
    // 到期时间按本地时区喂给 datetime-local（具体值随运行环境时区）
    expect((w.find('input[type="datetime-local"]').element as HTMLInputElement).value).toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/)
  })

  it('读取失败：错误就地展示', async () => {
    vi.mocked(s3api.getObjectRetention).mockRejectedValue(new Error('bucket has no object lock'))
    const w = mountDialog({ detail: fullDetail })
    await flushPromises()
    expect(w.find('.detail-tbl').text()).toContain('bucket has no object lock')
  })

  it('快速切换对象时过期响应被丢弃（以最后一次读取为准）', async () => {
    let release: (v: ObjectRetention) => void = () => {}
    const pending = new Promise<ObjectRetention>((resolve) => {
      release = resolve
    })
    vi.mocked(s3api.getObjectRetention)
      .mockReturnValueOnce(pending)
      .mockResolvedValue(retentionOn)
    vi.mocked(s3api.getObjectLegalHold).mockResolvedValue({ bucket: 'b1', key: '', versionId: '', status: 'ON' })

    const w = mountDialog({ detail: plainDetail })
    await flushPromises() // 第一次读取挂起中
    await w.setProps({ detail: fullDetail })
    await flushPromises() // 第二次读取完成（COMPLIANCE）
    expect((w.find('select').element as HTMLSelectElement).value).toBe('COMPLIANCE')

    release(retentionOff) // 过期响应后到：不得覆盖新状态
    await flushPromises()
    expect((w.find('select').element as HTMLSelectElement).value).toBe('COMPLIANCE')
  })
})

describe('ObjectDetailDialog 保留期与法定保留写入', () => {
  it('设置未来保留期 → PUT 成功并回显新状态', async () => {
    const w = mountDialog({ detail: fullDetail })
    await flushPromises()
    await w.find('select').setValue('COMPLIANCE')
    await w.find('input[type="datetime-local"]').setValue('2031-02-03T04:05')
    await btn(w, 'detail.retentionSave').trigger('click')
    await flushPromises()
    expect(s3api.putObjectRetention).toHaveBeenCalledWith('acc-1', {
      bucket: 'b1',
      key: 'docs/report.pdf',
      mode: 'COMPLIANCE',
      retainUntilDate: new Date('2031-02-03T04:05').toISOString(),
    })
    const text = w.find('.detail-tbl').text()
    expect(text).toContain('COMPLIANCE')
    expect(text).not.toContain('detail.retentionInvalid')
  })

  it('保留期为空 → 本地拦截，不发请求', async () => {
    const w = mountDialog({ detail: fullDetail })
    await flushPromises()
    await w.find('input[type="datetime-local"]').setValue('')
    await btn(w, 'detail.retentionSave').trigger('click')
    await flushPromises()
    expect(s3api.putObjectRetention).not.toHaveBeenCalled()
    expect(w.find('.detail-tbl').text()).toContain('detail.retentionInvalid')
  })

  it('保留期是过去时间 → 本地拦截，不发请求', async () => {
    const w = mountDialog({ detail: fullDetail })
    await flushPromises()
    await w.find('input[type="datetime-local"]').setValue('2000-01-01T00:00')
    await btn(w, 'detail.retentionSave').trigger('click')
    await flushPromises()
    expect(s3api.putObjectRetention).not.toHaveBeenCalled()
    expect(w.find('.detail-tbl').text()).toContain('detail.retentionInvalid')
  })

  it('PUT 保留期失败（400/403/409）：原样展示后端文案', async () => {
    vi.mocked(s3api.putObjectRetention).mockRejectedValue(new Error('409 ObjectLocked'))
    const w = mountDialog({ detail: fullDetail })
    await flushPromises()
    await w.find('input[type="datetime-local"]').setValue('2031-02-03T04:05')
    await btn(w, 'detail.retentionSave').trigger('click')
    await flushPromises()
    expect(w.find('.detail-tbl').text()).toContain('409 ObjectLocked')
  })

  it('法定保留 OFF → 开启：PUT status=ON 并回显 ON', async () => {
    const w = mountDialog({ detail: fullDetail })
    await flushPromises()
    expect(w.find('.detail-tbl').text()).toContain('detail.legalHoldOff')
    await btn(w, 'detail.legalHoldTurnOn').trigger('click')
    await flushPromises()
    expect(s3api.putObjectLegalHold).toHaveBeenCalledWith('acc-1', {
      bucket: 'b1',
      key: 'docs/report.pdf',
      status: 'ON',
    })
    expect(w.find('.detail-tbl').text()).toContain('detail.legalHoldOn')
    expect(w.find('.detail-tbl').text()).toContain('detail.legalHoldTurnOff')
  })

  it('法定保留 ON → 关闭：PUT status=OFF 并回显 OFF', async () => {
    vi.mocked(s3api.getObjectLegalHold).mockResolvedValue({ bucket: 'b1', key: '', versionId: '', status: 'ON' })
    vi.mocked(s3api.putObjectLegalHold).mockResolvedValue({ bucket: 'b1', key: '', versionId: '', status: 'OFF' })
    const w = mountDialog({ detail: fullDetail })
    await flushPromises()
    await btn(w, 'detail.legalHoldTurnOff').trigger('click')
    await flushPromises()
    expect(s3api.putObjectLegalHold).toHaveBeenCalledWith('acc-1', {
      bucket: 'b1',
      key: 'docs/report.pdf',
      status: 'OFF',
    })
    expect(w.find('.detail-tbl').text()).toContain('detail.legalHoldOff')
  })

  it('PUT 法定保留失败：错误就地展示', async () => {
    vi.mocked(s3api.putObjectLegalHold).mockRejectedValue(new Error('403 AccessDenied'))
    const w = mountDialog({ detail: fullDetail })
    await flushPromises()
    await btn(w, 'detail.legalHoldTurnOn').trigger('click')
    await flushPromises()
    expect(w.find('.detail-tbl').text()).toContain('403 AccessDenied')
  })
})
