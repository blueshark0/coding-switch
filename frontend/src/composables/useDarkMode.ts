import { onMounted, onUnmounted, ref } from 'vue'

const isBrowser = typeof window !== 'undefined' && typeof document !== 'undefined'

const readDarkMode = () =>
  isBrowser ? document.documentElement.classList.contains('dark') : false

export const useDarkMode = () => {
  const isDarkMode = ref(readDarkMode())
  let themeObserver: MutationObserver | null = null

  const syncThemeState = () => {
    isDarkMode.value = readDarkMode()
  }

  const getCssVarValue = (name: string, fallback: string) => {
    if (!isBrowser) return fallback
    const value = getComputedStyle(document.documentElement).getPropertyValue(name)
    return value?.trim() || fallback
  }

  onMounted(() => {
    if (!isBrowser || themeObserver) return

    syncThemeState()
    themeObserver = new MutationObserver((mutations) => {
      if (mutations.some((mutation) => mutation.attributeName === 'class')) {
        syncThemeState()
      }
    })
    themeObserver.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ['class'],
    })
  })

  onUnmounted(() => {
    if (!themeObserver) return
    themeObserver.disconnect()
    themeObserver = null
  })

  return {
    getCssVarValue,
    isDarkMode,
  }
}
