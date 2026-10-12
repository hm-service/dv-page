<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { avatarPath } from '../api/profile'
import { playHeroSequence, prefersReducedMotion } from '../composables/useHeroSequence'
import { languages, t, type Language } from '../i18n'

const props = defineProps<{
  name: string
  lang: Language
}>()

const emit = defineEmits<{ next: []; 'switch-language': [] }>()

const overlay = ref<HTMLElement | null>(null)
const glass = ref<HTMLElement | null>(null)
const axisLine = ref<HTMLElement | null>(null)
const avatar = ref<HTMLElement | null>(null)
const identity = ref<HTMLElement | null>(null)
const nameText = ref<HTMLElement | null>(null)
const nameCursor = ref<HTMLElement | null>(null)

onMounted(() => {
  void playHeroSequence(
    {
      overlay: overlay.value,
      glass: glass.value,
      axisLine: axisLine.value,
      avatar: avatar.value,
      identity: identity.value,
      nameText: nameText.value,
      nameCursor: nameCursor.value,
    },
    props.name,
    prefersReducedMotion(),
  )
})
</script>

<template>
  <section class="hero-section view-panel view-center">
    <div ref="overlay" class="hero-overlay"></div>
    <div ref="glass" class="hero-glass"></div>
    <div class="hero-stage">
      <div class="hero-lane hero-lane-left">
        <img
          ref="avatar"
          :src="avatarPath"
          :alt="t(lang, 'hero.avatar-alt', { name })"
          class="avatar avatar-image"
        />
      </div>
      <span ref="axisLine" class="axis-line" aria-hidden="true"></span>
      <div ref="identity" class="hero-lane hero-lane-right identity">
        <h1 class="user-id" :aria-label="name">
          <span ref="nameText" class="user-id-text"></span>
          <span ref="nameCursor" class="user-id-cursor" aria-hidden="true"></span>
        </h1>
      </div>
    </div>
    <button class="hero-scroll-hint" type="button" @click="emit('next')">
      {{ t(lang, 'hero.scroll-hint') }}
    </button>
    <button
      class="lang-switch"
      type="button"
      :title="t(lang, 'language.switch-aria')"
      :aria-label="t(lang, 'language.switch-aria')"
      @click="emit('switch-language')"
    >
      <svg class="lang-switch-icon" viewBox="0 0 16 16" aria-hidden="true">
        <circle cx="8" cy="8" r="6.25" />
        <path d="M1.75 8h12.5" />
        <path d="M8 1.75c1.9 2 2.85 3.9 2.85 6.25S9.9 14.25 8 14.25 5.15 12.35 5.15 8 6.1 3.75 8 1.75Z" />
      </svg>
      <!-- 所有语言名叠在同一格：按钮宽度按最长的那份定，切换时不抖 -->
      <span class="lang-switch-labels">
        <span
          v-for="code in languages"
          :key="code"
          :class="{ 'is-active': code === lang }"
        >
          {{ t(code, 'language.name') }}
        </span>
      </span>
    </button>
  </section>
</template>

<style scoped>
.hero-section {
    position: relative;
    background: url("../assets/bg.jpg") no-repeat center center;
    background-size: cover;
    overflow: hidden;
}

.hero-scroll-hint {
    position: absolute;
    left: 50%;
    bottom: 34px;
    z-index: 3;
    display: inline-flex;
    align-items: center;
    gap: 10px;
    padding: 10px 16px;
    border: 1px solid rgba(255, 255, 255, 0.14);
    border-radius: 999px;
    background: rgba(9, 14, 24, 0.28);
    backdrop-filter: blur(12px);
    -webkit-backdrop-filter: blur(12px);
    color: rgba(255, 255, 255, 0.82);
    font-size: 0.82rem;
    letter-spacing: 0.08em;
    cursor: pointer;
    transform: translateX(-50%);
    opacity: 0;
    animation: hintReveal 1s ease 2.1s forwards;
}

.hero-scroll-hint::after {
    content: "";
    width: 8px;
    height: 8px;
    border-right: 1.5px solid rgba(255, 255, 255, 0.86);
    border-bottom: 1.5px solid rgba(255, 255, 255, 0.86);
    transform: rotate(45deg) translateY(-1px);
}

/* 与滚动提示同一套玻璃质感，只是更轻，停在首屏右上角 */
.lang-switch {
    position: absolute;
    top: 26px;
    right: 26px;
    z-index: 3;
    display: inline-flex;
    align-items: center;
    gap: 7px;
    padding: 8px 15px;
    border: 1px solid rgba(255, 255, 255, 0.14);
    border-radius: 999px;
    background: rgba(9, 14, 24, 0.28);
    backdrop-filter: blur(12px);
    -webkit-backdrop-filter: blur(12px);
    color: rgba(255, 255, 255, 0.82);
    font: inherit;
    font-size: 0.76rem;
    /* 行高按字号定，免得中日韩与拉丁字形各撑出一个高度 */
    line-height: 1;
    letter-spacing: 0.1em;
    cursor: pointer;
    opacity: 0;
    animation: langReveal 0.9s ease 2.1s forwards;
    transition:
        color 0.25s ease,
        border-color 0.25s ease,
        background-color 0.25s ease;
}

.lang-switch:hover {
    border-color: rgba(255, 255, 255, 0.34);
    background: rgba(9, 14, 24, 0.42);
    color: #ffffff;
}

