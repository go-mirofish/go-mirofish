<template>
  <div class="roadmap">
    <section id="doc-roadmap-hero" class="roadmap-hero">
      <div class="kicker">{{ $t('docs.roadmap.kicker') }}</div>
      <div class="title">{{ $t('docs.roadmap.title') }}</div>
      <p class="lead">{{ $t('docs.roadmap.lead') }}</p>
    </section>

    <section id="doc-roadmap-themes" class="roadmap-timeline" aria-labelledby="roadmap-h2">
      <h2 id="roadmap-h2" class="section-h">{{ $t('docs.roadmap.sectionThemes') }}</h2>
      <p class="section-sub">{{ $t('docs.roadmap.sectionThemesSub') }}</p>
      <ol class="steps" :aria-label="$t('docs.roadmap.timelineAria')">
        <li
          v-for="(step, i) in steps"
          :id="step.anchor"
          :key="step.anchor"
          class="step"
          :class="{ 'step--in': visible.has(step.anchor) }"
          :data-index="i"
          :ref="(el) => setStepRef(el, step.anchor)"
        >
          <div class="step-gutter" aria-hidden="true">
            <span class="step-dot" />
            <span v-if="i < steps.length - 1" class="step-rail" />
          </div>
          <div class="step-card">
            <div class="step-meta">
              <span class="status" :class="'status--' + step.statusKey">{{ $t('docs.roadmap.status.' + step.statusKey) }}</span>
              <span class="order mono">{{ $t('docs.roadmap.itemN', { n: i + 1, total: steps.length }) }}</span>
            </div>
            <h3 class="step-h">{{ $t('docs.roadmap.' + step.key + 'Title') }}</h3>
            <p class="step-body">{{ $t('docs.roadmap.' + step.key + 'Body') }}</p>
          </div>
        </li>
      </ol>
    </section>
  </div>
</template>

<script setup>
import { nextTick, onMounted, onUnmounted, ref } from 'vue'

const steps = [
  { key: 't1', anchor: 'doc-roadmap-t1', statusKey: 'ongoing' },
  { key: 't2', anchor: 'doc-roadmap-t2', statusKey: 'consider' },
  { key: 't3', anchor: 'doc-roadmap-t3', statusKey: 'near' },
  { key: 't4', anchor: 'doc-roadmap-t4', statusKey: 'future' },
  { key: 't5', anchor: 'doc-roadmap-t5', statusKey: 'direction' },
  { key: 't6', anchor: 'doc-roadmap-t6', statusKey: 'arch' },
]

const elByAnchor = new Map()
const visible = ref(new Set())
let io = null

function setStepRef(el, anchor) {
  if (el) elByAnchor.set(anchor, el)
  else elByAnchor.delete(anchor)
}

onMounted(() => {
  if (typeof IntersectionObserver === 'undefined') {
    visible.value = new Set(steps.map((s) => s.anchor))
    return
  }
  void nextTick(() => {
    io = new IntersectionObserver(
      (entries) => {
        const next = new Set(visible.value)
        for (const e of entries) {
          const id = e.target.id
          if (e.isIntersecting) next.add(id)
        }
        visible.value = next
      },
      { root: null, rootMargin: '0px 0px -12% 0px', threshold: [0.08, 0.2, 0.4] }
    )
    for (const s of steps) {
      const el = elByAnchor.get(s.anchor)
      if (el) io.observe(el)
    }
  })
})

onUnmounted(() => {
  if (io) {
    io.disconnect()
    io = null
  }
  elByAnchor.clear()
})
</script>

<style scoped>
.roadmap {
  display: flex;
  flex-direction: column;
  gap: clamp(1.5rem, 3vw, 2.5rem);
  max-width: 52rem;
  margin: 0 auto;
}

.roadmap-hero {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
}

