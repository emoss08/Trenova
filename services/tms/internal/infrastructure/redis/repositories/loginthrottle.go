package repositories

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	corerepositories "github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/shared/tokenutils"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	loginThrottlePrefix  = "login_throttle:{auth}"
	loginThrottleUnknown = "unknown"
)

type LoginThrottleStoreParams struct {
	fx.In

	Client *redis.Client
	Logger *zap.Logger
}

type loginThrottleStore struct {
	client *redis.Client
	policy corerepositories.LoginThrottlePolicy
	l      *zap.Logger
}

func NewLoginThrottleStore(p LoginThrottleStoreParams) corerepositories.LoginThrottleStore {
	return NewLoginThrottleStoreWithPolicy(
		p.Client,
		corerepositories.DefaultLoginThrottlePolicy(),
		p.Logger,
	)
}

func NewLoginThrottleStoreWithPolicy(
	client *redis.Client,
	policy corerepositories.LoginThrottlePolicy,
	logger *zap.Logger,
) corerepositories.LoginThrottleStore {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &loginThrottleStore{
		client: client,
		policy: policy,
		l:      logger.Named("redis.login-throttle"),
	}
}

func (s *loginThrottleStore) Check(
	ctx context.Context,
	key corerepositories.LoginThrottleKey,
) (*corerepositories.LoginThrottleDecision, error) {
	pipe := s.client.Pipeline()
	lockTTL := pipe.PTTL(ctx, s.accountLockKey(key.Account))
	accountFailures := pipe.Get(ctx, s.accountFailuresKey(key.Account))
	ipFailures := pipe.Get(ctx, s.ipFailuresKey(key.ClientIP))
	ipTTL := pipe.PTTL(ctx, s.ipFailuresKey(key.ClientIP))
	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		return nil, fmt.Errorf("check login throttle: %w", err)
	}

	decision := &corerepositories.LoginThrottleDecision{
		AccountFailures: intOrZero(accountFailures),
		IPFailures:      intOrZero(ipFailures),
	}

	if ttl := lockTTL.Val(); ttl > 0 {
		decision.Blocked = true
		decision.Scope = corerepositories.LoginThrottleScopeAccount
		decision.RetryAfter = ttl
		return decision, nil
	}

	if decision.IPFailures >= s.policy.IPThreshold && s.policy.IPThreshold > 0 {
		decision.Blocked = true
		decision.Scope = corerepositories.LoginThrottleScopeIP
		decision.RetryAfter = positiveOr(ipTTL.Val(), s.policy.IPWindow)
	}

	return decision, nil
}

func (s *loginThrottleStore) RecordFailure(
	ctx context.Context,
	key corerepositories.LoginThrottleKey,
) (*corerepositories.LoginThrottleDecision, error) {
	accountKey := s.accountFailuresKey(key.Account)
	ipKey := s.ipFailuresKey(key.ClientIP)

	pipe := s.client.TxPipeline()
	accountFailures := pipe.Incr(ctx, accountKey)
	pipe.Expire(ctx, accountKey, s.policy.AccountWindow)
	ipFailures := pipe.Incr(ctx, ipKey)
	pipe.ExpireNX(ctx, ipKey, s.policy.IPWindow)
	ipTTL := pipe.PTTL(ctx, ipKey)
	if _, err := pipe.Exec(ctx); err != nil {
		return nil, fmt.Errorf("record login failure: %w", err)
	}

	decision := &corerepositories.LoginThrottleDecision{
		AccountFailures: int(accountFailures.Val()),
		IPFailures:      int(ipFailures.Val()),
	}

	if lock := s.policy.AccountLockFor(decision.AccountFailures); lock > 0 {
		if err := s.client.Set(ctx, s.accountLockKey(key.Account), "1", lock).Err(); err != nil {
			return nil, fmt.Errorf("lock account after login failures: %w", err)
		}
		decision.Blocked = true
		decision.Scope = corerepositories.LoginThrottleScopeAccount
		decision.RetryAfter = lock
		return decision, nil
	}

	if s.policy.IPThreshold > 0 && decision.IPFailures >= s.policy.IPThreshold {
		decision.Blocked = true
		decision.Scope = corerepositories.LoginThrottleScopeIP
		decision.RetryAfter = positiveOr(ipTTL.Val(), s.policy.IPWindow)
	}

	return decision, nil
}

func (s *loginThrottleStore) Reset(ctx context.Context, account string) error {
	if err := s.client.Del(ctx, s.accountFailuresKey(account), s.accountLockKey(account)).Err(); err != nil {
		return fmt.Errorf("reset login throttle: %w", err)
	}

	return nil
}

func (s *loginThrottleStore) accountFailuresKey(account string) string {
	return loginThrottlePrefix + ":account:" + accountDigest(account) + ":failures"
}

func (s *loginThrottleStore) accountLockKey(account string) string {
	return loginThrottlePrefix + ":account:" + accountDigest(account) + ":lock"
}

func (s *loginThrottleStore) ipFailuresKey(clientIP string) string {
	ip := strings.TrimSpace(clientIP)
	if ip == "" {
		ip = loginThrottleUnknown
	}

	return loginThrottlePrefix + ":ip:" + ip
}

func accountDigest(account string) string {
	return tokenutils.Hash(strings.ToLower(strings.TrimSpace(account)))
}

func intOrZero(cmd *redis.StringCmd) int {
	value, err := cmd.Int()
	if err != nil {
		return 0
	}

	return value
}

func positiveOr(value, fallback time.Duration) time.Duration {
	if value > 0 {
		return value
	}

	return fallback
}
