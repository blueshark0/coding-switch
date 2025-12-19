package services

import "fmt"

// validateConfig 验证所有 provider 的配置
// 返回警告列表（非阻塞性错误）
func (prs *ProviderRelayService) validateConfig() []string {
	warnings := make([]string, 0)

	for _, platform := range AllPlatforms() {
		kind := platform.String()
		providers, err := prs.providerService.LoadProviders(kind)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("[%s] 加载配置失败: %v", kind, err))
			continue
		}

		enabledCount := 0
		for _, p := range providers {
			if !p.Enabled {
				continue
			}
			enabledCount++

			// 验证每个启用的 provider
			if errs := p.ValidateConfiguration(); len(errs) > 0 {
				for _, errMsg := range errs {
					warnings = append(warnings, fmt.Sprintf("[%s/%s] %s", kind, p.Name, errMsg))
				}
			}

		}

		if enabledCount == 0 {
			warnings = append(warnings, fmt.Sprintf("[%s] 没有启用的 provider", kind))
		}
	}

	return warnings
}

// loadConfig 加载应用设置和 providers
func (prs *ProviderRelayService) loadConfig(kind string) (AppSettings, []Provider, error) {
	appSettings, err := prs.appSettingsService.GetAppSettings()
	if err != nil {
		return AppSettings{}, nil, fmt.Errorf("failed to load app settings: %w", err)
	}

	providers, err := prs.providerService.LoadProviders(kind)
	if err != nil {
		return AppSettings{}, nil, fmt.Errorf("failed to load providers: %w", err)
	}

	return appSettings, providers, nil
}
