package services

// PlatformHandler 定义平台处理器接口
// 每个平台（Claude/Codex/Gemini）实现此接口以提取请求信息
type PlatformHandler interface {
	// ExtractRequestInfo 从请求中提取平台特定信息
	// 参数:
	//   - path: 请求路径
	//   - bodyBytes: 请求体字节
	//   - query: 查询参数
	//   - headers: 请求头
	// 返回:
	//   - endpoint: 目标端点
	//   - requestedModel: 请求的模型名
	//   - isStream: 是否流式请求
	//   - routeOptions: 路由选项（可选）
	ExtractRequestInfo(path string, bodyBytes []byte, query, headers map[string]string) (
		endpoint string,
		requestedModel string,
		isStream bool,
		routeOptions *relayRouteOptions,
	)
}

// GetPlatformHandler 返回对应平台的处理器
func GetPlatformHandler(platform Platform, defaultEndpoint string) PlatformHandler {
	switch platform {
	case PlatformClaude:
		return NewClaudeHandler(defaultEndpoint)
	case PlatformCodex:
		return NewCodexHandler(defaultEndpoint)
	case PlatformGemini:
		return NewGeminiHandler()
	default:
		return NewClaudeHandler(defaultEndpoint)
	}
}
