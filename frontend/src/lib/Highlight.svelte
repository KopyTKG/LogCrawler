<script lang="ts">
  import { highlight, tagHue } from './highlight'

  let { text }: { text: string } = $props()

  const pieces = $derived(highlight(text))

  function hue(cls: string): string | undefined {
    const m = /tag-(\S+)$/.exec(cls)
    return m ? String(tagHue(m[1])) : undefined
  }
</script>

{#each pieces as p, i (i)}{#if p.cls}<span class={p.cls} style:--h={hue(p.cls)}>{p.text}</span
    >{:else}{p.text}{/if}{/each}
