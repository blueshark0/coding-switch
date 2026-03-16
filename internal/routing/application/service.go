package application

import (
	"context"
	"fmt"

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
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetProfile(ctx context.Context, platform string) (domain.RouteProfile, error) {
	parsed, err := kernel.ParsePlatform(platform)
	if err != nil {
		return domain.RouteProfile{}, err
	}
	return s.repo.GetProfile(ctx, parsed)
}

func (s *Service) SaveProfile(ctx context.Context, profile domain.RouteProfile) (domain.RouteProfile, error) {
	profile = profile.Normalize()
	if err := profile.Validate(); err != nil {
		return domain.RouteProfile{}, err
	}
	current, err := s.repo.GetProfile(ctx, profile.Platform)
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
	return s.repo.SaveProfile(ctx, profile)
}

func (s *Service) GetAppPreferences(ctx context.Context) (domain.AppPreferences, error) {
	return s.repo.GetAppPreferences(ctx)
}

func (s *Service) SaveAppPreferences(ctx context.Context, preferences domain.AppPreferences) (domain.AppPreferences, error) {
	return s.repo.SaveAppPreferences(ctx, preferences)
}
