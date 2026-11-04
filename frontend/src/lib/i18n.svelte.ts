import en, { type Key } from './locales/en'
import cs from './locales/cs'

export type Locale = 'en' | 'cs'
export const LOCALES: Locale[] = ['cs', 'en']

const dictionaries: Record<Locale, Record<string, string>> = { en, cs }

function stored(name: string): string | null {
  try {
    return localStorage.getItem(name)
  } catch {
    return null // storage unavailable: nothing remembered
  }
}

function remember(name: string, value: string) {
  try {
    localStorage.setItem(name, value)
  } catch {
    // storage unavailable: the choice lasts until the app closes
  }
}

function initial(): Locale {
  const s = stored('locale')
  if (s === 'en' || s === 'cs') return s
  return navigator.language.toLowerCase().startsWith('cs') ? 'cs' : 'en'
}

class I18n {
  locale = $state<Locale>(initial())

  set(l: Locale) {
    this.locale = l
    document.documentElement.lang = l
    remember('locale', l)
  }

  /** The string for a key in the current language, with {name} placeholders filled from vars. */
  t(key: Key, vars: Record<string, string | number> = {}): string {
    const text = dictionaries[this.locale][key] ?? en[key]
    return text.replace(/\{(\w+)\}/g, (m, name) => (name in vars ? String(vars[name]) : m))
  }

  /** Like t, for a key that has .one / .few / .other forms, picked by the count n. */
  tn(key: string, n: number, vars: Record<string, string | number> = {}): string {
    const d = dictionaries[this.locale]
    const rule = new Intl.PluralRules(this.locale).select(n)
    const k = (`${key}.${rule}` in d ? `${key}.${rule}` : `${key}.other`) as Key
    return this.t(k, { n, ...vars })
  }
}

export const i18n = new I18n()
document.documentElement.lang = i18n.locale

export const t = (key: Key, vars?: Record<string, string | number>) => i18n.t(key, vars)
export const tn = (key: string, n: number, vars?: Record<string, string | number>) => i18n.tn(key, n, vars)
