import { computed, onMounted, ref } from 'vue'
import { fetchCurrentVersion } from '../../services/version'
import type { AutomationCard } from '../../data/cards'
import { providerTabs, type ProviderTab } from '../../constants/platforms'
import { showToast } from '../../utils/toast'
import type { TranslateFn } from './types'
import { useProviderCatalog } from './useProviderCatalog'
import { useProviderPreferences } from './useProviderPreferences'
import { useProviderStats } from './useProviderStats'

type UseProviderWorkbenchOptions = {
  t: TranslateFn
}

export const useProviderWorkbench = ({ t }: UseProviderWorkbenchOptions) => {
  const tabs = providerTabs
  const appVersion = ref('')
  const selectedIndex = ref(0)

  const activeTab = computed<ProviderTab>(
    () => tabs[selectedIndex.value]?.id ?? tabs[0].id,
  )
  const {
    activeCards,
    archivedCards,
    closeConfirm,
    closeCopy,
    closeModal,
    confirmCopy,
    confirmRemove,
    confirmState,
    configure,
    copyState,
    draggingId,
    initializeProviderCatalog,
    isTopProvider,
    modalState,
    onDragEnd,
    onDragStart,
    onDrop,
    openCreateModal,
    pinProvider,
    requestCopy,
    requestRemove,
    submitModal,
    toggleArchive,
  } = useProviderCatalog({
    activeTab,
    t,
    selectTab: (tabId) => {
      const idx = tabs.findIndex((tab) => tab.id === tabId)
      if (idx >= 0) {
        onTabChange(idx)
      }
    },
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

  // 归档默认供应商被后端 Validate 拒绝，前端先行拦截给出可操作提示。
  const archiveProvider = (card: AutomationCard) => {
    if (isDefaultProvider(card.name)) {
      showToast(t('components.main.archiveBlockedBecauseDefault'), 'error')
      return
    }
    toggleArchive(card.id)
  }

  const restoreProvider = (card: AutomationCard) => {
    toggleArchive(card.id)
  }

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
    archiveProvider,
    archivedCards,
    closeConfirm,
    closeCopy,
    closeModal,
    confirmCopy,
    confirmRemove,
    confirmState,
    configure,
    copyState,
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
    requestCopy,
    requestRemove,
    restoreProvider,
    selectedIndex,
    showHeatmap,
    showHomeTitle,
    submitModal,
    tabs,
    toggleDefaultProvider,
    usageHeatmap,
  }
}
