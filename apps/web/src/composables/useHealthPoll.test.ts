import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { defineComponent } from 'vue'
import { useHealthPoll } from './useHealthPoll'
import { api } from '../api'

vi.mock('../api', () => ({ api: { health: vi.fn() } }))

// 受控定时器：手动推进时间，断言轮询节奏。
describe('useHealthPoll', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.mocked(api.health).mockReset()
  })
  afterEach(() => {
    vi.useRealTimers()
  })

  function host(opts: Parameters<typeof useHealthPoll>[0]) {
    let poll!: ReturnType<typeof useHealthPoll>
    const Host = defineComponent({
      setup() {
        poll = useHealthPoll(opts)
        return () => null
      },
    })
    const w = mount(Host)
    return { w, poll: () => poll }
  }

  it('后端仍不可用时按间隔重复探测', async () => {
    vi.mocked(api.health).mockRejectedValue(new Error('down'))
    const onRecover = vi.fn()
    const { w, poll } = host({ intervalMs: 1000, onRecover })
    poll().start()

    await vi.advanceTimersByTimeAsync(1000)
    expect(api.health).toHaveBeenCalledTimes(1)
    await vi.advanceTimersByTimeAsync(1000)
    expect(api.health).toHaveBeenCalledTimes(2)
    expect(onRecover).not.toHaveBeenCalled()
    expect(poll().polling.value).toBe(true)
    w.unmount()
  })

  it('恢复后回调 onRecover 并停止轮询', async () => {
    vi.mocked(api.health).mockResolvedValue({ status: 'ok', version: 'v1' })
    const onRecover = vi.fn()
    const { w, poll } = host({ intervalMs: 1000, onRecover })
    poll().start()

    await vi.advanceTimersByTimeAsync(1000)
    expect(onRecover).toHaveBeenCalledTimes(1)
    expect(poll().polling.value).toBe(false)
    // 停止后不再探测。
    await vi.advanceTimersByTimeAsync(5000)
    expect(api.health).toHaveBeenCalledTimes(1)
    w.unmount()
  })

  it('start 幂等：重复调用不叠加定时器', async () => {
    vi.mocked(api.health).mockRejectedValue(new Error('down'))
    const { w, poll } = host({ intervalMs: 1000, onRecover: vi.fn() })
    poll().start()
    poll().start()
    await vi.advanceTimersByTimeAsync(1000)
    expect(api.health).toHaveBeenCalledTimes(1)
    w.unmount()
  })

  it('卸载时停止轮询（不再探测）', async () => {
    vi.mocked(api.health).mockRejectedValue(new Error('down'))
    const { w, poll } = host({ intervalMs: 1000, onRecover: vi.fn() })
    poll().start()
    w.unmount()
    await vi.advanceTimersByTimeAsync(5000)
    expect(api.health).not.toHaveBeenCalled()
  })
})
