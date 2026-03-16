package infrastructure

import (
	"log"

	observabilitydomain "codeswitch/internal/observability/domain"
	modelpricing "codeswitch/resources/model-pricing"
)

type SQLiteQueries struct {
	pricing *modelpricing.Service
}

func NewSQLiteQueries() *SQLiteQueries {
	svc, err := modelpricing.DefaultService()
	if err != nil {
		log.Printf("pricing service init failed: %v", err)
	}
	return &SQLiteQueries{pricing: svc}
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
