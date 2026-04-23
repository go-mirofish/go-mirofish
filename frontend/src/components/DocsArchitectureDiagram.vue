<template>
  <figure class="docs-arch" :aria-label="$t('docs.overviewArchFigureAria')">
    <h2 class="docs-h2 docs-h2--arch">{{ $t('docs.overviewArchTitle') }}</h2>
    <div class="docs-arch__inner">
      <svg
        class="docs-arch__svg"
        viewBox="0 0 560 256"
        xmlns="http://www.w3.org/2000/svg"
        role="img"
        focusable="false"
      >
        <defs>
          <marker
            id="arch-arrow"
            viewBox="0 0 10 10"
            refX="8"
            refY="5"
            markerWidth="5"
            markerHeight="5"
            orient="auto"
          >
            <path d="M 0 0 L 10 5 L 0 10 z" class="docs-arch__arrowhead" />
          </marker>
        </defs>

        <!-- Browser -->
        <rect x="190" y="4" width="180" height="40" rx="6" class="docs-arch__box" />
        <text x="280" y="30" text-anchor="middle" class="docs-arch__t">{{ $t('docs.overviewArchBrowser') }}</text>

        <line x1="280" y1="44" x2="280" y2="56" class="docs-arch__line" marker-end="url(#arch-arrow)" />
        <text x="288" y="54" class="docs-arch__hint">{{ $t('docs.overviewArchHttp') }}</text>

        <!-- Web + optional gateway -->
        <rect
          x="120"
          y="60"
          width="320"
          height="48"
          rx="6"
          class="docs-arch__box docs-arch__box--emph"
        />
        <text x="280" y="86" text-anchor="middle" class="docs-arch__t">
          {{ $t('docs.overviewArchWebLayer') }}
        </text>
        <text x="280" y="100" text-anchor="middle" class="docs-arch__t docs-arch__t--muted">
          {{ $t('docs.overviewArchGoNote') }}
        </text>

        <line x1="280" y1="108" x2="280" y2="120" class="docs-arch__line" marker-end="url(#arch-arrow)" />
        <text x="288" y="118" class="docs-arch__hint">{{ $t('docs.overviewArchApiPath') }}</text>

        <!-- Flask -->
        <rect x="160" y="124" width="240" height="40" rx="6" class="docs-arch__box" />
        <text x="280" y="150" text-anchor="middle" class="docs-arch__t">
          {{ $t('docs.overviewArchFlask') }}
        </text>

        <!-- Fork: vertical then horizontal then down -->
        <line x1="280" y1="164" x2="280" y2="180" class="docs-arch__line" />
        <line x1="150" y1="180" x2="410" y2="180" class="docs-arch__line" />
        <line x1="150" y1="180" x2="150" y2="196" class="docs-arch__line" marker-end="url(#arch-arrow)" />
        <line x1="410" y1="180" x2="410" y2="196" class="docs-arch__line" marker-end="url(#arch-arrow)" />

        <rect x="80" y="200" width="140" height="48" rx="6" class="docs-arch__box" />
        <text x="150" y="224" text-anchor="middle" class="docs-arch__t">{{ $t('docs.overviewArchLlm') }}</text>
        <text x="150" y="240" text-anchor="middle" class="docs-arch__t docs-arch__t--tiny">
          {{ $t('docs.overviewArchLlmSub') }}
        </text>

        <rect x="340" y="200" width="140" height="48" rx="6" class="docs-arch__box" />
        <text x="410" y="224" text-anchor="middle" class="docs-arch__t">{{ $t('docs.overviewArchZep') }}</text>
        <text x="410" y="240" text-anchor="middle" class="docs-arch__t docs-arch__t--tiny">
          {{ $t('docs.overviewArchZepSub') }}
        </text>
      </svg>
    </div>
  </figure>
</template>

<style scoped>
.docs-arch {
  margin: 0 0 1.5rem;
  max-width: 100%;
}
.docs-h2--arch {
  margin-top: 0;
  margin-bottom: 0.75rem;
  border-top: none;
  padding-top: 0;
}
.docs-h2--arch::before {
  content: none;
}
.docs-arch__inner {
  border: 1px solid var(--doc-border, #e5e7eb);
  border-radius: 8px;
  background: var(--doc-bg, #f4f4f5);
  padding: 0.75rem 0.5rem 0.5rem;
  overflow-x: auto;
}
.docs-arch__svg {
  display: block;
  width: 100%;
  min-width: 320px;
  max-width: 680px;
  height: auto;
  margin: 0 auto;
}
.docs-arch__box {
  fill: var(--doc-surface, #fff);
  stroke: var(--doc-border, #e5e7eb);
  stroke-width: 1.5;
}
.docs-arch__box--emph {
  fill: var(--doc-code-bg, #f3f4f6);
  stroke: var(--doc-accent, #00add8);
  stroke-width: 1.5;
}
html[data-theme='dark'] .docs-arch__box--emph {
  fill: color-mix(in srgb, var(--doc-accent, #4dd0f5) 12%, var(--doc-surface, #12141a));
}
.docs-arch__t {
  fill: var(--doc-text, #111827);
  font-size: 12px;
  font-family: var(--doc-font-sans, system-ui, sans-serif);
  font-weight: 600;
  pointer-events: none;
}
.docs-arch__t--muted {
  font-size: 10px;
  font-weight: 500;
  fill: var(--doc-muted, #6b7280);
}
.docs-arch__t--tiny {
  font-size: 9px;
  font-weight: 500;
  fill: var(--doc-muted, #6b7280);
}
.docs-arch__line {
  stroke: var(--doc-muted, #6b7280);
  stroke-width: 1.25;
  fill: none;
}
.docs-arch__arrowhead {
  fill: var(--doc-muted, #6b7280);
}
.docs-arch__hint {
  font-size: 8px;
  font-family: var(--doc-font-mono, ui-monospace, monospace);
  fill: var(--doc-muted, #6b7280);
  pointer-events: none;
}
</style>
