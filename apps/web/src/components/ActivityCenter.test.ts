import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import ActivityCenter from './ActivityCenter.vue'
import { activities, clearActivity, reportActivity } from '../store'

vi.mock('../i18n', () => ({
  t: (k: string) => k,
  tf: (k: string, v: Record<string, unknown>) => `${k}:${JSON.stringify(v)}`,
}))

function reset() {
  for (const key of Object.keys(activities)) clearActivity(key)
}

describe('ActivityCenter', () => {
  beforeEach(reset)
  afterEach(reset)

  it('无活动时不渲染（不占顶栏空间）', () => {
    const w = mount(ActivityCenter)
    expect(w.find('.activity').exists()).toBe(false)
    w.unmount()
  })

  it('total 为 0 的活动视为无活动（过滤掉，不渲染角标）', () => {
    reportActivity({ key: 'upload:panel', label: 'up', done: 0, total: 0, pct: 0 })
    const w = mount(ActivityCenter)
    expect(w.find('.activity').exists()).toBe(false)
    w.unmount()
  })

  it('有活动时渲染角标：百分比取多条平均，aria-label 带条数', () => {
    reportActivity({ key: 'upload:panel', label: 'up', done: 1, total: 4, pct: 25 })
    reportActivity({ key: 'migrate', label: 'mv', done: 8, total: 10, pct: 75 })
    const w = mount(ActivityCenter)

    const btn = w.get('.activity-btn')
    expect(btn.text()).toContain('50%') // (25 + 75) / 2
    expect(btn.attributes('aria-label')).toContain('activity.aria')
    expect(btn.attributes('aria-label')).toContain('"n":2')
    expect(btn.attributes('aria-expanded')).toBe('false')
    w.unmount()
  })

  it('点击角标展开明细（每条任务一行进度），再点收起', async () => {
    reportActivity({ key: 'upload:panel', label: 'up', done: 1, total: 4, pct: 25 })
    reportActivity({ key: 'migrate', label: 'mv', done: 8, total: 10, pct: 75 })
    const w = mount(ActivityCenter)

    await w.get('.activity-btn').trigger('click')
    expect(w.get('.activity-btn').attributes('aria-expanded')).toBe('true')
    const items = w.findAll('.act-item')
    expect(items).toHaveLength(2)
    expect(items[0].text()).toContain('up')
    expect(items[0].text()).toContain('1/4 · 25%')
    expect(items[1].text()).toContain('8/10 · 75%')
    expect(w.get('.act-hint').text()).toBe('activity.hint')

    await w.get('.activity-btn').trigger('click')
    expect(w.findAll('.act-item')).toHaveLength(0)
    w.unmount()
  })

  it('点击弹层外部关闭、点击弹层内部保持打开', async () => {
    reportActivity({ key: 'migrate', label: 'mv', done: 1, total: 2, pct: 50 })
    const w = mount(ActivityCenter, { attachTo: document.body })
    await w.get('.activity-btn').trigger('click')
    expect(w.find('.activity-pop').exists()).toBe(true)

    // 内部点击：保持打开
    await w.get('.act-head').trigger('click')
    expect(w.find('.activity-pop').exists()).toBe(true)

    // 外部点击：关闭
    document.body.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    await nextTick()
    expect(w.find('.activity-pop').exists()).toBe(false)
    w.unmount()
  })

  it('Escape 关闭弹层（键盘用户）', async () => {
    reportActivity({ key: 'migrate', label: 'mv', done: 1, total: 2, pct: 50 })
    const w = mount(ActivityCenter, { attachTo: document.body })
    await w.get('.activity-btn').trigger('click')
    expect(w.find('.activity-pop').exists()).toBe(true)

    // 非 Escape 键不关闭（键栈里只认 Escape）
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowDown', bubbles: true }))
    await nextTick()
    expect(w.find('.activity-pop').exists()).toBe(true)

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))
    await nextTick()
    expect(w.find('.activity-pop').exists()).toBe(false)
    w.unmount()
  })

  it('未展开时点击外部不触发展开关闭逻辑（open 守卫），卸载后监听已移除', async () => {
    reportActivity({ key: 'migrate', label: 'mv', done: 1, total: 2, pct: 50 })
    const w = mount(ActivityCenter, { attachTo: document.body })
    document.body.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    await nextTick()
    expect(w.find('.activity-pop').exists()).toBe(false)

    w.unmount()
    // 卸载后再次派发：监听器已移除，不应抛错
    expect(() => document.body.dispatchEvent(new MouseEvent('click', { bubbles: true }))).not.toThrow()
  })
})
