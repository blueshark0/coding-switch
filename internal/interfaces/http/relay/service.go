package relay

import (
	"context"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	observabilitydomain "codeswitch/internal/observability/domain"
	observabilityinfra "codeswitch/internal/observability/infrastructure"
	routingapp "codeswitch/internal/routing/application"
	sessionapp "codeswitch/internal/sessions/application"
	sessioninfra "codeswitch/internal/sessions/infrastructure"
	"codeswitch/internal/shared/worker"
	"github.com/gin-gonic/gin"
)

const (
	requestLogBufferSize      = 4096
	requestLogBatchSize       = 100
	requestLogFlushInterval   = 250 * time.Millisecond
	requestLogCleanupInterval = 6 * time.Hour
	requestLogRetentionDays   = 60

	sessionUpdateBufferSize    = 512
	sessionUpdateBatchSize     = 50
	sessionUpdateFlushInterval = 500 * time.Millisecond
)

type sessionUpdateRequest struct {
	platform   string
	sessionID  string
	generation uint64
}

type Server struct {
	routingService      *routingapp.Service
	sessionService      *sessionapp.Service
	sessionCache        *sessioninfra.Cache
	listener            net.Listener
	server              *http.Server
	addr                string
	requestLogWorker    *worker.BackgroundWorker[*observabilitydomain.RequestLog]
	sessionUpdateWorker *worker.BackgroundWorker[sessionUpdateRequest]
	shutdownCh          chan struct{}
	backgroundWG        sync.WaitGroup
	shutdownOnce        sync.Once
	proxyMu             sync.RWMutex
	proxyEnabled        bool
	proxyURL            string
}

func NewServer(routingService *routingapp.Service, sessionService *sessionapp.Service, addr string) *Server {
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
	server.requestLogWorker = worker.New[*observabilitydomain.RequestLog](worker.Config{
		Name:          "request_log",
		BufferSize:    requestLogBufferSize,
		BatchSize:     requestLogBatchSize,
		FlushInterval: requestLogFlushInterval,
		DropWhenFull:  true,
	}, &observabilityinfra.RequestLogWriter{})
	server.requestLogWorker.Start()
	server.sessionUpdateWorker = worker.New[sessionUpdateRequest](worker.Config{
		Name:          "session_update",
		BufferSize:    sessionUpdateBufferSize,
		BatchSize:     sessionUpdateBatchSize,
		FlushInterval: sessionUpdateFlushInterval,
	}, newSessionUpdateProcessor(server.sessionCache))
	server.sessionUpdateWorker.Start()
	server.startRequestLogRetentionTask()
	server.initProxyConfig()
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
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	s.server = &http.Server{
		Addr:    s.addr,
		Handler: router,
	}
	s.listener = listener
	log.Printf("provider relay server listening on %s\n", s.addr)
	go func() {
		if err := s.server.Serve(listener); err != nil && err != http.ErrServerClosed {
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
		s.listener = nil
	}
	s.stopBackgroundWorkers()
	return err
}

func (s *Server) Addr() string {
	return s.addr
}