.kicker {
  font-size: 0.7rem;
  font-weight: 800;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: var(--doc-muted, #6b7280);
}

.title {
  margin: 0;
  font-size: clamp(1.35rem, 2.8vw, 1.75rem);
  font-weight: 900;
  letter-spacing: -0.03em;
  line-height: 1.2;
  color: var(--doc-text, #0f172a);
}

.lead {
  margin: 0.25rem 0 0;
  font-size: 0.95rem;
  line-height: 1.65;
  color: var(--doc-muted, #475569);
  max-width: 48ch;
}

.section-h {
  margin: 0 0 0.35rem;
  font-size: 1.05rem;
  font-weight: 850;
  letter-spacing: -0.02em;
  color: var(--doc-text, #0f172a);
}

.section-sub {
  margin: 0 0 1rem;
  font-size: 0.9rem;
  line-height: 1.55;
  color: var(--doc-muted, #64748b);
  max-width: 52ch;
}

.roadmap-timeline {
  min-width: 0;
}

.steps {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 0;
}

.step {
  display: grid;
  grid-template-columns: 1.75rem 1fr;
  gap: 0.75rem 0.9rem;
  position: relative;
  opacity: 0;
  transform: translate3d(0, 10px, 0);
  transition:
    opacity 0.5s cubic-bezier(0.22, 1, 0.36, 1),
    transform 0.55s cubic-bezier(0.22, 1, 0.36, 1);
  transition-delay: calc(var(--i, 0) * 35ms);
}

.step--in {
  opacity: 1;
  transform: translate3d(0, 0, 0);
}

.step:nth-child(1) {
  --i: 0;
}
.step:nth-child(2) {
  --i: 1;
}
.step:nth-child(3) {
  --i: 2;
}
.step:nth-child(4) {
  --i: 3;
}
.step:nth-child(5) {
  --i: 4;
}
.step:nth-child(6) {
  --i: 5;
}

.step-gutter {
  display: flex;
  flex-direction: column;
  align-items: center;
  width: 1.75rem;
  min-height: 100%;
  padding-top: 0.35rem;
}

.step-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: color-mix(in srgb, var(--doc-text, #0f172a) 88%, var(--doc-muted, #94a3b8));
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--doc-surface, #fff) 90%, var(--doc-border, #e2e8f0));
  flex-shrink: 0;
}

.step-rail {
  flex: 1;
  width: 2px;
  min-height: 1.5rem;
  margin: 0.25rem 0 0.15rem;
  background: linear-gradient(180deg, var(--doc-border, #e2e8f0), color-mix(in srgb, var(--doc-border, #e2e8f0) 25%, transparent));
  border-radius: 1px;
}

.step-card {
  border: 1px solid var(--doc-border, #e2e8f0);
  border-radius: var(--doc-radius, 10px);
  background: var(--doc-surface, #fff);
  box-shadow: var(--doc-shadow-soft, 0 1px 2px rgba(15, 23, 42, 0.05));
  padding: 0.9rem 1rem 1rem;
  margin-bottom: 0.55rem;
}

.step-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 0.4rem 0.75rem;
  margin-bottom: 0.4rem;
}

.status {
  font-size: 0.65rem;
  font-weight: 800;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  border-radius: 999px;
  padding: 0.2em 0.6em;
}

.status--ongoing {
  color: #0369a1;
  background: color-mix(in srgb, #0ea5e9 12%, transparent);
}
.status--consider {
  color: #6d28d9;
  background: color-mix(in srgb, #8b5cf6 12%, transparent);
}
.status--near {
  color: #b45309;
  background: color-mix(in srgb, #f59e0b 14%, transparent);
}
.status--future {
  color: #475569;
  background: color-mix(in srgb, #64748b 10%, transparent);
}
.status--direction {
  color: #0f766e;
  background: color-mix(in srgb, #14b8a6 12%, transparent);
}
.status--arch {
  color: #1d4ed8;
  background: color-mix(in srgb, #3b82f6 10%, transparent);
}

.order {
  font-size: 0.7rem;
  color: var(--doc-muted, #94a3b8);
}

.step-h {
  margin: 0 0 0.45rem;
  font-size: 0.98rem;
  font-weight: 800;
  letter-spacing: -0.02em;
  line-height: 1.3;
  color: var(--doc-text, #0f172a);
}

.step-body {
  margin: 0;
  font-size: 0.9rem;
  line-height: 1.6;
  color: var(--doc-muted, #475569);
}

.mono {
  font-family: ui-monospace, 'Cascadia Code', 'Segoe UI Mono', monospace;
  font-size: 0.72em;
}

@media (prefers-reduced-motion: reduce) {
  .step {
    opacity: 1;
    transform: none;
    transition: none;
  }

  .step--in {
    opacity: 1;
  }
}
</style>
