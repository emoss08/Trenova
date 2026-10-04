package repositories

import (
	"context"
	"time"
)

type LoginThrottleScope string

const (
	LoginThrottleScopeAccount LoginThrottleScope = "account"
	LoginThrottleScopeIP      LoginThrottleScope = "ip"
)

type LoginThrottlePolicy struct {
	AccountThreshold int
	AccountWindow    time.Duration
	AccountBaseLock  time.Duration
	AccountMaxLock   time.Duration
	IPThreshold      int
	IPWindow         time.Duration
}

func DefaultLoginThrottlePolicy() LoginThrottlePolicy {
	return LoginThrottlePolicy{
		AccountThreshold: 5,
		AccountWindow:    15 * time.Minute,
		AccountBaseLock:  30 * time.Second,
		AccountMaxLock:   15 * time.Minute,
		IPThreshold:      20,
		IPWindow:         time.Hour,
	}
}

func (p LoginThrottlePolicy) AccountLockFor(failures int) time.Duration {
	if failures < p.AccountThreshold || p.AccountThreshold <= 0 {
		return 0
	}

	lock := p.AccountBaseLock
	for range min(failures-p.AccountThreshold, 16) {
		lock *= 2
		if lock >= p.AccountMaxLock {
			return p.AccountMaxLock
		}
	}

	return min(lock, p.AccountMaxLock)
}

type LoginThrottleKey struct {
	Account  string
	ClientIP string
}

type LoginThrottleDecision struct {
	Blocked         bool
	Scope           LoginThrottleScope
	RetryAfter      time.Duration
	AccountFailures int
	IPFailures      int
}

type LoginThrottleStore interface {
	Check(ctx context.Context, key LoginThrottleKey) (*LoginThrottleDecision, error)
	RecordFailure(ctx context.Context, key LoginThrottleKey) (*LoginThrottleDecision, error)
	Reset(ctx context.Context, account string) error
}
