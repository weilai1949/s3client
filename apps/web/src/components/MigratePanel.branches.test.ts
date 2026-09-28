// MigratePanel.branches.test.ts —— 自 MigratePanel.test.ts 拆出（KNOWN_ISSUES #60）：
// 防御分支、虚拟列表滚动、v-model 接线、跨重启恢复。断言逐字搬移，与拆分前等价。
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import MigratePanel from './MigratePanel.vue'
import { s3api, subscribeMigrateEvents } from '../api'
import type { MigrateProgress } from '../api'
import { currentAccount, state, toast } from '../store'
import { tf } from '../i18n'
import type { Account, ObjectItem } from '../types'

/** MigratePanel 通过 defineExpose 暴露给测试的成员（组件 setup 状态无公开类型）。 */
interface MigrateVm {
  loadAllSourceObjects: () => Promise<void>
  loadSourceBuckets?: () => Promise<void> | void
  migrate: () => Promise<void>
  cancelMigrate?: () => Promise<void>
  cancelling: boolean
  ensureTargetAccount: () => void
  targetAccountId: string | undefined
  error: unknown
  selected?: Set<string>
  onListScroll: () => void
  measureViewport: () => void
  viewportH: number
}

vi.mock('../api', () => ({
  s3api: {
    listBuckets: vi.fn(),
    listObjects: vi.fn(),
    migrateAsync: vi.fn(),
    migrateJobs: vi.fn(),
    migrateJobStatus: vi.fn(),
    migrateJobCancel: vi.fn(),
  },
  subscribeMigrateEvents: vi.fn(() => () => {}),
}))

vi.mock('../store', async () => {
  const { reactive } = await import('vue')
  return {
    state: reactive({ accounts: [] as Account[], currentAccountId: '' }),
    currentAccount: vi.fn(),
    toast: vi.fn(),
    selectAccount: vi.fn(),
    requestTab: vi.fn(),
  }
})

vi.mock('../i18n', () => ({
  t: (k: string) => k,
  tf: vi.fn((k: string) => k),
}))

const acc1: Account = {
  id: 'acc-1', name: 'acc-one', endpoint: 'http://minio:9000', region: 'r',
  accessKey: 'ak', secretSet: true, bucket: 'src-bucket', pathStyle: true, useSSL: false,
}
const acc2: Account = { ...acc1, id: 'acc-2', name: 'acc-two', bucket: 'dst-bucket' }

const objA: ObjectItem = { key: 'a.txt', size: 10, lastModified: '2024-01-01', etag: 'e1', isDir: false }
const objB: ObjectItem = { key: 'b.bin', size: 20, lastModified: '2024-01-02', etag: 'e2', isDir: false }
const objDir: ObjectItem = { key: 'dir/', size: 0, lastModified: '', etag: '', isDir: true }

const ModalDialogStub = {
  name: 'ModalDialog',
  props: ['open', 'title', 'width'],
  template: '<div class="dlg-stub"><slot /></div>',
}

let mounted: ReturnType<typeof mount> | undefined
afterEach(() => {
  mounted?.unmount()
  mounted = undefined
})

function mountPanel() {
  mounted = mount(MigratePanel, { global: { stubs: { ModalDialog: ModalDialogStub } } })
  return mounted
}

function findButton(w: ReturnType<typeof mount>, text: string) {
  const btn = w.findAll('button').find((b) => b.text() === text)
  expect(btn, `button "${text}" should exist`).toBeTruthy()
  return btn!
}

beforeEach(() => {
  vi.clearAllMocks()
  state.accounts = [acc1, acc2]
  state.currentAccountId = 'acc-1'
  vi.mocked(currentAccount).mockImplementation(() =>
    state.accounts.find((a) => a.id === state.currentAccountId),
  )
  vi.mocked(s3api.listBuckets).mockResolvedValue({ buckets: [{ name: 'b-one', creationDate: '2024-01-01' }] })
  vi.mocked(s3api.migrateJobs).mockResolvedValue({ jobs: [] })
  vi.mocked(s3api.listObjects).mockResolvedValue({
    objects: [objA, objB, objDir], commonPrefixes: [], isTruncated: false, nextToken: '',
  })
})

