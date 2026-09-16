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

describe('i18n 字面量键覆盖', () => {
  it('扫描到源码与字典（前置自检）', () => {
    expect(Object.keys(sources).length).toBeGreaterThan(50)
    expect(Object.keys(dictionaries).length).toBeGreaterThan(3)
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
