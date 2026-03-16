package infrastructure

import (
	"strings"

	observabilitydomain "codeswitch/internal/observability/domain"

	"github.com/daodao97/xgo/xdb"
)

func (q *SQLiteQueries) ListRequestLogs(platform string, provider string, limit int) ([]observabilitydomain.RequestLog, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	options := []xdb.Option{
		xdb.OrderByDesc("id"),
		xdb.Limit(limit),
	}
	if platform != "" {
		options = append(options, xdb.WhereEq("platform", platform))
	}
	if provider != "" {
		options = append(options, xdb.WhereEq("provider", provider))
	}

	records, err := requestLogModel().Selects(options...)
	if err != nil {
		return nil, err
	}

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
	return logs, nil
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