describe('MigratePanel defensive guards (vm direct)', () => {
  it('migrate 无账号源时提前返回（!src 守卫）', async () => {
    const w = mountPanel()
    await flushPromises()
    const vm = w.vm as unknown as MigrateVm
    await expect(vm.migrate()).resolves.toBeUndefined()
  })
})

describe('MigratePanel selectTarget defensive guard', () => {
  it('migrate with cleared target → selectTarget 错误', async () => {
    const w = mountPanel()
    await flushPromises()
    await findButton(w, 'migrate.listObjects').trigger('click')
    await flushPromises()
    await w.findAll('.v-row input[type="checkbox"]')[0].setValue(true)
    await nextTick()
    const vm = w.vm as unknown as MigrateVm
    // ensureTargetAccount 会自动填充；显式清空以命中防御守卫
    vm.targetAccountId = ''
    await vm.migrate()
    expect(String(vm.error)).toMatch(/selectTarget|migrate.selectTarget/)
  })
})

describe('MigratePanel virtual list scroll', () => {
  it('scrolling the list updates scrollTop via onListScroll', async () => {
    const w = mountPanel()
    await flushPromises()
    await findButton(w, 'migrate.listObjects').trigger('click')
    await flushPromises()
    const tbl = w.find('.tbl-virtual')
    expect(tbl.exists()).toBe(true)
    // happy-dom 无法真实滚动；直接派发 scroll 事件并断言 viewport 计算不抛错
    await tbl.trigger('scroll')
    await nextTick()
    expect(w.find('.tbl-virtual').exists()).toBe(true)
  })
})

describe('MigratePanel bucket/prefix v-model wiring', () => {
  it('binds source/target bucket selects, target prefix and result dialog close', async () => {
    vi.mocked(s3api.migrateAsync).mockResolvedValue({ jobId: 'j9' } as Awaited<ReturnType<typeof s3api.migrateAsync>>)
    vi.mocked(s3api.migrateJobStatus).mockResolvedValue({
      progress: { status: 'done' }, result: { migrated: 1, failed: 0 },
    } as Awaited<ReturnType<typeof s3api.migrateJobStatus>>)
    let progressCb!: (p: MigrateProgress) => void
    vi.mocked(subscribeMigrateEvents).mockImplementation((_id, onP) => {
      progressCb = onP
      return () => {}
    })

    const w = mountPanel()
    await flushPromises()
    // 源 bucket select（selects[0]）：默认 + b-one，设置值后重新列出
    const srcSel = w.findAll('select')[0]
    await srcSel.setValue('b-one')
    await findButton(w, 'migrate.listObjects').trigger('click')
    await flushPromises()
    expect(s3api.listObjects).toHaveBeenLastCalledWith('acc-1', {
      bucket: 'b-one', prefix: '', delimiter: '/', maxKeys: '200',
    })
    // 目标 bucket select（selects[2]）与目标前缀输入
    await w.findAll('select')[2].setValue('b-one')
    await w.find('input[placeholder="migrate.targetPrefixPh"]').setValue('dst/')
    // 勾选并迁移 → 结果弹窗打开（title 传给 ModalDialog）
    await w.find('.toolbar input[type="checkbox"]').setValue(true)
    await findButton(w, 'migrate.start').trigger('click')
    await flushPromises()
    expect(s3api.migrateAsync).toHaveBeenCalledWith(expect.objectContaining({
      sourceBucket: 'b-one', targetBucket: 'b-one', targetPrefix: 'dst/',
    }))
    progressCb({ done: 1, total: 1, migrated: 1, failed: 0, status: 'done' })
    await flushPromises()
    const dlg = w.findComponent({ name: 'ModalDialog' })
    expect(dlg.props('open')).toBe(true)
    expect(dlg.props('title')).toBe('migrate.resultTitle')
    // ModalDialog @close 事件路径：关闭结果弹窗
    ;dlg.vm.$emit('close')
    await nextTick()
    expect(dlg.props('open')).toBe(false)
  })
})

