import type { components } from './api/schema'

// ---------------------------------------------------------------------------
// 与后端共享的实体类型：**直接派生自 spec**（`pnpm gen:api` 生成的 schema.d.ts），
// 不再手写字段清单（ROADMAP §三 #10，schema-first）。
//
// 为什么可行：这 4 个是 `components.schemas` 里的共享 schema，后端有三道门禁逐字段钉住
//   - `TestOpenAPI_ResponseSchemasMatchDTOs` —— schema 字段集 ⇄ 真实 DTO 字段集双向；
//   - `TestOpenAPI_AccountSchemaIsAccountView` —— Account 逐字段 + 明确不含 secretKey；
//   - `TestCommittedOpenAPISpecMatchesRuntime` —— docs/api/openapi.json ⇄ 运行时规范。
// 于是「后端改字段 → spec 变 → 生成物变 → 这里变 → `vue-tsc` 红」，漂移不再靠人工发现。
// ---------------------------------------------------------------------------

/** 后端返回的账号视图：不回传 secretKey，secretSet 表示是否已设置密钥。 */
export type Account = components['schemas']['Account']

/** 新建 / 编辑账号的提交载荷：secretKey 仅用于输入（编辑时留空表示保持不变）。 */
export interface AccountInput {
  name: string
  endpoint: string
  publicEndpoint?: string
  region: string
  accessKey: string
  secretKey: string
  bucket: string
  pathStyle: boolean
  useSSL: boolean
}

export type ObjectItem = components['schemas']['ObjectItem']

export type ListObjectsResponse = components['schemas']['ListObjectsResp']

export interface PresignResponse {
  method: 'get' | 'put' | 'post'
  bucket: string
  key: string
  url: string
  fields?: Record<string, string>
  expiresIn: number
}

export type BucketItem = components['schemas']['Bucket']

export interface ObjectMeta {
  key: string
  size: number
  lastModified: string
  etag: string
  contentType: string
  storageClass?: string
  metadata?: Record<string, string>
}

/** 桶属性：区域 / 创建时间 / 版本控制状态。 */
export interface BucketInfo {
  bucket: string
  region: string
  createdAt: string
  versioning: '' | 'Enabled' | 'Suspended'
}

/** 单个对象版本（ListObjectVersions）。 */
export interface ObjectVersion {
  key: string
  versionId: string
  isLatest: boolean
  lastModified: string
  size: number
  etag: string
  storageClass?: string
}

export interface ListVersionsResponse {
  versions: ObjectVersion[]
  deleteMarkers: { key: string; versionId: string; isLatest: boolean; lastModified: string }[]
  isTruncated: boolean
  nextKeyMarker: string
  nextVersionIdMarker: string
}

/** 生命周期过期规则（简化版：前缀 + 天数）。 */
export interface LifecycleRule {
  id: string
  prefix: string
  days: number
}

export interface MigrationResult {
  migrated: number
  failed: number
  lastError?: string
  failedKeys?: string[]
}

/** 异步任务状态：interrupted = 进程重启导致中断，需人工对账（如移动任务源未删）。 */
export type JobStatus = 'running' | 'done' | 'cancelled' | 'interrupted'

/** 服务端持久化的异步任务记录（GET /api/migrate/jobs）。 */
export interface JobRecord {
  id: string
  created: string
  total: number
  status: JobStatus
  // 内联而非引用 api.ts 的 MigrateProgress：api.ts 已 import types.ts，
  // 反向引用会形成模块循环。
  progress: { done: number; total: number; migrated: number; failed: number; key?: string; error?: string; status?: string }
  result: MigrationResult
}

/** 桶 CORS 规则。 */
export interface CorsRule {
  id?: string
  allowedMethods: string[]
  allowedOrigins: string[]
  allowedHeaders?: string[]
  exposeHeaders?: string[]
  maxAgeSeconds?: number
}

export type SortKey = 'name' | 'size' | 'time'

export interface Entry {
  kind: 'folder' | 'file'
  key: string
  name: string
  size?: number
  lastModified?: string
  object?: ObjectItem
}

/** 前端本地保存的后端连接配置（多服务端可选） */
export interface ServerProfile {
  id: string
  name: string
  base: string
  token: string
}

