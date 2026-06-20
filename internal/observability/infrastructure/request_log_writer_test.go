package infrastructure

import (
	"path/filepath"
	"testing"

	observabilitydomain "codeswitch/internal/observability/domain"
	"codeswitch/internal/shared/storage"

	"github.com/daodao97/xgo/xdb"
)

func TestRequestLogWriter_ProcessWritesBatch(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "request_log.db")
	if err := xdb.Init(map[string]*xdb.Config{
		storage.RequestLogDBName: {
			Name:        storage.RequestLogDBName,
			Driver:      "sqlite",
			DSN:         dbPath,
			MaxOpenConn: 1,
			MaxIdleConn: 1,
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

	writer := &RequestLogWriter{}
	err = writer.Process([]*observabilitydomain.RequestLog{
		{
			Platform:          "claude",
			Model:             "claude-sonnet-4",
			Provider:          "alpha-temp",
			HttpCode:          502,
			InputTokens:       11,
			OutputTokens:      22,
			CacheCreateTokens: 3,
			CacheReadTokens:   4,
			ReasoningTokens:   5,
			IsStream:          true,
			IsFast:            true,
			DurationSec:       1.25,
			CreatedAt:         "2026-05-29 06:30:15",
		},
		nil,
		{
			Platform:    "codex",
			Model:       "gpt-5",
			Provider:    "ice",
			HttpCode:    200,
			DurationSec: 0.75,
		},
	})
	if err != nil {
		t.Fatalf("Process: %v", err)
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM request_log").Scan(&count); err != nil {
		t.Fatalf("count request_log: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 request_log rows, got %d", count)
	}

	var provider string
	var httpCode int
	var inputTokens int
	var isStream int
	var isFast int
	var durationSec float64
	var createdAt string
	if err := db.QueryRow(`
		SELECT provider, http_code, input_tokens, is_stream, is_fast, duration_sec, strftime('%Y-%m-%d %H:%M:%S', created_at)
		FROM request_log
		WHERE platform = 'claude'
	`).Scan(&provider, &httpCode, &inputTokens, &isStream, &isFast, &durationSec, &createdAt); err != nil {
		t.Fatalf("query written row: %v", err)
	}
	if provider != "alpha-temp" {
		t.Fatalf("expected provider alpha-temp, got %q", provider)
	}
	if httpCode != 502 {
		t.Fatalf("expected http_code 502, got %d", httpCode)
	}
	if inputTokens != 11 {
		t.Fatalf("expected input_tokens 11, got %d", inputTokens)
	}
	if isStream != 1 {
		t.Fatalf("expected is_stream 1, got %d", isStream)
	}
	if isFast != 1 {
		t.Fatalf("expected is_fast 1, got %d", isFast)
	}
	if durationSec != 1.25 {
		t.Fatalf("expected duration_sec 1.25, got %f", durationSec)
	}
	if createdAt != "2026-05-29 06:30:15" {
		t.Fatalf("expected created_at to be preserved, got %q", createdAt)
	}
}
