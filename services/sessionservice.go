package services

import (
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	"codeswitch/internal/shared/kernel"
	"github.com/daodao97/xgo/xdb"
)

// SessionBinding 会话绑定信息
type SessionBinding struct {
	Platform      string    `json:"platform"`
	SessionID     string    `json:"session_id"`
	ProviderName  string    `json:"provider_name"`
	LastSuccessAt time.Time `json:"last_success_at"`
	CreatedAt     time.Time `json:"created_at"`
}

// SessionService 会话管理服务，负责维护会话与供应商的绑定关系
type SessionService struct {
	dbName        string
	cleanupTicker *time.Ticker
	cleanupStop   chan struct{}
	cleanupWG     sync.WaitGroup

	cachesMu sync.Mutex
	caches   []sessionCacheInvalidator
}

type sessionCacheInvalidator interface {
	InvalidateSession(platform, sessionID string)
}

func NewSessionService(dbName string) *SessionService {
	if dbName == "" {
		dbName = SessionDBName
	}
	return &SessionService{dbName: dbName}
}

// GetSessionProvider 获取会话绑定的供应商名称
// 返回空字符串表示该会话未绑定或已过期
func (s *SessionService) GetSessionProvider(platform, sessionID string) (string, error) {
	if sessionID == "" {
		return "", nil
	}

	db, err := xdb.DB(s.dbName)
	if err != nil {
		return "", fmt.Errorf("获取数据库连接失败: %w", err)
	}

	var providerName string
	var lastSuccessAt time.Time

	query := `SELECT provider_name, last_success_at
		FROM session_provider_binding
		WHERE platform = ? AND session_id = ?`

	err = db.QueryRow(query, platform, sessionID).Scan(&providerName, &lastSuccessAt)
	if err == sql.ErrNoRows {
		return "", nil // 会话未绑定
	}
	if err != nil {
		return "", fmt.Errorf("查询会话绑定失败: %w", err)
	}

	// 检查是否过期
	if s.isExpired(platform, lastSuccessAt) {
		// 过期则删除记录
		if err := s.deleteSessionBinding(db, platform, sessionID); err != nil {
			log.Printf("[WARN] 删除过期会话失败: %v\n", err)
		}
		return "", nil
	}

	return providerName, nil
}

// BindSessionToProvider 绑定会话到指定供应商
func (s *SessionService) BindSessionToProvider(platform, sessionID, providerName string) error {
	if sessionID == "" || providerName == "" {
		return nil // 参数无效时静默返回
	}

	db, err := xdb.DB(s.dbName)
	if err != nil {
		return fmt.Errorf("获取数据库连接失败: %w", err)
	}

	now := time.Now()
	query := `INSERT INTO session_provider_binding (platform, session_id, provider_name, last_success_at, created_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(platform, session_id)
		DO UPDATE SET provider_name = excluded.provider_name, last_success_at = excluded.last_success_at`

	_, err = db.Exec(query, platform, sessionID, providerName, now, now)
	if err != nil {
		return fmt.Errorf("绑定会话失败: %w", err)
	}

	log.Printf("[INFO] 会话绑定: %s/%s -> %s\n", platform, sessionID, providerName)
	return nil
}

