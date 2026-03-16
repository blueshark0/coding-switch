package infrastructure

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"codeswitch/internal/routing/domain"
	"codeswitch/internal/shared/kernel"
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

func (s *SQLiteStore) GetProfile(ctx context.Context, platform kernel.Platform) (domain.RouteProfile, error) {
	db, err := xdb.DB(s.dbName)
	if err != nil {
		return domain.RouteProfile{}, err
	}
	profile := domain.RouteProfile{
		Platform:  platform,
		Providers: []domain.Provider{},
	}
	var defaultProviderID sql.NullInt64
	if err := db.QueryRowContext(ctx, `SELECT default_provider_id FROM route_profiles WHERE platform = ?`, platform.String()).Scan(&defaultProviderID); err != nil {
		if err != sql.ErrNoRows {
			return domain.RouteProfile{}, err
		}
	} else if defaultProviderID.Valid {
		value := int(defaultProviderID.Int64)
		profile.DefaultProviderID = &value
	}

	rows, err := db.QueryContext(ctx, `SELECT id, name, api_url, api_key, official_site, icon, tint, accent,
		enabled, position, supported_models_json, model_mapping_json
		FROM providers
		WHERE platform = ?
		ORDER BY position ASC, id ASC`, platform.String())
	if err != nil {
		return domain.RouteProfile{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var provider domain.Provider
		var enabled int
		var supportedModelsJSON string
		var modelMappingJSON string
		if err := rows.Scan(
			&provider.ID,
			&provider.Name,
			&provider.APIURL,
			&provider.APIKey,
			&provider.OfficialSite,
			&provider.Icon,
			&provider.Tint,
			&provider.Accent,
			&enabled,
			&provider.Position,
			&supportedModelsJSON,
			&modelMappingJSON,
		); err != nil {
			return domain.RouteProfile{}, err
		}
		provider.Enabled = enabled == 1
		provider.SupportedModels = map[string]bool{}
		provider.ModelMapping = map[string]string{}
		if err := decodeJSONMap(supportedModelsJSON, &provider.SupportedModels); err != nil {
			return domain.RouteProfile{}, err
		}
		if err := decodeJSONMap(modelMappingJSON, &provider.ModelMapping); err != nil {
			return domain.RouteProfile{}, err
		}
		profile.Providers = append(profile.Providers, provider)
	}
	if err := rows.Err(); err != nil {
		return domain.RouteProfile{}, err
	}
	return profile.Normalize(), nil
}

func (s *SQLiteStore) SaveProfile(ctx context.Context, profile domain.RouteProfile) (domain.RouteProfile, error) {
	db, err := xdb.DB(s.dbName)
	if err != nil {
		return domain.RouteProfile{}, err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return domain.RouteProfile{}, err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	var defaultProviderID any
	if profile.DefaultProviderID != nil {
		defaultProviderID = *profile.DefaultProviderID
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO route_profiles(platform, default_provider_id, updated_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(platform) DO UPDATE SET default_provider_id = excluded.default_provider_id, updated_at = CURRENT_TIMESTAMP`,
		profile.Platform.String(), defaultProviderID,
	); err != nil {
		return domain.RouteProfile{}, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM providers WHERE platform = ?`, profile.Platform.String()); err != nil {
		return domain.RouteProfile{}, err
	}
	for _, provider := range profile.Providers {
		supportedModelsJSON, err := encodeJSONMap(provider.SupportedModels)
		if err != nil {
			return domain.RouteProfile{}, err
		}
		modelMappingJSON, err := encodeJSONMap(provider.ModelMapping)
		if err != nil {
			return domain.RouteProfile{}, err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO providers(
			platform, id, name, api_url, api_key, official_site, icon, tint, accent, enabled, position,
			supported_models_json, model_mapping_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			profile.Platform.String(),
			provider.ID,
			provider.Name,
			provider.APIURL,
			provider.APIKey,
			provider.OfficialSite,
			provider.Icon,
			provider.Tint,
			provider.Accent,
			boolToInt(provider.Enabled),
			provider.Position,
			supportedModelsJSON,
			modelMappingJSON,
		); err != nil {
			return domain.RouteProfile{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return domain.RouteProfile{}, err
	}
	tx = nil
	return s.GetProfile(ctx, profile.Platform)
}

func (s *SQLiteStore) GetAppPreferences(ctx context.Context) (domain.AppPreferences, error) {
	db, err := xdb.DB(s.dbName)
	if err != nil {
		return domain.AppPreferences{}, err
	}
	preferences := domain.DefaultAppPreferences()
	var showHeatmap int
	var showHomeTitle int
	if err := db.QueryRowContext(ctx, `SELECT show_heatmap, show_home_title FROM app_preferences WHERE id = 1`).Scan(&showHeatmap, &showHomeTitle); err != nil {
		if err == sql.ErrNoRows {
			return preferences, nil
		}
		return domain.AppPreferences{}, err
	}
	preferences.ShowHeatmap = showHeatmap == 1
	preferences.ShowHomeTitle = showHomeTitle == 1
	return preferences, nil
}

func (s *SQLiteStore) SaveAppPreferences(ctx context.Context, preferences domain.AppPreferences) (domain.AppPreferences, error) {
	db, err := xdb.DB(s.dbName)
	if err != nil {
		return domain.AppPreferences{}, err
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO app_preferences(id, show_heatmap, show_home_title, updated_at)
		VALUES (1, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			show_heatmap = excluded.show_heatmap,
			show_home_title = excluded.show_home_title,
			updated_at = CURRENT_TIMESTAMP`,
		boolToInt(preferences.ShowHeatmap),
		boolToInt(preferences.ShowHomeTitle),
	); err != nil {
		return domain.AppPreferences{}, err
	}
	return s.GetAppPreferences(ctx)
}

func (s *SQLiteStore) HasLegacyImport(ctx context.Context) (bool, error) {
	return s.metaEquals(ctx, legacyImportMetaKey, "1")
}

func (s *SQLiteStore) MarkLegacyImported(ctx context.Context) error {
	db, err := xdb.DB(s.dbName)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `INSERT INTO schema_meta(key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, legacyImportMetaKey, "1")
	return err
}

func (s *SQLiteStore) metaEquals(ctx context.Context, key string, want string) (bool, error) {
	db, err := xdb.DB(storage.CoreDBName)
	if err != nil {
		return false, err
	}
	var value string
	if err := db.QueryRowContext(ctx, `SELECT value FROM schema_meta WHERE key = ?`, key).Scan(&value); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return value == want, nil
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
