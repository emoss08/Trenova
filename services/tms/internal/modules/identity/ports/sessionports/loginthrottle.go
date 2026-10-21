package sessionports

import (
	"context"
	"strings"
	"time"

	"github.com/emoss08/trenova/pkg/requestmeta"
)

type LoginThrottleScope string

const LoginThrottleScopeAccount LoginThrottleScope = "account"

const LoginThrottleScopeIP LoginThrottleScope = "ip"

type LoginThrottlePolicy struct {
	AccountThreshold int
	AccountWindow    time.Duration
	AccountBaseLock  time.Duration
	AccountMaxLock   time.Duration
	IPThreshold      int
	IPWindow         time.Duration
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

type LoginThrottleKey struct {
	Account  string
	ClientIP string
}

func NewLoginThrottleKey(ctx context.Context, emailAddress string) LoginThrottleKey {
	key := LoginThrottleKey{Account: strings.ToLower(strings.TrimSpace(emailAddress))}
	if meta, ok := requestmeta.From(ctx); ok {
		key.ClientIP = strings.TrimSpace(meta.ClientIP)
	}
	return key
}

type LoginThrottleDecision struct {
	Blocked         bool
	Scope           LoginThrottleScope
	RetryAfter      time.Duration
	AccountFailures int
	IPFailures      int
	Tripped         bool
	TrippedScope    LoginThrottleScope
}

type LoginThrottleStore interface {
	Begin(ctx context.Context, key LoginThrottleKey) (*LoginThrottleDecision, error)
	Succeed(ctx context.Context, key LoginThrottleKey) error
}
