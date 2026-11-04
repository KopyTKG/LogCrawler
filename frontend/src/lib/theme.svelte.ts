export type Theme = 'system' | 'light' | 'dark'
export const THEMES: Theme[] = ['system', 'light', 'dark']

function initial(): Theme {
  try {
    const s = localStorage.getItem('theme')
    if (s === 'light' || s === 'dark') return s
  } catch {
    // storage unavailable: follow the system
  }
  return 'system'
}

class ThemeState {
  theme = $state<Theme>(initial())

  constructor() {
    this.apply()
  }

  set(t: Theme) {
    this.theme = t
    this.apply()
    try {
      localStorage.setItem('theme', t)
    } catch {
      // storage unavailable: the choice lasts until the app closes
    }
  }

  // app.css reads data-theme; with no attribute the system's light/dark setting decides.
  private apply() {
    const root = document.documentElement
    if (this.theme === 'system') root.removeAttribute('data-theme')
    else root.setAttribute('data-theme', this.theme)
  }
}

export const themeState = new ThemeState()
