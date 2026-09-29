import { readFileSync } from 'node:fs'
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
 * `docs/accessibility.md` §4 第 2 条：「没有引入任何自动化可访问性检测工具」——
 * 这里刻意只用仓库已有的 raw 导入能力，不改变该事实）。
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

it('门禁前置成立：能读到 styles.css', () => {
  // 防空跑：路径漂移或读取口径失效时不得静默变绿。
  expect(stylesCss.length, '未读到 src/styles.css（读取口径需同步）').toBeGreaterThan(1000)
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