describe('MigratePanel final branches', () => {
  it('loadSourceBuckets with no account returns early', async () => {
    state.accounts = []
    state.currentAccountId = ''
    const w = mountPanel()
    await flushPromises()
    expect(s3api.listBuckets).not.toHaveBeenCalled()
    await (w.vm as unknown as MigrateVm).loadSourceBuckets?.()
    expect(s3api.listBuckets).not.toHaveBeenCalled()
    state.accounts = [acc1]
    state.currentAccountId = 'acc-1'
  })

  it('toggle deselect removes key from selection (UI double check)', async () => {
    const w = mountPanel()
    await flushPromises()
    await findButton(w, 'migrate.listObjects').trigger('click')
    await flushPromises()
    const cb = w.findAll('.v-row input[type="checkbox"]')[0]
    await cb.setValue(true) // 勾选 → 选中
    await nextTick()
    expect((w.vm as unknown as MigrateVm).selected?.has?.('a.txt')).toBe(true)
    await cb.setValue(false) // 取消 → 反选
    await nextTick()
    expect((w.vm as unknown as MigrateVm).selected?.has?.('a.txt')).toBe(false)
  })

  it('cancelMigrate without job or while cancelling is a no-op', async () => {
    const w = mountPanel()
    await flushPromises()
    const vm = w.vm as unknown as MigrateVm
    await expect(vm.cancelMigrate?.()).resolves.toBeUndefined()
    vm.cancelling = true
    await expect(vm.cancelMigrate?.()).resolves.toBeUndefined()
  })

  it('ensureTargetAccount is idempotent', async () => {
    const w = mountPanel()
    await flushPromises()
    const vm = w.vm as unknown as MigrateVm
    vm.ensureTargetAccount()
    const before = vm.targetAccountId
    vm.ensureTargetAccount() // 已有 target → 不改变
    expect(vm.targetAccountId).toBe(before)
  })
})

