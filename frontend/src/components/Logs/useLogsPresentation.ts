import { computed, type Ref } from 'vue'
import type { ChartOptions } from 'chart.js'
import type { LogStats, LogStatsSeries } from '../../services/logs'
import {
  formatDateTime,
  formatHourBucketLabel,
  padDatePart,
  parseDateTime,
  startOfTodayLocal,
} from '../../utils/dateTime'

type TranslateFn = (key: string, named?: Record<string, unknown>) => string

type UseLogsPresentationOptions = {
  getCssVarValue: (name: string, fallback: string) => string
  isDarkMode: Ref<boolean>
  stats: Ref<LogStats | null>
  t: TranslateFn
}

export const useLogsPresentation = ({
  getCssVarValue,
  isDarkMode,
  stats,
  t,
}: UseLogsPresentationOptions) => {
  const statsSeries = computed<LogStatsSeries[]>(() => stats.value?.series ?? [])

  const chartData = computed(() => {
    const series = statsSeries.value
    return {
      labels: series.map((item) => formatHourBucketLabel(item.day)),
      datasets: [
        {
          label: t('components.logs.tokenLabels.cost'),
          data: series.map((item) => Number((item.total_cost ?? 0).toFixed(4))),
          borderColor: '#f97316',
          backgroundColor: 'rgba(249, 115, 22, 0.2)',
          tension: 0.3,
          fill: false,
          yAxisID: 'yCost',
        },
        {
          label: t('components.logs.tokenLabels.input'),
          data: series.map((item) => item.input_tokens ?? 0),
          borderColor: '#34d399',
          backgroundColor: 'rgba(52, 211, 153, 0.25)',
          tension: 0.35,
          fill: true,
        },
        {
          label: t('components.logs.tokenLabels.output'),
          data: series.map((item) => item.output_tokens ?? 0),
          borderColor: '#60a5fa',
          backgroundColor: 'rgba(96, 165, 250, 0.2)',
          tension: 0.35,
          fill: true,
        },
        {
          label: t('components.logs.tokenLabels.reasoning'),
          data: series.map((item) => item.reasoning_tokens ?? 0),
          borderColor: '#f472b6',
          backgroundColor: 'rgba(244, 114, 182, 0.2)',
          tension: 0.35,
          fill: true,
        },
        {
          label: t('components.logs.tokenLabels.cacheWrite'),
          data: series.map((item) => item.cache_create_tokens ?? 0),
          borderColor: '#fbbf24',
          backgroundColor: 'rgba(251, 191, 36, 0.2)',
          tension: 0.35,
          fill: false,
        },
        {
          label: t('components.logs.tokenLabels.cacheRead'),
          data: series.map((item) => item.cache_read_tokens ?? 0),
          borderColor: '#38bdf8',
          backgroundColor: 'rgba(56, 189, 248, 0.15)',
          tension: 0.35,
          fill: false,
        },
      ],
    }
  })

  const chartOptions = computed<ChartOptions<'line'>>(() => {
    const legendColor = getCssVarValue('--mac-text', isDarkMode.value ? '#f8fafc' : '#0f172a')
    const axisColor = getCssVarValue(
      '--mac-text-secondary',
      isDarkMode.value ? '#cbd5f5' : '#94a3b8',
    )
    const axisStrongColor = getCssVarValue(
      '--mac-text',
      isDarkMode.value ? '#e2e8f0' : '#475569',
    )
    const gridColor = isDarkMode.value
      ? 'rgba(148, 163, 184, 0.35)'
      : 'rgba(148, 163, 184, 0.2)'

    return {
      responsive: true,
      maintainAspectRatio: false,
      interaction: {
        mode: 'index',
        intersect: false,
      },
      plugins: {
        legend: {
          labels: {
            color: legendColor,
            font: {
              size: 12,
              weight: 500,
            },
          },
        },
      },
      scales: {
        x: {
          grid: { display: false },
          ticks: { color: axisColor },
        },
        y: {
          beginAtZero: true,
          ticks: { color: axisColor },
          grid: { color: gridColor },
        },
        yCost: {
          position: 'right',
          beginAtZero: true,
          grid: { drawOnChartArea: false },
          ticks: {
            color: axisStrongColor,
            callback: (value: string | number) => {
              const numeric = typeof value === 'number' ? value : Number(value)
              if (Number.isNaN(numeric)) return '$0'
              if (numeric >= 1) return `$${numeric.toFixed(2)}`
              return `$${numeric.toFixed(4)}`
            },
          },
        },
      },
    }
  })

  const formatTime = (value?: string) => {
    const date = parseDateTime(value)
    if (!date) return value || '—'
    return formatDateTime(date)
  }

  const formatStream = (value?: boolean | number) => {
    const isOn = value === true || value === 1
    return isOn ? t('components.logs.streamOn') : t('components.logs.streamOff')
  }

  const formatDuration = (value?: number) => {
    if (!value || Number.isNaN(value)) return '—'
    return `${value.toFixed(2)}s`
  }

  const httpCodeClass = (code: number) => {
    if (code >= 500) return 'http-server-error'
    if (code >= 400) return 'http-client-error'
    if (code >= 300) return 'http-redirect'
    if (code >= 200) return 'http-success'
    return 'http-info'
  }

  const durationColor = (value?: number) => {
    if (!value || Number.isNaN(value)) return 'neutral'
    if (value < 2) return 'fast'
    if (value < 5) return 'medium'
    return 'slow'
  }

  const formatNumber = (value?: number) => {
    if (value === undefined || value === null) return '—'
    return value.toLocaleString()
  }

  const formatCurrency = (value?: number) => {
    if (value === undefined || value === null || Number.isNaN(value)) {
      return '$0.0000'
    }
    if (value >= 1) {
      return `$${value.toFixed(2)}`
    }
    if (value >= 0.01) {
      return `$${value.toFixed(3)}`
    }
    return `$${value.toFixed(4)}`
  }

  const summaryDateLabel = computed(() => {
    const firstBucket = statsSeries.value.find((item) => item.day)
    const parsed = parseDateTime(firstBucket?.day ?? '')
    const date = parsed ?? startOfTodayLocal()
    return `${date.getFullYear()}-${padDatePart(date.getMonth() + 1)}-${padDatePart(date.getDate())}`
  })

  const statsCards = computed(() => {
    const data = stats.value
    const summaryDate = summaryDateLabel.value
    const totalTokens =
      (data?.input_tokens ?? 0) + (data?.output_tokens ?? 0) + (data?.reasoning_tokens ?? 0)

    return [
      {
        key: 'requests',
        label: t('components.logs.summary.total'),
        hint: t('components.logs.summary.requests'),
        value: data ? formatNumber(data.total_requests) : '—',
      },
      {
        key: 'tokens',
        label: t('components.logs.summary.tokens'),
        hint: t('components.logs.summary.tokenHint'),
        value: data ? formatNumber(totalTokens) : '—',
      },
      {
        key: 'cacheReads',
        label: t('components.logs.summary.cache'),
        hint: t('components.logs.summary.cacheHint'),
        value: data ? formatNumber(data.cache_read_tokens) : '—',
      },
      {
        key: 'cost',
        label: t('components.logs.tokenLabels.cost'),
        hint: summaryDate ? t('components.logs.summary.todayScope', { date: summaryDate }) : '',
        value: formatCurrency(data?.cost_total ?? 0),
      },
    ]
  })

  return {
    chartData,
    chartOptions,
    durationColor,
    formatDuration,
    formatNumber,
    formatStream,
    formatTime,
    httpCodeClass,
    statsCards,
  }
}
