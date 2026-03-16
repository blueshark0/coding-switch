package wails

import (
	sessionapp "codeswitch/internal/sessions/application"
	"codeswitch/services"
)

type SessionFacade struct {
	service *sessionapp.Service
}

func NewSessionFacade(service *sessionapp.Service) *SessionFacade {
	return &SessionFacade{service: service}
}

func (f *SessionFacade) ListByPlatform(platform string) ([]services.SessionBinding, error) {
	return f.service.ListByPlatform(platform)
}

func (f *SessionFacade) Unbind(platform, sessionID string) error {
	return f.service.Unbind(platform, sessionID)
}