describe('MigratePanel remaining branches', () => {
  it('桶列表响应缺 buckets 键时源/目标桶列表均回退为空', async () => {
    vi.mocked(s3api.listBuckets).mockResolvedValue({} as Awaited<ReturnType<typeof s3api.listBuckets>>)
    const w = mountPanel()
    await flushPromises()
    // 源桶下拉：仅默认选项（res.buckets ?? []）
    expect(w.findAll('select')[0].findAll('option')).toHaveLength(1)
    // 目标桶下拉：仅两个账号 option，无桶
    expect(w.findAll('select')[1].findAll('option')).toHaveLength(2)
  })

  it('ensureTargetAccount 回退 state.accounts[0].id（源账号不存在且无其他账号）', async () => {
    state.accounts = [{ ...acc1, id: undefined }] as unknown as Account[]
    state.currentAccountId = 'zz-missing'
    vi.mocked(currentAccount).mockReturnValue(undefined)
    const w = mountPanel()
    await flushPromises()
    // other?.id 与 srcId 均空 → state.accounts[0].id（undefined 值执行该分支）
    expect((w.vm as unknown as MigrateVm).targetAccountId).toBeUndefined()
  })

  it('migrateJobStatus 响应缺 result 键时回退 0/0 并 toastOk', async () => {
    let progressCb!: (p: MigrateProgress) => void
    vi.mocked(s3api.migrateAsync).mockResolvedValue({ jobId: 'j10' } as Awaited<ReturnType<typeof s3api.migrateAsync>>)
    vi.mocked(s3api.migrateJobStatus).mockResolvedValue({ progress: { status: 'done' } } as Awaited<ReturnType<typeof s3api.migrateJobStatus>>)
    vi.mocked(subscribeMigrateEvents).mockImplementation((_id, onP) => {
      progressCb = onP
      return () => {}
    })
    const w = mountPanel()
    await flushPromises()
    await w.find('.toolbar input[type="checkbox"]').setValue(true)
    await findButton(w, 'migrate.start').trigger('click')
    await flushPromises()
    progressCb({ done: 1, total: 1, migrated: 1, failed: 0, status: 'done' })
    await flushPromises()
    // st.result 缺省 → { migrated: 0, failed: 0 } → toastOk
    expect(toast).toHaveBeenCalledWith('migrate.toastOk')
  })

  it('无默认桶配置的账号源桶下拉回退 default', async () => {
    state.accounts = [{ ...acc1, bucket: '' }]
    mountPanel()
    await flushPromises()
    expect(vi.mocked(tf)).toHaveBeenCalledWith('migrate.defaultBucket', { name: 'default' })
  })

  it('结果弹窗：有失败但无 lastError 时不渲染 firstError', async () => {
    let progressCb!: (p: MigrateProgress) => void
    vi.mocked(s3api.migrateAsync).mockResolvedValue({ jobId: 'j11' } as Awaited<ReturnType<typeof s3api.migrateAsync>>)
    vi.mocked(s3api.migrateJobStatus).mockResolvedValue({
      progress: { status: 'done' },
      result: { migrated: 0, failed: 1, failedKeys: ['k-bad'], lastError: '' },
    } as Awaited<ReturnType<typeof s3api.migrateJobStatus>>)
    vi.mocked(subscribeMigrateEvents).mockImplementation((_id, onP) => {
      progressCb = onP
      return () => {}
    })
    const w = mountPanel()
    await flushPromises()
    await w.find('.toolbar input[type="checkbox"]').setValue(true)
    await findButton(w, 'migrate.start').trigger('click')
    await flushPromises()
    progressCb({ done: 1, total: 1, migrated: 0, failed: 1, status: 'done' })
    await flushPromises()
    expect(w.text()).toContain('k-bad')
    expect(w.text()).not.toContain('migrate.firstError')
  })

  it('进度 done 为 0 时进度条显示 0%', async () => {
    let progressCb!: (p: MigrateProgress) => void
    vi.mocked(s3api.migrateAsync).mockResolvedValue({ jobId: 'j12' } as Awaited<ReturnType<typeof s3api.migrateAsync>>)
    vi.mocked(s3api.migrateJobStatus).mockResolvedValue({
      progress: { status: 'done' },
      result: { migrated: 1, failed: 0 },
    } as Awaited<ReturnType<typeof s3api.migrateJobStatus>>)
    vi.mocked(subscribeMigrateEvents).mockImplementation((_id, onP) => {
      progressCb = onP
      return () => {}
    })
    const w = mountPanel()
    await flushPromises()
    await w.find('.toolbar input[type="checkbox"]').setValue(true)
    await findButton(w, 'migrate.start').trigger('click')
    await flushPromises()
    // 已完成的 key 为 0 → 0%
    progressCb({ done: 0, total: 4, migrated: 0, failed: 0, status: 'running' })
    await nextTick()
    expect(w.find('.progress').attributes('aria-valuenow')).toBe('0')
    expect(w.find('.progress .bar').attributes('style')).toContain('0%')
    progressCb({ done: 4, total: 4, migrated: 4, failed: 0, status: 'done' })
    await flushPromises()
  })

  it('scrollEl 未绑定时 onListScroll/measureViewport 空安全，绑定后按 clientHeight 测量', async () => {
    const w = mountPanel()
    const vm = w.vm as unknown as MigrateVm
    // 列表尚未渲染（scrollEl 为 null）→ 守卫直接跳过
    vm.onListScroll()
    vm.measureViewport()
    expect(vm.viewportH).toBe(480)
    await flushPromises()
    // 列表渲染绑定 scrollEl → 测量（happy-dom clientHeight=0 → 480 兜底）
    expect(w.find('.tbl-virtual').exists()).toBe(true)
    expect(vm.viewportH).toBe(480)
    const wrap = w.find('.tbl-virtual')
    Object.defineProperty(wrap.element, 'clientHeight', { value: 600, configurable: true })
    vm.measureViewport()
    expect(vm.viewportH).toBe(600)
  })
})

