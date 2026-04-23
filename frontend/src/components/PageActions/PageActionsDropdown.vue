<template>
  <div class="page-actions" role="group" :aria-label="$t('home.pageActionsGroupAria')">
    <a
      class="doc-pill doc-pill--action doc-pill--github"
      :href="repoUrl"
      target="_blank"
      rel="noopener noreferrer"
    >
      {{ $t('nav.visitGithub') }}
    </a>
    <details ref="menuRef" class="page-actions-details">
      <summary class="doc-pill doc-pill--action page-actions-summary">
        {{ $t('home.pageActionsMore') }}
        <span class="page-actions-chevron" aria-hidden="true">▾</span>
      </summary>
      <div class="page-actions-panel" role="menu" @keydown="onPanelKeydown">
        <button type="button" class="page-actions-item" role="menuitem" @click="copyPageLink">
          <span class="page-actions-item-label">{{
            copiedLink ? $t('home.pageActionsLinkCopied') : $t('home.pageActionsCopyLink')
          }}</span>
        </button>
        <a
          class="page-actions-item"
          role="menuitem"
          :href="readmeUrl"
          target="_blank"
          rel="noopener noreferrer"
          @click="closeMenu"
        >
          <span class="page-actions-item-label">{{ $t('home.pageActionsReadme') }}</span>
        </a>
        <a
          class="page-actions-item"
          role="menuitem"
          :href="chatgptHref"
          target="_blank"
          rel="noopener noreferrer"
          @click="closeMenu"
        >
          <span class="page-actions-item-label">{{ $t('home.pageActionsOpenChatgpt') }}</span>
        </a>
        <a
          class="page-actions-item"
          role="menuitem"
          :href="claudeHref"
          target="_blank"
          rel="noopener noreferrer"
          @click="closeMenu"
        >
          <span class="page-actions-item-label">{{ $t('home.pageActionsOpenClaude') }}</span>
        </a>
        <button type="button" class="page-actions-item" role="menuitem" @click="onJumpPlayground">
          <span class="page-actions-item-label">{{ $t('home.docJumpPlayground') }}</span>
        </button>
      </div>
    </details>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { buildLlmPrompt, chatgptUrlFromPrompt, claudeUrlFromPrompt } from './llmLinks.js'

const props = defineProps({
  /** Primary GitHub repository URL (source / issues). */
  repoUrl: {
    type: String,
    default: 'https://github.com/go-mirofish/go-mirofish',
  },
})

const emit = defineEmits(['scroll-playground'])

const { t } = useI18n()
const menuRef = ref(null)
const copiedLink = ref(false)
let copyTimer = null

const pageUrl = computed(() => {
  if (typeof window === 'undefined') return props.repoUrl
  return window.location.href
})

const readmeUrl = computed(
  () => `${props.repoUrl.replace(/\/$/, '')}/blob/main/README.md`
)

const llmPrompt = computed(() => buildLlmPrompt(t, pageUrl.value))

const chatgptHref = computed(() => chatgptUrlFromPrompt(llmPrompt.value))
const claudeHref = computed(() => claudeUrlFromPrompt(llmPrompt.value))

function closeMenu() {
  if (menuRef.value) menuRef.value.removeAttribute('open')
}

function onPanelKeydown(e) {
  if (e.key === 'Escape' && menuRef.value) {
    menuRef.value.removeAttribute('open')
  }
}

async function copyPageLink() {
  try {
    await navigator.clipboard.writeText(pageUrl.value)
    copiedLink.value = true
    if (copyTimer) clearTimeout(copyTimer)
    copyTimer = setTimeout(() => {
      copiedLink.value = false
      copyTimer = null
    }, 1500)
  } catch {
    copiedLink.value = false
  }
  closeMenu()
}

function onJumpPlayground() {
  emit('scroll-playground')
  closeMenu()
}
</script>

<style scoped>
.page-actions {
  display: inline-flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.5rem;
  margin-left: 0.15rem;
}

.page-actions-details {
  position: relative;
}

.page-actions-summary {
  list-style: none;
  cursor: pointer;
  user-select: none;
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
}

.page-actions-summary::-webkit-details-marker {
  display: none;
}

.page-actions-chevron {
  font-size: 0.65rem;
  opacity: 0.8;
  line-height: 1;
}

.page-actions-details[open] .page-actions-chevron {
  transform: rotate(0deg);
}

.doc-pill--action {
  text-decoration: none;
  color: var(--doc-text);
  border-color: var(--doc-border);
  background: var(--doc-surface);
  transition: border-color 0.15s, box-shadow 0.15s;
}

.doc-pill--action:hover {
  border-color: var(--doc-accent);
  box-shadow: 0 2px 10px var(--doc-accent-soft);
}

.doc-pill--github {
  color: var(--doc-text);
}

.page-actions-panel {
  position: absolute;
  z-index: 40;
  right: 0;
  top: calc(100% + 0.35rem);
  min-width: 12rem;
  max-width: min(20rem, calc(100vw - 2rem));
  padding: 0.35rem 0;
  background: var(--doc-surface);
  border: 1px solid var(--doc-border);
  border-radius: 8px;
  box-shadow: 0 10px 30px -8px rgba(0, 0, 0, 0.12);
}

.page-actions-item {
  display: block;
  width: 100%;
  text-align: left;
  padding: 0.5rem 0.9rem;
  font-family: var(--doc-font-sans, system-ui, sans-serif);
  font-size: 0.8rem;
  color: var(--doc-text);
  background: transparent;
  border: none;
  cursor: pointer;
  text-decoration: none;
  transition: background 0.1s;
}

.page-actions-item:hover,
.page-actions-item:focus-visible {
  background: var(--doc-bg, #f4f4f5);
  outline: none;
}

.page-actions-item-label {
  display: block;
  line-height: 1.35;
}
</style>
