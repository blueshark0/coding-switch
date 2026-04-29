package wails

import (
	observabilityapp "codeswitch/internal/observability/application"
	observabilitydomain "codeswitch/internal/observability/domain"
)

type ObservabilityFacade struct {
	service *observabilityapp.Service
}

func NewObservabilityFacade(service *observabilityapp.Service) *ObservabilityFacade {
	return &ObservabilityFacade{service: service}
}

func (f *ObservabilityFacade) ListRequestLogs(platform string, provider string, rangeKey string, costTier string, page int, pageSize int) (observabilitydomain.RequestLogPage, error) {
	return f.service.ListRequestLogs(platform, provider, rangeKey, costTier, page, pageSize)
}

func (f *ObservabilityFacade) ListProviders(platform string) ([]string, error) {
	return f.service.ListProviders(platform)
}

func (f *ObservabilityFacade) StatsSince(platform string, provider string, rangeKey string, costTier string) (observabilitydomain.LogStats, error) {
	return f.service.StatsSince(platform, provider, rangeKey, costTier)
}

func (f *ObservabilityFacade) ProviderDailyStats(platform string) ([]observabilitydomain.ProviderDailyStat, error) {
	return f.service.ProviderDailyStats(platform)
}

func (f *ObservabilityFacade) HeatmapStats(days int) ([]observabilitydomain.HeatmapStat, error) {
	return f.service.HeatmapStats(days)
}
