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
})
