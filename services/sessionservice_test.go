package services

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/daodao97/xgo/xdb"
)

func TestSessionService_GetPlatformSessions_FiltersExpiredAndSorts(t *testing.T) {
	service, db := setupSessionServiceTestDB(t)
	now := time.Now()

	records := []SessionBinding{
		{
			Platform:      "claude",
			SessionID:     "claude-alpha-new",
			ProviderName:  "alpha",
			LastSuccessAt: now.Add(-30 * time.Second),
			CreatedAt:     now.Add(-5 * time.Minute),
		},
		{
			Platform:      "claude",
			SessionID:     "claude-alpha-old",
			ProviderName:  "alpha",
			LastSuccessAt: now.Add(-2 * time.Minute),
			CreatedAt:     now.Add(-8 * time.Minute),
		},
		{
			Platform:      "claude",
			SessionID:     "claude-beta",
			ProviderName:  "beta",
			LastSuccessAt: now.Add(-1 * time.Minute),
			CreatedAt:     now.Add(-10 * time.Minute),
		},
		{
			Platform:      "codex",
			SessionID:     "codex-alpha",
			ProviderName:  "alpha",
			LastSuccessAt: now.Add(-1 * time.Minute),
			CreatedAt:     now.Add(-10 * time.Minute),
		},
		{
			Platform:      "claude",
			SessionID:     "claude-expired",
			ProviderName:  "alpha",
			LastSuccessAt: now.Add(-20 * time.Minute),
			CreatedAt:     now.Add(-30 * time.Minute),
		},
	}
	insertSessionBindings(t, db, records)

	got, err := service.GetPlatformSessions("claude")
	if err != nil {
		t.Fatalf("GetPlatformSessions() error = %v", err)
	}

	wantOrder := []string{"claude-alpha-new", "claude-alpha-old", "claude-beta"}
	if len(got) != len(wantOrder) {
		t.Fatalf("len(sessions) = %d, want %d", len(got), len(wantOrder))
	}

	for i, wantSessionID := range wantOrder {
		if got[i].SessionID != wantSessionID {
			t.Fatalf("sessions[%d].SessionID = %q, want %q", i, got[i].SessionID, wantSessionID)
		}
		if got[i].Platform != "claude" {
			t.Fatalf("sessions[%d].Platform = %q, want claude", i, got[i].Platform)
		}
	}
}

func TestSessionService_GetPlatformSessions_ReflectsUnbind(t *testing.T) {
	service, db := setupSessionServiceTestDB(t)
	now := time.Now()

	records := []SessionBinding{
		{
			Platform:      "claude",
			SessionID:     "keep-me",
			ProviderName:  "alpha",
			LastSuccessAt: now.Add(-1 * time.Minute),
			CreatedAt:     now.Add(-6 * time.Minute),
		},
		{
			Platform:      "claude",
			SessionID:     "unbind-me",
			ProviderName:  "beta",
			LastSuccessAt: now.Add(-2 * time.Minute),
			CreatedAt:     now.Add(-7 * time.Minute),
		},
	}
	insertSessionBindings(t, db, records)

	if err := service.UnbindSession("claude", "unbind-me"); err != nil {
		t.Fatalf("UnbindSession() error = %v", err)
	}

	got, err := service.GetPlatformSessions("claude")
	if err != nil {
		t.Fatalf("GetPlatformSessions() error = %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("len(sessions) = %d, want 1", len(got))
	}
	if got[0].SessionID != "keep-me" {
		t.Fatalf("remaining session = %q, want keep-me", got[0].SessionID)
	}
}

func setupSessionServiceTestDB(t *testing.T) (*SessionService, *sql.DB) {
	t.Helper()

	dbName := fmt.Sprintf("session_test_%d", time.Now().UnixNano())
	dbPath := filepath.Join(t.TempDir(), "session_test.db")
	dsn := fmt.Sprintf("%s?cache=shared&mode=rwc&_journal_mode=WAL&_busy_timeout=5000", dbPath)

	if err := xdb.Inits([]xdb.Config{
		{
			Name:        dbName,
			Driver:      "sqlite",
			DSN:         dsn,
			MaxOpenConn: 1,
			MaxIdleConn: 1,
		},
	}); err != nil {
		t.Fatalf("xdb.Inits() error = %v", err)
	}

	db, err := xdb.DB(dbName)
	if err != nil {
		t.Fatalf("xdb.DB() error = %v", err)
	}

	if err := ensureSessionBindingTableWithDB(db); err != nil {
		t.Fatalf("ensureSessionBindingTableWithDB() error = %v", err)
	}

	return NewSessionService(dbName), db
}

func insertSessionBindings(t *testing.T, db *sql.DB, records []SessionBinding) {
	t.Helper()

	stmt := `INSERT INTO session_provider_binding (platform, session_id, provider_name, last_success_at, created_at)
		VALUES (?, ?, ?, ?, ?)`
	for _, record := range records {
		if _, err := db.Exec(
			stmt,
			record.Platform,
			record.SessionID,
			record.ProviderName,
			record.LastSuccessAt,
			record.CreatedAt,
		); err != nil {
			t.Fatalf("insert session_provider_binding error = %v", err)
		}
	}
}
