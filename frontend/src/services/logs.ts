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

type RequestLogQuery = {
  platform?: string
  provider?: string
  limit?: number
}

export const fetchRequestLogs = async (query: RequestLogQuery = {}): Promise<RequestLog[]> => {
  const platform = query.platform ?? ''
  const provider = query.provider ?? ''
  const limit = query.limit ?? 100
  return ListRequestLogs(platform, provider, limit)
}

export const fetchLogProviders = async (platform = ''): Promise<string[]> => {
  return ListProviders(platform)
}

type LogStatsQuery = {
  platform?: string
  provider?: string
}

export const fetchLogStats = async (query: LogStatsQuery = {}): Promise<LogStats> => {
  const platform = query.platform ?? ''
  const provider = query.provider ?? ''
  return StatsSince(platform, provider)
}

export const fetchProviderDailyStats = async (platform = ''): Promise<ProviderDailyStat[]> => {
  return ProviderDailyStats(platform)
}

export const fetchHeatmapStats = async (days: number): Promise<HeatmapStat[]> => {
  const range = Number.isFinite(days) && days > 0 ? Math.floor(days) : 30
  return HeatmapStats(range)
}
