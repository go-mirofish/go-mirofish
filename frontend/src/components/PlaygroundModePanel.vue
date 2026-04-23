<template>
  <div class="playground-shell">
    <div class="mode-row">
      <button
        v-for="option in modeOptions"
        :key="option.key"
        type="button"
        class="mode-pill"
        :class="{ active: mode === option.key }"
        @click="mode = option.key"
      >
        {{ option.label }}
      </button>
    </div>

    <section v-if="mode === 'demo'" class="mode-panel">
      <div class="mode-header">
        <p class="mode-kicker">Static playground</p>
        <h3 class="mode-title">{{ fixture?.scenario?.title || 'Fixture-driven replay' }}</h3>
        <p class="mode-desc">
          {{ fixture?.scenario?.subtitle || 'A zero-cost interactive demo backed by committed JSON fixtures.' }}
        </p>
      </div>

      <div class="metric-grid">
        <div class="metric-card">
          <div class="metric-value">{{ fixture?.benchmark?.stress?.success_count ?? '—' }}/{{ fixture?.benchmark?.stress?.request_count ?? '—' }}</div>
          <div class="metric-label">Stress pass</div>
        </div>
        <div class="metric-card">
          <div class="metric-value">{{ fixture?.benchmark?.stress?.latency_ms?.p95 ?? '—' }}ms</div>
          <div class="metric-label">P95 latency</div>
        </div>
        <div class="metric-card">
          <div class="metric-value">{{ fixture?.benchmark?.full_flow?.upstream_status ?? '—' }}</div>
          <div class="metric-label">Live failure code</div>
        </div>
      </div>

      <div class="console-box">
        <div class="console-section">
          <div class="console-header">
            <span class="console-label">Seed excerpt</span>
            <span class="console-meta">Precomputed fixture</span>
          </div>
          <p class="fixture-copy">{{ fixture?.scenario?.seed_excerpt }}</p>
        </div>

        <div class="console-divider">
          <span>Fixture replay</span>
        </div>

        <div class="step-tab-row">
          <button
            v-for="step in demoSteps"
            :key="step.id"
            type="button"
            class="step-tab"
            :class="{ active: activeStepId === step.id }"
            @click="activeStepId = step.id"
          >
            {{ step.number }} · {{ step.title }}
          </button>
        </div>

        <div v-if="activeStep" class="step-panel">
          <h4>{{ activeStep.title }}</h4>
          <p class="step-summary">{{ activeStep.summary }}</p>
          <div class="stat-row">
            <div v-for="item in activeStep.stats" :key="item.label" class="stat-chip">
              <span>{{ item.label }}</span>
              <strong>{{ item.value }}</strong>
            </div>
          </div>
          <ul class="bullet-list">
            <li v-for="line in activeStep.bullets" :key="line">{{ line }}</li>
          </ul>
        </div>
      </div>
    </section>

    <section v-else-if="mode === 'local'" class="mode-panel">
      <div class="mode-header">
        <p class="mode-kicker">Real product</p>
        <h3 class="mode-title">Connect your local backend</h3>
        <p class="mode-desc">Run the real product on your own machine, then point this UI at that backend or local gateway.</p>
      </div>

      <div class="console-box">
        <div class="console-section">
          <div class="console-header">
            <span class="console-label">Local API base URL</span>
            <span class="console-meta">Self-hosted only</span>
          </div>
          <div class="runtime-row">
            <input
              v-model="draftApiBaseUrl"
              class="runtime-input"
              type="text"
              placeholder="http://127.0.0.1:5001"
              :disabled="loading"
            />
            <button type="button" class="runtime-btn" @click="saveApiBaseUrl" :disabled="loading">Save</button>
            <button type="button" class="runtime-btn runtime-btn--ghost" @click="resetApiBaseUrlField" :disabled="loading">Reset</button>
          </div>
          <p class="runtime-hint">Current runtime target: <code>{{ apiBaseUrl }}</code></p>
          <p v-if="connectionMessage" class="runtime-status" :class="connectionState">{{ connectionMessage }}</p>
          <button type="button" class="probe-btn" @click="probeLocalBackend" :disabled="loading">
            {{ loading ? 'Checking…' : 'Test local backend' }}
          </button>
        </div>

        <div class="console-divider">
          <span>Run for real</span>
        </div>

        <div class="console-section">
          <div class="console-header">
            <span class="console-label">Reality seed</span>
            <span class="console-meta">Formats: PDF, MD, TXT</span>
          </div>
          <div
            class="upload-zone"
            :class="{ 'drag-over': isDragOver, 'has-files': files.length > 0 }"
            @dragover.prevent="handleDragOver"
            @dragleave.prevent="handleDragLeave"
            @drop.prevent="handleDrop"
            @click="triggerFileInput"
          >
            <input
              ref="fileInput"
              type="file"
              multiple
              accept=".pdf,.md,.txt"
              @change="handleFileSelect"
              class="file-input-hidden"
              :disabled="loading"
            />
            <div v-if="files.length === 0" class="upload-placeholder">
              <div class="upload-icon">↑</div>
              <div class="upload-title">Drag files to upload</div>
              <div class="upload-hint">or click to browse files</div>
            </div>
            <div v-else class="file-list">
              <div v-for="(file, index) in files" :key="index" class="file-item">
                <span class="file-icon">📄</span>
                <span class="file-name">{{ file.name }}</span>
                <button type="button" @click.stop="removeFile(index)" class="remove-btn">×</button>
              </div>
            </div>
          </div>
        </div>

        <div class="console-divider">
          <span>Prompt</span>
        </div>

        <div class="console-section">
          <div class="input-wrapper">
            <textarea
              v-model="simulationRequirement"
              class="code-input"
              rows="6"
              :disabled="loading"
              placeholder="// Describe your simulation or prediction requirement in natural language"
            />
            <div class="model-badge">Local / self-hosted mode</div>
          </div>
        </div>

        <div class="console-section btn-section">
          <button
            type="button"
            class="start-engine-btn"
            @click="startRealRun"
            :disabled="!canSubmit || loading"
          >
            <span>Open real workbench</span>
            <span class="btn-arrow">→</span>
          </button>
        </div>
      </div>
    </section>

    <section v-else class="mode-panel">
      <div class="mode-header">
        <p class="mode-kicker">Advanced mode</p>
        <h3 class="mode-title">Bring your own keys or local models</h3>
        <p class="mode-desc">The public site stays cheap because real inference is pushed to your own backend, your own provider account, or your own local model endpoint.</p>
      </div>

      <div class="console-box">
        <div class="console-section">
          <div class="console-header">
            <span class="console-label">BYOK pattern</span>
            <span class="console-meta">No shared hosted inference</span>
          </div>
          <ul class="bullet-list">
            <li>Use a local OpenAI-compatible server such as llama.cpp or Ollama.</li>
            <li>Or point the backend at your own cloud provider account.</li>
            <li>The public playground remains fixture-driven; real runs happen only on your machine.</li>
          </ul>
        </div>

        <div class="console-divider">
          <span>.env example</span>
        </div>

        <div class="console-section">
          <pre class="env-block">{{ envSnippet }}</pre>
        </div>

        <div class="mode-note">
          <strong>Best production split:</strong> docs and showcase stay static, the playground stays precomputed, real execution stays local/self-hosted, and BYOK remains optional.
        </div>
      </div>
    </section>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { setPendingUpload } from '../store/pendingUpload'
