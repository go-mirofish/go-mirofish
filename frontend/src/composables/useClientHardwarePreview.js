import { ref, onMounted, onUnmounted } from 'vue'

/**
 * Lightweight client capability hints for run preview (browser-only, not the gateway host).
 */
export function useClientHardwarePreview() {
  const cpuCores = ref(null)
  const deviceMemGib = ref(null)
  const displayShort = ref('')
  const timeZone = ref('')

  function refresh() {
    if (typeof navigator === 'undefined') return
    cpuCores.value = navigator.hardwareConcurrency ?? null
    deviceMemGib.value =
      typeof navigator !== 'undefined' && navigator.deviceMemory != null ? navigator.deviceMemory : null
    try {
      timeZone.value = Intl.DateTimeFormat().resolvedOptions().timeZone || ''
    } catch {
      timeZone.value = ''
    }
    if (typeof window !== 'undefined' && window.screen) {
      displayShort.value = `${window.screen.width}×${window.screen.height}`
    }
  }

  onMounted(() => {
    refresh()
    if (typeof window !== 'undefined') {
      window.addEventListener('resize', refresh)
    }
  })
  onUnmounted(() => {
    if (typeof window !== 'undefined') {
      window.removeEventListener('resize', refresh)
    }
  })

  return { cpuCores, deviceMemGib, displayShort, timeZone, refresh }
}
