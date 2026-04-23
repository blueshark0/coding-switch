package relay

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	observabilitydomain "codeswitch/internal/observability/domain"
	routingdomain "codeswitch/internal/routing/domain"
	"codeswitch/internal/shared/kernel"
	"github.com/daodao97/xgo/xrequest"
	"github.com/gin-gonic/gin"
)

func (s *Server) readRequestBody(c *gin.Context) ([]byte, error) {
	if c.Request.Body == nil {
		return nil, nil
	}
	data, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil, err
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(data))
	return data, nil
}

func (s *Server) registerRoutes(router gin.IRouter) {
	router.POST("/v1/messages", s.proxyHandler(kernel.PlatformClaude, "/v1/messages"))
	router.POST("/responses", s.proxyHandler(kernel.PlatformCodex, "/responses"))
	router.POST("/responses/compact", s.proxyHandler(kernel.PlatformCodex, "/responses/compact"))
	router.POST("/gemini/v1beta/*proxyPath", s.proxyHandler(kernel.PlatformGemini, ""))
	router.POST("/gemini/v1/*proxyPath", s.proxyHandler(kernel.PlatformGemini, ""))
}

func (s *Server) proxyHandler(platform kernel.Platform, defaultEndpoint string) gin.HandlerFunc {
	handler := GetPlatformHandler(platform, defaultEndpoint)
	return func(c *gin.Context) {
		bodyBytes, err := s.readRequestBody(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}
		query := flattenQuery(c.Request.URL.Query())
		headers := cloneHeaders(c.Request.Header)
		requestMeta := handler.ExtractRequestMeta(c.Request.URL.Path, bodyBytes, query, headers)
		if requestMeta.RequestedModel == "" {
			relayDebugf("request missing model: platform=%s path=%s", platform, c.Request.URL.Path)
		}
		profile, err := s.loadProfile(platform.String())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		ctx := &RelayContext{
			GinCtx:      c,
			Platform:    platform,
			BodyBytes:   bodyBytes,
			Query:       query,
			Headers:     headers,
			Profile:     profile,
			RequestMeta: requestMeta,
		}
		if ok, err := s.routeToManualProvider(ctx); !ok {
			errorMsg := "路由失败"
			if err != nil {
				errorMsg = err.Error()
			}
			c.JSON(http.StatusBadGateway, gin.H{"error": errorMsg})
		}
	}
}

func (s *Server) forwardRequest(
	c *gin.Context,
	platform kernel.Platform,
	provider routingdomain.Provider,
	endpoint string,
	query map[string]string,
	clientHeaders map[string]string,
	bodyBytes []byte,
	isStream bool,
	model string,
) (bool, error) {
	targetURL := joinURL(provider.APIURL, endpoint)
	headers := cloneMap(clientHeaders)
	if _, ok := headers["Authorization"]; ok && provider.APIKey != "" {
		headers["Authorization"] = fmt.Sprintf("Bearer %s", provider.APIKey)
	}
	if _, ok := headers["Accept"]; !ok {
		headers["Accept"] = "application/json"
	}
	requestLog := &observabilitydomain.RequestLog{
		Platform: platform.String(),
		Provider: provider.Name,
		Model:    model,
		IsStream: isStream,
	}
	start := time.Now()
	defer func() {
		requestLog.DurationSec = time.Since(start).Seconds()
		s.enqueueRequestLog(requestLog)
	}()
	req := xrequest.New().
		SetHeaders(headers).
		SetQueryParams(query).
		SetRetry(1, 500*time.Millisecond).
		SetTimeout(900 * time.Second)
	req = req.SetBody(bytes.NewReader(bodyBytes))
	if proxyURL := s.getProxyURLForProvider(&provider); proxyURL != "" {
		req = req.SetProxy(proxyURL)
	}
	resp, err := req.Post(targetURL)
	if err != nil {
		return false, err
	}
	if resp == nil {
		return false, fmt.Errorf("empty response")
	}
	if resp.Error() != nil {
		return false, resp.Error()
	}
	status := resp.StatusCode()
	requestLog.HttpCode = status
	if status >= http.StatusOK && status < http.StatusMultipleChoices {
		_, copyErr := resp.ToHttpResponseWriter(c.Writer, RequestLogHook(platform, requestLog))
		return copyErr == nil, copyErr
	}
	return false, fmt.Errorf("upstream status %d", status)
}

func joinURL(base string, endpoint string) string {
	base = strings.TrimSuffix(base, "/")
	endpoint = "/" + strings.TrimPrefix(endpoint, "/")
	return base + endpoint
}
