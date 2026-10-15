import { describe, expect, it } from 'vitest'

import { useSeqGuard } from './useSeqGuard'

describe('useSeqGuard', () => {
  it('next() 每次发放新一代 token：旧 token 立即过期，新的才是当前', () => {
    const guard = useSeqGuard()
    const first = guard.next()
    expect(guard.isCurrent(first)).toBe(true)
    const second = guard.next()
    expect(guard.isCurrent(first)).toBe(false)
    expect(guard.isCurrent(second)).toBe(true)
  })

  it('invalidate() 作废在飞 token 且不发放新 token', () => {
    const guard = useSeqGuard()
    const inFlight = guard.next()
    guard.invalidate()
    expect(guard.isCurrent(inFlight)).toBe(false)
    // invalidate 后 next() 发放的 token 才是当前（计数器只增不减）
    expect(guard.isCurrent(guard.next())).toBe(true)
  })

  it('两个 guard 各自计数、互不影响（不同数据源彼此独立）', () => {
    const buckets = useSeqGuard()
    const objects = useSeqGuard()
    const bucketToken = buckets.next()
    objects.next() // 对象列表推进自己的代次，不得作废桶列表在飞请求（review F1）
    expect(buckets.isCurrent(bucketToken)).toBe(true)
  })
})
