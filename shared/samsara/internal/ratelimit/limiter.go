package ratelimit

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"
)

type Limiter interface {
	Wait(ctx context.Context, method, path string) error
	Penalize(method, path string, d time.Duration)
}

type bucket struct {
	limiter      *rate.Limiter
	blockedUntil atomic.Int64
}

type Set struct {
	global   *rate.Limiter
	now      func() time.Time
	mu       sync.RWMutex
	buckets  map[string]*bucket
	lastUsed atomic.Int64
}

func NewSet() *Set {
	return newSet(time.Now)
}

func newSet(now func() time.Time) *Set {
	s := &Set{
		global:  rate.NewLimiter(rate.Limit(TokenRequestsPerSecond), 1),
		now:     now,
		buckets: make(map[string]*bucket),
	}
	s.touch()
	return s
}

func (s *Set) Wait(ctx context.Context, method, path string) error {
	s.touch()
	if endpoint, ok := Classify(method, path); ok {
		b := s.bucket(endpoint)
		if err := s.waitUnblocked(ctx, b); err != nil {
			return err
		}
		if b.limiter != nil {
			if err := b.limiter.Wait(ctx); err != nil {
				return fmt.Errorf(
					"samsara %s rate limit (%s): %w",
					endpoint.Key,
					endpoint.Tier,
					err,
				)
			}
		}
	}
	if err := s.global.Wait(ctx); err != nil {
		return fmt.Errorf("samsara token rate limit: %w", err)
	}
	return nil
}

func (s *Set) Penalize(method, path string, d time.Duration) {
	if d <= 0 {
		return
	}
	endpoint, ok := Classify(method, path)
	if !ok {
		return
	}
	b := s.bucket(endpoint)
	until := s.now().Add(d).UnixNano()
	for {
		current := b.blockedUntil.Load()
		if current >= until || b.blockedUntil.CompareAndSwap(current, until) {
			return
		}
	}
}

func (s *Set) waitUnblocked(ctx context.Context, b *bucket) error {
	for {
		until := b.blockedUntil.Load()
		if until == 0 {
			return nil
		}
		remaining := time.Duration(until - s.now().UnixNano())
		if remaining <= 0 {
			return nil
		}
		timer := time.NewTimer(remaining)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func (s *Set) bucket(endpoint Endpoint) *bucket {
	s.mu.RLock()
	b, ok := s.buckets[endpoint.Key]
	s.mu.RUnlock()
	if ok {
		return b
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if b, ok = s.buckets[endpoint.Key]; ok {
		return b
	}
	b = &bucket{}
	if limit := endpoint.Tier.Limit(); limit != rate.Inf {
		b.limiter = rate.NewLimiter(limit, 1)
	}
	s.buckets[endpoint.Key] = b
	return b
}

func (s *Set) touch() {
	s.lastUsed.Store(s.now().UnixNano())
}

func (s *Set) idleSince(now time.Time) time.Duration {
	return time.Duration(now.UnixNano() - s.lastUsed.Load())
}
