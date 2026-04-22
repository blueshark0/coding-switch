package infrastructure

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/daodao97/xgo/xdb"
	_ "modernc.org/sqlite"
)

func TestSQLiteStoreEnsureSchema_MigratesLegacyAppPreferencesColumns(t *testing.T) {
	dbName := "routing_test_" + t.Name()
	dbPath := filepath.Join(t.TempDir(), "app.db")

	if err := xdb.Init(map[string]*xdb.Config{
		dbName: {
			Name:   dbName,
			Driver: "sqlite",
			DSN:    dbPath,
		},
	}); err != nil {
		t.Fatalf("xdb.Init: %v", err)
	}

	db, err := xdb.DB(dbName)
	if err != nil {
		t.Fatalf("xdb.DB: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	if _, err := db.Exec(`
		CREATE TABLE app_preferences (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			show_heatmap INTEGER NOT NULL DEFAULT 1,
			show_home_title INTEGER NOT NULL DEFAULT 1,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		INSERT INTO app_preferences (id, show_heatmap, show_home_title)
		VALUES (1, 0, 1);
	`); err != nil {
		t.Fatalf("seed legacy app_preferences: %v", err)
	}

	store := NewSQLiteStoreWithDBName(dbName)
	if err := store.EnsureSchema(); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}

	for _, column := range []string{"proxy_enabled", "proxy_url"} {
		var count int
		if err := db.QueryRow(
			"SELECT COUNT(*) FROM pragma_table_info('app_preferences') WHERE name = ?",
			column,
		).Scan(&count); err != nil {
			t.Fatalf("query column %s: %v", column, err)
		}
		if count != 1 {
			t.Fatalf("expected column %s to exist once, got %d", column, count)
		}
	}

	prefs, err := store.GetAppPreferences(context.Background())
	if err != nil {
		t.Fatalf("GetAppPreferences: %v", err)
	}
	if prefs.ShowHeatmap {
		t.Fatal("expected existing show_heatmap value to be preserved")
	}
	if !prefs.ShowHomeTitle {
		t.Fatal("expected existing show_home_title value to be preserved")
	}
	if prefs.ProxyEnabled {
		t.Fatal("expected proxy_enabled default false after migration")
	}
	if prefs.ProxyURL != "" {
		t.Fatalf("expected empty proxy_url after migration, got %q", prefs.ProxyURL)
	}
}

func TestSQLiteStoreEnsureSchema_CreatesDefaultAppPreferencesRow(t *testing.T) {
	dbName := "routing_test_" + t.Name()
	dbPath := filepath.Join(t.TempDir(), "app.db")

	if err := xdb.Init(map[string]*xdb.Config{
		dbName: {
			Name:   dbName,
			Driver: "sqlite",
			DSN:    dbPath,
		},
	}); err != nil {
		t.Fatalf("xdb.Init: %v", err)
	}

	db, err := xdb.DB(dbName)
	if err != nil {
		t.Fatalf("xdb.DB: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	if _, err := db.Exec(`
		CREATE TABLE app_preferences (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			show_heatmap INTEGER NOT NULL DEFAULT 1,
			show_home_title INTEGER NOT NULL DEFAULT 1,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`); err != nil {
		t.Fatalf("create legacy app_preferences: %v", err)
	}

	store := NewSQLiteStoreWithDBName(dbName)
	if err := store.EnsureSchema(); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}

	var (
		id            int
		showHeatmap   int
		showHomeTitle int
		proxyEnabled  int
		proxyURL      string
	)
	if err := db.QueryRow(
		`SELECT id, show_heatmap, show_home_title, proxy_enabled, proxy_url
		 FROM app_preferences WHERE id = 1`,
	).Scan(&id, &showHeatmap, &showHomeTitle, &proxyEnabled, &proxyURL); err != nil {
		t.Fatalf("query default app_preferences row: %v", err)
	}

	if got := fmt.Sprintf("%d:%d:%d:%d:%s", id, showHeatmap, showHomeTitle, proxyEnabled, proxyURL); got != "1:1:1:0:" {
		t.Fatalf("unexpected default row: %s", got)
	}
}
