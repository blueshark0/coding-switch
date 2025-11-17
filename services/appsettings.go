package services

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

const (
	appSettingsDir  = ".codex-swtich"
	appSettingsFile = "app.json"
)

type AppSettings struct {
	ShowHeatmap            bool   `json:"show_heatmap"`
	ShowHomeTitle          bool   `json:"show_home_title"`
	AutoStart              bool   `json:"auto_start"`
	EnableProviderFallback bool   `json:"enable_provider_fallback"`
	RoutingMode            string `json:"routing_mode"`              // "auto" 或 "manual"
	DefaultClaudeProvider  string `json:"default_claude_provider"`   // Claude 默认供应商名称
	DefaultCodexProvider   string `json:"default_codex_provider"`    // Codex 默认供应商名称
}

type AppSettingsService struct {
	path             string
	mu               sync.Mutex
	autoStartService *AutoStartService
}

func NewAppSettingsService(autoStartService *AutoStartService) *AppSettingsService {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	path := filepath.Join(home, appSettingsDir, appSettingsFile)
	return &AppSettingsService{
		path:             path,
		autoStartService: autoStartService,
	}
}

func (as *AppSettingsService) defaultSettings() AppSettings {
	// 检查当前开机自启动状态
	autoStartEnabled := false
	if as.autoStartService != nil {
		if enabled, err := as.autoStartService.IsEnabled(); err == nil {
			autoStartEnabled = enabled
		}
	}

	return AppSettings{
		ShowHeatmap:            true,
		ShowHomeTitle:          true,
		AutoStart:              autoStartEnabled,
		EnableProviderFallback: true,
		RoutingMode:            "auto",  // 默认使用自动路由模式
		DefaultClaudeProvider:  "",      // 默认无指定供应商
		DefaultCodexProvider:   "",      // 默认无指定供应商
	}
}

// GetAppSettings returns the persisted app settings or defaults if the file does not exist.
func (as *AppSettingsService) GetAppSettings() (AppSettings, error) {
	as.mu.Lock()
	defer as.mu.Unlock()
	return as.loadLocked()
}

// SaveAppSettings persists the provided settings to disk.
func (as *AppSettingsService) SaveAppSettings(settings AppSettings) (AppSettings, error) {
	as.mu.Lock()
	defer as.mu.Unlock()

	// 同步开机自启动状态
	if as.autoStartService != nil {
		if settings.AutoStart {
			if err := as.autoStartService.Enable(); err != nil {
				return settings, err
			}
		} else {
			if err := as.autoStartService.Disable(); err != nil {
				return settings, err
			}
		}
	}

	if err := as.saveLocked(settings); err != nil {
		return settings, err
	}
	return settings, nil
}

func (as *AppSettingsService) loadLocked() (AppSettings, error) {
	settings := as.defaultSettings()
	data, err := os.ReadFile(as.path)
	if err != nil {
		if os.IsNotExist(err) {
			return settings, nil
		}
		return settings, err
	}
	if len(data) == 0 {
		return settings, nil
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		return settings, err
	}
	return settings, nil
}

func (as *AppSettingsService) saveLocked(settings AppSettings) error {
	dir := filepath.Dir(as.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(as.path, data, 0o644)
}

// ValidateDefaultProviders 验证默认供应商配置是否有效
// 返回错误列表（供应商不存在或未启用）
func (as *AppSettingsService) ValidateDefaultProviders(providerService *ProviderService, settings AppSettings) []string {
	errors := make([]string, 0)

	// 仅在手动路由模式下进行验证
	if settings.RoutingMode != "manual" {
		return errors
	}

	// 验证 Claude 默认供应商
	if settings.DefaultClaudeProvider != "" {
		providers, err := providerService.LoadProviders("claude")
		if err != nil {
			errors = append(errors, "无法加载 Claude 供应商配置")
		} else {
			found := false
			enabled := false
			for _, p := range providers {
				if p.Name == settings.DefaultClaudeProvider {
					found = true
					enabled = p.Enabled
					break
				}
			}
			if !found {
				errors = append(errors, "Claude 默认供应商不存在: "+settings.DefaultClaudeProvider)
			} else if !enabled {
				errors = append(errors, "Claude 默认供应商已被禁用: "+settings.DefaultClaudeProvider)
			}
		}
	}

	// 验证 Codex 默认供应商
	if settings.DefaultCodexProvider != "" {
		providers, err := providerService.LoadProviders("codex")
		if err != nil {
			errors = append(errors, "无法加载 Codex 供应商配置")
		} else {
			found := false
			enabled := false
			for _, p := range providers {
				if p.Name == settings.DefaultCodexProvider {
					found = true
					enabled = p.Enabled
					break
				}
			}
			if !found {
				errors = append(errors, "Codex 默认供应商不存在: "+settings.DefaultCodexProvider)
			} else if !enabled {
				errors = append(errors, "Codex 默认供应商已被禁用: "+settings.DefaultCodexProvider)
			}
		}
	}

	return errors
}
