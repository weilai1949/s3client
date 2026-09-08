import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import AclDialog from './AclDialog.vue'
import { s3api } from '../api'
import { copyText } from '../clipboard'
import { toast } from '../store'

vi.mock('../api', () => ({
  s3api: {
    getObjectAcl: vi.fn(),
    putObjectAcl: vi.fn(async () => ({ acl: 'private' })),
  },
}))

vi.mock('../store', () => ({ toasts: [], toast: vi.fn() }))

vi.mock('../clipboard', () => ({ copyText: vi.fn(async () => {}) }))

vi.mock('../i18n', () => ({
  t: (k: string) => k,
  tf: (k: string, vars?: Record<string, string | number>) => {
    let s = k
    for (const [kk, v] of Object.entries(vars ?? {})) s = s.replaceAll(`{${kk}}`, String(v))
    return s
  },
  locale: () => 'en-US',
}))

const ACL_RESULT = {
  bucket: 'b',
  key: 'k',
  public: true,
  owner: 'alice',
  grants: [
    { grantee: 'u1', permission: 'READ' },
    { grantee: 'u2', permission: 'WRITE' },
  ],
  url: 'https://ex.test/obj',
}

function mountDialog() {
  return mount(AclDialog, {
    props: { open: false, accountId: 'acc-1', bucket: 'b', objectKey: 'k' },
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

afterEach(() => {
  vi.clearAllMocks()
})

describe('AclDialog', () => {
  it('opens into loading, then renders ACL data (owner, grants, URL, preselect)', async () => {
    type AclResult = Awaited<ReturnType<typeof s3api.getObjectAcl>>
    let resolveLoad!: (v: AclResult) => void
    vi.mocked(s3api.getObjectAcl).mockImplementation(
      () => new Promise((resolve) => { resolveLoad = resolve }),
    )
    const w = mountDialog()
    await w.setProps({ open: true })
    await flushPromises()
    // 加载态：只有 loading 文案，无表单按钮
    expect(document.body.textContent).toContain('acl.loading')
    expect(document.body.textContent).not.toContain('acl.copyPublicUrl')

    resolveLoad(ACL_RESULT)
    await flushPromises()
    const text = document.body.textContent ?? ''
    expect(text).toContain('acl.title')
    expect(text).toContain('acl.owner')
    expect(text).toContain('acl.publicUrl')
    expect(text).not.toContain('acl.loading')
    // grants 表格直接渲染 grantee/permission
    const cells = Array.from(document.body.querySelectorAll('.tbl td')).map((td) => td.textContent)
    expect(cells).toEqual(['u1', 'READ', 'u2', 'WRITE'])
    // ACL 下拉当前选中 public（public: true）
    const sel = document.body.querySelector('select') as HTMLSelectElement
    expect(sel.selectedIndex).toBe(1)
    // 有 URL 时复制按钮可用
    expect(bodyBtn('acl.copyPublicUrl').disabled).toBe(false)
    w.unmount()
  })

  it('renders with no grants and no public URL', async () => {
    vi.mocked(s3api.getObjectAcl).mockResolvedValue({ ...ACL_RESULT, grants: [], url: '' })
    const w = mountDialog()
    await w.setProps({ open: true })
    await flushPromises()
    expect(document.body.querySelector('.tbl')).toBeNull()
    expect(document.body.textContent).not.toContain('acl.publicUrl')
    expect(document.body.textContent).toContain('acl.owner')
    w.unmount()
  })

  it('owner 缺省(null/undefined)→ `r.owner ?? \'\'` 兜底为空', async () => {
    const w = mountDialog()
    try {
      vi.mocked(s3api.getObjectAcl).mockResolvedValue({ ...ACL_RESULT, owner: undefined })
      await w.setProps({ open: true })
      await flushPromises()
      // aclOwner 为空 → 页面显示 owner 行(名称回退为占位符文案由 i18n 决定)
      const text = document.body.textContent ?? ''
      expect(text).toContain('acl.owner')
      expect(text).not.toContain('alice')
    } finally {
      w.unmount()
    }
  })

  it('copy guard: empty URL never reaches clipboard/toast', async () => {
    vi.mocked(s3api.getObjectAcl).mockResolvedValue({ ...ACL_RESULT, url: '' })
    const w = mountDialog()
    await w.setProps({ open: true })
    await flushPromises()
    expect(bodyBtn('acl.copyPublicUrl').disabled).toBe(true)
    await (w.vm as unknown as { copyPublicUrl: () => Promise<void> }).copyPublicUrl()
    expect(copyText).not.toHaveBeenCalled()
    expect(toast).not.toHaveBeenCalled()
    w.unmount()
  })

  it('copies public URL and shows toast', async () => {
    vi.mocked(s3api.getObjectAcl).mockResolvedValue(ACL_RESULT)
    const w = mountDialog()
    await w.setProps({ open: true })
    await flushPromises()
    clickBody('acl.copyPublicUrl')
    await flushPromises()
    expect(copyText).toHaveBeenCalledWith('https://ex.test/obj')
    expect(toast).toHaveBeenCalledWith('acl.toastCopiedLink')
    w.unmount()
  })

  it('saves public-read ACL (guards double submit), toasts and closes', async () => {
    vi.mocked(s3api.getObjectAcl).mockResolvedValue(ACL_RESULT)
    let resolvePut!: (v: { acl: string }) => void
    const pending = new Promise<{ acl: string }>((resolve) => { resolvePut = resolve })
    vi.mocked(s3api.putObjectAcl).mockImplementation(async () => pending)
    const w = mountDialog()
    await w.setProps({ open: true })
    await flushPromises()

    clickBody('common.save')
    await flushPromises()
    // saving 中按钮禁用；直接调方法验证 saving 防重入
    expect(bodyBtn('common.save').disabled).toBe(true)
    await (w.vm as unknown as { submitAcl: () => Promise<void> }).submitAcl()
    expect(vi.mocked(s3api.putObjectAcl)).toHaveBeenCalledTimes(1)
    expect(vi.mocked(s3api.putObjectAcl)).toHaveBeenCalledWith('acc-1', {
      bucket: 'b',
      key: 'k',
      acl: 'public-read',
    })

    resolvePut({ acl: 'public-read' })
    await flushPromises()
    expect(toast).toHaveBeenCalledWith('acl.toastPublic')
    expect(w.emitted('close')).toBeTruthy()
    expect(bodyBtn('common.save').disabled).toBe(false)
    w.unmount()
  })

  it('saves private ACL when not public', async () => {
    vi.mocked(s3api.getObjectAcl).mockResolvedValue({ ...ACL_RESULT, public: false })
    const w = mountDialog()
    await w.setProps({ open: true })
    await flushPromises()
    clickBody('common.save')
    await flushPromises()
    expect(vi.mocked(s3api.putObjectAcl)).toHaveBeenCalledWith('acc-1', {
      bucket: 'b',
      key: 'k',
      acl: 'private',
    })
    expect(toast).toHaveBeenCalledWith('acl.toastPrivate')
    expect(w.emitted('close')).toBeTruthy()
    w.unmount()
  })

  it('ACL 下拉 v-model 切换 private → 保存 private payload', async () => {
    vi.mocked(s3api.getObjectAcl).mockResolvedValue(ACL_RESULT) // public: true
    const w = mountDialog()
    await w.setProps({ open: true })
    await flushPromises()
    const sel = document.body.querySelector('select') as HTMLSelectElement
    sel.selectedIndex = 0 // private（:value=false）
    sel.dispatchEvent(new Event('change'))
    await flushPromises()
    clickBody('common.save')
    await flushPromises()
    expect(vi.mocked(s3api.putObjectAcl)).toHaveBeenCalledWith('acc-1', {
      bucket: 'b',
      key: 'k',
      acl: 'private',
    })
    expect(toast).toHaveBeenCalledWith('acl.toastPrivate')
    expect(w.emitted('close')).toBeTruthy()
    w.unmount()
  })

  it('emits error when saving fails', async () => {
    vi.mocked(s3api.getObjectAcl).mockResolvedValue(ACL_RESULT)
    vi.mocked(s3api.putObjectAcl).mockRejectedValue(new Error('acl-boom'))
    const w = mountDialog()
    await w.setProps({ open: true })
    await flushPromises()
    clickBody('common.save')
    await flushPromises()
    expect(w.emitted('error')).toEqual([['acl-boom']])
    expect(w.emitted('close')).toBeUndefined()
    w.unmount()
  })

  it('emits error when loading ACL fails', async () => {
    vi.mocked(s3api.getObjectAcl).mockRejectedValue(new Error('load-boom'))
    const w = mountDialog()
    await w.setProps({ open: true })
    await flushPromises()
    expect(w.emitted('error')).toEqual([['load-boom']])
    expect(document.body.textContent).not.toContain('acl.loading')
    w.unmount()
  })

  it('close button emits close', async () => {
    vi.mocked(s3api.getObjectAcl).mockResolvedValue(ACL_RESULT)
    const w = mountDialog()
    await w.setProps({ open: true })
    await flushPromises()
    clickBody('common.close')
    expect(w.emitted('close')).toBeTruthy()
    w.unmount()
  })

  it('open→false 走 watcher 的 early-return：不再发起任何加载', async () => {
    vi.mocked(s3api.getObjectAcl).mockResolvedValue(ACL_RESULT)
    const w = mountDialog()
    await w.setProps({ open: true })
    await flushPromises()
    expect(vi.mocked(s3api.getObjectAcl)).toHaveBeenCalledTimes(1)
    await w.setProps({ open: false })
    await flushPromises()
    expect(vi.mocked(s3api.getObjectAcl)).toHaveBeenCalledTimes(1)
    expect(w.emitted('error')).toBeUndefined()
    w.unmount()
  })

  it('ModalDialog 自身的 close 事件（遮罩/X）转发为父级 close', async () => {
    vi.mocked(s3api.getObjectAcl).mockResolvedValue(ACL_RESULT)
    const w = mountDialog()
    await w.setProps({ open: true })
    await flushPromises()
    w.findComponent({ name: 'ModalDialog' }).vm.$emit('close')
    expect(w.emitted('close')).toBeTruthy()
    w.unmount()
  })
})