/* 有图标才看得出它可切，文字留着写「当前语言」 */
.lang-switch-icon {
    width: 13px;
    height: 13px;
    fill: none;
    stroke: currentColor;
    stroke-width: 1.2;
    opacity: 0.85;
}

.lang-switch-labels {
    display: grid;
}

.lang-switch-labels > span {
    grid-area: 1 / 1;
    text-align: center;
    visibility: hidden;
}

.lang-switch-labels > .is-active {
    visibility: visible;
}

.lang-switch:focus-visible {
    outline: 1px solid rgba(255, 255, 255, 0.6);
    outline-offset: 2px;
}

.hero-overlay {
    position: absolute;
    inset: 0;
    background: linear-gradient(120deg, rgba(10, 14, 23, 0.78), rgba(0, 0, 0, 0.52));
    opacity: 0;
    transform: scale(1.04);
    will-change: opacity, transform;
}

.hero-glass {
    position: absolute;
    inset: 0;
    background: rgba(255, 255, 255, 0);
    backdrop-filter: blur(0px);
    -webkit-backdrop-filter: blur(0px);
    will-change: backdrop-filter, background-color;
}

.hero-stage {
    position: relative;
    z-index: 2;
    width: min(92vw, 720px);
    display: flex;
    flex-direction: column;
    justify-content: center;
    align-items: center;
    gap: 20px;
    min-height: 260px;
    padding: 0 2.5rem;
}

.hero-lane {
    display: flex;
    justify-content: center;
    align-items: center;
    min-width: 0;
    width: 100%;
}

.hero-lane-left {
    min-height: 120px;
}

.hero-lane-right {
    min-height: 92px;
}

.axis-line {
    width: min(62vw, 420px);
    height: 2px;
    border-radius: 999px;
    background: rgba(255, 255, 255, 0.88);
    box-shadow: 0 0 18px rgba(255, 255, 255, 0.18);
    opacity: 0;
    transform: scaleX(0.16);
    transform-origin: center;
    will-change: opacity, transform;
}

.avatar {
    opacity: 0;
    transform: translateY(48px) scale(0.94);
    transition: transform 0.35s ease;
    will-change: opacity, transform;
}

.avatar-image {
    width: 82px;
    height: 82px;
    border: 3px solid rgba(255, 255, 255, 0.8);
    border-radius: 50%;
    object-fit: cover;
    box-shadow: 0 12px 32px rgba(0, 0, 0, 0.28);
}

.avatar:hover {
    transform: scale(1.05);
}

.identity {
    display: flex;
    align-items: center;
    justify-content: center;
    min-width: 0;
    opacity: 0;
    transform: translateY(-26px);
    will-change: opacity, transform;
}

.user-id {
    color: #ffffff;
    font-size: clamp(1.8rem, 4vw, 3.2rem);
    font-weight: 600;
    letter-spacing: 0.08em;
    text-shadow: 0 2px 5px rgba(0, 0, 0, 0.5);
    white-space: nowrap;
    display: inline-flex;
    align-items: center;
    min-height: 1.2em;
    line-height: 1.1;
}

.user-id-text {
    display: inline-block;
}

.user-id-cursor {
    width: 0.42em;
    height: 0.92em;
    margin-left: 0.14em;
    border-radius: 2px;
    background: rgba(255, 255, 255, 0.95);
    box-shadow: 0 0 12px rgba(255, 255, 255, 0.2);
    opacity: 0;
}

.user-id-cursor.is-visible {
    opacity: 1;
    animation: cursorPulse 0.9s ease-in-out infinite;
}

@keyframes cursorPulse {
    0%,
    100% {
        opacity: 1;
        transform: scaleY(1);
    }
    50% {
        opacity: 0.22;
        transform: scaleY(0.88);
    }
}

@keyframes hintReveal {
    from {
        opacity: 0;
        transform: translateX(-50%) translateY(10px);
    }
    to {
        opacity: 1;
        transform: translateX(-50%) translateY(0);
    }
}

@keyframes langReveal {
    from {
        opacity: 0;
        transform: translateY(-8px);
    }
    to {
        opacity: 1;
        transform: translateY(0);
    }
}

@media (max-width: 640px) {
    .hero-stage {
        width: min(94vw, 480px);
        gap: 14px;
        min-height: 220px;
        padding: 0 1.5rem;
    }

    .hero-lane-left {
        min-height: 96px;
    }

    .hero-lane-right {
        min-height: 72px;
    }

    .axis-line {
        width: min(72vw, 320px);
    }

    .user-id {
        font-size: clamp(1.45rem, 6vw, 2.2rem);
        letter-spacing: 0.05em;
    }

    .user-id-cursor {
        width: 0.36em;
        height: 0.86em;
    }

    .avatar-image {
        width: 72px;
        height: 72px;
    }

    .hero-scroll-hint {
        bottom: 22px;
        padding: 9px 14px;
        font-size: 0.74rem;
    }

    .lang-switch {
        top: 16px;
        right: 16px;
        padding: 7px 12px;
        font-size: 0.72rem;
    }
}

@media (prefers-reduced-motion: reduce) {
    .hero-overlay,
    .hero-glass,
    .hero-stage,
    .axis-line,
    .avatar,
    .identity,
    .user-id-cursor,
    .hero-scroll-hint,
    .lang-switch {
        animation: none !important;
        transition: none !important;
    }

    /* 关掉动画后得自己把入场态收尾，否则按钮停在 opacity 0 */
    .hero-scroll-hint {
        opacity: 1;
        transform: translateX(-50%);
    }

    .lang-switch {
        opacity: 1;
    }
}
</style>
