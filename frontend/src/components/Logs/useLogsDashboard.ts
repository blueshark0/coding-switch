import { computed, onMounted, onUnmounted, reactive, ref, watch, type Ref } from 'vue'
import {
  fetchRequestLogs,
  fetchLogProviders,
  fetchLogStats,
  type LogStats,
  type RequestLog,
} from '../../services/logs'
import { useLogsPresentation } from './useLogsPresentation'

type TranslateFn = (key: string, named?: Record<string, unknown>) => string

type UseLogsDashboardOptions = {
  getCssVarValue: (name: string, fallback: string) => string
  isDarkMode: Ref<boolean>
  t: TranslateFn
}

const PAGE_SIZE = 15
const REFRESH_INTERVAL = 30

export const useLogsDashboard = ({
  getCssVarValue,
  isDarkMode,
  t,
}: UseLogsDashboardOptions) => {
  const logs = ref<RequestLog[]>([])
  const stats = ref<LogStats | null>(null)
  const loading = ref(false)
  const filters = reactive({ platform: '', provider: '' })
  const page = ref(1)
  const providerOptions = ref<string[]>([])
  const countdown = ref(REFRESH_INTERVAL)
  let timer: number | undefined

  const pagedLogs = computed(() => {
    const start = (page.value - 1) * PAGE_SIZE
    return logs.value.slice(start, start + PAGE_SIZE)
  })

  const totalPages = computed(() => Math.max(1, Math.ceil(logs.value.length / PAGE_SIZE)))
  const {
    chartData,
    chartOptions,
    durationColor,
    formatDuration,
    formatNumber,
    formatStream,
    formatTime,
    httpCodeClass,
    statsCards,
  } = useLogsPresentation({
    getCssVarValue,
    isDarkMode,
    stats,
    t,
  })

  const resetTimer = () => {
    countdown.value = REFRESH_INTERVAL
  }

  const loadLogs = async () => {
    loading.value = true
    try {
      const data = await fetchRequestLogs({
        platform: filters.platform,
        provider: filters.provider,
        limit: 200,
      })
      logs.value = data ?? []
      page.value = Math.min(page.value, totalPages.value)
    } catch (error) {
      console.error('failed to load request logs', error)
    } finally {
      loading.value = false
    }
  }

  const loadStats = async () => {
    try {
      const data = await fetchLogStats({
        platform: filters.platform,
        provider: filters.provider,
      })
      stats.value = data ?? null
    } catch (error) {
      console.error('failed to load log stats', error)
    }
  }

  const loadDashboard = async () => {
    await Promise.all([loadLogs(), loadStats()])
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
        void loadDashboard()
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
    await loadDashboard()
    resetTimer()
  }

  const manualRefresh = () => {
    resetTimer()
    void loadDashboard()
  }

  const nextPage = () => {
    if (page.value < totalPages.value) {
      page.value += 1
    }
  }

  const prevPage = () => {
    if (page.value > 1) {
      page.value -= 1
    }
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
    countdown,
    durationColor,
    filters,
    formatDuration,
    formatNumber,
    formatStream,
    formatTime,
    httpCodeClass,
    loading,
    manualRefresh,
    nextPage,
    page,
    pagedLogs,
    prevPage,
    providerOptions,
    statsCards,
    totalPages,
  }
}
