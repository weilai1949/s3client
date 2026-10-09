import { expect, it } from 'vitest'
import ts from 'typescript'
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
 *   1. 类型导出（`type X` / `interface X`）自 2026-10-09（评审 R7）起由本文件的
 *      「类型导出半边」覆盖——旧版此处写的是「不覆盖，vue-tsc 已覆盖」，属**误判**
 *      （vue-tsc 不报未使用导出），`types.ts` 的三个死类型因此长期无人发现。
 *      剩余口径限制：`export { type X }` 列表形态、`.d.ts` 生成物、同文件内部引用
 *      （含互引用环）不算缺口，见「类型导出半边」段注释；
 *   2. 动态成员访问（`s3api[name]`）无法静态识别——当前全仓无此写法；
 *   3. B 半边按**引用标识符**计数（TS AST，`.vue` 的 `<template>` 原样按裸词计数）：
 *      同名标识符跨文件抵消仍然存在（与后端 Gate 1 同向，只会漏报）；而**注释与字符串
 *      字面量里的同名整词不再算作引用**——2026-09-29 收口 KNOWN_ISSUES #65，口径与后端
 *      `deadcode_gate_test.go` 的 AST 判定对齐，并用合成源码的口径用例钉住。
 *      `as` 重命名导入与 default 导出会破坏该计数，已用前置断言拦死——出现即红灯，先改写再进门禁。
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

/** 节点是否带 `export` 修饰符。 */
function hasExportModifier(node: ts.Node): boolean {
  const mods = (node as { modifiers?: ts.NodeArray<ts.ModifierLike> }).modifiers
  return mods !== undefined && mods.some((m) => m.kind === ts.SyntaxKind.ExportKeyword)
}

/** 该标识符是否是「被导出的声明」自身的名字（自己不能证明自己被引用）。 */
function isExportedDeclarationName(id: ts.Identifier): boolean {
  const p: ts.Node | undefined = id.parent
  if (!p) return false
  if (ts.isVariableDeclaration(p)) {
    // `export const X` 的 export 修饰符挂在 VariableStatement 上，不在声明本身。
    const stmt: ts.Node | undefined = p.parent?.parent
    return stmt !== undefined && ts.isVariableStatement(stmt) && hasExportModifier(stmt) && p.name === id
  }
  if (
    ts.isFunctionDeclaration(p) ||
    ts.isClassDeclaration(p) ||
    ts.isEnumDeclaration(p) ||
    ts.isModuleDeclaration(p) ||
    // 类型半边（评审 R7）：类型别名与接口的声明名同样不能自证被引用。
    ts.isTypeAliasDeclaration(p) ||
    ts.isInterfaceDeclaration(p)
  ) {
    return hasExportModifier(p) && p.name === id
  }
  return false
}

/**
 * 从 TS 源码里收集「算作引用」的标识符名。
 *
 * 用 TypeScript **AST** 而非正则——正则会被注释与字符串字面量里的同名整词骗过
 * （KNOWN_ISSUES #65 的漏报路径），而与后端 `deadcode_gate_test.go` 的 AST 口径对齐。
 *
 * 排除项（都不是「生产引用」）：
 *   - `import` 语句：说明符是字符串，绑定名是本地别名；
 *   - `export { X }` / `export * from`：导出声明本身；
 *   - 被导出声明**自身的名字**：否则每个导出都会自证被引用。
 *
 * 保留项（沿用旧的裸词口径，避免把真实引用误杀）：类型位置的标识符、属性名、
 * 对象字面量的键、模板字面量 `${…}` 插值内的表达式。注释与字符串字面量天然不在 AST 中。
 */
function collectIdentifiers(code: string): string[] {
  const sf = ts.createSourceFile('gate.ts', code, ts.ScriptTarget.Latest, /* setParentNodes */ true)
  const names: string[] = []
  const visit = (node: ts.Node): void => {
    if (ts.isImportDeclaration(node) || ts.isImportEqualsDeclaration(node)) return
    if (ts.isExportDeclaration(node)) return
    if (ts.isIdentifier(node)) {
      if (!isExportedDeclarationName(node)) names.push(node.text)
      return
    }
    ts.forEachChild(node, visit)
  }
  visit(sf)
  return names
}

