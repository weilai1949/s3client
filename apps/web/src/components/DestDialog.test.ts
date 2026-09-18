import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import DestDialog from './DestDialog.vue'
import { s3api, subscribeMigrateEvents, type MigrateProgress } from '../api'
import { createProgressToast, toast } from '../store'

vi.mock('../api', () => ({
  s3api: {
    listBuckets: vi.fn(),
    copyObject: vi.fn(async () => ({ copied: 'dst/x', bucket: 'b2' })),
    renameObject: vi.fn(async () => ({ renamed: 'b2/dst/x' })),
    copyPrefixAsync: vi.fn(async () => ({ jobId: 'job-copy', total: 2 })),
    deletePrefixAsync: vi.fn(async () => ({ jobId: 'job-del', truncated: false })),
    copyFilesAsync: vi.fn(async () => ({ jobId: 'job-files', total: 2 })),
  },
  subscribeMigrateEvents: vi.fn(),
}))

vi.mock('../store', () => ({
  toasts: [],
  toast: vi.fn(),
  createProgressToast: vi.fn(() => () => {}),
}))

vi.mock('../i18n', () => ({
  t: (k: string) => k,
  tf: (k: string) => k,
  locale: () => 'en-US',
}))

type Dialog = ReturnType<typeof mountDest>

const DEFAULT_EVENTS = (_jobId: string, onProgress: (p: MigrateProgress) => void) => {
  queueMicrotask(() => {
    onProgress({ status: 'done', migrated: 1, failed: 0, done: 1, total: 1 })
  })
  return () => {}
}

function mountDest(overrides: Record<string, unknown> = {}) {
  return track(mount(DestDialog, {
    props: {
      open: false,
      accountId: 'acc-1',
      sourceBucket: 'b1',
      kind: 'file',
      mode: 'copy',
      objectKey: 'k',
      ...overrides,
    },
    attachTo: document.body,
  }))
}

