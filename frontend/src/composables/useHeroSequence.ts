const easing = {
  reveal: 'cubic-bezier(0.22, 1, 0.36, 1)',
  settle: 'cubic-bezier(0.2, 0.9, 0.2, 1)',
} as const

const durations = {
  overlay: 2200,
  line: 520,
  avatar: 700,
  identity: 420,
  glass: 1200,
} as const

const delays = {
  line: 120,
  intro: 140,
} as const

/** 逐字打字的间隔：越到后面越慢，收尾更有停顿感。 */
const typingStepMs = 22
const typingBaseMs = 70

export interface HeroElements {
  overlay: HTMLElement | null
  glass: HTMLElement | null
  axisLine: HTMLElement | null
  avatar: HTMLElement | null
  identity: HTMLElement | null
  nameText: HTMLElement | null
  nameCursor: HTMLElement | null
}

export function prefersReducedMotion(): boolean {
  return window.matchMedia('(prefers-reduced-motion: reduce)').matches
}

const wait = (ms: number) =>
  new Promise<void>((resolve) => {
    window.setTimeout(resolve, ms)
  })

function applyFinalFrame(element: HTMLElement, keyframes: Keyframe[]): void {
  const lastFrame = keyframes[keyframes.length - 1]
  for (const [property, value] of Object.entries(lastFrame)) {
    if (property === 'offset' || property === 'easing' || value === undefined) {
      continue
    }
    element.style.setProperty(
      property.replace(/[A-Z]/g, (char) => `-${char.toLowerCase()}`),
      String(value),
    )
  }
}

function animate(
  element: HTMLElement | null,
  keyframes: Keyframe[],
  options: KeyframeAnimationOptions,
  reduceMotion: boolean,
): Promise<void> {
  if (!element) {
    return Promise.resolve()
  }

  if (reduceMotion) {
    applyFinalFrame(element, keyframes)
    return Promise.resolve()
  }

  const animation = element.animate(keyframes, { fill: 'forwards', ...options })
  return animation.finished.then(
    () => undefined,
    () => undefined,
  )
}

/** 打字帧由名字本身推导，换名字不需要改配置。 */
function typingFrames(name: string): string[] {
  return Array.from({ length: name.length }, (_, index) => name.slice(0, index + 1))
}

async function runTypingSequence(elements: HeroElements, name: string): Promise<void> {
  const { nameText, nameCursor } = elements
  if (!nameText || !nameCursor) {
    return
  }

  nameCursor.classList.add('is-visible')
  const frames = typingFrames(name)

  for (let index = 0; index < frames.length; index += 1) {
    nameText.textContent = frames[index]
    if (index < frames.length - 1) {
      await wait(typingBaseMs + typingStepMs * index)
    }
  }
}

export async function playHeroSequence(
  elements: HeroElements,
  name: string,
  reduceMotion: boolean,
): Promise<void> {
  const { overlay, glass, axisLine, avatar, identity, nameCursor } = elements

  nameCursor?.classList.add('is-visible')

  void animate(
    overlay,
    [
      { opacity: 0, transform: 'scale(1.04)' },
      { opacity: 1, transform: 'scale(1)' },
    ],
    { duration: durations.overlay, easing: easing.reveal },
    reduceMotion,
  )

  await wait(delays.line)
  await animate(
    axisLine,
    [
      { opacity: 0, transform: 'scaleX(0.16)' },
      { opacity: 1, transform: 'scaleX(1)' },
    ],
    { duration: durations.line, easing: easing.reveal },
    reduceMotion,
  )

  await wait(delays.intro)
  const avatarIntro = animate(
    avatar,
    [
      { opacity: 0, transform: 'translateY(48px) scale(0.94)' },
      { opacity: 1, transform: 'translateY(0) scale(1)' },
    ],
    { duration: durations.avatar, easing: easing.settle },
    reduceMotion,
  )

  const identityIntro = animate(
    identity,
    [
      { opacity: 0, transform: 'translateY(-26px)' },
      { opacity: 1, transform: 'translateY(0)' },
    ],
    { duration: durations.identity, easing: easing.settle },
    reduceMotion,
  )

  await avatarIntro
  void animate(
    glass,
    [
      { backgroundColor: 'rgba(255, 255, 255, 0)', backdropFilter: 'blur(0px)' },
      { backgroundColor: 'rgba(255, 255, 255, 0.06)', backdropFilter: 'blur(5px)' },
    ],
    { duration: durations.glass, easing: easing.reveal },
    reduceMotion,
  )

  await identityIntro
  await runTypingSequence(elements, name)
}
