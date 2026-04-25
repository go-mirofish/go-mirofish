// Central manifest for /docs navigation + rendering.
// Keeps docs structure declarative so the layout can render sidebar/TOC/footer consistently.

export const DOCS_REPO_EDIT_BASE =
  'https://github.com/go-mirofish/go-mirofish/tree/main/'

/**
 * @typedef {{
 *   key: string,
 *   titleKey: string,
 *   path: string,
 *   type: 'overview' | 'markdown' | 'page',
 *   sourcePath?: string, // repo-relative .md path
 *   componentName?: string, // for type=page
 * }} DocEntry
 */

/** @type {{ key: string, titleKey: string, entries: DocEntry[] }[]} */
export const DOCS_GROUPS = [
  {
    key: 'start',
    titleKey: 'docs.navGroupStart',
    entries: [
      { key: 'overview', titleKey: 'docs.navOverview', path: '/docs', type: 'overview' },
      {
        key: 'installation',
        titleKey: 'docs.navInstallation',
        path: '/docs/installation',
        type: 'markdown',
        sourcePath: 'docs/getting-started/installation.md',
      },
    ],
  },
  {
    key: 'config',
    titleKey: 'docs.navGroupConfig',
    entries: [
      {
        key: 'ollama',
        titleKey: 'docs.navOllama',
        path: '/docs/ollama',
        type: 'markdown',
        sourcePath: 'docs/configuration/ollama.md',
      },
      {
        key: 'providers',
        titleKey: 'docs.navProviders',
        path: '/docs/providers',
        type: 'markdown',
        sourcePath: 'docs/configuration/providers.md',
      },
    ],
  },
  {
    key: 'proof',
    titleKey: 'docs.navGroupProof',
    entries: [
      {
        key: 'benchmark',
        titleKey: 'docs.navBenchmark',
        path: '/docs/benchmark',
        type: 'page',
        componentName: 'DocsBenchmarks',
        sourcePath: 'docs/report/benchmark-report.md',
      },
      {
        key: 'showcase',
        titleKey: 'docs.navShowcase',
        path: '/docs/showcase',
        type: 'page',
        componentName: 'DocsShowcase',
        sourcePath: 'docs/showcase.md',
      },
    ],
  },
  {
    key: 'contrib',
    titleKey: 'docs.navGroupContrib',
    entries: [
      {
        key: 'contributing',
        titleKey: 'docs.navContributing',
        path: '/docs/contributing',
        type: 'page',
        componentName: 'DocsContributing',
        sourcePath: 'docs/contributing/README.md',
      },
      {
        key: 'roadmap',
        titleKey: 'docs.navRoadmap',
        path: '/docs/roadmap',
        type: 'page',
        componentName: 'DocsRoadmap',
        sourcePath: 'docs/roadmap/roadmap.md',
      },
      {
        key: 'future-consideration',
        titleKey: 'docs.navFutureConsideration',
        path: '/docs/future-consideration',
        type: 'markdown',
        sourcePath: 'docs/roadmap/future-consideration.md',
      },
    ],
  },
]

/** @returns {DocEntry[]} */
export function allDocsEntries() {
  return DOCS_GROUPS.flatMap((g) => g.entries)
}

/** @param {string} path */
export function findEntryByPath(path) {
  return allDocsEntries().find((e) => e.path === path) || null
}

/** @param {string | undefined} sourcePath */
export function toEditUrl(sourcePath) {
  if (!sourcePath) return null
  return `${DOCS_REPO_EDIT_BASE}${sourcePath}`
}
