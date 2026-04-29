package infrastructure

import (
	"database/sql"
	"fmt"
	"sort"
	"time"

	observabilitydomain "codeswitch/internal/observability/domain"
	"codeswitch/internal/shared/storage"
	modelpricing "codeswitch/resources/model-pricing"

	"github.com/daodao97/xgo/xdb"
)

func (q *SQLiteQueries) HeatmapStats(days int) ([]observabilitydomain.HeatmapStat, error) {
	if days <= 0 {
		days = 30
	}
	cacheKey := fmt.Sprintf("days:%d", days)
	if cached, ok := q.heatmapCache.Get(cacheKey); ok {
		return cached, nil
	}
	totalHours := days * 24
	if totalHours <= 0 {
		totalHours = 24
	}

	rangeStart := startOfHour(time.Now())
	if totalHours > 1 {
		rangeStart = rangeStart.Add(-time.Duration(totalHours-1) * time.Hour)
	}

	db, err := xdb.DB(storage.RequestLogDBName)
	if err != nil {
		return nil, err
	}
	rows, err := db.Query(`SELECT strftime('%Y-%m-%d %H:00:00', created_at, 'localtime') AS bucket,
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
		ORDER BY bucket DESC`, rangeStart.Format(timeLayout))
	if err != nil {
		if isNoSuchTableErr(err) {
			return []observabilitydomain.HeatmapStat{}, nil
		}
		return nil, err
	}
	defer rows.Close()

	hourBuckets := map[int64]*observabilitydomain.HeatmapStat{}
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
			bucket = &observabilitydomain.HeatmapStat{Day: bucketTime.Format("01-02 15")}
			hourBuckets[hourKey] = bucket
		}
		bucket.TotalRequests += nullInt64(totalRequests)
		bucket.InputTokens += nullInt64(input)
		bucket.OutputTokens += nullInt64(output)
		bucket.ReasoningTokens += nullInt64(reasoning)
		cost := q.calculateCost(modelName.String, modelpricing.UsageSnapshot{
			InputTokens:       int(nullInt64(input)),
			OutputTokens:      int(nullInt64(output)),
			CacheCreateTokens: int(nullInt64(cacheCreate)),
			CacheReadTokens:   int(nullInt64(cacheRead)),
		})
		bucket.TotalCost += cost.TotalCost
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(hourBuckets) == 0 {
		return q.heatmapCache.Set(cacheKey, []observabilitydomain.HeatmapStat{}), nil
	}

	hourKeys := make([]int64, 0, len(hourBuckets))
	for key := range hourBuckets {
		hourKeys = append(hourKeys, key)
	}
	sort.Slice(hourKeys, func(i, j int) bool {
		return hourKeys[i] > hourKeys[j]
	})

	stats := make([]observabilitydomain.HeatmapStat, 0, min(len(hourKeys), totalHours))
	for _, key := range hourKeys {
		stats = append(stats, *hourBuckets[key])
		if len(stats) >= totalHours {
			break
		}
	}
	return q.heatmapCache.Set(cacheKey, stats), nil
}

