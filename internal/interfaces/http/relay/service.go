package relay

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	observabilitydomain "codeswitch/internal/observability/domain"
	observabilityinfra "codeswitch/internal/observability/infrastructure"
	routingapp "codeswitch/internal/routing/application"
	sessioninfra "codeswitch/internal/sessions/infrastructure"
	"codeswitch/services"
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

type Server struct {
	routingService      *routingapp.Service
	sessionService      *services.SessionService
	sessionCache        *sessioninfra.Cache
	server              *http.Server
	addr                string
	requestLogWorker    *services.BackgroundWorker[*observabilitydomain.RequestLog]
	sessionUpdateWorker *services.BackgroundWorker[sessionUpdateRequest]
	shutdownCh          chan struct{}
	backgroundWG        sync.WaitGroup
	shutdownOnce        sync.Once
}

func NewServer(routingService *routingapp.Service, sessionService *services.SessionService, addr string) *Server {
	if addr == "" {
		addr = ":18100"
	}
	server := &Server{
		routingService: routingService,
		sessionService: sessionService,
		sessionCache:   sessioninfra.NewCache(sessionService),
		addr:           addr,
		shutdownCh:     make(chan struct{}),
	}
	server.requestLogWorker = services.NewBackgroundWorker[*observabilitydomain.RequestLog](services.WorkerConfig{
		Name:          "request_log",
		BufferSize:    requestLogBufferSize,
		BatchSize:     requestLogBatchSize,
		FlushInterval: requestLogFlushInterval,
	}, &observabilityinfra.RequestLogWriter{})
	server.requestLogWorker.Start()
	server.sessionUpdateWorker = services.NewBackgroundWorker[sessionUpdateRequest](services.WorkerConfig{
		Name:          "session_update",
		BufferSize:    sessionUpdateBufferSize,
		BatchSize:     sessionUpdateBatchSize,
		FlushInterval: sessionUpdateFlushInterval,
	}, newSessionUpdateProcessor(server.sessionCache))
	server.sessionUpdateWorker.Start()
	server.startRequestLogRetentionTask()
	return server
}

func (s *Server) Start() error {
	if warnings := s.validateConfig(); len(warnings) > 0 {
		log.Println("======== Provider 配置验证警告 ========")
		for _, warn := range warnings {
			log.Printf("⚠️  %s\n", warn)
		}
		log.Println("========================================")
	}
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	s.registerRoutes(router)
	s.server = &http.Server{
		Addr:    s.addr,
		Handler: router,
	}
	log.Printf("provider relay server listening on %s\n", s.addr)
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("provider relay server error: %v\n", err)
		}
	}()
	return nil
}

func (s *Server) Stop() error {
	var err error
	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err = s.server.Shutdown(ctx)
	}
	s.stopBackgroundWorkers()
	return err
}

func (s *Server) Addr() string {
	return s.addr
}
