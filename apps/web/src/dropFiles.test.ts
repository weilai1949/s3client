import { describe, expect, it } from 'vitest'
import { MAX_DROPPED_FILES, filesFromDrop } from './dropFiles'

/** 假文件条目：file() 成功回调直接产出 File。 */
function fileEntry(name: string, fullPath = '/' + name): FileSystemEntry {
  return {
    isFile: true,
    isDirectory: false,
    name,
    fullPath,
    file: (ok: (f: File) => void) => ok(new File(['x'], name)),
  } as unknown as FileSystemEntry
}

/** 假文件条目：file() 走错误回调（读失败/被移动）——该条应被跳过而不中断整次拖放。 */
function brokenFileEntry(name: string): FileSystemEntry {
  return {
    isFile: true,
    isDirectory: false,
    name,
    fullPath: '/' + name,
    file: (_ok: (f: File) => void, err: (e: unknown) => void) => err(new Error('gone')),
  } as unknown as FileSystemEntry
}

/** 假目录条目：readEntries 按批次依次返回，最后一批为空数组（真实 API 语义）。 */
function dirEntry(
  name: string,
  batches: FileSystemEntry[][],
  fail = false,
): FileSystemEntry {
  let call = 0
  return {
    isFile: false,
    isDirectory: true,
    name,
    fullPath: '/' + name,
    createReader: () => ({
      readEntries: (ok: (entries: FileSystemEntry[]) => void, err?: (e: unknown) => void) => {
        if (fail) {
          err?.(new DOMException('denied'))
          return
        }
        ok(batches[call++] ?? [])
      },
    }),
  } as unknown as FileSystemEntry
}

function item(entry: FileSystemEntry | null, kind: string = 'file'): DataTransferItem {
  return { kind, webkitGetAsEntry: () => entry } as unknown as DataTransferItem
}

function fileList(files: File[]): FileList {
  return files as unknown as FileList
}

describe('filesFromDrop 拖放展开', () => {
  it('单文件：路径取 entry.fullPath（去掉前导斜杠）', async () => {
    const out = await filesFromDrop({ items: [item(fileEntry('a.txt', '/a.txt'))] })
    expect(out).toHaveLength(1)
    expect(out[0].file.name).toBe('a.txt')
    expect(out[0].path).toBe('a.txt')
  })

  it('目录递归：保留拖入根的名字与层级（photos/2024/a.jpg）', async () => {
    // photos → [a.jpg, 2024/]；2024 → [b.jpg]
    const sub = dirEntry('2024', [[fileEntry('b.jpg', '/photos/2024/b.jpg')], []])
    const photos = dirEntry('photos', [[fileEntry('a.jpg', '/photos/a.jpg'), sub], []])
    const out = await filesFromDrop({ items: [item(photos)] })
    expect(out.map((f) => f.path)).toEqual(['photos/a.jpg', 'photos/2024/b.jpg'])
  })

  it('readEntries 分批返回（>100 条）时循环读取到空数组为止', async () => {
    const dir = dirEntry('d', [
      [fileEntry('one', '/d/one'), fileEntry('two', '/d/two')],
      [fileEntry('three', '/d/three')],
      [],
    ])
    const out = await filesFromDrop({ items: [item(dir)] })
    expect(out.map((f) => f.path)).toEqual(['d/one', 'd/two', 'd/three'])
  })

  it('目录读取失败：按空目录处理，不抛错', async () => {
    const out = await filesFromDrop({ items: [item(dirEntry('broken', [], true))] })
    expect(out).toEqual([])
  })

  it('文件读取失败：跳过该条，其余文件照常上传', async () => {
    const dir = dirEntry('d', [[brokenFileEntry('bad'), fileEntry('good', '/d/good')], []])
    const out = await filesFromDrop({ items: [item(dir)] })
    expect(out.map((f) => f.path)).toEqual(['d/good'])
  })

  it('entry 没有可用 fullPath 时退回文件名（某些实现只给根路径）', async () => {
    const out = await filesFromDrop({ items: [item(fileEntry('bare.txt', '/'))] })
    expect(out.map((f) => f.path)).toEqual(['bare.txt'])
  })

  it('跳过非文件项（拖入文本/链接）', async () => {
    const out = await filesFromDrop({ items: [item(null, 'string'), item(fileEntry('a.txt'), 'file')] })
    expect(out.map((f) => f.path)).toEqual(['a.txt'])
  })

  it('全部拿不到 entry 时退回扁平 files（老浏览器兜底）', async () => {
    const a = new File(['1'], 'a.txt')
    const b = new File(['2'], 'b.txt')
    const out = await filesFromDrop({ items: [item(null)], files: fileList([a, b]) })
    expect(out.map((f) => f.path)).toEqual(['a.txt', 'b.txt'])
  })

  it('没有 items（只有 files）时同样退回扁平 files', async () => {
    const out = await filesFromDrop({ files: fileList([new File(['1'], 'only.txt')]) })
    expect(out.map((f) => f.path)).toEqual(['only.txt'])
  })

  it('无 items 也无 files：返回空（拖入非文件内容不报错）', async () => {
    expect(await filesFromDrop({})).toEqual([])
  })

  it('超过上限即停止收集（病态目录树不把内存打满）', async () => {
    const many = Array.from({ length: MAX_DROPPED_FILES + 5 }, (_, i) => fileEntry(`f${i}`, `/d/f${i}`))
    const out = await filesFromDrop({ items: [item(dirEntry('d', [many, []]))] })
    expect(out).toHaveLength(MAX_DROPPED_FILES)
  })
})
