import { afterEach, describe, expect, it } from 'vitest'
import { i18n } from './i18n.svelte'
import { themeState } from './theme.svelte'

afterEach(() => {
  i18n.set('en')
  themeState.set('system')
})

describe('i18n', () => {
  it('fills placeholders', () => {
    expect(i18n.t('detail.context', { n: 15 })).toBe('Show 15 events before')
  })

  it('picks the Czech plural form', () => {
    i18n.set('cs')
    expect(i18n.tn('failbar.failed', 1)).toBe('1 selhané volání')
    expect(i18n.tn('failbar.failed', 3)).toBe('3 selhaná volání')
    expect(i18n.tn('failbar.failed', 5)).toBe('5 selhaných volání')
  })

  it('falls back to .other when a language has no such form', () => {
    expect(i18n.tn('failbar.errors', 3)).toBe('3 errors')
  })

  it('remembers the language and sets the page language', () => {
    i18n.set('cs')
    expect(document.documentElement.lang).toBe('cs')
    expect(localStorage.getItem('locale')).toBe('cs')
  })
})

describe('theme', () => {
  it('sets data-theme, and removes it for the system setting', () => {
    themeState.set('dark')
    expect(document.documentElement.dataset.theme).toBe('dark')
    themeState.set('system')
    expect(document.documentElement.dataset.theme).toBeUndefined()
  })
})
