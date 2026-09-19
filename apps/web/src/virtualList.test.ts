import { describe, expect, it } from 'vitest'
import { OVERSCAN, ROW_HEIGHT, virtualWindow } from './virtualList'

/**
 * 窗口化数学（review §F2 的根因之一：起点完全由传入的 scrollTop 决定）。
 * 断言的是窗口边界/垫片高度这些外部可见的输出。
 */
describe('virtualWindow', () => {
  it('顶部：从 0 开始，下方垫片覆盖剩余行', () => {
    const w = virtualWindow(1000, 0, 480, ROW_HEIGHT, OVERSCAN)
    expect(w.start).toBe(0)
    expect(w.end).toBe(Math.ceil(480 / ROW_HEIGHT) + OVERSCAN * 2)
    expect(w.padTop).toBe(0)
    expect(w.padBottom).toBe((1000 - w.end) * ROW_HEIGHT)
  })

  it('中部：起点回退 overscan 行，上下垫片之和 + 窗口行数 = 总行数', () => {
    const scrollTop = ROW_HEIGHT * 100
    const w = virtualWindow(1000, scrollTop, 480, ROW_HEIGHT, OVERSCAN)
    expect(w.start).toBe(100 - OVERSCAN)
    expect(w.padTop).toBe((100 - OVERSCAN) * ROW_HEIGHT)
    expect(w.padTop / ROW_HEIGHT + (w.end - w.start) + w.padBottom / ROW_HEIGHT).toBe(1000)
  })

  it('条目少于窗口容量：不产生负垫片', () => {
    const w = virtualWindow(3, 0, 480, ROW_HEIGHT, OVERSCAN)
    expect(w.end).toBe(3)
    expect(w.padBottom).toBe(0)
  })

  it('scrollTop 为负（越界回弹）：起点钳到 0', () => {
    const w = virtualWindow(10, -500, 480, ROW_HEIGHT, OVERSCAN)
    expect(w.start).toBe(0)
    expect(w.padTop).toBe(0)
  })
})
