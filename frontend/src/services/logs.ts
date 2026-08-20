import {
  HeatmapStats,
  ListProviders,
  ListRequestLogs,
  ProviderDailyStats,
  StatsSince,
} from '../../bindings/codeswitch/internal/interfaces/wails/observabilityfacade'
import type {
  HeatmapStat,
  LogStats,
  LogStatsSeries,
  ProviderDailyStat,
  RequestLog,
  RequestLogPage,
} from '../../bindings/codeswitch/internal/observability/domain/models'

export type {
  HeatmapStat,
  LogStats,
  LogStatsSeries,
  ProviderDailyStat,
  RequestLog,
  RequestLogPage,
}

export const LOG_RANGE_OPTIONS = ['today', 'last3days', 'last7days'] as const
export const LOG_COST_TIER_OPTIONS = ['all', 'low', 'medium', 'high'] as const
export const LOG_PAGE_SIZE_OPTIONS = [15, 30, 50, 100] as const
export const DEFAULT_LOG_PAGE_SIZE = 15

export type LogRangeKey = (typeof LOG_RANGE_OPTIONS)[number]
export type LogCostTier = (typeof LOG_COST_TIER_OPTIONS)[number]
export type LogPageSize = (typeof LOG_PAGE_SIZE_OPTIONS)[number]

export const DEFAULT_LOG_RANGE: LogRangeKey = 'today'

export const normalizeLogRangeKey = (value?: string): LogRangeKey => {
  switch (value) {
    case 'last3days':
      return 'last3days'
    case 'last7days':
      return 'last7days'
    default:
      return DEFAULT_LOG_RANGE
  }
}

export const normalizeLogCostTier = (value?: string): LogCostTier => {
  switch (value) {
    case 'low':
      return 'low'
    case 'medium':
      return 'medium'
    case 'high':
      return 'high'
    default:
      return 'all'
  }
}

export const normalizeLogPageSize = (value?: number): LogPageSize => {
  return LOG_PAGE_SIZE_OPTIONS.includes(value as LogPageSize)
    ? (value as LogPageSize)
    : DEFAULT_LOG_PAGE_SIZE
}

type RequestLogQuery = {
  platform?: string
  provider?: string
  rangeKey?: LogRangeKey
  costTier?: LogCostTier
  page?: number
  pageSize?: number
}

export const fetchRequestLogs = async (query: RequestLogQuery = {}): Promise<RequestLogPage> => {
  const platform = query.platform ?? ''
  const provider = query.provider ?? ''
  const rangeKey = normalizeLogRangeKey(query.rangeKey)
  const costTier = normalizeLogCostTier(query.costTier)
  const page = Math.max(1, Math.floor(query.page ?? 1))
  const pageSize = normalizeLogPageSize(query.pageSize)
  return ListRequestLogs(platform, provider, rangeKey, costTier, page, pageSize)
}

export const fetchLogProviders = async (platform = ''): Promise<string[]> => {
  return ListProviders(platform)
}

type LogStatsQuery = {
  platform?: string
  provider?: string
  rangeKey?: LogRangeKey
  costTier?: LogCostTier
}

export const fetchLogStats = async (query: LogStatsQuery = {}): Promise<LogStats> => {
  const platform = query.platform ?? ''
  const provider = query.provider ?? ''
  const rangeKey = normalizeLogRangeKey(query.rangeKey)
  const costTier = normalizeLogCostTier(query.costTier)
  return StatsSince(platform, provider, rangeKey, costTier)
}

export const fetchProviderDailyStats = async (platform = ''): Promise<ProviderDailyStat[]> => {
  return ProviderDailyStats(platform)
}

export const fetchHeatmapStats = async (days: number): Promise<HeatmapStat[]> => {
  const range = Number.isFinite(days) && days > 0 ? Math.floor(days) : 30
  return HeatmapStats(range)
}
