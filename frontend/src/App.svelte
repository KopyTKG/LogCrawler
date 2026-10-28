<script lang="ts">
  import { onMount } from 'svelte'
  import { Events } from '@wailsio/runtime'
  import TreeList from './lib/TreeList.svelte'
  import Detail from './lib/Detail.svelte'
  import InfoPanel from './lib/InfoPanel.svelte'
  import Timeline from './lib/Timeline.svelte'
  import ComparePanel from './lib/ComparePanel.svelte'
  import { viewer } from './lib/viewer.svelte'

  let showInfo = $state(false)
  let searchBox = $state<HTMLInputElement>()

  onMount(() => {
    viewer.init()
    const offDrop = Events.On('file-dropped', (e: { data: string }) => viewer.openPath(e.data))
    const offChanged = Events.On('file-changed', () => (viewer.changedOnDisk = true))
    return () => {
      offDrop()
      offChanged()
    }
  })

  // Search as the user types, after a short pause.
  $effect(() => {
    const q = { ...viewer.query }
    void q
    const t = setTimeout(() => viewer.search(), 200)
    return () => clearTimeout(t)
  })

  function onWindowKey(e: KeyboardEvent) {
    if ((e.ctrlKey || e.metaKey) && e.key === 'o') {
      e.preventDefault()
      viewer.pickAndOpen()
    } else if ((e.ctrlKey || e.metaKey) && e.key === 'f') {
      e.preventDefault()
      searchBox?.focus()
      searchBox?.select()
    } else if ((e.ctrlKey || e.metaKey) && e.key === 'r') {
      e.preventDefault()
      viewer.reload()
    } else if (e.key === 'F8') {
      e.preventDefault()
      viewer.stepFailure(e.shiftKey ? -1 : 1)
    } else if (e.key === 'F3') {
      e.preventDefault()
      viewer.stepHit(e.shiftKey ? -1 : 1)
    }
  }

  const problemCount = $derived(viewer.file?.problems?.length ?? 0)
</script>

<svelte:window onkeydown={onWindowKey} />

