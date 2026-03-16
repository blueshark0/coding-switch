package application

import hotkeydomain "codeswitch/internal/hotkeys/domain"

type Repository interface {
	List() ([]hotkeydomain.Hotkey, error)
	Update(id int, key int, modifier int) error
	Close() error
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetHotkeys() ([]hotkeydomain.Hotkey, error) {
	return s.repo.List()
}

func (s *Service) UpHotkey(id int, key int, modifier int) error {
	return s.repo.Update(id, key, modifier)
}

func (s *Service) Close() error {
	return s.repo.Close()
}
