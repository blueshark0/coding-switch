package application

import (
	"context"
	"strings"
	"testing"

	"codeswitch/internal/routing/domain"
	"codeswitch/internal/shared/kernel"
)

type routingRepoStub struct {
	getProfileCalls  int
	saveProfileCalls int
	profiles         map[kernel.Platform]domain.RouteProfile
}

func (r *routingRepoStub) GetProfile(_ context.Context, platform kernel.Platform) (domain.RouteProfile, error) {
	r.getProfileCalls++
	return r.profiles[platform], nil
}

func (r *routingRepoStub) SaveProfile(_ context.Context, profile domain.RouteProfile) (domain.RouteProfile, error) {
	r.saveProfileCalls++
	r.profiles[profile.Platform] = profile.Normalize()
	return r.profiles[profile.Platform], nil
}

func (r *routingRepoStub) GetAppPreferences(_ context.Context) (domain.AppPreferences, error) {
	return domain.DefaultAppPreferences(), nil
}

func (r *routingRepoStub) SaveAppPreferences(_ context.Context, preferences domain.AppPreferences) (domain.AppPreferences, error) {
	return preferences, nil
}

func TestServiceGetProfileUsesCache(t *testing.T) {
	repo := &routingRepoStub{
		profiles: map[kernel.Platform]domain.RouteProfile{
			kernel.PlatformClaude: {
				Platform: kernel.PlatformClaude,
				Providers: []domain.Provider{
					{ID: 1, Name: "alpha", Position: 1},
				},
			},
		},
	}
	service := NewService(repo)

	first, err := service.GetProfile(context.Background(), kernel.PlatformClaude.String())
	if err != nil {
		t.Fatalf("first get profile: %v", err)
	}
	first.Providers[0].Name = "mutated"

	second, err := service.GetProfile(context.Background(), kernel.PlatformClaude.String())
	if err != nil {
		t.Fatalf("second get profile: %v", err)
	}

	if repo.getProfileCalls != 1 {
		t.Fatalf("expected 1 repo call, got %d", repo.getProfileCalls)
	}
	if second.Providers[0].Name != "alpha" {
		t.Fatalf("expected cached profile clone, got %+v", second.Providers[0])
	}
}

func TestServiceSaveProfileRefreshesCache(t *testing.T) {
	repo := &routingRepoStub{
		profiles: map[kernel.Platform]domain.RouteProfile{
			kernel.PlatformClaude: {
				Platform:          kernel.PlatformClaude,
				DefaultProviderID: intPtr(1),
				Providers: []domain.Provider{
					{ID: 1, Name: "alpha", Position: 1},
				},
			},
		},
	}
	service := NewService(repo)

	saved, err := service.SaveProfile(context.Background(), domain.RouteProfile{
		Platform:          kernel.PlatformClaude,
		DefaultProviderID: intPtr(1),
		Providers: []domain.Provider{
			{ID: 1, Name: "alpha", Position: 1, APIURL: "https://example.com", APIKey: "secret"},
		},
	})
	if err != nil {
		t.Fatalf("save profile: %v", err)
	}
	if repo.saveProfileCalls != 1 {
		t.Fatalf("expected save to hit repo once, got %d", repo.saveProfileCalls)
	}
	if repo.getProfileCalls != 1 {
		t.Fatalf("expected save validation to read repo once, got %d", repo.getProfileCalls)
	}

	profile, err := service.GetProfile(context.Background(), kernel.PlatformClaude.String())
	if err != nil {
		t.Fatalf("get cached saved profile: %v", err)
	}
	if repo.getProfileCalls != 1 {
		t.Fatalf("expected cached read after save, got %d repo reads", repo.getProfileCalls)
	}
	if profile.Providers[0].APIURL != saved.Providers[0].APIURL {
		t.Fatalf("expected cache to be refreshed with saved profile, got %+v", profile.Providers[0])
	}
}

