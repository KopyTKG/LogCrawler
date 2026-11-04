import type { Piece } from './highlight'
import type { Key } from './locales/en'

/** An event of the log text: its first line and the continuation lines under it. */
export interface Entry {
  head: string
  /** Leading whitespace of the head line: a group's text indents its children. */
  indent: string
  rest: string[]
}

export interface Block {
  /** An i18n key. */
  label: Key
  /** Where in the event's first line the JSON starts. */
  start: number
  json?: Piece[]
  table?: { head: Key[]; rows: string[][] }
  stack?: string
}

const TIME = /^\s*\d{2}:\d{2}:\d{2}\.\d{3} /

/** Group log text into events; a line without a timestamp belongs to the event above it. */
export function entries(text: string): Entry[] {
  const out: Entry[] = []
  for (const line of text.split('\n')) {
    if (TIME.test(line) || out.length === 0)
      out.push({
        head: line.trimStart(),
        indent: line.slice(0, line.length - line.trimStart().length),
        rest: [],
      })
    else out[out.length - 1].rest.push(line)
  }
  while (out.length && out[out.length - 1].head === '' && out[out.length - 1].rest.length === 0) out.pop()
  return out
}

const JSON_TOKEN = /"(?:\\.|[^"\\])*"(\s*:)?|-?\d+(?:\.\d+)?(?:[eE][+-]?\d+)?|true|false|null/g

/** Pretty-printed JSON with keys, strings and numbers told apart. */
export function jsonPieces(value: unknown): Piece[] {
  const src = JSON.stringify(value, null, 2)
  const out: Piece[] = []
  let last = 0
  for (const m of src.matchAll(JSON_TOKEN)) {
    if (m.index > last) out.push({ text: src.slice(last, m.index), cls: '' })
    const t = m[0]
    let cls = 'tk-num'
    if (t[0] === '"') cls = m[1] ? 'tk-key' : 'tk-str'
    else if (t === 'null') cls = 'tk-null'
    else if (t === 'true' || t === 'false') cls = 'tk-bool'
    out.push({ text: t, cls })
    last = m.index + t.length
  }
  if (last < src.length) out.push({ text: src.slice(last), cls: '' })
  return out
}

function label(head: string): Key {
  if (head.includes('✕ body') || head.includes('← body')) return 'block.responseBody'
  if (head.includes('→ body')) return 'block.requestBody'
  if (head.includes('✕ sent')) return 'block.requestSent'
  if (/[→←] headers/.test(head)) return 'block.headers'
  if (/\d ERROR /.test(head)) return 'block.error'
  if (/payload/i.test(head)) return 'block.payload'
  if (/params|parameters/i.test(head)) return 'block.parameters'
  return 'block.data'
}

const OFFERS = /offers \d+ transition\(s\): /
const OFFER = /#(\d+) "([^"]*)" \(([^,)]*), oneClick=(true|false)\)/g

/** "issue 7 offers 2 transition(s): #1 "Name" (Type, oneClick=false), …" as a table. */
function transitionsBlock(head: string): Block | null {
  const m = OFFERS.exec(head)
  if (!m) return null
  const start = m.index + m[0].length
  const rows = [...head.slice(start).matchAll(OFFER)].map((o) => [
    o[1],
    o[2],
    o[3],
    o[4] === 'true' ? 'yes' : '',
  ])
  if (!rows.length) return null
  return {
    label: 'block.transitions',
    start,
    table: { head: ['col.id', 'col.name', 'col.type', 'col.oneClick'], rows },
  }
}

/** The JSON object or array an event carries after its text, if its tail parses as one. */
export function jsonBlock(head: string): Block | null {
  const t = transitionsBlock(head)
  if (t) return t
  for (let i = head.search(/[{[]/); i >= 0 && i < head.length;) {
    try {
      const value = JSON.parse(head.slice(i))
      if (value === null || typeof value !== 'object') return null
      const b: Block = { label: label(head), start: i }
      if (!Array.isArray(value) && typeof value.stack === 'string') {
        b.stack = value.stack
        delete value.stack
      }
      if (Object.keys(value).length > 0) b.json = jsonPieces(value)
      return b.json || b.stack ? b : null
    } catch {
      const next = head.slice(i + 1).search(/[{[]/)
      if (next < 0) return null
      i += 1 + next
    }
  }
  return null
}
