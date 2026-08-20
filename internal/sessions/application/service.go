package application

import (
	"fmt"
	"sync"

	sessiondomain "codeswitch/internal/sessions/domain"
)

type Repository interface {
	GetSessionProvider(platform, sessionID string) (string, error)
	BindSessionToProvider(platform, sessionID, providerName string) error
	UpdateSessionSuccess(platform, sessionID string) error
	GetProviderSessions(platform, providerName string) ([]sessiondomain.SessionBinding, error)
	GetPlatformSessions(platform string) ([]sessiondomain.SessionBinding, error)
	UnbindSession(platform, sessionID string) error
	CleanExpiredSessions() error
}

type CacheInvalidator interface {
	InvalidateSession(platform, sessionID string)
}

type Service struct {
	repo Repository

	cachesMu sync.Mutex
	caches   []CacheInvalidator
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetSessionProvider(platform, sessionID string) (string, error) {
	return s.repo.GetSessionProvider(platform, sessionID)
}

func (s *Service) BindSessionToProvider(platform, sessionID, providerName string) error {
	return s.repo.BindSessionToProvider(platform, sessionID, providerName)
}

func (s *Service) UpdateSessionSuccess(platform, sessionID string) error {
	return s.repo.UpdateSessionSuccess(platform, sessionID)
}

func (s *Service) ListByPlatform(platform string) ([]sessiondomain.SessionBinding, error) {
	return s.repo.GetPlatformSessions(platform)
}

func (s *Service) ListByProvider(platform, providerName string) ([]sessiondomain.SessionBinding, error) {
	return s.repo.GetProviderSessions(platform, providerName)
}

func (s *Service) Unbind(platform, sessionID string) error {
	if platform == "" || sessionID == "" {
		return fmt.Errorf("平台和会话ID不能为空")
	}
	if err := s.repo.UnbindSession(platform, sessionID); err != nil {
		return err
	}
	s.invalidateCaches(platform, sessionID)
	return nil
}

func (s *Service) CleanExpiredSessions() error {
	return s.repo.CleanExpiredSessions()
}

func (s *Service) RegisterCache(cache CacheInvalidator) {
	if cache == nil {
		return
	}
	s.cachesMu.Lock()
	defer s.cachesMu.Unlock()
	s.caches = append(s.caches, cache)
}

func (s *Service) invalidateCaches(platform, sessionID string) {
	s.cachesMu.Lock()
	defer s.cachesMu.Unlock()
	for _, cache := range s.caches {
		if cache != nil {
			cache.InvalidateSession(platform, sessionID)
		}
	}
}
