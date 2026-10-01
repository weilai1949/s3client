import type {
  Account,
  AccountInput,
  BucketInfo,
  BucketItem,
  CorsRule,
  JobRecord,
  LifecycleRule,
  ListObjectsResponse,
  ListVersionsResponse,
  ObjectMeta,
  PresignResponse,
} from '../types'
import { opPath, request } from './http'
import { operations } from './operations'
import { downloadZipToDisk } from './download'
import { migrateJobStatus } from './jobs'

/**
 * 领域端点封装（从 `api.ts` 拆出，行为不变）：每个方法 = 一个后端 `/api/*` 端点。
 * 只做 URL/查询串拼装与类型标注，不含缓存、重试或状态。
 *
 * **2026-10-01 起（ROADMAP §三 #10）**：路径与 HTTP 方法一律取自 `operations`
 * （由 `pnpm gen:api` 从 `docs/api/openapi.json` 生成），**不再手写 URL 字面量**
 * 与 `method: 'POST'` 字面量——spec 改路径 / 换方法时生成物同步变，前端跟着变；
 * spec 删操作则 `OperationId` 收窄、`vue-tsc` 直接红。查询串仍是前端自己拼
 * （query 名不在 path 模板里，由 `docs/api/openapi.json` 的 parameters 契约与
 * 后端 `openapi_query_params_test.go` 守）。
 */
