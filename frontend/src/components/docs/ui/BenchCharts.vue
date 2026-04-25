<template>
  <div ref="rootRef" class="bench-charts" v-if="hasAny">
    <div class="chart-block">
      <div class="chart-title">{{ $t('docs.bench.chartLatencyTitle') }}</div>
      <p v-if="!latencyBars.length" class="chart-empty">{{ $t('docs.bench.chartNoLatency') }}</p>
      <div v-show="latencyBars.length" ref="latEl" class="chart-svg" />
    </div>
    <div class="chart-block">
      <div class="chart-title">{{ $t('docs.bench.chartRequestsTitle') }}</div>
      <p v-if="!reqSegments.length" class="chart-empty">{{ $t('docs.bench.chartNoRequests') }}</p>
      <div v-show="reqSegments.length" ref="reqEl" class="chart-svg" />
    </div>
  </div>
  <p v-else class="chart-empty alone">{{ $t('docs.bench.chartsNoData') }}</p>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { axisBottom, axisLeft, max, scaleBand, scaleLinear, select } from 'd3'

const props = defineProps({
  data: { type: Object, default: () => ({}) },
})

const latEl = ref(null)
const reqEl = ref(null)
const rootRef = ref(null)
let ro

const latencyBars = computed(() => {
  const d = props.data || {}
  const L = d.stress?.latency_ms
  if (L && typeof L === 'object') {
    const out = []
    for (const key of ['min', 'p50', 'p95', 'max']) {
      const v = L[key]
      if (typeof v === 'number' && !Number.isNaN(v)) out.push({ key, label: key, value: v })
    }
    return out
  }
  const out = []
  if (typeof d.startup_latency_ms === 'number') out.push({ key: 'startup', label: 'startup', value: d.startup_latency_ms })
  if (typeof d.total_runtime_ms === 'number') out.push({ key: 'runtime', label: 'total', value: d.total_runtime_ms })
  return out
})

const reqSegments = computed(() => {
  const d = props.data || {}
  const s = d.stress
  if (s && typeof s.request_count === 'number') {
    const ok = s.success_count ?? 0
    const fail = s.failure_count ?? 0
    if (ok + fail > 0) {
      return [
        { key: 'ok', label: 'OK', value: ok, className: 'seg-ok' },
        { key: 'fail', label: 'Fail', value: fail, className: 'seg-fail' },
      ]
    }
  }
  return []
})

const hasAny = computed(() => latencyBars.value.length > 0 || reqSegments.value.length > 0)

function getCssVar(name) {
  if (typeof document === 'undefined') return '#6b7280'
  return getComputedStyle(document.documentElement).getPropertyValue(name).trim() || '#6b7280'
}

function drawLatency() {
  const el = latEl.value
  if (!el || !latencyBars.value.length) return
  const margin = { top: 8, right: 12, bottom: 36, left: 44 }
  const w0 = el.clientWidth || 400
  const h = 200
  const width = w0
  const innerW = width - margin.left - margin.right
  const innerH = h - margin.top - margin.bottom
  const data = latencyBars.value

  select(el).selectAll('*').remove()
  const svg = select(el)
    .append('svg')
    .attr('width', width)
    .attr('height', h)
    .attr('role', 'img')
  const g = svg.append('g').attr('transform', `translate(${margin.left},${margin.top})`)
  const x = scaleBand()
    .domain(data.map((d) => d.key))
    .range([0, innerW])
    .padding(0.2)
  const yMax = max(data, (d) => d.value) * 1.08
  const y = scaleLinear().domain([0, yMax || 1]).range([innerH, 0])
  const accent = getCssVar('--doc-accent') || '#00add8'
  g.selectAll('rect')
    .data(data)
    .enter()
    .append('rect')
    .attr('x', (d) => x(d.key))
    .attr('y', (d) => y(d.value))
    .attr('width', x.bandwidth())
    .attr('height', (d) => innerH - y(d.value))
    .attr('fill', accent)
    .attr('rx', 0)
  g.append('g')
    .attr('transform', `translate(0,${innerH})`)
    .call(
      axisBottom(x)
        .tickFormat((d) => data.find((b) => b.key === d)?.label ?? d)
    )
  g.append('g').call(axisLeft(y).ticks(5).tickFormat((v) => `${v}ms`))
  g.selectAll('.domain, .tick line').attr('stroke', getCssVar('--doc-border'))
  g.selectAll('.tick text').attr('fill', getCssVar('--doc-muted'))
}

