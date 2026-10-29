package authstore

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/ports/sessionports"
	"github.com/emoss08/trenova/shared/tokenutils"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	loginThrottlePrefix       = "login_throttle:{auth}"
	loginThrottleUnknown      = "unknown"
	loginThrottleResultFields = 6
	loginThrottleScopeIPCode  = 2
	loginThrottleLockSteps    = 18
)

type LoginThrottleStoreParams struct {
	fx.In

	Client *redis.Client
	Logger *zap.Logger
}

type loginThrottleStore struct {
	client    *redis.Client
	policy    sessionports.LoginThrottlePolicy
	beginArgs []any
	l         *zap.Logger
}

func loginThrottleBeginArgs(policy sessionports.LoginThrottlePolicy) []any {
	args := make([]any, 0, 4+loginThrottleLockSteps)
	args = append(args,
		policy.IPThreshold,
		policy.AccountWindow.Milliseconds(),
		policy.IPWindow.Milliseconds(),
		policy.AccountThreshold,
	)
	if policy.AccountThreshold <= 0 {
		return args
	}
	for step := range loginThrottleLockSteps {
		args = append(args, policy.AccountLockFor(policy.AccountThreshold+step).Milliseconds())
	}
	return args
}

func NewLoginThrottleStore(p LoginThrottleStoreParams) sessionports.LoginThrottleStore {
	return NewLoginThrottleStoreWithPolicy(
		p.Client,
		sessionports.DefaultLoginThrottlePolicy(),
		p.Logger,
	)
}

func NewLoginThrottleStoreWithPolicy(
	client *redis.Client,
	policy sessionports.LoginThrottlePolicy,
	logger *zap.Logger,
) sessionports.LoginThrottleStore {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &loginThrottleStore{
		client:    client,
		policy:    policy,
		beginArgs: loginThrottleBeginArgs(policy),
		l:         logger.Named("redis.login-throttle"),
	}
}

var loginThrottleBeginScript = redis.NewScript(`
local lockTTL = redis.call('PTTL', KEYS[1])
if lockTTL > 0 then
	return {1, 1, lockTTL, tonumber(redis.call('GET', KEYS[2]) or '0'), 0, 0}
end
local ipThreshold = tonumber(ARGV[1])
local ipCount = tonumber(redis.call('GET', KEYS[3]) or '0')
if ipThreshold > 0 and ipCount >= ipThreshold then
	return {1, 2, redis.call('PTTL', KEYS[3]), 0, ipCount, 0}
end
local accountFailures = redis.call('INCR', KEYS[2])
redis.call('PEXPIRE', KEYS[2], ARGV[2])
local ipFailures = redis.call('INCR', KEYS[3])
if redis.call('PTTL', KEYS[3]) < 0 then
	redis.call('PEXPIRE', KEYS[3], ARGV[3])
end
local threshold = tonumber(ARGV[4])
local steps = #ARGV - 4
if threshold > 0 and steps > 0 and accountFailures >= threshold then
	local lock = tonumber(ARGV[4 + math.min(accountFailures - threshold + 1, steps)])
	redis.call('SET', KEYS[1], '1', 'PX', lock)
	return {0, 1, lock, accountFailures, ipFailures, 1}
end
if ipThreshold > 0 and ipFailures >= ipThreshold then
	return {0, 2, redis.call('PTTL', KEYS[3]), accountFailures, ipFailures, 1}
end
return {0, 0, 0, accountFailures, ipFailures, 0}
`)

var loginThrottleSucceedScript = redis.NewScript(`
redis.call('DEL', KEYS[1], KEYS[2])
local ipFailures = tonumber(redis.call('GET', KEYS[3]) or '0')
if ipFailures > 0 then
	redis.call('DECR', KEYS[3])
end
return 1
`)

func (s *loginThrottleStore) Begin(
	ctx context.Context,
	key sessionports.LoginThrottleKey,
) (*sessionports.LoginThrottleDecision, error) {
	result, err := loginThrottleBeginScript.Run(
		ctx,
		s.client,
		[]string{
			s.accountLockKey(key.Account),
			s.accountFailuresKey(key.Account),
			s.ipFailuresKey(key.ClientIP),
		},
		s.beginArgs...,
	).Int64Slice()
	if err != nil {
		return nil, fmt.Errorf("begin login attempt: %w", err)
	}

	return s.decisionFrom(result)
}

func (s *loginThrottleStore) decisionFrom(
	result []int64,
) (*sessionports.LoginThrottleDecision, error) {
	if len(result) != loginThrottleResultFields {
		return nil, fmt.Errorf("begin login attempt: unexpected result %v", result)
	}

	scope := throttleScope(result[1])
	window := time.Duration(result[2]) * time.Millisecond
	decision := &sessionports.LoginThrottleDecision{
		AccountFailures: int(result[3]),
		IPFailures:      int(result[4]),
	}
	switch {
	case result[0] == 1:
		decision.Blocked = true
		decision.Scope = scope
		decision.RetryAfter = positiveOr(window, s.windowFor(scope))
	case result[5] == 1:
		decision.Tripped = true
		decision.TrippedScope = scope
		decision.RetryAfter = positiveOr(window, s.windowFor(scope))
	}

	return decision, nil
}

func (s *loginThrottleStore) windowFor(scope sessionports.LoginThrottleScope) time.Duration {
	if scope == sessionports.LoginThrottleScopeIP {
		return s.policy.IPWindow
	}
	return s.policy.AccountBaseLock
}

func throttleScope(code int64) sessionports.LoginThrottleScope {
	if code == loginThrottleScopeIPCode {
		return sessionports.LoginThrottleScopeIP
	}
	return sessionports.LoginThrottleScopeAccount
}

func (s *loginThrottleStore) Succeed(
	ctx context.Context,
	key sessionports.LoginThrottleKey,
) error {
	if err := loginThrottleSucceedScript.Run(
		ctx,
		s.client,
		[]string{
			s.accountFailuresKey(key.Account),
			s.accountLockKey(key.Account),
			s.ipFailuresKey(key.ClientIP),
		},
	).Err(); err != nil {
		return fmt.Errorf("clear login throttle: %w", err)
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

func positiveOr(value, fallback time.Duration) time.Duration {
	if value > 0 {
		return value
	}

	return fallback
}