func TestServiceExportProviderConfigIncludesAllProfiles(t *testing.T) {
	repo := &routingRepoStub{
		profiles: map[kernel.Platform]domain.RouteProfile{
			kernel.PlatformClaude: testProfile(kernel.PlatformClaude, 1, "claude-provider"),
			kernel.PlatformCodex:  testProfile(kernel.PlatformCodex, 2, "codex-provider"),
			kernel.PlatformGemini: testProfile(kernel.PlatformGemini, 3, "gemini-provider"),
		},
	}
	repo.profiles[kernel.PlatformClaude].Providers[0].APIKey = "secret"
	repo.profiles[kernel.PlatformClaude].Providers[0].SupportedModels = map[string]bool{"claude-*": true}
	repo.profiles[kernel.PlatformClaude].Providers[0].ModelMapping = map[string]string{"claude-*": "anthropic/claude-*"}
	repo.profiles[kernel.PlatformClaude].Providers[0].ProxyMode = domain.ProxyModeCustom
	repo.profiles[kernel.PlatformClaude].Providers[0].ProxyURL = "socks5://127.0.0.1:1080"
	service := NewService(repo)

	bundle, err := service.ExportProviderConfig(context.Background())
	if err != nil {
		t.Fatalf("ExportProviderConfig: %v", err)
	}

	if bundle.Format != domain.ProviderConfigFormat {
		t.Fatalf("unexpected format: %s", bundle.Format)
	}
	if bundle.Version != domain.ProviderConfigVersion {
		t.Fatalf("unexpected version: %d", bundle.Version)
	}
	if bundle.ExportedAt == "" {
		t.Fatal("expected exportedAt to be set")
	}
	if len(bundle.Profiles) != 3 {
		t.Fatalf("expected 3 profiles, got %d", len(bundle.Profiles))
	}

	claudeProvider := bundle.Profiles[0].Providers[0]
	if claudeProvider.APIKey != "secret" {
		t.Fatalf("expected API key to be exported, got %q", claudeProvider.APIKey)
	}
	if !claudeProvider.SupportedModels["claude-*"] {
		t.Fatalf("expected supported models to be exported, got %+v", claudeProvider.SupportedModels)
	}
	if claudeProvider.ModelMapping["claude-*"] != "anthropic/claude-*" {
		t.Fatalf("expected model mapping to be exported, got %+v", claudeProvider.ModelMapping)
	}
	if claudeProvider.ProxyMode != domain.ProxyModeCustom || claudeProvider.ProxyURL == "" {
		t.Fatalf("expected proxy config to be exported, got %+v", claudeProvider)
	}
}

func TestServiceImportProviderConfigOverwritesProfilesAndAllowsRenamingIDs(t *testing.T) {
	repo := &routingRepoStub{
		profiles: map[kernel.Platform]domain.RouteProfile{
			kernel.PlatformClaude: testProfile(kernel.PlatformClaude, 1, "old-claude"),
			kernel.PlatformCodex:  testProfile(kernel.PlatformCodex, 2, "old-codex"),
			kernel.PlatformGemini: testProfile(kernel.PlatformGemini, 3, "old-gemini"),
		},
	}
	service := NewService(repo)
	if _, err := service.GetProfile(context.Background(), kernel.PlatformClaude.String()); err != nil {
		t.Fatalf("prime cache: %v", err)
	}

	bundle := validProviderConfigBundle()
	bundle.Profiles[0].Providers[0].ID = 1
	bundle.Profiles[0].Providers[0].Name = "renamed-claude"
	bundle.Profiles[0].Providers[0].APIKey = "restored-key"
	bundle.Profiles[0].DefaultProviderID = intPtr(1)

	imported, err := service.ImportProviderConfig(context.Background(), bundle)
	if err != nil {
		t.Fatalf("ImportProviderConfig: %v", err)
	}

	if repo.saveProfileCalls != 3 {
		t.Fatalf("expected 3 saved profiles, got %d", repo.saveProfileCalls)
	}
	if imported.Profiles[0].Providers[0].Name != "renamed-claude" {
		t.Fatalf("expected imported profile in response, got %+v", imported.Profiles[0].Providers[0])
	}

	profile, err := service.GetProfile(context.Background(), kernel.PlatformClaude.String())
	if err != nil {
		t.Fatalf("GetProfile after import: %v", err)
	}
	if profile.Providers[0].Name != "renamed-claude" || profile.Providers[0].APIKey != "restored-key" {
		t.Fatalf("expected cache to reflect imported profile, got %+v", profile.Providers[0])
	}
}

