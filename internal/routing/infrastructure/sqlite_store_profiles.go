package infrastructure

import (
	"context"
	"database/sql"

	"codeswitch/internal/routing/domain"
	"codeswitch/internal/shared/kernel"
)

func (s *SQLiteStore) GetProfile(ctx context.Context, platform kernel.Platform) (domain.RouteProfile, error) {
	db, err := queryDB(s.dbName)
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
	db, err := queryDB(s.dbName)
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

	existingIDs, err := loadProviderIDs(ctx, tx, profile.Platform.String())
	if err != nil {
		return domain.RouteProfile{}, err
	}
	incomingIDs := make(map[int]struct{}, len(profile.Providers))
	for _, provider := range profile.Providers {
		incomingIDs[provider.ID] = struct{}{}
	}
	for _, providerID := range existingIDs {
		if _, ok := incomingIDs[providerID]; ok {
			continue
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM providers WHERE platform = ? AND id = ?`, profile.Platform.String(), providerID); err != nil {
			return domain.RouteProfile{}, err
		}
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
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(platform, id) DO UPDATE SET
			name = excluded.name,
			api_url = excluded.api_url,
			api_key = excluded.api_key,
			official_site = excluded.official_site,
			icon = excluded.icon,
			tint = excluded.tint,
			accent = excluded.accent,
			enabled = excluded.enabled,
			position = excluded.position,
			supported_models_json = excluded.supported_models_json,
			model_mapping_json = excluded.model_mapping_json`,
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

func loadProviderIDs(ctx context.Context, tx *sql.Tx, platform string) ([]int, error) {
	rows, err := tx.QueryContext(ctx, `SELECT id FROM providers WHERE platform = ?`, platform)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make([]int, 0)
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ids, nil
}
