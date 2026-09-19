import type { ServerProfile } from '../types'
import { request } from './http'
import * as store from './storage'

/**
 * 前端 API 模块的公开面（barrel）。
 *
 * 2026-09-19 由单文件 `api.ts`（838 行）拆分为：
 *   - `storage.ts`   浏览器凭据/多服务器 profile 存储（依赖图最底层）
 *   - `http.ts`      传输层（base/Bearer/Content-Type + 错误归一）
 *   - `endpoints.ts` 领域端点封装（`s3api`）
 *   - `jobs.ts`      异步任务 SSE 订阅 + 状态回读兜底
 *   - `download.ts`  ZIP 流式落盘
 *   - `upload.ts`    预签名直传（XHR）
 *
 * 对外契约不变：`./api` / `../api` 仍解析到本文件（目录 `index.ts`），
 * 因此所有既有 import 路径无需改动。仅在此处组装跨模块的 `api` 对象。
 */

export { s3api } from './endpoints'
export { subscribeMigrateEvents, type MigrateProgress } from './jobs'
export { directUpload } from './upload'

/** 连接/凭据/多服务器配置等「客户端自身」能力（与领域端点 `s3api` 区分）。 */
export const api = {
  get base(): string {
    return store.getBase()
  },
  set base(v: string) {
    store.setBase(v)
  },
  get token(): string {
    return store.readToken()
  },
  set token(v: string) {
    store.writeToken(v)
  },
  /** Token 是否「跨会话保留」（开启后写 localStorage）。默认 false = 仅 sessionStorage。 */
  get isTokenPersistent(): boolean {
    return store.tokenPersistent()
  },
  /**
   * 切换「跨会话保留」：开启时把 token 同时落到 localStorage；
   * 关闭时从 localStorage 清除并保留在 sessionStorage。
   */
  setTokenPersistent(enabled: boolean) {
    store.setTokenPersistent(enabled)
  },
  get isTauri(): boolean {
    return store.isTauri()
  },

  /** 探测后端健康（/api/health，免鉴权）；用于不可用后的自动恢复轮询。 */
  health(): Promise<{ status: string; version: string; store?: { ok: boolean } }> {
    return request<{ status: string; version: string; store?: { ok: boolean } }>('/api/health')
  },

  listServers(): ServerProfile[] {
    return store.listServers()
  },

  activeServerId(): string {
    return store.activeServerId()
  },

  getActiveServer(): ServerProfile | undefined {
    return store.getActiveServer()
  },

  /** 设为当前生效并同步 base/token */
  selectServer(id: string): ServerProfile | undefined {
    return store.selectServer(id)
  },

  upsertServer(input: { id?: string; name: string; base: string; token: string }): ServerProfile {
    return store.upsertServer(input)
  },

  deleteServer(id: string): void {
    store.deleteServer(id)
  },
}