import { useRuntimeApiBaseUrl } from '../composables/runtimeTarget'
import { playgroundMode as mode } from '../composables/playgroundMode'

const router = useRouter()
const { apiBaseUrl, setRuntimeApiBaseUrl, resetRuntimeApiBaseUrl } = useRuntimeApiBaseUrl()
const fixture = ref(null)
const activeStepId = ref('graph-build')
const files = ref([])
const fileInput = ref(null)
const isDragOver = ref(false)
const loading = ref(false)
const simulationRequirement = ref('')
const connectionMessage = ref('')
const connectionState = ref('')
const draftApiBaseUrl = ref(apiBaseUrl.value)

const modeOptions = [
  { key: 'demo', label: 'Demo playground' },
  { key: 'local', label: 'Local mode' },
  { key: 'byok', label: 'Advanced BYOK' },
]

const demoSteps = computed(() => fixture.value?.steps || [])
const activeStep = computed(() => demoSteps.value.find((step) => step.id === activeStepId.value) || demoSteps.value[0] || null)
const canSubmit = computed(() => simulationRequirement.value.trim() !== '' && files.value.length > 0)

const envSnippet = `LLM_BASE_URL=http://127.0.0.1:8080/v1
LLM_MODEL_NAME=your-model-name
LLM_API_KEY=optional-for-local-openai-compatible
ZEP_API_KEY=your-zep-key`

