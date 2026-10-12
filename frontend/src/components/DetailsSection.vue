<script setup lang="ts">
import RecordBar from './RecordBar.vue'
import type { SiteInfo } from '../api/profile'
import type { Language } from '../i18n'

defineProps<{
  active: boolean
  site: SiteInfo
  /** 版权署名，来自 /api/profile。 */
  name: string
  lang: Language
}>()
</script>

<template>
  <section
    class="details-section view-panel view-center details-panel"
    :class="{ 'is-active': active }"
  >
    <div class="details-grid details-empty" aria-hidden="true"></div>

    <RecordBar
      :icp="site.icp"
      :icp-url="site.icpUrl"
      :copyright-year="site.copyrightYear"
      :copyright-holder="name"
      :lang="lang"
    />
  </section>
</template>

<style scoped>
.details-section {
    position: relative;
    background: radial-gradient(
        circle at top left,
        rgba(255, 255, 255, 0.94),
        rgba(248, 250, 252, 0.98) 40%,
        #e5edf5 100%
    );
    overflow: hidden;
}

.details-section::before,
.details-section::after {
    content: "";
    position: absolute;
    border-radius: 999px;
    pointer-events: none;
    opacity: 0.6;
}

.details-section::before {
    top: 12%;
    left: -6%;
    width: 360px;
    height: 360px;
    background: radial-gradient(circle, rgba(191, 219, 254, 0.52), rgba(191, 219, 254, 0));
}

.details-section::after {
    right: -8%;
    bottom: 14%;
    width: 380px;
    height: 380px;
    background: radial-gradient(circle, rgba(226, 232, 240, 0.86), rgba(254, 249, 195, 0));
}

.details-grid {
    position: relative;
    z-index: 1;
    width: min(92vw, 960px);
    opacity: 0;
    transform: translateY(24px);
    transition:
        opacity 0.65s cubic-bezier(0.22, 1, 0.36, 1),
        transform 0.65s cubic-bezier(0.22, 1, 0.36, 1);
}

.details-section.is-active .details-grid {
    opacity: 1;
    transform: translateY(0);
}

.details-panel {
    padding: 3.5rem 1.5rem;
}

.details-empty {
    min-height: calc(100vh - 180px);
}

@media (max-width: 640px) {
    .details-panel {
        padding: 3.5rem 1.25rem;
    }

    .details-empty {
        min-height: calc(100vh - 150px);
    }
}

@media (prefers-reduced-motion: reduce) {
    .details-grid {
        transition: none;
    }
}
</style>
