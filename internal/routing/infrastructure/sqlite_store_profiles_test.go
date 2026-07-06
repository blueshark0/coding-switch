package infrastructure

import (
	"context"
	"path/filepath"
	"testing"

	"codeswitch/internal/routing/domain"
	"codeswitch/internal/shared/kernel"
	"github.com/daodao97/xgo/xdb"
	_ "modernc.org/sqlite"
)

func TestSQLiteStoreSaveProfilePersistsRequestBodyReplacements(t *testing.T) {
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

	store := NewSQLiteStoreWithDBName(dbName)
	if err := store.EnsureSchema(); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}

	saved, err := store.SaveProfile(context.Background(), domain.RouteProfile{
		Platform:          kernel.PlatformClaude,
		DefaultProviderID: intPtr(1),
		Providers: []domain.Provider{
			{
				ID:       1,
				Name:     "alpha",
				APIURL:   "https://alpha.example",
				APIKey:   "secret",
				Position: 1,
				RequestBodyReplacements: []domain.RequestBodyReplacement{
					{Path: "metadata.user_id", Value: `"fixed-user"`},
					{Path: "temperature", Value: `0.2`},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("SaveProfile: %v", err)
	}
	if got := saved.Providers[0].RequestBodyReplacements; len(got) != 2 {
		t.Fatalf("expected saved replacements, got %+v", got)
	}

	loaded, err := store.GetProfile(context.Background(), kernel.PlatformClaude)
	if err != nil {
		t.Fatalf("GetProfile: %v", err)
	}
	got := loaded.Providers[0].RequestBodyReplacements
	if len(got) != 2 {
		t.Fatalf("expected 2 replacements, got %+v", got)
	}
	if got[0].Path != "metadata.user_id" || got[0].Value != `"fixed-user"` {
		t.Fatalf("unexpected first replacement: %+v", got[0])
	}
	if got[1].Path != "temperature" || got[1].Value != `0.2` {
		t.Fatalf("unexpected second replacement: %+v", got[1])
	}
}

func intPtr(value int) *int {
	return &value
}
