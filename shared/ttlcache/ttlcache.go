package ttlcache

import (
	"sync"
	"time"
)

type entry[V any] struct {
	value     V
	expiresAt time.Time
}

type Cache[K comparable, V any] struct {
	mu         sync.Mutex
	entries    map[K]entry[V]
	ttl        time.Duration
	maxEntries int
	now        func() time.Time
}

type Options struct {
	TTL        time.Duration
	MaxEntries int
	Now        func() time.Time
}

func New[K comparable, V any](opts Options) *Cache[K, V] {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	maxEntries := opts.MaxEntries
	if maxEntries <= 0 {
		maxEntries = 1
	}
	return &Cache[K, V]{
		entries:    make(map[K]entry[V], min(maxEntries, 1024)),
		ttl:        opts.TTL,
		maxEntries: maxEntries,
		now:        now,
	}
}

func (c *Cache[K, V]) Get(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	found, ok := c.entries[key]
	if !ok {
		var zero V
		return zero, false
	}
	if !c.now().Before(found.expiresAt) {
		delete(c.entries, key)
		var zero V
		return zero, false
	}
	return found.value, true
}

func (c *Cache[K, V]) Set(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := c.now()
	if _, exists := c.entries[key]; !exists && len(c.entries) >= c.maxEntries {
		c.evict(now)
	}
	c.entries[key] = entry[V]{value: value, expiresAt: now.Add(c.ttl)}
}

func (c *Cache[K, V]) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.entries)
}

func (c *Cache[K, V]) evict(now time.Time) {
	for key, found := range c.entries {
		if !now.Before(found.expiresAt) {
			delete(c.entries, key)
		}
	}
	if len(c.entries) < c.maxEntries {
		return
	}

	var oldestKey K
	var oldest time.Time
	first := true
	for key, found := range c.entries {
		if first || found.expiresAt.Before(oldest) {
			oldestKey, oldest, first = key, found.expiresAt, false
		}
	}
	delete(c.entries, oldestKey)
}
