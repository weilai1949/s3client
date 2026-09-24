import { expect, it } from 'vitest'
import * as apiModule from './api'
import { api, s3api } from './api'

/**
 * 前端死代码门禁（API 公开面 + 非 API 模块导出）—— 对应后端
 * `apps/server/deadcode_gate_test.go` 的前端半边。
 *
 * 为什么需要它：`pnpm lint`（eslint + vue-tsc）只报**未使用的局部变量**，看不见
 * 「导出后无人引用」的符号；而测试文件的引用会让这类符号一直"活着"。真实事故：
 * `s3api.copyFiles` / `deletePrefix` / `copyPrefix` / `migrate` / `migrateSync`
 * 五个方法只被 `api.test.ts` 调用、生产零引用，在 100% 覆盖率门禁下长期存活
 * （见 docs/archive/review-2026-09-19.md §A3）。
 *
 * 门禁口径分两半：
 *   A. `src/api` 对外暴露的每个成员，必须在**至少一个非测试源文件**里被引用。
 *      - 对象成员（`s3api.*` / `api.*`）：按 import 别名精确匹配 `别名.成员`，
 *        不用裸词匹配——避免 `'migrate'` 这类字符串字面量或 `migrateAsync` 之类的前缀
 *        碰撞造成假绿（这正是子串匹配式门禁的教训）。
 *      - 具名导出：匹配该名字的标识符引用。
 *   B. 非 api 模块（components / composables / store / i18n …）的每个**运行期导出**
 *      必须被生产代码引用（剥掉 import 与导出声明后裸词计数，仅测试引用 = 死）；
 *      每个生产源模块（含 .vue 组件）必须被生产代码 import，孤儿即死代码。
 *      两半共同镜像后端 `apps/server/deadcode_gate_test.go` 的「零生产引用」口径。
 *
 * 盲区（有意接受，写清以免被误读为"该类风险已收敛"）：
 *   1. 不覆盖类型导出（`type X`）——类型仅存在于编译期，`vue-tsc` 已覆盖；
 *   2. 动态成员访问（`s3api[name]`）无法静态识别——当前全仓无此写法；
 *   3. B 半边按裸词计数：同名标识符跨文件抵消、注释/字符串中的同名整词会被算作
 *      引用（只会漏报、不会误报，与后端 Gate 1 的同名抵消同向）；`as` 重命名导入
 *      与 default 导出会破坏该计数，已用前置断言拦死——出现即红灯，先改写再进门禁。
 */

// 用 Vite 的 import.meta.glob 读取源码文本：不引入 node:fs / @types/node（前端依赖最小化），
// 路径也由构建器解析，不存在"cwd 不对导致扫到 0 个文件"的静默失败——下面的自检再兜一层。
const rawSources = import.meta.glob('./**/*.{ts,vue}', {
  query: '?raw',
  import: 'default',
  eager: true,
}) as Record<string, string>

/** 生产源文件（路径 → 文本）：排除测试文件。 */
const files: Array<[string, string]> = Object.entries(rawSources).filter(
  ([path]) => !/\.test\.ts$/.test(path),
)

/** 允许「故意无生产调用」的公开面成员；新增条目必须写明理由与移除条件。 */
const INTENTIONAL_UNUSED: ReadonlyMap<string, string> = new Map([
  // 当前为空：公开面即生产面，无预留符号。
])

/** 匹配"来自 api 模块"的 import 说明符：`./api`、`../api`、`../api/endpoints` 等。 */
const API_SPECIFIER_RE = /(?:^|\/)api(?:\/[\w.-]+)?$/
const IMPORT_RE = /import\s+([\s\S]*?)\s+from\s+['"]([^'"]+)['"]/g

interface ApiRefs {
  /** 对象名（s3api/api）→ 被引用的成员集合。 */
  members: Map<string, Set<string>>
  /** 被引用的具名导出集合。 */
  named: Set<string>
}

