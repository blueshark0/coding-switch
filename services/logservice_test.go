package services

import (
	"fmt"
	"math"
	"path/filepath"
	"testing"
	"time"

	modelpricing "codeswitch/resources/model-pricing"

	"github.com/daodao97/xgo/xdb"
)

type statsTestRecord struct {
	platform          string
	provider          string
	model             string
	httpCode          int
	inputTokens       int
	outputTokens      int
	reasoningTokens   int
	cacheCreateTokens int
	cacheReadTokens   int
	createdAt         time.Time
}

type expectedStatsTotals struct {
	totalRequests     int64
	inputTokens       int64
	outputTokens      int64
	reasoningTokens   int64
	cacheCreateTokens int64
	cacheReadTokens   int64
	costInput         float64
	costOutput        float64
	costCacheCreate   float64
	costCacheRead     float64
	costTotal         float64
}

func TestLogService_StatsSince_AppliesPlatformAndProviderFilters(t *testing.T) {
	setupStatsTestDB(t)

	now := time.Now().Add(-5 * time.Minute)
	records := []statsTestRecord{
		{
			platform:          "claude",
			provider:          "provider-a",
			model:             "claude-3-5-haiku-latest",
			httpCode:          200,
			inputTokens:       100,
			outputTokens:      40,
			reasoningTokens:   10,
			cacheCreateTokens: 6,
			cacheReadTokens:   3,
			createdAt:         now,
		},
		{
			platform:          "claude",
			provider:          "provider-b",
			model:             "claude-3-5-haiku-latest",
			httpCode:          200,
			inputTokens:       80,
			outputTokens:      30,
			reasoningTokens:   5,
			cacheCreateTokens: 2,
			cacheReadTokens:   1,
			createdAt:         now.Add(-2 * time.Minute),
		},
		{
			platform:          "codex",
			provider:          "provider-a",
			model:             "claude-3-5-haiku-latest",
			httpCode:          200,
			inputTokens:       60,
			outputTokens:      20,
			reasoningTokens:   7,
			cacheCreateTokens: 1,
			cacheReadTokens:   0,
			createdAt:         now.Add(-1 * time.Minute),
		},
		{
			platform:          "claude",
			provider:          "provider-a",
			model:             "claude-3-5-haiku-latest",
			httpCode:          200,
			inputTokens:       45,
			outputTokens:      25,
			reasoningTokens:   3,
			cacheCreateTokens: 2,
			cacheReadTokens:   2,
			createdAt:         now.Add(-30 * time.Second),
		},
	}
	insertStatsTestRecords(t, records)

	logService := NewLogService()
	testCases := []struct {
		name     string
		platform string
		provider string
	}{
		{name: "platform and provider", platform: "claude", provider: "provider-a"},
		{name: "platform only", platform: "claude", provider: ""},
		{name: "provider only", platform: "", provider: "provider-a"},
		{name: "no matches", platform: "claude", provider: "missing-provider"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := logService.StatsSince(tc.platform, tc.provider)
			if err != nil {
				t.Fatalf("StatsSince() error = %v", err)
			}

			want := aggregateExpectedStats(logService, records, tc.platform, tc.provider)
			assertStatsTotals(t, got, want)

			if len(got.Series) != 24 {
				t.Fatalf("len(Series) = %d, want 24", len(got.Series))
			}

			var seriesRequests int64
			for _, bucket := range got.Series {
				seriesRequests += bucket.TotalRequests
			}
			if seriesRequests != got.TotalRequests {
				t.Fatalf("sum(series.total_requests) = %d, want %d", seriesRequests, got.TotalRequests)
			}

			if tc.name == "no matches" {
				for idx, bucket := range got.Series {
					if bucket.TotalRequests != 0 || bucket.TotalCost != 0 {
						t.Fatalf("series[%d] should be empty, got requests=%d cost=%f", idx, bucket.TotalRequests, bucket.TotalCost)
					}
				}
			}
		})
	}
}

func setupStatsTestDB(t *testing.T) {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "request_log_test.db")
	dsn := fmt.Sprintf("%s?cache=shared&mode=rwc&_journal_mode=WAL&_busy_timeout=5000", dbPath)
	if err := xdb.Inits([]xdb.Config{
		{
			Name:        RequestLogDBName,
			Driver:      "sqlite",
			DSN:         dsn,
			MaxOpenConn: 1,
			MaxIdleConn: 1,
		},
	}); err != nil {
		t.Fatalf("xdb.Inits() error = %v", err)
	}

	db, err := xdb.DB(RequestLogDBName)
	if err != nil {
		t.Fatalf("xdb.DB() error = %v", err)
	}
	if err := ensureRequestLogTableWithDB(db); err != nil {
		t.Fatalf("ensureRequestLogTableWithDB() error = %v", err)
	}
}

