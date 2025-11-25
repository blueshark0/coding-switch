package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/daodao97/xgo/xdb"
	"github.com/daodao97/xgo/xrequest"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	_ "modernc.org/sqlite"
)

const (
	requestLogBufferSize      = 1024
	requestLogBatchSize       = 50
	requestLogFlushInterval   = 500 * time.Millisecond
	requestLogCleanupInterval = 6 * time.Hour
	requestLogRetentionDays   = 60

	sessionUpdateBufferSize    = 100
	sessionUpdateBatchSize     = 20
	sessionUpdateFlushInterval = 200 * time.Millisecond
)

type sessionUpdateRequest struct {
	platform  string
	sessionID string
}

type ProviderRelayService struct {
	providerService    *ProviderService
	appSettingsService *AppSettingsService
	sessionService     *SessionService
	sessionCache       *SessionCache
	server             *http.Server
	addr               string
	requestLogCh       chan *RequestLog
	sessionUpdateCh    chan sessionUpdateRequest
	shutdown           chan struct{}
	backgroundWG       sync.WaitGroup
	shutdownOnce       sync.Once
}

func NewProviderRelayService(providerService *ProviderService, appSettingsService *AppSettingsService, sessionService *SessionService, addr string) *ProviderRelayService {
	if addr == "" {
		addr = ":18100"
	}

	prs := &ProviderRelayService{
		providerService:    providerService,
		appSettingsService: appSettingsService,
		sessionService:     sessionService,
		sessionCache:       NewSessionCache(sessionService),
		addr:               addr,
		requestLogCh:       make(chan *RequestLog, requestLogBufferSize),
		sessionUpdateCh:    make(chan sessionUpdateRequest, sessionUpdateBufferSize),
		shutdown:           make(chan struct{}),
	}

	prs.startLogWriter()
	prs.startRequestLogRetentionTask()
	prs.startSessionUpdateWorker()

	return prs
}

func (prs *ProviderRelayService) Start() error {
	// 启动前验证配置
	if warnings := prs.validateConfig(); len(warnings) > 0 {
		log.Println("======== Provider 配置验证警告 ========")
		for _, warn := range warnings {
			log.Printf("⚠️  %s\n", warn)
		}
		log.Println("========================================")
	}

	// 配置 Gin 为 Release 模式（生产环境），减少详细日志
	// 因为已有 request_log 数据库记录，无需重复记录 HTTP 请求
	gin.SetMode(gin.ReleaseMode)

	router := gin.Default()
	prs.registerRoutes(router)

	prs.server = &http.Server{
		Addr:    prs.addr,
		Handler: router,
	}

	log.Printf("provider relay server listening on %s\n", prs.addr)

	go func() {
		if err := prs.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("provider relay server error: %v\n", err)
		}
	}()
	return nil
}

// validateConfig 验证所有 provider 的配置
// 返回警告列表（非阻塞性错误）
func (prs *ProviderRelayService) validateConfig() []string {
	warnings := make([]string, 0)

	for _, platform := range AllPlatforms() {
		kind := platform.String()
		providers, err := prs.providerService.LoadProviders(kind)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("[%s] 加载配置失败: %v", kind, err))
			continue
		}

		enabledCount := 0
		for _, p := range providers {
			if !p.Enabled {
				continue
			}
			enabledCount++

			// 验证每个启用的 provider
			if errs := p.ValidateConfiguration(); len(errs) > 0 {
				for _, errMsg := range errs {
					warnings = append(warnings, fmt.Sprintf("[%s/%s] %s", kind, p.Name, errMsg))
				}
			}

			// 检查是否配置了模型白名单或映射
			if (p.SupportedModels == nil || len(p.SupportedModels) == 0) &&
				(p.ModelMapping == nil || len(p.ModelMapping) == 0) {
				warnings = append(warnings, fmt.Sprintf(
					"[%s/%s] 未配置 supportedModels 或 modelMapping，将假设支持所有模型（可能导致降级失败）",
					kind, p.Name))
			}
		}

		if enabledCount == 0 {
			warnings = append(warnings, fmt.Sprintf("[%s] 没有启用的 provider", kind))
		}
	}

	return warnings
}

