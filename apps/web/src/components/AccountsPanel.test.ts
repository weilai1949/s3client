import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import AccountsPanel from './AccountsPanel.vue'
import ModalDialog from './ModalDialog.vue'
import { s3api } from '../api'
import { state, toast, selectAccount, requestAccountForm } from '../store'
import { confirmDialog } from '../confirm'
import type { Account, AccountInput } from '../types'

vi.mock('../api', () => ({
  s3api: {
    listAccounts: vi.fn(async () => ({ accounts: [] })),
    createAccount: vi.fn(async () => ({ id: 'new-1' })),
    updateAccount: vi.fn(async () => ({ id: 'acc-1' })),
    deleteAccount: vi.fn(async () => ({ deleted: 'acc-1' })),
    testAccount: vi.fn(async () => ({ ok: true, bucket: 'b1' })),
    previewBuckets: vi.fn(async () => ({ buckets: [] })),
    listBuckets: vi.fn(async () => ({ buckets: [] })),
  },
}))

vi.mock('../store', async () => {
  const { reactive } = await import('vue')
  const accountFormRequest = reactive({ seq: 0 })
  return {
    state: reactive({ accounts: [] as Account[], currentAccountId: '' }),
    toast: vi.fn(),
    selectAccount: vi.fn(),
    accountFormRequest,
    requestAccountForm: vi.fn(() => {
      accountFormRequest.seq++
    }),
  }
})

vi.mock('../confirm', () => ({
  confirmDialog: vi.fn(async () => true),
}))

vi.mock('../i18n', () => ({
  t: (k: string) => k,
  tf: (k: string) => k,
  locale: () => 'en-US',
}))

const acc1: Account = {
  id: 'acc-1',
  name: 'MyAcc',
  endpoint: 'oss-cn-hangzhou.aliyuncs.com',
  publicEndpoint: 'oss-cn-hangzhou.aliyuncs.com',
  region: 'oss-cn-hangzhou',
  accessKey: 'ak1',
  secretSet: true,
  bucket: 'b1',
  pathStyle: false,
  useSSL: true,
}

const acc2: Account = {
  id: 'acc-2',
  name: 'Other',
  endpoint: 's3.us-east-1.amazonaws.com',
  region: 'us-east-1',
  accessKey: 'ak2',
  secretSet: true,
  bucket: '',
  pathStyle: false,
  useSSL: true,
}

// ModalDialog 渲染经 Teleport；面板测试用直通 stub 将表单内容留在 wrapper 内
const modalStub = {
  name: 'ModalDialog',
  props: ['open', 'title', 'width'],
  template: '<div v-if="open" class="modal-stub"><slot /></div>',
}

function mountPanel() {
  return mount(AccountsPanel, { global: { stubs: { ModalDialog: modalStub } } })
}

function findButton(w: ReturnType<typeof mount>, text: string) {
  const btn = w.findAll('button').find((b) => b.text() === text)
  expect(btn, `button "${text}" should exist`).toBeTruthy()
  return btn!
}

function fieldInput(w: ReturnType<typeof mount>, labelText: string) {
  const label = w.findAll('label.field').find((l) => l.text().includes(labelText))
  expect(label, `label containing "${labelText}" should exist`).toBeTruthy()
  const input = label!.find('input')
  expect(input.exists(), `input within "${labelText}"`).toBe(true)
  return input
}

beforeEach(() => {
  vi.clearAllMocks()
  state.accounts = []
  state.currentAccountId = 'acc-1'
})