func TestServiceImportProviderConfigRejectsInvalidBundles(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*domain.ProviderConfigBundle)
		wantErr string
	}{
		{
			name: "unknown format",
			mutate: func(bundle *domain.ProviderConfigBundle) {
				bundle.Format = "other"
			},
			wantErr: "unsupported provider config format",
		},
		{
			name: "unknown version",
			mutate: func(bundle *domain.ProviderConfigBundle) {
				bundle.Version = 99
			},
			wantErr: "unsupported provider config version",
		},
		{
			name: "missing platform",
			mutate: func(bundle *domain.ProviderConfigBundle) {
				bundle.Profiles = bundle.Profiles[:2]
			},
			wantErr: "missing platform profile: gemini",
		},
		{
			name: "duplicate platform",
			mutate: func(bundle *domain.ProviderConfigBundle) {
				bundle.Profiles[1].Platform = kernel.PlatformClaude
			},
			wantErr: "duplicate platform profile: claude",
		},
		{
			name: "unknown platform",
			mutate: func(bundle *domain.ProviderConfigBundle) {
				bundle.Profiles[0].Platform = kernel.Platform("unknown")
			},
			wantErr: "unknown platform: unknown",
		},
		{
			name: "default provider missing",
			mutate: func(bundle *domain.ProviderConfigBundle) {
				bundle.Profiles[0].DefaultProviderID = intPtr(999)
			},
			wantErr: "default provider 999 not found",
		},
		{
			name: "duplicate provider name",
			mutate: func(bundle *domain.ProviderConfigBundle) {
				bundle.Profiles[0].Providers = append(bundle.Profiles[0].Providers, domain.Provider{
					ID:       99,
					Name:     bundle.Profiles[0].Providers[0].Name,
					Position: 2,
				})
			},
			wantErr: "duplicate provider name",
		},
		{
			name: "invalid model mapping",
			mutate: func(bundle *domain.ProviderConfigBundle) {
				bundle.Profiles[0].Providers[0].SupportedModels = map[string]bool{"allowed-model": true}
				bundle.Profiles[0].Providers[0].ModelMapping = map[string]string{"external-model": "missing-model"}
			},
			wantErr: "模型映射无效",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &routingRepoStub{profiles: map[kernel.Platform]domain.RouteProfile{}}
			service := NewService(repo)
			bundle := validProviderConfigBundle()
			tt.mutate(&bundle)

			_, err := service.ImportProviderConfig(context.Background(), bundle)
			if err == nil {
				t.Fatal("expected import to fail")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error containing %q, got %q", tt.wantErr, err.Error())
			}
			if repo.saveProfileCalls != 0 {
				t.Fatalf("invalid bundle should not save profiles, got %d saves", repo.saveProfileCalls)
			}
		})
	}
}

func testProfile(platform kernel.Platform, providerID int, name string) domain.RouteProfile {
	return domain.RouteProfile{
		Platform:          platform,
		DefaultProviderID: intPtr(providerID),
		Providers: []domain.Provider{
			{
				ID:           providerID,
				Name:         name,
				APIURL:       "https://api.example.com",
				APIKey:       "key",
				OfficialSite: "https://example.com",
				Icon:         "aicoding",
				Tint:         "rgba(10, 132, 255, 0.14)",
				Accent:       "#0a84ff",
				Position:     1,
			},
		},
	}
}

func validProviderConfigBundle() domain.ProviderConfigBundle {
	return domain.ProviderConfigBundle{
		Format:     domain.ProviderConfigFormat,
		Version:    domain.ProviderConfigVersion,
		ExportedAt: "2026-05-06T00:00:00Z",
		Profiles: []domain.RouteProfile{
			testProfile(kernel.PlatformClaude, 10, "imported-claude"),
			testProfile(kernel.PlatformCodex, 20, "imported-codex"),
			testProfile(kernel.PlatformGemini, 30, "imported-gemini"),
		},
	}
}

func intPtr(value int) *int {
	return &value
}