func (prs *ProviderRelayService) Stop() error {
	var err error
	if prs.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err = prs.server.Shutdown(ctx)
	}
	prs.stopBackgroundWorkers()
	return err
}

func (prs *ProviderRelayService) Addr() string {
	return prs.addr
}

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

// loadConfig 加载应用设置和 providers
func (prs *ProviderRelayService) loadConfig(kind string) (AppSettings, []Provider, error) {
	appSettings, err := prs.appSettingsService.GetAppSettings()
	if err != nil {
		return AppSettings{}, nil, fmt.Errorf("failed to load app settings: %w", err)
	}

	providers, err := prs.providerService.LoadProviders(kind)
	if err != nil {
		return AppSettings{}, nil, fmt.Errorf("failed to load providers: %w", err)
	}

	return appSettings, providers, nil
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
	kind string,
	provider Provider,
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
		SetTimeout(360 * time.Second)

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
		_, copyErr := resp.ToHttpResponseWriter(c.Writer, RequestLogHook(c, kind, requestLog))
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

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (prs *ProviderRelayService) startLogWriter() {
	if prs == nil {
		return
	}
	prs.backgroundWG.Add(1)
	go func() {
		defer prs.backgroundWG.Done()
		ticker := time.NewTicker(requestLogFlushInterval)
		defer ticker.Stop()
		batch := make([]xdb.Record, 0, requestLogBatchSize)
		flush := func() {
			if len(batch) == 0 {
				return
			}
			if _, err := requestLogModel().InsertBatch(batch); err != nil {
				log.Printf("批量写入 request_log 失败: %v\n", err)
				for _, record := range batch {
					if _, insertErr := requestLogModel().Insert(record); insertErr != nil {
						log.Printf("写入 request_log 失败: %v\n", insertErr)
					}
				}
			}
			batch = batch[:0]
		}
		for {
			select {
			case logEntry, ok := <-prs.requestLogCh:
				if !ok {
					flush()
					return
				}
				if logEntry == nil {
					continue
				}
				batch = append(batch, recordFromRequestLog(logEntry))
				if len(batch) >= requestLogBatchSize {
					flush()
				}
			case <-ticker.C:
				flush()
			case <-prs.shutdown:
				flush()
				return
			}
		}
	}()
}

func (prs *ProviderRelayService) startRequestLogRetentionTask() {
	if prs == nil {
		return
	}
	if requestLogRetentionDays <= 0 {
		return
	}
	prs.backgroundWG.Add(1)
	go func() {
		defer prs.backgroundWG.Done()
		ticker := time.NewTicker(requestLogCleanupInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := cleanupOldRequestLogs(RequestLogDBName, requestLogRetentionDays); err != nil {
					log.Printf("定时清理 request_log 失败: %v\n", err)
				}
			case <-prs.shutdown:
				return
			}
		}
	}()
}

func (prs *ProviderRelayService) startSessionUpdateWorker() {
	if prs == nil {
		return
	}
	prs.backgroundWG.Add(1)
	go func() {
		defer prs.backgroundWG.Done()
		ticker := time.NewTicker(sessionUpdateFlushInterval)
		defer ticker.Stop()
		batch := make([]sessionUpdateRequest, 0, sessionUpdateBatchSize)

		flush := func() {
			if len(batch) == 0 {
				return
			}
			// 批量更新会话时间
			for _, req := range batch {
				if err := prs.sessionCache.UpdateSessionSuccess(req.platform, req.sessionID); err != nil {
					log.Printf("[WARN] 异步更新会话时间失败: %v\n", err)
				}
			}
			batch = batch[:0]
		}

		for {
			select {
			case req, ok := <-prs.sessionUpdateCh:
				if !ok {
					flush()
					return
				}
				batch = append(batch, req)
				if len(batch) >= sessionUpdateBatchSize {
					flush()
				}
			case <-ticker.C:
				flush()
			case <-prs.shutdown:
				flush()
				return
			}
		}
	}()
}

func (prs *ProviderRelayService) stopBackgroundWorkers() {
	if prs == nil {
		return
	}
	prs.shutdownOnce.Do(func() {
		close(prs.shutdown)
		close(prs.requestLogCh)
		close(prs.sessionUpdateCh)
	})
	prs.backgroundWG.Wait()
}

