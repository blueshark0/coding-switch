import { computed, onMounted, ref } from 'vue'
import { fetchCurrentVersion } from '../../services/version'
import { providerTabs, type ProviderTab } from '../../constants/platforms'
import type { LocaleRef, TranslateFn } from './types'
import { useProviderCatalog } from './useProviderCatalog'
import { useProviderPreferences } from './useProviderPreferences'
import { useProviderStats } from './useProviderStats'

type UseProviderWorkbenchOptions = {
  locale: LocaleRef
  t: TranslateFn
}

export const useProviderWorkbench = ({ locale, t }: UseProviderWorkbenchOptions) => {
  const tabs = providerTabs
  const appVersion = ref('')
  const selectedIndex = ref(0)

  const activeTab = computed<ProviderTab>(
    () => tabs[selectedIndex.value]?.id ?? tabs[0].id,
  )
  const {
    activeCards,
    closeConfirm,
    closeModal,
    confirmRemove,
    confirmState,
    configure,
    draggingId,
    initializeProviderCatalog,
    isTopProvider,
    modalState,
    onDragEnd,
    onDragStart,
    onDrop,
    openCreateModal,
    pinProvider,
    requestRemove,
    submitModal,
    updateProviderEnabled,
  } = useProviderCatalog({
    activeTab,
    t,
  })
  const {
    activeProxyBusy,
    activeProxyState,
    currentProxyLabel,
    handleTabActivated,
    initializeProviderStats,
    onProxyToggle,
    providerStatDisplay,
    usageHeatmap,
  } = useProviderStats({
    activeTab,
    locale,
    t,
  })
  const {
    initializeProviderPreferences,
    isDefaultProvider,
    showHeatmap,
    showHomeTitle,
    toggleDefaultProvider,
  } = useProviderPreferences({
    activeTab,
    t,
  })

  const loadAppVersion = async () => {
    try {
      const version = await fetchCurrentVersion()
      appVersion.value = version || ''
    } catch (error) {
      console.error('failed to load app version', error)
    }
  }

  const onTabChange = (idx: number) => {
    selectedIndex.value = idx
    const nextTab = tabs[idx]?.id
    if (nextTab) {
      handleTabActivated(nextTab as ProviderTab)
    }
  }

  onMounted(async () => {
    await Promise.all([
      initializeProviderCatalog(),
      initializeProviderPreferences(),
      initializeProviderStats(),
      loadAppVersion(),
    ])
  })

  return {
    activeTab,
    activeCards,
    activeProxyBusy,
    activeProxyState,
    appVersion,
    closeConfirm,
    closeModal,
    confirmRemove,
    confirmState,
    configure,
    currentProxyLabel,
    draggingId,
    isDefaultProvider,
    isTopProvider,
    modalState,
    onDragEnd,
    onDragStart,
    onDrop,
    onProxyToggle,
    onTabChange,
    openCreateModal,
    pinProvider,
    providerStatDisplay,
    requestRemove,
    selectedIndex,
    showHeatmap,
    showHomeTitle,
    submitModal,
    tabs,
    toggleDefaultProvider,
    updateProviderEnabled,
    usageHeatmap,
  }
}
