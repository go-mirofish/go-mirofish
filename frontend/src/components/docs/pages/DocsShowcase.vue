<template>
  <div class="showcase">
    <section id="doc-show-hero" class="hero">
      <div class="hero-text">
        <div class="kicker">{{ $t('docs.showcase.kicker') }}</div>
        <div class="title">{{ $t('docs.showcase.title') }}</div>
        <div class="sub">{{ $t('docs.showcase.subtitle') }}</div>
        <div class="cta">
          <router-link class="btn primary" to="/docs/installation">{{ $t('docs.showcase.ctaRunLocal') }}</router-link>
          <a class="btn" href="https://gomirofish.vercel.app" target="_blank" rel="noopener noreferrer">{{ $t('docs.showcase.ctaViewDemo') }} ↗</a>
        </div>
      </div>
      <div class="hero-card">
        <div class="card-title">{{ $t('docs.showcase.flowTitle') }}</div>
        <div class="steps">
          <div v-for="s in steps" :key="s.no" class="step">
            <div class="no mono">{{ s.no }}</div>
            <div class="label">{{ s.label }}</div>
          </div>
        </div>
      </div>
    </section>

    <section id="doc-show-highlights" class="features">
      <div class="section-title">{{ $t('docs.showcase.featuresTitle') }}</div>
      <div class="grid">
        <div v-for="f in features" :key="f.title" class="feature">
          <div class="ft">{{ f.title }}</div>
          <div class="fd">{{ f.desc }}</div>
        </div>
      </div>
    </section>

    <section id="doc-show-shots" class="gallery">
      <div class="section-title">{{ $t('docs.showcase.galleryTitle') }}</div>
      <div class="shots">
        <button
          v-for="(s, idx) in screenshots"
          :key="'shot-' + s.n"
          class="shot"
          type="button"
          @click="open(idx)"
        >
          <img :src="s.src" :alt="s.caption" loading="lazy" decoding="async" />
          <div class="cap"><span class="mono">{{ s.step }}</span> — {{ s.caption }}</div>
        </button>
      </div>
    </section>

    <section id="doc-show-faq" class="faq">
      <div class="section-title">{{ $t('docs.showcase.faqTitle') }}</div>
      <details v-for="q in faq" :key="q.q" class="q">
        <summary>{{ q.q }}</summary>
        <div class="a">{{ q.a }}</div>
      </details>
    </section>

    <div v-if="lightbox.open" class="lb" @click.self="close">
      <button class="lb-close" type="button" @click="close">×</button>
      <button class="lb-nav lb-prev" type="button" @click="prev">‹</button>
      <img class="lb-img" :src="screenshots[lightbox.index].src" :alt="screenshots[lightbox.index].caption" />
      <button class="lb-nav lb-next" type="button" @click="next">›</button>
      <div class="lb-cap">{{ screenshots[lightbox.index].caption }}</div>
    </div>
  </div>
</template>

<script setup>
import { computed, reactive } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

/** Public assets: frontend/public/static/image/Screenshot/Screenshot(n).png */
function screenshotUrl(n) {
  const base = import.meta.env.BASE_URL || '/'
  const prefix = base.endsWith('/') ? base : `${base}/`
  return `${prefix}static/image/Screenshot/Screenshot(${n}).png`
}

const steps = computed(() => [
  { no: '01', label: t('docs.showcase.step1') },
  { no: '02', label: t('docs.showcase.step2') },
  { no: '03', label: t('docs.showcase.step3') },
  { no: '04', label: t('docs.showcase.step4') },
  { no: '05', label: t('docs.showcase.step5') },
])

const features = computed(() => [
  { title: t('docs.showcase.fGraphTitle'), desc: t('docs.showcase.fGraphDesc') },
  { title: t('docs.showcase.fSimTitle'), desc: t('docs.showcase.fSimDesc') },
  { title: t('docs.showcase.fReportTitle'), desc: t('docs.showcase.fReportDesc') },
  { title: t('docs.showcase.fChatTitle'), desc: t('docs.showcase.fChatDesc') },
  { title: t('docs.showcase.fHistoryTitle'), desc: t('docs.showcase.fHistoryDesc') },
  { title: t('docs.showcase.fLocalTitle'), desc: t('docs.showcase.fLocalDesc') },
])

const shotMeta = [
  { n: 1, step: '01' },
  { n: 2, step: '03' },
  { n: 3, step: '04' },
  { n: 4, step: '04' },
  { n: 5, step: '—' },
  { n: 6, step: '05' },
]

const screenshots = computed(() =>
  shotMeta.map(({ n, step }) => ({
    n,
    step,
    src: screenshotUrl(n),
    caption: t(`docs.showcase.shot${n}`),
  }))
)

const faq = computed(() => [
  { q: t('docs.showcase.faq1q'), a: t('docs.showcase.faq1a') },
  { q: t('docs.showcase.faq2q'), a: t('docs.showcase.faq2a') },
  { q: t('docs.showcase.faq3q'), a: t('docs.showcase.faq3a') },
])

const lightbox = reactive({ open: false, index: 0 })
function open(i) {
  lightbox.open = true
  lightbox.index = i
}
function close() {
  lightbox.open = false
}
function prev() {
  lightbox.index = (lightbox.index - 1 + screenshots.value.length) % screenshots.value.length
}
function next() {
  lightbox.index = (lightbox.index + 1) % screenshots.value.length
}
</script>

<style scoped>
.showcase {
  display: flex;
  flex-direction: column;
  gap: clamp(12px, 1.5vw, 18px);
  width: 100%;
  min-width: 0;
}
#doc-show-hero,
#doc-show-highlights,
#doc-show-shots,
#doc-show-faq {
  scroll-margin-top: 88px;
}

