/**
 * 桶策略可视化编辑器所需的领域模型。
 *
 * 设计取舍：把 Statement 拆为 Effect / Principal / Action[] / Resource[] / Condition?
 * 不支持嵌套 NotPrincipal / NotAction（国内云厂商控制台一般也不暴露），保持 UX 简单。
 * Sid 可选；同名 Sid 在 S3 后端会被去重/拒绝，编辑时会自动补 UUID。
 */
export interface PolicyStatement {
  /** 可选 sid（AWS 标识）；留空时编辑器自动生成。 */
  sid?: string
  /** Allow / Deny */
  effect: 'Allow' | 'Deny'
  /** Principal：'*' 简写 / 'AWS:arn:...' / '{ "AWS": [...] }'。可视化编辑器只暴露 '*' 与单值 AWS。 */
  principal: string
  /** 动作列表，如 s3:GetObject。 */
  actions: string[]
  /** 资源列表，桶 ARN 或对象 ARN（*）。 */
  resources: string[]
}

export interface PolicyDoc {
  Version: '2012-10-17'
  Statement: PolicyStatement[]
}

/** 把桶策略 JSON 字符串解析为可视化结构；解析失败返回 null，由 UI 走「原始 JSON」分支。 */
export function parsePolicy(raw: string): PolicyDoc | null {
  if (!raw.trim()) return { Version: '2012-10-17', Statement: [] }
  let obj: unknown
  try {
    obj = JSON.parse(raw)
  } catch {
    return null
  }
  if (!obj || typeof obj !== 'object') return null
  const o = obj as Record<string, unknown>
  const version = typeof o.Version === 'string' ? o.Version : '2012-10-17'
  const statements = Array.isArray(o.Statement) ? o.Statement : []
  const out: PolicyStatement[] = []
  for (const s of statements) {
    if (!s || typeof s !== 'object') return null // 出现非对象 = 用户在写复杂结构，回退原始编辑
    const stmt = s as Record<string, unknown>
    if (typeof stmt.Effect !== 'string' || (stmt.Effect !== 'Allow' && stmt.Effect !== 'Deny')) return null
    const principal = normalizePrincipal(stmt.Principal)
    if (principal === null) return null
    const actions = normalizeStringArray(stmt.Action) ?? normalizeStringArray(stmt.NotAction)
    if (!actions) return null
    const resources = normalizeStringArray(stmt.Resource) ?? normalizeStringArray(stmt.NotResource)
    if (!resources) return null
    out.push({
      sid: typeof stmt.Sid === 'string' ? stmt.Sid : undefined,
      effect: stmt.Effect,
      principal,
      actions,
      resources,
    })
  }
  return { Version: version as '2012-10-17', Statement: out }
}

/** 把可视化结构序列化为 S3 兼容的桶策略 JSON。 */
export function serializePolicy(doc: PolicyDoc): string {
  const out: Record<string, unknown> = { Version: doc.Version, Statement: [] as unknown[] }
  const arr = out.Statement as unknown[]
  for (const s of doc.Statement) {
    const stmt: Record<string, unknown> = {
      Sid: s.sid || undefined,
      Effect: s.effect,
      Principal: principalToJSON(s.principal),
      Action: s.actions,
      Resource: s.resources,
    }
    // 清掉 undefined 字段，避免出现在 JSON 里。
    for (const k of Object.keys(stmt)) {
      if (stmt[k] === undefined) delete stmt[k]
    }
    arr.push(stmt)
  }
  return JSON.stringify(out, null, 2)
}

function normalizePrincipal(p: unknown): string | null {
  if (p == null) return null
  if (typeof p === 'string') return p === '*' ? '*' : stripAWSPrefix(p)
  if (typeof p === 'object') {
    const o = p as Record<string, unknown>
    // 仅支持单值 AWS 或 *；多 Principal（数组）不支持。
    const aws = o.AWS
    if (typeof aws === 'string') return stripAWSPrefix(aws)
    if (aws === '*') return '*'
  }
  return null
}

function stripAWSPrefix(s: string): string {
  // 输入可能为 "arn:..." 或 "AWS:arn:..."；去掉冗余前缀便于编辑。
  return s.startsWith('AWS:') ? s.slice(4) : s
}

function principalToJSON(p: string): Record<string, string> | string {
  return p === '*' ? '*' : { AWS: p }
}

function normalizeStringArray(v: unknown): string[] | null {
  if (v == null) return null
  if (typeof v === 'string') return [v]
  if (Array.isArray(v)) {
    if (!v.every((x) => typeof x === 'string')) return null
    return v as string[]
  }
  return null
}

/** 常用桶策略模板（控制台习惯：单键应用）。 */
export interface PolicyTemplate {
  id: string
  label: string
  build: (bucket: string) => PolicyDoc
}

export const POLICY_TEMPLATES: PolicyTemplate[] = [
  {
    id: 'public-read',
    label: '公共读（GetObject）',
    build: (bucket) => ({
      Version: '2012-10-17',
      Statement: [
        {
          sid: 'AllowPublicRead',
          effect: 'Allow',
          principal: '*',
          actions: ['s3:GetObject'],
          resources: [`arn:aws:s3:::${bucket}/*`],
        },
      ],
    }),
  },
  {
    id: 'public-read-write',
    label: '公共读写',
    build: (bucket) => ({
      Version: '2012-10-17',
      Statement: [
        {
          sid: 'AllowPublicReadWrite',
          effect: 'Allow',
          principal: '*',
          actions: ['s3:GetObject', 's3:PutObject', 's3:DeleteObject'],
          resources: [`arn:aws:s3:::${bucket}/*`],
        },
      ],
    }),
  },
  {
    id: 'deny-list',
    label: '拒绝 List（仅写允许）',
    build: (bucket) => ({
      Version: '2012-10-17',
      Statement: [
        {
          sid: 'DenyListBucket',
          effect: 'Deny',
          principal: '*',
          actions: ['s3:ListBucket'],
          resources: [`arn:aws:s3:::${bucket}`],
        },
      ],
    }),
  },
  {
    id: 'clear',
    label: '清空策略',
    build: () => ({ Version: '2012-10-17', Statement: [] }),
  },
]

/** 校验策略 doc 本身是否合法（前端提前拦下无效 JSON）。 */
export function validateDoc(doc: PolicyDoc): string | null {
  if (doc.Version !== '2012-10-17') return 'Version 必须为 "2012-10-17"'
  if (!Array.isArray(doc.Statement)) return 'Statement 必须为数组'
  const seen = new Set<string>()
  for (const [i, s] of doc.Statement.entries()) {
    if (s.effect !== 'Allow' && s.effect !== 'Deny') return `第 ${i + 1} 条 Effect 非法`
    if (!s.principal) return `第 ${i + 1} 条 Principal 不能为空（用 * 表示所有人）`
    if (!s.actions.length) return `第 ${i + 1} 条 Action 不能为空`
    if (!s.resources.length) return `第 ${i + 1} 条 Resource 不能为空`
    if (s.sid) {
      if (seen.has(s.sid)) return `Sid "${s.sid}" 重复（S3 桶策略 Sid 必须唯一）`
      seen.add(s.sid)
    }
  }
  return null
}
