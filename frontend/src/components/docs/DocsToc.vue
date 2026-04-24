<template>
  <div class="toc-wrap">
    <div class="toc-title">{{ $t('docs.onThisPage') }}</div>
    <a
      v-for="h in headings"
      :key="h.id"
      class="toc-item"
      :class="[`lvl-${h.level}`, { active: h.id === activeId }]"
      :href="`#${h.id}`"
      @click.prevent="scrollTo(h.id)"
    >
      {{ h.text }}
    </a>
  </div>
</template>

<script setup>
const props = defineProps({
  headings: { type: Array, default: () => [] },
  activeId: { type: String, default: '' },
})

function scrollTo(id) {
  const el = document.getElementById(id)
  if (!el) return
  el.scrollIntoView({ behavior: 'smooth', block: 'start' })
  history.replaceState(null, '', `#${id}`)
}
</script>

<style scoped>
.toc-wrap {
  padding: var(--doc-rail-pad-y, 12px) var(--doc-rail-pad-x, 12px);
  box-sizing: border-box;
  flex: 1;
  min-height: 0;
  overflow: auto;
  -webkit-overflow-scrolling: touch;
}

.toc-title {
  font-size: 11px;
  font-weight: 800;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--doc-muted);
  margin: 0 0 8px;
  padding: 0 2px;
  flex-shrink: 0;
}

.toc-item {
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 3;
  line-clamp: 3;
  text-decoration: none;
  color: var(--doc-muted);
  font-size: 12px;
  line-height: 1.45;
  padding: var(--doc-rail-item-py, 8px) var(--doc-rail-item-px, 10px);
  border-radius: var(--doc-rail-radius, 2px);
  border: 1px solid transparent;
  overflow: hidden;
  word-break: break-word;
  hyphens: auto;
  box-sizing: border-box;
}

.toc-item:hover {
  color: var(--doc-text);
  background: color-mix(in srgb, var(--doc-text) 6%, transparent);
  border-color: color-mix(in srgb, var(--doc-text) 10%, var(--doc-border));
}

.toc-item.active {
  color: var(--doc-text);
  border-color: color-mix(in srgb, var(--doc-accent) 28%, var(--doc-border));
  background: color-mix(in srgb, var(--doc-accent) 12%, var(--doc-surface));
}

.toc-item.lvl-3 {
  padding-left: calc(var(--doc-rail-item-px, 10px) + 10px);
}
.toc-item.lvl-4 {
  padding-left: calc(var(--doc-rail-item-px, 10px) + 18px);
}
</style>

