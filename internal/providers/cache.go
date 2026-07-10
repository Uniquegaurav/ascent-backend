package providers

import (
	"fmt"
	"sync"
	"time"
)

// ttlCache is a small in-memory TTL cache, copied from internal/places so the
// external content providers never hammer the (free, but rate-limited) upstream
// APIs. Content here changes slowly, so long TTLs are safe. Single instance
// today; swap for Redis when the API scales out.
type ttlCache struct {
	mu  sync.Mutex
	ttl time.Duration
	max int
	m   map[string]cacheEntry
}

type cacheEntry struct {
	val     any
	expires time.Time
}

func newTTLCache(ttl time.Duration, max int) *ttlCache {
	return &ttlCache{ttl: ttl, max: max, m: make(map[string]cacheEntry)}
}

func (c *ttlCache) get(key string) (any, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.m[key]
	if !ok || time.Now().After(e.expires) {
		return nil, false
	}
	return e.val, true
}

func (c *ttlCache) put(key string, v any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.m) >= c.max {
		// Evict expired first; if still full, drop arbitrary entries. Crude but
		// bounded, and eviction pressure is rare at current scale.
		now := time.Now()
		for k, e := range c.m {
			if now.After(e.expires) {
				delete(c.m, k)
			}
		}
		for k := range c.m {
			if len(c.m) < c.max {
				break
			}
			delete(c.m, k)
		}
	}
	c.m[key] = cacheEntry{val: v, expires: time.Now().Add(c.ttl)}
}

// geoCell quantizes coordinates to ~5km buckets so nearby lookups share cached
// results instead of each triggering a fresh upstream query.
func geoCell(lat, lng float64) string {
	const step = 0.05
	return fmt.Sprintf("%.0f:%.0f", lat/step, lng/step)
}
