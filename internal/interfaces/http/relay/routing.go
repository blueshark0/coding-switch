package relay

import (
	"fmt"
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
	sessionID := ctx.RequestMeta.SessionID
	relayDebugf("session=%s platform=%s", sessionID, kind)

	if provider, sessionAlreadyBound, sessionGeneration, ok := s.resolveBoundProvider(ctx, kind, sessionID); ok {
		fwdCtx, err := s.prepareForwardContext(ctx, provider)
		if err != nil {
			return false, err
		}
		return s.executeAndHandleSession(fwdCtx, sessionID, sessionAlreadyBound, sessionGeneration)
	}

	provider, err := s.resolveDefaultProvider(ctx.Profile, ctx.RequestMeta.RequestedModel)
	if err != nil {
		return false, err
	}
	fwdCtx, err := s.prepareForwardContext(ctx, provider)
	if err != nil {
		return false, err
	}
	return s.executeAndHandleSession(fwdCtx, sessionID, false, s.sessionGeneration(kind, sessionID))
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
	if provider.APIURL == "" || provider.APIKey == "" {
		return fmt.Errorf("供应商 %s 配置不完整", provider.Name)
	}
	return nil
}

func (s *Server) resolveBoundProvider(
	ctx *RelayContext,
	kind string,
	sessionID string,
) (*routingdomain.Provider, bool, uint64, bool) {
	if sessionID == "" || s.sessionCache == nil {
		return nil, false, 0, false
	}

	boundProviderName, generation, err := s.sessionCache.GetSessionProviderSnapshot(kind, sessionID)
	if err != nil {
		relayWarnf("查询会话绑定失败: %v", err)
		return nil, false, generation, false
	}
	if boundProviderName == "" {
		return nil, false, generation, false
	}

	relayDebugf("session already bound: %s", boundProviderName)
	provider, err := s.resolveProviderForRequest(ctx.Profile, boundProviderName, ctx.RequestMeta.RequestedModel)
	if err == nil {
		return provider, true, generation, true
	}

	relayWarnf("检测到失效会话绑定，session=%s provider=%s error=%v", sessionID, boundProviderName, err)
	s.clearInvalidSessionBinding(kind, sessionID)
	return nil, false, generation, false
}

func (s *Server) sessionGeneration(kind, sessionID string) uint64 {
	if sessionID == "" || s.sessionCache == nil {
		return 0
	}
	return s.sessionCache.SessionGeneration(kind, sessionID)
}

func (s *Server) clearInvalidSessionBinding(kind, sessionID string) {
	if sessionID == "" {
		return
	}
	if s.sessionCache != nil {
		s.sessionCache.InvalidateSession(kind, sessionID)
	}
	if s.sessionService == nil {
		return
	}
	if err := s.sessionService.Unbind(kind, sessionID); err != nil {
		relayWarnf("清理失效会话绑定失败: %v", err)
	}
}

func (s *Server) resolveDefaultProvider(
	profile routingdomain.RouteProfile,
	requestedModel string,
) (*routingdomain.Provider, error) {
	defaultProvider := profile.DefaultProvider()
	if defaultProvider == nil {
		return nil, fmt.Errorf("未配置默认供应商")
	}
	relayDebugf("using default provider: %s", defaultProvider.Name)
	return s.resolveProviderForRequest(profile, defaultProvider.Name, requestedModel)
}

func (s *Server) resolveProviderForRequest(
	profile routingdomain.RouteProfile,
	providerName string,
	requestedModel string,
) (*routingdomain.Provider, error) {
	provider := s.findProvider(profile.Providers, providerName)
	if provider == nil {
		return nil, fmt.Errorf("供应商 %s 不存在", providerName)
	}
	if err := s.validateProvider(provider); err != nil {
		return nil, err
	}
	if requestedModel != "" && !provider.IsModelSupported(requestedModel) {
		return nil, fmt.Errorf("供应商 %s 不支持模型 %s", provider.Name, requestedModel)
	}
	return provider, nil
}

