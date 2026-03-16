package domain

import (
	"fmt"
	"slices"
	"strings"

	"codeswitch/internal/shared/kernel"
)

type AppPreferences struct {
	ShowHeatmap   bool `json:"show_heatmap"`
	ShowHomeTitle bool `json:"show_home_title"`
}

func DefaultAppPreferences() AppPreferences {
	return AppPreferences{
		ShowHeatmap:   true,
		ShowHomeTitle: true,
	}
}

type Provider struct {
	ID              int               `json:"id"`
	Name            string            `json:"name"`
	APIURL          string            `json:"apiUrl"`
	APIKey          string            `json:"apiKey"`
	OfficialSite    string            `json:"officialSite"`
	Icon            string            `json:"icon"`
	Tint            string            `json:"tint"`
	Accent          string            `json:"accent"`
	Enabled         bool              `json:"enabled"`
	Position        int               `json:"position"`
	SupportedModels map[string]bool   `json:"supportedModels,omitempty"`
	ModelMapping    map[string]string `json:"modelMapping,omitempty"`
}

type RouteProfile struct {
	Platform          kernel.Platform `json:"platform"`
	DefaultProviderID *int            `json:"defaultProviderId"`
	Providers         []Provider      `json:"providers"`
}

func (p RouteProfile) Normalize() RouteProfile {
	cloned := RouteProfile{
		Platform:  p.Platform,
		Providers: make([]Provider, 0, len(p.Providers)),
	}
	if p.DefaultProviderID != nil {
		defaultID := *p.DefaultProviderID
		cloned.DefaultProviderID = &defaultID
	}
	for idx, provider := range p.Providers {
		next := provider
		if next.Position <= 0 {
			next.Position = idx + 1
		}
		if next.SupportedModels == nil {
			next.SupportedModels = map[string]bool{}
		}
		if next.ModelMapping == nil {
			next.ModelMapping = map[string]string{}
		}
		cloned.Providers = append(cloned.Providers, next)
	}
	slices.SortFunc(cloned.Providers, func(a, b Provider) int {
		switch {
		case a.Position < b.Position:
			return -1
		case a.Position > b.Position:
			return 1
		default:
			return 0
		}
	})
	for idx := range cloned.Providers {
		cloned.Providers[idx].Position = idx + 1
	}
	return cloned
}

func (p RouteProfile) Validate() error {
	if p.Platform == "" {
		return fmt.Errorf("platform is required")
	}
	seenIDs := make(map[int]struct{}, len(p.Providers))
	seenPositions := make(map[int]struct{}, len(p.Providers))
	var defaultProvider *Provider
	for _, provider := range p.Providers {
		if provider.ID == 0 {
			return fmt.Errorf("provider id is required")
		}
		if provider.Name == "" {
			return fmt.Errorf("provider name is required")
		}
		if _, ok := seenIDs[provider.ID]; ok {
			return fmt.Errorf("duplicate provider id: %d", provider.ID)
		}
		seenIDs[provider.ID] = struct{}{}
		if provider.Position <= 0 {
			return fmt.Errorf("provider %s has invalid position %d", provider.Name, provider.Position)
		}
		if _, ok := seenPositions[provider.Position]; ok {
			return fmt.Errorf("duplicate provider position: %d", provider.Position)
		}
		seenPositions[provider.Position] = struct{}{}
		if errs := provider.ValidateConfiguration(); len(errs) > 0 {
			return fmt.Errorf("[%s] %s", provider.Name, strings.Join(errs, "; "))
		}
		if p.DefaultProviderID != nil && provider.ID == *p.DefaultProviderID {
			defaultProvider = &provider
		}
	}
	if p.DefaultProviderID != nil {
		if defaultProvider == nil {
			return fmt.Errorf("default provider %d not found", *p.DefaultProviderID)
		}
		if !defaultProvider.Enabled {
			return fmt.Errorf("default provider %s is disabled", defaultProvider.Name)
		}
	}
	return nil
}

