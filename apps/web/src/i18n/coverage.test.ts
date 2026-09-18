import { describe, expect, it } from 'vitest'
import { i18nKeyCount } from './index'

// 静态扫描：源码中以字面量形式传给 t()/tf() 的键，必须都能在字典中解析。
//
// 背景（2026-09-16 评估 L4 / R1）：`objects.toastCopyFailed`、`batchEdit.tagsNeedKey`、
// `common.working` 三个键被引用但未定义，用户界面直接显示原始 key。此前的单测用
// `vi.mock('../i18n', () => ({ t: (k) => k }))` 回显 key，因此断言 `toBe('...key')` 恒真，
// 缺失无法被发现。本测试直接读源码与字典，不经过 mock，作为该类缺陷的兜底门禁。
//
// 用 Vite 的 import.meta.glob(?raw) 读取源码文本，而非 node:fs —— 保持前端零额外依赖
// （见 docs/decisions/0004-minimal-frontend-deps.md），也无需 @types/node。
//
// 动态拼接键（如 regions.ts 的 `provider.${p}.label`）是模板字面量，不匹配字面量正则，
// 因此天然被排除，不会误报。

/** 源码文件（排除测试）→ 原始文本。 */
const sources = import.meta.glob('../**/*.{ts,vue}', {
  query: '?raw',
  import: 'default',
  eager: true,
}) as Record<string, string>

/** 字典文件 → 原始文本。 */
const dictionaries = import.meta.glob('./messages/*.ts', {
  query: '?raw',
  import: 'default',
  eager: true,
}) as Record<string, string>

/** 字典中定义的全部键（zh-CN 与 en-US 的并集）。 */
function definedKeys(): Set<string> {
  const keys = new Set<string>()
  for (const [path, text] of Object.entries(dictionaries)) {
    if (path.endsWith('types.ts')) continue
    // 形如 'some.key': '...'，键含命名空间点号
    for (const m of text.matchAll(/['"]([A-Za-z][A-Za-z0-9_]*\.[A-Za-z0-9_.]+)['"]\s*:/g)) {
      keys.add(m[1])
    }
  }
  return keys
}

/** 源码中以字面量传给 t()/tf() 的键 → 引用位置（用于报错定位）。 */
function usedLiteralKeys(): Map<string, string[]> {
  const used = new Map<string, string[]>()
  const re = /\b(?:t|tf)\(\s*(['"])([^'"]+)\1/g
  for (const [path, text] of Object.entries(sources)) {
    if (path.endsWith('.test.ts')) continue
    let m: RegExpExecArray | null
    while ((m = re.exec(text)) !== null) {
      const key = m[2]
      const where = path.replace(/^\.\.\//, '')
      const list = used.get(key)
      if (list) list.push(where)
      else used.set(key, [where])
    }
  }
  return used
}

/** 生产源码中出现的全部带引号字符串字面量（不限 t()/tf() 调用）。
 *
 * 不限于 `t('x')`：部分键以**数据结构字段**形式被引用（如 `bucketPolicy.ts` 的
 * `label: 'policy.tplPublicRead'`、`DestDialog.vue` 三元表达式里的 `'dest.actionCopy'`），
 * 它们不匹配 t()/tf() 正则，但确实在用。用「键文本是否出现」判定可覆盖这两类，
 * 同时仍然排除测试文件（测试引用不能算生产使用，否则死键会被测试永久掩盖）。
 */
function usedKeyTexts(): Set<string> {
  const used = new Set<string>()
  for (const [path, text] of Object.entries(sources)) {
    // 排除测试（测试引用不算生产使用）与**字典自身**（否则每个键都能在
    // messages/*.ts 里找到自己，死键检测恒真）。
    if (path.endsWith('.test.ts')) continue
    if (path.includes('messages/')) continue
    for (const m of text.matchAll(/['"]([A-Za-z][A-Za-z0-9_]*\.[A-Za-z0-9_.]+)['"]/g)) {
      used.add(m[1])
    }
  }
  return used
}

/** 动态拼接键 → 匹配正则：`` `provider.${p}.label` `` → `^provider\.[\w.-]+\.label$`。
 *
 * 这些键由模板字面量拼出，源码中不存在完整字面量，必须按前缀模式豁免，
 * 否则会被误判为死键。目前全仓仅 4 处（provider.group / provider.label /
 * provider.desc / storage.class）。
 */
function dynamicKeyPatterns(): RegExp[] {
  const out: RegExp[] = []
  for (const [path, text] of Object.entries(sources)) {
    if (path.endsWith('.test.ts')) continue
    if (path.includes('messages/')) continue
    for (const m of text.matchAll(/`([^`]*\$\{[^`]*)`/g)) {
      const tpl = m[1]
      // 必须是「字面命名空间前缀 + 插值」形状，如 `provider.group.${id}`。
      // 否则会命中非 i18n 的模板串（`/api/accounts/${id}`、`Bearer ${token}`、
      // `s-${Date.now()}`），其通配正则几乎匹配任意键，导致死键检测恒真。
      if (!/^[A-Za-z][A-Za-z0-9_]*\.[A-Za-z0-9_.]*\$\{/.test(tpl)) continue
      const parts = tpl
        .split(/\$\{[^}]*\}/)
        .map((s) => s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'))
      out.push(new RegExp('^' + parts.join('[A-Za-z0-9_-]+') + '$'))
    }
  }
  return out
}

describe('i18n 字面量键覆盖', () => {
  it('扫描到源码与字典（前置自检）', () => {
    expect(Object.keys(sources).length).toBeGreaterThan(50)
    expect(Object.keys(dictionaries).length).toBeGreaterThan(3)
  })

  // 反向门禁（2026-09-17）：此前只校验「被引用 → 有定义」，没有校验
  // 「有定义 → 被引用」，导致约 40 个死键长期堆积在字典里（2026-09-16 评估 D7，
  // 因 i18n 被排除出覆盖率统计而漏网）。死键会让译者重复劳动、也让「键数」
  // 这一指标虚高。本用例把死键变成红灯，使字典只能增不能烂。
  it('字典中不存在未被引用的死键（防止字典膨胀）', () => {
    const used = usedKeyTexts()
    const dyn = dynamicKeyPatterns()
    const dead: string[] = []
    for (const key of definedKeys()) {
      if (used.has(key)) continue
      if (dyn.some((r) => r.test(key))) continue
      dead.push(key)
    }
    expect(dead.sort()).toEqual([])
  })

  it('每个被引用的字面量键都有定义（防止用户看到原始 key）', () => {
    const defined = definedKeys()
    const missing: string[] = []
    for (const [key, refs] of usedLiteralKeys()) {
      if (!defined.has(key)) missing.push(`${key}  <- ${[...new Set(refs)].join(', ')}`)
    }
    expect(missing.sort()).toEqual([])
  })

  it('字典非空且中英键数一致', () => {
    expect(definedKeys().size).toBeGreaterThanOrEqual(640)
    expect(i18nKeyCount('zh-CN')).toBe(i18nKeyCount('en-US'))
  })
})
