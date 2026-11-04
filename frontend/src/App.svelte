<script lang="ts">
  import { onMount } from 'svelte'
  import { Events } from '@wailsio/runtime'
  import TreeList from './lib/TreeList.svelte'
  import Detail from './lib/Detail.svelte'
  import InfoPanel from './lib/InfoPanel.svelte'
  import Timeline from './lib/Timeline.svelte'
  import ComparePanel from './lib/ComparePanel.svelte'
  import { viewer } from './lib/viewer.svelte'
  import { i18n, t, tn, LOCALES } from './lib/i18n.svelte'
  import { themeState, THEMES } from './lib/theme.svelte'
  import { appSettings } from './lib/settings.svelte'
  import SettingsPanel from './lib/SettingsPanel.svelte'

  let showInfo = $state(false)
  let showSettings = $state(false)
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
    } else if ((e.ctrlKey || e.metaKey) && (e.key === '=' || e.key === '+')) {
      e.preventDefault()
      appSettings.set(appSettings.scale + 10)
    } else if ((e.ctrlKey || e.metaKey) && (e.key === '-' || e.key === '_')) {
      e.preventDefault()
      appSettings.set(appSettings.scale - 10)
    } else if ((e.ctrlKey || e.metaKey) && e.key === '0') {
      e.preventDefault()
      appSettings.set(100)
    } else if (e.key === 'F8') {
      e.preventDefault()
      viewer.stepFailure(e.shiftKey ? -1 : 1)
    } else if (e.key === 'F3') {
      e.preventDefault()
      viewer.stepHit(e.shiftKey ? -1 : 1)
    }
  }

  // Detail pane width, resizable with the splitter and remembered between runs.
  const MIN_PANE_REM = 35
  const remPx = () => parseFloat(getComputedStyle(document.documentElement).fontSize) || 16
  let bodyEl = $state<HTMLDivElement>()
  let dragging = $state(false)
  // Rolled out only for a selection; the bookmark tab and the button in the bar fold it away by hand.
  let folded = $state(false)
  const open = $derived(!folded && viewer.selected !== null)

  // Picking a row rolls the detail out again, even after it was folded by hand.
  $effect(() => {
    if (viewer.selected !== null) folded = false
  })
  let detailWidth = $state(storedWidth())

  function storedWidth(): number {
    try {
      const n = Number(localStorage.getItem('detailWidth'))
      if (n >= MIN_PANE_REM * remPx()) return n
    } catch {
      // storage unavailable: use the default
    }
    return MIN_PANE_REM * remPx()
  }

  function setWidth(w: number) {
    const min = MIN_PANE_REM * remPx()
    const max = Math.max(min, (bodyEl?.clientWidth ?? 1200) - min)
    detailWidth = Math.round(Math.min(max, Math.max(min, w)))
  }

  function saveWidth() {
    try {
      localStorage.setItem('detailWidth', String(detailWidth))
    } catch {
      // storage unavailable: width is not remembered
    }
  }

  function startDrag(e: PointerEvent) {
    dragging = true
    ;(e.currentTarget as HTMLElement).setPointerCapture(e.pointerId)
  }

  function onDrag(e: PointerEvent) {
    const detail = bodyEl?.querySelector('.detail')
    if (!dragging || !detail) return
    // the right edge of the detail pane does not move while dragging
    setWidth(detail.getBoundingClientRect().right - e.clientX)
  }

  function endDrag() {
    if (!dragging) return
    dragging = false
    saveWidth()
  }

  function onSplitterKey(e: KeyboardEvent) {
    if (e.key !== 'ArrowLeft' && e.key !== 'ArrowRight') return
    e.preventDefault()
    setWidth(detailWidth + (e.key === 'ArrowLeft' ? 24 : -24))
    saveWidth()
  }

  const problemCount = $derived(viewer.file?.problems?.length ?? 0)
</script>

<svelte:window onkeydown={onWindowKey} />

