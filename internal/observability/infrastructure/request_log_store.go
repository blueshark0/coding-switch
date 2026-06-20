package infrastructure

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	observabilitydomain "codeswitch/internal/observability/domain"
	"codeswitch/internal/shared/storage"

	"github.com/daodao97/xgo/xdb"
)

const RequestLogRetentionDays = 30

const requestLogCheckpointThreshold = 1000

var requestLogWriteMu sync.Mutex

func requestLogModel() xdb.Model {
	return xdb.New("request_log", xdb.WithConn(storage.RequestLogDBName))
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func recordFromRequestLog(logEntry *observabilitydomain.RequestLog) xdb.Record {
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
		"is_fast":             boolToInt(logEntry.IsFast),
		"duration_sec":        logEntry.DurationSec,
		"created_at":          requestLogCreatedAt(logEntry.CreatedAt),
	}
}

func requestLogCreatedAt(createdAt string) string {
	if strings.TrimSpace(createdAt) != "" {
		return createdAt
	}
	return time.Now().UTC().Format(timeLayout)
}

func CleanupOldRequestLogs(retentionDays int) error {
	if retentionDays <= 0 {
		return nil
	}
	db, err := xdb.DB(storage.RequestLogDBName)
	if err != nil {
		return err
	}

	requestLogWriteMu.Lock()
	defer requestLogWriteMu.Unlock()

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
		if rows >= requestLogCheckpointThreshold {
			if _, err := db.Exec("PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
				log.Printf("执行 wal_checkpoint 失败: %v\n", err)
			}
		}
	}
	if _, err := db.Exec("PRAGMA optimize"); err != nil {
		log.Printf("执行 PRAGMA optimize 失败: %v\n", err)
	}
	return nil
}

func isNoSuchTableErr(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "no such table")
}

func isSQLiteBusyErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "database is locked") ||
		strings.Contains(msg, "sqlite_busy") ||
		strings.Contains(msg, "database table is locked") ||
		strings.Contains(msg, "sqlite_locked")
}

func EnsureRequestLogTableWithDB(db *sql.DB) error {
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
		is_fast INTEGER DEFAULT 0,
		duration_sec REAL DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`

	if _, err := db.Exec(createTableSQL); err != nil {
		return err
	}

	columns := []struct {
		name       string
		definition string
	}{
		{name: "created_at", definition: "DATETIME"},
		{name: "is_stream", definition: "INTEGER DEFAULT 0"},
		{name: "is_fast", definition: "INTEGER DEFAULT 0"},
		{name: "duration_sec", definition: "REAL DEFAULT 0"},
	}
	for _, column := range columns {
		query := fmt.Sprintf("SELECT COUNT(*) FROM pragma_table_info('request_log') WHERE name = '%s'", column.name)
		var count int
		if err := db.QueryRow(query).Scan(&count); err != nil {
			return err
		}
		if count == 0 {
			alter := fmt.Sprintf("ALTER TABLE request_log ADD COLUMN %s %s", column.name, column.definition)
			if _, err := db.Exec(alter); err != nil {
				return err
			}
		}
	}
	if _, err := db.Exec("UPDATE request_log SET created_at = CURRENT_TIMESTAMP WHERE created_at IS NULL OR created_at = ''"); err != nil {
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
