import { computed, onUnmounted, reactive, ref, type ComputedRef } from 'vue'
import {
  buildUsageHeatmapMatrix,
  calculateHeatmapDayRange,
  DEFAULT_HEATMAP_DAYS,
  generateFallbackUsageHeatmap,
  type UsageHeatmapWeek,
} from '../../data/usageHeatmap'
import { providerTabIds, type ProviderTab } from '../../constants/platforms'
import { disableProxy, enableProxy, fetchProxyStatus } from '../../services/claudeSettings'
import {
  fetchHeatmapStats,
  fetchProviderDailyStats,
  type ProviderDailyStat,
} from '../../services/logs'
import type { LocaleRef, ProviderStatDisplay, TranslateFn } from './types'

type UseProviderStatsOptions = {
  activeTab: ComputedRef<ProviderTab>
  locale: LocaleRef
  t: TranslateFn
}

const HEATMAP_DAYS = DEFAULT_HEATMAP_DAYS
const SUCCESS_RATE_THRESHOLDS = {
  healthy: 0.95,
  warning: 0.8,
} as const

const createTabRecord = <T>(initialValue: T): Record<ProviderTab, T> => ({
  claude: initialValue,
  codex: initialValue,
  gemini: initialValue,
})

const clamp = (value: number, min: number, max: number) => {
  if (max <= min) return min
  return Math.min(Math.max(value, min), max)
}

const formatMetric = (value: number) => value.toLocaleString()
const normalizeProviderKey = (value: string) => value?.trim().toLowerCase() ?? ''

