package wails

import (
	hotkeyapp "codeswitch/internal/hotkeys/application"
	hotkeydomain "codeswitch/internal/hotkeys/domain"
)

type HotkeyFacade struct {
	service *hotkeyapp.Service
}

func NewHotkeyFacade(service *hotkeyapp.Service) *HotkeyFacade {
	return &HotkeyFacade{service: service}
}

func (f *HotkeyFacade) GetHotkeys() ([]hotkeydomain.Hotkey, error) {
	return f.service.GetHotkeys()
}

func (f *HotkeyFacade) UpHotkey(id int, key int, modifier int) error {
	return f.service.UpHotkey(id, key, modifier)
}

func (f *HotkeyFacade) Close() error {
	return f.service.Close()
}
