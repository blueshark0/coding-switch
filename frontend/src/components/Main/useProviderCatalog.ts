import { computed, reactive, ref, type ComputedRef } from 'vue'
import { automationCardGroups, createAutomationCards, type AutomationCard } from '../../data/cards'
import { providerTabs, providerTabIds, type ProviderTab } from '../../constants/platforms'
import { getIconOptions } from '../../icons/lobeIconMap'
import { loadProviders, saveProviders } from '../../services/providers'
import { getErrorMessage } from '../../utils/errors'
import { showToast } from '../../utils/toast'
import type { TranslateFn, VendorForm } from './types'

type UseProviderCatalogOptions = {
  activeTab: ComputedRef<ProviderTab>
  t: TranslateFn
}

const buildDefaultCards = () =>
  reactive<Record<ProviderTab, AutomationCard[]>>({
    claude: createAutomationCards(automationCardGroups.claude),
    codex: createAutomationCards(automationCardGroups.codex),
    gemini: createAutomationCards(automationCardGroups.gemini),
  })

const moveCardToFrontInList = (providers: AutomationCard[], cardId: number) => {
  if (providers[0]?.id === cardId) {
    return false
  }

  const fromIndex = providers.findIndex((card) => card.id === cardId)
  if (fromIndex < 0) {
    return false
  }

  const [moved] = providers.splice(fromIndex, 1)
  providers.unshift(moved)
  return true
}

