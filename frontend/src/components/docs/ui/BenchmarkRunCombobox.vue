<template>
  <div ref="rootRef" class="bench-combo">
    <span class="bench-combo__label">{{ $t('docs.bench.selectRun') }}</span>
    <div class="bench-combo__control" :class="{ 'bench-combo__control--open': open }">
      <button
        type="button"
        class="bench-combo__trigger"
        :aria-expanded="open"
        aria-haspopup="listbox"
        :disabled="!options.length"
        @click="toggle"
      >
        <span class="bench-combo__value">{{ currentLabel || '-' }}</span>
        <span class="bench-combo__chev" aria-hidden="true">▾</span>
      </button>
      <div v-show="open && options.length" class="bench-combo__panel" :aria-label="$t('docs.bench.searchRuns')">
        <input
          ref="searchRef"
          v-model="query"
          class="bench-combo__search"
          type="search"
          :placeholder="$t('docs.bench.searchRunsPlaceholder')"
          autocomplete="off"
          @keydown.escape.prevent="close"
        />
        <ul class="bench-combo__list" role="listbox" tabindex="-1">
          <li
            v-for="opt in filtered"
            :key="opt.key"
            class="bench-combo__item"
            :class="{ 'bench-combo__item--active': opt.key === modelValue }"
            role="option"
            :aria-selected="opt.key === modelValue"
            @mousedown.prevent
            @click="pick(opt.key)"
          >
            <span class="bench-combo__item-text">{{ opt.label }}</span>
          </li>
        </ul>
        <p v-if="!filtered.length" class="bench-combo__empty">{{ $t('docs.bench.searchRunsNoMatch') }}</p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'

const props = defineProps({
  modelValue: { type: String, default: '' },
  options: { type: Array, default: () => [] },
  /** (opt) => string full-text search */
  getSearchText: { type: Function, default: (o) => o.search || o.label || '' },
})

const emit = defineEmits(['update:modelValue'])

const open = ref(false)
const query = ref('')
const rootRef = ref(null)
const searchRef = ref(null)

const current = computed(() => props.options.find((o) => o.key === props.modelValue))
const currentLabel = computed(() => current.value?.label || '')

const filtered = computed(() => {
  const q = query.value.trim().toLowerCase()
  if (!q) return props.options
  return props.options.filter((o) => (props.getSearchText(o) || '').toLowerCase().includes(q))
})

function close() {
  open.value = false
  query.value = ''
}

function toggle() {
  if (!props.options.length) return
  open.value = !open.value
  if (open.value) {
    nextTick(() => {
      searchRef.value?.focus()
      searchRef.value?.select?.()
    })
  } else {
    query.value = ''
  }
}

function pick(key) {
  emit('update:modelValue', key)
  close()
}

function onDocClick(e) {
  const el = rootRef.value
  if (!el || !open.value) return
  if (el.contains(e.target)) return
  close()
}

onMounted(() => {
  document.addEventListener('click', onDocClick, true)
})
onBeforeUnmount(() => {
  document.removeEventListener('click', onDocClick, true)
})

watch(
  () => props.modelValue,
  () => {
    if (open.value) close()
  }
)
</script>

<style scoped>
.bench-combo {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
  width: 100%;
  max-width: 100%;
}

.bench-combo__label {
  font-size: 11px;
  font-weight: 800;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--doc-muted);
}

.bench-combo__control {
  position: relative;
  min-width: 0;
}

.bench-combo__trigger {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  width: 100%;
  min-width: 0;
  min-height: 40px;
  padding: 8px 12px;
  box-sizing: border-box;
  border: 1px solid var(--doc-border);
  border-radius: 0;
  background: var(--doc-surface);
  color: var(--doc-text);
  font: inherit;
  text-align: left;
  cursor: pointer;
}
.bench-combo__trigger:hover:not(:disabled) {
  border-color: color-mix(in srgb, var(--doc-accent) 25%, var(--doc-border));
}
.bench-combo__trigger:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

.bench-combo__value {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
  font-weight: 600;
}
.bench-combo__chev {
  flex-shrink: 0;
  font-size: 10px;
  color: var(--doc-muted);
}
.bench-combo__control--open .bench-combo__chev {
  transform: rotate(180deg);
}

.bench-combo__panel {
  position: absolute;
  z-index: 40;
  top: calc(100% + 4px);
  left: 0;
  right: 0;
  min-width: min(100%, 420px);
  max-height: min(50vh, 360px);
  display: flex;
  flex-direction: column;
  border: 1px solid var(--doc-border);
  background: var(--doc-surface);
  box-shadow: var(--doc-shadow-soft);
}

.bench-combo__search {
  border: 0;
  border-bottom: 1px solid var(--doc-border);
  padding: 10px 12px;
  font: inherit;
  font-size: 13px;
  background: color-mix(in srgb, var(--doc-surface) 90%, var(--doc-bg));
  color: var(--doc-text);
  box-sizing: border-box;
  width: 100%;
  outline: none;
}
.bench-combo__search::placeholder {
  color: var(--doc-muted);
}
.bench-combo__search:focus {
  background: var(--doc-surface);
}

.bench-combo__list {
  list-style: none;
  margin: 0;
  padding: 6px 0;
  overflow-y: auto;
  flex: 1;
  min-height: 0;
}

.bench-combo__item {
  margin: 0;
  padding: 0;
}
.bench-combo__item-text {
  display: block;
  padding: 8px 12px;
  font-size: 12px;
  line-height: 1.4;
  cursor: pointer;
  color: var(--doc-text);
  word-break: break-word;
}
.bench-combo__item:hover .bench-combo__item-text {
  background: color-mix(in srgb, var(--doc-text) 6%, transparent);
}
.bench-combo__item--active .bench-combo__item-text {
  background: color-mix(in srgb, var(--doc-accent) 12%, var(--doc-surface));
  border-left: 2px solid var(--doc-accent);
  padding-left: 10px;
}

.bench-combo__empty {
  margin: 0;
  padding: 12px;
  font-size: 12px;
  color: var(--doc-muted);
}
</style>
