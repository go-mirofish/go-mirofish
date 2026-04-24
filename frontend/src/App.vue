<template>
  <router-view />
  <Analytics />
</template>

<script setup>
import { Analytics } from '@vercel/analytics/vue'
</script>

<style>
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

#app {
  font-family: var(--doc-font-sans, 'JetBrains Mono', 'Space Grotesk', 'Noto Sans SC', monospace);
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  color: var(--doc-text, #111827);
  background-color: var(--doc-bg, #f4f4f5);
}

::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

::-webkit-scrollbar-track {
  background: var(--doc-bg, #f1f1f1);
}

::-webkit-scrollbar-thumb {
  background: var(--doc-muted, #6b7280);
}

::-webkit-scrollbar-thumb:hover {
  background: var(--doc-text, #374151);
}

html[data-theme='dark'] ::-webkit-scrollbar-thumb {
  background: var(--doc-console-scroll, #4b5563);
}
html[data-theme='dark'] ::-webkit-scrollbar-thumb:hover {
  background: var(--doc-muted, #9ca3af);
}

button {
  font-family: inherit;
}
</style>
