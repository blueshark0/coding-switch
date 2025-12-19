package services

import (
	"fmt"
	"log"
	"time"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// ReplaceModelInRequestBody 替换请求体中的模型名
// 使用 gjson + sjson 实现高性能 JSON 操作，避免完整反序列化
func ReplaceModelInRequestBody(bodyBytes []byte, newModel string) ([]byte, error) {
	// 检查请求体中是否存在 model 字段
	result := gjson.GetBytes(bodyBytes, "model")
	if !result.Exists() {
		return bodyBytes, fmt.Errorf("请求体中未找到 model 字段")
	}

	// 使用 sjson.SetBytes 替换模型名（高性能操作）
	modified, err := sjson.SetBytes(bodyBytes, "model", newModel)
	if err != nil {
		return bodyBytes, fmt.Errorf("替换模型名失败: %w", err)
	}

	return modified, nil
}

type relayRouteOptions struct {
	endpointMutator    func(endpoint, requestedModel, effectiveModel string) (string, error)
	queryMutator       func(query map[string]string, provider Provider) map[string]string
	headerMutator      func(headers map[string]string, provider Provider) map[string]string
	bodyModelFormatter func(model string) string
	forceBodyRewrite   bool
}

// routeToManualProvider 手动路由模式下的请求处理
// 返回值：(是否成功, 错误信息)
func (prs *ProviderRelayService) routeToManualProvider(ctx *RelayContext) (bool, error) {
	kind := ctx.Platform.String()
	sessionID := ctx.Platform.ExtractSessionID(ctx.Headers, ctx.BodyBytes)
	log.Printf("[INFO] [手动路由] 会话ID: %s\n", sessionID)

	var targetProviderName string
	sessionAlreadyBound := false

	// 步骤1：检查会话是否已绑定（使用缓存减少数据库查询）
	if sessionID != "" {
		boundProvider, err := prs.sessionCache.GetSessionProvider(kind, sessionID)
		if err != nil {
			log.Printf("[WARN] 查询会话绑定失败: %v\n", err)
		} else if boundProvider != "" {
			targetProviderName = boundProvider
			sessionAlreadyBound = true
			log.Printf("[INFO] [手动路由] 会话已绑定到供应商: %s\n", boundProvider)
		}
	}

	// 步骤2：如果未绑定或已过期，使用默认供应商
	if targetProviderName == "" {
		targetProviderName = ctx.Platform.GetDefaultProvider(ctx.AppSettings)
		if targetProviderName == "" {
			return false, fmt.Errorf("未配置默认供应商")
		}
		log.Printf("[INFO] [手动路由] 使用默认供应商: %s\n", targetProviderName)
	}

	// 步骤3：查找目标供应商
	provider := prs.findProvider(ctx.Providers, targetProviderName)
	if provider == nil {
		return false, fmt.Errorf("供应商 %s 不存在", targetProviderName)
	}

	// 步骤4：验证供应商配置
	if err := prs.validateProvider(provider); err != nil {
		return false, err
	}

	// 步骤5：检查模型支持
	if ctx.RequestedModel != "" && !provider.IsModelSupported(ctx.RequestedModel) {
		return false, fmt.Errorf("供应商 %s 不支持模型 %s", provider.Name, ctx.RequestedModel)
	}

	// 步骤6：准备转发上下文
	fwdCtx, err := prs.prepareForwardContext(ctx, provider)
	if err != nil {
		return false, err
	}

	// 步骤7：转发请求
	return prs.executeAndHandleSession(fwdCtx, sessionID, sessionAlreadyBound)
}

// findProvider 从 providers 列表中查找指定名称的供应商
func (prs *ProviderRelayService) findProvider(providers []Provider, name string) *Provider {
	for i := range providers {
		if providers[i].Name == name {
			return &providers[i]
		}
	}
	return nil
}

// validateProvider 验证供应商配置
func (prs *ProviderRelayService) validateProvider(provider *Provider) error {
	if !provider.Enabled {
		return fmt.Errorf("供应商 %s 已被禁用", provider.Name)
	}
	if provider.APIURL == "" || provider.APIKey == "" {
		return fmt.Errorf("供应商 %s 配置不完整", provider.Name)
	}
	return nil
}

// prepareForwardContext 准备转发上下文（模型映射、端点变换等）
func (prs *ProviderRelayService) prepareForwardContext(ctx *RelayContext, provider *Provider) (*ForwardContext, error) {
	effectiveModel := provider.GetEffectiveModel(ctx.RequestedModel)
	currentBodyBytes := ctx.BodyBytes

	// 处理模型名替换
	bodyModelValue := effectiveModel
	if ctx.RouteOptions != nil && ctx.RouteOptions.bodyModelFormatter != nil {
		bodyModelValue = ctx.RouteOptions.bodyModelFormatter(effectiveModel)
	}

	bodyHasModelField := gjson.GetBytes(ctx.BodyBytes, "model").Exists()
	shouldRewriteBody := effectiveModel != ctx.RequestedModel && ctx.RequestedModel != ""
	if ctx.RouteOptions != nil && ctx.RouteOptions.forceBodyRewrite {
		shouldRewriteBody = shouldRewriteBody || (bodyHasModelField && ctx.RequestedModel != "")
	}

	if shouldRewriteBody && bodyHasModelField {
		log.Printf("[INFO] [手动路由] 映射模型: %s -> %s\n", ctx.RequestedModel, bodyModelValue)
		modifiedBody, err := ReplaceModelInRequestBody(ctx.BodyBytes, bodyModelValue)
		if err != nil {
			return nil, fmt.Errorf("替换模型名失败: %w", err)
		}
		currentBodyBytes = modifiedBody
	}

	// 处理端点变换
	targetEndpoint := ctx.Endpoint
	if ctx.RouteOptions != nil && ctx.RouteOptions.endpointMutator != nil {
		newEndpoint, err := ctx.RouteOptions.endpointMutator(targetEndpoint, ctx.RequestedModel, effectiveModel)
		if err != nil {
			return nil, err
		}
		targetEndpoint = newEndpoint
	}

	// 处理查询参数变换
	targetQuery := ctx.Query
	if ctx.RouteOptions != nil && ctx.RouteOptions.queryMutator != nil {
		targetQuery = ctx.RouteOptions.queryMutator(cloneMap(ctx.Query), *provider)
	}
	if targetQuery == nil {
		targetQuery = make(map[string]string)
	}

	// 处理请求头变换
	targetHeaders := ctx.Headers
	if ctx.RouteOptions != nil && ctx.RouteOptions.headerMutator != nil {
		targetHeaders = ctx.RouteOptions.headerMutator(cloneMap(ctx.Headers), *provider)
	}
	if targetHeaders == nil {
		targetHeaders = make(map[string]string)
	}

	return &ForwardContext{
		GinCtx:    ctx.GinCtx,
		Platform:  ctx.Platform,
		Provider:  provider,
		Endpoint:  targetEndpoint,
		Query:     targetQuery,
		Headers:   targetHeaders,
		BodyBytes: currentBodyBytes,
		IsStream:  ctx.IsStream,
		Model:     effectiveModel,
	}, nil
}

// executeAndHandleSession 执行请求并处理会话绑定
func (prs *ProviderRelayService) executeAndHandleSession(
	fwdCtx *ForwardContext,
	sessionID string,
	sessionAlreadyBound bool,
) (bool, error) {
	kind := fwdCtx.Platform.String()
	log.Printf("[INFO] [手动路由] 转发请求到供应商: %s | 模型: %s\n", fwdCtx.Provider.Name, fwdCtx.Model)

	startTime := time.Now()
	ok, err := prs.forwardRequest(
		fwdCtx.GinCtx,
		fwdCtx.Platform,
		*fwdCtx.Provider,
		fwdCtx.Endpoint,
		fwdCtx.Query,
		fwdCtx.Headers,
		fwdCtx.BodyBytes,
		fwdCtx.IsStream,
		fwdCtx.Model,
	)
	duration := time.Since(startTime)

	if ok {
		log.Printf("[INFO] [手动路由] ✓ 请求成功: %s | 耗时: %.2fs\n", fwdCtx.Provider.Name, duration.Seconds())
		prs.handleSuccessfulSession(kind, sessionID, fwdCtx.Provider.Name, sessionAlreadyBound)
		return true, nil
	}

	errorMsg := "未知错误"
	if err != nil {
		errorMsg = err.Error()
	}
	log.Printf("[WARN] [手动路由] ✗ 请求失败: %s | 错误: %s | 耗时: %.2fs\n",
		fwdCtx.Provider.Name, errorMsg, duration.Seconds())

	return false, err
}

// handleSuccessfulSession 处理成功请求后的会话绑定
func (prs *ProviderRelayService) handleSuccessfulSession(kind, sessionID, providerName string, alreadyBound bool) {
	if sessionID == "" {
		return
	}

	if !alreadyBound {
		if err := prs.sessionCache.BindSessionToProvider(kind, sessionID, providerName); err != nil {
			log.Printf("[WARN] 绑定会话失败: %v\n", err)
		}
	} else {
		// 异步更新最后成功时间
		if prs.sessionUpdateWorker != nil {
			prs.sessionUpdateWorker.Enqueue(sessionUpdateRequest{platform: kind, sessionID: sessionID})
			return
		}
		if err := prs.sessionCache.UpdateSessionSuccess(kind, sessionID); err != nil {
			log.Printf("[WARN] 更新会话时间失败: %v\n", err)
		}
	}
}
