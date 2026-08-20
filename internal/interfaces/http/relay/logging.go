package relay

import (
	"log"
	"os"
	"strings"
	"time"

	observabilitydomain "codeswitch/internal/observability/domain"
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
	if s == nil || s.retentionCleanup == nil {
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
				if err := s.retentionCleanup(); err != nil {
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
	if logEntry == nil || s.requestLogSink == nil {
		return
	}
	if err := s.requestLogSink.ProcessSingle(logEntry); err != nil {
		log.Printf("写入 request_log 失败: %v\n", err)
	}
}

func (s *Server) requestLogHook(platform kernel.Platform, usage *observabilitydomain.RequestLog) func(data []byte) (bool, []byte) {
	var parser TokenUsageParser
	if s.tokenParserFor != nil {
		parser = s.tokenParserFor(platform)
	}
	return func(data []byte) (bool, []byte) {
		payload := strings.TrimSpace(string(data))
		parseEventPayload(payload, parser, usage)
		return true, data
	}
}

func parseEventPayload(payload string, parser TokenUsageParser, usage *observabilitydomain.RequestLog) {
	if parser == nil || usage == nil || payload == "" {
		return
	}
	foundData := false
	lines := strings.Split(payload, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "data:") {
			foundData = true
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if data == "" {
				continue
			}
			parser.Parse(data, usage)
		}
	}
	// 非流式响应（stream:false、/responses/compact、Gemini generateContent）
	// 是纯 JSON 而非 SSE：没有 data: 行时把整个 payload 交给解析器。
	if !foundData && strings.HasPrefix(strings.TrimSpace(payload), "{") {
		parser.Parse(payload, usage)
	}
}
