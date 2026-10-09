import type { components, operations } from './api/schema'

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

// FinOps 成本看板（ROADMAP §三 #7）：响应类型直接派生自 spec 的 operation 200 响应，
// 漂移由 `pnpm gen:api` + `vue-tsc` 拦（后端另有响应契约门禁钉注册表）。
export type StorageReport = operations['storageReport']['responses'][200]['content']['application/json']
export type StorageClassUsage = StorageReport['byStorageClass'][number]
export type PrefixUsage = StorageReport['byPrefix'][number]
export type StorageRecommendation = StorageReport['recommendations'][number]

export interface PresignResponse {
  method: 'get' | 'put' | 'post'
  bucket: string
  key: string
  url: string
  fields?: Record<string, string>
  expiresIn: number
  /**
   * 条件写（`ifMatch` / `ifNoneMatch`）时后端参与签名的请求头——**直传时必须原样带上**，
   * 否则 S3 按「签名不匹配」拒绝；无条件预签名时后端返回 `{}`。
   */
  headers?: Record<string, string>
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
  /** 服务端校验和；无校验和或厂商不支持时为 null / 缺省。 */
  checksums?: ObjectChecksums | null
}

/** 桶属性：区域 / 创建时间 / 版本控制状态。 */
export interface BucketInfo {
  bucket: string
  region: string
  createdAt: string
  versioning: '' | 'Enabled' | 'Suspended'
}

// ---------------------------------------------------------------------------
// S3 新协议特性（ROADMAP §三 #5）：Object Lock / 对象保留期 / 法定保留 / 校验和。
// 字段与 `docs/api/openapi.json` 对应操作的响应一一对应。
// ---------------------------------------------------------------------------

/** Object Lock / 对象保留期共用的保留模式。 */
export type RetentionMode = 'GOVERNANCE' | 'COMPLIANCE'

/** 服务端存储的校验和（HeadObject）；无校验和或厂商不支持时为 null。 */
export interface ObjectChecksums {
  crc64nvme?: string
  crc32c?: string
  sha256?: string
  sha1?: string
  /** FULL_OBJECT | COMPOSITE_*（分段合成，不可全对象比对）。 */
  type?: string
}

/** 桶级 Object Lock 配置（GET/PUT `/bucket/object-lock`）。 */
export interface ObjectLockConfig {
  bucket: string
  enabled: boolean
  /** 未配置默认保留时为空串。 */
  defaultRetentionMode: '' | RetentionMode
  defaultRetentionDays: number
  defaultRetentionYears: number
}

/** 单个对象版本的保留期（GET/PUT `/object-retention`）。 */
export interface ObjectRetention {
  bucket: string
  key: string
  versionId: string
  configured: boolean
  /** `configured=false` 时为空串。 */
  mode: '' | RetentionMode
  /** RFC3339 到期时间；`configured=false` 时为空串。 */
  retainUntilDate: string
}

/** 单个对象版本的法定保留状态（GET/PUT `/object-legal-hold`）。 */
export interface ObjectLegalHold {
  bucket: string
  key: string
  versionId: string
  status: 'ON' | 'OFF'
}

/**
 * `verifyChecksum` 可用的校验来源；`'none'` 表示无可验证来源——
 * 属如实降级（不是错误），UI 需据此显示「无可验证来源」。
 */
export type VerifyMethod = 'crc64nvme' | 'crc32c' | 'sha256' | 'sha1' | 'etag-md5' | 'none'

/** 服务端校验结果（POST `/verify-checksum`）。 */
export interface VerifyResult {
  bucket: string
  key: string
  versionId: string
  method: VerifyMethod
  local: string
  remote: string
  match: boolean
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

/**
 * 计划任务（POST/PUT/GET `/api/schedules`）：cron 定时增量同步（ROADMAP #6）。
 * `lastRunAt`/`lastJobId`/`lastError` 为运行态：从未运行时服务端省略。
 */
export interface Schedule {
  id: string
  sourceAccountId: string
  sourceBucket: string
  sourcePrefix?: string
  targetAccountId: string
  targetBucket: string
  targetPrefix?: string
  mode: 'etag' | 'size_mtime' | 'always'
  cron: string
  enabled: boolean
  createdAt: string
  nextRunAt: string
  lastRunAt?: string
  lastJobId?: string
  lastError?: string
}

/** 创建 / 更新计划的请求体（与后端 `scheduleRequest` DTO 同字段）。 */
export interface ScheduleInput {
  sourceAccountId: string
  sourceBucket: string
  sourcePrefix?: string
  targetAccountId: string
  targetBucket: string
  targetPrefix?: string
  mode: 'etag' | 'size_mtime' | 'always'
  cron: string
  enabled?: boolean
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

