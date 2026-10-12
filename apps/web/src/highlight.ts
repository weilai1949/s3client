/** 过滤命中片段：`hit` 段渲染为高亮（`<mark>`），其余原样。 */
export interface HighlightSegment {
  text: string
  hit: boolean
}

/**
 * 按查询词切分文本，用于在对象名里高亮命中部分。
 *
 * 为什么需要：过滤后只显示「筛剩的行」，用户看不出**为什么**这一行还在——尤其在
 * 长目录里按片段搜索时。高亮命中片段是列表搜索的通用做法。
 *
 * 约定：大小写不敏感匹配，但输出保留原文大小写（展示层不动原文）；
 * 空查询返回整段未命中，调用方无需分支。
 */
export function highlightSegments(text: string, query: string): HighlightSegment[] {
  const q = query.trim().toLowerCase()
  if (!q) return [{ text, hit: false }]
  const lower = text.toLowerCase()
  const out: HighlightSegment[] = []
  let from = 0
  for (;;) {
    const at = lower.indexOf(q, from)
    if (at < 0) break
    if (at > from) out.push({ text: text.slice(from, at), hit: false })
    out.push({ text: text.slice(at, at + q.length), hit: true })
    from = at + q.length
  }
  // 无命中：整段原样（与空查询同形状，渲染层无需额外分支）
  if (!out.length) return [{ text, hit: false }]
  if (from < text.length) out.push({ text: text.slice(from), hit: false })
  return out
}
