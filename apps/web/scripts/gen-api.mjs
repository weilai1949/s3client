#!/usr/bin/env node
/**
 * 从黄金契约 `docs/api/openapi.json` 生成前端代码（ROADMAP §三 #10，schema-first）。
 *
 * 产物两份，都必须**提交进仓库**、由 `src/api/generated.gate.test.ts` 钉新鲜度：
 *   1. `src/api/schema.d.ts`   —— openapi-typescript 生成的 `paths` / `components` / `operations` 类型；
 *   2. `src/api/operations.ts` —— 本脚本生成的**运行时零依赖**数据表
 *                                 （operationId → { method, path, params }），供 `endpoints.ts` 拼 URL。
 *
 * 为什么需要它：后端已有 7+ 道契约测试钉 `handler 代码 ⇄ OpenAPI`，但**前端 `types.ts` /
 * `endpoints.ts` 与 spec 之间此前没有任何测试**——路径改名、方法换向都不会让前端红。
 * 把 method/path 从 spec 生成出来之后，这类漂移**结构上不可能发生**（spec 变，前端跟着变；
 * spec 删操作，`vue-tsc` 直接红）。
 *
 * 用法：
 *   pnpm gen:api            # 写入两份产物
 *   pnpm gen:api --check    # 只校验产物是否与 spec 一致（不写），不一致 exit 1
 *
 * 生成器均为 devDependency（ADR-004：`dependencies` 只允许 `vue`），产物是纯 TS、不引第三方运行时。
 */
import { readFileSync, writeFileSync, mkdirSync } from 'node:fs'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import openapiTS, { astToString } from 'openapi-typescript'

const here = dirname(fileURLToPath(import.meta.url))
const webRoot = resolve(here, '..')
const repoRoot = resolve(webRoot, '..', '..')
const specRel = 'docs/api/openapi.json'
const specPath = join(repoRoot, specRel)
const schemaOut = join(webRoot, 'src', 'api', 'schema.d.ts')
const opsOut = join(webRoot, 'src', 'api', 'operations.ts')
const check = process.argv.includes('--check')

/** 文件头——**不含时间戳**：生成物必须字节稳定，否则 diff 门禁永远红。 */
const banner = (how) =>
  `// AUTO-GENERATED —— 不要手改。用 \`pnpm gen:api\` 重新生成。\n` +
  `// 源：${specRel}（${how}）\n` +
  `// 门禁：src/api/generated.gate.test.ts（产物过期即红灯）。\n\n`

// ---- 1) schema.d.ts（openapi-typescript） ----
const specUrl = new URL(`file://${specPath}`)
const schemaBody = astToString(await openapiTS(specUrl))
const schemaText = banner('openapi-typescript 生成类型') + schemaBody

// ---- 2) operations.ts（本脚本：operationId → method / path / 路径参数） ----
const HTTP_METHODS = ['get', 'put', 'post', 'delete', 'patch', 'head', 'options', 'trace']
const ops = []
for (const [path, item] of Object.entries(JSON.parse(readFileSync(specPath, 'utf8')).paths ?? {})) {
  for (const method of HTTP_METHODS) {
    const op = item?.[method]
    if (!op || typeof op !== 'object') continue
    if (!op.operationId) {
      process.stderr.write(`spec 缺 operationId：${method.toUpperCase()} ${path}\n`)
      process.exit(1)
    }
    ops.push({
      id: op.operationId,
      method: method.toUpperCase(),
      path,
      params: Array.from(path.matchAll(/\{([^}]+)\}/g), (m) => m[1]),
    })
  }
}
ops.sort((a, b) => (a.id < b.id ? -1 : a.id > b.id ? 1 : 0))
if (new Set(ops.map((o) => o.id)).size !== ops.length) {
  process.stderr.write('operationId 有重复，无法生成唯一键\n')
  process.exit(1)
}

const rows = ops.map(
  (o) =>
    `  ${o.id}: { method: ${JSON.stringify(o.method)}, path: ${JSON.stringify(o.path)}, params: [${o.params
      .map((p) => JSON.stringify(p))
      .join(', ')}] },`,
)
const opsText =
  banner('operationId → method / path / 路径参数') +
  `/** 单个操作的静态契约：method 与 path 逐字来自 spec，禁止在前端手写 URL 模板。 */\n` +
  `export interface Operation {\n` +
  `  readonly method: 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH'\n` +
  `  readonly path: string\n` +
  `  /** path 模板里 \`{占位符}\` 的名字，顺序即模板出现顺序。 */\n` +
  `  readonly params: readonly string[]\n}\n\n` +
  `export const operations = {\n${rows.join('\n')}\n} as const\n\n` +
  `/** spec 里全部操作的标识（${ops.length} 个，见 ${specRel}）。 */\n` +
  `export type OperationId = keyof typeof operations\n`

// ---- 3) 写入 / 校验 ----
let ok = true
for (const [path, text] of [
  [schemaOut, schemaText],
  [opsOut, opsText],
]) {
  const rel = path.slice(webRoot.length + 1)
  if (check) {
    let cur = ''
    try {
      cur = readFileSync(path, 'utf8')
    } catch {
      /* 缺文件按不一致处理 */
    }
    if (cur !== text) {
      process.stderr.write(`${rel} 与 spec 不一致 —— 跑 \`pnpm gen:api\` 重新生成\n`)
      ok = false
    }
  } else {
    mkdirSync(dirname(path), { recursive: true })
    writeFileSync(path, text)
  }
}
if (!ok) process.exit(1)
if (!check) process.stdout.write(`gen:api → ${ops.length} 个操作（schema.d.ts + operations.ts）\n`)