/** SFC 的 `<script>`（含 `setup`）与 `<template>` 块。 */
const SFC_SCRIPT_RE = /<script\b[^>]*>([\s\S]*?)<\/script>/gi
const SFC_TEMPLATE_RE = /<template\b[^>]*>([\s\S]*?)<\/template>/i
/**
 * 匹配 SFC 模板里的 HTML 注释。
 *
 * 用 `new RegExp` 而非正则字面量 **不是为了风格**：字面量形式 `/<!--…/` 会被 Semgrep
 * （GitLab SAST 的分析器）判成语法错误——JS 的 Annex B 把 `<!--` 当行注释起始，它的解析器
 * 于是在这里报 `` `/` was unexpected `` 并**整份跳过该文件**（2026-09-29 实跑
 * `make gcl GCL_JOBS=semgrep-sast` 时发现）。换成构造式后该文件回到扫描范围内，行为完全等价。
 */
const HTML_COMMENT_RE = new RegExp('<!--[\\s\\S]*?-->', 'g')

/**
 * 抹掉「不算引用」的文本，返回**只由真实引用构成**的正文，供裸词计数。
 *
 * 为什么 `.vue` 不能整文件丢给 TS 解析：SFC 原始文本不是合法 TS。用扫描器（`createScanner`）
 * 也不行——它对含 `${…}` 插值的模板字面量会**吞掉其后整段文件**（实测 `i18n/index.ts` 的
 * `` s.replaceAll(`{${k}}`, …) `` 之后 393 字节被当成一个模板 token，`setLocale` 随之消失，
 * 直接误报 18 个真实导出为死代码）。故：**只**把 `<script>` 块交给 AST，`<template>` 原样保留
 * （模板里的组件名 / 表达式是真实引用），`<style>` 丢弃。
 */
function usageBody(path: string, text: string): string {
  if (!path.endsWith('.vue')) return collectIdentifiers(text).join(' ')
  const idents: string[] = []
  for (const m of text.matchAll(SFC_SCRIPT_RE)) idents.push(...collectIdentifiers(m[1]))
  const tpl = SFC_TEMPLATE_RE.exec(text)
  const template = (tpl ? tpl[1] : '').replace(HTML_COMMENT_RE, ' ')
  return `${template} ${idents.join(' ')}`
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
  return [base, `${base}.ts`, `${base}.d.ts`, `${base}.vue`, `${base}/index.ts`, `${base}/index.vue`]
}

/**
 * 在给定源码集合上求「零生产引用的非 API 运行期导出」。
 *
 * 抽成**纯函数**以便用合成源码做口径测试（与后端 `deadcode_gate_test.go` 的
 * 「每条判定都配一个用合成源码写的口径测试」同做法）——不依赖「仓库里正好有个反例」
 * 来证明门禁有效。
 */
function findUnreferencedRuntimeExports(
  sources: ReadonlyArray<readonly [string, string]>,
): { red: string[]; scanned: number } {
  const bodies = sources.map(([path, text]) => [path, usageBody(path, text)] as const)
  const red: string[] = []
  let scanned = 0
  for (const [path, text] of sources) {
    if (isApiPath(path)) continue
    for (const name of runtimeExportNames(text)) {
      scanned++
      const re = new RegExp(`\\b${name.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}\\b`)
      if (bodies.some(([, body]) => re.test(body))) continue
      red.push(`${path} → ${name}`)
    }
  }
  return { red: red.sort(), scanned }
}

it('非 API 运行期导出必须被生产代码引用（零引用即死代码）', () => {
  const { red, scanned } = findUnreferencedRuntimeExports(files)
  // 防空跑：解析口径失效时不得静默变绿（真实基数 79 文件 / 108 导出）。
  expect(files.length).toBeGreaterThanOrEqual(70)
  expect(scanned, '未抽取到任何非 API 运行期导出（解析口径失效）').toBeGreaterThanOrEqual(90)
  expect(
    red,
    `以下运行期导出在生产代码中零引用（仅测试引用 = 死代码）：\n  ${red.join('\n  ')}\n` +
      '请删除该导出，并同步改写只引用它的测试。',
  ).toEqual([])
})

/**
 * 口径测试：**注释与字符串字面量里的同名整词不算引用**（KNOWN_ISSUES #65）。
 *
 * 背景：`usageBody()` 此前只用正则剥掉 import 与导出声明，随后按**裸词**计数，于是
 * 「导出零调用，但别处有一句提到它的**注释**」即被判为已引用。真实事故：
 * `useKeydownStack.ts` 的 `isTopKeydown` 唯一调用点早已删除，仅因 `ConfirmDialog.vue`
 * 留了一句说明注释，门禁 8 例全绿。**只会漏报**，但漏报的正是本门禁要拦的东西。
 *
 * 本用例用**合成源码**断言，不依赖仓库里正好存在这样一个反例。
 */
