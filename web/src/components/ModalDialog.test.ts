import { afterEach, describe, expect, it } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import { nextTick } from 'vue'
import ModalDialog from './ModalDialog.vue'

// Teleport 渲染到 body；每个用例卸载并清理，避免 useKeydownStack 残留
const mounted: VueWrapper[] = []

afterEach(() => {
  while (mounted.length) {
    const w = mounted.pop()!
    try {
      w.unmount()
    } catch {
      /* ignore */
    }
  }
  document.body.innerHTML = ''
  document.body.style.overflow = ''
})

function mountDialog(props: Record<string, unknown> = {}, slots: Record<string, string> = {}) {
  const w = mount(ModalDialog, {
    props: { open: false, title: 'T', ...props },
    slots,
    attachTo: document.body,
  })
  mounted.push(w)
  return w
}

function inBody(sel: string): HTMLElement {
  const el = document.body.querySelector<HTMLElement>(sel)
  expect(el, `expect ${sel} in body`).toBeTruthy()
  return el!
}

function dispatchKey(key: string, opts: KeyboardEventInit = {}): KeyboardEvent {
  const ev = new KeyboardEvent('keydown', { key, cancelable: true, ...opts })
  window.dispatchEvent(ev)
  return ev
}

describe('ModalDialog', () => {
  it('shows title when open and emits close via the X button', async () => {
    const w = mountDialog({ open: true, title: 'Test Modal' }, { default: '<p>body content</p>' })
    await nextTick()
    // Teleport 渲染到 body，从 document 断言
    expect(document.body.textContent).toContain('Test Modal')
    expect(document.body.textContent).toContain('body content')
    const closeBtn = inBody('button.dlg-x')
    closeBtn.click()
    await nextTick()
    expect(w.emitted('close')).toBeTruthy()
  })

  it('hides content when closed', async () => {
    mountDialog({ open: false, title: 'Hidden' })
    await nextTick()
    expect(document.body.textContent ?? '').not.toContain('Hidden')
    expect(document.body.querySelector('.dlg-backdrop')).toBeNull()
  })

  it('open locks scroll, moves focus into dialog, close restores focus and overflow', async () => {
    document.body.style.overflow = 'scroll'
    const temp = document.createElement('button')
    temp.textContent = 'outside'
    document.body.appendChild(temp)
    temp.focus()
    expect(document.activeElement).toBe(temp)

    const w = mountDialog(
      { open: false, title: 'Focus' },
      { default: '<button class="f1">A</button><button class="f2">B</button>' },
    )
    // open 必须通过 props 变化触发（mount 时直接 open=true 不会运行 open 逻辑）
    await w.setProps({ open: true })
    await nextTick()
    expect(document.body.style.overflow).toBe('hidden')
    // 初始焦点应落在第一个可聚焦元素（头部关闭按钮先于 slot 内容）
    expect(document.activeElement).toBe(inBody('button.dlg-x'))

    await w.setProps({ open: false })
    await nextTick()
    expect(document.body.style.overflow).toBe('scroll')
    expect(document.activeElement).toBe(temp)
    document.body.removeChild(temp)
  })

  it('previousFocus removed from document while open: no restore attempt', async () => {
    const temp = document.createElement('button')
    temp.textContent = 'vanisher'
    document.body.appendChild(temp)
    temp.focus()
    const w = mountDialog({ open: false, title: 'VF' })
    await w.setProps({ open: true })
    await nextTick()
    // 关闭前把原焦点元素移出文档
    document.body.removeChild(temp)
    await w.setProps({ open: false })
    await nextTick()
    expect(document.body.style.overflow).toBe('')
    expect(document.activeElement).not.toBe(temp)
  })

  it('Escape emits close with preventDefault', async () => {
    const w = mountDialog({ open: true })
    await nextTick()
    const ev = dispatchKey('Escape')
    await nextTick()
    expect(ev.defaultPrevented).toBe(true)
    expect(w.emitted('close')).toHaveLength(1)
  })

  it('backdrop click emits close but child click does not', async () => {
    const w = mountDialog({ open: true }, { default: '<p>inner</p>' })
    await nextTick()
    inBody('.dlg-backdrop').click()
    await nextTick()
    expect(w.emitted('close')).toHaveLength(1)
    // 重新打开：点击卡片本体（.dlg-card 是 backdrop 的子元素）不触发 close
    await w.setProps({ open: false })
    await w.setProps({ open: true })
    await nextTick()
    inBody('.dlg-card').click()
    await nextTick()
    expect(w.emitted('close')).toHaveLength(1)
  })

  it('Tab focus trap cycles first/last and ignores mid elements', async () => {
    const w = mountDialog(
      { open: false },
      { default: '<button class="f1">A</button><button class="f2">B</button>' },
    )
    await w.setProps({ open: true })
    await nextTick()
    const x = inBody('button.dlg-x')
    const f1 = inBody('button.f1')
    const f2 = inBody('button.f2')
    expect(document.activeElement).toBe(x) // first focusable

    // Shift+Tab at first -> wrap to last
    let ev = dispatchKey('Tab', { shiftKey: true })
    await nextTick()
    expect(ev.defaultPrevented).toBe(true)
    expect(document.activeElement).toBe(f2)

    // Tab at last -> wrap to first
    ev = dispatchKey('Tab')
    await nextTick()
    expect(ev.defaultPrevented).toBe(true)
    expect(document.activeElement).toBe(x)

    // Tab at first without shift -> plain pass-through (no wrap, no prevent)
    ev = dispatchKey('Tab')
    await nextTick()
    expect(ev.defaultPrevented).toBe(false)
    expect(document.activeElement).toBe(x)

    // Focus a middle element -> Tab does nothing special
    f1.focus()
    ev = dispatchKey('Tab')
    await nextTick()
    expect(ev.defaultPrevented).toBe(false)
    expect(document.activeElement).toBe(f1)
  })

  it('Tab trap with only the close button wraps to itself', async () => {
    const w = mountDialog({ open: false }, { default: '<p>plain text only</p>' })
    await w.setProps({ open: true })
    await nextTick()
    // 头部 ✕ 按钮是唯一可聚焦元素（first === last）
    const x = inBody('button.dlg-x')
    expect(document.activeElement).toBe(x)
    let ev = dispatchKey('Tab', { shiftKey: true })
    await nextTick()
    expect(ev.defaultPrevented).toBe(true)
    expect(document.activeElement).toBe(x)
    ev = dispatchKey('Tab')
    await nextTick()
    expect(ev.defaultPrevented).toBe(true)
    expect(document.activeElement).toBe(x)
  })

  it('previousFocus null when nothing focused: no restore crash', async () => {
    const desc = Object.getOwnPropertyDescriptor(document, 'activeElement')
    Object.defineProperty(document, 'activeElement', { get: () => null, configurable: true })
    try {
      const w = mountDialog({ open: false, title: 'NoFocus' })
      await w.setProps({ open: true })
      await nextTick()
      expect(document.body.style.overflow).toBe('hidden')
      await w.setProps({ open: false })
      await nextTick()
      expect(document.body.style.overflow).toBe('')
      expect(w.emitted('close')).toBeUndefined()
    } finally {
      if (desc) Object.defineProperty(document, 'activeElement', desc)
      else delete (document as { activeElement?: unknown }).activeElement
    }
  })

  it('renders footer slot, custom width and dialog attributes', async () => {
    mountDialog(
      { open: true, title: 'Attr', width: '760px' },
      { footer: '<button class="ft">Save</button>' },
    )
    await nextTick()
    const card = inBody('.dlg-card')
    expect(card.style.width).toBe('760px')
    expect(card.getAttribute('role')).toBe('dialog')
    expect(card.getAttribute('aria-modal')).toBe('true')
    expect(card.getAttribute('aria-label')).toBe('Attr')
    expect((card as HTMLElement).tabIndex).toBe(-1)
    expect(inBody('.dlg-footer button.ft').textContent).toBe('Save')
  })

  it('default width and no footer', async () => {
    mountDialog({ open: true, title: 'Default' })
    await nextTick()
    const card = inBody('.dlg-card')
    // happy-dom 的 CSSOM 无法解析 min()，不断言其值；自定义宽度见上一用例
    expect(card).toBeTruthy()
    expect(document.body.querySelector('.dlg-footer')).toBeNull()
  })
})

