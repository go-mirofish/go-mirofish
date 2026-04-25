<template>
  <div class="report-preparing">
    <div class="report-preparing__intro">
      <h2 class="report-preparing__title">{{ $t('step4.reportBuildTitle') }}</h2>
      <p class="report-preparing__hint">{{ $t('step4.reportBuildHint') }}</p>
    </div>
    <div class="report-skeleton" aria-hidden="true">
      <div class="report-skeleton__meta">
        <span class="sk-line sk-line--tag" />
        <span class="sk-line sk-line--id" />
      </div>
      <span class="sk-line sk-line--headline" />
      <span class="sk-line sk-line--summary" />
      <span class="sk-line sk-line--summary sk-line--summary-short" />
      <div class="report-skeleton__rule" />
      <div
        v-for="n in 3"
        :key="n"
        class="report-skeleton__section"
      >
        <span class="sk-line sk-line--section-num" />
        <div class="report-skeleton__section-lines">
          <span class="sk-line sk-line--section-title" />
          <span class="sk-line sk-line--section-body" />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
</script>

<style scoped>
.report-preparing {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: stretch;
  justify-content: flex-start;
  gap: 24px;
  padding: 28px 32px 40px;
  min-height: 0;
  color: var(--doc-muted);
}

.report-preparing__intro {
  text-align: left;
}

.report-preparing__title {
  margin: 0 0 10px;
  font-size: 1.05rem;
  font-weight: 600;
  color: var(--doc-text);
  letter-spacing: -0.02em;
}

.report-preparing__hint {
  margin: 0;
  font-size: 13px;
  line-height: 1.55;
  color: var(--doc-muted);
  max-width: 42em;
}

.report-skeleton {
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 20px 0 0;
  border-top: 1px solid var(--doc-border);
}

.report-skeleton__meta {
  display: flex;
  align-items: center;
  gap: 12px;
}

.report-skeleton__rule {
  height: 1px;
  margin: 4px 0 8px;
  background: linear-gradient(
    90deg,
    var(--doc-border),
    transparent
  );
}

.report-skeleton__section {
  display: flex;
  gap: 14px;
  align-items: flex-start;
  padding: 12px 0;
  border-bottom: 1px solid color-mix(in srgb, var(--doc-border) 70%, transparent);
}

.report-skeleton__section:last-child {
  border-bottom: none;
}

.report-skeleton__section-lines {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
}

.sk-line {
  display: block;
  height: 10px;
  border-radius: 5px;
  position: relative;
  overflow: hidden;
  background: linear-gradient(
    100deg,
    var(--doc-border) 0%,
    color-mix(in srgb, var(--doc-text) 12%, var(--doc-border)) 48%,
    var(--doc-border) 100%
  );
  background-size: 200% 100%;
  animation: report-sk-shimmer 1.35s ease-in-out infinite;
}

.sk-line--tag {
  width: 120px;
  height: 22px;
  border-radius: 4px;
}

.sk-line--id {
  width: 180px;
  height: 12px;
  margin-top: 2px;
  opacity: 0.75;
}

.sk-line--headline {
  width: 72%;
  max-width: 480px;
  height: 22px;
  margin-top: 4px;
}

.sk-line--summary {
  width: 100%;
}

.sk-line--summary-short {
  width: 55%;
  opacity: 0.9;
}

.sk-line--section-num {
  width: 28px;
  height: 14px;
  flex-shrink: 0;
  margin-top: 4px;
  opacity: 0.7;
}

.sk-line--section-title {
  width: 58%;
  height: 14px;
}

.sk-line--section-body {
  width: 100%;
  height: 9px;
  opacity: 0.65;
}

@keyframes report-sk-shimmer {
  0% {
    background-position: 100% 0;
  }
  100% {
    background-position: -100% 0;
  }
}
</style>
