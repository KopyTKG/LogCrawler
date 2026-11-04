export const MIN_SCALE = 80
export const MAX_SCALE = 200
export const STEP = 10
const BASE_PX = 16

function initial(): number {
  try {
    const n = Number(localStorage.getItem('scale'))
    if (n >= MIN_SCALE && n <= MAX_SCALE) return n
  } catch {
    // storage unavailable: use 100 %
  }
  return 100
}

/**
 * UI scale in percent. Everything is sized in rem, so changing the root font size scales the whole
 * window, layout included, without the coordinate quirks of CSS zoom.
 */
class AppSettings {
  scale = $state(initial())

  constructor() {
    this.apply()
  }

  set(percent: number) {
    this.scale = Math.min(MAX_SCALE, Math.max(MIN_SCALE, Math.round(percent / STEP) * STEP))
    this.apply()
    try {
      localStorage.setItem('scale', String(this.scale))
    } catch {
      // storage unavailable: the choice lasts until the app closes
    }
  }

  private apply() {
    document.documentElement.style.fontSize = `${(BASE_PX * this.scale) / 100}px`
  }
}

export const appSettings = new AppSettings()
