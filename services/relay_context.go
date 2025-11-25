package services

import "github.com/gin-gonic/gin"

// RelayContext 封装请求转发所需的上下文
type RelayContext struct {
	GinCtx         *gin.Context
	Platform       Platform
	Endpoint       string
	BodyBytes      []byte
	RequestedModel string
	IsStream       bool
	Query          map[string]string
	Headers        map[string]string
	AppSettings    AppSettings
	Providers      []Provider
	RouteOptions   *relayRouteOptions
}

// ForwardContext 转发请求上下文
type ForwardContext struct {
	GinCtx    *gin.Context
	Platform  Platform
	Provider  *Provider
	Endpoint  string
	Query     map[string]string
	Headers   map[string]string
	BodyBytes []byte
	IsStream  bool
	Model     string
}

// RouteResult 路由结果
type RouteResult struct {
	TargetProvider      *Provider
	BoundProviderName   string
	SessionAlreadyBound bool
	SessionID           string
}
