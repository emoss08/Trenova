package repositories

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/redishelpers"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultStateTTL = 24 * time.Hour
	stateKeyPrefix  = "usstates"
)

type UsStateCacheRepositoryParams struct {
	fx.In

	Client *redis.Client
	Logger *zap.Logger
}

type usStateCacheRepository struct {
	client *redis.Client
	l      *zap.Logger
}

func NewUsStateCacheRepository(p UsStateCacheRepositoryParams) repositories.UsStateCacheRepository {
	log := p.Logger.Named("redis.usstate-cache-repository")
	return &usStateCacheRepository{
		client: p.Client,
		l:      log,
	}
}

func (r *usStateCacheRepository) Set(ctx context.Context, states []*usstate.UsState) error {
	log := r.l.With(zap.String("operation", "Set"))

	if err := redishelpers.SetJSON(ctx, r.client, stateKeyPrefix, states, defaultStateTTL); err != nil {
		log.Error("failed to set states in cache", zap.Error(err))
		return err
	}

	log.Debug("stored states in cache", zap.Int("stateCount", len(states)))
	return nil
}

func (r *usStateCacheRepository) GetByAbbreviation(
	ctx context.Context,
	abbreviation string,
) (*usstate.UsState, error) {
	log := r.l.With(
		zap.String("operation", "GetByAbbreviation"),
		zap.String("abbreviation", abbreviation),
	)

	states := make([]*usstate.UsState, 0)
	if err := redishelpers.GetJSON(ctx, r.client, stateKeyPrefix, &states); err != nil {
		if redishelpers.IsRedisNil(err) {
			return nil, errortypes.NewNotFoundError("states not cached")
		}

		log.Error("failed to get states from cache", zap.Error(err))
		return nil, err
	}

	for _, state := range states {
		if state.Abbreviation == abbreviation {
			return state, nil
		}
	}

	return nil, errortypes.NewNotFoundError("state not found in cache")
}

func (r *usStateCacheRepository) Invalidate(ctx context.Context) error {
	log := r.l.With(zap.String("operation", "Invalidate"))

	if err := r.client.Del(ctx, stateKeyPrefix).Err(); err != nil {
		log.Error("failed to invalidate states in cache", zap.Error(err))
		return err
	}

	log.Debug("invalidated states in cache")
	return nil
}
