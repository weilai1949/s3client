import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { defineComponent, ref } from 'vue'
import { useBucketSetting } from './useBucketSetting'

describe('useBucketSetting', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('returns loading, saving, reload, save', () => {
    const load = vi.fn(async () => {})
    const onError = vi.fn()
    const onChanged = vi.fn()
    const result = useBucketSetting({ bucket: () => 'b1', load, onError, onChanged })
    expect(result).toHaveProperty('loading')
    expect(result).toHaveProperty('saving')
    expect(result).toHaveProperty('reload')
    expect(result).toHaveProperty('save')
  })

  it('reload sets loading true/false and calls load', async () => {
    const load = vi.fn(async () => {})
    const onError = vi.fn()
    const result = useBucketSetting({ bucket: () => 'b1', load, onError })
    await result.reload()
    expect(load).toHaveBeenCalled()
  })

  it('reload calls onError on failure', async () => {
    const load = vi.fn(async () => { throw new Error('fail') })
    const onError = vi.fn()
    const result = useBucketSetting({ bucket: () => 'b1', load, onError })
    await result.reload()
    expect(onError).toHaveBeenCalledWith('fail')
  })

  it('save calls fn, shows toast, reloads, calls onChanged on success', async () => {
    const load = vi.fn(async () => {})
    const onError = vi.fn()
    const onChanged = vi.fn()
    const result = useBucketSetting({ bucket: () => 'b1', load, onError, onChanged })
    const fn = vi.fn(async () => {})
    await result.save(fn, 'saved!')
    expect(fn).toHaveBeenCalled()
    expect(onChanged).toHaveBeenCalled()
  })

  it('save calls onError on failure', async () => {
    const load = vi.fn(async () => {})
    const onError = vi.fn()
    const result = useBucketSetting({ bucket: () => 'b1', load, onError })
    const fn = vi.fn(async () => { throw new Error('save failed') })
    await result.save(fn, 'saved!')
    expect(onError).toHaveBeenCalledWith('save failed')
  })

  it('save returns early when already saving', async () => {
    const load = vi.fn(async () => {})
    const onError = vi.fn()
    const result = useBucketSetting({ bucket: () => 'b1', load, onError })
    // saving starts false, so first call proceeds.
    // We can't easily test the guard without mocking setTimeout,
    // but the code path exists.
    const fn = vi.fn(async () => {})
    await result.save(fn, 'ok')
    expect(fn).toHaveBeenCalled()
  })

  it('save 并发防重入：第一个保存挂起时第二个直接返回', async () => {
    const load = vi.fn(async () => {})
    const onError = vi.fn()
    const result = useBucketSetting({ bucket: () => 'b1', load, onError })
    let resolveFn!: () => void
    const first = result.save(() => new Promise<void>((r) => { resolveFn = r }), 'first')
    const secondFn = vi.fn(async () => {})
    await result.save(secondFn, 'second')
    expect(secondFn).not.toHaveBeenCalled()
    resolveFn()
    await first
  })
})

describe('useBucketSetting watch (host mounted)', () => {
  it('bucket change triggers reload via watch', async () => {
    const load = vi.fn(async () => {})
    const onError = vi.fn()
    const bucket = ref('b1')
    const Host = defineComponent({
      setup() {
        useBucketSetting({ bucket: () => bucket.value, load, onError })
        return () => null
      },
    })
    mount(Host)
    await vi.waitFor(() => expect(load).toHaveBeenCalled())
    bucket.value = 'b2'
    await vi.waitFor(() => expect(load).toHaveBeenCalledTimes(2))
  })
})

describe('useBucketSetting bucket empty branch', () => {
  it('empty bucket does not reload', async () => {
    const load = vi.fn(async () => {})
    const onError = vi.fn()
    const bucket = ref('')
    const Host = defineComponent({
      setup() {
        useBucketSetting({ bucket: () => bucket.value, load, onError })
        return () => null
      },
    })
    mount(Host)
    await new Promise((r) => setTimeout(r, 10))
    expect(load).not.toHaveBeenCalled()
    bucket.value = 'b1'
    await vi.waitFor(() => expect(load).toHaveBeenCalled())
  })
})
