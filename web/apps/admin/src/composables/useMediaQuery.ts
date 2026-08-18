import { onMounted, onUnmounted, ref, type Ref } from 'vue'

/** Reactive match for a CSS media query. */
export function useMediaQuery(query: string): Ref<boolean> {
  const matches = ref(false)
  let mql: MediaQueryList | null = null

  function onChange(e: MediaQueryListEvent | MediaQueryList) {
    matches.value = e.matches
  }

  onMounted(() => {
    if (typeof window === 'undefined' || !window.matchMedia) return
    mql = window.matchMedia(query)
    matches.value = mql.matches
    if (typeof mql.addEventListener === 'function') {
      mql.addEventListener('change', onChange)
    } else {
      // Safari < 14
      mql.addListener(onChange)
    }
  })

  onUnmounted(() => {
    if (!mql) return
    if (typeof mql.removeEventListener === 'function') {
      mql.removeEventListener('change', onChange)
    } else {
      mql.removeListener(onChange)
    }
    mql = null
  })

  return matches
}

/** Admin mobile breakpoint — aligns with Element Plus `sm` (768px). */
export function useIsMobile(): Ref<boolean> {
  return useMediaQuery('(max-width: 768px)')
}

/** Tight header: icon-only domain items and compact right tools. */
export function useIsCompact(): Ref<boolean> {
  return useMediaQuery('(max-width: 1100px)')
}
