<script lang="ts">
  import { t } from './i18n.svelte'
  import { viewer } from './viewer.svelte'

  const items = $derived(viewer.failures?.items ?? [])
  const total = $derived(Math.max(1, viewer.file?.lines ?? 1))
  const at = $derived(viewer.selectedNode?.line ?? 0)
</script>

{#if items.length}
  <div class="strip" aria-label={t('timeline.label')}>
    {#each items as f (f.id)}
      <button
        class="tick"
        class:current={f.line === at}
        style:top="{(f.line / total) * 100}%"
        title={t('timeline.line', { n: f.line })}
        aria-label={t('timeline.failure', { n: f.line })}
        onclick={() => viewer.reveal(f.id)}
      ></button>
    {/each}
    {#if at > 0}<span class="cursor" style:top="{(at / total) * 100}%"></span>{/if}
  </div>
{/if}

<style>
  .strip {
    position: relative;
    flex: none;
    width: 0.875rem;
    margin: 0.25rem;
    background: var(--bg-alt);
    border: 1px solid var(--border);
    border-radius: 3px;
  }
  .tick {
    position: absolute;
    left: 0;
    right: 0;
    height: 0.3125rem;
    margin-top: -2px;
    padding: 0;
    border: 0;
    border-radius: 1px;
    background: var(--error);
    opacity: 0.8;
  }
  .tick:hover,
  .tick.current {
    opacity: 1;
    outline: 2px solid var(--fg);
    z-index: 1;
  }
  .cursor {
    position: absolute;
    left: -2px;
    right: -2px;
    height: 2px;
    background: var(--accent);
    pointer-events: none;
  }
</style>
