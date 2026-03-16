package infrastructure

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	platformproxydomain "codeswitch/internal/platformproxy/domain"
	"github.com/pelletier/go-toml/v2"
)

const (
	codexSettingsDir      = ".codex"
	codexConfigFileName   = "config.toml"
	codexBackupConfigName = "cc-studio.back.config.toml"
	codexAuthFileName     = "auth.json"
	codexBackupAuthName   = "cc-studio.back.auth.json"
	codexProviderKey      = "code-switch"
	codexEnvKey           = "OPENAI_API_KEY"
	codexWireAPI          = "responses"
	codexTokenValue       = "code-switch"
)

type CodexManager struct {
	relayAddr string
}

func NewCodexManager(relayAddr string) *CodexManager {
	return &CodexManager{relayAddr: relayAddr}
}

func (m *CodexManager) ProxyStatus() (platformproxydomain.Status, error) {
	status := platformproxydomain.Status{Enabled: false, BaseURL: m.baseURL()}
	config, err := m.readConfig()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return status, nil
		}
		return status, err
	}
	provider, ok := config.ModelProviders[codexProviderKey]
	if !ok {
		return status, nil
	}
	baseURL := m.baseURL()
	if strings.EqualFold(config.ModelProvider, codexProviderKey) && strings.EqualFold(provider.BaseURL, baseURL) {
		status.Enabled = true
	}
	return status, nil
}

func (m *CodexManager) EnableProxy() error {
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
		if err := toml.Unmarshal(content, &raw); err != nil {
			return err
		}
	} else {
		raw = make(map[string]any)
	}
	if raw == nil {
		raw = make(map[string]any)
	}
	raw["model_provider"] = codexProviderKey
	modelProviders := make(map[string]map[string]any)
	provider := make(map[string]any)
	provider["name"] = codexProviderKey
	provider["base_url"] = m.baseURL()
	provider["wire_api"] = codexWireAPI
	provider["requires_openai_auth"] = false
	modelProviders[codexProviderKey] = provider
	raw["model_providers"] = modelProviders
	data, err := toml.Marshal(raw)
	if err != nil {
		return err
	}
	if err := os.WriteFile(settingsPath, stripModelProvidersHeader(data), 0o600); err != nil {
		return err
	}
	return m.writeAuthFile()
}

func (m *CodexManager) DisableProxy() error {
	settingsPath, backupPath, err := m.paths()
	if err != nil {
		return err
	}
	if err := os.Remove(settingsPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if _, err := os.Stat(backupPath); err == nil {
		if err := os.Rename(backupPath, settingsPath); err != nil {
			return err
		}
	}
	return m.restoreAuthFile()
}

func (m *CodexManager) readConfig() (*codexConfig, error) {
	settingsPath, _, err := m.paths()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return nil, err
	}
	var cfg codexConfig
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.ModelProviders == nil {
		cfg.ModelProviders = make(map[string]codexProvider)
	}
	return &cfg, nil
}

func (m *CodexManager) paths() (settingsPath string, backupPath string, err error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", err
	}
	dir := filepath.Join(home, codexSettingsDir)
	return filepath.Join(dir, codexConfigFileName), filepath.Join(dir, codexBackupConfigName), nil
}

func (m *CodexManager) authPaths() (string, string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", err
	}
	dir := filepath.Join(home, codexSettingsDir)
	return filepath.Join(dir, codexAuthFileName), filepath.Join(dir, codexBackupAuthName), nil
}

func (m *CodexManager) baseURL() string {
	return normalizeRelayBaseURL(m.relayAddr)
}

type codexConfig struct {
	ModelProvider  string                   `toml:"model_provider"`
	ModelProviders map[string]codexProvider `toml:"model_providers"`
}

type codexProvider struct {
	BaseURL string `toml:"base_url"`
}

func stripModelProvidersHeader(data []byte) []byte {
	lines := strings.Split(string(data), "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "[model_providers]" {
			continue
		}
		result = append(result, line)
	}
	return []byte(strings.Join(result, "\n"))
}

func (m *CodexManager) writeAuthFile() error {
	authPath, backupPath, err := m.authPaths()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(authPath), 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(authPath); err == nil {
		content, readErr := os.ReadFile(authPath)
		if readErr != nil {
			return readErr
		}
		if err := os.WriteFile(backupPath, content, 0o600); err != nil {
			return err
		}
	}
	payload := map[string]string{codexEnvKey: codexTokenValue}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(authPath, data, 0o600)
}

func (m *CodexManager) restoreAuthFile() error {
	authPath, backupPath, err := m.authPaths()
	if err != nil {
		return err
	}
	if err := os.Remove(authPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if _, err := os.Stat(backupPath); err == nil {
		return os.Rename(backupPath, authPath)
	}
	return nil
}
