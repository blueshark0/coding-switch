package infrastructure

import (
	"strings"

	observabilitydomain "codeswitch/internal/observability/domain"
	"codeswitch/internal/shared/storage"

	"github.com/daodao97/xgo/xdb"
)

// SessionUsageByPlatform 按会话聚合 request_log，供会话页把不可读的会话
// ID 换成请求数/tokens/费用/模型等可读摘要。只统计带了 session_id 的行
//（session_id 列是后补的，历史行为空，自然被排除）。
func (q *SQLiteQueries) SessionUsageByPlatform(platform string) (map[string]observabilitydomain.SessionUsage, error) {
	result := make(map[string]observabilitydomain.SessionUsage)
	if q == nil || strings.TrimSpace(platform) == "" {
		return result, nil
	}
	db, err := xdb.DB(storage.RequestLogDBName)
	if err != nil {
		return nil, err
	}
	rows, err := db.Query(`
		SELECT session_id,
			COUNT(*) AS requests,
			COALESCE(SUM(input_tokens), 0) AS input_tokens,
			COALESCE(SUM(output_tokens), 0) AS output_tokens,
			COALESCE(SUM(total_cost), 0) AS total_cost,
			GROUP_CONCAT(DISTINCT model) AS models
		FROM request_log
		WHERE platform = ? AND session_id IS NOT NULL AND session_id != ''
		GROUP BY session_id`, platform)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var usage observabilitydomain.SessionUsage
		var models string
		if err := rows.Scan(
			&usage.SessionID,
			&usage.Requests,
			&usage.InputTokens,
			&usage.OutputTokens,
			&usage.TotalCost,
			&models,
		); err != nil {
			return nil, err
		}
		usage.Models = splitDistinctModels(models)
		result[usage.SessionID] = usage
	}
	return result, rows.Err()
}

func splitDistinctModels(models string) []string {
	parts := strings.Split(models, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
