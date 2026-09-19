import { onBeforeUnmount, ref } from 'vue'
import { api } from '../api'

/**
 * 后端健康轮询与自动恢复（ASSESSMENT S5）。
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
  let disposed = false
  /** 在途探测的代次：stop() / 恢复后自增，使迟到的失败响应不能续跑（review §F9①）。 */
  let gen = 0

  async function probeOnce(myGen: number) {
    try {
      await api.health()
      if (myGen !== gen) return
      stop()
      await opts.onRecover()
    } catch {
      // 已停止/已卸载（代次已变）：丢弃这次探测结果，不再续跑
      if (myGen === gen) schedule()
    }
  }

  function schedule() {
    timer = setTimeout(() => {
      timer = undefined
      void probeOnce(gen)
    }, intervalMs)
  }

  /** 后端出错后调用：开始轮询直到恢复。 */
  function start() {
    if (polling.value || disposed) return
    polling.value = true
    schedule()
  }

  /** 停止轮询（恢复成功或组件卸载）。 */
  function stop() {
    polling.value = false
    gen++
    if (timer !== undefined) {
      clearTimeout(timer)
      timer = undefined
    }
  }

  onBeforeUnmount(() => {
    disposed = true
    stop()
  })

  return { polling, start, stop }
}
