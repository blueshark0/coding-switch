package services

import "github.com/tidwall/gjson"

// ClaudeHandler 处理 Claude 平台特定逻辑
type ClaudeHandler struct {
	defaultEndpoint string
}

// NewClaudeHandler 创建 Claude 处理器
func NewClaudeHandler(endpoint string) *ClaudeHandler {
	return &ClaudeHandler{defaultEndpoint: endpoint}
}

// ExtractRequestInfo 从请求中提取 Claude 特定信息
func (h *ClaudeHandler) ExtractRequestInfo(path string, bodyBytes []byte, query, headers map[string]string) (
	endpoint string,
	requestedModel string,
	isStream bool,
	routeOptions *relayRouteOptions,
) {
	endpoint = h.defaultEndpoint
	requestedModel = gjson.GetBytes(bodyBytes, "model").String()
	isStream = gjson.GetBytes(bodyBytes, "stream").Bool()
	routeOptions = nil
	return
}
