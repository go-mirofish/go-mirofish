/** Display titles for first segment of bundled benchmark filenames (`scenario__profile__variant.json` — `__` avoids ambiguity with hyphens in scenario slugs). */
export const BENCHMARK_SCENARIO_LABELS = {
  'defi-stress': 'DeFi sentiment',
  'urban-planning': 'Urban planning',
  'literary-sim': 'Literary / Lost',
  'product-launch': 'Product launch',
  'incident-drill': 'Zero-day incident',
}

/**
 * @param {string} variant segment after profile (e.g. `latest`, `20260424T170506Z`)
 */
export function formatVariantForDisplay(variant) {
  if (variant === 'latest') return 'Latest'
  const m = /^(\d{4})(\d{2})(\d{2})T(\d{2})(\d{2})(\d{2})Z$/.exec(variant)
  if (m) return `Run ${m[1]}-${m[2]}-${m[3]} ${m[4]}:${m[5]} UTC`
  return variant
}

/**
 * @param {string} rel path under docs/ like `bundled-benchmarks/defi-stress__small__latest.json`
 * @returns {{ title: string, profile: string, variant: string, display: string, search: string }}
 */
export function parseBundledBenchmarkPath(rel) {
  const base = (rel || '').split('/').pop() || ''
  const stem = base.replace(/\.json$/i, '')
  const parts = stem.split('__')
  if (parts.length < 3) {
    return {
      title: stem,
      profile: '',
      variant: '',
      display: stem,
      search: `${stem} ${base}`.toLowerCase(),
    }
  }
  const [scenario, profile, ...rest] = parts
  const variant = rest.join('--') || 'latest'
  const title = BENCHMARK_SCENARIO_LABELS[scenario] || scenario.replace(/-/g, ' ')
  const variantLabel = formatVariantForDisplay(variant)
  const display = `${title} · ${profile} · ${variantLabel}`
  return {
    title,
    profile,
    variant: variantLabel,
    display,
    search: [display, stem, scenario, profile, variant, base].join(' ').toLowerCase(),
  }
}
