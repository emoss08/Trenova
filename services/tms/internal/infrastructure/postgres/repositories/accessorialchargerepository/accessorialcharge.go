package accessorialchargerepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
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

func NewRepository(p Params) repositories.AccessorialChargeRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.accessorialcharge-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListAccessorialChargeRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		"acc",
		req.Filter,
		(*accessorialcharge.AccessorialCharge)(nil),
	)

	return q.Limit(req.Filter.Limit).Offset(req.Filter.Offset)
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListAccessorialChargeRequest,
) (*pagination.ListResult[*accessorialcharge.AccessorialCharge], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.Any("req", req),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entities := make([]*accessorialcharge.AccessorialCharge, 0, req.Filter.Limit)

	total, err := db.NewSelect().Model(&entities).Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
		return r.filterQuery(sq, req)
	}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan accessorial charges", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*accessorialcharge.AccessorialCharge]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetAccessorialChargeByIDRequest,
) (*accessorialcharge.AccessorialCharge, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.Any("req", req),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(accessorialcharge.AccessorialCharge)
	err = db.NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("acc.id = ?", req.ID).
				Where("acc.organization_id = ?", req.OrgID).
				Where("acc.business_unit_id = ?", req.BuID)
		}).Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "AccessorialCharge")
	}

	return entity, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *accessorialcharge.AccessorialCharge,
) (*accessorialcharge.AccessorialCharge, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("accID", entity.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	if _, err = db.NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to insert accessorial charge", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *accessorialcharge.AccessorialCharge,
) (*accessorialcharge.AccessorialCharge, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("accID", entity.ID.String()),
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
		OmitZero().
		Where("acc.version = ?", ov).
		Returning("*").
		Exec(ctx)
	if rErr != nil {
		log.Error("failed to update accessorial charge", zap.Error(rErr))
		return nil, rErr
	}

	roErr := dberror.CheckRowsAffected(results, "AccessorialCharge", entity.ID.String())
	if roErr != nil {
		return nil, roErr
	}

	return entity, nil
}
