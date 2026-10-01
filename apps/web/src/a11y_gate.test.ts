import { readFileSync, readdirSync } from 'node:fs'
import { join } from 'node:path'
import { expect, it } from 'vitest'

/**
 * 无障碍回归门禁 —— 把 `docs/accessibility.md` 中**已修**的口径钉住，防止回退。
 *
 * 为什么需要它：`styles.css` 是纯样式表，vitest 跑在 happy-dom 上**不做 CSS 级联**，
 * 因此「焦点轮廓是否覆盖某类控件」「减少动效偏好是否被尊重」这类结论没有任何测试能观测。
 * 这两条此前正是 [`KNOWN_ISSUES.md`](../../docs/KNOWN_ISSUES.md) #67 的开放项：
 *   - `<html lang>` 不随语言更新（运行时行为，另由 `i18n/index.test.ts` 的用例钉住）；
 *   - **未处理 `prefers-reduced-motion`**——全仓 0 命中，而界面有 rise / toast-in / skel /
 *     modal-fade / pop-in 等动效，前庭敏感用户无法关掉；
 *   - **`textarea` 不在统一 `:focus-visible` 列表里**——而表单基础规则写了 `outline: none`，
 *     故 `BucketPolicyVisualEditor.vue` 的 `<textarea>` 拿不到与其它控件一致的 2px 轮廓。
 *
 * 本文件用**源码形态**断言（不引 axe-core / pa11y 等新依赖，见 ADR-004 与
 * `docs/accessibility.md` §4 第 2 条）——这里刻意只用仓库已有的 raw 导入能力。
 * 渲染态的规则集扫描另有两处（2026-09-30 起）：`e2e/a11y.spec.ts`（真实浏览器整页）与
 * `src/a11y_axe.test.ts`（组件挂载态，`vitest-axe` 仅 devDependency）；本文件仍只管源码形态，
 * 三者互补而不重叠。
 */

// 读样式表原文：这里**不能**用 Vite 的 `?raw`——vitest 默认 `css: false` 会把 CSS 模块
// 整体替换为空模块，实测 `import.meta.glob('./*.css', { query: '?raw' })` 能匹配到
// `./styles.css` 这个键，但**取值长度为 0**；静态 `import css from './styles.css?raw'`
// 同样得到空串。（对照：`e2e/screenshots.spec.ts` 能用 `fileURLToPath(import.meta.url)`，
// 那是 Playwright 的真 Node ESM；vitest 的 happy-dom 环境下 `import.meta.url` 是
// `http://` 形式，`fileURLToPath` 会抛「URL must be of scheme file」。）
// 故按 vitest 项目根（= `apps/web`，与 `pnpm test` 的 cwd 一致）取相对路径——
// 下面第一条用例的防空跑断言会兜住「cwd 不对 → 读到 0 字节」这种静默失效。
const stylesCss = readFileSync('src/styles.css', 'utf8')

/** 操作失败横幅的 class 形态（`docs/accessibility.md` §1.4「操作成功 / 失败统一播报」口径）。 */
const FAILURE_BANNER_CLASS = /\bclass="(?:msg err|modal-err)"/

it('门禁前置成立：能读到 styles.css', () => {
  // 防空跑：路径漂移或读取口径失效时不得静默变绿。
  expect(stylesCss.length, '未读到 src/styles.css（读取口径需同步）').toBeGreaterThan(1000)
})

it('操作失败横幅统一进 live region：每个 .msg err / .modal-err 都声明 role="alert"', () => {
  // 成功反馈统一走 `toast()` → `Toasts` 容器的 `aria-live="polite"`（由 Toasts.test.ts 行为断言）；
  // 失败反馈有一部分是**面板内联横幅**，不经过 toast——这些横幅必须自带 `role="alert"`，
  // 否则屏幕阅读器对「错误出现在屏幕上」毫无感知。本门禁防漏加与回退。
  const dir = 'src/components'
  const files = readdirSync(dir).filter((f) => f.endsWith('.vue'))
  expect(files.length, '扫描面塌缩：src/components 下未识别到 .vue').toBeGreaterThan(30)

  const offenders: string[] = []
  for (const f of files) {
    const src = readFileSync(join(dir, f), 'utf8')
    // `[^>]` 亦匹配换行，故多行开标签也能取到；到第一个 '>'截断，不会跨标签。
    for (const tag of src.match(/<[a-zA-Z][^>]*>/g) ?? []) {
      if (FAILURE_BANNER_CLASS.test(tag) && !/role="alert"/.test(tag)) {
        offenders.push(`${f}: ${tag.trim()}`)
      }
    }
  }
  expect(
    offenders,
    `以下操作失败横幅没有 role="alert"（失败不播报，辅助技术无感知）：\n${offenders.join('\n')}`,
  ).toEqual([])
})

