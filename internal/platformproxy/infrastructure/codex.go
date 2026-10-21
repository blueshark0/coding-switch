package infrastructure

import (
	"encoding/json"
	"errors"
	"log"
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
	// 已处于代理状态时跳过备份：重复 Enable 会用已代理内容覆盖唯一备份，
	// 导致用户原始配置永久丢失。
	alreadyProxied := false
	if status, statusErr := m.ProxyStatus(); statusErr == nil {
		alreadyProxied = status.Enabled
	}
	var raw map[string]any
	if _, err := os.Stat(settingsPath); err == nil {
		content, readErr := os.ReadFile(settingsPath)
		if readErr != nil {
			return readErr
		}
		if !alreadyProxied {
			if err := os.WriteFile(backupPath, content, 0o600); err != nil {
				return err
			}
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
	// 合并而非整表覆盖 model_providers，保留用户自定义的其他 provider。
	modelProviders := make(map[string]any)
	if existing, ok := raw["model_providers"].(map[string]any); ok {
		modelProviders = existing
	}
	modelProviders[codexProviderKey] = map[string]any{
		"name":                 codexProviderKey,
		"base_url":             m.baseURL(),
		"wire_api":             codexWireAPI,
		"requires_openai_auth": false,
	}
	raw["model_providers"] = modelProviders
	data, err := toml.Marshal(raw)
	if err != nil {
		return err
	}
	if err := os.WriteFile(settingsPath, stripRedundantTableHeaders(data, "model_providers", "mcp_servers"), 0o600); err != nil {
		return err
	}
	return m.writeAuthFile(alreadyProxied)
}

func (m *CodexManager) DisableProxy() error {
	settingsPath, backupPath, err := m.paths()
	if err != nil {
		return err
	}
	// MCP 条目与模型代理配置同住 config.toml，而下方是整文件备份恢复——
	// 备份里的 MCP 状态与当前可能不一致（启用后用户可能注入过、也可能
	// 注入后又移除了）。MCP 条目当前的存在性是唯一事实来源：恢复备份后
	// 按它调和，两个方向都要处理（有则重注入，无则确保删除）。
	hadMcpEntry := false
	if present, statusErr := m.McpStatus(); statusErr == nil {
		hadMcpEntry = present
	}
	if err := os.Remove(settingsPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if _, err := os.Stat(backupPath); err == nil {
		if err := os.Rename(backupPath, settingsPath); err != nil {
			return err
		}
	}
	if hadMcpEntry {
		if err := m.InjectMcp(); err != nil {
			log.Printf("platformproxy: 恢复 codex MCP 条目失败: %v", err)
		}
	} else if err := m.RemoveMcp(); err != nil {
		log.Printf("platformproxy: 清理备份恢复的 codex MCP 条目失败: %v", err)
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

// stripRedundantTableHeaders 剔除 go-toml v2 序列化嵌套 map 时多产生的
// 空父表头（如裸的 [model_providers] / [mcp_servers] 行）。
func stripRedundantTableHeaders(data []byte, tables ...string) []byte {
	redundant := make(map[string]bool, len(tables))
	for _, table := range tables {
		redundant["["+table+"]"] = true
	}
	lines := strings.Split(string(data), "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		if redundant[strings.TrimSpace(line)] {
			continue
		}
		result = append(result, line)
	}
	return []byte(strings.Join(result, "\n"))
}

func (m *CodexManager) writeAuthFile(alreadyProxied bool) error {
	authPath, backupPath, err := m.authPaths()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(authPath), 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(authPath); err == nil && !alreadyProxied {
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
