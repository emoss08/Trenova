package usstaterepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/uptrace/bun"
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

func New(p Params) repositories.UsStateRepository {
	return &repository{
		db:    p.DB,
		l:     p.Logger.Named("postgres.us-state-repository"),
		cache: p.Cache,
	}
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetUsStateByIDRequest,
) (*usstate.UsState, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("id", req.StateID.String()),
	)

	entity := new(usstate.UsState)
	err := r.db.DB().
		NewSelect().
		Model(entity).
		Where("ust.id = ?", req.StateID).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get us state", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "UsState")
	}

	return entity, nil
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *pagination.SelectQueryRequest,
) (*pagination.ListResult[*usstate.UsState], error) {
	entities := make([]*usstate.UsState, 0, req.Pagination.Limit)

	q := r.db.DB().
		NewSelect().
		Model(&entities).
		Column("id", "name", "abbreviation").
		Limit(req.Pagination.Limit).
		Offset(req.Pagination.Offset)

	if req.Query != "" {
		q.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.WhereOr("LOWER(name) LIKE LOWER(?)", dbhelper.WrapWildcard(req.Query)).
				WhereOr("LOWER(abbreviation) LIKE LOWER(?)", dbhelper.WrapWildcard(req.Query))
		})
	}

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}

	return &pagination.ListResult[*usstate.UsState]{
		Items: entities,
		Total: total,
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
		return state, nil
	}

	allStates := make([]*usstate.UsState, 0)
	if err = r.db.DB().NewSelect().Model(&allStates).Scan(ctx); err != nil {
		log.Error("failed to load us states", zap.Error(err))
		return nil, err
	}

	if cacheErr := r.cache.Set(ctx, allStates); cacheErr != nil {
		log.Warn("failed to populate us states cache", zap.Error(cacheErr))
	}

	for _, s := range allStates {
		if s.Abbreviation == abbreviation {
			return s, nil
		}
	}

	return nil, errortypes.NewNotFoundError("us state not found")
}
