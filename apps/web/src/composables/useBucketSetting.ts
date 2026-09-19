import { ref, watch } from 'vue'
import { toast } from '../store'
import { toErrorMessage } from '../errors'

/**
 * 桶级设置页签的公共骨架：桶变化自动加载（immediate）、loading/saving 状态、
 * 统一错误上报与保存后刷新。让每个页签只剩自己的取数/提交逻辑。
 */
export function useBucketSetting(opts: {
  /** 当前桶（函数式读取，保持响应式）。 */
  bucket: () => string
  /** 加载逻辑（只负责取数并填充各自状态）。 */
  load: () => Promise<void>
  /** 错误上报（通常 emit('error')）。 */
  onError: (msg: string) => void
  /** 保存成功后回调（通常 emit('changed') 刷新桶列表）。 */
  onChanged?: () => void
}) {
  const loading = ref(false)
  const saving = ref(false)
  // reload 竞态守卫：快速切桶/保存后刷新会让多个请求并发，
  // 只有最后一次发起的响应才允许写入状态（ASSESSMENT S8）。
  let reloadSeq = 0

  async function reload() {
    const seq = ++reloadSeq
    loading.value = true
    try {
      await opts.load()
    } catch (e) {
      if (seq === reloadSeq) opts.onError(toErrorMessage(e))
    } finally {
      if (seq === reloadSeq) loading.value = false
    }
  }

  watch(
    () => opts.bucket(),
    (b) => {
      if (b) reload()
    },
    { immediate: true },
  )

  /** 提交（保存/清空统一入口）：成功 toast + 重新加载 + onChanged。 */
  async function save(fn: () => Promise<void>, successMsg: string) {
    if (saving.value) return
    saving.value = true
    try {
      await fn()
      toast(successMsg)
      await reload()
      opts.onChanged?.()
    } catch (e) {
      opts.onError(toErrorMessage(e))
    } finally {
      saving.value = false
    }
  }

  return { loading, saving, reload, save }
}