func (prs *ProviderRelayService) enqueueRequestLog(logEntry *RequestLog) {
	if prs == nil || logEntry == nil {
		return
	}
	select {
	case <-prs.shutdown:
		prs.writeRequestLogSync(logEntry)
		return
	default:
	}
	select {
	case prs.requestLogCh <- logEntry:
	default:
		log.Printf("request_log 缓冲已满，回退为同步写入\n")
		prs.writeRequestLogSync(logEntry)
	}
}

func (prs *ProviderRelayService) writeRequestLogSync(logEntry *RequestLog) {
	if logEntry == nil {
		return
	}
	if _, err := requestLogModel().Insert(recordFromRequestLog(logEntry)); err != nil {
		log.Printf("写入 request_log 失败: %v\n", err)
	}
}

func recordFromRequestLog(logEntry *RequestLog) xdb.Record {
	return xdb.Record{
		"platform":            logEntry.Platform,
		"model":               logEntry.Model,
		"provider":            logEntry.Provider,
		"http_code":           logEntry.HttpCode,
		"input_tokens":        logEntry.InputTokens,
		"output_tokens":       logEntry.OutputTokens,
		"cache_create_tokens": logEntry.CacheCreateTokens,
		"cache_read_tokens":   logEntry.CacheReadTokens,
		"reasoning_tokens":    logEntry.ReasoningTokens,
		"is_stream":           boolToInt(logEntry.IsStream),
		"duration_sec":        logEntry.DurationSec,
	}
}

func cleanupOldRequestLogs(dbName string, retentionDays int) error {
	if retentionDays <= 0 {
		return nil
	}
	db, err := xdb.DB(dbName)
	if err != nil {
		return err
	}
	cutoff := fmt.Sprintf("-%d day", retentionDays)
	res, err := db.Exec("DELETE FROM request_log WHERE created_at < datetime('now', ?)", cutoff)
	if err != nil {
		if isNoSuchTableErr(err) {
			return nil
		}
		return err
	}
	if rows, err := res.RowsAffected(); err == nil && rows > 0 {
		log.Printf("清理 %d 条过期 request_log 记录\n", rows)
	}
	if _, err := db.Exec("PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
		log.Printf("执行 wal_checkpoint 失败: %v\n", err)
	}
	if _, err := db.Exec("PRAGMA optimize"); err != nil {
		log.Printf("执行 PRAGMA optimize 失败: %v\n", err)
	}
	return nil
}

func RequestLogHook(c *gin.Context, kind string, usage *RequestLog) func(data []byte) (bool, []byte) { // SSE 钩子：累计字节和解析 token 用量
	return func(data []byte) (bool, []byte) {
		payload := strings.TrimSpace(string(data))

		parserFn := ClaudeCodeParseTokenUsageFromResponse
		if kind == PlatformCodex.String() {
			parserFn = CodexParseTokenUsageFromResponse
		} else if kind == PlatformGemini.String() {
			parserFn = GeminiParseTokenUsageFromResponse
		}
		parseEventPayload(payload, parserFn, usage)

		return true, data
	}
}

func parseEventPayload(payload string, parser func(string, *RequestLog), usage *RequestLog) {
	lines := strings.Split(payload, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "data:") {
			parser(strings.TrimPrefix(line, "data: "), usage)
		}
	}
}

type RequestLog struct {
	ID                int64   `json:"id"`
	Platform          string  `json:"platform"` // claude code or codex
	Model             string  `json:"model"`
	Provider          string  `json:"provider"` // provider name
	HttpCode          int     `json:"http_code"`
	InputTokens       int     `json:"input_tokens"`
	OutputTokens      int     `json:"output_tokens"`
	CacheCreateTokens int     `json:"cache_create_tokens"`
	CacheReadTokens   int     `json:"cache_read_tokens"`
	ReasoningTokens   int     `json:"reasoning_tokens"`
	IsStream          bool    `json:"is_stream"`
	DurationSec       float64 `json:"duration_sec"`
	CreatedAt         string  `json:"created_at"`
	InputCost         float64 `json:"input_cost"`
	OutputCost        float64 `json:"output_cost"`
	CacheCreateCost   float64 `json:"cache_create_cost"`
	CacheReadCost     float64 `json:"cache_read_cost"`
	Ephemeral5mCost   float64 `json:"ephemeral_5m_cost"`
	Ephemeral1hCost   float64 `json:"ephemeral_1h_cost"`
	TotalCost         float64 `json:"total_cost"`
	HasPricing        bool    `json:"has_pricing"`
}

