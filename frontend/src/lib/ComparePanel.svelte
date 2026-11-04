<script lang="ts">
  import { t } from './i18n.svelte'
  import type { Comparison } from '../../bindings/logcrawler/internal/logtree/models'

  let { c, onclose }: { c: Comparison; onclose: () => void } = $props()

  const rows = $derived([
    [t('compare.events'), c.a.events, c.b.events],
    [t('compare.requests'), c.a.requests, c.b.requests],
    [t('compare.failed'), c.a.failed, c.b.failed],
    [t('compare.errors'), c.a.errors, c.b.errors],
    [t('compare.unanswered'), c.a.unanswered, c.b.unanswered],
    [t('compare.slow'), c.a.slow, c.b.slow],
  ] as [string, number, number][])
</script>

<svelte:window onkeydown={(e) => e.key === 'Escape' && onclose()} />

<div class="scrim" role="presentation" onclick={onclose}>
  <div
    class="modal"
    role="dialog"
    aria-modal="true"
    aria-label={t('compare.dialog')}
    tabindex="-1"
    onclick={(e) => e.stopPropagation()}
    onkeydown={() => {}}
  >
    <header>
      <h2>{t('compare.heading')}</h2>
      <button onclick={onclose}>{t('compare.close')}</button>
    </header>
    <table>
      <thead>
        <tr><th></th><th>{c.a.name}</th><th>{c.b.name}</th></tr>
      </thead>
      <tbody>
        {#each rows as [label, a, b] (label)}
          <tr class:diff={a !== b}><td>{label}</td><td>{a}</td><td>{b}</td></tr>
        {/each}
      </tbody>
    </table>

    <h3>{t('compare.endpoints')}</h3>
    {#if !c.endpoints?.length}
      <p class="dim">{t('compare.noneEither')}</p>
    {:else}
      <table>
        <thead><tr><th>{t('compare.endpoint')}</th><th>{c.a.name}</th><th>{c.b.name}</th></tr></thead>
        <tbody>
          {#each c.endpoints as e (e.endpoint)}
            <tr class:diff={!e.a || !e.b}><td class="mono">{e.endpoint}</td><td>{e.a}</td><td>{e.b}</td></tr>
          {/each}
        </tbody>
      </table>
    {/if}

    <h3>{t('compare.onlyOne')}</h3>
    <div class="two">
      <div><strong>{c.a.name}</strong><br />{c.onlyA?.join(', ') || '—'}</div>
      <div><strong>{c.b.name}</strong><br />{c.onlyB?.join(', ') || '—'}</div>
    </div>
  </div>
</div>

<style>
  .scrim {
    position: fixed;
    inset: 0;
    background: rgb(0 0 0 / 0.45);
    display: grid;
    place-items: center;
    z-index: 10;
  }
  .modal {
    background: var(--bg);
    border: 1px solid var(--border);
    border-radius: 0.5rem;
    padding: 0.75rem 1.125rem 1.125rem;
    width: min(47.5rem, 92vw);
    max-height: 86vh;
    overflow: auto;
    user-select: text;
  }
  header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }
  h2,
  h3 {
    margin: 0.5rem 0;
  }
  h3 {
    font-size: 0.8125rem;
    margin-top: 1rem;
  }
  table {
    width: 100%;
    border-collapse: collapse;
  }
  th,
  td {
    text-align: left;
    padding: 3px 0.5rem;
    border-bottom: 1px solid var(--border);
  }
  th:not(:first-child),
  td:not(:first-child) {
    text-align: right;
    width: 18%;
  }
  tr.diff td {
    background: var(--warn-bg);
  }
  .mono {
    font-family: var(--mono);
    font-size: 0.75rem;
  }
  .dim {
    color: var(--fg-dim);
  }
  .two {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 0.75rem;
  }
</style>