func (s *Server) prepareForwardContext(ctx *RelayContext, provider *routingdomain.Provider) (*ForwardContext, error) {
	effectiveModel := provider.GetEffectiveModel(ctx.RequestMeta.RequestedModel)
	currentBodyBytes := ctx.BodyBytes
	bodyModelValue := effectiveModel
	if ctx.RequestMeta.RouteOptions != nil && ctx.RequestMeta.RouteOptions.bodyModelFormatter != nil {
		bodyModelValue = ctx.RequestMeta.RouteOptions.bodyModelFormatter(effectiveModel)
	}
	bodyHasModelField := ctx.RequestMeta.BodyHasModelField
	shouldRewriteBody := effectiveModel != ctx.RequestMeta.RequestedModel && ctx.RequestMeta.RequestedModel != ""
	if ctx.RequestMeta.RouteOptions != nil && ctx.RequestMeta.RouteOptions.forceBodyRewrite {
		shouldRewriteBody = shouldRewriteBody || (bodyHasModelField && ctx.RequestMeta.RequestedModel != "")
	}
	if shouldRewriteBody && bodyHasModelField {
		relayDebugf("remap model: %s -> %s", ctx.RequestMeta.RequestedModel, bodyModelValue)
		modifiedBody, err := ReplaceModelInRequestBody(ctx.BodyBytes, bodyModelValue)
		if err != nil {
			return nil, fmt.Errorf("替换模型名失败: %w", err)
		}
		currentBodyBytes = modifiedBody
	}
	targetEndpoint := ctx.RequestMeta.Endpoint
	if ctx.RequestMeta.RouteOptions != nil && ctx.RequestMeta.RouteOptions.endpointMutator != nil {
		newEndpoint, err := ctx.RequestMeta.RouteOptions.endpointMutator(targetEndpoint, ctx.RequestMeta.RequestedModel, effectiveModel)
		if err != nil {
			return nil, err
		}
		targetEndpoint = newEndpoint
	}
	targetQuery := ctx.Query
	if ctx.RequestMeta.RouteOptions != nil && ctx.RequestMeta.RouteOptions.queryMutator != nil {
		targetQuery = ctx.RequestMeta.RouteOptions.queryMutator(cloneMap(ctx.Query), *provider)
	}
	if targetQuery == nil {
		targetQuery = make(map[string]string)
	}
	targetHeaders := ctx.Headers
	if ctx.RequestMeta.RouteOptions != nil && ctx.RequestMeta.RouteOptions.headerMutator != nil {
		targetHeaders = ctx.RequestMeta.RouteOptions.headerMutator(cloneMap(ctx.Headers), *provider)
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
		IsStream:  ctx.RequestMeta.IsStream,
		IsFast:    ctx.RequestMeta.IsFast,
		Model:     effectiveModel,
	}, nil
}

func (s *Server) executeAndHandleSession(
	fwdCtx *ForwardContext,
	sessionID string,
	sessionAlreadyBound bool,
	sessionGeneration uint64,
) (bool, error) {
	kind := fwdCtx.Platform.String()
	relayDebugf("forwarding provider=%s model=%s", fwdCtx.Provider.Name, fwdCtx.Model)
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
		fwdCtx.IsFast,
		fwdCtx.Model,
	)
	duration := time.Since(startTime)
	if ok {
		s.handleSuccessfulSession(kind, sessionID, fwdCtx.Provider.Name, sessionAlreadyBound, sessionGeneration)
		relayDebugf("request succeeded provider=%s duration=%.2fs", fwdCtx.Provider.Name, duration.Seconds())
		return true, nil
	}
	errorMsg := "未知错误"
	if err != nil {
		errorMsg = err.Error()
	}
	relayWarnf("请求失败: provider=%s error=%s duration=%.2fs", fwdCtx.Provider.Name, errorMsg, duration.Seconds())
	return false, err
}

func (s *Server) handleSuccessfulSession(kind, sessionID, providerName string, alreadyBound bool, generation uint64) {
	if sessionID == "" {
		return
	}
	if !alreadyBound {
		if err := s.sessionCache.BindSessionToProviderGeneration(kind, sessionID, providerName, generation); err != nil {
			relayWarnf("绑定会话失败: %v", err)
		}
		return
	}
	if !s.sessionCache.RecordSessionSuccessGeneration(kind, sessionID, providerName, generation) {
		return
	}
	if s.sessionUpdateWorker != nil {
		s.sessionUpdateWorker.Enqueue(sessionUpdateRequest{platform: kind, sessionID: sessionID, generation: generation})
		return
	}
	if err := s.sessionCache.UpdateSessionSuccessGeneration(kind, sessionID, generation); err != nil {
		relayWarnf("更新会话时间失败: %v", err)
	}
}
