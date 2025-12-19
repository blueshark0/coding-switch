package services

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
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
	providerService     *ProviderService
	appSettingsService  *AppSettingsService
	sessionService      *SessionService
	sessionCache        *SessionCache
	server              *http.Server
	addr                string
	requestLogWorker    *BackgroundWorker[*RequestLog]
	sessionUpdateWorker *BackgroundWorker[sessionUpdateRequest]
	shutdownCh          chan struct{}
	backgroundWG        sync.WaitGroup
	shutdownOnce        sync.Once
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
		shutdownCh:         make(chan struct{}),
	}

	prs.requestLogWorker = NewBackgroundWorker[*RequestLog](WorkerConfig{
		Name:          "request_log",
		BufferSize:    requestLogBufferSize,
		BatchSize:     requestLogBatchSize,
		FlushInterval: requestLogFlushInterval,
	}, &RequestLogWriter{})
	prs.requestLogWorker.Start()

	prs.sessionUpdateWorker = NewBackgroundWorker[sessionUpdateRequest](WorkerConfig{
		Name:          "session_update",
		BufferSize:    sessionUpdateBufferSize,
		BatchSize:     sessionUpdateBatchSize,
		FlushInterval: sessionUpdateFlushInterval,
	}, NewSessionUpdateProcessor(prs.sessionCache))
	prs.sessionUpdateWorker.Start()

	prs.startRequestLogRetentionTask()

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
