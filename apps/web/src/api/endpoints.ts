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
import { request } from './http'
import { downloadZipToDisk } from './download'
import { migrateJobStatus } from './jobs'

/**
 * 领域端点封装（从 `api.ts` 拆出，行为不变）：每个方法 = 一个后端 `/api/*` 端点。
 * 只做 URL/查询串拼装与类型标注，不含缓存、重试或状态。
 */
export const s3api = {
  listAccounts: () => request<{ accounts: Account[] }>('/api/accounts'),
  createAccount: (a: AccountInput) => request<Account>('/api/accounts', { method: 'POST', body: JSON.stringify(a) }),
  updateAccount: (id: string, a: Partial<AccountInput>) =>
    request<Account>(`/api/accounts/${id}`, { method: 'PUT', body: JSON.stringify(a) }),
  deleteAccount: (id: string) => request<{ deleted: string }>(`/api/accounts/${id}`, { method: 'DELETE' }),
  testAccount: (id: string) => request<{ ok: boolean; bucket: string; error?: string }>(`/api/accounts/${id}/test`, { method: 'POST' }),
  listBuckets: (id: string) => request<{ buckets: BucketItem[] }>(`/api/accounts/${id}/buckets`),
  createBucket: (id: string, body: { name: string; region?: string; acl?: string }) =>
    request<{ created: string; region: string; acl: string }>(`/api/accounts/${id}/bucket`, { method: 'POST', body: JSON.stringify(body) }),
  deleteBucket: (id: string, name: string) =>
    request<{ deleted: string }>(`/api/accounts/${id}/bucket?name=${encodeURIComponent(name)}`, { method: 'DELETE' }),
  previewBuckets: (a: AccountInput) =>
    request<{ buckets: BucketItem[] }>('/api/accounts/preview-buckets', { method: 'POST', body: JSON.stringify(a) }),

  listObjects: (id: string, q: Record<string, string>, opts: { signal?: AbortSignal } = {}) => {
    const qs = new URLSearchParams(q).toString()
    return request<ListObjectsResponse>(`/api/accounts/${id}/objects?${qs}`, { signal: opts.signal })
  },
  headObject: (id: string, q: Record<string, string>) => {
    const qs = new URLSearchParams(q).toString()
    return request<ObjectMeta>(`/api/accounts/${id}/head?${qs}`)
  },
  mkdirObject: (id: string, body: { bucket?: string; key: string }) =>
    request<{ created: string; bucket: string }>(`/api/accounts/${id}/mkdir`, { method: 'POST', body: JSON.stringify(body) }),
  renameObject: (id: string, body: { bucket?: string; key: string; newKey: string; newBucket?: string }) =>
    request<{ renamed: string }>(`/api/accounts/${id}/rename`, { method: 'POST', body: JSON.stringify(body) }),
  copyObject: (id: string, body: { bucket?: string; key: string; newKey: string; newBucket?: string }) =>
    request<{ copied: string; bucket: string }>(`/api/accounts/${id}/copy-object`, { method: 'POST', body: JSON.stringify(body) }),
  /** 异步批量复制/移动：立即返回 jobId，进度走 migrate jobs SSE；`deleteSource` 由服务端在任务内删源。 */
  copyFilesAsync: (id: string, body: { bucket?: string; targetBucket?: string; targetPrefix?: string; keys: string[]; deleteSource?: boolean }) =>
    request<{ jobId: string; total: number }>(`/api/accounts/${id}/copy-objects/async`, { method: 'POST', body: JSON.stringify(body) }),
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
    }>(`/api/accounts/${id}/object-acl?${qs.toString()}`)
  },
  putObjectAcl: (id: string, body: { bucket?: string; key: string; acl: string }) =>
    request<{ acl: string }>(`/api/accounts/${id}/object-acl`, { method: 'PUT', body: JSON.stringify(body) }),
  getObjectTags: (id: string, query: { bucket?: string; key: string }) => {
    const qs = new URLSearchParams()
    qs.set('key', query.key)
    if (query.bucket) qs.set('bucket', query.bucket)
    return request<{ tags: { key: string; value: string }[] }>(`/api/accounts/${id}/object-tags?${qs.toString()}`)
  },
  putObjectTags: (id: string, body: { bucket?: string; key: string; tags: { key: string; value: string }[] }) =>
    request<{ tags: { key: string; value: string }[] }>(`/api/accounts/${id}/object-tags`, { method: 'PUT', body: JSON.stringify(body) }),
  getBucketInfo: (id: string, bucket?: string) =>
    request<BucketInfo>(`/api/accounts/${id}/bucket-info?bucket=${encodeURIComponent(bucket ?? '')}`),
  putBucketVersioning: (id: string, body: { bucket?: string; status: 'Enabled' | 'Suspended' }) =>
    request<{ versioning: string }>(`/api/accounts/${id}/bucket-versioning`, { method: 'PUT', body: JSON.stringify(body) }),
  getBucketEncryption: (id: string, bucket?: string) =>
    request<{ bucket: string; configured: boolean; algorithm: string; kmsKeyId: string; bucketKeyEnabled: boolean }>(`/api/accounts/${id}/bucket/encryption?bucket=${encodeURIComponent(bucket ?? '')}`),
  putBucketEncryption: (id: string, body: { bucket?: string; algorithm: string; kmsKeyId?: string; bucketKeyEnabled?: boolean }) =>
    request<{ configured: boolean; algorithm: string }>(`/api/accounts/${id}/bucket/encryption`, { method: 'PUT', body: JSON.stringify(body) }),
  deleteBucketEncryption: (id: string, bucket?: string) =>
    request<{ deleted: string }>(`/api/accounts/${id}/bucket/encryption?bucket=${encodeURIComponent(bucket ?? '')}`, { method: 'DELETE' }),
  getBucketCors: (id: string, bucket?: string) =>
    request<{ bucket: string; rules: CorsRule[] }>(`/api/accounts/${id}/bucket/cors?bucket=${encodeURIComponent(bucket ?? '')}`),
  putBucketCors: (id: string, body: { bucket?: string; rules: CorsRule[] }) =>
    request<{ updated: number; deleted?: string }>(`/api/accounts/${id}/bucket/cors`, { method: 'PUT', body: JSON.stringify(body) }),
  deleteBucketCors: (id: string, bucket?: string) =>
    request<{ deleted: string }>(`/api/accounts/${id}/bucket/cors?bucket=${encodeURIComponent(bucket ?? '')}`, { method: 'DELETE' }),
  getBucketWebsite: (id: string, bucket?: string) =>
    request<{ bucket: string; configured: boolean; indexDocument: string; errorDocument: string; redirectAllRequestsTo: string }>(`/api/accounts/${id}/bucket/website?bucket=${encodeURIComponent(bucket ?? '')}`),
  putBucketWebsite: (id: string, body: { bucket?: string; indexDocument?: string; errorDocument?: string; redirectAllRequestsTo?: string }) =>
    request<{ configured: boolean }>(`/api/accounts/${id}/bucket/website`, { method: 'PUT', body: JSON.stringify(body) }),
  deleteBucketWebsite: (id: string, bucket?: string) =>
    request<{ deleted: string }>(`/api/accounts/${id}/bucket/website?bucket=${encodeURIComponent(bucket ?? '')}`, { method: 'DELETE' }),
  getBucketPolicy: (id: string, bucket?: string) =>
    request<{ bucket: string; configured: boolean; policy: string }>(`/api/accounts/${id}/bucket/policy?bucket=${encodeURIComponent(bucket ?? '')}`),
  putBucketPolicy: (id: string, body: { bucket?: string; policy: string }) =>
    request<{ configured: boolean; deleted?: string }>(`/api/accounts/${id}/bucket/policy`, { method: 'PUT', body: JSON.stringify(body) }),
  deleteBucketPolicy: (id: string, bucket?: string) =>
    request<{ deleted: string }>(`/api/accounts/${id}/bucket/policy?bucket=${encodeURIComponent(bucket ?? '')}`, { method: 'DELETE' }),
  getBucketTags: (id: string, bucket?: string) =>
    request<{ bucket: string; tags: { key: string; value: string }[] }>(`/api/accounts/${id}/bucket/tags?bucket=${encodeURIComponent(bucket ?? '')}`),
  putBucketTags: (id: string, body: { bucket?: string; tags: { key: string; value: string }[] }) =>
    request<{ updated: number; deleted?: string }>(`/api/accounts/${id}/bucket/tags`, { method: 'PUT', body: JSON.stringify(body) }),
  deleteBucketTags: (id: string, bucket?: string) =>
    request<{ deleted: string }>(`/api/accounts/${id}/bucket/tags?bucket=${encodeURIComponent(bucket ?? '')}`, { method: 'DELETE' }),
  listVersions: (id: string, q: { bucket?: string; prefix?: string; keyMarker?: string; versionIdMarker?: string }) => {
    const qs = new URLSearchParams()
    if (q.bucket) qs.set('bucket', q.bucket)
    if (q.prefix) qs.set('prefix', q.prefix)
    if (q.keyMarker) qs.set('keyMarker', q.keyMarker)
    if (q.versionIdMarker) qs.set('versionIdMarker', q.versionIdMarker)
    return request<ListVersionsResponse>(`/api/accounts/${id}/versions?${qs.toString()}`)
  },
  deleteObjectVersion: (id: string, q: { bucket?: string; key: string; versionId: string }) => {
    const qs = new URLSearchParams()
    qs.set('key', q.key)
    qs.set('versionId', q.versionId)
    if (q.bucket) qs.set('bucket', q.bucket)
    return request<{ deleted: string; versionId: string }>(`/api/accounts/${id}/version?${qs.toString()}`, { method: 'DELETE' })
  },
  restoreObjectVersion: (id: string, body: { bucket?: string; key: string; versionId: string }) =>
    request<{ restored: string; versionId: string }>(`/api/accounts/${id}/version/restore`, { method: 'POST', body: JSON.stringify(body) }),
  restoreDeleteMarker: (id: string, body: { bucket?: string; key: string; versionId: string }) =>
    request<{ restored: string; versionId: string }>(`/api/accounts/${id}/delete-marker/restore`, { method: 'POST', body: JSON.stringify(body) }),
  listTrash: (id: string, q: { bucket?: string; prefix?: string; keyMarker?: string; versionIdMarker?: string; maxKeys?: number }) => {
    const qs = new URLSearchParams()
    if (q.bucket) qs.set('bucket', q.bucket)
    if (q.prefix) qs.set('prefix', q.prefix)
    if (q.keyMarker) qs.set('keyMarker', q.keyMarker)
    if (q.versionIdMarker) qs.set('versionIdMarker', q.versionIdMarker)
    if (q.maxKeys) qs.set('maxKeys', String(q.maxKeys))
    return request<{ deleteMarkers: { key: string; versionId: string; isLatest: boolean; lastModified: string }[]; isTruncated: boolean; nextKeyMarker: string; nextVersionIdMarker: string }>(`/api/accounts/${id}/trash?${qs.toString()}`)
  },
  purgeTrashObject: (id: string, body: { bucket?: string; key: string }) =>
    request<{ purged: string; deleted: number }>(`/api/accounts/${id}/trash/purge`, { method: 'POST', body: JSON.stringify(body) }),
  changeStorageClass: (id: string, body: { bucket?: string; key: string; versionId?: string; storageClass: string }) =>
    request<{ changed: string; versionId: string; storageClass: string }>(`/api/accounts/${id}/storage-class`, { method: 'POST', body: JSON.stringify(body) }),
  setHeaders: (id: string, body: { bucket?: string; key: string; contentType?: string; metadata?: Record<string, string> }) =>
    request<{ updated: string }>(`/api/accounts/${id}/set-headers`, { method: 'POST', body: JSON.stringify(body) }),
  getLifecycle: (id: string, bucket?: string) =>
    request<{ rules: LifecycleRule[] }>(`/api/accounts/${id}/lifecycle?bucket=${encodeURIComponent(bucket ?? '')}`),
  putLifecycle: (id: string, body: { bucket?: string; rules: LifecycleRule[] }) =>
    request<{ updated: number }>(`/api/accounts/${id}/lifecycle`, { method: 'PUT', body: JSON.stringify(body) }),
  presign: (id: string, body: { method?: string; key: string; bucket?: string; versionId?: string; expiresIn?: number }) =>
    request<PresignResponse>(`/api/accounts/${id}/presign`, { method: 'POST', body: JSON.stringify(body) }),
  multipartInit: (id: string, body: { bucket?: string; key: string; contentType?: string }) =>
    request<{ uploadId: string; key: string; bucket: string }>(`/api/accounts/${id}/multipart/init`, { method: 'POST', body: JSON.stringify(body) }),
  multipartPart: (id: string, body: { bucket?: string; key: string; uploadId: string; partNumber: number; expiresIn?: number }) =>
    request<{ partNumber: number; url: string; expiresIn: number }>(`/api/accounts/${id}/multipart/part`, { method: 'POST', body: JSON.stringify(body) }),
  multipartComplete: (id: string, body: { bucket?: string; key: string; uploadId: string; parts: { partNumber: number; etag: string }[] }) =>
    request<{ completed: string }>(`/api/accounts/${id}/multipart/complete`, { method: 'POST', body: JSON.stringify(body) }),
  multipartAbort: (id: string, body: { bucket?: string; key: string; uploadId: string }) =>
    request<{ aborted: boolean }>(`/api/accounts/${id}/multipart/abort`, { method: 'POST', body: JSON.stringify(body) }),
  /**
   * 批量删除：响应为 `{deleted, failed, lastError?}`——`deleted` 只计真正删成功的 key
   * （S3 对逐 key 失败仍返回 200，被拒的计入 `failed`）。超过 1000 个 key 的请求会被
   * 服务端 400 拒绝，调用方必须按 `DELETE_MAX_KEYS_PER_REQUEST` 分片（见 src/limits.ts）。
   */
  deleteObjects: (id: string, body: { bucket?: string; keys: string[] }) =>
    request<{ deleted: number; failed?: number; lastError?: string }>(`/api/accounts/${id}/delete`, { method: 'POST', body: JSON.stringify(body) }),
  /** 异步前缀删除：立即返回 jobId，进度走 migrate jobs SSE。 */
  deletePrefixAsync: (id: string, body: { bucket?: string; prefix: string }) =>
    request<{ jobId: string; total: number; truncated?: boolean }>(`/api/accounts/${id}/delete-prefix/async`, { method: 'POST', body: JSON.stringify(body) }),
  copyPrefixAsync: (id: string, body: { bucket?: string; prefix: string; targetBucket?: string; targetPrefix: string }) =>
    request<{ jobId: string; total: number; truncated?: boolean }>(`/api/accounts/${id}/copy-prefix/async`, { method: 'POST', body: JSON.stringify(body) }),
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
    request<{ jobId: string; total: number }>('/api/migrate/async', { method: 'POST', body: JSON.stringify(body) }),

  /** 异步任务清单（含进程重启后中断的任务，用于「未完成任务」对账视图）。 */
  migrateJobs: () => request<{ jobs: JobRecord[] }>('/api/migrate/jobs'),

  /** 异步任务状态（实现与 SSE 的 EOF 兜底轮询共用，见 `jobs.ts`）。 */
  migrateJobStatus,

  migrateJobCancel: (jobId: string) =>
    request<{ jobId: string; cancelled: boolean; done?: boolean }>(
      `/api/migrate/jobs/${encodeURIComponent(jobId)}/cancel`,
      { method: 'POST' },
    ),

}
