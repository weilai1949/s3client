import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

/**
 * P0-2 验收（docs/archive/review-2026-09-19.md §F1）：`s3c.servers` 被写坏时整站白屏。
 *
 * 与 App.test.ts 的区别：这里**不 mock `./api`**，用真实存储模块 + 真实 App 模板，
 * 复现「渲染期调用 `api.getActiveServer()` 抛异常 → 渲染中断 → 白屏」的完整路径。
 * 白屏的严重性在于唯一能修复存储的 Server 面板也随之不可达，用户只能手工清 localStorage。
 *
 * 验收：`[null]` / `[1]` / `[{"base":{}}]` 三种存储下 App 必须正常渲染。
 */

// happy-dom 默认不提供 localStorage；用 Map 替身补齐（与 api.test.ts 同一形状）。
class MemStorage implements Storage {
  private m = new Map<string, string>()
  get length(): number { return this.m.size }
  clear() { this.m.clear() }
  key(i: number): string | null { return [...this.m.keys()][i] ?? null }
  getItem(k: string): string | null { return this.m.get(k) ?? null }
  setItem(k: string, v: string) { this.m.set(k, String(v)) }
  removeItem(k: string) { this.m.delete(k) }
}

let memLocal: MemStorage

beforeEach(() => {
  memLocal = new MemStorage()
  Object.defineProperty(globalThis, 'localStorage', { value: memLocal, configurable: true, writable: true })
  Object.defineProperty(globalThis, 'sessionStorage', { value: new MemStorage(), configurable: true, writable: true })
  vi.resetModules()
  // App 挂载时会拉账号列表；这里只关心渲染期不再抛异常，返回空清单即可。
  vi.stubGlobal('fetch', vi.fn(async () => ({
    ok: true,
    status: 200,
    statusText: 'OK',
    json: async () => ({ accounts: [] }),
    headers: new Map<string, string>(),
    body: null,
  })))
})

const childStubs = {
  AccountsPanel: true,
  ObjectsPanel: true,
  UploadPanel: true,
  MigratePanel: true,
  BucketsPanel: true,
  RecycleBinPanel: true,
  ServerPanel: true,
  ConfirmDialog: true,
  PromptDialog: true,
  Toasts: true,
}

async function mountAppWithCorruptServers(raw: string) {
  memLocal.setItem('s3c.servers', raw)
  const { default: App } = await import('./App.vue')
  return mount(App, { global: { stubs: childStubs } })
}

describe('App 存储被写坏（P0-2）', () => {
  for (const raw of ['[null]', '[1]', '[{"base":{}}]', '["x"]', '[{}]']) {
    // 显式 timeout：本文件的每个用例都 `vi.resetModules()` 后**重新 import 整张 App 模块图**
    // （这正是它能复现「渲染期读坏存储」的原因），是全仓最重的单测文件。
    // 默认 5s 在「覆盖率插桩（istanbul）+ 全量并行」下会被击穿：2026-09-29 实测该文件单独跑
    // 1.4s、加覆盖率全量跑 2.6s，但**首个用例**（承担首次动态 import 成本）并发时超过 5s → 假红。
    // 放宽到 20s 是留 4× 余量；这是**资源竞争**而非死循环——先验证过：同一次全量跑加
    // `--testTimeout=20000` 即全绿（74 文件 / 1132 例）。
    // 不要用增大全局 testTimeout 来掩盖：其它文件不需要这么长的窗口。
    it(`s3c.servers = ${raw} 时正常渲染（不白屏）`, { timeout: 20_000 }, async () => {
      const wrapper = await mountAppWithCorruptServers(raw)
      // 顶栏存在 = 渲染未被中断（白屏时连 header 都挂不上）。
      expect(wrapper.find('header').exists()).toBe(true)
      expect(wrapper.findAll('nav button').length).toBeGreaterThan(0)
      // 侧边栏的 Server 入口必须仍可达 —— 它是用户自救的唯一入口。
      expect(wrapper.text()).toMatch(/服务器设置|Server/)
      wrapper.unmount()
    })
  }
})
