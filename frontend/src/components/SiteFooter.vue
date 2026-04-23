<template>
  <footer
    class="doc-site-footer"
    :class="{ 'doc-site-footer--compact': compact }"
  >
    <p class="doc-site-footer__line">
      <i18n-t keypath="footer.maintainedBy" tag="span">
        <template #link>
          <a
            class="doc-site-footer__link"
            href="https://github.com/JustineDevs/"
            target="_blank"
            rel="noopener noreferrer"
          >{{ $t('footer.authorName') }}</a>
        </template>
      </i18n-t>
    </p>
  </footer>
</template>

<script setup>
defineProps({
  /**
   * Tighter bar for workbench / process (full-height layouts).
   */
  compact: {
    type: Boolean,
    default: false,
  },
})
</script>