/** 解析正文中对 api 模块公开面的引用（按 import 别名绑定，避免裸词碰撞）。 */
function collectRefs(sources: Array<[string, string]>): ApiRefs {
  const members = new Map<string, Set<string>>([
    ['s3api', new Set()],
    ['api', new Set()],
  ])
  const named = new Set<string>()

  for (const [, text] of sources) {
    /** 别名 → 原导出名（对象导出） */
    const objectAliases = new Map<string, string>()
    /** 别名 → 原导出名（具名导出） */
    const namedAliases = new Map<string, string>()
    /** 命名空间别名（`import * as ns`） */
    const namespaces = new Set<string>()

    for (const m of text.matchAll(IMPORT_RE)) {
      const clause = m[1].trim()
      if (!API_SPECIFIER_RE.test(m[2])) continue
      const ns = /^\*\s+as\s+(\w+)$/.exec(clause)
      if (ns) {
        namespaces.add(ns[1])
        continue
      }
      const braces = /\{([\s\S]*)\}/.exec(clause)
      if (!braces) continue
      for (const raw of braces[1].split(',')) {
        const part = raw.trim()
        if (!part || part.startsWith('type ')) continue // 类型导出不参与运行期引用
        const asMatch = /^(\w+)\s+as\s+(\w+)$/.exec(part)
        const original = asMatch ? asMatch[1] : part
        const local = asMatch ? asMatch[2] : part
        if (!/^\w+$/.test(original)) continue
        if (original === 's3api' || original === 'api') objectAliases.set(local, original)
        else namedAliases.set(local, original)
      }
    }

    // 去掉 import 语句后再找引用，否则 `import { s3api }` 自身会被当成一次使用。
    const body = text.replace(IMPORT_RE, '')

    for (const [local, original] of objectAliases) {
      const re = new RegExp(`\\b${local}\\s*\\.\\s*(\\w+)`, 'g')
      for (const m of body.matchAll(re)) members.get(original)?.add(m[1])
    }
    for (const ns of namespaces) {
      // `ns.s3api.x` / `ns.api.x`
      const nested = new RegExp(`\\b${ns}\\s*\\.\\s*(s3api|api)\\s*\\.\\s*(\\w+)`, 'g')
      for (const m of body.matchAll(nested)) members.get(m[1])?.add(m[2])
      // `ns.<具名导出>`
      const flat = new RegExp(`\\b${ns}\\s*\\.\\s*(\\w+)`, 'g')
      for (const m of body.matchAll(flat)) named.add(m[1])
    }
    for (const [local, original] of namedAliases) {
      if (new RegExp(`\\b${local}\\b`).test(body)) named.add(original)
    }
  }
  return { members, named }
}

const refs = collectRefs(files)

/** 公开面 = 两个对象导出的成员 + 运行期具名导出。 */
const surface: string[] = [
  ...Object.keys(s3api).map((n) => `s3api.${n}`),
  ...Object.keys(api).map((n) => `api.${n}`),
  ...Object.keys(apiModule)
    .filter((n) => n !== 'api' && n !== 's3api')
    .map((n) => `export ${n}`),
]

function isUsed(entry: string): boolean {
  if (entry.startsWith('s3api.')) return refs.members.get('s3api')?.has(entry.slice(6)) ?? false
  if (entry.startsWith('api.')) return refs.members.get('api')?.has(entry.slice(4)) ?? false
  return refs.named.has(entry.slice(7))
}

it('门禁自检：扫描范围与公开面解析有效（防空跑变绿）', () => {
  // 路径/解析失效时，下面的下限会立刻失败，而不是"没有违规所以通过"。
  expect(files.length).toBeGreaterThanOrEqual(50)
  expect(Object.keys(s3api).length).toBeGreaterThanOrEqual(40)
  expect(Object.keys(api).length).toBeGreaterThanOrEqual(8)
  const totalRefs =
    (refs.members.get('s3api')?.size ?? 0) + (refs.members.get('api')?.size ?? 0) + refs.named.size
  expect(totalRefs).toBeGreaterThanOrEqual(30)
  expect(surface.length).toBeGreaterThanOrEqual(50)
})

it('API 公开面不允许「导出后无人引用」的死符号', () => {
  const unused = surface
    .filter((entry) => !isUsed(entry))
    .filter((entry) => !INTENTIONAL_UNUSED.has(entry))
    .sort()
  expect(
    unused,
    `以下公开面成员在生产代码中零引用（仅测试引用不算使用）：\n  ${unused.join('\n  ')}\n` +
      '请删除该导出；若确为对外预留，请在 INTENTIONAL_UNUSED 中登记并写明理由。',
  ).toEqual([])
})

it('INTENTIONAL_UNUSED 不得登记已不存在的符号（避免豁免清单腐烂）', () => {
  const stale = [...INTENTIONAL_UNUSED.keys()].filter((k) => !surface.includes(k))
  expect(stale, `豁免清单中的符号已不存在，请删除：${stale.join(', ')}`).toEqual([])
})

