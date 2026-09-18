import { onBeforeUnmount, ref } from 'vue'
import { api } from '../api'

/**
 * 后端健康轮询与自动恢复（roadmap #8 / ASSESSMENT S5）。
 *
 * 此前仅在挂载时 load 一次：后端重启或网络抖动后前端一直停留在错误态，必须手动刷新。
 * 本组合式在「已知后端不可用」时按固定间隔探测 /api/health，一旦恢复即回调
 * onRecover 重新拉取数据并清除错误；未出错时不轮询，避免无谓请求。
 */
export interface HealthPollOptions {
  /** 探测间隔（毫秒）。 */
  intervalMs?: number
  /** 恢复时回调（通常是重新 loadAccounts）。 */
  onRecover: () => void | Promise<void>
}

export function useHealthPoll(opts: HealthPollOptions) {
  const intervalMs = opts.intervalMs ?? 5000
  const polling = ref(false)
  let timer: ReturnType<typeof setTimeout> | undefined

  async function probeOnce() {
    try {
      await api.health()
      stop()
      await opts.onRecover()
    } catch {
      // 仍不可用：安排下一次探测。timer 在触发前已清空，故此处不会叠加定时器。
      schedule()
    }
  }

  function schedule() {
    timer = setTimeout(() => {
      timer = undefined
      void probeOnce()
    }, intervalMs)
  }

  /** 后端出错后调用：开始轮询直到恢复。 */
  function start() {
    if (polling.value) return
    polling.value = true
    schedule()
  }

  /** 停止轮询（恢复成功或组件卸载）。 */
  function stop() {
    polling.value = false
    if (timer !== undefined) {
      clearTimeout(timer)
      timer = undefined
    }
  }

  onBeforeUnmount(stop)

  return { polling, start, stop }
}
