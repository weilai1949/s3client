import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import Toasts from './Toasts.vue'
import { dismissToast, toast, toasts } from '../store'

vi.mock('../i18n', () => ({
  t: (k: string) => k,
  tf: (k: string) => k,
}))

describe('Toasts', () => {
  beforeEach(() => {
    toasts.splice(0, toasts.length)
  })

  it('renders ok/err toasts with icon and text, close button dismisses', async () => {
    const onClick = vi.fn()
    toasts.push({ id: 1, kind: 'ok', text: 'uploaded ok' })
    toasts.push({ id: 2, kind: 'err', text: 'upload failed', action: { label: 'view', onClick } })

    const w = mount(Toasts)

    const items = w.findAll('.toast')
    expect(items).toHaveLength(2)
    // ok → '✓' 图标；err → '✕' 图标
    expect(items[0].find('.t-icon').text()).toBe('✓')
    expect(items[1].find('.t-icon').text()).toBe('✕')
    expect(items[0].text()).toContain('uploaded ok')
    expect(items[1].text()).toContain('upload failed')
    // 两条 toast 的关闭按钮（aria-label common.close）
    const closeBtns = w.findAll('button[aria-label="common.close"]')
    expect(closeBtns).toHaveLength(2)
    await closeBtns[1].trigger('click')
    await nextTick()
    expect(w.findAll('.toast')).toHaveLength(1)
    expect(toasts.some((t) => t.id === 2)).toBe(false)
    w.unmount()
  })

  it('renders action button and invokes its onClick', async () => {
    const onClick = vi.fn()
    toasts.push({ id: 3, kind: 'ok', text: 'done', action: { label: '查看对象', onClick } })

    const w = mount(Toasts)
    const actionBtn = w.find('button.toast-action')
    expect(actionBtn.exists()).toBe(true)
    expect(actionBtn.text()).toBe('查看对象')
    await actionBtn.trigger('click')
    expect(onClick).toHaveBeenCalledTimes(1)
    w.unmount()
  })

  it('renders toast without action button when action is absent', () => {
    toasts.push({ id: 4, kind: 'ok', text: 'plain' })
    const w = mount(Toasts)
    expect(w.find('button.toast-action').exists()).toBe(false)
    // 无 action 时仍有 close 按钮
    expect(w.find('button[aria-label="common.close"]').exists()).toBe(true)
    w.unmount()
  })

  it('dismissToast removes gone toast without timer errors', () => {
    // 直接 push（无 timer 注册），dismiss 应安全清理
    toasts.push({ id: 5, kind: 'err', text: 'temporary' })
    dismissToast(5)
    expect(toasts.some((t) => t.id === 5)).toBe(false)
  })

  it('鼠标悬停 / 焦点进入冻结自动消失，移开后按剩余时长续跑', async () => {
    vi.useFakeTimers()
    try {
      const id = toast('hover me')
      const w = mount(Toasts)
      await w.find('.toast').trigger('mouseenter')
      vi.advanceTimersByTime(10_000)
      expect(toasts.some((t) => t.id === id)).toBe(true)

      await w.find('.toast').trigger('mouseleave')
      vi.advanceTimersByTime(3601)
      expect(toasts.some((t) => t.id === id)).toBe(false)

      // 键盘用户 Tab 到关闭 / 动作按钮上（focusin）同样冻结
      const id2 = toast('focus me')
      await nextTick()
      const item = w.findAll('.toast')[0]
      await item.trigger('focusin')
      vi.advanceTimersByTime(10_000)
      expect(toasts.some((t) => t.id === id2)).toBe(true)
      await item.trigger('focusout')
      vi.advanceTimersByTime(3601)
      expect(toasts.some((t) => t.id === id2)).toBe(false)
      w.unmount()
    } finally {
      vi.useRealTimers()
    }
  })

  it('操作成功 / 失败统一经容器的 aria-live 播报', async () => {
    const w = mount(Toasts)
    const region = w.find('[aria-live="polite"]')
    expect(region.exists(), 'toast 容器必须是 aria-live 播报区').toBe(true)

    // 成功进同一播报区
    toasts.push({ id: 6, kind: 'ok', text: '上传完成' })
    await nextTick()
    expect(region.text()).toContain('上传完成')

    // 失败走同一播报区、同一批 DOM
    toasts.push({ id: 7, kind: 'err', text: '上传失败' })
    await nextTick()
    expect(region.text()).toContain('上传失败')

    // 每条 toast 自身也是 role=status（polite 隐式播报区）
    const items = w.findAll('.toast')
    expect(items).toHaveLength(2)
    expect(items.every((el) => el.attributes('role') === 'status')).toBe(true)
    w.unmount()
  })
})