onMounted(async () => {
  try {
    const response = await fetch('/playground/demo-fixture.json')
    if (!response.ok) throw new Error(`fixture fetch failed: ${response.status}`)
    fixture.value = await response.json()
    activeStepId.value = fixture.value?.steps?.[0]?.id || activeStepId.value
    simulationRequirement.value = fixture.value?.scenario?.prompt || ''
  } catch {
    fixture.value = null
  }
})

const triggerFileInput = () => {
  if (!loading.value) fileInput.value?.click()
}

const addFiles = (incomingFiles) => {
  const validFiles = incomingFiles.filter((file) => {
    const ext = file.name.split('.').pop().toLowerCase()
    return ['pdf', 'md', 'txt'].includes(ext)
  })
  files.value.push(...validFiles)
}

const handleFileSelect = (event) => {
  addFiles(Array.from(event.target.files))
}

const handleDragOver = () => {
  if (!loading.value) isDragOver.value = true
}

const handleDragLeave = () => {
  isDragOver.value = false
}

const handleDrop = (event) => {
  isDragOver.value = false
  if (!loading.value) addFiles(Array.from(event.dataTransfer.files))
}

const removeFile = (index) => {
  files.value.splice(index, 1)
}

const saveApiBaseUrl = () => {
  const saved = setRuntimeApiBaseUrl(draftApiBaseUrl.value)
  draftApiBaseUrl.value = saved
  connectionMessage.value = `Saved runtime target: ${saved}`
  connectionState.value = 'ready'
}

const resetApiBaseUrlField = () => {
  draftApiBaseUrl.value = resetRuntimeApiBaseUrl()
  connectionMessage.value = `Reset runtime target to ${draftApiBaseUrl.value}`
  connectionState.value = 'ready'
}

const probeLocalBackend = async () => {
  loading.value = true
  connectionMessage.value = ''
  connectionState.value = ''
  const target = setRuntimeApiBaseUrl(draftApiBaseUrl.value)

  try {
    const response = await fetch(`${target}/api/graph/project/list?limit=1`, {
      headers: {
        Accept: 'application/json',
      },
    })

    if (!response.ok) {
      throw new Error(`HTTP ${response.status}`)
    }

    const payload = await response.json()
    if (payload.success === false) {
      throw new Error(payload.error || 'Unknown API error')
    }

    connectionMessage.value = `Connected to ${target}`
    connectionState.value = 'success'
  } catch (error) {
    connectionMessage.value = `Connection failed: ${error.message}`
    connectionState.value = 'failed'
  } finally {
    loading.value = false
  }
}

const startRealRun = () => {
  if (!canSubmit.value || loading.value) return
  setPendingUpload(files.value, simulationRequirement.value)
  router.push({
    name: 'Process',
    params: { projectId: 'new' },
  })
}
</script>

