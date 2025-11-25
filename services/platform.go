package services

// Platform 定义支持的平台类型
type Platform string

const (
	PlatformClaude Platform = "claude"
	PlatformCodex  Platform = "codex"
	PlatformGemini Platform = "gemini"
)

// AllPlatforms 返回所有支持的平台
func AllPlatforms() []Platform {
	return []Platform{PlatformClaude, PlatformCodex, PlatformGemini}
}

// String 实现 Stringer 接口
func (p Platform) String() string {
	return string(p)
}

// GetDefaultProvider 获取 AppSettings 中对应平台的默认供应商
func (p Platform) GetDefaultProvider(settings AppSettings) string {
	switch p {
	case PlatformClaude:
		return settings.DefaultClaudeProvider
	case PlatformCodex:
		return settings.DefaultCodexProvider
	case PlatformGemini:
		return settings.DefaultGeminiProvider
	default:
		return ""
	}
}