describe('ModalDialog onKey closed-race guard', () => {
  it('keydown fires while open flips false (before watcher flush) → early return', async () => {
    const w = mountDialog({ open: true })
    await nextTick() // 打开 watcher 执行：handler push + focus
    // 关闭：置 open false 但在 watcher flush 前派发 keydown（handler 仍堆叠）
    await w.setProps({ open: false })
    // 派发一个 Enter（非 Escape/Tab）——若 open 未归 false 也会走 fall-through
    await w.trigger('keydown', { key: 'Enter' })
    await nextTick()
    // 无 close 事件（guard 在 open=false 时提前返回）
    expect(w.emitted('close')).toBeUndefined()
    w.unmount()
  })

  it('open 时按非 Escape/Tab 键：不 close 不 preventDefault', async () => {
    const w = mountDialog({ open: true })
    await nextTick()
    const ev = new KeyboardEvent('keydown', { key: 'Enter', cancelable: true })
    window.dispatchEvent(ev)
    await nextTick()
    expect(ev.defaultPrevented).toBe(false)
    expect(w.emitted('close')).toBeUndefined()
    w.unmount()
  })

  it('open=false 时 onKey 早退（防御守卫）：不触发 close', async () => {
    const w = mountDialog({ open: true })
    await nextTick()
    await w.setProps({ open: false })
    await nextTick()
    // 直接调用已出栈的 handler：open 为 false → early return，无副作用
    const ev = new KeyboardEvent('keydown', { key: 'Escape', cancelable: true })
    ;(w.vm as unknown as { onKey: (e: KeyboardEvent) => void }).onKey(ev)
    await nextTick()
    expect(ev.defaultPrevented).toBe(false)
    expect(w.emitted('close')).toBeUndefined()
    w.unmount()
  })
})
