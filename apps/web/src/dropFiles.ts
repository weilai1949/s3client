/**
 * 拖放收集（对象页拖拽上传）：把一次拖放展开为「文件 + 相对路径」列表。
 *
 * 为什么需要它：对象面板原先只能点「上传」按钮开文件选择框，而同类控制台（COS / OSS）
 * 都支持把文件、乃至**整个目录**拖进当前目录。目录拖放必须走 `webkitGetAsEntry()`
 * 递归展开——`DataTransfer.files` 只给扁平的顶层文件，目录会变成一个 0 字节条目，
 * 那样等于静默丢结构，比不支持更糟。
 *
 * 路径语义：保留拖入根的名字（拖 `photos/` 得到 `photos/a.jpg`），与浏览器上传行为一致。
 */

/** 单次拖放收集的文件上限：兜住病态目录树，避免一次性把内存打满。 */
export const MAX_DROPPED_FILES = 5000

export interface DroppedFile {
  file: File
  /** 相对路径（含拖入的顶层目录名；纯文件拖放等于文件名）。 */
  path: string
}

/** 拖放数据源的最小结构：DataTransfer 天然满足，测试也无需构造完整 DataTransfer。
 *  用 ArrayLike 而不是数组：`DataTransfer.items` 是 DataTransferItemList（类数组、无迭代器）。 */
export interface DropSource {
  items?: ArrayLike<DataTransferItem>
  files?: ArrayLike<File> | null
}

/** 读目录的全部条目：`readEntries` 每次最多返回 100 条，必须循环到返回空数组为止。 */
function readAllEntries(reader: FileSystemDirectoryReader): Promise<FileSystemEntry[]> {
  return new Promise((resolve, reject) => {
    const out: FileSystemEntry[] = []
    const step = () => {
      reader.readEntries((batch) => {
        if (!batch.length) {
          resolve(out)
          return
        }
        out.push(...batch)
        step()
      }, reject)
    }
    step()
  })
}

/** 取 file 对象；读失败（权限/被移动）返回 null，由调用方跳过该条而不中断整次拖放。 */
function readFileEntry(entry: FileSystemFileEntry): Promise<File | null> {
  return new Promise((resolve) => {
    entry.file((f) => resolve(f), () => resolve(null))
  })
}

async function collect(entry: FileSystemEntry, out: DroppedFile[]): Promise<void> {
  if (out.length >= MAX_DROPPED_FILES) return
  if (entry.isFile) {
    const file = await readFileEntry(entry as FileSystemFileEntry)
    if (file) out.push({ file, path: entry.fullPath.replace(/^\/+/, '') || file.name })
    return
  }
  // 目录读取失败（上限/权限）按空目录处理：不中断整次拖放。
  const entries = await readAllEntries((entry as FileSystemDirectoryEntry).createReader()).catch(
    () => [] as FileSystemEntry[],
  )
  for (const child of entries) await collect(child, out)
}

/**
 * 展开一次拖放。
 *
 * 优先用 `items` 的 entry（可递归目录）；一律拿不到 entry 时（老浏览器 / 非目录拖放）
 * 退回扁平的 `files`，至少把文件传上去。
 */
export async function filesFromDrop(src: DropSource): Promise<DroppedFile[]> {
  const out: DroppedFile[] = []
  const entries: FileSystemEntry[] = []
  for (const item of src.items ? Array.from(src.items) : []) {
    if (item.kind !== 'file') continue
    const entry = item.webkitGetAsEntry()
    if (entry) entries.push(entry)
  }
  if (entries.length) {
    for (const entry of entries) await collect(entry, out)
    return out
  }
  for (const file of Array.from(src.files ?? [])) out.push({ file, path: file.name })
  return out
}