<style scoped>
.playground-shell {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.mode-row {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.mode-pill {
  border: 1px solid var(--doc-border);
  background: var(--doc-surface);
  color: var(--doc-text);
  border-radius: 999px;
  padding: 0.5rem 0.85rem;
  font-family: var(--doc-font-mono);
  font-size: 0.72rem;
  cursor: pointer;
}

.mode-pill.active {
  border-color: var(--doc-accent);
  background: color-mix(in srgb, var(--doc-accent) 12%, var(--doc-surface));
}

.mode-panel {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.mode-header {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.mode-kicker {
  margin: 0;
  color: var(--doc-accent);
  font-family: var(--doc-font-mono);
  font-size: 0.72rem;
  text-transform: uppercase;
  letter-spacing: 0.08em;
}

.mode-title {
  margin: 0;
  font-size: 1.25rem;
}

.mode-desc {
  margin: 0;
  color: var(--doc-muted);
  line-height: 1.6;
}

.metric-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0.75rem;
}

.metric-card {
  border: 1px solid var(--doc-border);
  background: var(--doc-surface);
  border-radius: 10px;
  padding: 0.9rem;
}

.metric-value {
  font-family: var(--doc-font-mono);
  font-weight: 700;
  font-size: 1rem;
}

.metric-label {
  margin-top: 0.35rem;
  color: var(--doc-muted);
  font-size: 0.78rem;
}

.console-box {
  border: 1px solid var(--doc-border);
  border-radius: 14px;
  background: var(--doc-surface);
  overflow: hidden;
}

.console-section {
  padding: 1rem 1.1rem;
}

.console-section.btn-section {
  padding-top: 0;
}

.console-header {
  display: flex;
  justify-content: space-between;
  gap: 1rem;
  margin-bottom: 0.75rem;
  font-family: var(--doc-font-mono);
  font-size: 0.72rem;
  color: var(--doc-muted);
}

.console-divider {
  display: flex;
  align-items: center;
  margin: 0 1rem;
}

.console-divider::before,
.console-divider::after {
  content: "";
  flex: 1;
  height: 1px;
  background: var(--doc-border);
}

.console-divider span {
  padding: 0 0.75rem;
  font-family: var(--doc-font-mono);
  font-size: 0.65rem;
  color: var(--doc-muted);
  letter-spacing: 0.08em;
}

.fixture-copy,
.step-summary {
  margin: 0;
  color: var(--doc-text);
  line-height: 1.6;
}

.step-tab-row {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
  padding: 1rem;
  padding-bottom: 0;
}

.step-tab {
  border: 1px solid var(--doc-border);
  background: var(--doc-upload-surface);
  color: var(--doc-text);
  border-radius: 8px;
  padding: 0.55rem 0.8rem;
  font-family: var(--doc-font-mono);
  font-size: 0.7rem;
  cursor: pointer;
}

.step-tab.active {
  border-color: var(--doc-accent);
  background: color-mix(in srgb, var(--doc-accent) 10%, var(--doc-surface));
}

.step-panel {
  padding: 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.8rem;
}

.step-panel h4 {
  margin: 0;
}

.stat-row {
  display: flex;
  gap: 0.6rem;
  flex-wrap: wrap;
}

.stat-chip {
  border: 1px solid var(--doc-border);
  background: var(--doc-upload-surface);
  border-radius: 999px;
  padding: 0.45rem 0.7rem;
  display: flex;
  gap: 0.5rem;
  align-items: center;
  font-size: 0.74rem;
}

.bullet-list {
  margin: 0;
  padding-left: 1.1rem;
  color: var(--doc-muted);
  line-height: 1.6;
}

.mode-note {
  border: 1px solid var(--doc-border);
  background: color-mix(in srgb, var(--doc-accent) 8%, var(--doc-surface));
  border-radius: 10px;
  padding: 0.9rem 1rem;
  color: var(--doc-text);
  line-height: 1.6;
}

.runtime-row {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.runtime-input {
  flex: 1;
  min-width: 240px;
  border: 1px solid var(--doc-border);
  background: var(--doc-upload-surface);
  color: var(--doc-text);
  border-radius: 8px;
  padding: 0.8rem 0.9rem;
  font-family: var(--doc-font-mono);
}

.runtime-btn,
.probe-btn {
  border: 1px solid var(--doc-border);
  background: var(--doc-surface);
  color: var(--doc-text);
  border-radius: 8px;
  padding: 0.75rem 0.9rem;
  font-family: var(--doc-font-mono);
  cursor: pointer;
}

.runtime-btn--ghost {
  background: var(--doc-upload-surface);
}

.probe-btn {
  margin-top: 0.75rem;
}

.runtime-hint,
.runtime-status {
  margin: 0.75rem 0 0;
  color: var(--doc-muted);
  font-size: 0.82rem;
}

.runtime-status.success {
  color: #2e7d32;
}

.runtime-status.failed {
  color: #c62828;
}

.runtime-status.ready {
  color: var(--doc-text);
}

.file-input-hidden {
  display: none;
}

.upload-zone {
  border: 1px dashed var(--doc-dashed);
  min-height: 170px;
  overflow-y: auto;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: background 0.2s, border-color 0.2s;
  background: var(--doc-upload-surface);
  border-radius: 8px;
}

.upload-zone.has-files {
  align-items: flex-start;
}

.upload-zone:hover,
.upload-zone.drag-over {
  background: var(--doc-code-bg);
  border-color: var(--doc-accent);
}

.upload-placeholder {
  text-align: center;
}

.upload-icon {
  width: 40px;
  height: 40px;
  border: 1px solid var(--doc-border);
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0 auto 15px;
  color: var(--doc-muted);
  border-radius: 6px;
}

.upload-title {
  font-weight: 500;
  font-size: 0.9rem;
  margin-bottom: 5px;
}

.upload-hint {
  font-family: var(--doc-font-mono);
  font-size: 0.75rem;
  color: var(--doc-muted);
}

.file-list {
  width: 100%;
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.file-item {
  display: flex;
  align-items: center;
  background: var(--doc-surface);
  padding: 8px 12px;
  border: 1px solid var(--doc-border);
  font-family: var(--doc-font-mono);
  font-size: 0.85rem;
  border-radius: 6px;
}

.file-name {
  flex: 1;
  margin: 0 10px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.remove-btn {
  background: none;
  border: none;
  cursor: pointer;
  font-size: 1.2rem;
  color: var(--doc-muted);
}

.input-wrapper {
  position: relative;
  border: 1px solid var(--doc-border);
  background: var(--doc-upload-surface);
  border-radius: 8px;
}

.code-input {
  width: 100%;
  box-sizing: border-box;
  border: none;
  background: transparent;
  color: var(--doc-text);
  padding: 1rem 1.1rem;
  font-family: var(--doc-font-mono);
  font-size: 0.875rem;
  line-height: 1.6;
  resize: vertical;
  outline: none;
  min-height: 140px;
}

.model-badge {
  position: absolute;
  bottom: 8px;
  right: 12px;
  font-family: var(--doc-font-mono);
  font-size: 0.65rem;
  color: var(--doc-muted);
}

.start-engine-btn {
  width: 100%;
  background: var(--doc-cta-primary-bg);
  color: var(--doc-cta-primary-fg);
  border: 1px solid var(--doc-cta-primary-bg);
  padding: 1rem 1.25rem;
  font-family: var(--doc-font-mono);
  font-weight: 700;
  font-size: 1rem;
  display: flex;
  justify-content: space-between;
  align-items: center;
  cursor: pointer;
  transition: background 0.2s, border-color 0.2s, transform 0.15s, color 0.2s;
  border-radius: 8px;
}

.start-engine-btn:hover:not(:disabled) {
  background: var(--doc-accent);
  border-color: var(--doc-accent);
  transform: translateY(-1px);
}

.start-engine-btn:disabled {
  background: var(--doc-cta-locked-bg);
  color: var(--doc-cta-locked-fg);
  border-color: var(--doc-cta-locked-bg);
  cursor: not-allowed;
}

.env-block {
  margin: 0;
  border: 1px solid var(--doc-border);
  background: var(--doc-upload-surface);
  border-radius: 8px;
  padding: 1rem;
  color: var(--doc-text);
  overflow-x: auto;
  font-family: var(--doc-font-mono);
  font-size: 0.8rem;
  line-height: 1.7;
}

@media (max-width: 900px) {
  .metric-grid {
    grid-template-columns: 1fr;
  }

  .runtime-row {
    flex-direction: column;
  }

  .runtime-input {
    min-width: 0;
  }
}
</style>
