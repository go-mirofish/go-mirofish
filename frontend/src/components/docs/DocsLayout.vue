<template>
  <div class="docs-shell">
    <header class="docs-topbar">
      <div class="topbar-left">
        <router-link to="/" class="brand">go-mirofish</router-link>
        <span class="version-pill">v{{ appVersion }}</span>
      </div>

      <div class="topbar-center">
        <span class="title">{{ title }}</span>
      </div>

      <div class="topbar-right">
        <router-link to="/" class="back-link">{{ $t('docs.backToApp') }}</router-link>
        <ThemeToggle />
        <LanguageSwitcher />
        <a
          class="gh-link"
          href="https://github.com/go-mirofish/go-mirofish"
          target="_blank"
          rel="noopener noreferrer"
        >
          {{ $t('nav.visitGithub') }} ↗
        </a>
      </div>
    </header>

    <div class="docs-body" :class="{ 'docs-body--notoc': !hasToc }">
      <aside class="sidebar">
        <slot name="sidebar" />
      </aside>

      <main class="content">
        <slot />

        <footer class="docs-footer">
          <div class="footer-left">
            <slot name="footer-left" />
          </div>
          <div class="footer-right">
            <slot name="footer-right" />
          </div>
        </footer>
      </main>

      <aside v-if="hasToc" class="toc">
        <slot name="toc" />
      </aside>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import LanguageSwitcher from '../LanguageSwitcher.vue'
import ThemeToggle from '../ThemeToggle.vue'

const props = defineProps({
  title: { type: String, default: '' },
  hasToc: { type: Boolean, default: false },
})

const appVersion = computed(() => {
  // Vite exposes package version as import.meta.env only if configured.
  // Fall back to the package.json version injected at build time via dependency (go-mirofish).
  return (import.meta?.env?.VITE_APP_VERSION || '0.1.0').toString()
})
</script>

<style scoped>
.docs-shell {
  min-height: 100vh;
  background: var(--doc-bg);
  color: var(--doc-text);
  font-family: var(--doc-font-sans);
  /* Shared docs rail: sidebar + TOC use the same padding + hit targets (DocsSidebar / DocsToc). */
  --doc-rail-pad-x: 12px;
  --doc-rail-pad-y: 12px;
  --doc-rail-item-px: 10px;
  --doc-rail-item-py: 8px;
  --doc-rail-radius: 2px;
  --doc-main-pad-x: clamp(12px, 2.2vw, 28px);
  --doc-main-pad-b: clamp(12px, 1.2vw, 20px);
  --doc-main-pad-t: clamp(10px, 1vw, 16px);
}

.docs-topbar {
  position: sticky;
  top: 0;
  z-index: 50;
  display: grid;
  grid-template-columns: 1fr auto 1fr;
  align-items: center;
  gap: 16px;
  padding: 12px 20px;
  background: var(--doc-topbar-surface);
  border-bottom: 1px solid var(--doc-border);
  backdrop-filter: blur(10px);
  -webkit-backdrop-filter: blur(10px);
}

