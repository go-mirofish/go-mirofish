<template>
  <nav class="nav" :aria-label="$t('docs.sidebarAria')">
    <div v-for="group in groups" :key="group.key" class="group">
      <div class="group-title">{{ $t(group.titleKey) }}</div>
      <router-link
        v-for="entry in group.entries"
        :key="entry.key"
        class="item"
        :to="entry.path"
        :class="{ active: entry.path === activePath }"
      >
        {{ $t(entry.titleKey) }}
      </router-link>
    </div>
  </nav>
</template>

<script setup>
defineProps({
  groups: { type: Array, required: true },
  activePath: { type: String, required: true },
})
</script>

<style scoped>
.nav {
  padding: var(--doc-rail-pad-y, 12px) var(--doc-rail-pad-x, 12px);
  box-sizing: border-box;
}

.group + .group {
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid var(--doc-border);
}

.group-title {
  font-size: 11px;
  font-weight: 800;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--doc-muted);
  margin: 0 0 8px;
  padding: 0 2px;
}

.item {
  display: block;
  padding: var(--doc-rail-item-py, 8px) var(--doc-rail-item-px, 10px);
  border-radius: var(--doc-rail-radius, 2px);
  color: var(--doc-text);
  text-decoration: none;
  border: 1px solid transparent;
  font-weight: 600;
  font-size: 13px;
  line-height: 1.35;
  box-sizing: border-box;
}

.item:hover {
  background: color-mix(in srgb, var(--doc-text) 6%, transparent);
  border-color: color-mix(in srgb, var(--doc-text) 12%, var(--doc-border));
}

.item.active {
  background: color-mix(in srgb, var(--doc-accent) 12%, var(--doc-surface));
  border-color: color-mix(in srgb, var(--doc-accent) 32%, var(--doc-border));
}
</style>

