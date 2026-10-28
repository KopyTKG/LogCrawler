import { SvelteMap, SvelteSet } from 'svelte/reactivity'
import { LogService } from '../../bindings/logcrawler'
import type { FileInfo } from '../../bindings/logcrawler'
import type {
  Comparison,
  Failures,
  NodeInfo,
  Query,
  Result,
} from '../../bindings/logcrawler/internal/logtree/models'

export type View = 'screens' | 'requests'

export interface Row {
  node: NodeInfo
  depth: number
  open: boolean
  hit: boolean
}

const ROOT = -1

function emptyQuery(): Query {
  return { text: '', tag: '', problems: false, slow: false }
}

function message(e: unknown): string {
  return e instanceof Error ? e.message : String(e)
}

/** Everything the window shows: the opened log, which nodes are expanded, the search, the selection. */
class Viewer {
  file = $state<FileInfo | null>(null)
  view = $state<View>('screens')
  loading = $state(false)
  error = $state('')
  notice = $state('')

  query = $state<Query>(emptyQuery())
  result = $state<Result | null>(null)
  hitIndex = $state(-1)

  selected = $state<number | null>(null)
  selectedNode = $state<NodeInfo | null>(null)
  detail = $state('')
  /** Bumped whenever the list should scroll the selected row into view. */
  revealTick = $state(0)

  failures = $state<Failures | null>(null)
  version = $state('')
  recent = $state<string[]>([])
  /** The open file changed on disk after it was read. */
  changedOnDisk = $state(false)
  comparison = $state<Comparison | null>(null)

  /** Show the events before the selected one in the detail pane. */
  showContext = $state(false)
  contextCount = 15
  before = $state('')

  children = new SvelteMap<number, NodeInfo[]>()
  expanded = new SvelteSet<number>()

  private searchToken = 0
  private detailToken = 0
  private noticeTimer: ReturnType<typeof setTimeout> | undefined

  visible = $derived(this.result ? new Set(this.result.visible ?? []) : null)
  hits = $derived(new Set(this.result?.hits ?? []))
  hitList = $derived(this.result?.hits ?? [])

  /** The expanded tree flattened into the rows that are on screen, in order. */
  rows = $derived.by(() => {
    const out: Row[] = []
    const roots = this.children.get(ROOT)
    if (!roots) return out
    const vis = this.visible
    const hits = this.hits
    const walk = (list: NodeInfo[], depth: number, filter: boolean) => {
      for (const node of list) {
        if (filter && vis && !vis.has(node.id)) continue
        const open = node.hasChildren && this.expanded.has(node.id)
        out.push({ node, depth, open, hit: hits.has(node.id) })
        const kids = open ? this.children.get(node.id) : undefined
        // A hit shows all of its own lines; only the path to a hit is filtered.
        if (kids) walk(kids, depth + 1, filter && !hits.has(node.id))
      }
    }
    walk(roots, 0, true)
    return out
  })

  async openPath(path: string) {
    if (!path) return
    this.loading = true
    this.error = ''
    try {
      const info = await LogService.Open(path)
      this.file = info
      this.changedOnDisk = false
      this.query = emptyQuery()
      this.result = null
      await this.reset()
      this.recent = (await LogService.Recent()) ?? []
    } catch (e) {
      this.error = message(e)
    } finally {
      this.loading = false
    }
  }

  async pickAndOpen() {
    try {
      await this.openPath(await LogService.PickFile())
    } catch (e) {
      this.error = message(e)
    }
  }

  async init() {
    try {
      this.version = await LogService.Version()
      const saved = await LogService.SavedView()
      if (saved === 'screens' || saved === 'requests') this.view = saved
      this.recent = (await LogService.Recent()) ?? []
      await this.openPath(await LogService.InitialFile())
    } catch (e) {
      this.error = message(e)
    }
  }

  async reload() {
    if (!this.file) return
    await this.openPath(this.file.path)
    this.changedOnDisk = false
  }

  /** Drop all tree state and load the roots of the current view. */
  private async reset() {
    this.children.clear()
    this.expanded.clear()
    this.selected = null
    this.selectedNode = null
    this.detail = ''
    this.hitIndex = -1
    const roots = (await LogService.Children(this.view, ROOT)) ?? []
    this.children.set(ROOT, roots)
    for (const r of roots) this.expanded.add(r.id)
    await this.loadMissing(roots.map((r) => r.id))
    this.failures = await LogService.Failures(this.view)
    if (this.searching) await this.search()
  }

  get searching(): boolean {
    const q = this.query
    return q.text.trim() !== '' || q.tag !== '' || q.problems || q.slow
  }

  async setView(v: View) {
    if (v === this.view) return
    this.view = v
    this.result = null
    void LogService.SaveView(v)
    if (this.file) await this.reset()
  }

  private async loadMissing(ids: number[]) {
    const need = ids.filter((id) => !this.children.has(id))
    if (need.length === 0) return
    const got = await LogService.ChildrenBatch(this.view, need)
    for (const [id, kids] of Object.entries(got ?? {})) this.children.set(Number(id), kids ?? [])
  }

