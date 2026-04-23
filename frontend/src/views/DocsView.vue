<template>
  <div class="home-container doc-site docs-view">
    <header class="doc-topbar">
      <div class="doc-topbar-inner">
        <router-link to="/" class="doc-topbar-brand doc-topbar-brand-link">go-mirofish</router-link>
        <span class="doc-topbar-meta">{{ $t('docs.topbarMeta') }}</span>
        <div class="doc-topbar-actions">
          <router-link to="/docs" class="doc-topbar-nav-link doc-topbar-nav-link--active">
            {{ $t('nav.docs') }}
          </router-link>
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

    <div class="docs-layout">
      <aside class="docs-sidebar" :aria-label="$t('docs.sidebarAria')">
        <nav class="docs-nav">
          <p class="docs-nav-group">{{ $t('docs.navGroupStart') }}</p>
          <router-link class="docs-nav-item" to="/docs" :class="{ active: isOverview }">
            {{ $t('docs.navOverview') }}
          </router-link>
          <router-link class="docs-nav-item" to="/docs/installation" active-class="active">
            {{ $t('docs.navInstallation') }}
          </router-link>

          <p class="docs-nav-group">{{ $t('docs.navGroupConfig') }}</p>
          <router-link class="docs-nav-item" to="/docs/ollama" active-class="active">
            {{ $t('docs.navOllama') }}
          </router-link>
          <router-link class="docs-nav-item" to="/docs/providers" active-class="active">
            {{ $t('docs.navProviders') }}
          </router-link>

          <p class="docs-nav-group">{{ $t('docs.navGroupContrib') }}</p>
          <router-link class="docs-nav-item" to="/docs/contributing" active-class="active">
            {{ $t('docs.navContributing') }}
          </router-link>
        </nav>
      </aside>

      <div class="docs-main">
        <div class="docs-toolbar">
          <h1 class="docs-page-title">{{ pageTitle }}</h1>
          <PageActionsDropdown @scroll-playground="goToPlayground" />
        </div>

        <article v-if="isOverview" class="docs-article">
          <img
            class="docs-overview-logo"
            :src="overviewLogoSrc"
            :alt="$t('docs.overviewLogoAlt')"
            decoding="async"
          />
          <DocsArchitectureDiagram />
          <h2 class="docs-h2">{{ $t('docs.overviewWhatTitle') }}</h2>
          <p class="docs-p">{{ $t('docs.overviewWhatBody') }}</p>
          <h2 class="docs-h2">{{ $t('docs.overviewStackTitle') }}</h2>
          <ul class="docs-list">
            <li>{{ $t('docs.overviewStackItem1') }}</li>
            <li>{{ $t('docs.overviewStackItem2') }}</li>
            <li>{{ $t('docs.overviewStackItem3') }}</li>
            <li>{{ $t('docs.overviewStackItem4') }}</li>
          </ul>
          <p class="docs-p">
            <router-link class="docs-inline-link" to="/docs/installation">
              {{ $t('docs.overviewCtaInstall') }} →
            </router-link>
          </p>
        </article>

        <article v-else class="docs-article docs-article--md" v-html="renderedHtml" />
      </div>
    </div>
    <SiteFooter />
  </div>
</template>

<script setup>
import { marked } from 'marked'
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import LanguageSwitcher from '../components/LanguageSwitcher.vue'
import ThemeToggle from '../components/ThemeToggle.vue'
import SiteFooter from '../components/SiteFooter.vue'
import PageActionsDropdown from '../components/PageActions/PageActionsDropdown.vue'
import DocsArchitectureDiagram from '../components/DocsArchitectureDiagram.vue'
import overviewLogoSrc from '../assets/logo/go-mirofish-thumbnail.png'

import installationSource from '@docs/getting-started/installation.md?raw'
import ollamaSource from '@docs/configuration/ollama.md?raw'
import providersSource from '@docs/configuration/providers.md?raw'
import contributingSource from '@docs/contributing/README.md?raw'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()

const rawByKey = {
  installation: installationSource,
  ollama: ollamaSource,
  providers: providersSource,
  contributing: contributingSource,
}

const validSections = new Set(['installation', 'ollama', 'providers', 'contributing'])

const section = computed(() => {
  if (route.name === 'Docs') return 'overview'
  const s = route.params.section
  if (typeof s === 'string' && validSections.has(s)) return s
  return 'overview'
})

const isOverview = computed(() => route.name === 'Docs')

const pageTitle = computed(() => {
  if (section.value === 'overview') return t('docs.titleOverview')
  if (section.value === 'installation') return t('docs.titleInstallation')
  if (section.value === 'ollama') return t('docs.titleOllama')
  if (section.value === 'providers') return t('docs.titleProviders')
  if (section.value === 'contributing') return t('docs.titleContributing')
  return t('docs.titleDocs')
})

const renderedHtml = ref('')

/**
 * Marked has no GitHub "alert" blocks; it emits [!NOTE] as plain text in blockquotes. Promote to asides
 * and strip the label line so the UI matches common docs renderers.
 */
function promoteGitHubAlertBlockquotes(html) {
  return html.replace(
    /<blockquote>\s*<p>\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\](?:<br\s*\/?>|\s*<\/p>\s*<p>)([\s\S]*?)<\/blockquote>/gi,
    (_, kind, body) => {
      const c = String(kind).toLowerCase()
      return `<aside class="doc-alert doc-alert--${c}">${body}</aside>`
    }
  )
}

function renderForSection(s) {
  if (s === 'overview' || !rawByKey[s]) {
    renderedHtml.value = ''
    return
  }
  marked.setOptions({ gfm: true, breaks: true })
  const raw = rawByKey[s]
  renderedHtml.value = promoteGitHubAlertBlockquotes(marked.parse(raw))
}

watch(
  () => section.value,
  (s) => renderForSection(s),
  { immediate: true }
)

function goToPlayground() {
  router.push({ path: '/', hash: '#playground' })
}
</script>
