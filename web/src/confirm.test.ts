import { describe, expect, it, beforeEach } from 'vitest'
import { confirmState, confirmDialog, settleConfirm } from './confirm'

beforeEach(() => {
  confirmState.open = false
  confirmState.resolve = null
  confirmState.title = ''
  confirmState.message = ''
  confirmState.danger = true
  confirmState.confirmText = ''
})

describe('confirmDialog', () => {
  it('opens dialog and returns promise', async () => {
    const p = confirmDialog({ title: 'Delete?', message: 'Sure?' })
    expect(confirmState.open).toBe(true)
    expect(confirmState.title).toBe('Delete?')
    expect(confirmState.message).toBe('Sure?')
    expect(confirmState.danger).toBe(true)
    settleConfirm(true)
    await expect(p).resolves.toBe(true)
  })

  it('defaults danger to true', async () => {
    confirmDialog({ title: 'T', message: 'M' })
    expect(confirmState.danger).toBe(true)
    settleConfirm(true)
  })

  it('settles old dialog when new one opens', async () => {
    const p1 = confirmDialog({ title: 'T1', message: 'M1' })
    const p2 = confirmDialog({ title: 'T2', message: 'M2' })
    // p1 is settled with false by the second confirmDialog call
    await expect(p1).resolves.toBe(false)
    settleConfirm(true)
    await expect(p2).resolves.toBe(true)
  })
})

describe('settleConfirm', () => {
  it('closes dialog and resolves with ok=true', async () => {
    const p = confirmDialog({ title: 'T', message: 'M' })
    settleConfirm(true)
    expect(confirmState.open).toBe(false)
    expect(confirmState.resolve).toBeNull()
    await expect(p).resolves.toBe(true)
  })

  it('closes dialog and resolves with ok=false', async () => {
    const p = confirmDialog({ title: 'T', message: 'M' })
    settleConfirm(false)
    expect(confirmState.open).toBe(false)
    await expect(p).resolves.toBe(false)
  })
})
