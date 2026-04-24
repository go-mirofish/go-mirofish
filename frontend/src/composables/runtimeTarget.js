import { computed, ref } from 'vue'

const STORAGE_KEY = 'go-mirofish-api-base-url'
const LEGACY_BACKEND_BASES = new Set([
  'http://localhost:5001',
  'http://127.0.0.1:5001'
])
const DEFAULT_API_BASE = import.meta.env.VITE_API_BASE_URL || ''

const runtimeApiBaseUrl = ref(DEFAULT_API_BASE)

function normalizeApiBaseUrl(value) {
  const trimmed = String(value || '').trim()
  if (!trimmed) return DEFAULT_API_BASE
  if (LEGACY_BACKEND_BASES.has(trimmed.replace(/\/+$/, ''))) {
    return DEFAULT_API_BASE
  }
  return trimmed.replace(/\/+$/, '')
}

if (typeof window !== 'undefined') {
  runtimeApiBaseUrl.value = normalizeApiBaseUrl(
    window.localStorage.getItem(STORAGE_KEY) || DEFAULT_API_BASE
  )
}

export function getRuntimeApiBaseUrl() {
  return normalizeApiBaseUrl(runtimeApiBaseUrl.value)
}

export function setRuntimeApiBaseUrl(value) {
  const normalized = normalizeApiBaseUrl(value)
  runtimeApiBaseUrl.value = normalized
  if (typeof window !== 'undefined') {
    window.localStorage.setItem(STORAGE_KEY, normalized)
  }
  return normalized
}

export function resetRuntimeApiBaseUrl() {
  runtimeApiBaseUrl.value = DEFAULT_API_BASE
  if (typeof window !== 'undefined') {
    window.localStorage.removeItem(STORAGE_KEY)
  }
  return runtimeApiBaseUrl.value
}

export function useRuntimeApiBaseUrl() {
  return {
    apiBaseUrl: computed(() => runtimeApiBaseUrl.value),
    defaultApiBaseUrl: DEFAULT_API_BASE,
    normalizeApiBaseUrl,
    setRuntimeApiBaseUrl,
    resetRuntimeApiBaseUrl,
  }
}
