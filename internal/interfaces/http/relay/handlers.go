package relay

import (
	"strings"

	routingdomain "codeswitch/internal/routing/domain"
	"codeswitch/internal/shared/kernel"
	"github.com/tidwall/gjson"
)

type ClaudeHandler struct {
	defaultEndpoint string
}

func NewClaudeHandler(endpoint string) *ClaudeHandler {
	return &ClaudeHandler{defaultEndpoint: endpoint}
}

func (h *ClaudeHandler) ExtractRequestInfo(path string, bodyBytes []byte, query, headers map[string]string) (
	endpoint string,
	requestedModel string,
	isStream bool,
	routeOptions *relayRouteOptions,
) {
	endpoint = h.defaultEndpoint
	requestedModel = gjson.GetBytes(bodyBytes, "model").String()
	isStream = gjson.GetBytes(bodyBytes, "stream").Bool()
	return
}

type CodexHandler struct {
	defaultEndpoint string
}

func NewCodexHandler(endpoint string) *CodexHandler {
	return &CodexHandler{defaultEndpoint: endpoint}
}

func (h *CodexHandler) ExtractRequestInfo(path string, bodyBytes []byte, query, headers map[string]string) (
	endpoint string,
	requestedModel string,
	isStream bool,
	routeOptions *relayRouteOptions,
) {
	endpoint = h.defaultEndpoint
	requestedModel = gjson.GetBytes(bodyBytes, "model").String()
	isStream = gjson.GetBytes(bodyBytes, "stream").Bool()
	return
}

type GeminiHandler struct{}

func NewGeminiHandler() *GeminiHandler {
	return &GeminiHandler{}
}

func (h *GeminiHandler) ExtractRequestInfo(path string, bodyBytes []byte, query, headers map[string]string) (
	endpoint string,
	requestedModel string,
	isStream bool,
	routeOptions *relayRouteOptions,
) {
	endpoint = strings.TrimPrefix(path, "/gemini")
	var bodyModel string
	requestedModel, bodyModel = extractGeminiModel(endpoint, bodyBytes)
	isStream = detectGeminiStream(endpoint, query, headers, bodyBytes)
	routeOptions = h.createRouteOptions(bodyModel != "")
	return
}

func (h *GeminiHandler) createRouteOptions(forceBodyRewrite bool) *relayRouteOptions {
	return &relayRouteOptions{
		endpointMutator: func(endpoint, _, effectiveModel string) (string, error) {
			if effectiveModel == "" {
				return endpoint, nil
			}
			return replaceGeminiModelInPath(endpoint, effectiveModel), nil
		},
		queryMutator: func(q map[string]string, provider routingdomain.Provider) map[string]string {
			if q == nil {
				q = make(map[string]string)
			}
			if provider.APIKey != "" {
				q["key"] = provider.APIKey
			}
			return q
		},
		headerMutator: func(headers map[string]string, provider routingdomain.Provider) map[string]string {
			if headers == nil {
				headers = make(map[string]string)
			}
			if provider.APIKey != "" {
				headers["x-goog-api-key"] = provider.APIKey
			}
			return headers
		},
		bodyModelFormatter: formatGeminiModelForBody,
		forceBodyRewrite:   forceBodyRewrite,
	}
}

func GetPlatformHandler(platform kernel.Platform, defaultEndpoint string) PlatformHandler {
	switch platform {
	case kernel.PlatformClaude:
		return NewClaudeHandler(defaultEndpoint)
	case kernel.PlatformCodex:
		return NewCodexHandler(defaultEndpoint)
	case kernel.PlatformGemini:
		return NewGeminiHandler()
	default:
		return NewClaudeHandler(defaultEndpoint)
	}
}

func extractGeminiModel(path string, bodyBytes []byte) (string, string) {
	bodyModel := strings.TrimSpace(gjson.GetBytes(bodyBytes, "model").String())

	modelFromPath := ""
	if idx := strings.Index(path, "/models/"); idx >= 0 {
		rest := path[idx+len("/models/"):]
		delimiter := strings.IndexAny(rest, ":?")
		if delimiter >= 0 {
			modelFromPath = rest[:delimiter]
		} else {
			modelFromPath = rest
		}
	}

	requested := normalizeGeminiModel(bodyModel)
	if requested == "" {
		requested = normalizeGeminiModel(modelFromPath)
	}
	return requested, bodyModel
}

func normalizeGeminiModel(model string) string {
	trimmed := strings.TrimSpace(model)
	if strings.Contains(trimmed, "/models/") {
		parts := strings.SplitN(trimmed, "/models/", 2)
		trimmed = parts[1]
	}
	trimmed = strings.TrimPrefix(trimmed, "models/")
	return trimmed
}

func formatGeminiModelForBody(model string) string {
	if model == "" {
		return ""
	}
	if strings.HasPrefix(model, "models/") {
		return model
	}
	return "models/" + model
}

func replaceGeminiModelInPath(path, effectiveModel string) string {
	if path == "" || effectiveModel == "" {
		return path
	}
	idx := strings.Index(path, "/models/")
	if idx < 0 {
		return path
	}
	start := idx + len("/models/")
	rest := path[start:]
	delimiter := strings.IndexAny(rest, ":?")
	if delimiter < 0 {
		return path[:start] + effectiveModel
	}
	return path[:start] + effectiveModel + rest[delimiter:]
}

func detectGeminiStream(path string, query map[string]string, headers map[string]string, body []byte) bool {
	if gjson.GetBytes(body, "stream").Bool() {
		return true
	}
	lowerPath := strings.ToLower(path)
	if strings.Contains(lowerPath, ":stream") || strings.Contains(lowerPath, "/stream") {
		return true
	}
	if alt, ok := query["alt"]; ok && strings.EqualFold(alt, "sse") {
		return true
	}
	accept := headers["Accept"]
	if accept == "" {
		accept = headers["accept"]
	}
	return strings.Contains(strings.ToLower(accept), "event-stream")
}
