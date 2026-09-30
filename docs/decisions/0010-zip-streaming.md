# ADR-010：ZIP 服务端流式打包（不落盘 + 客户端流式落盘 / blob 兜底）

## Status

Accepted

## Date

2026-09-18（回溯记录；`service/zip.go` 于 2026-09-18 首次引入；前端 `api/download.ts`
于 2026-09-19 引入）

## Context

批量下载多对象需要打包；对象数量 / 体积可能很大。若服务端先落盘临时 ZIP 再返回，会引入
磁盘占用与临时文件生命周期管理成本；若全读进内存，则与容器内存限额冲突。客户端侧，
`<a download>` 整包进内存同样有 OOM 风险。

现状（2026-09-30 回读源码核实）：

- [`apps/server/internal/service/zip.go`](../../apps/server/internal/service/zip.go)
  `WriteObjectsZip`：流式写入 `zip.Writer`（写入目标由调用方给 `io.Writer`，即响应流）；
  拉取并发 4、写入串行；失败 key 记入清单并写入包内 `_下载失败清单.txt`；
  `SanitizeZipName` 防 zip-slip；`LikelyCompressed` 对已压缩类型用 `Store` 而非 `Deflate`；
  `ctxCancelReader` 在 ctx 取消时关闭底层 body 中断阻塞读。
- 前端 [`apps/web/src/api/download.ts`](../../apps/web/src/api/download.ts)：
  优先 **File System Access API**（`showSaveFilePicker` + `createWritable` 流式落盘），
  否则 **blob 兜底**——`Content-Length > 500MB` 或长度缺失且 `keys > 50` 时**拒绝**（防 OOM）；
  object URL 延迟 60s 回收（避免中断尚未开始的下载）。
- [`user-guide.md`](../user-guide.md) §五：服务端流式打包，不在服务端落盘。

## Decision

**ZIP 由服务端边取边压边写响应流，不在服务端落盘；浏览器侧优先流式写盘，无该能力时
blob 兜底并设 500MB / 50 keys 上限。**

## Alternatives Considered

### 服务端先打包成临时文件再返回
- Pros：可支持重试、`Content-Length` 确定、便于校验。
- Cons：磁盘占用与临时文件清理 / 残留管理（`zip_leak_test.go` 的存在说明此面被重点防守过）。
- （「当时的对比讨论」未找到原始记录，此条为回溯补记，**未验证**。）

### 前端逐个下载不打包
- Pros：服务端零实现。
- Cons：N 次请求、无法「一键下载一批」，与批量下载的产品需求冲突。
- 被拒。

### 服务端全部读入内存再打包
- Pros：实现简单。
- Cons：内存 O(总大小)，与 512MB 容器限额冲突。
- 被拒。

## Consequences

- 流式响应无 `Content-Length` → 客户端不能依赖大小做进度条；blob 兜底因此设
  500MB / 50 keys 上限并在超限时显式报错（`api.zipTooLarge`）。
- 拉取与写入解耦后，**取消路径必须继续消费 results 通道**，否则 worker 永久阻塞
  （`zip.go` 注释 review §B4）。
- 包内失败清单 `_下载失败清单.txt` 成为跨语言可读的失败契约。
