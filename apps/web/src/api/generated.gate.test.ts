import { readdirSync, readFileSync } from 'node:fs'
import { join } from 'node:path'
import { spawnSync } from 'node:child_process'
import { describe, expect, it } from 'vitest'
import { opPath } from './http'
import { operations, type OperationId } from './operations'

/**
 * 生成物门禁（ROADMAP §三 #10，schema-first）。
 *
 * 两份产物由 `pnpm gen:api` 从黄金契约 `docs/api/openapi.json` 生成并**提交进仓库**：
 *   - `src/api/schema.d.ts`   —— openapi-typescript 类型（`types.ts` 从它派生 4 个共享实体）
 *   - `src/api/operations.ts` —— operationId → method / path / 路径参数（`endpoints.ts` 只用它拼 URL）
 *
 * 本文件钉两件事：
 *   1. **新鲜度**——spec 改了而忘了重新生成即红灯（这是「diff 门禁」的那半边）；
 *   2. **生成物自身的结构不变量**——防空跑：条数塌缩、占位符与 params 对不上都要红。
 *
 * 说明：`--check` 直接跑仓库自己的生成脚本，因此门禁与生成器**不可能口径分叉**
 * （不会出现「门禁用另一套解析器、生成物却过不了」）。
 */
const genApi = () =>
  spawnSync(process.execPath, ['scripts/gen-api.mjs', '--check'], {
    cwd: process.cwd(),
    encoding: 'utf8',
  })

/**
 * 前端**有意不消费**的 spec 操作（KNOWN_ISSUES #75）。分三类：
 *   - 同步变体已由 async 版取代：`copyObjects` / `copyPrefix` / `deletePrefix` / `migrate` / `migrateSync`；
 *   - 账号详情由列表 + 编辑态复用：`getAccount`；
 *   - 供工具 / 运维而非 UI：`metrics` / `openapi`。
 * spec 增删操作时必须同步本清单或接入 s3api，否则下方穷尽性用例红灯（见 generated.gate.test.ts 头注释）。
 */
const unreferencedOperations = new Set([
  'copyObjects',
  'copyPrefix',
  'deletePrefix',
  'getAccount',
  'metrics',
  'migrate',
  'migrateSync',
  'openapi',
])

/** 递归收集 src 下的生产 TS 文件（排除 *.test.ts）。 */
function productionTsFiles(dir: string, out: string[] = []): string[] {
  for (const e of readdirSync(dir, { withFileTypes: true })) {
    const p = join(dir, e.name)
    if (e.isDirectory()) productionTsFiles(p, out)
    else if (e.name.endsWith('.ts') && !e.name.endsWith('.test.ts')) out.push(p)
  }
  return out
}

/** 生产源码里所有 `opPath('<opId>')` 引用的 operationId 集合。 */
function referencedOperationIds(): Set<string> {
  const out = new Set<string>()
  for (const file of productionTsFiles(join(process.cwd(), 'src'))) {
    for (const m of readFileSync(file, 'utf8').matchAll(/opPath\('([A-Za-z0-9_]+)'/g)) out.add(m[1])
  }
  return out
}

describe('gen:api 产物', () => {
  it('与 docs/api/openapi.json 逐字节一致（过期即红灯）', () => {
    const r = genApi()
    expect(
      r.status,
      `生成物已过期 —— 跑 \`pnpm gen:api\` 重新生成后连同 spec 一起提交。\n${r.stderr}`,
    ).toBe(0)
  })

  it('结构自检：条数不塌缩、path 归 /api/、占位符与 params 逐个对齐', () => {
    const ids = Object.keys(operations) as OperationId[]
    // spec 现有 70 个操作；低于 60 视为解析口径塌缩（防「空表永远绿」）。
    expect(ids.length, `只生成了 ${ids.length} 个操作`).toBeGreaterThanOrEqual(60)
    expect(new Set(ids).size).toBe(ids.length)

    for (const id of ids) {
      const op = operations[id]
      expect(op.path, `${id} 的 path 不是 /api/ 开头`).toMatch(/^\/api\//)
      expect(['GET', 'POST', 'PUT', 'DELETE', 'PATCH'], `${id} 的 method 非法`).toContain(op.method)
      const path: string = op.path
      const inPath = Array.from(path.matchAll(/\{([^}]+)\}/g), (m) => m[1])
      expect(inPath, `${id} 的占位符与 params 不一致`).toEqual([...op.params])
    }
  })

  it('opPath：无占位符的操作可省参数，有占位符的逐个 encodeURIComponent', () => {
    expect(opPath('listAccounts')).toBe('/api/accounts')
    expect(opPath('health')).toBe('/api/health')
    // 含 `/` 与空格的 id 必须被编码，否则会被当成路径分隔符
    expect(opPath('deleteAccount', { id: 'a/b c' })).toBe('/api/accounts/a%2Fb%20c')
    expect(opPath('migrateJobCancel', { id: 'job-1' })).toBe('/api/migrate/jobs/job-1/cancel')
  })

  it('穷尽性：source 引用的 opId 必须存在，operations 中未被引用的必须正好是白名单', () => {
    // 双向穷尽（KNOWN_ISSUES #75）——补上原 `>= 60` 阈值无法覆盖的「删掉一个 spec
    // 路径再重新生成仍全绿」缺口：
    //   反向：生产源码 `opPath('X')` 引用的 X 必须仍存在于 operations（spec 删路径后
    //         的旧引用 = stale ref → 红灯）；
    //   正向：operations 中未被源码引用的 opId 必须**正好等于**白名单——spec 新增操作
    //         却忘了接前端、或白名单失真（引用了白名单里的 id / 漏登记）都会红。
    const referenced = referencedOperationIds()
    const ids = Object.keys(operations) as OperationId[]

    const stale = [...referenced].filter((id) => !(id in operations)).sort()
    expect(stale, `source 引用了 spec 中不存在的操作: ${stale.join(', ')}`).toEqual([])

    const unreferenced = ids.filter((id) => !referenced.has(id)).sort()
    expect(
      unreferenced,
      'operations 中未被前端消费的 opId 与白名单不一致——新增 spec 操作要么接 s3api，要么登记到 unreferencedOperations',
    ).toEqual([...unreferencedOperations].sort())
  })
})
