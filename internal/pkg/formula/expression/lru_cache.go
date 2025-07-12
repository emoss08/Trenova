package expression

import (
	"container/list"
	"sync"
)

// LRUCache implements a thread-safe LRU cache for compiled expressions
type LRUCache struct {
	capacity int
	cache    map[string]*list.Element
	lru      *list.List
	mu       sync.RWMutex

	// Metrics
	hits      int64
	misses    int64
	evictions int64
}

// cacheEntry holds the cached data
type cacheEntry struct {
	key        string
	expression *CompiledExpression
}

// NewLRUCache creates a new LRU cache with the specified capacity
func NewLRUCache(capacity int) *LRUCache {
	if capacity <= 0 {
		capacity = 100 // Default capacity
	}

	return &LRUCache{
		capacity: capacity,
		cache:    make(map[string]*list.Element),
		lru:      list.New(),
	}
}

// Get retrieves an expression from the cache
func (c *LRUCache) Get(key string) (*CompiledExpression, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.cache[key]; ok {
		// Move to front (most recently used)
		c.lru.MoveToFront(elem)
		c.hits++
		return elem.Value.(*cacheEntry).expression, true
	}

	c.misses++
	return nil, false
}

// Put adds or updates an expression in the cache
func (c *LRUCache) Put(key string, expr *CompiledExpression) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if key already exists
	if elem, ok := c.cache[key]; ok {
		// Update existing entry and move to front
		c.lru.MoveToFront(elem)
		elem.Value.(*cacheEntry).expression = expr
		return
	}

	// Add new entry
	entry := &cacheEntry{
		key:        key,
		expression: expr,
	}
	elem := c.lru.PushFront(entry)
	c.cache[key] = elem

	// Evict least recently used if at capacity
	if c.lru.Len() > c.capacity {
		c.evictLRU()
	}
}

// evictLRU removes the least recently used entry
func (c *LRUCache) evictLRU() {
	elem := c.lru.Back()
	if elem != nil {
		c.lru.Remove(elem)
		entry := elem.Value.(*cacheEntry)
		delete(c.cache, entry.key)
		c.evictions++
	}
}

// Clear removes all entries from the cache
func (c *LRUCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = make(map[string]*list.Element)
	c.lru = list.New()
}

// Size returns the current number of entries in the cache
func (c *LRUCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.lru.Len()
}

// Capacity returns the maximum capacity of the cache
func (c *LRUCache) Capacity() int {
	return c.capacity
}

// Stats returns cache statistics
func (c *LRUCache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	total := c.hits + c.misses
	hitRate := 0.0
	if total > 0 {
		hitRate = float64(c.hits) / float64(total)
	}

	return CacheStats{
		Hits:      c.hits,
		Misses:    c.misses,
		Evictions: c.evictions,
		Size:      c.lru.Len(),
		Capacity:  c.capacity,
		HitRate:   hitRate,
	}
}

// CacheStats holds cache performance statistics
type CacheStats struct {
	Hits      int64   // Number of cache hits
	Misses    int64   // Number of cache misses
	Evictions int64   // Number of evictions
	Size      int     // Current number of entries
	Capacity  int     // Maximum capacity
	HitRate   float64 // Hit rate (0.0 to 1.0)
}

// Resize changes the capacity of the cache
func (c *LRUCache) Resize(newCapacity int) {
	if newCapacity <= 0 {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.capacity = newCapacity

	// Evict entries if necessary
	for c.lru.Len() > c.capacity {
		c.evictLRU()
	}
}

// SetEvictionCallback sets a callback function to be called when an entry is evicted
type EvictionCallback func(key string, expr *CompiledExpression)

// WithEvictionCallback creates a new LRU cache with an eviction callback
type LRUCacheWithCallback struct {
	*LRUCache
	callback EvictionCallback
}

// NewLRUCacheWithCallback creates an LRU cache with eviction callback
func NewLRUCacheWithCallback(capacity int, callback EvictionCallback) *LRUCacheWithCallback {
	return &LRUCacheWithCallback{
		LRUCache: NewLRUCache(capacity),
		callback: callback,
	}
}

// Put adds or updates an expression in the cache with callback support
func (c *LRUCacheWithCallback) Put(key string, expr *CompiledExpression) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if key already exists
	if elem, ok := c.cache[key]; ok {
		// Update existing entry and move to front
		c.lru.MoveToFront(elem)
		elem.Value.(*cacheEntry).expression = expr
		return
	}

	// Add new entry
	entry := &cacheEntry{
		key:        key,
		expression: expr,
	}
	elem := c.lru.PushFront(entry)
	c.cache[key] = elem

	// Evict least recently used if at capacity
	if c.lru.Len() > c.capacity {
		c.evictLRUWithCallback()
	}
}

// evictLRUWithCallback removes the least recently used entry and calls the callback
func (c *LRUCacheWithCallback) evictLRUWithCallback() {
	elem := c.lru.Back()
	if elem != nil {
		c.lru.Remove(elem)
		entry := elem.Value.(*cacheEntry)
		delete(c.cache, entry.key)
		c.evictions++

		// Call eviction callback if set
		if c.callback != nil {
			c.callback(entry.key, entry.expression)
		}
	}
}

// Preload adds multiple expressions to the cache
func (c *LRUCache) Preload(expressions map[string]*CompiledExpression) {
	for key, expr := range expressions {
		c.Put(key, expr)
	}
}

// GetMultiple retrieves multiple expressions from the cache
func (c *LRUCache) GetMultiple(keys []string) map[string]*CompiledExpression {
	result := make(map[string]*CompiledExpression)

	for _, key := range keys {
		if expr, ok := c.Get(key); ok {
			result[key] = expr
		}
	}

	return result
}

// Contains checks if a key exists in the cache without updating LRU order
func (c *LRUCache) Contains(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	_, ok := c.cache[key]
	return ok
}
