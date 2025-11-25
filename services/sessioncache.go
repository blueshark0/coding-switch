package services

import (
	"fmt"
	"log"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
)

const (
	sessionCacheSize = 1000
)

// sessionCacheEntry 缓存条目
type sessionCacheEntry struct {
	providerName  string
	lastSuccessAt time.Time
}

// SessionCache 会话绑定缓存，减少数据库查询
type SessionCache struct {
	cache   *lru.Cache[string, *sessionCacheEntry]
	service *SessionService
}

// NewSessionCache 创建会话缓存
func NewSessionCache(service *SessionService) *SessionCache {
	cache, err := lru.New[string, *sessionCacheEntry](sessionCacheSize)
	if err != nil {
		log.Printf("[ERROR] 创建会话缓存失败: %v\n", err)
		return &SessionCache{service: service}
	}
	sc := &SessionCache{
		cache:   cache,
		service: service,
	}
	if service != nil {
		service.registerCache(sc)
	}
	return sc
}

// cacheKey 生成缓存键
func (sc *SessionCache) cacheKey(platform, sessionID string) string {
	return fmt.Sprintf("%s:%s", platform, sessionID)
}

// GetSessionProvider 获取会话绑定的供应商（优先从缓存读取）
func (sc *SessionCache) GetSessionProvider(platform, sessionID string) (string, error) {
	if sessionID == "" {
		return "", nil
	}

	key := sc.cacheKey(platform, sessionID)

	// 尝试从缓存读取
	if sc.cache != nil {
		if entry, ok := sc.cache.Get(key); ok {
			// 检查是否过期
			if !sc.isExpired(platform, entry.lastSuccessAt) {
				return entry.providerName, nil
			}
			// 过期则删除缓存
			sc.cache.Remove(key)
		}
	}

	// 缓存未命中，查询数据库
	providerName, err := sc.service.GetSessionProvider(platform, sessionID)
	if err != nil {
		return "", err
	}

	// 写入缓存
	if providerName != "" && sc.cache != nil {
		sc.cache.Add(key, &sessionCacheEntry{
			providerName:  providerName,
			lastSuccessAt: time.Now(),
		})
	}

	return providerName, nil
}

// BindSessionToProvider 绑定会话到供应商（同时更新缓存）
func (sc *SessionCache) BindSessionToProvider(platform, sessionID, providerName string) error {
	if sessionID == "" || providerName == "" {
		return nil
	}

	// 先写数据库
	if err := sc.service.BindSessionToProvider(platform, sessionID, providerName); err != nil {
		return err
	}

	// 更新缓存
	if sc.cache != nil {
		key := sc.cacheKey(platform, sessionID)
		sc.cache.Add(key, &sessionCacheEntry{
			providerName:  providerName,
			lastSuccessAt: time.Now(),
		})
	}

	return nil
}

// UpdateSessionSuccess 更新会话成功时间（同时更新缓存）
func (sc *SessionCache) UpdateSessionSuccess(platform, sessionID string) error {
	if sessionID == "" {
		return nil
	}

	// 先写数据库
	if err := sc.service.UpdateSessionSuccess(platform, sessionID); err != nil {
		return err
	}

	// 更新缓存中的时间
	if sc.cache != nil {
		key := sc.cacheKey(platform, sessionID)
		if entry, ok := sc.cache.Get(key); ok {
			entry.lastSuccessAt = time.Now()
		}
	}

	return nil
}

// InvalidateSession 使会话缓存失效
func (sc *SessionCache) InvalidateSession(platform, sessionID string) {
	if sc.cache != nil {
		key := sc.cacheKey(platform, sessionID)
		sc.cache.Remove(key)
	}
}

// isExpired 检查是否过期
func (sc *SessionCache) isExpired(platform string, lastSuccessAt time.Time) bool {
	timeout := ClaudeSessionTimeout
	if platform == "codex" {
		timeout = CodexSessionTimeout
	} else if platform == "gemini" {
		timeout = GeminiSessionTimeout
	}
	return time.Since(lastSuccessAt) > timeout
}

// Stats 返回缓存统计信息
func (sc *SessionCache) Stats() (size int, capacity int) {
	if sc.cache == nil {
		return 0, 0
	}
	return sc.cache.Len(), sessionCacheSize
}