.hero {
  display: grid;
  grid-template-columns: minmax(0, 1.25fr) minmax(0, 0.85fr);
  gap: clamp(12px, 1.5vw, 18px);
  align-items: stretch;
}
.hero-text,
.hero-card,
.features,
.gallery,
.faq {
  border: 1px solid var(--doc-border);
  background: var(--doc-surface);
  border-radius: var(--doc-radius);
  padding: clamp(14px, 1.5vw, 20px);
  box-shadow: var(--doc-shadow-soft);
}
.kicker {
  font-size: 11px;
  font-weight: 900;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--doc-muted);
}
.title {
  margin-top: 10px;
  font-size: clamp(1.35rem, 2.2vw, 1.65rem);
  font-weight: 950;
  letter-spacing: -0.03em;
  line-height: 1.2;
}
.sub {
  margin-top: 10px;
  color: var(--doc-muted);
  line-height: 1.7;
  max-width: 65ch;
}
.cta {
  display: flex;
  gap: 10px;
  margin-top: 14px;
  flex-wrap: wrap;
}
.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 10px 14px;
  border-radius: 0;
  border: 1px solid var(--doc-border);
  background: color-mix(in srgb, var(--doc-surface) 70%, transparent);
  color: var(--doc-text);
  text-decoration: none;
  font-weight: 900;
  font-size: 0.9rem;
}
.btn.primary {
  background: var(--doc-cta-primary-bg, #111827);
  color: var(--doc-cta-primary-fg, #fff);
  border-color: var(--doc-cta-primary-bg, #111827);
}

.card-title,
.section-title {
  font-size: 11px;
  font-weight: 900;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--doc-muted);
  margin-bottom: 12px;
}
.steps {
  display: grid;
  grid-template-columns: 1fr;
  gap: 10px;
}
.step {
  display: flex;
  gap: 10px;
  align-items: center;
  padding: 10px;
  border-radius: 0;
  border: 1px solid var(--doc-border);
  background: var(--doc-upload-surface);
}
.no {
  width: 40px;
  text-align: center;
  font-weight: 900;
  color: var(--doc-muted);
  flex-shrink: 0;
}
.label {
  font-weight: 900;
  color: var(--doc-text);
  font-size: 0.9rem;
  line-height: 1.35;
}
.mono {
  font-family: var(--doc-font-mono);
}

.grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: clamp(8px, 1.2vw, 12px);
}
.feature {
  border: 1px solid var(--doc-border);
  background: var(--doc-upload-surface);
  border-radius: 0;
  padding: 12px;
  min-width: 0;
}
.ft {
  font-weight: 950;
  font-size: 0.95rem;
}
.fd {
  margin-top: 8px;
  color: var(--doc-muted);
  line-height: 1.6;
  font-size: 0.8125rem;
}

.shots {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: clamp(8px, 1.2vw, 12px);
}
.shot {
  border: 1px solid var(--doc-border);
  border-radius: 0;
  overflow: hidden;
  background: var(--doc-upload-surface);
  padding: 0;
  cursor: pointer;
  text-align: left;
  min-width: 0;
}
.shot img {
  width: 100%;
  aspect-ratio: 16 / 10;
  height: auto;
  object-fit: cover;
  object-position: top center;
  display: block;
}
.cap {
  padding: 10px 10px 12px;
  color: var(--doc-muted);
  font-size: 0.75rem;
  line-height: 1.45;
  word-break: break-word;
}

.q {
  border: 1px solid var(--doc-border);
  border-radius: 0;
  padding: 10px 12px;
  background: var(--doc-upload-surface);
}
.q summary {
  cursor: pointer;
  font-weight: 950;
  color: var(--doc-text);
}
.a {
  margin-top: 10px;
  color: var(--doc-muted);
  line-height: 1.7;
}

.lb {
  position: fixed;
  inset: 0;
  background: color-mix(in srgb, #000 62%, transparent);
  z-index: 9999;
  display: grid;
  place-items: center;
  padding: 20px;
}
.lb-img {
  max-width: min(1100px, 92vw);
  max-height: 76vh;
  border-radius: 0;
  border: 1px solid color-mix(in srgb, #fff 18%, transparent);
  object-fit: contain;
}
.lb-cap {
  margin-top: 12px;
  color: #fff;
  max-width: 92vw;
  text-align: center;
  font-size: 0.875rem;
  line-height: 1.4;
}
.lb-close {
  position: fixed;
  top: 18px;
  right: 18px;
  width: 40px;
  height: 40px;
  border-radius: 0;
  border: 1px solid color-mix(in srgb, #fff 18%, transparent);
  background: transparent;
  color: #fff;
  font-size: 26px;
  cursor: pointer;
}
.lb-nav {
  position: fixed;
  top: 50%;
  transform: translateY(-50%);
  width: 44px;
  height: 44px;
  border-radius: 0;
  border: 1px solid color-mix(in srgb, #fff 18%, transparent);
  background: transparent;
  color: #fff;
  font-size: 30px;
  cursor: pointer;
}
.lb-prev {
  left: 18px;
}
.lb-next {
  right: 18px;
}

@media (max-width: 1100px) {
  .hero {
    grid-template-columns: 1fr;
  }
  .grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .shots {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 640px) {
  .grid {
    grid-template-columns: 1fr;
  }
  .shots {
    grid-template-columns: 1fr;
  }
  .shot img {
    aspect-ratio: 16 / 9;
  }
}
</style>
