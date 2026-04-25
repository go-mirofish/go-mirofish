<template>
  <router-view />
  <Analytics />
</template>

<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useHead, useSeoMeta } from '@unhead/vue'
import { Analytics } from '@vercel/analytics/vue'
import {
  SITE_BASE,
  resolveSeoForRoute,
  getSoftwareApplicationJsonLd,
  getWebSiteJsonLd,
} from '@/seo/site.js'

const route = useRoute()
const seo = computed(() => resolveSeoForRoute(route))
const absoluteOgImage = () =>
  `${String(SITE_BASE).replace(/\/$/, '')}/og-image.png`

useSeoMeta({
  title: () => seo.value.title,
  description: () => seo.value.description,
  ogTitle: () => seo.value.title,
  ogDescription: () => seo.value.description,
  ogUrl: () => seo.value.absoluteUrl,
  ogType: 'website',
  ogImage: () => absoluteOgImage(),
  ogSiteName: 'go-mirofish',
  ogLocale: 'en_US',
  twitterCard: 'summary_large_image',
  twitterTitle: () => seo.value.title,
  twitterDescription: () => seo.value.description,
  twitterImage: () => absoluteOgImage(),
  keywords: () => seo.value.keywords,
  robots: () =>
    seo.value.noindex ? 'noindex, nofollow' : 'index, follow',
})

useHead({
  link: () => [
    { rel: 'canonical', key: 'canonical', href: seo.value.absoluteUrl },
  ],
  script: () => [
    {
      type: 'application/ld+json',
      key: 'ld-website',
      innerHTML: JSON.stringify(getWebSiteJsonLd()),
    },
    {
      type: 'application/ld+json',
      key: 'ld-software',
      innerHTML: JSON.stringify(getSoftwareApplicationJsonLd()),
    },
  ],
})
</script>

<style>
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

#app {
  min-width: 0;
  width: 100%;
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
