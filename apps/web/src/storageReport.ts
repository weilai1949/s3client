/**
 * FinOps 成本看板的纯展示辅助（ROADMAP §三 #7）。
 *
 * 把字节 / 金额格式化抽成独立模块，便于单测与复用；组件只负责取数与渲染。
 */

/** 人类可读字节数：1024 进制、≥1024 保留一位小数，否则取整；非有限或 ≤0 视为 0。 */
export function fmtBytes(n: number): string {
  const units = ['B', 'KiB', 'MiB', 'GiB', 'TiB']
  let v = Number.isFinite(n) && n > 0 ? n : 0
  let i = 0
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024
    i++
  }
  return `${i === 0 ? Math.round(v) : v.toFixed(1)} ${units[i]}`
}

/** 美元金额（两位小数）；非有限视为 0。 */
export function fmtCost(n: number): string {
  return `$${(Number.isFinite(n) ? n : 0).toFixed(2)}`
}
