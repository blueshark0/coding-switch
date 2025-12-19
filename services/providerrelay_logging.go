package services

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/daodao97/xgo/xdb"
)

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
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
			case <-prs.shutdownCh:
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
		close(prs.shutdownCh)
	})
	if prs.requestLogWorker != nil {
		prs.requestLogWorker.Stop()
	}
	if prs.sessionUpdateWorker != nil {
		prs.sessionUpdateWorker.Stop()
	}
	prs.backgroundWG.Wait()
}

func (prs *ProviderRelayService) enqueueRequestLog(logEntry *RequestLog) {
	if prs == nil || logEntry == nil {
		return
	}
	if prs.requestLogWorker == nil {
		prs.writeRequestLogSync(logEntry)
		return
	}
	prs.requestLogWorker.Enqueue(logEntry)
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

func RequestLogHook(platform Platform, usage *RequestLog) func(data []byte) (bool, []byte) { // SSE 钩子：累计字节和解析 token 用量
	parser := GetTokenParser(platform)
	return func(data []byte) (bool, []byte) {
		payload := strings.TrimSpace(string(data))
		parseEventPayload(payload, parser, usage)
		return true, data
	}
}

func parseEventPayload(payload string, parser TokenUsageParser, usage *RequestLog) {
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
