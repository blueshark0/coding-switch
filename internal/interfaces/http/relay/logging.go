package relay

import (
	"log"
	"os"
	"strings"
	"time"

	observabilitydomain "codeswitch/internal/observability/domain"
	observabilityinfra "codeswitch/internal/observability/infrastructure"
	"codeswitch/internal/shared/kernel"
)

var relayDebugEnabled = os.Getenv("CODE_SWITCH_DEBUG_RELAY") == "1"

func relayDebugf(format string, args ...any) {
	if !relayDebugEnabled {
		return
	}
	log.Printf(format, args...)
}

func relayWarnf(format string, args ...any) {
	log.Printf("[WARN] [relay] "+format, args...)
}

func (s *Server) startRequestLogRetentionTask() {
	if s == nil || observabilityinfra.RequestLogRetentionDays <= 0 {
		return
	}
	s.backgroundWG.Add(1)
	go func() {
		defer s.backgroundWG.Done()
		ticker := time.NewTicker(requestLogCleanupInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := observabilityinfra.CleanupOldRequestLogs(observabilityinfra.RequestLogRetentionDays); err != nil {
					log.Printf("定时清理 request_log 失败: %v\n", err)
				}
			case <-s.shutdownCh:
				return
			}
		}
	}()
}

func (s *Server) stopBackgroundWorkers() {
	if s == nil {
		return
	}
	s.shutdownOnce.Do(func() {
		close(s.shutdownCh)
	})
	if s.requestLogWorker != nil {
		s.requestLogWorker.Stop()
	}
	if s.sessionUpdateWorker != nil {
		s.sessionUpdateWorker.Stop()
	}
	s.backgroundWG.Wait()
}

func (s *Server) enqueueRequestLog(logEntry *observabilitydomain.RequestLog) {
	if s == nil || logEntry == nil {
		return
	}
	if s.requestLogWorker == nil {
		s.writeRequestLogSync(logEntry)
		return
	}
	s.requestLogWorker.Enqueue(logEntry)
}

func (s *Server) writeRequestLogSync(logEntry *observabilitydomain.RequestLog) {
	if logEntry == nil {
		return
	}
	if err := (&observabilityinfra.RequestLogWriter{}).ProcessSingle(logEntry); err != nil {
		log.Printf("写入 request_log 失败: %v\n", err)
	}
}

func RequestLogHook(platform kernel.Platform, usage *observabilitydomain.RequestLog) func(data []byte) (bool, []byte) {
	parser := observabilityinfra.GetTokenParser(platform)
	return func(data []byte) (bool, []byte) {
		payload := strings.TrimSpace(string(data))
		parseEventPayload(payload, parser, usage)
		return true, data
	}
}

func parseEventPayload(payload string, parser observabilityinfra.TokenUsageParser, usage *observabilitydomain.RequestLog) {
	if parser == nil || usage == nil || payload == "" {
		return
	}
	lines := strings.Split(payload, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "data:") {
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if data == "" {
				continue
			}
			parser.Parse(data, usage)
		}
	}
}
