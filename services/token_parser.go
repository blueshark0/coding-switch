package services

import "github.com/tidwall/gjson"

// TokenUsageParser 解析不同平台的 Token 用量
type TokenUsageParser interface {
	Parse(data string, usage *RequestLog)
}

// GetTokenParser 根据平台获取解析器
func GetTokenParser(platform Platform) TokenUsageParser {
	switch platform {
	case PlatformCodex:
		return &CodexTokenParser{}
	case PlatformGemini:
		return &GeminiTokenParser{}
	default:
		return &ClaudeTokenParser{}
	}
}

// ClaudeTokenParser Claude 平台解析器
type ClaudeTokenParser struct{}

// Parse 解析 Claude 响应中的 Token 用量
func (p *ClaudeTokenParser) Parse(data string, usage *RequestLog) {
	usage.InputTokens += int(gjson.Get(data, "message.usage.input_tokens").Int())
	usage.OutputTokens += int(gjson.Get(data, "message.usage.output_tokens").Int())
	usage.CacheCreateTokens += int(gjson.Get(data, "message.usage.cache_creation_input_tokens").Int())
	usage.CacheReadTokens += int(gjson.Get(data, "message.usage.cache_read_input_tokens").Int())
	usage.InputTokens += int(gjson.Get(data, "usage.input_tokens").Int())
	usage.OutputTokens += int(gjson.Get(data, "usage.output_tokens").Int())
}

// CodexTokenParser Codex 平台解析器
type CodexTokenParser struct{}

// Parse 解析 Codex 响应中的 Token 用量
func (p *CodexTokenParser) Parse(data string, usage *RequestLog) {
	usage.InputTokens += int(gjson.Get(data, "response.usage.input_tokens").Int())
	usage.OutputTokens += int(gjson.Get(data, "response.usage.output_tokens").Int())
	usage.CacheReadTokens += int(gjson.Get(data, "response.usage.input_tokens_details.cached_tokens").Int())
	usage.ReasoningTokens += int(gjson.Get(data, "response.usage.output_tokens_details.reasoning_tokens").Int())
}

// GeminiTokenParser Gemini 平台解析器
type GeminiTokenParser struct{}

// Parse 解析 Gemini 响应中的 Token 用量
func (p *GeminiTokenParser) Parse(data string, usage *RequestLog) {
	usage.InputTokens += int(gjson.Get(data, "usageMetadata.promptTokenCount").Int())
	usage.OutputTokens += int(gjson.Get(data, "usageMetadata.candidatesTokenCount").Int())
	usage.CacheReadTokens += int(gjson.Get(data, "usageMetadata.cachedContentTokenCount").Int())
	if total := gjson.Get(data, "usageMetadata.totalTokenCount").Int(); total > 0 {
		// 如果提供了 totalTokenCount，用它来补充输出 token 统计
		remaining := int(total) - usage.InputTokens
		if remaining > usage.OutputTokens {
			usage.OutputTokens = remaining
		}
	}
}
