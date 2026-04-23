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
    <p class="doc-site-footer__line doc-site-footer__bmc">
      <a
        class="doc-site-footer__bmc-link"
        href="https://www.buymeacoffee.com/justinedevs"
        target="_blank"
        rel="noopener noreferrer"
      >
        <img
          class="doc-site-footer__bmc-badge"
          src="https://img.shields.io/badge/Buy%20Me%20A%20Coffee-ffdd00?style=flat-square&logo=buy-me-a-coffee&logoColor=black"
          :alt="$t('footer.buyMeACoffeeAlt')"
          loading="lazy"
          decoding="async"
        />
      </a>
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
