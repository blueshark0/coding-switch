import { onMounted, onUnmounted, ref, type ComputedRef } from 'vue'
import type { AutomationCard } from '../../data/cards'
import { fetchAppSettings, saveAppSettings, type AppSettings } from '../../services/appSettings'
import type { ProviderTab } from '../../constants/platforms'
import type { TranslateFn } from './types'

type UseProviderPreferencesOptions = {
  activeTab: ComputedRef<ProviderTab>
  t: TranslateFn
}

const DEFAULT_APP_SETTINGS: AppSettings = {
  show_heatmap: true,
  show_home_title: true,
  default_claude_provider: '',
  default_codex_provider: '',
  default_gemini_provider: '',
}

export const useProviderPreferences = ({
  activeTab,
  t,
}: UseProviderPreferencesOptions) => {
  const appSettings = ref<AppSettings>({ ...DEFAULT_APP_SETTINGS })
  const showHeatmap = ref(true)
  const showHomeTitle = ref(true)

  const getDefaultProviderName = (tab: ProviderTab) => {
    if (tab === 'claude') return appSettings.value.default_claude_provider
    if (tab === 'codex') return appSettings.value.default_codex_provider
    return appSettings.value.default_gemini_provider
  }

  const setDefaultProviderName = (tab: ProviderTab, providerName: string) => {
    if (tab === 'claude') {
      appSettings.value.default_claude_provider = providerName
      return
    }
    if (tab === 'codex') {
      appSettings.value.default_codex_provider = providerName
      return
    }
    appSettings.value.default_gemini_provider = providerName
  }

  const syncVisibilitySettings = () => {
    showHeatmap.value = appSettings.value.show_heatmap
    showHomeTitle.value = appSettings.value.show_home_title
  }

  const loadAppSettings = async () => {
    try {
      const data = await fetchAppSettings()
      appSettings.value = {
        default_claude_provider: data?.default_claude_provider ?? '',
        default_codex_provider: data?.default_codex_provider ?? '',
        default_gemini_provider: data?.default_gemini_provider ?? '',
        show_heatmap: data?.show_heatmap ?? true,
        show_home_title: data?.show_home_title ?? true,
      }
    } catch (error) {
      console.error('failed to load app settings', error)
      appSettings.value = { ...DEFAULT_APP_SETTINGS }
    }

    syncVisibilitySettings()
  }

  const initializeProviderPreferences = async () => {
    await loadAppSettings()
  }

  const handleAppSettingsUpdated = () => {
    void loadAppSettings()
  }

  const isDefaultProvider = (providerName: string) => {
    return getDefaultProviderName(activeTab.value) === providerName
  }

  const toggleDefaultProvider = async (card: AutomationCard) => {
    const tab = activeTab.value
    if (!card.enabled) {
      window.alert(t('components.main.providerDisabledWarning'))
      return
    }

    const previousSettings = { ...appSettings.value }
    setDefaultProviderName(tab, isDefaultProvider(card.name) ? '' : card.name)

    try {
      appSettings.value = await saveAppSettings(appSettings.value)
      syncVisibilitySettings()
    } catch (error) {
      console.error('Failed to save app settings', error)
      appSettings.value = previousSettings
      syncVisibilitySettings()
      window.alert(t('components.main.saveFailed'))
    }
  }

  onMounted(() => {
    window.addEventListener('app-settings-updated', handleAppSettingsUpdated)
  })

  onUnmounted(() => {
    window.removeEventListener('app-settings-updated', handleAppSettingsUpdated)
  })

  return {
    initializeProviderPreferences,
    isDefaultProvider,
    showHeatmap,
    showHomeTitle,
    toggleDefaultProvider,
  }
}