export const useProviderCatalog = ({ activeTab, t }: UseProviderCatalogOptions) => {
  const cards = buildDefaultCards()
  const draggingId = ref<number | null>(null)
  const defaultIconKey = getIconOptions()[0] ?? 'aicoding'
  const savingTabs = reactive<Record<ProviderTab, boolean>>({
    claude: false,
    codex: false,
    gemini: false,
  })

  const createDefaultForm = (): VendorForm => ({
    name: '',
    apiUrl: '',
    apiKey: '',
    officialSite: '',
    icon: defaultIconKey,
    enabled: true,
    supportedModels: {},
    modelMapping: {},
    proxyMode: '',
    proxyUrl: '',
  })

  const modalState = reactive({
    open: false,
    tabId: providerTabs[0].id as ProviderTab,
    editingId: null as number | null,
    form: createDefaultForm(),
    errors: {
      name: '',
      apiUrl: '',
      proxyUrl: '',
    },
  })

  const confirmState = reactive({
    open: false,
    card: null as AutomationCard | null,
    tabId: providerTabs[0].id as ProviderTab,
  })

  const activeCards = computed(() => cards[activeTab.value] ?? [])

  const serializeProviders = (providers: AutomationCard[]) =>
    providers.map((provider, index) => ({ ...provider, position: index + 1 }))

  const replaceProviders = (tabId: ProviderTab, data: AutomationCard[]) => {
    cards[tabId].splice(0, cards[tabId].length, ...createAutomationCards(data))
  }

  const cloneCards = (data: AutomationCard[]) => createAutomationCards(data)

  const persistProviders = async (tabId: ProviderTab) => {
    await saveProviders(tabId, serializeProviders(cards[tabId]))
  }

  const initializeProviderCatalog = async () => {
    await Promise.all(
      providerTabIds.map(async (tab) => {
        try {
          replaceProviders(tab, await loadProviders(tab))
        } catch (error) {
          console.error('Failed to load providers', error)
          showToast(getErrorMessage(error, t('components.main.providerSaveFailed')), 'error')
        }
      }),
    )
  }

  const resetModalForm = () => {
    Object.assign(modalState.form, createDefaultForm())
    modalState.errors.name = ''
    modalState.errors.apiUrl = ''
    modalState.errors.proxyUrl = ''
  }

  const normalizeIconKey = (icon: string) => {
    return icon.toString().trim().toLowerCase() || defaultIconKey
  }

  const normalizeProviderName = (name: string) => name.trim().toLowerCase()

  const buildCardValues = (form: VendorForm) => ({
    apiKey: form.apiKey.trim(),
    apiUrl: form.apiUrl.trim(),
    enabled: form.enabled,
    icon: normalizeIconKey(form.icon || defaultIconKey),
    modelMapping: { ...(form.modelMapping ?? {}) },
    officialSite: form.officialSite.trim(),
    supportedModels: { ...(form.supportedModels ?? {}) },
    proxyMode: form.proxyMode ?? '',
    proxyUrl: (form.proxyUrl ?? '').trim(),
  })

  const nextProviderID = (tabId: ProviderTab) => {
    const currentIDs = cards[tabId].map((card) => card.id)
    return Math.max(0, ...currentIDs) + 1
  }

  const hasDuplicateName = (tabId: ProviderTab, name: string, editingID: number | null) => {
    const normalizedName = normalizeProviderName(name)
    return cards[tabId].some((provider) => {
      if (editingID !== null && provider.id === editingID) {
        return false
      }
      return normalizeProviderName(provider.name) === normalizedName
    })
  }

  const validateProviderName = (tabId: ProviderTab, editingID: number | null) => {
    const name = modalState.form.name.trim()
    if (!name) {
      modalState.errors.name = t('components.main.form.errors.nameRequired')
      return false
    }
    if (hasDuplicateName(tabId, name, editingID)) {
      modalState.errors.name = t('components.main.form.errors.duplicateName')
      return false
    }
    modalState.errors.name = ''
    return true
  }

  const validateApiUrl = (value: string) => {
    try {
      const parsed = new URL(value)
      if (!/^https?:/.test(parsed.protocol)) {
        throw new Error('protocol')
      }
      modalState.errors.apiUrl = ''
      return true
    } catch {
      modalState.errors.apiUrl = t('components.main.form.errors.invalidUrl')
      return false
    }
  }

  const PROXY_PREFIXES = ['http://', 'https://', 'socks5://', 'socks5h://']

  const validateProxyUrl = (mode: string, value: string) => {
    if (mode !== 'custom') {
      modalState.errors.proxyUrl = ''
      return true
    }
    if (!value.trim()) {
      modalState.errors.proxyUrl = t('components.main.form.errors.proxyUrlRequired')
      return false
    }
    if (!PROXY_PREFIXES.some((p) => value.trim().startsWith(p))) {
      modalState.errors.proxyUrl = t('components.main.form.errors.invalidProxyUrl')
      return false
    }
    modalState.errors.proxyUrl = ''
    return true
  }

  const saveMutation = async (
    tabId: ProviderTab,
    mutate: (providers: AutomationCard[]) => boolean | void,
  ) => {
    if (savingTabs[tabId]) {
      return false
    }

    const snapshot = cloneCards(cards[tabId])
    const changed = mutate(cards[tabId])
    if (changed === false) {
      return false
    }

    savingTabs[tabId] = true
    try {
      await persistProviders(tabId)
      return true
    } catch (error) {
      console.error('Failed to save providers', error)
      replaceProviders(tabId, snapshot)
      showToast(getErrorMessage(error, t('components.main.providerSaveFailed')), 'error')
      return false
    } finally {
      savingTabs[tabId] = false
    }
  }

  const buildNewCard = (form: VendorForm): AutomationCard => ({
    id: nextProviderID(modalState.tabId),
    name: form.name.trim(),
    accent: '#0a84ff',
    tint: 'rgba(15, 23, 42, 0.12)',
    ...buildCardValues(form),
  })

  const openCreateModal = () => {
    modalState.tabId = activeTab.value
    modalState.editingId = null
    resetModalForm()
    modalState.open = true
  }

  const openEditModal = (card: AutomationCard) => {
    modalState.tabId = activeTab.value
    modalState.editingId = card.id
    Object.assign(modalState.form, {
      name: card.name,
      ...buildCardValues(card),
    })
    modalState.errors.name = ''
    modalState.errors.apiUrl = ''
    modalState.errors.proxyUrl = ''
    modalState.open = true
  }

  const closeModal = () => {
    modalState.open = false
  }

  const closeConfirm = () => {
    confirmState.open = false
    confirmState.card = null
  }

  const submitModal = async () => {
    const tabId = modalState.tabId
    const apiUrl = modalState.form.apiUrl.trim()
    if (
      !validateProviderName(tabId, modalState.editingId) ||
      !validateApiUrl(apiUrl) ||
      !validateProxyUrl(modalState.form.proxyMode ?? '', modalState.form.proxyUrl ?? '')
    ) {
      return
    }

    const nextValues = {
      ...buildCardValues(modalState.form),
      apiUrl,
    }

    const success = await saveMutation(tabId, (providers) => {
      if (modalState.editingId !== null) {
        const editingIndex = providers.findIndex((card) => card.id === modalState.editingId)
        if (editingIndex < 0) {
          return false
        }
        providers.splice(editingIndex, 1, {
          ...providers[editingIndex],
          ...nextValues,
        })
        return true
      }

      providers.unshift(buildNewCard(modalState.form))
      return true
    })

    if (success) {
      closeModal()
    }
  }

  const configure = (card: AutomationCard) => {
    openEditModal(card)
  }

  const remove = async (id: number, tabId: ProviderTab = activeTab.value) => {
    await saveMutation(tabId, (providers) => {
      const index = providers.findIndex((card) => card.id === id)
      if (index < 0) {
        return false
      }
      providers.splice(index, 1)
      return true
    })
  }

  const requestRemove = (card: AutomationCard) => {
    confirmState.card = card
    confirmState.tabId = activeTab.value
    confirmState.open = true
  }

  const confirmRemove = async () => {
    if (!confirmState.card) {
      return
    }
    await remove(confirmState.card.id, confirmState.tabId)
    closeConfirm()
  }

  const isTopProvider = (cardId: number) => activeCards.value[0]?.id === cardId

  const pinProvider = (cardId: number) => {
    void saveMutation(activeTab.value, (providers) => moveCardToFrontInList(providers, cardId))
  }

  const updateProviderEnabled = (card: AutomationCard, enabled: boolean) => {
    const tabId = activeTab.value
    void saveMutation(tabId, (providers) => {
      const index = providers.findIndex((provider) => provider.id === card.id)
      if (index < 0) {
        return false
      }
      providers.splice(index, 1, {
        ...providers[index],
        enabled,
      })
      return true
    })
  }

  const onDragStart = (id: number, event: DragEvent) => {
    draggingId.value = id
    if (event.dataTransfer) {
      event.dataTransfer.effectAllowed = 'move'
    }
  }

  const onDrop = (targetId: number) => {
    if (draggingId.value === null || draggingId.value === targetId) {
      return
    }

    const tabId = activeTab.value
    const draggingCardID = draggingId.value
    draggingId.value = null
    void saveMutation(tabId, (providers) => {
      const fromIndex = providers.findIndex((card) => card.id === draggingCardID)
      const toIndex = providers.findIndex((card) => card.id === targetId)
      if (fromIndex < 0 || toIndex < 0) {
        return false
      }

      const [moved] = providers.splice(fromIndex, 1)
      providers.splice(toIndex, 0, moved)
      return true
    })
  }

  const onDragEnd = () => {
    draggingId.value = null
  }

  return {
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
  }
}