it('口径：注释与字符串里的同名整词不算引用', () => {
  const synthetic: Array<[string, string]> = [
    // 名字只出现在注释里 → 必须判死
    ['./src/only-comment.ts', 'export function ghost() {\n  return 1\n}\n// 说明：ghost 已经没有调用方了\n'],
    // 名字只出现在字符串字面量里 → 必须判死
    ['./src/only-string.ts', 'export function quoted() {\n  return 2\n}\nconst note = \'quoted\'\n'],
    // 名字出现在模板字面量的**字面量文本**里 → 必须判死
    ['./src/only-template.ts', 'export function templated() {\n  return 3\n}\nconst msg = `templated`\n'],
    // 真实调用（含模板字面量插值）→ 不得误判
    ['./src/real-use.ts', 'export function live() {\n  return 4\n}\nconsole.log(live())\n'],
    ['./src/real-interp.ts', 'export function interp() {\n  return 5\n}\nconst m = `v=${interp()}`\n'],
  ]
  const { red } = findUnreferencedRuntimeExports(synthetic)
  expect(red).toEqual([
    './src/only-comment.ts → ghost',
    './src/only-string.ts → quoted',
    './src/only-template.ts → templated',
  ])
  // 反向守卫：真实引用（普通调用 / 模板插值）不得被误杀
  expect(red).not.toContain('./src/real-use.ts → live')
  expect(red).not.toContain('./src/real-interp.ts → interp')
})

// ===== 类型导出半边（评审 R7 收口盲区 1） =====
//
// 背景：`vue-tsc` 不报「导出后无人引用」的类型，运行期半边（runtimeExportNames）
// 又只抽 function/class/const——`types.ts` 的 `StorageClassUsage` / `PrefixUsage` /
// `StorageRecommendation` 三个类型导出因此全仓零引用而无人发现（评审 R7）。
//
// 口径限制（有意接受，与运行期半边的裸词口径同向，只会漏报）：
//   - `export { type X }` / `export type { X } from` 列表形态不在抽取范围
//     （再导出面以声明处为准）；
//   - `.d.ts` 生成物不扫（由 `pnpm gen:api` 与 vue-tsc 负责）；
//   - 同文件内部引用（含自引用 / 互引用环）算引用——引用可达性按裸标识符计。

/** 抽取类型导出名：`export type X = …` / `export interface X …`（含 declare）。 */
function typeExportNames(text: string): string[] {
  const names: string[] = []
  for (const m of text.matchAll(/\bexport\s+(?:declare\s+)?(?:type|interface)\s+([A-Za-z_$][\w$]*)/g)) {
    names.push(m[1])
  }
  return names
}

/**
 * 在给定源码集合上求「零生产引用的类型导出」。
 *
 * 与运行期半边同构：抽成纯函数以便用合成源码做口径测试；引用判定复用
 * `usageBody`（剥 import / 导出声明名，AST 级排除注释与字符串，`.vue` 取
 * script + template）。`.d.ts` 生成物跳过。
 */
function findUnreferencedTypeExports(
  sources: ReadonlyArray<readonly [string, string]>,
): { red: string[]; scanned: number } {
  const bodies = sources.map(([path, text]) => [path, usageBody(path, text)] as const)
  const red: string[] = []
  let scanned = 0
  for (const [path, text] of sources) {
    if (path.endsWith('.d.ts')) continue
    for (const name of typeExportNames(text)) {
      scanned++
      const re = new RegExp(`\\b${name.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}\\b`)
      if (bodies.some(([, body]) => re.test(body))) continue
      red.push(`${path} → ${name}`)
    }
  }
  return { red: red.sort(), scanned }
}

it('类型导出必须被生产代码引用（零引用即死代码，评审 R7）', () => {
  const { red, scanned } = findUnreferencedTypeExports(files)
  // 防空跑：解析口径失效时不得静默变绿（真实基数见扫描数下限）。
  expect(files.length).toBeGreaterThanOrEqual(50)
  expect(scanned, '未抽取到任何类型导出（解析口径失效）').toBeGreaterThanOrEqual(40)
  expect(
    red,
    `以下类型导出在生产代码中零引用（仅测试引用 = 死代码）：\n  ${red.join('\n  ')}\n` +
      '请删除该导出，并同步改写只引用它的测试。',
  ).toEqual([])
})

