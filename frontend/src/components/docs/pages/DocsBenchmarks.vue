<template>
  <div class="bench">
    <div id="doc-bench-hero" class="hero">
      <div class="hero-left">
        <div class="hero-title">{{ $t('docs.bench.title') }}</div>
        <div class="hero-sub">{{ $t('docs.bench.subtitle') }}</div>
        <div class="pill-row">
          <StatusPill :label="$t('docs.bench.backendBoot')" :status="status.backendBoot" />
          <StatusPill :label="$t('docs.bench.gatewayBoot')" :status="status.gatewayBoot" />
          <StatusPill :label="$t('docs.bench.stress')" :status="status.stress" />
          <StatusPill :label="$t('docs.bench.e2e')" :status="status.e2e" />
        </div>
      </div>

      <div class="hero-right">
        <BenchmarkRunCombobox
          v-if="runs.length"
          v-model="selectedKey"
          :options="runOptions"
        />
        <div v-else class="no-runs">{{ $t('docs.bench.noRuns') }}</div>
      </div>
    </div>

    <div id="doc-bench-metrics" class="metrics">
      <MetricCard :label="$t('docs.bench.p50')" :value="fmtMs(metrics.p50)" />
      <MetricCard :label="$t('docs.bench.p95')" :value="fmtMs(metrics.p95)" />
      <MetricCard :label="$t('docs.bench.max')" :value="fmtMs(metrics.max)" />
      <MetricCard :label="$t('docs.bench.successRate')" :value="metrics.successRate" />
      <MetricCard :label="$t('docs.bench.requests')" :value="metrics.requests" />
      <MetricCard :label="$t('docs.bench.throughput')" :value="metrics.throughput" :hint="$t('docs.bench.throughputHint')" />
    </div>

    <div id="doc-bench-charts" class="chart-wrap">
      <div class="chart-wrap-title">{{ $t('docs.bench.tocCharts') }}</div>
      <BenchCharts :data="data" />
    </div>

    <div id="doc-bench-grid" class="grid">
      <section class="panel">
        <div class="panel-title">{{ $t('docs.bench.timeline') }}</div>
        <div class="steps">
          <div v-for="s in steps" :key="s.key" class="step" :class="`step--${s.status}`">
            <div class="step-dot"></div>
            <div class="step-body">
              <div class="step-head">
                <div class="step-name">{{ s.label }}</div>
                <div class="step-meta mono" v-if="s.meta">{{ s.meta }}</div>
              </div>
              <div class="step-sub" v-if="s.sub">{{ s.sub }}</div>
            </div>
          </div>
        </div>
      </section>

      <section class="panel">
        <div class="panel-title">{{ $t('docs.bench.artifacts') }}</div>
        <div class="art">
          <button class="copy" type="button" @click="copyJson">{{ $t('docs.bench.copyJson') }}</button>
          <details class="details">
            <summary>{{ $t('docs.bench.viewJson') }}</summary>
            <pre class="json"><code>{{ pretty }}</code></pre>
          </details>
          <div class="small">{{ $t('docs.bench.artifactsHint') }}</div>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { parseBundledBenchmarkPath } from '../lib/benchmarkRunLabel.js'
import MetricCard from '../ui/MetricCard.vue'
import StatusPill from '../ui/StatusPill.vue'
import BenchCharts from '../ui/BenchCharts.vue'
import BenchmarkRunCombobox from '../ui/BenchmarkRunCombobox.vue'

const { t } = useI18n()

// Committed short-name JSON under docs/bundled-benchmarks/ (see docs/bundled-benchmarks/README.md).
const modules = import.meta.glob('../../../../../docs/bundled-benchmarks/**/*.json', { eager: true })

function normalizeData(mod) {
  if (mod && typeof mod === 'object' && 'default' in mod) return mod.default
  return mod
}

function toBundledRel(vitePath) {
  return (vitePath || '')
    .replace(/.*[/\\]docs[/\\]bundled-benchmarks[/\\]/, 'bundled-benchmarks/')
    .replace(/\\/g, '/')
}

const runs = Object.entries(modules)
  .map(([path, mod]) => {
    const data = normalizeData(mod) ?? mod
    const rel = toBundledRel(path)
    const { display, search } = parseBundledBenchmarkPath(rel)
    return { key: path, label: display, search, data }
  })
  .filter((r) => r.data && typeof r.data === 'object')
  .sort((a, b) => a.label.localeCompare(b.label, undefined, { numeric: true }))

const runOptions = computed(() => runs.map((r) => ({ key: r.key, label: r.label, search: r.search })))

const selectedKey = ref(runs[0]?.key || '')
const data = computed(() => {
  const row = runs.find((r) => r.key === selectedKey.value)
  return row?.data && typeof row.data === 'object' ? row.data : {}
})

const pretty = computed(() => JSON.stringify(data.value, null, 2))

