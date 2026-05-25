package infrastructure

import (
	"fmt"
	"log"
	"time"

	sessionapp "codeswitch/internal/sessions/application"
	"codeswitch/internal/shared/kernel"

	lru "github.com/hashicorp/golang-lru/v2"
)

const (
	sessionCacheSize       = 1000
	sessionPersistInterval = 60 * time.Second
)

type sessionCacheEntry struct {
	providerName     string
	lastSuccessAt    time.Time
	lastPersistedAt  time.Time
	pendingPersistAt time.Time
	generation       uint64
}

type Cache struct {
	cache   *lru.Cache[string, *sessionCacheEntry]
	service *sessionapp.Service
}

func NewCache(service *sessionapp.Service) *Cache {
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
	providerName, _, err := sc.GetSessionProviderSnapshot(platform, sessionID)
	return providerName, err
}

func (sc *Cache) GetSessionProviderSnapshot(platform, sessionID string) (string, uint64, error) {
	if sessionID == "" {
		return "", 0, nil
	}
	key := sc.cacheKey(platform, sessionID)
	generation := uint64(0)
	if sc.cache != nil {
		if entry, ok := sc.cache.Get(key); ok {
			generation = entry.generation
			if !sc.isExpired(platform, entry.lastSuccessAt) {
				return entry.providerName, generation, nil
			}
			entry.providerName = ""
			entry.lastSuccessAt = time.Time{}
			entry.lastPersistedAt = time.Time{}
			entry.pendingPersistAt = time.Time{}
		}
	}
	if sc.service == nil {
		return "", generation, nil
	}
	providerName, err := sc.service.GetSessionProvider(platform, sessionID)
	if err != nil {
		return "", generation, err
	}
	if providerName != "" && sc.cache != nil {
		if entry, ok := sc.cache.Get(key); ok && entry != nil {
			entry.providerName = providerName
			entry.lastSuccessAt = time.Now()
			entry.lastPersistedAt = time.Time{}
			return providerName, entry.generation, nil
		}
		sc.cache.Add(key, &sessionCacheEntry{
			providerName:    providerName,
			lastSuccessAt:   time.Now(),
			lastPersistedAt: time.Time{},
			generation:      generation,
		})
	}
	return providerName, generation, nil
}

func (sc *Cache) BindSessionToProvider(platform, sessionID, providerName string) error {
	return sc.BindSessionToProviderGeneration(platform, sessionID, providerName, sc.SessionGeneration(platform, sessionID))
}

func (sc *Cache) BindSessionToProviderGeneration(platform, sessionID, providerName string, generation uint64) error {
	if sessionID == "" || providerName == "" {
		return nil
	}
	if !sc.isCurrentGeneration(platform, sessionID, generation) {
		return nil
	}
	if sc.service != nil {
		if err := sc.service.BindSessionToProvider(platform, sessionID, providerName); err != nil {
			return err
		}
	}
	if sc.cache != nil {
		key := sc.cacheKey(platform, sessionID)
		now := time.Now()
		sc.cache.Add(key, &sessionCacheEntry{
			providerName:    providerName,
			lastSuccessAt:   now,
			lastPersistedAt: now,
			generation:      generation,
		})
	}
	return nil
}

func (sc *Cache) UpdateSessionSuccess(platform, sessionID string) error {
	return sc.UpdateSessionSuccessGeneration(platform, sessionID, sc.SessionGeneration(platform, sessionID))
}

func (sc *Cache) UpdateSessionSuccessGeneration(platform, sessionID string, generation uint64) error {
	if sessionID == "" {
		return nil
	}
	if !sc.isCurrentGeneration(platform, sessionID, generation) {
		sc.ClearPendingSessionSuccess(platform, sessionID)
		return nil
	}
	if sc.service != nil {
		if err := sc.service.UpdateSessionSuccess(platform, sessionID); err != nil {
			sc.ClearPendingSessionSuccess(platform, sessionID)
			return err
		}
	}
	if sc.cache != nil {
		now := time.Now()
		key := sc.cacheKey(platform, sessionID)
		if entry, ok := sc.cache.Get(key); ok {
			entry.lastSuccessAt = now
			entry.lastPersistedAt = now
			entry.pendingPersistAt = time.Time{}
		}
	}
	return nil
}

func (sc *Cache) RecordSessionSuccess(platform, sessionID, providerName string) bool {
	return sc.RecordSessionSuccessGeneration(platform, sessionID, providerName, sc.SessionGeneration(platform, sessionID))
}

func (sc *Cache) RecordSessionSuccessGeneration(platform, sessionID, providerName string, generation uint64) bool {
	if sessionID == "" {
		return false
	}
	if sc.cache == nil {
		return true
	}
	if !sc.isCurrentGeneration(platform, sessionID, generation) {
		return false
	}
	now := time.Now()
	key := sc.cacheKey(platform, sessionID)
	entry, ok := sc.cache.Get(key)
	if !ok || entry == nil {
		entry = &sessionCacheEntry{providerName: providerName, generation: generation}
		sc.cache.Add(key, entry)
	}
	if providerName != "" {
		entry.providerName = providerName
	}
	entry.lastSuccessAt = now

	referenceTime := entry.lastPersistedAt
	if entry.pendingPersistAt.After(referenceTime) {
		referenceTime = entry.pendingPersistAt
	}
	if !referenceTime.IsZero() && now.Sub(referenceTime) < sessionPersistInterval {
		return false
	}
	entry.pendingPersistAt = now
	return true
}

func (sc *Cache) InvalidateSession(platform, sessionID string) {
	if sc.cache != nil {
		key := sc.cacheKey(platform, sessionID)
		entry, ok := sc.cache.Get(key)
		if !ok || entry == nil {
			sc.cache.Add(key, &sessionCacheEntry{generation: 1})
			return
		}
		entry.generation++
		entry.providerName = ""
		entry.lastSuccessAt = time.Time{}
		entry.lastPersistedAt = time.Time{}
		entry.pendingPersistAt = time.Time{}
	}
}

func (sc *Cache) isExpired(platform string, lastSuccessAt time.Time) bool {
	timeout := kernel.Platform(platform).SessionTimeout()
	return time.Since(lastSuccessAt) > timeout
}

func (sc *Cache) ClearPendingSessionSuccess(platform, sessionID string) {
	if sc.cache == nil || sessionID == "" {
		return
	}
	key := sc.cacheKey(platform, sessionID)
	entry, ok := sc.cache.Get(key)
	if !ok || entry == nil {
		return
	}
	entry.pendingPersistAt = time.Time{}
}

func (sc *Cache) SessionGeneration(platform, sessionID string) uint64 {
	if sc.cache == nil || sessionID == "" {
		return 0
	}
	key := sc.cacheKey(platform, sessionID)
	if entry, ok := sc.cache.Get(key); ok && entry != nil {
		return entry.generation
	}
	return 0
}

func (sc *Cache) IsCurrentGeneration(platform, sessionID string, generation uint64) bool {
	return sc.isCurrentGeneration(platform, sessionID, generation)
}

func (sc *Cache) isCurrentGeneration(platform, sessionID string, generation uint64) bool {
	if sc.cache == nil || sessionID == "" {
		return true
	}
	return sc.SessionGeneration(platform, sessionID) == generation
}

func (sc *Cache) Stats() (size int, capacity int) {
	if sc.cache == nil {
		return 0, 0
	}
	return sc.cache.Len(), sessionCacheSize
}
