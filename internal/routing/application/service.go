package application

import (
	"context"
	"fmt"
	"sync"
	"time"

	"codeswitch/internal/routing/domain"
	"codeswitch/internal/shared/kernel"
)

type Repository interface {
	GetProfile(ctx context.Context, platform kernel.Platform) (domain.RouteProfile, error)
	SaveProfile(ctx context.Context, profile domain.RouteProfile) (domain.RouteProfile, error)
	GetAppPreferences(ctx context.Context) (domain.AppPreferences, error)
	SaveAppPreferences(ctx context.Context, preferences domain.AppPreferences) (domain.AppPreferences, error)
}

type Service struct {
	repo Repository

	profileCacheMu sync.RWMutex
	profileCache   map[kernel.Platform]domain.RouteProfile
}

func NewService(repo Repository) *Service {
	return &Service{
		repo:         repo,
		profileCache: make(map[kernel.Platform]domain.RouteProfile),
	}
}

func (s *Service) GetProfile(ctx context.Context, platform string) (domain.RouteProfile, error) {
	parsed, err := kernel.ParsePlatform(platform)
	if err != nil {
		return domain.RouteProfile{}, err
	}
	return s.getProfile(ctx, parsed)
}

func (s *Service) SaveProfile(ctx context.Context, profile domain.RouteProfile) (domain.RouteProfile, error) {
	profile = profile.Normalize()
	if err := profile.Validate(); err != nil {
		return domain.RouteProfile{}, err
	}
	current, err := s.getProfile(ctx, profile.Platform)
	if err != nil {
		return domain.RouteProfile{}, err
	}
	originalNames := make(map[int]string, len(current.Providers))
	for _, provider := range current.Providers {
		originalNames[provider.ID] = provider.Name
	}
	for _, provider := range profile.Providers {
		if oldName, ok := originalNames[provider.ID]; ok && oldName != provider.Name {
			return domain.RouteProfile{}, fmt.Errorf("provider id %d 的 name 不可修改", provider.ID)
		}
	}
	saved, err := s.repo.SaveProfile(ctx, profile)
	if err != nil {
		return domain.RouteProfile{}, err
	}
	s.storeProfileCache(saved)
	return saved, nil
}

func (s *Service) ExportProviderConfig(ctx context.Context) (domain.ProviderConfigBundle, error) {
	profiles := make([]domain.RouteProfile, 0, len(kernel.AllPlatforms()))
	for _, platform := range kernel.AllPlatforms() {
		profile, err := s.getProfile(ctx, platform)
		if err != nil {
			return domain.ProviderConfigBundle{}, err
		}
		profiles = append(profiles, profile.Normalize())
	}
	return domain.ProviderConfigBundle{
		Format:     domain.ProviderConfigFormat,
		Version:    domain.ProviderConfigVersion,
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Profiles:   profiles,
	}, nil
}

func (s *Service) ImportProviderConfig(ctx context.Context, bundle domain.ProviderConfigBundle) (domain.ProviderConfigBundle, error) {
	profiles, err := validateProviderConfigBundle(bundle)
	if err != nil {
		return domain.ProviderConfigBundle{}, err
	}

	savedProfiles := make([]domain.RouteProfile, 0, len(profiles))
	for _, profile := range profiles {
		saved, err := s.repo.SaveProfile(ctx, profile)
		if err != nil {
			return domain.ProviderConfigBundle{}, err
		}
		s.storeProfileCache(saved)
		savedProfiles = append(savedProfiles, saved.Normalize())
	}

	return domain.ProviderConfigBundle{
		Format:     domain.ProviderConfigFormat,
		Version:    domain.ProviderConfigVersion,
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Profiles:   savedProfiles,
	}, nil
}

func (s *Service) GetAppPreferences(ctx context.Context) (domain.AppPreferences, error) {
	return s.repo.GetAppPreferences(ctx)
}

func (s *Service) SaveAppPreferences(ctx context.Context, preferences domain.AppPreferences) (domain.AppPreferences, error) {
	return s.repo.SaveAppPreferences(ctx, preferences)
}

func (s *Service) getProfile(ctx context.Context, platform kernel.Platform) (domain.RouteProfile, error) {
	if cached, ok := s.loadProfileCache(platform); ok {
		return cached, nil
	}
	profile, err := s.repo.GetProfile(ctx, platform)
	if err != nil {
		return domain.RouteProfile{}, err
	}
	s.storeProfileCache(profile)
	return cloneRouteProfile(profile), nil
}

func (s *Service) loadProfileCache(platform kernel.Platform) (domain.RouteProfile, bool) {
	s.profileCacheMu.RLock()
	defer s.profileCacheMu.RUnlock()

	profile, ok := s.profileCache[platform]
	if !ok {
		return domain.RouteProfile{}, false
	}
	return cloneRouteProfile(profile), true
}

func (s *Service) storeProfileCache(profile domain.RouteProfile) {
	s.profileCacheMu.Lock()
	defer s.profileCacheMu.Unlock()
	s.profileCache[profile.Platform] = cloneRouteProfile(profile.Normalize())
}

func cloneRouteProfile(profile domain.RouteProfile) domain.RouteProfile {
	cloned := domain.RouteProfile{
		Platform:  profile.Platform,
		Providers: make([]domain.Provider, 0, len(profile.Providers)),
	}
	if profile.DefaultProviderID != nil {
		value := *profile.DefaultProviderID
		cloned.DefaultProviderID = &value
	}
	for _, provider := range profile.Providers {
		next := provider
		next.SupportedModels = cloneBoolMap(provider.SupportedModels)
		next.ModelMapping = cloneStringMap(provider.ModelMapping)
		cloned.Providers = append(cloned.Providers, next)
	}
	return cloned
}

func cloneBoolMap(input map[string]bool) map[string]bool {
	if input == nil {
		return map[string]bool{}
	}
	cloned := make(map[string]bool, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func cloneStringMap(input map[string]string) map[string]string {
	if input == nil {
		return map[string]string{}
	}
	cloned := make(map[string]string, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func validateProviderConfigBundle(bundle domain.ProviderConfigBundle) ([]domain.RouteProfile, error) {
	if bundle.Format != domain.ProviderConfigFormat {
		return nil, fmt.Errorf("unsupported provider config format: %s", bundle.Format)
	}
	if bundle.Version != domain.ProviderConfigVersion {
		return nil, fmt.Errorf("unsupported provider config version: %d", bundle.Version)
	}

	byPlatform := make(map[kernel.Platform]domain.RouteProfile, len(bundle.Profiles))
	for _, profile := range bundle.Profiles {
		platform, err := kernel.ParsePlatform(profile.Platform.String())
		if err != nil {
			return nil, err
		}
		if _, exists := byPlatform[platform]; exists {
			return nil, fmt.Errorf("duplicate platform profile: %s", platform)
		}
		profile.Platform = platform
		profile = profile.Normalize()
		if err := profile.Validate(); err != nil {
			return nil, err
		}
		byPlatform[platform] = profile
	}

	profiles := make([]domain.RouteProfile, 0, len(kernel.AllPlatforms()))
	for _, platform := range kernel.AllPlatforms() {
		profile, ok := byPlatform[platform]
		if !ok {
			return nil, fmt.Errorf("missing platform profile: %s", platform)
		}
		profiles = append(profiles, profile)
	}
	return profiles, nil
}
