package kernel

import (
	"fmt"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

type Platform string

const (
	PlatformClaude Platform = "claude"
	PlatformCodex  Platform = "codex"
	PlatformGemini Platform = "gemini"
)

func AllPlatforms() []Platform {
	return []Platform{PlatformClaude, PlatformCodex, PlatformGemini}
}

func ParsePlatform(value string) (Platform, error) {
	switch Platform(value) {
	case PlatformClaude, PlatformCodex, PlatformGemini:
		return Platform(value), nil
	default:
		return "", fmt.Errorf("unknown platform: %s", value)
	}
}

func MustPlatform(value string) Platform {
	platform, err := ParsePlatform(value)
	if err != nil {
		panic(err)
	}
	return platform
}

func (p Platform) String() string {
	return string(p)
}

func (p Platform) SessionTimeout() time.Duration {
	switch p {
	case PlatformCodex:
		return 15 * time.Minute
	case PlatformGemini:
		return 60 * time.Minute
	default:
		return 5 * time.Minute
	}
}

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
