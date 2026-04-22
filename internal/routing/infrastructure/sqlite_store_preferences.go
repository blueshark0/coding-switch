package infrastructure

import (
	"context"
	"database/sql"

	"codeswitch/internal/routing/domain"
)

func (s *SQLiteStore) GetAppPreferences(ctx context.Context) (domain.AppPreferences, error) {
	db, err := queryDB(s.dbName)
	if err != nil {
		return domain.AppPreferences{}, err
	}
	preferences := domain.DefaultAppPreferences()
	var showHeatmap int
	var showHomeTitle int
	var proxyEnabled int
	var proxyURL string
	if err := db.QueryRowContext(ctx,
		`SELECT show_heatmap, show_home_title, proxy_enabled, proxy_url FROM app_preferences WHERE id = 1`,
	).Scan(&showHeatmap, &showHomeTitle, &proxyEnabled, &proxyURL); err != nil {
		if err == sql.ErrNoRows {
			return preferences, nil
		}
		return domain.AppPreferences{}, err
	}
	preferences.ShowHeatmap = showHeatmap == 1
	preferences.ShowHomeTitle = showHomeTitle == 1
	preferences.ProxyEnabled = proxyEnabled == 1
	preferences.ProxyURL = proxyURL
	return preferences, nil
}

func (s *SQLiteStore) SaveAppPreferences(ctx context.Context, preferences domain.AppPreferences) (domain.AppPreferences, error) {
	if err := domain.ValidateProxyURL(preferences.ProxyURL); err != nil {
		return domain.AppPreferences{}, err
	}
	db, err := queryDB(s.dbName)
	if err != nil {
		return domain.AppPreferences{}, err
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO app_preferences(id, show_heatmap, show_home_title, proxy_enabled, proxy_url, updated_at)
			VALUES (1, ?, ?, ?, ?, CURRENT_TIMESTAMP)
			ON CONFLICT(id) DO UPDATE SET
				show_heatmap = excluded.show_heatmap,
				show_home_title = excluded.show_home_title,
				proxy_enabled = excluded.proxy_enabled,
				proxy_url = excluded.proxy_url,
				updated_at = CURRENT_TIMESTAMP`,
		boolToInt(preferences.ShowHeatmap),
		boolToInt(preferences.ShowHomeTitle),
		boolToInt(preferences.ProxyEnabled),
		preferences.ProxyURL,
	); err != nil {
		return domain.AppPreferences{}, err
	}
	return s.GetAppPreferences(ctx)
}