export const useProviderStats = ({ activeTab, locale, t }: UseProviderStatsOptions) => {
  const usageHeatmap = ref<UsageHeatmapWeek[]>(generateFallbackUsageHeatmap(HEATMAP_DAYS))

  const proxyStates = reactive(createTabRecord(false))
  const proxyBusy = reactive(createTabRecord(false))
  const providerStatsLoaded = reactive(createTabRecord(false))
  const providerStatsMap = reactive<Record<ProviderTab, Record<string, ProviderDailyStat>>>({
    claude: {},
    codex: {},
    gemini: {},
  })

  let providerStatsTimer: number | undefined
  const providerStatsRequests = createTabRecord<Promise<void> | null>(null)

  const currencyFormatter = computed(
    () =>
      new Intl.NumberFormat(locale.value || 'en', {
        style: 'currency',
        currency: 'USD',
        minimumFractionDigits: 2,
        maximumFractionDigits: 2,
      }),
  )

  const activeProxyState = computed(() => proxyStates[activeTab.value])
  const activeProxyBusy = computed(() => proxyBusy[activeTab.value])
  const currentProxyLabel = computed(() => {
    if (activeTab.value === 'claude') {
      return t('components.main.relayToggle.hostClaude')
    }
    if (activeTab.value === 'codex') {
      return t('components.main.relayToggle.hostCodex')
    }
    return t('components.main.relayToggle.hostGemini')
  })

  const loadUsageHeatmap = async () => {
    try {
      const rangeDays = calculateHeatmapDayRange(HEATMAP_DAYS)
      const stats = await fetchHeatmapStats(rangeDays)
      usageHeatmap.value = buildUsageHeatmapMatrix(stats, HEATMAP_DAYS)
    } catch (error) {
      console.error('Failed to load usage heatmap stats', error)
    }
  }

  const refreshProxyState = async (tab: ProviderTab) => {
    try {
      const status = await fetchProxyStatus(tab)
      proxyStates[tab] = Boolean(status?.enabled)
    } catch (error) {
      console.error(`Failed to fetch proxy status for ${tab}`, error)
      proxyStates[tab] = false
    }
  }

  const loadProviderStats = async (tab: ProviderTab) => {
    if (providerStatsRequests[tab]) {
      return providerStatsRequests[tab]
    }

    providerStatsRequests[tab] = (async () => {
      try {
        const stats = await fetchProviderDailyStats(tab)
        const mapped: Record<string, ProviderDailyStat> = {}
        ;(stats ?? []).forEach((stat) => {
          mapped[normalizeProviderKey(stat.provider)] = stat
        })
        const hadExistingStats = Object.keys(providerStatsMap[tab] ?? {}).length > 0
        if ((stats?.length ?? 0) > 0 || !hadExistingStats) {
          providerStatsMap[tab] = mapped
        }
        providerStatsLoaded[tab] = true
      } catch (error) {
        console.error(`Failed to load provider stats for ${tab}`, error)
        if (!providerStatsLoaded[tab]) {
          providerStatsLoaded[tab] = true
        }
      } finally {
        providerStatsRequests[tab] = null
      }
    })()

    try {
      await providerStatsRequests[tab]
    } finally {
      providerStatsRequests[tab] = null
    }
  }

  const startProviderStatsTimer = () => {
    stopProviderStatsTimer()
    providerStatsTimer = window.setInterval(() => {
      void loadProviderStats(activeTab.value)
    }, 300_000)
  }

  const stopProviderStatsTimer = () => {
    if (providerStatsTimer) {
      clearInterval(providerStatsTimer)
      providerStatsTimer = undefined
    }
  }

  const initializeProviderStats = async () => {
    void loadUsageHeatmap()
    await Promise.all(providerTabIds.map(refreshProxyState))
    await loadProviderStats(activeTab.value)
    startProviderStatsTimer()
  }

  const handleTabActivated = (tab: ProviderTab) => {
    void refreshProxyState(tab)
    if (!providerStatsLoaded[tab]) {
      void loadProviderStats(tab)
    }
  }

  const formatSuccessRateLabel = (value: number) => {
    const percent = clamp(value, 0, 1) * 100
    const decimals = percent >= 99.5 || percent === 0 ? 0 : 1
    return `${t('components.main.providers.successRate')}: ${percent.toFixed(decimals)}%`
  }

  const successRateClassName = (value: number) => {
    const rate = clamp(value, 0, 1)
    if (rate >= SUCCESS_RATE_THRESHOLDS.healthy) return 'success-good'
    if (rate >= SUCCESS_RATE_THRESHOLDS.warning) return 'success-warn'
    return 'success-bad'
  }

  const providerStatDisplay = (providerName: string): ProviderStatDisplay => {
    const tab = activeTab.value
    if (!providerStatsLoaded[tab]) {
      return { state: 'loading', message: t('components.main.providers.loading') }
    }

    const stat = providerStatsMap[tab]?.[normalizeProviderKey(providerName)]
    if (!stat) {
      return { state: 'empty', message: t('components.main.providers.noData') }
    }

    const totalTokens = stat.input_tokens + stat.output_tokens
    const successRateValue = Number.isFinite(stat.success_rate)
      ? clamp(stat.success_rate, 0, 1)
      : null

    return {
      state: 'ready',
      requests: `${t('components.main.providers.requests')}: ${formatMetric(stat.total_requests)}`,
      tokens: `${t('components.main.providers.tokens')}: ${formatMetric(totalTokens)}`,
      cost: `${t('components.main.providers.cost')}: ${currencyFormatter.value.format(Math.max(stat.cost_total, 0))}`,
      successRateLabel:
        successRateValue !== null ? formatSuccessRateLabel(successRateValue) : '',
      successRateClass:
        successRateValue !== null ? successRateClassName(successRateValue) : '',
    }
  }

  const onProxyToggle = async () => {
    const tab = activeTab.value
    if (proxyBusy[tab]) return

    proxyBusy[tab] = true
    const nextState = !proxyStates[tab]
    try {
      if (nextState) {
        await enableProxy(tab)
      } else {
        await disableProxy(tab)
      }
      proxyStates[tab] = nextState
    } catch (error) {
      console.error(`Failed to toggle proxy for ${tab}`, error)
    } finally {
      proxyBusy[tab] = false
    }
  }

  onUnmounted(() => {
    stopProviderStatsTimer()
  })

  return {
    activeProxyBusy,
    activeProxyState,
    currentProxyLabel,
    handleTabActivated,
    initializeProviderStats,
    onProxyToggle,
    providerStatDisplay,
    usageHeatmap,
  }
}
