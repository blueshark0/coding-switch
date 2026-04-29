package infrastructure

import (
	"strings"

	observabilitydomain "codeswitch/internal/observability/domain"
)

const (
	requestLogCostTierAll    = "all"
	requestLogCostTierLow    = "low"
	requestLogCostTierMedium = "medium"
	requestLogCostTierHigh   = "high"

	defaultRequestLogPageSize = 15
)

func normalizeRequestLogPagination(page int, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	switch pageSize {
	case 15, 30, 50, 100:
	default:
		pageSize = defaultRequestLogPageSize
	}
	return page, pageSize
}

func normalizeRequestLogCostTier(costTier string) string {
	switch strings.ToLower(strings.TrimSpace(costTier)) {
	case requestLogCostTierLow:
		return requestLogCostTierLow
	case requestLogCostTierMedium:
		return requestLogCostTierMedium
	case requestLogCostTierHigh:
		return requestLogCostTierHigh
	default:
		return requestLogCostTierAll
	}
}

func matchesRequestLogCostTier(logEntry observabilitydomain.RequestLog, costTier string) bool {
	switch normalizeRequestLogCostTier(costTier) {
	case requestLogCostTierLow:
		return logEntry.HasPricing && logEntry.TotalCost < 0.1
	case requestLogCostTierMedium:
		return logEntry.HasPricing && logEntry.TotalCost >= 0.1 && logEntry.TotalCost <= 0.5
	case requestLogCostTierHigh:
		return logEntry.HasPricing && logEntry.TotalCost > 0.5
	default:
		return true
	}
}