.topbar-left,
.topbar-right {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.topbar-right {
  justify-content: flex-end;
}

.brand {
  font-family: var(--doc-font-mono);
  font-weight: 800;
  letter-spacing: -0.02em;
  color: var(--doc-text);
  text-decoration: none;
}

.version-pill {
  font-family: var(--doc-font-mono);
  font-size: 12px;
  padding: 4px 10px;
  border-radius: 0;
  border: 1px solid var(--doc-border);
  background: var(--doc-surface);
  color: var(--doc-muted);
}

.title {
  font-size: 13px;
  font-weight: 700;
  color: var(--doc-muted);
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.back-link {
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--doc-muted);
  text-decoration: none;
  padding: 8px 10px;
  border-radius: 0;
  border: 1px solid var(--doc-border);
  background: color-mix(in srgb, var(--doc-surface) 70%, transparent);
}
.back-link:hover {
  color: var(--doc-text);
  border-color: color-mix(in srgb, var(--doc-text) 18%, var(--doc-border));
}

.gh-link {
  font-size: 12px;
  color: var(--doc-muted);
  text-decoration: none;
  padding: 6px 8px;
  border-radius: 0;
}
.gh-link:hover {
  color: var(--doc-text);
  background: color-mix(in srgb, var(--doc-text) 6%, transparent);
}

.docs-body {
  display: grid;
  grid-template-columns: minmax(200px, 240px) minmax(0, 1fr) minmax(176px, 220px);
  gap: 0;
  max-width: min(1400px, 100%);
  margin: 0 auto;
  padding: clamp(12px, 1.5vw, 20px) clamp(8px, 1.2vw, 16px) clamp(20px, 2vw, 32px);
  width: 100%;
  box-sizing: border-box;
  align-items: stretch;
}

.docs-body--notoc {
  grid-template-columns: minmax(200px, 240px) minmax(0, 1fr);
  max-width: min(1200px, 100%);
}

/* One connected “sheet”: no gutter between columns; single vertical hairlines. */
.sidebar {
  position: sticky;
  top: 78px;
  align-self: start;
  height: calc(100vh - 92px);
  overflow: auto;
  border: 1px solid var(--doc-border);
  border-right: 1px solid var(--doc-border);
  background: var(--doc-surface);
  border-radius: var(--doc-radius) 0 0 var(--doc-radius);
}

.content {
  min-width: 0;
  max-width: 100%;
  overflow-x: auto;
  overflow-y: visible;
  padding: var(--doc-main-pad-t) var(--doc-main-pad-x) var(--doc-main-pad-b);
  box-sizing: border-box;
  border: 1px solid var(--doc-border);
  border-left: 0;
  border-right: 1px solid var(--doc-border);
  background: var(--doc-surface);
  border-radius: 0;
}

.toc {
  position: sticky;
  top: 78px;
  align-self: start;
  height: calc(100vh - 92px);
  overflow: auto;
  border: 1px solid var(--doc-border);
  border-left: 0;
  background: var(--doc-surface);
  border-radius: 0 var(--doc-radius) var(--doc-radius) 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.docs-body--notoc .content {
  border-right: 1px solid var(--doc-border);
  border-radius: 0 var(--doc-radius) var(--doc-radius) 0;
}

.docs-footer {
  margin-top: 28px;
  padding-top: 18px;
  border-top: 1px solid var(--doc-border);
  display: flex;
  gap: 12px;
  justify-content: space-between;
  color: var(--doc-muted);
  font-size: 12px;
}

@media (max-width: 1200px) {
  .docs-body {
    grid-template-columns: minmax(200px, 220px) minmax(0, 1fr) minmax(160px, 200px);
  }
}

@media (max-width: 1100px) {
  .docs-body {
    grid-template-columns: 220px minmax(0, 1fr);
  }
  .toc {
    display: none;
  }
  /* TOC slot hidden: close the “sheet” on the right of main. */
  .docs-body .content {
    border-right: 1px solid var(--doc-border);
    border-radius: 0 var(--doc-radius) var(--doc-radius) 0;
  }
}

@media (max-width: 900px) {
  .docs-body {
    grid-template-columns: 200px minmax(0, 1fr);
  }
}

@media (max-width: 860px) {
  .docs-topbar {
    grid-template-columns: 1fr 1fr;
    grid-template-areas:
      'left right'
      'center center';
  }
  .topbar-left { grid-area: left; }
  .topbar-right { grid-area: right; }
  .topbar-center { grid-area: center; justify-self: center; }

  .docs-body {
    grid-template-columns: 1fr;
    gap: 12px;
    padding-left: 12px;
    padding-right: 12px;
  }
  .sidebar,
  .content,
  .toc {
    border: 1px solid var(--doc-border);
    border-radius: var(--doc-radius);
  }
  .sidebar {
    position: relative;
    top: auto;
    height: auto;
    max-height: none;
  }
}
</style>