export const s3api = {
  listAccounts: () => request<{ accounts: Account[] }>(opPath('listAccounts')),
  createAccount: (a: AccountInput) =>
    request<Account>(opPath('createAccount'), { method: operations.createAccount.method, body: JSON.stringify(a) }),
  updateAccount: (id: string, a: Partial<AccountInput>) =>
    request<Account>(opPath('updateAccount', { id }), { method: operations.updateAccount.method, body: JSON.stringify(a) }),
  deleteAccount: (id: string) =>
    request<{ deleted: string }>(opPath('deleteAccount', { id }), { method: operations.deleteAccount.method }),
  testAccount: (id: string) =>
    request<{ ok: boolean; bucket: string; error?: string }>(opPath('testAccount', { id }), { method: operations.testAccount.method }),
  listBuckets: (id: string) => request<{ buckets: BucketItem[] }>(opPath('listBuckets', { id })),
  createBucket: (id: string, body: { name: string; region?: string; acl?: string }) =>
    request<{ created: string; region: string; acl: string }>(opPath('createBucket', { id }), { method: operations.createBucket.method, body: JSON.stringify(body) }),
  deleteBucket: (id: string, name: string) =>
    request<{ deleted: string }>(`${opPath('deleteBucket', { id })}?name=${encodeURIComponent(name)}`, { method: operations.deleteBucket.method }),
  previewBuckets: (a: AccountInput) =>
    request<{ buckets: BucketItem[] }>(opPath('previewBuckets'), { method: operations.previewBuckets.method, body: JSON.stringify(a) }),

  listObjects: (id: string, q: Record<string, string>, opts: { signal?: AbortSignal } = {}) => {
    const qs = new URLSearchParams(q).toString()
    return request<ListObjectsResponse>(`${opPath('listObjects', { id })}?${qs}`, { signal: opts.signal })
  },
  headObject: (id: string, q: Record<string, string>) => {
    const qs = new URLSearchParams(q).toString()
    return request<ObjectMeta>(`${opPath('headObject', { id })}?${qs}`)
  },
  mkdirObject: (id: string, body: { bucket?: string; key: string }) =>
    request<{ created: string; bucket: string }>(opPath('mkdirObject', { id }), { method: operations.mkdirObject.method, body: JSON.stringify(body) }),
  renameObject: (id: string, body: { bucket?: string; key: string; newKey: string; newBucket?: string }) =>
    request<{ renamed: string }>(opPath('renameObject', { id }), { method: operations.renameObject.method, body: JSON.stringify(body) }),
  copyObject: (id: string, body: { bucket?: string; key: string; newKey: string; newBucket?: string }) =>
    request<{ copied: string; bucket: string }>(opPath('copyObject', { id }), { method: operations.copyObject.method, body: JSON.stringify(body) }),
  /** 异步批量复制/移动：立即返回 jobId，进度走 migrate jobs SSE；`deleteSource` 由服务端在任务内删源。 */
  copyFilesAsync: (id: string, body: { bucket?: string; targetBucket?: string; targetPrefix?: string; keys: string[]; deleteSource?: boolean }) =>
    request<{ jobId: string; total: number }>(opPath('copyObjectsAsync', { id }), { method: operations.copyObjectsAsync.method, body: JSON.stringify(body) }),
  getObjectAcl: (id: string, query: { bucket?: string; key: string }) => {
    const qs = new URLSearchParams()
    qs.set('key', query.key)
    if (query.bucket) qs.set('bucket', query.bucket)
    return request<{
      bucket: string
      key: string
      owner?: string
      public: boolean
      grants: { grantee: string; permission: string }[]
      url: string
    }>(`${opPath('getObjectAcl', { id })}?${qs.toString()}`)
  },
  putObjectAcl: (id: string, body: { bucket?: string; key: string; acl: string }) =>
    request<{ acl: string }>(opPath('putObjectAcl', { id }), { method: operations.putObjectAcl.method, body: JSON.stringify(body) }),
  getObjectTags: (id: string, query: { bucket?: string; key: string }) => {
    const qs = new URLSearchParams()
    qs.set('key', query.key)
    if (query.bucket) qs.set('bucket', query.bucket)
    return request<{ tags: { key: string; value: string }[] }>(`${opPath('getObjectTags', { id })}?${qs.toString()}`)
  },
  putObjectTags: (id: string, body: { bucket?: string; key: string; tags: { key: string; value: string }[] }) =>
    request<{ tags: { key: string; value: string }[] }>(opPath('putObjectTags', { id }), { method: operations.putObjectTags.method, body: JSON.stringify(body) }),
  getBucketInfo: (id: string, bucket?: string) =>
    request<BucketInfo>(`${opPath('getBucketInfo', { id })}?bucket=${encodeURIComponent(bucket ?? '')}`),
  putBucketVersioning: (id: string, body: { bucket?: string; status: 'Enabled' | 'Suspended' }) =>
    request<{ versioning: string }>(opPath('putBucketVersioning', { id }), { method: operations.putBucketVersioning.method, body: JSON.stringify(body) }),
  getBucketEncryption: (id: string, bucket?: string) =>
    request<{ bucket: string; configured: boolean; algorithm: string; kmsKeyId: string; bucketKeyEnabled: boolean }>(`${opPath('getBucketEncryption', { id })}?bucket=${encodeURIComponent(bucket ?? '')}`),
  putBucketEncryption: (id: string, body: { bucket?: string; algorithm: string; kmsKeyId?: string; bucketKeyEnabled?: boolean }) =>
    request<{ configured: boolean; algorithm: string }>(opPath('putBucketEncryption', { id }), { method: operations.putBucketEncryption.method, body: JSON.stringify(body) }),
  deleteBucketEncryption: (id: string, bucket?: string) =>
    request<{ deleted: string }>(`${opPath('deleteBucketEncryption', { id })}?bucket=${encodeURIComponent(bucket ?? '')}`, { method: operations.deleteBucketEncryption.method }),
  getBucketCors: (id: string, bucket?: string) =>
    request<{ bucket: string; rules: CorsRule[] }>(`${opPath('getBucketCors', { id })}?bucket=${encodeURIComponent(bucket ?? '')}`),
  putBucketCors: (id: string, body: { bucket?: string; rules: CorsRule[] }) =>
    request<{ updated: number; deleted?: string }>(opPath('putBucketCors', { id }), { method: operations.putBucketCors.method, body: JSON.stringify(body) }),
  deleteBucketCors: (id: string, bucket?: string) =>
    request<{ deleted: string }>(`${opPath('deleteBucketCors', { id })}?bucket=${encodeURIComponent(bucket ?? '')}`, { method: operations.deleteBucketCors.method }),
  getBucketWebsite: (id: string, bucket?: string) =>
    request<{ bucket: string; configured: boolean; indexDocument: string; errorDocument: string; redirectAllRequestsTo: string }>(`${opPath('getBucketWebsite', { id })}?bucket=${encodeURIComponent(bucket ?? '')}`),
  putBucketWebsite: (id: string, body: { bucket?: string; indexDocument?: string; errorDocument?: string; redirectAllRequestsTo?: string }) =>
    request<{ configured: boolean }>(opPath('putBucketWebsite', { id }), { method: operations.putBucketWebsite.method, body: JSON.stringify(body) }),
  deleteBucketWebsite: (id: string, bucket?: string) =>
    request<{ deleted: string }>(`${opPath('deleteBucketWebsite', { id })}?bucket=${encodeURIComponent(bucket ?? '')}`, { method: operations.deleteBucketWebsite.method }),
  getBucketPolicy: (id: string, bucket?: string) =>
    request<{ bucket: string; configured: boolean; policy: string }>(`${opPath('getBucketPolicy', { id })}?bucket=${encodeURIComponent(bucket ?? '')}`),
  putBucketPolicy: (id: string, body: { bucket?: string; policy: string }) =>
    request<{ configured: boolean; deleted?: string }>(opPath('putBucketPolicy', { id }), { method: operations.putBucketPolicy.method, body: JSON.stringify(body) }),
  deleteBucketPolicy: (id: string, bucket?: string) =>
    request<{ deleted: string }>(`${opPath('deleteBucketPolicy', { id })}?bucket=${encodeURIComponent(bucket ?? '')}`, { method: operations.deleteBucketPolicy.method }),
  getBucketTags: (id: string, bucket?: string) =>
    request<{ bucket: string; tags: { key: string; value: string }[] }>(`${opPath('getBucketTags', { id })}?bucket=${encodeURIComponent(bucket ?? '')}`),
  putBucketTags: (id: string, body: { bucket?: string; tags: { key: string; value: string }[] }) =>
    request<{ updated: number; deleted?: string }>(opPath('putBucketTags', { id }), { method: operations.putBucketTags.method, body: JSON.stringify(body) }),
  deleteBucketTags: (id: string, bucket?: string) =>
    request<{ deleted: string }>(`${opPath('deleteBucketTags', { id })}?bucket=${encodeURIComponent(bucket ?? '')}`, { method: operations.deleteBucketTags.method }),
  listVersions: (id: string, q: { bucket?: string; prefix?: string; keyMarker?: string; versionIdMarker?: string }) => {
    const qs = new URLSearchParams()
    if (q.bucket) qs.set('bucket', q.bucket)
    if (q.prefix) qs.set('prefix', q.prefix)
    if (q.keyMarker) qs.set('keyMarker', q.keyMarker)
    if (q.versionIdMarker) qs.set('versionIdMarker', q.versionIdMarker)
    return request<ListVersionsResponse>(`${opPath('listObjectVersions', { id })}?${qs.toString()}`)
  },
  deleteObjectVersion: (id: string, q: { bucket?: string; key: string; versionId: string }) => {
    const qs = new URLSearchParams()
    qs.set('key', q.key)
    qs.set('versionId', q.versionId)
    if (q.bucket) qs.set('bucket', q.bucket)
    return request<{ deleted: string; versionId: string }>(`${opPath('deleteObjectVersion', { id })}?${qs.toString()}`, { method: operations.deleteObjectVersion.method })
  },
  restoreObjectVersion: (id: string, body: { bucket?: string; key: string; versionId: string }) =>
    request<{ restored: string; versionId: string }>(opPath('restoreObjectVersion', { id }), { method: operations.restoreObjectVersion.method, body: JSON.stringify(body) }),
  restoreDeleteMarker: (id: string, body: { bucket?: string; key: string; versionId: string }) =>
    request<{ restored: string; versionId: string }>(opPath('restoreDeleteMarker', { id }), { method: operations.restoreDeleteMarker.method, body: JSON.stringify(body) }),
  listTrash: (id: string, q: { bucket?: string; prefix?: string; keyMarker?: string; versionIdMarker?: string; maxKeys?: number }) => {
    const qs = new URLSearchParams()
    if (q.bucket) qs.set('bucket', q.bucket)
    if (q.prefix) qs.set('prefix', q.prefix)
    if (q.keyMarker) qs.set('keyMarker', q.keyMarker)
    if (q.versionIdMarker) qs.set('versionIdMarker', q.versionIdMarker)
    if (q.maxKeys) qs.set('maxKeys', String(q.maxKeys))
    return request<{ deleteMarkers: { key: string; versionId: string; isLatest: boolean; lastModified: string }[]; isTruncated: boolean; nextKeyMarker: string; nextVersionIdMarker: string }>(`${opPath('listTrash', { id })}?${qs.toString()}`)
  },
  purgeTrashObject: (id: string, body: { bucket?: string; key: string }) =>
    request<{ purged: string; deleted: number }>(opPath('purgeTrashObject', { id }), { method: operations.purgeTrashObject.method, body: JSON.stringify(body) }),
  changeStorageClass: (id: string, body: { bucket?: string; key: string; versionId?: string; storageClass: string }) =>
    request<{ changed: string; versionId: string; storageClass: string }>(opPath('changeStorageClass', { id }), { method: operations.changeStorageClass.method, body: JSON.stringify(body) }),
  setHeaders: (id: string, body: { bucket?: string; key: string; contentType?: string; metadata?: Record<string, string> }) =>
    request<{ updated: string }>(opPath('setHeaders', { id }), { method: operations.setHeaders.method, body: JSON.stringify(body) }),
  getLifecycle: (id: string, bucket?: string) =>
    request<{ rules: LifecycleRule[] }>(`${opPath('getLifecycle', { id })}?bucket=${encodeURIComponent(bucket ?? '')}`),
  putLifecycle: (id: string, body: { bucket?: string; rules: LifecycleRule[] }) =>
    request<{ updated: number }>(opPath('putLifecycle', { id }), { method: operations.putLifecycle.method, body: JSON.stringify(body) }),
  presign: (id: string, body: { method?: string; key: string; bucket?: string; versionId?: string; expiresIn?: number }) =>
    request<PresignResponse>(opPath('presign', { id }), { method: operations.presign.method, body: JSON.stringify(body) }),
  multipartInit: (id: string, body: { bucket?: string; key: string; contentType?: string }) =>
    request<{ uploadId: string; key: string; bucket: string }>(opPath('multipartInit', { id }), { method: operations.multipartInit.method, body: JSON.stringify(body) }),
  multipartPart: (id: string, body: { bucket?: string; key: string; uploadId: string; partNumber: number; expiresIn?: number }) =>
    request<{ partNumber: number; url: string; expiresIn: number }>(opPath('multipartPart', { id }), { method: operations.multipartPart.method, body: JSON.stringify(body) }),
  multipartComplete: (id: string, body: { bucket?: string; key: string; uploadId: string; parts: { partNumber: number; etag: string }[] }) =>
    request<{ completed: string }>(opPath('multipartComplete', { id }), { method: operations.multipartComplete.method, body: JSON.stringify(body) }),
  multipartAbort: (id: string, body: { bucket?: string; key: string; uploadId: string }) =>
    request<{ aborted: boolean }>(opPath('multipartAbort', { id }), { method: operations.multipartAbort.method, body: JSON.stringify(body) }),
  /**
   * 批量删除：响应为 `{deleted, failed, lastError?}`——`deleted` 只计真正删成功的 key
   * （S3 对逐 key 失败仍返回 200，被拒的计入 `failed`）。超过 1000 个 key 的请求会被
   * 服务端 400 拒绝，调用方必须按 `DELETE_MAX_KEYS_PER_REQUEST` 分片（见 src/limits.ts）。
   */
  deleteObjects: (id: string, body: { bucket?: string; keys: string[] }) =>
    request<{ deleted: number; failed?: number; lastError?: string }>(opPath('deleteObjects', { id }), { method: operations.deleteObjects.method, body: JSON.stringify(body) }),
  /** 异步前缀删除：立即返回 jobId，进度走 migrate jobs SSE。 */
  deletePrefixAsync: (id: string, body: { bucket?: string; prefix: string }) =>
    request<{ jobId: string; total: number; truncated?: boolean }>(opPath('deletePrefixAsync', { id }), { method: operations.deletePrefixAsync.method, body: JSON.stringify(body) }),
  copyPrefixAsync: (id: string, body: { bucket?: string; prefix: string; targetBucket?: string; targetPrefix: string }) =>
    request<{ jobId: string; total: number; truncated?: boolean }>(opPath('copyPrefixAsync', { id }), { method: operations.copyPrefixAsync.method, body: JSON.stringify(body) }),
  /** 流式 ZIP 落盘（优先 File System Access API）。 */
  downloadZipToDisk: (id: string, body: { bucket?: string; keys: string[] }, suggestedName?: string) =>
    downloadZipToDisk(id, body, suggestedName),

  migrateAsync: (body: {
    sourceAccountId: string
    sourceBucket?: string
    sourceKeys: string[]
    targetAccountId: string
    targetBucket?: string
    targetPrefix?: string
  }) =>
    request<{ jobId: string; total: number }>(opPath('migrateAsync'), { method: operations.migrateAsync.method, body: JSON.stringify(body) }),

  /** 异步任务清单（含进程重启后中断的任务，用于「未完成任务」对账视图）。 */
  migrateJobs: () => request<{ jobs: JobRecord[] }>(opPath('migrateJobs')),

  /** 异步任务状态（实现与 SSE 的 EOF 兜底轮询共用，见 `jobs.ts`）。 */
  migrateJobStatus,

  migrateJobCancel: (jobId: string) =>
    request<{ jobId: string; cancelled: boolean; done?: boolean }>(
      opPath('migrateJobCancel', { id: jobId }),
      { method: operations.migrateJobCancel.method },
    ),

}
