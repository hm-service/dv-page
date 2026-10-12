<script setup lang="ts">
import { t, type Language } from '../i18n'

defineProps<{
  icp: string
  icpUrl: string
  copyrightYear: number
  copyrightHolder: string
  lang: Language
}>()
</script>

<template>
  <footer class="record-bar" :aria-label="t(lang, 'record.label')">
    <a :href="icpUrl" target="_blank" rel="noreferrer">{{ icp }}</a>
    <span class="record-item">
      {{ t(lang, 'record.copyright', { year: copyrightYear, name: copyrightHolder }) }}
    </span>
  </footer>
</template>

<style scoped>
.record-bar {
    position: absolute;
    left: 0;
    right: 0;
    bottom: 0;
    z-index: 2;
    display: flex;
    flex-wrap: wrap;
    justify-content: center;
    align-items: center;
    gap: 6px 18px;
    padding: 14px 18px 16px;
    border-top: 1px solid rgba(15, 23, 42, 0.08);
    background: rgba(255, 255, 255, 0.72);
    box-shadow: 0 -10px 30px rgba(148, 163, 184, 0.08);
    transform: none;
}

.record-bar a {
    color: rgba(15, 23, 42, 0.78);
    text-decoration: none;
}

.record-bar a:hover {
    color: #0f172a;
}

.record-item {
    color: rgba(15, 23, 42, 0.68);
    font-size: 0.78rem;
    letter-spacing: 0.02em;
    white-space: nowrap;
}

@media (max-width: 640px) {
    .record-bar {
        gap: 6px 12px;
        padding: 12px 14px 14px;
    }

    .record-item {
        font-size: 0.74rem;
    }
}
</style>
