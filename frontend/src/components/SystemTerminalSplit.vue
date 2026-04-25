<template>
  <div class="system-terminal-split">
    <div class="terminal-grid">
      <!-- Left: 2-col header grid + log output -->
      <section class="terminal-col terminal-output" aria-label="Output terminal">
        <div class="output-top-grid">
          <div class="output-top-left">
            <div class="log-header">
              <div class="log-header-titles">
                <span class="log-title">{{ panelTitle }}</span>
                <span class="log-subtitle">{{ $t('systemTerminal.outputSubtitle') }}</span>
              </div>
            </div>
          </div>
          <div class="output-top-right">
            <div class="output-meta">
              <span class="output-meta-id" :title="idLabel || undefined">{{ idLabel || '—' }}</span>
              <span
                class="output-live"
                :class="{ on: pipelineAnimating }"
                :aria-label="$t('systemTerminal.processLive')"
              >
                <span class="output-live-dot" />
                {{ $t('systemTerminal.processLive') }}
              </span>
            </div>
          </div>
        </div>
        <div class="log-content" ref="logContentRef" tabindex="0">
          <div class="log-line" v-for="(log, idx) in logs" :key="idx">
            <span class="log-time">{{ log.time }}</span>
            <span class="log-msg">{{ log.msg }}</span>
          </div>
        </div>
      </section>

      <!-- Right: local device metrics -->
      <section class="terminal-col terminal-hardware" :aria-label="$t('systemTerminal.hardwareTitle')">
        <div class="hw-header">
          <div class="hw-header-titles">
            <span class="hw-title">{{ $t('systemTerminal.hardwareTitle') }}</span>
            <span class="hw-subtitle">{{ $t('systemTerminal.hardwareSubtitle') }}</span>
          </div>
        </div>
        <p class="hw-scope" role="note">{{ $t('systemTerminal.localScopeNote') }}</p>
        <dl class="hw-metrics">
          <div v-if="localClient" class="hw-row hw-row-client">
            <dt>{{ $t('systemTerminal.client') }}</dt>
            <dd>{{ localClient }}</dd>
          </div>
          <div v-if="localDisplay" class="hw-row">
            <dt>{{ $t('systemTerminal.display') }}</dt>
            <dd>{{ localDisplay }}</dd>
          </div>
          <div v-if="localTimeZone" class="hw-row">
            <dt>{{ $t('systemTerminal.timeZone') }}</dt>
            <dd>{{ localTimeZone }}</dd>
          </div>
          <div class="hw-row">
            <dt>{{ $t('systemTerminal.cpuCores') }}</dt>
            <dd>{{ hardware.cpuCores ?? '—' }}</dd>
          </div>
          <div class="hw-row" :class="{ warn: hardware.eventLoopLag > 50 }">
            <dt>{{ $t('systemTerminal.eventLoop') }}</dt>
            <dd>{{ hardware.eventLoopLag != null ? `${hardware.eventLoopLag} ${$t('systemTerminal.ms')}` : '—' }}</dd>
          </div>
          <div v-if="hardware.jsHeap" class="hw-row">
            <dt>{{ $t('systemTerminal.jsHeap') }}</dt>
            <dd>{{ hardware.jsHeap }}</dd>
          </div>
          <div v-if="hardware.deviceMem != null" class="hw-row">
            <dt>{{ $t('systemTerminal.deviceMem') }}</dt>
            <dd>{{ hardware.deviceMem }} GiB</dd>
          </div>
          <div v-if="hardware.network" class="hw-row hw-row-network">
            <dt>{{ $t('systemTerminal.network') }}</dt>
            <dd>{{ hardware.network }}</dd>
          </div>
          <div v-if="batteryLine" class="hw-row">
            <dt>{{ $t('systemTerminal.battery') }}</dt>
            <dd>{{ batteryLine }}</dd>
          </div>
          <div class="hw-row hw-thermal">
            <dt>{{ $t('systemTerminal.thermal') }}</dt>
            <dd :title="$t('systemTerminal.thermalHint')">{{ $t('systemTerminal.thermalNA') }}</dd>
          </div>
        </dl>
      </section>
    </div>

    <!-- Full-width pipeline status (animated when pipelineAnimating) -->
    <div v-if="showProcessStatus" class="process-status" :class="{ animating: pipelineAnimating }">
      <div class="process-status-head">
        <span class="process-status-title">{{ $t('systemTerminal.processStatusTitle') }}</span>
        <div class="process-bar-bg" aria-hidden="true">
          <div
            class="process-bar-fill"
            :class="{ indeterminate: pipelineAnimating }"
            :style="pipelineAnimating ? null : pipelineBarStyle"
          />
        </div>
      </div>
      <ol class="process-steps" :aria-label="$t('systemTerminal.processStatusTitle')">
        <li
          v-for="(name, idx) in workflowLabels"
          :key="idx"
          class="process-step"
          :class="processStepClass(idx)"
        >
          <span class="process-step-idx">{{ idx + 1 }}</span>
          <span class="process-step-name">{{ name }}</span>
        </li>
      </ol>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'

