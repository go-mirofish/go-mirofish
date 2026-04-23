<template>
  <div class="theme-toggle" ref="rootRef">
    <button
      type="button"
      class="theme-toggle__trigger"
      :aria-label="$t('theme.toggleAria')"
      :aria-expanded="open"
      @click="open = !open"
    >
      <span class="theme-toggle__icon" aria-hidden="true">{{ currentIcon }}</span>
      <span class="theme-toggle__label">{{ currentLabel }}</span>
      <span class="theme-toggle__caret" aria-hidden="true">{{ open ? '▲' : '▼' }}</span>
    </button>
    <ul v-if="open" class="theme-toggle__menu" role="listbox">
      <li
        v-for="opt in options"
        :key="opt.value"
        class="theme-toggle__option"
        :class="{ 'theme-toggle__option--active': opt.value === themePreference }"
        role="option"
        :aria-selected="opt.value === themePreference"
        @click="choose(opt.value)"
      >
        {{ opt.label }}
      </li>
    </ul>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { themePreference, setThemePreference } from '@/composables/theme.js'

const { t } = useI18n()
const open = ref(false)
const rootRef = ref(null)

const options = computed(() => [
  { value: 'light', label: t('theme.light'), icon: '☀' },
  { value: 'dark', label: t('theme.dark'), icon: '☽' },
  { value: 'system', label: t('theme.system'), icon: '◐' },
])

const currentLabel = computed(() => {
  const p = themePreference.value
  const f = options.value.find((o) => o.value === p)
  return f ? f.label : t('theme.system')
})

const currentIcon = computed(() => {
  const p = themePreference.value
  const f = options.value.find((o) => o.value === p)
  return f ? f.icon : '◐'
})

function choose(v) {
  setThemePreference(v)
  open.value = false
}

const onClickOutside = (e) => {
  if (rootRef.value && !rootRef.value.contains(e.target)) {
    open.value = false
  }
}

onMounted(() => {
  document.addEventListener('click', onClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('click', onClickOutside)
})
</script>

<style scoped>
.theme-toggle {
  position: relative;
  display: inline-block;
  font-family: var(--doc-font-sans, system-ui, sans-serif);
  font-size: 0.8rem;
}
.theme-toggle__trigger {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  background: transparent;
  color: var(--doc-text, #111827);
  border: 1px solid var(--doc-border, #e5e7eb);
  padding: 4px 10px;
  font: inherit;
  border-radius: 6px;
  cursor: pointer;
  transition: border-color 0.2s, color 0.2s, background 0.2s;
  max-width: 10rem;
}
.theme-toggle__trigger:hover {
  border-color: var(--doc-muted, #6b7280);
}
.theme-toggle__icon {
  font-size: 0.9rem;
  line-height: 1;
  opacity: 0.9;
}
.theme-toggle__label {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
  min-width: 0;
}
.theme-toggle__caret {
  font-size: 0.6rem;
  line-height: 1;
  opacity: 0.7;
  flex-shrink: 0;
}
.theme-toggle__menu {
  position: absolute;
  top: 100%;
  right: 0;
  margin: 4px 0 0;
  list-style: none;
  min-width: 9rem;
  padding: 4px 0;
  z-index: 2000;
  background: var(--doc-surface, #fff);
  border: 1px solid var(--doc-border, #e5e7eb);
  border-radius: 6px;
  box-shadow: 0 6px 20px rgba(0, 0, 0, 0.12);
}
html[data-theme='dark'] .theme-toggle__menu {
  box-shadow: 0 6px 24px rgba(0, 0, 0, 0.45);
}
.theme-toggle__option {
  padding: 6px 12px;
  color: var(--doc-text, #111827);
  cursor: pointer;
  transition: background 0.15s;
  white-space: nowrap;
}
.theme-toggle__option:hover {
  background: var(--doc-bg, #f4f4f5);
}
.theme-toggle__option--active {
  color: var(--doc-accent, #00add8);
  font-weight: 600;
}
</style>
