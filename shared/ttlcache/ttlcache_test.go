package ttlcache

import (
	"sync"
	"testing"
	"time"
)

type clock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *clock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func TestGetReturnsWhatWasSetUntilItExpires(t *testing.T) {
	t.Parallel()

	c := &clock{now: time.Unix(1_800_000_000, 0)}
	cache := New[string, int](Options{TTL: time.Minute, MaxEntries: 10, Now: c.Now})

	cache.Set("a", 1)
	if got, ok := cache.Get("a"); !ok || got != 1 {
		t.Fatalf("expected a=1, got %d, %v", got, ok)
	}

	c.Advance(59 * time.Second)
	if _, ok := cache.Get("a"); !ok {
		t.Fatal("expected a to live until its ttl")
	}

	c.Advance(time.Second)
	if _, ok := cache.Get("a"); ok {
		t.Fatal("expected a to expire at its ttl")
	}
	if cache.Len() != 0 {
		t.Fatalf("expected the expired entry to be dropped, got %d", cache.Len())
	}
}

func TestSetEvictsTheOldestOnceFull(t *testing.T) {
	t.Parallel()

	c := &clock{now: time.Unix(1_800_000_000, 0)}
	cache := New[string, int](Options{TTL: time.Hour, MaxEntries: 2, Now: c.Now})

	cache.Set("a", 1)
	c.Advance(time.Second)
	cache.Set("b", 2)
	c.Advance(time.Second)
	cache.Set("c", 3)

	if _, ok := cache.Get("a"); ok {
		t.Fatal("expected the oldest entry to be evicted")
	}
	if got, ok := cache.Get("c"); !ok || got != 3 {
		t.Fatalf("expected c=3, got %d, %v", got, ok)
	}
	if cache.Len() != 2 {
		t.Fatalf("expected the cache to stay at its cap, got %d", cache.Len())
	}

	cache.Set("b", 20)
	if got, _ := cache.Get("b"); got != 20 {
		t.Fatalf("expected overwriting a present key to keep it, got %d", got)
	}
}

func TestEvictPrefersExpiredEntries(t *testing.T) {
	t.Parallel()

	c := &clock{now: time.Unix(1_800_000_000, 0)}
	cache := New[string, int](Options{TTL: time.Minute, MaxEntries: 2, Now: c.Now})

	cache.Set("old", 1)
	c.Advance(2 * time.Minute)
	cache.Set("fresh", 2)
	cache.Set("new", 3)

	if _, ok := cache.Get("fresh"); !ok {
		t.Fatal("expected the live entry to survive when an expired one could go")
	}
}

func TestCacheIsSafeForConcurrentUse(t *testing.T) {
	t.Parallel()

	cache := New[int, int](Options{TTL: time.Minute, MaxEntries: 64})
	var wg sync.WaitGroup
	for worker := range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range 500 {
				cache.Set(worker*1000+i, i)
				cache.Get(worker*1000 + i)
			}
		}()
	}
	wg.Wait()
	if cache.Len() > 64 {
		t.Fatalf("expected the cap to hold under concurrency, got %d", cache.Len())
	}
}
