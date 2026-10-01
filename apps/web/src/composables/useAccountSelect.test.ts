import { beforeEach, describe, expect, it, vi } from 'vitest'

// 只需 composable 依赖的两个导出：state 与 rememberedAccountId。
vi.mock('../store', async () => {
  const { reactive } = await import('vue')
  return {
    state: reactive({ accounts: [] as Account[], currentAccountId: '' }),
    rememberedAccountId: vi.fn(() => ''),
  }
})

import { state, rememberedAccountId } from '../store'
import { resolveAccountSelect } from './useAccountSelect'
import type { Account } from '../types'

function acc(id: string): Account {
  return {
    id,
    name: id,
    endpoint: '127.0.0.1:9000',
    region: 'us-east-1',
    accessKey: 'ak',
    secretSet: true,
    bucket: 'b1',
    pathStyle: true,
    useSSL: false,
   createdAt: '2024-01-01T00:00:00Z', publicEndpoint: '', updatedAt: '2024-01-01T00:00:00Z'}
}

beforeEach(() => {
  state.accounts = []
  state.currentAccountId = ''
  vi.mocked(rememberedAccountId).mockReturnValue('')
})

describe('resolveAccountSelect', () => {
  it('remembered 账号仍存在时优先恢复（优先级高于 currentAccountId）', () => {
    state.accounts = [acc('a1'), acc('a2')]
    state.currentAccountId = 'a1'
    vi.mocked(rememberedAccountId).mockReturnValue('a2')
    expect(resolveAccountSelect()).toBe('a2')
  })

  it('remembered 失效时沿用仍有效的 currentAccountId（不回退第一个）', () => {
    state.accounts = [acc('a1'), acc('a2')]
    state.currentAccountId = 'a2'
    vi.mocked(rememberedAccountId).mockReturnValue('gone-acc')
    expect(resolveAccountSelect()).toBe('a2')
  })

  it('currentAccountId 已不在账号列表时回退第一个账号', () => {
    state.accounts = [acc('a1'), acc('a2')]
    state.currentAccountId = 'stale-acc'
    expect(resolveAccountSelect()).toBe('a1')
  })

  it('无 remembered / current 为空时取第一个账号', () => {
    state.accounts = [acc('a1'), acc('a2')]
    state.currentAccountId = ''
    expect(resolveAccountSelect()).toBe('a1')
  })

  it('无账号可选时返回空串', () => {
    state.accounts = []
    state.currentAccountId = ''
    expect(resolveAccountSelect()).toBe('')
  })
})
