package usstaterepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.UsStateRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.us-state-repository"),
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
	entities := make([]*usstate.UsState, 0, req.Pagination.SafeLimit())

	q := r.db.DB().
		NewSelect().
		Model(&entities).
		Column("id", "created_at", "name", "abbreviation", "country_iso3").
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
	entity := new(usstate.UsState)
	err := r.db.DB().
		NewSelect().
		Model(entity).
		Where(
			buncolgen.UsStateColumns.Abbreviation.Eq(),
			strings.ToUpper(strings.TrimSpace(abbreviation)),
		).
		Scan(ctx)
	if err != nil {
		r.l.Debug("failed to get us state by abbreviation",
			zap.String("abbreviation", abbreviation),
			zap.Error(err),
		)
		return nil, dberror.HandleNotFoundError(err, "UsState")
	}

	return entity, nil
}

func (r *repository) GetByIDs(ctx context.Context, ids []pulid.ID) ([]*usstate.UsState, error) {
	entities := make([]*usstate.UsState, 0, len(ids))
	if len(ids) == 0 {
		return entities, nil
	}

	err := r.db.DB().
		NewSelect().
		Model(&entities).
		Where(buncolgen.UsStateColumns.ID.In(), bun.List(ids)).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to get us states by ids", zap.Error(err))
		return nil, fmt.Errorf("get us states by ids: %w", err)
	}

	return entities, nil
}
