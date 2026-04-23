<template>
  <div class="home-container doc-site">
    <header class="doc-topbar">
      <div class="doc-topbar-inner">
        <router-link to="/" class="doc-topbar-brand doc-topbar-brand-link">go-mirofish</router-link>
        <span class="doc-topbar-meta">{{ $t('home.docTopbarMeta') }}</span>
        <div class="doc-topbar-actions">
          <router-link to="/docs" class="doc-topbar-nav-link">{{ $t('nav.docs') }}</router-link>
          <ThemeToggle />
          <LanguageSwitcher />
          <a
            href="https://github.com/go-mirofish/go-mirofish"
            target="_blank"
            rel="noopener noreferrer"
          >
            {{ $t('nav.visitGithub') }} ↗
          </a>
        </div>
      </div>
    </header>

    <div class="doc-page">
      <header class="doc-hero">
        <div class="doc-hero-copy">
          <div class="doc-badge-row">
            <span class="doc-pill doc-pill--accent">{{ $t('home.tagline') }}</span>
            <span class="doc-pill doc-pill--muted">{{ $t('home.version') }}</span>
          </div>
          <h1 class="doc-h1">
            <span class="doc-h1-line">{{ $t('home.heroTitle1') }}</span>
            <span class="doc-h1-gradient">{{ $t('home.heroTitle2') }}</span>
          </h1>
          <div class="doc-prose">
            <p>
              <i18n-t keypath="home.heroDesc" tag="span">
                <template #brand><span class="highlight-bold">{{ $t('home.heroDescBrand') }}</span></template>
                <template #agentScale><span class="highlight-orange">{{ $t('home.heroDescAgentScale') }}</span></template>
                <template #optimalSolution><span class="highlight-code">{{ $t('home.heroDescOptimalSolution') }}</span></template>
              </i18n-t>
            </p>
            <p class="doc-slogan">
              {{ $t('home.slogan') }}<span class="blinking-cursor">_</span>
            </p>
          </div>
        </div>
        <div class="doc-hero-media">
          <img :src="heroLogoUrl" alt="go-mirofish" class="doc-hero-logo" />
          <div class="doc-hero-cta">
            <a
              class="doc-anchor-btn doc-github-stars"
              href="https://github.com/go-mirofish/go-mirofish"
              target="_blank"
              rel="noopener noreferrer"
              :aria-label="$t('home.githubStarsAria')"
            >
              <span class="doc-github-stars-icon" aria-hidden="true">★</span>
              <span v-if="githubStars != null" class="doc-github-stars-count">{{ formatStars(githubStars) }}</span>
              <span v-else class="doc-github-stars-count doc-github-stars-count--loading">…</span>
            </a>
            <button type="button" class="doc-anchor-btn" @click="scrollToPlayground">
              {{ $t('home.docJumpPlayground') }}
            </button>
          </div>
        </div>
      </header>

      <div class="doc-content-grid" id="playground">
        <article class="doc-article" aria-label="Capabilities">
          <h2 class="doc-h2">{{ $t('home.docSectionStatus') }}</h2>
          <p class="doc-lead">
            <strong>{{ $t('home.systemReady') }}</strong> {{ $t('home.systemReadyDesc') }}
          </p>
          <div class="doc-metrics">
            <div class="doc-metric">
              <div class="doc-metric-val">{{ $t('home.metricLowCost') }}</div>
              <div class="doc-metric-label">{{ $t('home.metricLowCostDesc') }}</div>
            </div>
            <div class="doc-metric">
              <div class="doc-metric-val">{{ $t('home.metricHighAvail') }}</div>
              <div class="doc-metric-label">{{ $t('home.metricHighAvailDesc') }}</div>
            </div>
          </div>

          <h2 class="doc-h2">{{ $t('home.docSectionWorkflow') }}</h2>
          <p class="doc-lead" style="margin-top:0">{{ $t('home.workflowSequence') }}</p>
          <div class="doc-step-list">
            <div v-for="step in workflowSteps" :key="step.num" class="doc-step">
              <div class="doc-step-num">{{ step.num }}</div>
              <div>
                <div class="doc-step-title">{{ $t(step.titleKey) }}</div>
                <div class="doc-step-desc">{{ $t(step.descKey) }}</div>
              </div>
            </div>
          </div>
        </article>

        <aside class="doc-aside" aria-label="Playground">
          <p class="doc-aside-kicker">{{ $t('home.docSectionPlayground') }}</p>
          <p class="doc-aside-desc">{{ $t('home.docPlaygroundLead') }}</p>
          <div class="doc-playground-shell">
            <div class="doc-playground-inner">
              <div class="console-box">
                <div class="console-section">
                  <div class="console-header">
                    <span class="console-label">{{ $t('home.realitySeed') }}</span>
                    <span class="console-meta">{{ $t('home.supportedFormats') }}</span>
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
                      <div class="upload-title">{{ $t('home.dragToUpload') }}</div>
                      <div class="upload-hint">{{ $t('home.orBrowse') }}</div>
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
                  <span>{{ $t('home.inputParams') }}</span>
                </div>

                <div class="console-section">
                  <div class="console-header">
                    <span class="console-label">{{ $t('home.simulationPrompt') }}</span>
                  </div>
                  <div class="input-wrapper">
                    <textarea
                      v-model="formData.simulationRequirement"
                      class="code-input"
                      :placeholder="$t('home.promptPlaceholder')"
                      rows="6"
                      :disabled="loading"
                    />
                    <div class="model-badge">{{ $t('home.engineBadge') }}</div>
                  </div>
                </div>

                <div class="console-section btn-section">
                  <button
                    type="button"
                    class="start-engine-btn"
                    @click="startSimulation"
                    :disabled="!canSubmit || loading"
                  >
                    <span v-if="!loading">{{ $t('home.startEngine') }}</span>
                    <span v-else>{{ $t('home.initializing') }}</span>
                    <span class="btn-arrow">→</span>
                  </button>
                </div>
              </div>
            </div>
          </div>
        </aside>
      </div>

      <section class="doc-history-wrap">
        <h2 class="doc-h2">{{ $t('home.docSectionHistory') }}</h2>
        <HistoryDatabase />
      </section>
    </div>
    <SiteFooter />
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import HistoryDatabase from '../components/HistoryDatabase.vue'
import LanguageSwitcher from '../components/LanguageSwitcher.vue'
import ThemeToggle from '../components/ThemeToggle.vue'
import SiteFooter from '../components/SiteFooter.vue'
import heroLogoUrl from '@/assets/logo/go-mirofish-thumbnail.png'