func insertStatsTestRecords(t *testing.T, records []statsTestRecord) {
	t.Helper()

	db, err := xdb.DB(RequestLogDBName)
	if err != nil {
		t.Fatalf("xdb.DB() error = %v", err)
	}

	stmt := `INSERT INTO request_log (
		platform, model, provider, http_code, input_tokens, output_tokens,
		cache_create_tokens, cache_read_tokens, reasoning_tokens, is_stream, duration_sec, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	for _, record := range records {
		if _, err := db.Exec(
			stmt,
			record.platform,
			record.model,
			record.provider,
			record.httpCode,
			record.inputTokens,
			record.outputTokens,
			record.cacheCreateTokens,
			record.cacheReadTokens,
			record.reasoningTokens,
			0,
			0,
			record.createdAt.Format(timeLayout),
		); err != nil {
			t.Fatalf("insert request_log error = %v", err)
		}
	}
}

func aggregateExpectedStats(logService *LogService, records []statsTestRecord, platform string, provider string) expectedStatsTotals {
	var totals expectedStatsTotals
	for _, record := range records {
		if platform != "" && record.platform != platform {
			continue
		}
		if provider != "" && record.provider != provider {
			continue
		}
		totals.totalRequests++
		totals.inputTokens += int64(record.inputTokens)
		totals.outputTokens += int64(record.outputTokens)
		totals.reasoningTokens += int64(record.reasoningTokens)
		totals.cacheCreateTokens += int64(record.cacheCreateTokens)
		totals.cacheReadTokens += int64(record.cacheReadTokens)
		cost := logService.calculateCost(record.model, modelpricing.UsageSnapshot{
			InputTokens:       record.inputTokens,
			OutputTokens:      record.outputTokens,
			CacheCreateTokens: record.cacheCreateTokens,
			CacheReadTokens:   record.cacheReadTokens,
		})
		totals.costInput += cost.InputCost
		totals.costOutput += cost.OutputCost
		totals.costCacheCreate += cost.CacheCreateCost
		totals.costCacheRead += cost.CacheReadCost
		totals.costTotal += cost.TotalCost
	}
	return totals
}

func assertStatsTotals(t *testing.T, got LogStats, want expectedStatsTotals) {
	t.Helper()

	if got.TotalRequests != want.totalRequests {
		t.Fatalf("TotalRequests = %d, want %d", got.TotalRequests, want.totalRequests)
	}
	if got.InputTokens != want.inputTokens {
		t.Fatalf("InputTokens = %d, want %d", got.InputTokens, want.inputTokens)
	}
	if got.OutputTokens != want.outputTokens {
		t.Fatalf("OutputTokens = %d, want %d", got.OutputTokens, want.outputTokens)
	}
	if got.ReasoningTokens != want.reasoningTokens {
		t.Fatalf("ReasoningTokens = %d, want %d", got.ReasoningTokens, want.reasoningTokens)
	}
	if got.CacheCreateTokens != want.cacheCreateTokens {
		t.Fatalf("CacheCreateTokens = %d, want %d", got.CacheCreateTokens, want.cacheCreateTokens)
	}
	if got.CacheReadTokens != want.cacheReadTokens {
		t.Fatalf("CacheReadTokens = %d, want %d", got.CacheReadTokens, want.cacheReadTokens)
	}
	assertFloatAlmostEqual(t, "CostInput", got.CostInput, want.costInput)
	assertFloatAlmostEqual(t, "CostOutput", got.CostOutput, want.costOutput)
	assertFloatAlmostEqual(t, "CostCacheCreate", got.CostCacheCreate, want.costCacheCreate)
	assertFloatAlmostEqual(t, "CostCacheRead", got.CostCacheRead, want.costCacheRead)
	assertFloatAlmostEqual(t, "CostTotal", got.CostTotal, want.costTotal)
}

func assertFloatAlmostEqual(t *testing.T, field string, got float64, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-12 {
		t.Fatalf("%s = %.15f, want %.15f", field, got, want)
	}
}
