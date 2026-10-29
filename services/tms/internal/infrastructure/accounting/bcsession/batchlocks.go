package bcsession

import (
	"context"
	"sync"
)

type batchLock struct {
	held  chan struct{}
	users int
}

type BatchLocks struct {
	mu      sync.Mutex
	byBatch map[string]*batchLock
}

func NewBatchLocks() *BatchLocks {
	return &BatchLocks{byBatch: make(map[string]*batchLock)}
}

func (l *BatchLocks) acquire(ctx context.Context, key string) (func(), error) {
	l.mu.Lock()
	entry, ok := l.byBatch[key]
	if !ok {
		entry = &batchLock{held: make(chan struct{}, 1)}
		l.byBatch[key] = entry
	}
	entry.users++
	l.mu.Unlock()

	select {
	case entry.held <- struct{}{}:
	case <-ctx.Done():
		l.forget(key, entry)
		return nil, ctx.Err()
	}
	var once sync.Once
	return func() {
		once.Do(func() {
			<-entry.held
			l.forget(key, entry)
		})
	}, nil
}

func (l *BatchLocks) forget(key string, entry *batchLock) {
	l.mu.Lock()
	defer l.mu.Unlock()
	entry.users--
	if entry.users == 0 && l.byBatch[key] == entry {
		delete(l.byBatch, key)
	}
}

func (c *Session) LockBatch(ctx context.Context, key string) (func(), error) {
	return c.locks.acquire(ctx, key)
}
