package services

import (
	"strings"

	"github.com/tidwall/gjson"
)

// GeminiHandler 处理 Gemini 平台特定逻辑
type GeminiHandler struct{}

// NewGeminiHandler 创建 Gemini 处理器
func NewGeminiHandler() *GeminiHandler {
	return &GeminiHandler{}
}

// ExtractRequestInfo 从请求中提取 Gemini 特定信息
func (h *GeminiHandler) ExtractRequestInfo(path string, bodyBytes []byte, query, headers map[string]string) (
	endpoint string,
	requestedModel string,
	isStream bool,
	routeOptions *relayRouteOptions,
) {
	// 动态提取 endpoint
	endpoint = strings.TrimPrefix(path, "/gemini")

	// 双重模型提取
	var bodyModel string
	requestedModel, bodyModel = extractGeminiModel(endpoint, bodyBytes)

	// 多源流式检测
	isStream = detectGeminiStream(endpoint, query, headers, bodyBytes)

	// 动态构造 Gemini options
	routeOptions = h.createRouteOptions(bodyModel != "")

	return
}

// createRouteOptions 创建 Gemini 路由选项
func (h *GeminiHandler) createRouteOptions(forceBodyRewrite bool) *relayRouteOptions {
	return &relayRouteOptions{
		endpointMutator: func(endpoint, _, effectiveModel string) (string, error) {
			if effectiveModel == "" {
				return endpoint, nil
			}
			return replaceGeminiModelInPath(endpoint, effectiveModel), nil
		},
		queryMutator: func(q map[string]string, provider Provider) map[string]string {
			if q == nil {
				q = make(map[string]string)
			}
			if provider.APIKey != "" {
				q["key"] = provider.APIKey
			}
			return q
		},
		headerMutator: func(headers map[string]string, provider Provider) map[string]string {
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

// extractGeminiModel 从路径和请求体中提取 Gemini 模型名
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

// normalizeGeminiModel 标准化 Gemini 模型名
func normalizeGeminiModel(model string) string {
	trimmed := strings.TrimSpace(model)
	if strings.Contains(trimmed, "/models/") {
		parts := strings.SplitN(trimmed, "/models/", 2)
		trimmed = parts[1]
	}
	trimmed = strings.TrimPrefix(trimmed, "models/")
	return trimmed
}

// formatGeminiModelForBody 格式化 Gemini 模型名用于请求体
func formatGeminiModelForBody(model string) string {
	if model == "" {
		return ""
	}
	if strings.HasPrefix(model, "models/") {
		return model
	}
	return "models/" + model
}

// replaceGeminiModelInPath 替换路径中的 Gemini 模型名
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

// detectGeminiStream 检测 Gemini 请求是否为流式
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
	if strings.Contains(strings.ToLower(accept), "event-stream") {
		return true
	}

	return false
}
