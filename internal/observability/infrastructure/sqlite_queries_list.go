package infrastructure

import (
	"strings"

	observabilitydomain "codeswitch/internal/observability/domain"

	"github.com/daodao97/xgo/xdb"
)

func (q *SQLiteQueries) ListRequestLogs(platform string, provider string, rangeKey string, costTier string, page int, pageSize int) (observabilitydomain.RequestLogPage, error) {
	page, pageSize = normalizeRequestLogPagination(page, pageSize)
	result := observabilitydomain.RequestLogPage{
		Items:    []observabilitydomain.RequestLog{},
		Page:     page,
		PageSize: pageSize,
	}
	rangeSpec := buildLogRangeSpec(rangeKey, timeNow())
	options := []xdb.Option{
		xdb.WhereGte("created_at", rangeSpec.startUTCString()),
		xdb.WhereLt("created_at", rangeSpec.endUTCString()),
	}
	if platform != "" {
		options = append(options, xdb.WhereEq("platform", platform))
	}
	if provider != "" {
		options = append(options, xdb.WhereEq("provider", provider))
	}

	normalizedCostTier := normalizeRequestLogCostTier(costTier)
	if normalizedCostTier == requestLogCostTierAll {
		total, records, err := requestLogModel().Page(page, pageSize, append(options, xdb.OrderByDesc("id"))...)
		if err != nil {
			if isNoSuchTableErr(err) {
				return result, nil
			}
			return result, err
		}
		result.Total = total
		result.Items = q.requestLogsFromRecords(records)
		return result, nil
	}

	records, err := requestLogModel().Selects(append(options, xdb.OrderByDesc("id"))...)
	if err != nil {
		if isNoSuchTableErr(err) {
			return result, nil
		}
		return result, err
	}

	filteredLogs := make([]observabilitydomain.RequestLog, 0, len(records))
	for _, logEntry := range q.requestLogsFromRecords(records) {
		if matchesRequestLogCostTier(logEntry, normalizedCostTier) {
			filteredLogs = append(filteredLogs, logEntry)
		}
	}
	result.Total = int64(len(filteredLogs))
	start := (page - 1) * pageSize
	if start >= len(filteredLogs) {
		return result, nil
	}
	end := min(start+pageSize, len(filteredLogs))
	result.Items = filteredLogs[start:end]
	return result, nil
}

func (q *SQLiteQueries) requestLogsFromRecords(records []xdb.Record) []observabilitydomain.RequestLog {
	logs := make([]observabilitydomain.RequestLog, 0, len(records))
	for _, record := range records {
		logEntry := observabilitydomain.RequestLog{
			ID:                record.GetInt64("id"),
			Platform:          record.GetString("platform"),
			Model:             record.GetString("model"),
			Provider:          record.GetString("provider"),
			HttpCode:          record.GetInt("http_code"),
			InputTokens:       record.GetInt("input_tokens"),
			OutputTokens:      record.GetInt("output_tokens"),
			CacheCreateTokens: record.GetInt("cache_create_tokens"),
			CacheReadTokens:   record.GetInt("cache_read_tokens"),
			ReasoningTokens:   record.GetInt("reasoning_tokens"),
			CreatedAt:         record.GetString("created_at"),
			IsStream:          record.GetBool("is_stream"),
			DurationSec:       record.GetFloat64("duration_sec"),
		}
		q.decorateCost(&logEntry)
		logs = append(logs, logEntry)
	}
	return logs
}

func (q *SQLiteQueries) ListProviders(platform string) ([]string, error) {
	options := []xdb.Option{
		xdb.Field("DISTINCT provider as provider"),
		xdb.WhereNotEq("provider", ""),
		xdb.OrderByAsc("provider"),
	}
	if platform != "" {
		options = append(options, xdb.WhereEq("platform", platform))
	}

	records, err := requestLogModel().Selects(options...)
	if err != nil {
		return nil, err
	}

	providers := make([]string, 0, len(records))
	for _, record := range records {
		name := strings.TrimSpace(record.GetString("provider"))
		if name != "" {
			providers = append(providers, name)
		}
	}
	return providers, nil
}
