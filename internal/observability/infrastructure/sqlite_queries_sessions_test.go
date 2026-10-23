package infrastructure

import (
	"path/filepath"
	"testing"

	observabilitydomain "codeswitch/internal/observability/domain"
	"codeswitch/internal/shared/storage"
	modelpricing "codeswitch/resources/model-pricing"

	"github.com/daodao97/xgo/xdb"
)

func TestSessionUsageByPlatform_AggregatesPerSession(t *testing.T) {
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

	pricing, err := modelpricing.NewService()
	if err != nil {
		t.Fatalf("modelpricing.NewService: %v", err)
	}
	writer := &RequestLogWriter{pricing: pricing}
	err = writer.Process([]*observabilitydomain.RequestLog{
		{Platform: "claude", Model: "claude-sonnet-4", Provider: "alpha", SessionID: "s-1", HttpCode: 200, InputTokens: 10, OutputTokens: 20},
		{Platform: "claude", Model: "claude-opus-4", Provider: "alpha", SessionID: "s-1", HttpCode: 200, InputTokens: 30, OutputTokens: 40},
		{Platform: "claude", Model: "claude-sonnet-4", Provider: "beta", SessionID: "s-2", HttpCode: 200, InputTokens: 5, OutputTokens: 6},
		{Platform: "codex", Model: "gpt-5", Provider: "ice", SessionID: "c-1", HttpCode: 200, InputTokens: 7, OutputTokens: 8},
		// 无会话 ID 的行（历史数据）不参与聚合。
		{Platform: "claude", Model: "claude-sonnet-4", Provider: "alpha", HttpCode: 200, InputTokens: 100, OutputTokens: 100},
	})
	if err != nil {
		t.Fatalf("Process: %v", err)
	}

	usage, err := NewSQLiteQueries().SessionUsageByPlatform("claude")
	if err != nil {
		t.Fatalf("SessionUsageByPlatform: %v", err)
	}
	if len(usage) != 2 {
		t.Fatalf("expected 2 claude sessions, got %d", len(usage))
	}

	s1 := usage["s-1"]
	if s1.Requests != 2 {
		t.Fatalf("expected s-1 requests 2, got %d", s1.Requests)
	}
	if s1.InputTokens != 40 || s1.OutputTokens != 60 {
		t.Fatalf("expected s-1 tokens 40/60, got %d/%d", s1.InputTokens, s1.OutputTokens)
	}
	if s1.TotalCost <= 0 {
		t.Fatalf("expected s-1 total cost > 0, got %f", s1.TotalCost)
	}
	if len(s1.Models) != 2 {
		t.Fatalf("expected s-1 to have 2 distinct models, got %v", s1.Models)
	}

	if usage["s-2"].Requests != 1 {
		t.Fatalf("expected s-2 requests 1, got %d", usage["s-2"].Requests)
	}

	codexUsage, err := NewSQLiteQueries().SessionUsageByPlatform("codex")
	if err != nil {
		t.Fatalf("SessionUsageByPlatform codex: %v", err)
	}
	if len(codexUsage) != 1 || codexUsage["c-1"].Requests != 1 {
		t.Fatalf("expected 1 codex session with 1 request, got %+v", codexUsage)
	}
}
