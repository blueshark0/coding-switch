package application

import (
	"fmt"

	platformproxydomain "codeswitch/internal/platformproxy/domain"
	"codeswitch/internal/shared/kernel"
)

type Manager interface {
	ProxyStatus() (platformproxydomain.Status, error)
	EnableProxy() error
	DisableProxy() error
}

type Service struct {
	managers map[kernel.Platform]Manager
}

func NewService(managers map[kernel.Platform]Manager) *Service {
	return &Service{managers: managers}
}

func (s *Service) GetStatus(platform string) (platformproxydomain.Status, error) {
	manager, err := s.manager(platform)
	if err != nil {
		return platformproxydomain.Status{}, err
	}
	return manager.ProxyStatus()
}

func (s *Service) Enable(platform string) error {
	manager, err := s.manager(platform)
	if err != nil {
		return err
	}
	return manager.EnableProxy()
}

func (s *Service) Disable(platform string) error {
	manager, err := s.manager(platform)
	if err != nil {
		return err
	}
	return manager.DisableProxy()
}

func (s *Service) manager(platform string) (Manager, error) {
	parsed, err := kernel.ParsePlatform(platform)
	if err != nil {
		return nil, err
	}
	manager := s.managers[parsed]
	if manager == nil {
		return nil, fmt.Errorf("proxy manager not configured: %s", platform)
	}
	return manager, nil
}
