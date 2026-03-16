package infrastructure

import (
	"fmt"
	"log"
	"time"

	"codeswitch/internal/shared/kernel"
	"codeswitch/services"

	lru "github.com/hashicorp/golang-lru/v2"
)

const sessionCacheSize = 1000

type sessionCacheEntry struct {
	providerName  string
	lastSuccessAt time.Time
}

type Cache struct {
	cache   *lru.Cache[string, *sessionCacheEntry]
	service *services.SessionService
}

func NewCache(service *services.SessionService) *Cache {
	cache, err := lru.New[string, *sessionCacheEntry](sessionCacheSize)
	if err != nil {
		log.Printf("[ERROR] 创建会话缓存失败: %v\n", err)
		return &Cache{service: service}
	}
	sc := &Cache{
		cache:   cache,
		service: service,
	}
	if service != nil {
		service.RegisterCache(sc)
	}
	return sc
}

func (sc *Cache) cacheKey(platform, sessionID string) string {
	return fmt.Sprintf("%s:%s", platform, sessionID)
}

func (sc *Cache) GetSessionProvider(platform, sessionID string) (string, error) {
	if sessionID == "" {
		return "", nil
	}
	key := sc.cacheKey(platform, sessionID)
	if sc.cache != nil {
		if entry, ok := sc.cache.Get(key); ok {
			if !sc.isExpired(platform, entry.lastSuccessAt) {
				return entry.providerName, nil
			}
			sc.cache.Remove(key)
		}
	}
	providerName, err := sc.service.GetSessionProvider(platform, sessionID)
	if err != nil {
		return "", err
	}
	if providerName != "" && sc.cache != nil {
		sc.cache.Add(key, &sessionCacheEntry{
			providerName:  providerName,
			lastSuccessAt: time.Now(),
		})
	}
	return providerName, nil
}

func (sc *Cache) BindSessionToProvider(platform, sessionID, providerName string) error {
	if sessionID == "" || providerName == "" {
		return nil
	}
	if err := sc.service.BindSessionToProvider(platform, sessionID, providerName); err != nil {
		return err
	}
	if sc.cache != nil {
		key := sc.cacheKey(platform, sessionID)
		sc.cache.Add(key, &sessionCacheEntry{
			providerName:  providerName,
			lastSuccessAt: time.Now(),
		})
	}
	return nil
}

func (sc *Cache) UpdateSessionSuccess(platform, sessionID string) error {
	if sessionID == "" {
		return nil
	}
	if err := sc.service.UpdateSessionSuccess(platform, sessionID); err != nil {
		return err
	}
	if sc.cache != nil {
		key := sc.cacheKey(platform, sessionID)
		if entry, ok := sc.cache.Get(key); ok {
			entry.lastSuccessAt = time.Now()
		}
	}
	return nil
}

func (sc *Cache) InvalidateSession(platform, sessionID string) {
	if sc.cache != nil {
		key := sc.cacheKey(platform, sessionID)
		sc.cache.Remove(key)
	}
}

func (sc *Cache) isExpired(platform string, lastSuccessAt time.Time) bool {
	timeout := kernel.Platform(platform).SessionTimeout()
	return time.Since(lastSuccessAt) > timeout
}

func (sc *Cache) Stats() (size int, capacity int) {
	if sc.cache == nil {
		return 0, 0
	}
	return sc.cache.Len(), sessionCacheSize
}