func (p RouteProfile) FindProviderByID(id int) *Provider {
	for idx := range p.Providers {
		if p.Providers[idx].ID == id {
			return &p.Providers[idx]
		}
	}
	return nil
}

func (p RouteProfile) FindProviderByName(name string) *Provider {
	for idx := range p.Providers {
		if p.Providers[idx].Name == name {
			return &p.Providers[idx]
		}
	}
	return nil
}

func (p RouteProfile) DefaultProvider() *Provider {
	if p.DefaultProviderID == nil {
		return nil
	}
	return p.FindProviderByID(*p.DefaultProviderID)
}

func (p *RouteProfile) SetDefaultProvider(id *int) error {
	if id == nil {
		p.DefaultProviderID = nil
		return nil
	}
	provider := p.FindProviderByID(*id)
	if provider == nil {
		return fmt.Errorf("provider %d not found", *id)
	}
	if !provider.Enabled {
		return fmt.Errorf("provider %s is disabled", provider.Name)
	}
	value := *id
	p.DefaultProviderID = &value
	return nil
}

func (p *Provider) IsModelSupported(modelName string) bool {
	if (p.SupportedModels == nil || len(p.SupportedModels) == 0) &&
		(p.ModelMapping == nil || len(p.ModelMapping) == 0) {
		return true
	}
	if p.SupportedModels != nil && p.SupportedModels[modelName] {
		return true
	}
	if p.SupportedModels != nil {
		for supportedModel := range p.SupportedModels {
			if matchWildcard(supportedModel, modelName) {
				return true
			}
		}
	}
	if p.ModelMapping != nil {
		if _, exists := p.ModelMapping[modelName]; exists {
			return true
		}
		for pattern := range p.ModelMapping {
			if matchWildcard(pattern, modelName) {
				return true
			}
		}
	}
	return false
}

func (p *Provider) GetEffectiveModel(requestedModel string) string {
	if p.ModelMapping == nil || len(p.ModelMapping) == 0 {
		return requestedModel
	}
	if mappedModel, exists := p.ModelMapping[requestedModel]; exists {
		return mappedModel
	}
	for pattern, replacement := range p.ModelMapping {
		if matchWildcard(pattern, requestedModel) {
			return applyWildcardMapping(pattern, replacement, requestedModel)
		}
	}
	return requestedModel
}

func (p *Provider) ValidateConfiguration() []string {
	errors := make([]string, 0)
	if p.ModelMapping != nil && len(p.ModelMapping) > 0 &&
		p.SupportedModels != nil && len(p.SupportedModels) > 0 {
		for externalModel, internalModel := range p.ModelMapping {
			if strings.Contains(internalModel, "*") {
				continue
			}
			supported := false
			if p.SupportedModels[internalModel] {
				supported = true
			} else {
				for supportedPattern := range p.SupportedModels {
					if matchWildcard(supportedPattern, internalModel) {
						supported = true
						break
					}
				}
			}
			if !supported {
				errors = append(errors, fmt.Sprintf(
					"模型映射无效：'%s' -> '%s'，目标模型 '%s' 不在 supportedModels 中",
					externalModel, internalModel, internalModel,
				))
			}
		}
	}
	return errors
}

func matchWildcard(pattern, text string) bool {
	if !strings.Contains(pattern, "*") {
		return pattern == text
	}
	parts := strings.Split(pattern, "*")
	if len(parts) == 2 {
		prefix, suffix := parts[0], parts[1]
		return strings.HasPrefix(text, prefix) && strings.HasSuffix(text, suffix)
	}
	return false
}

func applyWildcardMapping(pattern, replacement, input string) string {
	if !strings.Contains(pattern, "*") || !strings.Contains(replacement, "*") {
		return replacement
	}
	parts := strings.Split(pattern, "*")
	if len(parts) != 2 {
		return replacement
	}
	prefix, suffix := parts[0], parts[1]
	if !strings.HasPrefix(input, prefix) || !strings.HasSuffix(input, suffix) {
		return replacement
	}
	wildcardPart := input[len(prefix) : len(input)-len(suffix)]
	return strings.Replace(replacement, "*", wildcardPart, 1)
}