// claude code usage parser
func ClaudeCodeParseTokenUsageFromResponse(data string, usage *RequestLog) {
	usage.InputTokens += int(gjson.Get(data, "message.usage.input_tokens").Int())
	usage.OutputTokens += int(gjson.Get(data, "message.usage.output_tokens").Int())
	usage.CacheCreateTokens += int(gjson.Get(data, "message.usage.cache_creation_input_tokens").Int())
	usage.CacheReadTokens += int(gjson.Get(data, "message.usage.cache_read_input_tokens").Int())

	usage.InputTokens += int(gjson.Get(data, "usage.input_tokens").Int())
	usage.OutputTokens += int(gjson.Get(data, "usage.output_tokens").Int())
}

// codex usage parser
func CodexParseTokenUsageFromResponse(data string, usage *RequestLog) {
	usage.InputTokens += int(gjson.Get(data, "response.usage.input_tokens").Int())
	usage.OutputTokens += int(gjson.Get(data, "response.usage.output_tokens").Int())
	usage.CacheReadTokens += int(gjson.Get(data, "response.usage.input_tokens_details.cached_tokens").Int())
	usage.ReasoningTokens += int(gjson.Get(data, "response.usage.output_tokens_details.reasoning_tokens").Int())
}

// gemini usage parser
func GeminiParseTokenUsageFromResponse(data string, usage *RequestLog) {
	usage.InputTokens += int(gjson.Get(data, "usageMetadata.promptTokenCount").Int())
	usage.OutputTokens += int(gjson.Get(data, "usageMetadata.candidatesTokenCount").Int())
	usage.CacheReadTokens += int(gjson.Get(data, "usageMetadata.cachedContentTokenCount").Int())
	if total := gjson.Get(data, "usageMetadata.totalTokenCount").Int(); total > 0 {
		// 如果提供了 totalTokenCount，用它来补充输出 token 统计
		remaining := int(total) - usage.InputTokens
		if remaining > usage.OutputTokens {
			usage.OutputTokens = remaining
		}
	}
}

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

// extractSessionID 从请求中提取会话标识
// Claude Code: metadata.user_id (请求体)
// Codex: session_id (请求头)
func (prs *ProviderRelayService) extractSessionID(c *gin.Context, kind string, bodyBytes []byte) string {
	if kind == PlatformClaude.String() {
		// Claude Code 从请求体中提取 metadata.user_id
		sessionID := gjson.GetBytes(bodyBytes, "metadata.user_id").String()
		return sessionID
	} else if kind == PlatformCodex.String() {
		// Codex 从请求头中提取 session_id
		sessionID := c.GetHeader("session_id")
		return sessionID
	}
	if kind == PlatformGemini.String() {
		// Gemini 请求优先读取 header 中的 session_id（便于保持会话路由一致）
		sessionID := c.GetHeader("x-gemini-api-privileged-user-id")
		if sessionID != "" {
			return sessionID
		}
		return c.GetHeader("x-session-id")
	}
	return ""
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
	sessionID := prs.extractSessionID(ctx.GinCtx, kind, ctx.BodyBytes)
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
	if errs := provider.ValidateConfiguration(); len(errs) > 0 {
		return fmt.Errorf("供应商 %s 配置验证失败: %v", provider.Name, errs)
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
		kind,
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
		select {
		case prs.sessionUpdateCh <- sessionUpdateRequest{platform: kind, sessionID: sessionID}:
		default:
			if err := prs.sessionCache.UpdateSessionSuccess(kind, sessionID); err != nil {
				log.Printf("[WARN] 更新会话时间失败: %v\n", err)
			}
		}
	}
}
