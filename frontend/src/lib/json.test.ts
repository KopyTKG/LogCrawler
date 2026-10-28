import { describe, expect, it } from 'vitest'
import { entries, jsonBlock } from './json'

const text = (b: ReturnType<typeof jsonBlock>) => b?.json?.map((p) => p.text).join('')

describe('entries', () => {
  it('attaches lines without a timestamp to the event above', () => {
    const e = entries('10:00:00.000 LOG   [detail] a:\n  [1] x\n  [2] y\n10:00:01.000 LOG   b')
    expect(e).toHaveLength(2)
    expect(e[0].rest).toEqual(['  [1] x', '  [2] y'])
  })
})

describe('jsonBlock', () => {
  it('labels and pretty-prints a failure body', () => {
    const b = jsonBlock('10:00:00.000 LOG   [http] #7 ✕ body {"status":500,"requestId":"abc"}')
    expect(b?.label).toBe('Response body')
    expect(text(b)).toBe('{\n  "status": 500,\n  "requestId": "abc"\n}')
  })

  it('splits the stack of an error out as plain text', () => {
    const b = jsonBlock('10:00:00.000 ERROR {"message":"x","stack":"A\\n  at b"}')
    expect(b?.label).toBe('Error')
    expect(b?.stack).toBe('A\n  at b')
    expect(text(b)).not.toContain('stack')
  })

  it('finds JSON after text that has braces of its own', () => {
    expect(jsonBlock('10:00:00.000 LOG   [x] {not json} then {"a":1}')?.json).toBeTruthy()
  })

  it('returns null when there is no JSON', () => {
    expect(jsonBlock('10:00:00.000 LOG   [nav] A → B')).toBeNull()
    expect(jsonBlock('10:00:00.000 LOG   x {broken')).toBeNull()
  })
})
