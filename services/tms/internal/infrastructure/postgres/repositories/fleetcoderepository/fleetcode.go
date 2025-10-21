package fleetcoderepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/fleetcode"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/utils/querybuilder"
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

func NewRepository(p Params) repositories.FleetCodeRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.fleetcoderepository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListFleetCodeRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		"fc",
		req.Filter,
		(*fleetcode.FleetCode)(nil),
	)

	if req.IncludeManagerDetails {
		q = q.Relation("Manager")
	}

	return q.Limit(req.Filter.Limit).Offset(req.Filter.Offset)
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListFleetCodeRequest,
) (*pagination.ListResult[*fleetcode.FleetCode], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.String("orgId", req.Filter.TenantOpts.OrgID.String()),
		zap.String("buId", req.Filter.TenantOpts.BuID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entities := make([]*fleetcode.FleetCode, 0, req.Filter.Limit)

	total, err := db.NewSelect().Model(&entities).Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
		return r.filterQuery(sq, req)
	}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan fleet codes", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*fleetcode.FleetCode]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetFleetCodeByIDRequest,
) (*fleetcode.FleetCode, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("entityID", req.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(fleetcode.FleetCode)
	query := db.NewSelect().Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("fc.id = ?", req.ID).
				Where("fc.organization_id = ?", req.OrgID).
				Where("fc.business_unit_id = ?", req.BuID)
		})

	if req.IncludeManagerDetails {
		query.Relation("Manager")
	}

	if err = query.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "Fleet code")
	}

	return entity, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *fleetcode.FleetCode,
) (*fleetcode.FleetCode, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("entityID", entity.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	if _, err = db.NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to insert fleet code", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *fleetcode.FleetCode,
) (*fleetcode.FleetCode, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("entityID", entity.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	ov := entity.Version
	entity.Version++

	results, rErr := db.NewUpdate().
		Model(entity).
		WherePK().
		Where("fc.version = ?", ov).
		Returning("*").
		Exec(ctx)
	if rErr != nil {
		log.Error("failed to update fleet code", zap.Error(rErr))
		return nil, rErr
	}

	roErr := dberror.CheckRowsAffected(results, "Fleet code", entity.ID.String())
	if roErr != nil {
		return nil, roErr
	}

	return entity, nil
}