<main data-file-drop-target>
  <header>
    <button onclick={() => viewer.pickAndOpen()} title="Ctrl+O">{t('app.open')}</button>
    {#if viewer.file}
      <span class="name" title={viewer.file.path}>{viewer.file.name}</span>
      <div class="seg" role="group" aria-label={t('view.label')}>
        <button class:on={viewer.view === 'screens'} onclick={() => viewer.setView('screens')}
          >{t('view.screens')}</button
        >
        <button class:on={viewer.view === 'requests'} onclick={() => viewer.setView('requests')}
          >{t('view.requests')}</button
        >
      </div>
      <input
        type="search"
        placeholder={t('search.placeholder')}
        bind:this={searchBox}
        bind:value={viewer.query.text}
        onkeydown={(e) => {
          if (e.key === 'Enter') viewer.stepHit(e.shiftKey ? -1 : 1)
        }}
      />
      <select bind:value={viewer.query.tag} aria-label={t('tag.label')}>
        <option value="">{t('tag.all')}</option>
        {#each viewer.file.tags as t (t)}<option value={t}>{t}</option>{/each}
      </select>
      <label class="check"
        ><input type="checkbox" bind:checked={viewer.query.problems} /> {t('filter.problems')}</label
      >
      <label class="check"
        ><input type="checkbox" bind:checked={viewer.query.slow} /> {t('filter.slow')}</label
      >
      {#if viewer.result}
        <span class="dim">
          {tn('match', viewer.hitList.length, { plus: viewer.result.truncated ? '+' : '' })}
        </span>
        <button onclick={() => viewer.stepHit(-1)} disabled={!viewer.hitList.length} title="Shift+F3"
          >↑</button
        >
        <button onclick={() => viewer.stepHit(1)} disabled={!viewer.hitList.length} title="F3">↓</button>
      {/if}
      <span class="spacer"></span>
      <button onclick={() => viewer.expandAll()} title={t('tree.expandAllTitle')}
        >{t('tree.expandAll')}</button
      >
      <button onclick={() => viewer.collapseAll()}>{t('tree.collapse')}</button>
      <button onclick={() => viewer.exportMatches()} title={t('export.title')}>{t('export')}</button>
      <button onclick={() => viewer.compare()} title={t('compare.title')}>{t('compare')}</button>
      <button onclick={() => viewer.reload()} title="Ctrl+R">{t('reload')}</button>
      <button class:on={showInfo} onclick={() => (showInfo = !showInfo)}>
        {t('info')}{#if problemCount > 0}&nbsp;<span class="warnmark">⚠ {problemCount}</span>{/if}
      </button>
    {/if}
  </header>

  {#if viewer.changedOnDisk}
    <div class="changed" role="status">
      {t('app.changed')}
      <button onclick={() => viewer.reload()}>{t('reload')}</button>
    </div>
  {/if}

  {#if viewer.error}
    <div class="banner" role="alert">
      {viewer.error}
      <button onclick={() => (viewer.error = '')}>{t('app.dismiss')}</button>
    </div>
  {/if}

  {#if viewer.file && viewer.failures}
    {@const f = viewer.failures}
    {@const total = f.items?.length ?? 0}
    <div class="failbar" class:clean={total === 0} role="status">
      {#if total === 0}
        <span>{t('failbar.clean')}</span>
      {:else}
        <span class="summary">
          {#if f.failed}<strong>{tn('failbar.failed', f.failed)}</strong>{/if}
          {#if f.errors}<strong>{tn('failbar.errors', f.errors)}</strong>{/if}
          {#if f.unanswered}<strong>{t('failbar.unanswered', { n: f.unanswered })}</strong>{/if}
        </span>
        <button onclick={() => viewer.stepFailure(-1)} title="Shift+F8">{t('failbar.previous')}</button>
        <button onclick={() => viewer.stepFailure(1)} title="F8">{t('failbar.next')}</button>
      {/if}
      {#if f.slow}<span class="slowmark">⏱ {tn('failbar.slow', f.slow)}</span>{/if}
    </div>
  {/if}

  {#if !viewer.file}
    <div class="empty">
      {#if viewer.loading}{t('app.reading')}{:else}
        <p>{t('app.dropHere')}</p>
        <button onclick={() => viewer.pickAndOpen()}>{t('app.openLog')}</button>
        {#if viewer.recent.length}
          <h4>{t('app.recent')}</h4>
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
    <div class="body" bind:this={bodyEl}>
      <section class="tree"><TreeList /></section>
      {#if open}
        <!-- svelte-ignore a11y_no_noninteractive_tabindex, a11y_no_noninteractive_element_interactions -->
        <div
          class="splitter"
          class:dragging
          role="separator"
          aria-orientation="vertical"
          aria-label={t('detail.resize')}
          tabindex="0"
          onpointerdown={startDrag}
          onpointermove={onDrag}
          onpointerup={endDrag}
          onpointercancel={endDrag}
          onkeydown={onSplitterKey}
        ></div>
      {:else}
        <button
          class="bookmark"
          disabled={viewer.selected === null}
          onclick={() => (folded = false)}
          title={viewer.selected === null ? t('detail.none') : t('detail.bookmarkShow')}
          >{t('detail.bookmark')}</button
        >
      {/if}
      <section class="detail" class:open class:dragging inert={!open} style:--w="{detailWidth}px">
        <div class="inner">
          <div class="detailbar">
            <span class="dim">{viewer.selected === null ? t('detail.none') : t('detail.some')}</span>
            <span class="actions">
              <label class="check" title={t('detail.contextTitle')}>
                <input
                  type="checkbox"
                  checked={viewer.showContext}
                  onchange={(e) => viewer.setContext(e.currentTarget.checked)}
                />
                {t('detail.context', { n: viewer.contextCount })}
              </label>
              <button
                disabled={viewer.selected === null}
                onclick={() => viewer.copyReport()}
                title={t('detail.copyReportTitle')}>{t('detail.copyReport')}</button
              >
              <button disabled={!viewer.detail} onclick={() => viewer.copy(viewer.detail)}
                >{t('detail.copy')}</button
              >
              <button onclick={() => (folded = true)} title={t('detail.fold')} aria-label={t('detail.fold')}
                >›</button
              >
            </span>
          </div>
          <div class="detailbody">
            <Detail
              text={viewer.detail}
              node={viewer.selectedNode}
              file={viewer.file}
              before={viewer.before}
            />
          </div>
        </div>
      </section>
      <Timeline />
      {#if showInfo}<aside><InfoPanel file={viewer.file} version={viewer.version} /></aside>{/if}
    </div>
  {/if}

  {#if viewer.comparison}<ComparePanel
      c={viewer.comparison}
      onclose={() => (viewer.comparison = null)}
    />{/if}

  <footer>
    <span>LogCrawler {viewer.version || 'dev'} · {t('footer.license')}</span>
    <span class="prefs">
      <button onclick={() => (showSettings = true)}>⚙ {t('settings.open')}</button>
      <span class="seg" role="group" aria-label={t('footer.language')}>
        {#each LOCALES as l (l)}
          <button class:on={i18n.locale === l} onclick={() => i18n.set(l)}>{l.toUpperCase()}</button>
        {/each}
      </span>
      <span class="seg" role="group" aria-label={t('footer.theme')}>
        {#each THEMES as th (th)}
          <button class:on={themeState.theme === th} onclick={() => themeState.set(th)}
            >{t(`theme.${th}`)}</button
          >
        {/each}
      </span>
    </span>
  </footer>

  {#if showSettings}<SettingsPanel onclose={() => (showSettings = false)} />{/if}

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
    gap: 0.5rem;
    padding: 0.375rem 0.5rem;
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
    border-radius: 0.3125rem 0 0 0.3125rem;
  }
  .seg button:last-child {
    border-radius: 0 0.3125rem 0.3125rem 0;
    margin-left: -1px;
  }
  .check {
    display: inline-flex;
    gap: 0.25rem;
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
    gap: 0.625rem;
    padding: 0.25rem 0.625rem;
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
    margin-left: 0.5rem;
  }
  .failbar .summary strong:first-child {
    margin-left: 0;
  }
  .changed {
    background: var(--warn-bg);
    color: var(--warn);
    padding: 0.25rem 0.625rem;
    display: flex;
    gap: 0.625rem;
    align-items: center;
  }
  .slowmark {
    color: var(--warn);
  }
  .actions {
    display: flex;
    gap: 0.5rem;
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
    padding: 0.375rem 0.625rem;
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
    min-width: 35rem;
  }
  .splitter {
    width: 0.3125rem;
    margin: 0 -2px;
    z-index: 1;
    cursor: col-resize;
    background: var(--border);
    background-clip: content-box;
    padding: 0 2px;
    touch-action: none;
  }
  .splitter:hover,
  .splitter:focus-visible,
  .splitter.dragging {
    background-color: var(--accent);
  }
  aside {
    width: 21.25rem;
    border-left: 1px solid var(--border);
    overflow: auto;
    background: var(--bg-alt);
  }
  .detail {
    flex: none;
    width: 0;
    min-width: 0;
    overflow: hidden;
    transition:
      width 0.25s ease,
      min-width 0.25s ease;
  }
  .detail.open {
    width: var(--w);
    min-width: 35rem;
  }
  .detail.dragging {
    transition: none;
  }
  .inner {
    width: var(--w);
    min-width: 35rem;
    height: 100%;
    display: flex;
    flex-direction: column;
  }
  .bookmark {
    flex: none;
    align-self: flex-start;
    margin-top: 0.75rem;
    writing-mode: vertical-rl;
    padding: 0.625rem 0.25rem;
    border: 1px solid var(--border);
    border-right: 0;
    border-radius: 0.375rem 0 0 0.375rem;
    background: var(--bg-alt);
    color: var(--accent);
    font: 600 0.6875rem var(--sans);
    letter-spacing: 0.05em;
    text-transform: uppercase;
  }
  .bookmark:disabled {
    color: var(--fg-dim);
    opacity: 0.6;
  }
  @media (prefers-reduced-motion: reduce) {
    .detail {
      transition: none;
    }
  }
  .detailbar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 0.25rem 0.5rem;
    background: var(--bg-alt);
  }
  .detailbody {
    flex: 1;
    padding: 0.375rem 0.625rem;
    overflow: auto;
  }
  footer {
    flex: none;
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 2px 0.625rem;
    border-top: 1px solid var(--border);
    background: var(--bg-alt);
    color: var(--fg-dim);
    font-size: 0.6875rem;
  }
  .prefs {
    display: flex;
    gap: 0.625rem;
  }
  footer button {
    padding: 0 0.5rem;
    font-size: 0.6875rem;
  }
  .toast {
    position: absolute;
    bottom: 1rem;
    left: 50%;
    transform: translateX(-50%);
    background: var(--fg);
    color: var(--bg);
    padding: 0.375rem 0.875rem;
    border-radius: 0.375rem;
  }
</style>
