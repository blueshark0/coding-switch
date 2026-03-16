package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	observabilityinfra "codeswitch/internal/observability/infrastructure"
	routinginfra "codeswitch/internal/routing/infrastructure"
	"codeswitch/internal/shared/storage"

	"github.com/daodao97/xgo/xdb"
)

// DatabaseInitializer 负责数据库连接池初始化和表结构管理
type DatabaseInitializer struct {
	dbDir       string
	initialized bool
	mu          sync.Mutex
}

// NewDatabaseInitializer 创建数据库初始化器
func NewDatabaseInitializer() *DatabaseInitializer {
	home, _ := os.UserHomeDir()
	return &DatabaseInitializer{
		dbDir: filepath.Join(home, ".code-switch"),
	}
}

// Initialize 初始化所有数据库连接和表结构
func (di *DatabaseInitializer) Initialize() error {
	di.mu.Lock()
	defer di.mu.Unlock()

	if di.initialized {
		return nil
	}

	// 确保目录存在
	if err := os.MkdirAll(di.dbDir, 0o755); err != nil {
		return fmt.Errorf("创建数据库目录失败: %w", err)
	}

	// 初始化连接池
	if err := di.initConnections(); err != nil {
		return err
	}

	// 创建表结构
	if err := di.ensureTables(); err != nil {
		return err
	}

	// 配置 PRAGMA
	di.configurePragmas()

	// 执行数据迁移
	if err := di.runMigrations(); err != nil {
		log.Printf("数据迁移警告: %v\n", err)
		// 不阻塞启动
	}

	// 启动时清理过期日志
	if err := observabilityinfra.CleanupOldRequestLogs(observabilityinfra.RequestLogRetentionDays); err != nil {
		log.Printf("启动时清理历史 request_log 失败: %v\n", err)
	}

	di.initialized = true
	return nil
}

// initConnections 初始化数据库连接池
func (di *DatabaseInitializer) initConnections() error {
	appDBPath := filepath.Join(di.dbDir, "app.db")
	requestLogDBPath := filepath.Join(di.dbDir, "request_log.db")
	sessionDBPath := filepath.Join(di.dbDir, "session.db")

	appDSN := fmt.Sprintf("%s?cache=shared&mode=rwc&_journal_mode=WAL&_busy_timeout=5000", appDBPath)
	requestLogDSN := fmt.Sprintf("%s?cache=shared&mode=rwc&_journal_mode=WAL&_busy_timeout=5000", requestLogDBPath)
	sessionDSN := fmt.Sprintf("%s?cache=shared&mode=rwc&_journal_mode=WAL&_busy_timeout=5000", sessionDBPath)

	return xdb.Inits([]xdb.Config{
		{
			Name:        storage.CoreDBName,
			Driver:      "sqlite",
			DSN:         appDSN,
			MaxOpenConn: 1,
			MaxIdleConn: 1,
		},
		{
			Name:        storage.RequestLogDBName,
			Driver:      "sqlite",
			DSN:         requestLogDSN,
			MaxOpenConn: 4,
			MaxIdleConn: 4,
		},
		{
			Name:        storage.SessionDBName,
			Driver:      "sqlite",
			DSN:         sessionDSN,
			MaxOpenConn: 5,
			MaxIdleConn: 2,
		},
	})
}

// ensureTables 确保所有表结构存在
func (di *DatabaseInitializer) ensureTables() error {
	store := routinginfra.NewSQLiteStore()
	if err := store.EnsureSchema(); err != nil {
		return fmt.Errorf("初始化 app 配置表失败: %w", err)
	}
	if err := ensureRequestLogTable(RequestLogDBName); err != nil {
		return fmt.Errorf("初始化 request_log 表失败: %w", err)
	}
	if err := ensureSessionBindingTable(SessionDBName); err != nil {
		return fmt.Errorf("初始化 session_provider_binding 表失败: %w", err)
	}
	return nil
}

// configurePragmas 配置所有数据库的 PRAGMA
func (di *DatabaseInitializer) configurePragmas() {
	configureSQLitePragmas(CoreDBName)
	configureSQLitePragmas(RequestLogDBName)
	configureSQLitePragmas(SessionDBName)
}

// runMigrations 执行数据迁移
func (di *DatabaseInitializer) runMigrations() error {
	requestLogDBPath := filepath.Join(di.dbDir, "request_log.db")
	sessionDBPath := filepath.Join(di.dbDir, "session.db")
	errs := []error{
		migrateLegacyTables(requestLogDBPath, sessionDBPath),
	}
	store := routinginfra.NewSQLiteStore()
	if err := routinginfra.NewLegacyImporter(store).EnsureImported(context.Background()); err != nil {
		errs = append(errs, fmt.Errorf("legacy config import: %w", err))
	}
	return errors.Join(errs...)
}

// ensureRequestLogTable 确保 request_log 表存在
func ensureRequestLogTable(dbName string) error {
	db, err := xdb.DB(dbName)
	if err != nil {
		return err
	}
	return observabilityinfra.EnsureRequestLogTableWithDB(db)
}

// ensureSessionBindingTable 确保 session_provider_binding 表存在
func ensureSessionBindingTable(dbName string) error {
	db, err := xdb.DB(dbName)
	if err != nil {
		return err
	}
	return ensureSessionBindingTableWithDB(db)
}

// ensureSessionBindingTableWithDB 使用数据库连接确保 session_provider_binding 表存在
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

// configureSQLitePragmas 配置 SQLite PRAGMA
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

// migrateLegacyTables 迁移历史表数据
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

// migrateTable 迁移单个表数据
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

// sqliteTableExists 检查表是否存在
func sqliteTableExists(db *sql.DB, table string) (bool, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?", table).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// sqliteRowCount 获取表行数
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

// attachDatabase 附加数据库
func attachDatabase(db *sql.DB, alias, path string) error {
	if alias == "" || path == "" {
		return fmt.Errorf("attach 参数无效")
	}
	escaped := strings.ReplaceAll(path, "'", "''")
	stmt := fmt.Sprintf("ATTACH DATABASE '%s' AS %s", escaped, alias)
	_, err := db.Exec(stmt)
	return err
}

// detachDatabase 分离数据库
func detachDatabase(db *sql.DB, alias string) {
	if alias == "" {
		return
	}
	if _, err := db.Exec(fmt.Sprintf("DETACH DATABASE %s", alias)); err != nil {
		log.Printf("分离数据库 %s 失败: %v\n", alias, err)
	}
}

// isNoSuchTableErr 检查是否为表不存在错误
func isNoSuchTableErr(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "no such table")
}