async function openDialog(w: Dialog, buckets: Array<{ name: string; creationDate: string }> = [
  { name: 'b1', creationDate: '2024-01-01' },
  { name: 'b2', creationDate: '2024-01-02' },
]) {
  vi.mocked(s3api.listBuckets).mockResolvedValue({ buckets })
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

function pathInput(): HTMLInputElement {
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

beforeEach(() => {
  vi.mocked(subscribeMigrateEvents).mockImplementation(DEFAULT_EVENTS as never)
})

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

describe('DestDialog', () => {
  it('open loads buckets and prefills source/path', async () => {
    const w = mountDest({ kind: 'file' })
    await openDialog(w)
    const text = document.body.textContent ?? ''
    expect(text).toContain('dest.source')
    expect(text).toContain('dest.targetBucket')
    expect(text).toContain('dest.targetPath')
    expect(text).toContain('common.copy')
    // 默认目标桶 = 源桶，路径 = 对象键
    expect(selectEl().selectedIndex).toBe(0) // b1
    expect(pathInput().value).toBe('k')
  })

  it('bucket 列表加载失败时 emit error', async () => {
    vi.mocked(s3api.listBuckets).mockRejectedValue(new Error('buckets down'))
    const w = mountDest()
    await w.setProps({ open: true })
    await flushPromises()
    expect(w.emitted('error')).toEqual([['buckets down']])
  })

  it('cancel button emits close', async () => {
    const w = mountDest()
    await openDialog(w)
    clickBody('common.cancel')
    expect(w.emitted('close')).toBeTruthy()
  })

  it('submit without bucket emits errBucket and does not copy', async () => {
    const w = mountDest()
    await openDialog(w, []) // 无桶 → 仅空选项
    const select = selectEl()
    // 列表为空 select 只有空选项（v-if !buckets.length）
    expect(select.options).toHaveLength(1)
    expect(select.options[0].value).toBe('')
    expect(document.body.textContent).toContain('dest.noBucket')
    // 用户手动清空目标桶（v-model 写回 ''）
    selectIndex(select, 0)
    await flushPromises()
    clickBody('common.copy')
    await flushPromises()
    expect(w.emitted('error')).toEqual([['dest.errBucket']])
    expect(vi.mocked(s3api.copyObject)).not.toHaveBeenCalled()
  })

  it('single file without path emits errPath', async () => {
    const w = mountDest({ kind: 'file' })
    await openDialog(w)
    pathInput().value = ''
    pathInput().dispatchEvent(new Event('input'))
    await flushPromises()
    clickBody('common.copy')
    await flushPromises()
    expect(w.emitted('error')).toEqual([['dest.errPath']])
    expect(vi.mocked(s3api.copyObject)).not.toHaveBeenCalled()
  })

  it('copies a file: copyObject + toast + submit + busy reset', async () => {
    const w = mountDest({ kind: 'file' })
    await openDialog(w)
    selectIndex(selectEl(), 1) // b2
    pathInput().value = 'dst/x'
    pathInput().dispatchEvent(new Event('input'))
    await flushPromises()
    clickBody('common.copy')
    await flushPromises()
    expect(vi.mocked(s3api.copyObject)).toHaveBeenCalledWith('acc-1', {
      bucket: 'b1',
      key: 'k',
      newBucket: 'b2',
      newKey: 'dst/x',
    })
    expect(toast).toHaveBeenCalledWith('dest.toastCopiedTo')
    expect(w.emitted('submit')).toBeTruthy()
    expect(bodyBtn('common.copy').disabled).toBe(false)
  })

  it('moves a file via renameObject', async () => {
    const w = mountDest({ kind: 'file', mode: 'move' })
    await openDialog(w)
    selectIndex(selectEl(), 1)
    pathInput().value = 'dst/x'
    pathInput().dispatchEvent(new Event('input'))
    await flushPromises()
    clickBody('common.move')
    await flushPromises()
    expect(vi.mocked(s3api.renameObject)).toHaveBeenCalledWith('acc-1', {
      bucket: 'b1',
      key: 'k',
      newBucket: 'b2',
      newKey: 'dst/x',
    })
    expect(toast).toHaveBeenCalledWith('dest.toastMovedTo')
    expect(w.emitted('submit')).toBeTruthy()
  })

  it('busy guard blocks a second submit while a copy is pending', async () => {
    let resolveCopy!: (v: { copied: string; bucket: string }) => void
    const pending = new Promise<{ copied: string; bucket: string }>((resolve) => { resolveCopy = resolve })
    vi.mocked(s3api.copyObject).mockImplementation(async () => pending)
    const w = mountDest({ kind: 'file' })
    await openDialog(w)
    pathInput().value = 'dst/x'
    pathInput().dispatchEvent(new Event('input'))
    await flushPromises()
    clickBody('common.copy')
    await flushPromises()
    // busy 期间再调用 submit：直接返回
    await (w.vm as unknown as { submitDest: () => Promise<void> }).submitDest()
    expect(vi.mocked(s3api.copyObject)).toHaveBeenCalledTimes(1)
    resolveCopy({ copied: 'dst/x', bucket: 'b1' })
    await flushPromises()
    expect(w.emitted('submit')).toBeTruthy()
  })

  it('emits error when file copy fails', async () => {
    vi.mocked(s3api.copyObject).mockRejectedValue(new Error('copy-boom'))
    const w = mountDest({ kind: 'file' })
    await openDialog(w)
    pathInput().value = 'dst/x'
    pathInput().dispatchEvent(new Event('input'))
    await flushPromises()
    clickBody('common.copy')
    await flushPromises()
    expect(w.emitted('error')).toEqual([['copy-boom']])
    expect(bodyBtn('common.copy').disabled).toBe(false)
  })

  it('folder copy: async prefix copy then done toast', async () => {
    const w = mountDest({ kind: 'folder' })
    await openDialog(w)
    clickBody('common.copy')
    await flushPromises()
    expect(document.body.textContent).toContain('dest.hintFolder')
    expect(vi.mocked(s3api.copyPrefixAsync)).toHaveBeenCalledWith('acc-1', {
      bucket: 'b1',
      prefix: 'k',
      targetBucket: 'b1',
      targetPrefix: 'k/',
    })
    expect(toast).toHaveBeenNthCalledWith(1, 'dest.toastCopyingFolder')
    expect(toast).toHaveBeenLastCalledWith('dest.toastCopiedN')
    expect(w.emitted('submit')).toBeTruthy()
  })

  it('unmount aborts the in-flight SSE subscription', async () => {
    const stop = vi.fn()
    let held: ((p: MigrateProgress) => void) | undefined
    vi.mocked(subscribeMigrateEvents).mockImplementation(((_jobId: string, onProgress: (p: MigrateProgress) => void) => {
      held = onProgress
      return stop
    }) as never)
    // 单独挂载（不 track）：本用例自行卸载，避免 afterEach 重复卸载。
    const w = mount(DestDialog, {
      props: { open: false, accountId: 'acc-1', sourceBucket: 'b1', kind: 'folder', mode: 'copy', objectKey: 'k' },
      attachTo: document.body,
    })
    await openDialog(w)
    clickBody('common.copy')
    await flushPromises()
    expect(held).toBeTruthy()
    expect(stop).not.toHaveBeenCalled()

    w.unmount()
    expect(stop).toHaveBeenCalledTimes(1)
  })

  it('folder copy partial failure toast (cancelled job)', async () => {
    vi.mocked(subscribeMigrateEvents).mockImplementation(((_jobId: string, onProgress: (p: MigrateProgress) => void) => {
      queueMicrotask(() => onProgress({ status: 'cancelled', migrated: 0, failed: 3, done: 0, total: 4 }))
      return () => {}
    }) as never)
    const w = mountDest({ kind: 'folder' })
    await openDialog(w)
    clickBody('common.copy')
    await flushPromises()
    expect(toast).toHaveBeenLastCalledWith('dest.toastCopiedPartial')
  })

  it('folder move partial keeps sources and deletes nothing', async () => {
    vi.mocked(subscribeMigrateEvents).mockImplementation(((_jobId: string, onProgress: (p: MigrateProgress) => void) => {
      queueMicrotask(() => onProgress({ status: 'done', migrated: 0, failed: 2, done: 0, total: 2 }))
      return () => {}
    }) as never)
    const w = mountDest({ kind: 'folder', mode: 'move' })
    await openDialog(w)
    clickBody('common.move')
    await flushPromises()
    expect(toast).toHaveBeenLastCalledWith('dest.toastMovePartialKeep', 'err')
    expect(vi.mocked(s3api.deletePrefixAsync)).not.toHaveBeenCalled()
    expect(w.emitted('submit')).toBeUndefined()
  })

  it('folder move success: copy, delete source, threaded progress toast', async () => {
    vi.mocked(subscribeMigrateEvents).mockImplementation(((_jobId: string, onProgress: (p: MigrateProgress) => void) => {
      queueMicrotask(() => {
        if (_jobId === 'job-copy') {
          onProgress({ status: 'done', migrated: 2, failed: 0, done: 2, total: 2 })
        } else {
          onProgress({ status: 'running', migrated: 0, failed: 0, done: 3, total: 10 })
          onProgress({ status: 'done', migrated: 5, failed: 0, done: 5, total: 10 })
        }
      })
      return () => {}
    }) as never)
    let prog!: (s: string) => void
    vi.mocked(createProgressToast).mockImplementation(() => {
      prog = vi.fn()
      return prog
    })
    const w = mountDest({ kind: 'folder', mode: 'move' })
    await openDialog(w)
    clickBody('common.move')
    await flushPromises()
    expect(vi.mocked(s3api.deletePrefixAsync)).toHaveBeenCalledWith('acc-1', {
      bucket: 'b1',
      prefix: 'k',
    })
    expect(createProgressToast).toHaveBeenCalledTimes(1)
    expect(prog).toHaveBeenCalledWith('dest.toastDeleteProgress')
    expect(toast).toHaveBeenLastCalledWith('dest.toastMovedN')
    expect(w.emitted('submit')).toBeTruthy()
  })

  it('folder move success with truncated delete reports truncated toast', async () => {
    vi.mocked(s3api.deletePrefixAsync).mockResolvedValue({ jobId: 'job-del', total: 0, truncated: true })
    const w = mountDest({ kind: 'folder', mode: 'move' })
    await openDialog(w)
    clickBody('common.move')
    await flushPromises()
    expect(toast).toHaveBeenLastCalledWith('dest.toastDeletedTruncated')
  })

  it('folder move emits error when SSE fails', async () => {
    vi.mocked(subscribeMigrateEvents).mockImplementation(((_jobId: string, _onProgress: (p: MigrateProgress) => void, onError?: (e: Error) => void) => {
      queueMicrotask(() => onError?.(new Error('sse-broken')))
      return () => {}
    }) as never)
    const w = mountDest({ kind: 'folder' })
    await openDialog(w)
    clickBody('common.copy')
    await flushPromises()
    expect(w.emitted('error')).toEqual([['sse-broken']])
  })

  it('multi copy: empty prefix allowed, keys passed, working + done toasts', async () => {
    const w = mountDest({ kind: 'multi', keys: ['a', 'b'] })
    await openDialog(w)
    expect(document.body.textContent).toContain('dest.selected')
    expect(document.body.textContent).toContain('dest.targetPathMulti')
    clickBody('common.copy')
    await flushPromises()
    expect(vi.mocked(s3api.copyFilesAsync)).toHaveBeenCalledWith('acc-1', {
      bucket: 'b1',
      targetBucket: 'b1',
      targetPrefix: '',
      keys: ['a', 'b'],
      deleteSource: false,
    })
    expect(vi.mocked(toast).mock.calls.map((c) => c[0])).toEqual(['dest.toastWorking', 'dest.toastDone'])
    expect(w.emitted('submit')).toBeTruthy()
  })

  it('multi move with explicit prefix deletes sources', async () => {
    const w = mountDest({ kind: 'multi', mode: 'move', keys: ['a', 'b'] })
    await openDialog(w)
    pathInput().value = 'p/'
    pathInput().dispatchEvent(new Event('input'))
    await flushPromises()
    clickBody('common.move')
    await flushPromises()
    expect(vi.mocked(s3api.copyFilesAsync)).toHaveBeenCalledWith('acc-1', {
      bucket: 'b1',
      targetBucket: 'b1',
      targetPrefix: 'p/',
      keys: ['a', 'b'],
      deleteSource: true,
    })
    expect(vi.mocked(toast).mock.calls.map((c) => c[0])).toEqual(['dest.toastWorking', 'dest.toastDone'])
  })

  it('multi copy partial failure shows donePartial toast', async () => {
    vi.mocked(subscribeMigrateEvents).mockImplementation(((_jobId: string, onProgress: (p: MigrateProgress) => void) => {
      queueMicrotask(() => onProgress({ status: 'done', migrated: 1, failed: 1, done: 1, total: 2 }))
      return () => {}
    }) as never)
    const w = mountDest({ kind: 'multi', keys: ['a', 'b'] })
    await openDialog(w)
    clickBody('common.copy')
    await flushPromises()
    expect(toast).toHaveBeenLastCalledWith('dest.toastDonePartial')
  })

  it('open→false 走 watcher 的 early-return：不再重新拉取桶列表', async () => {
    const w = mountDest()
    await openDialog(w)
    expect(vi.mocked(s3api.listBuckets)).toHaveBeenCalledTimes(1)
    await w.setProps({ open: false })
    await flushPromises()
    expect(vi.mocked(s3api.listBuckets)).toHaveBeenCalledTimes(1)
    expect(w.emitted('error')).toBeUndefined()
  })

  it('ModalDialog 自身的 close 事件转发为父级 close', async () => {
    const w = mountDest()
    await openDialog(w)
    w.findComponent({ name: 'ModalDialog' }).vm.$emit('close')
    expect(w.emitted('close')).toBeTruthy()
  })

  it('listBuckets 响应缺 buckets 键 → 桶列表为空', async () => {
    vi.mocked(s3api.listBuckets).mockResolvedValue({} as unknown as Awaited<ReturnType<typeof s3api.listBuckets>>)
    const w = mountDest()
    await w.setProps({ open: true })
    await flushPromises()
    expect(selectEl().options).toHaveLength(1)
    expect(selectEl().options[0].value).toBe('')
  })

  it('进度事件缺 migrated/failed 字段时按 0 计并完成', async () => {
    vi.mocked(subscribeMigrateEvents).mockImplementation(((_jobId: string, onProgress: (p: MigrateProgress) => void) => {
      queueMicrotask(() => onProgress({ status: 'done', done: 1, total: 1 } as never))
      return () => {}
    }) as never)
    const w = mountDest({ kind: 'folder' })
    await openDialog(w)
    clickBody('common.copy')
    await flushPromises()
    expect(toast).toHaveBeenLastCalledWith('dest.toastCopiedN')
    expect(w.emitted('submit')).toBeTruthy()
  })

  it('folder 目标路径以 / 结尾时不重复追加', async () => {
    const w = mountDest({ kind: 'folder' })
    await openDialog(w)
    pathInput().value = 'dir/'
    pathInput().dispatchEvent(new Event('input'))
    await flushPromises()
    clickBody('common.copy')
    await flushPromises()
    expect(vi.mocked(s3api.copyPrefixAsync)).toHaveBeenCalledWith('acc-1', expect.objectContaining({
      targetPrefix: 'dir/',
    }))
  })

  it('multi 无 keys 时传空数组且无斜杠前缀自动补全', async () => {
    const w = mountDest({ kind: 'multi' })
    await openDialog(w)
    // 选中数 n=0（keys?.length ?? 0）
    expect(document.body.textContent).toContain('dest.selected')
    pathInput().value = 'p'
    pathInput().dispatchEvent(new Event('input'))
    await flushPromises()
    clickBody('common.copy')
    await flushPromises()
    expect(vi.mocked(s3api.copyFilesAsync)).toHaveBeenCalledWith('acc-1', expect.objectContaining({
      targetPrefix: 'p/',
      keys: [],
    }))
  })
})
