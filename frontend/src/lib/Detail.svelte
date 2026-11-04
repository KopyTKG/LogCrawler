<script lang="ts">
  import { t } from './i18n.svelte'
  import type { FileInfo } from '../../bindings/logcrawler'
  import type { NodeInfo } from '../../bindings/logcrawler/internal/logtree/models'
  import Highlight from './Highlight.svelte'
  import { entries, jsonBlock } from './json'

  let {
    text,
    node,
    file,
    before = '',
  }: { text: string; node: NodeInfo | null; file: FileInfo; before?: string } = $props()

  const MAX = 300

  // Parsed is what people read; raw is the log text untouched, for copying or checking the parser.
  let tab = $state<'parsed' | 'raw'>('parsed')

  const prior = $derived(before ? entries(before) : [])
  const all = $derived(entries(text))
  const shown = $derived(all.slice(0, MAX).map((e) => ({ ...e, block: jsonBlock(e.head) })))
</script>

<div class="detail">
  <div class="tabs" role="tablist" aria-label={t('detail.tabs')}>
    <button
      role="tab"
      aria-selected={tab === 'parsed'}
      class:on={tab === 'parsed'}
      onclick={() => (tab = 'parsed')}>{t('detail.parsed')}</button
    >
    <button role="tab" aria-selected={tab === 'raw'} class:on={tab === 'raw'} onclick={() => (tab = 'raw')}
      >{t('detail.raw')}</button
    >
  </div>
  {#if tab === 'parsed' && node?.kind === 'session' && file.header?.length}
    <div class="block">
      <div class="cap">{t('detail.header')}{file.title ? ` — ${file.title}` : ''}</div>
      <dl>
        {#each file.header ?? [] as f, i (i)}
          <dt>{f.key}</dt>
          <dd>{f.value}</dd>
        {/each}
      </dl>
    </div>
  {/if}
  {#if prior.length}
    <div class="cap before">{t('detail.before', { n: prior.length })}</div>
    <div class="prior">
      {#each prior as e, i (i)}
        <div class="line"><Highlight text={e.head} /></div>
        {#each e.rest as r, j (j)}<div class="line cont">{r}</div>{/each}
      {/each}
    </div>
    <div class="cap">{t('detail.selected')}</div>
  {/if}
  {#each shown as e, i (i)}
    <div class="entry">
      <div class="line">
        {#if tab === 'raw'}{e.indent}{/if}<Highlight
          text={tab === 'parsed' && e.block ? e.head.slice(0, e.block.start).trimEnd() : e.head}
        />
      </div>
      {#each e.rest as r, j (j)}<div class="line cont">{r}</div>{/each}
      {#if tab === 'parsed' && e.block}
        <div class="block">
          <div class="cap">{t(e.block.label)}</div>
          {#if e.block.json}<pre class="json">{#each e.block.json as p, k (k)}{#if p.cls}<span class={p.cls}
                    >{p.text}</span
                  >{:else}{p.text}{/if}{/each}</pre>{/if}
          {#if e.block.table}
            <table>
              <thead
                ><tr
                  >{#each e.block.table.head as h (h)}<th>{t(h)}</th>{/each}</tr
                ></thead
              >
              <tbody>
                {#each e.block.table.rows as row, r (r)}
                  <tr
                    >{#each row as cell, c (c)}<td>{cell === 'yes' ? t('yes') : cell}</td>{/each}</tr
                  >
                {/each}
              </tbody>
            </table>
          {/if}
          {#if e.block.stack}
            <div class="cap sub">{t('detail.stack')}</div>
            <pre class="stack">{e.block.stack}</pre>
          {/if}
        </div>
      {/if}
    </div>
  {/each}
  {#if all.length > MAX}
    <div class="more">{t('detail.more', { n: all.length - MAX })}</div>
  {/if}
</div>

<style>
  .tabs {
    display: flex;
    gap: 2px;
    margin-bottom: 0.375rem;
    border-bottom: 1px solid var(--border);
    font: 600 0.75rem var(--sans);
  }
  .tabs button {
    border: 0;
    border-bottom: 2px solid transparent;
    border-radius: 0;
    background: none;
    color: var(--fg-dim);
    padding: 3px 0.625rem;
  }
  .tabs button.on {
    color: var(--fg);
    border-bottom-color: var(--accent);
  }
  .detail {
    font: 0.75rem/1.45 var(--mono);
    user-select: text;
  }
  .line {
    white-space: pre-wrap;
    word-break: break-all;
  }
  .cont {
    color: var(--fg-dim);
  }
  .entry {
    margin-bottom: 0.25rem;
  }
  .block {
    margin: 0.25rem 0 0.5rem 0.75rem;
    border-left: 3px solid var(--border);
    padding-left: 0.625rem;
  }
  .cap {
    font: 600 0.6875rem var(--sans);
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--fg-dim);
    margin-bottom: 2px;
  }
  .prior {
    opacity: 0.75;
    border-left: 3px solid var(--border);
    padding-left: 0.625rem;
    margin-bottom: 0.375rem;
  }
  .cap.before {
    margin-top: 0;
  }
  .cap.sub {
    margin-top: 0.375rem;
  }
  pre {
    margin: 0;
    white-space: pre-wrap;
    word-break: break-all;
  }
  .stack {
    color: var(--fg-dim);
    font-size: 0.6875rem;
  }
  table {
    border-collapse: collapse;
  }
  th {
    font: 600 0.6875rem var(--sans);
    color: var(--fg-dim);
    text-align: left;
  }
  th,
  td {
    padding: 1px 0.875rem 1px 0;
  }
  dl {
    display: grid;
    grid-template-columns: max-content 1fr;
    gap: 1px 0.75rem;
    margin: 0;
  }
  dt {
    color: var(--fg-dim);
  }
  dd {
    margin: 0;
  }
  .more {
    color: var(--fg-dim);
    padding: 0.25rem 0;
  }
</style>
