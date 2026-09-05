/**
 * 浏览器 E2E 共用 fixture：极简 in-memory 后端 stub。
 *
 * 设计取舍：
 *   - 不引入 MSW / wiremock，避免依赖；直接用 Node http 暴露一组 /api/* 桩路由。
 *   - 不模拟完整 SDK 行为（签名 / 分段上传）；只覆盖浏览器侧能验证的入口
 *     （账号 CRUD、列表对象、上传预签名 → 实际写入内存）。
 *   - 跑测试时 vite preview 在 4173，fixture 在 4180；前端通过相对路径访问后端，
 *     由 baseURL + 同源策略自动代理或直连。本测试不依赖网络拓扑，使用
 *     page.route() 直接拦截请求并回放 fixture 响应（更稳）。
 */
import { test as base, type APIRequestContext } from '@playwright/test'

export interface FakeAccount {
  id: string
  name: string
  endpoint: string
  accessKey: string
  secretKey: string
  region: string
  bucket?: string
  pathStyle?: boolean
}

export interface FakeObject {
  key: string
  size: number
  lastModified: string
  etag: string
  contentType: string
  storageClass: string
  isDir: boolean
}

export interface BackendStub {
  accounts: FakeAccount[]
  objects: Map<string, FakeObject[]>
  presignedPUTs: string[]
  presignedGETs: string[]
}

export function emptyStub(): BackendStub {
  return { accounts: [], objects: new Map(), presignedPUTs: [], presignedGETs: [] }
}

export const test = base.extend<{ stub: BackendStub }>({
  stub: async ({ page: _ }, use) => {
    const stub = emptyStub()
    await use(stub)
  },
})

export { expect } from '@playwright/test'
export type { APIRequestContext }
