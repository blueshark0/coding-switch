import type { ComputedRef } from 'vue'
import type { AutomationCard } from '../../data/cards'
import type { ProviderTab } from '../../constants/platforms'
import { useAppSettingsStore } from '../../composables/useAppSettingsStore'
import type { AppSettings } from '../../services/appSettings'
import { getErrorMessage } from '../../utils/errors'
import { showToast } from '../../utils/toast'
import type { TranslateFn } from './types'

type UseProviderPreferencesOptions = {
  activeTab: ComputedRef<ProviderTab>
  t: TranslateFn
}

export const useProviderPreferences = ({
  activeTab,
  t,
}: UseProviderPreferencesOptions) => {
  const {
    getDefaultProviderName,
    loadSettings,
    showHeatmap,
    showHomeTitle,
    updateSettings,
  } = useAppSettingsStore()

  const initializeProviderPreferences = async (force = false) => {
    await loadSettings(force)
  }

  const isDefaultProvider = (providerName: string) => {
    return getDefaultProviderName(activeTab.value) === providerName
  }

  const toggleDefaultProvider = async (card: AutomationCard) => {
    const tab = activeTab.value

    try {
      await updateSettings({
        [defaultProviderKey(tab)]: isDefaultProvider(card.name) ? '' : card.name,
      })
    } catch (error) {
      console.error('Failed to save app settings', error)
      showToast(getErrorMessage(error, t('components.main.saveFailed')), 'error')
    }
  }

  return {
    initializeProviderPreferences,
    isDefaultProvider,
    showHeatmap,
    showHomeTitle,
    toggleDefaultProvider,
  }
}

const defaultProviderKey = (tab: ProviderTab): keyof AppSettings => {
  if (tab === 'claude') return 'default_claude_provider'
  if (tab === 'codex') return 'default_codex_provider'
  return 'default_gemini_provider'
}
