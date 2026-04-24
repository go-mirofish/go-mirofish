<template>
  <span class="pill" :class="`pill--${status}`">
    <span class="dot" aria-hidden="true"></span>
    <span class="text">{{ label }}</span>
  </span>
</template>

<script setup>
defineProps({
  label: { type: String, required: true },
  status: { type: String, default: 'pending' }, // pass | partial | fail | pending
})
</script>

<style scoped>
.pill {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px;
  border-radius: 0;
  border: 1px solid var(--doc-border);
  background: var(--doc-upload-surface);
  color: var(--doc-muted);
  font-weight: 800;
  font-size: 11px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}
.dot {
  width: 8px;
  height: 8px;
  border-radius: 0;
  background: var(--doc-dashed);
}
.pill--pass {
  background: var(--doc-badge-ok-bg);
  color: var(--doc-badge-ok-fg);
  border-color: color-mix(in srgb, var(--doc-badge-ok-fg) 28%, var(--doc-border));
}
.pill--pass .dot { background: var(--doc-badge-ok-fg); }

.pill--partial {
  background: color-mix(in srgb, #f59e0b 14%, var(--doc-surface));
  color: #f59e0b;
  border-color: color-mix(in srgb, #f59e0b 28%, var(--doc-border));
}
.pill--partial .dot { background: #f59e0b; }

.pill--fail {
  background: color-mix(in srgb, #dc2626 12%, var(--doc-surface));
  color: #dc2626;
  border-color: color-mix(in srgb, #dc2626 26%, var(--doc-border));
}
.pill--fail .dot { background: #dc2626; }
</style>

