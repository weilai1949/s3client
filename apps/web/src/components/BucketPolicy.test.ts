import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, shallowMount } from '@vue/test-utils'
import { nextTick } from 'vue'
import BucketPolicy from './BucketPolicy.vue'
import BucketPolicyVisualEditor from './BucketPolicyVisualEditor.vue'
import { s3api } from '../api'
import { toast } from '../store'
import { confirmDialog } from '../confirm'

vi.mock('../api', () => ({
  s3api: {
    getBucketPolicy: vi.fn(async () => ({ bucket: 'b1', configured: false, policy: '' })),
    putBucketPolicy: vi.fn(async () => ({ configured: true })),
    deleteBucketPolicy: vi.fn(async () => ({ deleted: 'b1' })),
  },
}))
vi.mock('../store', () => ({ toast: vi.fn() }))
vi.mock('../confirm', () => ({ confirmDialog: vi.fn(async () => true) }))
vi.mock('../i18n', () => ({
  t: (k: string) => k,
  tf: (k: string) => k,
  locale: () => 'en-US',
}))

const validJson = JSON.stringify({
  Version: '2012-10-17',
  Statement: [{ Sid: 'S1', Effect: 'Allow', Principal: '*', Action: 's3:GetObject', Resource: 'arn:aws:s3:::my-bucket/*' }],
})

function mountPolicy() {
  return shallowMount(BucketPolicy, { props: { accountId: 'acc-1', bucket: 'b1' } })
}

function saveBtn(w: ReturnType<typeof mountPolicy>) {
  const btn = w.findAll('button').find(
    (b) => b.text() === 'common.save' || b.text() === 'common.saving',
  )
  expect(btn, 'save button should exist').toBeTruthy()
  return btn!
}

function removeBtn(w: ReturnType<typeof mountPolicy>) {
  const btn = w.findAll('button').find((b) => b.text() === 'policy.removeBtn')
  expect(btn, 'remove button should exist').toBeTruthy()
  return btn!
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(s3api.getBucketPolicy).mockImplementation(async () => ({
    bucket: 'b1', configured: false, policy: '',
  }))
  vi.mocked(s3api.putBucketPolicy).mockImplementation(async () => ({ configured: true }))
  vi.mocked(s3api.deleteBucketPolicy).mockImplementation(async () => ({ deleted: 'b1' }))
  vi.mocked(confirmDialog).mockImplementation(async () => true)
})