describe('AccountsPanel', () => {
  it('加载账号并渲染表格：选中态、默认桶、未测试健康状态', async () => {
    vi.mocked(s3api.listAccounts).mockResolvedValue({ accounts: [acc1, acc2] })
    const w = mountPanel()
    await flushPromises()
    expect(w.text()).toContain('MyAcc')
    expect(w.text()).toContain('Other')
    expect(w.text()).toContain('oss-cn-hangzhou.aliyuncs.com')
    // 空桶显示 default 占位
    expect(w.text()).toContain('default')
    const radios = w.findAll('input[name="acc"]')
    expect((radios[0].element as HTMLInputElement).checked).toBe(true)
    expect((radios[1].element as HTMLInputElement).checked).toBe(false)
    expect(w.findAll('tbody tr')[0].classes()).toContain('selected')
    expect(w.text()).toContain('accounts.healthUntested')
    expect(w.find('.empty').exists()).toBe(false)
  })

  it('加载账号失败：渲染错误与「重试」，不再抛未捕获 rejection（可重试恢复）', async () => {
    vi.mocked(s3api.listAccounts).mockRejectedValueOnce(new Error('backend down'))
    const w = mountPanel()
    await flushPromises()
    // 后端不可用时页面不得崩溃，且必须给出可操作的重试入口
    expect(w.find('.msg.err').text()).toContain('accounts.loadFailed')
    expect(findButton(w, 'common.retry')).toBeTruthy()

    vi.mocked(s3api.listAccounts).mockResolvedValue({ accounts: [acc1] })
    await findButton(w, 'common.retry').trigger('click')
    await flushPromises()
    expect(w.text()).toContain('MyAcc')
    expect(w.find('.msg.err').exists()).toBe(false)
  })

  it('空态提示 + 添加按钮打开新增表单（s3 默认值）', async () => {
    vi.mocked(s3api.listAccounts).mockResolvedValue({ accounts: [] })
    const w = mountPanel()
    await flushPromises()
    expect(w.find('.empty').text()).toContain('accounts.empty')
    await findButton(w, 'accounts.add').trigger('click')
    expect(w.find('.modal-stub').exists()).toBe(true)
    // startCreate → applyProvider('s3')
    expect((w.find('input[placeholder="accounts.endpointPh"]').element as HTMLInputElement).value).toBe('127.0.0.1:9000')
    expect((w.find('input[list="region-list"]').element as HTMLInputElement).value).toBe('us-east-1')
    const checkboxes = w.findAll('input[type="checkbox"]')
    expect((checkboxes[0].element as HTMLInputElement).checked).toBe(true) // pathStyle
    expect((checkboxes[1].element as HTMLInputElement).checked).toBe(false) // useSSL
    // 新建模式：保存标题 + fetchHint 提示 + 取消关闭
    expect(w.text()).toContain('accounts.saveLogin')
    expect(w.text()).toContain('accounts.fetchHint')
    await findButton(w, 'common.cancel').trigger('click')
    expect(w.find('.modal-stub').exists()).toBe(false)
  })

  it('切换服务商应用预设；区域变更自动填 Endpoint（公共云同步公网地址）', async () => {
    vi.mocked(s3api.listAccounts).mockResolvedValue({ accounts: [] })
    const w = mountPanel()
    await flushPromises()
    await findButton(w, 'accounts.add').trigger('click')

    // 切到阿里云 OSS：pathStyle=false / useSSL=true / 公网端点同步
    await findButton(w, 'provider.oss.label').trigger('click')
    expect((w.find('input[placeholder="accounts.endpointPh"]').element as HTMLInputElement).value).toBe('oss-cn-hangzhou.aliyuncs.com')
    expect((w.find('input[placeholder="accounts.publicEndpointPh"]').element as HTMLInputElement).value).toBe('oss-cn-hangzhou.aliyuncs.com')
    const ossCheckboxes = w.findAll('input[type="checkbox"]')
    expect((ossCheckboxes[0].element as HTMLInputElement).checked).toBe(false)
    expect((ossCheckboxes[1].element as HTMLInputElement).checked).toBe(true)
    // 预设区域 → regionHint
    expect(w.text()).toContain('accounts.regionHint')
    const region = w.find('input[list="region-list"]')
    await region.setValue('oss-cn-beijing')
    await region.trigger('change')
    expect((w.find('input[placeholder="accounts.endpointPh"]').element as HTMLInputElement).value).toBe('oss-cn-beijing.aliyuncs.com')
    expect((w.find('input[placeholder="accounts.publicEndpointPh"]').element as HTMLInputElement).value).toBe('oss-cn-beijing.aliyuncs.com')

    // 未知区域：preset 未命中 → endpoint 不变
    await region.setValue('weird-region')
    await region.trigger('change')
    expect((w.find('input[placeholder="accounts.endpointPh"]').element as HTMLInputElement).value).toBe('oss-cn-beijing.aliyuncs.com')

    // AWS：端点留空由 SDK 推导，且不同步公网端点（预设 endpoint 亦为空）
    await findButton(w, 'provider.aws.label').trigger('click')
    expect((w.find('input[placeholder="accounts.endpointPh"]').element as HTMLInputElement).value).toBe('')
    expect((w.find('input[placeholder="accounts.publicEndpointPh"]').element as HTMLInputElement).value).toBe('')
    await region.setValue('us-west-2')
    await region.trigger('change')
    expect((w.find('input[placeholder="accounts.endpointPh"]').element as HTMLInputElement).value).toBe('')
    expect((w.find('input[placeholder="accounts.publicEndpointPh"]').element as HTMLInputElement).value).toBe('')
  })

  it('新建流程：拉取桶自动填充 + 提交创建', async () => {
    vi.mocked(s3api.listAccounts).mockResolvedValue({ accounts: [] })
    vi.mocked(s3api.previewBuckets).mockResolvedValueOnce({ buckets: [{ name: 'b9', creationDate: 'x' }] })
    let created!: AccountInput
    // submit 成功后 resetForm() 会原地清空 reactive form，故在调用时浅拷贝捕获
    vi.mocked(s3api.createAccount).mockImplementationOnce(async (f: AccountInput) => {
      created = { ...f }
      return { id: 'new-1' } as Awaited<ReturnType<typeof s3api.createAccount>>
    })
    const w = mountPanel()
    await flushPromises()
    await findButton(w, 'accounts.add').trigger('click')

    await fieldInput(w, 'accounts.name').setValue('NewAcc')
    await fieldInput(w, 'AccessKey ID').setValue('ak')
    await fieldInput(w, 'AccessKey Secret').setValue('sk')
    await findButton(w, 'accounts.fetchBuckets').trigger('click')
    await flushPromises()
    expect(s3api.previewBuckets).toHaveBeenCalledTimes(1)
    expect(s3api.previewBuckets).toHaveBeenCalledWith(expect.objectContaining({ name: 'NewAcc', accessKey: 'ak', secretKey: 'sk' }))
    // 唯一桶自动选中
    expect((w.find('input[placeholder="accounts.bucketManualPh"]').element as HTMLInputElement).value).toBe('b9')
    expect(toast).toHaveBeenCalledWith('accounts.toastFetched')

    await findButton(w, 'accounts.saveLogin').trigger('click')
    await flushPromises()
    expect(s3api.createAccount).toHaveBeenCalledTimes(1)
    expect(created).toMatchObject({ name: 'NewAcc', bucket: 'b9', accessKey: 'ak', secretKey: 'sk', pathStyle: true })
    expect(selectAccount).toHaveBeenCalledWith('new-1')
    expect(toast).toHaveBeenCalledWith('accounts.toastCreated')
    expect(w.emitted('changed')).toBeTruthy()
    expect(w.find('.modal-stub').exists()).toBe(false)
  })

  it('编辑流程：预填表单 + listBuckets + 提交更新', async () => {
    vi.mocked(s3api.listAccounts).mockResolvedValue({ accounts: [acc1] })
    vi.mocked(s3api.listBuckets).mockResolvedValueOnce({ buckets: [{ name: 'b1', creationDate: 'x' }, { name: 'b2', creationDate: 'y' }] })
    let updated!: Partial<AccountInput>
    vi.mocked(s3api.updateAccount).mockImplementationOnce(async (_id: string, f: Partial<AccountInput>) => {
      updated = { ...f }
      return { id: 'acc-1' } as Awaited<ReturnType<typeof s3api.updateAccount>>
    })
    const w = mountPanel()
    await flushPromises()
    await findButton(w, 'common.edit').trigger('click')
    await flushPromises()

    expect((w.find('input[placeholder="accounts.namePh"]').element as HTMLInputElement).value).toBe('MyAcc')
    expect((w.find('input[placeholder="accounts.endpointPh"]').element as HTMLInputElement).value).toBe('oss-cn-hangzhou.aliyuncs.com')
    expect((w.find('input[list="region-list"]').element as HTMLInputElement).value).toBe('oss-cn-hangzhou')
    expect((fieldInput(w, 'AccessKey ID').element as HTMLInputElement).value).toBe('ak1')
    const secret = fieldInput(w, 'AccessKey Secret')
    expect((secret.element as HTMLInputElement).value).toBe('')
    expect(secret.attributes('placeholder')).toBe('accounts.secretKeepPh')
    // 编辑态：保存按钮、桶选项，不再显示 fetchHint
    expect(w.text()).toContain('common.save')
    expect(w.find('select').findAll('option').map((o) => o.text())).toEqual(['accounts.bucketSelect', 'b1', 'b2'])
    expect(s3api.listBuckets).toHaveBeenCalledWith('acc-1')

    await fieldInput(w, 'accounts.name').setValue('MyAcc2')
    await findButton(w, 'common.save').trigger('click')
    await flushPromises()
    expect(s3api.updateAccount).toHaveBeenCalledTimes(1)
    expect(updated).toMatchObject({ name: 'MyAcc2', secretKey: '', bucket: 'b1' })
    expect(selectAccount).toHaveBeenCalledWith('acc-1')
    expect(toast).toHaveBeenCalledWith('accounts.toastUpdated')
    expect(w.emitted('changed')).toBeTruthy()
  })

  it('拉取桶失败显示错误；缺少凭据提示 needCreds', async () => {
    vi.mocked(s3api.listAccounts).mockResolvedValue({ accounts: [] })
    vi.mocked(s3api.previewBuckets).mockRejectedValueOnce(new Error('preview boom'))
    const w = mountPanel()
    await flushPromises()
    await findButton(w, 'accounts.add').trigger('click')
    // 缺少 accessKey/secret → 本地校验错误，不走 API
    await findButton(w, 'accounts.fetchBuckets').trigger('click')
    await flushPromises()
    expect(s3api.previewBuckets).not.toHaveBeenCalled()
    expect(w.text()).toContain('accounts.needCreds')

    // 补全凭据 → API 拒绝 → bucketErr 展示
    await fieldInput(w, 'AccessKey ID').setValue('ak')
    await fieldInput(w, 'AccessKey Secret').setValue('sk')
    await findButton(w, 'accounts.fetchBuckets').trigger('click')
    await flushPromises()
    expect(s3api.previewBuckets).toHaveBeenCalledTimes(1)
    expect(w.findAll('span.badge').some((s) => s.text() === 'preview boom')).toBe(true)
    // loading 结束按钮恢复可用
    expect(findButton(w, 'accounts.fetchBuckets').attributes('disabled')).toBeUndefined()
  })

  it('拉取桶返回空 buckets 字段时兜底为空列表', async () => {
    vi.mocked(s3api.listAccounts).mockResolvedValue({ accounts: [] })
    vi.mocked(s3api.previewBuckets).mockResolvedValueOnce({} as Awaited<ReturnType<typeof s3api.previewBuckets>>)
    const w = mountPanel()
    await flushPromises()
    await findButton(w, 'accounts.add').trigger('click')
    await fieldInput(w, 'AccessKey ID').setValue('ak')
    await fieldInput(w, 'AccessKey Secret').setValue('sk')
    await findButton(w, 'accounts.fetchBuckets').trigger('click')
    await flushPromises()
    // res.buckets ?? [] → 无选项，仅占位
    expect(w.find('select').findAll('option').map((o) => o.text())).toEqual(['accounts.bucketSelect'])
    expect(toast).toHaveBeenCalledWith('accounts.toastFetched')
  })

  it('健康检查：checking / ok / fail(带错误) / 异常', async () => {
    vi.mocked(s3api.listAccounts).mockResolvedValue({ accounts: [acc1] })
    let resolveTest!: (v: Awaited<ReturnType<typeof s3api.testAccount>>) => void
    vi.mocked(s3api.testAccount)
      .mockImplementationOnce(() => new Promise((r) => { resolveTest = r }))
      .mockResolvedValueOnce({ ok: false, bucket: '', error: 'conn refused' })
      .mockRejectedValueOnce(new Error('net down'))
      .mockResolvedValueOnce({ ok: true, bucket: 'b1' })
    const w = mountPanel()
    await flushPromises()

    // 测试中（deferred 未决）
    await findButton(w, 'accounts.test').trigger('click')
    expect(w.text()).toContain('accounts.healthChecking')
    resolveTest({ ok: true, bucket: 'b1' })
    await flushPromises()
    expect(w.text()).toContain('accounts.healthOk')

    // ok=false + err → fail badge 带 title
    await findButton(w, 'accounts.test').trigger('click')
    await flushPromises()
    const bad = w.find('span.tag.bad')
    expect(bad.text()).toBe('accounts.healthFail')
    expect(bad.attributes('title')).toBe('conn refused')

    // 异常 → 展示 toErrorMessage
    await findButton(w, 'accounts.test').trigger('click')
    await flushPromises()
    expect(w.find('span.tag.bad').attributes('title')).toBe('net down')

    // 再次成功
    await findButton(w, 'accounts.test').trigger('click')
    await flushPromises()
    expect(w.text()).toContain('accounts.healthOk')
  })

  it('删除：确认取消不删除；确认后删除并清空当前账号', async () => {
    vi.mocked(s3api.listAccounts).mockResolvedValue({ accounts: [acc1] })
    vi.mocked(confirmDialog).mockResolvedValueOnce(false)
    const w = mountPanel()
    await flushPromises()
    await findButton(w, 'common.delete').trigger('click')
    await flushPromises()
    expect(s3api.deleteAccount).not.toHaveBeenCalled()
    expect(w.text()).toContain('MyAcc')

    vi.mocked(confirmDialog).mockResolvedValueOnce(true)
    await findButton(w, 'common.delete').trigger('click')
    await flushPromises()
    expect(s3api.deleteAccount).toHaveBeenCalledWith('acc-1')
    expect(selectAccount).toHaveBeenCalledWith('')
    expect(toast).toHaveBeenCalledWith(expect.stringContaining('accounts.toastDeleted'))
    expect(w.emitted('changed')).toBeTruthy()
  })

  it('删除当前账号之外的账号不切换选中；删除失败展示错误', async () => {
    vi.mocked(s3api.listAccounts).mockResolvedValue({ accounts: [acc1, acc2] })
    vi.mocked(s3api.deleteAccount)
      .mockResolvedValueOnce({ deleted: 'acc-2' })
      .mockRejectedValueOnce(new Error('rm fail'))
    const w = mountPanel()
    await flushPromises()
    // 删除第二行（acc-2，非当前账号）
    await w.findAll('tbody tr')[1].find('button.danger.sm').trigger('click')
    await flushPromises()
    expect(s3api.deleteAccount).toHaveBeenCalledWith('acc-2')
    // 删除的不是当前账号 → 不调用 selectAccount
    expect(selectAccount).not.toHaveBeenCalled()

    await findButton(w, 'common.delete').trigger('click')
    await flushPromises()
    expect(w.find('.msg.err').text()).toBe('rm fail')
  })

  it('单选账号触发 selectAccount', async () => {
    vi.mocked(s3api.listAccounts).mockResolvedValue({ accounts: [acc1, acc2] })
    const w = mountPanel()
    await flushPromises()
    await w.findAll('input[name="acc"]')[1].setValue()
    expect(selectAccount).toHaveBeenCalledWith('acc-2')
  })

  it('requestAccountForm 自动打开新增表单', async () => {
    vi.mocked(s3api.listAccounts).mockResolvedValue({ accounts: [] })
    const w = mountPanel()
    await flushPromises()
    requestAccountForm()
    await flushPromises()
    expect(w.find('.modal-stub').exists()).toBe(true)
  })

  it('提交失败展示错误且表单保持打开', async () => {
    vi.mocked(s3api.listAccounts).mockResolvedValue({ accounts: [] })
    vi.mocked(s3api.createAccount).mockRejectedValueOnce(new Error('400 bad request'))
    const w = mountPanel()
    await flushPromises()
    await findButton(w, 'accounts.add').trigger('click')
    await findButton(w, 'accounts.saveLogin').trigger('click')
    await flushPromises()
    expect(w.find('.msg.err').text()).toBe('400 bad request')
    expect(w.find('.modal-stub').exists()).toBe(true)
    expect(selectAccount).not.toHaveBeenCalled()
  })

  it('编辑无 publicEndpoint 的账号：回退为空字符串', async () => {
    vi.mocked(s3api.listAccounts).mockResolvedValue({ accounts: [acc2] })
    vi.mocked(s3api.listBuckets).mockResolvedValue({ buckets: [] })
    const w = mountPanel()
    await flushPromises()
    await findButton(w, 'common.edit').trigger('click')
    await flushPromises()
    expect((w.find('input[placeholder="accounts.publicEndpointPh"]').element as HTMLInputElement).value).toBe('')
  })

  it('表单字段绑定：publicEndpoint / 桶选择 + 手动输入 / pathStyle / useSSL；ModalDialog close 关闭', async () => {
    vi.mocked(s3api.listAccounts).mockResolvedValue({ accounts: [] })
    vi.mocked(s3api.previewBuckets).mockResolvedValueOnce({ buckets: [{ name: 'b9', creationDate: 'x' }] })
    let created!: AccountInput
    vi.mocked(s3api.createAccount).mockImplementationOnce(async (f: AccountInput) => {
      created = { ...f }
      return { id: 'new-1' } as Awaited<ReturnType<typeof s3api.createAccount>>
    })
    const w = mountPanel()
    await flushPromises()
    await findButton(w, 'accounts.add').trigger('click')

    // endpoint / publicEndpoint 输入绑定
    const ep = fieldInput(w, 'accounts.endpoint')
    await ep.setValue('minio.local:9000')
    expect((ep.element as HTMLInputElement).value).toBe('minio.local:9000')
    const pub = fieldInput(w, 'accounts.publicEndpoint')
    await pub.setValue('https://pub.example.com')
    expect((pub.element as HTMLInputElement).value).toBe('https://pub.example.com')

    // 拉取桶后：select 选中桶
    await fieldInput(w, 'AccessKey ID').setValue('ak')
    await fieldInput(w, 'AccessKey Secret').setValue('sk')
    await findButton(w, 'accounts.fetchBuckets').trigger('click')
    await flushPromises()
    const bucketSelect = w.find('.bucket-row select')
    expect(bucketSelect.findAll('option').map((o) => o.text())).toEqual(['accounts.bucketSelect', 'b9'])
    await bucketSelect.setValue('b9')
    expect((bucketSelect.element as HTMLSelectElement).value).toBe('b9')

    // 手动输入桶名覆盖下拉选择
    const manual = w.find('input[placeholder="accounts.bucketManualPh"]')
    await manual.setValue('manual-bucket')
    expect((manual.element as HTMLInputElement).value).toBe('manual-bucket')

    // pathStyle / useSSL 复选框绑定（s3 默认 pathStyle=true / useSSL=false）
    const boxes = w.findAll('input[type="checkbox"]')
    expect((boxes[0].element as HTMLInputElement).checked).toBe(true)
    await boxes[0].setValue(false)
    expect((boxes[0].element as HTMLInputElement).checked).toBe(false)
    expect((boxes[1].element as HTMLInputElement).checked).toBe(false)
    await boxes[1].setValue(true)
    expect((boxes[1].element as HTMLInputElement).checked).toBe(true)

    // 提交：表单值完整到达 createAccount
    await findButton(w, 'accounts.saveLogin').trigger('click')
    await flushPromises()
    expect(created).toMatchObject({
      publicEndpoint: 'https://pub.example.com',
      bucket: 'manual-bucket',
      pathStyle: false,
      useSSL: true,
    })

    // ModalDialog close 事件关闭表单
    await findButton(w, 'accounts.add').trigger('click')
    expect(w.find('.modal-stub').exists()).toBe(true)
    w.findComponent(ModalDialog).vm.$emit('close')
    await nextTick()
    expect(w.find('.modal-stub').exists()).toBe(false)
  })
})
