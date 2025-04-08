package repositories

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/cache/redis"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

const (
	defaultStateTTL = 24 * time.Hour
	stateKeyPrefix  = "usstates"
)

type StateRepositoryParams struct {
	fx.In

	Cache  *redis.Client
	Logger *logger.Logger
}

type stateRepository struct {
	cache    *redis.Client
	l        *zerolog.Logger
	cacheTTL time.Duration
}

func NewStateRepository(p StateRepositoryParams) repositories.UsStateCacheRepository {
	log := p.Logger.With().
		Str("repository", "us_state").
		Str("component", "redis").
		Logger()

	return &stateRepository{
		cache:    p.Cache,
		l:        &log,
		cacheTTL: defaultStateTTL,
	}
}

// Get retrieves all states from the cache
func (sr *stateRepository) Get(ctx context.Context) (*ports.ListResult[*usstate.UsState], error) {
	log := sr.l.With().Str("operation", "Get").Logger()

	states := make([]*usstate.UsState, 0)

	if err := sr.cache.GetJSON(ctx, stateKeyPrefix, &states); err != nil {
		if eris.Is(err, redis.ErrNil) {
			log.Debug().Msg("no states found in cache")
			return nil, eris.New("no states found in cache")
		}

		return nil, eris.Wrap(err, "failed to get states from cache")
	}

	log.Debug().
		Int("stateCount", len(states)).
		Msg("retrieved states from cache")

	return &ports.ListResult[*usstate.UsState]{
		Items: states,
		Total: len(states),
	}, nil
}

func (sr *stateRepository) Set(ctx context.Context, states []*usstate.UsState) error {
	log := sr.l.With().Str("operation", "Set").Logger()

	key := stateKeyPrefix
	if err := sr.cache.SetJSON(ctx, key, states, sr.cacheTTL); err != nil {
		return eris.Wrap(err, "failed to set states in cache")
	}

	log.Debug().
		Int("stateCount", len(states)).
		Msg("stored states in cache")

	return nil
}

func (sr *stateRepository) Invalidate(ctx context.Context) error {
	log := sr.l.With().Str("operation", "Invalidate").Logger()

	if err := sr.cache.Del(ctx, stateKeyPrefix); err != nil {
		return eris.Wrap(err, "failed to invalidate states in cache")
	}

	log.Debug().Msg("invalidated states in cache")
	return nil
}
