/**
 * 请求代际守卫（2026-10-14 评审 O1 / KNOWN_ISSUES #88）。
 *
 * 收敛仓库里反复手写的「自增计数 → await → 用序号判定响应是否过期」惯用法：
 *
 *   const guard = useSeqGuard()
 *   async function load() {
 *     const seq = guard.next()
 *     loading.value = true
 *     try {
 *       const res = await api()
 *       if (!guard.isCurrent(seq)) return // 过期响应：新请求已接管
 *       apply(res)
 *     } catch (e) {
 *       if (guard.isCurrent(seq)) error.value = toErrorMessage(e)
 *     } finally {
 *       if (guard.isCurrent(seq)) loading.value = false
 *     }
 *   }
 *
 * token 与「哪个计数器」绑定在同一个 guard 对象上：两处独立数据源各持有自己的 guard，
 * 就不会像 review F1（前端 `loadBuckets` / 对象 `load` 共用 `loadSeq`）那样把两个计数器
 * 混用——同一个 guard 的 `next()` / `isCurrent()` 天然成对。
 *
 * `invalidate()` 用于**数据源消失**（如源账号被清空）：令所有在飞请求的 token 立即过期，
 * 但不领取新 token（区别于 `next()`）。
 */
export interface SeqGuard {
  /** 领取下一代 token（自增）；发起请求时调用，把返回值交给 `isCurrent()` 判定。 */
  next(): number
  /** token 是否仍是最新一代：`false` = 该响应已过期，必须丢弃。 */
  isCurrent(token: number): boolean
  /** 作废全部在飞 token（令其过期），不领取新 token。 */
  invalidate(): void
}

export function useSeqGuard(): SeqGuard {
  let seq = 0
  return {
    next: () => ++seq,
    isCurrent: (token) => token === seq,
    invalidate: () => {
      seq++
    },
  }
}
