package expression

import (
	"container/list"
	"sync"
)

type LRUCache struct {
	capacity  int
	cache     map[string]*list.Element
	lru       *list.List
	mu        sync.RWMutex
	hits      int64
	misses    int64
	evictions int64
}

type cacheEntry struct {
	key        string
	expression *CompiledExpression
}

func NewLRUCache(capacity int) *LRUCache {
	if capacity <= 0 {
		capacity = 100
	}

	return &LRUCache{
		capacity: capacity,
		cache:    make(map[string]*list.Element),
		lru:      list.New(),
	}
}

func (c *LRUCache) Get(key string) (*CompiledExpression, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.cache[key]; ok {
		c.lru.MoveToFront(elem)
		c.hits++

		entry, eOk := elem.Value.(*cacheEntry)
		if !eOk {
			return nil, false
		}

		return entry.expression, true
	}

	c.misses++
	return nil, false
}

func (c *LRUCache) Put(key string, expr *CompiledExpression) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.cache[key]; ok {
		c.lru.MoveToFront(elem)
		entry, eOk := elem.Value.(*cacheEntry)
		if !eOk {
			return
		}

		entry.expression = expr
		return
	}

	entry := &cacheEntry{
		key:        key,
		expression: expr,
	}
	elem := c.lru.PushFront(entry)
	c.cache[key] = elem

	if c.lru.Len() > c.capacity {
		c.evictLRU()
	}
}

func (c *LRUCache) evictLRU() {
	elem := c.lru.Back()
	if elem != nil {
		c.lru.Remove(elem)
		entry, eOk := elem.Value.(*cacheEntry)
		if !eOk {
			return
		}

		delete(c.cache, entry.key)
		c.evictions++
	}
}

func (c *LRUCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = make(map[string]*list.Element)
	c.lru = list.New()
}

func (c *LRUCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.lru.Len()
}

func (c *LRUCache) Capacity() int {
	return c.capacity
}

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

type CacheStats struct {
	Hits      int64
	Misses    int64
	Evictions int64
	Size      int
	Capacity  int
	HitRate   float64
}

func (c *LRUCache) Resize(newCapacity int) {
	if newCapacity <= 0 {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.capacity = newCapacity

	for c.lru.Len() > c.capacity {
		c.evictLRU()
	}
}

type EvictionCallback func(key string, expr *CompiledExpression)

type LRUCacheWithCallback struct {
	*LRUCache
	callback EvictionCallback
}

func NewLRUCacheWithCallback(capacity int, callback EvictionCallback) *LRUCacheWithCallback {
	return &LRUCacheWithCallback{
		LRUCache: NewLRUCache(capacity),
		callback: callback,
	}
}

func (c *LRUCacheWithCallback) Put(key string, expr *CompiledExpression) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.cache[key]; ok {
		c.lru.MoveToFront(elem)
		entry, eOk := elem.Value.(*cacheEntry)
		if !eOk {
			return
		}

		entry.expression = expr
		return
	}

	entry := &cacheEntry{
		key:        key,
		expression: expr,
	}
	elem := c.lru.PushFront(entry)
	c.cache[key] = elem

	if c.lru.Len() > c.capacity {
		c.evictLRUWithCallback()
	}
}

func (c *LRUCacheWithCallback) evictLRUWithCallback() {
	elem := c.lru.Back()
	if elem != nil {
		c.lru.Remove(elem)
		entry, eOk := elem.Value.(*cacheEntry)
		if !eOk {
			return
		}

		delete(c.cache, entry.key)
		c.evictions++

		if c.callback != nil {
			c.callback(entry.key, entry.expression)
		}
	}
}

func (c *LRUCache) Preload(expressions map[string]*CompiledExpression) {
	for key, expr := range expressions {
		c.Put(key, expr)
	}
}

func (c *LRUCache) GetMultiple(keys []string) map[string]*CompiledExpression {
	result := make(map[string]*CompiledExpression)

	for _, key := range keys {
		if expr, ok := c.Get(key); ok {
			result[key] = expr
		}
	}

	return result
}

func (c *LRUCache) Contains(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	_, ok := c.cache[key]
	return ok
}
