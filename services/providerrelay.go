package services

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
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
	requestLogCh       chan *ReqeustLog
	sessionUpdateCh    chan sessionUpdateRequest
	shutdown           chan struct{}
	backgroundWG       sync.WaitGroup
	shutdownOnce       sync.Once
}

func NewProviderRelayService(providerService *ProviderService, appSettingsService *AppSettingsService, sessionService *SessionService, addr string) *ProviderRelayService {
	if addr == "" {
		addr = ":18100"
	}

	home, _ := os.UserHomeDir()
	dbDir := filepath.Join(home, ".code-switch")
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		log.Printf("创建数据库目录失败: %v\n", err)
	}
	appDBPath := filepath.Join(dbDir, "app.db")
	requestLogDBPath := filepath.Join(dbDir, "request_log.db")
	sessionDBPath := filepath.Join(dbDir, "session.db")
	appDSN := fmt.Sprintf("%s?cache=shared&mode=rwc&_journal_mode=WAL&_busy_timeout=5000", appDBPath)
	requestLogDSN := fmt.Sprintf("%s?cache=shared&mode=rwc&_journal_mode=WAL&_busy_timeout=5000", requestLogDBPath)
	sessionDSN := fmt.Sprintf("%s?cache=shared&mode=rwc&_journal_mode=WAL&_busy_timeout=5000", sessionDBPath)

	if err := xdb.Inits([]xdb.Config{
		{
			Name:        CoreDBName,
			Driver:      "sqlite",
			DSN:         appDSN,
			MaxOpenConn: 1,
			MaxIdleConn: 1,
		},
		{
			Name:        RequestLogDBName,
			Driver:      "sqlite",
			DSN:         requestLogDSN,
			MaxOpenConn: 4,
			MaxIdleConn: 4,
		},
		{
			Name:        SessionDBName,
			Driver:      "sqlite",
			DSN:         sessionDSN,
			MaxOpenConn: 5,
			MaxIdleConn: 2,
		},
	}); err != nil {
		log.Printf("初始化数据库失败: %v\n", err)
	} else {
		if err := ensureRequestLogTable(RequestLogDBName); err != nil {
			log.Printf("初始化 request_log 表失败: %v\n", err)
		}
		if err := ensureSessionBindingTable(SessionDBName); err != nil {
			log.Printf("初始化 session_provider_binding 表失败: %v\n", err)
		}
		configureSQLitePragmas(CoreDBName)
		configureSQLitePragmas(RequestLogDBName)
		configureSQLitePragmas(SessionDBName)
		if err := migrateLegacyTables(requestLogDBPath, sessionDBPath); err != nil {
			log.Printf("迁移历史数据失败: %v\n", err)
		}
		if err := cleanupOldRequestLogs(RequestLogDBName, requestLogRetentionDays); err != nil {
			log.Printf("启动时清理历史 request_log 失败: %v\n", err)
		}
	}

	prs := &ProviderRelayService{
		providerService:    providerService,
		appSettingsService: appSettingsService,
		sessionService:     sessionService,
		sessionCache:       NewSessionCache(sessionService),
		addr:               addr,
		requestLogCh:       make(chan *ReqeustLog, requestLogBufferSize),
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

	for _, kind := range []string{"claude", "codex"} {
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

func (prs *ProviderRelayService) registerRoutes(router gin.IRouter) {
	router.POST("/v1/messages", prs.proxyHandler("claude", "/v1/messages"))
	router.POST("/responses", prs.proxyHandler("codex", "/responses"))
}

func (prs *ProviderRelayService) proxyHandler(kind string, endpoint string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var bodyBytes []byte
		if c.Request.Body != nil {
			data, err := io.ReadAll(c.Request.Body)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
				return
			}
			bodyBytes = data
			c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}

		isStream := gjson.GetBytes(bodyBytes, "stream").Bool()
		requestedModel := gjson.GetBytes(bodyBytes, "model").String()

		// 如果未指定模型，记录警告但不拦截
		if requestedModel == "" {
			log.Printf("[WARN] 请求未指定模型名\n")
		}

		// 读取应用设置
		appSettings, err := prs.appSettingsService.GetAppSettings()
		if err != nil {
			log.Printf("[ERROR] 无法读取应用设置: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load app settings"})
			return
		}

		query := flattenQuery(c.Request.URL.Query())
		clientHeaders := cloneHeaders(c.Request.Header)

		// 加载 providers（一次性加载，避免后续重复 I/O）
		providers, err := prs.providerService.LoadProviders(kind)
		if err != nil {
			log.Printf("[ERROR] 加载 providers 失败: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load providers"})
			return
		}

		// 使用手动会话路由模式
		log.Printf("[INFO] 使用手动会话路由模式\n")
		ok, err := prs.routeToManualProvider(c, kind, endpoint, bodyBytes, requestedModel, isStream, query, clientHeaders, appSettings, providers)
		if !ok {
			errorMsg := "手动路由失败"
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
	headers["Authorization"] = fmt.Sprintf("Bearer %s", provider.APIKey)
	if _, ok := headers["Accept"]; !ok {
		headers["Accept"] = "application/json"
	}

	requestLog := &ReqeustLog{
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
		_, copyErr := resp.ToHttpResponseWriter(c.Writer, ReqeustLogHook(c, kind, requestLog))
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

func ensureRequestLogColumn(db *sql.DB, column string, definition string) error {
	query := fmt.Sprintf("SELECT COUNT(*) FROM pragma_table_info('request_log') WHERE name = '%s'", column)
	var count int
	if err := db.QueryRow(query).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		alter := fmt.Sprintf("ALTER TABLE request_log ADD COLUMN %s %s", column, definition)
		if _, err := db.Exec(alter); err != nil {
			return err
		}
	}
	return nil
}

func ensureRequestLogTable(dbName string) error {
	db, err := xdb.DB(dbName)
	if err != nil {
		return err
	}
	return ensureRequestLogTableWithDB(db)
}

func ensureSessionBindingTable(dbName string) error {
	db, err := xdb.DB(dbName)
	if err != nil {
		return err
	}
	return ensureSessionBindingTableWithDB(db)
}

func ensureSessionBindingTableWithDB(db *sql.DB) error {
	const createTableSQL = `CREATE TABLE IF NOT EXISTS session_provider_binding (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		platform TEXT NOT NULL,
		session_id TEXT NOT NULL,
		provider_name TEXT NOT NULL,
		last_success_at DATETIME NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(platform, session_id)
	)`

	if _, err := db.Exec(createTableSQL); err != nil {
		return err
	}

	const createIndexSQL = `CREATE INDEX IF NOT EXISTS idx_session_lookup
		ON session_provider_binding(platform, session_id, last_success_at)`

	if _, err := db.Exec(createIndexSQL); err != nil {
		return err
	}
	extraIndexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_session_cleanup ON session_provider_binding(platform, last_success_at)",
		"CREATE INDEX IF NOT EXISTS idx_session_provider_lookup ON session_provider_binding(platform, provider_name, last_success_at DESC)",
	}
	for _, stmt := range extraIndexes {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}

	return nil
}

