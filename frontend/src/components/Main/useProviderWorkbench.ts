import { computed, onMounted, ref } from 'vue'
import { fetchCurrentVersion } from '../../services/version'
import { providerTabs, type ProviderTab } from '../../constants/platforms'
import {
  exportProviderConfig,
  importProviderConfig,
  type ProviderConfigBundle,
} from '../../services/routing'
import { getErrorMessage } from '../../utils/errors'
import { showToast } from '../../utils/toast'
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
  const providerConfigBusy = ref(false)
  const providerConfigInput = ref<HTMLInputElement | null>(null)

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

  const reloadProviderState = async () => {
    await Promise.all([
      initializeProviderCatalog(),
      initializeProviderPreferences(true),
      initializeProviderStats(),
    ])
  }

  const formatTimestamp = (value: Date) => {
    const pad = (num: number) => num.toString().padStart(2, '0')
    return `${value.getFullYear()}${pad(value.getMonth() + 1)}${pad(value.getDate())}-${pad(value.getHours())}${pad(value.getMinutes())}${pad(value.getSeconds())}`
  }

  const downloadJsonFile = (filename: string, payload: unknown) => {
    const blob = new Blob([JSON.stringify(payload, null, 2)], {
      type: 'application/json;charset=utf-8',
    })
    const url = URL.createObjectURL(blob)
    const anchor = document.createElement('a')
    anchor.href = url
    anchor.download = filename
    anchor.rel = 'noopener'
    document.body.appendChild(anchor)
    anchor.click()
    anchor.remove()
    URL.revokeObjectURL(url)
  }

  const exportProviders = async () => {
    if (providerConfigBusy.value) return

    providerConfigBusy.value = true
    try {
      const bundle = await exportProviderConfig()
      downloadJsonFile(
        `codeswitch-provider-config-${formatTimestamp(new Date())}.json`,
        bundle,
      )
      showToast(t('components.main.providerConfig.exportSuccess'), 'success')
    } catch (error) {
      console.error('Failed to export provider config', error)
      showToast(getErrorMessage(error, t('components.main.providerConfig.exportFailed')), 'error')
    } finally {
      providerConfigBusy.value = false
    }
  }

  const importProviders = async (file: File) => {
    if (providerConfigBusy.value) return
    if (!window.confirm(t('components.main.providerConfig.importConfirm'))) {
      return
    }

    providerConfigBusy.value = true
    try {
      const raw = await file.text()
      const parsed = JSON.parse(raw) as ProviderConfigBundle
      await importProviderConfig(parsed)
      await reloadProviderState()
      showToast(t('components.main.providerConfig.importSuccess'), 'success')
    } catch (error) {
      console.error('Failed to import provider config', error)
      showToast(getErrorMessage(error, t('components.main.providerConfig.importFailed')), 'error')
    } finally {
      providerConfigBusy.value = false
    }
  }

  const openImportDialog = () => {
    providerConfigInput.value?.click()
  }

  const onImportFileChange = async (event: Event) => {
    const input = event.target as HTMLInputElement | null
    const file = input?.files?.[0] ?? null
    if (input) {
      input.value = ''
    }
    if (!file) {
      return
    }
    await importProviders(file)
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
    exportProviders,
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
    openImportDialog,
    openCreateModal,
    pinProvider,
    providerConfigBusy,
    providerConfigInput,
    providerStatDisplay,
    requestRemove,
    selectedIndex,
    showHeatmap,
    showHomeTitle,
    submitModal,
    tabs,
    toggleDefaultProvider,
    usageHeatmap,
    onImportFileChange,
  }
}
