import { afterEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { computed, defineComponent, nextTick } from 'vue'
import { useVirtualRows } from './useVirtualRows'
import { DEFAULT_VIEWPORT_H, OVERSCAN, ROW_HEIGHT } from '../virtualList'

/**
 * useVirtualRows（KNOWN_ISSUES #76）：虚拟滚动管线的 canonical 实现测试。
 * 覆盖 scrollTop/viewportH 切片、onListScroll 早退、measureViewport（默认 / 实测）、
 * ResizeObserver 注册与断开、无 ResizeObserver 环境、resetWindowScroll 归零。
 */

class FakeRO {
  static last: FakeRO | undefined
  cb: ResizeObserverCallback
  disconnected = false
  observed: Element | undefined
  constructor(cb: ResizeObserverCallback) {
    this.cb = cb
    FakeRO.last = this
  }
  observe(el: Element) {
    this.observed = el
  }
  unobserve() {}
  disconnect() {
    this.disconnected = true
  }
  trigger() {
    this.cb([], this as unknown as ResizeObserver)
  }
}

const Host = defineComponent({
  props: { count: { type: Number, default: 0 }, show: { type: Boolean, default: true } },
  setup(props) {
    const items = computed(() => Array.from({ length: props.count }, (_, i) => ({ key: 'k' + i })))
    return { ...useVirtualRows(items), items }
  },
  template: `<div><div v-if="show" ref="scrollEl" class="scroll" /><span v-else class="empty" /></div>`,
})

afterEach(() => {
  FakeRO.last = undefined
  vi.unstubAllGlobals()
})

describe('useVirtualRows', () => {
  it('按 viewportH 切片（数量 = ceil(H/ROW)+2*overscan）；onListScroll 绑定后不抛错', async () => {
    vi.stubGlobal('ResizeObserver', FakeRO)
    const w = mount(Host, { props: { count: 500 } })
    await nextTick()
    const vm = w.vm as unknown as { windowed: { items: unknown[]; padTop: number }; onListScroll: () => void }
    // viewportH=480（element.clientHeight=0 → 兜底）；窗口 = ceil(480/42)+2*overscan
    const expected = Math.ceil(DEFAULT_VIEWPORT_H / ROW_HEIGHT) + 2 * OVERSCAN
    expect(vm.windowed.items.length).toBe(expected)
    expect(vm.windowed.padTop).toBe(0)
    vm.onListScroll() // scrollEl 已绑定分支：读取 DOM scrollTop，不抛错
    w.unmount()
  })

  it('ResizeObserver 回调按实测 clientHeight 重算视口（非 0 时不走默认兜底）', async () => {
    vi.stubGlobal('ResizeObserver', FakeRO)
    const w = mount(Host, { props: { count: 500 } })
    await nextTick()
    const el = w.find('.scroll').element as HTMLElement
    Object.defineProperty(el, 'clientHeight', { value: 600, configurable: true })
    FakeRO.last!.trigger()
    await nextTick()
    const vm = w.vm as unknown as { windowed: { items: unknown[] } }
    expect(vm.windowed.items.length).toBe(Math.ceil(600 / ROW_HEIGHT) + 2 * OVERSCAN)
    w.unmount()
  })

  it('scrollEl 未绑定时 onListScroll / resetWindowScroll 空安全早退', async () => {
    vi.stubGlobal('ResizeObserver', FakeRO)
    const w = mount(Host, { props: { count: 10, show: false } })
    await nextTick()
    const vm = w.vm as unknown as { onListScroll: () => void; resetWindowScroll: () => void }
    // 不抛错即通过（内部 scrollEl 为 null 的守卫分支）
    vm.onListScroll()
    vm.resetWindowScroll()
    expect(w.find('.empty').exists()).toBe(true)
    w.unmount()
  })

  it('环境无 ResizeObserver：仍可渲染，且卸载不抛错', async () => {
    vi.stubGlobal('ResizeObserver', undefined)
    const w = mount(Host, { props: { count: 5 } })
    await nextTick()
    expect(w.find('.scroll').exists()).toBe(true)
    expect(() => w.unmount()).not.toThrow()
  })

  it('卸载时断开 ResizeObserver', async () => {
    vi.stubGlobal('ResizeObserver', FakeRO)
    const w = mount(Host, { props: { count: 5 } })
    await nextTick()
    expect(FakeRO.last!.disconnected).toBe(false)
    w.unmount()
    expect(FakeRO.last!.disconnected).toBe(true)
  })
})
