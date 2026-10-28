<script lang="ts">
  import { viewer } from './viewer.svelte'

  const items = $derived(viewer.failures?.items ?? [])
  const total = $derived(Math.max(1, viewer.file?.lines ?? 1))
  const at = $derived(viewer.selectedNode?.line ?? 0)
</script>

{#if items.length}
  <div class="strip" aria-label="Where the failures are in the file">
    {#each items as f (f.id)}
      <button
        class="tick"
        class:current={f.line === at}
        style:left="{(f.line / total) * 100}%"
        title="line {f.line}"
        aria-label="Failure at line {f.line}"
        onclick={() => viewer.reveal(f.id)}
      ></button>
    {/each}
    {#if at > 0}<span class="cursor" style:left="{(at / total) * 100}%"></span>{/if}
  </div>
{/if}

<style>
  .strip {
    position: relative;
    height: 14px;
    margin: 4px 10px;
    background: var(--bg-alt);
    border: 1px solid var(--border);
    border-radius: 3px;
  }
  .tick {
    position: absolute;
    top: 0;
    bottom: 0;
    width: 5px;
    margin-left: -2px;
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
    top: -2px;
    bottom: -2px;
    width: 2px;
    background: var(--accent);
    pointer-events: none;
  }
</style>