const props = defineProps({
  panelTitle: { type: String, required: true },
  idLabel: { type: String, default: '' },
  logs: { type: Array, default: () => [] },
  /** 1–5 = current focus step in the five-step workbench (matches `main.stepNames`) */
  workflowStep: { type: Number, default: 1 },
  /** Pulse / shimmer the active step and the progress bar */
  pipelineAnimating: { type: Boolean, default: false },
  /** Show the pipeline strip (default on) */
  showProcessStatus: { type: Boolean, default: true }
})

const { t, tm } = useI18n()

const logContentRef = ref(null)

const localClient = ref('')
const localDisplay = ref('')
const localTimeZone = ref('')

const hardware = ref({
  cpuCores: null,
  eventLoopLag: null,
  jsHeap: null,
  deviceMem: null,
  network: null
})

const battery = ref({ level: null, charging: null })

const workflowLabels = computed(() => {
  const names = tm('main.stepNames')
  return Array.isArray(names) && names.length ? names : [
    'Graph Build', 'Env Setup', 'Run Simulation', 'Report Generation', 'Deep Interaction'
  ]
})

const activeIndex = computed(() => {
  const s = Math.min(5, Math.max(1, Math.floor(props.workflowStep || 1)))
  return s - 1
})

const pipelineBarStyle = computed(() => {
  const n = workflowLabels.value.length
  if (n < 1) return { width: '0%' }
  const pct = ((activeIndex.value + 1) / n) * 100
  return { width: `${pct}%` }
})

function processStepClass(idx) {
  const active = activeIndex.value
  if (idx < active) return 'is-done'
  if (idx === active) return props.pipelineAnimating ? 'is-active is-pulse' : 'is-active'
  return 'is-pending'
}

const batteryLine = computed(() => {
  if (battery.value.level == null) return null
  const pct = Math.round(battery.value.level * 100)
  const ch = battery.value.charging ? ` · ${t('systemTerminal.charging')}` : ''
  return `${pct}%${ch}`
})

let nextTickAt = 0
let eventLoopTimer = null
let refreshTimer = null
let batteryObj = null
let batterySync = null
let connHandler = null

function measureEventLoopLag() {
  const now = performance.now()
  if (nextTickAt) {
    hardware.value.eventLoopLag = Math.max(0, Math.round(now - nextTickAt))
  }
  nextTickAt = now + 1000
}

function refreshLocalDisplayAndZone() {
  if (typeof screen !== 'undefined' && typeof window !== 'undefined') {
    const dpr = window.devicePixelRatio ?? 1
    localDisplay.value = `${screen.width}×${screen.height} · ${dpr}× dpr`
  }
  try {
    const tz = Intl.DateTimeFormat().resolvedOptions().timeZone
    localTimeZone.value = tz || ''
  } catch {
    localTimeZone.value = ''
  }
}

async function resolveLocalClient() {
  try {
    const uad = navigator.userAgentData
    if (uad) {
      const h = await uad.getHighEntropyValues([
        'platform',
        'platformVersion',
        'architecture',
        'fullVersionList'
      ])
      const os = [h.platform, h.platformVersion].filter(Boolean).join(' ')
      const arch = h.architecture ? ` · ${h.architecture}` : ''
      let browser = ''
      if (h.fullVersionList?.length) {
        const pick = h.fullVersionList[h.fullVersionList.length - 1]
        if (pick?.brand && pick.version) {
          browser = ` · ${pick.brand} ${pick.version.split('.')[0]}`
        }
      }
      const line = (os + arch + browser).trim()
      if (line) {
        localClient.value = line
        return
      }
    }
  } catch {
    /* fall through */
  }
  const plat = typeof navigator !== 'undefined' ? navigator.userAgent : ''
  if (plat) {
    localClient.value = plat.length > 100 ? `${plat.slice(0, 97)}…` : plat
  }
}

