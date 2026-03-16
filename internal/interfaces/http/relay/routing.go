package relay

import (
	"fmt"
	"log"
	"time"

	routingdomain "codeswitch/internal/routing/domain"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func ReplaceModelInRequestBody(bodyBytes []byte, newModel string) ([]byte, error) {
	result := gjson.GetBytes(bodyBytes, "model")
	if !result.Exists() {
		return bodyBytes, fmt.Errorf("请求体中未找到 model 字段")
	}
	modified, err := sjson.SetBytes(bodyBytes, "model", newModel)
	if err != nil {
		return bodyBytes, fmt.Errorf("替换模型名失败: %w", err)
	}
	return modified, nil
}

func (s *Server) routeToManualProvider(ctx *RelayContext) (bool, error) {
	kind := ctx.Platform.String()
	sessionID := ctx.Platform.ExtractSessionID(ctx.Headers, ctx.BodyBytes)
	log.Printf("[INFO] [手动路由] 会话ID: %s\n", sessionID)
	var targetProviderName string
	sessionAlreadyBound := false
	if sessionID != "" {
		boundProvider, err := s.sessionCache.GetSessionProvider(kind, sessionID)
		if err != nil {
			log.Printf("[WARN] 查询会话绑定失败: %v\n", err)
		} else if boundProvider != "" {
			targetProviderName = boundProvider
			sessionAlreadyBound = true
			log.Printf("[INFO] [手动路由] 会话已绑定到供应商: %s\n", boundProvider)
		}
	}
	if targetProviderName == "" {
		provider := ctx.Profile.DefaultProvider()
		if provider != nil {
			targetProviderName = provider.Name
		}
		if targetProviderName == "" {
			return false, fmt.Errorf("未配置默认供应商")
		}
		log.Printf("[INFO] [手动路由] 使用默认供应商: %s\n", targetProviderName)
	}
	provider := s.findProvider(ctx.Profile.Providers, targetProviderName)
	if provider == nil {
		return false, fmt.Errorf("供应商 %s 不存在", targetProviderName)
	}
	if err := s.validateProvider(provider); err != nil {
		return false, err
	}
	if ctx.RequestedModel != "" && !provider.IsModelSupported(ctx.RequestedModel) {
		return false, fmt.Errorf("供应商 %s 不支持模型 %s", provider.Name, ctx.RequestedModel)
	}
	fwdCtx, err := s.prepareForwardContext(ctx, provider)
	if err != nil {
		return false, err
	}
	return s.executeAndHandleSession(fwdCtx, sessionID, sessionAlreadyBound)
}

func (s *Server) findProvider(providers []routingdomain.Provider, name string) *routingdomain.Provider {
	for i := range providers {
		if providers[i].Name == name {
			return &providers[i]
		}
	}
	return nil
}

func (s *Server) validateProvider(provider *routingdomain.Provider) error {
	if !provider.Enabled {
		return fmt.Errorf("供应商 %s 已被禁用", provider.Name)
	}
	if provider.APIURL == "" || provider.APIKey == "" {
		return fmt.Errorf("供应商 %s 配置不完整", provider.Name)
	}
	return nil
}

func (s *Server) prepareForwardContext(ctx *RelayContext, provider *routingdomain.Provider) (*ForwardContext, error) {
	effectiveModel := provider.GetEffectiveModel(ctx.RequestedModel)
	currentBodyBytes := ctx.BodyBytes
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
	targetEndpoint := ctx.Endpoint
	if ctx.RouteOptions != nil && ctx.RouteOptions.endpointMutator != nil {
		newEndpoint, err := ctx.RouteOptions.endpointMutator(targetEndpoint, ctx.RequestedModel, effectiveModel)
		if err != nil {
			return nil, err
		}
		targetEndpoint = newEndpoint
	}
	targetQuery := ctx.Query
	if ctx.RouteOptions != nil && ctx.RouteOptions.queryMutator != nil {
		targetQuery = ctx.RouteOptions.queryMutator(cloneMap(ctx.Query), *provider)
	}
	if targetQuery == nil {
		targetQuery = make(map[string]string)
	}
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

func (s *Server) executeAndHandleSession(
	fwdCtx *ForwardContext,
	sessionID string,
	sessionAlreadyBound bool,
) (bool, error) {
	kind := fwdCtx.Platform.String()
	log.Printf("[INFO] [手动路由] 转发请求到供应商: %s | 模型: %s\n", fwdCtx.Provider.Name, fwdCtx.Model)
	startTime := time.Now()
	ok, err := s.forwardRequest(
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
		s.handleSuccessfulSession(kind, sessionID, fwdCtx.Provider.Name, sessionAlreadyBound)
		return true, nil
	}
	errorMsg := "未知错误"
	if err != nil {
		errorMsg = err.Error()
	}
	log.Printf("[WARN] [手动路由] ✗ 请求失败: %s | 错误: %s | 耗时: %.2fs\n", fwdCtx.Provider.Name, errorMsg, duration.Seconds())
	return false, err
}

func (s *Server) handleSuccessfulSession(kind, sessionID, providerName string, alreadyBound bool) {
	if sessionID == "" {
		return
	}
	if !alreadyBound {
		if err := s.sessionCache.BindSessionToProvider(kind, sessionID, providerName); err != nil {
			log.Printf("[WARN] 绑定会话失败: %v\n", err)
		}
	} else {
		if s.sessionUpdateWorker != nil {
			s.sessionUpdateWorker.Enqueue(sessionUpdateRequest{platform: kind, sessionID: sessionID})
			return
		}
		if err := s.sessionCache.UpdateSessionSuccess(kind, sessionID); err != nil {
			log.Printf("[WARN] 更新会话时间失败: %v\n", err)
		}
	}
}