function evalPass(v) {
  if (v === true || v === 'ok' || v === 'pass' || v === 'PASS') return 'pass'
  if (v === 'partial' || v === 'PARTIAL' || v === 'warn') return 'partial'
  if (v === false || v === 'fail' || v === 'FAIL') return 'fail'
  return 'pending'
}

const status = computed(() => {
  const d = data.value || {}
  if (d.proof) {
    return {
      backendBoot: evalPass(d.proof.backend_boot),
      gatewayBoot: evalPass(d.proof.gateway_boot),
      stress: evalPass(d.proof.bounded_stress),
      e2e: evalPass(d.proof.full_flow),
    }
  }
  if (d.backend_health || d.gateway_health) {
    return {
      backendBoot: d.backend_health?.status === 'ok' ? 'pass' : 'pending',
      gatewayBoot: d.gateway_health?.status === 'ok' ? 'pass' : 'pending',
      stress:
        d.stress?.request_count > 0 && d.stress?.failure_count === 0
          ? 'pass'
          : d.stress?.request_count
            ? 'partial'
            : 'pending',
      e2e: d.benchmark?.ok === true ? 'pass' : d.benchmark?.ok === false ? 'fail' : 'pending',
    }
  }
  if (d.evaluation?.status) {
    return {
      backendBoot: 'pending',
      gatewayBoot: 'pending',
      stress: 'pending',
      e2e: evalPass(d.evaluation.status),
    }
  }
  return { backendBoot: 'pending', gatewayBoot: 'pending', stress: 'pending', e2e: 'pending' }
})

const metrics = computed(() => {
  const d = data.value || {}
  const latency = d?.latency || d?.latency_ms || d?.summary?.latency || d?.stress?.latency_ms || {}
  const p50 = latency?.p50 ?? d?.p50 ?? d?.latency_p50
  const p95 = latency?.p95 ?? d?.p95 ?? d?.latency_p95
  const max = latency?.max ?? d?.max ?? d?.latency_max
  const req = d?.requests ?? d?.summary?.requests ?? d?.stress?.request_count
  const ok = d?.successes ?? d?.summary?.successes ?? d?.stress?.success_count
  let rate = '-'
  if (req != null && ok != null) rate = `${Math.round((ok / req) * 100)}%`
  else if (d.evaluation?.status) rate = d.evaluation.status === 'pass' ? '100%' : d.evaluation.status
  let tput = d.throughput_rps ?? d.summary?.throughput_rps
  if (tput == null && d.example_key != null && d.interaction_count != null && d.total_runtime_ms) {
    tput = d.interaction_count / (d.total_runtime_ms / 1000)
  }
  const throughput =
    tput != null && typeof tput === 'number' && !Number.isNaN(tput) ? tput.toFixed(1) : '-'

  return {
    p50: typeof p50 === 'number' ? p50 : d.startup_latency_ms != null ? d.startup_latency_ms : null,
    p95: typeof p95 === 'number' ? p95 : d.total_runtime_ms != null ? d.total_runtime_ms : null,
    max: typeof max === 'number' ? max : null,
    successRate: rate,
    requests: req != null ? String(req) : d.agent_count != null ? String(d.agent_count) : '-',
    throughput,
  }
})

const steps = computed(() => {
  const d = data.value || {}
  if (d.benchmark?.summary) {
    const s = d.benchmark.summary
    if (s.project_id || s.graph_id) {
      return [
        { key: 'proj', label: t('docs.bench.phaseProject'), status: s.project_id ? 'pass' : 'pending', meta: null, sub: s.project_id },
        { key: 'g', label: t('docs.bench.phaseGraph'), status: s.graph_id ? 'pass' : 'pending', meta: null, sub: s.graph_id },
        { key: 'sim', label: t('docs.bench.phaseSim'), status: s.simulation_id ? 'pass' : 'pending', meta: null, sub: s.simulation_id },
        { key: 'rep', label: t('docs.bench.phaseReport'), status: s.report_id ? 'pass' : 'pending', meta: null, sub: s.report_id },
      ]
    }
  }
  const phases = d?.phases || d?.timeline || {}
  const mk = (key, label) => {
    const p = phases?.[key] || {}
    const st = (p.status || p.result || '').toString().toLowerCase()
    const status = st.includes('pass') || st === 'ok' ? 'pass' : st.includes('partial') ? 'partial' : st.includes('fail') ? 'fail' : 'pending'
    const meta = p.duration_ms ? `${p.duration_ms}ms` : p.duration ? String(p.duration) : null
    const sub = p.id || p.graph_id || p.project_id || p.simulation_id || p.report_id || p.evidence || null
    return { key, label, status, meta, sub }
  }
  if (d.example_key) {
    return [
      {
        key: 'ex',
        label: t('docs.bench.phaseExample', { name: d.title || d.example_key }),
        status: d.evaluation?.status === 'pass' ? 'pass' : d.evaluation?.status === 'fail' ? 'fail' : 'pending',
        meta: d.total_runtime_ms != null ? `${d.total_runtime_ms.toFixed(2)}ms` : null,
        sub: t('docs.bench.profileMeta', { profile: d.profile, agents: d.agent_count }),
      },
    ]
  }
  return [
    mk('ontology', 'Ontology'),
    mk('graph', 'Graph'),
    mk('simulation', 'Simulation'),
    mk('report', 'Report'),
  ]
})

