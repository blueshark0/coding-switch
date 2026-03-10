package services

import "testing"

func TestCodexTokenParser_Parse_SeparatesCachedInput(t *testing.T) {
	parser := &CodexTokenParser{}
	usage := &RequestLog{}

	parser.Parse(`{"response":{"usage":{"input_tokens":1200,"output_tokens":320,"input_tokens_details":{"cached_tokens":900},"output_tokens_details":{"reasoning_tokens":40}}}}`, usage)

	if usage.InputTokens != 300 {
		t.Fatalf("InputTokens = %d, want 300", usage.InputTokens)
	}
	if usage.CacheReadTokens != 900 {
		t.Fatalf("CacheReadTokens = %d, want 900", usage.CacheReadTokens)
	}
	if usage.OutputTokens != 320 {
		t.Fatalf("OutputTokens = %d, want 320", usage.OutputTokens)
	}
	if usage.ReasoningTokens != 40 {
		t.Fatalf("ReasoningTokens = %d, want 40", usage.ReasoningTokens)
	}
}

func TestCodexTokenParser_Parse_ClampsNegativeNonCachedInput(t *testing.T) {
	parser := &CodexTokenParser{}
	usage := &RequestLog{}

	parser.Parse(`{"response":{"usage":{"input_tokens":10,"output_tokens":20,"input_tokens_details":{"cached_tokens":25},"output_tokens_details":{"reasoning_tokens":3}}}}`, usage)

	if usage.InputTokens != 0 {
		t.Fatalf("InputTokens = %d, want 0", usage.InputTokens)
	}
	if usage.CacheReadTokens != 25 {
		t.Fatalf("CacheReadTokens = %d, want 25", usage.CacheReadTokens)
	}
	if usage.OutputTokens != 20 {
		t.Fatalf("OutputTokens = %d, want 20", usage.OutputTokens)
	}
	if usage.ReasoningTokens != 3 {
		t.Fatalf("ReasoningTokens = %d, want 3", usage.ReasoningTokens)
	}
}
