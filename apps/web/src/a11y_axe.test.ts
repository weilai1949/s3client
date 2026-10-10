import { afterEach, describe, expect, it, vi } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import { nextTick } from 'vue'
import { axe } from 'vitest-axe'

import ModalDialog from './components/ModalDialog.vue'
import ConfirmDialog from './components/ConfirmDialog.vue'
import PromptDialog from './components/PromptDialog.vue'
import Toasts from './components/Toasts.vue'
import ObjectList from './components/ObjectList.vue'
import { confirmState } from './confirm'
import { promptState } from './prompt'
import { toasts } from './store'
import type { Entry } from './types'

/**
 * 组件级无障碍扫描（ROADMAP §三 #17④）。
 *
 * 与 [`e2e/a11y.spec.ts`](../e2e/a11y.spec.ts) 的分工：E2E 那侧扫**真实浏览器 + 真实构建产物**
 * 的整页状态，但只有 4 个初始界面状态、覆盖不到组件的边界态（对话框打开、toast 堆叠、
 * 列表带选中行……）。本文件对**挂载后的组件 DOM** 跑同一套 WCAG 规则集，补这块。
 *
 * 判定口径与 E2E 一致：只把 **serious / critical** 判失败（moderate / minor 打印供人工判断），
 * 避免为规则洁癖加一堆无依据的屏蔽项。
 *
 * happy-dom 的限制：**不做 CSS 级联 / 布局**，因此 `color-contrast` 规则在这里判不了对比度
 * （既报不出真问题，也可能误报）——本文件显式关掉它，对比度由 E2E 侧的 axe 与
 * `apps/server/contrast_gate_test.go` 的静态 token 表负责（见 docs/accessibility.md §5.3）。
 */

/** 阻塞级影响面：serious / critical 一律不得出现。 */
const BLOCKING_IMPACTS = new Set(['serious', 'critical'])

/** axe 规则集：WCAG 2.0 / 2.1 的 A + AA（与 e2e/a11y.spec.ts 逐字一致）。 */
const WCAG_TAGS = ['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa']

function formatViolations(
  violations: { id: string; impact?: string | null; help: string; nodes: unknown[] }[],
): string {
  return violations
    .map((v) => `  [${v.impact ?? 'unknown'}] ${v.id}: ${v.help}（${v.nodes.length} 个节点）`)
    .join('\n')
}

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
  confirmState.open = false
  promptState.open = false
  toasts.splice(0, toasts.length)
  vi.clearAllMocks()
})

/** 对挂载出的 DOM 跑一次 axe：引擎自检 + 扫描面自检 + 阻塞级违规断言。 */
async function scan(root: Element, label: string): Promise<void> {
  const results = await axe(root, {
    runOnly: { type: 'tag', values: WCAG_TAGS },
    rules: { 'color-contrast': { enabled: false } },
  })
  expect(results.testEngine.name, 'axe 引擎未生效').toBe('axe-core')
  expect(results.passes.length, `${label}：axe 没有产出任何通过项，扫描面疑似为空`).toBeGreaterThan(0)

  const blocking = results.violations.filter((v) => BLOCKING_IMPACTS.has(v.impact ?? ''))
  expect(
    blocking,
    `${label}：出现 ${blocking.length} 个 serious/critical 无障碍违规：\n${formatViolations(blocking)}`,
  ).toEqual([])
  // 非阻塞（moderate / minor）照 E2E 的口径打印供人工判断，但不红灯。
  const advisory = results.violations.filter((v) => !BLOCKING_IMPACTS.has(v.impact ?? ''))
  if (advisory.length) console.warn(`${label} 非阻塞违规：\n${formatViolations(advisory)}`)
}

function file(key: string): Entry {
  return {
    kind: 'file',
    key,
    name: key,
    size: 2048,
    lastModified: '2024-06-01T10:00:00Z',
    object: { key, size: 2048, lastModified: '2024-06-01T10:00:00Z', etag: 'e1', isDir: false, storageClass: 'STANDARD' },
  }
}

describe('组件级 axe 扫描', () => {
  it('空跑自检：注入已知违规必须被报出（否则 0 违规只是扫描器没生效）', async () => {
    const bad = document.createElement('img')
    bad.setAttribute('src', 'x.png')
    document.body.appendChild(bad)
    const results = await axe(bad, {
      runOnly: { type: 'tag', values: WCAG_TAGS },
      rules: { 'color-contrast': { enabled: false } },
    })
    expect(results.violations.map((v) => v.id)).toContain('image-alt')
    document.body.removeChild(bad)
  })

  it('ModalDialog（含 footer 插槽）', async () => {
    const w = mount(ModalDialog, {
      props: { open: true, title: '测试弹窗' },
      slots: { default: '<p>正文</p>', footer: '<button>保存</button>' },
      attachTo: document.body,
    })
    mounted.push(w)
    await nextTick()
    await scan(document.body, 'ModalDialog')
  })

  it('ConfirmDialog（alertdialog，危险态）', async () => {
    const w = mount(ConfirmDialog, { attachTo: document.body })
    mounted.push(w)
    confirmState.title = '删除对象？'
    confirmState.message = '此操作不可撤销。'
    confirmState.danger = true
    confirmState.confirmText = '删除'
    confirmState.open = true
    await nextTick()
    await scan(document.body, 'ConfirmDialog')
  })

  it('PromptDialog（带校验失败文案）', async () => {
    const w = mount(PromptDialog, { attachTo: document.body })
    mounted.push(w)
    promptState.title = '重命名'
    promptState.label = '新名称'
    promptState.confirmText = '重命名'
    promptState.error = '名称已存在'
    promptState.open = true
    await nextTick()
    await scan(document.body, 'PromptDialog')
  })

  it('Toasts（成功 + 失败堆叠）', async () => {
    toasts.push({ id: 1, kind: 'ok', text: '上传完成' })
    toasts.push({ id: 2, kind: 'err', text: '上传失败' })
    const w = mount(Toasts, { attachTo: document.body })
    mounted.push(w)
    await nextTick()
    await scan(document.body, 'Toasts')
  })

  it('ObjectList（带 caption、选中态与可排序表头的列表视图）', async () => {
    class RO {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
    vi.stubGlobal('ResizeObserver', RO)
    const w = mount(ObjectList, {
      props: {
        entries: [file('a.txt'), file('b.txt')],
        bucketView: 'list',
        selected: new Set(['a.txt']),
        sortKey: 'name',
        sortDir: 1,
        filter: '',
        filterActive: false,
        loading: false,
        totalCount: 2,
        nextToken: '',
        isTruncated: false,
        loadingAll: false,
        listGen: 0,
        ctxEntryKey: null,
      },
      attachTo: document.body,
    })
    mounted.push(w)
    await nextTick()
    await scan(document.body, 'ObjectList')
    vi.unstubAllGlobals()
  })
})
