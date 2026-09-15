package ratelimit

import (
	"context"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
)

type MemoryStoreOptions struct {
	CleanupInterval time.Duration
	Now             func() time.Time
}

type MemoryStore struct {
	now             func() time.Time
	cleanupInterval time.Duration

	mu          sync.Mutex
	buckets     map[string]int64
	lastCleanup time.Time
}

func NewMemoryStore(opts MemoryStoreOptions) *MemoryStore {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	cleanup := opts.CleanupInterval
	if cleanup <= 0 {
		cleanup = time.Minute
	}
	return &MemoryStore{
		now:             now,
		cleanupInterval: cleanup,
		buckets:         make(map[string]int64),
	}
}

func (s *MemoryStore) Check(
	_ context.Context,
	requests []repositories.RateLimitRequest,
) ([]repositories.RateLimitDecision, error) {
	if err := validateRequests(requests); err != nil {
		return nil, err
	}

	current := s.now()
	nowMicros := current.UnixMicro()

	s.mu.Lock()
	defer s.mu.Unlock()

	s.cleanupLocked(current, nowMicros)

	results := make([]gcraResult, len(requests))
	allAllowed := true
	for i := range requests {
		results[i] = evaluateGCRA(nowMicros, s.buckets[requests[i].Key], requests[i])
		if !results[i].allowed {
			allAllowed = false
		}
	}

	decisions := make([]repositories.RateLimitDecision, len(requests))
	for i := range requests {
		if allAllowed {
			s.buckets[requests[i].Key] = results[i].newTAT
		}
		decisions[i] = decisionFrom(requests[i], results[i])
	}

	return decisions, nil
}

func (s *MemoryStore) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.buckets)
}

func (s *MemoryStore) cleanupLocked(current time.Time, nowMicros int64) {
	if !s.lastCleanup.IsZero() && current.Sub(s.lastCleanup) < s.cleanupInterval {
		return
	}
	for key, tat := range s.buckets {
		if tat <= nowMicros {
			delete(s.buckets, key)
		}
	}
	s.lastCleanup = current
}