function refreshStaticMetrics() {
  refreshLocalDisplayAndZone()
  if (typeof navigator !== 'undefined' && navigator.hardwareConcurrency) {
    hardware.value.cpuCores = navigator.hardwareConcurrency
  }
  if (typeof navigator !== 'undefined' && navigator.deviceMemory) {
    hardware.value.deviceMem = navigator.deviceMemory
  }
  const mem = performance.memory
  if (mem && mem.usedJSHeapSize != null) {
    const used = (mem.usedJSHeapSize / (1024 * 1024)).toFixed(1)
    const limit = mem.jsHeapSizeLimit
      ? (mem.jsHeapSizeLimit / (1024 * 1024)).toFixed(0)
      : null
    hardware.value.jsHeap = limit ? `${used} / ${limit} MB` : `${used} MB`
  }
  const c = typeof navigator !== 'undefined' && navigator.connection
  if (c) {
    const parts = [c.effectiveType, c.downlink != null ? `${c.downlink} Mbps` : null, c.rtt != null ? `RTT ${c.rtt}ms` : null].filter(
      Boolean
    )
    hardware.value.network = parts.join(' · ') || null
  }
}

async function initBattery() {
  try {
    if (navigator.getBattery) {
      batteryObj = await navigator.getBattery()
      batterySync = () => {
        if (!batteryObj) return
        battery.value = { level: batteryObj.level, charging: batteryObj.charging }
      }
      batterySync()
      batteryObj.addEventListener('levelchange', batterySync)
      batteryObj.addEventListener('chargingchange', batterySync)
    }
  } catch {
    /* ignore */
  }
}

watch(
  () => props.logs?.length,
  () => {
    nextTick(() => {
      const el = logContentRef.value
      if (el) el.scrollTop = el.scrollHeight
    })
  }
)

onMounted(() => {
  void resolveLocalClient()
  nextTickAt = performance.now() + 1000
  eventLoopTimer = window.setInterval(measureEventLoopLag, 1000)
  measureEventLoopLag()
  refreshStaticMetrics()
  refreshTimer = window.setInterval(refreshStaticMetrics, 3000)
  initBattery()
  if (typeof navigator !== 'undefined' && navigator.connection) {
    connHandler = () => refreshStaticMetrics()
    navigator.connection.addEventListener('change', connHandler)
  }
  window.addEventListener('resize', refreshLocalDisplayAndZone)
})

onUnmounted(() => {
  if (eventLoopTimer) clearInterval(eventLoopTimer)
  if (refreshTimer) clearInterval(refreshTimer)
  if (batteryObj && batterySync) {
    batteryObj.removeEventListener('levelchange', batterySync)
    batteryObj.removeEventListener('chargingchange', batterySync)
  }
  if (typeof navigator !== 'undefined' && navigator.connection && connHandler) {
    navigator.connection.removeEventListener('change', connHandler)
  }
  window.removeEventListener('resize', refreshLocalDisplayAndZone)
})
</script>

<style scoped>
.system-terminal-split {
  width: 100%;
  flex-shrink: 0;
  padding: 10px 16px 12px;
  border-top: 1px solid var(--doc-console-border, #27272a);
  background: var(--doc-bg, transparent);
}

.terminal-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(220px, 300px);
  gap: 12px;
  /* Equal-height columns: shorter panel stretches to the taller one. */
  align-items: stretch;
}

@media (max-width: 900px) {
  .terminal-grid {
    grid-template-columns: 1fr;
  }
}

