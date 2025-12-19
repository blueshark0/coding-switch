package services

import (
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

// SessionTimeout returns the session TTL for the platform.
func (p Platform) SessionTimeout() time.Duration {
	switch p {
	case PlatformCodex:
		return CodexSessionTimeout
	case PlatformGemini:
		return GeminiSessionTimeout
	default:
		return ClaudeSessionTimeout
	}
}

// ExtractSessionID returns the session ID for the platform from headers/body.
func (p Platform) ExtractSessionID(headers map[string]string, bodyBytes []byte) string {
	switch p {
	case PlatformClaude:
		return gjson.GetBytes(bodyBytes, "metadata.user_id").String()
	case PlatformCodex:
		return headerValue(headers, "session_id")
	case PlatformGemini:
		if sessionID := headerValue(headers, "x-gemini-api-privileged-user-id"); sessionID != "" {
			return sessionID
		}
		return headerValue(headers, "x-session-id")
	default:
		return ""
	}
}

func headerValue(headers map[string]string, name string) string {
	if headers == nil {
		return ""
	}
	if value, ok := headers[name]; ok {
		return value
	}
	target := strings.ToLower(name)
	for key, value := range headers {
		if strings.ToLower(key) == target {
			return value
		}
	}
	return ""
}
