import { describe, expect, it } from 'vitest'
import { highlightSegments } from './highlight'

describe('highlightSegments', () => {
  it('空查询 / 纯空白查询：整段未命中', () => {
    expect(highlightSegments('a.txt', '')).toEqual([{ text: 'a.txt', hit: false }])
    expect(highlightSegments('a.txt', '   ')).toEqual([{ text: 'a.txt', hit: false }])
  })

  it('无命中：整段原样返回（不是空数组）', () => {
    expect(highlightSegments('a.txt', 'zzz')).toEqual([{ text: 'a.txt', hit: false }])
  })

  it('前缀命中：命中段 + 尾部未命中段', () => {
    expect(highlightSegments('report.csv', 'rep')).toEqual([
      { text: 'rep', hit: true },
      { text: 'ort.csv', hit: false },
    ])
  })

  it('中缀命中：三段（前 / 命中 / 后）', () => {
    expect(highlightSegments('my-report.csv', 'report')).toEqual([
      { text: 'my-', hit: false },
      { text: 'report', hit: true },
      { text: '.csv', hit: false },
    ])
  })

  it('后缀命中：不产生空的尾段', () => {
    expect(highlightSegments('a.txt', '.txt')).toEqual([
      { text: 'a', hit: false },
      { text: '.txt', hit: true },
    ])
  })

  it('多次命中：逐段切分', () => {
    expect(highlightSegments('abab', 'ab')).toEqual([
      { text: 'ab', hit: true },
      { text: 'ab', hit: true },
    ])
  })

  it('大小写不敏感匹配、保留原文大小写', () => {
    expect(highlightSegments('ReadMe.md', 'readme')).toEqual([
      { text: 'ReadMe', hit: true },
      { text: '.md', hit: false },
    ])
  })

  it('查询词含前后空格时按去除空白后的词匹配', () => {
    expect(highlightSegments('a.txt', ' txt ')).toEqual([
      { text: 'a.', hit: false },
      { text: 'txt', hit: true },
    ])
  })
})
