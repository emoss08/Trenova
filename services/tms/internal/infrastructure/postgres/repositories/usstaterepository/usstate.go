package usstaterepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
	Cache  repositories.UsStateCacheRepository
}

type repository struct {
	db    *postgres.Connection
	l     *zap.Logger
	cache repositories.UsStateCacheRepository
}

func NewRepository(p Params) repositories.UsStateRepository {
	return &repository{
		db:    p.DB,
		l:     p.Logger.Named("postgres.usstate-repository"),
		cache: p.Cache,
	}
}

func (r *repository) List(
	ctx context.Context,
) (*pagination.ListResult[*usstate.UsState], error) {
	log := r.l.With(zap.String("operation", "List"))

	dba, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	states, err := r.cache.Get(ctx)
	if err == nil && states.Total > 0 {
		log.Debug("got states from cache", zap.Int("stateCount", states.Total))
		return states, nil
	}

	dbStates := make([]*usstate.UsState, 0, 50)

	count, err := dba.NewSelect().Model(&dbStates).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to list us states", zap.Error(err))
		return nil, err
	}

	if err = r.cache.Set(ctx, dbStates); err != nil {
		log.Error("failed to set states in cache", zap.Error(err))
	}

	return &pagination.ListResult[*usstate.UsState]{
		Items: dbStates,
		Total: count,
	}, nil
}

func (r *repository) GetByAbbreviation(
	ctx context.Context,
	abbreviation string,
) (*usstate.UsState, error) {
	log := r.l.With(
		zap.String("operation", "GetByAbbreviation"),
		zap.String("abbreviation", abbreviation),
	)

	state, err := r.cache.GetByAbbreviation(ctx, abbreviation)
	if err == nil {
		log.Debug("got state from cache")
		return state, nil
	}

	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, err
	}

	state = new(usstate.UsState)

	if err = dba.NewSelect().
		Model(state).
		Where("abbreviation = ?", abbreviation).
		Scan(ctx); err != nil {
		log.Error("failed to get us state by abbreviation", zap.Error(err))
		return nil, err
	}

	// * Don't need to explicitly cache this single state as it's already cached
	// * as part of the full state list when List() is called

	return state, nil
}