describe('BucketPolicy', () => {
  it('loading → 渲染提示与可视化编辑器（raw 回显）', async () => {
    vi.mocked(s3api.getBucketPolicy).mockResolvedValue({
      bucket: 'b1',
      configured: true,
      policy: validJson,
    })
    const w = mountPolicy()
    expect(w.text()).toContain('policy.loading')
    await flushPromises()
    expect(w.text()).toContain('policy.hint')
    const editor = w.findComponent(BucketPolicyVisualEditor)
    expect(editor.exists()).toBe(true)
    expect(editor.props('raw')).toBe(validJson)
    expect(saveBtn(w).exists()).toBe(true)
    expect(removeBtn(w).exists()).toBe(true)
  })

  it('编辑器 error 事件向上转发', async () => {
    const w = mountPolicy()
    await flushPromises()
    w.findComponent(BucketPolicyVisualEditor).vm.$emit('error', 'editor-err')
    await nextTick()
    expect(w.emitted('error')).toEqual([['editor-err']])
  })

  it('编辑器 update → 保存：put → toast → changed → 重新加载', async () => {
    const w = mountPolicy()
    await flushPromises()
    const editor = w.findComponent(BucketPolicyVisualEditor)
    editor.vm.$emit('update', validJson)
    await nextTick()
    await saveBtn(w).trigger('click')
    await flushPromises()
    expect(vi.mocked(s3api.putBucketPolicy)).toHaveBeenCalledWith('acc-1', {
      bucket: 'b1',
      policy: validJson,
    })
    expect(toast).toHaveBeenCalledWith('policy.toastSaved')
    expect(w.emitted('changed')).toHaveLength(1)
    // save 成功后 reload → getBucketPolicy 再次被调用
    expect(vi.mocked(s3api.getBucketPolicy)).toHaveBeenCalledTimes(2)
  })

  it('空 draft 保存：报错且不调用 put', async () => {
    const w = mountPolicy()
    await flushPromises()
    await saveBtn(w).trigger('click')
    await flushPromises()
    expect(w.emitted('error')).toEqual([['policy.emptyErr']])
    expect(vi.mocked(s3api.putBucketPolicy)).not.toHaveBeenCalled()
  })

  it('非法 JSON 保存：报错且不调用 put', async () => {
    const w = mountPolicy()
    await flushPromises()
    w.findComponent(BucketPolicyVisualEditor).vm.$emit('update', 'not-json')
    await nextTick()
    await saveBtn(w).trigger('click')
    await flushPromises()
    expect(w.emitted('error')).toEqual([['policy.invalidJson']])
    expect(vi.mocked(s3api.putBucketPolicy)).not.toHaveBeenCalled()
  })

  it('remove 确认后：delete → toast → changed → policy 清空', async () => {
    const w = mountPolicy()
    await flushPromises()
    await removeBtn(w).trigger('click')
    expect(confirmDialog).toHaveBeenCalledWith(
      expect.objectContaining({ title: 'policy.removeTitle', danger: true }),
    )
    expect(vi.mocked(s3api.deleteBucketPolicy)).toHaveBeenCalledWith('acc-1', 'b1')
    await flushPromises()
    expect(toast).toHaveBeenCalledWith('policy.toastRemoved')
    expect(w.emitted('changed')).toHaveLength(1)
    // policy 被清空后编辑器 raw 也应同步为空
    expect(w.findComponent(BucketPolicyVisualEditor).props('raw')).toBe('')
  })

  it('remove 取消：不调用 delete', async () => {
    vi.mocked(confirmDialog).mockResolvedValue(false)
    const w = mountPolicy()
    await flushPromises()
    await removeBtn(w).trigger('click')
    expect(vi.mocked(s3api.deleteBucketPolicy)).not.toHaveBeenCalled()
    expect(toast).not.toHaveBeenCalled()
  })

  it('save 进行中：按钮禁用并显示 saving', async () => {
    let resolvePut!: (v: Awaited<ReturnType<typeof s3api.putBucketPolicy>>) => void
    vi.mocked(s3api.putBucketPolicy).mockImplementationOnce(
      () => new Promise((r) => { resolvePut = r }),
    )
    const w = mountPolicy()
    await flushPromises()
    w.findComponent(BucketPolicyVisualEditor).vm.$emit('update', validJson)
    await nextTick()
    await saveBtn(w).trigger('click')
    await nextTick()
    expect(saveBtn(w).text()).toBe('common.saving')
    expect(saveBtn(w).attributes('disabled')).toBeDefined()
    resolvePut({ configured: true })
    await flushPromises()
    expect(toast).toHaveBeenCalledWith('policy.toastSaved')
  })

  it('load 失败 → emit error', async () => {
    vi.mocked(s3api.getBucketPolicy).mockRejectedValueOnce(new Error('load fail'))
    const w = mountPolicy()
    await flushPromises()
    expect(w.emitted('error')).toEqual([['load fail']])
  })

  it('put 失败 → emit error 且不 toast', async () => {
    vi.mocked(s3api.getBucketPolicy).mockResolvedValue({
      bucket: 'b1', configured: false, policy: '',
    })
    vi.mocked(s3api.putBucketPolicy).mockRejectedValue(new Error('put fail'))
    const w = mountPolicy()
    await flushPromises()
    w.findComponent(BucketPolicyVisualEditor).vm.$emit('update', validJson)
    await nextTick()
    await saveBtn(w).trigger('click')
    await flushPromises()
    expect(w.emitted('error')).toEqual([['put fail']])
    expect(toast).not.toHaveBeenCalled()
  })
})
