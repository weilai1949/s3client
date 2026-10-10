import { computed, onBeforeUnmount, onMounted, ref, watch, type ComputedRef } from 'vue'
import { DEFAULT_VIEWPORT_H, OVERSCAN, ROW_HEIGHT, virtualWindow } from '../virtualList'

/**
 * 虚拟滚动管线（KNOWN_ISSUES #76）：把此前在 ObjectList / MigratePanel / VersionsDialog /
 * RecycleBinPanel 逐字复制 4 份的「scrollTop/viewportH → windowed 切片 + scroll 监听 +
 * ResizeObserver 测量 + 换源归零」收敛为单一实现。
 *
 * 关键点（各组件注释理由一并上移）：
 *   - **行高单一来源**：切片用 `ROW_HEIGHT`，模板的 `v-row` 内联样式也用同一常量，CSS 不再
 *     另存字面量（此前 CSS 38px 与 ROW_HEIGHT 漂移导致窗口错位，review §F3/§F9③）；
 *   - **ref 绑定后再挂 ResizeObserver**：首屏骨架屏 / 空列表 / 弹窗未开时 `scrollEl` 为
 *     null，`onMounted` 一次性挂载会永久失效（`viewportH` 恒为默认，review §F9②）——
 *     改 `watch(scrollEl)` 在绑定/解绑时测量并挂/断观察器；`onMounted(measureViewport)`
 *     与 `onBeforeUnmount(disconnect)` 只作兜底；
 *   - **换源归零**：调用方在数据源变化时调 `resetWindowScroll()`——残留偏移会让
 *     `items.slice(start, end)` 为空 → 空白表（review §F2）；同时写回真实 DOM scrollTop，
 *     避免下一次滚动事件把陈旧偏移写回。
 *
 * `items` 传响应式数组（通常是 computed）；返回的 `windowed.items` 即窗口内元素。
 */
export function useVirtualRows<T>(items: ComputedRef<T[]>) {
  const scrollEl = ref<HTMLElement | null>(null)
  const scrollTop = ref(0)
  const viewportH = ref(DEFAULT_VIEWPORT_H)

  const windowed = computed(() => {
    const win = virtualWindow(items.value.length, scrollTop.value, viewportH.value, ROW_HEIGHT, OVERSCAN)
    return { ...win, items: items.value.slice(win.start, win.end) }
  })

  function onListScroll() {
    if (scrollEl.value) scrollTop.value = scrollEl.value.scrollTop
  }

  function measureViewport() {
    if (scrollEl.value) viewportH.value = scrollEl.value.clientHeight || DEFAULT_VIEWPORT_H
  }

  let resizeObs: ResizeObserver | undefined

  watch(scrollEl, (el) => {
    resizeObs?.disconnect()
    resizeObs = undefined
    if (!el || typeof ResizeObserver === 'undefined') return
    measureViewport()
    resizeObs = new ResizeObserver(measureViewport)
    resizeObs.observe(el)
  })

  function resetWindowScroll() {
    scrollTop.value = 0
    if (scrollEl.value) scrollEl.value.scrollTop = 0
  }

  onMounted(measureViewport)
  onBeforeUnmount(() => resizeObs?.disconnect())

  return { scrollEl, windowed, onListScroll, resetWindowScroll }
}
