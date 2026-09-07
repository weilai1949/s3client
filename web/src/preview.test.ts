import { describe, expect, it } from 'vitest'
import { previewKind } from './preview'

describe('previewKind', () => {
  const cases: Array<[string, string]> = [
    ['photo.PNG', 'image'],
    ['pic.jpeg', 'image'],
    ['pic.JPG', 'image'],
    ['anim.webp', 'image'],
    ['img.svg', 'image'],
    ['img.avif', 'image'],
    ['clip.MP4', 'video'],
    ['movie.webm', 'video'],
    ['mov.mov', 'video'],
    ['song.mp3', 'audio'],
    ['wave.wav', 'audio'],
    ['flac.x.flac', 'audio'],
    ['doc.PDF', 'pdf'],
    ['readme.txt', 'text'],
    ['page.html', 'text'],
    ['app.tsx', 'text'],
    ['main.go', 'text'],
    ['Makefile', 'none'],
    ['noext', 'none'],
    ['archive.zip', 'none'],
  ]
  for (const [key, want] of cases) {
    it(`${key} → ${want}`, () => {
      expect(previewKind(key)).toBe(want)
    })
  }
})
