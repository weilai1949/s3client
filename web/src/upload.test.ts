import { describe, expect, it, vi } from 'vitest'

vi.mock('./api', () => ({
  s3api: {
    presign: vi.fn(),
    multipartInit: vi.fn(),
    multipartPart: vi.fn(),
    multipartComplete: vi.fn(),
    multipartAbort: vi.fn(),
  },
  directUpload: vi.fn(),
}))

import { MULTIPART_THRESHOLD, calcMultipartParts, shouldUseMultipart, withRetries, uploadObject } from './upload'
import { s3api, directUpload } from './api'

describe('upload multipart helpers', () => {
  it('calcMultipartParts rounds up', () => {
    expect(calcMultipartParts(0)).toBe(0)
    expect(calcMultipartParts(1)).toBe(1)
    expect(calcMultipartParts(10 * 1024 * 1024)).toBe(1)
    expect(calcMultipartParts(10 * 1024 * 1024 + 1)).toBe(2)
  })

  it('shouldUseMultipart at 100MB threshold', () => {
    expect(shouldUseMultipart(MULTIPART_THRESHOLD - 1)).toBe(false)
    expect(shouldUseMultipart(MULTIPART_THRESHOLD)).toBe(true)
  })
})

describe('withRetries', () => {
  it('retries on failure then succeeds', async () => {
    let n = 0
    const result = await withRetries(
      async () => {
        n++
        if (n < 3) throw new Error('transient')
        return 'ok'
      },
      [1, 1, 1],
    )
    expect(result).toBe('ok')
    expect(n).toBe(3)
  })

  it('does not retry AbortError', async () => {
    let n = 0
    await expect(
      withRetries(async () => {
        n++
        throw new DOMException('Aborted', 'AbortError')
      }, [1, 1]),
    ).rejects.toMatchObject({ name: 'AbortError' })
    expect(n).toBe(1)
  })

  it('exhausts retries and throws last error', async () => {
    let n = 0
    await expect(
      withRetries(async () => {
        n++
        throw new Error(`fail-${n}`)
      }, [1, 1]),
    ).rejects.toThrow('fail-3')
    expect(n).toBe(3)
  })

  it('respects abort signal between attempts', async () => {
    const ctrl = new AbortController()
    let n = 0
    const p = withRetries(
      async () => {
        n++
        if (n === 1) {
          ctrl.abort()
          throw new Error('first')
        }
        return 'never'
      },
      [5, 5],
      ctrl.signal,
    )
    await expect(p).rejects.toMatchObject({ name: 'AbortError' })
    expect(n).toBe(1)
  })

  it('aborts during sleep delay', async () => {
    const ctrl = new AbortController()
    let n = 0
    const p = withRetries(
      async () => {
        n++
        throw new Error(`fail-${n}`)
      },
      [100],
      ctrl.signal,
    )
    setTimeout(() => ctrl.abort(), 10)
    await expect(p).rejects.toMatchObject({ name: 'AbortError' })
    expect(n).toBe(1)
  })
})

describe('uploadObject', () => {
  it('small file uses presign + directUpload', async () => {
    const file = new File(['small'], 'small.txt', { type: 'text/plain' })
    s3api.presign.mockResolvedValue({ url: 'https://presigned.url' })
    directUpload.mockResolvedValue(undefined)
    await uploadObject(file, { accId: 'acc1', key: 'small.txt' })
    expect(s3api.presign).toHaveBeenCalled()
    expect(directUpload).toHaveBeenCalled()
  })

  it('large file uses multipart upload', async () => {
    const file = new File([new ArrayBuffer(MULTIPART_THRESHOLD)], 'large.bin', { type: 'application/octet-stream' })
    s3api.multipartInit.mockResolvedValue({ uploadId: 'up1', key: 'large.bin', bucket: 'mybucket' })
    s3api.multipartPart.mockResolvedValue({ partNumber: 1, url: 'https://part.url', expiresIn: 3600 })
    s3api.multipartComplete.mockResolvedValue({ completed: 'ok' })
    await uploadObject(file, { accId: 'acc1', bucket: 'mybucket', key: 'large.bin' })
    expect(s3api.multipartInit).toHaveBeenCalled()
    expect(s3api.multipartComplete).toHaveBeenCalled()
  })

  it('aborts when signal is already aborted', async () => {
    const ctrl = new AbortController()
    ctrl.abort()
    const file = new File(['small'], 'small.txt', { type: 'text/plain' })
    await expect(uploadObject(file, { accId: 'acc1', key: 'small.txt' }, undefined, ctrl.signal))
      .rejects.toMatchObject({ name: 'AbortError' })
  })
})