func (q *SQLiteQueries) StatsSince(platform string, provider string, rangeKey string, costTier string) (observabilitydomain.LogStats, error) {
	rangeSpec := buildLogRangeSpec(rangeKey, timeNow())
	normalizedCostTier := normalizeRequestLogCostTier(costTier)
	cacheKey := fmt.Sprintf("platform:%s|provider:%s|range:%s|cost:%s", platform, provider, rangeSpec.key, normalizedCostTier)
	if cached, ok := q.statsCache.Get(cacheKey); ok {
		return cached, nil
	}
	stats := observabilitydomain.LogStats{
		Series: make([]observabilitydomain.LogStatsSeries, 0, rangeSpec.seriesCount),
	}
	seriesBuckets, bucketIndexes := newStatsSeriesBuckets(rangeSpec)

	if normalizedCostTier != requestLogCostTierAll {
		return q.statsSinceCostTierFiltered(cacheKey, stats, rangeSpec, seriesBuckets, bucketIndexes, platform, provider, normalizedCostTier)
	}

	db, err := xdb.DB(storage.RequestLogDBName)
	if err != nil {
		return stats, err
	}

	args := []any{rangeSpec.startUTCString(), rangeSpec.endUTCString()}
	filterClause := ""
	if platform != "" {
		filterClause += " AND platform = ?"
		args = append(args, platform)
	}
	if provider != "" {
		filterClause += " AND provider = ?"
		args = append(args, provider)
	}

	rows, err := db.Query(fmt.Sprintf(`SELECT strftime('%s', created_at, 'localtime') AS bucket,
			model,
			COUNT(*) AS total_requests,
			SUM(input_tokens) AS input_tokens,
		SUM(output_tokens) AS output_tokens,
			SUM(reasoning_tokens) AS reasoning_tokens,
			SUM(cache_create_tokens) AS cache_create_tokens,
			SUM(cache_read_tokens) AS cache_read_tokens
			FROM request_log
			WHERE created_at >= ? AND created_at < ?%s
			GROUP BY bucket, model
			ORDER BY bucket ASC`, rangeSpec.bucketSQLFormat(), filterClause), args...)
	if err != nil {
		if isNoSuchTableErr(err) {
			appendStatsSeries(&stats, seriesBuckets)
			return stats, nil
		}
		return stats, err
	}
	defer rows.Close()

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
		bucketIndex, ok := bucketIndexes[bucketTime.Format(timeLayout)]
		if !ok {
			continue
		}

		bucket := seriesBuckets[bucketIndex]
		bucket.TotalRequests += nullInt64(totalRequests)
		bucket.InputTokens += nullInt64(input)
		bucket.OutputTokens += nullInt64(output)
		bucket.ReasoningTokens += nullInt64(reasoning)
		bucket.CacheCreateTokens += nullInt64(cacheCreate)
		bucket.CacheReadTokens += nullInt64(cacheRead)

		cost := q.calculateCost(modelName.String, modelpricing.UsageSnapshot{
			InputTokens:       int(nullInt64(input)),
			OutputTokens:      int(nullInt64(output)),
			CacheCreateTokens: int(nullInt64(cacheCreate)),
			CacheReadTokens:   int(nullInt64(cacheRead)),
		})
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
	appendStatsSeries(&stats, seriesBuckets)
	return q.statsCache.Set(cacheKey, stats), nil
}

func (q *SQLiteQueries) statsSinceCostTierFiltered(
	cacheKey string,
	stats observabilitydomain.LogStats,
	rangeSpec logRangeSpec,
	seriesBuckets []*observabilitydomain.LogStatsSeries,
	bucketIndexes map[string]int,
	platform string,
	provider string,
	costTier string,
) (observabilitydomain.LogStats, error) {
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

	records, err := requestLogModel().Selects(options...)
	if err != nil {
		if isNoSuchTableErr(err) {
			appendStatsSeries(&stats, seriesBuckets)
			return stats, nil
		}
		return stats, err
	}

	for _, logEntry := range q.requestLogsFromRecords(records) {
		if !matchesRequestLogCostTier(logEntry, costTier) {
			continue
		}
		bucketKey, ok := requestLogStatsBucketKey(logEntry.CreatedAt, rangeSpec)
		if !ok {
			continue
		}
		bucketIndex, ok := bucketIndexes[bucketKey]
		if !ok {
			continue
		}

		bucket := seriesBuckets[bucketIndex]
		bucket.TotalRequests++
		bucket.InputTokens += int64(logEntry.InputTokens)
		bucket.OutputTokens += int64(logEntry.OutputTokens)
		bucket.ReasoningTokens += int64(logEntry.ReasoningTokens)
		bucket.CacheCreateTokens += int64(logEntry.CacheCreateTokens)
		bucket.CacheReadTokens += int64(logEntry.CacheReadTokens)
		bucket.TotalCost += logEntry.TotalCost

		stats.TotalRequests++
		stats.InputTokens += int64(logEntry.InputTokens)
		stats.OutputTokens += int64(logEntry.OutputTokens)
		stats.ReasoningTokens += int64(logEntry.ReasoningTokens)
		stats.CacheCreateTokens += int64(logEntry.CacheCreateTokens)
		stats.CacheReadTokens += int64(logEntry.CacheReadTokens)
		stats.CostInput += logEntry.InputCost
		stats.CostOutput += logEntry.OutputCost
		stats.CostCacheCreate += logEntry.CacheCreateCost
		stats.CostCacheRead += logEntry.CacheReadCost
		stats.CostTotal += logEntry.TotalCost
	}

	appendStatsSeries(&stats, seriesBuckets)
	return q.statsCache.Set(cacheKey, stats), nil
}

