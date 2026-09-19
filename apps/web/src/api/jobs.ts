import type { MigrationResult } from '../types'
import { request } from './http'
import { getBase, readToken } from './storage'

/** 异步任务（迁移/复制/删除）进度订阅：SSE + EOF 后状态回读兜底（从 `api.ts` 拆出）。 */

export interface MigrateProgress {
  done: number
  total: number
  migrated: number
  failed: number
  key?: string
  error?: string
  status?: string
}

/** 查询单个异步任务状态（`s3api.migrateJobStatus` 与本模块的 EOF 兜底轮询共用同一实现）。 */
export function migrateJobStatus(jobId: string) {
  return request<{ jobId: string; done: boolean; progress: MigrateProgress; result?: MigrationResult }>(
    `/api/migrate/jobs/${encodeURIComponent(jobId)}`,
  )
}

// 流 EOF 后回读 job 状态的重试参数：任务仍在运行则轮询直到终态（防 Promise 永久悬挂）；
// 连续多次回读失败视为网络不可用，快速 onError 让调用方复位（三个调用方均无自身超时兜底）。
const JOB_STATUS_POLL_MS = 30_000 // 任务仍在运行时的最长轮询
const JOB_STATUS_POLL_INTERVAL_MS = 500
const JOB_STATUS_MAX_CONSECUTIVE_FAILURES = 3

// SSE 空闲超时：服务端按固定间隔发 `event: ping` 心跳（handler 侧 15s），
// 因此「长时间收不到任何字节」只可能是连接被静默中断（NAT/代理超时、半开连接）。
// reader.read() 本身没有超时，会永久挂起 → EOF 后的补偿轮询根本不会启动，
// 调用方按钮永久禁用（review §F6）。任何数据（含心跳）到达都会重置该计时。
const SSE_IDLE_TIMEOUT_MS = 45_000

/** 订阅迁移 SSE 进度（fetch 流式，支持 Bearer）。返回 abort 函数。 */
export function subscribeMigrateEvents(
  jobId: string,
  onProgress: (p: MigrateProgress) => void,
  onError: (err: Error) => void,
  idleTimeoutMs: number = SSE_IDLE_TIMEOUT_MS,
): () => void {
  const ctrl = new AbortController()
  const headers: Record<string, string> = { Accept: 'text/event-stream' }
  const token = readToken()
  if (token) headers['Authorization'] = `Bearer ${token}`
  let reader: ReadableStreamDefaultReader<Uint8Array> | undefined
  /** 主动释放流：空闲超时/解析异常时不留悬挂连接与未消费的 reader。 */
  function releaseStream() {
    ctrl.abort()
    // 依据 ReadableStream 规范，cancel() 总是返回已敲定的 promise；
    // abort() 已让读取循环退出，故此处无需再挂 catch 兜底。
    void reader?.cancel()
  }
  ;(async () => {
    let lastStatus: string | undefined
    try {
      const res = await fetch(`${getBase()}/api/migrate/jobs/${encodeURIComponent(jobId)}/events`, {
        headers,
        signal: ctrl.signal,
      })
      if (!res.ok || !res.body) {
        throw new Error(`${res.status} ${res.statusText}`)
      }
      reader = res.body.getReader()
      const dec = new TextDecoder()
      let buf = ''
      /** 读一块数据，超过 idleTimeoutMs 未收到任何字节即判连接静默中断。 */
      const readWithIdleTimeout = async () => {
        const read = reader!.read()
        let timer: ReturnType<typeof setTimeout> | undefined
        try {
          return await Promise.race([
            read,
            new Promise<never>((_, reject) => {
              timer = setTimeout(() => reject(new Error(`migrate job ${jobId} stream idle timeout`)), idleTimeoutMs)
            }),
          ])
        } finally {
          clearTimeout(timer)
        }
      }
      for (;;) {
        // 每成功读到一块数据（进度事件或心跳）后重新开始计时。
        const { done, value } = await readWithIdleTimeout()
        if (done) break
        buf += dec.decode(value, { stream: true })
        let idx: number
        while ((idx = buf.indexOf('\n\n')) >= 0) {
          const block = buf.slice(0, idx)
          buf = buf.slice(idx + 2)
          let eventName = ''
          for (const line of block.split('\n')) {
            if (line.startsWith('event:')) {
              eventName = line.slice(6).trim()
            } else if (line.startsWith('data:')) {
              // 心跳 ping 忽略
              if (eventName === 'ping') continue
              const raw = line.startsWith('data: ') ? line.slice(6) : line.slice(5).trimStart()
              try {
                const p = JSON.parse(raw) as MigrateProgress
                if (p.status) lastStatus = p.status
                onProgress(p)
              } catch {
                /* ignore partial */
              }
            }
          }
        }
      }
      // 流正常 EOF 但未收到终态：回读并轮询 job 状态，直到终态或超时，
      // 避免 UI 永久卡在「迁移中」（旧实现只回读一次：回读失败或任务仍未完成即悬挂，
      // 三个调用方 ctxDeleteFolder / DestDialog / MigratePanel 均无超时兜底）。
      if (lastStatus !== 'done' && lastStatus !== 'cancelled' && !ctrl.signal.aborted) {
        const deadline = Date.now() + JOB_STATUS_POLL_MS
        let consecutiveFailures = 0
        for (;;) {
          if (ctrl.signal.aborted) break
          try {
            const st = await migrateJobStatus(jobId)
            consecutiveFailures = 0
            if (st.done) {
              // 终态：无论回读进度是否自带 status，都合成终态事件让调用方 resolve。
              const s = st.progress.status
              onProgress({ ...st.progress, status: s === 'cancelled' ? 'cancelled' : 'done' })
              break
            }
            // 任务仍在运行：上报一次当前进度（非终态），继续轮询。
            onProgress({ ...st.progress, status: st.progress.status || 'running' })
          } catch {
            // 网络抖动：连续失败达上限即判网络不可用，快速 onError（不再死等 deadline）。
            consecutiveFailures++
            if (consecutiveFailures >= JOB_STATUS_MAX_CONSECUTIVE_FAILURES) {
              onError(new Error(`migrate job ${jobId} status unavailable`))
              break
            }
          }
          if (Date.now() >= deadline) {
            onError(new Error(`migrate job ${jobId} status timeout`))
            break
          }
          await new Promise((r) => setTimeout(r, JOB_STATUS_POLL_INTERVAL_MS))
        }
      }
    } catch (e) {
      if (!ctrl.signal.aborted) {
        releaseStream()
        onError(e instanceof Error ? e : new Error(String(e)))
      }
    }
  })()
  return () => ctrl.abort()
}
