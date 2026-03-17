package infrastructure

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"codeswitch/internal/shared/storage"

	"github.com/daodao97/xgo/xdb"
	_ "modernc.org/sqlite"
)

func TestListRequestLogs_FiltersByNaturalDayRange(t *testing.T) {
	loc := time.FixedZone("UTC+8", 8*60*60)
	now := time.Date(2026, time.March, 16, 12, 0, 0, 0, loc)
	db := setupRangeQueryTestDB(t)
	setRangeTestClock(t, now)

	seedRequestLog(t, db, time.Date(2026, time.March, 16, 9, 0, 0, 0, loc), "claude", "alpha", "gpt-4.1")
	seedRequestLog(t, db, time.Date(2026, time.March, 15, 11, 0, 0, 0, loc), "claude", "alpha", "gpt-4.1")
	seedRequestLog(t, db, time.Date(2026, time.March, 14, 8, 0, 0, 0, loc), "claude", "beta", "gpt-4.1")
	seedRequestLog(t, db, time.Date(2026, time.March, 10, 10, 0, 0, 0, loc), "codex", "alpha", "gpt-4.1")
	seedRequestLog(t, db, time.Date(2026, time.March, 9, 10, 0, 0, 0, loc), "claude", "alpha", "gpt-4.1")

	queries := NewSQLiteQueries()

	todayLogs, err := queries.ListRequestLogs("", "", "today", 100)
	if err != nil {
		t.Fatalf("ListRequestLogs today: %v", err)
	}
	if got := len(todayLogs); got != 1 {
		t.Fatalf("expected 1 today log, got %d", got)
	}

	last3DaysLogs, err := queries.ListRequestLogs("", "", "last3days", 100)
	if err != nil {
		t.Fatalf("ListRequestLogs last3days: %v", err)
	}
	if got := len(last3DaysLogs); got != 3 {
		t.Fatalf("expected 3 logs in last3days, got %d", got)
	}

	last7DaysLogs, err := queries.ListRequestLogs("", "", "last7days", 100)
	if err != nil {
		t.Fatalf("ListRequestLogs last7days: %v", err)
	}
	if got := len(last7DaysLogs); got != 4 {
		t.Fatalf("expected 4 logs in last7days, got %d", got)
	}

	filteredLogs, err := queries.ListRequestLogs("claude", "alpha", "last3days", 100)
	if err != nil {
		t.Fatalf("ListRequestLogs filtered last3days: %v", err)
	}
	if got := len(filteredLogs); got != 2 {
		t.Fatalf("expected 2 filtered logs in last3days, got %d", got)
	}
}

