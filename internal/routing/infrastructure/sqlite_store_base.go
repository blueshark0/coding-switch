package infrastructure

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"codeswitch/internal/shared/storage"

	"github.com/daodao97/xgo/xdb"
)

const (
	legacyImportMetaKey = "routing.legacy_imported"
)

type SQLiteStore struct {
	dbName string
}

func NewSQLiteStore() *SQLiteStore {
	return &SQLiteStore{dbName: storage.CoreDBName}
}

func NewSQLiteStoreWithDBName(dbName string) *SQLiteStore {
	if dbName == "" {
		dbName = storage.CoreDBName
	}
	return &SQLiteStore{dbName: dbName}
}

func (s *SQLiteStore) EnsureSchema() error {
	db, err := xdb.DB(s.dbName)
	if err != nil {
		return err
	}
	statements := []string{
		`CREATE TABLE IF NOT EXISTS route_profiles (
			platform TEXT PRIMARY KEY,
			default_provider_id INTEGER NULL,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS providers (
			platform TEXT NOT NULL,
			id INTEGER NOT NULL,
			name TEXT NOT NULL,
			api_url TEXT NOT NULL DEFAULT '',
			api_key TEXT NOT NULL DEFAULT '',
			official_site TEXT NOT NULL DEFAULT '',
			icon TEXT NOT NULL DEFAULT '',
			tint TEXT NOT NULL DEFAULT '',
			accent TEXT NOT NULL DEFAULT '',
			enabled INTEGER NOT NULL DEFAULT 0,
			position INTEGER NOT NULL,
			supported_models_json TEXT NOT NULL DEFAULT '{}',
			model_mapping_json TEXT NOT NULL DEFAULT '{}',
			PRIMARY KEY(platform, id),
			UNIQUE(platform, name)
		)`,
		`CREATE TABLE IF NOT EXISTS app_preferences (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			show_heatmap INTEGER NOT NULL DEFAULT 1,
			show_home_title INTEGER NOT NULL DEFAULT 1,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS schema_meta (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_providers_platform_position ON providers(platform, position ASC)`,
	}
	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	if _, err := db.Exec(`INSERT INTO app_preferences (id, show_heatmap, show_home_title)
		VALUES (1, 1, 1)
		ON CONFLICT(id) DO NOTHING`); err != nil {
		return err
	}
	return nil
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func encodeJSONMap[T any](value T) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func decodeJSONMap[T any](raw string, target *T) error {
	if raw == "" {
		return nil
	}
	if err := json.Unmarshal([]byte(raw), target); err != nil {
		return fmt.Errorf("decode json map: %w", err)
	}
	return nil
}

func queryDB(dbName string) (*sql.DB, error) {
	return xdb.DB(dbName)
}
