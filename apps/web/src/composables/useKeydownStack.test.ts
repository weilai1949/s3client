import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { defineComponent, nextTick, ref } from 'vue'
import { pushKeydown, useKeydownStack } from './useKeydownStack'

// 观测方式统一为「派发一次真实 keydown，看 handler 是否被调用」——即**外部可见行为**。
// 此前这里用已删除的 `isTopKeydown(handler)` 直接读内部栈状态，属白盒断言：
// 它既让一个生产代码零引用的导出靠测试「续命」（死代码），又只能证明栈里有谁、
// 证明不了「真的有且只有栈顶收到事件」。改为派发事件后，两者一起被覆盖。

/** 派发一次 Esc keydown。 */
function pressEscape() {
  window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
}

describe('pushKeydown', () => {
  it('push 后 keydown 触达 handler；pop 后不再触达，且监听被摘除', () => {
    const handler = vi.fn()
    const pop = pushKeydown(handler)
    pressEscape()
    expect(handler).toHaveBeenCalledTimes(1)
    pop()
    pressEscape()
    expect(handler).toHaveBeenCalledTimes(1)
  })

  it('只有栈顶 handler 收到 keydown；弹出栈顶后下一个生效', () => {
    const h1 = vi.fn()
    const h2 = vi.fn()
    const pop1 = pushKeydown(h1)
    const pop2 = pushKeydown(h2)
    pressEscape()
    expect(h2).toHaveBeenCalledTimes(1)
    expect(h1).not.toHaveBeenCalled()
    pop2()
    pressEscape()
    expect(h1).toHaveBeenCalledTimes(1)
    pop1()
  })

  it('pop 重复调用是 no-op（lastIndexOf 为 -1 → splice 跳过）', () => {
    const handler = vi.fn()
    const pop = pushKeydown(handler)
    pop()
    pop() // 第二次：handler 已不在栈中，i >= 0 为 false
    pressEscape()
    expect(handler).not.toHaveBeenCalled()
  })

  it('栈空后 dispatch 的空栈分支安全（不抛错、无 handler 可调）', () => {
    const handler = vi.fn()
    const pop = pushKeydown(handler)
    pop()
    expect(() => pressEscape()).not.toThrow()
  })
})

describe('useKeydownStack', () => {
  it('未传 active 时在 mount 注册、unmount 注销', () => {
    const handler = vi.fn()
    const Host = defineComponent({
      setup() {
        useKeydownStack(handler)
        return () => null
      },
    })
    const wrapper = mount(Host)
    pressEscape()
    expect(handler).toHaveBeenCalledTimes(1)
    wrapper.unmount()
    pressEscape()
    expect(handler).toHaveBeenCalledTimes(1)
  })

  it('active 为 true 时注册、转 false 时注销', async () => {
    const handler = vi.fn()
    const active = ref(true)
    const Host = defineComponent({
      setup() {
        useKeydownStack(handler, active)
        return () => null
      },
    })
    const wrapper = mount(Host)
    pressEscape()
    expect(handler).toHaveBeenCalledTimes(1)
    active.value = false
    // 不用 vi.waitFor：它的回调会被重试，而这里必须「派发一次事件」才能观测，
    // 重试会重复派发把计数推高。watch 默认 pre-flush，await nextTick() 即可确定性等到注销完成。
    await nextTick()
    pressEscape()
    expect(handler).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })

  it('连续两次 truthy 激活：activate 的 pop 守卫防重复入栈', async () => {
    const handler = vi.fn()
    // 用对象承载 truthy 值：每次替换触发 watch，但两次都是 truthy → activate 连续执行
    const active = ref<unknown>({ on: true })
    const Host = defineComponent({
      setup() {
        useKeydownStack(handler, active as never)
        return () => null
      },
    })
    const wrapper = mount(Host)
    pressEscape()
    expect(handler).toHaveBeenCalledTimes(1)
    active.value = { on: true, n: 2 } // 第二次 truthy 触发 activate → pop 已存在 → early-return
    await nextTick()
    pressEscape()
    // 未重复入栈：每次 keydown 只被调用一次（若入栈两次，dispatch 只调栈顶，仍是 1 次；
    // 故另用 unmount 后的调用次数钉住「栈里确实只有一份」）。
    expect(handler).toHaveBeenCalledTimes(2)
    wrapper.unmount()
    pressEscape()
    expect(handler).toHaveBeenCalledTimes(2)
  })
})
