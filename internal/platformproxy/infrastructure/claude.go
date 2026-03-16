package infrastructure

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	platformproxydomain "codeswitch/internal/platformproxy/domain"
)

const (
	claudeSettingsDir      = ".claude"
	claudeSettingsFileName = "settings.json"
	claudeBackupFileName   = "cc-studio.back.settings.json"
	claudeAuthTokenValue   = "code-switch"
)

type ClaudeManager struct {
	relayAddr string
}

func NewClaudeManager(relayAddr string) *ClaudeManager {
	return &ClaudeManager{relayAddr: relayAddr}
}

func (m *ClaudeManager) ProxyStatus() (platformproxydomain.Status, error) {
	status := platformproxydomain.Status{Enabled: false, BaseURL: m.baseURL()}
	settingsPath, _, err := m.paths()
	if err != nil {
		return status, err
	}
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return status, nil
		}
		return status, err
	}
	var payload claudeSettingsFile
	if err := json.Unmarshal(data, &payload); err != nil {
		return status, nil
	}
	baseURL := m.baseURL()
	status.Enabled = strings.EqualFold(payload.Env["ANTHROPIC_AUTH_TOKEN"], claudeAuthTokenValue) &&
		strings.EqualFold(payload.Env["ANTHROPIC_BASE_URL"], baseURL)
	return status, nil
}

func (m *ClaudeManager) EnableProxy() error {
	settingsPath, backupPath, err := m.paths()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		return err
	}
	var raw map[string]any
	if _, err := os.Stat(settingsPath); err == nil {
		content, readErr := os.ReadFile(settingsPath)
		if readErr != nil {
			return readErr
		}
		if err := os.WriteFile(backupPath, content, 0o600); err != nil {
			return err
		}
		if err := json.Unmarshal(content, &raw); err != nil {
			raw = make(map[string]any)
		}
	} else {
		raw = make(map[string]any)
	}
	if raw == nil {
		raw = make(map[string]any)
	}
	var envMap map[string]any
	if envVal, ok := raw["env"]; ok {
		if envMapTyped, ok := envVal.(map[string]any); ok {
			envMap = envMapTyped
		} else {
			envMap = make(map[string]any)
		}
	} else {
		envMap = make(map[string]any)
	}
	envMap["ANTHROPIC_AUTH_TOKEN"] = claudeAuthTokenValue
	envMap["ANTHROPIC_BASE_URL"] = m.baseURL()
	raw["env"] = envMap
	payload, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(settingsPath, payload, 0o600)
}

func (m *ClaudeManager) DisableProxy() error {
	settingsPath, backupPath, err := m.paths()
	if err != nil {
		return err
	}
	if err := os.Remove(settingsPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if _, err := os.Stat(backupPath); err == nil {
		return os.Rename(backupPath, settingsPath)
	} else if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return nil
}

func (m *ClaudeManager) paths() (settingsPath string, backupPath string, err error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", err
	}
	dir := filepath.Join(home, claudeSettingsDir)
	return filepath.Join(dir, claudeSettingsFileName), filepath.Join(dir, claudeBackupFileName), nil
}

func (m *ClaudeManager) baseURL() string {
	return normalizeRelayBaseURL(m.relayAddr)
}

type claudeSettingsFile struct {
	Env map[string]string `json:"env"`
}
