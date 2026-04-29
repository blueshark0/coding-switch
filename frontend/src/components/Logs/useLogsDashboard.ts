import { computed, onMounted, onUnmounted, reactive, ref, watch, type Ref } from 'vue'
import {
  DEFAULT_LOG_RANGE,
  DEFAULT_LOG_PAGE_SIZE,
  fetchRequestLogs,
  fetchLogProviders,
  fetchLogStats,
  LOG_PAGE_SIZE_OPTIONS,
  LOG_RANGE_OPTIONS,
  normalizeLogPageSize,
  normalizeLogRangeKey,
  type LogStats,
  type LogRangeKey,
  type RequestLog,
} from '../../services/logs'
import { useLogsPresentation } from './useLogsPresentation'
import type { CostTier } from './costTier'

type TranslateFn = (key: string, named?: Record<string, unknown>) => string

type UseLogsDashboardOptions = {
  getCssVarValue: (name: string, fallback: string) => string
  isDarkMode: Ref<boolean>
  t: TranslateFn
}

const REFRESH_INTERVAL = 30

export const useLogsDashboard = ({
  getCssVarValue,
  isDarkMode,
  t,
}: UseLogsDashboardOptions) => {
  const logs = ref<RequestLog[]>([])
  const stats = ref<LogStats | null>(null)
  const loading = ref(false)
  const filters = reactive<{
    platform: string
    provider: string
    rangeKey: LogRangeKey
    costTier: CostTier
  }>({
    platform: '',
    provider: '',
    rangeKey: DEFAULT_LOG_RANGE,
    costTier: 'all',
  })
  const page = ref(1)
  const pageSize = ref(DEFAULT_LOG_PAGE_SIZE)
  const totalLogs = ref(0)
  const providerOptions = ref<string[]>([])
  const countdown = ref(REFRESH_INTERVAL)
  let timer: number | undefined
  let dashboardLoadPromise: Promise<void> | null = null
  let queuedDashboardReload = false
  const selectedRangeKey = computed(() => normalizeLogRangeKey(filters.rangeKey))
  const rangeOptions = computed(() =>
    LOG_RANGE_OPTIONS.map((value) => ({
      value,
      label: t(`components.logs.filters.ranges.${value}`),
    })),
  )
  const costTierOptions = computed<Array<{ value: CostTier; label: string }>>(() => [
    {
      value: 'all',
      label: t('components.logs.filters.allCostTiers'),
    },
    {
      value: 'low',
      label: t('components.logs.filters.costTiers.low'),
    },
    {
      value: 'medium',
      label: t('components.logs.filters.costTiers.medium'),
    },
    {
      value: 'high',
      label: t('components.logs.filters.costTiers.high'),
    },
  ])

  const pageSizeOptions = computed(() => LOG_PAGE_SIZE_OPTIONS.map((value) => ({ value })))
  const pagedLogs = computed(() => logs.value)
  const totalPages = computed(() => Math.max(1, Math.ceil(totalLogs.value / pageSize.value)))
  const {
    chartData,
    chartOptions,
    durationColor,
    formatDuration,
    formatCurrency,
    formatNumber,
    formatStream,
    formatTime,
    httpCodeClass,
    statsCards,
  } = useLogsPresentation({
    getCssVarValue,
    isDarkMode,
    stats,
    rangeKey: selectedRangeKey,
    t,
  })

  const resetTimer = () => {
    countdown.value = REFRESH_INTERVAL
  }

  const loadDashboard = (options: { skipIfLoading?: boolean } = {}) => {
    if (dashboardLoadPromise) {
      return options.skipIfLoading ? null : dashboardLoadPromise
    }

    dashboardLoadPromise = (async () => {
      loading.value = true
      try {
        do {
          queuedDashboardReload = false
          const requestedPage = Math.max(1, Math.floor(page.value))
          const requestedPageSize = normalizeLogPageSize(pageSize.value)
          const [logData, statData] = await Promise.all([
            fetchRequestLogs({
              platform: filters.platform,
              provider: filters.provider,
              rangeKey: selectedRangeKey.value,
              costTier: filters.costTier,
              page: requestedPage,
              pageSize: requestedPageSize,
            }),
            fetchLogStats({
              platform: filters.platform,
              provider: filters.provider,
              rangeKey: selectedRangeKey.value,
              costTier: filters.costTier,
            }),
          ])
          const nextLogs = logData?.items ?? []
          logs.value = nextLogs
          totalLogs.value = logData?.total ?? 0
          pageSize.value = normalizeLogPageSize(logData?.page_size ?? requestedPageSize)
          stats.value = statData ?? null
          const nextTotalPages = Math.max(1, Math.ceil(totalLogs.value / pageSize.value))
          if (requestedPage > nextTotalPages) {
            page.value = nextTotalPages
            queuedDashboardReload = true
          } else {
            page.value = logData?.page ?? requestedPage
          }
        } while (queuedDashboardReload)
      } catch (error) {
        console.error('failed to load dashboard data', error)
      } finally {
        loading.value = false
        dashboardLoadPromise = null
      }
    })()

    return dashboardLoadPromise
  }

  const loadProviderOptions = async () => {
    try {
      const list = await fetchLogProviders(filters.platform)
      providerOptions.value = list ?? []
      if (filters.provider && !providerOptions.value.includes(filters.provider)) {
        filters.provider = ''
      }
    } catch (error) {
      console.error('failed to load provider options', error)
    }
  }

  const stopCountdown = () => {
    if (timer) {
      clearInterval(timer)
      timer = undefined
    }
  }

  const startCountdown = () => {
    stopCountdown()
    timer = window.setInterval(() => {
      if (countdown.value <= 1) {
        countdown.value = REFRESH_INTERVAL
        void loadDashboard({ skipIfLoading: true })
      } else {
        countdown.value -= 1
      }
    }, 1000)
  }

  const handleVisibilityChange = () => {
    if (document.hidden) {
      stopCountdown()
    } else {
      startCountdown()
    }
  }

  const applyFilters = async () => {
    page.value = 1
    if (dashboardLoadPromise) {
      queuedDashboardReload = true
    }
    await loadDashboard()
    resetTimer()
  }

  const manualRefresh = () => {
    resetTimer()
    if (dashboardLoadPromise) {
      queuedDashboardReload = true
    }
    void loadDashboard()
  }

  const nextPage = () => {
    if (page.value < totalPages.value) {
      page.value += 1
      void loadDashboard()
    }
  }

  const prevPage = () => {
    if (page.value > 1) {
      page.value -= 1
      void loadDashboard()
    }
  }

  const updatePageSize = () => {
    pageSize.value = normalizeLogPageSize(Number(pageSize.value))
    page.value = 1
    if (dashboardLoadPromise) {
      queuedDashboardReload = true
    }
    void loadDashboard()
  }

  watch(
    () => filters.platform,
    async () => {
      await loadProviderOptions()
    },
  )

  onMounted(async () => {
    await Promise.all([loadDashboard(), loadProviderOptions()])
    startCountdown()
    document.addEventListener('visibilitychange', handleVisibilityChange)
  })

  onUnmounted(() => {
    stopCountdown()
    document.removeEventListener('visibilitychange', handleVisibilityChange)
  })

  return {
    applyFilters,
    chartData,
    chartOptions,
    costTierOptions,
    countdown,
    durationColor,
    filters,
    formatDuration,
    formatCurrency,
    formatNumber,
    formatStream,
    formatTime,
    httpCodeClass,
    loading,
    manualRefresh,
    nextPage,
    page,
    pageSize,
    pageSizeOptions,
    pagedLogs,
    prevPage,
    providerOptions,
    rangeOptions,
    statsCards,
    totalLogs,
    totalPages,
    updatePageSize,
  }
}
