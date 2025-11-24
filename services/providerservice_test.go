package services

import (
	"strings"
	"testing"
)

// ==================== 通配符匹配测试 ====================

func TestMatchWildcard(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		text     string
		expected bool
	}{
		// 精确匹配
		{
			name:     "精确匹配-成功",
			pattern:  "claude-sonnet-4",
			text:     "claude-sonnet-4",
			expected: true,
		},
		{
			name:     "精确匹配-失败",
			pattern:  "claude-sonnet-4",
			text:     "claude-opus-4",
			expected: false,
		},

		// 前缀通配符
		{
			name:     "前缀通配符-成功",
			pattern:  "claude-*",
			text:     "claude-sonnet-4",
			expected: true,
		},
		{
			name:     "前缀通配符-多段匹配",
			pattern:  "claude-*",
			text:     "claude-sonnet-4-latest",
			expected: true,
		},
		{
			name:     "前缀通配符-失败",
			pattern:  "claude-*",
			text:     "gpt-4",
			expected: false,
		},

		// 后缀通配符
		{
			name:     "后缀通配符-成功",
			pattern:  "*-4",
			text:     "claude-sonnet-4",
			expected: true,
		},
		{
			name:     "后缀通配符-失败",
			pattern:  "*-4",
			text:     "claude-sonnet-3.5",
			expected: false,
		},

		// 中间通配符
		{
			name:     "中间通配符-成功",
			pattern:  "claude-*-4",
			text:     "claude-sonnet-4",
			expected: true,
		},
		{
			name:     "中间通配符-多段匹配",
			pattern:  "claude-*-4",
			text:     "claude-opus-mini-4",
			expected: true,
		},
		{
			name:     "中间通配符-失败前缀",
			pattern:  "claude-*-4",
			text:     "gpt-sonnet-4",
			expected: false,
		},
		{
			name:     "中间通配符-失败后缀",
			pattern:  "claude-*-4",
			text:     "claude-sonnet-3",
			expected: false,
		},

		// 边界情况
		{
			name:     "空前缀",
			pattern:  "*-sonnet",
			text:     "claude-sonnet",
			expected: true,
		},
		{
			name:     "空后缀",
			pattern:  "claude-*",
			text:     "claude-",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchWildcard(tt.pattern, tt.text)
			if result != tt.expected {
				t.Errorf("matchWildcard(%q, %q) = %v, 期望 %v",
					tt.pattern, tt.text, result, tt.expected)
			}
		})
	}
}

// ==================== 通配符映射应用测试 ====================

func TestApplyWildcardMapping(t *testing.T) {
	tests := []struct {
		name        string
		pattern     string
		replacement string
		input       string
		expected    string
	}{
		// 前缀通配符映射
		{
			name:        "前缀通配符映射",
			pattern:     "claude-*",
			replacement: "anthropic/claude-*",
			input:       "claude-sonnet-4",
			expected:    "anthropic/claude-sonnet-4",
		},
		{
			name:        "前缀通配符映射-多段",
			pattern:     "claude-*",
			replacement: "anthropic/claude-*",
			input:       "claude-opus-4-latest",
			expected:    "anthropic/claude-opus-4-latest",
		},

		// 中间通配符映射
		{
			name:        "中间通配符映射",
			pattern:     "claude-*-4",
			replacement: "anthropic/claude-*-v4",
			input:       "claude-sonnet-4",
			expected:    "anthropic/claude-sonnet-v4",
		},

		// 无通配符（直接返回 replacement）
		{
			name:        "无通配符-pattern",
			pattern:     "claude-sonnet-4",
			replacement: "anthropic/claude-sonnet-4",
			input:       "claude-sonnet-4",
			expected:    "anthropic/claude-sonnet-4",
		},
		{
			name:        "无通配符-replacement",
			pattern:     "claude-*",
			replacement: "fixed-model",
			input:       "claude-sonnet-4",
			expected:    "fixed-model",
		},

		// 边界情况
		{
			name:        "空匹配部分",
			pattern:     "claude-*",
			replacement: "anthropic/claude-*",
			input:       "claude-",
			expected:    "anthropic/claude-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := applyWildcardMapping(tt.pattern, tt.replacement, tt.input)
			if result != tt.expected {
				t.Errorf("applyWildcardMapping(%q, %q, %q) = %q, 期望 %q",
					tt.pattern, tt.replacement, tt.input, result, tt.expected)
			}
		})
	}
}