it('焦点可见：统一 2px 轮廓的选择器列表覆盖 textarea', () => {
  const m = /([^{}]*:focus-visible[^{}]*)\{([^}]*)\}/.exec(stylesCss)
  expect(m, 'styles.css 找不到 :focus-visible 规则块').not.toBeNull()
  const [, selectors, decls] = m as RegExpExecArray
  // 回归点：#67③ —— 列表里缺 textarea，而 `input, select, textarea { outline: none }` 抹掉了默认轮廓。
  expect(
    selectors,
    `:focus-visible 选择器列表未覆盖 textarea（键盘用户看不出焦点在哪）：\n${selectors}`,
  ).toMatch(/\btextarea:focus-visible\b/)
  expect(decls, ':focus-visible 规则块应给出 2px 轮廓').toMatch(/outline:\s*2px/)
})

it('减少动效：prefers-reduced-motion 媒体查询存在并关闭 animation / transition', () => {
  const idx = stylesCss.indexOf('prefers-reduced-motion')
  // 回归点：#67② —— 此前全仓 0 命中，动效在「减少动效」偏好下照常播放。
  expect(idx, 'styles.css 缺少 prefers-reduced-motion 媒体查询').toBeGreaterThan(-1)
  const block = stylesCss.slice(idx)
  expect(block, 'prefers-reduced-motion 块未关闭 animation').toMatch(/animation:\s*none/)
  expect(block, 'prefers-reduced-motion 块未关闭 transition').toMatch(/transition:\s*none/)
})

/** 组件源码（与 styles.css 同样的相对路径口径，由下方前置用例兜住读取失效）。 */
const componentSources = (() => {
  const dir = 'src/components'
  const out = new Map<string, string>()
  for (const f of readdirSync(dir)) {
    if (f.endsWith('.vue')) out.set(f, readFileSync(join(dir, f), 'utf8'))
  }
  return out
})()

it('门禁前置成立：能读到组件源码', () => {
  expect(componentSources.size, '未读到 src/components/*.vue（读取口径需同步）').toBeGreaterThan(30)
})

it('每张数据表都有 <caption class="sr-only">（表格有可访问名称）', () => {
  const offenders: string[] = []
  for (const [f, src] of componentSources) {
    for (const m of src.matchAll(/<table\b[^>]*>/g)) {
      const after = src.slice(m.index! + m[0].length, m.index! + m[0].length + 120)
      if (!/^\s*<caption class="sr-only">/.test(after)) offenders.push(`${f}: ${m[0].trim()}`)
    }
  }
  expect(
    offenders,
    `以下 <table> 缺 sr-only <caption>（辅助技术读不到表名，且 <caption> 必须是首个子元素）：\n${offenders.join('\n')}`,
  ).toEqual([])
})

/** 有行选中（表内复选框 / 单选框）的组件——新增此类表必须同步登记。 */
const SELECTABLE_TABLES = ['ObjectList.vue', 'MigratePanel.vue', 'AccountsPanel.vue', 'ServerPanel.vue']

it('行选中语义：有行选中的表，其数据行声明 aria-selected', () => {
  for (const f of SELECTABLE_TABLES) {
    const src = componentSources.get(f)
    expect(src, `缺 ${f}`).toBeTruthy()
    expect(src, `${f} 的数据行缺 :aria-selected=`).toMatch(/<tr[^>]*:aria-selected=/)
  }
  // 自检：全仓声明了 :aria-selected 的组件恰好就是这份名单，防扫描面塌缩成「永远为真」。
  //（`aria-multiselectable` 刻意**不**用：axe 的 `aria-allowed-attr` 判定它在原生 `<table>` 上
  //  非法——只有 grid / listbox / select 等选择型角色支持，本仓库不为它把表升级成 role="grid"。）
  const declared = [...componentSources].filter(([, s]) => /:aria-selected=/.test(s))
  expect(declared.map(([f]) => f).sort()).toEqual([...SELECTABLE_TABLES].sort())
})
