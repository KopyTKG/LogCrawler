<script lang="ts">
  import { t } from './i18n.svelte'
  import { appSettings, MIN_SCALE, MAX_SCALE, STEP } from './settings.svelte'

  let { onclose }: { onclose: () => void } = $props()
</script>

<svelte:window onkeydown={(e) => e.key === 'Escape' && onclose()} />

<div class="scrim" role="presentation" onclick={onclose}>
  <div
    class="modal"
    role="dialog"
    aria-modal="true"
    aria-label={t('settings.title')}
    tabindex="-1"
    onclick={(e) => e.stopPropagation()}
    onkeydown={() => {}}
  >
    <header>
      <h2>{t('settings.title')}</h2>
      <button onclick={onclose}>{t('compare.close')}</button>
    </header>

    <label class="field" for="scale">
      <span>{t('settings.scale')}</span>
      <span class="dim">{t('settings.scaleHint')}</span>
    </label>
    <div class="scale">
      <button
        onclick={() => appSettings.set(appSettings.scale - STEP)}
        disabled={appSettings.scale <= MIN_SCALE}
        aria-label={t('settings.smaller')}>−</button
      >
      <input
        id="scale"
        type="range"
        min={MIN_SCALE}
        max={MAX_SCALE}
        step={STEP}
        value={appSettings.scale}
        oninput={(e) => appSettings.set(Number(e.currentTarget.value))}
      />
      <button
        onclick={() => appSettings.set(appSettings.scale + STEP)}
        disabled={appSettings.scale >= MAX_SCALE}
        aria-label={t('settings.larger')}>+</button
      >
      <output>{appSettings.scale} %</output>
      <button onclick={() => appSettings.set(100)} disabled={appSettings.scale === 100}
        >{t('settings.reset')}</button
      >
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
    width: min(32rem, 92vw);
  }
  header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }
  h2 {
    margin: 0.5rem 0;
  }
  .field {
    display: flex;
    flex-direction: column;
    margin-top: 1rem;
    font-weight: 600;
  }
  .field .dim {
    font-weight: 400;
  }
  .dim {
    color: var(--fg-dim);
  }
  .scale {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin-top: 0.5rem;
  }
  input[type='range'] {
    flex: 1;
  }
  output {
    min-width: 3.5rem;
    text-align: right;
  }
</style>
