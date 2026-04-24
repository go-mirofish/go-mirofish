<template>
  <figure class="docs-arch" :aria-label="$t('docs.overviewArchFigureAria')">
    <h2 class="docs-h2 docs-h2--arch">{{ $t('docs.overviewArchTitle') }}</h2>
    <div class="docs-arch__inner">
      <svg
        class="docs-arch__svg"
        viewBox="0 0 520 300"
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

        <!-- 1. Browser -->
        <rect x="180" y="2" width="160" height="30" class="docs-arch__box" />
        <text x="260" y="22" text-anchor="middle" class="docs-arch__t">{{ $t('docs.overviewArchBrowser') }}</text>

        <line x1="260" y1="32" x2="260" y2="42" class="docs-arch__line" marker-end="url(#arch-arrow)" />
        <text x="268" y="40" class="docs-arch__hint">{{ $t('docs.overviewArchHttp') }}</text>

        <!-- 2. Vue + Vite -->
        <rect x="140" y="46" width="240" height="34" class="docs-arch__box" />
        <text x="260" y="67" text-anchor="middle" class="docs-arch__t">{{ $t('docs.overviewArchWebLayer') }}</text>

        <line x1="260" y1="80" x2="260" y2="90" class="docs-arch__line" marker-end="url(#arch-arrow)" />
        <text x="268" y="88" class="docs-arch__hint">{{ $t('docs.overviewArchProxy') }}</text>

        <!-- 3. Go gateway (control plane) -->
        <rect
          x="100"
          y="94"
          width="320"
          height="44"
          class="docs-arch__box docs-arch__box--emph"
        />
        <text x="260" y="115" text-anchor="middle" class="docs-arch__t">{{ $t('docs.overviewArchGateway') }}</text>
        <text x="260" y="130" text-anchor="middle" class="docs-arch__t docs-arch__t--muted">
          {{ $t('docs.overviewArchGatewaySub') }}
        </text>

        <line x1="260" y1="138" x2="260" y2="150" class="docs-arch__line" marker-end="url(#arch-arrow)" />
        <text x="268" y="148" class="docs-arch__hint">{{ $t('docs.overviewArchBackendPath') }}</text>

        <!-- 4. Python API -->
        <rect x="140" y="154" width="240" height="34" class="docs-arch__box" />
        <text x="260" y="175" text-anchor="middle" class="docs-arch__t">{{ $t('docs.overviewArchFlask') }}</text>

        <line x1="260" y1="188" x2="260" y2="198" class="docs-arch__line" />
        <line x1="130" y1="198" x2="390" y2="198" class="docs-arch__line" />
        <line x1="150" y1="198" x2="150" y2="208" class="docs-arch__line" marker-end="url(#arch-arrow)" />
        <line x1="370" y1="198" x2="370" y2="208" class="docs-arch__line" marker-end="url(#arch-arrow)" />

        <rect x="60" y="210" width="180" height="38" class="docs-arch__box" />
        <text x="150" y="230" text-anchor="middle" class="docs-arch__t">{{ $t('docs.overviewArchLlm') }}</text>
        <text x="150" y="244" text-anchor="middle" class="docs-arch__t docs-arch__t--tiny">
          {{ $t('docs.overviewArchLlmSub') }}
        </text>

        <rect x="280" y="210" width="180" height="38" class="docs-arch__box" />
        <text x="370" y="230" text-anchor="middle" class="docs-arch__t">{{ $t('docs.overviewArchZep') }}</text>
        <text x="370" y="244" text-anchor="middle" class="docs-arch__t docs-arch__t--tiny">
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
  border-radius: 0;
  background: var(--doc-bg, #f4f4f5);
  padding: 0.75rem 0.5rem 0.5rem;
  overflow-x: auto;
}
.docs-arch__svg {
  display: block;
  width: 100%;
  min-width: 300px;
  max-width: 640px;
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
  font-size: 11px;
  font-family: var(--doc-font-sans, system-ui, sans-serif);
  font-weight: 600;
  pointer-events: none;
}
.docs-arch__t--muted {
  font-size: 9px;
  font-weight: 500;
  fill: var(--doc-muted, #6b7280);
}
.docs-arch__t--tiny {
  font-size: 8px;
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
