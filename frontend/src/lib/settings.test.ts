import { afterEach, describe, expect, it } from 'vitest'
import { appSettings } from './settings.svelte'

afterEach(() => appSettings.set(100))

describe('UI scale', () => {
  it('sets the root font size and remembers the choice', () => {
    appSettings.set(150)
    expect(document.documentElement.style.fontSize).toBe('24px')
    expect(localStorage.getItem('scale')).toBe('150')
  })

  it('stays within the limits and on the step', () => {
    appSettings.set(500)
    expect(appSettings.scale).toBe(200)
    appSettings.set(10)
    expect(appSettings.scale).toBe(80)
    appSettings.set(113)
    expect(appSettings.scale).toBe(110)
  })
})