function drawReq() {
  const el = reqEl.value
  if (!el || !reqSegments.value.length) return
  const margin = { top: 8, right: 12, bottom: 32, left: 44 }
  const w0 = el.clientWidth || 400
  const h = 120
  const width = w0
  const innerW = width - margin.left - margin.right
  const innerH = h - margin.top - margin.bottom
  const segs = reqSegments.value
  const total = segs.reduce((a, s) => a + s.value, 0) || 1

  select(el).selectAll('*').remove()
  const svg = select(el)
    .append('svg')
    .attr('width', width)
    .attr('height', h)
    .attr('role', 'img')
  const g = svg.append('g').attr('transform', `translate(${margin.left},${margin.top})`)
  const ok = segs.find((s) => s.key === 'ok')?.value ?? 0
  const fail = segs.find((s) => s.key === 'fail')?.value ?? 0
  const wOk = (ok / total) * innerW
  const wFail = (fail / total) * innerW
  const fillOk = getCssVar('--doc-badge-ok-fg') || '#15803d'
  const fillFail = '#dc2626'
  g.append('rect')
    .attr('x', 0)
    .attr('y', innerH * 0.2)
    .attr('width', wOk)
    .attr('height', innerH * 0.5)
    .attr('fill', fillOk)
  g.append('rect')
    .attr('x', wOk)
    .attr('y', innerH * 0.2)
    .attr('width', wFail)
    .attr('height', innerH * 0.5)
    .attr('fill', fillFail)
  g.append('text')
    .attr('x', 2)
    .attr('y', innerH * 0.12)
    .attr('font-size', 11)
    .attr('font-weight', 700)
    .attr('fill', getCssVar('--doc-text'))
    .text(`${ok} success · ${fail} fail · ${total} total`)
  const xS = scaleLinear().domain([0, total]).range([0, innerW])
  g.append('g')
    .attr('transform', `translate(0,${innerH * 0.75})`)
    .call(
      axisBottom(xS)
        .ticks(Math.min(6, total + 1))
        .tickFormat((d) => String(d))
    )
  g.selectAll('.domain, .tick line').attr('stroke', getCssVar('--doc-border'))
  g.selectAll('.tick text').attr('fill', getCssVar('--doc-muted'))
}

function redraw() {
  drawLatency()
  drawReq()
}

onMounted(() => {
  redraw()
  if (typeof ResizeObserver !== 'undefined' && rootRef.value) {
    ro = new ResizeObserver(() => redraw())
    ro.observe(rootRef.value)
  }
})

onBeforeUnmount(() => {
  if (ro) ro.disconnect()
  ro = null
})

watch(
  () => [props.data, latencyBars.value, reqSegments.value],
  () => {
    requestAnimationFrame(redraw)
  },
  { deep: true }
)
</script>

<style scoped>
.bench-charts {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
  min-width: 0;
}
.chart-block {
  border: 1px solid var(--doc-border);
  background: var(--doc-surface);
  border-radius: var(--doc-radius, 0);
  padding: 14px 16px 10px;
  min-width: 0;
  box-shadow: var(--doc-shadow-soft);
  box-sizing: border-box;
}
.chart-title {
  font-size: 11px;
  font-weight: 900;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: var(--doc-muted);
  margin: 0 0 8px;
}
.chart-svg {
  width: 100%;
  min-height: 200px;
}
.chart-svg:empty {
  min-height: 0;
}
.chart-empty {
  margin: 0;
  font-size: 12px;
  color: var(--doc-muted);
  line-height: 1.4;
}
.chart-empty.alone {
  padding: 10px 0;
}
@media (max-width: 1024px) {
  .bench-charts {
    grid-template-columns: 1fr;
  }
  .chart-block {
    padding: 12px 14px 8px;
  }
}
</style>
