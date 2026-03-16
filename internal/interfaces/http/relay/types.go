package relay

import (
	"net/http"

	routingdomain "codeswitch/internal/routing/domain"
	"codeswitch/internal/shared/kernel"
	"github.com/gin-gonic/gin"
)

type relayRouteOptions struct {
	endpointMutator    func(endpoint, requestedModel, effectiveModel string) (string, error)
	queryMutator       func(query map[string]string, provider routingdomain.Provider) map[string]string
	headerMutator      func(headers map[string]string, provider routingdomain.Provider) map[string]string
	bodyModelFormatter func(model string) string
	forceBodyRewrite   bool
}

type RelayContext struct {
	GinCtx         *gin.Context
	Platform       kernel.Platform
	Endpoint       string
	BodyBytes      []byte
	RequestedModel string
	IsStream       bool
	Query          map[string]string
	Headers        map[string]string
	Profile        routingdomain.RouteProfile
	RouteOptions   *relayRouteOptions
}

type ForwardContext struct {
	GinCtx    *gin.Context
	Platform  kernel.Platform
	Provider  *routingdomain.Provider
	Endpoint  string
	Query     map[string]string
	Headers   map[string]string
	BodyBytes []byte
	IsStream  bool
	Model     string
}

type PlatformHandler interface {
	ExtractRequestInfo(path string, bodyBytes []byte, query, headers map[string]string) (
		endpoint string,
		requestedModel string,
		isStream bool,
		routeOptions *relayRouteOptions,
	)
}

func cloneHeaders(header http.Header) map[string]string {
	cloned := make(map[string]string, len(header))
	for key, values := range header {
		if len(values) > 0 {
			cloned[key] = values[len(values)-1]
		}
	}
	return cloned
}

func cloneMap(m map[string]string) map[string]string {
	cloned := make(map[string]string, len(m))
	for k, v := range m {
		cloned[k] = v
	}
	return cloned
}

func flattenQuery(values map[string][]string) map[string]string {
	query := make(map[string]string, len(values))
	for key, items := range values {
		if len(items) > 0 {
			query[key] = items[len(items)-1]
		}
	}
	return query
}