function fmtMs(v) {
  if (typeof v !== 'number') return '-'
  return `${v.toFixed(2)}ms`
}

async function copyJson() {
  try {
    await navigator.clipboard.writeText(pretty.value)
  } catch {
    // ignore
  }
}
</script>

<style scoped>
.bench {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
#doc-bench-hero,
#doc-bench-metrics,
#doc-bench-charts,
#doc-bench-grid { scroll-margin-top: 88px; }

.chart-wrap {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
}
.chart-wrap-title {
  font-size: 11px;
  font-weight: 900;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--doc-muted);
}

.hero {
  border: 1px solid var(--doc-border);
  background: var(--doc-surface);
  border-radius: var(--doc-radius);
  padding: clamp(14px, 2vw, 18px) clamp(14px, 2.2vw, 18px) clamp(12px, 1.5vw, 16px);
  box-shadow: var(--doc-shadow-soft);
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 14px;
  flex-wrap: wrap;
  min-width: 0;
  width: 100%;
  box-sizing: border-box;
}

.hero-left {
  flex: 1 1 min(100%, 360px);
  min-width: 0;
}
.hero-title {
  font-size: clamp(1.125rem, 2.2vw, 1.375rem);
  font-weight: 900;
  letter-spacing: -0.02em;
}
.hero-sub {
  margin-top: 6px;
  color: var(--doc-muted);
  line-height: 1.6;
  max-width: 64ch;
}
.pill-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 12px;
}

.hero-right {
  flex: 1 1 240px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  align-items: stretch;
}
.no-runs {
  font-size: 12px;
  line-height: 1.5;
  color: var(--doc-muted);
  max-width: 40ch;
}

/* 6 metrics: 3×2 (row 1: P50–MAX, row 2: success–throughput) */
.metrics {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  grid-auto-rows: minmax(min-content, auto);
  gap: 12px;
  width: 100%;
  min-width: 0;
  align-items: stretch;
}

.grid {
  display: grid;
  grid-template-columns: 1.2fr 0.8fr;
  gap: 10px;
}

.panel {
  border: 1px solid var(--doc-border);
  background: var(--doc-surface);
  border-radius: var(--doc-radius);
  padding: 16px;
  box-shadow: var(--doc-shadow-soft);
  min-width: 0;
}
.panel-title {
  font-size: 11px;
  font-weight: 900;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--doc-muted);
  margin-bottom: 12px;
}

.steps {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.step {
  display: flex;
  gap: 10px;
  align-items: flex-start;
  padding: 10px 10px;
  border: 1px solid var(--doc-border);
  border-radius: 0;
  background: var(--doc-upload-surface);
}
.step-dot {
  width: 10px;
  height: 10px;
  border-radius: 0;
  margin-top: 4px;
  background: var(--doc-dashed);
  flex-shrink: 0;
}
.step--pass .step-dot { background: var(--doc-badge-ok-fg); }
.step--partial .step-dot { background: #f59e0b; }
.step--fail .step-dot { background: #dc2626; }
.step-head {
  display: flex;
  gap: 10px;
  align-items: baseline;
  justify-content: space-between;
}
.step-name { font-weight: 900; color: var(--doc-text); }
.step-meta { color: var(--doc-muted); font-size: 12px; }
.step-sub { margin-top: 6px; color: var(--doc-muted); font-size: 12px; line-height: 1.5; }

.mono { font-family: var(--doc-font-mono); }

.copy {
  border: 1px solid var(--doc-border);
  background: var(--doc-surface);
  color: var(--doc-text);
  border-radius: 0;
  padding: 10px 12px;
  cursor: pointer;
  font-weight: 900;
}
.copy:hover {
  border-color: color-mix(in srgb, var(--doc-accent) 30%, var(--doc-border));
}
.details {
  margin-top: 12px;
  border: 1px solid var(--doc-border);
  border-radius: 0;
  padding: 10px 12px;
}
.json {
  margin: 10px 0 0;
  padding: 10px;
  background: var(--doc-code-bg);
  border-radius: 0;
  overflow: auto;
  max-height: 360px;
}
.small { margin-top: 10px; color: var(--doc-muted); font-size: 12px; line-height: 1.5; }

@media (max-width: 1100px) {
  .grid { grid-template-columns: 1fr; }
}

/* Mid widths: 2×3 so cards stay readable */
@media (max-width: 680px) {
  .metrics {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 700px) {
  .hero {
    flex-direction: column;
    align-items: stretch;
  }
  .hero-left,
  .hero-right {
    flex: 1 1 auto;
    width: 100%;
  }
}

@media (max-width: 400px) {
  .metrics {
    grid-template-columns: 1fr;
  }
}
</style>

