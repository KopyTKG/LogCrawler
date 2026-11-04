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
    expect(b?.label).toBe('block.responseBody')
    expect(text(b)).toBe('{\n  "status": 500,\n  "requestId": "abc"\n}')
  })

  it('splits the stack of an error out as plain text', () => {
    const b = jsonBlock('10:00:00.000 ERROR {"message":"x","stack":"A\\n  at b"}')
    expect(b?.label).toBe('block.error')
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

describe('transitionsBlock', () => {
  it('turns an offered-transitions line into a table', () => {
    const head =
      '14:20:19.604 LOG  [transition] issue 7 offers 2 transition(s): #1 "Přijmout, hned" (StateChange, oneClick=true), #2 "Zamítnout" (StateChange, oneClick=false)'
    const b = jsonBlock(head)
    expect(b?.label).toBe('block.transitions')
    expect(b?.table?.rows).toEqual([
      ['1', 'Přijmout, hned', 'StateChange', 'yes'],
      ['2', 'Zamítnout', 'StateChange', ''],
    ])
    expect(head.slice(0, b?.start).trimEnd()).toMatch(/offers 2 transition\(s\):$/)
  })
})

describe('entries of a group', () => {
  it('reads indented child lines as events, keeping the indent', () => {
    const e = entries('10:00:00.000 LOG   IssueDetail\n  10:00:01.000 LOG   [http] #1 ← body {"a":1}')
    expect(e).toHaveLength(2)
    expect(e[1].indent).toBe('  ')
    expect(e[1].head).toBe('10:00:01.000 LOG   [http] #1 ← body {"a":1}')
    expect(jsonBlock(e[1].head)?.label).toBe('block.responseBody')
  })
})