func ensureRequestLogTableWithDB(db *sql.DB) error {
	const createTableSQL = `CREATE TABLE IF NOT EXISTS request_log (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		platform TEXT,
		model TEXT,
		provider TEXT,
		http_code INTEGER,
		input_tokens INTEGER,
		output_tokens INTEGER,
		cache_create_tokens INTEGER,
		cache_read_tokens INTEGER,
		reasoning_tokens INTEGER,
		is_stream INTEGER DEFAULT 0,
		duration_sec REAL DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`

	if _, err := db.Exec(createTableSQL); err != nil {
		return err
	}

	if err := ensureRequestLogColumn(db, "created_at", "DATETIME DEFAULT CURRENT_TIMESTAMP"); err != nil {
		return err
	}
	if err := ensureRequestLogColumn(db, "is_stream", "INTEGER DEFAULT 0"); err != nil {
		return err
	}
	if err := ensureRequestLogColumn(db, "duration_sec", "REAL DEFAULT 0"); err != nil {
		return err
	}
	indexStatements := []string{
		"CREATE INDEX IF NOT EXISTS idx_request_log_platform_created_at ON request_log(platform, created_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_request_log_provider_created_at ON request_log(provider, created_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_request_log_platform_provider_created_at ON request_log(platform, provider, created_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_request_log_created_at ON request_log(created_at)",
	}
	for _, stmt := range indexStatements {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}

	return nil
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

func (prs *ProviderRelayService) enqueueRequestLog(logEntry *ReqeustLog) {
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

func (prs *ProviderRelayService) writeRequestLogSync(logEntry *ReqeustLog) {
	if logEntry == nil {
		return
	}
	if _, err := requestLogModel().Insert(recordFromRequestLog(logEntry)); err != nil {
		log.Printf("写入 request_log 失败: %v\n", err)
	}
}

func recordFromRequestLog(logEntry *ReqeustLog) xdb.Record {
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

func configureSQLitePragmas(dbName string) {
	db, err := xdb.DB(dbName)
	if err != nil {
		return
	}
	statements := []string{
		"PRAGMA synchronous = NORMAL",
		"PRAGMA journal_size_limit = 67108864",
		"PRAGMA wal_autocheckpoint = 1000",
	}
	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			log.Printf("设置 SQLite PRAGMA 失败 (%s): %v\n", stmt, err)
		}
	}
}

func migrateLegacyTables(requestLogDBPath, sessionDBPath string) error {
	var errs []error
	if err := migrateTable(CoreDBName, RequestLogDBName, requestLogDBPath, "request_log"); err != nil {
		errs = append(errs, fmt.Errorf("request_log: %w", err))
	}
	if err := migrateTable(CoreDBName, SessionDBName, sessionDBPath, "session_provider_binding"); err != nil {
		errs = append(errs, fmt.Errorf("session_provider_binding: %w", err))
	}
	return errors.Join(errs...)
}

func migrateTable(srcDBName, destDBName, destPath, table string) error {
	if destPath == "" {
		return fmt.Errorf("目标数据库路径为空: %s", table)
	}
	srcDB, err := xdb.DB(srcDBName)
	if err != nil {
		return err
	}
	destDB, err := xdb.DB(destDBName)
	if err != nil {
		return err
	}

	exists, err := sqliteTableExists(srcDB, table)
	if err != nil || !exists {
		return err
	}

	destRows, err := sqliteRowCount(destDB, table)
	if err != nil {
		return err
	}
	if destRows > 0 {
		return nil
	}

	srcRows, err := sqliteRowCount(srcDB, table)
	if err != nil {
		return err
	}
	if srcRows == 0 {
		return nil
	}

	alias := fmt.Sprintf("%s_migrate", table)
	if err := attachDatabase(srcDB, alias, destPath); err != nil {
		return err
	}
	defer detachDatabase(srcDB, alias)

	stmt := fmt.Sprintf("INSERT INTO %s.%s SELECT * FROM main.%s", alias, table, table)
	if _, err := srcDB.Exec(stmt); err != nil {
		return err
	}
	log.Printf("迁移 %d 条 %s 记录到新数据库\n", srcRows, table)
	return nil
}

func sqliteTableExists(db *sql.DB, table string) (bool, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?", table).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func sqliteRowCount(db *sql.DB, table string) (int64, error) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)
	var count int64
	if err := db.QueryRow(query).Scan(&count); err != nil {
		if isNoSuchTableErr(err) {
			return 0, nil
		}
		return 0, err
	}
	return count, nil
}

func attachDatabase(db *sql.DB, alias, path string) error {
	if alias == "" || path == "" {
		return fmt.Errorf("attach 参数无效")
	}
	escaped := strings.ReplaceAll(path, "'", "''")
	stmt := fmt.Sprintf("ATTACH DATABASE '%s' AS %s", escaped, alias)
	_, err := db.Exec(stmt)
	return err
}

func detachDatabase(db *sql.DB, alias string) {
	if alias == "" {
		return
	}
	if _, err := db.Exec(fmt.Sprintf("DETACH DATABASE %s", alias)); err != nil {
		log.Printf("分离数据库 %s 失败: %v\n", alias, err)
	}
}

func ReqeustLogHook(c *gin.Context, kind string, usage *ReqeustLog) func(data []byte) (bool, []byte) { // SSE 钩子：累计字节和解析 token 用量
	return func(data []byte) (bool, []byte) {
		payload := strings.TrimSpace(string(data))

		parserFn := ClaudeCodeParseTokenUsageFromResponse
		if kind == "codex" {
			parserFn = CodexParseTokenUsageFromResponse
		}
		parseEventPayload(payload, parserFn, usage)

		return true, data
	}
}

func parseEventPayload(payload string, parser func(string, *ReqeustLog), usage *ReqeustLog) {
	lines := strings.Split(payload, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "data:") {
			parser(strings.TrimPrefix(line, "data: "), usage)
		}
	}
}

