<template>
  <article class="md" v-html="html" ref="root" />
</template>

<script setup>
import { Marked, Renderer } from 'marked'
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { docSlugify, plainHeadingText } from './markdown/slugify.js'

const props = defineProps({
  source: { type: String, default: '' },
})

const root = ref(null)
const marked = new Marked()

function preprocessCallouts(raw) {
  // Support :::note / :::warning style blocks.
  return raw.replace(
    /::: *(note|tip|important|warning|caution)\s*\n([\s\S]*?)\n:::\s*/gi,
    (_, kind, body) => `> [!${String(kind).toUpperCase()}]\n> ${String(body).trim().replace(/\n/g, '\n> ')}\n`
  )
}

function promoteGitHubAlertBlockquotes(html) {
  return html.replace(
    /<blockquote>\s*<p>\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\](?:<br\s*\/?>|\s*<\/p>\s*<p>)([\s\S]*?)<\/blockquote>/gi,
    (_, kind, body) => {
      const c = String(kind).toLowerCase()
      return `<aside class="doc-alert doc-alert--${c}"><div class="doc-alert__title">${kind}</div>${body}</aside>`
    }
  )
}

function makeRenderer() {
  const r = new Renderer()
  r.heading = function (token) {
    const level = token.depth
    const inner = this.parser.parseInline(token.tokens)
    const id = docSlugify(plainHeadingText(token))
    return `<h${level} id="${id}"><a class="md-anchor" href="#${id}">#</a>${inner}</h${level}>\n`
  }
  r.code = function (token) {
    const lang = (token.lang || '').trim()
    const code = String(token.text ?? '')
    const forHtml = code.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
    const forAttr = forHtml.replace(/"/g, '&quot;')
    const label = lang ? `<span class="md-code-lang">${lang}</span>` : ''
    return `<div class="md-codeblock"><div class="md-codebar">${label}<button type="button" class="md-copy" data-copy="${forAttr}">Copy</button></div><pre><code>${forHtml}</code></pre></div>\n`
  }
  return r
}

function wrapTables(html) {
  return String(html)
    .replace(/<table>/g, '<div class="md-table-wrap"><table>')
    .replace(/<\/table>/g, '</table></div>')
}

marked.use({ gfm: true, breaks: true, renderer: makeRenderer() })

const html = computed(() => {
  if (!props.source) return ''
  const raw = preprocessCallouts(props.source)
  const out = String(marked.parse(raw))
  return promoteGitHubAlertBlockquotes(wrapTables(out))
})

async function wireCopyButtons() {
  await nextTick()
  const el = root.value
  if (!el) return
  el.querySelectorAll('button.md-copy').forEach((btn) => {
    if (btn.__wired) return
    btn.__wired = true
    btn.addEventListener('click', async () => {
      const text = btn.getAttribute('data-copy') || ''
      try {
        await navigator.clipboard.writeText(text)
        btn.textContent = 'Copied'
        setTimeout(() => (btn.textContent = 'Copy'), 900)
      } catch {
        // ignore
      }
    })
  })
}

onMounted(wireCopyButtons)
watch(() => html.value, wireCopyButtons)
</script>

<style scoped>
.md {
  background: var(--doc-surface);
  border: 1px solid var(--doc-border);
  border-radius: var(--doc-radius);
  padding: clamp(20px, 2vw, 32px) clamp(18px, 2.4vw, 40px);
  box-shadow: var(--doc-shadow-soft);
  max-width: 100%;
  box-sizing: border-box;
  width: 100%;
}

.md :deep(h1),
.md :deep(h2),
.md :deep(h3),
.md :deep(h4) { scroll-margin-top: 86px; }
.md :deep(h1) { font-size: 28px; margin: 0 0 14px; }
.md :deep(h1:first-child) { margin-top: 0; }
.md :deep(h2) { font-size: 20px; margin: 22px 0 10px; }
.md :deep(h2:first-child) { margin-top: 0; }
.md :deep(h3) { font-size: 16px; margin: 18px 0 8px; }
.md :deep(img) {
  max-width: 100%;
  height: auto;
  display: block;
  border-radius: var(--doc-radius, 2px);
}
.md :deep(p) { color: var(--doc-text); line-height: 1.75; margin: 10px 0; }
.md :deep(a) { color: var(--doc-accent); text-decoration: none; }
.md :deep(a:hover) { text-decoration: underline; }

.md :deep(ul),
.md :deep(ol) {
  margin: 10px 0 12px;
  padding-left: 1.5rem;
  list-style-position: outside;
  color: var(--doc-text);
  line-height: 1.65;
}
.md :deep(ul) { list-style-type: disc; }
.md :deep(ol) { list-style-type: decimal; }
.md :deep(li) { margin: 0.4em 0; padding-left: 0.2em; }
.md :deep(li::marker) { color: var(--doc-muted); }
.md :deep(hr) {
  border: 0;
  border-top: 1px solid var(--doc-border);
  margin: 1.25rem 0;
}
.md :deep(p code),
.md :deep(li code) {
  padding: 0.1em 0.35em;
  border-radius: 2px;
  background: color-mix(in srgb, var(--doc-code-bg) 90%, var(--doc-surface));
  font-size: 0.9em;
}

.md :deep(.md-anchor) {
  opacity: 0;
  margin-right: 8px;
  font-family: var(--doc-font-mono);
  color: var(--doc-muted);
  text-decoration: none;
}
.md :deep(h1:hover .md-anchor),
.md :deep(h2:hover .md-anchor),
.md :deep(h3:hover .md-anchor),
.md :deep(h4:hover .md-anchor) { opacity: 1; }

.md :deep(.md-table-wrap) {
  display: block;
  width: 100%;
  max-width: 100%;
  overflow-x: auto;
  margin: 16px 0;
  -webkit-overflow-scrolling: touch;
  border-radius: var(--doc-radius, 2px);
  border: 1px solid var(--doc-border);
  background: var(--doc-surface);
  box-sizing: border-box;
}

.md :deep(.md-table-wrap table) {
  width: 100%;
  min-width: 520px;
  border-collapse: collapse;
  margin: 0;
  table-layout: auto;
  border: none;
  border-radius: 0;
}

.md :deep(table) {
  width: 100%;
  border-collapse: collapse;
  margin: 12px 0;
  overflow: hidden;
  border-radius: 0;
  border: 1px solid var(--doc-border);
}
.md :deep(th),
.md :deep(td) {
  padding: 10px 12px;
  border-bottom: 1px solid var(--doc-border);
  text-align: left;
  vertical-align: top;
  word-break: break-word;
  overflow-wrap: anywhere;
  hyphens: auto;
}
.md :deep(th) {
  background: var(--doc-upload-surface);
  font-size: 12px;
  color: var(--doc-muted);
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.md :deep(.doc-alert) {
  border: 1px solid var(--doc-border);
  border-left: 4px solid var(--doc-accent);
  border-radius: var(--doc-radius, 2px);
  padding: 14px 18px 16px 20px;
  margin: 16px 0;
  background: color-mix(in srgb, var(--doc-upload-surface) 80%, transparent);
  box-sizing: border-box;
  overflow: visible;
}
.md :deep(.doc-alert__title) {
  font-weight: 800;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  font-size: 11px;
  color: var(--doc-muted);
  margin: 0 0 10px;
}
.md :deep(.doc-alert p) {
  margin: 8px 0;
  color: var(--doc-text);
}
.md :deep(.doc-alert p:first-of-type) {
  margin-top: 0;
}
.md :deep(.doc-alert p:last-of-type) {
  margin-bottom: 0;
}
.md :deep(.doc-alert ul),
.md :deep(.doc-alert ol) {
  margin: 10px 0 0;
  padding-left: 1.4rem;
  list-style-position: outside;
  color: var(--doc-text);
}
.md :deep(.doc-alert li) {
  margin: 0.45em 0;
  line-height: 1.6;
  padding-left: 0.25em;
}
.md :deep(.doc-alert a) {
  color: var(--doc-accent);
}
.md :deep(.doc-alert--warning),
.md :deep(.doc-alert--caution) { border-left-color: #f59e0b; }
.md :deep(.doc-alert--tip) { border-left-color: #3b82f6; }
.md :deep(.doc-alert--note),
.md :deep(.doc-alert--important) { border-left-color: var(--doc-accent); }

.md :deep(.md-codeblock) {
  display: flex;
  flex-direction: column;
  border: 1px solid var(--doc-border);
  border-radius: var(--doc-radius, 2px);
  overflow: hidden;
  margin: 16px 0;
  max-width: 100%;
  min-width: 0;
  background: var(--doc-surface);
}
.md :deep(.md-codebar) {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-shrink: 0;
  gap: 10px;
  padding: 8px 12px 7px 14px;
  border-bottom: 1px solid var(--doc-border);
  background: color-mix(in srgb, var(--doc-surface) 70%, transparent);
}
.md :deep(.md-code-lang) {
  font-family: var(--doc-font-mono);
  font-size: 11px;
  color: var(--doc-muted);
}
.md :deep(.md-codeblock pre) {
  margin: 0;
  padding: 11px 14px 12px 14px;
  overflow: auto;
  max-width: 100%;
  background: var(--doc-code-bg);
  border: 0;
  min-height: 0;
  line-height: 1.5;
  -webkit-overflow-scrolling: touch;
}
.md :deep(.md-codeblock code) {
  display: block;
  width: max-content;
  min-width: 100%;
  white-space: pre;
  font-family: var(--doc-font-mono);
  font-size: 12px;
  color: var(--doc-text);
}
.md :deep(p code) {
  display: inline;
  white-space: pre-wrap;
  word-break: break-word;
  width: auto;
  min-width: 0;
}
.md :deep(li code) {
  display: inline;
  white-space: pre-wrap;
  word-break: break-word;
}
.md :deep(.md-copy) {
  border: 1px solid var(--doc-border);
  background: var(--doc-surface);
  color: var(--doc-muted);
  padding: 6px 10px;
  border-radius: 0;
  cursor: pointer;
  font-weight: 700;
  font-size: 11px;
}
.md :deep(.md-copy:hover) {
  color: var(--doc-text);
  border-color: color-mix(in srgb, var(--doc-text) 14%, var(--doc-border));
}
</style>

