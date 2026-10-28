import { describe, expect, it } from 'vitest'
import { highlight, tagHue } from './highlight'

const classes = (line: string) =>
  highlight(line)
    .filter((p) => p.cls)
    .map((p) => `${p.cls}:${p.text.trim()}`)

describe('highlight', () => {
  it('colours the parts of a failed http line', () => {
    expect(classes('19:17:09.647 LOG   [http] #7 ✕ 500 GET https://x.test/a 8526ms — failed')).toEqual([
      'tk-time:19:17:09.647',
      'tk-tag tag-http:[http]',
      'tk-id:#7',
      'tk-fail:✕',
      'tk-fail:500',
      'tk-method:GET',
      'tk-url:https://x.test/a',
      'tk-dur:8526ms',
    ])
  })

  it('classes a success status by its range', () => {
    expect(classes('#1 ← 200 GET /x 5ms')).toContain('st-2:200')
    expect(classes('#1 ← 404 GET /x 5ms')).toContain('st-4:404')
  })

  it('marks ERROR and WARN', () => {
    expect(classes('19:00:00.000 ERROR {"a":1}')).toContain('tk-lvl-error:ERROR')
    expect(classes('WARN something')).toContain('tk-lvl-warn:WARN')
  })

  it('keeps every character of the line', () => {
    const line = '19:17:09.647 LOG   [nav] A → B {"x":"y"}'
    expect(
      highlight(line)
        .map((p) => p.text)
        .join(''),
    ).toBe(line)
  })

  it('gives a tag the same hue every time, known tags fixed ones', () => {
    expect(tagHue('http')).toBe(210)
    expect(tagHue('somethingnew')).toBe(tagHue('somethingnew'))
  })
})
