import { computed, reactive, ref, type ComputedRef } from 'vue'
import { automationCardGroups, createAutomationCards, type AutomationCard } from '../../data/cards'
import { providerTabs, providerTabIds, type ProviderTab } from '../../constants/platforms'
import { getIconOptions } from '../../icons/lobeIconMap'
import { loadProviders, saveProviders } from '../../services/providers'
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

export const useProviderCatalog = ({ activeTab, t }: UseProviderCatalogOptions) => {
  const cards = buildDefaultCards()
  const draggingId = ref<number | null>(null)
  const editingCard = ref<AutomationCard | null>(null)
  const defaultIconKey = getIconOptions()[0] ?? 'aicoding'

  const createDefaultForm = (): VendorForm => ({
    name: '',
    apiUrl: '',
    apiKey: '',
    officialSite: '',
    icon: defaultIconKey,
    enabled: true,
    supportedModels: {},
    modelMapping: {},
  })

  const modalState = reactive({
    open: false,
    tabId: providerTabs[0].id as ProviderTab,
    editingId: null as number | null,
    form: createDefaultForm(),
    errors: {
      apiUrl: '',
    },
  })

  const confirmState = reactive({
    open: false,
    card: null as AutomationCard | null,
    tabId: providerTabs[0].id as ProviderTab,
  })

  const activeCards = computed(() => cards[activeTab.value] ?? [])

  const serializeProviders = (providers: AutomationCard[]) =>
    providers.map((provider) => ({ ...provider }))

  const persistProviders = async (tabId: ProviderTab) => {
    try {
      await saveProviders(tabId, serializeProviders(cards[tabId]))
    } catch (error) {
      console.error('Failed to save providers', error)
    }
  }

  const replaceProviders = (tabId: ProviderTab, data: AutomationCard[]) => {
    cards[tabId].splice(0, cards[tabId].length, ...createAutomationCards(data))
  }

  const initializeProviderCatalog = async () => {
    await Promise.all(
      providerTabIds.map(async (tab) => {
        try {
          const saved = await loadProviders(tab)
          if (Array.isArray(saved)) {
            replaceProviders(tab, saved as AutomationCard[])
          } else {
            await persistProviders(tab)
          }
        } catch (error) {
          console.error('Failed to load providers', error)
        }
      }),
    )
  }

  const resetModalForm = () => {
    Object.assign(modalState.form, createDefaultForm())
    modalState.errors.apiUrl = ''
  }

  const normalizeIconKey = (icon: string) => {
    return icon.toString().trim().toLowerCase() || defaultIconKey
  }

  const buildCardValues = (form: VendorForm) => ({
    apiKey: form.apiKey.trim(),
    apiUrl: form.apiUrl.trim(),
    enabled: form.enabled,
    icon: normalizeIconKey(form.icon || defaultIconKey),
    modelMapping: form.modelMapping || {},
    officialSite: form.officialSite.trim(),
    supportedModels: form.supportedModels || {},
  })

  const openCreateModal = () => {
    modalState.tabId = activeTab.value
    modalState.editingId = null
    editingCard.value = null
    resetModalForm()
    modalState.open = true
  }

  const openEditModal = (card: AutomationCard) => {
    modalState.tabId = activeTab.value
    modalState.editingId = card.id
    editingCard.value = card
    Object.assign(modalState.form, {
      name: card.name,
      ...buildCardValues(card),
    })
    modalState.errors.apiUrl = ''
    modalState.open = true
  }

  const closeModal = () => {
    modalState.open = false
  }

  const closeConfirm = () => {
    confirmState.open = false
    confirmState.card = null
  }

  const validateApiUrl = (value: string) => {
    try {
      const parsed = new URL(value)
      if (!/^https?:/.test(parsed.protocol)) {
        throw new Error('protocol')
      }
      return true
    } catch {
      modalState.errors.apiUrl = t('components.main.form.errors.invalidUrl')
      return false
    }
  }

  const buildNewCard = (form: VendorForm): AutomationCard => ({
    id: Date.now(),
    name: form.name.trim() || 'Untitled vendor',
    accent: '#0a84ff',
    tint: 'rgba(15, 23, 42, 0.12)',
    ...buildCardValues(form),
  })

  const submitModal = () => {
    const list = cards[modalState.tabId]
    if (!list) return

    const apiUrl = modalState.form.apiUrl.trim()
    modalState.errors.apiUrl = ''
    if (!validateApiUrl(apiUrl)) {
      return
    }

    const nextValues = {
      ...buildCardValues(modalState.form),
      apiUrl,
    }

    if (editingCard.value) {
      Object.assign(editingCard.value, nextValues)
    } else {
      list.unshift(buildNewCard(modalState.form))
    }

    void persistProviders(modalState.tabId)
    closeModal()
  }

  const configure = (card: AutomationCard) => {
    openEditModal(card)
  }

  const remove = (id: number, tabId: ProviderTab = activeTab.value) => {
    const list = cards[tabId]
    if (!list) return

    const index = list.findIndex((card) => card.id === id)
    if (index > -1) {
      list.splice(index, 1)
      void persistProviders(tabId)
    }
  }

  const requestRemove = (card: AutomationCard) => {
    confirmState.card = card
    confirmState.tabId = activeTab.value
    confirmState.open = true
  }

  const confirmRemove = () => {
    if (!confirmState.card) return
    remove(confirmState.card.id, confirmState.tabId)
    closeConfirm()
  }

  const isTopProvider = (cardId: number) => activeCards.value[0]?.id === cardId

  const moveCardToFront = (tabId: ProviderTab, cardId: number) => {
    const list = cards[tabId]
    if (!list || list[0]?.id === cardId) return false

    const fromIndex = list.findIndex((card) => card.id === cardId)
    if (fromIndex < 0) return false

    const [moved] = list.splice(fromIndex, 1)
    list.unshift(moved)
    return true
  }

  const pinProvider = (cardId: number) => {
    if (!moveCardToFront(activeTab.value, cardId)) return
    void persistProviders(activeTab.value)
  }

  const updateProviderEnabled = (card: AutomationCard, enabled: boolean) => {
    card.enabled = enabled
    void persistProviders(activeTab.value)
  }

  const onDragStart = (id: number, event: DragEvent) => {
    draggingId.value = id
    if (event.dataTransfer) {
      event.dataTransfer.effectAllowed = 'move'
    }
  }

  const onDrop = (targetId: number) => {
    if (draggingId.value === null || draggingId.value === targetId) return

    const currentTab = activeTab.value
    const list = cards[currentTab]
    if (!list) return

    const fromIndex = list.findIndex((card) => card.id === draggingId.value)
    const toIndex = list.findIndex((card) => card.id === targetId)
    if (fromIndex === -1 || toIndex === -1) return

    const [moved] = list.splice(fromIndex, 1)
    list.splice(toIndex, 0, moved)
    draggingId.value = null
    void persistProviders(currentTab)
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
