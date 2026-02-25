package services

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/daodao97/xgo/xrequest"
	"github.com/gin-gonic/gin"
)

// readRequestBody 读取并重置请求体
func (prs *ProviderRelayService) readRequestBody(c *gin.Context) ([]byte, error) {
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

func (prs *ProviderRelayService) registerRoutes(router gin.IRouter) {
	router.POST("/v1/messages", prs.proxyHandler(PlatformClaude, "/v1/messages"))
	router.POST("/responses", prs.proxyHandler(PlatformCodex, "/responses"))
	router.POST("/gemini/v1beta/*proxyPath", prs.proxyHandler(PlatformGemini, ""))
	router.POST("/gemini/v1/*proxyPath", prs.proxyHandler(PlatformGemini, ""))
}

func (prs *ProviderRelayService) proxyHandler(platform Platform, defaultEndpoint string) gin.HandlerFunc {
	handler := GetPlatformHandler(platform, defaultEndpoint)

	return func(c *gin.Context) {
		// 1. 读取请求体
		bodyBytes, err := prs.readRequestBody(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		// 2. 提取通用信息
		query := flattenQuery(c.Request.URL.Query())
		headers := cloneHeaders(c.Request.Header)

		// 3. 委托给平台处理器提取特定信息
		endpoint, requestedModel, isStream, routeOptions := handler.ExtractRequestInfo(
			c.Request.URL.Path, bodyBytes, query, headers,
		)

		// 4. 记录警告（可选）
		if requestedModel == "" {
			log.Printf("[WARN] 请求未指定模型名\n")
		}

		// 5. 加载配置
		appSettings, providers, err := prs.loadConfig(platform.String())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// 6. 构建 RelayContext 并路由
		ctx := &RelayContext{
			GinCtx:         c,
			Platform:       platform,
			Endpoint:       endpoint,
			BodyBytes:      bodyBytes,
			RequestedModel: requestedModel,
			IsStream:       isStream,
			Query:          query,
			Headers:        headers,
			AppSettings:    appSettings,
			Providers:      providers,
			RouteOptions:   routeOptions,
		}

		if ok, err := prs.routeToManualProvider(ctx); !ok {
			errorMsg := "路由失败"
			if err != nil {
				errorMsg = err.Error()
			}
			c.JSON(http.StatusBadGateway, gin.H{"error": errorMsg})
		}
	}
}

func (prs *ProviderRelayService) forwardRequest(
	c *gin.Context,
	platform Platform,
	provider Provider,
	endpoint string,
	query map[string]string,
	clientHeaders map[string]string,
	bodyBytes []byte,
	isStream bool,
	model string,
) (bool, error) {
	kind := platform.String()
	targetURL := joinURL(provider.APIURL, endpoint)
	headers := cloneMap(clientHeaders)
	if _, ok := headers["Authorization"]; ok && provider.APIKey != "" {
		headers["Authorization"] = fmt.Sprintf("Bearer %s", provider.APIKey)
	}
	if _, ok := headers["Accept"]; !ok {
		headers["Accept"] = "application/json"
	}

	requestLog := &RequestLog{
		Platform: kind,
		Provider: provider.Name,
		Model:    model,
		IsStream: isStream,
	}
	start := time.Now()
	defer func() {
		requestLog.DurationSec = time.Since(start).Seconds()
		prs.enqueueRequestLog(requestLog)
	}()

	req := xrequest.New().
		SetHeaders(headers).
		SetQueryParams(query).
		SetRetry(1, 500*time.Millisecond).
		SetTimeout(900 * time.Second)

	reqBody := bytes.NewReader(bodyBytes)
	req = req.SetBody(reqBody)

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

func joinURL(base string, endpoint string) string {
	base = strings.TrimSuffix(base, "/")
	endpoint = "/" + strings.TrimPrefix(endpoint, "/")
	return base + endpoint
}
