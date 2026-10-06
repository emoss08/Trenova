package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
)

const (
	providerBreakerPrefix = "ai:breaker"
	providerBreakerValue  = "open"
)

type ProviderBreakerParams struct {
	fx.In

	Client *redis.Client
}

type providerBreakerRepository struct {
	client *redis.Client
}

func NewProviderBreakerRepository(p ProviderBreakerParams) repositories.ProviderBreakerRepository {
	return &providerBreakerRepository{client: p.Client}
}

func providerBreakerKey(providerID pulid.ID) string {
	return providerBreakerPrefix + ":" + providerID.String()
}

func (r *providerBreakerRepository) Rest(
	ctx context.Context,
	providerID pulid.ID,
	cooldown time.Duration,
) error {
	if providerID.IsNil() || cooldown <= 0 {
		return nil
	}

	err := r.client.Set(ctx, providerBreakerKey(providerID), providerBreakerValue, cooldown).Err()
	if err != nil {
		return fmt.Errorf("rest provider: %w", err)
	}

	return nil
}

func (r *providerBreakerRepository) Resting(
	ctx context.Context,
	providerIDs []pulid.ID,
) (map[pulid.ID]time.Duration, error) {
	if len(providerIDs) == 0 {
		return map[pulid.ID]time.Duration{}, nil
	}

	cmds := make([]*redis.DurationCmd, len(providerIDs))
	_, err := r.client.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		for idx, id := range providerIDs {
			cmds[idx] = pipe.PTTL(ctx, providerBreakerKey(id))
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("read resting providers: %w", err)
	}

	resting := make(map[pulid.ID]time.Duration, len(providerIDs))
	for idx, cmd := range cmds {
		if remaining := cmd.Val(); remaining > 0 {
			resting[providerIDs[idx]] = remaining
		}
	}

	return resting, nil
}
