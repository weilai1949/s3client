import { nextTick, watch, type Ref, type WatchSource } from 'vue'

/**
 * 模态焦点陷阱。
 *
 * 背景（ROADMAP §三 #17①）：`Tab` 回卷原本只写在 `ModalDialog` 里，`ConfirmDialog` /
 * `PromptDialog` / `PreviewOverlay` 只靠「键栈栈顶独占 + 初始聚焦」，键盘用户能 Tab 逃出对话框。
 * 本组合式把「记住/恢复焦点 + 初始移入 + 首尾回卷」抽成一份实现，四个模态共用。
 *
 * `Tab` 回卷不由本模块自行监听 window——模态各自的 `useKeydownStack` handler 是唯一栈顶，
 * 必须由它调用 `trapTab(e)`，否则与「一次按键只影响最上层模态」的 LIFO 语义冲突。
 */
const FOCUSABLE =
  'a[href], button:not([disabled]), textarea, input, select, [tabindex]:not([tabindex="-1"])'

export interface FocusTrapOptions {
  /** 初始聚焦完成后调用一次（如 `PromptDialog` 需要顺带全选输入框文本）。 */
  afterOpen?: () => void
}

/**
 * @param card    对话框容器的模板 ref（`v-if` 渲染出后才存在）
 * @param active  模态是否打开
 */
export function useFocusTrap(
  card: Ref<HTMLElement | undefined>,
  active: WatchSource<boolean>,
  options?: FocusTrapOptions,
) {
  let previousFocus: HTMLElement | null = null

  function focusables(): HTMLElement[] {
    // 经 `await nextTick()` 后才调用，此时 `v-if` 已渲染出 card。
    return Array.from(card.value!.querySelectorAll<HTMLElement>(FOCUSABLE)).filter(
      (el) => el.offsetParent !== null,
    )
  }

  // 必须 `await nextTick()`：不能用 `flush: 'post'`——`v-model` 是运行时指令，它的 mounted 钩子
  // 同样排在 post 队列里且晚于本 watcher，先跑会拿到尚未写入 DOM 的输入值（全选落空）。
  watch(active, async (on) => {
    if (on) {
      previousFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null
      await nextTick()
      // 打开与关闭 / 卸载发生在相邻 tick 时，`v-if` 已把模板 ref 置空，没有可聚焦容器。
      if (!card.value) return
      // 每个模态至少含一个可聚焦元素（关闭 / 确认按钮），首元素恒存在，无需兜底。
      focusables()[0]!.focus()
      options?.afterOpen?.()
    } else {
      if (previousFocus && document.contains(previousFocus)) {
        previousFocus.focus()
      }
      previousFocus = null
    }
  })

  /**
   * `Tab` 焦点陷阱：在最后一个元素上 `Tab` 回卷到第一个、在第一个元素上 `Shift+Tab` 回卷到最后一个，
   * 中间位置不干预（不 `preventDefault`）。由模态自己的 keydown 栈 handler 调用。
   */
  function trapTab(e: KeyboardEvent) {
    if (e.key !== 'Tab') return
    const els = focusables()
    const first = els[0]
    const last = els[els.length - 1]
    if (e.shiftKey && document.activeElement === first) {
      e.preventDefault()
      last.focus()
    } else if (!e.shiftKey && document.activeElement === last) {
      e.preventDefault()
      first.focus()
    }
  }

  return { trapTab }
}