func TestStatsSince_UsesAdaptiveBucketsForRange(t *testing.T) {
	loc := time.FixedZone("UTC+8", 8*60*60)
	now := time.Date(2026, time.March, 16, 12, 0, 0, 0, loc)
	db := setupRangeQueryTestDB(t)
	setRangeTestClock(t, now)

	seedRequestLogWithUsage(t, db, time.Date(2026, time.March, 16, 9, 15, 0, 0, loc), "claude", "alpha", "gpt-4.1", 10, 5, 2)
	seedRequestLogWithUsage(t, db, time.Date(2026, time.March, 16, 9, 45, 0, 0, loc), "claude", "alpha", "gpt-4.1", 2, 3, 1)
	seedRequestLogWithUsage(t, db, time.Date(2026, time.March, 15, 11, 0, 0, 0, loc), "claude", "alpha", "gpt-4.1", 7, 1, 0)
	seedRequestLogWithUsage(t, db, time.Date(2026, time.March, 14, 8, 0, 0, 0, loc), "claude", "beta", "gpt-4.1", 4, 6, 0)
	seedRequestLogWithUsage(t, db, time.Date(2026, time.March, 13, 20, 0, 0, 0, loc), "claude", "alpha", "gpt-4.1", 8, 1, 0)

	queries := NewSQLiteQueries()

	todayStats, err := queries.StatsSince("claude", "alpha", "today")
	if err != nil {
		t.Fatalf("StatsSince today: %v", err)
	}
	if got := len(todayStats.Series); got != 24 {
		t.Fatalf("expected 24 hourly buckets for today, got %d", got)
	}
	if todayStats.TotalRequests != 2 {
		t.Fatalf("expected 2 today requests, got %d", todayStats.TotalRequests)
	}
	if todayStats.InputTokens != 12 || todayStats.OutputTokens != 8 || todayStats.ReasoningTokens != 3 {
		t.Fatalf("unexpected today totals: input=%d output=%d reasoning=%d", todayStats.InputTokens, todayStats.OutputTokens, todayStats.ReasoningTokens)
	}
	if bucket := todayStats.Series[9]; bucket.TotalRequests != 2 {
		t.Fatalf("expected 09:00 bucket to have 2 requests, got %d", bucket.TotalRequests)
	}

	last3DaysStats, err := queries.StatsSince("claude", "", "last3days")
	if err != nil {
		t.Fatalf("StatsSince last3days: %v", err)
	}
	if got := len(last3DaysStats.Series); got != 3 {
		t.Fatalf("expected 3 daily buckets for last3days, got %d", got)
	}
	if last3DaysStats.TotalRequests != 4 {
		t.Fatalf("expected 4 requests in last3days, got %d", last3DaysStats.TotalRequests)
	}
	if got := last3DaysStats.Series[0].Day; got != "2026-03-14 00:00:00" {
		t.Fatalf("unexpected first last3days bucket: %s", got)
	}
	if last3DaysStats.Series[0].TotalRequests != 1 || last3DaysStats.Series[1].TotalRequests != 1 || last3DaysStats.Series[2].TotalRequests != 2 {
		t.Fatalf("unexpected last3days bucket counts: %+v", last3DaysStats.Series)
	}

	last7DaysStats, err := queries.StatsSince("claude", "alpha", "last7days")
	if err != nil {
		t.Fatalf("StatsSince last7days: %v", err)
	}
	if got := len(last7DaysStats.Series); got != 7 {
		t.Fatalf("expected 7 daily buckets for last7days, got %d", got)
	}
	if last7DaysStats.TotalRequests != 4 {
		t.Fatalf("expected 4 alpha requests in last7days, got %d", last7DaysStats.TotalRequests)
	}
}

func setupRangeQueryTestDB(t *testing.T) *sql.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "request_log.db")
	if err := xdb.Init(map[string]*xdb.Config{
		storage.RequestLogDBName: {
			Name:   storage.RequestLogDBName,
			Driver: "sqlite",
			DSN:    dbPath,
		},
	}); err != nil {
		t.Fatalf("xdb.Init: %v", err)
	}

	db, err := xdb.DB(storage.RequestLogDBName)
	if err != nil {
		t.Fatalf("xdb.DB: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	if err := EnsureRequestLogTableWithDB(db); err != nil {
		t.Fatalf("EnsureRequestLogTableWithDB: %v", err)
	}
	return db
}

func setRangeTestClock(t *testing.T, now time.Time) {
	t.Helper()

	originalLocal := time.Local
	originalNow := timeNow
	time.Local = now.Location()
	timeNow = func() time.Time { return now }
	t.Cleanup(func() {
		time.Local = originalLocal
		timeNow = originalNow
	})
}

func seedRequestLog(t *testing.T, db *sql.DB, createdAt time.Time, platform string, provider string, model string) {
	t.Helper()
	seedRequestLogWithUsage(t, db, createdAt, platform, provider, model, 1, 1, 0)
}

func seedRequestLogWithUsage(
	t *testing.T,
	db *sql.DB,
	createdAt time.Time,
	platform string,
	provider string,
	model string,
	inputTokens int,
	outputTokens int,
	reasoningTokens int,
) {
	t.Helper()

	_, err := db.Exec(
		`INSERT INTO request_log (
			platform,
			model,
			provider,
			http_code,
			input_tokens,
			output_tokens,
			cache_create_tokens,
			cache_read_tokens,
			reasoning_tokens,
			is_stream,
			duration_sec,
			created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		platform,
		model,
		provider,
		200,
		inputTokens,
		outputTokens,
		0,
		0,
		reasoningTokens,
		0,
		0.5,
		createdAt.UTC().Format(timeLayout),
	)
	if err != nil {
		t.Fatalf("seed request_log: %v", err)
	}
}
