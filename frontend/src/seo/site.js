/**
 * Public site base URL for canonical, OG, sitemap, and JSON-LD.
 * Override with VITE_SITE_BASE_URL for staging or a custom domain.
 */
export const SITE_BASE =
  (typeof import.meta !== 'undefined' && import.meta.env?.VITE_SITE_BASE_URL) ||
  'https://go-mirofish.vercel.app'

export const SITE_NAME = 'go-mirofish'

const DEFAULT = {
  title: `${SITE_NAME} | Local prediction stack`,
  description:
    'go-mirofish is a local-first prediction stack for document ingestion, knowledge graphs, social simulation, and reports on your machine.',
  path: '/',
  keywords: [
    'knowledge graph',
    'local-first',
    'simulation',
    'Mirofish',
    'document ingestion',
    'ontology',
  ].join(', '),
}

const DOCS_SECTIONS = {
  installation: {
    title: 'Installation & setup | go-mirofish documentation',
    description:
      'Install go-mirofish: Docker, gateway, Vite UI, and environment variables for a local prediction stack on your machine.',
  },
  ollama: {
    title: 'Ollama & local models | go-mirofish documentation',
    description:
      'Configure Ollama and local LLM endpoints to run the go-mirofish gateway and simulations without a hosted API.',
  },
  providers: {
    title: 'LLM providers & API keys | go-mirofish documentation',
    description:
      'Set up OpenAI-compatible providers, models, and optional boost endpoints for go-mirofish document and graph workflows.',
  },
  benchmark: {
    title: 'Benchmarks & performance | go-mirofish documentation',
    description:
      'Run and interpret bundled go-mirofish benchmarks: scenarios, load tests, and parity checks on your hardware.',
  },
  showcase: {
    title: 'Showcase & scenarios | go-mirofish documentation',
    description:
      'Explore example go-mirofish use cases: urban planning, product launches, incident drills, and literary simulations.',
  },
  contributing: {
    title: 'Contributing | go-mirofish',
    description:
      'Conventions, commits, and how to contribute to the go-mirofish open-source fork (Go gateway, Vue UI).',
  },
  roadmap: {
    title: 'Roadmap | go-mirofish',
    description:
      'Planned and in-progress go-mirofish work: features, parity, and the Go migration story.',
  },
  'future-consideration': {
    title: 'Future consideration | go-mirofish',
    description:
      'Long-term ideas, experiments, and non-commitments for the go-mirofish project.',
  },
}

const DOCS_LANDING = {
  title: 'Documentation | go-mirofish',
  description:
    'Guides for installing go-mirofish, configuring LLMs, running benchmarks, and contributing to the project.',
  path: '/docs',
}

const ROUTES = {
  Home: {
    ...DEFAULT,
  },
  Docs: { ...DOCS_LANDING },
  DocsSection: (route) => {
    const id = String(route.params.section || '')
    const block = DOCS_SECTIONS[id]
    if (block) {
      return {
        title: block.title,
        description: block.description,
        path: route.path,
        keywords: ['go-mirofish', 'documentation', id.replace(/-/g, ' ')].join(
          ', '
        ),
      }
    }
    return {
      title: DOCS_LANDING.title,
      description: DOCS_LANDING.description,
      path: route.path,
    }
  },
  Process: {
    title: 'Project workspace | go-mirofish',
    description:
      'Local document processing and knowledge graph for a project. Use your local stack to open this view.',
    path: null,
    noindex: true,
  },
  Simulation: {
    title: 'Simulation | go-mirofish',
    description:
      'Configure and run a social or scenario simulation backed by the go-mirofish graph and gateway.',
    path: null,
    noindex: true,
  },
  SimulationRun: {
    title: 'Simulation run | go-mirofish',
    description:
      'Live run of a go-mirofish scenario simulation with events, metrics, and graph updates.',
    path: null,
    noindex: true,
  },
  Report: {
    title: 'Report | go-mirofish',
    description:
      'Read or export a generated go-mirofish report for a session or simulation run.',
    path: null,
    noindex: true,
  },
  Interaction: {
    title: 'Interaction | go-mirofish',
    description:
      'Interactive follow-up and exploration for a go-mirofish report or graph slice.',
    path: null,
    noindex: true,
  },
}

/**
 * @param {import('vue-router').RouteLocationNormalizedLoaded} route
 */
export function resolveSeoForRoute(route) {
  const name = String(route.name || 'Home')
  const base = SITE_BASE.replace(/\/$/, '')
  const pathOnly = route.path || '/'
  const absoluteUrl =
    pathOnly === '/' ? `${base}/` : `${base}${pathOnly}`.split('#')[0]

  const def = ROUTES[name]
  let rec =
    typeof def === 'function' ? def(route) : (def && { ...def }) || { ...DEFAULT }

  if (Object.prototype.hasOwnProperty.call(rec, 'path') && rec.path == null) {
    rec = { ...rec, path: pathOnly }
  }
  if (rec.path == null) {
    rec = { ...rec, path: pathOnly }
  }

  const noindex = Boolean(rec.noindex)
  const keywords = rec.keywords != null ? rec.keywords : DEFAULT.keywords
  return {
    title: rec.title,
    description: rec.description,
    fullPath: route.fullPath,
    canonicalPath: rec.path,
    absoluteUrl: absoluteUrl || `${base}/`,
    keywords,
    noindex,
  }
}

export function getSoftwareApplicationJsonLd() {
  return {
    '@context': 'https://schema.org',
    '@type': 'SoftwareApplication',
    name: SITE_NAME,
    applicationCategory: 'DeveloperApplication',
    operatingSystem: 'Any',
    offers: { '@type': 'Offer', price: '0', priceCurrency: 'USD' },
    description: DEFAULT.description,
    url: SITE_BASE,
  }
}

export function getWebSiteJsonLd() {
  return {
    '@context': 'https://schema.org',
    '@type': 'WebSite',
    name: SITE_NAME,
    url: `${SITE_BASE.replace(/\/$/, '')}/`,
    description: DEFAULT.description,
  }
}