/**
 * 测试名不得硬编码源码行号 —— 对应 docs/archive/review-2026-09-19.md §4.2。
 *
 * 真实事故：21 个用例名写着 `（line 125 else）`、`（line 111）` 这类源码行号，而行号
 * **早已过期**（实际 `??` 在 `api/storage.ts:49`、非数组回退在 `:185`，用例名却指向
 * `:125`）。测试于是在描述一个不存在的版本：既误导读者，也让「测试即文档」失效。
 * 行号属于实现细节，不是行为断言，本门禁禁止它出现在用例名里。
 *
 * 允许：注释里引用行号（局部解释）、断言消息里的行号。
 * 禁止：`it(...)` / `test(...)` / `describe(...)` 的名称字符串含行号。
 */
const TEST_NAME_RE = /\b(?:it|test|describe)\s*\(\s*(?:'([^']*)'|"([^"]*)")/g
/** 名称中的行号形态：`line 111`、`180 行`、`（209 行）`、`line 614/611/627`。 */
const LINE_REF_IN_NAME_RE = /\bline\s*\d|\d+\s*行/

it('测试名不得硬编码源码行号（行号会过期，让用例描述一个不存在的版本）', () => {
  const offenders: string[] = []
  let scanned = 0
  for (const [path, text] of Object.entries(rawSources)) {
    if (!/\.test\.ts$/.test(path)) continue
    scanned++
    for (const m of text.matchAll(TEST_NAME_RE)) {
      const name = m[1] ?? m[2] ?? ''
      if (LINE_REF_IN_NAME_RE.test(name)) offenders.push(`${path.replace(/^\.\//, '')}: ${name}`)
    }
  }
  // 防空跑：路径解析失效时不得静默变绿。
  expect(scanned, '未扫描到任何测试文件（解析口径失效）').toBeGreaterThanOrEqual(20)
  expect(
    offenders,
    `以下用例名硬编码了源码行号（改为描述行为，行号请放注释里）：\n  ${offenders.join('\n  ')}`,
  ).toEqual([])
})

// ===== B 半边：非 API 模块导出（盲区 2 收口） =====
// 口径见文件头 B 段：剥掉 import 与导出声明后按裸标识符计数——「声明之外全仓零出现」
// 即零生产引用 = 死代码（仅测试引用同样算死），与后端 Gate 1 完全同向。

/** 是否属于 src/api（其公开面由上面的 API 半边守，这里跳过避免重复口径）。 */
function isApiPath(path: string): boolean {
  return /^\.\/(?:src\/)?api\//.test(path)
}

/** 入口模块：由 index.html 直接加载，没有源内 importer。 */
const ENTRY_MODULE_RE = /^\.\/(?:src\/)?main\.ts$/

/** 抹掉 import 语句与导出声明后的正文：导出名在剩余文本里出现才算被引用。 */
function usageBody(text: string): string {
  return text
    .replace(IMPORT_RE, '')
    .replace(/\bexport\s+(?:async\s+)?(?:function\*?|class|const|let|var)\s+[A-Za-z_$][\w$]*/g, '\u0000')
    .replace(/\bexport\s+\{[^}]*\}/g, '\u0000')
}

/** 抽取运行期导出名：function / const / class / 列表导出（type 与 default 不在此列）。 */
function runtimeExportNames(text: string): string[] {
  const names: string[] = []
  for (const m of text.matchAll(/\bexport\s+(?:async\s+)?(?:function\*?|class|const|let|var)\s+([A-Za-z_$][\w$]*)/g)) {
    names.push(m[1])
  }
  for (const m of text.matchAll(/\bexport\s+\{([^}]*)\}/g)) {
    for (const part of m[1].split(',')) {
      const p = part.trim()
      if (!p || p.startsWith('type ')) continue
      const as = /^([A-Za-z_$][\w$]*)\s+as\s+([A-Za-z_$][\w$]*)$/.exec(p)
      names.push(as ? as[2] : p)
    }
  }
  return names
}

