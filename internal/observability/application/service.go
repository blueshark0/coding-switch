package application

import observabilitydomain "codeswitch/internal/observability/domain"

type Queries interface {
	ListRequestLogs(platform string, provider string, rangeKey string, costTier string, page int, pageSize int) (observabilitydomain.RequestLogPage, error)
	ListProviders(platform string) ([]string, error)
	StatsSince(platform string, provider string, rangeKey string, costTier string) (observabilitydomain.LogStats, error)
	ProviderDailyStats(platform string) ([]observabilitydomain.ProviderDailyStat, error)
	HeatmapStats(days int) ([]observabilitydomain.HeatmapStat, error)
}

type Service struct {
	queries Queries
}

func NewService(queries Queries) *Service {
	return &Service{queries: queries}
}

func (s *Service) ListRequestLogs(platform string, provider string, rangeKey string, costTier string, page int, pageSize int) (observabilitydomain.RequestLogPage, error) {
	return s.queries.ListRequestLogs(platform, provider, rangeKey, costTier, page, pageSize)
}

func (s *Service) ListProviders(platform string) ([]string, error) {
	return s.queries.ListProviders(platform)
}

func (s *Service) StatsSince(platform string, provider string, rangeKey string, costTier string) (observabilitydomain.LogStats, error) {
	return s.queries.StatsSince(platform, provider, rangeKey, costTier)
}

func (s *Service) ProviderDailyStats(platform string) ([]observabilitydomain.ProviderDailyStat, error) {
	return s.queries.ProviderDailyStats(platform)
}

func (s *Service) HeatmapStats(days int) ([]observabilitydomain.HeatmapStat, error) {
	return s.queries.HeatmapStats(days)
}
