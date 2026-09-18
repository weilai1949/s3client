import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { defineComponent, nextTick, ref } from 'vue'
import { pushKeydown, isTopKeydown, useKeydownStack } from './useKeydownStack'

describe('pushKeydown / isTopKeydown', () => {
  it('pushKeydown adds handler to stack and returns pop function', () => {
    const handler = vi.fn()
    const pop = pushKeydown(handler)
    expect(isTopKeydown(handler)).toBe(true)
    pop()
    expect(isTopKeydown(handler)).toBe(false)
  })

  it('isTopKeydown returns false when handler is not on top', () => {
    const h1 = vi.fn()
    const h2 = vi.fn()
    const pop1 = pushKeydown(h1)
    const pop2 = pushKeydown(h2)
    expect(isTopKeydown(h1)).toBe(false)
    expect(isTopKeydown(h2)).toBe(true)
    pop2()
    expect(isTopKeydown(h1)).toBe(true)
    pop1()
  })

  it('isTopKeydown returns false when stack is empty', () => {
    const handler = vi.fn()
    const pop = pushKeydown(handler)
    pop()
    expect(isTopKeydown(handler)).toBe(false)
  })

  it('pop 重复调用是 no-op（lastIndexOf 为 -1 → splice 跳过）', () => {
    const handler = vi.fn()
    const pop = pushKeydown(handler)
    pop()
    pop() // 第二次：handler 已不在栈中，i >= 0 为 false
    expect(isTopKeydown(handler)).toBe(false)
    // 监听已移除：dispatch 不再触达 handler
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    expect(handler).not.toHaveBeenCalled()
  })

  it('dispatch calls only the topmost handler', () => {
    const h1 = vi.fn()
    const h2 = vi.fn()
    const pop1 = pushKeydown(h1)
    const pop2 = pushKeydown(h2)
    expect(isTopKeydown(h2)).toBe(true)
    const event = new KeyboardEvent('keydown')
    h2(event)
    expect(h2).toHaveBeenCalledWith(event)
    expect(h1).not.toHaveBeenCalled()
    pop2()
    pop1()
  })

  it('real window keydown dispatches to the topmost handler', () => {
    const h = vi.fn()
    const pop = pushKeydown(h)
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    expect(h).toHaveBeenCalledTimes(1)
    pop()
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    expect(h).toHaveBeenCalledTimes(1)
  })
})

describe('useKeydownStack', () => {
  it('registers handler on mount when no active ref (lines 70-71)', () => {
    const handler = vi.fn()
    const Host = defineComponent({
      setup() {
        useKeydownStack(handler)
        return () => null
      },
    })
    const wrapper = mount(Host)
    expect(isTopKeydown(handler)).toBe(true)
    wrapper.unmount()
    expect(isTopKeydown(handler)).toBe(false)
  })

  it('registers handler when active is true and removes when false', async () => {
    const handler = vi.fn()
    const active = ref(true)
    const Host = defineComponent({
      setup() {
        useKeydownStack(handler, active)
        return () => null
      },
    })
    const wrapper = mount(Host)
    expect(isTopKeydown(handler)).toBe(true)
    active.value = false
    await vi.waitFor(() => expect(isTopKeydown(handler)).toBe(false))
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
    expect(isTopKeydown(handler)).toBe(true)
    active.value = { on: true, n: 2 } // 第二次 truthy 触发 activate → pop 已存在 → early-return
    await nextTick()
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    expect(handler).toHaveBeenCalledTimes(1) // 未重复入栈：handler 不会被调用两次
    wrapper.unmount()
    expect(isTopKeydown(handler)).toBe(false)
  })
})