describe('MigratePanel 未完成任务（跨重启恢复）', () => {
  const interrupted = {
    id: 'job-int', created: '2026-09-16T10:00:00Z', total: 5, status: 'interrupted' as const,
    progress: { done: 2, total: 5, migrated: 2, failed: 1, status: 'interrupted' },
    result: { migrated: 2, failed: 1 },
  }
  const doneJob = {
    id: 'job-done', created: '2026-09-16T09:00:00Z', total: 1, status: 'done' as const,
    progress: { done: 1, total: 1, migrated: 1, failed: 0, status: 'done' },
    result: { migrated: 1, failed: 0 },
  }

  it('展示 interrupted 任务，过滤掉已完成任务', async () => {
    vi.mocked(s3api.migrateJobs).mockResolvedValue({ jobs: [interrupted, doneJob] })
    const w = mountPanel()
    await flushPromises()

    expect(s3api.migrateJobs).toHaveBeenCalled()
    const block = w.find('.unfinished')
    expect(block.exists()).toBe(true)
    // interrupted 必须可见：这是「移动任务复制成功但源未删除」的唯一提示
    expect(block.text()).toContain('job-int')
    expect(block.text()).toContain('migrate.statusInterrupted')
    expect(block.text()).toContain('migrate.unfinishedProgress')
    // done 任务不进入未完成视图
    expect(block.text()).not.toContain('job-done')
  })

  it('无未完成任务时不渲染区块', async () => {
    vi.mocked(s3api.migrateJobs).mockResolvedValue({ jobs: [doneJob] })
    const w = mountPanel()
    await flushPromises()
    expect(w.find('.unfinished').exists()).toBe(false)
  })

  it('running 任务也展示（仍在进行中）', async () => {
    vi.mocked(s3api.migrateJobs).mockResolvedValue({
      jobs: [{ ...interrupted, id: 'job-run', status: 'running' as const }],
    })
    const w = mountPanel()
    await flushPromises()
    const block = w.find('.unfinished')
    expect(block.text()).toContain('job-run')
    expect(block.text()).toContain('migrate.statusRunning')
  })

  it('加载失败时展示错误且不崩溃', async () => {
    vi.mocked(s3api.migrateJobs).mockRejectedValue(new Error('boom'))
    const w = mountPanel()
    await flushPromises()
    expect(w.find('.unfinished').exists()).toBe(true)
    expect(w.find('.unfinished').text()).toContain('boom')
  })

  it('dismiss 仅从本地列表移除', async () => {
    vi.mocked(s3api.migrateJobs).mockResolvedValue({ jobs: [interrupted] })
    const w = mountPanel()
    await flushPromises()
    expect(w.find('.unfinished').text()).toContain('job-int')

    const btn = w.findAll('.unfinished button').find((b) => b.text() === 'migrate.dismiss')
    expect(btn).toBeTruthy()
    await btn!.trigger('click')
    // 唯一一条被忽略后整块消失（v-if 依据 unfinished.length）
    expect(w.find('.unfinished').exists()).toBe(false)
  })

  it('刷新按钮重新拉取清单', async () => {
    vi.mocked(s3api.migrateJobs).mockResolvedValue({ jobs: [interrupted] })
    const w = mountPanel()
    await flushPromises()
    const calls = vi.mocked(s3api.migrateJobs).mock.calls.length

    const refresh = w.findAll('.unfinished button').find((b) => b.text() === 'common.refresh')
    expect(refresh).toBeTruthy()
    await refresh!.trigger('click')
    await flushPromises()
    expect(vi.mocked(s3api.migrateJobs).mock.calls.length).toBe(calls + 1)
  })

  it('迁移完成（终态）后不再出现在未完成列表', async () => {
    // 先返回 running，再返回 done：模拟任务完成后刷新
    vi.mocked(s3api.migrateJobs).mockResolvedValue({ jobs: [{ ...interrupted, status: 'running' as const }] })
    const w = mountPanel()
    await flushPromises()
    expect(w.find('.unfinished').text()).toContain('job-int')

    vi.mocked(s3api.migrateJobs).mockResolvedValue({ jobs: [doneJob] })
    await (w.vm as unknown as { loadUnfinishedJobs: () => Promise<void> }).loadUnfinishedJobs()
    await flushPromises()
    expect(w.find('.unfinished').exists()).toBe(false)
  })

  it('jobs 为 null 时按空清单处理（后端返回 null 而非 []）', async () => {
    vi.mocked(s3api.migrateJobs).mockResolvedValue({ jobs: null as unknown as [] })
    const w = mountPanel()
    await flushPromises()
    expect(w.find('.unfinished').exists()).toBe(false)
  })

  it('有失败数的中断任务展示失败计数', async () => {
    vi.mocked(s3api.migrateJobs).mockResolvedValue({ jobs: [interrupted] })
    const w = mountPanel()
    await flushPromises()
    // interrupted 含 failed=1 → 渲染 resultFail 提示，提醒用户存在半成功对象
    expect(w.find('.unfinished').text()).toContain('migrate.resultFail')
  })

  it('无失败数的中断任务不展示失败计数', async () => {
    vi.mocked(s3api.migrateJobs).mockResolvedValue({
      jobs: [{ ...interrupted, result: { migrated: 2, failed: 0 } }],
    })
    const w = mountPanel()
    await flushPromises()
    expect(w.find('.unfinished').text()).not.toContain('migrate.resultFail')
  })
})
