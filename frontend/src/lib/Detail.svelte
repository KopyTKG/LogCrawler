<script lang="ts">
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

  const prior = $derived(before ? entries(before) : [])
  const all = $derived(entries(text))
  const shown = $derived(all.slice(0, MAX).map((e) => ({ ...e, block: jsonBlock(e.head) })))
</script>

<div class="detail">
  {#if node?.kind === 'session' && file.header?.length}
    <div class="block">
      <div class="cap">Log header{file.title ? ` — ${file.title}` : ''}</div>
      <dl>
        {#each file.header ?? [] as f, i (i)}
          <dt>{f.key}</dt>
          <dd>{f.value}</dd>
        {/each}
      </dl>
    </div>
  {/if}
  {#if prior.length}
    <div class="cap before">Before — {prior.length} earlier events</div>
    <div class="prior">
      {#each prior as e, i (i)}
        <div class="line"><Highlight text={e.head} /></div>
        {#each e.rest as r, j (j)}<div class="line cont">{r}</div>{/each}
      {/each}
    </div>
    <div class="cap">Selected</div>
  {/if}
  {#each shown as e, i (i)}
    <div class="entry">
      <div class="line"><Highlight text={e.head} /></div>
      {#each e.rest as r, j (j)}<div class="line cont">{r}</div>{/each}
      {#if e.block}
        <div class="block">
          <div class="cap">{e.block.label}</div>
          {#if e.block.json}<pre class="json">{#each e.block.json as p, k (k)}{#if p.cls}<span class={p.cls}
                    >{p.text}</span
                  >{:else}{p.text}{/if}{/each}</pre>{/if}
          {#if e.block.stack}
            <div class="cap sub">Stack</div>
            <pre class="stack">{e.block.stack}</pre>
          {/if}
        </div>
      {/if}
    </div>
  {/each}
  {#if all.length > MAX}
    <div class="more">… {all.length - MAX} more lines — Copy has them all</div>
  {/if}
</div>

<style>
  .detail {
    font: 12px/1.45 var(--mono);
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
    margin-bottom: 4px;
  }
  .block {
    margin: 4px 0 8px 12px;
    border-left: 3px solid var(--border);
    padding-left: 10px;
  }
  .cap {
    font: 600 11px var(--sans);
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--fg-dim);
    margin-bottom: 2px;
  }
  .prior {
    opacity: 0.75;
    border-left: 3px solid var(--border);
    padding-left: 10px;
    margin-bottom: 6px;
  }
  .cap.before {
    margin-top: 0;
  }
  .cap.sub {
    margin-top: 6px;
  }
  pre {
    margin: 0;
    white-space: pre-wrap;
    word-break: break-all;
  }
  .stack {
    color: var(--fg-dim);
    font-size: 11px;
  }
  dl {
    display: grid;
    grid-template-columns: max-content 1fr;
    gap: 1px 12px;
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
    padding: 4px 0;
  }
</style>