const GITHUB_REPO_API = 'https://api.github.com/repos/go-mirofish/go-mirofish'
const githubStars = ref(null)

const formatStars = (n) => (typeof n === 'number' ? n.toLocaleString() : String(n))

onMounted(async () => {
  try {
    const r = await fetch(GITHUB_REPO_API, {
      headers: { Accept: 'application/vnd.github+json' }
    })
    if (!r.ok) return
    const d = await r.json()
    if (typeof d.stargazers_count === 'number') {
      githubStars.value = d.stargazers_count
    }
  } catch {
    // keep null; UI shows placeholder
  }
})

const workflowSteps = [
  { num: '01', titleKey: 'home.step01Title', descKey: 'home.step01Desc' },
  { num: '02', titleKey: 'home.step02Title', descKey: 'home.step02Desc' },
  { num: '03', titleKey: 'home.step03Title', descKey: 'home.step03Desc' },
  { num: '04', titleKey: 'home.step04Title', descKey: 'home.step04Desc' },
  { num: '05', titleKey: 'home.step05Title', descKey: 'home.step05Desc' },
]

const router = useRouter()

const formData = ref({
  simulationRequirement: ''
})

const files = ref([])

const loading = ref(false)
const error = ref('')
const isDragOver = ref(false)

const fileInput = ref(null)

const canSubmit = computed(() => {
  return formData.value.simulationRequirement.trim() !== '' && files.value.length > 0
})

const triggerFileInput = () => {
  if (!loading.value) {
    fileInput.value?.click()
  }
}

const handleFileSelect = (event) => {
  const selectedFiles = Array.from(event.target.files)
  addFiles(selectedFiles)
}