type ReqeustLog struct {
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
func ClaudeCodeParseTokenUsageFromResponse(data string, usage *ReqeustLog) {
	usage.InputTokens += int(gjson.Get(data, "message.usage.input_tokens").Int())
	usage.OutputTokens += int(gjson.Get(data, "message.usage.output_tokens").Int())
	usage.CacheCreateTokens += int(gjson.Get(data, "message.usage.cache_creation_input_tokens").Int())
	usage.CacheReadTokens += int(gjson.Get(data, "message.usage.cache_read_input_tokens").Int())

	usage.InputTokens += int(gjson.Get(data, "usage.input_tokens").Int())
	usage.OutputTokens += int(gjson.Get(data, "usage.output_tokens").Int())
	log.Println("claudecode response data -->", data, fmt.Sprintf("%v", usage))
}

// codex usage parser
func CodexParseTokenUsageFromResponse(data string, usage *ReqeustLog) {
	usage.InputTokens += int(gjson.Get(data, "response.usage.input_tokens").Int())
	usage.OutputTokens += int(gjson.Get(data, "response.usage.output_tokens").Int())
	usage.CacheReadTokens += int(gjson.Get(data, "response.usage.input_tokens_details.cached_tokens").Int())
	usage.ReasoningTokens += int(gjson.Get(data, "response.usage.output_tokens_details.reasoning_tokens").Int())
	log.Println("codex response data -->", data, fmt.Sprintf("%v", usage))
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
	if kind == "claude" {
		// Claude Code 从请求体中提取 metadata.user_id
		sessionID := gjson.GetBytes(bodyBytes, "metadata.user_id").String()
		return sessionID
	} else if kind == "codex" {
		// Codex 从请求头中提取 session_id
		sessionID := c.GetHeader("session_id")
		return sessionID
	}
	return ""
}

// routeToManualProvider 手动路由模式下的请求处理
// 返回值：(是否成功, 错误信息)
func (prs *ProviderRelayService) routeToManualProvider(
	c *gin.Context,
	kind string,
	endpoint string,
	bodyBytes []byte,
	requestedModel string,
	isStream bool,
	query map[string]string,
	clientHeaders map[string]string,
	appSettings AppSettings,
	providers []Provider,
) (bool, error) {
	sessionID := prs.extractSessionID(c, kind, bodyBytes)
	log.Printf("[INFO] [手动路由] 会话ID: %s\n", sessionID)

	var targetProviderName string
	boundProviderName := ""
	sessionAlreadyBound := false

	// 步骤1：检查会话是否已绑定（使用缓存减少数据库查询）
	if sessionID != "" {
		boundProvider, err := prs.sessionCache.GetSessionProvider(kind, sessionID)
		if err != nil {
			log.Printf("[WARN] 查询会话绑定失败: %v\n", err)
		} else if boundProvider != "" {
			targetProviderName = boundProvider
			boundProviderName = boundProvider
			sessionAlreadyBound = true
			log.Printf("[INFO] [手动路由] 会话已绑定到供应商: %s\n", boundProvider)
		}
	}

	// 步骤2：如果未绑定或已过期，使用默认供应商
	if targetProviderName == "" {
		if kind == "claude" {
			targetProviderName = appSettings.DefaultClaudeProvider
		} else {
			targetProviderName = appSettings.DefaultCodexProvider
		}

		if targetProviderName == "" {
			return false, fmt.Errorf("未配置默认供应商")
		}

		log.Printf("[INFO] [手动路由] 使用默认供应商: %s\n", targetProviderName)
	}

	// 步骤3：从传入的 providers 中查找目标供应商（避免重复文件 I/O）
	var provider *Provider
	for i := range providers {
		if providers[i].Name == targetProviderName {
			provider = &providers[i]
			break
		}
	}
	if provider == nil {
		return false, fmt.Errorf("供应商 %s 不存在", targetProviderName)
	}

	// 步骤4：验证供应商配置
	if !provider.Enabled {
		return false, fmt.Errorf("供应商 %s 已被禁用", provider.Name)
	}

	if provider.APIURL == "" || provider.APIKey == "" {
		return false, fmt.Errorf("供应商 %s 配置不完整", provider.Name)
	}

	if errs := provider.ValidateConfiguration(); len(errs) > 0 {
		return false, fmt.Errorf("供应商 %s 配置验证失败: %v", provider.Name, errs)
	}

	// 步骤5：检查模型支持
	if requestedModel != "" && !provider.IsModelSupported(requestedModel) {
		return false, fmt.Errorf("供应商 %s 不支持模型 %s", provider.Name, requestedModel)
	}

	// 步骤6：获取有效模型名（可能需要映射）
	effectiveModel := provider.GetEffectiveModel(requestedModel)
	currentBodyBytes := bodyBytes

	if effectiveModel != requestedModel && requestedModel != "" {
		log.Printf("[INFO] [手动路由] 映射模型: %s -> %s\n", requestedModel, effectiveModel)
		modifiedBody, err := ReplaceModelInRequestBody(bodyBytes, effectiveModel)
		if err != nil {
			return false, fmt.Errorf("替换模型名失败: %w", err)
		}
		currentBodyBytes = modifiedBody
	}

	// 步骤7：转发请求
	log.Printf("[INFO] [手动路由] 转发请求到供应商: %s | 模型: %s\n", provider.Name, effectiveModel)
	startTime := time.Now()
	ok, err := prs.forwardRequest(c, kind, *provider, endpoint, query, clientHeaders, currentBodyBytes, isStream, effectiveModel)
	duration := time.Since(startTime)

	if ok {
		log.Printf("[INFO] [手动路由] ✓ 请求成功: %s | 耗时: %.2fs\n", provider.Name, duration.Seconds())

		// 步骤8：请求成功后的处理
		if sessionID != "" {
			if !sessionAlreadyBound {
				if err := prs.sessionCache.BindSessionToProvider(kind, sessionID, provider.Name); err != nil {
					log.Printf("[WARN] 绑定会话失败: %v\n", err)
				} else {
					boundProviderName = provider.Name
					sessionAlreadyBound = true
				}
			} else if boundProviderName != "" {
				// 如果已绑定，异步更新最后成功时间
				select {
				case prs.sessionUpdateCh <- sessionUpdateRequest{platform: kind, sessionID: sessionID}:
				default:
					// 缓冲已满，回退为同步更新
					if err := prs.sessionCache.UpdateSessionSuccess(kind, sessionID); err != nil {
						log.Printf("[WARN] 更新会话时间失败: %v\n", err)
					}
				}
			}
		}

		return true, nil
	}

	// 步骤9：请求失败
	errorMsg := "未知错误"
	if err != nil {
		errorMsg = err.Error()
	}
	log.Printf("[WARN] [手动路由] ✗ 请求失败: %s | 错误: %s | 耗时: %.2fs\n",
		provider.Name, errorMsg, duration.Seconds())

	return false, err
}
