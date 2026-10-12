import { onBeforeUnmount, onMounted, ref, type Ref } from 'vue'

const wheelThreshold = 18
const touchThreshold = 42

export interface ViewStackOptions {
  /** 过渡动画的时长；用于过渡期间的输入锁，避免连点跳过一屏。 */
  lockMs: number
  viewCount: number
}

export interface ViewStack {
  view: Ref<number>
  showView: (index: number) => void
}

/** 两屏整屏切换：滚轮 / 键盘 / 触屏都折算成「上一屏、下一屏」。 */
export function useViewStack({ lockMs, viewCount }: ViewStackOptions): ViewStack {
  const view = ref(0)
  let locked = false
  let touchStartY = 0

  const showView = (index: number) => {
    if (locked) {
      return
    }

    const next = Math.max(0, Math.min(viewCount - 1, index))
    if (next === view.value) {
      return
    }

    const previous = view.value
    view.value = next
    locked = true
    window.setTimeout(
      () => {
        locked = false
      },
      previous === next ? 0 : lockMs,
    )
  }

  const step = (direction: number) => showView(view.value + direction)

  const onWheel = (event: WheelEvent) => {
    if (Math.abs(event.deltaY) < wheelThreshold) {
      return
    }

    event.preventDefault()
    step(event.deltaY > 0 ? 1 : -1)
  }

  const onKeydown = (event: KeyboardEvent) => {
    if (['ArrowDown', 'PageDown', 'Space'].includes(event.code)) {
      event.preventDefault()
      step(1)
      return
    }

    if (['ArrowUp', 'PageUp'].includes(event.code)) {
      event.preventDefault()
      step(-1)
    }
  }

  const onTouchStart = (event: TouchEvent) => {
    touchStartY = event.touches[0]?.clientY ?? 0
  }

  const onTouchEnd = (event: TouchEvent) => {
    const touchEndY = event.changedTouches[0]?.clientY ?? touchStartY
    const deltaY = touchStartY - touchEndY
    if (Math.abs(deltaY) < touchThreshold) {
      return
    }

    step(deltaY > 0 ? 1 : -1)
  }

  onMounted(() => {
    window.addEventListener('wheel', onWheel, { passive: false })
    window.addEventListener('keydown', onKeydown)
    window.addEventListener('touchstart', onTouchStart, { passive: true })
    window.addEventListener('touchend', onTouchEnd, { passive: true })
  })

  onBeforeUnmount(() => {
    window.removeEventListener('wheel', onWheel)
    window.removeEventListener('keydown', onKeydown)
    window.removeEventListener('touchstart', onTouchStart)
    window.removeEventListener('touchend', onTouchEnd)
  })

  return { view, showView }
}
