<script lang="ts">
  import { t } from './i18n.svelte'
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

  <h3>{t('info.sessions')}</h3>
  {#each file.sessions as s, i (i)}
    <p>
      <strong>{s.label || t('info.session')}</strong><br />
      {#if s.startedAt}<span class="dim">{s.startedAt}</span><br />{/if}
      {t('info.counts', { events: s.events, requests: s.requests, failed: s.failed })}
    </p>
  {/each}

  <h3>{t('info.problems', { n: file.problems?.length ?? 0 })}</h3>
  {#if !file.problems?.length}
    <p class="dim">{t('info.none')}</p>
  {:else}
    <ul>
      {#each file.problems ?? [] as p (p.line)}
        <li><code>{t('info.line', { n: p.line })}</code> {p.why}<br /><code class="text">{p.text}</code></li>
      {/each}
    </ul>
  {/if}
</div>

{#if version}<p class="dim ver">LogCrawler {version}</p>{/if}

<style>
  .panel {
    padding: 0.5rem 0.75rem;
    user-select: text;
  }
  h3 {
    margin: 0.75rem 0 0.25rem;
    font-size: 0.8125rem;
  }
  dl {
    display: grid;
    grid-template-columns: max-content 1fr;
    gap: 2px 0.625rem;
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
    margin: 0.25rem 0;
  }
  ul {
    padding-left: 1rem;
    margin: 0.25rem 0;
  }
  .dim {
    color: var(--fg-dim);
  }
  .text {
    word-break: break-all;
  }
</style>
