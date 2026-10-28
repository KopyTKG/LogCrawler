import type { Piece } from './highlight'

/** An event of the log text: its first line and the continuation lines under it. */
export interface Entry {
  head: string
  rest: string[]
}

export interface Block {
  label: string
  json?: Piece[]
  stack?: string
}

const TIME = /^\d{2}:\d{2}:\d{2}\.\d{3} /

/** Group log text into events; a line without a timestamp belongs to the event above it. */
export function entries(text: string): Entry[] {
  const out: Entry[] = []
  for (const line of text.split('\n')) {
    if (TIME.test(line) || out.length === 0) out.push({ head: line, rest: [] })
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

function label(head: string): string {
  if (head.includes('✕ body')) return 'Response body'
  if (/\d ERROR /.test(head)) return 'Error'
  if (/payload/i.test(head)) return 'Payload'
  if (/params|parameters/i.test(head)) return 'Parameters'
  return 'Data'
}

/** The JSON object or array an event carries after its text, if its tail parses as one. */
export function jsonBlock(head: string): Block | null {
  for (let i = head.search(/[{[]/); i >= 0 && i < head.length;) {
    try {
      const value = JSON.parse(head.slice(i))
      if (value === null || typeof value !== 'object') return null
      const b: Block = { label: label(head) }
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
