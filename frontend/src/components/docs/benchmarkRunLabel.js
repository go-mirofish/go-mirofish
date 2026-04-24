/** Human-readable titles for bundled benchmark filename slugs (before first `--`). */
const SCENARIO_LABELS = {
  'defi-stress': 'DeFi sentiment',
  'urban-planning': 'Urban planning',
  'literary-sim': 'Literary / Lost',
  'product-launch': 'Product launch',
  'incident-drill': 'Zero-day incident',
}

function basename(path) {
  const s = path.replace(/\\/g, '/')
  const i = s.lastIndexOf('/')
  return i >= 0 ? s.slice(i + 1) : s
}

/**
 * @param {string} filePath — e.g. `defi-stress--small--latest.json` or full Vite path
 * @returns {string} e.g. `DeFi sentiment · small · latest`
 */
export function formatBenchmarkRunLabel(filePath) {
  const base = basename(filePath).replace(/\.json$/i, '')
  const parts = base.split('--').filter(Boolean)
  if (parts.length < 3) {
    return base.replace(/--/g, ' · ')
  }
  const [scenario, profile, variant] = parts
  const title = SCENARIO_LABELS[scenario] || scenario.replace(/-/g, ' ')
  let v = variant
  if (variant === 'latest') {
    v = 'latest'
  } else if (/^\d{8}T\d{6}Z$/i.test(variant)) {
    const y = variant.slice(0, 4)
    const m = variant.slice(4, 6)
    const d = variant.slice(6, 8)
    v = `snapshot ${y}-${m}-${d}`
  }
  return `${title} · ${profile} · ${v}`
}

export function benchmarkRunSearchText(filePath, displayLabel) {
  const base = basename(filePath).replace(/\.json$/i, '')
  return `${displayLabel} ${base}`.toLowerCase()
}