/** 相对说明符 → 候选规范路径（相对 importer 所在目录，补扩展名与目录 index）。 */
function resolveSpec(importer: string, spec: string): string[] {
  if (!spec.startsWith('.')) return []
  const dir = importer.replace(/^\.\//, '').split('/').slice(0, -1)
  for (const part of spec.split('/')) {
    if (part === '' || part === '.') continue
    if (part === '..') dir.pop()
    else dir.push(part)
  }
  const base = dir.join('/')
  return [base, `${base}.ts`, `${base}.vue`, `${base}/index.ts`, `${base}/index.vue`]
}

it('非 API 运行期导出必须被生产代码引用（零引用即死代码）', () => {
  const bodies = files.map(([path, text]) => [path, usageBody(text)] as const)
  const red: string[] = []
  let scanned = 0
  for (const [path, text] of files) {
    if (isApiPath(path)) continue
    for (const name of runtimeExportNames(text)) {
      scanned++
      const re = new RegExp(`\\b${name.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}\\b`)
      if (bodies.some(([, body]) => re.test(body))) continue
      red.push(`${path} → ${name}`)
    }
  }
  // 防空跑：解析口径失效时不得静默变绿（真实基数 79 文件 / 108 导出）。
  expect(files.length).toBeGreaterThanOrEqual(70)
  expect(scanned, '未抽取到任何非 API 运行期导出（解析口径失效）').toBeGreaterThanOrEqual(90)
  expect(
    red.sort(),
    `以下运行期导出在生产代码中零引用（仅测试引用 = 死代码）：\n  ${red.join('\n  ')}\n` +
      '请删除该导出，并同步改写只引用它的测试。',
  ).toEqual([])
})

it('每个生产源模块都必须被生产代码 import（孤儿即死代码）', () => {
  const importedAny = new Set<string>()
  const importedAsComponent = new Set<string>()
  for (const [path, text] of files) {
    for (const m of text.matchAll(IMPORT_RE)) {
      const clause = m[1].trim()
      // `import type { X }` 只消费类型，不算组件/模块被真正使用；default 绑定才说明 .vue 组件在用。
      const typeOnly = /^type\s/.test(clause)
      const hasDefault = !typeOnly && !clause.startsWith('{') && !clause.startsWith('*')
      for (const cand of resolveSpec(path, m[2])) {
        importedAny.add(cand)
        if (hasDefault && cand.endsWith('.vue')) importedAsComponent.add(cand)
      }
    }
    // `export { X } from './y'`（含 type / 命名空间形态）是消费边但不是 import 语句，
    // 漏算它会把被再导出的模块误判成孤儿——api/index.ts 的 `export { directUpload } from './upload'` 即实例。
    for (const m of text.matchAll(/\bexport\s+(?:type\s+)?(?:\*\s*(?:as\s+\w+)?|\{[^}]*\})\s+from\s+['"]([^'"]+)['"]/g)) {
      for (const cand of resolveSpec(path, m[1])) importedAny.add(cand)
    }
  }
  const orphans: string[] = []
  let vueCount = 0
  for (const [path] of files) {
    const norm = path.replace(/^\.\//, '')
    if (path.endsWith('.vue')) {
      vueCount++
      if (!importedAsComponent.has(norm)) orphans.push(`${path}（组件未被生产代码默认导入）`)
    } else if (!ENTRY_MODULE_RE.test(path) && !importedAny.has(norm)) {
      orphans.push(`${path}（模块无人 import）`)
    }
  }
  // 防空跑：真实基数 37 个 .vue。
  expect(vueCount).toBeGreaterThanOrEqual(30)
  expect(
    orphans.sort(),
    `以下生产源模块无人 import（仅测试 import 不算）：\n  ${orphans.join('\n  ')}\n` +
      '请删除该文件，或让生产代码真正用起来。',
  ).toEqual([])
})

it('门禁前置成立：非 API 源码无 default 导出、无 as 重命名 import', () => {
  const defaults: string[] = []
  const aliases: string[] = []
  for (const [path, text] of files) {
    if (isApiPath(path)) continue
    if (/\bexport\s+default\b/.test(text)) defaults.push(path)
    for (const m of text.matchAll(IMPORT_RE)) {
      if (API_SPECIFIER_RE.test(m[2])) continue // api 模块的别名由 API 半边精确匹配，不在此列
      const braces = /\{([^}]*)\}/.exec(m[1])
      if (!braces) continue
      for (const part of braces[1].split(',')) {
        if (/^\w+\s+as\s+\w+$/.test(part.trim())) aliases.push(`${path}: ${part.trim()}`)
      }
    }
  }
  expect(
    defaults,
    '非 API 模块出现 default 导出：本门禁只守具名导出，请改为具名导出或扩展门禁',
  ).toEqual([])
  expect(
    aliases,
    `非 API 模块出现 as 重命名 import（原名裸词计数会失真）：\n  ${aliases.join('\n  ')}\n` +
      '请改回直名 import，或扩展本门禁处理别名',
  ).toEqual([])
})
