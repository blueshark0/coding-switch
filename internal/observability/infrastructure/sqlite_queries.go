package infrastructure

import (
	"log"
	"time"

	observabilitydomain "codeswitch/internal/observability/domain"
	"codeswitch/internal/shared/storage"
	modelpricing "codeswitch/resources/model-pricing"
)

type SQLiteQueries struct {
	pricing *modelpricing.Service

	statsCache         *ttlCache[observabilitydomain.LogStats]
	providerDailyCache *ttlCache[[]observabilitydomain.ProviderDailyStat]
	heatmapCache       *ttlCache[[]observabilitydomain.HeatmapStat]
}

func NewSQLiteQueries() *SQLiteQueries {
	pricingPath, _ := storage.AppDataPath("model_prices_and_context_window.json")
	svc, err := modelpricing.NewServiceWithFile(pricingPath)
	if err != nil {
		log.Printf("pricing service init failed: %v", err)
	}
	return &SQLiteQueries{
		pricing:            svc,
		statsCache:         newTTLCache(30*time.Second, cloneLogStats),
		providerDailyCache: newTTLCache(5*time.Minute, cloneProviderDailyStats),
		heatmapCache:       newTTLCache(5*time.Minute, cloneHeatmapStats),
	}
}

func (q *SQLiteQueries) decorateCost(logEntry *observabilitydomain.RequestLog) {
	if q == nil || q.pricing == nil || logEntry == nil {
		return
	}
	usage := modelpricing.UsageSnapshot{
		InputTokens:       logEntry.InputTokens,
		OutputTokens:      logEntry.OutputTokens,
		CacheCreateTokens: logEntry.CacheCreateTokens,
		CacheReadTokens:   logEntry.CacheReadTokens,
		IsFast:            logEntry.IsFast,
	}
	cost := q.pricing.CalculateCost(logEntry.Model, usage)
	logEntry.HasPricing = cost.HasPricing
	logEntry.InputCost = cost.InputCost
	logEntry.OutputCost = cost.OutputCost
	logEntry.CacheCreateCost = cost.CacheCreateCost
	logEntry.CacheReadCost = cost.CacheReadCost
	logEntry.Ephemeral5mCost = cost.Ephemeral5mCost
	logEntry.Ephemeral1hCost = cost.Ephemeral1hCost
	logEntry.TotalCost = cost.TotalCost
}

func (q *SQLiteQueries) calculateCost(model string, usage modelpricing.UsageSnapshot) modelpricing.CostBreakdown {
	if q == nil || q.pricing == nil {
		return modelpricing.CostBreakdown{}
	}
	return q.pricing.CalculateCost(model, usage)
}

func cloneLogStats(stats observabilitydomain.LogStats) observabilitydomain.LogStats {
	cloned := stats
	if stats.Series == nil {
		cloned.Series = []observabilitydomain.LogStatsSeries{}
		return cloned
	}
	cloned.Series = append([]observabilitydomain.LogStatsSeries(nil), stats.Series...)
	return cloned
}

func cloneProviderDailyStats(stats []observabilitydomain.ProviderDailyStat) []observabilitydomain.ProviderDailyStat {
	if stats == nil {
		return []observabilitydomain.ProviderDailyStat{}
	}
	return append([]observabilitydomain.ProviderDailyStat(nil), stats...)
}

func cloneHeatmapStats(stats []observabilitydomain.HeatmapStat) []observabilitydomain.HeatmapStat {
	if stats == nil {
		return []observabilitydomain.HeatmapStat{}
	}
	return append([]observabilitydomain.HeatmapStat(nil), stats...)
}
