import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import UploadQueue from './UploadQueue.vue'
import type { UploadQueueItem } from '../composables/useUploadQueue'

vi.mock('../i18n', () => ({
  t: (k: string) => k,
  tf: (k: string) => k,
}))

function makeItem(overrides: Partial<UploadQueueItem>): UploadQueueItem {
  return {
    id: 1,
    file: new File(['x'], 'a.txt'),
    key: 'a.txt',
    pct: 0,
    status: 'pending',
    ...overrides,
  }
}

describe('UploadQueue', () => {
  it('renders nothing when items is empty', () => {
    const w = mount(UploadQueue, { props: { items: [] } })
    expect(w.find('.upload-inline').exists()).toBe(false)
    expect(w.text()).toBe('')
  })

  it('renders per-item status text and overall progress for mixed statuses', () => {
    const items = [
      makeItem({ id: 1, key: 'p.txt', status: 'pending', pct: 0 }),
      makeItem({ id: 2, key: 's.txt', status: 'signing', pct: 5 }),
      makeItem({ id: 3, key: 'u.txt', status: 'uploading', pct: 50 }),
      makeItem({ id: 4, key: 'd.txt', status: 'done', pct: 100 }),
      makeItem({ id: 5, key: 'c.txt', status: 'cancelled', pct: 0 }),
      makeItem({ id: 6, key: 'e1.txt', status: 'err', pct: 10, err: 'boom' }),
      makeItem({ id: 7, key: 'e2.txt', status: 'err', pct: 10 }),
    ]
    const w = mount(UploadQueue, { props: { items } })
    const root = w.find('.upload-inline')
    expect(root.exists()).toBe(true)

    // 总体进度： (0+5+50+100+0+10+10)/7 = 25；done 计数 = 1
    expect(w.text()).toContain('1/7 · 25%')
    const bar = root.find('.progress .bar')
    expect(bar.attributes('style')).toContain('width: 25%')

    const badges = w.findAll('.upload-items .badge')
    expect(badges[0].text()).toContain('p.txt — 0%')
    expect(badges[1].text()).toContain('s.txt — 5%')
    expect(badges[2].text()).toContain('u.txt — 50%')
    expect(badges[3].text()).toContain('d.txt — ✓')
    expect(badges[4].text()).toContain('c.txt — upload.statusCancelled')
    expect(badges[5].text()).toContain('e1.txt — boom')
    expect(badges[6].text()).toContain('e2.txt — upload.statusErr')
  })

  it('shows cancel buttons only for uploading/pending and emits cancel with the item', async () => {
    const items = [
      makeItem({ id: 1, key: 'a.txt', status: 'pending', pct: 0 }),
      makeItem({ id: 2, key: 'u.txt', status: 'uploading', pct: 30 }),
      makeItem({ id: 3, key: 'd.txt', status: 'done', pct: 100 }),
      makeItem({ id: 4, key: 'e.txt', status: 'err', pct: 0, err: 'x' }),
    ]
    const w = mount(UploadQueue, { props: { items } })
    const cancelBtns = w.findAll('button.cancel-btn')
    expect(cancelBtns).toHaveLength(2)
    await cancelBtns[0].trigger('click')
    expect(w.emitted('cancel')?.[0]).toEqual([items[0]])
    await cancelBtns[1].trigger('click')
    expect(w.emitted('cancel')?.[1]).toEqual([items[1]])
  })

  it('renders 0% overall when nothing has progressed', () => {
    const items = [
      makeItem({ id: 1, bucket: 'b1', key: 'a.txt', status: 'pending', pct: 0 }),
      makeItem({ id: 2, bucket: 'b2', key: 'a.txt', status: 'pending', pct: 0 }),
    ]
    const w = mount(UploadQueue, { props: { items } })
    expect(w.text()).toContain('0/2 · 0%')
  })

  it('v-for 用唯一 id 作 key：同 bucket+key 条目移除前者时保留后者的 DOM 节点', async () => {
    // 两个目录选同名文件 / 重复入队同名文件 → bucket|key 组合键碰撞
    const a = makeItem({ id: 1, bucket: 'b1', key: 'same.txt', status: 'pending', pct: 0 })
    const b = makeItem({ id: 2, bucket: 'b1', key: 'same.txt', status: 'done', pct: 100 })
    const w = mount(UploadQueue, { props: { items: [a, b] } })
    const badges = w.findAll('.upload-items .badge.mono')
    expect(badges).toHaveLength(2)
    const survivor = badges[1].element

    await w.setProps({ items: [b] })
    const after = w.findAll('.upload-items .badge.mono')
    expect(after).toHaveLength(1)
    expect(after[0].text()).toContain('✓')
    // 组合键碰撞时 keyed diff 把 b 的内容打到 a 的节点上（存活条目换节点）；id 键则原地保留
    expect(after[0].element).toBe(survivor)
  })

  it('空条目时 uploadPct 走 early-return 返回 0（避免 reduce 空数组）', () => {
    const w = mount(UploadQueue, { props: { items: [] } })
    expect((w.vm as unknown as { uploadPct: number }).uploadPct).toBe(0)
  })
})
