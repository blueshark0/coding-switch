package services

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const (
	claudeSettingsDir      = ".claude"
	claudeSettingsFileName = "settings.json"
	claudeBackupFileName   = "cc-studio.back.settings.json"
	claudeAuthTokenValue   = "code-switch"
)

type ClaudeProxyStatus struct {
	Enabled bool   `json:"enabled"`
	BaseURL string `json:"base_url"`
}

type ClaudeSettingsService struct {
	relayAddr string
}

func NewClaudeSettingsService(relayAddr string) *ClaudeSettingsService {
	return &ClaudeSettingsService{relayAddr: relayAddr}
}

func (css *ClaudeSettingsService) ProxyStatus() (ClaudeProxyStatus, error) {
	status := ClaudeProxyStatus{Enabled: false, BaseURL: css.baseURL()}
	settingsPath, _, err := css.paths()
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
	baseURL := css.baseURL()
	enabled := strings.EqualFold(payload.Env["ANTHROPIC_AUTH_TOKEN"], claudeAuthTokenValue) &&
		strings.EqualFold(payload.Env["ANTHROPIC_BASE_URL"], baseURL)
	status.Enabled = enabled
	return status, nil
}

func (css *ClaudeSettingsService) EnableProxy() error {
	settingsPath, backupPath, err := css.paths()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		return err
	}

	// 读取并备份现有配置
	var raw map[string]any
	if _, err := os.Stat(settingsPath); err == nil {
		content, readErr := os.ReadFile(settingsPath)
		if readErr != nil {
			return readErr
		}
		// 备份原文件
		if err := os.WriteFile(backupPath, content, 0o600); err != nil {
			return err
		}
		// 解析现有配置
		if err := json.Unmarshal(content, &raw); err != nil {
			// 如果解析失败，创建新配置
			raw = make(map[string]any)
		}
	} else {
		// 文件不存在，创建新配置
		raw = make(map[string]any)
	}

	// 确保 raw 已初始化
	if raw == nil {
		raw = make(map[string]any)
	}

	// 获取或创建 env 字段
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

	// 只更新代理相关的配置，保留其他配置
	envMap["ANTHROPIC_AUTH_TOKEN"] = claudeAuthTokenValue
	envMap["ANTHROPIC_BASE_URL"] = css.baseURL()
	raw["env"] = envMap

	payload, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(settingsPath, payload, 0o600)
}

func (css *ClaudeSettingsService) DisableProxy() error {
	settingsPath, backupPath, err := css.paths()
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
	} else if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return nil
}

func (css *ClaudeSettingsService) paths() (settingsPath string, backupPath string, err error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", err
	}
	dir := filepath.Join(home, claudeSettingsDir)
	return filepath.Join(dir, claudeSettingsFileName), filepath.Join(dir, claudeBackupFileName), nil
}

func (css *ClaudeSettingsService) baseURL() string {
	addr := strings.TrimSpace(css.relayAddr)
	if addr == "" {
		addr = ":18100"
	}
	if strings.HasPrefix(addr, "http://") || strings.HasPrefix(addr, "https://") {
		return addr
	}
	host := addr
	if strings.HasPrefix(host, ":") {
		host = "127.0.0.1" + host
	}
	if !strings.Contains(host, "://") {
		host = "http://" + host
	}
	return host
}

type claudeSettingsFile struct {
	Env map[string]string `json:"env"`
}