// ==================== IsModelSupported 测试 ====================

func TestProvider_IsModelSupported(t *testing.T) {
	tests := []struct {
		name      string
		provider  Provider
		modelName string
		expected  bool
	}{
		// 向后兼容：未配置白名单和映射
		{
			name:      "向后兼容-未配置",
			provider:  Provider{},
			modelName: "any-model",
			expected:  true,
		},

		// 场景 A：原生支持（精确匹配）
		{
			name: "原生支持-精确匹配-成功",
			provider: Provider{
				SupportedModels: map[string]bool{
					"claude-sonnet-4": true,
					"claude-opus-4":   true,
				},
			},
			modelName: "claude-sonnet-4",
			expected:  true,
		},
		{
			name: "原生支持-精确匹配-失败",
			provider: Provider{
				SupportedModels: map[string]bool{
					"claude-sonnet-4": true,
				},
			},
			modelName: "gpt-4",
			expected:  false,
		},

		// 场景 A+：原生支持（通配符匹配）
		{
			name: "原生支持-通配符匹配-成功",
			provider: Provider{
				SupportedModels: map[string]bool{
					"claude-*": true,
				},
			},
			modelName: "claude-sonnet-4",
			expected:  true,
		},
		{
			name: "原生支持-通配符匹配-失败",
			provider: Provider{
				SupportedModels: map[string]bool{
					"claude-*": true,
				},
			},
			modelName: "gpt-4",
			expected:  false,
		},

		// 场景 B：映射支持（精确匹配）
		{
			name: "映射支持-精确匹配-成功",
			provider: Provider{
				SupportedModels: map[string]bool{
					"anthropic/claude-sonnet-4": true,
				},
				ModelMapping: map[string]string{
					"claude-sonnet-4": "anthropic/claude-sonnet-4",
				},
			},
			modelName: "claude-sonnet-4",
			expected:  true,
		},

		// 场景 B+：映射支持（通配符匹配）
		{
			name: "映射支持-通配符匹配-成功",
			provider: Provider{
				SupportedModels: map[string]bool{
					"anthropic/claude-*": true,
				},
				ModelMapping: map[string]string{
					"claude-*": "anthropic/claude-*",
				},
			},
			modelName: "claude-sonnet-4",
			expected:  true,
		},

		// 混合模式
		{
			name: "混合模式-原生+映射",
			provider: Provider{
				SupportedModels: map[string]bool{
					"native-model":    true,
					"vendor/external": true,
				},
				ModelMapping: map[string]string{
					"external": "vendor/external",
				},
			},
			modelName: "external",
			expected:  true,
		},
		{
			name: "混合模式-只在原生",
			provider: Provider{
				SupportedModels: map[string]bool{
					"native-model": true,
				},
				ModelMapping: map[string]string{
					"external": "vendor/external",
				},
			},
			modelName: "native-model",
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.provider.IsModelSupported(tt.modelName)
			if result != tt.expected {
				t.Errorf("IsModelSupported(%q) = %v, 期望 %v",
					tt.modelName, result, tt.expected)
			}
		})
	}
}

// ==================== GetEffectiveModel 测试 ====================

