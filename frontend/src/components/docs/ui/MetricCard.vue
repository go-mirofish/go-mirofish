<template>
  <div class="card">
    <div class="label">{{ label }}</div>
    <div class="value mono">{{ value }}</div>
    <div v-if="hint" class="hint">{{ hint }}</div>
  </div>
</template>

<script setup>
defineProps({
  label: { type: String, required: true },
  value: { type: String, required: true },
  hint: { type: String, default: '' },
})
</script>

<style scoped>
.card {
  display: flex;
  flex-direction: column;
  align-items: stretch;
  justify-content: flex-start;
  border: 1px solid var(--doc-border);
  background: var(--doc-surface);
  border-radius: var(--doc-radius, 0);
  padding: 12px 14px 14px;
  box-shadow: var(--doc-shadow-soft);
  min-width: 0;
  max-width: 100%;
  box-sizing: border-box;
}
.label {
  font-size: 10px;
  font-weight: 800;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: var(--doc-muted);
  line-height: 1.25;
  word-wrap: break-word;
  overflow-wrap: anywhere;
}
.value {
  margin-top: 8px;
  font-size: clamp(14px, 2.6vw, 22px);
  font-weight: 900;
  color: var(--doc-text);
  line-height: 1.2;
  word-break: break-word;
  overflow-wrap: anywhere;
  font-variant-numeric: tabular-nums;
}
.hint {
  margin-top: 8px;
  font-size: 11px;
  color: var(--doc-muted);
  line-height: 1.45;
  word-wrap: break-word;
}
.mono { font-family: var(--doc-font-mono); }
</style>

