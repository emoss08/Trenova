package outboundlimit

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/ratelimit"
	"github.com/emoss08/trenova/shared/restx"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	redisKeyPrefix          = "outbound:"
	backgroundKeySuffix     = ":bg"
	backgroundSharePercent  = 60
	percentDenominator      = 100
	defaultOutboundPeriod   = time.Minute
	maxPolicyRequestsPerRun = 2
)

type Params struct {
	fx.In

	Config *config.Config
	Client *redis.Client `optional:"true"`
	Logger *zap.Logger
}

var _ restx.Limiter = (*Limiter)(nil)

type Limiter struct {
	store repositories.RateLimitStore
}

func New(p Params) (restx.Limiter, error) {
	rateCfg := &p.Config.Security.RateLimit
	replicas := max(p.Config.CarrierIntelligence.GetReplicaHint(), 1)
	memory := newReplicaScaledStore(
		ratelimit.NewMemoryStore(
			ratelimit.MemoryStoreOptions{CleanupInterval: rateCfg.GetCleanupInterval()},
		),
		replicas,
	)

	logger := p.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	log := logger.Named("carrierintel.outbound")

	if p.Client == nil || rateCfg.GetStore() == config.RateLimitStoreMemory {
		log.Warn(
			"outbound carrier intelligence limits use the in-process store",
			zap.Int("replicaHint", replicas),
		)
		return newLimiter(memory), nil
	}

	primary, err := ratelimit.NewRedisStore(p.Client, redisKeyPrefix)
	if err != nil {
		return nil, fmt.Errorf("configure outbound rate limit store: %w", err)
	}

	return newLimiter(ratelimit.NewResilientStore(ratelimit.ResilientStoreOptions{
		Primary:     primary,
		Fallback:    memory,
		FailureMode: rateCfg.GetFailureMode(),
		Timeout:     rateCfg.GetStoreTimeout(),
		Logger:      log,
	})), nil
}

func newLimiter(store repositories.RateLimitStore) *Limiter {
	return &Limiter{store: store}
}

func (l *Limiter) Acquire(ctx context.Context, bucket restx.Bucket) error {
	if bucket.Key == "" || bucket.Limit <= 0 {
		return nil
	}

	period := bucket.Period
	if period <= 0 {
		period = defaultOutboundPeriod
	}
	burst := max(bucket.Burst, 1)
	cost := max(bucket.Cost, 1)

	requests := make([]repositories.RateLimitRequest, 1, maxPolicyRequestsPerRun)
	requests[0] = repositories.RateLimitRequest{
		Key:    bucket.Key,
		Policy: repositories.RateLimitPolicy{Rate: bucket.Limit, Period: period, Burst: burst},
		Cost:   cost,
	}
	if !carrierintel.IsInteractiveContext(ctx) {
		requests = append(requests, repositories.RateLimitRequest{
			Key: bucket.Key + backgroundKeySuffix,
			Policy: repositories.RateLimitPolicy{
				Rate:   backgroundShare(bucket.Limit),
				Period: period,
				Burst:  backgroundShare(burst),
			},
			Cost: cost,
		})
	}

	decisions, err := l.store.Check(ctx, requests)
	if err != nil {
		return fmt.Errorf("check outbound rate limit %q: %w", bucket.Key, err)
	}

	allowed := true
	var retryAfter time.Duration
	for idx := range decisions {
		if decisions[idx].Allowed {
			continue
		}
		allowed = false
		retryAfter = max(retryAfter, decisions[idx].RetryAfter)
	}
	if allowed {
		return nil
	}
	return &restx.RateLimitedError{RetryAfter: retryAfter, Key: bucket.Key}
}

func backgroundShare(value int) int {
	return max(1, value*backgroundSharePercent/percentDenominator)
}

type replicaScaledStore struct {
	inner    repositories.RateLimitStore
	replicas int
}

func newReplicaScaledStore(
	inner repositories.RateLimitStore,
	replicas int,
) repositories.RateLimitStore {
	if replicas <= 1 {
		return inner
	}
	return &replicaScaledStore{inner: inner, replicas: replicas}
}

func (s *replicaScaledStore) Check(
	ctx context.Context,
	requests []repositories.RateLimitRequest,
) ([]repositories.RateLimitDecision, error) {
	scaled := make([]repositories.RateLimitRequest, len(requests))
	for idx := range requests {
		scaled[idx] = requests[idx]
		scaled[idx].Policy.Rate = max(1, requests[idx].Policy.Rate/s.replicas)
	}
	return s.inner.Check(ctx, scaled)
}