// UpdateSessionSuccess 更新会话的最后成功时间（延长会话有效期）
func (s *SessionService) UpdateSessionSuccess(platform, sessionID string) error {
	if sessionID == "" {
		return nil
	}

	db, err := xdb.DB(s.dbName)
	if err != nil {
		return fmt.Errorf("获取数据库连接失败: %w", err)
	}

	query := `UPDATE session_provider_binding
		SET last_success_at = ?
		WHERE platform = ? AND session_id = ?`

	result, err := db.Exec(query, time.Now(), platform, sessionID)
	if err != nil {
		return fmt.Errorf("更新会话时间失败: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows > 0 {
		log.Printf("[INFO] 会话续期: %s/%s\n", platform, sessionID)
	}

	return nil
}

// IsSessionExpired 检查会话是否过期
func (s *SessionService) IsSessionExpired(platform, sessionID string) (bool, error) {
	if sessionID == "" {
		return true, nil
	}

	db, err := xdb.DB(s.dbName)
	if err != nil {
		return true, fmt.Errorf("获取数据库连接失败: %w", err)
	}

	var lastSuccessAt time.Time
	query := `SELECT last_success_at FROM session_provider_binding
		WHERE platform = ? AND session_id = ?`

	err = db.QueryRow(query, platform, sessionID).Scan(&lastSuccessAt)
	if err == sql.ErrNoRows {
		return true, nil // 未找到记录，视为过期
	}
	if err != nil {
		return true, fmt.Errorf("查询会话失败: %w", err)
	}

	return s.isExpired(platform, lastSuccessAt), nil
}

// CleanExpiredSessions 清理所有过期的会话绑定记录
func (s *SessionService) CleanExpiredSessions() error {
	db, err := xdb.DB(s.dbName)
	if err != nil {
		return fmt.Errorf("获取数据库连接失败: %w", err)
	}

	var totalDeleted int64
	now := time.Now()
	clauses := make([]struct {
		platform string
		cutoff   time.Time
	}, 0, len(kernel.AllPlatforms()))
	for _, platform := range kernel.AllPlatforms() {
		clauses = append(clauses, struct {
			platform string
			cutoff   time.Time
		}{
			platform: platform.String(),
			cutoff:   now.Add(-platform.SessionTimeout()),
		})
	}
	for _, clause := range clauses {
		result, err := db.Exec(`DELETE FROM session_provider_binding WHERE platform = ? AND last_success_at < ?`, clause.platform, clause.cutoff)
		if err != nil {
			return fmt.Errorf("清理过期会话失败: %w", err)
		}
		if rows, err := result.RowsAffected(); err == nil {
			totalDeleted += rows
		}
	}
	if totalDeleted > 0 {
		log.Printf("[INFO] 清理了 %d 个过期会话\n", totalDeleted)
	}

	return nil
}

// StartCleanupTask 启动后台定时清理任务
func (s *SessionService) StartCleanupTask() {
	if s.cleanupTicker != nil {
		return
	}
	ticker := time.NewTicker(5 * time.Minute) // 每5分钟清理一次
	s.cleanupTicker = ticker
	s.cleanupStop = make(chan struct{})
	s.cleanupWG.Add(1)
	go func() {
		defer s.cleanupWG.Done()
		for {
			select {
			case <-ticker.C:
				if err := s.CleanExpiredSessions(); err != nil {
					log.Printf("[ERROR] 定时清理过期会话失败: %v\n", err)
				}
			case <-s.cleanupStop:
				return
			}
		}
	}()
	log.Println("[INFO] 会话清理定时任务已启动（每5分钟执行）")
}

// StopCleanupTask 停止后台清理任务
func (s *SessionService) StopCleanupTask() {
	if s.cleanupTicker == nil {
		return
	}
	s.cleanupTicker.Stop()
	if s.cleanupStop != nil {
		close(s.cleanupStop)
	}
	s.cleanupWG.Wait()
	s.cleanupTicker = nil
	s.cleanupStop = nil
}

// isExpired 检查给定的时间是否已过期（内部辅助方法）
func (s *SessionService) isExpired(platform string, lastSuccessAt time.Time) bool {
	timeout := kernel.Platform(platform).SessionTimeout()
	return time.Since(lastSuccessAt) > timeout
}

// deleteSessionBinding 删除会话绑定记录（内部辅助方法）
func (s *SessionService) deleteSessionBinding(db *sql.DB, platform, sessionID string) error {
	query := `DELETE FROM session_provider_binding WHERE platform = ? AND session_id = ?`
	_, err := db.Exec(query, platform, sessionID)
	return err
}

// GetProviderSessions 获取指定供应商的所有活跃会话绑定
func (s *SessionService) GetProviderSessions(platform, providerName string) ([]SessionBinding, error) {
	if platform == "" || providerName == "" {
		return []SessionBinding{}, nil
	}

	db, err := xdb.DB(s.dbName)
	if err != nil {
		return nil, fmt.Errorf("获取数据库连接失败: %w", err)
	}

	query := `SELECT platform, session_id, provider_name, last_success_at, created_at
		FROM session_provider_binding
		WHERE platform = ? AND provider_name = ?
		ORDER BY last_success_at DESC`

	rows, err := db.Query(query, platform, providerName)
	if err != nil {
		return nil, fmt.Errorf("查询会话绑定失败: %w", err)
	}
	defer rows.Close()

	var sessions []SessionBinding
	for rows.Next() {
		var session SessionBinding
		if err := rows.Scan(&session.Platform, &session.SessionID, &session.ProviderName, &session.LastSuccessAt, &session.CreatedAt); err != nil {
			log.Printf("[WARN] 扫描会话记录失败: %v\n", err)
			continue
		}

		// 过滤掉已过期的会话
		if !s.isExpired(session.Platform, session.LastSuccessAt) {
			sessions = append(sessions, session)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历会话记录失败: %w", err)
	}

	return sessions, nil
}

// GetPlatformSessions 获取指定平台的所有活跃会话绑定
func (s *SessionService) GetPlatformSessions(platform string) ([]SessionBinding, error) {
	if platform == "" {
		return []SessionBinding{}, nil
	}

	db, err := xdb.DB(s.dbName)
	if err != nil {
		return nil, fmt.Errorf("获取数据库连接失败: %w", err)
	}

	query := `SELECT platform, session_id, provider_name, last_success_at, created_at
		FROM session_provider_binding
		WHERE platform = ?
		ORDER BY provider_name ASC, last_success_at DESC`

	rows, err := db.Query(query, platform)
	if err != nil {
		return nil, fmt.Errorf("查询会话绑定失败: %w", err)
	}
	defer rows.Close()

	var sessions []SessionBinding
	for rows.Next() {
		var session SessionBinding
		if err := rows.Scan(&session.Platform, &session.SessionID, &session.ProviderName, &session.LastSuccessAt, &session.CreatedAt); err != nil {
			log.Printf("[WARN] 扫描会话记录失败: %v\n", err)
			continue
		}

		// 过滤掉已过期的会话
		if !s.isExpired(session.Platform, session.LastSuccessAt) {
			sessions = append(sessions, session)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历会话记录失败: %w", err)
	}

	return sessions, nil
}

// UnbindSession 解除指定会话的绑定
func (s *SessionService) UnbindSession(platform, sessionID string) error {
	if platform == "" || sessionID == "" {
		return fmt.Errorf("平台和会话ID不能为空")
	}

	db, err := xdb.DB(s.dbName)
	if err != nil {
		return fmt.Errorf("获取数据库连接失败: %w", err)
	}

	if err := s.deleteSessionBinding(db, platform, sessionID); err != nil {
		return fmt.Errorf("解除会话绑定失败: %w", err)
	}

	s.invalidateCaches(platform, sessionID)

	log.Printf("[INFO] 会话解绑: %s/%s\n", platform, sessionID)
	return nil
}

// RegisterCache 允许缓存实现登记，便于解绑时同步清理缓存
func (s *SessionService) RegisterCache(cache sessionCacheInvalidator) {
	if cache == nil {
		return
	}
	s.cachesMu.Lock()
	defer s.cachesMu.Unlock()
	s.caches = append(s.caches, cache)
}

// invalidateCaches 通知所有注册的 SessionCache 立即失效指定会话
func (s *SessionService) invalidateCaches(platform, sessionID string) {
	s.cachesMu.Lock()
	defer s.cachesMu.Unlock()
	for _, cache := range s.caches {
		if cache != nil {
			cache.InvalidateSession(platform, sessionID)
		}
	}
}
