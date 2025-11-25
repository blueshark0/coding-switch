package services

import "github.com/tidwall/gjson"

// CodexHandler 处理 Codex 平台特定逻辑
type CodexHandler struct {
	defaultEndpoint string
}

// NewCodexHandler 创建 Codex 处理器
func NewCodexHandler(endpoint string) *CodexHandler {
	return &CodexHandler{defaultEndpoint: endpoint}
}

// ExtractRequestInfo 从请求中提取 Codex 特定信息
func (h *CodexHandler) ExtractRequestInfo(path string, bodyBytes []byte, query, headers map[string]string) (
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
