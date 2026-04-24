<template>
  <div class="contrib">
    <section id="doc-contrib-hero" class="hero">
      <div>
        <div class="title">{{ $t('docs.contrib.title') }}</div>
        <div class="sub">{{ $t('docs.contrib.subtitle') }}</div>
      </div>
      <div class="links">
        <a class="btn" :href="repo" target="_blank" rel="noopener noreferrer">{{ $t('docs.contrib.openRepo') }} ↗</a>
        <a class="btn primary" :href="issues" target="_blank" rel="noopener noreferrer">{{ $t('docs.contrib.openIssues') }} ↗</a>
      </div>
    </section>

    <section class="panel">
      <div class="panel-title">{{ $t('docs.contrib.quickStart') }}</div>
      <div class="tabs">
        <button class="tab" :class="{ active: os === 'win' }" @click="os = 'win'">Windows</button>
        <button class="tab" :class="{ active: os === 'mac' }" @click="os = 'mac'">macOS</button>
        <button class="tab" :class="{ active: os === 'linux' }" @click="os = 'linux'">Linux</button>
      </div>
      <pre class="code"><code>{{ commands[os] }}</code></pre>
    </section>

    <section id="doc-contrib-work" class="grid">
      <div class="panel">
        <div class="panel-title">{{ $t('docs.contrib.workflows') }}</div>
        <div class="table">
          <div class="row head">
            <div>Task</div><div>Command</div>
          </div>
          <div class="row" v-for="w in workflows" :key="w.task">
            <div class="task">{{ w.task }}</div>
            <div class="cmd mono"><code>{{ w.cmd }}</code></div>
          </div>
        </div>
      </div>

      <div class="panel">
        <div class="panel-title">{{ $t('docs.contrib.checklist') }}</div>
        <label class="check" v-for="c in checklist" :key="c">
          <input type="checkbox" />
          <span>{{ c }}</span>
        </label>
      </div>
    </section>

    <section id="doc-contrib-guides" class="panel guidelines-md">
      <div class="panel-title">{{ $t('docs.contrib.guidelines') }}</div>
      <DocsMarkdown v-if="contribMd" :source="contribMd" />
      <p v-else class="empty-md">{{ $t('docs.contrib.guidelinesEmpty') }}</p>
    </section>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import DocsMarkdown from '../DocsMarkdown.vue'

const os = ref('win')
const repo = 'https://github.com/go-mirofish/go-mirofish'
const issues = 'https://github.com/go-mirofish/go-mirofish/issues'

const commands = {
  win: `# From repo root\nnpm run setup:all\nnpm run dev`,
  mac: `# From repo root\nnpm run setup:all\nnpm run dev`,
  linux: `# From repo root\nnpm run setup:all\nnpm run dev`,
}

const workflows = computed(() => [
  { task: 'Start dev stack', cmd: 'npm run dev' },
  { task: 'Build frontend', cmd: 'npm run build' },
  { task: 'Run commit helper', cmd: 'npm run commit' },
  { task: 'Security check staged files', cmd: 'npm run security:check' },
])

const checklist = computed(() => [
  'Explained the why (not only the what)',
  'No secrets (.env / keys) included',
  'UI verified in dark + light',
  'Docs updated if behavior changed',
])

const mdModules = import.meta.glob('../../../../../docs/contributing/*.md', { as: 'raw', eager: true })
const contribMd = computed(() => {
  return mdModules['../../../../../docs/contributing/README.md'] || ''
})
</script>

<style scoped>
.contrib {
  display: flex;
  flex-direction: column;
  gap: clamp(12px, 1.5vw, 18px);
  width: 100%;
  min-width: 0;
}
.hero, .panel {
  border: 1px solid var(--doc-border);
  background: var(--doc-surface);
  border-radius: var(--doc-radius);
  padding: 16px;
  box-shadow: var(--doc-shadow-soft);
}
.hero { display: flex; justify-content: space-between; gap: 14px; flex-wrap: wrap; }
.title { font-size: 24px; font-weight: 950; letter-spacing: -0.03em; }
.sub { margin-top: 8px; color: var(--doc-muted); line-height: 1.7; max-width: 70ch; }
.links { display: flex; gap: 10px; align-items: center; }
.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 10px 12px;
  border-radius: 0;
  border: 1px solid var(--doc-border);
  background: color-mix(in srgb, var(--doc-surface) 70%, transparent);
  color: var(--doc-text);
  text-decoration: none;
  font-weight: 900;
}
.btn.primary {
  background: var(--doc-cta-primary-bg, #111827);
  color: var(--doc-cta-primary-fg, #fff);
  border-color: var(--doc-cta-primary-bg, #111827);
}
.panel-title {
  font-size: 11px;
  font-weight: 900;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--doc-muted);
  margin-bottom: 12px;
}
.tabs { display: flex; gap: 8px; flex-wrap: wrap; }
.tab {
  border: 1px solid var(--doc-border);
  background: var(--doc-upload-surface);
  color: var(--doc-text);
  padding: 8px 10px;
  border-radius: 0;
  cursor: pointer;
  font-weight: 900;
}
.tab.active {
  background: var(--doc-cta-primary-bg, #111827);
  color: var(--doc-cta-primary-fg, #fff);
  border-color: var(--doc-cta-primary-bg, #111827);
}
.code {
  margin-top: 12px;
  padding: 12px;
  border-radius: 0;
  background: var(--doc-code-bg);
  border: 1px solid var(--doc-border);
  overflow: auto;
}
.grid {
  display: grid;
  grid-template-columns: minmax(0, 1.2fr) minmax(0, 0.8fr);
  gap: clamp(10px, 1.2vw, 14px);
}
.table { border: 1px solid var(--doc-border); border-radius: 0; overflow: hidden; }
.row { display: grid; grid-template-columns: 1fr 1.4fr; gap: 12px; padding: 10px 12px; border-bottom: 1px solid var(--doc-border); }
.row.head { background: var(--doc-upload-surface); color: var(--doc-muted); font-size: 11px; letter-spacing: 0.12em; text-transform: uppercase; font-weight: 900; }
.task { font-weight: 900; }
.mono { font-family: var(--doc-font-mono); }
.check { display: flex; align-items: center; gap: 10px; padding: 8px 0; color: var(--doc-text); }
.check input { accent-color: var(--doc-accent); }

@media (max-width: 1100px) {
  .grid { grid-template-columns: 1fr; }
}
.guidelines-md :deep(.md) {
  margin-top: 4px;
  border: 0;
  box-shadow: none;
  background: transparent;
  padding: 0;
}
.guidelines-md :deep(.md .md-codeblock),
.guidelines-md :deep(.md .doc-alert),
.guidelines-md :deep(.md .md-table-wrap) {
  box-shadow: var(--doc-shadow-soft);
}
.guidelines-md :deep(.md .md-codeblock),
.guidelines-md :deep(.md .doc-alert) {
  background: var(--doc-surface);
  border: 1px solid var(--doc-border);
}
.empty-md { margin: 0; color: var(--doc-muted); line-height: 1.6; }
.hero, .grid, #doc-contrib-guides { scroll-margin-top: 88px; }
</style>

