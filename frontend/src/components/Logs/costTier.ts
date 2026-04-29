import type { LogCostTier, RequestLog } from '../../services/logs'

export const COST_TIER_OPTIONS = ['all', 'low', 'medium', 'high'] as const

export type CostTier = LogCostTier
type PricedCostTier = Exclude<CostTier, 'all'>

const classifyPricedCostTier = (
  log: Pick<RequestLog, 'has_pricing' | 'total_cost'>,
): PricedCostTier | '' => {
  if (!log.has_pricing) return ''
  if (log.total_cost < 0.1) return 'low'
  if (log.total_cost > 0.5) return 'high'
  return 'medium'
}

export const matchesCostTier = (
  log: Pick<RequestLog, 'has_pricing' | 'total_cost'>,
  selectedTier: CostTier,
): boolean => {
  if (selectedTier === 'all') return true
  return classifyPricedCostTier(log) === selectedTier
}

export const getCostValueClass = (
  log: Pick<RequestLog, 'has_pricing' | 'total_cost'>,
): string => {
  const tier = classifyPricedCostTier(log)
  if (tier === 'low') return 'token-cost-low'
  if (tier === 'high') return 'token-cost-high'
  if (tier === 'medium') return 'token-cost-medium'
  return ''
}
