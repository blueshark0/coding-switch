package services

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/daodao97/xgo/xdb"
)

const (
	// ClaudeSessionTimeout Claude Code 会话超时时间（5分钟）
	ClaudeSessionTimeout = 5 * time.Minute
	// CodexSessionTimeout Codex 会话超时时间（15分钟）
	CodexSessionTimeout = 15 * time.Minute
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
type SessionService struct{}

func NewSessionService() *SessionService {
	return &SessionService{}
}

// GetSessionProvider 获取会话绑定的供应商名称
// 返回空字符串表示该会话未绑定或已过期
func (s *SessionService) GetSessionProvider(platform, sessionID string) (string, error) {
	if sessionID == "" {
		return "", nil
	}

	db, err := xdb.DB("default")
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

	db, err := xdb.DB("default")
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

	db, err := xdb.DB("default")
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

	db, err := xdb.DB("default")
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
	db, err := xdb.DB("default")
	if err != nil {
		return fmt.Errorf("获取数据库连接失败: %w", err)
	}

	// 计算过期时间点
	claudeExpiredTime := time.Now().Add(-ClaudeSessionTimeout)
	codexExpiredTime := time.Now().Add(-CodexSessionTimeout)

	query := `DELETE FROM session_provider_binding
		WHERE (platform = 'claude' AND last_success_at < ?)
		   OR (platform = 'codex' AND last_success_at < ?)`

	result, err := db.Exec(query, claudeExpiredTime, codexExpiredTime)
	if err != nil {
		return fmt.Errorf("清理过期会话失败: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows > 0 {
		log.Printf("[INFO] 清理了 %d 个过期会话\n", rows)
	}

	return nil
}

// StartCleanupTask 启动后台定时清理任务
func (s *SessionService) StartCleanupTask() {
	ticker := time.NewTicker(5 * time.Minute) // 每5分钟清理一次
	go func() {
		for range ticker.C {
			if err := s.CleanExpiredSessions(); err != nil {
				log.Printf("[ERROR] 定时清理过期会话失败: %v\n", err)
			}
		}
	}()
	log.Println("[INFO] 会话清理定时任务已启动（每5分钟执行）")
}

// isExpired 检查给定的时间是否已过期（内部辅助方法）
func (s *SessionService) isExpired(platform string, lastSuccessAt time.Time) bool {
	timeout := ClaudeSessionTimeout
	if platform == "codex" {
		timeout = CodexSessionTimeout
	}
	return time.Since(lastSuccessAt) > timeout
}

// deleteSessionBinding 删除会话绑定记录（内部辅助方法）
func (s *SessionService) deleteSessionBinding(db *sql.DB, platform, sessionID string) error {
	query := `DELETE FROM session_provider_binding WHERE platform = ? AND session_id = ?`
	_, err := db.Exec(query, platform, sessionID)
	return err
}

// GetProviderSessions 获取指定供应商的所有活跃会话绑定
func (s *SessionService) GetProviderSessions(providerName string) ([]SessionBinding, error) {
	if providerName == "" {
		return []SessionBinding{}, nil
	}

	db, err := xdb.DB("default")
	if err != nil {
		return nil, fmt.Errorf("获取数据库连接失败: %w", err)
	}

	query := `SELECT platform, session_id, provider_name, last_success_at, created_at
		FROM session_provider_binding
		WHERE provider_name = ?
		ORDER BY last_success_at DESC`

	rows, err := db.Query(query, providerName)
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

	db, err := xdb.DB("default")
	if err != nil {
		return fmt.Errorf("获取数据库连接失败: %w", err)
	}

	if err := s.deleteSessionBinding(db, platform, sessionID); err != nil {
		return fmt.Errorf("解除会话绑定失败: %w", err)
	}

	log.Printf("[INFO] 会话解绑: %s/%s\n", platform, sessionID)
	return nil
}
