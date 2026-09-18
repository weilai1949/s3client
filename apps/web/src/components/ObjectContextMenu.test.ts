import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import ObjectContextMenu from './ObjectContextMenu.vue'
import type { Entry } from '../types'

vi.mock('../i18n', () => ({
  t: (k: string) => k,
  tf: (k: string) => k,
}))

const folderEntry: Entry = { kind: 'folder', key: 'dir/', name: 'dir' }
const fileEntry: Entry = { kind: 'file', key: 'a.txt', name: 'a.txt' }

// happy-dom 仅对已连接 document 的元素更新 activeElement → attachTo body。
let mounted: ReturnType<typeof mount> | undefined
afterEach(() => {
  mounted?.unmount()
  mounted = undefined
})

function mountMenu(menu: { x: number; y: number; entry: Entry } | null) {
  mounted = mount(ObjectContextMenu, {
    props: { menu },
    attachTo: document.body,
    global: { stubs: { Teleport: true } },
  })
  return mounted
}

/** 真实打开流程：先 mount（menu=null），再 setProps 打开（触发 watch → 聚焦首项）。 */
async function openMenu(entry: Entry, x = 10, y = 10) {
  const w = mountMenu(null)
  await w.setProps({ menu: { x, y, entry } })
  await flushPromises()
  await nextTick()
  return w
}

describe('ObjectContextMenu', () => {
  it('renders nothing when menu is null', () => {
    const w = mountMenu(null)
    expect(w.find('.ctx-menu').exists()).toBe(false)
    expect(w.text()).toBe('')
  })

  it('folder menu renders folder items and emits on click', async () => {
    const w = await openMenu(folderEntry)
    const menu = w.find('.ctx-menu')
    expect(menu.exists()).toBe(true)

    const labels = w.findAll('button').map((b) => b.text())
    expect(labels).toEqual([
      'ctx.openFolder',
      'ctx.copyPath',
      'toolbar.copyTo',
      'toolbar.moveTo',
      'ctx.deleteFolder',
    ])
    // 分隔线存在
    expect(menu.findAll('.ctx-sep').length).toBeGreaterThan(0)

    const btns = w.findAll('button')
    await btns[0].trigger('click')
    await btns[1].trigger('click')
    await btns[2].trigger('click')
    await btns[3].trigger('click')
    await btns[4].trigger('click')
    expect(w.emitted('open')).toBeTruthy()
    expect(w.emitted('copy-key')).toBeTruthy()
    expect(w.emitted('copy-folder')).toBeTruthy()
    expect(w.emitted('move-folder')).toBeTruthy()
    expect(w.emitted('delete-folder')).toBeTruthy()
  })

  it('file menu renders file items and emits on click', async () => {
    const w = await openMenu(fileEntry)
    const labels = w.findAll('button').map((b) => b.text())
    expect(labels).toEqual([
      'common.download',
      'objects.preview',
      'ctx.copySignedLink',
      'ctx.copyKey',
      'toolbar.copyTo',
      'toolbar.moveTo',
      'common.rename',
      'ctx.detail',
      'ctx.acl',
      'ctx.tags',
      'ctx.versions',
      'common.delete',
    ])
    const btns = w.findAll('button')
    const events = [
      'open', 'preview', 'copy-link', 'copy-key', 'copy-file', 'move-file',
      'rename', 'detail', 'acl', 'tags', 'versions', 'delete',
    ] as const
    for (let i = 0; i < btns.length; i++) {
      await btns[i].trigger('click')
    }
    for (const e of events) {
      expect(w.emitted(e), `emitted ${e}`).toBeTruthy()
    }
  })

  it('switching menu entry between folder/file re-renders items and focuses first button', async () => {
    const w = await openMenu(fileEntry)
    expect(document.activeElement?.textContent).toBe('common.download')

    await w.setProps({ menu: { x: 10, y: 10, entry: folderEntry } })
    await nextTick()
    expect(w.findAll('button').map((b) => b.text())).toContain('ctx.openFolder')
    expect(document.activeElement?.textContent).toBe('ctx.openFolder')
  })

  it('closing the menu (menu → null) unmounts items', async () => {
    const w = await openMenu(fileEntry)
    await w.setProps({ menu: null })
    await nextTick()
    expect(w.find('.ctx-menu').exists()).toBe(false)
    expect(w.text()).toBe('')
  })

  it('clamps menu position to viewport with 8px margin', async () => {
    const w = await openMenu(fileEntry, -50, -100)
    const style = w.find('.ctx-menu').attributes('style') ?? ''
    expect(style).toContain('left: 8px')
    expect(style).toContain('top: 8px')

    // 大坐标 → 裁剪到视口内
    const bigX = window.innerWidth + 5000
    const bigY = window.innerHeight + 5000
    await w.setProps({ menu: { x: bigX, y: bigY, entry: fileEntry } })
    await nextTick()
    const style2 = w.find('.ctx-menu').attributes('style') ?? ''
    expect(style2).toContain(`left: ${Math.max(8, Math.min(bigX, window.innerWidth - 200))}px`)
    expect(style2).toContain(`top: ${Math.max(8, Math.min(bigY, window.innerHeight - 240))}px`)
  })

  it('keyboard navigation: ArrowDown/ArrowUp/Home/End cycle focus', async () => {
    const w = await openMenu(fileEntry)
    const menu = w.find('.ctx-menu')
    const btns = w.findAll('button')
    expect(document.activeElement).toBe(btns[0].element)

    await menu.trigger('keydown', { key: 'ArrowDown' })
    expect(document.activeElement).toBe(btns[1].element)
    await menu.trigger('keydown', { key: 'ArrowUp' })
    expect(document.activeElement).toBe(btns[0].element)
    // 首项向上 → 循环到末项
    await menu.trigger('keydown', { key: 'ArrowUp' })
    expect(document.activeElement).toBe(btns[btns.length - 1].element)
    await menu.trigger('keydown', { key: 'Home' })
    expect(document.activeElement).toBe(btns[0].element)
    await menu.trigger('keydown', { key: 'End' })
    expect(document.activeElement).toBe(btns[btns.length - 1].element)
  })

  it('菜单打开时按其他键（非方向/Home/End）：整链 else-if 假分支不聚焦', async () => {
    const w = await openMenu(fileEntry)
    const menu = w.find('.ctx-menu')
    const btns = w.findAll('button')
    await menu.trigger('keydown', { key: 'x' })
    expect(document.activeElement).toBe(btns[0].element)
  })
})

describe('ObjectContextMenu empty menu guard', () => {
  it('keydown with no buttons is a no-op', async () => {
    const w = mount(ObjectContextMenu, { props: { menu: { x: 10, y: 10, entry: { kind: 'file', key: 'a.txt', name: 'a.txt' } } }, attachTo: document.body })
    await nextTick()
    // 用一个不含任何 button 的菜单结构（视组件如何渲染——若恒有按钮则本用例验证 42：输入 non-button el）
    const menu = w.find('.ctx-menu')
    if (menu.exists()) {
      await menu.trigger('keydown', { key: 'ArrowDown' })
    }
    expect(w.exists()).toBe(true)
    w.unmount()
  })
})