func TestProvider_GetEffectiveModel(t *testing.T) {
	tests := []struct {
		name           string
		provider       Provider
		requestedModel string
		expected       string
	}{
		// 无映射
		{
			name:           "无映射-返回原名",
			provider:       Provider{},
			requestedModel: "claude-sonnet-4",
			expected:       "claude-sonnet-4",
		},

		// 精确映射
		{
			name: "精确映射-成功",
			provider: Provider{
				ModelMapping: map[string]string{
					"claude-sonnet-4": "anthropic/claude-sonnet-4",
				},
			},
			requestedModel: "claude-sonnet-4",
			expected:       "anthropic/claude-sonnet-4",
		},
		{
			name: "精确映射-无匹配",
			provider: Provider{
				ModelMapping: map[string]string{
					"claude-sonnet-4": "anthropic/claude-sonnet-4",
				},
			},
			requestedModel: "gpt-4",
			expected:       "gpt-4",
		},

		// 通配符映射
		{
			name: "通配符映射-前缀",
			provider: Provider{
				ModelMapping: map[string]string{
					"claude-*": "anthropic/claude-*",
				},
			},
			requestedModel: "claude-sonnet-4",
			expected:       "anthropic/claude-sonnet-4",
		},
		{
			name: "通配符映射-中间",
			provider: Provider{
				ModelMapping: map[string]string{
					"claude-*-4": "anthropic/claude-*-v4",
				},
			},
			requestedModel: "claude-sonnet-4",
			expected:       "anthropic/claude-sonnet-v4",
		},

		// 精确优先于通配符
		{
			name: "精确映射优先",
			provider: Provider{
				ModelMapping: map[string]string{
					"claude-sonnet-4": "exact-match",
					"claude-*":        "wildcard-match",
				},
			},
			requestedModel: "claude-sonnet-4",
			expected:       "exact-match",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.provider.GetEffectiveModel(tt.requestedModel)
			if result != tt.expected {
				t.Errorf("GetEffectiveModel(%q) = %q, 期望 %q",
					tt.requestedModel, result, tt.expected)
			}
		})
	}
}

// ==================== ValidateConfiguration 测试 ====================

func TestProvider_ValidateConfiguration(t *testing.T) {
	tests := []struct {
		name          string
		provider      Provider
		expectErrors  bool
		errorContains string
	}{
		// 有效配置
		{
			name: "有效配置-完整",
			provider: Provider{
				Name: "test-provider",
				SupportedModels: map[string]bool{
					"model-a":          true,
					"internal-model-b": true,
				},
				ModelMapping: map[string]string{
					"external-model-b": "internal-model-b",
				},
			},
			expectErrors: false,
		},

		// 无效映射：目标模型不在白名单
		{
			name: "无效映射-目标不在白名单",
			provider: Provider{
				Name: "test-provider",
				SupportedModels: map[string]bool{
					"model-a": true,
				},
				ModelMapping: map[string]string{
					"external": "model-b",
				},
			},
			expectErrors:  true,
			errorContains: "不在 supportedModels 中",
		},

		// 警告：只配置映射未配置白名单
		{
			name: "警告-无白名单",
			provider: Provider{
				Name: "test-provider",
				ModelMapping: map[string]string{
					"external": "internal",
				},
			},
			expectErrors:  true,
			errorContains: "未配置 supportedModels",
		},

		// 警告：自映射
		{
			name: "警告-自映射",
			provider: Provider{
				Name: "test-provider",
				SupportedModels: map[string]bool{
					"model-a": true,
				},
				ModelMapping: map[string]string{
					"model-a": "model-a",
				},
			},
			expectErrors:  true,
			errorContains: "映射到自身",
		},

		// 通配符映射（不验证）
		{
			name: "通配符映射-跳过验证",
			provider: Provider{
				Name: "test-provider",
				SupportedModels: map[string]bool{
					"anthropic/claude-*": true,
				},
				ModelMapping: map[string]string{
					"claude-*": "anthropic/claude-*",
				},
			},
			expectErrors: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := tt.provider.ValidateConfiguration()

			if tt.expectErrors && len(errors) == 0 {
				t.Errorf("期望有验证错误，但没有返回错误")
			}

			if !tt.expectErrors && len(errors) > 0 {
				t.Errorf("不期望有验证错误，但返回了: %v", errors)
			}

			if tt.expectErrors && tt.errorContains != "" {
				found := false
				for _, err := range errors {
					if strings.Contains(err, tt.errorContains) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("期望错误信息包含 %q，但实际错误是: %v", tt.errorContains, errors)
				}
			}
		})
	}
}