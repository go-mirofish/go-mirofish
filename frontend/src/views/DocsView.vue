<template>
  <DocsLayout :title="pageTitle" :hasToc="Boolean(tocHeadings.length)">
    <template #sidebar>
      <DocsSidebar :groups="DOCS_GROUPS" :activePath="activePath" />
    </template>

    <template v-if="tocHeadings.length" #toc>
      <DocsToc :headings="tocHeadings" :activeId="activeTocId" />
    </template>

    <template #default>
      <div class="page-head">
        <div class="page-title">{{ pageTitle }}</div>
        <PageActionsDropdown @scroll-playground="goToPlayground" />
      </div>

      <section v-if="entry?.type === 'overview'" class="overview doc-main">
        <div class="overview-logo-wrap">
          <img class="overview-logo" :src="overviewLogoSrc" :alt="$t('docs.overviewLogoAlt')" decoding="async" />
        </div>
        <DocsArchitectureDiagram />
        <div class="overview-grid">
          <div class="overview-card">
            <div class="kicker">{{ $t('docs.overviewWhatTitle') }}</div>
            <div class="body">{{ $t('docs.overviewWhatBody') }}</div>
          </div>
          <div class="overview-card">
            <div class="kicker">{{ $t('docs.overviewStackTitle') }}</div>
            <ul class="list">
              <li>{{ $t('docs.overviewStackItem1') }}</li>
              <li>{{ $t('docs.overviewStackItem2') }}</li>
              <li>{{ $t('docs.overviewStackItem3') }}</li>
              <li>{{ $t('docs.overviewStackItem4') }}</li>
              <li>{{ $t('docs.overviewStackItem5') }}</li>
            </ul>
          </div>
        </div>

        <div class="overview-mirofish">
          <h2 class="overview-section-h">{{ $t('docs.overviewVsSectionTitle') }}</h2>
          <p class="overview-lead">{{ $t('docs.overviewVsSectionLead') }}</p>
          <div class="overview-vs-table-wrap" role="region" :aria-label="$t('docs.overviewVsTableAria')">
            <table class="overview-vs-table">
              <caption class="overview-vs-caption">{{ $t('docs.overviewVsTableCaption') }}</caption>
              <thead>
                <tr>
                  <th scope="col">{{ $t('docs.overviewVsColTopic') }}</th>
                  <th scope="col">{{ $t('docs.overviewVsColPlain') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr>
                  <td class="overview-vs-td-label">{{ $t('docs.overviewVsRow1Label') }}</td>
                  <td>{{ $t('docs.overviewVsRow1Text') }}</td>
                </tr>
                <tr>
                  <td class="overview-vs-td-label">{{ $t('docs.overviewVsRow2Label') }}</td>
                  <td>{{ $t('docs.overviewVsRow2Text') }}</td>
                </tr>
                <tr>
                  <td class="overview-vs-td-label">{{ $t('docs.overviewVsRow3Label') }}</td>
                  <td>{{ $t('docs.overviewVsRow3Text') }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        <div class="overview-links">
          <router-link class="pill" to="/docs/installation">{{ $t('docs.overviewCtaInstall') }} →</router-link>
          <router-link class="pill" to="/docs/benchmark">{{ $t('docs.overviewCtaBenchmark') }} →</router-link>
        </div>
      </section>

      <div
        v-else
        class="doc-page"
        :class="{
          'doc-page--prose': entry?.type === 'markdown',
          'doc-page--wide': entry?.componentName === 'DocsBenchmarks' || entry?.componentName === 'DocsShowcase' || entry?.componentName === 'DocsRoadmap',
        }"
      >
        <DocsBenchmarks v-if="entry?.componentName === 'DocsBenchmarks'" />
        <DocsShowcase v-else-if="entry?.componentName === 'DocsShowcase'" />
        <DocsContributing v-else-if="entry?.componentName === 'DocsContributing'" />
        <DocsRoadmap v-else-if="entry?.componentName === 'DocsRoadmap'" />
        <DocsMarkdown v-else :source="markdownSource" />
      </div>
    </template>

    <template #footer-left>
      <span v-if="sourcePathDisplay">{{ $t('docs.source') }}: <code>{{ sourcePathDisplay }}</code></span>
    </template>
    <template #footer-right>
      <a v-if="editUrl" class="edit" :href="editUrl" target="_blank" rel="noopener noreferrer">
        {{ $t('docs.editOnGithub') }} ↗
      </a>
    </template>
  </DocsLayout>
</template>

<script setup>
import { computed, nextTick, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import PageActionsDropdown from '../components/PageActions/PageActionsDropdown.vue'
import DocsArchitectureDiagram from '../components/DocsArchitectureDiagram.vue'
import overviewLogoSrc from '../assets/logo/go-mirofish-thumbnail.png'

import DocsLayout from '../components/docs/DocsLayout.vue'
import DocsSidebar from '../components/docs/DocsSidebar.vue'
import DocsToc from '../components/docs/DocsToc.vue'
import DocsMarkdown from '../components/docs/DocsMarkdown.vue'
import DocsBenchmarks from '../components/docs/pages/DocsBenchmarks.vue'
import DocsShowcase from '../components/docs/pages/DocsShowcase.vue'
import DocsContributing from '../components/docs/pages/DocsContributing.vue'
import DocsRoadmap from '../components/docs/pages/DocsRoadmap.vue'

import { DOCS_GROUPS, findEntryByPath, toEditUrl } from '../docs/manifest'
import { docSlugify } from '../components/docs/markdown/slugify.js'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()

const activePath = computed(() => (route.name === 'Docs' ? '/docs' : `/docs/${route.params.section}`))
const entry = computed(() => findEntryByPath(activePath.value))

const pageTitle = computed(() => {
  if (entry.value?.key === 'overview') return t('docs.titleOverview')
  if (entry.value?.key === 'installation') return t('docs.titleInstallation')
  if (entry.value?.key === 'ollama') return t('docs.titleOllama')
  if (entry.value?.key === 'providers') return t('docs.titleProviders')
  if (entry.value?.key === 'benchmark') return t('docs.titleBenchmark')
  if (entry.value?.key === 'showcase') return t('docs.titleShowcase')
  if (entry.value?.key === 'contributing') return t('docs.titleContributing')
  if (entry.value?.key === 'roadmap') return t('docs.titleRoadmap')
  if (entry.value?.key === 'future-consideration') return t('docs.titleFutureConsideration')
  return t('docs.titleDocs')
})

// Load markdown sources via Vite virtual import mapping.
const mdModules = import.meta.glob('../../../docs/**/*.md', { as: 'raw', eager: true })
const markdownSource = computed(() => {
  const sp = entry.value?.sourcePath
  if (!sp) return ''
  const key = `../../../${sp}`
  return mdModules[key] || ''
})

const activeTocId = ref('')
let tocObserver = null

function extractTocFromMarkdown(source) {
  const headings = []
  const lines = String(source || '').split('\n')
  for (const line of lines) {
    const m = line.match(/^(#{2,4})\s+(.+)$/)
    if (!m) continue
    const level = m[1].length
    const text = m[2].replace(/\s+#.*$/, '').trim()
    const id = docSlugify(text)
    headings.push({ level, text, id })
  }
  return headings
}

const tocHeadings = computed(() => {
  const e = entry.value
  if (!e) return []
  if (e.componentName === 'DocsBenchmarks') {
    return [
      { level: 2, text: t('docs.bench.tocSummary'), id: 'doc-bench-hero' },
      { level: 2, text: t('docs.bench.tocMetrics'), id: 'doc-bench-metrics' },
      { level: 2, text: t('docs.bench.tocCharts'), id: 'doc-bench-charts' },
      { level: 2, text: t('docs.bench.tocGrid'), id: 'doc-bench-grid' },
    ]
  }
  if (e.componentName === 'DocsShowcase') {
    return [
      { level: 2, text: t('docs.showcase.tocHero'), id: 'doc-show-hero' },
      { level: 2, text: t('docs.showcase.tocHighlights'), id: 'doc-show-highlights' },
      { level: 2, text: t('docs.showcase.tocShots'), id: 'doc-show-shots' },
      { level: 2, text: t('docs.showcase.tocFaq'), id: 'doc-show-faq' },
    ]
  }
  if (e.componentName === 'DocsContributing') {
    return [
      { level: 2, text: t('docs.contrib.tocStart'), id: 'doc-contrib-hero' },
      { level: 2, text: t('docs.contrib.tocWork'), id: 'doc-contrib-work' },
      { level: 2, text: t('docs.contrib.tocGuidelines'), id: 'doc-contrib-guides' },
    ]
  }
  if (e.componentName === 'DocsRoadmap') {
    return [
      { level: 2, text: t('docs.roadmap.tocIntro'), id: 'doc-roadmap-hero' },
      { level: 2, text: t('docs.roadmap.tocThemes'), id: 'doc-roadmap-themes' },
      { level: 2, text: t('docs.roadmap.tocMore'), id: 'doc-roadmap-more' },
    ]
  }
  if (e.type === 'markdown') {
    return extractTocFromMarkdown(markdownSource.value)
  }
  return []
})

async function setupScrollSpy() {
  if (typeof IntersectionObserver === 'undefined') return
  if (tocObserver) {
    tocObserver.disconnect()
    tocObserver = null
  }
  await nextTick()
  const ids = tocHeadings.value.map((h) => h.id)
  const els = ids.map((id) => document.getElementById(id)).filter(Boolean)
  if (!els.length) {
    activeTocId.value = ''
    return
  }
  const obs = new IntersectionObserver(
    (entries) => {
      const visible = entries.filter((e) => e.isIntersecting).sort((a, b) => b.intersectionRatio - a.intersectionRatio)
      if (visible[0]?.target?.id) activeTocId.value = visible[0].target.id
    },
    { root: null, rootMargin: '0px 0px -70% 0px', threshold: [0.1, 0.2, 0.4, 0.6, 0.8] }
  )
  els.forEach((el) => obs.observe(el))
  tocObserver = obs
}

watch(
  [() => tocHeadings.value, () => entry.value?.path, () => markdownSource.value],
  () => {
    void setupScrollSpy()
  },
  { immediate: true, flush: 'post' }
)

onUnmounted(() => {
  if (tocObserver) {
    tocObserver.disconnect()
    tocObserver = null
  }
})

const editUrl = computed(() => toEditUrl(entry.value?.sourcePath))

const sourcePathDisplay = computed(() => {
  const sp = entry.value?.sourcePath
  if (typeof sp === 'string' && sp) return sp
  return ''
})

function goToPlayground() {
  router.push({ path: '/', hash: '#playground' })
}
</script>

<style scoped>
.page-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 10px 14px;
  margin-bottom: clamp(8px, 1.2vw, 14px);
}

.page-title {
  font-size: clamp(1.1rem, 2.2vw, 1.375rem);
  font-weight: 900;
  letter-spacing: -0.02em;
  line-height: 1.25;
  min-width: 0;
}

.doc-main,
.doc-page {
  width: 100%;
  min-width: 0;
  box-sizing: border-box;
}

.doc-page {
  display: flex;
  flex-direction: column;
  gap: clamp(12px, 1.5vw, 20px);
  max-width: min(100%, 56rem);
  margin-left: auto;
  margin-right: auto;
  /* Main column already has --doc-main-pad-x on .content; avoid double horizontal inset. */
  padding: 0;
  min-width: 0;
  overflow: visible;
}

.doc-page--prose {
  max-width: min(100%, 50rem);
}

.doc-page--wide {
  max-width: 100%;
  margin-left: 0;
  margin-right: 0;
}

.overview {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.overview-logo-wrap {
  display: flex;
  justify-content: center;
  align-items: center;
  width: 100%;
  padding: 12px 0 8px;
}

.overview-logo {
  width: min(300px, 90vw);
  max-height: 200px;
  height: auto;
  object-fit: contain;
  display: block;
  opacity: 1;
  filter: drop-shadow(0 2px 12px color-mix(in srgb, var(--doc-text) 14%, transparent));
}

.overview-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
}

.overview-mirofish {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.overview-section-h {
  margin: 8px 0 0;
  font-size: clamp(1.05rem, 2vw, 1.2rem);
  font-weight: 850;
  letter-spacing: -0.02em;
  line-height: 1.3;
  color: var(--doc-text);
}

.overview-lead {
  margin: 0 0 4px;
  font-size: 0.9rem;
  line-height: 1.6;
  color: var(--doc-muted);
  max-width: 72ch;
}

.overview-vs-table-wrap {
  width: 100%;
  min-width: 0;
  overflow-x: auto;
  -webkit-overflow-scrolling: touch;
  border: 1px solid var(--doc-border);
  border-radius: var(--doc-radius);
  background: var(--doc-surface);
  box-shadow: var(--doc-shadow-soft);
}

.overview-vs-table {
  width: 100%;
  min-width: 520px;
  border-collapse: collapse;
  font-size: 0.9rem;
  line-height: 1.55;
  color: var(--doc-text);
}

.overview-vs-caption {
  caption-side: top;
  text-align: left;
  padding: 12px 14px 0;
  font-size: 0.8rem;
  font-weight: 700;
  color: var(--doc-muted);
}

.overview-vs-table thead th {
  text-align: left;
  padding: 10px 14px;
  font-size: 0.7rem;
  font-weight: 800;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  color: var(--doc-muted);
  background: color-mix(in srgb, var(--doc-code-bg) 60%, var(--doc-surface));
  border-bottom: 1px solid var(--doc-border);
}

.overview-vs-table tbody td {
  padding: 12px 14px;
  vertical-align: top;
  border-bottom: 1px solid var(--doc-border);
}

.overview-vs-table tbody tr:last-child td {
  border-bottom: none;
}

.overview-vs-td-label {
  font-weight: 700;
  color: var(--doc-text);
  width: 28%;
  min-width: 7.5rem;
  background: color-mix(in srgb, var(--doc-bg) 35%, var(--doc-surface));
}

.overview-card {
  border: 1px solid var(--doc-border);
  background: var(--doc-surface);
  border-radius: var(--doc-radius);
  padding: 16px;
  box-shadow: var(--doc-shadow-soft);
}

.kicker {
  font-size: 11px;
  font-weight: 800;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--doc-muted);
  margin-bottom: 8px;
}

.body {
  color: var(--doc-text);
  line-height: 1.7;
}

.list {
  margin: 0;
  padding-left: 18px;
  color: var(--doc-text);
  line-height: 1.7;
}

.overview-links {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}

.pill {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  border: 1px solid var(--doc-border);
  padding: 10px 12px;
  border-radius: 0;
  background: color-mix(in srgb, var(--doc-surface) 70%, transparent);
  text-decoration: none;
  font-weight: 800;
  color: var(--doc-text);
}
.pill:hover {
  border-color: color-mix(in srgb, var(--doc-accent) 32%, var(--doc-border));
}

.edit {
  color: var(--doc-muted);
  text-decoration: none;
  font-weight: 800;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  font-size: 11px;
}
.edit:hover { color: var(--doc-text); }

@media (max-width: 860px) {
  .overview-grid {
    grid-template-columns: 1fr;
  }
  .page-head {
    flex-direction: column;
    align-items: stretch;
  }
  .page-title {
    line-height: 1.2;
  }
}
</style>