  async toggle(node: NodeInfo) {
    if (!node.hasChildren) return
    if (this.expanded.has(node.id)) {
      this.expanded.delete(node.id)
      return
    }
    try {
      await this.loadMissing([node.id])
      this.expanded.add(node.id)
    } catch (e) {
      this.error = message(e)
    }
  }

  async search() {
    const token = ++this.searchToken
    this.hitIndex = -1
    if (!this.searching) {
      this.result = null
      return
    }
    try {
      const res = await LogService.Search(this.view, { ...this.query })
      if (token !== this.searchToken || !res) return
      const hits = new Set(res.hits ?? [])
      const ancestors = (res.visible ?? []).filter((id) => !hits.has(id))
      await this.loadMissing(ancestors)
      if (token !== this.searchToken) return
      for (const id of ancestors) this.expanded.add(id)
      this.result = res
    } catch (e) {
      this.error = message(e)
    }
  }

  async select(node: NodeInfo) {
    this.selected = node.id
    this.selectedNode = node
    void this.loadBefore(node)
    const token = ++this.detailToken
    try {
      const text = await LogService.NodeText(this.view, node.id)
      if (token === this.detailToken) this.detail = text
    } catch (e) {
      this.error = message(e)
    }
  }

  async loadBefore(node: NodeInfo) {
    this.before = ''
    if (!this.showContext || (node.kind !== 'event' && node.kind !== 'exchange' && node.kind !== 'line'))
      return
    try {
      const text = await LogService.Before(this.view, node.id, this.contextCount)
      if (this.selected === node.id) this.before = text
    } catch (e) {
      this.error = message(e)
    }
  }

  async setContext(on: boolean) {
    this.showContext = on
    if (this.selectedNode) await this.loadBefore(this.selectedNode)
  }

  /** The redacted bug-report text for the selection, on the clipboard. */
  async copyReport() {
    if (this.selected === null) return
    try {
      await this.copy(await LogService.Report(this.view, this.selected, this.contextCount))
    } catch (e) {
      this.error = message(e)
    }
  }

  /** Expand every session, screen and group; calls stay collapsed (they could be thousands). */
  async expandAll() {
    try {
      let level = this.children.get(ROOT) ?? []
      while (level.length) {
        const open = level.filter((n) => n.hasChildren && n.kind !== 'exchange' && n.kind !== 'event')
        await this.loadMissing(open.map((n) => n.id))
        for (const n of open) this.expanded.add(n.id)
        level = open.flatMap((n) => this.children.get(n.id) ?? [])
      }
    } catch (e) {
      this.error = message(e)
    }
  }

  collapseAll() {
    this.expanded.clear()
    for (const r of this.children.get(ROOT) ?? []) this.expanded.add(r.id)
  }

  async exportMatches() {
    try {
      const path = await LogService.Export(this.view, { ...this.query })
      if (path) this.say('Exported')
    } catch (e) {
      this.error = message(e)
    }
  }

  async compare() {
    try {
      const c = await LogService.PickCompare()
      if (c) this.comparison = c
    } catch (e) {
      this.error = message(e)
    }
  }

  async stepHit(delta: 1 | -1) {
    const list = this.hitList
    if (list.length === 0) return
    this.hitIndex = (this.hitIndex + delta + list.length) % list.length
    const id = list[this.hitIndex]
    const row = this.rows.find((r) => r.node.id === id)
    if (row) {
      await this.select(row.node)
      this.revealTick++
    }
  }

  /** Select the next (or previous) failure after the current selection, wrapping around. */
  async stepFailure(delta: 1 | -1) {
    const items = this.failures?.items ?? []
    if (items.length === 0) return
    const at = this.selectedNode?.line ?? 0
    let target = delta === 1 ? items.find((f) => f.line > at) : items.findLast((f) => f.line < at)
    target ??= delta === 1 ? items[0] : items[items.length - 1]
    await this.reveal(target.id)
  }

  /** Expand down to a node, select it and scroll it into view. */
  async reveal(id: number) {
    try {
      if (this.searching && !this.visible?.has(id)) {
        this.query = emptyQuery()
        this.result = null
      }
      const path = (await LogService.Path(this.view, id)) ?? []
      await this.loadMissing(path)
      for (const p of path) this.expanded.add(p)
      const parent = path[path.length - 1]
      const node = this.children.get(parent)?.find((n) => n.id === id)
      if (!node) return
      await this.select(node)
      this.revealTick++
    } catch (e) {
      this.error = message(e)
    }
  }

  async copy(text: string) {
    let ok = false
    try {
      ok = await LogService.Copy(text)
    } catch {
      /* fall back to the browser clipboard below */
    }
    if (!ok) {
      try {
        await navigator.clipboard.writeText(text)
        ok = true
      } catch {
        /* nothing left to try */
      }
    }
    this.say(ok ? 'Copied to clipboard' : 'Could not copy')
  }

  say(text: string) {
    this.notice = text
    clearTimeout(this.noticeTimer)
    this.noticeTimer = setTimeout(() => (this.notice = ''), 2000)
  }
}

export const viewer = new Viewer()