it('口径：类型导出的声明名自身不算引用；注释里的同名整词也不算', () => {
  const synthetic: Array<[string, string]> = [
    // 只有声明本身 → 必须判死（若声明名自证被引用，此例假绿）
    ['./src/dead-type.ts', 'export type Lonely = string\n'],
    // 只在注释里被提到 → 必须判死（AST 级口径排除注释）
    ['./src/comment-only-type.ts', 'export interface Mentioned { a: number }\n// Mentioned 已无人使用\n'],
    // 同文件内部真实引用（类型标注） → 不得误杀
    ['./src/used-here.ts', 'export type Local = string\nexport const v: Local = "x"\n'],
  ]
  const { red } = findUnreferencedTypeExports(synthetic)
  expect(red).toEqual(['./src/comment-only-type.ts → Mentioned', './src/dead-type.ts → Lonely'])
})

it('口径：跨文件的真实类型引用保活，不得误杀', () => {
  const synthetic: Array<[string, string]> = [
    ['./src/only-decl.ts', 'export type Shared = string\n'],
    ['./src/uses-shared.ts', 'export type Wrap = Shared[]\n'],
  ]
  const { red } = findUnreferencedTypeExports(synthetic)
  // Shared 被 uses-shared 引用 → 保活；Wrap 自身零引用 → 照常判死（两条分开断言）。
  expect(red).not.toContain('./src/only-decl.ts → Shared')
  expect(red).toContain('./src/uses-shared.ts → Wrap')
})

/**
 * 导出面死代码的源码形态补拦（review-2026-09-24 Nit「导出面死代码」）。
 *
 * 上面 B 半边按裸词计数：`updateToast` 这种「导出后只在**本模块内部**被调用」的符号，
 * 在 `usageBody`（剥掉导出声明后的正文）里照样出现，会被算成生产引用 → 「全仓无人
 * import」的导出能全绿混过，属该口径的结构性盲区（不是漏扫，是口径本身看不见）。
 * 因此这里直接断言源码形态：下列符号必须是模块私有，源码里不得出现
 * `export function updateToast` / `export const UPLOAD_CONCURRENCY` / `export { X }`。
 *
 * 允许的收口方式只有两种：删掉实现（若连内部引用一起消失），或让它真正被外部模块
 * 引用（那就要同时满足 B 半边的生产引用口径）。二者都不是时本用例保持红灯。
 */
const MODULE_PRIVATE_NAMES = ['updateToast', 'applyTheme', 'UPLOAD_CONCURRENCY']

/** `export function X` / `export const X` 形态。 */
function isDirectExport(text: string, name: string): boolean {
  return new RegExp(`\\bexport\\s+(?:async\\s+)?(?:function\\*?|class|const|let|var)\\s+${name}\\b`).test(text)
}

/** `export { X }` / `export { X as Y }` / `export { X } from ...` 列表形态（对外名取原名 X）。 */
function isListedExport(text: string, name: string): boolean {
  return new RegExp(`\\bexport\\s*\\{[^}]*\\b${name}\\b[^}]*\\}(?:\\s*from\\s*['"][^'"]+['"])?`).test(text)
}

it('仅模块内部使用的实现必须是模块私有（导出面死代码，B 半边裸词计数的盲区）', () => {
  const offenders: string[] = []
  const missing: string[] = []
  for (const name of MODULE_PRIVATE_NAMES) {
    const decl = new RegExp(`\\b(?:function|const|let|var)\\s+${name}\\b`)
    if (!files.some(([, text]) => decl.test(text))) {
      missing.push(name)
      continue
    }
    for (const [path, text] of files) {
      if (!decl.test(text)) continue
      if (isDirectExport(text, name) || isListedExport(text, name)) offenders.push(`${path} → export ${name}`)
    }
  }
  // 防空跑：符号被整段删除时不得静默变绿（本用例描述的符号必须真实存在且非导出）。
  expect(missing, `MODULE_PRIVATE_NAMES 中的符号已从源码消失，请同步更新本用例：${missing.join(', ')}`).toEqual([])
  expect(
    offenders,
    `以下实现只在本模块内部使用，却带着 export（导出面死代码）：\n  ${offenders.join('\n  ')}\n` +
      '去掉 export 关键字（保留实现与内部调用），并把直接 import 它的测试改写为公开入口断言。',
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
