package bootstrap

import (
	"io"
	"sync"
	"time"

	appdesktop "codeswitch/internal/app/desktop"
	configsyncapp "codeswitch/internal/configsync/application"
	configsyncinfra "codeswitch/internal/configsync/infrastructure"
	hotkeyinfra "codeswitch/internal/hotkeys/infrastructure"
	relayhttp "codeswitch/internal/interfaces/http/relay"
	facades "codeswitch/internal/interfaces/wails"
	observabilityinfra "codeswitch/internal/observability/infrastructure"
	platformproxyapp "codeswitch/internal/platformproxy/application"
	platformproxyinfra "codeswitch/internal/platformproxy/infrastructure"
	routingapp "codeswitch/internal/routing/application"
	routinginfra "codeswitch/internal/routing/infrastructure"
	sessionapp "codeswitch/internal/sessions/application"
	sessioninfra "codeswitch/internal/sessions/infrastructure"
	"codeswitch/internal/shared/kernel"
	"codeswitch/internal/shared/logging"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/services/dock"
)

const defaultSessionCleanupInterval = 5 * time.Minute

type Container struct {
	logFile       io.WriteCloser
	relayServer   *relayhttp.Server
	cleanupRunner *sessionapp.CleanupRunner
	hotkeyService *facades.HotkeyFacade
	logsWindow    *appdesktop.LogsWindowService
	routingFacade *facades.RoutingFacade
	syncFacade    *facades.ConfigSyncFacade
	sessionFacade *facades.SessionFacade
	obsFacade     *facades.ObservabilityFacade
	proxyFacade   *facades.PlatformProxyFacade
	dockService   *dock.DockService
	shutdownOnce  sync.Once
}

func Initialize(appVersion string) (*Container, error) {
	logFile := logging.Setup()
	if err := NewInitializer().Initialize(); err != nil {
		if logFile != nil {
			_ = logFile.Close()
		}
		return nil, err
	}

	hotkeyStore, err := hotkeyinfra.NewSQLiteStore()
	if err != nil {
		if logFile != nil {
			_ = logFile.Close()
		}
		return nil, err
	}

	logsWindow := appdesktop.NewLogsWindowService()
	sessionRepo := sessioninfra.NewSQLiteRepository("")
	sessionService := sessionapp.NewService(sessionRepo)
	routingService := routingapp.NewService(routinginfra.NewSQLiteStore())
	// 仅监听回环地址：绑定 0.0.0.0 会让同一局域网任意主机免费使用已配置的付费 API key。
	relayServer := relayhttp.NewServer(routingService, sessionService, "127.0.0.1:18100", relayhttp.Options{
		SessionCache:   sessioninfra.NewCache(sessionService),
		RequestLogSink: &observabilityinfra.RequestLogWriter{},
		TokenParserFor: func(platform kernel.Platform) relayhttp.TokenUsageParser {
			return observabilityinfra.GetTokenParser(platform)
		},
		RetentionCleanup: func() error {
			return observabilityinfra.CleanupOldRequestLogs(observabilityinfra.RequestLogRetentionDays)
		},
	})
	// 偏好保存（含 configsync 快照恢复）后由 application 层统一推送代理配置到 relay。
	routingService.RegisterProxyConfigSink(relayServer)
	platformProxyService := platformproxyapp.NewService(map[kernel.Platform]platformproxyapp.Manager{
		kernel.PlatformClaude: platformproxyinfra.NewClaudeManager(relayServer.Addr()),
		kernel.PlatformCodex:  platformproxyinfra.NewCodexManager(relayServer.Addr()),
		kernel.PlatformGemini: platformproxyinfra.NewGeminiManager(relayServer.Addr()),
	})
	configSyncService := configsyncapp.NewService(
		configsyncinfra.NewSQLiteStore(),
		configsyncinfra.NewWebDAVClient(),
		routingService,
		appVersion,
	)

	return &Container{
		logFile:       logFile,
		relayServer:   relayServer,
		cleanupRunner: sessionapp.NewCleanupRunner(sessionService, defaultSessionCleanupInterval),
		hotkeyService: facades.NewHotkeyFacade(hotkeyStore),
		logsWindow:    logsWindow,
		routingFacade: facades.NewRoutingFacade(routingService),
		syncFacade:    facades.NewConfigSyncFacade(configSyncService),
		sessionFacade: facades.NewSessionFacade(sessionService),
		obsFacade:     facades.NewObservabilityFacade(observabilityinfra.NewSQLiteQueries()),
		proxyFacade:   facades.NewPlatformProxyFacade(platformProxyService),
		dockService:   dock.New(),
	}, nil
}

func (c *Container) ApplicationServices(extra ...application.Service) []application.Service {
	services := []application.Service{
		application.NewService(c.logsWindow),
		application.NewService(c.hotkeyService),
		application.NewService(c.routingFacade),
		application.NewService(c.syncFacade),
		application.NewService(c.sessionFacade),
		application.NewService(c.obsFacade),
		application.NewService(c.proxyFacade),
		application.NewService(c.dockService),
	}
	services = append(services, extra...)
	return services
}

func (c *Container) LogsWindowService() *appdesktop.LogsWindowService {
	return c.logsWindow
}

func (c *Container) DockService() *dock.DockService {
	return c.dockService
}

func (c *Container) StartBackground() error {
	if c == nil {
		return nil
	}
	if err := c.relayServer.Start(); err != nil {
		return err
	}
	if c.cleanupRunner != nil {
		c.cleanupRunner.Start()
	}
	return nil
}

func (c *Container) Shutdown() {
	if c == nil {
		return
	}
	c.shutdownOnce.Do(func() {
		if c.relayServer != nil {
			_ = c.relayServer.Stop()
		}
		if c.cleanupRunner != nil {
			c.cleanupRunner.Stop()
		}
		if c.hotkeyService != nil {
			_ = c.hotkeyService.Close()
		}
		if c.logFile != nil {
			_ = c.logFile.Close()
		}
	})
}
