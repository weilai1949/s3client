import { describe, expect, it } from 'vitest'
import { fmtBytes, fmtCost } from './storageReport'

// 纯格式化辅助的行为测试（ROADMAP §三 #7）：覆盖各量级与非法输入，钉住展示口径。

describe('fmtBytes', () => {
  it('≤0 与非有限值按 0 处理', () => {
    expect(fmtBytes(0)).toBe('0 B')
    expect(fmtBytes(-5)).toBe('0 B')
    expect(fmtBytes(Number.NaN)).toBe('0 B')
  })

  it('小于 1 KiB 取整字节', () => {
    expect(fmtBytes(512)).toBe('512 B')
  })

  it('≥1024 按 1024 进制保留一位小数', () => {
    expect(fmtBytes(1536)).toBe('1.5 KiB')
    expect(fmtBytes(1024 ** 3)).toBe('1.0 GiB')
  })

  it('超出最大单位时停在 TiB', () => {
    expect(fmtBytes(1024 ** 5)).toBe('1024.0 TiB')
  })
})

describe('fmtCost', () => {
  it('固定两位小数美元', () => {
    expect(fmtCost(0.53)).toBe('$0.53')
    expect(fmtCost(2)).toBe('$2.00')
  })

  it('非有限值按 0', () => {
    expect(fmtCost(Number.POSITIVE_INFINITY)).toBe('$0.00')
  })
})