<main data-file-drop-target>
  <header>
    <button onclick={() => viewer.pickAndOpen()} title="Ctrl+O">Open…</button>
    {#if viewer.file}
      <span class="name" title={viewer.file.path}>{viewer.file.name}</span>
      <div class="seg" role="group" aria-label="View">
        <button class:on={viewer.view === 'screens'} onclick={() => viewer.setView('screens')}>Screens</button
        >
        <button class:on={viewer.view === 'requests'} onclick={() => viewer.setView('requests')}
          >Requests</button
        >
      </div>
      <input
        type="search"
        placeholder="Search (Ctrl+F)"
        bind:this={searchBox}
        bind:value={viewer.query.text}
        onkeydown={(e) => {
          if (e.key === 'Enter') viewer.stepHit(e.shiftKey ? -1 : 1)
        }}
      />
      <select bind:value={viewer.query.tag} aria-label="Tag">
        <option value="">All tags</option>
        {#each viewer.file.tags as t (t)}<option value={t}>{t}</option>{/each}
      </select>
      <label class="check"><input type="checkbox" bind:checked={viewer.query.problems} /> Problems only</label
      >
      <label class="check"><input type="checkbox" bind:checked={viewer.query.slow} /> Slow calls</label>
      {#if viewer.result}
        <span class="dim">
          {viewer.hitList.length}{viewer.result.truncated ? '+' : ''} match{viewer.hitList.length === 1
            ? ''
            : 'es'}
        </span>
        <button onclick={() => viewer.stepHit(-1)} disabled={!viewer.hitList.length} title="Shift+F3"
          >↑</button
        >
        <button onclick={() => viewer.stepHit(1)} disabled={!viewer.hitList.length} title="F3">↓</button>
      {/if}
      <span class="spacer"></span>
      <button onclick={() => viewer.expandAll()} title="Expand every session, screen and group"
        >Expand all</button
      >
      <button onclick={() => viewer.collapseAll()}>Collapse</button>
      <button
        onclick={() => viewer.exportMatches()}
        title="Save the matches, or the failures when nothing is searched">Export…</button
      >
      <button onclick={() => viewer.compare()} title="Compare with another log">Compare…</button>
      <button onclick={() => viewer.reload()} title="Ctrl+R">Reload</button>
      <button class:on={showInfo} onclick={() => (showInfo = !showInfo)}>
        Info{#if problemCount > 0}&nbsp;<span class="warnmark">⚠ {problemCount}</span>{/if}
      </button>
    {/if}
  </header>

  {#if viewer.changedOnDisk}
    <div class="changed" role="status">
      This file changed on disk.
      <button onclick={() => viewer.reload()}>Reload</button>
    </div>
  {/if}

  {#if viewer.error}
    <div class="banner" role="alert">
      {viewer.error}
      <button onclick={() => (viewer.error = '')}>Dismiss</button>
    </div>
  {/if}

  {#if viewer.file && viewer.failures}
    {@const f = viewer.failures}
    {@const total = f.items?.length ?? 0}
    <div class="failbar" class:clean={total === 0} role="status">
      {#if total === 0}
        <span>✓ No failed calls or errors in this log</span>
      {:else}
        <span class="summary">
          {#if f.failed}<strong>{f.failed}</strong> failed call{f.failed === 1 ? '' : 's'}{/if}
          {#if f.errors}<strong>{f.errors}</strong> error{f.errors === 1 ? '' : 's'}{/if}
          {#if f.unanswered}<strong>{f.unanswered}</strong> unanswered{/if}
        </span>
        <button onclick={() => viewer.stepFailure(-1)} title="Shift+F8">◀ Previous</button>
        <button onclick={() => viewer.stepFailure(1)} title="F8">Next failure ▶</button>
      {/if}
      {#if f.slow}<span class="slowmark">⏱ {f.slow} slow call{f.slow === 1 ? '' : 's'}</span>{/if}
    </div>
  {/if}

  {#if viewer.file}<Timeline />{/if}

  {#if !viewer.file}
    <div class="empty">
      {#if viewer.loading}Reading…{:else}
        <p>Drop a log file here, or</p>
        <button onclick={() => viewer.pickAndOpen()}>Open a log…</button>
        {#if viewer.recent.length}
          <h4>Recent</h4>
          <ul class="recent">
            {#each viewer.recent as p (p)}
              <li>
                <button class="linklike" onclick={() => viewer.openPath(p)} title={p}
                  >{p.split(/[\\/]/).pop()}</button
                >
              </li>
            {/each}
          </ul>
        {/if}
      {/if}
    </div>
  {:else}
    <div class="body">
      <section class="tree"><TreeList /></section>
      {#if showInfo}<aside><InfoPanel file={viewer.file} version={viewer.version} /></aside>{/if}
    </div>
    <section class="detail">
      <div class="detailbar">
        <span class="dim"
          >{viewer.selected === null ? 'Select a row to see its log text' : 'Log text of the selection'}</span
        >
        <span class="actions">
          <label class="check" title="Show the events that happened just before the selection">
            <input
              type="checkbox"
              checked={viewer.showContext}
              onchange={(e) => viewer.setContext(e.currentTarget.checked)}
            />
            Show {viewer.contextCount} events before
          </label>
          <button
            disabled={viewer.selected === null}
            onclick={() => viewer.copyReport()}
            title="Redacted summary with the app version and the events before it">Copy bug report</button
          >
          <button disabled={!viewer.detail} onclick={() => viewer.copy(viewer.detail)}>Copy</button>
        </span>
      </div>
      <div class="detailbody">
        <Detail text={viewer.detail} node={viewer.selectedNode} file={viewer.file} before={viewer.before} />
      </div>
    </section>
  {/if}

  {#if viewer.comparison}<ComparePanel
      c={viewer.comparison}
      onclose={() => (viewer.comparison = null)}
    />{/if}

  {#if viewer.notice}<div class="toast" role="status">{viewer.notice}</div>{/if}
</main>

<style>
  main {
    height: 100%;
    display: flex;
    flex-direction: column;
    position: relative;
  }
  :global(.file-drop-target-active) {
    outline: 3px dashed var(--accent);
    outline-offset: -6px;
  }
  header {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 6px 8px;
    border-bottom: 1px solid var(--border);
    background: var(--bg-alt);
    flex-wrap: wrap;
  }
  .name {
    font-weight: 600;
    max-width: 22ch;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .seg {
    display: inline-flex;
  }
  .seg button:first-child {
    border-radius: 5px 0 0 5px;
  }
  .seg button:last-child {
    border-radius: 0 5px 5px 0;
    margin-left: -1px;
  }
  .check {
    display: inline-flex;
    gap: 4px;
    align-items: center;
  }
  .dim {
    color: var(--fg-dim);
  }
  .spacer {
    flex: 1;
  }
  .warnmark {
    color: var(--warn);
  }
  .failbar {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 4px 10px;
    background: var(--error-bg);
    color: var(--error);
    border-bottom: 1px solid var(--border);
  }
  .failbar.clean {
    background: var(--bg-alt);
    color: var(--c-ok);
  }
  .failbar .summary {
    flex: 1;
  }
  .failbar .summary strong {
    margin-left: 8px;
  }
  .failbar .summary strong:first-child {
    margin-left: 0;
  }
  .changed {
    background: var(--warn-bg);
    color: var(--warn);
    padding: 4px 10px;
    display: flex;
    gap: 10px;
    align-items: center;
  }
  .slowmark {
    color: var(--warn);
  }
  .actions {
    display: flex;
    gap: 8px;
    align-items: center;
  }
  .recent {
    list-style: none;
    padding: 0;
    margin: 0;
  }
  .linklike {
    background: none;
    border: 0;
    color: var(--accent);
    padding: 2px;
  }
  .banner {
    background: var(--error-bg);
    color: var(--error);
    padding: 6px 10px;
    display: flex;
    justify-content: space-between;
    align-items: center;
  }
  .empty {
    flex: 1;
    display: grid;
    place-content: center;
    text-align: center;
    color: var(--fg-dim);
  }
  .body {
    flex: 1;
    min-height: 0;
    display: flex;
  }
  .tree {
    flex: 1;
    min-width: 0;
  }
  aside {
    width: 340px;
    border-left: 1px solid var(--border);
    overflow: auto;
    background: var(--bg-alt);
  }
  .detail {
    height: 38%;
    min-height: 140px;
    border-top: 1px solid var(--border);
    display: flex;
    flex-direction: column;
  }
  .detailbar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 4px 8px;
    background: var(--bg-alt);
  }
  .detailbody {
    flex: 1;
    padding: 6px 10px;
    overflow: auto;
  }
  .toast {
    position: absolute;
    bottom: 16px;
    left: 50%;
    transform: translateX(-50%);
    background: var(--fg);
    color: var(--bg);
    padding: 6px 14px;
    border-radius: 6px;
  }
</style>