const handleDragOver = (e) => {
  if (!loading.value) {
    isDragOver.value = true
  }
}

const handleDragLeave = (e) => {
  isDragOver.value = false
}

const handleDrop = (e) => {
  isDragOver.value = false
  if (loading.value) return
  
  const droppedFiles = Array.from(e.dataTransfer.files)
  addFiles(droppedFiles)
}

const addFiles = (newFiles) => {
  const validFiles = newFiles.filter(file => {
    const ext = file.name.split('.').pop().toLowerCase()
    return ['pdf', 'md', 'txt'].includes(ext)
  })
  files.value.push(...validFiles)
}

const removeFile = (index) => {
  files.value.splice(index, 1)
}

const scrollToPlayground = () => {
  document.getElementById('playground')?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

const startSimulation = () => {
  if (!canSubmit.value || loading.value) return
  
  import('../store/pendingUpload.js').then(({ setPendingUpload }) => {
    setPendingUpload(files.value, formData.value.simulationRequirement)
    
    router.push({
      name: 'Process',
      params: { projectId: 'new' }
    })
  })
}
</script>

<style scoped>
/* Playground panel: form controls (doc shell in @/styles/doc-layout.css) */
:deep(.doc-playground-inner) .console-box {
  border: none;
  padding: 0;
}

.file-input-hidden {
  display: none;
}

.highlight-bold {
  color: var(--doc-text);
  font-weight: 700;
}

.highlight-orange {
  color: var(--doc-accent);
  font-weight: 600;
  font-family: "JetBrains Mono", ui-monospace, monospace;
}

.highlight-code {
  background: var(--doc-inline-code-bg);
  padding: 0.1rem 0.35rem;
  border-radius: 4px;
  font-family: "JetBrains Mono", ui-monospace, monospace;
  font-size: 0.9em;
}

.console-section {
  padding: 1.25rem;
}

.console-section.btn-section {
  padding-top: 0;
}

.console-header {
  display: flex;
  justify-content: space-between;
  margin-bottom: 0.75rem;
  font-family: "JetBrains Mono", ui-monospace, monospace;
  font-size: 0.75rem;
  color: var(--doc-muted);
}

.upload-zone {
  border: 1px dashed var(--doc-dashed);
  min-height: 200px;
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
  font-family: "JetBrains Mono", ui-monospace, monospace;
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
  font-family: "JetBrains Mono", ui-monospace, monospace;
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

.console-divider {
  display: flex;
  align-items: center;
  margin: 8px 0;
}

.console-divider::before,
.console-divider::after {
  content: "";
  flex: 1;
  height: 1px;
  background: var(--doc-border);
}

.console-divider span {
  padding: 0 12px;
  font-family: "JetBrains Mono", ui-monospace, monospace;
  font-size: 0.65rem;
  color: var(--doc-muted);
  letter-spacing: 0.06em;
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
  padding: 1rem 1.1rem;
  font-family: "JetBrains Mono", ui-monospace, monospace;
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
  font-family: "JetBrains Mono", ui-monospace, monospace;
  font-size: 0.65rem;
  color: var(--doc-muted);
}

.start-engine-btn {
  width: 100%;
  background: var(--doc-cta-primary-bg);
  color: var(--doc-cta-primary-fg);
  border: 1px solid var(--doc-cta-primary-bg);
  padding: 1rem 1.25rem;
  font-family: "JetBrains Mono", ui-monospace, monospace;
  font-weight: 700;
  font-size: 1rem;
  display: flex;
  justify-content: space-between;
  align-items: center;
  cursor: pointer;
  transition: background 0.2s, border-color 0.2s, transform 0.15s, color 0.2s;
  border-radius: 8px;
}

.start-engine-btn:not(:disabled) {
  animation: home-pulse 2.2s ease infinite;
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
  animation: none;
}

@keyframes home-pulse {
  0%,
  100% {
    box-shadow: 0 0 0 0 rgba(0, 173, 216, 0.2);
  }
  50% {
    box-shadow: 0 0 0 6px rgba(0, 173, 216, 0);
  }
}
</style>