.terminal-col {
  background: var(--doc-console-bg, #0a0a0c);
  color: var(--doc-console-fg, #d4d4d8);
  font-family: 'JetBrains Mono', ui-monospace, monospace;
  border: 1px solid var(--doc-console-border, #27272a);
  border-radius: 2px;
  min-height: 0;
  display: flex;
  flex-direction: column;
  align-self: stretch;
}

.terminal-output {
  min-width: 0;
  width: 100%;
}

.output-top-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 8px 12px;
  align-items: start;
  padding: 8px 12px 0;
  border-bottom: 1px solid var(--doc-console-border, #333);
  flex-shrink: 0;
}

@media (max-width: 520px) {
  .output-top-grid {
    grid-template-columns: 1fr;
  }
}

.output-top-left .log-header {
  border-bottom: none;
  padding: 0 0 6px 0;
}

.output-meta {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 6px;
  min-width: 0;
  padding: 0 0 6px 0;
}

.output-meta-id {
  font-size: 9px;
  color: var(--doc-console-muted, #9ca3af);
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.output-live {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 8px;
  text-transform: uppercase;
  letter-spacing: 0.12em;
  color: var(--doc-console-muted, #71717a);
  font-weight: 600;
}

.output-live.on {
  color: var(--doc-accent, #22c55e);
}

.output-live-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--doc-console-muted, #52525b);
  flex-shrink: 0;
}

.output-live.on .output-live-dot {
  background: var(--doc-accent, #22c55e);
  box-shadow: 0 0 0 0 color-mix(in srgb, var(--doc-accent, #22c55e) 45%, transparent);
  animation: live-breathe 1.2s ease-in-out infinite;
}

@keyframes live-breathe {
  0%,
  100% {
    transform: scale(1);
    opacity: 1;
  }
  50% {
    transform: scale(1.2);
    opacity: 0.75;
  }
}

.terminal-hardware {
  padding: 0 0 8px;
}

.log-header,
.hw-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 8px;
  font-size: 10px;
  color: var(--doc-console-muted, #9ca3af);
  flex-shrink: 0;
}

.hw-header {
  border-bottom: 1px solid var(--doc-console-border, #333);
  padding: 10px 12px 8px;
}

.hw-header-titles {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.hw-subtitle {
  font-size: 9px;
  opacity: 0.88;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  font-weight: 500;
}

.hw-scope {
  margin: 0 12px 8px;
  font-size: 9px;
  line-height: 1.4;
  color: var(--doc-console-muted, #9ca3af);
}

.log-header-titles {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.log-title {
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.04em;
  color: var(--doc-console-fg, #d4d4d8);
}

.log-subtitle {
  font-size: 9px;
  opacity: 0.85;
  text-transform: uppercase;
  letter-spacing: 0.06em;
}

.log-content {
  /* Fills remaining height so output column matches hardware column; scroll inside. */
  flex: 1 1 0;
  min-height: 0;
  overflow-y: auto;
  padding: 6px 12px 8px;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.log-content::-webkit-scrollbar {
  width: 4px;
}
.log-content::-webkit-scrollbar-thumb {
  background: var(--doc-console-scroll, #3f3f46);
  border-radius: 2px;
}

.log-line {
  font-size: 11px;
  display: flex;
  gap: 12px;
  line-height: 1.5;
}
.log-time {
  color: var(--doc-console-muted, #9ca3af);
  min-width: 75px;
  flex-shrink: 0;
}
.log-msg {
  color: var(--doc-console-fg, #d4d4d8);
  word-break: break-all;
  min-width: 0;
}

.hw-title {
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.04em;
  color: var(--doc-console-fg, #d4d4d8);
}

.hw-metrics {
  margin: 0;
  padding: 4px 12px 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.hw-row {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  gap: 8px;
  font-size: 10px;
  line-height: 1.4;
}
.hw-row dt {
  margin: 0;
  color: var(--doc-console-muted, #9ca3af);
  font-weight: 400;
  flex: 0 0 42%;
}
.hw-row dd {
  margin: 0;
  text-align: right;
  color: var(--doc-console-fg, #e4e4e7);
  font-size: 10px;
  word-break: break-word;
}
.hw-row.warn dd {
  color: #fbbf24;
}
.hw-row-network dd {
  font-size: 9px;
}
.hw-thermal dd {
  cursor: help;
  color: var(--doc-console-muted, #9ca3af);
}

.hw-row-client {
  flex-direction: column;
  align-items: stretch;
  gap: 4px;
}
.hw-row-client dt {
  flex: none;
}
.hw-row-client dd {
  text-align: left;
  font-size: 9px;
  line-height: 1.35;
  max-width: 100%;
}

/* Full-width process status */
.process-status {
  margin-top: 12px;
  padding: 10px 12px 12px;
  border: 1px solid var(--doc-console-border, #27272a);
  border-radius: 4px;
  background: color-mix(in srgb, var(--doc-console-bg, #0a0a0c) 85%, var(--doc-bg, #09090b));
  position: relative;
  overflow: hidden;
}

.process-status.animating::after {
  content: '';
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  height: 2px;
  background: linear-gradient(
    90deg,
    transparent,
    color-mix(in srgb, var(--doc-accent, #00add8) 55%, transparent),
    transparent
  );
  background-size: 200% 100%;
  animation: status-shimmer 1.5s ease-in-out infinite;
  pointer-events: none;
}

@keyframes status-shimmer {
  0% {
    background-position: 200% 0;
  }
  100% {
    background-position: -200% 0;
  }
}

.process-status-head {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 10px;
}

.process-status-title {
  font-size: 9px;
  font-weight: 600;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--doc-console-muted, #9ca3af);
}

.process-bar-bg {
  height: 4px;
  border-radius: 2px;
  background: var(--doc-console-border, #27272a);
  overflow: hidden;
  position: relative;
}

.process-bar-fill {
  height: 100%;
  border-radius: 2px;
  background: linear-gradient(
    90deg,
    color-mix(in srgb, var(--doc-accent, #00add8) 80%, #1e3a4a),
    var(--doc-accent, #00add8)
  );
  transition: width 0.45s cubic-bezier(0.25, 0.8, 0.25, 1);
  width: 0;
}

.process-bar-fill.indeterminate {
  width: 100% !important;
  background: linear-gradient(
    90deg,
    color-mix(in srgb, var(--doc-accent, #00add8) 20%, transparent),
    color-mix(in srgb, var(--doc-accent, #00add8) 60%, transparent),
    color-mix(in srgb, var(--doc-accent, #00add8) 20%, transparent)
  );
  background-size: 200% 100%;
  animation: bar-scan 1.1s linear infinite;
}

.process-status.animating .process-bar-fill:not(.indeterminate) {
  box-shadow: 0 0 12px color-mix(in srgb, var(--doc-accent, #00add8) 35%, transparent);
}

@keyframes bar-scan {
  0% {
    background-position: 200% 0;
  }
  100% {
    background-position: -200% 0;
  }
}

.process-steps {
  list-style: none;
  margin: 0;
  padding: 0;
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 6px;
}

@media (max-width: 900px) {
  .process-steps {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

.process-step {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  font-size: 8px;
  line-height: 1.2;
  color: var(--doc-console-muted, #9ca3af);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.process-step-idx {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  border-radius: 3px;
  border: 1px solid var(--doc-console-border, #3f3f46);
  background: var(--doc-console-bg, #18181b);
  font-size: 9px;
  font-weight: 700;
  flex-shrink: 0;
}

.process-step-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
}

.process-step.is-pending .process-step-idx {
  opacity: 0.55;
}

.process-step.is-active .process-step-idx {
  border-color: color-mix(in srgb, var(--doc-accent, #00add8) 65%, #3f3f46);
  color: var(--doc-console-fg, #fafafa);
  background: color-mix(in srgb, var(--doc-accent, #00add8) 22%, #18181b);
}

.process-step.is-active .process-step-name {
  color: var(--doc-console-fg, #e4e4e7);
  font-weight: 600;
}

.process-step.is-pulse .process-step-idx {
  animation: step-pulse 1.1s ease-in-out infinite;
}

@keyframes step-pulse {
  0%,
  100% {
    box-shadow: 0 0 0 0 color-mix(in srgb, var(--doc-accent, #00add8) 40%, transparent);
  }
  50% {
    box-shadow: 0 0 0 4px color-mix(in srgb, var(--doc-accent, #00add8) 0%, transparent);
  }
}

.process-step.is-done .process-step-idx {
  background: color-mix(in srgb, #22c55e 22%, #18181b);
  border-color: #3f6f4e;
  color: #86efac;
}

.process-step.is-done .process-step-name {
  color: var(--doc-console-muted, #a1a1aa);
}
</style>
