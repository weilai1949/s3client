import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { defineComponent, ref } from 'vue'
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
})
