/**
 * 虚拟列表窗口化计算（`ObjectList` 与 `MigratePanel` 共用）。
 *
 * 两个组件此前各自复制了一份「行高 + overscan + slice/垫片」的代码，
 * 常量一旦漂移就会出现滚动窗口与真实行高错位（review §F9③ 即 ROW_HEIGHT=38
 * 与 CSS 行高 42px 不一致）。这里收敛为单一实现，行高由调用方从同一个常量传入。
 */

export interface VirtualWindow {
  start: number
  end: number
  padTop: number
  padBottom: number
}

/** 列表虚拟化的默认行高（px）。
 *
 * 必须与 CSS 行高一致：`styles.css` 的 `table.tbl td { padding: 11px 14px }`
 * + 13px 字号（line-height 1.6 ≈ 21px，content-box 下 21+22=43，实测算得 42px），
 * 模板里 `.tbl-virtual .v-row { height: 42px }` 与之对齐。曾为 38（低于真实行高，
 * 导致滚动窗口与垫片高度错位，review §F9③）。
 */
export const ROW_HEIGHT = 42

/** 上下各多渲染的行数：滚动时留出缓冲，避免快速滚动出现空白。 */
export const OVERSCAN = 12

/** 未测到容器高度时的兜底视口（px）。 */
export const DEFAULT_VIEWPORT_H = 480

/**
 * 计算虚拟窗口。
 *
 * @param total     条目总数
 * @param scrollTop 滚动容器的 scrollTop（进入新列表前必须归零，见 review §F2）
 * @param viewportH 滚动容器可视高度
 * @param rowHeight 行高（必须与渲染用的 CSS 行高一致）
 * @param overscan  上下各多渲染的行数
 */
export function virtualWindow(
  total: number,
  scrollTop: number,
  viewportH: number,
  rowHeight: number,
  overscan: number,
): VirtualWindow {
  const start = Math.max(0, Math.floor(scrollTop / rowHeight) - overscan)
  const count = Math.ceil(viewportH / rowHeight) + overscan * 2
  const end = Math.min(total, start + count)
  return {
    start,
    end,
    padTop: start * rowHeight,
    padBottom: Math.max(0, (total - end) * rowHeight),
  }
}
