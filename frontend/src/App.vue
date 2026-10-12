<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { fetchProfile, type Profile } from './api/profile'
import DetailsSection from './components/DetailsSection.vue'
import HeroSection from './components/HeroSection.vue'
import { prefersReducedMotion } from './composables/useHeroSequence'
import { useViewStack } from './composables/useViewStack'
import { exposeDebug } from './debug'
import { otherLanguage, t, type Language } from './i18n'
import { detectBrowserLanguage, readStoredLanguage, storeLanguage } from './i18n/storage'

const profile = ref<Profile | null>(null)
const loadFailed = ref(false)
/** 用户选过的语言优先；没选过则先按浏览器猜，拿到接口结果后再以服务端协商为准。 */
const lang = ref<Language>(readStoredLanguage() ?? detectBrowserLanguage())

const reduceMotion = prefersReducedMotion()
const { view, showView } = useViewStack({ lockMs: reduceMotion ? 0 : 920, viewCount: 2 })

const setLanguage = (next: Language) => {
  storeLanguage(next)
  lang.value = next
}

watch(
  [lang, profile],
  () => {
    document.documentElement.lang = lang.value
    if (profile.value) {
      document.title = t(lang.value, 'site.title', { name: profile.value.name })
    }
  },
  { immediate: true },
)

onMounted(async () => {
  exposeDebug({
    state: () => ({
      name: profile.value?.name ?? null,
      site: profile.value?.site ?? null,
      lang: lang.value,
      view: view.value,
    }),
    showView,
    setLanguage,
  })

  const stored = readStoredLanguage()
  try {
    const fetched = await fetchProfile(stored)

    profile.value = fetched
    if (!stored) {
      lang.value = fetched.lang
    }
  } catch (error) {
    loadFailed.value = true
    console.error(error)
  }
})
</script>

<template>
  <main
    class="page-stack"
    :class="{ 'is-second-view': view === 1 }"
    :data-current-view="view"
  >
    <template v-if="profile">
      <HeroSection
        :name="profile.name"
        :lang="lang"
        @next="showView(1)"
        @switch-language="setLanguage(otherLanguage(lang))"
      />
      <DetailsSection
        :active="view === 1"
        :site="profile.site"
        :name="profile.name"
        :lang="lang"
      />
    </template>
    <p v-else-if="loadFailed" class="profile-error">{{ t(lang, 'app.profile-error') }}</p>
  </main>
</template>

<style scoped>
.profile-error {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100vh;
    padding: 0 1.5rem;
    color: rgba(226, 232, 240, 0.82);
    font-size: 0.95rem;
    letter-spacing: 0.04em;
    text-align: center;
}
</style>
