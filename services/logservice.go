package services

import (
	"database/sql"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	modelpricing "codeswitch/resources/model-pricing"

	"github.com/daodao97/xgo/xdb"
)

const timeLayout = "2006-01-02 15:04:05"

type LogService struct {
	pricing *modelpricing.Service
}

func NewLogService() *LogService {
	svc, err := modelpricing.DefaultService()
	if err != nil {
		log.Printf("pricing service init failed: %v", err)
	}
	return &LogService{pricing: svc}
}

func (ls *LogService) ListRequestLogs(platform string, provider string, limit int) ([]ReqeustLog, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	model := xdb.New("request_log")
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
	records, err := model.Selects(options...)
	if err != nil {
		return nil, err
	}
	logs := make([]ReqeustLog, 0, len(records))
	for _, record := range records {
		logEntry := ReqeustLog{
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
		ls.decorateCost(&logEntry)
		logs = append(logs, logEntry)
	}
	return logs, nil
}

func (ls *LogService) ListProviders(platform string) ([]string, error) {
	model := xdb.New("request_log")
	options := []xdb.Option{
		xdb.Field("DISTINCT provider as provider"),
		xdb.WhereNotEq("provider", ""),
		xdb.OrderByAsc("provider"),
	}
	if platform != "" {
		options = append(options, xdb.WhereEq("platform", platform))
	}
	records, err := model.Selects(options...)
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

func (ls *LogService) HeatmapStats(days int) ([]HeatmapStat, error) {
	if days <= 0 {
		days = 30
	}
	totalHours := days * 24
	if totalHours <= 0 {
		totalHours = 24
	}
	rangeStart := startOfHour(time.Now())
	if totalHours > 1 {
		rangeStart = rangeStart.Add(-time.Duration(totalHours-1) * time.Hour)
	}
	db, err := xdb.DB("default")
	if err != nil {
		return nil, err
	}
	query := `SELECT strftime('%Y-%m-%d %H:00:00', created_at, 'localtime') AS bucket,
		model,
		COUNT(*) AS total_requests,
		SUM(input_tokens) AS input_tokens,
		SUM(output_tokens) AS output_tokens,
		SUM(reasoning_tokens) AS reasoning_tokens,
		SUM(cache_create_tokens) AS cache_create_tokens,
		SUM(cache_read_tokens) AS cache_read_tokens
		FROM request_log
		WHERE created_at >= ?
		GROUP BY bucket, model
		ORDER BY bucket DESC`
	rows, err := db.Query(query, rangeStart.Format(timeLayout))
	if err != nil {
		if isNoSuchTableErr(err) {
			return []HeatmapStat{}, nil
		}
		return nil, err
	}
	defer rows.Close()
	hourBuckets := map[int64]*HeatmapStat{}
	for rows.Next() {
		var bucketStr sql.NullString
		var modelName sql.NullString
		var totalRequests sql.NullInt64
		var input, output, reasoning, cacheCreate, cacheRead sql.NullInt64
		if err := rows.Scan(&bucketStr, &modelName, &totalRequests, &input, &output, &reasoning, &cacheCreate, &cacheRead); err != nil {
			return nil, err
		}
		if !bucketStr.Valid {
			continue
		}
		bucketTime, err := time.ParseInLocation(timeLayout, bucketStr.String, time.Local)
		if err != nil {
			continue
		}
		hourKey := bucketTime.Unix()
		bucket := hourBuckets[hourKey]
		if bucket == nil {
			bucket = &HeatmapStat{Day: bucketTime.Format("01-02 15")}
			hourBuckets[hourKey] = bucket
		}
		bucket.TotalRequests += nullInt64(totalRequests)
		bucket.InputTokens += nullInt64(input)
		bucket.OutputTokens += nullInt64(output)
		bucket.ReasoningTokens += nullInt64(reasoning)
		usage := modelpricing.UsageSnapshot{
			InputTokens:       int(nullInt64(input)),
			OutputTokens:      int(nullInt64(output)),
			CacheCreateTokens: int(nullInt64(cacheCreate)),
			CacheReadTokens:   int(nullInt64(cacheRead)),
		}
		cost := ls.calculateCost(modelName.String, usage)
		bucket.TotalCost += cost.TotalCost
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(hourBuckets) == 0 {
		return []HeatmapStat{}, nil
	}
	hourKeys := make([]int64, 0, len(hourBuckets))
	for key := range hourBuckets {
		hourKeys = append(hourKeys, key)
	}
	sort.Slice(hourKeys, func(i, j int) bool {
		return hourKeys[i] > hourKeys[j]
	})
	stats := make([]HeatmapStat, 0, min(len(hourKeys), totalHours))
	for _, key := range hourKeys {
		stats = append(stats, *hourBuckets[key])
		if len(stats) >= totalHours {
			break
		}
	}
	return stats, nil
}

func (ls *LogService) StatsSince(platform string) (LogStats, error) {
	const seriesHours = 24

	stats := LogStats{
		Series: make([]LogStatsSeries, 0, seriesHours),
	}
	now := time.Now()
	seriesStart := startOfDay(now)
	seriesEnd := seriesStart.Add(seriesHours * time.Hour)
	queryStart := seriesStart.Add(-24 * time.Hour)
	db, err := xdb.DB("default")
	if err != nil {
		return stats, err
	}
	args := []any{queryStart.Format(timeLayout)}
	platformFilter := ""
	if platform != "" {
		platformFilter = " AND platform = ?"
		args = append(args, platform)
	}
	query := fmt.Sprintf(`SELECT strftime('%%Y-%%m-%%d %%H:00:00', created_at, 'localtime') AS bucket,
		model,
		COUNT(*) AS total_requests,
		SUM(input_tokens) AS input_tokens,
		SUM(output_tokens) AS output_tokens,
		SUM(reasoning_tokens) AS reasoning_tokens,
		SUM(cache_create_tokens) AS cache_create_tokens,
		SUM(cache_read_tokens) AS cache_read_tokens
		FROM request_log
		WHERE created_at >= ?%s
		GROUP BY bucket, model
		ORDER BY bucket ASC`, platformFilter)
	rows, err := db.Query(query, args...)
	if err != nil {
		if isNoSuchTableErr(err) {
			return stats, nil
		}
		return stats, err
	}
	defer rows.Close()
	seriesBuckets := make([]*LogStatsSeries, seriesHours)
	for i := 0; i < seriesHours; i++ {
		bucketTime := seriesStart.Add(time.Duration(i) * time.Hour)
		seriesBuckets[i] = &LogStatsSeries{Day: bucketTime.Format(timeLayout)}
	}
	for rows.Next() {
		var bucketStr sql.NullString
		var modelName sql.NullString
		var totalRequests sql.NullInt64
		var input, output, reasoning, cacheCreate, cacheRead sql.NullInt64
		if err := rows.Scan(&bucketStr, &modelName, &totalRequests, &input, &output, &reasoning, &cacheCreate, &cacheRead); err != nil {
			return stats, err
		}
		if !bucketStr.Valid {
			continue
		}
		bucketTime, err := time.ParseInLocation(timeLayout, bucketStr.String, time.Local)
		if err != nil {
			continue
		}
		if bucketTime.Before(seriesStart) || !bucketTime.Before(seriesEnd) {
			continue
		}
		bucketIndex := int(bucketTime.Sub(seriesStart) / time.Hour)
		if bucketIndex < 0 || bucketIndex >= seriesHours {
			continue
		}
		bucket := seriesBuckets[bucketIndex]
		bucket.TotalRequests += nullInt64(totalRequests)
		bucket.InputTokens += nullInt64(input)
		bucket.OutputTokens += nullInt64(output)
		bucket.ReasoningTokens += nullInt64(reasoning)
		bucket.CacheCreateTokens += nullInt64(cacheCreate)
		bucket.CacheReadTokens += nullInt64(cacheRead)
		usage := modelpricing.UsageSnapshot{
			InputTokens:       int(nullInt64(input)),
			OutputTokens:      int(nullInt64(output)),
			CacheCreateTokens: int(nullInt64(cacheCreate)),
			CacheReadTokens:   int(nullInt64(cacheRead)),
		}
		cost := ls.calculateCost(modelName.String, usage)
		bucket.TotalCost += cost.TotalCost
		stats.TotalRequests += nullInt64(totalRequests)
		stats.InputTokens += nullInt64(input)
		stats.OutputTokens += nullInt64(output)
		stats.ReasoningTokens += nullInt64(reasoning)
		stats.CacheCreateTokens += nullInt64(cacheCreate)
		stats.CacheReadTokens += nullInt64(cacheRead)
		stats.CostInput += cost.InputCost
		stats.CostOutput += cost.OutputCost
		stats.CostCacheCreate += cost.CacheCreateCost
		stats.CostCacheRead += cost.CacheReadCost
		stats.CostTotal += cost.TotalCost
	}
	if err := rows.Err(); err != nil {
		return stats, err
	}
	for i := 0; i < seriesHours; i++ {
		stats.Series = append(stats.Series, *seriesBuckets[i])
	}
	return stats, nil
}

func (ls *LogService) ProviderDailyStats(platform string) ([]ProviderDailyStat, error) {
	start := startOfDay(time.Now())
	end := start.Add(24 * time.Hour)
	db, err := xdb.DB("default")
	if err != nil {
		return nil, err
	}
	args := []any{start.Format(timeLayout), end.Format(timeLayout)}
	platformFilter := ""
	if platform != "" {
		platformFilter = " AND platform = ?"
		args = append(args, platform)
	}
	query := fmt.Sprintf(`SELECT COALESCE(NULLIF(TRIM(provider), ''), '(unknown)') AS provider,
		model,
		COUNT(*) AS total_requests,
		SUM(CASE WHEN http_code BETWEEN 200 AND 299 THEN 1 ELSE 0 END) AS successful_requests,
		SUM(input_tokens) AS input_tokens,
		SUM(output_tokens) AS output_tokens,
		SUM(reasoning_tokens) AS reasoning_tokens,
		SUM(cache_create_tokens) AS cache_create_tokens,
		SUM(cache_read_tokens) AS cache_read_tokens
		FROM request_log
		WHERE created_at >= ? AND created_at < ?%s
		GROUP BY provider, model`, platformFilter)
	rows, err := db.Query(query, args...)
	if err != nil {
		if isNoSuchTableErr(err) {
			return []ProviderDailyStat{}, nil
		}
		return nil, err
	}
	defer rows.Close()
	statMap := map[string]*ProviderDailyStat{}
	for rows.Next() {
		var provider string
		var modelName sql.NullString
		var totalRequests sql.NullInt64
		var successfulRequests sql.NullInt64
		var input, output, reasoning, cacheCreate, cacheRead sql.NullInt64
		if err := rows.Scan(&provider, &modelName, &totalRequests, &successfulRequests, &input, &output, &reasoning, &cacheCreate, &cacheRead); err != nil {
			return nil, err
		}
		stat := statMap[provider]
		if stat == nil {
			stat = &ProviderDailyStat{Provider: provider}
			statMap[provider] = stat
		}
		tr := nullInt64(totalRequests)
		succ := nullInt64(successfulRequests)
		stat.TotalRequests += tr
		stat.SuccessfulRequests += succ
		stat.FailedRequests += tr - succ
		stat.InputTokens += nullInt64(input)
		stat.OutputTokens += nullInt64(output)
		stat.ReasoningTokens += nullInt64(reasoning)
		stat.CacheCreateTokens += nullInt64(cacheCreate)
		stat.CacheReadTokens += nullInt64(cacheRead)
		usage := modelpricing.UsageSnapshot{
			InputTokens:       int(nullInt64(input)),
			OutputTokens:      int(nullInt64(output)),
			CacheCreateTokens: int(nullInt64(cacheCreate)),
			CacheReadTokens:   int(nullInt64(cacheRead)),
		}
		cost := ls.calculateCost(modelName.String, usage)
		stat.CostTotal += cost.TotalCost
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	stats := make([]ProviderDailyStat, 0, len(statMap))
	for _, stat := range statMap {
		if stat.TotalRequests > 0 {
			stat.SuccessRate = float64(stat.SuccessfulRequests) / float64(stat.TotalRequests)
		}
		stats = append(stats, *stat)
	}
	sort.Slice(stats, func(i, j int) bool {
		if stats[i].TotalRequests == stats[j].TotalRequests {
			return stats[i].Provider < stats[j].Provider
		}
		return stats[i].TotalRequests > stats[j].TotalRequests
	})
	return stats, nil
}

func (ls *LogService) decorateCost(logEntry *ReqeustLog) {
	if ls == nil || ls.pricing == nil || logEntry == nil {
		return
	}
	usage := modelpricing.UsageSnapshot{
		InputTokens:       logEntry.InputTokens,
		OutputTokens:      logEntry.OutputTokens,
		CacheCreateTokens: logEntry.CacheCreateTokens,
		CacheReadTokens:   logEntry.CacheReadTokens,
	}
	cost := ls.pricing.CalculateCost(logEntry.Model, usage)
	logEntry.HasPricing = cost.HasPricing
	logEntry.InputCost = cost.InputCost
	logEntry.OutputCost = cost.OutputCost
	logEntry.CacheCreateCost = cost.CacheCreateCost
	logEntry.CacheReadCost = cost.CacheReadCost
	logEntry.Ephemeral5mCost = cost.Ephemeral5mCost
	logEntry.Ephemeral1hCost = cost.Ephemeral1hCost
	logEntry.TotalCost = cost.TotalCost
}

func (ls *LogService) calculateCost(model string, usage modelpricing.UsageSnapshot) modelpricing.CostBreakdown {
	if ls == nil || ls.pricing == nil {
		return modelpricing.CostBreakdown{}
	}
	return ls.pricing.CalculateCost(model, usage)
}

func parseCreatedAt(record xdb.Record) (time.Time, bool) {
	if t := record.GetTime("created_at"); t != nil {
		return t.In(time.Local), true
	}
	raw := strings.TrimSpace(record.GetString("created_at"))
	if raw == "" {
		return time.Time{}, false
	}

	layouts := []string{
		timeLayout,
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05 -0700",
		"2006-01-02 15:04:05 -0700 MST",
		"2006-01-02 15:04:05 MST",
		"2006-01-02T15:04:05-0700",
	}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, raw); err == nil {
			return parsed.In(time.Local), true
		}
		if parsed, err := time.ParseInLocation(layout, raw, time.Local); err == nil {
			return parsed.In(time.Local), true
		}
	}

	if normalized := strings.Replace(raw, " ", "T", 1); normalized != raw {
		if parsed, err := time.Parse(time.RFC3339, normalized); err == nil {
			return parsed.In(time.Local), true
		}
	}

	if len(raw) >= len("2006-01-02") {
		if parsed, err := time.ParseInLocation("2006-01-02", raw[:10], time.Local); err == nil {
			return parsed, false
		}
	}

	return time.Time{}, false
}

func dayFromTimestamp(value string) string {
	if len(value) >= len("2006-01-02") {
		if t, err := time.ParseInLocation(timeLayout, value, time.Local); err == nil {
			return t.Format("2006-01-02")
		}
		return value[:10]
	}
	return value
}

func startOfDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

func startOfHour(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, t.Hour(), 0, 0, 0, t.Location())
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func isNoSuchTableErr(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "no such table")
}

func nullInt64(v sql.NullInt64) int64 {
	if v.Valid {
		return v.Int64
	}
	return 0
}

type HeatmapStat struct {
	Day             string  `json:"day"`
	TotalRequests   int64   `json:"total_requests"`
	InputTokens     int64   `json:"input_tokens"`
	OutputTokens    int64   `json:"output_tokens"`
	ReasoningTokens int64   `json:"reasoning_tokens"`
	TotalCost       float64 `json:"total_cost"`
}

type LogStats struct {
	TotalRequests     int64            `json:"total_requests"`
	InputTokens       int64            `json:"input_tokens"`
	OutputTokens      int64            `json:"output_tokens"`
	ReasoningTokens   int64            `json:"reasoning_tokens"`
	CacheCreateTokens int64            `json:"cache_create_tokens"`
	CacheReadTokens   int64            `json:"cache_read_tokens"`
	CostTotal         float64          `json:"cost_total"`
	CostInput         float64          `json:"cost_input"`
	CostOutput        float64          `json:"cost_output"`
	CostCacheCreate   float64          `json:"cost_cache_create"`
	CostCacheRead     float64          `json:"cost_cache_read"`
	Series            []LogStatsSeries `json:"series"`
}

type ProviderDailyStat struct {
	Provider           string  `json:"provider"`
	TotalRequests      int64   `json:"total_requests"`
	SuccessfulRequests int64   `json:"successful_requests"`
	FailedRequests     int64   `json:"failed_requests"`
	SuccessRate        float64 `json:"success_rate"`
	InputTokens        int64   `json:"input_tokens"`
	OutputTokens       int64   `json:"output_tokens"`
	ReasoningTokens    int64   `json:"reasoning_tokens"`
	CacheCreateTokens  int64   `json:"cache_create_tokens"`
	CacheReadTokens    int64   `json:"cache_read_tokens"`
	CostTotal          float64 `json:"cost_total"`
}

type LogStatsSeries struct {
	Day               string  `json:"day"`
	TotalRequests     int64   `json:"total_requests"`
	InputTokens       int64   `json:"input_tokens"`
	OutputTokens      int64   `json:"output_tokens"`
	ReasoningTokens   int64   `json:"reasoning_tokens"`
	CacheCreateTokens int64   `json:"cache_create_tokens"`
	CacheReadTokens   int64   `json:"cache_read_tokens"`
	TotalCost         float64 `json:"total_cost"`
}
