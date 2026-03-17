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
} from '../../bindings/codeswitch/internal/observability/domain/models'

export type { HeatmapStat, LogStats, LogStatsSeries, ProviderDailyStat, RequestLog }

export const LOG_RANGE_OPTIONS = ['today', 'last3days', 'last7days'] as const

export type LogRangeKey = (typeof LOG_RANGE_OPTIONS)[number]

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

type RequestLogQuery = {
  platform?: string
  provider?: string
  rangeKey?: LogRangeKey
  limit?: number
}

export const fetchRequestLogs = async (query: RequestLogQuery = {}): Promise<RequestLog[]> => {
  const platform = query.platform ?? ''
  const provider = query.provider ?? ''
  const rangeKey = normalizeLogRangeKey(query.rangeKey)
  const limit = query.limit ?? 100
  return ListRequestLogs(platform, provider, rangeKey, limit)
}

export const fetchLogProviders = async (platform = ''): Promise<string[]> => {
  return ListProviders(platform)
}

type LogStatsQuery = {
  platform?: string
  provider?: string
  rangeKey?: LogRangeKey
}

export const fetchLogStats = async (query: LogStatsQuery = {}): Promise<LogStats> => {
  const platform = query.platform ?? ''
  const provider = query.provider ?? ''
  const rangeKey = normalizeLogRangeKey(query.rangeKey)
  return StatsSince(platform, provider, rangeKey)
}

export const fetchProviderDailyStats = async (platform = ''): Promise<ProviderDailyStat[]> => {
  return ProviderDailyStats(platform)
}

export const fetchHeatmapStats = async (days: number): Promise<HeatmapStat[]> => {
  const range = Number.isFinite(days) && days > 0 ? Math.floor(days) : 30
  return HeatmapStats(range)
}
