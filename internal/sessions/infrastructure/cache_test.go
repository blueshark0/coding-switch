package infrastructure

import (
	"testing"
	"time"
)

func TestCacheRecordSessionSuccessThrottlesPendingUpdates(t *testing.T) {
	cache := NewCache(nil)

	if !cache.RecordSessionSuccess("codex", "session-1", "alpha") {
		t.Fatal("expected first success to schedule persistence")
	}
	if cache.RecordSessionSuccess("codex", "session-1", "alpha") {
		t.Fatal("expected pending success not to schedule again")
	}

	cache.ClearPendingSessionSuccess("codex", "session-1")

	if !cache.RecordSessionSuccess("codex", "session-1", "alpha") {
		t.Fatal("expected cleared pending flag to allow reschedule")
	}
}

func TestCacheRecordSessionSuccessRespectsPersistInterval(t *testing.T) {
	cache := NewCache(nil)
	key := cache.cacheKey("claude", "session-2")
	cache.cache.Add(key, &sessionCacheEntry{
		providerName:    "beta",
		lastPersistedAt: timeNow().Add(-30 * time.Second),
	})

	if cache.RecordSessionSuccess("claude", "session-2", "beta") {
		t.Fatal("expected recent persisted entry to be skipped")
	}

	entry, ok := cache.cache.Get(key)
	if !ok || entry == nil {
		t.Fatal("expected cache entry to remain present")
	}
	entry.lastPersistedAt = timeNow().Add(-sessionPersistInterval - time.Second)

	if !cache.RecordSessionSuccess("claude", "session-2", "beta") {
		t.Fatal("expected expired persisted entry to schedule update")
	}
}

func timeNow() time.Time {
	return time.Now()
}
