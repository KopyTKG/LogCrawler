<script lang="ts">
  import { tick } from 'svelte'
  import { viewer } from './viewer.svelte'
  import Highlight from './Highlight.svelte'
  import { tagHue } from './highlight'
  import { Severity } from '../../bindings/logcrawler/internal/logtree/models'

  const ROW = 24

  function fmtMs(ms: number): string {
    if (ms < 1000) return `${ms} ms`
    if (ms < 60000) return `${(ms / 1000).toFixed(1)} s`
    return `${Math.floor(ms / 60000)} min ${Math.round((ms % 60000) / 1000)} s`
  }
  const OVERSCAN = 8

  let scroller: HTMLDivElement
  let scrollTop = $state(0)
  let height = $state(400)

  const rows = $derived(viewer.rows)
  const start = $derived(Math.max(0, Math.floor(scrollTop / ROW) - OVERSCAN))
  const end = $derived(Math.min(rows.length, Math.ceil((scrollTop + height) / ROW) + OVERSCAN))
  const slice = $derived(rows.slice(start, end))

  function scrollToIndex(i: number) {
    if (i < 0 || !scroller) return
    const top = i * ROW
    if (top < scroller.scrollTop) scroller.scrollTop = top
    else if (top + ROW > scroller.scrollTop + scroller.clientHeight)
      scroller.scrollTop = top + ROW - scroller.clientHeight
  }

  $effect(() => {
    void viewer.revealTick
    tick().then(() => scrollToIndex(rows.findIndex((r) => r.node.id === viewer.selected)))
  })

  function onKey(e: KeyboardEvent) {
    const i = rows.findIndex((r) => r.node.id === viewer.selected)
    const row = rows[i]
    let target = -1
    switch (e.key) {
      case 'ArrowDown':
        target = Math.min(rows.length - 1, i + 1)
        break
      case 'ArrowUp':
        target = Math.max(0, i - 1)
        break
      case 'Home':
        target = 0
        break
      case 'End':
        target = rows.length - 1
        break
      case 'PageDown':
        target = Math.min(rows.length - 1, i + Math.floor(height / ROW))
        break
      case 'PageUp':
        target = Math.max(0, i - Math.floor(height / ROW))
        break
      case 'ArrowRight':
        if (row && row.node.hasChildren && !row.open) viewer.toggle(row.node)
        else target = Math.min(rows.length - 1, i + 1)
        break
      case 'ArrowLeft':
        if (row?.open) viewer.toggle(row.node)
        else if (row) {
          for (let j = i - 1; j >= 0; j--)
            if (rows[j].depth < row.depth) {
              target = j
              break
            }
        }
        break
      case 'Enter':
        if (row) viewer.toggle(row.node)
        break
      default:
        return
    }
    e.preventDefault()
    if (target >= 0 && rows[target]) {
      viewer.select(rows[target].node)
      scrollToIndex(target)
    }
  }
</script>

<div
  class="scroller"
  bind:this={scroller}
  bind:clientHeight={height}
  onscroll={() => (scrollTop = scroller.scrollTop)}
  onkeydown={onKey}
  role="tree"
  tabindex="0"
>
  <div class="space" style:height="{rows.length * ROW}px">
    {#each slice as row, k (row.node.id)}
      {@const n = row.node}
      <!-- svelte-ignore a11y_click_events_have_key_events -->
      <div
        class="row"
        class:selected={viewer.selected === n.id}
        class:hit={row.hit}
        class:error={n.sev === Severity.SevError}
        class:warn={n.sev === Severity.SevWarn}
        data-kind={n.kind}
        class:container={n.kind === 'session' || n.kind === 'screen' || n.kind === 'group'}
        style:top="{(start + k) * ROW}px"
        style:padding-left="{8 + row.depth * 16}px"
        role="treeitem"
        tabindex="-1"
        aria-selected={viewer.selected === n.id}
        aria-expanded={n.hasChildren ? row.open : undefined}
        onclick={() => viewer.select(n)}
        ondblclick={() => viewer.toggle(n)}
      >
        <button
          class="twist"
          tabindex="-1"
          aria-label={row.open ? 'Collapse' : 'Expand'}
          style:visibility={n.hasChildren ? 'visible' : 'hidden'}
          onclick={(e) => {
            e.stopPropagation()
            viewer.toggle(n)
          }}>{row.open ? '▾' : '▸'}</button
        >
        {#if n.gapMs >= 5000}<span class="gap" title="Time since the previous event">+{fmtMs(n.gapMs)}</span
          >{/if}
        {#if n.time}<span class="time" title={n.date}>{n.time}</span>{/if}
        {#if n.tag}<span class="tag tk-tag" style:--h={tagHue(n.tag)}>{n.tag}</span>{/if}
        <span class="label" title={n.label}>
          {#if n.kind === 'session' || n.kind === 'screen' || n.kind === 'group'}{n.label}{:else}<Highlight
              text={n.tag ? n.label.replace(/^\[[^\]]*\] ?/, '') : n.label}
            />{/if}
        </span>
        {#if n.slow}<span class="badge slow" title="Slow call">⏱ {fmtMs(n.durationMs)}</span>{/if}
        {#if n.link >= 0}
          <button
            class="linkbtn"
            tabindex="-1"
            title="This error repeats a failed call — go to the call"
            onclick={(e) => {
              e.stopPropagation()
              viewer.reveal(n.link)
            }}>↳ call</button
          >
        {/if}
        {#if n.errors > 0}<span class="badge err">{n.errors}</span>{/if}
        {#if n.warnings > 0}<span class="badge wrn">{n.warnings}</span>{/if}
      </div>
    {/each}
  </div>
</div>

<style>
  .scroller {
    height: 100%;
    overflow: auto;
    position: relative;
  }
  .space {
    position: relative;
    min-width: 100%;
  }
  .row {
    position: absolute;
    left: 0;
    right: 0;
    height: 24px;
    display: flex;
    align-items: center;
    gap: 6px;
    white-space: nowrap;
    cursor: default;
    font-family: var(--mono);
    font-size: 12px;
    padding-right: 8px;
  }
  .row:hover {
    background: var(--bg-hover);
  }
  .row.container {
    font-family: var(--sans);
    font-size: 13px;
    font-weight: 600;
  }
  .row[data-kind='screen'] .label {
    color: var(--accent);
  }
  .row.hit .label {
    background: var(--hit);
  }
  .row.error {
    background: var(--error-bg);
  }
  .row.warn {
    background: var(--warn-bg);
  }
  .row.selected {
    background: var(--bg-select) !important;
  }
  .twist {
    width: 16px;
    padding: 0;
    border: 0;
    background: none;
    color: var(--fg-dim);
    line-height: 1;
  }
  .time {
    color: var(--c-time);
  }
  .tag {
    background: var(--bg-alt);
    border: 1px solid var(--border);
    border-radius: 3px;
    padding: 0 4px;
    font-size: 11px;
  }
  .label {
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .badge {
    border-radius: 9px;
    padding: 0 6px;
    font: 600 11px/16px var(--sans);
  }
  .gap {
    color: var(--warn);
    font-size: 11px;
  }
  .badge.slow {
    background: var(--warn-bg);
    color: var(--warn);
  }
  .linkbtn {
    padding: 0 6px;
    font-size: 11px;
    line-height: 16px;
    border-radius: 9px;
  }
  .badge.err {
    background: var(--error-bg);
    color: var(--error);
  }
  .badge.wrn {
    background: var(--warn-bg);
    color: var(--warn);
  }
</style>
