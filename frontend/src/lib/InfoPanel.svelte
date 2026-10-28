<script lang="ts">
  import type { FileInfo } from '../../bindings/logcrawler'

  let { file, version = '' }: { file: FileInfo; version?: string } = $props()
</script>

<div class="panel">
  {#if file.title}<h3>{file.title}</h3>{/if}
  <dl>
    {#each file.header as f (f.key)}
      <dt>{f.key}</dt>
      <dd>{f.value}</dd>
    {/each}
  </dl>

  <h3>Sessions</h3>
  {#each file.sessions as s, i (i)}
    <p>
      <strong>{s.label || 'session'}</strong><br />
      {#if s.startedAt}<span class="dim">{s.startedAt}</span><br />{/if}
      {s.events} events · {s.requests} requests · {s.failed} failed
    </p>
  {/each}

  <h3>Lines the parser could not place ({file.problems?.length ?? 0})</h3>
  {#if !file.problems?.length}
    <p class="dim">None.</p>
  {:else}
    <ul>
      {#each file.problems ?? [] as p (p.line)}
        <li><code>line {p.line}</code> {p.why}<br /><code class="text">{p.text}</code></li>
      {/each}
    </ul>
  {/if}
</div>

{#if version}<p class="dim ver">LogCrawler {version}</p>{/if}

<style>
  .panel {
    padding: 8px 12px;
    user-select: text;
  }
  h3 {
    margin: 12px 0 4px;
    font-size: 13px;
  }
  dl {
    display: grid;
    grid-template-columns: max-content 1fr;
    gap: 2px 10px;
    margin: 0;
  }
  dt {
    color: var(--fg-dim);
  }
  dd {
    margin: 0;
    word-break: break-all;
  }
  p {
    margin: 4px 0;
  }
  ul {
    padding-left: 16px;
    margin: 4px 0;
  }
  .dim {
    color: var(--fg-dim);
  }
  .text {
    word-break: break-all;
  }
</style>
