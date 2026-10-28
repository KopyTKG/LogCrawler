/** Splits one log line into coloured pieces. The class names are styled in app.css. */
export interface Piece {
  text: string
  cls: string
}

const KNOWN_HUES: Record<string, number> = {
  http: 210,
  nav: 140,
  transition: 280,
  detail: 30,
  app: 175,
  auth: 340,
  sync: 60,
}

/** A stable hue (0-359) for a tag; known tags keep fixed colours, unknown ones are hashed. */
export function tagHue(tag: string): number {
  if (tag in KNOWN_HUES) return KNOWN_HUES[tag]
  let h = 0
  for (const c of tag) h = (h * 31 + c.charCodeAt(0)) % 360
  return h
}

const TOKEN = new RegExp(
  [
    '^\\d{2}:\\d{2}:\\d{2}\\.\\d{3}', // time
    '\\[[A-Za-z]+\\]', // tag
    '#\\d+', // http id
    '[←✕→] ?\\d{3}\\b', // arrow + status
    '[←✕→]', // arrow
    '\\b(?:GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS)\\b',
    '\\d+ms\\b',
    'https?://[^\\s"\']+',
    '\\b(?:WARN|ERROR)\\b',
  ].join('|'),
  'g',
)

function statusClass(code: number): string {
  if (code >= 500) return 'st-5'
  if (code >= 400) return 'st-4'
  if (code >= 300) return 'st-3'
  return 'st-2'
}

function classify(m: string): Piece[] {
  if (/^\d{2}:\d{2}:\d{2}\.\d{3}$/.test(m)) return [{ text: m, cls: 'tk-time' }]
  if (m === 'LOG') return [{ text: m, cls: 'tk-lvl-log' }]
  if (m === 'ERROR') return [{ text: m, cls: 'tk-lvl-error' }]
  if (m === 'WARN') return [{ text: m, cls: 'tk-lvl-warn' }]
  if (m[0] === '[') return [{ text: m, cls: `tk-tag tag-${m.slice(1, -1)}` }]
  if (m[0] === '#') return [{ text: m, cls: 'tk-id' }]
  if (/^[←✕→]/.test(m)) {
    const arrow = m[0]
    const cls = arrow === '✕' ? 'tk-fail' : arrow === '←' ? 'tk-ok' : 'tk-req'
    const rest = m.slice(1)
    if (!rest.trim()) return [{ text: m, cls }]
    const code = Number(rest.trim())
    return [
      { text: arrow, cls },
      { text: rest, cls: arrow === '✕' ? 'tk-fail' : statusClass(code) },
    ]
  }
  if (/^[A-Z]+$/.test(m)) return [{ text: m, cls: 'tk-method' }]
  if (/ms$/.test(m)) return [{ text: m, cls: 'tk-dur' }]
  return [{ text: m, cls: 'tk-url' }]
}

export function highlight(line: string): Piece[] {
  const out: Piece[] = []
  let last = 0
  for (const match of line.matchAll(TOKEN)) {
    const i = match.index
    if (i > last) out.push({ text: line.slice(last, i), cls: '' })
    out.push(...classify(match[0]))
    last = i + match[0].length
  }
  if (last < line.length) out.push({ text: line.slice(last), cls: '' })
  return out
}
