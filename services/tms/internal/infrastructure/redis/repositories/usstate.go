package repositories

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/redis"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultStateTTL = 24 * time.Hour
	stateKeyPrefix  = "usstates"
)

type UsStateRepositoryParams struct {
	fx.In

	Cache  *redis.Connection
	Logger *zap.Logger
}

type usStateRepository struct {
	cache *redis.Connection
	l     *zap.Logger
}

func NewUsStateRepository(p UsStateRepositoryParams) repositories.UsStateCacheRepository {
	return &usStateRepository{
		cache: p.Cache,
		l:     p.Logger.Named("redis.usstate-repository"),
	}
}

func (usr *usStateRepository) Get(
	ctx context.Context,
) (*pagination.ListResult[*usstate.UsState], error) {
	log := usr.l.With(zap.String("operation", "Get"))

	states := make([]*usstate.UsState, 0)

	if err := usr.cache.GetJSON(ctx, stateKeyPrefix, &states); err != nil {
		log.Error("failed to get states from cache", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*usstate.UsState]{
		Items: states,
		Total: len(states),
	}, nil
}

func (usr *usStateRepository) Set(ctx context.Context, states []*usstate.UsState) error {
	log := usr.l.With(zap.String("operation", "Set"))

	key := stateKeyPrefix
	if err := usr.cache.SetJSON(ctx, key, states, defaultStateTTL); err != nil {
		log.Error("failed to set states in cache", zap.Error(err))
		return err
	}

	log.Debug("stored states in cache", zap.Int("stateCount", len(states)))

	return nil
}

func (usr *usStateRepository) Invalidate(ctx context.Context) error {
	log := usr.l.With(zap.String("operation", "Invalidate"))

	if err := usr.cache.Delete(ctx, stateKeyPrefix); err != nil {
		log.Error("failed to invalidate states in cache", zap.Error(err))
		return err
	}

	log.Debug("invalidated states in cache")
	return nil
}

func (usr *usStateRepository) GetByAbbreviation(
	ctx context.Context,
	abbreviation string,
) (*usstate.UsState, error) {
	log := usr.l.With(
		zap.String("operation", "GetByAbbreviation"),
		zap.String("abbreviation", abbreviation),
	)

	states := make([]*usstate.UsState, 0)

	if err := usr.cache.GetJSON(ctx, stateKeyPrefix, &states); err != nil {
		log.Error("failed to get states from cache", zap.Error(err))
		return nil, err
	}

	for _, state := range states {
		if state.Abbreviation == abbreviation {
			log.Debug("found state in cache")
			return state, nil
		}
	}

	log.Debug("state not found in cache")
	return nil, errortypes.NewNotFoundError("state not found in cache")
}
