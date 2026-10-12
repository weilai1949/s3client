import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import HighlightText from './HighlightText.vue'

describe('HighlightText', () => {
  it('无过滤词：整段文本，无 mark', () => {
    const w = mount(HighlightText, { props: { text: 'a.txt', query: '' } })
    expect(w.text()).toBe('a.txt')
    expect(w.findAll('mark')).toHaveLength(0)
  })

  it('命中片段包在 mark 里，未命中部分保持原样（文本拼接后与原文一致）', () => {
    const w = mount(HighlightText, { props: { text: 'my-report.csv', query: 'report' } })
    const marks = w.findAll('mark')
    expect(marks).toHaveLength(1)
    expect(marks[0].text()).toBe('report')
    expect(w.text()).toBe('my-report.csv')
  })

  it('多次命中：多个 mark', () => {
    const w = mount(HighlightText, { props: { text: 'abab', query: 'ab' } })
    expect(w.findAll('mark')).toHaveLength(2)
  })

  it('过滤词变化时重新计算（响应式）', async () => {
    const w = mount(HighlightText, { props: { text: 'a.txt', query: '' } })
    await w.setProps({ query: 'txt' })
    expect(w.findAll('mark')).toHaveLength(1)
  })
})
