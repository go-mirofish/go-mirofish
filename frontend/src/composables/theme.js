import { ref } from 'vue'

const STORAGE = 'go-mirofish-theme'

function readPreference() {
  if (typeof localStorage === 'undefined') return 'system'
  const v = localStorage.getItem(STORAGE)
  if (v === 'light' || v === 'dark' || v === 'system') return v
  return 'system'
}

function computeResolved(pref) {
  if (pref === 'light' || pref === 'dark') return pref
  if (typeof window === 'undefined' || !window.matchMedia) return 'light'
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

export const themePreference = ref('system')
export const themeResolved = ref('light')

function applyToDocument() {
  if (typeof document === 'undefined') return
  const r = themeResolved.value
  document.documentElement.setAttribute('data-theme', r)
  document.documentElement.style.colorScheme = r === 'dark' ? 'dark' : 'light'
}

function sync() {
  themePreference.value = readPreference()
  themeResolved.value = computeResolved(themePreference.value)
  applyToDocument()
}

/**
 * @param {'light' | 'dark' | 'system'} pref
 */
export function setThemePreference(pref) {
  if (pref !== 'light' && pref !== 'dark' && pref !== 'system') return
  if (typeof localStorage !== 'undefined') {
    localStorage.setItem(STORAGE, pref)
  }
  sync()
}

export function initTheme() {
  if (typeof window === 'undefined') return
  sync()
  const mq = window.matchMedia('(prefers-color-scheme: dark)')
  mq.addEventListener('change', () => {
    if (readPreference() === 'system') sync()
  })
}
