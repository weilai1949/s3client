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

  it('stop() 后在途探测失败不再续跑（已停止即终结）', async () => {
    let rejectProbe!: (e: Error) => void
    vi.mocked(api.health).mockImplementationOnce(() => new Promise((_, rej) => (rejectProbe = rej)))
    const { w, poll } = host({ intervalMs: 1000, onRecover: vi.fn() })
    poll().start()
    await vi.advanceTimersByTimeAsync(1000)
    poll().stop()
    rejectProbe(new Error('down'))
    await vi.advanceTimersByTimeAsync(10_000)
    expect(api.health).toHaveBeenCalledTimes(1)
    expect(poll().polling.value).toBe(false)
    w.unmount()
  })

  it('卸载后再 start() 不会重新开始轮询', async () => {
    const { w, poll } = host({ intervalMs: 1000, onRecover: vi.fn() })
    w.unmount()
    poll().start()
    expect(poll().polling.value).toBe(false)
    await vi.advanceTimersByTimeAsync(10_000)
    expect(api.health).not.toHaveBeenCalled()
  })

  it('卸载时在途探测成功：不再回调 onRecover，也不重启轮询', async () => {
    let resolveProbe!: (v: { status: string; version: string }) => void
    vi.mocked(api.health).mockImplementationOnce(() => new Promise((res) => (resolveProbe = res)))
    const onRecover = vi.fn()
    const { w, poll } = host({ intervalMs: 1000, onRecover })
    poll().start()
    await vi.advanceTimersByTimeAsync(1000)

    w.unmount()
    resolveProbe({ status: 'ok', version: 'v1' }) // 探测在卸载后才成功
    await vi.advanceTimersByTimeAsync(5000)
    expect(onRecover).not.toHaveBeenCalled()
    expect(api.health).toHaveBeenCalledTimes(1)
    expect(poll().polling.value).toBe(false)
  })

  it('卸载时在途探测失败后不再续跑（不叠加新定时器）', async () => {
    let rejectProbe!: (e: Error) => void
    vi.mocked(api.health).mockImplementationOnce(() => new Promise((_, rej) => (rejectProbe = rej)))
    const { w, poll } = host({ intervalMs: 1000, onRecover: vi.fn() })
    poll().start()
    await vi.advanceTimersByTimeAsync(1000) // 探测发出，仍未决
    expect(api.health).toHaveBeenCalledTimes(1)

    w.unmount()
    rejectProbe(new Error('down')) // 卸载后才失败
    await vi.advanceTimersByTimeAsync(10_000)
    // 卸载后不得再有探测：轮询已终结
    expect(api.health).toHaveBeenCalledTimes(1)
    expect(poll().polling.value).toBe(false)
  })
})
