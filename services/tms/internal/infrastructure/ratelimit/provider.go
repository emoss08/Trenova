package ratelimit

import (
	"fmt"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type StoreParams struct {
	fx.In

	Config  *config.Config
	Client  *redis.Client     `optional:"true"`
	Metrics *metrics.Registry `optional:"true"`
	Logger  *zap.Logger
}

func NewStore(p StoreParams) (repositories.RateLimitStore, error) {
	cfg := p.Config.Security.RateLimit
	memory := NewMemoryStore(MemoryStoreOptions{CleanupInterval: cfg.GetCleanupInterval()})

	switch cfg.GetStore() {
	case config.RateLimitStoreMemory:
		p.Logger.Named("ratelimit").Warn(
			"rate limiting is using the in-process store; limits apply per replica",
		)
		return memory, nil
	case config.RateLimitStoreRedis:
		if p.Client == nil {
			return nil, ErrMissingClient
		}
		primary, err := NewRedisStore(p.Client, cfg.GetKeyPrefix())
		if err != nil {
			return nil, err
		}

		var onFailure func(mode string)
		if p.Metrics != nil && p.Metrics.RateLimit != nil {
			onFailure = p.Metrics.RateLimit.RecordStoreFailure
		}

		return NewResilientStore(ResilientStoreOptions{
			Primary:     primary,
			Fallback:    memory,
			FailureMode: cfg.GetFailureMode(),
			Timeout:     cfg.GetStoreTimeout(),
			Logger:      p.Logger,
			OnFailure:   onFailure,
		}), nil
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnknownStore, cfg.GetStore())
	}
}
