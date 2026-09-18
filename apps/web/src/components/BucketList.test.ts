import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import BucketList from './BucketList.vue'
import { fmtDate } from '../format'
import type { BucketItem } from '../types'

vi.mock('../i18n', () => ({
  t: (k: string) => k,
  tf: (k: string) => k,
}))

const sample: BucketItem[] = [
  { name: 'alpha', creationDate: '2024-01-02T00:00:00Z' },
  { name: 'beta', creationDate: '' },
]

function mountList(buckets: BucketItem[]) {
  return mount(BucketList, { props: { buckets } })
}

describe('BucketList', () => {
  it('empty 状态 + create 按钮 emit', async () => {
    const w = mountList([])
    expect(w.text()).toContain('buckets.emptyHint')
    await w.find('button').trigger('click')
    expect(w.emitted('create')).toBeTruthy()
  })

  it('渲染行、计数徽标与格式化日期', () => {
    const w = mountList(sample)
    expect(w.text()).toContain('buckets.countN')
    const rows = w.findAll('tbody tr')
    expect(rows).toHaveLength(2)
    expect(rows[0].text()).toContain('alpha')
    expect(rows[0].text()).toContain(fmtDate('2024-01-02T00:00:00Z'))
    // creationDate 为空 → fmtDate 返回空串，单元格仅占位
    expect(rows[1].text()).toContain('beta')
  })

  it('lifecycle / enter / remove 按钮 emit 对应事件', async () => {
    const w = mountList(sample)
    const btn = (text: string, idx: number) =>
      w.findAll('button').filter((b) => b.text() === text)[idx]
    await btn('buckets.tabLifecycle', 0)!.trigger('click')
    expect(w.emitted('lifecycle')).toEqual([['alpha']])
    await btn('buckets.enter', 1)!.trigger('click')
    expect(w.emitted('enter')).toEqual([['beta']])
    await btn('common.delete', 1)!.trigger('click')
    expect(w.emitted('remove')).toEqual([[sample[1]]])
  })
})
