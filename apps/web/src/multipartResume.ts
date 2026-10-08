/**
 * 分段上传的断点续传记录（ROADMAP §三 #8）。
 *
 * 只持久化「定位 + 进度」：文件指纹（name/size/lastModified）、目标（账号 / 桶 / key）、
 * uploadId 与已完成分段的段号 + ETag。**绝不持久化 AccessKey / SecretKey / 任何凭证**——
 * 凭证只存在于既有凭据存储（见 `api/storage.ts`），不进入本模块。
 *
 * 服务端 ListParts 清单是**唯一真值**：这里读出的记录只是「候选会话」，
 * 复用前必须由 `s3api.multipartParts` 对齐（见 `upload.ts`）。
 */

/** 单条续传记录。 */
export interface MultipartResumeRecord {
  accId: string
  key: string
  bucket?: string
  uploadId: string
  parts: { partNumber: number; etag: string }[]
}

/** 续传记录的 localStorage 键（单键存全部记录，不新增散键）。 */
export const RESUME_STORAGE_KEY = 's3c.multipart.resume.v1'

/** 最多保留的续传记录数：超出时丢弃最早写入的记录，防大文件分段清单撑爆 localStorage。 */
export const MAX_RESUME_RECORDS = 20

/** 文件指纹：`name:size:lastModified`。不含凭证与内容；同一文件未改名 / 未改动时稳定。 */
export function fileFingerprint(file: File): string {
  return `${file.name}:${file.size}:${file.lastModified}`
}

/** localStorage 句柄；不可用（未定义 / 被策略禁用）时返回 undefined，续传降级为不持久化。 */
function localStore(): Storage | undefined {
  try {
    return typeof localStorage === 'undefined' ? undefined : localStorage
  } catch {
    // 某些沙箱 / 隐私模式下访问 localStorage 直接抛 SecurityError。
    return undefined
  }
}

/** 解析整表；记录损坏（非法 JSON / 非对象）时当作空表，由调用方重新 init。 */
function readStore(store: Storage): Record<string, MultipartResumeRecord> {
  const raw = store.getItem(RESUME_STORAGE_KEY)
  if (!raw) return {}
  try {
    const parsed: unknown = JSON.parse(raw)
    if (parsed !== null && typeof parsed === 'object') {
      return parsed as Record<string, MultipartResumeRecord>
    }
  } catch {
    /* 非法 JSON：忽略损坏记录 */
  }
  return {}
}

/** 写整表；配额满 / 存储被禁时静默放弃续传记录，不影响当前上传。 */
function writeStore(store: Storage, all: Record<string, MultipartResumeRecord>): void {
  try {
    store.setItem(RESUME_STORAGE_KEY, JSON.stringify(all))
  } catch {
    /* 续传记录是尽力而为的优化，写不进去不能让上传失败 */
  }
}

/** 读取某文件的续传记录（无记录 / 存储不可用时返回 undefined）。 */
export function loadResume(file: File): MultipartResumeRecord | undefined {
  const store = localStore()
  if (!store) return undefined
  return readStore(store)[fileFingerprint(file)]
}

/** 写入 / 更新某文件的续传记录（超出上限时丢最早记录）。 */
export function saveResume(file: File, record: MultipartResumeRecord): void {
  const store = localStore()
  if (!store) return
  const all = readStore(store)
  all[fileFingerprint(file)] = record
  const keys = Object.keys(all)
  if (keys.length > MAX_RESUME_RECORDS) {
    for (const k of keys.slice(0, keys.length - MAX_RESUME_RECORDS)) delete all[k]
  }
  writeStore(store, all)
}

/** 清除某文件的续传记录（完成 / 明确中止后调用）。 */
export function clearResume(file: File): void {
  const store = localStore()
  if (!store) return
  const all = readStore(store)
  delete all[fileFingerprint(file)]
  writeStore(store, all)
}
