package infrastructure

import (
	observabilitydomain "codeswitch/internal/observability/domain"
	"codeswitch/internal/shared/kernel"

	"github.com/tidwall/gjson"
)

type TokenUsageParser interface {
	Parse(data string, usage *observabilitydomain.RequestLog)
}

func GetTokenParser(platform kernel.Platform) TokenUsageParser {
	switch platform {
	case kernel.PlatformCodex:
		return &CodexTokenParser{}
	case kernel.PlatformGemini:
		return &GeminiTokenParser{}
	default:
		return &ClaudeTokenParser{}
	}
}

type ClaudeTokenParser struct{}

func (p *ClaudeTokenParser) Parse(data string, usage *observabilitydomain.RequestLog) {
	usage.InputTokens += int(gjson.Get(data, "message.usage.input_tokens").Int())
	usage.OutputTokens += int(gjson.Get(data, "message.usage.output_tokens").Int())
	usage.CacheCreateTokens += int(gjson.Get(data, "message.usage.cache_creation_input_tokens").Int())
	usage.CacheReadTokens += int(gjson.Get(data, "message.usage.cache_read_input_tokens").Int())
	usage.InputTokens += int(gjson.Get(data, "usage.input_tokens").Int())
	usage.OutputTokens += int(gjson.Get(data, "usage.output_tokens").Int())
}

type CodexTokenParser struct{}

func (p *CodexTokenParser) Parse(data string, usage *observabilitydomain.RequestLog) {
	totalInput := int(gjson.Get(data, "response.usage.input_tokens").Int())
	cachedInput := int(gjson.Get(data, "response.usage.input_tokens_details.cached_tokens").Int())
	nonCachedInput := totalInput - cachedInput
	if nonCachedInput < 0 {
		nonCachedInput = 0
	}
	usage.InputTokens += nonCachedInput
	usage.OutputTokens += int(gjson.Get(data, "response.usage.output_tokens").Int())
	usage.CacheReadTokens += cachedInput
	usage.ReasoningTokens += int(gjson.Get(data, "response.usage.output_tokens_details.reasoning_tokens").Int())
}

type GeminiTokenParser struct{}

func (p *GeminiTokenParser) Parse(data string, usage *observabilitydomain.RequestLog) {
	usage.InputTokens += int(gjson.Get(data, "usageMetadata.promptTokenCount").Int())
	usage.OutputTokens += int(gjson.Get(data, "usageMetadata.candidatesTokenCount").Int())
	usage.CacheReadTokens += int(gjson.Get(data, "usageMetadata.cachedContentTokenCount").Int())
	if total := gjson.Get(data, "usageMetadata.totalTokenCount").Int(); total > 0 {
		remaining := int(total) - usage.InputTokens
		if remaining > usage.OutputTokens {
			usage.OutputTokens = remaining
		}
	}
}
