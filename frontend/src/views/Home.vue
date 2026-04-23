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
          <p class="doc-aside-kicker">Production split</p>
          <p class="doc-aside-desc">
            Static docs and showcase, fixture-driven public playground, local/self-hosted real product, and optional BYOK for advanced users.
          </p>
          <PlaygroundModePanel />
        </aside>

        <div
          v-if="playgroundMode === 'demo'"
          class="playground-zero-cost-note"
        >
          <strong>{{ $t('home.zeroCostNoteLead') }}</strong>
          {{ $t('home.zeroCostNoteDesc') }}
        </div>
      </div>

      <section class="doc-history-wrap" :aria-label="$t('home.docSectionHistory')">
        <HistoryDatabase />
      </section>
    </div>
    <SiteFooter />
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import HistoryDatabase from '../components/HistoryDatabase.vue'
import LanguageSwitcher from '../components/LanguageSwitcher.vue'
import ThemeToggle from '../components/ThemeToggle.vue'
import SiteFooter from '../components/SiteFooter.vue'
import PlaygroundModePanel from '../components/PlaygroundModePanel.vue'
import { playgroundMode } from '../composables/playgroundMode'
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

const scrollToPlayground = () => {
  document.getElementById('playground')?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}
</script>

<style scoped>
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
</style>