func requestLogStatsBucketKey(createdAt string, rangeSpec logRangeSpec) (string, bool) {
	createdAtUTC, err := parseRequestLogCreatedAt(createdAt)
	if err != nil {
		return "", false
	}
	localTime := createdAtUTC.In(time.Local)
	if rangeSpec.bucketGranularity == logRangeDaily {
		return startOfDay(localTime).Format(timeLayout), true
	}
	return startOfHour(localTime).Format(timeLayout), true
}

func parseRequestLogCreatedAt(createdAt string) (time.Time, error) {
	layouts := []string{
		timeLayout,
		"2006-01-02 15:04:05 -0700 MST",
		time.RFC3339,
		time.RFC3339Nano,
	}
	var lastErr error
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, createdAt)
		if err == nil {
			return parsed, nil
		}
		lastErr = err
	}
	return time.Time{}, lastErr
}

func newStatsSeriesBuckets(rangeSpec logRangeSpec) ([]*observabilitydomain.LogStatsSeries, map[string]int) {
	buckets := make([]*observabilitydomain.LogStatsSeries, 0, rangeSpec.seriesCount)
	indexes := make(map[string]int, rangeSpec.seriesCount)
	bucketTime := rangeSpec.startLocal

	for i := 0; i < rangeSpec.seriesCount; i++ {
		key := bucketTime.Format(timeLayout)
		buckets = append(buckets, &observabilitydomain.LogStatsSeries{Day: key})
		indexes[key] = i

		if rangeSpec.bucketGranularity == logRangeDaily {
			bucketTime = bucketTime.AddDate(0, 0, 1)
			continue
		}
		bucketTime = bucketTime.Add(time.Hour)
	}
	return buckets, indexes
}

func appendStatsSeries(stats *observabilitydomain.LogStats, seriesBuckets []*observabilitydomain.LogStatsSeries) {
	if stats == nil {
		return
	}
	for _, bucket := range seriesBuckets {
		if bucket == nil {
			continue
		}
		stats.Series = append(stats.Series, *bucket)
	}
}

func (q *SQLiteQueries) ProviderDailyStats(platform string) ([]observabilitydomain.ProviderDailyStat, error) {
	cacheKey := fmt.Sprintf("platform:%s", platform)
	if cached, ok := q.providerDailyCache.Get(cacheKey); ok {
		return cached, nil
	}
	start := startOfDay(time.Now())
	end := start.Add(24 * time.Hour)

	db, err := xdb.DB(storage.RequestLogDBName)
	if err != nil {
		return nil, err
	}

	args := []any{start.Format(timeLayout), end.Format(timeLayout)}
	platformFilter := ""
	if platform != "" {
		platformFilter = " AND platform = ?"
		args = append(args, platform)
	}

	rows, err := db.Query(fmt.Sprintf(`SELECT COALESCE(NULLIF(TRIM(provider), ''), '(unknown)') AS provider,
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
		GROUP BY provider, model`, platformFilter), args...)
	if err != nil {
		if isNoSuchTableErr(err) {
			return []observabilitydomain.ProviderDailyStat{}, nil
		}
		return nil, err
	}
	defer rows.Close()

	statMap := map[string]*observabilitydomain.ProviderDailyStat{}
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
			stat = &observabilitydomain.ProviderDailyStat{Provider: provider}
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
		stat.CostTotal += q.calculateCost(modelName.String, modelpricing.UsageSnapshot{
			InputTokens:       int(nullInt64(input)),
			OutputTokens:      int(nullInt64(output)),
			CacheCreateTokens: int(nullInt64(cacheCreate)),
			CacheReadTokens:   int(nullInt64(cacheRead)),
		}).TotalCost
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	stats := make([]observabilitydomain.ProviderDailyStat, 0, len(statMap))
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
	return q.providerDailyCache.Set(cacheKey, stats), nil
}
