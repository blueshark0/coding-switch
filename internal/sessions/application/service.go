package application

import "codeswitch/services"

type Repository interface {
	GetPlatformSessions(platform string) ([]services.SessionBinding, error)
	UnbindSession(platform, sessionID string) error
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListByPlatform(platform string) ([]services.SessionBinding, error) {
	return s.repo.GetPlatformSessions(platform)
}

func (s *Service) Unbind(platform, sessionID string) error {
	return s.repo.UnbindSession(platform, sessionID)
}
